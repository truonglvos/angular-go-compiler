package phases

import (
	"fmt"

	"ngc-go/packages/compiler/src/output"
	"ngc-go/packages/compiler/src/template/pipeline/ir"
	ir_expression "ngc-go/packages/compiler/src/template/pipeline/ir/src/expression"
	ir_operation "ngc-go/packages/compiler/src/template/pipeline/ir/src/operations"
	ops_create "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/create"
	ops_shared "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/shared"
	ir_variable "ngc-go/packages/compiler/src/template/pipeline/ir/src/variable"

	pipeline_compilation "ngc-go/packages/compiler/src/template/pipeline/src/compilation"
)

// ResolveContexts resolves `ir.ContextExpr` expressions (which represent embedded view or component contexts) to
// either the `ctx` parameter to component functions (for the current view context) or to variables
// that store those contexts (for contexts accessed via the `nextContext()` instruction).
func ResolveContexts(job *pipeline_compilation.CompilationJob) {
	for _, unit := range job.GetUnits() {
		processLexicalScope(unit, unit.GetCreate())
		processLexicalScope(unit, unit.GetUpdate())
	}
}

func processLexicalScope(
	view pipeline_compilation.CompilationUnit,
	opsList *ir_operation.OpList,
) {
	// Track the expressions used to access all available contexts within the current view, by the
	// view `ir.XrefId`.
	scope := make(map[ir_operation.XrefId]output.OutputExpression)

	// The current view's context is accessible via the `ctx` parameter.
	scope[view.GetXref()] = output.NewReadVarExpr("ctx", nil, nil)

	for op := opsList.Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
		switch op.GetKind() {
		case ir.OpKindVariable:
			if varOp, ok := op.(*ops_shared.VariableOp); ok {
				if contextVar, ok := varOp.Variable.(*ir_variable.ContextVariable); ok {
					scope[contextVar.View] = ir_expression.NewReadVariableExpr(varOp.Xref)
				}
			}
		case ir.OpKindAnimation, ir.OpKindAnimationListener, ir.OpKindListener, ir.OpKindTwoWayListener:
			var handlerOps *ir_operation.OpList
			if animOp, ok := op.(*ops_create.AnimationOp); ok {
				handlerOps = animOp.HandlerOps
			} else if animListenerOp, ok := op.(*ops_create.AnimationListenerOp); ok {
				handlerOps = animListenerOp.HandlerOps
			} else if listenerOp, ok := op.(*ops_create.ListenerOp); ok {
				handlerOps = listenerOp.HandlerOps
			} else if twoWayOp, ok := op.(*ops_create.TwoWayListenerOp); ok {
				handlerOps = twoWayOp.HandlerOps
			}
			if handlerOps != nil {
				processLexicalScope(view, handlerOps)
			}
		case ir.OpKindRepeaterCreate:
			if repeaterOp, ok := op.(*ops_create.RepeaterCreateOp); ok {
				if repeaterOp.TrackByOps != nil {
					processLexicalScope(view, repeaterOp.TrackByOps)
				}
			}
		}
	}

	// Check if this is the root view by casting view to ViewCompilationUnit and accessing Job
	viewUnit, ok := view.(*pipeline_compilation.ViewCompilationUnit)
	if ok && viewUnit.Job != nil && view == viewUnit.Job.Root {
		// Prefer `ctx` of the root view to any variables which happen to contain the root context.
		scope[view.GetXref()] = output.NewReadVarExpr("ctx", nil, nil)
	}

	for op := opsList.Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
		ir_expression.TransformExpressionsInOp(
			op,
			func(expr output.OutputExpression, flags ir_expression.VisitorContextFlag) output.OutputExpression {
				if contextExpr, ok := expr.(*ir_expression.ContextExpr); ok {
					if ctxExpr, exists := scope[contextExpr.View]; exists {
						return ctxExpr
					}
					panic(fmt.Sprintf("No context found for reference to view %d from view %d", contextExpr.View, view.GetXref()))
				}
				return expr
			},
			ir_expression.VisitorContextFlagNone,
		)
	}
}
