package phases

import (
	"fmt"

	"ngc-go/packages/compiler/src/output"
	r3_identifiers "ngc-go/packages/compiler/src/render3/r3_identifiers"
	"ngc-go/packages/compiler/src/template/pipeline/ir"
	"ngc-go/packages/compiler/src/template/pipeline/ir/src/expression"
	ir_operations "ngc-go/packages/compiler/src/template/pipeline/ir/src/operations"
	ops_create "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/create"
	ops_shared "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/shared"
	ops_update "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/update"
	ir_variable "ngc-go/packages/compiler/src/template/pipeline/ir/src/variable"

	pipeline "ngc-go/packages/compiler/src/template/pipeline/src/compilation"
	pipeline_instruction "ngc-go/packages/compiler/src/template/pipeline/src/instruction"
	"ngc-go/packages/compiler/src/util"
)

// Map of target resolvers for event listeners.
var globalTargetResolvers = map[string]*output.ExternalReference{
	"window":   r3_identifiers.ResolveWindow,
	"document": r3_identifiers.ResolveDocument,
	"body":     r3_identifiers.ResolveBody,
}

// DOM properties that need to be remapped on the compiler side.
// Note: this mapping has to be kept in sync with the equally named mapping in the runtime.
var domPropertyRemapping = map[string]string{
	"class":      "className",
	"for":        "htmlFor",
	"formaction": "formAction",
	"innerHtml":  "innerHTML",
	"readonly":   "readOnly",
	"tabindex":   "tabIndex",
}

// Reify compiles semantic operations across all views and generates output statements
// with actual runtime calls in their place.
//
// Reification replaces semantic operations with selected Ivy instructions and other generated code
// structures. After reification, the create/update operation lists of all views should only contain
// `ir.StatementOp`s (which wrap generated `o.Statement`s).
func Reify(job *pipeline.CompilationJob) {
	for _, unit := range job.GetUnits() {
		reifyCreateOperations(unit, unit.GetCreate())
		reifyUpdateOperations(unit, unit.GetUpdate())
	}
}

// This function can be used a sanity check -- it walks every expression in the const pool, and
// every expression reachable from an op, and makes sure that there are no IR expressions
// left. This is nice to use for debugging mysterious failures where an IR expression cannot be
// output from the output AST code.
func ensureNoIrForDebug(job *pipeline.CompilationJob) {
	// TODO: Implement if needed for debugging
	// This would require access to job.pool.statements which may not be available in the same way
}

