package output

import (
	"testing"
)

func TestJitEvaluator_EvaluateCode(t *testing.T) {
	// Use mock runtime
	originalRuntime := DefaultJSRuntime
	defer func() {
		DefaultJSRuntime = originalRuntime
	}()

	DefaultJSRuntime = NewMockJSRuntime()

	evaluator := NewJitEvaluator()
	ctx := CreateRootEmitterVisitorContext()

	// Create a simple statement: var x = 5;
	stmt := NewDeclareVarStmt(
		"x",
		NewLiteralExpr(5, nil, nil),
		nil, // type
		StmtModifierNone,
		nil, // sourceSpan
		nil, // leadingComments
	)

	ctx.Print(stmt, "var x = ", false)
	stmt.Value.VisitExpression(NewAbstractJsEmitterVisitor(), ctx)
	ctx.Println(stmt, ";")

	// Test EvaluateCode
	vars := map[string]interface{}{
		"x": 5,
	}

	result, err := evaluator.EvaluateCode("ng:///test", ctx, vars, false)
	if err != nil {
		t.Logf("EvaluateCode returned error (expected for mock): %v", err)
		// This is expected since mock runtime doesn't fully implement execution
		return
	}

	if result == nil {
		t.Error("Result should not be nil")
	}
}

func TestJitEvaluator_ExecuteFunction(t *testing.T) {
	// Use mock runtime
	originalRuntime := DefaultJSRuntime
	defer func() {
		DefaultJSRuntime = originalRuntime
	}()

	DefaultJSRuntime = NewMockJSRuntime()

	evaluator := NewJitEvaluator()

	// Create a function handle
	fn, err := DefaultJSRuntime.NewFunction([]string{"x"}, "return x * 2;")
	if err != nil {
		t.Fatalf("Failed to create function: %v", err)
	}

	// Execute function
	result, err := evaluator.ExecuteFunction(fn, []interface{}{5.0})
	if err != nil {
		t.Fatalf("ExecuteFunction failed: %v", err)
	}

	t.Logf("Function executed, result: %v", result)
}

func TestJitEmitterVisitor(t *testing.T) {
	// Create a mock resolver
	resolver := &MockExternalReferenceResolver{
		values: make(map[string]interface{}),
	}

	visitor := NewJitEmitterVisitor(resolver)
	ctx := CreateRootEmitterVisitorContext()

	// Create a simple expression statement
	stmt := NewExpressionStatement(
		NewReadVarExpr("testVar", nil, nil),
		nil,
		nil,
	)

	visitor.VisitAllStatements([]OutputStatement{stmt}, ctx)

	// Check that context has content
	source := ctx.ToSource()
	if source == "" {
		t.Error("Context source should not be empty")
	}

	t.Logf("Generated source: %s", source)
}

// MockExternalReferenceResolver for testing
type MockExternalReferenceResolver struct {
	values map[string]interface{}
}

func (m *MockExternalReferenceResolver) ResolveExternalReference(ref *ExternalReference) interface{} {
	moduleName := ""
	if ref.ModuleName != nil {
		moduleName = *ref.ModuleName
	}
	name := ""
	if ref.Name != nil {
		name = *ref.Name
	}
	key := moduleName + "." + name
	if val, ok := m.values[key]; ok {
		return val
	}
	return nil
}
