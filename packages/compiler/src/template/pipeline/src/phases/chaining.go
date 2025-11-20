package phases

import (
	"ngc-go/packages/compiler/src/output"
	render3 "ngc-go/packages/compiler/src/render3/r3_identifiers"
	"ngc-go/packages/compiler/src/template/pipeline/ir"
	"ngc-go/packages/compiler/src/template/pipeline/ir/src/operations"
	ops_shared "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/shared"
	"ngc-go/packages/compiler/src/util"

	pipeline "ngc-go/packages/compiler/src/template/pipeline/src/compilation"
)

var chainCompatibility = map[output.ExternalReference]output.ExternalReference{
	*render3.AriaProperty:             *render3.AriaProperty,
	*render3.Attribute:                *render3.Attribute,
	*render3.ClassProp:                *render3.ClassProp,
	*render3.Element:                  *render3.Element,
	*render3.ElementContainer:         *render3.ElementContainer,
	*render3.ElementContainerEnd:      *render3.ElementContainerEnd,
	*render3.ElementContainerStart:    *render3.ElementContainerStart,
	*render3.ElementEnd:               *render3.ElementEnd,
	*render3.ElementStart:             *render3.ElementStart,
	*render3.DomProperty:              *render3.DomProperty,
	*render3.I18nExp:                  *render3.I18nExp,
	*render3.Listener:                 *render3.Listener,
	*render3.Property:                 *render3.Property,
	*render3.StyleProp:                *render3.StyleProp,
	*render3.SyntheticHostListener:    *render3.SyntheticHostListener,
	*render3.SyntheticHostProperty:    *render3.SyntheticHostProperty,
	*render3.TemplateCreate:           *render3.TemplateCreate,
	*render3.TwoWayProperty:           *render3.TwoWayProperty,
	*render3.TwoWayListener:           *render3.TwoWayListener,
	*render3.DeclareLet:               *render3.DeclareLet,
	*render3.ConditionalCreate:        *render3.ConditionalBranchCreate,
	*render3.ConditionalBranchCreate:  *render3.ConditionalBranchCreate,
	*render3.DomElement:               *render3.DomElement,
	*render3.DomElementStart:          *render3.DomElementStart,
	*render3.DomElementEnd:            *render3.DomElementEnd,
	*render3.DomElementContainer:      *render3.DomElementContainer,
	*render3.DomElementContainerStart: *render3.DomElementContainerStart,
	*render3.DomElementContainerEnd:   *render3.DomElementContainerEnd,
	*render3.DomListener:              *render3.DomListener,
	*render3.DomTemplate:              *render3.DomTemplate,
	*render3.AnimationEnter:           *render3.AnimationEnter,
	*render3.AnimationLeave:           *render3.AnimationLeave,
	*render3.AnimationEnterListener:   *render3.AnimationEnterListener,
	*render3.AnimationLeaveListener:   *render3.AnimationLeaveListener,
}

// MAX_CHAIN_LENGTH limits the maximum number of chained instructions to prevent running out of stack depth
const MAX_CHAIN_LENGTH = 256

// Chain post-processes a reified view compilation and converts sequential calls to chainable instructions
// into chain calls.
//
// For example, two `elementStart` operations in sequence:
//
// ```ts
// elementStart(0, 'div');
// elementStart(1, 'span');
// ```
//
// Can be called as a chain instead:
//
// ```ts
// elementStart(0, 'div')(1, 'span');
// ```
func Chain(job *pipeline.CompilationJob) {
	for _, unit := range job.GetUnits() {
		chainOperationsInList(unit.GetCreate())
		chainOperationsInList(unit.GetUpdate())
	}
}

type chain struct {
	op          *ops_shared.StatementOp
	instruction output.ExternalReference
	expression  output.OutputExpression
	length      int
}

func chainOperationsInList(opList *operations.OpList) {
	var currentChain *chain = nil

	for op := opList.Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
		if op.GetKind() != ir.OpKindStatement {
			currentChain = nil
			continue
		}

		stmtOp, ok := op.(*ops_shared.StatementOp)
		if !ok {
			currentChain = nil
			continue
		}

		exprStmt, ok := stmtOp.Statement.(*output.ExpressionStatement)
		if !ok {
			currentChain = nil
			continue
		}

		invokeExpr, ok := exprStmt.Expr.(*output.InvokeFunctionExpr)
		if !ok {
			currentChain = nil
			continue
		}

		externalExpr, ok := invokeExpr.Fn.(*output.ExternalExpr)
		if !ok {
			currentChain = nil
			continue
		}

		instruction := externalExpr.Value

		// Check if this instruction is chainable
		compatibleRef, isChainable := chainCompatibility[*instruction]
		if !isChainable {
			currentChain = nil
			continue
		}

		// This instruction can be chained. It can either be added on to the previous chain (if
		// compatible) or it can be the start of a new chain.
		if currentChain != nil &&
			chainCompatibility[currentChain.instruction] == compatibleRef &&
			currentChain.length < MAX_CHAIN_LENGTH {
			// This instruction can be added onto the previous chain.
			chainedExpr := callFn(currentChain.expression, invokeExpr.Args, invokeExpr.SourceSpan, invokeExpr.Pure)
			currentChain.expression = chainedExpr
			stmtOp.Statement = toStmt(chainedExpr, invokeExpr.SourceSpan)
			currentChain.length++
			opList.Remove(op)
		} else {
			// Leave this instruction alone for now, but consider it the start of a new chain.
			currentChain = &chain{
				op:          stmtOp,
				instruction: *instruction,
				expression:  invokeExpr,
				length:      1,
			}
		}
	}
}

// callFn creates a new InvokeFunctionExpr with the given expression as the function and args as arguments
func callFn(fn output.OutputExpression, args []output.OutputExpression, sourceSpan *util.ParseSourceSpan, pure bool) output.OutputExpression {
	return output.NewInvokeFunctionExpr(fn, args, nil, sourceSpan, pure)
}

// toStmt creates an ExpressionStatement from an expression
func toStmt(expr output.OutputExpression, sourceSpan *util.ParseSourceSpan) output.OutputStatement {
	return output.NewExpressionStatement(expr, sourceSpan, nil)
}
