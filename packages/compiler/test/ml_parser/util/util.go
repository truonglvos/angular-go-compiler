package util

import (
	"ngc-go/packages/compiler/src/ml_parser"
	"strings"
)

type SerializerVisitor struct{}

func NewSerializerVisitor() *SerializerVisitor {
	return &SerializerVisitor{}
}

func (s *SerializerVisitor) VisitElement(element *ml_parser.Element, context interface{}) interface{} {
	attrs := s.visitAll(convertAttributesToNodes(element.Attrs), " ", " ")
	attrs += s.visitAll(convertDirectivesToNodes(element.Directives), " ", " ")

	tagDef := ml_parser.GetHtmlTagDefinition(element.Name)
	if tagDef.IsVoid() {
		return "<" + element.Name + attrs + "/>"
	}

	return "<" + element.Name + attrs + ">" + s.visitAll(element.Children, "", "") + "</" + element.Name + ">"
}

func (s *SerializerVisitor) VisitAttribute(attribute *ml_parser.Attribute, context interface{}) interface{} {
	return attribute.Name + "=\"" + attribute.Value + "\""
}

func (s *SerializerVisitor) VisitText(text *ml_parser.Text, context interface{}) interface{} {
	return text.Value
}

func (s *SerializerVisitor) VisitComment(comment *ml_parser.Comment, context interface{}) interface{} {
	value := ""
	if comment.Value != nil {
		value = *comment.Value
	}
	return "<!--" + value + "-->"
}

func (s *SerializerVisitor) VisitExpansion(expansion *ml_parser.Expansion, context interface{}) interface{} {
	return "{" + expansion.SwitchValue + ", " + expansion.Type + "," + s.visitAll(convertExpansionCasesToNodes(expansion.Cases), "", "") + "}"
}

func (s *SerializerVisitor) VisitExpansionCase(expansionCase *ml_parser.ExpansionCase, context interface{}) interface{} {
	return " " + expansionCase.Value + " {" + s.visitAll(expansionCase.Expression, "", "") + "}"
}

func (s *SerializerVisitor) VisitBlock(block *ml_parser.Block, context interface{}) interface{} {
	params := " "
	if len(block.Parameters) > 0 {
		params = " (" + s.visitAll(convertBlockParametersToNodes(block.Parameters), ";", " ") + ") "
	}
	return "@" + block.Name + params + "{" + s.visitAll(block.Children, "", "") + "}"
}

func (s *SerializerVisitor) VisitBlockParameter(parameter *ml_parser.BlockParameter, context interface{}) interface{} {
	return parameter.Expression
}

func (s *SerializerVisitor) VisitLetDeclaration(decl *ml_parser.LetDeclaration, context interface{}) interface{} {
	return "@let " + decl.Name + " = " + decl.Value + ";"
}

func (s *SerializerVisitor) VisitComponent(node *ml_parser.Component, context interface{}) interface{} {
	attrs := s.visitAll(convertAttributesToNodes(node.Attrs), " ", " ")
	attrs += s.visitAll(convertDirectivesToNodes(node.Directives), " ", " ")
	return "<" + node.FullName + attrs + ">" + s.visitAll(node.Children, "", "") + "</" + node.FullName + ">"
}

func (s *SerializerVisitor) VisitDirective(directive *ml_parser.Directive, context interface{}) interface{} {
	return "@" + directive.Name + s.visitAll(convertAttributesToNodes(directive.Attrs), " ", " ")
}

func (s *SerializerVisitor) Visit(node ml_parser.Node, context interface{}) interface{} {
	return node.Visit(s, context)
}

func (s *SerializerVisitor) visitAll(nodes []ml_parser.Node, separator, prefix string) string {
	if len(nodes) > 0 {
		results := make([]string, len(nodes))
		for i, node := range nodes {
			result := node.Visit(s, nil)
			if str, ok := result.(string); ok {
				results[i] = str
			}
		}
		return prefix + strings.Join(results, separator)
	}
	return ""
}

// Helper functions for converting slices to []ml_parser.Node

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

var serializerVisitor = NewSerializerVisitor()

// SerializeNodes serializes nodes to strings
func SerializeNodes(nodes []ml_parser.Node) []string {
	result := make([]string, len(nodes))
	for i, node := range nodes {
		serialized := node.Visit(serializerVisitor, nil)
		if str, ok := serialized.(string); ok {
			result[i] = str
		}
	}
	return result
}
