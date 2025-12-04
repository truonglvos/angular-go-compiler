package view

import (
	"fmt"
	"ngc-go/packages/compiler/src/expression_parser"
	"ngc-go/packages/compiler/src/i18n"
	"ngc-go/packages/compiler/src/ml_parser"
	"ngc-go/packages/compiler/src/render3"
	"ngc-go/packages/compiler/src/template_parser"
	"ngc-go/packages/compiler/src/util"
	"strings"
)

// Render3ParseOptions are options for parsing R3 templates
type Render3ParseOptions struct {
	CollectCommentNodes bool
	SelectorlessEnabled bool
}

// HtmlAstToRender3Ast converts HTML AST nodes to R3 AST nodes
func HtmlAstToRender3Ast(
	htmlNodes []ml_parser.Node,
	bindingParser *template_parser.BindingParser,
	options Render3ParseOptions,
) *Render3ParseResult {
	transformer := NewHtmlAstToIvyAst(bindingParser, options)
	ivyNodes := make([]render3.Node, 0)

	// Visit all nodes
	// Convert htmlNodes to []ml_parser.Node for context
	htmlNodesSlice := make([]ml_parser.Node, len(htmlNodes))
	for i, node := range htmlNodes {
		htmlNodesSlice[i] = node
	}

	fmt.Printf("[DEBUG] HtmlAstToRender3Ast: visiting %d html nodes\n", len(htmlNodes))
	for i, node := range htmlNodes {
		fmt.Printf("[DEBUG] HtmlAstToRender3Ast: visiting node[%d], type=%T\n", i, node)
		// Pass siblings array as context for blocks
		result := node.Visit(transformer, htmlNodesSlice)
		fmt.Printf("[DEBUG] HtmlAstToRender3Ast: node[%d] Visit returned: %v (type=%T)\n", i, result != nil, result)
		if result != nil {
			if r3Node, ok := result.(render3.Node); ok {
				fmt.Printf("[DEBUG] HtmlAstToRender3Ast: node[%d] is render3.Node, appending\n", i)
				ivyNodes = append(ivyNodes, r3Node)
			} else {
				fmt.Printf("[DEBUG] HtmlAstToRender3Ast: node[%d] is NOT render3.Node\n", i)
			}
		}
	}
	fmt.Printf("[DEBUG] HtmlAstToRender3Ast: total ivyNodes=%d\n", len(ivyNodes))

	allErrors := append(bindingParser.Errors, transformer.Errors...)

	result := &Render3ParseResult{
		Nodes:              ivyNodes,
		Errors:             allErrors,
		StyleUrls:          transformer.StyleUrls,
		Styles:             transformer.Styles,
		NgContentSelectors: transformer.NgContentSelectors,
	}

	if options.CollectCommentNodes {
		result.CommentNodes = transformer.CommentNodes
	}

	return result
}

// HtmlAstToIvyAst transforms HTML AST to Ivy AST
type HtmlAstToIvyAst struct {
	bindingParser      *template_parser.BindingParser
	options            Render3ParseOptions
	Errors             []*util.ParseError
	Styles             []string
	StyleUrls          []string
	NgContentSelectors []string
	CommentNodes       []*render3.Comment
	inI18nBlock        bool
	processedNodes     map[ml_parser.Node]bool // Track processed nodes to avoid duplicates
}

// NewHtmlAstToIvyAst creates a new HtmlAstToIvyAst transformer
func NewHtmlAstToIvyAst(bindingParser *template_parser.BindingParser, options Render3ParseOptions) *HtmlAstToIvyAst {
	return &HtmlAstToIvyAst{
		bindingParser:      bindingParser,
		options:            options,
		Errors:             []*util.ParseError{},
		Styles:             []string{},
		StyleUrls:          []string{},
		NgContentSelectors: []string{},
		CommentNodes:       []*render3.Comment{},
		processedNodes:     make(map[ml_parser.Node]bool),
	}
}

// convertParsedEventToBoundEvent converts a ParsedEvent to BoundEvent
// For animation events, it parses phase from event name (e.g., "enter" from "animate.enter")
func (t *HtmlAstToIvyAst) convertParsedEventToBoundEvent(pe *expression_parser.ParsedEvent) *render3.BoundEvent {
	var target *string
	var phase *string
	// If TargetOrPhase is set, it's either a target or a phase
	// For animation events, it's a phase; for regular events, it's a target
	if pe.Type == expression_parser.ParsedEventTypeAnimation {
		// For animation events, parse phase from event name (e.g., "enter" from "animate.enter")
		if pe.TargetOrPhase != nil {
			phase = pe.TargetOrPhase
		} else {
			// Parse phase from event name if it starts with "animate."
			if strings.HasPrefix(pe.Name, "animate.") {
				parts := strings.Split(pe.Name, ".")
				if len(parts) > 1 {
					phaseStr := parts[1]
					phase = &phaseStr
				}
			}
		}
	} else {
		target = pe.TargetOrPhase
	}
	return render3.NewBoundEvent(
		pe.Name,
		pe.Type,
		pe.Handler,
		target,
		phase,
		pe.SourceSpan,
		pe.HandlerSpan,
		pe.KeySpan,
	)
}

