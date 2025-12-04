package view

import (
	"fmt"
	"ngc-go/packages/compiler/src/core"
	"ngc-go/packages/compiler/src/expression_parser"
	"ngc-go/packages/compiler/src/ml_parser"
	"ngc-go/packages/compiler/src/render3"
	"ngc-go/packages/compiler/src/render3/view"
	viewi18n "ngc-go/packages/compiler/src/render3/view/i18n"
	"ngc-go/packages/compiler/src/schema"
	"ngc-go/packages/compiler/src/template_parser"
	"ngc-go/packages/compiler/src/util"
)

// ParseR3Options are options for parsing R3 templates
type ParseR3Options struct {
	PreserveWhitespaces *bool
	LeadingTriviaChars  []string
	IgnoreError         *bool
	SelectorlessEnabled *bool
}

// ParseR3 parses an HTML string to R3 AST nodes
// This is equivalent to the TypeScript parseR3 function
func ParseR3(input string, options *ParseR3Options) *Render3ParseResult {
	if options == nil {
		options = &ParseR3Options{}
	}

	htmlParser := ml_parser.NewHtmlParser()

	leadingTriviaChars := view.LEADING_TRIVIA_CHARS
	if options.LeadingTriviaChars != nil {
		leadingTriviaChars = options.LeadingTriviaChars
	}

	selectorlessEnabled := false
	if options.SelectorlessEnabled != nil {
		selectorlessEnabled = *options.SelectorlessEnabled
	}

	tokenizeOptions := &ml_parser.TokenizeOptions{
		TokenizeExpansionForms: boolPtr(true),
		TokenizeLet:            boolPtr(true),
		LeadingTriviaChars:     leadingTriviaChars,
		SelectorlessEnabled:    &selectorlessEnabled,
	}

	parseResult := htmlParser.Parse(input, "path:://to/template", tokenizeOptions)

	ignoreError := false
	if options.IgnoreError != nil {
		ignoreError = *options.IgnoreError
	}

	// Debug: log errors
	if len(parseResult.Errors) > 0 {
		fmt.Printf("ParseR3: HTML parser returned %d errors\n", len(parseResult.Errors))
		for i, e := range parseResult.Errors {
			fmt.Printf("  Error[%d]: %s\n", i, e.String())
		}
	}

	if len(parseResult.Errors) > 0 && !ignoreError {
		msg := ""
		for _, e := range parseResult.Errors {
			msg += e.String() + "\n"
		}
		panic(fmt.Errorf("parse error: %s", msg))
	}

	// Debug: log HTML parser output
	fmt.Printf("ParseR3: HTML parser returned %d root nodes\n", len(parseResult.RootNodes))
	for i, node := range parseResult.RootNodes {
		fmt.Printf("  HTML Node[%d]: %T\n", i, node)
	}

	// Process i18n meta (matches TypeScript: let htmlNodes = processI18nMeta(parseResult).rootNodes)
	htmlNodes := processI18nMeta(parseResult).RootNodes
	fmt.Printf("ParseR3: After processI18nMeta, %d nodes\n", len(htmlNodes))
	fmt.Printf("ParseR3: Calling HtmlAstToRender3Ast with %d html nodes\n", len(htmlNodes))

	preserveWhitespaces := false
	if options.PreserveWhitespaces != nil {
		preserveWhitespaces = *options.PreserveWhitespaces
	}

	// Apply whitespace visitor if !preserveWhitespaces (matches TypeScript)
	if !preserveWhitespaces {
		whitespaceVisitor := ml_parser.NewWhitespaceVisitor(true /* preserveSignificantWhitespace */, nil, true)
		visitedNodes := ml_parser.VisitAllWithSiblings(whitespaceVisitor, htmlNodes, nil)
		htmlNodes = convertToMlNodes(visitedNodes)
	}

	// Create binding parser (matches TypeScript)
	lexer := expression_parser.NewLexer()
	parser := expression_parser.NewParser(lexer, selectorlessEnabled)
	schemaRegistry := NewMockSchemaRegistry(
		map[string]bool{"invalidProp": false},
		map[string]string{"mappedAttr": "mappedProp"},
		map[string]bool{"unknown": false, "un-known": false},
		[]string{"onEvent"},
		[]string{"onEvent"},
	)
	bindingParser := template_parser.NewBindingParser(parser, schemaRegistry, []*util.ParseError{})

	// Convert HTML AST to R3 AST (matches TypeScript: htmlAstToRender3Ast)
	r3Result := view.HtmlAstToRender3Ast(htmlNodes, bindingParser, view.Render3ParseOptions{
		CollectCommentNodes: false,
		SelectorlessEnabled: selectorlessEnabled,
	})

	if len(r3Result.Errors) > 0 && !ignoreError {
		msg := ""
		for _, e := range r3Result.Errors {
			msg += e.String() + "\n"
		}
		panic(fmt.Errorf("parse error: %s", msg))
	}

	return &Render3ParseResult{
		Nodes:  r3Result.Nodes,
		Errors: r3Result.Errors,
	}
}