func reifyCreateOperations(unit pipeline.CompilationUnit, ops *ir_operations.OpList) {
	for op := ops.Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
		expression.TransformExpressionsInOp(op, reifyIrExpression, expression.VisitorContextFlagNone)

		switch op.GetKind() {
		case ir.OpKindText:
			textOp, ok := op.(*ops_create.TextOp)
			if !ok {
				panic("expected TextOp")
			}
			if textOp.Handle == nil || textOp.Handle.Slot == nil {
				panic("expected slot to be assigned")
			}
			ops.Replace(op, pipeline_instruction.Text(*textOp.Handle.Slot, textOp.InitialValue, textOp.SourceSpan))
		case ir.OpKindElementStart:
			elementStartOp, ok := op.(*ops_create.ElementStartOp)
			if !ok {
				panic("expected ElementStartOp")
			}
			if elementStartOp.Handle == nil || elementStartOp.Handle.Slot == nil {
				panic("expected slot to be assigned")
			}
			if elementStartOp.Tag == nil {
				panic("expected tag to be set")
			}
			var attributesPtr *int
			if elementStartOp.Attributes != 0 {
				attrIdx := int(elementStartOp.Attributes)
				attributesPtr = &attrIdx
			}
			var localRefsPtr *int
			if localRefsIdx, ok := elementStartOp.LocalRefs.(ir_operations.ConstIndex); ok && localRefsIdx != 0 {
				idx := int(localRefsIdx)
				localRefsPtr = &idx
			}
			if unit.GetJob().Mode == pipeline.TemplateCompilationModeDomOnly {
				ops.Replace(op, pipeline_instruction.DomElementStart(
					*elementStartOp.Handle.Slot,
					*elementStartOp.Tag,
					attributesPtr,
					localRefsPtr,
					elementStartOp.StartSourceSpan,
				))
			} else {
				ops.Replace(op, pipeline_instruction.ElementStart(
					*elementStartOp.Handle.Slot,
					*elementStartOp.Tag,
					attributesPtr,
					localRefsPtr,
					elementStartOp.StartSourceSpan,
				))
			}
		case ir.OpKindElement:
			elementOp, ok := op.(*ops_create.ElementOp)
			if !ok {
				panic("expected ElementOp")
			}
			if elementOp.Handle == nil || elementOp.Handle.Slot == nil {
				panic("expected slot to be assigned")
			}
			if elementOp.Tag == nil {
				panic("expected tag to be set")
			}
			var attributesPtr *int
			if elementOp.Attributes != 0 {
				attrIdx := int(elementOp.Attributes)
				attributesPtr = &attrIdx
			}
			var localRefsPtr *int
			if localRefsIdx, ok := elementOp.LocalRefs.(ir_operations.ConstIndex); ok && localRefsIdx != 0 {
				idx := int(localRefsIdx)
				localRefsPtr = &idx
			}
			if unit.GetJob().Mode == pipeline.TemplateCompilationModeDomOnly {
				ops.Replace(op, pipeline_instruction.DomElement(
					*elementOp.Handle.Slot,
					*elementOp.Tag,
					attributesPtr,
					localRefsPtr,
					elementOp.WholeSourceSpan,
				))
			} else {
				ops.Replace(op, pipeline_instruction.Element(
					*elementOp.Handle.Slot,
					*elementOp.Tag,
					attributesPtr,
					localRefsPtr,
					elementOp.WholeSourceSpan,
				))
			}
		case ir.OpKindElementEnd:
			elementEndOp, ok := op.(*ops_create.ElementEndOp)
			if !ok {
				panic("expected ElementEndOp")
			}
			if unit.GetJob().Mode == pipeline.TemplateCompilationModeDomOnly {
				ops.Replace(op, pipeline_instruction.DomElementEnd(elementEndOp.SourceSpan))
			} else {
				ops.Replace(op, pipeline_instruction.ElementEnd(elementEndOp.SourceSpan))
			}
		case ir.OpKindContainerStart:
			containerStartOp, ok := op.(*ops_create.ContainerStartOp)
			if !ok {
				panic("expected ContainerStartOp")
			}
			if containerStartOp.Handle == nil || containerStartOp.Handle.Slot == nil {
				panic("expected slot to be assigned")
			}
			var attributesPtr *int
			if containerStartOp.Attributes != 0 {
				attrIdx := int(containerStartOp.Attributes)
				attributesPtr = &attrIdx
			}
			var localRefsPtr *int
			if localRefsIdx, ok := containerStartOp.LocalRefs.(ir_operations.ConstIndex); ok && localRefsIdx != 0 {
				idx := int(localRefsIdx)
				localRefsPtr = &idx
			}
			if unit.GetJob().Mode == pipeline.TemplateCompilationModeDomOnly {
				ops.Replace(op, pipeline_instruction.DomElementContainerStart(
					*containerStartOp.Handle.Slot,
					attributesPtr,
					localRefsPtr,
					containerStartOp.StartSourceSpan,
				))
			} else {
				ops.Replace(op, pipeline_instruction.ElementContainerStart(
					*containerStartOp.Handle.Slot,
					attributesPtr,
					localRefsPtr,
					containerStartOp.StartSourceSpan,
				))
			}
		case ir.OpKindContainer:
			containerOp, ok := op.(*ops_create.ContainerOp)
			if !ok {
				panic("expected ContainerOp")
			}
			if containerOp.Handle == nil || containerOp.Handle.Slot == nil {
				panic("expected slot to be assigned")
			}
			var attributesPtr *int
			if containerOp.Attributes != 0 {
				attrIdx := int(containerOp.Attributes)
				attributesPtr = &attrIdx
			}
			var localRefsPtr *int
			if localRefsIdx, ok := containerOp.LocalRefs.(ir_operations.ConstIndex); ok && localRefsIdx != 0 {
				idx := int(localRefsIdx)
				localRefsPtr = &idx
			}
			if unit.GetJob().Mode == pipeline.TemplateCompilationModeDomOnly {
				ops.Replace(op, pipeline_instruction.DomElementContainer(
					*containerOp.Handle.Slot,
					attributesPtr,
					localRefsPtr,
					containerOp.WholeSourceSpan,
				))
			} else {
				ops.Replace(op, pipeline_instruction.ElementContainer(
					*containerOp.Handle.Slot,
					attributesPtr,
					localRefsPtr,
					containerOp.WholeSourceSpan,
				))
			}
		case ir.OpKindContainerEnd:
			if unit.GetJob().Mode == pipeline.TemplateCompilationModeDomOnly {
				ops.Replace(op, pipeline_instruction.DomElementContainerEnd())
			} else {
				ops.Replace(op, pipeline_instruction.ElementContainerEnd())
			}
		case ir.OpKindI18nStart:
			i18nStartOp, ok := op.(*ops_create.I18nStartOp)
			if !ok {
				panic("expected I18nStartOp")
			}
			if i18nStartOp.Handle == nil || i18nStartOp.Handle.Slot == nil {
				panic("expected slot to be assigned")
			}
			if i18nStartOp.MessageIndex == 0 {
				panic("expected messageIndex to be set")
			}
			ops.Replace(op, pipeline_instruction.I18nStart(
				*i18nStartOp.Handle.Slot,
				int(i18nStartOp.MessageIndex),
				i18nStartOp.SubTemplateIndex,
				i18nStartOp.SourceSpan,
			))
		case ir.OpKindI18nEnd:
			i18nEndOp, ok := op.(*ops_create.I18nEndOp)
			if !ok {
				panic("expected I18nEndOp")
			}
			ops.Replace(op, pipeline_instruction.I18nEnd(i18nEndOp.SourceSpan))
		case ir.OpKindI18n:
			i18nOp, ok := op.(*ops_create.I18nOp)
			if !ok {
				panic("expected I18nOp")
			}
			if i18nOp.Handle == nil || i18nOp.Handle.Slot == nil {
				panic("expected slot to be assigned")
			}
			if i18nOp.MessageIndex == 0 {
				panic("expected messageIndex to be set")
			}
			ops.Replace(op, pipeline_instruction.I18n(
				*i18nOp.Handle.Slot,
				int(i18nOp.MessageIndex),
				i18nOp.SubTemplateIndex,
				i18nOp.SourceSpan,
			))
		case ir.OpKindI18nAttributes:
			i18nAttrOp, ok := op.(*ops_create.I18nAttributesOp)
			if !ok {
				panic("expected I18nAttributesOp")
			}
			if i18nAttrOp.Handle == nil || i18nAttrOp.Handle.Slot == nil {
				panic("expected slot to be assigned")
			}
			if i18nAttrOp.I18nAttributesConfig == 0 {
				panic("AssertionError: i18nAttributesConfig was not set")
			}
			ops.Replace(op, pipeline_instruction.I18nAttributes(
				*i18nAttrOp.Handle.Slot,
				int(i18nAttrOp.I18nAttributesConfig),
			))
		case ir.OpKindTemplate:
			templateOp, ok := op.(*ops_create.TemplateOp)
			if !ok {
				panic("expected TemplateOp")
			}
			viewUnit, ok := unit.(*pipeline.ViewCompilationUnit)
			if !ok {
				panic("AssertionError: must be compiling a component")
			}
			if _, ok := templateOp.LocalRefs.([]ops_create.LocalRef); ok {
				panic("AssertionError: local refs array should have been extracted into a constant")
			}
			childView := viewUnit.Job.Views[templateOp.Xref]
			if childView == nil {
				panic(fmt.Sprintf("expected child view %d to exist", templateOp.Xref))
			}
			if childView.FnName == nil {
				panic("expected child view to have been named")
			}
			if childView.Decls == nil {
				panic("expected child view decls to be set")
			}
			if childView.Vars == nil {
				panic("expected child view vars to be set")
			}
			if templateOp.Handle == nil || templateOp.Handle.Slot == nil {
				panic("expected slot to be assigned")
			}
			var attributesPtr *int
			if templateOp.Attributes != 0 {
				attrIdx := int(templateOp.Attributes)
				attributesPtr = &attrIdx
			}
			var localRefsPtr *int
			if localRefsIdx, ok := templateOp.LocalRefs.(ir_operations.ConstIndex); ok && localRefsIdx != 0 {
				idx := int(localRefsIdx)
				localRefsPtr = &idx
			}
			// Block templates can't have directives so we can always generate them as DOM-only.
			if templateOp.TemplateKind == ir.TemplateKindBlock || unit.GetJob().Mode == pipeline.TemplateCompilationModeDomOnly {
				ops.Replace(op, pipeline_instruction.DomTemplate(
					*templateOp.Handle.Slot,
					output.NewReadVarExpr(*childView.FnName, nil, nil),
					*childView.Decls,
					*childView.Vars,
					templateOp.Tag,
					attributesPtr,
					localRefsPtr,
					templateOp.StartSourceSpan,
				))
			} else {
				ops.Replace(op, pipeline_instruction.Template(
					*templateOp.Handle.Slot,
					output.NewReadVarExpr(*childView.FnName, nil, nil),
					*childView.Decls,
					*childView.Vars,
					templateOp.Tag,
					attributesPtr,
					localRefsPtr,
					templateOp.StartSourceSpan,
				))
			}
		case ir.OpKindDisableBindings:
			ops.Replace(op, pipeline_instruction.DisableBindings())
		case ir.OpKindEnableBindings:
			ops.Replace(op, pipeline_instruction.EnableBindings())
		case ir.OpKindPipe:
			pipeOp, ok := op.(*ops_create.PipeOp)
			if !ok {
				panic("expected PipeOp")
			}
			if pipeOp.Handle == nil || pipeOp.Handle.Slot == nil {
				panic("expected slot to be assigned")
			}
			ops.Replace(op, pipeline_instruction.Pipe(*pipeOp.Handle.Slot, pipeOp.Name))
		case ir.OpKindDeclareLet:
			declareLetOp, ok := op.(*ops_create.DeclareLetOp)
			if !ok {
				panic("expected DeclareLetOp")
			}
			if declareLetOp.Handle == nil || declareLetOp.Handle.Slot == nil {
				panic("expected slot to be assigned")
			}
			ops.Replace(op, pipeline_instruction.DeclareLet(*declareLetOp.Handle.Slot, declareLetOp.SourceSpan))
		case ir.OpKindAnimationString:
			animStringOp, ok := op.(*ops_create.AnimationStringOp)
			if !ok {
				panic("expected AnimationStringOp")
			}
			ops.Replace(op, pipeline_instruction.AnimationString(
				animStringOp.AnimationKind,
				animStringOp.Expression,
				animStringOp.Sanitizer,
				animStringOp.SourceSpan,
			))
		case ir.OpKindAnimation:
			animOp, ok := op.(*ops_create.AnimationOp)
			if !ok {
				panic("expected AnimationOp")
			}
			if animOp.HandlerFnName == nil {
				panic("expected handlerFnName to be set")
			}
			animationCallbackFn := reifyListenerHandler(
				unit,
				*animOp.HandlerFnName,
				animOp.HandlerOps,
				false, // consumesDollarEvent
			)
			ops.Replace(op, pipeline_instruction.Animation(
				animOp.AnimationKind,
				animationCallbackFn,
				animOp.Sanitizer,
				animOp.SourceSpan,
			))
		case ir.OpKindAnimationListener:
			animListenerOp, ok := op.(*ops_create.AnimationListenerOp)
			if !ok {
				panic("expected AnimationListenerOp")
			}
			if animListenerOp.HandlerFnName == nil {
				panic("expected handlerFnName to be set")
			}
			animationListenerFn := reifyListenerHandler(
				unit,
				*animListenerOp.HandlerFnName,
				animListenerOp.HandlerOps,
				animListenerOp.ConsumesDollarEvent,
			)
			ops.Replace(op, pipeline_instruction.AnimationListener(
				animListenerOp.AnimationKind,
				animationListenerFn,
				nil, // eventTargetResolver
				animListenerOp.SourceSpan,
			))
		case ir.OpKindListener:
			listenerOp, ok := op.(*ops_create.ListenerOp)
			if !ok {
				panic("expected ListenerOp")
			}
			if listenerOp.HandlerFnName == nil {
				panic("expected handlerFnName to be set")
			}
			listenerFn := reifyListenerHandler(
				unit,
				*listenerOp.HandlerFnName,
				listenerOp.HandlerOps,
				listenerOp.ConsumesDollarEvent,
			)
			var eventTargetResolver *output.ExternalReference
			if listenerOp.EventTarget != nil {
				resolver, exists := globalTargetResolvers[*listenerOp.EventTarget]
				if !exists {
					panic(fmt.Sprintf(
						"Unexpected global target '%s' defined for '%s' event. Supported list of global targets: window,document,body.",
						*listenerOp.EventTarget,
						listenerOp.Name,
					))
				}
				eventTargetResolver = resolver
			}
			if unit.GetJob().Mode == pipeline.TemplateCompilationModeDomOnly &&
				!listenerOp.HostListener &&
				!listenerOp.IsLegacyAnimationListener {
				ops.Replace(op, pipeline_instruction.DomListener(
					listenerOp.Name,
					listenerFn,
					eventTargetResolver,
					listenerOp.SourceSpan,
				))
			} else {
				ops.Replace(op, pipeline_instruction.Listener(
					listenerOp.Name,
					listenerFn,
					eventTargetResolver,
					listenerOp.HostListener && listenerOp.IsLegacyAnimationListener,
					listenerOp.SourceSpan,
				))
			}
		case ir.OpKindTwoWayListener:
			twoWayListenerOp, ok := op.(*ops_create.TwoWayListenerOp)
			if !ok {
				panic("expected TwoWayListenerOp")
			}
			if twoWayListenerOp.HandlerFnName == nil {
				panic("expected handlerFnName to be set")
			}
			ops.Replace(op, pipeline_instruction.TwoWayListener(
				twoWayListenerOp.Name,
				reifyListenerHandler(unit, *twoWayListenerOp.HandlerFnName, twoWayListenerOp.HandlerOps, true),
				twoWayListenerOp.SourceSpan,
			))
		case ir.OpKindVariable:
			variableOp, ok := op.(*ops_shared.VariableOp)
			if !ok {
				panic("expected VariableOp")
			}
			varName := variableOp.Variable.(ir_variable.SemanticVariable).GetName()
			if varName == nil {
				panic(fmt.Sprintf("AssertionError: unnamed variable %d", variableOp.Xref))
			}
			ops.Replace(op, ops_shared.NewStatementOp(
				output.NewDeclareVarStmt(
					*varName,
					variableOp.Initializer,
					nil, // type
					output.StmtModifierFinal,
					nil, // sourceSpan
					nil, // leadingComments
				),
			))
		case ir.OpKindNamespace:
			namespaceOp, ok := op.(*ops_create.NamespaceOp)
			if !ok {
				panic("expected NamespaceOp")
			}
			switch namespaceOp.Active {
			case ir.NamespaceHTML:
				ops.Replace(op, pipeline_instruction.NamespaceHTML())
			case ir.NamespaceSVG:
				ops.Replace(op, pipeline_instruction.NamespaceSVG())
			case ir.NamespaceMath:
				ops.Replace(op, pipeline_instruction.NamespaceMath())
			}
		case ir.OpKindDefer:
			deferOp, ok := op.(*ops_create.DeferOp)
			if !ok {
				panic("expected DeferOp")
			}
			if deferOp.Handle == nil || deferOp.Handle.Slot == nil {
				panic("expected slot to be assigned")
			}
			if deferOp.MainSlot == nil || deferOp.MainSlot.Slot == nil {
				panic("expected mainSlot to be assigned")
			}
			var loadingSlotPtr *int
			if deferOp.LoadingSlot != nil && deferOp.LoadingSlot.Slot != nil {
				loadingSlotPtr = deferOp.LoadingSlot.Slot
			}
			var placeholderSlotPtr *int
			if deferOp.PlaceholderSlot != nil && deferOp.PlaceholderSlot.Slot != nil {
				placeholderSlotPtr = deferOp.PlaceholderSlot.Slot
			}
			var errorSlotPtr *int
			if deferOp.ErrorSlot != nil && deferOp.ErrorSlot.Slot != nil {
				errorSlotPtr = deferOp.ErrorSlot.Slot
			}
			timerScheduling := deferOp.LoadingMinimumTime != nil || deferOp.LoadingAfterTime != nil || deferOp.PlaceholderMinimumTime != nil
			ops.Replace(op, pipeline_instruction.Defer(
				*deferOp.Handle.Slot,
				*deferOp.MainSlot.Slot,
				deferOp.ResolverFn,
				loadingSlotPtr,
				placeholderSlotPtr,
				errorSlotPtr,
				deferOp.LoadingConfig,
				deferOp.PlaceholderConfig,
				timerScheduling,
				deferOp.SourceSpan,
				&deferOp.Flags,
			))
		case ir.OpKindDeferOn:
			deferOnOp, ok := op.(*ops_create.DeferOnOp)
			if !ok {
				panic("expected DeferOnOp")
			}
			var args []output.OutputExpression
			switch deferOnOp.Trigger.GetKind() {
			case ir.DeferTriggerKindNever, ir.DeferTriggerKindIdle, ir.DeferTriggerKindImmediate:
				// No args
			case ir.DeferTriggerKindTimer:
				timerTrigger, ok := deferOnOp.Trigger.(*ops_create.DeferTimerTrigger)
				if !ok {
					panic("expected DeferTimerTrigger")
				}
				args = []output.OutputExpression{output.NewLiteralExpr(timerTrigger.Delay, nil, nil)}
			case ir.DeferTriggerKindViewport:
				viewportTrigger, ok := deferOnOp.Trigger.(*ops_create.DeferViewportTrigger)
				if !ok {
					panic("expected DeferViewportTrigger")
				}
				// `hydrate` triggers don't support targets.
				if deferOnOp.Modifier == ir.DeferOpModifierKindHydrate {
					if viewportTrigger.Options != nil {
						args = []output.OutputExpression{viewportTrigger.Options}
					}
				} else {
					// The slots not being defined at this point is invalid, however we
					// catch it during type checking. Pass in null in such cases.
					var targetSlot *int
					if viewportTrigger.TargetSlot != nil && viewportTrigger.TargetSlot.Slot != nil {
						targetSlot = viewportTrigger.TargetSlot.Slot
					}
					args = []output.OutputExpression{output.NewLiteralExpr(targetSlot, nil, nil)}
					if viewportTrigger.TargetSlotViewSteps != nil && *viewportTrigger.TargetSlotViewSteps != 0 {
						args = append(args, output.NewLiteralExpr(*viewportTrigger.TargetSlotViewSteps, nil, nil))
					} else if viewportTrigger.Options != nil {
						args = append(args, output.NewLiteralExpr(nil, nil, nil))
					}
					if viewportTrigger.Options != nil {
						args = append(args, viewportTrigger.Options)
					}
				}
			case ir.DeferTriggerKindInteraction, ir.DeferTriggerKindHover:
				interactionTrigger, ok := deferOnOp.Trigger.(*ops_create.DeferInteractionTrigger)
				if !ok {
					hoverTrigger, ok := deferOnOp.Trigger.(*ops_create.DeferHoverTrigger)
					if !ok {
						panic("expected DeferInteractionTrigger or DeferHoverTrigger")
					}
					// `hydrate` triggers don't support targets.
					if deferOnOp.Modifier == ir.DeferOpModifierKindHydrate {
						args = []output.OutputExpression{}
					} else {
						var targetSlot *int
						if hoverTrigger.TargetSlot != nil && hoverTrigger.TargetSlot.Slot != nil {
							targetSlot = hoverTrigger.TargetSlot.Slot
						}
						args = []output.OutputExpression{output.NewLiteralExpr(targetSlot, nil, nil)}
						if hoverTrigger.TargetSlotViewSteps != nil && *hoverTrigger.TargetSlotViewSteps != 0 {
							args = append(args, output.NewLiteralExpr(*hoverTrigger.TargetSlotViewSteps, nil, nil))
						}
					}
				} else {
					// `hydrate` triggers don't support targets.
					if deferOnOp.Modifier == ir.DeferOpModifierKindHydrate {
						args = []output.OutputExpression{}
					} else {
						var targetSlot *int
						if interactionTrigger.TargetSlot != nil && interactionTrigger.TargetSlot.Slot != nil {
							targetSlot = interactionTrigger.TargetSlot.Slot
						}
						args = []output.OutputExpression{output.NewLiteralExpr(targetSlot, nil, nil)}
						if interactionTrigger.TargetSlotViewSteps != nil && *interactionTrigger.TargetSlotViewSteps != 0 {
							args = append(args, output.NewLiteralExpr(*interactionTrigger.TargetSlotViewSteps, nil, nil))
						}
					}
				}
			default:
				panic(fmt.Sprintf(
					"AssertionError: Unsupported reification of defer trigger kind %v",
					deferOnOp.Trigger.GetKind(),
				))
			}
			ops.Replace(op, pipeline_instruction.DeferOn(
				deferOnOp.Trigger.GetKind(),
				args,
				deferOnOp.Modifier,
				deferOnOp.SourceSpan,
			))
		case ir.OpKindProjectionDef:
			projectionDefOp, ok := op.(*ops_create.ProjectionDefOp)
			if !ok {
				panic("expected ProjectionDefOp")
			}
			ops.Replace(op, pipeline_instruction.ProjectionDef(projectionDefOp.Def))
		case ir.OpKindProjection:
			projectionOp, ok := op.(*ops_create.ProjectionOp)
			if !ok {
				panic("expected ProjectionOp")
			}
			if projectionOp.Handle == nil || projectionOp.Handle.Slot == nil {
				panic("No slot was assigned for project instruction")
			}
			var fallbackViewFnName *string
			var fallbackDecls *int
			var fallbackVars *int
			if projectionOp.FallbackView != 0 {
				viewUnit, ok := unit.(*pipeline.ViewCompilationUnit)
				if !ok {
					panic("AssertionError: must be compiling a component")
				}
				fallbackView := viewUnit.Job.Views[projectionOp.FallbackView]
				if fallbackView == nil {
					panic("AssertionError: projection had fallback view xref, but fallback view was not found")
				}
				if fallbackView.FnName == nil || fallbackView.Decls == nil || fallbackView.Vars == nil {
					panic("AssertionError: expected projection fallback view to have been named and counted")
				}
				fallbackViewFnName = fallbackView.FnName
				fallbackDecls = fallbackView.Decls
				fallbackVars = fallbackView.Vars
			}
			var attributesPtr *output.LiteralArrayExpr
			if projectionOp.Attributes != nil {
				attributesPtr = projectionOp.Attributes
			}
			ops.Replace(op, pipeline_instruction.Projection(
				*projectionOp.Handle.Slot,
				projectionOp.ProjectionSlotIndex,
				attributesPtr,
				fallbackViewFnName,
				fallbackDecls,
				fallbackVars,
				projectionOp.SourceSpan,
			))
		case ir.OpKindConditionalCreate:
			conditionalCreateOp, ok := op.(*ops_create.ConditionalCreateOp)
			if !ok {
				panic("expected ConditionalCreateOp")
			}
			viewUnit, ok := unit.(*pipeline.ViewCompilationUnit)
			if !ok {
				panic("AssertionError: must be compiling a component")
			}
			if _, ok := conditionalCreateOp.LocalRefs.([]ops_create.LocalRef); ok {
				panic("AssertionError: local refs array should have been extracted into a constant")
			}
			conditionalCreateChildView := viewUnit.Job.Views[conditionalCreateOp.Xref]
			if conditionalCreateChildView == nil {
				panic(fmt.Sprintf("expected child view %d to exist", conditionalCreateOp.Xref))
			}
			if conditionalCreateChildView.FnName == nil {
				panic("expected conditional create child view to have been named")
			}
			if conditionalCreateChildView.Decls == nil {
				panic("expected conditional create child view decls to be set")
			}
			if conditionalCreateChildView.Vars == nil {
				panic("expected conditional create child view vars to be set")
			}
			if conditionalCreateOp.Handle == nil || conditionalCreateOp.Handle.Slot == nil {
				panic("expected slot to be assigned")
			}
			var attributesPtr *int
			if conditionalCreateOp.Attributes != 0 {
				attrIdx := int(conditionalCreateOp.Attributes)
				attributesPtr = &attrIdx
			}
			var localRefsPtr *int
			if localRefsIdx, ok := conditionalCreateOp.LocalRefs.(ir_operations.ConstIndex); ok && localRefsIdx != 0 {
				idx := int(localRefsIdx)
				localRefsPtr = &idx
			}
			ops.Replace(op, pipeline_instruction.ConditionalCreate(
				*conditionalCreateOp.Handle.Slot,
				output.NewReadVarExpr(*conditionalCreateChildView.FnName, nil, nil),
				*conditionalCreateChildView.Decls,
				*conditionalCreateChildView.Vars,
				conditionalCreateOp.Tag,
				attributesPtr,
				localRefsPtr,
				conditionalCreateOp.StartSourceSpan,
			))
		case ir.OpKindConditionalBranchCreate:
			conditionalBranchCreateOp, ok := op.(*ops_create.ConditionalBranchCreateOp)
			if !ok {
				panic("expected ConditionalBranchCreateOp")
			}
			viewUnit, ok := unit.(*pipeline.ViewCompilationUnit)
			if !ok {
				panic("AssertionError: must be compiling a component")
			}
			if _, ok := conditionalBranchCreateOp.LocalRefs.([]ops_create.LocalRef); ok {
				panic("AssertionError: local refs array should have been extracted into a constant")
			}
			conditionalBranchCreateChildView := viewUnit.Job.Views[conditionalBranchCreateOp.Xref]
			if conditionalBranchCreateChildView == nil {
				panic(fmt.Sprintf("expected child view %d to exist", conditionalBranchCreateOp.Xref))
			}
			if conditionalBranchCreateChildView.FnName == nil {
				panic("expected conditional branch create child view to have been named")
			}
			if conditionalBranchCreateChildView.Decls == nil {
				panic("expected conditional branch create child view decls to be set")
			}
			if conditionalBranchCreateChildView.Vars == nil {
				panic("expected conditional branch create child view vars to be set")
			}
			if conditionalBranchCreateOp.Handle == nil || conditionalBranchCreateOp.Handle.Slot == nil {
				panic("expected slot to be assigned")
			}
			var attributesPtr *int
			if conditionalBranchCreateOp.Attributes != 0 {
				attrIdx := int(conditionalBranchCreateOp.Attributes)
				attributesPtr = &attrIdx
			}
			var localRefsPtr *int
			if localRefsIdx, ok := conditionalBranchCreateOp.LocalRefs.(ir_operations.ConstIndex); ok && localRefsIdx != 0 {
				idx := int(localRefsIdx)
				localRefsPtr = &idx
			}
			ops.Replace(op, pipeline_instruction.ConditionalBranchCreate(
				*conditionalBranchCreateOp.Handle.Slot,
				output.NewReadVarExpr(*conditionalBranchCreateChildView.FnName, nil, nil),
				*conditionalBranchCreateChildView.Decls,
				*conditionalBranchCreateChildView.Vars,
				conditionalBranchCreateOp.Tag,
				attributesPtr,
				localRefsPtr,
				conditionalBranchCreateOp.StartSourceSpan,
			))
		case ir.OpKindRepeaterCreate:
			repeaterCreateOp, ok := op.(*ops_create.RepeaterCreateOp)
			if !ok {
				panic("expected RepeaterCreateOp")
			}
			if repeaterCreateOp.Handle == nil || repeaterCreateOp.Handle.Slot == nil {
				panic("No slot was assigned for repeater instruction")
			}
			viewUnit, ok := unit.(*pipeline.ViewCompilationUnit)
			if !ok {
				panic("AssertionError: must be compiling a component")
			}
			repeaterView := viewUnit.Job.Views[repeaterCreateOp.Xref]
			if repeaterView == nil {
				panic(fmt.Sprintf("expected repeater view %d to exist", repeaterCreateOp.Xref))
			}
			if repeaterView.FnName == nil {
				panic("AssertionError: expected repeater primary view to have been named")
			}
			var emptyViewFnName *string
			var emptyDecls *int
			var emptyVars *int
			var emptyTag *string
			var emptyAttributesPtr *int
			if repeaterCreateOp.EmptyView != 0 {
				emptyView := viewUnit.Job.Views[repeaterCreateOp.EmptyView]
				if emptyView == nil {
					panic("AssertionError: repeater had empty view xref, but empty view was not found")
				}
				if emptyView.FnName == nil || emptyView.Decls == nil || emptyView.Vars == nil {
					panic("AssertionError: expected repeater empty view to have been named and counted")
				}
				emptyViewFnName = emptyView.FnName
				emptyDecls = emptyView.Decls
				emptyVars = emptyView.Vars
				emptyTag = repeaterCreateOp.EmptyTag
				if repeaterCreateOp.EmptyAttributes != 0 {
					attrIdx := int(repeaterCreateOp.EmptyAttributes)
					emptyAttributesPtr = &attrIdx
				}
			}
			if repeaterCreateOp.Decls == nil {
				panic("expected repeater decls to be set")
			}
			if repeaterCreateOp.Vars == nil {
				panic("expected repeater vars to be set")
			}
			var attributesPtr *int
			if repeaterCreateOp.Attributes != 0 {
				attrIdx := int(repeaterCreateOp.Attributes)
				attributesPtr = &attrIdx
			}
			ops.Replace(op, pipeline_instruction.RepeaterCreate(
				*repeaterCreateOp.Handle.Slot,
				*repeaterView.FnName,
				*repeaterCreateOp.Decls,
				*repeaterCreateOp.Vars,
				repeaterCreateOp.Tag,
				attributesPtr,
				reifyTrackBy(unit, repeaterCreateOp),
				repeaterCreateOp.UsesComponentInstance,
				emptyViewFnName,
				emptyDecls,
				emptyVars,
				emptyTag,
				emptyAttributesPtr,
				repeaterCreateOp.WholeSourceSpan,
			))
		case ir.OpKindSourceLocation:
			sourceLocationOp, ok := op.(*ops_create.SourceLocationOp)
			if !ok {
				panic("expected SourceLocationOp")
			}
			locations := make([]output.OutputExpression, len(sourceLocationOp.Locations))
			for i, loc := range sourceLocationOp.Locations {
				if loc.TargetSlot == nil || loc.TargetSlot.Slot == nil {
					panic("No slot was assigned for source location")
				}
				locations[i] = output.NewLiteralArrayExpr(
					[]output.OutputExpression{
						output.NewLiteralExpr(*loc.TargetSlot.Slot, nil, nil),
						output.NewLiteralExpr(loc.Offset, nil, nil),
						output.NewLiteralExpr(loc.Line, nil, nil),
						output.NewLiteralExpr(loc.Column, nil, nil),
					},
					nil,
					nil,
				)
			}
			locationsLiteral := output.NewLiteralArrayExpr(locations, nil, nil)
			ops.Replace(op, pipeline_instruction.AttachSourceLocation(sourceLocationOp.TemplatePath, locationsLiteral))
		case ir.OpKindControlCreate:
			controlCreateOp, ok := op.(*ops_create.ControlCreateOp)
			if !ok {
				panic("expected ControlCreateOp")
			}
			ops.Replace(op, pipeline_instruction.ControlCreate(controlCreateOp.SourceSpan))
		case ir.OpKindStatement:
			// Pass statement operations directly through.
		default:
			panic(fmt.Sprintf("AssertionError: Unsupported reification of create op %v", op.GetKind()))
		}
	}
}

