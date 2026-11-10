package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mesinkasir/gax/internal/collection"
	"github.com/mesinkasir/gax/internal/data"
	"github.com/mesinkasir/gax/internal/render"
)

const (
	Reset   = "\033[0m"
	Bold    = "\033[1m"
	Dim     = "\033[2m"
	Cyan    = "\033[36m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Magenta = "\033[35m"
	Blue    = "\033[34m"
	Red     = "\033[31m"
)

func banner() {
	fmt.Println("")
	fmt.Println(Cyan + Bold + "  ██████╗  █████╗ ██╗  ██╗" + Reset)
	fmt.Println(Cyan + Bold + " ██╔════╝ ██╔══██╗╚██╗██╔╝" + Reset + "  " + Dim + "Static Site Generator" + Reset)
	fmt.Println(Cyan + Bold + " ██║  ███╗███████║ ╚███╔╝ " + Reset + "  " + Dim + "Slim, Fast, 11ty-like" + Reset)
	fmt.Println(Cyan + Bold + " ██║   ██║██╔══██║ ██╔██╗ " + Reset)
	fmt.Println(Cyan + Bold + " ╚██████╔╝██║  ██║██╔╝ ██╗" + Reset + "  " + Green + "v1.0.0" + Reset + Dim + " | gax.axcora.com" + Reset)
	fmt.Println(Cyan + Bold + "  ╚═════╝ ╚═╝  ╚═╝╚═╝  ╚═╝" + Reset)
	fmt.Println("")
}

func logInfo(msg string) { fmt.Printf("  %s%s%s %s\n", Blue, "●", Reset, msg) }
func logOk(msg string)   { fmt.Printf("  %s%s%s %s\n", Green, "✔", Reset, msg) }
func logWarn(msg string) { fmt.Printf("  %s%s%s %s\n", Yellow, "⚠", Reset, msg) }
func logBuild(file string) {
	fmt.Printf("  %s%s%s %s%s%s\n", Dim, "→", Reset, Dim, file, Reset)
}

func main() {
	banner()
	if len(os.Args) < 2 {
		help()
		return
	}

	cmd := os.Args[1]
	start := time.Now()

	global := data.Load("_data")

	cols, allPosts, tags := collection.Build("content", global)

	switch cmd {
	case "build":
		fmt.Printf("%s%s building %s%s\n", Bold, "GAX", "your site...", Reset)
		fmt.Println("")

		render.BuildSite(cols, allPosts, tags, global, "templates", "content", "site", "public", false)

		elapsed := time.Since(start)
		fmt.Println("")
		logOk(fmt.Sprintf("Built in %s", elapsed.Round(time.Millisecond)))
		fmt.Printf("\n  %s%s%s\n", Bold, "Output:", Reset)
		fmt.Printf("  %s  site/%s (%d pages)\n", Green+"●"+Reset, Dim, len(allPosts)+2)
		fmt.Printf("  %s  sitemap.xml, rss.xml, feed.xml, robots.txt\n", Green+"●"+Reset)
		fmt.Printf("\n  %s%s%s %s\n", Dim, "—", Reset, "gax.axcora.com | Fast, Slim, Go SSG")
		fmt.Println("")

	case "start", "dev", "serve":
		fmt.Printf("%s%s dev server starting...%s\n", Bold, "GAX", Reset)
		fmt.Println("")

		render.BuildSite(cols, allPosts, tags, global, "templates", "content", "site", "public", true)

		elapsed := time.Since(start)
		logOk(fmt.Sprintf("Built in %s", elapsed.Round(time.Millisecond)))
		fmt.Println("")

		fmt.Printf("  %s %s%s%s\n", Green+"●"+Reset, Bold, "Local:", Reset+"   http://localhost:8080")
		fmt.Printf("  %s %s%s%s\n", Blue+"●"+Reset, Bold, "Network:", Reset+" http://localhost:8080")
		fmt.Printf("  %s %s%s%s\n", Magenta+"●"+Reset, Bold, "Watch:", Reset+"  content/, templates/, _data/, public/")
		fmt.Println("")
		fmt.Printf("  %s Live reload enabled - editing will trigger rebuild\n", Dim+"●"+Reset)
		fmt.Println("")

		go watchAndRebuild()
		render.Serve("site", ":8080")

	case "version", "--version", "-v":
		fmt.Printf("  GAX v1.0.0 - gax.axcora.com\n")

	default:
		help()
	}
}

func help() {
	fmt.Printf("  %sUsage:%s\n", Bold, Reset)
	fmt.Printf("    gax build     Build your site to ./site\n")
	fmt.Printf("    gax start     Start dev server with live reload\n")
	fmt.Printf("    gax dev       Alias for start\n")
	fmt.Printf("\n  %sDocs:%s\n", Bold, Reset)
	fmt.Printf("    https://gax.axcora.com\n")
	fmt.Println("")
}

func watchAndRebuild() {
	last := make(map[string]time.Time)
	for {
		time.Sleep(700 * time.Millisecond)
		changed := false
		var changedFile string

		dirs := []string{"content", "templates", "_data", "public"}
		for _, d := range dirs {
			filepath.Walk(d, func(p string, info os.FileInfo, err error) error {
				if err != nil {
					return nil
				}
				if info.IsDir() {
					return nil
				}
				mt := info.ModTime()
				if prev, ok := last[p]; !ok {
					last[p] = mt
				} else if mt.After(prev) {
					last[p] = mt
					changed = true
					changedFile = p
				}
				return nil
			})
		}

		if changed {
			fmt.Println("")
			logInfo(fmt.Sprintf("Change detected: %s%s%s", Yellow, changedFile, Reset))

			global := data.Load("_data")
			cols, allPosts, tags := collection.Build("content", global)

			t0 := time.Now()
			render.BuildSite(cols, allPosts, tags, global, "templates", "content", "site", "public", true)
			logOk(fmt.Sprintf("Rebuilt in %s", time.Since(t0).Round(time.Millisecond)))
			fmt.Printf("  %s Watching...\n", Dim+"●"+Reset)
		}
	}
}
