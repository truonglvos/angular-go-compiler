package render3_injector_compiler

import (
	"ngc-go/packages/compiler/output"
	"ngc-go/packages/compiler/render3"
	"ngc-go/packages/compiler/render3/r3_identifiers"
	"ngc-go/packages/compiler/render3/view"
)

// R3InjectorMetadata contains metadata for an injector
type R3InjectorMetadata struct {
	Name      string
	Type      render3.R3Reference
	Providers output.OutputExpression
	Imports   []output.OutputExpression
}

// CompileInjector compiles an injector definition
func CompileInjector(meta R3InjectorMetadata) render3.R3CompiledExpression {
	definitionMap := view.NewDefinitionMap()

	if meta.Providers != nil {
		definitionMap.Set("providers", meta.Providers)
	}

	if len(meta.Imports) > 0 {
		definitionMap.Set("imports", output.NewLiteralArrayExpr(meta.Imports, nil, nil))
	}

	expression := output.NewInvokeFunctionExpr(
		output.NewExternalExpr(r3_identifiers.DefineInjector, nil, nil, nil),
		[]output.OutputExpression{definitionMap.ToLiteralMap()},
		nil, // typ
		nil, // sourceSpan
		true, // pure
	)
	typ := CreateInjectorType(meta)
	return render3.R3CompiledExpression{
		Expression: expression,
		Type:       typ,
		Statements: []output.OutputStatement{},
	}
}

// CreateInjectorType creates the injector type
func CreateInjectorType(meta R3InjectorMetadata) output.Type {
	return output.NewExpressionType(
		output.NewExternalExpr(
			r3_identifiers.InjectorDeclaration,
			nil,
			[]output.Type{
				output.NewExpressionType(meta.Type.Type, output.TypeModifierNone, nil),
			},
			nil,
		),
		output.TypeModifierNone,
		nil,
	)
}
