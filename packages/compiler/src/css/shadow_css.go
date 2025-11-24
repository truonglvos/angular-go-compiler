package css

import (
	"fmt"
	"ngc-go/packages/compiler/src/core"
	"regexp"
	"strings"
)

// Animation keywords that should not be modified during keyframe scoping
var animationKeywords = map[string]bool{
	// global values
	"inherit": true,
	"initial": true,
	"revert":  true,
	"unset":   true,
	// animation-direction
	"alternate":         true,
	"alternate-reverse": true,
	"normal":            true,
	"reverse":           true,
	// animation-fill-mode
	"backwards": true,
	"both":      true,
	"forwards":  true,
	"none":      true,
	// animation-play-state
	"paused":  true,
	"running": true,
	// animation-timing-function
	"ease":        true,
	"ease-in":     true,
	"ease-in-out": true,
	"ease-out":    true,
	"linear":      true,
	"step-start":  true,
	"step-end":    true,
	// `steps()` function
	"end":        true,
	"jump-both":  true,
	"jump-end":   true,
	"jump-none":  true,
	"jump-start": true,
	"start":      true,
}

// Scoped at-rule identifiers
var scopedAtRuleIdentifiers = []string{
	"@media",
	"@supports",
	"@document",
	"@layer",
	"@container",
	"@scope",
	"@starting-style",
}

// ShadowCss provides ShadowDOM CSS styling shim
type ShadowCss struct {
	safeSelector                    *SafeSelector
	shouldScopeIndicator            *bool
	animationDeclarationKeyframesRe *regexp.Regexp
}

// NewShadowCss creates a new ShadowCss instance
func NewShadowCss() *ShadowCss {
	// Regex to match potential keyframe names:
	// 1. Double quoted string: "((?:[^"\\]|\\.)*)"
	// 2. Single quoted string: '((?:[^'\\]|\\.)*)'
	// 3. Identifier: ([-a-zA-Z0-9_]+)
	// We use this to find candidates and then manually check boundaries
	animationRe := regexp.MustCompile(`"((?:[^"\\]|\\.)*)"|'((?:[^'\\]|\\.)*)'|([-a-zA-Z0-9_]+)`)

	return &ShadowCss{
		safeSelector:                    nil,
		shouldScopeIndicator:            nil,
		animationDeclarationKeyframesRe: animationRe,
	}
}

// ShimCssText shims CSS text with the given selector
func (sc *ShadowCss) ShimCssText(cssText string, selector string, hostSelector string) string {
	if hostSelector == "" {
		hostSelector = ""
	}

	// Collect comments and replace them with a placeholder
	comments := []string{}
	commentRe := regexp.MustCompile(`/\*[\s\S]*?\*/`)
	commentWithHashRe := regexp.MustCompile(`/\*\s*#\s*source(Mapping)?URL=`)
	newLinesRe := regexp.MustCompile(`\r?\n`)
	commentPlaceholder := "%COMMENT%"
	commentWithHashPlaceHolderRe := regexp.MustCompile(commentPlaceholder)

	cssText = commentRe.ReplaceAllStringFunc(cssText, func(m string) string {
		if commentWithHashRe.MatchString(m) {
			comments = append(comments, m)
		} else {
			newLinesMatches := newLinesRe.FindAllString(m, -1)
			newLinesStr := strings.Join(newLinesMatches, "") + "\n"
			comments = append(comments, newLinesStr)
		}
		return commentPlaceholder
	})

	cssText = sc.insertDirectives(cssText)
	scopedCssText := sc.scopeCssText(cssText, selector, hostSelector)

	// Add back comments at the original position
	commentIdx := 0
	scopedCssText = commentWithHashPlaceHolderRe.ReplaceAllStringFunc(scopedCssText, func(_ string) string {
		if commentIdx < len(comments) {
			result := comments[commentIdx]
			commentIdx++
			return result
		}
		return ""
	})

	return scopedCssText
}

func (sc *ShadowCss) insertDirectives(cssText string) string {
	cssText = sc.insertPolyfillDirectivesInCssText(cssText)
	return sc.insertPolyfillRulesInCssText(cssText)
}

func (sc *ShadowCss) scopeKeyframesRelatedCss(cssText string, scopeSelector string) string {
	unscopedKeyframesSet := make(map[string]bool)

	scopedKeyframesCssText := ProcessRules(cssText, func(rule *CssRule) *CssRule {
		return sc.scopeLocalKeyframeDeclarations(rule, scopeSelector, unscopedKeyframesSet)
	})

	return ProcessRules(scopedKeyframesCssText, func(rule *CssRule) *CssRule {
		return sc.scopeAnimationRule(rule, scopeSelector, unscopedKeyframesSet)
	})
}

func (sc *ShadowCss) scopeLocalKeyframeDeclarations(rule *CssRule, scopeSelector string, unscopedKeyframesSet map[string]bool) *CssRule {
	// Pattern: (^@(?:-webkit-)?keyframes(?:\s+))(['"]?)(.+)\2(\s*)$
	// Go doesn't support backreferences, so we match double-quoted, single-quoted, or unquoted separately
	keyframeRe := regexp.MustCompile(`(^@(?:-webkit-)?keyframes(?:\s+))(['"]?)(.+?)(['"]?)(\s*)$`)

	newSelector := keyframeRe.ReplaceAllStringFunc(rule.Selector, func(match string) string {
		matches := keyframeRe.FindStringSubmatch(match)
		if len(matches) < 6 {
			return match
		}
		start := matches[1]
		quote1 := matches[2]
		keyframeName := matches[3]
		quote2 := matches[4]
		endSpaces := matches[5]

		// Determine if quoted
		isQuoted := (quote1 == `"` || quote1 == `'`) && quote1 == quote2
		unescapedName := unescapeQuotes(keyframeName, isQuoted)
		unscopedKeyframesSet[unescapedName] = true

		quote := quote1
		if quote == "" {
			quote = quote2
		}
		return fmt.Sprintf("%s%s%s_%s%s%s", start, quote, scopeSelector, keyframeName, quote, endSpaces)
	})

	return &CssRule{
		Selector: newSelector,
		Content:  rule.Content,
	}
}

func (sc *ShadowCss) scopeAnimationKeyframe(keyframe string, scopeSelector string, unscopedKeyframesSet map[string]bool) string {
	// Remove leading/trailing whitespace
	trimmed := strings.TrimSpace(keyframe)
	if trimmed == "" {
		return keyframe
	}

	// Check for quotes
	var quote string
	var name string
	if strings.HasPrefix(trimmed, `"`) && strings.HasSuffix(trimmed, `"`) {
		quote = `"`
		name = trimmed[1 : len(trimmed)-1]
	} else if strings.HasPrefix(trimmed, `'`) && strings.HasSuffix(trimmed, `'`) {
		quote = `'`
		name = trimmed[1 : len(trimmed)-1]
	} else {
		name = trimmed
	}

	// Unescape quotes in name if necessary to check against unscoped set
	unescaped := name
	if quote != "" {
		isQuoted := quote == `"` || quote == `'`
		unescaped = unescapeQuotes(name, isQuoted)
	}

	prefix := ""
	if unscopedKeyframesSet[unescaped] {
		prefix = scopeSelector + "_"
	}

	// Reconstruct
	// Preserve original whitespace
	startSpace := ""
	endSpace := ""
	if len(keyframe) > len(trimmed) {
		startSpace = keyframe[:strings.Index(keyframe, trimmed)]
		endSpace = keyframe[strings.Index(keyframe, trimmed)+len(trimmed):]
	}

	return fmt.Sprintf("%s%s%s%s%s%s", startSpace, quote, prefix, name, quote, endSpace)
}

