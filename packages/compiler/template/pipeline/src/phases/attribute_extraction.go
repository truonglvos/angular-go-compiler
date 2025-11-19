package phases

import (
	"fmt"

	"ngc-go/packages/compiler/core"
	"ngc-go/packages/compiler/output"
	ir "ngc-go/packages/compiler/template/pipeline/ir/src"
	ir_expression "ngc-go/packages/compiler/template/pipeline/ir/src/expression"
	ir_operation "ngc-go/packages/compiler/template/pipeline/ir/src/operations"
	ops_create "ngc-go/packages/compiler/template/pipeline/ir/src/ops/create"
	ops_update "ngc-go/packages/compiler/template/pipeline/ir/src/ops/update"

	pipeline "ngc-go/packages/compiler/template/pipeline/src/compilation"
	pipeline_util "ngc-go/packages/compiler/template/pipeline/src/util"
)

// ExtractAttributes finds all extractable attribute and binding ops, and creates ExtractedAttributeOps for them.
// In cases where no instruction needs to be generated for the attribute or binding, it is removed.
func ExtractAttributes(job *pipeline.CompilationJob) {
	for _, unit := range job.GetUnits() {
		elements := pipeline_util.CreateOpXrefMap(unit)
		for op := unit.GetCreate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			switch op.GetKind() {
			case ir.OpKindAttribute:
				extractAttributeOp(unit, op, elements, job)
			case ir.OpKindProperty:
				propertyOp, ok := op.(*ops_update.PropertyOp)
				if !ok {
					continue
				}
				// Check if this is a legacy animation or animation binding
				// For now, we'll skip animation bindings
				// TODO: Handle animation bindings properly

				var bindingKind ir.BindingKind
				// Check if this has i18n context
				// For PropertyOp, we need to check if it has i18n message
				// Since PropertyOp doesn't have i18n fields directly, we'll use Property binding kind
				// TODO: Check i18n context properly
				bindingKind = ir.BindingKindProperty

				extractedAttrOp := ops_create.NewExtractedAttributeOp(
					propertyOp.Target,
					bindingKind,
					nil, // namespace
					propertyOp.Name,
					nil,                      // expression
					0,                        // i18nContext
					nil,                      // i18nMessage
					core.SecurityContextNONE, // securityContext
				)
				targetOp := lookupElement(elements, propertyOp.Target)
				unit.GetCreate().InsertBefore(targetOp, extractedAttrOp)
			case ir.OpKindControl:
				controlOp, ok := op.(*ops_update.ControlOp)
				if !ok {
					continue
				}
				extractedAttrOp := ops_create.NewExtractedAttributeOp(
					controlOp.Target,
					ir.BindingKindProperty,
					nil, // namespace
					"field",
					nil,                      // expression
					0,                        // i18nContext
					nil,                      // i18nMessage
					core.SecurityContextNONE, // securityContext
				)
				targetOp := lookupElement(elements, controlOp.Target)
				unit.GetCreate().InsertBefore(targetOp, extractedAttrOp)
			case ir.OpKindTwoWayProperty:
				twoWayOp, ok := op.(*ops_update.TwoWayPropertyOp)
				if !ok {
					continue
				}
				extractedAttrOp := ops_create.NewExtractedAttributeOp(
					twoWayOp.Target,
					ir.BindingKindTwoWayProperty,
					nil, // namespace
					twoWayOp.Name,
					nil,                      // expression
					0,                        // i18nContext
					nil,                      // i18nMessage
					core.SecurityContextNONE, // securityContext
				)
				targetOp := lookupElement(elements, twoWayOp.Target)
				unit.GetCreate().InsertBefore(targetOp, extractedAttrOp)
			case ir.OpKindStyleProp, ir.OpKindClassProp:
				// TODO: Can style or class bindings be i18n attributes?
				// The old compiler treated empty style bindings as regular bindings for the purpose of
				// directive matching. That behavior is incorrect, but we emulate it in compatibility
				// mode.
				if job.Compatibility == ir.CompatibilityModeTemplateDefinitionBuilder {
					var name string
					var target ir_operation.XrefId
					if stylePropOp, ok := op.(*ops_update.StylePropOp); ok {
						name = stylePropOp.Name
						target = stylePropOp.Target
						// Check if expression is EmptyExpr
						if emptyExpr, ok := stylePropOp.Expression.(*ir_expression.EmptyExpr); ok && emptyExpr != nil {
							extractedAttrOp := ops_create.NewExtractedAttributeOp(
								target,
								ir.BindingKindProperty,
								nil, // namespace
								name,
								nil,                       // expression
								0,                         // i18nContext
								nil,                       // i18nMessage
								core.SecurityContextSTYLE, // securityContext
							)
							targetOp := lookupElement(elements, target)
							unit.GetCreate().InsertBefore(targetOp, extractedAttrOp)
						}
					} else if classPropOp, ok := op.(*ops_update.ClassPropOp); ok {
						name = classPropOp.Name
						target = classPropOp.Target
						// Check if expression is EmptyExpr
						if emptyExpr, ok := classPropOp.Expression.(*ir_expression.EmptyExpr); ok && emptyExpr != nil {
							extractedAttrOp := ops_create.NewExtractedAttributeOp(
								target,
								ir.BindingKindProperty,
								nil, // namespace
								name,
								nil,                      // expression
								0,                        // i18nContext
								nil,                      // i18nMessage
								core.SecurityContextNONE, // securityContext
							)
							targetOp := lookupElement(elements, target)
							unit.GetCreate().InsertBefore(targetOp, extractedAttrOp)
						}
					}
				}
			case ir.OpKindListener:
				listenerOp, ok := op.(*ops_create.ListenerOp)
				if !ok {
					continue
				}
				// Check if this is a legacy animation listener
				// TODO: Check IsLegacyAnimationListener properly
				if false { // !listenerOp.IsLegacyAnimationListener
					extractedAttrOp := ops_create.NewExtractedAttributeOp(
						listenerOp.Target,
						ir.BindingKindProperty,
						nil, // namespace
						listenerOp.Name,
						nil,                      // expression
						0,                        // i18nContext
						nil,                      // i18nMessage
						core.SecurityContextNONE, // securityContext
					)
					if job.Kind == pipeline.CompilationJobKindHost {
						if job.Compatibility == ir.CompatibilityModeTemplateDefinitionBuilder {
							// TemplateDefinitionBuilder does not extract listener bindings to the const array
							// (which is honestly pretty inconsistent).
							break
						}
						// This attribute will apply to the enclosing host binding compilation unit, so order
						// doesn't matter.
						unit.GetCreate().Push(extractedAttrOp)
					} else {
						targetOp := lookupElement(elements, listenerOp.Target)
						unit.GetCreate().InsertBefore(targetOp, extractedAttrOp)
					}
				}
			case ir.OpKindTwoWayListener:
				twoWayListenerOp, ok := op.(*ops_create.TwoWayListenerOp)
				if !ok {
					continue
				}
				// Two-way listeners aren't supported in host bindings.
				if job.Kind != pipeline.CompilationJobKindHost {
					extractedAttrOp := ops_create.NewExtractedAttributeOp(
						twoWayListenerOp.Target,
						ir.BindingKindProperty,
						nil, // namespace
						twoWayListenerOp.Name,
						nil,                      // expression
						0,                        // i18nContext
						nil,                      // i18nMessage
						core.SecurityContextNONE, // securityContext
					)
					targetOp := lookupElement(elements, twoWayListenerOp.Target)
					unit.GetCreate().InsertBefore(targetOp, extractedAttrOp)
				}
			}
		}
		for op := unit.GetUpdate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			switch op.GetKind() {
			case ir.OpKindAttribute:
				extractAttributeOp(unit, op, elements, job)
			case ir.OpKindProperty:
				propertyOp, ok := op.(*ops_update.PropertyOp)
				if !ok {
					continue
				}
				var bindingKind ir.BindingKind
				bindingKind = ir.BindingKindProperty

				extractedAttrOp := ops_create.NewExtractedAttributeOp(
					propertyOp.Target,
					bindingKind,
					nil, // namespace
					propertyOp.Name,
					nil,                      // expression
					0,                        // i18nContext
					nil,                      // i18nMessage
					core.SecurityContextNONE, // securityContext
				)
				targetOp := lookupElement(elements, propertyOp.Target)
				unit.GetCreate().InsertBefore(targetOp, extractedAttrOp)
			case ir.OpKindControl:
				controlOp, ok := op.(*ops_update.ControlOp)
				if !ok {
					continue
				}
				extractedAttrOp := ops_create.NewExtractedAttributeOp(
					controlOp.Target,
					ir.BindingKindProperty,
					nil, // namespace
					"field",
					nil,                      // expression
					0,                        // i18nContext
					nil,                      // i18nMessage
					core.SecurityContextNONE, // securityContext
				)
				targetOp := lookupElement(elements, controlOp.Target)
				unit.GetCreate().InsertBefore(targetOp, extractedAttrOp)
			case ir.OpKindTwoWayProperty:
				twoWayOp, ok := op.(*ops_update.TwoWayPropertyOp)
				if !ok {
					continue
				}
				extractedAttrOp := ops_create.NewExtractedAttributeOp(
					twoWayOp.Target,
					ir.BindingKindTwoWayProperty,
					nil, // namespace
					twoWayOp.Name,
					nil,                      // expression
					0,                        // i18nContext
					nil,                      // i18nMessage
					core.SecurityContextNONE, // securityContext
				)
				targetOp := lookupElement(elements, twoWayOp.Target)
				unit.GetCreate().InsertBefore(targetOp, extractedAttrOp)
			case ir.OpKindStyleProp, ir.OpKindClassProp:
				if job.Compatibility == ir.CompatibilityModeTemplateDefinitionBuilder {
					var name string
					var target ir_operation.XrefId
					if stylePropOp, ok := op.(*ops_update.StylePropOp); ok {
						name = stylePropOp.Name
						target = stylePropOp.Target
						if emptyExpr, ok := stylePropOp.Expression.(*ir_expression.EmptyExpr); ok && emptyExpr != nil {
							extractedAttrOp := ops_create.NewExtractedAttributeOp(
								target,
								ir.BindingKindProperty,
								nil, // namespace
								name,
								nil,                       // expression
								0,                         // i18nContext
								nil,                       // i18nMessage
								core.SecurityContextSTYLE, // securityContext
							)
							targetOp := lookupElement(elements, target)
							unit.GetCreate().InsertBefore(targetOp, extractedAttrOp)
						}
					} else if classPropOp, ok := op.(*ops_update.ClassPropOp); ok {
						name = classPropOp.Name
						target = classPropOp.Target
						if emptyExpr, ok := classPropOp.Expression.(*ir_expression.EmptyExpr); ok && emptyExpr != nil {
							extractedAttrOp := ops_create.NewExtractedAttributeOp(
								target,
								ir.BindingKindProperty,
								nil, // namespace
								name,
								nil,                      // expression
								0,                        // i18nContext
								nil,                      // i18nMessage
								core.SecurityContextNONE, // securityContext
							)
							targetOp := lookupElement(elements, target)
							unit.GetCreate().InsertBefore(targetOp, extractedAttrOp)
						}
					}
				}
			case ir.OpKindListener:
				listenerOp, ok := op.(*ops_create.ListenerOp)
				if !ok {
					continue
				}
				if false { // !listenerOp.IsLegacyAnimationListener
					extractedAttrOp := ops_create.NewExtractedAttributeOp(
						listenerOp.Target,
						ir.BindingKindProperty,
						nil, // namespace
						listenerOp.Name,
						nil,                      // expression
						0,                        // i18nContext
						nil,                      // i18nMessage
						core.SecurityContextNONE, // securityContext
					)
					if job.Kind == pipeline.CompilationJobKindHost {
						if job.Compatibility == ir.CompatibilityModeTemplateDefinitionBuilder {
							break
						}
						unit.GetCreate().Push(extractedAttrOp)
					} else {
						targetOp := lookupElement(elements, listenerOp.Target)
						unit.GetCreate().InsertBefore(targetOp, extractedAttrOp)
					}
				}
			case ir.OpKindTwoWayListener:
				twoWayListenerOp, ok := op.(*ops_create.TwoWayListenerOp)
				if !ok {
					continue
				}
				if job.Kind != pipeline.CompilationJobKindHost {
					extractedAttrOp := ops_create.NewExtractedAttributeOp(
						twoWayListenerOp.Target,
						ir.BindingKindProperty,
						nil, // namespace
						twoWayListenerOp.Name,
						nil,                      // expression
						0,                        // i18nContext
						nil,                      // i18nMessage
						core.SecurityContextNONE, // securityContext
					)
					targetOp := lookupElement(elements, twoWayListenerOp.Target)
					unit.GetCreate().InsertBefore(targetOp, extractedAttrOp)
				}
			}
		}
	}
}

