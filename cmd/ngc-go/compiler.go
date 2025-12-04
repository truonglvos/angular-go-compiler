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
func CompileProject(rootPath string, outputPath string) error {
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

	// Determine output directory
	var outputDir string
	if outputPath != "" {
		// Use provided output path (can be absolute or relative)
		if filepath.IsAbs(outputPath) {
			outputDir = outputPath
		} else {
			outputDir = filepath.Join(rootPath, outputPath)
		}
	} else {
		// Default output directory
		outputDir = filepath.Join(rootPath, "dist", "ngc-go")
	}

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("error creating output directory: %v", err)
	}
	
	fmt.Printf("📁 Output directory: %s\n", outputDir)
	fmt.Println("")

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

	// Improved regex patterns to handle various formats
	compRe := regexp.MustCompile(`@Component\s*\(\s*\{([\s\S]*?)\}\s*\)`)
	classRe := regexp.MustCompile(`export\s+(?:default\s+)?class\s+(\w+)\s*(?:extends|implements)?`)
	templateRe := regexp.MustCompile(`template\s*:\s*` + "`" + `([\s\S]*?)` + "`")
	// Support both single and double quotes, and handle ./ prefix
	templateUrlRe := regexp.MustCompile(`templateUrl\s*:\s*['\"]([^'\"]+)['\"]`)
	selectorRe := regexp.MustCompile(`selector\s*:\s*['\"]([^'\"]+)['\"]`)

	var filesChecked int
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

		filesChecked++
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		content := string(data)

		// Check if it's a component
		if !compRe.MatchString(content) {
			return nil
		}

		// Extract class name - improved regex to handle default exports
		classMatch := classRe.FindStringSubmatch(content)
		if len(classMatch) < 2 {
			fmt.Printf("   ⚠️  Found @Component in %s but could not extract class name\n", path)
			return nil
		}
		className := classMatch[1]

		// Extract component metadata
		compMatch := compRe.FindStringSubmatch(content)
		if len(compMatch) < 2 {
			return nil
		}
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

		fmt.Printf("   ✓ Found component: %s (selector: %s, template: %s)\n", 
			className, comp.Selector, 
			func() string {
				if comp.Template != "" {
					return "inline"
				}
				if comp.TemplateUrl != "" {
					return comp.TemplateUrl
				}
				return "none"
			}())

		components = append(components, comp)
		return nil
	})

	if err == nil {
		fmt.Printf("   📂 Scanned %d TypeScript files\n", filesChecked)
	}

	return components, err
}

// compileComponent compiles a single Angular component
func compileComponent(comp ComponentInfo, outputDir string) error {
	fmt.Printf("   🔍 Debug: Template='%s', TemplateUrl='%s'\n", 
		func() string {
			if comp.Template != "" {
				return fmt.Sprintf("inline (%d chars)", len(comp.Template))
			}
			return "empty"
		}(),
		comp.TemplateUrl)

	if comp.Template == "" && comp.TemplateUrl == "" {
		return fmt.Errorf("component has no template")
	}

	var templateContent string
	var templatePath string

	// Get template content - either inline or from file
	if comp.Template != "" {
		// Inline template
		templateContent = comp.Template
		templatePath = comp.FilePath
		fmt.Printf("   📝 Using inline template from: %s\n", templatePath)
	} else if comp.TemplateUrl != "" {
		// External template file - resolve path relative to component file
		componentDir := filepath.Dir(comp.FilePath)
		templatePath = filepath.Join(componentDir, comp.TemplateUrl)
		
		// Normalize path (handle ./ prefix)
		templatePath = filepath.Clean(templatePath)
		
		fmt.Printf("   🔍 Attempting to read template from: %s\n", templatePath)
		
		// Read template file
		data, err := os.ReadFile(templatePath)
		if err != nil {
			return fmt.Errorf("error reading template file %s: %v", templatePath, err)
		}
		templateContent = string(data)
		fmt.Printf("   📖 Read template from: %s (%d bytes)\n", templatePath, len(data))
	} else {
		return fmt.Errorf("no template or templateUrl found")
	}

	// Parse template using ml_parser
	fmt.Printf("   🔍 Parsing template (%d bytes)...\n", len(templateContent))
	htmlParser := ml_parser.NewHtmlParser()
	parseResult := htmlParser.Parse(templateContent, templatePath, nil)

	if len(parseResult.Errors) > 0 {
		errMsg := fmt.Sprintf("error parsing template: %d errors found", len(parseResult.Errors))
		for i, err := range parseResult.Errors {
			if i < 5 { // Show first 5 errors
				errMsg += fmt.Sprintf("\n      - %v", err)
			}
		}
		if len(parseResult.Errors) > 5 {
			errMsg += fmt.Sprintf("\n      ... and %d more errors", len(parseResult.Errors)-5)
		}
		return fmt.Errorf(errMsg)
	}

	fmt.Printf("   📝 Template parsed: %d nodes\n", len(parseResult.RootNodes))

	// Generate code from AST
	fmt.Printf("   🔧 Generating factory code...\n")
	codeGen := NewCodeGenerator()
	generatedCode := codeGen.Generate(parseResult.RootNodes, comp.ClassName)
	
	// Create output file
	outputFile := filepath.Join(outputDir, strings.ToLower(comp.ClassName)+".ngfactory.js")
	fmt.Printf("   🔍 Writing output file to: %s\n", outputFile)
	
	// Build final output with imports and metadata
	outputContent := fmt.Sprintf(`// Compiled by ngc-go
// Component: %s
// Selector: %s
// Template: %s
// Template nodes: %d

import { ɵɵelement, ɵɵelementStart, ɵɵelementEnd, ɵɵtext, ɵɵtextInterpolate, ɵɵattribute } from '@angular/core';

%s
`, comp.ClassName, comp.Selector, templatePath, len(parseResult.RootNodes), generatedCode)

	if err := os.WriteFile(outputFile, []byte(outputContent), 0644); err != nil {
		return fmt.Errorf("error writing output file %s: %v", outputFile, err)
	}

	fmt.Printf("   📄 Output file created: %s (%d bytes)\n", outputFile, len(outputContent))

	return nil
}
