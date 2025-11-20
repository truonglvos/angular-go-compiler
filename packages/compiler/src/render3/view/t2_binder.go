package view

import (
	"fmt"
	"strings"

	"ngc-go/packages/compiler/src/css"
	"ngc-go/packages/compiler/src/expression_parser"
	"ngc-go/packages/compiler/src/render3"
)

// diff computes a difference between full list (first argument) and
// list of items that should be excluded from the full list (second argument).
func diff(fullList []string, itemsToExclude []string) []string {
	exclude := make(map[string]bool)
	for _, item := range itemsToExclude {
		exclude[item] = true
	}
	result := []string{}
	for _, item := range fullList {
		if !exclude[item] {
			result = append(result, item)
		}
	}
	return result
}

// BindingsMap is a shorthand for a map between a binding AST node and the entity it's targeting.
type BindingsMap map[interface{}]interface{} // BoundAttribute | BoundEvent | TextAttribute -> DirectiveT | Template | Element

// ReferenceMap is a shorthand for a map between a reference AST node and the entity it's targeting.
type ReferenceMap map[*render3.Reference]interface{} // Reference -> Template | Element | {directive: DirectiveT; node: Exclude<DirectiveOwner, HostElement>}

// MatchedDirectives is a mapping between AST nodes and the directives that have been matched on them.
type MatchedDirectives map[DirectiveOwner][]interface{} // DirectiveOwner -> DirectiveT[]

// ScopedNodeEntities is a mapping between a scoped node and the template entities that exist in it.
// nil represents the root scope.
type ScopedNodeEntities map[ScopedNode]map[TemplateEntity]bool // ScopedNode | null -> Set<TemplateEntity>

// DeferBlockScope represents a defer block paired with its corresponding scope.
type DeferBlockScope struct {
	Block *render3.DeferredBlock
	Scope *Scope
}

// DeferBlockScopes is a shorthand tuple type where a defer block is paired with its corresponding scope.
type DeferBlockScopes []DeferBlockScope

// DirectiveMatcher is an object used to match template nodes to directives.
// This can be either a SelectorMatcher or SelectorlessMatcher
type DirectiveMatcher interface{}

// FindMatchingDirectivesAndPipes finds matching directives and pipes in a template.
// Given a template string and a set of available directive selectors,
// computes a list of matching selectors and splits them into 2 buckets:
// (1) eagerly used in a template and (2) directives used only in defer
// blocks. Similarly, returns 2 lists of pipes (eager and deferrable).
//
// Note: deferrable directives selectors and pipes names used in `@defer`
// blocks are **candidates** and API caller should make sure that:
//
//   - A Component where a given template is defined is standalone
//   - Underlying dependency classes are also standalone
//   - Dependency class symbols are not eagerly used in a TS file
//     where a host component (that owns the template) is located
func FindMatchingDirectivesAndPipes(template string, directiveSelectors []string) map[string]interface{} {
	matcher := css.NewSelectorMatcher[DirectiveMeta]()
	for _, selector := range directiveSelectors {
		// Create a fake directive instance to account for the logic inside
		// of the `R3TargetBinder` class (which invokes the `hasBindingPropertyName`
		// function internally).
		fakeDirective := &fakeDirectiveMeta{
			selector: selector,
			exportAs: nil,
			inputs:   &fakeInputOutputPropertySet{},
			outputs:  &fakeInputOutputPropertySet{},
		}
		var directiveMeta DirectiveMeta = fakeDirective
		parsedSelectors, err := css.ParseCssSelector(selector)
		if err == nil {
			matcher.AddSelectables(parsedSelectors, &directiveMeta)
		}
	}
	// Note: parseTemplate is not yet implemented in Go, so this function
	// will need to be completed when parseTemplate is available
	// parsedTemplate := parseTemplate(template, "")
	// binder := NewR3TargetBinder(matcher)
	// bound := binder.Bind(&Target{Template: parsedTemplate.Nodes})

	// For now, return empty result
	return map[string]interface{}{
		"directives": map[string]interface{}{
			"regular":         []string{},
			"deferCandidates": []string{},
		},
		"pipes": map[string]interface{}{
			"regular":         []string{},
			"deferCandidates": []string{},
		},
	}
}

// fakeDirectiveMeta is a fake directive metadata for testing
type fakeDirectiveMeta struct {
	selector string
	exportAs []string
	inputs   InputOutputPropertySet
	outputs  InputOutputPropertySet
}

func (f *fakeDirectiveMeta) Name() string {
	return f.selector
}

func (f *fakeDirectiveMeta) Selector() *string {
	return &f.selector
}

func (f *fakeDirectiveMeta) IsComponent() bool {
	return false
}

func (f *fakeDirectiveMeta) Inputs() InputOutputPropertySet {
	return f.inputs
}

func (f *fakeDirectiveMeta) Outputs() InputOutputPropertySet {
	return f.outputs
}

func (f *fakeDirectiveMeta) ExportAs() []string {
	return f.exportAs
}

func (f *fakeDirectiveMeta) IsStructural() bool {
	return false
}

func (f *fakeDirectiveMeta) NgContentSelectors() []string {
	return nil
}

func (f *fakeDirectiveMeta) PreserveWhitespaces() bool {
	return false
}

func (f *fakeDirectiveMeta) AnimationTriggerNames() *LegacyAnimationTriggerNames {
	return nil
}

// fakeInputOutputPropertySet is a fake input/output property set
type fakeInputOutputPropertySet struct{}

func (f *fakeInputOutputPropertySet) HasBindingPropertyName(propertyName string) bool {
	return false
}

// R3TargetBinder processes `Target`s with a given set of directives and performs a binding operation, which
// returns an object similar to TypeScript's `ts.TypeChecker` that contains knowledge about the
// target.
type R3TargetBinder struct {
	directiveMatcher DirectiveMatcher
}

// NewR3TargetBinder creates a new R3TargetBinder
func NewR3TargetBinder(directiveMatcher DirectiveMatcher) *R3TargetBinder {
	return &R3TargetBinder{
		directiveMatcher: directiveMatcher,
	}
}

// Bind performs a binding operation on the given `Target` and return a `BoundTarget` which contains
// metadata about the types referenced in the template.
func (b *R3TargetBinder) Bind(target *Target) BoundTarget {
	if target.Template == nil && target.Host == nil {
		panic("Empty bound targets are not supported")
	}

	directives := make(MatchedDirectives)
	eagerDirectives := []interface{}{}
	missingDirectives := make(map[string]bool)
	bindings := make(BindingsMap)
	references := make(ReferenceMap)
	scopedNodeEntities := make(ScopedNodeEntities)
	expressions := make(map[expression_parser.AST]TemplateEntity)
	symbols := make(map[TemplateEntity]*render3.Template)
	nestingLevel := make(map[ScopedNode]int)
	usedPipes := make(map[string]bool)
	eagerPipes := make(map[string]bool)
	deferBlocks := []DeferBlockScope{}

	if target.Template != nil {
		// First, parse the template into a `Scope` structure. This operation captures the syntactic
		// scopes in the template and makes them available for later use.
		scope := NewScope().Apply(target.Template)

		// Use the `Scope` to extract the entities present at every level of the template.
		extractScopedNodeEntities(scope, scopedNodeEntities)

		// Next, perform directive matching on the template using the `DirectiveBinder`. This returns:
		//   - directives: Map of nodes (elements & ng-templates) to the directives on them.
		//   - bindings: Map of inputs, outputs, and attributes to the directive/element that claims
		//     them. TODO(alxhub): handle multiple directives claiming an input/output/etc.
		//   - references: Map of #references to their targets.
		DirectiveBinderApply(
			target.Template,
			b.directiveMatcher,
			directives,
			eagerDirectives,
			missingDirectives,
			bindings,
			references,
		)

		// Finally, run the TemplateBinder to bind references, variables, and other entities within the
		// template. This extracts all the metadata that doesn't depend on directive matching.
		TemplateBinderApplyWithScope(
			target.Template,
			scope,
			expressions,
			symbols,
			nestingLevel,
			usedPipes,
			eagerPipes,
			deferBlocks,
		)
	}

	// Bind the host element in a separate scope. Note that it only uses the
	// `TemplateBinder` since directives don't apply inside a host context.
	if target.Host != nil {
		directives[target.Host.Node] = target.Host.Directives
		TemplateBinderApplyWithScope(
			target.Host.Node,
			NewScope().Apply(target.Host.Node),
			expressions,
			symbols,
			nestingLevel,
			usedPipes,
			eagerPipes,
			deferBlocks,
		)
	}

	return NewR3BoundTarget(
		target,
		directives,
		eagerDirectives,
		missingDirectives,
		bindings,
		references,
		expressions,
		symbols,
		nestingLevel,
		scopedNodeEntities,
		usedPipes,
		eagerPipes,
		deferBlocks,
	)
}

