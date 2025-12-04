package render3

import (
	"ngc-go/packages/compiler/src/core"
	"ngc-go/packages/compiler/src/expression_parser"
	"ngc-go/packages/compiler/src/util"
)

// I18nMeta represents i18n metadata (can be *i18n.Message or other types)
type I18nMeta interface{}

// Node represents a node in the R3 AST
type Node interface {
	SourceSpan() *util.ParseSourceSpan
	Visit(visitor Visitor) interface{}
}

// Comment represents a comment node
type Comment struct {
	Value      string
	sourceSpan *util.ParseSourceSpan
}

// NewComment creates a new Comment node
func NewComment(value string, sourceSpan *util.ParseSourceSpan) *Comment {
	return &Comment{
		Value:      value,
		sourceSpan: sourceSpan,
	}
}

// SourceSpan returns the source span
func (c *Comment) SourceSpan() *util.ParseSourceSpan {
	return c.sourceSpan
}

// Visit visits the node with a visitor
func (c *Comment) Visit(visitor Visitor) interface{} {
	panic("visit() not implemented for Comment")
}

// Text represents a text node
type Text struct {
	Value      string
	sourceSpan *util.ParseSourceSpan
}

// NewText creates a new Text node
func NewText(value string, sourceSpan *util.ParseSourceSpan) *Text {
	return &Text{
		Value:      value,
		sourceSpan: sourceSpan,
	}
}

// SourceSpan returns the source span
func (t *Text) SourceSpan() *util.ParseSourceSpan {
	return t.sourceSpan
}

// Visit visits the node with a visitor
func (t *Text) Visit(visitor Visitor) interface{} {
	return visitor.VisitText(t)
}

// BoundText represents a bound text node
type BoundText struct {
	Value      expression_parser.AST
	sourceSpan *util.ParseSourceSpan
	I18n       I18nMeta
}

// NewBoundText creates a new BoundText node
func NewBoundText(value expression_parser.AST, sourceSpan *util.ParseSourceSpan, i18nMeta I18nMeta) *BoundText {
	return &BoundText{
		Value:      value,
		sourceSpan: sourceSpan,
		I18n:       i18nMeta,
	}
}

// SourceSpan returns the source span
func (bt *BoundText) SourceSpan() *util.ParseSourceSpan {
	return bt.sourceSpan
}

// Visit visits the node with a visitor
func (bt *BoundText) Visit(visitor Visitor) interface{} {
	return visitor.VisitBoundText(bt)
}

// TextAttribute represents a text attribute
type TextAttribute struct {
	Name       string
	Value      string
	sourceSpan *util.ParseSourceSpan
	KeySpan    *util.ParseSourceSpan
	ValueSpan  *util.ParseSourceSpan
	I18n       I18nMeta
}

// NewTextAttribute creates a new TextAttribute
func NewTextAttribute(name, value string, sourceSpan *util.ParseSourceSpan, keySpan *util.ParseSourceSpan, valueSpan *util.ParseSourceSpan, i18nMeta I18nMeta) *TextAttribute {
	return &TextAttribute{
		Name:       name,
		Value:      value,
		sourceSpan: sourceSpan,
		KeySpan:    keySpan,
		ValueSpan:  valueSpan,
		I18n:       i18nMeta,
	}
}

// SourceSpan returns the source span
func (ta *TextAttribute) SourceSpan() *util.ParseSourceSpan {
	return ta.sourceSpan
}

// Visit visits the node with a visitor
func (ta *TextAttribute) Visit(visitor Visitor) interface{} {
	return visitor.VisitTextAttribute(ta)
}

// BoundAttribute represents a bound attribute
type BoundAttribute struct {
	Name            string
	Type            expression_parser.BindingType
	SecurityContext core.SecurityContext
	Value           expression_parser.AST
	Unit            *string
	sourceSpan      *util.ParseSourceSpan
	KeySpan         *util.ParseSourceSpan
	ValueSpan       *util.ParseSourceSpan
	I18n            I18nMeta
}

// NewBoundAttribute creates a new BoundAttribute
func NewBoundAttribute(
	name string,
	bindingType expression_parser.BindingType,
	securityContext core.SecurityContext,
	value expression_parser.AST,
	unit *string,
	sourceSpan *util.ParseSourceSpan,
	keySpan *util.ParseSourceSpan,
	valueSpan *util.ParseSourceSpan,
	i18nMeta I18nMeta,
) *BoundAttribute {
	return &BoundAttribute{
		Name:            name,
		Type:            bindingType,
		SecurityContext: securityContext,
		Value:           value,
		Unit:            unit,
		sourceSpan:      sourceSpan,
		KeySpan:         keySpan,
		ValueSpan:       valueSpan,
		I18n:            i18nMeta,
	}
}

// FromBoundElementProperty creates a BoundAttribute from a BoundElementProperty
func FromBoundElementProperty(prop *expression_parser.BoundElementProperty, i18nMeta I18nMeta) *BoundAttribute {
	if prop.KeySpan == nil {
		panic("Unexpected state: keySpan must be defined for bound attributes")
	}
	return NewBoundAttribute(
		prop.Name,
		prop.Type,
		prop.SecurityContext,
		prop.Value,
		prop.Unit,
		prop.SourceSpan,
		prop.KeySpan,
		prop.ValueSpan,
		i18nMeta,
	)
}

// SourceSpan returns the source span
func (ba *BoundAttribute) SourceSpan() *util.ParseSourceSpan {
	return ba.sourceSpan
}

// Visit visits the node with a visitor
func (ba *BoundAttribute) Visit(visitor Visitor) interface{} {
	return visitor.VisitBoundAttribute(ba)
}

// BoundEvent represents a bound event
type BoundEvent struct {
	Name        string
	Type        expression_parser.ParsedEventType
	Handler     expression_parser.AST
	Target      *string
	Phase       *string
	sourceSpan  *util.ParseSourceSpan
	HandlerSpan *util.ParseSourceSpan
	KeySpan     *util.ParseSourceSpan
}

// NewBoundEvent creates a new BoundEvent
func NewBoundEvent(
	name string,
	eventType expression_parser.ParsedEventType,
	handler expression_parser.AST,
	target *string,
	phase *string,
	sourceSpan *util.ParseSourceSpan,
	handlerSpan *util.ParseSourceSpan,
	keySpan *util.ParseSourceSpan,
) *BoundEvent {
	return &BoundEvent{
		Name:        name,
		Type:        eventType,
		Handler:     handler,
		Target:      target,
		Phase:       phase,
		sourceSpan:  sourceSpan,
		HandlerSpan: handlerSpan,
		KeySpan:     keySpan,
	}
}

// FromParsedEvent creates a BoundEvent from a ParsedEvent
func FromParsedEvent(event *expression_parser.ParsedEvent) *BoundEvent {
	var target *string
	var phase *string

	if event.Type == expression_parser.ParsedEventTypeRegular {
		target = event.TargetOrPhase
	} else if event.Type == expression_parser.ParsedEventTypeLegacyAnimation {
		phase = event.TargetOrPhase
	}

	if event.KeySpan == nil {
		panic("Unexpected state: keySpan must be defined for bound event")
	}

	return NewBoundEvent(
		event.Name,
		event.Type,
		event.Handler,
		target,
		phase,
		event.SourceSpan,
		event.HandlerSpan,
		event.KeySpan,
	)
}

