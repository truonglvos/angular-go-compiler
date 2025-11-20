package test

import (
	"regexp"
	"testing"

	"ngc-go/packages/compiler/util"
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
			// 4-byte - using UTF-16 surrogate pairs
			{string([]rune{0xD800, 0xDC00}), []byte{0xF0, 0x90, 0x80, 0x80}},
			{string([]rune{0xD834, 0xDF06}), []byte{0xF0, 0x9D, 0x8C, 0x86}},
			{string([]rune{0xDBFF, 0xDFFF}), []byte{0xF4, 0x8F, 0xBF, 0xBF}},
			// unmatched surrogate halves
			// high surrogates: 0xD800 to 0xDBFF
			{string(rune(0xD800)), []byte{0xED, 0xA0, 0x80}},
			{string([]rune{0xD800, 0xD800}), []byte{0xED, 0xA0, 0x80, 0xED, 0xA0, 0x80}},
			{string([]rune{0xD800, 'A'}), []byte{0xED, 0xA0, 0x80, 'A'}},
			{string([]rune{0xD800, 0xD834, 0xDF06, 0xD800}), []byte{0xED, 0xA0, 0x80, 0xF0, 0x9D, 0x8C, 0x86, 0xED, 0xA0, 0x80}},
			{string(rune(0xD9AF)), []byte{0xED, 0xA6, 0xAF}},
			{string(rune(0xDBFF)), []byte{0xED, 0xAF, 0xBF}},
			// low surrogates: 0xDC00 to 0xDFFF
			{string(rune(0xDC00)), []byte{0xED, 0xB0, 0x80}},
			{string([]rune{0xDC00, 0xDC00}), []byte{0xED, 0xB0, 0x80, 0xED, 0xB0, 0x80}},
			{string([]rune{0xDC00, 'A'}), []byte{0xED, 0xB0, 0x80, 'A'}},
			{string([]rune{0xDC00, 0xD834, 0xDF06, 0xDC00}), []byte{0xED, 0xB0, 0x80, 0xF0, 0x9D, 0x8C, 0x86, 0xED, 0xB0, 0x80}},
			{string(rune(0xDEEE)), []byte{0xED, 0xBB, 0xAE}},
			{string(rune(0xDFFF)), []byte{0xED, 0xBF, 0xBF}},
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

