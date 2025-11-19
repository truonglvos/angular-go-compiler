package ops_create

import (
	"fmt"

	"ngc-go/packages/compiler/i18n"
	"ngc-go/packages/compiler/output"
	ir "ngc-go/packages/compiler/template/pipeline/ir/src"
	ir_operations "ngc-go/packages/compiler/template/pipeline/ir/src/operations"
	ir_traits "ngc-go/packages/compiler/template/pipeline/ir/src/traits"
	"ngc-go/packages/compiler/util"
)

// LocalRef represents a local reference on an element
type LocalRef struct {
	// User-defined name of the local ref variable
	Name string
	// Target of the local reference variable (often `''`)
	Target string
}

// ElementOrContainerOpBase is the base interface for Element, ElementStart, and Template operations
type ElementOrContainerOpBase struct {
	ir_operations.OpBase
	Xref            ir_operations.XrefId
	Handle          *ir.SlotHandle
	NumSlotsUsed    int
	Attributes      ir_operations.ConstIndex
	LocalRefs       interface{} // []LocalRef | ir.ConstIndex | null
	NonBindable     bool
	StartSourceSpan *util.ParseSourceSpan
	WholeSourceSpan *util.ParseSourceSpan
}

// GetConsumesSlotTrait returns the ConsumesSlotOpTrait
func (e *ElementOrContainerOpBase) GetConsumesSlotTrait() *ir_traits.ConsumesSlotOpTrait {
	return &ir_traits.ConsumesSlotOpTrait{
		Handle:       e.Handle,
		NumSlotsUsed: e.NumSlotsUsed,
		Xref:         e.Xref,
	}
}

// GetXref returns the xref ID
func (e *ElementOrContainerOpBase) GetXref() ir_operations.XrefId {
	return e.Xref
}

// SetXref sets the xref ID
func (e *ElementOrContainerOpBase) SetXref(xref ir_operations.XrefId) {
	e.Xref = xref
}

// ElementOpBase extends ElementOrContainerOpBase with element-specific fields
type ElementOpBase struct {
	ElementOrContainerOpBase
	Tag       *string
	Namespace ir.Namespace
}

// ElementStartOp represents the start of an element in the creation IR
type ElementStartOp struct {
	ElementOpBase
	I18nPlaceholder interface{} // *i18n.TagPlaceholder
}

// NewElementStartOp creates a new ElementStartOp
func NewElementStartOp(
	tag string,
	xref ir_operations.XrefId,
	namespace ir.Namespace,
	i18nPlaceholder interface{}, // *i18n.TagPlaceholder
	startSourceSpan *util.ParseSourceSpan,
	wholeSourceSpan *util.ParseSourceSpan,
) *ElementStartOp {
	tagPtr := &tag
	return &ElementStartOp{
		ElementOpBase: ElementOpBase{
			ElementOrContainerOpBase: ElementOrContainerOpBase{
				OpBase:          ir_operations.NewOpBase(),
				Xref:            xref,
				Handle:          ir.NewSlotHandle(),
				NumSlotsUsed:    1,
				Attributes:      0,
				LocalRefs:       []LocalRef{},
				NonBindable:     false,
				StartSourceSpan: startSourceSpan,
				WholeSourceSpan: wholeSourceSpan,
			},
			Tag:       tagPtr,
			Namespace: namespace,
		},
		I18nPlaceholder: i18nPlaceholder,
	}
}

// GetKind returns the operation kind
func (e *ElementStartOp) GetKind() ir.OpKind {
	return ir.OpKindElementStart
}

// ElementOp represents an element with no children in the creation IR
type ElementOp struct {
	ElementOpBase
	I18nPlaceholder interface{} // *i18n.TagPlaceholder
}

// NewElementOp creates a new ElementOp
func NewElementOp(
	tag string,
	xref ir_operations.XrefId,
	namespace ir.Namespace,
	i18nPlaceholder interface{}, // *i18n.TagPlaceholder
	startSourceSpan *util.ParseSourceSpan,
	wholeSourceSpan *util.ParseSourceSpan,
) *ElementOp {
	tagPtr := &tag
	return &ElementOp{
		ElementOpBase: ElementOpBase{
			ElementOrContainerOpBase: ElementOrContainerOpBase{
				OpBase:          ir_operations.NewOpBase(),
				Xref:            xref,
				Handle:          ir.NewSlotHandle(),
				NumSlotsUsed:    1,
				Attributes:      0,
				LocalRefs:       []LocalRef{},
				NonBindable:     false,
				StartSourceSpan: startSourceSpan,
				WholeSourceSpan: wholeSourceSpan,
			},
			Tag:       tagPtr,
			Namespace: namespace,
		},
		I18nPlaceholder: i18nPlaceholder,
	}
}

// GetKind returns the operation kind
func (e *ElementOp) GetKind() ir.OpKind {
	return ir.OpKindElement
}

// ElementEndOp represents the end of an element structure in the creation IR
type ElementEndOp struct {
	ir_operations.OpBase
	Xref       ir_operations.XrefId
	SourceSpan *util.ParseSourceSpan
}

// NewElementEndOp creates a new ElementEndOp
func NewElementEndOp(xref ir_operations.XrefId, sourceSpan *util.ParseSourceSpan) *ElementEndOp {
	return &ElementEndOp{
		OpBase:     ir_operations.NewOpBase(),
		Xref:       xref,
		SourceSpan: sourceSpan,
	}
}

// GetKind returns the operation kind
func (e *ElementEndOp) GetKind() ir.OpKind {
	return ir.OpKindElementEnd
}

// GetXref returns the xref ID
func (e *ElementEndOp) GetXref() ir_operations.XrefId {
	return e.Xref
}

// SetXref sets the xref ID
func (e *ElementEndOp) SetXref(xref ir_operations.XrefId) {
	e.Xref = xref
}

// ContainerStartOp represents the start of a container in the creation IR
type ContainerStartOp struct {
	ElementOrContainerOpBase
}

// NewContainerStartOp creates a new ContainerStartOp
func NewContainerStartOp(
	xref ir_operations.XrefId,
	startSourceSpan *util.ParseSourceSpan,
	wholeSourceSpan *util.ParseSourceSpan,
) *ContainerStartOp {
	return &ContainerStartOp{
		ElementOrContainerOpBase: ElementOrContainerOpBase{
			OpBase:          ir_operations.NewOpBase(),
			Xref:            xref,
			Handle:          ir.NewSlotHandle(),
			NumSlotsUsed:    1,
			Attributes:      0,
			LocalRefs:       []LocalRef{},
			NonBindable:     false,
			StartSourceSpan: startSourceSpan,
			WholeSourceSpan: wholeSourceSpan,
		},
	}
}

// GetKind returns the operation kind
func (c *ContainerStartOp) GetKind() ir.OpKind {
	return ir.OpKindContainerStart
}

// ContainerOp represents an empty container in the creation IR
type ContainerOp struct {
	ElementOrContainerOpBase
}

// NewContainerOp creates a new ContainerOp
func NewContainerOp(
	xref ir_operations.XrefId,
	startSourceSpan *util.ParseSourceSpan,
	wholeSourceSpan *util.ParseSourceSpan,
) *ContainerOp {
	return &ContainerOp{
		ElementOrContainerOpBase: ElementOrContainerOpBase{
			OpBase:          ir_operations.NewOpBase(),
			Xref:            xref,
			Handle:          ir.NewSlotHandle(),
			NumSlotsUsed:    1,
			Attributes:      0,
			LocalRefs:       []LocalRef{},
			NonBindable:     false,
			StartSourceSpan: startSourceSpan,
			WholeSourceSpan: wholeSourceSpan,
		},
	}
}

