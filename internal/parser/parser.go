package parser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

func ParseFile(content string) (map[string]interface{}, string) {
	fm := make(map[string]interface{})
	body := content
	content = strings.ReplaceAll(content, "\r\n", "\n")
	if strings.HasPrefix(content, "---") {
		parts := strings.SplitN(content, "---", 3)
		if len(parts) >= 3 {
			fm = ParseYAML(parts[1])
			body = strings.TrimSpace(parts[2])
		}
	}
	return fm, body
}

type stackItem struct {
	indent int
	key string
	m map[string]interface{}
	arr []interface{}
	isMap bool
	parent map[string]interface{}
}

func ParseYAML(yamlStr string) map[string]interface{} {
	result := make(map[string]interface{})
	lines := strings.Split(yamlStr, "\n")
	stack := []stackItem{{indent: -1, m: result, isMap: true, key: ""}}
	for _, raw := range lines {
		if strings.TrimSpace(raw) == "" { continue }
		indent := 0
		for _, ch := range raw { if ch == ' ' { indent++ } else { break } }
		trimmed := strings.TrimSpace(raw)
		for len(stack) > 1 && indent <= stack[len(stack)-1].indent {
			stack = stack[:len(stack)-1]
		}
		cur := &stack[len(stack)-1]
		if strings.HasPrefix(trimmed, "- ") {
			val := strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
			if cur.isMap && len(cur.m) == 0 && cur.parent!= nil && cur.key!= "" {
				cur.parent[cur.key] = []interface{}{}
				newArrItem := stackItem{indent: cur.indent, key: cur.key, arr: cur.parent[cur.key].([]interface{}), isMap: false, parent: cur.parent}
				stack[len(stack)-1] = newArrItem
				cur = &stack[len(stack)-1]
			}
			if cur.isMap {
				listKey := cur.key
				if cur.parent!= nil && cur.key!= "" {
					if _, ok := cur.parent[cur.key].([]interface{});!ok {
						if _, ok := cur.parent[cur.key].(map[string]interface{}); ok {
							cur.parent[cur.key] = []interface{}{}
						}
					}
					if arr, ok := cur.parent[cur.key].([]interface{}); ok {
						cur.isMap = false
						cur.arr = arr
					}
				} else {
					found := false
					for k, v := range cur.m {
						if _, ok := v.([]interface{}); ok {
							listKey = k
							found = true
							break
						}
					}
					if!found {
						if cur.key!= "" { listKey = cur.key } else { listKey = "items" }
						cur.m[listKey] = []interface{}{}
					}
					if arr, ok := cur.m[listKey].([]interface{}); ok {
						cur.isMap = false
						cur.arr = arr
						cur.key = listKey
					}
				}
			}
			if!cur.isMap {
				if strings.Contains(val, ":") {
					item := make(map[string]interface{})
					kv := strings.SplitN(val, ":", 2)
					k := strings.TrimSpace(kv[0])
					v := strings.TrimSpace(kv[1])
					if v!= "" { item[k] = parseValue(v) }
					if cur.parent!= nil {
						arr := cur.parent[cur.key].([]interface{})
						arr = append(arr, item)
						cur.parent[cur.key] = arr
						cur.arr = arr
					} else {
						cur.arr = append(cur.arr, item)
						cur.m[cur.key] = cur.arr
					}
					stack = append(stack, stackItem{indent: indent, m: item, isMap: true, key: "", parent: nil})
				} else {
					if cur.parent!= nil {
						arr := cur.parent[cur.key].([]interface{})
						arr = append(arr, parseValue(val))
						cur.parent[cur.key] = arr
						cur.arr = arr
					} else {
						cur.arr = append(cur.arr, parseValue(val))
						cur.m[cur.key] = cur.arr
					}
				}
			}
			continue
		}
		if strings.Contains(trimmed, ":") {
			parts := strings.SplitN(trimmed, ":", 2)
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			if cur.isMap {
				if value == "" {
					newObj := make(map[string]interface{})
					cur.m[key] = newObj
					stack = append(stack, stackItem{indent: indent, m: newObj, isMap: true, key: key, parent: cur.m})
				} else {
					if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
						cur.m[key] = parseArray(value)
					} else if strings.HasPrefix(value, "{") && strings.HasSuffix(value, "}") {
						cur.m[key] = parseInlineMap(value)
					} else {
						cur.m[key] = parseValue(value)
					}
					cur.key = key
				}
			} else {
				if len(cur.arr) > 0 {
					if lastMap, ok := cur.arr[len(cur.arr)-1].(map[string]interface{}); ok {
						if value == "" {
							newObj := make(map[string]interface{})
							lastMap[key] = newObj
							stack = append(stack, stackItem{indent: indent, m: newObj, isMap: true, key: key, parent: lastMap})
						} else {
							lastMap[key] = parseValue(value)
						}
						cur.arr[len(cur.arr)-1] = lastMap
						if cur.parent!= nil { cur.parent[cur.key] = cur.arr }
					}
				}
			}
			continue
		}
	}
	flattenNested(result)
	return result
}

