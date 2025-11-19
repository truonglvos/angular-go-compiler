package phases

import (
	ir "ngc-go/packages/compiler/template/pipeline/ir/src"
	ir_operation "ngc-go/packages/compiler/template/pipeline/ir/src/operations"
	ops_create "ngc-go/packages/compiler/template/pipeline/ir/src/ops/create"
	ops_update "ngc-go/packages/compiler/template/pipeline/ir/src/ops/update"
	ir_traits "ngc-go/packages/compiler/template/pipeline/ir/src/traits"

	pipeline "ngc-go/packages/compiler/template/pipeline/src/compilation"
)

// Rule represents a rule for ordering operations
type Rule struct {
	Test      func(op ir_operation.Op) bool
	Transform func(ops []ir_operation.Op) []ir_operation.Op
}

// kindTest creates a test function for a specific OpKind
func kindTest(kind ir.OpKind) func(op ir_operation.Op) bool {
	return func(op ir_operation.Op) bool {
		return op.GetKind() == kind
	}
}

// kindWithInterpolationTest creates a test function for ops with interpolation
func kindWithInterpolationTest(
	kind ir.OpKind,
	interpolation bool,
) func(op ir_operation.Op) bool {
	return func(op ir_operation.Op) bool {
		if op.GetKind() != kind {
			return false
		}

		// Check if expression is an Interpolation
		var expr interface{}
		switch updateOp := op.(type) {
		case *ops_update.PropertyOp:
			expr = updateOp.Expression
		case *ops_update.AttributeOp:
			expr = updateOp.Expression
		case *ops_update.DomPropertyOp:
			expr = updateOp.Expression
		default:
			return false
		}

		_, isInterpolation := expr.(*ops_update.Interpolation)
		return isInterpolation == interpolation
	}
}

// basicListenerKindTest tests if an op is a basic listener kind
func basicListenerKindTest(op ir_operation.Op) bool {
	kind := op.GetKind()
	if kind == ir.OpKindListener {
		if listenerOp, ok := op.(*ops_create.ListenerOp); ok {
			return !(listenerOp.HostListener && listenerOp.IsLegacyAnimationListener)
		}
	}
	return kind == ir.OpKindTwoWayListener ||
		kind == ir.OpKindAnimation ||
		kind == ir.OpKindAnimationListener
}

// nonInterpolationPropertyKindTest tests if an op is a non-interpolation property
func nonInterpolationPropertyKindTest(op ir_operation.Op) bool {
	kind := op.GetKind()
	if kind != ir.OpKindProperty && kind != ir.OpKindTwoWayProperty {
		return false
	}

	var expr interface{}
	if propertyOp, ok := op.(*ops_update.PropertyOp); ok {
		expr = propertyOp.Expression
	} else if twoWayOp, ok := op.(*ops_update.TwoWayPropertyOp); ok {
		expr = twoWayOp.Expression
	} else {
		return false
	}

	_, isInterpolation := expr.(*ops_update.Interpolation)
	return !isInterpolation
}

// CREATE_ORDERING defines the ordering rules for create operations
var CREATE_ORDERING = []Rule{
	{
		Test: func(op ir_operation.Op) bool {
			if op.GetKind() != ir.OpKindListener {
				return false
			}
			if listenerOp, ok := op.(*ops_create.ListenerOp); ok {
				return listenerOp.HostListener && listenerOp.IsLegacyAnimationListener
			}
			return false
		},
	},
	{Test: basicListenerKindTest},
}

// UPDATE_ORDERING defines the ordering rules for update operations
var UPDATE_ORDERING = []Rule{
	{Test: kindTest(ir.OpKindStyleMap), Transform: keepLast},
	{Test: kindTest(ir.OpKindClassMap), Transform: keepLast},
	{Test: kindTest(ir.OpKindStyleProp)},
	{Test: kindTest(ir.OpKindClassProp)},
	{Test: kindWithInterpolationTest(ir.OpKindAttribute, true)},
	{Test: kindWithInterpolationTest(ir.OpKindProperty, true)},
	{Test: nonInterpolationPropertyKindTest},
	{Test: kindWithInterpolationTest(ir.OpKindAttribute, false)},
}

// UPDATE_HOST_ORDERING defines the ordering rules for host binding update operations
var UPDATE_HOST_ORDERING = []Rule{
	{Test: kindWithInterpolationTest(ir.OpKindDomProperty, true)},
	{Test: kindWithInterpolationTest(ir.OpKindDomProperty, false)},
	{Test: kindTest(ir.OpKindAttribute)},
	{Test: kindTest(ir.OpKindStyleMap), Transform: keepLast},
	{Test: kindTest(ir.OpKindClassMap), Transform: keepLast},
	{Test: kindTest(ir.OpKindStyleProp)},
	{Test: kindTest(ir.OpKindClassProp)},
}

