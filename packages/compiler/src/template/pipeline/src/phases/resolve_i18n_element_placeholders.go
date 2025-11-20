package phases

import (
	"ngc-go/packages/compiler/src/i18n"
	"ngc-go/packages/compiler/src/template/pipeline/ir"
	ir_operations "ngc-go/packages/compiler/src/template/pipeline/ir/src/operations"
	ops_create "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/create"

	pipeline "ngc-go/packages/compiler/src/template/pipeline/src/compilation"
)

// ResolveI18nElementPlaceholders resolves the element placeholders in i18n messages.
func ResolveI18nElementPlaceholders(job *pipeline.ComponentCompilationJob) {
	// Record all of the element and i18n context ops for use later.
	i18nContexts := make(map[ir_operations.XrefId]*ops_create.I18nContextOp)
	elements := make(map[ir_operations.XrefId]*ops_create.ElementStartOp)
	for _, unit := range job.GetUnits() {
		for op := unit.GetCreate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			switch op.GetKind() {
			case ir.OpKindI18nContext:
				i18nContextOp, ok := op.(*ops_create.I18nContextOp)
				if ok {
					i18nContexts[i18nContextOp.Xref] = i18nContextOp
				}
			case ir.OpKindElementStart:
				elementStartOp, ok := op.(*ops_create.ElementStartOp)
				if ok {
					elements[elementStartOp.Xref] = elementStartOp
				}
			}
		}
	}

	resolvePlaceholdersForView(job, job.Root, i18nContexts, elements, nil)
}

// CurrentOps tracks the current i18n block and context
type CurrentOps struct {
	I18nBlock   *ops_create.I18nStartOp
	I18nContext *ops_create.I18nContextOp
}

