package i18n_translation_bundle

import (
	"fmt"
	"strings"

	"ngc-go/packages/compiler/src/core"
	"ngc-go/packages/compiler/src/i18n"
	"ngc-go/packages/compiler/src/i18n/serializers"
	"ngc-go/packages/compiler/src/ml_parser"
	"ngc-go/packages/compiler/src/util"
)

// TranslationBundle is a container for translated messages
type TranslationBundle struct {
	i18nNodesByMsgID map[string][]i18n.Node
	locale           *string
	digest           func(*i18n.Message) string
	mapperFactory    func(*i18n.Message) serializers.PlaceholderMapper
	i18nToHtml       *I18nToHtmlVisitor
}

// NewTranslationBundle creates a new TranslationBundle
func NewTranslationBundle(
	i18nNodesByMsgID map[string][]i18n.Node,
	locale *string,
	digest func(*i18n.Message) string,
	mapperFactory func(*i18n.Message) serializers.PlaceholderMapper,
	missingTranslation core.MissingTranslationStrategy,
	console util.Console,
) *TranslationBundle {
	bundle := &TranslationBundle{
		i18nNodesByMsgID: i18nNodesByMsgID,
		locale:           locale,
		digest:           digest,
		mapperFactory:    mapperFactory,
	}

	bundle.i18nToHtml = NewI18nToHtmlVisitor(
		i18nNodesByMsgID,
		locale,
		digest,
		mapperFactory,
		missingTranslation,
		console,
	)

	return bundle
}

// LoadTranslationBundle creates a TranslationBundle by parsing the given content with the serializer
func LoadTranslationBundle(
	content string,
	url string,
	serializer serializers.Serializer,
	missingTranslation core.MissingTranslationStrategy,
	console util.Console,
) *TranslationBundle {
	locale, i18nNodesByMsgID := serializer.Load(content, url)
	digestFn := func(m *i18n.Message) string {
		return serializer.Digest(m)
	}
	mapperFactory := func(m *i18n.Message) serializers.PlaceholderMapper {
		return serializer.CreateNameMapper(m)
	}
	return NewTranslationBundle(
		i18nNodesByMsgID,
		locale,
		digestFn,
		mapperFactory,
		missingTranslation,
		console,
	)
}

// Get returns the translation as HTML nodes from the given source message
func (tb *TranslationBundle) Get(srcMsg *i18n.Message) ([]ml_parser.Node, error) {
	html := tb.i18nToHtml.Convert(srcMsg)

	if len(html.Errors) > 0 {
		return nil, fmt.Errorf(strings.Join(html.Errors, "\n"))
	}

	return html.Nodes, nil
}

// Has checks if a translation exists for the given source message
func (tb *TranslationBundle) Has(srcMsg *i18n.Message) bool {
	id := tb.digest(srcMsg)
	_, exists := tb.i18nNodesByMsgID[id]
	return exists
}

// I18nToHtmlVisitor converts i18n nodes to HTML nodes
type I18nToHtmlVisitor struct {
	i18nNodesByMsgID           map[string][]i18n.Node
	locale                     *string
	digest                     func(*i18n.Message) string
	mapperFactory              func(*i18n.Message) serializers.PlaceholderMapper
	missingTranslationStrategy core.MissingTranslationStrategy
	console                    util.Console
	srcMsg                     *i18n.Message
	errors                     []string
	contextStack               []contextStackEntry
	mapper                     func(string) string
}

type contextStackEntry struct {
	msg    *i18n.Message
	mapper func(string) string
}

// NewI18nToHtmlVisitor creates a new I18nToHtmlVisitor
func NewI18nToHtmlVisitor(
	i18nNodesByMsgID map[string][]i18n.Node,
	locale *string,
	digest func(*i18n.Message) string,
	mapperFactory func(*i18n.Message) serializers.PlaceholderMapper,
	missingTranslationStrategy core.MissingTranslationStrategy,
	console util.Console,
) *I18nToHtmlVisitor {
	return &I18nToHtmlVisitor{
		i18nNodesByMsgID:           i18nNodesByMsgID,
		locale:                     locale,
		digest:                     digest,
		mapperFactory:              mapperFactory,
		missingTranslationStrategy: missingTranslationStrategy,
		console:                    console,
		errors:                     []string{},
		contextStack:               []contextStackEntry{},
	}
}

