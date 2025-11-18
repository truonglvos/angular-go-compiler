package expression_parser

import (
	"fmt"
	"ngc-go/packages/compiler/core"
	"ngc-go/packages/compiler/util"
	"strings"
)

// InterpolationPiece represents a piece of interpolation
type InterpolationPiece struct {
	Text  string
	Start int
	End   int
}

// SplitInterpolation represents a split interpolation result
type SplitInterpolation struct {
	Strings     []InterpolationPiece
	Expressions []InterpolationPiece
	Offsets     []int
}

// NewSplitInterpolation creates a new SplitInterpolation
func NewSplitInterpolation(strings []InterpolationPiece, expressions []InterpolationPiece, offsets []int) *SplitInterpolation {
	return &SplitInterpolation{
		Strings:     strings,
		Expressions: expressions,
		Offsets:     offsets,
	}
}

// TemplateBindingParseResult represents the result of parsing template bindings
type TemplateBindingParseResult struct {
	TemplateBindings []TemplateBinding
	Warnings         []string
	Errors           []*util.ParseError
}

// NewTemplateBindingParseResult creates a new TemplateBindingParseResult
func NewTemplateBindingParseResult(templateBindings []TemplateBinding, warnings []string, errors []*util.ParseError) *TemplateBindingParseResult {
	return &TemplateBindingParseResult{
		TemplateBindings: templateBindings,
		Warnings:         warnings,
		Errors:           errors,
	}
}

// ParseFlags represents the possible parse modes to be used as a bitmask
type ParseFlags int

const (
	ParseFlagsNone ParseFlags = 0
	// ParseFlagsAction indicates whether an output binding is being parsed
	ParseFlagsAction ParseFlags = 1 << 0
)

// ParseContextFlags describes a stateful context an expression parser is in
type ParseContextFlags int

const (
	ParseContextFlagsNone ParseContextFlags = 0
	// ParseContextFlagsWritable is a context in which a value may be written to an lvalue
	ParseContextFlagsWritable ParseContextFlags = 1
)

// getLocation returns the location string from a ParseSourceSpan
func getLocation(span *util.ParseSourceSpan) string {
	if span != nil && span.Start != nil {
		loc := span.Start.String()
		if loc != "" {
			return loc
		}
	}
	return "(unknown)"
}

// Parser parses expressions
type Parser struct {
	lexer                        *Lexer
	supportsDirectPipeReferences bool
}

// NewParser creates a new Parser
func NewParser(lexer *Lexer, supportsDirectPipeReferences bool) *Parser {
	return &Parser{
		lexer:                        lexer,
		supportsDirectPipeReferences: supportsDirectPipeReferences,
	}
}

// ParseAction parses an action expression
func (p *Parser) ParseAction(input string, parseSourceSpan *util.ParseSourceSpan, absoluteOffset int) *ASTWithSource {
	errors := []*util.ParseError{}
	p.checkNoInterpolation(&errors, input, parseSourceSpan)
	stripped, _ := p.stripComments(input)
	tokens := p.lexer.Tokenize(stripped)
	ast := newParseAST(
		input,
		parseSourceSpan,
		absoluteOffset,
		tokens,
		ParseFlagsAction,
		&errors,
		0,
		p.supportsDirectPipeReferences,
	).parseChain()

	return NewASTWithSource(ast, &input, getLocation(parseSourceSpan), absoluteOffset, errors)
}

// ParseBinding parses a binding expression
func (p *Parser) ParseBinding(input string, parseSourceSpan *util.ParseSourceSpan, absoluteOffset int) *ASTWithSource {
	errors := []*util.ParseError{}
	ast := p.parseBindingAST(input, parseSourceSpan, absoluteOffset, &errors)
	return NewASTWithSource(ast, &input, getLocation(parseSourceSpan), absoluteOffset, errors)
}

// ParseSimpleBinding parses a simple binding expression (for host bindings)
func (p *Parser) ParseSimpleBinding(input string, parseSourceSpan *util.ParseSourceSpan, absoluteOffset int) *ASTWithSource {
	errors := []*util.ParseError{}
	ast := p.parseBindingAST(input, parseSourceSpan, absoluteOffset, &errors)
	simpleExpressionErrors := p.checkSimpleExpression(ast)

	if len(simpleExpressionErrors) > 0 {
		errors = append(errors, getParseError(
			"Host binding expression cannot contain "+strings.Join(simpleExpressionErrors, " "),
			input,
			"",
			parseSourceSpan,
		))
	}
	return NewASTWithSource(ast, &input, getLocation(parseSourceSpan), absoluteOffset, errors)
}

// parseBindingAST parses a binding AST
func (p *Parser) parseBindingAST(input string, parseSourceSpan *util.ParseSourceSpan, absoluteOffset int, errors *[]*util.ParseError) AST {
	p.checkNoInterpolation(errors, input, parseSourceSpan)
	stripped, _ := p.stripComments(input)
	tokens := p.lexer.Tokenize(stripped)
	return newParseAST(
		input,
		parseSourceSpan,
		absoluteOffset,
		tokens,
		ParseFlagsNone,
		errors,
		0,
		p.supportsDirectPipeReferences,
	).parseChain()
}

// ParseTemplateBindings parses microsyntax template expression and returns a list of bindings
func (p *Parser) ParseTemplateBindings(
	templateKey string,
	templateValue string,
	parseSourceSpan *util.ParseSourceSpan,
	absoluteKeyOffset int,
	absoluteValueOffset int,
) *TemplateBindingParseResult {
	tokens := p.lexer.Tokenize(templateValue)
	errors := []*util.ParseError{}
	parser := newParseAST(
		templateValue,
		parseSourceSpan,
		absoluteValueOffset,
		tokens,
		ParseFlagsNone,
		&errors,
		0,
		p.supportsDirectPipeReferences,
	)
	return parser.parseTemplateBindings(&TemplateBindingIdentifier{
		Source: templateKey,
		Span:   NewAbsoluteSourceSpan(absoluteKeyOffset, absoluteKeyOffset+len(templateKey)),
	})
}

// ParseInterpolation parses an interpolation expression
func (p *Parser) ParseInterpolation(
	input string,
	parseSourceSpan *util.ParseSourceSpan,
	absoluteOffset int,
	interpolatedTokens interface{}, // InterpolatedAttributeToken[] | InterpolatedTextToken[] | null
) *ASTWithSource {
	errors := []*util.ParseError{}
	split := p.splitInterpolation(input, parseSourceSpan, &errors, interpolatedTokens)
	if len(split.Expressions) == 0 {
		return nil
	}

	expressionNodes := []AST{}

	for i := 0; i < len(split.Expressions); i++ {
		// TODO: Handle interpolatedTokens for more accurate error messages
		expressionText := split.Expressions[i].Text
		stripped, hasComments := p.stripComments(expressionText)
		tokens := p.lexer.Tokenize(stripped)

		if hasComments && strings.TrimSpace(stripped) == "" && len(tokens) == 0 {
			errors = append(errors, getParseError(
				"Interpolation expression cannot only contain a comment",
				input,
				fmt.Sprintf("at column %d in", split.Expressions[i].Start),
				parseSourceSpan,
			))
			continue
		}

		ast := newParseAST(
			expressionText,
			parseSourceSpan,
			absoluteOffset,
			tokens,
			ParseFlagsNone,
			&errors,
			split.Offsets[i],
			p.supportsDirectPipeReferences,
		).parseChain()
		expressionNodes = append(expressionNodes, ast)
	}

	return p.createInterpolationAST(
		split.Strings,
		expressionNodes,
		input,
		getLocation(parseSourceSpan),
		absoluteOffset,
		&errors,
	)
}

