package viewi18n

import (
	"regexp"
	"strings"

	"ngc-go/packages/compiler/src/i18n"
	"ngc-go/packages/compiler/src/ml_parser"
	"ngc-go/packages/compiler/src/output"
)

// Name of the i18n attributes
const I18N_ATTR = "i18n"
const I18N_ATTR_PREFIX = "i18n-"

// Prefix of var expressions used in ICUs
const I18N_ICU_VAR_PREFIX = "VAR_"

// IsI18nAttribute checks if an attribute name is an i18n attribute
func IsI18nAttribute(name string) bool {
	return name == I18N_ATTR || strings.HasPrefix(name, I18N_ATTR_PREFIX)
}

// HasI18nAttrs checks if a node has i18n attributes
func HasI18nAttrs(node ml_parser.Node) bool {
	var attrs []*ml_parser.Attribute
	switch n := node.(type) {
	case *ml_parser.Element:
		attrs = n.Attrs
	case *ml_parser.Component:
		attrs = n.Attrs
	default:
		return false
	}

	for _, attr := range attrs {
		if IsI18nAttribute(attr.Name) {
			return true
		}
	}
	return false
}

// IcuFromI18nMessage extracts the ICU placeholder from an i18n message
func IcuFromI18nMessage(message *i18n.Message) *i18n.IcuPlaceholder {
	if len(message.Nodes) == 0 {
		return nil
	}
	if icu, ok := message.Nodes[0].(*i18n.IcuPlaceholder); ok {
		return icu
	}
	return nil
}

// PlaceholdersToParams converts a map of placeholders to a map of literal expressions
func PlaceholdersToParams(placeholders map[string][]string) map[string]*output.LiteralExpr {
	params := make(map[string]*output.LiteralExpr)
	for key, values := range placeholders {
		var value string
		if len(values) > 1 {
			value = "[" + strings.Join(values, "|") + "]"
		} else {
			value = values[0]
		}
		params[key] = output.NewLiteralExpr(value, nil, nil)
	}
	return params
}

// FormatI18nPlaceholderNamesInMap formats the placeholder names in a map of placeholders to expressions
// The placeholder names are converted from "internal" format (e.g. `START_TAG_DIV_1`) to "external"
// format (e.g. `startTagDiv_1`).
func FormatI18nPlaceholderNamesInMap(
	params map[string]output.OutputExpression,
	useCamelCase bool,
) map[string]output.OutputExpression {
	_params := make(map[string]output.OutputExpression)
	if params != nil && len(params) > 0 {
		for key, value := range params {
			_params[FormatI18nPlaceholderName(key, useCamelCase)] = value
		}
	}
	return _params
}

// FormatI18nPlaceholderName converts internal placeholder names to public-facing format
// (for example to use in goog.getMsg call).
// Example: `START_TAG_DIV_1` is converted to `startTagDiv_1`.
func FormatI18nPlaceholderName(name string, useCamelCase bool) string {
	publicName := ToPublicName(name)
	if !useCamelCase {
		return publicName
	}
	chunks := strings.Split(publicName, "_")
	if len(chunks) == 1 {
		// if no "_" found - just lowercase the value
		return strings.ToLower(name)
	}
	var postfix string
	// eject last element if it's a number
	numberRegex := regexp.MustCompile(`^\d+$`)
	if numberRegex.MatchString(chunks[len(chunks)-1]) {
		postfix = chunks[len(chunks)-1]
		chunks = chunks[:len(chunks)-1]
	}
	raw := strings.ToLower(chunks[0])
	if len(chunks) > 1 {
		for _, c := range chunks[1:] {
			if len(c) > 0 {
				raw += strings.ToUpper(c[:1]) + strings.ToLower(c[1:])
			}
		}
	}
	if postfix != "" {
		return raw + "_" + postfix
	}
	return raw
}

// ToPublicName converts an internal name to a public name (XMB/XTB format)
// XMB/XTB placeholders can only contain A-Z, 0-9 and _
func ToPublicName(internalName string) string {
	// Convert to uppercase and replace invalid characters with _
	result := strings.ToUpper(internalName)
	invalidCharRegex := regexp.MustCompile(`[^A-Z0-9_]`)
	return invalidCharRegex.ReplaceAllString(result, "_")
}
