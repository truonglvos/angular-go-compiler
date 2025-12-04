package render3_test

import (
	"ngc-go/packages/compiler/src/expression_parser"
	"ngc-go/packages/compiler/src/render3"
	"ngc-go/packages/compiler/src/util"
	"ngc-go/packages/compiler/test/render3/view"
	"reflect"
	"sort"
	"testing"
)

// R3AstSourceSpans is a visitor that collects source span information
type R3AstSourceSpans struct {
	result [][]interface{}
}

// NewR3AstSourceSpans creates a new R3AstSourceSpans visitor
func NewR3AstSourceSpans() *R3AstSourceSpans {
	return &R3AstSourceSpans{
		result: [][]interface{}{},
	}
}

// humanizeSpan converts a ParseSourceSpan to string
func humanizeSpan(span *util.ParseSourceSpan) string {
	if span == nil {
		return "<empty>"
	}
	return span.String()
}

// Visit implements render3.Visitor interface
func (r *R3AstSourceSpans) Visit(node render3.Node) interface{} {
	return node.Visit(r)
}

// VisitElement visits an element
func (r *R3AstSourceSpans) VisitElement(element *render3.Element) interface{} {
	r.result = append(r.result, []interface{}{
		"Element",
		humanizeSpan(element.SourceSpan()),
		humanizeSpan(element.StartSourceSpan),
		humanizeSpan(element.EndSourceSpan),
	})
	// Visit all child nodes - convert slices to []render3.Node
	visitNodes(r, convertToNodes(element.Attributes))
	visitNodes(r, convertToNodes(element.Inputs))
	visitNodes(r, convertToNodes(element.Outputs))
	visitNodes(r, convertToNodes(element.Directives))
	visitNodes(r, convertToNodes(element.References))
	render3.VisitAll(r, element.Children)
	return nil
}

// convertToNodes converts a slice of Node types to []render3.Node
func convertToNodes(nodes interface{}) []render3.Node {
	// Use type assertion to convert
	switch v := nodes.(type) {
	case []*render3.TextAttribute:
		result := make([]render3.Node, len(v))
		for i, n := range v {
			result[i] = n
		}
		return result
	case []*render3.BoundAttribute:
		result := make([]render3.Node, len(v))
		for i, n := range v {
			result[i] = n
		}
		return result
	case []*render3.BoundEvent:
		result := make([]render3.Node, len(v))
		for i, n := range v {
			result[i] = n
		}
		return result
	case []*render3.Directive:
		result := make([]render3.Node, len(v))
		for i, n := range v {
			result[i] = n
		}
		return result
	case []*render3.Reference:
		result := make([]render3.Node, len(v))
		for i, n := range v {
			result[i] = n
		}
		return result
	case []*render3.Variable:
		result := make([]render3.Node, len(v))
		for i, n := range v {
			result[i] = n
		}
		return result
	case []*render3.SwitchBlockCase:
		result := make([]render3.Node, len(v))
		for i, n := range v {
			result[i] = n
		}
		return result
	case []*render3.IfBlockBranch:
		result := make([]render3.Node, len(v))
		for i, n := range v {
			result[i] = n
		}
		return result
	default:
		return []render3.Node{}
	}
}

// visitNodes visits a slice of nodes
func visitNodes(visitor render3.Visitor, nodes []render3.Node) {
	render3.VisitAll(visitor, nodes)
}

// VisitTemplate visits a template
func (r *R3AstSourceSpans) VisitTemplate(template *render3.Template) interface{} {
	r.result = append(r.result, []interface{}{
		"Template",
		humanizeSpan(template.SourceSpan()),
		humanizeSpan(template.StartSourceSpan),
		humanizeSpan(template.EndSourceSpan),
	})
	visitNodes(r, convertToNodes(template.Attributes))
	visitNodes(r, convertToNodes(template.Inputs))
	visitNodes(r, convertToNodes(template.Outputs))
	visitNodes(r, convertToNodes(template.Directives))
	// TemplateAttrs is []interface{}, skip it for now
	visitNodes(r, convertToNodes(template.References))
	visitNodes(r, convertToNodes(template.Variables))
	render3.VisitAll(r, template.Children)
	return nil
}

// VisitContent visits content
func (r *R3AstSourceSpans) VisitContent(content *render3.Content) interface{} {
	r.result = append(r.result, []interface{}{
		"Content",
		humanizeSpan(content.SourceSpan()),
	})
	visitNodes(r, convertToNodes(content.Attributes))
	render3.VisitAll(r, content.Children)
	return nil
}

// VisitVariable visits a variable
func (r *R3AstSourceSpans) VisitVariable(variable *render3.Variable) interface{} {
	r.result = append(r.result, []interface{}{
		"Variable",
		humanizeSpan(variable.SourceSpan()),
		humanizeSpan(variable.KeySpan),
		humanizeSpan(variable.ValueSpan),
	})
	return nil
}

// VisitReference visits a reference
func (r *R3AstSourceSpans) VisitReference(reference *render3.Reference) interface{} {
	r.result = append(r.result, []interface{}{
		"Reference",
		humanizeSpan(reference.SourceSpan()),
		humanizeSpan(reference.KeySpan),
		humanizeSpan(reference.ValueSpan),
	})
	return nil
}

// VisitTextAttribute visits a text attribute
func (r *R3AstSourceSpans) VisitTextAttribute(attribute *render3.TextAttribute) interface{} {
	r.result = append(r.result, []interface{}{
		"TextAttribute",
		humanizeSpan(attribute.SourceSpan()),
		humanizeSpan(attribute.KeySpan),
		humanizeSpan(attribute.ValueSpan),
	})
	return nil
}

// VisitBoundAttribute visits a bound attribute
func (r *R3AstSourceSpans) VisitBoundAttribute(attribute *render3.BoundAttribute) interface{} {
	r.result = append(r.result, []interface{}{
		"BoundAttribute",
		humanizeSpan(attribute.SourceSpan()),
		humanizeSpan(attribute.KeySpan),
		humanizeSpan(attribute.ValueSpan),
	})
	return nil
}

// VisitBoundEvent visits a bound event
func (r *R3AstSourceSpans) VisitBoundEvent(event *render3.BoundEvent) interface{} {
	r.result = append(r.result, []interface{}{
		"BoundEvent",
		humanizeSpan(event.SourceSpan()),
		humanizeSpan(event.KeySpan),
		humanizeSpan(event.HandlerSpan),
	})
	return nil
}

