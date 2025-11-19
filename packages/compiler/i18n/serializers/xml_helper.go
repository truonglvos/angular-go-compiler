package serializers

import (
	"regexp"
)

// EscapeXml escapes XML special characters
func EscapeXml(text string) string {
	escapedChars := []struct {
		re  *regexp.Regexp
		rep string
	}{
		{regexp.MustCompile(`&`), `&amp;`},
		{regexp.MustCompile(`"`), `&quot;`},
		{regexp.MustCompile(`'`), `&apos;`},
		{regexp.MustCompile(`<`), `&lt;`},
		{regexp.MustCompile(`>`), `&gt;`},
	}

	result := text
	for _, ec := range escapedChars {
		result = ec.re.ReplaceAllString(result, ec.rep)
	}
	return result
}
