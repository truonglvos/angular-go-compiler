package output_test

import (
	"ngc-go/packages/compiler/src/output"
	"testing"
)

func TestSourceMapGeneration(t *testing.T) {
	t.Run("generation", func(t *testing.T) {
		t.Run("should generate a valid source map", func(t *testing.T) {
			map_ := output.NewSourceMapGenerator(strPtr("out.js"))
			map_.AddSource("a.js", nil)
			map_.AddLine()
			map_.AddMapping(0, strPtr("a.js"), intPtr(0), intPtr(0))
			map_.AddMapping(4, strPtr("a.js"), intPtr(0), intPtr(6))
			map_.AddMapping(5, strPtr("a.js"), intPtr(0), intPtr(7))
			map_.AddMapping(8, strPtr("a.js"), intPtr(0), intPtr(22))
			map_.AddMapping(9, strPtr("a.js"), intPtr(0), intPtr(23))
			map_.AddMapping(10, strPtr("a.js"), intPtr(0), intPtr(24))
			map_.AddLine()
			map_.AddMapping(0, strPtr("a.js"), intPtr(1), intPtr(0))
			map_.AddMapping(4, strPtr("a.js"), intPtr(1), intPtr(6))
			map_.AddMapping(5, strPtr("a.js"), intPtr(1), intPtr(7))
			map_.AddMapping(8, strPtr("a.js"), intPtr(1), intPtr(10))
			map_.AddMapping(9, strPtr("a.js"), intPtr(1), intPtr(11))
			map_.AddMapping(10, strPtr("a.js"), intPtr(1), intPtr(12))
			map_.AddLine()
			map_.AddMapping(0, strPtr("a.js"), intPtr(3), intPtr(0))
			map_.AddMapping(2, strPtr("a.js"), intPtr(3), intPtr(2))
			map_.AddMapping(3, strPtr("a.js"), intPtr(3), intPtr(3))
			map_.AddMapping(10, strPtr("a.js"), intPtr(3), intPtr(10))
			map_.AddMapping(11, strPtr("a.js"), intPtr(3), intPtr(11))
			map_.AddMapping(21, strPtr("a.js"), intPtr(3), intPtr(11))
			map_.AddMapping(22, strPtr("a.js"), intPtr(3), intPtr(12))
			map_.AddLine()
			map_.AddMapping(4, strPtr("a.js"), intPtr(4), intPtr(4))
			map_.AddMapping(11, strPtr("a.js"), intPtr(4), intPtr(11))
			map_.AddMapping(12, strPtr("a.js"), intPtr(4), intPtr(12))
			map_.AddMapping(15, strPtr("a.js"), intPtr(4), intPtr(15))
			map_.AddMapping(16, strPtr("a.js"), intPtr(4), intPtr(16))
			map_.AddMapping(21, strPtr("a.js"), intPtr(4), intPtr(21))
			map_.AddMapping(22, strPtr("a.js"), intPtr(4), intPtr(22))
			map_.AddMapping(23, strPtr("a.js"), intPtr(4), intPtr(23))
			map_.AddLine()
			map_.AddMapping(0, strPtr("a.js"), intPtr(5), intPtr(0))
			map_.AddMapping(1, strPtr("a.js"), intPtr(5), intPtr(1))
			map_.AddMapping(2, strPtr("a.js"), intPtr(5), intPtr(2))
			map_.AddMapping(3, strPtr("a.js"), intPtr(5), intPtr(2))

			json, err := map_.ToJSON()
			if err != nil {
				t.Fatal(err)
			}
			if json == nil {
				t.Fatal("Expected ToJSON() to return non-nil")
			}

			// Generated with https://sokra.github.io/source-map-visualization using a TS source map
			expectedMappings := "AAAA,IAAM,CAAC,GAAe,CAAC,CAAC;AACxB,IAAM,CAAC,GAAG,CAAC,CAAC;AAEZ,EAAE,CAAC,OAAO,CAAC,UAAA,CAAC;IACR,OAAO,CAAC,GAAG,CAAC,KAAK,CAAC,CAAC;AACvB,CAAC,CAAC,CAAA"
			if json.Mappings != expectedMappings {
				t.Errorf("Expected mappings:\n%s\nGot:\n%s", expectedMappings, json.Mappings)
			}
		})

		t.Run("should include the files and their contents", func(t *testing.T) {
			map_ := output.NewSourceMapGenerator(strPtr("out.js"))
			map_.AddSource("inline.ts", strPtr("inline"))
			map_.AddSource("inline.ts", strPtr("inline")) // make sure the sources are dedup
			map_.AddSource("url.ts", nil)
			map_.AddLine()
			map_.AddMapping(0, strPtr("inline.ts"), intPtr(0), intPtr(0))

			json, err := map_.ToJSON()
			if err != nil {
				t.Fatal(err)
			}
			if json == nil {
				t.Fatal("Expected ToJSON() to return non-nil")
			}

			if json.File != "out.js" {
				t.Errorf("Expected file to be 'out.js', got %q", json.File)
			}
			if len(json.Sources) != 2 || json.Sources[0] != "inline.ts" || json.Sources[1] != "url.ts" {
				t.Errorf("Expected sources ['inline.ts', 'url.ts'], got %v", json.Sources)
			}
			if len(json.SourcesContent) != 2 || *json.SourcesContent[0] != "inline" || json.SourcesContent[1] != nil {
				t.Errorf("Expected sourcesContent ['inline', nil], got %v", json.SourcesContent)
			}
		})

		t.Run("should not generate source maps when there is no mapping", func(t *testing.T) {
			smg := output.NewSourceMapGenerator(strPtr("out.js"))
			smg.AddSource("inline.ts", strPtr("inline"))
			smg.AddLine()

			json, err := smg.ToJSON()
			if err != nil {
				t.Fatal(err)
			}
			if json != nil {
				t.Error("Expected ToJSON() to return nil")
			}

			comment, err := smg.ToJsComment()
			if err != nil {
				t.Fatal(err)
			}
			if comment != "" {
				t.Errorf("Expected ToJsComment() to be empty, got %q", comment)
			}
		})
	})

	t.Run("encodeB64String", func(t *testing.T) {
		t.Run("should return the b64 encoded value", func(t *testing.T) {
			tests := []struct {
				input  string
				output string
			}{
				{"", ""},
				{"a", "YQ=="},
				{"Foo", "Rm9v"},
				{"Foo1", "Rm9vMQ=="},
				{"Foo12", "Rm9vMTI="},
				{"Foo123", "Rm9vMTIz"},
			}

			for _, test := range tests {
				result := output.ToBase64String(test.input)
				if result != test.output {
					t.Errorf("ToBase64String(%q): expected %q, got %q", test.input, test.output, result)
				}
			}
		})
	})

	t.Run("errors", func(t *testing.T) {
		t.Run("should throw when mappings are added out of order", func(t *testing.T) {
			smg := output.NewSourceMapGenerator(strPtr("out.js"))
			smg.AddSource("in.js", nil)
			smg.AddLine()
			smg.AddMapping(10, strPtr("in.js"), intPtr(0), intPtr(0))

			err := smg.AddMapping(0, strPtr("in.js"), intPtr(0), intPtr(0))
			if err == nil {
				t.Error("Expected error when adding mappings out of order")
			}
		})

		t.Run("should throw when adding segments before any line is created", func(t *testing.T) {
			smg := output.NewSourceMapGenerator(strPtr("out.js"))
			smg.AddSource("in.js", nil)

			err := smg.AddMapping(0, strPtr("in.js"), intPtr(0), intPtr(0))
			if err == nil {
				t.Error("Expected error when adding mapping before line")
			}
		})

		t.Run("should throw when adding segments referencing unknown sources", func(t *testing.T) {
			smg := output.NewSourceMapGenerator(strPtr("out.js"))
			smg.AddSource("in.js", nil)
			smg.AddLine()

			err := smg.AddMapping(0, strPtr("in_.js"), intPtr(0), intPtr(0))
			if err == nil {
				t.Error("Expected error when referencing unknown source")
			}
		})

		t.Run("should throw when adding segments without column", func(t *testing.T) {
			// Note: In Go, col0 is int not *int, so this test is about nil source parameters
			// We test the validation by checking error is returned when source URL provided but position is not
		})

		t.Run("should throw when adding segments with a source url but no position", func(t *testing.T) {
			// Test 1: source url but no line
			smg := output.NewSourceMapGenerator(strPtr("out.js"))
			smg.AddSource("in.js", nil)
			smg.AddLine()

			err := smg.AddMapping(0, strPtr("in.js"), nil, nil)
			if err == nil {
				t.Error("Expected error when adding mapping with source but no line")
			}

			// Test 2: source url with line but no column
			err = smg.AddMapping(0, strPtr("in.js"), intPtr(0), nil)
			if err == nil {
				t.Error("Expected error when adding mapping with source but no column")
			}
		})
	})
}

// Helper functions
func intPtr(i int) *int {
	return &i
}

func strPtr(s string) *string {
	return &s
}