// reifyUpdateOperations reifies update operations
func reifyUpdateOperations(unit pipeline.CompilationUnit, ops *ir_operations.OpList) {
	for op := ops.Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
		expression.TransformExpressionsInOp(op, reifyIrExpression, expression.VisitorContextFlagNone)

		switch op.GetKind() {
		case ir.OpKindAdvance:
			advanceOp, ok := op.(*ops_update.AdvanceOp)
			if !ok {
				panic("expected AdvanceOp")
			}
			ops.Replace(op, pipeline_instruction.Advance(advanceOp.AdvanceBy, nil))
		case ir.OpKindProperty:
			propertyOp, ok := op.(*ops_update.PropertyOp)
			if !ok {
				panic("expected PropertyOp")
			}
			if unit.GetJob().Mode == pipeline.TemplateCompilationModeDomOnly &&
				propertyOp.BindingKind != ir.BindingKindLegacyAnimation &&
				propertyOp.BindingKind != ir.BindingKindAnimation {
				ops.Replace(op, reifyDomProperty(propertyOp))
			} else {
				ops.Replace(op, reifyProperty(propertyOp))
			}
		case ir.OpKindControl:
			controlOp, ok := op.(*ops_update.ControlOp)
			if !ok {
				panic("expected ControlOp")
			}
			ops.Replace(op, reifyControl(controlOp))
		case ir.OpKindTwoWayProperty:
			twoWayPropertyOp, ok := op.(*ops_update.TwoWayPropertyOp)
			if !ok {
				panic("expected TwoWayPropertyOp")
			}
			ops.Replace(op, pipeline_instruction.TwoWayProperty(
				twoWayPropertyOp.Name,
				twoWayPropertyOp.Expression,
				twoWayPropertyOp.Sanitizer,
				nil,
			))
		case ir.OpKindStyleProp:
			stylePropOp, ok := op.(*ops_update.StylePropOp)
			if !ok {
				panic("expected StylePropOp")
			}
			ops.Replace(op, pipeline_instruction.StyleProp(
				stylePropOp.Name,
				stylePropOp.Expression,
				stylePropOp.Unit,
				nil,
			))
		case ir.OpKindClassProp:
			classPropOp, ok := op.(*ops_update.ClassPropOp)
			if !ok {
				panic("expected ClassPropOp")
			}
			expr := convertExpressionToOutputExpression(classPropOp.Expression, nil)
			ops.Replace(op, pipeline_instruction.ClassProp(
				classPropOp.Name,
				expr,
				nil,
			))
		case ir.OpKindStyleMap:
			styleMapOp, ok := op.(*ops_update.StyleMapOp)
			if !ok {
				panic("expected StyleMapOp")
			}
			ops.Replace(op, pipeline_instruction.StyleMap(styleMapOp.Expression, nil))
		case ir.OpKindClassMap:
			classMapOp, ok := op.(*ops_update.ClassMapOp)
			if !ok {
				panic("expected ClassMapOp")
			}
			ops.Replace(op, pipeline_instruction.ClassMap(classMapOp.Expression, nil))
		case ir.OpKindI18nExpression:
			i18nExpOp, ok := op.(*ops_update.I18nExpressionOp)
			if !ok {
				panic("expected I18nExpressionOp")
			}
			ops.Replace(op, pipeline_instruction.I18nExp(i18nExpOp.Expression, i18nExpOp.SourceSpan))
		case ir.OpKindI18nApply:
			i18nApplyOp, ok := op.(*ops_update.I18nApplyOp)
			if !ok {
				panic("expected I18nApplyOp")
			}
			if i18nApplyOp.Handle == nil || i18nApplyOp.Handle.Slot == nil {
				panic("expected slot to be assigned")
			}
			ops.Replace(op, pipeline_instruction.I18nApply(*i18nApplyOp.Handle.Slot, i18nApplyOp.SourceSpan))
		case ir.OpKindInterpolateText:
			interpolateTextOp, ok := op.(*ops_update.InterpolateTextOp)
			if !ok {
				panic("expected InterpolateTextOp")
			}
			ops.Replace(op, pipeline_instruction.TextInterpolate(
				interpolateTextOp.Interpolation.Strings,
				interpolateTextOp.Interpolation.Expressions,
				interpolateTextOp.SourceSpan,
			))
		case ir.OpKindAttribute:
			attributeOp, ok := op.(*ops_update.AttributeOp)
			if !ok {
				panic("expected AttributeOp")
			}
			ops.Replace(op, pipeline_instruction.Attribute(
				attributeOp.Name,
				attributeOp.Expression,
				attributeOp.Sanitizer,
				attributeOp.Namespace,
				attributeOp.SourceSpan,
			))
		case ir.OpKindDomProperty:
			domPropertyOp, ok := op.(*ops_update.DomPropertyOp)
			if !ok {
				panic("expected DomPropertyOp")
			}
			if _, ok := domPropertyOp.Expression.(*ops_update.Interpolation); ok {
				panic("not yet handled")
			} else {
				if domPropertyOp.BindingKind == ir.BindingKindLegacyAnimation ||
					domPropertyOp.BindingKind == ir.BindingKindAnimation {
					var expr output.OutputExpression
					if e, ok := domPropertyOp.Expression.(output.OutputExpression); ok {
						expr = e
					} else {
						panic(fmt.Sprintf("invalid expression type: %T", domPropertyOp.Expression))
					}
					ops.Replace(op, pipeline_instruction.SyntheticHostProperty(
						domPropertyOp.Name,
						expr,
						nil,
					))
				} else {
					ops.Replace(op, reifyDomProperty(domPropertyOp))
				}
			}
		case ir.OpKindVariable:
			variableOp, ok := op.(*ops_shared.VariableOp)
			if !ok {
				panic("expected VariableOp")
			}
			varName := variableOp.Variable.(ir_variable.SemanticVariable).GetName()
			if varName == nil {
				panic(fmt.Sprintf("AssertionError: unnamed variable %d", variableOp.Xref))
			}
			ops.Replace(op, ops_shared.NewStatementOp(
				output.NewDeclareVarStmt(
					*varName,
					variableOp.Initializer,
					nil, // type
					output.StmtModifierFinal,
					nil, // sourceSpan
					nil, // leadingComments
				),
			))
		case ir.OpKindConditional:
			conditionalOp, ok := op.(*ops_update.ConditionalOp)
			if !ok {
				panic("expected ConditionalOp")
			}
			if conditionalOp.Processed == nil {
				panic("Conditional test was not set.")
			}
			ops.Replace(op, pipeline_instruction.Conditional(
				conditionalOp.Processed,
				conditionalOp.ContextValue,
				nil,
			))
		case ir.OpKindRepeater:
			repeaterOp, ok := op.(*ops_update.RepeaterOp)
			if !ok {
				panic("expected RepeaterOp")
			}
			ops.Replace(op, pipeline_instruction.Repeater(repeaterOp.Collection, nil))
		case ir.OpKindDeferWhen:
			deferWhenOp, ok := op.(*ops_update.DeferWhenOp)
			if !ok {
				panic("expected DeferWhenOp")
			}
			ops.Replace(op, pipeline_instruction.DeferWhen(
				deferWhenOp.Modifier,
				deferWhenOp.Expr,
				deferWhenOp.SourceSpan,
			))
		case ir.OpKindStoreLet:
			storeLetOp, ok := op.(*ops_update.StoreLetOp)
			if !ok {
				panic("expected StoreLetOp")
			}
			panic(fmt.Sprintf("AssertionError: unexpected storeLet %s", storeLetOp.DeclaredName))
		case ir.OpKindStatement:
			// Pass statement operations directly through.
		default:
			panic(fmt.Sprintf("AssertionError: Unsupported reification of update op %v", op.GetKind()))
		}
	}
}