// VisitElement visits an element node
func (t *HtmlAstToIvyAst) VisitElement(element *ml_parser.Element, context interface{}) interface{} {
	fmt.Printf("[DEBUG] VisitElement: START, element.Name=%q\n", element.Name)
	// Preparse element to check if it's ng-content, script, style, etc.
	preparsedElement := template_parser.PreparseElement(element)

	// Handle script, style, stylesheet elements - return nil (skip them)
	if preparsedElement.Type == template_parser.PreparsedElementTypeScript {
		return nil
	} else if preparsedElement.Type == template_parser.PreparsedElementTypeStyle {
		// Extract style content
		if len(element.Children) == 1 {
			if textNode, ok := element.Children[0].(*ml_parser.Text); ok {
				t.Styles = append(t.Styles, textNode.Value)
			}
		}
		return nil
	} else if preparsedElement.Type == template_parser.PreparsedElementTypeStylesheet {
		if preparsedElement.HrefAttr != nil {
			t.StyleUrls = append(t.StyleUrls, *preparsedElement.HrefAttr)
		}
		return nil
	}

	children := make([]render3.Node, 0)
	for _, child := range element.Children {
		res := child.Visit(t, nil)
		if res != nil {
			if node, ok := res.(render3.Node); ok {
				children = append(children, node)
			}
		}
	}

	// Handle ng-content element
	if preparsedElement.Type == template_parser.PreparsedElementTypeNgContent {
		// Parse attributes for ng-content
		attrs := make([]*render3.TextAttribute, 0)
		for _, attr := range element.Attrs {
			attrs = append(attrs, render3.NewTextAttribute(
				attr.Name,
				attr.Value,
				attr.SourceSpan(),
				attr.KeySpan,
				attr.ValueSpan,
				attr.I18n(),
			))
		}

		// Filter whitespace-only text nodes from children
		// For ng-content, we need to filter out text nodes that only contain whitespace
		// This includes actual whitespace characters and escape sequences like \n, \t, etc.
		filteredChildren := make([]render3.Node, 0)
		for _, child := range children {
			shouldSkip := false
			if textNode, ok := child.(*render3.Text); ok {
				value := textNode.Value
				// Check if the value is empty or contains only whitespace
				// Handle both actual whitespace characters and escape sequences like \n, \t, \r
				// First, check if trimmed value is empty (handles actual newlines, tabs, etc.)
				trimmed := strings.TrimSpace(value)
				if trimmed == "" {
					shouldSkip = true
				} else {
					// Also check for escape sequences that represent whitespace
					// Replace common escape sequences with actual whitespace and check again
					normalized := strings.ReplaceAll(value, "\\n", "\n")
					normalized = strings.ReplaceAll(normalized, "\\t", "\t")
					normalized = strings.ReplaceAll(normalized, "\\r", "\r")
					normalized = strings.ReplaceAll(normalized, "\\f", "\f")
					normalized = strings.ReplaceAll(normalized, "\\v", "\v")
					normalizedTrimmed := strings.TrimSpace(normalized)
					if normalizedTrimmed == "" {
						shouldSkip = true
					}
				}
			} else if boundTextNode, ok := child.(*render3.BoundText); ok {
				// For BoundText, check if the expression source is whitespace-only
				if boundTextNode.Value != nil {
					if astWithSource, ok := boundTextNode.Value.(*expression_parser.ASTWithSource); ok && astWithSource.Source != nil {
						source := *astWithSource.Source
						trimmed := strings.TrimSpace(source)
						if trimmed == "" {
							shouldSkip = true
						}
					}
				}
			}
			if !shouldSkip {
				filteredChildren = append(filteredChildren, child)
			}
		}

		selector := preparsedElement.SelectAttr
		t.NgContentSelectors = append(t.NgContentSelectors, selector)

		return render3.NewContent(
			selector,
			attrs,
			filteredChildren,
			element.IsSelfClosing,
			element.SourceSpan(),
			element.StartSourceSpan,
			element.EndSourceSpan,
			element.I18n(),
		)
	}

	// Parse attributes into different categories
	attrs := make([]*render3.TextAttribute, 0)
	inputs := make([]*render3.BoundAttribute, 0)
	outputs := make([]*render3.BoundEvent, 0)
	references := make([]*render3.Reference, 0)

	// Track matchable attributes for directive matching
	matchableAttrs := []string{}

	// Parse template bindings (structural directives like *ngIf)
	parsedProperties := []*expression_parser.ParsedProperty{}
	// Separate arrays for inline template properties (from *ngFor parsing)
	inlineTemplateProperties := []*expression_parser.ParsedProperty{}
	inlineTemplateVariables := []*expression_parser.ParsedVariable{}
	hasTemplateAttrs := false

	for _, attr := range element.Attrs {
		name := strings.TrimSpace(attr.Name)
		value := attr.Value
		fmt.Printf("[DEBUG] VisitElement: processing attr, name=%q (original=%q), value=%q\n", name, attr.Name, value)

		// Skip let-* attributes (they are handled separately for ng-template)
		if strings.HasPrefix(name, "let-") {
			continue
		}

		// Check for reference (#ref or ref-)
		if len(name) > 0 && name[0] == '#' {
			refName := name[1:]
			refValue := value
			if refValue == "" {
				refValue = ""
			}
			// Create KeySpan that only includes the identifier (after "#")
			var keySpan *util.ParseSourceSpan
			if attr.KeySpan != nil {
				// KeySpan should start after "#" prefix
				keySpanStart := attr.KeySpan.Start.MoveBy(1) // Skip "#"
				keySpanEnd := keySpanStart.MoveBy(len(refName))
				detailsStr := refName
				keySpan = util.NewParseSourceSpan(keySpanStart, keySpanEnd, keySpanStart, &detailsStr)
			} else {
				keySpan = attr.KeySpan
			}
			references = append(references, render3.NewReference(
				refName,
				refValue,
				attr.SourceSpan(),
				keySpan,
				attr.ValueSpan,
			))
			matchableAttrs = append(matchableAttrs, name, value)
			continue
		}

		// Check for reference (ref-* or data-ref-*)
		normalizedNameForRef := normalizeAttributeName(name)
		if strings.HasPrefix(normalizedNameForRef, "ref-") {
			refName := normalizedNameForRef[4:] // Remove "ref-" prefix
			refValue := value
			if refValue == "" {
				refValue = ""
			}
			// Create KeySpan that only includes the identifier (after "ref-" or "data-ref-")
			var keySpan *util.ParseSourceSpan
			if attr.KeySpan != nil {
				normalizationAdjustment := len(name) - len(normalizedNameForRef)
				// KeySpan should start after "ref-" prefix
				keySpanStart := attr.KeySpan.Start.MoveBy(normalizationAdjustment + 4) // Skip "ref-"
				keySpanEnd := keySpanStart.MoveBy(len(refName))
				detailsStr := refName
				keySpan = util.NewParseSourceSpan(keySpanStart, keySpanEnd, keySpanStart, &detailsStr)
			} else {
				keySpan = attr.KeySpan
			}
			references = append(references, render3.NewReference(
				refName,
				refValue,
				attr.SourceSpan(),
				keySpan,
				attr.ValueSpan,
			))
			matchableAttrs = append(matchableAttrs, name, value)
			continue
		}

		// Check for template binding (*ngIf="value | async")
		if len(name) > 0 && name[0] == '*' {
			hasTemplateAttrs = true
			tplKey := name[1:] // Remove "*" prefix
			absoluteValueOffset := attr.SourceSpan().Start.Offset
			if attr.ValueSpan != nil {
				absoluteValueOffset = attr.ValueSpan.FullStart.Offset
			}
			// Parse into separate arrays for inline template
			t.bindingParser.ParseInlineTemplateBinding(
				tplKey,
				value,
				attr.SourceSpan(),
				absoluteValueOffset,
				&matchableAttrs,
				&inlineTemplateProperties,
				&inlineTemplateVariables,
				true, // isIvyAst - use binding-specific spans
			)
			continue
		}

		// Check for two-way binding ([(prop)]="value") - MUST check before [prop] binding
		if len(name) > 4 && name[0] == '[' && name[1] == '(' && name[len(name)-2] == ')' && name[len(name)-1] == ']' {
			// Extract identifier from [(prop)] -> (prop)
			identifier := name[2 : len(name)-2]
			// Extract propName from (prop) -> prop
			propName := identifier
			if len(identifier) > 2 && identifier[0] == '(' && identifier[len(identifier)-1] == ')' {
				propName = identifier[1 : len(identifier)-1]
			}
			absoluteOffset := attr.SourceSpan().Start.Offset
			if attr.ValueSpan != nil {
				absoluteOffset = attr.ValueSpan.FullStart.Offset
			}
			// Create adjusted KeySpan to exclude '[(' and ')]' delimiters, and also exclude '(' and ')' from identifier
			// KeySpan should only include propName (e.g., "prop"), not identifier (e.g., "(prop)")
			// In TypeScript, createKeySpan creates KeySpan with identifier = "(prop)", but we need to adjust it
			// to only include "prop" for the test case expectation
			var adjustedKeySpan *util.ParseSourceSpan
			if attr.KeySpan != nil {
				// attr.KeySpan covers the entire attribute name "[(prop)]"
				// We need to create a KeySpan that only covers "prop"
				// Start: after "[(" and "(" = attr.KeySpan.Start + 3
				// End: before ")" and ")]" = Start + len(propName)
				detailsStr := propName
				keySpanStart := attr.KeySpan.Start.MoveBy(2)     // Skip "[(" (2 chars)
				keySpanEnd := keySpanStart.MoveBy(len(propName)) // Only include propName
				adjustedKeySpan = util.NewParseSourceSpan(
					keySpanStart,
					keySpanEnd,
					keySpanStart,
					&detailsStr,
				)
				fmt.Printf("[DEBUG] VisitElement: banana box binding - attr.KeySpan=%q, adjustedKeySpan=%q (propName=%q)\n",
					attr.KeySpan.String(), adjustedKeySpan.String(), propName)
			} else {
				adjustedKeySpan = attr.KeySpan
			}
			// Pass identifier to ParsePropertyBinding to match TypeScript behavior
			// TypeScript passes identifier = "(prop)" to parsePropertyBinding
			t.bindingParser.ParsePropertyBinding(
				identifier, // Pass (prop) to match TypeScript behavior
				value,
				false, // isHost
				true,  // isPartOfAssignmentBinding
				attr.SourceSpan(),
				absoluteOffset,
				attr.ValueSpan,
				&matchableAttrs,
				&parsedProperties,
				adjustedKeySpan, // KeySpan already has detailsStr = propName
			)
			// Also parse as event binding for two-way binding
			// For two-way binding, event name is propName + "Change"
			eventName := propName + "Change"
			parsedEvents := []*expression_parser.ParsedEvent{}
			t.bindingParser.ParseEvent(
				eventName,
				value,
				true, // isAssignmentEvent
				attr.SourceSpan(),
				attr.ValueSpan,
				&matchableAttrs,
				&parsedEvents,
				adjustedKeySpan, // KeySpan is still for propName, not eventName
			)
			fmt.Printf("[DEBUG] VisitElement: banana box binding - parsedEvents count=%d, eventName=%q\n", len(parsedEvents), eventName)
			// Convert ParsedEvent to BoundEvent
			for _, pe := range parsedEvents {
				outputs = append(outputs, t.convertParsedEventToBoundEvent(pe))
			}
			fmt.Printf("[DEBUG] VisitElement: banana box binding - outputs count=%d after adding events\n", len(outputs))
			continue
		}

		// Check for property binding ([prop]="value")
		if len(name) > 2 && name[0] == '[' && name[len(name)-1] == ']' {
			propName := name[1 : len(name)-1]
			absoluteOffset := attr.SourceSpan().Start.Offset
			if attr.ValueSpan != nil {
				absoluteOffset = attr.ValueSpan.FullStart.Offset
			}
			// Create adjusted KeySpan to exclude '[' and ']' delimiters
			var adjustedKeySpan *util.ParseSourceSpan
			if attr.KeySpan != nil {
				// KeySpan should start after '[' and end before ']'
				detailsStr := propName
				adjustedKeySpan = util.NewParseSourceSpan(
					attr.KeySpan.Start.MoveBy(1),               // Skip '['
					attr.KeySpan.Start.MoveBy(1+len(propName)), // End before ']'
					attr.KeySpan.Start.MoveBy(1),
					&detailsStr,
				)
			} else {
				adjustedKeySpan = attr.KeySpan
			}
			beforeCount := len(parsedProperties)
			t.bindingParser.ParsePropertyBinding(
				propName,
				value,
				false, // isHost
				false, // isPartOfAssignmentBinding
				attr.SourceSpan(),
				absoluteOffset,
				attr.ValueSpan,
				&matchableAttrs,
				&parsedProperties,
				adjustedKeySpan,
			)
			afterCount := len(parsedProperties)
			fmt.Printf("[DEBUG] VisitElement: ParsePropertyBinding for [%s]=\"%s\", beforeCount=%d, afterCount=%d\n", propName, value, beforeCount, afterCount)
			continue
		}

		// Check for property binding (bind-prop="value" or data-bind-prop="value")
		if strings.HasPrefix(name, "bind-") || strings.HasPrefix(name, "data-bind-") {
			propName := name
			prefixLen := 0
			if strings.HasPrefix(name, "bind-") {
				propName = name[5:] // Remove "bind-" prefix
				prefixLen = 5
			} else if strings.HasPrefix(name, "data-bind-") {
				propName = name[10:] // Remove "data-bind-" prefix
				prefixLen = 10
			}
			fmt.Printf("[DEBUG] VisitElement: found bind-* attribute, name=%q, propName=%q, value=%q\n", name, propName, value)
			absoluteOffset := attr.SourceSpan().Start.Offset
			if attr.ValueSpan != nil {
				absoluteOffset = attr.ValueSpan.FullStart.Offset
			}
			// Adjust KeySpan to exclude the "bind-" or "data-bind-" prefix
			var adjustedKeySpan *util.ParseSourceSpan
			if attr.KeySpan != nil {
				adjustedKeySpan = util.NewParseSourceSpan(
					attr.KeySpan.Start.MoveBy(prefixLen),
					attr.KeySpan.End,
					attr.KeySpan.Start.MoveBy(prefixLen),
					attr.KeySpan.Details,
				)
			} else {
				adjustedKeySpan = attr.KeySpan
			}
			t.bindingParser.ParsePropertyBinding(
				propName,
				value,
				false, // isHost
				false, // isPartOfAssignmentBinding
				attr.SourceSpan(),
				absoluteOffset,
				attr.ValueSpan,
				&matchableAttrs,
				&parsedProperties,
				adjustedKeySpan,
			)
			continue
		}

		// Check for event binding ((event)="handler")
		if len(name) > 2 && name[0] == '(' && name[len(name)-1] == ')' {
			eventName := name[1 : len(name)-1]
			// Create adjusted KeySpan to exclude '(' and ')' delimiters
			var adjustedKeySpan *util.ParseSourceSpan
			if attr.KeySpan != nil {
				// KeySpan should start after '(' and end before ')'
				detailsStr := eventName
				adjustedKeySpan = util.NewParseSourceSpan(
					attr.KeySpan.Start.MoveBy(1),                // Skip '('
					attr.KeySpan.Start.MoveBy(1+len(eventName)), // End before ')'
					attr.KeySpan.Start.MoveBy(1),
					&detailsStr,
				)
			} else {
				adjustedKeySpan = attr.KeySpan
			}
			parsedEvents := []*expression_parser.ParsedEvent{}
			t.bindingParser.ParseEvent(
				eventName,
				value,
				false, // isAssignmentEvent
				attr.SourceSpan(),
				attr.ValueSpan,
				&matchableAttrs,
				&parsedEvents,
				adjustedKeySpan,
			)

			// Convert ParsedEvent to BoundEvent
			for _, pe := range parsedEvents {
				outputs = append(outputs, t.convertParsedEventToBoundEvent(pe))
			}
			continue
		}

		// Check for event binding (on-event="handler" or data-on-event="handler")
		if strings.HasPrefix(name, "on-") || strings.HasPrefix(name, "data-on-") {
			eventName := name
			prefixLen := 0
			if strings.HasPrefix(name, "on-") {
				eventName = name[3:] // Remove "on-" prefix
				prefixLen = 3
			} else if strings.HasPrefix(name, "data-on-") {
				eventName = name[8:] // Remove "data-on-" prefix
				prefixLen = 8
			}
			// Create adjusted KeySpan to exclude "on-" or "data-on-" prefix
			var adjustedKeySpan *util.ParseSourceSpan
			if attr.KeySpan != nil {
				detailsStr := eventName
				adjustedKeySpan = util.NewParseSourceSpan(
					attr.KeySpan.Start.MoveBy(prefixLen),
					attr.KeySpan.Start.MoveBy(prefixLen+len(eventName)),
					attr.KeySpan.Start.MoveBy(prefixLen),
					&detailsStr,
				)
			} else {
				adjustedKeySpan = attr.KeySpan
			}
			parsedEvents := []*expression_parser.ParsedEvent{}
			t.bindingParser.ParseEvent(
				eventName,
				value,
				false, // isAssignmentEvent
				attr.SourceSpan(),
				attr.ValueSpan,
				&matchableAttrs,
				&parsedEvents,
				adjustedKeySpan,
			)

			// Convert ParsedEvent to BoundEvent
			for _, pe := range parsedEvents {
				outputs = append(outputs, t.convertParsedEventToBoundEvent(pe))
			}
			continue
		}

		// Check for two-way binding (bindon-prop="value" or data-bindon-prop="value")
		if strings.HasPrefix(name, "bindon-") || strings.HasPrefix(name, "data-bindon-") {
			propName := name
			prefixLen := 0
			if strings.HasPrefix(name, "bindon-") {
				propName = name[7:] // Remove "bindon-" prefix
				prefixLen = 7
			} else if strings.HasPrefix(name, "data-bindon-") {
				propName = name[12:] // Remove "data-bindon-" prefix
				prefixLen = 12
			}
			absoluteOffset := attr.SourceSpan().Start.Offset
			if attr.ValueSpan != nil {
				absoluteOffset = attr.ValueSpan.FullStart.Offset
			}
			// Create adjusted KeySpan to exclude "bindon-" or "data-bindon-" prefix
			var adjustedKeySpan *util.ParseSourceSpan
			if attr.KeySpan != nil {
				// KeySpan should start after prefix
				detailsStr := propName
				adjustedKeySpan = util.NewParseSourceSpan(
					attr.KeySpan.Start.MoveBy(prefixLen),
					attr.KeySpan.Start.MoveBy(prefixLen+len(propName)),
					attr.KeySpan.Start.MoveBy(prefixLen),
					&detailsStr,
				)
			} else {
				adjustedKeySpan = attr.KeySpan
			}
			t.bindingParser.ParsePropertyBinding(
				propName,
				value,
				false, // isHost
				true,  // isPartOfAssignmentBinding
				attr.SourceSpan(),
				absoluteOffset,
				attr.ValueSpan,
				&matchableAttrs,
				&parsedProperties,
				adjustedKeySpan,
			)
			// Also parse as event binding for two-way binding
			parsedEvents := []*expression_parser.ParsedEvent{}
			t.bindingParser.ParseEvent(
				propName,
				value,
				true, // isAssignmentEvent
				attr.SourceSpan(),
				attr.ValueSpan,
				&matchableAttrs,
				&parsedEvents,
				adjustedKeySpan,
			)
			// Convert ParsedEvent to BoundEvent
			for _, pe := range parsedEvents {
				outputs = append(outputs, t.convertParsedEventToBoundEvent(pe))
			}
			continue
		}

		// Normalize attribute name (remove data- prefix)
		normalizedName := normalizeAttributeName(name)

		// Skip let-* variables for ng-template - they're handled separately
		if strings.ToLower(element.Name) == "ng-template" && strings.HasPrefix(normalizedName, "let-") {
			continue
		}

		// Create adjusted KeySpan to account for removed 'data-' prefix
		var adjustedKeySpan *util.ParseSourceSpan
		if attr.KeySpan != nil {
			normalizationAdjustment := len(name) - len(normalizedName)
			if normalizationAdjustment > 0 {
				// Adjust KeySpan to exclude 'data-' prefix
				detailsStr := normalizedName
				adjustedKeySpan = util.NewParseSourceSpan(
					attr.KeySpan.Start.MoveBy(normalizationAdjustment),
					attr.KeySpan.Start.MoveBy(normalizationAdjustment+len(normalizedName)),
					attr.KeySpan.Start.MoveBy(normalizationAdjustment),
					&detailsStr,
				)
			} else {
				adjustedKeySpan = attr.KeySpan
			}
		} else {
			adjustedKeySpan = attr.KeySpan
		}

		// Check for interpolation in attribute value
		hasInterpolation := t.bindingParser.ParsePropertyInterpolation(
			normalizedName,
			value,
			attr.SourceSpan(),
			attr.ValueSpan,
			&matchableAttrs,
			&parsedProperties,
			adjustedKeySpan,
			attr.ValueTokens,
		)

		if !hasInterpolation {
			// Regular attribute
			t.bindingParser.ParseLiteralAttr(
				normalizedName,
				&value,
				attr.SourceSpan(),
				attr.SourceSpan().Start.Offset,
				attr.ValueSpan,
				&matchableAttrs,
				&parsedProperties,
				adjustedKeySpan,
			)
		}
	}

	// Convert ParsedProperty to BoundAttribute or TextAttribute
	fmt.Printf("[DEBUG] VisitElement: converting parsedProperties (element attrs), count=%d\n", len(parsedProperties))
	fmt.Printf("[DEBUG] VisitElement: converting inlineTemplateProperties (template attrs), count=%d\n", len(inlineTemplateProperties))
	templateAttrs := []interface{}{} // BoundAttribute | TextAttribute

	// First, convert element properties (parsedProperties) to attrs/inputs
	for i, prop := range parsedProperties {
		fmt.Printf("[DEBUG] VisitElement: parsedProperties[%d]: name=%q, IsLiteral=%v\n", i, prop.Name, prop.IsLiteral)
		if prop.IsLiteral {
			// This is a text attribute, not a binding (element attribute)
			valueStr := ""
			if prop.Expression != nil && prop.Expression.Source != nil {
				valueStr = *prop.Expression.Source
			}
			textAttr := render3.NewTextAttribute(
				prop.Name,
				valueStr,
				prop.SourceSpan,
				prop.KeySpan,
				prop.ValueSpan,
				nil, // i18n
			)
			attrs = append(attrs, textAttr)
		} else {
			// This is a binding
			fmt.Printf("[DEBUG] VisitElement: creating BoundAttribute for prop=%q, Type=%d\n", prop.Name, prop.Type)
			boundProp := t.bindingParser.CreateBoundElementProperty(
				&element.Name,
				prop,
				false, // skipValidation
				true,  // mapPropertyName
			)
			fmt.Printf("[DEBUG] VisitElement: BoundAttribute created: name=%q, Type=%d, Value=%v\n", boundProp.Name, boundProp.Type, boundProp.Value)
			// For two-way binding [(prop)], adjust KeySpan to exclude '(' and ')' if needed
			keySpan := boundProp.KeySpan
			// Check if prop.Name has the form "(prop)" - this indicates a banana box binding
			// We adjust KeySpan regardless of prop.Type because the attribute name itself tells us it's a two-way binding
			if prop.Name != "" && len(prop.Name) > 2 && prop.Name[0] == '(' && prop.Name[len(prop.Name)-1] == ')' {
				// Extract propName from (prop) -> prop
				propName := prop.Name[1 : len(prop.Name)-1]
				if keySpan != nil {
					// KeySpan currently includes "(prop)", we need to adjust it to only include "prop"
					// The KeySpan should start after '(' and end before ')'
					detailsStr := propName
					keySpan = util.NewParseSourceSpan(
						keySpan.Start.MoveBy(1),               // Skip '('
						keySpan.Start.MoveBy(1+len(propName)), // End before ')'
						keySpan.Start.MoveBy(1),
						&detailsStr,
					)
					fmt.Printf("[DEBUG] VisitElement: adjusted KeySpan for banana box binding - prop.Name=%q, propName=%q, keySpan=%q\n", prop.Name, propName, keySpan.String())
				}
			}
			boundAttr := render3.NewBoundAttribute(
				boundProp.Name,
				boundProp.Type,
				boundProp.SecurityContext,
				boundProp.Value,
				boundProp.Unit,
				boundProp.SourceSpan,
				keySpan,
				boundProp.ValueSpan,
				nil, // i18n
			)
			inputs = append(inputs, boundAttr)
		}
	}

	// Then, convert inline template properties to templateAttrs
	for i, prop := range inlineTemplateProperties {
		fmt.Printf("[DEBUG] VisitElement: inlineTemplateProperties[%d]: name=%q, IsLiteral=%v\n", i, prop.Name, prop.IsLiteral)
		if prop.IsLiteral {
			valueStr := ""
			if prop.Expression != nil && prop.Expression.Source != nil {
				valueStr = *prop.Expression.Source
			}
			textAttr := render3.NewTextAttribute(
				prop.Name,
				valueStr,
				prop.SourceSpan,
				prop.KeySpan,
				prop.ValueSpan,
				nil, // i18n
			)
			templateAttrs = append(templateAttrs, textAttr)
		} else {
			boundProp := t.bindingParser.CreateBoundElementProperty(
				&element.Name,
				prop,
				false, // skipValidation
				true,  // mapPropertyName
			)
			boundAttr := render3.NewBoundAttribute(
				boundProp.Name,
				boundProp.Type,
				boundProp.SecurityContext,
				boundProp.Value,
				boundProp.Unit,
				boundProp.SourceSpan,
				boundProp.KeySpan,
				boundProp.ValueSpan,
				nil, // i18n
			)
			templateAttrs = append(templateAttrs, boundAttr)
		}
	}

	// Process directives (e.g., @Dir without attributes or @Dir(a="1" [b]="two"))
	directives := []*render3.Directive{}
	for _, directive := range element.Directives {
		// Check if this is an animation shorthand (@animation without parentheses)
		// When selectorlessEnabled=false, standalone @ directives are animation shorthands
		// When selectorlessEnabled=true, they are actual directives
		if !t.options.SelectorlessEnabled && len(directive.Attrs) == 0 {
			// This could be @animation shorthand - treat as a bound property
			sourceSpanStr := ""
			if directive.SourceSpan() != nil {
				sourceSpanStr = directive.SourceSpan().String()
			}

			if len(sourceSpanStr) > 1 && sourceSpanStr[0] == '@' {
				// This is @animation shorthand
				localParsedProps := []*expression_parser.ParsedProperty{}
				absoluteOffset := 0
				if directive.SourceSpan() != nil {
					absoluteOffset = directive.SourceSpan().Start.Offset
				}

				// Pass the StartSourceSpan as KeySpan - ParsePropertyBinding will adjust it
				// for the @ prefix via isLegacyAnimationLabel()
				keySpan := directive.StartSourceSpan
				if keySpan == nil {
					keySpan = directive.SourceSpan()
				}

				t.bindingParser.ParsePropertyBinding(
					sourceSpanStr, // Pass full "@animation" name
					"",            // Empty value
					false,         // isHost
					false,         // isPartOfAssignmentBinding
					directive.SourceSpan(),
					absoluteOffset,
					nil, // ValueSpan
					&matchableAttrs,
					&localParsedProps,
					keySpan, // Let ParsePropertyBinding adjust for @ prefix
				)

				for _, prop := range localParsedProps {
					boundProp := t.bindingParser.CreateBoundElementProperty(
						&element.Name,
						prop,
						false, // skipValidation
						true,  // mapPropertyName
					)
					if boundProp != nil {
						boundAttr := render3.NewBoundAttribute(
							boundProp.Name,
							boundProp.Type,
							boundProp.SecurityContext,
							boundProp.Value,
							boundProp.Unit,
							boundProp.SourceSpan,
							boundProp.KeySpan,
							boundProp.ValueSpan,
							nil, // i18n
						)

						if hasTemplateAttrs {
							templateAttrs = append(templateAttrs, boundAttr)
						} else {
							inputs = append(inputs, boundAttr)
						}
					}
				}
				continue
			}
		}

		fmt.Printf("[DEBUG] VisitElement: processing directive, Name=%q, StartSourceSpan=%v, SourceSpan=%v\n",
			directive.Name, func() string {
				if directive.StartSourceSpan == nil {
					return "<nil>"
				}
				return directive.StartSourceSpan.String()
			}(), func() string {
				if directive.SourceSpan() == nil {
					return "<nil>"
				}
				return directive.SourceSpan().String()
			}())

		// Parse directive attributes (similar to element attributes)
		directiveAttrs := []*render3.TextAttribute{}
		directiveInputs := []*render3.BoundAttribute{}
		directiveOutputs := []*render3.BoundEvent{}
		directiveReferences := []*render3.Reference{}
		directiveParsedProperties := []*expression_parser.ParsedProperty{}
		directiveParsedEvents := []*expression_parser.ParsedEvent{}
		directiveMatchableAttrs := []string{}

		for _, attr := range directive.Attrs {
			name := strings.TrimSpace(attr.Name)
			value := attr.Value

			// Check for reference (#ref or ref-)
			if len(name) > 0 && name[0] == '#' {
				refName := name[1:]
				refValue := value
				if refValue == "" {
					refValue = ""
				}
				directiveReferences = append(directiveReferences, render3.NewReference(
					refName,
					refValue,
					attr.SourceSpan(),
					attr.KeySpan,
					attr.ValueSpan,
				))
				directiveMatchableAttrs = append(directiveMatchableAttrs, name, value)
				continue
			}

			// Check for property binding ([prop]="value")
			if len(name) > 2 && name[0] == '[' && name[len(name)-1] == ']' {
				propName := name[1 : len(name)-1]
				absoluteOffset := attr.SourceSpan().Start.Offset
				if attr.ValueSpan != nil {
					absoluteOffset = attr.ValueSpan.FullStart.Offset
				}
				var adjustedKeySpan *util.ParseSourceSpan
				if attr.KeySpan != nil {
					detailsStr := propName
					adjustedKeySpan = util.NewParseSourceSpan(
						attr.KeySpan.Start.MoveBy(1),
						attr.KeySpan.Start.MoveBy(1+len(propName)),
						attr.KeySpan.Start.MoveBy(1),
						&detailsStr,
					)
				} else {
					adjustedKeySpan = attr.KeySpan
				}
				t.bindingParser.ParsePropertyBinding(
					propName,
					value,
					false, // isHost
					false, // isPartOfAssignmentBinding
					attr.SourceSpan(),
					absoluteOffset,
					attr.ValueSpan,
					&directiveMatchableAttrs,
					&directiveParsedProperties,
					adjustedKeySpan,
				)
				continue
			}

			// Check for event binding ((event)="handler")
			if len(name) > 2 && name[0] == '(' && name[len(name)-1] == ')' {
				eventName := name[1 : len(name)-1]
				var adjustedKeySpan *util.ParseSourceSpan
				if attr.KeySpan != nil {
					detailsStr := eventName
					adjustedKeySpan = util.NewParseSourceSpan(
						attr.KeySpan.Start.MoveBy(1),
						attr.KeySpan.Start.MoveBy(1+len(eventName)),
						attr.KeySpan.Start.MoveBy(1),
						&detailsStr,
					)
				} else {
					adjustedKeySpan = attr.KeySpan
				}
				t.bindingParser.ParseEvent(
					eventName,
					value,
					false, // isAssignmentEvent
					attr.SourceSpan(),
					attr.ValueSpan,
					&directiveMatchableAttrs,
					&directiveParsedEvents,
					adjustedKeySpan,
				)
				continue
			}

			// Normalize attribute name (remove data- prefix)
			normalizedName := normalizeAttributeName(name)
			var adjustedKeySpan *util.ParseSourceSpan
			if attr.KeySpan != nil {
				normalizationAdjustment := len(name) - len(normalizedName)
				if normalizationAdjustment > 0 {
					detailsStr := normalizedName
					adjustedKeySpan = util.NewParseSourceSpan(
						attr.KeySpan.Start.MoveBy(normalizationAdjustment),
						attr.KeySpan.Start.MoveBy(normalizationAdjustment+len(normalizedName)),
						attr.KeySpan.Start.MoveBy(normalizationAdjustment),
						&detailsStr,
					)
				} else {
					adjustedKeySpan = attr.KeySpan
				}
			} else {
				adjustedKeySpan = attr.KeySpan
			}

			// Check for interpolation in attribute value
			hasInterpolation := t.bindingParser.ParsePropertyInterpolation(
				normalizedName,
				value,
				attr.SourceSpan(),
				attr.ValueSpan,
				&directiveMatchableAttrs,
				&directiveParsedProperties,
				adjustedKeySpan,
				attr.ValueTokens,
			)

			if !hasInterpolation {
				// Regular attribute
				t.bindingParser.ParseLiteralAttr(
					normalizedName,
					&value,
					attr.SourceSpan(),
					attr.SourceSpan().Start.Offset,
					attr.ValueSpan,
					&directiveMatchableAttrs,
					&directiveParsedProperties,
					adjustedKeySpan,
				)
			}
		}

		// Convert ParsedProperty to BoundAttribute or TextAttribute
		for _, prop := range directiveParsedProperties {
			if prop.IsLiteral {
				valueStr := ""
				if prop.Expression != nil && prop.Expression.Source != nil {
					valueStr = *prop.Expression.Source
				}
				textAttr := render3.NewTextAttribute(
					prop.Name,
					valueStr,
					prop.SourceSpan,
					prop.KeySpan,
					prop.ValueSpan,
					nil, // i18n
				)
				directiveAttrs = append(directiveAttrs, textAttr)
			} else {
				boundProp := t.bindingParser.CreateBoundElementProperty(
					&element.Name,
					prop,
					false, // skipValidation
					true,  // mapPropertyName
				)
				if boundProp != nil {
					boundAttr := render3.NewBoundAttribute(
						boundProp.Name,
						boundProp.Type,
						boundProp.SecurityContext,
						boundProp.Value,
						boundProp.Unit,
						boundProp.SourceSpan,
						boundProp.KeySpan,
						boundProp.ValueSpan,
						nil, // i18n
					)
					directiveInputs = append(directiveInputs, boundAttr)
				}
			}
		}

		// Convert ParsedEvent to BoundEvent
		for _, pe := range directiveParsedEvents {
			directiveOutputs = append(directiveOutputs, t.convertParsedEventToBoundEvent(pe))
		}

		// Create Directive node
		directiveNode := render3.NewDirective(
			directive.Name,
			directiveAttrs,
			directiveInputs,
			directiveOutputs,
			directiveReferences,
			directive.SourceSpan(),
			directive.StartSourceSpan,
			directive.EndSourceSpan,
			nil, // i18n
		)
		directives = append(directives, directiveNode)
	}

	// Check if this is an ng-template element
	isNgTemplate := strings.ToLower(element.Name) == "ng-template"

	// If we have template attributes, create a Template node wrapping the element
	if hasTemplateAttrs {
		// Create the element node (without template attributes)
		elementNode := render3.NewElement(
			element.Name,
			attrs,
			inputs,
			outputs,
			directives, // directives
			children,
			references,
			element.IsSelfClosing,
			element.SourceSpan(),
			element.StartSourceSpan,
			element.EndSourceSpan,
			element.IsVoid,
			element.I18n(),
		)

		// Convert parsed variables to Variable nodes
		fmt.Printf("[DEBUG] VisitElement: converting inlineTemplateVariables, count=%d\n", len(inlineTemplateVariables))
		templateVariables := []*render3.Variable{}
		for i, parsedVar := range inlineTemplateVariables {
			fmt.Printf("[DEBUG] VisitElement: inlineTemplateVariables[%d]: name=%q, value=%q\n", i, parsedVar.Name, parsedVar.Value)
			variable := render3.NewVariable(
				parsedVar.Name,
				parsedVar.Value,
				parsedVar.SourceSpan,
				parsedVar.KeySpan,
				parsedVar.ValueSpan,
			)
			templateVariables = append(templateVariables, variable)
		}

		// Extract TextAttribute and BoundAttribute from templateAttrs for typed fields
		templateTextAttrs := []*render3.TextAttribute{}
		templateBoundAttrs := []*render3.BoundAttribute{}

		for _, attr := range templateAttrs {
			if textAttr, ok := attr.(*render3.TextAttribute); ok {
				templateTextAttrs = append(templateTextAttrs, textAttr)
			} else if boundAttr, ok := attr.(*render3.BoundAttribute); ok {
				templateBoundAttrs = append(templateBoundAttrs, boundAttr)
			}
		}

		// Create Template node with element as child
		tagName := element.Name
		return render3.NewTemplate(
			&tagName,
			templateTextAttrs,           // attributes - typed field
			templateBoundAttrs,          // inputs - typed field
			[]*render3.BoundEvent{},     // outputs
			[]*render3.Directive{},      // directives
			templateAttrs,               // templateAttrs - for tests that access directly
			[]render3.Node{elementNode}, // children
			references,
			templateVariables, // variables
			element.IsSelfClosing,
			element.SourceSpan(),
			element.StartSourceSpan,
			element.EndSourceSpan,
			element.I18n(),
		)
	}

	// If this is an ng-template element, convert it to a Template node
	if isNgTemplate {
		// Parse variables from let-* attributes
		variables := []*render3.Variable{}
		for _, attr := range element.Attrs {
			name := strings.TrimSpace(attr.Name)
			// Normalize attribute name (remove data- prefix)
			normalizedName := normalizeAttributeName(name)
			if strings.HasPrefix(normalizedName, "let-") {
				varName := normalizedName[4:] // Remove "let-" prefix
				varValue := attr.Value
				if varValue == "" {
					varValue = "$implicit"
				}
				// Calculate prefix length to adjust KeySpan
				prefixLen := len(name) - len(normalizedName) + 4 // data- prefix + "let-"
				// Create adjusted KeySpan to exclude "let-" or "data-let-" prefix
				var adjustedKeySpan *util.ParseSourceSpan
				if attr.KeySpan != nil {
					detailsStr := varName
					adjustedKeySpan = util.NewParseSourceSpan(
						attr.KeySpan.Start.MoveBy(prefixLen),
						attr.KeySpan.Start.MoveBy(prefixLen+len(varName)),
						attr.KeySpan.Start.MoveBy(prefixLen),
						&detailsStr,
					)
				} else {
					adjustedKeySpan = attr.KeySpan
				}
				variables = append(variables, render3.NewVariable(
					varName,
					varValue,
					attr.SourceSpan(),
					adjustedKeySpan,
					attr.ValueSpan,
				))
			}
		}
		// Convert attributes to template attributes
		templateAttrsFromElement := []interface{}{}
		for _, attr := range attrs {
			templateAttrsFromElement = append(templateAttrsFromElement, attr)
		}
		for _, input := range inputs {
			templateAttrsFromElement = append(templateAttrsFromElement, input)
		}
		var tagName *string = nil // ng-template doesn't have a tag name in Template node
		return render3.NewTemplate(
			tagName,
			attrs,                    // attributes - typed field
			inputs,                   // inputs - typed field
			outputs,                  // outputs
			directives,               // directives
			templateAttrsFromElement, // templateAttrs - for tests that access directly
			children,
			references,
			variables,
			element.IsSelfClosing,
			element.SourceSpan(),
			element.StartSourceSpan,
			element.EndSourceSpan,
			element.I18n(),
		)
	}

	result := render3.NewElement(
		element.Name,
		attrs,
		inputs,
		outputs,
		directives, // directives
		children,
		references,
		element.IsSelfClosing,
		element.SourceSpan(),
		element.StartSourceSpan,
		element.EndSourceSpan,
		element.IsVoid,
		element.I18n(),
	)
	fmt.Printf("[DEBUG] VisitElement: END, element.Name=%q, returning Element with %d inputs, %d outputs, %d directives\n", element.Name, len(inputs), len(outputs), len(directives))
	return result
}

