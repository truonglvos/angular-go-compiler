package partial

import (
	"ngc-go/packages/compiler/src/facade"
	"ngc-go/packages/compiler/src/output"
	"ngc-go/packages/compiler/src/render3"
	r3_identifiers "ngc-go/packages/compiler/src/render3/r3_identifiers"
	"ngc-go/packages/compiler/src/render3/view"
)

// MINIMUM_PARTIAL_LINKER_VERSION_FACTORY is the minimum version of the compiler that can process this partial declaration.
// Every time we make a breaking change to the declaration interface or partial-linker behavior, we
// must update this constant to prevent old partial-linkers from incorrectly processing the
// declaration.
//
// Do not include any prerelease in these versions as they are ignored.
const MINIMUM_PARTIAL_LINKER_VERSION_FACTORY = "12.0.0"

// CompileDeclareFactoryFunction compiles a factory function declaration
func CompileDeclareFactoryFunction(meta render3.R3ConstructorFactoryMetadata) render3.R3CompiledExpression {
	definitionMap := view.NewDefinitionMap()
	definitionMap.Set("minVersion", output.NewLiteralExpr(MINIMUM_PARTIAL_LINKER_VERSION_FACTORY, output.InferredType, nil))
	definitionMap.Set("version", output.NewLiteralExpr("0.0.0-PLACEHOLDER", output.InferredType, nil))
	definitionMap.Set("ngImport", output.NewExternalExpr(r3_identifiers.Core, nil, nil, nil))
	definitionMap.Set("type", meta.Type.Value)
	definitionMap.Set("deps", CompileDependencies(meta.Deps))
	targetName := getFactoryTargetName(meta.Target)
	definitionMap.Set("target", output.NewReadPropExpr(
		output.NewExternalExpr(r3_identifiers.FactoryTarget, nil, nil, nil),
		targetName,
		output.InferredType,
		nil,
	))

	return render3.R3CompiledExpression{
		Expression: output.NewInvokeFunctionExpr(
			output.NewExternalExpr(r3_identifiers.DeclareFactory, nil, nil, nil),
			[]output.OutputExpression{definitionMap.ToLiteralMap()},
			nil,
			nil,
			false,
		),
		Statements: []output.OutputStatement{},
		Type:       render3.CreateFactoryType(meta),
	}
}

// getFactoryTargetName returns the name of the factory target enum value
func getFactoryTargetName(target facade.FactoryTarget) string {
	switch target {
	case facade.FactoryTargetDirective:
		return "Directive"
	case facade.FactoryTargetComponent:
		return "Component"
	case facade.FactoryTargetInjectable:
		return "Injectable"
	case facade.FactoryTargetPipe:
		return "Pipe"
	case facade.FactoryTargetNgModule:
		return "NgModule"
	default:
		return "Directive"
	}
}