// SourceSpan returns the source span
func (be *BoundEvent) SourceSpan() *util.ParseSourceSpan {
	return be.sourceSpan
}

// Visit visits the node with a visitor
func (be *BoundEvent) Visit(visitor Visitor) interface{} {
	return visitor.VisitBoundEvent(be)
}

// Element represents an element node
type Element struct {
	Name            string
	Attributes      []*TextAttribute
	Inputs          []*BoundAttribute
	Outputs         []*BoundEvent
	Directives      []*Directive
	Children        []Node
	References      []*Reference
	IsSelfClosing   bool
	sourceSpan      *util.ParseSourceSpan
	StartSourceSpan *util.ParseSourceSpan
	EndSourceSpan   *util.ParseSourceSpan
	IsVoid          bool
	I18n            I18nMeta
}

// NewElement creates a new Element node
func NewElement(
	name string,
	attributes []*TextAttribute,
	inputs []*BoundAttribute,
	outputs []*BoundEvent,
	directives []*Directive,
	children []Node,
	references []*Reference,
	isSelfClosing bool,
	sourceSpan *util.ParseSourceSpan,
	startSourceSpan *util.ParseSourceSpan,
	endSourceSpan *util.ParseSourceSpan,
	isVoid bool,
	i18nMeta I18nMeta,
) *Element {
	return &Element{
		Name:            name,
		Attributes:      attributes,
		Inputs:          inputs,
		Outputs:         outputs,
		Directives:      directives,
		Children:        children,
		References:      references,
		IsSelfClosing:   isSelfClosing,
		sourceSpan:      sourceSpan,
		StartSourceSpan: startSourceSpan,
		EndSourceSpan:   endSourceSpan,
		IsVoid:          isVoid,
		I18n:            i18nMeta,
	}
}

// SourceSpan returns the source span
func (e *Element) SourceSpan() *util.ParseSourceSpan {
	return e.sourceSpan
}

// GetName returns the element name
func (e *Element) GetName() string {
	return e.Name
}

// GetAttributes returns the text attributes
func (e *Element) GetAttributes() []*TextAttribute {
	return e.Attributes
}

// GetInputs returns the bound attributes (inputs)
func (e *Element) GetInputs() []*BoundAttribute {
	return e.Inputs
}

// GetOutputs returns the bound events (outputs)
func (e *Element) GetOutputs() []*BoundEvent {
	return e.Outputs
}

// Visit visits the node with a visitor
func (e *Element) Visit(visitor Visitor) interface{} {
	return visitor.VisitElement(e)
}

// DeferredTrigger is the base class for deferred triggers
type DeferredTrigger struct {
	NameSpan           *util.ParseSourceSpan
	sourceSpan         *util.ParseSourceSpan
	PrefetchSpan       *util.ParseSourceSpan
	WhenOrOnSourceSpan *util.ParseSourceSpan
	HydrateSpan        *util.ParseSourceSpan
}

// NewDeferredTrigger creates a new DeferredTrigger
func NewDeferredTrigger(
	nameSpan *util.ParseSourceSpan,
	sourceSpan *util.ParseSourceSpan,
	prefetchSpan *util.ParseSourceSpan,
	whenOrOnSourceSpan *util.ParseSourceSpan,
	hydrateSpan *util.ParseSourceSpan,
) *DeferredTrigger {
	return &DeferredTrigger{
		NameSpan:           nameSpan,
		sourceSpan:         sourceSpan,
		PrefetchSpan:       prefetchSpan,
		WhenOrOnSourceSpan: whenOrOnSourceSpan,
		HydrateSpan:        hydrateSpan,
	}
}

// SourceSpan returns the source span
func (dt *DeferredTrigger) SourceSpan() *util.ParseSourceSpan {
	return dt.sourceSpan
}

// Visit visits the node with a visitor
func (dt *DeferredTrigger) Visit(visitor Visitor) interface{} {
	return visitor.VisitDeferredTrigger(dt)
}

// BoundDeferredTrigger represents a bound deferred trigger
type BoundDeferredTrigger struct {
	*DeferredTrigger
	Value expression_parser.AST
}

// NewBoundDeferredTrigger creates a new BoundDeferredTrigger
func NewBoundDeferredTrigger(
	value expression_parser.AST,
	sourceSpan *util.ParseSourceSpan,
	prefetchSpan *util.ParseSourceSpan,
	whenSourceSpan *util.ParseSourceSpan,
	hydrateSpan *util.ParseSourceSpan,
) *BoundDeferredTrigger {
	return &BoundDeferredTrigger{
		DeferredTrigger: NewDeferredTrigger(
			nil, // nameSpan is null for 'when' triggers
			sourceSpan,
			prefetchSpan,
			whenSourceSpan,
			hydrateSpan,
		),
		Value: value,
	}
}

// NeverDeferredTrigger represents a never deferred trigger
type NeverDeferredTrigger struct {
	*DeferredTrigger
}

// NewNeverDeferredTrigger creates a new NeverDeferredTrigger
func NewNeverDeferredTrigger(
	nameSpan *util.ParseSourceSpan,
	sourceSpan *util.ParseSourceSpan,
	prefetchSpan *util.ParseSourceSpan,
	onSourceSpan *util.ParseSourceSpan,
	hydrateSpan *util.ParseSourceSpan,
) *NeverDeferredTrigger {
	return &NeverDeferredTrigger{
		DeferredTrigger: NewDeferredTrigger(nameSpan, sourceSpan, prefetchSpan, onSourceSpan, hydrateSpan),
	}
}

// IdleDeferredTrigger represents an idle deferred trigger
type IdleDeferredTrigger struct {
	*DeferredTrigger
}

// NewIdleDeferredTrigger creates a new IdleDeferredTrigger
func NewIdleDeferredTrigger(
	nameSpan *util.ParseSourceSpan,
	sourceSpan *util.ParseSourceSpan,
	prefetchSpan *util.ParseSourceSpan,
	onSourceSpan *util.ParseSourceSpan,
	hydrateSpan *util.ParseSourceSpan,
) *IdleDeferredTrigger {
	return &IdleDeferredTrigger{
		DeferredTrigger: NewDeferredTrigger(nameSpan, sourceSpan, prefetchSpan, onSourceSpan, hydrateSpan),
	}
}

// ImmediateDeferredTrigger represents an immediate deferred trigger
type ImmediateDeferredTrigger struct {
	*DeferredTrigger
}

// NewImmediateDeferredTrigger creates a new ImmediateDeferredTrigger
func NewImmediateDeferredTrigger(
	nameSpan *util.ParseSourceSpan,
	sourceSpan *util.ParseSourceSpan,
	prefetchSpan *util.ParseSourceSpan,
	onSourceSpan *util.ParseSourceSpan,
	hydrateSpan *util.ParseSourceSpan,
) *ImmediateDeferredTrigger {
	return &ImmediateDeferredTrigger{
		DeferredTrigger: NewDeferredTrigger(nameSpan, sourceSpan, prefetchSpan, onSourceSpan, hydrateSpan),
	}
}

// HoverDeferredTrigger represents a hover deferred trigger
type HoverDeferredTrigger struct {
	*DeferredTrigger
	Reference *string
}