// convertExpressionToOutputExpression converts an interface{} expression to output.OutputExpression
func convertExpressionToOutputExpression(expr interface{}, sourceSpan *util.ParseSourceSpan) output.OutputExpression {
	if interp, ok := expr.(*ops_update.Interpolation); ok {
		// Convert interpolation to expression using the same logic as instruction.go
		interpolationArgs := []output.OutputExpression{}
		for i, str := range interp.Strings {
			interpolationArgs = append(interpolationArgs, output.NewLiteralExpr(str, nil, nil))
			if i < len(interp.Expressions) {
				interpolationArgs = append(interpolationArgs, interp.Expressions[i])
			}
		}
		// Use ValueInterpolateConfig logic (for property bindings)
		n := len(interp.Expressions)
		config := pipeline_instruction.ValueInterpolateConfig
		if n < len(config.Constant) {
			allArgs := interpolationArgs
			mappedN := config.Mapping(len(interpolationArgs))
			if mappedN < len(config.Constant) {
				return output.NewInvokeFunctionExpr(
					output.NewExternalExpr(&config.Constant[mappedN], nil, nil, nil),
					allArgs,
					nil,
					sourceSpan,
					false,
				)
			}
		}
		if config.Variable != nil {
			return output.NewInvokeFunctionExpr(
				output.NewExternalExpr(config.Variable, nil, nil, nil),
				[]output.OutputExpression{output.NewLiteralArrayExpr(interpolationArgs, nil, nil)},
				nil,
				sourceSpan,
				false,
			)
		}
		panic("unable to call variadic function")
	} else if e, ok := expr.(output.OutputExpression); ok {
		return e
	} else {
		panic(fmt.Sprintf("invalid expression type: %T", expr))
	}
}

