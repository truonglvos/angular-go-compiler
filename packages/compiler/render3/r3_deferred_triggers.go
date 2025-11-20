package render3

import (
	"ngc-go/packages/compiler/core"
	"ngc-go/packages/compiler/expression_parser"
	"ngc-go/packages/compiler/ml_parser"
	"ngc-go/packages/compiler/util"
	"regexp"
	"strconv"
	"strings"
)

// Pattern for a timing value in a trigger
var timePattern = regexp.MustCompile(`^\d+\.?\d*(ms|s)?$`)

// Pattern for a separator between keywords in a trigger expression
var separatorPattern = regexp.MustCompile(`^\s$`)

// CommaDelimitedSyntax maps opening characters to closing characters
var commaDelimitedSyntax = map[int]int{
	core.CharLBRACE:   core.CharRBRACE,   // Object literals
	core.CharLBRACKET: core.CharRBRACKET, // Array literals
	core.CharLPAREN:   core.CharRPAREN,   // Function calls
}

// OnTriggerType represents possible types of `on` triggers
type OnTriggerType string

const (
	OnTriggerTypeIdle        OnTriggerType = "idle"
	OnTriggerTypeTimer       OnTriggerType = "timer"
	OnTriggerTypeInteraction OnTriggerType = "interaction"
	OnTriggerTypeImmediate   OnTriggerType = "immediate"
	OnTriggerTypeHover       OnTriggerType = "hover"
	OnTriggerTypeViewport    OnTriggerType = "viewport"
	OnTriggerTypeNever       OnTriggerType = "never"
)

// ReferenceTriggerValidator is a function that validates the structure of a reference-based trigger
type ReferenceTriggerValidator func(triggerType OnTriggerType, parameters []ParsedParameter) error

// ParsedParameter represents parsed information about a defer trigger parameter
type ParsedParameter struct {
	// Expression of the parameter
	Expression string
	// Index within the trigger at which the parameter starts
	Start int
}

// ParseNeverTrigger parses a `never` deferred trigger
func ParseNeverTrigger(
	param *ml_parser.BlockParameter,
	triggers *DeferredBlockTriggers,
	errors *[]*util.ParseError,
) {
	sourceSpan := param.SourceSpan()
	neverIndex := strings.Index(param.Expression, "never")
	var neverSourceSpan *util.ParseSourceSpan
	if neverIndex != -1 {
		neverSourceSpan = util.NewParseSourceSpan(
			sourceSpan.Start.MoveBy(neverIndex),
			sourceSpan.Start.MoveBy(neverIndex+len("never")),
			nil,
			nil,
		)
	}
	prefetchSpan := getPrefetchSpan(param.Expression, sourceSpan)
	hydrateSpan := getHydrateSpan(param.Expression, sourceSpan)

	if neverIndex == -1 {
		*errors = append(*errors, util.NewParseError(sourceSpan, `Could not find "never" keyword in expression`))
	} else {
		trackTrigger(
			"never",
			triggers,
			errors,
			NewNeverDeferredTrigger(neverSourceSpan, sourceSpan, prefetchSpan, nil, hydrateSpan),
		)
	}
}

// ParseWhenTrigger parses a `when` deferred trigger
func ParseWhenTrigger(
	param *ml_parser.BlockParameter,
	bindingParser BindingParser,
	triggers *DeferredBlockTriggers,
	errors *[]*util.ParseError,
) {
	sourceSpan := param.SourceSpan()
	whenIndex := strings.Index(param.Expression, "when")
	var whenSourceSpan *util.ParseSourceSpan
	if whenIndex != -1 {
		whenSourceSpan = util.NewParseSourceSpan(
			sourceSpan.Start.MoveBy(whenIndex),
			sourceSpan.Start.MoveBy(whenIndex+len("when")),
			nil,
			nil,
		)
	}
	prefetchSpan := getPrefetchSpan(param.Expression, sourceSpan)
	hydrateSpan := getHydrateSpan(param.Expression, sourceSpan)

	if whenIndex == -1 {
		*errors = append(*errors, util.NewParseError(sourceSpan, `Could not find "when" keyword in expression`))
	} else {
		start := GetTriggerParametersStart(param.Expression, whenIndex+1)
		parsed := bindingParser.ParseBinding(
			param.Expression[start:],
			false,
			sourceSpan,
			sourceSpan.Start.Offset+start,
		)
		trackTrigger(
			"when",
			triggers,
			errors,
			NewBoundDeferredTrigger(parsed.AST, sourceSpan, prefetchSpan, whenSourceSpan, hydrateSpan),
		)
	}
}

