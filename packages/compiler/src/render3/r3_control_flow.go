package render3

import (
	"ngc-go/packages/compiler/src/expressionparser"
	"ngc-go/packages/compiler/src/ml_parser"
	"ngc-go/packages/compiler/src/util"
	"regexp"
	"strings"
)

// Pattern for the expression in a for loop block
var forLoopExpressionPattern = regexp.MustCompile(`^\s*([0-9A-Za-z_$]*)\s+of\s+([\S\s]*)`)

// Pattern for the tracking expression in a for loop block
var forLoopTrackPattern = regexp.MustCompile(`^track\s+([\S\s]*)`)

// Pattern for the `as` expression in a conditional block
var conditionalAliasPattern = regexp.MustCompile(`^(as\s+)(.*)`)

// Pattern used to identify an `else if` block
var elseIfPattern = regexp.MustCompile(`^else[^\S\r\n]+if`)

// Pattern used to identify a `let` parameter
var forLoopLetPattern = regexp.MustCompile(`^let\s+([\S\s]*)`)

// Pattern used to validate a JavaScript identifier
var identifierPattern = regexp.MustCompile(`^[$A-Z_][0-9A-Z_$]*$`)

// Pattern to group a string into leading whitespace, non whitespace, and trailing whitespace
var charactersInSurroundingWhitespacePattern = regexp.MustCompile(`(\s*)(\S+)(\s*)`)

// Names of variables that are allowed to be used in the `let` expression of a `for` loop
var allowedForLoopLetVariables = map[string]bool{
	"$index": true,
	"$first": true,
	"$last":  true,
	"$even":  true,
	"$odd":   true,
	"$count": true,
}

// BindingParser is an interface for parsing bindings
// TODO: This should be defined in template_parser package
type BindingParser interface {
	ParseBinding(expression string, allowPipes bool, sourceSpan *util.ParseSourceSpan, absoluteOffset int) *expressionparser.ASTWithSource
}

// IsConnectedForLoopBlock determines if a block with a specific name can be connected to a `for` block
func IsConnectedForLoopBlock(name string) bool {
	return name == "empty"
}

// IsConnectedIfLoopBlock determines if a block with a specific name can be connected to an `if` block
func IsConnectedIfLoopBlock(name string) bool {
	return name == "else" || elseIfPattern.MatchString(name)
}

// CreateIfBlockResult represents the result of creating an if block
type CreateIfBlockResult struct {
	Node   *IfBlock
	Errors []*util.ParseError
}

