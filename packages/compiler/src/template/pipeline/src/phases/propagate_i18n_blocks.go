package phases

import (
	"ngc-go/packages/compiler/src/template/pipeline/ir"
	ops_create "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/create"

	pipeline "ngc-go/packages/compiler/src/template/pipeline/src/compilation"
)

// PropagateI18nBlocks propagates i18n blocks down through child templates that act as placeholders in the root i18n
// message. Specifically, perform an in-order traversal of all the views, and add i18nStart/i18nEnd
// op pairs into descending views. Also, assign an increasing sub-template index to each
// descending view.
func PropagateI18nBlocks(job *pipeline.ComponentCompilationJob) {
	propagateI18nBlocksToTemplates(job.Root, 0)
}

// propagateI18nBlocksToTemplates propagates i18n ops in the given view through to any child views recursively.
func propagateI18nBlocksToTemplates(
	unit *pipeline.ViewCompilationUnit,
	subTemplateIndex int,
) int {
	var i18nBlock *ops_create.I18nStartOp = nil
	for op := unit.GetCreate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
		switch op.GetKind() {
		case ir.OpKindI18nStart:
			i18nStartOp, ok := op.(*ops_create.I18nStartOp)
			if !ok {
				continue
			}
			if subTemplateIndex == 0 {
				i18nStartOp.SubTemplateIndex = nil
			} else {
				i18nStartOp.SubTemplateIndex = &subTemplateIndex
			}
			i18nBlock = i18nStartOp
		case ir.OpKindI18nEnd:
			// When we exit a root-level i18n block, reset the sub-template index counter.
			if i18nBlock != nil && i18nBlock.SubTemplateIndex == nil {
				subTemplateIndex = 0
			}
			i18nBlock = nil
		case ir.OpKindConditionalCreate:
			conditionalOp, ok := op.(*ops_create.ConditionalCreateOp)
			if !ok {
				continue
			}
			view, exists := unit.Job.Views[conditionalOp.Xref]
			if !exists {
				continue
			}
			subTemplateIndex = propagateI18nBlocksForView(
				view,
				i18nBlock,
				conditionalOp.I18nPlaceholder,
				subTemplateIndex,
			)
		case ir.OpKindConditionalBranchCreate:
			conditionalBranchOp, ok := op.(*ops_create.ConditionalBranchCreateOp)
			if !ok {
				continue
			}
			view, exists := unit.Job.Views[conditionalBranchOp.Xref]
			if !exists {
				continue
			}
			subTemplateIndex = propagateI18nBlocksForView(
				view,
				i18nBlock,
				conditionalBranchOp.I18nPlaceholder,
				subTemplateIndex,
			)
		case ir.OpKindTemplate:
			templateOp, ok := op.(*ops_create.TemplateOp)
			if !ok {
				continue
			}
			view, exists := unit.Job.Views[templateOp.Xref]
			if !exists {
				continue
			}
			subTemplateIndex = propagateI18nBlocksForView(
				view,
				i18nBlock,
				templateOp.I18nPlaceholder,
				subTemplateIndex,
			)
		case ir.OpKindRepeaterCreate:
			repeaterOp, ok := op.(*ops_create.RepeaterCreateOp)
			if !ok {
				continue
			}
			// Propagate i18n blocks to the @for template.
			forView, exists := unit.Job.Views[repeaterOp.Xref]
			if exists {
				subTemplateIndex = propagateI18nBlocksForView(
					forView,
					i18nBlock,
					repeaterOp.I18nPlaceholder,
					subTemplateIndex,
				)
			}
			// Then if there's an @empty template, propagate the i18n blocks for it as well.
			if repeaterOp.EmptyView != 0 {
				emptyView, exists := unit.Job.Views[repeaterOp.EmptyView]
				if exists {
					subTemplateIndex = propagateI18nBlocksForView(
						emptyView,
						i18nBlock,
						repeaterOp.EmptyI18nPlaceholder,
						subTemplateIndex,
					)
				}
			}
		case ir.OpKindProjection:
			projectionOp, ok := op.(*ops_create.ProjectionOp)
			if !ok {
				continue
			}
			if projectionOp.FallbackView != 0 {
				fallbackView, exists := unit.Job.Views[projectionOp.FallbackView]
				if exists {
					subTemplateIndex = propagateI18nBlocksForView(
						fallbackView,
						i18nBlock,
						projectionOp.FallbackViewI18nPlaceholder,
						subTemplateIndex,
					)
				}
			}
		}
	}
	return subTemplateIndex
}

// propagateI18nBlocksForView propagates i18n blocks for a view.
func propagateI18nBlocksForView(
	view *pipeline.ViewCompilationUnit,
	i18nBlock *ops_create.I18nStartOp,
	i18nPlaceholder interface{}, // *i18n.TagPlaceholder | *i18n.BlockPlaceholder
	subTemplateIndex int,
) int {
	// We found an <ng-template> inside an i18n block; increment the sub-template counter and
	// wrap the template's view in a child i18n block.
	if i18nPlaceholder != nil {
		if i18nBlock == nil {
			panic("Expected template with i18n placeholder to be in an i18n block.")
		}
		subTemplateIndex++
		wrapTemplateWithI18n(view, i18nBlock)
	}

	// Continue traversing inside the template's view.
	return propagateI18nBlocksToTemplates(view, subTemplateIndex)
}

// wrapTemplateWithI18n wraps a template view with i18n start and end ops.
func wrapTemplateWithI18n(unit *pipeline.ViewCompilationUnit, parentI18n *ops_create.I18nStartOp) {
	// Only add i18n ops if they have not already been propagated to this template.
	head := unit.GetCreate().Head()
	if head == nil || head.GetKind() == ir.OpKindListEnd {
		return
	}
	nextOp := head.Next()
	if nextOp == nil || nextOp.GetKind() != ir.OpKindI18nStart {
		id := unit.Job.AllocateXrefId()
		// Nested ng-template i18n start/end ops should not receive source spans.
		i18nStartOp := ops_create.NewI18nStartOp(id, parentI18n.Message, parentI18n.Root, nil)
		unit.GetCreate().InsertAfter(head, i18nStartOp)
		tail := unit.GetCreate().Tail()
		if tail != nil {
			i18nEndOp := ops_create.NewI18nEndOp(id, nil)
			unit.GetCreate().InsertBefore(tail, i18nEndOp)
		}
	}
}
