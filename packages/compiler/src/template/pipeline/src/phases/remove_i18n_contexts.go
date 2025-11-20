package phases

import (
	"ngc-go/packages/compiler/src/template/pipeline/ir"
	ops_create "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/create"

	pipeline "ngc-go/packages/compiler/src/template/pipeline/src/compilation"
)

// RemoveI18nContexts removes the i18n context ops after they are no longer needed, and null out references to them to
// be safe.
func RemoveI18nContexts(job *pipeline.CompilationJob) {
	for _, unit := range job.GetUnits() {
		for op := unit.GetCreate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			switch op.GetKind() {
			case ir.OpKindI18nContext:
				unit.GetCreate().Remove(op)
			case ir.OpKindI18nStart:
				i18nStartOp, ok := op.(*ops_create.I18nStartOp)
				if ok {
					i18nStartOp.Context = 0
				}
			}
		}
	}
}
