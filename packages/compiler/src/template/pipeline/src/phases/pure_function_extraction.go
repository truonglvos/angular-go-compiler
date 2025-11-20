package phases

import (
	"fmt"

	"ngc-go/packages/compiler/src/output"
	constant_pool "ngc-go/packages/compiler/src/pool"
	ir "ngc-go/packages/compiler/src/template/pipeline/ir/src"
	ir_expression "ngc-go/packages/compiler/src/template/pipeline/ir/src/expression"

	pipeline "ngc-go/packages/compiler/src/template/pipeline/src/compilation"
)

// ExtractPureFunctions extracts pure functions from expressions and moves them to the constant pool.
func ExtractPureFunctions(job *pipeline.CompilationJob) {
	for _, unit := range job.GetUnits() {
		for op := unit.GetCreate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			ir_expression.VisitExpressionsInOp(op, func(expr output.OutputExpression, flags ir_expression.VisitorContextFlag) {
				pureFuncExpr, ok := expr.(*ir_expression.PureFunctionExpr)
				if !ok || pureFuncExpr.Body == nil {
					return
				}

				constantDef := &PureFunctionConstant{numArgs: len(pureFuncExpr.Args)}
				pureFuncExpr.Fn = job.Pool.GetSharedConstant(constantDef, pureFuncExpr.Body)
				pureFuncExpr.Body = nil
			})
		}
		for op := unit.GetUpdate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			ir_expression.VisitExpressionsInOp(op, func(expr output.OutputExpression, flags ir_expression.VisitorContextFlag) {
				pureFuncExpr, ok := expr.(*ir_expression.PureFunctionExpr)
				if !ok || pureFuncExpr.Body == nil {
					return
				}

				constantDef := &PureFunctionConstant{numArgs: len(pureFuncExpr.Args)}
				pureFuncExpr.Fn = job.Pool.GetSharedConstant(constantDef, pureFuncExpr.Body)
				pureFuncExpr.Body = nil
			})
		}
	}
}

// PureFunctionConstant is a shared constant definition for pure functions
type PureFunctionConstant struct {
	constant_pool.GenericKeyFn
	numArgs int
}

// KeyOf generates a key for a pure function expression
func (p *PureFunctionConstant) KeyOf(expr output.OutputExpression) string {
	// Check if it's a PureFunctionParameterExpr and return param(index) format
	if paramExpr, ok := expr.(*ir_expression.PureFunctionParameterExpr); ok {
		return fmt.Sprintf("param(%d)", paramExpr.Index)
	}
	// Otherwise, use the generic key function
	return p.GenericKeyFn.KeyOf(expr)
}

// ToSharedConstantDeclaration creates a declaration statement for the shared constant
func (p *PureFunctionConstant) ToSharedConstantDeclaration(declName string, keyExpr output.OutputExpression) output.OutputStatement {
	// Create function parameters
	fnParams := make([]*output.FnParam, p.numArgs)
	for idx := 0; idx < p.numArgs; idx++ {
		paramName := fmt.Sprintf("a%d", idx)
		fnParams[idx] = output.NewFnParam(paramName, nil)
	}

	// Transform PureFunctionParameterExpr to ReadVarExpr
	returnExpr := ir_expression.TransformExpressionsInExpression(
		keyExpr,
		func(expr output.OutputExpression, flags ir_expression.VisitorContextFlag) output.OutputExpression {
			if paramExpr, ok := expr.(*ir_expression.PureFunctionParameterExpr); ok {
				paramName := fmt.Sprintf("a%d", paramExpr.Index)
				return output.NewReadVarExpr(paramName, nil, nil)
			}
			return expr
		},
		ir_expression.VisitorContextFlagNone,
	)

	arrowFn := output.NewArrowFunctionExpr(fnParams, returnExpr, nil, nil)
	return output.NewDeclareVarStmt(declName, arrowFn, nil, output.StmtModifierFinal, nil, nil)
}
