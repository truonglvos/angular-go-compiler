package expression_parser_test

import (
	"testing"

	"ngc-go/packages/compiler/src/expression_parser"
)

// Helper functions to match TypeScript test structure
func lex(text string) []*expression_parser.Token {
	lexer := expression_parser.NewLexer()
	return lexer.Tokenize(text)
}

func expectToken(t *testing.T, token *expression_parser.Token, index, end int) {
	if token == nil {
		t.Fatalf("Expected token, got nil")
	}
	if token.Index != index {
		t.Errorf("Expected token.Index = %d, got %d", index, token.Index)
	}
	if token.End != end {
		t.Errorf("Expected token.End = %d, got %d", end, token.End)
	}
}

func expectCharacterToken(t *testing.T, token *expression_parser.Token, index, end int, character string) {
	if len(character) != 1 {
		t.Fatalf("Character must be single character, got %q", character)
	}
	expectToken(t, token, index, end)
	code := int(character[0])
	if !token.IsCharacter(code) {
		t.Errorf("Expected character token with code %d, got type %v", code, token.Type)
	}
}

func expectOperatorToken(t *testing.T, token *expression_parser.Token, index, end int, operator string) {
	expectToken(t, token, index, end)
	if !token.IsOperator(operator) {
		t.Errorf("Expected operator token %q, got %q", operator, token.String())
	}
}

func expectNumberToken(t *testing.T, token *expression_parser.Token, index, end int, n float64) {
	expectToken(t, token, index, end)
	if !token.IsNumber() {
		t.Errorf("Expected number token, got type %v", token.Type)
	}
	if token.ToNumber() != n {
		t.Errorf("Expected number %f, got %f", n, token.ToNumber())
	}
}

func expectStringToken(t *testing.T, token *expression_parser.Token, index, end int, str string, kind expression_parser.StringTokenKind) {
	expectToken(t, token, index, end)
	if !token.IsString() {
		t.Errorf("Expected string token, got type %v", token.Type)
	}
	if token.Kind() != kind {
		t.Errorf("Expected string token kind %v, got %v", kind, token.Kind())
	}
	if token.String() != str {
		t.Errorf("Expected string %q, got %q", str, token.String())
	}
}

func expectIdentifierToken(t *testing.T, token *expression_parser.Token, index, end int, identifier string) {
	expectToken(t, token, index, end)
	if !token.IsIdentifier() {
		t.Errorf("Expected identifier token, got type %v", token.Type)
	}
	if token.String() != identifier {
		t.Errorf("Expected identifier %q, got %q", identifier, token.String())
	}
}

func expectPrivateIdentifierToken(t *testing.T, token *expression_parser.Token, index, end int, identifier string) {
	expectToken(t, token, index, end)
	if !token.IsPrivateIdentifier() {
		t.Errorf("Expected private identifier token, got type %v", token.Type)
	}
	if token.String() != identifier {
		t.Errorf("Expected private identifier %q, got %q", identifier, token.String())
	}
}

func expectKeywordToken(t *testing.T, token *expression_parser.Token, index, end int, keyword string) {
	expectToken(t, token, index, end)
	if !token.IsKeyword() {
		t.Errorf("Expected keyword token, got type %v", token.Type)
	}
	if token.String() != keyword {
		t.Errorf("Expected keyword %q, got %q", keyword, token.String())
	}
}

func expectErrorToken(t *testing.T, token *expression_parser.Token, index, end int, message string) {
	expectToken(t, token, index, end)
	if !token.IsError() {
		t.Errorf("Expected error token, got type %v", token.Type)
	}
	if token.String() != message {
		t.Errorf("Expected error message %q, got %q", message, token.String())
	}
}

func expectRegExpBodyToken(t *testing.T, token *expression_parser.Token, index, end int, str string) {
	expectToken(t, token, index, end)
	if !token.IsRegExpBody() {
		t.Errorf("Expected regexp body token, got type %v", token.Type)
	}
	if token.String() != str {
		t.Errorf("Expected regexp body %q, got %q", str, token.String())
	}
}

func expectRegExpFlagsToken(t *testing.T, token *expression_parser.Token, index, end int, str string) {
	expectToken(t, token, index, end)
	if !token.IsRegExpFlags() {
		t.Errorf("Expected regexp flags token, got type %v", token.Type)
	}
	if token.String() != str {
		t.Errorf("Expected regexp flags %q, got %q", str, token.String())
	}
}

