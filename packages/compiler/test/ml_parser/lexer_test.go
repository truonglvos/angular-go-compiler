package ml_parser_test

import (
	"fmt"
	"ngc-go/packages/compiler/src/ml_parser"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestHtmlLexer_LineColumnNumbers(t *testing.T) {
	t.Run("should work without newlines", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "0:0"},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END, "0:2"},
			[]interface{}{ml_parser.TokenTypeTEXT, "0:3"},
			[]interface{}{ml_parser.TokenTypeTAG_CLOSE, "0:4"},
			[]interface{}{ml_parser.TokenTypeEOF, "0:8"},
		}
		result := tokenizeAndHumanizeLineColumn("<t>a</t>", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeLineColumn() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should work with one newline", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "0:0"},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END, "0:2"},
			[]interface{}{ml_parser.TokenTypeTEXT, "0:3"},
			[]interface{}{ml_parser.TokenTypeTAG_CLOSE, "1:1"},
			[]interface{}{ml_parser.TokenTypeEOF, "1:5"},
		}
		result := tokenizeAndHumanizeLineColumn("<t>\na</t>", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeLineColumn() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should work with multiple newlines", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "0:0"},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END, "1:0"},
			[]interface{}{ml_parser.TokenTypeTEXT, "1:1"},
			[]interface{}{ml_parser.TokenTypeTAG_CLOSE, "2:1"},
			[]interface{}{ml_parser.TokenTypeEOF, "2:5"},
		}
		result := tokenizeAndHumanizeLineColumn("<t\n>\na</t>", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeLineColumn() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should work with CR and LF", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "0:0"},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END, "1:0"},
			[]interface{}{ml_parser.TokenTypeTEXT, "1:1"},
			[]interface{}{ml_parser.TokenTypeTAG_CLOSE, "2:1"},
			[]interface{}{ml_parser.TokenTypeEOF, "2:5"},
		}
		result := tokenizeAndHumanizeLineColumn("<t\n>\r\na\r</t>", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeLineColumn() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should skip over leading trivia for source-span start", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "0:0", "0:0"},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END, "0:2", "0:2"},
			[]interface{}{ml_parser.TokenTypeTEXT, "1:3", "0:3"},
			[]interface{}{ml_parser.TokenTypeTAG_CLOSE, "1:4", "1:4"},
			[]interface{}{ml_parser.TokenTypeEOF, "1:8", "1:8"},
		}
		options := &ml_parser.TokenizeOptions{LeadingTriviaChars: []string{"\n", " ", "\t"}}
		result := tokenizeAndHumanizeFullStart("<t>\n \t a</t>", options)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeFullStart() mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestHtmlLexer_ContentRanges(t *testing.T) {
	t.Run("should only process the text within the range", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTEXT, "line 1\nline 2\nline 3"},
			[]interface{}{ml_parser.TokenTypeEOF, ""},
		}
		options := &ml_parser.TokenizeOptions{
			Range: &ml_parser.LexerRange{StartPos: 19, StartLine: 2, StartCol: 7, EndPos: 39},
		}
		result := tokenizeAndHumanizeSourceSpans(
			"pre 1\npre 2\npre 3 `line 1\nline 2\nline 3` post 1\n post 2\n post 3",
			options,
		)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeSourceSpans() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should take into account preceding (non-processed) lines and columns", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTEXT, "2:7"},
			[]interface{}{ml_parser.TokenTypeEOF, "4:6"},
		}
		options := &ml_parser.TokenizeOptions{
			Range: &ml_parser.LexerRange{StartPos: 19, StartLine: 2, StartCol: 7, EndPos: 39},
		}
		result := tokenizeAndHumanizeLineColumn(
			"pre 1\npre 2\npre 3 `line 1\nline 2\nline 3` post 1\n post 2\n post 3",
			options,
		)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeLineColumn() mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestHtmlLexer_Comments(t *testing.T) {
	t.Run("should parse comments", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeCOMMENT_START},
			[]interface{}{ml_parser.TokenTypeRAW_TEXT, "t\ne\ns\nt"},
			[]interface{}{ml_parser.TokenTypeCOMMENT_END},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("<!--t\ne\rs\r\nt-->", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should store the locations", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeCOMMENT_START, "<!--"},
			[]interface{}{ml_parser.TokenTypeRAW_TEXT, "t\ne\rs\r\nt"},
			[]interface{}{ml_parser.TokenTypeCOMMENT_END, "-->"},
			[]interface{}{ml_parser.TokenTypeEOF, ""},
		}
		result := tokenizeAndHumanizeSourceSpans("<!--t\ne\rs\r\nt-->", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeSourceSpans() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should report <!- without -", func(t *testing.T) {

		expected := []interface{}{
			[]interface{}{"Unexpected character \"a\"", "0:3"},
		}
		result := tokenizeAndHumanizeErrors("<!-a", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeErrors() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should report missing end comment", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{"Unexpected character \"EOF\"", "0:4"},
		}
		result := tokenizeAndHumanizeErrors("<!--", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeErrors() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should accept comments finishing by too many dashes (even number)", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeCOMMENT_START, "<!--"},
			[]interface{}{ml_parser.TokenTypeRAW_TEXT, " test --"},
			[]interface{}{ml_parser.TokenTypeCOMMENT_END, "-->"},
			[]interface{}{ml_parser.TokenTypeEOF, ""},
		}
		result := tokenizeAndHumanizeSourceSpans("<!-- test ---->", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeSourceSpans() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should accept comments finishing by too many dashes (odd number)", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeCOMMENT_START, "<!--"},
			[]interface{}{ml_parser.TokenTypeRAW_TEXT, " test -"},
			[]interface{}{ml_parser.TokenTypeCOMMENT_END, "-->"},
			[]interface{}{ml_parser.TokenTypeEOF, ""},
		}
		result := tokenizeAndHumanizeSourceSpans("<!-- test --->", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeSourceSpans() mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestHtmlLexer_Doctype(t *testing.T) {
	t.Run("should parse doctypes", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeDOC_TYPE, "DOCTYPE html"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("<!DOCTYPE html>", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should store the locations", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeDOC_TYPE, "<!DOCTYPE html>"},
			[]interface{}{ml_parser.TokenTypeEOF, ""},
		}
		result := tokenizeAndHumanizeSourceSpans("<!DOCTYPE html>", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeSourceSpans() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should report missing end doctype", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{"Unexpected character \"EOF\"", "0:2"},
		}
		result := tokenizeAndHumanizeErrors("<!", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeErrors() mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestHtmlLexer_CDATA(t *testing.T) {
	t.Run("should parse CDATA", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeCDATA_START},
			[]interface{}{ml_parser.TokenTypeRAW_TEXT, "t\ne\ns\nt"},
			[]interface{}{ml_parser.TokenTypeCDATA_END},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("<![CDATA[t\ne\rs\r\nt]]>", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should store the locations", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeCDATA_START, "<![CDATA["},
			[]interface{}{ml_parser.TokenTypeRAW_TEXT, "t\ne\rs\r\nt"},
			[]interface{}{ml_parser.TokenTypeCDATA_END, "]]>"},
			[]interface{}{ml_parser.TokenTypeEOF, ""},
		}
		result := tokenizeAndHumanizeSourceSpans("<![CDATA[t\ne\rs\r\nt]]>", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeSourceSpans() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should report <![ without CDATA[", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{"Unexpected character \"a\"", "0:3"},
		}
		result := tokenizeAndHumanizeErrors("<![a", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeErrors() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should report missing end cdata", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{"Unexpected character \"EOF\"", "0:9"},
		}
		result := tokenizeAndHumanizeErrors("<![CDATA[", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeErrors() mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestHtmlLexer_OpenTags(t *testing.T) {
	t.Run("should parse open tags without prefix", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "test"},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("<test>", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse namespace prefix", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "ns1", "test"},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("<ns1:test>", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse void tags", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "test"},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END_VOID},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("<test/>", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should allow whitespace after the tag name", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "test"},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("<test >", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should store the locations", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "<test"},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END, ">"},
			[]interface{}{ml_parser.TokenTypeEOF, ""},
		}
		result := tokenizeAndHumanizeSourceSpans("<test>", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeSourceSpans() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("tags", func(t *testing.T) {
		t.Run("terminated with EOF", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{ml_parser.TokenTypeINCOMPLETE_TAG_OPEN, "<div"},
				[]interface{}{ml_parser.TokenTypeEOF, ""},
			}
			result := tokenizeAndHumanizeSourceSpans("<div", nil)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("tokenizeAndHumanizeSourceSpans() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("after tag name", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{ml_parser.TokenTypeINCOMPLETE_TAG_OPEN, "<div"},
				[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "<span"},
				[]interface{}{ml_parser.TokenTypeTAG_OPEN_END, ">"},
				[]interface{}{ml_parser.TokenTypeINCOMPLETE_TAG_OPEN, "<div"},
				[]interface{}{ml_parser.TokenTypeTAG_CLOSE, "</span>"},
				[]interface{}{ml_parser.TokenTypeEOF, ""},
			}
			result := tokenizeAndHumanizeSourceSpans("<div<span><div</span>", nil)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("tokenizeAndHumanizeSourceSpans() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("in attribute", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{ml_parser.TokenTypeINCOMPLETE_TAG_OPEN, "<div"},
				[]interface{}{ml_parser.TokenTypeATTR_NAME, "class"},
				[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
				[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, "hi"},
				[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
				[]interface{}{ml_parser.TokenTypeATTR_NAME, "sty"},
				[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "<span"},
				[]interface{}{ml_parser.TokenTypeTAG_OPEN_END, ">"},
				[]interface{}{ml_parser.TokenTypeTAG_CLOSE, "</span>"},
				[]interface{}{ml_parser.TokenTypeEOF, ""},
			}
			result := tokenizeAndHumanizeSourceSpans(`<div class="hi" sty<span></span>`, nil)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("tokenizeAndHumanizeSourceSpans() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("after quote", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{ml_parser.TokenTypeINCOMPLETE_TAG_OPEN, "<div"},
				[]interface{}{ml_parser.TokenTypeTEXT, "\""},
				[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "<span"},
				[]interface{}{ml_parser.TokenTypeTAG_OPEN_END, ">"},
				[]interface{}{ml_parser.TokenTypeTAG_CLOSE, "</span>"},
				[]interface{}{ml_parser.TokenTypeEOF, ""},
			}
			result := tokenizeAndHumanizeSourceSpans(`<div "<span></span>`, nil)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("tokenizeAndHumanizeSourceSpans() mismatch (-want +got):\n%s", diff)
			}
		})
	})
}

func TestHtmlLexer_ComponentTags(t *testing.T) {
	options := &ml_parser.TokenizeOptions{
		SelectorlessEnabled: boolPtr(true),
	}

	t.Run("should parse a basic component tag", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeCOMPONENT_OPEN_START, "MyComp", "", ""},
			[]interface{}{ml_parser.TokenTypeCOMPONENT_OPEN_END},
			[]interface{}{ml_parser.TokenTypeTEXT, "hello"},
			[]interface{}{ml_parser.TokenTypeCOMPONENT_CLOSE, "MyComp", "", ""},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("<MyComp>hello</MyComp>", options)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse a component tag with a tag name", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeCOMPONENT_OPEN_START, "MyComp", "", "button"},
			[]interface{}{ml_parser.TokenTypeCOMPONENT_OPEN_END},
			[]interface{}{ml_parser.TokenTypeTEXT, "hello"},
			[]interface{}{ml_parser.TokenTypeCOMPONENT_CLOSE, "MyComp", "", "button"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("<MyComp:button>hello</MyComp:button>", options)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse a component tag with a tag name and namespace", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeCOMPONENT_OPEN_START, "MyComp", "svg", "title"},
			[]interface{}{ml_parser.TokenTypeCOMPONENT_OPEN_END},
			[]interface{}{ml_parser.TokenTypeTEXT, "hello"},
			[]interface{}{ml_parser.TokenTypeCOMPONENT_CLOSE, "MyComp", "svg", "title"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("<MyComp:svg:title>hello</MyComp:svg:title>", options)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse a self-closing component tag", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeCOMPONENT_OPEN_START, "MyComp", "", ""},
			[]interface{}{ml_parser.TokenTypeCOMPONENT_OPEN_END_VOID},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("<MyComp/>", options)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should produce spans for component tags", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeCOMPONENT_OPEN_START, "<MyComp:svg:title"},
			[]interface{}{ml_parser.TokenTypeCOMPONENT_OPEN_END, ">"},
			[]interface{}{ml_parser.TokenTypeTEXT, "hello"},
			[]interface{}{ml_parser.TokenTypeCOMPONENT_CLOSE, "</MyComp:svg:title>"},
			[]interface{}{ml_parser.TokenTypeEOF, ""},
		}
		result := tokenizeAndHumanizeSourceSpans("<MyComp:svg:title>hello</MyComp:svg:title>", options)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeSourceSpans() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse an incomplete component open tag", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeINCOMPLETE_COMPONENT_OPEN, "MyComp", "", "span"},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "", "class"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, "hi"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "", "sty"},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "span"},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END},
			[]interface{}{ml_parser.TokenTypeTAG_CLOSE, "", "span"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts(`<MyComp:span class="hi" sty<span></span>`, options)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse a component tag with raw text", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeCOMPONENT_OPEN_START, "MyComp", "", "script"},
			[]interface{}{ml_parser.TokenTypeCOMPONENT_OPEN_END},
			[]interface{}{ml_parser.TokenTypeRAW_TEXT, "t\ne\ns\nt"},
			[]interface{}{ml_parser.TokenTypeCOMPONENT_CLOSE, "MyComp", "", "script"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts(`<MyComp:script>t\ne\rs\r\nt</MyComp:script>`, options)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse a component tag with escapable raw text", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeCOMPONENT_OPEN_START, "MyComp", "", "title"},
			[]interface{}{ml_parser.TokenTypeCOMPONENT_OPEN_END},
			[]interface{}{ml_parser.TokenTypeESCAPABLE_RAW_TEXT, "t\ne\ns\nt"},
			[]interface{}{ml_parser.TokenTypeCOMPONENT_CLOSE, "MyComp", "", "title"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts(`<MyComp:title>t\ne\rs\r\nt</MyComp:title>`, options)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestHtmlLexer_SelectorlessDirectives(t *testing.T) {
	options := &ml_parser.TokenizeOptions{
		SelectorlessEnabled: boolPtr(true),
	}

	t.Run("should parse a basic directive", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "div"},
			[]interface{}{ml_parser.TokenTypeDIRECTIVE_NAME, "MyDir"},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END},
			[]interface{}{ml_parser.TokenTypeTAG_CLOSE, "", "div"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("<div @MyDir></div>", options)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse a directive with parentheses, but no attributes", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "div"},
			[]interface{}{ml_parser.TokenTypeDIRECTIVE_NAME, "MyDir"},
			[]interface{}{ml_parser.TokenTypeDIRECTIVE_OPEN},
			[]interface{}{ml_parser.TokenTypeDIRECTIVE_CLOSE},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END},
			[]interface{}{ml_parser.TokenTypeTAG_CLOSE, "", "div"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("<div @MyDir()></div>", options)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse a directive with a single attribute without a value", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "div"},
			[]interface{}{ml_parser.TokenTypeDIRECTIVE_NAME, "MyDir"},
			[]interface{}{ml_parser.TokenTypeDIRECTIVE_OPEN},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "", "foo"},
			[]interface{}{ml_parser.TokenTypeDIRECTIVE_CLOSE},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END},
			[]interface{}{ml_parser.TokenTypeTAG_CLOSE, "", "div"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("<div @MyDir(foo)></div>", options)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse a directive with attributes", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "div"},
			[]interface{}{ml_parser.TokenTypeDIRECTIVE_NAME, "MyDir"},
			[]interface{}{ml_parser.TokenTypeDIRECTIVE_OPEN},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "", "static"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, "one"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "", "[bound]"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, "expr"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "", "[(twoWay)]"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, "expr"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "", "#ref"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, "name"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "", "(click)"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, "handler()"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeDIRECTIVE_CLOSE},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END},
			[]interface{}{ml_parser.TokenTypeTAG_CLOSE, "", "div"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts(`<div @MyDir(static="one" [bound]="expr" [(twoWay)]="expr" #ref="name" (click)="handler()")></div>`, options)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse a directive mixed in with other attributes", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "div"},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "", "before"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, "value"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeDIRECTIVE_NAME, "OneDir"},
			[]interface{}{ml_parser.TokenTypeDIRECTIVE_OPEN},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "", "[one]"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, "1"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "", "two"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, "2"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeDIRECTIVE_CLOSE},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "", "middle"},
			[]interface{}{ml_parser.TokenTypeDIRECTIVE_NAME, "TwoDir"},
			[]interface{}{ml_parser.TokenTypeDIRECTIVE_NAME, "ThreeDir"},
			[]interface{}{ml_parser.TokenTypeDIRECTIVE_OPEN},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "", "(three)"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, "handleThree()"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeDIRECTIVE_CLOSE},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "", "after"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, "value"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END},
			[]interface{}{ml_parser.TokenTypeTAG_CLOSE, "", "div"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts(`<div before="value" @OneDir([one]="1" two="2") middle @TwoDir @ThreeDir((three)="handleThree()") after="value"></div>`, options)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should not pick up selectorless-like text inside a tag", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "div"},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END},
			[]interface{}{ml_parser.TokenTypeTEXT, "@MyDir()"},
			[]interface{}{ml_parser.TokenTypeTAG_CLOSE, "", "div"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("<div>@MyDir()</div>", options)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should not pick up selectorless-like text inside an attribute", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "div"},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "", "hello"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, "@MyDir"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END},
			[]interface{}{ml_parser.TokenTypeTAG_CLOSE, "", "div"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts(`<div hello="@MyDir"></div>`, options)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should produce spans for directives", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "<div"},
			[]interface{}{ml_parser.TokenTypeDIRECTIVE_NAME, "@Empty"},
			[]interface{}{ml_parser.TokenTypeDIRECTIVE_NAME, "@NoAttrs"},
			[]interface{}{ml_parser.TokenTypeDIRECTIVE_OPEN, "("},
			[]interface{}{ml_parser.TokenTypeDIRECTIVE_CLOSE, ")"},
			[]interface{}{ml_parser.TokenTypeDIRECTIVE_NAME, "@WithAttr"},
			[]interface{}{ml_parser.TokenTypeDIRECTIVE_OPEN, "("},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "[one]"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, "1"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "two"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, "2"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeDIRECTIVE_CLOSE, ")"},
			[]interface{}{ml_parser.TokenTypeDIRECTIVE_NAME, "@WithSimpleAttr"},
			[]interface{}{ml_parser.TokenTypeDIRECTIVE_OPEN, "("},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "simple"},
			[]interface{}{ml_parser.TokenTypeDIRECTIVE_CLOSE, ")"},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END, ">"},
			[]interface{}{ml_parser.TokenTypeTAG_CLOSE, "</div>"},
			[]interface{}{ml_parser.TokenTypeEOF, ""},
		}
		result := tokenizeAndHumanizeSourceSpans(`<div @Empty @NoAttrs() @WithAttr([one]="1" two="2") @WithSimpleAttr(simple)></div>`, options)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeSourceSpans() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should not capture whitespace in directive spans", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "<div"},
			[]interface{}{ml_parser.TokenTypeDIRECTIVE_NAME, "@Dir"},
			[]interface{}{ml_parser.TokenTypeDIRECTIVE_OPEN, "("},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "one"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, "1"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "(two)"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, "handleTwo()"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeDIRECTIVE_CLOSE, ")"},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END, ">"},
			[]interface{}{ml_parser.TokenTypeTAG_CLOSE, "</div>"},
			[]interface{}{ml_parser.TokenTypeEOF, ""},
		}
		result := tokenizeAndHumanizeSourceSpans(`<div    @Dir   (  one="1"    (two)="handleTwo()"     )     ></div>`, options)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeSourceSpans() mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestHtmlLexer_EscapableRawText(t *testing.T) {
	t.Run("should parse text", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "title"},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END},
			[]interface{}{ml_parser.TokenTypeESCAPABLE_RAW_TEXT, "t\ne\ns\nt"},
			[]interface{}{ml_parser.TokenTypeTAG_CLOSE, "", "title"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts(`<title>t\ne\rs\r\nt</title>`, nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should detect entities", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "title"},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END},
			[]interface{}{ml_parser.TokenTypeESCAPABLE_RAW_TEXT, ""},
			[]interface{}{ml_parser.TokenTypeENCODED_ENTITY, "&", "&amp;"},
			[]interface{}{ml_parser.TokenTypeESCAPABLE_RAW_TEXT, ""},
			[]interface{}{ml_parser.TokenTypeTAG_CLOSE, "", "title"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts(`<title>&amp;</title>`, nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should ignore other opening tags", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "title"},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END},
			[]interface{}{ml_parser.TokenTypeESCAPABLE_RAW_TEXT, "a<div>"},
			[]interface{}{ml_parser.TokenTypeTAG_CLOSE, "", "title"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts(`<title>a<div></title>`, nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should ignore other closing tags", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "title"},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END},
			[]interface{}{ml_parser.TokenTypeESCAPABLE_RAW_TEXT, "a</test>"},
			[]interface{}{ml_parser.TokenTypeTAG_CLOSE, "", "title"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts(`<title>a</test></title>`, nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should store the locations", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "<title"},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END, ">"},
			[]interface{}{ml_parser.TokenTypeESCAPABLE_RAW_TEXT, "a"},
			[]interface{}{ml_parser.TokenTypeTAG_CLOSE, "</title>"},
			[]interface{}{ml_parser.TokenTypeEOF, ""},
		}
		result := tokenizeAndHumanizeSourceSpans(`<title>a</title>`, nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeSourceSpans() mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestHtmlLexer_ParsableData(t *testing.T) {
	t.Run("should parse an SVG <title> tag", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "svg", "title"},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END},
			[]interface{}{ml_parser.TokenTypeTEXT, "test"},
			[]interface{}{ml_parser.TokenTypeTAG_CLOSE, "svg", "title"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts(`<svg:title>test</svg:title>`, nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse an SVG <title> tag with children", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "svg", "title"},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "f"},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END},
			[]interface{}{ml_parser.TokenTypeTEXT, "test"},
			[]interface{}{ml_parser.TokenTypeTAG_CLOSE, "", "f"},
			[]interface{}{ml_parser.TokenTypeTAG_CLOSE, "svg", "title"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts(`<svg:title><f>test</f></svg:title>`, nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})
}

// Helpers

func tokenizeAndHumanizeLineColumn(input string, options *ml_parser.TokenizeOptions) []interface{} {
	result := ml_parser.Tokenize(input, "someUrl", nil, options)
	humanized := []interface{}{}
	for _, token := range result.Tokens {
		humanized = append(humanized, []interface{}{
			token.Type(),
			HumanizeLineColumn(token.SourceSpan().Start),
		})
	}
	return humanized
}

func tokenizeAndHumanizeFullStart(input string, options *ml_parser.TokenizeOptions) []interface{} {
	result := ml_parser.Tokenize(input, "someUrl", nil, options)
	humanized := []interface{}{}
	for _, token := range result.Tokens {
		humanized = append(humanized, []interface{}{
			token.Type(),
			HumanizeLineColumn(token.SourceSpan().Start),
			HumanizeLineColumn(token.SourceSpan().FullStart),
		})
	}
	return humanized
}

func tokenizeAndHumanizeSourceSpans(input string, options *ml_parser.TokenizeOptions) []interface{} {
	result := ml_parser.Tokenize(input, "someUrl", nil, options)
	humanized := []interface{}{}
	for _, token := range result.Tokens {
		humanized = append(humanized, []interface{}{
			token.Type(),
			token.SourceSpan().String(),
		})
	}
	return humanized
}

func tokenizeAndHumanizeParts(input string, options *ml_parser.TokenizeOptions) []interface{} {
	result := ml_parser.Tokenize(input, "someUrl", nil, options)
	humanized := []interface{}{}
	for _, token := range result.Tokens {
		parts := []interface{}{token.Type()}
		for _, part := range token.Parts() {
			parts = append(parts, part)
		}
		humanized = append(humanized, parts)
	}
	return humanized
}

func tokenizeAndHumanizeErrors(input string, options *ml_parser.TokenizeOptions) []interface{} {
	result := ml_parser.Tokenize(input, "someUrl", nil, options)
	humanized := []interface{}{}
	for _, err := range result.Errors {
		humanized = append(humanized, []interface{}{
			err.Msg,
			HumanizeLineColumn(err.Span.Start),
		})
	}
	return humanized
}

func tokenizeWithoutErrors(input string, options *ml_parser.TokenizeOptions) *ml_parser.TokenizeResult {
	result := ml_parser.Tokenize(input, "someUrl", nil, options)
	if len(result.Errors) > 0 {
		panic(fmt.Errorf("Unexpected errors: %v", result.Errors))
	}
	return result
}

func humanizeParts(tokens []ml_parser.Token) []interface{} {
	humanized := []interface{}{}
	for _, token := range tokens {
		parts := []interface{}{token.Type()}
		for _, part := range token.Parts() {
			parts = append(parts, part)
		}
		humanized = append(humanized, parts)
	}
	return humanized
}

func TestHtmlLexer_ExpansionForms(t *testing.T) {
	t.Run("should parse an expansion form", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeEXPANSION_FORM_START},
			[]interface{}{ml_parser.TokenTypeRAW_TEXT, "one.two"},
			[]interface{}{ml_parser.TokenTypeRAW_TEXT, "three"},
			[]interface{}{ml_parser.TokenTypeEXPANSION_CASE_VALUE, "=4"},
			[]interface{}{ml_parser.TokenTypeEXPANSION_CASE_EXP_START},
			[]interface{}{ml_parser.TokenTypeTEXT, "four"},
			[]interface{}{ml_parser.TokenTypeEXPANSION_CASE_EXP_END},
			[]interface{}{ml_parser.TokenTypeEXPANSION_CASE_VALUE, "=5"},
			[]interface{}{ml_parser.TokenTypeEXPANSION_CASE_EXP_START},
			[]interface{}{ml_parser.TokenTypeTEXT, "five"},
			[]interface{}{ml_parser.TokenTypeEXPANSION_CASE_EXP_END},
			[]interface{}{ml_parser.TokenTypeEXPANSION_CASE_VALUE, "foo"},
			[]interface{}{ml_parser.TokenTypeEXPANSION_CASE_EXP_START},
			[]interface{}{ml_parser.TokenTypeTEXT, "bar"},
			[]interface{}{ml_parser.TokenTypeEXPANSION_CASE_EXP_END},
			[]interface{}{ml_parser.TokenTypeEXPANSION_FORM_END},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("{one.two, three, =4 {four} =5 {five} foo {bar} }", &ml_parser.TokenizeOptions{TokenizeExpansionForms: boolPtr(true)})
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse an expansion form with text elements surrounding it", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTEXT, "before"},
			[]interface{}{ml_parser.TokenTypeEXPANSION_FORM_START},
			[]interface{}{ml_parser.TokenTypeRAW_TEXT, "one.two"},
			[]interface{}{ml_parser.TokenTypeRAW_TEXT, "three"},
			[]interface{}{ml_parser.TokenTypeEXPANSION_CASE_VALUE, "=4"},
			[]interface{}{ml_parser.TokenTypeEXPANSION_CASE_EXP_START},
			[]interface{}{ml_parser.TokenTypeTEXT, "four"},
			[]interface{}{ml_parser.TokenTypeEXPANSION_CASE_EXP_END},
			[]interface{}{ml_parser.TokenTypeEXPANSION_FORM_END},
			[]interface{}{ml_parser.TokenTypeTEXT, "after"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("before{one.two, three, =4 {four}}after", &ml_parser.TokenizeOptions{TokenizeExpansionForms: boolPtr(true)})
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse an expansion form as a tag single child", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "div"},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "span"},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END},
			[]interface{}{ml_parser.TokenTypeEXPANSION_FORM_START},
			[]interface{}{ml_parser.TokenTypeRAW_TEXT, "a"},
			[]interface{}{ml_parser.TokenTypeRAW_TEXT, "b"},
			[]interface{}{ml_parser.TokenTypeEXPANSION_CASE_VALUE, "=4"},
			[]interface{}{ml_parser.TokenTypeEXPANSION_CASE_EXP_START},
			[]interface{}{ml_parser.TokenTypeTEXT, "c"},
			[]interface{}{ml_parser.TokenTypeEXPANSION_CASE_EXP_END},
			[]interface{}{ml_parser.TokenTypeEXPANSION_FORM_END},
			[]interface{}{ml_parser.TokenTypeTAG_CLOSE, "", "span"},
			[]interface{}{ml_parser.TokenTypeTAG_CLOSE, "", "div"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("<div><span>{a, b, =4 {c}}</span></div>", &ml_parser.TokenizeOptions{TokenizeExpansionForms: boolPtr(true)})
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse an expansion form with whitespace surrounding it", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "div"},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "span"},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END},
			[]interface{}{ml_parser.TokenTypeTEXT, " "},
			[]interface{}{ml_parser.TokenTypeEXPANSION_FORM_START},
			[]interface{}{ml_parser.TokenTypeRAW_TEXT, "a"},
			[]interface{}{ml_parser.TokenTypeRAW_TEXT, "b"},
			[]interface{}{ml_parser.TokenTypeEXPANSION_CASE_VALUE, "=4"},
			[]interface{}{ml_parser.TokenTypeEXPANSION_CASE_EXP_START},
			[]interface{}{ml_parser.TokenTypeTEXT, "c"},
			[]interface{}{ml_parser.TokenTypeEXPANSION_CASE_EXP_END},
			[]interface{}{ml_parser.TokenTypeEXPANSION_FORM_END},
			[]interface{}{ml_parser.TokenTypeTEXT, " "},
			[]interface{}{ml_parser.TokenTypeTAG_CLOSE, "", "span"},
			[]interface{}{ml_parser.TokenTypeTAG_CLOSE, "", "div"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("<div><span> {a, b, =4 {c}} </span></div>", &ml_parser.TokenizeOptions{TokenizeExpansionForms: boolPtr(true)})
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse an expansion forms with elements in it", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeEXPANSION_FORM_START},
			[]interface{}{ml_parser.TokenTypeRAW_TEXT, "one.two"},
			[]interface{}{ml_parser.TokenTypeRAW_TEXT, "three"},
			[]interface{}{ml_parser.TokenTypeEXPANSION_CASE_VALUE, "=4"},
			[]interface{}{ml_parser.TokenTypeEXPANSION_CASE_EXP_START},
			[]interface{}{ml_parser.TokenTypeTEXT, "four "},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "b"},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END},
			[]interface{}{ml_parser.TokenTypeTEXT, "a"},
			[]interface{}{ml_parser.TokenTypeTAG_CLOSE, "", "b"},
			[]interface{}{ml_parser.TokenTypeEXPANSION_CASE_EXP_END},
			[]interface{}{ml_parser.TokenTypeEXPANSION_FORM_END},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("{one.two, three, =4 {four <b>a</b>}}", &ml_parser.TokenizeOptions{TokenizeExpansionForms: boolPtr(true)})
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse an expansion forms containing an interpolation", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeEXPANSION_FORM_START},
			[]interface{}{ml_parser.TokenTypeRAW_TEXT, "one.two"},
			[]interface{}{ml_parser.TokenTypeRAW_TEXT, "three"},
			[]interface{}{ml_parser.TokenTypeEXPANSION_CASE_VALUE, "=4"},
			[]interface{}{ml_parser.TokenTypeEXPANSION_CASE_EXP_START},
			[]interface{}{ml_parser.TokenTypeTEXT, "four "},
			[]interface{}{ml_parser.TokenTypeINTERPOLATION, "{{", "a", "}}"},
			[]interface{}{ml_parser.TokenTypeTEXT, ""},
			[]interface{}{ml_parser.TokenTypeEXPANSION_CASE_EXP_END},
			[]interface{}{ml_parser.TokenTypeEXPANSION_FORM_END},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("{one.two, three, =4 {four {{a}}}}", &ml_parser.TokenizeOptions{TokenizeExpansionForms: boolPtr(true)})
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse nested expansion forms", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeEXPANSION_FORM_START},
			[]interface{}{ml_parser.TokenTypeRAW_TEXT, "one.two"},
			[]interface{}{ml_parser.TokenTypeRAW_TEXT, "three"},
			[]interface{}{ml_parser.TokenTypeEXPANSION_CASE_VALUE, "=4"},
			[]interface{}{ml_parser.TokenTypeEXPANSION_CASE_EXP_START},
			[]interface{}{ml_parser.TokenTypeEXPANSION_FORM_START},
			[]interface{}{ml_parser.TokenTypeRAW_TEXT, "xx"},
			[]interface{}{ml_parser.TokenTypeRAW_TEXT, "yy"},
			[]interface{}{ml_parser.TokenTypeEXPANSION_CASE_VALUE, "=x"},
			[]interface{}{ml_parser.TokenTypeEXPANSION_CASE_EXP_START},
			[]interface{}{ml_parser.TokenTypeTEXT, "one"},
			[]interface{}{ml_parser.TokenTypeEXPANSION_CASE_EXP_END},
			[]interface{}{ml_parser.TokenTypeEXPANSION_FORM_END},
			[]interface{}{ml_parser.TokenTypeTEXT, " "},
			[]interface{}{ml_parser.TokenTypeEXPANSION_CASE_EXP_END},
			[]interface{}{ml_parser.TokenTypeEXPANSION_FORM_END},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts(`{one.two, three, =4 { {xx, yy, =x {one}} }}`, &ml_parser.TokenizeOptions{TokenizeExpansionForms: boolPtr(true)})
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestHtmlLexer_LineEndingNormalization(t *testing.T) {
	t.Run("should normalize line-endings in expansion forms if i18nNormalizeLineEndingsInICUs is true", func(t *testing.T) {
		input := "{\r\n" +
			"    messages.length,\r\n" +
			"    plural,\r\n" +
			"    =0 {You have \r\nno\r\n messages}\r\n" +
			"    =1 {One {{message}}}}\r\n"

		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeEXPANSION_FORM_START},
			[]interface{}{ml_parser.TokenTypeRAW_TEXT, "\n    messages.length"},
			[]interface{}{ml_parser.TokenTypeRAW_TEXT, "plural"},
			[]interface{}{ml_parser.TokenTypeEXPANSION_CASE_VALUE, "=0"},
			[]interface{}{ml_parser.TokenTypeEXPANSION_CASE_EXP_START},
			[]interface{}{ml_parser.TokenTypeTEXT, "You have \nno\n messages"},
			[]interface{}{ml_parser.TokenTypeEXPANSION_CASE_EXP_END},
			[]interface{}{ml_parser.TokenTypeEXPANSION_CASE_VALUE, "=1"},
			[]interface{}{ml_parser.TokenTypeEXPANSION_CASE_EXP_START},
			[]interface{}{ml_parser.TokenTypeTEXT, "One "},
			[]interface{}{ml_parser.TokenTypeINTERPOLATION, "{{", "message", "}}"},
			[]interface{}{ml_parser.TokenTypeTEXT, ""},
			[]interface{}{ml_parser.TokenTypeEXPANSION_CASE_EXP_END},
			[]interface{}{ml_parser.TokenTypeEXPANSION_FORM_END},
			[]interface{}{ml_parser.TokenTypeTEXT, "\n"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}

		result := tokenizeWithoutErrors(input, &ml_parser.TokenizeOptions{
			TokenizeExpansionForms:         boolPtr(true),
			EscapedString:                  boolPtr(true),
			I18nNormalizeLineEndingsInICUs: boolPtr(true),
		})

		if diff := cmp.Diff(expected, humanizeParts(result.Tokens)); diff != "" {
			t.Errorf("humanizeParts() mismatch (-want +got):\n%s", diff)
		}
		if len(result.NonNormalizedIcuExpressions) != 0 {
			t.Errorf("Expected 0 non-normalized ICU expressions, got %d", len(result.NonNormalizedIcuExpressions))
		}
	})
}

func TestHtmlLexer_Errors(t *testing.T) {
	t.Run("should report unescaped \"{\" on error", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{
				"Unexpected character \"EOF\" (Do you have an unescaped \"{\" in your template? Use \"{{ '{' }}\") to escape it.)",
				"0:21",
			},
		}
		result := tokenizeAndHumanizeErrors("<p>before { after</p>", &ml_parser.TokenizeOptions{TokenizeExpansionForms: boolPtr(true)})
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeErrors() mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestHtmlLexer_UnicodeCharacters(t *testing.T) {
	t.Run("should support unicode characters", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "<p"},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END, ">"},
			[]interface{}{ml_parser.TokenTypeTEXT, "İ"},
			[]interface{}{ml_parser.TokenTypeTAG_CLOSE, "</p>"},
			[]interface{}{ml_parser.TokenTypeEOF, ""},
		}
		result := tokenizeAndHumanizeSourceSpans("<p>İ</p>", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeSourceSpans() mismatch (-want +got):\n%s", diff)
		}
	})
	t.Run("after quote", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeINCOMPLETE_TAG_OPEN, "<div"},
			[]interface{}{ml_parser.TokenTypeTEXT, "\""},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "<span"},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END, ">"},
			[]interface{}{ml_parser.TokenTypeTAG_CLOSE, "</span>"},
			[]interface{}{ml_parser.TokenTypeEOF, ""},
		}
		result := tokenizeAndHumanizeSourceSpans("<div \"<span></span>", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeSourceSpans() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("component tags", func(t *testing.T) {
		options := &ml_parser.TokenizeOptions{SelectorlessEnabled: boolPtr(true)}

		t.Run("should parse a basic component tag", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{ml_parser.TokenTypeCOMPONENT_OPEN_START, "MyComp", "", ""},
				[]interface{}{ml_parser.TokenTypeCOMPONENT_OPEN_END},
				[]interface{}{ml_parser.TokenTypeTEXT, "hello"},
				[]interface{}{ml_parser.TokenTypeCOMPONENT_CLOSE, "MyComp", "", ""},
				[]interface{}{ml_parser.TokenTypeEOF},
			}
			result := tokenizeAndHumanizeParts("<MyComp>hello</MyComp>", options)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should parse a component tag with a tag name", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{ml_parser.TokenTypeCOMPONENT_OPEN_START, "MyComp", "", "button"},
				[]interface{}{ml_parser.TokenTypeCOMPONENT_OPEN_END},
				[]interface{}{ml_parser.TokenTypeTEXT, "hello"},
				[]interface{}{ml_parser.TokenTypeCOMPONENT_CLOSE, "MyComp", "", "button"},
				[]interface{}{ml_parser.TokenTypeEOF},
			}
			result := tokenizeAndHumanizeParts("<MyComp:button>hello</MyComp:button>", options)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should parse a component tag with a tag name and namespace", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{ml_parser.TokenTypeCOMPONENT_OPEN_START, "MyComp", "svg", "title"},
				[]interface{}{ml_parser.TokenTypeCOMPONENT_OPEN_END},
				[]interface{}{ml_parser.TokenTypeTEXT, "hello"},
				[]interface{}{ml_parser.TokenTypeCOMPONENT_CLOSE, "MyComp", "svg", "title"},
				[]interface{}{ml_parser.TokenTypeEOF},
			}
			result := tokenizeAndHumanizeParts("<MyComp:svg:title>hello</MyComp:svg:title>", options)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should parse a self-closing component tag", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{ml_parser.TokenTypeCOMPONENT_OPEN_START, "MyComp", "", ""},
				[]interface{}{ml_parser.TokenTypeCOMPONENT_OPEN_END_VOID},
				[]interface{}{ml_parser.TokenTypeEOF},
			}
			result := tokenizeAndHumanizeParts("<MyComp/>", options)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should produce spans for component tags", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{ml_parser.TokenTypeCOMPONENT_OPEN_START, "<MyComp:svg:title"},
				[]interface{}{ml_parser.TokenTypeCOMPONENT_OPEN_END, ">"},
				[]interface{}{ml_parser.TokenTypeTEXT, "hello"},
				[]interface{}{ml_parser.TokenTypeCOMPONENT_CLOSE, "</MyComp:svg:title>"},
				[]interface{}{ml_parser.TokenTypeEOF, ""},
			}
			result := tokenizeAndHumanizeSourceSpans("<MyComp:svg:title>hello</MyComp:svg:title>", options)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("tokenizeAndHumanizeSourceSpans() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should parse an incomplete component open tag", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{ml_parser.TokenTypeINCOMPLETE_COMPONENT_OPEN, "MyComp", "", "span"},
				[]interface{}{ml_parser.TokenTypeATTR_NAME, "", "class"},
				[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
				[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, "hi"},
				[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
				[]interface{}{ml_parser.TokenTypeATTR_NAME, "", "sty"},
				[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "span"},
				[]interface{}{ml_parser.TokenTypeTAG_OPEN_END},
				[]interface{}{ml_parser.TokenTypeTAG_CLOSE, "", "span"},
				[]interface{}{ml_parser.TokenTypeEOF},
			}
			result := tokenizeAndHumanizeParts("<MyComp:span class=\"hi\" sty<span></span>", options)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should parse a component tag with raw text", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{ml_parser.TokenTypeCOMPONENT_OPEN_START, "MyComp", "", "script"},
				[]interface{}{ml_parser.TokenTypeCOMPONENT_OPEN_END},
				[]interface{}{ml_parser.TokenTypeRAW_TEXT, "t\ne\ns\nt"},
				[]interface{}{ml_parser.TokenTypeCOMPONENT_CLOSE, "MyComp", "", "script"},
				[]interface{}{ml_parser.TokenTypeEOF},
			}
			result := tokenizeAndHumanizeParts("<MyComp:script>t\ne\rs\r\nt</MyComp:script>", options)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should parse a component tag with escapable raw text", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{ml_parser.TokenTypeCOMPONENT_OPEN_START, "MyComp", "", "title"},
				[]interface{}{ml_parser.TokenTypeCOMPONENT_OPEN_END},
				[]interface{}{ml_parser.TokenTypeESCAPABLE_RAW_TEXT, "t\ne\ns\nt"},
				[]interface{}{ml_parser.TokenTypeCOMPONENT_CLOSE, "MyComp", "", "title"},
				[]interface{}{ml_parser.TokenTypeEOF},
			}
			result := tokenizeAndHumanizeParts("<MyComp:title>t\ne\rs\r\nt</MyComp:title>", options)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
			}
		})
	})

	t.Run("selectorless directives", func(t *testing.T) {
		options := &ml_parser.TokenizeOptions{SelectorlessEnabled: boolPtr(true)}

		t.Run("should parse a basic directive", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "div"},
				[]interface{}{ml_parser.TokenTypeDIRECTIVE_NAME, "MyDir"},
				[]interface{}{ml_parser.TokenTypeTAG_OPEN_END},
				[]interface{}{ml_parser.TokenTypeTAG_CLOSE, "", "div"},
				[]interface{}{ml_parser.TokenTypeEOF},
			}
			result := tokenizeAndHumanizeParts("<div @MyDir></div>", options)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
			}
		})

		t.Run("should parse a directive with parentheses, but no attributes", func(t *testing.T) {
			expected := []interface{}{
				[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "div"},
				[]interface{}{ml_parser.TokenTypeDIRECTIVE_NAME, "MyDir"},
				[]interface{}{ml_parser.TokenTypeDIRECTIVE_OPEN},
				[]interface{}{ml_parser.TokenTypeDIRECTIVE_CLOSE},
				[]interface{}{ml_parser.TokenTypeTAG_OPEN_END},
				[]interface{}{ml_parser.TokenTypeTAG_CLOSE, "", "div"},
				[]interface{}{ml_parser.TokenTypeEOF},
			}
			result := tokenizeAndHumanizeParts("<div @MyDir()></div>", options)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
			}
		})
	})
}

