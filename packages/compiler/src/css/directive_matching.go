package css

import (
	"fmt"
	"regexp"
	"strings"
)

// SelectorRegexp represents the regex group indices for selector parsing
type SelectorRegexp int

const (
	SelectorRegexpAll SelectorRegexp = iota
	SelectorRegexpNot
	SelectorRegexpTag
	SelectorRegexpPrefix
	SelectorRegexpAttribute
	SelectorRegexpAttributeDoubleQuote
	SelectorRegexpAttributeSingleQuote
	SelectorRegexpAttributeUnquotedValue
	SelectorRegexpAttributeNoValue
	SelectorRegexpNotEnd
	SelectorRegexpSeparator
)

// selectorRegexp matches CSS selector patterns
// Note: Go regexp doesn't support backreferences, so we match double-quoted, single-quoted, and unquoted values separately
var selectorRegexp = regexp.MustCompile(
	`(\:not\()|` + // 1: ":not("
		`(([\.\#]?)[-\w]+)|` + // 2: "tag"; 3: "."/"#";
		// "-" should appear first in the regexp below as FF31 parses "[.-\w]" as a range
		// 4: attribute; 5: attribute_string (double quote); 6: attribute_value (double quote)
		// 7: attribute_string (single quote); 8: attribute_value (single quote)
		// 9: attribute_string (unquoted); 10: attribute_value (unquoted)
		// 11: attribute (no value)
		`(?:\[([-.\w*\\$]+)(?:=(")([^\]"]*)"|(')([^\]']*)'|(=)([^\]\s]+)|())\])|` + // "[name]", "[name=value]", "[name="value"]", "[name='value']"
		`(\))|` + // 12: ")"
		`(\s*,\s*)`, // 13: ","
)

// CssSelector represents a CSS selector
type CssSelector struct {
	Element      *string
	ClassNames   []string
	Attrs        []string // Pairs: [name, value, name, value, ...]
	NotSelectors []*CssSelector
}

// NewCssSelector creates a new CssSelector
func NewCssSelector() *CssSelector {
	return &CssSelector{
		ClassNames:   []string{},
		Attrs:        []string{},
		NotSelectors: []*CssSelector{},
	}
}

