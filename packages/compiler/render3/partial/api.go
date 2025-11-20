package partial

import (
	"ngc-go/packages/compiler/core"
	"ngc-go/packages/compiler/facade"
	"ngc-go/packages/compiler/output"
)

// R3PartialDeclaration is the base interface for all partial declarations
type R3PartialDeclaration struct {
	// The minimum version of the compiler that can process this partial declaration.
	MinVersion string

	// Version number of the Angular compiler that was used to compile this declaration. The linker
	// will be able to detect which version a library is using and interpret its metadata accordingly.
	Version string

	// A reference to the `@angular/core` ES module, which allows access
	// to all Angular exports, including Ivy instructions.
	NgImport output.OutputExpression

	// Reference to the decorated class, which is subject to this partial declaration.
	Type output.OutputExpression
}

// LegacyInputPartialMapping represents legacy input partial mapping
// TODO(legacy-partial-output-inputs): Remove in v18.
// https://github.com/angular/angular/blob/d4b423690210872b5c32a322a6090beda30b05a3/packages/core/src/compiler/compiler_facade_interface.ts#L197-L199
type LegacyInputPartialMapping interface{}

// LegacyInputPartialMappingString is a string type for legacy input mapping
type LegacyInputPartialMappingString string

// LegacyInputPartialMappingTuple is a tuple type for legacy input mapping
type LegacyInputPartialMappingTuple struct {
	BindingPropertyName string
	ClassPropertyName   string
	TransformFunction   *output.OutputExpression
}

// R3DeclareInputMetadata represents the new input metadata format
type R3DeclareInputMetadata struct {
	ClassPropertyName string
	PublicName        string
	IsSignal          bool
	IsRequired        bool
	TransformFunction *output.OutputExpression
}

// R3DeclareDirectiveHostMetadata represents host bindings metadata
type R3DeclareDirectiveHostMetadata struct {
	// A mapping of attribute names to their value expression.
	Attributes map[string]output.OutputExpression

	// A mapping of event names to their unparsed event handler expression.
	Listeners map[string]string

	// A mapping of bound properties to their unparsed binding expression.
	Properties map[string]string

	// The value of the class attribute, if present. This is stored outside of `attributes` as its
	// string value must be known statically.
	ClassAttribute *string

	// The value of the style attribute, if present. This is stored outside of `attributes` as its
	// string value must be known statically.
	StyleAttribute *string
}

// R3DeclareHostDirectiveMetadata describes the shape of the object literal that can be
// passed in as a part of the `hostDirectives` array.
type R3DeclareHostDirectiveMetadata struct {
	Directive output.OutputExpression
	Inputs    []string
	Outputs   []string
}

// R3DeclareDirectiveMetadata describes the shape of the object that the `ɵɵngDeclareDirective()` function accepts.
type R3DeclareDirectiveMetadata struct {
	R3PartialDeclaration

	// Unparsed selector of the directive.
	Selector *string

	// A mapping of inputs from class property names to binding property names, or to a tuple of
	// binding property name and class property name if the names are different.
	Inputs map[string]interface{} // R3DeclareInputMetadata | LegacyInputPartialMapping

	// A mapping of outputs from class property names to binding property names.
	Outputs map[string]string

	// Information about host bindings present on the component.
	Host *R3DeclareDirectiveHostMetadata

	// Information about the content queries made by the directive.
	Queries []R3DeclareQueryMetadata

	// Information about the view queries made by the directive.
	ViewQueries []R3DeclareQueryMetadata

	// The list of providers provided by the directive.
	Providers *output.OutputExpression

	// The names by which the directive is exported.
	ExportAs []string

	// Whether the directive has an inheritance clause. Defaults to false.
	UsesInheritance *bool

	// Whether the directive implements the `ngOnChanges` hook. Defaults to false.
	UsesOnChanges *bool

	// Whether the directive is standalone. Defaults to false.
	IsStandalone *bool

	// Whether the directive is a signal-based directive. Defaults to false.
	IsSignal *bool

	// Additional directives applied to the directive host.
	HostDirectives []R3DeclareHostDirectiveMetadata
}

