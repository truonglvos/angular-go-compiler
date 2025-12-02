package util

import (
	"fmt"
	"regexp"
	"strings"

	"ngc-go/packages/compiler/src/core"
)

// ParseLocation represents a location in the source file
type ParseLocation struct {
	File   *ParseSourceFile
	Offset int
	Line   int
	Col    int
}

// NewParseLocation creates a new ParseLocation
func NewParseLocation(file *ParseSourceFile, offset, line, col int) *ParseLocation {
	return &ParseLocation{
		File:   file,
		Offset: offset,
		Line:   line,
		Col:    col,
	}
}

// String returns a string representation of the location
func (p *ParseLocation) String() string {
	if p.Offset >= 0 {
		return fmt.Sprintf("%s@%d:%d", p.File.URL, p.Line, p.Col)
	}
	return p.File.URL
}

// MoveBy moves the location by delta characters
func (p *ParseLocation) MoveBy(delta int) *ParseLocation {
	source := p.File.Content
	len := len(source)
	offset := p.Offset
	line := p.Line
	col := p.Col

	for offset > 0 && delta < 0 {
		offset--
		delta++
		ch := source[offset]
		if ch == '\n' {
			line--
			priorLine := strings.LastIndex(source[:offset], "\n")
			if priorLine > 0 {
				col = offset - priorLine
			} else {
				col = offset
			}
		} else {
			col--
		}
	}

	for offset < len && delta > 0 {
		ch := source[offset]
		offset++
		delta--
		if ch == '\n' {
			line++
			col = 0
		} else {
			col++
		}
	}

	return NewParseLocation(p.File, offset, line, col)
}

// GetContext returns the source context around the location
func (p *ParseLocation) GetContext(maxChars, maxLines int) *Context {
	content := p.File.Content
	startOffset := p.Offset

	// Check if offset is valid (similar to TypeScript's != null check)
	if startOffset < 0 {
		return nil
	}

	if startOffset > len(content)-1 {
		startOffset = len(content) - 1
	}

	endOffset := startOffset
	ctxChars := 0
	ctxLines := 0

	for ctxChars < maxChars && startOffset > 0 {
		startOffset--
		ctxChars++
		if content[startOffset] == '\n' {
			ctxLines++
			if ctxLines == maxLines {
				break
			}
		}
	}

	ctxChars = 0
	ctxLines = 0
	for ctxChars < maxChars && endOffset < len(content)-1 {
		endOffset++
		ctxChars++
		if content[endOffset] == '\n' {
			ctxLines++
			if ctxLines == maxLines {
				break
			}
		}
	}

	return &Context{
		Before: content[startOffset:p.Offset],
		After:  content[p.Offset : endOffset+1],
	}
}

// Context represents source context around a location
type Context struct {
	Before string
	After  string
}

// ParseSourceFile represents a source file
type ParseSourceFile struct {
	Content string
	URL     string
}

// NewParseSourceFile creates a new ParseSourceFile
func NewParseSourceFile(content, url string) *ParseSourceFile {
	return &ParseSourceFile{
		Content: content,
		URL:     url,
	}
}

// ParseSourceSpan represents a span of source code
type ParseSourceSpan struct {
	Start     *ParseLocation
	End       *ParseLocation
	FullStart *ParseLocation
	Details   *string
}

// NewParseSourceSpan creates a new ParseSourceSpan
func NewParseSourceSpan(start, end *ParseLocation, fullStart *ParseLocation, details *string) *ParseSourceSpan {
	if fullStart == nil {
		fullStart = start
	}
	return &ParseSourceSpan{
		Start:     start,
		End:       end,
		FullStart: fullStart,
		Details:   details,
	}
}

// String returns the source code in this span
func (p *ParseSourceSpan) String() string {
	return p.Start.File.Content[p.Start.Offset:p.End.Offset]
}

// ParseErrorLevel represents the level of a parse error
type ParseErrorLevel int

const (
	ParseErrorLevelWarning ParseErrorLevel = iota
	ParseErrorLevelError
)

// ParseError represents a parse error
type ParseError struct {
	Span         *ParseSourceSpan
	Msg          string
	Level        ParseErrorLevel
	RelatedError error
}

// NewParseError creates a new ParseError
func NewParseError(span *ParseSourceSpan, msg string) *ParseError {
	return &ParseError{
		Span:  span,
		Msg:   msg,
		Level: ParseErrorLevelError,
	}
}