// GetKind returns the operation kind
func (c *ContainerOp) GetKind() ir.OpKind {
	return ir.OpKindContainer
}

// ContainerEndOp represents the end of a container structure in the creation IR
type ContainerEndOp struct {
	ir_operations.OpBase
	Xref       ir_operations.XrefId
	SourceSpan *util.ParseSourceSpan
}

// NewContainerEndOp creates a new ContainerEndOp
func NewContainerEndOp(xref ir_operations.XrefId, sourceSpan *util.ParseSourceSpan) *ContainerEndOp {
	return &ContainerEndOp{
		OpBase:     ir_operations.NewOpBase(),
		Xref:       xref,
		SourceSpan: sourceSpan,
	}
}

// GetKind returns the operation kind
func (c *ContainerEndOp) GetKind() ir.OpKind {
	return ir.OpKindContainerEnd
}

// GetXref returns the xref ID
func (c *ContainerEndOp) GetXref() ir_operations.XrefId {
	return c.Xref
}

// SetXref sets the xref ID
func (c *ContainerEndOp) SetXref(xref ir_operations.XrefId) {
	c.Xref = xref
}

// DisableBindingsOp causes binding to be disabled in descendents of a non-bindable container
type DisableBindingsOp struct {
	ir_operations.OpBase
	Xref ir_operations.XrefId
}

// NewDisableBindingsOp creates a new DisableBindingsOp
func NewDisableBindingsOp(xref ir_operations.XrefId) *DisableBindingsOp {
	return &DisableBindingsOp{
		OpBase: ir_operations.NewOpBase(),
		Xref:   xref,
	}
}

// GetKind returns the operation kind
func (d *DisableBindingsOp) GetKind() ir.OpKind {
	return ir.OpKindDisableBindings
}

// GetXref returns the xref ID
func (d *DisableBindingsOp) GetXref() ir_operations.XrefId {
	return d.Xref
}

// SetXref sets the xref ID
func (d *DisableBindingsOp) SetXref(xref ir_operations.XrefId) {
	d.Xref = xref
}

// EnableBindingsOp causes binding to be re-enabled after visiting descendants of a non-bindable container
type EnableBindingsOp struct {
	ir_operations.OpBase
	Xref ir_operations.XrefId
}

// NewEnableBindingsOp creates a new EnableBindingsOp
func NewEnableBindingsOp(xref ir_operations.XrefId) *EnableBindingsOp {
	return &EnableBindingsOp{
		OpBase: ir_operations.NewOpBase(),
		Xref:   xref,
	}
}

// GetKind returns the operation kind
func (e *EnableBindingsOp) GetKind() ir.OpKind {
	return ir.OpKindEnableBindings
}

// GetXref returns the xref ID
func (e *EnableBindingsOp) GetXref() ir_operations.XrefId {
	return e.Xref
}

// SetXref sets the xref ID
func (e *EnableBindingsOp) SetXref(xref ir_operations.XrefId) {
	e.Xref = xref
}

// TextOp represents a text node in the creation IR
type TextOp struct {
	ir_operations.OpBase
	Xref           ir_operations.XrefId
	Handle         *ir.SlotHandle
	NumSlotsUsed   int
	InitialValue   string
	IcuPlaceholder *string
	SourceSpan     *util.ParseSourceSpan
}

// NewTextOp creates a new TextOp
func NewTextOp(
	xref ir_operations.XrefId,
	initialValue string,
	icuPlaceholder *string,
	sourceSpan *util.ParseSourceSpan,
) *TextOp {
	return &TextOp{
		OpBase:         ir_operations.NewOpBase(),
		Xref:           xref,
		Handle:         ir.NewSlotHandle(),
		NumSlotsUsed:   1,
		InitialValue:   initialValue,
		IcuPlaceholder: icuPlaceholder,
		SourceSpan:     sourceSpan,
	}
}

// GetKind returns the operation kind
func (t *TextOp) GetKind() ir.OpKind {
	return ir.OpKindText
}

// GetXref returns the xref ID
func (t *TextOp) GetXref() ir_operations.XrefId {
	return t.Xref
}

// SetXref sets the xref ID
func (t *TextOp) SetXref(xref ir_operations.XrefId) {
	t.Xref = xref
}

// GetConsumesSlotTrait returns the ConsumesSlotOpTrait
func (t *TextOp) GetConsumesSlotTrait() *ir_traits.ConsumesSlotOpTrait {
	return &ir_traits.ConsumesSlotOpTrait{
		Handle:       t.Handle,
		NumSlotsUsed: t.NumSlotsUsed,
		Xref:         t.Xref,
	}
}

// TemplateOp represents an embedded view declaration in the creation IR
type TemplateOp struct {
	ElementOpBase
	TemplateKind       ir.TemplateKind
	Decls              *int
	Vars               *int
	FunctionNameSuffix string
	I18nPlaceholder    interface{} // *i18n.TagPlaceholder | *i18n.BlockPlaceholder
}

// NewTemplateOp creates a new TemplateOp
func NewTemplateOp(
	xref ir_operations.XrefId,
	templateKind ir.TemplateKind,
	tag *string,
	functionNameSuffix string,
	namespace ir.Namespace,
	i18nPlaceholder interface{}, // *i18n.TagPlaceholder | *i18n.BlockPlaceholder
	startSourceSpan *util.ParseSourceSpan,
	wholeSourceSpan *util.ParseSourceSpan,
) *TemplateOp {
	return &TemplateOp{
		ElementOpBase: ElementOpBase{
			ElementOrContainerOpBase: ElementOrContainerOpBase{
				OpBase:          ir_operations.NewOpBase(),
				Xref:            xref,
				Handle:          ir.NewSlotHandle(),
				NumSlotsUsed:    1,
				Attributes:      0,
				LocalRefs:       []LocalRef{},
				NonBindable:     false,
				StartSourceSpan: startSourceSpan,
				WholeSourceSpan: wholeSourceSpan,
			},
			Tag:       tag,
			Namespace: namespace,
		},
		TemplateKind:       templateKind,
		Decls:              nil,
		Vars:               nil,
		FunctionNameSuffix: functionNameSuffix,
		I18nPlaceholder:    i18nPlaceholder,
	}
}

// GetKind returns the operation kind
func (t *TemplateOp) GetKind() ir.OpKind {
	return ir.OpKindTemplate
}

// ConditionalCreateOp creates a conditional (e.g. a if or switch)
type ConditionalCreateOp struct {
	ElementOpBase
	TemplateKind       ir.TemplateKind
	Decls              *int
	Vars               *int
	FunctionNameSuffix string
	I18nPlaceholder    interface{} // *i18n.TagPlaceholder | *i18n.BlockPlaceholder
}

// NewConditionalCreateOp creates a new ConditionalCreateOp
func NewConditionalCreateOp(
	xref ir_operations.XrefId,
	templateKind ir.TemplateKind,
	tag *string,
	functionNameSuffix string,
	namespace ir.Namespace,
	i18nPlaceholder interface{}, // *i18n.TagPlaceholder | *i18n.BlockPlaceholder
	startSourceSpan *util.ParseSourceSpan,
	wholeSourceSpan *util.ParseSourceSpan,
) *ConditionalCreateOp {
	return &ConditionalCreateOp{
		ElementOpBase: ElementOpBase{
			ElementOrContainerOpBase: ElementOrContainerOpBase{
				OpBase:          ir_operations.NewOpBase(),
				Xref:            xref,
				Handle:          ir.NewSlotHandle(),
				NumSlotsUsed:    1,
				Attributes:      0,
				LocalRefs:       []LocalRef{},
				NonBindable:     false,
				StartSourceSpan: startSourceSpan,
				WholeSourceSpan: wholeSourceSpan,
			},
			Tag:       tag,
			Namespace: namespace,
		},
		TemplateKind:       templateKind,
		Decls:              nil,
		Vars:               nil,
		FunctionNameSuffix: functionNameSuffix,
		I18nPlaceholder:    i18nPlaceholder,
	}
}

