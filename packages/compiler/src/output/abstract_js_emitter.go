package output

import (
	"fmt"
	"strings"
)

const makeTemplateObjectPolyfill = "(this&&this.__makeTemplateObject||function(e,t){return Object.defineProperty?Object.defineProperty(e,\"raw\",{value:t}):e.raw=t,e})"

// AbstractJsEmitterVisitor is the base class for JavaScript emitters
type AbstractJsEmitterVisitor struct {
	*AbstractEmitterVisitor
}

// NewAbstractJsEmitterVisitor creates a new AbstractJsEmitterVisitor
func NewAbstractJsEmitterVisitor() *AbstractJsEmitterVisitor {
	return &AbstractJsEmitterVisitor{
		AbstractEmitterVisitor: NewAbstractEmitterVisitor(false),
	}
}

// VisitWrappedNodeExpr visits a wrapped node expression
func (v *AbstractJsEmitterVisitor) VisitWrappedNodeExpr(ast *WrappedNodeExpr, context interface{}) interface{} {
	panic("Cannot emit a WrappedNodeExpr in Javascript.")
}

// VisitDeclareVarStmt visits a declare variable statement
func (v *AbstractJsEmitterVisitor) VisitDeclareVarStmt(stmt *DeclareVarStmt, context interface{}) interface{} {
	ctx := v.getContext(context)
	ctx.Print(stmt, fmt.Sprintf("var %s", stmt.Name), false)
	if stmt.Value != nil {
		ctx.Print(stmt, " = ", false)
		stmt.Value.VisitExpression(v, ctx)
	}
	ctx.Println(stmt, ";")
	return nil
}

// VisitTaggedTemplateLiteralExpr visits a tagged template literal expression
func (v *AbstractJsEmitterVisitor) VisitTaggedTemplateLiteralExpr(expr *TaggedTemplateLiteralExpr, context interface{}) interface{} {
	ctx := v.getContext(context)
	// The following convoluted piece of code is effectively the downlevelled equivalent of
	// ```
	// tag`...`
	// ```
	// which is effectively like:
	// ```
	// tag(__makeTemplateObject(cooked, raw), expression1, expression2, ...);
	// ```
	elements := expr.Template.Elements
	expr.Tag.VisitExpression(v, ctx)
	ctx.Print(expr, fmt.Sprintf("(%s(", makeTemplateObjectPolyfill), false)

	cookedParts := []string{}
	for _, part := range elements {
		cookedParts = append(cookedParts, EscapeIdentifier(part.Text, false, true))
	}
	ctx.Print(expr, fmt.Sprintf("[%s], ", strings.Join(cookedParts, ", ")), false)

	rawParts := []string{}
	for _, part := range elements {
		rawParts = append(rawParts, EscapeIdentifier(part.RawText, false, true))
	}
	ctx.Print(expr, fmt.Sprintf("[%s])", strings.Join(rawParts, ", ")), false)

	for _, expression := range expr.Template.Expressions {
		ctx.Print(expr, ", ", false)
		expression.VisitExpression(v, ctx)
	}
	ctx.Print(expr, ")", false)
	return nil
}

