package output

import (
	"fmt"
	"ngc-go/packages/compiler/src/util"
	"regexp"
	"strings"
)

var (
	singleQuoteEscapeStringRe = regexp.MustCompile(`'|\\|\n|\r|\$`)
	legalIdentifierRe         = regexp.MustCompile(`(?i)^[$A-Z_][0-9A-Z_$]*$`)
	indentWith                = "  "
)

var binaryOperators = map[BinaryOperator]string{
	BinaryOperatorAnd:                       "&&",
	BinaryOperatorBigger:                    ">",
	BinaryOperatorBiggerEquals:              ">=",
	BinaryOperatorBitwiseOr:                 "|",
	BinaryOperatorBitwiseAnd:                "&",
	BinaryOperatorDivide:                    "/",
	BinaryOperatorAssign:                    "=",
	BinaryOperatorEquals:                    "==",
	BinaryOperatorIdentical:                 "===",
	BinaryOperatorLower:                     "<",
	BinaryOperatorLowerEquals:               "<=",
	BinaryOperatorMinus:                     "-",
	BinaryOperatorModulo:                    "%",
	BinaryOperatorExponentiation:            "**",
	BinaryOperatorMultiply:                  "*",
	BinaryOperatorNotEquals:                 "!=",
	BinaryOperatorNotIdentical:              "!==",
	BinaryOperatorNullishCoalesce:           "??",
	BinaryOperatorOr:                        "||",
	BinaryOperatorPlus:                      "+",
	BinaryOperatorIn:                        "in",
	BinaryOperatorAdditionAssignment:        "+=",
	BinaryOperatorSubtractionAssignment:     "-=",
	BinaryOperatorMultiplicationAssignment:  "*=",
	BinaryOperatorDivisionAssignment:        "/=",
	BinaryOperatorRemainderAssignment:       "%=",
	BinaryOperatorExponentiationAssignment:  "**=",
	BinaryOperatorAndAssignment:             "&&=",
	BinaryOperatorOrAssignment:              "||=",
	BinaryOperatorNullishCoalesceAssignment: "??=",
}

// EmittedLine represents a line being emitted
type EmittedLine struct {
	PartsLength int
	Parts       []string
	SrcSpans    []*util.ParseSourceSpan
	Indent      int
}

// NewEmittedLine creates a new EmittedLine
func NewEmittedLine(indent int) *EmittedLine {
	return &EmittedLine{
		PartsLength: 0,
		Parts:       []string{},
		SrcSpans:    []*util.ParseSourceSpan{},
		Indent:      indent,
	}
}

// EmitterVisitorContext represents the context for emitting code
type EmitterVisitorContext struct {
	lines  []*EmittedLine
	indent int
}

// CreateRoot creates a root EmitterVisitorContext
func CreateRootEmitterVisitorContext() *EmitterVisitorContext {
	return NewEmitterVisitorContext(0)
}

// NewEmitterVisitorContext creates a new EmitterVisitorContext
func NewEmitterVisitorContext(indent int) *EmitterVisitorContext {
	return &EmitterVisitorContext{
		lines:  []*EmittedLine{NewEmittedLine(indent)},
		indent: indent,
	}
}

// currentLine returns the current line being built
func (ctx *EmitterVisitorContext) currentLine() *EmittedLine {
	return ctx.lines[len(ctx.lines)-1]
}

// Println prints a line
func (ctx *EmitterVisitorContext) Println(from interface{}, lastPart string) {
	ctx.Print(from, lastPart, true)
}

// LineIsEmpty checks if the current line is empty
func (ctx *EmitterVisitorContext) LineIsEmpty() bool {
	return len(ctx.currentLine().Parts) == 0
}

// LineLength returns the length of the current line
func (ctx *EmitterVisitorContext) LineLength() int {
	line := ctx.currentLine()
	return line.Indent*len(indentWith) + line.PartsLength
}

