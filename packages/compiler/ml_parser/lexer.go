package ml_parser

import (
	"ngc-go/packages/compiler/util"
	"strings"
)

// CharacterCursor interface for moving through input text
type CharacterCursor interface {
	Init()
	Peek() int
	Advance()
	GetSpan(start CharacterCursor, leadingTriviaCodePoints []int) *util.ParseSourceSpan
	GetChars(start CharacterCursor) string
	CharsLeft() int
	Diff(other CharacterCursor) int
	Clone() CharacterCursor
}

// CursorState represents the state of a character cursor
type CursorState struct {
	Peek   int
	Offset int
	Line   int
	Column int
}

// PlainCharacterCursor implements CharacterCursor for plain text
type PlainCharacterCursor struct {
	state CursorState
	file  *util.ParseSourceFile
	input string
	end   int
}

// NewPlainCharacterCursor creates a new PlainCharacterCursor
func NewPlainCharacterCursor(file *util.ParseSourceFile, range_ *LexerRange) *PlainCharacterCursor {
	return &PlainCharacterCursor{
		file:  file,
		input: file.Content,
		end:   range_.EndPos,
		state: CursorState{
			Peek:   -1,
			Offset: range_.StartPos,
			Line:   range_.StartLine,
			Column: range_.StartCol,
		},
	}
}

// Clone creates a copy of the cursor
func (p *PlainCharacterCursor) Clone() CharacterCursor {
	return &PlainCharacterCursor{
		file:  p.file,
		input: p.input,
		end:   p.end,
		state: CursorState{
			Peek:   p.state.Peek,
			Offset: p.state.Offset,
			Line:   p.state.Line,
			Column: p.state.Column,
		},
	}
}

// Init initializes the cursor
func (p *PlainCharacterCursor) Init() {
	p.updatePeek(&p.state)
}

// Peek returns the current character
func (p *PlainCharacterCursor) Peek() int {
	return p.state.Peek
}

// Advance advances the cursor by one character
func (p *PlainCharacterCursor) Advance() {
	p.advanceState(&p.state)
}

// GetSpan returns a span from start to current position
func (p *PlainCharacterCursor) GetSpan(start CharacterCursor, leadingTriviaCodePoints []int) *util.ParseSourceSpan {
	if start == nil {
		start = p
	}
	fullStart := start

	if leadingTriviaCodePoints != nil {
		startCursor := start.(*PlainCharacterCursor)
		for p.Diff(start) > 0 {
			peek := startCursor.Peek()
			found := false
			for _, codePoint := range leadingTriviaCodePoints {
				if peek == codePoint {
					found = true
					break
				}
			}
			if !found {
				break
			}
			if fullStart == start {
				start = start.Clone()
			}
			start.Advance()
		}
	}

	startCursor := start.(*PlainCharacterCursor)
	endCursor := p
	startLocation := util.NewParseLocation(
		startCursor.file,
		startCursor.state.Offset,
		startCursor.state.Line,
		startCursor.state.Column,
	)
	endLocation := util.NewParseLocation(
		endCursor.file,
		endCursor.state.Offset,
		endCursor.state.Line,
		endCursor.state.Column,
	)

	var fullStartLocation *util.ParseLocation
	if fullStart != start {
		fullStartCursor := fullStart.(*PlainCharacterCursor)
		fullStartLocation = util.NewParseLocation(
			fullStartCursor.file,
			fullStartCursor.state.Offset,
			fullStartCursor.state.Line,
			fullStartCursor.state.Column,
		)
	} else {
		fullStartLocation = startLocation
	}

	return util.NewParseSourceSpan(startLocation, endLocation, fullStartLocation, nil)
}

// GetChars returns characters from start to current position
func (p *PlainCharacterCursor) GetChars(start CharacterCursor) string {
	startCursor := start.(*PlainCharacterCursor)
	return p.input[startCursor.state.Offset:p.state.Offset]
}