// Scope represents a binding scope within a template.
//
// Any variables, references, or other named entities declared within the template will
// be captured and available by name in `namedEntities`. Additionally, child templates will
// be analyzed and have their child `Scope`s available in `childScopes`.
type Scope struct {
	// NamedEntities are named members of the `Scope`, such as `Reference`s or `Variable`s.
	NamedEntities map[string]TemplateEntity

	// ElementLikeInScope is a set of element-like nodes that belong to this scope.
	ElementLikeInScope map[interface{}]bool // Element | Component

	// ChildScopes are child `Scope`s for immediately nested `ScopedNode`s.
	ChildScopes map[ScopedNode]*Scope

	// IsDeferred indicates whether this scope is deferred or if any of its ancestors are deferred.
	IsDeferred bool

	parentScope *Scope
	rootNode    ScopedNode
}

// NewScope creates a new root scope
func NewScope() *Scope {
	return &Scope{
		NamedEntities:      make(map[string]TemplateEntity),
		ElementLikeInScope: make(map[interface{}]bool),
		ChildScopes:        make(map[ScopedNode]*Scope),
		IsDeferred:         false,
		parentScope:        nil,
		rootNode:           nil,
	}
}

// newScopeWithParent creates a new scope with a parent
func newScopeWithParent(parentScope *Scope, rootNode ScopedNode) *Scope {
	isDeferred := false
	if parentScope != nil && parentScope.IsDeferred {
		isDeferred = true
	} else {
		// Check if rootNode is a DeferredBlock
		if _, ok := rootNode.(*render3.DeferredBlock); ok {
			isDeferred = true
		}
	}
	return &Scope{
		NamedEntities:      make(map[string]TemplateEntity),
		ElementLikeInScope: make(map[interface{}]bool),
		ChildScopes:        make(map[ScopedNode]*Scope),
		IsDeferred:         isDeferred,
		parentScope:        parentScope,
		rootNode:           rootNode,
	}
}

// Apply processes a template (either as a `Template` sub-template with variables, or a plain array of
// template `Node`s) and construct its `Scope`.
func (s *Scope) Apply(template interface{}) *Scope {
	scope := NewScope()
	scope.ingest(template)
	return scope
}

// ingest is an internal method to process the scoped node and populate the `Scope`.
func (s *Scope) ingest(nodeOrNodes interface{}) {
	switch n := nodeOrNodes.(type) {
	case *render3.Template:
		// Variables on an <ng-template> are defined in the inner scope.
		for _, variable := range n.Variables {
			s.VisitVariable(variable)
		}
		// Process the nodes of the template.
		for _, node := range n.Children {
			node.Visit(s)
		}
	case *render3.IfBlockBranch:
		if n.ExpressionAlias != nil {
			s.VisitVariable(n.ExpressionAlias)
		}
		for _, node := range n.Children {
			node.Visit(s)
		}
	case *render3.ForLoopBlock:
		s.VisitVariable(n.Item)
		for _, v := range n.ContextVariables {
			s.VisitVariable(v)
		}
		for _, node := range n.Children {
			node.Visit(s)
		}
	case *render3.SwitchBlockCase,
		*render3.ForLoopBlockEmpty,
		*render3.DeferredBlock,
		*render3.DeferredBlockError,
		*render3.DeferredBlockPlaceholder,
		*render3.DeferredBlockLoading,
		*render3.Content:
		var children []render3.Node
		switch n := nodeOrNodes.(type) {
		case *render3.SwitchBlockCase:
			children = n.Children
		case *render3.ForLoopBlockEmpty:
			children = n.Children
		case *render3.DeferredBlock:
			children = n.Children
		case *render3.DeferredBlockError:
			children = n.Children
		case *render3.DeferredBlockPlaceholder:
			children = n.Children
		case *render3.DeferredBlockLoading:
			children = n.Children
		case *render3.Content:
			children = n.Children
		}
		for _, node := range children {
			node.Visit(s)
		}
	case []render3.Node:
		// No overarching `Template` instance, so process the nodes directly.
		for _, node := range n {
			node.Visit(s)
		}
	}
	// HostElement is skipped
}

// Visit implements the Visitor interface
func (s *Scope) Visit(node render3.Node) interface{} {
	return node.Visit(s)
}

// VisitElement visits an Element node
func (s *Scope) VisitElement(element *render3.Element) interface{} {
	s.visitElementLike(element)
	return nil
}

// VisitTemplate visits a Template node
func (s *Scope) VisitTemplate(template *render3.Template) interface{} {
	for _, directive := range template.Directives {
		directive.Visit(s)
	}

	// References on a <ng-template> are defined in the outer scope, so capture them before
	// processing the template's child scope.
	for _, ref := range template.References {
		s.VisitReference(ref)
	}

	// Next, create an inner scope and process the template within it.
	s.ingestScopedNode(template)
	return nil
}

// VisitVariable visits a Variable node
func (s *Scope) VisitVariable(variable *render3.Variable) interface{} {
	// Declare the variable if it's not already.
	s.maybeDeclare(variable)
	return nil
}

// VisitReference visits a Reference node
func (s *Scope) VisitReference(reference *render3.Reference) interface{} {
	// Declare the variable if it's not already.
	s.maybeDeclare(reference)
	return nil
}

// VisitDeferredBlock visits a DeferredBlock node
func (s *Scope) VisitDeferredBlock(deferred *render3.DeferredBlock) interface{} {
	s.ingestScopedNode(deferred)
	if deferred.Placeholder != nil {
		deferred.Placeholder.Visit(s)
	}
	if deferred.Loading != nil {
		deferred.Loading.Visit(s)
	}
	if deferred.Error != nil {
		deferred.Error.Visit(s)
	}
	return nil
}

// VisitDeferredBlockPlaceholder visits a DeferredBlockPlaceholder node
func (s *Scope) VisitDeferredBlockPlaceholder(block *render3.DeferredBlockPlaceholder) interface{} {
	s.ingestScopedNode(block)
	return nil
}

// VisitDeferredBlockError visits a DeferredBlockError node
func (s *Scope) VisitDeferredBlockError(block *render3.DeferredBlockError) interface{} {
	s.ingestScopedNode(block)
	return nil
}

// VisitDeferredBlockLoading visits a DeferredBlockLoading node
func (s *Scope) VisitDeferredBlockLoading(block *render3.DeferredBlockLoading) interface{} {
	s.ingestScopedNode(block)
	return nil
}

// VisitSwitchBlock visits a SwitchBlock node
func (s *Scope) VisitSwitchBlock(block *render3.SwitchBlock) interface{} {
	for _, caseNode := range block.Cases {
		caseNode.Visit(s)
	}
	return nil
}

// VisitSwitchBlockCase visits a SwitchBlockCase node
func (s *Scope) VisitSwitchBlockCase(block *render3.SwitchBlockCase) interface{} {
	s.ingestScopedNode(block)
	return nil
}

// VisitForLoopBlock visits a ForLoopBlock node
func (s *Scope) VisitForLoopBlock(block *render3.ForLoopBlock) interface{} {
	s.ingestScopedNode(block)
	if block.Empty != nil {
		block.Empty.Visit(s)
	}
	return nil
}

// VisitForLoopBlockEmpty visits a ForLoopBlockEmpty node
func (s *Scope) VisitForLoopBlockEmpty(block *render3.ForLoopBlockEmpty) interface{} {
	s.ingestScopedNode(block)
	return nil
}

// VisitIfBlock visits an IfBlock node
func (s *Scope) VisitIfBlock(block *render3.IfBlock) interface{} {
	for _, branch := range block.Branches {
		branch.Visit(s)
	}
	return nil
}

// VisitIfBlockBranch visits an IfBlockBranch node
func (s *Scope) VisitIfBlockBranch(block *render3.IfBlockBranch) interface{} {
	s.ingestScopedNode(block)
	return nil
}

// VisitContent visits a Content node
func (s *Scope) VisitContent(content *render3.Content) interface{} {
	s.ingestScopedNode(content)
	return nil
}

// VisitLetDeclaration visits a LetDeclaration node
func (s *Scope) VisitLetDeclaration(decl *render3.LetDeclaration) interface{} {
	s.maybeDeclare(decl)
	return nil
}