// lookupElement looks up an element in the given map by xref ID.
func lookupElement(
	elements map[ir_operation.XrefId]pipeline_util.OpXrefMapEntry,
	xref ir_operation.XrefId,
) pipeline_util.OpXrefMapEntry {
	el, exists := elements[xref]
	if !exists {
		panic(fmt.Sprintf("All attributes should have an element-like target: %d", xref))
	}
	return el
}

// extractAttributeOp extracts an attribute binding.
func extractAttributeOp(
	unit pipeline.CompilationUnit,
	op ir_operation.Op,
	elements map[ir_operation.XrefId]pipeline_util.OpXrefMapEntry,
	job *pipeline.CompilationJob,
) {
	attributeOp, ok := op.(*ops_update.AttributeOp)
	if !ok {
		return
	}

	// Check if expression is an Interpolation
	if _, ok := attributeOp.Expression.(*ops_update.Interpolation); ok {
		return
	}

	// Check if expression is constant
	expr, ok := attributeOp.Expression.(output.OutputExpression)
	if !ok {
		return
	}

	extractable := attributeOp.IsTextAttribute || expr.IsConstant()
	if job.Compatibility == ir.CompatibilityModeTemplateDefinitionBuilder {
		// TemplateDefinitionBuilder only extracts text attributes. It does not extract attribute
		// bindings, even if they are constants.
		extractable = extractable && attributeOp.IsTextAttribute
	}

	if extractable {
		var bindingKind ir.BindingKind
		if attributeOp.IsStructuralTemplateAttribute {
			bindingKind = ir.BindingKindTemplate
		} else {
			bindingKind = ir.BindingKindAttribute
		}

		extractedAttrOp := ops_create.NewExtractedAttributeOp(
			attributeOp.Target,
			bindingKind,
			nil, // namespace - TODO: get from attributeOp
			attributeOp.Name,
			expr,
			0,                        // i18nContext - TODO: get from attributeOp
			nil,                      // i18nMessage - TODO: get from attributeOp
			core.SecurityContextNONE, // securityContext - TODO: get from attributeOp
		)

		if job.Kind == pipeline.CompilationJobKindHost {
			// This attribute will apply to the enclosing host binding compilation unit, so order doesn't
			// matter.
			unit.GetCreate().Push(extractedAttrOp)
		} else {
			ownerOp := lookupElement(elements, attributeOp.Target)
			unit.GetCreate().InsertBefore(ownerOp, extractedAttrOp)
		}
		unit.GetUpdate().Remove(op)
	}
}
