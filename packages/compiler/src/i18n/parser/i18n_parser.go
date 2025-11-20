package i18n_parser

import (
	"fmt"
	"regexp"
	"strings"

	"ngc-go/packages/compiler/src/expressionparser"
	"ngc-go/packages/compiler/src/i18n"
	"ngc-go/packages/compiler/src/i18n/serializers"
	"ngc-go/packages/compiler/src/ml_parser"
	"ngc-go/packages/compiler/src/util"
)

// VisitNodeFn is a function type for visiting nodes
type VisitNodeFn func(html ml_parser.Node, i18n i18n.Node) i18n.Node

// I18nMessageFactory is a function type that creates i18n messages
type I18nMessageFactory func(
	nodes []ml_parser.Node,
	meaning *string,
	description *string,
	customID *string,
	visitNodeFn VisitNodeFn,
) *i18n.Message

// CreateI18nMessageFactory returns a function converting HTML nodes to an i18n Message
func CreateI18nMessageFactory(
	containerBlocks map[string]bool,
	retainEmptyTokens bool,
	preserveExpressionWhitespace bool,
) I18nMessageFactory {
	expParser := expressionparser.NewParser(expressionparser.NewLexer(), false)
	visitor := NewI18nVisitor(
		expParser,
		containerBlocks,
		retainEmptyTokens,
		preserveExpressionWhitespace,
	)
	return func(nodes []ml_parser.Node, meaning *string, description *string, customID *string, visitNodeFn VisitNodeFn) *i18n.Message {
		meaningStr := ""
		if meaning != nil {
			meaningStr = *meaning
		}
		descriptionStr := ""
		if description != nil {
			descriptionStr = *description
		}
		customIDStr := ""
		if customID != nil {
			customIDStr = *customID
		}
		return visitor.ToI18nMessage(nodes, meaningStr, descriptionStr, customIDStr, visitNodeFn)
	}
}

// I18nMessageVisitorContext represents the context for visiting i18n messages
type I18nMessageVisitorContext struct {
	IsIcu                bool
	IcuDepth             int
	PlaceholderRegistry  *serializers.PlaceholderRegistry
	PlaceholderToContent map[string]i18n.MessagePlaceholder
	PlaceholderToMessage map[string]*i18n.Message
	VisitNodeFn          VisitNodeFn
}

// NoopVisitNodeFn is a no-op visit node function
func NoopVisitNodeFn(html ml_parser.Node, i18n i18n.Node) i18n.Node {
	return i18n
}

// I18nVisitor implements ml_parser.Visitor to convert HTML nodes to i18n nodes
type I18nVisitor struct {
	expressionParser             *expressionparser.Parser
	containerBlocks              map[string]bool
	retainEmptyTokens            bool
	preserveExpressionWhitespace bool
}

// Visit implements ml_parser.Visitor interface
func (v *I18nVisitor) Visit(node ml_parser.Node, context interface{}) interface{} {
	return node.Visit(v, context)
}

// NewI18nVisitor creates a new I18nVisitor
func NewI18nVisitor(
	expressionParser *expressionparser.Parser,
	containerBlocks map[string]bool,
	retainEmptyTokens bool,
	preserveExpressionWhitespace bool,
) *I18nVisitor {
	return &I18nVisitor{
		expressionParser:             expressionParser,
		containerBlocks:              containerBlocks,
		retainEmptyTokens:            retainEmptyTokens,
		preserveExpressionWhitespace: preserveExpressionWhitespace,
	}
}

// ToI18nMessage converts HTML nodes to an i18n Message
func (v *I18nVisitor) ToI18nMessage(
	nodes []ml_parser.Node,
	meaning string,
	description string,
	customID string,
	visitNodeFn VisitNodeFn,
) *i18n.Message {
	context := &I18nMessageVisitorContext{
		IsIcu:                len(nodes) == 1 && isExpansion(nodes[0]),
		IcuDepth:             0,
		PlaceholderRegistry:  serializers.NewPlaceholderRegistry(),
		PlaceholderToContent: make(map[string]i18n.MessagePlaceholder),
		PlaceholderToMessage: make(map[string]*i18n.Message),
		VisitNodeFn:          visitNodeFn,
	}

	if visitNodeFn == nil {
		context.VisitNodeFn = NoopVisitNodeFn
	}

	i18nodes := visitAll(v, nodes, context)

	return i18n.NewMessage(
		i18nodes,
		context.PlaceholderToContent,
		context.PlaceholderToMessage,
		meaning,
		description,
		customID,
	)
}

