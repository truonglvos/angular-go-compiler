package ml_parser_test

import (
	"testing"

	"ngc-go/packages/compiler/src/ml_parser"
)

func TestEntityDecoding(t *testing.T) {
	parser := ml_parser.NewHtmlParser()

	t.Run("simple amp entity", func(t *testing.T) {
		result := parser.Parse("<div>&amp;</div>", "TestComp", nil)
		if len(result.Errors) > 0 {
			t.Errorf("Unexpected errors: %v", result.Errors)
		}
		t.Logf("Parsed successfully: %d nodes", len(result.RootNodes))
	})

	t.Run("numeric hex entity", func(t *testing.T) {
		// First tokenize to see what tokens are generated
		tokenizeResult := ml_parser.Tokenize("<div>&#x1F6C8;</div>", "TestComp", nil, nil)
		t.Logf("Generated %d tokens, %d errors:", len(tokenizeResult.Tokens), len(tokenizeResult.Errors))
		for _, err := range tokenizeResult.Errors {
			t.Logf("  Error: %v", err)
		}
		for i, tok := range tokenizeResult.Tokens {
			t.Logf("  Token %d: Type=%d, Parts=%v", i, tok.Type(), tok.Parts())
			if i > 50 {
				t.Fatal("Too many tokens - infinite loop detected!")
			}
		}
		// Don't continue to parser - just test tokenizer for now

		result := parser.Parse("<div>&#x1F6C8;</div>", "TestComp", nil)
		if len(result.Errors) > 0 {
			t.Errorf("Unexpected errors: %v", result.Errors)
		}
		t.Logf("Parsed successfully: %d nodes", len(result.RootNodes))
	})
}
