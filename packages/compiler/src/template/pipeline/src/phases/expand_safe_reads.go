package phases

import (
	"ngc-go/packages/compiler/src/output"
	"ngc-go/packages/compiler/src/template/pipeline/ir"
	ir_expression "ngc-go/packages/compiler/src/template/pipeline/ir/src/expression"
	ir_operation "ngc-go/packages/compiler/src/template/pipeline/ir/src/operations"

	pipeline "ngc-go/packages/compiler/src/template/pipeline/src/compilation"
)

// SafeTransformContext is the context for safe transform operations
type SafeTransformContext struct {
	Job *pipeline.CompilationJob
}

// ExpandSafeReads finds all unresolved safe read expressions, and converts them into the appropriate output AST
// reads, guarded by null checks. We generate temporaries as needed, to avoid re-evaluating the same
// sub-expression multiple times.
// Safe read expressions such as `a?.b` have different semantics in Angular templates as
// compared to JavaScript. In particular, they default to `null` instead of `undefined`.
func ExpandSafeReads(job *pipeline.CompilationJob) {
	ctx := &SafeTransformContext{Job: job}
	for _, unit := range job.GetUnits() {
		for op := unit.GetCreate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			ir_expression.TransformExpressionsInOp(op, func(expr output.OutputExpression, flags ir_expression.VisitorContextFlag) output.OutputExpression {
				return safeTransform(expr, ctx)
			}, ir_expression.VisitorContextFlagNone)
			ir_expression.TransformExpressionsInOp(op, ternaryTransform, ir_expression.VisitorContextFlagNone)
		}
		for op := unit.GetUpdate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			ir_expression.TransformExpressionsInOp(op, func(expr output.OutputExpression, flags ir_expression.VisitorContextFlag) output.OutputExpression {
				return safeTransform(expr, ctx)
			}, ir_expression.VisitorContextFlagNone)
			ir_expression.TransformExpressionsInOp(op, ternaryTransform, ir_expression.VisitorContextFlagNone)
		}
	}
}

// needsTemporaryInSafeAccess checks if an expression requires a temporary variable to be generated.
func needsTemporaryInSafeAccess(e output.OutputExpression) bool {
	// TODO: We probably want to use an expression visitor to recursively visit all descendents.
	// However, that would potentially do a lot of extra work (because it cannot short circuit), so we
	// implement the logic ourselves for now.
	switch expr := e.(type) {
	case *output.UnaryOperatorExpr:
		return needsTemporaryInSafeAccess(expr.Expr)
	case *output.BinaryOperatorExpr:
		return needsTemporaryInSafeAccess(expr.Lhs) || needsTemporaryInSafeAccess(expr.Rhs)
	case *output.ConditionalExpr:
		if expr.FalseCase != nil && needsTemporaryInSafeAccess(expr.FalseCase) {
			return true
		}
		return needsTemporaryInSafeAccess(expr.Condition) || needsTemporaryInSafeAccess(expr.TrueCase)
	case *output.NotExpr:
		return needsTemporaryInSafeAccess(expr.Condition)
	case *ir_expression.AssignTemporaryExpr:
		return needsTemporaryInSafeAccess(expr.Expr)
	case *output.ReadPropExpr:
		return needsTemporaryInSafeAccess(expr.Receiver)
	case *output.ReadKeyExpr:
		return needsTemporaryInSafeAccess(expr.Receiver) || needsTemporaryInSafeAccess(expr.Index)
	case *output.ParenthesizedExpr:
		return needsTemporaryInSafeAccess(expr.Expr)
	case *output.InvokeFunctionExpr:
		return true
	case *output.LiteralArrayExpr:
		return true
	case *output.LiteralMapExpr:
		return true
	case *ir_expression.SafeInvokeFunctionExpr:
		return true
	case *ir_expression.PipeBindingExpr:
		return true
	default:
		return false
	}
}

// temporariesIn finds all temporary assignments in an expression
func temporariesIn(e output.OutputExpression) map[ir_operation.XrefId]bool {
	temporaries := make(map[ir_operation.XrefId]bool)
	// TODO: Although it's not currently supported by the transform helper, we should be able to
	// short-circuit exploring the tree to do less work. In particular, we don't have to penetrate
	// into the subexpressions of temporary assignments.
	ir_expression.TransformExpressionsInExpression(
		e,
		func(expr output.OutputExpression, flags ir_expression.VisitorContextFlag) output.OutputExpression {
			if assignTmp, ok := expr.(*ir_expression.AssignTemporaryExpr); ok {
				temporaries[assignTmp.Xref] = true
			}
			return expr
		},
		ir_expression.VisitorContextFlagNone,
	)
	return temporaries
}

