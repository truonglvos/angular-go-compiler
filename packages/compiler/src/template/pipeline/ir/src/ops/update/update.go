package ops_update

import (
	"fmt"

	"ngc-go/packages/compiler/src/i18n"
	"ngc-go/packages/compiler/src/output"
	"ngc-go/packages/compiler/src/template/pipeline/ir"
	ir_operations "ngc-go/packages/compiler/src/template/pipeline/ir/src/operations"
	ir_traits "ngc-go/packages/compiler/src/template/pipeline/ir/src/traits"
	"ngc-go/packages/compiler/src/util"
)

// Interpolation represents a logical operation to perform string interpolation on a text node
// Interpolation inputs are stored as static `string`s and dynamic `o.Expression`s, in separate
// arrays. Thus, the interpolation `A{{b}}C{{d}}E` is stored as 3 static strings `['A', 'C', 'E']`
// and 2 dynamic expressions `[b, d]`.
type Interpolation struct {
	Strings          []string
	Expressions      []output.OutputExpression
	I18nPlaceholders []string
}

// NewInterpolation creates a new Interpolation
func NewInterpolation(
	strings []string,
	expressions []output.OutputExpression,
	i18nPlaceholders []string,
) (*Interpolation, error) {
	if len(i18nPlaceholders) != 0 && len(i18nPlaceholders) != len(expressions) {
		return nil, fmt.Errorf(
			"expected %d placeholders to match interpolation expression count, but got %d",
			len(expressions),
			len(i18nPlaceholders),
		)
	}
	return &Interpolation{
		Strings:          strings,
		Expressions:      expressions,
		I18nPlaceholders: i18nPlaceholders,
	}, nil
}

// InterpolateTextOp is a logical operation to perform string interpolation on a text node
type InterpolateTextOp struct {
	ir_operations.OpBase
	Target        ir_operations.XrefId
	Interpolation *Interpolation
	SourceSpan    *util.ParseSourceSpan
}

// NewInterpolateTextOp creates a new InterpolateTextOp
func NewInterpolateTextOp(
	xref ir_operations.XrefId,
	interpolation *Interpolation,
	sourceSpan *util.ParseSourceSpan,
) *InterpolateTextOp {
	return &InterpolateTextOp{
		OpBase:        ir_operations.NewOpBase(),
		Target:        xref,
		Interpolation: interpolation,
		SourceSpan:    sourceSpan,
	}
}

// GetKind returns the operation kind
func (i *InterpolateTextOp) GetKind() ir.OpKind {
	return ir.OpKindInterpolateText
}

// GetXref returns the xref ID
func (i *InterpolateTextOp) GetXref() ir_operations.XrefId {
	return i.Target
}

// SetXref sets the xref ID
func (i *InterpolateTextOp) SetXref(xref ir_operations.XrefId) {
	i.Target = xref
}

// HasConsumesVarsTrait implements ConsumesVarsTraitInterface
func (i *InterpolateTextOp) HasConsumesVarsTrait() bool {
	return true
}

// GetDependsOnSlotContextTrait returns the DependsOnSlotContextOpTrait
func (i *InterpolateTextOp) GetDependsOnSlotContextTrait() *ir_traits.DependsOnSlotContextOpTrait {
	return &ir_traits.DependsOnSlotContextOpTrait{
		Target:     i.Target,
		SourceSpan: i.SourceSpan,
	}
}

// BindingOp is an intermediate binding op, that has not yet been processed into an individual property,
// attribute, style, etc.
type BindingOp struct {
	ir_operations.OpBase
	Target                        ir_operations.XrefId
	BindingKind                   ir.BindingKind
	Name                          string
	Expression                    interface{} // output.OutputExpression | *Interpolation
	Unit                          *string
	SecurityContext               interface{} // core.SecurityContext | []core.SecurityContext
	IsTextAttribute               bool
	IsStructuralTemplateAttribute bool
	TemplateKind                  *ir.TemplateKind
	I18nContext                   ir_operations.XrefId
	I18nMessage                   *i18n.Message
	SourceSpan                    *util.ParseSourceSpan
}

