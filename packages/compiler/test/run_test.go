package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	baseDir := "/Users/truong/Documents/go/packages/compiler/test"
	
	// Test util
	fmt.Println("=== Running util tests ===")
	utilDir := filepath.Join(baseDir, "util")
	cmd := exec.Command("go", "test", "-v")
	cmd.Dir = utilDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		fmt.Printf("Util tests failed: %v\n", err)
	} else {
		fmt.Println("Util tests passed!")
	}
	
	fmt.Println("\n=== Running expression_parser tests ===")
	exprDir := filepath.Join(baseDir, "expression_parser")
	cmd2 := exec.Command("go", "test", "-v")
	cmd2.Dir = exprDir
	cmd2.Stdout = os.Stdout
	cmd2.Stderr = os.Stderr
	err2 := cmd2.Run()
	if err2 != nil {
		fmt.Printf("Expression parser tests failed: %v\n", err2)
	} else {
		fmt.Println("Expression parser tests passed!")
	}
}