func TestLexer_Token(t *testing.T) {
	t.Run("should tokenize a simple identifier", func(t *testing.T) {
		tokens := lex("j")
		if len(tokens) != 1 {
			t.Fatalf("Expected 1 token, got %d", len(tokens))
		}
		expectIdentifierToken(t, tokens[0], 0, 1, "j")
	})

	t.Run("should tokenize \"this\"", func(t *testing.T) {
		tokens := lex("this")
		if len(tokens) != 1 {
			t.Fatalf("Expected 1 token, got %d", len(tokens))
		}
		expectKeywordToken(t, tokens[0], 0, 4, "this")
	})

	t.Run("should tokenize a dotted identifier", func(t *testing.T) {
		tokens := lex("j.k")
		if len(tokens) != 3 {
			t.Fatalf("Expected 3 tokens, got %d", len(tokens))
		}
		expectIdentifierToken(t, tokens[0], 0, 1, "j")
		expectCharacterToken(t, tokens[1], 1, 2, ".")
		expectIdentifierToken(t, tokens[2], 2, 3, "k")
	})

	t.Run("should tokenize a private identifier", func(t *testing.T) {
		tokens := lex("#a")
		if len(tokens) != 1 {
			t.Fatalf("Expected 1 token, got %d", len(tokens))
		}
		expectPrivateIdentifierToken(t, tokens[0], 0, 2, "#a")
	})

	t.Run("should tokenize a property access with private identifier", func(t *testing.T) {
		tokens := lex("j.#k")
		if len(tokens) != 3 {
			t.Fatalf("Expected 3 tokens, got %d", len(tokens))
		}
		expectIdentifierToken(t, tokens[0], 0, 1, "j")
		expectCharacterToken(t, tokens[1], 1, 2, ".")
		expectPrivateIdentifierToken(t, tokens[2], 2, 4, "#k")
	})

	t.Run("should throw an invalid character error when a hash character is discovered but not indicating a private identifier", func(t *testing.T) {
		tokens := lex("#")
		if len(tokens) == 0 {
			t.Fatal("Expected at least one token")
		}
		expectErrorToken(t, tokens[0], 0, 1, "Lexer Error: Invalid character [#] at column 0 in expression [#]")

		tokens = lex("#0")
		if len(tokens) == 0 {
			t.Fatal("Expected at least one token")
		}
		expectErrorToken(t, tokens[0], 0, 1, "Lexer Error: Invalid character [#] at column 0 in expression [#0]")
	})

	t.Run("should tokenize an operator", func(t *testing.T) {
		tokens := lex("j-k")
		if len(tokens) != 3 {
			t.Fatalf("Expected 3 tokens, got %d", len(tokens))
		}
		expectOperatorToken(t, tokens[1], 1, 2, "-")
	})

	t.Run("should tokenize an indexed operator", func(t *testing.T) {
		tokens := lex("j[k]")
		if len(tokens) != 4 {
			t.Fatalf("Expected 4 tokens, got %d", len(tokens))
		}
		expectCharacterToken(t, tokens[1], 1, 2, "[")
		expectCharacterToken(t, tokens[3], 3, 4, "]")
	})

	t.Run("should tokenize a safe indexed operator", func(t *testing.T) {
		tokens := lex("j?.[k]")
		if len(tokens) != 5 {
			t.Fatalf("Expected 5 tokens, got %d", len(tokens))
		}
		expectOperatorToken(t, tokens[1], 1, 3, "?.")
		expectCharacterToken(t, tokens[2], 3, 4, "[")
		expectCharacterToken(t, tokens[4], 5, 6, "]")
	})

	t.Run("should tokenize numbers", func(t *testing.T) {
		tokens := lex("88")
		if len(tokens) != 1 {
			t.Fatalf("Expected 1 token, got %d", len(tokens))
		}
		expectNumberToken(t, tokens[0], 0, 2, 88)
	})

	t.Run("should tokenize numbers within index ops", func(t *testing.T) {
		tokens := lex("a[22]")
		if len(tokens) < 3 {
			t.Fatalf("Expected at least 3 tokens, got %d", len(tokens))
		}
		expectNumberToken(t, tokens[2], 2, 4, 22)
	})

	t.Run("should tokenize simple quoted strings", func(t *testing.T) {
		tokens := lex(`"a"`)
		if len(tokens) == 0 {
			t.Fatal("Expected at least one token")
		}
		expectStringToken(t, tokens[0], 0, 3, "a", expression_parser.StringTokenKindPlain)
	})

	t.Run("should tokenize quoted strings with escaped quotes", func(t *testing.T) {
		tokens := lex(`"a\""`)
		if len(tokens) == 0 {
			t.Fatal("Expected at least one token")
		}
		expectStringToken(t, tokens[0], 0, 5, `a"`, expression_parser.StringTokenKindPlain)
	})

	t.Run("should tokenize a string", func(t *testing.T) {
		tokens := lex(`j-a.bc[22]+1.3|f:'a\'c':"d\"e"`)
		if len(tokens) < 16 {
			t.Fatalf("Expected at least 16 tokens, got %d", len(tokens))
		}
		expectIdentifierToken(t, tokens[0], 0, 1, "j")
		expectOperatorToken(t, tokens[1], 1, 2, "-")
		expectIdentifierToken(t, tokens[2], 2, 3, "a")
		expectCharacterToken(t, tokens[3], 3, 4, ".")
		expectIdentifierToken(t, tokens[4], 4, 6, "bc")
		expectCharacterToken(t, tokens[5], 6, 7, "[")
		expectNumberToken(t, tokens[6], 7, 9, 22)
		expectCharacterToken(t, tokens[7], 9, 10, "]")
		expectOperatorToken(t, tokens[8], 10, 11, "+")
		expectNumberToken(t, tokens[9], 11, 14, 1.3)
		expectOperatorToken(t, tokens[10], 14, 15, "|")
		expectIdentifierToken(t, tokens[11], 15, 16, "f")
		expectCharacterToken(t, tokens[12], 16, 17, ":")
		expectStringToken(t, tokens[13], 17, 23, "a'c", expression_parser.StringTokenKindPlain)
		expectCharacterToken(t, tokens[14], 23, 24, ":")
		expectStringToken(t, tokens[15], 24, 30, `d"e`, expression_parser.StringTokenKindPlain)
	})

	t.Run("should tokenize undefined", func(t *testing.T) {
		tokens := lex("undefined")
		if len(tokens) == 0 {
			t.Fatal("Expected at least one token")
		}
		expectKeywordToken(t, tokens[0], 0, 9, "undefined")
		if !tokens[0].IsKeywordUndefined() {
			t.Error("Expected IsKeywordUndefined to be true")
		}
	})

	t.Run("should tokenize typeof", func(t *testing.T) {
		tokens := lex("typeof")
		if len(tokens) == 0 {
			t.Fatal("Expected at least one token")
		}
		expectKeywordToken(t, tokens[0], 0, 6, "typeof")
		if !tokens[0].IsKeywordTypeof() {
			t.Error("Expected IsKeywordTypeof to be true")
		}
	})

	t.Run("should tokenize void", func(t *testing.T) {
		tokens := lex("void")
		if len(tokens) == 0 {
			t.Fatal("Expected at least one token")
		}
		expectKeywordToken(t, tokens[0], 0, 4, "void")
		if !tokens[0].IsKeywordVoid() {
			t.Error("Expected IsKeywordVoid to be true")
		}
	})

	t.Run("should tokenize in keyword", func(t *testing.T) {
		tokens := lex("in")
		if len(tokens) == 0 {
			t.Fatal("Expected at least one token")
		}
		expectKeywordToken(t, tokens[0], 0, 2, "in")
		if !tokens[0].IsKeywordIn() {
			t.Error("Expected IsKeywordIn to be true")
		}
	})

	t.Run("should ignore whitespace", func(t *testing.T) {
		tokens := lex("a \t \n \r b")
		if len(tokens) < 2 {
			t.Fatalf("Expected at least 2 tokens, got %d", len(tokens))
		}
		expectIdentifierToken(t, tokens[0], 0, 1, "a")
		// Find the 'b' token (skip whitespace)
		var bToken *expression_parser.Token
		for _, tok := range tokens {
			if tok.IsIdentifier() && tok.String() == "b" {
				bToken = tok
				break
			}
		}
		if bToken == nil {
			t.Fatal("Could not find 'b' token")
		}
		// The index should account for whitespace
		if bToken.Index < 8 {
			t.Errorf("Expected 'b' token index >= 8, got %d", bToken.Index)
		}
	})

	t.Run("should tokenize quoted string", func(t *testing.T) {
		str := `['\'', "\""]`
		tokens := lex(str)
		if len(tokens) < 4 {
			t.Fatalf("Expected at least 4 tokens, got %d", len(tokens))
		}
		expectStringToken(t, tokens[1], 1, 5, "'", expression_parser.StringTokenKindPlain)
		expectStringToken(t, tokens[3], 7, 11, `"`, expression_parser.StringTokenKindPlain)
	})

	t.Run("should tokenize escaped quoted string", func(t *testing.T) {
		str := `"\"\n\f\r\t\v\u00A0"`
		tokens := lex(str)
		if len(tokens) == 0 {
			t.Fatal("Expected at least one token")
		}
		if tokens[0].String() != "\"\n\f\r\t\v\u00A0" {
			t.Errorf("Expected escaped string, got %q", tokens[0].String())
		}
	})

	t.Run("should tokenize unicode", func(t *testing.T) {
		tokens := lex(`"\u00A0"`)
		if len(tokens) == 0 {
			t.Fatal("Expected at least one token")
		}
		if tokens[0].String() != "\u00a0" {
			t.Errorf("Expected unicode string, got %q", tokens[0].String())
		}
	})

	t.Run("should tokenize relation", func(t *testing.T) {
		tokens := lex("! == != < > <= >= === !==")
		if len(tokens) < 9 {
			t.Fatalf("Expected at least 9 tokens, got %d", len(tokens))
		}
		expectOperatorToken(t, tokens[0], 0, 1, "!")
		expectOperatorToken(t, tokens[1], 2, 4, "==")
		expectOperatorToken(t, tokens[2], 5, 7, "!=")
		expectOperatorToken(t, tokens[3], 8, 9, "<")
		expectOperatorToken(t, tokens[4], 10, 11, ">")
		expectOperatorToken(t, tokens[5], 12, 14, "<=")
		expectOperatorToken(t, tokens[6], 15, 17, ">=")
		expectOperatorToken(t, tokens[7], 18, 21, "===")
		expectOperatorToken(t, tokens[8], 22, 25, "!==")
	})

	t.Run("should tokenize statements", func(t *testing.T) {
		tokens := lex("a;b;")
		if len(tokens) < 4 {
			t.Fatalf("Expected at least 4 tokens, got %d", len(tokens))
		}
		expectIdentifierToken(t, tokens[0], 0, 1, "a")
		expectCharacterToken(t, tokens[1], 1, 2, ";")
		expectIdentifierToken(t, tokens[2], 2, 3, "b")
		expectCharacterToken(t, tokens[3], 3, 4, ";")
	})

	t.Run("should tokenize function invocation", func(t *testing.T) {
		tokens := lex("a()")
		if len(tokens) < 3 {
			t.Fatalf("Expected at least 3 tokens, got %d", len(tokens))
		}
		expectIdentifierToken(t, tokens[0], 0, 1, "a")
		expectCharacterToken(t, tokens[1], 1, 2, "(")
		expectCharacterToken(t, tokens[2], 2, 3, ")")
	})

	t.Run("should tokenize simple method invocations", func(t *testing.T) {
		tokens := lex("a.method()")
		if len(tokens) < 5 {
			t.Fatalf("Expected at least 5 tokens, got %d", len(tokens))
		}
		expectIdentifierToken(t, tokens[2], 2, 8, "method")
	})

	t.Run("should tokenize method invocation", func(t *testing.T) {
		tokens := lex("a.b.c (d) - e.f()")
		if len(tokens) < 14 {
			t.Fatalf("Expected at least 14 tokens, got %d", len(tokens))
		}
		expectIdentifierToken(t, tokens[0], 0, 1, "a")
		expectCharacterToken(t, tokens[1], 1, 2, ".")
		expectIdentifierToken(t, tokens[2], 2, 3, "b")
		expectCharacterToken(t, tokens[3], 3, 4, ".")
		expectIdentifierToken(t, tokens[4], 4, 5, "c")
		expectCharacterToken(t, tokens[5], 6, 7, "(")
		expectIdentifierToken(t, tokens[6], 7, 8, "d")
		expectCharacterToken(t, tokens[7], 8, 9, ")")
		expectOperatorToken(t, tokens[8], 10, 11, "-")
		expectIdentifierToken(t, tokens[9], 12, 13, "e")
		expectCharacterToken(t, tokens[10], 13, 14, ".")
		expectIdentifierToken(t, tokens[11], 14, 15, "f")
		expectCharacterToken(t, tokens[12], 15, 16, "(")
		expectCharacterToken(t, tokens[13], 16, 17, ")")
	})

	t.Run("should tokenize safe function invocation", func(t *testing.T) {
		tokens := lex("a?.()")
		if len(tokens) < 4 {
			t.Fatalf("Expected at least 4 tokens, got %d", len(tokens))
		}
		expectIdentifierToken(t, tokens[0], 0, 1, "a")
		expectOperatorToken(t, tokens[1], 1, 3, "?.")
		expectCharacterToken(t, tokens[2], 3, 4, "(")
		expectCharacterToken(t, tokens[3], 4, 5, ")")
	})

	t.Run("should tokenize a safe method invocations", func(t *testing.T) {
		tokens := lex("a.method?.()")
		if len(tokens) < 6 {
			t.Fatalf("Expected at least 6 tokens, got %d", len(tokens))
		}
		expectIdentifierToken(t, tokens[0], 0, 1, "a")
		expectCharacterToken(t, tokens[1], 1, 2, ".")
		expectIdentifierToken(t, tokens[2], 2, 8, "method")
		expectOperatorToken(t, tokens[3], 8, 10, "?.")
		expectCharacterToken(t, tokens[4], 10, 11, "(")
		expectCharacterToken(t, tokens[5], 11, 12, ")")
	})

	t.Run("should tokenize number", func(t *testing.T) {
		tokens := lex("0.5")
		if len(tokens) == 0 {
			t.Fatal("Expected at least one token")
		}
		expectNumberToken(t, tokens[0], 0, 3, 0.5)
	})

	t.Run("should tokenize multiplication and exponentiation", func(t *testing.T) {
		tokens := lex("1 * 2 ** 3")
		if len(tokens) < 5 {
			t.Fatalf("Expected at least 5 tokens, got %d", len(tokens))
		}
		expectNumberToken(t, tokens[0], 0, 1, 1)
		expectOperatorToken(t, tokens[1], 2, 3, "*")
		expectNumberToken(t, tokens[2], 4, 5, 2)
		expectOperatorToken(t, tokens[3], 6, 8, "**")
		expectNumberToken(t, tokens[4], 9, 10, 3)
	})

	t.Run("should tokenize number with exponent", func(t *testing.T) {
		tokens := lex("0.5E-10")
		if len(tokens) == 0 {
			t.Fatal("Expected at least one token")
		}
		expectNumberToken(t, tokens[0], 0, 7, 0.5e-10)

		tokens = lex("0.5E+10")
		if len(tokens) == 0 {
			t.Fatal("Expected at least one token")
		}
		expectNumberToken(t, tokens[0], 0, 7, 0.5e10)
	})

	t.Run("should return exception for invalid exponent", func(t *testing.T) {
		tokens := lex("0.5E-")
		if len(tokens) == 0 {
			t.Fatal("Expected at least one token")
		}
		expectErrorToken(t, tokens[0], 4, 5, "Lexer Error: Invalid exponent at column 4 in expression [0.5E-]")

		tokens = lex("0.5E-A")
		if len(tokens) == 0 {
			t.Fatal("Expected at least one token")
		}
		expectErrorToken(t, tokens[0], 4, 5, "Lexer Error: Invalid exponent at column 4 in expression [0.5E-A]")
	})

	t.Run("should tokenize number starting with a dot", func(t *testing.T) {
		tokens := lex(".5")
		if len(tokens) == 0 {
			t.Fatal("Expected at least one token")
		}
		expectNumberToken(t, tokens[0], 0, 2, 0.5)
	})

	t.Run("should throw error on invalid unicode", func(t *testing.T) {
		tokens := lex(`'\u1''bla'`)
		if len(tokens) == 0 {
			t.Fatal("Expected at least one token")
		}
		expectErrorToken(t, tokens[0], 2, 2, "Lexer Error: Invalid unicode escape [\\u1''b] at column 2 in expression ['\\u1''bla']")
	})

	t.Run("should tokenize ?. as operator", func(t *testing.T) {
		tokens := lex("?.")
		if len(tokens) == 0 {
			t.Fatal("Expected at least one token")
		}
		expectOperatorToken(t, tokens[0], 0, 2, "?.")
	})

	t.Run("should tokenize ?? as operator", func(t *testing.T) {
		tokens := lex("??")
		if len(tokens) == 0 {
			t.Fatal("Expected at least one token")
		}
		expectOperatorToken(t, tokens[0], 0, 2, "??")
	})

	t.Run("should tokenize number with separator", func(t *testing.T) {
		expectNumberToken(t, lex("123_456")[0], 0, 7, 123_456)
		expectNumberToken(t, lex("1_000_000_000")[0], 0, 13, 1_000_000_000)
		expectNumberToken(t, lex("123_456.78")[0], 0, 10, 123_456.78)
		expectNumberToken(t, lex("123_456_789.123_456_789")[0], 0, 23, 123_456_789.123_456_789)
		expectNumberToken(t, lex("1_2_3_4")[0], 0, 7, 1_2_3_4)
		expectNumberToken(t, lex("1_2_3_4.5_6_7_8")[0], 0, 15, 1_2_3_4.5_6_7_8)
	})

	t.Run("should tokenize number starting with an underscore as an identifier", func(t *testing.T) {
		expectIdentifierToken(t, lex("_123")[0], 0, 4, "_123")
		expectIdentifierToken(t, lex("_123_")[0], 0, 5, "_123_")
		expectIdentifierToken(t, lex("_1_2_3_")[0], 0, 7, "_1_2_3_")
	})

	t.Run("should throw error for invalid number separators", func(t *testing.T) {
		expectErrorToken(t, lex("123_")[0], 3, 3, "Lexer Error: Invalid numeric separator at column 3 in expression [123_]")
		expectErrorToken(t, lex("12__3")[0], 2, 2, "Lexer Error: Invalid numeric separator at column 2 in expression [12__3]")
		expectErrorToken(t, lex("1_2_3_.456")[0], 5, 5, "Lexer Error: Invalid numeric separator at column 5 in expression [1_2_3_.456]")
		expectErrorToken(t, lex("1_2_3._456")[0], 6, 6, "Lexer Error: Invalid numeric separator at column 6 in expression [1_2_3._456]")
	})

	t.Run("should tokenize assignment operators", func(t *testing.T) {
		expectOperatorToken(t, lex("=")[0], 0, 1, "=")
		expectOperatorToken(t, lex("+=")[0], 0, 2, "+=")
		expectOperatorToken(t, lex("-=")[0], 0, 2, "-=")
		expectOperatorToken(t, lex("*=")[0], 0, 2, "*=")
		tokens := lex("a /= b")
		if len(tokens) < 3 {
			t.Fatalf("Expected at least 3 tokens, got %d", len(tokens))
		}
		expectOperatorToken(t, tokens[1], 2, 4, "/=")
		expectOperatorToken(t, lex("%=")[0], 0, 2, "%=")
		expectOperatorToken(t, lex("**=")[0], 0, 3, "**=")
		expectOperatorToken(t, lex("&&=")[0], 0, 3, "&&=")
		expectOperatorToken(t, lex("||=")[0], 0, 3, "||=")
		expectOperatorToken(t, lex("??=")[0], 0, 3, "??=")
	})

	t.Run("template literals", func(t *testing.T) {
		t.Run("should tokenize template literal with no interpolations", func(t *testing.T) {
			tokens := lex("`hello world`")
			if len(tokens) != 1 {
				t.Fatalf("Expected 1 token, got %d", len(tokens))
			}
			expectStringToken(t, tokens[0], 0, 13, "hello world", expression_parser.StringTokenKindTemplateLiteralEnd)
		})

		t.Run("should tokenize template literal containing strings", func(t *testing.T) {
			expectStringToken(t, lex("`a \"b\" c`")[0], 0, 9, `a "b" c`, expression_parser.StringTokenKindTemplateLiteralEnd)
			expectStringToken(t, lex("`a 'b' c`")[0], 0, 9, `a 'b' c`, expression_parser.StringTokenKindTemplateLiteralEnd)
			expectStringToken(t, lex("`a \\`b\\` c`")[0], 0, 11, "a `b` c", expression_parser.StringTokenKindTemplateLiteralEnd)
			expectStringToken(t, lex("`a \"'\\`b\\`'\" c`")[0], 0, 15, "a \"'`b`'\" c", expression_parser.StringTokenKindTemplateLiteralEnd)
		})

		t.Run("should tokenize unicode inside a template string", func(t *testing.T) {
			tokens := lex("`\\u00A0`")
			if len(tokens) == 0 {
				t.Fatal("Expected at least one token")
			}
			if tokens[0].String() != "\u00a0" {
				t.Errorf("Expected unicode string, got %q", tokens[0].String())
			}
		})

		t.Run("should tokenize template literal with an interpolation in the end", func(t *testing.T) {
			tokens := lex("`hello ${name}`")
			if len(tokens) != 5 {
				t.Fatalf("Expected 5 tokens, got %d", len(tokens))
			}
			expectStringToken(t, tokens[0], 0, 7, "hello ", expression_parser.StringTokenKindTemplateLiteralPart)
			expectOperatorToken(t, tokens[1], 7, 9, "${")
			expectIdentifierToken(t, tokens[2], 9, 13, "name")
			expectCharacterToken(t, tokens[3], 13, 14, "}")
			expectStringToken(t, tokens[4], 14, 15, "", expression_parser.StringTokenKindTemplateLiteralEnd)
		})

		t.Run("should tokenize template literal with an interpolation in the beginning", func(t *testing.T) {
			tokens := lex("`${name} Johnson`")
			if len(tokens) != 5 {
				t.Fatalf("Expected 5 tokens, got %d", len(tokens))
			}
			expectStringToken(t, tokens[0], 0, 1, "", expression_parser.StringTokenKindTemplateLiteralPart)
			expectOperatorToken(t, tokens[1], 1, 3, "${")
			expectIdentifierToken(t, tokens[2], 3, 7, "name")
			expectCharacterToken(t, tokens[3], 7, 8, "}")
			expectStringToken(t, tokens[4], 8, 17, " Johnson", expression_parser.StringTokenKindTemplateLiteralEnd)
		})

		t.Run("should tokenize template literal with an interpolation in the middle", func(t *testing.T) {
			tokens := lex("`foo${bar}baz`")
			if len(tokens) != 5 {
				t.Fatalf("Expected 5 tokens, got %d", len(tokens))
			}
			expectStringToken(t, tokens[0], 0, 4, "foo", expression_parser.StringTokenKindTemplateLiteralPart)
			expectOperatorToken(t, tokens[1], 4, 6, "${")
			expectIdentifierToken(t, tokens[2], 6, 9, "bar")
			expectCharacterToken(t, tokens[3], 9, 10, "}")
			expectStringToken(t, tokens[4], 10, 14, "baz", expression_parser.StringTokenKindTemplateLiteralEnd)
		})

		t.Run("should be able to use interpolation characters inside template string", func(t *testing.T) {
			expectStringToken(t, lex("`foo $`")[0], 0, 7, "foo $", expression_parser.StringTokenKindTemplateLiteralEnd)
			expectStringToken(t, lex("`foo }`")[0], 0, 7, "foo }", expression_parser.StringTokenKindTemplateLiteralEnd)
			expectStringToken(t, lex("`foo $ {}`")[0], 0, 10, "foo $ {}", expression_parser.StringTokenKindTemplateLiteralEnd)
			expectStringToken(t, lex("`foo \\${bar}`")[0], 0, 13, "foo ${bar}", expression_parser.StringTokenKindTemplateLiteralEnd)
		})

		t.Run("should tokenize template literal with several interpolations", func(t *testing.T) {
			tokens := lex("`${a} - ${b} - ${c}`")
			if len(tokens) != 13 {
				t.Fatalf("Expected 13 tokens, got %d", len(tokens))
			}
			expectStringToken(t, tokens[0], 0, 1, "", expression_parser.StringTokenKindTemplateLiteralPart)
			expectOperatorToken(t, tokens[1], 1, 3, "${")
			expectIdentifierToken(t, tokens[2], 3, 4, "a")
			expectCharacterToken(t, tokens[3], 4, 5, "}")
			expectStringToken(t, tokens[4], 5, 8, " - ", expression_parser.StringTokenKindTemplateLiteralPart)
			expectOperatorToken(t, tokens[5], 8, 10, "${")
			expectIdentifierToken(t, tokens[6], 10, 11, "b")
			expectCharacterToken(t, tokens[7], 11, 12, "}")
			expectStringToken(t, tokens[8], 12, 15, " - ", expression_parser.StringTokenKindTemplateLiteralPart)
			expectOperatorToken(t, tokens[9], 15, 17, "${")
			expectIdentifierToken(t, tokens[10], 17, 18, "c")
			expectCharacterToken(t, tokens[11], 18, 19, "}")
		})

		t.Run("should tokenize template literal with an object literal inside the interpolation", func(t *testing.T) {
			tokens := lex("`foo ${{$: true}} baz`")
			if len(tokens) != 9 {
				t.Fatalf("Expected 9 tokens, got %d", len(tokens))
			}
			expectStringToken(t, tokens[0], 0, 5, "foo ", expression_parser.StringTokenKindTemplateLiteralPart)
			expectOperatorToken(t, tokens[1], 5, 7, "${")
			expectCharacterToken(t, tokens[2], 7, 8, "{")
			expectIdentifierToken(t, tokens[3], 8, 9, "$")
			expectCharacterToken(t, tokens[4], 9, 10, ":")
			expectKeywordToken(t, tokens[5], 11, 15, "true")
			expectCharacterToken(t, tokens[6], 15, 16, "}")
			expectCharacterToken(t, tokens[7], 16, 17, "}")
			expectStringToken(t, tokens[8], 17, 22, " baz", expression_parser.StringTokenKindTemplateLiteralEnd)
		})

		t.Run("should tokenize template literal with template literals inside the interpolation", func(t *testing.T) {
			tokens := lex("`foo ${`hello ${`${a} - b`}`} baz`")
			if len(tokens) != 13 {
				t.Fatalf("Expected 13 tokens, got %d", len(tokens))
			}
			expectStringToken(t, tokens[0], 0, 5, "foo ", expression_parser.StringTokenKindTemplateLiteralPart)
			expectOperatorToken(t, tokens[1], 5, 7, "${")
			expectStringToken(t, tokens[2], 7, 14, "hello ", expression_parser.StringTokenKindTemplateLiteralPart)
			expectOperatorToken(t, tokens[3], 14, 16, "${")
			expectStringToken(t, tokens[4], 16, 17, "", expression_parser.StringTokenKindTemplateLiteralPart)
			expectOperatorToken(t, tokens[5], 17, 19, "${")
			expectIdentifierToken(t, tokens[6], 19, 20, "a")
			expectCharacterToken(t, tokens[7], 20, 21, "}")
			expectStringToken(t, tokens[8], 21, 26, " - b", expression_parser.StringTokenKindTemplateLiteralEnd)
			expectCharacterToken(t, tokens[9], 26, 27, "}")
			expectStringToken(t, tokens[10], 27, 28, "", expression_parser.StringTokenKindTemplateLiteralEnd)
			expectCharacterToken(t, tokens[11], 28, 29, "}")
			expectStringToken(t, tokens[12], 29, 34, " baz", expression_parser.StringTokenKindTemplateLiteralEnd)
		})

		t.Run("should tokenize two template literal right after each other", func(t *testing.T) {
			tokens := lex("`hello ${name}``see ${name} later`")
			if len(tokens) != 10 {
				t.Fatalf("Expected 10 tokens, got %d", len(tokens))
			}
			expectStringToken(t, tokens[0], 0, 7, "hello ", expression_parser.StringTokenKindTemplateLiteralPart)
			expectOperatorToken(t, tokens[1], 7, 9, "${")
			expectIdentifierToken(t, tokens[2], 9, 13, "name")
			expectCharacterToken(t, tokens[3], 13, 14, "}")
			expectStringToken(t, tokens[4], 14, 15, "", expression_parser.StringTokenKindTemplateLiteralEnd)
			expectStringToken(t, tokens[5], 15, 20, "see ", expression_parser.StringTokenKindTemplateLiteralPart)
			expectOperatorToken(t, tokens[6], 20, 22, "${")
			expectIdentifierToken(t, tokens[7], 22, 26, "name")
			expectCharacterToken(t, tokens[8], 26, 27, "}")
			expectStringToken(t, tokens[9], 27, 34, " later", expression_parser.StringTokenKindTemplateLiteralEnd)
		})

		t.Run("should tokenize a concatenated template literal", func(t *testing.T) {
			tokens := lex("`hello ${name}` + 123")
			if len(tokens) != 7 {
				t.Fatalf("Expected 7 tokens, got %d", len(tokens))
			}
			expectStringToken(t, tokens[0], 0, 7, "hello ", expression_parser.StringTokenKindTemplateLiteralPart)
			expectOperatorToken(t, tokens[1], 7, 9, "${")
			expectIdentifierToken(t, tokens[2], 9, 13, "name")
			expectCharacterToken(t, tokens[3], 13, 14, "}")
			expectStringToken(t, tokens[4], 14, 15, "", expression_parser.StringTokenKindTemplateLiteralEnd)
			expectOperatorToken(t, tokens[5], 16, 17, "+")
			expectNumberToken(t, tokens[6], 18, 21, 123)
		})

		t.Run("should tokenize a template literal with a pipe inside an interpolation", func(t *testing.T) {
			tokens := lex("`hello ${name | capitalize}!!!`")
			if len(tokens) != 7 {
				t.Fatalf("Expected 7 tokens, got %d", len(tokens))
			}
			expectStringToken(t, tokens[0], 0, 7, "hello ", expression_parser.StringTokenKindTemplateLiteralPart)
			expectOperatorToken(t, tokens[1], 7, 9, "${")
			expectIdentifierToken(t, tokens[2], 9, 13, "name")
			expectOperatorToken(t, tokens[3], 14, 15, "|")
			expectIdentifierToken(t, tokens[4], 16, 26, "capitalize")
			expectCharacterToken(t, tokens[5], 26, 27, "}")
			expectStringToken(t, tokens[6], 27, 31, "!!!", expression_parser.StringTokenKindTemplateLiteralEnd)
		})

		t.Run("should tokenize a template literal with a pipe inside a parenthesized interpolation", func(t *testing.T) {
			tokens := lex("`hello ${(name | capitalize)}!!!`")
			if len(tokens) != 9 {
				t.Fatalf("Expected 9 tokens, got %d", len(tokens))
			}
			expectStringToken(t, tokens[0], 0, 7, "hello ", expression_parser.StringTokenKindTemplateLiteralPart)
			expectOperatorToken(t, tokens[1], 7, 9, "${")
			expectCharacterToken(t, tokens[2], 9, 10, "(")
			expectIdentifierToken(t, tokens[3], 10, 14, "name")
			expectOperatorToken(t, tokens[4], 15, 16, "|")
			expectIdentifierToken(t, tokens[5], 17, 27, "capitalize")
			expectCharacterToken(t, tokens[6], 27, 28, ")")
			expectCharacterToken(t, tokens[7], 28, 29, "}")
			expectStringToken(t, tokens[8], 29, 33, "!!!", expression_parser.StringTokenKindTemplateLiteralEnd)
		})

		t.Run("should tokenize a template literal in an literal object value", func(t *testing.T) {
			tokens := lex("{foo: `${name}`}")
			if len(tokens) != 9 {
				t.Fatalf("Expected 9 tokens, got %d", len(tokens))
			}
			expectCharacterToken(t, tokens[0], 0, 1, "{")
			expectIdentifierToken(t, tokens[1], 1, 4, "foo")
			expectCharacterToken(t, tokens[2], 4, 5, ":")
			expectStringToken(t, tokens[3], 6, 7, "", expression_parser.StringTokenKindTemplateLiteralPart)
			expectOperatorToken(t, tokens[4], 7, 9, "${")
			expectIdentifierToken(t, tokens[5], 9, 13, "name")
			expectCharacterToken(t, tokens[6], 13, 14, "}")
			expectStringToken(t, tokens[7], 14, 15, "", expression_parser.StringTokenKindTemplateLiteralEnd)
			expectCharacterToken(t, tokens[8], 15, 16, "}")
		})

		t.Run("should produce an error if a template literal is not terminated", func(t *testing.T) {
			expectErrorToken(t, lex("`hello")[0], 6, 6, "Lexer Error: Unterminated template literal at column 6 in expression [`hello]")
		})

		t.Run("should produce an error for an unterminated template literal with an interpolation", func(t *testing.T) {
			tokens := lex("`hello ${name}!")
			if len(tokens) != 5 {
				t.Fatalf("Expected 5 tokens, got %d", len(tokens))
			}
			expectStringToken(t, tokens[0], 0, 7, "hello ", expression_parser.StringTokenKindTemplateLiteralPart)
			expectOperatorToken(t, tokens[1], 7, 9, "${")
			expectIdentifierToken(t, tokens[2], 9, 13, "name")
			expectCharacterToken(t, tokens[3], 13, 14, "}")
			expectErrorToken(t, tokens[4], 15, 15, "Lexer Error: Unterminated template literal at column 15 in expression [`hello ${name}!]")
		})

		t.Run("should produce an error for an unterminate template literal interpolation", func(t *testing.T) {
			tokens := lex("`hello ${name!`")
			if len(tokens) != 5 {
				t.Fatalf("Expected 5 tokens, got %d", len(tokens))
			}
			expectStringToken(t, tokens[0], 0, 7, "hello ", expression_parser.StringTokenKindTemplateLiteralPart)
			expectOperatorToken(t, tokens[1], 7, 9, "${")
			expectIdentifierToken(t, tokens[2], 9, 13, "name")
			expectOperatorToken(t, tokens[3], 13, 14, "!")
			expectErrorToken(t, tokens[4], 15, 15, "Lexer Error: Unterminated template literal at column 15 in expression [`hello ${name!`]")
		})

		t.Run("should tokenize tagged template literal with no interpolations", func(t *testing.T) {
			tokens := lex("tag`hello world`")
			if len(tokens) != 2 {
				t.Fatalf("Expected 2 tokens, got %d", len(tokens))
			}
			expectIdentifierToken(t, tokens[0], 0, 3, "tag")
			expectStringToken(t, tokens[1], 3, 16, "hello world", expression_parser.StringTokenKindTemplateLiteralEnd)
		})

		t.Run("should tokenize nested tagged template literals", func(t *testing.T) {
			tokens := lex("tag`hello ${tag`world`}`")
			if len(tokens) != 7 {
				t.Fatalf("Expected 7 tokens, got %d", len(tokens))
			}
			expectIdentifierToken(t, tokens[0], 0, 3, "tag")
			expectStringToken(t, tokens[1], 3, 10, "hello ", expression_parser.StringTokenKindTemplateLiteralPart)
			expectOperatorToken(t, tokens[2], 10, 12, "${")
			expectIdentifierToken(t, tokens[3], 12, 15, "tag")
			expectStringToken(t, tokens[4], 15, 22, "world", expression_parser.StringTokenKindTemplateLiteralEnd)
			expectCharacterToken(t, tokens[5], 22, 23, "}")
			expectStringToken(t, tokens[6], 23, 24, "", expression_parser.StringTokenKindTemplateLiteralEnd)
		})
	})

	t.Run("regular expressions", func(t *testing.T) {
		t.Run("should tokenize a simple regex", func(t *testing.T) {
			tokens := lex("/abc/")
			if len(tokens) != 1 {
				t.Fatalf("Expected 1 token, got %d", len(tokens))
			}
			expectRegExpBodyToken(t, tokens[0], 0, 5, "abc")
		})

		t.Run("should tokenize a regex with flags", func(t *testing.T) {
			tokens := lex("/abc/gim")
			if len(tokens) != 2 {
				t.Fatalf("Expected 2 tokens, got %d", len(tokens))
			}
			expectRegExpBodyToken(t, tokens[0], 0, 5, "abc")
			expectRegExpFlagsToken(t, tokens[1], 5, 8, "gim")
		})

		t.Run("should tokenize an identifier immediately after a regex", func(t *testing.T) {
			tokens := lex("/abc/ g")
			if len(tokens) != 2 {
				t.Fatalf("Expected 2 tokens, got %d", len(tokens))
			}
			expectRegExpBodyToken(t, tokens[0], 0, 5, "abc")
			expectIdentifierToken(t, tokens[1], 6, 7, "g")
		})

		t.Run("should tokenize a regex with an escaped slashes", func(t *testing.T) {
			tokens := lex("/^http:\\/\\/foo\\.bar/")
			if len(tokens) != 1 {
				t.Fatalf("Expected 1 token, got %d", len(tokens))
			}
			expectRegExpBodyToken(t, tokens[0], 0, 20, "^http:\\/\\/foo\\.bar")
		})

		t.Run("should tokenize a regex with un-escaped slashes in a character class", func(t *testing.T) {
			tokens := lex("/[a/]$/")
			if len(tokens) != 1 {
				t.Fatalf("Expected 1 token, got %d", len(tokens))
			}
			expectRegExpBodyToken(t, tokens[0], 0, 7, "[a/]$")
		})

		t.Run("should tokenize a regex with a backslash", func(t *testing.T) {
			tokens := lex("/a\\w+/")
			if len(tokens) != 1 {
				t.Fatalf("Expected 1 token, got %d", len(tokens))
			}
			expectRegExpBodyToken(t, tokens[0], 0, 6, "a\\w+")
		})

		t.Run("should tokenize a regex after an operator", func(t *testing.T) {
			tokens := lex("a = /b/")
			if len(tokens) != 3 {
				t.Fatalf("Expected 3 tokens, got %d", len(tokens))
			}
			expectIdentifierToken(t, tokens[0], 0, 1, "a")
			expectOperatorToken(t, tokens[1], 2, 3, "=")
			expectRegExpBodyToken(t, tokens[2], 4, 7, "b")
		})

		t.Run("should tokenize a regex inside parentheses", func(t *testing.T) {
			tokens := lex("log(/a/)")
			if len(tokens) != 4 {
				t.Fatalf("Expected 4 tokens, got %d", len(tokens))
			}
			expectIdentifierToken(t, tokens[0], 0, 3, "log")
			expectCharacterToken(t, tokens[1], 3, 4, "(")
			expectRegExpBodyToken(t, tokens[2], 4, 7, "a")
			expectCharacterToken(t, tokens[3], 7, 8, ")")
		})

		t.Run("should tokenize a regex at the beggining of an array", func(t *testing.T) {
			tokens := lex("[/a/]")
			if len(tokens) != 3 {
				t.Fatalf("Expected 3 tokens, got %d", len(tokens))
			}
			expectCharacterToken(t, tokens[0], 0, 1, "[")
			expectRegExpBodyToken(t, tokens[1], 1, 4, "a")
			expectCharacterToken(t, tokens[2], 4, 5, "]")
		})

		t.Run("should tokenize a regex in the middle of an array", func(t *testing.T) {
			tokens := lex("[1, /a/, 2]")
			if len(tokens) != 7 {
				t.Fatalf("Expected 7 tokens, got %d", len(tokens))
			}
			expectCharacterToken(t, tokens[0], 0, 1, "[")
			expectNumberToken(t, tokens[1], 1, 2, 1)
			expectCharacterToken(t, tokens[2], 2, 3, ",")
			expectRegExpBodyToken(t, tokens[3], 4, 7, "a")
			expectCharacterToken(t, tokens[4], 7, 8, ",")
			expectNumberToken(t, tokens[5], 9, 10, 2)
			expectCharacterToken(t, tokens[6], 10, 11, "]")
		})

		t.Run("should tokenize a regex inside an object literal", func(t *testing.T) {
			tokens := lex("{a: /b/}")
			if len(tokens) != 5 {
				t.Fatalf("Expected 5 tokens, got %d", len(tokens))
			}
			expectCharacterToken(t, tokens[0], 0, 1, "{")
			expectIdentifierToken(t, tokens[1], 1, 2, "a")
			expectCharacterToken(t, tokens[2], 2, 3, ":")
			expectRegExpBodyToken(t, tokens[3], 4, 7, "b")
			expectCharacterToken(t, tokens[4], 7, 8, "}")
		})

		t.Run("should tokenize a regex after a negation operator", func(t *testing.T) {
			tokens := lex("log(!/a/.test(\"1\"))")
			if len(tokens) != 10 {
				t.Fatalf("Expected 10 tokens, got %d", len(tokens))
			}
			expectIdentifierToken(t, tokens[0], 0, 3, "log")
			expectCharacterToken(t, tokens[1], 3, 4, "(")
			expectOperatorToken(t, tokens[2], 4, 5, "!")
			expectRegExpBodyToken(t, tokens[3], 5, 8, "a")
			expectCharacterToken(t, tokens[4], 8, 9, ".")
			expectIdentifierToken(t, tokens[5], 9, 13, "test")
			expectCharacterToken(t, tokens[6], 13, 14, "(")
			expectStringToken(t, tokens[7], 14, 17, "1", expression_parser.StringTokenKindPlain)
			expectCharacterToken(t, tokens[8], 17, 18, ")")
			expectCharacterToken(t, tokens[9], 18, 19, ")")
		})

		t.Run("should tokenize a regex after several negation operators", func(t *testing.T) {
			tokens := lex("log(!!!!!!/a/.test(\"1\"))")
			if len(tokens) != 15 {
				t.Fatalf("Expected 15 tokens, got %d", len(tokens))
			}
			expectIdentifierToken(t, tokens[0], 0, 3, "log")
			expectCharacterToken(t, tokens[1], 3, 4, "(")
			expectOperatorToken(t, tokens[2], 4, 5, "!")
			expectOperatorToken(t, tokens[3], 5, 6, "!")
			expectOperatorToken(t, tokens[4], 6, 7, "!")
			expectOperatorToken(t, tokens[5], 7, 8, "!")
			expectOperatorToken(t, tokens[6], 8, 9, "!")
			expectOperatorToken(t, tokens[7], 9, 10, "!")
			expectRegExpBodyToken(t, tokens[8], 10, 13, "a")
			expectCharacterToken(t, tokens[9], 13, 14, ".")
			expectIdentifierToken(t, tokens[10], 14, 18, "test")
			expectCharacterToken(t, tokens[11], 18, 19, "(")
			expectStringToken(t, tokens[12], 19, 22, "1", expression_parser.StringTokenKindPlain)
			expectCharacterToken(t, tokens[13], 22, 23, ")")
			expectCharacterToken(t, tokens[14], 23, 24, ")")
		})

		t.Run("should tokenize a method call on a regex", func(t *testing.T) {
			tokens := lex("/abc/.test(\"foo\")")
			if len(tokens) != 6 {
				t.Fatalf("Expected 6 tokens, got %d", len(tokens))
			}
			expectRegExpBodyToken(t, tokens[0], 0, 5, "abc")
			expectCharacterToken(t, tokens[1], 5, 6, ".")
			expectIdentifierToken(t, tokens[2], 6, 10, "test")
			expectCharacterToken(t, tokens[3], 10, 11, "(")
			expectStringToken(t, tokens[4], 11, 16, "foo", expression_parser.StringTokenKindPlain)
			expectCharacterToken(t, tokens[5], 16, 17, ")")
		})

		t.Run("should tokenize a method call with a regex parameter", func(t *testing.T) {
			tokens := lex("\"foo\".match(/abc/)")
			if len(tokens) != 6 {
				t.Fatalf("Expected 6 tokens, got %d", len(tokens))
			}
			expectStringToken(t, tokens[0], 0, 5, "foo", expression_parser.StringTokenKindPlain)
			expectCharacterToken(t, tokens[1], 5, 6, ".")
			expectIdentifierToken(t, tokens[2], 6, 11, "match")
			expectCharacterToken(t, tokens[3], 11, 12, "(")
			expectRegExpBodyToken(t, tokens[4], 12, 17, "abc")
			expectCharacterToken(t, tokens[5], 17, 18, ")")
		})

		t.Run("should not tokenize a regex preceded by a square bracket", func(t *testing.T) {
			tokens := lex("a[0] /= b")
			if len(tokens) != 6 {
				t.Fatalf("Expected 6 tokens, got %d", len(tokens))
			}
			expectIdentifierToken(t, tokens[0], 0, 1, "a")
			expectCharacterToken(t, tokens[1], 1, 2, "[")
			expectNumberToken(t, tokens[2], 2, 3, 0)
			expectCharacterToken(t, tokens[3], 3, 4, "]")
			expectOperatorToken(t, tokens[4], 5, 7, "/=")
			expectIdentifierToken(t, tokens[5], 8, 9, "b")
		})

		t.Run("should not tokenize a regex preceded by an identifier", func(t *testing.T) {
			tokens := lex("a / b")
			if len(tokens) != 3 {
				t.Fatalf("Expected 3 tokens, got %d", len(tokens))
			}
			expectIdentifierToken(t, tokens[0], 0, 1, "a")
			expectOperatorToken(t, tokens[1], 2, 3, "/")
			expectIdentifierToken(t, tokens[2], 4, 5, "b")
		})

		t.Run("should not tokenize a regex preceded by a number", func(t *testing.T) {
			tokens := lex("1 / b")
			if len(tokens) != 3 {
				t.Fatalf("Expected 3 tokens, got %d", len(tokens))
			}
			expectNumberToken(t, tokens[0], 0, 1, 1)
			expectOperatorToken(t, tokens[1], 2, 3, "/")
			expectIdentifierToken(t, tokens[2], 4, 5, "b")
		})

		t.Run("should not tokenize a regex that is preceded by a string", func(t *testing.T) {
			tokens := lex("\"a\" / b")
			if len(tokens) != 3 {
				t.Fatalf("Expected 3 tokens, got %d", len(tokens))
			}
			expectStringToken(t, tokens[0], 0, 3, "a", expression_parser.StringTokenKindPlain)
			expectOperatorToken(t, tokens[1], 4, 5, "/")
			expectIdentifierToken(t, tokens[2], 6, 7, "b")
		})

		t.Run("should not tokenize a regex preceded by a closing parenthesis", func(t *testing.T) {
			tokens := lex("(a) / b")
			if len(tokens) != 5 {
				t.Fatalf("Expected 5 tokens, got %d", len(tokens))
			}
			expectCharacterToken(t, tokens[0], 0, 1, "(")
			expectIdentifierToken(t, tokens[1], 1, 2, "a")
			expectCharacterToken(t, tokens[2], 2, 3, ")")
			expectOperatorToken(t, tokens[3], 4, 5, "/")
			expectIdentifierToken(t, tokens[4], 6, 7, "b")
		})

		t.Run("should not tokenize a regex that is preceded by a keyword", func(t *testing.T) {
			tokens := lex("this / b")
			if len(tokens) != 3 {
				t.Fatalf("Expected 3 tokens, got %d", len(tokens))
			}
			expectKeywordToken(t, tokens[0], 0, 4, "this")
			expectOperatorToken(t, tokens[1], 5, 6, "/")
			expectIdentifierToken(t, tokens[2], 7, 8, "b")
		})

		t.Run("should not tokenize a regex preceded by a non-null assertion on an identifier", func(t *testing.T) {
			tokens := lex("foo! / 2")
			if len(tokens) != 4 {
				t.Fatalf("Expected 4 tokens, got %d", len(tokens))
			}
			expectIdentifierToken(t, tokens[0], 0, 3, "foo")
			expectOperatorToken(t, tokens[1], 3, 4, "!")
			expectOperatorToken(t, tokens[2], 5, 6, "/")
			expectNumberToken(t, tokens[3], 7, 8, 2)
		})

		t.Run("should not tokenize a regex preceded by a non-null assertion on a function call", func(t *testing.T) {
			tokens := lex("foo()! / 2")
			if len(tokens) != 6 {
				t.Fatalf("Expected 6 tokens, got %d", len(tokens))
			}
			expectIdentifierToken(t, tokens[0], 0, 3, "foo")
			expectCharacterToken(t, tokens[1], 3, 4, "(")
			expectCharacterToken(t, tokens[2], 4, 5, ")")
			expectOperatorToken(t, tokens[3], 5, 6, "!")
			expectOperatorToken(t, tokens[4], 7, 8, "/")
			expectNumberToken(t, tokens[5], 9, 10, 2)
		})

		t.Run("should not tokenize a regex preceded by a non-null assertion on an array", func(t *testing.T) {
			tokens := lex("[1]! / 2")
			if len(tokens) != 6 {
				t.Fatalf("Expected 6 tokens, got %d", len(tokens))
			}
			expectCharacterToken(t, tokens[0], 0, 1, "[")
			expectNumberToken(t, tokens[1], 1, 2, 1)
			expectCharacterToken(t, tokens[2], 2, 3, "]")
			expectOperatorToken(t, tokens[3], 3, 4, "!")
			expectOperatorToken(t, tokens[4], 5, 6, "/")
			expectNumberToken(t, tokens[5], 7, 8, 2)
		})

		t.Run("should not tokenize consecutive regexes", func(t *testing.T) {
			tokens := lex("/ 1 / 2 / 3 / 4")
			if len(tokens) != 6 {
				t.Fatalf("Expected 6 tokens, got %d", len(tokens))
			}
			expectRegExpBodyToken(t, tokens[0], 0, 5, " 1 ")
			expectNumberToken(t, tokens[1], 6, 7, 2)
			expectOperatorToken(t, tokens[2], 8, 9, "/")
			expectNumberToken(t, tokens[3], 10, 11, 3)
			expectOperatorToken(t, tokens[4], 12, 13, "/")
			expectNumberToken(t, tokens[5], 14, 15, 4)
		})

		t.Run("should not tokenize regex-like characters inside of a pipe", func(t *testing.T) {
			tokens := lex("foo / 1000 | date: 'M/d/yy'")
			if len(tokens) != 7 {
				t.Fatalf("Expected 7 tokens, got %d", len(tokens))
			}
			expectIdentifierToken(t, tokens[0], 0, 3, "foo")
			expectOperatorToken(t, tokens[1], 4, 5, "/")
			expectNumberToken(t, tokens[2], 6, 10, 1000)
			expectOperatorToken(t, tokens[3], 11, 12, "|")
			expectIdentifierToken(t, tokens[4], 13, 17, "date")
			expectCharacterToken(t, tokens[5], 17, 18, ":")
			expectStringToken(t, tokens[6], 19, 27, "M/d/yy", expression_parser.StringTokenKindPlain)
		})

		t.Run("should produce an error for an unterminated regex", func(t *testing.T) {
			expectErrorToken(t, lex("/a")[0], 2, 2, "Lexer Error: Unterminated regular expression at column 2 in expression [/a]")
		})
	})
}