func flattenNested(m map[string]interface{}) {
	for key, val := range m {
		if mm, ok := val.(map[string]interface{}); ok {
			if len(mm) == 1 {
				if inner, ok := mm["list"]; ok { if list, ok := inner.([]interface{}); ok { m[key] = list; continue } }
				if inner, ok := mm["items"]; ok { if list, ok := inner.([]interface{}); ok { m[key] = list; continue } }
			}
			flattenNested(mm)
		}
		if arr, ok := val.([]interface{}); ok {
			for _, item := range arr { if sub, ok := item.(map[string]interface{}); ok { flattenNested(sub) } }
		}
	}
}

func parseValue(s string) interface{} {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "\"'")
	if s == "true" { return true }
	if s == "false" { return false }
	if s == "null" || s == "~" { return nil }
	if strings.Contains(s, ".") { if f, err := strconv.ParseFloat(s, 64); err == nil { return f } } else if strings.ContainsAny(s, "0123456789") { if i, err := strconv.Atoi(s); err == nil { return i } }
	return s
}

func parseArray(s string) []interface{} {
	s = strings.Trim(s, "[]")
	if s == "" { return []interface{}{} }
	var result []interface{}
	for _, item := range strings.Split(s, ",") { result = append(result, parseValue(strings.TrimSpace(item))) }
	return result
}

func parseInlineMap(s string) map[string]interface{} {
	s = strings.Trim(s, "{}")
	if s == "" { return map[string]interface{}{} }
	result := make(map[string]interface{})
	for _, pair := range strings.Split(s, ",") {
		kv := strings.SplitN(pair, ":", 2)
		if len(kv) == 2 { key := strings.TrimSpace(kv[0]); value := parseValue(strings.TrimSpace(kv[1])); result[key] = value }
	}
	return result
}

