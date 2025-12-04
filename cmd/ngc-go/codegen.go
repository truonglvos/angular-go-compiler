package main

import (
	"fmt"
	"strings"

	ml_parser "ngc-go/packages/compiler/src/ml_parser"
)

// CodeGenerator generates JavaScript/TypeScript code from template AST
type CodeGenerator struct {
	indentLevel int
	builder     strings.Builder
	elementIdx  int
}

// NewCodeGenerator creates a new code generator
func NewCodeGenerator() *CodeGenerator {
	return &CodeGenerator{
		indentLevel: 0,
		elementIdx:  0,
	}
}

// Generate generates code from template AST nodes
func (cg *CodeGenerator) Generate(rootNodes []ml_parser.Node, componentName string) string {
	cg.builder.Reset()
	cg.elementIdx = 0

	// Generate factory function
	cg.write("export function %sFactory() {\n", componentName)
	cg.indentLevel++
	cg.write("return function %s_Template(rf, ctx) {\n", componentName)
	cg.indentLevel++

	// Generate instructions for each root node
	for _, node := range rootNodes {
		cg.generateNode(node)
	}

	cg.indentLevel--
	cg.write("};\n")
	cg.indentLevel--
	cg.write("}\n")

	return cg.builder.String()
}

// generateNode generates code for a single AST node
func (cg *CodeGenerator) generateNode(node ml_parser.Node) {
	switch n := node.(type) {
	case *ml_parser.Element:
		cg.generateElement(n)
	case *ml_parser.Text:
		cg.generateText(n)
	case *ml_parser.Comment:
		cg.generateComment(n)
	case *ml_parser.Component:
		cg.generateComponent(n)
	case *ml_parser.Block:
		cg.generateBlock(n)
	case *ml_parser.Expansion:
		cg.generateExpansion(n)
	default:
		// Unknown node type - skip or log warning
		cg.write("// Unknown node type: %T\n", node)
	}
}

// generateElement generates code for an element node
func (cg *CodeGenerator) generateElement(elem *ml_parser.Element) {
	idx := cg.elementIdx
	cg.elementIdx++

	// Generate element creation instruction
	// Format: ɵɵelementStart(index, name) ... ɵɵelementEnd()
	cg.write("if (rf & 1) {\n")
	cg.indentLevel++
	cg.write("ɵɵelementStart(%d, \"%s\");\n", idx, elem.Name)

	// Add attributes
	for _, attr := range elem.Attrs {
		cg.write("ɵɵattribute(\"%s\", \"%s\");\n", attr.Name, escapeString(attr.Value))
	}

	// Generate children (in creation mode)
	for _, child := range elem.Children {
		cg.generateNode(child)
	}

	cg.write("ɵɵelementEnd();\n")
	cg.indentLevel--
	cg.write("}\n")

	// Generate update instructions (rf & 2)
	if len(elem.Children) > 0 {
		cg.write("if (rf & 2) {\n")
		cg.indentLevel++
		// For now, we'll generate update instructions for children
		// In a full implementation, this would handle bindings, etc.
		cg.indentLevel--
		cg.write("}\n")
	}
}

// generateText generates code for a text node
func (cg *CodeGenerator) generateText(text *ml_parser.Text) {
	// Generate text instruction
	// Format: ɵɵtext(index, value)
	idx := cg.elementIdx
	cg.elementIdx++

	cg.write("if (rf & 1) {\n")
	cg.indentLevel++

	// Check if text has interpolation
	if len(text.Tokens) > 0 {
		// Has interpolation - use ɵɵtextInterpolate
		value := text.Value
		cg.write("ɵɵtext(%d);\n", idx)
		cg.write("ɵɵtextInterpolate(\"%s\");\n", escapeString(value))
	} else {
		// Plain text
		cg.write("ɵɵtext(%d, \"%s\");\n", idx, escapeString(text.Value))
	}

	cg.indentLevel--
	cg.write("}\n")
}

// generateComment generates code for a comment node
func (cg *CodeGenerator) generateComment(comment *ml_parser.Comment) {
	// Comments are typically ignored in Angular templates during compilation
	// but we can generate them for debugging
	if comment.Value != nil {
		cg.write("// %s\n", *comment.Value)
	}
}

// generateComponent generates code for a component node
func (cg *CodeGenerator) generateComponent(comp *ml_parser.Component) {
	idx := cg.elementIdx
	cg.elementIdx++

	// Generate component instruction
	// Format: ɵɵelementStart(index, name) ... ɵɵelementEnd()
	cg.write("if (rf & 1) {\n")
	cg.indentLevel++

	tagName := comp.ComponentName
	if comp.TagName != nil {
		tagName = *comp.TagName
	}

	cg.write("ɵɵelementStart(%d, \"%s\");\n", idx, tagName)

	// Add attributes
	for _, attr := range comp.Attrs {
		cg.write("ɵɵattribute(\"%s\", \"%s\");\n", attr.Name, escapeString(attr.Value))
	}

	// Generate children
	for _, child := range comp.Children {
		cg.generateNode(child)
	}

	cg.write("ɵɵelementEnd();\n")
	cg.indentLevel--
	cg.write("}\n")
}

// generateBlock generates code for a block node (e.g., @if, @for)
func (cg *CodeGenerator) generateBlock(block *ml_parser.Block) {
	// Generate block instruction based on block type
	// This is a simplified version - full implementation would handle different block types
	cg.write("// Block: %s\n", block.Name)

	// Generate children
	for _, child := range block.Children {
		cg.generateNode(child)
	}
}

// generateExpansion generates code for an ICU expansion node
func (cg *CodeGenerator) generateExpansion(exp *ml_parser.Expansion) {
	// ICU expansions are complex - simplified version for now
	cg.write("// ICU Expansion: %s\n", exp.Type)

	for _, expCase := range exp.Cases {
		cg.write("// Case: %s\n", expCase.Value)
		for _, child := range expCase.Expression {
			cg.generateNode(child)
		}
	}
}

// write writes a formatted string to the builder
func (cg *CodeGenerator) write(format string, args ...interface{}) {
	indent := strings.Repeat("  ", cg.indentLevel)
	cg.builder.WriteString(indent)
	cg.builder.WriteString(fmt.Sprintf(format, args...))
}

// escapeString escapes special characters in a string for JavaScript
func escapeString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	s = strings.ReplaceAll(s, "\t", "\\t")
	return s
}