// ParseInterpolationExpression parses a single interpolation expression (for ICU switch expressions)
func (p *Parser) ParseInterpolationExpression(
	expression string,
	parseSourceSpan *util.ParseSourceSpan,
	absoluteOffset int,
) *ASTWithSource {
	stripped, _ := p.stripComments(expression)
	tokens := p.lexer.Tokenize(stripped)
	errors := []*util.ParseError{}
	ast := newParseAST(
		expression,
		parseSourceSpan,
		absoluteOffset,
		tokens,
		ParseFlagsNone,
		&errors,
		0,
		p.supportsDirectPipeReferences,
	).parseChain()
	strings := []string{"", ""} // The prefix and suffix strings are both empty
	return p.createInterpolationAST(
		convertStringsToPieces(strings),
		[]AST{ast},
		expression,
		getLocation(parseSourceSpan),
		absoluteOffset,
		&errors,
	)
}

// WrapLiteralPrimitive wraps a literal primitive value
func (p *Parser) WrapLiteralPrimitive(
	input *string,
	sourceSpanOrLocation interface{}, // ParseSourceSpan | string
	absoluteOffset int,
) *ASTWithSource {
	var sourceLen int
	if input != nil {
		sourceLen = len(*input)
	}
	span := NewParseSpan(0, sourceLen)
	var location string
	switch v := sourceSpanOrLocation.(type) {
	case string:
		location = v
	case *util.ParseSourceSpan:
		location = getLocation(v)
	default:
		location = "(unknown)"
	}
	return NewASTWithSource(
		NewLiteralPrimitive(span, span.ToAbsolute(absoluteOffset), input),
		input,
		location,
		absoluteOffset,
		[]*util.ParseError{},
	)
}

// Helper functions

func (p *Parser) checkNoInterpolation(errors *[]*util.ParseError, input string, parseSourceSpan *util.ParseSourceSpan) {
	startIndex := -1
	endIndex := -1

	for charIndex := range p.forEachUnquotedChar(input, 0) {
		if startIndex == -1 {
			if strings.HasPrefix(input[charIndex:], "{{") {
				startIndex = charIndex
			}
		} else {
			endIndex = p.getInterpolationEndIndex(input, "}}", charIndex)
			if endIndex > -1 {
				break
			}
		}
	}

	if startIndex > -1 && endIndex > -1 {
		*errors = append(*errors, getParseError(
			"Got interpolation ({{}}) where expression was expected",
			input,
			fmt.Sprintf("at column %d in", startIndex),
			parseSourceSpan,
		))
	}
}

func (p *Parser) stripComments(input string) (stripped string, hasComments bool) {
	i := p.commentStart(input)
	if i != nil {
		return input[:*i], true
	}
	return input, false
}

func (p *Parser) commentStart(input string) *int {
	var outerQuote *rune
	for i := 0; i < len(input)-1; i++ {
		char := rune(input[i])
		nextChar := rune(input[i+1])

		if char == core.CharSLASH && nextChar == core.CharSLASH && outerQuote == nil {
			return &i
		}

		if outerQuote != nil && *outerQuote == char {
			outerQuote = nil
		} else if outerQuote == nil && core.IsQuote(int(char)) {
			outerQuote = &char
		}
	}
	return nil
}

func (p *Parser) getInterpolationEndIndex(input string, expressionEnd string, start int) int {
	for charIndex := range p.forEachUnquotedChar(input, start) {
		if strings.HasPrefix(input[charIndex:], expressionEnd) {
			return charIndex
		}

		// Nothing else in the expression matters after we've
		// hit a comment so look directly for the end token.
		if strings.HasPrefix(input[charIndex:], "//") {
			idx := strings.Index(input[charIndex:], expressionEnd)
			if idx != -1 {
				return charIndex + idx
			}
			return -1
		}
	}
	return -1
}

func (p *Parser) forEachUnquotedChar(input string, start int) <-chan int {
	ch := make(chan int)
	go func() {
		defer close(ch)
		var currentQuote *rune
		escapeCount := 0
		for i := start; i < len(input); i++ {
			char := rune(input[i])
			// Skip the characters inside quotes. Note that we only care about the outer-most
			// quotes matching up and we need to account for escape characters.
			if core.IsQuote(int(char)) &&
				(currentQuote == nil || *currentQuote == char) &&
				escapeCount%2 == 0 {
				if currentQuote == nil {
					currentQuote = &char
				} else {
					currentQuote = nil
				}
			} else if currentQuote == nil {
				ch <- i
			}
			if char == '\\' {
				escapeCount++
			} else {
				escapeCount = 0
			}
		}
	}()
	return ch
}

func (p *Parser) checkSimpleExpression(ast AST) []string {
	checker := NewSimpleExpressionChecker()
	ast.Visit(checker, nil)
	return checker.Errors
}

func (p *Parser) splitInterpolation(
	input string,
	parseSourceSpan *util.ParseSourceSpan,
	errors *[]*util.ParseError,
	interpolatedTokens interface{},
) *SplitInterpolation {
	stringPieces := []InterpolationPiece{}
	expressions := []InterpolationPiece{}
	offsets := []int{}
	// TODO: Handle interpolatedTokens for inputToTemplateIndexMap
	i := 0
	atInterpolation := false
	extendLastString := false
	interpStart := "{{"
	interpEnd := "}}"
	for i < len(input) {
		if !atInterpolation {
			// parse until starting {{
			start := i
			idx := strings.Index(input[i:], interpStart)
			if idx == -1 {
				i = len(input)
			} else {
				i = i + idx
			}
			text := input[start:i]
			stringPieces = append(stringPieces, InterpolationPiece{Text: text, Start: start, End: i})
			atInterpolation = true
		} else {
			// parse from starting {{ to ending }} while ignoring content inside quotes.
			fullStart := i
			exprStart := fullStart + len(interpStart)
			exprEnd := p.getInterpolationEndIndex(input, interpEnd, exprStart)
			if exprEnd == -1 {
				// Could not find the end of the interpolation; do not parse an expression.
				// Instead we should extend the content on the last raw string.
				atInterpolation = false
				extendLastString = true
				break
			}
			fullEnd := exprEnd + len(interpEnd)

			text := input[exprStart:exprEnd]
			if strings.TrimSpace(text) == "" {
				*errors = append(*errors, getParseError(
					"Blank expressions are not allowed in interpolated strings",
					input,
					fmt.Sprintf("at column %d in", i),
					parseSourceSpan,
				))
			}
			expressions = append(expressions, InterpolationPiece{Text: text, Start: fullStart, End: fullEnd})
			// TODO: Handle inputToTemplateIndexMap
			offset := fullStart + len(interpStart)
			offsets = append(offsets, offset)

			i = fullEnd
			atInterpolation = false
		}
	}
	if !atInterpolation {
		// If we are now at a text section, add the remaining content as a raw string.
		if extendLastString {
			if len(stringPieces) > 0 {
				piece := &stringPieces[len(stringPieces)-1]
				piece.Text += input[i:]
				piece.End = len(input)
			}
		} else {
			stringPieces = append(stringPieces, InterpolationPiece{Text: input[i:], Start: i, End: len(input)})
		}
	}
	return NewSplitInterpolation(stringPieces, expressions, offsets)
}