// isExpansion checks if a node is an Expansion node
func isExpansion(node ml_parser.Node) bool {
	_, ok := node.(*ml_parser.Expansion)
	return ok
}

// visitAll visits all nodes with the visitor
func visitAll(visitor *I18nVisitor, nodes []ml_parser.Node, context *I18nMessageVisitorContext) []i18n.Node {
	result := make([]i18n.Node, 0, len(nodes))
	for _, node := range nodes {
		visitResult := node.Visit(visitor, context)
		if i18nNode, ok := visitResult.(i18n.Node); ok {
			result = append(result, i18nNode)
		}
	}
	return result
}

// VisitElement visits an Element node
func (v *I18nVisitor) VisitElement(element *ml_parser.Element, context interface{}) interface{} {
	return v.visitElementLike(element, context)
}

// VisitText visits a Text node
func (v *I18nVisitor) VisitText(text *ml_parser.Text, context interface{}) interface{} {
	ctx := context.(*I18nMessageVisitorContext)

	var node i18n.Node
	if len(text.Tokens) == 1 {
		span := text.SourceSpan()
		// compiler.ParseSourceSpan should be the same as util.ParseSourceSpan
		// We'll use type assertion to convert
		node = i18n.NewText(text.Value, span)
	} else {
		tokens := make([]ml_parser.Token, len(text.Tokens))
		for i, tok := range text.Tokens {
			tokens[i] = tok
		}
		node = v.visitTextWithInterpolation(tokens, text.SourceSpan(), ctx, getI18n(text))
	}
	return ctx.VisitNodeFn(text, node)
}

// VisitComment visits a Comment node
func (v *I18nVisitor) VisitComment(comment *ml_parser.Comment, context interface{}) interface{} {
	// Comments are typically ignored in i18n
	return nil
}

// VisitAttribute visits an Attribute node
func (v *I18nVisitor) VisitAttribute(attribute *ml_parser.Attribute, context interface{}) interface{} {
	ctx := context.(*I18nMessageVisitorContext)

	var node i18n.Node
	if attribute.ValueTokens == nil || len(attribute.ValueTokens) == 1 {
		valueSpan := attribute.ValueSpan
		if valueSpan == nil {
			valueSpan = attribute.SourceSpan()
		}
		node = i18n.NewText(attribute.Value, valueSpan)
	} else {
		valueSpan := attribute.ValueSpan
		if valueSpan == nil {
			valueSpan = attribute.SourceSpan()
		}
		tokens := make([]ml_parser.Token, len(attribute.ValueTokens))
		for i, tok := range attribute.ValueTokens {
			tokens[i] = tok
		}
		node = v.visitTextWithInterpolation(tokens, valueSpan, ctx, getI18n(attribute))
	}
	return ctx.VisitNodeFn(attribute, node)
}

