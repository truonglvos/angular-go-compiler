package phases

import (
	"ngc-go/packages/compiler/output"
	ir "ngc-go/packages/compiler/template/pipeline/ir/src"
	ops_update "ngc-go/packages/compiler/template/pipeline/ir/src/ops/update"

	pipeline "ngc-go/packages/compiler/template/pipeline/src/compilation"
)

// SpecializeStyleBindings transforms special-case bindings with 'style' or 'class' in their names.
// Must run before the main binding specialization pass.
func SpecializeStyleBindings(job *pipeline.CompilationJob) {
	for _, unit := range job.GetUnits() {
		for op := unit.GetUpdate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			if op.GetKind() != ir.OpKindBinding {
				continue
			}

			bindingOp, ok := op.(*ops_update.BindingOp)
			if !ok {
				continue
			}

			switch bindingOp.BindingKind {
			case ir.BindingKindClassName:
				if _, ok := bindingOp.Expression.(*ops_update.Interpolation); ok {
					panic("Unexpected interpolation in ClassName binding")
				}
				expr, _ := bindingOp.Expression.(output.OutputExpression)
				classPropOp := ops_update.NewClassPropOp(bindingOp.Target, bindingOp.Name, expr)
				unit.GetUpdate().Replace(op, classPropOp)
			case ir.BindingKindStyleProperty:
				expr := bindingOp.Expression
				stylePropOp := ops_update.NewStylePropOp(bindingOp.Target, bindingOp.Name, expr, bindingOp.Unit)
				unit.GetUpdate().Replace(op, stylePropOp)
			case ir.BindingKindProperty, ir.BindingKindTemplate:
				if bindingOp.Name == "style" {
					styleMapOp := ops_update.NewStyleMapOp(bindingOp.Target, bindingOp.Expression)
					unit.GetUpdate().Replace(op, styleMapOp)
				} else if bindingOp.Name == "class" {
					classMapOp := ops_update.NewClassMapOp(bindingOp.Target, bindingOp.Expression)
					unit.GetUpdate().Replace(op, classMapOp)
				}
			}
		}
	}
}
