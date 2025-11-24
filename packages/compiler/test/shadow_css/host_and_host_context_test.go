package shadow_css_test

import (
	"testing"
)

func TestHostAndHostContext(t *testing.T) {
	t.Run(":host", func(t *testing.T) {
		t.Run("should handle no context", func(t *testing.T) {
			result := shim(":host {}", "contenta", "a-host")
			expected := "[a-host] {}"
			if !equalCss(result, expected) {
				t.Errorf("Expected %q, got %q", expected, result)
			}
		})

		t.Run("should handle tag selector", func(t *testing.T) {
			result := shim(":host(ul) {}", "contenta", "a-host")
			expected := "ul[a-host] {}"
			if !equalCss(result, expected) {
				t.Errorf("Expected %q, got %q", expected, result)
			}
		})

		t.Run("should handle class selector", func(t *testing.T) {
			result := shim(":host(.x) {}", "contenta", "a-host")
			expected := ".x[a-host] {}"
			if !equalCss(result, expected) {
				t.Errorf("Expected %q, got %q", expected, result)
			}
		})

		t.Run("should handle attribute selector", func(t *testing.T) {
			testCases := []struct {
				input    string
				expected string
			}{
				{":host([a=\"b\"]) {}", "[a=\"b\"][a-host] {}"},
				{":host([a=b]) {}", "[a=b][a-host] {}"},
			}

			for _, tc := range testCases {
				result := shim(tc.input, "contenta", "a-host")
				if !equalCss(result, tc.expected) {
					t.Errorf("For input %q, expected %q, got %q", tc.input, tc.expected, result)
				}
			}
		})

		t.Run("should handle attribute and next operator without spaces", func(t *testing.T) {
			result := shim(":host[foo]>div {}", "contenta", "a-host")
			expected := "[foo][a-host] > div[contenta] {}"
			if !equalCss(result, expected) {
				t.Errorf("Expected %q, got %q", expected, result)
			}
		})

		// Note: Skipping the escaped class selector test as it's marked as xit in TypeScript

		t.Run("should handle multiple tag selectors", func(t *testing.T) {
			testCases := []struct {
				input    string
				expected string
			}{
				{":host(ul,li) {}", "ul[a-host], li[a-host] {}"},
				{":host(ul,li) > .z {}", "ul[a-host] > .z[contenta], li[a-host] > .z[contenta] {}"},
			}

			for _, tc := range testCases {
				result := shim(tc.input, "contenta", "a-host")
				if !equalCss(result, tc.expected) {
					t.Errorf("For input %q, expected %q, got %q", tc.input, tc.expected, result)
				}
			}
		})

		t.Run("should handle compound class selectors", func(t *testing.T) {
			result := shim(":host(.a.b) {}", "contenta", "a-host")
			expected := ".a.b[a-host] {}"
			if !equalCss(result, expected) {
				t.Errorf("Expected %q, got %q", expected, result)
			}
		})

		t.Run("should handle multiple class selectors", func(t *testing.T) {
			testCases := []struct {
				input    string
				expected string
			}{
				{":host(.x,.y) {}", ".x[a-host], .y[a-host] {}"},
				{":host(.x,.y) > .z {}", ".x[a-host] > .z[contenta], .y[a-host] > .z[contenta] {}"},
			}

			for _, tc := range testCases {
				result := shim(tc.input, "contenta", "a-host")
				if !equalCss(result, tc.expected) {
					t.Errorf("For input %q, expected %q, got %q", tc.input, tc.expected, result)
				}
			}
		})

		t.Run("should handle multiple attribute selectors", func(t *testing.T) {
			result := shim(":host([a=\"b\"],[c=d]) {}", "contenta", "a-host")
			expected := "[a=\"b\"][a-host], [c=d][a-host] {}"
			if !equalCss(result, expected) {
				t.Errorf("Expected %q, got %q", expected, result)
			}
		})

		t.Run("should handle pseudo selectors", func(t *testing.T) {
			testCases := []struct {
				input    string
				expected string
			}{
				{":host(:before) {}", "[a-host]:before {}"},
				{":host:before {}", "[a-host]:before {}"},
				{":host:nth-child(8n+1) {}", "[a-host]:nth-child(8n+1) {}"},
				{":host(:nth-child(3n of :not(p, a))) {}", "[a-host]:nth-child(3n of :not(p, a)) {}"},
				{":host:nth-of-type(8n+1) {}", "[a-host]:nth-of-type(8n+1) {}"},
				{":host(.class):before {}", ".class[a-host]:before {}"},
				{":host.class:before {}", ".class[a-host]:before {}"},
				{":host(:not(p)):before {}", "[a-host]:not(p):before {}"},
				{":host(:not(:has(p))) {}", "[a-host]:not(:has(p)) {}"},
				{":host:not(:host.foo) {}", "[a-host]:not([a-host].foo) {}"},
				{":host:not(.foo:host) {}", "[a-host]:not(.foo[a-host]) {}"},
				{":host:not(:host.foo, :host.bar) {}", "[a-host]:not([a-host].foo, .bar[a-host]) {}"},
				{":host:not(:host.foo, .bar :host) {}", "[a-host]:not([a-host].foo, .bar [a-host]) {}"},
				{":host:not(.foo, .bar) {}", "[a-host]:not(.foo, .bar) {}"},
				{":host:not(:has(p, a)) {}", "[a-host]:not(:has(p, a)) {}"},
				{":host(:not(.foo, .bar)) {}", "[a-host]:not(.foo, .bar) {}"},
				{":host:has(> child-element:not(.foo)) {}", "[a-host]:has(> child-element:not(.foo)) {}"},
			}

			for _, tc := range testCases {
				result := shim(tc.input, "contenta", "a-host")
				if !equalCss(result, tc.expected) {
					t.Errorf("For input %q, expected %q, got %q", tc.input, tc.expected, result)
				}
			}
		})

		t.Run("should handle unexpected selectors in the most reasonable way", func(t *testing.T) {
			testCases := []struct {
				input    string
				expected string
			}{
				{"cmp:host {}", "cmp[a-host] {}"},
				{"cmp:host >>> {}", "cmp[a-host] {}"},
				{"cmp:host child {}", "cmp[a-host] child[contenta] {}"},
				{"cmp:host >>> child {}", "cmp[a-host] child {}"},
				{"cmp :host {}", "cmp [a-host] {}"},
				{"cmp :host >>> {}", "cmp [a-host] {}"},
				{"cmp :host child {}", "cmp [a-host] child[contenta] {}"},
				{"cmp :host >>> child {}", "cmp [a-host] child {}"},
			}

			for _, tc := range testCases {
				result := shim(tc.input, "contenta", "a-host")
				if !equalCss(result, tc.expected) {
					t.Errorf("For input %q, expected %q, got %q", tc.input, tc.expected, result)
				}
			}
		})

		t.Run("should support newlines in the same selector and content", func(t *testing.T) {
			selector := `.foo:not(
        :host) {
          background-color:
            green;
      }`
			result := shim(selector, "contenta", "a-host")
			expected := ".foo[contenta]:not( [a-host]) { background-color:green;}"
			if !equalCss(result, expected) {
				t.Errorf("Expected %q, got %q", expected, result)
			}
		})
	})

	t.Run(":host-context", func(t *testing.T) {
		t.Run("should transform :host-context with pseudo selectors", func(t *testing.T) {
			testCases := []struct {
				input    string
				expected string
				host     string
			}{
				{":host-context(backdrop:not(.borderless)) .backdrop {}", "backdrop:not(.borderless)[hosta] .backdrop[contenta], backdrop:not(.borderless) [hosta] .backdrop[contenta] {}", "hosta"},
				{":where(:host-context(backdrop)) {}", ":where(backdrop[hosta]), :where(backdrop [hosta]) {}", "hosta"},
				{":where(:host-context(outer1)) :host(bar) {}", ":where(outer1) bar[hosta] {}", "hosta"},
				{":where(:host-context(.one)) :where(:host-context(.two)) {}", ":where(.one.two[a-host]), :where(.one.two [a-host]), :where(.one .two[a-host]), :where(.one .two [a-host]), :where(.two .one[a-host]), :where(.two .one [a-host]) {}", "a-host"},
				{":where(:host-context(backdrop)) .foo ~ .bar {}", ":where(backdrop[hosta]) .foo[contenta] ~ .bar[contenta], :where(backdrop [hosta]) .foo[contenta] ~ .bar[contenta] {}", "hosta"},
				{":where(:host-context(backdrop)) :host {}", ":where(backdrop) [hosta] {}", "hosta"},
				{"div:where(:host-context(backdrop)) :host {}", "div:where(backdrop) [hosta] {}", "hosta"},
			}

			for _, tc := range testCases {
				result := shim(tc.input, "contenta", tc.host)
				if !equalCss(result, tc.expected) {
					t.Errorf("For input %q, expected %q, got %q", tc.input, tc.expected, result)
				}
			}
		})

		t.Run("should transform :host-context with nested pseudo selectors", func(t *testing.T) {
			testCases := []struct {
				input    string
				expected string
			}{
				{":host-context(:where(.foo:not(.bar))) {}", ":where(.foo:not(.bar))[hosta], :where(.foo:not(.bar)) [hosta] {}"},
				{":host-context(:is(.foo:not(.bar))) {}", ":is(.foo:not(.bar))[hosta], :is(.foo:not(.bar)) [hosta] {}"},
				{":host-context(:where(.foo:not(.bar, .baz))) .inner {}", ":where(.foo:not(.bar, .baz))[hosta] .inner[contenta], :where(.foo:not(.bar, .baz)) [hosta] .inner[contenta] {}"},
			}

			for _, tc := range testCases {
				result := shim(tc.input, "contenta", "hosta")
				if !equalCss(result, tc.expected) {
					t.Errorf("For input %q, expected %q, got %q", tc.input, tc.expected, result)
				}
			}
		})

		t.Run("should handle tag selector", func(t *testing.T) {
			testCases := []struct {
				input    string
				expected string
			}{
				{":host-context(div) {}", "div[a-host], div [a-host] {}"},
				{":host-context(ul) > .y {}", "ul[a-host] > .y[contenta], ul [a-host] > .y[contenta] {}"},
			}

			for _, tc := range testCases {
				result := shim(tc.input, "contenta", "a-host")
				if !equalCss(result, tc.expected) {
					t.Errorf("For input %q, expected %q, got %q", tc.input, tc.expected, result)
				}
			}
		})

		t.Run("should handle class selector", func(t *testing.T) {
			testCases := []struct {
				input    string
				expected string
			}{
				{":host-context(.x) {}", ".x[a-host], .x [a-host] {}"},
				{":host-context(.x) > .y {}", ".x[a-host] > .y[contenta], .x [a-host] > .y[contenta] {}"},
			}

			for _, tc := range testCases {
				result := shim(tc.input, "contenta", "a-host")
				if !equalCss(result, tc.expected) {
					t.Errorf("For input %q, expected %q, got %q", tc.input, tc.expected, result)
				}
			}
		})

		t.Run("should handle attribute selector", func(t *testing.T) {
			testCases := []struct {
				input    string
				expected string
			}{
				{":host-context([a=\"b\"]) {}", "[a=\"b\"][a-host], [a=\"b\"] [a-host] {}"},
				{":host-context([a=b]) {}", "[a=b][a-host], [a=b] [a-host] {}"},
			}

			for _, tc := range testCases {
				result := shim(tc.input, "contenta", "a-host")
				if !equalCss(result, tc.expected) {
					t.Errorf("For input %q, expected %q, got %q", tc.input, tc.expected, result)
				}
			}
		})

		t.Run("should handle multiple :host-context() selectors", func(t *testing.T) {
			result1 := shim(":host-context(.one):host-context(.two) {}", "contenta", "a-host")
			expected1 := ".one.two[a-host], .one.two [a-host], .one .two[a-host], .one .two [a-host], .two .one[a-host], .two .one [a-host] {}"
			if !equalCss(result1, expected1) {
				t.Errorf("Expected %q, got %q", expected1, result1)
			}

			result2 := shim(":host-context(.X):host-context(.Y):host-context(.Z) {}", "contenta", "a-host")
			expected2 := ".X.Y.Z[a-host], .X.Y.Z [a-host], .X.Y .Z[a-host], .X.Y .Z [a-host], .X.Z .Y[a-host], .X.Z .Y [a-host], .X .Y.Z[a-host], .X .Y.Z [a-host], .X .Y .Z[a-host], .X .Y .Z [a-host], .X .Z .Y[a-host], .X .Z .Y [a-host], .Y.Z .X[a-host], .Y.Z .X [a-host], .Y .Z .X[a-host], .Y .Z .X [a-host], .Z .Y .X[a-host], .Z .Y .X [a-host] {}"
			if !equalCss(result2, expected2) {
				t.Errorf("Expected %q, got %q", expected2, result2)
			}
		})

		t.Run("should handle :host-context with no ancestor selectors", func(t *testing.T) {
			testCases := []struct {
				input    string
				expected string
				host     string
			}{
				{":host-context .inner {}", "[a-host] .inner[contenta] {}", "a-host"},
				{":host-context() .inner {}", "[a-host] .inner[contenta] {}", "a-host"},
				{":host-context :host-context(.a) {}", ".a[host-a], .a [host-a] {}", "host-a"},
			}

			for _, tc := range testCases {
				result := shim(tc.input, "contenta", tc.host)
				if !equalCss(result, tc.expected) {
					t.Errorf("For input %q, expected %q, got %q", tc.input, tc.expected, result)
				}
			}
		})

		t.Run("should handle selectors", func(t *testing.T) {
			result := shim(":host-context(.one,.two) .inner {}", "contenta", "a-host")
			expected := ".one[a-host] .inner[contenta], .one [a-host] .inner[contenta], .two[a-host] .inner[contenta], .two [a-host] .inner[contenta] {}"
			if !equalCss(result, expected) {
				t.Errorf("Expected %q, got %q", expected, result)
			}
		})

		t.Run("should handle :host-context with comma-separated child selector", func(t *testing.T) {
			testCases := []struct {
				input    string
				expected string
			}{
				{":host-context(.foo) a:not(.a, .b) {}", ".foo[a-host] a[contenta]:not(.a, .b), .foo [a-host] a[contenta]:not(.a, .b) {}"},
				{":host-context(.foo) a:not([a], .b), .bar, :host-context(.baz) a:not([c], .d) {}", ".foo[a-host] a[contenta]:not([a], .b), .foo [a-host] a[contenta]:not([a], .b), .bar[contenta], .baz[a-host] a[contenta]:not([c], .d), .baz [a-host] a[contenta]:not([c], .d) {}"},
			}

			for _, tc := range testCases {
				result := shim(tc.input, "contenta", "a-host")
				if !equalCss(result, tc.expected) {
					t.Errorf("For input %q, expected %q, got %q", tc.input, tc.expected, result)
				}
			}
		})
	})

	t.Run(":host-context and :host combination selector", func(t *testing.T) {
		t.Run("should handle selectors on the same element", func(t *testing.T) {
			result := shim(":host-context(div):host(.x) > .y {}", "contenta", "a-host")
			expected := "div.x[a-host] > .y[contenta] {}"
			if !equalCss(result, expected) {
				t.Errorf("Expected %q, got %q", expected, result)
			}
		})

		t.Run("should handle no selector :host", func(t *testing.T) {
			testCases := []struct {
				input    string
				expected string
			}{
				{":host:host-context(.one) {}", ".one[a-host][a-host], .one [a-host] {}"},
				{":host-context(.one) :host {}", ".one [a-host] {}"},
			}

			for _, tc := range testCases {
				result := shim(tc.input, "contenta", "a-host")
				if !equalCss(result, tc.expected) {
					t.Errorf("For input %q, expected %q, got %q", tc.input, tc.expected, result)
				}
			}
		})

		t.Run("should handle selectors on different elements", func(t *testing.T) {
			testCases := []struct {
				input    string
				expected string
			}{
				{":host-context(div) :host(.x) > .y {}", "div .x[a-host] > .y[contenta] {}"},
				{":host-context(div) > :host(.x) > .y {}", "div > .x[a-host] > .y[contenta] {}"},
			}

			for _, tc := range testCases {
				result := shim(tc.input, "contenta", "a-host")
				if !equalCss(result, tc.expected) {
					t.Errorf("For input %q, expected %q, got %q", tc.input, tc.expected, result)
				}
			}
		})

		t.Run("should parse multiple rules containing :host-context and :host", func(t *testing.T) {
			input := `
            :host-context(outer1) :host(bar) {}
            :host-context(outer2) :host(foo) {}
        `
			result := shim(input, "contenta", "a-host")
			expected := "outer1 bar[a-host] {} outer2 foo[a-host] {}"
			if !equalCss(result, expected) {
				t.Errorf("Expected %q, got %q", expected, result)
			}
		})
	})
}

