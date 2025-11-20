package template_parser

import (
	"fmt"
	"sort"
	"strings"

	"ngc-go/packages/compiler/src/core"
	"ngc-go/packages/compiler/src/css"
	"ngc-go/packages/compiler/src/expression_parser"
	"ngc-go/packages/compiler/src/ml_parser"
	"ngc-go/packages/compiler/src/schema"
	"ngc-go/packages/compiler/src/util"
)

const PROPERTY_PARTS_SEPARATOR = "."
const ATTRIBUTE_PREFIX = "attr"
const ANIMATE_PREFIX = "animate"
const CLASS_PREFIX = "class"
const STYLE_PREFIX = "style"
const TEMPLATE_ATTR_PREFIX = "*"
const LEGACY_ANIMATE_PROP_PREFIX = "animate-"

// HostProperties represents host properties map
type HostProperties map[string]string

// HostListeners represents host listeners map
type HostListeners map[string]string

// BindingParser parses bindings in templates and in the directive host area
type BindingParser struct {
	exprParser     *expression_parser.Parser
	schemaRegistry schema.ElementSchemaRegistry
	Errors         []*util.ParseError
}

// NewBindingParser creates a new BindingParser
func NewBindingParser(
	exprParser *expression_parser.Parser,
	schemaRegistry schema.ElementSchemaRegistry,
	errors []*util.ParseError,
) *BindingParser {
	return &BindingParser{
		exprParser:     exprParser,
		schemaRegistry: schemaRegistry,
		Errors:         errors,
	}
}

// GetErrors returns the errors
func (bp *BindingParser) GetErrors() []*util.ParseError {
	return bp.Errors
}

// CreateBoundHostProperties creates bound host properties
func (bp *BindingParser) CreateBoundHostProperties(
	properties HostProperties,
	sourceSpan *util.ParseSourceSpan,
) []*expression_parser.ParsedProperty {
	boundProps := []*expression_parser.ParsedProperty{}
	for propName, expression := range properties {
		bp.ParsePropertyBinding(
			propName,
			expression,
			true,  // isHost
			false, // isPartOfAssignmentBinding
			sourceSpan,
			sourceSpan.Start.Offset,
			nil,         // valueSpan
			&[]string{}, // targetMatchableAttrs (not used)
			&boundProps,
			sourceSpan, // keySpan (use sourceSpan as keySpan)
		)
	}
	return boundProps
}

// CreateDirectiveHostEventAsts creates directive host event ASTs
func (bp *BindingParser) CreateDirectiveHostEventAsts(
	hostListeners HostListeners,
	sourceSpan *util.ParseSourceSpan,
) []*expression_parser.ParsedEvent {
	targetEvents := []*expression_parser.ParsedEvent{}
	for propName, expression := range hostListeners {
		// Use the `sourceSpan` for `keySpan` and `handlerSpan`
		targetMatchableAttrs := []string{}
		bp.parseEvent(
			propName,
			expression,
			false, // isAssignmentEvent
			sourceSpan,
			sourceSpan,            // handlerSpan
			&targetMatchableAttrs, // targetMatchableAttrs (not used)
			&targetEvents,
			sourceSpan, // keySpan
		)
	}
	return targetEvents
}

// ParseInterpolation parses an interpolation expression
func (bp *BindingParser) ParseInterpolation(
	value string,
	sourceSpan *util.ParseSourceSpan,
	interpolatedTokens interface{}, // []ml_parser.InterpolatedAttributeToken | []ml_parser.InterpolatedTextToken | nil
) *expression_parser.ASTWithSource {
	absoluteOffset := sourceSpan.FullStart.Offset

	defer func() {
		if r := recover(); r != nil {
			errMsg := fmt.Sprintf("%v", r)
			bp.reportError(errMsg, sourceSpan, util.ParseErrorLevelError)
		}
	}()

	ast := bp.exprParser.ParseInterpolation(
		value,
		sourceSpan,
		absoluteOffset,
		interpolatedTokens,
	)
	if ast != nil {
		bp.Errors = append(bp.Errors, ast.Errors...)
	}
	return ast
}

// ParseInterpolationExpression parses a single interpolation expression (for ICU switch expressions)
func (bp *BindingParser) ParseInterpolationExpression(
	expression string,
	sourceSpan *util.ParseSourceSpan,
) *expression_parser.ASTWithSource {
	absoluteOffset := sourceSpan.Start.Offset

	defer func() {
		if r := recover(); r != nil {
			errMsg := fmt.Sprintf("%v", r)
			bp.reportError(errMsg, sourceSpan, util.ParseErrorLevelError)
		}
	}()

	ast := bp.exprParser.ParseInterpolationExpression(
		expression,
		sourceSpan,
		absoluteOffset,
	)
	if ast != nil {
		bp.Errors = append(bp.Errors, ast.Errors...)
	}
	return ast
}

