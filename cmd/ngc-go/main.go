package main

import (
	"fmt"
	"os"
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
	// Import from compiler-cli package instead
	// For now, keep using local function
	return CompileProject(root)
}
