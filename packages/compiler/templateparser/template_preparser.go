package templateparser

import (
	"strings"

	"ngc-go/packages/compiler/ml_parser"
)

const NG_CONTENT_SELECT_ATTR = "select"
const LINK_ELEMENT = "link"
const LINK_STYLE_REL_ATTR = "rel"
const LINK_STYLE_HREF_ATTR = "href"
const LINK_STYLE_REL_VALUE = "stylesheet"
const STYLE_ELEMENT = "style"
const SCRIPT_ELEMENT = "script"
const NG_NON_BINDABLE_ATTR = "ngNonBindable"
const NG_PROJECT_AS = "ngProjectAs"

// PreparsedElementType represents the type of a preparsed element
type PreparsedElementType int

const (
	PreparsedElementTypeNgContent PreparsedElementType = iota
	PreparsedElementTypeStyle
	PreparsedElementTypeStylesheet
	PreparsedElementTypeScript
	PreparsedElementTypeOther
)

// PreparsedElement represents a preparsed element
type PreparsedElement struct {
	Type        PreparsedElementType
	SelectAttr  string
	HrefAttr    *string
	NonBindable bool
	ProjectAs   string
}

// PreparseElement preparses an element to extract special attributes and determine its type
func PreparseElement(ast *ml_parser.Element) *PreparsedElement {
	var selectAttr *string
	var hrefAttr *string
	var relAttr *string
	nonBindable := false
	projectAs := ""

	for _, attr := range ast.Attrs {
		lcAttrName := strings.ToLower(attr.Name)
		if lcAttrName == NG_CONTENT_SELECT_ATTR {
			selectAttr = &attr.Value
		} else if lcAttrName == LINK_STYLE_HREF_ATTR {
			hrefAttr = &attr.Value
		} else if lcAttrName == LINK_STYLE_REL_ATTR {
			relAttr = &attr.Value
		} else if attr.Name == NG_NON_BINDABLE_ATTR {
			nonBindable = true
		} else if attr.Name == NG_PROJECT_AS {
			if len(attr.Value) > 0 {
				projectAs = attr.Value
			}
		}
	}

	normalizedSelectAttr := normalizeNgContentSelect(selectAttr)
	nodeName := strings.ToLower(ast.Name)

	var elementType PreparsedElementType = PreparsedElementTypeOther
	if ml_parser.IsNgContent(nodeName) {
		elementType = PreparsedElementTypeNgContent
	} else if nodeName == STYLE_ELEMENT {
		elementType = PreparsedElementTypeStyle
	} else if nodeName == SCRIPT_ELEMENT {
		elementType = PreparsedElementTypeScript
	} else if nodeName == LINK_ELEMENT && relAttr != nil && *relAttr == LINK_STYLE_REL_VALUE {
		elementType = PreparsedElementTypeStylesheet
	}

	return &PreparsedElement{
		Type:        elementType,
		SelectAttr:  normalizedSelectAttr,
		HrefAttr:    hrefAttr,
		NonBindable: nonBindable,
		ProjectAs:   projectAs,
	}
}

// normalizeNgContentSelect normalizes the ng-content select attribute
func normalizeNgContentSelect(selectAttr *string) string {
	if selectAttr == nil || len(*selectAttr) == 0 {
		return "*"
	}
	return *selectAttr
}