// VisitComponent visits a Component node
func (s *Scope) VisitComponent(component *render3.Component) interface{} {
	s.visitElementLike(component)
	return nil
}

// VisitDirective visits a Directive node
func (s *Scope) VisitDirective(directive *render3.Directive) interface{} {
	for _, ref := range directive.References {
		s.VisitReference(ref)
	}
	return nil
}

// Unused visitors
func (s *Scope) VisitBoundAttribute(attr *render3.BoundAttribute) interface{} { return nil }
func (s *Scope) VisitBoundEvent(event *render3.BoundEvent) interface{}        { return nil }
func (s *Scope) VisitBoundText(text *render3.BoundText) interface{}           { return nil }
func (s *Scope) VisitText(text *render3.Text) interface{}                     { return nil }
func (s *Scope) VisitTextAttribute(attr *render3.TextAttribute) interface{}   { return nil }
func (s *Scope) VisitIcu(icu *render3.Icu) interface{}                        { return nil }
func (s *Scope) VisitDeferredTrigger(trigger *render3.DeferredTrigger) interface{} {
	return nil
}
func (s *Scope) VisitUnknownBlock(block *render3.UnknownBlock) interface{} { return nil }

// visitElementLike visits an Element or Component node
func (s *Scope) visitElementLike(node interface{}) {
	var directives []*render3.Directive
	var references []*render3.Reference
	var children []render3.Node

	switch n := node.(type) {
	case *render3.Element:
		directives = n.Directives
		references = n.References
		children = n.Children
	case *render3.Component:
		directives = n.Directives
		references = n.References
		children = n.Children
	}

	for _, directive := range directives {
		directive.Visit(s)
	}
	for _, ref := range references {
		s.VisitReference(ref)
	}
	for _, child := range children {
		child.Visit(s)
	}
	s.ElementLikeInScope[node] = true
}

// maybeDeclare declares something with a name, as long as that name isn't taken.
func (s *Scope) maybeDeclare(thing TemplateEntity) {
	var name string
	switch t := thing.(type) {
	case *render3.Reference:
		name = t.Name
	case *render3.Variable:
		name = t.Name
	case *render3.LetDeclaration:
		name = t.Name
	default:
		return
	}

	if _, exists := s.NamedEntities[name]; !exists {
		s.NamedEntities[name] = thing
	}
}

// Lookup looks up a variable within this `Scope`.
//
// This can recurse into a parent `Scope` if it's available.
func (s *Scope) Lookup(name string) TemplateEntity {
	if entity, exists := s.NamedEntities[name]; exists {
		// Found in the local scope.
		return entity
	} else if s.parentScope != nil {
		// Not in the local scope, but there's a parent scope so check there.
		return s.parentScope.Lookup(name)
	} else {
		// At the top level and it wasn't found.
		return nil
	}
}

// GetChildScope gets the child scope for a `ScopedNode`.
//
// This should always be defined.
func (s *Scope) GetChildScope(node ScopedNode) *Scope {
	if res, exists := s.ChildScopes[node]; exists {
		return res
	}
	panic(fmt.Sprintf("Assertion error: child scope for %v not found", node))
}

// ingestScopedNode processes a scoped node and creates a child scope
func (s *Scope) ingestScopedNode(node ScopedNode) {
	scope := newScopeWithParent(s, node)
	scope.ingest(node)
	s.ChildScopes[node] = scope
}

// DirectiveBinder processes a template and matches directives on nodes (elements and templates).
//
// Usually used via the static `Apply` function.
type DirectiveBinder struct {
	directiveMatcher  DirectiveMatcher
	directives        MatchedDirectives
	eagerDirectives   []interface{}
	missingDirectives map[string]bool
	bindings          BindingsMap
	references        ReferenceMap
	isInDeferBlock    bool
}

// DirectiveBinderApply processes a template (list of `Node`s) and perform directive matching against each node.
//
// template: the list of template `Node`s to match (recursively).
// directiveMatcher: a `SelectorMatcher` containing the directives that are in scope for this template.
// Returns three maps which contain information about directives in the template: the
// `directives` map which lists directives matched on each node, the `bindings` map which
// indicates which directives claimed which bindings (inputs, outputs, etc), and the `references`
// map which resolves #references (`Reference`s) within the template to the named directive or
// template node.
func DirectiveBinderApply(
	template []render3.Node,
	directiveMatcher DirectiveMatcher,
	directives MatchedDirectives,
	eagerDirectives []interface{},
	missingDirectives map[string]bool,
	bindings BindingsMap,
	references ReferenceMap,
) {
	matcher := &DirectiveBinder{
		directiveMatcher:  directiveMatcher,
		directives:        directives,
		eagerDirectives:   eagerDirectives,
		missingDirectives: missingDirectives,
		bindings:          bindings,
		references:        references,
		isInDeferBlock:    false,
	}
	matcher.ingest(template)
}

func (db *DirectiveBinder) ingest(template []render3.Node) {
	for _, node := range template {
		node.Visit(db)
	}
}

// Visit implements Visitor interface
func (db *DirectiveBinder) Visit(node render3.Node) interface{} {
	return node.Visit(db)
}

// VisitElement visits an Element node
func (db *DirectiveBinder) VisitElement(element *render3.Element) interface{} {
	db.visitElementOrTemplate(element)
	return nil
}

// VisitTemplate visits a Template node
func (db *DirectiveBinder) VisitTemplate(template *render3.Template) interface{} {
	db.visitElementOrTemplate(template)
	return nil
}

// VisitDeferredBlock visits a DeferredBlock node
func (db *DirectiveBinder) VisitDeferredBlock(deferred *render3.DeferredBlock) interface{} {
	wasInDeferBlock := db.isInDeferBlock
	db.isInDeferBlock = true
	for _, child := range deferred.Children {
		child.Visit(db)
	}
	db.isInDeferBlock = wasInDeferBlock

	if deferred.Placeholder != nil {
		deferred.Placeholder.Visit(db)
	}
	if deferred.Loading != nil {
		deferred.Loading.Visit(db)
	}
	if deferred.Error != nil {
		deferred.Error.Visit(db)
	}
	return nil
}

// VisitDeferredBlockPlaceholder visits a DeferredBlockPlaceholder node
func (db *DirectiveBinder) VisitDeferredBlockPlaceholder(block *render3.DeferredBlockPlaceholder) interface{} {
	for _, child := range block.Children {
		child.Visit(db)
	}
	return nil
}

// VisitDeferredBlockError visits a DeferredBlockError node
func (db *DirectiveBinder) VisitDeferredBlockError(block *render3.DeferredBlockError) interface{} {
	for _, child := range block.Children {
		child.Visit(db)
	}
	return nil
}

// VisitDeferredBlockLoading visits a DeferredBlockLoading node
func (db *DirectiveBinder) VisitDeferredBlockLoading(block *render3.DeferredBlockLoading) interface{} {
	for _, child := range block.Children {
		child.Visit(db)
	}
	return nil
}

// VisitSwitchBlock visits a SwitchBlock node
func (db *DirectiveBinder) VisitSwitchBlock(block *render3.SwitchBlock) interface{} {
	for _, caseNode := range block.Cases {
		caseNode.Visit(db)
	}
	return nil
}

// VisitSwitchBlockCase visits a SwitchBlockCase node
func (db *DirectiveBinder) VisitSwitchBlockCase(block *render3.SwitchBlockCase) interface{} {
	for _, node := range block.Children {
		node.Visit(db)
	}
	return nil
}

// VisitForLoopBlock visits a ForLoopBlock node
func (db *DirectiveBinder) VisitForLoopBlock(block *render3.ForLoopBlock) interface{} {
	block.Item.Visit(db)
	for _, v := range block.ContextVariables {
		v.Visit(db)
	}
	for _, node := range block.Children {
		node.Visit(db)
	}
	if block.Empty != nil {
		block.Empty.Visit(db)
	}
	return nil
}

// VisitForLoopBlockEmpty visits a ForLoopBlockEmpty node
func (db *DirectiveBinder) VisitForLoopBlockEmpty(block *render3.ForLoopBlockEmpty) interface{} {
	for _, node := range block.Children {
		node.Visit(db)
	}
	return nil
}

// VisitIfBlock visits an IfBlock node
func (db *DirectiveBinder) VisitIfBlock(block *render3.IfBlock) interface{} {
	for _, branch := range block.Branches {
		branch.Visit(db)
	}
	return nil
}