// NewHoverDeferredTrigger creates a new HoverDeferredTrigger
func NewHoverDeferredTrigger(
	reference *string,
	nameSpan *util.ParseSourceSpan,
	sourceSpan *util.ParseSourceSpan,
	prefetchSpan *util.ParseSourceSpan,
	onSourceSpan *util.ParseSourceSpan,
	hydrateSpan *util.ParseSourceSpan,
) *HoverDeferredTrigger {
	return &HoverDeferredTrigger{
		DeferredTrigger: NewDeferredTrigger(nameSpan, sourceSpan, prefetchSpan, onSourceSpan, hydrateSpan),
		Reference:       reference,
	}
}

// TimerDeferredTrigger represents a timer deferred trigger
type TimerDeferredTrigger struct {
	*DeferredTrigger
	Delay int
}

// NewTimerDeferredTrigger creates a new TimerDeferredTrigger
func NewTimerDeferredTrigger(
	delay int,
	nameSpan *util.ParseSourceSpan,
	sourceSpan *util.ParseSourceSpan,
	prefetchSpan *util.ParseSourceSpan,
	onSourceSpan *util.ParseSourceSpan,
	hydrateSpan *util.ParseSourceSpan,
) *TimerDeferredTrigger {
	return &TimerDeferredTrigger{
		DeferredTrigger: NewDeferredTrigger(nameSpan, sourceSpan, prefetchSpan, onSourceSpan, hydrateSpan),
		Delay:           delay,
	}
}

// InteractionDeferredTrigger represents an interaction deferred trigger
type InteractionDeferredTrigger struct {
	*DeferredTrigger
	Reference *string
}

// NewInteractionDeferredTrigger creates a new InteractionDeferredTrigger
func NewInteractionDeferredTrigger(
	reference *string,
	nameSpan *util.ParseSourceSpan,
	sourceSpan *util.ParseSourceSpan,
	prefetchSpan *util.ParseSourceSpan,
	onSourceSpan *util.ParseSourceSpan,
	hydrateSpan *util.ParseSourceSpan,
) *InteractionDeferredTrigger {
	return &InteractionDeferredTrigger{
		DeferredTrigger: NewDeferredTrigger(nameSpan, sourceSpan, prefetchSpan, onSourceSpan, hydrateSpan),
		Reference:       reference,
	}
}

// ViewportDeferredTrigger represents a viewport deferred trigger
type ViewportDeferredTrigger struct {
	*DeferredTrigger
	Reference *string
	Options   *expression_parser.LiteralMap
}

// NewViewportDeferredTrigger creates a new ViewportDeferredTrigger
func NewViewportDeferredTrigger(
	reference *string,
	options *expression_parser.LiteralMap,
	nameSpan *util.ParseSourceSpan,
	sourceSpan *util.ParseSourceSpan,
	prefetchSpan *util.ParseSourceSpan,
	onSourceSpan *util.ParseSourceSpan,
	hydrateSpan *util.ParseSourceSpan,
) *ViewportDeferredTrigger {
	return &ViewportDeferredTrigger{
		DeferredTrigger: NewDeferredTrigger(nameSpan, sourceSpan, prefetchSpan, onSourceSpan, hydrateSpan),
		Reference:       reference,
		Options:         options,
	}
}

// DeferredBlockTriggers represents all deferred block triggers
type DeferredBlockTriggers struct {
	When        *BoundDeferredTrigger
	Idle        *IdleDeferredTrigger
	Immediate   *ImmediateDeferredTrigger
	Hover       *HoverDeferredTrigger
	Timer       *TimerDeferredTrigger
	Interaction *InteractionDeferredTrigger
	Viewport    *ViewportDeferredTrigger
	Never       *NeverDeferredTrigger
}

// BlockNode is the base class for block nodes
type BlockNode struct {
	NameSpan        *util.ParseSourceSpan
	sourceSpan      *util.ParseSourceSpan
	StartSourceSpan *util.ParseSourceSpan
	EndSourceSpan   *util.ParseSourceSpan
}

// NewBlockNode creates a new BlockNode
func NewBlockNode(
	nameSpan *util.ParseSourceSpan,
	sourceSpan *util.ParseSourceSpan,
	startSourceSpan *util.ParseSourceSpan,
	endSourceSpan *util.ParseSourceSpan,
) *BlockNode {
	return &BlockNode{
		NameSpan:        nameSpan,
		sourceSpan:      sourceSpan,
		StartSourceSpan: startSourceSpan,
		EndSourceSpan:   endSourceSpan,
	}
}

// SourceSpan returns the source span
func (bn *BlockNode) SourceSpan() *util.ParseSourceSpan {
	return bn.sourceSpan
}

// DeferredBlockPlaceholder represents a deferred block placeholder
type DeferredBlockPlaceholder struct {
	*BlockNode
	Children    []Node
	MinimumTime *int
	I18n        I18nMeta
}

// NewDeferredBlockPlaceholder creates a new DeferredBlockPlaceholder
func NewDeferredBlockPlaceholder(
	children []Node,
	minimumTime *int,
	nameSpan *util.ParseSourceSpan,
	sourceSpan *util.ParseSourceSpan,
	startSourceSpan *util.ParseSourceSpan,
	endSourceSpan *util.ParseSourceSpan,
	i18nMeta I18nMeta,
) *DeferredBlockPlaceholder {
	return &DeferredBlockPlaceholder{
		BlockNode:   NewBlockNode(nameSpan, sourceSpan, startSourceSpan, endSourceSpan),
		Children:    children,
		MinimumTime: minimumTime,
		I18n:        i18nMeta,
	}
}

// Visit visits the node with a visitor
func (dbp *DeferredBlockPlaceholder) Visit(visitor Visitor) interface{} {
	return visitor.VisitDeferredBlockPlaceholder(dbp)
}

// DeferredBlockLoading represents a deferred block loading state
type DeferredBlockLoading struct {
	*BlockNode
	Children    []Node
	AfterTime   *int
	MinimumTime *int
	I18n        I18nMeta
}

// NewDeferredBlockLoading creates a new DeferredBlockLoading
func NewDeferredBlockLoading(
	children []Node,
	afterTime *int,
	minimumTime *int,
	nameSpan *util.ParseSourceSpan,
	sourceSpan *util.ParseSourceSpan,
	startSourceSpan *util.ParseSourceSpan,
	endSourceSpan *util.ParseSourceSpan,
	i18nMeta I18nMeta,
) *DeferredBlockLoading {
	return &DeferredBlockLoading{
		BlockNode:   NewBlockNode(nameSpan, sourceSpan, startSourceSpan, endSourceSpan),
		Children:    children,
		AfterTime:   afterTime,
		MinimumTime: minimumTime,
		I18n:        i18nMeta,
	}
}

// Visit visits the node with a visitor
func (dbl *DeferredBlockLoading) Visit(visitor Visitor) interface{} {
	return visitor.VisitDeferredBlockLoading(dbl)
}

// DeferredBlockError represents a deferred block error state
type DeferredBlockError struct {
	*BlockNode
	Children []Node
	I18n     I18nMeta
}

// NewDeferredBlockError creates a new DeferredBlockError
func NewDeferredBlockError(
	children []Node,
	nameSpan *util.ParseSourceSpan,
	sourceSpan *util.ParseSourceSpan,
	startSourceSpan *util.ParseSourceSpan,
	endSourceSpan *util.ParseSourceSpan,
	i18nMeta I18nMeta,
) *DeferredBlockError {
	return &DeferredBlockError{
		BlockNode: NewBlockNode(nameSpan, sourceSpan, startSourceSpan, endSourceSpan),
		Children:  children,
		I18n:      i18nMeta,
	}
}

