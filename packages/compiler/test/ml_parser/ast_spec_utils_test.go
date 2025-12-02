package ml_parser_test

import (
	"fmt"
	"ngc-go/packages/compiler/src/ml_parser"
	"ngc-go/packages/compiler/src/util"
)

func HumanizeDom(parseResult *ml_parser.ParseTreeResult, addSourceSpan bool) []interface{} {
	if len(parseResult.Errors) > 0 {
		errorString := ""
		for _, err := range parseResult.Errors {
			errorString += err.String() + "\n"
		}
		panic(fmt.Errorf("Unexpected parse errors:\n%s", errorString))
	}

	return HumanizeNodes(parseResult.RootNodes, addSourceSpan)
}

func HumanizeDomSourceSpans(parseResult *ml_parser.ParseTreeResult) []interface{} {
	return HumanizeDom(parseResult, true)
}

func HumanizeNodes(nodes []ml_parser.Node, addSourceSpan bool) []interface{} {
	humanizer := NewHumanizer(addSourceSpan)
	ml_parser.VisitAll(humanizer, nodes, nil)
	return humanizer.Result
}

func HumanizeLineColumn(location *util.ParseLocation) string {
	return fmt.Sprintf("%d:%d", location.Line, location.Col)
}

type Humanizer struct {
	Result            []interface{}
	elDepth           int
	includeSourceSpan bool
}

func NewHumanizer(includeSourceSpan bool) *Humanizer {
	return &Humanizer{
		Result:            []interface{}{},
		includeSourceSpan: includeSourceSpan,
	}
}

func (h *Humanizer) VisitElement(element *ml_parser.Element, context interface{}) interface{} {
	res := h.appendContext(element, []interface{}{"Element", element.Name, h.elDepth})
	h.elDepth++
	if element.IsSelfClosing {
		res = append(res, "#selfClosing")
	}
	if h.includeSourceSpan {
		res = append(res, element.StartSourceSpan.String())
		if element.EndSourceSpan != nil {
			res = append(res, element.EndSourceSpan.String())
		} else {
			res = append(res, nil)
		}
	}
	h.Result = append(h.Result, res)
	ml_parser.VisitAll(h, convertAttributesToNodes(element.Attrs), nil)
	ml_parser.VisitAll(h, convertDirectivesToNodes(element.Directives), nil)
	ml_parser.VisitAll(h, element.Children, nil)
	h.elDepth--
	return nil
}

func (h *Humanizer) VisitAttribute(attribute *ml_parser.Attribute, context interface{}) interface{} {
	valueTokens := attribute.ValueTokens
	if valueTokens == nil {
		valueTokens = []ml_parser.InterpolatedAttributeToken{}
	}
	contextData := []interface{}{"Attribute", attribute.Name, attribute.Value}
	for _, token := range valueTokens {
		parts := token.Parts()
		convertedParts := make([]interface{}, len(parts))
		for i, p := range parts {
			convertedParts[i] = p
		}
		contextData = append(contextData, convertedParts)
	}
	res := h.appendContext(attribute, contextData)
	h.Result = append(h.Result, res)
	return nil
}

func (h *Humanizer) VisitText(text *ml_parser.Text, context interface{}) interface{} {
	contextData := []interface{}{"Text", text.Value, h.elDepth}
	for _, token := range text.Tokens {
		parts := token.Parts()
		convertedParts := make([]interface{}, len(parts))
		for i, p := range parts {
			convertedParts[i] = p
		}
		contextData = append(contextData, convertedParts)
	}
	res := h.appendContext(text, contextData)
	h.Result = append(h.Result, res)
	return nil
}

func (h *Humanizer) VisitComment(comment *ml_parser.Comment, context interface{}) interface{} {
	value := ""
	if comment.Value != nil {
		value = *comment.Value
	}
	res := h.appendContext(comment, []interface{}{"Comment", value, h.elDepth})
	h.Result = append(h.Result, res)
	return nil
}

func (h *Humanizer) VisitExpansion(expansion *ml_parser.Expansion, context interface{}) interface{} {
	res := h.appendContext(expansion, []interface{}{"Expansion", expansion.SwitchValue, expansion.Type, h.elDepth})
	h.elDepth++
	h.Result = append(h.Result, res)
	ml_parser.VisitAll(h, convertExpansionCasesToNodes(expansion.Cases), nil)
	h.elDepth--
	return nil
}

func (h *Humanizer) VisitExpansionCase(expansionCase *ml_parser.ExpansionCase, context interface{}) interface{} {
	res := h.appendContext(expansionCase, []interface{}{"ExpansionCase", expansionCase.Value, h.elDepth})
	h.Result = append(h.Result, res)
	return nil
}

