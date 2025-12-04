package output

import (
	"fmt"
)

// ExternalReferenceResolver resolves external references
type ExternalReferenceResolver interface {
	ResolveExternalReference(ref *ExternalReference) interface{}
}

// JitEvaluator is a helper class to manage the evaluation of JIT generated code
type JitEvaluator struct{}

// NewJitEvaluator creates a new JitEvaluator
func NewJitEvaluator() *JitEvaluator {
	return &JitEvaluator{}
}

// EvaluateStatements evaluates an array of Angular statement AST nodes
func (je *JitEvaluator) EvaluateStatements(
	sourceURL string,
	statements []OutputStatement,
	refResolver ExternalReferenceResolver,
	createSourceMaps bool,
) (map[string]interface{}, error) {
	converter := NewJitEmitterVisitor(refResolver)
	ctx := CreateRootEmitterVisitorContext()

	// Ensure generated code is in strict mode
	if len(statements) > 0 && !isUseStrictStatement(statements[0]) {
		useStrict := NewExpressionStatement(
			NewLiteralExpr("use strict", nil, nil),
			nil,
			nil,
		)
		statements = append([]OutputStatement{useStrict}, statements...)
	}

	converter.VisitAllStatements(statements, ctx)
	converter.CreateReturnStmt(ctx)

	args := converter.GetArgs()
	return je.EvaluateCode(sourceURL, ctx, args, createSourceMaps)
}

// EvaluateCode evaluates a piece of JIT generated code
func (je *JitEvaluator) EvaluateCode(
	sourceURL string,
	ctx *EmitterVisitorContext,
	vars map[string]interface{},
	createSourceMap bool,
) (map[string]interface{}, error) {
	fnArgNames := []string{}
	fnArgValues := []interface{}{}

	for argName, argValue := range vars {
		fnArgValues = append(fnArgValues, argValue)
		fnArgNames = append(fnArgNames, argName)
	}

	// Create function body
	fnBody := fmt.Sprintf(`"use strict";%s\n//# sourceURL=%s`, ctx.ToSource(), sourceURL)

	// TODO: Implement source map generation when NewTrustedFunctionForJIT is available
	// if createSourceMap {
	// 	// using `new Function(...)` generates a header, 1 line of no arguments, 2 lines otherwise
	// 	// E.g. ```
	// 	// function anonymous(a,b,c
	// 	// /**/) { ... }```
	// 	// We don't want to hard code this fact, so we auto detect it via an empty function first.
	// 	// emptyFn := NewTrustedFunctionForJIT(append(fnArgNames, "return null;")...)
	// 	// headerLines := ... // calculate header lines
	// 	// sourceMapGen, err := ctx.ToSourceMapGenerator(sourceURL, headerLines)
	// 	// if err != nil {
	// 	// 	return nil, err
	// 	// }
	// 	// jsComment, err := sourceMapGen.ToJsComment()
	// 	// if err != nil {
	// 	// 	return nil, err
	// 	// }
	// 	// fnBody += "\n" + jsComment
	// }

	// Create function using JavaScript runtime
	fn, err := NewTrustedFunctionForJIT(append(fnArgNames, fnBody)...)
	if err != nil {
		return nil, fmt.Errorf("failed to create function: %w", err)
	}

	// Execute function
	result, err := je.ExecuteFunction(fn, fnArgValues)
	if err != nil {
		return nil, fmt.Errorf("failed to execute function: %w", err)
	}

	// Result should be a map[string]interface{}
	if resultMap, ok := result.(map[string]interface{}); ok {
		return resultMap, nil
	}

	return map[string]interface{}{}, nil
}

// ExecuteFunction executes a JIT generated function by calling it
// This method can be overridden in tests to capture the functions that are generated
func (je *JitEvaluator) ExecuteFunction(fn interface{}, args []interface{}) (interface{}, error) {
	if DefaultJSRuntime == nil {
		return nil, fmt.Errorf("JavaScript runtime not initialized. Call InitDefaultJSRuntime first")
	}

	fnHandle, ok := fn.(FunctionHandle)
	if !ok {
		return nil, fmt.Errorf("invalid function handle type")
	}

	return DefaultJSRuntime.ExecuteFunction(fnHandle, args)
}

// JitEmitterVisitor is an Angular AST visitor that converts AST nodes into executable JavaScript code
type JitEmitterVisitor struct {
	*AbstractJsEmitterVisitor
	refResolver      ExternalReferenceResolver
	evalArgNames     []string
	evalArgValues    []interface{}
	evalExportedVars []string
}

// NewJitEmitterVisitor creates a new JitEmitterVisitor
func NewJitEmitterVisitor(refResolver ExternalReferenceResolver) *JitEmitterVisitor {
	return &JitEmitterVisitor{
		AbstractJsEmitterVisitor: NewAbstractJsEmitterVisitor(),
		refResolver:              refResolver,
		evalArgNames:             []string{},
		evalArgValues:            []interface{}{},
		evalExportedVars:         []string{},
	}
}

