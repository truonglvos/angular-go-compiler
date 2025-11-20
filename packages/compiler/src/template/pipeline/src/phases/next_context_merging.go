package phases

import (
	"ngc-go/packages/compiler/src/output"
	"ngc-go/packages/compiler/src/template/pipeline/ir"
	"ngc-go/packages/compiler/src/template/pipeline/ir/src/expression"
	"ngc-go/packages/compiler/src/template/pipeline/ir/src/operations"
	ops_create "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/create"
	ops_shared "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/shared"

	pipeline "ngc-go/packages/compiler/src/template/pipeline/src/compilation"
)

// MergeNextContextExpressions merges logically sequential `NextContextExpr` operations.
//
// `NextContextExpr` can be referenced repeatedly, "popping" the runtime's context stack each time.
// When two such expressions appear back-to-back, it's possible to merge them together into a single
// `NextContextExpr` that steps multiple contexts. This merging is possible if all conditions are met:
//
//   - The result of the `NextContextExpr` that's folded into the subsequent one is not stored (that
//     is, the call is purely side-effectful).
//   - No operations in between them uses the implicit context.
func MergeNextContextExpressions(job *pipeline.CompilationJob) {
	for _, unit := range job.GetUnits() {
		for op := unit.GetCreate().Head(); op != nil; op = op.Next() {
			kind := op.GetKind()
			if kind == ir.OpKindListener || kind == ir.OpKindAnimation ||
				kind == ir.OpKindAnimationListener || kind == ir.OpKindTwoWayListener {
				if listenerOp, ok := op.(*ops_create.ListenerOp); ok {
					mergeNextContextsInOps(listenerOp.HandlerOps)
				} else if animOp, ok := op.(*ops_create.AnimationOp); ok {
					mergeNextContextsInOps(animOp.HandlerOps)
				} else if animListenerOp, ok := op.(*ops_create.AnimationListenerOp); ok {
					mergeNextContextsInOps(animListenerOp.HandlerOps)
				} else if twoWayOp, ok := op.(*ops_create.TwoWayListenerOp); ok {
					mergeNextContextsInOps(twoWayOp.HandlerOps)
				}
			}
		}
		mergeNextContextsInOps(unit.GetUpdate())
	}
}

func mergeNextContextsInOps(opsList *operations.OpList) {
	for op := opsList.Head(); op != nil; op = op.Next() {
		// Look for a candidate operations to maybe merge.
		if op.GetKind() != ir.OpKindStatement {
			continue
		}

		stmtOp, ok := op.(*ops_shared.StatementOp)
		if !ok {
			continue
		}

		exprStmt, ok := stmtOp.Statement.(*output.ExpressionStatement)
		if !ok {
			continue
		}

		nextCtxExpr, ok := exprStmt.Expr.(*expression.NextContextExpr)
		if !ok {
			continue
		}

		mergeSteps := nextCtxExpr.Steps

		// Try to merge this `expression.NextContextExpr`.
		tryToMerge := true
		for candidate := op.Next(); candidate != nil && candidate.GetKind() != ir.OpKindListEnd && tryToMerge; candidate = candidate.Next() {
			expression.VisitExpressionsInOp(
				candidate,
				func(expr output.OutputExpression, flags expression.VisitorContextFlag) {
					if !expression.IsIrExpression(expr) {
						return
					}

					if !tryToMerge {
						// Either we've already merged, or failed to merge.
						return
					}

					if flags&expression.VisitorContextFlagInChildOperation != 0 {
						// We cannot merge into child operations.
						return
					}

					if nextCtx, ok := expr.(*expression.NextContextExpr); ok {
						// Merge the previous `ir.NextContextExpr` into this one.
						nextCtx.Steps += mergeSteps
						opsList.Remove(op)
						tryToMerge = false
					} else if getCurrentView, ok := expr.(*expression.GetCurrentViewExpr); ok {
						// Can't merge past a dependency on the context.
						tryToMerge = false
						_ = getCurrentView
					} else if ref, ok := expr.(*expression.ReferenceExpr); ok {
						// Can't merge past a dependency on the context.
						tryToMerge = false
						_ = ref
					} else if ctxLetRef, ok := expr.(*expression.ContextLetReferenceExpr); ok {
						// Can't merge past a dependency on the context.
						tryToMerge = false
						_ = ctxLetRef
					}
				},
			)
		}
	}
}