func (p *Parser) createInterpolationAST(
	strings []InterpolationPiece,
	expressions []AST,
	input string,
	location string,
	absoluteOffset int,
	errors *[]*util.ParseError,
) *ASTWithSource {
	stringStrs := make([]string, len(strings))
	for i, s := range strings {
		stringStrs[i] = s.Text
	}
	span := NewParseSpan(0, len(input))
	interpolation := NewInterpolation(
		span,
		span.ToAbsolute(absoluteOffset),
		stringStrs,
		expressions,
	)
	return NewASTWithSource(interpolation, &input, location, absoluteOffset, *errors)
}

func convertStringsToPieces(strs []string) []InterpolationPiece {
	pieces := make([]InterpolationPiece, len(strs))
	for i, s := range strs {
		pieces[i] = InterpolationPiece{
			Text:  s,
			Start: 0,
			End:   len(s),
		}
	}
	return pieces
}

// _ParseAST is the internal parser class
type parseAST struct {
	input                        string
	parseSourceSpan              *util.ParseSourceSpan
	absoluteOffset               int
	tokens                       []*Token
	parseFlags                   ParseFlags
	errors                       *[]*util.ParseError
	offset                       int
	supportsDirectPipeReferences bool
	index                        int
	rparensExpected              int
	rbracketsExpected            int
	rbracesExpected              int
	context                      ParseContextFlags
	sourceSpanCache              map[string]*AbsoluteSourceSpan
}

// newParseAST creates a new parseAST
func newParseAST(
	input string,
	parseSourceSpan *util.ParseSourceSpan,
	absoluteOffset int,
	tokens []*Token,
	parseFlags ParseFlags,
	errors *[]*util.ParseError,
	offset int,
	supportsDirectPipeReferences bool,
) *parseAST {
	return &parseAST{
		input:                        input,
		parseSourceSpan:              parseSourceSpan,
		absoluteOffset:               absoluteOffset,
		tokens:                       tokens,
		parseFlags:                   parseFlags,
		errors:                       errors,
		offset:                       offset,
		supportsDirectPipeReferences: supportsDirectPipeReferences,
		index:                        0,
		rparensExpected:              0,
		rbracketsExpected:            0,
		rbracesExpected:              0,
		context:                      ParseContextFlagsNone,
		sourceSpanCache:              make(map[string]*AbsoluteSourceSpan),
	}
}

// peek returns the token at the given offset
func (p *parseAST) peek(offset int) *Token {
	i := p.index + offset
	if i < len(p.tokens) {
		return p.tokens[i]
	}
	return EOF
}

// next returns the next token
func (p *parseAST) next() *Token {
	return p.peek(0)
}

// atEOF checks if all tokens have been processed
func (p *parseAST) atEOF() bool {
	return p.index >= len(p.tokens)
}

// inputIndex returns the index of the next token to be processed
func (p *parseAST) inputIndex() int {
	if p.atEOF() {
		return p.currentEndIndex()
	}
	return p.next().Index + p.offset
}

// currentEndIndex returns the end index of the last processed token
func (p *parseAST) currentEndIndex() int {
	if p.index > 0 {
		curToken := p.peek(-1)
		return curToken.End + p.offset
	}
	if len(p.tokens) == 0 {
		return len(p.input) + p.offset
	}
	return p.next().Index + p.offset
}

// currentAbsoluteOffset returns the absolute offset of the start of the current token
func (p *parseAST) currentAbsoluteOffset() int {
	return p.absoluteOffset + p.inputIndex()
}

// span returns a ParseSpan from start to the current position
func (p *parseAST) span(start int, artificialEndIndex ...int) *ParseSpan {
	endIndex := p.currentEndIndex()
	if len(artificialEndIndex) > 0 && artificialEndIndex[0] > endIndex {
		endIndex = artificialEndIndex[0]
	}
	if start > endIndex {
		tmp := endIndex
		endIndex = start
		start = tmp
	}
	return NewParseSpan(start, endIndex)
}

// sourceSpan returns an AbsoluteSourceSpan
func (p *parseAST) sourceSpan(start int, artificialEndIndex ...int) *AbsoluteSourceSpan {
	var endIdx *int
	if len(artificialEndIndex) > 0 {
		endIdx = &artificialEndIndex[0]
	}
	serial := fmt.Sprintf("%d@%d", start, p.inputIndex())
	if endIdx != nil {
		serial += fmt.Sprintf(":%d", *endIdx)
	}
	if cached, ok := p.sourceSpanCache[serial]; ok {
		return cached
	}
	span := p.span(start, artificialEndIndex...)
	absSpan := span.ToAbsolute(p.absoluteOffset)
	p.sourceSpanCache[serial] = absSpan
	return absSpan
}

// advance advances to the next token
func (p *parseAST) advance() {
	p.index++
}

// withContext executes a callback in the provided context
func (p *parseAST) withContext(context ParseContextFlags, cb func() interface{}) interface{} {
	p.context |= context
	ret := cb()
	p.context ^= context
	return ret
}

// consumeOptionalCharacter consumes an optional character
func (p *parseAST) consumeOptionalCharacter(code int) bool {
	if p.next().IsCharacter(code) {
		p.advance()
		return true
	}
	return false
}

// peekKeywordLet checks if the next token is the 'let' keyword
func (p *parseAST) peekKeywordLet() bool {
	return p.next().IsKeywordLet()
}

// peekKeywordAs checks if the next token is the 'as' keyword
func (p *parseAST) peekKeywordAs() bool {
	return p.next().IsKeywordAs()
}

// expectCharacter consumes an expected character or emits an error
func (p *parseAST) expectCharacter(code int) {
	if !p.consumeOptionalCharacter(code) {
		p.error(fmt.Sprintf("Missing expected %c", code))
	}
}

// consumeOptionalOperator consumes an optional operator
func (p *parseAST) consumeOptionalOperator(op string) bool {
	if p.next().IsOperator(op) {
		p.advance()
		return true
	}
	return false
}

// isAssignmentOperator checks if a token is an assignment operator
func (p *parseAST) isAssignmentOperator(token *Token) bool {
	return token.Type == TokenTypeOperator && IsAssignmentOperation(token.StrValue)
}

