package phases

import (
	ir "ngc-go/packages/compiler/template/pipeline/ir/src"
	ir_operations "ngc-go/packages/compiler/template/pipeline/ir/src/operations"
	ops_create "ngc-go/packages/compiler/template/pipeline/ir/src/ops/create"
	ops_update "ngc-go/packages/compiler/template/pipeline/ir/src/ops/update"

	pipeline "ngc-go/packages/compiler/template/pipeline/src/compilation"
)

// ResolveI18nExpressionPlaceholders resolves the i18n expression placeholders in i18n messages.
func ResolveI18nExpressionPlaceholders(job *pipeline.ComponentCompilationJob) {
	// Record all of the i18n context ops, and the sub-template index for each i18n op.
	subTemplateIndices := make(map[ir_operations.XrefId]*int)
	i18nContexts := make(map[ir_operations.XrefId]*ops_create.I18nContextOp)
	icuPlaceholders := make(map[ir_operations.XrefId]*ops_create.IcuPlaceholderOp)
	for _, unit := range job.GetUnits() {
		for op := unit.GetCreate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			switch op.GetKind() {
			case ir.OpKindI18nStart:
				i18nStartOp, ok := op.(*ops_create.I18nStartOp)
				if ok {
					subTemplateIndices[i18nStartOp.Xref] = i18nStartOp.SubTemplateIndex
				}
			case ir.OpKindI18nContext:
				i18nContextOp, ok := op.(*ops_create.I18nContextOp)
				if ok {
					i18nContexts[i18nContextOp.Xref] = i18nContextOp
				}
			case ir.OpKindIcuPlaceholder:
				icuPlaceholderOp, ok := op.(*ops_create.IcuPlaceholderOp)
				if ok {
					icuPlaceholders[icuPlaceholderOp.Xref] = icuPlaceholderOp
				}
			}
		}
	}

	// Keep track of the next available expression index for each i18n message.
	expressionIndices := make(map[ir_operations.XrefId]int)

	// Keep track of a reference index for each expression.
	// We use different references for normal i18n expression and attribute i18n expressions. This is
	// because child i18n blocks in templates don't get their own context, since they're rolled into
	// the translated message of the parent, but they may target a different slot.
	referenceIndex := func(op *ops_update.I18nExpressionOp) ir_operations.XrefId {
		if op.Usage == ir.I18nExpressionForI18nText {
			return op.I18nOwner
		}
		return op.Context
	}

	for _, unit := range job.GetUnits() {
		for op := unit.GetUpdate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			if op.GetKind() == ir.OpKindI18nExpression {
				i18nExprOp, ok := op.(*ops_update.I18nExpressionOp)
				if !ok {
					continue
				}
				refIndex := referenceIndex(i18nExprOp)
				index := expressionIndices[refIndex]
				subTemplateIndex := subTemplateIndices[i18nExprOp.I18nOwner]
				value := ops_create.I18nParamValue{
					Value:            index,
					SubTemplateIndex: subTemplateIndex,
					Flags:            ir.I18nParamValueFlagsExpressionIndex,
				}
				updatePlaceholder(i18nExprOp, value, i18nContexts, icuPlaceholders)
				expressionIndices[refIndex] = index + 1
			}
		}
	}
}

func updatePlaceholder(
	op *ops_update.I18nExpressionOp,
	value ops_create.I18nParamValue,
	i18nContexts map[ir_operations.XrefId]*ops_create.I18nContextOp,
	icuPlaceholders map[ir_operations.XrefId]*ops_create.IcuPlaceholderOp,
) {
	if op.I18nPlaceholder != nil {
		i18nContext, exists := i18nContexts[op.Context]
		if !exists {
			return
		}
		var params map[string][]ops_create.I18nParamValue
		if op.ResolutionTime == ir.I18nParamResolutionTimeCreation {
			params = i18nContext.Params
		} else {
			params = i18nContext.PostprocessingParams
		}
		values := params[*op.I18nPlaceholder]
		values = append(values, value)
		params[*op.I18nPlaceholder] = values
	}
	if op.IcuPlaceholder != nil {
		icuPlaceholderOp, exists := icuPlaceholders[*op.IcuPlaceholder]
		if exists {
			icuPlaceholderOp.ExpressionPlaceholders = append(icuPlaceholderOp.ExpressionPlaceholders, value)
		}
	}
}