func (sc *ShadowCss) scopeAnimationRule(rule *CssRule, scopeSelector string, unscopedKeyframesSet map[string]bool) *CssRule {
	// Replace animation property
	animationRe := regexp.MustCompile(`((?:^|\s+|;)(?:-webkit-)?animation\s*:\s*),*([^;]+)`)

	content := animationRe.ReplaceAllStringFunc(rule.Content, func(match string) string {
		matches := animationRe.FindStringSubmatch(match)
		if len(matches) < 3 {
			return match
		}
		start := matches[1]
		animationDeclarations := matches[2]

		// Process each keyframe in animation declarations
		// We use the token regex to find candidates and then check boundaries manually
		keyframeRe := sc.animationDeclarationKeyframesRe

		tokenMatches := keyframeRe.FindAllStringIndex(animationDeclarations, -1)
		if tokenMatches == nil {
			return match
		}

		var sb strings.Builder
		lastIndex := 0

		for _, loc := range tokenMatches {
			tokenStart := loc[0]
			tokenEnd := loc[1]
			token := animationDeclarations[tokenStart:tokenEnd]

			// Append everything before this token
			sb.WriteString(animationDeclarations[lastIndex:tokenStart])
			lastIndex = tokenEnd

			// Check boundaries
			// Preceded by: start of string, whitespace, or comma
			validPrefix := false
			if tokenStart == 0 {
				validPrefix = true
			} else {
				charBefore := animationDeclarations[tokenStart-1]
				if charBefore == ' ' || charBefore == '\t' || charBefore == '\n' || charBefore == '\r' || charBefore == ',' {
					validPrefix = true
				}
			}

			// Followed by: end of string, whitespace, or comma
			validSuffix := false
			if tokenEnd == len(animationDeclarations) {
				validSuffix = true
			} else {
				charAfter := animationDeclarations[tokenEnd]
				if charAfter == ' ' || charAfter == '\t' || charAfter == '\n' || charAfter == '\r' || charAfter == ',' {
					validSuffix = true
				}
			}

			if validPrefix && validSuffix {
				// It's a candidate keyframe
				// Determine if it's quoted or unquoted
				if strings.HasPrefix(token, `"`) || strings.HasPrefix(token, `'`) {
					// Quoted
					scoped := sc.scopeAnimationKeyframe(token, scopeSelector, unscopedKeyframesSet)
					sb.WriteString(scoped)
				} else {
					// Unquoted
					if animationKeywords[token] {
						sb.WriteString(token)
					} else {
						scoped := sc.scopeAnimationKeyframe(token, scopeSelector, unscopedKeyframesSet)
						sb.WriteString(scoped)
					}
				}
			} else {
				// Not a keyframe (part of something else?)
				sb.WriteString(token)
			}
		}

		// Append remaining
		sb.WriteString(animationDeclarations[lastIndex:])

		return start + sb.String()
	})

	// Replace animation-name property
	animationNameRe := regexp.MustCompile(`((?:^|\s+|;)(?:-webkit-)?animation-name(?:\s*):(?:\s*))([^;]+)`)
	content = animationNameRe.ReplaceAllStringFunc(content, func(match string) string {
		matches := animationNameRe.FindStringSubmatch(match)
		if len(matches) < 3 {
			return match
		}
		start := matches[1]
		commaSeparatedKeyframes := matches[2]

		keyframes := strings.Split(commaSeparatedKeyframes, ",")
		scopedKeyframes := make([]string, len(keyframes))
		for i, keyframe := range keyframes {
			scopedKeyframes[i] = sc.scopeAnimationKeyframe(keyframe, scopeSelector, unscopedKeyframesSet)
		}
		return start + strings.Join(scopedKeyframes, ",")
	})

	return &CssRule{
		Selector: rule.Selector,
		Content:  content,
	}
}

func (sc *ShadowCss) insertPolyfillDirectivesInCssText(cssText string) string {
	// Pattern: polyfill-next-selector[^}]*content:[\s]*?(['"])(.*?)\1[;\s]*}([^{]*?){
	// Go doesn't support backreferences (\1), so we match double-quoted and single-quoted separately
	// Pattern breakdown:
	// - polyfill-next-selector[^}]*content:[\s]*? - prefix
	// - (['"]) - quote character (group 1 in TS, but we'll split)
	// - (.*?) - content (group 2 in TS)
	// - \1 - backreference to quote (not supported in Go)
	// - [;\s]*}([^{]*?){ - suffix with selector (group 3 in TS)
	// We'll create two patterns: one for double quotes, one for single quotes
	// Combined pattern: polyfill-next-selector[^}]*content:[\s]*?("(.*?)"|'(.*?)')[;\s]*}([^{]*?){
	doubleQuotedRe := regexp.MustCompile(`polyfill-next-selector[^}]*content:[\s]*?"(.*?)"[;\s]*}([^{]*?){`)
	singleQuotedRe := regexp.MustCompile(`polyfill-next-selector[^}]*content:[\s]*?'(.*?)'[;\s]*}([^{]*?){`)

	// Process double-quoted matches
	cssText = doubleQuotedRe.ReplaceAllStringFunc(cssText, func(match string) string {
		matches := doubleQuotedRe.FindStringSubmatch(match)
		if len(matches) >= 3 {
			return matches[1] + "{"
		}
		return match
	})

	// Process single-quoted matches
	cssText = singleQuotedRe.ReplaceAllStringFunc(cssText, func(match string) string {
		matches := singleQuotedRe.FindStringSubmatch(match)
		if len(matches) >= 3 {
			return matches[1] + "{"
		}
		return match
	})

	return cssText
}

func (sc *ShadowCss) insertPolyfillRulesInCssText(cssText string) string {
	// Pattern: (polyfill-rule)[^}]*(content:[\s]*(['"])(.*?)\3)[;\s]*[^}]*}
	// Go doesn't support backreferences (\3), so we match double-quoted and single-quoted separately
	// Pattern breakdown:
	// - (polyfill-rule) - group 1
	// - [^}]* - any chars except }
	// - (content:[\s]*(['"])(.*?)\3) - group 2: content with quote and value
	//   - (['"]) - group 3: quote character
	//   - (.*?) - group 4: content value
	//   - \3 - backreference to quote (not supported in Go)
	// - [;\s]*[^}]*} - suffix
	// We'll create two patterns: one for double quotes, one for single quotes
	doubleQuotedRe := regexp.MustCompile(`(polyfill-rule)[^}]*(content:[\s]*"(.*?)")[;\s]*[^}]*}`)
	singleQuotedRe := regexp.MustCompile(`(polyfill-rule)[^}]*(content:[\s]*'(.*?)')[;\s]*[^}]*}`)

	// Process double-quoted matches
	cssText = doubleQuotedRe.ReplaceAllStringFunc(cssText, func(match string) string {
		matches := doubleQuotedRe.FindStringSubmatch(match)
		if len(matches) >= 4 {
			rule := match
			rule = strings.Replace(rule, matches[1], "", 1)
			rule = strings.Replace(rule, matches[2], "", 1)
			return matches[3] + rule
		}
		return match
	})

	// Process single-quoted matches
	cssText = singleQuotedRe.ReplaceAllStringFunc(cssText, func(match string) string {
		matches := singleQuotedRe.FindStringSubmatch(match)
		if len(matches) >= 4 {
			rule := match
			rule = strings.Replace(rule, matches[1], "", 1)
			rule = strings.Replace(rule, matches[2], "", 1)
			return matches[3] + rule
		}
		return match
	})

	return cssText
}

