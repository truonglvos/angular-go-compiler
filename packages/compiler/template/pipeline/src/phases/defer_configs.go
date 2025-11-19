package phases

import (
	ir "ngc-go/packages/compiler/template/pipeline/ir/src"
	ir_expression "ngc-go/packages/compiler/template/pipeline/ir/src/expression"
	ops_create "ngc-go/packages/compiler/template/pipeline/ir/src/ops/create"

	pipeline_compilation "ngc-go/packages/compiler/template/pipeline/src/compilation"
	pipeline_convension "ngc-go/packages/compiler/template/pipeline/src/convension"
)

// ConfigureDeferInstructions finds the config options for defer instructions, and creates the corresponding const array.
// Defer instructions take a configuration array, which should be collected into the component consts.
func ConfigureDeferInstructions(job *pipeline_compilation.ComponentCompilationJob) {
	for _, unit := range job.GetUnits() {
		for op := unit.GetCreate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			if op.GetKind() != ir.OpKindDefer {
				continue
			}

			deferOp, ok := op.(*ops_create.DeferOp)
			if !ok {
				continue
			}

			if deferOp.PlaceholderMinimumTime != nil {
				configArray := pipeline_convension.LiteralOrArrayLiteral([]interface{}{*deferOp.PlaceholderMinimumTime})
				deferOp.PlaceholderConfig = ir_expression.NewConstCollectedExpr(configArray)
			}
			if deferOp.LoadingMinimumTime != nil || deferOp.LoadingAfterTime != nil {
				configValues := make([]interface{}, 0, 2)
				if deferOp.LoadingMinimumTime != nil {
					configValues = append(configValues, *deferOp.LoadingMinimumTime)
				} else {
					configValues = append(configValues, nil)
				}
				if deferOp.LoadingAfterTime != nil {
					configValues = append(configValues, *deferOp.LoadingAfterTime)
				} else {
					configValues = append(configValues, nil)
				}
				configArray := pipeline_convension.LiteralOrArrayLiteral(configValues)
				deferOp.LoadingConfig = ir_expression.NewConstCollectedExpr(configArray)
			}
		}
	}
}
