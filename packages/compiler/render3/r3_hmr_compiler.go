package render3

import (
	"net/url"
	"ngc-go/packages/compiler/output"
	"ngc-go/packages/compiler/render3/r3_identifiers"
)

// R3HmrMetadata contains metadata necessary to compile HMR-related code call
type R3HmrMetadata struct {
	// Component class for which HMR is being enabled
	Type output.OutputExpression

	// Name of the component class
	ClassName string

	// File path of the component class
	FilePath string

	// Namespace dependencies
	// When the compiler generates new imports, they get produced as namespace imports
	// (e.g. import * as i0 from '@angular/core'). These namespaces have to be captured and passed
	// along to the update callback.
	NamespaceDependencies []R3HmrNamespaceDependency

	// Local dependencies
	// HMR update functions cannot contain imports so any locals the generated code depends on
	// (e.g. references to imports within the same file or imported symbols) have to be passed in
	// as function parameters. This array contains the names and runtime representation of the locals.
	LocalDependencies []R3HmrLocalDependency
}

// R3HmrNamespaceDependency represents an HMR dependency on a namespace import
type R3HmrNamespaceDependency struct {
	// Module name of the import
	ModuleName string

	// Name under which to refer to the namespace inside
	// HMR-related code. Must be a valid JS identifier.
	AssignedName string
}

// R3HmrLocalDependency represents a local dependency
type R3HmrLocalDependency struct {
	// Name of the local dependency
	Name string

	// Runtime representation of the local dependency
	RuntimeRepresentation output.OutputExpression
}

// CompileHmrInitializer compiles the expression that initializes HMR for a class
func CompileHmrInitializer(meta R3HmrMetadata) output.OutputExpression {
	moduleName := "m"
	dataName := "d"
	timestampName := "t"
	idName := "id"
	importCallbackName := meta.ClassName + "_HmrLoad"
	namespaces := make([]output.OutputExpression, len(meta.NamespaceDependencies))
	for i, dep := range meta.NamespaceDependencies {
		namespaces[i] = output.NewExternalExpr(
			&output.ExternalReference{ModuleName: &dep.ModuleName, Name: nil},
			nil,
			nil,
			nil,
		)
	}

	// m.default
	moduleVar := output.NewReadVarExpr(moduleName, nil, nil)
	defaultRead := output.NewReadPropExpr(moduleVar, "default", nil, nil)

	// ɵɵreplaceMetadata(Comp, m.default, [...namespaces], [...locals], import.meta, id);
	replaceCall := output.NewInvokeFunctionExpr(
		output.NewExternalExpr(r3_identifiers.ReplaceMetadata, nil, nil, nil),
		[]output.OutputExpression{
			meta.Type,
			defaultRead,
			output.NewLiteralArrayExpr(namespaces, nil, nil),
			output.NewLiteralArrayExpr(
				func() []output.OutputExpression {
					locals := make([]output.OutputExpression, len(meta.LocalDependencies))
					for i, l := range meta.LocalDependencies {
						locals[i] = l.RuntimeRepresentation
					}
					return locals
				}(),
				nil,
				nil,
			),
			output.NewReadPropExpr(
				output.NewReadVarExpr("import", nil, nil),
				"meta",
				nil,
				nil,
			),
			output.NewReadVarExpr(idName, nil, nil),
		},
		nil,
		nil,
		false,
	)

	// (m) => m.default && ɵɵreplaceMetadata(...)
	replaceCallback := output.NewArrowFunctionExpr(
		[]*output.FnParam{output.NewFnParam(moduleName, output.DynamicType)},
		output.NewBinaryOperatorExpr(
			output.BinaryOperatorAnd,
			defaultRead,
			replaceCall,
			nil,
			nil,
		),
		output.InferredType,
		nil,
	)

	// getReplaceMetadataURL(id, timestamp, import.meta.url)
	urlExpr := output.NewInvokeFunctionExpr(
		output.NewExternalExpr(r3_identifiers.GetReplaceMetadataURL, nil, nil, nil),
		[]output.OutputExpression{
			output.NewReadVarExpr(idName, nil, nil),
			output.NewReadVarExpr(timestampName, nil, nil),
			output.NewReadPropExpr(
				output.NewReadPropExpr(
					output.NewReadVarExpr("import", nil, nil),
					"meta",
					nil,
					nil,
				),
				"url",
				nil,
				nil,
			),
		},
		nil,
		nil,
		false,
	)

	// function Cmp_HmrLoad(t) {
	//   import(/* @vite-ignore */ url).then((m) => m.default && replaceMetadata(...));
	// }
	viteIgnore := "@vite-ignore"
	importCallback := output.NewDeclareFunctionStmt(
		importCallbackName,
		[]*output.FnParam{output.NewFnParam(timestampName, output.DynamicType)},
		[]output.OutputStatement{
			output.NewExpressionStatement(
				output.NewInvokeFunctionExpr(
					output.NewReadPropExpr(
						output.NewDynamicImportExpr(urlExpr, nil, &viteIgnore),
						"then",
						nil,
						nil,
					),
					[]output.OutputExpression{replaceCallback},
					nil,
					nil,
					false,
				),
				nil,
				nil,
			),
		},
		nil,
		output.StmtModifierFinal,
		nil,
		nil,
	)

	// (d) => d.id === id && Cmp_HmrLoad(d.timestamp)
	dataVar := output.NewReadVarExpr(dataName, nil, nil)
	updateCallback := output.NewArrowFunctionExpr(
		[]*output.FnParam{output.NewFnParam(dataName, output.DynamicType)},
		output.NewBinaryOperatorExpr(
			output.BinaryOperatorAnd,
			output.NewBinaryOperatorExpr(
				output.BinaryOperatorIdentical,
				output.NewReadPropExpr(dataVar, "id", nil, nil),
				output.NewReadVarExpr(idName, nil, nil),
				nil,
				nil,
			),
			output.NewInvokeFunctionExpr(
				output.NewReadVarExpr(importCallbackName, nil, nil),
				[]output.OutputExpression{
					output.NewReadPropExpr(dataVar, "timestamp", nil, nil),
				},
				nil,
				nil,
				false,
			),
			nil,
			nil,
		),
		output.InferredType,
		nil,
	)

	// Cmp_HmrLoad(Date.now());
	// Initial call to kick off the loading in order to avoid edge cases with components
	// coming from lazy chunks that change before the chunk has loaded.
	initialCall := output.NewInvokeFunctionExpr(
		output.NewReadVarExpr(importCallbackName, nil, nil),
		[]output.OutputExpression{
			output.NewInvokeFunctionExpr(
				output.NewReadPropExpr(
					output.NewReadVarExpr("Date", nil, nil),
					"now",
					nil,
					nil,
				),
				[]output.OutputExpression{},
				nil,
				nil,
				false,
			),
		},
		nil,
		nil,
		false,
	)

	// import.meta.hot
	hotRead := output.NewReadPropExpr(
		output.NewReadPropExpr(
			output.NewReadVarExpr("import", nil, nil),
			"meta",
			nil,
			nil,
		),
		"hot",
		nil,
		nil,
	)

	// import.meta.hot.on('angular:component-update', () => ...);
	hotListener := output.NewInvokeFunctionExpr(
		output.NewReadPropExpr(hotRead, "on", nil, nil),
		[]output.OutputExpression{
			output.NewLiteralExpr("angular:component-update", nil, nil),
			updateCallback,
		},
		nil,
		nil,
		false,
	)

	// Encode the ID
	idValue := url.QueryEscape(meta.FilePath + "@" + meta.ClassName)

	return output.NewInvokeFunctionExpr(
		output.NewArrowFunctionExpr(
			[]*output.FnParam{},
			[]output.OutputStatement{
				// const id = <id>;
				output.NewDeclareVarStmt(
					idName,
					output.NewLiteralExpr(idValue, nil, nil),
					nil,
					output.StmtModifierFinal,
					nil,
					nil,
				),
				// function Cmp_HmrLoad() {...}.
				importCallback,
				// ngDevMode && Cmp_HmrLoad(Date.now());
				output.NewExpressionStatement(
					DevOnlyGuardedExpression(initialCall),
					nil,
					nil,
				),
				// ngDevMode && import.meta.hot && import.meta.hot.on(...)
				output.NewExpressionStatement(
					DevOnlyGuardedExpression(
						output.NewBinaryOperatorExpr(
							output.BinaryOperatorAnd,
							hotRead,
							hotListener,
							nil,
							nil,
						),
					),
					nil,
					nil,
				),
			},
			output.InferredType,
			nil,
		),
		[]output.OutputExpression{},
		nil,
		nil,
		false,
	)
}