// VisitExpansion visits an Expansion node
func (v *I18nVisitor) VisitExpansion(icu *ml_parser.Expansion, context interface{}) interface{} {
	ctx := context.(*I18nMessageVisitorContext)

	ctx.IcuDepth++
	i18nIcuCases := make(map[string]i18n.Node)
	i18nIcu := i18n.NewIcu(icu.SwitchValue, icu.Type, i18nIcuCases, icu.SourceSpan(), "")

	for _, caze := range icu.Cases {
		caseNodes := make([]i18n.Node, 0, len(caze.Expression))
		for _, node := range caze.Expression {
			visitResult := node.Visit(v, context)
			if i18nNode, ok := visitResult.(i18n.Node); ok {
				caseNodes = append(caseNodes, i18nNode)
			}
		}
		i18nIcuCases[caze.Value] = i18n.NewContainer(caseNodes, caze.ExpSourceSpan)
	}
	ctx.IcuDepth--

	if ctx.IsIcu || ctx.IcuDepth > 0 {
		// Returns an ICU node when:
		// - the message (vs a part of the message) is an ICU message, or
		// - the ICU message is nested.
		expPh := ctx.PlaceholderRegistry.GetUniquePlaceholder(fmt.Sprintf("VAR_%s", icu.Type))
		i18nIcu.ExpressionPlaceholder = expPh
		ctx.PlaceholderToContent[expPh] = i18n.MessagePlaceholder{
			Text:       icu.SwitchValue,
			SourceSpan: icu.SwitchValueSourceSpan,
		}
		return ctx.VisitNodeFn(icu, i18nIcu)
	}

	// Else returns a placeholder
	// ICU placeholders should not be replaced with their original content but with the their
	// translations.
	phName := ctx.PlaceholderRegistry.GetPlaceholderName("ICU", icu.SourceSpan().String())
	ctx.PlaceholderToMessage[phName] = v.ToI18nMessage([]ml_parser.Node{icu}, "", "", "", nil)
	node := i18n.NewIcuPlaceholder(i18nIcu, phName, icu.SourceSpan())
	return ctx.VisitNodeFn(icu, node)
}

// VisitExpansionCase visits an ExpansionCase node
func (v *I18nVisitor) VisitExpansionCase(expansionCase *ml_parser.ExpansionCase, context interface{}) interface{} {
	panic("Unreachable code")
}

// VisitBlock visits a Block node
func (v *I18nVisitor) VisitBlock(block *ml_parser.Block, context interface{}) interface{} {
	ctx := context.(*I18nMessageVisitorContext)

	children := visitAll(v, block.Children, ctx)

	if v.containerBlocks[block.Name] {
		return i18n.NewContainer(children, block.SourceSpan())
	}

	parameters := make([]string, len(block.Parameters))
	for i, param := range block.Parameters {
		parameters[i] = param.Expression
	}
	startPhName := ctx.PlaceholderRegistry.GetStartBlockPlaceholderName(block.Name, parameters)
	closePhName := ctx.PlaceholderRegistry.GetCloseBlockPlaceholderName(block.Name)

	ctx.PlaceholderToContent[startPhName] = i18n.MessagePlaceholder{
		Text:       block.StartSourceSpan.String(),
		SourceSpan: block.StartSourceSpan,
	}

	endSpan := block.EndSourceSpan
	if endSpan == nil {
		endSpan = block.SourceSpan()
	}
	endSpanText := "}"
	if block.EndSourceSpan != nil {
		endSpanText = block.EndSourceSpan.String()
	}
	ctx.PlaceholderToContent[closePhName] = i18n.MessagePlaceholder{
		Text:       endSpanText,
		SourceSpan: endSpan,
	}

	node := i18n.NewBlockPlaceholder(
		block.Name,
		parameters,
		startPhName,
		closePhName,
		children,
		block.SourceSpan(),
		block.StartSourceSpan,
		block.EndSourceSpan,
	)
	return ctx.VisitNodeFn(block, node)
}

// VisitBlockParameter visits a BlockParameter node
func (v *I18nVisitor) VisitBlockParameter(blockParameter *ml_parser.BlockParameter, context interface{}) interface{} {
	panic("Unreachable code")
}

// VisitLetDeclaration visits a LetDeclaration node
func (v *I18nVisitor) VisitLetDeclaration(letDeclaration *ml_parser.LetDeclaration, context interface{}) interface{} {
	// LetDeclaration is typically not part of i18n messages
	return nil
}

// VisitComponent visits a Component node
func (v *I18nVisitor) VisitComponent(component *ml_parser.Component, context interface{}) interface{} {
	return v.visitElementLike(component, context)
}