// eliminateTemporaryAssignments eliminates temporary assignments from an expression
func eliminateTemporaryAssignments(
	e output.OutputExpression,
	tmps map[ir_operation.XrefId]bool,
	ctx *SafeTransformContext,
) output.OutputExpression {
	// TODO: We can be more efficient than the transform helper here. We don't need to visit any
	// descendents of temporary assignments.
	return ir_expression.TransformExpressionsInExpression(
		e,
		func(expr output.OutputExpression, flags ir_expression.VisitorContextFlag) output.OutputExpression {
			if assignTmp, ok := expr.(*ir_expression.AssignTemporaryExpr); ok && tmps[assignTmp.Xref] {
				read := ir_expression.NewReadTemporaryExpr(assignTmp.Xref)
				// `TemplateDefinitionBuilder` has the (accidental?) behavior of generating assignments of
				// temporary variables to themselves. This happens because some subexpression that the
				// temporary refers to, possibly through nested temporaries, has a function call. We copy that
				// behavior here.
				if ctx.Job.Compatibility == ir.CompatibilityModeTemplateDefinitionBuilder {
					return ir_expression.NewAssignTemporaryExpr(read, read.Xref)
				}
				return read
			}
			return expr
		},
		ir_expression.VisitorContextFlagNone,
	)
}

// safeTernaryWithTemporary creates a safe ternary guarded by the input expression, and with a body generated by the provided
// callback on the input expression. Generates a temporary variable assignment if needed, and
// deduplicates nested temporary assignments if needed.
func safeTernaryWithTemporary(
	guard output.OutputExpression,
	body func(output.OutputExpression) output.OutputExpression,
	ctx *SafeTransformContext,
) *ir_expression.SafeTernaryExpr {
	var result [2]output.OutputExpression
	if needsTemporaryInSafeAccess(guard) {
		xref := ctx.Job.AllocateXrefId()
		result[0] = ir_expression.NewAssignTemporaryExpr(guard, xref)
		result[1] = ir_expression.NewReadTemporaryExpr(xref)
	} else {
		result[0] = guard
		result[1] = guard.Clone()
		// Consider an expression like `a?.[b?.c()]?.d`. The `b?.c()` will be transformed first,
		// introducing a temporary assignment into the key. Then, as part of expanding the `?.d`. That
		// assignment will be duplicated into both the guard and expression sides. We de-duplicate it,
		// by transforming it from an assignment into a read on the expression side.
		result[1] = eliminateTemporaryAssignments(result[1], temporariesIn(result[0]), ctx)
	}
	return ir_expression.NewSafeTernaryExpr(result[0], body(result[1]))
}

// isSafeAccessExpression checks if an expression is a safe access expression
func isSafeAccessExpression(e output.OutputExpression) bool {
	_, ok1 := e.(*ir_expression.SafePropertyReadExpr)
	_, ok2 := e.(*ir_expression.SafeKeyedReadExpr)
	_, ok3 := e.(*ir_expression.SafeInvokeFunctionExpr)
	return ok1 || ok2 || ok3
}

// isUnsafeAccessExpression checks if an expression is an unsafe access expression
func isUnsafeAccessExpression(e output.OutputExpression) bool {
	_, ok1 := e.(*output.ReadPropExpr)
	_, ok2 := e.(*output.ReadKeyExpr)
	_, ok3 := e.(*output.InvokeFunctionExpr)
	return ok1 || ok2 || ok3
}

// isAccessExpression checks if an expression is an access expression (safe or unsafe)
func isAccessExpression(e output.OutputExpression) bool {
	return isSafeAccessExpression(e) || isUnsafeAccessExpression(e)
}

// deepestSafeTernary finds the deepest SafeTernaryExpr in an access expression
func deepestSafeTernary(e output.OutputExpression) *ir_expression.SafeTernaryExpr {
	if !isAccessExpression(e) {
		return nil
	}
	var receiver output.OutputExpression
	switch expr := e.(type) {
	case *ir_expression.SafePropertyReadExpr:
		receiver = expr.Receiver
	case *ir_expression.SafeKeyedReadExpr:
		receiver = expr.Receiver
	case *ir_expression.SafeInvokeFunctionExpr:
		receiver = expr.Receiver
	case *output.ReadPropExpr:
		receiver = expr.Receiver
	case *output.ReadKeyExpr:
		receiver = expr.Receiver
	case *output.InvokeFunctionExpr:
		receiver = expr.Fn
	default:
		return nil
	}
	if st, ok := receiver.(*ir_expression.SafeTernaryExpr); ok {
		deepest := st
		for {
			if stExpr, ok := deepest.Expr.(*ir_expression.SafeTernaryExpr); ok {
				deepest = stExpr
			} else {
				break
			}
		}
		return deepest
	}
	return nil
}

