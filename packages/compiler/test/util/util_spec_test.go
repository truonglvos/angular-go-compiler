package util_test

import (
	"ngc-go/packages/compiler/src/util"
	"regexp"
	"testing"
)

func TestUtil(t *testing.T) {
	t.Run("splitAtColon", func(t *testing.T) {
		t.Run("should split when a single : is present", func(t *testing.T) {
			result := util.SplitAtColon("a:b", []string{})
			if len(result) != 2 || result[0] != "a" || result[1] != "b" {
				t.Errorf("Expected ['a', 'b'], got %v", result)
			}
		})

		t.Run("should trim parts", func(t *testing.T) {
			result := util.SplitAtColon(" a : b ", []string{})
			if len(result) != 2 || result[0] != "a" || result[1] != "b" {
				t.Errorf("Expected ['a', 'b'], got %v", result)
			}
		})

		t.Run("should support multiple :", func(t *testing.T) {
			result := util.SplitAtColon("a:b:c", []string{})
			if len(result) != 2 || result[0] != "a" || result[1] != "b:c" {
				t.Errorf("Expected ['a', 'b:c'], got %v", result)
			}
		})

		t.Run("should use the default value when no : is present", func(t *testing.T) {
			result := util.SplitAtColon("ab", []string{"c", "d"})
			if len(result) != 2 || result[0] != "c" || result[1] != "d" {
				t.Errorf("Expected ['c', 'd'], got %v", result)
			}
		})
	})

	t.Run("RegExp", func(t *testing.T) {
		t.Run("should escape regexp", func(t *testing.T) {
			re := regexp.MustCompile(util.EscapeRegExp("b"))
			if !re.MatchString("abc") {
				t.Error("Expected regexp to match 'abc'")
			}
			if re.MatchString("adc") {
				t.Error("Expected regexp to not match 'adc'")
			}
			
			re = regexp.MustCompile(util.EscapeRegExp("a.b"))
			if !re.MatchString("a.b") {
				t.Error("Expected regexp to match 'a.b'")
			}
			if re.MatchString("axb") {
				t.Error("Expected regexp to not match 'axb'")
			}
		})
	})

	t.Run("utf8encode", func(t *testing.T) {
		t.Run("should encode to utf8", func(t *testing.T) {
			tests := []struct {
				input        string
				outputBytes  []byte
				description  string
			}{
				{"abc", []byte{0x61, 0x62, 0x63}, "ascii"},
				// 1-byte
				{"\x00", []byte{0x00}, "null byte"},
				// 2-byte
				{"\u0080", []byte{0xc2, 0x80}, "2-byte U+0080"},
				{"\u05ca", []byte{0xd7, 0x8a}, "2-byte U+05CA"},
				{"\u07ff", []byte{0xdf, 0xbf}, "2-byte U+07FF"},
				// 3-byte
				{"\u0800", []byte{0xe0, 0xa0, 0x80}, "3-byte U+0800"},
				{"\u2c3c", []byte{0xe2, 0xb0, 0xbc}, "3-byte U+2C3C"},
				{"\uffff", []byte{0xef, 0xbf, 0xbf}, "3-byte U+FFFF"},
				// 4-byte (valid UTF-8)
				{"\U00010000", []byte{0xF0, 0x90, 0x80, 0x80}, "4-byte U+10000"},
				{"\U0001D306", []byte{0xF0, 0x9D, 0x8C, 0x86}, "4-byte U+1D306"},
				{"\U0010FFFF", []byte{0xF4, 0x8F, 0xBF, 0xBF}, "4-byte U+10FFFF"},
			}
			
			for _, test := range tests {
				encoded := util.UTF8Encode(test.input)
				
				// Compare byte slices
				if len(encoded) != len(test.outputBytes) {
					t.Errorf("UTF8Encode(%s): expected %d bytes, got %d bytes", 
						test.description, len(test.outputBytes), len(encoded))
					continue
				}
				
				for i := 0; i < len(encoded); i++ {
					if encoded[i] != test.outputBytes[i] {
						t.Errorf("UTF8Encode(%s): byte[%d] expected 0x%02x, got 0x%02x",
							test.description, i, test.outputBytes[i], encoded[i])
					}
				}
			}
		})
		
		// Note: Surrogate tests skipped - Go handles surrogates differently than JavaScript
		// Go automatically replaces invalid surrogates with replacement character (U+FFFD)
		// JavaScript/WTF-8 preserves invalid surrogates as 3-byte sequences
	})

	t.Run("stringify", func(t *testing.T) {
		t.Run("should handle objects with no prototype", func(t *testing.T) {
			// In Go, maps don't have prototypes like JavaScript objects
			// We test with a map which is closest to Object.create(null) in JS
			m := make(map[string]interface{})
			result := util.Stringify(m)
			if result != "object" {
				t.Errorf("Expected Stringify(map) to be 'object', got %q", result)
			}
		})
	})
}