// GetKind returns the operation kind
func (c *ConditionalCreateOp) GetKind() ir.OpKind {
	return ir.OpKindConditionalCreate
}

// ConditionalBranchCreateOp creates a conditional branch (e.g. an else or case)
type ConditionalBranchCreateOp struct {
	ElementOpBase
	TemplateKind       ir.TemplateKind
	Decls              *int
	Vars               *int
	FunctionNameSuffix string
	I18nPlaceholder    interface{} // *i18n.TagPlaceholder | *i18n.BlockPlaceholder
}

// NewConditionalBranchCreateOp creates a new ConditionalBranchCreateOp
func NewConditionalBranchCreateOp(
	xref ir_operations.XrefId,
	templateKind ir.TemplateKind,
	tag *string,
	functionNameSuffix string,
	namespace ir.Namespace,
	i18nPlaceholder interface{}, // *i18n.TagPlaceholder | *i18n.BlockPlaceholder
	startSourceSpan *util.ParseSourceSpan,
	wholeSourceSpan *util.ParseSourceSpan,
) *ConditionalBranchCreateOp {
	return &ConditionalBranchCreateOp{
		ElementOpBase: ElementOpBase{
			ElementOrContainerOpBase: ElementOrContainerOpBase{
				OpBase:          ir_operations.NewOpBase(),
				Xref:            xref,
				Handle:          ir.NewSlotHandle(),
				NumSlotsUsed:    1,
				Attributes:      0,
				LocalRefs:       []LocalRef{},
				NonBindable:     false,
				StartSourceSpan: startSourceSpan,
				WholeSourceSpan: wholeSourceSpan,
			},
			Tag:       tag,
			Namespace: namespace,
		},
		TemplateKind:       templateKind,
		Decls:              nil,
		Vars:               nil,
		FunctionNameSuffix: functionNameSuffix,
		I18nPlaceholder:    i18nPlaceholder,
	}
}

// GetKind returns the operation kind
func (c *ConditionalBranchCreateOp) GetKind() ir.OpKind {
	return ir.OpKindConditionalBranchCreate
}

// RepeaterVarNames represents context variables available in a repeater block
type RepeaterVarNames struct {
	DollarIndex    map[string]bool // Set<string>
	DollarImplicit string
}

// RepeaterCreateOp creates a repeater (e.g. a for loop)
type RepeaterCreateOp struct {
	ElementOpBase
	Handle                *ir.SlotHandle
	NumSlotsUsed          int
	Decls                 *int
	Vars                  *int
	EmptyView             ir_operations.XrefId
	Track                 output.OutputExpression
	TrackByOps            *ir_operations.OpList
	TrackByFn             output.OutputExpression
	VarNames              RepeaterVarNames
	UsesComponentInstance bool
	FunctionNameSuffix    string
	EmptyTag              *string
	EmptyAttributes       ir_operations.ConstIndex
	I18nPlaceholder       interface{} // *i18n.BlockPlaceholder
	EmptyI18nPlaceholder  interface{} // *i18n.BlockPlaceholder
}

// NewRepeaterCreateOp creates a new RepeaterCreateOp
func NewRepeaterCreateOp(
	primaryView ir_operations.XrefId,
	emptyView ir_operations.XrefId,
	tag *string,
	track output.OutputExpression,
	varNames RepeaterVarNames,
	emptyTag *string,
	i18nPlaceholder interface{}, // *i18n.BlockPlaceholder
	emptyI18nPlaceholder interface{}, // *i18n.BlockPlaceholder
	startSourceSpan *util.ParseSourceSpan,
	wholeSourceSpan *util.ParseSourceSpan,
) *RepeaterCreateOp {
	numSlotsUsed := 2
	if emptyView != 0 {
		numSlotsUsed = 3
	}
	return &RepeaterCreateOp{
		ElementOpBase: ElementOpBase{
			ElementOrContainerOpBase: ElementOrContainerOpBase{
				OpBase:          ir_operations.NewOpBase(),
				Xref:            primaryView,
				Handle:          ir.NewSlotHandle(),
				NumSlotsUsed:    numSlotsUsed,
				Attributes:      0,
				LocalRefs:       []LocalRef{},
				NonBindable:     false,
				StartSourceSpan: startSourceSpan,
				WholeSourceSpan: wholeSourceSpan,
			},
			Tag:       tag,
			Namespace: ir.NamespaceHTML,
		},
		Handle:                ir.NewSlotHandle(),
		NumSlotsUsed:          numSlotsUsed,
		Decls:                 nil,
		Vars:                  nil,
		EmptyView:             emptyView,
		Track:                 track,
		TrackByOps:            nil,
		TrackByFn:             nil,
		VarNames:              varNames,
		UsesComponentInstance: false,
		FunctionNameSuffix:    "For",
		EmptyTag:              emptyTag,
		EmptyAttributes:       0,
		I18nPlaceholder:       i18nPlaceholder,
		EmptyI18nPlaceholder:  emptyI18nPlaceholder,
	}
}

// GetKind returns the operation kind
func (r *RepeaterCreateOp) GetKind() ir.OpKind {
	return ir.OpKindRepeaterCreate
}

// HasConsumesVarsTrait implements ConsumesVarsTraitInterface
func (r *RepeaterCreateOp) HasConsumesVarsTrait() bool {
	return true
}

// GetConsumesSlotTrait returns the ConsumesSlotOpTrait
func (r *RepeaterCreateOp) GetConsumesSlotTrait() *ir_traits.ConsumesSlotOpTrait {
	return &ir_traits.ConsumesSlotOpTrait{
		Handle:       r.Handle,
		NumSlotsUsed: r.NumSlotsUsed,
		Xref:         r.Xref,
	}
}

// IsElementOrContainerOp checks whether the given operation represents the creation of an element or container
func IsElementOrContainerOp(op ir_operations.CreateOp) bool {
	kind := op.GetKind()
	return kind == ir.OpKindElement ||
		kind == ir.OpKindElementStart ||
		kind == ir.OpKindContainer ||
		kind == ir.OpKindContainerStart ||
		kind == ir.OpKindTemplate ||
		kind == ir.OpKindRepeaterCreate ||
		kind == ir.OpKindConditionalCreate ||
		kind == ir.OpKindConditionalBranchCreate
}

// ListenerOp represents an event listener on an element in the creation IR
type ListenerOp struct {
	ir_operations.OpBase
	Target                    ir_operations.XrefId
	TargetSlot                *ir.SlotHandle
	HostListener              bool
	Name                      string
	Tag                       *string
	HandlerOps                *ir_operations.OpList
	HandlerFnName             *string
	ConsumesDollarEvent       bool
	IsLegacyAnimationListener bool
	LegacyAnimationPhase      *string
	EventTarget               *string
	SourceSpan                *util.ParseSourceSpan
}

// NewListenerOp creates a new ListenerOp
func NewListenerOp(
	target ir_operations.XrefId,
	targetSlot *ir.SlotHandle,
	name string,
	tag *string,
	handlerOps []ir_operations.UpdateOp,
	legacyAnimationPhase *string,
	eventTarget *string,
	hostListener bool,
	sourceSpan *util.ParseSourceSpan,
) *ListenerOp {
	handlerList := ir_operations.NewOpList()
	// TODO: Add handlerOps to handlerList
	_ = handlerList
	return &ListenerOp{
		OpBase:                    ir_operations.NewOpBase(),
		Target:                    target,
		TargetSlot:                targetSlot,
		Tag:                       tag,
		HostListener:              hostListener,
		Name:                      name,
		HandlerOps:                handlerList,
		HandlerFnName:             nil,
		ConsumesDollarEvent:       false,
		IsLegacyAnimationListener: legacyAnimationPhase != nil,
		LegacyAnimationPhase:      legacyAnimationPhase,
		EventTarget:               eventTarget,
		SourceSpan:                sourceSpan,
	}
}

