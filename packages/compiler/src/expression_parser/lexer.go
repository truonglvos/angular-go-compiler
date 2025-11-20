package expression_parser

import (
	"ngc-go/packages/compiler/src/core"
	"strconv"
	"strings"
)

// TokenType represents the type of a token
type TokenType int

const (
	TokenTypeCharacter TokenType = iota
	TokenTypeIdentifier
	TokenTypePrivateIdentifier
	TokenTypeKeyword
	TokenTypeString
	TokenTypeOperator
	TokenTypeNumber
	TokenTypeRegExpBody
	TokenTypeRegExpFlags
	TokenTypeError
)

// StringTokenKind represents the kind of a string token
type StringTokenKind int

const (
	StringTokenKindPlain StringTokenKind = iota
	StringTokenKindTemplateLiteralPart
	StringTokenKindTemplateLiteralEnd
)

var keywords = []string{
	"var",
	"let",
	"as",
	"null",
	"undefined",
	"true",
	"false",
	"if",
	"else",
	"this",
	"typeof",
	"void",
	"in",
}

// Token represents a token in the expression
type Token struct {
	Index    int
	End      int
	Type     TokenType
	NumValue float64
	StrValue string
	// StringKind is only valid for String tokens
	StringKind StringTokenKind
}

// NewToken creates a new Token
func NewToken(index, end int, typ TokenType, numValue float64, strValue string) *Token {
	return &Token{
		Index:    index,
		End:      end,
		Type:     typ,
		NumValue: numValue,
		StrValue: strValue,
	}
}

// IsCharacter checks if the token is a character with the given code
func (t *Token) IsCharacter(code int) bool {
	return t.Type == TokenTypeCharacter && int(t.NumValue) == code
}

// IsNumber checks if the token is a number
func (t *Token) IsNumber() bool {
	return t.Type == TokenTypeNumber
}

// IsString checks if the token is a string
func (t *Token) IsString() bool {
	return t.Type == TokenTypeString
}

// IsOperator checks if the token is an operator with the given value
func (t *Token) IsOperator(operator string) bool {
	return t.Type == TokenTypeOperator && t.StrValue == operator
}

// IsIdentifier checks if the token is an identifier
func (t *Token) IsIdentifier() bool {
	return t.Type == TokenTypeIdentifier
}

// IsPrivateIdentifier checks if the token is a private identifier
func (t *Token) IsPrivateIdentifier() bool {
	return t.Type == TokenTypePrivateIdentifier
}

// IsKeyword checks if the token is a keyword
func (t *Token) IsKeyword() bool {
	return t.Type == TokenTypeKeyword
}

// IsKeywordLet checks if the token is the 'let' keyword
func (t *Token) IsKeywordLet() bool {
	return t.Type == TokenTypeKeyword && t.StrValue == "let"
}

// IsKeywordAs checks if the token is the 'as' keyword
func (t *Token) IsKeywordAs() bool {
	return t.Type == TokenTypeKeyword && t.StrValue == "as"
}

// IsKeywordNull checks if the token is the 'null' keyword
func (t *Token) IsKeywordNull() bool {
	return t.Type == TokenTypeKeyword && t.StrValue == "null"
}

// IsKeywordUndefined checks if the token is the 'undefined' keyword
func (t *Token) IsKeywordUndefined() bool {
	return t.Type == TokenTypeKeyword && t.StrValue == "undefined"
}

// IsKeywordTrue checks if the token is the 'true' keyword
func (t *Token) IsKeywordTrue() bool {
	return t.Type == TokenTypeKeyword && t.StrValue == "true"
}

// IsKeywordFalse checks if the token is the 'false' keyword
func (t *Token) IsKeywordFalse() bool {
	return t.Type == TokenTypeKeyword && t.StrValue == "false"
}

// IsKeywordThis checks if the token is the 'this' keyword
func (t *Token) IsKeywordThis() bool {
	return t.Type == TokenTypeKeyword && t.StrValue == "this"
}