// VisitAttribute visits an attribute node
func (t *HtmlAstToIvyAst) VisitAttribute(attribute *ml_parser.Attribute, context interface{}) interface{} {
	return render3.NewTextAttribute(
		attribute.Name,
		attribute.Value,
		attribute.SourceSpan(),
		attribute.KeySpan,
		attribute.ValueSpan,
		attribute.I18n(),
	)
}

// VisitText visits a text node
// Matches TypeScript: visitText(text: html.Text) which calls _visitTextWithInterpolation
func (t *HtmlAstToIvyAst) VisitText(text *ml_parser.Text, context interface{}) interface{} {
	// Check if text has interpolation tokens
	hasInterpolation := false
	if text.Tokens != nil && len(text.Tokens) > 0 {
		for _, token := range text.Tokens {
			if token.Type() == ml_parser.TokenTypeINTERPOLATION {
				hasInterpolation = true
				break
			}
		}
	}

	if hasInterpolation {
		// Parse interpolation and create BoundText
		fmt.Printf("[DEBUG] VisitText: parsing interpolation, Value=%q, SourceSpan=%v, FullStart.Offset=%d\n",
			text.Value, text.SourceSpan(), text.SourceSpan().FullStart.Offset)

		// Find the first INTERPOLATION token to get its SourceSpan for accurate offset calculation
		// When HTML entities are decoded, the Value string has different length than the original template,
		// so we need to use the INTERPOLATION token's SourceSpan to calculate correct absolute offsets
		var interpolationSourceSpan *util.ParseSourceSpan
		if text.Tokens != nil {
			for _, token := range text.Tokens {
				if token.Type() == ml_parser.TokenTypeINTERPOLATION {
					interpolationSourceSpan = token.SourceSpan()
					fmt.Printf("[DEBUG] VisitText: found INTERPOLATION token, SourceSpan=%v, Start.Offset=%d\n",
						interpolationSourceSpan, interpolationSourceSpan.Start.Offset)
					break
				}
			}
		}

		// Use interpolation token's SourceSpan if available, otherwise use text's SourceSpan
		// The interpolation token's SourceSpan has the correct absolute offset in the original template
		sourceSpanToUse := text.SourceSpan()
		if interpolationSourceSpan != nil {
			// Create a new SourceSpan that starts from the interpolation token
			// Use interpolationSourceSpan.Start as FullStart to ensure absoluteOffset is calculated correctly
			sourceSpanToUse = util.NewParseSourceSpan(
				interpolationSourceSpan.Start,
				text.SourceSpan().End,
				interpolationSourceSpan.Start, // Use Start as FullStart for correct absoluteOffset
				text.SourceSpan().Details,
			)
		}

		expr := t.bindingParser.ParseInterpolation(
			text.Value,
			sourceSpanToUse,
			text.Tokens,
		)
		if expr != nil {
			return render3.NewBoundText(
				expr,
				text.SourceSpan(),
				text.I18n(),
			)
		}
	}

	// No interpolation, create regular Text
	return render3.NewText(
		text.Value,
		text.SourceSpan(),
	)
}