// VisitDirective visits a Directive node
func (v *I18nVisitor) VisitDirective(directive *ml_parser.Directive, context interface{}) interface{} {
	panic("Unreachable code")
}

// getI18n gets i18n metadata from a node
func getI18n(node ml_parser.Node) interface{} {
	switch n := node.(type) {
	case *ml_parser.Text:
		if n.NodeWithI18n != nil {
			return n.NodeWithI18n.I18n()
		}
	case *ml_parser.Attribute:
		if n.NodeWithI18n != nil {
			return n.NodeWithI18n.I18n()
		}
	case *ml_parser.Element:
		if n.NodeWithI18n != nil {
			return n.NodeWithI18n.I18n()
		}
	case *ml_parser.Component:
		if n.NodeWithI18n != nil {
			return n.NodeWithI18n.I18n()
		}
	}
	return nil
}

// convertUtilToCompilerSpan converts util.ParseSourceSpan to compiler.ParseSourceSpan
// Note: compiler.ParseSourceSpan is defined in i18n package as an alias or type
// We need to check the actual definition, but for now we'll use unsafe conversion
func convertUtilToCompilerSpan(span *util.ParseSourceSpan) interface{} {
	// Since compiler.ParseSourceSpan might be the same as util.ParseSourceSpan
	// or a type alias, we'll return it as interface{} and let the caller handle it
	// This is a workaround until we know the exact type relationship
	return span
}

// visitElementLike visits an Element or Component node
func (v *I18nVisitor) visitElementLike(node interface{}, context interface{}) interface{} {
	ctx := context.(*I18nMessageVisitorContext)

	var children []ml_parser.Node
	var attrs map[string]string
	var nodeName string
	var isVoid bool
	var sourceSpan *util.ParseSourceSpan
	var startSourceSpan *util.ParseSourceSpan
	var endSourceSpan *util.ParseSourceSpan

	switch n := node.(type) {
	case *ml_parser.Element:
		children = n.Children
		attrs = make(map[string]string)
		for _, attr := range n.Attrs {
			attrs[attr.Name] = attr.Value
		}
		for _, dir := range n.Directives {
			for _, attr := range dir.Attrs {
				attrs[attr.Name] = attr.Value
			}
		}
		nodeName = n.Name
		tagDef := ml_parser.GetHtmlTagDefinition(n.Name)
		isVoid = tagDef.IsVoid()
		sourceSpan = n.SourceSpan()
		startSourceSpan = n.StartSourceSpan
		endSourceSpan = n.EndSourceSpan
	case *ml_parser.Component:
		children = n.Children
		attrs = make(map[string]string)
		for _, attr := range n.Attrs {
			attrs[attr.Name] = attr.Value
		}
		for _, dir := range n.Directives {
			for _, attr := range dir.Attrs {
				attrs[attr.Name] = attr.Value
			}
		}
		nodeName = n.FullName
		if n.TagName != nil {
			tagDef := ml_parser.GetHtmlTagDefinition(*n.TagName)
			isVoid = tagDef.IsVoid()
		} else {
			isVoid = false
		}
		sourceSpan = n.SourceSpan()
		startSourceSpan = n.StartSourceSpan
		endSourceSpan = n.EndSourceSpan
	default:
		panic(fmt.Sprintf("unexpected node type: %T", node))
	}

	i18nChildren := visitAll(v, children, ctx)

	startPhName := ctx.PlaceholderRegistry.GetStartTagPlaceholderName(nodeName, attrs, isVoid)
	ctx.PlaceholderToContent[startPhName] = i18n.MessagePlaceholder{
		Text:       startSourceSpan.String(),
		SourceSpan: startSourceSpan,
	}

	closePhName := ""
	if !isVoid {
		closePhName = ctx.PlaceholderRegistry.GetCloseTagPlaceholderName(nodeName)
		endSpan := endSourceSpan
		if endSpan == nil {
			endSpan = sourceSpan
		}
		ctx.PlaceholderToContent[closePhName] = i18n.MessagePlaceholder{
			Text:       fmt.Sprintf("</%s>", nodeName),
			SourceSpan: endSpan,
		}
	}

	i18nNode := i18n.NewTagPlaceholder(
		nodeName,
		attrs,
		startPhName,
		closePhName,
		i18nChildren,
		isVoid,
		sourceSpan,
		startSourceSpan,
		endSourceSpan,
	)

	var htmlNode ml_parser.Node
	switch n := node.(type) {
	case *ml_parser.Element:
		htmlNode = n
	case *ml_parser.Component:
		htmlNode = n
	}
	return ctx.VisitNodeFn(htmlNode, i18nNode)
}