// ParseOnTrigger parses an `on` trigger
func ParseOnTrigger(
	param *ml_parser.BlockParameter,
	bindingParser BindingParser,
	triggers *DeferredBlockTriggers,
	errors *[]*util.ParseError,
	placeholder *DeferredBlockPlaceholder,
) {
	sourceSpan := param.SourceSpan()
	onIndex := strings.Index(param.Expression, "on")
	var onSourceSpan *util.ParseSourceSpan
	if onIndex != -1 {
		onSourceSpan = util.NewParseSourceSpan(
			sourceSpan.Start.MoveBy(onIndex),
			sourceSpan.Start.MoveBy(onIndex+len("on")),
			nil,
			nil,
		)
	}
	prefetchSpan := getPrefetchSpan(param.Expression, sourceSpan)
	hydrateSpan := getHydrateSpan(param.Expression, sourceSpan)

	if onIndex == -1 {
		*errors = append(*errors, util.NewParseError(sourceSpan, `Could not find "on" keyword in expression`))
	} else {
		start := GetTriggerParametersStart(param.Expression, onIndex+1)
		isHydrationTrigger := strings.HasPrefix(param.Expression, "hydrate")
		parser := NewOnTriggerParser(
			param.Expression,
			bindingParser,
			start,
			sourceSpan,
			triggers,
			errors,
			func(triggerType OnTriggerType, parameters []ParsedParameter) error {
				if isHydrationTrigger {
					return validateHydrateReferenceBasedTrigger(triggerType, parameters)
				}
				return validatePlainReferenceBasedTrigger(triggerType, parameters)
			},
			isHydrationTrigger,
			prefetchSpan,
			onSourceSpan,
			hydrateSpan,
		)
		parser.Parse()
	}
}

// getPrefetchSpan gets the prefetch span from an expression
func getPrefetchSpan(expression string, sourceSpan *util.ParseSourceSpan) *util.ParseSourceSpan {
	if !strings.HasPrefix(expression, "prefetch") {
		return nil
	}
	return util.NewParseSourceSpan(sourceSpan.Start, sourceSpan.Start.MoveBy(len("prefetch")), nil, nil)
}

// getHydrateSpan gets the hydrate span from an expression
func getHydrateSpan(expression string, sourceSpan *util.ParseSourceSpan) *util.ParseSourceSpan {
	if !strings.HasPrefix(expression, "hydrate") {
		return nil
	}
	return util.NewParseSourceSpan(sourceSpan.Start, sourceSpan.Start.MoveBy(len("hydrate")), nil, nil)
}

// OnTriggerParser parses `on` triggers
type OnTriggerParser struct {
	expression         string
	bindingParser      BindingParser
	start              int
	span               *util.ParseSourceSpan
	triggers           *DeferredBlockTriggers
	errors             *[]*util.ParseError
	validator          ReferenceTriggerValidator
	isHydrationTrigger bool
	prefetchSpan       *util.ParseSourceSpan
	onSourceSpan       *util.ParseSourceSpan
	hydrateSpan        *util.ParseSourceSpan
	index              int
	tokens             []*expression_parser.Token
}

