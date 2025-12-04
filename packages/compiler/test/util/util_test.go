package util_test

import (
	"regexp"
	"testing"

	"ngc-go/packages/compiler/src/util"
)

func TestSplitAtColon(t *testing.T) {
	t.Run("should split when a single \":\" is present", func(t *testing.T) {
		result := util.SplitAtColon("a:b", []string{})
		if len(result) != 2 {
			t.Errorf("Expected 2 elements, got %d", len(result))
		}
		if result[0] != "a" {
			t.Errorf("Expected first element to be 'a', got '%s'", result[0])
		}
		if result[1] != "b" {
			t.Errorf("Expected second element to be 'b', got '%s'", result[1])
		}
	})

	t.Run("should trim parts", func(t *testing.T) {
		result := util.SplitAtColon(" a : b ", []string{})
		if len(result) != 2 {
			t.Errorf("Expected 2 elements, got %d", len(result))
		}
		if result[0] != "a" {
			t.Errorf("Expected first element to be 'a', got '%s'", result[0])
		}
		if result[1] != "b" {
			t.Errorf("Expected second element to be 'b', got '%s'", result[1])
		}
	})

	t.Run("should support multiple \":\"", func(t *testing.T) {
		result := util.SplitAtColon("a:b:c", []string{})
		if len(result) != 2 {
			t.Errorf("Expected 2 elements, got %d", len(result))
		}
		if result[0] != "a" {
			t.Errorf("Expected first element to be 'a', got '%s'", result[0])
		}
		if result[1] != "b:c" {
			t.Errorf("Expected second element to be 'b:c', got '%s'", result[1])
		}
	})

	t.Run("should use the default value when no \":\" is present", func(t *testing.T) {
		defaultValues := []string{"c", "d"}
		result := util.SplitAtColon("ab", defaultValues)
		if len(result) != 2 {
			t.Errorf("Expected 2 elements, got %d", len(result))
		}
		if result[0] != "c" {
			t.Errorf("Expected first element to be 'c', got '%s'", result[0])
		}
		if result[1] != "d" {
			t.Errorf("Expected second element to be 'd', got '%s'", result[1])
		}
	})
}

func TestEscapeRegExp(t *testing.T) {
	t.Run("should escape regexp", func(t *testing.T) {
		// Test that escaped regexp matches correctly
		escaped := util.EscapeRegExp("b")
		re := regexp.MustCompile(escaped)
		if !re.MatchString("abc") {
			t.Error("Expected regexp to match 'abc'")
		}
		if re.MatchString("adc") {
			t.Error("Expected regexp not to match 'adc'")
		}

		// Test escaping special characters
		escaped = util.EscapeRegExp("a.b")
		re = regexp.MustCompile(escaped)
		if !re.MatchString("a.b") {
			t.Error("Expected regexp to match 'a.b'")
		}
		if re.MatchString("axb") {
			t.Error("Expected regexp not to match 'axb'")
		}
	})
}

