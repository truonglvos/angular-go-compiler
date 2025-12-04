package main_test

import (
	"ngc-go/packages/compiler/src/schema"
	"strings"
	"testing"
)

func TestSecurityRelated(t *testing.T) {
	t.Run("should have no overlap between IFRAME_SECURITY_SENSITIVE_ATTRS and SECURITY_SCHEMA", func(t *testing.T) {
		// The `IFRAME_SECURITY_SENSITIVE_ATTRS` and `SECURITY_SCHEMA` tokens configure sanitization
		// and validation rules and used to pick the right sanitizer function.
		// This test verifies that there is no overlap between two sets of rules to flag
		// a situation when 2 sanitizer functions may be needed at the same time (in which
		// case, compiler logic should be extended to support that).

		securitySchema := schema.SecuritySchema()
		schemaSet := make(map[string]bool)

		for key := range securitySchema {
			schemaSet[strings.ToLower(key)] = true
		}

		hasOverlap := false
		for attr := range schema.IframeSecuritySensitiveAttrs {
			if schemaSet["*|"+attr] || schemaSet["iframe|"+attr] {
				hasOverlap = true
				break
			}
		}

		if hasOverlap {
			t.Error("Expected no overlap between IFRAME_SECURITY_SENSITIVE_ATTRS and SECURITY_SCHEMA")
		}
	})
}
