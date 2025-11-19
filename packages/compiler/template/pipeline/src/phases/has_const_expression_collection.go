package phases

import (
	"ngc-go/packages/compiler/output"
	ir "ngc-go/packages/compiler/template/pipeline/ir/src"
	ir_expression "ngc-go/packages/compiler/template/pipeline/ir/src/expression"

	pipeline "ngc-go/packages/compiler/template/pipeline/src/compilation"
)

// CollectConstExpressions collects `ir.ConstCollectedExpr` expressions and lifts them into the component const array.
// `ir.ConstCollectedExpr` may be present in any IR expression. This means that expression needs to
// be lifted into the component const array, and replaced with a reference to the const array at its
// usage site. This phase walks the IR and performs this transformation.
func CollectConstExpressions(job *pipeline.ComponentCompilationJob) {
	for _, unit := range job.GetUnits() {
		for op := unit.GetCreate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			ir_expression.TransformExpressionsInOp(
				op,
				func(expr output.OutputExpression, flags ir_expression.VisitorContextFlag) output.OutputExpression {
					constCollected, ok := expr.(*ir_expression.ConstCollectedExpr)
					if !ok {
						return expr
					}
					constIndex := job.AddConst(constCollected.Expr, nil)
					return output.NewLiteralExpr(int(constIndex), nil, nil)
				},
				ir_expression.VisitorContextFlagNone,
			)
		}
		for op := unit.GetUpdate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			ir_expression.TransformExpressionsInOp(
				op,
				func(expr output.OutputExpression, flags ir_expression.VisitorContextFlag) output.OutputExpression {
					constCollected, ok := expr.(*ir_expression.ConstCollectedExpr)
					if !ok {
						return expr
					}
					constIndex := job.AddConst(constCollected.Expr, nil)
					return output.NewLiteralExpr(int(constIndex), nil, nil)
				},
				ir_expression.VisitorContextFlagNone,
			)
		}
	}
}