// _visitTextWithInterpolation is a helper function to parse text with interpolation
// Matches TypeScript: _visitTextWithInterpolation
func (t *HtmlAstToIvyAst) _visitTextWithInterpolation(
	value string,
	sourceSpan *util.ParseSourceSpan,
	interpolatedTokens interface{}, // []ml_parser.InterpolatedAttributeToken | []ml_parser.InterpolatedTextToken | nil
	i18nMeta interface{}, // i18n.I18nMeta | nil
) render3.Node {
	// Replace ngsp (non-breaking space) characters
	valueNoNgsp := strings.ReplaceAll(value, "\u00a0", " ")

	expr := t.bindingParser.ParseInterpolation(valueNoNgsp, sourceSpan, interpolatedTokens)
	if expr != nil {
		return render3.NewBoundText(expr, sourceSpan, i18nMeta)
	}
	return render3.NewText(valueNoNgsp, sourceSpan)
}

// VisitComment visits a comment node
func (t *HtmlAstToIvyAst) VisitComment(comment *ml_parser.Comment, context interface{}) interface{} {
	if t.options.CollectCommentNodes {
		val := ""
		if comment.Value != nil {
			val = *comment.Value
		}
		t.CommentNodes = append(t.CommentNodes, render3.NewComment(val, comment.SourceSpan()))
	}
	return nil
}