// NewOnTriggerParser creates a new OnTriggerParser
func NewOnTriggerParser(
	expression string,
	bindingParser BindingParser,
	start int,
	span *util.ParseSourceSpan,
	triggers *DeferredBlockTriggers,
	errors *[]*util.ParseError,
	validator ReferenceTriggerValidator,
	isHydrationTrigger bool,
	prefetchSpan *util.ParseSourceSpan,
	onSourceSpan *util.ParseSourceSpan,
	hydrateSpan *util.ParseSourceSpan,
) *OnTriggerParser {
	lexer := expression_parser.NewLexer()
	tokens := lexer.Tokenize(expression[start:])
	return &OnTriggerParser{
		expression:         expression,
		bindingParser:      bindingParser,
		start:              start,
		span:               span,
		triggers:           triggers,
		errors:             errors,
		validator:          validator,
		isHydrationTrigger: isHydrationTrigger,
		prefetchSpan:       prefetchSpan,
		onSourceSpan:       onSourceSpan,
		hydrateSpan:        hydrateSpan,
		index:              0,
		tokens:             tokens,
	}
}

// Parse parses the triggers
func (p *OnTriggerParser) Parse() {
	for len(p.tokens) > 0 && p.index < len(p.tokens) {
		token := p.token()

		if !token.IsIdentifier() {
			p.unexpectedToken(token)
			break
		}

		// An identifier immediately followed by a comma or the end of
		// the expression cannot have parameters so we can exit early
		if p.isFollowedByOrLast(core.CharCOMMA) {
			p.consumeTrigger(token, []ParsedParameter{})
			p.advance()
		} else if p.isFollowedByOrLast(core.CharLPAREN) {
			p.advance() // Advance to the opening paren
			prevErrors := len(*p.errors)
			parameters := p.consumeParameters()
			if len(*p.errors) != prevErrors {
				break
			}
			p.consumeTrigger(token, parameters)
			p.advance() // Advance past the closing paren
		} else if p.index < len(p.tokens)-1 {
			p.unexpectedToken(p.tokens[p.index+1])
		}

		p.advance()
	}
}

// advance advances the parser index
func (p *OnTriggerParser) advance() {
	p.index++
}

// isFollowedByOrLast checks if the current token is followed by a character or is the last token
func (p *OnTriggerParser) isFollowedByOrLast(char int) bool {
	if p.index == len(p.tokens)-1 {
		return true
	}
	return p.tokens[p.index+1].IsCharacter(char)
}

// token returns the current token
func (p *OnTriggerParser) token() *expression_parser.Token {
	if p.index >= len(p.tokens) {
		return p.tokens[len(p.tokens)-1]
	}
	return p.tokens[p.index]
}