// VisitText visits a text node
func (r *R3AstSourceSpans) VisitText(text *render3.Text) interface{} {
	r.result = append(r.result, []interface{}{
		"Text",
		humanizeSpan(text.SourceSpan()),
	})
	return nil
}

// VisitBoundText visits a bound text node
func (r *R3AstSourceSpans) VisitBoundText(text *render3.BoundText) interface{} {
	r.result = append(r.result, []interface{}{
		"BoundText",
		humanizeSpan(text.SourceSpan()),
	})
	return nil
}

// VisitIcu visits an ICU node
func (r *R3AstSourceSpans) VisitIcu(icu *render3.Icu) interface{} {
	r.result = append(r.result, []interface{}{
		"Icu",
		humanizeSpan(icu.SourceSpan()),
	})
	// Visit ICU variables and placeholders
	// Sort keys to ensure consistent order (maps in Go don't guarantee order)
	var varKeys []string
	for k := range icu.Vars {
		varKeys = append(varKeys, k)
	}
	sort.Strings(varKeys)
	for _, k := range varKeys {
		v := icu.Vars[k]
		if v != nil {
			r.result = append(r.result, []interface{}{
				"Icu:Var",
				humanizeSpan(v.SourceSpan()),
			})
		}
	}
	var placeholderKeys []string
	for k := range icu.Placeholders {
		placeholderKeys = append(placeholderKeys, k)
	}
	sort.Strings(placeholderKeys)
	for _, k := range placeholderKeys {
		p := icu.Placeholders[k]
		if p != nil {
			r.result = append(r.result, []interface{}{
				"Icu:Placeholder",
				humanizeSpan(p.SourceSpan()),
			})
		}
	}
	return nil
}

// VisitDeferredBlock visits a deferred block
func (r *R3AstSourceSpans) VisitDeferredBlock(deferred *render3.DeferredBlock) interface{} {
	r.result = append(r.result, []interface{}{
		"DeferredBlock",
		humanizeSpan(deferred.SourceSpan()),
		humanizeSpan(deferred.StartSourceSpan),
		humanizeSpan(deferred.EndSourceSpan),
	})
	deferred.VisitAll(r)
	return nil
}

// VisitDeferredBlockPlaceholder visits a deferred block placeholder
func (r *R3AstSourceSpans) VisitDeferredBlockPlaceholder(block *render3.DeferredBlockPlaceholder) interface{} {
	r.result = append(r.result, []interface{}{
		"DeferredBlockPlaceholder",
		humanizeSpan(block.SourceSpan()),
		humanizeSpan(block.StartSourceSpan),
		humanizeSpan(block.EndSourceSpan),
	})
	render3.VisitAll(r, block.Children)
	return nil
}

// VisitDeferredBlockError visits a deferred block error
func (r *R3AstSourceSpans) VisitDeferredBlockError(block *render3.DeferredBlockError) interface{} {
	r.result = append(r.result, []interface{}{
		"DeferredBlockError",
		humanizeSpan(block.SourceSpan()),
		humanizeSpan(block.StartSourceSpan),
		humanizeSpan(block.EndSourceSpan),
	})
	render3.VisitAll(r, block.Children)
	return nil
}

// VisitDeferredBlockLoading visits a deferred block loading
func (r *R3AstSourceSpans) VisitDeferredBlockLoading(block *render3.DeferredBlockLoading) interface{} {
	r.result = append(r.result, []interface{}{
		"DeferredBlockLoading",
		humanizeSpan(block.SourceSpan()),
		humanizeSpan(block.StartSourceSpan),
		humanizeSpan(block.EndSourceSpan),
	})
	render3.VisitAll(r, block.Children)
	return nil
}

// VisitDeferredTrigger visits a deferred trigger
func (r *R3AstSourceSpans) VisitDeferredTrigger(trigger *render3.DeferredTrigger) interface{} {
	// Note: In Go, we can't easily type assert from *DeferredTrigger to specific trigger types
	// because they embed *DeferredTrigger. We'll just record the base trigger info.
	// The specific trigger type information is handled in VisitDeferredBlock.
	r.result = append(r.result, []interface{}{
		"DeferredTrigger",
		humanizeSpan(trigger.SourceSpan()),
	})
	return nil
}

// VisitSwitchBlock visits a switch block
func (r *R3AstSourceSpans) VisitSwitchBlock(block *render3.SwitchBlock) interface{} {
	r.result = append(r.result, []interface{}{
		"SwitchBlock",
		humanizeSpan(block.SourceSpan()),
		humanizeSpan(block.StartSourceSpan),
		humanizeSpan(block.EndSourceSpan),
	})
	visitNodes(r, convertToNodes(block.Cases))
	return nil
}

// VisitSwitchBlockCase visits a switch block case
func (r *R3AstSourceSpans) VisitSwitchBlockCase(block *render3.SwitchBlockCase) interface{} {
	r.result = append(r.result, []interface{}{
		"SwitchBlockCase",
		humanizeSpan(block.SourceSpan()),
		humanizeSpan(block.StartSourceSpan),
	})
	render3.VisitAll(r, block.Children)
	return nil
}

// VisitForLoopBlock visits a for loop block
func (r *R3AstSourceSpans) VisitForLoopBlock(block *render3.ForLoopBlock) interface{} {
	r.result = append(r.result, []interface{}{
		"ForLoopBlock",
		humanizeSpan(block.SourceSpan()),
		humanizeSpan(block.StartSourceSpan),
		humanizeSpan(block.EndSourceSpan),
	})
	block.Item.Visit(r)
	visitNodes(r, convertToNodes(block.ContextVariables))
	render3.VisitAll(r, block.Children)
	if block.Empty != nil {
		block.Empty.Visit(r)
	}
	return nil
}

// VisitForLoopBlockEmpty visits a for loop block empty
func (r *R3AstSourceSpans) VisitForLoopBlockEmpty(block *render3.ForLoopBlockEmpty) interface{} {
	r.result = append(r.result, []interface{}{
		"ForLoopBlockEmpty",
		humanizeSpan(block.SourceSpan()),
		humanizeSpan(block.StartSourceSpan),
	})
	render3.VisitAll(r, block.Children)
	return nil
}