func (sc *ShadowCss) scopeCssText(cssText string, scopeSelector string, hostSelector string) string {
	unscopedRules := sc.extractUnscopedRulesFromCssText(cssText)

	// Replace :host and :host-context with -shadowcsshost and -shadowcsshostcontext respectively
	cssText = sc.insertPolyfillHostInCssText(cssText)
	cssText = sc.convertColonHost(cssText)
	cssText = sc.convertColonHostContext(cssText)
	cssText = sc.convertShadowDOMSelectors(cssText)

	if scopeSelector != "" {
		cssText = sc.scopeKeyframesRelatedCss(cssText, scopeSelector)
		cssText = sc.scopeSelectors(cssText, scopeSelector, hostSelector)
	}

	cssText = cssText + "\n" + unscopedRules
	return strings.TrimSpace(cssText)
}

func (sc *ShadowCss) extractUnscopedRulesFromCssText(cssText string) string {
	// Pattern: (polyfill-unscoped-rule)[^}]*(content:[\s]*(['"])(.*?)\3)[;\s]*[^}]*}
	// Go doesn't support backreferences (\3), so we match double-quoted and single-quoted separately
	// Pattern breakdown:
	// - (polyfill-unscoped-rule) - group 1
	// - [^}]* - any chars except }
	// - (content:[\s]*(['"])(.*?)\3) - group 2: content with quote and value
	//   - (['"]) - group 3: quote character
	//   - (.*?) - group 4: content value
	//   - \3 - backreference to quote (not supported in Go)
	// - [;\s]*[^}]*} - suffix
	// We'll create two patterns: one for double quotes, one for single quotes
	doubleQuotedRe := regexp.MustCompile(`(polyfill-unscoped-rule)[^}]*(content:[\s]*"(.*?)")[;\s]*[^}]*}`)
	singleQuotedRe := regexp.MustCompile(`(polyfill-unscoped-rule)[^}]*(content:[\s]*'(.*?)')[;\s]*[^}]*}`)

	result := ""

	// Process double-quoted matches
	doubleMatches := doubleQuotedRe.FindAllStringSubmatch(cssText, -1)
	for _, match := range doubleMatches {
		if len(match) >= 4 {
			rule := match[0]
			rule = strings.Replace(rule, match[2], "", 1)
			rule = strings.Replace(rule, match[1], match[3], 1)
			result += rule + "\n\n"
		}
	}

	// Process single-quoted matches
	singleMatches := singleQuotedRe.FindAllStringSubmatch(cssText, -1)
	for _, match := range singleMatches {
		if len(match) >= 4 {
			rule := match[0]
			rule = strings.Replace(rule, match[2], "", 1)
			rule = strings.Replace(rule, match[1], match[3], 1)
			result += rule + "\n\n"
		}
	}

	return result
}

func (sc *ShadowCss) convertColonHost(cssText string) string {
	// Pattern: -shadowcsshost(?:\(([^)]+)\))?([^,{]*)
	// Note: Go regexp doesn't support non-capturing groups in the same way, but we can work around it
	cssColonHostRe := regexp.MustCompile(`-shadowcsshost(?:\(([^)]+)\))?([^,{]*)`)

	return cssColonHostRe.ReplaceAllStringFunc(cssText, func(match string) string {
		matches := cssColonHostRe.FindStringSubmatch(match)
		if len(matches) < 3 {
			return match
		}
		hostSelectors := matches[1]
		otherSelectors := matches[2]

		if hostSelectors != "" {
			convertedSelectors := []string{}
			for _, hostSelector := range sc.splitOnTopLevelCommas(hostSelectors) {
				trimmedHostSelector := strings.TrimSpace(hostSelector)
				if trimmedHostSelector == "" {
					break
				}
				convertedSelector := polyfillHostNoCombinator +
					strings.ReplaceAll(trimmedHostSelector, polyfillHost, "") +
					otherSelectors
				convertedSelectors = append(convertedSelectors, convertedSelector)
			}
			return strings.Join(convertedSelectors, ",")
		} else {
			return polyfillHostNoCombinator + otherSelectors
		}
	})
}

func (sc *ShadowCss) splitOnTopLevelCommas(text string) []string {
	length := len(text)
	parens := 0
	prev := 0
	result := []string{}

	for i := 0; i < length; i++ {
		charCode := int(text[i])

		if charCode == core.CharLPAREN {
			parens++
		} else if charCode == core.CharRPAREN {
			parens--
			if parens < 0 {
				// Found an extra closing paren
				result = append(result, text[prev:i])
				return result
			}
		} else if charCode == core.CharCOMMA && parens == 0 {
			// Found a top-level comma
			result = append(result, text[prev:i])
			prev = i + 1
		}
	}

	// Yield the final chunk
	result = append(result, text[prev:])
	return result
}

func (sc *ShadowCss) convertColonHostContext(cssText string) string {
	results := []string{}
	for _, part := range sc.splitOnTopLevelCommas(cssText) {
		results = append(results, sc.convertColonHostContextInSelectorPart(part))
	}
	return strings.Join(results, ",")
}

func (sc *ShadowCss) convertColonHostContextInSelectorPart(cssText string) string {
	// Pattern: (:(where|is)\()?(-shadowcsscontext(?:\(([^)]+)\))?([^{]*))
	cssColonHostContextReGlobal := regexp.MustCompile(`(:(?:where|is)\()?(-shadowcsscontext(?:\(([^)]+)\))?([^{]*))`)

	return cssColonHostContextReGlobal.ReplaceAllStringFunc(cssText, func(match string) string {
		matches := cssColonHostContextReGlobal.FindStringSubmatch(match)
		if len(matches) < 5 {
			return match
		}
		pseudoPrefix := matches[1]
		selectorText := matches[2]

		contextSelectorGroups := [][]string{{}}

		startIndex := strings.Index(selectorText, polyfillHostContext)
		for startIndex != -1 {
			afterPrefix := selectorText[startIndex+len(polyfillHostContext):]

			if len(afterPrefix) == 0 || afterPrefix[0] != '(' {
				// Edge case of :host-context with no parens
				selectorText = afterPrefix
				startIndex = strings.Index(selectorText, polyfillHostContext)
				continue
			}

			// Extract comma-separated selectors between the parentheses
			newContextSelectors := []string{}
			endIndex := 0
			for _, selector := range sc.splitOnTopLevelCommas(afterPrefix[1:]) {
				endIndex += len(selector) + 1
				trimmed := strings.TrimSpace(selector)
				if trimmed != "" {
					newContextSelectors = append(newContextSelectors, trimmed)
				}
			}

			// Duplicate the current selector group for each of these new selectors
			contextSelectorGroupsLength := len(contextSelectorGroups)
			RepeatGroups(&contextSelectorGroups, len(newContextSelectors))
			for i := 0; i < len(newContextSelectors); i++ {
				for j := 0; j < contextSelectorGroupsLength; j++ {
					contextSelectorGroups[j+i*contextSelectorGroupsLength] = append(
						contextSelectorGroups[j+i*contextSelectorGroupsLength],
						newContextSelectors[i],
					)
				}
			}

			// Update the selectorText
			selectorText = afterPrefix[endIndex+1:]
			startIndex = strings.Index(selectorText, polyfillHostContext)
		}

		// Combine the context selectors
		combined := []string{}
		for _, contextSelectors := range contextSelectorGroups {
			combined = append(combined, combineHostContextSelectors(contextSelectors, selectorText, pseudoPrefix))
		}
		return strings.Join(combined, ", ")
	})
}