// reifyDomProperty reifies a DOM property binding operation.
// This is an optimized version of reifyProperty that avoids unnecessarily trying to bind
// to directive inputs at runtime for views that don't import any directives.
func reifyDomProperty(op interface{}) ir_operations.UpdateOp {
	var name string
	var expression interface{} // output.OutputExpression | *Interpolation
	var sanitizer output.OutputExpression

	switch opType := op.(type) {
	case *ops_update.DomPropertyOp:
		name = opType.Name
		expression = opType.Expression
		sanitizer = opType.Sanitizer
	case *ops_update.PropertyOp:
		name = opType.Name
		expression = opType.Expression
		sanitizer = opType.Sanitizer
	default:
		panic(fmt.Sprintf("unexpected op type: %T", op))
	}

	remappedName, exists := domPropertyRemapping[name]
	if !exists {
		remappedName = name
	}
	return pipeline_instruction.DomProperty(remappedName, expression, sanitizer, nil)
}

// reifyProperty reifies a property binding operation.
// The returned statement attempts to bind to directive inputs before falling back to a DOM property.
func reifyProperty(op *ops_update.PropertyOp) ir_operations.UpdateOp {
	if isAriaAttribute(op.Name) {
		return pipeline_instruction.AriaProperty(op.Name, op.Expression, nil)
	}
	return pipeline_instruction.Property(op.Name, op.Expression, op.Sanitizer, op.SourceSpan)
}

