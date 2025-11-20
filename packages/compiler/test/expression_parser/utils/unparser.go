package utils

import (
	"fmt"
	"strings"

	"ngc-go/packages/compiler/src/expression_parser"
)

// Unparser is a visitor that unparses AST to string (always uses double quotes for strings)
type Unparser struct {
	expression strings.Builder
}

// NewUnparser creates a new Unparser
func NewUnparser() *Unparser {
	return &Unparser{}
}

// Unparse unparses an AST to string
func (u *Unparser) Unparse(ast expression_parser.AST) string {
	u.expression.Reset()
	ast.Visit(u, nil)
	return u.expression.String()
}

// Visit is the default visit method
func (u *Unparser) Visit(ast expression_parser.AST, context interface{}) interface{} {
	ast.Visit(u, context)
	return nil
}

// VisitUnary visits a unary expression
func (u *Unparser) VisitUnary(ast *expression_parser.Unary, context interface{}) interface{} {
	u.expression.WriteString(ast.Operator)
	ast.Expr.Visit(u, context)
	return nil
}

// VisitBinary visits a binary expression
func (u *Unparser) VisitBinary(ast *expression_parser.Binary, context interface{}) interface{} {
	ast.Left.Visit(u, context)
	u.expression.WriteString(" ")
	u.expression.WriteString(ast.Operation)
	u.expression.WriteString(" ")
	ast.Right.Visit(u, context)
	return nil
}

// VisitChain visits a chain expression
func (u *Unparser) VisitChain(ast *expression_parser.Chain, context interface{}) interface{} {
	len := len(ast.Expressions)
	for i, expr := range ast.Expressions {
		expr.Visit(u, context)
		if i < len-1 {
			u.expression.WriteString("; ")
		} else {
			u.expression.WriteString(";")
		}
	}
	return nil
}

// VisitConditional visits a conditional expression
func (u *Unparser) VisitConditional(ast *expression_parser.Conditional, context interface{}) interface{} {
	ast.Condition.Visit(u, context)
	u.expression.WriteString(" ? ")
	ast.TrueExp.Visit(u, context)
	u.expression.WriteString(" : ")
	ast.FalseExp.Visit(u, context)
	return nil
}

// VisitThisReceiver visits a this receiver
func (u *Unparser) VisitThisReceiver(ast *expression_parser.ThisReceiver, context interface{}) interface{} {
	// This receiver is implicit, so nothing to write
	return nil
}

// VisitImplicitReceiver visits an implicit receiver
func (u *Unparser) VisitImplicitReceiver(ast *expression_parser.ImplicitReceiver, context interface{}) interface{} {
	// Implicit receiver is implicit, so nothing to write
	return nil
}

// VisitInterpolation visits an interpolation
func (u *Unparser) VisitInterpolation(ast *expression_parser.Interpolation, context interface{}) interface{} {
	for i := 0; i < len(ast.Strings); i++ {
		u.expression.WriteString(ast.Strings[i])
		if i < len(ast.Expressions) {
			u.expression.WriteString("{{ ")
			ast.Expressions[i].Visit(u, context)
			u.expression.WriteString(" }}")
		}
	}
	return nil
}

// VisitKeyedRead visits a keyed read
func (u *Unparser) VisitKeyedRead(ast *expression_parser.KeyedRead, context interface{}) interface{} {
	ast.Receiver.Visit(u, context)
	u.expression.WriteString("[")
	ast.Key.Visit(u, context)
	u.expression.WriteString("]")
	return nil
}

// VisitLiteralArray visits a literal array
func (u *Unparser) VisitLiteralArray(ast *expression_parser.LiteralArray, context interface{}) interface{} {
	u.expression.WriteString("[")
	for i, expr := range ast.Expressions {
		if i > 0 {
			u.expression.WriteString(", ")
		}
		expr.Visit(u, context)
	}
	u.expression.WriteString("]")
	return nil
}