// consumeTrigger consumes a trigger
func (p *OnTriggerParser) consumeTrigger(identifier *expression_parser.Token, parameters []ParsedParameter) {
	triggerNameStartSpan := p.span.Start.MoveBy(
		p.start + identifier.Index - p.tokens[0].Index,
	)
	nameSpan := util.NewParseSourceSpan(
		triggerNameStartSpan,
		triggerNameStartSpan.MoveBy(len(identifier.StrValue)),
		nil,
		nil,
	)
	currentToken := p.token()
	endSpan := triggerNameStartSpan.MoveBy(currentToken.End - identifier.Index)

	// Put the prefetch and on spans with the first trigger
	isFirstTrigger := identifier.Index == 0
	var onSourceSpan *util.ParseSourceSpan
	var prefetchSourceSpan *util.ParseSourceSpan
	var hydrateSourceSpan *util.ParseSourceSpan
	if isFirstTrigger {
		onSourceSpan = p.onSourceSpan
		prefetchSourceSpan = p.prefetchSpan
		hydrateSourceSpan = p.hydrateSpan
	}
	var sourceSpanStart *util.ParseLocation
	if isFirstTrigger {
		sourceSpanStart = p.span.Start
	} else {
		sourceSpanStart = triggerNameStartSpan
	}
	sourceSpan := util.NewParseSourceSpan(sourceSpanStart, endSpan, nil, nil)

	triggerName := identifier.StrValue
	var trigger DeferredTriggerInterface
	var err error

	switch triggerName {
	case string(OnTriggerTypeIdle):
		trigger, err = createIdleTrigger(
			parameters,
			nameSpan,
			sourceSpan,
			prefetchSourceSpan,
			onSourceSpan,
			hydrateSourceSpan,
		)

	case string(OnTriggerTypeTimer):
		trigger, err = createTimerTrigger(
			parameters,
			nameSpan,
			sourceSpan,
			p.prefetchSpan,
			p.onSourceSpan,
			p.hydrateSpan,
		)

	case string(OnTriggerTypeInteraction):
		trigger, err = createInteractionTrigger(
			parameters,
			nameSpan,
			sourceSpan,
			p.prefetchSpan,
			p.onSourceSpan,
			p.hydrateSpan,
			p.validator,
		)

	case string(OnTriggerTypeImmediate):
		trigger, err = createImmediateTrigger(
			parameters,
			nameSpan,
			sourceSpan,
			p.prefetchSpan,
			p.onSourceSpan,
			p.hydrateSpan,
		)

	case string(OnTriggerTypeHover):
		trigger, err = createHoverTrigger(
			parameters,
			nameSpan,
			sourceSpan,
			p.prefetchSpan,
			p.onSourceSpan,
			p.hydrateSpan,
			p.validator,
		)

	case string(OnTriggerTypeViewport):
		trigger, err = createViewportTrigger(
			p.start,
			p.isHydrationTrigger,
			p.bindingParser,
			parameters,
			nameSpan,
			sourceSpan,
			p.prefetchSpan,
			p.onSourceSpan,
			p.hydrateSpan,
			p.validator,
		)

	default:
		err = &ParseError{Message: `Unrecognized trigger type "` + triggerName + `"`}
	}

	if err != nil {
		p.error(identifier, err.Error())
	} else {
		p.trackTrigger(triggerName, trigger)
	}
}

// consumeParameters consumes parameters from the expression
func (p *OnTriggerParser) consumeParameters() []ParsedParameter {
	parameters := []ParsedParameter{}

	if !p.token().IsCharacter(core.CharLPAREN) {
		p.unexpectedToken(p.token())
		return parameters
	}

	p.advance()

	commaDelimStack := []int{}
	var tokens []*expression_parser.Token

	for p.index < len(p.tokens) {
		token := p.token()

		// Stop parsing if we've hit the end character and we're outside of a comma-delimited syntax
		if token.IsCharacter(core.CharRPAREN) && len(commaDelimStack) == 0 {
			if len(tokens) > 0 {
				parameters = append(parameters, ParsedParameter{
					Expression: p.tokenRangeText(tokens),
					Start:      tokens[0].Index,
				})
			}
			break
		}

		// Handle comma-delimited syntax
		if token.Type == expression_parser.TokenTypeCharacter {
			if closingChar, ok := commaDelimitedSyntax[int(token.NumValue)]; ok {
				commaDelimStack = append(commaDelimStack, closingChar)
			}
		}

		if len(commaDelimStack) > 0 && token.IsCharacter(commaDelimStack[len(commaDelimStack)-1]) {
			commaDelimStack = commaDelimStack[:len(commaDelimStack)-1]
		}

		// If we hit a comma outside of a comma-delimited syntax, it means
		// that we're at the top level and we're starting a new parameter
		if len(commaDelimStack) == 0 && token.IsCharacter(core.CharCOMMA) && len(tokens) > 0 {
			parameters = append(parameters, ParsedParameter{
				Expression: p.tokenRangeText(tokens),
				Start:      tokens[0].Index,
			})
			p.advance()
			tokens = []*expression_parser.Token{}
			continue
		}

		// Otherwise treat the token as a plain text character in the current parameter
		tokens = append(tokens, token)
		p.advance()
	}

	if !p.token().IsCharacter(core.CharRPAREN) || len(commaDelimStack) > 0 {
		p.error(p.token(), "Unexpected end of expression")
	}

	if p.index < len(p.tokens)-1 && !p.tokens[p.index+1].IsCharacter(core.CharCOMMA) {
		p.unexpectedToken(p.tokens[p.index+1])
	}

	return parameters
}