// VisitIfBlockBranch visits an IfBlockBranch node
func (db *DirectiveBinder) VisitIfBlockBranch(block *render3.IfBlockBranch) interface{} {
	if block.ExpressionAlias != nil {
		block.ExpressionAlias.Visit(db)
	}
	for _, node := range block.Children {
		node.Visit(db)
	}
	return nil
}

// VisitContent visits a Content node
func (db *DirectiveBinder) VisitContent(content *render3.Content) interface{} {
	for _, child := range content.Children {
		child.Visit(db)
	}
	return nil
}

// VisitComponent visits a Component node
func (db *DirectiveBinder) VisitComponent(node *render3.Component) interface{} {
	if selectorlessMatcher, ok := db.directiveMatcher.(*css.SelectorlessMatcher[DirectiveMeta]); ok {
		componentMatches := selectorlessMatcher.Match(node.ComponentName)
		directives := make([]DirectiveMeta, len(componentMatches))
		for i, m := range componentMatches {
			directives[i] = m
		}
		if len(directives) > 0 {
			db.trackSelectorlessMatchesAndDirectives(node, directives)
		} else {
			db.missingDirectives[node.ComponentName] = true
		}
	}

	for _, directive := range node.Directives {
		directive.Visit(db)
	}
	for _, child := range node.Children {
		child.Visit(db)
	}
	return nil
}

// VisitDirective visits a Directive node
func (db *DirectiveBinder) VisitDirective(node *render3.Directive) interface{} {
	if selectorlessMatcher, ok := db.directiveMatcher.(*css.SelectorlessMatcher[DirectiveMeta]); ok {
		directiveMatches := selectorlessMatcher.Match(node.Name)
		directives := make([]DirectiveMeta, len(directiveMatches))
		for i, m := range directiveMatches {
			directives[i] = m
		}
		if len(directives) > 0 {
			db.trackSelectorlessMatchesAndDirectives(node, directives)
		} else {
			db.missingDirectives[node.Name] = true
		}
	}
	return nil
}

// visitElementOrTemplate visits an Element or Template node
func (db *DirectiveBinder) visitElementOrTemplate(node interface{}) {
	if selectorMatcher, ok := db.directiveMatcher.(*css.SelectorMatcher[DirectiveMeta]); ok {
		directives := []DirectiveMeta{}
		cssSelector := CreateCssSelectorFromNode(node.(render3.Node))
		selectorMatcher.Match(cssSelector, func(c *css.CssSelector, a *DirectiveMeta) {
			directives = append(directives, *a)
		})
		db.trackSelectorBasedBindingsAndDirectives(node, directives)
	} else {
		// Handle references for non-selector matcher
		var references []*render3.Reference
		switch n := node.(type) {
		case *render3.Element:
			references = n.References
		case *render3.Template:
			references = n.References
		}
		for _, ref := range references {
			if strings.TrimSpace(ref.Value) == "" {
				db.references[ref] = node
			}
		}
	}

	var directives []*render3.Directive
	var children []render3.Node
	switch n := node.(type) {
	case *render3.Element:
		directives = n.Directives
		children = n.Children
	case *render3.Template:
		directives = n.Directives
		children = n.Children
	}

	for _, directive := range directives {
		directive.Visit(db)
	}
	for _, child := range children {
		child.Visit(db)
	}
}

// trackMatchedDirectives tracks matched directives
func (db *DirectiveBinder) trackMatchedDirectives(node DirectiveOwner, directives []DirectiveMeta) {
	if len(directives) > 0 {
		// Convert []DirectiveMeta to []interface{}
		dirs := make([]interface{}, len(directives))
		for i, d := range directives {
			dirs[i] = d
		}
		db.directives[node] = dirs
		if !db.isInDeferBlock {
			db.eagerDirectives = append(db.eagerDirectives, dirs...)
		}
	}
}

// trackSelectorlessMatchesAndDirectives tracks selectorless matches and directives
func (db *DirectiveBinder) trackSelectorlessMatchesAndDirectives(
	node interface{},
	directives []DirectiveMeta,
) {
	if len(directives) == 0 {
		return
	}

	var owner DirectiveOwner
	switch n := node.(type) {
	case *render3.Component:
		owner = n
	case *render3.Directive:
		owner = n
	default:
		return
	}

	db.trackMatchedDirectives(owner, directives)

	setBinding := func(meta DirectiveMeta, attribute interface{}, ioType string) {
		var name string
		switch a := attribute.(type) {
		case *render3.BoundAttribute:
			name = a.Name
		case *render3.BoundEvent:
			name = a.Name
		case *render3.TextAttribute:
			name = a.Name
		default:
			return
		}

		var hasBinding bool
		if ioType == "inputs" {
			hasBinding = meta.Inputs().HasBindingPropertyName(name)
		} else if ioType == "outputs" {
			hasBinding = meta.Outputs().HasBindingPropertyName(name)
		}

		if hasBinding {
			db.bindings[attribute] = meta
		}
	}

	for _, directive := range directives {
		switch n := node.(type) {
		case *render3.Component:
			for _, input := range n.Inputs {
				setBinding(directive, input, "inputs")
			}
			for _, attr := range n.Attributes {
				setBinding(directive, attr, "inputs")
			}
			for _, output := range n.Outputs {
				setBinding(directive, output, "outputs")
			}
		case *render3.Directive:
			// Directives don't have inputs/outputs directly
		}
	}

	// TODO(crisbeto): currently it's unclear how references should behave under selectorless,
	// given that there's one named class which can bring in multiple host directives.
	// For the time being only register the first directive as the reference target.
	var references []*render3.Reference
	switch n := node.(type) {
	case *render3.Component:
		references = n.References
	case *render3.Directive:
		references = n.References
	}
	for _, ref := range references {
		if len(directives) > 0 {
			db.references[ref] = &ReferenceTargetWithDirective{
				Directive: directives[0],
				Node:      owner,
			}
		}
	}
}

// trackSelectorBasedBindingsAndDirectives tracks selector-based bindings and directives
func (db *DirectiveBinder) trackSelectorBasedBindingsAndDirectives(
	node interface{},
	directives []DirectiveMeta,
) {
	var owner DirectiveOwner
	switch n := node.(type) {
	case *render3.Element:
		owner = n
	case *render3.Template:
		owner = n
	default:
		return
	}

	db.trackMatchedDirectives(owner, directives)

	// Resolve any references that are created on this node.
	var references []*render3.Reference
	switch n := node.(type) {
	case *render3.Element:
		references = n.References
	case *render3.Template:
		references = n.References
	}

	for _, ref := range references {
		var dirTarget DirectiveMeta

		// If the reference expression is empty, then it matches the "primary" directive on the node
		// (if there is one). Otherwise it matches the host node itself (either an element or
		// <ng-template> node).
		if strings.TrimSpace(ref.Value) == "" {
			// This could be a reference to a component if there is one.
			for _, dir := range directives {
				if dir.IsComponent() {
					dirTarget = dir
					break
				}
			}
		} else {
			// This should be a reference to a directive exported via exportAs.
			for _, dir := range directives {
				exportAs := dir.ExportAs()
				if exportAs != nil {
					for _, value := range exportAs {
						if value == ref.Value {
							dirTarget = dir
							break
						}
					}
					if dirTarget != nil {
						break
					}
				}
			}
			// Check if a matching directive was found.
			if dirTarget == nil {
				// No matching directive was found - this reference points to an unknown target. Leave it
				// unmapped.
				continue
			}
		}

		if dirTarget != nil {
			// This reference points to a directive.
			db.references[ref] = &ReferenceTargetWithDirective{
				Directive: dirTarget,
				Node:      owner,
			}
		} else {
			// This reference points to the node itself.
			db.references[ref] = node
		}
	}

	// Associate attributes/bindings on the node with directives or with the node itself.
	setAttributeBinding := func(attribute interface{}, ioType string) {
		var name string
		switch a := attribute.(type) {
		case *render3.BoundAttribute:
			name = a.Name
		case *render3.BoundEvent:
			name = a.Name
		case *render3.TextAttribute:
			name = a.Name
		default:
			return
		}

		var dir DirectiveMeta
		for _, d := range directives {
			var hasBinding bool
			if ioType == "inputs" {
				hasBinding = d.Inputs().HasBindingPropertyName(name)
			} else if ioType == "outputs" {
				hasBinding = d.Outputs().HasBindingPropertyName(name)
			}
			if hasBinding {
				dir = d
				break
			}
		}

		binding := node
		if dir != nil {
			binding = dir
		}
		db.bindings[attribute] = binding
	}

	switch n := node.(type) {
	case *render3.Element:
		for _, input := range n.Inputs {
			setAttributeBinding(input, "inputs")
		}
		for _, attr := range n.Attributes {
			setAttributeBinding(attr, "inputs")
		}
		for _, output := range n.Outputs {
			setAttributeBinding(output, "outputs")
		}
	case *render3.Template:
		for _, input := range n.Inputs {
			setAttributeBinding(input, "inputs")
		}
		for _, attr := range n.Attributes {
			setAttributeBinding(attr, "inputs")
		}
		for _, attr := range n.TemplateAttrs {
			setAttributeBinding(attr, "inputs")
		}
		for _, output := range n.Outputs {
			setAttributeBinding(output, "outputs")
		}
	}
}