// CharsLeft returns the number of characters left
func (p *PlainCharacterCursor) CharsLeft() int {
	return p.end - p.state.Offset
}

// Diff returns the difference between this cursor and another
func (p *PlainCharacterCursor) Diff(other CharacterCursor) int {
	otherCursor := other.(*PlainCharacterCursor)
	return p.state.Offset - otherCursor.state.Offset
}

// charAt returns the character at a position
func (p *PlainCharacterCursor) charAt(pos int) int {
	if pos >= len(p.input) {
		return 0 // EOF
	}
	return int(p.input[pos])
}

// advanceState advances the cursor state
func (p *PlainCharacterCursor) advanceState(state *CursorState) {
	if state.Offset >= p.end {
		panic("Unexpected character \"EOF\"")
	}
	currentChar := p.charAt(state.Offset)
	if currentChar == '\n' {
		state.Line++
		state.Column = 0
	} else {
		state.Column++
	}
	state.Offset++
	p.updatePeek(state)
}

// updatePeek updates the peek character
func (p *PlainCharacterCursor) updatePeek(state *CursorState) {
	if state.Offset >= p.end {
		state.Peek = 0 // EOF
	} else {
		state.Peek = p.charAt(state.Offset)
	}
}

// EscapedCharacterCursor implements CharacterCursor for escaped strings
type EscapedCharacterCursor struct {
	*PlainCharacterCursor
	internalState CursorState
}

// NewEscapedCharacterCursor creates a new EscapedCharacterCursor
func NewEscapedCharacterCursor(file *util.ParseSourceFile, range_ *LexerRange) *EscapedCharacterCursor {
	plain := NewPlainCharacterCursor(file, range_)
	return &EscapedCharacterCursor{
		PlainCharacterCursor: plain,
		internalState:        plain.state,
	}
}

// Clone creates a copy of the cursor
func (e *EscapedCharacterCursor) Clone() CharacterCursor {
	return &EscapedCharacterCursor{
		PlainCharacterCursor: e.PlainCharacterCursor.Clone().(*PlainCharacterCursor),
		internalState:        e.internalState,
	}
}

// Advance advances the cursor, processing escape sequences
func (e *EscapedCharacterCursor) Advance() {
	e.state = e.internalState
	e.PlainCharacterCursor.Advance()
	e.processEscapeSequence()
}

// Init initializes the cursor
func (e *EscapedCharacterCursor) Init() {
	e.PlainCharacterCursor.Init()
	e.processEscapeSequence()
}

// GetChars returns characters from start to current position
func (e *EscapedCharacterCursor) GetChars(start CharacterCursor) string {
	startCursor := start.(*EscapedCharacterCursor)
	var chars strings.Builder
	cursor := startCursor.Clone().(*EscapedCharacterCursor)
	for cursor.internalState.Offset < e.internalState.Offset {
		chars.WriteRune(rune(cursor.Peek()))
		cursor.Advance()
	}
	return chars.String()
}

