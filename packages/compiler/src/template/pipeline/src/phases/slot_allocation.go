package phases

import (
	"ngc-go/packages/compiler/src/template/pipeline/ir"
	"ngc-go/packages/compiler/src/template/pipeline/ir/src/operations"
	ops_create "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/create"
	ir_traits "ngc-go/packages/compiler/src/template/pipeline/ir/src/traits"

	pipeline "ngc-go/packages/compiler/src/template/pipeline/src/compilation"
)

// AllocateSlots assigns data slots for all operations which implement `ConsumesSlotOpTrait`, and propagate the
// assigned data slots of those operations to any expressions which reference them via
// `UsesSlotIndexTrait`.
//
// This phase is also responsible for counting the number of slots used for each view (its `decls`)
// and propagating that number into the `Template` operations which declare embedded views.
func AllocateSlots(job *pipeline.ComponentCompilationJob) {
	// Map of all declarations in all views within the component which require an assigned slot index.
	// This map needs to be global (across all views within the component) since it's possible to
	// reference a slot from one view from an expression within another (e.g. local references work
	// this way).
	slotMap := make(map[operations.XrefId]int)

	// Process all views in the component and assign slot indexes.
	for _, unit := range job.GetUnits() {
		// Slot indices start at 0 for each view (and are not unique between views).
		slotCount := 0

		for op := unit.GetCreate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			// Only consider declarations which consume data slots.
			if !ir_traits.HasConsumesSlotTrait(op) {
				continue
			}

			// Get the ConsumesSlotOpTrait
			if traitOp, ok := op.(interface {
				GetConsumesSlotTrait() *ir_traits.ConsumesSlotOpTrait
			}); ok {
				trait := traitOp.GetConsumesSlotTrait()
				if trait.Handle == nil {
					trait.Handle = ir.NewSlotHandle()
				}

				// Assign slots to this declaration starting at the current `slotCount`.
				trait.Handle.Slot = &slotCount

				// And track its assigned slot in the `slotMap`.
				slotMap[trait.Xref] = slotCount

				// Each declaration may use more than 1 slot, so increment `slotCount` to reserve the number
				// of slots required.
				slotCount += trait.NumSlotsUsed
			}
		}

		// Record the total number of slots used on the view itself. This will later be propagated into
		// `ir.TemplateOp`s which declare those views (except for the root view).
		if viewUnit, ok := unit.(*pipeline.ViewCompilationUnit); ok {
			viewUnit.Decls = &slotCount
		}
	}

	// After slot assignment, `slotMap` now contains slot assignments for every declaration in the
	// whole template, across all views. Next, look for expressions which implement
	// `UsesSlotIndexExprTrait` and propagate the assigned slot indexes into them.
	// Additionally, this second scan allows us to find `ir.TemplateOp`s which declare views and
	// propagate the number of slots used for each view into the operations which declares it.
	for _, unit := range job.GetUnits() {
		// Process create ops
		for op := unit.GetCreate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			processOpForSlotPropagation(op, job)
		}

		// Process update ops
		for op := unit.GetUpdate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			processOpForSlotPropagation(op, job)
		}
	}
}

func processOpForSlotPropagation(op operations.Op, job *pipeline.ComponentCompilationJob) {
	kind := op.GetKind()
	if kind == ir.OpKindTemplate || kind == ir.OpKindConditionalCreate ||
		kind == ir.OpKindConditionalBranchCreate || kind == ir.OpKindRepeaterCreate {
		// Record the number of slots used by the view this operations declares in the
		// operations itself, so it can be emitted later.
		childView := job.Views[op.(operations.CreateOp).GetXref()]
		if childView != nil && childView.Decls != nil {
			if templateOp, ok := op.(*ops_create.TemplateOp); ok {
				templateOp.Decls = childView.Decls
			} else if conditionalOp, ok := op.(*ops_create.ConditionalCreateOp); ok {
				conditionalOp.Decls = childView.Decls
			} else if branchOp, ok := op.(*ops_create.ConditionalBranchCreateOp); ok {
				branchOp.Decls = childView.Decls
			} else if repeaterOp, ok := op.(*ops_create.RepeaterCreateOp); ok {
				repeaterOp.Decls = childView.Decls
				// TODO: currently we handle the decls for the RepeaterCreate empty template in the reify
				// phase. We should handle that here instead.
			}
		}
	}
}
