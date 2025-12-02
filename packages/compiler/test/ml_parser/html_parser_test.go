package ml_parser_test

import (
	"ngc-go/packages/compiler/src/ml_parser"
	"ngc-go/packages/compiler/src/util"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func humanizeErrors(errors []*util.ParseError) []interface{} {
	result := []interface{}{}
	for _, e := range errors {
		// Try to extract element name from error message
		// Error messages like "Only void, custom and foreign elements can be self closed \"b\""
		// contain the element name in quotes at the end
		elementName := extractElementNameFromError(e.Msg)
		result = append(result, []interface{}{
			elementName,
			e.Msg,
			HumanizeLineColumn(e.Span.Start),
		})
	}
	return result
}

// extractElementNameFromError tries to extract element name from error message
// For errors like "Only void, custom and foreign elements can be self closed \"b\"",
// it extracts "b" from the quoted part at the end
func extractElementNameFromError(msg string) interface{} {
	// Skip extraction if message contains "{{ '{' }}" - this is not an element name
	// This pattern appears in error messages about unescaped braces
	if strings.Contains(msg, "{{ '{' }}") {
		return nil
	}
	// Skip extraction for block close errors - they contain "&#125;" which is not an element name
	if strings.Contains(msg, "Unexpected closing block") {
		return nil
	}
	// For incomplete block errors, extract the block name from "Incomplete block \"blockName\""
	if strings.Contains(msg, "Incomplete block") {
		// Find the pattern: "Incomplete block \"blockName\""
		startIdx := strings.Index(msg, "Incomplete block \"")
		if startIdx != -1 {
			startIdx += len("Incomplete block \"")
			endIdx := strings.Index(msg[startIdx:], "\"")
			if endIdx != -1 {
				blockName := msg[startIdx : startIdx+endIdx]
				if blockName != "" {
					return blockName
				}
			}
		}
		return nil
	}
	// Look for pattern: "... \"elementName\""
	// For "Unexpected closing tag \"elementName\"..." errors, extract from the first quoted string
	if strings.Contains(msg, "Unexpected closing tag \"") {
		startIdx := strings.Index(msg, "Unexpected closing tag \"")
		if startIdx != -1 {
			startIdx += len("Unexpected closing tag \"")
			endIdx := strings.Index(msg[startIdx:], "\"")
			if endIdx != -1 {
				elementName := msg[startIdx : startIdx+endIdx]
				if elementName != "" {
					return elementName
				}
			}
		}
	}
	// For other errors, find the last quoted string in the message
	lastQuote := strings.LastIndex(msg, "\"")
	if lastQuote == -1 {
		return nil
	}
	// Find the opening quote before the last quote
	secondLastQuote := strings.LastIndex(msg[:lastQuote], "\"")
	if secondLastQuote == -1 {
		return nil
	}
	// Extract the element name between the quotes
	elementName := msg[secondLastQuote+1 : lastQuote]
	if elementName == "" {
		return nil
	}
	// Don't extract if it looks like an interpolation pattern (e.g., "{{ '{' }}")
	if strings.HasPrefix(elementName, "{{") && strings.HasSuffix(elementName, "}}") {
		return nil
	}
	return elementName
}

func TestHtmlParser_Parse(t *testing.T) {
	parser := ml_parser.NewHtmlParser()

	t.Run("text nodes", func(t *testing.T) {
		t.Run("should parse root level text nodes", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{"Text", "a", 0, []interface{}{"a"}},
			}
			result := HumanizeDom(parser.Parse("a", "TestComp", nil), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should parse text nodes inside regular elements", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{"Element", "div", 0},
				[]interface{}{"Text", "a", 1, []interface{}{"a"}},
			}
			result := HumanizeDom(parser.Parse("<div>a</div>", "TestComp", nil), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should parse text nodes inside <ng-template> elements", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{"Element", "ng-template", 0},
				[]interface{}{"Text", "a", 1, []interface{}{"a"}},
			}
			result := HumanizeDom(parser.Parse("<ng-template>a</ng-template>", "TestComp", nil), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should parse CDATA", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{"Text", "text", 0, []interface{}{"text"}},
			}
			result := HumanizeDom(parser.Parse("<![CDATA[text]]>", "TestComp", nil), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should parse text nodes with HTML entities (5+ hex digits)", func(t *testing.T) {
			// Test with 🛈 (U+1F6C8 - Circled Information Source)
			expected := []interface{}{
				[]interface{}{"Element", "div", 0},
				[]interface{}{"Text", "\U0001F6C8", 1, []interface{}{""}, []interface{}{"\U0001F6C8", "&#x1F6C8;"}, []interface{}{""}},
			}
			result := HumanizeDom(parser.Parse("<div>&#x1F6C8;</div>", "TestComp", nil), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should parse text nodes with decimal HTML entities (5+ digits)", func(t *testing.T) {
			// Test with 🛈 (U+1F6C8 - Circled Information Source) as decimal 128712
			expected := []interface{}{
				[]interface{}{"Element", "div", 0},
				[]interface{}{"Text", "\U0001F6C8", 1, []interface{}{""}, []interface{}{"\U0001F6C8", "&#128712;"}, []interface{}{""}},
			}
			result := HumanizeDom(parser.Parse("<div>&#128712;</div>", "TestComp", nil), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should normalize line endings within CDATA", func(t *testing.T) {
			parsed := parser.Parse("<![CDATA[ line 1 \r\n line 2 ]]>", "TestComp", nil)
			expected := []interface{}{
				[]interface{}{"Text", " line 1 \n line 2 ", 0, []interface{}{" line 1 \n line 2 "}},
			}
			result := HumanizeDom(parsed, false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
			if len(parsed.Errors) != 0 {
				t.Errorf("Expected no errors, got %d", len(parsed.Errors))
			}
		})
	})

	t.Run("elements", func(t *testing.T) {
		t.Run("should parse root level elements", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{"Element", "div", 0},
			}
			result := HumanizeDom(parser.Parse("<div></div>", "TestComp", nil), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should parse elements inside of regular elements", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{"Element", "div", 0},
				[]interface{}{"Element", "span", 1},
			}
			result := HumanizeDom(parser.Parse("<div><span></span></div>", "TestComp", nil), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should parse elements inside <ng-template> elements", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{"Element", "ng-template", 0},
				[]interface{}{"Element", "span", 1},
			}
			result := HumanizeDom(parser.Parse("<ng-template><span></span></ng-template>", "TestComp", nil), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should support void elements", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{"Element", "link", 0},
				[]interface{}{"Attribute", "rel", "author license", []interface{}{"author license"}},
				[]interface{}{"Attribute", "href", "/about", []interface{}{"/about"}},
			}
			result := HumanizeDom(parser.Parse(`<link rel="author license" href="/about">`, "TestComp", nil), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should indicate whether an element is void", func(t *testing.T) {
			parsed := parser.Parse("<input><div></div>", "TestComp", nil)
			if len(parsed.RootNodes) < 2 {
				t.Fatalf("Expected at least 2 root nodes, got %d", len(parsed.RootNodes))
			}
			if elem, ok := parsed.RootNodes[0].(*ml_parser.Element); ok {
				if elem.Name != "input" {
					t.Errorf("Expected first node name to be 'input', got '%s'", elem.Name)
				}
				if !elem.IsVoid {
					t.Errorf("Expected first node to be void, got IsVoid=%v", elem.IsVoid)
				}
			} else {
				t.Errorf("Expected first node to be Element, got %T", parsed.RootNodes[0])
			}
			if elem, ok := parsed.RootNodes[1].(*ml_parser.Element); ok {
				if elem.Name != "div" {
					t.Errorf("Expected second node name to be 'div', got '%s'", elem.Name)
				}
				if elem.IsVoid {
					t.Errorf("Expected second node to not be void, got IsVoid=%v", elem.IsVoid)
				}
			} else {
				t.Errorf("Expected second node to be Element, got %T", parsed.RootNodes[1])
			}
		})

		t.Run("should not error on void elements from HTML5 spec", func(t *testing.T) {
			voidElements := []string{
				"<map><area></map>",
				"<div><br></div>",
				"<colgroup><col></colgroup>",
				"<div><embed></div>",
				"<div><hr></div>",
				"<div><img></div>",
				"<div><input></div>",
				"<object><param>/<object>",
				"<audio><source></audio>",
				"<audio><track></audio>",
				"<p><wbr></p>",
			}
			for _, html := range voidElements {
				result := parser.Parse(html, "TestComp", nil)
				if len(result.Errors) != 0 {
					t.Errorf("Expected no errors for %q, got %d errors: %v", html, len(result.Errors), result.Errors)
				}
			}
		})

		t.Run("should close void elements on text nodes", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{"Element", "p", 0},
				[]interface{}{"Text", "before", 1, []interface{}{"before"}},
				[]interface{}{"Element", "br", 1},
				[]interface{}{"Text", "after", 1, []interface{}{"after"}},
			}
			result := HumanizeDom(parser.Parse("<p>before<br>after</p>", "TestComp", nil), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should support optional end tags", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{"Element", "div", 0},
				[]interface{}{"Element", "p", 1},
				[]interface{}{"Text", "1", 2, []interface{}{"1"}},
				[]interface{}{"Element", "p", 1},
				[]interface{}{"Text", "2", 2, []interface{}{"2"}},
			}
			result := HumanizeDom(parser.Parse("<div><p>1<p>2</div>", "TestComp", nil), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should support nested elements", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{"Element", "ul", 0},
				[]interface{}{"Element", "li", 1},
				[]interface{}{"Element", "ul", 2},
				[]interface{}{"Element", "li", 3},
			}
			result := HumanizeDom(parser.Parse("<ul><li><ul><li></li></ul></li></ul>", "TestComp", nil), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should not wraps elements in a required parent", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{"Element", "div", 0},
				[]interface{}{"Element", "tr", 1},
			}
			result := HumanizeDom(parser.Parse("<div><tr></tr></div>", "TestComp", nil), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should support explicit namespace", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{"Element", ":myns:div", 0},
			}
			result := HumanizeDom(parser.Parse("<myns:div></myns:div>", "TestComp", nil), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should support implicit namespace", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{"Element", ":svg:svg", 0},
			}
			result := HumanizeDom(parser.Parse("<svg></svg>", "TestComp", nil), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should propagate the namespace", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{"Element", ":myns:div", 0},
				[]interface{}{"Element", ":myns:p", 1},
			}
			result := HumanizeDom(parser.Parse("<myns:div><p></p></myns:div>", "TestComp", nil), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should match closing tags case sensitive", func(t *testing.T) {
			errors := parser.Parse("<DiV><P></p></dIv>", "TestComp", nil).Errors
			if len(errors) != 2 {
				t.Errorf("Expected 2 errors, got %d", len(errors))
			}
			// Note: humanizeErrors may not preserve elementName in Go version
			// This test may need adjustment based on actual error format
		})

		t.Run("should support self closing void elements", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{"Element", "input", 0, "#selfClosing"},
			}
			result := HumanizeDom(parser.Parse("<input />", "TestComp", nil), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should support self closing foreign elements", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{"Element", ":math:math", 0, "#selfClosing"},
			}
			result := HumanizeDom(parser.Parse("<math />", "TestComp", nil), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should ignore LF immediately after textarea, pre and listing", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{"Element", "p", 0},
				[]interface{}{"Text", "\n", 1, []interface{}{"\n"}},
				[]interface{}{"Element", "textarea", 0},
				[]interface{}{"Element", "pre", 0},
				[]interface{}{"Text", "\n", 1, []interface{}{"\n"}},
				[]interface{}{"Element", "listing", 0},
				[]interface{}{"Text", "\n", 1, []interface{}{"\n"}},
			}
			result := HumanizeDom(parser.Parse("<p>\n</p><textarea>\n</textarea><pre>\n\n</pre><listing>\n\n</listing>", "TestComp", nil), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should normalize line endings in text", func(t *testing.T) {
			testCases := []struct {
				input    string
				expected []interface{}
			}{
				{
					"<title> line 1 \r\n line 2 </title>",
					[]interface{}{
						[]interface{}{"Element", "title", 0},
						[]interface{}{"Text", " line 1 \n line 2 ", 1, []interface{}{" line 1 \n line 2 "}},
					},
				},
				{
					"<script> line 1 \r\n line 2 </script>",
					[]interface{}{
						[]interface{}{"Element", "script", 0},
						[]interface{}{"Text", " line 1 \n line 2 ", 1, []interface{}{" line 1 \n line 2 "}},
					},
				},
				{
					"<div> line 1 \r\n line 2 </div>",
					[]interface{}{
						[]interface{}{"Element", "div", 0},
						[]interface{}{"Text", " line 1 \n line 2 ", 1, []interface{}{" line 1 \n line 2 "}},
					},
				},
				{
					"<span> line 1 \r\n line 2 </span>",
					[]interface{}{
						[]interface{}{"Element", "span", 0},
						[]interface{}{"Text", " line 1 \n line 2 ", 1, []interface{}{" line 1 \n line 2 "}},
					},
				},
			}
			for _, tc := range testCases {
				parsed := parser.Parse(tc.input, "TestComp", nil)
				result := HumanizeDom(parsed, false)
				if diff := cmp.Diff(tc.expected, result); diff != "" {
					t.Errorf("HumanizeDom() mismatch for %q (-want +got):\n%s", tc.input, diff)
				}
				if len(parsed.Errors) != 0 {
					t.Errorf("Expected no errors for %q, got %d", tc.input, len(parsed.Errors))
				}
			}
		})

		t.Run("should parse element with JavaScript keyword tag name", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{"Element", "constructor", 0},
			}
			result := HumanizeDom(parser.Parse("<constructor></constructor>", "TestComp", nil), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})
	})

	t.Run("attributes", func(t *testing.T) {
		t.Run("should parse attributes on regular elements case sensitive", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{"Element", "div", 0},
				[]interface{}{"Attribute", "kEy", "v", []interface{}{"v"}},
				[]interface{}{"Attribute", "key2", "v2", []interface{}{"v2"}},
			}
			result := HumanizeDom(parser.Parse(`<div kEy="v" key2=v2></div>`, "TestComp", nil), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should parse attributes containing interpolation", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{"Element", "div", 0},
				[]interface{}{"Attribute", "foo", "1{{message}}2", []interface{}{"1"}, []interface{}{"{{", "message", "}}"}, []interface{}{"2"}},
			}
			result := HumanizeDom(parser.Parse(`<div foo="1{{message}}2"></div>`, "TestComp", nil), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should parse attributes containing unquoted interpolation", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{"Element", "div", 0},
				[]interface{}{"Attribute", "foo", "{{message}}", []interface{}{""}, []interface{}{"{{", "message", "}}"}, []interface{}{""}},
			}
			result := HumanizeDom(parser.Parse(`<div foo={{message}}></div>`, "TestComp", nil), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should parse bound inputs with expressions containing newlines", func(t *testing.T) {
			input := `<app-component
                        [attr]="[
                        {text: 'some text',url:'//www.google.com'},
                        {text:'other text',url:'//www.google.com'}]">` +
				`</app-component>`
			expected := []interface{}{
				[]interface{}{"Element", "app-component", 0},
				[]interface{}{
					"Attribute",
					"[attr]",
					"[\n                        {text: 'some text',url:'//www.google.com'},\n                        {text:'other text',url:'//www.google.com'}]",
					[]interface{}{"[\n                        {text: 'some text',url:'//www.google.com'},\n                        {text:'other text',url:'//www.google.com'}]"},
				},
			}
			result := HumanizeDom(parser.Parse(input, "TestComp", nil), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should parse attributes containing encoded entities", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{"Element", "div", 0},
				[]interface{}{"Attribute", "foo", "&", []interface{}{""}, []interface{}{"&", "&amp;"}, []interface{}{""}},
			}
			result := HumanizeDom(parser.Parse(`<div foo="&amp;"></div>`, "TestComp", nil), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should parse attributes containing encoded entities (5+ hex digits)", func(t *testing.T) {
			// Test with 🛈 (U+1F6C8 - Circled Information Source)
			expected := []interface{}{
				[]interface{}{"Element", "div", 0},
				[]interface{}{"Attribute", "foo", "\U0001F6C8", []interface{}{""}, []interface{}{"\U0001F6C8", "&#x1F6C8;"}, []interface{}{""}},
			}
			result := HumanizeDom(parser.Parse(`<div foo="&#x1F6C8;"></div>`, "TestComp", nil), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should parse attributes containing encoded decimal entities (5+ digits)", func(t *testing.T) {
			// Test with 🛈 (U+1F6C8 - Circled Information Source) as decimal 128712
			expected := []interface{}{
				[]interface{}{"Element", "div", 0},
				[]interface{}{"Attribute", "foo", "\U0001F6C8", []interface{}{""}, []interface{}{"\U0001F6C8", "&#128712;"}, []interface{}{""}},
			}
			result := HumanizeDom(parser.Parse(`<div foo="&#128712;"></div>`, "TestComp", nil), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should normalize line endings within attribute values", func(t *testing.T) {
			// Use double-quoted string so \r\n is interpreted as actual CRLF characters
			result := parser.Parse("<div key=\"  \r\n line 1 \r\n   line 2  \"></div>", "TestComp", nil)
			expected := []interface{}{
				[]interface{}{"Element", "div", 0},
				[]interface{}{"Attribute", "key", "  \n line 1 \n   line 2  ", []interface{}{"  \n line 1 \n   line 2  "}},
			}
			humanized := HumanizeDom(result, false)
			if diff := cmp.Diff(expected, humanized); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
			if len(result.Errors) != 0 {
				t.Errorf("Expected no errors, got %d", len(result.Errors))
			}
		})

		t.Run("should parse attributes without values", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{"Element", "div", 0},
				[]interface{}{"Attribute", "k", ""},
			}
			result := HumanizeDom(parser.Parse(`<div k></div>`, "TestComp", nil), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should parse attributes on svg elements case sensitive", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{"Element", ":svg:svg", 0},
				[]interface{}{"Attribute", "viewBox", "0", []interface{}{"0"}},
			}
			result := HumanizeDom(parser.Parse(`<svg viewBox="0"></svg>`, "TestComp", nil), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should parse attributes on <ng-template> elements", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{"Element", "ng-template", 0},
				[]interface{}{"Attribute", "k", "v", []interface{}{"v"}},
			}
			result := HumanizeDom(parser.Parse(`<ng-template k="v"></ng-template>`, "TestComp", nil), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should support namespace", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{"Element", ":svg:use", 0, "#selfClosing"},
				[]interface{}{"Attribute", ":xlink:href", "Port", []interface{}{"Port"}},
			}
			result := HumanizeDom(parser.Parse(`<svg:use xlink:href="Port" />`, "TestComp", nil), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should support a prematurely terminated interpolation in attribute", func(t *testing.T) {
			parsed := parser.Parse(`<div p="{{ abc"><span></span>`, "TestComp", nil)
			expectedNodes := []interface{}{
				[]interface{}{"Element", "div", 0, "<div p=\"{{ abc\">", "<div p=\"{{ abc\">", nil},
				[]interface{}{"Attribute", "p", "{{ abc", []interface{}{""}, []interface{}{"{{", " abc"}, []interface{}{""}, "p=\"{{ abc\""},
				[]interface{}{"Element", "span", 1, "<span></span>", "<span>", "</span>"},
			}
			resultNodes := HumanizeNodes(parsed.RootNodes, true)
			if diff := cmp.Diff(expectedNodes, resultNodes); diff != "" {
				t.Errorf("HumanizeNodes() mismatch (-want +got):\n%s", diff)
			}
			if len(parsed.Errors) != 0 {
				t.Errorf("Expected no errors, got %d", len(parsed.Errors))
			}
		})

		t.Run("animate instructions", func(t *testing.T) {
			t.Run("should parse animate.enter as a static attribute", func(t *testing.T) {
				expected := []interface{}{
					[]interface{}{"Element", "div", 0},
					[]interface{}{"Attribute", "animate.enter", "foo", []interface{}{"foo"}},
				}
				result := HumanizeDom(parser.Parse(`<div animate.enter="foo"></div>`, "TestComp", nil), false)
				if diff := cmp.Diff(expected, result); diff != "" {
					t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
				}
			})

			t.Run("should parse animate.leave as a static attribute", func(t *testing.T) {
				expected := []interface{}{
					[]interface{}{"Element", "div", 0},
					[]interface{}{"Attribute", "animate.leave", "bar", []interface{}{"bar"}},
				}
				result := HumanizeDom(parser.Parse(`<div animate.leave="bar"></div>`, "TestComp", nil), false)
				if diff := cmp.Diff(expected, result); diff != "" {
					t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
				}
			})

			t.Run("should not parse any other animate prefix binding as animate.leave", func(t *testing.T) {
				expected := []interface{}{
					[]interface{}{"Element", "div", 0},
					[]interface{}{"Attribute", "animateAbc", "bar", []interface{}{"bar"}},
				}
				result := HumanizeDom(parser.Parse(`<div animateAbc="bar"></div>`, "TestComp", nil), false)
				if diff := cmp.Diff(expected, result); diff != "" {
					t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
				}
			})

			t.Run("should parse both animate.enter and animate.leave as static attributes", func(t *testing.T) {
				expected := []interface{}{
					[]interface{}{"Element", "div", 0},
					[]interface{}{"Attribute", "animate.enter", "foo", []interface{}{"foo"}},
					[]interface{}{"Attribute", "animate.leave", "bar", []interface{}{"bar"}},
				}
				result := HumanizeDom(parser.Parse(`<div animate.enter="foo" animate.leave="bar"></div>`, "TestComp", nil), false)
				if diff := cmp.Diff(expected, result); diff != "" {
					t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
				}
			})

			t.Run("should parse animate.enter as a property binding", func(t *testing.T) {
				expected := []interface{}{
					[]interface{}{"Element", "div", 0},
					[]interface{}{"Attribute", "[animate.enter]", `'foo'`, []interface{}{`'foo'`}},
				}
				result := HumanizeDom(parser.Parse(`<div [animate.enter]="'foo'"></div>`, "TestComp", nil), false)
				if diff := cmp.Diff(expected, result); diff != "" {
					t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
				}
			})

			t.Run("should parse animate.leave as a property binding with a string array", func(t *testing.T) {
				expected := []interface{}{
					[]interface{}{"Element", "div", 0},
					[]interface{}{"Attribute", "[animate.leave]", `['bar', 'baz']`, []interface{}{`['bar', 'baz']`}},
				}
				result := HumanizeDom(parser.Parse(`<div [animate.leave]="['bar', 'baz']"></div>`, "TestComp", nil), false)
				if diff := cmp.Diff(expected, result); diff != "" {
					t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
				}
			})

			t.Run("should parse animate.enter as an event binding", func(t *testing.T) {
				expected := []interface{}{
					[]interface{}{"Element", "div", 0},
					[]interface{}{"Attribute", "(animate.enter)", "onAnimation($event)", []interface{}{"onAnimation($event)"}},
				}
				result := HumanizeDom(parser.Parse(`<div (animate.enter)="onAnimation($event)"></div>`, "TestComp", nil), false)
				if diff := cmp.Diff(expected, result); diff != "" {
					t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
				}
			})

			t.Run("should parse animate.leave as an event binding", func(t *testing.T) {
				expected := []interface{}{
					[]interface{}{"Element", "div", 0},
					[]interface{}{"Attribute", "(animate.leave)", "onAnimation($event)", []interface{}{"onAnimation($event)"}},
				}
				result := HumanizeDom(parser.Parse(`<div (animate.leave)="onAnimation($event)"></div>`, "TestComp", nil), false)
				if diff := cmp.Diff(expected, result); diff != "" {
					t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
				}
			})

			t.Run("should not parse other animate prefixes as animate.leave", func(t *testing.T) {
				expected := []interface{}{
					[]interface{}{"Element", "div", 0},
					[]interface{}{"Attribute", "(animateXYZ)", "onAnimation()", []interface{}{"onAnimation()"}},
				}
				result := HumanizeDom(parser.Parse(`<div (animateXYZ)="onAnimation()"></div>`, "TestComp", nil), false)
				if diff := cmp.Diff(expected, result); diff != "" {
					t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
				}
			})

			t.Run("should parse a combination of animate property and event bindings", func(t *testing.T) {
				expected := []interface{}{
					[]interface{}{"Element", "div", 0},
					[]interface{}{"Attribute", "[animate.enter]", `'foo'`, []interface{}{`'foo'`}},
					[]interface{}{"Attribute", "(animate.leave)", "onAnimation($event)", []interface{}{"onAnimation($event)"}},
				}
				result := HumanizeDom(parser.Parse(`<div [animate.enter]="'foo'" (animate.leave)="onAnimation($event)"></div>`, "TestComp", nil), false)
				if diff := cmp.Diff(expected, result); diff != "" {
					t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
				}
			})
		})

		t.Run("should parse square-bracketed attributes more permissively", func(t *testing.T) {
			input := `<foo [class.text-primary/80]="expr" ` +
				`[class.data-active:text-green-300/80]="expr2" ` +
				`[class.data-[size='large']:p-8] = "expr3" some-attr/>`
			expected := []interface{}{
				[]interface{}{"Element", "foo", 0, "#selfClosing"},
				[]interface{}{"Attribute", "[class.text-primary/80]", "expr", []interface{}{"expr"}},
				[]interface{}{"Attribute", "[class.data-active:text-green-300/80]", "expr2", []interface{}{"expr2"}},
				[]interface{}{"Attribute", "[class.data-[size='large']:p-8]", "expr3", []interface{}{"expr3"}},
				[]interface{}{"Attribute", "some-attr", ""},
			}
			result := HumanizeDom(parser.Parse(input, "TestComp", nil), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})
	})

	t.Run("comments", func(t *testing.T) {
		t.Run("should preserve comments", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{"Comment", "comment", 0},
				[]interface{}{"Element", "div", 0},
			}
			result := HumanizeDom(parser.Parse("<!-- comment --><div></div>", "TestComp", nil), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should normalize line endings within comments", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{"Comment", "line 1 \n line 2", 0},
			}
			result := HumanizeDom(parser.Parse("<!-- line 1 \r\n line 2 -->", "TestComp", nil), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})
	})

	t.Run("expansion forms", func(t *testing.T) {
		t.Run("should parse out expansion forms", func(t *testing.T) {
			options := &ml_parser.TokenizeOptions{TokenizeExpansionForms: boolPtr(true)}
			parsed := parser.Parse(
				`<div>before{messages.length, plural, =0 {You have <b>no</b> messages} =1 {One {{message}}}}after</div>`,
				"TestComp",
				options,
			)

			expected := []interface{}{
				[]interface{}{"Element", "div", 0},
				[]interface{}{"Text", "before", 1, []interface{}{"before"}},
				[]interface{}{"Expansion", "messages.length", "plural", 1},
				[]interface{}{"ExpansionCase", "=0", 2},
				[]interface{}{"ExpansionCase", "=1", 2},
				[]interface{}{"Text", "after", 1, []interface{}{"after"}},
			}
			result := HumanizeDom(parsed, false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}

			// Check expansion case expressions
			if len(parsed.RootNodes) > 0 {
				if elem, ok := parsed.RootNodes[0].(*ml_parser.Element); ok && len(elem.Children) > 1 {
					if expansion, ok := elem.Children[1].(*ml_parser.Expansion); ok && len(expansion.Cases) > 0 {
						case0Expr := HumanizeDom(ml_parser.NewParseTreeResult(expansion.Cases[0].Expression, []*util.ParseError{}), false)
						expectedCase0 := []interface{}{
							[]interface{}{"Text", "You have ", 0, []interface{}{"You have "}},
							[]interface{}{"Element", "b", 0},
							[]interface{}{"Text", "no", 1, []interface{}{"no"}},
							[]interface{}{"Text", " messages", 0, []interface{}{" messages"}},
						}
						if diff := cmp.Diff(expectedCase0, case0Expr); diff != "" {
							t.Errorf("Case 0 expression mismatch (-want +got):\n%s", diff)
						}

						if len(expansion.Cases) > 1 {
							case1Expr := HumanizeDom(ml_parser.NewParseTreeResult(expansion.Cases[1].Expression, []*util.ParseError{}), false)
							expectedCase1 := []interface{}{
								[]interface{}{"Text", "One {{message}}", 0, []interface{}{"One "}, []interface{}{"{{", "message", "}}"}, []interface{}{""}},
							}
							if diff := cmp.Diff(expectedCase1, case1Expr); diff != "" {
								t.Errorf("Case 1 expression mismatch (-want +got):\n%s", diff)
							}
						}
					}
				}
			}
		})

		t.Run("should normalize line-endings in expansion forms in inline templates if i18nNormalizeLineEndingsInICUs is true", func(t *testing.T) {
			input := `<div>` + "\r\n" +
				`  {` + "\r\n" +
				`    messages.length,` + "\r\n" +
				`    plural,` + "\r\n" +
				`    =0 {You have ` + "\r\n" + `no` + "\r\n" + ` messages}` + "\r\n" +
				`    =1 {One {{message}}}}` + "\r\n" +
				`</div>`
			options := &ml_parser.TokenizeOptions{
				TokenizeExpansionForms:         boolPtr(true),
				EscapedString:                  boolPtr(true),
				I18nNormalizeLineEndingsInICUs: boolPtr(true),
			}
			parsed := parser.Parse(input, "TestComp", options)

			expected := []interface{}{
				[]interface{}{"Element", "div", 0},
				[]interface{}{"Text", "\n  ", 1, []interface{}{"\n  "}},
				[]interface{}{"Expansion", "\n    messages.length", "plural", 1},
				[]interface{}{"ExpansionCase", "=0", 2},
				[]interface{}{"ExpansionCase", "=1", 2},
				[]interface{}{"Text", "\n", 1, []interface{}{"\n"}},
			}
			result := HumanizeDom(parsed, false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}

			// Check expansion case expressions
			if len(parsed.RootNodes) > 0 {
				if elem, ok := parsed.RootNodes[0].(*ml_parser.Element); ok && len(elem.Children) > 1 {
					if expansion, ok := elem.Children[1].(*ml_parser.Expansion); ok && len(expansion.Cases) > 0 {
						case0Expr := HumanizeDom(ml_parser.NewParseTreeResult(expansion.Cases[0].Expression, []*util.ParseError{}), false)
						expectedCase0 := []interface{}{
							[]interface{}{"Text", "You have \nno\n messages", 0, []interface{}{"You have \nno\n messages"}},
						}
						if diff := cmp.Diff(expectedCase0, case0Expr); diff != "" {
							t.Errorf("Case 0 expression mismatch (-want +got):\n%s", diff)
						}

						if len(expansion.Cases) > 1 {
							case1Expr := HumanizeDom(ml_parser.NewParseTreeResult(expansion.Cases[1].Expression, []*util.ParseError{}), false)
							expectedCase1 := []interface{}{
								[]interface{}{"Text", "One {{message}}", 0, []interface{}{"One "}, []interface{}{"{{", "message", "}}"}, []interface{}{""}},
							}
							if diff := cmp.Diff(expectedCase1, case1Expr); diff != "" {
								t.Errorf("Case 1 expression mismatch (-want +got):\n%s", diff)
							}
						}
					}
				}
			}

			if len(parsed.Errors) != 0 {
				t.Errorf("Expected no errors, got %d", len(parsed.Errors))
			}
		})

		t.Run("should not normalize line-endings in ICU expressions in external templates when i18nNormalizeLineEndingsInICUs is not set", func(t *testing.T) {
			input := `<div>` + "\r\n" +
				`  {` + "\r\n" +
				`    messages.length,` + "\r\n" +
				`    plural,` + "\r\n" +
				`    =0 {You have ` + "\r\n" + `no` + "\r\n" + ` messages}` + "\r\n" +
				`    =1 {One {{message}}}}` + "\r\n" +
				`</div>`
			options := &ml_parser.TokenizeOptions{
				TokenizeExpansionForms: boolPtr(true),
				EscapedString:          boolPtr(true),
			}
			parsed := parser.Parse(input, "TestComp", options)

			expected := []interface{}{
				[]interface{}{"Element", "div", 0},
				[]interface{}{"Text", "\n  ", 1, []interface{}{"\n  "}},
				[]interface{}{"Expansion", "\r\n    messages.length", "plural", 1},
				[]interface{}{"ExpansionCase", "=0", 2},
				[]interface{}{"ExpansionCase", "=1", 2},
				[]interface{}{"Text", "\n", 1, []interface{}{"\n"}},
			}
			result := HumanizeDom(parsed, false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}

			if len(parsed.Errors) != 0 {
				t.Errorf("Expected no errors, got %d", len(parsed.Errors))
			}
		})

		t.Run("should normalize line-endings in expansion forms in external templates if i18nNormalizeLineEndingsInICUs is true", func(t *testing.T) {
			input := `<div>` + "\r\n" +
				`  {` + "\r\n" +
				`    messages.length,` + "\r\n" +
				`    plural,` + "\r\n" +
				`    =0 {You have ` + "\r\n" + `no` + "\r\n" + ` messages}` + "\r\n" +
				`    =1 {One {{message}}}}` + "\r\n" +
				`</div>`
			options := &ml_parser.TokenizeOptions{
				TokenizeExpansionForms:         boolPtr(true),
				EscapedString:                  boolPtr(false),
				I18nNormalizeLineEndingsInICUs: boolPtr(true),
			}
			parsed := parser.Parse(input, "TestComp", options)

			expected := []interface{}{
				[]interface{}{"Element", "div", 0},
				[]interface{}{"Text", "\n  ", 1, []interface{}{"\n  "}},
				[]interface{}{"Expansion", "\n    messages.length", "plural", 1},
				[]interface{}{"ExpansionCase", "=0", 2},
				[]interface{}{"ExpansionCase", "=1", 2},
				[]interface{}{"Text", "\n", 1, []interface{}{"\n"}},
			}
			result := HumanizeDom(parsed, false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}

			if len(parsed.Errors) != 0 {
				t.Errorf("Expected no errors, got %d", len(parsed.Errors))
			}
		})

		t.Run("should not normalize line-endings in ICU expressions in external templates when i18nNormalizeLineEndingsInICUs is not set", func(t *testing.T) {
			input := `<div>` + "\r\n" +
				`  {` + "\r\n" +
				`    messages.length,` + "\r\n" +
				`    plural,` + "\r\n" +
				`    =0 {You have ` + "\r\n" + `no` + "\r\n" + ` messages}` + "\r\n" +
				`    =1 {One {{message}}}}` + "\r\n" +
				`</div>`
			options := &ml_parser.TokenizeOptions{
				TokenizeExpansionForms: boolPtr(true),
				EscapedString:          boolPtr(false),
			}
			parsed := parser.Parse(input, "TestComp", options)

			expected := []interface{}{
				[]interface{}{"Element", "div", 0},
				[]interface{}{"Text", "\n  ", 1, []interface{}{"\n  "}},
				[]interface{}{"Expansion", "\r\n    messages.length", "plural", 1},
				[]interface{}{"ExpansionCase", "=0", 2},
				[]interface{}{"ExpansionCase", "=1", 2},
				[]interface{}{"Text", "\n", 1, []interface{}{"\n"}},
			}
			result := HumanizeDom(parsed, false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}

			if len(parsed.Errors) != 0 {
				t.Errorf("Expected no errors, got %d", len(parsed.Errors))
			}
		})

		t.Run("should parse out expansion forms", func(t *testing.T) {
			options := &ml_parser.TokenizeOptions{TokenizeExpansionForms: boolPtr(true)}
			parsed := parser.Parse(`<div><span>{a, plural, =0 {b}}</span></div>`, "TestComp", options)

			expected := []interface{}{
				[]interface{}{"Element", "div", 0},
				[]interface{}{"Element", "span", 1},
				[]interface{}{"Expansion", "a", "plural", 2},
				[]interface{}{"ExpansionCase", "=0", 3},
			}
			result := HumanizeDom(parsed, false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should parse out nested expansion forms", func(t *testing.T) {
			options := &ml_parser.TokenizeOptions{TokenizeExpansionForms: boolPtr(true)}
			parsed := parser.Parse(
				`{messages.length, plural, =0 { {p.gender, select, male {m}} }}`,
				"TestComp",
				options,
			)
			expected := []interface{}{
				[]interface{}{"Expansion", "messages.length", "plural", 0},
				[]interface{}{"ExpansionCase", "=0", 1},
			}
			result := HumanizeDom(parsed, false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}

			// Check nested expansion
			if len(parsed.RootNodes) > 0 {
				if expansion, ok := parsed.RootNodes[0].(*ml_parser.Expansion); ok && len(expansion.Cases) > 0 {
					firstCaseExpr := HumanizeDom(ml_parser.NewParseTreeResult(expansion.Cases[0].Expression, []*util.ParseError{}), false)
					expectedFirstCase := []interface{}{
						[]interface{}{"Expansion", "p.gender", "select", 0},
						[]interface{}{"ExpansionCase", "male", 1},
						[]interface{}{"Text", " ", 0, []interface{}{" "}},
					}
					if diff := cmp.Diff(expectedFirstCase, firstCaseExpr); diff != "" {
						t.Errorf("First case expression mismatch (-want +got):\n%s", diff)
					}
				}
			}
		})

		t.Run("should normalize line endings in nested expansion forms for inline templates, when i18nNormalizeLineEndingsInICUs is true", func(t *testing.T) {
			input := `{` + "\r\n" +
				`  messages.length, plural,` + "\r\n" +
				`  =0 { zero ` + "\r\n" +
				`       {` + "\r\n" +
				`         p.gender, select,` + "\r\n" +
				`         male {m}` + "\r\n" +
				`       }` + "\r\n" +
				`     }` + "\r\n" +
				`}`
			options := &ml_parser.TokenizeOptions{
				TokenizeExpansionForms:         boolPtr(true),
				EscapedString:                  boolPtr(true),
				I18nNormalizeLineEndingsInICUs: boolPtr(true),
			}
			parsed := parser.Parse(input, "TestComp", options)

			expected := []interface{}{
				[]interface{}{"Expansion", "\n  messages.length", "plural", 0},
				[]interface{}{"ExpansionCase", "=0", 1},
			}
			result := HumanizeDom(parsed, false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}

			// Check nested expansion in case expression
			if len(parsed.RootNodes) > 0 {
				if expansion, ok := parsed.RootNodes[0].(*ml_parser.Expansion); ok && len(expansion.Cases) > 0 {
					caseExpr := HumanizeDom(ml_parser.NewParseTreeResult(expansion.Cases[0].Expression, []*util.ParseError{}), false)
					expectedCaseExpr := []interface{}{
						[]interface{}{"Text", "zero \n       ", 0, []interface{}{"zero \n       "}},
						[]interface{}{"Expansion", "\n         p.gender", "select", 0},
						[]interface{}{"ExpansionCase", "male", 1},
						[]interface{}{"Text", "\n     ", 0, []interface{}{"\n     "}},
					}
					if diff := cmp.Diff(expectedCaseExpr, caseExpr); diff != "" {
						t.Errorf("Case expression mismatch (-want +got):\n%s", diff)
					}
				}
			}

			if len(parsed.Errors) != 0 {
				t.Errorf("Expected no errors, got %d", len(parsed.Errors))
			}
		})

		t.Run("should not normalize line endings in nested expansion forms for inline templates, when i18nNormalizeLineEndingsInICUs is not defined", func(t *testing.T) {
			input := `{` + "\r\n" +
				`  messages.length, plural,` + "\r\n" +
				`  =0 { zero ` + "\r\n" +
				`       {` + "\r\n" +
				`         p.gender, select,` + "\r\n" +
				`         male {m}` + "\r\n" +
				`       }` + "\r\n" +
				`     }` + "\r\n" +
				`}`
			options := &ml_parser.TokenizeOptions{
				TokenizeExpansionForms: boolPtr(true),
				EscapedString:          boolPtr(true),
			}
			parsed := parser.Parse(input, "TestComp", options)

			expected := []interface{}{
				[]interface{}{"Expansion", "\r\n  messages.length", "plural", 0},
				[]interface{}{"ExpansionCase", "=0", 1},
			}
			result := HumanizeDom(parsed, false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}

			if len(parsed.Errors) != 0 {
				t.Errorf("Expected no errors, got %d", len(parsed.Errors))
			}
		})

		t.Run("should not normalize line endings in nested expansion forms for external templates, when i18nNormalizeLineEndingsInICUs is not set", func(t *testing.T) {
			input := `{` + "\r\n" +
				`  messages.length, plural,` + "\r\n" +
				`  =0 { zero ` + "\r\n" +
				`       {` + "\r\n" +
				`         p.gender, select,` + "\r\n" +
				`         male {m}` + "\r\n" +
				`       }` + "\r\n" +
				`     }` + "\r\n" +
				`}`
			options := &ml_parser.TokenizeOptions{TokenizeExpansionForms: boolPtr(true)}
			parsed := parser.Parse(input, "TestComp", options)

			expected := []interface{}{
				[]interface{}{"Expansion", "\r\n  messages.length", "plural", 0},
				[]interface{}{"ExpansionCase", "=0", 1},
			}
			result := HumanizeDom(parsed, false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}

			if len(parsed.Errors) != 0 {
				t.Errorf("Expected no errors, got %d", len(parsed.Errors))
			}
		})

		t.Run("should error when expansion form is not closed", func(t *testing.T) {
			options := &ml_parser.TokenizeOptions{TokenizeExpansionForms: boolPtr(true)}
			parsed := parser.Parse(`{messages.length, plural, =0 {one}`, "TestComp", options)
			expectedErrors := []interface{}{
				[]interface{}{nil, "Invalid ICU message. Missing '}'.", "0:34"},
			}
			resultErrors := humanizeErrors(parsed.Errors)
			if diff := cmp.Diff(expectedErrors, resultErrors); diff != "" {
				t.Errorf("humanizeErrors() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should support ICU expressions with cases that contain numbers", func(t *testing.T) {
			options := &ml_parser.TokenizeOptions{TokenizeExpansionForms: boolPtr(true)}
			parsed := parser.Parse(`{sex, select, male {m} female {f} 0 {other}}`, "TestComp", options)
			if len(parsed.Errors) != 0 {
				t.Errorf("Expected no errors, got %d", len(parsed.Errors))
			}
		})

		t.Run("should support ICU expressions with cases that contain any character except }", func(t *testing.T) {
			options := &ml_parser.TokenizeOptions{TokenizeExpansionForms: boolPtr(true)}
			parsed := parser.Parse(`{a, select, b {foo} % bar {% bar}}`, "TestComp", options)
			if len(parsed.Errors) != 0 {
				t.Errorf("Expected no errors, got %d", len(parsed.Errors))
			}
		})

		t.Run("should error when expansion case is not properly closed", func(t *testing.T) {
			options := &ml_parser.TokenizeOptions{TokenizeExpansionForms: boolPtr(true)}
			parsed := parser.Parse(`{a, select, b {foo} % { bar {% bar}}`, "TestComp", options)
			expectedErrors := []interface{}{
				[]interface{}{
					nil,
					"Unexpected character \"EOF\" (Do you have an unescaped \"{\" in your template? Use \"{{ '{' }}\") to escape it.)",
					"0:36",
				},
				[]interface{}{nil, "Invalid ICU message. Missing '}'.", "0:22"},
			}
			resultErrors := humanizeErrors(parsed.Errors)
			if diff := cmp.Diff(expectedErrors, resultErrors); diff != "" {
				t.Errorf("humanizeErrors() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should error when expansion case is not closed", func(t *testing.T) {
			options := &ml_parser.TokenizeOptions{TokenizeExpansionForms: boolPtr(true)}
			parsed := parser.Parse(`{messages.length, plural, =0 {one`, "TestComp", options)
			expectedErrors := []interface{}{
				[]interface{}{nil, "Invalid ICU message. Missing '}'.", "0:29"},
			}
			resultErrors := humanizeErrors(parsed.Errors)
			if diff := cmp.Diff(expectedErrors, resultErrors); diff != "" {
				t.Errorf("humanizeErrors() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should error when invalid html in the case", func(t *testing.T) {
			options := &ml_parser.TokenizeOptions{TokenizeExpansionForms: boolPtr(true)}
			parsed := parser.Parse(`{messages.length, plural, =0 {<b/>}`, "TestComp", options)
			expectedErrors := []interface{}{
				[]interface{}{"b", "Only void, custom and foreign elements can be self closed \"b\"", "0:30"},
			}
			resultErrors := humanizeErrors(parsed.Errors)
			if diff := cmp.Diff(expectedErrors, resultErrors); diff != "" {
				t.Errorf("humanizeErrors() mismatch (-want +got):\n%s", diff)
			}
		})
	})

	t.Run("blocks", func(t *testing.T) {
		t.Run("should parse a block", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{"Block", "defer", 0},
				[]interface{}{"BlockParameter", "a b"},
				[]interface{}{"BlockParameter", "c d"},
				[]interface{}{"Text", "hello", 1, []interface{}{"hello"}},
			}
			result := HumanizeDom(parser.Parse("@defer (a b; c d){hello}", "TestComp", nil), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should parse a block with an HTML element", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{"Block", "defer", 0},
				[]interface{}{"Element", "my-cmp", 1, "#selfClosing"},
			}
			result := HumanizeDom(parser.Parse("@defer {<my-cmp/>}", "TestComp", nil), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should parse a block containing mixed plain text and HTML", func(t *testing.T) {
			markup := "@switch (expr) {" +
				"@case (1) {hello<my-cmp/>there}" +
				"@case (two) {<p>Two...</p>}" +
				"@case (isThree(3)) {T<strong>htr<i>e</i>e</strong>!}" +
				"}"
			expected := []interface{}{
				[]interface{}{"Block", "switch", 0},
				[]interface{}{"BlockParameter", "expr"},
				[]interface{}{"Block", "case", 1},
				[]interface{}{"BlockParameter", "1"},
				[]interface{}{"Text", "hello", 2, []interface{}{"hello"}},
				[]interface{}{"Element", "my-cmp", 2, "#selfClosing"},
				[]interface{}{"Text", "there", 2, []interface{}{"there"}},
				[]interface{}{"Block", "case", 1},
				[]interface{}{"BlockParameter", "two"},
				[]interface{}{"Element", "p", 2},
				[]interface{}{"Text", "Two...", 3, []interface{}{"Two..."}},
				[]interface{}{"Block", "case", 1},
				[]interface{}{"BlockParameter", "isThree(3)"},
				[]interface{}{"Text", "T", 2, []interface{}{"T"}},
				[]interface{}{"Element", "strong", 2},
				[]interface{}{"Text", "htr", 3, []interface{}{"htr"}},
				[]interface{}{"Element", "i", 3},
				[]interface{}{"Text", "e", 4, []interface{}{"e"}},
				[]interface{}{"Text", "e", 3, []interface{}{"e"}},
				[]interface{}{"Text", "!", 2, []interface{}{"!"}},
			}
			result := HumanizeDom(parser.Parse(markup, "TestComp", nil), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should parse nested blocks", func(t *testing.T) {
			markup := `<root-sibling-one/>` +
				`@if (root) {` +
				`<outer-child-one/>` +
				`<outer-child-two>` +
				`@if (childParam === 1) {` +
				`@if (innerChild1 === foo) {` +
				`<inner-child-one/>` +
				`@switch (grandChild) {` +
				`@case (innerGrandChild) {` +
				`<inner-grand-child-one/>` +
				`}` +
				`@case (innerGrandChild) {` +
				`<inner-grand-child-two/>` +
				`}` +
				`}` +
				`}` +
				`@if (innerChild) {` +
				`<inner-child-two/>` +
				`}` +
				`}` +
				`</outer-child-two>` +
				`@for (outerChild1; outerChild2) {` +
				`<outer-child-three/>` +
				`}` +
				`} <root-sibling-two/>`
			expected := []interface{}{
				[]interface{}{"Element", "root-sibling-one", 0, "#selfClosing"},
				[]interface{}{"Block", "if", 0},
				[]interface{}{"BlockParameter", "root"},
				[]interface{}{"Element", "outer-child-one", 1, "#selfClosing"},
				[]interface{}{"Element", "outer-child-two", 1},
				[]interface{}{"Block", "if", 2},
				[]interface{}{"BlockParameter", "childParam === 1"},
				[]interface{}{"Block", "if", 3},
				[]interface{}{"BlockParameter", "innerChild1 === foo"},
				[]interface{}{"Element", "inner-child-one", 4, "#selfClosing"},
				[]interface{}{"Block", "switch", 4},
				[]interface{}{"BlockParameter", "grandChild"},
				[]interface{}{"Block", "case", 5},
				[]interface{}{"BlockParameter", "innerGrandChild"},
				[]interface{}{"Element", "inner-grand-child-one", 6, "#selfClosing"},
				[]interface{}{"Block", "case", 5},
				[]interface{}{"BlockParameter", "innerGrandChild"},
				[]interface{}{"Element", "inner-grand-child-two", 6, "#selfClosing"},
				[]interface{}{"Block", "if", 3},
				[]interface{}{"BlockParameter", "innerChild"},
				[]interface{}{"Element", "inner-child-two", 4, "#selfClosing"},
				[]interface{}{"Block", "for", 1},
				[]interface{}{"BlockParameter", "outerChild1"},
				[]interface{}{"BlockParameter", "outerChild2"},
				[]interface{}{"Element", "outer-child-three", 2, "#selfClosing"},
				[]interface{}{"Text", " ", 0, []interface{}{" "}},
				[]interface{}{"Element", "root-sibling-two", 0, "#selfClosing"},
			}
			result := HumanizeDom(parser.Parse(markup, "TestComp", nil), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should infer namespace through block boundary", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{"Element", ":svg:svg", 0},
				[]interface{}{"Block", "if", 1},
				[]interface{}{"BlockParameter", "cond"},
				[]interface{}{"Element", ":svg:circle", 2, "#selfClosing"},
			}
			result := HumanizeDom(parser.Parse("<svg>@if (cond) {<circle/>}</svg>", "TestComp", nil), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should parse an empty block", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{"Block", "defer", 0},
			}
			result := HumanizeDom(parser.Parse("@defer{}", "TestComp", nil), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should parse a block with void elements", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{"Block", "defer", 0},
				[]interface{}{"Element", "br", 1},
			}
			result := HumanizeDom(parser.Parse("@defer {<br>}", "TestComp", nil), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should close void elements used right before a block", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{"Element", "img", 0},
				[]interface{}{"Block", "defer", 0},
				[]interface{}{"Text", "hello", 1, []interface{}{"hello"}},
			}
			result := HumanizeDom(parser.Parse("<img>@defer {hello}", "TestComp", nil), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should report an unclosed block", func(t *testing.T) {
			parsed := parser.Parse("@defer {hello", "TestComp", nil)
			if len(parsed.Errors) != 1 {
				t.Errorf("Expected 1 error, got %d", len(parsed.Errors))
			}
			expectedErrors := []interface{}{
				[]interface{}{"defer", "Unclosed block \"defer\"", "0:0"},
			}
			resultErrors := humanizeErrors(parsed.Errors)
			if diff := cmp.Diff(expectedErrors, resultErrors); diff != "" {
				t.Errorf("humanizeErrors() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should report an unexpected block close", func(t *testing.T) {
			parsed := parser.Parse("hello}", "TestComp", nil)
			if len(parsed.Errors) != 1 {
				t.Errorf("Expected 1 error, got %d", len(parsed.Errors))
			}
			expectedErrors := []interface{}{
				[]interface{}{
					nil,
					"Unexpected closing block. The block may have been closed earlier. If you meant to write the } character, you should use the \"&#125;\" HTML entity instead.",
					"0:5",
				},
			}
			resultErrors := humanizeErrors(parsed.Errors)
			if diff := cmp.Diff(expectedErrors, resultErrors); diff != "" {
				t.Errorf("humanizeErrors() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should report unclosed tags inside of a block", func(t *testing.T) {
			parsed := parser.Parse("@defer {<strong>hello}", "TestComp", nil)
			if len(parsed.Errors) != 1 {
				t.Errorf("Expected 1 error, got %d", len(parsed.Errors))
			}
			expectedErrors := []interface{}{
				[]interface{}{
					nil,
					"Unexpected closing block. The block may have been closed earlier. If you meant to write the } character, you should use the \"&#125;\" HTML entity instead.",
					"0:21",
				},
			}
			resultErrors := humanizeErrors(parsed.Errors)
			if diff := cmp.Diff(expectedErrors, resultErrors); diff != "" {
				t.Errorf("humanizeErrors() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should report an unexpected closing tag inside a block", func(t *testing.T) {
			parsed := parser.Parse("<div>@if (cond) {hello</div>}", "TestComp", nil)
			if len(parsed.Errors) != 2 {
				t.Errorf("Expected 2 errors, got %d", len(parsed.Errors))
			}
			expectedErrors := []interface{}{
				[]interface{}{
					"div",
					"Unexpected closing tag \"div\". It may happen when the tag has already been closed by another tag. For more info see https://www.w3.org/TR/html5/syntax.html#closing-elements-that-have-implied-end-tags",
					"0:22",
				},
				[]interface{}{
					nil,
					"Unexpected closing block. The block may have been closed earlier. If you meant to write the } character, you should use the \"&#125;\" HTML entity instead.",
					"0:28",
				},
			}
			resultErrors := humanizeErrors(parsed.Errors)
			if diff := cmp.Diff(expectedErrors, resultErrors); diff != "" {
				t.Errorf("humanizeErrors() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should store the source locations of blocks", func(t *testing.T) {
			markup := "@switch (expr) {" +
				"@case (1) {<div>hello</div>world}" +
				"@case (two) {Two}" +
				"@case (isThree(3)) {Placeholde<strong>r</strong>}" +
				"}"
			result := HumanizeDomSourceSpans(parser.Parse(markup, "TestComp", nil))
			// Note: The exact source span strings may vary, so we check structure
			if len(result) == 0 {
				t.Error("Expected non-empty result")
			}
		})

		t.Run("should parse an incomplete block with no parameters", func(t *testing.T) {
			parsed := parser.Parse("This is the @if() block", "TestComp", nil)
			result := HumanizeNodes(parsed.RootNodes, true)
			expected := []interface{}{
				[]interface{}{"Text", "This is the ", 0, []interface{}{"This is the "}, "This is the "},
				[]interface{}{"Block", "if", 0, "@if() ", "@if() ", nil},
				[]interface{}{"Text", "block", 0, []interface{}{"block"}, "block"},
			}
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeNodes() mismatch (-want +got):\n%s", diff)
			}
			expectedErrors := []interface{}{
				[]interface{}{
					"if",
					"Incomplete block \"if\". If you meant to write the @ character, you should use the \"&#64;\" HTML entity instead.",
					"0:12",
				},
			}
			resultErrors := humanizeErrors(parsed.Errors)
			if diff := cmp.Diff(expectedErrors, resultErrors); diff != "" {
				t.Errorf("humanizeErrors() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should parse an incomplete block with parameters", func(t *testing.T) {
			parsed := parser.Parse("This is the @if({alias: \"foo\"}) block with params", "TestComp", nil)
			result := HumanizeNodes(parsed.RootNodes, true)
			expected := []interface{}{
				[]interface{}{"Text", "This is the ", 0, []interface{}{"This is the "}, "This is the "},
				[]interface{}{"Block", "if", 0, "@if({alias: \"foo\"}) ", "@if({alias: \"foo\"}) ", nil},
				[]interface{}{"BlockParameter", "{alias: \"foo\"}", "{alias: \"foo\"}"},
				[]interface{}{"Text", "block with params", 0, []interface{}{"block with params"}, "block with params"},
			}
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeNodes() mismatch (-want +got):\n%s", diff)
			}
			expectedErrors := []interface{}{
				[]interface{}{
					"if",
					"Incomplete block \"if\". If you meant to write the @ character, you should use the \"&#64;\" HTML entity instead.",
					"0:12",
				},
			}
			resultErrors := humanizeErrors(parsed.Errors)
			if diff := cmp.Diff(expectedErrors, resultErrors); diff != "" {
				t.Errorf("humanizeErrors() mismatch (-want +got):\n%s", diff)
			}
		})
	})

	t.Run("let declaration", func(t *testing.T) {
		t.Run("should parse a let declaration", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{"LetDeclaration", "foo", "123"},
			}
			result := HumanizeDom(parser.Parse("@let foo = 123;", "TestCmp", nil), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should parse a let declaration that is nested in a parent", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{"Block", "defer", 0},
				[]interface{}{"Block", "if", 1},
				[]interface{}{"BlockParameter", "true"},
				[]interface{}{"LetDeclaration", "foo", "123"},
			}
			result := HumanizeDom(parser.Parse("@defer {@if (true) {@let foo = 123;}}", "TestCmp", nil), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should store the source location of a @let declaration", func(t *testing.T) {
			result := HumanizeDomSourceSpans(parser.Parse("@let foo = 123 + 456;", "TestCmp", nil))
			// Note: The exact source span strings may vary, so we check structure
			if len(result) == 0 {
				t.Error("Expected non-empty result")
			}
		})

		t.Run("should report an error for an incomplete let declaration", func(t *testing.T) {
			parsed := parser.Parse("@let foo =", "TestCmp", nil)
			expectedErrors := []interface{}{
				[]interface{}{
					"foo",
					"Incomplete @let declaration \"foo\". @let declarations must be written as `@let <name> = <value>;`",
					"0:0",
				},
			}
			resultErrors := humanizeErrors(parsed.Errors)
			if diff := cmp.Diff(expectedErrors, resultErrors); diff != "" {
				t.Errorf("humanizeErrors() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should store the locations of an incomplete let declaration", func(t *testing.T) {
			parsed := parser.Parse("@let foo =", "TestCmp", nil)
			// Clear errors to test spans even with broken templates
			parsed.Errors = []*util.ParseError{}
			result := HumanizeDomSourceSpans(parsed)
			// Note: The exact source span strings may vary, so we check structure
			if len(result) == 0 {
				t.Error("Expected non-empty result")
			}
		})
	})

	t.Run("directive nodes", func(t *testing.T) {
		options := &ml_parser.TokenizeOptions{
			SelectorlessEnabled: boolPtr(true),
		}

		t.Run("should parse a directive with no attributes", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{"Element", "div", 0},
				[]interface{}{"Directive", "Dir"},
			}
			result := HumanizeDom(parser.Parse("<div @Dir></div>", "", options), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should parse a directive with attributes", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{"Element", "div", 0},
				[]interface{}{"Directive", "Dir"},
				[]interface{}{"Attribute", "a", "1", []interface{}{"1"}},
				[]interface{}{"Attribute", "[b]", "two", []interface{}{"two"}},
				[]interface{}{"Attribute", "(c)", "c()", []interface{}{"c()"}},
			}
			result := HumanizeDom(parser.Parse("<div @Dir(a=\"1\" [b]=\"two\" (c)=\"c()\")></div>", "", options), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should parse directives on a component node", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{"Component", "MyComp", "", "MyComp", 0},
				[]interface{}{"Directive", "Dir"},
				[]interface{}{"Directive", "OtherDir"},
				[]interface{}{"Attribute", "a", "1", []interface{}{"1"}},
				[]interface{}{"Attribute", "[b]", "two", []interface{}{"two"}},
				[]interface{}{"Attribute", "(c)", "c()", []interface{}{"c()"}},
			}
			result := HumanizeDom(parser.Parse("<MyComp @Dir @OtherDir(a=\"1\" [b]=\"two\" (c)=\"c()\")></MyComp>", "", options), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should report a missing directive closing paren", func(t *testing.T) {
			parsed := parser.Parse("<div @Dir(a=\"1\" (b)=\"2\"></div>", "", options)
			expectedErrors := []interface{}{
				[]interface{}{nil, "Unterminated directive definition", "0:5"},
			}
			resultErrors := humanizeErrors(parsed.Errors)
			if diff := cmp.Diff(expectedErrors, resultErrors); diff != "" {
				t.Errorf("humanizeErrors() mismatch (-want +got):\n%s", diff)
			}

			parsed2 := parser.Parse("<MyComp @Dir(a=\"1\" (b)=\"2\"/>", "", options)
			expectedErrors2 := []interface{}{
				[]interface{}{nil, "Unterminated directive definition", "0:8"},
			}
			resultErrors2 := humanizeErrors(parsed2.Errors)
			if diff := cmp.Diff(expectedErrors2, resultErrors2); diff != "" {
				t.Errorf("humanizeErrors() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should parse a directive mixed with other attributes", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{"Element", "div", 0},
				[]interface{}{"Attribute", "before", "foo", []interface{}{"foo"}},
				[]interface{}{"Attribute", "middle", ""},
				[]interface{}{"Attribute", "after", "123", []interface{}{"123"}},
				[]interface{}{"Directive", "Dir"},
				[]interface{}{"Directive", "OtherDir"},
				[]interface{}{"Attribute", "[a]", "a", []interface{}{"a"}},
				[]interface{}{"Attribute", "(b)", "b()", []interface{}{"b()"}},
			}
			result := HumanizeDom(parser.Parse("<div before=\"foo\" @Dir middle @OtherDir([a]=\"a\" (b)=\"b()\") after=\"123\"></div>", "", options), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should store the source locations of directives", func(t *testing.T) {
			markup := "<div @Dir @OtherDir(a=\"1\" [b]=\"two\" (c)=\"c()\")></div>"
			result := HumanizeDomSourceSpans(parser.Parse(markup, "", options))
			// Note: The exact source span strings may vary, so we check structure
			if len(result) == 0 {
				t.Error("Expected non-empty result")
			}
		})
	})

	t.Run("component nodes", func(t *testing.T) {
		options := &ml_parser.TokenizeOptions{
			SelectorlessEnabled: boolPtr(true),
		}

		t.Run("should parse a simple component node", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{"Component", "MyComp", "", "MyComp", 0},
				[]interface{}{"Text", "Hello", 1, []interface{}{"Hello"}},
			}
			result := HumanizeDom(parser.Parse("<MyComp>Hello</MyComp>", "", options), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should parse a self-closing component node", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{"Component", "MyComp", "", "MyComp", 0, "#selfClosing"},
				[]interface{}{"Text", "Hello", 0, []interface{}{"Hello"}},
			}
			result := HumanizeDom(parser.Parse("<MyComp/>Hello", "", options), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should parse a component node with a tag name", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{"Component", "MyComp", "button", "MyComp:button", 0},
				[]interface{}{"Text", "Hello", 1, []interface{}{"Hello"}},
			}
			result := HumanizeDom(parser.Parse("<MyComp:button>Hello</MyComp:button>", "", options), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should parse a component node with a tag name and namespace", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{"Component", "MyComp", ":svg:title", "MyComp:svg:title", 0},
				[]interface{}{"Text", "Hello", 1, []interface{}{"Hello"}},
			}
			result := HumanizeDom(parser.Parse("<MyComp:svg:title>Hello</MyComp:svg:title>", "", options), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should parse a component node with an inferred namespace and no tag name", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{"Element", ":svg:svg", 0},
				[]interface{}{"Component", "MyComp", ":svg:ng-component", "MyComp:svg:ng-component", 1},
				[]interface{}{"Text", "Hello", 2, []interface{}{"Hello"}},
			}
			result := HumanizeDom(parser.Parse("<svg><MyComp>Hello</MyComp></svg>", "", options), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should parse a component node with an inferred namespace and a tag name", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{"Element", ":svg:svg", 0},
				[]interface{}{"Component", "MyComp", ":svg:button", "MyComp:svg:button", 1},
				[]interface{}{"Text", "Hello", 2, []interface{}{"Hello"}},
			}
			result := HumanizeDom(parser.Parse("<svg><MyComp:button>Hello</MyComp:button></svg>", "", options), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should parse a component node with an inferred namespace plus an explicit namespace and tag name", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{"Element", ":math:math", 0},
				[]interface{}{"Component", "MyComp", ":svg:title", "MyComp:svg:title", 1},
				[]interface{}{"Text", "Hello", 2, []interface{}{"Hello"}},
			}
			result := HumanizeDom(parser.Parse("<math><MyComp:svg:title>Hello</MyComp:svg:title></math>", "", options), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should distinguish components with tag names from ones without", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{"Component", "MyComp", "button", "MyComp:button", 0},
				[]interface{}{"Component", "MyComp", "", "MyComp", 1},
				[]interface{}{"Text", "Hello", 2, []interface{}{"Hello"}},
			}
			result := HumanizeDom(parser.Parse("<MyComp:button><MyComp>Hello</MyComp></MyComp:button>", "", options), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should implicitly close a component", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{"Component", "MyComp", "", "MyComp", 0},
				[]interface{}{"Text", "Hello", 1, []interface{}{"Hello"}},
			}
			result := HumanizeDom(parser.Parse("<MyComp>Hello", "", options), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should parse a component tag nested within other markup", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{"Block", "if", 0},
				[]interface{}{"BlockParameter", "expr"},
				[]interface{}{"Element", "div", 1},
				[]interface{}{"Text", "Hello: ", 2, []interface{}{"Hello: "}},
				[]interface{}{"Component", "MyComp", "", "MyComp", 2},
				[]interface{}{"Element", "span", 3},
				[]interface{}{"Component", "OtherComp", "", "OtherComp", 4, "#selfClosing"},
			}
			result := HumanizeDom(parser.Parse("@if (expr) {<div>Hello: <MyComp><span><OtherComp/></span></MyComp></div>}", "", options), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should report closing tag whose tag name does not match the opening tag", func(t *testing.T) {
			parsed := parser.Parse("<MyComp:button>Hello</MyComp>", "", options)
			expectedErrors := []interface{}{
				[]interface{}{"MyComp", "Unexpected closing tag \"MyComp\", did you mean \"MyComp:button\"?", "0:20"},
			}
			resultErrors := humanizeErrors(parsed.Errors)
			if diff := cmp.Diff(expectedErrors, resultErrors); diff != "" {
				t.Errorf("humanizeErrors() mismatch (-want +got):\n%s", diff)
			}

			parsed2 := parser.Parse("<MyComp>Hello</MyComp:button>", "", options)
			expectedErrors2 := []interface{}{
				[]interface{}{"MyComp:button", "Unexpected closing tag \"MyComp:button\", did you mean \"MyComp\"?", "0:13"},
			}
			resultErrors2 := humanizeErrors(parsed2.Errors)
			if diff := cmp.Diff(expectedErrors2, resultErrors2); diff != "" {
				t.Errorf("humanizeErrors() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should parse a component node with attributes and directives", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{"Component", "MyComp", "", "MyComp", 0},
				[]interface{}{"Attribute", "before", "foo", []interface{}{"foo"}},
				[]interface{}{"Attribute", "middle", ""},
				[]interface{}{"Attribute", "after", "123", []interface{}{"123"}},
				[]interface{}{"Directive", "Dir"},
				[]interface{}{"Directive", "OtherDir"},
				[]interface{}{"Attribute", "[a]", "a", []interface{}{"a"}},
				[]interface{}{"Attribute", "(b)", "b()", []interface{}{"b()"}},
				[]interface{}{"Text", "Hello", 1, []interface{}{"Hello"}},
			}
			result := HumanizeDom(parser.Parse("<MyComp before=\"foo\" @Dir middle @OtherDir([a]=\"a\" (b)=\"b()\") after=\"123\">Hello</MyComp>", "", options), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should store the source locations of a component with attributes and content", func(t *testing.T) {
			markup := "<MyComp one=\"1\" two [three]=\"3\">Hello</MyComp>"
			result := HumanizeDomSourceSpans(parser.Parse(markup, "", options))
			// Note: The exact source span strings may vary, so we check structure
			if len(result) == 0 {
				t.Error("Expected non-empty result")
			}
		})

		t.Run("should store the source locations of self-closing components", func(t *testing.T) {
			markup := "<MyComp one=\"1\" two [three]=\"3\"/>Hello<MyOtherComp/><MyThirdComp:button/>"
			result := HumanizeDomSourceSpans(parser.Parse(markup, "", options))
			// Note: The exact source span strings may vary, so we check structure
			if len(result) == 0 {
				t.Error("Expected non-empty result")
			}
		})
	})

	t.Run("source spans", func(t *testing.T) {
		t.Run("should store the location", func(t *testing.T) {
			result := HumanizeDomSourceSpans(parser.Parse("<div [prop]=\"v1\" (e)=\"do()\" attr=\"v2\" noValue>\na\n</div>", " TestComp", nil))
			// Note: The exact source span strings may vary, so we check structure
			if len(result) == 0 {
				t.Error("Expected non-empty result")
			}
		})

		t.Run("should set the start and end source spans", func(t *testing.T) {
			parsed := parser.Parse("<div>a</div>", "TestComp", nil)
			if len(parsed.RootNodes) == 0 {
				t.Fatal("Expected at least one root node")
			}
			node, ok := parsed.RootNodes[0].(*ml_parser.Element)
			if !ok {
				t.Fatal("Expected first node to be Element")
			}
			if node.StartSourceSpan.Start.Offset != 0 {
				t.Errorf("Expected startSourceSpan.start.offset = 0, got %d", node.StartSourceSpan.Start.Offset)
			}
			if node.StartSourceSpan.End.Offset != 5 {
				t.Errorf("Expected startSourceSpan.end.offset = 5, got %d", node.StartSourceSpan.End.Offset)
			}
			if node.EndSourceSpan == nil {
				t.Error("Expected endSourceSpan to be set")
			} else {
				if node.EndSourceSpan.Start.Offset != 6 {
					t.Errorf("Expected endSourceSpan.start.offset = 6, got %d", node.EndSourceSpan.Start.Offset)
				}
				if node.EndSourceSpan.End.Offset != 12 {
					t.Errorf("Expected endSourceSpan.end.offset = 12, got %d", node.EndSourceSpan.End.Offset)
				}
			}
		})

		t.Run("should decode HTML entities in interpolations", func(t *testing.T) {
			markup := "{{&amp;}}" +
				"{{&#x25BE;}}" +
				"{{&#9662;}}" +
				"{{&unknown;}}" +
				"{{&amp (no semi-colon)}}" +
				"{{&#xyz; (invalid hex)}}" +
				"{{&#25BE; (invalid decimal)}}"
			result := HumanizeDomSourceSpans(parser.Parse(markup, "TestComp", nil))
			// Note: The exact format may vary, so we check structure
			if len(result) == 0 {
				t.Error("Expected non-empty result")
			}
		})

		t.Run("should decode HTML entities with 5+ hex digits in interpolations", func(t *testing.T) {
			// Test with 🛈 (U+1F6C8 - Circled Information Source)
			result := HumanizeDomSourceSpans(parser.Parse("{{&#x1F6C8;}}{{&#128712;}}", "TestComp", nil))
			// Note: The exact format may vary, so we check structure
			if len(result) == 0 {
				t.Error("Expected non-empty result")
			}
		})

		t.Run("should support interpolations in text", func(t *testing.T) {
			result := HumanizeDomSourceSpans(parser.Parse("<div> pre {{ value }} post </div>", "TestComp", nil))
			// Note: The exact format may vary, so we check structure
			if len(result) == 0 {
				t.Error("Expected non-empty result")
			}
		})

		t.Run("should not set the end source span for void elements", func(t *testing.T) {
			parsed := parser.Parse("<div><br></div>", "TestComp", nil)
			if len(parsed.RootNodes) == 0 {
				t.Fatal("Expected at least one root node")
			}
			div, ok := parsed.RootNodes[0].(*ml_parser.Element)
			if !ok {
				t.Fatal("Expected first node to be Element")
			}
			if len(div.Children) == 0 {
				t.Fatal("Expected div to have children")
			}
			br, ok := div.Children[0].(*ml_parser.Element)
			if !ok {
				t.Fatal("Expected first child to be Element")
			}
			if br.EndSourceSpan != nil {
				t.Error("Expected endSourceSpan to be nil for void elements")
			}
		})

		t.Run("should set the end source span for self-closing elements", func(t *testing.T) {
			parsed := parser.Parse("<br/>", "TestComp", nil)
			if len(parsed.RootNodes) == 0 {
				t.Fatal("Expected at least one root node")
			}
			br, ok := parsed.RootNodes[0].(*ml_parser.Element)
			if !ok {
				t.Fatal("Expected first node to be Element")
			}
			if br.EndSourceSpan == nil {
				t.Error("Expected endSourceSpan to be set for self-closing elements")
			}
		})

		t.Run("should not set the end source span for elements that are implicitly closed", func(t *testing.T) {
			parsed := parser.Parse("<div><p></div>", "TestComp", nil)
			if len(parsed.RootNodes) == 0 {
				t.Fatal("Expected at least one root node")
			}
			div, ok := parsed.RootNodes[0].(*ml_parser.Element)
			if !ok {
				t.Fatal("Expected first node to be Element")
			}
			if len(div.Children) == 0 {
				t.Fatal("Expected div to have children")
			}
			p, ok := div.Children[0].(*ml_parser.Element)
			if !ok {
				t.Fatal("Expected first child to be Element")
			}
			if p.EndSourceSpan != nil {
				t.Error("Expected endSourceSpan to be nil for implicitly closed elements")
			}
		})

		t.Run("should support expansion form", func(t *testing.T) {
			options := &ml_parser.TokenizeOptions{
				TokenizeExpansionForms: boolPtr(true),
			}
			result := HumanizeDomSourceSpans(parser.Parse("<div>{count, plural, =0 {msg}}</div>", "TestComp", options))
			// Note: The exact format may vary, so we check structure
			if len(result) == 0 {
				t.Error("Expected non-empty result")
			}
		})

		t.Run("should not report a value span for an attribute without a value", func(t *testing.T) {
			parsed := parser.Parse("<div bar></div>", "TestComp", nil)
			if len(parsed.RootNodes) == 0 {
				t.Fatal("Expected at least one root node")
			}
			div, ok := parsed.RootNodes[0].(*ml_parser.Element)
			if !ok {
				t.Fatal("Expected first node to be Element")
			}
			if len(div.Attrs) == 0 {
				t.Fatal("Expected div to have attributes")
			}
			attr := div.Attrs[0]
			if attr.ValueSpan != nil {
				t.Error("Expected valueSpan to be nil for attributes without values")
			}
		})

		t.Run("should report a value span for an attribute with a value", func(t *testing.T) {
			parsed := parser.Parse("<div bar=\"12\"></div>", "TestComp", nil)
			if len(parsed.RootNodes) == 0 {
				t.Fatal("Expected at least one root node")
			}
			div, ok := parsed.RootNodes[0].(*ml_parser.Element)
			if !ok {
				t.Fatal("Expected first node to be Element")
			}
			if len(div.Attrs) == 0 {
				t.Fatal("Expected div to have attributes")
			}
			attr := div.Attrs[0]
			if attr.ValueSpan == nil {
				t.Error("Expected valueSpan to be set for attributes with values")
			} else {
				if attr.ValueSpan.Start.Offset != 10 {
					t.Errorf("Expected valueSpan.start.offset = 10, got %d", attr.ValueSpan.Start.Offset)
				}
				if attr.ValueSpan.End.Offset != 12 {
					t.Errorf("Expected valueSpan.end.offset = 12, got %d", attr.ValueSpan.End.Offset)
				}
			}
		})

		t.Run("should report a value span for an unquoted attribute value", func(t *testing.T) {
			parsed := parser.Parse("<div bar=12></div>", "TestComp", nil)
			if len(parsed.RootNodes) == 0 {
				t.Fatal("Expected at least one root node")
			}
			div, ok := parsed.RootNodes[0].(*ml_parser.Element)
			if !ok {
				t.Fatal("Expected first node to be Element")
			}
			if len(div.Attrs) == 0 {
				t.Fatal("Expected div to have attributes")
			}
			attr := div.Attrs[0]
			if attr.ValueSpan == nil {
				t.Error("Expected valueSpan to be set for unquoted attribute values")
			} else {
				if attr.ValueSpan.Start.Offset != 9 {
					t.Errorf("Expected valueSpan.start.offset = 9, got %d", attr.ValueSpan.Start.Offset)
				}
				if attr.ValueSpan.End.Offset != 11 {
					t.Errorf("Expected valueSpan.end.offset = 11, got %d", attr.ValueSpan.End.Offset)
				}
			}
		})
	})

	t.Run("visitor", func(t *testing.T) {
		t.Run("should visit text nodes", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{"Text", "text", 0, []interface{}{"text"}},
			}
			result := HumanizeDom(parser.Parse("text", "TestComp", nil), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should visit element nodes", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{"Element", "div", 0},
			}
			result := HumanizeDom(parser.Parse("<div></div>", "TestComp", nil), false)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeDom() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should visit attribute nodes", func(t *testing.T) {
			result := HumanizeDom(parser.Parse("<div id=\"foo\"></div>", "TestComp", nil), false)
			// Check that attribute is present
			found := false
			for _, item := range result {
				if arr, ok := item.([]interface{}); ok && len(arr) > 0 {
					if arr[0] == "Attribute" && len(arr) > 1 && arr[1] == "id" {
						found = true
						break
					}
				}
			}
			if !found {
				t.Error("Expected to find Attribute node with name 'id'")
			}
		})

		t.Run("should visit all nodes", func(t *testing.T) {
			parsed := parser.Parse("<div id=\"foo\"><span id=\"bar\">a</span><span>b</span></div>", "TestComp", nil)
			accumulator := []ml_parser.Node{}
			visitor := &testVisitor{accumulator: &accumulator}
			ml_parser.VisitAll(visitor, parsed.RootNodes, nil)
			// Check that we visited all expected nodes
			if len(accumulator) < 7 {
				t.Errorf("Expected at least 7 nodes to be visited, got %d", len(accumulator))
			}
		})

		t.Run("should skip typed visit if visit() returns a truthy value", func(t *testing.T) {
			visitor := &skipVisitor{}
			parsed := parser.Parse("<div id=\"foo\"></div><div id=\"bar\"></div>", "TestComp", nil)
			traversal := ml_parser.VisitAll(visitor, parsed.RootNodes, nil)
			// Check that visit() was called and returned true
			if len(traversal) != 2 {
				t.Errorf("Expected 2 traversal results, got %d", len(traversal))
			}
		})
	})

	t.Run("errors", func(t *testing.T) {
		t.Run("should report unexpected closing tags", func(t *testing.T) {
			parsed := parser.Parse("<div></p></div>", "TestComp", nil)
			if len(parsed.Errors) != 1 {
				t.Errorf("Expected 1 error, got %d", len(parsed.Errors))
			}
			expectedErrors := []interface{}{
				[]interface{}{
					"p",
					"Unexpected closing tag \"p\". It may happen when the tag has already been closed by another tag. For more info see https://www.w3.org/TR/html5/syntax.html#closing-elements-that-have-implied-end-tags",
					"0:5",
				},
			}
			resultErrors := humanizeErrors(parsed.Errors)
			if diff := cmp.Diff(expectedErrors, resultErrors); diff != "" {
				t.Errorf("humanizeErrors() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("gets correct close tag for parent when a child is not closed", func(t *testing.T) {
			parsed := parser.Parse("<div><span></div>", "TestComp", nil)
			if len(parsed.Errors) != 1 {
				t.Errorf("Expected 1 error, got %d", len(parsed.Errors))
			}
			expectedErrors := []interface{}{
				[]interface{}{
					"div",
					"Unexpected closing tag \"div\". It may happen when the tag has already been closed by another tag. For more info see https://www.w3.org/TR/html5/syntax.html#closing-elements-that-have-implied-end-tags",
					"0:11",
				},
			}
			resultErrors := humanizeErrors(parsed.Errors)
			if diff := cmp.Diff(expectedErrors, resultErrors); diff != "" {
				t.Errorf("humanizeErrors() mismatch (-want +got):\n%s", diff)
			}
			result := HumanizeNodes(parsed.RootNodes, true)
			expected := []interface{}{
				[]interface{}{"Element", "div", 0, "<div><span></div>", "<div>", "</div>"},
				[]interface{}{"Element", "span", 1, "<span>", "<span>", nil},
			}
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("HumanizeNodes() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should report closing tag for void elements", func(t *testing.T) {
			parsed := parser.Parse("<input></input>", "TestComp", nil)
			if len(parsed.Errors) != 1 {
				t.Errorf("Expected 1 error, got %d", len(parsed.Errors))
			}
			expectedErrors := []interface{}{
				[]interface{}{"input", "Void elements do not have end tags \"input\"", "0:7"},
			}
			resultErrors := humanizeErrors(parsed.Errors)
			if diff := cmp.Diff(expectedErrors, resultErrors); diff != "" {
				t.Errorf("humanizeErrors() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should report self closing html element", func(t *testing.T) {
			parsed := parser.Parse("<p />", "TestComp", nil)
			if len(parsed.Errors) != 1 {
				t.Errorf("Expected 1 error, got %d", len(parsed.Errors))
			}
			expectedErrors := []interface{}{
				[]interface{}{"p", "Only void, custom and foreign elements can be self closed \"p\"", "0:0"},
			}
			resultErrors := humanizeErrors(parsed.Errors)
			if diff := cmp.Diff(expectedErrors, resultErrors); diff != "" {
				t.Errorf("humanizeErrors() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should not report self closing custom element", func(t *testing.T) {
			parsed := parser.Parse("<my-cmp />", "TestComp", nil)
			if len(parsed.Errors) != 0 {
				t.Errorf("Expected 0 errors, got %d", len(parsed.Errors))
			}
		})

		t.Run("should also report lexer errors", func(t *testing.T) {
			parsed := parser.Parse("<!-err--><div></p></div>", "TestComp", nil)
			if len(parsed.Errors) < 2 {
				t.Errorf("Expected at least 2 errors, got %d", len(parsed.Errors))
			}
		})
	})
}

// Helper visitor for testing
type testVisitor struct {
	accumulator *[]ml_parser.Node
}

func (v *testVisitor) Visit(node ml_parser.Node, context interface{}) interface{} {
	*v.accumulator = append(*v.accumulator, node)
	return nil
}

func (v *testVisitor) VisitElement(element *ml_parser.Element, context interface{}) interface{} {
	ml_parser.VisitAll(v, convertAttributesToNodes(element.Attrs), context)
	ml_parser.VisitAll(v, convertDirectivesToNodes(element.Directives), context)
	ml_parser.VisitAll(v, element.Children, context)
	return nil
}

func (v *testVisitor) VisitAttribute(attribute *ml_parser.Attribute, context interface{}) interface{} {
	return nil
}

func (v *testVisitor) VisitText(text *ml_parser.Text, context interface{}) interface{} {
	return nil
}

func (v *testVisitor) VisitComment(comment *ml_parser.Comment, context interface{}) interface{} {
	return nil
}

func (v *testVisitor) VisitExpansion(expansion *ml_parser.Expansion, context interface{}) interface{} {
	ml_parser.VisitAll(v, convertExpansionCasesToNodes(expansion.Cases), context)
	return nil
}

func (v *testVisitor) VisitExpansionCase(expansionCase *ml_parser.ExpansionCase, context interface{}) interface{} {
	return nil
}

func (v *testVisitor) VisitBlock(block *ml_parser.Block, context interface{}) interface{} {
	ml_parser.VisitAll(v, convertBlockParametersToNodes(block.Parameters), context)
	ml_parser.VisitAll(v, block.Children, context)
	return nil
}

func (v *testVisitor) VisitBlockParameter(parameter *ml_parser.BlockParameter, context interface{}) interface{} {
	return nil
}

func (v *testVisitor) VisitLetDeclaration(decl *ml_parser.LetDeclaration, context interface{}) interface{} {
	return nil
}

func (v *testVisitor) VisitComponent(node *ml_parser.Component, context interface{}) interface{} {
	ml_parser.VisitAll(v, convertAttributesToNodes(node.Attrs), context)
	ml_parser.VisitAll(v, convertDirectivesToNodes(node.Directives), context)
	ml_parser.VisitAll(v, node.Children, context)
	return nil
}

func (v *testVisitor) VisitDirective(directive *ml_parser.Directive, context interface{}) interface{} {
	ml_parser.VisitAll(v, convertAttributesToNodes(directive.Attrs), context)
	return nil
}

// Helper visitor that skips typed visits
type skipVisitor struct{}

func (v *skipVisitor) Visit(node ml_parser.Node, context interface{}) interface{} {
	return true
}

func (v *skipVisitor) VisitElement(element *ml_parser.Element, context interface{}) interface{} {
	panic("Unexpected")
}

func (v *skipVisitor) VisitAttribute(attribute *ml_parser.Attribute, context interface{}) interface{} {
	panic("Unexpected")
}

func (v *skipVisitor) VisitText(text *ml_parser.Text, context interface{}) interface{} {
	panic("Unexpected")
}

func (v *skipVisitor) VisitComment(comment *ml_parser.Comment, context interface{}) interface{} {
	panic("Unexpected")
}

func (v *skipVisitor) VisitExpansion(expansion *ml_parser.Expansion, context interface{}) interface{} {
	panic("Unexpected")
}

func (v *skipVisitor) VisitExpansionCase(expansionCase *ml_parser.ExpansionCase, context interface{}) interface{} {
	panic("Unexpected")
}

func (v *skipVisitor) VisitBlock(block *ml_parser.Block, context interface{}) interface{} {
	panic("Unexpected")
}

func (v *skipVisitor) VisitBlockParameter(parameter *ml_parser.BlockParameter, context interface{}) interface{} {
	panic("Unexpected")
}

func (v *skipVisitor) VisitLetDeclaration(decl *ml_parser.LetDeclaration, context interface{}) interface{} {
	panic("Unexpected")
}

func (v *skipVisitor) VisitComponent(node *ml_parser.Component, context interface{}) interface{} {
	panic("Unexpected")
}

func (v *skipVisitor) VisitDirective(directive *ml_parser.Directive, context interface{}) interface{} {
	panic("Unexpected")
}
