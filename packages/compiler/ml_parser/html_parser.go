package ml_parser

// HtmlParser extends Parser for HTML parsing
type HtmlParser struct {
	*Parser
}

// NewHtmlParser creates a new HtmlParser
func NewHtmlParser() *HtmlParser {
	return &HtmlParser{
		Parser: NewParser(GetHtmlTagDefinition),
	}
}

// Parse parses HTML source
func (h *HtmlParser) Parse(source, url string, options *TokenizeOptions) *ParseTreeResult {
	return h.Parser.Parse(source, url, options)
}
