package core

// ViewEncapsulation represents the encapsulation strategy for component styles
type ViewEncapsulation int

const (
	ViewEncapsulationEmulated ViewEncapsulation = iota
	// Historically the 1 value was for Native encapsulation which has been removed as of v11.
	_ // Reserved for historical Native
	ViewEncapsulationNone
	ViewEncapsulationShadowDom
	ViewEncapsulationExperimentalIsolatedShadowDom
)

// ChangeDetectionStrategy represents the change detection strategy
type ChangeDetectionStrategy int

const (
	ChangeDetectionStrategyOnPush ChangeDetectionStrategy = iota
	ChangeDetectionStrategyDefault
)

// Input represents an input property configuration
type Input struct {
	Alias     *string
	Required  *bool
	Transform func(value interface{}) interface{}
	IsSignal  bool
}

// InputFlags describes flags for an input
type InputFlags int

const (
	InputFlagsNone InputFlags = iota
	InputFlagsSignalBased InputFlags = 1 << iota
	InputFlagsHasDecoratorInputTransform
)

// Output represents an output property configuration
type Output struct {
	Alias *string
}

// HostBinding represents a host binding configuration
type HostBinding struct {
	HostPropertyName *string
}

// HostListener represents a host listener configuration
type HostListener struct {
	EventName *string
	Args      []string
}

// SchemaMetadata represents schema metadata
type SchemaMetadata struct {
	Name string
}

var (
	CUSTOM_ELEMENTS_SCHEMA = SchemaMetadata{Name: "custom-elements"}
	NO_ERRORS_SCHEMA       = SchemaMetadata{Name: "no-errors-schema"}
)

// SecurityContext represents the security context for sanitization
type SecurityContext int

const (
	SecurityContextNONE SecurityContext = iota
	SecurityContextHTML
	SecurityContextSTYLE
	SecurityContextSCRIPT
	SecurityContextURL
	SecurityContextRESOURCE_URL
)

// InjectFlags represents injection flags for dependency injection
type InjectFlags int

const (
	InjectFlagsDefault InjectFlags = iota
	InjectFlagsHost                = 1 << iota
	InjectFlagsSelf
	InjectFlagsSkipSelf
	InjectFlagsOptional
	InjectFlagsForPipe
)

// MissingTranslationStrategy represents the strategy for handling missing translations
type MissingTranslationStrategy int

const (
	MissingTranslationStrategyError MissingTranslationStrategy = iota
	MissingTranslationStrategyWarning
	MissingTranslationStrategyIgnore
)

// SelectorFlags are flags used to generate R3-style CSS Selectors
type SelectorFlags int

const (
	SelectorFlagsNOT       SelectorFlags = 0b0001 // Beginning of a new negative selector
	SelectorFlagsATTRIBUTE SelectorFlags = 0b0010 // Mode for matching attributes
	SelectorFlagsELEMENT   SelectorFlags = 0b0100 // Mode for matching tag names
	SelectorFlagsCLASS     SelectorFlags = 0b1000 // Mode for matching class names
)

// R3CssSelector represents an R3 CSS selector
type R3CssSelector []interface{} // string | SelectorFlags

// R3CssSelectorList represents a list of R3 CSS selectors
type R3CssSelectorList []R3CssSelector

// RenderFlags are flags passed into template functions to determine which blocks should be executed
type RenderFlags int

const (
	RenderFlagsCreate RenderFlags = 0b01  // Whether to run the creation block
	RenderFlagsUpdate RenderFlags = 0b10  // Whether to run the update block
)

// AttributeMarker is a set of marker values to be used in the attributes arrays
type AttributeMarker int

const (
	AttributeMarkerNamespaceURI AttributeMarker = iota
	AttributeMarkerClasses
	AttributeMarkerStyles
	AttributeMarkerBindings
	AttributeMarkerTemplate
	AttributeMarkerProjectAs
	AttributeMarkerI18n
)

// EmitDistinctChangesOnlyDefaultValue stores the default value of emitDistinctChangesOnly
const EmitDistinctChangesOnlyDefaultValue = true

// ParseSelectorToR3Selector parses a selector string to R3 selector format
func ParseSelectorToR3Selector(selector *string) R3CssSelectorList {
	if selector == nil || *selector == "" {
		return R3CssSelectorList{}
	}
	// This will be implemented when we integrate with directivematching package
	// For now, return empty list
	return R3CssSelectorList{}
}