// Visit visits the node with a visitor
func (dbe *DeferredBlockError) Visit(visitor Visitor) interface{} {
	return visitor.VisitDeferredBlockError(dbe)
}

// DeferredBlock represents a deferred block
type DeferredBlock struct {
	*BlockNode
	Children                []Node
	Triggers                *DeferredBlockTriggers
	PrefetchTriggers        *DeferredBlockTriggers
	HydrateTriggers         *DeferredBlockTriggers
	Placeholder             *DeferredBlockPlaceholder
	Loading                 *DeferredBlockLoading
	Error                   *DeferredBlockError
	MainBlockSpan           *util.ParseSourceSpan
	I18n                    I18nMeta
	definedTriggers         []string
	definedPrefetchTriggers []string
	definedHydrateTriggers  []string
}

// NewDeferredBlock creates a new DeferredBlock
func NewDeferredBlock(
	children []Node,
	triggers *DeferredBlockTriggers,
	prefetchTriggers *DeferredBlockTriggers,
	hydrateTriggers *DeferredBlockTriggers,
	placeholder *DeferredBlockPlaceholder,
	loading *DeferredBlockLoading,
	errorBlock *DeferredBlockError,
	nameSpan *util.ParseSourceSpan,
	sourceSpan *util.ParseSourceSpan,
	mainBlockSpan *util.ParseSourceSpan,
	startSourceSpan *util.ParseSourceSpan,
	endSourceSpan *util.ParseSourceSpan,
	i18nMeta I18nMeta,
) *DeferredBlock {
	db := &DeferredBlock{
		BlockNode:        NewBlockNode(nameSpan, sourceSpan, startSourceSpan, endSourceSpan),
		Children:         children,
		Triggers:         triggers,
		PrefetchTriggers: prefetchTriggers,
		HydrateTriggers:  hydrateTriggers,
		Placeholder:      placeholder,
		Loading:          loading,
		Error:            errorBlock,
		MainBlockSpan:    mainBlockSpan,
		I18n:             i18nMeta,
	}

	// Cache the keys for efficient traversal
	db.definedTriggers = getDefinedTriggerKeys(triggers)
	db.definedPrefetchTriggers = getDefinedTriggerKeys(prefetchTriggers)
	db.definedHydrateTriggers = getDefinedTriggerKeys(hydrateTriggers)

	return db
}

// getDefinedTriggerKeys extracts the keys of defined triggers
func getDefinedTriggerKeys(triggers *DeferredBlockTriggers) []string {
	keys := []string{}
	if triggers == nil {
		return keys
	}
	if triggers.When != nil {
		keys = append(keys, "when")
	}
	if triggers.Idle != nil {
		keys = append(keys, "idle")
	}
	if triggers.Immediate != nil {
		keys = append(keys, "immediate")
	}
	if triggers.Hover != nil {
		keys = append(keys, "hover")
	}
	if triggers.Timer != nil {
		keys = append(keys, "timer")
	}
	if triggers.Interaction != nil {
		keys = append(keys, "interaction")
	}
	if triggers.Viewport != nil {
		keys = append(keys, "viewport")
	}
	if triggers.Never != nil {
		keys = append(keys, "never")
	}
	return keys
}

// Visit visits the node with a visitor
func (db *DeferredBlock) Visit(visitor Visitor) interface{} {
	return visitor.VisitDeferredBlock(db)
}

// VisitAll visits all nodes in the deferred block
func (db *DeferredBlock) VisitAll(visitor Visitor) {
	// Visit hydrate triggers first to match insertion order
	db.visitTriggers(db.definedHydrateTriggers, db.HydrateTriggers, visitor)
	db.visitTriggers(db.definedTriggers, db.Triggers, visitor)
	db.visitTriggers(db.definedPrefetchTriggers, db.PrefetchTriggers, visitor)
	VisitAll(visitor, db.Children)

	remainingBlocks := []Node{}
	if db.Placeholder != nil {
		remainingBlocks = append(remainingBlocks, db.Placeholder)
	}
	if db.Loading != nil {
		remainingBlocks = append(remainingBlocks, db.Loading)
	}
	if db.Error != nil {
		remainingBlocks = append(remainingBlocks, db.Error)
	}
	VisitAll(visitor, remainingBlocks)
}

// visitTriggers visits triggers by keys
func (db *DeferredBlock) visitTriggers(keys []string, triggers *DeferredBlockTriggers, visitor Visitor) {
	if triggers == nil {
		return
	}
	for _, key := range keys {
		var trigger Node
		switch key {
		case "when":
			if triggers.When != nil {
				trigger = triggers.When
			}
		case "idle":
			if triggers.Idle != nil {
				trigger = triggers.Idle
			}
		case "immediate":
			if triggers.Immediate != nil {
				trigger = triggers.Immediate
			}
		case "hover":
			if triggers.Hover != nil {
				trigger = triggers.Hover
			}
		case "timer":
			if triggers.Timer != nil {
				trigger = triggers.Timer
			}
		case "interaction":
			if triggers.Interaction != nil {
				trigger = triggers.Interaction
			}
		case "viewport":
			if triggers.Viewport != nil {
				trigger = triggers.Viewport
			}
		case "never":
			if triggers.Never != nil {
				trigger = triggers.Never
			}
		}
		if trigger != nil {
			trigger.Visit(visitor)
		}
	}
}

// SwitchBlock represents a switch block
type SwitchBlock struct {
	*BlockNode
	Expression    expression_parser.AST
	Cases         []*SwitchBlockCase
	UnknownBlocks []*UnknownBlock
}

// NewSwitchBlock creates a new SwitchBlock
func NewSwitchBlock(
	expression expression_parser.AST,
	cases []*SwitchBlockCase,
	unknownBlocks []*UnknownBlock,
	sourceSpan *util.ParseSourceSpan,
	startSourceSpan *util.ParseSourceSpan,
	endSourceSpan *util.ParseSourceSpan,
	nameSpan *util.ParseSourceSpan,
) *SwitchBlock {
	return &SwitchBlock{
		BlockNode:     NewBlockNode(nameSpan, sourceSpan, startSourceSpan, endSourceSpan),
		Expression:    expression,
		Cases:         cases,
		UnknownBlocks: unknownBlocks,
	}
}

// Visit visits the node with a visitor
func (sb *SwitchBlock) Visit(visitor Visitor) interface{} {
	return visitor.VisitSwitchBlock(sb)
}

// SwitchBlockCase represents a switch block case
type SwitchBlockCase struct {
	*BlockNode
	Expression expression_parser.AST
	Children   []Node
	I18n       I18nMeta
}

// NewSwitchBlockCase creates a new SwitchBlockCase
func NewSwitchBlockCase(
	expression expression_parser.AST,
	children []Node,
	sourceSpan *util.ParseSourceSpan,
	startSourceSpan *util.ParseSourceSpan,
	endSourceSpan *util.ParseSourceSpan,
	nameSpan *util.ParseSourceSpan,
	i18nMeta I18nMeta,
) *SwitchBlockCase {
	return &SwitchBlockCase{
		BlockNode:  NewBlockNode(nameSpan, sourceSpan, startSourceSpan, endSourceSpan),
		Expression: expression,
		Children:   children,
		I18n:       i18nMeta,
	}
}