// processEscapeSequence processes escape sequences
func (e *EscapedCharacterCursor) processEscapeSequence() {
	peek := e.internalState.Peek

	if peek == '\\' {
		// We have hit an escape sequence
		e.internalState = e.state

		// Move past the backslash
		e.advanceState(&e.internalState)

		peek = e.internalState.Peek

		// Check for standard control char sequences
		switch peek {
		case 'n':
			e.state.Peek = '\n'
		case 'r':
			e.state.Peek = '\r'
		case 'v':
			e.state.Peek = '\v'
		case 't':
			e.state.Peek = '\t'
		case 'b':
			e.state.Peek = '\b'
		case 'f':
			e.state.Peek = '\f'
		case 'u':
			// Unicode code-point sequence
			e.advanceState(&e.internalState)
			if e.internalState.Peek == '{' {
				// Variable length Unicode, e.g. `\x{123}`
				e.advanceState(&e.internalState)
				digitStart := e.Clone().(*EscapedCharacterCursor)
				length := 0
				for e.internalState.Peek != '}' {
					e.advanceState(&e.internalState)
					length++
				}
				e.state.Peek = e.decodeHexDigits(digitStart, length)
			} else {
				// Fixed length Unicode, e.g. `\u1234`
				digitStart := e.Clone().(*EscapedCharacterCursor)
				e.advanceState(&e.internalState)
				e.advanceState(&e.internalState)
				e.advanceState(&e.internalState)
				e.state.Peek = e.decodeHexDigits(digitStart, 4)
			}
		case 'x':
			// Hex char code, e.g. `\x2F`
			e.advanceState(&e.internalState)
			digitStart := e.Clone().(*EscapedCharacterCursor)
			e.advanceState(&e.internalState)
			e.state.Peek = e.decodeHexDigits(digitStart, 2)
		default:
			// Octal or normal character
			if isOctalDigit(peek) {
				octal := ""
				length := 0
				previous := e.Clone().(*EscapedCharacterCursor)
				for isOctalDigit(e.internalState.Peek) && length < 3 {
					previous = e.Clone().(*EscapedCharacterCursor)
					octal += string(rune(e.internalState.Peek))
					e.advanceState(&e.internalState)
					length++
				}
				// Parse octal
				var codePoint int
				for i, ch := range octal {
					codePoint = codePoint*8 + int(ch-'0')
					_ = i
				}
				e.state.Peek = codePoint
				e.internalState = previous.internalState
			} else if isNewLine(e.internalState.Peek) {
				// Line continuation `\` followed by a new line
				e.advanceState(&e.internalState)
				e.state = e.internalState
			} else {
				// Escaped normal character
				e.state.Peek = e.internalState.Peek
			}
		}
	}
}

// decodeHexDigits decodes hexadecimal digits
func (e *EscapedCharacterCursor) decodeHexDigits(start *EscapedCharacterCursor, length int) int {
	hex := e.input[start.internalState.Offset : start.internalState.Offset+length]
	var codePoint int
	for i, ch := range hex {
		var digit int
		if ch >= '0' && ch <= '9' {
			digit = int(ch - '0')
		} else if ch >= 'a' && ch <= 'f' {
			digit = int(ch-'a') + 10
		} else if ch >= 'A' && ch <= 'F' {
			digit = int(ch-'A') + 10
		} else {
			panic("Invalid hexadecimal escape sequence")
		}
		codePoint = codePoint*16 + digit
		_ = i
	}
	return codePoint
}

// CursorError represents a cursor error
type CursorError struct {
	Msg    string
	Cursor CharacterCursor
}

// Error implements the error interface
func (c *CursorError) Error() string {
	return c.Msg
}

// Helper functions from chars package
func isNewLine(code int) bool {
	return code == '\n' || code == '\r'
}

func isOctalDigit(code int) bool {
	return code >= '0' && code <= '7'
}

// Tokenizer tokenizes HTML/XML source
type Tokenizer struct {
	cursor                         CharacterCursor
	tokenizeIcu                    bool
	leadingTriviaCodePoints        []int
	currentTokenStart              CharacterCursor
	currentTokenType               TokenType
	expansionCaseStack             []TokenType
	openDirectiveCount             int
	inInterpolation                bool
	preserveLineEndings            bool
	i18nNormalizeLineEndingsInICUs bool
	tokenizeBlocks                 bool
	tokenizeLet                    bool
	selectorlessEnabled            bool
	tokens                         []Token
	errors                         []*util.ParseError
	nonNormalizedIcuExpressions    []Token
	getTagDefinition               func(tagName string) TagDefinition
}

