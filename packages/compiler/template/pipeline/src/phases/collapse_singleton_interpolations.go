package phases

import (
	ir "ngc-go/packages/compiler/template/pipeline/ir/src"
	ops_update "ngc-go/packages/compiler/template/pipeline/ir/src/ops/update"

	pipeline "ngc-go/packages/compiler/template/pipeline/src/compilation"
)

// CollapseSingletonInterpolations collapses attribute or style interpolations of the form
// `[attr.foo]="{{foo}}""` into a plain instruction, instead of an interpolated one.
//
// (We cannot do this for singleton property interpolations, because they need to stringify their expressions)
//
// The reification step is also capable of performing this transformation, but doing it early in the
// pipeline allows other phases to accurately know what instruction will be emitted.
func CollapseSingletonInterpolations(job *pipeline.CompilationJob) {
	for _, unit := range job.GetUnits() {
		for op := unit.GetUpdate().Head(); op != nil; op = op.Next() {
			kind := op.GetKind()
			eligibleOpKind := kind == ir.OpKindAttribute ||
				kind == ir.OpKindStyleProp ||
				kind == ir.OpKindStyleMap ||
				kind == ir.OpKindClassMap

			if eligibleOpKind {
				var interpolation *ops_update.Interpolation
				var ok bool

				if bindingOp, ok2 := op.(*ops_update.BindingOp); ok2 {
					interpolation, ok = bindingOp.Expression.(*ops_update.Interpolation)
					if ok && isSingletonInterpolation(interpolation) {
						bindingOp.Expression = interpolation.Expressions[0]
					}
				} else if propertyOp, ok2 := op.(*ops_update.PropertyOp); ok2 {
					interpolation, ok = propertyOp.Expression.(*ops_update.Interpolation)
					if ok && isSingletonInterpolation(interpolation) {
						propertyOp.Expression = interpolation.Expressions[0]
					}
				} else if attributeOp, ok2 := op.(*ops_update.AttributeOp); ok2 {
					interpolation, ok = attributeOp.Expression.(*ops_update.Interpolation)
					if ok && isSingletonInterpolation(interpolation) {
						attributeOp.Expression = interpolation.Expressions[0]
					}
				}
			}
		}
	}
}

// isSingletonInterpolation checks if an interpolation is a singleton (only one expression, empty strings)
func isSingletonInterpolation(interpolation *ops_update.Interpolation) bool {
	return len(interpolation.Expressions) == 1 &&
		len(interpolation.Strings) == 2 &&
		interpolation.Strings[0] == "" &&
		interpolation.Strings[1] == ""
}