// resolvePlaceholdersForView recursively resolves element and template tag placeholders in the given view.
func resolvePlaceholdersForView(
	job *pipeline.ComponentCompilationJob,
	unit *pipeline.ViewCompilationUnit,
	i18nContexts map[ir_operations.XrefId]*ops_create.I18nContextOp,
	elements map[ir_operations.XrefId]*ops_create.ElementStartOp,
	pendingStructuralDirective interface{}, // *ops.TemplateOp | *ops.ConditionalCreateOp | *ops.ConditionalBranchCreateOp
) {
	// Track the current i18n op and corresponding i18n context op as we step through the creation IR.
	var currentOps *CurrentOps
	pendingStructuralDirectiveCloses := make(map[ir_operations.XrefId]interface{}) // *ops.TemplateOp | *ops.ConditionalCreateOp | *ops.ConditionalBranchCreateOp

	for op := unit.Create.Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
		switch op.GetKind() {
		case ir.OpKindI18nStart:
			i18nStartOp, ok := op.(*ops_create.I18nStartOp)
			if !ok {
				continue
			}
			if i18nStartOp.Context == 0 {
				panic("Could not find i18n context for i18n op")
			}
			i18nContext, exists := i18nContexts[i18nStartOp.Context]
			if !exists {
				panic("Could not find i18n context")
			}
			currentOps = &CurrentOps{
				I18nBlock:   i18nStartOp,
				I18nContext: i18nContext,
			}
		case ir.OpKindI18nEnd:
			currentOps = nil
		case ir.OpKindElementStart:
			elementStartOp, ok := op.(*ops_create.ElementStartOp)
			if !ok {
				continue
			}
			// For elements with i18n placeholders, record its slot value in the params map under the
			// corresponding tag start placeholder.
			if elementStartOp.I18nPlaceholder != nil {
				if currentOps == nil {
					panic("i18n tag placeholder should only occur inside an i18n block")
				}
				recordElementStart(
					elementStartOp,
					currentOps.I18nContext,
					currentOps.I18nBlock,
					pendingStructuralDirective,
				)
				// If there is a separate close tag placeholder for this element, save the pending
				// structural directive so we can pass it to the closing tag as well.
				if tagPlaceholder, ok := elementStartOp.I18nPlaceholder.(*i18n.TagPlaceholder); ok {
					if pendingStructuralDirective != nil && tagPlaceholder.CloseName != "" {
						pendingStructuralDirectiveCloses[elementStartOp.Xref] = pendingStructuralDirective
					}
				}
				// Clear out the pending structural directive now that its been accounted for.
				pendingStructuralDirective = nil
			}
		case ir.OpKindElementEnd:
			elementEndOp, ok := op.(*ops_create.ElementEndOp)
			if !ok {
				continue
			}
			// For elements with i18n placeholders, record its slot value in the params map under the
			// corresponding tag close placeholder.
			startOp := elements[elementEndOp.Xref]
			if startOp != nil && startOp.I18nPlaceholder != nil {
				if currentOps == nil {
					panic("AssertionError: i18n tag placeholder should only occur inside an i18n block")
				}
				recordElementClose(
					startOp,
					currentOps.I18nContext,
					currentOps.I18nBlock,
					pendingStructuralDirectiveCloses[elementEndOp.Xref],
				)
				// Clear out the pending structural directive close that was accounted for.
				delete(pendingStructuralDirectiveCloses, elementEndOp.Xref)
			}
		case ir.OpKindProjection:
			projectionOp, ok := op.(*ops_create.ProjectionOp)
			if !ok {
				continue
			}
			// For content projections with i18n placeholders, record its slot value in the params map
			// under the corresponding tag start and close placeholders.
			if projectionOp.I18nPlaceholder != nil {
				if currentOps == nil {
					panic("i18n tag placeholder should only occur inside an i18n block")
				}
				recordElementStart(
					projectionOp,
					currentOps.I18nContext,
					currentOps.I18nBlock,
					pendingStructuralDirective,
				)
				recordElementClose(
					projectionOp,
					currentOps.I18nContext,
					currentOps.I18nBlock,
					pendingStructuralDirective,
				)
				// Clear out the pending structural directive now that its been accounted for.
				pendingStructuralDirective = nil
			}

			if projectionOp.FallbackView != 0 {
				view := job.Views[projectionOp.FallbackView]
				if view == nil {
					continue
				}
				if projectionOp.FallbackViewI18nPlaceholder == nil {
					resolvePlaceholdersForView(job, view, i18nContexts, elements, nil)
				} else {
					if currentOps == nil {
						panic("i18n tag placeholder should only occur inside an i18n block")
					}
					slot := 0
					if projectionOp.Handle != nil && projectionOp.Handle.Slot != nil {
						slot = *projectionOp.Handle.Slot
					}
					recordTemplateStart(
						job,
						view,
						slot,
						projectionOp.FallbackViewI18nPlaceholder,
						currentOps.I18nContext,
						currentOps.I18nBlock,
						pendingStructuralDirective,
					)
					resolvePlaceholdersForView(job, view, i18nContexts, elements, nil)
					recordTemplateClose(
						job,
						view,
						slot,
						projectionOp.FallbackViewI18nPlaceholder,
						currentOps.I18nContext,
						currentOps.I18nBlock,
						pendingStructuralDirective,
					)
					pendingStructuralDirective = nil
				}
			}
		case ir.OpKindConditionalCreate, ir.OpKindConditionalBranchCreate, ir.OpKindTemplate:
			var templateOp interface{} // *ops.TemplateOp | *ops.ConditionalCreateOp | *ops.ConditionalBranchCreateOp
			var view *pipeline.ViewCompilationUnit
			var i18nPlaceholder interface{} // *i18n.TagPlaceholder | *i18n.BlockPlaceholder
			var templateKind ir.TemplateKind
			var handle *ir.SlotHandle

			switch opType := op.(type) {
			case *ops_create.TemplateOp:
				templateOp = opType
				view = job.Views[opType.Xref]
				i18nPlaceholder = opType.I18nPlaceholder
				templateKind = opType.TemplateKind
				handle = opType.Handle
			case *ops_create.ConditionalCreateOp:
				templateOp = opType
				view = job.Views[opType.Xref]
				i18nPlaceholder = opType.I18nPlaceholder
				templateKind = opType.TemplateKind
				handle = opType.Handle
			case *ops_create.ConditionalBranchCreateOp:
				templateOp = opType
				view = job.Views[opType.Xref]
				i18nPlaceholder = opType.I18nPlaceholder
				templateKind = opType.TemplateKind
				handle = opType.Handle
			default:
				continue
			}

			if view == nil {
				continue
			}

			if i18nPlaceholder == nil {
				// If there is no i18n placeholder, just recurse into the view in case it contains i18n
				// blocks.
				resolvePlaceholdersForView(job, view, i18nContexts, elements, nil)
			} else {
				if currentOps == nil {
					panic("i18n tag placeholder should only occur inside an i18n block")
				}
				if templateKind == ir.TemplateKindStructural {
					// If this is a structural directive template, don't record anything yet. Instead pass
					// the current template as a pending structural directive to be recorded when we find
					// the element, content, or template it belongs to. This allows us to create combined
					// values that represent, e.g. the start of a template and element at the same time.
					resolvePlaceholdersForView(job, view, i18nContexts, elements, templateOp)
				} else {
					// If this is some other kind of template, we can record its start, recurse into its
					// view, and then record its end.
					slot := 0
					if handle != nil && handle.Slot != nil {
						slot = *handle.Slot
					}
					recordTemplateStart(
						job,
						view,
						slot,
						i18nPlaceholder,
						currentOps.I18nContext,
						currentOps.I18nBlock,
						pendingStructuralDirective,
					)
					resolvePlaceholdersForView(job, view, i18nContexts, elements, nil)
					recordTemplateClose(
						job,
						view,
						slot,
						i18nPlaceholder,
						currentOps.I18nContext,
						currentOps.I18nBlock,
						pendingStructuralDirective,
					)
					pendingStructuralDirective = nil
				}
			}
		case ir.OpKindRepeaterCreate:
			repeaterOp, ok := op.(*ops_create.RepeaterCreateOp)
			if !ok {
				continue
			}
			if pendingStructuralDirective != nil {
				panic("AssertionError: Unexpected structural directive associated with @for block")
			}
			// RepeaterCreate has 3 slots: the first is for the op itself, the second is for the @for
			// template and the (optional) third is for the @empty template.
			forSlot := 0
			if repeaterOp.Handle != nil && repeaterOp.Handle.Slot != nil {
				forSlot = *repeaterOp.Handle.Slot + 1
			}
			forView := job.Views[repeaterOp.Xref]
			if forView == nil {
				continue
			}
			// First record all of the placeholders for the @for template.
			if repeaterOp.I18nPlaceholder == nil {
				// If there is no i18n placeholder, just recurse into the view in case it contains i18n
				// blocks.
				resolvePlaceholdersForView(job, forView, i18nContexts, elements, nil)
			} else {
				if currentOps == nil {
					panic("i18n tag placeholder should only occur inside an i18n block")
				}
				recordTemplateStart(
					job,
					forView,
					forSlot,
					repeaterOp.I18nPlaceholder,
					currentOps.I18nContext,
					currentOps.I18nBlock,
					pendingStructuralDirective,
				)
				resolvePlaceholdersForView(job, forView, i18nContexts, elements, nil)
				recordTemplateClose(
					job,
					forView,
					forSlot,
					repeaterOp.I18nPlaceholder,
					currentOps.I18nContext,
					currentOps.I18nBlock,
					pendingStructuralDirective,
				)
				pendingStructuralDirective = nil
			}
			// Then if there's an @empty template, add its placeholders as well.
			if repeaterOp.EmptyView != 0 {
				// RepeaterCreate has 3 slots: the first is for the op itself, the second is for the @for
				// template and the (optional) third is for the @empty template.
				emptySlot := 0
				if repeaterOp.Handle != nil && repeaterOp.Handle.Slot != nil {
					emptySlot = *repeaterOp.Handle.Slot + 2
				}
				emptyView := job.Views[repeaterOp.EmptyView]
				if emptyView == nil {
					continue
				}
				if repeaterOp.EmptyI18nPlaceholder == nil {
					// If there is no i18n placeholder, just recurse into the view in case it contains i18n
					// blocks.
					resolvePlaceholdersForView(job, emptyView, i18nContexts, elements, nil)
				} else {
					if currentOps == nil {
						panic("i18n tag placeholder should only occur inside an i18n block")
					}
					recordTemplateStart(
						job,
						emptyView,
						emptySlot,
						repeaterOp.EmptyI18nPlaceholder,
						currentOps.I18nContext,
						currentOps.I18nBlock,
						pendingStructuralDirective,
					)
					resolvePlaceholdersForView(job, emptyView, i18nContexts, elements, nil)
					recordTemplateClose(
						job,
						emptyView,
						emptySlot,
						repeaterOp.EmptyI18nPlaceholder,
						currentOps.I18nContext,
						currentOps.I18nBlock,
						pendingStructuralDirective,
					)
					pendingStructuralDirective = nil
				}
			}
		}
	}
}

