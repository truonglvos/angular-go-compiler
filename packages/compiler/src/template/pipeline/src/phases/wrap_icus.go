package phases

import (
	ir "ngc-go/packages/compiler/src/template/pipeline/ir/src"
	ir_operation "ngc-go/packages/compiler/src/template/pipeline/ir/src/operations"
	ops_create "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/create"

	pipeline "ngc-go/packages/compiler/src/template/pipeline/src/compilation"
)

// WrapI18nIcus wraps ICUs that do not already belong to an i18n block in a new i18n block.
func WrapI18nIcus(job *pipeline.CompilationJob) {
	for _, unit := range job.GetUnits() {
		var currentI18nOp *ops_create.I18nStartOp = nil
		var addedI18nId ir_operation.XrefId = 0
		hasAddedI18nId := false

		for op := unit.GetCreate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			switch op.GetKind() {
			case ir.OpKindI18nStart:
				if i18nStartOp, ok := op.(*ops_create.I18nStartOp); ok {
					currentI18nOp = i18nStartOp
				}
			case ir.OpKindI18nEnd:
				currentI18nOp = nil
			case ir.OpKindIcuStart:
				if currentI18nOp == nil {
					addedI18nId = job.AllocateXrefId()
					hasAddedI18nId = true
					icuStartOp, ok := op.(*ops_create.IcuStartOp)
					if !ok {
						continue
					}
					// ICU i18n start/end ops_create should not receive source spans.
					i18nStartOp := ops_create.NewI18nStartOp(addedI18nId, icuStartOp.Message, 0, nil)
					unit.GetCreate().InsertBefore(op, i18nStartOp)
				}
			case ir.OpKindIcuEnd:
				if hasAddedI18nId {
					i18nEndOp := ops_create.NewI18nEndOp(addedI18nId, nil)
					unit.GetCreate().InsertAfter(op, i18nEndOp)
					hasAddedI18nId = false
					addedI18nId = 0
				}
			}
		}
	}
}