// CreateIfBlock creates an `if` loop block from an HTML AST node
func CreateIfBlock(
	ast *ml_parser.Block,
	connectedBlocks []*ml_parser.Block,
	visitor ml_parser.Visitor,
	bindingParser BindingParser,
) CreateIfBlockResult {
	errors := validateIfConnectedBlocks(connectedBlocks)
	branches := []*IfBlockBranch{}
	mainBlockParams := parseConditionalBlockParameters(ast, &errors, bindingParser)

	if mainBlockParams != nil {
		children := convertToR3Nodes(ml_parser.VisitAll(visitor, ast.Children, nil))
		branches = append(branches, NewIfBlockBranch(
			mainBlockParams.Expression,
			children,
			mainBlockParams.ExpressionAlias,
			ast.SourceSpan(),
			ast.StartSourceSpan,
			ast.EndSourceSpan,
			ast.NameSpan,
			ast.I18n,
		))
	}

	for _, block := range connectedBlocks {
		if elseIfPattern.MatchString(block.Name) {
			params := parseConditionalBlockParameters(block, &errors, bindingParser)
			if params != nil {
				children := convertToR3Nodes(ml_parser.VisitAll(visitor, block.Children, nil))
				branches = append(branches, NewIfBlockBranch(
					params.Expression,
					children,
					params.ExpressionAlias,
					block.SourceSpan(),
					block.StartSourceSpan,
					block.EndSourceSpan,
					block.NameSpan,
					block.I18n,
				))
			}
		} else if block.Name == "else" {
			children := convertToR3Nodes(ml_parser.VisitAll(visitor, block.Children, nil))
			branches = append(branches, NewIfBlockBranch(
				nil,
				children,
				nil,
				block.SourceSpan(),
				block.StartSourceSpan,
				block.EndSourceSpan,
				block.NameSpan,
				block.I18n,
			))
		}
	}

	// The outer IfBlock should have a span that encapsulates all branches
	var ifBlockStartSourceSpan *util.ParseSourceSpan
	var ifBlockEndSourceSpan *util.ParseSourceSpan
	if len(branches) > 0 {
		ifBlockStartSourceSpan = branches[0].StartSourceSpan
		ifBlockEndSourceSpan = branches[len(branches)-1].EndSourceSpan
	} else {
		ifBlockStartSourceSpan = ast.StartSourceSpan
		ifBlockEndSourceSpan = ast.EndSourceSpan
	}

	wholeSourceSpan := ast.SourceSpan()
	if len(branches) > 0 {
		lastBranch := branches[len(branches)-1]
		wholeSourceSpan = util.NewParseSourceSpan(ifBlockStartSourceSpan.Start, lastBranch.SourceSpan().End, nil, nil)
	}

	return CreateIfBlockResult{
		Node:   NewIfBlock(branches, wholeSourceSpan, ast.StartSourceSpan, ifBlockEndSourceSpan, ast.NameSpan),
		Errors: errors,
	}
}

// CreateForLoopResult represents the result of creating a for loop block
type CreateForLoopResult struct {
	Node   *ForLoopBlock
	Errors []*util.ParseError
}

// CreateForLoop creates a `for` loop block from an HTML AST node
func CreateForLoop(
	ast *ml_parser.Block,
	connectedBlocks []*ml_parser.Block,
	visitor ml_parser.Visitor,
	bindingParser BindingParser,
) CreateForLoopResult {
	errors := []*util.ParseError{}
	params := parseForLoopParameters(ast, &errors, bindingParser)
	var node *ForLoopBlock
	var empty *ForLoopBlockEmpty

	for _, block := range connectedBlocks {
		if block.Name == "empty" {
			if empty != nil {
				errors = append(errors, util.NewParseError(block.SourceSpan(), "@for loop can only have one @empty block"))
			} else if len(block.Parameters) > 0 {
				errors = append(errors, util.NewParseError(block.SourceSpan(), "@empty block cannot have parameters"))
			} else {
				children := convertToR3Nodes(ml_parser.VisitAll(visitor, block.Children, nil))
				empty = NewForLoopBlockEmpty(
					children,
					block.SourceSpan(),
					block.StartSourceSpan,
					block.EndSourceSpan,
					block.NameSpan,
					block.I18n,
				)
			}
		} else {
			errors = append(errors, util.NewParseError(block.SourceSpan(), `Unrecognized @for loop block "`+block.Name+`"`))
		}
	}

	if params != nil {
		if params.TrackBy == nil {
			errors = append(errors, util.NewParseError(ast.StartSourceSpan, `@for loop must have a "track" expression`))
		} else {
			// The `for` block has a main span that includes the `empty` branch
			var endSpan *util.ParseSourceSpan
			if empty != nil {
				endSpan = empty.EndSourceSpan
			} else {
				endSpan = ast.EndSourceSpan
			}
			sourceSpan := util.NewParseSourceSpan(ast.SourceSpan().Start, endSpan.End, nil, nil)
			validateTrackByExpression(params.TrackBy.Expression, params.TrackBy.KeywordSpan, &errors)
			children := convertToR3Nodes(ml_parser.VisitAll(visitor, ast.Children, nil))
			node = NewForLoopBlock(
				params.ItemName,
				params.Expression,
				params.TrackBy.Expression,
				params.TrackBy.KeywordSpan,
				params.Context,
				children,
				empty,
				ast.SourceSpan(),
				sourceSpan,
				ast.StartSourceSpan,
				endSpan,
				ast.NameSpan,
				ast.I18n,
			)
		}
	}

	return CreateForLoopResult{Node: node, Errors: errors}
}