// ConvertResult represents the result of conversion
type ConvertResult struct {
	Nodes  []ml_parser.Node
	Errors []string
}

// Convert converts a source message to HTML nodes
func (v *I18nToHtmlVisitor) Convert(srcMsg *i18n.Message) *ConvertResult {
	v.contextStack = []contextStackEntry{}
	v.errors = []string{}

	// i18n to text
	text := v.convertToText(srcMsg)

	// text to html
	url := ""
	if len(srcMsg.Nodes) > 0 {
		url = srcMsg.Nodes[0].SourceSpan().Start.File.URL
	}

	htmlParser := ml_parser.NewHtmlParser()
	TokenizeExpansionForms := true
	options := &ml_parser.TokenizeOptions{
		TokenizeExpansionForms: &TokenizeExpansionForms,
	}
	html := htmlParser.Parse(text, url, options)

	// Convert ParseError to string
	htmlErrorStrings := make([]string, len(html.Errors))
	for i, err := range html.Errors {
		htmlErrorStrings[i] = err.Error()
	}

	return &ConvertResult{
		Nodes:  html.RootNodes,
		Errors: append(v.errors, htmlErrorStrings...),
	}
}

// VisitText visits a Text node
func (v *I18nToHtmlVisitor) VisitText(text *i18n.Text, context interface{}) interface{} {
	// `convert()` uses an `HtmlParser` to return `html.Node`s
	// we should then make sure that any special characters are escaped
	return serializers.EscapeXml(text.Value)
}

// VisitContainer visits a Container node
func (v *I18nToHtmlVisitor) VisitContainer(container *i18n.Container, context interface{}) interface{} {
	parts := make([]string, len(container.Children))
	for i, child := range container.Children {
		result := child.Visit(v, nil)
		if str, ok := result.(string); ok {
			parts[i] = str
		} else {
			parts[i] = ""
		}
	}
	return strings.Join(parts, "")
}

// VisitIcu visits an Icu node
func (v *I18nToHtmlVisitor) VisitIcu(icu *i18n.Icu, context interface{}) interface{} {
	cases := make([]string, 0, len(icu.Cases))
	for k, node := range icu.Cases {
		result := node.Visit(v, nil)
		if str, ok := result.(string); ok {
			cases = append(cases, k+" {"+str+"}")
		}
	}

	// TODO: Once all format switch to using expression placeholders
	// we should throw when the placeholder is not in the source message
	exp := icu.Expression
	if placeholder, exists := v.srcMsg.Placeholders[icu.Expression]; exists {
		exp = placeholder.Text
	}

	return "{" + exp + ", " + icu.Type + ", " + strings.Join(cases, " ") + "}"
}

// VisitPlaceholder visits a Placeholder node
func (v *I18nToHtmlVisitor) VisitPlaceholder(ph *i18n.Placeholder, context interface{}) interface{} {
	phName := v.mapper(ph.Name)
	if placeholder, exists := v.srcMsg.Placeholders[phName]; exists {
		return placeholder.Text
	}

	if msg, exists := v.srcMsg.PlaceholderToMessage[phName]; exists {
		return v.convertToText(msg)
	}

	v.addError(ph, `Unknown placeholder "`+ph.Name+`"`)
	return ""
}

// VisitTagPlaceholder visits a TagPlaceholder node
func (v *I18nToHtmlVisitor) VisitTagPlaceholder(ph *i18n.TagPlaceholder, context interface{}) interface{} {
	tag := ph.Tag
	attrs := make([]string, 0, len(ph.Attrs))
	for name, value := range ph.Attrs {
		attrs = append(attrs, name+`="`+value+`"`)
	}
	attrsStr := strings.Join(attrs, " ")
	if attrsStr != "" {
		attrsStr = " " + attrsStr
	}

	if ph.IsVoid {
		return "<" + tag + attrsStr + "/>"
	}

	parts := make([]string, len(ph.Children))
	for i, child := range ph.Children {
		result := child.Visit(v, nil)
		if str, ok := result.(string); ok {
			parts[i] = str
		} else {
			parts[i] = ""
		}
	}
	children := strings.Join(parts, "")
	return "<" + tag + attrsStr + ">" + children + "</" + tag + ">"
}