// Print prints to the context
func (ctx *EmitterVisitorContext) Print(from interface{}, part string, newLine bool) {
	if len(part) > 0 {
		line := ctx.currentLine()
		line.Parts = append(line.Parts, part)
		line.PartsLength += len(part)

		var sourceSpan *util.ParseSourceSpan
		if from != nil {
			if withSpan, ok := from.(interface {
				GetSourceSpan() *util.ParseSourceSpan
			}); ok {
				sourceSpan = withSpan.GetSourceSpan()
			}
		}
		line.SrcSpans = append(line.SrcSpans, sourceSpan)
	}
	if newLine {
		ctx.lines = append(ctx.lines, NewEmittedLine(ctx.indent))
	}
}

// RemoveEmptyLastLine removes the empty last line
func (ctx *EmitterVisitorContext) RemoveEmptyLastLine() {
	if ctx.LineIsEmpty() {
		ctx.lines = ctx.lines[:len(ctx.lines)-1]
	}
}

// IncIndent increases the indent
func (ctx *EmitterVisitorContext) IncIndent() {
	ctx.indent++
	if ctx.LineIsEmpty() {
		ctx.currentLine().Indent = ctx.indent
	}
}

// DecIndent decreases the indent
func (ctx *EmitterVisitorContext) DecIndent() {
	ctx.indent--
	if ctx.LineIsEmpty() {
		ctx.currentLine().Indent = ctx.indent
	}
}

// ToSource converts the context to source code
func (ctx *EmitterVisitorContext) ToSource() string {
	lines := ctx.sourceLines()
	result := []string{}
	for _, line := range lines {
		if len(line.Parts) > 0 {
			result = append(result, createIndent(line.Indent)+strings.Join(line.Parts, ""))
		} else {
			result = append(result, "")
		}
	}
	return strings.Join(result, "\n")
}

// ToSourceMapGenerator converts the context to a source map generator
func (ctx *EmitterVisitorContext) ToSourceMapGenerator(genFilePath string, startsAtLine int) (*SourceMapGenerator, error) {
	mapGen := NewSourceMapGenerator(&genFilePath)

	firstOffsetMapped := false
	mapFirstOffsetIfNeeded := func() error {
		if !firstOffsetMapped {
			space := " "
			mapGen.AddSource(genFilePath, &space)
			if err := mapGen.AddMapping(0, &genFilePath, intPtr(0), intPtr(0)); err != nil {
				return err
			}
			firstOffsetMapped = true
		}
		return nil
	}

	for i := 0; i < startsAtLine; i++ {
		mapGen.AddLine()
		if err := mapFirstOffsetIfNeeded(); err != nil {
			return nil, err
		}
	}

	lines := ctx.sourceLines()
	for lineIdx, line := range lines {
		mapGen.AddLine()

		spans := line.SrcSpans
		parts := line.Parts
		col0 := line.Indent * len(indentWith)
		spanIdx := 0

		// skip leading parts without source spans
		for spanIdx < len(spans) && spans[spanIdx] == nil {
			col0 += len(parts[spanIdx])
			spanIdx++
		}

		if spanIdx < len(spans) && lineIdx == 0 && col0 == 0 {
			firstOffsetMapped = true
		} else {
			if err := mapFirstOffsetIfNeeded(); err != nil {
				return nil, err
			}
		}

		for spanIdx < len(spans) {
			span := spans[spanIdx]
			if span == nil {
				spanIdx++
				continue
			}

			source := span.Start.File
			sourceLine := span.Start.Line
			sourceCol := span.Start.Col

			sourceURL := source.URL
			content := source.Content
			mapGen.AddSource(sourceURL, &content)
			if err := mapGen.AddMapping(col0, &sourceURL, &sourceLine, &sourceCol); err != nil {
				return nil, err
			}

			col0 += len(parts[spanIdx])
			spanIdx++

			// assign parts without span or the same span to the previous segment
			for spanIdx < len(spans) && (spans[spanIdx] == span || spans[spanIdx] == nil) {
				col0 += len(parts[spanIdx])
				spanIdx++
			}
		}
	}

	return mapGen, nil
}

