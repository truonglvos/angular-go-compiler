package view

import (
	"ngc-go/packages/compiler/src/expression_parser"
	"ngc-go/packages/compiler/src/render3"
)

// ScopedNode represents a node that has a `Scope` associated with it.
// This is a union type: Template | SwitchBlockCase | IfBlockBranch | ForLoopBlock | ForLoopBlockEmpty | DeferredBlock | DeferredBlockError | DeferredBlockLoading | DeferredBlockPlaceholder | Content | HostElement
// In Go, we use render3.Node with type checking via IsScopedNode() since we cannot add methods to non-local types
type ScopedNode interface {
	render3.Node
}

// ReferenceTarget represents possible values that a reference can be resolved to.
// This is a union type: {directive: DirectiveT; node: Exclude<DirectiveOwner, HostElement>} | Element | Template
type ReferenceTarget interface {
	// Marker interface for reference targets
	isReferenceTarget()
}

// ReferenceTargetWithDirective represents a reference target that includes a directive
type ReferenceTargetWithDirective struct {
	Directive interface{} // DirectiveT
	Node      DirectiveOwner
}

func (r *ReferenceTargetWithDirective) isReferenceTarget() {}

// ReferenceTargetElement represents a reference target that is an Element
type ReferenceTargetElement struct {
	Element *render3.Element
}

func (r *ReferenceTargetElement) isReferenceTarget() {}

// ReferenceTargetTemplate represents a reference target that is a Template
type ReferenceTargetTemplate struct {
	Template *render3.Template
}

func (r *ReferenceTargetTemplate) isReferenceTarget() {}

// TemplateEntity represents an entity that is local to the template and defined within the template.
// This is a union type: Reference | Variable | LetDeclaration
// In Go, we use interface{} with type assertions since we cannot add methods to non-local types
type TemplateEntity interface{}

// IsTemplateEntity checks if a value is a TemplateEntity
func IsTemplateEntity(v interface{}) bool {
	switch v.(type) {
	case *render3.Reference, *render3.Variable, *render3.LetDeclaration:
		return true
	default:
		return false
	}
}

// DirectiveOwner represents nodes that can have directives applied to them.
// This is a union type: Element | Template | Component | Directive | HostElement
// In Go, we use interface{} with type assertions since we cannot add methods to non-local types
type DirectiveOwner interface {
	render3.Node
}

// IsDirectiveOwner checks if a node is a DirectiveOwner
func IsDirectiveOwner(node render3.Node) bool {
	switch node.(type) {
	case *render3.Element, *render3.Template, *render3.Component, *render3.Directive, *render3.HostElement:
		return true
	default:
		return false
	}
}

// IsScopedNode checks if a node is a ScopedNode
func IsScopedNode(node render3.Node) bool {
	switch node.(type) {
	case *render3.Template, *render3.SwitchBlockCase, *render3.IfBlockBranch,
		*render3.ForLoopBlock, *render3.ForLoopBlockEmpty, *render3.DeferredBlock,
		*render3.DeferredBlockError, *render3.DeferredBlockLoading,
		*render3.DeferredBlockPlaceholder, *render3.Content, *render3.HostElement:
		return true
	default:
		return false
	}
}

/*
 * t2 is the replacement for the `TemplateDefinitionBuilder`. It handles the operations of
 * analyzing Angular templates, extracting semantic info, and ultimately producing a template
 * definition function which renders the template using Ivy instructions.
 *
 * t2 data is also utilized by the template type-checking facilities to understand a template enough
 * to generate type-checking code for it.
 */

// Target represents a logical target for analysis, which could contain a template or other types of bindings.
type Target struct {
	Template []render3.Node // optional
	Host     *HostTarget    // optional
}

// HostTarget represents host bindings target
type HostTarget struct {
	Node       *render3.HostElement
	Directives []interface{} // DirectiveT[]
}

// InputOutputPropertySet is a data structure which can indicate whether a given property name is present or not.
//
// This is used to represent the set of inputs or outputs present on a directive, and allows the
// binder to query for the presence of a mapping for property names.
type InputOutputPropertySet interface {
	HasBindingPropertyName(propertyName string) bool
}

// LegacyAnimationTriggerNames is a data structure which captures the animation trigger names that are statically resolvable
// and whether some names could not be statically evaluated.
type LegacyAnimationTriggerNames struct {
	IncludesDynamicAnimations bool
	StaticTriggerNames        []string
}

