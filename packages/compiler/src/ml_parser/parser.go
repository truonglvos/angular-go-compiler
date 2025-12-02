package ml_parser

import (
	"fmt"
	"ngc-go/packages/compiler/src/util"
	"strings"
)

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
	fmt.Printf("[DEBUG] Parse: START, source length=%d\n", len(source))
	tokenizeResult := Tokenize(source, url, p.GetTagDefinition, options)
	fmt.Printf("[DEBUG] Parse: Tokenize done, tokens count=%d, errors count=%d\n", len(tokenizeResult.Tokens), len(tokenizeResult.Errors))
	treeBuilder := NewTreeBuilder(tokenizeResult.Tokens, p.GetTagDefinition)
	fmt.Printf("[DEBUG] Parse: TreeBuilder created, calling Build()\n")
	treeBuilder.Build()
	fmt.Printf("[DEBUG] Parse: Build() done\n")

	// Combine errors from tokenization and tree building
	allErrors := tokenizeResult.Errors
	for _, err := range treeBuilder.Errors() {
		// Preserve TreeError information by keeping the TreeError itself
		// We'll need to convert it to ParseError for compatibility, but we can
		// access ElementName from the original TreeError in humanizeErrors
		allErrors = append(allErrors, err.ParseError)
	}
	return NewParseTreeResult(treeBuilder.RootNodes(), allErrors)
}

// Tokenize tokenizes source code
func Tokenize(source, url string, getTagDefinition func(tagName string) TagDefinition, options *TokenizeOptions) *TokenizeResult {
	file := util.NewParseSourceFile(source, url)
	tokenizer := NewTokenizer(file, getTagDefinition, options)
	tokenizer.Tokenize()
	return NewTokenizeResult(tokenizer.tokens, tokenizer.errors, tokenizer.nonNormalizedIcuExpressions)
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
	inExpansionContext    bool // Flag to indicate we're parsing expansion case expression
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
		inExpansionContext:    false,
	}
	if len(tokens) > 0 {
		tb.advance()
	}
	return tb
}

