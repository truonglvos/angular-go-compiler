package css_test

import (
	"strings"
	"testing"

	"ngc-go/packages/compiler/src/css"
)

// Helper to create a CSS selector from a simple descriptor
func getSelectorFor(desc map[string]interface{}) *css.CssSelector {
	selector := css.NewCssSelector()

	tag := ""
	if t, ok := desc["tag"].(string); ok {
		tag = t
	}
	selector.SetElement(tag)

	if classes, ok := desc["classes"].(string); ok {
		// Split classes by whitespace
		trimmed := strings.TrimSpace(classes)
		if trimmed != "" {
			classNames := strings.Fields(trimmed)
			for _, className := range classNames {
				selector.AddClassName(className)
			}
		}
	}

	if attrs, ok := desc["attrs"].([][]string); ok {
		for _, attr := range attrs {
			name := attr[0]
			value := ""
			if len(attr) > 1 {
				value = attr[1]
			}
			selector.AddAttribute(name, value)
		}
	}

	return selector
}

func intPtr(i int) *int {
	return &i
}

func TestSelectorMatcher(t *testing.T) {
	t.Run("should select by element name case sensitive", func(t *testing.T) {
		matcher := css.NewSelectorMatcher[int]()
		s1, _ := css.ParseCssSelector("someTag")
		matcher.AddSelectables(s1, intPtr(1))

		matched := []interface{}{}
		selectableCollector := func(selector *css.CssSelector, context *int) {
			matched = append(matched, selector, *context)
		}

		// Should not match different tag
		result := matcher.Match(getSelectorFor(map[string]interface{}{"tag": "SOMEOTHERTAG"}), selectableCollector)
		if result != false {
			t.Errorf("Expected false, got %v", result)
		}
		if len(matched) != 0 {
			t.Errorf("Expected no matches, got %v", matched)
		}

		// Should not match uppercase
		matched = []interface{}{}
		result = matcher.Match(getSelectorFor(map[string]interface{}{"tag": "SOMETAG"}), selectableCollector)
		if result != false {
			t.Errorf("Expected false, got %v", result)
		}
		if len(matched) != 0 {
			t.Errorf("Expected no matches, got %v", matched)
		}

		// Should match exact case
		matched = []interface{}{}
		result = matcher.Match(getSelectorFor(map[string]interface{}{"tag": "someTag"}), selectableCollector)
		if result != true {
			t.Errorf("Expected true, got %v", result)
		}
		if len(matched) != 2 || matched[0] != s1[0] || matched[1] != 1 {
			t.Errorf("Expected [s1[0], 1], got %v", matched)
		}
	})

	t.Run("should select by class name case insensitive", func(t *testing.T) {
		matcher := css.NewSelectorMatcher[int]()
		s1, _ := css.ParseCssSelector(".someClass")
		matcher.AddSelectables(s1, intPtr(1))
		s2, _ := css.ParseCssSelector(".someClass.class2")
		matcher.AddSelectables(s2, intPtr(2))

		matched := []interface{}{}
		selectableCollector := func(selector *css.CssSelector, context *int) {
			matched = append(matched, selector, *context)
		}

		// Should not match different class
		result := matcher.Match(getSelectorFor(map[string]interface{}{"classes": "SOMEOTHERCLASS"}), selectableCollector)
		if result != false {
			t.Errorf("Expected false, got %v", result)
		}
		if len(matched) != 0 {
			t.Errorf("Expected no matches, got %v", matched)
		}

		// Should match case insensitive
		matched = []interface{}{}
		result = matcher.Match(getSelectorFor(map[string]interface{}{"classes": "SOMECLASS"}), selectableCollector)
		if result != true {
			t.Errorf("Expected true, got %v", result)
		}
		if len(matched) != 2 || matched[0] != s1[0] || matched[1] != 1 {
			t.Errorf("Expected [s1[0], 1], got %v", matched)
		}

		// Should match multiple classes
		matched = []interface{}{}
		result = matcher.Match(getSelectorFor(map[string]interface{}{"classes": "someClass class2"}), selectableCollector)
		if result != true {
			t.Errorf("Expected true, got %v", result)
		}
		if len(matched) != 4 {
			t.Errorf("Expected 4 elements (s1[0], 1, s2[0], 2), got %v", matched)
		}
	})

	t.Run("should not throw for class name constructor", func(t *testing.T) {
		matcher := css.NewSelectorMatcher[int]()
		matched := []interface{}{}
		selectableCollector := func(selector *css.CssSelector, context *int) {
			matched = append(matched, selector, *context)
		}

		result := matcher.Match(getSelectorFor(map[string]interface{}{"classes": "constructor"}), selectableCollector)
		if result != false {
			t.Errorf("Expected false, got %v", result)
		}
		if len(matched) != 0 {
			t.Errorf("Expected no matches, got %v", matched)
		}
	})

	t.Run("should select by attr name case sensitive independent of the value", func(t *testing.T) {
		matcher := css.NewSelectorMatcher[int]()
		s1, _ := css.ParseCssSelector("[someAttr]")
		matcher.AddSelectables(s1, intPtr(1))
		s2, _ := css.ParseCssSelector("[someAttr][someAttr2]")
		matcher.AddSelectables(s2, intPtr(2))

		matched := []interface{}{}
		selectableCollector := func(selector *css.CssSelector, context *int) {
			matched = append(matched, selector, *context)
		}

		// Should not match different attr
		result := matcher.Match(getSelectorFor(map[string]interface{}{"attrs": [][]string{{"SOMEOTHERATTR", ""}}}), selectableCollector)
		if result != false {
			t.Errorf("Expected false, got %v", result)
		}
		if len(matched) != 0 {
			t.Errorf("Expected no matches, got %v", matched)
		}

		// Should not match uppercase attr name
		matched = []interface{}{}
		result = matcher.Match(getSelectorFor(map[string]interface{}{"attrs": [][]string{{"SOMEATTR", ""}}}), selectableCollector)
		if result != false {
			t.Errorf("Expected false, got %v", result)
		}
		if len(matched) != 0 {
			t.Errorf("Expected no matches, got %v", matched)
		}

		// Should not match uppercase attr name with value
		matched = []interface{}{}
		result = matcher.Match(getSelectorFor(map[string]interface{}{"attrs": [][]string{{"SOMEATTR", "someValue"}}}), selectableCollector)
		if result != false {
			t.Errorf("Expected false, got %v", result)
		}
		if len(matched) != 0 {
			t.Errorf("Expected no matches, got %v", matched)
		}

		// Should match correct attr name with empty value - should match both s1 and s2
		matched = []interface{}{}
		result = matcher.Match(getSelectorFor(map[string]interface{}{"attrs": [][]string{{"someAttr", ""}, {"someAttr2", ""}}}), selectableCollector)
		if result != true {
			t.Errorf("Expected true, got %v", result)
		}
		if len(matched) != 4 {
			t.Errorf("Expected 4 elements (s1[0], 1, s2[0], 2), got %v (len=%d)", matched, len(matched))
		}

		// Should match with someValue - both s1 and s2
		matched = []interface{}{}
		result = matcher.Match(getSelectorFor(map[string]interface{}{"attrs": [][]string{{"someAttr", "someValue"}, {"someAttr2", ""}}}), selectableCollector)
		if result != true {
			t.Errorf("Expected true, got %v", result)
		}
		if len(matched) != 4 {
			t.Errorf("Expected 4 elements (s1[0], 1, s2[0], 2), got %v (len=%d)", matched, len(matched))
		}

		// Should match with reversed order - both s1 and s2
		matched = []interface{}{}
		result = matcher.Match(getSelectorFor(map[string]interface{}{"attrs": [][]string{{"someAttr2", ""}, {"someAttr", "someValue"}}}), selectableCollector)
		if result != true {
			t.Errorf("Expected true, got %v", result)
		}
		if len(matched) != 4 {
			t.Errorf("Expected 4 elements (s1[0], 1, s2[0], 2), got %v (len=%d)", matched, len(matched))
		}

		// Should match with reversed order and different values - both s1 and s2
		matched = []interface{}{}
		result = matcher.Match(getSelectorFor(map[string]interface{}{"attrs": [][]string{{"someAttr2", "someValue"}, {"someAttr", ""}}}), selectableCollector)
		if result != true {
			t.Errorf("Expected true, got %v", result)
		}
		if len(matched) != 4 {
			t.Errorf("Expected 4 elements (s1[0], 1, s2[0], 2), got %v (len=%d)", matched, len(matched))
		}
	})

	t.Run("should support dot in attribute names", func(t *testing.T) {
		matcher := css.NewSelectorMatcher[int]()
		s1, _ := css.ParseCssSelector("[foo.bar]")
		matcher.AddSelectables(s1, intPtr(1))

		matched := []interface{}{}
		selectableCollector := func(selector *css.CssSelector, context *int) {
			matched = append(matched, selector, *context)
		}

		result := matcher.Match(getSelectorFor(map[string]interface{}{"attrs": [][]string{{"barfoo", ""}}}), selectableCollector)
		if result != false {
			t.Errorf("Expected false, got %v", result)
		}
		if len(matched) != 0 {
			t.Errorf("Expected no matches, got %v", matched)
		}

		matched = []interface{}{}
		result = matcher.Match(getSelectorFor(map[string]interface{}{"attrs": [][]string{{"foo.bar", ""}}}), selectableCollector)
		if result != true {
			t.Errorf("Expected true, got %v", result)
		}
		if len(matched) != 2 || matched[0] != s1[0] || matched[1] != 1 {
			t.Errorf("Expected [s1[0], 1], got %v", matched)
		}
	})

	t.Run("should select by attr name case sensitive and value case insensitive", func(t *testing.T) {
		matcher := css.NewSelectorMatcher[int]()
		s1, _ := css.ParseCssSelector("[someAttr=someValue]")
		matcher.AddSelectables(s1, intPtr(1))

		matched := []interface{}{}
		selectableCollector := func(selector *css.CssSelector, context *int) {
			matched = append(matched, selector, *context)
		}

		result := matcher.Match(getSelectorFor(map[string]interface{}{"attrs": [][]string{{"SOMEATTR", "SOMEOTHERATTR"}}}), selectableCollector)
		if result != false {
			t.Errorf("Expected false, got %v", result)
		}
		if len(matched) != 0 {
			t.Errorf("Expected no matches, got %v", matched)
		}

		matched = []interface{}{}
		result = matcher.Match(getSelectorFor(map[string]interface{}{"attrs": [][]string{{"SOMEATTR", "SOMEVALUE"}}}), selectableCollector)
		if result != false {
			t.Errorf("Expected false, got %v", result)
		}
		if len(matched) != 0 {
			t.Errorf("Expected no matches, got %v", matched)
		}

		matched = []interface{}{}
		result = matcher.Match(getSelectorFor(map[string]interface{}{"attrs": [][]string{{"someAttr", "SOMEVALUE"}}}), selectableCollector)
		if result != true {
			t.Errorf("Expected true, got %v", result)
		}
		if len(matched) != 2 || matched[0] != s1[0] || matched[1] != 1 {
			t.Errorf("Expected [s1[0], 1], got %v", matched)
		}
	})

	t.Run("should select by element name, class name and attribute name with value", func(t *testing.T) {
		matcher := css.NewSelectorMatcher[int]()
		s1, _ := css.ParseCssSelector("someTag.someClass[someAttr=someValue]")
		matcher.AddSelectables(s1, intPtr(1))

		matched := []interface{}{}
		selectableCollector := func(selector *css.CssSelector, context *int) {
			matched = append(matched, selector, *context)
		}

		result := matcher.Match(getSelectorFor(map[string]interface{}{
			"tag":     "someOtherTag",
			"classes": "someOtherClass",
			"attrs":   [][]string{{"someOtherAttr", ""}},
		}), selectableCollector)
		if result != false {
			t.Errorf("Expected false, got %v", result)
		}
		if len(matched) != 0 {
			t.Errorf("Expected no matches, got %v", matched)
		}

		matched = []interface{}{}
		result = matcher.Match(getSelectorFor(map[string]interface{}{
			"tag":     "someTag",
			"classes": "someOtherClass",
			"attrs":   [][]string{{"someOtherAttr", ""}},
		}), selectableCollector)
		if result != false {
			t.Errorf("Expected false, got %v", result)
		}
		if len(matched) != 0 {
			t.Errorf("Expected no matches, got %v", matched)
		}

		matched = []interface{}{}
		result = matcher.Match(getSelectorFor(map[string]interface{}{
			"tag":     "someTag",
			"classes": "someClass",
			"attrs":   [][]string{{"someOtherAttr", ""}},
		}), selectableCollector)
		if result != false {
			t.Errorf("Expected false, got %v", result)
		}
		if len(matched) != 0 {
			t.Errorf("Expected no matches, got %v", matched)
		}

		matched = []interface{}{}
		result = matcher.Match(getSelectorFor(map[string]interface{}{
			"tag":     "someTag",
			"classes": "someClass",
			"attrs":   [][]string{{"someAttr", ""}},
		}), selectableCollector)
		if result != false {
			t.Errorf("Expected false, got %v", result)
		}
		if len(matched) != 0 {
			t.Errorf("Expected no matches, got %v", matched)
		}

		matched = []interface{}{}
		result = matcher.Match(getSelectorFor(map[string]interface{}{
			"tag":     "someTag",
			"classes": "someClass",
			"attrs":   [][]string{{"someAttr", "someValue"}},
		}), selectableCollector)
		if result != true {
			t.Errorf("Expected true, got %v", result)
		}
		if len(matched) != 2 || matched[0] != s1[0] || matched[1] != 1 {
			t.Errorf("Expected [s1[0], 1], got %v", matched)
		}
	})

	t.Run("should select by many attributes and independent of the value", func(t *testing.T) {
		matcher := css.NewSelectorMatcher[int]()
		s1, _ := css.ParseCssSelector("input[type=text][control]")
		matcher.AddSelectables(s1, intPtr(1))

		matched := []interface{}{}
		selectableCollector := func(selector *css.CssSelector, context *int) {
			matched = append(matched, selector, *context)
		}

		cssSelector := css.NewCssSelector()
		cssSelector.SetElement("input")
		cssSelector.AddAttribute("type", "text")
		cssSelector.AddAttribute("control", "one")

		result := matcher.Match(cssSelector, selectableCollector)
		if result != true {
			t.Errorf("Expected true, got %v", result)
		}
		if len(matched) != 2 || matched[0] != s1[0] || matched[1] != 1 {
			t.Errorf("Expected [s1[0], 1], got %v", matched)
		}
	})

	t.Run("should select independent of the order in the css selector", func(t *testing.T) {
		matcher := css.NewSelectorMatcher[int]()
		s1, _ := css.ParseCssSelector("[someAttr].someClass")
		matcher.AddSelectables(s1, intPtr(1))
		s2, _ := css.ParseCssSelector(".someClass[someAttr]")
		matcher.AddSelectables(s2, intPtr(2))
		s3, _ := css.ParseCssSelector(".class1.class2")
		matcher.AddSelectables(s3, intPtr(3))
		s4, _ := css.ParseCssSelector(".class2.class1")
		matcher.AddSelectables(s4, intPtr(4))

		matched := []interface{}{}
		selectableCollector := func(selector *css.CssSelector, context *int) {
			matched = append(matched, selector, *context)
		}

		parsed1, _ := css.ParseCssSelector("[someAttr].someClass")
		result := matcher.Match(parsed1[0], selectableCollector)
		if result != true {
			t.Errorf("Expected true, got %v", result)
		}
		if len(matched) != 4 {
			t.Errorf("Expected 4 elements (s1[0], 1, s2[0], 2), got %v (len=%d)", matched, len(matched))
		}

		matched = []interface{}{}
		parsed2, _ := css.ParseCssSelector(".someClass[someAttr]")
		result = matcher.Match(parsed2[0], selectableCollector)
		if result != true {
			t.Errorf("Expected true, got %v", result)
		}
		if len(matched) != 4 {
			t.Errorf("Expected 4 elements (s1[0], 1, s2[0], 2), got %v (len=%d)", matched, len(matched))
		}

		matched = []interface{}{}
		parsed3, _ := css.ParseCssSelector(".class1.class2")
		result = matcher.Match(parsed3[0], selectableCollector)
		if result != true {
			t.Errorf("Expected true, got %v", result)
		}
		if len(matched) != 4 {
			t.Errorf("Expected 4 elements (s3[0], 3, s4[0], 4), got %v (len=%d)", matched, len(matched))
		}

		matched = []interface{}{}
		parsed4, _ := css.ParseCssSelector(".class2.class1")
		result = matcher.Match(parsed4[0], selectableCollector)
		if result != true {
			t.Errorf("Expected true, got %v", result)
		}
		// Order might be different for class2.class1
		if len(matched) != 4 {
			t.Errorf("Expected 4 elements, got %v (len=%d)", matched, len(matched))
		}
	})

	t.Run("should not select with a matching :not selector", func(t *testing.T) {
		matcher := css.NewSelectorMatcher[int]()
		s1, _ := css.ParseCssSelector("p:not(.someClass)")
		matcher.AddSelectables(s1, intPtr(1))
		s2, _ := css.ParseCssSelector("p:not([someAttr])")
		matcher.AddSelectables(s2, intPtr(2))
		s3, _ := css.ParseCssSelector(":not(.someClass)")
		matcher.AddSelectables(s3, intPtr(3))
		s4, _ := css.ParseCssSelector(":not(p)")
		matcher.AddSelectables(s4, intPtr(4))
		s5, _ := css.ParseCssSelector(":not(p[someAttr])")
		matcher.AddSelectables(s5, intPtr(5))

		matched := []interface{}{}
		selectableCollector := func(selector *css.CssSelector, context *int) {
			matched = append(matched, selector, *context)
		}

		result := matcher.Match(getSelectorFor(map[string]interface{}{
			"tag":     "p",
			"classes": "someClass",
			"attrs":   [][]string{{"someAttr", ""}},
		}), selectableCollector)
		if result != false {
			t.Errorf("Expected false, got %v", result)
		}
		if len(matched) != 0 {
			t.Errorf("Expected no matches, got %v", matched)
		}
	})

	t.Run("should select with a non matching :not selector", func(t *testing.T) {
		matcher := css.NewSelectorMatcher[int]()
		s1, _ := css.ParseCssSelector("p:not(.someClass)")
		matcher.AddSelectables(s1, intPtr(1))
		s2, _ := css.ParseCssSelector("p:not(.someOtherClass[someAttr])")
		matcher.AddSelectables(s2, intPtr(2))
		s3, _ := css.ParseCssSelector(":not(.someClass)")
		matcher.AddSelectables(s3, intPtr(3))
		s4, _ := css.ParseCssSelector(":not(.someOtherClass[someAttr])")
		matcher.AddSelectables(s4, intPtr(4))

		matched := []interface{}{}
		selectableCollector := func(selector *css.CssSelector, context *int) {
			matched = append(matched, selector, *context)
		}

		result := matcher.Match(getSelectorFor(map[string]interface{}{
			"tag":     "p",
			"attrs":   [][]string{{"someOtherAttr", ""}},
			"classes": "someOtherClass",
		}), selectableCollector)
		if result != true {
			t.Errorf("Expected true, got %v", result)
		}
		if len(matched) != 8 {
			t.Errorf("Expected 8 elements (s1[0], 1, s2[0], 2, s3[0], 3, s4[0], 4), got %v (len=%d)", matched, len(matched))
		}
	})

	t.Run("should match * with :not selector", func(t *testing.T) {
		matcher := css.NewSelectorMatcher[int]()
		s1, _ := css.ParseCssSelector(":not([a])")
		matcher.AddSelectables(s1, intPtr(1))

		called := false
		result := matcher.Match(getSelectorFor(map[string]interface{}{"tag": "div"}), func(c *css.CssSelector, a *int) {
			called = true
		})
		if result != true {
			t.Errorf("Expected true, got %v", result)
		}
		if !called {
			t.Errorf("Expected callback to be called")
		}
	})

	t.Run("should match with multiple :not selectors", func(t *testing.T) {
		matcher := css.NewSelectorMatcher[int]()
		s1, _ := css.ParseCssSelector("div:not([a]):not([b])")
		matcher.AddSelectables(s1, intPtr(1))

		matched := []interface{}{}
		selectableCollector := func(selector *css.CssSelector, context *int) {
			matched = append(matched, selector, *context)
		}

		result := matcher.Match(getSelectorFor(map[string]interface{}{"tag": "div", "attrs": [][]string{{"a", ""}}}), selectableCollector)
		if result != false {
			t.Errorf("Expected false, got %v", result)
		}

		matched = []interface{}{}
		result = matcher.Match(getSelectorFor(map[string]interface{}{"tag": "div", "attrs": [][]string{{"b", ""}}}), selectableCollector)
		if result != false {
			t.Errorf("Expected false, got %v", result)
		}

		matched = []interface{}{}
		result = matcher.Match(getSelectorFor(map[string]interface{}{"tag": "div", "attrs": [][]string{{"c", ""}}}), selectableCollector)
		if result != true {
			t.Errorf("Expected true, got %v", result)
		}
		if len(matched) != 2 {
			t.Errorf("Expected 2 elements, got %v (len=%d)", matched, len(matched))
		}
	})

	t.Run("should select with one match in a list", func(t *testing.T) {
		matcher := css.NewSelectorMatcher[int]()
		s1, _ := css.ParseCssSelector("input[type=text], textbox")
		matcher.AddSelectables(s1, intPtr(1))

		matched := []interface{}{}
		selectableCollector := func(selector *css.CssSelector, context *int) {
			matched = append(matched, selector, *context)
		}

		result := matcher.Match(getSelectorFor(map[string]interface{}{"tag": "textbox"}), selectableCollector)
		if result != true {
			t.Errorf("Expected true, got %v", result)
		}
		if len(matched) != 2 || matched[0] != s1[1] || matched[1] != 1 {
			t.Errorf("Expected [s1[1], 1], got %v", matched)
		}

		matched = []interface{}{}
		result = matcher.Match(getSelectorFor(map[string]interface{}{
			"tag":   "input",
			"attrs": [][]string{{"type", "text"}},
		}), selectableCollector)
		if result != true {
			t.Errorf("Expected true, got %v", result)
		}
		if len(matched) != 2 || matched[0] != s1[0] || matched[1] != 1 {
			t.Errorf("Expected [s1[0], 1], got %v", matched)
		}
	})

	t.Run("should not select twice with two matches in a list", func(t *testing.T) {
		matcher := css.NewSelectorMatcher[int]()
		s1, _ := css.ParseCssSelector("input, .someClass")
		matcher.AddSelectables(s1, intPtr(1))

		matched := []interface{}{}
		selectableCollector := func(selector *css.CssSelector, context *int) {
			matched = append(matched, selector, *context)
		}

		result := matcher.Match(getSelectorFor(map[string]interface{}{
			"tag":     "input",
			"classes": "someclass",
		}), selectableCollector)
		if result != true {
			t.Errorf("Expected true, got %v", result)
		}
		if len(matched) != 2 {
			t.Errorf("Expected 2 elements (matched once), got %v (len=%d)", matched, len(matched))
		}
		if matched[0] != s1[0] || matched[1] != 1 {
			t.Errorf("Expected [s1[0], 1], got %v", matched)
		}
	})
}

