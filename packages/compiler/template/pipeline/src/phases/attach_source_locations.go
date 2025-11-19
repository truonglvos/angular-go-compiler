package phases

import (
	ir "ngc-go/packages/compiler/template/pipeline/ir/src"
	ops_create "ngc-go/packages/compiler/template/pipeline/ir/src/ops/create"
	ir_traits "ngc-go/packages/compiler/template/pipeline/ir/src/traits"
	"ngc-go/packages/compiler/util"

	pipeline "ngc-go/packages/compiler/template/pipeline/src/compilation"
)

// AttachSourceLocations locates all of the elements defined in a creation block and outputs an op
// that will expose their definition location in the DOM.
func AttachSourceLocations(job *pipeline.ComponentCompilationJob) {
	if !job.EnableDebugLocations || job.RelativeTemplatePath == nil {
		return
	}

	for _, unit := range job.GetUnits() {
		viewUnit, ok := unit.(*pipeline.ViewCompilationUnit)
		if !ok {
			continue
		}

		var locations []ops_create.ElementSourceLocation

		for op := viewUnit.GetCreate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			kind := op.GetKind()
			if kind != ir.OpKindElementStart && kind != ir.OpKindElement {
				continue
			}

			// Get the start source span
			var startSpan *util.ParseSourceSpan
			if elementOp, ok := op.(interface {
				GetStartSourceSpan() *util.ParseSourceSpan
			}); ok {
				startSpan = elementOp.GetStartSourceSpan()
			} else {
				continue
			}

			if startSpan == nil || startSpan.Start == nil {
				continue
			}

			// Get the slot handle
			var handle *ir.SlotHandle
			if slotOp, ok := op.(interface {
				GetConsumesSlotTrait() *ir_traits.ConsumesSlotOpTrait
			}); ok {
				trait := slotOp.GetConsumesSlotTrait()
				if trait != nil {
					handle = trait.Handle
				}
			}

			if handle == nil {
				continue
			}

			start := startSpan.Start
			locations = append(locations, ops_create.ElementSourceLocation{
				TargetSlot: handle,
				Offset:     start.Offset,
				Line:       start.Line,
				Column:     start.Col,
			})
		}

		if len(locations) > 0 && job.RelativeTemplatePath != nil {
			sourceLocationOp := ops_create.NewSourceLocationOp(*job.RelativeTemplatePath, locations)
			viewUnit.GetCreate().Push(sourceLocationOp)
		}
	}
}