// VisitExpansion visits an expansion node
func (t *HtmlAstToIvyAst) VisitExpansion(expansion *ml_parser.Expansion, context interface{}) interface{} {
	if expansion.I18n() == nil {
		// do not generate Icu in case it was created
		// outside of i18n block in a template
		return nil
	}

	// Check if i18n is a Message
	message, ok := expansion.I18n().(*i18n.Message)
	if !ok {
		// Not a Message, return nil
		return nil
	}

	vars := make(map[string]*render3.BoundText)
	placeholders := make(map[string]render3.Node)

	// Extract VARs from ICUs - we process them separately while
	// assembling resulting message via goog.getMsg function, since
	// we need to pass them to top-level goog.getMsg call
	const I18nICUVarPrefix = "VAR_"
	for key, value := range message.Placeholders {
		if strings.HasPrefix(key, I18nICUVarPrefix) {
			// Currently when the `plural` or `select` keywords in an ICU contain trailing spaces (e.g.
			// `{count, select , ...}`), these spaces are also included into the key names in ICU vars
			// (e.g. "VAR_SELECT "). These trailing spaces are not desirable, since they will later be
			// converted into `_` symbols while normalizing placeholder names, which might lead to
			// mismatches at runtime (i.e. placeholder will not be replaced with the correct value).
			formattedKey := strings.TrimSpace(key)

			ast := t.bindingParser.ParseInterpolationExpression(value.Text, value.SourceSpan)
			if ast != nil {
				vars[formattedKey] = render3.NewBoundText(ast, value.SourceSpan, nil)
			}
		} else {
			// Parse placeholders with interpolation
			placeholderNode := t._visitTextWithInterpolation(value.Text, value.SourceSpan, nil, nil)
			if placeholderNode != nil {
				placeholders[key] = placeholderNode
			}
		}
	}

	return render3.NewIcu(vars, placeholders, expansion.SourceSpan(), expansion.I18n())
}

// VisitExpansionCase visits an expansion case
func (t *HtmlAstToIvyAst) VisitExpansionCase(expansionCase *ml_parser.ExpansionCase, context interface{}) interface{} {
	return nil
}