// Visit visits the node with a visitor
func (sbc *SwitchBlockCase) Visit(visitor Visitor) interface{} {
	return visitor.VisitSwitchBlockCase(sbc)
}

// ForLoopBlock represents a for loop block
type ForLoopBlock struct {
	*BlockNode
	Item             *Variable
	Expression       *expression_parser.ASTWithSource
	TrackBy          *expression_parser.ASTWithSource
	TrackKeywordSpan *util.ParseSourceSpan
	ContextVariables []*Variable
	Children         []Node
	Empty            *ForLoopBlockEmpty
	MainBlockSpan    *util.ParseSourceSpan
	I18n             I18nMeta
}

// NewForLoopBlock creates a new ForLoopBlock
func NewForLoopBlock(
	item *Variable,
	expression *expression_parser.ASTWithSource,
	trackBy *expression_parser.ASTWithSource,
	trackKeywordSpan *util.ParseSourceSpan,
	contextVariables []*Variable,
	children []Node,
	empty *ForLoopBlockEmpty,
	sourceSpan *util.ParseSourceSpan,
	mainBlockSpan *util.ParseSourceSpan,
	startSourceSpan *util.ParseSourceSpan,
	endSourceSpan *util.ParseSourceSpan,
	nameSpan *util.ParseSourceSpan,
	i18nMeta I18nMeta,
) *ForLoopBlock {
	return &ForLoopBlock{
		BlockNode:        NewBlockNode(nameSpan, sourceSpan, startSourceSpan, endSourceSpan),
		Item:             item,
		Expression:       expression,
		TrackBy:          trackBy,
		TrackKeywordSpan: trackKeywordSpan,
		ContextVariables: contextVariables,
		Children:         children,
		Empty:            empty,
		MainBlockSpan:    mainBlockSpan,
		I18n:             i18nMeta,
	}
}

// Visit visits the node with a visitor
func (flb *ForLoopBlock) Visit(visitor Visitor) interface{} {
	return visitor.VisitForLoopBlock(flb)
}

// ForLoopBlockEmpty represents the empty block of a for loop
type ForLoopBlockEmpty struct {
	*BlockNode
	Children []Node
	I18n     I18nMeta
}

// NewForLoopBlockEmpty creates a new ForLoopBlockEmpty
func NewForLoopBlockEmpty(
	children []Node,
	sourceSpan *util.ParseSourceSpan,
	startSourceSpan *util.ParseSourceSpan,
	endSourceSpan *util.ParseSourceSpan,
	nameSpan *util.ParseSourceSpan,
	i18nMeta I18nMeta,
) *ForLoopBlockEmpty {
	return &ForLoopBlockEmpty{
		BlockNode: NewBlockNode(nameSpan, sourceSpan, startSourceSpan, endSourceSpan),
		Children:  children,
		I18n:      i18nMeta,
	}
}

// Visit visits the node with a visitor
func (flbe *ForLoopBlockEmpty) Visit(visitor Visitor) interface{} {
	return visitor.VisitForLoopBlockEmpty(flbe)
}

// IfBlock represents an if block
type IfBlock struct {
	*BlockNode
	Branches []*IfBlockBranch
}

// NewIfBlock creates a new IfBlock
func NewIfBlock(
	branches []*IfBlockBranch,
	sourceSpan *util.ParseSourceSpan,
	startSourceSpan *util.ParseSourceSpan,
	endSourceSpan *util.ParseSourceSpan,
	nameSpan *util.ParseSourceSpan,
) *IfBlock {
	return &IfBlock{
		BlockNode: NewBlockNode(nameSpan, sourceSpan, startSourceSpan, endSourceSpan),
		Branches:  branches,
	}
}

// Visit visits the node with a visitor
func (ib *IfBlock) Visit(visitor Visitor) interface{} {
	return visitor.VisitIfBlock(ib)
}

// IfBlockBranch represents a branch of an if block
type IfBlockBranch struct {
	*BlockNode
	Expression      expression_parser.AST
	Children        []Node
	ExpressionAlias *Variable
	I18n            I18nMeta
}

// NewIfBlockBranch creates a new IfBlockBranch
func NewIfBlockBranch(
	expression expression_parser.AST,
	children []Node,
	expressionAlias *Variable,
	sourceSpan *util.ParseSourceSpan,
	startSourceSpan *util.ParseSourceSpan,
	endSourceSpan *util.ParseSourceSpan,
	nameSpan *util.ParseSourceSpan,
	i18nMeta I18nMeta,
) *IfBlockBranch {
	return &IfBlockBranch{
		BlockNode:       NewBlockNode(nameSpan, sourceSpan, startSourceSpan, endSourceSpan),
		Expression:      expression,
		Children:        children,
		ExpressionAlias: expressionAlias,
		I18n:            i18nMeta,
	}
}

// Visit visits the node with a visitor
func (ibb *IfBlockBranch) Visit(visitor Visitor) interface{} {
	return visitor.VisitIfBlockBranch(ibb)
}

// UnknownBlock represents an unknown block
type UnknownBlock struct {
	Name       string
	sourceSpan *util.ParseSourceSpan
	NameSpan   *util.ParseSourceSpan
}

// NewUnknownBlock creates a new UnknownBlock
func NewUnknownBlock(
	name string,
	sourceSpan *util.ParseSourceSpan,
	nameSpan *util.ParseSourceSpan,
) *UnknownBlock {
	return &UnknownBlock{
		Name:       name,
		sourceSpan: sourceSpan,
		NameSpan:   nameSpan,
	}
}

// SourceSpan returns the source span
func (ub *UnknownBlock) SourceSpan() *util.ParseSourceSpan {
	return ub.sourceSpan
}

// Visit visits the node with a visitor
func (ub *UnknownBlock) Visit(visitor Visitor) interface{} {
	return visitor.VisitUnknownBlock(ub)
}

// LetDeclaration represents a let declaration
type LetDeclaration struct {
	Name       string
	Value      expression_parser.AST
	sourceSpan *util.ParseSourceSpan
	NameSpan   *util.ParseSourceSpan
	ValueSpan  *util.ParseSourceSpan
}

// NewLetDeclaration creates a new LetDeclaration
func NewLetDeclaration(
	name string,
	value expression_parser.AST,
	sourceSpan *util.ParseSourceSpan,
	nameSpan *util.ParseSourceSpan,
	valueSpan *util.ParseSourceSpan,
) *LetDeclaration {
	return &LetDeclaration{
		Name:       name,
		Value:      value,
		sourceSpan: sourceSpan,
		NameSpan:   nameSpan,
		ValueSpan:  valueSpan,
	}
}

// SourceSpan returns the source span
func (ld *LetDeclaration) SourceSpan() *util.ParseSourceSpan {
	return ld.sourceSpan
}

// Visit visits the node with a visitor
func (ld *LetDeclaration) Visit(visitor Visitor) interface{} {
	return visitor.VisitLetDeclaration(ld)
}

// Component represents a component node
type Component struct {
	ComponentName   string
	TagName         *string
	FullName        string
	Attributes      []*TextAttribute
	Inputs          []*BoundAttribute
	Outputs         []*BoundEvent
	Directives      []*Directive
	Children        []Node
	References      []*Reference
	IsSelfClosing   bool
	sourceSpan      *util.ParseSourceSpan
	StartSourceSpan *util.ParseSourceSpan
	EndSourceSpan   *util.ParseSourceSpan
	I18n            I18nMeta
}

