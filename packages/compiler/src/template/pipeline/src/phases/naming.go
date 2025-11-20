package phases

import (
	"fmt"
	"regexp"
	"strings"

	"ngc-go/packages/compiler/src/output"
	"ngc-go/packages/compiler/src/template/pipeline/ir"
	"ngc-go/packages/compiler/src/template/pipeline/ir/src/expression"
	"ngc-go/packages/compiler/src/template/pipeline/ir/src/operations"
	ops_create "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/create"
	"ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/shared"
	ops_update "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/update"
	ir_variable "ngc-go/packages/compiler/src/template/pipeline/ir/src/variable"

	"ngc-go/packages/compiler/src/template/pipeline/src/compilation"
)

// sanitizeIdentifier sanitizes an identifier to be a valid JavaScript identifier
func sanitizeIdentifier(name string) string {
	// Remove or replace invalid characters
	re := regexp.MustCompile(`[^a-zA-Z0-9_$]`)
	sanitized := re.ReplaceAllString(name, "_")

	// Ensure it doesn't start with a number
	if len(sanitized) > 0 && sanitized[0] >= '0' && sanitized[0] <= '9' {
		sanitized = "_" + sanitized
	}

	return sanitized
}

// NameFunctionsAndVariables generates names for functions and variables across all views.
// This includes propagating those names into any `ir.ReadVariableExpr`s of those variables, so that
// the reads can be emitted correctly.
func NameFunctionsAndVariables(job *compilation.CompilationJob) {
	// Check if this is a template compilation job
	if job.Kind != compilation.CompilationJobKindTmpl {
		return
	}

	// Get ComponentCompilationJob by accessing through root unit
	rootUnit := job.GetRoot()
	viewUnit, ok := rootUnit.(*compilation.ViewCompilationUnit)
	if !ok || viewUnit.Job == nil {
		return
	}
	componentJob := viewUnit.Job

	state := &namingState{index: 0}
	compatibility := job.Compatibility == ir.CompatibilityModeTemplateDefinitionBuilder
	addNamesToView(componentJob.Root, componentJob.ComponentName, state, compatibility)
}

type namingState struct {
	index int
}

func addNamesToView(
	unit compilation.CompilationUnit,
	baseName string,
	state *namingState,
	compatibility bool,
) {
	if unit.GetFnName() == nil {
		// Ensure unique names for view units. This is necessary because there might be multiple
		// components with same names in the context of the same pool. Only add the suffix
		// if really needed.
		job := unit.GetJob()
		fnSuffix := job.GetFnSuffix()
		name := sanitizeIdentifier(fmt.Sprintf("%s_%s", baseName, fnSuffix))
		uniqueName := job.Pool.UniqueName(name, false)
		unit.SetFnName(uniqueName)
	}

	// Keep track of the names we assign to variables in the view. We'll need to propagate these
	// into reads of those variables afterwards.
	varNames := make(map[operations.XrefId]string)

	// Process create ops
	for op := unit.GetCreate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
		processOpForNaming(op, unit, baseName, varNames, state, compatibility)
	}

	// Process update ops
	for op := unit.GetUpdate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
		processOpForNaming(op, unit, baseName, varNames, state, compatibility)
	}

	// Having named all variables declared in the view, now we can push those names into the
	// `ir.ReadVariableExpr` expressions which represent reads of those variables.
	for op := unit.GetCreate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
		expression.VisitExpressionsInOp(op, func(expr output.OutputExpression, flags expression.VisitorContextFlag) {
			if readVar, ok := expr.(*expression.ReadVariableExpr); ok {
				if readVar.Name == nil {
					if name, exists := varNames[readVar.Xref]; exists {
						readVar.Name = &name
					} else {
						panic(fmt.Sprintf("Variable %d not yet named", readVar.Xref))
					}
				}
			}
		})
	}

	for op := unit.GetUpdate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
		expression.VisitExpressionsInOp(op, func(expr output.OutputExpression, flags expression.VisitorContextFlag) {
			if readVar, ok := expr.(*expression.ReadVariableExpr); ok {
				if readVar.Name == nil {
					if name, exists := varNames[readVar.Xref]; exists {
						readVar.Name = &name
					} else {
						panic(fmt.Sprintf("Variable %d not yet named", readVar.Xref))
					}
				}
			}
		})
	}
}

