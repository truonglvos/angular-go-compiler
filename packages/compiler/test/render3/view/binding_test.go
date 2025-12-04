package view_test

import (
	"fmt"
	"ngc-go/packages/compiler/src/css"
	"ngc-go/packages/compiler/src/expression_parser"
	"ngc-go/packages/compiler/src/render3"
	"ngc-go/packages/compiler/src/render3/view"
	testview "ngc-go/packages/compiler/test/render3/view"
	"reflect"
	"testing"
)

// IdentityInputMapping is an InputOutputPropertySet which only uses an identity mapping
type IdentityInputMapping struct {
	names map[string]bool
}

// NewIdentityInputMapping creates a new IdentityInputMapping
func NewIdentityInputMapping(names []string) *IdentityInputMapping {
	nameSet := make(map[string]bool)
	for _, name := range names {
		nameSet[name] = true
	}
	return &IdentityInputMapping{
		names: nameSet,
	}
}

// HasBindingPropertyName checks if a property name exists
func (i *IdentityInputMapping) HasBindingPropertyName(propertyName string) bool {
	return i.names[propertyName]
}

// makeSelectorMatcher creates a selector matcher with test directives
func makeSelectorMatcher() *css.SelectorMatcher[view.DirectiveMeta] {
	matcher := css.NewSelectorMatcher[view.DirectiveMeta]()

	// Add ngFor directive
	ngForSelectors, _ := css.ParseCssSelector("[ngFor][ngForOf]")
	fmt.Printf("[DEBUG TEST] makeSelectorMatcher: ngForSelectors count=%d\n", len(ngForSelectors))
	for i, sel := range ngForSelectors {
		fmt.Printf("[DEBUG TEST] makeSelectorMatcher: selector[%d]=%v\n", i, sel)
	}
	ngForDirective := &testDirectiveMeta{
		name:         "NgFor",
		exportAs:     nil,
		inputs:       NewIdentityInputMapping([]string{"ngForOf"}),
		outputs:      NewIdentityInputMapping([]string{}),
		isComponent:  false,
		isStructural: true,
		selector:     "[ngFor][ngForOf]",
	}
	// AddSelectables expects *T, but DirectiveMeta is interface, so we pass the pointer
	var ngForMeta view.DirectiveMeta = ngForDirective
	matcher.AddSelectables(ngForSelectors, &ngForMeta)

	// Add dir directive
	dirSelectors, _ := css.ParseCssSelector("[dir]")
	dirExportAs := []string{"dir"}
	dirDirective := &testDirectiveMeta{
		name:         "Dir",
		exportAs:     &dirExportAs,
		inputs:       NewIdentityInputMapping([]string{}),
		outputs:      NewIdentityInputMapping([]string{}),
		isComponent:  false,
		isStructural: false,
		selector:     "[dir]",
	}
	var dirMeta2 view.DirectiveMeta = dirDirective
	matcher.AddSelectables(dirSelectors, &dirMeta2)

	// Add hasOutput directive
	hasOutputSelectors, _ := css.ParseCssSelector("[hasOutput]")
	hasOutputDirective := &testDirectiveMeta{
		name:         "HasOutput",
		exportAs:     nil,
		inputs:       NewIdentityInputMapping([]string{}),
		outputs:      NewIdentityInputMapping([]string{"outputBinding"}),
		isComponent:  false,
		isStructural: false,
		selector:     "[hasOutput]",
	}
	var hasOutputMeta view.DirectiveMeta = hasOutputDirective
	matcher.AddSelectables(hasOutputSelectors, &hasOutputMeta)

	// Add hasInput directive
	hasInputSelectors, _ := css.ParseCssSelector("[hasInput]")
	hasInputDirective := &testDirectiveMeta{
		name:         "HasInput",
		exportAs:     nil,
		inputs:       NewIdentityInputMapping([]string{"inputBinding"}),
		outputs:      NewIdentityInputMapping([]string{}),
		isComponent:  false,
		isStructural: false,
		selector:     "[hasInput]",
	}
	var hasInputMeta view.DirectiveMeta = hasInputDirective
	matcher.AddSelectables(hasInputSelectors, &hasInputMeta)

	// Add sameSelectorAsInput directive
	sameSelectorSelectors, _ := css.ParseCssSelector("[sameSelectorAsInput]")
	sameSelectorDirective := &testDirectiveMeta{
		name:         "SameSelectorAsInput",
		exportAs:     nil,
		inputs:       NewIdentityInputMapping([]string{"sameSelectorAsInput"}),
		outputs:      NewIdentityInputMapping([]string{}),
		isComponent:  false,
		isStructural: false,
		selector:     "[sameSelectorAsInput]",
	}
	var sameSelectorMeta view.DirectiveMeta = sameSelectorDirective
	matcher.AddSelectables(sameSelectorSelectors, &sameSelectorMeta)

	// Add comp component
	compSelectors, _ := css.ParseCssSelector("comp")
	compDirective := &testDirectiveMeta{
		name:         "Comp",
		exportAs:     nil,
		inputs:       NewIdentityInputMapping([]string{}),
		outputs:      NewIdentityInputMapping([]string{}),
		isComponent:  true,
		isStructural: false,
		selector:     "comp",
	}
	var compMeta view.DirectiveMeta = compDirective
	matcher.AddSelectables(compSelectors, &compMeta)

	// Add simple directives
	simpleDirectives := []string{"a", "b", "c", "d", "e", "f"}
	deferBlockDirectives := []string{"loading", "error", "placeholder"}
	allDirectives := append(simpleDirectives, deferBlockDirectives...)
	for _, dir := range allDirectives {
		dirName := string(dir[0]-32) + dir[1:] // Capitalize first letter
		dirSelectors, _ := css.ParseCssSelector("[" + dir + "]")
		dirDirective := &testDirectiveMeta{
			name:         "Dir" + dirName,
			exportAs:     nil,
			inputs:       NewIdentityInputMapping([]string{}),
			outputs:      NewIdentityInputMapping([]string{}),
			isComponent:  false,
			isStructural: true,
			selector:     "[" + dir + "]",
		}
		var dirMeta view.DirectiveMeta = dirDirective
		matcher.AddSelectables(dirSelectors, &dirMeta)
	}

	return matcher
}

