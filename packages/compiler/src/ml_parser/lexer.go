package ml_parser

import (
	"fmt"
	"ngc-go/packages/compiler/src/core"
	"ngc-go/packages/compiler/src/util"
	"strconv"
	"strings"
	"unicode/utf8"
)

var SUPPORTED_BLOCKS = []string{
	"@if",
	"@else", // Covers `@else if` as well
	"@for",
	"@switch",
	"@case",
	"@default",
	"@empty",
	"@defer",
	"@placeholder",
	"@loading",
	"@error",
}

var INTERPOLATION = struct {
	start string
	end   string
}{
	start: "{{",
	end:   "}}",
}

// Tokenize tokenizes the source
func (t *Tokenizer) Tokenize() {
	defer func() {
		if r := recover(); r != nil {
			t.handleError(r)
		}
	}()
	iterationCount := 0
	for t.cursor.Peek() != core.CharEOF {
		iterationCount++
		if iterationCount > 1000 {
			peekChar := t.cursor.Peek()
			if peekChar >= 32 && peekChar < 127 {
				fmt.Printf("[DEBUG] Tokenize: INFINITE LOOP DETECTED! iterationCount=%d, peek='%c' (%d), CharEOF=%d\n",
					iterationCount, peekChar, peekChar, core.CharEOF)
			} else {
				fmt.Printf("[DEBUG] Tokenize: INFINITE LOOP DETECTED! iterationCount=%d, peek=%d, CharEOF=%d\n",
					iterationCount, peekChar, core.CharEOF)
			}
			break
		}
		if iterationCount <= 20 || iterationCount%100 == 0 {
			peekChar := t.cursor.Peek()
			if peekChar >= 32 && peekChar < 127 {
				fmt.Printf("[DEBUG] Tokenize: iteration=%d, peek='%c' (%d)\n", iterationCount, peekChar, peekChar)
			} else {
				fmt.Printf("[DEBUG] Tokenize: iteration=%d, peek=%d\n", iterationCount, peekChar)
			}
		}
		start := t.cursor.Clone()
		if t._attemptCharCode(core.CharLT) {
			if iterationCount <= 20 {
				fmt.Printf("[DEBUG] Tokenize: found '<', checking next char\n")
			}
			if t._attemptCharCode(core.CharBANG) {
				if iterationCount <= 20 {
					fmt.Printf("[DEBUG] Tokenize: found '<!'\n")
				}
				if t._attemptCharCode(core.CharLBRACKET) {
					if iterationCount <= 20 {
						fmt.Printf("[DEBUG] Tokenize: found '<![', consuming CDATA\n")
					}
					func() {
						defer func() {
							if r := recover(); r != nil {
								t.handleError(r)
							}
						}()
						t._consumeCdata(start)
					}()
				} else if t._attemptCharCode(core.CharMINUS) {
					if iterationCount <= 20 {
						fmt.Printf("[DEBUG] Tokenize: found '<!-', consuming comment\n")
					}
					func() {
						defer func() {
							if r := recover(); r != nil {
								// Advance cursor past the comment to allow tokenization to continue
								for t.cursor.Peek() != core.CharEOF {
									if t._attemptStr("-->") {
										break
									}
									if t.cursor.Peek() == core.CharGT {
										t.cursor.Advance()
										break
									}
									t.cursor.Advance()
								}
								t.handleError(r)
							}
						}()
						t._consumeComment(start)
					}()
				} else {
					if iterationCount <= 20 {
						fmt.Printf("[DEBUG] Tokenize: found '<!', consuming doctype\n")
					}
					func() {
						defer func() {
							if r := recover(); r != nil {
								t.handleError(r)
							}
						}()
						t._consumeDocType(start)
					}()
				}
			} else if t._attemptCharCode(core.CharSLASH) {
				if iterationCount <= 20 {
					fmt.Printf("[DEBUG] Tokenize: found '</', consuming tag close\n")
				}
				t._consumeTagClose(start)
			} else {
				if iterationCount <= 20 {
					fmt.Printf("[DEBUG] Tokenize: found '<', consuming tag open\n")
				}
				tagInfo := t._consumeTagOpen(start)
				if iterationCount <= 20 {
					fmt.Printf("[DEBUG] Tokenize: _consumeTagOpen done, peek=%d\n", t.cursor.Peek())
				}
				// Check if we need to consume raw text (for script, style, title, textarea, etc.)
				if tagInfo != nil && !tagInfo.isSelfClosing {
					prefix := tagInfo.prefix
					tagName := tagInfo.name
					if iterationCount <= 20 {
						fmt.Printf("[DEBUG] Tokenize: checking content type for tag=%s, prefix=%s, isSelfClosing=%v\n", tagName, prefix, tagInfo.isSelfClosing)
					}
					// Use getTagDefinition if provided, otherwise use default HTML tag definitions
					getTagDef := t.getTagDefinition
					if getTagDef == nil {
						getTagDef = GetHtmlTagDefinition
					}
					if getTagDef != nil {
						tagDef := getTagDef(tagName)
						if iterationCount <= 20 {
							fmt.Printf("[DEBUG] Tokenize: tagDef=%v\n", tagDef != nil)
						}
						if tagDef != nil {
							var prefixPtr *string
							if prefix != "" {
								prefixPtr = &prefix
							}
							contentType := tagDef.GetContentType(prefixPtr)
							if iterationCount <= 20 {
								fmt.Printf("[DEBUG] Tokenize: contentType=%v (RAW_TEXT=%v, ESCAPABLE_RAW_TEXT=%v)\n", contentType, TagContentTypeRAW_TEXT, TagContentTypeESCAPABLE_RAW_TEXT)
							}
							// Find the open token (TAG_OPEN_START or COMPONENT_OPEN_START) to pass to _consumeRawTextWithTagClose
							var openToken Token
							for i := len(t.tokens) - 1; i >= 0; i-- {
								token := t.tokens[i]
								if token != nil {
									tokenType := token.Type()
									if tokenType == TokenTypeTAG_OPEN_START || tokenType == TokenTypeCOMPONENT_OPEN_START {
										openToken = token
										break
									}
								}
							}
							if contentType == TagContentTypeRAW_TEXT {
								if iterationCount <= 20 {
									fmt.Printf("[DEBUG] Tokenize: consuming RAW_TEXT for tag=%s\n", tagName)
								}
								t._consumeRawTextWithTagClose(openToken, tagInfo.closingTagName, false)
							} else if contentType == TagContentTypeESCAPABLE_RAW_TEXT {
								if iterationCount <= 20 {
									fmt.Printf("[DEBUG] Tokenize: consuming ESCAPABLE_RAW_TEXT for tag=%s\n", tagName)
								}
								t._consumeRawTextWithTagClose(openToken, tagInfo.closingTagName, true)
							}
						}
					}
				}
			}
		} else if t.tokenizeLet &&
			// Use `peek` instead of `attempCharCode` since we
			// don't want to advance in case it's not `@let`.
			t.cursor.Peek() == core.CharAT &&
			!t.inInterpolation &&
			t._isLetStart() {
			t._consumeLetDeclaration(start)
		} else {
			// Check if we should tokenize expansion forms FIRST (before blocks)
			// This ensures that } in expansion context is tokenized as EXPANSION_*_END, not BLOCK_CLOSE
			peekChar := t.cursor.Peek()
			shouldTokenizeExpansion := t.tokenizeIcu && !t.inInterpolation
			if peekChar == core.CharLBRACE || peekChar == core.CharRBRACE {
				fmt.Printf("[DEBUG] Tokenize: found '%c', checking expansion form, tokenizeIcu=%v, inInterpolation=%v, peek=%d, iteration=%d\n",
					peekChar, t.tokenizeIcu, t.inInterpolation, peekChar, iterationCount)
			}
			expansionFormTokenized := false
			if shouldTokenizeExpansion {
				expansionFormTokenized = t._tokenizeExpansionForm()
				if peekChar == core.CharLBRACE || peekChar == core.CharRBRACE {
					fmt.Printf("[DEBUG] Tokenize: _tokenizeExpansionForm returned %v, peek after=%d\n", expansionFormTokenized, t.cursor.Peek())
				}
			} else if peekChar == core.CharLBRACE || peekChar == core.CharRBRACE {
				fmt.Printf("[DEBUG] Tokenize: NOT checking expansion form (shouldTokenizeExpansion=false), tokenizeIcu=%v, inInterpolation=%v\n",
					t.tokenizeIcu, t.inInterpolation)
			}

			// Only check for block tokens if expansion form was NOT tokenized
			// and we're not in expansion context
			if !expansionFormTokenized {
				if t.tokenizeBlocks && t._isBlockStart() {
					t._consumeBlockStart(start)
					continue
				} else if t.tokenizeBlocks &&
					!t.inInterpolation &&
					!t._isInExpansionCase() &&
					!t._isInExpansionForm() &&
					t._attemptCharCode(core.CharRBRACE) {
					t._consumeBlockEnd(start)
					continue
				}
			}

			if !expansionFormTokenized {
				// In (possibly interpolated) text the end of the text is given by `isTextEnd()`, while
				// the premature end of an interpolation is given by the start of a new HTML element.
				if peekChar == core.CharLBRACE {
					fmt.Printf("[DEBUG] Tokenize: NOT tokenizing expansion form, calling _consumeWithInterpolation, tokenizeIcu=%v, inInterpolation=%v, peek=%d\n",
						t.tokenizeIcu, t.inInterpolation, peekChar)
				}
				t._consumeWithInterpolation(
					TokenTypeTEXT,
					TokenTypeINTERPOLATION,
					func() bool { return t._isTextEnd() },
					func() bool { return t._isTagStart() },
				)
			}
		}
	}

	// Add EOF token
	t._beginToken(TokenTypeEOF, t.cursor.Clone())
	t._endToken([]string{}, nil)
}

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
	// log.Printf("GetSpan: start=%T, p=%T\n", start, p)
	if start == nil {
		start = p
	}
	fullStart := start

	if leadingTriviaCodePoints != nil {
		for p.Diff(start) > 0 {
			peek := start.Peek()
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
	var startPlain *PlainCharacterCursor
	if startEscaped, ok := start.(*EscapedCharacterCursor); ok {
		startPlain = startEscaped.PlainCharacterCursor
	} else if startP, ok := start.(*PlainCharacterCursor); ok {
		startPlain = startP
	} else {
		panic(fmt.Sprintf("Unexpected cursor type: %T", start))
	}
	return p.input[startPlain.state.Offset:p.state.Offset]
}

// CharsLeft returns the number of characters left
func (p *PlainCharacterCursor) CharsLeft() int {
	return p.end - p.state.Offset
}

// Diff returns the difference between this cursor and another
func (p *PlainCharacterCursor) Diff(other CharacterCursor) int {
	var otherPlain *PlainCharacterCursor
	if otherEscaped, ok := other.(*EscapedCharacterCursor); ok {
		otherPlain = otherEscaped.PlainCharacterCursor
	} else if otherP, ok := other.(*PlainCharacterCursor); ok {
		otherPlain = otherP
	} else {
		panic(fmt.Sprintf("Unexpected cursor type: %T", other))
	}
	return p.state.Offset - otherPlain.state.Offset
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
		// LF: treat as newline
		state.Line++
		state.Column = 0
		state.Offset++
		p.updatePeek(state)
	} else if currentChar == '\r' {
		// Check if next character is '\n' (CRLF)
		if state.Offset+1 < p.end && p.charAt(state.Offset+1) == '\n' {
			// CRLF: advance past both characters, treat as single newline
			state.Line++
			state.Column = 0
			state.Offset += 2 // Skip both \r and \n
		} else {
			// Standalone CR: don't increment column (matches TypeScript behavior)
			// TypeScript's advanceState only increments column if !isNewLine(currentChar)
			// and CR is considered a newline character, so it doesn't increment column
			state.Offset++
		}
		p.updatePeek(state)
	} else if core.IsNewLine(currentChar) {
		// Other newline characters: don't increment column
		state.Offset++
		p.updatePeek(state)
	} else {
		// Increment column for all non-newline characters
		state.Column++
		state.Offset++
		p.updatePeek(state)
	}
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
	// Copy internalState to state before advancing
	e.state = e.internalState
	// Advance the plain cursor (this updates PlainCharacterCursor.state)
	e.PlainCharacterCursor.Advance()
	// Update internalState to match PlainCharacterCursor.state after advance
	e.internalState = e.PlainCharacterCursor.state
	// Process escape sequences (this may modify e.state.Peek)
	e.processEscapeSequence()
}

// Init initializes the cursor
func (e *EscapedCharacterCursor) Init() {
	e.PlainCharacterCursor.Init()
	e.internalState = e.PlainCharacterCursor.state
	e.processEscapeSequence()
}

