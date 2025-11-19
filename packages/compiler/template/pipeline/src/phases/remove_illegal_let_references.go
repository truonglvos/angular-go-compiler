package phases

import (
	"ngc-go/packages/compiler/output"
	ir "ngc-go/packages/compiler/template/pipeline/ir/src"
	ir_expression "ngc-go/packages/compiler/template/pipeline/ir/src/expression"
	ops_shared "ngc-go/packages/compiler/template/pipeline/ir/src/ops/shared"
	ir_variable "ngc-go/packages/compiler/template/pipeline/ir/src/variable"
	pipeline "ngc-go/packages/compiler/template/pipeline/src/compilation"
)

// RemoveIllegalLetReferences removes illegal forward references to `@let` declarations.
// It's not allowed to access a `@let` declaration before it has been defined. This is enforced
// already via template type checking, however it can trip some of the assertions in the pipeline.
// E.g. the naming phase can fail because we resolved the variable here, but the variable doesn't
// exist anymore because the optimization phase removed it since it's invalid. To avoid surfacing
// confusing errors to users in the case where template type checking isn't running (e.g. in JIT
// mode) this phase detects illegal forward references and replaces them with `undefined`.
// Eventually users will see the proper error from the template type checker.
func RemoveIllegalLetReferences(job *pipeline.CompilationJob) {
	for _, unit := range job.GetUnits() {
		for op := unit.GetUpdate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			if op.GetKind() != ir.OpKindVariable {
				continue
			}

			variableOp, ok := op.(*ops_shared.VariableOp)
			if !ok {
				continue
			}

		// Check if this is an identifier variable with a StoreLetExpr initializer
		identifierVar, ok := variableOp.Variable.(*ir_variable.IdentifierVariable)
		if !ok || identifierVar.Kind != ir.SemanticVariableKindIdentifier {
			continue
		}

		_, ok = variableOp.Initializer.(*ir_expression.StoreLetExpr)
		if !ok {
			continue
		}

			name := identifierVar.Identifier

			// Walk backwards through the update list and replace any LexicalReadExpr with this name
			current := op
			for current != nil && current.GetKind() != ir.OpKindListEnd {
				ir_expression.TransformExpressionsInOp(
					current,
					func(expr output.OutputExpression, flags ir_expression.VisitorContextFlag) output.OutputExpression {
						if lexicalRead, ok := expr.(*ir_expression.LexicalReadExpr); ok && lexicalRead.Name == name {
							return output.NewLiteralExpr(nil, nil, nil) // undefined
						}
						return expr
					},
					ir_expression.VisitorContextFlagNone,
				)
				current = current.GetPrev()
			}
		}
	}
}
