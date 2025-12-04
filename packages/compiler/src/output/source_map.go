package output

import (
	"encoding/json"
	"fmt"
	"ngc-go/packages/compiler/src/util"
	"sort"
	"strings"
)

const (
	// Version is the source map version
	Version     = 3
	jsB64Prefix = "# sourceMappingURL=data:application/json;base64,"
)

// Segment represents a segment in a source map line
type Segment struct {
	Col0        int
	SourceURL   *string
	SourceLine0 *int
	SourceCol0  *int
}

// SourceMap represents a source map
type SourceMap struct {
	Version        int
	File           string
	SourceRoot     string
	Sources        []string
	SourcesContent []*string // null is represented as nil
	Mappings       string
}

// SourceMapGenerator generates source maps
type SourceMapGenerator struct {
	sourcesContent map[string]*string // null is represented as nil
	lines          [][]Segment
	lastCol0       int
	hasMappings    bool
	file           *string
}

// NewSourceMapGenerator creates a new SourceMapGenerator
func NewSourceMapGenerator(file *string) *SourceMapGenerator {
	return &SourceMapGenerator{
		sourcesContent: make(map[string]*string),
		lines:          [][]Segment{},
		lastCol0:       0,
		hasMappings:    false,
		file:           file,
	}
}

// AddSource adds a source file to the source map
// The content is `nil` when the content is expected to be loaded using the URL
func (smg *SourceMapGenerator) AddSource(url string, content *string) *SourceMapGenerator {
	if _, exists := smg.sourcesContent[url]; !exists {
		smg.sourcesContent[url] = content
	}
	return smg
}

// AddLine adds a new line to the source map
func (smg *SourceMapGenerator) AddLine() *SourceMapGenerator {
	smg.lines = append(smg.lines, []Segment{})
	smg.lastCol0 = 0
	return smg
}

// AddMapping adds a mapping to the current line
func (smg *SourceMapGenerator) AddMapping(col0 int, sourceURL *string, sourceLine0 *int, sourceCol0 *int) error {
	currentLine := smg.currentLine()
	if currentLine == nil {
		return fmt.Errorf("a line must be added before mappings can be added")
	}
	if sourceURL != nil {
		if _, exists := smg.sourcesContent[*sourceURL]; !exists {
			return fmt.Errorf("unknown source file \"%s\"", *sourceURL)
		}
	}
	if col0 < smg.lastCol0 {
		return fmt.Errorf("mapping should be added in output order")
	}
	if sourceURL != nil && (sourceLine0 == nil || sourceCol0 == nil) {
		return fmt.Errorf("the source location must be provided when a source url is provided")
	}

	smg.hasMappings = true
	smg.lastCol0 = col0
	*currentLine = append(*currentLine, Segment{
		Col0:        col0,
		SourceURL:   sourceURL,
		SourceLine0: sourceLine0,
		SourceCol0:  sourceCol0,
	})
	return nil
}

// currentLine returns the current line being built
func (smg *SourceMapGenerator) currentLine() *[]Segment {
	if len(smg.lines) == 0 {
		return nil
	}
	return &smg.lines[len(smg.lines)-1]
}

// ToJSON converts the source map to JSON format
func (smg *SourceMapGenerator) ToJSON() (*SourceMap, error) {
	if !smg.hasMappings {
		return nil, nil
	}

	sourcesIndex := make(map[string]int)
	sources := []string{}
	sourcesContent := []*string{}

	// Collect and sort sources for deterministic order
	for url := range smg.sourcesContent {
		sources = append(sources, url)
	}
	sort.Strings(sources)

	// Build index and content arrays in sorted order
	for i, url := range sources {
		sourcesIndex[url] = i
		content := smg.sourcesContent[url]
		sourcesContent = append(sourcesContent, content)
	}

	mappings := ""
	lastCol0 := 0
	lastSourceIndex := 0
	lastSourceLine0 := 0
	lastSourceCol0 := 0

	segmentStrs := []string{}
	for _, segments := range smg.lines {
		lastCol0 = 0
		lineSegments := []string{}

		for _, segment := range segments {
			// zero-based starting column of the line in the generated code
			segAsStr := toBase64VLQ(segment.Col0 - lastCol0)
			lastCol0 = segment.Col0

			if segment.SourceURL != nil {
				// zero-based index into the "sources" list
				sourceIndex := sourcesIndex[*segment.SourceURL]
				segAsStr += toBase64VLQ(sourceIndex - lastSourceIndex)
				lastSourceIndex = sourceIndex
				// the zero-based starting line in the original source
				segAsStr += toBase64VLQ(*segment.SourceLine0 - lastSourceLine0)
				lastSourceLine0 = *segment.SourceLine0
				// the zero-based starting column in the original source
				segAsStr += toBase64VLQ(*segment.SourceCol0 - lastSourceCol0)
				lastSourceCol0 = *segment.SourceCol0
			}

			lineSegments = append(lineSegments, segAsStr)
		}
		segmentStrs = append(segmentStrs, strings.Join(lineSegments, ","))
	}

	mappings = strings.Join(segmentStrs, ";")

	file := ""
	if smg.file != nil {
		file = *smg.file
	}

	return &SourceMap{
		Version:        Version,
		File:           file,
		SourceRoot:     "",
		Sources:        sources,
		SourcesContent: sourcesContent,
		Mappings:       mappings,
	}, nil
}

// ToJsComment converts the source map to a JavaScript comment
func (smg *SourceMapGenerator) ToJsComment() (string, error) {
	if !smg.hasMappings {
		return "", nil
	}

	sourceMap, err := smg.ToJSON()
	if err != nil {
		return "", err
	}

	jsonBytes, err := json.Marshal(sourceMap)
	if err != nil {
		return "", err
	}

	b64 := ToBase64String(string(jsonBytes))
	return "//" + jsB64Prefix + b64, nil
}

// ToBase64String converts a string to base64
func ToBase64String(value string) string {
	encoded := util.UTF8Encode(value)
	b64 := ""

	for i := 0; i < len(encoded); {
		i1 := int(encoded[i])
		i++
		var i2 *int
		var i3 *int
		if i < len(encoded) {
			val := int(encoded[i])
			i2 = &val
			i++
		}
		if i < len(encoded) {
			val := int(encoded[i])
			i3 = &val
			i++
		}

		b64 += string(toBase64Digit(i1 >> 2))
		i2Val := 0
		if i2 != nil {
			i2Val = *i2
		}
		b64 += string(toBase64Digit(((i1 & 3) << 4) | (i2Val >> 4)))
		if i2 == nil {
			b64 += "="
		} else {
			i3Val := 0
			if i3 != nil {
				i3Val = *i3
			}
			b64 += string(toBase64Digit(((i2Val & 15) << 2) | (i3Val >> 6)))
		}
		if i2 == nil || i3 == nil {
			b64 += "="
		} else {
			b64 += string(toBase64Digit(*i3 & 63))
		}
	}

	return b64
}

// toBase64VLQ converts a number to base64 VLQ encoding
func toBase64VLQ(value int) string {
	if value < 0 {
		value = (-value << 1) + 1
	} else {
		value = value << 1
	}

	out := ""
	for {
		digit := value & 31
		value = value >> 5
		if value > 0 {
			digit = digit | 32
		}
		out += string(toBase64Digit(digit))
		if value == 0 {
			break
		}
	}

	return out
}

const b64Digits = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

// toBase64Digit converts a value to a base64 digit
func toBase64Digit(value int) byte {
	if value < 0 || value >= 64 {
		panic(fmt.Sprintf("can only encode value in the range [0, 63], got %d", value))
	}
	return b64Digits[value]
}
