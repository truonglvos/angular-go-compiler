package util

import (
	"fmt"
	"regexp"
	"strings"
)

var dashCaseRegexp = regexp.MustCompile(`-+([a-z0-9])`)

// DashCaseToCamelCase converts a dash-case string to camelCase
func DashCaseToCamelCase(input string) string {
	return dashCaseRegexp.ReplaceAllStringFunc(input, func(match string) string {
		parts := dashCaseRegexp.FindStringSubmatch(match)
		if len(parts) > 1 {
			return strings.ToUpper(parts[1])
		}
		return match
	})
}

// SplitAtColon splits a string at the colon character
func SplitAtColon(input string, defaultValues []string) []string {
	return splitAt(input, ':', defaultValues)
}

// SplitAtPeriod splits a string at the period character
func SplitAtPeriod(input string, defaultValues []string) []string {
	return splitAt(input, '.', defaultValues)
}

func splitAt(input string, character rune, defaultValues []string) []string {
	index := strings.IndexRune(input, character)
	if index == -1 {
		return defaultValues
	}
	return []string{
		strings.TrimSpace(input[:index]),
		strings.TrimSpace(input[index+1:]),
	}
}

// NoUndefined converts undefined (nil) to null equivalent
func NoUndefined[T any](val *T) *T {
	if val == nil {
		return nil
	}
	return val
}

// Error creates an error with a formatted message
func Error(msg string) error {
	return fmt.Errorf("Internal Error: %s", msg)
}

// EscapeRegExp escapes characters that have a special meaning in Regular Expressions
func EscapeRegExp(s string) string {
	re := regexp.MustCompile(`([.*+?^=!:${}()|[\]/\\])`)
	return re.ReplaceAllString(s, `\$1`)
}

// Byte represents a byte
type Byte = uint8

// UTF8Encode encodes a string to UTF-8 bytes
// This matches JavaScript's utf8Encode behavior, including WTF-8 handling for invalid UTF-8
// JavaScript processes strings as UTF-16 (charCodeAt), so we need to handle surrogates correctly
func UTF8Encode(str string) []Byte {
	var encoded []Byte
	bytes := []byte(str)
	
	// Check if string contains invalid UTF-8 surrogates (WTF-8)
	// Invalid surrogates are encoded as 0xED 0xA0-0xBF 0x80-0xBF in UTF-8
	// These are invalid UTF-8 sequences that should be passed through as-is
	hasInvalidUTF8 := false
	for i := 0; i < len(bytes); i++ {
		if bytes[i] == 0xED && i+2 < len(bytes) {
			// Check if this is a surrogate sequence (0xED 0xA0-0xBF 0x80-0xBF)
			if (bytes[i+1] >= 0xA0 && bytes[i+1] <= 0xBF) && 
			   (bytes[i+2] >= 0x80 && bytes[i+2] <= 0xBF) {
				hasInvalidUTF8 = true
				break
			}
		}
	}
	
	// If we detect invalid UTF-8 surrogates, pass through bytes directly (WTF-8)
	if hasInvalidUTF8 {
		for i := 0; i < len(bytes); i++ {
			encoded = append(encoded, Byte(bytes[i]))
		}
		return encoded
	}
	
	// Process rune-by-rune for valid UTF-8
	// Note: Go already combines valid surrogate pairs into single code points
	// So we just need to encode the code points
	runes := []rune(str)
	for i := 0; i < len(runes); i++ {
		codePoint := int(runes[i])

		// Go already handles surrogate pair combination, so codePoint might be > 0xFFFF
		// for valid pairs (e.g., 0x10000 for \uD800\uDC00)
		// We just need to encode it correctly

		if codePoint <= 0x7F {
			encoded = append(encoded, Byte(codePoint))
		} else if codePoint <= 0x7FF {
			encoded = append(encoded,
				Byte(((codePoint>>6)&0x1F)|0xC0),
				Byte((codePoint&0x3F)|0x80),
			)
		} else if codePoint <= 0xFFFF {
			encoded = append(encoded,
				Byte((codePoint>>12)|0xE0),
				Byte(((codePoint>>6)&0x3F)|0x80),
				Byte((codePoint&0x3F)|0x80),
			)
		} else if codePoint <= 0x1FFFFF {
			encoded = append(encoded,
				Byte(((codePoint>>18)&0x07)|0xF0),
				Byte(((codePoint>>12)&0x3F)|0x80),
				Byte(((codePoint>>6)&0x3F)|0x80),
				Byte((codePoint&0x3F)|0x80),
			)
		}
	}

	return encoded
}

// Stringify converts a token to its string representation
func Stringify(token interface{}) string {
	if s, ok := token.(string); ok {
		return s
	}

	// Handle arrays/slices
	if arr, ok := token.([]interface{}); ok {
		parts := make([]string, len(arr))
		for i, v := range arr {
			parts[i] = Stringify(v)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	}

	if token == nil {
		return "null"
	}

	// Try to get name from token if it has one
	if named, ok := token.(interface{ Name() string }); ok {
		return named.Name()
	}

	if named, ok := token.(interface{ OverriddenName() string }); ok {
		return named.OverriddenName()
	}

	// Fallback to string representation
	return fmt.Sprintf("%v", token)
}

// Console represents a console interface
type Console interface {
	Log(message string)
	Warn(message string)
	Error(message string)
}