// visitTextWithInterpolation converts text and interpolated tokens into text and placeholder pieces
func (v *I18nVisitor) visitTextWithInterpolation(
	tokens []ml_parser.Token,
	sourceSpan *util.ParseSourceSpan,
	context *I18nMessageVisitorContext,
	previousI18n interface{},
) i18n.Node {
	nodes := make([]i18n.Node, 0)
	hasInterpolation := false

	for _, token := range tokens {
		tokenType := token.Type()
		switch tokenType {
		case ml_parser.TokenTypeINTERPOLATION, ml_parser.TokenTypeATTR_VALUE_INTERPOLATION:
			hasInterpolation = true
			parts := token.Parts()
			if len(parts) < 3 {
				continue
			}
			startMarker := parts[0]
			expression := parts[1]
			endMarker := parts[2]

			baseName := extractPlaceholderName(expression)
			if baseName == "" {
				baseName = "INTERPOLATION"
			}
			phName := context.PlaceholderRegistry.GetPlaceholderName(baseName, expression)

			if v.preserveExpressionWhitespace {
				context.PlaceholderToContent[phName] = i18n.MessagePlaceholder{
					Text:       strings.Join(parts, ""),
					SourceSpan: token.SourceSpan(),
				}
				nodes = append(nodes, i18n.NewPlaceholder(expression, phName, token.SourceSpan()))
			} else {
				normalized := v.normalizeExpression(token)
				context.PlaceholderToContent[phName] = i18n.MessagePlaceholder{
					Text:       fmt.Sprintf("%s%s%s", startMarker, normalized, endMarker),
					SourceSpan: token.SourceSpan(),
				}
				nodes = append(nodes, i18n.NewPlaceholder(normalized, phName, token.SourceSpan()))
			}
		default:
			parts := token.Parts()
			if len(parts) == 0 {
				continue
			}
			textValue := parts[0]
			if len(textValue) > 0 || v.retainEmptyTokens {
				if len(nodes) > 0 {
					if lastText, ok := nodes[len(nodes)-1].(*i18n.Text); ok {
						lastText.Value += textValue
						// Create new Text with merged source span
						mergedSpan := util.NewParseSourceSpan(
							lastText.SourceSpan().Start,
							token.SourceSpan().End,
							lastText.SourceSpan().FullStart,
							lastText.SourceSpan().Details,
						)
						// Replace the last node with new one
						nodes[len(nodes)-1] = i18n.NewText(lastText.Value, mergedSpan)
					} else {
						nodes = append(nodes, i18n.NewText(textValue, token.SourceSpan()))
					}
				} else {
					nodes = append(nodes, i18n.NewText(textValue, token.SourceSpan()))
				}
			} else {
				if v.retainEmptyTokens {
					nodes = append(nodes, i18n.NewText(textValue, token.SourceSpan()))
				}
			}
		}
	}

	if hasInterpolation {
		reusePreviousSourceSpans(nodes, previousI18n)
		return i18n.NewContainer(nodes, sourceSpan)
	} else {
		if len(nodes) > 0 {
			return nodes[0]
		}
		return i18n.NewText("", sourceSpan)
	}
}

// normalizeExpression normalizes expression whitespace by parsing and re-serializing it
func (v *I18nVisitor) normalizeExpression(token ml_parser.Token) string {
	parts := token.Parts()
	if len(parts) < 2 {
		return ""
	}
	expression := parts[1]
	expr := v.expressionParser.ParseBinding(expression, token.SourceSpan(), token.SourceSpan().Start.Offset)
	return expressionparser.Serialize(expr)
}