// NewBindingOp creates a new BindingOp
func NewBindingOp(
	target ir_operations.XrefId,
	bindingKind ir.BindingKind,
	name string,
	expression interface{}, // output.OutputExpression | *Interpolation
	unit *string,
	securityContext interface{}, // core.SecurityContext | []core.SecurityContext
	isTextAttribute bool,
	isStructuralTemplateAttribute bool,
	templateKind *ir.TemplateKind,
	i18nMessage *i18n.Message,
	sourceSpan *util.ParseSourceSpan,
) *BindingOp {
	return &BindingOp{
		OpBase:                        ir_operations.NewOpBase(),
		Target:                        target,
		BindingKind:                   bindingKind,
		Name:                          name,
		Expression:                    expression,
		Unit:                          unit,
		SecurityContext:               securityContext,
		IsTextAttribute:               isTextAttribute,
		IsStructuralTemplateAttribute: isStructuralTemplateAttribute,
		TemplateKind:                  templateKind,
		I18nContext:                   0,
		I18nMessage:                   i18nMessage,
		SourceSpan:                    sourceSpan,
	}
}

// GetKind returns the operation kind
func (b *BindingOp) GetKind() ir.OpKind {
	return ir.OpKindBinding
}

// GetXref returns the xref ID
func (b *BindingOp) GetXref() ir_operations.XrefId {
	return b.Target
}

// SetXref sets the xref ID
func (b *BindingOp) SetXref(xref ir_operations.XrefId) {
	b.Target = xref
}

// PropertyOp is an operation to bind an expression to a property of an element
type PropertyOp struct {
	ir_operations.OpBase
	Target          ir_operations.XrefId
	Name            string
	Expression      interface{} // output.OutputExpression | *Interpolation
	BindingKind     ir.BindingKind
	SecurityContext interface{} // core.SecurityContext | []core.SecurityContext
	Sanitizer       output.OutputExpression
	I18nContext     ir_operations.XrefId
	I18nMessage     *i18n.Message
	SourceSpan      *util.ParseSourceSpan
}

// NewPropertyOp creates a new PropertyOp
func NewPropertyOp(
	target ir_operations.XrefId,
	name string,
	expression interface{}, // output.OutputExpression | *Interpolation
	bindingKind ir.BindingKind,
	sanitizer output.OutputExpression,
) *PropertyOp {
	return &PropertyOp{
		OpBase:      ir_operations.NewOpBase(),
		Target:      target,
		Name:        name,
		Expression:  expression,
		BindingKind: bindingKind,
		Sanitizer:   sanitizer,
		I18nContext: 0,
		I18nMessage: nil,
		SourceSpan:  nil,
	}
}

// GetKind returns the operation kind
func (p *PropertyOp) GetKind() ir.OpKind {
	return ir.OpKindProperty
}

// GetXref returns the xref ID
func (p *PropertyOp) GetXref() ir_operations.XrefId {
	return p.Target
}

// SetXref sets the xref ID
func (p *PropertyOp) SetXref(xref ir_operations.XrefId) {
	p.Target = xref
}

// StylePropOp is an operation to bind an expression to a style property of an element
type StylePropOp struct {
	ir_operations.OpBase
	Target     ir_operations.XrefId
	Name       string
	Expression interface{} // output.OutputExpression | *Interpolation
	Unit       *string
}

// NewStylePropOp creates a new StylePropOp
func NewStylePropOp(
	target ir_operations.XrefId,
	name string,
	expression interface{}, // output.OutputExpression | *Interpolation
	unit *string,
) *StylePropOp {
	return &StylePropOp{
		OpBase:     ir_operations.NewOpBase(),
		Target:     target,
		Name:       name,
		Expression: expression,
		Unit:       unit,
	}
}

// GetKind returns the operation kind
func (s *StylePropOp) GetKind() ir.OpKind {
	return ir.OpKindStyleProp
}

// GetXref returns the xref ID
func (s *StylePropOp) GetXref() ir_operations.XrefId {
	return s.Target
}

// SetXref sets the xref ID
func (s *StylePropOp) SetXref(xref ir_operations.XrefId) {
	s.Target = xref
}

// ClassPropOp is an operation to bind an expression to a class property of an element
type ClassPropOp struct {
	ir_operations.OpBase
	Target     ir_operations.XrefId
	Name       string
	Expression interface{} // output.OutputExpression | *Interpolation
}