// CreateReturnStmt creates a return statement
func (jev *JitEmitterVisitor) CreateReturnStmt(ctx *EmitterVisitorContext) {
	entries := []*LiteralMapEntry{}
	for _, resultVar := range jev.evalExportedVars {
		entries = append(entries, NewLiteralMapEntry(
			resultVar,
			NewReadVarExpr(resultVar, nil, nil),
			false,
		))
	}

	stmt := NewReturnStatement(
		NewLiteralMapExpr(entries, nil, nil),
		nil,
		nil,
	)
	stmt.VisitStatement(jev, ctx)
}

// GetArgs returns the arguments map
func (jev *JitEmitterVisitor) GetArgs() map[string]interface{} {
	result := make(map[string]interface{})
	for i := 0; i < len(jev.evalArgNames); i++ {
		result[jev.evalArgNames[i]] = jev.evalArgValues[i]
	}
	return result
}

// VisitAllStatements overrides the base implementation to ensure proper visitor dispatch
func (jev *JitEmitterVisitor) VisitAllStatements(statements []OutputStatement, ctx *EmitterVisitorContext) {
	for _, stmt := range statements {
		stmt.VisitStatement(jev, ctx) // Pass jev, not the embedded visitor
	}
}

// VisitAllExpressions overrides the base implementation to ensure proper visitor dispatch
func (jev *JitEmitterVisitor) VisitAllExpressions(expressions []OutputExpression, ctx *EmitterVisitorContext, separator string) {
	jev.VisitAllObjects(func(expr OutputExpression) {
		expr.VisitExpression(jev, ctx) // Pass jev, not the embedded visitor
	}, expressions, ctx, separator)
}

// VisitExpressionStmt overrides to ensure proper visitor dispatch
func (jev *JitEmitterVisitor) VisitExpressionStmt(stmt *ExpressionStatement, context interface{}) interface{} {
	ctx := jev.getContext(context)
	stmt.Expr.VisitExpression(jev, ctx) // Use jev instead of base visitor
	ctx.Println(stmt, ";")
	return nil
}

// VisitLiteralArrayExpr overrides to ensure proper visitor dispatch
func (jev *JitEmitterVisitor) VisitLiteralArrayExpr(ast *LiteralArrayExpr, context interface{}) interface{} {
	ctx := jev.getContext(context)
	ctx.Print(ast, "[", false)
	jev.VisitAllExpressions(ast.Entries, ctx, ",") // This will now use our overridden version
	ctx.Print(ast, "]", false)
	return nil
}

// VisitExternalExpr visits an external expression
func (jev *JitEmitterVisitor) VisitExternalExpr(ast *ExternalExpr, context interface{}) interface{} {
	ctx := jev.getContext(context)
	value := jev.refResolver.ResolveExternalReference(ast.Value)
	jev.emitReferenceToExternal(ast, value, ctx)
	return nil
}

// VisitWrappedNodeExpr visits a wrapped node expression
func (jev *JitEmitterVisitor) VisitWrappedNodeExpr(ast *WrappedNodeExpr, context interface{}) interface{} {
	ctx := jev.getContext(context)
	jev.emitReferenceToExternal(ast, ast.Node, ctx)
	return nil
}

// VisitDeclareVarStmt visits a declare variable statement
func (jev *JitEmitterVisitor) VisitDeclareVarStmt(stmt *DeclareVarStmt, context interface{}) interface{} {
	if stmt.GetModifiers()&StmtModifierExported != 0 {
		jev.evalExportedVars = append(jev.evalExportedVars, stmt.Name)
	}
	return jev.AbstractJsEmitterVisitor.VisitDeclareVarStmt(stmt, context)
}

// VisitDeclareFunctionStmt visits a declare function statement
func (jev *JitEmitterVisitor) VisitDeclareFunctionStmt(stmt *DeclareFunctionStmt, context interface{}) interface{} {
	if stmt.GetModifiers()&StmtModifierExported != 0 {
		jev.evalExportedVars = append(jev.evalExportedVars, stmt.Name)
	}
	return jev.AbstractJsEmitterVisitor.VisitDeclareFunctionStmt(stmt, context)
}

// emitReferenceToExternal emits a reference to an external value
func (jev *JitEmitterVisitor) emitReferenceToExternal(
	ast OutputExpression,
	value interface{},
	ctx *EmitterVisitorContext,
) {
	id := -1
	for i, v := range jev.evalArgValues {
		if v == value {
			id = i
			break
		}
	}

	if id == -1 {
		id = len(jev.evalArgValues)
		jev.evalArgValues = append(jev.evalArgValues, value)
		name := identifierName(value)
		if name == "" {
			name = "val"
		}
		jev.evalArgNames = append(jev.evalArgNames, fmt.Sprintf("jit_%s_%d", name, id))
	}
	ctx.Print(ast, jev.evalArgNames[id], false)
}

// identifierName gets the identifier name from a value
// This is a placeholder - would need to be implemented based on the actual value type
func identifierName(value interface{}) string {
	// TODO: Implement proper identifier name extraction
	// This would typically extract a name from a reference or value
	_ = value // suppress unused parameter warning
	return ""
}

// isUseStrictStatement checks if a statement is a "use strict" statement
func isUseStrictStatement(stmt OutputStatement) bool {
	if exprStmt, ok := stmt.(*ExpressionStatement); ok {
		if litExpr, ok := exprStmt.Expr.(*LiteralExpr); ok {
			if str, ok := litExpr.Value.(string); ok {
				return str == "use strict"
			}
		}
	}
	return false
}
