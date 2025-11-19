package phases

import (
	"fmt"

	"ngc-go/packages/compiler/output"
	ir_expression "ngc-go/packages/compiler/template/pipeline/ir/src/expression"
	ir_operation "ngc-go/packages/compiler/template/pipeline/ir/src/operations"
	ops_update "ngc-go/packages/compiler/template/pipeline/ir/src/ops/update"
	ir_traits "ngc-go/packages/compiler/template/pipeline/ir/src/traits"

	pipeline "ngc-go/packages/compiler/template/pipeline/src/compilation"
)

// GenerateAdvance generates `ir.AdvanceOp`s in between `ir.UpdateOp`s that ensure the runtime's
// implicit slot context will be advanced correctly.
func GenerateAdvance(job *pipeline.CompilationJob) {
	for _, unit := range job.GetUnits() {
		// First build a map of all of the declarations in the view that have assigned slots.
		slotMap := make(map[ir_operation.XrefId]int)
		for op := unit.GetCreate().Head(); op != nil; op = op.Next() {
			if !ir_traits.HasConsumesSlotTrait(op) {
				continue
			}

			if createOp, ok := op.(ir_operation.CreateOp); ok {
				if trait := getConsumesSlotTrait(op); trait != nil {
					if trait.Handle == nil || trait.Handle.Slot == nil {
						panic("AssertionError: expected slots to have been allocated before generating advance() calls")
					}
					slotMap[createOp.GetXref()] = *trait.Handle.Slot
				}
			}
		}

		// Next, step through the update operations and generate `ir.AdvanceOp`s as required to ensure
		// the runtime's implicit slot counter will be set to the correct slot before executing each
		// update operation which depends on it.
		//
		// To do that, we track what the runtime's slot counter will be through the update operations.
		slotContext := 0
		for op := unit.GetUpdate().Head(); op != nil; op = op.Next() {
			var consumer *ir_traits.DependsOnSlotContextOpTrait

			if ir_traits.HasDependsOnSlotContextTrait(op) {
				if depOp, ok := op.(interface {
					GetDependsOnSlotContextTrait() *ir_traits.DependsOnSlotContextOpTrait
				}); ok {
					consumer = depOp.GetDependsOnSlotContextTrait()
				}
			} else {
				ir_expression.VisitExpressionsInOp(op, func(expr output.OutputExpression, flags ir_expression.VisitorContextFlag) {
					if consumer == nil && ir_traits.HasDependsOnSlotContextTrait(expr) {
						if depExpr, ok := expr.(interface {
							GetDependsOnSlotContextTrait() *ir_traits.DependsOnSlotContextOpTrait
						}); ok {
							consumer = depExpr.GetDependsOnSlotContextTrait()
						}
					}
				})
			}

			if consumer == nil {
				continue
			}

			slot, ok := slotMap[consumer.Target]
			if !ok {
				// We expect ops that _do_ depend on the slot counter to point at declarations that exist in
				// the `slotMap`.
				panic(fmt.Sprintf("AssertionError: reference to unknown slot for target %d", consumer.Target))
			}

			// Does the slot counter need to be adjusted?
			if slotContext != slot {
				// If so, generate an `ir.AdvanceOp` to advance the counter.
				delta := slot - slotContext
				if delta < 0 {
					panic("AssertionError: slot counter should never need to move backwards")
				}

				advanceOp := ops_update.NewAdvanceOp(delta)
				unit.GetUpdate().InsertBefore(op, advanceOp)
				slotContext = slot
			}
		}
	}
}

// getConsumesSlotTrait gets the ConsumesSlotOpTrait from an op
func getConsumesSlotTrait(op ir_operation.Op) *ir_traits.ConsumesSlotOpTrait {
	if traitOp, ok := op.(interface {
		GetConsumesSlotTrait() *ir_traits.ConsumesSlotOpTrait
	}); ok {
		return traitOp.GetConsumesSlotTrait()
	}
	return nil
}