func MdToHTML(md string) string {
	lines := strings.Split(md, "\n")

	footnotes := map[string]string{}
	reFootDef := regexp.MustCompile(`^\[\^(.+?)\]:\s*(.*)`)
	var cleanedLines []string
	for _, l := range lines {
		if m := reFootDef.FindStringSubmatch(l); m!= nil {
			footnotes[m[1]] = m[2]
			continue
		}
		cleanedLines = append(cleanedLines, l)
	}
	lines = cleanedLines

	var out []string
	inList := false
	inCodeBlock := false
	inBlockquote := false
	inTable := false
	var tableHeaders []string
	var tableRows [][]string
	codeLang := ""

	flushTable := func() {
		if!inTable { return }
		out = append(out, "<table>")
		out = append(out, "<thead><tr>")
		for _, h := range tableHeaders { out = append(out, "<th>"+processInline(h)+"</th>") }
		out = append(out, "</tr></thead>")
		out = append(out, "<tbody>")
		for _, row := range tableRows {
			out = append(out, "<tr>")
			for _, c := range row { out = append(out, "<td>"+processInline(c)+"</td>") }
			out = append(out, "</tr>")
		}
		out = append(out, "</tbody></table>")
		inTable = false
		tableHeaders = nil
		tableRows = nil
	}

	isTableSeparator := func(s string) bool {
		s = strings.TrimSpace(s)
		if!strings.Contains(s, "|") &&!strings.Contains(s, "-") { return false }
		re := regexp.MustCompile(`^[\|\s\-:]+$`)
		return re.MatchString(s) && strings.Contains(s, "-")
	}

	for i := 0; i < len(lines); i++ {
		l := lines[i]
		t := strings.TrimSpace(l)

		if strings.HasPrefix(l, "```") {
			flushTable()
			if inList { out = append(out, "</ul>"); inList = false }
			if inBlockquote { out = append(out, "</blockquote>"); inBlockquote = false }
			if!inCodeBlock {
				inCodeBlock = true
				codeLang = strings.TrimPrefix(l, "```")
				codeLang = strings.TrimSpace(codeLang)
				if codeLang!= "" {
					out = append(out, "<pre><code class=\"language-"+codeLang+"\">")
				} else {
					out = append(out, "<pre><code>")
				}
			} else {
				inCodeBlock = false
				out = append(out, "</code></pre>")
			}
			continue
		}
		if inCodeBlock { out = append(out, l); continue }

		if strings.Contains(t, "|") && i+1 < len(lines) && isTableSeparator(lines[i+1]) {
			flushTable()
			if inList { out = append(out, "</ul>"); inList = false }
			if inBlockquote { out = append(out, "</blockquote>"); inBlockquote = false }
			headers := strings.Split(t, "|")
			for j, h := range headers { headers[j] = strings.TrimSpace(h) }
			
			var cleanHeaders []string
			for _, h := range headers { if h!= "" || len(headers) <= 2 { cleanHeaders = append(cleanHeaders, h) } }
			if len(cleanHeaders) == 0 { cleanHeaders = headers }
			tableHeaders = cleanHeaders
			inTable = true
			i++ 
			continue
		}
		if inTable {
			if t == "" ||!strings.Contains(t, "|") {
				flushTable()
			} else {
				cols := strings.Split(t, "|")
				var cleanCols []string
				for _, c := range cols {
					c = strings.TrimSpace(c)
					if c!= "" || len(cols) <= 2 {
						cleanCols = append(cleanCols, c)
					}
				}
				
				if strings.HasPrefix(t, "|") && len(cleanCols) > 0 && cleanCols[0] == "" {
					cleanCols = cleanCols[1:]
				}
				if strings.HasSuffix(strings.TrimSpace(t), "|") && len(cleanCols) > 0 && cleanCols[len(cleanCols)-1] == "" {
					cleanCols = cleanCols[:len(cleanCols)-1]
				}
				tableRows = append(tableRows, cleanCols)
				continue
			}
		}

		if strings.HasPrefix(t, "> ") || t == ">" {
			if!inBlockquote {
				if inList { out = append(out, "</ul>"); inList = false }
				out = append(out, "<blockquote>"); inBlockquote = true
			}
			content := strings.TrimPrefix(t, "> ")
			content = strings.TrimPrefix(content, ">")
			if content == "" {
				out = append(out, "<br>")
			} else {
				out = append(out, "<p>"+processInline(strings.TrimSpace(content))+"</p>")
			}
			continue
		} else {
			if inBlockquote { out = append(out, "</blockquote>"); inBlockquote = false }
		}

		if t == "" {
			if inList { out = append(out, "</ul>"); inList = false }
			continue
		}

		
		if strings.HasPrefix(t, "# ") {
			if inList { out = append(out, "</ul>"); inList = false }
			title := strings.TrimPrefix(t, "# ")
			id := makeID(title)
			out = append(out, fmt.Sprintf("<h1 id=\"%s\">%s</h1>", id, processInline(title)))
			continue
		}
		if strings.HasPrefix(t, "## ") {
			if inList { out = append(out, "</ul>"); inList = false }
			title := strings.TrimPrefix(t, "## ")
			id := makeID(title)
			out = append(out, fmt.Sprintf("<h2 id=\"%s\">%s</h2>", id, processInline(title)))
			continue
		}
		if strings.HasPrefix(t, "### ") {
			if inList { out = append(out, "</ul>"); inList = false }
			title := strings.TrimPrefix(t, "### ")
			id := makeID(title)
			out = append(out, fmt.Sprintf("<h3 id=\"%s\">%s</h3>", id, processInline(title)))
			continue
		}
		if strings.HasPrefix(t, "#### ") {
			if inList { out = append(out, "</ul>"); inList = false }
			title := strings.TrimPrefix(t, "#### ")
			id := makeID(title)
			out = append(out, fmt.Sprintf("<h4 id=\"%s\">%s</h4>", id, processInline(title)))
			continue
		}

		if strings.HasPrefix(t, "- ") || strings.HasPrefix(t, "* ") {
			if!inList { out = append(out, "<ul>"); inList = true }
			txt := strings.TrimPrefix(t, "- ")
			txt = strings.TrimPrefix(txt, "* ")
			txt = processInline(txt)
			out = append(out, "<li>"+txt+"</li>")
			continue
		}
		if inList { out = append(out, "</ul>"); inList = false }
		out = append(out, "<p>"+processInline(t)+"</p>")
	}
	if inList { out = append(out, "</ul>") }
	if inBlockquote { out = append(out, "</blockquote>") }
	flushTable()

	if len(footnotes) > 0 {
		out = append(out, "<hr><section class=\"footnotes\"><ol>")
		for k, v := range footnotes {
			out = append(out, fmt.Sprintf("<li id=\"fn-%s\">%s <a href=\"#fnref-%s\">↩</a></li>", k, processInline(v), k))
		}
		out = append(out, "</ol></section>")
	}

	return strings.Join(out, "\n")
}