// ParseInlineTemplateBinding parses the bindings in a microsyntax expression
func (bp *BindingParser) ParseInlineTemplateBinding(
	tplKey string,
	tplValue string,
	sourceSpan *util.ParseSourceSpan,
	absoluteValueOffset int,
	targetMatchableAttrs *[]string, // [][]string - array of [key, value] pairs
	targetProps *[]*expression_parser.ParsedProperty,
	targetVars *[]*expression_parser.ParsedVariable,
	isIvyAst bool,
) {
	absoluteKeyOffset := sourceSpan.Start.Offset + len(TEMPLATE_ATTR_PREFIX)
	bindings := bp.parseTemplateBindings(
		tplKey,
		tplValue,
		sourceSpan,
		absoluteKeyOffset,
		absoluteValueOffset,
	)

	for _, binding := range bindings {
		// sourceSpan is for the entire HTML attribute. bindingSpan is for a particular
		// binding within the microsyntax expression so it's more narrow than sourceSpan.
		bindingSpan := moveParseSourceSpan(sourceSpan, binding.SourceSpan())
		key := ""
		var keySpan *util.ParseSourceSpan

		switch b := binding.(type) {
		case *expression_parser.VariableBinding:
			key = b.Key.Source
			keySpan = moveParseSourceSpanFromAbsolute(sourceSpan, b.Key.Span)
			value := "$implicit"
			var valueSpan *util.ParseSourceSpan
			if b.Value != nil {
				value = b.Value.Source
				valueSpan = moveParseSourceSpanFromAbsolute(sourceSpan, b.Value.Span)
			}
			*targetVars = append(*targetVars, expression_parser.NewParsedVariable(
				key,
				value,
				bindingSpan,
				keySpan,
				valueSpan,
			))
		case *expression_parser.ExpressionBinding:
			key = b.Key.Source
			keySpan = moveParseSourceSpanFromAbsolute(sourceSpan, b.Key.Span)
			if b.Value != nil {
				srcSpan := bindingSpan
				if !isIvyAst {
					srcSpan = sourceSpan
				}
				valueSpan := moveParseSourceSpanFromAbsolute(sourceSpan, b.Value.SourceSpan())
				bp.parsePropertyAst(
					key,
					b.Value,
					false, // isPartOfAssignmentBinding
					srcSpan,
					keySpan,
					valueSpan,
					targetMatchableAttrs,
					targetProps,
				)
			} else {
				// Literal attribute with no RHS
				*targetMatchableAttrs = append(*targetMatchableAttrs, key, "")
				bp.ParseLiteralAttr(
					key,
					nil, // value
					keySpan,
					absoluteValueOffset,
					nil, // valueSpan
					targetMatchableAttrs,
					targetProps,
					keySpan,
				)
			}
		}
	}
}

// parseTemplateBindings parses the bindings in a microsyntax expression
func (bp *BindingParser) parseTemplateBindings(
	tplKey string,
	tplValue string,
	sourceSpan *util.ParseSourceSpan,
	absoluteKeyOffset int,
	absoluteValueOffset int,
) []expression_parser.TemplateBinding {
	defer func() {
		if r := recover(); r != nil {
			errMsg := fmt.Sprintf("%v", r)
			bp.reportError(errMsg, sourceSpan, util.ParseErrorLevelError)
		}
	}()

	bindingsResult := bp.exprParser.ParseTemplateBindings(
		tplKey,
		tplValue,
		sourceSpan,
		absoluteKeyOffset,
		absoluteValueOffset,
	)
	if bindingsResult != nil {
		bp.Errors = append(bp.Errors, bindingsResult.Errors...)
		for _, warning := range bindingsResult.Warnings {
			bp.reportError(warning, sourceSpan, util.ParseErrorLevelWarning)
		}
		return bindingsResult.TemplateBindings
	}
	return []expression_parser.TemplateBinding{}
}