// IsKeywordTypeof checks if the token is the 'typeof' keyword
func (t *Token) IsKeywordTypeof() bool {
	return t.Type == TokenTypeKeyword && t.StrValue == "typeof"
}

// IsKeywordVoid checks if the token is the 'void' keyword
func (t *Token) IsKeywordVoid() bool {
	return t.Type == TokenTypeKeyword && t.StrValue == "void"
}

// IsKeywordIn checks if the token is the 'in' keyword
func (t *Token) IsKeywordIn() bool {
	return t.Type == TokenTypeKeyword && t.StrValue == "in"
}

// IsError checks if the token is an error
func (t *Token) IsError() bool {
	return t.Type == TokenTypeError
}

// IsRegExpBody checks if the token is a regexp body
func (t *Token) IsRegExpBody() bool {
	return t.Type == TokenTypeRegExpBody
}

// IsRegExpFlags checks if the token is a regexp flags
func (t *Token) IsRegExpFlags() bool {
	return t.Type == TokenTypeRegExpFlags
}

// ToNumber converts the token to a number
func (t *Token) ToNumber() float64 {
	if t.Type == TokenTypeNumber {
		return t.NumValue
	}
	return -1
}

// IsTemplateLiteralPart checks if the token is a template literal part
func (t *Token) IsTemplateLiteralPart() bool {
	return t.IsString() && t.Kind() == StringTokenKindTemplateLiteralPart
}

// IsTemplateLiteralEnd checks if the token is a template literal end
func (t *Token) IsTemplateLiteralEnd() bool {
	return t.IsString() && t.Kind() == StringTokenKindTemplateLiteralEnd
}

// IsTemplateLiteralInterpolationStart checks if the token is a template literal interpolation start
func (t *Token) IsTemplateLiteralInterpolationStart() bool {
	return t.IsOperator("${")
}

// Kind returns the kind of the string token (only valid for StringToken)
func (t *Token) Kind() StringTokenKind {
	return t.StringKind
}

// String returns the string representation of the token
func (t *Token) String() string {
	switch t.Type {
	case TokenTypeCharacter, TokenTypeIdentifier, TokenTypeKeyword, TokenTypeOperator, TokenTypePrivateIdentifier, TokenTypeString, TokenTypeError, TokenTypeRegExpBody, TokenTypeRegExpFlags:
		return t.StrValue
	case TokenTypeNumber:
		return strconv.FormatFloat(t.NumValue, 'f', -1, 64)
	default:
		return ""
	}
}

// StringToken extends Token with a kind field
type StringToken struct {
	*Token
	kind StringTokenKind
}

// NewStringToken creates a new StringToken
func NewStringToken(index, end int, strValue string, kind StringTokenKind) *StringToken {
	token := NewToken(index, end, TokenTypeString, 0, strValue)
	token.StringKind = kind
	return &StringToken{
		Token: token,
		kind:  kind,
	}
}

// Kind returns the kind of the string token
func (s *StringToken) Kind() StringTokenKind {
	return s.kind
}

// Lexer tokenizes expressions
type Lexer struct{}

// NewLexer creates a new Lexer
func NewLexer() *Lexer {
	return &Lexer{}
}

// Tokenize tokenizes the given text
func (l *Lexer) Tokenize(text string) []*Token {
	scanner := newScanner(text)
	return scanner.scan()
}

// EOF represents the end of file token
var EOF = NewToken(-1, -1, TokenTypeCharacter, 0, "")

type scanner struct {
	input      string
	length     int
	peek       rune
	index      int
	tokens     []*Token
	braceStack []string // 'interpolation' or 'expression'
}

func newScanner(input string) *scanner {
	s := &scanner{
		input:      input,
		length:     len(input),
		index:      -1,
		tokens:     []*Token{},
		braceStack: []string{},
	}
	s.advance()
	return s
}

func (s *scanner) advance() {
	s.index++
	if s.index >= s.length {
		s.peek = core.CharEOF
	} else {
		s.peek = rune(s.input[s.index])
	}
}

