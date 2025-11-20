package viewi18n

import (
	"fmt"
	"strings"

	i18n "ngc-go/packages/compiler/src/i18n"
)

// IcuSerializerVisitor is a visitor that serializes ICU nodes to strings
type IcuSerializerVisitor struct{}

// NewIcuSerializerVisitor creates a new IcuSerializerVisitor
func NewIcuSerializerVisitor() *IcuSerializerVisitor {
	return &IcuSerializerVisitor{}
}

// VisitText visits a Text node
func (v *IcuSerializerVisitor) VisitText(text *i18n.Text, context interface{}) interface{} {
	return text.Value
}

// VisitContainer visits a Container node
func (v *IcuSerializerVisitor) VisitContainer(container *i18n.Container, context interface{}) interface{} {
	parts := make([]string, 0, len(container.Children))
	for _, child := range container.Children {
		result := child.Visit(v, context)
		if str, ok := result.(string); ok {
			parts = append(parts, str)
		}
	}
	return strings.Join(parts, "")
}

// VisitIcu visits an Icu node
func (v *IcuSerializerVisitor) VisitIcu(icu *i18n.Icu, context interface{}) interface{} {
	strCases := make([]string, 0, len(icu.Cases))
	for k, caseNode := range icu.Cases {
		result := caseNode.Visit(v, context)
		var caseStr string
		if str, ok := result.(string); ok {
			caseStr = str
		}
		strCases = append(strCases, fmt.Sprintf("%s {%s}", k, caseStr))
	}
	result := fmt.Sprintf("{%s, %s, %s}", icu.ExpressionPlaceholder, icu.Type, strings.Join(strCases, " "))
	return result
}

// VisitTagPlaceholder visits a TagPlaceholder node
func (v *IcuSerializerVisitor) VisitTagPlaceholder(ph *i18n.TagPlaceholder, context interface{}) interface{} {
	if ph.IsVoid {
		return v.formatPh(ph.StartName)
	}
	parts := make([]string, 0, len(ph.Children)+2)
	parts = append(parts, v.formatPh(ph.StartName))
	for _, child := range ph.Children {
		result := child.Visit(v, context)
		if str, ok := result.(string); ok {
			parts = append(parts, str)
		}
	}
	parts = append(parts, v.formatPh(ph.CloseName))
	return strings.Join(parts, "")
}

// VisitPlaceholder visits a Placeholder node
func (v *IcuSerializerVisitor) VisitPlaceholder(ph *i18n.Placeholder, context interface{}) interface{} {
	return v.formatPh(ph.Name)
}

// VisitBlockPlaceholder visits a BlockPlaceholder node
func (v *IcuSerializerVisitor) VisitBlockPlaceholder(ph *i18n.BlockPlaceholder, context interface{}) interface{} {
	parts := make([]string, 0, len(ph.Children)+2)
	parts = append(parts, v.formatPh(ph.StartName))
	for _, child := range ph.Children {
		result := child.Visit(v, context)
		if str, ok := result.(string); ok {
			parts = append(parts, str)
		}
	}
	parts = append(parts, v.formatPh(ph.CloseName))
	return strings.Join(parts, "")
}

// VisitIcuPlaceholder visits an IcuPlaceholder node
func (v *IcuSerializerVisitor) VisitIcuPlaceholder(ph *i18n.IcuPlaceholder, context interface{}) interface{} {
	return v.formatPh(ph.Name)
}

// formatPh formats a placeholder value
func (v *IcuSerializerVisitor) formatPh(value string) string {
	return fmt.Sprintf("{%s}", FormatI18nPlaceholderName(value, false))
}

var serializer = NewIcuSerializerVisitor()

// SerializeIcuNode serializes an ICU node to a string
func SerializeIcuNode(icu *i18n.Icu) string {
	result := icu.Visit(serializer, nil)
	if str, ok := result.(string); ok {
		return str
	}
	return ""
}