func processOpForNaming(
	op operations.Op,
	unit compilation.CompilationUnit,
	baseName string,
	varNames map[operations.XrefId]string,
	state *namingState,
	compatibility bool,
) {
	fnName := unit.GetFnName()
	if fnName == nil {
		return
	}

	switch op.GetKind() {
	case ir.OpKindProperty, ir.OpKindDomProperty:
		if propertyOp, ok := op.(*ops_update.PropertyOp); ok {
			if propertyOp.BindingKind == ir.BindingKindLegacyAnimation {
				propertyOp.Name = "@" + propertyOp.Name
			}
		} else if domPropertyOp, ok := op.(*ops_update.DomPropertyOp); ok {
			if domPropertyOp.BindingKind == ir.BindingKindLegacyAnimation {
				domPropertyOp.Name = "@" + domPropertyOp.Name
			}
		}
	case ir.OpKindAnimation:
		if animOp, ok := op.(*ops_create.AnimationOp); ok {
			if animOp.HandlerFnName == nil {
				animationKind := strings.Replace(animOp.Name, ".", "", -1)
				name := fmt.Sprintf("%s_%s_cb", *fnName, animationKind)
				sanitized := sanitizeIdentifier(name)
				animOp.HandlerFnName = &sanitized
			}
		}
	case ir.OpKindAnimationListener:
		if animListenerOp, ok := op.(*ops_create.AnimationListenerOp); ok {
			if animListenerOp.HandlerFnName == nil {
				if !animListenerOp.HostListener && (animListenerOp.TargetSlot == nil || animListenerOp.TargetSlot.Slot == nil) {
					panic("Expected a slot to be assigned")
				}
				animationKind := strings.Replace(animListenerOp.Name, ".", "", -1)
				var name string
				if animListenerOp.HostListener {
					name = fmt.Sprintf("%s_%s_HostBindingHandler", baseName, animationKind)
				} else {
					tag := ""
					if animListenerOp.Tag != nil {
						tag = strings.Replace(*animListenerOp.Tag, "-", "_", -1)
					}
					slot := 0
					if animListenerOp.TargetSlot != nil && animListenerOp.TargetSlot.Slot != nil {
						slot = *animListenerOp.TargetSlot.Slot
					}
					name = fmt.Sprintf("%s_%s_%s_%d_listener", *fnName, tag, animationKind, slot)
				}
				sanitized := sanitizeIdentifier(name)
				animListenerOp.HandlerFnName = &sanitized
			}
		}
	case ir.OpKindListener:
		if listenerOp, ok := op.(*ops_create.ListenerOp); ok {
			if listenerOp.HandlerFnName == nil {
				if !listenerOp.HostListener && (listenerOp.TargetSlot == nil || listenerOp.TargetSlot.Slot == nil) {
					panic("Expected a slot to be assigned")
				}
				animation := ""
				if listenerOp.IsLegacyAnimationListener {
					listenerOp.Name = fmt.Sprintf("@%s.%s", listenerOp.Name, listenerOp.LegacyAnimationPhase)
					animation = "animation"
				}
				var name string
				if listenerOp.HostListener {
					name = fmt.Sprintf("%s_%s%s_HostBindingHandler", baseName, animation, listenerOp.Name)
				} else {
					tag := ""
					if listenerOp.Tag != nil {
						tag = strings.Replace(*listenerOp.Tag, "-", "_", -1)
					}
					slot := 0
					if listenerOp.TargetSlot != nil && listenerOp.TargetSlot.Slot != nil {
						slot = *listenerOp.TargetSlot.Slot
					}
					name = fmt.Sprintf("%s_%s_%s%s_%d_listener", *fnName, tag, animation, listenerOp.Name, slot)
				}
				sanitized := sanitizeIdentifier(name)
				listenerOp.HandlerFnName = &sanitized
			}
		}
	case ir.OpKindTwoWayListener:
		if twoWayOp, ok := op.(*ops_create.TwoWayListenerOp); ok {
			if twoWayOp.HandlerFnName == nil {
				if twoWayOp.TargetSlot == nil || twoWayOp.TargetSlot.Slot == nil {
					panic("Expected a slot to be assigned")
				}
				tag := ""
				if twoWayOp.Tag != nil {
					tag = strings.Replace(*twoWayOp.Tag, "-", "_", -1)
				}
				slot := 0
				if twoWayOp.TargetSlot != nil && twoWayOp.TargetSlot.Slot != nil {
					slot = *twoWayOp.TargetSlot.Slot
				}
				name := fmt.Sprintf("%s_%s_%s_%d_listener", *fnName, tag, twoWayOp.Name, slot)
				sanitized := sanitizeIdentifier(name)
				twoWayOp.HandlerFnName = &sanitized
			}
		}
	case ir.OpKindVariable:
		if varOp, ok := op.(*shared.VariableOp); ok {
			varName := getVariableName(unit, varOp.Variable, state, compatibility)
			varNames[varOp.Xref] = varName
		}
	case ir.OpKindRepeaterCreate:
		if repeaterOp, ok := op.(*ops_create.RepeaterCreateOp); ok {
			viewUnit, ok := unit.(*compilation.ViewCompilationUnit)
			if !ok {
				panic("AssertionError: must be compiling a component")
			}
			if repeaterOp.Handle == nil || repeaterOp.Handle.Slot == nil {
				panic("Expected slot to be assigned")
			}
			slot := *repeaterOp.Handle.Slot
			if repeaterOp.EmptyView != 0 {
				emptyView := viewUnit.Job.Views[repeaterOp.EmptyView]
				if emptyView != nil {
					// Repeater empty view function is at slot +2 (metadata is in the first slot).
					addNamesToView(emptyView, fmt.Sprintf("%s_%sEmpty_%d", baseName, repeaterOp.FunctionNameSuffix, slot+2), state, compatibility)
				}
			}
			// Repeater primary view function is at slot +1 (metadata is in the first slot).
			primaryView := viewUnit.Job.Views[repeaterOp.Xref]
			if primaryView != nil {
				addNamesToView(primaryView, fmt.Sprintf("%s_%s_%d", baseName, repeaterOp.FunctionNameSuffix, slot+1), state, compatibility)
			}
		}
	case ir.OpKindProjection:
		if projectionOp, ok := op.(*ops_create.ProjectionOp); ok {
			viewUnit, ok := unit.(*compilation.ViewCompilationUnit)
			if !ok {
				panic("AssertionError: must be compiling a component")
			}
			if projectionOp.Handle == nil || projectionOp.Handle.Slot == nil {
				panic("Expected slot to be assigned")
			}
			slot := *projectionOp.Handle.Slot
			if projectionOp.FallbackView != 0 {
				fallbackView := viewUnit.Job.Views[projectionOp.FallbackView]
				if fallbackView != nil {
					addNamesToView(fallbackView, fmt.Sprintf("%s_ProjectionFallback_%d", baseName, slot), state, compatibility)
				}
			}
		}
	case ir.OpKindConditionalCreate, ir.OpKindConditionalBranchCreate, ir.OpKindTemplate:
		if viewUnit, ok := unit.(*compilation.ViewCompilationUnit); ok {
			var childView *compilation.ViewCompilationUnit
			var handle *ir.SlotHandle
			var functionNameSuffix string

			if conditionalOp, ok := op.(*ops_create.ConditionalCreateOp); ok {
				childView = viewUnit.Job.Views[conditionalOp.Xref]
				handle = conditionalOp.Handle
				functionNameSuffix = conditionalOp.FunctionNameSuffix
			} else if branchOp, ok := op.(*ops_create.ConditionalBranchCreateOp); ok {
				childView = viewUnit.Job.Views[branchOp.Xref]
				handle = branchOp.Handle
				functionNameSuffix = branchOp.FunctionNameSuffix
			} else if templateOp, ok := op.(*ops_create.TemplateOp); ok {
				childView = viewUnit.Job.Views[templateOp.Xref]
				handle = templateOp.Handle
				functionNameSuffix = templateOp.FunctionNameSuffix
			}

			if childView != nil {
				if handle == nil || handle.Slot == nil {
					panic("Expected slot to be assigned")
				}
				slot := *handle.Slot
				suffix := ""
				if functionNameSuffix != "" {
					suffix = "_" + functionNameSuffix
				}
				addNamesToView(childView, fmt.Sprintf("%s%s_%d", baseName, suffix, slot), state, compatibility)
			}
		}
	case ir.OpKindStyleProp:
		if stylePropOp, ok := op.(*ops_update.StylePropOp); ok {
			stylePropOp.Name = normalizeStylePropName(stylePropOp.Name)
			if compatibility {
				stylePropOp.Name = stripImportant(stylePropOp.Name)
			}
		}
	case ir.OpKindClassProp:
		if classPropOp, ok := op.(*ops_update.ClassPropOp); ok {
			if compatibility {
				classPropOp.Name = stripImportant(classPropOp.Name)
			}
		}
	}
}

