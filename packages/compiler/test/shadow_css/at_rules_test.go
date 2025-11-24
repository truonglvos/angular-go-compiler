package shadow_css_test

import (
	"strings"
	"testing"
)

func TestAtRules(t *testing.T) {
	t.Run("@media", func(t *testing.T) {
		t.Run("should handle media rules with simple rules", func(t *testing.T) {
			css := "@media screen and (max-width: 800px) {div {font-size: 50px;}} div {}"
			expected := "@media screen and (max-width:800px) {div[contenta] {font-size:50px;}} div[contenta] {}"
			result := shim(css, "contenta")
			if !equalCss(result, expected) {
				t.Errorf("Expected %q, got %q", expected, result)
			}
		})

		t.Run("should handle media rules with both width and height", func(t *testing.T) {
			css := "@media screen and (max-width:800px, max-height:100%) {div {font-size:50px;}}"
			expected := "@media screen and (max-width:800px, max-height:100%) {div[contenta] {font-size:50px;}}"
			result := shim(css, "contenta")
			if !equalCss(result, expected) {
				t.Errorf("Expected %q, got %q", expected, result)
			}
		})
	})

	t.Run("@page", func(t *testing.T) {
		t.Run("should preserve @page rules", func(t *testing.T) {
			contentAttr := "contenta"
			css := `
        @page {
          margin-right: 4in;

          @top-left {
            content: "Hamlet";
          }

          @top-right {
            content: "Page " counter(page);
          }
        }

        @page main {
          margin-left: 4in;
        }

        @page :left {
          margin-left: 3cm;
          margin-right: 4cm;
        }

        @page :right {
          margin-left: 4cm;
          margin-right: 3cm;
        }
      `
			result := shim(css, contentAttr)
			if !equalCss(result, css) {
				t.Errorf("Expected %q, got %q", css, result)
			}
			if strings.Contains(result, contentAttr) {
				t.Errorf("Expected result to not contain %q, but it did", contentAttr)
			}
		})

		t.Run("should strip ::ng-deep and :host from within @page rules", func(t *testing.T) {
			testCases := []struct {
				input    string
				expected string
			}{
				{"@page { margin-right: 4in; }", "@page { margin-right:4in;}"},
				{"@page { ::ng-deep @top-left { content: \"Hamlet\";}}", "@page { @top-left { content:\"Hamlet\";}}"},
				{"@page { :host ::ng-deep @top-left { content:\"Hamlet\";}}", "@page { @top-left { content:\"Hamlet\";}}"},
			}

			for _, tc := range testCases {
				result := shim(tc.input, "contenta", "h")
				if !equalCss(result, tc.expected) {
					t.Errorf("For input %q, expected %q, got %q", tc.input, tc.expected, result)
				}
			}
		})
	})

	t.Run("@supports", func(t *testing.T) {
		t.Run("should handle support rules", func(t *testing.T) {
			css := "@supports (display: flex) {section {display: flex;}}"
			expected := "@supports (display:flex) {section[contenta] {display:flex;}}"
			result := shim(css, "contenta")
			if !equalCss(result, expected) {
				t.Errorf("Expected %q, got %q", expected, result)
			}
		})

		t.Run("should strip ::ng-deep and :host from within @supports", func(t *testing.T) {
			css := "@supports (display: flex) { @font-face { :host ::ng-deep font-family{} } }"
			expected := "@supports (display:flex) { @font-face { font-family{}}}"
			result := shim(css, "contenta", "h")
			if !equalCss(result, expected) {
				t.Errorf("Expected %q, got %q", expected, result)
			}
		})
	})

	t.Run("@font-face", func(t *testing.T) {
		t.Run("should strip ::ng-deep and :host from within @font-face", func(t *testing.T) {
			testCases := []struct {
				input    string
				expected string
			}{
				{"@font-face { font-family {} }", "@font-face { font-family {}}"},
				{"@font-face { ::ng-deep font-family{} }", "@font-face { font-family{}}"},
				{"@font-face { :host ::ng-deep font-family{} }", "@font-face { font-family{}}"},
			}

			for _, tc := range testCases {
				result := shim(tc.input, "contenta", "h")
				if !equalCss(result, tc.expected) {
					t.Errorf("For input %q, expected %q, got %q", tc.input, tc.expected, result)
				}
			}
		})
	})

	t.Run("@import", func(t *testing.T) {
		t.Run("should pass through @import directives", func(t *testing.T) {
			styleStr := "@import url(\"https://fonts.googleapis.com/css?family=Roboto\");"
			css := shim(styleStr, "contenta")
			if !equalCss(css, styleStr) {
				t.Errorf("Expected %q, got %q", styleStr, css)
			}
		})

		t.Run("should shim rules after @import", func(t *testing.T) {
			styleStr := "@import url(\"a\"); div {}"
			css := shim(styleStr, "contenta")
			expected := "@import url(\"a\"); div[contenta] {}"
			if !equalCss(css, expected) {
				t.Errorf("Expected %q, got %q", expected, css)
			}
		})

		t.Run("should shim rules with quoted content after @import", func(t *testing.T) {
			styleStr := "@import url(\"a\"); div {background-image: url(\"a.jpg\"); color: red;}"
			css := shim(styleStr, "contenta")
			expected := "@import url(\"a\"); div[contenta] {background-image:url(\"a.jpg\"); color:red;}"
			if !equalCss(css, expected) {
				t.Errorf("Expected %q, got %q", expected, css)
			}
		})

		t.Run("should pass through @import directives whose URL contains colons and semicolons", func(t *testing.T) {
			styleStr := "@import url(\"https://fonts.googleapis.com/css2?family=Roboto:wght@400;500&display=swap\");"
			css := shim(styleStr, "contenta")
			if !equalCss(css, styleStr) {
				t.Errorf("Expected %q, got %q", styleStr, css)
			}
		})

		t.Run("should shim rules after @import with colons and semicolons", func(t *testing.T) {
			styleStr := "@import url(\"https://fonts.googleapis.com/css2?family=Roboto:wght@400;500&display=swap\"); div {}"
			css := shim(styleStr, "contenta")
			expected := "@import url(\"https://fonts.googleapis.com/css2?family=Roboto:wght@400;500&display=swap\"); div[contenta] {}"
			if !equalCss(css, expected) {
				t.Errorf("Expected %q, got %q", expected, css)
			}
		})
	})

	t.Run("@container", func(t *testing.T) {
		t.Run("should scope normal selectors inside an unnamed container rules", func(t *testing.T) {
			css := `@container max(max-width: 500px) {
               .item {
                 color: red;
               }
             }`
			result := shim(css, "host-a")
			expected := `
        @container max(max-width: 500px) {
           .item[host-a] {
             color: red;
           }
         }`
			if !equalCss(result, expected) {
				t.Errorf("Expected %q, got %q", expected, result)
			}
		})

		t.Run("should scope normal selectors inside a named container rules", func(t *testing.T) {
			css := `
          @container container max(max-width: 500px) {
               .item {
                 color: red;
               }
          }`
			result := shim(css, "host-a")
			expected := `
        @container container max(max-width: 500px) {
          .item[host-a] {
            color: red;
          }
        }`
			if !equalCss(result, expected) {
				t.Errorf("Expected %q, got %q", expected, result)
			}
		})
	})

	t.Run("@scope", func(t *testing.T) {
		t.Run("should scope normal selectors inside a scope rule with scoping limits", func(t *testing.T) {
			css := `
          @scope (.media-object) to (.content > *) {
              img { border-radius: 50%; }
              .content { padding: 1em; }
          }`
			result := shim(css, "host-a")
			expected := `
        @scope (.media-object) to (.content > *) {
          img[host-a] { border-radius: 50%; }
          .content[host-a] { padding: 1em; }
        }`
			if !equalCss(result, expected) {
				t.Errorf("Expected %q, got %q", expected, result)
			}
		})

		t.Run("should scope normal selectors inside a scope rule", func(t *testing.T) {
			css := `
          @scope (.light-scheme) {
              a { color: darkmagenta; }
          }`
			result := shim(css, "host-a")
			expected := `
        @scope (.light-scheme) {
          a[host-a] { color: darkmagenta; }
        }`
			if !equalCss(result, expected) {
				t.Errorf("Expected %q, got %q", expected, result)
			}
		})
	})

	t.Run("@document", func(t *testing.T) {
		t.Run("should handle document rules", func(t *testing.T) {
			css := "@document url(http://www.w3.org/) {div {font-size:50px;}}"
			expected := "@document url(http://www.w3.org/) {div[contenta] {font-size:50px;}}"
			result := shim(css, "contenta")
			if !equalCss(result, expected) {
				t.Errorf("Expected %q, got %q", expected, result)
			}
		})
	})

	t.Run("@layer", func(t *testing.T) {
		t.Run("should handle layer rules", func(t *testing.T) {
			css := "@layer utilities {section {display: flex;}}"
			expected := "@layer utilities {section[contenta] {display:flex;}}"
			result := shim(css, "contenta")
			if !equalCss(result, expected) {
				t.Errorf("Expected %q, got %q", expected, result)
			}
		})
	})

	t.Run("@starting-style", func(t *testing.T) {
		t.Run("should scope normal selectors inside a starting-style rule", func(t *testing.T) {
			css := `
          @starting-style {
              img { border-radius: 50%; }
              .content { padding: 1em; }
          }`
			result := shim(css, "host-a")
			expected := `
        @starting-style {
          img[host-a] { border-radius: 50%; }
          .content[host-a] { padding: 1em; }
        }`
			if !equalCss(result, expected) {
				t.Errorf("Expected %q, got %q", expected, result)
			}
		})
	})
}