// DirectiveMeta represents metadata regarding a directive that's needed to match it against template elements.
// This is provided by a consumer of the t2 APIs.
type DirectiveMeta interface {
	// Name returns the name of the directive class (used for debugging).
	Name() string

	// Selector returns the selector for the directive or `null` if there isn't one.
	Selector() *string

	// IsComponent returns whether the directive is a component.
	IsComponent() bool

	// Inputs returns the set of inputs which this directive claims.
	// Goes from property names to field names.
	Inputs() InputOutputPropertySet

	// Outputs returns the set of outputs which this directive claims.
	// Goes from property names to field names.
	Outputs() InputOutputPropertySet

	// ExportAs returns the name under which the directive is exported, if any (exportAs in Angular).
	// Returns nil otherwise
	ExportAs() []string

	// IsStructural returns whether the directive is a structural directive (e.g. `<div *ngIf></div>`).
	IsStructural() bool

	// NgContentSelectors returns, if the directive is a component, the selectors of its `ng-content` elements.
	NgContentSelectors() []string

	// PreserveWhitespaces returns whether the template of the component preserves whitespaces.
	PreserveWhitespaces() bool

	// AnimationTriggerNames returns the name of legacy animations that the user defines in the component.
	// Only includes the legacy animation names.
	AnimationTriggerNames() *LegacyAnimationTriggerNames
}

// TargetBinder is an interface to the binding API, which processes a template and returns an object similar to the
// `ts.TypeChecker`.
//
// The returned `BoundTarget` has an API for extracting information about the processed target.
type TargetBinder interface {
	// Bind processes a target and returns a BoundTarget
	Bind(target *Target) BoundTarget
}

// BoundTarget represents the result of performing the binding operation against a `Target`.
//
// The original `Target` is accessible, as well as a suite of methods for extracting binding
// information regarding the `Target`.
type BoundTarget interface {
	// Target returns the original `Target` that was bound.
	Target() *Target

	// GetDirectivesOfNode returns, for a given template node (either an `Element` or a `Template`), the set of directives
	// which matched the node, if any.
	GetDirectivesOfNode(node DirectiveOwner) []interface{} // DirectiveT[] | null

	// GetReferenceTarget returns, for a given `Reference`, the reference's target - either an `Element`, a `Template`, or
	// a directive on a particular node.
	GetReferenceTarget(ref *render3.Reference) ReferenceTarget

	// GetConsumerOfBinding returns, for a given binding, the entity to which the binding is being made.
	//
	// This will either be a directive or the node itself.
	GetConsumerOfBinding(
		binding interface{}, // BoundAttribute | BoundEvent | TextAttribute
	) interface{} // DirectiveT | Element | Template | null

	// GetExpressionTarget returns, if the given `AST` expression refers to a `Reference` or `Variable` within the `Target`, then
	// return that.
	//
	// Otherwise, returns `null`.
	//
	// This is only defined for `AST` expressions that read or write to a property of an
	// `ImplicitReceiver`.
	GetExpressionTarget(expr expression_parser.AST) TemplateEntity

	// GetDefinitionNodeOfSymbol returns, given a particular `Reference` or `Variable`, the `ScopedNode` which created it.
	//
	// All `Variable`s are defined on node, so this will always return a value for a `Variable`
	// from the `Target`. Returns `null` otherwise.
	GetDefinitionNodeOfSymbol(symbol TemplateEntity) ScopedNode // ScopedNode | null

	// GetNestingLevel returns the nesting level of a particular `ScopedNode`.
	//
	// This starts at 1 for top-level nodes within the `Target` and increases for nodes
	// nested at deeper levels.
	GetNestingLevel(node ScopedNode) int

	// GetEntitiesInScope returns all `Reference`s and `Variables` visible within the given `ScopedNode` (or at the top
	// level, if `null` is passed).
	// Returns a slice representing a ReadonlySet<TemplateEntity> (in TypeScript)
	// node can be nil to represent top level
	GetEntitiesInScope(node ScopedNode) []TemplateEntity // ReadonlySet<TemplateEntity>, node: ScopedNode | null

	// GetUsedDirectives returns a list of all the directives used by the target,
	// including directives from `@defer` blocks.
	GetUsedDirectives() []interface{} // DirectiveT[]

	// GetEagerlyUsedDirectives returns a list of eagerly used directives from the target.
	// Note: this list *excludes* directives from `@defer` blocks.
	GetEagerlyUsedDirectives() []interface{} // DirectiveT[]

	// GetUsedPipes returns a list of all the pipes used by the target,
	// including pipes from `@defer` blocks.
	GetUsedPipes() []string

	// GetEagerlyUsedPipes returns a list of eagerly used pipes from the target.
	// Note: this list *excludes* pipes from `@defer` blocks.
	GetEagerlyUsedPipes() []string

	// GetDeferBlocks returns a list of all `@defer` blocks used by the target.
	GetDeferBlocks() []*render3.DeferredBlock

	// GetDeferredTriggerTarget gets the element that a specific deferred block trigger is targeting.
	// block: Block that the trigger belongs to.
	// trigger: Trigger whose target is being looked up.
	GetDeferredTriggerTarget(block *render3.DeferredBlock, trigger render3.DeferredTriggerInterface) *render3.Element

	// IsDeferred returns whether a given node is located in a `@defer` block.
	IsDeferred(node *render3.Element) bool

	// ReferencedDirectiveExists checks whether a component/directive that was referenced directly in the template exists.
	// name: Name of the component/directive.
	ReferencedDirectiveExists(name string) bool
}