// VisitBlock visits a block node
func (t *HtmlAstToIvyAst) VisitBlock(block *ml_parser.Block, context interface{}) interface{} {
	// Context should be a slice of siblings
	siblings, ok := context.([]ml_parser.Node)
	if !ok {
		return nil
	}

	// Find index of this block in siblings
	index := -1
	for i, node := range siblings {
		if node == block {
			index = i
			break
		}
	}
	if index == -1 {
		return nil
	}

	// Check if this block has already been processed (as part of a connected block)
	if t.processedNodes[block] {
		return nil
	}

	var result render3.Node
	var errors []*util.ParseError

	switch block.Name {
	case "let":
		// @let declarations
		if len(block.Parameters) == 0 {
			errors = []*util.ParseError{
				util.NewParseError(block.SourceSpan(), "@let declaration must have a name and value"),
			}
			result = render3.NewUnknownBlock(block.Name, block.SourceSpan(), block.NameSpan)
		} else {
			// Parse @let name = value;
			// Parameters contain the full expression: "name = value"
			paramExpr := block.Parameters[0].Expression
			// Split by '=' to get name and value
			parts := strings.SplitN(paramExpr, "=", 2)
			if len(parts) != 2 {
				errors = []*util.ParseError{
					util.NewParseError(block.Parameters[0].SourceSpan(), "@let declaration must be in format: @let name = value;"),
				}
				result = render3.NewUnknownBlock(block.Name, block.SourceSpan(), block.NameSpan)
			} else {
				name := strings.TrimSpace(parts[0])
				valueExpr := strings.TrimSpace(parts[1])
				// Remove trailing semicolon if present
				valueExpr = strings.TrimSuffix(valueExpr, ";")

				// Parse the value expression
				paramSourceSpan := block.Parameters[0].SourceSpan()
				valueOffset := paramSourceSpan.Start.Offset
				// Adjust offset to point to the value part
				valueOffset += strings.Index(paramExpr, "=") + 1
				valueOffset += len(parts[1]) - len(strings.TrimLeft(parts[1], " \t"))

				parsedValue := t.bindingParser.ParseBinding(
					valueExpr,
					false, // isHostBinding
					paramSourceSpan,
					valueOffset,
				)

				// Create name span and value span
				nameStart := block.SourceSpan().Start.MoveBy(strings.Index(block.SourceSpan().String(), name))
				nameSpan := util.NewParseSourceSpan(nameStart, nameStart.MoveBy(len(name)), nameStart, nil)
				valueSpan := paramSourceSpan

				result = render3.NewLetDeclaration(name, parsedValue.AST, block.SourceSpan(), nameSpan, valueSpan)
				errors = parsedValue.Errors
			}
		}

	case "if":
		connectedBlocks := t.findConnectedBlocks(index, siblings, render3.IsConnectedIfLoopBlock)
		createResult := render3.CreateIfBlock(block, connectedBlocks, t, t.bindingParser)
		result = createResult.Node
		errors = createResult.Errors
		// Mark connected blocks as processed
		for _, connectedBlock := range connectedBlocks {
			t.processedNodes[connectedBlock] = true
		}

	case "for":
		connectedBlocks := t.findConnectedBlocks(index, siblings, render3.IsConnectedForLoopBlock)
		createResult := render3.CreateForLoop(block, connectedBlocks, t, t.bindingParser)
		result = createResult.Node
		errors = createResult.Errors
		// Mark connected blocks as processed
		for _, connectedBlock := range connectedBlocks {
			t.processedNodes[connectedBlock] = true
		}

	case "switch":
		createResult := render3.CreateSwitchBlock(block, t, t.bindingParser)
		result = createResult.Node
		errors = createResult.Errors

	case "defer":
		connectedBlocks := t.findConnectedBlocks(index, siblings, render3.IsConnectedDeferLoopBlock)
		createResult := render3.CreateDeferredBlock(block, connectedBlocks, t, t.bindingParser)
		result = createResult.Node
		errors = createResult.Errors
		// Mark connected blocks as processed
		for _, connectedBlock := range connectedBlocks {
			t.processedNodes[connectedBlock] = true
		}

	default:
		// Unknown block
		var errorMessage string
		if render3.IsConnectedDeferLoopBlock(block.Name) {
			errorMessage = fmt.Sprintf("@%s block can only be used after an @defer block.", block.Name)
			t.processedNodes[block] = true
		} else if render3.IsConnectedForLoopBlock(block.Name) {
			errorMessage = fmt.Sprintf("@%s block can only be used after an @for block.", block.Name)
			t.processedNodes[block] = true
		} else if render3.IsConnectedIfLoopBlock(block.Name) {
			errorMessage = fmt.Sprintf("@%s block can only be used after an @if or @else if block.", block.Name)
			t.processedNodes[block] = true
		} else {
			errorMessage = fmt.Sprintf("Unrecognized block @%s.", block.Name)
		}
		result = render3.NewUnknownBlock(block.Name, block.SourceSpan(), block.NameSpan)
		errors = []*util.ParseError{
			util.NewParseError(block.SourceSpan(), errorMessage),
		}
	}

	// Add errors
	t.Errors = append(t.Errors, errors...)

	return result
}

// findConnectedBlocks finds connected blocks following a primary block
func (t *HtmlAstToIvyAst) findConnectedBlocks(
	primaryBlockIndex int,
	siblings []ml_parser.Node,
	predicate func(string) bool,
) []*ml_parser.Block {
	relatedBlocks := []*ml_parser.Block{}

	for i := primaryBlockIndex + 1; i < len(siblings); i++ {
		node := siblings[i]

		// Skip over comments
		if _, ok := node.(*ml_parser.Comment); ok {
			continue
		}

		// Ignore empty text nodes between blocks
		if textNode, ok := node.(*ml_parser.Text); ok {
			if strings.TrimSpace(textNode.Value) == "" {
				// Mark as processed so it's not generated between connected nodes
				t.processedNodes[node] = true
				continue
			}
		}

		// Check if this is a connected block
		if block, ok := node.(*ml_parser.Block); ok {
			if predicate(block.Name) {
				relatedBlocks = append(relatedBlocks, block)
			} else {
				// Stop at first non-connected block
				break
			}
		} else {
			// Stop at first non-block node
			break
		}
	}

	return relatedBlocks
}

// VisitBlockParameter visits a block parameter
func (t *HtmlAstToIvyAst) VisitBlockParameter(parameter *ml_parser.BlockParameter, context interface{}) interface{} {
	return nil
}

// normalizeAttributeName removes the 'data-' prefix from attribute names
// This matches TypeScript's normalizeAttributeName function
func normalizeAttributeName(attrName string) string {
	if strings.HasPrefix(strings.ToLower(attrName), "data-") {
		return attrName[5:]
	}
	return attrName
}

// VisitLetDeclaration visits a let declaration
func (t *HtmlAstToIvyAst) VisitLetDeclaration(decl *ml_parser.LetDeclaration, context interface{}) interface{} {
	value := t.bindingParser.ParseBinding(
		decl.Value,
		false,
		decl.ValueSpan,
		decl.ValueSpan.Start.Offset,
	)

	// Add parsing errors
	t.Errors = append(t.Errors, value.Errors...)

	if len(value.Errors) == 0 {
		// Check if value is empty expression
		if _, isEmpty := value.AST.(*expression_parser.EmptyExpr); isEmpty {
			t.Errors = append(t.Errors, util.NewParseError(
				decl.ValueSpan,
				"@let declaration value cannot be empty",
			))
		}
	}

	return render3.NewLetDeclaration(
		decl.Name,
		value.AST,
		decl.SourceSpan(),
		decl.NameSpan,
		decl.ValueSpan,
	)
}