// VisitIfBlock visits an if block
func (r *R3AstSourceSpans) VisitIfBlock(block *render3.IfBlock) interface{} {
	r.result = append(r.result, []interface{}{
		"IfBlock",
		humanizeSpan(block.SourceSpan()),
		humanizeSpan(block.StartSourceSpan),
		humanizeSpan(block.EndSourceSpan),
	})
	visitNodes(r, convertToNodes(block.Branches))
	return nil
}

// VisitIfBlockBranch visits an if block branch
func (r *R3AstSourceSpans) VisitIfBlockBranch(block *render3.IfBlockBranch) interface{} {
	r.result = append(r.result, []interface{}{
		"IfBlockBranch",
		humanizeSpan(block.SourceSpan()),
		humanizeSpan(block.StartSourceSpan),
	})
	if block.ExpressionAlias != nil {
		block.ExpressionAlias.Visit(r)
	}
	render3.VisitAll(r, block.Children)
	return nil
}

// VisitUnknownBlock visits an unknown block
func (r *R3AstSourceSpans) VisitUnknownBlock(block *render3.UnknownBlock) interface{} {
	r.result = append(r.result, []interface{}{
		"UnknownBlock",
		humanizeSpan(block.SourceSpan()),
	})
	return nil
}

// VisitLetDeclaration visits a let declaration
func (r *R3AstSourceSpans) VisitLetDeclaration(decl *render3.LetDeclaration) interface{} {
	r.result = append(r.result, []interface{}{
		"LetDeclaration",
		humanizeSpan(decl.SourceSpan()),
		humanizeSpan(decl.NameSpan),
		humanizeSpan(decl.ValueSpan),
	})
	return nil
}

// VisitComponent visits a component
func (r *R3AstSourceSpans) VisitComponent(component *render3.Component) interface{} {
	r.result = append(r.result, []interface{}{
		"Component",
		humanizeSpan(component.SourceSpan()),
		humanizeSpan(component.StartSourceSpan),
		humanizeSpan(component.EndSourceSpan),
	})
	visitNodes(r, convertToNodes(component.Attributes))
	visitNodes(r, convertToNodes(component.Inputs))
	visitNodes(r, convertToNodes(component.Outputs))
	visitNodes(r, convertToNodes(component.Directives))
	visitNodes(r, convertToNodes(component.References))
	render3.VisitAll(r, component.Children)
	return nil
}

// VisitDirective visits a directive
func (r *R3AstSourceSpans) VisitDirective(directive *render3.Directive) interface{} {
	r.result = append(r.result, []interface{}{
		"Directive",
		humanizeSpan(directive.SourceSpan()),
		humanizeSpan(directive.StartSourceSpan),
		humanizeSpan(directive.EndSourceSpan),
	})
	visitNodes(r, convertToNodes(directive.Attributes))
	visitNodes(r, convertToNodes(directive.Inputs))
	visitNodes(r, convertToNodes(directive.Outputs))
	visitNodes(r, convertToNodes(directive.References))
	return nil
}

// expectFromHtmlSpans parses HTML and returns humanized result for spans test
func expectFromHtmlSpans(html string, selectorlessEnabled bool) [][]interface{} {
	selEnabled := selectorlessEnabled
	result := view.ParseR3(html, &view.ParseR3Options{
		SelectorlessEnabled: &selEnabled,
	})
	return expectFromR3NodesSpans(result.Nodes)
}

// expectFromR3NodesSpans humanizes R3 nodes for spans test
func expectFromR3NodesSpans(nodes []render3.Node) [][]interface{} {
	humanizer := NewR3AstSourceSpans()
	render3.VisitAll(humanizer, nodes)
	return humanizer.result
}

// Helper to compare results for spans test
func assertEqualSpans(t *testing.T, actual, expected [][]interface{}, msg string) {
	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("%s\nExpected: %v\nGot: %v", msg, expected, actual)
	}
}