// CreateSwitchBlockResult represents the result of creating a switch block
type CreateSwitchBlockResult struct {
	Node   *SwitchBlock
	Errors []*util.ParseError
}

// CreateSwitchBlock creates a switch block from an HTML AST node
func CreateSwitchBlock(
	ast *ml_parser.Block,
	visitor ml_parser.Visitor,
	bindingParser BindingParser,
) CreateSwitchBlockResult {
	errors := validateSwitchBlock(ast)
	var primaryExpression expressionparser.AST
	if len(ast.Parameters) > 0 {
		primaryExpression = parseBlockParameterToBinding(ast.Parameters[0], bindingParser, nil).AST
	} else {
		primaryExpression = bindingParser.ParseBinding("", false, ast.SourceSpan(), 0).AST
	}
	cases := []*SwitchBlockCase{}
	unknownBlocks := []*UnknownBlock{}
	var defaultCase *SwitchBlockCase

	// Here we assume that all the blocks are valid given that we validated them above
	for _, node := range ast.Children {
		block, ok := node.(*ml_parser.Block)
		if !ok {
			continue
		}

		if (block.Name != "case" || len(block.Parameters) == 0) && block.Name != "default" {
			unknownBlocks = append(unknownBlocks, NewUnknownBlock(block.Name, block.SourceSpan(), block.NameSpan))
			continue
		}

		var expr expressionparser.AST
		if block.Name == "case" {
			expr = parseBlockParameterToBinding(block.Parameters[0], bindingParser, nil).AST
		} else {
			expr = nil
		}
		children := convertToR3Nodes(ml_parser.VisitAll(visitor, block.Children, nil))
		astCase := NewSwitchBlockCase(
			expr,
			children,
			block.SourceSpan(),
			block.StartSourceSpan,
			block.EndSourceSpan,
			block.NameSpan,
			block.I18n,
		)

		if expr == nil {
			defaultCase = astCase
		} else {
			cases = append(cases, astCase)
		}
	}

	// Ensure that the default case is last in the array
	if defaultCase != nil {
		cases = append(cases, defaultCase)
	}

	return CreateSwitchBlockResult{
		Node: NewSwitchBlock(
			primaryExpression,
			cases,
			unknownBlocks,
			ast.SourceSpan(),
			ast.StartSourceSpan,
			ast.EndSourceSpan,
			ast.NameSpan,
		),
		Errors: errors,
	}
}

// ForLoopParameters represents parsed parameters for a for loop
type ForLoopParameters struct {
	ItemName   *Variable
	TrackBy    *TrackByExpression
	Expression *expressionparser.ASTWithSource
	Context    []*Variable
}

// TrackByExpression represents a track by expression
type TrackByExpression struct {
	Expression  *expressionparser.ASTWithSource
	KeywordSpan *util.ParseSourceSpan
}

// ConditionalBlockParameters represents parsed parameters for a conditional block
type ConditionalBlockParameters struct {
	Expression      expressionparser.AST
	ExpressionAlias *Variable
}

