package core

import (
	"regexp"
	"strings"
)

// Version represents a semantic version
type Version struct {
	Full  string
	Major string
	Minor string
	Patch string
}

// NewVersion creates a new Version from a full version string
func NewVersion(full string) *Version {
	parts := strings.Split(full, ".")
	v := &Version{Full: full}
	if len(parts) > 0 {
		v.Major = parts[0]
	}
	if len(parts) > 1 {
		v.Minor = parts[1]
	}
	if len(parts) > 2 {
		v.Patch = strings.Join(parts[2:], ".")
	}
	return v
}

var v1To18Regexp = regexp.MustCompile(`^([1-9]|1[0-8])\.`)

// GetJitStandaloneDefaultForVersion returns the default JIT standalone setting for a version
func GetJitStandaloneDefaultForVersion(version string) bool {
	if strings.HasPrefix(version, "0.") {
		// 0.0.0 is always "latest", default is true
		return true
	}
	if v1To18Regexp.MatchString(version) {
		// Angular v2 - v18 default is false
		return false
	}
	// All other Angular versions (v19+) default to true
	return true
}