// recordElementStart records an i18n param value for the start of an element.
func recordElementStart(
	op interface{}, // *ops_create.ElementStartOp | *ops_create.ProjectionOp
	i18nContext *ops_create.I18nContextOp,
	i18nBlock *ops_create.I18nStartOp,
	structuralDirective interface{}, // *ops_create.TemplateOp | *ops_create.ConditionalCreateOp | *ops_create.ConditionalBranchCreateOp
) {
	var i18nPlaceholder *i18n.TagPlaceholder
	var slot int
	var handle *ir.SlotHandle

	switch opType := op.(type) {
	case *ops_create.ElementStartOp:
		if tagPlaceholder, ok := opType.I18nPlaceholder.(*i18n.TagPlaceholder); ok {
			i18nPlaceholder = tagPlaceholder
		} else {
			return
		}
		handle = opType.Handle
	case *ops_create.ProjectionOp:
		if tagPlaceholder, ok := opType.I18nPlaceholder.(*i18n.TagPlaceholder); ok {
			i18nPlaceholder = tagPlaceholder
		} else {
			return
		}
		handle = opType.Handle
	default:
		return
	}

	if handle == nil || handle.Slot == nil {
		return
	}
	slot = *handle.Slot

	startName := i18nPlaceholder.StartName
	closeName := i18nPlaceholder.CloseName
	flags := ir.I18nParamValueFlagsElementTag | ir.I18nParamValueFlagsOpenTag
	var value interface{} // string | int | struct{Element int; Template int}
	value = slot

	// If the element is associated with a structural directive, start it as well.
	if structuralDirective != nil {
		flags |= ir.I18nParamValueFlagsTemplateTag
		var structSlot int
		switch structType := structuralDirective.(type) {
		case *ops_create.TemplateOp:
			if structType.Handle != nil && structType.Handle.Slot != nil {
				structSlot = *structType.Handle.Slot
			}
		case *ops_create.ConditionalCreateOp:
			if structType.Handle != nil && structType.Handle.Slot != nil {
				structSlot = *structType.Handle.Slot
			}
		case *ops_create.ConditionalBranchCreateOp:
			if structType.Handle != nil && structType.Handle.Slot != nil {
				structSlot = *structType.Handle.Slot
			}
		}
		value = struct {
			Element  int
			Template int
		}{Element: slot, Template: structSlot}
	}

	// For self-closing tags, there is no close tag placeholder. Instead, the start tag
	// placeholder accounts for the start and close of the element.
	if closeName == "" {
		flags |= ir.I18nParamValueFlagsCloseTag
	}

	subTemplateIndex := i18nBlock.SubTemplateIndex
	addParam(i18nContext.Params, startName, value, subTemplateIndex, flags)
}