// NewTokenizer creates a new Tokenizer
func NewTokenizer(file *util.ParseSourceFile, getTagDefinition func(tagName string) TagDefinition, options *TokenizeOptions) *Tokenizer {
	tokenizeIcu := false
	if options != nil && options.TokenizeExpansionForms != nil {
		tokenizeIcu = *options.TokenizeExpansionForms
	}

	var leadingTriviaCodePoints []int
	if options != nil && options.LeadingTriviaChars != nil {
		leadingTriviaCodePoints = make([]int, len(options.LeadingTriviaChars))
		for i, c := range options.LeadingTriviaChars {
			if len(c) > 0 {
				leadingTriviaCodePoints[i] = int(c[0])
			}
		}
	}

	range_ := &LexerRange{
		EndPos:    len(file.Content),
		StartPos:  0,
		StartLine: 0,
		StartCol:  0,
	}
	if options != nil && options.Range != nil {
		range_ = options.Range
	}

	var cursor CharacterCursor
	if options != nil && options.EscapedString != nil && *options.EscapedString {
		cursor = NewEscapedCharacterCursor(file, range_)
	} else {
		cursor = NewPlainCharacterCursor(file, range_)
	}

	preserveLineEndings := false
	if options != nil && options.PreserveLineEndings != nil {
		preserveLineEndings = *options.PreserveLineEndings
	}

	i18nNormalizeLineEndingsInICUs := false
	if options != nil && options.I18nNormalizeLineEndingsInICUs != nil {
		i18nNormalizeLineEndingsInICUs = *options.I18nNormalizeLineEndingsInICUs
	}

	tokenizeBlocks := true
	if options != nil && options.TokenizeBlocks != nil {
		tokenizeBlocks = *options.TokenizeBlocks
	}

	tokenizeLet := true
	if options != nil && options.TokenizeLet != nil {
		tokenizeLet = *options.TokenizeLet
	}

	selectorlessEnabled := false
	if options != nil && options.SelectorlessEnabled != nil {
		selectorlessEnabled = *options.SelectorlessEnabled
	}

	t := &Tokenizer{
		cursor:                         cursor,
		tokenizeIcu:                    tokenizeIcu,
		leadingTriviaCodePoints:        leadingTriviaCodePoints,
		expansionCaseStack:             []TokenType{},
		preserveLineEndings:            preserveLineEndings,
		i18nNormalizeLineEndingsInICUs: i18nNormalizeLineEndingsInICUs,
		tokenizeBlocks:                 tokenizeBlocks,
		tokenizeLet:                    tokenizeLet,
		selectorlessEnabled:            selectorlessEnabled,
		tokens:                         []Token{},
		errors:                         []*util.ParseError{},
		nonNormalizedIcuExpressions:    []Token{},
		getTagDefinition:               getTagDefinition,
	}

	t.cursor.Init()
	return t
}

// Tokenize tokenizes the source
func (t *Tokenizer) Tokenize() {
	// TODO: Implement full tokenization logic
	// This is a placeholder - requires full implementation of all tokenization methods
	for t.cursor.Peek() != 0 { // EOF
		// Process tokens
		// This will be implemented with all the consume methods
		break
	}

	// Add EOF token
	t.beginToken(TokenTypeEOF)
	t.endToken([]string{})
}

// beginToken begins a new token
func (t *Tokenizer) beginToken(tokenType TokenType) {
	t.currentTokenStart = t.cursor.Clone()
	t.currentTokenType = tokenType
}

// endToken ends the current token
func (t *Tokenizer) endToken(parts []string) Token {
	if t.currentTokenStart == nil {
		panic("Programming error - attempted to end a token when there was no start to the token")
	}
	if t.currentTokenType == -1 {
		panic("Programming error - attempted to end a token which has no token type")
	}

	sourceSpan := t.cursor.GetSpan(t.currentTokenStart, t.leadingTriviaCodePoints)
	token := NewTokenBase(t.currentTokenType, parts, sourceSpan)
	t.tokens = append(t.tokens, token)

	t.currentTokenStart = nil
	t.currentTokenType = -1

	return token
}

// Update Tokenize function to use the Tokenizer
func init() {
	// This will be called when the package is initialized
}
