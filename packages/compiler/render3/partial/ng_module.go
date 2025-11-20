package partial

import (
	"errors"

	"ngc-go/packages/compiler/output"
	"ngc-go/packages/compiler/render3"
	r3_identifiers "ngc-go/packages/compiler/render3/r3_identifiers"
	render3_module_compiler "ngc-go/packages/compiler/render3/r3_module_compiler"
	"ngc-go/packages/compiler/render3/view"
)

// MINIMUM_PARTIAL_LINKER_VERSION_NGMODULE is the minimum version of the compiler that can process this partial declaration.
// Every time we make a breaking change to the declaration interface or partial-linker behavior, we
// must update this constant to prevent old partial-linkers from incorrectly processing the
// declaration.
//
// Do not include any prerelease in these versions as they are ignored.
const MINIMUM_PARTIAL_LINKER_VERSION_NGMODULE = "14.0.0"

// CompileDeclareNgModuleFromMetadata compiles an NgModule declaration defined by the `R3NgModuleMetadata`.
func CompileDeclareNgModuleFromMetadata(meta render3_module_compiler.R3NgModuleMetadata) render3.R3CompiledExpression {
	definitionMap := createNgModuleDefinitionMap(meta)

	expression := output.NewInvokeFunctionExpr(
		output.NewExternalExpr(r3_identifiers.DeclareNgModule, nil, nil, nil),
		[]output.OutputExpression{definitionMap.ToLiteralMap()},
		nil,
		nil,
		false,
	)
	typ := render3_module_compiler.CreateNgModuleType(meta)

	return render3.R3CompiledExpression{
		Expression: expression,
		Type:       typ,
		Statements: []output.OutputStatement{},
	}
}

// createNgModuleDefinitionMap gathers the declaration fields for an NgModule into a `DefinitionMap`.
func createNgModuleDefinitionMap(
	meta render3_module_compiler.R3NgModuleMetadata,
) *view.DefinitionMap {
	definitionMap := view.NewDefinitionMap()
	common := meta.GetCommon()

	if common.Kind == render3_module_compiler.R3NgModuleMetadataKindLocal {
		panic(errors.New(
			"Invalid path! Local compilation mode should not get into the partial compilation path",
		))
	}

	definitionMap.Set("minVersion", output.NewLiteralExpr(MINIMUM_PARTIAL_LINKER_VERSION_NGMODULE, output.InferredType, nil))
	definitionMap.Set("version", output.NewLiteralExpr("0.0.0-PLACEHOLDER", output.InferredType, nil))
	definitionMap.Set("ngImport", output.NewExternalExpr(r3_identifiers.Core, nil, nil, nil))
	definitionMap.Set("type", common.Type.Value)

	// We only generate the keys in the metadata if the arrays contain values.

	// We must wrap the arrays inside a function if any of the values are a forward reference to a
	// not-yet-declared class. This is to support JIT execution of the `ɵɵngDeclareNgModule()` call.
	// In the linker these wrappers are stripped and then reapplied for the `ɵɵdefineNgModule()` call.

	if globalMeta, ok := meta.(*render3_module_compiler.R3NgModuleMetadataGlobal); ok {
		if len(globalMeta.Bootstrap) > 0 {
			definitionMap.Set("bootstrap", render3.RefsToArray(globalMeta.Bootstrap, globalMeta.ContainsForwardDecls))
		}

		if len(globalMeta.Declarations) > 0 {
			definitionMap.Set("declarations", render3.RefsToArray(globalMeta.Declarations, globalMeta.ContainsForwardDecls))
		}

		if len(globalMeta.Imports) > 0 {
			definitionMap.Set("imports", render3.RefsToArray(globalMeta.Imports, globalMeta.ContainsForwardDecls))
		}

		if len(globalMeta.Exports) > 0 {
			definitionMap.Set("exports", render3.RefsToArray(globalMeta.Exports, globalMeta.ContainsForwardDecls))
		}
	}

	if common.Schemas != nil && len(common.Schemas) > 0 {
		schemaValues := make([]output.OutputExpression, len(common.Schemas))
		for i, ref := range common.Schemas {
			schemaValues[i] = ref.Value
		}
		definitionMap.Set("schemas", output.NewLiteralArrayExpr(schemaValues, nil, nil))
	}

	if common.ID != nil {
		definitionMap.Set("id", common.ID)
	}

	return definitionMap
}

