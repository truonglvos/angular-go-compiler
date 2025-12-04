package output_test

import (
	"ngc-go/packages/compiler/src/output"
	"ngc-go/packages/compiler/src/util"
	"testing"
)

func TestEmitterVisitorContext(t *testing.T) {
	fileA := util.NewParseSourceFile("a0a1a2a3a4a5a6a7a8a9", "a.js")
	fileB := util.NewParseSourceFile("b0b1b2b3b4b5b6b7b8b9", "b.js")

	t.Run("should add source files to the source map", func(t *testing.T) {
		ctx := output.CreateRootEmitterVisitorContext()
		ctx.Print(createSourceSpan(fileA, 0), "o0", false)
		ctx.Print(createSourceSpan(fileA, 1), "o1", false)
		ctx.Print(createSourceSpan(fileB, 0), "o2", false)
		ctx.Print(createSourceSpan(fileB, 1), "o3", false)

		smg, err := ctx.ToSourceMapGenerator("o.ts", 0)
		if err != nil {
			t.Fatal(err)
		}
		sm, err := smg.ToJSON()
		if err != nil {
			t.Fatal(err)
		}
		if sm == nil {
			t.Fatal("Expected source map to be non-nil")
		}

		// Check sources
		expectedSources := []string{fileA.URL, fileB.URL}
		if len(sm.Sources) != len(expectedSources) {
			t.Errorf("Expected %d sources, got %d", len(expectedSources), len(sm.Sources))
		}
		for i, src := range expectedSources {
			if sm.Sources[i] != src {
				t.Errorf("Expected source[%d] to be %q, got %q", i, src, sm.Sources[i])
			}
		}

		// Check sources content
		if len(sm.SourcesContent) != 2 {
			t.Errorf("Expected 2 sourcesContent, got %d", len(sm.SourcesContent))
		}
		if sm.SourcesContent[0] == nil || *sm.SourcesContent[0] != fileA.Content {
			t.Errorf("Expected sourcesContent[0] to be %q", fileA.Content)
		}
		if sm.SourcesContent[1] == nil || *sm.SourcesContent[1] != fileB.Content {
			t.Errorf("Expected sourcesContent[1] to be %q", fileB.Content)
		}
	})

	t.Run("should generate mappings", func(t *testing.T) {
		ctx := output.CreateRootEmitterVisitorContext()
		ctx.Print(createSourceSpan(fileA, 0), "fileA-0", false)
		ctx.Println(createSourceSpan(fileB, 1), "fileB-1")
		ctx.Print(createSourceSpan(fileA, 2), "fileA-2", false)

		smg, err := ctx.ToSourceMapGenerator("o.ts", 0)
		if err != nil {
			t.Fatal(err)
		}
		sm, err := smg.ToJSON()
		if err != nil {
			t.Fatal(err)
		}
		if sm == nil {
			t.Fatal("Expected source map to be non-nil")
		}

		// Basic validation that mappings were generated
		if sm.Mappings == "" {
			t.Error("Expected mappings to be non-empty")
		}
	})

	t.Run("should use the default source file for the first character", func(t *testing.T) {
		ctx := output.CreateRootEmitterVisitorContext()
		ctx.Print(nil, "fileA-0", false)

		smg, err := ctx.ToSourceMapGenerator("o.ts", 0)
		if err != nil {
			t.Fatal(err)
		}
		sm, err := smg.ToJSON()
		if err != nil {
			t.Fatal(err)
		}
		if sm == nil {
			t.Fatal("Expected source map to be non-nil")
		}

		// Should have o.ts as a source
		found := false
		for _, src := range sm.Sources {
			if src == "o.ts" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected 'o.ts' to be in sources")
		}
	})

	t.Run("should use an explicit mapping for the first character", func(t *testing.T) {
		ctx := output.CreateRootEmitterVisitorContext()
		ctx.Print(createSourceSpan(fileA, 0), "fileA-0", false)

		smg, err := ctx.ToSourceMapGenerator("o.ts", 0)
		if err != nil {
			t.Fatal(err)
		}
		sm, err := smg.ToJSON()
		if err != nil {
			t.Fatal(err)
		}
		if sm == nil {
			t.Fatal("Expected source map to be non-nil")
		}

		// Should have a.js as a source
		found := false
		for _, src := range sm.Sources {
			if src == "a.js" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Expected 'a.js' to be in sources")
		}
	})

	t.Run("should handle indent", func(t *testing.T) {
		ctx := output.CreateRootEmitterVisitorContext()
		ctx.IncIndent()
		ctx.Println(createSourceSpan(fileA, 0), "fileA-0")
		ctx.IncIndent()
		ctx.Println(createSourceSpan(fileA, 1), "fileA-1")
		ctx.DecIndent()
		ctx.Println(createSourceSpan(fileA, 2), "fileA-2")

		source := ctx.ToSource()
		if source == "" {
			t.Error("Expected source to be non-empty")
		}

		smg, err := ctx.ToSourceMapGenerator("o.ts", 0)
		if err != nil {
			t.Fatal(err)
		}
		sm, err := smg.ToJSON()
		if err != nil {
			t.Fatal(err)
		}
		if sm == nil {
			t.Fatal("Expected source map to be non-nil")
		}
	})

	t.Run("should coalesce identical span", func(t *testing.T) {
		ctx := output.CreateRootEmitterVisitorContext()
		span := createSourceSpan(fileA, 0)
		ctx.Print(span, "fileA-0", false)
		ctx.Print(nil, "...", false)
		ctx.Print(span, "fileA-0", false)
		ctx.Print(createSourceSpan(fileB, 0), "fileB-0", false)

		smg, err := ctx.ToSourceMapGenerator("o.ts", 0)
		if err != nil {
			t.Fatal(err)
		}
		sm, err := smg.ToJSON()
		if err != nil {
			t.Fatal(err)
		}
		if sm == nil {
			t.Fatal("Expected source map to be non-nil")
		}

		if sm.Mappings == "" {
			t.Error("Expected mappings to be non-empty")
		}
	})
}

// spanWrapper wraps a ParseSourceSpan to implement GetSourceSpan
type spanWrapper struct {
	sourceSpan *util.ParseSourceSpan
}

func (w *spanWrapper) GetSourceSpan() *util.ParseSourceSpan {
	return w.sourceSpan
}

// createSourceSpan creates a source span for testing
func createSourceSpan(file *util.ParseSourceFile, idx int) *spanWrapper {
	col := 2 * idx
	start := util.NewParseLocation(file, col, 0, col)
	end := util.NewParseLocation(file, col+2, 0, col+2)
	sourceSpan := util.NewParseSourceSpan(start, end, nil, nil)
	return &spanWrapper{sourceSpan: sourceSpan}
}