// NewClassPropOp creates a new ClassPropOp
func NewClassPropOp(
	target ir_operations.XrefId,
	name string,
	expression interface{}, // output.OutputExpression | *Interpolation
) *ClassPropOp {
	return &ClassPropOp{
		OpBase:     ir_operations.NewOpBase(),
		Target:     target,
		Name:       name,
		Expression: expression,
	}
}

// GetKind returns the operation kind
func (c *ClassPropOp) GetKind() ir.OpKind {
	return ir.OpKindClassProp
}

// GetXref returns the xref ID
func (c *ClassPropOp) GetXref() ir_operations.XrefId {
	return c.Target
}

// SetXref sets the xref ID
func (c *ClassPropOp) SetXref(xref ir_operations.XrefId) {
	c.Target = xref
}

// StyleMapOp is an operation to bind an expression to the styles of an element
type StyleMapOp struct {
	ir_operations.OpBase
	Target     ir_operations.XrefId
	Expression interface{} // output.OutputExpression | *Interpolation
}

// NewStyleMapOp creates a new StyleMapOp
func NewStyleMapOp(
	target ir_operations.XrefId,
	expression interface{}, // output.OutputExpression | *Interpolation
) *StyleMapOp {
	return &StyleMapOp{
		OpBase:     ir_operations.NewOpBase(),
		Target:     target,
		Expression: expression,
	}
}

// GetKind returns the operation kind
func (s *StyleMapOp) GetKind() ir.OpKind {
	return ir.OpKindStyleMap
}

// GetXref returns the xref ID
func (s *StyleMapOp) GetXref() ir_operations.XrefId {
	return s.Target
}

// SetXref sets the xref ID
func (s *StyleMapOp) SetXref(xref ir_operations.XrefId) {
	s.Target = xref
}

// ClassMapOp is an operation to bind an expression to the classes of an element
type ClassMapOp struct {
	ir_operations.OpBase
	Target     ir_operations.XrefId
	Expression interface{} // output.OutputExpression | *Interpolation
}

// NewClassMapOp creates a new ClassMapOp
func NewClassMapOp(
	target ir_operations.XrefId,
	expression interface{}, // output.OutputExpression | *Interpolation
) *ClassMapOp {
	return &ClassMapOp{
		OpBase:     ir_operations.NewOpBase(),
		Target:     target,
		Expression: expression,
	}
}

// GetKind returns the operation kind
func (c *ClassMapOp) GetKind() ir.OpKind {
	return ir.OpKindClassMap
}

// GetXref returns the xref ID
func (c *ClassMapOp) GetXref() ir_operations.XrefId {
	return c.Target
}

// SetXref sets the xref ID
func (c *ClassMapOp) SetXref(xref ir_operations.XrefId) {
	c.Target = xref
}

// AdvanceOp is an operation to advance the runtime's implicit slot context during the update phase of a view
type AdvanceOp struct {
	ir_operations.OpBase
	AdvanceBy int
}

// NewAdvanceOp creates a new AdvanceOp
func NewAdvanceOp(advanceBy int) *AdvanceOp {
	return &AdvanceOp{
		OpBase:    ir_operations.NewOpBase(),
		AdvanceBy: advanceBy,
	}
}

// GetKind returns the operation kind
func (a *AdvanceOp) GetKind() ir.OpKind {
	return ir.OpKindAdvance
}

// GetXref returns 0 (AdvanceOp doesn't have an xref)
func (a *AdvanceOp) GetXref() ir_operations.XrefId {
	return 0
}

// SetXref does nothing (AdvanceOp doesn't have an xref)
func (a *AdvanceOp) SetXref(xref ir_operations.XrefId) {
	// No-op
}

// AttributeOp is an operation to associate an attribute with an element
type AttributeOp struct {
	ir_operations.OpBase
	Target                        ir_operations.XrefId
	Namespace                     *string
	Name                          string
	Expression                    interface{} // output.OutputExpression | *Interpolation
	SecurityContext               interface{} // core.SecurityContext | []core.SecurityContext
	Sanitizer                     output.OutputExpression
	IsTextAttribute               bool
	IsStructuralTemplateAttribute bool
	TemplateKind                  *ir.TemplateKind
	I18nContext                   ir_operations.XrefId
	I18nMessage                   *i18n.Message
	SourceSpan                    *util.ParseSourceSpan
}

