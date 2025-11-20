package phases

import (
	"fmt"

	"ngc-go/packages/compiler/src/output"
	"ngc-go/packages/compiler/src/template/pipeline/ir"
	"ngc-go/packages/compiler/src/template/pipeline/ir/src/expression"
	ir_operations "ngc-go/packages/compiler/src/template/pipeline/ir/src/operations"
	ops_create "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/create"
	ops_shared "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/shared"
	ir_variable "ngc-go/packages/compiler/src/template/pipeline/ir/src/variable"

	pipeline_compilation "ngc-go/packages/compiler/src/template/pipeline/src/compilation"
)

// SavedView represents information about a SavedView variable
type SavedView struct {
	View     ir_operations.XrefId
	Variable ir_operations.XrefId
}

// ResolveNames resolves lexical references in views (`ir.LexicalReadExpr`) to either a target variable or to
// property reads on the top-level component context.
//
// Also matches `ir.RestoreViewExpr` expressions with the variables of their corresponding saved
// views.
func ResolveNames(job *pipeline_compilation.CompilationJob) {
	for _, unit := range job.GetUnits() {
		processLexicalScopeResolveName(unit, unit.GetCreate(), nil)
		processLexicalScopeResolveName(unit, unit.GetUpdate(), nil)
	}
}

func processLexicalScopeResolveName(
	unit pipeline_compilation.CompilationUnit,
	opsList *ir_operations.OpList,
	savedView *SavedView,
) {
	// Maps names defined in the lexical scope of this template to the `ir.XrefId`s of the variable
	// declarations which represent those values.
	//
	// Since variables are generated in each view for the entire lexical scope (including any
	// identifiers from parent templates) only local variables need be considered here.
	scope := make(map[string]ir_operations.XrefId)

	// Symbols defined within the current scope. They take precedence over ones defined outside.
	localDefinitions := make(map[string]ir_operations.XrefId)

	// First, step through the operations list and:
	// 1) build up the `scope` mapping
	// 2) recurse into any listener functions
	for op := opsList.Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
		switch op.GetKind() {
		case ir.OpKindVariable:
			if varOp, ok := op.(*ops_shared.VariableOp); ok {
				processVariableForResolveNames(varOp, scope, localDefinitions, &savedView)
			}
		case ir.OpKindAnimation, ir.OpKindAnimationListener, ir.OpKindListener, ir.OpKindTwoWayListener:
			var handlerOps *ir_operations.OpList
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
				// Listener functions have separate variable declarations, so process them as a separate
				// lexical scope.
				processLexicalScopeResolveName(unit, handlerOps, savedView)
			}
		case ir.OpKindRepeaterCreate:
			if repeaterOp, ok := op.(*ops_create.RepeaterCreateOp); ok {
				if repeaterOp.TrackByOps != nil {
					processLexicalScopeResolveName(unit, repeaterOp.TrackByOps, savedView)
				}
			}
		}
	}

	// Next, use the `scope` mapping to match `ir.LexicalReadExpr` with defined names in the lexical
	// scope. Also, look for `ir.RestoreViewExpr`s and match them with the snapshotted view context
	// variable.
	for op := opsList.Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
		kind := op.GetKind()
		if kind == ir.OpKindListener || kind == ir.OpKindTwoWayListener ||
			kind == ir.OpKindAnimation || kind == ir.OpKindAnimationListener {
			// Listeners were already processed above with their own scopes.
			continue
		}

		expression.TransformExpressionsInOp(
			op,
			func(expr output.OutputExpression, flags expression.VisitorContextFlag) output.OutputExpression {
				if lexicalRead, ok := expr.(*expression.LexicalReadExpr); ok {
					// `expr` is a read of a name within the lexical scope of this view.
					// Either that name is defined within the current view, or it represents a property from the
					// main component context.
					if xref, exists := localDefinitions[lexicalRead.Name]; exists {
						return expression.NewReadVariableExpr(xref)
					} else if xref, exists := scope[lexicalRead.Name]; exists {
						// This was a defined variable in the current scope.
						return expression.NewReadVariableExpr(xref)
					} else {
						// Reading from the component context.
						// Get ComponentCompilationJob by casting unit to ViewCompilationUnit
						viewUnit, ok := unit.(*pipeline_compilation.ViewCompilationUnit)
						if !ok || viewUnit.Job == nil {
							panic("Expected ViewCompilationUnit with ComponentCompilationJob")
						}
						componentJob := viewUnit.Job
						rootXref := componentJob.Root.Xref
						return output.NewReadPropExpr(expression.NewContextExpr(rootXref), lexicalRead.Name, nil, nil)
					}
				} else if restoreView, ok := expr.(*expression.RestoreViewExpr); ok {
					// `ir.RestoreViewExpr` happens in listener functions and restores a saved view from the
					// parent creation list. We expect to find that we captured the `savedView` previously, and
					// that it matches the expected view to be restored.
					if viewXref, ok := restoreView.View.(ir_operations.XrefId); ok {
						if savedView == nil || savedView.View != viewXref {
							panic(fmt.Sprintf("AssertionError: no saved view %d from view %d", viewXref, unit.GetXref()))
						}
						restoreView.View = expression.NewReadVariableExpr(savedView.Variable)
						return restoreView
					}
				}
				return expr
			},
			expression.VisitorContextFlagNone,
		)
	}

	// Verify no lexical reads remain
	for op := opsList.Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
		expression.VisitExpressionsInOp(op, func(expr output.OutputExpression, flags expression.VisitorContextFlag) {
			if lexicalRead, ok := expr.(*expression.LexicalReadExpr); ok {
				panic(fmt.Sprintf("AssertionError: no lexical reads should remain, but found read of %s", lexicalRead.Name))
			}
		})
	}
}

func processVariableForResolveNames(
	varOp *ops_shared.VariableOp,
	scope map[string]ir_operations.XrefId,
	localDefinitions map[string]ir_operations.XrefId,
	savedView **SavedView,
) {
	variable := varOp.Variable
	if identifierVar, ok := variable.(*ir_variable.IdentifierVariable); ok {
		if identifierVar.Local {
			if _, exists := localDefinitions[identifierVar.Identifier]; exists {
				return
			}
			localDefinitions[identifierVar.Identifier] = varOp.Xref
		} else if _, exists := scope[identifierVar.Identifier]; exists {
			return
		}
		scope[identifierVar.Identifier] = varOp.Xref
	} else if aliasVar, ok := variable.(*ir_variable.AliasVariable); ok {
		// This variable represents some kind of identifier which can be used in the template.
		if _, exists := scope[aliasVar.Identifier]; exists {
			return
		}
		scope[aliasVar.Identifier] = varOp.Xref
	} else if savedViewVar, ok := variable.(*ir_variable.SavedViewVariable); ok {
		// This variable represents a snapshot of the current view context, and can be used to
		// restore that context within listener functions.
		*savedView = &SavedView{
			View:     savedViewVar.View,
			Variable: varOp.Xref,
		}
	}
}