// SpanOf returns the source span at the given line and column
func (ctx *EmitterVisitorContext) SpanOf(lineNum, column int) *util.ParseSourceSpan {
	if lineNum < len(ctx.lines) {
		emittedLine := ctx.lines[lineNum]
		columnsLeft := column - len(createIndent(emittedLine.Indent))
		for partIndex := 0; partIndex < len(emittedLine.Parts); partIndex++ {
			part := emittedLine.Parts[partIndex]
			if len(part) > columnsLeft {
				return emittedLine.SrcSpans[partIndex]
			}
			columnsLeft -= len(part)
		}
	}
	return nil
}

// sourceLines returns the source lines (excluding empty last line)
func (ctx *EmitterVisitorContext) sourceLines() []*EmittedLine {
	if len(ctx.lines) > 0 && len(ctx.lines[len(ctx.lines)-1].Parts) == 0 {
		return ctx.lines[:len(ctx.lines)-1]
	}
	return ctx.lines
}

// AbstractEmitterVisitor is the base class for emitters
type AbstractEmitterVisitor struct {
	lastIfCondition       OutputExpression
	escapeDollarInStrings bool
}

// NewAbstractEmitterVisitor creates a new AbstractEmitterVisitor
func NewAbstractEmitterVisitor(escapeDollarInStrings bool) *AbstractEmitterVisitor {
	return &AbstractEmitterVisitor{
		lastIfCondition:       nil,
		escapeDollarInStrings: escapeDollarInStrings,
	}
}

// getContext converts interface{} to *EmitterVisitorContext
func (v *AbstractEmitterVisitor) getContext(context interface{}) *EmitterVisitorContext {
	if ctx, ok := context.(*EmitterVisitorContext); ok {
		return ctx
	}
	panic("context must be *EmitterVisitorContext")
}

// PrintLeadingComments prints leading comments
func (v *AbstractEmitterVisitor) PrintLeadingComments(stmt OutputStatement, ctx *EmitterVisitorContext) {
	// TODO: Implement when LeadingComments are available
}

// VisitExpressionStmt visits an expression statement
func (v *AbstractEmitterVisitor) VisitExpressionStmt(stmt *ExpressionStatement, context interface{}) interface{} {
	ctx := v.getContext(context)
	v.PrintLeadingComments(stmt, ctx)
	stmt.Expr.VisitExpression(v, ctx)
	ctx.Println(stmt, ";")
	return nil
}

// VisitReturnStmt visits a return statement
func (v *AbstractEmitterVisitor) VisitReturnStmt(stmt *ReturnStatement, context interface{}) interface{} {
	ctx := v.getContext(context)
	v.PrintLeadingComments(stmt, ctx)
	ctx.Print(stmt, "return ", false)
	stmt.Value.VisitExpression(v, ctx)
	ctx.Println(stmt, ";")
	return nil
}

// VisitIfStmt visits an if statement
func (v *AbstractEmitterVisitor) VisitIfStmt(stmt *IfStmt, context interface{}) interface{} {
	ctx := v.getContext(context)
	v.PrintLeadingComments(stmt, ctx)
	ctx.Print(stmt, "if (", false)
	v.lastIfCondition = stmt.Condition
	stmt.Condition.VisitExpression(v, ctx)
	v.lastIfCondition = nil
	ctx.Print(stmt, ") {", false)

	hasElseCase := len(stmt.FalseCase) > 0
	if len(stmt.TrueCase) <= 1 && !hasElseCase {
		ctx.Print(stmt, " ", false)
		v.VisitAllStatements(stmt.TrueCase, ctx)
		ctx.RemoveEmptyLastLine()
		ctx.Print(stmt, " ", false)
	} else {
		ctx.Println(nil, "")
		ctx.IncIndent()
		v.VisitAllStatements(stmt.TrueCase, ctx)
		ctx.DecIndent()
		if hasElseCase {
			ctx.Println(stmt, "} else {")
			ctx.IncIndent()
			v.VisitAllStatements(stmt.FalseCase, ctx)
			ctx.DecIndent()
		}
	}
	ctx.Println(stmt, "}")
	return nil
}