// VisitIcuPlaceholder visits an IcuPlaceholder node
func (v *I18nToHtmlVisitor) VisitIcuPlaceholder(ph *i18n.IcuPlaceholder, context interface{}) interface{} {
	// An ICU placeholder references the source message to be serialized
	if msg, exists := v.srcMsg.PlaceholderToMessage[ph.Name]; exists {
		return v.convertToText(msg)
	}
	return ""
}

// VisitBlockPlaceholder visits a BlockPlaceholder node
func (v *I18nToHtmlVisitor) VisitBlockPlaceholder(ph *i18n.BlockPlaceholder, context interface{}) interface{} {
	params := ""
	if len(ph.Parameters) > 0 {
		params = " (" + strings.Join(ph.Parameters, "; ") + ")"
	}

	parts := make([]string, len(ph.Children))
	for i, child := range ph.Children {
		result := child.Visit(v, nil)
		if str, ok := result.(string); ok {
			parts[i] = str
		} else {
			parts[i] = ""
		}
	}
	children := strings.Join(parts, "")
	return "@" + ph.Name + params + " {" + children + "}"
}

// convertToText converts a source message to a translated text string
func (v *I18nToHtmlVisitor) convertToText(srcMsg *i18n.Message) string {
	id := v.digest(srcMsg)
	mapper := v.mapperFactory(srcMsg)
	var nodes []i18n.Node

	v.contextStack = append(v.contextStack, contextStackEntry{
		msg:    v.srcMsg,
		mapper: v.mapper,
	})
	v.srcMsg = srcMsg

	if _, exists := v.i18nNodesByMsgID[id]; exists {
		// When there is a translation use its nodes as the source
		// And create a mapper to convert serialized placeholder names to internal names
		nodes = v.i18nNodesByMsgID[id]
		if mapper != nil {
			v.mapper = func(name string) string {
				if internalName := mapper.ToInternalName(name); internalName != nil {
					return *internalName
				}
				return name
			}
		} else {
			v.mapper = func(name string) string {
				return name
			}
		}
	} else {
		// When no translation has been found
		// - report an error / a warning / nothing,
		// - use the nodes from the original message
		// - placeholders are already internal and need no mapper
		if v.missingTranslationStrategy == core.MissingTranslationStrategyError {
			ctx := ""
			if v.locale != nil {
				ctx = ` for locale "` + *v.locale + `"`
			}
			v.addError(srcMsg.Nodes[0], `Missing translation for message "`+id+`"`+ctx)
		} else if v.console != nil && v.missingTranslationStrategy == core.MissingTranslationStrategyWarning {
			ctx := ""
			if v.locale != nil {
				ctx = ` for locale "` + *v.locale + `"`
			}
			v.console.Warn(`Missing translation for message "` + id + `"` + ctx)
		}
		nodes = srcMsg.Nodes
		v.mapper = func(name string) string {
			return name
		}
	}

	parts := make([]string, len(nodes))
	for i, node := range nodes {
		result := node.Visit(v, nil)
		if str, ok := result.(string); ok {
			parts[i] = str
		} else {
			parts[i] = ""
		}
	}
	text := strings.Join(parts, "")

	// Restore context
	if len(v.contextStack) > 0 {
		context := v.contextStack[len(v.contextStack)-1]
		v.contextStack = v.contextStack[:len(v.contextStack)-1]
		v.srcMsg = context.msg
		v.mapper = context.mapper
	}

	return text
}

// addError adds an error
func (v *I18nToHtmlVisitor) addError(node i18n.Node, msg string) {
	span := node.SourceSpan()
	errorMsg := fmt.Sprintf("%s: %s", span.String(), msg)
	v.errors = append(v.errors, errorMsg)
}