// NewComponent creates a new Component node
func NewComponent(
	componentName string,
	tagName *string,
	fullName string,
	attributes []*TextAttribute,
	inputs []*BoundAttribute,
	outputs []*BoundEvent,
	directives []*Directive,
	children []Node,
	references []*Reference,
	isSelfClosing bool,
	sourceSpan *util.ParseSourceSpan,
	startSourceSpan *util.ParseSourceSpan,
	endSourceSpan *util.ParseSourceSpan,
	i18nMeta I18nMeta,
) *Component {
	return &Component{
		ComponentName:   componentName,
		TagName:         tagName,
		FullName:        fullName,
		Attributes:      attributes,
		Inputs:          inputs,
		Outputs:         outputs,
		Directives:      directives,
		Children:        children,
		References:      references,
		IsSelfClosing:   isSelfClosing,
		sourceSpan:      sourceSpan,
		StartSourceSpan: startSourceSpan,
		EndSourceSpan:   endSourceSpan,
		I18n:            i18nMeta,
	}
}

// SourceSpan returns the source span
func (c *Component) SourceSpan() *util.ParseSourceSpan {
	return c.sourceSpan
}

// Visit visits the node with a visitor
func (c *Component) Visit(visitor Visitor) interface{} {
	return visitor.VisitComponent(c)
}

// Directive represents a directive node
type Directive struct {
	Name            string
	Attributes      []*TextAttribute
	Inputs          []*BoundAttribute
	Outputs         []*BoundEvent
	References      []*Reference
	sourceSpan      *util.ParseSourceSpan
	StartSourceSpan *util.ParseSourceSpan
	EndSourceSpan   *util.ParseSourceSpan
	I18n            I18nMeta
}

// NewDirective creates a new Directive node
func NewDirective(
	name string,
	attributes []*TextAttribute,
	inputs []*BoundAttribute,
	outputs []*BoundEvent,
	references []*Reference,
	sourceSpan *util.ParseSourceSpan,
	startSourceSpan *util.ParseSourceSpan,
	endSourceSpan *util.ParseSourceSpan,
	i18nMeta I18nMeta,
) *Directive {
	return &Directive{
		Name:            name,
		Attributes:      attributes,
		Inputs:          inputs,
		Outputs:         outputs,
		References:      references,
		sourceSpan:      sourceSpan,
		StartSourceSpan: startSourceSpan,
		EndSourceSpan:   endSourceSpan,
		I18n:            i18nMeta,
	}
}

// SourceSpan returns the source span
func (d *Directive) SourceSpan() *util.ParseSourceSpan {
	return d.sourceSpan
}

// Visit visits the node with a visitor
func (d *Directive) Visit(visitor Visitor) interface{} {
	return visitor.VisitDirective(d)
}

// Template represents a template node
type Template struct {
	TagName         *string
	Attributes      []*TextAttribute
	Inputs          []*BoundAttribute
	Outputs         []*BoundEvent
	Directives      []*Directive
	TemplateAttrs   []interface{} // BoundAttribute | TextAttribute
	Children        []Node
	References      []*Reference
	Variables       []*Variable
	IsSelfClosing   bool
	sourceSpan      *util.ParseSourceSpan
	StartSourceSpan *util.ParseSourceSpan
	EndSourceSpan   *util.ParseSourceSpan
	I18n            I18nMeta
}

// NewTemplate creates a new Template node
func NewTemplate(
	tagName *string,
	attributes []*TextAttribute,
	inputs []*BoundAttribute,
	outputs []*BoundEvent,
	directives []*Directive,
	templateAttrs []interface{},
	children []Node,
	references []*Reference,
	variables []*Variable,
	isSelfClosing bool,
	sourceSpan *util.ParseSourceSpan,
	startSourceSpan *util.ParseSourceSpan,
	endSourceSpan *util.ParseSourceSpan,
	i18nMeta I18nMeta,
) *Template {
	return &Template{
		TagName:         tagName,
		Attributes:      attributes,
		Inputs:          inputs,
		Outputs:         outputs,
		Directives:      directives,
		TemplateAttrs:   templateAttrs,
		Children:        children,
		References:      references,
		Variables:       variables,
		IsSelfClosing:   isSelfClosing,
		sourceSpan:      sourceSpan,
		StartSourceSpan: startSourceSpan,
		EndSourceSpan:   endSourceSpan,
		I18n:            i18nMeta,
	}
}

// SourceSpan returns the source span
func (t *Template) SourceSpan() *util.ParseSourceSpan {
	return t.sourceSpan
}

// Visit visits the node with a visitor
func (t *Template) Visit(visitor Visitor) interface{} {
	return visitor.VisitTemplate(t)
}

// GetTagName returns the tag name
func (t *Template) GetTagName() *string {
	return t.TagName
}

// GetAttributes returns the text attributes
func (t *Template) GetAttributes() []*TextAttribute {
	return t.Attributes
}

// GetInputs returns the bound attributes (inputs)
func (t *Template) GetInputs() []*BoundAttribute {
	return t.Inputs
}

// GetTemplateAttrs returns the template attributes (for backward compatibility)
func (t *Template) GetTemplateAttrs() []interface{} {
	return t.TemplateAttrs
}

// Content represents a content node
type Content struct {
	Selector        string
	Attributes      []*TextAttribute
	Children        []Node
	IsSelfClosing   bool
	sourceSpan      *util.ParseSourceSpan
	StartSourceSpan *util.ParseSourceSpan
	EndSourceSpan   *util.ParseSourceSpan
	I18n            I18nMeta
}

// NewContent creates a new Content node
func NewContent(
	selector string,
	attributes []*TextAttribute,
	children []Node,
	isSelfClosing bool,
	sourceSpan *util.ParseSourceSpan,
	startSourceSpan *util.ParseSourceSpan,
	endSourceSpan *util.ParseSourceSpan,
	i18nMeta I18nMeta,
) *Content {
	return &Content{
		Selector:        selector,
		Attributes:      attributes,
		Children:        children,
		IsSelfClosing:   isSelfClosing,
		sourceSpan:      sourceSpan,
		StartSourceSpan: startSourceSpan,
		EndSourceSpan:   endSourceSpan,
		I18n:            i18nMeta,
	}
}

// SourceSpan returns the source span
func (c *Content) SourceSpan() *util.ParseSourceSpan {
	return c.sourceSpan
}

// Visit visits the node with a visitor
func (c *Content) Visit(visitor Visitor) interface{} {
	return visitor.VisitContent(c)
}

// Variable represents a variable node
type Variable struct {
	Name       string
	Value      string
	sourceSpan *util.ParseSourceSpan
	KeySpan    *util.ParseSourceSpan
	ValueSpan  *util.ParseSourceSpan
}

// NewVariable creates a new Variable node
func NewVariable(
	name string,
	value string,
	sourceSpan *util.ParseSourceSpan,
	keySpan *util.ParseSourceSpan,
	valueSpan *util.ParseSourceSpan,
) *Variable {
	return &Variable{
		Name:       name,
		Value:      value,
		sourceSpan: sourceSpan,
		KeySpan:    keySpan,
		ValueSpan:  valueSpan,
	}
}