// ParseCssSelector parses a CSS selector string into CssSelector array
func ParseCssSelector(selector string) ([]*CssSelector, error) {
	results := []*CssSelector{}
	
	addResult := func(res []*CssSelector, cssSel *CssSelector) []*CssSelector {
		if len(cssSel.NotSelectors) > 0 &&
			cssSel.Element == nil &&
			len(cssSel.ClassNames) == 0 &&
			len(cssSel.Attrs) == 0 {
			star := "*"
			cssSel.Element = &star
		}
		return append(res, cssSel)
	}
	
	cssSelector := NewCssSelector()
	current := cssSelector
	inNot := false
	
	matches := selectorRegexp.FindAllStringSubmatch(selector, -1)
	for _, match := range matches {
		if len(match) > int(SelectorRegexpNot) && match[SelectorRegexpNot] != "" {
			if inNot {
				return nil, fmt.Errorf("nesting :not in a selector is not allowed")
			}
			inNot = true
			current = NewCssSelector()
			cssSelector.NotSelectors = append(cssSelector.NotSelectors, current)
		}
		
		tag := ""
		if len(match) > int(SelectorRegexpTag) && match[SelectorRegexpTag] != "" {
			tag = match[SelectorRegexpTag]
		}
		
		if tag != "" {
			prefix := ""
			if len(match) > int(SelectorRegexpPrefix) {
				prefix = match[SelectorRegexpPrefix]
			}
			
			if prefix == "#" {
				// #hash
				id := tag[1:]
				current.AddAttribute("id", id)
			} else if prefix == "." {
				// Class
				className := tag[1:]
				current.AddClassName(className)
			} else {
				// Element
				current.SetElement(tag)
			}
		}
		
		// Check for attribute with double quote
		if len(match) > int(SelectorRegexpAttributeDoubleQuote) && match[SelectorRegexpAttributeDoubleQuote] == `"` {
			attribute := ""
			if len(match) > int(SelectorRegexpAttribute) {
				attribute = match[SelectorRegexpAttribute]
			}
			attrValue := ""
			if len(match) > int(SelectorRegexpAttributeUnquotedValue)+3 {
				attrValue = match[SelectorRegexpAttributeUnquotedValue+3]
			}
			if attribute != "" {
				unescapedAttr, err := current.UnescapeAttribute(attribute)
				if err != nil {
					return nil, err
				}
				current.AddAttribute(unescapedAttr, attrValue)
			}
		} else if len(match) > int(SelectorRegexpAttributeSingleQuote) && match[SelectorRegexpAttributeSingleQuote] == `'` {
			// Check for attribute with single quote
			attribute := ""
			if len(match) > int(SelectorRegexpAttribute) {
				attribute = match[SelectorRegexpAttribute]
			}
			attrValue := ""
			if len(match) > int(SelectorRegexpAttributeUnquotedValue)+5 {
				attrValue = match[SelectorRegexpAttributeUnquotedValue+5]
			}
			if attribute != "" {
				unescapedAttr, err := current.UnescapeAttribute(attribute)
				if err != nil {
					return nil, err
				}
				current.AddAttribute(unescapedAttr, attrValue)
			}
		} else if len(match) > int(SelectorRegexpAttributeNoValue) && match[SelectorRegexpAttributeNoValue] != "" {
			// Check for attribute with unquoted value or no value
			attribute := ""
			if len(match) > int(SelectorRegexpAttribute) {
				attribute = match[SelectorRegexpAttribute]
			}
			attrValue := ""
			if len(match) > int(SelectorRegexpAttributeUnquotedValue)+9 {
				attrValue = match[SelectorRegexpAttributeUnquotedValue+9]
			}
			if attribute != "" {
				unescapedAttr, err := current.UnescapeAttribute(attribute)
				if err != nil {
					return nil, err
				}
				current.AddAttribute(unescapedAttr, attrValue)
			}
		} else if len(match) > int(SelectorRegexpAttribute) && match[SelectorRegexpAttribute] != "" {
			// Attribute without value
			attribute := match[SelectorRegexpAttribute]
			unescapedAttr, err := current.UnescapeAttribute(attribute)
			if err != nil {
				return nil, err
			}
			current.AddAttribute(unescapedAttr, "")
		}
		
		if len(match) > int(SelectorRegexpNotEnd) && match[SelectorRegexpNotEnd] != "" {
			inNot = false
			current = cssSelector
		}
		
		if len(match) > int(SelectorRegexpSeparator) && match[SelectorRegexpSeparator] != "" {
			if inNot {
				return nil, fmt.Errorf("multiple selectors in :not are not supported")
			}
			results = addResult(results, cssSelector)
			cssSelector = NewCssSelector()
			current = cssSelector
		}
	}
	
	results = addResult(results, cssSelector)
	return results, nil
}

// UnescapeAttribute unescapes \$ sequences from the CSS attribute selector
func (cs *CssSelector) UnescapeAttribute(attr string) (string, error) {
	result := ""
	escaping := false
	for i := 0; i < len(attr); i++ {
		char := attr[i]
		if char == '\\' {
			escaping = true
			continue
		}
		if char == '$' && !escaping {
			return "", fmt.Errorf(`error in attribute selector "%s". unescaped "$" is not supported. please escape with "\\$"`, attr)
		}
		escaping = false
		result += string(char)
	}
	return result, nil
}

// EscapeAttribute escapes $ sequences from the CSS attribute selector
func (cs *CssSelector) EscapeAttribute(attr string) string {
	result := strings.ReplaceAll(attr, "\\", "\\\\")
	result = strings.ReplaceAll(result, "$", "\\$")
	return result
}

