package phases

import (
	"ngc-go/packages/compiler/src/core"
	"ngc-go/packages/compiler/src/output"
	"ngc-go/packages/compiler/src/template/pipeline/ir"
	"ngc-go/packages/compiler/src/template/pipeline/ir/src/expression"
	"ngc-go/packages/compiler/src/template/pipeline/ir/src/operations"
	ops_create "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/create"

	"ngc-go/packages/compiler/src/template/pipeline/src/compilation"
	pipeline_convension "ngc-go/packages/compiler/src/template/pipeline/src/convension"
)

// CollectElementConsts converts the semantic attributes of element-like operations (elements, templates) into constant
// array expressions, and lifts them into the overall component `consts`.
func CollectElementConsts(job *compilation.CompilationJob) {
	// Collect all extracted attributes.
	allElementAttributes := make(map[operations.XrefId]*ElementAttributes)
	for _, unit := range job.GetUnits() {
		for op := unit.GetCreate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			if op.GetKind() == ir.OpKindExtractedAttribute {
				extractedAttrOp, ok := op.(*ops_create.ExtractedAttributeOp)
				if !ok {
					continue
				}
				attributes, exists := allElementAttributes[extractedAttrOp.Target]
				if !exists {
					attributes = NewElementAttributes(job.Compatibility)
					allElementAttributes[extractedAttrOp.Target] = attributes
				}
				var expr output.OutputExpression
				if extractedAttrOp.Expression != nil {
					if e, ok := extractedAttrOp.Expression.(output.OutputExpression); ok {
						expr = e
					}
				}
				attributes.Add(
					extractedAttrOp.BindingKind,
					extractedAttrOp.Name,
					expr,
					extractedAttrOp.Namespace,
					extractedAttrOp.TrustedValueFn,
				)
				unit.GetCreate().Remove(op)
			}
		}
	}

	// Serialize the extracted attributes into the const array.
	if job.Kind == compilation.CompilationJobKindTmpl {
		// Cast to ComponentCompilationJob by accessing through a ViewCompilationUnit
		var componentJob *compilation.ComponentCompilationJob
		if len(job.GetUnits()) > 0 {
			if viewUnit, ok := job.GetUnits()[0].(*compilation.ViewCompilationUnit); ok {
				componentJob = viewUnit.Job
			}
		}
		if componentJob == nil {
			return
		}
		for _, unit := range componentJob.GetUnits() {
			for op := unit.GetCreate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
				// TODO: Simplify and combine these cases.
				if op.GetKind() == ir.OpKindProjection {
					projectionOp, ok := op.(*ops_create.ProjectionOp)
					if !ok {
						continue
					}
					attributes, exists := allElementAttributes[projectionOp.Xref]
					if exists {
						attrArray := serializeAttributes(attributes)
						if len(attrArray.Entries) > 0 {
							projectionOp.Attributes = attrArray
						}
					}
				} else {
					// Type assert to CreateOp to check IsElementOrContainerOp
					createOp, ok := op.(operations.CreateOp)
					if !ok {
						continue
					}
					if !ops_create.IsElementOrContainerOp(createOp) {
						continue
					}
					constIndex := getConstIndex(componentJob, allElementAttributes, createOp.GetXref())
					// Set Attributes on the op
					if elementStartOp, ok := op.(*ops_create.ElementStartOp); ok {
						elementStartOp.Attributes = constIndex
					} else if elementOp, ok := op.(*ops_create.ElementOp); ok {
						elementOp.Attributes = constIndex
					} else if containerStartOp, ok := op.(*ops_create.ContainerStartOp); ok {
						containerStartOp.Attributes = constIndex
					} else if containerOp, ok := op.(*ops_create.ContainerOp); ok {
						containerOp.Attributes = constIndex
					} else if templateOp, ok := op.(*ops_create.TemplateOp); ok {
						templateOp.Attributes = constIndex
					}

					// TODO(dylhunn): `@for` loops with `@empty` blocks need to be special-cased here,
					// because the slot consumer trait currently only supports one slot per consumer and we
					// need two. This should be revisited when making the refactors mentioned in:
					// https://github.com/angular/angular/pull/53620#discussion_r1430918822
					if repeaterOp, ok := op.(*ops_create.RepeaterCreateOp); ok && repeaterOp.EmptyView != 0 {
						emptyConstIndex := getConstIndex(componentJob, allElementAttributes, repeaterOp.EmptyView)
						repeaterOp.EmptyAttributes = emptyConstIndex
					}
				}
			}
		}
	} else if job.Kind == compilation.CompilationJobKindHost {
		// Get the root unit and cast to HostBindingCompilationUnit
		rootUnit := job.GetRoot()
		hostUnit, ok := rootUnit.(*compilation.HostBindingCompilationUnit)
		if !ok {
			return
		}
		// TODO: If the host binding case further diverges, we may want to split it into its own
		// phase.
		for xref, attributes := range allElementAttributes {
			if xref != hostUnit.Xref {
				panic("An attribute would be const collected into the host binding's template function, but is not associated with the root xref.")
			}
			attrArray := serializeAttributes(attributes)
			if len(attrArray.Entries) > 0 {
				hostUnit.Attributes = attrArray
			}
		}
	}
}

