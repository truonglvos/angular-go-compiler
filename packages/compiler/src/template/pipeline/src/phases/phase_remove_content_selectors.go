package phases

import (
	"fmt"

	"ngc-go/packages/compiler/src/template/pipeline/ir"
	"ngc-go/packages/compiler/src/template/pipeline/ir/src/operations"
	ops_update "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/update"

	pipeline "ngc-go/packages/compiler/src/template/pipeline/src/compilation"
	pipeline_util "ngc-go/packages/compiler/src/template/pipeline/src/util"
)

// RemoveContentSelectors removes attributes of `ng-content` named 'select', because they control which
// content matches as a property of the `projection`, and are not a plain attribute.
func RemoveContentSelectors(job *pipeline.CompilationJob) {
	for _, unit := range job.GetUnits() {
		elements := pipeline_util.CreateOpXrefMap(unit)
		for op := unit.GetCreate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			if op.GetKind() == ir.OpKindBinding {
				bindingOp, ok := op.(*ops_update.BindingOp)
				if !ok {
					continue
				}
				target := lookupInXrefMap(elements, bindingOp.Target)
				if isSelectAttribute(bindingOp.Name) && target.GetKind() == ir.OpKindProjection {
					unit.GetUpdate().Remove(op)
				}
			}
		}
		for op := unit.GetUpdate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			if op.GetKind() == ir.OpKindBinding {
				bindingOp, ok := op.(*ops_update.BindingOp)
				if !ok {
					continue
				}
				target := lookupInXrefMap(elements, bindingOp.Target)
				if isSelectAttribute(bindingOp.Name) && target.GetKind() == ir.OpKindProjection {
					unit.GetUpdate().Remove(op)
				}
			}
		}
	}
}

// isSelectAttribute checks if a name is the 'select' attribute
func isSelectAttribute(name string) bool {
	return name == "select" || name == "SELECT"
}

// lookupInXrefMap looks up an element in the given map by xref ID.
func lookupInXrefMap(
	elements map[operations.XrefId]pipeline_util.OpXrefMapEntry,
	xref operations.XrefId,
) pipeline_util.OpXrefMapEntry {
	el, exists := elements[xref]
	if !exists {
		panic(fmt.Sprintf("All attributes should have an slottable target: %d", xref))
	}
	return el
}
