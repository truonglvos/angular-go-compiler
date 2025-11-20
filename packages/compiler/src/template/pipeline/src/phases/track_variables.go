package phases

import (
	"ngc-go/packages/compiler/src/output"
	"ngc-go/packages/compiler/src/template/pipeline/ir"
	ir_expression "ngc-go/packages/compiler/src/template/pipeline/ir/src/expression"
	ops_create "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/create"

	pipeline "ngc-go/packages/compiler/src/template/pipeline/src/compilation"
)

// GenerateTrackVariables finds variable usages inside the `track` expression on a `for` repeater,
// where the `$index` and `$item` variables are ambiently available, and replaces them with the
// appropriate output read.
func GenerateTrackVariables(job *pipeline.CompilationJob) {
	for _, unit := range job.GetUnits() {
		for op := unit.GetCreate().Head(); op != nil; op = op.Next() {
			if op.GetKind() != ir.OpKindRepeaterCreate {
				continue
			}

			repeaterOp, ok := op.(*ops_create.RepeaterCreateOp)
			if !ok {
				continue
			}

			repeaterOp.Track = ir_expression.TransformExpressionsInExpression(
				repeaterOp.Track,
				func(expr output.OutputExpression, flags ir_expression.VisitorContextFlag) output.OutputExpression {
					if lexicalRead, ok := expr.(*ir_expression.LexicalReadExpr); ok {
						// Check if this is $index
						if repeaterOp.VarNames.DollarIndex != nil {
							if _, exists := repeaterOp.VarNames.DollarIndex[lexicalRead.Name]; exists {
								return output.NewReadVarExpr("$index", nil, nil)
							}
						}
						// Check if this is $implicit (the item variable)
						if lexicalRead.Name == repeaterOp.VarNames.DollarImplicit {
							return output.NewReadVarExpr("$item", nil, nil)
						}

						// TODO: handle prohibited context variables (emit as globals?)
					}
					return expr
				},
				ir_expression.VisitorContextFlagNone,
			)
		}
	}
}