// R3HmrDefinition represents a compiled definition for a class
type R3HmrDefinition struct {
	Name        string
	Initializer output.OutputExpression
	Statements  []output.OutputStatement
}

// CompileHmrUpdateCallback compiles the HMR update callback for a class
func CompileHmrUpdateCallback(
	definitions []R3HmrDefinition,
	constantStatements []output.OutputStatement,
	meta R3HmrMetadata,
) *output.DeclareFunctionStmt {
	namespaces := "ɵɵnamespaces"
	params := []*output.FnParam{
		output.NewFnParam(meta.ClassName, output.DynamicType),
		output.NewFnParam(namespaces, output.DynamicType),
	}
	body := []output.OutputStatement{}

	// Add local dependencies as parameters
	for _, local := range meta.LocalDependencies {
		params = append(params, output.NewFnParam(local.Name, output.DynamicType))
	}

	// Declare variables that read out the individual namespaces
	for i, dep := range meta.NamespaceDependencies {
		body = append(body, output.NewDeclareVarStmt(
			dep.AssignedName,
			output.NewReadKeyExpr(
				output.NewReadVarExpr(namespaces, nil, nil),
				output.NewLiteralExpr(i, nil, nil),
				nil,
				nil,
			),
			output.DynamicType,
			output.StmtModifierFinal,
			nil,
			nil,
		))
	}

	body = append(body, constantStatements...)

	for _, field := range definitions {
		if field.Initializer != nil {
			// className.fieldName = initializer
			body = append(body, output.NewExpressionStatement(
				output.NewBinaryOperatorExpr(
					output.BinaryOperatorAssign,
					output.NewReadPropExpr(
						output.NewReadVarExpr(meta.ClassName, nil, nil),
						field.Name,
						nil,
						nil,
					),
					field.Initializer,
					nil,
					nil,
				),
				nil,
				nil,
			))

			body = append(body, field.Statements...)
		}
	}

	return output.NewDeclareFunctionStmt(
		meta.ClassName+"_UpdateMetadata",
		params,
		body,
		nil,
		output.StmtModifierFinal,
		nil,
		nil,
	)
}