// expectOperator consumes an expected operator or emits an error
func (p *parseAST) expectOperator(operator string) {
	if !p.consumeOptionalOperator(operator) {
		p.error(fmt.Sprintf("Missing expected operator %s", operator))
	}
}

// prettyPrintToken returns a string representation of a token
func (p *parseAST) prettyPrintToken(tok *Token) string {
	if tok == EOF {
		return "end of input"
	}
	return fmt.Sprintf("token %s", tok.String())
}

// expectIdentifierOrKeyword expects an identifier or keyword
func (p *parseAST) expectIdentifierOrKeyword() *string {
	n := p.next()
	if !n.IsIdentifier() && !n.IsKeyword() {
		if n.IsPrivateIdentifier() {
			p.reportErrorForPrivateIdentifier(n, "expected identifier or keyword")
		} else {
			p.error(fmt.Sprintf("Unexpected %s, expected identifier or keyword", p.prettyPrintToken(n)))
		}
		return nil
	}
	p.advance()
	str := n.String()
	return &str
}

// expectIdentifierOrKeywordOrString expects an identifier, keyword, or string
func (p *parseAST) expectIdentifierOrKeywordOrString() string {
	n := p.next()
	if !n.IsIdentifier() && !n.IsKeyword() && !n.IsString() {
		if n.IsPrivateIdentifier() {
			p.reportErrorForPrivateIdentifier(n, "expected identifier, keyword or string")
		} else {
			p.error(fmt.Sprintf("Unexpected %s, expected identifier, keyword, or string", p.prettyPrintToken(n)))
		}
		return ""
	}
	p.advance()
	return n.String()
}

// parseChain parses a chain of expressions
func (p *parseAST) parseChain() AST {
	exprs := []AST{}
	start := p.inputIndex()
	for p.index < len(p.tokens) {
		expr := p.parsePipe()
		exprs = append(exprs, expr)

		if p.consumeOptionalCharacter(core.CharSEMICOLON) {
			if p.parseFlags&ParseFlagsAction == 0 {
				p.error("Binding expression cannot contain chained expression")
			}
			for p.consumeOptionalCharacter(core.CharSEMICOLON) {
				// read all semicolons
			}
		} else if p.index < len(p.tokens) {
			errorIndex := p.index
			p.error(fmt.Sprintf("Unexpected token '%s'", p.next()))
			if p.index == errorIndex {
				break
			}
		}
	}
	if len(exprs) == 0 {
		artificialStart := p.offset
		artificialEnd := p.offset + len(p.input)
		return NewEmptyExpr(
			p.span(artificialStart, artificialEnd),
			p.sourceSpan(artificialStart, artificialEnd),
		)
	}
	if len(exprs) == 1 {
		return exprs[0]
	}
	return NewChain(p.span(start), p.sourceSpan(start), exprs)
}

// parsePipe parses a pipe expression
func (p *parseAST) parsePipe() AST {
	start := p.inputIndex()
	result := p.parseExpression()
	if p.consumeOptionalOperator("|") {
		if p.parseFlags&ParseFlagsAction != 0 {
			p.error("Cannot have a pipe in an action expression")
		}

		for {
			nameStart := p.inputIndex()
			nameId := p.expectIdentifierOrKeyword()
			var nameSpan *AbsoluteSourceSpan
			var fullSpanEnd *int
			if nameId != nil {
				nameSpan = p.sourceSpan(nameStart)
			} else {
				// No valid identifier was found, so we'll assume an empty pipe name ('').
				emptyStr := ""
				nameId = &emptyStr

				// However, there may have been whitespace present between the pipe character and the next
				// token in the sequence (or the end of input). We want to track this whitespace so that
				// the `BindingPipe` we produce covers not just the pipe character, but any trailing
				// whitespace beyond it. Another way of thinking about this is that the zero-length name
				// is assumed to be at the end of any whitespace beyond the pipe character.
				//
				// Therefore, we push the end of the `ParseSpan` for this pipe all the way up to the
				// beginning of the next token, or until the end of input if the next token is EOF.
				nextIdx := p.next().Index
				if nextIdx != -1 {
					end := nextIdx
					fullSpanEnd = &end
				} else {
					end := len(p.input) + p.offset
					fullSpanEnd = &end
				}

				// The `nameSpan` for an empty pipe name is zero-length at the end of any whitespace
				// beyond the pipe character.
				span := NewParseSpan(*fullSpanEnd, *fullSpanEnd)
				nameSpan = span.ToAbsolute(p.absoluteOffset)
			}

			args := []AST{}
			for p.consumeOptionalCharacter(core.CharCOLON) {
				args = append(args, p.parseExpression())
			}
			var pipeType BindingPipeType
			if p.supportsDirectPipeReferences {
				if *nameId != "" {
					charCode := int((*nameId)[0])
					if charCode == core.CharUnderscore || (charCode >= core.CharA && charCode <= core.CharZ) {
						pipeType = ReferencedDirectly
					} else {
						pipeType = ReferencedByName
					}
				} else {
					pipeType = ReferencedByName
				}
			} else {
				pipeType = ReferencedByName
			}

			if fullSpanEnd != nil {
				result = NewBindingPipe(
					p.span(start, *fullSpanEnd),
					p.sourceSpan(start, *fullSpanEnd),
					result,
					*nameId,
					args,
					pipeType,
					nameSpan,
				)
			} else {
				result = NewBindingPipe(
					p.span(start),
					p.sourceSpan(start),
					result,
					*nameId,
					args,
					pipeType,
					nameSpan,
				)
			}
			if !p.consumeOptionalOperator("|") {
				break
			}
		}
	}

	return result
}

// parseExpression parses an expression
func (p *parseAST) parseExpression() AST {
	return p.parseConditional()
}

// parseConditional parses a conditional expression
func (p *parseAST) parseConditional() AST {
	start := p.inputIndex()
	result := p.parseLogicalOr()

	if p.consumeOptionalOperator("?") {
		yes := p.parsePipe()
		var no AST
		if !p.consumeOptionalCharacter(core.CharCOLON) {
			end := p.inputIndex()
			expression := p.input[start-p.offset : end-p.offset]
			p.error(fmt.Sprintf("Conditional expression %s requires all 3 expressions", expression))
			no = NewEmptyExpr(p.span(start), p.sourceSpan(start))
		} else {
			no = p.parsePipe()
		}
		return NewConditional(p.span(start), p.sourceSpan(start), result, yes, no)
	}
	return result
}

// parseLogicalOr parses a logical OR expression
func (p *parseAST) parseLogicalOr() AST {
	// '||'
	start := p.inputIndex()
	result := p.parseLogicalAnd()
	for p.consumeOptionalOperator("||") {
		right := p.parseLogicalAnd()
		result = NewBinary(p.span(start), p.sourceSpan(start), "||", result, right)
	}
	return result
}

// parseLogicalAnd parses a logical AND expression
func (p *parseAST) parseLogicalAnd() AST {
	// '&&'
	start := p.inputIndex()
	result := p.parseNullishCoalescing()
	for p.consumeOptionalOperator("&&") {
		right := p.parseNullishCoalescing()
		result = NewBinary(p.span(start), p.sourceSpan(start), "&&", result, right)
	}
	return result
}

