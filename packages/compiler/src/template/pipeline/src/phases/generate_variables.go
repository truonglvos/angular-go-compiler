package phases

import (
	"ngc-go/packages/compiler/src/output"
	"ngc-go/packages/compiler/src/template/pipeline/ir"
	"ngc-go/packages/compiler/src/template/pipeline/ir/src/expression"
	"ngc-go/packages/compiler/src/template/pipeline/ir/src/operations"
	ops_create "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/create"
	"ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/shared"
	ir_variable "ngc-go/packages/compiler/src/template/pipeline/ir/src/variable"

	pipeline "ngc-go/packages/compiler/src/template/pipeline/src/compilation"
)

// GenerateVariables generates a preamble sequence for each view creation block and listener function which declares
// any variables that be referenced in other operations in the block.
// Variables generated include:
//   - a saved view context to be used to restore the current view in event listeners.
//   - the context of the restored view within event listener handlers.
//   - context variables from the current view as well as all parent views (including the root
//     context if needed).
//   - local references from elements within the current view and any lexical parents.
//
// Variables are generated here unconditionally, and may optimized away in future operations if it
// turns out their values (and any side effects) are unused.
func GenerateVariables(job *pipeline.ComponentCompilationJob) {
	recursivelyProcessView(job.Root, nil)
}

// Scope is the lexical scope of a view, including a reference to its parent view's scope, if any.
type Scope struct {
	// XrefId of the view to which this scope corresponds.
	View operations.XrefId

	ViewContextVariable *ir_variable.ContextVariable

	ContextVariables map[string]ir_variable.SemanticVariable

	Aliases map[ir_variable.AliasVariable]bool

	// Local references collected from elements within the view.
	References []Reference

	// `@let` declarations collected from the view.
	LetDeclarations []LetDeclaration

	// Scope of the parent view, if any.
	Parent *Scope
}

// Reference is information needed about a local reference collected from an element within a view.
type Reference struct {
	// Name given to the local reference variable within the template.
	// This is not the name which will be used for the variable declaration in the generated
	// template code.
	Name string

	// XrefId of the element-like node which this reference targets.
	// The reference may be either to the element (or template) itself, or to a directive on it.
	TargetId operations.XrefId

	TargetSlot *ir.SlotHandle

	// A generated offset of this reference among all the references on a specific element.
	Offset int

	Variable *ir_variable.IdentifierVariable
}

// LetDeclaration is information about `@let` declaration collected from a view.
type LetDeclaration struct {
	// XrefId of the `@let` declaration that the reference is pointing to.
	TargetId operations.XrefId

	// Slot in which the declaration is stored.
	TargetSlot *ir.SlotHandle

	// Variable referring to the declaration.
	Variable *ir_variable.IdentifierVariable
}