// tokenRangeText gets the text for a range of tokens
func (p *OnTriggerParser) tokenRangeText(tokens []*expression_parser.Token) string {
	if len(tokens) == 0 {
		return ""
	}

	return p.expression[p.start+tokens[0].Index : p.start+tokens[len(tokens)-1].End]
}

// trackTrigger adds a trigger to the triggers map
func (p *OnTriggerParser) trackTrigger(name string, trigger DeferredTriggerInterface) {
	trackTrigger(name, p.triggers, p.errors, trigger)
}

// error adds an error
func (p *OnTriggerParser) error(token *expression_parser.Token, message string) {
	newStart := p.span.Start.MoveBy(p.start + token.Index)
	newEnd := newStart.MoveBy(token.End - token.Index)
	*p.errors = append(*p.errors, util.NewParseError(
		util.NewParseSourceSpan(newStart, newEnd, nil, nil),
		message,
	))
}

// unexpectedToken adds an error for an unexpected token
func (p *OnTriggerParser) unexpectedToken(token *expression_parser.Token) {
	p.error(token, `Unexpected token "`+token.String()+`"`)
}

// DeferredTriggerInterface is an interface for deferred triggers
type DeferredTriggerInterface interface {
	SourceSpan() *util.ParseSourceSpan
}

// trackTrigger adds a trigger to a map of triggers
func trackTrigger(
	name string,
	allTriggers *DeferredBlockTriggers,
	errors *[]*util.ParseError,
	trigger DeferredTriggerInterface,
) {
	var existingTrigger DeferredTriggerInterface
	switch name {
	case "when":
		if allTriggers.When != nil {
			existingTrigger = allTriggers.When
		}
	case "idle":
		if allTriggers.Idle != nil {
			existingTrigger = allTriggers.Idle
		}
	case "immediate":
		if allTriggers.Immediate != nil {
			existingTrigger = allTriggers.Immediate
		}
	case "hover":
		if allTriggers.Hover != nil {
			existingTrigger = allTriggers.Hover
		}
	case "timer":
		if allTriggers.Timer != nil {
			existingTrigger = allTriggers.Timer
		}
	case "interaction":
		if allTriggers.Interaction != nil {
			existingTrigger = allTriggers.Interaction
		}
	case "viewport":
		if allTriggers.Viewport != nil {
			existingTrigger = allTriggers.Viewport
		}
	case "never":
		if allTriggers.Never != nil {
			existingTrigger = allTriggers.Never
		}
	}

	if existingTrigger != nil {
		*errors = append(*errors, util.NewParseError(trigger.SourceSpan(), `Duplicate "`+name+`" trigger is not allowed`))
	} else {
		switch name {
		case "when":
			if bdt, ok := trigger.(*BoundDeferredTrigger); ok {
				allTriggers.When = bdt
			}
		case "idle":
			if idt, ok := trigger.(*IdleDeferredTrigger); ok {
				allTriggers.Idle = idt
			}
		case "immediate":
			if imt, ok := trigger.(*ImmediateDeferredTrigger); ok {
				allTriggers.Immediate = imt
			}
		case "hover":
			if ht, ok := trigger.(*HoverDeferredTrigger); ok {
				allTriggers.Hover = ht
			}
		case "timer":
			if tt, ok := trigger.(*TimerDeferredTrigger); ok {
				allTriggers.Timer = tt
			}
		case "interaction":
			if it, ok := trigger.(*InteractionDeferredTrigger); ok {
				allTriggers.Interaction = it
			}
		case "viewport":
			if vt, ok := trigger.(*ViewportDeferredTrigger); ok {
				allTriggers.Viewport = vt
			}
		case "never":
			if nt, ok := trigger.(*NeverDeferredTrigger); ok {
				allTriggers.Never = nt
			}
		}
	}
}

// ParseError represents a parse error
type ParseError struct {
	Message string
}

// Error implements the error interface
func (e *ParseError) Error() string {
	return e.Message
}

