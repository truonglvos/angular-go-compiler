package expression_parser

import (
	"fmt"
	"strings"
)

// Serialize serializes the given AST into a normalized string format
func Serialize(expression *ASTWithSource) string {
	visitor := NewSerializeExpressionVisitor()
	return expression.Visit(visitor, nil).(string)
}

// SerializeExpressionVisitor is a visitor that serializes AST to string
type SerializeExpressionVisitor struct{}

// NewSerializeExpressionVisitor creates a new SerializeExpressionVisitor
func NewSerializeExpressionVisitor() *SerializeExpressionVisitor {
	return &SerializeExpressionVisitor{}
}

// VisitUnary visits a unary expression
func (s *SerializeExpressionVisitor) VisitUnary(ast *Unary, context interface{}) interface{} {
	return fmt.Sprintf("%s%s", ast.Operator, ast.Expr.Visit(s, context).(string))
}

// VisitBinary visits a binary expression
func (s *SerializeExpressionVisitor) VisitBinary(ast *Binary, context interface{}) interface{} {
	return fmt.Sprintf("%s %s %s",
		ast.Left.Visit(s, context).(string),
		ast.Operation,
		ast.Right.Visit(s, context).(string))
}

// VisitChain visits a chain expression
func (s *SerializeExpressionVisitor) VisitChain(ast *Chain, context interface{}) interface{} {
	parts := make([]string, len(ast.Expressions))
	for i, expr := range ast.Expressions {
		parts[i] = expr.Visit(s, context).(string)
	}
	return strings.Join(parts, "; ")
}

// VisitConditional visits a conditional expression
func (s *SerializeExpressionVisitor) VisitConditional(ast *Conditional, context interface{}) interface{} {
	return fmt.Sprintf("%s ? %s : %s",
		ast.Condition.Visit(s, context).(string),
		ast.TrueExp.Visit(s, context).(string),
		ast.FalseExp.Visit(s, context).(string))
}

// VisitThisReceiver visits a this receiver
func (s *SerializeExpressionVisitor) VisitThisReceiver(ast *ThisReceiver, context interface{}) interface{} {
	return "this"
}

// VisitImplicitReceiver visits an implicit receiver
func (s *SerializeExpressionVisitor) VisitImplicitReceiver(ast *ImplicitReceiver, context interface{}) interface{} {
	return ""
}

// VisitInterpolation visits an interpolation
func (s *SerializeExpressionVisitor) VisitInterpolation(ast *Interpolation, context interface{}) interface{} {
	parts := interleave(ast.Strings, ast.Expressions, s, context)
	return strings.Join(parts, "")
}

// VisitKeyedRead visits a keyed read
func (s *SerializeExpressionVisitor) VisitKeyedRead(ast *KeyedRead, context interface{}) interface{} {
	return fmt.Sprintf("%s[%s]",
		ast.Receiver.Visit(s, context).(string),
		ast.Key.Visit(s, context).(string))
}

// VisitLiteralArray visits a literal array
func (s *SerializeExpressionVisitor) VisitLiteralArray(ast *LiteralArray, context interface{}) interface{} {
	parts := make([]string, len(ast.Expressions))
	for i, expr := range ast.Expressions {
		parts[i] = expr.Visit(s, context).(string)
	}
	return fmt.Sprintf("[%s]", strings.Join(parts, ", "))
}

// VisitLiteralMap visits a literal map
func (s *SerializeExpressionVisitor) VisitLiteralMap(ast *LiteralMap, context interface{}) interface{} {
	keys := make([]string, len(ast.Keys))
	for i, key := range ast.Keys {
		if key.Quoted {
			keys[i] = fmt.Sprintf("'%s'", key.Key)
		} else {
			keys[i] = key.Key
		}
	}
	values := make([]string, len(ast.Values))
	for i, value := range ast.Values {
		values[i] = value.Visit(s, context).(string)
	}
	pairs := zip(keys, values)
	pairStrs := make([]string, len(pairs))
	for i, pair := range pairs {
		pairStrs[i] = fmt.Sprintf("%s: %s", pair[0], pair[1])
	}
	return fmt.Sprintf("{%s}", strings.Join(pairStrs, ", "))
}

// VisitLiteralPrimitive visits a literal primitive
func (s *SerializeExpressionVisitor) VisitLiteralPrimitive(ast *LiteralPrimitive, context interface{}) interface{} {
	if ast.Value == nil {
		return "null"
	}

	switch v := ast.Value.(type) {
	case float64:
		return fmt.Sprintf("%g", v)
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case bool:
		return fmt.Sprintf("%t", v)
	case string:
		return fmt.Sprintf("'%s'", strings.ReplaceAll(v, "'", "\\'"))
	default:
		// Check for undefined
		if _, ok := ast.Value.(interface{ IsUndefined() bool }); ok {
			return "undefined"
		}
		panic(fmt.Sprintf("Unsupported primitive type: %T", ast.Value))
	}
}

// VisitPipe visits a pipe expression
func (s *SerializeExpressionVisitor) VisitPipe(ast *BindingPipe, context interface{}) interface{} {
	return fmt.Sprintf("%s | %s",
		ast.Exp.Visit(s, context).(string),
		ast.Name)
}

// VisitPrefixNot visits a prefix not
func (s *SerializeExpressionVisitor) VisitPrefixNot(ast *PrefixNot, context interface{}) interface{} {
	return fmt.Sprintf("!%s", ast.Expression.Visit(s, context).(string))
}

// VisitNonNullAssert visits a non-null assertion
func (s *SerializeExpressionVisitor) VisitNonNullAssert(ast *NonNullAssert, context interface{}) interface{} {
	return fmt.Sprintf("%s!", ast.Expression.Visit(s, context).(string))
}