// Unused visitors
func (db *DirectiveBinder) VisitVariable(variable *render3.Variable) interface{}    { return nil }
func (db *DirectiveBinder) VisitReference(reference *render3.Reference) interface{} { return nil }
func (db *DirectiveBinder) VisitTextAttribute(attribute *render3.TextAttribute) interface{} {
	return nil
}
func (db *DirectiveBinder) VisitBoundAttribute(attribute *render3.BoundAttribute) interface{} {
	return nil
}
func (db *DirectiveBinder) VisitBoundEvent(attribute *render3.BoundEvent) interface{} {
	return nil
}
func (db *DirectiveBinder) VisitText(text *render3.Text) interface{}           { return nil }
func (db *DirectiveBinder) VisitBoundText(text *render3.BoundText) interface{} { return nil }
func (db *DirectiveBinder) VisitIcu(icu *render3.Icu) interface{}              { return nil }
func (db *DirectiveBinder) VisitDeferredTrigger(trigger *render3.DeferredTrigger) interface{} {
	return nil
}
func (db *DirectiveBinder) VisitUnknownBlock(block *render3.UnknownBlock) interface{} {
	return nil
}
func (db *DirectiveBinder) VisitLetDeclaration(decl *render3.LetDeclaration) interface{} {
	return nil
}

// TemplateBinder processes a template and extract metadata about expressions and symbols within.
//
// This is a companion to the `DirectiveBinder` that doesn't require knowledge of directives matched
// within the template in order to operate.
//
// Expressions are visited by the superclass `RecursiveAstVisitor`, with custom logic provided
// by overridden methods from that visitor.
type TemplateBinder struct {
	expression_parser.RecursiveAstVisitor
	bindings      map[expression_parser.AST]TemplateEntity
	symbols       map[TemplateEntity]*render3.Template
	usedPipes     map[string]bool
	eagerPipes    map[string]bool
	deferBlocks   *[]DeferBlockScope
	nestingLevel  map[ScopedNode]int
	scope         *Scope
	rootNode      ScopedNode
	level         int
	visitNodeFunc func(render3.Node) interface{}
}

// TemplateBinderApplyWithScope processes a template and extract metadata about expressions and symbols within.
//
// nodeOrNodes: the nodes of the template to process
// scope: the `Scope` of the template being processed.
// Returns three maps which contain metadata about the template: `expressions` which interprets
// special `AST` nodes in expressions as pointing to references or variables declared within the
// template, `symbols` which maps those variables and references to the nested `Template` which
// declares them, if any, and `nestingLevel` which associates each `Template` with a integer
// nesting level (how many levels deep within the template structure the `Template` is), starting
// at 1.
func TemplateBinderApplyWithScope(
	nodeOrNodes interface{},
	scope *Scope,
	expressions map[expression_parser.AST]TemplateEntity,
	symbols map[TemplateEntity]*render3.Template,
	nestingLevel map[ScopedNode]int,
	usedPipes map[string]bool,
	eagerPipes map[string]bool,
	deferBlocks []DeferBlockScope,
) {
	var template *render3.Template
	if t, ok := nodeOrNodes.(*render3.Template); ok {
		template = t
	}
	// The top-level template has nesting level 0.
	binder := &TemplateBinder{
		bindings:      expressions,
		symbols:       symbols,
		usedPipes:     usedPipes,
		eagerPipes:    eagerPipes,
		deferBlocks:   &deferBlocks,
		nestingLevel:  nestingLevel,
		scope:         scope,
		rootNode:      template,
		level:         0,
		visitNodeFunc: nil,
	}
	binder.visitNodeFunc = func(node render3.Node) interface{} {
		return binder.VisitNode(node)
	}
	binder.ingest(nodeOrNodes)
}

func (tb *TemplateBinder) ingest(nodeOrNodes interface{}) {
	switch n := nodeOrNodes.(type) {
	case *render3.Template:
		// For <ng-template>s, process only variables and child nodes. Inputs, outputs, templateAttrs,
		// and references were all processed in the scope of the containing template.
		for _, variable := range n.Variables {
			tb.visitNodeFunc(variable)
		}
		for _, node := range n.Children {
			tb.visitNodeFunc(node)
		}
		// Set the nesting level.
		tb.nestingLevel[n] = tb.level
	case *render3.IfBlockBranch:
		if n.ExpressionAlias != nil {
			tb.visitNodeFunc(n.ExpressionAlias)
		}
		for _, node := range n.Children {
			tb.visitNodeFunc(node)
		}
		tb.nestingLevel[n] = tb.level
	case *render3.ForLoopBlock:
		tb.visitNodeFunc(n.Item)
		for _, v := range n.ContextVariables {
			tb.visitNodeFunc(v)
		}
		if n.TrackBy != nil && n.TrackBy.AST != nil {
			n.TrackBy.AST.Visit(&tb.RecursiveAstVisitor, nil)
		}
		if n.Expression != nil && n.Expression.AST != nil {
			n.Expression.AST.Visit(&tb.RecursiveAstVisitor, nil)
		}
		for _, node := range n.Children {
			tb.visitNodeFunc(node)
		}
		tb.nestingLevel[n] = tb.level
	case *render3.DeferredBlock:
		if tb.scope.rootNode != n {
			panic(fmt.Sprintf("Assertion error: resolved incorrect scope for deferred block %v", n))
		}
		*tb.deferBlocks = append(*tb.deferBlocks, DeferBlockScope{Block: n, Scope: tb.scope})
		for _, node := range n.Children {
			tb.visitNodeFunc(node)
		}
		tb.nestingLevel[n] = tb.level
	case *render3.SwitchBlockCase,
		*render3.ForLoopBlockEmpty,
		*render3.DeferredBlockError,
		*render3.DeferredBlockPlaceholder,
		*render3.DeferredBlockLoading,
		*render3.Content:
		var children []render3.Node
		switch n := nodeOrNodes.(type) {
		case *render3.SwitchBlockCase:
			children = n.Children
			tb.nestingLevel[n] = tb.level
		case *render3.ForLoopBlockEmpty:
			children = n.Children
			tb.nestingLevel[n] = tb.level
		case *render3.DeferredBlockError:
			children = n.Children
			tb.nestingLevel[n] = tb.level
		case *render3.DeferredBlockPlaceholder:
			children = n.Children
			tb.nestingLevel[n] = tb.level
		case *render3.DeferredBlockLoading:
			children = n.Children
			tb.nestingLevel[n] = tb.level
		case *render3.Content:
			children = n.Children
			tb.nestingLevel[n] = tb.level
		}
		for _, node := range children {
			tb.visitNodeFunc(node)
		}
	case *render3.HostElement:
		// Host elements are always at the top level.
		tb.nestingLevel[n] = 0
	case []render3.Node:
		// Visit each node from the top-level template.
		for _, node := range n {
			tb.visitNodeFunc(node)
		}
	}
}

// VisitAST wraps RecursiveAstVisitor.Visit for AST expressions
func (tb *TemplateBinder) VisitAST(ast expression_parser.AST, context interface{}) interface{} {
	return ast.Visit(&tb.RecursiveAstVisitor, context)
}

// Visit implements render3.Visitor interface for template nodes
func (tb *TemplateBinder) Visit(node render3.Node) interface{} {
	return node.Visit(tb)
}

// VisitNode is an alias for Visit for clarity
func (tb *TemplateBinder) VisitNode(node render3.Node) interface{} {
	return tb.Visit(node)
}

// VisitTemplate visits a Template node
func (tb *TemplateBinder) VisitTemplate(template *render3.Template) interface{} {
	// First, visit inputs, outputs and template attributes of the template node.
	for _, input := range template.Inputs {
		tb.visitNodeFunc(input)
	}
	for _, output := range template.Outputs {
		tb.visitNodeFunc(output)
	}
	for _, directive := range template.Directives {
		tb.visitNodeFunc(directive)
	}
	for _, attr := range template.TemplateAttrs {
		tb.visitNodeFunc(attr.(render3.Node))
	}
	for _, ref := range template.References {
		tb.visitNodeFunc(ref)
	}

	// Next, recurse into the template.
	tb.ingestScopedNode(template)
	return nil
}

