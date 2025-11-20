package partial

import (
	"ngc-go/packages/compiler/src/output"
	"ngc-go/packages/compiler/src/render3"
	r3_identifiers "ngc-go/packages/compiler/src/render3/r3_identifiers"
	"ngc-go/packages/compiler/src/render3/view"
)

// MINIMUM_PARTIAL_LINKER_VERSION is the minimum version of the compiler that can process this partial declaration.
// Every time we make a breaking change to the declaration interface or partial-linker behavior, we
// must update this constant to prevent old partial-linkers from incorrectly processing the
// declaration.
//
// Do not include any prerelease in these versions as they are ignored.
const MINIMUM_PARTIAL_LINKER_VERSION_PIPE = "14.0.0"

// CompileDeclarePipeFromMetadata compiles a Pipe declaration defined by the `R3PipeMetadata`.
func CompileDeclarePipeFromMetadata(meta render3.R3PipeMetadata) render3.R3CompiledExpression {
	definitionMap := CreatePipeDefinitionMap(meta)

	expression := output.NewInvokeFunctionExpr(
		output.NewExternalExpr(r3_identifiers.DeclarePipe, nil, nil, nil),
		[]output.OutputExpression{definitionMap.ToLiteralMap()},
		nil,
		nil,
		false,
	)
	typ := render3.CreatePipeType(meta)

	return render3.R3CompiledExpression{
		Expression: expression,
		Type:       typ,
		Statements: []output.OutputStatement{},
	}
}

// CreatePipeDefinitionMap gathers the declaration fields for a Pipe into a `DefinitionMap`.
func CreatePipeDefinitionMap(
	meta render3.R3PipeMetadata,
) *view.DefinitionMap {
	definitionMap := view.NewDefinitionMap()

	definitionMap.Set("minVersion", output.NewLiteralExpr(MINIMUM_PARTIAL_LINKER_VERSION_PIPE, output.InferredType, nil))
	definitionMap.Set("version", output.NewLiteralExpr("0.0.0-PLACEHOLDER", output.InferredType, nil))
	definitionMap.Set("ngImport", output.NewExternalExpr(r3_identifiers.Core, nil, nil, nil))

	// e.g. `type: MyPipe`
	definitionMap.Set("type", meta.Type.Value)

	if meta.IsStandalone {
		definitionMap.Set("isStandalone", output.NewLiteralExpr(meta.IsStandalone, output.InferredType, nil))
	}

	// e.g. `name: "myPipe"`
	pipeName := meta.Name
	if meta.PipeName != nil {
		pipeName = *meta.PipeName
	}
	definitionMap.Set("name", output.NewLiteralExpr(pipeName, output.InferredType, nil))

	if !meta.Pure {
		// e.g. `pure: false`
		definitionMap.Set("pure", output.NewLiteralExpr(meta.Pure, output.InferredType, nil))
	}

	return definitionMap
}
