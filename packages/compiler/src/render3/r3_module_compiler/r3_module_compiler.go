package render3_module_compiler

import (
	"ngc-go/packages/compiler/src/facade"
	"ngc-go/packages/compiler/src/output"
	"ngc-go/packages/compiler/src/render3"
	"ngc-go/packages/compiler/src/render3/r3_identifiers"
	"ngc-go/packages/compiler/src/render3/view"
)

// R3SelectorScopeMode represents how the selector scope of an NgModule should be emitted
type R3SelectorScopeMode int

const (
	// R3SelectorScopeModeInline emits the declarations inline into the module definition
	R3SelectorScopeModeInline R3SelectorScopeMode = iota
	// R3SelectorScopeModeSideEffect emits the declarations using a side effectful function call
	R3SelectorScopeModeSideEffect
	// R3SelectorScopeModeOmit doesn't generate selector scopes at all
	R3SelectorScopeModeOmit
)

// R3NgModuleMetadataKind represents the type of the NgModule metadata
type R3NgModuleMetadataKind int

const (
	// R3NgModuleMetadataKindGlobal is used for full and partial compilation modes
	R3NgModuleMetadataKindGlobal R3NgModuleMetadataKind = iota
	// R3NgModuleMetadataKindLocal is used for the local compilation mode
	R3NgModuleMetadataKindLocal
)

// R3NgModuleMetadataCommon contains common metadata for NgModule
type R3NgModuleMetadataCommon struct {
	Kind              R3NgModuleMetadataKind
	Type              render3.R3Reference
	SelectorScopeMode R3SelectorScopeMode
	Schemas           []render3.R3Reference
	ID                output.OutputExpression
}

// R3NgModuleMetadataGlobal contains metadata for NgModule in global mode
type R3NgModuleMetadataGlobal struct {
	R3NgModuleMetadataCommon
	Bootstrap              []render3.R3Reference
	Declarations           []render3.R3Reference
	PublicDeclarationTypes []output.OutputExpression
	Imports                []render3.R3Reference
	IncludeImportTypes     bool
	Exports                []render3.R3Reference
	ContainsForwardDecls   bool
}

// R3NgModuleMetadataLocal contains metadata for NgModule in local mode
type R3NgModuleMetadataLocal struct {
	R3NgModuleMetadataCommon
	BootstrapExpression    output.OutputExpression
	DeclarationsExpression output.OutputExpression
	ImportsExpression      output.OutputExpression
	ExportsExpression      output.OutputExpression
}

// R3NgModuleMetadata is a union type for NgModule metadata
type R3NgModuleMetadata interface {
	GetCommon() *R3NgModuleMetadataCommon
}

// GetCommon returns the common metadata
func (m *R3NgModuleMetadataGlobal) GetCommon() *R3NgModuleMetadataCommon {
	return &m.R3NgModuleMetadataCommon
}

// GetCommon returns the common metadata
func (m *R3NgModuleMetadataLocal) GetCommon() *R3NgModuleMetadataCommon {
	return &m.R3NgModuleMetadataCommon
}