// handledOpKinds is the set of all op kinds we handle in the reordering phase
var handledOpKinds = map[ir.OpKind]bool{
	ir.OpKindListener:          true,
	ir.OpKindTwoWayListener:    true,
	ir.OpKindAnimationListener: true,
	ir.OpKindStyleMap:          true,
	ir.OpKindClassMap:          true,
	ir.OpKindStyleProp:         true,
	ir.OpKindClassProp:         true,
	ir.OpKindProperty:          true,
	ir.OpKindTwoWayProperty:    true,
	ir.OpKindDomProperty:       true,
	ir.OpKindAttribute:         true,
	ir.OpKindAnimation:         true,
}

// OrderOps orders operations according to their constraints.
// Many type of operations have ordering constraints that must be respected. For example, a
// `ClassMap` instruction must be ordered after a `StyleMap` instruction, in order to have
// predictable semantics that match TemplateDefinitionBuilder and don't break applications.
func OrderOps(job *pipeline.CompilationJob) {
	for _, unit := range job.GetUnits() {
		// Create mode:
		orderWithin(unit.GetCreate(), CREATE_ORDERING)

		// Update mode:
		var ordering []Rule
		if job.Kind == pipeline.CompilationJobKindHost {
			ordering = UPDATE_HOST_ORDERING
		} else {
			ordering = UPDATE_ORDERING
		}
		orderWithin(unit.GetUpdate(), ordering)
	}
}

// orderWithin orders all the ops within the specified group
func orderWithin(opList *ir_operation.OpList, ordering []Rule) {
	var opsToOrder []ir_operation.Op
	// Only reorder ops that target the same xref; do not mix ops that target different xrefs.
	var firstTargetInGroup *ir_operation.XrefId

	for op := opList.Head(); op != nil; op = op.Next() {
		if op.GetKind() == ir.OpKindListEnd {
			break
		}

		var currentTarget *ir_operation.XrefId
		if ir_traits.HasDependsOnSlotContextTrait(op) {
			if trait := getDependsOnSlotContextTrait(op); trait != nil {
				target := trait.Target
				currentTarget = &target
			}
		}

		shouldReorder := handledOpKinds[op.GetKind()]
		shouldBreak := !shouldReorder || (firstTargetInGroup != nil && currentTarget != nil && *currentTarget != *firstTargetInGroup)

		if shouldBreak {
			// Insert reordered ops before current op
			reordered := reorder(opsToOrder, ordering)
			for i := len(reordered) - 1; i >= 0; i-- {
				opList.InsertBefore(op, reordered[i])
			}
			opsToOrder = nil
			firstTargetInGroup = nil
		}

		if shouldReorder {
			opsToOrder = append(opsToOrder, op)
			opList.Remove(op)
			if firstTargetInGroup == nil {
				firstTargetInGroup = currentTarget
			}
		}
	}

	// Insert remaining reordered ops at the end
	if len(opsToOrder) > 0 {
		reordered := reorder(opsToOrder, ordering)
		for _, op := range reordered {
			opList.Push(op)
		}
	}
}

// reorder reorders the given list of ops according to the ordering defined by rules
func reorder(ops []ir_operation.Op, ordering []Rule) []ir_operation.Op {
	// Break the ops list into groups based on OpKind
	groups := make([][]ir_operation.Op, len(ordering))
	for i := range groups {
		groups[i] = []ir_operation.Op{}
	}

	for _, op := range ops {
		groupIndex := -1
		for i, rule := range ordering {
			if rule.Test(op) {
				groupIndex = i
				break
			}
		}
		if groupIndex >= 0 {
			groups[groupIndex] = append(groups[groupIndex], op)
		}
	}

	// Reassemble the groups into a single list, in the correct order
	var result []ir_operation.Op
	for i, group := range groups {
		transform := ordering[i].Transform
		if transform != nil {
			group = transform(group)
		}
		result = append(result, group...)
	}
	return result
}

// keepLast keeps only the last op in a list of ops
func keepLast(ops []ir_operation.Op) []ir_operation.Op {
	if len(ops) == 0 {
		return ops
	}
	return []ir_operation.Op{ops[len(ops)-1]}
}

// getDependsOnSlotContextTrait gets the DependsOnSlotContextOpTrait from an op
func getDependsOnSlotContextTrait(op ir_operation.Op) *ir_traits.DependsOnSlotContextOpTrait {
	if updateOp, ok := op.(interface {
		GetDependsOnSlotContextTrait() *ir_traits.DependsOnSlotContextOpTrait
	}); ok {
		return updateOp.GetDependsOnSlotContextTrait()
	}
	return nil
}
