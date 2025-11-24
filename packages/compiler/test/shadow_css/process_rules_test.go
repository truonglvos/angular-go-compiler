package shadow_css_test

import (
	"ngc-go/packages/compiler/src/css"
	"testing"
)

func TestProcessRules(t *testing.T) {
	t.Run("parse rules", func(t *testing.T) {
		captureRules := func(input string) []*css.CssRule {
			result := []*css.CssRule{}
			css.ProcessRules(input, func(rule *css.CssRule) *css.CssRule {
				result = append(result, rule)
				return rule
			})
			return result
		}

		t.Run("should work with empty css", func(t *testing.T) {
			rules := captureRules("")
			if len(rules) != 0 {
				t.Errorf("Expected empty rules, got %d", len(rules))
			}
		})

		t.Run("should capture a rule without body", func(t *testing.T) {
			rules := captureRules("a;")
			if len(rules) != 1 {
				t.Fatalf("Expected 1 rule, got %d", len(rules))
			}
			if rules[0].Selector != "a" || rules[0].Content != "" {
				t.Errorf("Expected selector 'a' and empty content, got selector %q and content %q", rules[0].Selector, rules[0].Content)
			}
		})

		t.Run("should capture css rules with body", func(t *testing.T) {
			rules := captureRules("a {b}")
			if len(rules) != 1 {
				t.Fatalf("Expected 1 rule, got %d", len(rules))
			}
			if rules[0].Selector != "a" || rules[0].Content != "b" {
				t.Errorf("Expected selector 'a' and content 'b', got selector %q and content %q", rules[0].Selector, rules[0].Content)
			}
		})

		t.Run("should capture css rules with nested rules", func(t *testing.T) {
			rules := captureRules("a {b {c}} d {e}")
			if len(rules) != 2 {
				t.Fatalf("Expected 2 rules, got %d", len(rules))
			}
			if rules[0].Selector != "a" || rules[0].Content != "b {c}" {
				t.Errorf("Expected selector 'a' and content 'b {c}', got selector %q and content %q", rules[0].Selector, rules[0].Content)
			}
			if rules[1].Selector != "d" || rules[1].Content != "e" {
				t.Errorf("Expected selector 'd' and content 'e', got selector %q and content %q", rules[1].Selector, rules[1].Content)
			}
		})

		t.Run("should capture multiple rules where some have no body", func(t *testing.T) {
			rules := captureRules("@import a ; b {c}")
			if len(rules) != 2 {
				t.Fatalf("Expected 2 rules, got %d", len(rules))
			}
			if rules[0].Selector != "@import a" || rules[0].Content != "" {
				t.Errorf("Expected selector '@import a' and empty content, got selector %q and content %q", rules[0].Selector, rules[0].Content)
			}
			if rules[1].Selector != "b" || rules[1].Content != "c" {
				t.Errorf("Expected selector 'b' and content 'c', got selector %q and content %q", rules[1].Selector, rules[1].Content)
			}
		})
	})

	t.Run("modify rules", func(t *testing.T) {
		t.Run("should allow to change the selector while preserving whitespaces", func(t *testing.T) {
			result := css.ProcessRules(
				"@import a; b {c {d}} e {f}",
				func(rule *css.CssRule) *css.CssRule {
					return css.NewCssRule(rule.Selector+"2", rule.Content)
				},
			)
			expected := "@import a2; b2 {c {d}} e2 {f}"
			if result != expected {
				t.Errorf("Expected %q, got %q", expected, result)
			}
		})

		t.Run("should allow to change the content", func(t *testing.T) {
			result := css.ProcessRules(
				"a {b}",
				func(rule *css.CssRule) *css.CssRule {
					return css.NewCssRule(rule.Selector, rule.Content+"2")
				},
			)
			expected := "a {b2}"
			if result != expected {
				t.Errorf("Expected %q, got %q", expected, result)
			}
		})
	})
}