func TestCssSelectorParse(t *testing.T) {
	t.Run("should detect element names", func(t *testing.T) {
		cssSelectors, _ := css.ParseCssSelector("sometag")
		if len(cssSelectors) != 1 {
			t.Fatalf("Expected 1 selector, got %d", len(cssSelectors))
		}
		cssSelector := cssSelectors[0]
		if cssSelector.GetElement() == nil || *cssSelector.GetElement() != "sometag" {
			t.Errorf("Expected element 'sometag', got %v", cssSelector.GetElement())
		}
		if cssSelector.String() != "sometag" {
			t.Errorf("Expected 'sometag', got %q", cssSelector.String())
		}
	})

	t.Run("should detect class names", func(t *testing.T) {
		cssSelectors, _ := css.ParseCssSelector(".someClass")
		if len(cssSelectors) != 1 {
			t.Fatalf("Expected 1 selector, got %d", len(cssSelectors))
		}
		cssSelector := cssSelectors[0]
		classNames := cssSelector.GetClassNames()
		if len(classNames) != 1 || classNames[0] != "someclass" {
			t.Errorf("Expected ['someclass'], got %v", classNames)
		}
		if cssSelector.String() != ".someclass" {
			t.Errorf("Expected '.someclass', got %q", cssSelector.String())
		}
	})

	t.Run("should detect attr names", func(t *testing.T) {
		cssSelectors, _ := css.ParseCssSelector("[attrname]")
		if len(cssSelectors) != 1 {
			t.Fatalf("Expected 1 selector, got %d", len(cssSelectors))
		}
		cssSelector := cssSelectors[0]
		// Use Attrs field directly, not GetAttrs() which includes class names
		attrs := cssSelector.Attrs
		if len(attrs) != 2 || attrs[0] != "attrname" || attrs[1] != "" {
			t.Errorf("Expected ['attrname', ''], got %v", attrs)
		}
		if cssSelector.String() != "[attrname]" {
			t.Errorf("Expected '[attrname]', got %q", cssSelector.String())
		}
	})

	t.Run("should detect attr values", func(t *testing.T) {
		cssSelectors, _ := css.ParseCssSelector("[attrname=attrvalue]")
		if len(cssSelectors) != 1 {
			t.Fatalf("Expected 1 selector, got %d", len(cssSelectors))
		}
		cssSelector := cssSelectors[0]
		attrs := cssSelector.Attrs
		if len(attrs) != 2 || attrs[0] != "attrname" || attrs[1] != "attrvalue" {
			t.Errorf("Expected ['attrname', 'attrvalue'], got %v", attrs)
		}
		if cssSelector.String() != "[attrname=attrvalue]" {
			t.Errorf("Expected '[attrname=attrvalue]', got %q", cssSelector.String())
		}
	})

	t.Run("should detect attr values with double quotes", func(t *testing.T) {
		cssSelectors, _ := css.ParseCssSelector(`[attrname="attrvalue"]`)
		if len(cssSelectors) != 1 {
			t.Fatalf("Expected 1 selector, got %d", len(cssSelectors))
		}
		cssSelector := cssSelectors[0]
		attrs := cssSelector.Attrs
		if len(attrs) != 2 || attrs[0] != "attrname" || attrs[1] != "attrvalue" {
			t.Errorf("Expected ['attrname', 'attrvalue'], got %v", attrs)
		}
		if cssSelector.String() != "[attrname=attrvalue]" {
			t.Errorf("Expected '[attrname=attrvalue]', got %q", cssSelector.String())
		}
	})

	t.Run("should detect #some-value syntax and treat as attribute", func(t *testing.T) {
		cssSelectors, _ := css.ParseCssSelector("#some-value")
		if len(cssSelectors) != 1 {
			t.Fatalf("Expected 1 selector, got %d", len(cssSelectors))
		}
		cssSelector := cssSelectors[0]
		attrs := cssSelector.Attrs
		if len(attrs) != 2 || attrs[0] != "id" || attrs[1] != "some-value" {
			t.Errorf("Expected ['id', 'some-value'], got %v", attrs)
		}
		if cssSelector.String() != "[id=some-value]" {
			t.Errorf("Expected '[id=some-value]', got %q", cssSelector.String())
		}
	})

	t.Run("should detect attr values with single quotes", func(t *testing.T) {
		cssSelectors, _ := css.ParseCssSelector("[attrname='attrvalue']")
		if len(cssSelectors) != 1 {
			t.Fatalf("Expected 1 selector, got %d", len(cssSelectors))
		}
		cssSelector := cssSelectors[0]
		attrs := cssSelector.Attrs
		if len(attrs) != 2 || attrs[0] != "attrname" || attrs[1] != "attrvalue" {
			t.Errorf("Expected ['attrname', 'attrvalue'], got %v", attrs)
		}
		if cssSelector.String() != "[attrname=attrvalue]" {
			t.Errorf("Expected '[attrname=attrvalue]', got %q", cssSelector.String())
		}
	})

	t.Run("should detect multiple parts", func(t *testing.T) {
		cssSelectors, _ := css.ParseCssSelector("sometag[attrname=attrvalue].someclass")
		if len(cssSelectors) != 1 {
			t.Fatalf("Expected 1 selector, got %d", len(cssSelectors))
		}
		cssSelector := cssSelectors[0]
		if cssSelector.GetElement() == nil || *cssSelector.GetElement() != "sometag" {
			t.Errorf("Expected element 'sometag', got %v", cssSelector.GetElement())
		}
		attrs := cssSelector.Attrs
		if len(attrs) != 2 || attrs[0] != "attrname" || attrs[1] != "attrvalue" {
			t.Errorf("Expected ['attrname', 'attrvalue'], got %v", attrs)
		}
		classNames := cssSelector.GetClassNames()
		if len(classNames) != 1 || classNames[0] != "someclass" {
			t.Errorf("Expected ['someclass'], got %v", classNames)
		}
		if cssSelector.String() != "sometag.someclass[attrname=attrvalue]" {
			t.Errorf("Expected 'sometag.someclass[attrname=attrvalue]', got %q", cssSelector.String())
		}
	})

	t.Run("should detect multiple attributes", func(t *testing.T) {
		cssSelectors, _ := css.ParseCssSelector("input[type=text][control]")
		if len(cssSelectors) != 1 {
			t.Fatalf("Expected 1 selector, got %d", len(cssSelectors))
		}
		cssSelector := cssSelectors[0]
		if cssSelector.GetElement() == nil || *cssSelector.GetElement() != "input" {
			t.Errorf("Expected element 'input', got %v", cssSelector.GetElement())
		}
		attrs := cssSelector.Attrs
		if len(attrs) != 4 || attrs[0] != "type" || attrs[1] != "text" || attrs[2] != "control" || attrs[3] != "" {
			t.Errorf("Expected ['type', 'text', 'control', ''], got %v", attrs)
		}
		if cssSelector.String() != "input[type=text][control]" {
			t.Errorf("Expected 'input[type=text][control]', got %q", cssSelector.String())
		}
	})

	t.Run("should detect :not", func(t *testing.T) {
		cssSelectors, _ := css.ParseCssSelector("sometag:not([attrname=attrvalue].someclass)")
		if len(cssSelectors) != 1 {
			t.Fatalf("Expected 1 selector, got %d", len(cssSelectors))
		}
		cssSelector := cssSelectors[0]
		if cssSelector.GetElement() == nil || *cssSelector.GetElement() != "sometag" {
			t.Errorf("Expected element 'sometag', got %v", cssSelector.GetElement())
		}
		if len(cssSelector.GetAttrs()) != 0 {
			t.Errorf("Expected no attrs, got %v", cssSelector.GetAttrs())
		}
		if len(cssSelector.GetClassNames()) != 0 {
			t.Errorf("Expected no classNames, got %v", cssSelector.GetClassNames())
		}

		notSelectors := cssSelector.GetNotSelectors()
		if len(notSelectors) != 1 {
			t.Fatalf("Expected 1 notSelector, got %d", len(notSelectors))
		}
		notSelector := notSelectors[0]
		if notSelector.GetElement() != nil {
			t.Errorf("Expected notSelector.element to be nil, got %v", notSelector.GetElement())
		}
		attrs := notSelector.Attrs
		if len(attrs) != 2 || attrs[0] != "attrname" || attrs[1] != "attrvalue" {
			t.Errorf("Expected ['attrname', 'attrvalue'], got %v", attrs)
		}
		classNames := notSelector.GetClassNames()
		if len(classNames) != 1 || classNames[0] != "someclass" {
			t.Errorf("Expected ['someclass'], got %v", classNames)
		}
		if cssSelector.String() != "sometag:not(.someclass[attrname=attrvalue])" {
			t.Errorf("Expected 'sometag:not(.someclass[attrname=attrvalue])', got %q", cssSelector.String())
		}
	})

	t.Run("should detect :not without truthy", func(t *testing.T) {
		cssSelectors, _ := css.ParseCssSelector(":not([attrname=attrvalue].someclass)")
		if len(cssSelectors) != 1 {
			t.Fatalf("Expected 1 selector, got %d", len(cssSelectors))
		}
		cssSelector := cssSelectors[0]
		if cssSelector.GetElement() == nil || *cssSelector.GetElement() != "*" {
			t.Errorf("Expected element '*', got %v", cssSelector.GetElement())
		}

		notSelectors := cssSelector.GetNotSelectors()
		if len(notSelectors) != 1 {
			t.Fatalf("Expected 1 notSelector, got %d", len(notSelectors))
		}
		notSelector := notSelectors[0]
		attrs := notSelector.Attrs
		if len(attrs) != 2 || attrs[0] != "attrname" || attrs[1] != "attrvalue" {
			t.Errorf("Expected ['attrname', 'attrvalue'], got %v", attrs)
		}
		classNames := notSelector.GetClassNames()
		if len(classNames) != 1 || classNames[0] != "someclass" {
			t.Errorf("Expected ['someclass'], got %v", classNames)
		}
		if cssSelector.String() != "*:not(.someclass[attrname=attrvalue])" {
			t.Errorf("Expected '*:not(.someclass[attrname=attrvalue])', got %q", cssSelector.String())
		}
	})

	t.Run("should detect lists of selectors", func(t *testing.T) {
		cssSelectors, _ := css.ParseCssSelector(".someclass,[attrname=attrvalue], sometag")
		if len(cssSelectors) != 3 {
			t.Fatalf("Expected 3 selectors, got %d", len(cssSelectors))
		}

		classNames := cssSelectors[0].GetClassNames()
		if len(classNames) != 1 || classNames[0] != "someclass" {
			t.Errorf("Expected ['someclass'], got %v", classNames)
		}

		attrs := cssSelectors[1].Attrs
		if len(attrs) != 2 || attrs[0] != "attrname" || attrs[1] != "attrvalue" {
			t.Errorf("Expected ['attrname', 'attrvalue'], got %v", attrs)
		}

		if cssSelectors[2].GetElement() == nil || *cssSelectors[2].GetElement() != "sometag" {
			t.Errorf("Expected element 'sometag', got %v", cssSelectors[2].GetElement())
		}
	})

	t.Run("should detect lists of selectors with :not", func(t *testing.T) {
		cssSelectors, err := css.ParseCssSelector("input[type=text], :not(textarea), textbox:not(.special)")
		if err != nil {
			t.Fatalf("ParseCssSelector error: %v", err)
		}
		if len(cssSelectors) != 3 {
			t.Fatalf("Expected 3 selectors, got %d", len(cssSelectors))
		}

		if cssSelectors[0].GetElement() == nil || *cssSelectors[0].GetElement() != "input" {
			t.Errorf("Expected element 'input', got %v", cssSelectors[0].GetElement())
		}
		attrs := cssSelectors[0].Attrs
		if len(attrs) != 2 || attrs[0] != "type" || attrs[1] != "text" {
			t.Errorf("Expected ['type', 'text'], got %v", attrs)
		}

		if cssSelectors[1].GetElement() == nil || *cssSelectors[1].GetElement() != "*" {
			t.Errorf("Expected element '*', got %v", cssSelectors[1].GetElement())
		}
		notSelectors1 := cssSelectors[1].GetNotSelectors()
		if len(notSelectors1) != 1 {
			t.Fatalf("Expected 1 notSelector, got %d", len(notSelectors1))
		}
		if notSelectors1[0].GetElement() == nil || *notSelectors1[0].GetElement() != "textarea" {
			t.Errorf("Expected notSelector element 'textarea', got %v", notSelectors1[0].GetElement())
		}

		if cssSelectors[2].GetElement() == nil || *cssSelectors[2].GetElement() != "textbox" {
			t.Errorf("Expected element 'textbox', got %v", cssSelectors[2].GetElement())
		}
		notSelectors2 := cssSelectors[2].GetNotSelectors()
		if len(notSelectors2) != 1 {
			t.Fatalf("Expected 1 notSelector, got %d", len(notSelectors2))
		}
		classNames := notSelectors2[0].GetClassNames()
		if len(classNames) != 1 || classNames[0] != "special" {
			t.Errorf("Expected ['special'], got %v", classNames)
		}
	})
}
