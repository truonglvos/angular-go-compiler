package main_test

import (
	"ngc-go/packages/compiler/src/css"
	"testing"
)

func TestIsStyleUrlResolvable(t *testing.T) {
	strPtr := func(s string) *string { return &s }
	
	t.Run("should resolve relative urls", func(t *testing.T) {
		if !css.IsStyleUrlResolvable(strPtr("someUrl.css")) {
			t.Error("Expected IsStyleUrlResolvable('someUrl.css') to be true")
		}
	})

	t.Run("should resolve package: urls", func(t *testing.T) {
		if !css.IsStyleUrlResolvable(strPtr("package:someUrl.css")) {
			t.Error("Expected IsStyleUrlResolvable('package:someUrl.css') to be true")
		}
	})

	t.Run("should not resolve empty urls", func(t *testing.T) {
		// Test null case
		if css.IsStyleUrlResolvable(nil) {
			t.Error("Expected IsStyleUrlResolvable(nil) to be false")
		}
		// Test empty string
		if css.IsStyleUrlResolvable(strPtr("")) {
			t.Error("Expected IsStyleUrlResolvable('') to be false")
		}
	})

	t.Run("should not resolve urls with other schema", func(t *testing.T) {
		if css.IsStyleUrlResolvable(strPtr("http://otherurl")) {
			t.Error("Expected IsStyleUrlResolvable('http://otherurl') to be false")
		}
	})

	t.Run("should not resolve urls with absolute paths", func(t *testing.T) {
		if css.IsStyleUrlResolvable(strPtr("/otherurl")) {
			t.Error("Expected IsStyleUrlResolvable('/otherurl') to be false")
		}
		if css.IsStyleUrlResolvable(strPtr("//otherurl")) {
			t.Error("Expected IsStyleUrlResolvable('//otherurl') to be false")
		}
	})
}
