package schema

import (
	"ngc-go/packages/compiler/src/core"
)

// ElementSchemaRegistry is an abstract interface for element schema registry
type ElementSchemaRegistry interface {
	// HasProperty checks if a property exists on an element
	HasProperty(tagName string, propName string, schemaMetas []*core.SchemaMetadata) bool

	// HasElement checks if an element exists
	HasElement(tagName string, schemaMetas []*core.SchemaMetadata) bool

	// SecurityContext returns the security context for a property
	SecurityContext(elementName string, propName string, isAttribute bool) core.SecurityContext

	// AllKnownElementNames returns all known element names
	AllKnownElementNames() []string

	// GetMappedPropName returns the mapped property name
	GetMappedPropName(propName string) string

	// GetDefaultComponentElementName returns the default component element name
	GetDefaultComponentElementName() string

	// ValidateProperty validates a property name
	ValidateProperty(name string) PropertyValidationResult

	// ValidateAttribute validates an attribute name
	ValidateAttribute(name string) PropertyValidationResult

	// NormalizeAnimationStyleProperty normalizes an animation style property name
	NormalizeAnimationStyleProperty(propName string) string

	// NormalizeAnimationStyleValue normalizes an animation style value
	NormalizeAnimationStyleValue(camelCaseProp string, userProvidedProp string, val interface{}) AnimationStyleValueResult
}

// PropertyValidationResult represents the result of property validation
type PropertyValidationResult struct {
	Error bool
	Msg   string
}

// AnimationStyleValueResult represents the result of animation style value normalization
type AnimationStyleValueResult struct {
	Error string
	Value string
}