// GetKind returns the operation kind
func (l *ListenerOp) GetKind() ir.OpKind {
	return ir.OpKindListener
}

// GetXref returns the xref ID
func (l *ListenerOp) GetXref() ir_operations.XrefId {
	return l.Target
}

// SetXref sets the xref ID
func (l *ListenerOp) SetXref(xref ir_operations.XrefId) {
	l.Target = xref
}

// TwoWayListenerOp represents the event side of a two-way binding on an element in the creation IR
type TwoWayListenerOp struct {
	ir_operations.OpBase
	Target        ir_operations.XrefId
	TargetSlot    *ir.SlotHandle
	Name          string
	Tag           *string
	HandlerOps    *ir_operations.OpList
	HandlerFnName *string
	SourceSpan    *util.ParseSourceSpan
}

// NewTwoWayListenerOp creates a new TwoWayListenerOp
func NewTwoWayListenerOp(
	target ir_operations.XrefId,
	targetSlot *ir.SlotHandle,
	name string,
	tag *string,
	handlerOps []ir_operations.UpdateOp,
	sourceSpan *util.ParseSourceSpan,
) *TwoWayListenerOp {
	handlerList := ir_operations.NewOpList()
	// TODO: Add handlerOps to handlerList
	_ = handlerList
	return &TwoWayListenerOp{
		OpBase:        ir_operations.NewOpBase(),
		Target:        target,
		TargetSlot:    targetSlot,
		Tag:           tag,
		Name:          name,
		HandlerOps:    handlerList,
		HandlerFnName: nil,
		SourceSpan:    sourceSpan,
	}
}

// GetKind returns the operation kind
func (t *TwoWayListenerOp) GetKind() ir.OpKind {
	return ir.OpKindTwoWayListener
}

// GetXref returns the xref ID
func (t *TwoWayListenerOp) GetXref() ir_operations.XrefId {
	return t.Target
}

// SetXref sets the xref ID
func (t *TwoWayListenerOp) SetXref(xref ir_operations.XrefId) {
	t.Target = xref
}

// PipeOp represents a pipe operation
type PipeOp struct {
	ir_operations.OpBase
	Xref         ir_operations.XrefId
	Handle       *ir.SlotHandle
	NumSlotsUsed int
	Name         string
}

// NewPipeOp creates a new PipeOp
func NewPipeOp(xref ir_operations.XrefId, slot *ir.SlotHandle, name string) *PipeOp {
	return &PipeOp{
		OpBase:       ir_operations.NewOpBase(),
		Xref:         xref,
		Handle:       slot,
		NumSlotsUsed: 1,
		Name:         name,
	}
}

// GetKind returns the operation kind
func (p *PipeOp) GetKind() ir.OpKind {
	return ir.OpKindPipe
}

// GetXref returns the xref ID
func (p *PipeOp) GetXref() ir_operations.XrefId {
	return p.Xref
}

// SetXref sets the xref ID
func (p *PipeOp) SetXref(xref ir_operations.XrefId) {
	p.Xref = xref
}

// GetConsumesSlotTrait returns the ConsumesSlotOpTrait
func (p *PipeOp) GetConsumesSlotTrait() *ir_traits.ConsumesSlotOpTrait {
	return &ir_traits.ConsumesSlotOpTrait{
		Handle:       p.Handle,
		NumSlotsUsed: p.NumSlotsUsed,
		Xref:         p.Xref,
	}
}

// NamespaceOp corresponds to a namespace instruction, for switching between HTML, SVG, and MathML
type NamespaceOp struct {
	ir_operations.OpBase
	Active ir.Namespace
}

// NewNamespaceOp creates a new NamespaceOp
func NewNamespaceOp(namespace ir.Namespace) *NamespaceOp {
	return &NamespaceOp{
		OpBase: ir_operations.NewOpBase(),
		Active: namespace,
	}
}

// GetKind returns the operation kind
func (n *NamespaceOp) GetKind() ir.OpKind {
	return ir.OpKindNamespace
}

// GetXref returns 0 (NamespaceOp doesn't have an xref)
func (n *NamespaceOp) GetXref() ir_operations.XrefId {
	return 0
}

// SetXref does nothing (NamespaceOp doesn't have an xref)
func (n *NamespaceOp) SetXref(xref ir_operations.XrefId) {
	// No-op
}

// ProjectionDefOp creates a content projection definition
type ProjectionDefOp struct {
	ir_operations.OpBase
	Def output.OutputExpression
}

// NewProjectionDefOp creates a new ProjectionDefOp
func NewProjectionDefOp(def output.OutputExpression) *ProjectionDefOp {
	return &ProjectionDefOp{
		OpBase: ir_operations.NewOpBase(),
		Def:    def,
	}
}

// GetKind returns the operation kind
func (p *ProjectionDefOp) GetKind() ir.OpKind {
	return ir.OpKindProjectionDef
}

// GetXref returns 0 (ProjectionDefOp doesn't have an xref)
func (p *ProjectionDefOp) GetXref() ir_operations.XrefId {
	return 0
}

// SetXref does nothing (ProjectionDefOp doesn't have an xref)
func (p *ProjectionDefOp) SetXref(xref ir_operations.XrefId) {
	// No-op
}

// ProjectionOp creates a content projection slot
type ProjectionOp struct {
	ir_operations.OpBase
	Xref                        ir_operations.XrefId
	Handle                      *ir.SlotHandle
	NumSlotsUsed                int
	ProjectionSlotIndex         int
	Attributes                  *output.LiteralArrayExpr
	LocalRefs                   []string
	Selector                    string
	I18nPlaceholder             interface{} // *i18n.TagPlaceholder
	SourceSpan                  *util.ParseSourceSpan
	FallbackView                ir_operations.XrefId
	FallbackViewI18nPlaceholder interface{} // *i18n.BlockPlaceholder
}

// NewProjectionOp creates a new ProjectionOp
func NewProjectionOp(
	xref ir_operations.XrefId,
	selector string,
	i18nPlaceholder interface{}, // *i18n.TagPlaceholder
	fallbackView ir_operations.XrefId,
	sourceSpan *util.ParseSourceSpan,
) *ProjectionOp {
	numSlotsUsed := 1
	if fallbackView != 0 {
		numSlotsUsed = 2
	}
	return &ProjectionOp{
		OpBase:                      ir_operations.NewOpBase(),
		Xref:                        xref,
		Handle:                      ir.NewSlotHandle(),
		NumSlotsUsed:                numSlotsUsed,
		Selector:                    selector,
		I18nPlaceholder:             i18nPlaceholder,
		FallbackView:                fallbackView,
		ProjectionSlotIndex:         0,
		Attributes:                  nil,
		LocalRefs:                   []string{},
		SourceSpan:                  sourceSpan,
		FallbackViewI18nPlaceholder: nil,
	}
}

// GetKind returns the operation kind
func (p *ProjectionOp) GetKind() ir.OpKind {
	return ir.OpKindProjection
}

// GetXref returns the xref ID
func (p *ProjectionOp) GetXref() ir_operations.XrefId {
	return p.Xref
}

// SetXref sets the xref ID
func (p *ProjectionOp) SetXref(xref ir_operations.XrefId) {
	p.Xref = xref
}