// CompileNgModule constructs an R3NgModuleDef for the given R3NgModuleMetadata
func CompileNgModule(meta R3NgModuleMetadata) render3.R3CompiledExpression {
	statements := []output.OutputStatement{}
	definitionMap := view.NewDefinitionMap()
	common := meta.GetCommon()

	definitionMap.Set("type", common.Type.Value)

	// Handle bootstrap for global mode
	if globalMeta, ok := meta.(*R3NgModuleMetadataGlobal); ok {
		if len(globalMeta.Bootstrap) > 0 {
			definitionMap.Set("bootstrap", render3.RefsToArray(globalMeta.Bootstrap, globalMeta.ContainsForwardDecls))
		}
	}

	// Handle selector scope based on mode
	if common.SelectorScopeMode == R3SelectorScopeModeInline {
		if globalMeta, ok := meta.(*R3NgModuleMetadataGlobal); ok {
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
	} else if common.SelectorScopeMode == R3SelectorScopeModeSideEffect {
		setNgModuleScopeCall := generateSetNgModuleScopeCall(meta)
		if setNgModuleScopeCall != nil {
			statements = append(statements, setNgModuleScopeCall)
		}
	}

	// Handle schemas
	if common.Schemas != nil && len(common.Schemas) > 0 {
		schemaValues := make([]output.OutputExpression, len(common.Schemas))
		for i, ref := range common.Schemas {
			schemaValues[i] = ref.Value
		}
		definitionMap.Set("schemas", output.NewLiteralArrayExpr(schemaValues, nil, nil))
	}

	// Handle ID
	if common.ID != nil {
		definitionMap.Set("id", common.ID)
		statements = append(statements, output.NewExpressionStatement(
			output.NewInvokeFunctionExpr(
				output.NewExternalExpr(r3_identifiers.RegisterNgModuleType, nil, nil, nil),
				[]output.OutputExpression{common.Type.Value, common.ID},
				nil,
				nil,
				false,
			),
			nil,
			nil,
		))
	}

	expression := output.NewInvokeFunctionExpr(
		output.NewExternalExpr(r3_identifiers.DefineNgModule, nil, nil, nil),
		[]output.OutputExpression{definitionMap.ToLiteralMap()},
		nil,
		nil,
		true, // pure
	)
	typ := CreateNgModuleType(meta)

	return render3.R3CompiledExpression{
		Expression: expression,
		Type:       typ,
		Statements: statements,
	}
}

// R3DeclareNgModuleFacadeInternal represents the internal structure of R3DeclareNgModuleFacade
// This matches the TypeScript interface R3DeclareNgModuleFacade
type R3DeclareNgModuleFacadeInternal struct {
	Type         interface{}
	Bootstrap    interface{}
	Declarations interface{}
	Imports      interface{}
	Exports      interface{}
	Schemas      interface{}
	ID           interface{}
}

// CompileNgModuleDeclarationExpression compiles a call to ɵɵdefineNgModule() from a call to ɵɵngDeclareNgModule()
func CompileNgModuleDeclarationExpression(meta facade.R3DeclareNgModuleFacade) output.OutputExpression {
	definitionMap := view.NewDefinitionMap()

	// Type assertion to get fields from meta
	// Since facade.R3DeclareNgModuleFacade is interface{}, we use type assertion
	if metaValue, ok := meta.(*R3DeclareNgModuleFacadeInternal); ok {
		definitionMap.Set("type", output.NewWrappedNodeExpr(metaValue.Type, nil, nil))
		if metaValue.Bootstrap != nil {
			definitionMap.Set("bootstrap", output.NewWrappedNodeExpr(metaValue.Bootstrap, nil, nil))
		}
		if metaValue.Declarations != nil {
			definitionMap.Set("declarations", output.NewWrappedNodeExpr(metaValue.Declarations, nil, nil))
		}
		if metaValue.Imports != nil {
			definitionMap.Set("imports", output.NewWrappedNodeExpr(metaValue.Imports, nil, nil))
		}
		if metaValue.Exports != nil {
			definitionMap.Set("exports", output.NewWrappedNodeExpr(metaValue.Exports, nil, nil))
		}
		if metaValue.Schemas != nil {
			definitionMap.Set("schemas", output.NewWrappedNodeExpr(metaValue.Schemas, nil, nil))
		}
		if metaValue.ID != nil {
			definitionMap.Set("id", output.NewWrappedNodeExpr(metaValue.ID, nil, nil))
		}
	} else if metaMap, ok := meta.(map[string]interface{}); ok {
		// Fallback: try as map[string]interface{}
		if typ, ok := metaMap["type"]; ok && typ != nil {
			definitionMap.Set("type", output.NewWrappedNodeExpr(typ, nil, nil))
		}
		if bootstrap, ok := metaMap["bootstrap"]; ok && bootstrap != nil {
			definitionMap.Set("bootstrap", output.NewWrappedNodeExpr(bootstrap, nil, nil))
		}
		if declarations, ok := metaMap["declarations"]; ok && declarations != nil {
			definitionMap.Set("declarations", output.NewWrappedNodeExpr(declarations, nil, nil))
		}
		if imports, ok := metaMap["imports"]; ok && imports != nil {
			definitionMap.Set("imports", output.NewWrappedNodeExpr(imports, nil, nil))
		}
		if exports, ok := metaMap["exports"]; ok && exports != nil {
			definitionMap.Set("exports", output.NewWrappedNodeExpr(exports, nil, nil))
		}
		if schemas, ok := metaMap["schemas"]; ok && schemas != nil {
			definitionMap.Set("schemas", output.NewWrappedNodeExpr(schemas, nil, nil))
		}
		if id, ok := metaMap["id"]; ok && id != nil {
			definitionMap.Set("id", output.NewWrappedNodeExpr(id, nil, nil))
		}
	}
	return output.NewInvokeFunctionExpr(
		output.NewExternalExpr(r3_identifiers.DefineNgModule, nil, nil, nil),
		[]output.OutputExpression{definitionMap.ToLiteralMap()},
		nil,
		nil,
		false,
	)
}

// CreateNgModuleType creates the NgModule type
func CreateNgModuleType(meta R3NgModuleMetadata) output.Type {
	if localMeta, ok := meta.(*R3NgModuleMetadataLocal); ok {
		return output.NewExpressionType(localMeta.Type.Value, output.TypeModifierNone, nil)
	}

	globalMeta := meta.(*R3NgModuleMetadataGlobal)
	publicDeclarationTypes := globalMeta.PublicDeclarationTypes
	var declarationsType output.Type = output.NoneType
	if publicDeclarationTypes == nil {
		declarationsType = tupleTypeOf(globalMeta.Declarations)
	} else {
		declarationsType = tupleOfTypes(publicDeclarationTypes)
	}

	var importsType output.Type = output.NoneType
	if globalMeta.IncludeImportTypes {
		importsType = tupleTypeOf(globalMeta.Imports)
	}

	return output.NewExpressionType(
		output.NewExternalExpr(
			r3_identifiers.NgModuleDeclaration,
			nil,
			[]output.Type{
				output.NewExpressionType(globalMeta.Type.Type, output.TypeModifierNone, nil),
				declarationsType,
				importsType,
				tupleTypeOf(globalMeta.Exports),
			},
			nil,
		),
		output.TypeModifierNone,
		nil,
	)
}

// generateSetNgModuleScopeCall generates a function call to ɵɵsetNgModuleScope
func generateSetNgModuleScopeCall(meta R3NgModuleMetadata) output.OutputStatement {
	scopeMap := view.NewDefinitionMap()
	common := meta.GetCommon()

	// Handle declarations - check kind separately as in TypeScript
	if common.Kind == R3NgModuleMetadataKindGlobal {
		if globalMeta, ok := meta.(*R3NgModuleMetadataGlobal); ok {
			if len(globalMeta.Declarations) > 0 {
				scopeMap.Set("declarations", render3.RefsToArray(globalMeta.Declarations, globalMeta.ContainsForwardDecls))
			}
		}
	} else {
		if localMeta, ok := meta.(*R3NgModuleMetadataLocal); ok {
			if localMeta.DeclarationsExpression != nil {
				scopeMap.Set("declarations", localMeta.DeclarationsExpression)
			}
		}
	}

	// Handle imports - check kind separately as in TypeScript
	if common.Kind == R3NgModuleMetadataKindGlobal {
		if globalMeta, ok := meta.(*R3NgModuleMetadataGlobal); ok {
			if len(globalMeta.Imports) > 0 {
				scopeMap.Set("imports", render3.RefsToArray(globalMeta.Imports, globalMeta.ContainsForwardDecls))
			}
		}
	} else {
		if localMeta, ok := meta.(*R3NgModuleMetadataLocal); ok {
			if localMeta.ImportsExpression != nil {
				scopeMap.Set("imports", localMeta.ImportsExpression)
			}
		}
	}

	// Handle exports - check kind separately as in TypeScript
	if common.Kind == R3NgModuleMetadataKindGlobal {
		if globalMeta, ok := meta.(*R3NgModuleMetadataGlobal); ok {
			if len(globalMeta.Exports) > 0 {
				scopeMap.Set("exports", render3.RefsToArray(globalMeta.Exports, globalMeta.ContainsForwardDecls))
			}
		}
	} else {
		if localMeta, ok := meta.(*R3NgModuleMetadataLocal); ok {
			if localMeta.ExportsExpression != nil {
				scopeMap.Set("exports", localMeta.ExportsExpression)
			}
		}
	}

	// Handle bootstrap - only for local mode as in TypeScript
	if common.Kind == R3NgModuleMetadataKindLocal {
		if localMeta, ok := meta.(*R3NgModuleMetadataLocal); ok {
			if localMeta.BootstrapExpression != nil {
				scopeMap.Set("bootstrap", localMeta.BootstrapExpression)
			}
		}
	}

	if len(scopeMap.Values) == 0 {
		return nil
	}

	// setNgModuleScope(...)
	fnCall := output.NewInvokeFunctionExpr(
		output.NewExternalExpr(r3_identifiers.SetNgModuleScope, nil, nil, nil),
		[]output.OutputExpression{common.Type.Value, scopeMap.ToLiteralMap()},
		nil,
		nil,
		false,
	)

	// (ngJitMode guard) && setNgModuleScope(...)
	guardedCall := render3.JitOnlyGuardedExpression(fnCall)

	// function() { (ngJitMode guard) && setNgModuleScope(...); }
	iife := output.NewFunctionExpr(
		[]*output.FnParam{},
		[]output.OutputStatement{
			output.NewExpressionStatement(guardedCall, nil, nil),
		},
		output.InferredType,
		nil,
		nil,
	)

	// (function() { (ngJitMode guard) && setNgModuleScope(...); })()
	iifeCall := output.NewInvokeFunctionExpr(iife, []output.OutputExpression{}, nil, nil, false)

	return output.NewExpressionStatement(iifeCall, nil, nil)
}

// tupleTypeOf creates a tuple type from an array of R3Reference
func tupleTypeOf(refs []render3.R3Reference) output.Type {
	if len(refs) == 0 {
		return output.NoneType
	}
	types := make([]output.OutputExpression, len(refs))
	for i, ref := range refs {
		types[i] = output.NewTypeofExpr(ref.Type, nil, nil)
	}
	return output.NewExpressionType(
		output.NewLiteralArrayExpr(types, nil, nil),
		output.TypeModifierNone,
		nil,
	)
}

// tupleOfTypes creates a tuple type from an array of expressions
func tupleOfTypes(types []output.OutputExpression) output.Type {
	if len(types) == 0 {
		return output.NoneType
	}
	typeofTypes := make([]output.OutputExpression, len(types))
	for i, typ := range types {
		typeofTypes[i] = output.NewTypeofExpr(typ, nil, nil)
	}
	return output.NewExpressionType(
		output.NewLiteralArrayExpr(typeofTypes, nil, nil),
		output.TypeModifierNone,
		nil,
	)
}