// VisitTemplateLiteralExpr visits a template literal expression
func (v *AbstractJsEmitterVisitor) VisitTemplateLiteralExpr(expr *TemplateLiteralExpr, context interface{}) interface{} {
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
func (v *AbstractJsEmitterVisitor) VisitTemplateLiteralElementExpr(expr *TemplateLiteralElementExpr, context interface{}) interface{} {
	ctx := v.getContext(context)
	ctx.Print(expr, expr.RawText, false)
	return nil
}

// VisitFunctionExpr visits a function expression
func (v *AbstractJsEmitterVisitor) VisitFunctionExpr(ast *FunctionExpr, context interface{}) interface{} {
	ctx := v.getContext(context)
	namePart := ""
	if ast.Name != nil {
		namePart = " " + *ast.Name
	}
	ctx.Print(ast, fmt.Sprintf("function%s(", namePart), false)
	v.visitParams(ast.Params, ctx)
	ctx.Println(ast, ") {")
	ctx.IncIndent()
	v.VisitAllStatements(ast.Statements, ctx)
	ctx.DecIndent()
	ctx.Print(ast, "}", false)
	return nil
}

// VisitArrowFunctionExpr visits an arrow function expression
func (v *AbstractJsEmitterVisitor) VisitArrowFunctionExpr(ast *ArrowFunctionExpr, context interface{}) interface{} {
	ctx := v.getContext(context)
	ctx.Print(ast, "(", false)
	v.visitParams(ast.Params, ctx)
	ctx.Print(ast, ") =>", false)

	if stmts, ok := ast.Body.([]OutputStatement); ok {
		ctx.Println(ast, "{")
		ctx.IncIndent()
		v.VisitAllStatements(stmts, ctx)
		ctx.DecIndent()
		ctx.Print(ast, "}", false)
	} else if expr, ok := ast.Body.(OutputExpression); ok {
		isObjectLiteral := false
		if _, ok := expr.(*LiteralMapExpr); ok {
			isObjectLiteral = true
		}

		if isObjectLiteral {
			ctx.Print(ast, "(", false)
		}

		expr.VisitExpression(v, ctx)

		if isObjectLiteral {
			ctx.Print(ast, ")", false)
		}
	}

	return nil
}

// VisitDeclareFunctionStmt visits a declare function statement
func (v *AbstractJsEmitterVisitor) VisitDeclareFunctionStmt(stmt *DeclareFunctionStmt, context interface{}) interface{} {
	ctx := v.getContext(context)
	ctx.Print(stmt, fmt.Sprintf("function %s(", stmt.Name), false)
	v.visitParams(stmt.Params, ctx)
	ctx.Println(stmt, ") {")
	ctx.IncIndent()
	v.VisitAllStatements(stmt.Statements, ctx)
	ctx.DecIndent()
	ctx.Println(stmt, "}")
	return nil
}

// VisitLocalizedString visits a localized string
func (v *AbstractJsEmitterVisitor) VisitLocalizedString(ast *LocalizedString, context interface{}) interface{} {
	ctx := v.getContext(context)
	// The following convoluted piece of code is effectively the downlevelled equivalent of
	// ```
	// $localize `...`
	// ```
	// which is effectively like:
	// ```
	// $localize(__makeTemplateObject(cooked, raw), expression1, expression2, ...);
	// ```
	ctx.Print(ast, fmt.Sprintf("$localize(%s(", makeTemplateObjectPolyfill), false)

	// TODO: Implement when LocalizedString is fully implemented
	// parts := []CookedRawString{ast.SerializeI18nHead()}
	// for i := 1; i < len(ast.MessageParts); i++ {
	// 	parts = append(parts, ast.SerializeI18nTemplatePart(i))
	// }
	// cookedParts := []string{}
	// for _, part := range parts {
	// 	cookedParts = append(cookedParts, EscapeIdentifier(part.Cooked, false, true))
	// }
	// ctx.Print(ast, fmt.Sprintf("[%s], ", strings.Join(cookedParts, ", ")), false)
	// rawParts := []string{}
	// for _, part := range parts {
	// 	rawParts = append(rawParts, EscapeIdentifier(part.Raw, false, true))
	// }
	// ctx.Print(ast, fmt.Sprintf("[%s])", strings.Join(rawParts, ", ")), false)

	// Placeholder implementation
	ctx.Print(ast, "[], []", false)

	// TODO: Add expressions when LocalizedString is fully implemented
	// for _, expression := range ast.Expressions {
	// 	ctx.Print(ast, ", ", false)
	// 	expression.VisitExpression(v, ctx)
	// }

	ctx.Print(ast, ")", false)
	return nil
}

// visitParams visits function parameters
func (v *AbstractJsEmitterVisitor) visitParams(params []*FnParam, ctx *EmitterVisitorContext) {
	v.VisitAllObjects(func(param *FnParam) {
		ctx.Print(nil, param.Name, false)
	}, params, ctx, ",")
}
