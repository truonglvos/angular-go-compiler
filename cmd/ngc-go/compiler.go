package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"ngc-go/packages/compiler/src/ml_parser"
)

// ComponentInfo contains information about an Angular component
type ComponentInfo struct {
	FilePath    string
	ClassName   string
	Selector    string
	Template    string
	TemplateUrl string
	Styles      []string
	StyleUrls   []string
}

// CompileProject compiles an Angular project
func CompileProject(rootPath string) error {
	fmt.Printf("🔨 Compiling Angular project at: %s\n", rootPath)
	fmt.Println("")

	// Find all TypeScript component files
	components, err := findComponents(rootPath)
	if err != nil {
		return fmt.Errorf("error finding components: %v", err)
	}

	if len(components) == 0 {
		fmt.Println("⚠️  No Angular components found")
		return nil
	}

	fmt.Printf("📦 Found %d component(s)\n", len(components))
	fmt.Println("")

	// Create output directory
	outputDir := filepath.Join(rootPath, "dist", "ngc-go")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("error creating output directory: %v", err)
	}

	// Compile each component
	successCount := 0
	for i, comp := range components {
		fmt.Printf("[%d/%d] Compiling %s...\n", i+1, len(components), comp.ClassName)

		if err := compileComponent(comp, outputDir); err != nil {
			fmt.Printf("   ❌ Error: %v\n", err)
			continue
		}

		successCount++
		fmt.Printf("   ✅ Compiled successfully\n")
	}

	fmt.Println("")
	fmt.Printf("✅ Compilation complete: %d/%d components compiled\n", successCount, len(components))

	if successCount < len(components) {
		return fmt.Errorf("some components failed to compile")
	}

	return nil
}

// findComponents finds all Angular components in the project
func findComponents(rootPath string) ([]ComponentInfo, error) {
	var components []ComponentInfo

	compRe := regexp.MustCompile(`@Component\s*\(\s*\{([\s\S]*?)\}\s*\)`)
	classRe := regexp.MustCompile(`export\s+class\s+(\w+)\s*(?:extends|implements)?`)
	templateRe := regexp.MustCompile(`template\s*:\s*` + "`" + `([\s\S]*?)` + "`")
	templateUrlRe := regexp.MustCompile(`templateUrl\s*:\s*['\"]([^'\"]+)['\"]`)
	selectorRe := regexp.MustCompile(`selector\s*:\s*['\"]([^'\"]+)['\"]`)

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			// Skip node_modules and dist directories
			if info.Name() == "node_modules" || info.Name() == "dist" {
				return filepath.SkipDir
			}
			return nil
		}

		if !strings.HasSuffix(path, ".ts") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		content := string(data)

		// Check if it's a component
		if !compRe.MatchString(content) {
			return nil
		}

		// Extract class name
		classMatch := classRe.FindStringSubmatch(content)
		if len(classMatch) < 2 {
			return nil
		}
		className := classMatch[1]

		// Extract component metadata
		compMatch := compRe.FindStringSubmatch(content)
		compBody := compMatch[1]

		comp := ComponentInfo{
			FilePath:  path,
			ClassName: className,
		}

		// Extract selector
		if selectorMatch := selectorRe.FindStringSubmatch(compBody); len(selectorMatch) >= 2 {
			comp.Selector = selectorMatch[1]
		}

		// Extract template
		if templateMatch := templateRe.FindStringSubmatch(compBody); len(templateMatch) >= 2 {
			comp.Template = templateMatch[1]
		} else if templateUrlMatch := templateUrlRe.FindStringSubmatch(compBody); len(templateUrlMatch) >= 2 {
			comp.TemplateUrl = templateUrlMatch[1]
		}

		components = append(components, comp)
		return nil
	})

	return components, err
}

// compileComponent compiles a single Angular component
func compileComponent(comp ComponentInfo, outputDir string) error {
	if comp.Template == "" && comp.TemplateUrl == "" {
		return fmt.Errorf("component has no template")
	}

	// For now, we'll parse the template if it's inline
	if comp.Template != "" {
		// Parse template using ml_parser
		htmlParser := ml_parser.NewHtmlParser()
		parseResult := htmlParser.Parse(comp.Template, comp.FilePath, nil)

		if len(parseResult.Errors) > 0 {
			return fmt.Errorf("error parsing template: %d errors found", len(parseResult.Errors))
		}

		// Create a compilation job
		// TODO: Integrate with full pipeline
		_ = outputDir

		fmt.Printf("   📝 Template parsed: %d nodes\n", len(parseResult.RootNodes))

		// For now, just create a placeholder output file
		outputFile := filepath.Join(outputDir, strings.ToLower(comp.ClassName)+".ngfactory.js")
		outputContent := fmt.Sprintf(`// Compiled by ngc-go
// Component: %s
// Selector: %s
// Template nodes: %d

export function %sFactory() {
  // TODO: Generate actual factory code
  return null;
}
`, comp.ClassName, comp.Selector, len(parseResult.RootNodes), comp.ClassName)

		if err := os.WriteFile(outputFile, []byte(outputContent), 0644); err != nil {
			return fmt.Errorf("error writing output file: %v", err)
		}

		fmt.Printf("   📄 Output: %s\n", outputFile)
	}

	return nil
}
