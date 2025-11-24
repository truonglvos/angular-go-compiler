package shadow_css_test

import (
	"testing"
)

func TestShadowCss(t *testing.T) {
	t.Run("should handle empty string", func(t *testing.T) {
		result := shim("", "contenta")
		if result != "" {
			t.Errorf("Expected empty string, got %q", result)
		}
	})

	t.Run("should add an attribute to every rule", func(t *testing.T) {
		css := "one {color: red;}two {color: red;}"
		expected := "one[contenta] {color:red;}two[contenta] {color:red;}"
		result := shim(css, "contenta")
		if !equalCss(result, expected) {
			t.Errorf("Expected %q, got %q", expected, result)
		}
	})

	t.Run("should handle invalid css", func(t *testing.T) {
		css := "one {color: red;}garbage"
		expected := "one[contenta] {color:red;}garbage"
		result := shim(css, "contenta")
		if !equalCss(result, expected) {
			t.Errorf("Expected %q, got %q", expected, result)
		}
	})

	t.Run("should add an attribute to every selector", func(t *testing.T) {
		css := "one, two {color: red;}"
		expected := "one[contenta], two[contenta] {color:red;}"
		result := shim(css, "contenta")
		if !equalCss(result, expected) {
			t.Errorf("Expected %q, got %q", expected, result)
		}
	})

	t.Run("should support newlines in the selector and content", func(t *testing.T) {
		css := `
      one,
      two {
        color: red;
      }
    `
		expected := `
      one[contenta],
      two[contenta] {
        color: red;
      }
    `
		result := shim(css, "contenta")
		if !equalCss(result, expected) {
			t.Errorf("Expected %q, got %q", expected, result)
		}
	})

	t.Run("should support newlines in the same selector and content", func(t *testing.T) {
		selector := `.foo:not(
      .bar) {
        background-color:
          green;
    }`
		result := shim(selector, "contenta", "a-host")
		expected := `.foo[contenta]:not( .bar) { background-color:green;}`

		if !equalCss(result, expected) {
			t.Errorf("Expected %q, got %q", expected, result)
		}
	})

	t.Run("should handle complicated selectors", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected string
		}{
			{"one::before {}", "one[contenta]::before {}"},
			{"one two {}", "one[contenta] two[contenta] {}"},
			{"one > two {}", "one[contenta] > two[contenta] {}"},
			{"one + two {}", "one[contenta] + two[contenta] {}"},
			{"one ~ two {}", "one[contenta] ~ two[contenta] {}"},
			{".one.two > three {}", ".one.two[contenta] > three[contenta] {}"},
			{"one[attr=\"value\"] {}", "one[attr=\"value\"][contenta] {}"},
			{"one[attr=value] {}", "one[attr=value][contenta] {}"},
			{"one[attr^=\"value\"] {}", "one[attr^=\"value\"][contenta] {}"},
			{"one[attr$=\"value\"] {}", "one[attr$=\"value\"][contenta] {}"},
			{"one[attr*=\"value\"] {}", "one[attr*=\"value\"][contenta] {}"},
			{"one[attr|=\"value\"] {}", "one[attr|=\"value\"][contenta] {}"},
			{"one[attr~=\"value\"] {}", "one[attr~=\"value\"][contenta] {}"},
			{"one[attr=\"va lue\"] {}", "one[attr=\"va lue\"][contenta] {}"},
			{"one[attr] {}", "one[attr][contenta] {}"},
			{"[is=\"one\"] {}", "[is=\"one\"][contenta] {}"},
			{"[attr] {}", "[attr][contenta] {}"},
		}

		for _, tc := range testCases {
			result := shim(tc.input, "contenta")
			if !equalCss(result, tc.expected) {
				t.Errorf("For input %q, expected %q, got %q", tc.input, tc.expected, result)
			}
		}
	})

	t.Run("should transform :host with attributes", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected string
		}{
			{":host [attr] {}", "[hosta] [attr][contenta] {}"},
			{":host(create-first-project) {}", "create-first-project[hosta] {}"},
			{":host[attr] {}", "[attr][hosta] {}"},
			{":host[attr]:where(:not(.cm-button)) {}", "[attr][hosta]:where(:not(.cm-button)) {}"},
		}

		for _, tc := range testCases {
			result := shim(tc.input, "contenta", "hosta")
			if !equalCss(result, tc.expected) {
				t.Errorf("For input %q, expected %q, got %q", tc.input, tc.expected, result)
			}
		}
	})

	t.Run("should handle escaped sequences in selectors", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected string
		}{
			{"one\\/two {}", "one\\/two[contenta] {}"},
			{"one\\:two {}", "one\\:two[contenta] {}"},
			{"one\\\\:two {}", "one\\\\[contenta]:two {}"},
			{".one\\:two {}", ".one\\:two[contenta] {}"},
			{".one\\:\\fc ber {}", ".one\\:\\fc ber[contenta] {}"},
			{".one\\:two .three\\:four {}", ".one\\:two[contenta] .three\\:four[contenta] {}"},
			{"div:where(.one) {}", "div[contenta]:where(.one) {}"},
			{"div:where() {}", "div[contenta]:where() {}"},
			{":where(a):where(b) {}", ":where(a[contenta]):where(b[contenta]) {}"},
			{"*:where(.one) {}", "*[contenta]:where(.one) {}"},
			{"*:where(.one) ::ng-deep .foo {}", "*[contenta]:where(.one) .foo {}"},
		}

		for _, tc := range testCases {
			result := shim(tc.input, "contenta", "hosta")
			if !equalCss(result, tc.expected) {
				t.Errorf("For input %q, expected %q, got %q", tc.input, tc.expected, result)
			}
		}
	})

	t.Run("should handle pseudo functions correctly", func(t *testing.T) {
		// :where()
		testCases := []struct {
			input    string
			expected string
			host     string
		}{
			{":where(.one) {}", ":where(.one[contenta]) {}", "hosta"},
			{":where(div.one span.two) {}", ":where(div.one[contenta] span.two[contenta]) {}", "hosta"},
			{":where(.one) .two {}", ":where(.one[contenta]) .two[contenta] {}", "hosta"},
			{":where(:host) {}", ":where([hosta]) {}", "hosta"},
			{":where(:host) .one {}", ":where([hosta]) .one[contenta] {}", "hosta"},
			{":where(.one) :where(:host) {}", ":where(.one) :where([hosta]) {}", "hosta"},
			{":where(.one :host) {}", ":where(.one [hosta]) {}", "hosta"},
			{"div :where(.one) {}", "div[contenta] :where(.one[contenta]) {}", "hosta"},
			{":host :where(.one .two) {}", "[hosta] :where(.one[contenta] .two[contenta]) {}", "hosta"},
			{":where(.one, .two) {}", ":where(.one[contenta], .two[contenta]) {}", "hosta"},
			{":where(.one > .two) {}", ":where(.one[contenta] > .two[contenta]) {}", "hosta"},
			{":where(> .one) {}", ":where( > .one[contenta]) {}", "hosta"},
			{":where(:not(.one) ~ .two) {}", ":where([contenta]:not(.one) ~ .two[contenta]) {}", "hosta"},
			{":where([foo]) {}", ":where([foo][contenta]) {}", "hosta"},
			// :is()
			{"div:is(.foo) {}", "div[contenta]:is(.foo) {}", "a-host"},
			{":is(.dark :host) {}", ":is(.dark [a-host]) {}", "a-host"},
			{":is(.dark) :is(:host) {}", ":is(.dark) :is([a-host]) {}", "a-host"},
			{":host:is(.foo) {}", "[a-host]:is(.foo) {}", "a-host"},
			{":is(.foo) {}", ":is(.foo[contenta]) {}", "a-host"},
			{":is(.foo, .bar, .baz) {}", ":is(.foo[contenta], .bar[contenta], .baz[contenta]) {}", "a-host"},
			{":is(.foo, .bar) :host {}", ":is(.foo, .bar) [a-host] {}", "a-host"},
			// :is() and :where()
			{":is(.foo, .bar) :is(.baz) :where(.one, .two) :host :where(.three:first-child) {}", ":is(.foo, .bar) :is(.baz) :where(.one, .two) [a-host] :where(.three[contenta]:first-child) {}", "a-host"},
			{":where(:is(a)) {}", ":where(:is(a[contenta])) {}", "hosta"},
			{":where(:is(a, b)) {}", ":where(:is(a[contenta], b[contenta])) {}", "hosta"},
			{":where(:host:is(.one, .two)) {}", ":where([hosta]:is(.one, .two)) {}", "hosta"},
			{":where(:host :is(.one, .two)) {}", ":where([hosta] :is(.one[contenta], .two[contenta])) {}", "hosta"},
			{":where(:is(a, b) :is(.one, .two)) {}", ":where(:is(a[contenta], b[contenta]) :is(.one[contenta], .two[contenta])) {}", "hosta"},
			{":where(:where(a:has(.foo), b) :is(.one, .two:where(.foo > .bar))) {}", ":where(:where(a[contenta]:has(.foo), b[contenta]) :is(.one[contenta], .two[contenta]:where(.foo > .bar))) {}", "hosta"},
			{":where(.two):first-child {}", "[contenta]:where(.two):first-child {}", "hosta"},
			{":first-child:where(.two) {}", "[contenta]:first-child:where(.two) {}", "hosta"},
			{":where(.two):nth-child(3) {}", "[contenta]:where(.two):nth-child(3) {}", "hosta"},
			{"table :where(td, th):hover { color: lime; }", "table[contenta] [contenta]:where(td, th):hover { color:lime;}", "hosta"},
			// :nth
			{":nth-child(3n of :not(p, a), :is(.foo)) {}", "[contenta]:nth-child(3n of :not(p, a), :is(.foo)) {}", "hosta"},
			{"li:nth-last-child(-n + 3) {}", "li[contenta]:nth-last-child(-n + 3) {}", "a-host"},
			{"dd:nth-last-of-type(3n) {}", "dd[contenta]:nth-last-of-type(3n) {}", "a-host"},
			{"dd:nth-of-type(even) {}", "dd[contenta]:nth-of-type(even) {}", "a-host"},
			// complex selectors
			{":host:is([foo],[foo-2])>div.example-2 {}", "[a-host]:is([foo],[foo-2]) > div.example-2[contenta] {}", "a-host"},
			{":host:is([foo], [foo-2]) > div.example-2 {}", "[a-host]:is([foo], [foo-2]) > div.example-2[contenta] {}", "a-host"},
			{":host:has([foo],[foo-2])>div.example-2 {}", "[a-host]:has([foo],[foo-2]) > div.example-2[contenta] {}", "a-host"},
			// :has()
			{"div:has(a) {}", "div[contenta]:has(a) {}", "hosta"},
			{"div:has(a) :host {}", "div:has(a) [hosta] {}", "hosta"},
			{":has(a) :host :has(b) {}", ":has(a) [hosta] [contenta]:has(b) {}", "hosta"},
			{"div:has(~ .one) {}", "div[contenta]:has(~ .one) {}", "hosta"},
			{":has(a) :has(b) {}", "[contenta]:has(a) [contenta]:has(b) {}", "hosta"},
			{":has(a, b) {}", "[contenta]:has(a, b) {}", "hosta"},
			{":has(a, b:where(.foo), :is(.bar)) {}", "[contenta]:has(a, b:where(.foo), :is(.bar)) {}", "hosta"},
			{":has(a, b:where(.foo), :is(.bar):first-child):first-letter {}", "[contenta]:has(a, b:where(.foo), :is(.bar):first-child):first-letter {}", "hosta"},
			{":where(a, b:where(.foo), :has(.bar):first-child) {}", ":where(a[contenta], b[contenta]:where(.foo), [contenta]:has(.bar):first-child) {}", "hosta"},
			{":has(.one :host, .two) {}", "[contenta]:has(.one [hosta], .two) {}", "hosta"},
			{":has(.one, :host) {}", "[contenta]:has(.one, [hosta]) {}", "hosta"},
		}

		for _, tc := range testCases {
			result := shim(tc.input, "contenta", tc.host)
			if !equalCss(result, tc.expected) {
				t.Errorf("For input %q, expected %q, got %q", tc.input, tc.expected, result)
			}
		}
	})

	t.Run("should handle :host inclusions inside pseudo-selectors selectors", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected string
		}{
			{".header:not(.admin) {}", ".header[contenta]:not(.admin) {}"},
			{".header:is(:host > .toolbar, :host ~ .panel) {}", ".header[contenta]:is([hosta] > .toolbar, [hosta] ~ .panel) {}"},
			{".header:where(:host > .toolbar, :host ~ .panel) {}", ".header[contenta]:where([hosta] > .toolbar, [hosta] ~ .panel) {}"},
			{".header:not(.admin, :host.super .header) {}", ".header[contenta]:not(.admin, .super[hosta] .header) {}"},
			{".header:not(.admin, :host.super .header, :host.mega .header) {}", ".header[contenta]:not(.admin, .super[hosta] .header, .mega[hosta] .header) {}"},
			{".one :where(.two, :host) {}", ".one :where(.two[contenta], [hosta]) {}"},
			{".one :where(:host, .two) {}", ".one :where([hosta], .two[contenta]) {}"},
			{":is(.foo):is(:host):is(.two) {}", ":is(.foo):is([hosta]):is(.two[contenta]) {}"},
			{":where(.one, :host .two):first-letter {}", "[contenta]:where(.one, [hosta] .two):first-letter {}"},
			{":first-child:where(.one, :host .two) {}", "[contenta]:first-child:where(.one, [hosta] .two) {}"},
			{":where(.one, :host .two):nth-child(3):is(.foo, a:where(.bar)) {}", "[contenta]:where(.one, [hosta] .two):nth-child(3):is(.foo, a:where(.bar)) {}"},
		}

		for _, tc := range testCases {
			result := shim(tc.input, "contenta", "hosta")
			if !equalCss(result, tc.expected) {
				t.Errorf("For input %q, expected %q, got %q", tc.input, tc.expected, result)
			}
		}
	})

	t.Run("should handle escaped selector with space (if followed by a hex char)", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected string
		}{
			{".\\fc ber {}", ".\\fc ber[contenta] {}"},
			{".\\fc ker {}", ".\\fc[contenta]   ker[contenta] {}"},
			{".pr\\fc fung {}", ".pr\\fc fung[contenta] {}"},
		}

		for _, tc := range testCases {
			result := shim(tc.input, "contenta")
			if result != tc.expected {
				t.Errorf("For input %q, expected %q, got %q", tc.input, tc.expected, result)
			}
		}
	})

	t.Run("should handle ::shadow", func(t *testing.T) {
		css := shim("x::shadow > y {}", "contenta")
		expected := "x[contenta] > y[contenta] {}"
		if !equalCss(css, expected) {
			t.Errorf("Expected %q, got %q", expected, css)
		}
	})

	t.Run("should leave calc() unchanged", func(t *testing.T) {
		styleStr := "div {height:calc(100% - 55px);}"
		css := shim(styleStr, "contenta")
		expected := "div[contenta] {height:calc(100% - 55px);}"
		if !equalCss(css, expected) {
			t.Errorf("Expected %q, got %q", expected, css)
		}
	})

	t.Run("should shim rules with quoted content", func(t *testing.T) {
		styleStr := "div {background-image: url(\"a.jpg\"); color: red;}"
		css := shim(styleStr, "contenta")
		expected := "div[contenta] {background-image:url(\"a.jpg\"); color:red;}"
		if !equalCss(css, expected) {
			t.Errorf("Expected %q, got %q", expected, css)
		}
	})

	t.Run("should shim rules with an escaped quote inside quoted content", func(t *testing.T) {
		styleStr := "div::after { content: \"\\\"\" }"
		css := shim(styleStr, "contenta")
		expected := "div[contenta]::after { content:\"\\\"\"}"
		if !equalCss(css, expected) {
			t.Errorf("Expected %q, got %q", expected, css)
		}
	})

	t.Run("should shim rules with curly braces inside quoted content", func(t *testing.T) {
		styleStr := "div::after { content: \"{}\" }"
		css := shim(styleStr, "contenta")
		expected := "div[contenta]::after { content:\"{}\"}"
		if !equalCss(css, expected) {
			t.Errorf("Expected %q, got %q", expected, css)
		}
	})

	t.Run("should keep retain multiline selectors", func(t *testing.T) {
		styleStr := ".foo,\n.bar { color: red;}"
		css := shim(styleStr, "contenta")
		expected := ".foo[contenta], \n.bar[contenta] { color: red;}"
		if css != expected {
			t.Errorf("Expected %q, got %q", expected, css)
		}
	})

	t.Run("comments", func(t *testing.T) {
		t.Run("should replace multiline comments with newline", func(t *testing.T) {
			result := shim("/* b {c} */ b {c}", "contenta")
			expected := "\n b[contenta] {c}"
			if result != expected {
				t.Errorf("Expected %q, got %q", expected, result)
			}
		})

		t.Run("should replace multiline comments with newline in the original position", func(t *testing.T) {
			result := shim("/* b {c}\n */ b {c}", "contenta")
			expected := "\n\n b[contenta] {c}"
			if result != expected {
				t.Errorf("Expected %q, got %q", expected, result)
			}
		})

		t.Run("should replace comments with newline in the original position", func(t *testing.T) {
			result := shim("/* b {c} */ b {c} /* a {c} */ a {c}", "contenta")
			expected := "\n b[contenta] {c} \n a[contenta] {c}"
			if result != expected {
				t.Errorf("Expected %q, got %q", expected, result)
			}
		})

		t.Run("should keep sourceMappingURL comments", func(t *testing.T) {
			result1 := shim("b {c} /*# sourceMappingURL=data:x */", "contenta")
			expected1 := "b[contenta] {c} /*# sourceMappingURL=data:x */"
			if result1 != expected1 {
				t.Errorf("Expected %q, got %q", expected1, result1)
			}

			result2 := shim("b {c}/* #sourceMappingURL=data:x */", "contenta")
			expected2 := "b[contenta] {c}/* #sourceMappingURL=data:x */"
			if result2 != expected2 {
				t.Errorf("Expected %q, got %q", expected2, result2)
			}
		})

		t.Run("should handle adjacent comments", func(t *testing.T) {
			result := shim("/* comment 1 */ /* comment 2 */ b {c}", "contenta")
			expected := "\n \n b[contenta] {c}"
			if result != expected {
				t.Errorf("Expected %q, got %q", expected, result)
			}
		})
	})
}