// NewParseWarning creates a new ParseWarning
func NewParseWarning(span *ParseSourceSpan, msg string) *ParseError {
	return &ParseError{
		Span:  span,
		Msg:   msg,
		Level: ParseErrorLevelWarning,
	}
}

// Error implements the error interface
func (p *ParseError) Error() string {
	return p.String()
}

// ContextualMessage returns the error message with context
func (p *ParseError) ContextualMessage() string {
	if p.Span == nil || p.Span.Start == nil {
		return p.Msg
	}
	ctx := p.Span.Start.GetContext(100, 3)
	if ctx != nil {
		levelStr := "ERROR"
		if p.Level == ParseErrorLevelWarning {
			levelStr = "WARNING"
		}
		return fmt.Sprintf(`%s ("%s[%s ->]%s")`, p.Msg, ctx.Before, levelStr, ctx.After)
	}
	return p.Msg
}

// String returns a string representation of the error
func (p *ParseError) String() string {
	if p.Span == nil {
		return p.Msg
	}
	details := ""
	if p.Span.Details != nil {
		details = fmt.Sprintf(", %s", *p.Span.Details)
	}
	if p.Span.Start == nil {
		return fmt.Sprintf("%s%s", p.ContextualMessage(), details)
	}
	return fmt.Sprintf("%s: %s%s", p.ContextualMessage(), p.Span.Start, details)
}

// IsWhitespace checks if a character is whitespace
func IsWhitespace(ch int) bool {
	return core.IsWhitespace(ch)
}

// IsDigit checks if a character is a digit
func IsDigit(ch int) bool {
	return core.IsDigit(ch)
}

// IsAsciiLetter checks if a character is an ASCII letter
func IsAsciiLetter(ch int) bool {
	return core.IsAsciiLetter(ch)
}

// IsAsciiHexDigit checks if a character is an ASCII hex digit
func IsAsciiHexDigit(ch int) bool {
	return core.IsAsciiHexDigit(ch)
}

// IsNewLine checks if a character is a newline
func IsNewLine(ch int) bool {
	return core.IsNewLine(ch)
}

// IsOctalDigit checks if a character is an octal digit
func IsOctalDigit(ch int) bool {
	return core.IsOctalDigit(ch)
}

// IsQuote checks if a character is a quote
func IsQuote(ch int) bool {
	return core.IsQuote(ch)
}

// R3JitTypeSourceSpan generates Source Span object for a given R3 Type for JIT mode
func R3JitTypeSourceSpan(kind string, typeName string, sourceUrl string) *ParseSourceSpan {
	sourceFileName := fmt.Sprintf("in %s %s in %s", kind, typeName, sourceUrl)
	sourceFile := NewParseSourceFile("", sourceFileName)
	return NewParseSourceSpan(
		NewParseLocation(sourceFile, -1, -1, -1),
		NewParseLocation(sourceFile, -1, -1, -1),
		nil,
		nil,
	)
}

// CompileIdentifierMetadata represents metadata for a compile identifier
type CompileIdentifierMetadata struct {
	Reference interface{}
}

var anonymousTypeIndex = 0

// IdentifierName returns the name of an identifier from CompileIdentifierMetadata
func IdentifierName(compileIdentifier *CompileIdentifierMetadata) *string {
	if compileIdentifier == nil || compileIdentifier.Reference == nil {
		return nil
	}

	ref := compileIdentifier.Reference

	// Check for __anonymousType
	if refMap, ok := ref.(map[string]interface{}); ok {
		if anonymousType, ok := refMap["__anonymousType"]; ok {
			if name, ok := anonymousType.(string); ok {
				return &name
			}
		}

		// Check for __forward_ref__
		if _, ok := refMap["__forward_ref__"]; ok {
			forwardRef := "__forward_ref__"
			return &forwardRef
		}
	}

	identifier := Stringify(ref)
	if strings.Contains(identifier, "(") {
		// Case: anonymous functions!
		anonymousTypeIndex++
		anonymousName := fmt.Sprintf("anonymous_%d", anonymousTypeIndex)

		// Store in reference if it's a map
		if refMap, ok := ref.(map[string]interface{}); ok {
			refMap["__anonymousType"] = anonymousName
		}

		return &anonymousName
	}

	sanitized := SanitizeIdentifier(identifier)
	return &sanitized
}

// SanitizeIdentifier sanitizes an identifier name by replacing non-word characters with underscores
func SanitizeIdentifier(name string) string {
	re := regexp.MustCompile(`\W`)
	return re.ReplaceAllString(name, "_")
}