// VisitInvokeFunctionExpr visits an invoke function expression
func (v *AbstractEmitterVisitor) VisitInvokeFunctionExpr(expr *InvokeFunctionExpr, context interface{}) interface{} {
	ctx := v.getContext(context)
	shouldParenthesize := false
	if _, ok := expr.Fn.(*ArrowFunctionExpr); ok {
		shouldParenthesize = true
	}

	if shouldParenthesize {
		ctx.Print(expr.Fn, "(", false)
	}
	expr.Fn.VisitExpression(v, ctx)
	if shouldParenthesize {
		ctx.Print(expr.Fn, ")", false)
	}
	ctx.Print(expr, "(", false)
	v.VisitAllExpressions(expr.Args, ctx, ",")
	ctx.Print(expr, ")", false)
	return nil
}

// VisitTaggedTemplateLiteralExpr visits a tagged template literal expression
func (v *AbstractEmitterVisitor) VisitTaggedTemplateLiteralExpr(expr *TaggedTemplateLiteralExpr, context interface{}) interface{} {
	ctx := v.getContext(context)
	expr.Tag.VisitExpression(v, ctx)
	expr.Template.VisitExpression(v, ctx)
	return nil
}

// VisitTemplateLiteralExpr visits a template literal expression
func (v *AbstractEmitterVisitor) VisitTemplateLiteralExpr(expr *TemplateLiteralExpr, context interface{}) interface{} {
	ctx := v.getContext(context)
	ctx.Print(expr, "`", false)
	for i := 0; i < len(expr.Elements); i++ {
		expr.Elements[i].VisitExpression(v, ctx)
		if i < len(expr.Expressions) {
			expression := expr.Expressions[i]
			ctx.Print(expression, "${", false)
			expression.VisitExpression(v, ctx)
			ctx.Print(expression, "}", false)
		}
	}
	ctx.Print(expr, "`", false)
	return nil
}

// VisitTemplateLiteralElementExpr visits a template literal element expression
func (v *AbstractEmitterVisitor) VisitTemplateLiteralElementExpr(expr *TemplateLiteralElementExpr, context interface{}) interface{} {
	ctx := v.getContext(context)
	ctx.Print(expr, expr.RawText, false)
	return nil
}

// VisitWrappedNodeExpr visits a wrapped node expression
func (v *AbstractEmitterVisitor) VisitWrappedNodeExpr(ast *WrappedNodeExpr, context interface{}) interface{} {
	_ = v.getContext(context)
	panic("Abstract emitter cannot visit WrappedNodeExpr.")
}

// VisitTypeofExpr visits a typeof expression
func (v *AbstractEmitterVisitor) VisitTypeofExpr(expr *TypeofExpr, context interface{}) interface{} {
	ctx := v.getContext(context)
	ctx.Print(expr, "typeof ", false)
	expr.Expr.VisitExpression(v, ctx)
	return nil
}

// VisitVoidExpr visits a void expression
func (v *AbstractEmitterVisitor) VisitVoidExpr(expr *VoidExpr, context interface{}) interface{} {
	ctx := v.getContext(context)
	ctx.Print(expr, "void ", false)
	expr.Expr.VisitExpression(v, ctx)
	return nil
}

// VisitReadVarExpr visits a read variable expression
func (v *AbstractEmitterVisitor) VisitReadVarExpr(ast *ReadVarExpr, context interface{}) interface{} {
	ctx := v.getContext(context)
	ctx.Print(ast, ast.Name, false)
	return nil
}