// VisitVariable visits a Variable node
func (tb *TemplateBinder) VisitVariable(variable *render3.Variable) interface{} {
	// Register the `Variable` as a symbol in the current `Template`.
	if tb.rootNode != nil {
		tb.symbols[variable] = tb.rootNode.(*render3.Template)
	}
	return nil
}

// VisitReference visits a Reference node
func (tb *TemplateBinder) VisitReference(reference *render3.Reference) interface{} {
	// Register the `Reference` as a symbol in the current `Template`.
	if tb.rootNode != nil {
		tb.symbols[reference] = tb.rootNode.(*render3.Template)
	}
	return nil
}

// VisitDeferredBlock visits a DeferredBlock node
func (tb *TemplateBinder) VisitDeferredBlock(deferred *render3.DeferredBlock) interface{} {
	tb.ingestScopedNode(deferred)
	if deferred.Triggers != nil && deferred.Triggers.When != nil && deferred.Triggers.When.Value != nil {
		deferred.Triggers.When.Value.Visit(&tb.RecursiveAstVisitor, nil)
	}
	if deferred.PrefetchTriggers != nil && deferred.PrefetchTriggers.When != nil && deferred.PrefetchTriggers.When.Value != nil {
		deferred.PrefetchTriggers.When.Value.Visit(&tb.RecursiveAstVisitor, nil)
	}
	if deferred.HydrateTriggers != nil {
		if deferred.HydrateTriggers.When != nil && deferred.HydrateTriggers.When.Value != nil {
			deferred.HydrateTriggers.When.Value.Visit(&tb.RecursiveAstVisitor, nil)
		}
		if deferred.HydrateTriggers.Never != nil {
			deferred.HydrateTriggers.Never.Visit(tb)
		}
	}
	if deferred.Placeholder != nil {
		tb.visitNodeFunc(deferred.Placeholder)
	}
	if deferred.Loading != nil {
		tb.visitNodeFunc(deferred.Loading)
	}
	if deferred.Error != nil {
		tb.visitNodeFunc(deferred.Error)
	}
	return nil
}

// VisitDeferredBlockPlaceholder visits a DeferredBlockPlaceholder node
func (tb *TemplateBinder) VisitDeferredBlockPlaceholder(block *render3.DeferredBlockPlaceholder) interface{} {
	tb.ingestScopedNode(block)
	return nil
}

// VisitDeferredBlockError visits a DeferredBlockError node
func (tb *TemplateBinder) VisitDeferredBlockError(block *render3.DeferredBlockError) interface{} {
	tb.ingestScopedNode(block)
	return nil
}

// VisitDeferredBlockLoading visits a DeferredBlockLoading node
func (tb *TemplateBinder) VisitDeferredBlockLoading(block *render3.DeferredBlockLoading) interface{} {
	tb.ingestScopedNode(block)
	return nil
}

// VisitSwitchBlockCase visits a SwitchBlockCase node
func (tb *TemplateBinder) VisitSwitchBlockCase(block *render3.SwitchBlockCase) interface{} {
	if block.Expression != nil {
		block.Expression.Visit(&tb.RecursiveAstVisitor, nil)
	}
	tb.ingestScopedNode(block)
	return nil
}

// VisitForLoopBlock visits a ForLoopBlock node
func (tb *TemplateBinder) VisitForLoopBlock(block *render3.ForLoopBlock) interface{} {
	if block.Expression != nil && block.Expression.AST != nil {
		block.Expression.AST.Visit(&tb.RecursiveAstVisitor, nil)
	}
	tb.ingestScopedNode(block)
	if block.Empty != nil {
		tb.visitNodeFunc(block.Empty)
	}
	return nil
}

// VisitForLoopBlockEmpty visits a ForLoopBlockEmpty node
func (tb *TemplateBinder) VisitForLoopBlockEmpty(block *render3.ForLoopBlockEmpty) interface{} {
	tb.ingestScopedNode(block)
	return nil
}

// VisitIfBlockBranch visits an IfBlockBranch node
func (tb *TemplateBinder) VisitIfBlockBranch(block *render3.IfBlockBranch) interface{} {
	if block.Expression != nil {
		block.Expression.Visit(&tb.RecursiveAstVisitor, nil)
	}
	tb.ingestScopedNode(block)
	return nil
}

// VisitContent visits a Content node
func (tb *TemplateBinder) VisitContent(content *render3.Content) interface{} {
	tb.ingestScopedNode(content)
	return nil
}

// VisitLetDeclaration visits a LetDeclaration node
func (tb *TemplateBinder) VisitLetDeclaration(decl *render3.LetDeclaration) interface{} {
	tb.RecursiveAstVisitor.Visit(decl.Value, nil)

	if tb.rootNode != nil {
		tb.symbols[decl] = tb.rootNode.(*render3.Template)
	}
	return nil
}

// VisitPipe visits a BindingPipe node
func (tb *TemplateBinder) VisitPipe(ast *expression_parser.BindingPipe, context interface{}) interface{} {
	tb.usedPipes[ast.Name] = true
	if !tb.scope.IsDeferred {
		tb.eagerPipes[ast.Name] = true
	}
	return tb.RecursiveAstVisitor.VisitPipe(ast, context)
}

// VisitPropertyRead visits a PropertyRead node
func (tb *TemplateBinder) VisitPropertyRead(ast *expression_parser.PropertyRead, context interface{}) interface{} {
	tb.maybeMap(ast, ast.Name)
	return tb.RecursiveAstVisitor.VisitPropertyRead(ast, context)
}

// VisitSafePropertyRead visits a SafePropertyRead node
func (tb *TemplateBinder) VisitSafePropertyRead(ast *expression_parser.SafePropertyRead, context interface{}) interface{} {
	tb.maybeMap(ast, ast.Name)
	return tb.RecursiveAstVisitor.VisitSafePropertyRead(ast, context)
}

// ingestScopedNode processes a scoped node recursively
func (tb *TemplateBinder) ingestScopedNode(node ScopedNode) {
	childScope := tb.scope.GetChildScope(node)
	binder := &TemplateBinder{
		RecursiveAstVisitor: expression_parser.RecursiveAstVisitor{},
		bindings:            tb.bindings,
		symbols:             tb.symbols,
		usedPipes:           tb.usedPipes,
		eagerPipes:          tb.eagerPipes,
		deferBlocks:         tb.deferBlocks,
		nestingLevel:        tb.nestingLevel,
		scope:               childScope,
		rootNode:            node,
		level:               tb.level + 1,
		visitNodeFunc:       tb.visitNodeFunc,
	}
	binder.ingest(node)
}

// maybeMap maps an AST expression to a template entity if it refers to one
func (tb *TemplateBinder) maybeMap(ast interface{}, name string) {
	var receiver expression_parser.AST
	switch a := ast.(type) {
	case *expression_parser.PropertyRead:
		receiver = a.Receiver
	case *expression_parser.SafePropertyRead:
		receiver = a.Receiver
	default:
		return
	}

	// If the receiver of the expression isn't the `ImplicitReceiver`, this isn't the root of an
	// `AST` expression that maps to a `Variable` or `Reference`.
	if _, ok := receiver.(*expression_parser.ImplicitReceiver); !ok {
		return
	}
	if _, ok := receiver.(*expression_parser.ThisReceiver); ok {
		return
	}

	// Check whether the name exists in the current scope. If so, map it. Otherwise, the name is
	// probably a property on the top-level component context.
	target := tb.scope.Lookup(name)
	if target != nil {
		switch a := ast.(type) {
		case *expression_parser.PropertyRead:
			tb.bindings[a] = target
		case *expression_parser.SafePropertyRead:
			tb.bindings[a] = target
		}
	}
}

// Unused visitors for TemplateBinder
func (tb *TemplateBinder) VisitElement(element *render3.Element) interface{} {
	for _, attr := range element.Attributes {
		tb.visitNodeFunc(attr)
	}
	for _, input := range element.Inputs {
		tb.visitNodeFunc(input)
	}
	for _, output := range element.Outputs {
		tb.visitNodeFunc(output)
	}
	for _, directive := range element.Directives {
		tb.visitNodeFunc(directive)
	}
	for _, ref := range element.References {
		tb.visitNodeFunc(ref)
	}
	for _, child := range element.Children {
		tb.visitNodeFunc(child)
	}
	return nil
}