func getConstIndex(
	job *compilation.ComponentCompilationJob,
	allElementAttributes map[operations.XrefId]*ElementAttributes,
	xref operations.XrefId,
) operations.ConstIndex {
	attributes, exists := allElementAttributes[xref]
	if exists {
		attrArray := serializeAttributes(attributes)
		if len(attrArray.Entries) > 0 {
			return job.AddConst(attrArray, nil)
		}
	}
	return 0
}

// ElementAttributes is a container for all of the various kinds of attributes which are applied on an element.
type ElementAttributes struct {
	known            map[ir.BindingKind]map[string]bool
	byKind           map[ir.BindingKind][]output.OutputExpression
	propertyBindings []output.OutputExpression
	projectAs        *string
	compatibility    ir.CompatibilityMode
}

// NewElementAttributes creates a new ElementAttributes
func NewElementAttributes(compatibility ir.CompatibilityMode) *ElementAttributes {
	return &ElementAttributes{
		known:            make(map[ir.BindingKind]map[string]bool),
		byKind:           make(map[ir.BindingKind][]output.OutputExpression),
		propertyBindings: nil,
		projectAs:        nil,
		compatibility:    compatibility,
	}
}

// GetAttributes returns the attributes array
func (e *ElementAttributes) GetAttributes() []output.OutputExpression {
	return e.byKind[ir.BindingKindAttribute]
}

// GetClasses returns the classes array
func (e *ElementAttributes) GetClasses() []output.OutputExpression {
	return e.byKind[ir.BindingKindClassName]
}

// GetStyles returns the styles array
func (e *ElementAttributes) GetStyles() []output.OutputExpression {
	return e.byKind[ir.BindingKindStyleProperty]
}

// GetBindings returns the bindings array
func (e *ElementAttributes) GetBindings() []output.OutputExpression {
	if e.propertyBindings == nil {
		return []output.OutputExpression{}
	}
	return e.propertyBindings
}

// GetTemplate returns the template array
func (e *ElementAttributes) GetTemplate() []output.OutputExpression {
	return e.byKind[ir.BindingKindTemplate]
}

// GetI18n returns the i18n array
func (e *ElementAttributes) GetI18n() []output.OutputExpression {
	return e.byKind[ir.BindingKindI18n]
}

// IsKnown checks if a binding kind and name combination is already known
func (e *ElementAttributes) IsKnown(kind ir.BindingKind, name string) bool {
	nameToValue, exists := e.known[kind]
	if !exists {
		nameToValue = make(map[string]bool)
		e.known[kind] = nameToValue
	}
	if nameToValue[name] {
		return true
	}
	nameToValue[name] = true
	return false
}

// Add adds an attribute to the ElementAttributes
func (e *ElementAttributes) Add(
	kind ir.BindingKind,
	name string,
	value output.OutputExpression,
	namespace *string,
	trustedValueFn output.OutputExpression,
) {
	// TemplateDefinitionBuilder puts duplicate attribute, class, and style values into the consts
	// array. This seems inefficient, we can probably keep just the first one or the last value
	// (whichever actually gets applied when multiple values are listed for the same attribute).
	allowDuplicates := e.compatibility == ir.CompatibilityModeTemplateDefinitionBuilder &&
		(kind == ir.BindingKindAttribute ||
			kind == ir.BindingKindClassName ||
			kind == ir.BindingKindStyleProperty)
	if !allowDuplicates && e.IsKnown(kind, name) {
		return
	}

	// TODO: Can this be its own phase
	if name == "ngProjectAs" {
		if value == nil {
			panic("ngProjectAs must have a string literal value")
		}
		literalExpr, ok := value.(*output.LiteralExpr)
		if !ok || literalExpr.Value == nil {
			panic("ngProjectAs must have a string literal value")
		}
		strValue, ok := literalExpr.Value.(string)
		if !ok {
			panic("ngProjectAs must have a string literal value")
		}
		e.projectAs = &strValue
		// TODO: TemplateDefinitionBuilder allows `ngProjectAs` to also be assigned as a literal
		// attribute. Is this sane?
	}

	array := e.arrayFor(kind)
	nameLiterals := getAttributeNameLiterals(namespace, name)
	for _, lit := range nameLiterals {
		array = append(array, lit)
	}
	if kind == ir.BindingKindAttribute || kind == ir.BindingKindStyleProperty {
		if value == nil {
			panic("Attribute, i18n attribute, & style element attributes must have a value")
		}
		if trustedValueFn != nil {
			if !expression.IsStringLiteral(value) {
				panic("AssertionError: extracted attribute value should be string literal")
			}
			literalExpr, ok := value.(*output.LiteralExpr)
			if !ok {
				panic("AssertionError: extracted attribute value should be string literal")
			}
			strValue, ok := literalExpr.Value.(string)
			if !ok {
				panic("AssertionError: extracted attribute value should be string literal")
			}
			templateElement := output.NewTemplateLiteralElementExpr(strValue, nil, "")
			templateLiteral := output.NewTemplateLiteralExpr([]*output.TemplateLiteralElementExpr{templateElement}, []output.OutputExpression{}, nil)
			taggedTemplate := output.NewTaggedTemplateLiteralExpr(trustedValueFn, templateLiteral, nil, nil)
			array = append(array, taggedTemplate)
		} else {
			array = append(array, value)
		}
	}
	// Update the map
	if kind == ir.BindingKindProperty || kind == ir.BindingKindTwoWayProperty {
		e.propertyBindings = array
	} else {
		e.byKind[kind] = array
	}
}

