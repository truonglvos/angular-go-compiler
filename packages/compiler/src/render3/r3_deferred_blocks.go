package render3

import (
	"ngc-go/packages/compiler/src/ml_parser"
	"ngc-go/packages/compiler/src/util"
	"regexp"
)

// Pattern to identify a `prefetch when` trigger
var prefetchWhenPattern = regexp.MustCompile(`^prefetch\s+when\s`)

// Pattern to identify a `prefetch on` trigger
var prefetchOnPattern = regexp.MustCompile(`^prefetch\s+on\s`)

// Pattern to identify a `hydrate when` trigger
var hydrateWhenPattern = regexp.MustCompile(`^hydrate\s+when\s`)

// Pattern to identify a `hydrate on` trigger
var hydrateOnPattern = regexp.MustCompile(`^hydrate\s+on\s`)

// Pattern to identify a `hydrate never` trigger
var hydrateNeverPattern = regexp.MustCompile(`^hydrate\s+never(\s*)$`)

// Pattern to identify a `minimum` parameter in a block
var minimumParameterPattern = regexp.MustCompile(`^minimum\s`)

// Pattern to identify an `after` parameter in a block
var afterParameterPattern = regexp.MustCompile(`^after\s`)

// Pattern to identify a `when` parameter in a block
var whenParameterPattern = regexp.MustCompile(`^when\s`)

// Pattern to identify an `on` parameter in a block
var onParameterPattern = regexp.MustCompile(`^on\s`)

// IsConnectedDeferLoopBlock determines if a block with a specific name can be connected to a `defer` block
func IsConnectedDeferLoopBlock(name string) bool {
	return name == "placeholder" || name == "loading" || name == "error"
}

// CreateDeferredBlockResult represents the result of creating a deferred block
type CreateDeferredBlockResult struct {
	Node   *DeferredBlock
	Errors []*util.ParseError
}

// CreateDeferredBlock creates a deferred block from an HTML AST node
func CreateDeferredBlock(
	ast *ml_parser.Block,
	connectedBlocks []*ml_parser.Block,
	visitor ml_parser.Visitor,
	bindingParser BindingParser,
) CreateDeferredBlockResult {
	errors := []*util.ParseError{}
	placeholder, loading, errorBlock := parseConnectedBlocks(connectedBlocks, &errors, visitor)
	triggers, prefetchTriggers, hydrateTriggers := parsePrimaryTriggers(
		ast,
		bindingParser,
		&errors,
		placeholder,
	)

	// The `defer` block has a main span encompassing all of the connected branches as well
	lastEndSourceSpan := ast.EndSourceSpan
	endOfLastSourceSpan := ast.SourceSpan().End
	if len(connectedBlocks) > 0 {
		lastConnectedBlock := connectedBlocks[len(connectedBlocks)-1]
		lastEndSourceSpan = lastConnectedBlock.EndSourceSpan
		endOfLastSourceSpan = lastConnectedBlock.SourceSpan().End
	}

	sourceSpanWithConnectedBlocks := util.NewParseSourceSpan(
		ast.SourceSpan().Start,
		endOfLastSourceSpan,
		nil,
		nil,
	)

	children := convertToR3Nodes(ml_parser.VisitAll(visitor, ast.Children, nil))
	node := NewDeferredBlock(
		children,
		triggers,
		prefetchTriggers,
		hydrateTriggers,
		placeholder,
		loading,
		errorBlock,
		ast.NameSpan,
		sourceSpanWithConnectedBlocks,
		ast.SourceSpan(),
		ast.StartSourceSpan,
		lastEndSourceSpan,
		ast.I18n,
	)

	return CreateDeferredBlockResult{Node: node, Errors: errors}
}