// ParseLiteralAttr parses a literal attribute
func (bp *BindingParser) ParseLiteralAttr(
	name string,
	value *string,
	sourceSpan *util.ParseSourceSpan,
	absoluteOffset int,
	valueSpan *util.ParseSourceSpan,
	targetMatchableAttrs *[]string, // [][]string - array of [key, value] pairs
	targetProps *[]*expression_parser.ParsedProperty,
	keySpan *util.ParseSourceSpan,
) {
	if isLegacyAnimationLabel(name) {
		name = name[1:]
		if keySpan != nil {
			keySpan = moveParseSourceSpanFromAbsolute(
				keySpan,
				expression_parser.NewAbsoluteSourceSpan(
					keySpan.Start.Offset+1,
					keySpan.End.Offset,
				),
			)
		}
		if value != nil {
			bp.reportError(
				"Assigning animation triggers via @prop=\"exp\" attributes with an expression is invalid. Use property bindings (e.g. [@prop]=\"exp\") or use an attribute without a value (e.g. @prop) instead.",
				sourceSpan,
				util.ParseErrorLevelError,
			)
		}
		bp.parseLegacyAnimation(
			name,
			value,
			sourceSpan,
			absoluteOffset,
			keySpan,
			valueSpan,
			targetMatchableAttrs,
			targetProps,
		)
	} else {
		valueStr := ""
		if value != nil {
			valueStr = *value
		}
		*targetProps = append(*targetProps, expression_parser.NewParsedProperty(
			name,
			bp.exprParser.WrapLiteralPrimitive(value, valueStr, absoluteOffset),
			expression_parser.ParsedPropertyTypeLiteralAttr,
			sourceSpan,
			keySpan,
			valueSpan,
		))
	}
}

// ParsePropertyBinding parses a property binding
func (bp *BindingParser) ParsePropertyBinding(
	name string,
	expression string,
	isHost bool,
	isPartOfAssignmentBinding bool,
	sourceSpan *util.ParseSourceSpan,
	absoluteOffset int,
	valueSpan *util.ParseSourceSpan,
	targetMatchableAttrs *[]string, // [][]string - array of [key, value] pairs
	targetProps *[]*expression_parser.ParsedProperty,
	keySpan *util.ParseSourceSpan,
) {
	if len(name) == 0 {
		bp.reportError("Property name is missing in binding", sourceSpan, util.ParseErrorLevelError)
	}

	isLegacyAnimationProp := false
	if strings.HasPrefix(name, LEGACY_ANIMATE_PROP_PREFIX) {
		isLegacyAnimationProp = true
		name = name[len(LEGACY_ANIMATE_PROP_PREFIX):]
		if keySpan != nil {
			keySpan = moveParseSourceSpanFromAbsolute(
				keySpan,
				expression_parser.NewAbsoluteSourceSpan(
					keySpan.Start.Offset+len(LEGACY_ANIMATE_PROP_PREFIX),
					keySpan.End.Offset,
				),
			)
		}
	} else if isLegacyAnimationLabel(name) {
		isLegacyAnimationProp = true
		name = name[1:]
		if keySpan != nil {
			keySpan = moveParseSourceSpanFromAbsolute(
				keySpan,
				expression_parser.NewAbsoluteSourceSpan(
					keySpan.Start.Offset+1,
					keySpan.End.Offset,
				),
			)
		}
	}

	if isLegacyAnimationProp {
		exprPtr := &expression
		bp.parseLegacyAnimation(
			name,
			exprPtr,
			sourceSpan,
			absoluteOffset,
			keySpan,
			valueSpan,
			targetMatchableAttrs,
			targetProps,
		)
	} else if strings.HasPrefix(name, ANIMATE_PREFIX+PROPERTY_PARTS_SEPARATOR) {
		var actualValueSpan *util.ParseSourceSpan
		if valueSpan != nil {
			actualValueSpan = valueSpan
		} else {
			actualValueSpan = sourceSpan
		}
		bp.parseAnimation(
			name,
			bp.parseBinding(expression, isHost, actualValueSpan, absoluteOffset),
			sourceSpan,
			keySpan,
			valueSpan,
			targetMatchableAttrs,
			targetProps,
		)
	} else {
		var actualValueSpan *util.ParseSourceSpan
		if valueSpan != nil {
			actualValueSpan = valueSpan
		} else {
			actualValueSpan = sourceSpan
		}
		bp.parsePropertyAst(
			name,
			bp.parseBinding(expression, isHost, actualValueSpan, absoluteOffset),
			isPartOfAssignmentBinding,
			sourceSpan,
			keySpan,
			valueSpan,
			targetMatchableAttrs,
			targetProps,
		)
	}
}

// ParsePropertyInterpolation parses a property interpolation
func (bp *BindingParser) ParsePropertyInterpolation(
	name string,
	value string,
	sourceSpan *util.ParseSourceSpan,
	valueSpan *util.ParseSourceSpan,
	targetMatchableAttrs *[]string, // [][]string - array of [key, value] pairs
	targetProps *[]*expression_parser.ParsedProperty,
	keySpan *util.ParseSourceSpan,
	interpolatedTokens interface{}, // []ml_parser.InterpolatedAttributeToken | []ml_parser.InterpolatedTextToken | nil
) bool {
	var actualValueSpan *util.ParseSourceSpan
	if valueSpan != nil {
		actualValueSpan = valueSpan
	} else {
		actualValueSpan = sourceSpan
	}
	expr := bp.ParseInterpolation(value, actualValueSpan, interpolatedTokens)
	if expr != nil {
		bp.parsePropertyAst(
			name,
			expr,
			false, // isPartOfAssignmentBinding
			sourceSpan,
			keySpan,
			valueSpan,
			targetMatchableAttrs,
			targetProps,
		)
		return true
	}
	return false
}

