package phases

import (
	ir_operation "ngc-go/packages/compiler/src/template/pipeline/ir/src/operations"
	ops_create "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/create"

	pipeline "ngc-go/packages/compiler/src/template/pipeline/src/compilation"
)

const containerTag = "ng-container"

// GenerateNgContainerOps replaces an `Element` or `ElementStart` whose tag is `ng-container` with a specific op.
func GenerateNgContainerOps(job *pipeline.CompilationJob) {
	for _, unit := range job.GetUnits() {
		updatedElementXrefs := make(map[ir_operation.XrefId]bool)

		for op := unit.GetCreate().Head(); op != nil; op = op.Next() {
			if elementStart, ok := op.(*ops_create.ElementStartOp); ok && elementStart.Tag != nil && *elementStart.Tag == containerTag {
				// Replace the `ElementStart` instruction with `ContainerStart`.
				containerStart := ops_create.NewContainerStartOp(
					elementStart.Xref,
					elementStart.StartSourceSpan,
					elementStart.WholeSourceSpan,
				)
				// Copy over the fields from ElementStartOp
				containerStart.Handle = elementStart.Handle
				containerStart.NumSlotsUsed = elementStart.NumSlotsUsed
				containerStart.Attributes = elementStart.Attributes
				containerStart.LocalRefs = elementStart.LocalRefs
				containerStart.NonBindable = elementStart.NonBindable

				unit.GetCreate().Replace(elementStart, containerStart)
				updatedElementXrefs[elementStart.Xref] = true
			}

			if elementEnd, ok := op.(*ops_create.ElementEndOp); ok && updatedElementXrefs[elementEnd.Xref] {
				// This `ElementEnd` is associated with an `ElementStart` we already transmuted.
				// Replace it with ContainerEndOp
				containerEnd := ops_create.NewContainerEndOp(elementEnd.Xref, elementEnd.SourceSpan)
				unit.GetCreate().Replace(elementEnd, containerEnd)
			}
		}
	}
}
