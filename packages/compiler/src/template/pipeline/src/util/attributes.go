package pipeline_util

const ariaPrefix = "aria-"

// IsAriaAttribute returns whether `name` is an ARIA attribute name.
//
// This is a heuristic based on whether name begins with and is longer than `aria-`.
func IsAriaAttribute(name string) bool {
	return len(name) > len(ariaPrefix) && name[:len(ariaPrefix)] == ariaPrefix
}