// GetConsumesSlotTrait returns the ConsumesSlotOpTrait
func (p *ProjectionOp) GetConsumesSlotTrait() *ir_traits.ConsumesSlotOpTrait {
	return &ir_traits.ConsumesSlotOpTrait{
		Handle:       p.Handle,
		NumSlotsUsed: p.NumSlotsUsed,
		Xref:         p.Xref,
	}
}

// ExtractedAttributeOp represents an attribute that has been extracted for inclusion in the consts array
type ExtractedAttributeOp struct {
	ir_operations.OpBase
	Target          ir_operations.XrefId
	BindingKind     ir.BindingKind
	Namespace       *string
	Name            string
	Expression      output.OutputExpression
	I18nContext     ir_operations.XrefId
	SecurityContext interface{} // core.SecurityContext | []core.SecurityContext
	TrustedValueFn  output.OutputExpression
	I18nMessage     *i18n.Message
}

// NewExtractedAttributeOp creates a new ExtractedAttributeOp
func NewExtractedAttributeOp(
	target ir_operations.XrefId,
	bindingKind ir.BindingKind,
	namespace *string,
	name string,
	expression output.OutputExpression,
	i18nContext ir_operations.XrefId,
	i18nMessage *i18n.Message,
	securityContext interface{}, // core.SecurityContext | []core.SecurityContext
) *ExtractedAttributeOp {
	return &ExtractedAttributeOp{
		OpBase:          ir_operations.NewOpBase(),
		Target:          target,
		BindingKind:     bindingKind,
		Namespace:       namespace,
		Name:            name,
		Expression:      expression,
		I18nContext:     i18nContext,
		I18nMessage:     i18nMessage,
		SecurityContext: securityContext,
		TrustedValueFn:  nil,
	}
}

// GetKind returns the operation kind
func (e *ExtractedAttributeOp) GetKind() ir.OpKind {
	return ir.OpKindExtractedAttribute
}

// GetXref returns the xref ID
func (e *ExtractedAttributeOp) GetXref() ir_operations.XrefId {
	return e.Target
}

// SetXref sets the xref ID
func (e *ExtractedAttributeOp) SetXref(xref ir_operations.XrefId) {
	e.Target = xref
}

// DeferTrigger represents a defer trigger
type DeferTrigger interface {
	GetKind() ir.DeferTriggerKind
}

// DeferTriggerWithTargetBase is the base for triggers with targets
type DeferTriggerWithTargetBase struct {
	Kind                ir.DeferTriggerKind
	TargetName          *string
	TargetXref          ir_operations.XrefId
	TargetSlot          *ir.SlotHandle
	TargetView          ir_operations.XrefId
	TargetSlotViewSteps *int
}

// GetKind returns the trigger kind
func (d *DeferTriggerWithTargetBase) GetKind() ir.DeferTriggerKind {
	return d.Kind
}

// DeferIdleTrigger represents an idle trigger
type DeferIdleTrigger struct {
	Kind ir.DeferTriggerKind
}

// GetKind returns the trigger kind
func (d *DeferIdleTrigger) GetKind() ir.DeferTriggerKind {
	return ir.DeferTriggerKindIdle
}

// DeferImmediateTrigger represents an immediate trigger
type DeferImmediateTrigger struct {
	Kind ir.DeferTriggerKind
}

// GetKind returns the trigger kind
func (d *DeferImmediateTrigger) GetKind() ir.DeferTriggerKind {
	return ir.DeferTriggerKindImmediate
}

// DeferNeverTrigger represents a never trigger
type DeferNeverTrigger struct {
	Kind ir.DeferTriggerKind
}

// GetKind returns the trigger kind
func (d *DeferNeverTrigger) GetKind() ir.DeferTriggerKind {
	return ir.DeferTriggerKindNever
}

// DeferHoverTrigger represents a hover trigger
type DeferHoverTrigger struct {
	DeferTriggerWithTargetBase
}

// DeferTimerTrigger represents a timer trigger
type DeferTimerTrigger struct {
	Kind  ir.DeferTriggerKind
	Delay int
}

// GetKind returns the trigger kind
func (d *DeferTimerTrigger) GetKind() ir.DeferTriggerKind {
	return ir.DeferTriggerKindTimer
}

// DeferInteractionTrigger represents an interaction trigger
type DeferInteractionTrigger struct {
	DeferTriggerWithTargetBase
}

// DeferViewportTrigger represents a viewport trigger
type DeferViewportTrigger struct {
	DeferTriggerWithTargetBase
	Options output.OutputExpression
}

// DeferOp configures a `@defer` block
type DeferOp struct {
	ir_operations.OpBase
	Xref                   ir_operations.XrefId
	Handle                 *ir.SlotHandle
	NumSlotsUsed           int
	MainView               ir_operations.XrefId
	MainSlot               *ir.SlotHandle
	LoadingView            ir_operations.XrefId
	LoadingSlot            *ir.SlotHandle
	PlaceholderView        ir_operations.XrefId
	PlaceholderSlot        *ir.SlotHandle
	ErrorView              ir_operations.XrefId
	ErrorSlot              *ir.SlotHandle
	PlaceholderMinimumTime *int
	LoadingMinimumTime     *int
	LoadingAfterTime       *int
	PlaceholderConfig      output.OutputExpression
	LoadingConfig          output.OutputExpression
	OwnResolverFn          output.OutputExpression
	ResolverFn             output.OutputExpression
	Flags                  ir.TDeferDetailsFlags
	SourceSpan             *util.ParseSourceSpan
}

// NewDeferOp creates a new DeferOp
func NewDeferOp(
	xref ir_operations.XrefId,
	main ir_operations.XrefId,
	mainSlot *ir.SlotHandle,
	ownResolverFn output.OutputExpression,
	resolverFn output.OutputExpression,
	sourceSpan *util.ParseSourceSpan,
) *DeferOp {
	return &DeferOp{
		OpBase:                 ir_operations.NewOpBase(),
		Xref:                   xref,
		Handle:                 ir.NewSlotHandle(),
		NumSlotsUsed:           2,
		MainView:               main,
		MainSlot:               mainSlot,
		LoadingView:            0,
		LoadingSlot:            nil,
		LoadingConfig:          nil,
		LoadingMinimumTime:     nil,
		LoadingAfterTime:       nil,
		PlaceholderView:        0,
		PlaceholderSlot:        nil,
		PlaceholderConfig:      nil,
		PlaceholderMinimumTime: nil,
		ErrorView:              0,
		ErrorSlot:              nil,
		OwnResolverFn:          ownResolverFn,
		ResolverFn:             resolverFn,
		Flags:                  0,
		SourceSpan:             sourceSpan,
	}
}

// GetKind returns the operation kind
func (d *DeferOp) GetKind() ir.OpKind {
	return ir.OpKindDefer
}

// GetXref returns the xref ID
func (d *DeferOp) GetXref() ir_operations.XrefId {
	return d.Xref
}

// SetXref sets the xref ID
func (d *DeferOp) SetXref(xref ir_operations.XrefId) {
	d.Xref = xref
}

// GetConsumesSlotTrait returns the ConsumesSlotOpTrait
func (d *DeferOp) GetConsumesSlotTrait() *ir_traits.ConsumesSlotOpTrait {
	return &ir_traits.ConsumesSlotOpTrait{
		Handle:       d.Handle,
		NumSlotsUsed: d.NumSlotsUsed,
		Xref:         d.Xref,
	}
}

// DeferOnOp controls when a `@defer` loads
type DeferOnOp struct {
	ir_operations.OpBase
	Defer      ir_operations.XrefId
	Trigger    DeferTrigger
	Modifier   ir.DeferOpModifierKind
	SourceSpan *util.ParseSourceSpan
}