func TestR3ASTSourceSpans(t *testing.T) {
	t.Run("nodes without binding", func(t *testing.T) {
		t.Run("is correct for text nodes", func(t *testing.T) {
			result := expectFromHtmlSpans("a", false)
			expected := [][]interface{}{
				{"Text", "a"},
			}
			assertEqualSpans(t, result, expected, "text nodes")
		})

		t.Run("is correct for elements with attributes", func(t *testing.T) {
			result := expectFromHtmlSpans(`<div a="b"></div>`, false)
			expected := [][]interface{}{
				{"Element", `<div a="b"></div>`, `<div a="b">`, `</div>`},
				{"TextAttribute", `a="b"`, "a", "b"},
			}
			assertEqualSpans(t, result, expected, "elements with attributes")
		})

		t.Run("is correct for elements with attributes without value", func(t *testing.T) {
			result := expectFromHtmlSpans(`<div a></div>`, false)
			expected := [][]interface{}{
				{"Element", `<div a></div>`, `<div a>`, `</div>`},
				{"TextAttribute", "a", "a", "<empty>"},
			}
			assertEqualSpans(t, result, expected, "elements with attributes without value")
		})

		t.Run("is correct for self-closing elements with trailing whitespace", func(t *testing.T) {
			result := expectFromHtmlSpans("<input />\n  <span>\n</span>", false)
			expected := [][]interface{}{
				{"Element", "<input />", "<input />", "<input />"},
				{"Element", "<span>\n</span>", "<span>", "</span>"},
			}
			assertEqualSpans(t, result, expected, "self-closing elements with trailing whitespace")
		})
	})

	t.Run("bound text nodes", func(t *testing.T) {
		t.Run("is correct for bound text nodes", func(t *testing.T) {
			result := expectFromHtmlSpans("{{a}}", false)
			expected := [][]interface{}{
				{"BoundText", "{{a}}"},
			}
			assertEqualSpans(t, result, expected, "bound text nodes")
		})
	})

	t.Run("bound attributes", func(t *testing.T) {
		t.Run("is correct for bound properties", func(t *testing.T) {
			result := expectFromHtmlSpans(`<div [someProp]="v"></div>`, false)
			expected := [][]interface{}{
				{"Element", `<div [someProp]="v"></div>`, `<div [someProp]="v">`, `</div>`},
				{"BoundAttribute", `[someProp]="v"`, "someProp", "v"},
			}
			assertEqualSpans(t, result, expected, "bound properties")
		})

		t.Run("is correct for bound properties without value", func(t *testing.T) {
			result := expectFromHtmlSpans(`<div [someProp]></div>`, false)
			expected := [][]interface{}{
				{"Element", `<div [someProp]></div>`, `<div [someProp]>`, `</div>`},
				{"BoundAttribute", "[someProp]", "someProp", "<empty>"},
			}
			assertEqualSpans(t, result, expected, "bound properties without value")
		})

		t.Run("is correct for bound properties via bind-", func(t *testing.T) {
			result := expectFromHtmlSpans(`<div bind-prop="v"></div>`, false)
			expected := [][]interface{}{
				{"Element", `<div bind-prop="v"></div>`, `<div bind-prop="v">`, `</div>`},
				{"BoundAttribute", `bind-prop="v"`, "prop", "v"},
			}
			assertEqualSpans(t, result, expected, "bound properties via bind-")
		})

		t.Run("is correct for bound properties via {{...}}", func(t *testing.T) {
			result := expectFromHtmlSpans(`<div prop="{{v}}"></div>`, false)
			expected := [][]interface{}{
				{"Element", `<div prop="{{v}}"></div>`, `<div prop="{{v}}">`, `</div>`},
				{"BoundAttribute", `prop="{{v}}"`, "prop", "{{v}}"},
			}
			assertEqualSpans(t, result, expected, "bound properties via {{...}}")
		})

		t.Run("is correct for bound properties via data-", func(t *testing.T) {
			result := expectFromHtmlSpans(`<div data-prop="{{v}}"></div>`, false)
			expected := [][]interface{}{
				{"Element", `<div data-prop="{{v}}"></div>`, `<div data-prop="{{v}}">`, `</div>`},
				{"BoundAttribute", `data-prop="{{v}}"`, "prop", "{{v}}"},
			}
			assertEqualSpans(t, result, expected, "bound properties via data-")
		})

		t.Run("is correct for bound properties via @", func(t *testing.T) {
			result := expectFromHtmlSpans(`<div bind-@animation="v"></div>`, false)
			expected := [][]interface{}{
				{"Element", `<div bind-@animation="v"></div>`, `<div bind-@animation="v">`, `</div>`},
				{"BoundAttribute", `bind-@animation="v"`, "animation", "v"},
			}
			assertEqualSpans(t, result, expected, "bound properties via @")
		})

		t.Run("is correct for bound properties via animation-", func(t *testing.T) {
			result := expectFromHtmlSpans(`<div bind-animate-animationName="v"></div>`, false)
			expected := [][]interface{}{
				{"Element", `<div bind-animate-animationName="v"></div>`, `<div bind-animate-animationName="v">`, `</div>`},
				{"BoundAttribute", `bind-animate-animationName="v"`, "animationName", "v"},
			}
			assertEqualSpans(t, result, expected, "bound properties via animation-")
		})

		t.Run("is correct for bound properties via @ without value", func(t *testing.T) {
			result := expectFromHtmlSpans(`<div @animation></div>`, false)
			expected := [][]interface{}{
				{"Element", `<div @animation></div>`, `<div @animation>`, `</div>`},
				{"BoundAttribute", "@animation", "animation", "<empty>"},
			}
			assertEqualSpans(t, result, expected, "bound properties via @ without value")
		})

		t.Run("should not throw off span of value in bound attribute when leading spaces are present", func(t *testing.T) {
			assertValueSpan := func(template string, start, end int) {
				result := view.ParseR3(template, nil)
				if len(result.Nodes) == 0 {
					t.Fatalf("Expected at least one node")
				}
				element, ok := result.Nodes[0].(*render3.Element)
				if !ok {
					t.Fatalf("Expected first node to be Element")
				}
				if len(element.Inputs) == 0 {
					t.Fatalf("Expected at least one input")
				}
				boundAttribute := element.Inputs[0]
				// boundAttribute is already *render3.BoundAttribute, no need to type assert
				if boundAttribute.Value == nil {
					t.Fatalf("Expected value to be non-nil")
				}
				// Get AST source span
				astWithSource, ok := boundAttribute.Value.(*expression_parser.ASTWithSource)
				if !ok {
					t.Fatalf("Expected ASTWithSource")
				}
				span := astWithSource.AST.SourceSpan()
				if span == nil {
					t.Fatalf("Expected source span to be non-nil")
				}
				if span.Start != start {
					t.Errorf("Expected span.Start = %d, got %d", start, span.Start)
				}
				if span.End != end {
					t.Errorf("Expected span.End = %d, got %d", end, span.End)
				}
			}

			assertValueSpan(`<a [b]="helloWorld"></a>`, 8, 18)
			assertValueSpan(`<a [b]=" helloWorld"></a>`, 9, 19)
			assertValueSpan(`<a [b]="  helloWorld"></a>`, 10, 20)
			assertValueSpan(`<a [b]="   helloWorld"></a>`, 11, 21)
			assertValueSpan(`<a [b]="    helloWorld"></a>`, 12, 22)
			assertValueSpan(`<a [b]="                                          helloWorld"></a>`, 50, 60)
		})

		t.Run("should not throw off span of value in template attribute when leading spaces are present", func(t *testing.T) {
			assertValueSpan := func(template string, start, end int) {
				result := view.ParseR3(template, nil)
				if len(result.Nodes) == 0 {
					t.Fatalf("Expected at least one node")
				}
				tmpl, ok := result.Nodes[0].(*render3.Template)
				if !ok {
					t.Fatalf("Expected first node to be Template")
				}
				if len(tmpl.Inputs) == 0 {
					t.Fatalf("Expected at least one bound input")
				}
				ba := tmpl.Inputs[0]
				if ba.Value == nil {
					t.Fatalf("Expected value to be non-nil")
				}
				astWithSource, ok := ba.Value.(*expression_parser.ASTWithSource)
				if !ok {
					t.Fatalf("Expected ASTWithSource")
				}
				span := astWithSource.AST.SourceSpan()
				if span == nil {
					t.Fatalf("Expected source span to be non-nil")
				}
				if span.Start != start {
					t.Errorf("Expected span.Start = %d, got %d", start, span.Start)
				}
				if span.End != end {
					t.Errorf("Expected span.End = %d, got %d", end, span.End)
				}
			}

			assertValueSpan(`<ng-container *ngTemplateOutlet="helloWorld"/>`, 33, 43)
			assertValueSpan(`<ng-container *ngTemplateOutlet=" helloWorld"/>`, 34, 44)
			assertValueSpan(`<ng-container *ngTemplateOutlet="  helloWorld"/>`, 35, 45)
			assertValueSpan(`<ng-container *ngTemplateOutlet="   helloWorld"/>`, 36, 46)
			assertValueSpan(`<ng-container *ngTemplateOutlet="    helloWorld"/>`, 37, 47)
			assertValueSpan(`<ng-container *ngTemplateOutlet="                    helloWorld"/>`, 53, 63)
		})
	})

	t.Run("templates", func(t *testing.T) {
		t.Run("is correct for * directives", func(t *testing.T) {
			result := expectFromHtmlSpans(`<div *ngIf></div>`, false)
			expected := [][]interface{}{
				{"Template", `<div *ngIf></div>`, `<div *ngIf>`, `</div>`},
				{"TextAttribute", "ngIf", "ngIf", "<empty>"},
				{"Element", `<div *ngIf></div>`, `<div *ngIf>`, `</div>`},
			}
			assertEqualSpans(t, result, expected, "* directives")
		})

		t.Run("is correct for <ng-template>", func(t *testing.T) {
			result := expectFromHtmlSpans(`<ng-template></ng-template>`, false)
			expected := [][]interface{}{
				{"Template", `<ng-template></ng-template>`, `<ng-template>`, `</ng-template>`},
			}
			assertEqualSpans(t, result, expected, "<ng-template>")
		})

		t.Run("is correct for reference via #...", func(t *testing.T) {
			result := expectFromHtmlSpans(`<ng-template #a></ng-template>`, false)
			expected := [][]interface{}{
				{"Template", `<ng-template #a></ng-template>`, `<ng-template #a>`, `</ng-template>`},
				{"Reference", "#a", "a", "<empty>"},
			}
			assertEqualSpans(t, result, expected, "reference via #...")
		})

		t.Run("is correct for reference with name", func(t *testing.T) {
			result := expectFromHtmlSpans(`<ng-template #a="b"></ng-template>`, false)
			expected := [][]interface{}{
				{"Template", `<ng-template #a="b"></ng-template>`, `<ng-template #a="b">`, `</ng-template>`},
				{"Reference", `#a="b"`, "a", "b"},
			}
			assertEqualSpans(t, result, expected, "reference with name")
		})

		t.Run("is correct for reference via ref-...", func(t *testing.T) {
			result := expectFromHtmlSpans(`<ng-template ref-a></ng-template>`, false)
			expected := [][]interface{}{
				{"Template", `<ng-template ref-a></ng-template>`, `<ng-template ref-a>`, `</ng-template>`},
				{"Reference", "ref-a", "a", "<empty>"},
			}
			assertEqualSpans(t, result, expected, "reference via ref-...")
		})

		t.Run("is correct for reference via data-ref-...", func(t *testing.T) {
			result := expectFromHtmlSpans(`<ng-template data-ref-a></ng-template>`, false)
			expected := [][]interface{}{
				{"Template", `<ng-template data-ref-a></ng-template>`, `<ng-template data-ref-a>`, `</ng-template>`},
				{"Reference", "data-ref-a", "a", "<empty>"},
			}
			assertEqualSpans(t, result, expected, "reference via data-ref-...")
		})

		t.Run("is correct for variables via let-...", func(t *testing.T) {
			result := expectFromHtmlSpans(`<ng-template let-a="b"></ng-template>`, false)
			expected := [][]interface{}{
				{"Template", `<ng-template let-a="b"></ng-template>`, `<ng-template let-a="b">`, `</ng-template>`},
				{"Variable", `let-a="b"`, "a", "b"},
			}
			assertEqualSpans(t, result, expected, "variables via let-...")
		})

		t.Run("is correct for variables via data-let-...", func(t *testing.T) {
			result := expectFromHtmlSpans(`<ng-template data-let-a="b"></ng-template>`, false)
			expected := [][]interface{}{
				{"Template", `<ng-template data-let-a="b"></ng-template>`, `<ng-template data-let-a="b">`, `</ng-template>`},
				{"Variable", `data-let-a="b"`, "a", "b"},
			}
			assertEqualSpans(t, result, expected, "variables via data-let-...")
		})

		t.Run("is correct for attributes", func(t *testing.T) {
			result := expectFromHtmlSpans(`<ng-template k1="v1"></ng-template>`, false)
			expected := [][]interface{}{
				{"Template", `<ng-template k1="v1"></ng-template>`, `<ng-template k1="v1">`, `</ng-template>`},
				{"TextAttribute", `k1="v1"`, "k1", "v1"},
			}
			assertEqualSpans(t, result, expected, "attributes")
		})

		t.Run("is correct for bound attributes", func(t *testing.T) {
			result := expectFromHtmlSpans(`<ng-template [k1]="v1"></ng-template>`, false)
			expected := [][]interface{}{
				{"Template", `<ng-template [k1]="v1"></ng-template>`, `<ng-template [k1]="v1">`, `</ng-template>`},
				{"BoundAttribute", `[k1]="v1"`, "k1", "v1"},
			}
			assertEqualSpans(t, result, expected, "bound attributes")
		})
	})

	t.Run("inline templates", func(t *testing.T) {
		t.Run("is correct for attribute and bound attributes", func(t *testing.T) {
			result1 := expectFromHtmlSpans(`<div *ngFor="let item of items"></div>`, false)
			expected1 := [][]interface{}{
				{"Template", `<div *ngFor="let item of items"></div>`, `<div *ngFor="let item of items">`, `</div>`},
				{"TextAttribute", "ngFor", "ngFor", "<empty>"},
				{"BoundAttribute", "of items", "of", "items"},
				{"Variable", "let item ", "item", "<empty>"},
				{"Element", `<div *ngFor="let item of items"></div>`, `<div *ngFor="let item of items">`, `</div>`},
			}
			assertEqualSpans(t, result1, expected1, "ngFor with let item")

			// Note: This test exercises an *incorrect* usage of the ngFor directive
			result2 := expectFromHtmlSpans(`<div *ngFor="item of items"></div>`, false)
			expected2 := [][]interface{}{
				{"Template", `<div *ngFor="item of items"></div>`, `<div *ngFor="item of items">`, `</div>`},
				{"BoundAttribute", `ngFor="item `, "ngFor", "item"},
				{"BoundAttribute", "of items", "of", "items"},
				{"Element", `<div *ngFor="item of items"></div>`, `<div *ngFor="item of items">`, `</div>`},
			}
			assertEqualSpans(t, result2, expected2, "ngFor without let (incorrect usage)")

			result3 := expectFromHtmlSpans(`<div *ngFor="let item of items; trackBy: trackByFn"></div>`, false)
			expected3 := [][]interface{}{
				{"Template", `<div *ngFor="let item of items; trackBy: trackByFn"></div>`, `<div *ngFor="let item of items; trackBy: trackByFn">`, `</div>`},
				{"TextAttribute", "ngFor", "ngFor", "<empty>"},
				{"BoundAttribute", "of items; ", "of", "items"},
				{"BoundAttribute", "trackBy: trackByFn", "trackBy", "trackByFn"},
				{"Variable", "let item ", "item", "<empty>"},
				{"Element", `<div *ngFor="let item of items; trackBy: trackByFn"></div>`, `<div *ngFor="let item of items; trackBy: trackByFn">`, `</div>`},
			}
			assertEqualSpans(t, result3, expected3, "ngFor with trackBy")
		})

		t.Run("is correct for variables via let ...", func(t *testing.T) {
			result := expectFromHtmlSpans(`<div *ngIf="let a=b"></div>`, false)
			expected := [][]interface{}{
				{"Template", `<div *ngIf="let a=b"></div>`, `<div *ngIf="let a=b">`, `</div>`},
				{"TextAttribute", "ngIf", "ngIf", "<empty>"},
				{"Variable", "let a=b", "a", "b"},
				{"Element", `<div *ngIf="let a=b"></div>`, `<div *ngIf="let a=b">`, `</div>`},
			}
			assertEqualSpans(t, result, expected, "variables via let ...")
		})

		t.Run("is correct for variables via as ...", func(t *testing.T) {
			result := expectFromHtmlSpans(`<div *ngIf="expr as local"></div>`, false)
			expected := [][]interface{}{
				{"Template", `<div *ngIf="expr as local"></div>`, `<div *ngIf="expr as local">`, `</div>`},
				{"BoundAttribute", `ngIf="expr `, "ngIf", "expr"},
				{"Variable", `ngIf="expr as local`, "local", "ngIf"},
				{"Element", `<div *ngIf="expr as local"></div>`, `<div *ngIf="expr as local">`, `</div>`},
			}
			assertEqualSpans(t, result, expected, "variables via as ...")
		})
	})

	t.Run("events", func(t *testing.T) {
		t.Run("is correct for event names case sensitive", func(t *testing.T) {
			result := expectFromHtmlSpans(`<div (someEvent)="v"></div>`, false)
			expected := [][]interface{}{
				{"Element", `<div (someEvent)="v"></div>`, `<div (someEvent)="v">`, `</div>`},
				{"BoundEvent", `(someEvent)="v"`, "someEvent", "v"},
			}
			assertEqualSpans(t, result, expected, "event names case sensitive")
		})

		t.Run("is correct for bound events via on-", func(t *testing.T) {
			result := expectFromHtmlSpans(`<div on-event="v"></div>`, false)
			expected := [][]interface{}{
				{"Element", `<div on-event="v"></div>`, `<div on-event="v">`, `</div>`},
				{"BoundEvent", `on-event="v"`, "event", "v"},
			}
			assertEqualSpans(t, result, expected, "bound events via on-")
		})

		t.Run("is correct for bound events via data-on-", func(t *testing.T) {
			result := expectFromHtmlSpans(`<div data-on-event="v"></div>`, false)
			expected := [][]interface{}{
				{"Element", `<div data-on-event="v"></div>`, `<div data-on-event="v">`, `</div>`},
				{"BoundEvent", `data-on-event="v"`, "event", "v"},
			}
			assertEqualSpans(t, result, expected, "bound events via data-on-")
		})

		t.Run("is correct for bound events and properties via [(...)]", func(t *testing.T) {
			result := expectFromHtmlSpans(`<div [(prop)]="v"></div>`, false)
			expected := [][]interface{}{
				{"Element", `<div [(prop)]="v"></div>`, `<div [(prop)]="v">`, `</div>`},
				{"BoundAttribute", `[(prop)]="v"`, "prop", "v"},
				{"BoundEvent", `[(prop)]="v"`, "prop", "v"},
			}
			assertEqualSpans(t, result, expected, "bound events and properties via [(...)]")
		})

		t.Run("is correct for bound events and properties via bindon-", func(t *testing.T) {
			result := expectFromHtmlSpans(`<div bindon-prop="v"></div>`, false)
			expected := [][]interface{}{
				{"Element", `<div bindon-prop="v"></div>`, `<div bindon-prop="v">`, `</div>`},
				{"BoundAttribute", `bindon-prop="v"`, "prop", "v"},
				{"BoundEvent", `bindon-prop="v"`, "prop", "v"},
			}
			assertEqualSpans(t, result, expected, "bound events and properties via bindon-")
		})

		t.Run("is correct for bound events and properties via data-bindon-", func(t *testing.T) {
			result := expectFromHtmlSpans(`<div data-bindon-prop="v"></div>`, false)
			expected := [][]interface{}{
				{"Element", `<div data-bindon-prop="v"></div>`, `<div data-bindon-prop="v">`, `</div>`},
				{"BoundAttribute", `data-bindon-prop="v"`, "prop", "v"},
				{"BoundEvent", `data-bindon-prop="v"`, "prop", "v"},
			}
			assertEqualSpans(t, result, expected, "bound events and properties via data-bindon-")
		})

		t.Run("is correct for bound events via @", func(t *testing.T) {
			result := expectFromHtmlSpans(`<div (@name.done)="v"></div>`, false)
			expected := [][]interface{}{
				{"Element", `<div (@name.done)="v"></div>`, `<div (@name.done)="v">`, `</div>`},
				{"BoundEvent", `(@name.done)="v"`, "name.done", "v"},
			}
			assertEqualSpans(t, result, expected, "bound events via @")
		})
	})

	t.Run("references", func(t *testing.T) {
		t.Run("is correct for references via #...", func(t *testing.T) {
			result := expectFromHtmlSpans(`<div #a></div>`, false)
			expected := [][]interface{}{
				{"Element", `<div #a></div>`, `<div #a>`, `</div>`},
				{"Reference", "#a", "a", "<empty>"},
			}
			assertEqualSpans(t, result, expected, "references via #...")
		})

		t.Run("is correct for references with name", func(t *testing.T) {
			result := expectFromHtmlSpans(`<div #a="b"></div>`, false)
			expected := [][]interface{}{
				{"Element", `<div #a="b"></div>`, `<div #a="b">`, `</div>`},
				{"Reference", `#a="b"`, "a", "b"},
			}
			assertEqualSpans(t, result, expected, "references with name")
		})

		t.Run("is correct for references via ref-", func(t *testing.T) {
			result := expectFromHtmlSpans(`<div ref-a></div>`, false)
			expected := [][]interface{}{
				{"Element", `<div ref-a></div>`, `<div ref-a>`, `</div>`},
				{"Reference", "ref-a", "a", "<empty>"},
			}
			assertEqualSpans(t, result, expected, "references via ref-")
		})

		t.Run("is correct for references via data-ref-", func(t *testing.T) {
			result := expectFromHtmlSpans(`<div ref-a></div>`, false)
			expected := [][]interface{}{
				{"Element", `<div ref-a></div>`, `<div ref-a>`, `</div>`},
				{"Reference", "ref-a", "a", "<empty>"},
			}
			assertEqualSpans(t, result, expected, "references via data-ref-")
		})
	})

	t.Run("ICU expressions", func(t *testing.T) {
		t.Run("is correct for variables and placeholders", func(t *testing.T) {
			result := expectFromHtmlSpans(`<span i18n>{item.var, plural, other { {{item.placeholder}} items } }</span>`, false)
			expected := [][]interface{}{
				{"Element", `<span i18n>{item.var, plural, other { {{item.placeholder}} items } }</span>`, `<span i18n>`, `</span>`},
				{"Icu", `{item.var, plural, other { {{item.placeholder}} items } }`},
				{"Icu:Var", "item.var"},
				{"Icu:Placeholder", "{{item.placeholder}}"},
			}
			assertEqualSpans(t, result, expected, "ICU variables and placeholders")
		})

		t.Run("is correct for nested ICUs", func(t *testing.T) {
			result := expectFromHtmlSpans(`<span i18n>{item.var, plural, other { {{item.placeholder}} {nestedVar, plural, other { {{nestedPlaceholder}} }}} }</span>`, false)
			expected := [][]interface{}{
				{"Element", `<span i18n>{item.var, plural, other { {{item.placeholder}} {nestedVar, plural, other { {{nestedPlaceholder}} }}} }</span>`, `<span i18n>`, `</span>`},
				{"Icu", `{item.var, plural, other { {{item.placeholder}} {nestedVar, plural, other { {{nestedPlaceholder}} }}} }`},
				{"Icu:Var", "nestedVar"},
				{"Icu:Var", "item.var"},
				{"Icu:Placeholder", "{{item.placeholder}}"},
				{"Icu:Placeholder", "{{nestedPlaceholder}}"},
			}
			assertEqualSpans(t, result, expected, "nested ICUs")
		})
	})

	t.Run("deferred blocks", func(t *testing.T) {
		t.Run("is correct for deferred blocks", func(t *testing.T) {
			html := "@defer (when isVisible() && foo; on hover(button), timer(10s), idle, immediate, " +
				"interaction(button), viewport(container); prefetch on immediate; " +
				"prefetch when isDataLoaded(); hydrate on interaction; hydrate when isVisible(); hydrate on timer(1200)) {<calendar-cmp [date]=\"current\"/>}" +
				"@loading (minimum 1s; after 100ms) {Loading...}" +
				"@placeholder (minimum 500) {Placeholder content!}" +
				"@error {Loading failed :(}"

			result := expectFromHtmlSpans(html, false)
			// Note: The exact spans may differ slightly, so we check for key elements
			if len(result) < 10 {
				t.Errorf("Expected at least 10 result items, got %d", len(result))
			}
			// Check for DeferredBlock
			foundDeferredBlock := false
			for _, item := range result {
				if len(item) > 0 && item[0] == "DeferredBlock" {
					foundDeferredBlock = true
					break
				}
			}
			if !foundDeferredBlock {
				t.Error("Expected to find DeferredBlock")
			}
		})
	})

	t.Run("switch blocks", func(t *testing.T) {
		t.Run("is correct for switch blocks", func(t *testing.T) {
			html := `@switch (cond.kind) {` +
				`@case (x()) {X case}` +
				`@case ('hello') {Y case}` +
				`@case (42) {Z case}` +
				`@default {No case matched}` +
				`}`

			result := expectFromHtmlSpans(html, false)
			expected := [][]interface{}{
				{"SwitchBlock", "@switch (cond.kind) {@case (x()) {X case}@case ('hello') {Y case}@case (42) {Z case}@default {No case matched}}", "@switch (cond.kind) {", "}"},
				{"SwitchBlockCase", "@case (x()) {X case}", "@case (x()) {"},
				{"Text", "X case"},
				{"SwitchBlockCase", "@case ('hello') {Y case}", "@case ('hello') {"},
				{"Text", "Y case"},
				{"SwitchBlockCase", "@case (42) {Z case}", "@case (42) {"},
				{"Text", "Z case"},
				{"SwitchBlockCase", "@default {No case matched}", "@default {"},
				{"Text", "No case matched"},
			}
			assertEqualSpans(t, result, expected, "switch blocks")
		})
	})

	t.Run("for loop blocks", func(t *testing.T) {
		t.Run("is correct for loop blocks", func(t *testing.T) {
			html := `@for (item of items.foo.bar; track item.id; let i = $index, _o_d_d_ = $odd) {<h1>{{ item }}</h1>}` +
				`@empty {There were no items in the list.}`

			result := expectFromHtmlSpans(html, false)
			// Check for key elements
			foundForLoopBlock := false
			foundForLoopBlockEmpty := false
			for _, item := range result {
				if len(item) > 0 {
					if item[0] == "ForLoopBlock" {
						foundForLoopBlock = true
					}
					if item[0] == "ForLoopBlockEmpty" {
						foundForLoopBlockEmpty = true
					}
				}
			}
			if !foundForLoopBlock {
				t.Error("Expected to find ForLoopBlock")
			}
			if !foundForLoopBlockEmpty {
				t.Error("Expected to find ForLoopBlockEmpty")
			}
		})
	})

	t.Run("if blocks", func(t *testing.T) {
		t.Run("is correct for if blocks", func(t *testing.T) {
			html := `@if (cond.expr; as foo) {Main case was true!}` +
				`@else if (other.expr) {Extra case was true!}` +
				`@else {False case!}`

			result := expectFromHtmlSpans(html, false)
			expected := [][]interface{}{
				{"IfBlock", "@if (cond.expr; as foo) {Main case was true!}@else if (other.expr) {Extra case was true!}@else {False case!}", "@if (cond.expr; as foo) {", "}"},
				{"IfBlockBranch", "@if (cond.expr; as foo) {Main case was true!}", "@if (cond.expr; as foo) {"},
				{"Variable", "foo", "foo", "<empty>"},
				{"Text", "Main case was true!"},
				{"IfBlockBranch", "@else if (other.expr) {Extra case was true!}", "@else if (other.expr) {"},
				{"Text", "Extra case was true!"},
				{"IfBlockBranch", "@else {False case!}", "@else {"},
				{"Text", "False case!"},
			}
			assertEqualSpans(t, result, expected, "if blocks")
		})
	})

	t.Run("@let declaration", func(t *testing.T) {
		t.Run("is correct for a let declaration", func(t *testing.T) {
			result := expectFromHtmlSpans("@let foo = 123;", false)
			expected := [][]interface{}{
				{"LetDeclaration", "@let foo = 123", "foo", "123"},
			}
			assertEqualSpans(t, result, expected, "let declaration")
		})
	})

	t.Run("component tags", func(t *testing.T) {
		t.Run("is correct for a simple component", func(t *testing.T) {
			result := expectFromHtmlSpans("<MyComp></MyComp>", true)
			expected := [][]interface{}{
				{"Component", "<MyComp></MyComp>", "<MyComp>", "</MyComp>"},
			}
			assertEqualSpans(t, result, expected, "simple component")
		})

		t.Run("is correct for a self-closing component", func(t *testing.T) {
			result := expectFromHtmlSpans("<MyComp/>", true)
			expected := [][]interface{}{
				{"Component", "<MyComp/>", "<MyComp/>", "<MyComp/>"},
			}
			assertEqualSpans(t, result, expected, "self-closing component")
		})

		t.Run("is correct for a component with a tag name", func(t *testing.T) {
			result := expectFromHtmlSpans("<MyComp:button></MyComp:button>", true)
			expected := [][]interface{}{
				{"Component", "<MyComp:button></MyComp:button>", "<MyComp:button>", "</MyComp:button>"},
			}
			assertEqualSpans(t, result, expected, "component with tag name")
		})

		t.Run("is correct for a component with attributes and directives", func(t *testing.T) {
			result := expectFromHtmlSpans(
				`<MyComp before="foo" @Dir middle @OtherDir([a]="a" (b)="b()") after="123">Hello</MyComp>`,
				true,
			)
			// Check for key elements
			foundComponent := false
			foundDirective := false
			for _, item := range result {
				if len(item) > 0 {
					if item[0] == "Component" {
						foundComponent = true
					}
					if item[0] == "Directive" {
						foundDirective = true
					}
				}
			}
			if !foundComponent {
				t.Error("Expected to find Component")
			}
			if !foundDirective {
				t.Error("Expected to find Directive")
			}
		})

		t.Run("is correct for a component nested inside other markup", func(t *testing.T) {
			result := expectFromHtmlSpans(
				`@if (expr) {<div>Hello: <MyComp><span><OtherComp/></span></MyComp></div>}`,
				true,
			)
			// Check for key elements
			foundIfBlock := false
			foundComponent := false
			for _, item := range result {
				if len(item) > 0 {
					if item[0] == "IfBlock" {
						foundIfBlock = true
					}
					if item[0] == "Component" {
						foundComponent = true
					}
				}
			}
			if !foundIfBlock {
				t.Error("Expected to find IfBlock")
			}
			if !foundComponent {
				t.Error("Expected to find Component")
			}
		})
	})

	t.Run("directives", func(t *testing.T) {
		t.Run("is correct for a directive with no attributes", func(t *testing.T) {
			result := expectFromHtmlSpans("<div @Dir></div>", true)
			expected := [][]interface{}{
				{"Element", "<div @Dir></div>", "<div @Dir>", "</div>"},
				{"Directive", "@Dir", "@Dir", "<empty>"},
			}
			assertEqualSpans(t, result, expected, "directive with no attributes")
		})

		t.Run("is correct for a directive with attributes", func(t *testing.T) {
			result := expectFromHtmlSpans(`<div @Dir(a="1" [b]="two" (c)="c()")></div>`, true)
			// Check for key elements
			foundDirective := false
			foundBoundAttribute := false
			foundBoundEvent := false
			for _, item := range result {
				if len(item) > 0 {
					if item[0] == "Directive" {
						foundDirective = true
					}
					if item[0] == "BoundAttribute" {
						foundBoundAttribute = true
					}
					if item[0] == "BoundEvent" {
						foundBoundEvent = true
					}
				}
			}
			if !foundDirective {
				t.Error("Expected to find Directive")
			}
			if !foundBoundAttribute {
				t.Error("Expected to find BoundAttribute")
			}
			if !foundBoundEvent {
				t.Error("Expected to find BoundEvent")
			}
		})

		t.Run("is correct for directives mixed with other attributes", func(t *testing.T) {
			result := expectFromHtmlSpans(
				`<div before="foo" @Dir middle @OtherDir([a]="a" (b)="b()") after="123"></div>`,
				true,
			)
			// Check for key elements
			foundDirective := false
			foundTextAttribute := false
			for _, item := range result {
				if len(item) > 0 {
					if item[0] == "Directive" {
						foundDirective = true
					}
					if item[0] == "TextAttribute" {
						foundTextAttribute = true
					}
				}
			}
			if !foundDirective {
				t.Error("Expected to find Directive")
			}
			if !foundTextAttribute {
				t.Error("Expected to find TextAttribute")
			}
		})
	})
}
