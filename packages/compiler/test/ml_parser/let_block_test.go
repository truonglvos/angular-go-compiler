package ml_parser_test

import (
	"testing"

	"ngc-go/packages/compiler/src/ml_parser"
)

func TestLetAndBlockTokenization(t *testing.T) {
	t.Run("@let declaration", func(t *testing.T) {
		tokenizeLet := true
		result := ml_parser.Tokenize("@let foo = 'bar';", "TestComp", nil, &ml_parser.TokenizeOptions{TokenizeLet: &tokenizeLet})
		t.Logf("Generated %d tokens, %d errors:", len(result.Tokens), len(result.Errors))
		for _, err := range result.Errors {
			t.Errorf("  Error: %v", err)
		}
		for i, tok := range result.Tokens {
			t.Logf("  Token %d: Type=%d, Parts=%v", i, tok.Type(), tok.Parts())
		}

		if len(result.Errors) > 0 {
			t.Errorf("Expected no errors for @let declaration")
		}
		// Should have: LET_START, LET_VALUE, LET_END, EOF
		if len(result.Tokens) < 4 {
			t.Errorf("Expected at least 4 tokens, got %d", len(result.Tokens))
		}
	})

	t.Run("@if block", func(t *testing.T) {
		tokenizeBlocks := true
		result := ml_parser.Tokenize("@if (condition) { content }", "TestComp", nil, &ml_parser.TokenizeOptions{TokenizeBlocks: &tokenizeBlocks})
		t.Logf("Generated %d tokens, %d errors:", len(result.Tokens), len(result.Errors))
		for _, err := range result.Errors {
			t.Errorf("  Error: %v", err)
		}
		for i, tok := range result.Tokens {
			t.Logf("  Token %d: Type=%d, Parts=%v", i, tok.Type(), tok.Parts())
		}

		if len(result.Errors) > 0 {
			t.Errorf("Expected no errors for @if block")
		}
		// Should have: BLOCK_OPEN_START, BLOCK_PARAMETER, BLOCK_OPEN_END, TEXT, BLOCK_CLOSE would come later
		if len(result.Tokens) < 4 {
			t.Errorf("Expected at least 4 tokens, got %d", len(result.Tokens))
		}
	})
}