// SourceSpan returns the source span
func (v *Variable) SourceSpan() *util.ParseSourceSpan {
	return v.sourceSpan
}

// Visit visits the node with a visitor
func (v *Variable) Visit(visitor Visitor) interface{} {
	return visitor.VisitVariable(v)
}

// Reference represents a reference node
type Reference struct {
	Name       string
	Value      string
	sourceSpan *util.ParseSourceSpan
	KeySpan    *util.ParseSourceSpan
	ValueSpan  *util.ParseSourceSpan
}

// NewReference creates a new Reference node
func NewReference(
	name string,
	value string,
	sourceSpan *util.ParseSourceSpan,
	keySpan *util.ParseSourceSpan,
	valueSpan *util.ParseSourceSpan,
) *Reference {
	return &Reference{
		Name:       name,
		Value:      value,
		sourceSpan: sourceSpan,
		KeySpan:    keySpan,
		ValueSpan:  valueSpan,
	}
}

// SourceSpan returns the source span
func (r *Reference) SourceSpan() *util.ParseSourceSpan {
	return r.sourceSpan
}

// Visit visits the node with a visitor
func (r *Reference) Visit(visitor Visitor) interface{} {
	return visitor.VisitReference(r)
}

// Icu represents an ICU node
type Icu struct {
	Vars         map[string]*BoundText
	Placeholders map[string]Node // Text | BoundText
	sourceSpan   *util.ParseSourceSpan
	I18n         I18nMeta
}

// NewIcu creates a new Icu node
func NewIcu(
	vars map[string]*BoundText,
	placeholders map[string]Node,
	sourceSpan *util.ParseSourceSpan,
	i18nMeta I18nMeta,
) *Icu {
	return &Icu{
		Vars:         vars,
		Placeholders: placeholders,
		sourceSpan:   sourceSpan,
		I18n:         i18nMeta,
	}
}

// SourceSpan returns the source span
func (i *Icu) SourceSpan() *util.ParseSourceSpan {
	return i.sourceSpan
}

// Visit visits the node with a visitor
func (i *Icu) Visit(visitor Visitor) interface{} {
	return visitor.VisitIcu(i)
}

// HostElement represents a host element node
type HostElement struct {
	TagNames   []string
	Bindings   []*BoundAttribute
	Listeners  []*BoundEvent
	sourceSpan *util.ParseSourceSpan
}

// NewHostElement creates a new HostElement
func NewHostElement(
	tagNames []string,
	bindings []*BoundAttribute,
	listeners []*BoundEvent,
	sourceSpan *util.ParseSourceSpan,
) *HostElement {
	if len(tagNames) == 0 {
		panic("HostElement must have at least one tag name")
	}
	return &HostElement{
		TagNames:   tagNames,
		Bindings:   bindings,
		Listeners:  listeners,
		sourceSpan: sourceSpan,
	}
}

// SourceSpan returns the source span
func (he *HostElement) SourceSpan() *util.ParseSourceSpan {
	return he.sourceSpan
}

// Visit visits the node with a visitor
func (he *HostElement) Visit(visitor Visitor) interface{} {
	panic("HostElement cannot be visited")
}

// Visitor is the interface for visiting R3 AST nodes
type Visitor interface {
	// Visit is an optional method that can be implemented to handle all nodes generically
	Visit(node Node) interface{}

	VisitElement(element *Element) interface{}
	VisitTemplate(template *Template) interface{}
	VisitContent(content *Content) interface{}
	VisitVariable(variable *Variable) interface{}
	VisitReference(reference *Reference) interface{}
	VisitTextAttribute(attribute *TextAttribute) interface{}
	VisitBoundAttribute(attribute *BoundAttribute) interface{}
	VisitBoundEvent(event *BoundEvent) interface{}
	VisitText(text *Text) interface{}
	VisitBoundText(text *BoundText) interface{}
	VisitIcu(icu *Icu) interface{}
	VisitDeferredBlock(deferred *DeferredBlock) interface{}
	VisitDeferredBlockPlaceholder(block *DeferredBlockPlaceholder) interface{}
	VisitDeferredBlockError(block *DeferredBlockError) interface{}
	VisitDeferredBlockLoading(block *DeferredBlockLoading) interface{}
	VisitDeferredTrigger(trigger *DeferredTrigger) interface{}
	VisitSwitchBlock(block *SwitchBlock) interface{}
	VisitSwitchBlockCase(block *SwitchBlockCase) interface{}
	VisitForLoopBlock(block *ForLoopBlock) interface{}
	VisitForLoopBlockEmpty(block *ForLoopBlockEmpty) interface{}
	VisitIfBlock(block *IfBlock) interface{}
	VisitIfBlockBranch(block *IfBlockBranch) interface{}
	VisitUnknownBlock(block *UnknownBlock) interface{}
	VisitLetDeclaration(decl *LetDeclaration) interface{}
	VisitComponent(component *Component) interface{}
	VisitDirective(directive *Directive) interface{}
}

// RecursiveVisitor is a visitor that recursively visits all nodes
type RecursiveVisitor struct{}

// NewRecursiveVisitor creates a new RecursiveVisitor
func NewRecursiveVisitor() *RecursiveVisitor {
	return &RecursiveVisitor{}
}

// Visit is the generic visit method that delegates to the node's Visit method
func (rv *RecursiveVisitor) Visit(node Node) interface{} {
	return node.Visit(rv)
}

// VisitElement visits an element
func (rv *RecursiveVisitor) VisitElement(element *Element) interface{} {
	// Convert slices to []Node
	attributes := make([]Node, len(element.Attributes))
	for i, attr := range element.Attributes {
		attributes[i] = attr
	}
	VisitAll(rv, attributes)

	inputs := make([]Node, len(element.Inputs))
	for i, input := range element.Inputs {
		inputs[i] = input
	}
	VisitAll(rv, inputs)

	outputs := make([]Node, len(element.Outputs))
	for i, output := range element.Outputs {
		outputs[i] = output
	}
	VisitAll(rv, outputs)

	directives := make([]Node, len(element.Directives))
	for i, directive := range element.Directives {
		directives[i] = directive
	}
	VisitAll(rv, directives)

	VisitAll(rv, element.Children)

	references := make([]Node, len(element.References))
	for i, ref := range element.References {
		references[i] = ref
	}
	VisitAll(rv, references)
	return nil
}

// VisitTemplate visits a template
func (rv *RecursiveVisitor) VisitTemplate(template *Template) interface{} {
	// Convert slices to []Node
	attributes := make([]Node, len(template.Attributes))
	for i, attr := range template.Attributes {
		attributes[i] = attr
	}
	VisitAll(rv, attributes)

	inputs := make([]Node, len(template.Inputs))
	for i, input := range template.Inputs {
		inputs[i] = input
	}
	VisitAll(rv, inputs)

	outputs := make([]Node, len(template.Outputs))
	for i, output := range template.Outputs {
		outputs[i] = output
	}
	VisitAll(rv, outputs)

	directives := make([]Node, len(template.Directives))
	for i, directive := range template.Directives {
		directives[i] = directive
	}
	VisitAll(rv, directives)

	VisitAll(rv, template.Children)

	references := make([]Node, len(template.References))
	for i, ref := range template.References {
		references[i] = ref
	}
	VisitAll(rv, references)

	variables := make([]Node, len(template.Variables))
	for i, variable := range template.Variables {
		variables[i] = variable
	}
	VisitAll(rv, variables)
	return nil
}

