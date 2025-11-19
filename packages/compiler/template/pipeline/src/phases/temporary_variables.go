package phases

import (
	"fmt"

	"ngc-go/packages/compiler/output"
	ir "ngc-go/packages/compiler/template/pipeline/ir/src"
	ir_expression "ngc-go/packages/compiler/template/pipeline/ir/src/expression"
	ir_operation "ngc-go/packages/compiler/template/pipeline/ir/src/operations"
	ops_create "ngc-go/packages/compiler/template/pipeline/ir/src/ops/create"
	ops_shared "ngc-go/packages/compiler/template/pipeline/ir/src/ops/shared"

	pipeline "ngc-go/packages/compiler/template/pipeline/src/compilation"
)

// GenerateTemporaryVariables finds all assignments and usages of temporary variables, which are linked to each other with cross
// references. Generate names for each cross-reference, and add a `DeclareVarStmt` to initialize
// them at the beginning of the update block.
//
// TODO: Sometimes, it will be possible to reuse names across different subexpressions. For example,
// in the double keyed read `a?.[f()]?.[f()]`, the two function calls have non-overlapping scopes.
// Implement an algorithm for reuse.
func GenerateTemporaryVariables(job *pipeline.CompilationJob) {
	for _, unit := range job.GetUnits() {
		createTemporaries := generateTemporaries(unit.GetCreate())
		for i := len(createTemporaries) - 1; i >= 0; i-- {
			unit.GetCreate().InsertBefore(unit.GetCreate().Head(), createTemporaries[i])
		}

		updateTemporaries := generateTemporaries(unit.GetUpdate())
		for i := len(updateTemporaries) - 1; i >= 0; i-- {
			unit.GetUpdate().InsertBefore(unit.GetUpdate().Head(), updateTemporaries[i])
		}
	}
}

func generateTemporaries(opsList *ir_operation.OpList) []*ops_shared.StatementOp {
	opCount := 0
	var generatedStatements []*ops_shared.StatementOp

	// For each op, search for any variables that are assigned or read. For each variable, generate a
	// name and produce a `DeclareVarStmt` to the beginning of the block.
	for op := opsList.Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
		// Identify the final time each temp var is read.
		finalReads := make(map[ir_operation.XrefId]*ir_expression.ReadTemporaryExpr)
		ir_expression.VisitExpressionsInOp(op, func(expr output.OutputExpression, flags ir_expression.VisitorContextFlag) {
			if flags&ir_expression.VisitorContextFlagInChildOperation != 0 {
				return
			}
			if readTemp, ok := expr.(*ir_expression.ReadTemporaryExpr); ok {
				finalReads[readTemp.Xref] = readTemp
			}
		})

		// Name the temp vars, accounting for the fact that a name can be reused after it has been
		// read for the final time.
		count := 0
		assigned := make(map[ir_operation.XrefId]bool)
		released := make(map[ir_operation.XrefId]bool)
		defs := make(map[ir_operation.XrefId]string)

		ir_expression.VisitExpressionsInOp(op, func(expr output.OutputExpression, flags ir_expression.VisitorContextFlag) {
			if flags&ir_expression.VisitorContextFlagInChildOperation != 0 {
				return
			}
			if assignTemp, ok := expr.(*ir_expression.AssignTemporaryExpr); ok {
				if !assigned[assignTemp.Xref] {
					assigned[assignTemp.Xref] = true
					// TODO: Exactly replicate the naming scheme used by `TemplateDefinitionBuilder`.
					// It seems to rely on an expression index instead of an op index.
					defs[assignTemp.Xref] = fmt.Sprintf("tmp_%d_%d", opCount, count)
					count++
				}
				assignName(defs, assignTemp)
			} else if readTemp, ok := expr.(*ir_expression.ReadTemporaryExpr); ok {
				if finalReads[readTemp.Xref] == readTemp {
					released[readTemp.Xref] = true
					count--
				}
				assignName(defs, readTemp)
			}
		})

		// Add declarations for the temp vars.
		uniqueNames := make(map[string]bool)
		for _, name := range defs {
			if !uniqueNames[name] {
				uniqueNames[name] = true
				stmt := output.NewDeclareVarStmt(name, nil, nil, output.StmtModifierNone, nil, nil)
				generatedStatements = append(generatedStatements, ops_shared.NewStatementOp(stmt))
			}
		}
		opCount++

		// Recursively process handler ops and trackByOps
		kind := op.GetKind()
		if kind == ir.OpKindListener || kind == ir.OpKindAnimation ||
			kind == ir.OpKindAnimationListener || kind == ir.OpKindTwoWayListener {
			var handlerOps *ir_operation.OpList
			if listenerOp, ok := op.(*ops_create.ListenerOp); ok {
				handlerOps = listenerOp.HandlerOps
			} else if animOp, ok := op.(*ops_create.AnimationOp); ok {
				handlerOps = animOp.HandlerOps
			} else if animListenerOp, ok := op.(*ops_create.AnimationListenerOp); ok {
				handlerOps = animListenerOp.HandlerOps
			} else if twoWayOp, ok := op.(*ops_create.TwoWayListenerOp); ok {
				handlerOps = twoWayOp.HandlerOps
			}
			if handlerOps != nil {
				handlerTemporaries := generateTemporaries(handlerOps)
				for i := len(handlerTemporaries) - 1; i >= 0; i-- {
					handlerOps.InsertBefore(handlerOps.Head(), handlerTemporaries[i])
				}
			}
		} else if kind == ir.OpKindRepeaterCreate {
			if repeaterOp, ok := op.(*ops_create.RepeaterCreateOp); ok {
				if repeaterOp.TrackByOps != nil {
					trackByTemporaries := generateTemporaries(repeaterOp.TrackByOps)
					for i := len(trackByTemporaries) - 1; i >= 0; i-- {
						repeaterOp.TrackByOps.InsertBefore(repeaterOp.TrackByOps.Head(), trackByTemporaries[i])
					}
				}
			}
		}
	}

	return generatedStatements
}

// assignName assigns a name to the temporary variable in the given temporary variable expression.
func assignName(names map[ir_operation.XrefId]string, expr interface{}) {
	var xref ir_operation.XrefId
	var namePtr **string

	if assignTemp, ok := expr.(*ir_expression.AssignTemporaryExpr); ok {
		xref = assignTemp.Xref
		namePtr = &assignTemp.Name
	} else if readTemp, ok := expr.(*ir_expression.ReadTemporaryExpr); ok {
		xref = readTemp.Xref
		namePtr = &readTemp.Name
	} else {
		return
	}

	name, exists := names[xref]
	if !exists {
		panic(fmt.Sprintf("Found xref with unassigned name: %d", xref))
	}
	*namePtr = &name
}
