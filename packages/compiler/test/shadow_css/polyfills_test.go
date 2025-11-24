package shadow_css_test

import (
	"strings"
	"testing"
)

func TestPolyfills(t *testing.T) {
	t.Run("should support polyfill-next-selector", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected string
		}{
			{"polyfill-next-selector {content: 'x > y'} z {}", "x[contenta] > y[contenta]{}"},
			{"polyfill-next-selector {content: \"x > y\"} z {}", "x[contenta] > y[contenta]{}"},
			{"polyfill-next-selector {content: 'button[priority=\"1\"]'} z {}", "button[priority=\"1\"][contenta]{}"},
		}

		for _, tc := range testCases {
			css := shim(tc.input, "contenta")
			if !equalCss(css, tc.expected) {
				t.Errorf("For input %q, expected %q, got %q", tc.input, tc.expected, css)
			}
		}
	})

	t.Run("should support polyfill-unscoped-rule", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected string
		}{
			{"polyfill-unscoped-rule {content: '#menu > .bar';color: blue;}", "#menu > .bar {;color: blue;}"},
			{"polyfill-unscoped-rule {content: \"#menu > .bar\";color: blue;}", "#menu > .bar {;color: blue;}"},
			{"polyfill-unscoped-rule {content: 'button[priority=\"1\"]'}", "button[priority=\"1\"] {}"},
		}

		for _, tc := range testCases {
			css := shim(tc.input, "contenta")
			if !strings.Contains(css, tc.expected) {
				t.Errorf("For input %q, expected to contain %q, got %q", tc.input, tc.expected, css)
			}
		}
	})

	t.Run("should support multiple instances polyfill-unscoped-rule", func(t *testing.T) {
		css := shim(
			"polyfill-unscoped-rule {content: 'foo';color: blue;}"+
				"polyfill-unscoped-rule {content: 'bar';color: blue;}",
			"contenta",
		)
		if !strings.Contains(css, "foo {;color: blue;}") {
			t.Errorf("Expected to contain 'foo {;color: blue;}', got %q", css)
		}
		if !strings.Contains(css, "bar {;color: blue;}") {
			t.Errorf("Expected to contain 'bar {;color: blue;}', got %q", css)
		}
	})

	t.Run("should support polyfill-rule", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected string
		}{
			{"polyfill-rule {content: ':host.foo .bar';color: blue;}", ".foo[a-host] .bar[contenta] {;color:blue;}"},
			{"polyfill-rule {content: \":host.foo .bar\";color:blue;}", ".foo[a-host] .bar[contenta] {;color:blue;}"},
			{"polyfill-rule {content: 'button[priority=\"1\"]'}", "button[priority=\"1\"][contenta] {}"},
		}

		for _, tc := range testCases {
			css := shim(tc.input, "contenta", "a-host")
			if !equalCss(css, tc.expected) {
				t.Errorf("For input %q, expected %q, got %q", tc.input, tc.expected, css)
			}
		}
	})
}

