package css

import (
	"fmt"
	"regexp"
	"strings"
)

// SelectorRegexp represents the regex group indices for selector parsing
type SelectorRegexp int

const (
	SelectorRegexpAll             SelectorRegexp = iota
	SelectorRegexpNot                            // 1: ":not("
	SelectorRegexpTag                            // 2: tag with prefix
	SelectorRegexpPrefix                         // 3: prefix (. or #)
	SelectorRegexpAttribute                      // 4: attribute name
	SelectorRegexpAttributeValue                 // 5: attribute value (double quoted)
	SelectorRegexpAttributeValue2                // 6: attribute value (single quoted)
	SelectorRegexpAttributeValue3                // 7: attribute value (unquoted)
	SelectorRegexpNotEnd                         // 8: ")"
	SelectorRegexpSeparator                      // 9: ","
)

// selectorRegexp matches CSS selector patterns
// Go doesn't support backreferences, so we accept any quote type and value without validating matching quotes
var selectorRegexp = regexp.MustCompile(
	`(\:not\()|` + // 1: ":not("
		`(([\.\#]?)[-\w]+)|` + // 2: "tag"; 3: "."/"#"
		// 4: attribute name; 5: double quoted value; 6: single quoted value; 7: unquoted value
		`(?:\[([-.\w*\\$]+)(?:=(?:"([^"]*)"|'([^']*)'|([^\]\s]+)))?\])|` + // [name], [name=value], [name="value"], [name='value']
		`(\))|` + // 8: ")"
		`(\s*,\s*)`, // 9: ","
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
	fmt.Printf("[DEBUG ParseCssSelector] Parsing selector=%q, found %d matches\n", selector, len(matches))
	for i, match := range matches {
		fmt.Printf("[DEBUG ParseCssSelector] match[%d]: inNot=%v, match=%q\n", i, inNot, match[0])
		if len(match) > int(SelectorRegexpNot) && match[SelectorRegexpNot] != "" {
			if inNot {
				return nil, fmt.Errorf("nesting :not in a selector is not allowed")
			}
			fmt.Printf("[DEBUG ParseCssSelector] Setting inNot=true\n")
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

		// Attribute handling: group 4 = name, groups 5/6/7 = value (one will match based on quote type)
		if len(match) > int(SelectorRegexpAttribute) && match[SelectorRegexpAttribute] != "" {
			attribute := match[SelectorRegexpAttribute] // Group 4: attribute name
			attrValue := ""

			// Check which value group matched (only one will be non-empty)
			// Group 5: double quoted, Group 6: single quoted, Group 7: unquoted
			if len(match) > int(SelectorRegexpAttributeValue) && match[SelectorRegexpAttributeValue] != "" {
				attrValue = match[SelectorRegexpAttributeValue] // Double quoted value
			} else if len(match) > int(SelectorRegexpAttributeValue2) && match[SelectorRegexpAttributeValue2] != "" {
				attrValue = match[SelectorRegexpAttributeValue2] // Single quoted value
			} else if len(match) > int(SelectorRegexpAttributeValue3) && match[SelectorRegexpAttributeValue3] != "" {
				attrValue = match[SelectorRegexpAttributeValue3] // Unquoted value
			}
			// else: no value, attrValue remains ""

			unescapedAttr, err := current.UnescapeAttribute(attribute)
			if err != nil {
				return nil, err
			}
			current.AddAttribute(unescapedAttr, attrValue)
		}

		if len(match) > int(SelectorRegexpNotEnd) && match[SelectorRegexpNotEnd] != "" {
			fmt.Printf("[DEBUG ParseCssSelector] Found ), setting inNot=false\n")
			inNot = false
			current = cssSelector
		}

		if len(match) > int(SelectorRegexpSeparator) && match[SelectorRegexpSeparator] != "" {
			if inNot {
				return nil, fmt.Errorf("multiple selectors in :not are not supported")
			}
			fmt.Printf("[DEBUG ParseCssSelector] Found separator, adding result and resetting\n")
			results = addResult(results, cssSelector)
			cssSelector = NewCssSelector()
			current = cssSelector
			inNot = false // Reset inNot for the next selector in the list
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

// GetElement returns the element name
func (cs *CssSelector) GetElement() *string {
	return cs.Element
}

// GetClassNames returns the class names
func (cs *CssSelector) GetClassNames() []string {
	return cs.ClassNames
}

// GetNotSelectors returns the :not selectors
func (cs *CssSelector) GetNotSelectors() []*CssSelector {
	return cs.NotSelectors
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
	fmt.Printf("[DEBUG CSS AddSelectables] Adding %d selectors\n", len(cssSelectors))
	var listContext *SelectorListContext
	if len(cssSelectors) > 1 {
		listContext = NewSelectorListContext(cssSelectors)
		sm.listContexts = append(sm.listContexts, listContext)
	}

	for i, cssSelector := range cssSelectors {
		elem := ""
		if cssSelector.Element != nil {
			elem = *cssSelector.Element
		}
		fmt.Printf("[DEBUG CSS AddSelectables] selector[%d]: element=%q, attrs=%v, attrs_len=%d\n", i, elem, cssSelector.Attrs, len(cssSelector.Attrs))
		sm.addSelectable(cssSelector, callbackCtxt, listContext)
	}
	fmt.Printf("[DEBUG CSS AddSelectables] Done adding selectors\n")
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
		fmt.Printf("[DEBUG CSS addPartial] Creating new nested matcher for key=%q\n", name)
		matcher = NewSelectorMatcher[T]()
		map_[name] = matcher
	} else {
		fmt.Printf("[DEBUG CSS addPartial] Reusing existing nested matcher for key=%q\n", name)
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
	fmt.Printf("[DEBUG CSS Match] element=%q, classNames=%v, attrs=%v, attrs_len=%d\n", element, classNames, attrs, len(attrs))

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
		fmt.Printf("[DEBUG CSS Match] Processing attr i=%d, name=%q, value=%q\n", i, name, value)

		terminalValuesMap, ok := sm.attrValueMap[name]
		fmt.Printf("[DEBUG CSS Match] attrValueMap[%q] exists=%v\n", name, ok)
		if ok {
			if value != "" {
				result = sm.matchTerminal(terminalValuesMap, "", cssSelector, matchedCallback) || result
			}
			result = sm.matchTerminal(terminalValuesMap, value, cssSelector, matchedCallback) || result
		}

		partialValuesMap, ok := sm.attrValuePartialMap[name]
		fmt.Printf("[DEBUG CSS Match] attrValuePartialMap[%q] exists=%v\n", name, ok)
		if ok {
			fmt.Printf("[DEBUG CSS Match] Calling matchPartial for name=%q, value=%q\n", name, value)
			if value != "" {
				result = sm.matchPartial(partialValuesMap, "", cssSelector, matchedCallback) || result
			}
			result = sm.matchPartial(partialValuesMap, value, cssSelector, matchedCallback) || result
		}
	}

	return result
}

func (sm *SelectorMatcher[T]) matchTerminal(map_ map[string][]*SelectorContext[T], name string, cssSelector *CssSelector, matchedCallback MatchCallback[T]) bool {
	if map_ == nil {
		return false
	}
	// Note: name can be "" (empty string) which is a valid attribute value, element name, or class name

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
	if map_ == nil {
		return false
	}
	// Note: name can be "" (empty string) which is a valid attribute value, so don't reject it

	nestedSelector, ok := map_[name]
	if !ok {
		fmt.Printf("[DEBUG CSS matchPartial] map_[%q] not found\n", name)
		return false
	}

	fmt.Printf("[DEBUG CSS matchPartial] Found nested matcher for %q, calling nested.Match...\n", name)
	result := nestedSelector.Match(cssSelector, matchedCallback)
	fmt.Printf("[DEBUG CSS matchPartial] nested.Match returned %v\n", result)
	return result
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
	Selector     *CssSelector
	CbContext    *T
	ListContext  *SelectorListContext
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