// VisitLiteralMap visits a literal map
func (u *Unparser) VisitLiteralMap(ast *expression_parser.LiteralMap, context interface{}) interface{} {
	u.expression.WriteString("{")
	for i := 0; i < len(ast.Keys); i++ {
		if i > 0 {
			u.expression.WriteString(", ")
		}
		key := ast.Keys[i]
		if key.Quoted {
			// Use JSON encoding for quoted keys
			u.expression.WriteString(fmt.Sprintf(`"%s"`, strings.ReplaceAll(key.Key, `"`, `\"`)))
		} else {
			u.expression.WriteString(key.Key)
		}
		u.expression.WriteString(": ")
		ast.Values[i].Visit(u, context)
	}
	u.expression.WriteString("}")
	return nil
}

// VisitLiteralPrimitive visits a literal primitive
func (u *Unparser) VisitLiteralPrimitive(ast *expression_parser.LiteralPrimitive, context interface{}) interface{} {
	if ast.Value == nil {
		u.expression.WriteString("null")
		return nil
	}

	switch v := ast.Value.(type) {
	case float64:
		u.expression.WriteString(fmt.Sprintf("%g", v))
	case int:
		u.expression.WriteString(fmt.Sprintf("%d", v))
	case int64:
		u.expression.WriteString(fmt.Sprintf("%d", v))
	case bool:
		u.expression.WriteString(fmt.Sprintf("%t", v))
	case string:
		// Always use double quotes in unparse (unlike serialize which uses single quotes)
		u.expression.WriteString(`"`)
		u.expression.WriteString(strings.ReplaceAll(v, `"`, `\"`))
		u.expression.WriteString(`"`)
	default:
		// Check for undefined
		if _, ok := ast.Value.(expression_parser.UndefinedValue); ok {
			u.expression.WriteString("undefined")
		} else {
			u.expression.WriteString(fmt.Sprintf("%v", v))
		}
	}
	return nil
}

// VisitPipe visits a pipe expression
func (u *Unparser) VisitPipe(ast *expression_parser.BindingPipe, context interface{}) interface{} {
	u.expression.WriteString("(")
	ast.Exp.Visit(u, context)
	u.expression.WriteString(" | ")
	u.expression.WriteString(ast.Name)
	for _, arg := range ast.Args {
		u.expression.WriteString(":")
		arg.Visit(u, context)
	}
	u.expression.WriteString(")")
	return nil
}

// VisitPrefixNot visits a prefix not
func (u *Unparser) VisitPrefixNot(ast *expression_parser.PrefixNot, context interface{}) interface{} {
	u.expression.WriteString("!")
	ast.Expression.Visit(u, context)
	return nil
}

// VisitNonNullAssert visits a non-null assertion
func (u *Unparser) VisitNonNullAssert(ast *expression_parser.NonNullAssert, context interface{}) interface{} {
	ast.Expression.Visit(u, context)
	u.expression.WriteString("!")
	return nil
}

// VisitPropertyRead visits a property read
func (u *Unparser) VisitPropertyRead(ast *expression_parser.PropertyRead, context interface{}) interface{} {
	ast.Receiver.Visit(u, context)
	// In TypeScript, ThisReceiver extends ImplicitReceiver, so instanceof ImplicitReceiver
	// returns true for both ImplicitReceiver and ThisReceiver
	_, isImplicit := ast.Receiver.(*expression_parser.ImplicitReceiver)
	_, isThis := ast.Receiver.(*expression_parser.ThisReceiver)
	if isImplicit || isThis {
		u.expression.WriteString(ast.Name)
	} else {
		u.expression.WriteString(".")
		u.expression.WriteString(ast.Name)
	}
	return nil
}

// VisitSafePropertyRead visits a safe property read
func (u *Unparser) VisitSafePropertyRead(ast *expression_parser.SafePropertyRead, context interface{}) interface{} {
	ast.Receiver.Visit(u, context)
	u.expression.WriteString("?.")
	u.expression.WriteString(ast.Name)
	return nil
}