// reusePreviousSourceSpans re-uses the source-spans from previousI18n metadata for the nodes
func reusePreviousSourceSpans(nodes []i18n.Node, previousI18n interface{}) {
	if previousI18n == nil {
		return
	}

	var container *i18n.Container
	if msg, ok := previousI18n.(*i18n.Message); ok {
		assertSingleContainerMessage(msg)
		if len(msg.Nodes) > 0 {
			if c, ok := msg.Nodes[0].(*i18n.Container); ok {
				container = c
			}
		}
	} else if c, ok := previousI18n.(*i18n.Container); ok {
		container = c
	}

	if container != nil {
		assertEquivalentNodes(container.Children, nodes)
		// Reuse source spans by creating new nodes with the previous source spans
		for i := 0; i < len(nodes); i++ {
			prevSpan := container.Children[i].SourceSpan()
			switch n := nodes[i].(type) {
			case *i18n.Text:
				nodes[i] = i18n.NewText(n.Value, prevSpan)
			case *i18n.Placeholder:
				nodes[i] = i18n.NewPlaceholder(n.Value, n.Name, prevSpan)
			case *i18n.Container:
				// For Container, we need to keep children but update source span
				// Since sourceSpan is private, we'll need to create a new Container
				// But this is complex, so we'll skip for now
			}
		}
	}
}

// assertSingleContainerMessage asserts that the message contains exactly one Container node
func assertSingleContainerMessage(message *i18n.Message) {
	nodes := message.Nodes
	if len(nodes) != 1 {
		panic("Unexpected previous i18n message - expected it to consist of only a single `Container` node.")
	}
	if _, ok := nodes[0].(*i18n.Container); !ok {
		panic("Unexpected previous i18n message - expected it to consist of only a single `Container` node.")
	}
}

// assertEquivalentNodes asserts that previousNodes and nodes have the same number of elements
// and corresponding elements have the same node type
func assertEquivalentNodes(previousNodes []i18n.Node, nodes []i18n.Node) {
	if len(previousNodes) != len(nodes) {
		panic(fmt.Sprintf(
			"The number of i18n message children changed between first and second pass.\n\nFirst pass (%d tokens):\n%s\n\nSecond pass (%d tokens):\n%s",
			len(previousNodes),
			formatNodes(previousNodes),
			len(nodes),
			formatNodes(nodes),
		))
	}
	for i := 0; i < len(previousNodes); i++ {
		if getNodeType(previousNodes[i]) != getNodeType(nodes[i]) {
			panic("The types of the i18n message children changed between first and second pass.")
		}
	}
}

// formatNodes formats nodes for error messages
func formatNodes(nodes []i18n.Node) string {
	lines := make([]string, len(nodes))
	for i, node := range nodes {
		lines[i] = fmt.Sprintf("\"%s\"", node.SourceSpan().String())
	}
	return strings.Join(lines, "\n")
}

// getNodeType returns the type name of a node
func getNodeType(node i18n.Node) string {
	switch node.(type) {
	case *i18n.Text:
		return "Text"
	case *i18n.Container:
		return "Container"
	case *i18n.Placeholder:
		return "Placeholder"
	case *i18n.Icu:
		return "Icu"
	case *i18n.TagPlaceholder:
		return "TagPlaceholder"
	case *i18n.IcuPlaceholder:
		return "IcuPlaceholder"
	case *i18n.BlockPlaceholder:
		return "BlockPlaceholder"
	default:
		return "Unknown"
	}
}

var customPhExp = regexp.MustCompile(`//[\s\S]*i18n[\s\S]*\([\s\S]*ph[\s\S]*=[\s\S]*("|')([\s\S]*?)\1[\s\S]*\)`)

// extractPlaceholderName extracts placeholder name from expression
func extractPlaceholderName(input string) string {
	matches := customPhExp.FindStringSubmatch(input)
	if len(matches) > 2 {
		return matches[2]
	}
	return ""
}