// parseNullishCoalescing parses a nullish coalescing expression
func (p *parseAST) parseNullishCoalescing() AST {
	// '??'
	start := p.inputIndex()
	result := p.parseEquality()
	for p.consumeOptionalOperator("??") {
		right := p.parseEquality()
		result = NewBinary(p.span(start), p.sourceSpan(start), "??", result, right)
	}
	return result
}

// parseEquality parses an equality expression
func (p *parseAST) parseEquality() AST {
	// '==','!=','===','!=='
	start := p.inputIndex()
	result := p.parseRelational()
	for p.next().Type == TokenTypeOperator {
		operator := p.next().StrValue
		switch operator {
		case "==", "===", "!=", "!==":
			p.advance()
			right := p.parseRelational()
			result = NewBinary(p.span(start), p.sourceSpan(start), operator, result, right)
			continue
		}
		break
	}
	return result
}

// parseRelational parses a relational expression
func (p *parseAST) parseRelational() AST {
	// '<', '>', '<=', '>=', 'in'
	start := p.inputIndex()
	result := p.parseAdditive()
	for p.next().Type == TokenTypeOperator || p.next().IsKeywordIn() {
		operator := p.next().StrValue
		switch operator {
		case "<", ">", "<=", ">=", "in":
			p.advance()
			right := p.parseAdditive()
			result = NewBinary(p.span(start), p.sourceSpan(start), operator, result, right)
			continue
		}
		break
	}
	return result
}

// parseAdditive parses an additive expression
func (p *parseAST) parseAdditive() AST {
	// '+', '-'
	start := p.inputIndex()
	result := p.parseMultiplicative()
	for p.next().Type == TokenTypeOperator {
		operator := p.next().StrValue
		switch operator {
		case "+", "-":
			p.advance()
			right := p.parseMultiplicative()
			result = NewBinary(p.span(start), p.sourceSpan(start), operator, result, right)
			continue
		}
		break
	}
	return result
}

// parseMultiplicative parses a multiplicative expression
func (p *parseAST) parseMultiplicative() AST {
	// '*', '%', '/'
	start := p.inputIndex()
	result := p.parseExponentiation()
	for p.next().Type == TokenTypeOperator {
		operator := p.next().StrValue
		switch operator {
		case "*", "%", "/":
			p.advance()
			right := p.parseExponentiation()
			result = NewBinary(p.span(start), p.sourceSpan(start), operator, result, right)
			continue
		}
		break
	}
	return result
}

// parseExponentiation parses an exponentiation expression
func (p *parseAST) parseExponentiation() AST {
	// '**'
	start := p.inputIndex()
	result := p.parsePrefix()
	for p.next().Type == TokenTypeOperator && p.next().StrValue == "**" {
		// This aligns with Javascript semantics which require any unary operator preceeding the
		// exponentiation operation to be explicitly grouped as either applying to the base or result
		// of the exponentiation operation.
		if _, ok := result.(*Unary); ok {
			p.error("Unary operator used immediately before exponentiation expression. Parenthesis must be used to disambiguate operator precedence")
		} else if _, ok := result.(*PrefixNot); ok {
			p.error("Unary operator used immediately before exponentiation expression. Parenthesis must be used to disambiguate operator precedence")
		} else if _, ok := result.(*TypeofExpression); ok {
			p.error("Unary operator used immediately before exponentiation expression. Parenthesis must be used to disambiguate operator precedence")
		} else if _, ok := result.(*VoidExpression); ok {
			p.error("Unary operator used immediately before exponentiation expression. Parenthesis must be used to disambiguate operator precedence")
		}
		p.advance()
		right := p.parseExponentiation()
		result = NewBinary(p.span(start), p.sourceSpan(start), "**", result, right)
	}
	return result
}

// parsePrefix parses a prefix expression
func (p *parseAST) parsePrefix() AST {
	if p.next().Type == TokenTypeOperator {
		start := p.inputIndex()
		operator := p.next().StrValue
		var result AST
		switch operator {
		case "+":
			p.advance()
			result = p.parsePrefix()
			return CreatePlus(p.span(start), p.sourceSpan(start), result)
		case "-":
			p.advance()
			result = p.parsePrefix()
			return CreateMinus(p.span(start), p.sourceSpan(start), result)
		case "!":
			p.advance()
			result = p.parsePrefix()
			return NewPrefixNot(p.span(start), p.sourceSpan(start), result)
		}
	} else if p.next().IsKeywordTypeof() {
		p.advance()
		start := p.inputIndex()
		result := p.parsePrefix()
		return NewTypeofExpression(p.span(start), p.sourceSpan(start), result)
	} else if p.next().IsKeywordVoid() {
		p.advance()
		start := p.inputIndex()
		result := p.parsePrefix()
		return NewVoidExpression(p.span(start), p.sourceSpan(start), result)
	}
	return p.parseCallChain()
}

// parseCallChain parses a call chain expression
func (p *parseAST) parseCallChain() AST {
	start := p.inputIndex()
	result := p.parsePrimary()
	for {
		if p.consumeOptionalCharacter(core.CharPERIOD) {
			result = p.parseAccessMember(result, start, false)
		} else if p.consumeOptionalOperator("?.") {
			if p.consumeOptionalCharacter(core.CharLPAREN) {
				result = p.parseCall(result, start, true)
			} else {
				if p.consumeOptionalCharacter(core.CharLBRACKET) {
					result = p.parseKeyedReadOrWrite(result, start, true)
				} else {
					result = p.parseAccessMember(result, start, true)
				}
			}
		} else if p.consumeOptionalCharacter(core.CharLBRACKET) {
			result = p.parseKeyedReadOrWrite(result, start, false)
		} else if p.consumeOptionalCharacter(core.CharLPAREN) {
			result = p.parseCall(result, start, false)
		} else if p.consumeOptionalOperator("!") {
			result = NewNonNullAssert(p.span(start), p.sourceSpan(start), result)
		} else if p.next().IsTemplateLiteralEnd() {
			result = p.parseNoInterpolationTaggedTemplateLiteral(result, start)
		} else if p.next().IsTemplateLiteralPart() {
			result = p.parseTaggedTemplateLiteral(result, start)
		} else {
			return result
		}
	}
}