// R3DeclareComponentMetadata describes the shape of the object that the `ɵɵngDeclareComponent()` function accepts.
type R3DeclareComponentMetadata struct {
	R3DeclareDirectiveMetadata

	// The component's unparsed template string as opaque expression. The template is represented
	// using either a string literal or template literal without substitutions, but its value is
	// not read directly. Instead, the template parser is given the full source file's text and
	// the range of this expression to parse directly from source.
	Template output.OutputExpression

	// Whether the template was inline (using `template`) or external (using `templateUrl`).
	// Defaults to false.
	IsInline *bool

	// CSS from inline styles and included styleUrls.
	Styles []string

	// List of components which matched in the template, including sufficient
	// metadata for each directive to attribute bindings and references within
	// the template to each directive specifically, if the runtime instructions
	// support this.
	Components []R3DeclareDirectiveDependencyMetadata

	// List of directives which matched in the template, including sufficient
	// metadata for each directive to attribute bindings and references within
	// the template to each directive specifically, if the runtime instructions
	// support this.
	Directives []R3DeclareDirectiveDependencyMetadata

	// List of dependencies which matched in the template, including sufficient
	// metadata for each directive/pipe to attribute bindings and references within
	// the template to each directive specifically, if the runtime instructions
	// support this.
	Dependencies []R3DeclareTemplateDependencyMetadata

	// List of defer block dependency functions, ordered by the appearance
	// of the corresponding deferred block in the template.
	DeferBlockDependencies []output.OutputExpression

	// A map of pipe names to an expression referencing the pipe type (possibly a forward reference
	// wrapped in a `forwardRef` invocation) which are used in the template.
	Pipes map[string]interface{} // output.OutputExpression | func() output.OutputExpression

	// The list of view providers defined in the component.
	ViewProviders *output.OutputExpression

	// A collection of animation triggers that will be used in the component template.
	Animations *output.OutputExpression

	// Strategy used for detecting changes in the component.
	// Defaults to `ChangeDetectionStrategy.Default`.
	ChangeDetection *core.ChangeDetectionStrategy

	// An encapsulation policy for the component's styling.
	// Defaults to `ViewEncapsulation.Emulated`.
	Encapsulation *core.ViewEncapsulation

	// Whether whitespace in the template should be preserved. Defaults to false.
	PreserveWhitespaces *bool
}

// R3DeclareTemplateDependencyMetadata is a union type for template dependencies
type R3DeclareTemplateDependencyMetadata interface {
	GetKind() string
}

// R3DeclareDirectiveDependencyMetadata represents directive/component dependency metadata
type R3DeclareDirectiveDependencyMetadata struct {
	Kind string // "directive" | "component"

	// Selector of the directive.
	Selector string

	// Reference to the directive class (possibly a forward reference wrapped in a `forwardRef`
	// invocation).
	Type interface{} // output.OutputExpression | func() output.OutputExpression

	// Property names of the directive's inputs.
	Inputs []string

	// Event names of the directive's outputs.
	Outputs []string

	// Names by which this directive exports itself for references.
	ExportAs []string
}

// GetKind returns the kind of this dependency
func (d *R3DeclareDirectiveDependencyMetadata) GetKind() string {
	return d.Kind
}

// R3DeclarePipeDependencyMetadata represents pipe dependency metadata
type R3DeclarePipeDependencyMetadata struct {
	Kind string // "pipe"

	Name string

	// Reference to the pipe class (possibly a forward reference wrapped in a `forwardRef`
	// invocation).
	Type interface{} // output.OutputExpression | func() output.OutputExpression
}

// GetKind returns the kind of this dependency
func (d *R3DeclarePipeDependencyMetadata) GetKind() string {
	return d.Kind
}

// R3DeclareNgModuleDependencyMetadata represents NgModule dependency metadata
type R3DeclareNgModuleDependencyMetadata struct {
	Kind string // "ngmodule"

	// Reference to the NgModule class (possibly a forward reference wrapped in a `forwardRef`
	// invocation).
	Type interface{} // output.OutputExpression | func() output.OutputExpression
}

// GetKind returns the kind of this dependency
func (d *R3DeclareNgModuleDependencyMetadata) GetKind() string {
	return d.Kind
}

// R3DeclareQueryMetadata represents query metadata
type R3DeclareQueryMetadata struct {
	// Name of the property on the class to update with query results.
	PropertyName string

	// Whether to read only the first matching result, or an array of results. Defaults to false.
	First *bool

	// Either an expression representing a type (possibly wrapped in a `forwardRef()`) or
	// `InjectionToken` for the query predicate, or a set of string selectors.
	Predicate interface{} // output.OutputExpression | []string

	// Whether to include only direct children or all descendants. Defaults to false.
	Descendants *bool

	// True to only fire changes if there are underlying changes to the query.
	EmitDistinctChangesOnly *bool

	// An expression representing a type to read from each matched node, or null if the default value
	// for a given node is to be returned.
	Read *output.OutputExpression

	// Whether or not this query should collect only static results. Defaults to false.
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
	Static *bool

	// Whether the query is signal-based.
	IsSignal bool
}

// R3DeclareNgModuleMetadata describes the shape of the objects that the `ɵɵngDeclareNgModule()` accepts.
type R3DeclareNgModuleMetadata struct {
	R3PartialDeclaration

	// An array of expressions representing the bootstrap components specified by the module.
	Bootstrap []output.OutputExpression

	// An array of expressions representing the directives and pipes declared by the module.
	Declarations []output.OutputExpression

	// An array of expressions representing the imports of the module.
	Imports []output.OutputExpression

	// An array of expressions representing the exports of the module.
	Exports []output.OutputExpression

	// The set of schemas that declare elements to be allowed in the NgModule.
	Schemas []output.OutputExpression

	// Unique ID or expression representing the unique ID of an NgModule.
	ID *output.OutputExpression
}