// IsElementSelector checks if this is an element selector
func (cs *CssSelector) IsElementSelector() bool {
	return cs.HasElementSelector() &&
		len(cs.ClassNames) == 0 &&
		len(cs.Attrs) == 0 &&
		len(cs.NotSelectors) == 0
}

// HasElementSelector checks if this selector has an element
func (cs *CssSelector) HasElementSelector() bool {
	return cs.Element != nil
}

// SetElement sets the element name
func (cs *CssSelector) SetElement(element string) {
	cs.Element = &element
}

// GetAttrs returns the attributes array
func (cs *CssSelector) GetAttrs() []string {
	result := []string{}
	if len(cs.ClassNames) > 0 {
		result = append(result, "class", strings.Join(cs.ClassNames, " "))
	}
	return append(result, cs.Attrs...)
}

// AddAttribute adds an attribute
func (cs *CssSelector) AddAttribute(name string, value string) {
	cs.Attrs = append(cs.Attrs, name)
	if value != "" {
		cs.Attrs = append(cs.Attrs, strings.ToLower(value))
	} else {
		cs.Attrs = append(cs.Attrs, "")
	}
}

// AddClassName adds a class name
func (cs *CssSelector) AddClassName(name string) {
	cs.ClassNames = append(cs.ClassNames, strings.ToLower(name))
}

// String returns the string representation of the selector
func (cs *CssSelector) String() string {
	res := ""
	if cs.Element != nil {
		res = *cs.Element
	}
	
	for _, klass := range cs.ClassNames {
		res += "." + klass
	}
	
	for i := 0; i < len(cs.Attrs); i += 2 {
		name := cs.EscapeAttribute(cs.Attrs[i])
		value := ""
		if i+1 < len(cs.Attrs) {
			value = cs.Attrs[i+1]
		}
		if value != "" {
			res += fmt.Sprintf("[%s=%s]", name, value)
		} else {
			res += fmt.Sprintf("[%s]", name)
		}
	}
	
	for _, notSelector := range cs.NotSelectors {
		res += fmt.Sprintf(":not(%s)", notSelector.String())
	}
	
	return res
}

// SelectorMatcher matches CSS selectors
type SelectorMatcher[T any] struct {
	elementMap          map[string][]*SelectorContext[T]
	elementPartialMap   map[string]*SelectorMatcher[T]
	classMap            map[string][]*SelectorContext[T]
	classPartialMap     map[string]*SelectorMatcher[T]
	attrValueMap        map[string]map[string][]*SelectorContext[T]
	attrValuePartialMap map[string]map[string]*SelectorMatcher[T]
	listContexts        []*SelectorListContext
}

// NewSelectorMatcher creates a new SelectorMatcher
func NewSelectorMatcher[T any]() *SelectorMatcher[T] {
	return &SelectorMatcher[T]{
		elementMap:          make(map[string][]*SelectorContext[T]),
		elementPartialMap:   make(map[string]*SelectorMatcher[T]),
		classMap:            make(map[string][]*SelectorContext[T]),
		classPartialMap:     make(map[string]*SelectorMatcher[T]),
		attrValueMap:        make(map[string]map[string][]*SelectorContext[T]),
		attrValuePartialMap: make(map[string]map[string]*SelectorMatcher[T]),
		listContexts:        []*SelectorListContext{},
	}
}

// CreateNotMatcher creates a not matcher
func CreateNotMatcher(notSelectors []*CssSelector) *SelectorMatcher[interface{}] {
	notMatcher := NewSelectorMatcher[interface{}]()
	notMatcher.AddSelectables(notSelectors, nil)
	return notMatcher
}

// AddSelectables adds selectables to the matcher
func (sm *SelectorMatcher[T]) AddSelectables(cssSelectors []*CssSelector, callbackCtxt *T) {
	var listContext *SelectorListContext
	if len(cssSelectors) > 1 {
		listContext = NewSelectorListContext(cssSelectors)
		sm.listContexts = append(sm.listContexts, listContext)
	}
	
	for _, cssSelector := range cssSelectors {
		sm.addSelectable(cssSelector, callbackCtxt, listContext)
	}
}

