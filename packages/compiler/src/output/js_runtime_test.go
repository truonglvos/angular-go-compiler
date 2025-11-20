package output

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// MockJSRuntime is a mock implementation for testing
type MockJSRuntime struct {
	functions map[string]string
	results   map[string]interface{}
}

func NewMockJSRuntime() *MockJSRuntime {
	return &MockJSRuntime{
		functions: make(map[string]string),
		results:   make(map[string]interface{}),
	}
}

func (m *MockJSRuntime) NewFunction(args []string, body string) (FunctionHandle, error) {
	// Create a simple function ID
	functionID := "mock_fn_123"
	source := "function(" + joinStrings(args, ", ") + ") { " + body + " }"

	m.functions[functionID] = source

	return &MockFunctionHandle{
		id:     functionID,
		source: source,
	}, nil
}

func (m *MockJSRuntime) ExecuteFunction(fn FunctionHandle, args []interface{}) (interface{}, error) {
	_, ok := fn.(*MockFunctionHandle)
	if !ok {
		return nil, &RuntimeError{Message: "invalid function handle"}
	}

	// Simple mock: if function is "return a + b", return sum of first two args
	if len(args) >= 2 {
		if a, ok := args[0].(float64); ok {
			if b, ok := args[1].(float64); ok {
				return a + b, nil
			}
		}
	}

	return nil, &RuntimeError{Message: "mock execution failed"}
}

func (m *MockJSRuntime) SupportsTrustedTypes() bool {
	return false
}

type MockFunctionHandle struct {
	id     string
	source string
}

func (m *MockFunctionHandle) String() string {
	return m.source
}

type RuntimeError struct {
	Message string
}

func (e *RuntimeError) Error() string {
	return e.Message
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

func TestMockJSRuntime(t *testing.T) {
	runtime := NewMockJSRuntime()

	// Test NewFunction
	fn, err := runtime.NewFunction([]string{"a", "b"}, "return a + b;")
	if err != nil {
		t.Fatalf("NewFunction failed: %v", err)
	}

	if fn.String() == "" {
		t.Error("Function source should not be empty")
	}

	// Test ExecuteFunction
	result, err := runtime.ExecuteFunction(fn, []interface{}{5.0, 10.0})
	if err != nil {
		t.Fatalf("ExecuteFunction failed: %v", err)
	}

	if result != 15.0 {
		t.Errorf("Expected result 15.0, got %v", result)
	}
}

func TestNodeJSRuntime_NewFunction(t *testing.T) {
	// Skip if Node.js is not available
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("Node.js not available, skipping integration test")
	}

	// Find helper script
	helperPath := findHelperScript()
	if helperPath == "" {
		t.Skip("Helper script not found, skipping integration test")
	}

	runtime := NewNodeJSRuntime(helperPath)

	// Test NewFunction
	fn, err := runtime.NewFunction([]string{"x"}, "return x * 2;")
	if err != nil {
		t.Fatalf("NewFunction failed: %v", err)
	}

	if fn.String() == "" {
		t.Error("Function source should not be empty")
	}

	t.Logf("Created function: %s", fn.String()[:50]+"...")
}

func TestNodeJSRuntime_ExecuteFunction(t *testing.T) {
	// Skip if Node.js is not available
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("Node.js not available, skipping integration test")
	}

	// Find helper script
	helperPath := findHelperScript()
	if helperPath == "" {
		t.Skip("Helper script not found, skipping integration test")
	}

	runtime := NewNodeJSRuntime(helperPath)

	// Create a function
	fn, err := runtime.NewFunction([]string{"a", "b"}, "return a + b;")
	if err != nil {
		t.Fatalf("NewFunction failed: %v", err)
	}

	// Execute function
	result, err := runtime.ExecuteFunction(fn, []interface{}{5, 10})
	if err != nil {
		t.Fatalf("ExecuteFunction failed: %v", err)
	}

	// Check result
	resultFloat, ok := result.(float64)
	if !ok {
		t.Fatalf("Expected float64 result, got %T", result)
	}

	if resultFloat != 15 {
		t.Errorf("Expected result 15, got %v", resultFloat)
	}

	t.Logf("Function executed successfully, result: %v", resultFloat)
}

func TestNewTrustedFunctionForJIT(t *testing.T) {
	// Use mock runtime for this test
	originalRuntime := DefaultJSRuntime
	defer func() {
		DefaultJSRuntime = originalRuntime
	}()

	DefaultJSRuntime = NewMockJSRuntime()

	// Test NewTrustedFunctionForJIT
	fn, err := NewTrustedFunctionForJIT("a", "b", "return a + b;")
	if err != nil {
		t.Fatalf("NewTrustedFunctionForJIT failed: %v", err)
	}

	if fn == nil {
		t.Error("Function handle should not be nil")
	}
}

func TestNewTrustedFunctionForJIT_NoRuntime(t *testing.T) {
	// Test without runtime initialized
	originalRuntime := DefaultJSRuntime
	DefaultJSRuntime = nil
	defer func() {
		DefaultJSRuntime = originalRuntime
	}()

	_, err := NewTrustedFunctionForJIT("x", "return x;")
	if err == nil {
		t.Error("Expected error when runtime is not initialized")
	}
}

func findHelperScript() string {
	// Try to find the helper script relative to the test file
	testDir := filepath.Dir(".")

	// Try different paths
	paths := []string{
		filepath.Join(testDir, "../../../tools/js-runtime-helper/index.js"),
		filepath.Join(testDir, "../../tools/js-runtime-helper/index.js"),
		"tools/js-runtime-helper/index.js",
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			absPath, _ := filepath.Abs(p)
			return absPath
		}
	}

	return ""
}
