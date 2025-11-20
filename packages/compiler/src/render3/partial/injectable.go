package partial

import (
	"ngc-go/packages/compiler/src/output"
	"ngc-go/packages/compiler/src/render3"
	r3_identifiers "ngc-go/packages/compiler/src/render3/r3_identifiers"
	"ngc-go/packages/compiler/src/render3/view"
)

// MINIMUM_PARTIAL_LINKER_VERSION_INJECTABLE is the minimum version of the compiler that can process this partial declaration.
// Every time we make a breaking change to the declaration interface or partial-linker behavior, we
// must update this constant to prevent old partial-linkers from incorrectly processing the
// declaration.
//
// Do not include any prerelease in these versions as they are ignored.
const MINIMUM_PARTIAL_LINKER_VERSION_INJECTABLE = "12.0.0"

// R3InjectableMetadata contains metadata for an injectable
type R3InjectableMetadata struct {
	// Name of the injectable type
	Name string

	// An expression representing a reference to the injectable itself
	Type render3.R3Reference

	// Number of generic type parameters of the type itself
	TypeArgumentCount int

	// If provided, specifies that the declared injectable belongs to a particular injector
	ProvidedIn render3.MaybeForwardRefExpression

	// If provided, an expression that evaluates to a class to use when creating an instance
	UseClass *render3.MaybeForwardRefExpression

	// If provided, an expression that evaluates to a function to use when creating an instance
	UseFactory *output.OutputExpression

	// If provided, an expression that evaluates to a token of another injectable that this injectable aliases
	UseExisting *render3.MaybeForwardRefExpression

	// If provided, an expression that evaluates to the value of the instance of this injectable
	UseValue *render3.MaybeForwardRefExpression

	// An array of dependencies to support instantiating this injectable via useClass or useFactory
	Deps *[]render3.R3DependencyMetadata
}

// CompileDeclareInjectableFromMetadata compiles a Injectable declaration defined by the `R3InjectableMetadata`.
func CompileDeclareInjectableFromMetadata(
	meta R3InjectableMetadata,
) render3.R3CompiledExpression {
	definitionMap := CreateInjectableDefinitionMap(meta)

	expression := output.NewInvokeFunctionExpr(
		output.NewExternalExpr(r3_identifiers.DeclareInjectable, nil, nil, nil),
		[]output.OutputExpression{definitionMap.ToLiteralMap()},
		nil,
		nil,
		false,
	)
	typ := createInjectableType(meta)

	return render3.R3CompiledExpression{
		Expression: expression,
		Type:       typ,
		Statements: []output.OutputStatement{},
	}
}

// CreateInjectableDefinitionMap gathers the declaration fields for a Injectable into a `DefinitionMap`.
func CreateInjectableDefinitionMap(
	meta R3InjectableMetadata,
) *view.DefinitionMap {
	definitionMap := view.NewDefinitionMap()

	definitionMap.Set("minVersion", output.NewLiteralExpr(MINIMUM_PARTIAL_LINKER_VERSION_INJECTABLE, output.InferredType, nil))
	definitionMap.Set("version", output.NewLiteralExpr("0.0.0-PLACEHOLDER", output.InferredType, nil))
	definitionMap.Set("ngImport", output.NewExternalExpr(r3_identifiers.Core, nil, nil, nil))
	definitionMap.Set("type", meta.Type.Value)

	// Only generate providedIn property if it has a non-null value
	if meta.ProvidedIn.Expression != nil {
		providedIn := render3.ConvertFromMaybeForwardRefExpression(meta.ProvidedIn)
		if literalExpr, ok := providedIn.(*output.LiteralExpr); ok {
			if literalExpr.Value != nil {
				definitionMap.Set("providedIn", providedIn)
			}
		} else {
			definitionMap.Set("providedIn", providedIn)
		}
	}

	if meta.UseClass != nil {
		definitionMap.Set("useClass", render3.ConvertFromMaybeForwardRefExpression(*meta.UseClass))
	}
	if meta.UseExisting != nil {
		definitionMap.Set("useExisting", render3.ConvertFromMaybeForwardRefExpression(*meta.UseExisting))
	}
	if meta.UseValue != nil {
		definitionMap.Set("useValue", render3.ConvertFromMaybeForwardRefExpression(*meta.UseValue))
	}
	// Factories do not contain `ForwardRef`s since any types are already wrapped in a function call
	// so the types will not be eagerly evaluated. Therefore we do not need to process this expression
	// with `convertFromProviderExpression()`.
	if meta.UseFactory != nil {
		definitionMap.Set("useFactory", *meta.UseFactory)
	}

	if meta.Deps != nil {
		deps := make([]output.OutputExpression, len(*meta.Deps))
		for i, dep := range *meta.Deps {
			deps[i] = CompileDependency(dep)
		}
		definitionMap.Set("deps", output.NewLiteralArrayExpr(deps, nil, nil))
	}

	return definitionMap
}

// createInjectableType creates the injectable type
// This is a placeholder - the actual implementation should be in injectable_compiler_2
func createInjectableType(_ R3InjectableMetadata) output.Type {
	return output.NewExpressionType(
		output.NewExternalExpr(r3_identifiers.InjectableDeclaration, nil, nil, nil),
		output.TypeModifierNone,
		nil,
	)
}