// isAriaAttribute checks if a property name is an ARIA attribute
func isAriaAttribute(name string) bool {
	return len(name) > 5 && name[:5] == "aria-"
}

// reifyControl reifies a control operation
func reifyControl(op *ops_update.ControlOp) ir_operations.UpdateOp {
	return pipeline_instruction.Control(op.Expression, op.Sanitizer, nil)
}

// reifyIrExpression reifies IR expressions
func reifyIrExpression(expr output.OutputExpression, flags expression.VisitorContextFlag) output.OutputExpression {
	if !expression.IsIrExpression(expr) {
		return expr
	}

	switch irExpr := expr.(type) {
	case *expression.NextContextExpr:
		return pipeline_instruction.NextContext(irExpr.Steps)
	case *expression.ReferenceExpr:
		if irExpr.TargetSlot == nil || irExpr.TargetSlot.Slot == nil {
			panic("expected targetSlot to be assigned")
		}
		return pipeline_instruction.Reference(*irExpr.TargetSlot.Slot + 1 + irExpr.Offset)
	case *expression.LexicalReadExpr:
		panic(fmt.Sprintf("AssertionError: unresolved LexicalRead of %s", irExpr.Name))
	case *expression.TwoWayBindingSetExpr:
		panic("AssertionError: unresolved TwoWayBindingSet")
	case *expression.RestoreViewExpr:
		if _, ok := irExpr.View.(int); ok {
			panic("AssertionError: unresolved RestoreView")
		}
		if viewExpr, ok := irExpr.View.(output.OutputExpression); ok {
			return pipeline_instruction.RestoreView(viewExpr)
		}
		panic(fmt.Sprintf("invalid view type: %T", irExpr.View))
	case *expression.ResetViewExpr:
		return pipeline_instruction.ResetView(irExpr.Expr)
	case *expression.GetCurrentViewExpr:
		return pipeline_instruction.GetCurrentView()
	case *expression.ReadVariableExpr:
		if irExpr.Name == nil {
			panic(fmt.Sprintf("Read of unnamed variable %d", irExpr.Xref))
		}
		return output.NewReadVarExpr(*irExpr.Name, nil, nil)
	case *expression.ReadTemporaryExpr:
		if irExpr.Name == nil {
			panic(fmt.Sprintf("Read of unnamed temporary %d", irExpr.Xref))
		}
		return output.NewReadVarExpr(*irExpr.Name, nil, nil)
	case *expression.AssignTemporaryExpr:
		if irExpr.Name == nil {
			panic(fmt.Sprintf("Assign of unnamed temporary %d", irExpr.Xref))
		}
		return output.NewReadVarExpr(*irExpr.Name, nil, nil).Set(irExpr.Expr)
	case *expression.PureFunctionExpr:
		if irExpr.Fn == nil {
			panic("AssertionError: expected PureFunctions to have been extracted")
		}
		if irExpr.VarOffset == nil {
			panic("expected varOffset to be set")
		}
		return pipeline_instruction.PureFunction(*irExpr.VarOffset, irExpr.Fn, irExpr.Args)
	case *expression.PureFunctionParameterExpr:
		panic("AssertionError: expected PureFunctionParameterExpr to have been extracted")
	case *expression.PipeBindingExpr:
		if irExpr.TargetSlot == nil || irExpr.TargetSlot.Slot == nil {
			panic("expected targetSlot to be assigned")
		}
		if irExpr.VarOffset == nil {
			panic("expected varOffset to be set")
		}
		return pipeline_instruction.PipeBind(*irExpr.TargetSlot.Slot, *irExpr.VarOffset, irExpr.Args)
	case *expression.PipeBindingVariadicExpr:
		if irExpr.TargetSlot == nil || irExpr.TargetSlot.Slot == nil {
			panic("expected targetSlot to be assigned")
		}
		if irExpr.VarOffset == nil {
			panic("expected varOffset to be set")
		}
		return pipeline_instruction.PipeBindV(*irExpr.TargetSlot.Slot, *irExpr.VarOffset, irExpr.Args)
	case *expression.SlotLiteralExpr:
		if irExpr.Slot == nil || irExpr.Slot.Slot == nil {
			panic("expected slot to be assigned")
		}
		return output.NewLiteralExpr(*irExpr.Slot.Slot, nil, nil)
	case *expression.ContextLetReferenceExpr:
		if irExpr.TargetSlot == nil || irExpr.TargetSlot.Slot == nil {
			panic("expected targetSlot to be assigned")
		}
		return pipeline_instruction.ReadContextLet(*irExpr.TargetSlot.Slot)
	case *expression.StoreLetExpr:
		return pipeline_instruction.StoreLet(irExpr.Value, irExpr.SourceSpan)
	case *expression.TrackContextExpr:
		return output.NewReadVarExpr("this", nil, nil)
	default:
		panic(fmt.Sprintf("AssertionError: Unsupported reification of ir.Expression kind: %T", expr))
	}
}