// parseForLoopParameters parses the parameters of a `for` loop block
func parseForLoopParameters(
	block *ml_parser.Block,
	errors *[]*util.ParseError,
	bindingParser BindingParser,
) *ForLoopParameters {
	if len(block.Parameters) == 0 {
		*errors = append(*errors, util.NewParseError(block.StartSourceSpan, "@for loop does not have an expression"))
		return nil
	}

	expressionParam := block.Parameters[0]
	secondaryParams := block.Parameters[1:]
	expression := stripOptionalParentheses(expressionParam, errors)
	if expression == nil {
		return nil
	}

	match := forLoopExpressionPattern.FindStringSubmatch(*expression)
	if match == nil || len(match) < 3 || strings.TrimSpace(match[2]) == "" {
		*errors = append(*errors, util.NewParseError(
			expressionParam.SourceSpan(),
			"Cannot parse expression. @for loop expression must match the pattern \"<identifier> of <expression>\"",
		))
		return nil
	}

	itemName := match[1]
	rawExpression := match[2]
	if allowedForLoopLetVariables[itemName] {
		var allowedVars []string
		for k := range allowedForLoopLetVariables {
			allowedVars = append(allowedVars, k)
		}
		*errors = append(*errors, util.NewParseError(
			expressionParam.SourceSpan(),
			"@for loop item name cannot be one of "+strings.Join(allowedVars, ", ")+".",
		))
	}

	// `expressionParam.expression` contains the variable declaration and the expression
	variableName := strings.Split(expressionParam.Expression, " ")[0]
	variableSpan := util.NewParseSourceSpan(
		expressionParam.SourceSpan().Start,
		expressionParam.SourceSpan().Start.MoveBy(len(variableName)),
		nil,
		nil,
	)
	result := &ForLoopParameters{
		ItemName:   NewVariable(itemName, "$implicit", variableSpan, variableSpan, nil),
		TrackBy:    nil,
		Expression: parseBlockParameterToBinding(expressionParam, bindingParser, &rawExpression),
		Context:    []*Variable{},
	}

	// Add ambiently-available context variables
	for variableName := range allowedForLoopLetVariables {
		emptySpanAfterForBlockStart := util.NewParseSourceSpan(
			block.StartSourceSpan.End,
			block.StartSourceSpan.End,
			nil,
			nil,
		)
		result.Context = append(result.Context, NewVariable(
			variableName,
			variableName,
			emptySpanAfterForBlockStart,
			emptySpanAfterForBlockStart,
			nil,
		))
	}

	for _, param := range secondaryParams {
		letMatch := forLoopLetPattern.FindStringSubmatch(param.Expression)
		if letMatch != nil && len(letMatch) > 1 {
			variablesSpan := util.NewParseSourceSpan(
				param.SourceSpan().Start.MoveBy(len(letMatch[0])-len(letMatch[1])),
				param.SourceSpan().End,
				nil,
				nil,
			)
			parseLetParameter(
				param.SourceSpan(),
				letMatch[1],
				variablesSpan,
				itemName,
				&result.Context,
				errors,
			)
			continue
		}

		trackMatch := forLoopTrackPattern.FindStringSubmatch(param.Expression)
		if trackMatch != nil && len(trackMatch) > 1 {
			if result.TrackBy != nil {
				*errors = append(*errors, util.NewParseError(param.SourceSpan(), `@for loop can only have one "track" expression`))
			} else {
				expr := parseBlockParameterToBinding(param, bindingParser, &trackMatch[1])
				if _, ok := expr.AST.(*expressionparser.EmptyExpr); ok {
					*errors = append(*errors, util.NewParseError(block.StartSourceSpan, `@for loop must have a "track" expression`))
				}
				keywordSpan := util.NewParseSourceSpan(
					param.SourceSpan().Start,
					param.SourceSpan().Start.MoveBy(5), // "track".length
					nil,
					nil,
				)
				result.TrackBy = &TrackByExpression{
					Expression:  expr,
					KeywordSpan: keywordSpan,
				}
			}
			continue
		}

		*errors = append(*errors, util.NewParseError(param.SourceSpan(), `Unrecognized @for loop parameter "`+param.Expression+`"`))
	}

	return result
}