// GetSpan returns a span from start to current position
func (e *EscapedCharacterCursor) GetSpan(start CharacterCursor, leadingTriviaCodePoints []int) *util.ParseSourceSpan {
	var startPlain *PlainCharacterCursor
	if start == nil {
		startPlain = e.PlainCharacterCursor
	} else if startEscaped, ok := start.(*EscapedCharacterCursor); ok {
		startPlain = startEscaped.PlainCharacterCursor
	} else if startP, ok := start.(*PlainCharacterCursor); ok {
		startPlain = startP
	} else {
		panic(fmt.Sprintf("Unexpected cursor type: %T", start))
	}
	return e.PlainCharacterCursor.GetSpan(startPlain, leadingTriviaCodePoints)
}

// GetChars returns characters from start to current position
func (e *EscapedCharacterCursor) GetChars(start CharacterCursor) string {
	startCursor := start.(*EscapedCharacterCursor)
	// Get characters directly from input based on offset, not from Peek()
	// This preserves CRLF sequences that may have been normalized in cursor state
	return e.input[startCursor.internalState.Offset:e.internalState.Offset]
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
		leadingTriviaCodePoints = make([]int, 0, len(options.LeadingTriviaChars))
		for _, c := range options.LeadingTriviaChars {
			if len(c) > 0 {
				// Use codePointAt(0) to match TypeScript behavior
				r, _ := utf8.DecodeRuneInString(c)
				leadingTriviaCodePoints = append(leadingTriviaCodePoints, int(r))
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

func (t *Tokenizer) _consumeCdata(start CharacterCursor) {
	t._beginToken(TokenTypeCDATA_START, start)
	t._requireStr("CDATA[")
	t._endToken([]string{}, nil)
	t._consumeRawText(false, func() bool { return t._attemptStr("]]>") })
	t._beginToken(TokenTypeCDATA_END, nil)
	t._requireStr("]]>")
	t._endToken([]string{}, nil)
}

func (t *Tokenizer) _consumeComment(start CharacterCursor) {
	defer func() {
		if r := recover(); r != nil {
			// If we error while consuming comment, advance cursor past the comment start
			// to avoid infinite loop and allow tokenization to continue
			// Try to find the end of the comment by looking for '-->' or '>'
			for t.cursor.Peek() != core.CharEOF {
				if t._attemptStr("-->") {
					break
				}
				if t.cursor.Peek() == core.CharGT {
					t.cursor.Advance()
					break
				}
				t.cursor.Advance()
			}
			// Re-panic to let Tokenize() handle the error
			panic(r)
		}
	}()
	t._beginToken(TokenTypeCOMMENT_START, start)
	t._requireCharCode(core.CharMINUS)
	t._endToken([]string{}, nil)
	t._consumeRawText(false, func() bool { return t._attemptStr("-->") })
	t._beginToken(TokenTypeCOMMENT_END, nil)
	t._requireStr("-->")
	t._endToken([]string{}, nil)
}

func (t *Tokenizer) _consumeDocType(start CharacterCursor) {
	t._beginToken(TokenTypeDOC_TYPE, start)
	contentStart := t.cursor.Clone()
	t._attemptUntilChar(core.CharGT)
	content := t.cursor.GetChars(contentStart)
	t.cursor.Advance()
	t._endToken([]string{content}, nil)
}

func (t *Tokenizer) _consumeTagClose(start CharacterCursor) {
	if t.selectorlessEnabled {
		// Check if this is a component close tag
		clone := start.Clone()
		for clone.Peek() != core.CharGT && !isSelectorlessNameStart(clone.Peek()) {
			clone.Advance()
		}
		if isSelectorlessNameStart(clone.Peek()) {
			// This is a component close tag
			t._beginToken(TokenTypeCOMPONENT_CLOSE, start)
			parts := t._consumeComponentName()
			t._attemptCharCodeUntilFn(isNotWhitespace)
			t._requireCharCode(core.CharGT)
			t._endToken(parts, nil)
			return
		}
	}

	// Regular tag close
	t._beginToken(TokenTypeTAG_CLOSE, start)
	t._attemptCharCodeUntilFn(isNotWhitespace)
	// Matches TypeScript: _consumePrefixAndName(isNameEnd) for tag close
	prefixAndName := t._consumePrefixAndName(isNameEnd)
	t._attemptCharCodeUntilFn(isNotWhitespace)
	t._requireCharCode(core.CharGT)
	// End token after consuming '>' so source span includes it
	t._endToken(prefixAndName, nil)
}

func (t *Tokenizer) _consumeAttributesAndDirectives() {
	fmt.Printf("[DEBUG] lexer._consumeAttributesAndDirectives: START, peek=%d\n", t.cursor.Peek())
	attrIterationCount := 0
	for !isAttributeTerminator(t.cursor.Peek()) {
		attrIterationCount++
		if attrIterationCount > 1000 {
			fmt.Printf("[DEBUG] lexer._consumeAttributesAndDirectives: INFINITE LOOP DETECTED! iterationCount=%d, peek=%d\n",
				attrIterationCount, t.cursor.Peek())
			break
		}
		if attrIterationCount <= 20 || attrIterationCount%100 == 0 {
			fmt.Printf("[DEBUG] lexer._consumeAttributesAndDirectives: iteration=%d, peek=%d\n", attrIterationCount, t.cursor.Peek())
		}
		t._attemptCharCodeUntilFn(isNotWhitespace)
		if isAttributeTerminator(t.cursor.Peek()) {
			fmt.Printf("[DEBUG] lexer._consumeAttributesAndDirectives: found end char, breaking\n")
			break
		}
		// If we encounter a quote that's not part of an attribute value (no '=' before it),
		// stop consuming attributes and let the tag close consume it as text
		// (e.g., `<t a='b' '>` - the second quote should be treated as text)
		if t.cursor.Peek() == core.CharSQ || t.cursor.Peek() == core.CharDQ {
			fmt.Printf("[DEBUG] lexer._consumeAttributesAndDirectives: found quote, stopping attribute consumption\n")
			break
		}
		// Check if next char is '=' - if so, this means we're in the middle of an attribute value
		// and shouldn't create a new attribute name
		if t.cursor.Peek() == core.CharEQ {
			// Skip '=' and whitespace, then consume the value
			t.cursor.Advance() // consume '='
			t._attemptCharCodeUntilFn(isNotWhitespace)
			t._consumeAttributeValue()
			continue
		}
		// Check if this is a directive (starts with '@')
		if t.cursor.Peek() == core.CharAT {
			t._consumeDirective()
		} else {
			// Save peek before consuming attr to check if cursor advanced
			peekBefore := t.cursor.Peek()
			offsetBefore := 0
			if plainCursor, ok := t.cursor.(*PlainCharacterCursor); ok {
				offsetBefore = plainCursor.state.Offset
			}
			t._consumeAttr()
			// If cursor didn't advance (e.g., quote without =), advance it to avoid infinite loop
			offsetAfter := 0
			if plainCursor, ok := t.cursor.(*PlainCharacterCursor); ok {
				offsetAfter = plainCursor.state.Offset
			}
			if offsetAfter == offsetBefore && peekBefore != core.CharGT && peekBefore != core.CharSLASH && peekBefore != core.CharEOF {
				fmt.Printf("[DEBUG] lexer._consumeAttributesAndDirectives: cursor didn't advance after _consumeAttr, peek=%d, forcing advance\n", peekBefore)
				t.cursor.Advance()
			}
		}
	}
}

func (t *Tokenizer) _consumeAttr() {
	t._beginToken(TokenTypeATTR_NAME, nil)

	attrNameStart := t.cursor.Peek()
	var nameEndPredicate func(code int) bool

	// Check if we're in directive context
	if t.openDirectiveCount > 0 {
		// If we're parsing attributes inside of directive syntax, we have to terminate the name
		// on the first non-matching closing paren. For example, if we have `@Dir(someAttr)`,
		// `@Dir` and `(` will have already been captured as `DIRECTIVE_NAME` and `DIRECTIVE_OPEN`
		// respectively, but the `)` will get captured as a part of the name for `someAttr`
		// because normally that would be an event binding.
		openParens := 0
		nameEndPredicate = func(code int) bool {
			if code == core.CharLPAREN {
				openParens++
			} else if code == core.CharRPAREN {
				if openParens == 0 {
					return true // Stop at closing paren if no open parens
				}
				openParens--
			}
			return isNameEnd(code)
		}
	} else if attrNameStart == core.CharLBRACKET {
		// For square-bracketed attributes, use permissive parsing
		// This allows mismatched brackets and more characters inside brackets
		// Track bracket depth and only check isNameEnd when brackets are balanced (depth <= 0)
		openBrackets := 0
		nameEndPredicate = func(code int) bool {
			// Update openBrackets first
			oldOpenBrackets := openBrackets
			if code == core.CharLBRACKET {
				openBrackets++
			} else if code == core.CharRBRACKET {
				openBrackets--
			}
			// Debug: log bracket updates and newline checks
			if code == core.CharLBRACKET || code == core.CharRBRACKET {
				fmt.Printf("[DEBUG] _consumeAttr: nameEndPredicate bracket update, code=%d, openBrackets: %d -> %d\n", code, oldOpenBrackets, openBrackets)
			}
			// Check for newline when openBrackets > 0 (matches TypeScript: chars.isNewLine(code) when openBrackets > 0)
			// This should stop parsing at newline, so the name doesn't include it
			isNewline := code == core.CharLF || code == core.CharCR
			if isNewline {
				fmt.Printf("[DEBUG] _consumeAttr: nameEndPredicate checking newline, openBrackets=%d, code=%d\n", openBrackets, code)
			}
			if openBrackets > 0 && isNewline {
				fmt.Printf("[DEBUG] _consumeAttr: nameEndPredicate returning true for newline, openBrackets=%d, code=%d ('%c')\n", openBrackets, code, rune(code))
				return true
			}
			// Only check for name-ending characters if the brackets are balanced or mismatched
			if openBrackets <= 0 {
				result := isNameEnd(code) || code == core.CharEQ
				if result {
					fmt.Printf("[DEBUG] _consumeAttr: nameEndPredicate returning true for nameEnd/EQ, openBrackets=%d, code=%d\n", openBrackets, code)
				}
				return result
			}
			return false
		}
	} else {
		nameEndPredicate = func(code int) bool {
			return isNameEnd(code) || code == core.CharEQ
		}
	}

	// Use _consumePrefixAndName with the appropriate nameEndPredicate
	prefixAndName := t._consumePrefixAndName(nameEndPredicate)
	t._endToken(prefixAndName, nil)

	if t._attemptCharCode(core.CharEQ) {
		t._attemptCharCodeUntilFn(isNotWhitespace)
		t._consumeAttributeValue()
	}
}

func (t *Tokenizer) _consumeDirective() {
	start := t.cursor.Clone()
	t._requireCharCode(core.CharAT)

	// nameStart should be at the position after @ (matches TypeScript: nameStart = start.clone(); nameStart.advance())
	// Clone from start (at @) and advance to position after @
	nameStart := start.Clone()
	nameStart.Advance()

	// Skip over the @ since it's not part of the name
	t.cursor.Advance()

	// Capture the rest of the name
	for isSelectorlessNameChar(t.cursor.Peek()) {
		t.cursor.Advance()
	}

	// Capture the opening token
	t._beginToken(TokenTypeDIRECTIVE_NAME, start)
	name := t.cursor.GetChars(nameStart)
	t._endToken([]string{name}, nil)
	t._attemptCharCodeUntilFn(isNotWhitespace)

	// Optionally there might be attributes bound to the specific directive
	// Stop parsing if there's no opening character for them
	if t.cursor.Peek() != core.CharLPAREN {
		return
	}

	t.openDirectiveCount++
	t._beginToken(TokenTypeDIRECTIVE_OPEN, nil)
	t.cursor.Advance()
	t._endToken([]string{}, nil)
	t._attemptCharCodeUntilFn(isNotWhitespace)

	// Capture all the attributes until we hit a closing paren
	for !isAttributeTerminator(t.cursor.Peek()) && t.cursor.Peek() != core.CharRPAREN {
		// Skip whitespace before consuming attribute
		t._attemptCharCodeUntilFn(isNotWhitespace)
		if isAttributeTerminator(t.cursor.Peek()) || t.cursor.Peek() == core.CharRPAREN {
			break
		}
		// Check if this is a directive (starts with '@')
		if t.cursor.Peek() == core.CharAT {
			t._consumeDirective()
		} else {
			t._consumeAttr()
		}
	}

	// Trim any trailing whitespace
	t._attemptCharCodeUntilFn(isNotWhitespace)
	t.openDirectiveCount--

	if t.cursor.Peek() != core.CharRPAREN {
		// Stop parsing, instead of throwing, if we've hit the end of the tag
		if t.cursor.Peek() == core.CharGT || t.cursor.Peek() == core.CharSLASH {
			return
		}
		panic(&CursorError{
			Msg:    _unexpectedCharacterErrorMsg(t.cursor.Peek()),
			Cursor: t.cursor.Clone(),
		})
	}

	// Capture the closing token
	t._beginToken(TokenTypeDIRECTIVE_CLOSE, nil)
	t.cursor.Advance()
	t._endToken([]string{}, nil)
	t._attemptCharCodeUntilFn(isNotWhitespace)
}

func isSelectorlessNameStart(code int) bool {
	// Matches TypeScript: code === chars.$_ || (code >= chars.$A && code <= chars.$Z)
	return code == core.CharUnderscore || (code >= core.CharA && code <= core.CharZ)
}

func isSelectorlessNameChar(code int) bool {
	// Matches TypeScript: isAsciiLetter(code) || code === chars.$DASH || code === chars.$UNDERSCORE || core.IsDigit(code)
	return core.IsAsciiLetter(code) || code == core.CharMINUS || code == core.CharUnderscore || core.IsDigit(code)
}

func isAttributeTerminator(code int) bool {
	// Matches TypeScript: code === chars.$SLASH || code === chars.$GT || code === chars.$LT || code === chars.$EOF
	return code == core.CharGT || code == core.CharSLASH || code == core.CharLT || code == core.CharEOF
}

func (t *Tokenizer) _consumeAttributeValue() {
	if t.cursor.Peek() == core.CharSQ || t.cursor.Peek() == core.CharDQ {
		quoteChar := t.cursor.Peek()
		t._consumeQuote(quoteChar)
		// In an attribute the end of the attribute value and the premature end to an interpolation
		// are both triggered by the `quoteChar`.
		// Note: We don't check EOF here - if EOF is encountered, _consumeQuote will report the error
		endPredicate := func() bool {
			return t.cursor.Peek() == quoteChar
		}
		t._consumeWithInterpolation(
			TokenTypeATTR_VALUE_TEXT,
			TokenTypeATTR_VALUE_INTERPOLATION,
			endPredicate,
			func() bool { return t._isTagStart() },
		)
		// Check if we hit EOF or '>' before finding closing quote
		if t.cursor.Peek() != quoteChar {
			if t.cursor.Peek() == core.CharEOF {
				panic(t._createError(
					"Unexpected character \"EOF\"",
					t.cursor.GetSpan(nil, nil),
				))
			} else if t.cursor.Peek() == core.CharGT {
				// If we hit '>', advance past it to report error at EOF position
				// This matches TypeScript behavior where it reports EOF error at position after '>'
				t.cursor.Advance()
				panic(t._createError(
					"Unexpected character \"EOF\"",
					t.cursor.GetSpan(nil, nil),
				))
			}
		}
		t._consumeQuote(quoteChar)
	} else {
		endPredicate := func() bool {
			peek := t.cursor.Peek()
			return isNameEnd(peek) || peek == core.CharEOF
		}
		t._consumeWithInterpolation(
			TokenTypeATTR_VALUE_TEXT,
			TokenTypeATTR_VALUE_INTERPOLATION,
			endPredicate,
			func() bool { return t._isTagStart() },
		)
	}
}

func (t *Tokenizer) _consumeQuote(quoteChar int) {
	fmt.Printf("[DEBUG] _consumeQuote: START, quoteChar=%d ('%c'), peek=%d\n", quoteChar, rune(quoteChar), t.cursor.Peek())
	t._beginToken(TokenTypeATTR_QUOTE, nil)
	t._requireCharCode(quoteChar)
	quoteStr := string(rune(quoteChar))
	fmt.Printf("[DEBUG] _consumeQuote: ending token with quote=%q\n", quoteStr)
	t._endToken([]string{quoteStr}, nil)
	fmt.Printf("[DEBUG] _consumeQuote: END, totalTokens=%d\n", len(t.tokens))
}

func (t *Tokenizer) _isLetStart() bool {
	return t.cursor.Peek() == core.CharAT && t._peekStr("@let")
}

func (t *Tokenizer) _consumeLetDeclaration(start CharacterCursor) {
	t._requireStr("@let")
	t._beginToken(TokenTypeLET_START, start)

	// Require at least one white space after the `@let`.
	if core.IsWhitespace(t.cursor.Peek()) {
		t._attemptCharCodeUntilFn(isNotWhitespace)
	} else {
		token := t._endToken([]string{t.cursor.GetChars(start)}, nil)
		token.(*TokenBase).tokenType = TokenTypeINCOMPLETE_LET
		return
	}

	startToken := t._endToken([]string{t._getLetDeclarationName()}, nil)

	// Skip over white space before the equals character.
	t._attemptCharCodeUntilFn(isNotWhitespace)

	// Expect an equals sign.
	if !t._attemptCharCode(core.CharEQ) {
		startToken.(*TokenBase).tokenType = TokenTypeINCOMPLETE_LET
		return
	}

	// Skip spaces after the equals.
	t._attemptCharCodeUntilFn(func(code int) bool {
		return isNotWhitespace(code) && !core.IsNewLine(code)
	})
	t._consumeLetDeclarationValue()

	// Terminate the `@let` with a semicolon.
	endChar := t.cursor.Peek()
	if endChar == core.CharSEMICOLON {
		t._beginToken(TokenTypeLET_END, nil)
		t._endToken([]string{}, nil)
		t.cursor.Advance()
	} else {
		startToken.(*TokenBase).tokenType = TokenTypeINCOMPLETE_LET
		startToken.(*TokenBase).sourceSpan = t.cursor.GetSpan(start, nil)
	}
}

func (t *Tokenizer) _isBlockStart() bool {
	if t.cursor.Peek() != core.CharAT {
		return false
	}
	for _, blockName := range SUPPORTED_BLOCKS {
		if t._peekStr(blockName) {
			return true
		}
	}
	return false
}

func (t *Tokenizer) _consumeBlockStart(start CharacterCursor) {
	t._requireCharCode(core.CharAT)
	t._beginToken(TokenTypeBLOCK_OPEN_START, start)
	startToken := t._endToken([]string{t._getBlockName()}, nil)

	if t.cursor.Peek() == core.CharLPAREN {
		// Advance past the opening paren.
		t.cursor.Advance()
		// Capture the parameters.
		t._consumeBlockParameters()
		// Allow spaces before the closing paren.
		t._attemptCharCodeUntilFn(isNotWhitespace)

		if t._attemptCharCode(core.CharRPAREN) {
			// Allow spaces after the paren.
			t._attemptCharCodeUntilFn(isNotWhitespace)
		} else {
			startToken.(*TokenBase).tokenType = TokenTypeINCOMPLETE_BLOCK_OPEN
			return
		}
	}

	if t._attemptCharCode(core.CharLBRACE) {
		t._beginToken(TokenTypeBLOCK_OPEN_END, nil)
		t._endToken([]string{}, nil)
	} else {
		startToken.(*TokenBase).tokenType = TokenTypeINCOMPLETE_BLOCK_OPEN
	}
}

func (t *Tokenizer) _tokenizeExpansionForm() bool {
	peekChar := t.cursor.Peek()
	if peekChar == core.CharRBRACE {
		isInCase := t._isInExpansionCase()
		isInForm := t._isInExpansionForm()
		fmt.Printf("[DEBUG] _tokenizeExpansionForm: found '}', checking expansion end, isInExpansionCase=%v, isInExpansionForm=%v, stackLen=%d\n",
			isInCase, isInForm, len(t.expansionCaseStack))
		if len(t.expansionCaseStack) > 0 {
			fmt.Printf("[DEBUG] _tokenizeExpansionForm: stack contents: %v\n", t.expansionCaseStack)
		}
	}
	if t._isExpansionFormStart() {
		t._consumeExpansionFormStart()
		return true
	}
	if t._isExpansionCaseStart() {
		t._consumeExpansionCaseStart()
		return true
	}
	if t._isExpansionCaseEnd() {
		fmt.Printf("[DEBUG] _tokenizeExpansionForm: found expansion case end\n")
		t._consumeExpansionCaseEnd()
		fmt.Printf("[DEBUG] _tokenizeExpansionForm: after consume case end, stackLen=%d\n", len(t.expansionCaseStack))
		return true
	}
	if t._isExpansionFormEnd() {
		fmt.Printf("[DEBUG] _tokenizeExpansionForm: found expansion form end\n")
		t._consumeExpansionFormEnd()
		return true
	}
	if peekChar == core.CharRBRACE {
		fmt.Printf("[DEBUG] _tokenizeExpansionForm: '}' not recognized as expansion end, returning false\n")
	}
	return false
}

func (t *Tokenizer) _isExpansionFormStart() bool {
	if t.cursor.Peek() != core.CharLBRACE {
		return false
	}
	// Peek ahead to check if it's interpolation ({{) vs ICU expansion ({identifier,)
	// If the next character is also {, it's interpolation, not expansion
	temp := t.cursor.Clone()
	temp.Advance()
	nextChar := temp.Peek()
	result := nextChar != core.CharLBRACE
	fmt.Printf("[DEBUG] _isExpansionFormStart: peek=%d, nextChar=%d, result=%v\n", t.cursor.Peek(), nextChar, result)
	return result
}

func (t *Tokenizer) _consumeExpansionFormStart() {
	t._beginToken(TokenTypeEXPANSION_FORM_START, nil)
	t._requireCharCode(core.CharLBRACE)
	t._endToken([]string{}, nil)
	t.expansionCaseStack = append(t.expansionCaseStack, TokenTypeEXPANSION_FORM_START)

	// Read switchValue (condition) until comma
	t._beginToken(TokenTypeRAW_TEXT, nil)
	conditionStart := t.cursor.Clone()
	t._attemptUntilChar(core.CharCOMMA)
	condition := t.cursor.GetChars(conditionStart)
	normalizedCondition := t._processCarriageReturns(condition)
	if t.i18nNormalizeLineEndingsInICUs {
		// We explicitly want to normalize line endings for this text.
		t._endToken([]string{normalizedCondition}, nil)
	} else {
		// We are not normalizing line endings.
		conditionToken := t._endToken([]string{condition}, nil)
		if normalizedCondition != condition {
			t.nonNormalizedIcuExpressions = append(t.nonNormalizedIcuExpressions, conditionToken)
		}
	}
	t._requireCharCode(core.CharCOMMA)
	t._attemptCharCodeUntilFn(isNotWhitespace)

	// Read type until comma
	t._beginToken(TokenTypeRAW_TEXT, nil)
	typeStart := t.cursor.Clone()
	t._attemptUntilChar(core.CharCOMMA)
	typ := t.cursor.GetChars(typeStart)
	t._endToken([]string{typ}, nil)
	t._requireCharCode(core.CharCOMMA)
	t._attemptCharCodeUntilFn(isNotWhitespace)
}

func (t *Tokenizer) _isExpansionCaseStart() bool {
	// Only consider it an expansion case start if we're already in an expansion form
	isInForm := t._isInExpansionForm()
	peekChar := t.cursor.Peek()
	isValidStart := peekChar == core.CharEQ || isAsciiLetter(peekChar) || isDigit(peekChar)
	result := isInForm && isValidStart
	if peekChar == core.CharEQ {
		fmt.Printf("[DEBUG] _isExpansionCaseStart: peek='=', isInForm=%v, isValidStart=%v, result=%v, stackLen=%d\n",
			isInForm, isValidStart, result, len(t.expansionCaseStack))
		if len(t.expansionCaseStack) > 0 {
			fmt.Printf("[DEBUG] _isExpansionCaseStart: top of stack=%d\n", t.expansionCaseStack[len(t.expansionCaseStack)-1])
		}
	}
	return result
}

func (t *Tokenizer) _consumeExpansionCaseStart() {
	// Matches TypeScript: _readUntil(chars.$LBRACE) - read until we hit '{'
	t._beginToken(TokenTypeEXPANSION_CASE_VALUE, nil)
	start := t.cursor.Clone()
	t._attemptUntilChar(core.CharLBRACE)
	value := strings.TrimSpace(t.cursor.GetChars(start))
	t._endToken([]string{value}, nil)

	// Skip whitespace after the value (matches TypeScript: _attemptCharCodeUntilFn(isNotWhitespace))
	t._attemptCharCodeUntilFn(isNotWhitespace)

	t._beginToken(TokenTypeEXPANSION_CASE_EXP_START, nil)
	t._requireCharCode(core.CharLBRACE)
	t._endToken([]string{}, nil)
	t._attemptCharCodeUntilFn(isNotWhitespace)
	t.expansionCaseStack = append(t.expansionCaseStack, TokenTypeEXPANSION_CASE_EXP_START)
}

func (t *Tokenizer) _isExpansionCaseEnd() bool {
	return t.cursor.Peek() == core.CharRBRACE && t._isInExpansionCase()
}

func (t *Tokenizer) _consumeExpansionCaseEnd() {
	t._beginToken(TokenTypeEXPANSION_CASE_EXP_END, nil)
	t._requireCharCode(core.CharRBRACE)
	t._endToken([]string{}, nil)
	t._attemptCharCodeUntilFn(isNotWhitespace)
	t.expansionCaseStack = t.expansionCaseStack[:len(t.expansionCaseStack)-1]
}

func (t *Tokenizer) _isExpansionFormEnd() bool {
	peekChar := t.cursor.Peek()
	isInForm := t._isInExpansionForm()
	result := peekChar == core.CharRBRACE && isInForm
	if peekChar == core.CharRBRACE {
		fmt.Printf("[DEBUG] _isExpansionFormEnd: peek='}', isInForm=%v, result=%v, stackLen=%d\n",
			isInForm, result, len(t.expansionCaseStack))
		if len(t.expansionCaseStack) > 0 {
			fmt.Printf("[DEBUG] _isExpansionFormEnd: top of stack=%d\n", t.expansionCaseStack[len(t.expansionCaseStack)-1])
		}
	}
	return result
}

func (t *Tokenizer) _consumeExpansionFormEnd() {
	t._beginToken(TokenTypeEXPANSION_FORM_END, nil)
	t._requireCharCode(core.CharRBRACE)
	t._endToken([]string{}, nil)
	t.expansionCaseStack = t.expansionCaseStack[:len(t.expansionCaseStack)-1]
}

// _processCarriageReturns processes carriage returns
func (t *Tokenizer) _processCarriageReturns(content string) string {
	if t.preserveLineEndings {
		return content
	}
	return strings.ReplaceAll(strings.ReplaceAll(content, "\r\n", "\n"), "\r", "\n")
}

// _normalizeCarriageReturns always normalizes carriage returns (for TEXT nodes)
func (t *Tokenizer) _normalizeCarriageReturns(content string) string {
	return strings.ReplaceAll(strings.ReplaceAll(content, "\r\n", "\n"), "\r", "\n")
}

// Helpers
func isAsciiLetter(code int) bool {
	return (code >= core.CharA && code <= core.CharZ) || (code >= core.CharLowerA && code <= core.CharLowerZ)
}

func isDigit(code int) bool {
	return code >= core.Char0 && code <= core.Char9
}

// TagInfo contains information about a consumed tag
type TagInfo struct {
	prefix         string
	name           string
	isSelfClosing  bool
	closingTagName string // Full closing tag name (e.g., "MyComp:title" for component tags)
}

func (t *Tokenizer) _consumeTagOpen(start CharacterCursor) *TagInfo {
	fmt.Printf("[DEBUG] _consumeTagOpen: START, peek=%d\n", t.cursor.Peek())
	var openTokenStarted bool

	// Use defer/recover to handle incomplete tags (terminated by EOF)
	defer func() {
		fmt.Printf("[DEBUG] _consumeTagOpen: END (defer), peek=%d\n", t.cursor.Peek())
		if r := recover(); r != nil {
			// Check if it's a ParseError (from _createError) or CursorError
			var isParseError bool
			if _, ok := r.(*util.ParseError); ok {
				isParseError = true
			} else if _, ok := r.(*CursorError); ok {
				isParseError = true
			}

			if isParseError {
				// We errored before we could close the opening tag, so it is incomplete.
				// Check if we have an open token that needs to be marked as incomplete
				if openTokenStarted {
					// Find the last TAG_OPEN_START token and change it to INCOMPLETE_TAG_OPEN
					for i := len(t.tokens) - 1; i >= 0; i-- {
						token := t.tokens[i]
						if token != nil && token.Type() == TokenTypeTAG_OPEN_START {
							if tagToken, ok := token.(*TagOpenStartToken); ok {
								tagToken.TokenBase.tokenType = TokenTypeINCOMPLETE_TAG_OPEN
								fmt.Printf("[DEBUG] _consumeTagOpen: changed token[%d] to INCOMPLETE_TAG_OPEN\n", i)
								break
							}
						}
					}
					// Check if error occurred during attribute consumption (not just tag name)
					// Errors during attribute consumption should be reported (e.g., missing closing quote)
					// Errors during tag name consumption (EOF) are expected for incomplete tags
					hasAttributes := false
					for i := len(t.tokens) - 1; i >= 0; i-- {
						token := t.tokens[i]
						if token != nil {
							tokenType := token.Type()
							if tokenType == TokenTypeATTR_NAME || tokenType == TokenTypeATTR_QUOTE || tokenType == TokenTypeATTR_VALUE_TEXT || tokenType == TokenTypeATTR_VALUE_INTERPOLATION {
								hasAttributes = true
								break
							}
							// Stop checking if we hit the TAG_OPEN_START token
							if tokenType == TokenTypeTAG_OPEN_START || tokenType == TokenTypeINCOMPLETE_TAG_OPEN {
								break
							}
						}
					}
					// If we have attributes, the error occurred during attribute consumption
					// and should be reported (even if it's EOF - e.g., missing closing quote)
					if hasAttributes {
						panic(r)
					}
					// If no attributes, check if it's an EOF error
					// EOF errors for incomplete tags without attributes are expected (tag terminated with EOF)
					if parseErr, ok := r.(*util.ParseError); ok {
						if parseErr.Msg != _unexpectedCharacterErrorMsg(core.CharEOF) {
							// Re-throw non-EOF errors so they can be reported
							panic(r)
						}
						// EOF errors for incomplete tags without attributes are expected, just return
						return
					} else if cursorErr, ok := r.(*CursorError); ok {
						if cursorErr.Msg != _unexpectedCharacterErrorMsg(core.CharEOF) {
							// Re-throw non-EOF errors so they can be reported
							panic(r)
						}
						// EOF errors for incomplete tags without attributes are expected, just return
						return
					}
					// Unknown error type, re-throw
					panic(r)
				} else {
					// When the start tag is invalid (no tag name consumed), assume we want a "<" as text.
					// Back to back text tokens are merged at the end.
					// The cursor was already advanced past "<" by _attemptCharCode(core.CharLT),
					// so we need to consume the rest as text until we hit a valid tag start or EOF
					// For "< a>", we want to consume everything until ">" as text
					textStart := start.Clone()
					// Consume everything until we hit ">" or EOF
					for t.cursor.Peek() != core.CharGT && t.cursor.Peek() != core.CharEOF {
						t.cursor.Advance()
					}
					// If we found ">", include it in the text token
					if t.cursor.Peek() == core.CharGT {
						t.cursor.Advance()
					}
					textValue := t.cursor.GetChars(textStart)
					fmt.Printf("[DEBUG] _consumeTagOpen: creating TEXT token with value=%q, textStart offset=%d, cursor offset=%d\n",
						textValue, textStart.Diff(t.cursor), t.cursor.Diff(textStart))
					t._beginToken(TokenTypeTEXT, textStart)
					t._endToken([]string{textValue}, nil)
					// Don't re-throw the error - allow tokenization to continue
					return
				}
			}
			panic(r)
		}
	}()

	var openToken Token
	var closingTagName string
	var prefix string
	var name string

	// Check if this is a component tag (selectorless enabled and starts with uppercase or underscore)
	if t.selectorlessEnabled && isSelectorlessNameStart(t.cursor.Peek()) {
		fmt.Printf("[DEBUG] _consumeTagOpen: detected component tag, peek=%d\n", t.cursor.Peek())
		openToken = t._consumeComponentOpenStart(start)
		parts := openToken.Parts()
		closingTagName = parts[0]
		if len(parts) > 1 {
			prefix = parts[1]
		}
		if len(parts) > 2 {
			name = parts[2]
		}
		// Build closingTagName like TypeScript: name + (prefix ? `:${prefix}` : '') + (tagName ? `:${tagName}` : '')
		if prefix != "" {
			closingTagName += ":" + prefix
		}
		if name != "" {
			closingTagName += ":" + name
		}
		t._attemptCharCodeUntilFn(isNotWhitespace)
	} else {
		// Regular tag - check ASCII letter BEFORE skipping whitespace
		if !isAsciiLetter(t.cursor.Peek()) {
			panic(t._createError(
				_unexpectedCharacterErrorMsg(t.cursor.Peek()),
				t.cursor.GetSpan(start, nil),
			))
		}
		fmt.Printf("[DEBUG] _consumeTagOpen: calling _consumePrefixAndName, peek=%d\n", t.cursor.Peek())
		prefixAndName := t._consumePrefixAndName(func(code int) bool {
			return isNameEnd(code) || code == core.CharSLASH
		})
		fmt.Printf("[DEBUG] _consumeTagOpen: got prefixAndName=%v, peek=%d\n", prefixAndName, t.cursor.Peek())
		prefix = prefixAndName[0]
		name = prefixAndName[1]
		if len(prefixAndName) > 2 {
			// This happens when we have a namespace
			name = prefixAndName[2]
		}
		closingTagName = name
		t._beginToken(TokenTypeTAG_OPEN_START, start)
		openTokenStarted = true
		openToken = t._endToken(prefixAndName, nil)
		// Skip whitespace after tag name (matches TypeScript line 826)
		t._attemptCharCodeUntilFn(isNotWhitespace)
	}

	// Consume attributes and directives
	t._consumeAttributesAndDirectives()
	fmt.Printf("[DEBUG] _consumeTagOpen: after _consumeAttributesAndDirectives, peek=%d\n", t.cursor.Peek())

	// Check if we have an incomplete tag due to newline in attribute name
	hasIncompleteAttr := false
	if len(t.tokens) > 0 {
		// Check all attribute tokens to see if any square-bracketed attribute name
		// was cut off by a newline (starts with '[' but doesn't end with ']')
		for i := len(t.tokens) - 1; i >= 0; i-- {
			if attrToken, ok := t.tokens[i].(*AttributeNameToken); ok {
				parts := attrToken.Parts()
				if len(parts) > 0 {
					attrName := parts[len(parts)-1]
					// Check if attribute name starts with '[' and doesn't end with ']'
					// This indicates it was cut off by a newline
					if len(attrName) > 0 && attrName[0] == '[' && attrName[len(attrName)-1] != ']' {
						// Found an incomplete square-bracketed attribute name
						hasIncompleteAttr = true
						break
					}
				}
			}
			// Stop at the first non-attribute token
			if token := t.tokens[i]; token != nil {
				tokenType := token.Type()
				if tokenType == TokenTypeTAG_OPEN_START || tokenType == TokenTypeCOMPONENT_OPEN_START {
					break
				}
			}
		}
	}

	// If we stopped at a quote (not part of attribute value), have incomplete attribute,
	// or encountered '<' (tag start), the tag is incomplete
	if t.cursor.Peek() == core.CharSQ || t.cursor.Peek() == core.CharDQ || hasIncompleteAttr || t.cursor.Peek() == core.CharLT {
		// Find the tag token (TAG_OPEN_START or COMPONENT_OPEN_START) and change it to incomplete
		// We need to search backwards from the end to find the tag token
		for i := len(t.tokens) - 1; i >= 0; i-- {
			token := t.tokens[i]
			if token != nil {
				tokenType := token.Type()
				if tokenType == TokenTypeTAG_OPEN_START {
					if tagToken, ok := token.(*TagOpenStartToken); ok {
						tagToken.TokenBase.tokenType = TokenTypeINCOMPLETE_TAG_OPEN
						break
					}
				} else if tokenType == TokenTypeCOMPONENT_OPEN_START {
					// Component tokens use TokenBase, so we need to update the token type
					// by creating a new token with the updated type
					parts := token.Parts()
					sourceSpan := token.SourceSpan()
					// Replace the old token with a new one with INCOMPLETE_COMPONENT_OPEN type
					newToken := NewTokenBase(TokenTypeINCOMPLETE_COMPONENT_OPEN, parts, sourceSpan)
					t.tokens[i] = newToken
					break
				}
			}
		}
		// Consume the remaining content as TEXT
		if t.cursor.Peek() == core.CharSLASH {
			t.cursor.Advance() // Skip '/' to match expected output
		}
		textStart := t.cursor.Clone()
		// If we stopped at a quote, consume the quote and everything until '>' or '<' (tag start) or EOF
		if t.cursor.Peek() == core.CharSQ || t.cursor.Peek() == core.CharDQ {
			// Consume quote and everything until '>', '<' (tag start), or EOF
			// Don't consume '<' (tag start) - let the main loop handle it
			for t.cursor.Peek() != core.CharGT && t.cursor.Peek() != core.CharLT && t.cursor.Peek() != core.CharEOF {
				t.cursor.Advance()
			}
			if t.cursor.Peek() == core.CharGT {
				t.cursor.Advance() // Consume '>'
			}
			// If we stopped at '<' (tag start), don't consume it - let the main loop handle it
			textContent := t.cursor.GetChars(textStart)
			t._beginToken(TokenTypeTEXT, textStart)
			t._endToken([]string{textContent}, t.cursor)
			return &TagInfo{prefix: prefix, name: name, isSelfClosing: false, closingTagName: closingTagName}
		}
		// If we stopped at '<' (tag start), don't consume it as TEXT
		// The main loop will consume it as a new tag
		if t.cursor.Peek() == core.CharLT {
			// Don't consume '<' - let the main loop handle it
			return &TagInfo{prefix: prefix, name: name, isSelfClosing: false, closingTagName: closingTagName}
		}
		// Otherwise, consume until '>' or EOF (for incomplete attribute case)
		for t.cursor.Peek() != core.CharGT && t.cursor.Peek() != core.CharEOF {
			t.cursor.Advance()
		}
		if t.cursor.Peek() == core.CharGT {
			t.cursor.Advance() // Consume '>'
		}
		textContent := t.cursor.GetChars(textStart)
		t._beginToken(TokenTypeTEXT, nil)
		t._endToken([]string{textContent}, nil)
		return &TagInfo{prefix: prefix, name: name, isSelfClosing: false, closingTagName: closingTagName}
	}

	// Check for self-closing or end tag
	if t._attemptCharCode(core.CharSLASH) {
		// Self-closing tag
		if openToken.Type() == TokenTypeCOMPONENT_OPEN_START {
			t._beginToken(TokenTypeCOMPONENT_OPEN_END_VOID, nil)
		} else {
			t._beginToken(TokenTypeTAG_OPEN_END_VOID, nil)
		}
		if !t._attemptCharCode(core.CharGT) {
			// EOF or other error - incomplete tag
			panic(&CursorError{
				Msg:    _unexpectedCharacterErrorMsg(t.cursor.Peek()),
				Cursor: t.cursor.Clone(),
			})
		}
		t._endToken([]string{}, nil)
		return &TagInfo{prefix: prefix, name: name, isSelfClosing: true, closingTagName: closingTagName}
	} else {
		// Check if we're at EOF before trying to consume '>'
		if t.cursor.Peek() == core.CharEOF {
			// Tag is incomplete - this will be handled by defer/recover
			panic(&CursorError{
				Msg:    _unexpectedCharacterErrorMsg(core.CharEOF),
				Cursor: t.cursor.Clone(),
			})
		}
		// End tag
		if openToken.Type() == TokenTypeCOMPONENT_OPEN_START {
			t._consumeComponentOpenEnd()
		} else {
			t._beginToken(TokenTypeTAG_OPEN_END, nil)
			if !t._attemptCharCode(core.CharGT) {
				// EOF or other error - incomplete tag
				panic(&CursorError{
					Msg:    _unexpectedCharacterErrorMsg(t.cursor.Peek()),
					Cursor: t.cursor.Clone(),
				})
			}
			t._endToken([]string{}, nil)
		}
		return &TagInfo{prefix: prefix, name: name, isSelfClosing: false, closingTagName: closingTagName}
	}
}

func (t *Tokenizer) _consumeComponentOpenStart(start CharacterCursor) Token {
	t._beginToken(TokenTypeCOMPONENT_OPEN_START, start)
	parts := t._consumeComponentName()
	return t._endToken(parts, nil)
}

func (t *Tokenizer) _consumeComponentName() []string {
	nameStart := t.cursor.Clone()
	for isSelectorlessNameChar(t.cursor.Peek()) {
		t.cursor.Advance()
	}
	name := t.cursor.GetChars(nameStart)
	prefix := ""
	tagName := ""
	if t.cursor.Peek() == core.CharCOLON {
		t.cursor.Advance()
		prefixAndName := t._consumePrefixAndName(isNameEnd)
		if len(prefixAndName) > 0 {
			prefix = prefixAndName[0]
		}
		if len(prefixAndName) > 1 {
			tagName = prefixAndName[1]
		}
	}
	return []string{name, prefix, tagName}
}

func (t *Tokenizer) _consumeComponentOpenEnd() {
	t._beginToken(TokenTypeCOMPONENT_OPEN_END, nil)
	if !t._attemptCharCode(core.CharGT) {
		// EOF or other error - incomplete tag
		panic(&CursorError{
			Msg:    _unexpectedCharacterErrorMsg(t.cursor.Peek()),
			Cursor: t.cursor.Clone(),
		})
	}
	t._endToken([]string{}, nil)
}

func (t *Tokenizer) _consumeRawTextWithTagClose(openToken Token, closingTagName string, consumeEntities bool) {
	// Consume raw text until we find the closing tag
	t._consumeRawText(consumeEntities, func() bool {
		if !t._attemptCharCode(core.CharLT) {
			return false
		}
		if !t._attemptCharCode(core.CharSLASH) {
			return false
		}
		t._attemptCharCodeUntilFn(isNotWhitespace)
		// Case-insensitive match for closing tag name using _attemptStrCaseInsensitive
		if !t._attemptStrCaseInsensitive(closingTagName) {
			return false
		}
		t._attemptCharCodeUntilFn(isNotWhitespace)
		return t._attemptCharCode(core.CharGT)
	})
	// Now consume the closing tag token
	// After _consumeRawText, cursor is at the position before </closingTagName>
	tagCloseStart := t.cursor.Clone()
	// Check if this is a component tag or regular tag
	if openToken != nil && openToken.Type() == TokenTypeCOMPONENT_OPEN_START {
		t._beginToken(TokenTypeCOMPONENT_CLOSE, tagCloseStart)
	} else {
		t._beginToken(TokenTypeTAG_CLOSE, tagCloseStart)
	}
	t._requireCharCodeUntilFn(func(code int) bool { return code == core.CharGT }, 3)
	t.cursor.Advance() // Consume the `>`
	// Use openToken.parts for component tags, or reconstruct for regular tags
	if openToken != nil {
		t._endToken(openToken.Parts(), nil)
	} else {
		// Fallback: reconstruct prefixAndName from closingTagName (no prefix for raw text tags)
		t._endToken([]string{"", closingTagName}, nil)
	}
}

func (t *Tokenizer) _consumeRawText(consumeEntities bool, endMarkerPredicate func() bool) {
	t._beginToken(TokenTypeRAW_TEXT, nil)
	if consumeEntities {
		t.currentTokenType = TokenTypeESCAPABLE_RAW_TEXT
	}
	parts := []string{}
	for {
		tagCloseStart := t.cursor.Clone()
		foundEndMarker := endMarkerPredicate()
		if foundEndMarker {
			// Reset cursor to position before closing tag so caller can consume it
			t.cursor = tagCloseStart
			break
		}
		// Reset cursor if endMarkerPredicate didn't find the marker
		t.cursor = tagCloseStart
		if consumeEntities && t.cursor.Peek() == core.CharAMPERSAND {
			t._endToken([]string{t._normalizeCarriageReturns(strings.Join(parts, ""))}, nil)
			parts = []string{}
			t._consumeEntity(TokenTypeESCAPABLE_RAW_TEXT)
			t._beginToken(TokenTypeESCAPABLE_RAW_TEXT, nil)
		} else {
			char := t._readChar()
			parts = append(parts, char)
		}
	}
	rawContent := strings.Join(parts, "")
	// In Go, raw strings (backticks) contain literal \n and \r sequences, not actual newlines.
	// We need to convert literal escape sequences to actual characters first.
	// This matches TypeScript behavior where single-quoted strings interpret escape sequences.
	processedContent := strings.ReplaceAll(strings.ReplaceAll(rawContent, "\\n", "\n"), "\\r", "\r")
	// For RAW_TEXT, always normalize line endings (similar to TypeScript's _CR_OR_CRLF_REGEXP)
	// This matches TypeScript's behavior where _processCarriageReturns uses /\r\n?/g
	// TypeScript regex /\r\n?/g matches \r\n or \r, so we need to replace both
	// Replace \r\n with \n first, then replace remaining \r with \n
	normalizedContent := strings.ReplaceAll(strings.ReplaceAll(processedContent, "\r\n", "\n"), "\r", "\n")
	t._endToken([]string{normalizedContent}, nil)
}

func (t *Tokenizer) _consumeEntity(textTokenType TokenType) {
	t._beginToken(TokenTypeENCODED_ENTITY, nil)
	start := t.cursor.Clone()
	t.cursor.Advance()
	if t._attemptCharCode(core.CharHASH) {
		isHex := t._attemptCharCode(core.CharLowerX) || t._attemptCharCode(core.CharX)
		codeStart := t.cursor.Clone()
		t._attemptCharCodeUntilFn(isDigitEntityEnd)
		if t.cursor.Peek() != core.CharSEMICOLON {
			// Advance cursor to include the peeked character in the string provided to the error message
			// This matches TypeScript behavior
			t.cursor.Advance()
			entityType := "decimal"
			if isHex {
				entityType = "hexadecimal"
			}
			// Use GetSpan with nil start to match TypeScript's getSpan() behavior
			// This will create a span from the current token start to the current cursor position
			panic(t._createError(
				fmt.Sprintf("Unable to parse entity \"%s\" - %s character reference entities must end with \";\"", t.cursor.GetChars(start), entityType),
				t.cursor.GetSpan(nil, nil),
			))
		}
		strNum := t.cursor.GetChars(codeStart)
		t.cursor.Advance()

		// Parse the numeric value
		var charCode int64
		var err error
		if isHex {
			charCode, err = strconv.ParseInt(strNum, 16, 32)
		} else {
			charCode, err = strconv.ParseInt(strNum, 10, 32)
		}

		if err != nil || charCode < 0 || charCode > 0x10FFFF {
			panic(t._createError(
				fmt.Sprintf("Unknown entity \"%s\" - invalid code point", t.cursor.GetChars(start)),
				t.cursor.GetSpan(start, nil),
			))
		}

		// Create the decoded character and encoded form
		decoded := string(rune(charCode))
		encoded := t.cursor.GetChars(start)
		t._endToken([]string{decoded, encoded}, nil)
	} else {
		nameStart := t.cursor.Clone()
		t._attemptCharCodeUntilFn(isNamedEntityEnd)
		if t.cursor.Peek() != core.CharSEMICOLON {
			// No semicolon found - treat as text
			// Read the entire entity name (including &) as text
			// Reset cursor to start (before &) to read the entire &name as text
			t._beginToken(textTokenType, start)
			// Get the entity name (without semicolon)
			entityName := t.cursor.GetChars(nameStart)
			// Create text token with & + entityName (e.g., "&amp")
			t._endToken([]string{"&" + entityName}, nil)
			// Cursor is already at the position after the entity name
		} else {
			name := t.cursor.GetChars(nameStart)
			t.cursor.Advance()

			// Look up the named entity
			char, ok := GetNamedEntity(name)
			if !ok {
				panic(t._createError(
					fmt.Sprintf("Unknown entity \"%s\" - use the \"&#<decimal>;\" or  \"&#x<hex>;\" syntax", name),
					t.cursor.GetSpan(start, nil),
				))
			}

			t._endToken([]string{char, fmt.Sprintf("&%s;", name)}, nil)
		}
	}
}

// _beginToken begins a new token
func (t *Tokenizer) _beginToken(tokenType TokenType, start CharacterCursor) {
	if start == nil {
		start = t.cursor.Clone()
	}
	t.currentTokenStart = start
	t.currentTokenType = tokenType
}

// _endToken ends the current token
func (t *Tokenizer) _endToken(parts []string, end CharacterCursor) Token {
	if t.currentTokenStart == nil {
		panic("Programming error - attempted to end a token when there was no start to the token")
	}
	if t.currentTokenType == -1 {
		panic("Programming error - attempted to end a token which has no token type")
	}

	if end == nil {
		end = t.cursor
	}
	// Use the provided end cursor (or current cursor if nil) to create the source span
	sourceSpan := end.GetSpan(t.currentTokenStart, t.leadingTriviaCodePoints)

	// Create the appropriate token type based on currentTokenType
	var token Token
	switch t.currentTokenType {
	case TokenTypeTAG_OPEN_START, TokenTypeINCOMPLETE_TAG_OPEN:
		prefix := ""
		name := ""
		if len(parts) > 0 {
			prefix = parts[0]
		}
		if len(parts) > 1 {
			name = parts[1]
		}
		token = NewTagOpenStartToken(prefix, name, sourceSpan)
		// If it's incomplete, we need to change the type
		if t.currentTokenType == TokenTypeINCOMPLETE_TAG_OPEN {
			token.(*TagOpenStartToken).TokenBase.tokenType = TokenTypeINCOMPLETE_TAG_OPEN
		}
	case TokenTypeCOMPONENT_OPEN_START, TokenTypeINCOMPLETE_COMPONENT_OPEN:
		// Component tokens have parts: [componentName, prefix, tagName]
		// Use TokenBase for component tokens
		token = NewTokenBase(t.currentTokenType, parts, sourceSpan)
	case TokenTypeTAG_CLOSE:
		prefix := ""
		name := ""
		if len(parts) > 0 {
			prefix = parts[0]
		}
		if len(parts) > 1 {
			name = parts[1]
		}
		token = NewTagCloseToken(prefix, name, sourceSpan)
	case TokenTypeCOMPONENT_CLOSE:
		// Component tokens have parts: [componentName, prefix, tagName]
		// Use TokenBase for component tokens
		token = NewTokenBase(TokenTypeCOMPONENT_CLOSE, parts, sourceSpan)
	case TokenTypeATTR_NAME:
		prefix := ""
		name := ""
		if len(parts) > 0 {
			prefix = parts[0]
		}
		if len(parts) > 1 {
			name = parts[1]
		}
		token = NewAttributeNameToken(prefix, name, sourceSpan)
	case TokenTypeATTR_VALUE_TEXT:
		value := ""
		if len(parts) > 0 {
			value = parts[0]
		}
		token = NewAttributeValueTextToken(value, sourceSpan)
	case TokenTypeTEXT, TokenTypeRAW_TEXT, TokenTypeESCAPABLE_RAW_TEXT:
		value := ""
		if len(parts) > 0 {
			value = parts[0]
		}
		token = NewTextToken(value, t.currentTokenType, sourceSpan)
	case TokenTypeINTERPOLATION:
		startMarker := ""
		expression := ""
		var endMarker *string
		if len(parts) > 0 {
			startMarker = parts[0]
		}
		if len(parts) > 1 {
			expression = parts[1]
		}
		if len(parts) > 2 {
			em := parts[2]
			endMarker = &em
		}
		token = NewInterpolationToken(startMarker, expression, endMarker, sourceSpan)
	case TokenTypeENCODED_ENTITY:
		decoded := ""
		encoded := ""
		if len(parts) > 0 {
			decoded = parts[0]
		}
		if len(parts) > 1 {
			encoded = parts[1]
		}
		token = NewEncodedEntityToken(decoded, encoded, sourceSpan)
	case TokenTypeCDATA_START, TokenTypeCDATA_END, TokenTypeDOC_TYPE, TokenTypeCOMMENT_START, TokenTypeCOMMENT_END:
		// These tokens don't have specific structs, they use TokenBase
		token = NewTokenBase(t.currentTokenType, parts, sourceSpan)
	default:
		// For other token types, use TokenBase
		token = NewTokenBase(t.currentTokenType, parts, sourceSpan)
	}

	t.tokens = append(t.tokens, token)
	if token.Type() == TokenTypeTAG_OPEN_START {
		fmt.Printf("[DEBUG] _endToken: added TAG_OPEN_START to tokens array, totalTokens=%d, parts=%v\n",
			len(t.tokens), token.Parts())
	}

	t.currentTokenStart = nil
	t.currentTokenType = -1

	return token
}

func (t *Tokenizer) _consumePrefixAndName(endPredicate func(code int) bool) []string {
	fmt.Printf("[DEBUG] _consumePrefixAndName: START, peek=%d, CharsLeft=%d\n", t.cursor.Peek(), t.cursor.CharsLeft())
	nameOrPrefixStart := t.cursor.Clone()
	prefix := ""
	prefixLoopCount := 0
	for t.cursor.Peek() != core.CharCOLON && !isPrefixEnd(t.cursor.Peek()) {
		prefixLoopCount++
		if prefixLoopCount > 1000 {
			peekBefore := t.cursor.Peek()
			charsLeft := t.cursor.CharsLeft()
			fmt.Printf("[DEBUG] _consumePrefixAndName: INFINITE LOOP in prefix loop! iterationCount=%d, peek=%d, isPrefixEnd=%v, CharsLeft=%d\n",
				prefixLoopCount, peekBefore, isPrefixEnd(peekBefore), charsLeft)
			// Check if cursor can advance
			t.cursor.Advance()
			peekAfter := t.cursor.Peek()
			fmt.Printf("[DEBUG] _consumePrefixAndName: after advance, peek=%d (changed: %v), CharsLeft=%d\n", peekAfter, peekBefore != peekAfter, t.cursor.CharsLeft())
			break
		}
		peekBeforeAdvance := t.cursor.Peek()
		charsLeftBefore := t.cursor.CharsLeft()
		if peekBeforeAdvance == core.CharEOF || charsLeftBefore <= 0 {
			fmt.Printf("[DEBUG] _consumePrefixAndName: reached EOF, breaking (peek=%d, CharsLeft=%d)\n", peekBeforeAdvance, charsLeftBefore)
			break
		}
		// Check if we should break based on the condition
		if t.cursor.Peek() == core.CharCOLON || isPrefixEnd(t.cursor.Peek()) {
			fmt.Printf("[DEBUG] _consumePrefixAndName: condition met, breaking (peek=%d, isColon=%v, isPrefixEnd=%v)\n",
				t.cursor.Peek(), t.cursor.Peek() == core.CharCOLON, isPrefixEnd(t.cursor.Peek()))
			break
		}
		t.cursor.Advance()
		peekAfterAdvance := t.cursor.Peek()
		charsLeftAfter := t.cursor.CharsLeft()
		if peekBeforeAdvance == peekAfterAdvance && prefixLoopCount > 10 {
			fmt.Printf("[DEBUG] _consumePrefixAndName: cursor not advancing! peek=%d (EOF=%d), CharsLeft before=%d, after=%d, iteration=%d\n",
				peekBeforeAdvance, core.CharEOF, charsLeftBefore, charsLeftAfter, prefixLoopCount)
			// Force break to avoid infinite loop
			break
		}
	}
	fmt.Printf("[DEBUG] _consumePrefixAndName: after prefix loop, peek=%d, prefixLoopCount=%d\n", t.cursor.Peek(), prefixLoopCount)
	var nameStart CharacterCursor
	if t.cursor.Peek() == core.CharCOLON {
		prefix = t.cursor.GetChars(nameOrPrefixStart)
		t.cursor.Advance()
		nameStart = t.cursor.Clone()
		fmt.Printf("[DEBUG] _consumePrefixAndName: found colon, prefix=%s, peek=%d\n", prefix, t.cursor.Peek())
	} else {
		// No prefix - nameStart is from the beginning, but cursor stays at current position
		// This handles cases like "ref-a" where the loop stopped at "-"
		// The cursor will continue reading from "-" until endPredicate returns true
		nameStart = nameOrPrefixStart
		fmt.Printf("[DEBUG] _consumePrefixAndName: no prefix, peek=%d\n", t.cursor.Peek())
	}
	// Matches TypeScript: _requireCharCodeUntilFn(endPredicate, prefix === '' ? 0 : 1)
	minLength := 0
	if prefix != "" {
		minLength = 1
	}
	fmt.Printf("[DEBUG] _consumePrefixAndName: calling _requireCharCodeUntilFn, minLength=%d, peek=%d\n", minLength, t.cursor.Peek())
	t._requireCharCodeUntilFn(endPredicate, minLength)
	fmt.Printf("[DEBUG] _consumePrefixAndName: after _requireCharCodeUntilFn, peek=%d\n", t.cursor.Peek())
	name := t.cursor.GetChars(nameStart)
	fmt.Printf("[DEBUG] _consumePrefixAndName: END, prefix=%q, name=%q, nameStart chars=%q\n", prefix, name, func() string {
		if nameStart != nil {
			return t.cursor.GetChars(nameStart)
		}
		return ""
	}())
	return []string{prefix, name}
}

func (t *Tokenizer) _consumeBlockEnd(start CharacterCursor) {
	t._beginToken(TokenTypeBLOCK_CLOSE, start)
	t._endToken([]string{}, nil)
}

func (t *Tokenizer) _consumeWithInterpolation(textTokenType TokenType, interpolationTokenType TokenType, isTextEnd func() bool, isTagStart func() bool) {
	peekChar := t.cursor.Peek()
	fmt.Printf("[DEBUG] _consumeWithInterpolation: START, peek=%d ('%c'), isTextEnd()=%v\n", peekChar, peekChar, isTextEnd())
	t._beginToken(textTokenType, nil)
	parts := []string{}
	interpIterationCount := 0
	// Stop when isTextEnd() is true, or when we hit EOF or '>' (tag end) in attribute value context
	for !isTextEnd() && t.cursor.Peek() != core.CharEOF && t.cursor.Peek() != core.CharGT {
		interpIterationCount++
		if interpIterationCount > 1000 {
			fmt.Printf("[DEBUG] _consumeWithInterpolation: INFINITE LOOP DETECTED! iterationCount=%d, peek=%d, isTextEnd()=%v\n",
				interpIterationCount, t.cursor.Peek(), isTextEnd())
			break
		}
		currentPeek := t.cursor.Peek()
		if currentPeek == core.CharRBRACE {
			fmt.Printf("[DEBUG] _consumeWithInterpolation: found '}', iteration=%d, isTextEnd()=%v, isInExpansionCase=%v, isInExpansionForm=%v\n",
				interpIterationCount, isTextEnd(), t._isInExpansionCase(), t._isInExpansionForm())
		}
		if interpIterationCount <= 20 || interpIterationCount%100 == 0 {
			fmt.Printf("[DEBUG] _consumeWithInterpolation: iteration=%d, peek=%d, isTextEnd()=%v\n",
				interpIterationCount, currentPeek, isTextEnd())
		}
		if interpIterationCount <= 20 {
			fmt.Printf("[DEBUG] _consumeWithInterpolation: before _attemptStr(INTERPOLATION.start), peek=%d ('%c'), CharsLeft=%d\n",
				t.cursor.Peek(), t.cursor.Peek(), t.cursor.CharsLeft())
		}
		// Clone cursor before attempting to match INTERPOLATION.start
		// This will be used as:
		// 1. End cursor for the TEXT token (before consuming {{)
		// 2. Start cursor for the INTERPOLATION token (before consuming {{)
		beforeInterpolationCursor := t.cursor.Clone()
		if t._attemptStr(INTERPOLATION.start) {
			if interpIterationCount <= 20 {
				fmt.Printf("[DEBUG] _consumeWithInterpolation: _attemptStr(INTERPOLATION.start) matched, starting interpolation\n")
			}
			// End the current text token before starting interpolation
			// Use beforeInterpolationCursor as end cursor to exclude {{ from TEXT token
			if len(parts) > 0 {
				t._endToken([]string{t._normalizeCarriageReturns(strings.Join(parts, ""))}, beforeInterpolationCursor)
			} else {
				// If parts is empty, we still need to end the token to close it properly
				t._endToken([]string{""}, beforeInterpolationCursor)
			}
			parts = []string{}
			// Use the cloned cursor (before consuming {{) as the start cursor for the interpolation token
			t._beginToken(interpolationTokenType, beforeInterpolationCursor)

			startMarker := INTERPOLATION.start

			// Set inInterpolation flag before consuming interpolation content
			wasInInterpolation := t.inInterpolation
			t.inInterpolation = true
			// Manually consume interpolation content, tracking quotes to ignore }} inside strings
			// This matches TypeScript's _consumeInterpolation logic
			interpolationParts := []string{}
			foundEnd := false
			var inQuote *int = nil // Track which quote character we're in (single or double)
			inComment := false
			interpLoopCount := 0
			tagStartEncountered := false
			// Track the start of the expression for tag start case
			expressionStart := t.cursor.Clone()
			// Check prematureEndPredicate (isTagStart) and isTextEnd() in while condition, matching TypeScript
			// isTextEnd() is checked when not in a quote to allow interpolation to end in attribute value
			// However, we also need to check isTextEnd() even when inQuote != nil to handle the case where
			// we're in a quote from the expression but encounter the matching quote of the attribute value
			// Also check for tag end '>' to stop when attribute value is not properly closed
			for t.cursor.Peek() != core.CharEOF && (isTagStart == nil || !isTagStart()) && t.cursor.Peek() != core.CharGT {
				// Check isTextEnd() - but only break if peek is a quote (attribute value context)
				// This handles the case where we encounter the matching quote of attribute value
				// even when we're inside a quote from the interpolation expression
				// We don't break if peek is '}' (block end) - interpolation should continue until }}
				if isTextEnd != nil && isTextEnd() {
					peekChar := t.cursor.Peek()
					// Only break if peek is a quote (single or double) - this indicates attribute value context
					// Don't break if peek is '}' (block end) - interpolation should continue
					if peekChar == core.CharSQ || peekChar == core.CharDQ {
						// Prematurely terminated interpolation (no }} found)
						// Don't set foundEnd=true - this is a premature termination, not a proper end
						if interpLoopCount <= 20 {
							fmt.Printf("[DEBUG] _consumeWithInterpolation: isTextEnd()=true and peek is quote (inQuote=%v, peek=%d), breaking (prematurely terminated)\n", inQuote != nil, peekChar)
						}
						break
					} else {
						// isTextEnd() is true but peek is not a quote (e.g., '}' in block context)
						// Don't break - interpolation should continue until }}
						if interpLoopCount <= 20 {
							fmt.Printf("[DEBUG] _consumeWithInterpolation: isTextEnd()=true but peek is not quote (peek=%d), continuing\n", peekChar)
						}
					}
				}
				interpLoopCount++
				if interpLoopCount > 1000 {
					fmt.Printf("[DEBUG] _consumeWithInterpolation: INFINITE LOOP in interpolation! iterationCount=%d, peek=%d, inQuote=%v, inComment=%v\n",
						interpLoopCount, t.cursor.Peek(), inQuote != nil, inComment)
					break
				}
				currentPeek := t.cursor.Peek()
				if currentPeek == core.CharEOF {
					break
				}
				// Check isTagStart() BEFORE checking INTERPOLATION.end, matching TypeScript
				// This handles the case where we encounter a tag start (like <! comment) in the interpolation
				if isTagStart != nil && isTagStart() {
					// We are starting what looks like an HTML element in the middle of this interpolation.
					// Reset the cursor to before the `<` character and end the interpolation token.
					// (This is actually wrong but here for backward compatibility).
					// Push the expression chars up to (but not including) the tag start
					expressionChars := t.cursor.GetChars(expressionStart)
					// Process carriage returns (normalize CRLF to LF)
					expressionChars = t._processCarriageReturns(expressionChars)
					interpolationParts = append(interpolationParts, expressionChars)
					if interpLoopCount <= 20 {
						fmt.Printf("[DEBUG] _consumeWithInterpolation: isTagStart()=true, pushing expression chars, breaking\n")
					}
					foundEnd = false // This is a premature termination, not a proper end
					break
				}
				// Only match }} when not in a quote - check this AFTER checking isTagStart()
				// This ensures }} is found before premature termination
				if inQuote == nil {
					if t._attemptStr(INTERPOLATION.end) {
						if interpLoopCount <= 20 {
							fmt.Printf("[DEBUG] _consumeWithInterpolation: found INTERPOLATION.end, setting foundEnd=true, inQuote=%v\n", inQuote)
						}
						foundEnd = true
						break
					} else if t._attemptStr("//") {
						// Once we are in a comment we ignore any quotes
						// Append the "//" to interpolationParts since it was consumed
						interpolationParts = append(interpolationParts, "//")
						inComment = true
						// Continue to read the rest of the comment
						continue
					}
					// Check isTagStart() BEFORE checking isTextEnd(), matching TypeScript
					// This handles the case where we encounter a tag start (like <! comment) in the interpolation
					if isTagStart != nil && isTagStart() {
						if interpLoopCount <= 20 {
							fmt.Printf("[DEBUG] _consumeWithInterpolation: isTagStart()=true before reading char (peek=%d), pushing expression chars, breaking\n", t.cursor.Peek())
						}
						// We are starting what looks like an HTML element in the middle of this interpolation.
						// Reset the cursor to before the `<` character and end the interpolation token.
						// (This is actually wrong but here for backward compatibility).
						// Push the expression chars up to (but not including) the tag start
						expressionChars := t.cursor.GetChars(expressionStart)
						// Process carriage returns (normalize CRLF to LF)
						expressionChars = t._processCarriageReturns(expressionChars)
						interpolationParts = append(interpolationParts, expressionChars)
						foundEnd = false // This is a premature termination, not a proper end
						break
					} else if isTagStart != nil && interpLoopCount <= 20 {
						fmt.Printf("[DEBUG] _consumeWithInterpolation: isTagStart()=false before reading char (peek=%d), isTextEnd()=%v\n", t.cursor.Peek(), isTextEnd != nil && isTextEnd())
					}
					// Check isTextEnd() before reading char when not in quote
					// This allows interpolation to end when matching quote is encountered in attribute value
					// We check BEFORE reading the char so that the quote char is not included in interpolation
					// This is a premature termination (no }} found), so don't set foundEnd=true
					if isTextEnd != nil && isTextEnd() {
						currentPeek := t.cursor.Peek()
						// If isTextEnd() is true because of tag start (peek next is '<'), but current peek is not '<',
						// we need to consume the current char (e.g., '}') before breaking, matching TypeScript behavior
						// Check if peek next is '<' by cloning cursor and advancing
						if currentPeek != core.CharLT {
							peekCursor := t.cursor.Clone()
							peekCursor.Advance()
							if peekCursor.Peek() == core.CharLT {
								// Peek next is '<', so consume current char before breaking
								char := t._readChar()
								interpolationParts = append(interpolationParts, char)
								if interpLoopCount <= 20 {
									fmt.Printf("[DEBUG] _consumeWithInterpolation: isTextEnd()=true, consumed char='%s' (code=%d) before breaking (peek next is '<')\n", char, int(char[0]))
								}
							}
						}
						if interpLoopCount <= 20 {
							fmt.Printf("[DEBUG] _consumeWithInterpolation: isTextEnd()=true before reading char (peek=%d), breaking (prematurely terminated)\n", t.cursor.Peek())
						}
						break
					}
				}
				// Read the next character
				char := t._readChar()
				interpolationParts = append(interpolationParts, char)
				charCode := int(char[0])
				if interpLoopCount <= 20 {
					fmt.Printf("[DEBUG] _consumeWithInterpolation: read char='%s' (code=%d), inQuote=%v, isTextEnd()=%v\n", char, charCode, inQuote != nil, isTextEnd != nil && isTextEnd())
				}
				if charCode == core.CharBACKSLASH {
					// Skip the next character because it was escaped.
					if t.cursor.Peek() != core.CharEOF {
						nextChar := t._readChar()
						interpolationParts = append(interpolationParts, nextChar)
					}
				} else if inQuote != nil && charCode == *inQuote {
					// Exiting the current quoted string
					if interpLoopCount <= 20 {
						fmt.Printf("[DEBUG] _consumeWithInterpolation: exiting quote, char='%s', inQuote=%d, peek after=%d, isTextEnd()=%v\n", char, *inQuote, t.cursor.Peek(), isTextEnd())
					}
					inQuote = nil
					// After exiting quote, continue the loop to check for }} before breaking
					// This matches TypeScript behavior where exiting quote doesn't immediately break
					// We need to check for }} first, then check isTextEnd() if }} is not found
				} else if !inComment && inQuote == nil && (charCode == core.CharSQ || charCode == core.CharDQ) {
					// Entering a new quoted string
					if interpLoopCount <= 20 {
						fmt.Printf("[DEBUG] _consumeWithInterpolation: entering quote, char='%s', charCode=%d\n", char, charCode)
					}
					inQuote = &charCode
				}
			}
			t.inInterpolation = wasInInterpolation

			expression := t._processCarriageReturns(strings.Join(interpolationParts, ""))
			// When we hit EOF without finding a closing interpolation marker,
			// we don't include the end marker (matches TypeScript behavior)
			fmt.Printf("[DEBUG] _consumeWithInterpolation: ending interpolation, foundEnd=%v, expression=%q, startMarker=%q, INTERPOLATION.end=%q\n", foundEnd, expression, startMarker, INTERPOLATION.end)
			if foundEnd {
				t._endToken([]string{startMarker, expression, INTERPOLATION.end}, nil)
			} else {
				// No end marker found - just include start marker and expression
				t._endToken([]string{startMarker, expression}, nil)
			}
			// If we encountered a tag start, break out of _consumeWithInterpolation entirely
			// (matches TypeScript behavior where it returns early)
			// But we need to create an empty TEXT token before returning
			if tagStartEncountered {
				t.inInterpolation = wasInInterpolation
				// Create an empty TEXT token before returning (matches TypeScript behavior)
				t._beginToken(textTokenType, nil)
				t._endToken([]string{""}, nil)
				return
			}
			t._beginToken(textTokenType, nil)
		} else {
			if interpIterationCount <= 20 {
				fmt.Printf("[DEBUG] _consumeWithInterpolation: _attemptStr(INTERPOLATION.start) did NOT match, peek=%d ('%c'), consuming as text\n",
					t.cursor.Peek(), t.cursor.Peek())
			}
			if t.cursor.Peek() == core.CharAMPERSAND {
				// Entity detected - check if it's a valid entity with semicolon
				// If not, treat it as text and read into current token
				entityStart := t.cursor.Clone()
				t.cursor.Advance() // Skip &
				if t._attemptCharCode(core.CharHASH) {
					// Numeric entity - always consume as entity (even if invalid) to report errors
					t.cursor = entityStart
					t._endToken([]string{t._normalizeCarriageReturns(strings.Join(parts, ""))}, nil)
					parts = []string{}
					t._consumeEntity(textTokenType)
					t._beginToken(textTokenType, nil)
					continue
				} else {
					// Named entity - check if it has semicolon
					nameStart := t.cursor.Clone()
					t._attemptCharCodeUntilFn(isNamedEntityEnd)
					if t.cursor.Peek() != core.CharSEMICOLON {
						// No semicolon - treat as text, read & and entity name into current token
						entityName := t.cursor.GetChars(nameStart)
						// Read & and entity name into parts
						parts = append(parts, "&"+entityName)
						// Cursor is already at position after entity name (after _attemptCharCodeUntilFn)
						// So we can continue to next iteration, which will read the next char
						// Don't reset cursor to entityStart - that would cause infinite loop
						continue
					}
					// Has semicolon - consume as entity
					t.cursor = entityStart
					t._endToken([]string{t._normalizeCarriageReturns(strings.Join(parts, ""))}, nil)
					parts = []string{}
					t._consumeEntity(textTokenType)
					t._beginToken(textTokenType, nil)
					continue
				}
			} else if isTagStart() {
				// We've reached the start of a tag, so we need to end the text token.
				// However, we don't consume the tag start here, as it will be handled by the main loop.
				break
			} else if currentPeek == core.CharEOF {
				// EOF reached, break to avoid infinite loop
				break
			} else {
				char := t._readChar()
				parts = append(parts, char)
				if interpIterationCount <= 20 {
					fmt.Printf("[DEBUG] _consumeWithInterpolation: consumed char='%s' as text, parts=%v\n", char, parts)
				}
			}
		}
	}
	t._endToken([]string{t._normalizeCarriageReturns(strings.Join(parts, ""))}, nil)
}

func (t *Tokenizer) _isTextEnd() bool {
	if t.cursor.Peek() == core.CharEOF || t._isTagStart() {
		return true
	}

	// Check for expansion forms (matches TypeScript _isTextEnd logic)
	if t.tokenizeIcu && !t.inInterpolation {
		// Check for expansion form start - even when in expansion case (for nested expansion forms)
		// This matches TypeScript behavior where isExpansionFormStart() is checked without
		// checking _isInExpansionCase()
		if t._isExpansionFormStart() {
			// start of an expansion form (including nested expansion forms)
			fmt.Printf("[DEBUG] _isTextEnd: found expansion form start, returning true\n")
			return true
		}

		if t._isExpansionCaseStart() {
			// start of an expansion case
			fmt.Printf("[DEBUG] _isTextEnd: found expansion case start, returning true\n")
			return true
		}

		if t.cursor.Peek() == core.CharRBRACE {
			isInCase := t._isInExpansionCase()
			isInForm := t._isInExpansionForm()
			fmt.Printf("[DEBUG] _isTextEnd: found '}', isInExpansionCase=%v, isInExpansionForm=%v, stackLen=%d\n",
				isInCase, isInForm, len(t.expansionCaseStack))
			if len(t.expansionCaseStack) > 0 {
				fmt.Printf("[DEBUG] _isTextEnd: stack contents: %v\n", t.expansionCaseStack)
			}
			if isInCase {
				// end of an expansion case
				fmt.Printf("[DEBUG] _isTextEnd: found expansion case end, returning true\n")
				return true
			}
			if isInForm {
				// end of an expansion form
				fmt.Printf("[DEBUG] _isTextEnd: found expansion form end, returning true\n")
				return true
			}
		}
	}

	// Check for blocks and @let declarations
	if t.tokenizeBlocks && !t.inInterpolation && !t._isInExpansionCase() && !t._isInExpansionForm() {
		if t._isBlockStart() {
			return true
		}
		// Check for @let declaration start
		if t.tokenizeLet && t._isLetStart() {
			return true
		}
		// Don't treat '}' as text end when in interpolation context
		if t.cursor.Peek() == core.CharRBRACE {
			return true
		}
	}

	return false
}

func (t *Tokenizer) _isTagStart() bool {
	if t.cursor.Peek() == core.CharLT {
		// We need to check if it's actually a tag start or just a '<' character in text.
		// A tag start is '<' followed by a letter, '/', '!', or '?'.
		// We need to peek ahead to check the next character.
		// Clone the cursor to peek ahead without advancing
		peekCursor := t.cursor.Clone()
		peekCursor.Advance()
		nextChar := peekCursor.Peek()
		// Valid tag starts: <!, </, <letter, <?, or EOF (incomplete tag)
		return nextChar == core.CharBANG ||
			nextChar == core.CharSLASH ||
			core.IsAsciiLetter(nextChar) ||
			nextChar == core.CharQUESTION ||
			nextChar == core.CharEOF
	}
	return false
}

func isNotWhitespace(code int) bool {
	return !core.IsWhitespace(code) || code == core.CharEOF
}

func isNameEnd(code int) bool {
	// Matches TypeScript: isWhitespace(code) || code === chars.$GT || code === chars.$SLASH || code === chars.$SQ || code === chars.$DQ || code === chars.$EQ || code === chars.$EOF || code === chars.$LT
	return core.IsWhitespace(code) || code == core.CharGT || code == core.CharSLASH || code == core.CharSQ || code == core.CharDQ || code == core.CharEQ || code == core.CharEOF || code == core.CharLT
}

func isPrefixEnd(code int) bool {
	// Matches TypeScript: (code < chars.$a || chars.$z < code) && (code < chars.$A || chars.$Z < code) && (code < chars.$0 || code > chars.$9)
	// This returns true if code is NOT a letter or digit
	return (code < core.CharLowerA || code > core.CharLowerZ) && (code < core.CharA || code > core.CharZ) && (code < core.Char0 || code > core.Char9)
}

// _getLetDeclarationName extracts the variable name from a @let declaration
func (t *Tokenizer) _getLetDeclarationName() string {
	nameCursor := t.cursor.Clone()
	allowDigit := false

	t._attemptCharCodeUntilFn(func(code int) bool {
		if core.IsAsciiLetter(code) || code == core.CharDollar || code == core.CharUnderscore ||
			(allowDigit && core.IsDigit(code)) {
			// `@let` names can't start with a digit, but digits are valid anywhere else in the name.
			allowDigit = true
			return false
		}
		return true
	})

	return strings.TrimSpace(t.cursor.GetChars(nameCursor))
}

// _consumeLetDeclarationValue consumes the value part of a @let declaration
func (t *Tokenizer) _consumeLetDeclarationValue() {
	start := t.cursor.Clone()
	t._beginToken(TokenTypeLET_VALUE, start)

	iteration := 0
	for t.cursor.Peek() != core.CharEOF {
		iteration++
		if iteration > 1000 {
			panic("_consumeLetDeclarationValue: infinite loop detected")
		}
		char := t.cursor.Peek()
		if iteration <= 20 {
			fmt.Printf("[DEBUG] _consumeLetDeclarationValue: iteration=%d, char=%d ('%c')\n", iteration, char, func() rune {
				if char >= 32 && char < 127 {
					return rune(char)
				}
				return '?'
			}())
		}

		// `@let` declarations terminate with a semicolon.
		if char == core.CharSEMICOLON {
			fmt.Printf("[DEBUG] _consumeLetDeclarationValue: found semicolon, breaking\n")
			break
		}

		// If we hit a quote, skip over its content since we don't care what's inside.
		if core.IsQuote(char) {
			if iteration <= 20 {
				fmt.Printf("[DEBUG] _consumeLetDeclarationValue: found quote %d ('%c'), skipping content\n", char, rune(char))
			}
			t.cursor.Advance() // Skip opening quote
			t._attemptCharCodeUntilFn(func(inner int) bool {
				if inner == core.CharBACKSLASH {
					t.cursor.Advance() // Skip escaped character
					return false
				}
				if inner == char {
					if iteration <= 20 {
						fmt.Printf("[DEBUG] _consumeLetDeclarationValue: found closing quote %d ('%c'), stopping\n", inner, rune(inner))
					}
				}
				return inner == char // Found closing quote
			})
			// Advance past the closing quote (matches TypeScript: this._cursor.advance() at line 465)
			peekAfterAttempt := t.cursor.Peek()
			if iteration <= 20 {
				fmt.Printf("[DEBUG] _consumeLetDeclarationValue: after _attemptCharCodeUntilFn, peek=%d ('%c')\n", peekAfterAttempt, func() rune {
					if peekAfterAttempt >= 32 && peekAfterAttempt < 127 {
						return rune(peekAfterAttempt)
					}
					return '?'
				}())
			}
			t.cursor.Advance()
			if iteration <= 20 {
				fmt.Printf("[DEBUG] _consumeLetDeclarationValue: after advancing past closing quote, peek=%d ('%c')\n", t.cursor.Peek(), func() rune {
					peek := t.cursor.Peek()
					if peek >= 32 && peek < 127 {
						return rune(peek)
					}
					return '?'
				}())
			}
		} else {
			// Advance past the current character (matches TypeScript: this._cursor.advance() at line 465)
			t.cursor.Advance()
		}
	}

	valueContent := t.cursor.GetChars(start)
	fmt.Printf("[DEBUG] _consumeLetDeclarationValue: END, valueContent=%q, length=%d\n", valueContent, len(valueContent))
	// Debug: print each character
	fmt.Printf("[DEBUG] _consumeLetDeclarationValue: valueContent bytes: ")
	for i, b := range []byte(valueContent) {
		if i < 50 {
			fmt.Printf("%d ", b)
		}
	}
	fmt.Printf("\n")
	t._endToken([]string{valueContent}, nil)
}

// _getBlockName extracts the block name (e.g., "if", "else if", "for")
func (t *Tokenizer) _getBlockName() string {
	// This allows us to capture something like `@else if`, but not `@ if`.
	spacesInNameAllowed := false
	nameCursor := t.cursor.Clone()

	t._attemptCharCodeUntilFn(func(code int) bool {
		if core.IsWhitespace(code) {
			return !spacesInNameAllowed
		}
		if isBlockNameChar(code) {
			spacesInNameAllowed = true
			return false
		}
		return true
	})

	return strings.TrimSpace(t.cursor.GetChars(nameCursor))
}

// _consumeBlockParameters consumes block parameters within parentheses
func (t *Tokenizer) _consumeBlockParameters() {
	// Trim the whitespace until the first parameter.
	t._attemptCharCodeUntilFn(isBlockParameterChar)

	for t.cursor.Peek() != core.CharRPAREN && t.cursor.Peek() != core.CharEOF {
		t._beginToken(TokenTypeBLOCK_PARAMETER, nil)
		start := t.cursor.Clone()
		var inQuote *int = nil
		openParens := 0

		// Consume the parameter until the next semicolon or closing paren.
		// Note that we skip over semicolons/parens inside of strings.
		for (t.cursor.Peek() != core.CharSEMICOLON && t.cursor.Peek() != core.CharEOF) || inQuote != nil {
			char := t.cursor.Peek()

			// Skip to the next character if it was escaped.
			if char == core.CharBACKSLASH {
				t.cursor.Advance()
			} else if inQuote != nil && char == *inQuote {
				inQuote = nil
			} else if inQuote == nil && core.IsQuote(char) {
				inQuote = &char
			} else if char == core.CharLPAREN && inQuote == nil {
				openParens++
			} else if char == core.CharRPAREN && inQuote == nil {
				if openParens == 0 {
					break
				} else if openParens > 0 {
					openParens--
				}
			}

			t.cursor.Advance()
		}

		t._endToken([]string{t.cursor.GetChars(start)}, nil)

		// Advance past the semicolon if present
		if t.cursor.Peek() == core.CharSEMICOLON {
			t.cursor.Advance()
		}

		// Skip to the next parameter.
		t._attemptCharCodeUntilFn(isBlockParameterChar)
	}
}

func isBlockNameChar(code int) bool {
	return core.IsAsciiLetter(code) || core.IsDigit(code) || code == core.CharUnderscore
}

func isBlockParameterChar(code int) bool {
	return code != core.CharSEMICOLON && isNotWhitespace(code)
}

func isDigitEntityEnd(code int) bool {
	// Stop when we encounter anything that's not a digit or hex digit
	return !core.IsDigit(code) && !core.IsAsciiHexDigit(code)
}

func isNamedEntityEnd(code int) bool {
	// Stop when we encounter anything that's not a letter or digit
	return !core.IsAsciiLetter(code) && !core.IsDigit(code)
}

func (t *Tokenizer) _createError(msg string, span *util.ParseSourceSpan) *util.ParseError {
	if t._isInExpansionForm() {
		msg += ` (Do you have an unescaped "{" in your template? Use "{{ '{' }}") to escape it.)`
	}
	t.currentTokenStart = nil
	t.currentTokenType = -1
	return util.NewParseError(span, msg)
}

func (t *Tokenizer) handleError(e interface{}) {
	if cursorErr, ok := e.(*CursorError); ok {
		t.errors = append(t.errors, t._createError(cursorErr.Msg, t.cursor.GetSpan(cursorErr.Cursor, t.leadingTriviaCodePoints)))
	} else if parseErr, ok := e.(*util.ParseError); ok {
		t.errors = append(t.errors, parseErr)
	} else if errStr, ok := e.(string); ok {
		// Handle string panics (like "Unexpected character \"EOF\"")
		t.errors = append(t.errors, t._createError(errStr, t.cursor.GetSpan(t.cursor.Clone(), t.leadingTriviaCodePoints)))
	} else {
		panic(e)
	}
}

func (t *Tokenizer) _attemptCharCode(charCode int) bool {
	if t.cursor.Peek() == charCode {
		t.cursor.Advance()
		return true
	}
	return false
}

func (t *Tokenizer) _attemptCharCodeCaseInsensitive(charCode int) bool {
	if compareCharCodeCaseInsensitive(t.cursor.Peek(), charCode) {
		t.cursor.Advance()
		return true
	}
	return false
}

func (t *Tokenizer) _requireCharCode(charCode int) {
	location := t.cursor.Clone()
	if !t._attemptCharCode(charCode) {
		panic(&CursorError{
			Msg:    _unexpectedCharacterErrorMsg(t.cursor.Peek()),
			Cursor: location,
		})
	}
}

func (t *Tokenizer) _attemptStr(charsStr string) bool {
	length := len(charsStr)
	if t.cursor.CharsLeft() < length {
		return false
	}
	initialPosition := t.cursor.Clone()
	for i := 0; i < length; i++ {
		if !t._attemptCharCode(int(charsStr[i])) {
			// If attempting to parse the string fails, we want to reset the parser
			// to where it was before the attempt
			t.cursor = initialPosition
			return false
		}
	}
	return true
}

func (t *Tokenizer) _attemptStrCaseInsensitive(charsStr string) bool {
	for i := 0; i < len(charsStr); i++ {
		if !t._attemptCharCodeCaseInsensitive(int(charsStr[i])) {
			return false
		}
	}
	return true
}

func (t *Tokenizer) _requireStr(charsStr string) {
	location := t.cursor.Clone()
	if !t._attemptStr(charsStr) {
		panic(&CursorError{
			Msg:    _unexpectedCharacterErrorMsg(t.cursor.Peek()),
			Cursor: location,
		})
	}
}

func (t *Tokenizer) _attemptCharCodeUntilFn(predicate func(code int) bool) {
	iteration := 0
	for !predicate(t.cursor.Peek()) {
		peek := t.cursor.Peek()
		if iteration < 50 && (peek == core.CharLF || peek == core.CharCR) {
			fmt.Printf("[DEBUG] _attemptCharCodeUntilFn: iteration=%d, peek=%d (newline), predicate returned false, advancing\n", iteration, peek)
		}
		// Debug: log every char when iteration < 50
		if iteration < 50 {
			fmt.Printf("[DEBUG] _attemptCharCodeUntilFn: iteration=%d, peek=%d ('%c'), predicate returned false, advancing\n", iteration, peek, func() rune {
				if peek >= 32 && peek < 127 {
					return rune(peek)
				}
				return '?'
			}())
		}
		t.cursor.Advance()
		iteration++
		if iteration > 1000 {
			panic("_attemptCharCodeUntilFn: infinite loop detected")
		}
	}
	peek := t.cursor.Peek()
	if peek == core.CharLF || peek == core.CharCR {
		fmt.Printf("[DEBUG] _attemptCharCodeUntilFn: stopped at newline, peek=%d, predicate returned true\n", peek)
	}
}

func (t *Tokenizer) _requireCharCodeUntilFn(predicate func(code int) bool, len int) {
	start := t.cursor.Clone()
	t._attemptCharCodeUntilFn(predicate)
	if t.cursor.Diff(start) < len {
		panic(&CursorError{
			Msg:    _unexpectedCharacterErrorMsg(t.cursor.Peek()),
			Cursor: start,
		})
	}
}

func (t *Tokenizer) _attemptUntilChar(char int) {
	for t.cursor.Peek() != char {
		t.cursor.Advance()
	}
}

func (t *Tokenizer) _readChar() string {
	// Don't rely upon reading directly from `_input` as the actual char value
	// may have been generated from an escape sequence.
	char := string(rune(t.cursor.Peek()))
	t.cursor.Advance()
	return char
}

func (t *Tokenizer) _peekStr(charsStr string) bool {
	length := len(charsStr)
	if t.cursor.CharsLeft() < length {
		return false
	}
	cursor := t.cursor.Clone()
	for i := 0; i < length; i++ {
		if cursor.Peek() != int(charsStr[i]) {
			return false
		}
		cursor.Advance()
	}
	return true
}

func compareCharCodeCaseInsensitive(code1, code2 int) bool {
	return toUpperCaseCharCode(code1) == toUpperCaseCharCode(code2)
}

func toUpperCaseCharCode(code int) int {
	if code >= 'a' && code <= 'z' {
		return code - 32
	}
	return code
}

func _unexpectedCharacterErrorMsg(charCode int) string {
	char := string(rune(charCode))
	if charCode == 0 {
		char = "EOF"
	}
	return fmt.Sprintf("Unexpected character \"%s\"", char)
}

func (t *Tokenizer) _isInExpansionForm() bool {
	return len(t.expansionCaseStack) > 0 && t.expansionCaseStack[len(t.expansionCaseStack)-1] == TokenTypeEXPANSION_FORM_START
}

func (t *Tokenizer) _isInExpansionCase() bool {
	return len(t.expansionCaseStack) > 0 && t.expansionCaseStack[len(t.expansionCaseStack)-1] == TokenTypeEXPANSION_CASE_EXP_START
}