func (sc *ShadowCss) convertShadowDOMSelectors(cssText string) string {
	result := cssText
	for _, pattern := range shadowDOMSelectorsRe {
		result = pattern.ReplaceAllString(result, " ")
	}
	return result
}

func (sc *ShadowCss) scopeSelectors(cssText string, scopeSelector string, hostSelector string) string {
	return ProcessRules(cssText, func(rule *CssRule) *CssRule {
		selector := rule.Selector
		content := rule.Content

		if len(selector) == 0 || selector[0] != '@' {
			selector = sc.scopeSelector(scopeSelector, hostSelector, selector, true)

			// Debug log for test case
			if strings.Contains(rule.Selector, ".foo:not") && strings.Contains(rule.Selector, ".bar") {
				fmt.Printf("  Scoped selector: %q\n", selector)
			}
		} else {
			// Check if it's a scoped at-rule
			isScopedAtRule := false
			for _, atRule := range scopedAtRuleIdentifiers {
				if strings.HasPrefix(selector, atRule) {
					isScopedAtRule = true
					break
				}
			}

			if isScopedAtRule {
				content = sc.scopeSelectors(content, scopeSelector, hostSelector)
			} else if strings.HasPrefix(selector, "@font-face") || strings.HasPrefix(selector, "@page") {
				content = sc.stripScopingSelectors(content)
			}
		}

		return &CssRule{
			Selector: selector,
			Content:  content,
		}
	})
}

func (sc *ShadowCss) stripScopingSelectors(cssText string) string {
	return ProcessRules(cssText, func(rule *CssRule) *CssRule {
		selector := rule.Selector
		selector = shadowDeepSelectors.ReplaceAllString(selector, " ")
		selector = polyfillHostNoCombinatorRe.ReplaceAllString(selector, " ")
		return &CssRule{
			Selector: selector,
			Content:  rule.Content,
		}
	})
}

func (sc *ShadowCss) scopeSelector(scopeSelector string, hostSelector string, selector string, isParentSelector bool) string {
	// Split the selector into independent parts by comma, unless comma is within parenthesis
	// Note: Go regexp doesn't support negative lookahead, so we'll use a simpler approach
	parts := sc.splitSelectorByComma(selector)

	scopedParts := []string{}
	for _, part := range parts {
		deepParts := shadowDeepSelectors.Split(part, -1)
		if len(deepParts) > 0 {
			shallowPart := deepParts[0]
			otherParts := deepParts[1:]

			if sc.selectorNeedsScoping(shallowPart, scopeSelector) {
				shallowPart = sc.applySelectorScope(scopeSelector, hostSelector, shallowPart, isParentSelector)
			}

			scopedPart := shallowPart
			if len(otherParts) > 0 {
				scopedPart += " " + strings.Join(otherParts, " ")
			}
			scopedParts = append(scopedParts, scopedPart)
		}
	}

	// Join parts, preserving newlines after commas from original selector
	// TypeScript uses .join(', ') but preserves newlines that were in the original
	// We need to find comma positions in the original selector and check for newlines
	result := ""
	parens := 0

	// Find all comma positions in original selector (outside parentheses)
	commaPositions := []int{}
	for i := 0; i < len(selector); i++ {
		if selector[i] == '(' {
			parens++
		} else if selector[i] == ')' {
			parens--
		} else if selector[i] == ',' && parens == 0 {
			commaPositions = append(commaPositions, i)
		}
	}

	// Join parts, using original separator pattern
	// In TypeScript, .join(', ') adds space after comma, but newlines in parts are preserved
	// So if a part starts with newline, we don't need to add newline in separator
	for i, part := range scopedParts {
		if i > 0 {
			// Check if the current part starts with newline
			// If it does, we just add ", " (comma + space), and the newline in part will be preserved
			// If it doesn't, we check if original had newline after comma
			if len(part) > 0 && part[0] == '\n' {
				// Part already has newline at start, just add ", " (comma + space)
				result += ", "
			} else {
				// Check if there's a newline after the comma at position i-1
				if i-1 < len(commaPositions) {
					commaPos := commaPositions[i-1]
					hasNewline := false
					// Check for newline after comma (after optional spaces)
					for j := commaPos + 1; j < len(selector); j++ {
						if selector[j] == '\n' {
							hasNewline = true
							break
						} else if selector[j] != ' ' && selector[j] != '\t' {
							break
						}
					}
					if hasNewline {
						result += ", \n"
					} else {
						result += ", "
					}
				} else {
					result += ", "
				}
			}
		}
		result += part
	}

	return result
}

func (sc *ShadowCss) splitSelectorByComma(selector string) []string {
	// Simple implementation: split by comma, but respect parentheses
	// This is a simplified version - the full TypeScript version uses negative lookahead
	// In TypeScript, the regex / ?,(?!(?:...)) ?/ matches optional spaces before and after comma,
	// which means when split, those spaces are removed. But we need to preserve newlines.
	// The regex matches: optional space before comma, comma, optional space after comma
	// We need to preserve newlines after comma to match TypeScript behavior for multiline selectors
	result := []string{}
	parens := 0
	start := 0

	for i := 0; i < len(selector); i++ {
		char := selector[i]
		if char == '(' {
			parens++
		} else if char == ')' {
			parens--
		} else if char == ',' && parens == 0 {
			// Trim spaces before comma, but preserve newlines and spaces after comma
			part := selector[start:i]
			// Remove trailing spaces (but not newlines) from the part before comma
			part = strings.TrimRight(part, " \t")
			if part != "" {
				result = append(result, part)
			}
			// Skip comma and optional spaces after comma, but preserve newlines
			start = i + 1
			// Skip spaces after comma, but keep newlines
			for start < len(selector) && (selector[start] == ' ' || selector[start] == '\t') {
				start++
			}
		}
	}
	finalPart := selector[start:]
	// Remove leading spaces (but not newlines) from final part
	finalPart = strings.TrimLeft(finalPart, " \t")
	if finalPart != "" {
		result = append(result, finalPart)
	}
	return result
}

func (sc *ShadowCss) selectorNeedsScoping(selector string, scopeSelector string) bool {
	// Escape brackets in scopeSelector
	escapedScopeSelector := strings.ReplaceAll(scopeSelector, "[", "\\[")
	escapedScopeSelector = strings.ReplaceAll(escapedScopeSelector, "]", "\\]")

	// Pattern: ^(scopeSelector)([>\s~+[.,{:][\s\S]*)?$
	pattern := fmt.Sprintf(`^(%s)([>\s~+[.,{:][\s\S]*)?$`, regexp.QuoteMeta(escapedScopeSelector))
	re := regexp.MustCompile(pattern)
	return !re.MatchString(selector)
}