// validateTrackByExpression validates a track by expression
func validateTrackByExpression(
	expr *expressionparser.ASTWithSource,
	parseSourceSpan *util.ParseSourceSpan,
	errors *[]*util.ParseError,
) {
	visitor := NewPipeVisitor()
	expr.AST.Visit(visitor, nil)
	if visitor.HasPipe {
		*errors = append(*errors, util.NewParseError(parseSourceSpan, "Cannot use pipes in track expressions"))
	}
}

// parseLetParameter parses the `let` parameter of a `for` loop block
func parseLetParameter(
	sourceSpan *util.ParseSourceSpan,
	expression string,
	span *util.ParseSourceSpan,
	loopItemName string,
	context *[]*Variable,
	errors *[]*util.ParseError,
) {
	parts := strings.Split(expression, ",")
	startSpan := span.Start
	for _, part := range parts {
		expressionParts := strings.Split(part, "=")
		var name string
		var variableName string
		if len(expressionParts) == 2 {
			name = strings.TrimSpace(expressionParts[0])
			variableName = strings.TrimSpace(expressionParts[1])
		}

		if len(name) == 0 || len(variableName) == 0 {
			*errors = append(*errors, util.NewParseError(
				sourceSpan,
				`Invalid @for loop "let" parameter. Parameter should match the pattern "<name> = <variable name>"`,
			))
		} else if !allowedForLoopLetVariables[variableName] {
			var allowedVars []string
			for k := range allowedForLoopLetVariables {
				allowedVars = append(allowedVars, k)
			}
			*errors = append(*errors, util.NewParseError(
				sourceSpan,
				`Unknown "let" parameter variable "`+variableName+`". The allowed variables are: `+strings.Join(allowedVars, ", "),
			))
		} else if name == loopItemName {
			*errors = append(*errors, util.NewParseError(
				sourceSpan,
				`Invalid @for loop "let" parameter. Variable cannot be called "`+loopItemName+`"`,
			))
		} else {
			// Check for duplicates
			hasDuplicate := false
			for _, v := range *context {
				if v.Name == name {
					hasDuplicate = true
					break
				}
			}
			if hasDuplicate {
				*errors = append(*errors, util.NewParseError(sourceSpan, `Duplicate "let" parameter variable "`+variableName+`"`))
			} else {
				var keySpan *util.ParseSourceSpan
				if len(expressionParts) == 2 {
					keyMatch := charactersInSurroundingWhitespacePattern.FindStringSubmatch(expressionParts[0])
					if keyMatch != nil && len(keyMatch) >= 3 {
						keyLeadingWhitespace := keyMatch[1]
						keyName := keyMatch[2]
						keySpan = util.NewParseSourceSpan(
							startSpan.MoveBy(len(keyLeadingWhitespace)),
							startSpan.MoveBy(len(keyLeadingWhitespace)+len(keyName)),
							nil,
							nil,
						)
					} else {
						keySpan = span
					}
				} else {
					keySpan = span
				}

				var valueSpan *util.ParseSourceSpan
				if len(expressionParts) == 2 {
					valueMatch := charactersInSurroundingWhitespacePattern.FindStringSubmatch(expressionParts[1])
					if valueMatch != nil && len(valueMatch) >= 3 {
						valueLeadingWhitespace := valueMatch[1]
						implicit := valueMatch[2]
						valueSpan = util.NewParseSourceSpan(
							startSpan.MoveBy(len(expressionParts[0])+1+len(valueLeadingWhitespace)),
							startSpan.MoveBy(len(expressionParts[0])+1+len(valueLeadingWhitespace)+len(implicit)),
							nil,
							nil,
						)
					}
				}
				var finalSourceSpan *util.ParseSourceSpan
				if valueSpan != nil {
					finalSourceSpan = util.NewParseSourceSpan(keySpan.Start, valueSpan.End, nil, nil)
				} else {
					finalSourceSpan = util.NewParseSourceSpan(keySpan.Start, keySpan.End, nil, nil)
				}
				*context = append(*context, NewVariable(name, variableName, finalSourceSpan, keySpan, valueSpan))
			}
		}
		startSpan = startSpan.MoveBy(len(part) + 1) // add 1 to move past the comma
	}
}