// NewAttributeOp creates a new AttributeOp
func NewAttributeOp(
	target ir_operations.XrefId,
	namespace *string,
	name string,
	expression interface{}, // output.OutputExpression | *Interpolation
	sanitizer output.OutputExpression,
	isTextAttribute bool,
	isStructuralTemplateAttribute bool,
	templateKind *ir.TemplateKind,
) *AttributeOp {
	return &AttributeOp{
		OpBase:                        ir_operations.NewOpBase(),
		Target:                        target,
		Namespace:                     namespace,
		Name:                          name,
		Expression:                    expression,
		Sanitizer:                     sanitizer,
		IsTextAttribute:               isTextAttribute,
		IsStructuralTemplateAttribute: isStructuralTemplateAttribute,
		TemplateKind:                  templateKind,
		I18nContext:                   0,
		I18nMessage:                   nil,
		SourceSpan:                    nil,
	}
}

// GetKind returns the operation kind
func (a *AttributeOp) GetKind() ir.OpKind {
	return ir.OpKindAttribute
}

// GetXref returns the xref ID
func (a *AttributeOp) GetXref() ir_operations.XrefId {
	return a.Target
}

// SetXref sets the xref ID
func (a *AttributeOp) SetXref(xref ir_operations.XrefId) {
	a.Target = xref
}

// DomPropertyOp is a binding to a native DOM property
type DomPropertyOp struct {
	ir_operations.OpBase
	Target          ir_operations.XrefId
	Name            string
	Expression      interface{} // output.OutputExpression | *Interpolation
	BindingKind     ir.BindingKind
	SecurityContext interface{} // core.SecurityContext | []core.SecurityContext
	Sanitizer       output.OutputExpression
}

// NewDomPropertyOp creates a new DomPropertyOp
func NewDomPropertyOp(
	target ir_operations.XrefId,
	name string,
	expression interface{}, // output.OutputExpression | *Interpolation
	bindingKind ir.BindingKind,
	sanitizer output.OutputExpression,
) *DomPropertyOp {
	return &DomPropertyOp{
		OpBase:      ir_operations.NewOpBase(),
		Target:      target,
		Name:        name,
		Expression:  expression,
		BindingKind: bindingKind,
		Sanitizer:   sanitizer,
	}
}

// GetKind returns the operation kind
func (d *DomPropertyOp) GetKind() ir.OpKind {
	return ir.OpKindDomProperty
}

// GetXref returns the xref ID
func (d *DomPropertyOp) GetXref() ir_operations.XrefId {
	return d.Target
}

// SetXref sets the xref ID
func (d *DomPropertyOp) SetXref(xref ir_operations.XrefId) {
	d.Target = xref
}

// TwoWayPropertyOp is an operation to bind an expression to the property side of a two-way binding
type TwoWayPropertyOp struct {
	ir_operations.OpBase
	Target     ir_operations.XrefId
	Name       string
	Expression output.OutputExpression
	Sanitizer  output.OutputExpression
}

// NewTwoWayPropertyOp creates a new TwoWayPropertyOp
func NewTwoWayPropertyOp(
	target ir_operations.XrefId,
	name string,
	expression output.OutputExpression,
	sanitizer output.OutputExpression,
) *TwoWayPropertyOp {
	return &TwoWayPropertyOp{
		OpBase:     ir_operations.NewOpBase(),
		Target:     target,
		Name:       name,
		Expression: expression,
		Sanitizer:  sanitizer,
	}
}

// GetKind returns the operation kind
func (t *TwoWayPropertyOp) GetKind() ir.OpKind {
	return ir.OpKindTwoWayProperty
}

// GetXref returns the xref ID
func (t *TwoWayPropertyOp) GetXref() ir_operations.XrefId {
	return t.Target
}

// SetXref sets the xref ID
func (t *TwoWayPropertyOp) SetXref(xref ir_operations.XrefId) {
	t.Target = xref
}

