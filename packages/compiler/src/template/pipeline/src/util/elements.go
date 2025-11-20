package pipeline_util

import (
	ir_operations "ngc-go/packages/compiler/src/template/pipeline/ir/src/operations"
	ops_create "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/create"
	ir_traits "ngc-go/packages/compiler/src/template/pipeline/ir/src/traits"
	pipeline_compilation "ngc-go/packages/compiler/src/template/pipeline/src/compilation"
)

// OpXrefMapEntry represents an entry in the op xref map
// It combines ConsumesSlotOpTrait and CreateOp
type OpXrefMapEntry interface {
	ir_operations.CreateOp
	GetConsumesSlotTrait() *ir_traits.ConsumesSlotOpTrait
}

// CreateOpXrefMap gets a map of all elements in the given view by their xref id
func CreateOpXrefMap(unit pipeline_compilation.CompilationUnit) map[ir_operations.XrefId]OpXrefMapEntry {
	result := make(map[ir_operations.XrefId]OpXrefMapEntry)

	for op := unit.GetCreate().Head(); op != nil; op = op.Next() {
		if !ir_traits.HasConsumesSlotTrait(op) {
			continue
		}

		// Type assert to get the xref and entry
		if createOp, ok := op.(ir_operations.CreateOp); ok {
			xref := createOp.GetXref()
			if entry, ok := op.(OpXrefMapEntry); ok {
				result[xref] = entry

				// TODO(dylhunn): `@for` loops with `@empty` blocks need to be special-cased here,
				// because the slot consumer trait currently only supports one slot per consumer and we
				// need two. This should be revisited when making the refactors mentioned in:
				// https://github.com/angular/angular/pull/53620#discussion_r1430918822
				if repeaterOp, ok := op.(*ops_create.RepeaterCreateOp); ok && repeaterOp.EmptyView != 0 {
					result[repeaterOp.EmptyView] = entry
				}
			}
		}
	}

	return result
}
