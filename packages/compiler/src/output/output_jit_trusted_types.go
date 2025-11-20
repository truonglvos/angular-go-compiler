package output

import "fmt"

// TrustedScript represents a trusted script type
// This is a placeholder for Trusted Types support in JavaScript
// In Go, this is just a string wrapper
type TrustedScript string

// TrustedTypePolicyFactory represents a Trusted Types policy factory
// This is a placeholder for JavaScript Trusted Types API
type TrustedTypePolicyFactory interface {
	CreatePolicy(policyName string, policyOptions TrustedTypePolicyOptions) TrustedTypePolicy
}

// TrustedTypePolicyOptions represents options for creating a Trusted Types policy
type TrustedTypePolicyOptions struct {
	CreateScript func(string) string
}

// TrustedTypePolicy represents a Trusted Types policy
// This is a placeholder for JavaScript Trusted Types API
type TrustedTypePolicy interface {
	CreateScript(input string) TrustedScript
}

// Note: Trusted Types is a browser security feature and doesn't directly apply to Go.
// This file provides placeholder types for compatibility with the TypeScript version.
// The actual Trusted Types logic would be handled in the JavaScript runtime when the
// generated code is executed.

// NewTrustedFunctionForJIT creates a new function for JIT compilation
// This uses the configured JavaScript runtime to create a function
// If Trusted Types are supported, it will use them; otherwise falls back to regular Function
func NewTrustedFunctionForJIT(args ...string) (FunctionHandle, error) {
	if DefaultJSRuntime == nil {
		return nil, fmt.Errorf("JavaScript runtime not initialized. Call InitDefaultJSRuntime first")
	}

	if len(args) == 0 {
		return nil, fmt.Errorf("at least one argument (function body) is required")
	}

	// Last argument is the function body, rest are parameter names
	paramNames := args[:len(args)-1]
	body := args[len(args)-1]

	return DefaultJSRuntime.NewFunction(paramNames, body)
}
