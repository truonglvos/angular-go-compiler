package ml_parser

import (
	"regexp"
	"strings"
)

const PreserveWsAttrName = "ngPreserveWhitespaces"

var skipWsTrimTags = map[string]bool{
	"pre":      true,
	"template": true,
	"textarea": true,
	"script":   true,
	"style":    true,
}

// Equivalent to \s with \u00a0 (non-breaking space) excluded
const wsChars = " \f\n\r\t\v\u1680\u180e\u2000-\u200a\u2028\u2029\u202f\u205f\u3000\ufeff"

var (
	noWsRegexp      = regexp.MustCompile(`[^` + wsChars + `]`)
	wsReplaceRegexp = regexp.MustCompile(`[` + wsChars + `]{2,}`)
)

// ReplaceNgsp replaces &ngsp; pseudo-entity with a space
func ReplaceNgsp(value string) string {
	return strings.ReplaceAll(value, NGSP_UNICODE, " ")
}

// HasPreserveWhitespacesAttr checks if attributes contain preserve whitespaces attribute
func HasPreserveWhitespacesAttr(attrs []*Attribute) bool {
	for _, attr := range attrs {
		if attr.Name == PreserveWsAttrName {
			return true
		}
	}
	return false
}

// WhitespaceVisitor visits nodes and removes/trims whitespace
type WhitespaceVisitor struct {
	preserveSignificantWhitespace bool
	originalNodeMap               map[Node]Node
	requireContext                bool
	icuExpansionDepth             int
}

// NewWhitespaceVisitor creates a new WhitespaceVisitor
func NewWhitespaceVisitor(preserveSignificantWhitespace bool, originalNodeMap map[Node]Node, requireContext bool) *WhitespaceVisitor {
	return &WhitespaceVisitor{
		preserveSignificantWhitespace: preserveSignificantWhitespace,
		originalNodeMap:               originalNodeMap,
		requireContext:                requireContext,
		icuExpansionDepth:             0,
	}
}

// VisitElement visits an element node
func (w *WhitespaceVisitor) VisitElement(element *Element, context interface{}) interface{} {
	if skipWsTrimTags[element.Name] || HasPreserveWhitespacesAttr(element.Attrs) {
		// Don't descend into elements where we need to preserve whitespaces
		// but still visit all attributes to eliminate one used as a marker to preserve WS
		newAttrs := visitAttributesWithSiblings(w, element.Attrs, context)
		newElement := NewElement(
			element.Name,
			convertToAttributes(newAttrs),
			element.Directives,
			element.Children,
			element.IsSelfClosing,
			element.SourceSpan(),
			element.StartSourceSpan,
			element.EndSourceSpan,
			element.IsVoid,
			element.NodeWithI18n.i18n,
		)
		if w.originalNodeMap != nil {
			w.originalNodeMap[newElement] = element
		}
		return newElement
	}

	newChildren := visitAllWithSiblings(w, element.Children, context)
	newElement := NewElement(
		element.Name,
		element.Attrs,
		element.Directives,
		convertToNodes(newChildren),
		element.IsSelfClosing,
		element.SourceSpan(),
		element.StartSourceSpan,
		element.EndSourceSpan,
		element.IsVoid,
		element.NodeWithI18n.i18n,
	)
	if w.originalNodeMap != nil {
		w.originalNodeMap[newElement] = element
	}
	return newElement
}

// VisitAttribute visits an attribute node
func (w *WhitespaceVisitor) VisitAttribute(attribute *Attribute, context interface{}) interface{} {
	if attribute.Name != PreserveWsAttrName {
		return attribute
	}
	return nil
}

// VisitText visits a text node
func (w *WhitespaceVisitor) VisitText(text *Text, context interface{}) interface{} {
	isNotBlank := noWsRegexp.MatchString(text.Value)

	var hasExpansionSibling bool
	if ctx, ok := context.(*SiblingVisitorContext); ok {
		hasExpansionSibling = (ctx.Prev != nil && isExpansion(ctx.Prev)) ||
			(ctx.Next != nil && isExpansion(ctx.Next))
	}

	// Do not trim whitespace within ICU expansions when preserving significant whitespace
	inIcuExpansion := w.icuExpansionDepth > 0
	if inIcuExpansion && w.preserveSignificantWhitespace {
		return text
	}

	if isNotBlank || hasExpansionSibling {
		// Process the whitespace in the tokens
		tokens := make([]InterpolatedTextToken, len(text.Tokens))
		for i, token := range text.Tokens {
			if textToken, ok := token.(*TextToken); ok && textToken.Type() == TokenTypeTEXT {
				tokens[i] = createWhitespaceProcessedTextToken(textToken)
			} else {
				tokens[i] = token
			}
		}

		// Fully trim message when significant whitespace is not preserved
		if !w.preserveSignificantWhitespace && len(tokens) > 0 {
			if len(tokens) > 0 {
				tokens[0] = trimLeadingWhitespace(tokens[0], context)
			}
			if len(tokens) > 0 {
				tokens[len(tokens)-1] = trimTrailingWhitespace(tokens[len(tokens)-1], context)
			}
		}

		// Process the whitespace of the value
		processed := processWhitespace(text.Value)
		value := processed
		if !w.preserveSignificantWhitespace {
			value = trimLeadingAndTrailingWhitespace(processed, context)
		}

		result := NewText(value, text.SourceSpan(), tokens, text.NodeWithI18n.i18n)
		if w.originalNodeMap != nil {
			w.originalNodeMap[result] = text
		}
		return result
	}

	return nil
}

