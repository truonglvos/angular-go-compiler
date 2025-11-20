package phases

import (
	"strings"

	"ngc-go/packages/compiler/src/constant"
	"ngc-go/packages/compiler/src/output"
	ir "ngc-go/packages/compiler/src/template/pipeline/ir/src"
	ir_expression "ngc-go/packages/compiler/src/template/pipeline/ir/src/expression"

	pipeline "ngc-go/packages/compiler/src/template/pipeline/src/compilation"
)

// OptimizeRegularExpressions optimizes regular expressions used in expressions.
func OptimizeRegularExpressions(job *pipeline.CompilationJob) {
	for _, unit := range job.GetUnits() {
		for op := unit.GetCreate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			ir_expression.TransformExpressionsInOp(
				op,
				func(expr output.OutputExpression, flags ir_expression.VisitorContextFlag) output.OutputExpression {
					regexExpr, ok := expr.(*output.RegularExpressionLiteralExpr)
					if !ok {
						return expr
					}
					// We can't optimize global regexes, because they're stateful.
					if regexExpr.Flags != nil && strings.Contains(*regexExpr.Flags, "g") {
						return expr
					}
					return job.Pool.GetSharedConstant(&RegularExpressionConstant{}, regexExpr)
				},
				ir_expression.VisitorContextFlagNone,
			)
		}
		for op := unit.GetUpdate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			ir_expression.TransformExpressionsInOp(
				op,
				func(expr output.OutputExpression, flags ir_expression.VisitorContextFlag) output.OutputExpression {
					regexExpr, ok := expr.(*output.RegularExpressionLiteralExpr)
					if !ok {
						return expr
					}
					// We can't optimize global regexes, because they're stateful.
					if regexExpr.Flags != nil && strings.Contains(*regexExpr.Flags, "g") {
						return expr
					}
					return job.Pool.GetSharedConstant(&RegularExpressionConstant{}, regexExpr)
				},
				ir_expression.VisitorContextFlagNone,
			)
		}
	}
}

// RegularExpressionConstant is a shared constant definition for regular expressions
type RegularExpressionConstant struct {
	constant.GenericKeyFn
}

// KeyOf generates a key for a regular expression
func (r *RegularExpressionConstant) KeyOf(expr output.OutputExpression) string {
	return r.GenericKeyFn.KeyOf(expr)
}

// ToSharedConstantDeclaration creates a declaration statement for the shared constant
func (r *RegularExpressionConstant) ToSharedConstantDeclaration(declName string, keyExpr output.OutputExpression) output.OutputStatement {
	return output.NewDeclareVarStmt(declName, keyExpr, nil, output.StmtModifierFinal, nil, nil)
}