// createIdleTrigger creates an idle trigger
func createIdleTrigger(
	parameters []ParsedParameter,
	nameSpan *util.ParseSourceSpan,
	sourceSpan *util.ParseSourceSpan,
	prefetchSpan *util.ParseSourceSpan,
	onSourceSpan *util.ParseSourceSpan,
	hydrateSpan *util.ParseSourceSpan,
) (DeferredTriggerInterface, error) {
	if len(parameters) > 0 {
		return nil, &ParseError{Message: `"` + string(OnTriggerTypeIdle) + `" trigger cannot have parameters`}
	}

	return NewIdleDeferredTrigger(nameSpan, sourceSpan, prefetchSpan, onSourceSpan, hydrateSpan), nil
}

// createTimerTrigger creates a timer trigger
func createTimerTrigger(
	parameters []ParsedParameter,
	nameSpan *util.ParseSourceSpan,
	sourceSpan *util.ParseSourceSpan,
	prefetchSpan *util.ParseSourceSpan,
	onSourceSpan *util.ParseSourceSpan,
	hydrateSpan *util.ParseSourceSpan,
) (DeferredTriggerInterface, error) {
	if len(parameters) != 1 {
		return nil, &ParseError{Message: `"` + string(OnTriggerTypeTimer) + `" trigger must have exactly one parameter`}
	}

	delay := ParseDeferredTime(parameters[0].Expression)
	if delay == nil {
		return nil, &ParseError{Message: `Could not parse time value of trigger "` + string(OnTriggerTypeTimer) + `"`}
	}

	return NewTimerDeferredTrigger(*delay, nameSpan, sourceSpan, prefetchSpan, onSourceSpan, hydrateSpan), nil
}

// createImmediateTrigger creates an immediate trigger
func createImmediateTrigger(
	parameters []ParsedParameter,
	nameSpan *util.ParseSourceSpan,
	sourceSpan *util.ParseSourceSpan,
	prefetchSpan *util.ParseSourceSpan,
	onSourceSpan *util.ParseSourceSpan,
	hydrateSpan *util.ParseSourceSpan,
) (DeferredTriggerInterface, error) {
	if len(parameters) > 0 {
		return nil, &ParseError{Message: `"` + string(OnTriggerTypeImmediate) + `" trigger cannot have parameters`}
	}

	return NewImmediateDeferredTrigger(nameSpan, sourceSpan, prefetchSpan, onSourceSpan, hydrateSpan), nil
}

// createHoverTrigger creates a hover trigger
func createHoverTrigger(
	parameters []ParsedParameter,
	nameSpan *util.ParseSourceSpan,
	sourceSpan *util.ParseSourceSpan,
	prefetchSpan *util.ParseSourceSpan,
	onSourceSpan *util.ParseSourceSpan,
	hydrateSpan *util.ParseSourceSpan,
	validator ReferenceTriggerValidator,
) (DeferredTriggerInterface, error) {
	if err := validator(OnTriggerTypeHover, parameters); err != nil {
		return nil, err
	}
	var reference *string
	if len(parameters) > 0 {
		ref := parameters[0].Expression
		reference = &ref
	}
	return NewHoverDeferredTrigger(reference, nameSpan, sourceSpan, prefetchSpan, onSourceSpan, hydrateSpan), nil
}

// createInteractionTrigger creates an interaction trigger
func createInteractionTrigger(
	parameters []ParsedParameter,
	nameSpan *util.ParseSourceSpan,
	sourceSpan *util.ParseSourceSpan,
	prefetchSpan *util.ParseSourceSpan,
	onSourceSpan *util.ParseSourceSpan,
	hydrateSpan *util.ParseSourceSpan,
	validator ReferenceTriggerValidator,
) (DeferredTriggerInterface, error) {
	if err := validator(OnTriggerTypeInteraction, parameters); err != nil {
		return nil, err
	}
	var reference *string
	if len(parameters) > 0 {
		ref := parameters[0].Expression
		reference = &ref
	}
	return NewInteractionDeferredTrigger(reference, nameSpan, sourceSpan, prefetchSpan, onSourceSpan, hydrateSpan), nil
}

