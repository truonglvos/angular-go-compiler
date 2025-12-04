package view

import (
	"fmt"
	"ngc-go/packages/compiler/src/expression_parser"
	"ngc-go/packages/compiler/src/ml_parser"
	"ngc-go/packages/compiler/src/render3"
	viewi18n "ngc-go/packages/compiler/src/render3/view/i18n"
	"ngc-go/packages/compiler/src/schema"
	"ngc-go/packages/compiler/src/template_parser"
	"ngc-go/packages/compiler/src/util"
)

// LEADING_TRIVIA_CHARS are characters that should be considered as leading trivia
var LEADING_TRIVIA_CHARS = []string{" ", "\n", "\r", "\t"}

// ParseTemplateOptions are options that can be used to modify how a template is parsed by `parseTemplate()`.
type ParseTemplateOptions struct {
	// PreserveWhitespaces includes whitespace nodes in the parsed output.
	PreserveWhitespaces *bool

	// PreserveLineEndings preserves original line endings instead of normalizing '\r\n' endings to '\n'.
	PreserveLineEndings *bool

	// PreserveSignificantWhitespace preserves whitespace significant to rendering.
	PreserveSignificantWhitespace *bool

	// Range is the start and end point of the text to parse within the `source` string.
	// The entire `source` string is parsed if this is not provided.
	Range *ml_parser.LexerRange

	// EscapedString indicates if this text is stored in a JavaScript string, then we have to deal with escape sequences.
	EscapedString *bool

	// LeadingTriviaChars is an array of characters that should be considered as leading trivia.
	// Leading trivia are characters that are not important to the developer, and so should not be
	// included in source-map segments.  A common example is whitespace.
	LeadingTriviaChars []string

	// EnableI18nLegacyMessageIdFormat renders `$localize` message ids with additional legacy message ids.
	//
	// This option defaults to `true` but in the future the default will be flipped.
	//
	// For now set this option to false if you have migrated the translation files to use the new
	// `$localize` message id format and you are not using compile time translation merging.
	EnableI18nLegacyMessageIdFormat *bool

	// I18nNormalizeLineEndingsInICUs indicates if this text is stored in an external template (e.g. via `templateUrl`) then we need to decide
	// whether or not to normalize the line-endings (from `\r\n` to `\n`) when processing ICU
	// expressions.
	//
	// If `true` then we will normalize ICU expression line endings.
	// The default is `false`, but this will be switched in a future major release.
	I18nNormalizeLineEndingsInICUs *bool

	// AlwaysAttemptHtmlToR3AstConversion indicates whether to always attempt to convert the parsed HTML AST to an R3 AST, despite HTML or i18n
	// Meta parse errors.
	//
	// This option is useful in the context of the language service, where we want to get as much
	// information as possible, despite any errors in the HTML. As an example, a user may be adding
	// a new tag and expecting autocomplete on that tag. In this scenario, the HTML is in an errored
	// state, as there is an incomplete open tag. However, we're still able to convert the HTML AST
	// nodes to R3 AST nodes in order to provide information for the language service.
	//
	// Note that even when `true` the HTML parse and i18n errors are still appended to the errors
	// output, but this is done after converting the HTML AST to R3 AST.
	AlwaysAttemptHtmlToR3AstConversion *bool

	// CollectCommentNodes includes HTML Comment nodes in a top-level comments array on the returned R3 AST.
	//
	// This option is required by tooling that needs to know the location of comment nodes within the
	// AST. A concrete example is @angular-eslint which requires this in order to enable
	// "eslint-disable" comments within HTML templates, which then allows users to turn off specific
	// rules on a case by case basis, instead of for their whole project within a configuration file.
	CollectCommentNodes *bool

	// EnableBlockSyntax indicates whether the @ block syntax is enabled.
	EnableBlockSyntax *bool

	// EnableLetSyntax indicates whether the `@let` syntax is enabled.
	EnableLetSyntax *bool

	// EnableSelectorless indicates whether the selectorless syntax is enabled.
	EnableSelectorless *bool
}