// VisitDeferredBlock visits a deferred block
func (rv *RecursiveVisitor) VisitDeferredBlock(deferred *DeferredBlock) interface{} {
	deferred.VisitAll(rv)
	return nil
}

// VisitDeferredBlockPlaceholder visits a deferred block placeholder
func (rv *RecursiveVisitor) VisitDeferredBlockPlaceholder(block *DeferredBlockPlaceholder) interface{} {
	VisitAll(rv, block.Children)
	return nil
}

// VisitDeferredBlockError visits a deferred block error
func (rv *RecursiveVisitor) VisitDeferredBlockError(block *DeferredBlockError) interface{} {
	VisitAll(rv, block.Children)
	return nil
}

// VisitDeferredBlockLoading visits a deferred block loading
func (rv *RecursiveVisitor) VisitDeferredBlockLoading(block *DeferredBlockLoading) interface{} {
	VisitAll(rv, block.Children)
	return nil
}

// VisitSwitchBlock visits a switch block
func (rv *RecursiveVisitor) VisitSwitchBlock(block *SwitchBlock) interface{} {
	// Convert slice to []Node
	cases := make([]Node, len(block.Cases))
	for i, c := range block.Cases {
		cases[i] = c
	}
	VisitAll(rv, cases)
	return nil
}

// VisitSwitchBlockCase visits a switch block case
func (rv *RecursiveVisitor) VisitSwitchBlockCase(block *SwitchBlockCase) interface{} {
	VisitAll(rv, block.Children)
	return nil
}

// VisitForLoopBlock visits a for loop block
func (rv *RecursiveVisitor) VisitForLoopBlock(block *ForLoopBlock) interface{} {
	blockItems := []Node{block.Item}
	for _, v := range block.ContextVariables {
		blockItems = append(blockItems, v)
	}
	blockItems = append(blockItems, block.Children...)
	if block.Empty != nil {
		blockItems = append(blockItems, block.Empty)
	}
	VisitAll(rv, blockItems)
	return nil
}

// VisitForLoopBlockEmpty visits a for loop block empty
func (rv *RecursiveVisitor) VisitForLoopBlockEmpty(block *ForLoopBlockEmpty) interface{} {
	VisitAll(rv, block.Children)
	return nil
}

// VisitIfBlock visits an if block
func (rv *RecursiveVisitor) VisitIfBlock(block *IfBlock) interface{} {
	// Convert slice to []Node
	branches := make([]Node, len(block.Branches))
	for i, branch := range block.Branches {
		branches[i] = branch
	}
	VisitAll(rv, branches)
	return nil
}

// VisitIfBlockBranch visits an if block branch
func (rv *RecursiveVisitor) VisitIfBlockBranch(block *IfBlockBranch) interface{} {
	blockItems := block.Children
	if block.ExpressionAlias != nil {
		blockItems = append(blockItems, block.ExpressionAlias)
	}
	VisitAll(rv, blockItems)
	return nil
}

// VisitContent visits content
func (rv *RecursiveVisitor) VisitContent(content *Content) interface{} {
	VisitAll(rv, content.Children)
	return nil
}

// VisitComponent visits a component
func (rv *RecursiveVisitor) VisitComponent(component *Component) interface{} {
	// Convert slices to []Node
	attributes := make([]Node, len(component.Attributes))
	for i, attr := range component.Attributes {
		attributes[i] = attr
	}
	VisitAll(rv, attributes)

	inputs := make([]Node, len(component.Inputs))
	for i, input := range component.Inputs {
		inputs[i] = input
	}
	VisitAll(rv, inputs)

	outputs := make([]Node, len(component.Outputs))
	for i, output := range component.Outputs {
		outputs[i] = output
	}
	VisitAll(rv, outputs)

	directives := make([]Node, len(component.Directives))
	for i, directive := range component.Directives {
		directives[i] = directive
	}
	VisitAll(rv, directives)

	VisitAll(rv, component.Children)

	references := make([]Node, len(component.References))
	for i, ref := range component.References {
		references[i] = ref
	}
	VisitAll(rv, references)
	return nil
}

// VisitDirective visits a directive
func (rv *RecursiveVisitor) VisitDirective(directive *Directive) interface{} {
	// Convert slices to []Node
	attributes := make([]Node, len(directive.Attributes))
	for i, attr := range directive.Attributes {
		attributes[i] = attr
	}
	VisitAll(rv, attributes)

	inputs := make([]Node, len(directive.Inputs))
	for i, input := range directive.Inputs {
		inputs[i] = input
	}
	VisitAll(rv, inputs)

	outputs := make([]Node, len(directive.Outputs))
	for i, output := range directive.Outputs {
		outputs[i] = output
	}
	VisitAll(rv, outputs)

	references := make([]Node, len(directive.References))
	for i, ref := range directive.References {
		references[i] = ref
	}
	VisitAll(rv, references)
	return nil
}

// VisitVariable visits a variable (no-op)
func (rv *RecursiveVisitor) VisitVariable(variable *Variable) interface{} {
	return nil
}

// VisitReference visits a reference (no-op)
func (rv *RecursiveVisitor) VisitReference(reference *Reference) interface{} {
	return nil
}

// VisitTextAttribute visits a text attribute (no-op)
func (rv *RecursiveVisitor) VisitTextAttribute(attribute *TextAttribute) interface{} {
	return nil
}

// VisitBoundAttribute visits a bound attribute (no-op)
func (rv *RecursiveVisitor) VisitBoundAttribute(attribute *BoundAttribute) interface{} {
	return nil
}

// VisitBoundEvent visits a bound event (no-op)
func (rv *RecursiveVisitor) VisitBoundEvent(event *BoundEvent) interface{} {
	return nil
}

// VisitText visits text (no-op)
func (rv *RecursiveVisitor) VisitText(text *Text) interface{} {
	return nil
}

// VisitBoundText visits bound text (no-op)
func (rv *RecursiveVisitor) VisitBoundText(text *BoundText) interface{} {
	return nil
}

// VisitIcu visits ICU (no-op)
func (rv *RecursiveVisitor) VisitIcu(icu *Icu) interface{} {
	return nil
}

// VisitDeferredTrigger visits a deferred trigger (no-op)
func (rv *RecursiveVisitor) VisitDeferredTrigger(trigger *DeferredTrigger) interface{} {
	return nil
}

// VisitUnknownBlock visits an unknown block (no-op)
func (rv *RecursiveVisitor) VisitUnknownBlock(block *UnknownBlock) interface{} {
	return nil
}

// VisitLetDeclaration visits a let declaration (no-op)
func (rv *RecursiveVisitor) VisitLetDeclaration(decl *LetDeclaration) interface{} {
	return nil
}

// VisitAll visits all nodes in a slice
func VisitAll(visitor Visitor, nodes []Node) []interface{} {
	result := []interface{}{}

	// Check if visitor has a generic Visit method
	if genericVisitor, ok := visitor.(interface {
		Visit(node Node) interface{}
	}); ok {
		for _, node := range nodes {
			genericVisitor.Visit(node)
		}
	} else {
		for _, node := range nodes {
			newNode := node.Visit(visitor)
			if newNode != nil {
				result = append(result, newNode)
			}
		}
	}
	return result
}
