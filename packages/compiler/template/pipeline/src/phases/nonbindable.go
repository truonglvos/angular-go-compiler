package phases

import (
	ir "ngc-go/packages/compiler/template/pipeline/ir/src"
	ir_operation "ngc-go/packages/compiler/template/pipeline/ir/src/operations"
	ops_create "ngc-go/packages/compiler/template/pipeline/ir/src/ops/create"

	pipeline "ngc-go/packages/compiler/template/pipeline/src/compilation"
)

// lookupElementNonBindAble looks up an element in the given map by xref ID
func lookupElementNonBindAble(
	elements map[ir_operation.XrefId]ir_operation.CreateOp,
	xref ir_operation.XrefId,
) ir_operation.CreateOp {
	el, ok := elements[xref]
	if !ok {
		panic("All attributes should have an element-like target.")
	}
	return el
}

// isElementOrContainerOp checks if an op is an element or container op
func isElementOrContainerOp(op ir_operation.Op) bool {
	kind := op.GetKind()
	return kind == ir.OpKindElement || kind == ir.OpKindElementStart ||
		kind == ir.OpKindContainer || kind == ir.OpKindContainerStart ||
		kind == ir.OpKindTemplate
}

// DisableBindings looks up elements and emits `disableBindings` and `enableBindings`
// instructions for containers marked with `ngNonBindable`.
// When a container is marked with `ngNonBindable`, the non-bindable characteristic also applies to
// all descendants of that container. Therefore, we must emit `disableBindings` and `enableBindings`
// instructions for every such container.
func DisableBindings(job *pipeline.CompilationJob) {
	elements := make(map[ir_operation.XrefId]ir_operation.CreateOp)
	for _, view := range job.GetUnits() {
		for op := view.GetCreate().Head(); op != nil; op = op.Next() {
			if !isElementOrContainerOp(op) {
				continue
			}
			if createOp, ok := op.(ir_operation.CreateOp); ok {
				elements[createOp.GetXref()] = createOp
			}
		}
	}

	for _, unit := range job.GetUnits() {
		for op := unit.GetCreate().Head(); op != nil; op = op.Next() {
			kind := op.GetKind()
			if kind == ir.OpKindElementStart || kind == ir.OpKindContainerStart {
				var nonBindable bool
				if elementStart, ok := op.(*ops_create.ElementStartOp); ok {
					nonBindable = elementStart.NonBindable
				} else if containerStart, ok := op.(*ops_create.ContainerStartOp); ok {
					nonBindable = containerStart.NonBindable
				}

				if nonBindable {
					if createOp, ok := op.(ir_operation.CreateOp); ok {
						disableOp := ops_create.NewDisableBindingsOp(createOp.GetXref())
						unit.GetCreate().InsertAfter(op, disableOp)
					}
				}
			}

			if kind == ir.OpKindElementEnd || kind == ir.OpKindContainerEnd {
				if createOp, ok := op.(ir_operation.CreateOp); ok {
					element := lookupElementNonBindAble(elements, createOp.GetXref())
					var nonBindable bool
					if elementStart, ok := element.(*ops_create.ElementStartOp); ok {
						nonBindable = elementStart.NonBindable
					} else if containerStart, ok := element.(*ops_create.ContainerStartOp); ok {
						nonBindable = containerStart.NonBindable
					} else if elementOp, ok := element.(*ops_create.ElementOp); ok {
						nonBindable = elementOp.NonBindable
					} else if containerOp, ok := element.(*ops_create.ContainerOp); ok {
						nonBindable = containerOp.NonBindable
					}

					if nonBindable {
						enableOp := ops_create.NewEnableBindingsOp(createOp.GetXref())
						unit.GetCreate().InsertBefore(op, enableOp)
					}
				}
			}
		}
	}
}
