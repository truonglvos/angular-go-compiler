package schema_test

import (
	"ngc-go/packages/compiler/src/schema"
	"testing"
)

func TestIsTrustedTypesSink(t *testing.T) {
	t.Run("should classify Trusted Types sinks", func(t *testing.T) {
		if !schema.IsTrustedTypesSink("iframe", "srcdoc") {
			t.Error("Expected IsTrustedTypesSink('iframe', 'srcdoc') to be true")
		}
		if !schema.IsTrustedTypesSink("p", "innerHTML") {
			t.Error("Expected IsTrustedTypesSink('p', 'innerHTML') to be true")
		}
		if !schema.IsTrustedTypesSink("embed", "src") {
			t.Error("Expected IsTrustedTypesSink('embed', 'src') to be true")
		}
		if schema.IsTrustedTypesSink("a", "href") {
			t.Error("Expected IsTrustedTypesSink('a', 'href') to be false")
		}
		if schema.IsTrustedTypesSink("base", "href") {
			t.Error("Expected IsTrustedTypesSink('base', 'href') to be false")
		}
		if schema.IsTrustedTypesSink("div", "style") {
			t.Error("Expected IsTrustedTypesSink('div', 'style') to be false")
		}
	})

	t.Run("should classify Trusted Types sinks case insensitive", func(t *testing.T) {
		if !schema.IsTrustedTypesSink("p", "iNnErHtMl") {
			t.Error("Expected IsTrustedTypesSink('p', 'iNnErHtMl') to be true")
		}
		if schema.IsTrustedTypesSink("p", "formaction") {
			t.Error("Expected IsTrustedTypesSink('p', 'formaction') to be false")
		}
		if schema.IsTrustedTypesSink("p", "formAction") {
			t.Error("Expected IsTrustedTypesSink('p', 'formAction') to be false")
		}
	})

	t.Run("should classify attributes as Trusted Types sinks", func(t *testing.T) {
		if !schema.IsTrustedTypesSink("p", "innerHtml") {
			t.Error("Expected IsTrustedTypesSink('p', 'innerHtml') to be true")
		}
		if schema.IsTrustedTypesSink("p", "formaction") {
			t.Error("Expected IsTrustedTypesSink('p', 'formaction') to be false")
		}
	})
}

