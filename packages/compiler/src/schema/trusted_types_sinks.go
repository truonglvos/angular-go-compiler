package schema

import (
	"strings"
)

// TrustedTypesSinks is the set of tagName|propertyName corresponding to Trusted Types sinks.
// Properties applying to all tags use '*'.
//
// Extracted from, and should be kept in sync with
// https://w3c.github.io/webappsec-trusted-types/dist/spec/#integrations
var TrustedTypesSinks = map[string]bool{
	// NOTE: All strings in this set *must* be lowercase!

	// TrustedHTML
	"iframe|srcdoc": true,
	"*|innerhtml":   true,
	"*|outerhtml":   true,

	// NB: no TrustedScript here, as the corresponding tags are stripped by the compiler.

	// TrustedScriptURL
	"embed|src":       true,
	"object|codebase": true,
	"object|data":     true,
}

// IsTrustedTypesSink returns true if the given property on the given DOM tag is a Trusted Types
// sink. In that case, use `ElementSchemaRegistry.securityContext` to determine which particular
// Trusted Type is required for values passed to the sink:
// - SecurityContext.HTML corresponds to TrustedHTML
// - SecurityContext.RESOURCE_URL corresponds to TrustedScriptURL
func IsTrustedTypesSink(tagName string, propName string) bool {
	// Make sure comparisons are case insensitive, so that case differences between attribute and
	// property names do not have a security impact.
	tagName = strings.ToLower(tagName)
	propName = strings.ToLower(propName)

	key := tagName + "|" + propName
	wildcardKey := "*|" + propName

	return TrustedTypesSinks[key] || TrustedTypesSinks[wildcardKey]
}