func (sc *ShadowCss) makeScopeMatcher(scopeSelector string) *regexp.Regexp {
	escapedScopeSelector := strings.ReplaceAll(scopeSelector, "[", "\\[")
	escapedScopeSelector = strings.ReplaceAll(escapedScopeSelector, "]", "\\]")
	pattern := fmt.Sprintf(`^(%s)%s`, regexp.QuoteMeta(escapedScopeSelector), selectorReSuffix)
	return regexp.MustCompile(pattern)
}

func (sc *ShadowCss) applySimpleSelectorScope(selector string, scopeSelector string, hostSelector string) string {
	if polyfillHostRe.MatchString(selector) {
		replaceBy := fmt.Sprintf("[%s]", hostSelector)
		result := selector

		// Replace -shadowcsshost-no-combinator patterns
		for polyfillHostNoCombinatorRe.MatchString(result) {
			result = polyfillHostNoCombinatorRe.ReplaceAllStringFunc(result, func(match string) string {
				matches := polyfillHostNoCombinatorRe.FindStringSubmatch(match)
				if len(matches) >= 2 {
					sel := matches[1]
					// Pattern: ([^:\)]*)(:*)(.*)
					selRe := regexp.MustCompile(`([^:\)]*)(:*)(.*)`)
					return selRe.ReplaceAllString(sel, fmt.Sprintf("${1}%s${2}${3}", replaceBy))
				}
				return match
			})
		}

		return polyfillHostRe.ReplaceAllString(result, replaceBy)
	}

	return scopeSelector + " " + selector
}