// parsePropertyAst parses a property AST
func (bp *BindingParser) parsePropertyAst(
	name string,
	ast *expression_parser.ASTWithSource,
	isPartOfAssignmentBinding bool,
	sourceSpan *util.ParseSourceSpan,
	keySpan *util.ParseSourceSpan,
	valueSpan *util.ParseSourceSpan,
	targetMatchableAttrs *[]string, // [][]string - array of [key, value] pairs
	targetProps *[]*expression_parser.ParsedProperty,
) {
	astSource := ""
	if ast.Source != nil {
		astSource = *ast.Source
	}
	*targetMatchableAttrs = append(*targetMatchableAttrs, name, astSource)
	propType := expression_parser.ParsedPropertyTypeDefault
	if isPartOfAssignmentBinding {
		propType = expression_parser.ParsedPropertyTypeTwoWay
	}
	*targetProps = append(*targetProps, expression_parser.NewParsedProperty(
		name,
		ast,
		propType,
		sourceSpan,
		keySpan,
		valueSpan,
	))
}

// parseAnimation parses an animation property
func (bp *BindingParser) parseAnimation(
	name string,
	ast *expression_parser.ASTWithSource,
	sourceSpan *util.ParseSourceSpan,
	keySpan *util.ParseSourceSpan,
	valueSpan *util.ParseSourceSpan,
	targetMatchableAttrs *[]string, // [][]string - array of [key, value] pairs
	targetProps *[]*expression_parser.ParsedProperty,
) {
	astSource := ""
	if ast.Source != nil {
		astSource = *ast.Source
	}
	*targetMatchableAttrs = append(*targetMatchableAttrs, name, astSource)
	*targetProps = append(*targetProps, expression_parser.NewParsedProperty(
		name,
		ast,
		expression_parser.ParsedPropertyTypeAnimation,
		sourceSpan,
		keySpan,
		valueSpan,
	))
}

// parseLegacyAnimation parses a legacy animation property
func (bp *BindingParser) parseLegacyAnimation(
	name string,
	expression *string,
	sourceSpan *util.ParseSourceSpan,
	absoluteOffset int,
	keySpan *util.ParseSourceSpan,
	valueSpan *util.ParseSourceSpan,
	targetMatchableAttrs *[]string, // [][]string - array of [key, value] pairs
	targetProps *[]*expression_parser.ParsedProperty,
) {
	if len(name) == 0 {
		bp.reportError("Animation trigger is missing", sourceSpan, util.ParseErrorLevelError)
	}

	// This will occur when a @trigger is not paired with an expression.
	// For animations it is valid to not have an expression since */void
	// states will be applied by angular when the element is attached/detached
	exprValue := "undefined"
	if expression != nil {
		exprValue = *expression
	}
	var actualValueSpan *util.ParseSourceSpan
	if valueSpan != nil {
		actualValueSpan = valueSpan
	} else {
		actualValueSpan = sourceSpan
	}
	ast := bp.parseBinding(exprValue, false, actualValueSpan, absoluteOffset)
	astSource := ""
	if ast.Source != nil {
		astSource = *ast.Source
	}
	*targetMatchableAttrs = append(*targetMatchableAttrs, name, astSource)
	*targetProps = append(*targetProps, expression_parser.NewParsedProperty(
		name,
		ast,
		expression_parser.ParsedPropertyTypeLegacyAnimation,
		sourceSpan,
		keySpan,
		valueSpan,
	))
}

// ParseBinding parses a binding expression
func (bp *BindingParser) ParseBinding(
	value string,
	isHostBinding bool,
	sourceSpan *util.ParseSourceSpan,
	absoluteOffset int,
) *expression_parser.ASTWithSource {
	defer func() {
		if r := recover(); r != nil {
			errMsg := fmt.Sprintf("%v", r)
			bp.reportError(errMsg, sourceSpan, util.ParseErrorLevelError)
		}
	}()

	var ast *expression_parser.ASTWithSource
	if isHostBinding {
		ast = bp.exprParser.ParseSimpleBinding(value, sourceSpan, absoluteOffset)
	} else {
		ast = bp.exprParser.ParseBinding(value, sourceSpan, absoluteOffset)
	}
	if ast != nil {
		bp.Errors = append(bp.Errors, ast.Errors...)
		return ast
	}
	return bp.exprParser.WrapLiteralPrimitive(nil, "ERROR", absoluteOffset)
}