// ControlOp is an operation to bind an expression to a `field` property of an element
type ControlOp struct {
	ir_operations.OpBase
	Target     ir_operations.XrefId
	Name       string
	Expression interface{} // output.OutputExpression | *Interpolation
	Sanitizer  output.OutputExpression
}

// NewControlOp creates a new ControlOp
func NewControlOp(
	target ir_operations.XrefId,
	name string,
	expression interface{}, // output.OutputExpression | *Interpolation
	sanitizer output.OutputExpression,
) *ControlOp {
	return &ControlOp{
		OpBase:     ir_operations.NewOpBase(),
		Target:     target,
		Name:       name,
		Expression: expression,
		Sanitizer:  sanitizer,
	}
}

// GetKind returns the operation kind
func (c *ControlOp) GetKind() ir.OpKind {
	return ir.OpKindControl
}

// GetXref returns the xref ID
func (c *ControlOp) GetXref() ir_operations.XrefId {
	return c.Target
}

// SetXref sets the xref ID
func (c *ControlOp) SetXref(xref ir_operations.XrefId) {
	c.Target = xref
}

// ConditionalOp is an op to conditionally render a template
type ConditionalOp struct {
	ir_operations.OpBase
	Target       ir_operations.XrefId
	Conditions   []interface{} // []ConditionalCaseExpr
	Processed    output.OutputExpression
	ContextValue output.OutputExpression
}

// NewConditionalOp creates a new ConditionalOp
func NewConditionalOp(
	target ir_operations.XrefId,
	conditions []interface{}, // []ConditionalCaseExpr
) *ConditionalOp {
	return &ConditionalOp{
		OpBase:       ir_operations.NewOpBase(),
		Target:       target,
		Conditions:   conditions,
		Processed:    nil,
		ContextValue: nil,
	}
}

// GetKind returns the operation kind
func (c *ConditionalOp) GetKind() ir.OpKind {
	return ir.OpKindConditional
}

// GetXref returns the xref ID
func (c *ConditionalOp) GetXref() ir_operations.XrefId {
	return c.Target
}

// SetXref sets the xref ID
func (c *ConditionalOp) SetXref(xref ir_operations.XrefId) {
	c.Target = xref
}

// HasConsumesVarsTrait implements ConsumesVarsTraitInterface
func (c *ConditionalOp) HasConsumesVarsTrait() bool {
	return true
}

// GetDependsOnSlotContextTrait returns the DependsOnSlotContextOpTrait
func (c *ConditionalOp) GetDependsOnSlotContextTrait() *ir_traits.DependsOnSlotContextOpTrait {
	return &ir_traits.DependsOnSlotContextOpTrait{
		Target:     c.Target,
		SourceSpan: nil, // TODO: Add sourceSpan if needed
	}
}

// I18nExpressionOp is an expression in an i18n message
type I18nExpressionOp struct {
	ir_operations.OpBase
	Context         ir_operations.XrefId
	Target          ir_operations.XrefId
	I18nOwner       ir_operations.XrefId
	Handle          *ir.SlotHandle
	Expression      output.OutputExpression
	IcuPlaceholder  *ir_operations.XrefId
	I18nPlaceholder *string
	ResolutionTime  ir.I18nParamResolutionTime
	Usage           ir.I18nExpressionFor
	Name            string
	SourceSpan      *util.ParseSourceSpan
}

// NewI18nExpressionOp creates a new I18nExpressionOp
func NewI18nExpressionOp(
	context ir_operations.XrefId,
	target ir_operations.XrefId,
	i18nOwner ir_operations.XrefId,
	handle *ir.SlotHandle,
	expression output.OutputExpression,
	icuPlaceholder *ir_operations.XrefId,
	i18nPlaceholder *string,
	resolutionTime ir.I18nParamResolutionTime,
	usage ir.I18nExpressionFor,
	name string,
	sourceSpan *util.ParseSourceSpan,
) *I18nExpressionOp {
	return &I18nExpressionOp{
		OpBase:          ir_operations.NewOpBase(),
		Context:         context,
		Target:          target,
		I18nOwner:       i18nOwner,
		Handle:          handle,
		Expression:      expression,
		IcuPlaceholder:  icuPlaceholder,
		I18nPlaceholder: i18nPlaceholder,
		ResolutionTime:  resolutionTime,
		Usage:           usage,
		Name:            name,
		SourceSpan:      sourceSpan,
	}
}