// parsePrimary parses a primary expression
func (p *parseAST) parsePrimary() AST {
	start := p.inputIndex()
	if p.consumeOptionalCharacter(core.CharLPAREN) {
		p.rparensExpected++
		result := p.parsePipe()
		if !p.consumeOptionalCharacter(core.CharRPAREN) {
			p.error("Missing closing parentheses")
			// Calling into `error` above will attempt to recover up until the next closing paren.
			// If that's the case, consume it so we can partially recover the expression.
			p.consumeOptionalCharacter(core.CharRPAREN)
		}
		p.rparensExpected--
		return NewParenthesizedExpression(p.span(start), p.sourceSpan(start), result)
	} else if p.next().IsKeywordNull() {
		p.advance()
		return NewLiteralPrimitive(p.span(start), p.sourceSpan(start), nil)
	} else if p.next().IsKeywordUndefined() {
		p.advance()
		return NewLiteralPrimitive(p.span(start), p.sourceSpan(start), nil) // void 0 in JS is nil
	} else if p.next().IsKeywordTrue() {
		p.advance()
		return NewLiteralPrimitive(p.span(start), p.sourceSpan(start), true)
	} else if p.next().IsKeywordFalse() {
		p.advance()
		return NewLiteralPrimitive(p.span(start), p.sourceSpan(start), false)
	} else if p.next().IsKeywordIn() {
		p.advance()
		return NewLiteralPrimitive(p.span(start), p.sourceSpan(start), "in")
	} else if p.next().IsKeywordThis() {
		p.advance()
		return NewThisReceiver(p.span(start), p.sourceSpan(start))
	} else if p.consumeOptionalCharacter(core.CharLBRACKET) {
		p.rbracketsExpected++
		elements := p.parseExpressionList(core.CharRBRACKET)
		p.rbracketsExpected--
		p.expectCharacter(core.CharRBRACKET)
		return NewLiteralArray(p.span(start), p.sourceSpan(start), elements)
	} else if p.next().IsCharacter(core.CharLBRACE) {
		return p.parseLiteralMap()
	} else if p.next().IsIdentifier() {
		return p.parseAccessMember(
			NewImplicitReceiver(p.span(start), p.sourceSpan(start)),
			start,
			false,
		)
	} else if p.next().IsNumber() {
		value := p.next().NumValue
		p.advance()
		return NewLiteralPrimitive(p.span(start), p.sourceSpan(start), value)
	} else if p.next().IsTemplateLiteralEnd() {
		return p.parseNoInterpolationTemplateLiteral()
	} else if p.next().IsTemplateLiteralPart() {
		return p.parseTemplateLiteral()
	} else if p.next().IsString() && p.next().Kind() == StringTokenKindPlain {
		literalValue := p.next().String()
		p.advance()
		return NewLiteralPrimitive(p.span(start), p.sourceSpan(start), literalValue)
	} else if p.next().IsPrivateIdentifier() {
		p.reportErrorForPrivateIdentifier(p.next(), "")
		return NewEmptyExpr(p.span(start), p.sourceSpan(start))
	} else if p.next().IsRegExpBody() {
		return p.parseRegularExpressionLiteral()
	} else if p.index >= len(p.tokens) {
		p.error(fmt.Sprintf("Unexpected end of expression: %s", p.input))
		return NewEmptyExpr(p.span(start), p.sourceSpan(start))
	} else {
		p.error(fmt.Sprintf("Unexpected token %s", p.next()))
		return NewEmptyExpr(p.span(start), p.sourceSpan(start))
	}
}

// parseExpressionList parses a list of expressions
func (p *parseAST) parseExpressionList(terminator int) []AST {
	result := []AST{}

	for {
		if !p.next().IsCharacter(terminator) {
			result = append(result, p.parsePipe())
		} else {
			break
		}
		if !p.consumeOptionalCharacter(core.CharCOMMA) {
			break
		}
	}
	return result
}

// parseLiteralMap parses a literal map/object
func (p *parseAST) parseLiteralMap() *LiteralMap {
	keys := []LiteralMapKey{}
	values := []AST{}
	start := p.inputIndex()
	p.expectCharacter(core.CharLBRACE)
	if !p.consumeOptionalCharacter(core.CharRBRACE) {
		p.rbracesExpected++
		for {
			keyStart := p.inputIndex()
			quoted := p.next().IsString()
			key := p.expectIdentifierOrKeywordOrString()
			literalMapKey := LiteralMapKey{Key: key, Quoted: quoted}
			keys = append(keys, literalMapKey)

			// Properties with quoted keys can't use the shorthand syntax.
			if quoted {
				p.expectCharacter(core.CharCOLON)
				values = append(values, p.parsePipe())
			} else if p.consumeOptionalCharacter(core.CharCOLON) {
				values = append(values, p.parsePipe())
			} else {
				literalMapKey.IsShorthandInitialized = true
				keys[len(keys)-1] = literalMapKey

				span := p.span(keyStart)
				sourceSpan := p.sourceSpan(keyStart)
				values = append(values, NewPropertyRead(
					span,
					sourceSpan,
					sourceSpan,
					NewImplicitReceiver(span, sourceSpan),
					key,
				))
			}
			if !p.consumeOptionalCharacter(core.CharCOMMA) {
				break
			}
			if p.next().IsCharacter(core.CharRBRACE) {
				break
			}
		}
		p.rbracesExpected--
		p.expectCharacter(core.CharRBRACE)
	}
	return NewLiteralMap(p.span(start), p.sourceSpan(start), keys, values)
}

// parseAccessMember parses a property access member
func (p *parseAST) parseAccessMember(readReceiver AST, start int, isSafe bool) AST {
	nameStart := p.inputIndex()
	id := p.withContext(ParseContextFlagsWritable, func() interface{} {
		idPtr := p.expectIdentifierOrKeyword()
		var id string
		if idPtr != nil {
			id = *idPtr
		}
		if len(id) == 0 {
			p.error("Expected identifier for property access", readReceiver.Span().End)
		}
		return id
	}).(string)
	nameSpan := p.sourceSpan(nameStart)

	if isSafe {
		if p.isAssignmentOperator(p.next()) {
			p.advance()
			p.error("The '?.' operator cannot be used in the assignment")
			return NewEmptyExpr(p.span(start), p.sourceSpan(start))
		}
		return NewSafePropertyRead(
			p.span(start),
			p.sourceSpan(start),
			nameSpan,
			readReceiver,
			id,
		)
	}
	if p.isAssignmentOperator(p.next()) {
		operation := p.next().StrValue

		if p.parseFlags&ParseFlagsAction == 0 {
			p.advance()
			p.error("Bindings cannot contain assignments")
			return NewEmptyExpr(p.span(start), p.sourceSpan(start))
		}
		receiver := NewPropertyRead(
			p.span(start),
			p.sourceSpan(start),
			nameSpan,
			readReceiver,
			id,
		)
		p.advance()
		value := p.parseConditional()
		return NewBinary(p.span(start), p.sourceSpan(start), operation, receiver, value)
	}
	return NewPropertyRead(
		p.span(start),
		p.sourceSpan(start),
		nameSpan,
		readReceiver,
		id,
	)
}

// parseCall parses a function call
func (p *parseAST) parseCall(receiver AST, start int, isSafe bool) AST {
	argumentStart := p.inputIndex()
	p.rparensExpected++
	args := p.parseCallArguments()
	argumentSpan := p.span(argumentStart, p.inputIndex()).ToAbsolute(p.absoluteOffset)
	p.expectCharacter(core.CharRPAREN)
	p.rparensExpected--
	span := p.span(start)
	sourceSpan := p.sourceSpan(start)
	if isSafe {
		return NewSafeCall(span, sourceSpan, receiver, args, argumentSpan)
	}
	return NewCall(span, sourceSpan, receiver, args, argumentSpan)
}