func TestHtmlLexer_EscapedStrings(t *testing.T) {
	t.Run("should unescape standard escape sequences", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTEXT, "' ' '"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("\\' \\' \\'", &ml_parser.TokenizeOptions{EscapedString: boolPtr(true)})
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestHtmlLexer_LetDeclarations(t *testing.T) {
	t.Run("should parse a @let declaration", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeLET_START, "foo"},
			[]interface{}{ml_parser.TokenTypeLET_VALUE, "123 + 456"},
			[]interface{}{ml_parser.TokenTypeLET_END},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("@let foo = 123 + 456;", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse @let declarations with arbitrary number of spaces", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeLET_START, "foo"},
			[]interface{}{ml_parser.TokenTypeLET_VALUE, "123 + 456"},
			[]interface{}{ml_parser.TokenTypeLET_END},
			[]interface{}{ml_parser.TokenTypeEOF},
		}

		testCases := []string{
			"@let               foo       =          123 + 456;",
			"@let foo=123 + 456;",
			"@let foo =123 + 456;",
			"@let foo=   123 + 456;",
		}

		for _, testCase := range testCases {
			result := tokenizeAndHumanizeParts(testCase, nil)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("tokenizeAndHumanizeParts() mismatch for %q (-want +got):\n%s", testCase, diff)
			}
		}
	})

	t.Run("should parse a @let declaration with newlines before/after its name", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeLET_START, "foo"},
			[]interface{}{ml_parser.TokenTypeLET_VALUE, "123"},
			[]interface{}{ml_parser.TokenTypeLET_END},
			[]interface{}{ml_parser.TokenTypeEOF},
		}

		testCases := []string{
			"@let\nfoo = 123;",
			"@let    \nfoo = 123;",
			"@let    \n              foo = 123;",
			"@let foo\n= 123;",
			"@let foo\n       = 123;",
			"@let foo   \n   = 123;",
			"@let  \n   foo   \n   = 123;",
		}

		for _, testCase := range testCases {
			result := tokenizeAndHumanizeParts(testCase, nil)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("tokenizeAndHumanizeParts() mismatch for %q (-want +got):\n%s", testCase, diff)
			}
		}
	})

	t.Run("should parse a @let declaration with new lines in its value", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeLET_START, "foo"},
			[]interface{}{ml_parser.TokenTypeLET_VALUE, "123 + \n 456 + \n789\n"},
			[]interface{}{ml_parser.TokenTypeLET_END},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("@let foo = \n123 + \n 456 + \n789\n;", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse a @let declaration inside of a block", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeBLOCK_OPEN_START, "defer"},
			[]interface{}{ml_parser.TokenTypeBLOCK_OPEN_END},
			[]interface{}{ml_parser.TokenTypeLET_START, "foo"},
			[]interface{}{ml_parser.TokenTypeLET_VALUE, "123 + 456"},
			[]interface{}{ml_parser.TokenTypeLET_END},
			[]interface{}{ml_parser.TokenTypeBLOCK_CLOSE},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("@defer {@let foo = 123 + 456;}", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse @let declaration using semicolon inside of a string", func(t *testing.T) {
		expected1 := []interface{}{
			[]interface{}{ml_parser.TokenTypeLET_START, "foo"},
			[]interface{}{ml_parser.TokenTypeLET_VALUE, "'a; b'"},
			[]interface{}{ml_parser.TokenTypeLET_END},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result1 := tokenizeAndHumanizeParts(`@let foo = 'a; b';`, nil)
		if diff := cmp.Diff(expected1, result1); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}

		expected2 := []interface{}{
			[]interface{}{ml_parser.TokenTypeLET_START, "foo"},
			[]interface{}{ml_parser.TokenTypeLET_VALUE, `"';'"`},
			[]interface{}{ml_parser.TokenTypeLET_END},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result2 := tokenizeAndHumanizeParts(`@let foo = "';'";`, nil)
		if diff := cmp.Diff(expected2, result2); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse @let declaration using escaped quotes in a string", func(t *testing.T) {
		// Use double-quoted string to match TypeScript template literal behavior
		// In TypeScript: `'\\';\\''` where \\ is escaped to single \
		// In Go double-quoted string: "\\" is also escaped to single \
		markup := "@let foo = '\\';\\'' + \"\\\",\";"
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeLET_START, "foo"},
			// Use double-quoted string to match actual value content (backslashes are single bytes)
			// valueContent bytes: 39 92 39 59 92 39 39 32 43 32 34 92 34 44 34
			// Which is: '\';\' + "\","
			[]interface{}{ml_parser.TokenTypeLET_VALUE, "'\\';\\'' + \"\\\",\""},
			[]interface{}{ml_parser.TokenTypeLET_END},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts(markup, nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse @let declaration using function calls in its value", func(t *testing.T) {
		markup := "@let foo = fn(a, b) + fn2(c, d, e);"
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeLET_START, "foo"},
			[]interface{}{ml_parser.TokenTypeLET_VALUE, "fn(a, b) + fn2(c, d, e)"},
			[]interface{}{ml_parser.TokenTypeLET_END},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts(markup, nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse @let declarations using array literals in their value", func(t *testing.T) {
		expected1 := []interface{}{
			[]interface{}{ml_parser.TokenTypeLET_START, "foo"},
			[]interface{}{ml_parser.TokenTypeLET_VALUE, "[1, 2, 3]"},
			[]interface{}{ml_parser.TokenTypeLET_END},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result1 := tokenizeAndHumanizeParts("@let foo = [1, 2, 3];", nil)
		if diff := cmp.Diff(expected1, result1); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}

		expected2 := []interface{}{
			[]interface{}{ml_parser.TokenTypeLET_START, "foo"},
			[]interface{}{ml_parser.TokenTypeLET_VALUE, "[0, [foo[1]], 3]"},
			[]interface{}{ml_parser.TokenTypeLET_END},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result2 := tokenizeAndHumanizeParts("@let foo = [0, [foo[1]], 3];", nil)
		if diff := cmp.Diff(expected2, result2); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse @let declarations using object literals", func(t *testing.T) {
		expected1 := []interface{}{
			[]interface{}{ml_parser.TokenTypeLET_START, "foo"},
			[]interface{}{ml_parser.TokenTypeLET_VALUE, "{a: 1, b: {c: something + 2}}"},
			[]interface{}{ml_parser.TokenTypeLET_END},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result1 := tokenizeAndHumanizeParts("@let foo = {a: 1, b: {c: something + 2}};", nil)
		if diff := cmp.Diff(expected1, result1); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}

		expected2 := []interface{}{
			[]interface{}{ml_parser.TokenTypeLET_START, "foo"},
			[]interface{}{ml_parser.TokenTypeLET_VALUE, "{}"},
			[]interface{}{ml_parser.TokenTypeLET_END},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result2 := tokenizeAndHumanizeParts("@let foo = {};", nil)
		if diff := cmp.Diff(expected2, result2); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}

		expected3 := []interface{}{
			[]interface{}{ml_parser.TokenTypeLET_START, "foo"},
			[]interface{}{ml_parser.TokenTypeLET_VALUE, `{foo: ";"}`},
			[]interface{}{ml_parser.TokenTypeLET_END},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result3 := tokenizeAndHumanizeParts(`@let foo = {foo: ";"};`, nil)
		if diff := cmp.Diff(expected3, result3); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse a @let declaration containing complex expression", func(t *testing.T) {
		markup := `@let foo = fn({a: 1, b: [otherFn([{c: ";"}], 321, {d: [',']})]});`
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeLET_START, "foo"},
			[]interface{}{ml_parser.TokenTypeLET_VALUE, `fn({a: 1, b: [otherFn([{c: ";"}], 321, {d: [',']})]})`},
			[]interface{}{ml_parser.TokenTypeLET_END},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts(markup, nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should handle @let declaration with invalid syntax in the value", func(t *testing.T) {
		expected1 := []interface{}{
			[]interface{}{"Unexpected character \"EOF\"", "0:13"},
		}
		result1 := tokenizeAndHumanizeErrors(`@let foo = ";`, nil)
		if diff := cmp.Diff(expected1, result1); diff != "" {
			t.Errorf("tokenizeAndHumanizeErrors() mismatch (-want +got):\n%s", diff)
		}

		expected2 := []interface{}{
			[]interface{}{ml_parser.TokenTypeLET_START, "foo"},
			[]interface{}{ml_parser.TokenTypeLET_VALUE, "{a: 1,"},
			[]interface{}{ml_parser.TokenTypeLET_END},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result2 := tokenizeAndHumanizeParts(`@let foo = {a: 1,;`, nil)
		if diff := cmp.Diff(expected2, result2); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}

		expected3 := []interface{}{
			[]interface{}{ml_parser.TokenTypeLET_START, "foo"},
			[]interface{}{ml_parser.TokenTypeLET_VALUE, "[1, "},
			[]interface{}{ml_parser.TokenTypeLET_END},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result3 := tokenizeAndHumanizeParts(`@let foo = [1, ;`, nil)
		if diff := cmp.Diff(expected3, result3); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}

		expected4 := []interface{}{
			[]interface{}{ml_parser.TokenTypeLET_START, "foo"},
			[]interface{}{ml_parser.TokenTypeLET_VALUE, "fn("},
			[]interface{}{ml_parser.TokenTypeLET_END},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result4 := tokenizeAndHumanizeParts(`@let foo = fn(;`, nil)
		if diff := cmp.Diff(expected4, result4); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse a @let declaration without a value", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeLET_START, "foo"},
			[]interface{}{ml_parser.TokenTypeLET_VALUE, ""},
			[]interface{}{ml_parser.TokenTypeLET_END},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("@let foo =;", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should handle no space after @let", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeINCOMPLETE_LET, "@let"},
			[]interface{}{ml_parser.TokenTypeTEXT, "Foo = 123;"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("@letFoo = 123;", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should handle unsupported characters in the name of @let", func(t *testing.T) {
		expected1 := []interface{}{
			[]interface{}{ml_parser.TokenTypeINCOMPLETE_LET, "foo"},
			[]interface{}{ml_parser.TokenTypeTEXT, "\\bar = 123;"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result1 := tokenizeAndHumanizeParts("@let foo\\bar = 123;", nil)
		if diff := cmp.Diff(expected1, result1); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}

		expected2 := []interface{}{
			[]interface{}{ml_parser.TokenTypeINCOMPLETE_LET, ""},
			[]interface{}{ml_parser.TokenTypeTEXT, "#foo = 123;"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result2 := tokenizeAndHumanizeParts("@let #foo = 123;", nil)
		if diff := cmp.Diff(expected2, result2); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}

		expected3 := []interface{}{
			[]interface{}{ml_parser.TokenTypeINCOMPLETE_LET, "foo"},
			[]interface{}{ml_parser.TokenTypeTEXT, "bar = 123;"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result3 := tokenizeAndHumanizeParts("@let foo\nbar = 123;", nil)
		if diff := cmp.Diff(expected3, result3); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should handle digits in the name of an @let", func(t *testing.T) {
		expected1 := []interface{}{
			[]interface{}{ml_parser.TokenTypeLET_START, "a123"},
			[]interface{}{ml_parser.TokenTypeLET_VALUE, "foo"},
			[]interface{}{ml_parser.TokenTypeLET_END},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result1 := tokenizeAndHumanizeParts("@let a123 = foo;", nil)
		if diff := cmp.Diff(expected1, result1); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}

		expected2 := []interface{}{
			[]interface{}{ml_parser.TokenTypeINCOMPLETE_LET, ""},
			[]interface{}{ml_parser.TokenTypeTEXT, "123a = 123;"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result2 := tokenizeAndHumanizeParts("@let 123a = 123;", nil)
		if diff := cmp.Diff(expected2, result2); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should handle an @let declaration without an ending token", func(t *testing.T) {
		expected1 := []interface{}{
			[]interface{}{ml_parser.TokenTypeINCOMPLETE_LET, "foo"},
			[]interface{}{ml_parser.TokenTypeLET_VALUE, "123 + 456"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result1 := tokenizeAndHumanizeParts("@let foo = 123 + 456", nil)
		if diff := cmp.Diff(expected1, result1); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}

		expected2 := []interface{}{
			[]interface{}{ml_parser.TokenTypeINCOMPLETE_LET, "foo"},
			[]interface{}{ml_parser.TokenTypeLET_VALUE, "123 + 456                  "},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result2 := tokenizeAndHumanizeParts("@let foo = 123 + 456                  ", nil)
		if diff := cmp.Diff(expected2, result2); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}

		expected3 := []interface{}{
			[]interface{}{ml_parser.TokenTypeINCOMPLETE_LET, "foo"},
			[]interface{}{ml_parser.TokenTypeLET_VALUE, "123, bar = 456"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result3 := tokenizeAndHumanizeParts("@let foo = 123, bar = 456", nil)
		if diff := cmp.Diff(expected3, result3); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should not parse @let inside an interpolation", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTEXT, ""},
			[]interface{}{ml_parser.TokenTypeINTERPOLATION, "{{", " @let foo = 123; ", "}}"},
			[]interface{}{ml_parser.TokenTypeTEXT, ""},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("{{ @let foo = 123; }}", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestHtmlLexer_Attributes(t *testing.T) {
	t.Run("should parse attributes without prefix", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "t"},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "", "a"},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("<t a>", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse attributes with interpolation", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "t"},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "", "a"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, ""},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_INTERPOLATION, "{{", "v", "}}"},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, ""},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "", "b"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, "s"},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_INTERPOLATION, "{{", "m", "}}"},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, "e"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "", "c"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, "s"},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_INTERPOLATION, "{{", "m//c", "}}"},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, "e"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts(`<t a="{{v}}" b="s{{m}}e" c="s{{m//c}}e">`, nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should end interpolation on an unescaped matching quote", func(t *testing.T) {
		expected1 := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "t"},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "", "a"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, ""},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_INTERPOLATION, "{{", ` a \" ' b `},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, ""},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result1 := tokenizeAndHumanizeParts(`<t a="{{ a \" ' b ">`, nil)
		if diff := cmp.Diff(expected1, result1); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}

		expected2 := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "t"},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "", "a"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "'"},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, ""},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_INTERPOLATION, "{{", ` a " \' b `},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, ""},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "'"},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result2 := tokenizeAndHumanizeParts(`<t a='{{ a " \' b '>`, nil)
		if diff := cmp.Diff(expected2, result2); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse attributes with prefix", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "t"},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "ns1", "a"},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("<t ns1:a>", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse attributes whose prefix is not valid", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "t"},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "", "(ns1:a)"},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("<t (ns1:a)>", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse attributes with single quote value", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "t"},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "", "a"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "'"},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, "b"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "'"},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("<t a='b'>", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse attributes with double quote value", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "t"},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "", "a"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, "b"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts(`<t a="b">`, nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse attributes with unquoted value", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "t"},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "", "a"},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, "b"},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("<t a=b>", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse attributes with unquoted interpolation value", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "a"},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "", "a"},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, ""},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_INTERPOLATION, "{{", "link.text", "}}"},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, ""},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("<a a={{link.text}}>", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse bound inputs with expressions containing newlines", func(t *testing.T) {
		input := `<app-component
        [attr]="[
        {text: 'some text',url:'//www.google.com'},
        {text:'other text',url:'//www.google.com'}]">`
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "app-component"},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "", "[attr]"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, "[\n        {text: 'some text',url:'//www.google.com'},\n        {text:'other text',url:'//www.google.com'}]"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts(input, nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse attributes with empty quoted value", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "t"},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "", "a"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, ""},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts(`<t a="">`, nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should allow whitespace", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "t"},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "", "a"},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, "b"},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("<t a = b >", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse attributes with entities in values", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "t"},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "", "a"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, ""},
			[]interface{}{ml_parser.TokenTypeENCODED_ENTITY, "A", "&#65;"},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, ""},
			[]interface{}{ml_parser.TokenTypeENCODED_ENTITY, "A", "&#x41;"},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, ""},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts(`<t a="&#65;&#x41;">`, nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should not decode entities without trailing \";\"", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "t"},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "", "a"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, "&amp"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "", "b"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, "c&&d"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts(`<t a="&amp" b="c&&d">`, nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse attributes with \"&\" in values", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "t"},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "", "a"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, "b && c &"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts(`<t a="b && c &">`, nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse values with CR and LF", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "t"},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "", "a"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "'"},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, "t\ne\ns\nt"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "'"},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("<t a='t\ne\rs\r\nt'>", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should store the locations", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "<t"},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "a"},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, "b"},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END, ">"},
			[]interface{}{ml_parser.TokenTypeEOF, ""},
		}
		result := tokenizeAndHumanizeSourceSpans("<t a=b>", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeSourceSpans() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should report missing closing single quote", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{"Unexpected character \"EOF\"", "0:8"},
		}
		result := tokenizeAndHumanizeErrors("<t a='b>", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeErrors() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should report missing closing double quote", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{"Unexpected character \"EOF\"", "0:8"},
		}
		result := tokenizeAndHumanizeErrors(`<t a="b>`, nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeErrors() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should permit more characters in square-bracketed attributes", func(t *testing.T) {
		expected1 := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "foo"},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "", "[class.text-primary/80]"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, "expr"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END_VOID},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result1 := tokenizeAndHumanizeParts(`<foo [class.text-primary/80]="expr"/>`, nil)
		if diff := cmp.Diff(expected1, result1); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}

		expected2 := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "foo"},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "", "[class.data-active:text-green-300/80]"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, "expr"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END_VOID},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result2 := tokenizeAndHumanizeParts(`<foo [class.data-active:text-green-300/80]="expr"/>`, nil)
		if diff := cmp.Diff(expected2, result2); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}

		expected3 := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "foo"},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "", `[class.data-[size='large']:p-8]`},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, "expr"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END_VOID},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result3 := tokenizeAndHumanizeParts(`<foo [class.data-[size='large']:p-8] = "expr"/>`, nil)
		if diff := cmp.Diff(expected3, result3); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}

		expected4 := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "foo"},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "", `[class.data-[size='large']:p-8]`},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END_VOID},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result4 := tokenizeAndHumanizeParts(`<foo [class.data-[size='large']:p-8]/>`, nil)
		if diff := cmp.Diff(expected4, result4); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}

		expected5 := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "foo"},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "", `[class.data-[size='hello white space']]`},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, "expr"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END_VOID},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result5 := tokenizeAndHumanizeParts(`<foo [class.data-[size='hello white space']]="expr"/>`, nil)
		if diff := cmp.Diff(expected5, result5); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}

		expected6 := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "foo"},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "", "[class.text-primary/80]"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, "expr"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "", "[class.data-active:text-green-300/80]"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, "expr2"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "", `[class.data-[size='large']:p-8]`},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, "expr3"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "", "some-attr"},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END_VOID},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result6 := tokenizeAndHumanizeParts(`<foo [class.text-primary/80]="expr" `+`[class.data-active:text-green-300/80]="expr2" `+`[class.data-[size='large']:p-8] = "expr3" some-attr/>`, nil)
		if diff := cmp.Diff(expected6, result6); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should allow mismatched square brackets in attribute name", func(t *testing.T) {
		expected1 := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "foo"},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "", "[class.a]b]c]"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, "expr"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END_VOID},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result1 := tokenizeAndHumanizeParts(`<foo [class.a]b]c]="expr"/>`, nil)
		if diff := cmp.Diff(expected1, result1); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}

		expected2 := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "foo"},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "", "[class.a[]][[]]b]][c]"},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END_VOID},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result2 := tokenizeAndHumanizeParts(`<foo [class.a[]][[]]b]][c]/>`, nil)
		if diff := cmp.Diff(expected2, result2); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should stop permissive parsing of square brackets on new line", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeINCOMPLETE_TAG_OPEN, "", "foo"},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "", "[class.text-"},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "", "primary"},
			[]interface{}{ml_parser.TokenTypeTEXT, "80]=\"expr\"/>"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("<foo [class.text-\nprimary/80]=\"expr\"/>", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestHtmlLexer_ClosingTags(t *testing.T) {
	t.Run("should parse closing tags without prefix", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_CLOSE, "", "test"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("</test>", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse closing tags with prefix", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_CLOSE, "ns1", "test"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("</ns1:test>", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should allow whitespace", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_CLOSE, "", "test"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("</ test >", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should store the locations", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_CLOSE, "</test>"},
			[]interface{}{ml_parser.TokenTypeEOF, ""},
		}
		result := tokenizeAndHumanizeSourceSpans("</test>", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeSourceSpans() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should report missing name after </", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{"Unexpected character \"EOF\"", "0:2"},
		}
		result := tokenizeAndHumanizeErrors("</", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeErrors() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should report missing >", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{"Unexpected character \"EOF\"", "0:6"},
		}
		result := tokenizeAndHumanizeErrors("</test", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeErrors() mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestHtmlLexer_Entities(t *testing.T) {
	t.Run("should parse named entities", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTEXT, "a"},
			[]interface{}{ml_parser.TokenTypeENCODED_ENTITY, "&", "&amp;"},
			[]interface{}{ml_parser.TokenTypeTEXT, "b"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("a&amp;b", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse hexadecimal entities", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTEXT, ""},
			[]interface{}{ml_parser.TokenTypeENCODED_ENTITY, "A", "&#x41;"},
			[]interface{}{ml_parser.TokenTypeTEXT, ""},
			[]interface{}{ml_parser.TokenTypeENCODED_ENTITY, "A", "&#X41;"},
			[]interface{}{ml_parser.TokenTypeTEXT, ""},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("&#x41;&#X41;", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse decimal entities", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTEXT, ""},
			[]interface{}{ml_parser.TokenTypeENCODED_ENTITY, "A", "&#65;"},
			[]interface{}{ml_parser.TokenTypeTEXT, ""},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("&#65;", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse entities with more than 4 hex digits", func(t *testing.T) {
		// Test 5 hex digit entity: &#x1F6C8; (🛈 - Circled Information Source)
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTEXT, ""},
			[]interface{}{ml_parser.TokenTypeENCODED_ENTITY, "\U0001F6C8", "&#x1F6C8;"},
			[]interface{}{ml_parser.TokenTypeTEXT, ""},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("&#x1F6C8;", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse entities with more than 4 decimal digits", func(t *testing.T) {
		// Test decimal entity: &#128712; (🛈 - Circled Information Source)
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTEXT, ""},
			[]interface{}{ml_parser.TokenTypeENCODED_ENTITY, "\U0001F6C8", "&#128712;"},
			[]interface{}{ml_parser.TokenTypeTEXT, ""},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("&#128712;", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should store the locations", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTEXT, "a"},
			[]interface{}{ml_parser.TokenTypeENCODED_ENTITY, "&amp;"},
			[]interface{}{ml_parser.TokenTypeTEXT, "b"},
			[]interface{}{ml_parser.TokenTypeEOF, ""},
		}
		result := tokenizeAndHumanizeSourceSpans("a&amp;b", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeSourceSpans() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should report malformed/unknown entities", func(t *testing.T) {
		expected1 := []interface{}{
			[]interface{}{"Unknown entity \"tbo\" - use the \"&#<decimal>;\" or  \"&#x<hex>;\" syntax", "0:0"},
		}
		result1 := tokenizeAndHumanizeErrors("&tbo;", nil)
		if diff := cmp.Diff(expected1, result1); diff != "" {
			t.Errorf("tokenizeAndHumanizeErrors() mismatch (-want +got):\n%s", diff)
		}

		expected2 := []interface{}{
			[]interface{}{"Unable to parse entity \"&#3s\" - decimal character reference entities must end with \";\"", "0:4"},
		}
		result2 := tokenizeAndHumanizeErrors("&#3sdf;", nil)
		if diff := cmp.Diff(expected2, result2); diff != "" {
			t.Errorf("tokenizeAndHumanizeErrors() mismatch (-want +got):\n%s", diff)
		}

		expected3 := []interface{}{
			[]interface{}{"Unable to parse entity \"&#xas\" - hexadecimal character reference entities must end with \";\"", "0:5"},
		}
		result3 := tokenizeAndHumanizeErrors("&#xasdf;", nil)
		if diff := cmp.Diff(expected3, result3); diff != "" {
			t.Errorf("tokenizeAndHumanizeErrors() mismatch (-want +got):\n%s", diff)
		}

		expected4 := []interface{}{
			[]interface{}{"Unexpected character \"EOF\"", "0:6"},
		}
		result4 := tokenizeAndHumanizeErrors("&#xABC", nil)
		if diff := cmp.Diff(expected4, result4); diff != "" {
			t.Errorf("tokenizeAndHumanizeErrors() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should not parse js object methods", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{"Unknown entity \"valueOf\" - use the \"&#<decimal>;\" or  \"&#x<hex>;\" syntax", "0:0"},
		}
		result := tokenizeAndHumanizeErrors("&valueOf;", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeErrors() mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestHtmlLexer_RegularText(t *testing.T) {
	t.Run("should parse text", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTEXT, "a"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("a", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse interpolation", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTEXT, ""},
			[]interface{}{ml_parser.TokenTypeINTERPOLATION, "{{", " a ", "}}"},
			[]interface{}{ml_parser.TokenTypeTEXT, "b"},
			[]interface{}{ml_parser.TokenTypeINTERPOLATION, "{{", " c // comment ", "}}"},
			[]interface{}{ml_parser.TokenTypeTEXT, "d"},
			[]interface{}{ml_parser.TokenTypeINTERPOLATION, "{{", ` e "}} ' " f `, "}}"},
			[]interface{}{ml_parser.TokenTypeTEXT, "g"},
			[]interface{}{ml_parser.TokenTypeINTERPOLATION, "{{", ` h // " i `, "}}"},
			[]interface{}{ml_parser.TokenTypeTEXT, ""},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts(`{{ a }}b{{ c // comment }}d{{ e "}} ' " f }}g{{ h // " i }}`, nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}

		expected2 := []interface{}{
			[]interface{}{ml_parser.TokenTypeTEXT, ""},
			[]interface{}{ml_parser.TokenTypeINTERPOLATION, "{{ a }}"},
			[]interface{}{ml_parser.TokenTypeTEXT, "b"},
			[]interface{}{ml_parser.TokenTypeINTERPOLATION, "{{ c // comment }}"},
			[]interface{}{ml_parser.TokenTypeTEXT, ""},
			[]interface{}{ml_parser.TokenTypeEOF, ""},
		}
		result2 := tokenizeAndHumanizeSourceSpans("{{ a }}b{{ c // comment }}", nil)
		if diff := cmp.Diff(expected2, result2); diff != "" {
			t.Errorf("tokenizeAndHumanizeSourceSpans() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should handle CR & LF in text", func(t *testing.T) {
		expected1 := []interface{}{
			[]interface{}{ml_parser.TokenTypeTEXT, "t\ne\ns\nt"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result1 := tokenizeAndHumanizeParts("t\ne\rs\r\nt", nil)
		if diff := cmp.Diff(expected1, result1); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}

		expected2 := []interface{}{
			[]interface{}{ml_parser.TokenTypeTEXT, "t\ne\rs\r\nt"},
			[]interface{}{ml_parser.TokenTypeEOF, ""},
		}
		result2 := tokenizeAndHumanizeSourceSpans("t\ne\rs\r\nt", nil)
		if diff := cmp.Diff(expected2, result2); diff != "" {
			t.Errorf("tokenizeAndHumanizeSourceSpans() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should handle CR & LF in interpolation", func(t *testing.T) {
		expected1 := []interface{}{
			[]interface{}{ml_parser.TokenTypeTEXT, ""},
			[]interface{}{ml_parser.TokenTypeINTERPOLATION, "{{", "t\ne\ns\nt", "}}"},
			[]interface{}{ml_parser.TokenTypeTEXT, ""},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result1 := tokenizeAndHumanizeParts("{{t\ne\rs\r\nt}}", nil)
		if diff := cmp.Diff(expected1, result1); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}

		expected2 := []interface{}{
			[]interface{}{ml_parser.TokenTypeTEXT, ""},
			[]interface{}{ml_parser.TokenTypeINTERPOLATION, "{{t\ne\rs\r\nt}}"},
			[]interface{}{ml_parser.TokenTypeTEXT, ""},
			[]interface{}{ml_parser.TokenTypeEOF, ""},
		}
		result2 := tokenizeAndHumanizeSourceSpans("{{t\ne\rs\r\nt}}", nil)
		if diff := cmp.Diff(expected2, result2); diff != "" {
			t.Errorf("tokenizeAndHumanizeSourceSpans() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse entities", func(t *testing.T) {
		expected1 := []interface{}{
			[]interface{}{ml_parser.TokenTypeTEXT, "a"},
			[]interface{}{ml_parser.TokenTypeENCODED_ENTITY, "&", "&amp;"},
			[]interface{}{ml_parser.TokenTypeTEXT, "b"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result1 := tokenizeAndHumanizeParts("a&amp;b", nil)
		if diff := cmp.Diff(expected1, result1); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}

		expected2 := []interface{}{
			[]interface{}{ml_parser.TokenTypeTEXT, "a"},
			[]interface{}{ml_parser.TokenTypeENCODED_ENTITY, "&amp;"},
			[]interface{}{ml_parser.TokenTypeTEXT, "b"},
			[]interface{}{ml_parser.TokenTypeEOF, ""},
		}
		result2 := tokenizeAndHumanizeSourceSpans("a&amp;b", nil)
		if diff := cmp.Diff(expected2, result2); diff != "" {
			t.Errorf("tokenizeAndHumanizeSourceSpans() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse text starting with \"&\"", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTEXT, "a && b &"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("a && b &", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should store the locations", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTEXT, "a"},
			[]interface{}{ml_parser.TokenTypeEOF, ""},
		}
		result := tokenizeAndHumanizeSourceSpans("a", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeSourceSpans() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should allow \"<\" in text nodes", func(t *testing.T) {
		expected1 := []interface{}{
			[]interface{}{ml_parser.TokenTypeTEXT, ""},
			[]interface{}{ml_parser.TokenTypeINTERPOLATION, "{{", " a < b ? c : d ", "}}"},
			[]interface{}{ml_parser.TokenTypeTEXT, ""},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result1 := tokenizeAndHumanizeParts("{{ a < b ? c : d }}", nil)
		if diff := cmp.Diff(expected1, result1); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}

		expected2 := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "<p"},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END, ">"},
			[]interface{}{ml_parser.TokenTypeTEXT, "a"},
			[]interface{}{ml_parser.TokenTypeINCOMPLETE_TAG_OPEN, "<b"},
			[]interface{}{ml_parser.TokenTypeTAG_CLOSE, "</p>"},
			[]interface{}{ml_parser.TokenTypeEOF, ""},
		}
		result2 := tokenizeAndHumanizeSourceSpans("<p>a<b</p>", nil)
		if diff := cmp.Diff(expected2, result2); diff != "" {
			t.Errorf("tokenizeAndHumanizeSourceSpans() mismatch (-want +got):\n%s", diff)
		}

		expected3 := []interface{}{
			[]interface{}{ml_parser.TokenTypeTEXT, "< a>"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result3 := tokenizeAndHumanizeParts("< a>", nil)
		if diff := cmp.Diff(expected3, result3); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should break out of interpolation in text token on valid start tag", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTEXT, ""},
			[]interface{}{ml_parser.TokenTypeINTERPOLATION, "{{", " a "},
			[]interface{}{ml_parser.TokenTypeTEXT, ""},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "b"},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "", "&&"},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "", "c"},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END},
			[]interface{}{ml_parser.TokenTypeTEXT, " d "},
			[]interface{}{ml_parser.TokenTypeBLOCK_CLOSE},
			[]interface{}{ml_parser.TokenTypeBLOCK_CLOSE},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("{{ a <b && c > d }}", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should break out of interpolation in text token on valid comment", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTEXT, ""},
			[]interface{}{ml_parser.TokenTypeINTERPOLATION, "{{", " a }"},
			[]interface{}{ml_parser.TokenTypeTEXT, ""},
			[]interface{}{ml_parser.TokenTypeCOMMENT_START},
			[]interface{}{ml_parser.TokenTypeRAW_TEXT, ""},
			[]interface{}{ml_parser.TokenTypeCOMMENT_END},
			[]interface{}{ml_parser.TokenTypeBLOCK_CLOSE},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("{{ a }<!---->}", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should end interpolation on a valid closing tag", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "p"},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END},
			[]interface{}{ml_parser.TokenTypeTEXT, ""},
			[]interface{}{ml_parser.TokenTypeINTERPOLATION, "{{", " a "},
			[]interface{}{ml_parser.TokenTypeTEXT, ""},
			[]interface{}{ml_parser.TokenTypeTAG_CLOSE, "", "p"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("<p>{{ a </p>", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should break out of interpolation in text token on valid CDATA", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTEXT, ""},
			[]interface{}{ml_parser.TokenTypeINTERPOLATION, "{{", " a }"},
			[]interface{}{ml_parser.TokenTypeTEXT, ""},
			[]interface{}{ml_parser.TokenTypeCDATA_START},
			[]interface{}{ml_parser.TokenTypeRAW_TEXT, ""},
			[]interface{}{ml_parser.TokenTypeCDATA_END},
			[]interface{}{ml_parser.TokenTypeBLOCK_CLOSE},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("{{ a }<![CDATA[]]>}", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should ignore invalid start tag in interpolation", func(t *testing.T) {
		// Note that if the `<=` is considered an "end of text" then the following `{` would
		// incorrectly be considered part of an ICU.
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "code"},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END},
			[]interface{}{ml_parser.TokenTypeTEXT, ""},
			[]interface{}{ml_parser.TokenTypeINTERPOLATION, "{{", "'<={'", "}}"},
			[]interface{}{ml_parser.TokenTypeTEXT, ""},
			[]interface{}{ml_parser.TokenTypeTAG_CLOSE, "", "code"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts(`<code>{{'<={'}}</code>`, &ml_parser.TokenizeOptions{TokenizeExpansionForms: boolPtr(true)})
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse start tags quotes in place of an attribute name as text", func(t *testing.T) {
		expected1 := []interface{}{
			[]interface{}{ml_parser.TokenTypeINCOMPLETE_TAG_OPEN, "", "t"},
			[]interface{}{ml_parser.TokenTypeTEXT, "\">"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result1 := tokenizeAndHumanizeParts(`<t ">`, nil)
		if diff := cmp.Diff(expected1, result1); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}

		expected2 := []interface{}{
			[]interface{}{ml_parser.TokenTypeINCOMPLETE_TAG_OPEN, "", "t"},
			[]interface{}{ml_parser.TokenTypeTEXT, "'>"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result2 := tokenizeAndHumanizeParts("<t '>", nil)
		if diff := cmp.Diff(expected2, result2); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse start tags quotes in place of an attribute name (after a valid attribute)", func(t *testing.T) {
		expected1 := []interface{}{
			[]interface{}{ml_parser.TokenTypeINCOMPLETE_TAG_OPEN, "", "t"},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "", "a"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, "b"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			// TODO(ayazhafiz): the " symbol should be a synthetic attribute,
			// allowing us to complete the opening tag correctly.
			[]interface{}{ml_parser.TokenTypeTEXT, "\">"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result1 := tokenizeAndHumanizeParts(`<t a="b" ">`, nil)
		if diff := cmp.Diff(expected1, result1); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}

		expected2 := []interface{}{
			[]interface{}{ml_parser.TokenTypeINCOMPLETE_TAG_OPEN, "", "t"},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "", "a"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "'"},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, "b"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "'"},
			// TODO(ayazhafiz): the ' symbol should be a synthetic attribute,
			// allowing us to complete the opening tag correctly.
			[]interface{}{ml_parser.TokenTypeTEXT, "'>"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result2 := tokenizeAndHumanizeParts("<t a='b' '>", nil)
		if diff := cmp.Diff(expected2, result2); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should be able to escape {", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTEXT, ""},
			[]interface{}{ml_parser.TokenTypeINTERPOLATION, "{{", ` "{" `, "}}"},
			[]interface{}{ml_parser.TokenTypeTEXT, ""},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts(`{{ "{" }}`, nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should be able to escape {{", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTEXT, ""},
			[]interface{}{ml_parser.TokenTypeINTERPOLATION, "{{", ` "{{" `, "}}"},
			[]interface{}{ml_parser.TokenTypeTEXT, ""},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts(`{{ "{{" }}`, nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should capture everything up to the end of file in the interpolation expression part if there are mismatched quotes", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTEXT, ""},
			[]interface{}{ml_parser.TokenTypeINTERPOLATION, "{{", ` "{{a}}' }}`},
			[]interface{}{ml_parser.TokenTypeTEXT, ""},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts(`{{ "{{a}}' }}`, nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should treat expansion form as text when they are not parsed", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "span"},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END},
			[]interface{}{ml_parser.TokenTypeTEXT, "{a, b, =4 {c}}"},
			[]interface{}{ml_parser.TokenTypeTAG_CLOSE, "", "span"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("<span>{a, b, =4 {c}}</span>", &ml_parser.TokenizeOptions{TokenizeExpansionForms: boolPtr(false), TokenizeBlocks: boolPtr(false)})
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestHtmlLexer_RawText(t *testing.T) {
	t.Run("should parse text", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "script"},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END},
			[]interface{}{ml_parser.TokenTypeRAW_TEXT, "t\ne\ns\nt"},
			[]interface{}{ml_parser.TokenTypeTAG_CLOSE, "", "script"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts(`<script>t\ne\rs\r\nt</script>`, nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should not detect entities", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "script"},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END},
			[]interface{}{ml_parser.TokenTypeRAW_TEXT, "&amp;"},
			[]interface{}{ml_parser.TokenTypeTAG_CLOSE, "", "script"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts(`<script>&amp;</SCRIPT>`, nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should ignore other opening tags", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "script"},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END},
			[]interface{}{ml_parser.TokenTypeRAW_TEXT, "a<div>"},
			[]interface{}{ml_parser.TokenTypeTAG_CLOSE, "", "script"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts(`<script>a<div></script>`, nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should ignore other closing tags", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "script"},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END},
			[]interface{}{ml_parser.TokenTypeRAW_TEXT, "a</test>"},
			[]interface{}{ml_parser.TokenTypeTAG_CLOSE, "", "script"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts(`<script>a</test></script>`, nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should store the locations", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "<script"},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END, ">"},
			[]interface{}{ml_parser.TokenTypeRAW_TEXT, "a"},
			[]interface{}{ml_parser.TokenTypeTAG_CLOSE, "</script>"},
			[]interface{}{ml_parser.TokenTypeEOF, ""},
		}
		result := tokenizeAndHumanizeSourceSpans(`<script>a</script>`, nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeSourceSpans() mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestHtmlLexer_Blocks(t *testing.T) {
	t.Run("should parse a block without parameters", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeBLOCK_OPEN_START, "if"},
			[]interface{}{ml_parser.TokenTypeBLOCK_OPEN_END},
			[]interface{}{ml_parser.TokenTypeTEXT, "hello"},
			[]interface{}{ml_parser.TokenTypeBLOCK_CLOSE},
			[]interface{}{ml_parser.TokenTypeEOF},
		}

		testCases := []string{
			"@if {hello}",
			"@if () {hello}",
			"@if(){hello}",
		}

		for _, testCase := range testCases {
			result := tokenizeAndHumanizeParts(testCase, nil)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("tokenizeAndHumanizeParts() mismatch for %q (-want +got):\n%s", testCase, diff)
			}
		}
	})

	t.Run("should parse a block with parameters", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeBLOCK_OPEN_START, "for"},
			[]interface{}{ml_parser.TokenTypeBLOCK_PARAMETER, "item of items"},
			[]interface{}{ml_parser.TokenTypeBLOCK_PARAMETER, "track item.id"},
			[]interface{}{ml_parser.TokenTypeBLOCK_OPEN_END},
			[]interface{}{ml_parser.TokenTypeTEXT, "hello"},
			[]interface{}{ml_parser.TokenTypeBLOCK_CLOSE},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("@for (item of items; track item.id) {hello}", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse a block with a trailing semicolon after the parameters", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeBLOCK_OPEN_START, "for"},
			[]interface{}{ml_parser.TokenTypeBLOCK_PARAMETER, "item of items"},
			[]interface{}{ml_parser.TokenTypeBLOCK_OPEN_END},
			[]interface{}{ml_parser.TokenTypeTEXT, "hello"},
			[]interface{}{ml_parser.TokenTypeBLOCK_CLOSE},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("@for (item of items;) {hello}", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse a block with a space in its name", func(t *testing.T) {
		expected1 := []interface{}{
			[]interface{}{ml_parser.TokenTypeBLOCK_OPEN_START, "else if"},
			[]interface{}{ml_parser.TokenTypeBLOCK_OPEN_END},
			[]interface{}{ml_parser.TokenTypeTEXT, "hello"},
			[]interface{}{ml_parser.TokenTypeBLOCK_CLOSE},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result1 := tokenizeAndHumanizeParts("@else if {hello}", nil)
		if diff := cmp.Diff(expected1, result1); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}

		expected2 := []interface{}{
			[]interface{}{ml_parser.TokenTypeBLOCK_OPEN_START, "else if"},
			[]interface{}{ml_parser.TokenTypeBLOCK_PARAMETER, "foo !== 2"},
			[]interface{}{ml_parser.TokenTypeBLOCK_OPEN_END},
			[]interface{}{ml_parser.TokenTypeTEXT, "hello"},
			[]interface{}{ml_parser.TokenTypeBLOCK_CLOSE},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result2 := tokenizeAndHumanizeParts("@else if (foo !== 2) {hello}", nil)
		if diff := cmp.Diff(expected2, result2); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse a block with an arbitrary amount of spaces around the parentheses", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeBLOCK_OPEN_START, "for"},
			[]interface{}{ml_parser.TokenTypeBLOCK_PARAMETER, "a"},
			[]interface{}{ml_parser.TokenTypeBLOCK_PARAMETER, "b"},
			[]interface{}{ml_parser.TokenTypeBLOCK_PARAMETER, "c"},
			[]interface{}{ml_parser.TokenTypeBLOCK_OPEN_END},
			[]interface{}{ml_parser.TokenTypeTEXT, "hello"},
			[]interface{}{ml_parser.TokenTypeBLOCK_CLOSE},
			[]interface{}{ml_parser.TokenTypeEOF},
		}

		testCases := []string{
			"@for(a; b; c){hello}",
			"@for      (a; b; c)      {hello}",
			"@for(a; b; c)      {hello}",
			"@for      (a; b; c){hello}",
		}

		for _, testCase := range testCases {
			result := tokenizeAndHumanizeParts(testCase, nil)
			if diff := cmp.Diff(expected, result); diff != "" {
				t.Errorf("tokenizeAndHumanizeParts() mismatch for %q (-want +got):\n%s", testCase, diff)
			}
		}
	})

	t.Run("should parse a block with multiple trailing semicolons", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeBLOCK_OPEN_START, "for"},
			[]interface{}{ml_parser.TokenTypeBLOCK_PARAMETER, "item of items"},
			[]interface{}{ml_parser.TokenTypeBLOCK_OPEN_END},
			[]interface{}{ml_parser.TokenTypeTEXT, "hello"},
			[]interface{}{ml_parser.TokenTypeBLOCK_CLOSE},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("@for (item of items;;;;;) {hello}", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse a block with trailing whitespace", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeBLOCK_OPEN_START, "defer"},
			[]interface{}{ml_parser.TokenTypeBLOCK_OPEN_END},
			[]interface{}{ml_parser.TokenTypeTEXT, "hello"},
			[]interface{}{ml_parser.TokenTypeBLOCK_CLOSE},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("@defer                        {hello}", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse a block with no trailing semicolon", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeBLOCK_OPEN_START, "for"},
			[]interface{}{ml_parser.TokenTypeBLOCK_PARAMETER, "item of items"},
			[]interface{}{ml_parser.TokenTypeBLOCK_OPEN_END},
			[]interface{}{ml_parser.TokenTypeTEXT, "hello"},
			[]interface{}{ml_parser.TokenTypeBLOCK_CLOSE},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("@for (item of items){hello}", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should handle semicolons, braces and parentheses used in a block parameter", func(t *testing.T) {
		input := `@for (a === ";"; b === ')'; c === "("; d === '}'; e === "{") {hello}`
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeBLOCK_OPEN_START, "for"},
			[]interface{}{ml_parser.TokenTypeBLOCK_PARAMETER, `a === ";"`},
			[]interface{}{ml_parser.TokenTypeBLOCK_PARAMETER, `b === ')'`},
			[]interface{}{ml_parser.TokenTypeBLOCK_PARAMETER, `c === "("`},
			[]interface{}{ml_parser.TokenTypeBLOCK_PARAMETER, `d === '}'`},
			[]interface{}{ml_parser.TokenTypeBLOCK_PARAMETER, `e === "{"`},
			[]interface{}{ml_parser.TokenTypeBLOCK_OPEN_END},
			[]interface{}{ml_parser.TokenTypeTEXT, "hello"},
			[]interface{}{ml_parser.TokenTypeBLOCK_CLOSE},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts(input, nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should handle object literals and function calls in block parameters", func(t *testing.T) {
		input := `@defer (on a({a: 1, b: 2}, false, {c: 3}); when b({d: 4})) {hello}`
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeBLOCK_OPEN_START, "defer"},
			[]interface{}{ml_parser.TokenTypeBLOCK_PARAMETER, "on a({a: 1, b: 2}, false, {c: 3})"},
			[]interface{}{ml_parser.TokenTypeBLOCK_PARAMETER, "when b({d: 4})"},
			[]interface{}{ml_parser.TokenTypeBLOCK_OPEN_END},
			[]interface{}{ml_parser.TokenTypeTEXT, "hello"},
			[]interface{}{ml_parser.TokenTypeBLOCK_CLOSE},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts(input, nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse block with unclosed parameters", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeINCOMPLETE_BLOCK_OPEN, "if"},
			[]interface{}{ml_parser.TokenTypeBLOCK_PARAMETER, "a === b {hello}"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts(`@if (a === b {hello}`, nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse block with stray parentheses in the parameter position", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeINCOMPLETE_BLOCK_OPEN, "if a"},
			[]interface{}{ml_parser.TokenTypeTEXT, "=== b) {hello"},
			[]interface{}{ml_parser.TokenTypeBLOCK_CLOSE},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts(`@if a === b) {hello}`, nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should report invalid quotes in a parameter", func(t *testing.T) {
		expected1 := []interface{}{
			[]interface{}{"Unexpected character \"EOF\"", "0:21"},
		}
		result1 := tokenizeAndHumanizeErrors(`@if (a === ") {hello}`, nil)
		if diff := cmp.Diff(expected1, result1); diff != "" {
			t.Errorf("tokenizeAndHumanizeErrors() mismatch (-want +got):\n%s", diff)
		}

		expected2 := []interface{}{
			[]interface{}{"Unexpected character \"EOF\"", "0:24"},
		}
		result2 := tokenizeAndHumanizeErrors(`@if (a === "hi') {hello}`, nil)
		if diff := cmp.Diff(expected2, result2); diff != "" {
			t.Errorf("tokenizeAndHumanizeErrors() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should report unclosed object literal inside a parameter", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeINCOMPLETE_BLOCK_OPEN, "if"},
			[]interface{}{ml_parser.TokenTypeBLOCK_PARAMETER, "{invalid: true"},
			[]interface{}{ml_parser.TokenTypeTEXT, "hello"},
			[]interface{}{ml_parser.TokenTypeBLOCK_CLOSE},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts(`@if ({invalid: true) hello}`, nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should handle a semicolon used in a nested string inside a block parameter", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeBLOCK_OPEN_START, "if"},
			[]interface{}{ml_parser.TokenTypeBLOCK_PARAMETER, `condition === "';'"`},
			[]interface{}{ml_parser.TokenTypeBLOCK_OPEN_END},
			[]interface{}{ml_parser.TokenTypeTEXT, "hello"},
			[]interface{}{ml_parser.TokenTypeBLOCK_CLOSE},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts(`@if (condition === "';'") {hello}`, nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should handle a semicolon next to an escaped quote used in a block parameter", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeBLOCK_OPEN_START, "if"},
			[]interface{}{ml_parser.TokenTypeBLOCK_PARAMETER, `condition === "\";"`},
			[]interface{}{ml_parser.TokenTypeBLOCK_OPEN_END},
			[]interface{}{ml_parser.TokenTypeTEXT, "hello"},
			[]interface{}{ml_parser.TokenTypeBLOCK_CLOSE},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts(`@if (condition === "\";") {hello}`, nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse mixed text and html content in a block", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeBLOCK_OPEN_START, "if"},
			[]interface{}{ml_parser.TokenTypeBLOCK_PARAMETER, "a === 1"},
			[]interface{}{ml_parser.TokenTypeBLOCK_OPEN_END},
			[]interface{}{ml_parser.TokenTypeTEXT, "foo "},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "b"},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END},
			[]interface{}{ml_parser.TokenTypeTEXT, "bar"},
			[]interface{}{ml_parser.TokenTypeTAG_CLOSE, "", "b"},
			[]interface{}{ml_parser.TokenTypeTEXT, " baz"},
			[]interface{}{ml_parser.TokenTypeBLOCK_CLOSE},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("@if (a === 1) {foo <b>bar</b> baz}", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse HTML tags with attributes containing curly braces inside blocks", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeBLOCK_OPEN_START, "if"},
			[]interface{}{ml_parser.TokenTypeBLOCK_PARAMETER, "a === 1"},
			[]interface{}{ml_parser.TokenTypeBLOCK_OPEN_END},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "div"},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "", "a"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, "}"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "", "b"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, "{"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END},
			[]interface{}{ml_parser.TokenTypeTAG_CLOSE, "", "div"},
			[]interface{}{ml_parser.TokenTypeBLOCK_CLOSE},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts(`@if (a === 1) {<div a="}" b="{"></div>}`, nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse HTML tags with attribute containing block syntax", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_START, "", "div"},
			[]interface{}{ml_parser.TokenTypeATTR_NAME, "", "a"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeATTR_VALUE_TEXT, "@if (foo) {}"},
			[]interface{}{ml_parser.TokenTypeATTR_QUOTE, "\""},
			[]interface{}{ml_parser.TokenTypeTAG_OPEN_END},
			[]interface{}{ml_parser.TokenTypeTAG_CLOSE, "", "div"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts(`<div a="@if (foo) {}"></div>`, nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse nested blocks", func(t *testing.T) {
		input := "@if (a) {" +
			"hello a" +
			"@if {" +
			"hello unnamed" +
			"@if (b) {" +
			"hello b" +
			"@if (c) {" +
			"hello c" +
			"}" +
			"}" +
			"}" +
			"}"
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeBLOCK_OPEN_START, "if"},
			[]interface{}{ml_parser.TokenTypeBLOCK_PARAMETER, "a"},
			[]interface{}{ml_parser.TokenTypeBLOCK_OPEN_END},
			[]interface{}{ml_parser.TokenTypeTEXT, "hello a"},
			[]interface{}{ml_parser.TokenTypeBLOCK_OPEN_START, "if"},
			[]interface{}{ml_parser.TokenTypeBLOCK_OPEN_END},
			[]interface{}{ml_parser.TokenTypeTEXT, "hello unnamed"},
			[]interface{}{ml_parser.TokenTypeBLOCK_OPEN_START, "if"},
			[]interface{}{ml_parser.TokenTypeBLOCK_PARAMETER, "b"},
			[]interface{}{ml_parser.TokenTypeBLOCK_OPEN_END},
			[]interface{}{ml_parser.TokenTypeTEXT, "hello b"},
			[]interface{}{ml_parser.TokenTypeBLOCK_OPEN_START, "if"},
			[]interface{}{ml_parser.TokenTypeBLOCK_PARAMETER, "c"},
			[]interface{}{ml_parser.TokenTypeBLOCK_OPEN_END},
			[]interface{}{ml_parser.TokenTypeTEXT, "hello c"},
			[]interface{}{ml_parser.TokenTypeBLOCK_CLOSE},
			[]interface{}{ml_parser.TokenTypeBLOCK_CLOSE},
			[]interface{}{ml_parser.TokenTypeBLOCK_CLOSE},
			[]interface{}{ml_parser.TokenTypeBLOCK_CLOSE},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts(input, nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse a block containing an expansion", func(t *testing.T) {
		result := tokenizeAndHumanizeParts(
			"@defer {{one.two, three, =4 {four} =5 {five} foo {bar} }}",
			&ml_parser.TokenizeOptions{TokenizeExpansionForms: boolPtr(true)},
		)

		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeBLOCK_OPEN_START, "defer"},
			[]interface{}{ml_parser.TokenTypeBLOCK_OPEN_END},
			[]interface{}{ml_parser.TokenTypeEXPANSION_FORM_START},
			[]interface{}{ml_parser.TokenTypeRAW_TEXT, "one.two"},
			[]interface{}{ml_parser.TokenTypeRAW_TEXT, "three"},
			[]interface{}{ml_parser.TokenTypeEXPANSION_CASE_VALUE, "=4"},
			[]interface{}{ml_parser.TokenTypeEXPANSION_CASE_EXP_START},
			[]interface{}{ml_parser.TokenTypeTEXT, "four"},
			[]interface{}{ml_parser.TokenTypeEXPANSION_CASE_EXP_END},
			[]interface{}{ml_parser.TokenTypeEXPANSION_CASE_VALUE, "=5"},
			[]interface{}{ml_parser.TokenTypeEXPANSION_CASE_EXP_START},
			[]interface{}{ml_parser.TokenTypeTEXT, "five"},
			[]interface{}{ml_parser.TokenTypeEXPANSION_CASE_EXP_END},
			[]interface{}{ml_parser.TokenTypeEXPANSION_CASE_VALUE, "foo"},
			[]interface{}{ml_parser.TokenTypeEXPANSION_CASE_EXP_START},
			[]interface{}{ml_parser.TokenTypeTEXT, "bar"},
			[]interface{}{ml_parser.TokenTypeEXPANSION_CASE_EXP_END},
			[]interface{}{ml_parser.TokenTypeEXPANSION_FORM_END},
			[]interface{}{ml_parser.TokenTypeBLOCK_CLOSE},
			[]interface{}{ml_parser.TokenTypeEOF},
		}

		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse a block containing an interpolation", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeBLOCK_OPEN_START, "defer"},
			[]interface{}{ml_parser.TokenTypeBLOCK_OPEN_END},
			[]interface{}{ml_parser.TokenTypeTEXT, ""},
			[]interface{}{ml_parser.TokenTypeINTERPOLATION, "{{", "message", "}}"},
			[]interface{}{ml_parser.TokenTypeTEXT, ""},
			[]interface{}{ml_parser.TokenTypeBLOCK_CLOSE},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("@defer {{{message}}}", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse an incomplete block start without parameters with surrounding text", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTEXT, "My email frodo"},
			[]interface{}{ml_parser.TokenTypeINCOMPLETE_BLOCK_OPEN, "for"},
			[]interface{}{ml_parser.TokenTypeTEXT, ".com"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("My email frodo@for.com", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse an incomplete block start at the end of the input", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTEXT, "My favorite console is "},
			[]interface{}{ml_parser.TokenTypeINCOMPLETE_BLOCK_OPEN, "switch"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("My favorite console is @switch", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse an incomplete block start with parentheses but without params", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTEXT, "Use the "},
			[]interface{}{ml_parser.TokenTypeINCOMPLETE_BLOCK_OPEN, "for"},
			[]interface{}{ml_parser.TokenTypeTEXT, "block"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("Use the @for() block", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse an incomplete block start with parentheses and params", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTEXT, "This is the "},
			[]interface{}{ml_parser.TokenTypeINCOMPLETE_BLOCK_OPEN, "if"},
			[]interface{}{ml_parser.TokenTypeBLOCK_PARAMETER, `{alias: "foo"}`},
			[]interface{}{ml_parser.TokenTypeTEXT, "expression"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts(`This is the @if({alias: "foo"}) expression`, nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse @ as text", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTEXT, "@"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("@", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse space followed by @ as text", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTEXT, " @"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts(" @", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse @ followed by space as text", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTEXT, "@ "},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("@ ", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse @ followed by newline and text as text", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTEXT, "@\nfoo"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("@\nfoo", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse @ in the middle of text as text", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTEXT, "foo bar @ baz clink"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("foo bar @ baz clink", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should parse incomplete block with space, then name as text", func(t *testing.T) {
		expected := []interface{}{
			[]interface{}{ml_parser.TokenTypeTEXT, "@ if"},
			[]interface{}{ml_parser.TokenTypeEOF},
		}
		result := tokenizeAndHumanizeParts("@ if", nil)
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("tokenizeAndHumanizeParts() mismatch (-want +got):\n%s", diff)
		}
	})
}
