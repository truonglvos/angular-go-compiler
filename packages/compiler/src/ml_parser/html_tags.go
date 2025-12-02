package ml_parser

import (
	"strings"
)

// HtmlTagDefinition implements TagDefinition for HTML tags
type HtmlTagDefinition struct {
	closedByChildren            map[string]bool
	contentType                 interface{} // TagContentType or map[string]TagContentType
	closedByParent              bool
	implicitNamespacePrefix     *string
	isVoid                      bool
	ignoreFirstLf               bool
	canSelfClose                bool
	preventNamespaceInheritance bool
}

// HtmlTagDefinitionOptions are options for creating an HtmlTagDefinition
type HtmlTagDefinitionOptions struct {
	ClosedByChildren            []string
	ClosedByParent              bool
	ImplicitNamespacePrefix     *string
	ContentType                 interface{} // TagContentType or map[string]TagContentType
	IsVoid                      bool
	IgnoreFirstLf               bool
	PreventNamespaceInheritance bool
	CanSelfClose                *bool
}

// NewHtmlTagDefinition creates a new HtmlTagDefinition
func NewHtmlTagDefinition(opts HtmlTagDefinitionOptions) *HtmlTagDefinition {
	closedByChildren := make(map[string]bool)
	if opts.ClosedByChildren != nil && len(opts.ClosedByChildren) > 0 {
		for _, tagName := range opts.ClosedByChildren {
			closedByChildren[tagName] = true
		}
	}

	canSelfClose := false
	if opts.CanSelfClose != nil {
		canSelfClose = *opts.CanSelfClose
	} else if opts.IsVoid {
		canSelfClose = true
	}

	contentType := opts.ContentType
	if contentType == nil {
		contentType = TagContentTypePARSABLE_DATA
	}

	return &HtmlTagDefinition{
		closedByChildren:            closedByChildren,
		contentType:                 contentType,
		closedByParent:              opts.ClosedByParent || opts.IsVoid,
		implicitNamespacePrefix:     opts.ImplicitNamespacePrefix,
		isVoid:                      opts.IsVoid,
		ignoreFirstLf:               opts.IgnoreFirstLf,
		canSelfClose:                canSelfClose,
		preventNamespaceInheritance: opts.PreventNamespaceInheritance,
	}
}

// ClosedByParent returns whether this tag is closed by parent
func (h *HtmlTagDefinition) ClosedByParent() bool {
	return h.closedByParent
}

// ImplicitNamespacePrefix returns the implicit namespace prefix
func (h *HtmlTagDefinition) ImplicitNamespacePrefix() *string {
	return h.implicitNamespacePrefix
}

// IsVoid returns whether this tag is void
func (h *HtmlTagDefinition) IsVoid() bool {
	return h.isVoid
}

// IgnoreFirstLf returns whether to ignore first line feed
func (h *HtmlTagDefinition) IgnoreFirstLf() bool {
	return h.ignoreFirstLf
}

// CanSelfClose returns whether this tag can self-close
func (h *HtmlTagDefinition) CanSelfClose() bool {
	return h.canSelfClose
}

// PreventNamespaceInheritance returns whether to prevent namespace inheritance
func (h *HtmlTagDefinition) PreventNamespaceInheritance() bool {
	return h.preventNamespaceInheritance
}

// IsClosedByChild returns whether this tag is closed by a child
func (h *HtmlTagDefinition) IsClosedByChild(name string) bool {
	return h.isVoid || h.closedByChildren[strings.ToLower(name)]
}

// GetContentType returns the content type for this tag
func (h *HtmlTagDefinition) GetContentType(prefix *string) TagContentType {
	if contentTypeMap, ok := h.contentType.(map[string]TagContentType); ok {
		if prefix != nil {
			if overrideType, exists := contentTypeMap[*prefix]; exists {
				return overrideType
			}
		}
		if defaultType, exists := contentTypeMap["default"]; exists {
			return defaultType
		}
	}
	if contentType, ok := h.contentType.(TagContentType); ok {
		return contentType
	}
	return TagContentTypePARSABLE_DATA
}

