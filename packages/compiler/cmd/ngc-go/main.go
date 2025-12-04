package main

import (
	"flag"
	"fmt"
	compiler "ngc-go/packages/compiler/src"
	"os"
	"path/filepath"
)

func main() {
	project := flag.String("p", "tsconfig.json", "Path to tsconfig.json")
	flag.Parse()

	absPath, err := filepath.Abs(*project)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving path: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Compiling Angular project: %s\n", absPath)

	// TODO: Load tsconfig and start compilation
	if err := runCompilation(absPath); err != nil {
		fmt.Fprintf(os.Stderr, "Compilation failed: %v\n", err)
		os.Exit(1)
	}
}

func runCompilation(tsConfigPath string) error {
	// Use the new compiler facade
	comp, err := compiler.NewCompiler(tsConfigPath)
	if err != nil {
		return fmt.Errorf("failed to create compiler: %w", err)
	}

	return comp.Compile()
}