// Render3ParseResult represents the result of parsing R3 template
type Render3ParseResult struct {
	Nodes  []render3.Node
	Errors []*util.ParseError
}

// processI18nMeta processes i18n metadata
func processI18nMeta(htmlAstWithErrors *ml_parser.ParseTreeResult) *ml_parser.ParseTreeResult {
	i18nMetaVisitor := viewi18n.NewI18nMetaVisitor(false, true, nil, true)
	visitedNodes := ml_parser.VisitAll(i18nMetaVisitor, htmlAstWithErrors.RootNodes, nil)
	rootNodes := convertToMlNodes(visitedNodes)
	return ml_parser.NewParseTreeResult(rootNodes, htmlAstWithErrors.Errors)
}

// convertToMlNodes converts a slice of interface{} to []ml_parser.Node
func convertToMlNodes(visitedNodes []interface{}) []ml_parser.Node {
	nodes := make([]ml_parser.Node, len(visitedNodes))
	for i, v := range visitedNodes {
		if n, ok := v.(ml_parser.Node); ok {
			nodes[i] = n
		}
	}
	return nodes
}

// boolPtr returns a pointer to a bool value
func boolPtr(b bool) *bool {
	return &b
}

// MockSchemaRegistry is a mock implementation of ElementSchemaRegistry for testing
type MockSchemaRegistry struct {
	existingProperties map[string]bool
	attrPropMapping    map[string]string
	existingElements   map[string]bool
	invalidProperties  []string
	invalidAttributes  []string
}

// NewMockSchemaRegistry creates a new MockSchemaRegistry
func NewMockSchemaRegistry(
	existingProperties map[string]bool,
	attrPropMapping map[string]string,
	existingElements map[string]bool,
	invalidProperties []string,
	invalidAttributes []string,
) *MockSchemaRegistry {
	return &MockSchemaRegistry{
		existingProperties: existingProperties,
		attrPropMapping:    attrPropMapping,
		existingElements:   existingElements,
		invalidProperties:  invalidProperties,
		invalidAttributes:  invalidAttributes,
	}
}

// HasProperty checks if a property exists
func (m *MockSchemaRegistry) HasProperty(tagName string, property string, schemas []*core.SchemaMetadata) bool {
	if value, ok := m.existingProperties[property]; ok {
		return value
	}
	return true // Default to true if not specified
}

// HasElement checks if an element exists
func (m *MockSchemaRegistry) HasElement(tagName string, schemaMetas []*core.SchemaMetadata) bool {
	tagLower := tagName
	if value, ok := m.existingElements[tagLower]; ok {
		return value
	}
	return true // Default to true if not specified
}

// AllKnownElementNames returns all known element names
func (m *MockSchemaRegistry) AllKnownElementNames() []string {
	names := make([]string, 0, len(m.existingElements))
	for name := range m.existingElements {
		names = append(names, name)
	}
	return names
}

// SecurityContext returns the security context
func (m *MockSchemaRegistry) SecurityContext(selector string, property string, isAttribute bool) core.SecurityContext {
	return core.SecurityContextNONE
}

// GetMappedPropName returns the mapped property name
func (m *MockSchemaRegistry) GetMappedPropName(attrName string) string {
	if mapped, ok := m.attrPropMapping[attrName]; ok {
		return mapped
	}
	return attrName
}

// GetDefaultComponentElementName returns the default component element name
func (m *MockSchemaRegistry) GetDefaultComponentElementName() string {
	return "ng-component"
}

// ValidateProperty validates a property name
func (m *MockSchemaRegistry) ValidateProperty(name string) schema.PropertyValidationResult {
	for _, invalid := range m.invalidProperties {
		if invalid == name {
			return schema.PropertyValidationResult{
				Error: true,
				Msg:   fmt.Sprintf("Binding to property '%s' is disallowed for security reasons", name),
			}
		}
	}
	return schema.PropertyValidationResult{Error: false}
}

