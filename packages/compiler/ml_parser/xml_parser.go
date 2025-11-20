package ml_parser

// XmlParser extends Parser for XML parsing
type XmlParser struct {
	*Parser
}

// NewXmlParser creates a new XmlParser
func NewXmlParser() *XmlParser {
	return &XmlParser{
		Parser: NewParser(GetXmlTagDefinition),
	}
}

// Parse parses XML source
func (x *XmlParser) Parse(source, url string, options *TokenizeOptions) *ParseTreeResult {
	// Blocks and let declarations aren't supported in an XML context
	if options == nil {
		options = &TokenizeOptions{}
	}
	xmlOptions := *options
	falseVal := false
	xmlOptions.TokenizeBlocks = &falseVal
	xmlOptions.TokenizeLet = &falseVal
	xmlOptions.SelectorlessEnabled = &falseVal

	return x.Parser.Parse(source, url, &xmlOptions)
}