// parseBinding is a helper that calls ParseBinding (for consistency with TypeScript naming)
func (bp *BindingParser) parseBinding(
	value string,
	isHostBinding bool,
	sourceSpan *util.ParseSourceSpan,
	absoluteOffset int,
) *expression_parser.ASTWithSource {
	return bp.ParseBinding(value, isHostBinding, sourceSpan, absoluteOffset)
}

// CreateBoundElementProperty creates a bound element property
func (bp *BindingParser) CreateBoundElementProperty(
	elementSelector *string,
	boundProp *expression_parser.ParsedProperty,
	skipValidation bool,
	mapPropertyName bool,
) *expression_parser.BoundElementProperty {
	if boundProp.IsLegacyAnimation {
		return expression_parser.NewBoundElementProperty(
			boundProp.Name,
			expression_parser.BindingTypeLegacyAnimation,
			core.SecurityContextNONE,
			boundProp.Expression,
			nil, // unit
			boundProp.SourceSpan,
			boundProp.KeySpan,
			boundProp.ValueSpan,
		)
	}

	var unit *string
	var bindingType expression_parser.BindingType
	var boundPropertyName *string
	parts := strings.Split(boundProp.Name, PROPERTY_PARTS_SEPARATOR)
	var securityContexts []core.SecurityContext

	// Check for special cases (prefix style, attr, class)
	if len(parts) > 1 {
		if parts[0] == ATTRIBUTE_PREFIX {
			boundPropertyName = new(string)
			*boundPropertyName = strings.Join(parts[1:], PROPERTY_PARTS_SEPARATOR)
			if !skipValidation {
				bp.validatePropertyOrAttributeName(*boundPropertyName, boundProp.SourceSpan, true)
			}
			selectorStr := ""
			if elementSelector != nil {
				selectorStr = *elementSelector
			}
			securityContexts = calcPossibleSecurityContexts(
				bp.schemaRegistry,
				selectorStr,
				*boundPropertyName,
				true,
			)

			nsSeparatorIdx := strings.Index(*boundPropertyName, ":")
			if nsSeparatorIdx > -1 {
				ns := (*boundPropertyName)[:nsSeparatorIdx]
				name := (*boundPropertyName)[nsSeparatorIdx+1:]
				merged := ml_parser.MergeNsAndName(ns, name)
				boundPropertyName = &merged
			}

			bindingType = expression_parser.BindingTypeAttribute
		} else if parts[0] == CLASS_PREFIX {
			boundPropertyName = &parts[1]
			bindingType = expression_parser.BindingTypeClass
			securityContexts = []core.SecurityContext{core.SecurityContextNONE}
		} else if parts[0] == STYLE_PREFIX {
			if len(parts) > 2 {
				unit = &parts[2]
			}
			boundPropertyName = &parts[1]
			bindingType = expression_parser.BindingTypeStyle
			securityContexts = []core.SecurityContext{core.SecurityContextSTYLE}
		} else if parts[0] == ANIMATE_PREFIX {
			boundPropertyName = &boundProp.Name
			bindingType = expression_parser.BindingTypeAnimation
			securityContexts = []core.SecurityContext{core.SecurityContextNONE}
		}
	}

	// If not a special case, use the full property name
	if boundPropertyName == nil {
		mappedPropName := bp.schemaRegistry.GetMappedPropName(boundProp.Name)
		if mapPropertyName {
			boundPropertyName = &mappedPropName
		} else {
			boundPropertyName = &boundProp.Name
		}
		selectorStr := ""
		if elementSelector != nil {
			selectorStr = *elementSelector
		}
		securityContexts = calcPossibleSecurityContexts(
			bp.schemaRegistry,
			selectorStr,
			mappedPropName,
			false,
		)
		if boundProp.Type == expression_parser.ParsedPropertyTypeTwoWay {
			bindingType = expression_parser.BindingTypeTwoWay
		} else {
			bindingType = expression_parser.BindingTypeProperty
		}
		if !skipValidation {
			bp.validatePropertyOrAttributeName(mappedPropName, boundProp.SourceSpan, false)
		}
	}

	securityContext := core.SecurityContextNONE
	if len(securityContexts) > 0 {
		securityContext = securityContexts[0]
	}

	return expression_parser.NewBoundElementProperty(
		*boundPropertyName,
		bindingType,
		securityContext,
		boundProp.Expression,
		unit,
		boundProp.SourceSpan,
		boundProp.KeySpan,
		boundProp.ValueSpan,
	)
}