// VisitSafeKeyedRead visits a safe keyed read
func (u *Unparser) VisitSafeKeyedRead(ast *expression_parser.SafeKeyedRead, context interface{}) interface{} {
	ast.Receiver.Visit(u, context)
	u.expression.WriteString("?.[")
	ast.Key.Visit(u, context)
	u.expression.WriteString("]")
	return nil
}

// VisitCall visits a call
func (u *Unparser) VisitCall(ast *expression_parser.Call, context interface{}) interface{} {
	ast.Receiver.Visit(u, context)
	u.expression.WriteString("(")
	for i, arg := range ast.Args {
		if i > 0 {
			u.expression.WriteString(", ")
		}
		arg.Visit(u, context)
	}
	u.expression.WriteString(")")
	return nil
}

// VisitSafeCall visits a safe call
func (u *Unparser) VisitSafeCall(ast *expression_parser.SafeCall, context interface{}) interface{} {
	ast.Receiver.Visit(u, context)
	u.expression.WriteString("?.(")
	for i, arg := range ast.Args {
		if i > 0 {
			u.expression.WriteString(", ")
		}
		arg.Visit(u, context)
	}
	u.expression.WriteString(")")
	return nil
}

// VisitTypeofExpression visits a typeof expression
func (u *Unparser) VisitTypeofExpression(ast *expression_parser.TypeofExpression, context interface{}) interface{} {
	u.expression.WriteString("typeof ")
	ast.Expression.Visit(u, context)
	return nil
}

// VisitVoidExpression visits a void expression
func (u *Unparser) VisitVoidExpression(ast *expression_parser.VoidExpression, context interface{}) interface{} {
	u.expression.WriteString("void ")
	ast.Expression.Visit(u, context)
	return nil
}

// VisitRegularExpressionLiteral visits a regular expression literal
func (u *Unparser) VisitRegularExpressionLiteral(ast *expression_parser.RegularExpressionLiteral, context interface{}) interface{} {
	u.expression.WriteString("/")
	u.expression.WriteString(ast.Body)
	u.expression.WriteString("/")
	if ast.Flags != nil {
		u.expression.WriteString(*ast.Flags)
	}
	return nil
}

// VisitASTWithSource visits an AST with source
func (u *Unparser) VisitASTWithSource(ast *expression_parser.ASTWithSource, context interface{}) interface{} {
	return ast.AST.Visit(u, context)
}

// VisitTemplateLiteral visits a template literal
func (u *Unparser) VisitTemplateLiteral(ast *expression_parser.TemplateLiteral, context interface{}) interface{} {
	u.expression.WriteString("`")
	for i := 0; i < len(ast.Elements); i++ {
		ast.Elements[i].Visit(u, context)
		if i < len(ast.Expressions) {
			u.expression.WriteString("${")
			ast.Expressions[i].Visit(u, context)
			u.expression.WriteString("}")
		}
	}
	u.expression.WriteString("`")
	return nil
}

// VisitTemplateLiteralElement visits a template literal element
func (u *Unparser) VisitTemplateLiteralElement(ast *expression_parser.TemplateLiteralElement, context interface{}) interface{} {
	u.expression.WriteString(ast.Text)
	return nil
}

// VisitTaggedTemplateLiteral visits a tagged template literal
func (u *Unparser) VisitTaggedTemplateLiteral(ast *expression_parser.TaggedTemplateLiteral, context interface{}) interface{} {
	ast.Tag.Visit(u, context)
	ast.Template.Visit(u, context)
	return nil
}

// VisitParenthesizedExpression visits a parenthesized expression
func (u *Unparser) VisitParenthesizedExpression(ast *expression_parser.ParenthesizedExpression, context interface{}) interface{} {
	u.expression.WriteString("(")
	ast.Expression.Visit(u, context)
	u.expression.WriteString(")")
	return nil
}

var sharedUnparser = NewUnparser()

// Unparse is a helper function that unparses an AST
func Unparse(ast expression_parser.AST) string {
	return sharedUnparser.Unparse(ast)
}

