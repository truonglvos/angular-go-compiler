package view

// Whether to produce instructions that will attach the source location to each DOM node.
//
// !!!Important!!! at the time of writing this flag isn't exposed externally, but internal debug
// tools enable it via a local change. Any modifications to this flag need to update the
// internal tooling as well.
var enableTemplateSourceLocations = false

// SetEnableTemplateSourceLocations is a utility function to enable source locations.
// Intended to be used **only** inside unit tests.
func SetEnableTemplateSourceLocations(value bool) {
	enableTemplateSourceLocations = value
}

// GetTemplateSourceLocationsEnabled gets whether template source locations are enabled.
func GetTemplateSourceLocationsEnabled() bool {
	return enableTemplateSourceLocations
}