// ParseEvent parses an event binding
func (bp *BindingParser) ParseEvent(
	name string,
	expression string,
	isAssignmentEvent bool,
	sourceSpan *util.ParseSourceSpan,
	handlerSpan *util.ParseSourceSpan,
	targetMatchableAttrs *[]string, // [][]string - array of [key, value] pairs
	targetEvents *[]*expression_parser.ParsedEvent,
	keySpan *util.ParseSourceSpan,
) {
	if len(name) == 0 {
		bp.reportError("Event name is missing in binding", sourceSpan, util.ParseErrorLevelError)
	}

	if isLegacyAnimationLabel(name) {
		name = name[1:]
		if keySpan != nil {
			keySpan = moveParseSourceSpanFromAbsolute(
				keySpan,
				expression_parser.NewAbsoluteSourceSpan(
					keySpan.Start.Offset+1,
					keySpan.End.Offset,
				),
			)
		}
		bp.parseLegacyAnimationEvent(
			name,
			expression,
			sourceSpan,
			handlerSpan,
			targetEvents,
			keySpan,
		)
	} else {
		bp.parseRegularEvent(
			name,
			expression,
			isAssignmentEvent,
			sourceSpan,
			handlerSpan,
			targetMatchableAttrs,
			targetEvents,
			keySpan,
		)
	}
}

// parseEvent is a helper that calls ParseEvent (for consistency with TypeScript naming)
func (bp *BindingParser) parseEvent(
	name string,
	expression string,
	isAssignmentEvent bool,
	sourceSpan *util.ParseSourceSpan,
	handlerSpan *util.ParseSourceSpan,
	targetMatchableAttrs *[]string,
	targetEvents *[]*expression_parser.ParsedEvent,
	keySpan *util.ParseSourceSpan,
) {
	bp.ParseEvent(name, expression, isAssignmentEvent, sourceSpan, handlerSpan, targetMatchableAttrs, targetEvents, keySpan)
}

// CalcPossibleSecurityContexts calculates possible security contexts
func (bp *BindingParser) CalcPossibleSecurityContexts(
	selector string,
	propName string,
	isAttribute bool,
) []core.SecurityContext {
	prop := bp.schemaRegistry.GetMappedPropName(propName)
	return calcPossibleSecurityContexts(bp.schemaRegistry, selector, prop, isAttribute)
}

// ParseEventListenerName parses an event listener name
func (bp *BindingParser) ParseEventListenerName(rawName string) (eventName string, target *string) {
	parts := util.SplitAtColon(rawName, []string{"", rawName})
	if len(parts) >= 2 && parts[0] != "" {
		targetStr := parts[0]
		target = &targetStr
		eventName = parts[1]
	} else {
		eventName = rawName
	}
	return eventName, target
}

// ParseLegacyAnimationEventName parses a legacy animation event name
func (bp *BindingParser) ParseLegacyAnimationEventName(rawName string) (eventName string, phase *string) {
	matches := util.SplitAtPeriod(rawName, []string{rawName, ""})
	eventName = matches[0]
	if len(matches) > 1 && matches[1] != "" {
		phaseStr := strings.ToLower(matches[1])
		phase = &phaseStr
	}
	return eventName, phase
}

// parseLegacyAnimationEvent parses a legacy animation event
func (bp *BindingParser) parseLegacyAnimationEvent(
	name string,
	expression string,
	sourceSpan *util.ParseSourceSpan,
	handlerSpan *util.ParseSourceSpan,
	targetEvents *[]*expression_parser.ParsedEvent,
	keySpan *util.ParseSourceSpan,
) {
	eventName, phase := bp.ParseLegacyAnimationEventName(name)
	ast := bp.parseAction(expression, handlerSpan)
	*targetEvents = append(*targetEvents, expression_parser.NewParsedEvent(
		eventName,
		phase,
		expression_parser.ParsedEventTypeLegacyAnimation,
		ast,
		sourceSpan,
		handlerSpan,
		keySpan,
	))

	if len(eventName) == 0 {
		bp.reportError("Animation event name is missing in binding", sourceSpan, util.ParseErrorLevelError)
	}
	if phase != nil {
		if *phase != "start" && *phase != "done" {
			bp.reportError(
				fmt.Sprintf("The provided animation output phase value \"%s\" for \"@%s\" is not supported (use start or done)", *phase, eventName),
				sourceSpan,
				util.ParseErrorLevelError,
			)
		}
	} else {
		bp.reportError(
			fmt.Sprintf("The animation trigger output event (@%s) is missing its phase value name (start or done are currently supported)", eventName),
			sourceSpan,
			util.ParseErrorLevelError,
		)
	}
}

