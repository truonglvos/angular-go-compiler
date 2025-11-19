package phases

import (
	"ngc-go/packages/compiler/output"
	ir "ngc-go/packages/compiler/template/pipeline/ir/src"
	ir_expression "ngc-go/packages/compiler/template/pipeline/ir/src/expression"
	ir_operation "ngc-go/packages/compiler/template/pipeline/ir/src/operations"
	ops_create "ngc-go/packages/compiler/template/pipeline/ir/src/ops/create"
	ops_update "ngc-go/packages/compiler/template/pipeline/ir/src/ops/update"
	ir_traits "ngc-go/packages/compiler/template/pipeline/ir/src/traits"

	pipeline "ngc-go/packages/compiler/template/pipeline/src/compilation"
)

// BlockState represents the state of an i18n block
type BlockState struct {
	BlockXref        ir_operation.XrefId
	LastSlotConsumer ir_operation.XrefId
}

// AssignI18nSlotDependencies updates i18n expression ops to target the last slot in their owning i18n block,
// and moves them after the last update instruction that depends on that slot.
func AssignI18nSlotDependencies(job *pipeline.CompilationJob) {
	for _, unit := range job.GetUnits() {
		// The first update op.
		var updateOp ir_operation.Op = unit.GetUpdate().Head()

		// I18n expressions currently being moved during the iteration.
		var i18nExpressionsInProgress []*ops_update.I18nExpressionOp

		// Non-null while we are iterating through an i18nStart/i18nEnd pair
		var state *BlockState

		for createOp := unit.GetCreate().Head(); createOp != nil; createOp = createOp.Next() {
			if i18nStart, ok := createOp.(*ops_create.I18nStartOp); ok {
				state = &BlockState{
					BlockXref:        i18nStart.Xref,
					LastSlotConsumer: i18nStart.Xref,
				}
			} else if _, ok := createOp.(*ops_create.I18nEndOp); ok {
				for _, op := range i18nExpressionsInProgress {
					op.Target = state.LastSlotConsumer
					unit.GetUpdate().InsertBefore(updateOp, op)
				}
				i18nExpressionsInProgress = nil
				state = nil
			}

			if ir_traits.HasConsumesSlotTrait(createOp) {
				if state != nil {
					if createOpWithXref, ok := createOp.(ir_operation.CreateOp); ok {
						state.LastSlotConsumer = createOpWithXref.GetXref()
					}
				}

				for {
					if updateOp == nil || updateOp.Next() == nil {
						break
					}

					if state != nil {
						if i18nExprOp, ok := updateOp.(*ops_update.I18nExpressionOp); ok &&
							i18nExprOp.Usage == ir.I18nExpressionForI18nText &&
							i18nExprOp.I18nOwner == state.BlockXref {
							opToRemove := updateOp
							updateOp = updateOp.Next()
							unit.GetUpdate().Remove(opToRemove)
							i18nExpressionsInProgress = append(i18nExpressionsInProgress, i18nExprOp)
							continue
						}
					}

					hasDifferentTarget := false
					if ir_traits.HasDependsOnSlotContextTrait(updateOp) {
						if depOp, ok := updateOp.(interface {
							GetDependsOnSlotContextTrait() *ir_traits.DependsOnSlotContextOpTrait
						}); ok {
							trait := depOp.GetDependsOnSlotContextTrait()
							if createOpWithXref, ok := createOp.(ir_operation.CreateOp); ok && trait.Target != createOpWithXref.GetXref() {
								hasDifferentTarget = true
							}
						}
					} else if updateOp.GetKind() == ir.OpKindStatement || updateOp.GetKind() == ir.OpKindVariable {
						// Some expressions may consume slots as well (e.g. `storeLet`).
						ir_expression.VisitExpressionsInOp(updateOp, func(expr output.OutputExpression, flags ir_expression.VisitorContextFlag) {
							if !hasDifferentTarget && ir_traits.HasDependsOnSlotContextTrait(expr) {
								if depExpr, ok := expr.(interface {
									GetDependsOnSlotContextTrait() *ir_traits.DependsOnSlotContextOpTrait
								}); ok {
									trait := depExpr.GetDependsOnSlotContextTrait()
									if createOpWithXref, ok := createOp.(ir_operation.CreateOp); ok && trait.Target != createOpWithXref.GetXref() {
										hasDifferentTarget = true
									}
								}
							}
						})
					}

					if hasDifferentTarget {
						break
					}

					updateOp = updateOp.Next()
				}
			}
		}
	}
}
