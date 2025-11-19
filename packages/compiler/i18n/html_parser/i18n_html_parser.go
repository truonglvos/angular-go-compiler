package i18n_html_parser

import (
	"strings"

	"ngc-go/packages/compiler/core"
	"ngc-go/packages/compiler/i18n"
	i18n_extractor_merger "ngc-go/packages/compiler/i18n/extractor_merger"
	"ngc-go/packages/compiler/i18n/serializers"
	i18n_translation_bundle "ngc-go/packages/compiler/i18n/translation_bundle"
	"ngc-go/packages/compiler/ml_parser"
	"ngc-go/packages/compiler/util"
)

// I18NHtmlParser implements HtmlParser with i18n support
type I18NHtmlParser struct {
	htmlParser        ml_parser.HtmlParser
	translationBundle *i18n_translation_bundle.TranslationBundle
}

// NewI18NHtmlParser creates a new I18NHtmlParser
func NewI18NHtmlParser(
	htmlParser ml_parser.HtmlParser,
	translations *string,
	translationsFormat *string,
	missingTranslation core.MissingTranslationStrategy,
	console util.Console,
) *I18NHtmlParser {
	parser := &I18NHtmlParser{
		htmlParser: htmlParser,
	}

	if translations != nil && *translations != "" {
		serializer := CreateSerializer(translationsFormat)
		parser.translationBundle = i18n_translation_bundle.LoadTranslationBundle(
			*translations,
			"i18n",
			serializer,
			missingTranslation,
			console,
		)
	} else {
		parser.translationBundle = i18n_translation_bundle.NewTranslationBundle(
			map[string][]i18n.Node{},
			nil,
			i18n.Digest,
			nil,
			missingTranslation,
			console,
		)
	}

	return parser
}

// Parse parses HTML with i18n support
func (p *I18NHtmlParser) Parse(source string, url string, options *ml_parser.TokenizeOptions) *ml_parser.ParseTreeResult {
	parseResult := p.htmlParser.Parse(source, url, options)

	if len(parseResult.Errors) > 0 {
		return ml_parser.NewParseTreeResult(parseResult.RootNodes, parseResult.Errors)
	}

	return i18n_extractor_merger.MergeTranslations(parseResult.RootNodes, p.translationBundle, []string{}, map[string][]string{})
}

// CreateSerializer creates a serializer based on format
func CreateSerializer(format *string) serializers.Serializer {
	formatStr := "xlf"
	if format != nil {
		formatStr = *format
	}

	formatStr = strings.ToLower(formatStr)

	switch formatStr {
	case "xmb":
		return serializers.NewXmb()
	case "xtb":
		return serializers.NewXtb()
	case "xliff2", "xlf2":
		return serializers.NewXliff2()
	case "xliff", "xlf":
		fallthrough
	default:
		return serializers.NewXliff()
	}
}