// createViewportTrigger creates a viewport trigger
func createViewportTrigger(
	start int,
	isHydrationTrigger bool,
	bindingParser BindingParser,
	parameters []ParsedParameter,
	nameSpan *util.ParseSourceSpan,
	sourceSpan *util.ParseSourceSpan,
	prefetchSpan *util.ParseSourceSpan,
	onSourceSpan *util.ParseSourceSpan,
	hydrateSpan *util.ParseSourceSpan,
	validator ReferenceTriggerValidator,
) (DeferredTriggerInterface, error) {
	if err := validator(OnTriggerTypeViewport, parameters); err != nil {
		return nil, err
	}

	var reference *string
	var options *expression_parser.LiteralMap

	if len(parameters) == 0 {
		reference = nil
		options = nil
	} else if !strings.HasPrefix(parameters[0].Expression, "{") {
		ref := parameters[0].Expression
		reference = &ref
		options = nil
	} else {
		parsed := bindingParser.ParseBinding(
			parameters[0].Expression,
			false,
			sourceSpan,
			sourceSpan.Start.Offset+start+parameters[0].Start,
		)

		literalMap, ok := parsed.AST.(*expression_parser.LiteralMap)
		if !ok {
			return nil, &ParseError{Message: `Options parameter of the "viewport" trigger must be an object literal`}
		}

		// Check for 'root' option
		for _, key := range literalMap.Keys {
			if key.Key == "root" {
				return nil, &ParseError{Message: `The "root" option is not supported in the options parameter of the "viewport" trigger`}
			}
		}

		triggerIndex := -1
		for i, key := range literalMap.Keys {
			if key.Key == "trigger" {
				triggerIndex = i
				break
			}
		}

		if triggerIndex == -1 {
			reference = nil
			options = literalMap
		} else {
			value := literalMap.Values[triggerIndex]
			propertyRead, ok := value.(*expression_parser.PropertyRead)
			if !ok {
				return nil, &ParseError{Message: `"trigger" option of the "viewport" trigger must be an identifier`}
			}

			_, isImplicitReceiver := propertyRead.Receiver.(*expression_parser.ImplicitReceiver)
			_, isThisReceiver := propertyRead.Receiver.(*expression_parser.ThisReceiver)
			if !isImplicitReceiver || isThisReceiver {
				return nil, &ParseError{Message: `"trigger" option of the "viewport" trigger must be an identifier`}
			}

			ref := propertyRead.Name
			reference = &ref

			// Filter out the trigger key and value
			filteredKeys := []expression_parser.LiteralMapKey{}
			filteredValues := []expression_parser.AST{}
			for i, key := range literalMap.Keys {
				if i != triggerIndex {
					filteredKeys = append(filteredKeys, key)
					filteredValues = append(filteredValues, literalMap.Values[i])
				}
			}
			options = expression_parser.NewLiteralMap(literalMap.Span(), literalMap.SourceSpan(), filteredKeys, filteredValues)
		}
	}

	if isHydrationTrigger && reference != nil {
		return nil, &ParseError{Message: `"viewport" hydration trigger cannot have a "trigger"`}
	}

	if options != nil {
		dynamicNode := DynamicAstValidatorFindDynamicNode(options)
		if dynamicNode != nil {
			return nil, &ParseError{
				Message: `Options of the "viewport" trigger must be an object literal containing only literal values, but "` + getTypeName(dynamicNode) + `" was found`,
			}
		}
	}

	return NewViewportDeferredTrigger(reference, options, nameSpan, sourceSpan, prefetchSpan, onSourceSpan, hydrateSpan), nil
}

// validatePlainReferenceBasedTrigger checks whether the structure of a non-hydrate reference-based trigger is valid
func validatePlainReferenceBasedTrigger(triggerType OnTriggerType, parameters []ParsedParameter) error {
	if len(parameters) > 1 {
		return &ParseError{Message: `"` + string(triggerType) + `" trigger can only have zero or one parameters`}
	}
	return nil
}

