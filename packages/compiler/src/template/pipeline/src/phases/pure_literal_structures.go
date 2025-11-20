package phases

import (
	"ngc-go/packages/compiler/src/output"
	ir "ngc-go/packages/compiler/src/template/pipeline/ir/src"
	ir_expression "ngc-go/packages/compiler/src/template/pipeline/ir/src/expression"

	pipeline "ngc-go/packages/compiler/src/template/pipeline/src/compilation"
)

// GeneratePureLiteralStructures transforms literal arrays and maps into pure function expressions.
func GeneratePureLiteralStructures(job *pipeline.CompilationJob) {
	for _, unit := range job.GetUnits() {
		for op := unit.GetUpdate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			ir_expression.TransformExpressionsInOp(
				op,
				func(expr output.OutputExpression, flags ir_expression.VisitorContextFlag) output.OutputExpression {
					if flags&ir_expression.VisitorContextFlagInChildOperation != 0 {
						return expr
					}

					if literalArray, ok := expr.(*output.LiteralArrayExpr); ok {
						return transformLiteralArray(literalArray)
					} else if literalMap, ok := expr.(*output.LiteralMapExpr); ok {
						return transformLiteralMap(literalMap)
					}

					return expr
				},
				ir_expression.VisitorContextFlagNone,
			)
		}
	}
}

func transformLiteralArray(expr *output.LiteralArrayExpr) output.OutputExpression {
	var derivedEntries []output.OutputExpression
	var nonConstantArgs []output.OutputExpression
	for _, entry := range expr.Entries {
		if entry.IsConstant() {
			derivedEntries = append(derivedEntries, entry)
		} else {
			idx := len(nonConstantArgs)
			nonConstantArgs = append(nonConstantArgs, entry)
			derivedEntries = append(derivedEntries, ir_expression.NewPureFunctionParameterExpr(idx))
		}
	}
	return ir_expression.NewPureFunctionExpr(output.NewLiteralArrayExpr(derivedEntries, nil, nil), nonConstantArgs)
}

func transformLiteralMap(expr *output.LiteralMapExpr) output.OutputExpression {
	var derivedEntries []*output.LiteralMapEntry
	var nonConstantArgs []output.OutputExpression
	for _, entry := range expr.Entries {
		if entry.Value.IsConstant() {
			derivedEntries = append(derivedEntries, entry)
		} else {
			idx := len(nonConstantArgs)
			nonConstantArgs = append(nonConstantArgs, entry.Value)
			derivedEntries = append(derivedEntries, output.NewLiteralMapEntry(
				entry.Key,
				ir_expression.NewPureFunctionParameterExpr(idx),
				entry.Quoted,
			))
		}
	}
	return ir_expression.NewPureFunctionExpr(output.NewLiteralMapExpr(derivedEntries, nil, nil), nonConstantArgs)
}
