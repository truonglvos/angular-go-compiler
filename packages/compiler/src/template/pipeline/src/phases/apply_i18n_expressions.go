package phases

import (
	"ngc-go/packages/compiler/src/template/pipeline/ir"
	ir_operations "ngc-go/packages/compiler/src/template/pipeline/ir/src/operations"
	ops_create "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/create"
	ops_update "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/update"

	pipeline_compilation "ngc-go/packages/compiler/src/template/pipeline/src/compilation"
)

// ApplyI18nExpressions adds apply operations after i18n expressions.
func ApplyI18nExpressions(job *pipeline_compilation.CompilationJob) {
	i18nContexts := make(map[ir_operations.XrefId]*ops_create.I18nContextOp)
	for _, unit := range job.GetUnits() {
		for op := unit.GetCreate().Head(); op != nil; op = op.Next() {
			if i18nContext, ok := op.(*ops_create.I18nContextOp); ok {
				i18nContexts[i18nContext.Xref] = i18nContext
			}
		}
	}

	for _, unit := range job.GetUnits() {
		for op := unit.GetUpdate().Head(); op != nil; op = op.Next() {
			// Only add apply after expressions that are not followed by more expressions.
			if i18nExprOp, ok := op.(*ops_update.I18nExpressionOp); ok {
				if needsApplication(i18nContexts, i18nExprOp, unit.GetUpdate()) {
					// TODO: what should be the source span for the apply op?
					applyOp := ops_update.NewI18nApplyOp(i18nExprOp.I18nOwner, i18nExprOp.Handle, nil)
					unit.GetUpdate().InsertAfter(op, applyOp)
				}
			}
		}
	}
}

// needsApplication checks whether the given expression op needs to be followed with an apply op.
func needsApplication(
	i18nContexts map[ir_operations.XrefId]*ops_create.I18nContextOp,
	op *ops_update.I18nExpressionOp,
	updateList *ir_operations.OpList,
) bool {
	// If the next op is not another expression, we need to apply.
	nextOp := op.Next()
	if nextOp == nil || nextOp.GetKind() != ir.OpKindI18nExpression {
		return true
	}

	nextI18nExpr, ok := nextOp.(*ops_update.I18nExpressionOp)
	if !ok {
		return true
	}

	context, ok := i18nContexts[op.Context]
	if !ok {
		panic("AssertionError: expected an I18nContextOp to exist for the I18nExpressionOp's context")
	}

	nextContext, ok := i18nContexts[nextI18nExpr.Context]
	if !ok {
		panic("AssertionError: expected an I18nContextOp to exist for the next I18nExpressionOp's context")
	}

	// If the next op is an expression targeting a different i18n block (or different element, in the
	// case of i18n attributes), we need to apply.

	// First, handle the case of i18n blocks.
	if context.I18nBlock != 0 {
		// This is a block context. Compare the blocks.
		if context.I18nBlock != nextContext.I18nBlock {
			return true
		}
		return false
	}

	// Second, handle the case of i18n attributes.
	if op.I18nOwner != nextI18nExpr.I18nOwner {
		return true
	}
	return false
}