// reifyListenerHandler turns listeners into a function expression, which may or may not have the `$event`
// parameter defined.
func reifyListenerHandler(
	unit pipeline.CompilationUnit,
	name string,
	handlerOps *ir_operations.OpList,
	consumesDollarEvent bool,
) output.OutputExpression {
	// First, reify all instruction calls within handlerOps.
	reifyUpdateOperations(unit, handlerOps)

	// Next, extract all the OutputStatement from the reified operations. We can expect that at this
	// point, all operations have been converted to statements.
	handlerStmts := []output.OutputStatement{}
	for op := handlerOps.Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
		if op.GetKind() != ir.OpKindStatement {
			panic(fmt.Sprintf("AssertionError: expected reified statements, but found op %v", op.GetKind()))
		}
		statementOp, ok := op.(*ops_shared.StatementOp)
		if !ok {
			panic("expected StatementOp")
		}
		handlerStmts = append(handlerStmts, statementOp.Statement)
	}

	// If `$event` is referenced, we need to generate it as a parameter.
	params := []*output.FnParam{}
	if consumesDollarEvent {
		// We need the `$event` parameter.
		params = append(params, output.NewFnParam("$event", output.DynamicType))
	}

	return output.NewFunctionExpr(params, handlerStmts, nil, nil, &name)
}