func (s *scanner) scan() []*Token {
	token := s.scanToken()
	for token != nil {
		s.tokens = append(s.tokens, token)
		token = s.scanToken()
	}
	return s.tokens
}

func (s *scanner) scanToken() *Token {
	input := s.input
	length := s.length
	peek := s.peek
	index := s.index

	// Skip whitespace
	for int(peek) <= core.CharSPACE {
		index++
		if index >= length {
			peek = core.CharEOF
			break
		} else {
			peek = rune(input[index])
		}
	}

	s.peek = peek
	s.index = index

	if index >= length {
		return nil
	}

	// Handle identifiers and numbers
	if isIdentifierStart(peek) {
		return s.scanIdentifier()
	}

	if core.IsDigit(int(peek)) {
		return s.scanNumber(index)
	}

	start := index
	switch int(peek) {
	case core.CharPERIOD:
		s.advance()
		if core.IsDigit(int(s.peek)) {
			return s.scanNumber(start)
		}
		return newCharacterToken(start, s.index, core.CharPERIOD)
	case core.CharLPAREN, core.CharRPAREN, core.CharLBRACKET, core.CharRBRACKET, core.CharCOMMA, core.CharCOLON, core.CharSEMICOLON:
		return s.scanCharacter(start, peek)
	case core.CharLBRACE:
		return s.scanOpenBrace(start, peek)
	case core.CharRBRACE:
		return s.scanCloseBrace(start, peek)
	case core.CharSQ, core.CharDQ:
		return s.scanString()
	case core.CharBT:
		s.advance()
		return s.scanTemplateLiteralPart(start)
	case core.CharHASH:
		return s.scanPrivateIdentifier()
	case core.CharPLUS:
		return s.scanComplexOperator(start, "+", core.CharEQ, "=")
	case core.CharMINUS:
		return s.scanComplexOperator(start, "-", core.CharEQ, "=")
	case core.CharSLASH:
		if s.isStartOfRegex() {
			return s.scanRegex(index)
		}
		return s.scanComplexOperator(start, "/", core.CharEQ, "=")
	case core.CharPERCENT:
		return s.scanComplexOperator(start, "%", core.CharEQ, "=")
	case core.CharCARET:
		return s.scanOperator(start, "^")
	case core.CharSTAR:
		return s.scanStar(start)
	case core.CharQUESTION:
		return s.scanQuestion(start)
	case core.CharLT, core.CharGT:
		return s.scanComplexOperator(start, string(peek), core.CharEQ, "=")
	case core.CharBANG, core.CharEQ:
		return s.scanComplexOperator(start, string(peek), core.CharEQ, "=", core.CharEQ)
	case core.CharAMPERSAND:
		return s.scanComplexOperator(start, "&", core.CharAMPERSAND, "&", core.CharEQ)
	case core.CharBAR:
		return s.scanComplexOperator(start, "|", core.CharBAR, "|", core.CharEQ)
	case core.CharNBSP:
		for core.IsWhitespace(int(s.peek)) {
			s.advance()
		}
		return s.scanToken()
	}

	s.advance()
	return s.error("Unexpected character ["+string(peek)+"]", 0)
}

func (s *scanner) scanCharacter(start int, code rune) *Token {
	s.advance()
	return newCharacterToken(start, s.index, code)
}

func (s *scanner) scanOperator(start int, str string) *Token {
	s.advance()
	return newOperatorToken(start, s.index, str)
}

func (s *scanner) scanOpenBrace(start int, code rune) *Token {
	s.braceStack = append(s.braceStack, "expression")
	s.advance()
	return newCharacterToken(start, s.index, code)
}

func (s *scanner) scanCloseBrace(start int, code rune) *Token {
	s.advance()

	if len(s.braceStack) > 0 {
		currentBrace := s.braceStack[len(s.braceStack)-1]
		s.braceStack = s.braceStack[:len(s.braceStack)-1]
		if currentBrace == "interpolation" {
			s.tokens = append(s.tokens, newCharacterToken(start, s.index, core.CharRBRACE))
			return s.scanTemplateLiteralPart(s.index)
		}
	}

	return newCharacterToken(start, s.index, code)
}

