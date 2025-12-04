package compiler

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"ngc-go/packages/compiler/src/config"
	"ngc-go/packages/compiler/src/ml_parser"
)

type Compiler struct {
	tsConfig     *config.TsConfig
	projectRoot  string
	tsconfigPath string
}

// NewCompiler creates a new compiler instance
func NewCompiler(tsconfigPath string) (*Compiler, error) {
	cfg, err := config.ParseTsConfig(tsconfigPath)
	if err != nil {
		return nil, err
	}

	absPath, _ := filepath.Abs(tsconfigPath)
	projectRoot := filepath.Dir(absPath)

	return &Compiler{
		tsConfig:     cfg,
		projectRoot:  projectRoot,
		tsconfigPath: absPath,
	}, nil
}

// Compile runs the compilation process
func (c *Compiler) Compile() error {
	fmt.Println("Starting compilation...")
	fmt.Printf("Project root: %s\n", c.projectRoot)

	// Discover files to compile
	files, err := c.discoverFiles()
	if err != nil {
		return fmt.Errorf("failed to discover files: %w", err)
	}

	fmt.Printf("Found %d files to compile\n", len(files))
	for _, file := range files {
		fmt.Printf("  - %s\n", file)
	}

	// For now, just look for .html files
	for _, file := range files {
		if strings.HasSuffix(file, ".html") {
			if err := c.compileTemplate(file); err != nil {
				fmt.Fprintf(os.Stderr, "Error compiling %s: %v\n", file, err)
			}
		}
	}

	return nil
}

// discoverFiles finds all files that need compilation
func (c *Compiler) discoverFiles() ([]string, error) {
	var files []string

	// If tsconfig specifies files explicitly, use those
	if len(c.tsConfig.Files) > 0 {
		for _, f := range c.tsConfig.Files {
			absPath := filepath.Join(c.projectRoot, f)
			files = append(files, absPath)

			// Also look for companion .html files
			htmlPath := strings.TrimSuffix(absPath, ".ts") + ".html"
			if _, err := os.Stat(htmlPath); err == nil {
				files = append(files, htmlPath)
			}
		}
	} else {
		// Otherwise, scan the project directory
		err := filepath.Walk(c.projectRoot, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && (strings.HasSuffix(path, ".ts") || strings.HasSuffix(path, ".html")) {
				files = append(files, path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	// Always scan src directory for additional HTML files
	srcDir := filepath.Join(c.projectRoot, "src")
	if info, err := os.Stat(srcDir); err == nil && info.IsDir() {
		filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && strings.HasSuffix(path, ".html") {
				// Check if not already in list
				exists := false
				for _, f := range files {
					if f == path {
						exists = true
						break
					}
				}
				if !exists {
					files = append(files, path)
				}
			}
			return nil
		})
	}

	return files, nil
}

// compileTemplate compiles a single HTML template file
func (c *Compiler) compileTemplate(path string) error {
	fmt.Printf("\n=== Compiling template: %s ===\n", path)

	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}

	fmt.Printf("Template content:\n%s\n", string(content))

	// Parse with ml_parser
	fmt.Println("\nParsing HTML...")
	result := c.parseHtml(string(content), path)

	if len(result.Errors) > 0 {
		fmt.Println("Parse errors:")
		for _, e := range result.Errors {
			fmt.Printf("  - %s\n", e.Error())
		}
	}

	fmt.Printf("Successfully parsed! Root nodes: %d\n", len(result.RootNodes))

	return nil
}

// parseHtml uses ml_parser to parse HTML content
func (c *Compiler) parseHtml(source string, sourceUrl string) *ml_parser.ParseTreeResult {
	// Get tag definition function from html_tags
	getTagDef := func(tagName string) ml_parser.TagDefinition {
		return ml_parser.GetHtmlTagDefinition(tagName)
	}

	parser := ml_parser.NewParser(getTagDef)
	result := parser.Parse(source, sourceUrl, &ml_parser.TokenizeOptions{
		TokenizeExpansionForms: nil,
	})
	return result
}
