package ml_parser

// InterpolationConfig represents the configuration for interpolation symbols
type InterpolationConfig struct {
	Start string
	End   string
}

// NewInterpolationConfig creates a new InterpolationConfig from markers
func NewInterpolationConfig(markers []string) *InterpolationConfig {
	if markers == nil || len(markers) != 2 {
		return DefaultInterpolationConfig
	}
	// TODO: Add assertion for interpolation symbols
	return &InterpolationConfig{
		Start: markers[0],
		End:   markers[1],
	}
}

// DefaultInterpolationConfig is the default interpolation configuration
var DefaultInterpolationConfig = &InterpolationConfig{
	Start: "{{",
	End:   "}}",
}

// DefaultContainerBlocks is the set of default container blocks
var DefaultContainerBlocks = map[string]bool{
	"switch": true,
}