// parseRegularEvent parses a regular event
func (bp *BindingParser) parseRegularEvent(
	name string,
	expression string,
	isAssignmentEvent bool,
	sourceSpan *util.ParseSourceSpan,
	handlerSpan *util.ParseSourceSpan,
	targetMatchableAttrs *[]string, // [][]string - array of [key, value] pairs
	targetEvents *[]*expression_parser.ParsedEvent,
	keySpan *util.ParseSourceSpan,
) {
	// long format: 'target: eventName'
	eventName, target := bp.ParseEventListenerName(name)
	prevErrorCount := len(bp.Errors)
	ast := bp.parseAction(expression, handlerSpan)
	isValid := len(bp.Errors) == prevErrorCount
	astSource := ""
	if ast.Source != nil {
		astSource = *ast.Source
	}
	*targetMatchableAttrs = append(*targetMatchableAttrs, name, astSource)

	// Don't try to validate assignment events if there were other
	// parsing errors to avoid adding more noise to the error logs.
	if isAssignmentEvent && isValid && !bp.isAllowedAssignmentEvent(ast.AST) {
		bp.reportError("Unsupported expression in a two-way binding", sourceSpan, util.ParseErrorLevelError)
	}

	eventType := expression_parser.ParsedEventTypeRegular
	if isAssignmentEvent {
		eventType = expression_parser.ParsedEventTypeTwoWay
	}
	if strings.HasPrefix(name, ANIMATE_PREFIX+PROPERTY_PARTS_SEPARATOR) {
		eventType = expression_parser.ParsedEventTypeAnimation
	}

	*targetEvents = append(*targetEvents, expression_parser.NewParsedEvent(
		eventName,
		target,
		eventType,
		ast,
		sourceSpan,
		handlerSpan,
		keySpan,
	))
	// Don't detect directives for event names for now,
	// so don't add the event name to the matchableAttrs
}

// parseAction parses an action expression
func (bp *BindingParser) parseAction(value string, sourceSpan *util.ParseSourceSpan) *expression_parser.ASTWithSource {
	absoluteOffset := 0
	if sourceSpan != nil && sourceSpan.Start != nil {
		absoluteOffset = sourceSpan.Start.Offset
	}

	defer func() {
		if r := recover(); r != nil {
			errMsg := fmt.Sprintf("%v", r)
			bp.reportError(errMsg, sourceSpan, util.ParseErrorLevelError)
		}
	}()

	ast := bp.exprParser.ParseAction(value, sourceSpan, absoluteOffset)
	if ast != nil {
		bp.Errors = append(bp.Errors, ast.Errors...)
	}
	if ast == nil || isEmptyExpr(ast.AST) {
		bp.reportError("Empty expressions are not allowed", sourceSpan, util.ParseErrorLevelError)
		return bp.exprParser.WrapLiteralPrimitive(nil, "ERROR", absoluteOffset)
	}
	return ast
}

// reportError reports an error
func (bp *BindingParser) reportError(
	message string,
	sourceSpan *util.ParseSourceSpan,
	level util.ParseErrorLevel,
) {
	if level == 0 {
		level = util.ParseErrorLevelError
	}
	bp.Errors = append(bp.Errors, util.NewParseError(sourceSpan, message))
	// Set level on the error
	if len(bp.Errors) > 0 {
		bp.Errors[len(bp.Errors)-1].Level = level
	}
}

// validatePropertyOrAttributeName validates a property or attribute name
func (bp *BindingParser) validatePropertyOrAttributeName(
	propName string,
	sourceSpan *util.ParseSourceSpan,
	isAttr bool,
) {
	var report schema.PropertyValidationResult
	if isAttr {
		report = bp.schemaRegistry.ValidateAttribute(propName)
	} else {
		report = bp.schemaRegistry.ValidateProperty(propName)
	}
	if report.Error {
		bp.reportError(report.Msg, sourceSpan, util.ParseErrorLevelError)
	}
}

// isAllowedAssignmentEvent checks if an AST is allowed to be used within the event side of a two-way binding
func (bp *BindingParser) isAllowedAssignmentEvent(ast expression_parser.AST) bool {
	if astWithSource, ok := ast.(*expression_parser.ASTWithSource); ok {
		return bp.isAllowedAssignmentEvent(astWithSource.AST)
	}

	if nonNullAssert, ok := ast.(*expression_parser.NonNullAssert); ok {
		return bp.isAllowedAssignmentEvent(nonNullAssert.Expression)
	}

	if call, ok := ast.(*expression_parser.Call); ok {
		if len(call.Args) == 1 {
			if propRead, ok := call.Receiver.(*expression_parser.PropertyRead); ok {
				if propRead.Name == "$any" {
					if _, ok := propRead.Receiver.(*expression_parser.ImplicitReceiver); ok {
						if _, isThisReceiver := propRead.Receiver.(*expression_parser.ThisReceiver); !isThisReceiver {
							return bp.isAllowedAssignmentEvent(call.Args[0])
						}
					}
				}
			}
		}
	}

	if propRead, ok := ast.(*expression_parser.PropertyRead); ok {
		return !hasRecursiveSafeReceiver(propRead)
	}

	if keyedRead, ok := ast.(*expression_parser.KeyedRead); ok {
		return !hasRecursiveSafeReceiver(keyedRead)
	}

	return false
}