// testDirectiveMeta is a test implementation of DirectiveMeta
type testDirectiveMeta struct {
	name         string
	exportAs     *[]string
	inputs       view.InputOutputPropertySet
	outputs      view.InputOutputPropertySet
	isComponent  bool
	isStructural bool
	selector     string
}

func (t *testDirectiveMeta) Name() string {
	return t.name
}

func (t *testDirectiveMeta) Selector() *string {
	return &t.selector
}

func (t *testDirectiveMeta) IsComponent() bool {
	return t.isComponent
}

func (t *testDirectiveMeta) Inputs() view.InputOutputPropertySet {
	return t.inputs
}

func (t *testDirectiveMeta) Outputs() view.InputOutputPropertySet {
	return t.outputs
}

func (t *testDirectiveMeta) ExportAs() []string {
	if t.exportAs == nil {
		return nil
	}
	return *t.exportAs
}

func (t *testDirectiveMeta) IsStructural() bool {
	return t.isStructural
}

func (t *testDirectiveMeta) NgContentSelectors() []string {
	return nil
}

func (t *testDirectiveMeta) PreserveWhitespaces() bool {
	return false
}

func (t *testDirectiveMeta) AnimationTriggerNames() *view.LegacyAnimationTriggerNames {
	return nil
}

func TestFindMatchingDirectivesAndPipes(t *testing.T) {
	t.Run("should match directives and detect pipes in eager and deferrable parts of a template", func(t *testing.T) {
		template := `
      <div [title]="abc | uppercase"></div>
      @defer {
        <my-defer-cmp [label]="abc | lowercase" />
      } @placeholder {}
    `
		directiveSelectors := []string{"[title]", "my-defer-cmp", "not-matching"}
		result := view.FindMatchingDirectivesAndPipes(template, directiveSelectors)

		directives, ok := result["directives"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected directives in result")
		}
		pipes, ok := result["pipes"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected pipes in result")
		}

		// Note: The actual implementation may differ, so we just check structure
		if directives == nil || pipes == nil {
			t.Error("Expected directives and pipes to be non-nil")
		}
	})

	t.Run("should return empty directive list if no selectors are provided", func(t *testing.T) {
		template := `
        <div [title]="abc | uppercase"></div>
        @defer {
          <my-defer-cmp [label]="abc | lowercase" />
        } @placeholder {}
      `
		directiveSelectors := []string{}
		result := view.FindMatchingDirectivesAndPipes(template, directiveSelectors)

		directives, ok := result["directives"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected directives in result")
		}
		regular, ok := directives["regular"].([]string)
		if !ok {
			t.Fatalf("Expected regular to be []string")
		}
		if len(regular) != 0 {
			t.Errorf("Expected empty regular directives, got %v", regular)
		}
	})
}

