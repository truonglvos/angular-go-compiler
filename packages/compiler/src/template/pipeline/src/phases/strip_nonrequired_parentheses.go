package phases

import (
	"ngc-go/packages/compiler/src/output"
	"ngc-go/packages/compiler/src/template/pipeline/ir"
	ir_expression "ngc-go/packages/compiler/src/template/pipeline/ir/src/expression"

	pipeline "ngc-go/packages/compiler/src/template/pipeline/src/compilation"
)

// StripNonrequiredParentheses strips all parentheses except in the following situations where they are required:
//
//  1. Unary operators in the base of an exponentiation expression. For example, `-2 ** 3` is not
//     valid JavaScript, but `(-2) ** 3` is.
//
//  2. When mixing nullish coalescing (`??`) and logical and/or operators (`&&`, `||`), we need
//     parentheses. For example, `a ?? b && c` is not valid JavaScript, but `a ?? (b && c)` is.
//     Note: Because of the outcome of https://github.com/microsoft/TypeScript/issues/62307
//     We need (for now) to keep parentheses around the `??` operator when it is used with and/or operators.
//     For example, `a ?? b && c` is not valid JavaScript, but `(a ?? b) && c` is.
//
//  3. Ternary expression used as an operand for nullish coalescing. Typescript generates incorrect
//     code if the parentheses are missing. For example when `(a ? b : c) ?? d` is translated to
//     typescript AST, the parentheses node is removed, and then the remaining AST is printed, it
//     incorrectly prints `a ? b : c ?? d`. This is different from how it handles the same situation
//     with `||` and `&&` where it prints the parentheses even if they are not present in the AST.
//     Note: We may be able to remove this case if Typescript resolves the following issue:
//     https://github.com/microsoft/TypeScript/issues/61369
func StripNonrequiredParentheses(job *pipeline.CompilationJob) {
	// Check which parentheses are required.
	requiredParens := make(map[*output.ParenthesizedExpr]bool)
	for _, unit := range job.GetUnits() {
		for op := unit.GetCreate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			ir_expression.VisitExpressionsInOp(op, func(expr output.OutputExpression, flags ir_expression.VisitorContextFlag) {
				if binaryOp, ok := expr.(*output.BinaryOperatorExpr); ok {
					switch binaryOp.Operator {
					case output.BinaryOperatorExponentiation:
						checkExponentiationParens(binaryOp, requiredParens)
					case output.BinaryOperatorNullishCoalesce:
						checkNullishCoalescingParens(binaryOp, requiredParens)
					case output.BinaryOperatorAnd, output.BinaryOperatorOr:
						// these 2 cases can be dropped if the regression introduced in 5.9.2 is fixed
						// see https://github.com/microsoft/TypeScript/issues/62307
						checkAndOrParens(binaryOp, requiredParens)
					}
				}
			})
		}
		for op := unit.GetUpdate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			ir_expression.VisitExpressionsInOp(op, func(expr output.OutputExpression, flags ir_expression.VisitorContextFlag) {
				if binaryOp, ok := expr.(*output.BinaryOperatorExpr); ok {
					switch binaryOp.Operator {
					case output.BinaryOperatorExponentiation:
						checkExponentiationParens(binaryOp, requiredParens)
					case output.BinaryOperatorNullishCoalesce:
						checkNullishCoalescingParens(binaryOp, requiredParens)
					case output.BinaryOperatorAnd, output.BinaryOperatorOr:
						checkAndOrParens(binaryOp, requiredParens)
					}
				}
			})
		}
	}

	// Remove any non-required parentheses.
	for _, unit := range job.GetUnits() {
		for op := unit.GetCreate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			ir_expression.TransformExpressionsInOp(
				op,
				func(expr output.OutputExpression, flags ir_expression.VisitorContextFlag) output.OutputExpression {
					if parenExpr, ok := expr.(*output.ParenthesizedExpr); ok {
						if requiredParens[parenExpr] {
							return expr
						}
						return parenExpr.Expr
					}
					return expr
				},
				ir_expression.VisitorContextFlagNone,
			)
		}
		for op := unit.GetUpdate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			ir_expression.TransformExpressionsInOp(
				op,
				func(expr output.OutputExpression, flags ir_expression.VisitorContextFlag) output.OutputExpression {
					if parenExpr, ok := expr.(*output.ParenthesizedExpr); ok {
						if requiredParens[parenExpr] {
							return expr
						}
						return parenExpr.Expr
					}
					return expr
				},
				ir_expression.VisitorContextFlagNone,
			)
		}
	}
}

func checkExponentiationParens(
	expr *output.BinaryOperatorExpr,
	requiredParens map[*output.ParenthesizedExpr]bool,
) {
	if parenExpr, ok := expr.Lhs.(*output.ParenthesizedExpr); ok {
		if _, ok := parenExpr.Expr.(*output.UnaryOperatorExpr); ok {
			requiredParens[parenExpr] = true
		}
	}
}

func checkNullishCoalescingParens(
	expr *output.BinaryOperatorExpr,
	requiredParens map[*output.ParenthesizedExpr]bool,
) {
	if parenExpr, ok := expr.Lhs.(*output.ParenthesizedExpr); ok {
		if isLogicalAndOr(parenExpr.Expr) {
			requiredParens[parenExpr] = true
		} else if _, ok := parenExpr.Expr.(*output.ConditionalExpr); ok {
			requiredParens[parenExpr] = true
		}
	}
	if parenExpr, ok := expr.Rhs.(*output.ParenthesizedExpr); ok {
		if isLogicalAndOr(parenExpr.Expr) {
			requiredParens[parenExpr] = true
		} else if _, ok := parenExpr.Expr.(*output.ConditionalExpr); ok {
			requiredParens[parenExpr] = true
		}
	}
}

func checkAndOrParens(expr *output.BinaryOperatorExpr, requiredParens map[*output.ParenthesizedExpr]bool) {
	if parenExpr, ok := expr.Lhs.(*output.ParenthesizedExpr); ok {
		if binaryOp, ok := parenExpr.Expr.(*output.BinaryOperatorExpr); ok {
			if binaryOp.Operator == output.BinaryOperatorNullishCoalesce {
				requiredParens[parenExpr] = true
			}
		}
	}
}

func isLogicalAndOr(expr output.OutputExpression) bool {
	if binaryOp, ok := expr.(*output.BinaryOperatorExpr); ok {
		return binaryOp.Operator == output.BinaryOperatorAnd || binaryOp.Operator == output.BinaryOperatorOr
	}
	return false
}