func (s *scanner) scanComplexOperator(start int, one string, twoCode int, two string, threeCode ...int) *Token {
	s.advance()
	str := one
	if int(s.peek) == twoCode {
		s.advance()
		str += two
	}
	if len(threeCode) > 0 && int(s.peek) == threeCode[0] {
		s.advance()
		str += string(rune(threeCode[0]))
	}
	return newOperatorToken(start, s.index, str)
}

func (s *scanner) scanIdentifier() *Token {
	start := s.index
	s.advance()
	for isIdentifierPart(s.peek) {
		s.advance()
	}
	str := s.input[start:s.index]
	for _, keyword := range keywords {
		if str == keyword {
			return newKeywordToken(start, s.index, str)
		}
	}
	return newIdentifierToken(start, s.index, str)
}

func (s *scanner) scanPrivateIdentifier() *Token {
	start := s.index
	s.advance()
	if !isIdentifierStart(s.peek) {
		return s.error("Invalid character [#]", -1)
	}
	for isIdentifierPart(s.peek) {
		s.advance()
	}
	identifierName := s.input[start:s.index]
	return newPrivateIdentifierToken(start, s.index, identifierName)
}

func (s *scanner) scanNumber(start int) *Token {
	simple := s.index == start
	hasSeparators := false
	s.advance() // Skip initial digit
	for {
		if core.IsDigit(int(s.peek)) {
			// Do nothing
		} else if s.peek == core.CharUnderscore {
			// Separators are only valid when they're surrounded by digits
			if s.index == 0 || s.index >= s.length-1 || !core.IsDigit(int(rune(s.input[s.index-1]))) || !core.IsDigit(int(rune(s.input[s.index+1]))) {
				return s.error("Invalid numeric separator", 0)
			}
			hasSeparators = true
		} else if s.peek == core.CharPERIOD {
			simple = false
		} else if isExponentStart(s.peek) {
			s.advance()
			if isExponentSign(s.peek) {
				s.advance()
			}
			if !core.IsDigit(int(s.peek)) {
				return s.error("Invalid exponent", -1)
			}
			simple = false
		} else {
			break
		}
		s.advance()
	}

	str := s.input[start:s.index]
	if hasSeparators {
		str = strings.ReplaceAll(str, "_", "")
	}
	var value float64
	if simple {
		val, err := strconv.ParseInt(str, 0, 64)
		if err != nil {
			value = 0
		} else {
			value = float64(val)
		}
	} else {
		val, err := strconv.ParseFloat(str, 64)
		if err != nil {
			value = 0
		} else {
			value = val
		}
	}
	return newNumberToken(start, s.index, value)
}

func (s *scanner) scanString() *Token {
	start := s.index
	quote := s.peek
	s.advance() // Skip initial quote

	buffer := ""
	marker := s.index
	input := s.input

	for s.peek != quote {
		if s.peek == core.CharBACKSLASH {
			result := s.scanStringBackslash(buffer, marker)
			if errToken, ok := result.(*Token); ok && errToken.Type == TokenTypeError {
				return errToken
			}
			buffer = result.(string)
			marker = s.index
		} else if s.peek == core.CharEOF {
			return s.error("Unterminated quote", 0)
		} else {
			s.advance()
		}
	}

	last := input[marker:s.index]
	s.advance() // Skip terminating quote

	return NewStringToken(start, s.index, buffer+last, StringTokenKindPlain).Token
}

func (s *scanner) scanQuestion(start int) *Token {
	s.advance()
	operator := "?"
	// `a ?? b` or `a ??= b`
	if s.peek == core.CharQUESTION {
		operator += "?"
		s.advance()
		if s.peek == core.CharEQ {
			operator += "="
			s.advance()
		}
	} else if s.peek == core.CharPERIOD {
		// `a?.b`
		operator += "."
		s.advance()
	}
	return newOperatorToken(start, s.index, operator)
}

