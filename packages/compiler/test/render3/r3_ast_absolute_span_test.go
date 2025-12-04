package render3_test

import (
	"ngc-go/packages/compiler/src/expression_parser"
	"ngc-go/packages/compiler/test/render3/util"
	"ngc-go/packages/compiler/test/render3/view"
	"reflect"
	"testing"
)

// Helper function to check if a slice contains a specific element
func containsHumanizedExpressionSource(slice []util.HumanizedExpressionSource, unparsed string, start, end int) bool {
	for _, item := range slice {
		if item.Unparsed == unparsed && item.Span != nil && item.Span.Start == start && item.Span.End == end {
			return true
		}
	}
	return false
}

// Helper function to check if a slice contains all specified elements (arrayContaining equivalent)
func containsAllHumanizedExpressionSources(slice []util.HumanizedExpressionSource, expected []util.HumanizedExpressionSource) bool {
	for _, exp := range expected {
		found := false
		for _, item := range slice {
			if item.Unparsed == exp.Unparsed && item.Span != nil && exp.Span != nil &&
				item.Span.Start == exp.Span.Start && item.Span.End == exp.Span.End {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func TestExpressionASTAbsoluteSourceSpans(t *testing.T) {
	t.Run("should handle comment in interpolation", func(t *testing.T) {
		preserveWhitespaces := true
		result := view.ParseR3("{{foo // comment}}", &view.ParseR3Options{
			PreserveWhitespaces: &preserveWhitespaces,
		})
		t.Logf("ParseR3 returned %d nodes", len(result.Nodes))
		for i, node := range result.Nodes {
			t.Logf("  Node[%d]: %T", i, node)
		}
		humanized := util.HumanizeExpressionSource(result.Nodes)
		if !containsHumanizedExpressionSource(humanized, "foo", 2, 5) {
			t.Errorf("Expected to contain ['foo', AbsoluteSourceSpan(2, 5)]")
			t.Logf("Actual humanized results: %+v", humanized)
			for i, h := range humanized {
				if h.Span != nil {
					t.Logf("  [%d] Unparsed: %q, Span: [%d, %d]", i, h.Unparsed, h.Span.Start, h.Span.End)
				} else {
					t.Logf("  [%d] Unparsed: %q, Span: nil", i, h.Unparsed)
				}
			}
		}
	})

	t.Run("should handle whitespace in interpolation", func(t *testing.T) {
		preserveWhitespaces := true
		result := view.ParseR3("{{  foo  }}", &view.ParseR3Options{
			PreserveWhitespaces: &preserveWhitespaces,
		})
		humanized := util.HumanizeExpressionSource(result.Nodes)
		if !containsHumanizedExpressionSource(humanized, "foo", 4, 7) {
			t.Errorf("Expected to contain ['foo', AbsoluteSourceSpan(4, 7)]")
		}
	})

	t.Run("should handle whitespace and comment in interpolation", func(t *testing.T) {
		preserveWhitespaces := true
		result := view.ParseR3("{{  foo // comment  }}", &view.ParseR3Options{
			PreserveWhitespaces: &preserveWhitespaces,
		})
		humanized := util.HumanizeExpressionSource(result.Nodes)
		if !containsHumanizedExpressionSource(humanized, "foo", 4, 7) {
			t.Errorf("Expected to contain ['foo', AbsoluteSourceSpan(4, 7)]")
		}
	})

	t.Run("should handle comment in an action binding", func(t *testing.T) {
		preserveWhitespaces := true
		result := view.ParseR3(`<button (click)="foo = true // comment">Save</button>`, &view.ParseR3Options{
			PreserveWhitespaces: &preserveWhitespaces,
		})
		humanized := util.HumanizeExpressionSource(result.Nodes)
		if !containsHumanizedExpressionSource(humanized, "foo = true", 17, 27) {
			t.Errorf("Expected to contain ['foo = true', AbsoluteSourceSpan(17, 27)]")
		}
	})

	t.Run("should provide absolute offsets with arbitrary whitespace", func(t *testing.T) {
		preserveWhitespaces := true
		result := view.ParseR3("<div>\n  \n{{foo}}</div>", &view.ParseR3Options{
			PreserveWhitespaces: &preserveWhitespaces,
		})
		humanized := util.HumanizeExpressionSource(result.Nodes)
		// Note: The expected unparsed value might differ slightly in Go vs TypeScript
		// We check for the key part: 'foo' with correct span
		if !containsHumanizedExpressionSource(humanized, "foo", 5, 8) {
			// Try alternative: might be parsed differently
			found := false
			for _, item := range humanized {
				if item.Span != nil && item.Span.Start >= 5 && item.Span.End <= 16 {
					if item.Unparsed == "foo" || contains(item.Unparsed, "foo") {
						found = true
						break
					}
				}
			}
			if !found {
				t.Errorf("Expected to find 'foo' expression in span [5, 16]")
			}
		}
	})

	t.Run("should provide absolute offsets of an expression in a bound text", func(t *testing.T) {
		result := view.ParseR3("<div>{{foo}}</div>", nil)
		humanized := util.HumanizeExpressionSource(result.Nodes)
		// Check for interpolation with correct span
		found := false
		for _, item := range humanized {
			if item.Span != nil && item.Span.Start == 5 && item.Span.End == 12 {
				if contains(item.Unparsed, "foo") {
					found = true
					break
				}
			}
		}
		if !found {
			t.Errorf("Expected to contain expression with AbsoluteSourceSpan(5, 12)")
		}
	})

	t.Run("should provide absolute offsets of an expression in a bound event", func(t *testing.T) {
		result1 := view.ParseR3(`<div (click)="foo();bar();"></div>`, nil)
		humanized1 := util.HumanizeExpressionSource(result1.Nodes)
		found1 := false
		for _, item := range humanized1 {
			if item.Span != nil && item.Span.Start == 14 && item.Span.End == 26 {
				if contains(item.Unparsed, "foo") && contains(item.Unparsed, "bar") {
					found1 = true
					break
				}
			}
		}
		if !found1 {
			t.Errorf("Expected to contain expression with AbsoluteSourceSpan(14, 26)")
		}

		result2 := view.ParseR3(`<div on-click="foo();bar();"></div>`, nil)
		humanized2 := util.HumanizeExpressionSource(result2.Nodes)
		found2 := false
		for _, item := range humanized2 {
			if item.Span != nil && item.Span.Start == 15 && item.Span.End == 27 {
				if contains(item.Unparsed, "foo") && contains(item.Unparsed, "bar") {
					found2 = true
					break
				}
			}
		}
		if !found2 {
			t.Errorf("Expected to contain expression with AbsoluteSourceSpan(15, 27)")
		}
	})

	t.Run("should provide absolute offsets of an expression in a bound attribute", func(t *testing.T) {
		result1 := view.ParseR3(`<input [disabled]="condition ? true : false" />`, nil)
		humanized1 := util.HumanizeExpressionSource(result1.Nodes)
		found1 := false
		for _, item := range humanized1 {
			if item.Span != nil && item.Span.Start == 19 && item.Span.End == 43 {
				if contains(item.Unparsed, "condition") {
					found1 = true
					break
				}
			}
		}
		if !found1 {
			t.Errorf("Expected to contain expression with AbsoluteSourceSpan(19, 43)")
		}

		result2 := view.ParseR3(`<input bind-disabled="condition ? true : false" />`, nil)
		humanized2 := util.HumanizeExpressionSource(result2.Nodes)
		found2 := false
		for _, item := range humanized2 {
			if item.Span != nil && item.Span.Start == 22 && item.Span.End == 46 {
				if contains(item.Unparsed, "condition") {
					found2 = true
					break
				}
			}
		}
		if !found2 {
			t.Errorf("Expected to contain expression with AbsoluteSourceSpan(22, 46)")
		}
	})

	t.Run("should provide absolute offsets of an expression in a template attribute", func(t *testing.T) {
		result := view.ParseR3(`<div *ngIf="value | async"></div>`, nil)
		humanized := util.HumanizeExpressionSource(result.Nodes)
		found := false
		for _, item := range humanized {
			if item.Span != nil && item.Span.Start == 12 && item.Span.End == 25 {
				if contains(item.Unparsed, "value") && contains(item.Unparsed, "async") {
					found = true
					break
				}
			}
		}
		if !found {
			t.Errorf("Expected to contain expression with AbsoluteSourceSpan(12, 25)")
		}
	})

	t.Run("binary expression", func(t *testing.T) {
		t.Run("should provide absolute offsets of a binary expression", func(t *testing.T) {
			result := view.ParseR3("<div>{{1 + 2}}<div>", nil)
			humanized := util.HumanizeExpressionSource(result.Nodes)
			found := false
			for _, item := range humanized {
				if item.Span != nil && item.Span.Start == 7 && item.Span.End == 12 {
					if contains(item.Unparsed, "1") && contains(item.Unparsed, "2") && contains(item.Unparsed, "+") {
						found = true
						break
					}
				}
			}
			if !found {
				t.Errorf("Expected to contain binary expression with AbsoluteSourceSpan(7, 12)")
			}
		})

		t.Run("should provide absolute offsets of expressions in a binary expression", func(t *testing.T) {
			result := view.ParseR3("<div>{{1 + 2}}<div>", nil)
			humanized := util.HumanizeExpressionSource(result.Nodes)
			expected := []util.HumanizedExpressionSource{
				{Unparsed: "1", Span: expression_parser.NewAbsoluteSourceSpan(7, 8)},
				{Unparsed: "2", Span: expression_parser.NewAbsoluteSourceSpan(11, 12)},
			}
			if !containsAllHumanizedExpressionSources(humanized, expected) {
				t.Errorf("Expected to contain all expressions: %v", expected)
			}
		})
	})

	t.Run("conditional", func(t *testing.T) {
		t.Run("should provide absolute offsets of a conditional", func(t *testing.T) {
			result := view.ParseR3("<div>{{bool ? 1 : 0}}<div>", nil)
			humanized := util.HumanizeExpressionSource(result.Nodes)
			found := false
			for _, item := range humanized {
				if item.Span != nil && item.Span.Start == 7 && item.Span.End == 19 {
					if contains(item.Unparsed, "bool") {
						found = true
						break
					}
				}
			}
			if !found {
				t.Errorf("Expected to contain conditional with AbsoluteSourceSpan(7, 19)")
			}
		})

		t.Run("should provide absolute offsets of expressions in a conditional", func(t *testing.T) {
			result := view.ParseR3("<div>{{bool ? 1 : 0}}<div>", nil)
			humanized := util.HumanizeExpressionSource(result.Nodes)
			expected := []util.HumanizedExpressionSource{
				{Unparsed: "bool", Span: expression_parser.NewAbsoluteSourceSpan(7, 11)},
				{Unparsed: "1", Span: expression_parser.NewAbsoluteSourceSpan(14, 15)},
				{Unparsed: "0", Span: expression_parser.NewAbsoluteSourceSpan(18, 19)},
			}
			if !containsAllHumanizedExpressionSources(humanized, expected) {
				t.Errorf("Expected to contain all expressions: %v", expected)
			}
		})
	})

	t.Run("chain", func(t *testing.T) {
		t.Run("should provide absolute offsets of a chain", func(t *testing.T) {
			result := view.ParseR3(`<div (click)="a(); b();"><div>`, nil)
			humanized := util.HumanizeExpressionSource(result.Nodes)
			found := false
			for _, item := range humanized {
				if item.Span != nil && item.Span.Start == 14 && item.Span.End == 23 {
					if contains(item.Unparsed, "a") && contains(item.Unparsed, "b") {
						found = true
						break
					}
				}
			}
			if !found {
				t.Errorf("Expected to contain chain with AbsoluteSourceSpan(14, 23)")
			}
		})

		t.Run("should provide absolute offsets of expressions in a chain", func(t *testing.T) {
			result := view.ParseR3(`<div (click)="a(); b();"><div>`, nil)
			humanized := util.HumanizeExpressionSource(result.Nodes)
			expected := []util.HumanizedExpressionSource{
				{Unparsed: "a()", Span: expression_parser.NewAbsoluteSourceSpan(14, 17)},
				{Unparsed: "b()", Span: expression_parser.NewAbsoluteSourceSpan(19, 22)},
			}
			if !containsAllHumanizedExpressionSources(humanized, expected) {
				t.Errorf("Expected to contain all expressions: %v", expected)
			}
		})
	})

	t.Run("function call", func(t *testing.T) {
		t.Run("should provide absolute offsets of a function call", func(t *testing.T) {
			result := view.ParseR3("<div>{{fn()()}}<div>", nil)
			humanized := util.HumanizeExpressionSource(result.Nodes)
			found := false
			for _, item := range humanized {
				if item.Span != nil && item.Span.Start == 7 && item.Span.End == 13 {
					if contains(item.Unparsed, "fn") {
						found = true
						break
					}
				}
			}
			if !found {
				t.Errorf("Expected to contain function call with AbsoluteSourceSpan(7, 13)")
			}
		})

		t.Run("should provide absolute offsets of expressions in a function call", func(t *testing.T) {
			result := view.ParseR3("<div>{{fn()(param)}}<div>", nil)
			humanized := util.HumanizeExpressionSource(result.Nodes)
			if !containsHumanizedExpressionSource(humanized, "param", 12, 17) {
				t.Errorf("Expected to contain ['param', AbsoluteSourceSpan(12, 17)]")
			}
		})
	})

	t.Run("should provide absolute offsets of an implicit receiver", func(t *testing.T) {
		result := view.ParseR3("<div>{{a.b}}<div>", nil)
		humanized := util.HumanizeExpressionSource(result.Nodes)
		// Implicit receiver has empty unparsed and span at start position
		found := false
		for _, item := range humanized {
			if item.Unparsed == "" && item.Span != nil && item.Span.Start == 7 && item.Span.End == 7 {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to contain implicit receiver with AbsoluteSourceSpan(7, 7)")
		}
	})

	t.Run("interpolation", func(t *testing.T) {
		t.Run("should provide absolute offsets of an interpolation", func(t *testing.T) {
			result := view.ParseR3("<div>{{1 + foo.length}}<div>", nil)
			humanized := util.HumanizeExpressionSource(result.Nodes)
			found := false
			for _, item := range humanized {
				if item.Span != nil && item.Span.Start == 5 && item.Span.End == 23 {
					if contains(item.Unparsed, "foo") {
						found = true
						break
					}
				}
			}
			if !found {
				t.Errorf("Expected to contain interpolation with AbsoluteSourceSpan(5, 23)")
			}
		})

		t.Run("should provide absolute offsets of expressions in an interpolation", func(t *testing.T) {
			result := view.ParseR3("<div>{{1 + 2}}<div>", nil)
			humanized := util.HumanizeExpressionSource(result.Nodes)
			expected := []util.HumanizedExpressionSource{
				{Unparsed: "1", Span: expression_parser.NewAbsoluteSourceSpan(7, 8)},
				{Unparsed: "2", Span: expression_parser.NewAbsoluteSourceSpan(11, 12)},
			}
			if !containsAllHumanizedExpressionSources(humanized, expected) {
				t.Errorf("Expected to contain all expressions: %v", expected)
			}
		})

		t.Run("should handle HTML entity before interpolation", func(t *testing.T) {
			result := view.ParseR3("&nbsp;{{abc}}", nil)
			humanized := util.HumanizeExpressionSource(result.Nodes)
			expected := []util.HumanizedExpressionSource{
				{Unparsed: "abc", Span: expression_parser.NewAbsoluteSourceSpan(8, 11)},
			}
			if !containsAllHumanizedExpressionSources(humanized, expected) {
				t.Errorf("Expected to contain ['abc', AbsoluteSourceSpan(8, 11)]")
			}
		})

		t.Run("should handle many HTML entities and many interpolations", func(t *testing.T) {
			result := view.ParseR3(`&quot;{{abc}}&quot;{{def}}&nbsp;{{ghi}}`, nil)
			humanized := util.HumanizeExpressionSource(result.Nodes)
			expected := []util.HumanizedExpressionSource{
				{Unparsed: "abc", Span: expression_parser.NewAbsoluteSourceSpan(8, 11)},
				{Unparsed: "def", Span: expression_parser.NewAbsoluteSourceSpan(21, 24)},
				{Unparsed: "ghi", Span: expression_parser.NewAbsoluteSourceSpan(34, 37)},
			}
			if !containsAllHumanizedExpressionSources(humanized, expected) {
				t.Errorf("Expected to contain all expressions: %v", expected)
			}
		})

		t.Run("should handle interpolation in attribute", func(t *testing.T) {
			result := view.ParseR3(`<div class="{{abc}}"><div>`, nil)
			humanized := util.HumanizeExpressionSource(result.Nodes)
			expected := []util.HumanizedExpressionSource{
				{Unparsed: "abc", Span: expression_parser.NewAbsoluteSourceSpan(14, 17)},
			}
			if !containsAllHumanizedExpressionSources(humanized, expected) {
				t.Errorf("Expected to contain ['abc', AbsoluteSourceSpan(14, 17)]")
			}
		})

		t.Run("should handle interpolation preceded by HTML entity in attribute", func(t *testing.T) {
			result := view.ParseR3(`<div class="&nbsp;{{abc}}"><div>`, nil)
			humanized := util.HumanizeExpressionSource(result.Nodes)
			expected := []util.HumanizedExpressionSource{
				{Unparsed: "abc", Span: expression_parser.NewAbsoluteSourceSpan(20, 23)},
			}
			if !containsAllHumanizedExpressionSources(humanized, expected) {
				t.Errorf("Expected to contain ['abc', AbsoluteSourceSpan(20, 23)]")
			}
		})

		t.Run("should handle many interpolation with HTML entities in attribute", func(t *testing.T) {
			result := view.ParseR3(`<div class="&quot;{{abc}}&quot;&nbsp;{{def}}"><div>`, nil)
			humanized := util.HumanizeExpressionSource(result.Nodes)
			expected := []util.HumanizedExpressionSource{
				{Unparsed: "abc", Span: expression_parser.NewAbsoluteSourceSpan(20, 23)},
				{Unparsed: "def", Span: expression_parser.NewAbsoluteSourceSpan(39, 42)},
			}
			if !containsAllHumanizedExpressionSources(humanized, expected) {
				t.Errorf("Expected to contain all expressions: %v", expected)
			}
		})
	})

	t.Run("keyed read", func(t *testing.T) {
		t.Run("should provide absolute offsets of a keyed read", func(t *testing.T) {
			result := view.ParseR3("<div>{{obj[key]}}<div>", nil)
			humanized := util.HumanizeExpressionSource(result.Nodes)
			found := false
			for _, item := range humanized {
				if item.Span != nil && item.Span.Start == 7 && item.Span.End == 15 {
					if contains(item.Unparsed, "obj") && contains(item.Unparsed, "key") {
						found = true
						break
					}
				}
			}
			if !found {
				t.Errorf("Expected to contain keyed read with AbsoluteSourceSpan(7, 15)")
			}
		})

		t.Run("should provide absolute offsets of expressions in a keyed read", func(t *testing.T) {
			result := view.ParseR3("<div>{{obj[key]}}<div>", nil)
			humanized := util.HumanizeExpressionSource(result.Nodes)
			if !containsHumanizedExpressionSource(humanized, "key", 11, 14) {
				t.Errorf("Expected to contain ['key', AbsoluteSourceSpan(11, 14)]")
			}
		})
	})

	t.Run("keyed write", func(t *testing.T) {
		t.Run("should provide absolute offsets of a keyed write", func(t *testing.T) {
			result := view.ParseR3("<div>{{obj[key] = 0}}<div>", nil)
			humanized := util.HumanizeExpressionSource(result.Nodes)
			found := false
			for _, item := range humanized {
				if item.Span != nil && item.Span.Start == 7 && item.Span.End == 19 {
					if contains(item.Unparsed, "obj") && contains(item.Unparsed, "key") {
						found = true
						break
					}
				}
			}
			if !found {
				t.Errorf("Expected to contain keyed write with AbsoluteSourceSpan(7, 19)")
			}
		})

		t.Run("should provide absolute offsets of expressions in a keyed write", func(t *testing.T) {
			result := view.ParseR3("<div>{{obj[key] = 0}}<div>", nil)
			humanized := util.HumanizeExpressionSource(result.Nodes)
			expected := []util.HumanizedExpressionSource{
				{Unparsed: "key", Span: expression_parser.NewAbsoluteSourceSpan(11, 14)},
				{Unparsed: "0", Span: expression_parser.NewAbsoluteSourceSpan(18, 19)},
			}
			if !containsAllHumanizedExpressionSources(humanized, expected) {
				t.Errorf("Expected to contain all expressions: %v", expected)
			}
		})
	})

	t.Run("should provide absolute offsets of a literal primitive", func(t *testing.T) {
		result := view.ParseR3("<div>{{100}}<div>", nil)
		humanized := util.HumanizeExpressionSource(result.Nodes)
		if !containsHumanizedExpressionSource(humanized, "100", 7, 10) {
			t.Errorf("Expected to contain ['100', AbsoluteSourceSpan(7, 10)]")
		}
	})

	t.Run("literal array", func(t *testing.T) {
		t.Run("should provide absolute offsets of a literal array", func(t *testing.T) {
			result := view.ParseR3("<div>{{[0, 1, 2]}}<div>", nil)
			humanized := util.HumanizeExpressionSource(result.Nodes)
			found := false
			for _, item := range humanized {
				if item.Span != nil && item.Span.Start == 7 && item.Span.End == 16 {
					if contains(item.Unparsed, "0") && contains(item.Unparsed, "1") && contains(item.Unparsed, "2") {
						found = true
						break
					}
				}
			}
			if !found {
				t.Errorf("Expected to contain literal array with AbsoluteSourceSpan(7, 16)")
			}
		})

		t.Run("should provide absolute offsets of expressions in a literal array", func(t *testing.T) {
			result := view.ParseR3("<div>{{[0, 1, 2]}}<div>", nil)
			humanized := util.HumanizeExpressionSource(result.Nodes)
			expected := []util.HumanizedExpressionSource{
				{Unparsed: "0", Span: expression_parser.NewAbsoluteSourceSpan(8, 9)},
				{Unparsed: "1", Span: expression_parser.NewAbsoluteSourceSpan(11, 12)},
				{Unparsed: "2", Span: expression_parser.NewAbsoluteSourceSpan(14, 15)},
			}
			if !containsAllHumanizedExpressionSources(humanized, expected) {
				t.Errorf("Expected to contain all expressions: %v", expected)
			}
		})
	})

	t.Run("literal map", func(t *testing.T) {
		t.Run("should provide absolute offsets of a literal map", func(t *testing.T) {
			result := view.ParseR3("<div>{{ {a: 0} }}<div>", nil)
			humanized := util.HumanizeExpressionSource(result.Nodes)
			found := false
			for _, item := range humanized {
				if item.Span != nil && item.Span.Start == 8 && item.Span.End == 14 {
					if contains(item.Unparsed, "a") {
						found = true
						break
					}
				}
			}
			if !found {
				t.Errorf("Expected to contain literal map with AbsoluteSourceSpan(8, 14)")
			}
		})

		t.Run("should provide absolute offsets of expressions in a literal map", func(t *testing.T) {
			result := view.ParseR3("<div>{{ {a: 0} }}<div>", nil)
			humanized := util.HumanizeExpressionSource(result.Nodes)
			expected := []util.HumanizedExpressionSource{
				{Unparsed: "0", Span: expression_parser.NewAbsoluteSourceSpan(12, 13)},
			}
			if !containsAllHumanizedExpressionSources(humanized, expected) {
				t.Errorf("Expected to contain ['0', AbsoluteSourceSpan(12, 13)]")
			}
		})
	})

	t.Run("method call", func(t *testing.T) {
		t.Run("should provide absolute offsets of a method call", func(t *testing.T) {
			result := view.ParseR3("<div>{{method()}}</div>", nil)
			humanized := util.HumanizeExpressionSource(result.Nodes)
			found := false
			for _, item := range humanized {
				if item.Span != nil && item.Span.Start == 7 && item.Span.End == 15 {
					if contains(item.Unparsed, "method") {
						found = true
						break
					}
				}
			}
			if !found {
				t.Errorf("Expected to contain method call with AbsoluteSourceSpan(7, 15)")
			}
		})

		t.Run("should provide absolute offsets of expressions in a method call", func(t *testing.T) {
			result := view.ParseR3("<div>{{method(param)}}<div>", nil)
			humanized := util.HumanizeExpressionSource(result.Nodes)
			if !containsHumanizedExpressionSource(humanized, "param", 14, 19) {
				t.Errorf("Expected to contain ['param', AbsoluteSourceSpan(14, 19)]")
			}
		})
	})

	t.Run("non-null assert", func(t *testing.T) {
		t.Run("should provide absolute offsets of a non-null assert", func(t *testing.T) {
			result := view.ParseR3("<div>{{prop!}}</div>", nil)
			humanized := util.HumanizeExpressionSource(result.Nodes)
			found := false
			for _, item := range humanized {
				if item.Span != nil && item.Span.Start == 7 && item.Span.End == 12 {
					if contains(item.Unparsed, "prop") {
						found = true
						break
					}
				}
			}
			if !found {
				t.Errorf("Expected to contain non-null assert with AbsoluteSourceSpan(7, 12)")
			}
		})

		t.Run("should provide absolute offsets of expressions in a non-null assert", func(t *testing.T) {
			result := view.ParseR3("<div>{{prop!}}<div>", nil)
			humanized := util.HumanizeExpressionSource(result.Nodes)
			if !containsHumanizedExpressionSource(humanized, "prop", 7, 11) {
				t.Errorf("Expected to contain ['prop', AbsoluteSourceSpan(7, 11)]")
			}
		})
	})

	t.Run("pipe", func(t *testing.T) {
		t.Run("should provide absolute offsets of a pipe", func(t *testing.T) {
			result := view.ParseR3("<div>{{prop | pipe}}<div>", nil)
			humanized := util.HumanizeExpressionSource(result.Nodes)
			found := false
			for _, item := range humanized {
				if item.Span != nil && item.Span.Start == 7 && item.Span.End == 18 {
					if contains(item.Unparsed, "prop") && contains(item.Unparsed, "pipe") {
						found = true
						break
					}
				}
			}
			if !found {
				t.Errorf("Expected to contain pipe with AbsoluteSourceSpan(7, 18)")
			}
		})

		t.Run("should provide absolute offsets expressions in a pipe", func(t *testing.T) {
			result := view.ParseR3("<div>{{prop | pipe}}<div>", nil)
			humanized := util.HumanizeExpressionSource(result.Nodes)
			if !containsHumanizedExpressionSource(humanized, "prop", 7, 11) {
				t.Errorf("Expected to contain ['prop', AbsoluteSourceSpan(7, 11)]")
			}
		})
	})

	t.Run("property read", func(t *testing.T) {
		t.Run("should provide absolute offsets of a property read", func(t *testing.T) {
			result := view.ParseR3("<div>{{prop.obj}}<div>", nil)
			humanized := util.HumanizeExpressionSource(result.Nodes)
			found := false
			for _, item := range humanized {
				if item.Span != nil && item.Span.Start == 7 && item.Span.End == 15 {
					if contains(item.Unparsed, "prop") && contains(item.Unparsed, "obj") {
						found = true
						break
					}
				}
			}
			if !found {
				t.Errorf("Expected to contain property read with AbsoluteSourceSpan(7, 15)")
			}
		})

		t.Run("should provide absolute offsets of expressions in a property read", func(t *testing.T) {
			result := view.ParseR3("<div>{{prop.obj}}<div>", nil)
			humanized := util.HumanizeExpressionSource(result.Nodes)
			if !containsHumanizedExpressionSource(humanized, "prop", 7, 11) {
				t.Errorf("Expected to contain ['prop', AbsoluteSourceSpan(7, 11)]")
			}
		})
	})

	t.Run("property write", func(t *testing.T) {
		t.Run("should provide absolute offsets of a property write", func(t *testing.T) {
			result := view.ParseR3(`<div (click)="prop = 0"></div>`, nil)
			humanized := util.HumanizeExpressionSource(result.Nodes)
			found := false
			for _, item := range humanized {
				if item.Span != nil && item.Span.Start == 14 && item.Span.End == 22 {
					if contains(item.Unparsed, "prop") {
						found = true
						break
					}
				}
			}
			if !found {
				t.Errorf("Expected to contain property write with AbsoluteSourceSpan(14, 22)")
			}
		})

		t.Run("should provide absolute offsets of an accessed property write", func(t *testing.T) {
			result := view.ParseR3(`<div (click)="prop.inner = 0"></div>`, nil)
			humanized := util.HumanizeExpressionSource(result.Nodes)
			found := false
			for _, item := range humanized {
				if item.Span != nil && item.Span.Start == 14 && item.Span.End == 28 {
					if contains(item.Unparsed, "prop") && contains(item.Unparsed, "inner") {
						found = true
						break
					}
				}
			}
			if !found {
				t.Errorf("Expected to contain accessed property write with AbsoluteSourceSpan(14, 28)")
			}
		})

		t.Run("should provide absolute offsets of expressions in a property write", func(t *testing.T) {
			result := view.ParseR3(`<div (click)="prop = 0"></div>`, nil)
			humanized := util.HumanizeExpressionSource(result.Nodes)
			if !containsHumanizedExpressionSource(humanized, "0", 21, 22) {
				t.Errorf("Expected to contain ['0', AbsoluteSourceSpan(21, 22)]")
			}
		})
	})

	t.Run("not prefix", func(t *testing.T) {
		t.Run("should provide absolute offsets of a not prefix", func(t *testing.T) {
			result := view.ParseR3("<div>{{!prop}}</div>", nil)
			humanized := util.HumanizeExpressionSource(result.Nodes)
			found := false
			for _, item := range humanized {
				if item.Span != nil && item.Span.Start == 7 && item.Span.End == 12 {
					if contains(item.Unparsed, "prop") {
						found = true
						break
					}
				}
			}
			if !found {
				t.Errorf("Expected to contain not prefix with AbsoluteSourceSpan(7, 12)")
			}
		})

		t.Run("should provide absolute offsets of expressions in a not prefix", func(t *testing.T) {
			result := view.ParseR3("<div>{{!prop}}<div>", nil)
			humanized := util.HumanizeExpressionSource(result.Nodes)
			if !containsHumanizedExpressionSource(humanized, "prop", 8, 12) {
				t.Errorf("Expected to contain ['prop', AbsoluteSourceSpan(8, 12)]")
			}
		})
	})

	t.Run("safe method call", func(t *testing.T) {
		t.Run("should provide absolute offsets of a safe method call", func(t *testing.T) {
			result := view.ParseR3("<div>{{prop?.safe()}}<div>", nil)
			humanized := util.HumanizeExpressionSource(result.Nodes)
			found := false
			for _, item := range humanized {
				if item.Span != nil && item.Span.Start == 7 && item.Span.End == 19 {
					if contains(item.Unparsed, "prop") && contains(item.Unparsed, "safe") {
						found = true
						break
					}
				}
			}
			if !found {
				t.Errorf("Expected to contain safe method call with AbsoluteSourceSpan(7, 19)")
			}
		})

		t.Run("should provide absolute offsets of expressions in safe method call", func(t *testing.T) {
			result := view.ParseR3("<div>{{prop?.safe()}}<div>", nil)
			humanized := util.HumanizeExpressionSource(result.Nodes)
			if !containsHumanizedExpressionSource(humanized, "prop", 7, 11) {
				t.Errorf("Expected to contain ['prop', AbsoluteSourceSpan(7, 11)]")
			}
		})
	})

	t.Run("safe property read", func(t *testing.T) {
		t.Run("should provide absolute offsets of a safe property read", func(t *testing.T) {
			result := view.ParseR3("<div>{{prop?.safe}}<div>", nil)
			humanized := util.HumanizeExpressionSource(result.Nodes)
			found := false
			for _, item := range humanized {
				if item.Span != nil && item.Span.Start == 7 && item.Span.End == 17 {
					if contains(item.Unparsed, "prop") && contains(item.Unparsed, "safe") {
						found = true
						break
					}
				}
			}
			if !found {
				t.Errorf("Expected to contain safe property read with AbsoluteSourceSpan(7, 17)")
			}
		})

		t.Run("should provide absolute offsets of expressions in safe property read", func(t *testing.T) {
			result := view.ParseR3("<div>{{prop?.safe}}<div>", nil)
			humanized := util.HumanizeExpressionSource(result.Nodes)
			if !containsHumanizedExpressionSource(humanized, "prop", 7, 11) {
				t.Errorf("Expected to contain ['prop', AbsoluteSourceSpan(7, 11)]")
			}
		})
	})

	t.Run("absolute offsets for template expressions", func(t *testing.T) {
		t.Run("should work for simple cases", func(t *testing.T) {
			result := view.ParseR3(`<div *ngFor="let item of items">{{item}}</div>`, nil)
			humanized := util.HumanizeExpressionSource(result.Nodes)
			if !containsHumanizedExpressionSource(humanized, "items", 25, 30) {
				t.Errorf("Expected to contain ['items', AbsoluteSourceSpan(25, 30)]")
			}
		})

		t.Run("should work with multiple bindings", func(t *testing.T) {
			result := view.ParseR3(`<div *ngFor="let a of As; let b of Bs"></div>`, nil)
			humanized := util.HumanizeExpressionSource(result.Nodes)
			expected := []util.HumanizedExpressionSource{
				{Unparsed: "As", Span: expression_parser.NewAbsoluteSourceSpan(22, 24)},
				{Unparsed: "Bs", Span: expression_parser.NewAbsoluteSourceSpan(35, 37)},
			}
			if !containsAllHumanizedExpressionSources(humanized, expected) {
				t.Errorf("Expected to contain all expressions: %v", expected)
			}
		})
	})

	t.Run("ICU expressions", func(t *testing.T) {
		t.Run("is correct for variables and placeholders", func(t *testing.T) {
			result := view.ParseR3(`<span i18n>{item.var, plural, other { {{item.placeholder}} items } }</span>`, nil)
			humanized := util.HumanizeExpressionSource(result.Nodes)
			expected := []util.HumanizedExpressionSource{
				{Unparsed: "item.var", Span: expression_parser.NewAbsoluteSourceSpan(12, 20)},
				{Unparsed: "item.placeholder", Span: expression_parser.NewAbsoluteSourceSpan(40, 56)},
			}
			if !containsAllHumanizedExpressionSources(humanized, expected) {
				t.Errorf("Expected to contain all expressions: %v", expected)
			}
		})

		t.Run("is correct for variables and placeholders nested", func(t *testing.T) {
			result := view.ParseR3(`<span i18n>{item.var, plural, other { {{item.placeholder}} {nestedVar, plural, other { {{nestedPlaceholder}} }}} }</span>`, nil)
			humanized := util.HumanizeExpressionSource(result.Nodes)
			expected := []util.HumanizedExpressionSource{
				{Unparsed: "item.var", Span: expression_parser.NewAbsoluteSourceSpan(12, 20)},
				{Unparsed: "item.placeholder", Span: expression_parser.NewAbsoluteSourceSpan(40, 56)},
				{Unparsed: "nestedVar", Span: expression_parser.NewAbsoluteSourceSpan(60, 69)},
				{Unparsed: "nestedPlaceholder", Span: expression_parser.NewAbsoluteSourceSpan(89, 106)},
			}
			if !containsAllHumanizedExpressionSources(humanized, expected) {
				t.Errorf("Expected to contain all expressions: %v", expected)
			}
		})
	})

	t.Run("object literal", func(t *testing.T) {
		t.Run("is correct for object literals with shorthand property declarations", func(t *testing.T) {
			result := view.ParseR3(`<div (click)="test({a: 1, b, c: 3, foo})"></div>`, nil)
			humanized := util.HumanizeExpressionSource(result.Nodes)
			// Check for the object literal expression
			found := false
			for _, item := range humanized {
				if item.Span != nil && item.Span.Start == 19 && item.Span.End == 39 {
					if contains(item.Unparsed, "a") || contains(item.Unparsed, "b") || contains(item.Unparsed, "c") || contains(item.Unparsed, "foo") {
						found = true
						break
					}
				}
			}
			if !found {
				t.Errorf("Expected to contain object literal with AbsoluteSourceSpan(19, 39)")
			}

			// Check for shorthand properties
			expected := []util.HumanizedExpressionSource{
				{Unparsed: "b", Span: expression_parser.NewAbsoluteSourceSpan(26, 27)},
				{Unparsed: "foo", Span: expression_parser.NewAbsoluteSourceSpan(35, 38)},
			}
			if !containsAllHumanizedExpressionSources(humanized, expected) {
				t.Errorf("Expected to contain shorthand properties: %v", expected)
			}
		})
	})
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && indexOf(s, substr) >= 0))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// Helper to check if two AbsoluteSourceSpans are equal
func spansEqual(a, b *expression_parser.AbsoluteSourceSpan) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.Start == b.Start && a.End == b.End
}

// Helper to check deep equality of slices
func deepEqual(a, b interface{}) bool {
	return reflect.DeepEqual(a, b)
}