var (
	defaultTagDefinition *HtmlTagDefinition
	tagDefinitions       map[string]*HtmlTagDefinition
)

// GetHtmlTagDefinition returns the HTML tag definition for a tag name
func GetHtmlTagDefinition(tagName string) TagDefinition {
	if tagDefinitions == nil {
		initHtmlTagDefinitions()
	}

	// Case-sensitive lookup first
	if def, exists := tagDefinitions[tagName]; exists {
		return def
	}

	// Case-insensitive lookup
	if def, exists := tagDefinitions[strings.ToLower(tagName)]; exists {
		return def
	}

	return defaultTagDefinition
}

func initHtmlTagDefinitions() {
	defaultTagDefinition = NewHtmlTagDefinition(HtmlTagDefinitionOptions{
		CanSelfClose: boolPtr(true),
	})

	tagDefinitions = make(map[string]*HtmlTagDefinition)

	// Void elements
	voidTags := []string{"base", "meta", "area", "embed", "link", "img", "input", "param", "hr", "br", "source", "track", "wbr", "col"}
	for _, tag := range voidTags {
		tagDefinitions[tag] = NewHtmlTagDefinition(HtmlTagDefinitionOptions{IsVoid: true})
	}

	// Paragraph tag
	tagDefinitions["p"] = NewHtmlTagDefinition(HtmlTagDefinitionOptions{
		ClosedByChildren: []string{
			"address", "article", "aside", "blockquote", "div", "dl", "fieldset",
			"footer", "form", "h1", "h2", "h3", "h4", "h5", "h6", "header",
			"hgroup", "hr", "main", "nav", "ol", "p", "pre", "section", "table", "ul",
		},
		ClosedByParent: true,
	})

	// Table tags
	tagDefinitions["thead"] = NewHtmlTagDefinition(HtmlTagDefinitionOptions{
		ClosedByChildren: []string{"tbody", "tfoot"},
	})
	tagDefinitions["tbody"] = NewHtmlTagDefinition(HtmlTagDefinitionOptions{
		ClosedByChildren: []string{"tbody", "tfoot"},
		ClosedByParent:   true,
	})
	tagDefinitions["tfoot"] = NewHtmlTagDefinition(HtmlTagDefinitionOptions{
		ClosedByChildren: []string{"tbody"},
		ClosedByParent:   true,
	})
	tagDefinitions["tr"] = NewHtmlTagDefinition(HtmlTagDefinitionOptions{
		ClosedByChildren: []string{"tr"},
		ClosedByParent:   true,
	})
	tagDefinitions["td"] = NewHtmlTagDefinition(HtmlTagDefinitionOptions{
		ClosedByChildren: []string{"td", "th"},
		ClosedByParent:   true,
	})
	tagDefinitions["th"] = NewHtmlTagDefinition(HtmlTagDefinitionOptions{
		ClosedByChildren: []string{"td", "th"},
		ClosedByParent:   true,
	})

	// SVG and MathML
	tagDefinitions["svg"] = NewHtmlTagDefinition(HtmlTagDefinitionOptions{
		ImplicitNamespacePrefix: stringPtr("svg"),
	})
	tagDefinitions["foreignObject"] = NewHtmlTagDefinition(HtmlTagDefinitionOptions{
		ImplicitNamespacePrefix:     stringPtr("svg"),
		PreventNamespaceInheritance: true,
	})
	tagDefinitions["math"] = NewHtmlTagDefinition(HtmlTagDefinitionOptions{
		ImplicitNamespacePrefix: stringPtr("math"),
	})

	// List tags
	tagDefinitions["li"] = NewHtmlTagDefinition(HtmlTagDefinitionOptions{
		ClosedByChildren: []string{"li"},
		ClosedByParent:   true,
	})
	tagDefinitions["dt"] = NewHtmlTagDefinition(HtmlTagDefinitionOptions{
		ClosedByChildren: []string{"dt", "dd"},
	})
	tagDefinitions["dd"] = NewHtmlTagDefinition(HtmlTagDefinitionOptions{
		ClosedByChildren: []string{"dt", "dd"},
		ClosedByParent:   true,
	})

	// Ruby tags
	tagDefinitions["rb"] = NewHtmlTagDefinition(HtmlTagDefinitionOptions{
		ClosedByChildren: []string{"rb", "rt", "rtc", "rp"},
		ClosedByParent:   true,
	})
	tagDefinitions["rt"] = NewHtmlTagDefinition(HtmlTagDefinitionOptions{
		ClosedByChildren: []string{"rb", "rt", "rtc", "rp"},
		ClosedByParent:   true,
	})
	tagDefinitions["rtc"] = NewHtmlTagDefinition(HtmlTagDefinitionOptions{
		ClosedByChildren: []string{"rb", "rtc", "rp"},
		ClosedByParent:   true,
	})
	tagDefinitions["rp"] = NewHtmlTagDefinition(HtmlTagDefinitionOptions{
		ClosedByChildren: []string{"rb", "rt", "rtc", "rp"},
		ClosedByParent:   true,
	})

	// Select tags
	tagDefinitions["optgroup"] = NewHtmlTagDefinition(HtmlTagDefinitionOptions{
		ClosedByChildren: []string{"optgroup"},
		ClosedByParent:   true,
	})
	tagDefinitions["option"] = NewHtmlTagDefinition(HtmlTagDefinitionOptions{
		ClosedByChildren: []string{"option", "optgroup"},
		ClosedByParent:   true,
	})

	// Special content types
	tagDefinitions["pre"] = NewHtmlTagDefinition(HtmlTagDefinitionOptions{
		IgnoreFirstLf: true,
	})
	tagDefinitions["listing"] = NewHtmlTagDefinition(HtmlTagDefinitionOptions{
		IgnoreFirstLf: true,
	})
	tagDefinitions["style"] = NewHtmlTagDefinition(HtmlTagDefinitionOptions{
		ContentType: TagContentTypeRAW_TEXT,
	})
	tagDefinitions["script"] = NewHtmlTagDefinition(HtmlTagDefinitionOptions{
		ContentType: TagContentTypeRAW_TEXT,
	})
	tagDefinitions["title"] = NewHtmlTagDefinition(HtmlTagDefinitionOptions{
		ContentType: map[string]TagContentType{
			"default": TagContentTypeESCAPABLE_RAW_TEXT,
			"svg":     TagContentTypePARSABLE_DATA,
		},
	})
	tagDefinitions["textarea"] = NewHtmlTagDefinition(HtmlTagDefinitionOptions{
		ContentType:   TagContentTypeESCAPABLE_RAW_TEXT,
		IgnoreFirstLf: true,
	})

	// Add common HTML standard tags with canSelfClose=false
	// This matches TypeScript's behavior where DomElementSchemaRegistry adds all known HTML tags
	// with canSelfClose=false. We add the most common ones here.
	commonHtmlTags := []string{"a", "abbr", "address", "article", "aside", "b", "bdi", "bdo", "blockquote",
		"body", "button", "canvas", "caption", "cite", "code", "colgroup", "data", "datalist", "dd", "del",
		"details", "dfn", "dialog", "div", "dl", "dt", "em", "fieldset", "figcaption", "figure", "footer",
		"form", "h1", "h2", "h3", "h4", "h5", "h6", "head", "header", "hgroup", "html", "i", "iframe",
		"ins", "kbd", "label", "legend", "main", "map", "mark", "menu", "meter", "nav", "noscript",
		"object", "ol", "output", "progress", "q", "s", "samp", "section", "small", "span", "strong",
		"sub", "summary", "sup", "table", "tbody", "tfoot", "thead", "time", "u", "ul", "var", "video"}
	for _, tag := range commonHtmlTags {
		if _, exists := tagDefinitions[tag]; !exists {
			tagDefinitions[tag] = NewHtmlTagDefinition(HtmlTagDefinitionOptions{
				CanSelfClose: boolPtr(false),
			})
		}
	}
}

func boolPtr(b bool) *bool {
	return &b
}

func stringPtr(s string) *string {
	return &s
}
