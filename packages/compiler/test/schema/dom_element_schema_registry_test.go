package schema_test

import (
	"ngc-go/packages/compiler/src/core"
	"ngc-go/packages/compiler/src/schema"
	"testing"
)

func TestDOMElementSchema(t *testing.T) {
	var registry *schema.DomElementSchemaRegistry

	setup := func() {
		registry = schema.NewDomElementSchemaRegistry()
	}

	t.Run("should detect elements", func(t *testing.T) {
		setup()
		
		if !registry.HasElement("div", []*core.SchemaMetadata{}) {
			t.Error("Expected HasElement('div') to be true")
		}
		if !registry.HasElement("b", []*core.SchemaMetadata{}) {
			t.Error("Expected HasElement('b') to be true")
		}
		if !registry.HasElement("ng-container", []*core.SchemaMetadata{}) {
			t.Error("Expected HasElement('ng-container') to be true")
		}
		if !registry.HasElement("ng-content", []*core.SchemaMetadata{}) {
			t.Error("Expected HasElement('ng-content') to be true")
		}

		if registry.HasElement("my-cmp", []*core.SchemaMetadata{}) {
			t.Error("Expected HasElement('my-cmp') to be false")
		}
		if registry.HasElement("abc", []*core.SchemaMetadata{}) {
			t.Error("Expected HasElement('abc') to be false")
		}
	})

	// https://github.com/angular/angular/issues/11219
	t.Run("should detect elements missing from chrome", func(t *testing.T) {
		setup()
		
		if !registry.HasElement("data", []*core.SchemaMetadata{}) {
			t.Error("Expected HasElement('data') to be true")
		}
		if !registry.HasElement("menuitem", []*core.SchemaMetadata{}) {
			t.Error("Expected HasElement('menuitem') to be true")
		}
		if !registry.HasElement("summary", []*core.SchemaMetadata{}) {
			t.Error("Expected HasElement('summary') to be true")
		}
		if !registry.HasElement("time", []*core.SchemaMetadata{}) {
			t.Error("Expected HasElement('time') to be true")
		}
	})

	t.Run("should detect properties on regular elements", func(t *testing.T) {
		setup()
		
		if !registry.HasProperty("div", "id", []*core.SchemaMetadata{}) {
			t.Error("Expected HasProperty('div', 'id') to be true")
		}
		if !registry.HasProperty("div", "title", []*core.SchemaMetadata{}) {
			t.Error("Expected HasProperty('div', 'title') to be true")
		}
		if !registry.HasProperty("div", "inert", []*core.SchemaMetadata{}) {
			t.Error("Expected HasProperty('div', 'inert') to be true")
		}
		if !registry.HasProperty("h1", "align", []*core.SchemaMetadata{}) {
			t.Error("Expected HasProperty('h1', 'align') to be true")
		}
		if !registry.HasProperty("h2", "align", []*core.SchemaMetadata{}) {
			t.Error("Expected HasProperty('h2', 'align') to be true")
		}
		if !registry.HasProperty("h3", "align", []*core.SchemaMetadata{}) {
			t.Error("Expected HasProperty('h3', 'align') to be true")
		}
		if !registry.HasProperty("h4", "align", []*core.SchemaMetadata{}) {
			t.Error("Expected HasProperty('h4', 'align') to be true")
		}
		if !registry.HasProperty("h5", "align", []*core.SchemaMetadata{}) {
			t.Error("Expected HasProperty('h5', 'align') to be true")
		}
		if !registry.HasProperty("h6", "align", []*core.SchemaMetadata{}) {
			t.Error("Expected HasProperty('h6', 'align') to be true")
		}
		if registry.HasProperty("h7", "align", []*core.SchemaMetadata{}) {
			t.Error("Expected HasProperty('h7', 'align') to be false")
		}
		if !registry.HasProperty("textarea", "disabled", []*core.SchemaMetadata{}) {
			t.Error("Expected HasProperty('textarea', 'disabled') to be true")
		}
		if !registry.HasProperty("input", "disabled", []*core.SchemaMetadata{}) {
			t.Error("Expected HasProperty('input', 'disabled') to be true")
		}
		if registry.HasProperty("div", "unknown", []*core.SchemaMetadata{}) {
			t.Error("Expected HasProperty('div', 'unknown') to be false")
		}
	})

	// https://github.com/angular/angular/issues/11219
	t.Run("should detect properties on elements missing from Chrome", func(t *testing.T) {
		setup()
		
		if !registry.HasProperty("data", "value", []*core.SchemaMetadata{}) {
			t.Error("Expected HasProperty('data', 'value') to be true")
		}
		if !registry.HasProperty("menuitem", "type", []*core.SchemaMetadata{}) {
			t.Error("Expected HasProperty('menuitem', 'type') to be true")
		}
		if !registry.HasProperty("menuitem", "default", []*core.SchemaMetadata{}) {
			t.Error("Expected HasProperty('menuitem', 'default') to be true")
		}
		if !registry.HasProperty("time", "dateTime", []*core.SchemaMetadata{}) {
			t.Error("Expected HasProperty('time', 'dateTime') to be true")
		}
	})

	t.Run("should detect different kinds of types", func(t *testing.T) {
		setup()
		
		// inheritance: video => media => [HTMLElement] => [Element]
		if !registry.HasProperty("video", "className", []*core.SchemaMetadata{}) {
			t.Error("Expected HasProperty('video', 'className') to be true") // from [Element]
		}
		if !registry.HasProperty("video", "id", []*core.SchemaMetadata{}) {
			t.Error("Expected HasProperty('video', 'id') to be true") // string
		}
		if !registry.HasProperty("video", "scrollLeft", []*core.SchemaMetadata{}) {
			t.Error("Expected HasProperty('video', 'scrollLeft') to be true") // number
		}
		if !registry.HasProperty("video", "height", []*core.SchemaMetadata{}) {
			t.Error("Expected HasProperty('video', 'height') to be true") // number
		}
		if !registry.HasProperty("video", "autoplay", []*core.SchemaMetadata{}) {
			t.Error("Expected HasProperty('video', 'autoplay') to be true") // boolean
		}
		if !registry.HasProperty("video", "classList", []*core.SchemaMetadata{}) {
			t.Error("Expected HasProperty('video', 'classList') to be true") // object
		}
		// from *; but events are not properties
		if registry.HasProperty("video", "click", []*core.SchemaMetadata{}) {
			t.Error("Expected HasProperty('video', 'click') to be false")
		}
	})

	t.Run("should treat custom elements as an unknown element by default", func(t *testing.T) {
		setup()
		
		if registry.HasProperty("custom-like", "unknown", []*core.SchemaMetadata{}) {
			t.Error("Expected HasProperty('custom-like', 'unknown') to be false")
		}
		if !registry.HasProperty("custom-like", "className", []*core.SchemaMetadata{}) {
			t.Error("Expected HasProperty('custom-like', 'className') to be true")
		}
		if !registry.HasProperty("custom-like", "style", []*core.SchemaMetadata{}) {
			t.Error("Expected HasProperty('custom-like', 'style') to be true")
		}
		if !registry.HasProperty("custom-like", "id", []*core.SchemaMetadata{}) {
			t.Error("Expected HasProperty('custom-like', 'id') to be true")
		}
	})

	t.Run("should return true for custom-like elements if the CUSTOM_ELEMENTS_SCHEMA was used", func(t *testing.T) {
		setup()
		
		customSchema := &core.SchemaMetadata{Name: "custom-elements"}
		if !registry.HasProperty("custom-like", "unknown", []*core.SchemaMetadata{customSchema}) {
			t.Error("Expected HasProperty('custom-like', 'unknown', [CUSTOM_ELEMENTS_SCHEMA]) to be true")
		}
		if !registry.HasElement("custom-like", []*core.SchemaMetadata{customSchema}) {
			t.Error("Expected HasElement('custom-like', [CUSTOM_ELEMENTS_SCHEMA]) to be true")
		}
	})

	t.Run("should return true for all elements if the NO_ERRORS_SCHEMA was used", func(t *testing.T) {
		setup()
		
		noErrorsSchema := &core.SchemaMetadata{Name: "no-errors-schema"}
		if !registry.HasProperty("custom-like", "unknown", []*core.SchemaMetadata{noErrorsSchema}) {
			t.Error("Expected HasProperty('custom-like', 'unknown', [NO_ERRORS_SCHEMA]) to be true")
		}
		if !registry.HasProperty("a", "unknown", []*core.SchemaMetadata{noErrorsSchema}) {
			t.Error("Expected HasProperty('a', 'unknown', [NO_ERRORS_SCHEMA]) to be true")
		}
		if !registry.HasElement("custom-like", []*core.SchemaMetadata{noErrorsSchema}) {
			t.Error("Expected HasElement('custom-like', [NO_ERRORS_SCHEMA]) to be true")
		}
		if !registry.HasElement("unknown", []*core.SchemaMetadata{noErrorsSchema}) {
			t.Error("Expected HasElement('unknown', [NO_ERRORS_SCHEMA]) to be true")
		}
	})

	t.Run("should re-map property names that are specified in DOM facade", func(t *testing.T) {
		setup()
		
		if registry.GetMappedPropName("readonly") != "readOnly" {
			t.Errorf("Expected GetMappedPropName('readonly') to be 'readOnly', got %q", registry.GetMappedPropName("readonly"))
		}
	})

	t.Run("should not re-map property names that are not specified in DOM facade", func(t *testing.T) {
		setup()
		
		if registry.GetMappedPropName("title") != "title" {
			t.Errorf("Expected GetMappedPropName('title') to be 'title', got %q", registry.GetMappedPropName("title"))
		}
		if registry.GetMappedPropName("exotic-unknown") != "exotic-unknown" {
			t.Errorf("Expected GetMappedPropName('exotic-unknown') to be 'exotic-unknown', got %q", registry.GetMappedPropName("exotic-unknown"))
		}
	})

	t.Run("should return an error message when asserting event properties", func(t *testing.T) {
		setup()
		
		report := registry.ValidateProperty("onClick")
		if !report.Error {
			t.Error("Expected ValidateProperty('onClick').Error to be true")
		}
		expectedMsg := "Binding to event property 'onClick' is disallowed for security reasons, please use (Click)=...\nIf 'onClick' is a directive input, make sure the directive is imported by the current module."
		if report.Msg != expectedMsg {
			t.Errorf("Expected error message:\n%s\nGot:\n%s", expectedMsg, report.Msg)
		}

		report = registry.ValidateProperty("onAnything")
		if !report.Error {
			t.Error("Expected ValidateProperty('onAnything').Error to be true")
		}
		expectedMsg = "Binding to event property 'onAnything' is disallowed for security reasons, please use (Anything)=...\nIf 'onAnything' is a directive input, make sure the directive is imported by the current module."
		if report.Msg != expectedMsg {
			t.Errorf("Expected error message:\n%s\nGot:\n%s", expectedMsg, report.Msg)
		}
	})

	t.Run("should return an error message when asserting event attributes", func(t *testing.T) {
		setup()
		
		report := registry.ValidateAttribute("onClick")
		if !report.Error {
			t.Error("Expected ValidateAttribute('onClick').Error to be true")
		}
		expectedMsg := "Binding to event attribute 'onClick' is disallowed for security reasons, please use (Click)=..."
		if report.Msg != expectedMsg {
			t.Errorf("Expected error message:\n%s\nGot:\n%s", expectedMsg, report.Msg)
		}

		report = registry.ValidateAttribute("onAnything")
		if !report.Error {
			t.Error("Expected ValidateAttribute('onAnything').Error to be true")
		}
		expectedMsg = "Binding to event attribute 'onAnything' is disallowed for security reasons, please use (Anything)=..."
		if report.Msg != expectedMsg {
			t.Errorf("Expected error message:\n%s\nGot:\n%s", expectedMsg, report.Msg)
		}
	})

	t.Run("should not return an error message when asserting non-event properties or attributes", func(t *testing.T) {
		setup()
		
		report := registry.ValidateProperty("title")
		if report.Error {
			t.Error("Expected ValidateProperty('title').Error to be false")
		}
		if report.Msg != "" {
			t.Errorf("Expected ValidateProperty('title').Msg to be empty, got %q", report.Msg)
		}

		report = registry.ValidateProperty("exotic-unknown")
		if report.Error {
			t.Error("Expected ValidateProperty('exotic-unknown').Error to be false")
		}
		if report.Msg != "" {
			t.Errorf("Expected ValidateProperty('exotic-unknown').Msg to be empty, got %q", report.Msg)
		}
	})

	t.Run("should return security contexts for elements", func(t *testing.T) {
		setup()
		
		if registry.SecurityContext("iframe", "srcdoc", false) != core.SecurityContextHTML {
			t.Error("Expected SecurityContext('iframe', 'srcdoc', false) to be HTML")
		}
		if registry.SecurityContext("p", "innerHTML", false) != core.SecurityContextHTML {
			t.Error("Expected SecurityContext('p', 'innerHTML', false) to be HTML")
		}
		if registry.SecurityContext("a", "href", false) != core.SecurityContextURL {
			t.Error("Expected SecurityContext('a', 'href', false) to be URL")
		}
		if registry.SecurityContext("a", "style", false) != core.SecurityContextSTYLE {
			t.Error("Expected SecurityContext('a', 'style', false) to be STYLE")
		}
		if registry.SecurityContext("ins", "cite", false) != core.SecurityContextURL {
			t.Error("Expected SecurityContext('ins', 'cite', false) to be URL")
		}
		if registry.SecurityContext("base", "href", false) != core.SecurityContextRESOURCE_URL {
			t.Error("Expected SecurityContext('base', 'href', false) to be RESOURCE_URL")
		}
	})

	t.Run("should check security contexts case insensitive", func(t *testing.T) {
		setup()
		
		if registry.SecurityContext("p", "iNnErHtMl", false) != core.SecurityContextHTML {
			t.Error("Expected SecurityContext('p', 'iNnErHtMl', false) to be HTML")
		}
		if registry.SecurityContext("p", "formaction", false) != core.SecurityContextURL {
			t.Error("Expected SecurityContext('p', 'formaction', false) to be URL")
		}
		if registry.SecurityContext("p", "formAction", false) != core.SecurityContextURL {
			t.Error("Expected SecurityContext('p', 'formAction', false) to be URL")
		}
	})

	t.Run("should check security contexts for attributes", func(t *testing.T) {
		setup()
		
		if registry.SecurityContext("p", "innerHtml", true) != core.SecurityContextHTML {
			t.Error("Expected SecurityContext('p', 'innerHtml', true) to be HTML")
		}
		if registry.SecurityContext("p", "formaction", true) != core.SecurityContextURL {
			t.Error("Expected SecurityContext('p', 'formaction', true) to be URL")
		}
	})

	t.Run("Angular custom elements", func(t *testing.T) {
		t.Run("should support <ng-container>", func(t *testing.T) {
			setup()
			
			if registry.HasProperty("ng-container", "id", []*core.SchemaMetadata{}) {
				t.Error("Expected HasProperty('ng-container', 'id') to be false")
			}
		})

		t.Run("should support <ng-content>", func(t *testing.T) {
			setup()
			
			if registry.HasProperty("ng-content", "id", []*core.SchemaMetadata{}) {
				t.Error("Expected HasProperty('ng-content', 'id') to be false")
			}
			if registry.HasProperty("ng-content", "select", []*core.SchemaMetadata{}) {
				t.Error("Expected HasProperty('ng-content', 'select') to be false")
			}
		})
	})

	t.Run("normalizeAnimationStyleProperty", func(t *testing.T) {
		t.Run("should normalize the given CSS property to camelCase", func(t *testing.T) {
			setup()
			
			if registry.NormalizeAnimationStyleProperty("border-radius") != "borderRadius" {
				t.Errorf("Expected NormalizeAnimationStyleProperty('border-radius') to be 'borderRadius', got %q",
					registry.NormalizeAnimationStyleProperty("border-radius"))
			}
			if registry.NormalizeAnimationStyleProperty("zIndex") != "zIndex" {
				t.Errorf("Expected NormalizeAnimationStyleProperty('zIndex') to be 'zIndex', got %q",
					registry.NormalizeAnimationStyleProperty("zIndex"))
			}
			if registry.NormalizeAnimationStyleProperty("-webkit-animation") != "WebkitAnimation" {
				t.Errorf("Expected NormalizeAnimationStyleProperty('-webkit-animation') to be 'WebkitAnimation', got %q",
					registry.NormalizeAnimationStyleProperty("-webkit-animation"))
			}
		})
	})

	t.Run("normalizeAnimationStyleValue", func(t *testing.T) {
		t.Run("should normalize the given dimensional CSS style value to contain a PX value when numeric", func(t *testing.T) {
			setup()
			
			result := registry.NormalizeAnimationStyleValue("borderRadius", "border-radius", 10)
			if result.Value != "10px" {
				t.Errorf("Expected value to be '10px', got %q", result.Value)
			}
		})

		t.Run("should not normalize any values that are of zero", func(t *testing.T) {
			setup()
			
			result := registry.NormalizeAnimationStyleValue("opacity", "opacity", 0)
			if result.Value != "0" {
				t.Errorf("Expected value to be '0', got %q", result.Value)
			}
			
			result = registry.NormalizeAnimationStyleValue("width", "width", 0)
			if result.Value != "0" {
				t.Errorf("Expected value to be '0', got %q", result.Value)
			}
		})

		t.Run("should retain the given dimensional CSS style value's unit if it already exists", func(t *testing.T) {
			setup()
			
			result := registry.NormalizeAnimationStyleValue("borderRadius", "border-radius", "10em")
			if result.Value != "10em" {
				t.Errorf("Expected value to be '10em', got %q", result.Value)
			}
		})

		t.Run("should trim the provided CSS style value", func(t *testing.T) {
			setup()
			
			result := registry.NormalizeAnimationStyleValue("color", "color", "   red ")
			if result.Value != "red" {
				t.Errorf("Expected value to be 'red', got %q", result.Value)
			}
		})

		t.Run("should stringify all non dimensional numeric style values", func(t *testing.T) {
			setup()
			
			result := registry.NormalizeAnimationStyleValue("zIndex", "zIndex", 10)
			if result.Value != "10" {
				t.Errorf("Expected value to be '10', got %q", result.Value)
			}
			
			result = registry.NormalizeAnimationStyleValue("opacity", "opacity", 0.5)
			if result.Value != "0.5" {
				t.Errorf("Expected value to be '0.5', got %q", result.Value)
			}
		})
	})
}

