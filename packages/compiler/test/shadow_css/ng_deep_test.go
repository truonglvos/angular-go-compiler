package shadow_css_test

import (
	"testing"
)

func TestNgDeep(t *testing.T) {
	t.Run("should handle /deep/", func(t *testing.T) {
		css := shim("x /deep/ y {}", "contenta")
		expected := "x[contenta] y {}"
		if !equalCss(css, expected) {
			t.Errorf("Expected %q, got %q", expected, css)
		}
	})

	t.Run("should handle >>>", func(t *testing.T) {
		css := shim("x >>> y {}", "contenta")
		expected := "x[contenta] y {}"
		if !equalCss(css, expected) {
			t.Errorf("Expected %q, got %q", expected, css)
		}
	})

	t.Run("should handle ::ng-deep", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected string
			host     string
		}{
			{"::ng-deep y {}", "y {}", ""},
			{"x ::ng-deep y {}", "x[contenta] y {}", ""},
			{":host > ::ng-deep .x {}", "[h] > .x {}", "h"},
			{":host ::ng-deep > .x {}", "[h] > .x {}", "h"},
			{":host > ::ng-deep > .x {}", "[h] > > .x {}", "h"},
		}

		for _, tc := range testCases {
			host := tc.host
			if host == "" {
				host = "h"
			}
			result := shim(tc.input, "contenta", host)
			if !equalCss(result, tc.expected) {
				t.Errorf("For input %q, expected %q, got %q", tc.input, tc.expected, result)
			}
		}
	})
}