func (tb *TemplateBinder) VisitBoundAttribute(attr *render3.BoundAttribute) interface{} {
	if attr.Value != nil {
		attr.Value.Visit(&tb.RecursiveAstVisitor, nil)
	}
	return nil
}

func (tb *TemplateBinder) VisitBoundEvent(event *render3.BoundEvent) interface{} {
	if event.Handler != nil {
		event.Handler.Visit(&tb.RecursiveAstVisitor, nil)
	}
	return nil
}

func (tb *TemplateBinder) VisitBoundText(text *render3.BoundText) interface{} {
	if text.Value != nil {
		text.Value.Visit(&tb.RecursiveAstVisitor, nil)
	}
	return nil
}

func (tb *TemplateBinder) VisitIcu(icu *render3.Icu) interface{} {
	for _, boundText := range icu.Vars {
		boundText.Value.Visit(&tb.RecursiveAstVisitor, nil)
	}
	for _, placeholder := range icu.Placeholders {
		if node, ok := placeholder.(render3.Node); ok {
			tb.visitNodeFunc(node)
		}
	}
	return nil
}

func (tb *TemplateBinder) VisitDeferredTrigger(trigger *render3.DeferredTrigger) interface{} {
	// Handle different trigger types by checking the underlying type
	// Note: DeferredTrigger is a struct, not an interface, so we need to check its methods
	// For now, we'll delegate to the trigger's Visit method if it exists
	return nil
}

func (tb *TemplateBinder) VisitSwitchBlock(block *render3.SwitchBlock) interface{} {
	if block.Expression != nil {
		block.Expression.Visit(&tb.RecursiveAstVisitor, nil)
	}
	for _, caseNode := range block.Cases {
		tb.visitNodeFunc(caseNode)
	}
	return nil
}

func (tb *TemplateBinder) VisitIfBlock(block *render3.IfBlock) interface{} {
	for _, branch := range block.Branches {
		tb.visitNodeFunc(branch)
	}
	return nil
}

func (tb *TemplateBinder) VisitComponent(component *render3.Component) interface{} {
	// Component is similar to Element
	for _, attr := range component.Attributes {
		tb.visitNodeFunc(attr)
	}
	for _, input := range component.Inputs {
		tb.visitNodeFunc(input)
	}
	for _, output := range component.Outputs {
		tb.visitNodeFunc(output)
	}
	for _, directive := range component.Directives {
		tb.visitNodeFunc(directive)
	}
	for _, ref := range component.References {
		tb.visitNodeFunc(ref)
	}
	for _, child := range component.Children {
		tb.visitNodeFunc(child)
	}
	return nil
}

func (tb *TemplateBinder) VisitDirective(directive *render3.Directive) interface{} {
	return nil
}

func (tb *TemplateBinder) VisitText(text *render3.Text) interface{} {
	return nil
}

func (tb *TemplateBinder) VisitTextAttribute(attr *render3.TextAttribute) interface{} {
	return nil
}

// VisitContent is already defined above, so we don't need to redefine it

func (tb *TemplateBinder) VisitUnknownBlock(block *render3.UnknownBlock) interface{} {
	return nil
}

// R3BoundTarget is a metadata container for a `Target` that allows queries for specific bits of metadata.
//
// See `BoundTarget` for documentation on the individual methods.
type R3BoundTarget struct {
	target             *Target
	directives         MatchedDirectives
	eagerDirectives    []interface{}
	missingDirectives  map[string]bool
	bindings           BindingsMap
	references         ReferenceMap
	exprTargets        map[expression_parser.AST]TemplateEntity
	symbols            map[TemplateEntity]*render3.Template
	nestingLevel       map[ScopedNode]int
	scopedNodeEntities ScopedNodeEntities
	usedPipes          map[string]bool
	eagerPipes         map[string]bool
	deferredBlocks     []*render3.DeferredBlock
	deferredScopes     map[*render3.DeferredBlock]*Scope
}

// NewR3BoundTarget creates a new R3BoundTarget
func NewR3BoundTarget(
	target *Target,
	directives MatchedDirectives,
	eagerDirectives []interface{},
	missingDirectives map[string]bool,
	bindings BindingsMap,
	references ReferenceMap,
	exprTargets map[expression_parser.AST]TemplateEntity,
	symbols map[TemplateEntity]*render3.Template,
	nestingLevel map[ScopedNode]int,
	scopedNodeEntities ScopedNodeEntities,
	usedPipes map[string]bool,
	eagerPipes map[string]bool,
	rawDeferred []DeferBlockScope,
) *R3BoundTarget {
	deferredBlocks := make([]*render3.DeferredBlock, len(rawDeferred))
	deferredScopes := make(map[*render3.DeferredBlock]*Scope)
	for i, current := range rawDeferred {
		deferredBlocks[i] = current.Block
		deferredScopes[current.Block] = current.Scope
	}

	return &R3BoundTarget{
		target:             target,
		directives:         directives,
		eagerDirectives:    eagerDirectives,
		missingDirectives:  missingDirectives,
		bindings:           bindings,
		references:         references,
		exprTargets:        exprTargets,
		symbols:            symbols,
		nestingLevel:       nestingLevel,
		scopedNodeEntities: scopedNodeEntities,
		usedPipes:          usedPipes,
		eagerPipes:         eagerPipes,
		deferredBlocks:     deferredBlocks,
		deferredScopes:     deferredScopes,
	}
}

// Target returns the original `Target` that was bound.
func (bt *R3BoundTarget) Target() *Target {
	return bt.target
}

// GetEntitiesInScope returns all `Reference`s and `Variables` visible within the given `ScopedNode` (or at the top
// level, if `null` is passed).
func (bt *R3BoundTarget) GetEntitiesInScope(node ScopedNode) []TemplateEntity {
	if entities, exists := bt.scopedNodeEntities[node]; exists {
		result := make([]TemplateEntity, 0, len(entities))
		for entity := range entities {
			result = append(result, entity)
		}
		return result
	}
	return []TemplateEntity{}
}

// GetDirectivesOfNode returns, for a given template node (either an `Element` or a `Template`), the set of directives
// which matched the node, if any.
func (bt *R3BoundTarget) GetDirectivesOfNode(node DirectiveOwner) []interface{} {
	if dirs, exists := bt.directives[node]; exists {
		return dirs
	}
	return nil
}

// GetReferenceTarget returns, for a given `Reference`, the reference's target - either an `Element`, a `Template`, or
// a directive on a particular node.
func (bt *R3BoundTarget) GetReferenceTarget(ref *render3.Reference) ReferenceTarget {
	if target, exists := bt.references[ref]; exists {
		if rt, ok := target.(ReferenceTarget); ok {
			return rt
		}
		// Handle different reference target types
		switch t := target.(type) {
		case *render3.Element:
			return &ReferenceTargetElement{Element: t}
		case *render3.Template:
			return &ReferenceTargetTemplate{Template: t}
		case *ReferenceTargetWithDirective:
			return t
		}
	}
	return nil
}

// GetConsumerOfBinding returns, for a given binding, the entity to which the binding is being made.
//
// This will either be a directive or the node itself.
func (bt *R3BoundTarget) GetConsumerOfBinding(
	binding interface{}, // BoundAttribute | BoundEvent | TextAttribute
) interface{} { // DirectiveT | Element | Template | null
	if consumer, exists := bt.bindings[binding]; exists {
		return consumer
	}
	return nil
}

// GetExpressionTarget returns, if the given `AST` expression refers to a `Reference` or `Variable` within the `Target`, then
// return that.
//
// Otherwise, returns `null`.
//
// This is only defined for `AST` expressions that read or write to a property of an
// `ImplicitReceiver`.
func (bt *R3BoundTarget) GetExpressionTarget(expr expression_parser.AST) TemplateEntity {
	if target, exists := bt.exprTargets[expr]; exists {
		return target
	}
	return nil
}

// GetDefinitionNodeOfSymbol returns, given a particular `Reference` or `Variable`, the `ScopedNode` which created it.
//
// All `Variable`s are defined on node, so this will always return a value for a `Variable`
// from the `Target`. Returns `null` otherwise.
func (bt *R3BoundTarget) GetDefinitionNodeOfSymbol(symbol TemplateEntity) ScopedNode {
	if template, exists := bt.symbols[symbol]; exists {
		return template
	}
	return nil
}

// GetNestingLevel returns the nesting level of a particular `ScopedNode`.
//
// This starts at 1 for top-level nodes within the `Target` and increases for nodes
// nested at deeper levels.
func (bt *R3BoundTarget) GetNestingLevel(node ScopedNode) int {
	if level, exists := bt.nestingLevel[node]; exists {
		return level
	}
	return 0
}

