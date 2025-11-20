package partial

import (
	"ngc-go/packages/compiler/output"
	"ngc-go/packages/compiler/render3"
	r3_identifiers "ngc-go/packages/compiler/render3/r3_identifiers"
	render3_injector_compiler "ngc-go/packages/compiler/render3/r3_injector_compiler"
	"ngc-go/packages/compiler/render3/view"
)

// MINIMUM_PARTIAL_LINKER_VERSION_INJECTOR is the minimum version of the compiler that can process this partial declaration.
// Every time we make a breaking change to the declaration interface or partial-linker behavior, we
// must update this constant to prevent old partial-linkers from incorrectly processing the
// declaration.
//
// Do not include any prerelease in these versions as they are ignored.
const MINIMUM_PARTIAL_LINKER_VERSION_INJECTOR = "12.0.0"

// CompileDeclareInjectorFromMetadata compiles an Injector declaration defined by the `R3InjectorMetadata`.
func CompileDeclareInjectorFromMetadata(meta render3_injector_compiler.R3InjectorMetadata) render3.R3CompiledExpression {
	definitionMap := createInjectorDefinitionMap(meta)

	expression := output.NewInvokeFunctionExpr(
		output.NewExternalExpr(r3_identifiers.DeclareInjector, nil, nil, nil),
		[]output.OutputExpression{definitionMap.ToLiteralMap()},
		nil,
		nil,
		false,
	)
	typ := render3_injector_compiler.CreateInjectorType(meta)

	return render3.R3CompiledExpression{
		Expression: expression,
		Type:       typ,
		Statements: []output.OutputStatement{},
	}
}

// createInjectorDefinitionMap gathers the declaration fields for an Injector into a `DefinitionMap`.
func createInjectorDefinitionMap(
	meta render3_injector_compiler.R3InjectorMetadata,
) *view.DefinitionMap {
	definitionMap := view.NewDefinitionMap()

	definitionMap.Set("minVersion", output.NewLiteralExpr(MINIMUM_PARTIAL_LINKER_VERSION_INJECTOR, output.InferredType, nil))
	definitionMap.Set("version", output.NewLiteralExpr("0.0.0-PLACEHOLDER", output.InferredType, nil))
	definitionMap.Set("ngImport", output.NewExternalExpr(r3_identifiers.Core, nil, nil, nil))

	definitionMap.Set("type", meta.Type.Value)
	definitionMap.Set("providers", meta.Providers)
	if len(meta.Imports) > 0 {
		definitionMap.Set("imports", output.NewLiteralArrayExpr(meta.Imports, nil, nil))
	}

	return definitionMap
}