func TestUTF8Encode(t *testing.T) {
	t.Run("should encode to utf8", func(t *testing.T) {
		// tests from https://github.com/mathiasbynens/wtf-8
		// Note: For unmatched surrogates, Go's UTF8Encode implementation processes
		// the string byte-by-byte through runes, so we need to ensure the input
		// matches what JavaScript would produce
		tests := []struct {
			input  string
			output []byte
		}{
			{"abc", []byte("abc")},
			// 1-byte
			{"\x00", []byte{0x00}},
			// 2-byte
			{string(rune(0x0080)), []byte{0xc2, 0x80}},
			{string(rune(0x05ca)), []byte{0xd7, 0x8a}},
			{string(rune(0x07ff)), []byte{0xdf, 0xbf}},
			// 3-byte
			{string(rune(0x0800)), []byte{0xe0, 0xa0, 0x80}},
			{string(rune(0x2c3c)), []byte{0xe2, 0xb0, 0xbc}},
			{string(rune(0xffff)), []byte{0xef, 0xbf, 0xbf}},
			// 4-byte - using UTF-16 surrogate pairs (valid pairs)
			// In JavaScript, these are 2 characters that get combined into 1 code point
			// In Go, we create from the combined code point directly
			{string(rune(0x10000)), []byte{0xF0, 0x90, 0x80, 0x80}},  // \uD800\uDC00 -> 0x10000
			{string(rune(0x1D306)), []byte{0xF0, 0x9D, 0x8C, 0x86}},  // \uD834\uDF06 -> 0x1D306
			{string(rune(0x10FFFF)), []byte{0xF4, 0x8F, 0xBF, 0xBF}}, // \uDBFF\uDFFF -> 0x10FFFF
			// unmatched surrogate halves
			// high surrogates: 0xD800 to 0xDBFF
			// For unmatched surrogates, create from UTF-8 bytes to match JavaScript WTF-8 behavior
			// Go will treat these as invalid UTF-8 and process them byte-by-byte through runes
			{string([]byte{0xED, 0xA0, 0x80}), []byte{0xED, 0xA0, 0x80}},                                                                                     // \uD800 as UTF-8
			{string([]byte{0xED, 0xA0, 0x80, 0xED, 0xA0, 0x80}), []byte{0xED, 0xA0, 0x80, 0xED, 0xA0, 0x80}},                                                 // \uD800\uD800
			{string([]byte{0xED, 0xA0, 0x80, 'A'}), []byte{0xED, 0xA0, 0x80, 'A'}},                                                                           // \uD800A
			{string([]byte{0xED, 0xA0, 0x80, 0xF0, 0x9D, 0x8C, 0x86, 0xED, 0xA0, 0x80}), []byte{0xED, 0xA0, 0x80, 0xF0, 0x9D, 0x8C, 0x86, 0xED, 0xA0, 0x80}}, // \uD800\uD834\uDF06\uD800
			{string([]byte{0xED, 0xA6, 0xAF}), []byte{0xED, 0xA6, 0xAF}},                                                                                     // \uD9AF
			{string([]byte{0xED, 0xAF, 0xBF}), []byte{0xED, 0xAF, 0xBF}},                                                                                     // \uDBFF
			// low surrogates: 0xDC00 to 0xDFFF
			{string([]byte{0xED, 0xB0, 0x80}), []byte{0xED, 0xB0, 0x80}},                                                                                     // \uDC00
			{string([]byte{0xED, 0xB0, 0x80, 0xED, 0xB0, 0x80}), []byte{0xED, 0xB0, 0x80, 0xED, 0xB0, 0x80}},                                                 // \uDC00\uDC00
			{string([]byte{0xED, 0xB0, 0x80, 'A'}), []byte{0xED, 0xB0, 0x80, 'A'}},                                                                           // \uDC00A
			{string([]byte{0xED, 0xB0, 0x80, 0xF0, 0x9D, 0x8C, 0x86, 0xED, 0xB0, 0x80}), []byte{0xED, 0xB0, 0x80, 0xF0, 0x9D, 0x8C, 0x86, 0xED, 0xB0, 0x80}}, // \uDC00\uD834\uDF06\uDC00
			{string([]byte{0xED, 0xBB, 0xAE}), []byte{0xED, 0xBB, 0xAE}},                                                                                     // \uDEEE
			{string([]byte{0xED, 0xBF, 0xBF}), []byte{0xED, 0xBF, 0xBF}},                                                                                     // \uDFFF
		}

		for _, tt := range tests {
			encoded := util.UTF8Encode(tt.input)
			// Compare bytes directly
			if len(encoded) != len(tt.output) {
				t.Errorf("UTF8Encode(%q) length = %d, want %d", tt.input, len(encoded), len(tt.output))
				continue
			}
			for i := range encoded {
				if encoded[i] != tt.output[i] {
					t.Errorf("UTF8Encode(%q)[%d] = 0x%02x, want 0x%02x", tt.input, i, encoded[i], tt.output[i])
				}
			}
		}
	})
}

func TestStringify(t *testing.T) {
	t.Run("should handle objects with no prototype", func(t *testing.T) {
		// In Go, we can't create objects with no prototype like JavaScript's Object.create(null)
		// But we can test with a map which is similar
		m := make(map[string]interface{})
		result := util.Stringify(m)
		// The result should be some string representation
		// In Go, an empty map stringifies differently than JS, but we check it doesn't panic
		if result == "" {
			t.Error("Expected non-empty string result")
		}
	})

	t.Run("should handle nil", func(t *testing.T) {
		result := util.Stringify(nil)
		if result != "null" {
			t.Errorf("Expected 'null', got '%s'", result)
		}
	})

	t.Run("should handle strings", func(t *testing.T) {
		result := util.Stringify("test")
		if result != "test" {
			t.Errorf("Expected 'test', got '%s'", result)
		}
	})
}