// VisitComment visits a comment node
func (w *WhitespaceVisitor) VisitComment(comment *Comment, context interface{}) interface{} {
	return comment
}

// VisitExpansion visits an expansion node
func (w *WhitespaceVisitor) VisitExpansion(expansion *Expansion, context interface{}) interface{} {
	w.icuExpansionDepth++
	defer func() { w.icuExpansionDepth-- }()

	newCases := visitAllWithSiblings(w, convertExpansionCasesToNodes(expansion.Cases), context)
	newExpansion := NewExpansion(
		expansion.SwitchValue,
		expansion.Type,
		convertToExpansionCases(newCases),
		expansion.SourceSpan(),
		expansion.SwitchValueSourceSpan,
		expansion.NodeWithI18n.i18n,
	)
	if w.originalNodeMap != nil {
		w.originalNodeMap[newExpansion] = expansion
	}
	return newExpansion
}

// VisitExpansionCase visits an expansion case node
func (w *WhitespaceVisitor) VisitExpansionCase(expansionCase *ExpansionCase, context interface{}) interface{} {
	newExpression := visitAllWithSiblings(w, expansionCase.Expression, context)
	newExpansionCase := NewExpansionCase(
		expansionCase.Value,
		convertToNodes(newExpression),
		expansionCase.SourceSpan(),
		expansionCase.ValueSourceSpan,
		expansionCase.ExpSourceSpan,
	)
	if w.originalNodeMap != nil {
		w.originalNodeMap[newExpansionCase] = expansionCase
	}
	return newExpansionCase
}

// VisitBlock visits a block node
func (w *WhitespaceVisitor) VisitBlock(block *Block, context interface{}) interface{} {
	newChildren := visitAllWithSiblings(w, block.Children, context)
	newBlock := NewBlock(
		block.Name,
		block.Parameters,
		convertToNodes(newChildren),
		block.SourceSpan(),
		block.NameSpan,
		block.StartSourceSpan,
		block.EndSourceSpan,
		block.NodeWithI18n.i18n,
	)
	if w.originalNodeMap != nil {
		w.originalNodeMap[newBlock] = block
	}
	return newBlock
}

// VisitBlockParameter visits a block parameter node
func (w *WhitespaceVisitor) VisitBlockParameter(parameter *BlockParameter, context interface{}) interface{} {
	return parameter
}

// VisitLetDeclaration visits a let declaration node
func (w *WhitespaceVisitor) VisitLetDeclaration(decl *LetDeclaration, context interface{}) interface{} {
	return decl
}

// VisitComponent visits a component node
func (w *WhitespaceVisitor) VisitComponent(component *Component, context interface{}) interface{} {
	if (component.TagName != nil && skipWsTrimTags[*component.TagName]) ||
		HasPreserveWhitespacesAttr(component.Attrs) {
		newAttrs := visitAttributesWithSiblings(w, component.Attrs, context)
		newComponent := NewComponent(
			component.ComponentName,
			component.TagName,
			component.FullName,
			convertToAttributes(newAttrs),
			component.Directives,
			component.Children,
			component.IsSelfClosing,
			component.SourceSpan(),
			component.StartSourceSpan,
			component.EndSourceSpan,
			component.NodeWithI18n.i18n,
		)
		if w.originalNodeMap != nil {
			w.originalNodeMap[newComponent] = component
		}
		return newComponent
	}

	newChildren := visitAllWithSiblings(w, component.Children, context)
	newComponent := NewComponent(
		component.ComponentName,
		component.TagName,
		component.FullName,
		component.Attrs,
		component.Directives,
		convertToNodes(newChildren),
		component.IsSelfClosing,
		component.SourceSpan(),
		component.StartSourceSpan,
		component.EndSourceSpan,
		component.NodeWithI18n.i18n,
	)
	if w.originalNodeMap != nil {
		w.originalNodeMap[newComponent] = component
	}
	return newComponent
}

// VisitDirective visits a directive node
func (w *WhitespaceVisitor) VisitDirective(directive *Directive, context interface{}) interface{} {
	return directive
}

// Visit is the default visit method
func (w *WhitespaceVisitor) Visit(node Node, context interface{}) interface{} {
	if w.requireContext && context == nil {
		panic("WhitespaceVisitor requires context. Visit via visitAllWithSiblings to get this context.")
	}
	return nil
}

// RemoveWhitespaces removes whitespaces from a parse tree result
func RemoveWhitespaces(htmlAstWithErrors *ParseTreeResult, preserveSignificantWhitespace bool) *ParseTreeResult {
	originalNodeMap := make(map[Node]Node)
	visitor := NewWhitespaceVisitor(preserveSignificantWhitespace, originalNodeMap, true)
	newRootNodes := visitAllWithSiblings(visitor, htmlAstWithErrors.RootNodes, nil)
	return NewParseTreeResult(convertToNodes(newRootNodes), htmlAstWithErrors.Errors)
}

