package css

import (
	"regexp"
)

// Some of the code comes from WebComponents.JS
// https://github.com/webcomponents/webcomponentsjs/blob/master/src/HTMLImports/path.js

var urlWithSchemaRegexp = regexp.MustCompile(`^([^:/?#]+):`)

// IsStyleUrlResolvable checks if a style URL is resolvable
func IsStyleUrlResolvable(url *string) bool {
	if url == nil || len(*url) == 0 || (*url)[0] == '/' {
		return false
	}
	schemeMatch := urlWithSchemaRegexp.FindStringSubmatch(*url)
	return schemeMatch == nil || schemeMatch[1] == "package" || schemeMatch[1] == "asset"
}