func (sc *ShadowCss) applySelectorScope(scopeSelector string, hostSelector string, selector string, isParentSelector bool) string {
	// Reset shouldScopeIndicator at the start of each call
	sc.shouldScopeIndicator = nil

	// Remove [is=...] from scopeSelector
	isRe := regexp.MustCompile(`\[is=([^\]]*)\]`)
	scopeSelector = isRe.ReplaceAllStringFunc(scopeSelector, func(match string) string {
		matches := isRe.FindStringSubmatch(match)
		if len(matches) >= 2 {
			return matches[1]
		}
		return match
	})

	attrName := fmt.Sprintf("[%s]", scopeSelector)

	// normalizeWhitespaceInSelector normalizes whitespace sequences in selector parts
	// It replaces newlines, tabs, and multiple spaces with a single space,
	// while preserving the structure of the selector
	normalizeWhitespaceInSelector := func(s string) string {
		// Replace all whitespace sequences (newlines, tabs, spaces) with a single space
		whitespaceRe := regexp.MustCompile(`\s+`)
		normalized := whitespaceRe.ReplaceAllString(s, " ")
		return normalized
	}

	scopeSelectorPart := func(p string) string {
		scopedP := strings.TrimSpace(p)
		if scopedP == "" {
			return p
		}

		if strings.Contains(p, polyfillHostNoCombinator) {
			scopedP = sc.applySimpleSelectorScope(p, scopeSelector, hostSelector)
			if !isPolyfillHostNoCombinatorOutsidePseudoFunction(p) {
				// Pattern: ([^:]*)(:*)([\s\S]*)
				selRe := regexp.MustCompile(`([^:]*)(:*)([\s\S]*)`)
				matches := selRe.FindStringSubmatch(scopedP)
				if len(matches) >= 4 {
					// Normalize whitespace in the 'after' part (matches[3]) to match TypeScript behavior
					// This ensures newlines and multiple spaces in pseudo-selector arguments are collapsed
					normalizedAfter := normalizeWhitespaceInSelector(matches[3])
					// Match TypeScript: scopedP = before + attrName + colon + after;
					// When before is empty (selector starts with :), attrName will be prepended
					// When before contains scopeSelector prefix (from _applySimpleSelectorScope when selector doesn't contain _polyfillHostRe),
					// we need to remove it first
					before := matches[1]
					beforeTrimmed := strings.TrimSpace(before)
					if beforeTrimmed != "" && (strings.HasPrefix(beforeTrimmed, scopeSelector+" ") || strings.HasPrefix(beforeTrimmed, "["+scopeSelector+"]")) {
						// Remove scopeSelector prefix added by _applySimpleSelectorScope
						// _applySimpleSelectorScope returns scopeSelector + " " + selector when selector doesn't contain _polyfillHostRe
						// In this case, we want to prepend attrName instead
						scopedP = attrName + matches[2] + normalizedAfter
					} else {
						// Match TypeScript: scopedP = before + attrName + colon + after;
						scopedP = matches[1] + attrName + matches[2] + normalizedAfter
					}
				}
			}
		} else {
			// Remove :host
			t := polyfillHostRe.ReplaceAllString(p, "")
			if len(t) > 0 {
				selRe := regexp.MustCompile(`([^:]*)(:*)([\s\S]*)`)
				matches := selRe.FindStringSubmatch(t)
				if len(matches) >= 4 {
					// Normalize whitespace in the 'after' part (matches[3]) to match TypeScript behavior
					// This ensures newlines and multiple spaces in pseudo-selector arguments are collapsed
					normalizedAfter := normalizeWhitespaceInSelector(matches[3])
					// If matches[1] is empty (selector starts with :), prepend attrName
					// Otherwise, insert attrName after matches[1]
					// This matches TypeScript: matches[1] + attrName + matches[2] + matches[3]
					// When matches[1] is empty, result is attrName + ":" + matches[3]
					scopedP = matches[1] + attrName + matches[2] + normalizedAfter
				} else {
					// If no match, just prepend attrName
					scopedP = attrName + t
				}
			} else {
				// If t is empty after removing :host, just prepend attrName
				scopedP = attrName + p
			}
		}

		return scopedP
	}

	pseudoFunctionAwareScopeSelectorPart := func(selectorPart string) string {
		scopedPart := ""

		// Check if the selector consists ONLY of top-level :where() and :is() selectors
		// We need to scan the string manually because regex FindAllStringIndex finds nested matches too
		isPurePseudo := true
		pseudoSelectorParts := []string{}
		cursor := 0

		for cursor < len(selectorPart) {
			// Skip leading whitespace if any (though splitSelectorByComma usually trims)
			// But here we want to know if there are characters that are NOT part of the pseudo selector

			// Check if starts with :where( or :is(
			remaining := selectorPart[cursor:]
			var match string
			if strings.HasPrefix(remaining, ":where(") {
				match = ":where("
			} else if strings.HasPrefix(remaining, ":is(") {
				match = ":is("
			} else {
				// Found something that is not a pseudo selector start
				isPurePseudo = false
				break
			}

			// Found a match, now find the closing parenthesis
			start := cursor
			cursor += len(match)
			openedBrackets := 1

			for cursor < len(selectorPart) {
				char := selectorPart[cursor]
				cursor++
				if char == '(' {
					openedBrackets++
				} else if char == ')' {
					openedBrackets--
					if openedBrackets == 0 {
						break
					}
				}
			}

			if openedBrackets > 0 {
				// Unbalanced parentheses
				isPurePseudo = false
				break
			}

			pseudoSelectorParts = append(pseudoSelectorParts, selectorPart[start:cursor])
		}

		// If selector consists of only :where() and :is() on the outer level
		if isPurePseudo && len(pseudoSelectorParts) > 0 {
			scopedParts := []string{}
			for _, part := range pseudoSelectorParts {
				cssPrefixWithPseudoSelectorFunction := regexp.MustCompile(`^:(?:where|is)\(`)
				match := cssPrefixWithPseudoSelectorFunction.FindString(part)
				if match != "" && len(part) > len(match) && part[len(part)-1] == ')' {
					// Unwrap the pseudo selector to scope its contents.
					// For example,
					// - `:where(selectorToScope)` -> `selectorToScope`;
					// - `:is(.foo, .bar)` -> `.foo, .bar`.
					selectorToScope := part[len(match) : len(part)-1]
					if strings.Contains(selectorToScope, polyfillHostNoCombinator) {
						scopedIndicator := true
						sc.shouldScopeIndicator = &scopedIndicator
					}
					shouldScope := false
					if sc.shouldScopeIndicator != nil {
						shouldScope = *sc.shouldScopeIndicator
					}
					scopedInnerPart := sc.scopeSelector(scopeSelector, hostSelector, selectorToScope, shouldScope)
					scopedParts = append(scopedParts, match+scopedInnerPart+")")
				}
			}
			scopedPart = strings.Join(scopedParts, "")
		} else {
			// Update _shouldScopeIndicator: if selectorPart contains polyfillHostNoCombinator, set it to true
			// This matches TypeScript: this._shouldScopeIndicator = this._shouldScopeIndicator || selectorPart.includes(_polyfillHostNoCombinator);
			// In TypeScript:
			// - If _shouldScopeIndicator is undefined, it's falsy, so result is selectorPart.includes(...)
			// - If _shouldScopeIndicator is false, result is selectorPart.includes(...)
			// - If _shouldScopeIndicator is true, result is true
			hasPolyfillHost := strings.Contains(selectorPart, polyfillHostNoCombinator)
			if sc.shouldScopeIndicator == nil {
				// If undefined (falsy), set to hasPolyfillHost
				if hasPolyfillHost {
					sc.shouldScopeIndicator = &hasPolyfillHost
				}
			} else {
				// If already set, use OR logic: keep true if already true, or set to true if hasPolyfillHost
				*sc.shouldScopeIndicator = *sc.shouldScopeIndicator || hasPolyfillHost
			}

			// Only scope if _shouldScopeIndicator is true
			// This matches TypeScript: scopedPart = this._shouldScopeIndicator ? _scopeSelectorPart(selectorPart) : selectorPart;
			// In TypeScript, if _shouldScopeIndicator is undefined or false, selectorPart is returned as-is
			// Note: _shouldScopeIndicator is set based on whether selectorPart contains _polyfillHostNoCombinator
			// After convertColonHost, :host is converted to -shadowcsshost-no-combinator, so if selector contains :host,
			// it will contain _polyfillHostNoCombinator and _shouldScopeIndicator will be true
			if sc.shouldScopeIndicator != nil && *sc.shouldScopeIndicator {
				scopedPart = scopeSelectorPart(selectorPart)
			} else {
				scopedPart = selectorPart
			}
		}

		return scopedPart
	}

	var oldSafeSelector *SafeSelector
	if isParentSelector {
		oldSafeSelector = sc.safeSelector
		sc.safeSelector = NewSafeSelector(selector)
		selector = sc.safeSelector.Content()
	}

	// Split by combinators (>, space, +, ~)
	// Pattern: ( |>|\+|~(?!=))(?!([^)(]*(?:\([^)(]*(?:\([^)(]*(?:\([^)(]*\)[^)(]*)*\)[^)(]*)*\)[^)(]*)*\)))
	// Go doesn't support negative lookahead, so we'll manually check parentheses
	// Split by combinators but respect parentheses (similar to splitSelectorByComma)
	scopedSelector := ""
	startIndex := 0
	parens := 0

	hasHost := strings.Contains(selector, polyfillHostNoCombinator)
	// Only scope parts after or on the same level as the first `-shadowcsshost-no-combinator`
	// when it is present. The selector has the same level when it is a part of a pseudo
	// selector, like `:where()`, for example `:where(:host, .foo)` would result in `.foo`
	// being scoped.
	// In TypeScript: if (isParentSelector || this._shouldScopeIndicator)
	// This means: if isParentSelector is true OR _shouldScopeIndicator is truthy (not undefined and not false)
	if isParentSelector || (sc.shouldScopeIndicator != nil && *sc.shouldScopeIndicator) {
		shouldScope := !hasHost
		sc.shouldScopeIndicator = &shouldScope
	}

	for i := 0; i < len(selector); i++ {
		char := selector[i]
		if char == '(' {
			parens++
		} else if char == ')' {
			parens--
		} else if parens == 0 {
			// Check for combinators: >, space, +, ~
			// Note: ~(?!=) means ~ not followed by = (to avoid matching ~= in attribute selectors)
			isCombinator := false
			separator := ""
			separatorLen := 0

			if char == '>' {
				isCombinator = true
				separator = ">"
				separatorLen = 1
			} else if char == '+' {
				isCombinator = true
				separator = "+"
				separatorLen = 1
			} else if char == '~' {
				// Check if it's ~ not followed by =
				if i+1 < len(selector) && selector[i+1] != '=' {
					isCombinator = true
					separator = "~"
					separatorLen = 1
				}
			} else if char == ' ' {
				// Space is a combinator, but we need to check if it's not inside parentheses
				// and not part of escaped hex value
				isCombinator = true
				separator = " "
				separatorLen = 1
			}

			if isCombinator {
				part := selector[startIndex:i]

				// Check for escaped hex value
				escapedHexRe := regexp.MustCompile(`__esc-ph-(\d+)__`)
				if escapedHexRe.MatchString(part) && i+1 < len(selector) {
					nextChar := selector[i+1]
					if (nextChar >= 'a' && nextChar <= 'f') || (nextChar >= 'A' && nextChar <= 'F') || (nextChar >= '0' && nextChar <= '9') {
						// This is not a separator, continue
						continue
					}
				}

				scopedPart := pseudoFunctionAwareScopeSelectorPart(part)
				scopedSelector += scopedPart + " " + separator + " "
				startIndex = i + separatorLen
			}
		}
	}

	part := selector[startIndex:]
	scopedSelector += pseudoFunctionAwareScopeSelectorPart(part)

	// Restore placeholders
	var result string
	if sc.safeSelector != nil {
		result = sc.safeSelector.Restore(scopedSelector)
	} else {
		result = scopedSelector
	}

	if isParentSelector {
		sc.safeSelector = oldSafeSelector
	}

	return result
}

func (sc *ShadowCss) insertPolyfillHostInCssText(selector string) string {
	result := colonHostContextRe.ReplaceAllString(selector, polyfillHostContext)
	result = colonHostRe.ReplaceAllString(result, polyfillHost)
	return result
}

// SafeSelector handles safe selector processing
type SafeSelector struct {
	placeholders []string
	index        int
	content      string
}