// ValidateAttribute validates an attribute name
func (m *MockSchemaRegistry) ValidateAttribute(name string) schema.PropertyValidationResult {
	for _, invalid := range m.invalidAttributes {
		if invalid == name {
			return schema.PropertyValidationResult{
				Error: true,
				Msg:   fmt.Sprintf("Binding to attribute '%s' is disallowed for security reasons", name),
			}
		}
	}
	return schema.PropertyValidationResult{Error: false}
}

// NormalizeAnimationStyleProperty normalizes an animation style property name
func (m *MockSchemaRegistry) NormalizeAnimationStyleProperty(propName string) string {
	return propName
}

// NormalizeAnimationStyleValue normalizes an animation style value
func (m *MockSchemaRegistry) NormalizeAnimationStyleValue(
	camelCaseProp string,
	userProvidedProp string,
	val interface{},
) schema.AnimationStyleValueResult {
	return schema.AnimationStyleValueResult{
		Error: "",
		Value: fmt.Sprintf("%v", val),
	}
}

// FindExpression finds an expression in a template by its string representation
// This is equivalent to the TypeScript findExpression function
func FindExpression(tmpl []render3.Node, expr string) expression_parser.AST {
	for _, node := range tmpl {
		result := findExpressionInNode(node, expr)
		if result != nil {
			// If result is ASTWithSource, return its AST
			if astWithSource, ok := result.(*expression_parser.ASTWithSource); ok {
				return astWithSource.AST
			}
			return result
		}
	}
	return nil
}

// findExpressionInNode finds an expression in a node
func findExpressionInNode(node render3.Node, expr string) expression_parser.AST {
	switch n := node.(type) {
	case *render3.Element:
		// Search in inputs, outputs, and children
		nodesToSearch := []render3.Node{}
		for _, input := range n.Inputs {
			nodesToSearch = append(nodesToSearch, input)
		}
		for _, output := range n.Outputs {
			nodesToSearch = append(nodesToSearch, output)
		}
		nodesToSearch = append(nodesToSearch, n.Children...)
		return FindExpression(nodesToSearch, expr)
	case *render3.Template:
		// Search in inputs, outputs, and children
		nodesToSearch := []render3.Node{}
		for _, input := range n.Inputs {
			nodesToSearch = append(nodesToSearch, input)
		}
		for _, output := range n.Outputs {
			nodesToSearch = append(nodesToSearch, output)
		}
		nodesToSearch = append(nodesToSearch, n.Children...)
		return FindExpression(nodesToSearch, expr)
	case *render3.Component:
		// Search in inputs, outputs, and children
		nodesToSearch := []render3.Node{}
		for _, input := range n.Inputs {
			nodesToSearch = append(nodesToSearch, input)
		}
		for _, output := range n.Outputs {
			nodesToSearch = append(nodesToSearch, output)
		}
		nodesToSearch = append(nodesToSearch, n.Children...)
		return FindExpression(nodesToSearch, expr)
	case *render3.Directive:
		// Search in inputs and outputs
		nodesToSearch := []render3.Node{}
		for _, input := range n.Inputs {
			nodesToSearch = append(nodesToSearch, input)
		}
		for _, output := range n.Outputs {
			nodesToSearch = append(nodesToSearch, output)
		}
		return FindExpression(nodesToSearch, expr)
	case *render3.BoundAttribute:
		ts := toStringExpression(n.Value)
		if ts == expr {
			return n.Value
		}
		return nil
	case *render3.BoundText:
		ts := toStringExpression(n.Value)
		if ts == expr {
			return n.Value
		}
		return nil
	case *render3.BoundEvent:
		ts := toStringExpression(n.Handler)
		if ts == expr {
			return n.Handler
		}
		return nil
	default:
		return nil
	}
}

// toStringExpression converts an AST to string representation
func toStringExpression(expr expression_parser.AST) string {
	// Unwrap ASTWithSource
	for {
		if astWithSource, ok := expr.(*expression_parser.ASTWithSource); ok {
			expr = astWithSource.AST
		} else {
			break
		}
	}

	switch e := expr.(type) {
	case *expression_parser.PropertyRead:
		if _, ok := e.Receiver.(*expression_parser.ImplicitReceiver); ok {
			return e.Name
		} else {
			return toStringExpression(e.Receiver) + "." + e.Name
		}
	case *expression_parser.ImplicitReceiver:
		return ""
	case *expression_parser.Interpolation:
		str := "{{"
		for i := 0; i < len(e.Expressions); i++ {
			if i < len(e.Strings) {
				str += e.Strings[i]
			}
			str += toStringExpression(e.Expressions[i])
		}
		if len(e.Strings) > 0 {
			str += e.Strings[len(e.Strings)-1]
		}
		str += "}}"
		return str
	default:
		return ""
	}
}