// VisitInstantiateExpr visits an instantiate expression
func (v *AbstractEmitterVisitor) VisitInstantiateExpr(ast *InstantiateExpr, context interface{}) interface{} {
	ctx := v.getContext(context)
	ctx.Print(ast, "new ", false)
	ast.ClassExpr.VisitExpression(v, ctx)
	ctx.Print(ast, "(", false)
	v.VisitAllExpressions(ast.Args, ctx, ",")
	ctx.Print(ast, ")", false)
	return nil
}

// VisitLiteralExpr visits a literal expression
func (v *AbstractEmitterVisitor) VisitLiteralExpr(ast *LiteralExpr, context interface{}) interface{} {
	ctx := v.getContext(context)
	switch val := ast.Value.(type) {
	case string:
		ctx.Print(ast, EscapeIdentifier(val, v.escapeDollarInStrings, true), false)
	default:
		ctx.Print(ast, fmt.Sprintf("%v", val), false)
	}
	return nil
}

// VisitRegularExpressionLiteral visits a regular expression literal
func (v *AbstractEmitterVisitor) VisitRegularExpressionLiteral(ast *RegularExpressionLiteralExpr, context interface{}) interface{} {
	ctx := v.getContext(context)
	flags := ""
	if ast.Flags != nil {
		flags = *ast.Flags
	}
	ctx.Print(ast, fmt.Sprintf("/%s/%s", ast.Body, flags), false)
	return nil
}

// VisitLocalizedString visits a localized string
func (v *AbstractEmitterVisitor) VisitLocalizedString(ast *LocalizedString, context interface{}) interface{} {
	ctx := v.getContext(context)
	// TODO: Implement when LocalizedString is fully implemented
	ctx.Print(ast, "$localize `", false)
	ctx.Print(ast, "`", false)
	return nil
}

// VisitConditionalExpr visits a conditional expression
func (v *AbstractEmitterVisitor) VisitConditionalExpr(ast *ConditionalExpr, context interface{}) interface{} {
	ctx := v.getContext(context)
	ctx.Print(ast, "(", false)
	ast.Condition.VisitExpression(v, ctx)
	ctx.Print(ast, "? ", false)
	ast.TrueCase.VisitExpression(v, ctx)
	ctx.Print(ast, ": ", false)
	if ast.FalseCase != nil {
		ast.FalseCase.VisitExpression(v, ctx)
	}
	ctx.Print(ast, ")", false)
	return nil
}

// VisitDynamicImportExpr visits a dynamic import expression
func (v *AbstractEmitterVisitor) VisitDynamicImportExpr(ast *DynamicImportExpr, context interface{}) interface{} {
	ctx := v.getContext(context)
	if url, ok := ast.URL.(string); ok {
		ctx.Print(ast, fmt.Sprintf("import(%s)", url), false)
	} else if expr, ok := ast.URL.(OutputExpression); ok {
		ctx.Print(ast, "import(", false)
		expr.VisitExpression(v, ctx)
		ctx.Print(ast, ")", false)
	}
	return nil
}

// VisitNotExpr visits a not expression
func (v *AbstractEmitterVisitor) VisitNotExpr(ast *NotExpr, context interface{}) interface{} {
	ctx := v.getContext(context)
	ctx.Print(ast, "!", false)
	ast.Condition.VisitExpression(v, ctx)
	return nil
}

// VisitUnaryOperatorExpr visits a unary operator expression
func (v *AbstractEmitterVisitor) VisitUnaryOperatorExpr(ast *UnaryOperatorExpr, context interface{}) interface{} {
	ctx := v.getContext(context)
	var opStr string
	switch ast.Operator {
	case UnaryOperatorPlus:
		opStr = "+"
	case UnaryOperatorMinus:
		opStr = "-"
	default:
		panic(fmt.Sprintf("Unknown operator %d", ast.Operator))
	}

	parens := ast != v.lastIfCondition
	if parens {
		ctx.Print(ast, "(", false)
	}
	ctx.Print(ast, opStr, false)
	ast.Expr.VisitExpression(v, ctx)
	if parens {
		ctx.Print(ast, ")", false)
	}
	return nil
}