// GetKind returns the operation kind
func (i *I18nExpressionOp) GetKind() ir.OpKind {
	return ir.OpKindI18nExpression
}

// GetXref returns the xref ID
func (i *I18nExpressionOp) GetXref() ir_operations.XrefId {
	return i.Target
}

// SetXref sets the xref ID
func (i *I18nExpressionOp) SetXref(xref ir_operations.XrefId) {
	i.Target = xref
}

// GetDependsOnSlotContextTrait returns the DependsOnSlotContextOpTrait
func (i *I18nExpressionOp) GetDependsOnSlotContextTrait() *ir_traits.DependsOnSlotContextOpTrait {
	return &ir_traits.DependsOnSlotContextOpTrait{
		Target:     i.Target,
		SourceSpan: i.SourceSpan,
	}
}

// HasConsumesVarsTrait implements ConsumesVarsTraitInterface
func (i *I18nExpressionOp) HasConsumesVarsTrait() bool {
	return true
}

// I18nApplyOp is an instruction that applies a set of i18n expressions
type I18nApplyOp struct {
	ir_operations.OpBase
	Owner      ir_operations.XrefId
	Handle     *ir.SlotHandle
	SourceSpan *util.ParseSourceSpan
}

// NewI18nApplyOp creates a new I18nApplyOp
func NewI18nApplyOp(owner ir_operations.XrefId, handle *ir.SlotHandle, sourceSpan *util.ParseSourceSpan) *I18nApplyOp {
	return &I18nApplyOp{
		OpBase:     ir_operations.NewOpBase(),
		Owner:      owner,
		Handle:     handle,
		SourceSpan: sourceSpan,
	}
}

// GetKind returns the operation kind
func (i *I18nApplyOp) GetKind() ir.OpKind {
	return ir.OpKindI18nApply
}

// GetXref returns the xref ID (returns Owner for I18nApplyOp)
func (i *I18nApplyOp) GetXref() ir_operations.XrefId {
	return i.Owner
}

// SetXref sets the xref ID (sets Owner for I18nApplyOp)
func (i *I18nApplyOp) SetXref(xref ir_operations.XrefId) {
	i.Owner = xref
}

// RepeaterOp is an update op for a repeater
type RepeaterOp struct {
	ir_operations.OpBase
	Target     ir_operations.XrefId
	Collection output.OutputExpression
}

// NewRepeaterOp creates a new RepeaterOp
func NewRepeaterOp(
	target ir_operations.XrefId,
	collection output.OutputExpression,
) *RepeaterOp {
	return &RepeaterOp{
		OpBase:     ir_operations.NewOpBase(),
		Target:     target,
		Collection: collection,
	}
}

// GetKind returns the operation kind
func (r *RepeaterOp) GetKind() ir.OpKind {
	return ir.OpKindRepeater
}

// GetXref returns the xref ID
func (r *RepeaterOp) GetXref() ir_operations.XrefId {
	return r.Target
}

// SetXref sets the xref ID
func (r *RepeaterOp) SetXref(xref ir_operations.XrefId) {
	r.Target = xref
}

// HasConsumesVarsTrait implements ConsumesVarsTraitInterface
func (r *RepeaterOp) HasConsumesVarsTrait() bool {
	return true
}

// GetDependsOnSlotContextTrait returns the DependsOnSlotContextOpTrait
func (r *RepeaterOp) GetDependsOnSlotContextTrait() *ir_traits.DependsOnSlotContextOpTrait {
	return &ir_traits.DependsOnSlotContextOpTrait{
		Target:     r.Target,
		SourceSpan: nil, // TODO: Add sourceSpan if needed
	}
}

// DeferWhenOp controls when a `@defer` loads, using a custom expression as the condition
type DeferWhenOp struct {
	ir_operations.OpBase
	Defer      ir_operations.XrefId
	Expr       output.OutputExpression
	Modifier   ir.DeferOpModifierKind
	SourceSpan *util.ParseSourceSpan
}

