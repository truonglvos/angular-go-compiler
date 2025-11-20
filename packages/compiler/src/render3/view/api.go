package view

import (
	"ngc-go/packages/compiler/src/core"
	"ngc-go/packages/compiler/src/output"
	"ngc-go/packages/compiler/src/render3"
	"ngc-go/packages/compiler/src/util"
)

// DeferBlockDepsEmitMode defines how dynamic imports for deferred dependencies should be emitted
// in the generated output:
//   - either in a function on per-component basis (in case of local compilation)
//   - or in a function on per-block basis (in full compilation mode)
type DeferBlockDepsEmitMode int

const (
	// DeferBlockDepsEmitModePerBlock - Dynamic imports are grouped on per-block basis.
	// This is used in full compilation mode, when compiler has more information
	// about particular dependencies that belong to this block.
	DeferBlockDepsEmitModePerBlock DeferBlockDepsEmitMode = iota

	// DeferBlockDepsEmitModePerComponent - Dynamic imports are grouped on per-component basis.
	// In local compilation, compiler doesn't have enough information to determine
	// which deferred dependencies belong to which block. In this case we group all
	// dynamic imports into a single file on per-component basis.
	DeferBlockDepsEmitModePerComponent
)

// DeclarationListEmitMode specifies how a list of declaration type references should be emitted into the generated code.
type DeclarationListEmitMode int

const (
	// DeclarationListEmitModeDirect - The list of declarations is emitted into the generated code as is.
	// ```ts
	// directives: [MyDir],
	// ```
	DeclarationListEmitModeDirect DeclarationListEmitMode = iota

	// DeclarationListEmitModeClosure - The list of declarations is emitted into the generated code wrapped inside a closure, which
	// is needed when at least one declaration is a forward reference.
	// ```ts
	// directives: function () { return [MyDir, ForwardDir]; },
	// ```
	DeclarationListEmitModeClosure

	// DeclarationListEmitModeClosureResolved - Similar to `Closure`, with the addition that the list of declarations can contain individual
	// items that are themselves forward references. This is relevant for JIT compilations, as
	// unwrapping the forwardRef cannot be done statically so must be deferred. This mode emits
	// the declaration list using a mapping transform through `resolveForwardRef` to ensure that
	// any forward references within the list are resolved when the outer closure is invoked.
	DeclarationListEmitModeClosureResolved

	// DeclarationListEmitModeRuntimeResolved - Runtime resolved declarations
	DeclarationListEmitModeRuntimeResolved
)

// R3TemplateDependencyKind represents the kind of template dependency
type R3TemplateDependencyKind int

const (
	R3TemplateDependencyKindDirective R3TemplateDependencyKind = iota
	R3TemplateDependencyKindPipe
	R3TemplateDependencyKindNgModule
)

// R3DirectiveMetadata contains information needed to compile a directive for the render3 runtime.
type R3DirectiveMetadata struct {
	// Name of the directive type.
	Name string

	// An expression representing a reference to the directive itself.
	Type render3.R3Reference

	// Number of generic type parameters of the type itself.
	TypeArgumentCount int

	// A source span for the directive type.
	TypeSourceSpan *util.ParseSourceSpan

	// Dependencies of the directive's constructor.
	// Can be []factory.R3DependencyMetadata, "invalid", or nil
	Deps interface{}

	// Unparsed selector of the directive, or `null` if there was no selector.
	Selector *string

	// Information about the content queries made by the directive.
	Queries []R3QueryMetadata

	// Information about the view queries made by the directive.
	ViewQueries []R3QueryMetadata

	// Mappings indicating how the directive interacts with its host element (host bindings,
	// listeners, etc).
	Host R3HostMetadata

	// Information about usage of specific lifecycle events which require special treatment in the
	// code generator.
	Lifecycle R3LifecycleMetadata

	// A mapping of inputs from class property names to binding property names, or to a tuple of
	// binding property name and class property name if the names are different.
	Inputs map[string]R3InputMetadata

	// A mapping of outputs from class property names to binding property names, or to a tuple of
	// binding property name and class property name if the names are different.
	Outputs map[string]string

	// Whether or not the component or directive inherits from another class
	UsesInheritance bool

	// Reference name under which to export the directive's type in a template,
	// if any.
	ExportAs []string // can be nil for null

	// The list of providers defined in the directive.
	Providers *output.OutputExpression // null if not set

	// Whether or not the component or directive is standalone.
	IsStandalone bool

	// Whether or not the component or directive is signal-based.
	IsSignal bool

	// Additional directives applied to the directive host.
	HostDirectives []R3HostDirectiveMetadata // can be nil for null
}

