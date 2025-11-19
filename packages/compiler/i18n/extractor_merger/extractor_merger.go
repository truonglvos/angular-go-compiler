package i18n_extractor_merger

import (
	"ngc-go/packages/compiler/i18n"
	i18n_translation_bundle "ngc-go/packages/compiler/i18n/translation_bundle"
	"ngc-go/packages/compiler/ml_parser"
	"ngc-go/packages/compiler/util"
)

// ExtractMessages extracts translatable messages from an HTML AST
func ExtractMessages(
	nodes []ml_parser.Node,
	implicitTags []string,
	implicitAttrs map[string][]string,
	preserveSignificantWhitespace bool,
) *ExtractionResult {
	visitor := NewVisitor(implicitTags, implicitAttrs, preserveSignificantWhitespace)
	return visitor.Extract(nodes)
}

// MergeTranslations merges translations into HTML nodes
func MergeTranslations(
	nodes []ml_parser.Node,
	translations *i18n_translation_bundle.TranslationBundle,
	implicitTags []string,
	implicitAttrs map[string][]string,
) *ml_parser.ParseTreeResult {
	visitor := NewVisitor(implicitTags, implicitAttrs, true)
	return visitor.Merge(nodes, translations)
}

// ExtractionResult represents the result of message extraction
type ExtractionResult struct {
	Messages []*i18n.Message
	Errors   []*util.ParseError
}

// VisitorMode represents the mode of the visitor
type VisitorMode int

const (
	VisitorModeExtract VisitorMode = iota
	VisitorModeMerge
)

// Visitor is used to extract translatable strings and merge translations
type Visitor struct {
	implicitTags                  []string
	implicitAttrs                 map[string][]string
	preserveSignificantWhitespace bool
	mode                          VisitorMode
	messages                      []*i18n.Message
	errors                        []*util.ParseError
	translations                  *i18n_translation_bundle.TranslationBundle
	createI18nMessage             I18nMessageFactory
	depth                         int
	inI18nNode                    bool
	inImplicitNode                bool
	inI18nBlock                   bool
	blockMeaningAndDesc           string
	blockChildren                 []ml_parser.Node
	blockStartDepth               int
	inIcu                         bool
	msgCountAtSectionStart        *int
}

// NewVisitor creates a new Visitor
func NewVisitor(
	implicitTags []string,
	implicitAttrs map[string][]string,
	preserveSignificantWhitespace bool,
) *Visitor {
	return &Visitor{
		implicitTags:                  implicitTags,
		implicitAttrs:                 implicitAttrs,
		preserveSignificantWhitespace: preserveSignificantWhitespace,
		messages:                      []*i18n.Message{},
		errors:                        []*util.ParseError{},
	}
}

// Extract extracts messages from the tree
func (v *Visitor) Extract(nodes []ml_parser.Node) *ExtractionResult {
	v.init(VisitorModeExtract)

	// TODO: Implement full extraction logic
	// This is a placeholder - the full implementation would visit all nodes
	// and extract i18n messages based on i18n attributes and comments

	return &ExtractionResult{
		Messages: v.messages,
		Errors:   v.errors,
	}
}

// Merge returns a tree where all translatable nodes are translated
func (v *Visitor) Merge(nodes []ml_parser.Node, translations *i18n_translation_bundle.TranslationBundle) *ml_parser.ParseTreeResult {
	v.init(VisitorModeMerge)
	v.translations = translations

	// TODO: Implement full merge logic
	// This is a placeholder - the full implementation would visit all nodes
	// and replace translatable content with translations

	return ml_parser.NewParseTreeResult(nodes, v.errors)
}

// init initializes the visitor
func (v *Visitor) init(mode VisitorMode) {
	v.mode = mode
	v.depth = 0
	v.inI18nNode = false
	v.inImplicitNode = false
	v.inI18nBlock = false
	v.blockMeaningAndDesc = ""
	v.blockChildren = []ml_parser.Node{}
	v.blockStartDepth = 0
	v.inIcu = false
	v.msgCountAtSectionStart = nil
	v.messages = []*i18n.Message{}
	v.errors = []*util.ParseError{}
}

// I18nMessageFactory is a function type that creates i18n messages
type I18nMessageFactory func(
	nodes []ml_parser.Node,
	meaning *string,
	description *string,
	customID *string,
	visitNodeFn func(html ml_parser.Node, i18n i18n.Node) i18n.Node,
) *i18n.Message
