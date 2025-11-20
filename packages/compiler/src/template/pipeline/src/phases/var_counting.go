package phases

import (
	"fmt"

	"ngc-go/packages/compiler/src/output"
	"ngc-go/packages/compiler/src/template/pipeline/ir"
	"ngc-go/packages/compiler/src/template/pipeline/ir/src/expression"
	ir_operations "ngc-go/packages/compiler/src/template/pipeline/ir/src/operations"
	ops_create "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/create"
	ops_update "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/update"
	ir_traits "ngc-go/packages/compiler/src/template/pipeline/ir/src/traits"

	pipeline_compilation "ngc-go/packages/compiler/src/template/pipeline/src/compilation"
)

// CountVariables counts the number of variable slots used within each view, and stores that on the view itself, as
// well as propagates it to the `ir.TemplateOp` for embedded views.
func CountVariables(job *pipeline_compilation.CompilationJob) {
	// First, count the vars used in each view, and update the view-level counter.
	for _, unit := range job.GetUnits() {
		varCount := 0

		// Count variables on top-level ops first. Don't explore nested expressions just yet.
		for op := unit.GetCreate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			if ir_traits.HasConsumesVarsTrait(op) {
				varCount += varsUsedByOp(op)
			}
		}
		for op := unit.GetUpdate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			if ir_traits.HasConsumesVarsTrait(op) {
				varCount += varsUsedByOp(op)
			}
		}

		// Count variables on expressions inside ops. We do this later because some of these expressions
		// might be conditional (e.g. `pipeBinding` inside of a ternary), and we don't want to interfere
		// with indices for top-level binding slots (e.g. `property`).
		for op := unit.GetCreate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			expression.VisitExpressionsInOp(op, func(expr output.OutputExpression, flags expression.VisitorContextFlag) {
				if !expression.IsIrExpression(expr) {
					return
				}

				// TemplateDefinitionBuilder assigns variable offsets for everything but pure functions
				// first, and then assigns offsets to pure functions lazily. We emulate that behavior by
				// assigning offsets in two passes instead of one, only in compatibility mode.
				if job.Compatibility == ir.CompatibilityModeTemplateDefinitionBuilder {
					if pureFunc, ok := expr.(*expression.PureFunctionExpr); ok {
						_ = pureFunc
						return
					}
				}

				// Some expressions require knowledge of the number of variable slots consumed.
				if usesVarOffset, ok := expr.(ir_traits.UsesVarOffsetTraitInterface); ok {
					usesVarOffset.SetVarOffset(varCount)
				}

				if ir_traits.HasConsumesVarsTrait(expr) {
					varCount += varsUsedByIrExpression(expr)
				}
			})
		}
		for op := unit.GetUpdate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			expression.VisitExpressionsInOp(op, func(expr output.OutputExpression, flags expression.VisitorContextFlag) {
				if !expression.IsIrExpression(expr) {
					return
				}

				// TemplateDefinitionBuilder assigns variable offsets for everything but pure functions
				// first, and then assigns offsets to pure functions lazily. We emulate that behavior by
				// assigning offsets in two passes instead of one, only in compatibility mode.
				if job.Compatibility == ir.CompatibilityModeTemplateDefinitionBuilder {
					if pureFunc, ok := expr.(*expression.PureFunctionExpr); ok {
						_ = pureFunc
						return
					}
				}

				// Some expressions require knowledge of the number of variable slots consumed.
				if usesVarOffset, ok := expr.(ir_traits.UsesVarOffsetTraitInterface); ok {
					usesVarOffset.SetVarOffset(varCount)
				}

				if ir_traits.HasConsumesVarsTrait(expr) {
					varCount += varsUsedByIrExpression(expr)
				}
			})
		}

		// Compatibility mode pass for pure function offsets (as explained above).
		if job.Compatibility == ir.CompatibilityModeTemplateDefinitionBuilder {
			for op := unit.GetCreate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
				expression.VisitExpressionsInOp(op, func(expr output.OutputExpression, flags expression.VisitorContextFlag) {
					if !expression.IsIrExpression(expr) {
						return
					}
					pureFunc, ok := expr.(*expression.PureFunctionExpr)
					if !ok {
						return
					}

					// Some expressions require knowledge of the number of variable slots consumed.
					if usesVarOffset, ok := expr.(ir_traits.UsesVarOffsetTraitInterface); ok {
						usesVarOffset.SetVarOffset(varCount)
					}

					if ir_traits.HasConsumesVarsTrait(expr) {
						varCount += varsUsedByIrExpression(expr)
					}
					_ = pureFunc
				})
			}
			for op := unit.GetUpdate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
				expression.VisitExpressionsInOp(op, func(expr output.OutputExpression, flags expression.VisitorContextFlag) {
					if !expression.IsIrExpression(expr) {
						return
					}
					pureFunc, ok := expr.(*expression.PureFunctionExpr)
					if !ok {
						return
					}

					// Some expressions require knowledge of the number of variable slots consumed.
					if usesVarOffset, ok := expr.(ir_traits.UsesVarOffsetTraitInterface); ok {
						usesVarOffset.SetVarOffset(varCount)
					}

					if ir_traits.HasConsumesVarsTrait(expr) {
						varCount += varsUsedByIrExpression(expr)
					}
					_ = pureFunc
				})
			}
		}

		unit.SetVars(varCount)
	}

	// Check if this is a ComponentCompilationJob by checking the Kind
	// ComponentCompilationJob has Kind = CompilationJobKindTmpl
	if job.Kind == pipeline_compilation.CompilationJobKindTmpl {
		// We need to access Views, which is only available on ComponentCompilationJob
		// Since job is *CompilationJob, we need to get the actual ComponentCompilationJob
		// We can do this by getting the root unit and accessing its Job field
		root := job.GetRoot()
		if root != nil {
			viewUnit, ok := root.(*pipeline_compilation.ViewCompilationUnit)
			if ok && viewUnit.Job != nil {
				componentJob := viewUnit.Job
				// Add var counts for each view to the `ir.TemplateOp` which declares that view (if the view is
				// an embedded view).
				for _, unit := range componentJob.GetUnits() {
					for op := unit.GetCreate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
						kind := op.GetKind()
						if kind != ir.OpKindTemplate && kind != ir.OpKindRepeaterCreate &&
							kind != ir.OpKindConditionalCreate && kind != ir.OpKindConditionalBranchCreate {
							continue
						}

						childView := componentJob.Views[op.(ir_operations.CreateOp).GetXref()]
						if childView != nil && childView.GetVars() != nil {
							vars := childView.GetVars()
							if templateOp, ok := op.(*ops_create.TemplateOp); ok {
								templateOp.Vars = vars
							} else if repeaterOp, ok := op.(*ops_create.RepeaterCreateOp); ok {
								repeaterOp.Vars = vars
								// TODO: currently we handle the vars for the RepeaterCreate empty template in the reify
								// phase. We should handle that here instead.
							} else if conditionalOp, ok := op.(*ops_create.ConditionalCreateOp); ok {
								conditionalOp.Vars = vars
							} else if branchOp, ok := op.(*ops_create.ConditionalBranchCreateOp); ok {
								branchOp.Vars = vars
							}
						}
					}
				}
			}
		}
	}
}