// parseCallArguments parses function call arguments
func (p *parseAST) parseCallArguments() []AST {
	if p.next().IsCharacter(core.CharRPAREN) {
		return []AST{}
	}
	positionals := []AST{}
	for {
		positionals = append(positionals, p.parsePipe())
		if !p.consumeOptionalCharacter(core.CharCOMMA) {
			break
		}
	}
	return positionals
}

// parseKeyedReadOrWrite parses a keyed read or write operation
func (p *parseAST) parseKeyedReadOrWrite(receiver AST, start int, isSafe bool) AST {
	return p.withContext(ParseContextFlagsWritable, func() interface{} {
		p.rbracketsExpected++
		key := p.parsePipe()
		if _, ok := key.(*EmptyExpr); ok {
			p.error("Key access cannot be empty")
		}
		p.rbracketsExpected--
		p.expectCharacter(core.CharRBRACKET)
		if p.isAssignmentOperator(p.next()) {
			operation := p.next().StrValue

			if isSafe {
				p.advance()
				p.error("The '?.' operator cannot be used in the assignment")
				return NewEmptyExpr(p.span(start), p.sourceSpan(start))
			}
			binaryReceiver := NewKeyedRead(
				p.span(start),
				p.sourceSpan(start),
				receiver,
				key,
			)
			p.advance()
			value := p.parseConditional()
			return NewBinary(
				p.span(start),
				p.sourceSpan(start),
				operation,
				binaryReceiver,
				value,
			)
		}
		if isSafe {
			return NewSafeKeyedRead(p.span(start), p.sourceSpan(start), receiver, key)
		}
		return NewKeyedRead(p.span(start), p.sourceSpan(start), receiver, key)
	}).(AST)
}

// parseTemplateLiteral parses a template literal
func (p *parseAST) parseTemplateLiteral() *TemplateLiteral {
	elements := []*TemplateLiteralElement{}
	expressions := []AST{}
	start := p.inputIndex()

	for p.next() != EOF {
		token := p.next()

		if token.IsTemplateLiteralPart() || token.IsTemplateLiteralEnd() {
			partStart := p.inputIndex()
			p.advance()
			elements = append(elements, NewTemplateLiteralElement(
				p.span(partStart),
				p.sourceSpan(partStart),
				token.StrValue,
			))
			if token.IsTemplateLiteralEnd() {
				break
			}
		} else if token.IsTemplateLiteralInterpolationStart() {
			p.advance()
			p.rbracesExpected++
			expression := p.parsePipe()
			if _, ok := expression.(*EmptyExpr); ok {
				p.error("Template literal interpolation cannot be empty")
			} else {
				expressions = append(expressions, expression)
			}
			p.rbracesExpected--
		} else {
			p.advance()
		}
	}

	return NewTemplateLiteral(p.span(start), p.sourceSpan(start), elements, expressions)
}

// parseNoInterpolationTemplateLiteral parses a template literal without interpolation
func (p *parseAST) parseNoInterpolationTemplateLiteral() *TemplateLiteral {
	text := p.next().StrValue
	start := p.inputIndex()
	p.advance()
	span := p.span(start)
	sourceSpan := p.sourceSpan(start)
	return NewTemplateLiteral(
		span,
		sourceSpan,
		[]*TemplateLiteralElement{NewTemplateLiteralElement(span, sourceSpan, text)},
		[]AST{},
	)
}

// parseTaggedTemplateLiteral parses a tagged template literal
func (p *parseAST) parseTaggedTemplateLiteral(tag AST, start int) AST {
	template := p.parseTemplateLiteral()
	return NewTaggedTemplateLiteral(p.span(start), p.sourceSpan(start), tag, template)
}

// parseNoInterpolationTaggedTemplateLiteral parses a tagged template literal without interpolation
func (p *parseAST) parseNoInterpolationTaggedTemplateLiteral(tag AST, start int) AST {
	template := p.parseNoInterpolationTemplateLiteral()
	return NewTaggedTemplateLiteral(p.span(start), p.sourceSpan(start), tag, template)
}

// parseRegularExpressionLiteral parses a regular expression literal
func (p *parseAST) parseRegularExpressionLiteral() AST {
	bodyToken := p.next()
	p.advance()

	if !bodyToken.IsRegExpBody() {
		return NewEmptyExpr(p.span(p.inputIndex()), p.sourceSpan(p.inputIndex()))
	}

	var flagsToken *Token
	supportedRegexFlags := map[rune]bool{'d': true, 'g': true, 'i': true, 'm': true, 's': true, 'u': true, 'v': true, 'y': true}

	if p.next().IsRegExpFlags() {
		flagsToken = p.next()
		p.advance()
		seenFlags := make(map[rune]bool)

		for i, char := range flagsToken.StrValue {
			if !supportedRegexFlags[char] {
				p.error(
					fmt.Sprintf("Unsupported regular expression flag \"%c\". The supported flags are: \"d\", \"g\", \"i\", \"m\", \"s\", \"u\", \"v\", \"y\"", char),
					flagsToken.Index+i,
				)
			} else if seenFlags[char] {
				p.error(fmt.Sprintf("Duplicate regular expression flag \"%c\"", char), flagsToken.Index+i)
			} else {
				seenFlags[char] = true
			}
		}
	}

	start := bodyToken.Index
	var end int
	if flagsToken != nil {
		end = flagsToken.End
	} else {
		end = bodyToken.End
	}

	var flags *string
	if flagsToken != nil {
		flagsStr := flagsToken.StrValue
		flags = &flagsStr
	}

	return NewRegularExpressionLiteral(
		p.span(start, end),
		p.sourceSpan(start, end),
		bodyToken.StrValue,
		flags,
	)
}

// expectTemplateBindingKey expects a template binding key
func (p *parseAST) expectTemplateBindingKey() *TemplateBindingIdentifier {
	result := ""
	operatorFound := false
	start := p.currentAbsoluteOffset()
	for {
		result += p.expectIdentifierOrKeywordOrString()
		operatorFound = p.consumeOptionalOperator("-")
		if operatorFound {
			result += "-"
		} else {
			break
		}
	}
	return &TemplateBindingIdentifier{
		Source: result,
		Span:   NewAbsoluteSourceSpan(start, start+len(result)),
	}
}

