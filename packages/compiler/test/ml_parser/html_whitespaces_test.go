package ml_parser_test

import (
	"ngc-go/packages/compiler/src/ml_parser"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// Note: convertAttributesToNodes, convertDirectivesToNodes, convertExpansionCasesToNodes,
// and convertBlockParametersToNodes are defined in ast_spec_utils.go.
// When building the entire package, Go will find them. The linter may report errors
// when processing this file individually, but the package will build correctly.

func TestRemoveWhitespaces(t *testing.T) {
	parseAndRemoveWS := func(template string, options *ml_parser.TokenizeOptions) []interface{} {
		return HumanizeDom(
			ml_parser.RemoveWhitespaces(
				ml_parser.NewHtmlParser().Parse(template, "TestComp", options),
				true, // preserveSignificantWhitespace
			),
			false,
		)
	}

	t.Run("should remove blank text nodes", func(t *testing.T) {
		expected := []interface{}{}
		result := parseAndRemoveWS(" ", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("parseAndRemoveWS() mismatch (-want +got):\n%s", diff)
		}

		result = parseAndRemoveWS("\n", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("parseAndRemoveWS() mismatch (-want +got):\n%s", diff)
		}

		result = parseAndRemoveWS("\t", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("parseAndRemoveWS() mismatch (-want +got):\n%s", diff)
		}

		result = parseAndRemoveWS("    \t    \n ", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("parseAndRemoveWS() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should remove whitespaces (space, tab, new line) between elements", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{"Element", "br", 0},
			[]interface{}{"Element", "br", 0},
			[]interface{}{"Element", "br", 0},
			[]interface{}{"Element", "br", 0},
		}
		result := parseAndRemoveWS("<br>  <br>\t<br>\n<br>", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("parseAndRemoveWS() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should remove whitespaces from child text nodes", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{"Element", "div", 0},
			[]interface{}{"Element", "span", 1},
		}
		result := parseAndRemoveWS("<div><span> </span></div>", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("parseAndRemoveWS() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should remove whitespaces from the beginning and end of a template", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{"Element", "br", 0},
		}
		result := parseAndRemoveWS(" <br>\t", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("parseAndRemoveWS() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should convert &ngsp; to a space and preserve it", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{"Element", "div", 0},
			[]interface{}{"Element", "span", 1},
			[]interface{}{"Text", "foo", 2, []interface{}{"foo"}},
			[]interface{}{"Text", " ", 1, []interface{}{""}, []interface{}{ml_parser.NGSP_UNICODE, "&ngsp;"}, []interface{}{""}},
			[]interface{}{"Element", "span", 1},
			[]interface{}{"Text", "bar", 2, []interface{}{"bar"}},
		}
		result := parseAndRemoveWS("<div><span>foo</span>&ngsp;<span>bar</span></div>", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("parseAndRemoveWS() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should replace multiple whitespaces with one space", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{"Text", " foo ", 0, []interface{}{" foo "}},
		}
		result := parseAndRemoveWS("\n\n\nfoo\t\t\t", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("parseAndRemoveWS() mismatch (-want +got):\n%s", diff)
		}

		expected = []interface{}{
			[]interface{}{"Text", " foo ", 0, []interface{}{" foo "}},
		}
		result = parseAndRemoveWS("   \n foo  \t ", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("parseAndRemoveWS() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should remove whitespace inside of blocks", func(t *testing.T) {
		markup := "@if (cond) {<br>  <br>\t<br>\n<br>}"
		expected := []interface{}{
			[]interface{}{"Block", "if", 0},
			[]interface{}{"BlockParameter", "cond"},
			[]interface{}{"Element", "br", 1},
			[]interface{}{"Element", "br", 1},
			[]interface{}{"Element", "br", 1},
			[]interface{}{"Element", "br", 1},
		}
		result := parseAndRemoveWS(markup, nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("parseAndRemoveWS() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should not replace &nbsp;", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{"Text", "\u00a0", 0, []interface{}{""}, []interface{}{"\u00a0", "&nbsp;"}, []interface{}{""}},
		}
		result := parseAndRemoveWS("&nbsp;", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("parseAndRemoveWS() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should not replace sequences of &nbsp;", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{
				"Text",
				"\u00a0\u00a0foo\u00a0\u00a0",
				0,
				[]interface{}{""},
				[]interface{}{"\u00a0", "&nbsp;"},
				[]interface{}{""},
				[]interface{}{"\u00a0", "&nbsp;"},
				[]interface{}{"foo"},
				[]interface{}{"\u00a0", "&nbsp;"},
				[]interface{}{""},
				[]interface{}{"\u00a0", "&nbsp;"},
				[]interface{}{""},
			},
		}
		result := parseAndRemoveWS("&nbsp;&nbsp;foo&nbsp;&nbsp;", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("parseAndRemoveWS() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should not replace single tab and newline with spaces", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{"Text", "\nfoo", 0, []interface{}{"\nfoo"}},
		}
		result := parseAndRemoveWS("\nfoo", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("parseAndRemoveWS() mismatch (-want +got):\n%s", diff)
		}

		expected = []interface{}{
			[]interface{}{"Text", "\tfoo", 0, []interface{}{"\tfoo"}},
		}
		result = parseAndRemoveWS("\tfoo", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("parseAndRemoveWS() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should preserve single whitespaces between interpolations", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{
				"Text",
				"{{fooExp}} {{barExp}}",
				0,
				[]interface{}{""},
				[]interface{}{"{{", "fooExp", "}}"},
				[]interface{}{" "},
				[]interface{}{"{{", "barExp", "}}"},
				[]interface{}{""},
			},
		}
		result := parseAndRemoveWS("{{fooExp}} {{barExp}}", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("parseAndRemoveWS() mismatch (-want +got):\n%s", diff)
		}

		expected = []interface{}{
			[]interface{}{
				"Text",
				"{{fooExp}}\t{{barExp}}",
				0,
				[]interface{}{""},
				[]interface{}{"{{", "fooExp", "}}"},
				[]interface{}{"\t"},
				[]interface{}{"{{", "barExp", "}}"},
				[]interface{}{""},
			},
		}
		result = parseAndRemoveWS("{{fooExp}}\t{{barExp}}", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("parseAndRemoveWS() mismatch (-want +got):\n%s", diff)
		}

		expected = []interface{}{
			[]interface{}{
				"Text",
				"{{fooExp}}\n{{barExp}}",
				0,
				[]interface{}{""},
				[]interface{}{"{{", "fooExp", "}}"},
				[]interface{}{"\n"},
				[]interface{}{"{{", "barExp", "}}"},
				[]interface{}{""},
			},
		}
		result = parseAndRemoveWS("{{fooExp}}\n{{barExp}}", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("parseAndRemoveWS() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should preserve whitespaces around interpolations", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{"Text", " {{exp}} ", 0, []interface{}{" "}, []interface{}{"{{", "exp", "}}"}, []interface{}{" "}},
		}
		result := parseAndRemoveWS(" {{exp}} ", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("parseAndRemoveWS() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should preserve whitespaces around ICU expansions", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{"Element", "span", 0},
			[]interface{}{"Text", " ", 1, []interface{}{" "}},
			[]interface{}{"Expansion", "a", "b", 1},
			[]interface{}{"ExpansionCase", "=4", 2},
			[]interface{}{"Text", " ", 1, []interface{}{" "}},
		}
		options := &ml_parser.TokenizeOptions{
			TokenizeExpansionForms: boolPtr(true),
		}
		result := parseAndRemoveWS("<span> {a, b, =4 {c}} </span>", options)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("parseAndRemoveWS() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should preserve whitespaces inside <pre> elements", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{"Element", "pre", 0},
			[]interface{}{"Element", "strong", 1},
			[]interface{}{"Text", "foo", 2, []interface{}{"foo"}},
			[]interface{}{"Text", "\n", 1, []interface{}{"\n"}},
			[]interface{}{"Element", "strong", 1},
			[]interface{}{"Text", "bar", 2, []interface{}{"bar"}},
		}
		result := parseAndRemoveWS("<pre><strong>foo</strong>\n<strong>bar</strong></pre>", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("parseAndRemoveWS() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should skip whitespace trimming in <textarea>", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{"Element", "textarea", 0},
			[]interface{}{"Text", "foo\n\n  bar", 1, []interface{}{"foo\n\n  bar"}},
		}
		result := parseAndRemoveWS("<textarea>foo\n\n  bar</textarea>", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("parseAndRemoveWS() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should preserve whitespaces inside elements annotated with ngPreserveWhitespaces", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{"Element", "div", 0},
			[]interface{}{"Element", "img", 1},
			[]interface{}{"Text", " ", 1, []interface{}{" "}},
			[]interface{}{"Element", "img", 1},
		}
		result := parseAndRemoveWS("<div "+ml_parser.PreserveWsAttrName+"><img> <img></div>", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("parseAndRemoveWS() mismatch (-want +got):\n%s", diff)
		}
	})
}
