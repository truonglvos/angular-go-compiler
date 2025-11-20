package render3

import (
	"ngc-go/packages/compiler/output"
	"ngc-go/packages/compiler/render3/r3_identifiers"
)

// R3PipeMetadata contains metadata for a pipe
type R3PipeMetadata struct {
	// Name of the pipe type
	Name string

	// An expression representing a reference to the pipe itself
	Type R3Reference

	// Number of generic type parameters of the type itself
	TypeArgumentCount int

	// Name of the pipe
	PipeName *string

	// Dependencies of the pipe's constructor
	Deps []R3DependencyMetadata

	// Whether the pipe is marked as pure
	Pure bool

	// Whether the pipe is standalone
	IsStandalone bool
}

// CompilePipeFromMetadata compiles a pipe definition from metadata
func CompilePipeFromMetadata(metadata R3PipeMetadata) R3CompiledExpression {
	definitionMapValues := []*output.LiteralMapEntry{}

	// e.g. `name: 'myPipe'`
	pipeName := metadata.Name
	if metadata.PipeName != nil {
		pipeName = *metadata.PipeName
	}
	definitionMapValues = append(definitionMapValues, output.NewLiteralMapEntry(
		"name",
		output.NewLiteralExpr(pipeName, nil, nil),
		false,
	))

	// e.g. `type: MyPipe`
	definitionMapValues = append(definitionMapValues, output.NewLiteralMapEntry(
		"type",
		metadata.Type.Value,
		false,
	))

	// e.g. `pure: true`
	definitionMapValues = append(definitionMapValues, output.NewLiteralMapEntry(
		"pure",
		output.NewLiteralExpr(metadata.Pure, nil, nil),
		false,
	))

	// Only add standalone if it's false (true is the default)
	if !metadata.IsStandalone {
		definitionMapValues = append(definitionMapValues, output.NewLiteralMapEntry(
			"standalone",
			output.NewLiteralExpr(false, nil, nil),
			false,
		))
	}

	typ := CreatePipeType(metadata)
	expression := output.NewInvokeFunctionExpr(
		output.NewExternalExpr(r3_identifiers.DefinePipe, nil, nil, nil),
		[]output.OutputExpression{output.NewLiteralMapExpr(definitionMapValues, nil, nil)},
		typ,
		nil,  // sourceSpan
		true, // pure
	)

	return R3CompiledExpression{
		Expression: expression,
		Type:       typ,
		Statements: []output.OutputStatement{},
	}
}

// CreatePipeType creates the pipe type
func CreatePipeType(metadata R3PipeMetadata) output.Type {
	// Use pipeName if provided, otherwise use name (matching TypeScript: metadata.pipeName ?? metadata.name)
	pipeName := metadata.Name
	if metadata.PipeName != nil {
		pipeName = *metadata.PipeName
	}
	pipeNameExpr := output.NewLiteralExpr(pipeName, nil, nil)

	return output.NewExpressionType(
		output.NewExternalExpr(r3_identifiers.PipeDeclaration, nil, nil, nil),
		output.TypeModifierNone,
		[]output.Type{
			TypeWithParameters(metadata.Type.Type, metadata.TypeArgumentCount),
			output.NewExpressionType(pipeNameExpr, output.TypeModifierNone, nil),
			output.NewExpressionType(output.NewLiteralExpr(metadata.IsStandalone, nil, nil), output.TypeModifierNone, nil),
		},
	)
}
