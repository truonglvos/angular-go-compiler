package phases

import (
	"ngc-go/packages/compiler/src/template/pipeline/ir"
	ir_operation "ngc-go/packages/compiler/src/template/pipeline/ir/src/operations"
	ops_create "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/create"
	ops_update "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/update"

	pipeline "ngc-go/packages/compiler/src/template/pipeline/src/compilation"
)

// RemoveUnusedI18nAttributesOps removes i18nAttributes ops that are not needed.
// i18nAttributes ops will be generated for each i18n attribute. However, not all i18n attributes
// will contain dynamic content, and so some of these i18nAttributes ops may be unnecessary.
func RemoveUnusedI18nAttributesOps(job *pipeline.CompilationJob) {
	for _, unit := range job.GetUnits() {
		ownersWithI18nExpressions := make(map[ir_operation.XrefId]bool)

		for op := unit.GetUpdate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			if op.GetKind() == ir.OpKindI18nExpression {
				i18nExprOp, ok := op.(*ops_update.I18nExpressionOp)
				if ok {
					ownersWithI18nExpressions[i18nExprOp.I18nOwner] = true
				}
			}
		}

		for op := unit.GetCreate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			if op.GetKind() == ir.OpKindI18nAttributes {
				i18nAttributesOp, ok := op.(*ops_create.I18nAttributesOp)
				if ok {
					if !ownersWithI18nExpressions[i18nAttributesOp.Xref] {
						unit.GetCreate().Remove(op)
					}
				}
			}
		}
	}
}