// ParsedTemplate contains information about the template which was extracted during parsing.
//
// This contains the actual parsed template as well as any metadata collected during its parsing,
// some of which might be useful for re-parsing the template with different options.
type ParsedTemplate struct {
	// PreserveWhitespaces includes whitespace nodes in the parsed output.
	PreserveWhitespaces *bool

	// Errors are any errors from parsing the template the first time.
	//
	// `nil` if there are no errors. Otherwise, the array of errors is guaranteed to be non-empty.
	Errors []*util.ParseError

	// Nodes are the template AST, parsed from the template.
	Nodes []render3.Node

	// StyleUrls are any styleUrls extracted from the metadata.
	StyleUrls []string

	// Styles are any inline styles extracted from the metadata.
	Styles []string

	// NgContentSelectors are any ng-content selectors extracted from the template.
	NgContentSelectors []string

	// CommentNodes are any R3 Comment Nodes extracted from the template when the `collectCommentNodes` parse template
	// option is enabled.
	CommentNodes []*render3.Comment
}

// ParseTemplate parses a template into render3 `Node`s and additional metadata, with no other dependencies.
//
// template: text of the template to parse
// templateUrl: URL to use for source mapping of the parsed template
// options: options to modify how the template is parsed
func ParseTemplate(
	template string,
	templateUrl string,
	options *ParseTemplateOptions,
) *ParsedTemplate {
	if options == nil {
		options = &ParseTemplateOptions{}
	}

	preserveWhitespaces := options.PreserveWhitespaces
	enableI18nLegacyMessageIdFormat := options.EnableI18nLegacyMessageIdFormat
	selectorlessEnabled := false
	if options.EnableSelectorless != nil {
		selectorlessEnabled = *options.EnableSelectorless
	}

	bindingParser := MakeBindingParser(selectorlessEnabled)
	htmlParser := ml_parser.NewHtmlParser()

	// Build TokenizeOptions from ParseTemplateOptions
	tokenizeOptions := &ml_parser.TokenizeOptions{
		LeadingTriviaChars:             LEADING_TRIVIA_CHARS,
		TokenizeExpansionForms:         boolPtr(true),
		Range:                          options.Range,
		EscapedString:                  options.EscapedString,
		I18nNormalizeLineEndingsInICUs: options.I18nNormalizeLineEndingsInICUs,
		PreserveLineEndings:            options.PreserveLineEndings,
		SelectorlessEnabled:            &selectorlessEnabled,
	}

	enableBlockSyntax := true
	if options.EnableBlockSyntax != nil {
		enableBlockSyntax = *options.EnableBlockSyntax
	}
	tokenizeOptions.TokenizeBlocks = &enableBlockSyntax

	enableLetSyntax := true
	if options.EnableLetSyntax != nil {
		enableLetSyntax = *options.EnableLetSyntax
	}
	tokenizeOptions.TokenizeLet = &enableLetSyntax

	// Merge additional options
	if options.LeadingTriviaChars != nil {
		tokenizeOptions.LeadingTriviaChars = options.LeadingTriviaChars
	}

	parseResult := htmlParser.Parse(template, templateUrl, tokenizeOptions)

	if options.AlwaysAttemptHtmlToR3AstConversion == nil || !*options.AlwaysAttemptHtmlToR3AstConversion {
		if parseResult.Errors != nil && len(parseResult.Errors) > 0 {
			parsedTemplate := &ParsedTemplate{
				PreserveWhitespaces: preserveWhitespaces,
				Errors:              parseResult.Errors,
				Nodes:               []render3.Node{},
				StyleUrls:           []string{},
				Styles:              []string{},
				NgContentSelectors:  []string{},
			}
			if options.CollectCommentNodes != nil && *options.CollectCommentNodes {
				parsedTemplate.CommentNodes = []*render3.Comment{}
			}
			return parsedTemplate
		}
	}

	rootNodes := parseResult.RootNodes
	fmt.Printf("ParseTemplate: htmlParser returned %d nodes\n", len(rootNodes))

	// We need to use the same `retainEmptyTokens` value for both parses to avoid
	// causing a mismatch when reusing source spans, even if the
	// `preserveSignificantWhitespace` behavior is different between the two
	// parses.
	preserveSignificantWhitespace := true
	if options.PreserveSignificantWhitespace != nil {
		preserveSignificantWhitespace = *options.PreserveSignificantWhitespace
	}
	_ = !preserveSignificantWhitespace // retainEmptyTokens - used in i18n visitor but not directly here

	// process i18n meta information (scan attributes, generate ids)
	// before we run whitespace removal process, because existing i18n
	// extraction process (ng extract-i18n) relies on a raw content to generate
	// message ids
	// keepI18nAttrs = !preserveWhitespaces
	keepI18nAttrs := true
	if preserveWhitespaces != nil && *preserveWhitespaces {
		keepI18nAttrs = false
	}

	enableI18nLegacy := enableI18nLegacyMessageIdFormat
	if enableI18nLegacy == nil {
		enableI18nLegacy = boolPtr(true) // default to true
	}

	i18nMetaVisitor := viewi18n.NewI18nMetaVisitor(
		keepI18nAttrs,
		*enableI18nLegacy,
		nil, // containerBlocks
		preserveSignificantWhitespace,
	)
	i18nMetaResult := i18nMetaVisitor.VisitAllWithErrors(rootNodes)

	if options.AlwaysAttemptHtmlToR3AstConversion == nil || !*options.AlwaysAttemptHtmlToR3AstConversion {
		if i18nMetaResult.Errors != nil && len(i18nMetaResult.Errors) > 0 {
			parsedTemplate := &ParsedTemplate{
				PreserveWhitespaces: preserveWhitespaces,
				Errors:              i18nMetaResult.Errors,
				Nodes:               []render3.Node{},
				StyleUrls:           []string{},
				Styles:              []string{},
				NgContentSelectors:  []string{},
			}
			if options.CollectCommentNodes != nil && *options.CollectCommentNodes {
				parsedTemplate.CommentNodes = []*render3.Comment{}
			}
			return parsedTemplate
		}
	}

	rootNodes = i18nMetaResult.RootNodes
	fmt.Printf("ParseTemplate: after i18nMetaVisitor, nodes count: %d\n", len(rootNodes))

	if preserveWhitespaces == nil || !*preserveWhitespaces {
		// Always preserve significant whitespace here because this is used to generate the `goog.getMsg`
		// and `$localize` calls which should retain significant whitespace in order to render the
		// correct output. We let this diverge from the message IDs generated earlier which might not
		// have preserved significant whitespace.
		//
		// This should use `visitAllWithSiblings` to set `WhitespaceVisitor` context correctly, however
		// there is an existing bug where significant whitespace is not properly retained in the JS
		// output of leading/trailing whitespace for ICU messages due to the existing lack of context
		// in `WhitespaceVisitor`. Using `visitAllWithSiblings` here would fix that bug and retain the
		// whitespace, however it would also change the runtime representation which we don't want to do
		// right now.
		whitespaceVisitor := ml_parser.NewWhitespaceVisitor(
			true,  // preserveSignificantWhitespace
			nil,   // originalNodeMap
			false, // requireContext
		)
		visitedNodes := ml_parser.VisitAll(whitespaceVisitor, rootNodes, nil)
		rootNodes = convertToMlNodes(visitedNodes)
		fmt.Printf("ParseTemplate: after whitespaceVisitor, nodes count: %d\n", len(rootNodes))

		// run i18n meta visitor again in case whitespaces are removed (because that might affect
		// generated i18n message content) and first pass indicated that i18n content is present in a
		// template. During this pass i18n IDs generated at the first pass will be preserved, so we can
		// mimic existing extraction process (ng extract-i18n)
		if i18nMetaVisitor.HasI18nMeta {
			// Note: In TypeScript, enableI18nLegacyMessageIdFormat is passed as undefined,
			// but Go doesn't have undefined, so we pass false
			enableI18nLegacy2 := false
			i18nMetaVisitor2 := viewi18n.NewI18nMetaVisitor(
				false,             // keepI18nAttrs
				enableI18nLegacy2, // enableI18nLegacyMessageIdFormat (undefined in TS, so false in Go)
				nil,               // containerBlocks
				true,              // preserveSignificantWhitespace
			)
			visitedNodes2 := ml_parser.VisitAll(i18nMetaVisitor2, rootNodes, nil)
			rootNodes = convertToMlNodes(visitedNodes2)
			fmt.Printf("ParseTemplate: after i18nMetaVisitor2, nodes count: %d\n", len(rootNodes))
		}
	}

	// Convert HTML AST to R3 AST
	collectCommentNodes := false
	if options.CollectCommentNodes != nil {
		collectCommentNodes = *options.CollectCommentNodes
	}
	parseResult2 := HtmlAstToRender3Ast(
		rootNodes,
		bindingParser,
		Render3ParseOptions{
			CollectCommentNodes: collectCommentNodes,
			SelectorlessEnabled: selectorlessEnabled,
		},
	)

	// Combine all errors
	allErrors := []*util.ParseError{}
	if parseResult2.Errors != nil {
		allErrors = append(allErrors, parseResult2.Errors...)
	}
	if parseResult.Errors != nil {
		allErrors = append(allErrors, parseResult.Errors...)
	}
	if i18nMetaResult.Errors != nil {
		allErrors = append(allErrors, i18nMetaResult.Errors...)
	}

	// errors: errors.length > 0 ? errors : null
	var finalErrors []*util.ParseError
	if len(allErrors) > 0 {
		finalErrors = allErrors
	}

	parsedTemplate := &ParsedTemplate{
		PreserveWhitespaces: preserveWhitespaces,
		Errors:              finalErrors,
		Nodes:               parseResult2.Nodes,
		StyleUrls:           parseResult2.StyleUrls,
		Styles:              parseResult2.Styles,
		NgContentSelectors:  parseResult2.NgContentSelectors,
	}

	if options.CollectCommentNodes != nil && *options.CollectCommentNodes {
		parsedTemplate.CommentNodes = parseResult2.CommentNodes
	}

	return parsedTemplate
}