func (s *scanner) scanTemplateLiteralPart(start int) *Token {
	buffer := ""
	marker := s.index

	for s.peek != core.CharBT {
		if s.peek == core.CharBACKSLASH {
			result := s.scanStringBackslash(buffer, marker)
			if errToken, ok := result.(*Token); ok && errToken.Type == TokenTypeError {
				return errToken
			}
			buffer = result.(string)
			marker = s.index
		} else if s.peek == core.CharDollar {
			dollar := s.index
			s.advance()
			if s.peek == core.CharLBRACE {
				s.braceStack = append(s.braceStack, "interpolation")
				s.tokens = append(s.tokens, NewStringToken(
					start,
					dollar,
					buffer+s.input[marker:dollar],
					StringTokenKindTemplateLiteralPart,
				).Token)
				s.advance()
				return newOperatorToken(dollar, s.index, s.input[dollar:s.index])
			}
		} else if s.peek == core.CharEOF {
			return s.error("Unterminated template literal", 0)
		} else {
			s.advance()
		}
	}

	last := s.input[marker:s.index]
	s.advance()
	return NewStringToken(start, s.index, buffer+last, StringTokenKindTemplateLiteralEnd).Token
}

func (s *scanner) error(message string, offset int) *Token {
	position := s.index + offset
	return newErrorToken(
		position,
		s.index,
		"Lexer Error: "+message+" at column "+strconv.Itoa(position)+" in expression ["+s.input+"]",
	)
}

func (s *scanner) scanStringBackslash(buffer string, marker int) interface{} {
	buffer += s.input[marker:s.index]
	var unescapedCode rune
	s.advance()
	if s.peek == core.CharLowerU {
		// 4 character hex code for unicode character
		if s.index+5 > s.length {
			return s.error("Invalid unicode escape", 0)
		}
		hex := s.input[s.index+1 : s.index+5]
		val, err := strconv.ParseInt(hex, 16, 32)
		if err != nil {
			return s.error("Invalid unicode escape [\\u"+hex+"]", 0)
		}
		unescapedCode = rune(val)
		for i := 0; i < 5; i++ {
			s.advance()
		}
	} else {
		unescapedCode = unescape(s.peek)
		s.advance()
	}
	buffer += string(unescapedCode)
	return buffer
}

func (s *scanner) scanStar(start int) *Token {
	s.advance()
	operator := "*"
	// `*`, `**`, `**=` or `*=`
	if s.peek == core.CharSTAR {
		operator += "*"
		s.advance()
		if s.peek == core.CharEQ {
			operator += "="
			s.advance()
		}
	} else if s.peek == core.CharEQ {
		operator += "="
		s.advance()
	}
	return newOperatorToken(start, s.index, operator)
}

func (s *scanner) isStartOfRegex() bool {
	if len(s.tokens) == 0 {
		return true
	}

	prevToken := s.tokens[len(s.tokens)-1]

	// If a slash is preceded by a `!` operator, we need to distinguish whether it's a
	// negation or a non-null assertion. Regexes can only be preceded by negations.
	if prevToken.IsOperator("!") {
		var beforePrevToken *Token
		if len(s.tokens) > 1 {
			beforePrevToken = s.tokens[len(s.tokens)-2]
		}
		isNegation := beforePrevToken == nil ||
			(beforePrevToken.Type != TokenTypeIdentifier &&
				!beforePrevToken.IsCharacter(core.CharRPAREN) &&
				!beforePrevToken.IsCharacter(core.CharRBRACKET))
		return isNegation
	}

	// Only consider the slash a regex if it's preceded either by:
	// - Any operator, aside from `!` which is special-cased above.
	// - Opening paren (e.g. `(/a/)`).
	// - Opening bracket (e.g. `[/a/]`).
	// - A comma (e.g. `[1, /a/]`).
	// - A colon (e.g. `{foo: /a/}`).
	return prevToken.Type == TokenTypeOperator ||
		prevToken.IsCharacter(core.CharLPAREN) ||
		prevToken.IsCharacter(core.CharLBRACKET) ||
		prevToken.IsCharacter(core.CharCOMMA) ||
		prevToken.IsCharacter(core.CharCOLON)
}