func makeID(s string) string {
	s = strings.ToLower(s)
	re := regexp.MustCompile(`[^a-z0-9]+`)
	s = re.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}

func processInline(text string) string {
	
	reFoot := regexp.MustCompile(`\[\^(.+?)\]`)
	text = reFoot.ReplaceAllString(text, `<sup id="fnref-$1"><a href="#fn-$1">[$1]</a></sup>`)

	re := regexp.MustCompile(`\*\*(.*?)\*\*`)
	text = re.ReplaceAllString(text, "<b>$1</b>")
	re = regexp.MustCompile(`\*(.*?)\*`)
	text = re.ReplaceAllString(text, "<i>$1</i>")
	re = regexp.MustCompile("`(.*?)`")
	text = re.ReplaceAllString(text, "<code>$1</code>")
	re = regexp.MustCompile(`\[(.*?)\]\((.*?)\)`)
	text = re.ReplaceAllString(text, "<a href=\"$2\">$1</a>")
	return text
}

func GenerateTOC(mdContent string) string {
	lines := strings.Split(mdContent, "\n")
	var headings []struct{ Level int; Title, ID string }

	reH := regexp.MustCompile(`^(#{1,4})\s+(.*)`)
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if m := reH.FindStringSubmatch(l); m!= nil {
			level := len(m[1])
			title := strings.TrimSpace(m[2])
			id := makeID(title)
			headings = append(headings, struct{ Level int; Title, ID string }{level, title, id})
		}
	}
	if len(headings) == 0 { return "" }

	var sb strings.Builder
	sb.WriteString(`<nav class="toc"><ul>`)
	prev := 1
	for _, h := range headings {
		if h.Level > prev {
			for i := prev; i < h.Level; i++ { sb.WriteString("<ul>") }
		} else if h.Level < prev {
			for i := h.Level; i < prev; i++ { sb.WriteString("</ul>") }
		}
		sb.WriteString(fmt.Sprintf(`<li class="toc-l%d"><a href="#%s">%s</a></li>`, h.Level, h.ID, h.Title))
		prev = h.Level
	}
	for i := 1; i < prev; i++ { sb.WriteString("</ul>") }
	sb.WriteString("</ul></nav>")
	return sb.String()
}