// parseConnectedBlocks parses connected blocks (placeholder, loading, error)
func parseConnectedBlocks(
	connectedBlocks []*ml_parser.Block,
	errors *[]*util.ParseError,
	visitor ml_parser.Visitor,
) (*DeferredBlockPlaceholder, *DeferredBlockLoading, *DeferredBlockError) {
	var placeholder *DeferredBlockPlaceholder
	var loading *DeferredBlockLoading
	var errorBlock *DeferredBlockError

	for _, block := range connectedBlocks {
		func() {
			defer func() {
				if r := recover(); r != nil {
					if err, ok := r.(error); ok {
						*errors = append(*errors, util.NewParseError(block.StartSourceSpan, err.Error()))
					} else {
						*errors = append(*errors, util.NewParseError(block.StartSourceSpan, "Unknown error"))
					}
				}
			}()

			if !IsConnectedDeferLoopBlock(block.Name) {
				*errors = append(*errors, util.NewParseError(block.StartSourceSpan, `Unrecognized block "@`+block.Name+`"`))
				return
			}

			switch block.Name {
			case "placeholder":
				if placeholder != nil {
					*errors = append(*errors, util.NewParseError(
						block.StartSourceSpan,
						"@defer block can only have one @placeholder block",
					))
				} else {
					placeholder = parsePlaceholderBlock(block, visitor)
				}

			case "loading":
				if loading != nil {
					*errors = append(*errors, util.NewParseError(
						block.StartSourceSpan,
						"@defer block can only have one @loading block",
					))
				} else {
					loading = parseLoadingBlock(block, visitor)
				}

			case "error":
				if errorBlock != nil {
					*errors = append(*errors, util.NewParseError(block.StartSourceSpan, "@defer block can only have one @error block"))
				} else {
					errorBlock = parseErrorBlock(block, visitor)
				}
			}
		}()
	}

	return placeholder, loading, errorBlock
}

// parsePlaceholderBlock parses a placeholder block
func parsePlaceholderBlock(ast *ml_parser.Block, visitor ml_parser.Visitor) *DeferredBlockPlaceholder {
	var minimumTime *int

	for _, param := range ast.Parameters {
		if minimumParameterPattern.MatchString(param.Expression) {
			if minimumTime != nil {
				panic(&ParseError{Message: `@placeholder block can only have one "minimum" parameter`})
			}

			parsedTime := ParseDeferredTime(
				param.Expression[GetTriggerParametersStart(param.Expression, 0):],
			)

			if parsedTime == nil {
				panic(&ParseError{Message: `Could not parse time value of parameter "minimum"`})
			}

			minimumTime = parsedTime
		} else {
			panic(&ParseError{Message: `Unrecognized parameter in @placeholder block: "` + param.Expression + `"`})
		}
	}

	children := convertToR3Nodes(ml_parser.VisitAll(visitor, ast.Children, nil))
	return NewDeferredBlockPlaceholder(
		children,
		minimumTime,
		ast.NameSpan,
		ast.SourceSpan(),
		ast.StartSourceSpan,
		ast.EndSourceSpan,
		ast.I18n,
	)
}

// parseLoadingBlock parses a loading block
func parseLoadingBlock(ast *ml_parser.Block, visitor ml_parser.Visitor) *DeferredBlockLoading {
	var afterTime *int
	var minimumTime *int

	for _, param := range ast.Parameters {
		if afterParameterPattern.MatchString(param.Expression) {
			if afterTime != nil {
				panic(&ParseError{Message: `@loading block can only have one "after" parameter`})
			}

			parsedTime := ParseDeferredTime(
				param.Expression[GetTriggerParametersStart(param.Expression, 0):],
			)

			if parsedTime == nil {
				panic(&ParseError{Message: `Could not parse time value of parameter "after"`})
			}

			afterTime = parsedTime
		} else if minimumParameterPattern.MatchString(param.Expression) {
			if minimumTime != nil {
				panic(&ParseError{Message: `@loading block can only have one "minimum" parameter`})
			}

			parsedTime := ParseDeferredTime(
				param.Expression[GetTriggerParametersStart(param.Expression, 0):],
			)

			if parsedTime == nil {
				panic(&ParseError{Message: `Could not parse time value of parameter "minimum"`})
			}

			minimumTime = parsedTime
		} else {
			panic(&ParseError{Message: `Unrecognized parameter in @loading block: "` + param.Expression + `"`})
		}
	}

	children := convertToR3Nodes(ml_parser.VisitAll(visitor, ast.Children, nil))
	return NewDeferredBlockLoading(
		children,
		afterTime,
		minimumTime,
		ast.NameSpan,
		ast.SourceSpan(),
		ast.StartSourceSpan,
		ast.EndSourceSpan,
		ast.I18n,
	)
}