// R3LifecycleMetadata contains information about usage of specific lifecycle events
type R3LifecycleMetadata struct {
	// Whether the directive uses NgOnChanges.
	UsesOnChanges bool
}

// R3ComponentMetadata contains information needed to compile a component for the render3 runtime.
type R3ComponentMetadata struct {
	R3DirectiveMetadata

	// Information about the component's template.
	Template R3ComponentTemplateMetadata

	Declarations []R3TemplateDependency

	// Metadata related to the deferred blocks in the component's template.
	Defer R3ComponentDeferMetadata

	// Specifies how the 'directives' and/or `pipes` array, if generated, need to be emitted.
	DeclarationListEmitMode DeclarationListEmitMode

	// A collection of styling data that will be applied and scoped to the component.
	Styles []string

	// A collection of style paths for external stylesheets that will be applied and scoped to the component.
	ExternalStyles []string // optional, can be nil

	// An encapsulation policy for the component's styling.
	// Possible values:
	// - `ViewEncapsulation.Emulated`: Apply modified component styles in order to emulate
	//                                 a native Shadow DOM CSS encapsulation behavior.
	// - `ViewEncapsulation.None`: Apply component styles globally without any sort of encapsulation.
	// - `ViewEncapsulation.ShadowDom`: Use the browser's native Shadow DOM API to encapsulate styles.
	Encapsulation core.ViewEncapsulation

	// A collection of animation triggers that will be used in the component template.
	Animations *output.OutputExpression // null if not set

	// The list of view providers defined in the component.
	ViewProviders *output.OutputExpression // null if not set

	// Path to the .ts file in which this template's generated code will be included, relative to
	// the compilation root. This will be used to generate identifiers that need to be globally
	// unique in certain contexts (such as g3).
	RelativeContextFilePath string

	// Whether translation variable name should contain external message id
	// (used by Closure Compiler's output of `goog.getMsg` for transition period).
	I18nUseExternalIds bool

	// Strategy used for detecting changes in the component.
	//
	// In global compilation mode the value is ChangeDetectionStrategy if available as it is
	// statically resolved during analysis phase. Whereas in local compilation mode the value is the
	// expression as appears in the decorator.
	ChangeDetection interface{} // core.ChangeDetectionStrategy | output.OutputExpression | nil

	// Relative path to the component's template from the root of the project.
	// Used to generate debugging information.
	RelativeTemplatePath *string

	// Whether any of the component's dependencies are directives.
	HasDirectiveDependencies bool

	// The imports expression as appears on the component decorate for standalone component. This
	// field is currently needed only for local compilation, and so in other compilation modes it may
	// not be set. If component has empty array imports then this field is not set.
	RawImports *output.OutputExpression // optional, can be nil
}

// R3ComponentTemplateMetadata contains information about the component's template.
type R3ComponentTemplateMetadata struct {
	// Parsed nodes of the template.
	Nodes []render3.Node

	// Any ng-content selectors extracted from the template. Contains `*` when an ng-content
	// element without selector is present.
	NgContentSelectors []string

	// Whether the template preserves whitespaces from the user's code.
	PreserveWhitespaces *bool // optional, can be nil
}

// R3ComponentDeferMetadata contains information about the deferred blocks in a component's template.
// This is a union type: either PerBlock mode with Blocks map, or PerComponent mode with DependenciesFn.
// Only one of Blocks or DependenciesFn should be set based on Mode.
type R3ComponentDeferMetadata struct {
	Mode DeferBlockDepsEmitMode

	// For PerBlock mode: blocks map (value can be null)
	Blocks map[*render3.DeferredBlock]*output.OutputExpression

	// For PerComponent mode: dependencies function (can be null)
	DependenciesFn *output.OutputExpression
}

// R3InputMetadata contains metadata for an individual input on a directive.
type R3InputMetadata struct {
	ClassPropertyName   string
	BindingPropertyName string
	Required            bool
	IsSignal            bool
	// Transform function for the input.
	//
	// Null if there is no transform, or if this is a signal input.
	// Signal inputs capture their transform as part of the `InputSignal`.
	TransformFunction *output.OutputExpression // null if not set
}

// R3TemplateDependency is a dependency that's used within a component template.
type R3TemplateDependency struct {
	Kind R3TemplateDependencyKind

	// The type of the dependency as an expression.
	Type output.OutputExpression
}

// R3DirectiveDependencyMetadata contains information about a directive that is used in a component template.
// Only the stable, public facing information of the directive is stored here.
type R3DirectiveDependencyMetadata struct {
	R3TemplateDependency

	// The selector of the directive.
	Selector string

	// The binding property names of the inputs of the directive.
	Inputs []string

	// The binding property names of the outputs of the directive.
	Outputs []string

	// Name under which the directive is exported, if any (exportAs in Angular). Null otherwise.
	ExportAs []string // can be nil for null

	// If true then this directive is actually a component; otherwise it is not.
	IsComponent bool
}