func (sm *SelectorMatcher[T]) addSelectable(cssSelector *CssSelector, callbackCtxt *T, listContext *SelectorListContext) {
	matcher := sm
	element := cssSelector.Element
	classNames := cssSelector.ClassNames
	attrs := cssSelector.Attrs
	selectable := NewSelectorContext(cssSelector, callbackCtxt, listContext)
	
	if element != nil {
		isTerminal := len(attrs) == 0 && len(classNames) == 0
		if isTerminal {
			sm.addTerminal(sm.elementMap, *element, selectable)
		} else {
			matcher = sm.addPartial(sm.elementPartialMap, *element)
		}
	}
	
	for i, className := range classNames {
		isTerminal := len(attrs) == 0 && i == len(classNames)-1
		if isTerminal {
			sm.addTerminal(matcher.classMap, className, selectable)
		} else {
			matcher = sm.addPartial(matcher.classPartialMap, className)
		}
	}
	
	for i := 0; i < len(attrs); i += 2 {
		isTerminal := i == len(attrs)-2
		name := attrs[i]
		value := ""
		if i+1 < len(attrs) {
			value = attrs[i+1]
		}
		
		if isTerminal {
			terminalMap := matcher.attrValueMap
			terminalValuesMap, ok := terminalMap[name]
			if !ok {
				terminalValuesMap = make(map[string][]*SelectorContext[T])
				terminalMap[name] = terminalValuesMap
			}
			sm.addTerminal(terminalValuesMap, value, selectable)
		} else {
			partialMap := matcher.attrValuePartialMap
			partialValuesMap, ok := partialMap[name]
			if !ok {
				partialValuesMap = make(map[string]*SelectorMatcher[T])
				partialMap[name] = partialValuesMap
			}
			matcher = sm.addPartial(partialValuesMap, value)
		}
	}
}

func (sm *SelectorMatcher[T]) addTerminal(map_ map[string][]*SelectorContext[T], name string, selectable *SelectorContext[T]) {
	terminalList, ok := map_[name]
	if !ok {
		terminalList = []*SelectorContext[T]{}
	}
	map_[name] = append(terminalList, selectable)
}

func (sm *SelectorMatcher[T]) addPartial(map_ map[string]*SelectorMatcher[T], name string) *SelectorMatcher[T] {
	matcher, ok := map_[name]
	if !ok {
		matcher = NewSelectorMatcher[T]()
		map_[name] = matcher
	}
	return matcher
}

// MatchCallback is a function type for match callbacks
type MatchCallback[T any] func(c *CssSelector, a *T)

// Match finds matching selectors
func (sm *SelectorMatcher[T]) Match(cssSelector *CssSelector, matchedCallback MatchCallback[T]) bool {
	result := false
	element := ""
	if cssSelector.Element != nil {
		element = *cssSelector.Element
	}
	classNames := cssSelector.ClassNames
	attrs := cssSelector.Attrs
	
	for _, listContext := range sm.listContexts {
		listContext.AlreadyMatched = false
	}
	
	result = sm.matchTerminal(sm.elementMap, element, cssSelector, matchedCallback) || result
	result = sm.matchPartial(sm.elementPartialMap, element, cssSelector, matchedCallback) || result
	
	for _, className := range classNames {
		result = sm.matchTerminal(sm.classMap, className, cssSelector, matchedCallback) || result
		result = sm.matchPartial(sm.classPartialMap, className, cssSelector, matchedCallback) || result
	}
	
	for i := 0; i < len(attrs); i += 2 {
		name := attrs[i]
		value := ""
		if i+1 < len(attrs) {
			value = attrs[i+1]
		}
		
		terminalValuesMap, ok := sm.attrValueMap[name]
		if ok {
			if value != "" {
				result = sm.matchTerminal(terminalValuesMap, "", cssSelector, matchedCallback) || result
			}
			result = sm.matchTerminal(terminalValuesMap, value, cssSelector, matchedCallback) || result
		}
		
		partialValuesMap, ok := sm.attrValuePartialMap[name]
		if ok {
			if value != "" {
				result = sm.matchPartial(partialValuesMap, "", cssSelector, matchedCallback) || result
			}
			result = sm.matchPartial(partialValuesMap, value, cssSelector, matchedCallback) || result
		}
	}
	
	return result
}