// reifyTrackBy reifies the tracking expression of a RepeaterCreateOp.
func reifyTrackBy(unit pipeline.CompilationUnit, op *ops_create.RepeaterCreateOp) output.OutputExpression {
	// If the tracking function was created already, there's nothing left to do.
	if op.TrackByFn != nil {
		return op.TrackByFn
	}

	params := []*output.FnParam{
		output.NewFnParam("$index", output.DynamicType),
		output.NewFnParam("$item", output.DynamicType),
	}
	var fn output.OutputExpression

	if op.TrackByOps == nil {
		// If there are no additional ops related to the tracking function, we just need
		// to turn it into a function that returns the result of the expression.
		if op.UsesComponentInstance {
			fn = output.NewFunctionExpr(
				params,
				[]output.OutputStatement{
					output.NewReturnStatement(op.Track, nil, nil),
				},
				nil,
				nil,
				nil,
			)
		} else {
			fn = output.NewArrowFunctionExpr(params, op.Track, output.DynamicType, nil)
		}
	} else {
		// Otherwise first we need to reify the track-related ops.
		reifyUpdateOperations(unit, op.TrackByOps)

		statements := []output.OutputStatement{}
		for trackOp := op.TrackByOps.Head(); trackOp != nil && trackOp.GetKind() != ir.OpKindListEnd; trackOp = trackOp.Next() {
			if trackOp.GetKind() != ir.OpKindStatement {
				panic(fmt.Sprintf("AssertionError: expected reified statements, but found op %v", trackOp.GetKind()))
			}
			statementOp, ok := trackOp.(*ops_shared.StatementOp)
			if !ok {
				panic("expected StatementOp")
			}
			statements = append(statements, statementOp.Statement)
		}

		// Afterwards we can create the function from those ops.
		if op.UsesComponentInstance || len(statements) != 1 {
			fn = output.NewFunctionExpr(params, statements, nil, nil, nil)
		} else {
			// Check if the first statement is a return statement
			if returnStmt, ok := statements[0].(*output.ReturnStatement); ok {
				fn = output.NewArrowFunctionExpr(params, returnStmt.Value, output.DynamicType, nil)
			} else {
				fn = output.NewFunctionExpr(params, statements, nil, nil, nil)
			}
		}
	}

	// Get shared function reference from pool
	viewUnit, ok := unit.(*pipeline.ViewCompilationUnit)
	if !ok {
		panic("expected ViewCompilationUnit")
	}
	op.TrackByFn = viewUnit.Job.Pool.GetSharedFunctionReference(fn, "_forTrack", true)
	return op.TrackByFn
}
