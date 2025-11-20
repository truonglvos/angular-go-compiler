package ml_parser

import "ngc-go/packages/compiler/util"

// TreeError represents a tree parsing error
type TreeError struct {
	*util.ParseError
	ElementName *string
}

// NewTreeError creates a new TreeError
func NewTreeError(elementName *string, span *util.ParseSourceSpan, msg string) *TreeError {
	return &TreeError{
		ParseError:  util.NewParseError(span, msg),
		ElementName: elementName,
	}
}

// ParseTreeResult represents the result of parsing a tree
type ParseTreeResult struct {
	RootNodes []Node
	Errors    []*util.ParseError
}

// NewParseTreeResult creates a new ParseTreeResult
func NewParseTreeResult(rootNodes []Node, errors []*util.ParseError) *ParseTreeResult {
	return &ParseTreeResult{
		RootNodes: rootNodes,
		Errors:    errors,
	}
}

// TokenizeOptions represents options for tokenization
type TokenizeOptions struct {
	TokenizeExpansionForms         *bool
	Range                          *LexerRange
	EscapedString                  *bool
	I18nNormalizeLineEndingsInICUs *bool
	LeadingTriviaChars             []string
	PreserveLineEndings            *bool
	TokenizeBlocks                 *bool
	TokenizeLet                    *bool
	SelectorlessEnabled            *bool
}

// LexerRange represents a range in the source
type LexerRange struct {
	StartPos  int
	StartLine int
	StartCol  int
	EndPos    int
}

// TokenizeResult represents the result of tokenization
type TokenizeResult struct {
	Tokens                      []Token
	Errors                      []*util.ParseError
	NonNormalizedIcuExpressions []Token
}

// NewTokenizeResult creates a new TokenizeResult
func NewTokenizeResult(tokens []Token, errors []*util.ParseError, nonNormalizedIcuExpressions []Token) *TokenizeResult {
	return &TokenizeResult{
		Tokens:                      tokens,
		Errors:                      errors,
		NonNormalizedIcuExpressions: nonNormalizedIcuExpressions,
	}
}

// Parser parses HTML/XML source into an AST
type Parser struct {
	GetTagDefinition func(tagName string) TagDefinition
}

// NewParser creates a new Parser
func NewParser(getTagDefinition func(tagName string) TagDefinition) *Parser {
	return &Parser{
		GetTagDefinition: getTagDefinition,
	}
}

// Parse parses source code into a ParseTreeResult
func (p *Parser) Parse(source, url string, options *TokenizeOptions) *ParseTreeResult {
	// TODO: Implement tokenization and tree building
	// This is a placeholder - full implementation will require lexer.go
	tokenizeResult := Tokenize(source, url, p.GetTagDefinition, options)
	treeBuilder := NewTreeBuilder(tokenizeResult.Tokens, p.GetTagDefinition)
	treeBuilder.Build()

	allErrors := tokenizeResult.Errors
	for _, err := range treeBuilder.Errors() {
		allErrors = append(allErrors, err.ParseError)
	}
	return NewParseTreeResult(treeBuilder.RootNodes(), allErrors)
}

// Tokenize tokenizes source code
func Tokenize(source, url string, getTagDefinition func(tagName string) TagDefinition, options *TokenizeOptions) *TokenizeResult {
	// TODO: Implement full tokenization
	// This requires lexer.go implementation
	// For now, return empty result
	return NewTokenizeResult([]Token{}, []*util.ParseError{}, []Token{})
}

// TreeBuilder builds a tree from tokens
type TreeBuilder struct {
	index                 int
	peek                  Token
	containerStack        []NodeContainer
	rootNodes             []Node
	errors                []*TreeError
	tokens                []Token
	tagDefinitionResolver func(tagName string) TagDefinition
}

// NodeContainer represents a container node
type NodeContainer interface {
	Node
}

// NewTreeBuilder creates a new TreeBuilder
func NewTreeBuilder(tokens []Token, tagDefinitionResolver func(tagName string) TagDefinition) *TreeBuilder {
	tb := &TreeBuilder{
		index:                 -1,
		containerStack:        []NodeContainer{},
		rootNodes:             []Node{},
		errors:                []*TreeError{},
		tokens:                tokens,
		tagDefinitionResolver: tagDefinitionResolver,
	}
	if len(tokens) > 0 {
		tb.advance()
	}
	return tb
}

// Build builds the tree from tokens
func (tb *TreeBuilder) Build() {
	// TODO: Implement full tree building logic
	// This is a placeholder - requires full implementation
	for tb.peek != nil && tb.peek.Type() != TokenTypeEOF {
		// Process tokens and build tree
		tb.advance()
	}

	// Check for unclosed containers
	for _, container := range tb.containerStack {
		if block, ok := container.(*Block); ok {
			tb.errors = append(tb.errors, NewTreeError(
				&block.Name,
				block.SourceSpan(),
				"Unclosed block \""+block.Name+"\"",
			))
		}
	}
}

func (tb *TreeBuilder) advance() Token {
	if tb.index < len(tb.tokens)-1 {
		tb.index++
	}
	if tb.index < len(tb.tokens) {
		tb.peek = tb.tokens[tb.index]
		return tb.peek
	}
	return nil
}

// RootNodes returns the root nodes
func (tb *TreeBuilder) RootNodes() []Node {
	return tb.rootNodes
}

// Errors returns the errors
func (tb *TreeBuilder) Errors() []*TreeError {
	return tb.errors
}