// Build builds the tree from tokens
func (tb *TreeBuilder) Build() {
	fmt.Printf("[DEBUG] Build: START, totalTokens=%d, index=%d\n", len(tb.tokens), tb.index)
	// Debug: print all tokens
	if len(tb.tokens) <= 20 {
		fmt.Printf("[DEBUG] Build: all tokens:\n")
		for i, token := range tb.tokens {
			tokenType := token.Type()
			parts := token.Parts()
			if len(parts) > 0 && len(parts[0]) > 50 {
				parts = []string{parts[0][:50] + "..."}
			}
			fmt.Printf("[DEBUG] Build: token[%d]: type=%d, parts=%v\n", i, tokenType, parts)
		}
	}
	if tb.peek != nil {
		fmt.Printf("[DEBUG] Build: initial peek.Type()=%d\n", tb.peek.Type())
	} else {
		fmt.Printf("[DEBUG] Build: initial peek is nil\n")
	}
	buildIterationCount := 0
	for tb.peek != nil && tb.peek.Type() != TokenTypeEOF {
		buildIterationCount++
		if buildIterationCount > 1000 {
			fmt.Printf("[DEBUG] Build: INFINITE LOOP DETECTED! iterationCount=%d, peek.Type()=%d, index=%d, totalTokens=%d\n",
				buildIterationCount, func() int {
					if tb.peek == nil {
						return -1
					}
					return int(tb.peek.Type())
				}(), tb.index, len(tb.tokens))
			break
		}
		if buildIterationCount <= 20 || buildIterationCount%100 == 0 {
			fmt.Printf("[DEBUG] Build: iteration=%d, peek.Type()=%d, index=%d, totalTokens=%d\n",
				buildIterationCount, func() int {
					if tb.peek == nil {
						return -1
					}
					return int(tb.peek.Type())
				}(), tb.index, len(tb.tokens))
		}
		switch tb.peek.Type() {
		case TokenTypeTAG_OPEN_START:
			token := tb.advance()
			parts := token.Parts()
			prefix := ""
			name := ""
			if len(parts) > 0 {
				prefix = strings.TrimSpace(parts[0])
			}
			if len(parts) > 1 {
				name = strings.TrimSpace(parts[1])
			}
			fmt.Printf("[DEBUG] Build: found TAG_OPEN_START, index=%d, totalTokens=%d, parts=%v, prefix=%q, name=%q\n", tb.index, len(tb.tokens), parts, prefix, name)
			// Try to get the underlying TagOpenStartToken
			// If token is TokenBase, we need to reconstruct it
			var startTag *TagOpenStartToken
			if tagToken, ok := token.(*TagOpenStartToken); ok {
				startTag = tagToken
			} else if baseToken, ok := token.(*TokenBase); ok {
				// Reconstruct TagOpenStartToken from TokenBase
				startTag = NewTagOpenStartToken(prefix, name, baseToken.SourceSpan())
			}
			if startTag != nil {
				fmt.Printf("[DEBUG] Build: calling _consumeStartTag, prefix=%q, name=%q\n", prefix, name)
				tb._consumeStartTag(startTag)
			} else {
				// This case should ideally not happen if tokenization is correct
				// but keeping the error for robustness.
				fmt.Printf("[DEBUG] Build: ERROR - cannot create TagOpenStartToken from token type %T\n", token)
				tb.errors = append(tb.errors, NewTreeError(
					nil,
					token.SourceSpan(),
					fmt.Sprintf("Unexpected token type %T for TAG_OPEN_START", token),
				))
			}
		case TokenTypeINCOMPLETE_TAG_OPEN:
			token := tb.advance()
			// IncompleteTagOpenToken has the same structure as TagOpenStartToken
			if incompleteTag, ok := token.(*TagOpenStartToken); ok {
				tb._consumeIncompleteStartTag(incompleteTag)
			}
		case TokenTypeTAG_CLOSE:
			token := tb.advance()
			if closeTag, ok := token.(*TagCloseToken); ok {
				tb._consumeEndTag(closeTag)
			}
		case TokenTypeCDATA_START:
			tb._closeVoidElement()
			tb._consumeCdata(tb.advance())
		case TokenTypeCOMMENT_START:
			tb._closeVoidElement()
			tb._consumeComment(tb.advance())
		case TokenTypeTEXT, TokenTypeRAW_TEXT, TokenTypeESCAPABLE_RAW_TEXT:
			tb._closeVoidElement()
			token := tb.advance()
			if textToken, ok := token.(*TextToken); ok {
				tb._consumeText(textToken)
			}
		case TokenTypeINTERPOLATION:
			// Interpolation tokens contain the expression content
			tb._closeVoidElement()
			token := tb.advance()
			// Get the interpolation content from token parts
			parts := token.Parts()
			if len(parts) > 0 {
				content := strings.Join(parts, "")
				fmt.Printf("[DEBUG] Build: INTERPOLATION token (standalone), parts=%v, content=%q\n", parts, content)
				var tokens []InterpolatedTextToken
				if interpToken, ok := token.(InterpolatedTextToken); ok {
					tokens = append(tokens, interpToken)
				}
				tb._addToParent(NewText(content, token.SourceSpan(), tokens, nil))
			}
		case TokenTypeDOC_TYPE:
			tb._closeVoidElement()
			tb._consumeDocType(tb.advance())
		case TokenTypeEXPANSION_FORM_START:
			tb._closeVoidElement()
			fmt.Printf("[DEBUG] Build: found EXPANSION_FORM_START, index=%d, totalTokens=%d\n", tb.index, len(tb.tokens))
			tb._consumeExpansion(tb.advance())
		case TokenTypeBLOCK_OPEN_START:
			tb._closeVoidElement()
			fmt.Printf("[DEBUG] Build: found BLOCK_OPEN_START, index=%d, totalTokens=%d\n", tb.index, len(tb.tokens))
			tb._consumeBlockOpen(tb.advance())
		case TokenTypeBLOCK_CLOSE:
			tb._closeVoidElement()
			fmt.Printf("[DEBUG] Build: found BLOCK_CLOSE, index=%d, totalTokens=%d, inExpansionContext=%v\n", tb.index, len(tb.tokens), tb.inExpansionContext)
			// In expansion context, BLOCK_CLOSE tokens should be ignored
			// They were likely meant to be EXPANSION_FORM_END or EXPANSION_CASE_EXP_END
			// but were tokenized as BLOCK_CLOSE due to lexer ordering
			if tb.inExpansionContext {
				fmt.Printf("[DEBUG] Build: skipping BLOCK_CLOSE in expansion context\n")
				tb.advance()
			} else {
				tb._consumeBlockClose(tb.advance())
			}
		case TokenTypeINCOMPLETE_BLOCK_OPEN:
			tb._closeVoidElement()
			fmt.Printf("[DEBUG] Build: found INCOMPLETE_BLOCK_OPEN, index=%d, totalTokens=%d\n", tb.index, len(tb.tokens))
			tb._consumeIncompleteBlock(tb.advance())
		case TokenTypeLET_START:
			tb._closeVoidElement()
			tb._consumeLet(tb.advance())
		case TokenTypeINCOMPLETE_LET:
			tb._closeVoidElement()
			tb._consumeIncompleteLet(tb.advance())
		case TokenTypeCOMPONENT_OPEN_START, TokenTypeINCOMPLETE_COMPONENT_OPEN:
			tb._closeVoidElement()
			tb._consumeComponentStartTag(tb.advance())
		case TokenTypeCOMPONENT_CLOSE:
			token := tb.advance()
			tb._consumeComponentEndTag(token)
		case TokenTypeATTR_VALUE_TEXT, TokenTypeATTR_VALUE_INTERPOLATION, TokenTypeATTR_QUOTE:
			// These tokens should only appear within attribute context, but if they appear
			// at top level (e.g., due to premature tag start in attribute value), skip them
			fmt.Printf("[DEBUG] Build: found attribute token at top level, skipping: type=%d, index=%d\n", tb.peek.Type(), tb.index)
			tb.advance()
		case TokenTypeEXPANSION_CASE_EXP_END, TokenTypeEXPANSION_FORM_END:
			// These tokens are delimiters for nested expansion forms within expansion case expressions
			// Skip them (matches TypeScript behavior where unmatched tokens are skipped)
			if buildIterationCount <= 10 {
				fmt.Printf("[DEBUG] Build: skipping expansion delimiter token, type=%d, index=%d\n", tb.peek.Type(), tb.index)
			}
			tb.advance()
		default:
			// Skip unknown token
			tb.advance()
		}
	}

	// Check for unclosed containers
	// Unlike HTML elements, blocks aren't closed implicitly by the end of the file.
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

func (tb *TreeBuilder) _consumeStartTag(startTag *TagOpenStartToken) {
	attrs := []*Attribute{}
	directives := []*Directive{}
	tb._consumeAttributesAndDirectives(&attrs, &directives)

	parent := tb._getClosestElementLikeParent()
	fullName := tb._getElementFullName(startTag, parent)
	tagDef := tb._getTagDefinition(fullName)
	isSelfClosing := false

	fmt.Printf("[DEBUG] _consumeStartTag: START, fullName=%q, peek.Type()=%d (TAG_OPEN_END_VOID=%d), index=%d, totalTokens=%d\n",
		fullName, func() int {
			if tb.peek == nil {
				return -1
			}
			return int(tb.peek.Type())
		}(), TokenTypeTAG_OPEN_END_VOID, tb.index, len(tb.tokens))

	// Check if this is a component (name starts with uppercase)
	// HTML tags are typically lowercase, so PascalCase names are likely components
	parts := startTag.Parts()
	var tagName string
	if len(parts) > 1 {
		tagName = parts[1]
	}
	// Check if tag name starts with uppercase (PascalCase convention for Angular components)
	isComponent := len(tagName) > 0 && tagName[0] >= 'A' && tagName[0] <= 'Z'

	// Check for end token
	if tb.peek != nil && tb.peek.Type() == TokenTypeTAG_OPEN_END_VOID {
		tb.advance()
		isSelfClosing = true
		fmt.Printf("[DEBUG] _consumeStartTag: found TAG_OPEN_END_VOID, fullName=%q, tagDef=%v, CanSelfClose=%v, IsVoid=%v, GetNsPrefix=%v, isComponent=%v\n",
			fullName, tagDef != nil, func() bool {
				if tagDef != nil {
					return tagDef.CanSelfClose()
				}
				return false
			}(), func() bool {
				if tagDef != nil {
					return tagDef.IsVoid()
				}
				return false
			}(), GetNsPrefix(&fullName) != nil, isComponent)
		// Only void, custom (component), and foreign (namespace prefix) elements can be self closed
		// TypeScript logic: if (!(tagDef?.canSelfClose || getNsPrefix(fullName) !== null || tagDef?.isVoid))
		// This means: allow self-close if tagDef.canSelfClose is true OR has namespace prefix OR is void
		// DEFAULT_TAG_DEFINITION has canSelfClose=true, which allows custom elements to self-close
		// Standard HTML tags (like 'b') have specific tag definitions without canSelfClose=true (unless void)
		// So the logic works: custom elements get DEFAULT_TAG_DEFINITION (canSelfClose=true), standard tags don't
		canSelfClose := tagDef != nil && tagDef.CanSelfClose()
		isVoid := tagDef != nil && tagDef.IsVoid()
		hasNamespacePrefix := GetNsPrefix(&fullName) != nil
		// Match TypeScript logic exactly
		if !(canSelfClose || hasNamespacePrefix || isVoid) {
			errMsg := fmt.Sprintf("Only void, custom and foreign elements can be self closed \"%s\"", startTag.Parts()[1])
			fmt.Printf("[DEBUG] _consumeStartTag: adding error: %q\n", errMsg)
			tb.errors = append(tb.errors, NewTreeError(
				&fullName,
				startTag.SourceSpan(),
				errMsg,
			))
			fmt.Printf("[DEBUG] _consumeStartTag: after adding error, len(errors)=%d\n", len(tb.errors))
		}
	} else if tb.peek != nil && tb.peek.Type() == TokenTypeTAG_OPEN_END {
		tb.advance()
		isSelfClosing = false
	}

	end := tb.peek.SourceSpan().FullStart
	span := util.NewParseSourceSpan(startTag.SourceSpan().Start, end, startTag.SourceSpan().FullStart, nil)
	startSpan := util.NewParseSourceSpan(startTag.SourceSpan().Start, end, startTag.SourceSpan().FullStart, nil)

	if isComponent {
		// Create Component node
		componentName := tagName
		var componentTagName *string
		componentFullName := componentName

		component := NewComponent(
			componentName,
			componentTagName,
			componentFullName,
			attrs,
			directives,
			[]Node{},
			isSelfClosing,
			span,
			startSpan,
			nil,
			nil,
		)

		parentContainer := tb._getContainer()
		isClosedByChild := false
		if parentContainer != nil {
			parentTagDef := tb._getTagDefinition(parentContainer)
			if parentTagDef != nil {
				isClosedByChild = parentTagDef.IsClosedByChild(componentName)
			}
		}
		tb._pushContainer(component, isClosedByChild)

		if isSelfClosing {
			// Components that are self-closed have their `endSourceSpan` set to the full span
			var componentType *Component
			popped := tb._popContainer(&componentFullName, componentType, span)
			if !popped {
				component.EndSourceSpan = span
				if span != nil {
					component.sourceSpan = util.NewParseSourceSpan(component.sourceSpan.Start, span.End, component.sourceSpan.FullStart, component.sourceSpan.Details)
				}
				if len(tb.containerStack) > 0 {
					lastIdx := len(tb.containerStack) - 1
					lastNode := tb.containerStack[lastIdx]
					if lastNode == component {
						tb.containerStack = tb.containerStack[:lastIdx]
					} else if comp, ok := lastNode.(*Component); ok && comp.FullName == componentFullName {
						tb.containerStack = tb.containerStack[:lastIdx]
					}
				}
			}
		}
		return
	}

	// Create Element node
	isVoid := false
	if tagDef != nil {
		isVoid = tagDef.IsVoid()
	}

	element := NewElement(
		fullName,
		attrs,
		directives,
		[]Node{},
		isSelfClosing,
		span,
		startSpan,
		nil,
		isVoid,
		nil,
	)

	parentContainer := tb._getContainer()
	isClosedByChild := false
	if parentContainer != nil {
		parentTagDef := tb._getTagDefinition(parentContainer)
		if parentTagDef != nil {
			isClosedByChild = parentTagDef.IsClosedByChild(element.Name)
		}
	}
	tb._pushContainer(element, isClosedByChild)

	if isSelfClosing {
		// Elements that are self-closed have their `endSourceSpan` set to the full span
		// as the element start tag also represents the end tag.
		// The element was just pushed to the stack, so it should be the last element
		// Use _popContainer to properly handle it (matches TypeScript: this._popContainer(fullName, html.Element, span))
		var elementType *Element
		popped := tb._popContainer(&fullName, elementType, span)
		if !popped {
			// Fallback: if _popContainer failed, manually handle self-closing element
			// This ensures the element is properly closed and removed from stack
			element.EndSourceSpan = span
			if span != nil {
				element.sourceSpan = util.NewParseSourceSpan(element.sourceSpan.Start, span.End, element.sourceSpan.FullStart, element.sourceSpan.Details)
			}
			// Remove from stack - element should be the last one since it was just pushed
			if len(tb.containerStack) > 0 {
				lastIdx := len(tb.containerStack) - 1
				lastNode := tb.containerStack[lastIdx]
				// Check if last node is our element (by pointer comparison or name match)
				if lastNode == element {
					tb.containerStack = tb.containerStack[:lastIdx]
				} else if elem, ok := lastNode.(*Element); ok && elem.Name == fullName {
					// If pointer doesn't match but name does, it's still our element
					tb.containerStack = tb.containerStack[:lastIdx]
				}
			}
		}
	}
}

func (tb *TreeBuilder) _consumeAttributesAndDirectives(attributesResult *[]*Attribute, directivesResult *[]*Directive) {
	iterationCount := 0
	for tb.peek != nil && (tb.peek.Type() == TokenTypeATTR_NAME || tb.peek.Type() == TokenTypeDIRECTIVE_NAME) {
		iterationCount++
		fmt.Printf("[DEBUG] _consumeAttributesAndDirectives: iteration=%d, peek.Type()=%d, index=%d\n", iterationCount, func() int {
			if tb.peek == nil {
				return -1
			}
			return int(tb.peek.Type())
		}(), tb.index)
		if iterationCount > 100 {
			// Safety break for infinite loops
			break
		}
		if tb.peek.Type() == TokenTypeDIRECTIVE_NAME {
			fmt.Printf("[DEBUG] _consumeAttributesAndDirectives: consuming directive, peek.Type()=%d, index=%d\n", tb.peek.Type(), tb.index)
			directive := tb._consumeDirective(tb.peek)
			*directivesResult = append(*directivesResult, directive)
			fmt.Printf("[DEBUG] _consumeAttributesAndDirectives: after consuming directive, peek.Type()=%d, index=%d\n", func() int {
				if tb.peek == nil {
					return -1
				}
				return int(tb.peek.Type())
			}(), tb.index)
			// After consuming directive, continue loop to check for more attributes/directives
			// The loop condition will naturally check if peek is still ATTR_NAME or DIRECTIVE_NAME
		} else {
			attrNameToken := tb.advance().(*AttributeNameToken)
			peekBeforeConsume := tb.peek
			fmt.Printf("[DEBUG] _consumeAttributesAndDirectives: before consuming attr, peek.Type()=%d, index=%d\n", func() int {
				if peekBeforeConsume == nil {
					return -1
				}
				return int(peekBeforeConsume.Type())
			}(), tb.index)
			attr := tb._consumeAttr(attrNameToken)
			*attributesResult = append(*attributesResult, attr)
			fmt.Printf("[DEBUG] _consumeAttributesAndDirectives: after consuming attr, peek.Type()=%d, index=%d, peek==peekBeforeConsume=%v\n", func() int {
				if tb.peek == nil {
					return -1
				}
				return int(tb.peek.Type())
			}(), tb.index, tb.peek == peekBeforeConsume)
			// Safety check: if peek hasn't changed after consuming attr, it means we didn't consume any token
			// This can happen with attributes without values (e.g., [attr]).
			// If we're at the end of tokens, break to avoid infinite loop
			if tb.peek != nil && tb.peek == peekBeforeConsume {
				if tb.index >= len(tb.tokens)-1 {
					fmt.Printf("[DEBUG] _consumeAttributesAndDirectives: at end of tokens, breaking\n")
					break
				}
				// If peek is DIRECTIVE_NAME, don't force advance - let the loop naturally continue
				// to consume the directive in the next iteration
				if tb.peek.Type() == TokenTypeDIRECTIVE_NAME {
					// Don't force advance - the loop will naturally continue and consume the directive
					fmt.Printf("[DEBUG] _consumeAttributesAndDirectives: peek is DIRECTIVE_NAME, continuing loop\n")
					continue
				}
				// Only force advance if the next token is ATTR_NAME
				// Otherwise, the loop condition will naturally break
				if tb.peek.Type() == TokenTypeATTR_NAME {
					// Force advance to consume the token and avoid infinite loop
					fmt.Printf("[DEBUG] _consumeAttributesAndDirectives: forcing advance, peek.Type()=%d\n", tb.peek.Type())
					tb.advance()
				} else {
					// Next token is not ATTR_NAME or DIRECTIVE_NAME, so break the loop
					fmt.Printf("[DEBUG] _consumeAttributesAndDirectives: peek is not ATTR_NAME or DIRECTIVE_NAME (%d), breaking\n", tb.peek.Type())
					break
				}
			}
		}
	}
}

func (tb *TreeBuilder) _consumeAttr(attrName *AttributeNameToken) *Attribute {
	fullName := MergeNsAndName(attrName.Parts()[0], attrName.Parts()[1])
	attrEnd := attrName.SourceSpan().End

	// Consume any quote
	if tb.peek != nil && tb.peek.Type() == TokenTypeATTR_QUOTE {
		tb.advance()
	}

	// Consume the attribute value
	value := ""
	valueTokens := []InterpolatedAttributeToken{}
	var valueStartSpan *util.ParseSourceSpan
	var valueEnd *util.ParseLocation

	// Use a variable to check token type (similar to TypeScript)
	nextTokenType := TokenType(-1)
	if tb.peek != nil {
		nextTokenType = tb.peek.Type()
	}
	// In Go, we may get ATTR_VALUE (16) instead of ATTR_VALUE_TEXT (17) in some cases
	// Check for both ATTR_VALUE and ATTR_VALUE_TEXT
	if nextTokenType == TokenTypeATTR_VALUE_TEXT || nextTokenType == TokenTypeATTR_VALUE {
		valueStartSpan = tb.peek.SourceSpan()
		valueEnd = tb.peek.SourceSpan().End
		for tb.peek != nil && (tb.peek.Type() == TokenTypeATTR_VALUE_TEXT ||
			tb.peek.Type() == TokenTypeATTR_VALUE ||
			tb.peek.Type() == TokenTypeATTR_VALUE_INTERPOLATION ||
			tb.peek.Type() == TokenTypeENCODED_ENTITY) {
			valueToken := tb.advance()
			if interpToken, ok := valueToken.(InterpolatedAttributeToken); ok {
				valueTokens = append(valueTokens, interpToken)
			}
			if valueToken.Type() == TokenTypeATTR_VALUE_INTERPOLATION {
				// For backward compatibility, decode HTML entities in interpolation expressions
				parts := valueToken.Parts()
				valueStr := strings.Join(parts, "")
				// Decode entities using regex-like replacement
				valueStr = decodeEntityInString(valueStr)
				value += valueStr
			} else if valueToken.Type() == TokenTypeENCODED_ENTITY {
				parts := valueToken.Parts()
				if len(parts) > 0 {
					value += parts[0]
				}
			} else {
				parts := valueToken.Parts()
				value += strings.Join(parts, "")
			}
			valueEnd = valueToken.SourceSpan().End
			attrEnd = valueEnd
		}
	}

	// Consume any quote
	if tb.peek != nil && tb.peek.Type() == TokenTypeATTR_QUOTE {
		quoteToken := tb.advance()
		attrEnd = quoteToken.SourceSpan().End
	}

	var valueSpan *util.ParseSourceSpan
	if valueStartSpan != nil && valueEnd != nil {
		valueSpan = util.NewParseSourceSpan(valueStartSpan.Start, valueEnd, valueStartSpan.FullStart, nil)
	}

	return NewAttribute(
		fullName,
		value,
		util.NewParseSourceSpan(attrName.SourceSpan().Start, attrEnd, attrName.SourceSpan().FullStart, nil),
		attrName.SourceSpan(),
		valueSpan,
		valueTokens,
		nil,
	)
}

func (tb *TreeBuilder) _consumeDirective(nameToken Token) *Directive {
	attributes := []*Attribute{}
	var startSourceSpanEnd *util.ParseLocation
	var endSourceSpan *util.ParseSourceSpan

	// Get directive name from token parts
	directiveName := ""
	if parts := nameToken.Parts(); len(parts) > 0 {
		directiveName = parts[0]
	}
	startSourceSpanEnd = nameToken.SourceSpan().End
	tb.advance()

	if tb.peek != nil && tb.peek.Type() == TokenTypeDIRECTIVE_OPEN {
		// Capture the opening token in the start span
		startSourceSpanEnd = tb.peek.SourceSpan().End
		tb.advance()

		// Consume attributes
		for tb.peek != nil && tb.peek.Type() == TokenTypeATTR_NAME {
			attrNameToken := tb.advance().(*AttributeNameToken)
			attributes = append(attributes, tb._consumeAttr(attrNameToken))
		}

		if tb.peek != nil && tb.peek.Type() == TokenTypeDIRECTIVE_CLOSE {
			endSourceSpan = tb.peek.SourceSpan()
			tb.advance()
		} else {
			tb.errors = append(tb.errors, NewTreeError(
				nil,
				nameToken.SourceSpan(),
				"Unterminated directive definition",
			))
		}
	}

	startSourceSpan := util.NewParseSourceSpan(
		nameToken.SourceSpan().Start,
		startSourceSpanEnd,
		nameToken.SourceSpan().FullStart,
		nil,
	)

	var sourceSpanEnd *util.ParseLocation
	if endSourceSpan != nil {
		sourceSpanEnd = endSourceSpan.End
	} else {
		sourceSpanEnd = nameToken.SourceSpan().End
	}

	sourceSpan := util.NewParseSourceSpan(
		startSourceSpan.Start,
		sourceSpanEnd,
		startSourceSpan.FullStart,
		nil,
	)

	return NewDirective(
		directiveName,
		attributes,
		sourceSpan,
		startSourceSpan,
		endSourceSpan,
	)
}

// decodeEntityInString decodes HTML entities in a string (for backward compatibility)
func decodeEntityInString(str string) string {
	// Simple regex-like replacement for &entity; patterns
	// This is a simplified version - full implementation would use proper regex
	result := str
	// For now, just return as-is since entities are already decoded by the lexer
	// This function exists for compatibility with TypeScript code
	return result
}

func (tb *TreeBuilder) _consumeIncompleteStartTag(startTag *TagOpenStartToken) {
	parts := startTag.Parts()
	prefix := parts[0]
	name := parts[1]
	attrs := []*Attribute{}

	for tb.peek.Type() == TokenTypeATTR_NAME {
		attrNameToken := tb.advance().(*AttributeNameToken)
		attrParts := attrNameToken.Parts()
		attrPrefix := attrParts[0]
		attrName := attrParts[1]
		attrValue := ""
		var valueSpan *util.ParseSourceSpan
		var valueTokens []InterpolatedAttributeToken

		if tb.peek.Type() == TokenTypeATTR_VALUE_TEXT {
			valueToken := tb.advance().(*AttributeValueTextToken)
			attrValue = valueToken.Parts()[0]
			valueSpan = valueToken.SourceSpan()
		}

		fullName := attrName
		if attrPrefix != "" {
			fullName = attrPrefix + ":" + attrName
		}

		attrs = append(attrs, NewAttribute(
			fullName,
			attrValue,
			util.NewParseSourceSpan(attrNameToken.SourceSpan().Start, valueSpan.End, attrNameToken.SourceSpan().Start, nil),
			attrNameToken.SourceSpan(),
			valueSpan,
			valueTokens,
			nil,
		))
	}

	fullName := name
	if prefix != "" {
		fullName = prefix + ":" + name
	}

	// For incomplete tags, we don't have an end token, so use the start tag's span
	span := startTag.SourceSpan()
	element := NewElement(
		fullName,
		attrs,
		[]*Directive{},
		[]Node{},
		false, // isSelfClosing?
		span,
		span,
		nil,
		false,
		nil,
	)

	tb._addToParent(element)
	// Add error for incomplete tag
	tb.errors = append(tb.errors, NewTreeError(
		&fullName,
		span,
		fmt.Sprintf("Opening tag \"%s\" not terminated.", fullName),
	))
	// Don't add to container stack since it's incomplete
}

func (tb *TreeBuilder) _consumeEndTag(endTag *TagCloseToken) {
	parent := tb._getClosestElementLikeParent()

	// Check if this is a component closing tag (name starts with uppercase)
	parts := endTag.Parts()
	var tagName string
	if len(parts) > 1 {
		tagName = parts[1]
	}
	isComponent := len(tagName) > 0 && tagName[0] >= 'A' && tagName[0] <= 'Z'

	var fullName string
	if isComponent {
		// For components, use the component name directly
		fullName = tagName
	} else {
		// For elements, use _getElementFullName
		fullName = tb._getElementFullName(endTag, parent)
	}

	// Check if it's a void element (only for non-components)
	if !isComponent {
		tagDef := tb._getTagDefinition(fullName)
		if tagDef != nil && tagDef.IsVoid() {
			tb.errors = append(tb.errors, NewTreeError(
				&fullName,
				endTag.SourceSpan(),
				fmt.Sprintf("Void elements do not have end tags \"%s\"", endTag.Parts()[1]),
			))
			return
		}
	}

	// Use _popContainer to find and close the element/component (may close implicitly closed elements)
	if isComponent {
		var componentType *Component
		if !tb._popContainer(&fullName, componentType, endTag.SourceSpan()) {
			errMsg := fmt.Sprintf("Unexpected closing tag \"%s\". It may happen when the tag has already been closed by another tag. For more info see https://www.w3.org/TR/html5/syntax.html#closing-elements-that-have-implied-end-tags", fullName)
			tb.errors = append(tb.errors, NewTreeError(
				&fullName,
				endTag.SourceSpan(),
				errMsg,
			))
		}
	} else {
		var elementType *Element
		if !tb._popContainer(&fullName, elementType, endTag.SourceSpan()) {
			errMsg := fmt.Sprintf("Unexpected closing tag \"%s\". It may happen when the tag has already been closed by another tag. For more info see https://www.w3.org/TR/html5/syntax.html#closing-elements-that-have-implied-end-tags", fullName)
			tb.errors = append(tb.errors, NewTreeError(
				&fullName,
				endTag.SourceSpan(),
				errMsg,
			))
		}
	}
}

func (tb *TreeBuilder) _consumeCdata(_startToken Token) {
	// Match TypeScript: _consumeCdata calls _consumeText with the next token
	if tb.peek.Type() == TokenTypeRAW_TEXT || tb.peek.Type() == TokenTypeESCAPABLE_RAW_TEXT || tb.peek.Type() == TokenTypeTEXT {
		textToken := tb.advance()
		if textToken, ok := textToken.(*TextToken); ok {
			tb._consumeText(textToken)
		}
	}
	tb._advanceIf(TokenTypeCDATA_END)
}

func (tb *TreeBuilder) _consumeComment(token Token) {
	content := ""
	startSpan := token.SourceSpan()
	endSpan := startSpan.End

	if tb.peek.Type() == TokenTypeRAW_TEXT || tb.peek.Type() == TokenTypeESCAPABLE_RAW_TEXT {
		token := tb.advance()
		if textToken, ok := token.(*TextToken); ok {
			content = textToken.Parts()[0]
			endSpan = textToken.SourceSpan().End
		}
	}

	if tb.peek.Type() == TokenTypeCOMMENT_END {
		endToken := tb.advance()
		endSpan = endToken.SourceSpan().End
	}

	content = strings.TrimSpace(content)
	fullSpan := util.NewParseSourceSpan(startSpan.Start, endSpan, startSpan.FullStart, nil)

	tb._addToParent(NewComment(&content, fullSpan))
}

func (tb *TreeBuilder) _consumeText(token *TextToken) {
	tokens := []InterpolatedTextToken{token}
	startSpan := token.SourceSpan()
	text := token.Parts()[0]

	// Check if we should ignore the first LF (matches TypeScript logic)
	if len(text) > 0 && text[0] == '\n' {
		parent := tb._getContainer()
		if parent != nil {
			// Check if parent has no children yet
			var parentChildren []Node
			switch p := parent.(type) {
			case *Element:
				parentChildren = p.Children
			case *Component:
				parentChildren = p.Children
			}
			if len(parentChildren) == 0 {
				// Check if parent tag has ignoreFirstLf flag
				tagDef := tb._getTagDefinition(parent)
				if tagDef != nil && tagDef.IgnoreFirstLf() {
					// Remove the first '\n' character
					text = text[1:]
					// Update the token's parts (create new token with updated text)
					token = NewTextToken(text, token.Type(), token.SourceSpan())
					tokens[0] = token
				}
			}
		}
	}

	// Collect all consecutive TEXT, INTERPOLATION, and ENCODED_ENTITY tokens
	for tb.peek.Type() == TokenTypeTEXT ||
		tb.peek.Type() == TokenTypeINTERPOLATION ||
		tb.peek.Type() == TokenTypeENCODED_ENTITY {
		token := tb.advance()
		tokens = append(tokens, token.(InterpolatedTextToken))

		if token.Type() == TokenTypeENCODED_ENTITY {
			// For ENCODED_ENTITY, use the decoded value (parts[0])
			parts := token.Parts()
			if len(parts) > 0 {
				text += parts[0]
			}
		} else if token.Type() == TokenTypeINTERPOLATION {
			// For INTERPOLATION, join all parts (includes {{, expression, }})
			parts := token.Parts()
			if len(parts) > 0 {
				joined := strings.Join(parts, "")
				fmt.Printf("[DEBUG] _consumeText: INTERPOLATION token, parts=%v, joined=%q\n", parts, joined)
				text += joined
			}
		} else {
			// For TEXT, join all parts
			text += strings.Join(token.Parts(), "")
		}
	}

	if len(text) > 0 {
		endSpan := tokens[len(tokens)-1].SourceSpan()
		fullSpan := util.NewParseSourceSpan(startSpan.Start, endSpan.End, startSpan.FullStart, startSpan.Details)
		tb._addToParent(NewText(text, fullSpan, tokens, nil))
	}
}

func (tb *TreeBuilder) _consumeDocType(token Token) {
	content := token.Parts()[0]
	tb._addToParent(NewComment(&content, token.SourceSpan()))
}

func (tb *TreeBuilder) _closeVoidElement() {
	el := tb._getContainer()
	if el != nil {
		tagDef := tb._getTagDefinition(el)
		if tagDef != nil && tagDef.IsVoid() {
			tb.containerStack = tb.containerStack[:len(tb.containerStack)-1]
		}
	}
}

func (tb *TreeBuilder) _advanceIf(tokenType TokenType) Token {
	if tb.peek != nil && tb.peek.Type() == tokenType {
		return tb.advance()
	}
	return nil
}

func (tb *TreeBuilder) _getContainer() NodeContainer {
	if len(tb.containerStack) > 0 {
		return tb.containerStack[len(tb.containerStack)-1]
	}
	return nil
}

func (tb *TreeBuilder) _getClosestElementLikeParent() Node {
	for i := len(tb.containerStack) - 1; i >= 0; i-- {
		current := tb.containerStack[i]
		if _, ok := current.(*Element); ok {
			return current
		}
		if _, ok := current.(*Component); ok {
			return current
		}
	}
	return nil
}

func (tb *TreeBuilder) _getTagDefinition(nodeOrName interface{}) TagDefinition {
	if tb.tagDefinitionResolver == nil {
		return nil
	}
	switch v := nodeOrName.(type) {
	case string:
		return tb.tagDefinitionResolver(v)
	case *Element:
		return tb.tagDefinitionResolver(v.Name)
	case *Component:
		if v.TagName != nil {
			return tb.tagDefinitionResolver(*v.TagName)
		}
		return nil
	default:
		return nil
	}
}

func (tb *TreeBuilder) _addToParent(node Node) {
	parent := tb._getContainer()
	if parent == nil {
		tb.rootNodes = append(tb.rootNodes, node)
	} else {
		// Add to parent's children
		switch p := parent.(type) {
		case *Element:
			p.Children = append(p.Children, node)
		case *Block:
			p.Children = append(p.Children, node)
		case *Component:
			p.Children = append(p.Children, node)
		}
	}
}

func (tb *TreeBuilder) _getElementFullName(token interface{}, parent Node) string {
	prefix := tb._getPrefix(token, parent)
	var name string
	switch t := token.(type) {
	case *TagOpenStartToken:
		parts := t.Parts()
		if len(parts) > 1 {
			name = parts[1]
		}
	case *TagCloseToken:
		parts := t.Parts()
		if len(parts) > 1 {
			name = parts[1]
		}
	}
	return MergeNsAndName(prefix, name)
}

func (tb *TreeBuilder) _getComponentFullName(token interface{}, parent Node) string {
	var componentName string
	if tok, ok := token.(Token); ok {
		parts := tok.Parts()
		if len(parts) > 0 {
			componentName = parts[0]
		}
	}
	tagName := tb._getComponentTagName(token, parent)
	if tagName == nil {
		return componentName
	}
	if strings.HasPrefix(*tagName, ":") {
		return componentName + *tagName
	}
	return componentName + ":" + *tagName
}

func (tb *TreeBuilder) _getComponentTagName(token interface{}, parent Node) *string {
	prefix := tb._getPrefix(token, parent)
	var tagName string
	if tok, ok := token.(Token); ok {
		parts := tok.Parts()
		if len(parts) > 2 {
			tagName = parts[2]
		}
	}
	if prefix == "" && tagName == "" {
		return nil
	} else if prefix == "" && tagName != "" {
		return &tagName
	} else {
		// Merge namespace and tag name
		// If tagName is empty but prefix is not, use 'ng-component' as fallback (matches TypeScript)
		if tagName == "" {
			tagName = "ng-component"
		}
		merged := MergeNsAndName(prefix, tagName)
		return &merged
	}
}

func (tb *TreeBuilder) _getPrefix(token interface{}, parent Node) string {
	var prefix, tagName string
	// Get token parts - works with any Token interface
	var parts []string
	if tok, ok := token.(Token); ok {
		parts = tok.Parts()
		tokenType := tok.Type()
		if tokenType == TokenTypeCOMPONENT_OPEN_START || tokenType == TokenTypeCOMPONENT_CLOSE || tokenType == TokenTypeINCOMPLETE_COMPONENT_OPEN {
			// Component tokens have parts: [componentName, prefix, tagName]
			if len(parts) > 1 {
				prefix = parts[1]
			}
			if len(parts) > 2 {
				tagName = parts[2]
			}
		} else {
			// Tag tokens have parts: [prefix, name]
			if len(parts) > 0 {
				prefix = parts[0]
			}
			if len(parts) > 1 {
				tagName = parts[1]
			}
		}
	}
	if prefix == "" {
		tagDef := tb._getTagDefinition(tagName)
		if tagDef != nil && tagDef.ImplicitNamespacePrefix() != nil {
			prefix = *tagDef.ImplicitNamespacePrefix()
		}
	}
	if prefix == "" && parent != nil {
		var parentName *string
		switch p := parent.(type) {
		case *Element:
			parentName = &p.Name
		case *Component:
			if p.TagName != nil {
				parentName = p.TagName
			}
		}
		if parentName != nil {
			_, localName := SplitNsName(*parentName, false)
			parentTagDef := tb._getTagDefinition(localName)
			if parentTagDef != nil && !parentTagDef.PreventNamespaceInheritance() {
				if prefixPtr := GetNsPrefix(parentName); prefixPtr != nil {
					prefix = *prefixPtr
				}
			}
		}
	}
	return prefix
}

func (tb *TreeBuilder) _pushContainer(node NodeContainer, isClosedByChild bool) {
	if isClosedByChild {
		if len(tb.containerStack) > 0 {
			tb.containerStack = tb.containerStack[:len(tb.containerStack)-1]
		}
	}
	tb._addToParent(node)
	tb.containerStack = append(tb.containerStack, node)
}

func (tb *TreeBuilder) _popContainer(expectedName *string, expectedType interface{}, endSourceSpan *util.ParseSourceSpan) bool {
	unexpectedCloseTagDetected := false
	for stackIndex := len(tb.containerStack) - 1; stackIndex >= 0; stackIndex-- {
		node := tb.containerStack[stackIndex]
		var nodeName string
		switch n := node.(type) {
		case *Component:
			nodeName = n.FullName
		case *Element:
			nodeName = n.Name
		case *Block:
			nodeName = n.Name
		}
		nameMatch := expectedName == nil || nodeName == *expectedName

		// Type matching: expectedType can be (*Element)(nil), (*Component)(nil), or (*Block)(nil)
		// In TypeScript: node instanceof expectedType
		// In Go: we check if node is of the same type as expectedType
		// Note: (*Element)(nil) is not the same as nil interface{}, so we need to check type differently
		typeMatch := false
		// Use type switch on expectedType to determine what type we're looking for
		switch expectedType.(type) {
		case *Element:
			// We're looking for *Element
			_, ok := node.(*Element)
			typeMatch = ok
		case *Component:
			// We're looking for *Component
			_, ok := node.(*Component)
			typeMatch = ok
		case *Block:
			// We're looking for *Block
			_, ok := node.(*Block)
			typeMatch = ok
		default:
			// expectedType is nil interface{} - match any type
			typeMatch = true
		}
		if nameMatch && typeMatch {
			// Record the parse span with the element that is being closed.
			// Any elements that are removed from the element stack at this point
			// are closed implicitly, so they won't get an end source span.
			switch n := node.(type) {
			case *Element:
				n.EndSourceSpan = endSourceSpan
				if endSourceSpan != nil {
					n.sourceSpan = util.NewParseSourceSpan(n.sourceSpan.Start, endSourceSpan.End, n.sourceSpan.FullStart, n.sourceSpan.Details)
				} else {
					// Update end even if endSourceSpan is nil
					n.sourceSpan = util.NewParseSourceSpan(n.sourceSpan.Start, n.sourceSpan.End, n.sourceSpan.FullStart, n.sourceSpan.Details)
				}
			case *Component:
				n.EndSourceSpan = endSourceSpan
				if endSourceSpan != nil {
					n.sourceSpan = util.NewParseSourceSpan(n.sourceSpan.Start, endSourceSpan.End, n.sourceSpan.FullStart, n.sourceSpan.Details)
				} else {
					n.sourceSpan = util.NewParseSourceSpan(n.sourceSpan.Start, n.sourceSpan.End, n.sourceSpan.FullStart, n.sourceSpan.Details)
				}
			case *Block:
				n.EndSourceSpan = endSourceSpan
				if endSourceSpan != nil {
					n.sourceSpan = util.NewParseSourceSpan(n.sourceSpan.Start, endSourceSpan.End, n.sourceSpan.FullStart, n.sourceSpan.Details)
				} else {
					n.sourceSpan = util.NewParseSourceSpan(n.sourceSpan.Start, n.sourceSpan.End, n.sourceSpan.FullStart, n.sourceSpan.Details)
				}
			}
			// Remove from stack (all elements from stackIndex to end are closed implicitly)
			// Match TypeScript: this._containerStack.splice(stackIndex, this._containerStack.length - stackIndex)
			tb.containerStack = tb.containerStack[:stackIndex]
			return !unexpectedCloseTagDetected
		}

		// Blocks and most elements are not self closing.
		// Note that we encountered an unexpected close tag but continue processing
		// the element stack so we can assign an `endSourceSpan` if there is a
		// corresponding start tag for this end tag in the stack.
		if _, isBlock := node.(*Block); isBlock {
			unexpectedCloseTagDetected = true
		} else {
			tagDef := tb._getTagDefinition(node)
			if tagDef == nil || !tagDef.ClosedByParent() {
				unexpectedCloseTagDetected = true
			}
		}
	}
	return false
}

func (tb *TreeBuilder) advance() Token {
	current := tb.peek
	if tb.index < len(tb.tokens)-1 {
		tb.index++
		tb.peek = tb.tokens[tb.index]
	} else {
		// We've reached the end of tokens, set peek to EOF
		if current != nil {
			tb.peek = NewEndOfFileToken(current.SourceSpan())
		} else {
			tb.peek = NewEndOfFileToken(nil)
		}
		// Increment index to mark that we've advanced past all tokens
		if tb.index < len(tb.tokens) {
			tb.index++
		}
	}
	return current
}

func (tb *TreeBuilder) _consumeExpansion(token Token) {
	// Advance to get switchValue (should be a TEXT token)
	switchValueToken := tb.advance()
	switchValue := ""
	if parts := switchValueToken.Parts(); len(parts) > 0 {
		switchValue = parts[0]
	}

	// Advance to get type (should be a TEXT token)
	typeToken := tb.advance()
	typ := ""
	if parts := typeToken.Parts(); len(parts) > 0 {
		typ = parts[0]
	}

	cases := []*ExpansionCase{}
	hasErrors := false

	// Read expansion cases
	for tb.peek != nil {
		if tb.peek.Type() == TokenTypeEXPANSION_CASE_VALUE {
			expCase := tb._parseExpansionCase()
			if expCase == nil {
				// Error occurred in expansion case - don't report ICU message error
				// The error from expansion case should be reported instead
				hasErrors = true
				return
			}
			cases = append(cases, expCase)
		} else if tb.peek.Type() == TokenTypeEXPANSION_FORM_END {
			break
		} else if tb.peek.Type() == TokenTypeBLOCK_CLOSE {
			// BLOCK_CLOSE might be tokenized instead of EXPANSION_FORM_END
			// Check if we're at the end of the expansion form
			break
		} else if tb.peek.Type() == TokenTypeTEXT || tb.peek.Type() == TokenTypeRAW_TEXT {
			// Skip whitespace
			tb.advance()
		} else {
			// Unexpected token
			break
		}
	}

	// Read the final } - only check if we don't have errors from expansion cases
	// Accept both EXPANSION_FORM_END and BLOCK_CLOSE as valid closing tokens
	if !hasErrors && tb.peek != nil {
		if tb.peek.Type() == TokenTypeEXPANSION_FORM_END {
			// Valid closing token
		} else if tb.peek.Type() == TokenTypeBLOCK_CLOSE {
			// BLOCK_CLOSE was tokenized instead of EXPANSION_FORM_END
			// This is acceptable - treat it as EXPANSION_FORM_END
		} else {
			tb.errors = append(tb.errors, NewTreeError(
				nil,
				tb.peek.SourceSpan(),
				"Invalid ICU message. Missing '}'.",
			))
			// Consume the unexpected token to avoid re-processing it in the main loop
			if tb.peek.Type() != TokenTypeEOF {
				tb.advance()
			}
			return
		}
	} else if !hasErrors {
		tb.errors = append(tb.errors, NewTreeError(
			nil,
			token.SourceSpan(),
			"Invalid ICU message. Missing '}'.",
		))
		return
	}

	sourceSpan := util.NewParseSourceSpan(
		token.SourceSpan().Start,
		tb.peek.SourceSpan().End,
		token.SourceSpan().FullStart,
		nil,
	)

	var switchValueSourceSpan *util.ParseSourceSpan
	if textToken, ok := switchValueToken.(*TextToken); ok {
		switchValueSourceSpan = textToken.SourceSpan()
	} else {
		switchValueSourceSpan = switchValueToken.SourceSpan()
	}

	tb._addToParent(NewExpansion(
		switchValue,
		typ,
		cases,
		sourceSpan,
		switchValueSourceSpan,
		nil,
	))

	tb.advance()
}

func (tb *TreeBuilder) _parseExpansionCase() *ExpansionCase {
	fmt.Printf("[DEBUG] _parseExpansionCase: START, index=%d, peek.Type()=%d\n", tb.index, func() int {
		if tb.peek == nil {
			return -1
		}
		return int(tb.peek.Type())
	}())
	valueToken := tb.advance()
	value := ""
	if parts := valueToken.Parts(); len(parts) > 0 {
		value = parts[0]
	}
	fmt.Printf("[DEBUG] _parseExpansionCase: value=%s, index=%d\n", value, tb.index)

	// Read {
	if tb.peek == nil || tb.peek.Type() != TokenTypeEXPANSION_CASE_EXP_START {
		if tb.peek != nil {
			fmt.Printf("[DEBUG] _parseExpansionCase: ERROR - expected EXPANSION_CASE_EXP_START but got type=%d\n", tb.peek.Type())
			tb.errors = append(tb.errors, NewTreeError(
				nil,
				tb.peek.SourceSpan(),
				"Invalid ICU message. Missing '{'.",
			))
		} else {
			fmt.Printf("[DEBUG] _parseExpansionCase: ERROR - peek is nil\n")
		}
		return nil
	}

	startToken := tb.advance()
	fmt.Printf("[DEBUG] _parseExpansionCase: got EXPANSION_CASE_EXP_START, calling _collectExpansionExpTokens, index=%d\n", tb.index)

	exp := tb._collectExpansionExpTokens(startToken)
	if exp == nil {
		fmt.Printf("[DEBUG] _parseExpansionCase: _collectExpansionExpTokens returned nil\n")
		return nil
	}
	fmt.Printf("[DEBUG] _parseExpansionCase: _collectExpansionExpTokens returned %d tokens, index=%d\n", len(exp), tb.index)

	// Get the end token - _collectExpansionExpTokens has already advanced past the closing token
	// Check what token we're at now
	var endToken Token
	if tb.peek != nil {
		// If the next token is BLOCK_CLOSE, it was the closing token that _collectExpansionExpTokens advanced past
		if tb.peek.Type() == TokenTypeBLOCK_CLOSE {
			fmt.Printf("[DEBUG] _parseExpansionCase: next token is BLOCK_CLOSE (was advanced past by _collectExpansionExpTokens), treating as EXPANSION_CASE_EXP_END\n")
			endToken = tb.advance()
			// Create a synthetic EXPANSION_CASE_EXP_END token
			endToken = NewTokenBase(TokenTypeEXPANSION_CASE_EXP_END, []string{}, endToken.SourceSpan())
		} else if tb.peek.Type() == TokenTypeEXPANSION_CASE_EXP_END {
			endToken = tb.advance()
		} else {
			// Unexpected token - might be TEXT or something else
			// This shouldn't happen, but handle it gracefully
			fmt.Printf("[DEBUG] _parseExpansionCase: unexpected token after _collectExpansionExpTokens, type=%d, treating as end\n", tb.peek.Type())
			endToken = tb.advance()
			// Create a synthetic EXPANSION_CASE_EXP_END token
			endToken = NewTokenBase(TokenTypeEXPANSION_CASE_EXP_END, []string{}, endToken.SourceSpan())
		}
	} else {
		// No more tokens - create a synthetic end token
		fmt.Printf("[DEBUG] _parseExpansionCase: no more tokens, creating synthetic end token\n")
		if len(exp) > 0 {
			lastToken := exp[len(exp)-1]
			endToken = NewTokenBase(TokenTypeEXPANSION_CASE_EXP_END, []string{}, lastToken.SourceSpan())
		} else {
			endToken = NewTokenBase(TokenTypeEXPANSION_CASE_EXP_END, []string{}, startToken.SourceSpan())
		}
	}
	fmt.Printf("[DEBUG] _parseExpansionCase: got endToken, type=%d, index=%d\n", endToken.Type(), tb.index)
	// Add EOF token to the end
	eofToken := NewTokenBase(TokenTypeEOF, []string{}, endToken.SourceSpan())
	exp = append(exp, eofToken)

	// Filter out any BLOCK_CLOSE tokens from exp - they shouldn't be there
	// but if they are (due to lexer issues), remove them
	filteredExp := []Token{}
	for _, token := range exp {
		if token.Type() != TokenTypeBLOCK_CLOSE {
			filteredExp = append(filteredExp, token)
		} else {
			fmt.Printf("[DEBUG] _parseExpansionCase: filtering out BLOCK_CLOSE token from exp\n")
		}
	}

	// Parse everything in between { and }
	fmt.Printf("[DEBUG] _parseExpansionCase: creating expansionCaseParser with %d tokens\n", len(filteredExp))
	for i, token := range filteredExp {
		if i < 5 {
			fmt.Printf("[DEBUG] _parseExpansionCase: filteredExp[%d]: type=%d, parts=%v\n", i, token.Type(), token.Parts())
		}
	}
	expansionCaseParser := NewTreeBuilder(filteredExp, tb.tagDefinitionResolver)
	expansionCaseParser.inExpansionContext = true // Mark that we're parsing expansion case expression
	expansionCaseParser.Build()
	if len(expansionCaseParser.errors) > 0 {
		fmt.Printf("[DEBUG] _parseExpansionCase: expansionCaseParser has %d errors\n", len(expansionCaseParser.errors))
		for i, err := range expansionCaseParser.errors {
			fmt.Printf("[DEBUG] _parseExpansionCase: error[%d]=%q, span=%v\n", i, err.String(), err.ParseError.Span)
		}
		// In TypeScript, errors from expansionCaseParser are directly appended
		// But we need to check for duplicates to avoid reporting the same error twice
		// The error from _collectExpansionExpTokens might have the same span as the error from expansionCaseParser
		for _, expErr := range expansionCaseParser.errors {
			isDuplicate := false
			for _, existingErr := range tb.errors {
				// Check if error message and span match
				if existingErr.ParseError.Msg == expErr.ParseError.Msg &&
					existingErr.ParseError.Span != nil && expErr.ParseError.Span != nil &&
					existingErr.ParseError.Span.Start.Offset == expErr.ParseError.Span.Start.Offset &&
					existingErr.ParseError.Span.Start.Line == expErr.ParseError.Span.Start.Line &&
					existingErr.ParseError.Span.Start.Col == expErr.ParseError.Span.Start.Col {
					isDuplicate = true
					fmt.Printf("[DEBUG] _parseExpansionCase: skipping duplicate error at offset %d\n", expErr.ParseError.Span.Start.Offset)
					break
				}
			}
			if !isDuplicate {
				tb.errors = append(tb.errors, expErr)
			}
		}
		return nil
	}
	fmt.Printf("[DEBUG] _parseExpansionCase: expansionCaseParser built successfully, rootNodes=%d\n", len(expansionCaseParser.rootNodes))

	sourceSpan := util.NewParseSourceSpan(
		valueToken.SourceSpan().Start,
		endToken.SourceSpan().End,
		valueToken.SourceSpan().FullStart,
		nil,
	)

	var valueSourceSpan *util.ParseSourceSpan
	if valueTokenParts := valueToken.Parts(); len(valueTokenParts) > 0 {
		valueSourceSpan = valueToken.SourceSpan()
	} else {
		valueSourceSpan = valueToken.SourceSpan()
	}

	expSourceSpan := util.NewParseSourceSpan(
		startToken.SourceSpan().Start,
		endToken.SourceSpan().End,
		startToken.SourceSpan().FullStart,
		nil,
	)

	return NewExpansionCase(
		value,
		expansionCaseParser.rootNodes,
		sourceSpan,
		valueSourceSpan,
		expSourceSpan,
	)
}

func (tb *TreeBuilder) _collectExpansionExpTokens(start Token) []Token {
	fmt.Printf("[DEBUG] _collectExpansionExpTokens: START, index=%d, totalTokens=%d\n", tb.index, len(tb.tokens))
	exp := []Token{}
	// Stack starts with EXPANSION_CASE_EXP_START for the outer case we're collecting
	// We need to track both expansion forms and expansion cases to know when we're done
	// The stack tracks all open expansion constructs (forms and cases) that we encounter
	expansionFormStack := []TokenType{TokenTypeEXPANSION_CASE_EXP_START}
	iterationCount := 0

	for {
		iterationCount++
		if iterationCount > 1000 {
			fmt.Printf("[DEBUG] _collectExpansionExpTokens: INFINITE LOOP DETECTED! iterationCount=%d, stackSize=%d, peek.Type()=%d, index=%d\n",
				iterationCount, len(expansionFormStack), func() int {
					if tb.peek == nil {
						return -1
					}
					return int(tb.peek.Type())
				}(), tb.index)
			tb.errors = append(tb.errors, NewTreeError(
				nil,
				start.SourceSpan(),
				"Invalid ICU message. Infinite loop detected.",
			))
			return nil
		}

		// Check if we've reached the end of tokens or EOF
		if tb.peek == nil || tb.index >= len(tb.tokens) {
			if tb.peek == nil {
				fmt.Printf("[DEBUG] _collectExpansionExpTokens: peek is nil, returning error at start.SourceSpan()=%v\n", start.SourceSpan())
			} else {
				fmt.Printf("[DEBUG] _collectExpansionExpTokens: reached end of tokens, index=%d, totalTokens=%d, returning error at start.SourceSpan()=%v\n", tb.index, len(tb.tokens), start.SourceSpan())
			}
			err := NewTreeError(
				nil,
				start.SourceSpan(),
				"Invalid ICU message. Missing '}'.",
			)
			fmt.Printf("[DEBUG] _collectExpansionExpTokens: creating error with span=%v\n", err.ParseError.Span)
			tb.errors = append(tb.errors, err)
			return nil
		}

		peekType := tb.peek.Type()
		if iterationCount <= 10 || iterationCount%50 == 0 {
			fmt.Printf("[DEBUG] _collectExpansionExpTokens: iteration=%d, peek.Type()=%d, stackSize=%d, index=%d\n",
				iterationCount, peekType, len(expansionFormStack), tb.index)
		}

		// Check for EOF early to avoid duplicate error (after checking peek != nil)
		if peekType == TokenTypeEOF {
			fmt.Printf("[DEBUG] _collectExpansionExpTokens: ERROR - reached EOF at start.SourceSpan()=%v\n", start.SourceSpan())
			err := NewTreeError(
				nil,
				start.SourceSpan(),
				"Invalid ICU message. Missing '}'.",
			)
			fmt.Printf("[DEBUG] _collectExpansionExpTokens: creating EOF error with span=%v\n", err.ParseError.Span)
			tb.errors = append(tb.errors, err)
			return nil
		}

		// Handle BLOCK_CLOSE tokens FIRST - they may have been tokenized
		// as BLOCK_CLOSE instead of EXPANSION_FORM_END or EXPANSION_CASE_EXP_END due to lexer ordering
		// Don't collect them into exp - they're delimiters, not content
		if peekType == TokenTypeBLOCK_CLOSE {
			if lastOnStack(expansionFormStack, TokenTypeEXPANSION_FORM_START) {
				// This BLOCK_CLOSE is actually closing a nested expansion form
				fmt.Printf("[DEBUG] _collectExpansionExpTokens: treating BLOCK_CLOSE as EXPANSION_FORM_END (nested), index=%d\n", tb.index)
				expansionFormStack = expansionFormStack[:len(expansionFormStack)-1]
				// Don't return - this was a nested expansion form, continue collecting
				// But don't collect the BLOCK_CLOSE token itself
				tb.advance()
				continue
			} else if lastOnStack(expansionFormStack, TokenTypeEXPANSION_CASE_EXP_START) {
				// This BLOCK_CLOSE is actually closing the outer expansion case
				// This is the end of the expression we're collecting
				fmt.Printf("[DEBUG] _collectExpansionExpTokens: treating BLOCK_CLOSE as outer case end, index=%d\n", tb.index)
				expansionFormStack = expansionFormStack[:len(expansionFormStack)-1]
				if len(expansionFormStack) == 0 {
					fmt.Printf("[DEBUG] _collectExpansionExpTokens: stack empty after BLOCK_CLOSE, returning %d tokens\n", len(exp))
					// Advance past the BLOCK_CLOSE token before returning
					// The caller will expect the next token to be the closing token
					tb.advance()
					return exp
				}
				// Continue if there are more nested forms
				tb.advance()
				continue
			} else if len(expansionFormStack) == 0 {
				// Stack is empty and we encounter BLOCK_CLOSE - this is the closing } of the outer expansion case
				// This is the end of the expression we're collecting
				fmt.Printf("[DEBUG] _collectExpansionExpTokens: stack empty, BLOCK_CLOSE is outer case end, returning %d tokens\n", len(exp))
				// Don't advance - let _parseExpansionCase handle the BLOCK_CLOSE token
				// This way _parseExpansionCase can see it and treat it as EXPANSION_CASE_EXP_END
				return exp
			} else {
				// Unexpected BLOCK_CLOSE - this might be the closing } of the expansion case
				// Don't collect it, just skip and let the caller handle it
				fmt.Printf("[DEBUG] _collectExpansionExpTokens: unexpected BLOCK_CLOSE, might be case end, skipping, index=%d\n", tb.index)
				tb.advance()
				continue
			}
		}

		// Only push expansion tokens to stack
		isExpansionStart := peekType == TokenTypeEXPANSION_FORM_START || peekType == TokenTypeEXPANSION_CASE_EXP_START
		if isExpansionStart {
			expansionFormStack = append(expansionFormStack, peekType)
			if iterationCount <= 10 {
				fmt.Printf("[DEBUG] _collectExpansionExpTokens: pushed to stack, peekType=%d (EXPANSION_FORM_START=%d, EXPANSION_CASE_EXP_START=%d), new stackSize=%d\n",
					peekType, TokenTypeEXPANSION_FORM_START, TokenTypeEXPANSION_CASE_EXP_START, len(expansionFormStack))
			}
		} else if iterationCount <= 10 && (peekType == TokenTypeCOMMENT_END || peekType == TokenTypeCDATA_END) {
			// Debug: show when we encounter COMMENT_END or CDATA_END (which should NOT be pushed)
			fmt.Printf("[DEBUG] _collectExpansionExpTokens: NOT pushing to stack, peekType=%d (COMMENT_END=%d, CDATA_END=%d, EXPANSION_FORM_START=%d, EXPANSION_CASE_EXP_START=%d)\n",
				peekType, TokenTypeCOMMENT_END, TokenTypeCDATA_END, TokenTypeEXPANSION_FORM_START, TokenTypeEXPANSION_CASE_EXP_START)
		}

		if peekType == TokenTypeEXPANSION_CASE_EXP_END {
			if lastOnStack(expansionFormStack, TokenTypeEXPANSION_CASE_EXP_START) {
				expansionFormStack = expansionFormStack[:len(expansionFormStack)-1]
				if iterationCount <= 10 {
					fmt.Printf("[DEBUG] _collectExpansionExpTokens: popped EXPANSION_CASE_EXP_START from stack, new stackSize=%d\n", len(expansionFormStack))
				}
				// If stack is empty, this is the outer case end - return without collecting
				if len(expansionFormStack) == 0 {
					fmt.Printf("[DEBUG] _collectExpansionExpTokens: stack empty (outer case closed), returning %d tokens\n", len(exp))
					// Don't advance - return immediately without collecting the outer EXPANSION_CASE_EXP_END
					return exp
				}
				// Nested expansion case - collect the token and continue
				fmt.Printf("[DEBUG] _collectExpansionExpTokens: nested expansion case closed, collecting token, stackSize=%d\n", len(expansionFormStack))
				// Fall through to collect the token
			} else {
				fmt.Printf("[DEBUG] _collectExpansionExpTokens: ERROR - stack mismatch for EXPANSION_CASE_EXP_END, stack=%v\n", expansionFormStack)
				tb.errors = append(tb.errors, NewTreeError(
					nil,
					start.SourceSpan(),
					"Invalid ICU message. Missing '}'.",
				))
				return nil
			}
		}

		if peekType == TokenTypeEXPANSION_FORM_END {
			if lastOnStack(expansionFormStack, TokenTypeEXPANSION_FORM_START) {
				expansionFormStack = expansionFormStack[:len(expansionFormStack)-1]
				if iterationCount <= 10 {
					fmt.Printf("[DEBUG] _collectExpansionExpTokens: popped EXPANSION_FORM_START from stack, new stackSize=%d\n", len(expansionFormStack))
				}
				// Collect the token and continue (matches TypeScript behavior)
				// Fall through to collect the token
			} else {
				fmt.Printf("[DEBUG] _collectExpansionExpTokens: ERROR - stack mismatch for EXPANSION_FORM_END\n")
				tb.errors = append(tb.errors, NewTreeError(
					nil,
					start.SourceSpan(),
					"Invalid ICU message. Missing '}'.",
				))
				return nil
			}
		}

		// Advance to next token
		oldIndex := tb.index
		currentToken := tb.advance()
		exp = append(exp, currentToken)
		if iterationCount <= 10 {
			fmt.Printf("[DEBUG] _collectExpansionExpTokens: collected token, type=%d, parts=%v, expLen=%d\n",
				currentToken.Type(), currentToken.Parts(), len(exp))
		}

		// Check if we actually advanced (prevent infinite loop)
		if iterationCount <= 10 {
			fmt.Printf("[DEBUG] _collectExpansionExpTokens: after advance, oldIndex=%d, newIndex=%d, totalTokens=%d, peek.Type()=%d\n",
				oldIndex, tb.index, len(tb.tokens), func() int {
					if tb.peek == nil {
						return -1
					}
					return int(tb.peek.Type())
				}())
		}

		// If we didn't advance (stuck at the same index), we've reached the end
		if tb.index == oldIndex && tb.index >= len(tb.tokens)-1 {
			fmt.Printf("[DEBUG] _collectExpansionExpTokens: cannot advance further, index=%d, totalTokens=%d, returning error at start.SourceSpan()=%v\n", tb.index, len(tb.tokens), start.SourceSpan())
			// Check if peek is EOF - if so, we already handled it above
			if tb.peek != nil && tb.peek.Type() == TokenTypeEOF {
				// Already handled EOF case above, just return
				fmt.Printf("[DEBUG] _collectExpansionExpTokens: cannot advance further, EOF already handled, returning\n")
				return nil
			}
			err := NewTreeError(
				nil,
				start.SourceSpan(),
				"Invalid ICU message. Missing '}'.",
			)
			fmt.Printf("[DEBUG] _collectExpansionExpTokens: creating stuck error with span=%v\n", err.ParseError.Span)
			tb.errors = append(tb.errors, err)
			return nil
		}
	}
}

// _consumeBlockOpen consumes a block open token
func (tb *TreeBuilder) _consumeBlockOpen(token Token) {
	parts := token.Parts()
	blockName := ""
	if len(parts) > 0 {
		blockName = parts[0]
	}

	parameters := []*BlockParameter{}

	// Consume all block parameters
	for tb.peek != nil && tb.peek.Type() == TokenTypeBLOCK_PARAMETER {
		paramToken := tb.advance()
		paramParts := paramToken.Parts()
		paramExpr := ""
		if len(paramParts) > 0 {
			paramExpr = paramParts[0]
		}
		parameters = append(parameters, NewBlockParameter(paramExpr, paramToken.SourceSpan()))
	}

	// Consume BLOCK_OPEN_END if present
	if tb.peek != nil && tb.peek.Type() == TokenTypeBLOCK_OPEN_END {
		tb.advance()
	}

	// Create source spans
	var end *util.ParseLocation
	if tb.peek != nil {
		end = tb.peek.SourceSpan().FullStart
	} else {
		end = token.SourceSpan().End
	}
	span := util.NewParseSourceSpan(token.SourceSpan().Start, end, token.SourceSpan().FullStart, nil)
	startSpan := util.NewParseSourceSpan(token.SourceSpan().Start, end, token.SourceSpan().FullStart, nil)

	// Create block node
	block := NewBlock(blockName, parameters, []Node{}, span, token.SourceSpan(), startSpan, nil, nil)
	tb._pushContainer(block, false)
}

// _consumeBlockClose consumes a block close token
func (tb *TreeBuilder) _consumeBlockClose(token Token) {
	if !tb._popContainer(nil, (*Block)(nil), token.SourceSpan()) {
		tb.errors = append(tb.errors, NewTreeError(
			nil,
			token.SourceSpan(),
			"Unexpected closing block. The block may have been closed earlier. If you meant to write the } character, you should use the \"&#125;\" HTML entity instead.",
		))
	}
}

// _consumeIncompleteBlock consumes an incomplete block open token
func (tb *TreeBuilder) _consumeIncompleteBlock(token Token) {
	parts := token.Parts()
	blockName := ""
	if len(parts) > 0 {
		blockName = parts[0]
	}

	parameters := []*BlockParameter{}

	// Consume all block parameters
	for tb.peek != nil && tb.peek.Type() == TokenTypeBLOCK_PARAMETER {
		paramToken := tb.advance()
		paramParts := paramToken.Parts()
		paramExpr := ""
		if len(paramParts) > 0 {
			paramExpr = paramParts[0]
		}
		parameters = append(parameters, NewBlockParameter(paramExpr, paramToken.SourceSpan()))
	}

	// Create source spans
	var end *util.ParseLocation
	if tb.peek != nil {
		end = tb.peek.SourceSpan().FullStart
	} else {
		end = token.SourceSpan().End
	}
	span := util.NewParseSourceSpan(token.SourceSpan().Start, end, token.SourceSpan().FullStart, nil)
	startSpan := util.NewParseSourceSpan(token.SourceSpan().Start, end, token.SourceSpan().FullStart, nil)

	// Create block node
	block := NewBlock(blockName, parameters, []Node{}, span, token.SourceSpan(), startSpan, nil, nil)
	tb._pushContainer(block, false)

	// Incomplete blocks don't have children so we close them immediately and report an error.
	tb._popContainer(nil, (*Block)(nil), nil)

	tb.errors = append(tb.errors, NewTreeError(
		&blockName,
		token.SourceSpan(),
		"Incomplete block \""+blockName+"\". If you meant to write the @ character, you should use the \"&#64;\" HTML entity instead.",
	))
}

// _consumeLet consumes a let declaration token
func (tb *TreeBuilder) _consumeLet(startToken Token) {
	parts := startToken.Parts()
	name := ""
	if len(parts) > 0 {
		name = parts[0]
	}

	var valueToken Token
	if tb.peek == nil || tb.peek.Type() != TokenTypeLET_VALUE {
		tb.errors = append(tb.errors, NewTreeError(
			&name,
			startToken.SourceSpan(),
			"Invalid @let declaration \""+name+"\". Declaration must have a value.",
		))
		return
	}
	valueToken = tb.advance()

	var endToken Token
	if tb.peek == nil || tb.peek.Type() != TokenTypeLET_END {
		tb.errors = append(tb.errors, NewTreeError(
			&name,
			startToken.SourceSpan(),
			"Unterminated @let declaration \""+name+"\". Declaration must be terminated with a semicolon.",
		))
		return
	}
	endToken = tb.advance()

	value := ""
	if len(valueToken.Parts()) > 0 {
		value = valueToken.Parts()[0]
	}

	end := endToken.SourceSpan().FullStart
	span := util.NewParseSourceSpan(
		startToken.SourceSpan().Start,
		end,
		startToken.SourceSpan().FullStart,
		nil,
	)

	// The start token usually captures the `@let`. Construct a name span by
	// offsetting the start by the length of any text before the name.
	startSpanStr := startToken.SourceSpan().String()
	nameOffset := strings.LastIndex(startSpanStr, name)
	var nameStart *util.ParseLocation
	if nameOffset >= 0 {
		// Calculate name start by moving forward from startToken's start
		nameStart = util.NewParseLocation(
			startToken.SourceSpan().Start.File,
			startToken.SourceSpan().Start.Offset+nameOffset,
			startToken.SourceSpan().Start.Line,
			startToken.SourceSpan().Start.Col+nameOffset,
		)
	} else {
		nameStart = startToken.SourceSpan().Start
	}
	nameSpan := util.NewParseSourceSpan(
		nameStart,
		startToken.SourceSpan().End,
		nameStart,
		nil,
	)

	node := NewLetDeclaration(name, value, span, nameSpan, valueToken.SourceSpan())
	tb._addToParent(node)
}

// _consumeIncompleteLet consumes an incomplete let declaration token
func (tb *TreeBuilder) _consumeIncompleteLet(token Token) {
	parts := token.Parts()
	name := ""
	if len(parts) > 0 {
		name = parts[0]
	}
	nameString := ""
	if name != "" {
		nameString = " \"" + name + "\""
	}

	// If there's at least a name, we can salvage an AST node that can be used for completions.
	if name != "" {
		startSpanStr := token.SourceSpan().String()
		nameOffset := strings.LastIndex(startSpanStr, name)
		var nameStart *util.ParseLocation
		if nameOffset >= 0 {
			nameStart = util.NewParseLocation(
				token.SourceSpan().Start.File,
				token.SourceSpan().Start.Offset+nameOffset,
				token.SourceSpan().Start.Line,
				token.SourceSpan().Start.Col+nameOffset,
			)
		} else {
			nameStart = token.SourceSpan().Start
		}
		nameSpan := util.NewParseSourceSpan(
			nameStart,
			token.SourceSpan().End,
			nameStart,
			nil,
		)
		valueSpan := util.NewParseSourceSpan(
			token.SourceSpan().Start,
			token.SourceSpan().Start,
			token.SourceSpan().Start,
			nil,
		)
		node := NewLetDeclaration(name, "", token.SourceSpan(), nameSpan, valueSpan)
		tb._addToParent(node)
	}

	tb.errors = append(tb.errors, NewTreeError(
		&name,
		token.SourceSpan(),
		"Incomplete @let declaration"+nameString+". @let declarations must be written as `@let <name> = <value>;`",
	))
}

// _consumeComponentStartTag consumes a component start tag token
func (tb *TreeBuilder) _consumeComponentStartTag(startToken Token) {
	parts := startToken.Parts()
	componentName := ""
	if len(parts) > 0 {
		componentName = parts[0]
	}

	attrs := []*Attribute{}
	directives := []*Directive{}
	tb._consumeAttributesAndDirectives(&attrs, &directives)

	closestElement := tb._getClosestElementLikeParent()
	tagName := tb._getComponentTagName(startToken, closestElement)
	fullName := tb._getComponentFullName(startToken, closestElement)

	selfClosing := false
	if tb.peek != nil && tb.peek.Type() == TokenTypeCOMPONENT_OPEN_END_VOID {
		selfClosing = true
		tb.advance()
	} else if tb.peek != nil && tb.peek.Type() == TokenTypeCOMPONENT_OPEN_END {
		tb.advance()
	}

	var end *util.ParseLocation
	if tb.peek != nil {
		end = tb.peek.SourceSpan().FullStart
	} else {
		end = startToken.SourceSpan().End
	}
	span := util.NewParseSourceSpan(
		startToken.SourceSpan().Start,
		end,
		startToken.SourceSpan().FullStart,
		nil,
	)
	startSpan := util.NewParseSourceSpan(
		startToken.SourceSpan().Start,
		end,
		startToken.SourceSpan().FullStart,
		nil,
	)

	node := NewComponent(
		componentName,
		tagName,
		fullName,
		attrs,
		directives,
		[]Node{},
		selfClosing,
		span,
		startSpan,
		nil,
		nil,
	)

	parent := tb._getContainer()
	isClosedByChild := false
	if parent != nil && tagName != nil {
		tagDef := tb._getTagDefinition(parent)
		if tagDef != nil && tagDef.IsClosedByChild(*tagName) {
			isClosedByChild = true
		}
	}
	tb._pushContainer(node, isClosedByChild)

	if selfClosing {
		tb._popContainer(&fullName, (*Component)(nil), span)
	} else if startToken.Type() == TokenTypeINCOMPLETE_COMPONENT_OPEN {
		tb._popContainer(&fullName, (*Component)(nil), nil)
		tb.errors = append(tb.errors, NewTreeError(
			&fullName,
			span,
			"Opening tag \""+fullName+"\" not terminated.",
		))
	}
}

// _consumeComponentEndTag consumes a component end tag token
func (tb *TreeBuilder) _consumeComponentEndTag(endToken Token) {
	fullName := tb._getComponentFullName(endToken, tb._getClosestElementLikeParent())
	parts := endToken.Parts()
	componentName := ""
	if len(parts) > 0 {
		componentName = parts[0]
	}

	if !tb._popContainer(&fullName, (*Component)(nil), endToken.SourceSpan()) {
		container := tb._getContainer()
		var suffix string
		if container != nil {
			if comp, ok := container.(*Component); ok && comp.ComponentName == componentName {
				suffix = ", did you mean \"" + comp.FullName + "\"?"
			} else {
				suffix = ". It may happen when the tag has already been closed by another tag."
			}
		} else {
			suffix = ". It may happen when the tag has already been closed by another tag."
		}

		errMsg := "Unexpected closing tag \"" + fullName + "\"" + suffix
		tb.errors = append(tb.errors, NewTreeError(
			&fullName,
			endToken.SourceSpan(),
			errMsg,
		))
	}
}

// lastOnStack checks if the last element on the stack matches the given element
func lastOnStack(stack []TokenType, element TokenType) bool {
	return len(stack) > 0 && stack[len(stack)-1] == element
}

// RootNodes returns the root nodes
func (tb *TreeBuilder) RootNodes() []Node {
	return tb.rootNodes
}

// Errors returns the errors
func (tb *TreeBuilder) Errors() []*TreeError {
	return tb.errors
}
