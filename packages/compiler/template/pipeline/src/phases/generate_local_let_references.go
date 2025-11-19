package phases

import (
	ir "ngc-go/packages/compiler/template/pipeline/ir/src"
	ir_expression "ngc-go/packages/compiler/template/pipeline/ir/src/expression"
	ops_shared "ngc-go/packages/compiler/template/pipeline/ir/src/ops/shared"
	ops_update "ngc-go/packages/compiler/template/pipeline/ir/src/ops/update"
	ir_variable "ngc-go/packages/compiler/template/pipeline/ir/src/variable"

	pipeline "ngc-go/packages/compiler/template/pipeline/src/compilation"
)

// GenerateLocalLetReferences replaces the `storeLet` ops with variables that can be
// used to reference the value within the same view.
func GenerateLocalLetReferences(job *pipeline.ComponentCompilationJob) {
	for _, unit := range job.GetUnits() {
		for op := unit.GetUpdate().Head(); op != nil; op = op.Next() {
			if storeLetOp, ok := op.(*ops_update.StoreLetOp); ok {
				varDecl := ir_variable.NewIdentifierVariable(storeLetOp.DeclaredName, true)

				variableOp := ops_shared.NewVariableOp(
					job.AllocateXrefId(),
					varDecl,
					ir_expression.NewStoreLetExpr(storeLetOp.Target, storeLetOp.Value, storeLetOp.SourceSpan),
					ir.VariableFlagsNone,
				)

				// Replace the StoreLetOp with VariableOp
				unit.GetUpdate().Replace(storeLetOp, variableOp)
			}
		}
	}
}
