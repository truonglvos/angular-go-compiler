package expression_parser_test

import (
	"testing"

	"ngc-go/packages/compiler/src/expression_parser"
)

func parseBinding(expression string, supportsDirectPipeReferences ...bool) *expression_parser.ASTWithSource {
	supportsDirect := false
	if len(supportsDirectPipeReferences) > 0 {
		supportsDirect = supportsDirectPipeReferences[0]
	}
	p := expression_parser.NewParser(expression_parser.NewLexer(), supportsDirect)
	return p.ParseBinding(expression, getFakeSpan(""), 0)
}

func checkAction(exp string, expected ...string) func(*testing.T) {
	return func(t *testing.T) {
		ast := parseAction(exp)
		expectedStr := exp
		if len(expected) > 0 {
			expectedStr = expected[0]
		}
		result := Unparse(ast.AST)
		if result != expectedStr {
			t.Errorf("Expected %q, got %q", expectedStr, result)
		}
		// TODO: Add validate check
	}
}

func checkBinding(exp string, expected ...string) func(*testing.T) {
	return func(t *testing.T) {
		ast := parseBinding(exp)
		expectedStr := exp
		if len(expected) > 0 {
			expectedStr = expected[0]
		}
		result := Unparse(ast.AST)
		if result != expectedStr {
			t.Errorf("Expected %q, got %q", expectedStr, result)
		}
		// TODO: Add validate check
	}
}

func expectActionError(text string, message string, errorCount ...int) func(*testing.T) {
	return func(t *testing.T) {
		ast := parseAction(text)
		errors := ast.Errors
		expectedCount := -1
		if len(errorCount) > 0 {
			expectedCount = errorCount[0]
		}
		if expectedCount >= 0 {
			if len(errors) != expectedCount {
				t.Errorf("Expected %d errors, got %d", expectedCount, len(errors))
				return
			}
		} else {
			if len(errors) == 0 {
				t.Errorf("Expected at least one error containing %q, but got no errors", message)
				return
			}
		}
		found := false
		for _, err := range errors {
			if contains(err.Msg, message) {
				found = true
				break
			}
		}
		if !found {
			errMsgs := ""
			for _, err := range errors {
				errMsgs += err.Msg + "\n"
			}
			t.Errorf("Expected an error containing %q, but got:\n%s", message, errMsgs)
		}
	}
}

func expectBindingError(text string, message string) func(*testing.T) {
	return func(t *testing.T) {
		ast := parseBinding(text)
		expectError(ast, message)(t)
	}
}

func expectError(ast *expression_parser.ASTWithSource, message string, errorCount ...int) func(*testing.T) {
	return func(t *testing.T) {
		errors := ast.Errors
		expectedCount := -1
		if len(errorCount) > 0 {
			expectedCount = errorCount[0]
		}
		if expectedCount >= 0 {
			if len(errors) != expectedCount {
				t.Errorf("Expected %d errors, got %d", expectedCount, len(errors))
				return
			}
		} else {
			if len(errors) == 0 {
				t.Errorf("Expected at least one error containing %q, but got no errors", message)
				return
			}
		}
		found := false
		for _, err := range errors {
			if contains(err.Msg, message) {
				found = true
				break
			}
		}
		if !found {
			errMsgs := ""
			for _, err := range errors {
				errMsgs += err.Msg + "\n"
			}
			t.Errorf("Expected an error containing %q, but got:\n%s", message, errMsgs)
		}
	}
}