// Helper functions

func trimLeadingWhitespace(token InterpolatedTextToken, context interface{}) InterpolatedTextToken {
	if textToken, ok := token.(*TextToken); ok && textToken.Type() == TokenTypeTEXT {
		ctx, ok := context.(*SiblingVisitorContext)
		if !ok || ctx.Prev != nil {
			return token
		}
		return transformTextToken(textToken, func(text string) string {
			return strings.TrimLeft(text, wsChars)
		})
	}
	return token
}

func trimTrailingWhitespace(token InterpolatedTextToken, context interface{}) InterpolatedTextToken {
	if textToken, ok := token.(*TextToken); ok && textToken.Type() == TokenTypeTEXT {
		ctx, ok := context.(*SiblingVisitorContext)
		if !ok || ctx.Next != nil {
			return token
		}
		return transformTextToken(textToken, func(text string) string {
			return strings.TrimRight(text, wsChars)
		})
	}
	return token
}

func trimLeadingAndTrailingWhitespace(text string, context interface{}) string {
	ctx, ok := context.(*SiblingVisitorContext)
	isFirstTokenInTag := !ok || ctx.Prev == nil
	isLastTokenInTag := !ok || ctx.Next == nil

	maybeTrimmedStart := text
	if isFirstTokenInTag {
		maybeTrimmedStart = strings.TrimLeft(text, wsChars)
	}
	maybeTrimmed := maybeTrimmedStart
	if isLastTokenInTag {
		maybeTrimmed = strings.TrimRight(maybeTrimmedStart, wsChars)
	}
	return maybeTrimmed
}

func createWhitespaceProcessedTextToken(token *TextToken) *TextToken {
	parts := token.Parts()
	if len(parts) > 0 {
		processed := processWhitespace(parts[0])
		return NewTextToken(processed, token.Type(), token.SourceSpan())
	}
	return token
}

func transformTextToken(token *TextToken, transform func(string) string) *TextToken {
	parts := token.Parts()
	if len(parts) > 0 {
		transformed := transform(parts[0])
		return NewTextToken(transformed, token.Type(), token.SourceSpan())
	}
	return token
}

func processWhitespace(text string) string {
	return wsReplaceRegexp.ReplaceAllString(ReplaceNgsp(text), " ")
}

// SiblingVisitorContext provides context about sibling nodes
type SiblingVisitorContext struct {
	Prev Node
	Next Node
}

// VisitAllWithSiblings visits all nodes with siblings context
func VisitAllWithSiblings(visitor *WhitespaceVisitor, nodes []Node, ctx interface{}) []interface{} {
	return visitAllWithSiblings(visitor, nodes, ctx)
}

func visitAllWithSiblings(visitor *WhitespaceVisitor, nodes []Node, _ interface{}) []interface{} {
	var result []interface{}

	for i, ast := range nodes {
		ctx := &SiblingVisitorContext{
			Prev: nil,
			Next: nil,
		}
		if i > 0 {
			ctx.Prev = nodes[i-1]
		}
		if i < len(nodes)-1 {
			ctx.Next = nodes[i+1]
		}

		astResult := ast.Visit(visitor, ctx)
		if astResult != nil {
			result = append(result, astResult)
		}
	}

	return result
}

func visitAttributesWithSiblings(visitor *WhitespaceVisitor, attrs []*Attribute, _ interface{}) []interface{} {
	var result []interface{}

	for i, attr := range attrs {
		ctx := &SiblingVisitorContext{
			Prev: nil,
			Next: nil,
		}
		if i > 0 {
			ctx.Prev = attrs[i-1]
		}
		if i < len(attrs)-1 {
			ctx.Next = attrs[i+1]
		}

		attrResult := attr.Visit(visitor, ctx)
		if attrResult != nil {
			result = append(result, attrResult)
		}
	}

	return result
}

// Helper conversion functions
func convertToNodes(results []interface{}) []Node {
	nodes := make([]Node, 0, len(results))
	for _, r := range results {
		if node, ok := r.(Node); ok {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

func convertToAttributes(results []interface{}) []*Attribute {
	attrs := make([]*Attribute, 0, len(results))
	for _, r := range results {
		if attr, ok := r.(*Attribute); ok {
			attrs = append(attrs, attr)
		}
	}
	return attrs
}

func convertExpansionCasesToNodes(cases []*ExpansionCase) []Node {
	nodes := make([]Node, len(cases))
	for i, c := range cases {
		nodes[i] = c
	}
	return nodes
}

func convertToExpansionCases(results []interface{}) []*ExpansionCase {
	cases := make([]*ExpansionCase, 0, len(results))
	for _, r := range results {
		if ec, ok := r.(*ExpansionCase); ok {
			cases = append(cases, ec)
		}
	}
	return cases
}

func isExpansion(node Node) bool {
	_, ok := node.(*Expansion)
	return ok
}