// validateHydrateReferenceBasedTrigger checks whether the structure of a hydrate trigger is valid
func validateHydrateReferenceBasedTrigger(triggerType OnTriggerType, parameters []ParsedParameter) error {
	if triggerType == OnTriggerTypeViewport {
		if len(parameters) > 1 {
			return &ParseError{Message: `Hydration trigger "` + string(triggerType) + `" cannot have more than one parameter`}
		}
		return nil
	}

	if len(parameters) > 0 {
		return &ParseError{Message: `Hydration trigger "` + string(triggerType) + `" cannot have parameters`}
	}

	return nil
}

// GetTriggerParametersStart gets the index within an expression at which the trigger parameters start
func GetTriggerParametersStart(value string, startPosition int) int {
	hasFoundSeparator := false

	for i := startPosition; i < len(value); i++ {
		if separatorPattern.MatchString(string(value[i])) {
			hasFoundSeparator = true
		} else if hasFoundSeparator {
			return i
		}
	}

	return -1
}

// ParseDeferredTime parses a time expression from a deferred trigger to milliseconds
func ParseDeferredTime(value string) *int {
	match := timePattern.FindStringSubmatch(value)
	if match == nil {
		return nil
	}

	timeStr := match[0]
	var units string
	if len(match) > 1 {
		units = match[1]
	}

	timeValue, err := strconv.ParseFloat(timeStr, 64)
	if err != nil {
		return nil
	}

	var milliseconds int
	if units == "s" {
		milliseconds = int(timeValue * 1000)
	} else {
		milliseconds = int(timeValue)
	}

	return &milliseconds
}

// DynamicAstValidator is a visitor that finds dynamic nodes in an AST
type DynamicAstValidator struct {
	*expression_parser.RecursiveAstVisitor
	dynamicNode expression_parser.AST
}

// NewDynamicAstValidator creates a new DynamicAstValidator
func NewDynamicAstValidator() *DynamicAstValidator {
	return &DynamicAstValidator{
		RecursiveAstVisitor: &expression_parser.RecursiveAstVisitor{},
		dynamicNode:         nil,
	}
}

// DynamicAstValidatorFindDynamicNode finds a dynamic node in an AST
func DynamicAstValidatorFindDynamicNode(ast expression_parser.AST) expression_parser.AST {
	visitor := NewDynamicAstValidator()
	ast.Visit(visitor, nil)
	return visitor.dynamicNode
}

// Visit visits an AST node
func (d *DynamicAstValidator) Visit(ast expression_parser.AST, context interface{}) interface{} {
	_, isASTWithSource := ast.(*expression_parser.ASTWithSource)
	_, isLiteralPrimitive := ast.(*expression_parser.LiteralPrimitive)
	_, isLiteralArray := ast.(*expression_parser.LiteralArray)
	_, isLiteralMap := ast.(*expression_parser.LiteralMap)

	if !isASTWithSource && !isLiteralPrimitive && !isLiteralArray && !isLiteralMap {
		d.dynamicNode = ast
	} else {
		d.RecursiveAstVisitor.Visit(ast, context)
	}
	return nil
}

// getTypeName gets the type name of an AST node
func getTypeName(ast expression_parser.AST) string {
	// Simple type name extraction - in Go we can use reflection or type switch
	switch ast.(type) {
	case *expression_parser.PropertyRead:
		return "PropertyRead"
	case *expression_parser.SafePropertyRead:
		return "SafePropertyRead"
	case *expression_parser.KeyedRead:
		return "KeyedRead"
	case *expression_parser.SafeKeyedRead:
		return "SafeKeyedRead"
	case *expression_parser.Call:
		return "Call"
	case *expression_parser.SafeCall:
		return "SafeCall"
	case *expression_parser.Binary:
		return "Binary"
	case *expression_parser.Unary:
		return "Unary"
	case *expression_parser.Conditional:
		return "Conditional"
	case *expression_parser.BindingPipe:
		return "BindingPipe"
	default:
		return "Unknown"
	}
}
