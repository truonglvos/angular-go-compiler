package output

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// JSRuntime represents a JavaScript runtime interface
// This allows different implementations (Node.js helper, embedded engine, etc.)
type JSRuntime interface {
	// NewFunction creates a new JavaScript function from source code
	// args: function parameter names
	// body: function body as string
	// Returns a function handle that can be executed
	NewFunction(args []string, body string) (FunctionHandle, error)

	// ExecuteFunction executes a function handle with given arguments
	ExecuteFunction(fn FunctionHandle, args []interface{}) (interface{}, error)

	// SupportsTrustedTypes returns true if the runtime supports Trusted Types
	SupportsTrustedTypes() bool
}

// FunctionHandle represents a handle to a JavaScript function
// The actual implementation depends on the runtime
type FunctionHandle interface {
	// String returns the function source code
	String() string
}

// NodeJSRuntime implements JSRuntime using a Node.js helper process
// This is the recommended approach for production use
type NodeJSRuntime struct {
	helperPath string
}

// NewNodeJSRuntime creates a new Node.js runtime
// helperPath: path to the Node.js helper script
func NewNodeJSRuntime(helperPath string) *NodeJSRuntime {
	return &NodeJSRuntime{
		helperPath: helperPath,
	}
}

// NewFunction creates a new JavaScript function using Node.js
func (r *NodeJSRuntime) NewFunction(args []string, body string) (FunctionHandle, error) {
	// Call Node.js helper to create function
	cmd := exec.Command("node", r.helperPath, "new-function")
	cmd.Stdin = strings.NewReader(fmt.Sprintf(`{"args":%s,"body":%s}`,
		mustJSON(args), mustJSON(body)))

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to create function: %w", err)
	}

	var result struct {
		FunctionID string `json:"functionId"`
		Source     string `json:"source"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &NodeJSFunctionHandle{
		runtime:    r,
		functionID: result.FunctionID,
		source:     result.Source,
	}, nil
}

// ExecuteFunction executes a function using Node.js
func (r *NodeJSRuntime) ExecuteFunction(fn FunctionHandle, args []interface{}) (interface{}, error) {
	nodeFn, ok := fn.(*NodeJSFunctionHandle)
	if !ok {
		return nil, fmt.Errorf("invalid function handle type")
	}

	cmd := exec.Command("node", r.helperPath, "execute")
	// Try functionId first, fallback to source if functionId fails
	input := map[string]interface{}{
		"functionId": nodeFn.functionID,
		"source":     nodeFn.source, // Fallback option
		"args":       args,
	}
	cmd.Stdin = strings.NewReader(mustJSON(input))

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute function: %w", err)
	}

	var result struct {
		Result interface{} `json:"result"`
		Error  string      `json:"error,omitempty"`
	}
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if result.Error != "" {
		return nil, fmt.Errorf("function execution error: %s", result.Error)
	}

	return result.Result, nil
}

// SupportsTrustedTypes returns true (Node.js can support Trusted Types)
func (r *NodeJSRuntime) SupportsTrustedTypes() bool {
	return true
}

// NodeJSFunctionHandle represents a function handle from Node.js runtime
type NodeJSFunctionHandle struct {
	runtime    *NodeJSRuntime
	functionID string
	source     string
}

func (f *NodeJSFunctionHandle) String() string {
	return f.source
}

// EmbeddedJSRuntime implements JSRuntime using an embedded JavaScript engine
// This could use goja, otto, or v8go
type EmbeddedJSRuntime struct {
	// TODO: Add embedded engine instance
	// For example: vm *goja.Runtime
}

// NewEmbeddedJSRuntime creates a new embedded JavaScript runtime
// engineType: "goja", "otto", or "v8go"
func NewEmbeddedJSRuntime(engineType string) (*EmbeddedJSRuntime, error) {
	// TODO: Initialize the embedded engine
	// This would require adding dependencies like:
	// - github.com/dop251/goja (pure Go JS engine)
	// - github.com/robertkrimen/otto (pure Go JS engine)
	// - github.com/rogchap/v8go (V8 bindings)
	return &EmbeddedJSRuntime{}, fmt.Errorf("embedded runtime not yet implemented, use NodeJSRuntime")
}

// NewFunction creates a new JavaScript function using embedded engine
func (r *EmbeddedJSRuntime) NewFunction(args []string, body string) (FunctionHandle, error) {
	// TODO: Implement using embedded engine
	return nil, fmt.Errorf("not implemented")
}

// ExecuteFunction executes a function using embedded engine
func (r *EmbeddedJSRuntime) ExecuteFunction(fn FunctionHandle, args []interface{}) (interface{}, error) {
	// TODO: Implement using embedded engine
	return nil, fmt.Errorf("not implemented")
}

// SupportsTrustedTypes returns false (embedded engines typically don't support Trusted Types)
func (r *EmbeddedJSRuntime) SupportsTrustedTypes() bool {
	return false
}

// DefaultJSRuntime is the default JavaScript runtime
// It uses Node.js helper by default
var DefaultJSRuntime JSRuntime

// InitDefaultJSRuntime initializes the default JavaScript runtime
// This should be called at startup
func InitDefaultJSRuntime(helperPath string) error {
	DefaultJSRuntime = NewNodeJSRuntime(helperPath)
	return nil
}

// Helper functions

func mustJSON(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal JSON: %v", err))
	}
	return string(data)
}