func (s *scanner) scanRegex(tokenStart int) *Token {
	s.advance()
	textStart := s.index
	inEscape := false
	inCharacterClass := false

	for {
		peek := s.peek

		if peek == core.CharEOF {
			return s.error("Unterminated regular expression", 0)
		}

		if inEscape {
			inEscape = false
		} else if peek == core.CharBACKSLASH {
			inEscape = true
		} else if peek == core.CharLBRACKET {
			inCharacterClass = true
		} else if peek == core.CharRBRACKET {
			inCharacterClass = false
		} else if peek == core.CharSLASH && !inCharacterClass {
			break
		}
		s.advance()
	}

	// Note that we want the text without the slashes,
	// but we still want the slashes to be part of the span.
	value := s.input[textStart:s.index]
	s.advance()
	bodyToken := newRegExpBodyToken(tokenStart, s.index, value)
	flagsToken := s.scanRegexFlags(s.index)

	if flagsToken != nil {
		s.tokens = append(s.tokens, bodyToken)
		return flagsToken
	}

	return bodyToken
}

func (s *scanner) scanRegexFlags(start int) *Token {
	if !core.IsAsciiLetter(int(s.peek)) {
		return nil
	}

	for core.IsAsciiLetter(int(s.peek)) {
		s.advance()
	}

	return newRegExpFlagsToken(start, s.index, s.input[start:s.index])
}

func isIdentifierStart(code rune) bool {
	return (core.CharA <= code && code <= core.CharZ) ||
		(core.CharLowerA <= code && code <= core.CharLowerZ) ||
		code == core.CharUnderscore ||
		code == core.CharDollar
}

func isIdentifierPart(code rune) bool {
	return core.IsAsciiLetter(int(code)) || core.IsDigit(int(code)) || code == core.CharUnderscore || code == core.CharDollar
}

func isExponentStart(code rune) bool {
	return code == core.CharE || code == core.CharLowerE
}

func isExponentSign(code rune) bool {
	return code == core.CharMINUS || code == core.CharPLUS
}

func unescape(code rune) rune {
	switch code {
	case core.CharLowerN:
		return core.CharLF
	case core.CharLowerF:
		return core.CharFF
	case core.CharLowerR:
		return core.CharCR
	case core.CharLowerT:
		return core.CharTAB
	case core.CharLowerV:
		return core.CharVTAB
	default:
		return code
	}
}

// Helper functions to create tokens
func newCharacterToken(index, end int, code rune) *Token {
	return NewToken(index, end, TokenTypeCharacter, float64(code), string(code))
}

func newIdentifierToken(index, end int, text string) *Token {
	return NewToken(index, end, TokenTypeIdentifier, 0, text)
}

func newPrivateIdentifierToken(index, end int, text string) *Token {
	return NewToken(index, end, TokenTypePrivateIdentifier, 0, text)
}

func newKeywordToken(index, end int, text string) *Token {
	return NewToken(index, end, TokenTypeKeyword, 0, text)
}

func newOperatorToken(index, end int, text string) *Token {
	return NewToken(index, end, TokenTypeOperator, 0, text)
}

func newNumberToken(index, end int, n float64) *Token {
	return NewToken(index, end, TokenTypeNumber, n, "")
}

func newErrorToken(index, end int, message string) *Token {
	return NewToken(index, end, TokenTypeError, 0, message)
}

func newRegExpBodyToken(index, end int, text string) *Token {
	return NewToken(index, end, TokenTypeRegExpBody, 0, text)
}

func newRegExpFlagsToken(index, end int, text string) *Token {
	return NewToken(index, end, TokenTypeRegExpFlags, 0, text)
}
