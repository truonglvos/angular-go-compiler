package output_test

import (
	"ngc-go/packages/compiler/src/output"
	"strings"
	"testing"
)

func TestAbstractEmitter(t *testing.T) {
	t.Run("escapeIdentifier", func(t *testing.T) {
		t.Run("should escape single quotes", func(t *testing.T) {
			result := output.EscapeIdentifier("'", false, true)
			expected := "'\\''"
			if result != expected {
				t.Errorf("Expected %q, got %q", expected, result)
			}
		})

		t.Run("should escape backslash", func(t *testing.T) {
			result := output.EscapeIdentifier("\\", false, true)
			expected := "'\\\\'"
			if result != expected {
				t.Errorf("Expected %q, got %q", expected, result)
			}
		})

		t.Run("should escape newlines", func(t *testing.T) {
			result := output.EscapeIdentifier("\n", false, true)
			expected := "'\\n'"
			if result != expected {
				t.Errorf("Expected %q, got %q", expected, result)
			}
		})

		t.Run("should escape carriage returns", func(t *testing.T) {
			result := output.EscapeIdentifier("\r", false, true)
			expected := "'\\r'"
			if result != expected {
				t.Errorf("Expected %q, got %q", expected, result)
			}
		})

		t.Run("should escape $", func(t *testing.T) {
			result := output.EscapeIdentifier("$", true, true)
			expected := "'\\$'"
			if result != expected {
				t.Errorf("Expected %q, got %q", expected, result)
			}
		})

		t.Run("should not escape $", func(t *testing.T) {
			result := output.EscapeIdentifier("$", false, true)
			expected := "'$'"
			if result != expected {
				t.Errorf("Expected %q, got %q", expected, result)
			}
		})

		t.Run("should add quotes for non-identifiers", func(t *testing.T) {
			result := output.EscapeIdentifier("==", false, false)
			expected := "'=='"
			if result != expected {
				t.Errorf("Expected %q, got %q", expected, result)
			}
		})

		t.Run("does not escape class (but it probably should)", func(t *testing.T) {
			result := output.EscapeIdentifier("class", false, false)
			expected := "class"
			if result != expected {
				t.Errorf("Expected %q, got %q", expected, result)
			}
		})
	})
}

// stripSourceMapAndNewLine is a utility function for testing
func stripSourceMapAndNewLine(source string) string {
	if strings.HasSuffix(source, "\n") {
		source = source[:len(source)-1]
	}
	smi := strings.LastIndex(source, "\n//#")
	if smi == -1 {
		return source
	}
	return source[:smi]
}

