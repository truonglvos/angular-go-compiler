package phases

import (
	"fmt"
	"strings"

	"ngc-go/packages/compiler/src/ml_parser"
	"ngc-go/packages/compiler/src/output"
	"ngc-go/packages/compiler/src/template/pipeline/ir"
	"ngc-go/packages/compiler/src/template/pipeline/ir/src/operations"
	ops_create "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/create"
	ops_update "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/update"

	pipeline "ngc-go/packages/compiler/src/template/pipeline/src/compilation"
	pipeline_util "ngc-go/packages/compiler/src/template/pipeline/src/util"
)

// SpecializeBindings specializes binding operations into more specific operations types.
func SpecializeBindings(job *pipeline.CompilationJob) {
	elements := make(map[operations.XrefId]operations.CreateOp)
	for _, unit := range job.GetUnits() {
		for op := unit.GetCreate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			createOp, ok := op.(operations.CreateOp)
			if !ok {
				continue
			}
			if ops_create.IsElementOrContainerOp(createOp) {
				elements[createOp.GetXref()] = createOp
			}
		}
	}

	for _, unit := range job.GetUnits() {
		// Process create ops
		for op := unit.GetCreate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			if op.GetKind() == ir.OpKindBinding {
				specializeBindingOp(unit, op, elements, job)
			}
		}
		// Process update ops
		for op := unit.GetUpdate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			if op.GetKind() == ir.OpKindBinding {
				specializeBindingOp(unit, op, elements, job)
			}
		}
	}
}

// specializeBindingOp specializes a single binding operations
func specializeBindingOp(
	unit pipeline.CompilationUnit,
	op operations.Op,
	elements map[operations.XrefId]operations.CreateOp,
	job *pipeline.CompilationJob,
) {
	bindingOp, ok := op.(*ops_update.BindingOp)
	if !ok {
		return
	}

	switch bindingOp.BindingKind {
	case ir.BindingKindAttribute:
		if bindingOp.Name == "ngNonBindable" {
			unit.GetUpdate().Remove(op)
			target := lookupElementBinding(elements, bindingOp.Target)
			// Set NonBindable on the element op
			if elementStartOp, ok := target.(*ops_create.ElementStartOp); ok {
				elementStartOp.NonBindable = true
			} else if elementOp, ok := target.(*ops_create.ElementOp); ok {
				elementOp.NonBindable = true
			} else if containerStartOp, ok := target.(*ops_create.ContainerStartOp); ok {
				containerStartOp.NonBindable = true
			} else if containerOp, ok := target.(*ops_create.ContainerOp); ok {
				containerOp.NonBindable = true
			}
		} else if strings.HasPrefix(bindingOp.Name, "animate.") {
			animBindingOp := ops_update.NewAnimationBindingOp(
				bindingOp.Target,
				bindingOp.Name,
				bindingOp.Expression,
			)
			unit.GetUpdate().Replace(op, animBindingOp)
		} else {
			namespace, name := ml_parser.SplitNsName(bindingOp.Name, false)
			var namespacePtr *string
			if namespace != "" {
				namespacePtr = &namespace
			}
			attrOp := ops_update.NewAttributeOp(
				bindingOp.Target,
				namespacePtr,
				name,
				bindingOp.Expression,
				nil, // sanitizer
				bindingOp.IsTextAttribute,
				bindingOp.IsStructuralTemplateAttribute,
				bindingOp.TemplateKind,
			)
			unit.GetUpdate().Replace(op, attrOp)
		}
	case ir.BindingKindAnimation:
		animBindingOp := ops_update.NewAnimationBindingOp(
			bindingOp.Target,
			bindingOp.Name,
			bindingOp.Expression,
		)
		unit.GetUpdate().Replace(op, animBindingOp)
	case ir.BindingKindProperty, ir.BindingKindLegacyAnimation:
		// Convert a property binding targeting an ARIA attribute (e.g. [aria-label]) into an
		// attribute binding when we know it can't also target an input. Note that a `Host` job is
		// always `DomOnly`, so this condition must be checked first.
		if job.Mode == pipeline.TemplateCompilationModeDomOnly && pipeline_util.IsAriaAttribute(bindingOp.Name) {
			attrOp := ops_update.NewAttributeOp(
				bindingOp.Target,
				nil, // namespace - ARIA attributes don't have namespace
				bindingOp.Name,
				bindingOp.Expression,
				nil, // sanitizer
				bindingOp.IsTextAttribute,
				bindingOp.IsStructuralTemplateAttribute,
				bindingOp.TemplateKind,
			)
			// Copy i18nContext from BindingOp to AttributeOp
			attrOp.I18nContext = bindingOp.I18nContext
			attrOp.I18nMessage = bindingOp.I18nMessage
			attrOp.SourceSpan = bindingOp.SourceSpan
			unit.GetUpdate().Replace(op, attrOp)
		} else if job.Kind == pipeline.CompilationJobKindHost {
			domPropOp := ops_update.NewDomPropertyOp(
				0, // target - not used for host bindings
				bindingOp.Name,
				bindingOp.Expression,
				bindingOp.BindingKind,
				nil, // sanitizer
			)
			unit.GetUpdate().Replace(op, domPropOp)
		} else if bindingOp.Name == "field" {
			controlOp := ops_update.NewControlOp(
				bindingOp.Target,
				bindingOp.Name,
				bindingOp.Expression,
				nil, // sanitizer
			)
			unit.GetUpdate().Replace(op, controlOp)
		} else {
			propOp := ops_update.NewPropertyOp(
				bindingOp.Target,
				bindingOp.Name,
				bindingOp.Expression,
				bindingOp.BindingKind,
				nil, // sanitizer
			)
			// Copy i18nContext from BindingOp to PropertyOp
			propOp.I18nContext = bindingOp.I18nContext
			propOp.I18nMessage = bindingOp.I18nMessage
			propOp.SourceSpan = bindingOp.SourceSpan
			unit.GetUpdate().Replace(op, propOp)
		}
	case ir.BindingKindTwoWayProperty:
		// Check if expression is an output.OutputExpression
		expr, ok := bindingOp.Expression.(output.OutputExpression)
		if !ok {
			panic(fmt.Sprintf("Expected value of two-way property binding \"%s\" to be an expression", bindingOp.Name))
		}
		twoWayOp := ops_update.NewTwoWayPropertyOp(
			bindingOp.Target,
			bindingOp.Name,
			expr,
			nil, // sanitizer
		)
		unit.GetUpdate().Replace(op, twoWayOp)
	case ir.BindingKindI18n, ir.BindingKindClassName, ir.BindingKindStyleProperty:
		panic(fmt.Sprintf("Unhandled binding of kind %v", bindingOp.BindingKind))
	}
}

// lookupElementBinding looks up an element in the given map by xref ID.
func lookupElementBinding(
	elements map[operations.XrefId]operations.CreateOp,
	xref operations.XrefId,
) operations.CreateOp {
	el, exists := elements[xref]
	if !exists {
		panic(fmt.Sprintf("All attributes should have an element-like target: %d", xref))
	}
	return el
}