// NewDeferOnOp creates a new DeferOnOp
func NewDeferOnOp(
	deferXref ir_operations.XrefId,
	trigger DeferTrigger,
	modifier ir.DeferOpModifierKind,
	sourceSpan *util.ParseSourceSpan,
) *DeferOnOp {
	return &DeferOnOp{
		OpBase:     ir_operations.NewOpBase(),
		Defer:      deferXref,
		Trigger:    trigger,
		Modifier:   modifier,
		SourceSpan: sourceSpan,
	}
}

// GetKind returns the operation kind
func (d *DeferOnOp) GetKind() ir.OpKind {
	return ir.OpKindDeferOn
}

// GetXref returns 0 (DeferOnOp doesn't have an xref)
func (d *DeferOnOp) GetXref() ir_operations.XrefId {
	return 0
}

// SetXref does nothing (DeferOnOp doesn't have an xref)
func (d *DeferOnOp) SetXref(xref ir_operations.XrefId) {
	// No-op
}

// DeclareLetOp reserves a slot during creation time for a `@let` declaration
type DeclareLetOp struct {
	ir_operations.OpBase
	Xref         ir_operations.XrefId
	Handle       *ir.SlotHandle
	NumSlotsUsed int
	SourceSpan   *util.ParseSourceSpan
	DeclaredName string
}

// NewDeclareLetOp creates a new DeclareLetOp
func NewDeclareLetOp(
	xref ir_operations.XrefId,
	declaredName string,
	sourceSpan *util.ParseSourceSpan,
) *DeclareLetOp {
	return &DeclareLetOp{
		OpBase:       ir_operations.NewOpBase(),
		Xref:         xref,
		DeclaredName: declaredName,
		SourceSpan:   sourceSpan,
		Handle:       ir.NewSlotHandle(),
		NumSlotsUsed: 1,
	}
}

// GetKind returns the operation kind
func (d *DeclareLetOp) GetKind() ir.OpKind {
	return ir.OpKindDeclareLet
}

// GetXref returns the xref ID
func (d *DeclareLetOp) GetXref() ir_operations.XrefId {
	return d.Xref
}

// SetXref sets the xref ID
func (d *DeclareLetOp) SetXref(xref ir_operations.XrefId) {
	d.Xref = xref
}

// GetConsumesSlotTrait returns the ConsumesSlotOpTrait
func (d *DeclareLetOp) GetConsumesSlotTrait() *ir_traits.ConsumesSlotOpTrait {
	return &ir_traits.ConsumesSlotOpTrait{
		Handle:       d.Handle,
		NumSlotsUsed: d.NumSlotsUsed,
		Xref:         d.Xref,
	}
}

// I18nParamValue represents a single value in an i18n param map
type I18nParamValue struct {
	// The value. This can be either a slot number, special string, or compound-value
	Value interface{} // string | int | struct{Element int; Template int}
	// The sub-template index associated with the value
	SubTemplateIndex *int
	// Flags associated with the value
	Flags ir.I18nParamValueFlags
}

// I18nMessageOp represents an i18n message that has been extracted for inclusion in the consts array
type I18nMessageOp struct {
	ir_operations.OpBase
	Xref                 ir_operations.XrefId
	I18nContext          ir_operations.XrefId
	I18nBlock            ir_operations.XrefId
	Message              *i18n.Message
	MessagePlaceholder   *string
	NeedsPostprocessing  bool
	Params               map[string]output.OutputExpression
	PostprocessingParams map[string]output.OutputExpression
	SubMessages          []ir_operations.XrefId
}

// NewI18nMessageOp creates a new I18nMessageOp
func NewI18nMessageOp(
	xref ir_operations.XrefId,
	i18nContext ir_operations.XrefId,
	i18nBlock ir_operations.XrefId,
	message *i18n.Message,
	messagePlaceholder *string,
	params map[string]output.OutputExpression,
	postprocessingParams map[string]output.OutputExpression,
	needsPostprocessing bool,
) *I18nMessageOp {
	return &I18nMessageOp{
		OpBase:               ir_operations.NewOpBase(),
		Xref:                 xref,
		I18nContext:          i18nContext,
		I18nBlock:            i18nBlock,
		Message:              message,
		MessagePlaceholder:   messagePlaceholder,
		Params:               params,
		PostprocessingParams: postprocessingParams,
		NeedsPostprocessing:  needsPostprocessing,
		SubMessages:          []ir_operations.XrefId{},
	}
}

// GetKind returns the operation kind
func (i *I18nMessageOp) GetKind() ir.OpKind {
	return ir.OpKindI18nMessage
}

// GetXref returns the xref ID
func (i *I18nMessageOp) GetXref() ir_operations.XrefId {
	return i.Xref
}

// SetXref sets the xref ID
func (i *I18nMessageOp) SetXref(xref ir_operations.XrefId) {
	i.Xref = xref
}

// I18nOpBase is the base for I18n operations
type I18nOpBase struct {
	ir_operations.OpBase
	Xref             ir_operations.XrefId
	Handle           *ir.SlotHandle
	NumSlotsUsed     int
	Root             ir_operations.XrefId
	Message          *i18n.Message
	MessageIndex     ir_operations.ConstIndex
	SubTemplateIndex *int
	Context          ir_operations.XrefId
	SourceSpan       *util.ParseSourceSpan
}

// GetConsumesSlotTrait returns the ConsumesSlotOpTrait
func (i *I18nOpBase) GetConsumesSlotTrait() *ir_traits.ConsumesSlotOpTrait {
	return &ir_traits.ConsumesSlotOpTrait{
		Handle:       i.Handle,
		NumSlotsUsed: i.NumSlotsUsed,
		Xref:         i.Xref,
	}
}

// GetXref returns the xref ID
func (i *I18nOpBase) GetXref() ir_operations.XrefId {
	return i.Xref
}

// SetXref sets the xref ID
func (i *I18nOpBase) SetXref(xref ir_operations.XrefId) {
	i.Xref = xref
}

// I18nOp represents an empty i18n block
type I18nOp struct {
	I18nOpBase
}

// NewI18nOp creates a new I18nOp
func NewI18nOp(
	xref ir_operations.XrefId,
	message *i18n.Message,
	root ir_operations.XrefId,
	sourceSpan *util.ParseSourceSpan,
) *I18nOp {
	if root == 0 {
		root = xref
	}
	return &I18nOp{
		I18nOpBase: I18nOpBase{
			OpBase:           ir_operations.NewOpBase(),
			Xref:             xref,
			Handle:           ir.NewSlotHandle(),
			NumSlotsUsed:     1,
			Root:             root,
			Message:          message,
			MessageIndex:     0,
			SubTemplateIndex: nil,
			Context:          0,
			SourceSpan:       sourceSpan,
		},
	}
}

// GetKind returns the operation kind
func (i *I18nOp) GetKind() ir.OpKind {
	return ir.OpKindI18n
}

// I18nStartOp represents the start of an i18n block
type I18nStartOp struct {
	I18nOpBase
}

// NewI18nStartOp creates a new I18nStartOp
func NewI18nStartOp(
	xref ir_operations.XrefId,
	message *i18n.Message,
	root ir_operations.XrefId,
	sourceSpan *util.ParseSourceSpan,
) *I18nStartOp {
	if root == 0 {
		root = xref
	}
	return &I18nStartOp{
		I18nOpBase: I18nOpBase{
			OpBase:           ir_operations.NewOpBase(),
			Xref:             xref,
			Handle:           ir.NewSlotHandle(),
			NumSlotsUsed:     1,
			Root:             root,
			Message:          message,
			MessageIndex:     0,
			SubTemplateIndex: nil,
			Context:          0,
			SourceSpan:       sourceSpan,
		},
	}
}

