package phases

import (
	"ngc-go/packages/compiler/src/output"
	"ngc-go/packages/compiler/src/template/pipeline/ir"
	ir_operation "ngc-go/packages/compiler/src/template/pipeline/ir/src/operations"
	ops_create "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/create"
	ops_shared "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/shared"
	ops_update "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/update"

	pipeline "ngc-go/packages/compiler/src/template/pipeline/src/compilation"
)

// ConvertAnimations converts animation binding operations to animation creation operations.
func ConvertAnimations(job *pipeline.CompilationJob) {
	elements := make(map[ir_operation.XrefId]ir_operation.CreateOp)
	for _, unit := range job.GetUnits() {
		for op := unit.GetCreate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			createOp, ok := op.(ir_operation.CreateOp)
			if !ok {
				continue
			}
			if !ops_create.IsElementOrContainerOp(createOp) {
				continue
			}
			elements[createOp.GetXref()] = createOp
		}
	}

	for _, unit := range job.GetUnits() {
		// Process both create and update operations
		for op := unit.GetCreate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			if op.GetKind() == ir.OpKindAnimationBinding {
				animBindingOp, ok := op.(*ops_update.AnimationBindingOp)
				if !ok {
					continue
				}
				createAnimationOp := getAnimationOp(animBindingOp)
				if job.Kind == pipeline.CompilationJobKindHost {
					unit.GetCreate().Push(createAnimationOp)
				} else {
					elementOp := lookupElementConvert(elements, animBindingOp.Target)
					unit.GetCreate().InsertAfter(createAnimationOp, elementOp)
				}
				unit.GetCreate().Remove(op)
			}
		}
		for op := unit.GetUpdate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			if op.GetKind() == ir.OpKindAnimationBinding {
				animBindingOp, ok := op.(*ops_update.AnimationBindingOp)
				if !ok {
					continue
				}
				createAnimationOp := getAnimationOp(animBindingOp)
				if job.Kind == pipeline.CompilationJobKindHost {
					unit.GetCreate().Push(createAnimationOp)
				} else {
					elementOp := lookupElementConvert(elements, animBindingOp.Target)
					unit.GetCreate().InsertAfter(createAnimationOp, elementOp)
				}
				unit.GetUpdate().Remove(op)
			}
		}
	}
}

// lookupElementConvert looks up an element in the given map by xref ID.
func lookupElementConvert(
	elements map[ir_operation.XrefId]ir_operation.CreateOp,
	xref ir_operation.XrefId,
) ir_operation.CreateOp {
	el, exists := elements[xref]
	if !exists {
		panic("All attributes should have an element-like target.")
	}
	return el
}

func getAnimationOp(op *ops_update.AnimationBindingOp) ir_operation.CreateOp {
	// Check if expression is a string (Interpolation with single string) or an Expression
	var expr output.OutputExpression
	var isString bool
	if e, ok := op.Expression.(output.OutputExpression); ok {
		expr = e
		// Check if it's a literal string
		if lit, ok := e.(*output.LiteralExpr); ok {
			if _, ok := lit.Value.(string); ok {
				isString = true
			}
		}
	} else {
		// If it's an Interpolation, check if it's a simple string case
		if interp, ok := op.Expression.(*ops_update.Interpolation); ok {
			// Simple string case: single string with no expressions
			if len(interp.Expressions) == 0 && len(interp.Strings) == 1 {
				isString = true
				expr = output.NewLiteralExpr(interp.Strings[0], nil, nil)
			} else {
				// Complex case - use first expression
				if len(interp.Expressions) > 0 {
					expr = interp.Expressions[0]
				} else {
					panic("Animation expression must have at least one expression")
				}
			}
		} else {
			panic("Animation expression must be an OutputExpression or Interpolation")
		}
	}

	animationKind := ir.AnimationKindLeave
	if op.Name == "animate.enter" {
		animationKind = ir.AnimationKindEnter
	}

	if isString {
		// this is a simple string case
		return ops_create.NewAnimationStringOp(
			op.Name,
			op.Target,
			animationKind,
			expr,
			nil, // securityContext - TODO: get from op if available
			nil, // sourceSpan - TODO: get from expr if available
		)
	} else {
		// Expression case - create AnimationOp with handler ops
		returnStmt := output.NewReturnStatement(expr, expr.GetSourceSpan(), nil)
		handlerOps := []ir_operation.UpdateOp{ops_shared.NewStatementOp(returnStmt)}
		return ops_create.NewAnimationOp(
			op.Name,
			op.Target,
			animationKind,
			handlerOps,
			nil, // securityContext - TODO: get from op if available
			nil, // sourceSpan - TODO: get from expr if available
		)
	}
}
