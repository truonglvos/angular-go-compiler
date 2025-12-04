package main

import (
	"fmt"
	"os"
)

func usage() {
	fmt.Println(`ngc-go - minimal scaffold
Usage: ngc-go <command> [args]

Commands:
  compile <path> [output]   Compile project
                            path: project root path
                            output: output directory (optional, default: dist/ngc-go)
  watch <path>              Watch and compile (placeholder)
  help                      Show help`)
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
		outputPath := ""
		if len(os.Args) >= 3 {
			path = os.Args[2]
		}
		if len(os.Args) >= 4 {
			outputPath = os.Args[3]
		}
		if err := compile(path, outputPath); err != nil {
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

func compile(root string, outputPath string) error {
	// Import from compiler-cli package instead
	// For now, keep using local function
	return CompileProject(root, outputPath)
}