// GetKind returns the operation kind
func (i *I18nStartOp) GetKind() ir.OpKind {
	return ir.OpKindI18nStart
}

// I18nEndOp represents the end of an i18n block
type I18nEndOp struct {
	ir_operations.OpBase
	Xref       ir_operations.XrefId
	SourceSpan *util.ParseSourceSpan
}

// NewI18nEndOp creates a new I18nEndOp
func NewI18nEndOp(xref ir_operations.XrefId, sourceSpan *util.ParseSourceSpan) *I18nEndOp {
	return &I18nEndOp{
		OpBase:     ir_operations.NewOpBase(),
		Xref:       xref,
		SourceSpan: sourceSpan,
	}
}

// GetKind returns the operation kind
func (i *I18nEndOp) GetKind() ir.OpKind {
	return ir.OpKindI18nEnd
}

// GetXref returns the xref ID
func (i *I18nEndOp) GetXref() ir_operations.XrefId {
	return i.Xref
}

// SetXref sets the xref ID
func (i *I18nEndOp) SetXref(xref ir_operations.XrefId) {
	i.Xref = xref
}

// IcuStartOp represents the start of an ICU expression
type IcuStartOp struct {
	ir_operations.OpBase
	Xref               ir_operations.XrefId
	Message            *i18n.Message
	MessagePlaceholder string
	Context            ir_operations.XrefId
	SourceSpan         *util.ParseSourceSpan
}

// NewIcuStartOp creates a new IcuStartOp
func NewIcuStartOp(
	xref ir_operations.XrefId,
	message *i18n.Message,
	messagePlaceholder string,
	sourceSpan *util.ParseSourceSpan,
) *IcuStartOp {
	return &IcuStartOp{
		OpBase:             ir_operations.NewOpBase(),
		Xref:               xref,
		Message:            message,
		MessagePlaceholder: messagePlaceholder,
		Context:            0,
		SourceSpan:         sourceSpan,
	}
}

// GetKind returns the operation kind
func (i *IcuStartOp) GetKind() ir.OpKind {
	return ir.OpKindIcuStart
}

// GetXref returns the xref ID
func (i *IcuStartOp) GetXref() ir_operations.XrefId {
	return i.Xref
}

// SetXref sets the xref ID
func (i *IcuStartOp) SetXref(xref ir_operations.XrefId) {
	i.Xref = xref
}

// IcuEndOp represents the end of an ICU expression
type IcuEndOp struct {
	ir_operations.OpBase
	Xref ir_operations.XrefId
}

// NewIcuEndOp creates a new IcuEndOp
func NewIcuEndOp(xref ir_operations.XrefId) *IcuEndOp {
	return &IcuEndOp{
		OpBase: ir_operations.NewOpBase(),
		Xref:   xref,
	}
}

// GetKind returns the operation kind
func (i *IcuEndOp) GetKind() ir.OpKind {
	return ir.OpKindIcuEnd
}

// GetXref returns the xref ID
func (i *IcuEndOp) GetXref() ir_operations.XrefId {
	return i.Xref
}

// SetXref sets the xref ID
func (i *IcuEndOp) SetXref(xref ir_operations.XrefId) {
	i.Xref = xref
}

// IcuPlaceholderOp represents a placeholder in an ICU expression
type IcuPlaceholderOp struct {
	ir_operations.OpBase
	Xref                   ir_operations.XrefId
	Name                   string
	Strings                []string
	ExpressionPlaceholders []I18nParamValue
}

// NewIcuPlaceholderOp creates a new IcuPlaceholderOp
func NewIcuPlaceholderOp(
	xref ir_operations.XrefId,
	name string,
	strings []string,
) *IcuPlaceholderOp {
	return &IcuPlaceholderOp{
		OpBase:                 ir_operations.NewOpBase(),
		Xref:                   xref,
		Name:                   name,
		Strings:                strings,
		ExpressionPlaceholders: []I18nParamValue{},
	}
}

// GetKind returns the operation kind
func (i *IcuPlaceholderOp) GetKind() ir.OpKind {
	return ir.OpKindIcuPlaceholder
}

// GetXref returns the xref ID
func (i *IcuPlaceholderOp) GetXref() ir_operations.XrefId {
	return i.Xref
}

// SetXref sets the xref ID
func (i *IcuPlaceholderOp) SetXref(xref ir_operations.XrefId) {
	i.Xref = xref
}

// I18nContextOp is an i18n context that is used to generate a translated i18n message
type I18nContextOp struct {
	ir_operations.OpBase
	ContextKind          ir.I18nContextKind
	Xref                 ir_operations.XrefId
	I18nBlock            ir_operations.XrefId
	Message              *i18n.Message
	Params               map[string][]I18nParamValue
	PostprocessingParams map[string][]I18nParamValue
	SourceSpan           *util.ParseSourceSpan
}

// NewI18nContextOp creates a new I18nContextOp
func NewI18nContextOp(
	contextKind ir.I18nContextKind,
	xref ir_operations.XrefId,
	i18nBlock ir_operations.XrefId,
	message *i18n.Message,
	sourceSpan *util.ParseSourceSpan,
) (*I18nContextOp, error) {
	if i18nBlock == 0 && contextKind != ir.I18nContextKindAttr {
		return nil, fmt.Errorf("i18nBlock must be provided for non-attribute contexts")
	}
	return &I18nContextOp{
		OpBase:               ir_operations.NewOpBase(),
		ContextKind:          contextKind,
		Xref:                 xref,
		I18nBlock:            i18nBlock,
		Message:              message,
		SourceSpan:           sourceSpan,
		Params:               make(map[string][]I18nParamValue),
		PostprocessingParams: make(map[string][]I18nParamValue),
	}, nil
}

// GetKind returns the operation kind
func (i *I18nContextOp) GetKind() ir.OpKind {
	return ir.OpKindI18nContext
}

// GetXref returns the xref ID
func (i *I18nContextOp) GetXref() ir_operations.XrefId {
	return i.Xref
}

// SetXref sets the xref ID
func (i *I18nContextOp) SetXref(xref ir_operations.XrefId) {
	i.Xref = xref
}

// I18nAttributesOp corresponds to i18n attributes on an element
type I18nAttributesOp struct {
	ir_operations.OpBase
	Xref                 ir_operations.XrefId
	Handle               *ir.SlotHandle
	NumSlotsUsed         int
	Target               ir_operations.XrefId
	I18nAttributesConfig ir_operations.ConstIndex
}

// NewI18nAttributesOp creates a new I18nAttributesOp
func NewI18nAttributesOp(
	xref ir_operations.XrefId,
	handle *ir.SlotHandle,
	target ir_operations.XrefId,
) *I18nAttributesOp {
	return &I18nAttributesOp{
		OpBase:               ir_operations.NewOpBase(),
		Xref:                 xref,
		Handle:               handle,
		NumSlotsUsed:         1,
		Target:               target,
		I18nAttributesConfig: 0,
	}
}

// GetKind returns the operation kind
func (i *I18nAttributesOp) GetKind() ir.OpKind {
	return ir.OpKindI18nAttributes
}

// GetXref returns the xref ID
func (i *I18nAttributesOp) GetXref() ir_operations.XrefId {
	return i.Xref
}

// SetXref sets the xref ID
func (i *I18nAttributesOp) SetXref(xref ir_operations.XrefId) {
	i.Xref = xref
}

// GetConsumesSlotTrait returns the ConsumesSlotOpTrait
func (i *I18nAttributesOp) GetConsumesSlotTrait() *ir_traits.ConsumesSlotOpTrait {
	return &ir_traits.ConsumesSlotOpTrait{
		Handle:       i.Handle,
		NumSlotsUsed: i.NumSlotsUsed,
		Xref:         i.Xref,
	}
}

