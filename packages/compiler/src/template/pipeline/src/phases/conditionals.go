package phases

import (
	"ngc-go/packages/compiler/src/output"
	"ngc-go/packages/compiler/src/template/pipeline/ir"
	"ngc-go/packages/compiler/src/template/pipeline/ir/src/expression"
	"ngc-go/packages/compiler/src/template/pipeline/ir/src/operations"
	ops_update "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/update"

	pipeline "ngc-go/packages/compiler/src/template/pipeline/src/compilation"
)

// GenerateConditionalExpressions collapses the various conditions of conditional ops (if, switch) into a single test expression.
func GenerateConditionalExpressions(job *pipeline.ComponentCompilationJob) {
	for _, unit := range job.GetUnits() {
		for op := unit.GetCreate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			if op.GetKind() != ir.OpKindConditional {
				continue
			}

			conditionalOp, ok := op.(*ops_update.ConditionalOp)
			if !ok {
				continue
			}

			var test output.OutputExpression

			// Any case with a `null` condition is `default`. If one exists, default to it instead.
			defaultCaseIndex := -1
			var defaultSlot *ir.SlotHandle
			for i, cond := range conditionalOp.Conditions {
				if caseExpr, ok := cond.(*expression.ConditionalCaseExpr); ok {
					if caseExpr.Expr == nil {
						defaultCaseIndex = i
						defaultSlot = caseExpr.TargetSlot
						break
					}
				}
			}

			if defaultCaseIndex >= 0 {
				// Remove the default case from conditions
				conditions := make([]interface{}, 0, len(conditionalOp.Conditions)-1)
				for i, cond := range conditionalOp.Conditions {
					if i != defaultCaseIndex {
						conditions = append(conditions, cond)
					}
				}
				conditionalOp.Conditions = conditions
				test = expression.NewSlotLiteralExpr(defaultSlot)
			} else {
				// By default, a switch evaluates to `-1`, causing no template to be displayed.
				test = output.NewLiteralExpr(-1, nil, nil)
			}

			// Switch expressions assign their main test to a temporary, to avoid re-executing it.
			var tmp *expression.AssignTemporaryExpr
			if conditionalOp.Processed != nil {
				xref := job.AllocateXrefId()
				tmp = expression.NewAssignTemporaryExpr(conditionalOp.Processed, xref)
			}
			var caseExpressionTemporaryXref operations.XrefId = 0
			hasCaseExpressionTemporary := false

			// For each remaining condition, test whether the temporary satisfies the check. (If no temp is
			// present, just check each expression directly.)
			for i := len(conditionalOp.Conditions) - 1; i >= 0; i-- {
				caseExpr, ok := conditionalOp.Conditions[i].(*expression.ConditionalCaseExpr)
				if !ok || caseExpr.Expr == nil {
					continue
				}

				if tmp != nil {
					var useTmp output.OutputExpression
					if i == 0 {
						useTmp = tmp
					} else {
						useTmp = expression.NewReadTemporaryExpr(tmp.Xref)
					}
					caseExpr.Expr = output.NewBinaryOperatorExpr(
						output.BinaryOperatorIdentical,
						useTmp,
						caseExpr.Expr,
						nil,
						nil,
					)
				} else if caseExpr.Alias != nil {
					// Since we can only pass one variable into the conditional instruction,
					// reuse the same variable to store the result of the expressions.
					if !hasCaseExpressionTemporary {
						caseExpressionTemporaryXref = job.AllocateXrefId()
						hasCaseExpressionTemporary = true
					}
					caseExpr.Expr = expression.NewAssignTemporaryExpr(caseExpr.Expr, caseExpressionTemporaryXref)
					conditionalOp.ContextValue = expression.NewReadTemporaryExpr(caseExpressionTemporaryXref)
				}

				test = output.NewConditionalExpr(
					caseExpr.Expr,
					expression.NewSlotLiteralExpr(caseExpr.TargetSlot),
					test,
					nil,
					nil,
				)
			}

			// Save the resulting aggregate expression.
			conditionalOp.Processed = test

			// Clear the original conditions array, since we no longer need it, and don't want it to
			// affect subsequent phases (e.g. pipe creation).
			conditionalOp.Conditions = []interface{}{}
		}

		for op := unit.GetUpdate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			if op.GetKind() != ir.OpKindConditional {
				continue
			}

			conditionalOp, ok := op.(*ops_update.ConditionalOp)
			if !ok {
				continue
			}

			var test output.OutputExpression

			// Any case with a `null` condition is `default`. If one exists, default to it instead.
			defaultCaseIndex := -1
			var defaultSlot *ir.SlotHandle
			for i, cond := range conditionalOp.Conditions {
				if caseExpr, ok := cond.(*expression.ConditionalCaseExpr); ok {
					if caseExpr.Expr == nil {
						defaultCaseIndex = i
						defaultSlot = caseExpr.TargetSlot
						break
					}
				}
			}

			if defaultCaseIndex >= 0 {
				// Remove the default case from conditions
				conditions := make([]interface{}, 0, len(conditionalOp.Conditions)-1)
				for i, cond := range conditionalOp.Conditions {
					if i != defaultCaseIndex {
						conditions = append(conditions, cond)
					}
				}
				conditionalOp.Conditions = conditions
				test = expression.NewSlotLiteralExpr(defaultSlot)
			} else {
				// By default, a switch evaluates to `-1`, causing no template to be displayed.
				test = output.NewLiteralExpr(-1, nil, nil)
			}

			// Switch expressions assign their main test to a temporary, to avoid re-executing it.
			var tmp *expression.AssignTemporaryExpr
			if conditionalOp.Processed != nil {
				xref := job.AllocateXrefId()
				tmp = expression.NewAssignTemporaryExpr(conditionalOp.Processed, xref)
			}
			var caseExpressionTemporaryXref operations.XrefId = 0
			hasCaseExpressionTemporary := false

			// For each remaining condition, test whether the temporary satisfies the check. (If no temp is
			// present, just check each expression directly.)
			for i := len(conditionalOp.Conditions) - 1; i >= 0; i-- {
				caseExpr, ok := conditionalOp.Conditions[i].(*expression.ConditionalCaseExpr)
				if !ok || caseExpr.Expr == nil {
					continue
				}

				if tmp != nil {
					var useTmp output.OutputExpression
					if i == 0 {
						useTmp = tmp
					} else {
						useTmp = expression.NewReadTemporaryExpr(tmp.Xref)
					}
					caseExpr.Expr = output.NewBinaryOperatorExpr(
						output.BinaryOperatorIdentical,
						useTmp,
						caseExpr.Expr,
						nil,
						nil,
					)
				} else if caseExpr.Alias != nil {
					// Since we can only pass one variable into the conditional instruction,
					// reuse the same variable to store the result of the expressions.
					if !hasCaseExpressionTemporary {
						caseExpressionTemporaryXref = job.AllocateXrefId()
						hasCaseExpressionTemporary = true
					}
					caseExpr.Expr = expression.NewAssignTemporaryExpr(caseExpr.Expr, caseExpressionTemporaryXref)
					conditionalOp.ContextValue = expression.NewReadTemporaryExpr(caseExpressionTemporaryXref)
				}

				test = output.NewConditionalExpr(
					caseExpr.Expr,
					expression.NewSlotLiteralExpr(caseExpr.TargetSlot),
					test,
					nil,
					nil,
				)
			}

			// Save the resulting aggregate expression.
			conditionalOp.Processed = test

			// Clear the original conditions array, since we no longer need it, and don't want it to
			// affect subsequent phases (e.g. pipe creation).
			conditionalOp.Conditions = []interface{}{}
		}
	}
}
