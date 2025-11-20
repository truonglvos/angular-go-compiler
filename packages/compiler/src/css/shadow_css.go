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
	// Note: Go regexp doesn't support lookahead (?=), so we match trailing comma, space, or end explicitly
	// Pattern: (^|\s+|,)(?:(?:(['"])((?:\\\\|\\\2|(?!\2).)+)\2)|(-?[A-Za-z][\w\-]*))(?=[,\s]|$)
	// We'll match: (^|\s+|,)(?:(?:(['"])((?:\\\\|\\\2|(?!\2).)+)\2)|(-?[A-Za-z][\w\-]*))([,\s]|$)
	animationRe := regexp.MustCompile(`(^|\s+|,)(?:(?:(['"])((?:\\\\|\\\2|(?!\2).)+)\2)|(-?[A-Za-z][\w\-]*))([,\s]|$)`)

	return &ShadowCss{
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
	// Pattern: ^(\s*)(['"]?)(.+?)\2(\s*)$
	keyframeRe := regexp.MustCompile(`^(\s*)(['"]?)(.+?)(['"]?)(\s*)$`)

	return keyframeRe.ReplaceAllStringFunc(keyframe, func(match string) string {
		matches := keyframeRe.FindStringSubmatch(match)
		if len(matches) < 6 {
			return match
		}
		spaces1 := matches[1]
		quote1 := matches[2]
		name := matches[3]
		quote2 := matches[4]
		spaces2 := matches[5]

		isQuoted := (quote1 == `"` || quote1 == `'`) && quote1 == quote2
		unescapedName := unescapeQuotes(name, isQuoted)

		prefix := ""
		if unscopedKeyframesSet[unescapedName] {
			prefix = scopeSelector + "_"
		}

		quote := quote1
		if quote == "" {
			quote = quote2
		}
		return fmt.Sprintf("%s%s%s%s%s%s", spaces1, quote, prefix, name, quote, spaces2)
	})
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
		// Note: Go regexp doesn't support lookahead, so we match trailing comma, space, or end explicitly
		// Pattern: (^|\s+|,)(?:(?:(['"])((?:\\\\|\\\2|(?!\2).)+)\2)|(-?[A-Za-z][\w\-]*))(?=[,\s]|$)
		// We'll use: (^|\s+|,)(?:(?:(['"])((?:\\\\|\\\2|(?!\2).)+)\2)|(-?[A-Za-z][\w\-]*))([,\s]|$)
		keyframeRe := regexp.MustCompile(`(^|\s+|,)(?:(?:(['"])((?:\\\\|\\\2|(?!\2).)+)\2)|(-?[A-Za-z][\w\-]*))([,\s]|$)`)

		processed := keyframeRe.ReplaceAllStringFunc(animationDeclarations, func(keyframeMatch string) string {
			keyframeMatches := keyframeRe.FindStringSubmatch(keyframeMatch)
			if len(keyframeMatches) < 7 {
				return keyframeMatch
			}
			leadingSpaces := keyframeMatches[1]
			quote1 := keyframeMatches[2]
			quotedName := keyframeMatches[3]
			nonQuotedName := keyframeMatches[4]
			trailing := keyframeMatches[5]

			if quotedName != "" {
				quote := quote1
				quotedKeyframe := fmt.Sprintf("%s%s%s", quote, quotedName, quote)
				scoped := sc.scopeAnimationKeyframe(quotedKeyframe, scopeSelector, unscopedKeyframesSet)
				return leadingSpaces + scoped + trailing
			} else if nonQuotedName != "" {
				if animationKeywords[nonQuotedName] {
					return keyframeMatch
				}
				scoped := sc.scopeAnimationKeyframe(nonQuotedName, scopeSelector, unscopedKeyframesSet)
				return leadingSpaces + scoped + trailing
			}
			return keyframeMatch
		})

		return start + processed
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
			scopedKeyframes[i] = sc.scopeAnimationKeyframe(strings.TrimSpace(keyframe), scopeSelector, unscopedKeyframesSet)
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
	// Go doesn't support backreferences, so we match double-quoted and single-quoted separately
	cssContentNextSelectorRe := regexp.MustCompile(`polyfill-next-selector[^}]*content:[\s]*?(["'])(.*?)\1[;\s]*}([^{]*?){`)

	return cssContentNextSelectorRe.ReplaceAllStringFunc(cssText, func(match string) string {
		matches := cssContentNextSelectorRe.FindStringSubmatch(match)
		if len(matches) >= 4 {
			return matches[2] + "{"
		}
		return match
	})
}