// NewDeferWhenOp creates a new DeferWhenOp
func NewDeferWhenOp(
	deferXref ir_operations.XrefId,
	expr output.OutputExpression,
	modifier ir.DeferOpModifierKind,
	sourceSpan *util.ParseSourceSpan,
) *DeferWhenOp {
	return &DeferWhenOp{
		OpBase:     ir_operations.NewOpBase(),
		Defer:      deferXref,
		Expr:       expr,
		Modifier:   modifier,
		SourceSpan: sourceSpan,
	}
}

// GetKind returns the operation kind
func (d *DeferWhenOp) GetKind() ir.OpKind {
	return ir.OpKindDeferWhen
}

// GetXref returns 0 (DeferWhenOp doesn't have an xref)
func (d *DeferWhenOp) GetXref() ir_operations.XrefId {
	return 0
}

// SetXref does nothing (DeferWhenOp doesn't have an xref)
func (d *DeferWhenOp) SetXref(xref ir_operations.XrefId) {
	// No-op
}

// HasConsumesVarsTrait implements ConsumesVarsTraitInterface
func (d *DeferWhenOp) HasConsumesVarsTrait() bool {
	return true
}

// GetDependsOnSlotContextTrait returns the DependsOnSlotContextOpTrait
func (d *DeferWhenOp) GetDependsOnSlotContextTrait() *ir_traits.DependsOnSlotContextOpTrait {
	return &ir_traits.DependsOnSlotContextOpTrait{
		Target:     d.Defer,
		SourceSpan: d.SourceSpan,
	}
}

// AnimationBindingOp is an operation to bind animation css classes to an element
type AnimationBindingOp struct {
	ir_operations.OpBase
	Target     ir_operations.XrefId
	Name       string
	Expression interface{} // output.OutputExpression | *Interpolation
}

// NewAnimationBindingOp creates a new AnimationBindingOp
func NewAnimationBindingOp(
	target ir_operations.XrefId,
	name string,
	expression interface{}, // output.OutputExpression | *Interpolation
) *AnimationBindingOp {
	return &AnimationBindingOp{
		OpBase:     ir_operations.NewOpBase(),
		Target:     target,
		Name:       name,
		Expression: expression,
	}
}

// GetKind returns the operation kind
func (a *AnimationBindingOp) GetKind() ir.OpKind {
	return ir.OpKindAnimationBinding
}

// GetXref returns the xref ID
func (a *AnimationBindingOp) GetXref() ir_operations.XrefId {
	return a.Target
}

// SetXref sets the xref ID
func (a *AnimationBindingOp) SetXref(xref ir_operations.XrefId) {
	a.Target = xref
}

// StoreLetOp is an update-time operation that stores the current value of a `@let` declaration
type StoreLetOp struct {
	ir_operations.OpBase
	Target       ir_operations.XrefId
	DeclaredName string
	Value        output.OutputExpression
	SourceSpan   *util.ParseSourceSpan
}

// NewStoreLetOp creates a new StoreLetOp
func NewStoreLetOp(
	target ir_operations.XrefId,
	declaredName string,
	value output.OutputExpression,
	sourceSpan *util.ParseSourceSpan,
) *StoreLetOp {
	return &StoreLetOp{
		OpBase:       ir_operations.NewOpBase(),
		Target:       target,
		DeclaredName: declaredName,
		Value:        value,
		SourceSpan:   sourceSpan,
	}
}

// GetKind returns the operation kind
func (s *StoreLetOp) GetKind() ir.OpKind {
	return ir.OpKindStoreLet
}

// GetXref returns the xref ID
func (s *StoreLetOp) GetXref() ir_operations.XrefId {
	return s.Target
}

// SetXref sets the xref ID
func (s *StoreLetOp) SetXref(xref ir_operations.XrefId) {
	s.Target = xref
}

// HasConsumesVarsTrait implements ConsumesVarsTraitInterface
func (s *StoreLetOp) HasConsumesVarsTrait() bool {
	return true
}

// GetDependsOnSlotContextTrait returns the DependsOnSlotContextOpTrait
func (s *StoreLetOp) GetDependsOnSlotContextTrait() *ir_traits.DependsOnSlotContextOpTrait {
	return &ir_traits.DependsOnSlotContextOpTrait{
		Target:     s.Target,
		SourceSpan: s.SourceSpan,
	}
}
