package serializers

import (
	"fmt"
	"sort"
	"strings"
)

// TagToPlaceholderNames maps HTML tags to placeholder names
var TagToPlaceholderNames = map[string]string{
	"A":     "LINK",
	"B":     "BOLD_TEXT",
	"BR":    "LINE_BREAK",
	"EM":    "EMPHASISED_TEXT",
	"H1":    "HEADING_LEVEL1",
	"H2":    "HEADING_LEVEL2",
	"H3":    "HEADING_LEVEL3",
	"H4":    "HEADING_LEVEL4",
	"H5":    "HEADING_LEVEL5",
	"H6":    "HEADING_LEVEL6",
	"HR":    "HORIZONTAL_RULE",
	"I":     "ITALIC_TEXT",
	"LI":    "LIST_ITEM",
	"LINK":  "MEDIA_LINK",
	"OL":    "ORDERED_LIST",
	"P":     "PARAGRAPH",
	"Q":     "QUOTATION",
	"S":     "STRIKETHROUGH_TEXT",
	"SMALL": "SMALL_TEXT",
	"SUB":   "SUBSTRIPT",
	"SUP":   "SUPERSCRIPT",
	"TBODY": "TABLE_BODY",
	"TD":    "TABLE_CELL",
	"TFOOT": "TABLE_FOOTER",
	"TH":    "TABLE_HEADER_CELL",
	"THEAD": "TABLE_HEADER",
	"TR":    "TABLE_ROW",
	"TT":    "MONOSPACED_TEXT",
	"U":     "UNDERLINED_TEXT",
	"UL":    "UNORDERED_LIST",
}

// PlaceholderRegistry creates unique names for placeholders with different content
type PlaceholderRegistry struct {
	placeHolderNameCounts map[string]int
	signatureToName        map[string]string
}

// NewPlaceholderRegistry creates a new PlaceholderRegistry
func NewPlaceholderRegistry() *PlaceholderRegistry {
	return &PlaceholderRegistry{
		placeHolderNameCounts: make(map[string]int),
		signatureToName:        make(map[string]string),
	}
}

// GetStartTagPlaceholderName gets a placeholder name for a start tag
func (pr *PlaceholderRegistry) GetStartTagPlaceholderName(tag string, attrs map[string]string, isVoid bool) string {
	signature := pr.hashTag(tag, attrs, isVoid)
	if name, exists := pr.signatureToName[signature]; exists {
		return name
	}
	
	upperTag := strings.ToUpper(tag)
	baseName := TagToPlaceholderNames[upperTag]
	if baseName == "" {
		baseName = "TAG_" + upperTag
	}
	
	name := pr.generateUniqueName(func() string {
		if isVoid {
			return baseName
		}
		return "START_" + baseName
	}())
	
	pr.signatureToName[signature] = name
	return name
}

// GetCloseTagPlaceholderName gets a placeholder name for a close tag
func (pr *PlaceholderRegistry) GetCloseTagPlaceholderName(tag string) string {
	signature := pr.hashClosingTag(tag)
	if name, exists := pr.signatureToName[signature]; exists {
		return name
	}
	
	upperTag := strings.ToUpper(tag)
	baseName := TagToPlaceholderNames[upperTag]
	if baseName == "" {
		baseName = "TAG_" + upperTag
	}
	
	name := pr.generateUniqueName("CLOSE_" + baseName)
	pr.signatureToName[signature] = name
	return name
}

// GetPlaceholderName gets a placeholder name for a named placeholder
func (pr *PlaceholderRegistry) GetPlaceholderName(name string, content string) string {
	upperName := strings.ToUpper(name)
	signature := "PH: " + upperName + "=" + content
	if uniqueName, exists := pr.signatureToName[signature]; exists {
		return uniqueName
	}
	
	uniqueName := pr.generateUniqueName(upperName)
	pr.signatureToName[signature] = uniqueName
	return uniqueName
}

// GetUniquePlaceholder gets a unique placeholder name
func (pr *PlaceholderRegistry) GetUniquePlaceholder(name string) string {
	return pr.generateUniqueName(strings.ToUpper(name))
}

// GetStartBlockPlaceholderName gets a placeholder name for a start block
func (pr *PlaceholderRegistry) GetStartBlockPlaceholderName(name string, parameters []string) string {
	signature := pr.hashBlock(name, parameters)
	if placeholder, exists := pr.signatureToName[signature]; exists {
		return placeholder
	}
	
	placeholder := pr.generateUniqueName("START_BLOCK_" + pr.toSnakeCase(name))
	pr.signatureToName[signature] = placeholder
	return placeholder
}

// GetCloseBlockPlaceholderName gets a placeholder name for a close block
func (pr *PlaceholderRegistry) GetCloseBlockPlaceholderName(name string) string {
	signature := pr.hashClosingBlock(name)
	if placeholder, exists := pr.signatureToName[signature]; exists {
		return placeholder
	}
	
	placeholder := pr.generateUniqueName("CLOSE_BLOCK_" + pr.toSnakeCase(name))
	pr.signatureToName[signature] = placeholder
	return placeholder
}

// hashTag generates a hash for a tag - does not take attribute order into account
func (pr *PlaceholderRegistry) hashTag(tag string, attrs map[string]string, isVoid bool) string {
	start := "<" + tag
	keys := make([]string, 0, len(attrs))
	for k := range attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	
	strAttrs := ""
	for _, name := range keys {
		strAttrs += " " + name + "=" + attrs[name]
	}
	
	end := "/>"
	if !isVoid {
		end = "></" + tag + ">"
	}
	
	return start + strAttrs + end
}

// hashClosingTag generates a hash for a closing tag
func (pr *PlaceholderRegistry) hashClosingTag(tag string) string {
	return pr.hashTag("/"+tag, map[string]string{}, false)
}

// hashBlock generates a hash for a block
func (pr *PlaceholderRegistry) hashBlock(name string, parameters []string) string {
	params := ""
	if len(parameters) > 0 {
		sortedParams := make([]string, len(parameters))
		copy(sortedParams, parameters)
		sort.Strings(sortedParams)
		params = " (" + strings.Join(sortedParams, "; ") + ")"
	}
	return "@" + name + params + " {}"
}

// hashClosingBlock generates a hash for a closing block
func (pr *PlaceholderRegistry) hashClosingBlock(name string) string {
	return pr.hashBlock("close_"+name, []string{})
}

// toSnakeCase converts a name to snake case
func (pr *PlaceholderRegistry) toSnakeCase(name string) string {
	result := ""
	for _, r := range strings.ToUpper(name) {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			result += string(r)
		} else {
			result += "_"
		}
	}
	return result
}

// generateUniqueName generates a unique name
func (pr *PlaceholderRegistry) generateUniqueName(base string) string {
	if _, seen := pr.placeHolderNameCounts[base]; !seen {
		pr.placeHolderNameCounts[base] = 1
		return base
	}
	
	id := pr.placeHolderNameCounts[base]
	pr.placeHolderNameCounts[base] = id + 1
	return base + "_" + fmt.Sprintf("%d", id)
}

