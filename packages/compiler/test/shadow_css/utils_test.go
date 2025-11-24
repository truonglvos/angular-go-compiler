package shadow_css_test

import (
	"ngc-go/packages/compiler/src/css"
	"strings"
)

// shim shims CSS text with the given selector
func shim(cssText string, contentAttr string, hostAttr ...string) string {
	shadowCss := css.NewShadowCss()
	host := ""
	if len(hostAttr) > 0 {
		host = hostAttr[0]
	}
	return shadowCss.ShimCssText(cssText, contentAttr, host)
}

// extractCssContent extracts CSS content for comparison
// It normalizes whitespace and removes extra spacing
func extractCssContent(css string) string {
	// Remove leading newline and spaces
	css = strings.TrimLeft(css, "\n\t ")
	// Remove trailing newline and spaces
	css = strings.TrimRight(css, "\n\t ")
	// Replace all whitespace sequences with single space
	re := strings.NewReplacer(
		"\n", " ",
		"\t", " ",
		"\r", " ",
	)
	css = re.Replace(css)
	// Collapse multiple spaces to single space
	for strings.Contains(css, "  ") {
		css = strings.ReplaceAll(css, "  ", " ")
	}
	// Remove space after colon
	css = strings.ReplaceAll(css, ": ", ":")
	// Remove space before }
	css = strings.ReplaceAll(css, " }", "}")
	return css
}

// equalCss compares two CSS strings after normalization
func equalCss(actual string, expected string) bool {
	return extractCssContent(actual) == extractCssContent(expected)
}