// validateIfConnectedBlocks checks that the shape of the blocks connected to an `@if` block is correct
func validateIfConnectedBlocks(connectedBlocks []*ml_parser.Block) []*util.ParseError {
	errors := []*util.ParseError{}
	hasElse := false

	for i, block := range connectedBlocks {
		if block.Name == "else" {
			if hasElse {
				errors = append(errors, util.NewParseError(block.StartSourceSpan, "Conditional can only have one @else block"))
			} else if len(connectedBlocks) > 1 && i < len(connectedBlocks)-1 {
				errors = append(errors, util.NewParseError(block.StartSourceSpan, "@else block must be last inside the conditional"))
			} else if len(block.Parameters) > 0 {
				errors = append(errors, util.NewParseError(block.StartSourceSpan, "@else block cannot have parameters"))
			}
			hasElse = true
		} else if !elseIfPattern.MatchString(block.Name) {
			errors = append(errors, util.NewParseError(block.StartSourceSpan, `Unrecognized conditional block @`+block.Name))
		}
	}

	return errors
}

// validateSwitchBlock checks that the shape of a `switch` block is valid
func validateSwitchBlock(ast *ml_parser.Block) []*util.ParseError {
	errors := []*util.ParseError{}
	hasDefault := false

	if len(ast.Parameters) != 1 {
		errors = append(errors, util.NewParseError(ast.StartSourceSpan, "@switch block must have exactly one parameter"))
		return errors
	}

	for _, node := range ast.Children {
		// Skip over comments and empty text nodes inside the switch block
		if _, ok := node.(*ml_parser.Comment); ok {
			continue
		}
		if text, ok := node.(*ml_parser.Text); ok {
			if strings.TrimSpace(text.Value) == "" {
				continue
			}
		}

		block, ok := node.(*ml_parser.Block)
		if !ok || (block.Name != "case" && block.Name != "default") {
			errors = append(errors, util.NewParseError(node.SourceSpan(), "@switch block can only contain @case and @default blocks"))
			continue
		}

		if block.Name == "default" {
			if hasDefault {
				errors = append(errors, util.NewParseError(block.StartSourceSpan, "@switch block can only have one @default block"))
			} else if len(block.Parameters) > 0 {
				errors = append(errors, util.NewParseError(block.StartSourceSpan, "@default block cannot have parameters"))
			}
			hasDefault = true
		} else if block.Name == "case" && len(block.Parameters) != 1 {
			errors = append(errors, util.NewParseError(block.StartSourceSpan, "@case block must have exactly one parameter"))
		}
	}

	return errors
}

// parseBlockParameterToBinding parses a block parameter into a binding AST
func parseBlockParameterToBinding(
	ast *ml_parser.BlockParameter,
	bindingParser BindingParser,
	part *string,
) *expressionparser.ASTWithSource {
	var start int
	var end int

	if part != nil {
		// Note: `lastIndexOf` here should be enough to know the start index of the expression
		start = strings.LastIndex(ast.Expression, *part)
		if start < 0 {
			start = 0
		}
		end = start + len(*part)
	} else {
		start = 0
		end = len(ast.Expression)
	}

	return bindingParser.ParseBinding(
		ast.Expression[start:end],
		false,
		ast.SourceSpan(),
		ast.SourceSpan().Start.Offset+start,
	)
}

