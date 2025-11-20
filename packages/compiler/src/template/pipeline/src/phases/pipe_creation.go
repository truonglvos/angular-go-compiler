package phases

import (
	"ngc-go/packages/compiler/src/output"
	"ngc-go/packages/compiler/src/template/pipeline/ir"
	"ngc-go/packages/compiler/src/template/pipeline/ir/src/expression"
	"ngc-go/packages/compiler/src/template/pipeline/ir/src/operations"
	ops_create "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/create"
	ir_traits "ngc-go/packages/compiler/src/template/pipeline/ir/src/traits"

	pipeline "ngc-go/packages/compiler/src/template/pipeline/src/compilation"
)

// CreatePipes generates pipe creation instructions. We do this based on the pipe bindings found in
// the update block, in the order we see them.
// When not in compatibility mode, we can simply group all these creation instructions together, to
// maximize chaining opportunities.
func CreatePipes(job *pipeline.CompilationJob) {
	for _, unit := range job.GetUnits() {
		processPipeBindingsInView(unit)
	}
}

func processPipeBindingsInView(unit pipeline.CompilationUnit) {
	for op := unit.GetUpdate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
		expression.VisitExpressionsInOp(op, func(expr output.OutputExpression, flags expression.VisitorContextFlag) {
			if !expression.IsIrExpression(expr) {
				return
			}

			// Check if it's a PipeBindingExpr
			pipeBinding, ok := expr.(*expression.PipeBindingExpr)
			if !ok {
				return
			}

			if flags&expression.VisitorContextFlagInChildOperation != 0 {
				panic("AssertionError: pipe bindings should not appear in child expressions")
			}

			if job := unit.GetJob(); job.Compatibility == ir.CompatibilityModeTemplateDefinitionBuilder {
				// TODO: We can delete this cast and check once compatibility mode is removed.
				// In compatibility mode, we need to find the target slot handle
				var targetXref operations.XrefId
				if updateOp, ok := op.(interface{ GetXref() operations.XrefId }); ok {
					targetXref = updateOp.GetXref()
				} else if updateOp, ok := op.(interface{ GetTarget() operations.XrefId }); ok {
					targetXref = updateOp.GetTarget()
				}
				if targetXref == 0 {
					panic("AssertionError: expected slot handle to be assigned for pipe creation")
				}
				addPipeToCreationBlock(unit, targetXref, pipeBinding)
			} else {
				// When not in compatibility mode, we just add the pipe to the end of the create block. This
				// is not only simpler and faster, but allows more chaining opportunities for other
				// instructions.
				pipeOp := ops_create.NewPipeOp(pipeBinding.Target, pipeBinding.TargetSlot, pipeBinding.Name)
				unit.GetCreate().Push(pipeOp)
			}
		})
	}
}

func addPipeToCreationBlock(
	unit pipeline.CompilationUnit,
	afterTargetXref operations.XrefId,
	binding *expression.PipeBindingExpr,
) {
	// Find the appropriate point to insert the Pipe creation operations.
	// We're looking for `afterTargetXref` (and also want to insert after any other pipe operations
	// which might be beyond it).
	for op := unit.GetCreate().Head().Next(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
		if !ir_traits.HasConsumesSlotTrait(op) {
			continue
		}

		// Type assert to CreateOp to access GetXref()
		createOp, ok := op.(operations.CreateOp)
		if !ok {
			continue
		}

		if createOp.GetXref() != afterTargetXref {
			continue
		}

		// We've found a tentative insertion point; however, we also want to skip past any _other_ pipe
		// operations present.
		for op.Next() != nil && op.Next().GetKind() == ir.OpKindPipe {
			op = op.Next()
		}

		pipe := ops_create.NewPipeOp(binding.Target, binding.TargetSlot, binding.Name)
		unit.GetCreate().InsertBefore(pipe, op.Next())

		// This completes adding the pipe to the creation block.
		return
	}

	// At this point, we've failed to add the pipe to the creation block.
	panic("AssertionError: unable to find insertion point for pipe " + binding.Name)
}