// VisitBinaryOperatorExpr visits a binary operator expression
func (v *AbstractEmitterVisitor) VisitBinaryOperatorExpr(ast *BinaryOperatorExpr, context interface{}) interface{} {
	ctx := v.getContext(context)
	operator, ok := binaryOperators[ast.Operator]
	if !ok {
		panic(fmt.Sprintf("Unknown operator %d", ast.Operator))
	}

	parens := ast != v.lastIfCondition
	if parens {
		ctx.Print(ast, "(", false)
	}
	ast.Lhs.VisitExpression(v, ctx)
	ctx.Print(ast, " "+operator+" ", false)
	ast.Rhs.VisitExpression(v, ctx)
	if parens {
		ctx.Print(ast, ")", false)
	}
	return nil
}

// VisitReadPropExpr visits a read property expression
func (v *AbstractEmitterVisitor) VisitReadPropExpr(ast *ReadPropExpr, context interface{}) interface{} {
	ctx := v.getContext(context)
	ast.Receiver.VisitExpression(v, ctx)
	ctx.Print(ast, ".", false)
	ctx.Print(ast, ast.Name, false)
	return nil
}

// VisitReadKeyExpr visits a read key expression
func (v *AbstractEmitterVisitor) VisitReadKeyExpr(ast *ReadKeyExpr, context interface{}) interface{} {
	ctx := v.getContext(context)
	ast.Receiver.VisitExpression(v, ctx)
	ctx.Print(ast, "[", false)
	ast.Index.VisitExpression(v, ctx)
	ctx.Print(ast, "]", false)
	return nil
}

// VisitLiteralArrayExpr visits a literal array expression
func (v *AbstractEmitterVisitor) VisitLiteralArrayExpr(ast *LiteralArrayExpr, context interface{}) interface{} {
	ctx := v.getContext(context)
	ctx.Print(ast, "[", false)
	v.VisitAllExpressions(ast.Entries, ctx, ",")
	ctx.Print(ast, "]", false)
	return nil
}

// VisitLiteralMapExpr visits a literal map expression
func (v *AbstractEmitterVisitor) VisitLiteralMapExpr(ast *LiteralMapExpr, context interface{}) interface{} {
	ctx := v.getContext(context)
	ctx.Print(ast, "{", false)
	handler := func(entry *LiteralMapEntry) {
		ctx.Print(ast, EscapeIdentifier(entry.Key, v.escapeDollarInStrings, entry.Quoted)+":", false)
		entry.Value.VisitExpression(v, ctx)
	}
	v.VisitAllObjects(handler, ast.Entries, ctx, ",")
	ctx.Print(ast, "}", false)
	return nil
}

// VisitCommaExpr visits a comma expression
func (v *AbstractEmitterVisitor) VisitCommaExpr(ast *CommaExpr, context interface{}) interface{} {
	ctx := v.getContext(context)
	ctx.Print(ast, "(", false)
	v.VisitAllExpressions(ast.Parts, ctx, ",")
	ctx.Print(ast, ")", false)
	return nil
}

// VisitParenthesizedExpr visits a parenthesized expression
func (v *AbstractEmitterVisitor) VisitParenthesizedExpr(ast *ParenthesizedExpr, context interface{}) interface{} {
	ctx := v.getContext(context)
	// We parenthesize everything regardless of an explicit ParenthesizedExpr, so we can just visit
	// the inner expression.
	ast.Expr.VisitExpression(v, ctx)
	return nil
}

// VisitAllExpressions visits all expressions
func (v *AbstractEmitterVisitor) VisitAllExpressions(expressions []OutputExpression, ctx *EmitterVisitorContext, separator string) {
	v.VisitAllObjects(func(expr OutputExpression) {
		expr.VisitExpression(v, ctx)
	}, expressions, ctx, separator)
}