// recordElementClose records an i18n param value for the closing of an element.
func recordElementClose(
	op interface{}, // *ops_create.ElementStartOp | *ops_create.ProjectionOp
	i18nContext *ops_create.I18nContextOp,
	i18nBlock *ops_create.I18nStartOp,
	structuralDirective interface{}, // *ops_create.TemplateOp | *ops_create.ConditionalCreateOp | *ops_create.ConditionalBranchCreateOp
) {
	var i18nPlaceholder *i18n.TagPlaceholder
	var slot int
	var handle *ir.SlotHandle

	switch opType := op.(type) {
	case *ops_create.ElementStartOp:
		if tagPlaceholder, ok := opType.I18nPlaceholder.(*i18n.TagPlaceholder); ok {
			i18nPlaceholder = tagPlaceholder
		} else {
			return
		}
		handle = opType.Handle
	case *ops_create.ProjectionOp:
		if tagPlaceholder, ok := opType.I18nPlaceholder.(*i18n.TagPlaceholder); ok {
			i18nPlaceholder = tagPlaceholder
		} else {
			return
		}
		handle = opType.Handle
	default:
		return
	}

	if handle == nil || handle.Slot == nil {
		return
	}
	slot = *handle.Slot

	closeName := i18nPlaceholder.CloseName
	// Self-closing tags don't have a closing tag placeholder, instead the element closing is
	// recorded via an additional flag on the element start value.
	if closeName != "" {
		flags := ir.I18nParamValueFlagsElementTag | ir.I18nParamValueFlagsCloseTag
		var value interface{} // string | int | struct{Element int; Template int}
		value = slot
		// If the element is associated with a structural directive, close it as well.
		if structuralDirective != nil {
			flags |= ir.I18nParamValueFlagsTemplateTag
			var structSlot int
			switch structType := structuralDirective.(type) {
			case *ops_create.TemplateOp:
				if structType.Handle != nil && structType.Handle.Slot != nil {
					structSlot = *structType.Handle.Slot
				}
			case *ops_create.ConditionalCreateOp:
				if structType.Handle != nil && structType.Handle.Slot != nil {
					structSlot = *structType.Handle.Slot
				}
			case *ops_create.ConditionalBranchCreateOp:
				if structType.Handle != nil && structType.Handle.Slot != nil {
					structSlot = *structType.Handle.Slot
				}
			}
			value = struct {
				Element  int
				Template int
			}{Element: slot, Template: structSlot}
		}
		subTemplateIndex := i18nBlock.SubTemplateIndex
		addParam(i18nContext.Params, closeName, value, subTemplateIndex, flags)
	}
}

