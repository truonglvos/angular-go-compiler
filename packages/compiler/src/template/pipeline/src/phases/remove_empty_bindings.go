package phases

import (
	"ngc-go/packages/compiler/src/template/pipeline/ir"
	ir_expression "ngc-go/packages/compiler/src/template/pipeline/ir/src/expression"
	ops_update "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/update"

	pipeline "ngc-go/packages/compiler/src/template/pipeline/src/compilation"
)

// RemoveEmptyBindings removes bindings with no content, which can be safely deleted.
func RemoveEmptyBindings(job *pipeline.CompilationJob) {
	for _, unit := range job.GetUnits() {
		for op := unit.GetUpdate().Head(); op != nil; op = op.Next() {
			switch op.GetKind() {
			case ir.OpKindAttribute, ir.OpKindBinding, ir.OpKindClassProp, ir.OpKindClassMap,
				ir.OpKindProperty, ir.OpKindStyleProp, ir.OpKindStyleMap:
				if bindingOp, ok := op.(*ops_update.BindingOp); ok {
					if _, ok := bindingOp.Expression.(*ir_expression.EmptyExpr); ok {
						unit.GetUpdate().Remove(op)
					}
				} else if propertyOp, ok := op.(*ops_update.PropertyOp); ok {
					if _, ok := propertyOp.Expression.(*ir_expression.EmptyExpr); ok {
						unit.GetUpdate().Remove(op)
					}
				} else if attributeOp, ok := op.(*ops_update.AttributeOp); ok {
					if _, ok := attributeOp.Expression.(*ir_expression.EmptyExpr); ok {
						unit.GetUpdate().Remove(op)
					}
				}
			}
		}
	}
}
