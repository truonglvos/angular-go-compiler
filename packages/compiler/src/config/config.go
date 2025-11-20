package config

import (
	"ngc-go/packages/compiler/src/core"
)

// CompilerConfig represents the compiler configuration
type CompilerConfig struct {
	DefaultEncapsulation      *core.ViewEncapsulation
	PreserveWhitespaces       bool
	StrictInjectionParameters bool
}

// NewCompilerConfig creates a new CompilerConfig with optional parameters
func NewCompilerConfig(opts ...CompilerConfigOption) *CompilerConfig {
	config := &CompilerConfig{
		DefaultEncapsulation:      ViewEncapsulationPtr(core.ViewEncapsulationEmulated),
		PreserveWhitespaces:       PreserveWhitespacesDefault(nil, false),
		StrictInjectionParameters: false,
	}

	for _, opt := range opts {
		opt(config)
	}

	return config
}

// CompilerConfigOption is a function that modifies CompilerConfig
type CompilerConfigOption func(*CompilerConfig)

// WithDefaultEncapsulation sets the default encapsulation
func WithDefaultEncapsulation(encapsulation core.ViewEncapsulation) CompilerConfigOption {
	return func(c *CompilerConfig) {
		c.DefaultEncapsulation = ViewEncapsulationPtr(encapsulation)
	}
}

// WithPreserveWhitespaces sets whether to preserve whitespaces
func WithPreserveWhitespaces(preserve bool) CompilerConfigOption {
	return func(c *CompilerConfig) {
		c.PreserveWhitespaces = preserve
	}
}

// WithStrictInjectionParameters sets strict injection parameters
func WithStrictInjectionParameters(strict bool) CompilerConfigOption {
	return func(c *CompilerConfig) {
		c.StrictInjectionParameters = strict
	}
}

// PreserveWhitespacesDefault returns the default value for preserveWhitespaces
func PreserveWhitespacesDefault(preserveWhitespacesOption *bool, defaultSetting bool) bool {
	if preserveWhitespacesOption == nil {
		return defaultSetting
	}
	return *preserveWhitespacesOption
}

// Helper function to get pointer to ViewEncapsulation
func ViewEncapsulationPtr(v core.ViewEncapsulation) *core.ViewEncapsulation {
	return &v
}