func checkActionWithError(text string, expected string, error string) func(*testing.T) {
	return func(t *testing.T) {
		checkAction(text, expected)(t)
		expectActionError(text, error)(t)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			indexOf(s, substr) >= 0)))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func TestParser(t *testing.T) {
	t.Run("parseAction", func(t *testing.T) {
		t.Run("should parse numbers", checkAction("1"))

		t.Run("should parse strings", func(t *testing.T) {
			checkAction("'1'", `"1"`)(t)
			checkAction(`"1"`)(t)
		})

		t.Run("should parse null", checkAction("null"))

		t.Run("should parse undefined", checkAction("undefined"))

		t.Run("should parse unary - and + expressions", func(t *testing.T) {
			checkAction("-1", "-1")(t)
			checkAction("+1", "+1")(t)
			checkAction(`-'1'`, `-"1"`)(t)
			checkAction(`+'1'`, `+"1"`)(t)
		})

		t.Run("should parse unary ! expressions", func(t *testing.T) {
			checkAction("!true")(t)
			checkAction("!!true")(t)
			checkAction("!!!true")(t)
		})

		t.Run("should parse postfix ! expression", func(t *testing.T) {
			checkAction("true!")(t)
			checkAction("a!.b")(t)
			checkAction("a!!!!.b")(t)
			checkAction("a!()")(t)
			checkAction("a.b!()")(t)
		})

		t.Run("should parse exponentiation expressions", func(t *testing.T) {
			checkAction("1*2**3", "1 * 2 ** 3")(t)
		})

		t.Run("should parse multiplicative expressions", func(t *testing.T) {
			checkAction("3*4/2%5", "3 * 4 / 2 % 5")(t)
		})

		t.Run("should parse additive expressions", checkAction("3 + 6 - 2"))

		t.Run("should parse relational expressions", func(t *testing.T) {
			checkAction("2 < 3")(t)
			checkAction("2 > 3")(t)
			checkAction("2 <= 2")(t)
			checkAction("2 >= 2")(t)
		})

		t.Run("should parse equality expressions", func(t *testing.T) {
			checkAction("2 == 3")(t)
			checkAction("2 != 3")(t)
		})

		t.Run("should parse strict equality expressions", func(t *testing.T) {
			checkAction("2 === 3")(t)
			checkAction("2 !== 3")(t)
		})

		t.Run("should parse expressions", func(t *testing.T) {
			checkAction("true && true")(t)
			checkAction("true || false")(t)
			checkAction("null ?? 0")(t)
			checkAction("null ?? undefined ?? 0")(t)
		})

		t.Run("should parse typeof expression", func(t *testing.T) {
			checkAction(`typeof {} === "object"`)(t)
			checkAction(`(!(typeof {} === "number"))`)(t)
		})

		t.Run("should parse void expression", func(t *testing.T) {
			checkAction(`void 0`)(t)
			checkAction(`(!(void 0))`)(t)
		})

		t.Run("should parse grouped expressions", checkAction("(1 + 2) * 3"))

		t.Run("should parse in expressions", func(t *testing.T) {
			checkAction(`'key' in obj`, `"key" in obj`)(t)
			checkAction(`('key' in obj) && true`, `("key" in obj) && true`)(t)
		})

		t.Run("should ignore comments in expressions", func(t *testing.T) {
			checkAction("a //comment", "a")(t)
		})

		t.Run("should retain // in string literals", func(t *testing.T) {
			checkAction(`"http://www.google.com"`, `"http://www.google.com"`)(t)
		})

		t.Run("should parse an empty string", checkAction(""))

		t.Run("should parse assignment operators with property reads", func(t *testing.T) {
			checkAction("a = b")(t)
			checkAction("a += b")(t)
			checkAction("a -= b")(t)
			checkAction("a *= b")(t)
			checkAction("a /= b")(t)
			checkAction("a %= b")(t)
			checkAction("a **= b")(t)
			checkAction("a &&= b")(t)
			checkAction("a ||= b")(t)
			checkAction("a ??= b")(t)
		})

		t.Run("should parse assignment operators with keyed reads", func(t *testing.T) {
			checkAction("a[0] = b")(t)
			checkAction("a[0] += b")(t)
			checkAction("a[0] -= b")(t)
			checkAction("a[0] *= b")(t)
			checkAction("a[0] /= b")(t)
			checkAction("a[0] %= b")(t)
			checkAction("a[0] **= b")(t)
			checkAction("a[0] &&= b")(t)
			checkAction("a[0] ||= b")(t)
			checkAction("a[0] ??= b")(t)
		})

		t.Run("literals", func(t *testing.T) {
			t.Run("should parse array", func(t *testing.T) {
				checkAction("[1][0]")(t)
				checkAction("[[1]][0][0]")(t)
				checkAction("[]")(t)
				checkAction("[].length")(t)
				checkAction("[1, 2].length")(t)
				checkAction("[1, 2,]", "[1, 2]")(t)
			})

			t.Run("should parse map", func(t *testing.T) {
				checkAction("{}")(t)
				checkAction(`{a: 1, "b": 2}[2]`)(t)
				checkAction(`{}["a"]`)(t)
				checkAction(`{a: 1, b: 2,}`, `{a: 1, b: 2}`)(t)
			})

			t.Run("should only allow identifier, string, or keyword as map key", func(t *testing.T) {
				expectActionError("{(:0}", "expected identifier, keyword, or string")(t)
				expectActionError("{1234:0}", "expected identifier, keyword, or string")(t)
				expectActionError("{#myField:0}", "expected identifier, keyword or string")(t)
			})

			t.Run("should parse property shorthand declarations", func(t *testing.T) {
				checkAction("{a, b, c}", "{a: a, b: b, c: c}")(t)
				checkAction("{a: 1, b}", "{a: 1, b: b}")(t)
				checkAction("{a, b: 1}", "{a: a, b: 1}")(t)
				checkAction("{a: 1, b, c: 2}", "{a: 1, b: b, c: 2}")(t)
			})

			t.Run("should not allow property shorthand declaration on quoted properties", func(t *testing.T) {
				expectActionError(`{"a-b"}`, "expected : at column 7")(t)
			})

			t.Run("should not infer invalid identifiers as shorthand property declarations", func(t *testing.T) {
				expectActionError("{a.b}", "expected } at column 3")(t)
				expectActionError(`{a["b"]}`, "expected } at column 3")(t)
				expectActionError("{1234}", " expected identifier, keyword, or string at column 2")(t)
			})
		})

		t.Run("member access", func(t *testing.T) {
			t.Run("should parse field access", func(t *testing.T) {
				checkAction("a")(t)
				checkAction("this.a", "a")(t)
				checkAction("a.a")(t)
			})

			t.Run("should error for private identifiers with implicit receiver", func(t *testing.T) {
				checkActionWithError(
					"#privateField",
					"",
					"Private identifiers are not supported. Unexpected private identifier: #privateField at column 1",
				)(t)
			})

			t.Run("should only allow identifier or keyword as member names", func(t *testing.T) {
				checkActionWithError("x.", "x.", "identifier or keyword")(t)
				checkActionWithError("x.(", "x.", "identifier or keyword")(t)
				checkActionWithError("x. 1234", "x.", "identifier or keyword")(t)
				checkActionWithError(`x."foo"`, "x.", "identifier or keyword")(t)
				checkActionWithError(
					"x.#privateField",
					"x.",
					"Private identifiers are not supported. Unexpected private identifier: #privateField, expected identifier or keyword",
				)(t)
			})

			t.Run("should parse safe field access", func(t *testing.T) {
				checkAction("a?.a")(t)
				checkAction("a.a?.a")(t)
			})

			t.Run("should parse incomplete safe field accesses", func(t *testing.T) {
				checkActionWithError("a?.a.", "a?.a.", "identifier or keyword")(t)
				checkActionWithError("a.a?.a.", "a.a?.a.", "identifier or keyword")(t)
				checkActionWithError("a.a?.a?. 1234", "a.a?.a?.", "identifier or keyword")(t)
			})
		})

		t.Run("property write", func(t *testing.T) {
			t.Run("should parse property writes", func(t *testing.T) {
				checkAction("a.a = 1 + 2")(t)
				checkAction("this.a.a = 1 + 2", "a.a = 1 + 2")(t)
				checkAction("a.a.a = 1 + 2")(t)
			})

			t.Run("malformed property writes", func(t *testing.T) {
				t.Run("should recover on empty rvalues", func(t *testing.T) {
					checkActionWithError("a.a = ", "a.a = ", "Unexpected end of expression")(t)
				})

				t.Run("should recover on incomplete rvalues", func(t *testing.T) {
					checkActionWithError("a.a = 1 + ", "a.a = 1 + ", "Unexpected end of expression")(t)
				})

				t.Run("should recover on missing properties", func(t *testing.T) {
					checkActionWithError(
						"a. = 1",
						"a. = 1",
						"Expected identifier for property access at column 2",
					)(t)
				})

				t.Run("should error on writes after a property write", func(t *testing.T) {
					ast := parseAction("a.a = 1 = 2")
					result := expression_parser.Serialize(ast)
					if result != "a.a = 1" {
						t.Errorf("Expected 'a.a = 1', got %q", result)
					}
					if len(ast.Errors) != 1 {
						t.Errorf("Expected 1 error, got %d", len(ast.Errors))
					} else if !contains(ast.Errors[0].Msg, "Unexpected token '='") {
						t.Errorf("Expected error containing 'Unexpected token '='', got %q", ast.Errors[0].Msg)
					}
				})
			})
		})

		t.Run("calls", func(t *testing.T) {
			t.Run("should parse calls", func(t *testing.T) {
				checkAction("fn()")(t)
				checkAction("add(1, 2)")(t)
				checkAction("a.add(1, 2)")(t)
				checkAction("fn().add(1, 2)")(t)
				checkAction("fn()(1, 2)")(t)
			})

			t.Run("should parse an EmptyExpr with a correct span for a trailing empty argument", func(t *testing.T) {
				ast := parseAction("fn(1, )")
				call, ok := ast.AST.(*expression_parser.Call)
				if !ok {
					t.Fatalf("Expected Call, got %T", ast.AST)
				}
				if len(call.Args) != 2 {
					t.Fatalf("Expected 2 args, got %d", len(call.Args))
				}
				// Check that second arg is EmptyExpr
				emptyExpr, ok := call.Args[1].(*expression_parser.EmptyExpr)
				if !ok {
					t.Fatalf("Expected EmptyExpr, got %T", call.Args[1])
				}
				// Check span
				if emptyExpr.SourceSpan().Start != 5 || emptyExpr.SourceSpan().End != 6 {
					t.Errorf("Expected span [5, 6], got [%d, %d]",
						emptyExpr.SourceSpan().Start, emptyExpr.SourceSpan().End)
				}
			})

			t.Run("should parse safe calls", func(t *testing.T) {
				checkAction("fn?.()")(t)
				checkAction("add?.(1, 2)")(t)
				checkAction("a.add?.(1, 2)")(t)
				checkAction("a?.add?.(1, 2)")(t)
				checkAction("fn?.().add?.(1, 2)")(t)
				checkAction("fn?.()?.(1, 2)")(t)
			})
		})

		t.Run("keyed read", func(t *testing.T) {
			t.Run("should parse keyed reads", func(t *testing.T) {
				checkBinding(`a["a"]`)(t)
				checkBinding(`this.a["a"]`, `a["a"]`)(t)
				checkBinding(`a.a["a"]`)(t)
			})

			t.Run("should parse safe keyed reads", func(t *testing.T) {
				checkBinding(`a?.["a"]`)(t)
				checkBinding(`this.a?.["a"]`, `a?.["a"]`)(t)
				checkBinding(`a.a?.["a"]`)(t)
				checkBinding(`a.a?.["a" | foo]`, `a.a?.[("a" | foo)]`)(t)
			})

			t.Run("malformed keyed reads", func(t *testing.T) {
				t.Run("should recover on missing keys", func(t *testing.T) {
					checkActionWithError("a[]", "a[]", "Key access cannot be empty")(t)
				})

				t.Run("should recover on incomplete expression keys", func(t *testing.T) {
					checkActionWithError("a[1 + ]", "a[1 + ]", "Unexpected token ]")(t)
				})

				t.Run("should recover on unterminated keys", func(t *testing.T) {
					checkActionWithError(
						"a[1 + 2",
						"a[1 + 2]",
						"Missing expected ] at the end of the expression",
					)(t)
				})

				t.Run("should recover on incomplete and unterminated keys", func(t *testing.T) {
					checkActionWithError(
						"a[1 + ",
						"a[1 + ]",
						"Missing expected ] at the end of the expression",
					)(t)
				})
			})
		})

		t.Run("keyed write", func(t *testing.T) {
			t.Run("should parse keyed writes", func(t *testing.T) {
				checkAction(`a["a"] = 1 + 2`)(t)
				checkAction(`this.a["a"] = 1 + 2`, `a["a"] = 1 + 2`)(t)
				checkAction(`a.a["a"] = 1 + 2`)(t)
			})

			t.Run("should report on safe keyed writes", func(t *testing.T) {
				expectActionError(`a?.["a"] = 123`, "cannot be used in the assignment")(t)
			})

			t.Run("malformed keyed writes", func(t *testing.T) {
				t.Run("should recover on empty rvalues", func(t *testing.T) {
					checkActionWithError(`a["a"] = `, `a["a"] = `, "Unexpected end of expression")(t)
				})

				t.Run("should recover on incomplete rvalues", func(t *testing.T) {
					checkActionWithError(`a["a"] = 1 + `, `a["a"] = 1 + `, "Unexpected end of expression")(t)
				})

				t.Run("should recover on missing keys", func(t *testing.T) {
					checkActionWithError("a[] = 1", "a[] = 1", "Key access cannot be empty")(t)
				})

				t.Run("should recover on incomplete expression keys", func(t *testing.T) {
					checkActionWithError("a[1 + ] = 1", "a[1 + ] = 1", "Unexpected token ]")(t)
				})

				t.Run("should recover on unterminated keys", func(t *testing.T) {
					checkActionWithError("a[1 + 2 = 1", "a[1 + 2] = 1", "Missing expected ]")(t)
				})

				t.Run("should recover on incomplete and unterminated keys", func(t *testing.T) {
					ast := parseAction("a[1 + = 1")
					result := Unparse(ast.AST)
					if result != "a[1 + ] = 1" {
						t.Errorf("Expected 'a[1 + ] = 1', got %q", result)
					}
					errors := ast.Errors
					if len(errors) != 2 {
						t.Errorf("Expected 2 errors, got %d", len(errors))
					} else {
						if !contains(errors[0].Msg, "Unexpected token =") {
							t.Errorf("Expected first error containing 'Unexpected token =', got %q", errors[0].Msg)
						}
						if !contains(errors[1].Msg, "Missing expected ]") {
							t.Errorf("Expected second error containing 'Missing expected ]', got %q", errors[1].Msg)
						}
					}
				})

				t.Run("should error on writes after a keyed write", func(t *testing.T) {
					ast := parseAction("a[1] = 1 = 2")
					result := expression_parser.Serialize(ast)
					if result != "a[1] = 1" {
						t.Errorf("Expected 'a[1] = 1', got %q", result)
					}
					if len(ast.Errors) != 1 {
						t.Errorf("Expected 1 error, got %d", len(ast.Errors))
					} else if !contains(ast.Errors[0].Msg, "Unexpected token '='") {
						t.Errorf("Expected error containing 'Unexpected token '='', got %q", ast.Errors[0].Msg)
					}
				})

				t.Run("should recover on parenthesized empty rvalues", func(t *testing.T) {
					ast := parseAction("(a[1] = b) = c = d")
					result := expression_parser.Serialize(ast)
					if result != "(a[1] = b)" {
						t.Errorf("Expected '(a[1] = b)', got %q", result)
					}
					if len(ast.Errors) != 1 {
						t.Errorf("Expected 1 error, got %d", len(ast.Errors))
					} else if !contains(ast.Errors[0].Msg, "Unexpected token '='") {
						t.Errorf("Expected error containing 'Unexpected token '='', got %q", ast.Errors[0].Msg)
					}
				})
			})
		})

		t.Run("conditional", func(t *testing.T) {
			t.Run("should parse ternary/conditional expressions", func(t *testing.T) {
				checkAction("7 == 3 + 4 ? 10 : 20")(t)
				checkAction("false ? 10 : 20")(t)
			})

			t.Run("should report incorrect ternary operator syntax", func(t *testing.T) {
				expectActionError("true?1", "Conditional expression true?1 requires all 3 expressions")(t)
			})
		})

		t.Run("assignment", func(t *testing.T) {
			t.Run("should support field assignments", func(t *testing.T) {
				checkAction("a = 12")(t)
				checkAction("a.a.a = 123")(t)
				checkAction("a = 123; b = 234;")(t)
			})

			t.Run("should report on safe field assignments", func(t *testing.T) {
				expectActionError("a?.a = 123", "cannot be used in the assignment")(t)
			})

			t.Run("should support array updates", checkAction("a[0] = 200"))
		})

		t.Run("should error when using pipes", func(t *testing.T) {
			expectActionError("x|blah", "Cannot have a pipe")(t)
		})

		t.Run("should report when encountering interpolation", func(t *testing.T) {
			expectActionError("{{a()}}", "Got interpolation ({{}}) where expression was expected")(t)
		})

		t.Run("should not report interpolation inside a string", func(t *testing.T) {
			ast1 := parseAction(`"{{a()}}"`)
			if len(ast1.Errors) != 0 {
				t.Errorf("Expected no errors, got %d", len(ast1.Errors))
			}
			ast2 := parseAction(`'{{a()}}'`)
			if len(ast2.Errors) != 0 {
				t.Errorf("Expected no errors, got %d", len(ast2.Errors))
			}
			// TypeScript: "{{a('\\"')}}" -> "{{a('\"')}}" (the \\" becomes \")
			ast3 := parseAction("\"{{a('\\\"')}}\"")
			if len(ast3.Errors) != 0 {
				t.Errorf("Expected no errors, got %d", len(ast3.Errors))
			}
			// TypeScript: '{{a("\\'")}}' -> '{{a("\'")}}' (the \\' in template literal becomes \')
			// In Go raw string (backtick), \ is literal, so \' is backslash + single quote
			ast4 := parseAction(`'{{a("\'")}}' `)
			if len(ast4.Errors) != 0 {
				t.Errorf("Expected no errors, got %d", len(ast4.Errors))
			}
		})

		t.Run("template literals", func(t *testing.T) {
			t.Run("should parse template literals without interpolations", func(t *testing.T) {
				checkBinding("`hello world`")(t)
				checkBinding("`foo $`")(t)
				checkBinding("`foo }`")(t)
				checkBinding("`foo $ {}`")(t)
			})

			t.Run("should parse template literals with interpolations", func(t *testing.T) {
				checkBinding("`hello ${name}`")(t)
				checkBinding("`${name} Johnson`")(t)
				checkBinding("`foo${bar}baz`")(t)
				checkBinding("`${a} - ${b} - ${c}`")(t)
				checkBinding("`foo ${{$: true}} baz`")(t)
				checkBinding("`foo ${`hello ${`${a} - b`}`} baz`")(t)
				checkBinding("[`hello ${name}`, `see ${name} later`]")(t)
				checkBinding("`hello ${name}` + 123")(t)
			})

			t.Run("should parse template literals with pipes inside interpolations", func(t *testing.T) {
				checkBinding("`hello ${name | capitalize}!!!`", "`hello ${(name | capitalize)}!!!`")(t)
				checkBinding("`hello ${(name | capitalize)}!!!`", "`hello ${((name | capitalize))}!!!`")(t)
			})

			t.Run("should parse template literals in objects literals", func(t *testing.T) {
				checkBinding("{\"a\": `" + "${name}" + "`}")(t)
				checkBinding("{\"a\": `hello " + "${name}" + "!`}")(t)
				checkBinding("{\"a\": `hello ${`hello ${`hello`}`}!`}")(t)
				checkBinding("{\"a\": `hello ${{\"b\": `hello`}}`}")(t)
			})

			t.Run("should report error if interpolation is empty", func(t *testing.T) {
				expectBindingError("`hello ${}`", "Template literal interpolation cannot be empty")(t)
			})

			t.Run("should parse tagged template literals with no interpolations", func(t *testing.T) {
				checkBinding("tag`hello!`")(t)
				checkBinding("tags.first`hello!`")(t)
				checkBinding("tags[0]`hello!`")(t)
				checkBinding("tag()`hello!`")(t)
				checkBinding("(tag ?? otherTag)`hello!`")(t)
				checkBinding("tag!`hello!`")(t)
			})

			t.Run("should parse tagged template literals with interpolations", func(t *testing.T) {
				checkBinding("tag`hello ${name}!`")(t)
				checkBinding("tags.first`hello ${name}!`")(t)
				checkBinding("tags[0]`hello ${name}!`")(t)
				checkBinding("tag()`hello ${name}!`")(t)
				checkBinding("(tag ?? otherTag)`hello ${name}!`")(t)
				checkBinding("tag!`hello ${name}!`")(t)
			})

			t.Run("should not mistake operator for tagged literal tag", func(t *testing.T) {
				checkBinding("typeof `hello!`")(t)
				checkBinding("typeof `hello ${name}!`")(t)
			})
		})

		t.Run("regular expression literals", func(t *testing.T) {
			t.Run("should parse a regular expression literal without flags", func(t *testing.T) {
				checkBinding("/abc/")(t)
				checkBinding("/[a/]$/")(t)
				checkBinding("/a\\w+/")(t)
				checkBinding("/^http:\\/\\/foo\\.bar/")(t)
			})

			t.Run("should parse a regular expression literal with flags", func(t *testing.T) {
				checkBinding("/abc/g")(t)
				checkBinding("/[a/]$/gi")(t)
				checkBinding("/a\\w+/gim")(t)
				checkBinding("/^http:\\/\\/foo\\.bar/i")(t)
			})

			t.Run("should parse a regular expression that is a part of other expressions", func(t *testing.T) {
				checkBinding(`/abc/.test("foo")`)(t)
				checkBinding(`"foo".match(/(abc)/)[1].toUpperCase()`)(t)
				checkBinding(`/abc/.test("foo") && something || somethingElse`)(t)
			})

			t.Run("should report invalid regular expression flag", func(t *testing.T) {
				expectBindingError(`"foo".match(/abc/O)`, `Unsupported regular expression flag "O"`)(t)
			})

			t.Run("should report duplicated regular expression flags", func(t *testing.T) {
				expectBindingError(`"foo".match(/abc/gig)`, `Duplicate regular expression flag "g"`)(t)
			})
		})
	})

	t.Run("parseBinding", func(t *testing.T) {
		t.Run("pipes", func(t *testing.T) {
			t.Run("should parse pipes", func(t *testing.T) {
				checkBinding("a(b | c)", "a((b | c))")(t)
				checkBinding("a.b(c.d(e) | f)", "a.b((c.d(e) | f))")(t)
				checkBinding("[1, 2, 3] | a", "([1, 2, 3] | a)")(t)
				checkBinding(`{a: 1, "b": 2} | c`, `({a: 1, "b": 2} | c)`)(t)
				checkBinding("a[b] | c", "(a[b] | c)")(t)
				checkBinding("a?.b | c", "(a?.b | c)")(t)
				checkBinding("true | a", "(true | a)")(t)
				checkBinding("a | b:c | d", "((a | b:c) | d)")(t)
				checkBinding("a | b:(c | d)", "(a | b:((c | d)))")(t)
			})

			t.Run("should parse incomplete pipes", func(t *testing.T) {
				cases := []struct {
					name   string
					input  string
					output string
					err    string
				}{
					{"should parse missing pipe names: end", "a | b | ", "((a | b) | )", "Unexpected end of input, expected identifier or keyword"},
					{"should parse missing pipe names: middle", "a | | b", "((a | ) | b)", "Unexpected token |, expected identifier or keyword"},
					{"should parse missing pipe names: start", " | a | b", "(( | a) | b)", "Unexpected token |"},
					{"should parse missing pipe args: end", "a | b | c: ", "((a | b) | c:)", "Unexpected end of expression"},
					{"should parse missing pipe args: middle", "a | b: | c", "((a | b:) | c)", "Unexpected token |"},
					{"should parse incomplete pipe args", "a | b: (a | ) + | c", "((a | b:((a | )) + ) | c)", "Unexpected token |"},
				}
				for _, tc := range cases {
					t.Run(tc.name, func(t *testing.T) {
						checkBinding(tc.input, tc.output)(t)
						expectBindingError(tc.input, tc.err)(t)
					})
				}

				t.Run("should parse an incomplete pipe with a source span that includes trailing whitespace", func(t *testing.T) {
					bindingText := "foo | "
					binding := parseBinding(bindingText)
					pipe, ok := binding.AST.(*expression_parser.BindingPipe)
					if !ok {
						t.Fatalf("Expected BindingPipe, got %T", binding.AST)
					}
					// The sourceSpan should include all characters of the input.
					if pipe.SourceSpan().Start != 0 || pipe.SourceSpan().End != len(bindingText) {
						t.Errorf("Expected sourceSpan [0, %d], got [%d, %d]",
							len(bindingText), pipe.SourceSpan().Start, pipe.SourceSpan().End)
					}
					// The nameSpan should be positioned at the end of the input.
					if pipe.NameSpan().Start != len(bindingText) || pipe.NameSpan().End != len(bindingText) {
						t.Errorf("Expected nameSpan [%d, %d], got [%d, %d]",
							len(bindingText), len(bindingText), pipe.NameSpan().Start, pipe.NameSpan().End)
					}
				})

				t.Run("should parse pipes with the correct type when supportsDirectPipeReferences is enabled", func(t *testing.T) {
					binding1 := parseBinding("0 | Foo", true)
					pipe1, ok := binding1.AST.(*expression_parser.BindingPipe)
					if !ok {
						t.Fatalf("Expected BindingPipe, got %T", binding1.AST)
					}
					if pipe1.Type != expression_parser.ReferencedDirectly {
						t.Errorf("Expected ReferencedDirectly, got %v", pipe1.Type)
					}
					binding2 := parseBinding("0 | foo", true)
					pipe2, ok := binding2.AST.(*expression_parser.BindingPipe)
					if !ok {
						t.Fatalf("Expected BindingPipe, got %T", binding2.AST)
					}
					if pipe2.Type != expression_parser.ReferencedByName {
						t.Errorf("Expected ReferencedByName, got %v", pipe2.Type)
					}
				})

				t.Run("should parse pipes with the correct type when supportsDirectPipeReferences is disabled", func(t *testing.T) {
					binding1 := parseBinding("0 | Foo", false)
					pipe1, ok := binding1.AST.(*expression_parser.BindingPipe)
					if !ok {
						t.Fatalf("Expected BindingPipe, got %T", binding1.AST)
					}
					if pipe1.Type != expression_parser.ReferencedByName {
						t.Errorf("Expected ReferencedByName, got %v", pipe1.Type)
					}
					binding2 := parseBinding("0 | foo", false)
					pipe2, ok := binding2.AST.(*expression_parser.BindingPipe)
					if !ok {
						t.Fatalf("Expected BindingPipe, got %T", binding2.AST)
					}
					if pipe2.Type != expression_parser.ReferencedByName {
						t.Errorf("Expected ReferencedByName, got %v", pipe2.Type)
					}
				})
			})

			t.Run("should only allow identifier or keyword as formatter names", func(t *testing.T) {
				expectBindingError(`"Foo"|(`, "identifier or keyword")(t)
				expectBindingError(`"Foo"|1234`, "identifier or keyword")(t)
				expectBindingError(`"Foo"|"uppercase"`, "identifier or keyword")(t)
				expectBindingError(`"Foo"|#privateIdentifier"`, "identifier or keyword")(t)
			})

			t.Run("should not crash when prefix part is not tokenizable", func(t *testing.T) {
				checkBinding(`"a:b"`, `"a:b"`)(t)
			})
		})

		t.Run("should store the source in the result", func(t *testing.T) {
			ast := parseBinding("someExpr")
			if ast.Source == nil || *ast.Source != "someExpr" {
				t.Errorf("Expected source 'someExpr', got %v", ast.Source)
			}
		})

		t.Run("should report chain expressions", func(t *testing.T) {
			ast := parseBinding("1;2")
			expectError(ast, "contain chained expression")(t)
		})

		t.Run("should report assignment", func(t *testing.T) {
			ast := parseBinding("a=2")
			expectError(ast, "contain assignments")(t)
		})

		t.Run("should report when encountering interpolation", func(t *testing.T) {
			expectBindingError("{{a.b}}", "Got interpolation ({{}}) where expression was expected")(t)
		})

		t.Run("should not report interpolation inside a string", func(t *testing.T) {
			ast1 := parseBinding(`"{{exp}}"`)
			if len(ast1.Errors) != 0 {
				t.Errorf("Expected no errors, got %d", len(ast1.Errors))
			}
			ast2 := parseBinding(`'{{exp}}'`)
			if len(ast2.Errors) != 0 {
				t.Errorf("Expected no errors, got %d", len(ast2.Errors))
			}
			// TypeScript: '{{\\"}}' -> '{{\"}}' (8 chars: ', {, {, \, ", }, }, ')
			// Go: "'{{\\\"}}'" -> '{{\"}}' (8 chars: ', {, {, \, ", }, }, ')
			ast3 := parseBinding("'{{\\\"}}'")
			if len(ast3.Errors) != 0 {
				t.Errorf("Expected no errors, got %d", len(ast3.Errors))
			}
			// TypeScript: '{{\\'}}' -> '{{\'}}' (8 chars: ', {, {, \, ', }, }, ')
			// In Go, \' is not a valid escape sequence, so we construct the string manually
			// The actual string should be: '{{\'}}' where the \' is a single backslash followed by single quote
			ast4 := parseBinding("'{{" + string(rune(92)) + "'}}'")
			if len(ast4.Errors) != 0 {
				t.Errorf("Expected no errors, got %d", len(ast4.Errors))
			}
		})

		t.Run("should parse conditional expression", checkBinding("a < b ? a : b"))

		t.Run("should ignore comments in bindings", func(t *testing.T) {
			checkBinding("a //comment", "a")(t)
		})

		t.Run("should retain // in string literals", func(t *testing.T) {
			checkBinding(`"http://www.google.com"`, `"http://www.google.com"`)(t)
		})

		t.Run("should expose object shorthand information in AST", func(t *testing.T) {
			p := expression_parser.NewParser(expression_parser.NewLexer(), false)
			ast := p.ParseBinding("{bla}", getFakeSpan(""), 0)
			_, ok := ast.AST.(*expression_parser.LiteralMap)
			if !ok {
				t.Fatalf("Expected LiteralMap, got %T", ast.AST)
			}
			// TODO: Check keys[0].isShorthandInitialized when available
		})
	})

	// Note: parseTemplateBindings, parseInterpolation, parseSimpleBinding,
	// error recovery, and parse spans tests are very complex and would require
	// additional helper functions. These should be added in subsequent iterations.
	// The current implementation covers the core parseAction and parseBinding test cases.
}