// parseConditionalBlockParameters parses the parameter of a conditional block (`if` or `else if`)
func parseConditionalBlockParameters(
	block *ml_parser.Block,
	errors *[]*util.ParseError,
	bindingParser BindingParser,
) *ConditionalBlockParameters {
	if len(block.Parameters) == 0 {
		*errors = append(*errors, util.NewParseError(block.StartSourceSpan, "Conditional block does not have an expression"))
		return nil
	}

	expression := parseBlockParameterToBinding(block.Parameters[0], bindingParser, nil)
	var expressionAlias *Variable

	// Start from 1 since we processed the first parameter already
	for i := 1; i < len(block.Parameters); i++ {
		param := block.Parameters[i]
		aliasMatch := conditionalAliasPattern.FindStringSubmatch(param.Expression)

		// For now conditionals can only have an `as` parameter
		if aliasMatch == nil || len(aliasMatch) < 3 {
			*errors = append(*errors, util.NewParseError(
				param.SourceSpan(),
				`Unrecognized conditional parameter "`+param.Expression+`"`,
			))
		} else if block.Name != "if" && !elseIfPattern.MatchString(block.Name) {
			*errors = append(*errors, util.NewParseError(
				param.SourceSpan(),
				`"as" expression is only allowed on @if and @else if blocks`,
			))
		} else if expressionAlias != nil {
			*errors = append(*errors, util.NewParseError(param.SourceSpan(), `Conditional can only have one "as" expression`))
		} else {
			name := strings.TrimSpace(aliasMatch[2])
			if identifierPattern.MatchString(name) {
				variableStart := param.SourceSpan().Start.MoveBy(len(aliasMatch[1]))
				variableSpan := util.NewParseSourceSpan(variableStart, variableStart.MoveBy(len(name)), nil, nil)
				expressionAlias = NewVariable(name, name, variableSpan, variableSpan, nil)
			} else {
				*errors = append(*errors, util.NewParseError(param.SourceSpan(), `"as" expression must be a valid JavaScript identifier`))
			}
		}
	}

	return &ConditionalBlockParameters{
		Expression:      expression.AST,
		ExpressionAlias: expressionAlias,
	}
}

// stripOptionalParentheses strips optional parentheses around from a control from expression parameter
func stripOptionalParentheses(param *ml_parser.BlockParameter, errors *[]*util.ParseError) *string {
	expression := param.Expression
	spaceRegex := regexp.MustCompile(`^\s$`)
	openParens := 0
	start := 0
	end := len(expression) - 1

	for i := 0; i < len(expression); i++ {
		char := expression[i]
		if char == '(' {
			start = i + 1
			openParens++
		} else if spaceRegex.MatchString(string(char)) {
			continue
		} else {
			break
		}
	}

	if openParens == 0 {
		return &expression
	}

	for i := len(expression) - 1; i >= 0; i-- {
		char := expression[i]
		if char == ')' {
			end = i
			openParens--
			if openParens == 0 {
				break
			}
		} else if spaceRegex.MatchString(string(char)) {
			continue
		} else {
			break
		}
	}

	if openParens != 0 {
		*errors = append(*errors, util.NewParseError(param.SourceSpan(), "Unclosed parentheses in expression"))
		return nil
	}

	result := expression[start:end]
	return &result
}

// PipeVisitor is a visitor that checks if an expression contains pipes
type PipeVisitor struct {
	*expressionparser.RecursiveAstVisitor
	HasPipe bool
}

// NewPipeVisitor creates a new PipeVisitor
func NewPipeVisitor() *PipeVisitor {
	return &PipeVisitor{
		RecursiveAstVisitor: &expressionparser.RecursiveAstVisitor{},
		HasPipe:             false,
	}
}

// VisitPipe visits a binding pipe
func (p *PipeVisitor) VisitPipe(ast *expressionparser.BindingPipe, context interface{}) interface{} {
	p.HasPipe = true
	return nil
}

// convertToR3Nodes converts a slice of ml_parser.Node results to []Node (R3 AST)
func convertToR3Nodes(results []interface{}) []Node {
	nodes := []Node{}
	for _, result := range results {
		if node, ok := result.(Node); ok {
			nodes = append(nodes, node)
		}
	}
	return nodes
}
