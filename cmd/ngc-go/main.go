package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func usage() {
	fmt.Println(`ngc-go - minimal scaffold
Usage: ngc-go <command> [args]

Commands:
  compile <path>   Compile project (placeholder)
  watch <path>     Watch and compile (placeholder)
  help             Show help`)
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}
	cmd := os.Args[1]
	switch cmd {
	case "help":
		usage()
	case "compile":
		path := "."
		if len(os.Args) >= 3 {
			path = os.Args[2]
		}
		if err := compile(path); err != nil {
			fmt.Fprintf(os.Stderr, "compile error: %v\n", err)
			os.Exit(1)
		}
	case "watch":
		fmt.Println("watch is not implemented yet")
	default:
		usage()
		os.Exit(1)
	}
}

func compile(root string) error {
	fmt.Printf("Scanning %s for TypeScript components...\n", root)
	compRe := regexp.MustCompile(`@Component\s*\(\s*\{([\s\S]*?)\}\s*\)`)
	templateRe := regexp.MustCompile(`template\s*:\s*` + "`" + `([\s\S]*?)` + "`")
	templateUrlRe := regexp.MustCompile(`templateUrl\s*:\s*['\"]([^'\"]+)['\"]`)
	var found int
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".ts") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		s := string(data)
		if compRe.MatchString(s) {
			found++
			match := compRe.FindStringSubmatch(s)
			compBody := match[1]
			fmt.Printf("Found component in %s\n", path)
			if m := templateRe.FindStringSubmatch(compBody); m != nil {
				fmt.Println(" - inline template (first 80 chars):")
				txt := m[1]
				if len(txt) > 80 {
					txt = txt[:80] + "..."
				}
				fmt.Printf("   `%s`\n", txt)
			} else if m := templateUrlRe.FindStringSubmatch(compBody); m != nil {
				fmt.Printf(" - templateUrl: %s\n", m[1])
			} else {
				fmt.Println(" - no template found")
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	fmt.Printf("Scan complete: %d components found\n", found)
	return nil
}