// VisitPropertyRead visits a property read
func (s *SerializeExpressionVisitor) VisitPropertyRead(ast *PropertyRead, context interface{}) interface{} {
	if _, ok := ast.Receiver.(*ImplicitReceiver); ok {
		return ast.Name
	}
	return fmt.Sprintf("%s.%s",
		ast.Receiver.Visit(s, context).(string),
		ast.Name)
}

// VisitSafePropertyRead visits a safe property read
func (s *SerializeExpressionVisitor) VisitSafePropertyRead(ast *SafePropertyRead, context interface{}) interface{} {
	return fmt.Sprintf("%s?.%s",
		ast.Receiver.Visit(s, context).(string),
		ast.Name)
}

// VisitSafeKeyedRead visits a safe keyed read
func (s *SerializeExpressionVisitor) VisitSafeKeyedRead(ast *SafeKeyedRead, context interface{}) interface{} {
	return fmt.Sprintf("%s?.[%s]",
		ast.Receiver.Visit(s, context).(string),
		ast.Key.Visit(s, context).(string))
}

// VisitCall visits a call
func (s *SerializeExpressionVisitor) VisitCall(ast *Call, context interface{}) interface{} {
	args := make([]string, len(ast.Args))
	for i, arg := range ast.Args {
		args[i] = arg.Visit(s, context).(string)
	}
	return fmt.Sprintf("%s(%s)",
		ast.Receiver.Visit(s, context).(string),
		strings.Join(args, ", "))
}

// VisitSafeCall visits a safe call
func (s *SerializeExpressionVisitor) VisitSafeCall(ast *SafeCall, context interface{}) interface{} {
	args := make([]string, len(ast.Args))
	for i, arg := range ast.Args {
		args[i] = arg.Visit(s, context).(string)
	}
	return fmt.Sprintf("%s?(%s)",
		ast.Receiver.Visit(s, context).(string),
		strings.Join(args, ", "))
}

// VisitTypeofExpression visits a typeof expression
func (s *SerializeExpressionVisitor) VisitTypeofExpression(ast *TypeofExpression, context interface{}) interface{} {
	return fmt.Sprintf("typeof %s", ast.Expression.Visit(s, context).(string))
}

// VisitVoidExpression visits a void expression
func (s *SerializeExpressionVisitor) VisitVoidExpression(ast *VoidExpression, context interface{}) interface{} {
	return fmt.Sprintf("void %s", ast.Expression.Visit(s, context).(string))
}

// VisitRegularExpressionLiteral visits a regular expression literal
func (s *SerializeExpressionVisitor) VisitRegularExpressionLiteral(ast *RegularExpressionLiteral, context interface{}) interface{} {
	flags := ""
	if ast.Flags != nil {
		flags = *ast.Flags
	}
	return fmt.Sprintf("/%s/%s", ast.Body, flags)
}

// VisitASTWithSource visits an AST with source
func (s *SerializeExpressionVisitor) VisitASTWithSource(ast *ASTWithSource, context interface{}) interface{} {
	return ast.AST.Visit(s, context).(string)
}

// VisitTemplateLiteral visits a template literal
func (s *SerializeExpressionVisitor) VisitTemplateLiteral(ast *TemplateLiteral, context interface{}) interface{} {
	result := ""
	for i := 0; i < len(ast.Elements); i++ {
		result += ast.Elements[i].Visit(s, context).(string)
		if i < len(ast.Expressions) {
			result += fmt.Sprintf("${%s}", ast.Expressions[i].Visit(s, context).(string))
		}
	}
	return fmt.Sprintf("`%s`", result)
}

// VisitTemplateLiteralElement visits a template literal element
func (s *SerializeExpressionVisitor) VisitTemplateLiteralElement(ast *TemplateLiteralElement, context interface{}) interface{} {
	return ast.Text
}

// VisitTaggedTemplateLiteral visits a tagged template literal
func (s *SerializeExpressionVisitor) VisitTaggedTemplateLiteral(ast *TaggedTemplateLiteral, context interface{}) interface{} {
	return fmt.Sprintf("%s%s",
		ast.Tag.Visit(s, context).(string),
		ast.Template.Visit(s, context).(string))
}

// VisitParenthesizedExpression visits a parenthesized expression
func (s *SerializeExpressionVisitor) VisitParenthesizedExpression(ast *ParenthesizedExpression, context interface{}) interface{} {
	return fmt.Sprintf("(%s)", ast.Expression.Visit(s, context).(string))
}

// Visit is the default visit method
func (s *SerializeExpressionVisitor) Visit(ast AST, context interface{}) interface{} {
	return ast.Visit(s, context)
}

// zip zips the two input arrays into a single array of pairs of elements at the same index
func zip(left, right []string) [][]string {
	if len(left) != len(right) {
		panic("Array lengths must match")
	}
	result := make([][]string, len(left))
	for i := range left {
		result[i] = []string{left[i], right[i]}
	}
	return result
}

// interleave interleaves the two arrays, starting with the first item on the left, then the first item
// on the right, second item from the left, and so on
func interleave(left []string, right []AST, visitor *SerializeExpressionVisitor, context interface{}) []string {
	maxLen := len(left)
	if len(right) > maxLen {
		maxLen = len(right)
	}
	result := make([]string, 0, maxLen*2)
	for i := 0; i < maxLen; i++ {
		if i < len(left) {
			result = append(result, left[i])
		}
		if i < len(right) {
			result = append(result, right[i].Visit(visitor, context).(string))
		}
	}
	return result
}