// varsUsedByOp counts the variables used by any particular `op`.
// Different operations that implement `ir.UsesVarsTrait` use different numbers of variables.
func varsUsedByOp(op ir_operations.Op) int {
	kind := op.GetKind()
	switch kind {
	case ir.OpKindAttribute:
		// All of these bindings use 1 variable slot, plus 1 slot for every interpolated expression,
		// if any.
		slots := 1
		if attributeOp, ok := op.(*ops_update.AttributeOp); ok {
			if interpolation, ok := attributeOp.Expression.(*ops_update.Interpolation); ok {
				if !isSingletonInterpolationv2(interpolation) {
					slots += len(interpolation.Expressions)
				}
			}
		}
		return slots
	case ir.OpKindProperty, ir.OpKindDomProperty:
		slots := 1

		// We need to assign a slot even for singleton interpolations, because the
		// runtime needs to store both the raw value and the stringified one.
		if propertyOp, ok := op.(*ops_update.PropertyOp); ok {
			if interpolation, ok := propertyOp.Expression.(*ops_update.Interpolation); ok {
				slots += len(interpolation.Expressions)
			}
		} else if domPropertyOp, ok := op.(*ops_update.DomPropertyOp); ok {
			if interpolation, ok := domPropertyOp.Expression.(*ops_update.Interpolation); ok {
				slots += len(interpolation.Expressions)
			}
		}
		return slots
	case ir.OpKindControl:
		// 1 for the [field] binding itself.
		// 1 for the control bindings object containing bound field states properties.
		return 2
	case ir.OpKindTwoWayProperty:
		// Two-way properties can only have expressions so they only need one variable slot.
		return 1
	case ir.OpKindStyleProp, ir.OpKindClassProp, ir.OpKindStyleMap, ir.OpKindClassMap:
		// Style & class bindings use 2 variable slots, plus 1 slot for every interpolated expression,
		// if any.
		slots := 2
		var expr interface{}
		if stylePropOp, ok := op.(*ops_update.StylePropOp); ok {
			expr = stylePropOp.Expression
		} else if classPropOp, ok := op.(*ops_update.ClassPropOp); ok {
			expr = classPropOp.Expression
		} else if styleMapOp, ok := op.(*ops_update.StyleMapOp); ok {
			expr = styleMapOp.Expression
		} else if classMapOp, ok := op.(*ops_update.ClassMapOp); ok {
			expr = classMapOp.Expression
		}
		if interpolation, ok := expr.(*ops_update.Interpolation); ok {
			slots += len(interpolation.Expressions)
		}
		return slots
	case ir.OpKindInterpolateText:
		// `ir.InterpolateTextOp`s use a variable slot for each dynamic expression.
		if interpolateOp, ok := op.(*ops_update.InterpolateTextOp); ok {
			return len(interpolateOp.Interpolation.Expressions)
		}
		return 0
	case ir.OpKindI18nExpression, ir.OpKindConditional, ir.OpKindDeferWhen, ir.OpKindStoreLet:
		return 1
	case ir.OpKindRepeaterCreate:
		// Repeaters may require an extra variable binding slot, if they have an empty view, for the
		// empty block tracking.
		// TODO: It's a bit odd to have a create mode instruction consume variable slots. Maybe we can
		// find a way to use the Repeater update op instead.
		if repeaterOp, ok := op.(*ops_create.RepeaterCreateOp); ok {
			if repeaterOp.EmptyView != 0 {
				return 1
			}
		}
		return 0
	default:
		panic(fmt.Sprintf("Unhandled op: %d", kind))
	}
}

// VarsUsedByIrExpression counts the variables used by an IR expression
func VarsUsedByIrExpression(expr interface{}) int {
	return varsUsedByIrExpression(expr)
}

func varsUsedByIrExpression(expr interface{}) int {
	if pureFunc, ok := expr.(*expression.PureFunctionExpr); ok {
		return 1 + len(pureFunc.Args)
	} else if pipeBinding, ok := expr.(*expression.PipeBindingExpr); ok {
		return 1 + len(pipeBinding.Args)
	} else if pipeVariadic, ok := expr.(*expression.PipeBindingVariadicExpr); ok {
		return 1 + pipeVariadic.NumArgs
	} else if storeLet, ok := expr.(*expression.StoreLetExpr); ok {
		_ = storeLet
		return 1
	}
	panic(fmt.Sprintf("AssertionError: unhandled ConsumesVarsTrait expression %T", expr))
}

// isSingletonInterpolationv2 checks if an interpolation is a singleton
func isSingletonInterpolationv2(interpolation *ops_update.Interpolation) bool {
	if len(interpolation.Expressions) != 1 || len(interpolation.Strings) != 2 {
		return false
	}
	if interpolation.Strings[0] != "" || interpolation.Strings[1] != "" {
		return false
	}
	return true
}