// safeTransform transforms safe access expressions into safe ternary expressions
// TODO: When strict compatibility with TemplateDefinitionBuilder is not required, we can use `&&`
// instead to save some code size.
func safeTransform(e output.OutputExpression, ctx *SafeTransformContext) output.OutputExpression {
	if !isAccessExpression(e) {
		return e
	}

	dst := deepestSafeTernary(e)

	if dst != nil {
		switch expr := e.(type) {
		case *output.InvokeFunctionExpr:
			// dst.expr = dst.expr.callFn(e.args)
			// Create a new InvokeFunctionExpr with dst.Expr as the function
			dst.Expr = output.NewInvokeFunctionExpr(dst.Expr, expr.Args, nil, nil, false)
			return getReceiver(expr)
		case *output.ReadPropExpr:
			// dst.expr = dst.expr.prop(e.name)
			dst.Expr = output.NewReadPropExpr(dst.Expr, expr.Name, nil, nil)
			return getReceiver(expr)
		case *output.ReadKeyExpr:
			// dst.expr = dst.expr.key(e.index)
			dst.Expr = output.NewReadKeyExpr(dst.Expr, expr.Index, nil, nil)
			return getReceiver(expr)
		case *ir_expression.SafeInvokeFunctionExpr:
			dst.Expr = safeTernaryWithTemporary(dst.Expr, func(r output.OutputExpression) output.OutputExpression {
				return output.NewInvokeFunctionExpr(r, expr.Args, nil, nil, false)
			}, ctx)
			return getReceiver(expr)
		case *ir_expression.SafePropertyReadExpr:
			dst.Expr = safeTernaryWithTemporary(dst.Expr, func(r output.OutputExpression) output.OutputExpression {
				return output.NewReadPropExpr(r, expr.Name, nil, nil)
			}, ctx)
			return getReceiver(expr)
		case *ir_expression.SafeKeyedReadExpr:
			dst.Expr = safeTernaryWithTemporary(dst.Expr, func(r output.OutputExpression) output.OutputExpression {
				return output.NewReadKeyExpr(r, expr.Index, nil, nil)
			}, ctx)
			return getReceiver(expr)
		}
	} else {
		switch expr := e.(type) {
		case *ir_expression.SafeInvokeFunctionExpr:
			return safeTernaryWithTemporary(expr.Receiver, func(r output.OutputExpression) output.OutputExpression {
				return output.NewInvokeFunctionExpr(r, expr.Args, nil, nil, false)
			}, ctx)
		case *ir_expression.SafePropertyReadExpr:
			return safeTernaryWithTemporary(expr.Receiver, func(r output.OutputExpression) output.OutputExpression {
				return output.NewReadPropExpr(r, expr.Name, nil, nil)
			}, ctx)
		case *ir_expression.SafeKeyedReadExpr:
			return safeTernaryWithTemporary(expr.Receiver, func(r output.OutputExpression) output.OutputExpression {
				return output.NewReadKeyExpr(r, expr.Index, nil, nil)
			}, ctx)
		}
	}

	return e
}

// getReceiver extracts the receiver from an access expression
func getReceiver(e output.OutputExpression) output.OutputExpression {
	switch expr := e.(type) {
	case *ir_expression.SafePropertyReadExpr:
		return expr.Receiver
	case *ir_expression.SafeKeyedReadExpr:
		return expr.Receiver
	case *ir_expression.SafeInvokeFunctionExpr:
		return expr.Receiver
	case *output.ReadPropExpr:
		return expr.Receiver
	case *output.ReadKeyExpr:
		return expr.Receiver
	case *output.InvokeFunctionExpr:
		// For InvokeFunctionExpr, the receiver is the function itself
		return expr.Fn
	default:
		return nil
	}
}

// ternaryTransform transforms SafeTernaryExpr into a ConditionalExpr
func ternaryTransform(e output.OutputExpression, flags ir_expression.VisitorContextFlag) output.OutputExpression {
	safeTernary, ok := e.(*ir_expression.SafeTernaryExpr)
	if !ok {
		return e
	}
	return output.NewParenthesizedExpr(
		output.NewConditionalExpr(
			output.NewBinaryOperatorExpr(output.BinaryOperatorEquals, safeTernary.Guard, output.NullExpr, nil, nil),
			output.NullExpr,
			safeTernary.Expr,
			nil, // typ
			nil, // sourceSpan
		),
		nil, // typ
		nil, // sourceSpan
	)
}