// R3DeclareInjectorMetadata describes the shape of the objects that the `ɵɵngDeclareInjector()` accepts.
type R3DeclareInjectorMetadata struct {
	R3PartialDeclaration

	// The list of providers provided by the injector.
	Providers *output.OutputExpression

	// The list of imports into the injector.
	Imports []output.OutputExpression
}

// R3DeclarePipeMetadata describes the shape of the object that the `ɵɵngDeclarePipe()` function accepts.
//
// This interface serves primarily as documentation, as conformance to this interface is not
// enforced during linking.
type R3DeclarePipeMetadata struct {
	R3PartialDeclaration

	// The name to use in templates to refer to this pipe.
	Name string

	// Whether this pipe is "pure".
	//
	// A pure pipe's `transform()` method is only invoked when its input arguments change.
	//
	// Default: true.
	Pure *bool

	// Whether the pipe is standalone.
	//
	// Default: false.
	IsStandalone *bool
}

// R3DeclareFactoryMetadata describes the shape of the object that the `ɵɵngDeclareFactory()` function accepts.
//
// This interface serves primarily as documentation, as conformance to this interface is not
// enforced during linking.
type R3DeclareFactoryMetadata struct {
	R3PartialDeclaration

	// A collection of dependencies that this factory relies upon.
	//
	// If this is `null`, then the type's constructor is nonexistent and will be inherited from an
	// ancestor of the type.
	//
	// If this is `'invalid'`, then one or more of the parameters wasn't resolvable and any attempt to
	// use these deps will result in a runtime error.
	Deps interface{} // []R3DeclareDependencyMetadata | "invalid" | nil

	// Type of the target being created by the factory.
	Target facade.FactoryTarget
}

// R3DeclareInjectableMetadata describes the shape of the object that the `ɵɵngDeclareInjectable()` function accepts.
//
// This interface serves primarily as documentation, as conformance to this interface is not
// enforced during linking.
type R3DeclareInjectableMetadata struct {
	R3PartialDeclaration

	// If provided, specifies that the declared injectable belongs to a particular injector:
	// - `InjectorType` such as `NgModule`,
	// - `'root'` the root injector
	// - `'any'` all injectors.
	// If not provided, then it does not belong to any injector. Must be explicitly listed in the
	// providers of an injector.
	ProvidedIn *output.OutputExpression

	// If provided, an expression that evaluates to a class to use when creating an instance of this
	// injectable.
	UseClass *output.OutputExpression

	// If provided, an expression that evaluates to a function to use when creating an instance of
	// this injectable.
	UseFactory *output.OutputExpression

	// If provided, an expression that evaluates to a token of another injectable that this injectable
	// aliases.
	UseExisting *output.OutputExpression

	// If provided, an expression that evaluates to the value of the instance of this injectable.
	UseValue *output.OutputExpression

	// An array of dependencies to support instantiating this injectable via `useClass` or
	// `useFactory`.
	Deps []R3DeclareDependencyMetadata
}

// R3DeclareDependencyMetadata indicates how a dependency should be injected into a factory.
type R3DeclareDependencyMetadata struct {
	// An expression representing the token or value to be injected, or `null` if the dependency is
	// not valid.
	//
	// If this dependency is due to the `@Attribute()` decorator, then this is an expression
	// evaluating to the name of the attribute.
	Token *output.OutputExpression

	// Whether the dependency is injecting an attribute value.
	// Default: false.
	Attribute *bool

	// Whether the dependency has an @Host qualifier.
	// Default: false,
	Host *bool

	// Whether the dependency has an @Optional qualifier.
	// Default: false,
	Optional *bool

	// Whether the dependency has an @Self qualifier.
	// Default: false,
	Self *bool

	// Whether the dependency has an @SkipSelf qualifier.
	// Default: false,
	SkipSelf *bool
}

// R3DeclareClassMetadata describes the shape of the object that the `ɵɵngDeclareClassMetadata()` function accepts.
//
// This interface serves primarily as documentation, as conformance to this interface is not
// enforced during linking.
type R3DeclareClassMetadata struct {
	R3PartialDeclaration

	// The Angular decorators of the class.
	Decorators output.OutputExpression

	// Optionally specifies the constructor parameters, their types and the Angular decorators of each
	// parameter. This property is omitted if the class does not have a constructor.
	CtorParameters *output.OutputExpression

	// Optionally specifies the Angular decorators applied to the class properties. This property is
	// omitted if no properties have any decorators.
	PropDecorators *output.OutputExpression
}

// R3DeclareClassMetadataAsync describes the shape of the object that the `ɵɵngDeclareClassMetadataAsync()` function accepts.
//
// This interface serves primarily as documentation, as conformance to this interface is not
// enforced during linking.
type R3DeclareClassMetadataAsync struct {
	R3PartialDeclaration

	// Function that loads the deferred dependencies associated with the component.
	ResolveDeferredDeps output.OutputExpression

	// Function that, when invoked with the resolved deferred
	// dependencies, will return the class metadata.
	ResolveMetadata output.OutputExpression
}