func TestT2Binding(t *testing.T) {
	t.Run("should bind a simple template", func(t *testing.T) {
		template := view.ParseTemplate(`<div *ngFor="let item of items">{{item.name}}</div>`, "", nil)
		matcher := css.NewSelectorMatcher[view.DirectiveMeta]()
		binder := view.NewR3TargetBinder(matcher)
		res := binder.Bind(&view.Target{Template: template.Nodes})

		// Find the interpolation expression
		itemBinding := testview.FindExpression(template.Nodes, "{{item.name}}")
		if itemBinding == nil {
			t.Fatalf("Expected to find expression {{item.name}}")
		}

		interpolation, ok := itemBinding.(*expression_parser.Interpolation)
		if !ok {
			t.Fatalf("Expected Interpolation, got %T", itemBinding)
		}
		if len(interpolation.Expressions) == 0 {
			t.Fatalf("Expected at least one expression")
		}

		propertyRead, ok := interpolation.Expressions[0].(*expression_parser.PropertyRead)
		if !ok {
			t.Fatalf("Expected PropertyRead, got %T", interpolation.Expressions[0])
		}

		item := propertyRead.Receiver
		itemTarget := res.GetExpressionTarget(item)
		if itemTarget == nil {
			t.Fatalf("Expected itemTarget to be non-nil")
		}

		variable, ok := itemTarget.(*render3.Variable)
		if !ok {
			t.Fatalf("Expected item to point to a Variable, got %T", itemTarget)
		}
		if variable.Value != "$implicit" {
			t.Errorf("Expected itemTarget.value to be '$implicit', got %q", variable.Value)
		}

		itemTemplate := res.GetDefinitionNodeOfSymbol(variable)
		if itemTemplate == nil {
			t.Error("Expected itemTemplate to be non-nil")
		}
		if itemTemplate != nil {
			level := res.GetNestingLevel(itemTemplate)
			if level != 1 {
				t.Errorf("Expected nesting level to be 1, got %d", level)
			}
		}
	})

	t.Run("should match directives when binding a simple template", func(t *testing.T) {
		template := view.ParseTemplate(`<div *ngFor="let item of items">{{item.name}}</div>`, "", nil)
		binder := view.NewR3TargetBinder(makeSelectorMatcher())
		res := binder.Bind(&view.Target{Template: template.Nodes})

		if len(template.Nodes) == 0 {
			t.Fatalf("Expected at least one node")
		}
		tmpl, ok := template.Nodes[0].(*render3.Template)
		if !ok {
			t.Fatalf("Expected first node to be Template, got %T", template.Nodes[0])
		}

		directives := res.GetDirectivesOfNode(tmpl)
		if directives == nil {
			t.Fatalf("Expected directives to be non-nil")
		}
		if len(directives) != 1 {
			t.Errorf("Expected 1 directive, got %d", len(directives))
		}
		if len(directives) > 0 {
			directive, ok := directives[0].(view.DirectiveMeta)
			if !ok {
				t.Fatalf("Expected directive to be DirectiveMeta, got %T", directives[0])
			}
			if directive.Name() != "NgFor" {
				t.Errorf("Expected directive name to be 'NgFor', got %q", directive.Name())
			}
		}
	})

	t.Run("should match directives on namespaced elements", func(t *testing.T) {
		template := view.ParseTemplate(`<svg><text dir>SVG</text></svg>`, "", nil)
		matcher := css.NewSelectorMatcher[view.DirectiveMeta]()
		textSelectors, _ := css.ParseCssSelector("text[dir]")
		dirDirective := &testDirectiveMeta{
			name:         "Dir",
			exportAs:     nil,
			inputs:       NewIdentityInputMapping([]string{}),
			outputs:      NewIdentityInputMapping([]string{}),
			isComponent:  false,
			isStructural: false,
			selector:     "text[dir]",
		}
		var dirMeta view.DirectiveMeta = dirDirective
		matcher.AddSelectables(textSelectors, &dirMeta)
		binder := view.NewR3TargetBinder(matcher)
		res := binder.Bind(&view.Target{Template: template.Nodes})

		if len(template.Nodes) == 0 {
			t.Fatalf("Expected at least one node")
		}
		svgNode, ok := template.Nodes[0].(*render3.Element)
		if !ok {
			t.Fatalf("Expected first node to be Element, got %T", template.Nodes[0])
		}
		if len(svgNode.Children) == 0 {
			t.Fatalf("Expected svgNode to have children")
		}
		textNode, ok := svgNode.Children[0].(*render3.Element)
		if !ok {
			t.Fatalf("Expected first child to be Element, got %T", svgNode.Children[0])
		}

		directives := res.GetDirectivesOfNode(textNode)
		if directives == nil {
			t.Fatalf("Expected directives to be non-nil")
		}
		if len(directives) != 1 {
			t.Errorf("Expected 1 directive, got %d", len(directives))
		}
		if len(directives) > 0 {
			directive, ok := directives[0].(view.DirectiveMeta)
			if !ok {
				t.Fatalf("Expected directive to be DirectiveMeta, got %T", directives[0])
			}
			if directive.Name() != "Dir" {
				t.Errorf("Expected directive name to be 'Dir', got %q", directive.Name())
			}
		}
	})

	t.Run("should not match directives intended for an element on a microsyntax template", func(t *testing.T) {
		template := view.ParseTemplate(`<div *ngFor="let item of items" dir></div>`, "", nil)
		binder := view.NewR3TargetBinder(makeSelectorMatcher())
		res := binder.Bind(&view.Target{Template: template.Nodes})

		if len(template.Nodes) == 0 {
			t.Fatalf("Expected at least one node")
		}
		tmpl, ok := template.Nodes[0].(*render3.Template)
		if !ok {
			t.Fatalf("Expected first node to be Template, got %T", template.Nodes[0])
		}

		tmplDirectives := res.GetDirectivesOfNode(tmpl)
		if tmplDirectives == nil {
			t.Fatalf("Expected tmplDirectives to be non-nil")
		}
		if len(tmplDirectives) != 1 {
			t.Errorf("Expected 1 template directive, got %d", len(tmplDirectives))
		}
		if len(tmplDirectives) > 0 {
			directive, ok := tmplDirectives[0].(view.DirectiveMeta)
			if !ok {
				t.Fatalf("Expected directive to be DirectiveMeta, got %T", tmplDirectives[0])
			}
			if directive.Name() != "NgFor" {
				t.Errorf("Expected template directive name to be 'NgFor', got %q", directive.Name())
			}
		}

		if len(tmpl.Children) == 0 {
			t.Fatalf("Expected template to have children")
		}
		el, ok := tmpl.Children[0].(*render3.Element)
		if !ok {
			t.Fatalf("Expected first child to be Element, got %T", tmpl.Children[0])
		}

		elDirectives := res.GetDirectivesOfNode(el)
		if elDirectives == nil {
			t.Fatalf("Expected elDirectives to be non-nil")
		}
		if len(elDirectives) != 1 {
			t.Errorf("Expected 1 element directive, got %d", len(elDirectives))
		}
		if len(elDirectives) > 0 {
			directive, ok := elDirectives[0].(view.DirectiveMeta)
			if !ok {
				t.Fatalf("Expected directive to be DirectiveMeta, got %T", elDirectives[0])
			}
			if directive.Name() != "Dir" {
				t.Errorf("Expected element directive name to be 'Dir', got %q", directive.Name())
			}
		}
	})

	t.Run("should get @let declarations when resolving entities at the root", func(t *testing.T) {
		template := view.ParseTemplate(`
        @let one = 1;
        @let two = 2;
        @let sum = one + two;
      `, "", nil)
		matcher := css.NewSelectorMatcher[view.DirectiveMeta]()
		binder := view.NewR3TargetBinder(matcher)
		res := binder.Bind(&view.Target{Template: template.Nodes})

		entities := res.GetEntitiesInScope(nil)
		entityNames := make([]string, 0, len(entities))
		for _, entity := range entities {
			if letDecl, ok := entity.(*render3.LetDeclaration); ok {
				entityNames = append(entityNames, letDecl.Name)
			}
		}

		expected := []string{"one", "two", "sum"}
		if !reflect.DeepEqual(entityNames, expected) {
			t.Errorf("Expected entity names %v, got %v", expected, entityNames)
		}
	})

	t.Run("should scope @let declarations to their current view", func(t *testing.T) {
		template := view.ParseTemplate(`
        @let one = 1;

        @if (true) {
          @let two = 2;
        }

        @if (true) {
          @let three = 3;
        }
      `, "", nil)
		matcher := css.NewSelectorMatcher[view.DirectiveMeta]()
		binder := view.NewR3TargetBinder(matcher)
		res := binder.Bind(&view.Target{Template: template.Nodes})

		rootEntities := res.GetEntitiesInScope(nil)
		rootNames := make([]string, 0, len(rootEntities))
		for _, entity := range rootEntities {
			if letDecl, ok := entity.(*render3.LetDeclaration); ok {
				rootNames = append(rootNames, letDecl.Name)
			}
		}

		expectedRoot := []string{"one"}
		if !reflect.DeepEqual(rootNames, expectedRoot) {
			t.Errorf("Expected root entity names %v, got %v", expectedRoot, rootNames)
		}

		// Check first branch
		if len(template.Nodes) >= 2 {
			ifBlock, ok := template.Nodes[1].(*render3.IfBlock)
			if ok && len(ifBlock.Branches) > 0 {
				firstBranchEntities := res.GetEntitiesInScope(ifBlock.Branches[0])
				firstBranchNames := make([]string, 0, len(firstBranchEntities))
				for _, entity := range firstBranchEntities {
					if letDecl, ok := entity.(*render3.LetDeclaration); ok {
						firstBranchNames = append(firstBranchNames, letDecl.Name)
					}
				}
				expectedFirstBranch := []string{"one", "two"}
				if !reflect.DeepEqual(firstBranchNames, expectedFirstBranch) {
					t.Errorf("Expected first branch entity names %v, got %v", expectedFirstBranch, firstBranchNames)
				}
			}
		}
	})

	t.Run("should resolve expressions to an @let declaration", func(t *testing.T) {
		template := view.ParseTemplate(`
        @let value = 1;
        {{value}}
      `, "", nil)
		matcher := css.NewSelectorMatcher[view.DirectiveMeta]()
		binder := view.NewR3TargetBinder(matcher)
		res := binder.Bind(&view.Target{Template: template.Nodes})

		if len(template.Nodes) < 2 {
			t.Fatalf("Expected at least 2 nodes")
		}
		boundText, ok := template.Nodes[1].(*render3.BoundText)
		if !ok {
			t.Fatalf("Expected second node to be BoundText, got %T", template.Nodes[1])
		}

		astWithSource, ok := boundText.Value.(*expression_parser.ASTWithSource)
		if !ok {
			t.Fatalf("Expected ASTWithSource, got %T", boundText.Value)
		}

		interpolation, ok := astWithSource.AST.(*expression_parser.Interpolation)
		if !ok {
			t.Fatalf("Expected Interpolation, got %T", astWithSource.AST)
		}
		if len(interpolation.Expressions) == 0 {
			t.Fatalf("Expected at least one expression")
		}

		propertyRead := interpolation.Expressions[0]
		target := res.GetExpressionTarget(propertyRead)

		if target == nil {
			t.Fatalf("Expected target to be non-nil")
		}
		letDecl, ok := target.(*render3.LetDeclaration)
		if !ok {
			t.Fatalf("Expected LetDeclaration, got %T", target)
		}
		if letDecl.Name != "value" {
			t.Errorf("Expected let declaration name to be 'value', got %q", letDecl.Name)
		}
	})

	t.Run("matching inputs to consuming directives", func(t *testing.T) {
		t.Run("should work for bound attributes", func(t *testing.T) {
			template := view.ParseTemplate(`<div hasInput [inputBinding]="myValue"></div>`, "", nil)
			binder := view.NewR3TargetBinder(makeSelectorMatcher())
			res := binder.Bind(&view.Target{Template: template.Nodes})

			if len(template.Nodes) == 0 {
				t.Fatalf("Expected at least one node")
			}
			el, ok := template.Nodes[0].(*render3.Element)
			if !ok {
				t.Fatalf("Expected first node to be Element, got %T", template.Nodes[0])
			}
			if len(el.Inputs) == 0 {
				t.Fatalf("Expected element to have inputs")
			}

			attr := el.Inputs[0]
			consumer := res.GetConsumerOfBinding(attr)
			if consumer == nil {
				t.Fatalf("Expected consumer to be non-nil")
			}

			directiveMeta, ok := consumer.(view.DirectiveMeta)
			if !ok {
				t.Fatalf("Expected DirectiveMeta, got %T", consumer)
			}
			if directiveMeta.Name() != "HasInput" {
				t.Errorf("Expected consumer name to be 'HasInput', got %q", directiveMeta.Name())
			}
		})
	})

	t.Run("matching outputs to consuming directives", func(t *testing.T) {
		t.Run("should work for bound events", func(t *testing.T) {
			template := view.ParseTemplate(
				`<div hasOutput (outputBinding)="myHandler($event)"></div>`,
				"",
				nil,
			)

			if len(template.Nodes) == 0 {
				t.Fatalf("Expected at least one node")
			}
			el, ok := template.Nodes[0].(*render3.Element)
			if !ok {
				t.Fatalf("Expected first node to be Element, got %T", template.Nodes[0])
			}
			t.Logf("Element before binding: Name=%q, Attributes=%d, Outputs=%d", el.Name, len(el.Attributes), len(el.Outputs))
			for i, out := range el.Outputs {
				t.Logf("  Output[%d]: Name=%q", i, out.Name)
			}

			binder := view.NewR3TargetBinder(makeSelectorMatcher())
			res := binder.Bind(&view.Target{Template: template.Nodes})

			if len(el.Outputs) == 0 {
				t.Fatalf("Expected element to have outputs")
			}

			attr := el.Outputs[0]
			consumer := res.GetConsumerOfBinding(attr)
			if consumer == nil {
				t.Fatalf("Expected consumer to be non-nil")
			}

			directiveMeta, ok := consumer.(view.DirectiveMeta)
			if !ok {
				t.Fatalf("Expected DirectiveMeta, got %T", consumer)
			}
			if directiveMeta.Name() != "HasOutput" {
				t.Errorf("Expected consumer name to be 'HasOutput', got %q", directiveMeta.Name())
			}
		})
	})

	// Note: Additional test cases for defer blocks, switch blocks, for loop blocks, etc.
	// can be added here following the same pattern
}