// VisitComponent visits a component node
func (t *HtmlAstToIvyAst) VisitComponent(component *ml_parser.Component, context interface{}) interface{} {
	fmt.Printf("[DEBUG] VisitComponent: START, component.ComponentName=%q\n", component.ComponentName)

	// Visit children
	children := make([]render3.Node, 0)
	for _, child := range component.Children {
		res := child.Visit(t, nil)
		if res != nil {
			if node, ok := res.(render3.Node); ok {
				children = append(children, node)
			}
		}
	}

	// Parse attributes into different categories
	attrs := make([]*render3.TextAttribute, 0)
	inputs := make([]*render3.BoundAttribute, 0)
	outputs := make([]*render3.BoundEvent, 0)
	references := make([]*render3.Reference, 0)

	// Track matchable attributes for directive matching
	matchableAttrs := []string{}

	// Parse properties
	parsedProperties := []*expression_parser.ParsedProperty{}

	// Use tagName for element selector (or componentName if tagName is nil)
	elementSelector := component.ComponentName
	if component.TagName != nil {
		elementSelector = *component.TagName
	}

	for _, attr := range component.Attrs {
		name := strings.TrimSpace(attr.Name)
		value := attr.Value

		// Check for reference (#ref or ref-)
		if len(name) > 0 && name[0] == '#' {
			refName := name[1:]
			refValue := value
			if refValue == "" {
				refValue = ""
			}
			references = append(references, render3.NewReference(
				refName,
				refValue,
				attr.SourceSpan(),
				attr.KeySpan,
				attr.ValueSpan,
			))
			matchableAttrs = append(matchableAttrs, name, value)
			continue
		}

		// Check for property binding ([prop]="value")
		if len(name) > 2 && name[0] == '[' && name[len(name)-1] == ']' {
			propName := name[1 : len(name)-1]
			absoluteOffset := attr.SourceSpan().Start.Offset
			if attr.ValueSpan != nil {
				absoluteOffset = attr.ValueSpan.FullStart.Offset
			}
			var adjustedKeySpan *util.ParseSourceSpan
			if attr.KeySpan != nil {
				detailsStr := propName
				adjustedKeySpan = util.NewParseSourceSpan(
					attr.KeySpan.Start.MoveBy(1),
					attr.KeySpan.Start.MoveBy(1+len(propName)),
					attr.KeySpan.Start.MoveBy(1),
					&detailsStr,
				)
			} else {
				adjustedKeySpan = attr.KeySpan
			}
			t.bindingParser.ParsePropertyBinding(
				propName,
				value,
				false, // isHost
				false, // isPartOfAssignmentBinding
				attr.SourceSpan(),
				absoluteOffset,
				attr.ValueSpan,
				&matchableAttrs,
				&parsedProperties,
				adjustedKeySpan,
			)
			continue
		}

		// Check for property binding (bind-prop="value" or data-bind-prop="value")
		if strings.HasPrefix(name, "bind-") || strings.HasPrefix(name, "data-bind-") {
			propName := name
			prefixLen := 0
			if strings.HasPrefix(name, "bind-") {
				propName = name[5:]
				prefixLen = 5
			} else if strings.HasPrefix(name, "data-bind-") {
				propName = name[10:]
				prefixLen = 10
			}
			absoluteOffset := attr.SourceSpan().Start.Offset
			if attr.ValueSpan != nil {
				absoluteOffset = attr.ValueSpan.FullStart.Offset
			}
			var adjustedKeySpan *util.ParseSourceSpan
			if attr.KeySpan != nil {
				adjustedKeySpan = util.NewParseSourceSpan(
					attr.KeySpan.Start.MoveBy(prefixLen),
					attr.KeySpan.End,
					attr.KeySpan.Start.MoveBy(prefixLen),
					attr.KeySpan.Details,
				)
			} else {
				adjustedKeySpan = attr.KeySpan
			}
			t.bindingParser.ParsePropertyBinding(
				propName,
				value,
				false, // isHost
				false, // isPartOfAssignmentBinding
				attr.SourceSpan(),
				absoluteOffset,
				attr.ValueSpan,
				&matchableAttrs,
				&parsedProperties,
				adjustedKeySpan,
			)
			continue
		}

		// Check for animation shorthand (@animation)
		if len(name) > 1 && name[0] == '@' {
			animationName := name[1:] // Remove "@" prefix
			absoluteOffset := attr.SourceSpan().Start.Offset
			if attr.ValueSpan != nil {
				absoluteOffset = attr.ValueSpan.FullStart.Offset
			}
			// Create adjusted KeySpan to exclude '@' prefix
			var adjustedKeySpan *util.ParseSourceSpan
			if attr.KeySpan != nil {
				detailsStr := animationName
				adjustedKeySpan = util.NewParseSourceSpan(
					attr.KeySpan.Start.MoveBy(1),                    // Skip '@'
					attr.KeySpan.Start.MoveBy(1+len(animationName)), // End after animation name
					attr.KeySpan.Start.MoveBy(1),
					&detailsStr,
				)
			} else {
				adjustedKeySpan = attr.KeySpan
			}
			t.bindingParser.ParsePropertyBinding(
				name, // Pass full "@animation" name to binding parser
				value,
				false, // isHost
				false, // isPartOfAssignmentBinding
				attr.SourceSpan(),
				absoluteOffset,
				attr.ValueSpan,
				&matchableAttrs,
				&parsedProperties,
				adjustedKeySpan,
			)
			continue
		}

		// Check for event binding ((event)="handler")
		if len(name) > 2 && name[0] == '(' && name[len(name)-1] == ')' {
			eventName := name[1 : len(name)-1]
			var adjustedKeySpan *util.ParseSourceSpan
			if attr.KeySpan != nil {
				detailsStr := eventName
				adjustedKeySpan = util.NewParseSourceSpan(
					attr.KeySpan.Start.MoveBy(1),
					attr.KeySpan.Start.MoveBy(1+len(eventName)),
					attr.KeySpan.Start.MoveBy(1),
					&detailsStr,
				)
			} else {
				adjustedKeySpan = attr.KeySpan
			}
			parsedEvents := []*expression_parser.ParsedEvent{}
			t.bindingParser.ParseEvent(
				eventName,
				value,
				false, // isAssignmentEvent
				attr.SourceSpan(),
				attr.ValueSpan,
				&matchableAttrs,
				&parsedEvents,
				adjustedKeySpan,
			)
			for _, pe := range parsedEvents {
				outputs = append(outputs, t.convertParsedEventToBoundEvent(pe))
			}
			continue
		}

		// Check for event binding (on-event="handler" or data-on-event="handler")
		if strings.HasPrefix(name, "on-") || strings.HasPrefix(name, "data-on-") {
			eventName := name
			prefixLen := 0
			if strings.HasPrefix(name, "on-") {
				eventName = name[3:]
				prefixLen = 3
			} else if strings.HasPrefix(name, "data-on-") {
				eventName = name[8:]
				prefixLen = 8
			}
			// Create adjusted KeySpan to exclude "on-" or "data-on-" prefix
			var adjustedKeySpan *util.ParseSourceSpan
			if attr.KeySpan != nil {
				detailsStr := eventName
				adjustedKeySpan = util.NewParseSourceSpan(
					attr.KeySpan.Start.MoveBy(prefixLen),
					attr.KeySpan.Start.MoveBy(prefixLen+len(eventName)),
					attr.KeySpan.Start.MoveBy(prefixLen),
					&detailsStr,
				)
			} else {
				adjustedKeySpan = attr.KeySpan
			}
			parsedEvents := []*expression_parser.ParsedEvent{}
			t.bindingParser.ParseEvent(
				eventName,
				value,
				false, // isAssignmentEvent
				attr.SourceSpan(),
				attr.ValueSpan,
				&matchableAttrs,
				&parsedEvents,
				adjustedKeySpan,
			)
			for _, pe := range parsedEvents {
				outputs = append(outputs, t.convertParsedEventToBoundEvent(pe))
			}
			continue
		}

		// Check for two-way binding ([(prop)]="value")
		if len(name) > 4 && name[0] == '[' && name[1] == '(' && name[len(name)-2] == ')' && name[len(name)-1] == ']' {
			propName := name[2 : len(name)-2]
			absoluteOffset := attr.SourceSpan().Start.Offset
			if attr.ValueSpan != nil {
				absoluteOffset = attr.ValueSpan.FullStart.Offset
			}
			var adjustedKeySpan *util.ParseSourceSpan
			if attr.KeySpan != nil {
				detailsStr := propName
				adjustedKeySpan = util.NewParseSourceSpan(
					attr.KeySpan.Start.MoveBy(2),
					attr.KeySpan.Start.MoveBy(2+len(propName)),
					attr.KeySpan.Start.MoveBy(2),
					&detailsStr,
				)
			} else {
				adjustedKeySpan = attr.KeySpan
			}
			t.bindingParser.ParsePropertyBinding(
				propName,
				value,
				false, // isHost
				true,  // isPartOfAssignmentBinding
				attr.SourceSpan(),
				absoluteOffset,
				attr.ValueSpan,
				&matchableAttrs,
				&parsedProperties,
				adjustedKeySpan,
			)
			parsedEvents := []*expression_parser.ParsedEvent{}
			t.bindingParser.ParseEvent(
				propName,
				value,
				true, // isAssignmentEvent
				attr.SourceSpan(),
				attr.ValueSpan,
				&matchableAttrs,
				&parsedEvents,
				adjustedKeySpan,
			)
			for _, pe := range parsedEvents {
				outputs = append(outputs, t.convertParsedEventToBoundEvent(pe))
			}
			continue
		}

		// Check for two-way binding (bindon-prop="value" or data-bindon-prop="value")
		if strings.HasPrefix(name, "bindon-") || strings.HasPrefix(name, "data-bindon-") {
			propName := name
			prefixLen := 0
			if strings.HasPrefix(name, "bindon-") {
				propName = name[7:]
				prefixLen = 7
			} else if strings.HasPrefix(name, "data-bindon-") {
				propName = name[12:]
				prefixLen = 12
			}
			absoluteOffset := attr.SourceSpan().Start.Offset
			if attr.ValueSpan != nil {
				absoluteOffset = attr.ValueSpan.FullStart.Offset
			}
			// Create adjusted KeySpan to exclude "bindon-" or "data-bindon-" prefix
			var adjustedKeySpan *util.ParseSourceSpan
			if attr.KeySpan != nil {
				// KeySpan should start after prefix
				detailsStr := propName
				adjustedKeySpan = util.NewParseSourceSpan(
					attr.KeySpan.Start.MoveBy(prefixLen),
					attr.KeySpan.Start.MoveBy(prefixLen+len(propName)),
					attr.KeySpan.Start.MoveBy(prefixLen),
					&detailsStr,
				)
			} else {
				adjustedKeySpan = attr.KeySpan
			}
			t.bindingParser.ParsePropertyBinding(
				propName,
				value,
				false, // isHost
				true,  // isPartOfAssignmentBinding
				attr.SourceSpan(),
				absoluteOffset,
				attr.ValueSpan,
				&matchableAttrs,
				&parsedProperties,
				adjustedKeySpan,
			)
			parsedEvents := []*expression_parser.ParsedEvent{}
			t.bindingParser.ParseEvent(
				propName,
				value,
				true, // isAssignmentEvent
				attr.SourceSpan(),
				attr.ValueSpan,
				&matchableAttrs,
				&parsedEvents,
				adjustedKeySpan,
			)
			for _, pe := range parsedEvents {
				outputs = append(outputs, t.convertParsedEventToBoundEvent(pe))
			}
			continue
		}

		// Normalize attribute name (remove data- prefix)
		normalizedName := normalizeAttributeName(name)
		var adjustedKeySpan *util.ParseSourceSpan
		if attr.KeySpan != nil {
			normalizationAdjustment := len(name) - len(normalizedName)
			if normalizationAdjustment > 0 {
				detailsStr := normalizedName
				adjustedKeySpan = util.NewParseSourceSpan(
					attr.KeySpan.Start.MoveBy(normalizationAdjustment),
					attr.KeySpan.Start.MoveBy(normalizationAdjustment+len(normalizedName)),
					attr.KeySpan.Start.MoveBy(normalizationAdjustment),
					&detailsStr,
				)
			} else {
				adjustedKeySpan = attr.KeySpan
			}
		} else {
			adjustedKeySpan = attr.KeySpan
		}

		// Check for interpolation in attribute value
		hasInterpolation := t.bindingParser.ParsePropertyInterpolation(
			normalizedName,
			value,
			attr.SourceSpan(),
			attr.ValueSpan,
			&matchableAttrs,
			&parsedProperties,
			adjustedKeySpan,
			attr.ValueTokens,
		)

		if !hasInterpolation {
			// Regular attribute
			t.bindingParser.ParseLiteralAttr(
				normalizedName,
				&value,
				attr.SourceSpan(),
				attr.SourceSpan().Start.Offset,
				attr.ValueSpan,
				&matchableAttrs,
				&parsedProperties,
				adjustedKeySpan,
			)
		}
	}

	// Convert ParsedProperty to BoundAttribute or TextAttribute
	for _, prop := range parsedProperties {
		if prop.IsLiteral {
			valueStr := ""
			if prop.Expression != nil && prop.Expression.Source != nil {
				valueStr = *prop.Expression.Source
			}
			textAttr := render3.NewTextAttribute(
				prop.Name,
				valueStr,
				prop.SourceSpan,
				prop.KeySpan,
				prop.ValueSpan,
				nil, // i18n
			)
			attrs = append(attrs, textAttr)
		} else {
			boundProp := t.bindingParser.CreateBoundElementProperty(
				&elementSelector,
				prop,
				false, // skipValidation
				true,  // mapPropertyName
			)
			if boundProp != nil {
				boundAttr := render3.NewBoundAttribute(
					boundProp.Name,
					boundProp.Type,
					boundProp.SecurityContext,
					boundProp.Value,
					boundProp.Unit,
					boundProp.SourceSpan,
					boundProp.KeySpan,
					boundProp.ValueSpan,
					nil, // i18n
				)
				inputs = append(inputs, boundAttr)
			}
		}
	}

	// Process directives
	directives := []*render3.Directive{}
	for _, directive := range component.Directives {
		// Check if this is an animation shorthand (@animation without parentheses)
		// When selectorlessEnabled=false, standalone @ directives are animation shorthands
		// When selectorlessEnabled=true, they are actual directives
		if !t.options.SelectorlessEnabled && len(directive.Attrs) == 0 {
			// This could be @animation shorthand - treat as a bound property
			sourceSpanStr := ""
			if directive.SourceSpan() != nil {
				sourceSpanStr = directive.SourceSpan().String()
			}

			if len(sourceSpanStr) > 1 && sourceSpanStr[0] == '@' {
				// This is @animation shorthand
				localParsedProps := []*expression_parser.ParsedProperty{}
				absoluteOffset := 0
				if directive.SourceSpan() != nil {
					absoluteOffset = directive.SourceSpan().Start.Offset
				}

				// Create adjusted KeySpan to exclude '@' prefix
				var adjustedKeySpan *util.ParseSourceSpan
				if directive.StartSourceSpan != nil {
					animationName := directive.Name
					detailsStr := animationName
					adjustedKeySpan = util.NewParseSourceSpan(
						directive.StartSourceSpan.Start.MoveBy(1), // Skip '@'
						directive.StartSourceSpan.End,
						directive.StartSourceSpan.Start.MoveBy(1),
						&detailsStr,
					)
				} else {
					adjustedKeySpan = directive.SourceSpan()
				}

				t.bindingParser.ParsePropertyBinding(
					sourceSpanStr, // Pass full "@animation" name
					"",            // Empty value
					false,         // isHost
					false,         // isPartOfAssignmentBinding
					directive.SourceSpan(),
					absoluteOffset,
					nil, // ValueSpan
					&matchableAttrs,
					&localParsedProps,
					adjustedKeySpan,
				)

				for _, prop := range localParsedProps {
					boundProp := t.bindingParser.CreateBoundElementProperty(
						&elementSelector,
						prop,
						false, // skipValidation
						true,  // mapPropertyName
					)
					if boundProp != nil {
						boundAttr := render3.NewBoundAttribute(
							boundProp.Name,
							boundProp.Type,
							boundProp.SecurityContext,
							boundProp.Value,
							boundProp.Unit,
							boundProp.SourceSpan,
							boundProp.KeySpan,
							boundProp.ValueSpan,
							nil, // i18n
						)
						inputs = append(inputs, boundAttr)
					}
				}
				continue
			}
		}

		// Parse directive attributes (similar to element attributes)
		directiveAttrs := []*render3.TextAttribute{}
		directiveInputs := []*render3.BoundAttribute{}
		directiveOutputs := []*render3.BoundEvent{}
		directiveReferences := []*render3.Reference{}
		directiveParsedProperties := []*expression_parser.ParsedProperty{}
		directiveParsedEvents := []*expression_parser.ParsedEvent{}
		directiveMatchableAttrs := []string{}

		for _, attr := range directive.Attrs {
			name := strings.TrimSpace(attr.Name)
			value := attr.Value

			// Check for reference (#ref or ref-)
			if len(name) > 0 && name[0] == '#' {
				refName := name[1:]
				refValue := value
				if refValue == "" {
					refValue = ""
				}
				directiveReferences = append(directiveReferences, render3.NewReference(
					refName,
					refValue,
					attr.SourceSpan(),
					attr.KeySpan,
					attr.ValueSpan,
				))
				directiveMatchableAttrs = append(directiveMatchableAttrs, name, value)
				continue
			}

			// Check for property binding ([prop]="value")
			if len(name) > 2 && name[0] == '[' && name[len(name)-1] == ']' {
				propName := name[1 : len(name)-1]
				absoluteOffset := attr.SourceSpan().Start.Offset
				if attr.ValueSpan != nil {
					absoluteOffset = attr.ValueSpan.FullStart.Offset
				}
				var adjustedKeySpan *util.ParseSourceSpan
				if attr.KeySpan != nil {
					detailsStr := propName
					adjustedKeySpan = util.NewParseSourceSpan(
						attr.KeySpan.Start.MoveBy(1),
						attr.KeySpan.Start.MoveBy(1+len(propName)),
						attr.KeySpan.Start.MoveBy(1),
						&detailsStr,
					)
				} else {
					adjustedKeySpan = attr.KeySpan
				}
				t.bindingParser.ParsePropertyBinding(
					propName,
					value,
					false, // isHost
					false, // isPartOfAssignmentBinding
					attr.SourceSpan(),
					absoluteOffset,
					attr.ValueSpan,
					&directiveMatchableAttrs,
					&directiveParsedProperties,
					adjustedKeySpan,
				)
				continue
			}

			// Check for event binding ((event)="handler")
			if len(name) > 2 && name[0] == '(' && name[len(name)-1] == ')' {
				eventName := name[1 : len(name)-1]
				var adjustedKeySpan *util.ParseSourceSpan
				if attr.KeySpan != nil {
					detailsStr := eventName
					adjustedKeySpan = util.NewParseSourceSpan(
						attr.KeySpan.Start.MoveBy(1),
						attr.KeySpan.Start.MoveBy(1+len(eventName)),
						attr.KeySpan.Start.MoveBy(1),
						&detailsStr,
					)
				} else {
					adjustedKeySpan = attr.KeySpan
				}
				t.bindingParser.ParseEvent(
					eventName,
					value,
					false, // isAssignmentEvent
					attr.SourceSpan(),
					attr.ValueSpan,
					&directiveMatchableAttrs,
					&directiveParsedEvents,
					adjustedKeySpan,
				)
				continue
			}

			// Normalize attribute name (remove data- prefix)
			normalizedName := normalizeAttributeName(name)
			var adjustedKeySpan *util.ParseSourceSpan
			if attr.KeySpan != nil {
				normalizationAdjustment := len(name) - len(normalizedName)
				if normalizationAdjustment > 0 {
					detailsStr := normalizedName
					adjustedKeySpan = util.NewParseSourceSpan(
						attr.KeySpan.Start.MoveBy(normalizationAdjustment),
						attr.KeySpan.Start.MoveBy(normalizationAdjustment+len(normalizedName)),
						attr.KeySpan.Start.MoveBy(normalizationAdjustment),
						&detailsStr,
					)
				} else {
					adjustedKeySpan = attr.KeySpan
				}
			} else {
				adjustedKeySpan = attr.KeySpan
			}

			// Check for interpolation in attribute value
			hasInterpolation := t.bindingParser.ParsePropertyInterpolation(
				normalizedName,
				value,
				attr.SourceSpan(),
				attr.ValueSpan,
				&directiveMatchableAttrs,
				&directiveParsedProperties,
				adjustedKeySpan,
				attr.ValueTokens,
			)

			if !hasInterpolation {
				// Regular attribute
				t.bindingParser.ParseLiteralAttr(
					normalizedName,
					&value,
					attr.SourceSpan(),
					attr.SourceSpan().Start.Offset,
					attr.ValueSpan,
					&directiveMatchableAttrs,
					&directiveParsedProperties,
					adjustedKeySpan,
				)
			}
		}

		// Convert ParsedProperty to BoundAttribute or TextAttribute
		for _, prop := range directiveParsedProperties {
			if prop.IsLiteral {
				valueStr := ""
				if prop.Expression != nil && prop.Expression.Source != nil {
					valueStr = *prop.Expression.Source
				}
				textAttr := render3.NewTextAttribute(
					prop.Name,
					valueStr,
					prop.SourceSpan,
					prop.KeySpan,
					prop.ValueSpan,
					nil, // i18n
				)
				directiveAttrs = append(directiveAttrs, textAttr)
			} else {
				boundProp := t.bindingParser.CreateBoundElementProperty(
					&elementSelector,
					prop,
					false, // skipValidation
					true,  // mapPropertyName
				)
				if boundProp != nil {
					boundAttr := render3.NewBoundAttribute(
						boundProp.Name,
						boundProp.Type,
						boundProp.SecurityContext,
						boundProp.Value,
						boundProp.Unit,
						boundProp.SourceSpan,
						boundProp.KeySpan,
						boundProp.ValueSpan,
						nil, // i18n
					)
					directiveInputs = append(directiveInputs, boundAttr)
				}
			}
		}

		// Convert ParsedEvent to BoundEvent
		for _, pe := range directiveParsedEvents {
			directiveOutputs = append(directiveOutputs, t.convertParsedEventToBoundEvent(pe))
		}

		// Create Directive node
		directiveNode := render3.NewDirective(
			directive.Name,
			directiveAttrs,
			directiveInputs,
			directiveOutputs,
			directiveReferences,
			directive.SourceSpan(),
			directive.StartSourceSpan,
			directive.EndSourceSpan,
			nil, // i18n
		)
		directives = append(directives, directiveNode)
	}

	// Create Component node
	result := render3.NewComponent(
		component.ComponentName,
		component.TagName,
		component.FullName,
		attrs,
		inputs,
		outputs,
		directives,
		children,
		references,
		component.IsSelfClosing,
		component.SourceSpan(),
		component.StartSourceSpan,
		component.EndSourceSpan,
		component.I18n(),
	)
	fmt.Printf("[DEBUG] VisitComponent: END, component.ComponentName=%q, returning Component with %d inputs, %d outputs, %d directives\n", component.ComponentName, len(inputs), len(outputs), len(directives))
	return result
}

// VisitDirective visits a directive node
func (t *HtmlAstToIvyAst) VisitDirective(directive *ml_parser.Directive, context interface{}) interface{} {
	return nil
}

// Visit implements the Visitor interface
func (t *HtmlAstToIvyAst) Visit(node ml_parser.Node, context interface{}) interface{} {
	return node.Visit(t, context)
}
