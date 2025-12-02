package ml_parser_test

import (
	"ngc-go/packages/compiler/src/ml_parser"
	"ngc-go/packages/compiler/test/ml_parser/util"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNodeSerializer(t *testing.T) {
	parser := ml_parser.NewHtmlParser()

	t.Run("should support element", func(t *testing.T) {
		html := "<p></p>"
		ast := parser.Parse(html, "url", nil)
		result := util.SerializeNodes(ast.RootNodes)
		expected := []string{html}
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("SerializeNodes() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should support attributes", func(t *testing.T) {
		html := "<p k=\"value\"></p>"
		ast := parser.Parse(html, "url", nil)
		result := util.SerializeNodes(ast.RootNodes)
		expected := []string{html}
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("SerializeNodes() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should support text", func(t *testing.T) {
		html := "some text"
		ast := parser.Parse(html, "url", nil)
		result := util.SerializeNodes(ast.RootNodes)
		expected := []string{html}
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("SerializeNodes() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should support expansion", func(t *testing.T) {
		html := "{number, plural, =0 {none} =1 {one} other {many}}"
		options := &ml_parser.TokenizeOptions{
			TokenizeExpansionForms: boolPtr(true),
		}
		ast := parser.Parse(html, "url", options)
		result := util.SerializeNodes(ast.RootNodes)
		expected := []string{html}
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("SerializeNodes() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should support comment", func(t *testing.T) {
		html := "<!--comment-->"
		options := &ml_parser.TokenizeOptions{
			TokenizeExpansionForms: boolPtr(true),
		}
		ast := parser.Parse(html, "url", options)
		result := util.SerializeNodes(ast.RootNodes)
		expected := []string{html}
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("SerializeNodes() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("should support nesting", func(t *testing.T) {
		html := `<div i18n="meaning|desc">
        <span>{{ interpolation }}</span>
        <!--comment-->
        <p expansion="true">
          {number, plural, =0 {{sex, select, other {<b>?</b>}}}}
        </p>
      </div>`
		options := &ml_parser.TokenizeOptions{
			TokenizeExpansionForms: boolPtr(true),
		}
		ast := parser.Parse(html, "url", options)
		result := util.SerializeNodes(ast.RootNodes)
		expected := []string{html}
		if diff := cmp.Diff(expected, result); diff != "" {
			t.Errorf("SerializeNodes() mismatch (-want +got):\n%s", diff)
		}
	})
}
