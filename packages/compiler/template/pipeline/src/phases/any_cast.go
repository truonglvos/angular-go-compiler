package phases

import (
	"ngc-go/packages/compiler/output"

	ir_expression "ngc-go/packages/compiler/template/pipeline/ir/src/expression"
	pipeline_compilation "ngc-go/packages/compiler/template/pipeline/src/compilation"
)

// DeleteAnyCasts finds any function calls to `$any`, excluding `this.$any`, and deletes them,
// since they have no runtime effects.
func DeleteAnyCasts(job *pipeline_compilation.CompilationJob) {
	for _, unit := range job.GetUnits() {
		// Iterate through all ops in create and update lists
		for op := unit.GetCreate().Head(); op != nil; op = op.Next() {
			ir_expression.TransformExpressionsInOp(op, removeAnys, ir_expression.VisitorContextFlagNone)
		}
		for op := unit.GetUpdate().Head(); op != nil; op = op.Next() {
			ir_expression.TransformExpressionsInOp(op, removeAnys, ir_expression.VisitorContextFlagNone)
		}
	}
}

// removeAnys removes $any function calls from expressions
func removeAnys(e output.OutputExpression, flags ir_expression.VisitorContextFlag) output.OutputExpression {
	if invoke, ok := e.(*output.InvokeFunctionExpr); ok {
		if lexicalRead, ok := invoke.Fn.(*ir_expression.LexicalReadExpr); ok && lexicalRead.Name == "$any" {
			if len(invoke.Args) != 1 {
				panic("The $any builtin function expects exactly one argument.")
			}
			return invoke.Args[0]
		}
	}
	return e
}
