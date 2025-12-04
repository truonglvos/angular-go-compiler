package view_test

import (
	"ngc-go/packages/compiler/src/render3/view"
	"testing"
)

func TestCollectCommentNodes(t *testing.T) {
	html := `
      <!-- eslint-disable-next-line -->
      <div *ngFor="let item of items">
        {{item.name}}
      </div>

      <div>
        <p>
          <!-- some nested comment -->
          <span>Text</span>
        </p>
      </div>
    `

	t.Run("should not include comment nodes by default", func(t *testing.T) {
		templateNoCommentsOption := view.ParseTemplate(html, "", nil)
		if templateNoCommentsOption.CommentNodes != nil {
			t.Errorf("Expected CommentNodes to be nil, got %v", templateNoCommentsOption.CommentNodes)
		}
	})

	t.Run("should not include comment nodes when option is disabled", func(t *testing.T) {
		collectComments := false
		templateCommentsOptionDisabled := view.ParseTemplate(html, "", &view.ParseTemplateOptions{
			CollectCommentNodes: &collectComments,
		})
		if templateCommentsOptionDisabled.CommentNodes != nil {
			t.Errorf("Expected CommentNodes to be nil, got %v", templateCommentsOptionDisabled.CommentNodes)
		}
	})

	t.Run("should include comment nodes when option is enabled", func(t *testing.T) {
		collectComments := true
		templateCommentsOptionEnabled := view.ParseTemplate(html, "", &view.ParseTemplateOptions{
			CollectCommentNodes: &collectComments,
		})
		if templateCommentsOptionEnabled.CommentNodes == nil {
			t.Fatal("Expected CommentNodes to be non-nil")
		}
		if len(templateCommentsOptionEnabled.CommentNodes) != 2 {
			t.Errorf("Expected 2 comment nodes, got %d", len(templateCommentsOptionEnabled.CommentNodes))
		}

		comment1 := templateCommentsOptionEnabled.CommentNodes[0]
		if comment1 == nil {
			t.Fatal("Expected first comment node to be non-nil")
		}
		if comment1.Value != "eslint-disable-next-line" {
			t.Errorf("Expected comment value 'eslint-disable-next-line', got %q", comment1.Value)
		}
		if comment1.SourceSpan() == nil {
			t.Error("Expected comment sourceSpan to be non-nil")
		} else {
			spanStr := comment1.SourceSpan().String()
			if spanStr != "<!-- eslint-disable-next-line -->" {
				t.Errorf("Expected sourceSpan '<!-- eslint-disable-next-line -->', got %q", spanStr)
			}
		}

		comment2 := templateCommentsOptionEnabled.CommentNodes[1]
		if comment2 == nil {
			t.Fatal("Expected second comment node to be non-nil")
		}
		if comment2.Value != "some nested comment" {
			t.Errorf("Expected comment value 'some nested comment', got %q", comment2.Value)
		}
		if comment2.SourceSpan() == nil {
			t.Error("Expected comment sourceSpan to be non-nil")
		} else {
			spanStr := comment2.SourceSpan().String()
			if spanStr != "<!-- some nested comment -->" {
				t.Errorf("Expected sourceSpan '<!-- some nested comment -->', got %q", spanStr)
			}
		}
	})
}