// GetUsedDirectives returns a list of all the directives used by the target,
// including directives from `@defer` blocks.
func (bt *R3BoundTarget) GetUsedDirectives() []interface{} {
	directiveSet := make(map[interface{}]bool)
	for _, dirs := range bt.directives {
		for _, dir := range dirs {
			directiveSet[dir] = true
		}
	}
	result := make([]interface{}, 0, len(directiveSet))
	for dir := range directiveSet {
		result = append(result, dir)
	}
	return result
}

// GetEagerlyUsedDirectives returns a list of eagerly used directives from the target.
// Note: this list *excludes* directives from `@defer` blocks.
func (bt *R3BoundTarget) GetEagerlyUsedDirectives() []interface{} {
	directiveSet := make(map[interface{}]bool)
	for _, dir := range bt.eagerDirectives {
		directiveSet[dir] = true
	}
	result := make([]interface{}, 0, len(directiveSet))
	for dir := range directiveSet {
		result = append(result, dir)
	}
	return result
}

// GetUsedPipes returns a list of all the pipes used by the target,
// including pipes from `@defer` blocks.
func (bt *R3BoundTarget) GetUsedPipes() []string {
	result := make([]string, 0, len(bt.usedPipes))
	for pipe := range bt.usedPipes {
		result = append(result, pipe)
	}
	return result
}

// GetEagerlyUsedPipes returns a list of eagerly used pipes from the target.
// Note: this list *excludes* pipes from `@defer` blocks.
func (bt *R3BoundTarget) GetEagerlyUsedPipes() []string {
	result := make([]string, 0, len(bt.eagerPipes))
	for pipe := range bt.eagerPipes {
		result = append(result, pipe)
	}
	return result
}

// GetDeferBlocks returns a list of all `@defer` blocks used by the target.
func (bt *R3BoundTarget) GetDeferBlocks() []*render3.DeferredBlock {
	return bt.deferredBlocks
}

// GetDeferredTriggerTarget gets the element that a specific deferred block trigger is targeting.
// block: Block that the trigger belongs to.
// trigger: Trigger whose target is being looked up.
func (bt *R3BoundTarget) GetDeferredTriggerTarget(block *render3.DeferredBlock, trigger render3.DeferredTriggerInterface) *render3.Element {
	// Only triggers that refer to DOM nodes can be resolved.
	if _, ok := trigger.(*render3.InteractionDeferredTrigger); !ok {
		if _, ok := trigger.(*render3.ViewportDeferredTrigger); !ok {
			if _, ok := trigger.(*render3.HoverDeferredTrigger); !ok {
				return nil
			}
		}
	}

	var name *string
	switch t := trigger.(type) {
	case *render3.InteractionDeferredTrigger:
		name = t.Reference
	case *render3.ViewportDeferredTrigger:
		name = t.Reference
	case *render3.HoverDeferredTrigger:
		name = t.Reference
	default:
		return nil
	}

	if name == nil {
		var target *render3.Element

		if block.Placeholder != nil {
			for _, child := range block.Placeholder.Children {
				// Skip over comment nodes. Currently by default the template parser doesn't capture
				// comments, but we have a safeguard here just in case since it can be enabled.
				if _, ok := child.(*render3.Comment); ok {
					continue
				}

				// We can only infer the trigger if there's one root element node. Any other
				// nodes at the root make it so that we can't infer the trigger anymore.
				if target != nil {
					return nil
				}

				if element, ok := child.(*render3.Element); ok {
					target = element
				}
			}
		}

		return target
	}

	outsideRef := bt.findEntityInScope(block, *name)

	// First try to resolve the target in the scope of the main deferred block. Note that we
	// skip triggers defined inside the main block itself, because they might not exist yet.
	if ref, ok := outsideRef.(*render3.Reference); ok {
		if bt.GetDefinitionNodeOfSymbol(ref) != block {
			target := bt.GetReferenceTarget(ref)
			if target != nil {
				return bt.referenceTargetToElement(target)
			}
		}
	}

	// If the trigger couldn't be found in the main block, check the
	// placeholder block which is shown before the main block has loaded.
	if block.Placeholder != nil {
		refInPlaceholder := bt.findEntityInScope(block.Placeholder, *name)
		var targetInPlaceholder ReferenceTarget
		if ref, ok := refInPlaceholder.(*render3.Reference); ok {
			targetInPlaceholder = bt.GetReferenceTarget(ref)
		}

		if targetInPlaceholder != nil {
			return bt.referenceTargetToElement(targetInPlaceholder)
		}
	}

	return nil
}

// IsDeferred returns whether a given node is located in a `@defer` block.
func (bt *R3BoundTarget) IsDeferred(element *render3.Element) bool {
	for _, block := range bt.deferredBlocks {
		scope, exists := bt.deferredScopes[block]
		if !exists {
			continue
		}

		stack := []*Scope{scope}

		for len(stack) > 0 {
			current := stack[len(stack)-1]
			stack = stack[:len(stack)-1]

			if current.ElementLikeInScope[element] {
				return true
			}

			for _, childScope := range current.ChildScopes {
				stack = append(stack, childScope)
			}
		}
	}

	return false
}

// ReferencedDirectiveExists checks whether a component/directive that was referenced directly in the template exists.
// name: Name of the component/directive.
func (bt *R3BoundTarget) ReferencedDirectiveExists(name string) bool {
	return !bt.missingDirectives[name]
}

// findEntityInScope finds an entity with a specific name in a scope.
// rootNode: Root node of the scope.
// name: Name of the entity.
func (bt *R3BoundTarget) findEntityInScope(rootNode ScopedNode, name string) TemplateEntity {
	entities := bt.GetEntitiesInScope(rootNode)

	for _, entity := range entities {
		var entityName string
		switch e := entity.(type) {
		case *render3.Reference:
			entityName = e.Name
		case *render3.Variable:
			entityName = e.Name
		case *render3.LetDeclaration:
			entityName = e.Name
		default:
			continue
		}

		if entityName == name {
			return entity
		}
	}

	return nil
}

// referenceTargetToElement coerces a `ReferenceTarget` to an `Element`, if possible.
func (bt *R3BoundTarget) referenceTargetToElement(target ReferenceTarget) *render3.Element {
	if elementTarget, ok := target.(*ReferenceTargetElement); ok {
		return elementTarget.Element
	}

	if _, ok := target.(*ReferenceTargetTemplate); ok {
		return nil
	}

	if directiveTarget, ok := target.(*ReferenceTargetWithDirective); ok {
		switch n := directiveTarget.Node.(type) {
		case *render3.Component:
			return nil
		case *render3.Directive:
			return nil
		case *render3.HostElement:
			return nil
		case *render3.Element:
			return bt.referenceTargetToElement(&ReferenceTargetElement{Element: n})
		case *render3.Template:
			return nil
		}
	}

	return nil
}

// extractScopedNodeEntities extracts scoped node entities from a scope
func extractScopedNodeEntities(rootScope *Scope, templateEntities ScopedNodeEntities) {
	entityMap := make(map[ScopedNode]map[string]TemplateEntity)

	var extractScopeEntities func(scope *Scope) map[string]TemplateEntity
	extractScopeEntities = func(scope *Scope) map[string]TemplateEntity {
		if entities, exists := entityMap[scope.rootNode]; exists {
			return entities
		}

		currentEntities := scope.NamedEntities

		var entities map[string]TemplateEntity
		if scope.parentScope != nil {
			parentEntities := extractScopeEntities(scope.parentScope)
			entities = make(map[string]TemplateEntity)
			for k, v := range parentEntities {
				entities[k] = v
			}
			for k, v := range currentEntities {
				entities[k] = v
			}
		} else {
			entities = make(map[string]TemplateEntity)
			for k, v := range currentEntities {
				entities[k] = v
			}
		}

		entityMap[scope.rootNode] = entities
		return entities
	}

	scopesToProcess := []*Scope{rootScope}
	for len(scopesToProcess) > 0 {
		scope := scopesToProcess[len(scopesToProcess)-1]
		scopesToProcess = scopesToProcess[:len(scopesToProcess)-1]
		for _, childScope := range scope.ChildScopes {
			scopesToProcess = append(scopesToProcess, childScope)
		}
		extractScopeEntities(scope)
	}

	for template, entities := range entityMap {
		entitySet := make(map[TemplateEntity]bool)
		for _, entity := range entities {
			entitySet[entity] = true
		}
		templateEntities[template] = entitySet
	}
}