func (sm *SelectorMatcher[T]) matchTerminal(map_ map[string][]*SelectorContext[T], name string, cssSelector *CssSelector, matchedCallback MatchCallback[T]) bool {
	if map_ == nil || name == "" {
		return false
	}
	
	selectables := map_[name]
	starSelectables, ok := map_["*"]
	if ok {
		selectables = append(selectables, starSelectables...)
	}
	
	if len(selectables) == 0 {
		return false
	}
	
	result := false
	for _, selectable := range selectables {
		if selectable.Finalize(cssSelector, matchedCallback) {
			result = true
		}
	}
	return result
}

func (sm *SelectorMatcher[T]) matchPartial(map_ map[string]*SelectorMatcher[T], name string, cssSelector *CssSelector, matchedCallback MatchCallback[T]) bool {
	if map_ == nil || name == "" {
		return false
	}
	
	nestedSelector, ok := map_[name]
	if !ok {
		return false
	}
	
	return nestedSelector.Match(cssSelector, matchedCallback)
}

// SelectorListContext represents a list of selectors
type SelectorListContext struct {
	AlreadyMatched bool
	Selectors      []*CssSelector
}

// NewSelectorListContext creates a new SelectorListContext
func NewSelectorListContext(selectors []*CssSelector) *SelectorListContext {
	return &SelectorListContext{
		AlreadyMatched: false,
		Selectors:      selectors,
	}
}

// SelectorContext represents a selector context
type SelectorContext[T any] struct {
	Selector    *CssSelector
	CbContext   *T
	ListContext *SelectorListContext
	NotSelectors []*CssSelector
}

// NewSelectorContext creates a new SelectorContext
func NewSelectorContext[T any](selector *CssSelector, cbContext *T, listContext *SelectorListContext) *SelectorContext[T] {
	return &SelectorContext[T]{
		Selector:     selector,
		CbContext:    cbContext,
		ListContext:  listContext,
		NotSelectors: selector.NotSelectors,
	}
}

// Finalize finalizes the selector match
func (sc *SelectorContext[T]) Finalize(cssSelector *CssSelector, callback MatchCallback[T]) bool {
	result := true
	if len(sc.NotSelectors) > 0 && (sc.ListContext == nil || !sc.ListContext.AlreadyMatched) {
		notMatcher := CreateNotMatcher(sc.NotSelectors)
		var nilCallback MatchCallback[interface{}]
		result = !notMatcher.Match(cssSelector, nilCallback)
	}
	if result && callback != nil && (sc.ListContext == nil || !sc.ListContext.AlreadyMatched) {
		if sc.ListContext != nil {
			sc.ListContext.AlreadyMatched = true
		}
		callback(sc.Selector, sc.CbContext)
	}
	return result
}

// SelectorlessMatcher matches selectors without full context
type SelectorlessMatcher[T any] struct {
	registry map[string][]T
}

// NewSelectorlessMatcher creates a new SelectorlessMatcher
func NewSelectorlessMatcher[T any](registry map[string][]T) *SelectorlessMatcher[T] {
	return &SelectorlessMatcher[T]{
		registry: registry,
	}
}

// Match matches a name
func (sm *SelectorlessMatcher[T]) Match(name string) []T {
	if values, ok := sm.registry[name]; ok {
		return values
	}
	return []T{}
}

