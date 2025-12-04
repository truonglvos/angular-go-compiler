package render3_test

import (
	"ngc-go/packages/compiler/src/template/pipeline/src/phases"
	"testing"
)

func TestStyleParsing(t *testing.T) {
	t.Run("should parse empty or blank strings", func(t *testing.T) {
		result1 := phases.Parse("")
		if len(result1) != 0 {
			t.Errorf("Expected empty result, got %v", result1)
		}

		result2 := phases.Parse("    ")
		if len(result2) != 0 {
			t.Errorf("Expected empty result, got %v", result2)
		}
	})

	t.Run("should parse a string into a key/value map", func(t *testing.T) {
		result := phases.Parse("width:100px;height:200px;opacity:0")
		expected := []string{"width", "100px", "height", "200px", "opacity", "0"}
		if !equalStringSlices(result, expected) {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})

	t.Run("should allow empty values", func(t *testing.T) {
		result := phases.Parse("width:;height:   ;")
		expected := []string{"width", "", "height", ""}
		if !equalStringSlices(result, expected) {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})

	t.Run("should trim values and properties", func(t *testing.T) {
		result := phases.Parse("width :333px ; height:666px    ; opacity: 0.5;")
		expected := []string{"width", "333px", "height", "666px", "opacity", "0.5"}
		if !equalStringSlices(result, expected) {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})

	t.Run("should not mess up with quoted strings that contain [:;] values", func(t *testing.T) {
		result := phases.Parse(`content: "foo; man: guy"; width: 100px`)
		expected := []string{"content", `"foo; man: guy"`, "width", "100px"}
		if !equalStringSlices(result, expected) {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})

	t.Run("should not mess up with quoted strings that contain inner quote values", func(t *testing.T) {
		quoteStr := `"one 'two' three "four" five"`
		result := phases.Parse(`content: ` + quoteStr + `; width: 123px`)
		expected := []string{"content", quoteStr, "width", "123px"}
		if !equalStringSlices(result, expected) {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})

	t.Run("should respect parenthesis that are placed within a style", func(t *testing.T) {
		result := phases.Parse(`background-image: url("foo.jpg")`)
		expected := []string{"background-image", `url("foo.jpg")`}
		if !equalStringSlices(result, expected) {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})

	t.Run("should respect multi-level parenthesis that contain special [:;] characters", func(t *testing.T) {
		result := phases.Parse(`color: rgba(calc(50 * 4), var(--cool), :5;); height: 100px;`)
		expected := []string{"color", "rgba(calc(50 * 4), var(--cool), :5;)", "height", "100px"}
		if !equalStringSlices(result, expected) {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})

	t.Run("should hyphenate style properties from camel case", func(t *testing.T) {
		result := phases.Parse("borderWidth: 200px")
		expected := []string{"border-width", "200px"}
		if !equalStringSlices(result, expected) {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})

	t.Run("should not remove quotes from string data types", func(t *testing.T) {
		result := phases.Parse(`content: "foo"`)
		expected := []string{"content", `"foo"`}
		if !equalStringSlices(result, expected) {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})

	t.Run("should not remove quotes that changes the value context from invalid to valid", func(t *testing.T) {
		result := phases.Parse(`width: "1px"`)
		expected := []string{"width", `"1px"`}
		if !equalStringSlices(result, expected) {
			t.Errorf("Expected %v, got %v", expected, result)
		}
	})

	t.Run("camelCasing => hyphenation", func(t *testing.T) {
		t.Run("should convert a camel-cased value to a hyphenated value", func(t *testing.T) {
			// Note: hyphenateStyleProperty is not exported, so we test it indirectly through Parse
			// Test cases from TypeScript:
			// expect(hyphenate('fooBar')).toEqual('foo-bar');
			// expect(hyphenate('fooBarMan')).toEqual('foo-bar-man');
			// expect(hyphenate('-fooBar-man')).toEqual('-foo-bar-man');
			
			// Test through Parse which uses hyphenateStyleProperty internally
			result1 := phases.Parse("fooBar: value")
			if len(result1) < 2 || result1[0] != "foo-bar" {
				t.Errorf("Expected 'foo-bar', got %q", result1[0])
			}

			result2 := phases.Parse("fooBarMan: value")
			if len(result2) < 2 || result2[0] != "foo-bar-man" {
				t.Errorf("Expected 'foo-bar-man', got %q", result2[0])
			}

			result3 := phases.Parse("-fooBar-man: value")
			if len(result3) < 2 || result3[0] != "-foo-bar-man" {
				t.Errorf("Expected '-foo-bar-man', got %q", result3[0])
			}
		})

		t.Run("should make everything lowercase", func(t *testing.T) {
			result := phases.Parse("-WebkitAnimation: value")
			if len(result) < 2 || result[0] != "-webkit-animation" {
				t.Errorf("Expected '-webkit-animation', got %q", result[0])
			}
		})
	})
}

// equalStringSlices compares two string slices for equality
func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

