package schema

import (
	"ngc-go/packages/compiler/src/core"
	"strings"
)

var (
	securitySchema map[string]core.SecurityContext
)

// SecuritySchema returns the security schema map
// This function initializes the schema on first call
func SecuritySchema() map[string]core.SecurityContext {
	if securitySchema == nil {
		securitySchema = make(map[string]core.SecurityContext)
		// Case is insignificant below, all element and attribute names are lower-cased for lookup.

		registerContext(core.SecurityContextHTML, []string{
			"iframe|srcdoc",
			"*|innerHTML",
			"*|outerHTML",
		})
		registerContext(core.SecurityContextSTYLE, []string{
			"*|style",
		})
		// NB: no SCRIPT contexts here, they are never allowed due to the parser stripping them.
		registerContext(core.SecurityContextURL, []string{
			"*|formAction",
			"area|href",
			"area|ping",
			"audio|src",
			"a|href",
			"a|ping",
			"blockquote|cite",
			"body|background",
			"del|cite",
			"form|action",
			"img|src",
			"input|src",
			"ins|cite",
			"q|cite",
			"source|src",
			"track|src",
			"video|poster",
			"video|src",
		})
		registerContext(core.SecurityContextRESOURCE_URL, []string{
			"applet|code",
			"applet|codebase",
			"base|href",
			"embed|src",
			"frame|src",
			"head|profile",
			"html|manifest",
			"iframe|src",
			"link|href",
			"media|src",
			"object|codebase",
			"object|data",
			"script|src",
		})
	}
	return securitySchema
}

func registerContext(ctx core.SecurityContext, specs []string) {
	for _, spec := range specs {
		securitySchema[strings.ToLower(spec)] = ctx
	}
}

// IframeSecuritySensitiveAttrs is the set of security-sensitive attributes of an `<iframe>` that *must* be
// applied as a static attribute only. This ensures that all security-sensitive
// attributes are taken into account while creating an instance of an `<iframe>`
// at runtime.
//
// Note: avoid using this set directly, use the `IsIframeSecuritySensitiveAttr` function
// in the code instead.
var IframeSecuritySensitiveAttrs = map[string]bool{
	"sandbox":         true,
	"allow":           true,
	"allowfullscreen": true,
	"referrerpolicy":  true,
	"csp":             true,
	"fetchpriority":   true,
}

// IsIframeSecuritySensitiveAttr checks whether a given attribute name might represent a security-sensitive
// attribute of an <iframe>.
func IsIframeSecuritySensitiveAttr(attrName string) bool {
	// The `setAttribute` DOM API is case-insensitive, so we lowercase the value
	// before checking it against a known security-sensitive attributes.
	return IframeSecuritySensitiveAttrs[strings.ToLower(attrName)]
}