// VisitAllObjects visits all objects with a handler
// This is a helper that works with interface{} and type assertions
func (v *AbstractEmitterVisitor) VisitAllObjects(
	handler interface{},
	expressions interface{},
	ctx *EmitterVisitorContext,
	separator string,
) {
	incrementedIndent := false

	// Type assertion based on the type of expressions
	switch exprs := expressions.(type) {
	case []OutputExpression:
		for i := 0; i < len(exprs); i++ {
			if i > 0 {
				if ctx.LineLength() > 80 {
					ctx.Print(nil, separator, true)
					if !incrementedIndent {
						ctx.IncIndent()
						ctx.IncIndent()
						incrementedIndent = true
					}
				} else {
					ctx.Print(nil, separator, false)
				}
			}
			if h, ok := handler.(func(OutputExpression)); ok {
				h(exprs[i])
			}
		}
	case []*LiteralMapEntry:
		for i := 0; i < len(exprs); i++ {
			if i > 0 {
				if ctx.LineLength() > 80 {
					ctx.Print(nil, separator, true)
					if !incrementedIndent {
						ctx.IncIndent()
						ctx.IncIndent()
						incrementedIndent = true
					}
				} else {
					ctx.Print(nil, separator, false)
				}
			}
			if h, ok := handler.(func(*LiteralMapEntry)); ok {
				h(exprs[i])
			}
		}
	}

	if incrementedIndent {
		ctx.DecIndent()
		ctx.DecIndent()
	}
}

// VisitAllStatements visits all statements
func (v *AbstractEmitterVisitor) VisitAllStatements(statements []OutputStatement, ctx *EmitterVisitorContext) {
	for _, stmt := range statements {
		stmt.VisitStatement(v, ctx)
	}
}

// Missing ExpressionVisitor methods - to be implemented by concrete emitters
func (v *AbstractEmitterVisitor) VisitArrowFunctionExpr(ast *ArrowFunctionExpr, context interface{}) interface{} {
	_ = v.getContext(context)
	panic("VisitArrowFunctionExpr must be implemented by concrete emitter")
}

func (v *AbstractEmitterVisitor) VisitFunctionExpr(ast *FunctionExpr, context interface{}) interface{} {
	_ = v.getContext(context)
	panic("VisitFunctionExpr must be implemented by concrete emitter")
}

func (v *AbstractEmitterVisitor) VisitExternalExpr(ast *ExternalExpr, context interface{}) interface{} {
	_ = v.getContext(context)
	panic("VisitExternalExpr must be implemented by concrete emitter")
}

// Missing StatementVisitor methods
func (v *AbstractEmitterVisitor) VisitDeclareVarStmt(stmt *DeclareVarStmt, context interface{}) interface{} {
	_ = v.getContext(context)
	panic("VisitDeclareVarStmt must be implemented by concrete emitter")
}

func (v *AbstractEmitterVisitor) VisitDeclareFunctionStmt(stmt *DeclareFunctionStmt, context interface{}) interface{} {
	_ = v.getContext(context)
	panic("VisitDeclareFunctionStmt must be implemented by concrete emitter")
}

// EscapeIdentifier escapes an identifier
func EscapeIdentifier(input string, escapeDollar bool, alwaysQuote bool) string {
	if input == "" {
		return ""
	}

	body := singleQuoteEscapeStringRe.ReplaceAllStringFunc(input, func(match string) string {
		if match == "$" {
			if escapeDollar {
				return "\\$"
			}
			return "$"
		} else if match == "\n" {
			return "\\n"
		} else if match == "\r" {
			return "\\r"
		} else {
			return "\\" + match
		}
	})

	requiresQuotes := alwaysQuote || !legalIdentifierRe.MatchString(body)
	if requiresQuotes {
		return "'" + body + "'"
	}
	return body
}

// createIndent creates an indent string
func createIndent(count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += indentWith
	}
	return result
}

// Helper function to create int pointer
func intPtr(i int) *int {
	return &i
}
