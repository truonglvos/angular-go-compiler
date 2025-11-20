package phases

import (
	"ngc-go/packages/compiler/src/template/pipeline/ir"
	ops_create "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/create"

	pipeline "ngc-go/packages/compiler/src/template/pipeline/src/compilation"
)

// EmitNamespaceChanges changes namespaces between HTML, SVG and MathML, depending on the next element.
func EmitNamespaceChanges(job *pipeline.CompilationJob) {
	for _, unit := range job.GetUnits() {
		activeNamespace := ir.NamespaceHTML

		for op := unit.GetCreate().Head(); op != nil; op = op.Next() {
			if elementStart, ok := op.(*ops_create.ElementStartOp); ok {
				if elementStart.Namespace != activeNamespace {
					namespaceOp := ops_create.NewNamespaceOp(elementStart.Namespace)
					unit.GetCreate().InsertBefore(op, namespaceOp)
					activeNamespace = elementStart.Namespace
				}
			}
		}
	}
}
