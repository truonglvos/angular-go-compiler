package phases

import (
	"ngc-go/packages/compiler/src/template/pipeline/ir"
	"ngc-go/packages/compiler/src/template/pipeline/ir/src/operations"
	ops_create "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/create"
	ir_traits "ngc-go/packages/compiler/src/template/pipeline/ir/src/traits"

	pipeline "ngc-go/packages/compiler/src/template/pipeline/src/compilation"
)

// ScopeDefer represents a scope for defer target resolution
type ScopeDefer struct {
	Targets map[string]TargetInfo
}

// TargetInfo contains information about a target
type TargetInfo struct {
	Xref operations.XrefId
	Slot *ir.SlotHandle
}

// ResolveDeferTargetNames resolves all defer target references to actual xrefs.
// Some `defer` conditions can reference other elements in the template, using their local reference
// names. However, the semantics are quite different from the normal local reference system: in
// particular, we need to look at local reference names in enclosing views.
func ResolveDeferTargetNames(job *pipeline.ComponentCompilationJob) {
	scopes := make(map[operations.XrefId]*ScopeDefer)

	getScopeForView := func(view *pipeline.ViewCompilationUnit) *ScopeDefer {
		if scope, exists := scopes[view.Xref]; exists {
			return scope
		}

		scope := &ScopeDefer{
			Targets: make(map[string]TargetInfo),
		}
		for op := view.Create.Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {

			createOp, convertOk := op.(operations.CreateOp)
			if !convertOk {
				continue
			}
			// add everything that can be referenced.
			if !ops_create.IsElementOrContainerOp(createOp) {
				continue
			}
			var localRefs []ops_create.LocalRef
			var handle *ir.SlotHandle
			var ok bool
			switch opType := op.(type) {
			case *ops_create.ElementStartOp:
				if refs, ok2 := opType.LocalRefs.([]ops_create.LocalRef); ok2 {
					localRefs = refs
					handle = opType.Handle
					ok = true
				}
			case *ops_create.ElementOp:
				if refs, ok2 := opType.LocalRefs.([]ops_create.LocalRef); ok2 {
					localRefs = refs
					handle = opType.Handle
					ok = true
				}
			case *ops_create.ContainerStartOp:
				if refs, ok2 := opType.LocalRefs.([]ops_create.LocalRef); ok2 {
					localRefs = refs
					handle = opType.Handle
					ok = true
				}
			case *ops_create.ContainerOp:
				if refs, ok2 := opType.LocalRefs.([]ops_create.LocalRef); ok2 {
					localRefs = refs
					handle = opType.Handle
					ok = true
				}
			case *ops_create.TemplateOp:
				if refs, ok2 := opType.LocalRefs.([]ops_create.LocalRef); ok2 {
					localRefs = refs
					handle = opType.Handle
					ok = true
				}
			}

			if !ok {
				panic("LocalRefs were already processed, but were needed to resolve defer targets.")
			}

			for _, ref := range localRefs {
				if ref.Target != "" {
					continue
				}
				// Type assert to CreateOp to access GetXref()
				createOp, ok := op.(operations.CreateOp)
				if !ok {
					continue
				}
				scope.Targets[ref.Name] = TargetInfo{
					Xref: createOp.GetXref(),
					Slot: handle,
				}
			}
		}

		scopes[view.Xref] = scope
		return scope
	}

	resolveTrigger := func(
		deferOwnerView *pipeline.ViewCompilationUnit,
		onOp *ops_create.DeferOnOp,
		placeholderView operations.XrefId,
	) {
		trigger := onOp.Trigger
		switch trigger.GetKind() {
		case ir.DeferTriggerKindIdle, ir.DeferTriggerKindNever, ir.DeferTriggerKindImmediate, ir.DeferTriggerKindTimer:
			return
		case ir.DeferTriggerKindHover, ir.DeferTriggerKindInteraction, ir.DeferTriggerKindViewport:
			var triggerWithTarget *ops_create.DeferTriggerWithTargetBase
			switch t := trigger.(type) {
			case *ops_create.DeferHoverTrigger:
				triggerWithTarget = &t.DeferTriggerWithTargetBase
			case *ops_create.DeferInteractionTrigger:
				triggerWithTarget = &t.DeferTriggerWithTargetBase
			case *ops_create.DeferViewportTrigger:
				triggerWithTarget = &t.DeferTriggerWithTargetBase
			default:
				return
			}
			if triggerWithTarget.TargetName == nil {
				// A `nil` target name indicates we should default to the first element in the
				// placeholder block.
				if placeholderView == 0 {
					panic("defer on trigger with no target name must have a placeholder block")
				}
				placeholder, exists := job.Views[placeholderView]
				if !exists {
					panic("AssertionError: could not find placeholder view for defer on trigger")
				}
				for op := placeholder.Create.Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
					if ir_traits.HasConsumesSlotTrait(op) {
						createOp, ok := op.(operations.CreateOp)
						if !ok {
							continue
						}
						if ops_create.IsElementOrContainerOp(createOp) || createOp.GetKind() == ir.OpKindProjection {
							// Type assert to CreateOp to access GetXref()
							triggerWithTarget.TargetXref = createOp.GetXref()
							triggerWithTarget.TargetView = placeholderView
							step := -1
							triggerWithTarget.TargetSlotViewSteps = &step
							if slotOp, ok := op.(interface {
								GetConsumesSlotTrait() *ir_traits.ConsumesSlotOpTrait
							}); ok {
								triggerWithTarget.TargetSlot = slotOp.GetConsumesSlotTrait().Handle
							}
							return
						}
					}
				}
				return
			}
			view := deferOwnerView
			step := 0
			if placeholderView != 0 {
				view = job.Views[placeholderView]
				step = -1
			}

			for view != nil {
				scope := getScopeForView(view)
				if targetInfo, exists := scope.Targets[*triggerWithTarget.TargetName]; exists {
					triggerWithTarget.TargetXref = targetInfo.Xref
					triggerWithTarget.TargetView = view.Xref
					triggerWithTarget.TargetSlotViewSteps = &step
					triggerWithTarget.TargetSlot = targetInfo.Slot
					return
				}

				if view.Parent != nil {
					view = job.Views[*view.Parent]
					step++
				} else {
					view = nil
				}
			}
		default:
			panic("Trigger kind not handled")
		}
	}

	// Find the defer ops, and assign the data about their targets.
	for _, unit := range job.GetUnits() {
		viewUnit, ok := unit.(*pipeline.ViewCompilationUnit)
		if !ok {
			continue
		}
		defers := make(map[operations.XrefId]*ops_create.DeferOp)
		for op := viewUnit.Create.Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			switch op.GetKind() {
			case ir.OpKindDefer:
				deferOp, ok := op.(*ops_create.DeferOp)
				if ok {
					defers[deferOp.Xref] = deferOp
				}
			case ir.OpKindDeferOn:
				onOp, ok := op.(*ops_create.DeferOnOp)
				if ok {
					deferOp := defers[onOp.Defer]
					if deferOp == nil {
						continue
					}
					placeholderView := deferOp.PlaceholderView
					if onOp.Modifier == ir.DeferOpModifierKindHydrate {
						placeholderView = deferOp.MainView
					}
					resolveTrigger(viewUnit, onOp, placeholderView)
				}
			}
		}
	}
}