// recordTemplateStart records an i18n param value for the start of a template.
func recordTemplateStart(
	job *pipeline.ComponentCompilationJob,
	view *pipeline.ViewCompilationUnit,
	slot int,
	i18nPlaceholder interface{}, // *i18n.TagPlaceholder | *i18n.BlockPlaceholder
	i18nContext *ops_create.I18nContextOp,
	i18nBlock *ops_create.I18nStartOp,
	structuralDirective interface{}, // *ops.TemplateOp | *ops.ConditionalCreateOp | *ops.ConditionalBranchCreateOp
) {
	var startName, closeName string
	switch ph := i18nPlaceholder.(type) {
	case *i18n.TagPlaceholder:
		startName = ph.StartName
		closeName = ph.CloseName
	case *i18n.BlockPlaceholder:
		startName = ph.StartName
		closeName = ph.CloseName
	default:
		return
	}

	flags := ir.I18nParamValueFlagsTemplateTag | ir.I18nParamValueFlagsOpenTag
	// For self-closing tags, there is no close tag placeholder. Instead, the start tag
	// placeholder accounts for the start and close of the element.
	if closeName == "" {
		flags |= ir.I18nParamValueFlagsCloseTag
	}

	// If the template is associated with a structural directive, record the structural directive's
	// start first. Since this template must be in the structural directive's view, we can just
	// directly use the current i18n block's sub-template index.
	if structuralDirective != nil {
		var structSlot int
		switch structType := structuralDirective.(type) {
		case *ops_create.TemplateOp:
			if structType.Handle != nil && structType.Handle.Slot != nil {
				structSlot = *structType.Handle.Slot
			}
		case *ops_create.ConditionalCreateOp:
			if structType.Handle != nil && structType.Handle.Slot != nil {
				structSlot = *structType.Handle.Slot
			}
		case *ops_create.ConditionalBranchCreateOp:
			if structType.Handle != nil && structType.Handle.Slot != nil {
				structSlot = *structType.Handle.Slot
			}
		}
		addParam(
			i18nContext.Params,
			startName,
			structSlot,
			i18nBlock.SubTemplateIndex,
			flags,
		)
	}

	// Record the start of the template. For the sub-template index, pass the index for the template's
	// view, rather than the current i18n block's index.
	subTemplateIndex := getSubTemplateIndexForTemplateTag(job, i18nBlock, view)
	addParam(
		i18nContext.Params,
		startName,
		slot,
		subTemplateIndex,
		flags,
	)
}