func (sc *ShadowCss) insertPolyfillRulesInCssText(cssText string) string {
	// Pattern: (polyfill-rule)[^}]*(content:[\s]*(['"])(.*?)\3)[;\s]*[^}]*}
	// Go doesn't support backreferences, so we match double-quoted and single-quoted separately
	cssContentRuleRe := regexp.MustCompile(`(polyfill-rule)[^}]*(content:[\s]*(["'])(.*?)\3)[;\s]*[^}]*}`)

	return cssContentRuleRe.ReplaceAllStringFunc(cssText, func(match string) string {
		matches := cssContentRuleRe.FindStringSubmatch(match)
		if len(matches) >= 5 {
			rule := match
			rule = strings.Replace(rule, matches[1], "", 1)
			rule = strings.Replace(rule, matches[2], "", 1)
			return matches[4] + rule
		}
		return match
	})
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
	// Go doesn't support backreferences, so we match double-quoted and single-quoted separately
	cssContentUnscopedRuleRe := regexp.MustCompile(`(polyfill-unscoped-rule)[^}]*(content:[\s]*(["'])(.*?)\3)[;\s]*[^}]*}`)

	result := ""
	matches := cssContentUnscopedRuleRe.FindAllStringSubmatch(cssText, -1)
	for _, match := range matches {
		if len(match) >= 5 {
			rule := match[0]
			rule = strings.Replace(rule, match[2], "", 1)
			rule = strings.Replace(rule, match[1], match[4], 1)
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

	return strings.Join(scopedParts, ", ")
}

func (sc *ShadowCss) splitSelectorByComma(selector string) []string {
	// Simple implementation: split by comma, but respect parentheses
	// This is a simplified version - the full TypeScript version uses negative lookahead
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
			result = append(result, selector[start:i])
			start = i + 1
		}
	}
	result = append(result, selector[start:])
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

	scopeSelectorPart := func(p string) string {
		scopedP := strings.TrimSpace(p)
		if scopedP == "" {
			return p
		}

		if strings.Contains(p, polyfillHostNoCombinator) {
			scopedP = sc.applySimpleSelectorScope(p, scopeSelector, hostSelector)
			if !polyfillHostNoCombinatorOutsidePseudoFunction.MatchString(p) {
				// Pattern: ([^:]*)(:*)([\s\S]*)
				selRe := regexp.MustCompile(`([^:]*)(:*)([\s\S]*)`)
				matches := selRe.FindStringSubmatch(scopedP)
				if len(matches) >= 4 {
					scopedP = matches[1] + attrName + matches[2] + matches[3]
				}
			}
		} else {
			// Remove :host
			t := polyfillHostRe.ReplaceAllString(p, "")
			if len(t) > 0 {
				selRe := regexp.MustCompile(`([^:]*)(:*)([\s\S]*)`)
				matches := selRe.FindStringSubmatch(t)
				if len(matches) >= 4 {
					scopedP = matches[1] + attrName + matches[2] + matches[3]
				}
			}
		}

		return scopedP
	}

	pseudoFunctionAwareScopeSelectorPart := func(selectorPart string) string {
		scopedPart := ""

		// Collect all outer :where() and :is() selectors
		pseudoSelectorParts := []string{}
		cssPrefixWithPseudoSelectorFunction := regexp.MustCompile(`:(?:where|is)\(`)
		matches := cssPrefixWithPseudoSelectorFunction.FindAllStringIndex(selectorPart, -1)

		for _, match := range matches {
			start := match[0]
			index := match[1]
			openedBrackets := 1

			for index < len(selectorPart) {
				currentSymbol := selectorPart[index]
				index++
				if currentSymbol == '(' {
					openedBrackets++
				} else if currentSymbol == ')' {
					openedBrackets--
					if openedBrackets == 0 {
						break
					}
				}
			}

			pseudoSelectorParts = append(pseudoSelectorParts, selectorPart[start:index])
		}

		// If selector consists of only :where() and :is() on the outer level
		if strings.Join(pseudoSelectorParts, "") == selectorPart {
			scopedParts := []string{}
			for _, part := range pseudoSelectorParts {
				cssPrefixWithPseudoSelectorFunction := regexp.MustCompile(`:(?:where|is)\(`)
				match := cssPrefixWithPseudoSelectorFunction.FindString(part)
				if match != "" {
					selectorToScope := part[len(match) : len(part)-1]
					if strings.Contains(selectorToScope, polyfillHostNoCombinator) {
						scopedIndicator := true
						sc.shouldScopeIndicator = &scopedIndicator
					}
					scopedInnerPart := sc.scopeSelector(scopeSelector, hostSelector, selectorToScope, false)
					scopedParts = append(scopedParts, match+scopedInnerPart+")")
				}
			}
			scopedPart = strings.Join(scopedParts, "")
		} else {
			shouldScope := sc.shouldScopeIndicator != nil && *sc.shouldScopeIndicator
			if strings.Contains(selectorPart, polyfillHostNoCombinator) {
				shouldScope = true
			}
			if sc.shouldScopeIndicator == nil {
				sc.shouldScopeIndicator = &shouldScope
			} else {
				*sc.shouldScopeIndicator = *sc.shouldScopeIndicator || shouldScope
			}

			if shouldScope {
				scopedPart = scopeSelectorPart(selectorPart)
			} else {
				scopedPart = selectorPart
			}
		}

		return scopedPart
	}

	if isParentSelector {
		sc.safeSelector = NewSafeSelector(selector)
		selector = sc.safeSelector.Content()
	}

	// Split by combinators (>, space, +, ~)
	// Pattern: ( |>|\+|~(?!=))(?!([^)(]*(?:\([^)(]*(?:\([^)(]*(?:\([^)(]*\)[^)(]*)*\)[^)(]*)*\)[^)(]*)*\)))
	// Go doesn't support negative lookahead, so we'll use a simpler approach
	sep := regexp.MustCompile(`( |>|\+|~)`)

	scopedSelector := ""
	startIndex := 0
	allMatches := sep.FindAllStringIndex(selector, -1)

	hasHost := strings.Contains(selector, polyfillHostNoCombinator)
	if isParentSelector || (sc.shouldScopeIndicator != nil && *sc.shouldScopeIndicator) {
		shouldScope := !hasHost
		sc.shouldScopeIndicator = &shouldScope
	}

	for _, match := range allMatches {
		separator := selector[match[0]:match[1]]
		part := selector[startIndex:match[0]]

		// Check for escaped hex value
		escapedHexRe := regexp.MustCompile(`__esc-ph-(\d+)__`)
		if escapedHexRe.MatchString(part) && match[1] < len(selector) {
			nextChar := selector[match[1]]
			if (nextChar >= 'a' && nextChar <= 'f') || (nextChar >= 'A' && nextChar <= 'F') || (nextChar >= '0' && nextChar <= '9') {
				continue
			}
		}

		scopedPart := pseudoFunctionAwareScopeSelectorPart(part)
		scopedSelector += scopedPart + " " + separator + " "
		startIndex = match[1]
	}

	part := selector[startIndex:]
	scopedSelector += pseudoFunctionAwareScopeSelectorPart(part)

	// Restore placeholders
	if sc.safeSelector != nil {
		return sc.safeSelector.Restore(scopedSelector)
	}
	return scopedSelector
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
	nthRe := regexp.MustCompile(`(:nth-[-\w]+)(\([^)]+\))`)
	ss.content = nthRe.ReplaceAllStringFunc(selector, func(match string) string {
		matches := nthRe.FindStringSubmatch(match)
		if len(matches) >= 3 {
			pseudo := matches[1]
			exp := matches[2]
			replaceBy := fmt.Sprintf("__ph-%d__", ss.index)
			ss.placeholders = append(ss.placeholders, exp)
			ss.index++
			return pseudo + replaceBy
		}
		return match
	})

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
		return matches[1] + rule.Selector + matches[3] + contentPrefix + rule.Content + suffix
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
	// Pattern: ((?:^|[^\\])(?:\\\\)*)\\(?=['"])
	unescapeRe := regexp.MustCompile(`((?:^|[^\\])(?:\\\\)*)\\(['"])`)
	return unescapeRe.ReplaceAllString(str, "${1}${2}")
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
	polyfillHostRe                                = regexp.MustCompile(`-shadowcsshost`)
	colonHostRe                                   = regexp.MustCompile(`:host`)
	colonHostContextRe                            = regexp.MustCompile(`:host-context`)
	polyfillHostNoCombinatorRe                    = regexp.MustCompile(`-shadowcsshost-no-combinator([^\s,]*)`)
	polyfillHostNoCombinatorOutsidePseudoFunction = regexp.MustCompile(`-shadowcsshost-no-combinator(?!\([^)]*\))`)
	shadowDOMSelectorsRe                          = []*regexp.Regexp{
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