// parseErrorBlock parses an error block
func parseErrorBlock(ast *ml_parser.Block, visitor ml_parser.Visitor) *DeferredBlockError {
	if len(ast.Parameters) > 0 {
		panic(&ParseError{Message: `@error block cannot have parameters`})
	}

	children := convertToR3Nodes(ml_parser.VisitAll(visitor, ast.Children, nil))
	return NewDeferredBlockError(
		children,
		ast.NameSpan,
		ast.SourceSpan(),
		ast.StartSourceSpan,
		ast.EndSourceSpan,
		ast.I18n,
	)
}

// parsePrimaryTriggers parses primary triggers from the block parameters
func parsePrimaryTriggers(
	ast *ml_parser.Block,
	bindingParser BindingParser,
	errors *[]*util.ParseError,
	placeholder *DeferredBlockPlaceholder,
) (*DeferredBlockTriggers, *DeferredBlockTriggers, *DeferredBlockTriggers) {
	triggers := &DeferredBlockTriggers{}
	prefetchTriggers := &DeferredBlockTriggers{}
	hydrateTriggers := &DeferredBlockTriggers{}

	for _, param := range ast.Parameters {
		// The lexer ignores the leading spaces so we can assume
		// that the expression starts with a keyword
		if whenParameterPattern.MatchString(param.Expression) {
			ParseWhenTrigger(param, bindingParser, triggers, errors)
		} else if onParameterPattern.MatchString(param.Expression) {
			ParseOnTrigger(param, bindingParser, triggers, errors, placeholder)
		} else if prefetchWhenPattern.MatchString(param.Expression) {
			ParseWhenTrigger(param, bindingParser, prefetchTriggers, errors)
		} else if prefetchOnPattern.MatchString(param.Expression) {
			ParseOnTrigger(param, bindingParser, prefetchTriggers, errors, placeholder)
		} else if hydrateWhenPattern.MatchString(param.Expression) {
			ParseWhenTrigger(param, bindingParser, hydrateTriggers, errors)
		} else if hydrateOnPattern.MatchString(param.Expression) {
			ParseOnTrigger(param, bindingParser, hydrateTriggers, errors, placeholder)
		} else if hydrateNeverPattern.MatchString(param.Expression) {
			ParseNeverTrigger(param, hydrateTriggers, errors)
		} else {
			*errors = append(*errors, util.NewParseError(param.SourceSpan(), "Unrecognized trigger"))
		}
	}

	// Check if hydrate never is present with other hydrate triggers
	if hydrateTriggers.Never != nil {
		hasOtherTriggers := hydrateTriggers.When != nil ||
			hydrateTriggers.Idle != nil ||
			hydrateTriggers.Immediate != nil ||
			hydrateTriggers.Hover != nil ||
			hydrateTriggers.Timer != nil ||
			hydrateTriggers.Interaction != nil ||
			hydrateTriggers.Viewport != nil

		if hasOtherTriggers {
			*errors = append(*errors, util.NewParseError(
				ast.StartSourceSpan,
				"Cannot specify additional `hydrate` triggers if `hydrate never` is present",
			))
		}
	}

	return triggers, prefetchTriggers, hydrateTriggers
}