func getVariableName(
	unit compilation.CompilationUnit,
	variable interface{},
	state *namingState,
	compatibility bool,
) string {
	// Type assertion to get the variable kind
	var name *string
	var kind ir.SemanticVariableKind

	if contextVar, ok := variable.(*ir_variable.ContextVariable); ok {
		kind = ir.SemanticVariableKindContext
		name = contextVar.GetName()
	} else if identifierVar, ok := variable.(*ir_variable.IdentifierVariable); ok {
		kind = ir.SemanticVariableKindIdentifier
		name = identifierVar.GetName()
	} else if savedViewVar, ok := variable.(*ir_variable.SavedViewVariable); ok {
		kind = ir.SemanticVariableKindSavedView
		name = savedViewVar.GetName()
	} else if aliasVar, ok := variable.(*ir_variable.AliasVariable); ok {
		kind = ir.SemanticVariableKindAlias
		name = aliasVar.GetName()
	}

	if name == nil {
		switch kind {
		case ir.SemanticVariableKindContext:
			varName := fmt.Sprintf("ctx_r%d", state.index)
			state.index++
			return varName
		case ir.SemanticVariableKindIdentifier:
			identifierVar := variable.(*ir_variable.IdentifierVariable)
			if compatibility {
				// TODO: Prefix increment and `_r` are for compatibility with the old naming scheme.
				// This has the potential to cause collisions when `ctx` is the identifier, so we need a
				// special check for that as well.
				compatPrefix := ""
				if identifierVar.Identifier == "ctx" {
					compatPrefix = "i"
				}
				state.index++
				return fmt.Sprintf("%s_%sr%d", identifierVar.Identifier, compatPrefix, state.index)
			} else {
				varName := fmt.Sprintf("%s_i%d", identifierVar.Identifier, state.index)
				state.index++
				return varName
			}
		default:
			// TODO: Prefix increment for compatibility only.
			state.index++
			return fmt.Sprintf("_r%d", state.index)
		}
	}
	return *name
}

// normalizeStylePropName normalizes a style prop name by hyphenating it (unless its a CSS variable).
func normalizeStylePropName(name string) string {
	if strings.HasPrefix(name, "--") {
		return name
	}
	return hyphenate(name)
}

// stripImportant strips `!important` out of the given style or class name.
func stripImportant(name string) string {
	importantIndex := strings.Index(name, "!important")
	if importantIndex > -1 {
		return name[:importantIndex]
	}
	return name
}
