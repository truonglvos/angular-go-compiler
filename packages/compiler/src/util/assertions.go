package util

import (
	"fmt"
	"regexp"
)

// unusableInterpolationRegexps contains regex patterns for unusable interpolation symbols
var unusableInterpolationRegexps = []*regexp.Regexp{
	regexp.MustCompile(`@`),              // control flow reserved symbol
	regexp.MustCompile(`^\s*$`),          // empty
	regexp.MustCompile(`[<>]`),           // html tag
	regexp.MustCompile(`^[{}]$`),         // i18n expansion
	regexp.MustCompile(`(?i)&(#|[a-z])`), // character reference (case insensitive)
	regexp.MustCompile(`^//`),            // comment
}

// AssertInterpolationSymbols validates interpolation symbols
// It checks that the value is either null or an array of exactly 2 strings [start, end],
// and that neither start nor end contains unusable interpolation symbols.
func AssertInterpolationSymbols(identifier string, value interface{}) error {
	if value == nil {
		return nil
	}

	// Check if value is an array/slice with exactly 2 elements
	valueSlice, ok := value.([]interface{})
	if !ok {
		// Try []string
		if strSlice, ok := value.([]string); ok {
			if len(strSlice) != 2 {
				return fmt.Errorf("expected '%s' to be an array, [start, end]", identifier)
			}
			start := strSlice[0]
			end := strSlice[1]
			return checkUnusableSymbols(start, end)
		}
		return fmt.Errorf("expected '%s' to be an array, [start, end]", identifier)
	}

	if len(valueSlice) != 2 {
		return fmt.Errorf("expected '%s' to be an array, [start, end]", identifier)
	}

	// Extract start and end strings
	start, ok1 := valueSlice[0].(string)
	end, ok2 := valueSlice[1].(string)
	if !ok1 || !ok2 {
		return fmt.Errorf("expected '%s' to be an array, [start, end]", identifier)
	}

	return checkUnusableSymbols(start, end)
}

// checkUnusableSymbols checks if start or end contains unusable interpolation symbols
func checkUnusableSymbols(start, end string) error {
	for _, regex := range unusableInterpolationRegexps {
		if regex.MatchString(start) {
			return fmt.Errorf("start symbol '%s' contains unusable interpolation symbol", start)
		}
		if regex.MatchString(end) {
			return fmt.Errorf("end symbol '%s' contains unusable interpolation symbol", end)
		}
	}
	return nil
}

