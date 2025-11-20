package phases

import (
	"ngc-go/packages/compiler/src/output"
	"ngc-go/packages/compiler/src/template/pipeline/ir"
	"ngc-go/packages/compiler/src/template/pipeline/ir/src/expression"

	pipeline "ngc-go/packages/compiler/src/template/pipeline/src/compilation"
)

// CreateVariadicPipes converts pipes with more than 4 arguments to variadic pipe expressions.
// Pipes that accept more than 4 arguments are variadic, and are handled with a different runtime
// instruction.
func CreateVariadicPipes(job *pipeline.CompilationJob) {
	for _, unit := range job.GetUnits() {
		for op := unit.GetUpdate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			expression.TransformExpressionsInOp(op, func(expr output.OutputExpression, flags expression.VisitorContextFlag) output.OutputExpression {
				pipeBinding, ok := expr.(*expression.PipeBindingExpr)
				if !ok {
					return expr
				}

				// Pipes are variadic if they have more than 4 arguments.
				if len(pipeBinding.Args) <= 4 {
					return expr
				}

				// Convert to variadic pipe binding
				argsArray := make([]output.OutputExpression, len(pipeBinding.Args))
				for i, arg := range pipeBinding.Args {
					argsArray[i] = arg
				}
				argsLiteral := output.NewLiteralArrayExpr(argsArray, nil, nil)
				return expression.NewPipeBindingVariadicExpr(
					pipeBinding.Target,
					pipeBinding.TargetSlot,
					pipeBinding.Name,
					argsLiteral,
					len(pipeBinding.Args),
				)
			}, expression.VisitorContextFlagNone)
		}
	}
}
