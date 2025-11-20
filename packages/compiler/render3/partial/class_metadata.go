package partial

import (
	"ngc-go/packages/compiler/output"
	"ngc-go/packages/compiler/render3/r3_identifiers"
	"ngc-go/packages/compiler/render3/view"
)

// R3ClassMetadata contains metadata of a class which captures the original Angular decorators of a class.
// The original decorators are preserved in the generated code to allow TestBed APIs to recompile the class
// using the original decorator with a set of overrides applied.
type R3ClassMetadata struct {
	// The class type for which the metadata is captured.
	Type output.OutputExpression

	// An expression representing the Angular decorators that were applied on the class.
	Decorators output.OutputExpression

	// An expression representing the Angular decorators applied to constructor parameters, or `null`
	// if there is no constructor.
	CtorParameters *output.OutputExpression

	// An expression representing the Angular decorators that were applied on the properties of the
	// class, or `null` if no properties have decorators.
	PropDecorators *output.OutputExpression
}

// CompileComponentMetadataAsyncResolver compiles the function that loads the dependencies for the
// entire component in `setClassMetadataAsync`.
func CompileComponentMetadataAsyncResolver(
	dependencies []view.R3DeferPerComponentDependency,
) *output.ArrowFunctionExpr {
	dynamicImports := make([]output.OutputExpression, len(dependencies))
	for i, dep := range dependencies {
		// e.g. `(m) => m.CmpA`
		var propName string
		if dep.IsDefaultImport {
			propName = "default"
		} else {
			propName = dep.SymbolName
		}
		innerFn := output.NewArrowFunctionExpr(
			[]*output.FnParam{
				output.NewFnParam("m", output.DynamicType),
			},
			output.NewReadPropExpr(
				output.NewReadVarExpr("m", output.DynamicType, nil),
				propName,
				output.DynamicType,
				nil,
			),
			output.DynamicType,
			nil,
		)

		// e.g. `import('./cmp-a').then(...)`
		dynamicImport := output.NewDynamicImportExpr(dep.ImportPath, nil, nil)
		thenProp := output.NewReadPropExpr(dynamicImport, "then", output.DynamicType, nil)
		dynamicImports[i] = output.NewInvokeFunctionExpr(
			thenProp,
			[]output.OutputExpression{innerFn},
			nil,
			nil,
			false,
		)
	}

	// e.g. `() => [ ... ];`
	return output.NewArrowFunctionExpr(
		[]*output.FnParam{},
		output.NewLiteralArrayExpr(dynamicImports, nil, nil),
		output.DynamicType,
		nil,
	)
}

// Every time we make a breaking change to the declaration interface or partial-linker behavior, we
// must update this constant to prevent old partial-linkers from incorrectly processing the
// declaration.
//
// Do not include any prerelease in these versions as they are ignored.
const MINIMUM_PARTIAL_LINKER_VERSION = "12.0.0"

// Minimum version at which deferred blocks are supported in the linker.
const MINIMUM_PARTIAL_LINKER_DEFER_SUPPORT_VERSION = "18.0.0"

// CompileDeclareClassMetadata compiles class metadata into a declare class metadata expression
func CompileDeclareClassMetadata(metadata R3ClassMetadata) output.OutputExpression {
	definitionMap := view.NewDefinitionMap()
	definitionMap.Set("minVersion", output.NewLiteralExpr(MINIMUM_PARTIAL_LINKER_VERSION, output.InferredType, nil))
	definitionMap.Set("version", output.NewLiteralExpr("0.0.0-PLACEHOLDER", output.InferredType, nil))
	definitionMap.Set("ngImport", output.NewExternalExpr(r3_identifiers.Core, nil, nil, nil))
	definitionMap.Set("type", metadata.Type)
	definitionMap.Set("decorators", metadata.Decorators)
	if metadata.CtorParameters != nil {
		definitionMap.Set("ctorParameters", *metadata.CtorParameters)
	} else {
		definitionMap.Set("ctorParameters", output.NewLiteralExpr(nil, output.InferredType, nil))
	}
	if metadata.PropDecorators != nil {
		definitionMap.Set("propDecorators", *metadata.PropDecorators)
	} else {
		definitionMap.Set("propDecorators", output.NewLiteralExpr(nil, output.InferredType, nil))
	}

	return output.NewInvokeFunctionExpr(
		output.NewExternalExpr(r3_identifiers.DeclareClassMetadata, nil, nil, nil),
		[]output.OutputExpression{definitionMap.ToLiteralMap()},
		nil,
		nil,
		false,
	)
}

// CompileComponentDeclareClassMetadata compiles component class metadata, handling deferred dependencies
func CompileComponentDeclareClassMetadata(
	metadata R3ClassMetadata,
	dependencies []view.R3DeferPerComponentDependency,
) output.OutputExpression {
	if dependencies == nil || len(dependencies) == 0 {
		return CompileDeclareClassMetadata(metadata)
	}

	definitionMap := view.NewDefinitionMap()
	callbackReturnDefinitionMap := view.NewDefinitionMap()
	callbackReturnDefinitionMap.Set("decorators", metadata.Decorators)
	if metadata.CtorParameters != nil {
		callbackReturnDefinitionMap.Set("ctorParameters", *metadata.CtorParameters)
	} else {
		callbackReturnDefinitionMap.Set("ctorParameters", output.NewLiteralExpr(nil, output.InferredType, nil))
	}
	if metadata.PropDecorators != nil {
		callbackReturnDefinitionMap.Set("propDecorators", *metadata.PropDecorators)
	} else {
		callbackReturnDefinitionMap.Set("propDecorators", output.NewLiteralExpr(nil, output.InferredType, nil))
	}

	definitionMap.Set("minVersion", output.NewLiteralExpr(MINIMUM_PARTIAL_LINKER_DEFER_SUPPORT_VERSION, output.InferredType, nil))
	definitionMap.Set("version", output.NewLiteralExpr("0.0.0-PLACEHOLDER", output.InferredType, nil))
	definitionMap.Set("ngImport", output.NewExternalExpr(r3_identifiers.Core, nil, nil, nil))
	definitionMap.Set("type", metadata.Type)
	definitionMap.Set("resolveDeferredDeps", CompileComponentMetadataAsyncResolver(dependencies))

	// Create arrow function with dependencies as parameters
	fnParams := make([]*output.FnParam, len(dependencies))
	for i, dep := range dependencies {
		fnParams[i] = output.NewFnParam(dep.SymbolName, output.DynamicType)
	}
	definitionMap.Set(
		"resolveMetadata",
		output.NewArrowFunctionExpr(
			fnParams,
			callbackReturnDefinitionMap.ToLiteralMap(),
			output.DynamicType,
			nil,
		),
	)

	return output.NewInvokeFunctionExpr(
		output.NewExternalExpr(r3_identifiers.DeclareClassMetadataAsync, nil, nil, nil),
		[]output.OutputExpression{definitionMap.ToLiteralMap()},
		nil,
		nil,
		false,
	)
}
