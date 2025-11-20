package expression_parser_test

import (
	"testing"

	"ngc-go/packages/compiler/src/expression_parser"
)

func parse(expression string) *expression_parser.ASTWithSource {
	return parser.ParseBinding(expression, getFakeSpan(""), 0)
}

func TestSerializer(t *testing.T) {
	t.Run("serialize", func(t *testing.T) {
		t.Run("serializes unary plus", func(t *testing.T) {
			result := expression_parser.Serialize(parse(" + 1234 "))
			if result != "+1234" {
				t.Errorf("Expected '+1234', got %q", result)
			}
		})

		t.Run("serializes unary negative", func(t *testing.T) {
			result := expression_parser.Serialize(parse(" - 1234 "))
			if result != "-1234" {
				t.Errorf("Expected '-1234', got %q", result)
			}
		})

		t.Run("serializes binary operations", func(t *testing.T) {
			result := expression_parser.Serialize(parse(" 1234   +   4321 "))
			if result != "1234 + 4321" {
				t.Errorf("Expected '1234 + 4321', got %q", result)
			}
		})

		t.Run("serializes exponentiation", func(t *testing.T) {
			result := expression_parser.Serialize(parse(" 1  *  2  **  3 "))
			if result != "1 * 2 ** 3" {
				t.Errorf("Expected '1 * 2 ** 3', got %q", result)
			}
		})

		t.Run("serializes chains", func(t *testing.T) {
			result := expression_parser.Serialize(parseAction(" 1234;   4321 "))
			if result != "1234; 4321" {
				t.Errorf("Expected '1234; 4321', got %q", result)
			}
		})

		t.Run("serializes conditionals", func(t *testing.T) {
			result := expression_parser.Serialize(parse(" cond   ?   1234   :   4321 "))
			if result != "cond ? 1234 : 4321" {
				t.Errorf("Expected 'cond ? 1234 : 4321', got %q", result)
			}
		})

		t.Run("serializes `this`", func(t *testing.T) {
			result := expression_parser.Serialize(parse(" this "))
			if result != "this" {
				t.Errorf("Expected 'this', got %q", result)
			}
		})

		t.Run("serializes keyed reads", func(t *testing.T) {
			result := expression_parser.Serialize(parse(" foo   [bar] "))
			if result != "foo[bar]" {
				t.Errorf("Expected 'foo[bar]', got %q", result)
			}
		})

		t.Run("serializes keyed write", func(t *testing.T) {
			result := expression_parser.Serialize(parse(" foo   [bar]   =   baz "))
			if result != "foo[bar] = baz" {
				t.Errorf("Expected 'foo[bar] = baz', got %q", result)
			}
		})

		t.Run("serializes array literals", func(t *testing.T) {
			result := expression_parser.Serialize(parse(" [   foo,   bar,   baz   ] "))
			if result != "[foo, bar, baz]" {
				t.Errorf("Expected '[foo, bar, baz]', got %q", result)
			}
		})

		t.Run("serializes object literals", func(t *testing.T) {
			result := expression_parser.Serialize(parse(" {   foo:   bar,   baz:   test   } "))
			if result != "{foo: bar, baz: test}" {
				t.Errorf("Expected '{foo: bar, baz: test}', got %q", result)
			}
		})

		t.Run("serializes primitives", func(t *testing.T) {
			result := expression_parser.Serialize(parse(` 'test' `))
			if result != "'test'" {
				t.Errorf("Expected ''test'', got %q", result)
			}
			result = expression_parser.Serialize(parse(` "test" `))
			if result != "'test'" {
				t.Errorf("Expected ''test'', got %q", result)
			}
			result = expression_parser.Serialize(parse(" true "))
			if result != "true" {
				t.Errorf("Expected 'true', got %q", result)
			}
			result = expression_parser.Serialize(parse(" false "))
			if result != "false" {
				t.Errorf("Expected 'false', got %q", result)
			}
			result = expression_parser.Serialize(parse(" 1234 "))
			if result != "1234" {
				t.Errorf("Expected '1234', got %q", result)
			}
			result = expression_parser.Serialize(parse(" null "))
			if result != "null" {
				t.Errorf("Expected 'null', got %q", result)
			}
			result = expression_parser.Serialize(parse(" undefined "))
			if result != "undefined" {
				t.Errorf("Expected 'undefined', got %q", result)
			}
		})

		t.Run("escapes string literals", func(t *testing.T) {
			result := expression_parser.Serialize(parse(` 'Hello, \'World\'...' `))
			if result != `'Hello, \'World\'...'` {
				t.Errorf("Expected 'Hello, \\'World\\'...', got %q", result)
			}
			result = expression_parser.Serialize(parse(` 'Hello, \"World\"...' `))
			if result != `'Hello, "World"...'` {
				t.Errorf("Expected 'Hello, \"World\"...', got %q", result)
			}
		})

		t.Run("serializes pipes", func(t *testing.T) {
			result := expression_parser.Serialize(parse(" foo   |   pipe "))
			if result != "foo | pipe" {
				t.Errorf("Expected 'foo | pipe', got %q", result)
			}
		})

		t.Run("serializes not prefixes", func(t *testing.T) {
			result := expression_parser.Serialize(parse(" !   foo "))
			if result != "!foo" {
				t.Errorf("Expected '!foo', got %q", result)
			}
		})

		t.Run("serializes non-null assertions", func(t *testing.T) {
			result := expression_parser.Serialize(parse(" foo   ! "))
			if result != "foo!" {
				t.Errorf("Expected 'foo!', got %q", result)
			}
		})

		t.Run("serializes property reads", func(t *testing.T) {
			result := expression_parser.Serialize(parse(" foo   .   bar "))
			if result != "foo.bar" {
				t.Errorf("Expected 'foo.bar', got %q", result)
			}
		})

		t.Run("serializes property writes", func(t *testing.T) {
			result := expression_parser.Serialize(parseAction(" foo   .   bar   =   baz "))
			if result != "foo.bar = baz" {
				t.Errorf("Expected 'foo.bar = baz', got %q", result)
			}
		})

		t.Run("serializes safe property reads", func(t *testing.T) {
			result := expression_parser.Serialize(parse(" foo   ?.   bar "))
			if result != "foo?.bar" {
				t.Errorf("Expected 'foo?.bar', got %q", result)
			}
		})

		t.Run("serializes safe keyed reads", func(t *testing.T) {
			result := expression_parser.Serialize(parse(" foo   ?.   [   bar   ] "))
			if result != "foo?.[bar]" {
				t.Errorf("Expected 'foo?.[bar]', got %q", result)
			}
		})

		t.Run("serializes calls", func(t *testing.T) {
			result := expression_parser.Serialize(parse(" foo   (   ) "))
			if result != "foo()" {
				t.Errorf("Expected 'foo()', got %q", result)
			}
			result = expression_parser.Serialize(parse(" foo   (   bar   ) "))
			if result != "foo(bar)" {
				t.Errorf("Expected 'foo(bar)', got %q", result)
			}
			result = expression_parser.Serialize(parse(" foo   (   bar   ,   ) "))
			if result != "foo(bar, )" {
				t.Errorf("Expected 'foo(bar, )', got %q", result)
			}
			result = expression_parser.Serialize(parse(" foo   (   bar   ,   baz   ) "))
			if result != "foo(bar, baz)" {
				t.Errorf("Expected 'foo(bar, baz)', got %q", result)
			}
		})

		t.Run("serializes safe calls", func(t *testing.T) {
			result := expression_parser.Serialize(parse(" foo   ?.   (   ) "))
			if result != "foo?.()" {
				t.Errorf("Expected 'foo?.()', got %q", result)
			}
			result = expression_parser.Serialize(parse(" foo   ?.   (   bar   ) "))
			if result != "foo?.(bar)" {
				t.Errorf("Expected 'foo?.(bar)', got %q", result)
			}
			result = expression_parser.Serialize(parse(" foo   ?.   (   bar   ,   ) "))
			if result != "foo?.(bar, )" {
				t.Errorf("Expected 'foo?.(bar, )', got %q", result)
			}
			result = expression_parser.Serialize(parse(" foo   ?.   (   bar   ,   baz   ) "))
			if result != "foo?.(bar, baz)" {
				t.Errorf("Expected 'foo?.(bar, baz)', got %q", result)
			}
		})

		t.Run("serializes void expressions", func(t *testing.T) {
			result := expression_parser.Serialize(parse(" void   0 "))
			if result != "void 0" {
				t.Errorf("Expected 'void 0', got %q", result)
			}
		})

		t.Run("serializes in expressions", func(t *testing.T) {
			result := expression_parser.Serialize(parse(" foo   in   bar "))
			if result != "foo in bar" {
				t.Errorf("Expected 'foo in bar', got %q", result)
			}
		})
	})
}
