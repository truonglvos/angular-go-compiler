package phases

import (
	"fmt"

	"ngc-go/packages/compiler/src/output"
	ir "ngc-go/packages/compiler/src/template/pipeline/ir/src"
	ir_expression "ngc-go/packages/compiler/src/template/pipeline/ir/src/expression"
	pipeline "ngc-go/packages/compiler/src/template/pipeline/src/compilation"
	instruction "ngc-go/packages/compiler/src/template/pipeline/src/instruction"
)

// TransformTwoWayBindingSet transforms a `TwoWayBindingSet` expression into an expression that either
// sets a value through the `twoWayBindingSet` instruction or falls back to setting
// the value directly. E.g. the expression `TwoWayBindingSet(target, value)` becomes:
// `ng.twoWayBindingSet(target, value) || (target = value)`.
func TransformTwoWayBindingSet(job *pipeline.CompilationJob) {
	for _, unit := range job.GetUnits() {
		for op := unit.GetCreate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			if op.GetKind() != ir.OpKindTwoWayListener {
				continue
			}

			ir_expression.TransformExpressionsInOp(
				op,
				func(expr output.OutputExpression, flags ir_expression.VisitorContextFlag) output.OutputExpression {
					if flags&ir_expression.VisitorContextFlagInChildOperation == 0 {
						return expr
					}

					twoWayBindingSet, ok := expr.(*ir_expression.TwoWayBindingSetExpr)
					if !ok {
						return expr
					}

					target := twoWayBindingSet.Target
					value := twoWayBindingSet.Value

					if readProp, ok := target.(*output.ReadPropExpr); ok {
						// `ng.twoWayBindingSet(target, value) || (target = value)`
						twoWayBindingSetCall := instruction.TwoWayBindingSet(readProp, value)
						assignment := readProp.Set(value)
						return or(twoWayBindingSetCall, assignment)
					} else if readKey, ok := target.(*output.ReadKeyExpr); ok {
						// `ng.twoWayBindingSet(target, value) || (target = value)`
						twoWayBindingSetCall := instruction.TwoWayBindingSet(readKey, value)
						assignment := readKey.Set(value)
						return or(twoWayBindingSetCall, assignment)
					} else if readVar, ok := target.(*ir_expression.ReadVariableExpr); ok {
						// ASSUMPTION: here we're assuming that `ReadVariableExpr` will be a reference
						// to a local template variable. This appears to be the case at the time of writing.
						// If the expression is targeting a variable read, we only emit the `twoWayBindingSet`
						// since the fallback would be attempting to write into a constant. Invalid usages will be
						// flagged during template type checking.
						return instruction.TwoWayBindingSet(readVar, value)
					}

					panic(fmt.Sprintf("Unsupported expression in two-way action binding: %T", target))
				},
				ir_expression.VisitorContextFlagInChildOperation,
			)
		}
	}
}

// or creates a binary OR expression: `lhs || rhs`
func or(lhs, rhs output.OutputExpression) output.OutputExpression {
	return output.NewBinaryOperatorExpr(
		output.BinaryOperatorOr,
		lhs,
		rhs,
		nil,
		nil,
	)
}