// recursivelyProcessView processes the given ViewCompilationUnit and generates preambles for it and any listeners that it
// declares.
// parentScope is a scope extracted from the parent view which captures any variables which
// should be inherited by this view. nil if the current view is the root view.
func recursivelyProcessView(view *pipeline.ViewCompilationUnit, parentScope *Scope) {
	// Extract a `Scope` from this view.
	scope := getScopeForView(view, parentScope)

	for op := view.Create.Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
		switch op.GetKind() {
		case ir.OpKindConditionalCreate:
			conditionalOp, ok := op.(*ops_create.ConditionalCreateOp)
			if ok {
				// Descend into child embedded views.
				recursivelyProcessView(view.Job.Views[conditionalOp.Xref], scope)
			}
		case ir.OpKindConditionalBranchCreate:
			branchOp, ok := op.(*ops_create.ConditionalBranchCreateOp)
			if ok {
				recursivelyProcessView(view.Job.Views[branchOp.Xref], scope)
			}
		case ir.OpKindTemplate:
			templateOp, ok := op.(*ops_create.TemplateOp)
			if ok {
				// Descend into child embedded views.
				recursivelyProcessView(view.Job.Views[templateOp.Xref], scope)
			}
		case ir.OpKindProjection:
			projectionOp, ok := op.(*ops_create.ProjectionOp)
			if ok && projectionOp.FallbackView != 0 {
				recursivelyProcessView(view.Job.Views[projectionOp.FallbackView], scope)
			}
		case ir.OpKindRepeaterCreate:
			repeaterOp, ok := op.(*ops_create.RepeaterCreateOp)
			if ok {
				// Descend into child embedded views.
				recursivelyProcessView(view.Job.Views[repeaterOp.Xref], scope)
				if repeaterOp.EmptyView != 0 {
					recursivelyProcessView(view.Job.Views[repeaterOp.EmptyView], scope)
				}
				if repeaterOp.TrackByOps != nil {
					newOps := generateVariablesInScopeForView(view, scope, false)
					ops := make([]operations.Op, len(newOps))
					for i, v := range newOps {
						ops[i] = v
					}
					repeaterOp.TrackByOps.Prepend(ops)
				}
			}
		case ir.OpKindAnimation, ir.OpKindAnimationListener, ir.OpKindListener, ir.OpKindTwoWayListener:
			// Prepend variables to listener handler functions.
			var handlerOps *operations.OpList
			switch opType := op.(type) {
			case *ops_create.AnimationOp:
				handlerOps = opType.HandlerOps
			case *ops_create.AnimationListenerOp:
				handlerOps = opType.HandlerOps
			case *ops_create.ListenerOp:
				handlerOps = opType.HandlerOps
			case *ops_create.TwoWayListenerOp:
				handlerOps = opType.HandlerOps
			}
			if handlerOps != nil {
				newOps := generateVariablesInScopeForView(view, scope, true)
				ops := make([]operations.Op, len(newOps))
				for i, v := range newOps {
					ops[i] = v
				}
				handlerOps.Prepend(ops)
			}
		}
	}

	newOps := generateVariablesInScopeForView(view, scope, false)
	ops := make([]operations.Op, len(newOps))
	for i, v := range newOps {
		ops[i] = v
	}
	view.Update.Prepend(ops)
}

// getScopeForView processes a view and generates a `Scope` representing the variables available for reference within
// that view.
func getScopeForView(view *pipeline.ViewCompilationUnit, parent *Scope) *Scope {
	scope := &Scope{
		View:                view.Xref,
		ViewContextVariable: ir_variable.NewContextVariable(view.Xref),
		ContextVariables:    make(map[string]ir_variable.SemanticVariable),
		Aliases:             make(map[ir_variable.AliasVariable]bool),
		References:          []Reference{},
		LetDeclarations:     []LetDeclaration{},
		Parent:              parent,
	}

	// Copy aliases from view
	for alias := range view.Aliases {
		scope.Aliases[alias] = true
	}

	for identifier, value := range view.ContextVariables {
		scope.ContextVariables[identifier] = ir_variable.NewIdentifierVariable(identifier, false)
		_ = value // value is stored but not used in scope
	}

	for op := view.Create.Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
		switch op.GetKind() {
		case ir.OpKindElementStart, ir.OpKindConditionalCreate, ir.OpKindConditionalBranchCreate, ir.OpKindTemplate:
			var localRefs []ops_create.LocalRef
			var ok bool
			switch opType := op.(type) {
			case *ops_create.ElementStartOp:
				if refs, ok2 := opType.LocalRefs.([]ops_create.LocalRef); ok2 {
					localRefs = refs
					ok = true
				}
			case *ops_create.ConditionalCreateOp:
				if refs, ok2 := opType.LocalRefs.([]ops_create.LocalRef); ok2 {
					localRefs = refs
					ok = true
				}
			case *ops_create.ConditionalBranchCreateOp:
				if refs, ok2 := opType.LocalRefs.([]ops_create.LocalRef); ok2 {
					localRefs = refs
					ok = true
				}
			case *ops_create.TemplateOp:
				if refs, ok2 := opType.LocalRefs.([]ops_create.LocalRef); ok2 {
					localRefs = refs
					ok = true
				}
			}

			if !ok {
				panic("AssertionError: expected localRefs to be an array")
			}

			// Type assert to CreateOp to access GetXref()
			createOp, ok := op.(operations.CreateOp)
			if !ok {
				continue
			}

			// Record available local references from this element.
			for offset := 0; offset < len(localRefs); offset++ {
				scope.References = append(scope.References, Reference{
					Name:       localRefs[offset].Name,
					TargetId:   createOp.GetXref(),
					TargetSlot: getHandle(op),
					Offset:     offset,
					Variable:   ir_variable.NewIdentifierVariable(localRefs[offset].Name, false),
				})
			}

		case ir.OpKindDeclareLet:
			declareLetOp, ok := op.(*ops_create.DeclareLetOp)
			if ok {
				scope.LetDeclarations = append(scope.LetDeclarations, LetDeclaration{
					TargetId:   declareLetOp.Xref,
					TargetSlot: declareLetOp.Handle,
					Variable:   ir_variable.NewIdentifierVariable(declareLetOp.DeclaredName, false),
				})
			}
		}
	}

	return scope
}