// ElementSourceLocation describes a location at which an element is defined within a template
type ElementSourceLocation struct {
	TargetSlot *ir.SlotHandle
	Offset     int
	Line       int
	Column     int
}

// SourceLocationOp attaches the location at which each element is defined within the source template
type SourceLocationOp struct {
	ir_operations.OpBase
	TemplatePath string
	Locations    []ElementSourceLocation
}

// NewSourceLocationOp creates a new SourceLocationOp
func NewSourceLocationOp(
	templatePath string,
	locations []ElementSourceLocation,
) *SourceLocationOp {
	return &SourceLocationOp{
		OpBase:       ir_operations.NewOpBase(),
		TemplatePath: templatePath,
		Locations:    locations,
	}
}

// GetKind returns the operation kind
func (s *SourceLocationOp) GetKind() ir.OpKind {
	return ir.OpKindSourceLocation
}

// GetXref returns 0 (SourceLocationOp doesn't have an xref)
func (s *SourceLocationOp) GetXref() ir_operations.XrefId {
	return 0
}

// SetXref does nothing (SourceLocationOp doesn't have an xref)
func (s *SourceLocationOp) SetXref(xref ir_operations.XrefId) {
	// No-op
}

// ControlCreateOp determines whether a `[control]` binding targets a specialized control directive
type ControlCreateOp struct {
	ir_operations.OpBase
	SourceSpan *util.ParseSourceSpan
}

// NewControlCreateOp creates a new ControlCreateOp
func NewControlCreateOp(sourceSpan *util.ParseSourceSpan) *ControlCreateOp {
	return &ControlCreateOp{
		OpBase:     ir_operations.NewOpBase(),
		SourceSpan: sourceSpan,
	}
}

// GetKind returns the operation kind
func (c *ControlCreateOp) GetKind() ir.OpKind {
	return ir.OpKindControlCreate
}

// GetXref returns 0 (ControlCreateOp doesn't have an xref)
func (c *ControlCreateOp) GetXref() ir_operations.XrefId {
	return 0
}

// SetXref does nothing (ControlCreateOp doesn't have an xref)
func (c *ControlCreateOp) SetXref(xref ir_operations.XrefId) {
	// No-op
}

// AnimationStringOp represents binding to an animation in the create IR
type AnimationStringOp struct {
	ir_operations.OpBase
	Target          ir_operations.XrefId
	Name            string
	AnimationKind   ir.AnimationKind
	Expression      interface{} // output.OutputExpression | *Interpolation
	I18nMessage     ir_operations.XrefId
	SecurityContext interface{} // core.SecurityContext | []core.SecurityContext
	Sanitizer       output.OutputExpression
	SourceSpan      *util.ParseSourceSpan
}

// NewAnimationStringOp creates a new AnimationStringOp
func NewAnimationStringOp(
	name string,
	target ir_operations.XrefId,
	animationKind ir.AnimationKind,
	expression interface{}, // output.OutputExpression | *Interpolation
	securityContext interface{}, // core.SecurityContext | []core.SecurityContext
	sourceSpan *util.ParseSourceSpan,
) *AnimationStringOp {
	return &AnimationStringOp{
		OpBase:          ir_operations.NewOpBase(),
		Name:            name,
		Target:          target,
		AnimationKind:   animationKind,
		Expression:      expression,
		I18nMessage:     0,
		SecurityContext: securityContext,
		Sanitizer:       nil,
		SourceSpan:      sourceSpan,
	}
}

// GetKind returns the operation kind
func (a *AnimationStringOp) GetKind() ir.OpKind {
	return ir.OpKindAnimationString
}

// GetXref returns the xref ID
func (a *AnimationStringOp) GetXref() ir_operations.XrefId {
	return a.Target
}

// SetXref sets the xref ID
func (a *AnimationStringOp) SetXref(xref ir_operations.XrefId) {
	a.Target = xref
}

// AnimationOp represents binding to an animation in the create IR
type AnimationOp struct {
	ir_operations.OpBase
	Target          ir_operations.XrefId
	Name            string
	AnimationKind   ir.AnimationKind
	HandlerOps      *ir_operations.OpList
	HandlerFnName   *string
	I18nMessage     ir_operations.XrefId
	SecurityContext interface{} // core.SecurityContext | []core.SecurityContext
	Sanitizer       output.OutputExpression
	SourceSpan      *util.ParseSourceSpan
}

// NewAnimationOp creates a new AnimationOp
func NewAnimationOp(
	name string,
	target ir_operations.XrefId,
	animationKind ir.AnimationKind,
	callbackOps []ir_operations.UpdateOp,
	securityContext interface{}, // core.SecurityContext | []core.SecurityContext
	sourceSpan *util.ParseSourceSpan,
) *AnimationOp {
	handlerOps := ir_operations.NewOpList()
	// TODO: Add callbackOps to handlerOps
	_ = handlerOps
	return &AnimationOp{
		OpBase:          ir_operations.NewOpBase(),
		Name:            name,
		Target:          target,
		AnimationKind:   animationKind,
		HandlerOps:      handlerOps,
		HandlerFnName:   nil,
		I18nMessage:     0,
		SecurityContext: securityContext,
		Sanitizer:       nil,
		SourceSpan:      sourceSpan,
	}
}

// GetKind returns the operation kind
func (a *AnimationOp) GetKind() ir.OpKind {
	return ir.OpKindAnimation
}

// GetXref returns the xref ID
func (a *AnimationOp) GetXref() ir_operations.XrefId {
	return a.Target
}

// SetXref sets the xref ID
func (a *AnimationOp) SetXref(xref ir_operations.XrefId) {
	a.Target = xref
}

// AnimationListenerOp represents an animation listener
type AnimationListenerOp struct {
	ir_operations.OpBase
	Target              ir_operations.XrefId
	TargetSlot          *ir.SlotHandle
	HostListener        bool
	Name                string
	AnimationKind       ir.AnimationKind
	Tag                 *string
	HandlerOps          *ir_operations.OpList
	HandlerFnName       *string
	ConsumesDollarEvent bool
	EventTarget         *string
	SourceSpan          *util.ParseSourceSpan
}

// NewAnimationListenerOp creates a new AnimationListenerOp
func NewAnimationListenerOp(
	target ir_operations.XrefId,
	targetSlot *ir.SlotHandle,
	name string,
	tag *string,
	handlerOps []ir_operations.UpdateOp,
	animationKind ir.AnimationKind,
	eventTarget *string,
	hostListener bool,
	sourceSpan *util.ParseSourceSpan,
) *AnimationListenerOp {
	handlerList := ir_operations.NewOpList()
	// TODO: Add handlerOps to handlerList
	_ = handlerList
	return &AnimationListenerOp{
		OpBase:              ir_operations.NewOpBase(),
		Target:              target,
		TargetSlot:          targetSlot,
		Tag:                 tag,
		HostListener:        hostListener,
		Name:                name,
		AnimationKind:       animationKind,
		HandlerOps:          handlerList,
		HandlerFnName:       nil,
		ConsumesDollarEvent: false,
		EventTarget:         eventTarget,
		SourceSpan:          sourceSpan,
	}
}

// GetKind returns the operation kind
func (a *AnimationListenerOp) GetKind() ir.OpKind {
	return ir.OpKindAnimationListener
}

// GetXref returns the xref ID
func (a *AnimationListenerOp) GetXref() ir_operations.XrefId {
	return a.Target
}

// SetXref sets the xref ID
func (a *AnimationListenerOp) SetXref(xref ir_operations.XrefId) {
	a.Target = xref
}