func (e *ElementAttributes) arrayFor(kind ir.BindingKind) []output.OutputExpression {
	if kind == ir.BindingKindProperty || kind == ir.BindingKindTwoWayProperty {
		if e.propertyBindings == nil {
			e.propertyBindings = []output.OutputExpression{}
		}
		return e.propertyBindings
	}
	if _, exists := e.byKind[kind]; !exists {
		e.byKind[kind] = []output.OutputExpression{}
	}
	return e.byKind[kind]
}

// getAttributeNameLiterals gets an array of literal expressions representing the attribute's namespaced name.
func getAttributeNameLiterals(namespace *string, name string) []*output.LiteralExpr {
	nameLiteral := output.NewLiteralExpr(name, nil, nil)

	if namespace != nil && *namespace != "" {
		return []*output.LiteralExpr{
			output.NewLiteralExpr(int(core.AttributeMarkerNamespaceURI), nil, nil),
			output.NewLiteralExpr(*namespace, nil, nil),
			nameLiteral,
		}
	}

	return []*output.LiteralExpr{nameLiteral}
}

// serializeAttributes serializes an ElementAttributes object into an array expression.
func serializeAttributes(attrs *ElementAttributes) *output.LiteralArrayExpr {
	attrArray := attrs.GetAttributes()

	if attrs.projectAs != nil {
		// Parse the attribute value into a CssSelectorList. Note that we only take the
		// first selector, because we don't support multiple selectors in ngProjectAs.
		selectorStr := attrs.projectAs
		parsedR3Selector := core.ParseSelectorToR3Selector(selectorStr)
		if len(parsedR3Selector) > 0 {
			attrArray = append(attrArray,
				output.NewLiteralExpr(int(core.AttributeMarkerProjectAs), nil, nil),
				pipeline_convension.LiteralOrArrayLiteral(parsedR3Selector[0]),
			)
		}
	}
	if len(attrs.GetClasses()) > 0 {
		attrArray = append(attrArray,
			output.NewLiteralExpr(int(core.AttributeMarkerClasses), nil, nil))
		attrArray = append(attrArray, attrs.GetClasses()...)
	}
	if len(attrs.GetStyles()) > 0 {
		attrArray = append(attrArray,
			output.NewLiteralExpr(int(core.AttributeMarkerStyles), nil, nil))
		attrArray = append(attrArray, attrs.GetStyles()...)
	}
	if len(attrs.GetBindings()) > 0 {
		attrArray = append(attrArray,
			output.NewLiteralExpr(int(core.AttributeMarkerBindings), nil, nil))
		attrArray = append(attrArray, attrs.GetBindings()...)
	}
	if len(attrs.GetTemplate()) > 0 {
		attrArray = append(attrArray,
			output.NewLiteralExpr(int(core.AttributeMarkerTemplate), nil, nil))
		attrArray = append(attrArray, attrs.GetTemplate()...)
	}
	if len(attrs.GetI18n()) > 0 {
		attrArray = append(attrArray,
			output.NewLiteralExpr(int(core.AttributeMarkerI18n), nil, nil))
		attrArray = append(attrArray, attrs.GetI18n()...)
	}
	return output.NewLiteralArrayExpr(attrArray, nil, nil)
}
