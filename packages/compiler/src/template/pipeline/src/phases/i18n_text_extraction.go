package phases

import (
	"ngc-go/packages/compiler/src/template/pipeline/ir"
	ir_operation "ngc-go/packages/compiler/src/template/pipeline/ir/src/operations"
	ops_create "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/create"
	ops_update "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/update"
	"ngc-go/packages/compiler/src/util"

	pipeline "ngc-go/packages/compiler/src/template/pipeline/src/compilation"
)

// ConvertI18nText removes text nodes within i18n blocks since they are already hardcoded into the i18n message.
// Also, replaces interpolations on these text nodes with i18n expressions of the non-text portions,
// which will be applied later.
func ConvertI18nText(job *pipeline.CompilationJob) {
	for _, unit := range job.GetUnits() {
		// Remove all text nodes within i18n blocks, their content is already captured in the i18n
		// message.
		var currentI18n *ops_create.I18nStartOp = nil
		var currentIcu *ops_create.IcuStartOp = nil
		textNodeI18nBlocks := make(map[ir_operation.XrefId]*ops_create.I18nStartOp)
		textNodeIcus := make(map[ir_operation.XrefId]*ops_create.IcuStartOp)
		icuPlaceholderByText := make(map[ir_operation.XrefId]*ops_create.IcuPlaceholderOp)
		for op := unit.GetCreate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			switch op.GetKind() {
			case ir.OpKindI18nStart:
				i18nStartOp, ok := op.(*ops_create.I18nStartOp)
				if !ok {
					continue
				}
				if i18nStartOp.Context == 0 {
					panic("I18n op should have its context set.")
				}
				currentI18n = i18nStartOp
			case ir.OpKindI18nEnd:
				currentI18n = nil
			case ir.OpKindIcuStart:
				icuStartOp, ok := op.(*ops_create.IcuStartOp)
				if !ok {
					continue
				}
				if icuStartOp.Context == 0 {
					panic("Icu op should have its context set.")
				}
				currentIcu = icuStartOp
			case ir.OpKindIcuEnd:
				currentIcu = nil
			case ir.OpKindText:
				textOp, ok := op.(*ops_create.TextOp)
				if !ok {
					continue
				}
				if currentI18n != nil {
					textNodeI18nBlocks[textOp.Xref] = currentI18n
					if currentIcu != nil {
						textNodeIcus[textOp.Xref] = currentIcu
					}
					if textOp.IcuPlaceholder != nil {
						// Create an op to represent the ICU placeholder. Initially set its static text to the
						// value of the text op, though this may be overwritten later if this text op is a
						// placeholder for an interpolation.
						icuPlaceholderOp := ops_create.NewIcuPlaceholderOp(
							job.AllocateXrefId(),
							*textOp.IcuPlaceholder,
							[]string{textOp.InitialValue},
						)
						unit.GetCreate().Replace(op, icuPlaceholderOp)
						icuPlaceholderByText[textOp.Xref] = icuPlaceholderOp
					} else {
						// Otherwise just remove the text op, since its value is already accounted for in the
						// translated message.
						unit.GetCreate().Remove(op)
					}
				}
			}
		}

		// Update any interpolations to the removed text, and instead represent them as a series of i18n
		// expressions that we then apply.
		for op := unit.GetUpdate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			if op.GetKind() == ir.OpKindInterpolateText {
				interpolateTextOp, ok := op.(*ops_update.InterpolateTextOp)
				if !ok {
					continue
				}
				if _, exists := textNodeI18nBlocks[interpolateTextOp.Target]; !exists {
					continue
				}

				i18nOp := textNodeI18nBlocks[interpolateTextOp.Target]
				icuOp, hasIcu := textNodeIcus[interpolateTextOp.Target]
				icuPlaceholder, hasIcuPlaceholder := icuPlaceholderByText[interpolateTextOp.Target]
				contextId := i18nOp.Context
				if hasIcu {
					contextId = icuOp.Context
				}
				resolutionTime := ir.I18nParamResolutionTimeCreation
				if hasIcu {
					resolutionTime = ir.I18nParamResolutionTimePostprocessing
				}
				var i18nExprOps []ir_operation.Op
				for i, expr := range interpolateTextOp.Interpolation.Expressions {
					// For now, this i18nExpression depends on the slot context of the enclosing i18n block.
					// Later, we will modify this, and advance to a different point.
					var icuPlaceholderXref *ir_operation.XrefId
					if hasIcuPlaceholder {
						icuPlaceholderXref = &icuPlaceholder.Xref
					}
					var i18nPlaceholder *string
					if i < len(interpolateTextOp.Interpolation.I18nPlaceholders) {
						placeholder := interpolateTextOp.Interpolation.I18nPlaceholders[i]
						i18nPlaceholder = &placeholder
					}
					var sourceSpan *util.ParseSourceSpan
					if expr.GetSourceSpan() != nil {
						sourceSpan = expr.GetSourceSpan()
					} else {
						sourceSpan = interpolateTextOp.SourceSpan
					}
					i18nExprOp := ops_update.NewI18nExpressionOp(
						contextId,
						interpolateTextOp.Target,
						i18nOp.Xref,
						i18nOp.Handle,
						expr,
						icuPlaceholderXref,
						i18nPlaceholder,
						resolutionTime,
						ir.I18nExpressionForI18nText,
						"",
						sourceSpan,
					)
					i18nExprOps = append(i18nExprOps, i18nExprOp)
				}
				// Replace the interpolate text op with the i18n expression ops
				if len(i18nExprOps) > 0 {
					unit.GetUpdate().Replace(op, i18nExprOps[0])
					for i := 1; i < len(i18nExprOps); i++ {
						unit.GetUpdate().InsertAfter(i18nExprOps[i-1], i18nExprOps[i])
					}
				}
				// If this interpolation is part of an ICU placeholder, add the strings and expressions to
				// the placeholder.
				if hasIcuPlaceholder {
					icuPlaceholder.Strings = interpolateTextOp.Interpolation.Strings
				}
			}
		}
	}
}