// NewSafeSelector creates a new SafeSelector
func NewSafeSelector(selector string) *SafeSelector {
	ss := &SafeSelector{
		placeholders: []string{},
		index:        0,
	}

	// Replace attribute selectors with placeholders
	attrRe := regexp.MustCompile(`(\[[^\]]*\])`)
	selector = ss.escapeRegexMatches(selector, attrRe)

	// Replace escape sequences
	escapeRe := regexp.MustCompile(`(\\.)`)
	selector = escapeRe.ReplaceAllStringFunc(selector, func(match string) string {
		replaceBy := fmt.Sprintf("__esc-ph-%d__", ss.index)
		ss.placeholders = append(ss.placeholders, match)
		ss.index++
		return replaceBy
	})

	// Replace nth-child expressions
	// The regex (:nth-[-\w]+)(\([^)]+\)) doesn't handle nested parentheses
	// We need to scan manually
	nthRe := regexp.MustCompile(`:nth-[-\w]+\(`)

	// We need to loop until no more matches are found
	for {
		matchLoc := nthRe.FindStringIndex(selector)
		if matchLoc == nil {
			break
		}

		start := matchLoc[0]
		// Find the matching closing parenthesis
		parens := 1
		end := -1
		for i := matchLoc[1]; i < len(selector); i++ {
			if selector[i] == '(' {
				parens++
			} else if selector[i] == ')' {
				parens--
				if parens == 0 {
					end = i + 1
					break
				}
			}
		}

		if end != -1 {
			fullMatch := selector[start:end]
			// Extract the pseudo part (e.g. :nth-child) and the expression part (e.g. (3n+1))
			parenIdx := strings.Index(fullMatch, "(")
			pseudo := fullMatch[:parenIdx]
			exp := fullMatch[parenIdx:]

			replaceBy := fmt.Sprintf("__ph-%d__", ss.index)
			ss.placeholders = append(ss.placeholders, exp)
			ss.index++

			selector = selector[:start] + pseudo + replaceBy + selector[end:]
		} else {
			// Unbalanced or no closing paren, skip this match to avoid infinite loop
			// In a real compiler we might want to error, but here we just break or skip
			break
		}
	}

	ss.content = selector
	return ss
}

func (ss *SafeSelector) escapeRegexMatches(content string, pattern *regexp.Regexp) string {
	return pattern.ReplaceAllStringFunc(content, func(match string) string {
		replaceBy := fmt.Sprintf("__ph-%d__", ss.index)
		ss.placeholders = append(ss.placeholders, match)
		ss.index++
		return replaceBy
	})
}

// Restore restores placeholders in content
func (ss *SafeSelector) Restore(content string) string {
	phRe := regexp.MustCompile(`__(?:ph|esc-ph)-(\d+)__`)
	return phRe.ReplaceAllStringFunc(content, func(match string) string {
		matches := phRe.FindStringSubmatch(match)
		if len(matches) >= 2 {
			idx := 0
			fmt.Sscanf(matches[1], "%d", &idx)
			if idx < len(ss.placeholders) {
				return ss.placeholders[idx]
			}
		}
		return match
	})
}

// Content returns the content
func (ss *SafeSelector) Content() string {
	return ss.content
}

// CssRule represents a CSS rule
type CssRule struct {
	Selector string
	Content  string
}

// NewCssRule creates a new CssRule
func NewCssRule(selector string, content string) *CssRule {
	return &CssRule{
		Selector: selector,
		Content:  content,
	}
}

// ProcessRules processes CSS rules
type RuleCallback func(rule *CssRule) *CssRule

// ProcessRules processes CSS input with a rule callback
func ProcessRules(input string, ruleCallback RuleCallback) string {
	escaped := escapeInStrings(input)
	inputWithEscapedBlocks := escapeBlocks(escaped, contentPairs, blockPlaceholder)
	nextBlockIndex := 0

	ruleRe := regexp.MustCompile(fmt.Sprintf(`(\s*(?:%s\s*)*)([^;\{\}]+?)(\s*)((?:{%s}?\s*;?)|(?:\s*;))`, commentPlaceholder, blockPlaceholder))

	escapedResult := ruleRe.ReplaceAllStringFunc(inputWithEscapedBlocks.escapedString, func(match string) string {
		matches := ruleRe.FindStringSubmatch(match)
		if len(matches) < 5 {
			return match
		}
		selector := matches[2]
		suffix := matches[4]
		content := ""
		contentPrefix := ""

		if strings.HasPrefix(suffix, "{"+blockPlaceholder) {
			if nextBlockIndex < len(inputWithEscapedBlocks.blocks) {
				content = inputWithEscapedBlocks.blocks[nextBlockIndex]
				nextBlockIndex++
			}
			suffix = suffix[len(blockPlaceholder)+1:]
			contentPrefix = "{"
		}

		rule := ruleCallback(NewCssRule(selector, content))
		result := matches[1] + rule.Selector + matches[3] + contentPrefix + rule.Content + suffix

		return result
	})

	return unescapeInStrings(escapedResult)
}

// StringWithEscapedBlocks represents a string with escaped blocks
type StringWithEscapedBlocks struct {
	escapedString string
	blocks        []string
}

func escapeBlocks(input string, charPairs map[string]string, placeholder string) *StringWithEscapedBlocks {
	resultParts := []string{}
	escapedBlocks := []string{}
	openCharCount := 0
	nonBlockStartIndex := 0
	blockStartIndex := -1
	var openChar, closeChar string

	for i := 0; i < len(input); i++ {
		char := input[i]
		if char == '\\' {
			i++
		} else if closeChar != "" && char == closeChar[0] {
			openCharCount--
			if openCharCount == 0 {
				escapedBlocks = append(escapedBlocks, input[blockStartIndex:i])
				resultParts = append(resultParts, placeholder)
				nonBlockStartIndex = i
				blockStartIndex = -1
				openChar = ""
				closeChar = ""
			}
		} else if openChar != "" && char == openChar[0] {
			openCharCount++
		} else if openCharCount == 0 {
			if closeCharForOpen, ok := charPairs[string(char)]; ok {
				openChar = string(char)
				closeChar = closeCharForOpen
				openCharCount = 1
				blockStartIndex = i + 1
				resultParts = append(resultParts, input[nonBlockStartIndex:blockStartIndex])
			}
		}
	}

	if blockStartIndex != -1 {
		escapedBlocks = append(escapedBlocks, input[blockStartIndex:])
		resultParts = append(resultParts, placeholder)
	} else {
		resultParts = append(resultParts, input[nonBlockStartIndex:])
	}

	return &StringWithEscapedBlocks{
		escapedString: strings.Join(resultParts, ""),
		blocks:        escapedBlocks,
	}
}

const (
	commaInPlaceholder = "%COMMA_IN_PLACEHOLDER%"
	semiInPlaceholder  = "%SEMI_IN_PLACEHOLDER%"
	colonInPlaceholder = "%COLON_IN_PLACEHOLDER%"
)

var (
	cssCommaInPlaceholderReGlobal = regexp.MustCompile(commaInPlaceholder)
	cssSemiInPlaceholderReGlobal  = regexp.MustCompile(semiInPlaceholder)
	cssColonInPlaceholderReGlobal = regexp.MustCompile(colonInPlaceholder)
)