// R3PipeDependencyMetadata contains information about a pipe that is used in a component template.
type R3PipeDependencyMetadata struct {
	R3TemplateDependency

	Name string
}

// R3NgModuleDependencyMetadata contains information about an NgModule that is used in a component template.
type R3NgModuleDependencyMetadata struct {
	R3TemplateDependency
}

// R3QueryMetadata contains information needed to compile a query (view or content).
type R3QueryMetadata struct {
	// Name of the property on the class to update with query results.
	PropertyName string

	// Whether to read only the first matching result, or an array of results.
	First bool

	// Either an expression representing a type or `InjectionToken` for the query
	// predicate, or a set of string selectors.
	//
	// Note: At compile time we split selectors as an optimization that avoids this
	// extra work at runtime creation phase.
	//
	// Notably, if the selector is not statically analyzable due to an expression,
	// the selectors may need to be split up at runtime.
	Predicate interface{} // render3.MaybeForwardRefExpression | []string

	// Whether to include only direct children or all descendants.
	Descendants bool

	// If the `QueryList` should fire change event only if actual change to query was computed (vs old
	// behavior where the change was fired whenever the query was recomputed, even if the recomputed
	// query resulted in the same list.)
	EmitDistinctChangesOnly bool

	// An expression representing a type to read from each matched node, or null if the default value
	// for a given node is to be returned.
	Read *output.OutputExpression // null if not set

	// Whether or not this query should collect only static results.
	//
	// If static is true, the query's results will be set on the component after nodes are created,
	// but before change detection runs. This means that any results that relied upon change detection
	// to run (e.g. results inside *ngIf or *ngFor views) will not be collected. Query results are
	// available in the ngOnInit hook.
	//
	// If static is false, the query's results will be set on the component after change detection
	// runs. This means that the query results can contain nodes inside *ngIf or *ngFor views, but
	// the results will not be available in the ngOnInit hook (only in the ngAfterContentInit for
	// content hooks and ngAfterViewInit for view hooks).
	//
	// Note: For signal-based queries, this option does not have any effect.
	Static bool

	// Whether the query is signal-based.
	IsSignal bool
}

// R3HostMetadata contains mappings indicating how the class interacts with its
// host element (host bindings, listeners, etc).
type R3HostMetadata struct {
	// A mapping of attribute binding keys to `o.Expression`s.
	Attributes map[string]output.OutputExpression

	// A mapping of event binding keys to unparsed expressions.
	Listeners map[string]string

	// A mapping of property binding keys to unparsed expressions.
	Properties map[string]string

	SpecialAttributes R3HostSpecialAttributes
}

// R3HostSpecialAttributes contains special host attributes
type R3HostSpecialAttributes struct {
	StyleAttr *string
	ClassAttr *string
}

// R3HostDirectiveMetadata contains information needed to compile a host directive for the render3 runtime.
type R3HostDirectiveMetadata struct {
	// An expression representing the host directive class itself.
	Directive render3.R3Reference

	// Whether the expression referring to the host directive is a forward reference.
	IsForwardReference bool

	// Inputs from the host directive that will be exposed on the host.
	Inputs map[string]string // can be nil for null

	// Outputs from the host directive that will be exposed on the host.
	Outputs map[string]string // can be nil for null
}

// R3DeferResolverFunctionMetadata contains information needed to compile the defer block resolver function.
type R3DeferResolverFunctionMetadata struct {
	Mode DeferBlockDepsEmitMode

	// For PerBlock mode: dependencies
	PerBlockDependencies []R3DeferPerBlockDependency

	// For PerComponent mode: dependencies
	PerComponentDependencies []R3DeferPerComponentDependency
}

// R3DeferPerBlockDependency contains information about a single dependency of a defer block in `PerBlock` mode.
type R3DeferPerBlockDependency struct {
	// Reference to a dependency.
	TypeReference output.OutputExpression

	// Dependency class name.
	SymbolName string

	// Whether this dependency can be defer-loaded.
	IsDeferrable bool

	// Import path where this dependency is located.
	ImportPath *string

	// Whether the symbol is the default export.
	IsDefaultImport bool
}

// R3DeferPerComponentDependency contains information about a single dependency of a defer block in `PerComponent` mode.
type R3DeferPerComponentDependency struct {
	// Dependency class name.
	SymbolName string

	// Import path where this dependency is located.
	ImportPath string

	// Whether the symbol is the default export.
	IsDefaultImport bool
}