// getHandle extracts the handle from an operations
func getHandle(op operations.Op) *ir.SlotHandle {
	switch opType := op.(type) {
	case *ops_create.ElementStartOp:
		return opType.Handle
	case *ops_create.ConditionalCreateOp:
		return opType.Handle
	case *ops_create.ConditionalBranchCreateOp:
		return opType.Handle
	case *ops_create.TemplateOp:
		return opType.Handle
	default:
		return nil
	}
}

// generateVariablesInScopeForView generates declarations for all variables that are in scope for a given view.
// This is a recursive process, as views inherit variables available from their parent view, which
// itself may have inherited variables, etc.
func generateVariablesInScopeForView(
	view *pipeline.ViewCompilationUnit,
	scope *Scope,
	isCallback bool,
) []*shared.VariableOp {
	newOps := []*shared.VariableOp{}

	if scope.View != view.Xref {
		// Before generating variables for a parent view, we need to switch to the context of the parent
		// view with a `nextContext` expression. This context switching operations itself declares a
		// variable, because the context of the view may be referenced directly.
		newOps = append(newOps, shared.NewVariableOp(
			view.Job.AllocateXrefId(),
			scope.ViewContextVariable,
			expression.NewNextContextExpr(),
			ir.VariableFlagsNone,
		))
	}

	// Add variables for all context variables available in this scope's view.
	scopeView := view.Job.Views[scope.View]
	for name, value := range scopeView.ContextVariables {
		context := expression.NewContextExpr(scope.View)
		// We either read the context, or, if the variable is CTX_REF, use the context directly.
		var variable output.OutputExpression
		if value == ir_variable.CTX_REF {
			variable = context
		} else {
			variable = output.NewReadPropExpr(context, value, nil, nil)
		}
		// Add the variable declaration.
		newOps = append(newOps, shared.NewVariableOp(
			view.Job.AllocateXrefId(),
			scope.ContextVariables[name],
			variable,
			ir.VariableFlagsNone,
		))
	}

	for alias := range scopeView.Aliases {
		// Create a copy of alias to avoid modifying the map key
		aliasCopy := alias
		newOps = append(newOps, shared.NewVariableOp(
			view.Job.AllocateXrefId(),
			&aliasCopy,
			aliasCopy.Expression.Clone(),
			ir.VariableFlagsAlwaysInline,
		))
	}

	// Add variables for all local references declared for elements in this scope.
	for _, ref := range scope.References {
		newOps = append(newOps, shared.NewVariableOp(
			view.Job.AllocateXrefId(),
			ref.Variable,
			expression.NewReferenceExpr(ref.TargetId, ref.TargetSlot, ref.Offset),
			ir.VariableFlagsNone,
		))
	}

	if scope.View != view.Xref || isCallback {
		for _, decl := range scope.LetDeclarations {
			newOps = append(newOps, shared.NewVariableOp(
				view.Job.AllocateXrefId(),
				decl.Variable,
				expression.NewContextLetReferenceExpr(decl.TargetId, decl.TargetSlot),
				ir.VariableFlagsNone,
			))
		}
	}

	if scope.Parent != nil {
		// Recursively add variables from the parent scope.
		parentOps := generateVariablesInScopeForView(view, scope.Parent, false)
		newOps = append(newOps, parentOps...)
	}
	return newOps
}