// elementRegistry is a global DomElementSchemaRegistry instance
var elementRegistry = schema.NewDomElementSchemaRegistry()

// MakeBindingParser constructs a `BindingParser` with a default configuration.
func MakeBindingParser(selectorlessEnabled bool) *template_parser.BindingParser {
	lexer := expression_parser.NewLexer()
	parser := expression_parser.NewParser(lexer, selectorlessEnabled)
	return template_parser.NewBindingParser(parser, elementRegistry, []*util.ParseError{})
}

// Render3ParseResult represents the result of the html AST to Ivy AST transformation
type Render3ParseResult struct {
	Nodes              []render3.Node
	Errors             []*util.ParseError
	Styles             []string
	StyleUrls          []string
	NgContentSelectors []string
	CommentNodes       []*render3.Comment
}

// convertToMlNodes converts a slice of interface{} to []ml_parser.Node
func convertToMlNodes(visitedNodes []interface{}) []ml_parser.Node {
	result := make([]ml_parser.Node, 0, len(visitedNodes))
	for _, node := range visitedNodes {
		if mlNode, ok := node.(ml_parser.Node); ok {
			result = append(result, mlNode)
		}
	}
	return result
}

// boolPtr returns a pointer to a bool value
func boolPtr(b bool) *bool {
	return &b
}