func escapeInStrings(input string) string {
	result := []rune(input)
	currentQuoteChar := rune(0)

	for i := 0; i < len(result); i++ {
		char := result[i]
		if char == '\\' {
			i++
		} else {
			if currentQuoteChar != 0 {
				if char == currentQuoteChar {
					currentQuoteChar = 0
				} else {
					var placeholder string
					switch char {
					case ';':
						placeholder = semiInPlaceholder
					case ',':
						placeholder = commaInPlaceholder
					case ':':
						placeholder = colonInPlaceholder
					}
					if placeholder != "" {
						// Replace character with placeholder
						newResult := string(result[:i]) + placeholder + string(result[i+1:])
						result = []rune(newResult)
						i += len(placeholder) - 1
					}
				}
			} else if char == '\'' || char == '"' {
				currentQuoteChar = char
			}
		}
	}

	return string(result)
}

func unescapeInStrings(input string) string {
	result := cssCommaInPlaceholderReGlobal.ReplaceAllString(input, ",")
	result = cssSemiInPlaceholderReGlobal.ReplaceAllString(result, ";")
	result = cssColonInPlaceholderReGlobal.ReplaceAllString(result, ":")
	return result
}

func unescapeQuotes(str string, isQuoted bool) string {
	if !isQuoted {
		return str
	}

	var sb strings.Builder
	sb.Grow(len(str))

	i := 0
	for i < len(str) {
		char := str[i]
		if char == '\\' {
			// Count consecutive backslashes
			backslashCount := 1
			j := i + 1
			for j < len(str) && str[j] == '\\' {
				backslashCount++
				j++
			}

			// Check what follows
			if j < len(str) && (str[j] == '"' || str[j] == '\'') {
				// Followed by a quote
				if backslashCount%2 == 1 {
					// Odd number of backslashes: the last one escapes the quote
					// Keep N-1 backslashes
					sb.WriteString(strings.Repeat("\\", backslashCount-1))
					// Append the quote
					sb.WriteByte(str[j])
					i = j + 1
					continue
				} else {
					// Even number of backslashes: quote is not escaped by them
					// Keep all backslashes
					sb.WriteString(strings.Repeat("\\", backslashCount))
					// Append the quote
					sb.WriteByte(str[j])
					i = j + 1
					continue
				}
			} else {
				// Not followed by a quote (or end of string)
				// Keep all backslashes
				sb.WriteString(strings.Repeat("\\", backslashCount))
				i = j
				continue
			}
		} else {
			sb.WriteByte(char)
			i++
		}
	}

	return sb.String()
}

func combineHostContextSelectors(contextSelectors []string, otherSelectors string, pseudoPrefix string) string {
	hostMarker := polyfillHostNoCombinator
	otherSelectorsHasHost := polyfillHostRe.MatchString(otherSelectors)

	if len(contextSelectors) == 0 {
		return hostMarker + otherSelectors
	}

	combined := []string{contextSelectors[len(contextSelectors)-1]}
	remaining := contextSelectors[:len(contextSelectors)-1]

	for len(remaining) > 0 {
		length := len(combined)
		contextSelector := remaining[len(remaining)-1]
		remaining = remaining[:len(remaining)-1]

		newCombined := make([]string, length*3)
		for i := 0; i < length; i++ {
			previousSelectors := combined[i]
			// Add as descendant
			newCombined[length*2+i] = previousSelectors + " " + contextSelector
			// Add as ancestor
			newCombined[length+i] = contextSelector + " " + previousSelectors
			// Add on same element
			newCombined[i] = contextSelector + previousSelectors
		}
		combined = newCombined
	}

	result := []string{}
	for _, s := range combined {
		if otherSelectorsHasHost {
			result = append(result, pseudoPrefix+s+otherSelectors)
		} else {
			result = append(result, pseudoPrefix+s+hostMarker+otherSelectors)
			result = append(result, pseudoPrefix+s+" "+hostMarker+otherSelectors)
		}
	}
	return strings.Join(result, ", ")
}

// RepeatGroups repeats groups in place
func RepeatGroups(groups *[][]string, multiples int) {
	length := len(*groups)
	for i := 1; i < multiples; i++ {
		for j := 0; j < length; j++ {
			newGroup := make([]string, len((*groups)[j]))
			copy(newGroup, (*groups)[j])
			*groups = append(*groups, newGroup)
		}
	}
}

// Constants
const (
	polyfillHost             = "-shadowcsshost"
	polyfillHostContext      = "-shadowcsscontext"
	polyfillHostNoCombinator = "-shadowcsshost-no-combinator"
	commentPlaceholder       = "%COMMENT%"
	blockPlaceholder         = "%BLOCK%"
	selectorReSuffix         = `([>\s~+[.,{:][\s\S]*)?$`
)

var (
	polyfillHostRe             = regexp.MustCompile(`-shadowcsshost`)
	colonHostRe                = regexp.MustCompile(`:host`)
	colonHostContextRe         = regexp.MustCompile(`:host-context`)
	polyfillHostNoCombinatorRe = regexp.MustCompile(`-shadowcsshost-no-combinator([^\s,]*)`)
	shadowDOMSelectorsRe       = []*regexp.Regexp{
		regexp.MustCompile(`::shadow`),
		regexp.MustCompile(`::content`),
		regexp.MustCompile(`/shadow-deep/`),
		regexp.MustCompile(`/shadow/`),
	}
	shadowDeepSelectors = regexp.MustCompile(`(?:>>>)|(?:\/deep\/)|(?:::ng-deep)`)
	contentPairs        = map[string]string{
		"{": "}",
	}
)

// isPolyfillHostNoCombinatorOutsidePseudoFunction checks if the string contains
// `-shadowcsshost-no-combinator` that is NOT inside a pseudo function (i.e., not followed by `(...)`)
// This replaces the TypeScript regex: `-shadowcsshost-no-combinator(?![^(]*\\))`
// Pattern `(?![^(]*\\))` means: not followed by zero or more non-`(` characters, then `)`
// In other words, it matches if it's NOT followed by `(` then some non-`(` chars, then `)`
// The regex matches if the pattern `[^(]*\\)` does NOT match after `polyfillHostNoCombinator`
func isPolyfillHostNoCombinatorOutsidePseudoFunction(s string) bool {
	idx := strings.Index(s, polyfillHostNoCombinator)
	if idx == -1 {
		return false
	}
	// Check if it's followed by a pseudo function pattern: `(...)`
	after := s[idx+len(polyfillHostNoCombinator):]

	// Pattern `(?![^(]*\\))` means: not followed by `(` then some non-`(` chars, then `)`
	// We need to check if `[^(]*\)` matches.
	// `[^(]*` matches zero or more non-`(` chars.
	// So we look for the first `)`. If we find one, and there are no `(` before it, then it matches.

	openIdx := strings.Index(after, "(")
	closeIdx := strings.Index(after, ")")

	if closeIdx != -1 {
		// Found a closing paren.
		if openIdx == -1 || closeIdx < openIdx {
			// No opening paren, or closing paren comes first.
			// This means `[^(]*\)` matches.
			// So negative lookahead fails.
			// So it is INSIDE.
			return false
		}
	}

	// If no closing paren, or opening paren comes first, then `[^(]*\)` does not match.
	// So negative lookahead succeeds.
	// So it is OUTSIDE.
	return true
}
