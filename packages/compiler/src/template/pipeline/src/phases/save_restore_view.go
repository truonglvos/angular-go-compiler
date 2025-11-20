package phases

import (
	"ngc-go/packages/compiler/src/output"
	"ngc-go/packages/compiler/src/template/pipeline/ir"
	"ngc-go/packages/compiler/src/template/pipeline/ir/src/expression"
	"ngc-go/packages/compiler/src/template/pipeline/ir/src/operations"
	ops_create "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/create"
	"ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/shared"
	ir_variable "ngc-go/packages/compiler/src/template/pipeline/ir/src/variable"

	pipeline "ngc-go/packages/compiler/src/template/pipeline/src/compilation"
)

// SaveAndRestoreView eagerly generates all save view variables; they will be optimized away later.
// When inside of a listener, we may need access to one or more enclosing views. Therefore, each
// view should save the current view, and each listener must have the ability to restore the
// appropriate view.
func SaveAndRestoreView(job *pipeline.ComponentCompilationJob) {
	for _, unit := range job.GetUnits() {
		viewUnit, ok := unit.(*pipeline.ViewCompilationUnit)
		if !ok {
			continue
		}

		// Prepend a variable to save the current view
		savedViewVar := ir_variable.NewSavedViewVariable(viewUnit.Xref)
		savedViewVarOp := shared.NewVariableOp(
			job.AllocateXrefId(),
			savedViewVar,
			expression.NewGetCurrentViewExpr(),
			ir.VariableFlagsNone,
		)
		viewUnit.Create.InsertBefore(savedViewVarOp, viewUnit.Create.Head())

		// Check each listener to see if it needs restore view
		for op := viewUnit.Create.Head(); op != nil; op = op.Next() {
			kind := op.GetKind()
			if kind != ir.OpKindListener && kind != ir.OpKindTwoWayListener &&
				kind != ir.OpKindAnimation && kind != ir.OpKindAnimationListener {
				continue
			}

			// Embedded views always need the save/restore view operations.
			needsRestoreView := viewUnit != job.Root

			if !needsRestoreView {
				// Check if listener references local refs
				var handlerOps *operations.OpList
				if listenerOp, ok := op.(*ops_create.ListenerOp); ok {
					handlerOps = listenerOp.HandlerOps
				} else if twoWayOp, ok := op.(*ops_create.TwoWayListenerOp); ok {
					handlerOps = twoWayOp.HandlerOps
				} else if animOp, ok := op.(*ops_create.AnimationOp); ok {
					handlerOps = animOp.HandlerOps
				} else if animListenerOp, ok := op.(*ops_create.AnimationListenerOp); ok {
					handlerOps = animListenerOp.HandlerOps
				}

				if handlerOps != nil {
					expression.VisitExpressionsInOp(op, func(expr output.OutputExpression, flags expression.VisitorContextFlag) {
						if _, ok := expr.(*expression.ReferenceExpr); ok {
							needsRestoreView = true
						} else if _, ok := expr.(*expression.ContextLetReferenceExpr); ok {
							needsRestoreView = true
						}
					})
				}
			}

			if needsRestoreView {
				// Type assertion: op is already checked to be a listener/animation op, so it implements CreateOp
				if createOp, ok := op.(operations.CreateOp); ok {
					addSaveRestoreViewOperationToListener(viewUnit, createOp)
				}
			}
		}
	}
}

func addSaveRestoreViewOperationToListener(
	unit *pipeline.ViewCompilationUnit,
	op operations.CreateOp,
) {
	var handlerOps *operations.OpList
	if listenerOp, ok := op.(*ops_create.ListenerOp); ok {
		handlerOps = listenerOp.HandlerOps
	} else if twoWayOp, ok := op.(*ops_create.TwoWayListenerOp); ok {
		handlerOps = twoWayOp.HandlerOps
	} else if animOp, ok := op.(*ops_create.AnimationOp); ok {
		handlerOps = animOp.HandlerOps
	} else if animListenerOp, ok := op.(*ops_create.AnimationListenerOp); ok {
		handlerOps = animListenerOp.HandlerOps
	}

	if handlerOps == nil {
		return
	}

	// Prepend a variable to restore the view
	contextVar := ir_variable.NewContextVariable(unit.Xref)
	restoreViewVarOp := shared.NewVariableOp(
		unit.Job.AllocateXrefId(),
		contextVar,
		expression.NewRestoreViewExpr(unit.Xref),
		ir.VariableFlagsNone,
	)
	handlerOps.InsertBefore(restoreViewVarOp, handlerOps.Head())

	// The "restore view" operations in listeners requires a call to `resetView` to reset the
	// context prior to returning from the listener operations. Find any `return` statements in
	// the listener body and wrap them in a call to reset the view.
	for handlerOp := handlerOps.Head(); handlerOp != nil; handlerOp = handlerOp.Next() {
		if handlerOp.GetKind() == ir.OpKindStatement {
			if stmtOp, ok := handlerOp.(*shared.StatementOp); ok {
				if returnStmt, ok := stmtOp.Statement.(*output.ReturnStatement); ok {
					returnStmt.Value = expression.NewResetViewExpr(returnStmt.Value)
				}
			}
		}
	}
}