func (h *Humanizer) VisitBlock(block *ml_parser.Block, context interface{}) interface{} {
	res := h.appendContext(block, []interface{}{"Block", block.Name, h.elDepth})
	h.elDepth++
	if h.includeSourceSpan {
		res = append(res, block.StartSourceSpan.String())
		if block.EndSourceSpan != nil {
			res = append(res, block.EndSourceSpan.String())
		} else {
			res = append(res, nil)
		}
	}
	h.Result = append(h.Result, res)
	ml_parser.VisitAll(h, convertBlockParametersToNodes(block.Parameters), nil)
	ml_parser.VisitAll(h, block.Children, nil)
	h.elDepth--
	return nil
}

func (h *Humanizer) VisitBlockParameter(parameter *ml_parser.BlockParameter, context interface{}) interface{} {
	h.Result = append(h.Result, h.appendContext(parameter, []interface{}{"BlockParameter", parameter.Expression}))
	return nil
}

func (h *Humanizer) VisitLetDeclaration(decl *ml_parser.LetDeclaration, context interface{}) interface{} {
	res := h.appendContext(decl, []interface{}{"LetDeclaration", decl.Name, decl.Value})
	if h.includeSourceSpan {
		if decl.NameSpan != nil {
			res = append(res, decl.NameSpan.String())
		} else {
			res = append(res, nil)
		}
		if decl.ValueSpan != nil {
			res = append(res, decl.ValueSpan.String())
		} else {
			res = append(res, nil)
		}
	}
	h.Result = append(h.Result, res)
	return nil
}

func (h *Humanizer) VisitComponent(node *ml_parser.Component, context interface{}) interface{} {
	tagName := ""
	if node.TagName != nil {
		tagName = *node.TagName
	}
	res := h.appendContext(node, []interface{}{"Component", node.ComponentName, tagName, node.FullName, h.elDepth})
	h.elDepth++
	if node.IsSelfClosing {
		res = append(res, "#selfClosing")
	}
	if h.includeSourceSpan {
		res = append(res, node.StartSourceSpan.String())
		if node.EndSourceSpan != nil {
			res = append(res, node.EndSourceSpan.String())
		} else {
			res = append(res, nil)
		}
	}
	h.Result = append(h.Result, res)
	ml_parser.VisitAll(h, convertAttributesToNodes(node.Attrs), nil)
	ml_parser.VisitAll(h, convertDirectivesToNodes(node.Directives), nil)
	ml_parser.VisitAll(h, node.Children, nil)
	h.elDepth--
	return nil
}

func (h *Humanizer) VisitDirective(directive *ml_parser.Directive, context interface{}) interface{} {
	res := h.appendContext(directive, []interface{}{"Directive", directive.Name})
	if h.includeSourceSpan {
		res = append(res, directive.StartSourceSpan.String())
		if directive.EndSourceSpan != nil {
			res = append(res, directive.EndSourceSpan.String())
		} else {
			res = append(res, nil)
		}
	}
	h.Result = append(h.Result, res)
	ml_parser.VisitAll(h, convertAttributesToNodes(directive.Attrs), nil)
	return nil
}

func (h *Humanizer) appendContext(ast ml_parser.Node, input []interface{}) []interface{} {
	if !h.includeSourceSpan {
		return input
	}
	input = append(input, ast.SourceSpan().String())
	if ast.SourceSpan().FullStart.Offset != ast.SourceSpan().Start.Offset {
		input = append(input, ast.SourceSpan().FullStart.File.Content[ast.SourceSpan().FullStart.Offset:ast.SourceSpan().End.Offset])
	}
	return input
}

// Visit implements ml_parser.Visitor
func (h *Humanizer) Visit(node ml_parser.Node, context interface{}) interface{} {
	// Call node.Visit to visit the node, which will call the appropriate Visit* method
	// Return a non-nil value (empty struct) to prevent VisitAll from calling ast.Visit again,
	// since we already handled the visit here. This matches TypeScript behavior where
	// visitor.visit returns a truthy value to skip the typed visit.
	node.Visit(h, context)
	return struct{}{} // Return non-nil to prevent duplicate visit
}

// Helpers for converting slices to []ml_parser.Node

func convertAttributesToNodes(attrs []*ml_parser.Attribute) []ml_parser.Node {
	nodes := make([]ml_parser.Node, len(attrs))
	for i, attr := range attrs {
		nodes[i] = attr
	}
	return nodes
}

func convertDirectivesToNodes(directives []*ml_parser.Directive) []ml_parser.Node {
	nodes := make([]ml_parser.Node, len(directives))
	for i, dir := range directives {
		nodes[i] = dir
	}
	return nodes
}

func convertExpansionCasesToNodes(cases []*ml_parser.ExpansionCase) []ml_parser.Node {
	nodes := make([]ml_parser.Node, len(cases))
	for i, c := range cases {
		nodes[i] = c
	}
	return nodes
}

func convertBlockParametersToNodes(params []*ml_parser.BlockParameter) []ml_parser.Node {
	nodes := make([]ml_parser.Node, len(params))
	for i, p := range params {
		nodes[i] = p
	}
	return nodes
}

func boolPtr(b bool) *bool {
	return &b
}
