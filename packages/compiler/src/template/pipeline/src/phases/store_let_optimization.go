package phases

import (
	"ngc-go/packages/compiler/src/output"
	"ngc-go/packages/compiler/src/template/pipeline/ir"
	"ngc-go/packages/compiler/src/template/pipeline/ir/src/expression"
	"ngc-go/packages/compiler/src/template/pipeline/ir/src/operations"
	ops_create "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/create"

	pipeline "ngc-go/packages/compiler/src/template/pipeline/src/compilation"
)

// OptimizeStoreLet removes any `storeLet` calls that aren't referenced outside of the current view.
func OptimizeStoreLet(job *pipeline.CompilationJob) {
	letUsedExternally := make(map[operations.XrefId]bool)
	declareLetOps := make(map[operations.XrefId]*ops_create.DeclareLetOp)

	// Since `@let` declarations can be referenced in child views, both in
	// the creation block (via listeners) and in the update block, we have
	// to look through all the ops to find the references.
	for _, unit := range job.GetUnits() {
		for op := unit.GetCreate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			// Take advantage that we're already looking through all the ops and track some more info.
			if op.GetKind() == ir.OpKindDeclareLet {
				if declareLetOp, ok := op.(*ops_create.DeclareLetOp); ok {
					declareLetOps[declareLetOp.GetXref()] = declareLetOp
				}
			}

			expression.VisitExpressionsInOp(op, func(expr output.OutputExpression, flags expression.VisitorContextFlag) {
				if contextLetRef, ok := expr.(*expression.ContextLetReferenceExpr); ok {
					letUsedExternally[contextLetRef.Target] = true
				}
			})
		}

		for op := unit.GetUpdate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			expression.VisitExpressionsInOp(op, func(expr output.OutputExpression, flags expression.VisitorContextFlag) {
				if contextLetRef, ok := expr.(*expression.ContextLetReferenceExpr); ok {
					letUsedExternally[contextLetRef.Target] = true
				}
			})
		}
	}

	for _, unit := range job.GetUnits() {
		for op := unit.GetUpdate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			expression.TransformExpressionsInOp(
				op,
				func(expr output.OutputExpression, flags expression.VisitorContextFlag) output.OutputExpression {
					storeLetExpr, ok := expr.(*expression.StoreLetExpr)
					if !ok {
						return expr
					}

					// If a @let isn't used in other views, we don't have to store its value.
					if !letUsedExternally[storeLetExpr.Target] {
						// Furthermore, if the @let isn't using pipes, we can also drop its declareLet op.
						// We need to keep the declareLet if there are pipes, because they can use DI which
						// requires the TNode created by declareLet.
						if !hasPipe(storeLetExpr) {
							if declareLetOp, exists := declareLetOps[storeLetExpr.Target]; exists {
								unit.GetCreate().Remove(declareLetOp)
							}
						}
						return storeLetExpr.Value
					}
					return expr
				},
				expression.VisitorContextFlagNone,
			)
		}
	}
}

// hasPipe determines if a `storeLet` expression contains a pipe.
func hasPipe(root *expression.StoreLetExpr) bool {
	result := false

	expression.TransformExpressionsInExpression(
		root,
		func(expr output.OutputExpression, flags expression.VisitorContextFlag) output.OutputExpression {
			if _, ok := expr.(*expression.PipeBindingExpr); ok {
				result = true
			} else if _, ok := expr.(*expression.PipeBindingVariadicExpr); ok {
				result = true
			}
			return expr
		},
		expression.VisitorContextFlagNone,
	)

	return result
}
