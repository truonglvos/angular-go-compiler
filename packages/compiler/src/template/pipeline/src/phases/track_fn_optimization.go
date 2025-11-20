package phases

import (
	"ngc-go/packages/compiler/src/output"
	r3_identifiers "ngc-go/packages/compiler/src/render3/r3_identifiers"
	"ngc-go/packages/compiler/src/template/pipeline/ir"
	"ngc-go/packages/compiler/src/template/pipeline/ir/src/expression"
	"ngc-go/packages/compiler/src/template/pipeline/ir/src/operations"
	ops_create "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/create"
	"ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/shared"

	pipeline "ngc-go/packages/compiler/src/template/pipeline/src/compilation"
)

// OptimizeTrackFns optimizes `track` functions in `for` repeaters. They can sometimes be "optimized,"
// i.e. transformed into inline expressions, in lieu of an external function call. For example,
// tracking by `$index` can be optimized into an inline `trackByIndex` reference. This phase checks
// track expressions for optimizable cases.
func OptimizeTrackFns(job *pipeline.CompilationJob) {
	for _, unit := range job.GetUnits() {
		for op := unit.GetCreate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			if op.GetKind() != ir.OpKindRepeaterCreate {
				continue
			}

			repeaterOp, ok := op.(*ops_create.RepeaterCreateOp)
			if !ok {
				continue
			}

			if readVar, ok := repeaterOp.Track.(*output.ReadVarExpr); ok && readVar.Name == "$index" {
				// Top-level access of `$index` uses the built in `repeaterTrackByIndex`.
				repeaterOp.TrackByFn = output.NewExternalExpr(r3_identifiers.RepeaterTrackByIndex, nil, nil, nil)
			} else if readVar, ok := repeaterOp.Track.(*output.ReadVarExpr); ok && readVar.Name == "$item" {
				// Top-level access of the item uses the built in `repeaterTrackByIdentity`.
				repeaterOp.TrackByFn = output.NewExternalExpr(r3_identifiers.RepeaterTrackByIdentity, nil, nil, nil)
			} else if isTrackByFunctionCall(job.GetRoot().GetXref(), repeaterOp.Track, unit.GetXref()) {
				// Mark the function as using the component instance to play it safe
				// since the method might be using `this` internally (see #53628).
				repeaterOp.UsesComponentInstance = true

				invokeExpr, ok := repeaterOp.Track.(*output.InvokeFunctionExpr)
				if !ok {
					continue
				}
				readProp, ok := invokeExpr.Fn.(*output.ReadPropExpr)
				if !ok {
					continue
				}
				contextExpr, ok := readProp.Receiver.(*expression.ContextExpr)
				if !ok {
					continue
				}

				// Top-level method calls in the form of `fn($index, item)` can be passed in directly.
				if contextExpr.View == unit.GetXref() {
					// TODO: this may be wrong
					repeaterOp.TrackByFn = readProp
				} else {
					// This is a plain method call, but not in the component's root view.
					// We need to get the component instance, and then call the method on it.
					componentInstance := output.NewExternalExpr(r3_identifiers.ComponentInstance, nil, nil, nil)
					componentInstanceCall := output.NewInvokeFunctionExpr(componentInstance, []output.OutputExpression{}, nil, nil, false)
					repeaterOp.TrackByFn = output.NewReadPropExpr(componentInstanceCall, readProp.Name, nil, nil)
					// Because the context is not available (without a special function), we don't want to
					// try to resolve it later. Let's get rid of it by overwriting the original track
					// expression (which won't be used anyway).
					repeaterOp.Track = repeaterOp.TrackByFn
				}
			} else {
				// The track function could not be optimized.
				// Replace context reads with a special IR expression, since context reads in a track
				// function are emitted specially.
				repeaterOp.Track = expression.TransformExpressionsInExpression(
					repeaterOp.Track,
					func(expr output.OutputExpression, flags expression.VisitorContextFlag) output.OutputExpression {
						if _, ok := expr.(*expression.PipeBindingExpr); ok {
							panic("Illegal State: Pipes are not allowed in this context")
						} else if _, ok := expr.(*expression.PipeBindingVariadicExpr); ok {
							panic("Illegal State: Pipes are not allowed in this context")
						} else if contextExpr, ok := expr.(*expression.ContextExpr); ok {
							repeaterOp.UsesComponentInstance = true
							return expression.NewTrackContextExpr(contextExpr.View)
						}
						return expr
					},
					expression.VisitorContextFlagNone,
				)

				// Also create an OpList for the tracking expression since it may need
				// additional ops when generating the final code (e.g. temporary variables).
				trackOpList := operations.NewOpList()
				returnStmt := output.NewReturnStatement(repeaterOp.Track, repeaterOp.Track.GetSourceSpan(), nil)
				stmtOp := shared.NewStatementOp(returnStmt)
				trackOpList.Push(stmtOp)
				repeaterOp.TrackByOps = trackOpList
			}
		}
	}
}

// isTrackByFunctionCall checks if an expression is a track-by function call in the form
// `fn($index, item)` where `fn` is a method on the component context.
func isTrackByFunctionCall(
	rootView operations.XrefId,
	expr output.OutputExpression,
	unitXref operations.XrefId,
) bool {
	invokeExpr, ok := expr.(*output.InvokeFunctionExpr)
	if !ok || len(invokeExpr.Args) == 0 || len(invokeExpr.Args) > 2 {
		return false
	}

	readProp, ok := invokeExpr.Fn.(*output.ReadPropExpr)
	if !ok {
		return false
	}

	contextExpr, ok := readProp.Receiver.(*expression.ContextExpr)
	if !ok || contextExpr.View != rootView {
		return false
	}

	if len(invokeExpr.Args) == 0 {
		return false
	}

	arg0, ok := invokeExpr.Args[0].(*output.ReadVarExpr)
	if !ok || arg0.Name != "$index" {
		return false
	}

	if len(invokeExpr.Args) == 1 {
		return true
	}

	if len(invokeExpr.Args) < 2 {
		return false
	}

	arg1, ok := invokeExpr.Args[1].(*output.ReadVarExpr)
	if !ok || arg1.Name != "$item" {
		return false
	}

	return true
}