// parseTemplateBindings parses template bindings
func (p *parseAST) parseTemplateBindings(templateKey *TemplateBindingIdentifier) *TemplateBindingParseResult {
	bindings := []TemplateBinding{}

	// The first binding is for the template key itself
	bindings = append(bindings, p.parseDirectiveKeywordBindings(templateKey)...)

	for p.index < len(p.tokens) {
		// If it starts with 'let', then this must be variable declaration
		letBinding := p.parseLetBinding()
		if letBinding != nil {
			bindings = append(bindings, letBinding)
		} else {
			// Two possible cases here, either `value "as" key` or
			// "directive-keyword expression". We don't know which case, but both
			// "value" and "directive-keyword" are template binding key, so consume
			// the key first.
			key := p.expectTemplateBindingKey()
			// Peek at the next token, if it is "as" then this must be variable
			// declaration.
			binding := p.parseAsBinding(key)
			if binding != nil {
				bindings = append(bindings, binding)
			} else {
				// Otherwise the key must be a directive keyword, like "of". Transform
				// the key to actual key. Eg. of -> ngForOf, trackBy -> ngForTrackBy
				if len(key.Source) > 0 {
					key.Source = templateKey.Source + strings.ToUpper(string(key.Source[0])) + key.Source[1:]
				}
				bindings = append(bindings, p.parseDirectiveKeywordBindings(key)...)
			}
		}
		p.consumeStatementTerminator()
	}

	return NewTemplateBindingParseResult(bindings, []string{}, *p.errors)
}

// parseDirectiveKeywordBindings parses directive keyword bindings
func (p *parseAST) parseDirectiveKeywordBindings(key *TemplateBindingIdentifier) []TemplateBinding {
	bindings := []TemplateBinding{}
	p.consumeOptionalCharacter(core.CharCOLON) // trackBy: trackByFunction
	value := p.getDirectiveBoundTarget()
	spanEnd := p.currentAbsoluteOffset()
	// The binding could optionally be followed by "as". For example,
	// *ngIf="cond | pipe as x". In this case, the key in the "as" binding
	// is "x" and the value is the template key itself ("ngIf"). Note that the
	// 'key' in the current context now becomes the "value" in the next binding.
	asBinding := p.parseAsBinding(key)
	if asBinding == nil {
		p.consumeStatementTerminator()
		spanEnd = p.currentAbsoluteOffset()
	}
	sourceSpan := NewAbsoluteSourceSpan(key.Span.Start, spanEnd)
	bindings = append(bindings, NewExpressionBinding(sourceSpan, key, value))
	if asBinding != nil {
		bindings = append(bindings, asBinding)
	}
	return bindings
}

// getDirectiveBoundTarget returns the expression AST for the bound target
func (p *parseAST) getDirectiveBoundTarget() *ASTWithSource {
	if p.next() == EOF || p.peekKeywordAs() || p.peekKeywordLet() {
		return nil
	}
	ast := p.parsePipe() // example: "condition | async"
	span := ast.Span()
	value := p.input[span.Start-p.offset : span.End-p.offset]
	return NewASTWithSource(
		ast,
		&value,
		getLocation(p.parseSourceSpan),
		p.absoluteOffset+span.Start,
		*p.errors,
	)
}

// parseAsBinding parses an 'as' binding
func (p *parseAST) parseAsBinding(value *TemplateBindingIdentifier) TemplateBinding {
	if !p.peekKeywordAs() {
		return nil
	}
	p.advance() // consume the 'as' keyword
	key := p.expectTemplateBindingKey()
	p.consumeStatementTerminator()
	sourceSpan := NewAbsoluteSourceSpan(value.Span.Start, p.currentAbsoluteOffset())
	return NewVariableBinding(sourceSpan, key, value)
}

// parseLetBinding parses a 'let' binding
func (p *parseAST) parseLetBinding() TemplateBinding {
	if !p.peekKeywordLet() {
		return nil
	}
	spanStart := p.currentAbsoluteOffset()
	p.advance() // consume the 'let' keyword
	key := p.expectTemplateBindingKey()
	var value *TemplateBindingIdentifier
	if p.consumeOptionalOperator("=") {
		value = p.expectTemplateBindingKey()
	}
	p.consumeStatementTerminator()
	sourceSpan := NewAbsoluteSourceSpan(spanStart, p.currentAbsoluteOffset())
	return NewVariableBinding(sourceSpan, key, value)
}

// consumeStatementTerminator consumes the optional statement terminator
func (p *parseAST) consumeStatementTerminator() {
	if !p.consumeOptionalCharacter(core.CharSEMICOLON) {
		p.consumeOptionalCharacter(core.CharCOMMA)
	}
}

// error records an error and skips tokens until a recoverable point
func (p *parseAST) error(message string, index ...int) {
	idx := p.index
	if len(index) > 0 {
		idx = index[0]
	}
	*p.errors = append(*p.errors, getParseError(
		message,
		p.input,
		p.getErrorLocationText(idx),
		p.parseSourceSpan,
	))
	p.skip()
}

// getErrorLocationText returns error location text
func (p *parseAST) getErrorLocationText(index int) string {
	if index < len(p.tokens) {
		return fmt.Sprintf("at column %d in", p.tokens[index].Index+1)
	}
	return "at the end of the expression"
}

// reportErrorForPrivateIdentifier reports an error for a private identifier
func (p *parseAST) reportErrorForPrivateIdentifier(token *Token, extraMessage string) {
	errorMessage := fmt.Sprintf("Private identifiers are not supported. Unexpected private identifier: %s", token)
	if extraMessage != "" {
		errorMessage += ", " + extraMessage
	}
	p.error(errorMessage)
}

// skip skips tokens until reaching a recoverable point
func (p *parseAST) skip() {
	n := p.next()
	for p.index < len(p.tokens) &&
		!n.IsCharacter(core.CharSEMICOLON) &&
		!n.IsOperator("|") &&
		(p.rparensExpected <= 0 || !n.IsCharacter(core.CharRPAREN)) &&
		(p.rbracesExpected <= 0 || !n.IsCharacter(core.CharRBRACE)) &&
		(p.rbracketsExpected <= 0 || !n.IsCharacter(core.CharRBRACKET)) &&
		((p.context&ParseContextFlagsWritable == 0) || !p.isAssignmentOperator(n)) {
		if p.next().Type == TokenTypeError {
			*p.errors = append(*p.errors, getParseError(
				p.next().String(),
				p.input,
				p.getErrorLocationText(p.next().Index),
				p.parseSourceSpan,
			))
		}
		p.advance()
		n = p.next()
	}
}

// getParseError creates a ParseError
func getParseError(message, input, locationText string, parseSourceSpan *util.ParseSourceSpan) *util.ParseError {
	if locationText != "" {
		locationText = " " + locationText + " "
	}
	location := getLocation(parseSourceSpan)
	errorMsg := fmt.Sprintf("Parser Error: %s%s[%s] in %s", message, locationText, input, location)
	return util.NewParseError(parseSourceSpan, errorMsg)
}

// SimpleExpressionChecker checks if an expression is simple
type SimpleExpressionChecker struct {
	*RecursiveAstVisitor
	Errors []string
}

// NewSimpleExpressionChecker creates a new SimpleExpressionChecker
func NewSimpleExpressionChecker() *SimpleExpressionChecker {
	return &SimpleExpressionChecker{
		RecursiveAstVisitor: &RecursiveAstVisitor{},
		Errors:              []string{},
	}
}

// VisitPipe visits a pipe expression (not allowed in simple expressions)
func (s *SimpleExpressionChecker) VisitPipe(ast *BindingPipe, context interface{}) interface{} {
	s.Errors = append(s.Errors, "pipes")
	return nil
}
