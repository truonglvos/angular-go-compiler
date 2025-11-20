package phases

import (
	"ngc-go/packages/compiler/src/output"
	"ngc-go/packages/compiler/src/template/pipeline/ir"
	"ngc-go/packages/compiler/src/template/pipeline/ir/src/expression"
	"ngc-go/packages/compiler/src/template/pipeline/ir/src/operations"
	ops_create "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/create"

	pipeline "ngc-go/packages/compiler/src/template/pipeline/src/compilation"
)

// ResolveDollarEvent transforms any variable inside a listener with the name `$event` into a output
// lexical read immediately, and does not participate in any of the normal logic for handling variables.
func ResolveDollarEvent(job *pipeline.CompilationJob) {
	for _, unit := range job.GetUnits() {
		transformDollarEvent(unit.GetCreate())
		transformDollarEvent(unit.GetUpdate())
	}
}

func transformDollarEvent(opsList *operations.OpList) {
	for op := opsList.Head(); op != nil; op = op.Next() {
		kind := op.GetKind()
		if kind == ir.OpKindListener || kind == ir.OpKindTwoWayListener || kind == ir.OpKindAnimationListener {
			expression.TransformExpressionsInOp(
				op,
				func(expr output.OutputExpression, flags expression.VisitorContextFlag) output.OutputExpression {
					if lexicalRead, ok := expr.(*expression.LexicalReadExpr); ok && lexicalRead.Name == "$event" {
						// Two-way listeners always consume `$event` so they omit this field.
						if listenerOp, ok := op.(*ops_create.ListenerOp); ok && (kind == ir.OpKindListener || kind == ir.OpKindAnimationListener) {
							listenerOp.ConsumesDollarEvent = true
						} else if animListenerOp, ok := op.(*ops_create.AnimationListenerOp); ok {
							animListenerOp.ConsumesDollarEvent = true
						}
						return output.NewReadVarExpr(lexicalRead.Name, nil, nil)
					}
					return expr
				},
				expression.VisitorContextFlagInChildOperation,
			)
		}
	}
}
