package phases

import (
	"fmt"

	ir "ngc-go/packages/compiler/template/pipeline/ir/src"
	ir_operation "ngc-go/packages/compiler/template/pipeline/ir/src/operations"
	ops_create "ngc-go/packages/compiler/template/pipeline/ir/src/ops/create"
	ops_update "ngc-go/packages/compiler/template/pipeline/ir/src/ops/update"
	"ngc-go/packages/compiler/util"

	pipeline "ngc-go/packages/compiler/template/pipeline/src/compilation"
)

// ConvertI18nBindings converts some binding instructions in the update block that may actually correspond to i18n bindings.
// In that case, they should be replaced with i18nExp instructions for the dynamic portions.
func ConvertI18nBindings(job *pipeline.CompilationJob) {
	i18nAttributesByElem := make(map[ir_operation.XrefId]*ops_create.I18nAttributesOp)
	for _, unit := range job.GetUnits() {
		for op := unit.GetCreate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			if op.GetKind() == ir.OpKindI18nAttributes {
				i18nAttributesOp, ok := op.(*ops_create.I18nAttributesOp)
				if ok {
					i18nAttributesByElem[i18nAttributesOp.Target] = i18nAttributesOp
				}
			}
		}

		for op := unit.GetUpdate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			switch op.GetKind() {
			case ir.OpKindProperty:
				propertyOp, ok := op.(*ops_update.PropertyOp)
				if !ok {
					continue
				}
				if propertyOp.I18nContext == 0 {
					continue
				}
				convertI18nBindingOpForProperty(unit, op, propertyOp, i18nAttributesByElem)
			case ir.OpKindAttribute:
				attributeOp, ok := op.(*ops_update.AttributeOp)
				if !ok {
					continue
				}
				if attributeOp.I18nContext == 0 {
					continue
				}
				convertI18nBindingOpForAttribute(unit, op, attributeOp, i18nAttributesByElem)
			}
		}
	}
}

func convertI18nBindingOpForProperty(
	unit pipeline.CompilationUnit,
	op ir_operation.Op,
	propertyOp *ops_update.PropertyOp,
	i18nAttributesByElem map[ir_operation.XrefId]*ops_create.I18nAttributesOp,
) {
	if propertyOp.I18nContext == 0 {
		return
	}

	// Check if expression is an Interpolation
	interpolation, ok := propertyOp.Expression.(*ops_update.Interpolation)
	if !ok {
		return
	}

	i18nAttributesForElem, exists := i18nAttributesByElem[propertyOp.Target]
	if !exists {
		panic("AssertionError: An i18n attribute binding instruction requires the owning element to have an I18nAttributes create instruction")
	}

	if i18nAttributesForElem.Target != propertyOp.Target {
		panic("AssertionError: Expected i18nAttributes target element to match binding target element")
	}

	var i18nExprOps []ir_operation.Op
	for i, expr := range interpolation.Expressions {
		if len(interpolation.I18nPlaceholders) != len(interpolation.Expressions) {
			panic(fmt.Sprintf(
				"AssertionError: An i18n attribute binding instruction requires the same number of expressions and placeholders, but found %d placeholders and %d expressions",
				len(interpolation.I18nPlaceholders),
				len(interpolation.Expressions),
			))
		}

		var i18nPlaceholder *string
		if i < len(interpolation.I18nPlaceholders) {
			placeholder := interpolation.I18nPlaceholders[i]
			i18nPlaceholder = &placeholder
		}

		var sourceSpan *util.ParseSourceSpan
		if expr.GetSourceSpan() != nil {
			sourceSpan = expr.GetSourceSpan()
		}
		i18nExprOp := ops_update.NewI18nExpressionOp(
			propertyOp.I18nContext,
			i18nAttributesForElem.Target,
			i18nAttributesForElem.Xref,
			i18nAttributesForElem.Handle,
			expr,
			nil, // icuPlaceholder
			i18nPlaceholder,
			ir.I18nParamResolutionTimeCreation,
			ir.I18nExpressionForI18nAttribute,
			propertyOp.Name,
			sourceSpan,
		)
		i18nExprOps = append(i18nExprOps, i18nExprOp)
	}

	// Replace the binding op with the i18n expression ops
	if len(i18nExprOps) > 0 {
		unit.GetUpdate().Replace(op, i18nExprOps[0])
		for i := 1; i < len(i18nExprOps); i++ {
			unit.GetUpdate().InsertAfter(i18nExprOps[i-1], i18nExprOps[i])
		}
	}
}

func convertI18nBindingOpForAttribute(
	unit pipeline.CompilationUnit,
	op ir_operation.Op,
	attributeOp *ops_update.AttributeOp,
	i18nAttributesByElem map[ir_operation.XrefId]*ops_create.I18nAttributesOp,
) {
	if attributeOp.I18nContext == 0 {
		return
	}

	// Check if expression is an Interpolation
	interpolation, ok := attributeOp.Expression.(*ops_update.Interpolation)
	if !ok {
		return
	}

	i18nAttributesForElem, exists := i18nAttributesByElem[attributeOp.Target]
	if !exists {
		panic("AssertionError: An i18n attribute binding instruction requires the owning element to have an I18nAttributes create instruction")
	}

	if i18nAttributesForElem.Target != attributeOp.Target {
		panic("AssertionError: Expected i18nAttributes target element to match binding target element")
	}

	var i18nExprOps []ir_operation.Op
	for i, expr := range interpolation.Expressions {
		if len(interpolation.I18nPlaceholders) != len(interpolation.Expressions) {
			panic(fmt.Sprintf(
				"AssertionError: An i18n attribute binding instruction requires the same number of expressions and placeholders, but found %d placeholders and %d expressions",
				len(interpolation.I18nPlaceholders),
				len(interpolation.Expressions),
			))
		}

		var i18nPlaceholder *string
		if i < len(interpolation.I18nPlaceholders) {
			placeholder := interpolation.I18nPlaceholders[i]
			i18nPlaceholder = &placeholder
		}

		var sourceSpan *util.ParseSourceSpan
		if expr.GetSourceSpan() != nil {
			sourceSpan = expr.GetSourceSpan()
		}
		i18nExprOp := ops_update.NewI18nExpressionOp(
			attributeOp.I18nContext,
			i18nAttributesForElem.Target,
			i18nAttributesForElem.Xref,
			i18nAttributesForElem.Handle,
			expr,
			nil, // icuPlaceholder
			i18nPlaceholder,
			ir.I18nParamResolutionTimeCreation,
			ir.I18nExpressionForI18nAttribute,
			attributeOp.Name,
			sourceSpan,
		)
		i18nExprOps = append(i18nExprOps, i18nExprOp)
	}

	// Replace the binding op with the i18n expression ops
	if len(i18nExprOps) > 0 {
		unit.GetUpdate().Replace(op, i18nExprOps[0])
		for i := 1; i < len(i18nExprOps); i++ {
			unit.GetUpdate().InsertAfter(i18nExprOps[i-1], i18nExprOps[i])
		}
	}
}