// recordTemplateClose records an i18n param value for the closing of a template.
func recordTemplateClose(
	job *pipeline.ComponentCompilationJob,
	view *pipeline.ViewCompilationUnit,
	slot int,
	i18nPlaceholder interface{}, // *i18n.TagPlaceholder | *i18n.BlockPlaceholder
	i18nContext *ops_create.I18nContextOp,
	i18nBlock *ops_create.I18nStartOp,
	structuralDirective interface{}, // *ops.TemplateOp | *ops.ConditionalCreateOp | *ops.ConditionalBranchCreateOp
) {
	var closeName string
	switch ph := i18nPlaceholder.(type) {
	case *i18n.TagPlaceholder:
		closeName = ph.CloseName
	case *i18n.BlockPlaceholder:
		closeName = ph.CloseName
	default:
		return
	}

	flags := ir.I18nParamValueFlagsTemplateTag | ir.I18nParamValueFlagsCloseTag
	// Self-closing tags don't have a closing tag placeholder, instead the template's closing is
	// recorded via an additional flag on the template start value.
	if closeName != "" {
		// Record the closing of the template. For the sub-template index, pass the index for the
		// template's view, rather than the current i18n block's index.
		subTemplateIndex := getSubTemplateIndexForTemplateTag(job, i18nBlock, view)
		addParam(
			i18nContext.Params,
			closeName,
			slot,
			subTemplateIndex,
			flags,
		)
		// If the template is associated with a structural directive, record the structural directive's
		// closing after. Since this template must be in the structural directive's view, we can just
		// directly use the current i18n block's sub-template index.
		if structuralDirective != nil {
			var structSlot int
			switch structType := structuralDirective.(type) {
			case *ops_create.TemplateOp:
				if structType.Handle != nil && structType.Handle.Slot != nil {
					structSlot = *structType.Handle.Slot
				}
			case *ops_create.ConditionalCreateOp:
				if structType.Handle != nil && structType.Handle.Slot != nil {
					structSlot = *structType.Handle.Slot
				}
			case *ops_create.ConditionalBranchCreateOp:
				if structType.Handle != nil && structType.Handle.Slot != nil {
					structSlot = *structType.Handle.Slot
				}
			}
			addParam(
				i18nContext.Params,
				closeName,
				structSlot,
				i18nBlock.SubTemplateIndex,
				flags,
			)
		}
	}
}

// getSubTemplateIndexForTemplateTag gets the subTemplateIndex for the given template op.
// For template ops, use the subTemplateIndex of the child i18n block inside the template.
func getSubTemplateIndexForTemplateTag(
	job *pipeline.ComponentCompilationJob,
	i18nOp *ops_create.I18nStartOp,
	view *pipeline.ViewCompilationUnit,
) *int {
	for op := view.Create.Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
		if op.GetKind() == ir.OpKindI18nStart {
			i18nStartOp, ok := op.(*ops_create.I18nStartOp)
			if ok {
				return i18nStartOp.SubTemplateIndex
			}
		}
	}
	return i18nOp.SubTemplateIndex
}

// addParam adds a param value to the given params map.
func addParam(
	params map[string][]ops_create.I18nParamValue,
	placeholder string,
	value interface{}, // string | int | struct{Element int; Template int}
	subTemplateIndex *int,
	flags ir.I18nParamValueFlags,
) {
	values, exists := params[placeholder]
	if !exists {
		values = []ops_create.I18nParamValue{}
	}
	values = append(values, ops_create.I18nParamValue{
		Value:            value,
		SubTemplateIndex: subTemplateIndex,
		Flags:            flags,
	})
	params[placeholder] = values
}