// hasRecursiveSafeReceiver checks if an AST has a recursive safe receiver
func hasRecursiveSafeReceiver(ast expression_parser.AST) bool {
	if _, ok := ast.(*expression_parser.SafePropertyRead); ok {
		return true
	}

	if _, ok := ast.(*expression_parser.SafeKeyedRead); ok {
		return true
	}

	if parenthesized, ok := ast.(*expression_parser.ParenthesizedExpression); ok {
		return hasRecursiveSafeReceiver(parenthesized.Expression)
	}

	if propRead, ok := ast.(*expression_parser.PropertyRead); ok {
		return hasRecursiveSafeReceiver(propRead.Receiver)
	}

	if keyedRead, ok := ast.(*expression_parser.KeyedRead); ok {
		return hasRecursiveSafeReceiver(keyedRead.Receiver)
	}

	if call, ok := ast.(*expression_parser.Call); ok {
		return hasRecursiveSafeReceiver(call.Receiver)
	}

	return false
}

// isLegacyAnimationLabel checks if a name is a legacy animation label
func isLegacyAnimationLabel(name string) bool {
	return len(name) > 0 && name[0] == '@'
}

// isEmptyExpr checks if an AST is an EmptyExpr
func isEmptyExpr(ast expression_parser.AST) bool {
	_, ok := ast.(*expression_parser.EmptyExpr)
	return ok
}

// calcPossibleSecurityContexts calculates possible security contexts
func calcPossibleSecurityContexts(
	registry schema.ElementSchemaRegistry,
	selector string,
	propName string,
	isAttribute bool,
) []core.SecurityContext {
	var ctxs []core.SecurityContext
	nameToContext := func(elName string) core.SecurityContext {
		return registry.SecurityContext(elName, propName, isAttribute)
	}

	if selector == "" {
		elementNames := registry.AllKnownElementNames()
		ctxs = make([]core.SecurityContext, len(elementNames))
		for i, elName := range elementNames {
			ctxs[i] = nameToContext(elName)
		}
	} else {
		ctxs = []core.SecurityContext{}
		selectors, err := css.ParseCssSelector(selector)
		if err != nil {
			return []core.SecurityContext{core.SecurityContextNONE}
		}
		for _, sel := range selectors {
			elementNames := []string{}
			if sel.Element != nil {
				elementNames = []string{*sel.Element}
			} else {
				elementNames = registry.AllKnownElementNames()
			}

			notElementNames := make(map[string]bool)
			for _, notSel := range sel.NotSelectors {
				if notSel.IsElementSelector() && notSel.Element != nil {
					notElementNames[*notSel.Element] = true
				}
			}

			possibleElementNames := []string{}
			for _, elName := range elementNames {
				if !notElementNames[elName] {
					possibleElementNames = append(possibleElementNames, elName)
				}
			}

			for _, elName := range possibleElementNames {
				ctxs = append(ctxs, nameToContext(elName))
			}
		}
	}

	if len(ctxs) == 0 {
		return []core.SecurityContext{core.SecurityContextNONE}
	}

	// Remove duplicates and sort
	seen := make(map[core.SecurityContext]bool)
	unique := []core.SecurityContext{}
	for _, ctx := range ctxs {
		if !seen[ctx] {
			seen[ctx] = true
			unique = append(unique, ctx)
		}
	}

	sort.Slice(unique, func(i, j int) bool {
		return int(unique[i]) < int(unique[j])
	})

	return unique
}

// moveParseSourceSpan moves a ParseSourceSpan based on an absolute span
func moveParseSourceSpan(
	sourceSpan *util.ParseSourceSpan,
	absoluteSpan *expression_parser.AbsoluteSourceSpan,
) *util.ParseSourceSpan {
	// The difference of two absolute offsets provide the relative offset
	startDiff := absoluteSpan.Start - sourceSpan.Start.Offset
	endDiff := absoluteSpan.End - sourceSpan.End.Offset
	return util.NewParseSourceSpan(
		sourceSpan.Start.MoveBy(startDiff),
		sourceSpan.End.MoveBy(endDiff),
		sourceSpan.FullStart.MoveBy(startDiff),
		sourceSpan.Details,
	)
}

// moveParseSourceSpanFromAbsolute is a helper that creates an AbsoluteSourceSpan and moves
func moveParseSourceSpanFromAbsolute(
	sourceSpan *util.ParseSourceSpan,
	absoluteSpan *expression_parser.AbsoluteSourceSpan,
) *util.ParseSourceSpan {
	return moveParseSourceSpan(sourceSpan, absoluteSpan)
}
