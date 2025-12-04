package util

import (
	"ngc-go/packages/compiler/src/expression_parser"
	"ngc-go/packages/compiler/src/render3"
	"ngc-go/packages/compiler/test/expression_parser/utils"
	"reflect"
)

// HumanizedExpressionSource represents a tuple of [unparsed AST, AST source span]
type HumanizedExpressionSource struct {
	Unparsed string
	Span     *expression_parser.AbsoluteSourceSpan
}

// HumanizeExpressionSource humanizes expression AST source spans in a template
func HumanizeExpressionSource(templateAsts []render3.Node) []HumanizedExpressionSource {
	humanizer := NewExpressionSourceHumanizer()
	// Create a wrapper to implement render3.Visitor interface
	wrapper := &render3VisitorWrapper{humanizer: humanizer}
	render3.VisitAll(wrapper, templateAsts)
	return humanizer.result
}

// render3VisitorWrapper wraps ExpressionSourceHumanizer to implement render3.Visitor
// This is needed because Go doesn't allow method overloading - we need both
// Visit(ast AST, context interface{}) for AstVisitor and Visit(node Node) for render3.Visitor
type render3VisitorWrapper struct {
	humanizer *ExpressionSourceHumanizer
}

// Visit implements render3.Visitor interface
func (w *render3VisitorWrapper) Visit(node render3.Node) interface{} {
	return node.Visit(w)
}

// Delegate all render3.Visitor methods to humanizer
func (w *render3VisitorWrapper) VisitElement(element *render3.Element) interface{} {
	return w.humanizer.VisitElement(element)
}
func (w *render3VisitorWrapper) VisitTemplate(template *render3.Template) interface{} {
	return w.humanizer.VisitTemplate(template)
}
func (w *render3VisitorWrapper) VisitContent(content *render3.Content) interface{} {
	return w.humanizer.VisitContent(content)
}
func (w *render3VisitorWrapper) VisitVariable(variable *render3.Variable) interface{} {
	return w.humanizer.VisitVariable(variable)
}
func (w *render3VisitorWrapper) VisitReference(reference *render3.Reference) interface{} {
	return w.humanizer.VisitReference(reference)
}
func (w *render3VisitorWrapper) VisitTextAttribute(attribute *render3.TextAttribute) interface{} {
	return w.humanizer.VisitTextAttribute(attribute)
}
func (w *render3VisitorWrapper) VisitBoundAttribute(attribute *render3.BoundAttribute) interface{} {
	return w.humanizer.VisitBoundAttribute(attribute)
}
func (w *render3VisitorWrapper) VisitBoundEvent(event *render3.BoundEvent) interface{} {
	return w.humanizer.VisitBoundEvent(event)
}
func (w *render3VisitorWrapper) VisitText(text *render3.Text) interface{} {
	return w.humanizer.VisitText(text)
}
func (w *render3VisitorWrapper) VisitBoundText(text *render3.BoundText) interface{} {
	return w.humanizer.VisitBoundText(text)
}
func (w *render3VisitorWrapper) VisitIcu(icu *render3.Icu) interface{} {
	return w.humanizer.VisitIcu(icu)
}
func (w *render3VisitorWrapper) VisitDeferredBlock(deferred *render3.DeferredBlock) interface{} {
	return w.humanizer.VisitDeferredBlock(deferred)
}
func (w *render3VisitorWrapper) VisitDeferredBlockPlaceholder(block *render3.DeferredBlockPlaceholder) interface{} {
	return w.humanizer.VisitDeferredBlockPlaceholder(block)
}
func (w *render3VisitorWrapper) VisitDeferredBlockError(block *render3.DeferredBlockError) interface{} {
	return w.humanizer.VisitDeferredBlockError(block)
}
func (w *render3VisitorWrapper) VisitDeferredBlockLoading(block *render3.DeferredBlockLoading) interface{} {
	return w.humanizer.VisitDeferredBlockLoading(block)
}
func (w *render3VisitorWrapper) VisitDeferredTrigger(trigger *render3.DeferredTrigger) interface{} {
	return w.humanizer.VisitDeferredTrigger(trigger)
}
func (w *render3VisitorWrapper) VisitSwitchBlock(block *render3.SwitchBlock) interface{} {
	return w.humanizer.VisitSwitchBlock(block)
}
func (w *render3VisitorWrapper) VisitSwitchBlockCase(block *render3.SwitchBlockCase) interface{} {
	return w.humanizer.VisitSwitchBlockCase(block)
}
func (w *render3VisitorWrapper) VisitForLoopBlock(block *render3.ForLoopBlock) interface{} {
	return w.humanizer.VisitForLoopBlock(block)
}
func (w *render3VisitorWrapper) VisitForLoopBlockEmpty(block *render3.ForLoopBlockEmpty) interface{} {
	return w.humanizer.VisitForLoopBlockEmpty(block)
}
func (w *render3VisitorWrapper) VisitIfBlock(block *render3.IfBlock) interface{} {
	return w.humanizer.VisitIfBlock(block)
}
func (w *render3VisitorWrapper) VisitIfBlockBranch(block *render3.IfBlockBranch) interface{} {
	return w.humanizer.VisitIfBlockBranch(block)
}
func (w *render3VisitorWrapper) VisitUnknownBlock(block *render3.UnknownBlock) interface{} {
	return w.humanizer.VisitUnknownBlock(block)
}
func (w *render3VisitorWrapper) VisitLetDeclaration(decl *render3.LetDeclaration) interface{} {
	return w.humanizer.VisitLetDeclaration(decl)
}
func (w *render3VisitorWrapper) VisitComponent(component *render3.Component) interface{} {
	return w.humanizer.VisitComponent(component)
}
func (w *render3VisitorWrapper) VisitDirective(directive *render3.Directive) interface{} {
	return w.humanizer.VisitDirective(directive)
}

// ExpressionSourceHumanizer is a visitor that records AST expressions and their spans
// It implements expression_parser.AstVisitor
type ExpressionSourceHumanizer struct {
	result []HumanizedExpressionSource
}

// NewExpressionSourceHumanizer creates a new ExpressionSourceHumanizer
func NewExpressionSourceHumanizer() *ExpressionSourceHumanizer {
	return &ExpressionSourceHumanizer{
		result: []HumanizedExpressionSource{},
	}
}

// RecordAst records an AST node
func (e *ExpressionSourceHumanizer) RecordAst(ast expression_parser.AST) {
	unparsed := utils.Unparse(ast)
	span := ast.SourceSpan()
	e.result = append(e.result, HumanizedExpressionSource{
		Unparsed: unparsed,
		Span:     span,
	})
}

// Visit implements expression_parser.AstVisitor interface - generic visit method for AST
// Note: This conflicts with render3.Visitor.Visit, but we need it for AstVisitor.
// For render3.Visitor, we rely on render3.VisitAll calling node.Visit(e) instead.
func (e *ExpressionSourceHumanizer) Visit(ast expression_parser.AST, context interface{}) interface{} {
	return ast.Visit(e, context)
}

// VisitASTWithSource visits an AST with source
func (e *ExpressionSourceHumanizer) VisitASTWithSource(ast *expression_parser.ASTWithSource, context interface{}) interface{} {
	e.RecordAst(ast)
	if ast.AST != nil {
		ast.AST.Visit(e, context)
	}
	return nil
}

// VisitBinary visits a binary expression
func (e *ExpressionSourceHumanizer) VisitBinary(ast *expression_parser.Binary, context interface{}) interface{} {
	e.RecordAst(ast)
	if ast.Left != nil {
		ast.Left.Visit(e, context)
	}
	if ast.Right != nil {
		ast.Right.Visit(e, context)
	}
	return nil
}

// VisitChain visits a chain expression
func (e *ExpressionSourceHumanizer) VisitChain(ast *expression_parser.Chain, context interface{}) interface{} {
	e.RecordAst(ast)
	for _, expr := range ast.Expressions {
		if expr != nil {
			expr.Visit(e, context)
		}
	}
	return nil
}

// VisitConditional visits a conditional expression
func (e *ExpressionSourceHumanizer) VisitConditional(ast *expression_parser.Conditional, context interface{}) interface{} {
	e.RecordAst(ast)
	if ast.Condition != nil {
		ast.Condition.Visit(e, context)
	}
	if ast.TrueExp != nil {
		ast.TrueExp.Visit(e, context)
	}
	if ast.FalseExp != nil {
		ast.FalseExp.Visit(e, context)
	}
	return nil
}

// VisitImplicitReceiver visits an implicit receiver
func (e *ExpressionSourceHumanizer) VisitImplicitReceiver(ast *expression_parser.ImplicitReceiver, context interface{}) interface{} {
	e.RecordAst(ast)
	return nil
}

// VisitInterpolation visits an interpolation
func (e *ExpressionSourceHumanizer) VisitInterpolation(ast *expression_parser.Interpolation, context interface{}) interface{} {
	e.RecordAst(ast)
	for _, expr := range ast.Expressions {
		if expr != nil {
			expr.Visit(e, context)
		}
	}
	return nil
}

// VisitKeyedRead visits a keyed read
func (e *ExpressionSourceHumanizer) VisitKeyedRead(ast *expression_parser.KeyedRead, context interface{}) interface{} {
	e.RecordAst(ast)
	if ast.Receiver != nil {
		ast.Receiver.Visit(e, context)
	}
	if ast.Key != nil {
		ast.Key.Visit(e, context)
	}
	return nil
}

// VisitLiteralPrimitive visits a literal primitive
func (e *ExpressionSourceHumanizer) VisitLiteralPrimitive(ast *expression_parser.LiteralPrimitive, context interface{}) interface{} {
	e.RecordAst(ast)
	return nil
}

// VisitLiteralArray visits a literal array
func (e *ExpressionSourceHumanizer) VisitLiteralArray(ast *expression_parser.LiteralArray, context interface{}) interface{} {
	e.RecordAst(ast)
	for _, expr := range ast.Expressions {
		if expr != nil {
			expr.Visit(e, context)
		}
	}
	return nil
}

// VisitLiteralMap visits a literal map
func (e *ExpressionSourceHumanizer) VisitLiteralMap(ast *expression_parser.LiteralMap, context interface{}) interface{} {
	e.RecordAst(ast)
	for _, expr := range ast.Values {
		if expr != nil {
			expr.Visit(e, context)
		}
	}
	return nil
}

// VisitNonNullAssert visits a non-null assertion
func (e *ExpressionSourceHumanizer) VisitNonNullAssert(ast *expression_parser.NonNullAssert, context interface{}) interface{} {
	e.RecordAst(ast)
	if ast.Expression != nil {
		ast.Expression.Visit(e, context)
	}
	return nil
}

// VisitPipe visits a pipe
func (e *ExpressionSourceHumanizer) VisitPipe(ast *expression_parser.BindingPipe, context interface{}) interface{} {
	e.RecordAst(ast)
	if ast.Exp != nil {
		ast.Exp.Visit(e, context)
	}
	for _, arg := range ast.Args {
		if arg != nil {
			arg.Visit(e, context)
		}
	}
	return nil
}

// VisitPrefixNot visits a prefix not
func (e *ExpressionSourceHumanizer) VisitPrefixNot(ast *expression_parser.PrefixNot, context interface{}) interface{} {
	e.RecordAst(ast)
	if ast.Expression != nil {
		ast.Expression.Visit(e, context)
	}
	return nil
}

// VisitPropertyRead visits a property read
func (e *ExpressionSourceHumanizer) VisitPropertyRead(ast *expression_parser.PropertyRead, context interface{}) interface{} {
	e.RecordAst(ast)
	if ast.Receiver != nil {
		ast.Receiver.Visit(e, context)
	}
	return nil
}

// VisitSafePropertyRead visits a safe property read
func (e *ExpressionSourceHumanizer) VisitSafePropertyRead(ast *expression_parser.SafePropertyRead, context interface{}) interface{} {
	e.RecordAst(ast)
	if ast.Receiver != nil {
		ast.Receiver.Visit(e, context)
	}
	return nil
}

// VisitSafeKeyedRead visits a safe keyed read
func (e *ExpressionSourceHumanizer) VisitSafeKeyedRead(ast *expression_parser.SafeKeyedRead, context interface{}) interface{} {
	e.RecordAst(ast)
	if ast.Receiver != nil {
		ast.Receiver.Visit(e, context)
	}
	if ast.Key != nil {
		ast.Key.Visit(e, context)
	}
	return nil
}

// VisitCall visits a call
func (e *ExpressionSourceHumanizer) VisitCall(ast *expression_parser.Call, context interface{}) interface{} {
	e.RecordAst(ast)
	if ast.Receiver != nil {
		ast.Receiver.Visit(e, context)
	}
	for _, arg := range ast.Args {
		if arg != nil {
			arg.Visit(e, context)
		}
	}
	return nil
}

// VisitSafeCall visits a safe call
func (e *ExpressionSourceHumanizer) VisitSafeCall(ast *expression_parser.SafeCall, context interface{}) interface{} {
	e.RecordAst(ast)
	if ast.Receiver != nil {
		ast.Receiver.Visit(e, context)
	}
	for _, arg := range ast.Args {
		if arg != nil {
			arg.Visit(e, context)
		}
	}
	return nil
}

// VisitUnary visits a unary expression
func (e *ExpressionSourceHumanizer) VisitUnary(ast *expression_parser.Unary, context interface{}) interface{} {
	e.RecordAst(ast)
	if ast.Expr != nil {
		ast.Expr.Visit(e, context)
	}
	return nil
}

// VisitThisReceiver visits a this receiver
func (e *ExpressionSourceHumanizer) VisitThisReceiver(ast *expression_parser.ThisReceiver, context interface{}) interface{} {
	e.RecordAst(ast)
	return nil
}

// Note: KeyedWrite, PropertyWrite, SafePropertyWrite are not used in expression source humanization
// They are write operations, not read operations, so they don't appear in templates

// VisitTypeofExpression visits a typeof expression
func (e *ExpressionSourceHumanizer) VisitTypeofExpression(ast *expression_parser.TypeofExpression, context interface{}) interface{} {
	e.RecordAst(ast)
	if ast.Expression != nil {
		ast.Expression.Visit(e, context)
	}
	return nil
}

// VisitVoidExpression visits a void expression
func (e *ExpressionSourceHumanizer) VisitVoidExpression(ast *expression_parser.VoidExpression, context interface{}) interface{} {
	e.RecordAst(ast)
	if ast.Expression != nil {
		ast.Expression.Visit(e, context)
	}
	return nil
}

// VisitTemplateLiteral visits a template literal
func (e *ExpressionSourceHumanizer) VisitTemplateLiteral(ast *expression_parser.TemplateLiteral, context interface{}) interface{} {
	e.RecordAst(ast)
	for _, element := range ast.Elements {
		if element != nil {
			element.Visit(e, context)
		}
	}
	return nil
}

// VisitTemplateLiteralElement visits a template literal element
func (e *ExpressionSourceHumanizer) VisitTemplateLiteralElement(ast *expression_parser.TemplateLiteralElement, context interface{}) interface{} {
	e.RecordAst(ast)
	return nil
}

// VisitTaggedTemplateLiteral visits a tagged template literal
func (e *ExpressionSourceHumanizer) VisitTaggedTemplateLiteral(ast *expression_parser.TaggedTemplateLiteral, context interface{}) interface{} {
	e.RecordAst(ast)
	if ast.Tag != nil {
		ast.Tag.Visit(e, context)
	}
	if ast.Template != nil {
		ast.Template.Visit(e, context)
	}
	return nil
}

// VisitParenthesizedExpression visits a parenthesized expression
func (e *ExpressionSourceHumanizer) VisitParenthesizedExpression(ast *expression_parser.ParenthesizedExpression, context interface{}) interface{} {
	e.RecordAst(ast)
	if ast.Expression != nil {
		ast.Expression.Visit(e, context)
	}
	return nil
}

// VisitRegularExpressionLiteral visits a regular expression literal
func (e *ExpressionSourceHumanizer) VisitRegularExpressionLiteral(ast *expression_parser.RegularExpressionLiteral, context interface{}) interface{} {
	e.RecordAst(ast)
	return nil
}

// VisitEmptyExpr visits an empty expression
func (e *ExpressionSourceHumanizer) VisitEmptyExpr(ast *expression_parser.EmptyExpr, context interface{}) interface{} {
	return nil
}

// Render3 Visitor methods

// VisitElement visits an element
func (e *ExpressionSourceHumanizer) VisitElement(element *render3.Element) interface{} {
	// Create wrapper for render3.VisitAll and node.Visit calls
	wrapper := &render3VisitorWrapper{humanizer: e}
	// Visit inputs
	for _, input := range element.Inputs {
		input.Visit(wrapper)
	}
	// Visit outputs
	for _, output := range element.Outputs {
		output.Visit(wrapper)
	}
	// Visit children
	render3.VisitAll(wrapper, element.Children)
	return nil
}

// VisitTemplate visits a template
func (e *ExpressionSourceHumanizer) VisitTemplate(template *render3.Template) interface{} {
	wrapper := &render3VisitorWrapper{humanizer: e}
	// Visit directives
	for _, directive := range template.Directives {
		directive.Visit(wrapper)
	}
	// Visit children
	render3.VisitAll(wrapper, template.Children)
	// Visit templateAttrs (matches TypeScript: t.visitAll(this, ast.templateAttrs))
	for _, attr := range template.TemplateAttrs {
		if node, ok := attr.(render3.Node); ok {
			node.Visit(wrapper)
		}
	}
	return nil
}

// VisitContent visits content
func (e *ExpressionSourceHumanizer) VisitContent(content *render3.Content) interface{} {
	wrapper := &render3VisitorWrapper{humanizer: e}
	render3.VisitAll(wrapper, content.Children)
	return nil
}

// VisitVariable visits a variable
func (e *ExpressionSourceHumanizer) VisitVariable(variable *render3.Variable) interface{} {
	return nil
}

// VisitReference visits a reference
func (e *ExpressionSourceHumanizer) VisitReference(reference *render3.Reference) interface{} {
	return nil
}

// VisitTextAttribute visits a text attribute
func (e *ExpressionSourceHumanizer) VisitTextAttribute(attribute *render3.TextAttribute) interface{} {
	return nil
}

// VisitBoundAttribute visits a bound attribute
func (e *ExpressionSourceHumanizer) VisitBoundAttribute(attribute *render3.BoundAttribute) interface{} {
	if attribute.Value != nil {
		attribute.Value.Visit(e, nil)
	}
	return nil
}

// VisitBoundEvent visits a bound event
func (e *ExpressionSourceHumanizer) VisitBoundEvent(event *render3.BoundEvent) interface{} {
	if event.Handler != nil {
		event.Handler.Visit(e, nil)
	}
	return nil
}

// VisitText visits a text node
func (e *ExpressionSourceHumanizer) VisitText(text *render3.Text) interface{} {
	return nil
}

// VisitBoundText visits a bound text node
func (e *ExpressionSourceHumanizer) VisitBoundText(text *render3.BoundText) interface{} {
	if text.Value != nil {
		text.Value.Visit(e, nil)
	}
	return nil
}

// VisitIcu visits an ICU node
func (e *ExpressionSourceHumanizer) VisitIcu(icu *render3.Icu) interface{} {
	// Visit ICU variables
	for _, v := range icu.Vars {
		if v != nil && v.Value != nil {
			v.Value.Visit(e, nil)
		}
	}
	// Visit ICU placeholders (they are Nodes, not expressions)
	for _, p := range icu.Placeholders {
		if p != nil {
			// Placeholders can be Text or BoundText
			if boundText, ok := p.(*render3.BoundText); ok && boundText.Value != nil {
				boundText.Value.Visit(e, nil)
			}
			// Text nodes don't have expressions to visit
		}
	}
	return nil
}

// VisitDeferredBlock visits a deferred block
// Matches TypeScript: visitDeferredBlock(deferred: t.DeferredBlock) { deferred.visitAll(this); }
func (e *ExpressionSourceHumanizer) VisitDeferredBlock(deferred *render3.DeferredBlock) interface{} {
	wrapper := &render3VisitorWrapper{humanizer: e}
	deferred.VisitAll(wrapper)
	return nil
}

// VisitDeferredBlockPlaceholder visits a deferred block placeholder
func (e *ExpressionSourceHumanizer) VisitDeferredBlockPlaceholder(block *render3.DeferredBlockPlaceholder) interface{} {
	wrapper := &render3VisitorWrapper{humanizer: e}
	render3.VisitAll(wrapper, block.Children)
	return nil
}

// VisitDeferredBlockError visits a deferred block error
func (e *ExpressionSourceHumanizer) VisitDeferredBlockError(block *render3.DeferredBlockError) interface{} {
	wrapper := &render3VisitorWrapper{humanizer: e}
	render3.VisitAll(wrapper, block.Children)
	return nil
}

// VisitDeferredBlockLoading visits a deferred block loading
func (e *ExpressionSourceHumanizer) VisitDeferredBlockLoading(block *render3.DeferredBlockLoading) interface{} {
	wrapper := &render3VisitorWrapper{humanizer: e}
	render3.VisitAll(wrapper, block.Children)
	return nil
}

// VisitDeferredTrigger visits a deferred trigger
// Matches TypeScript: visitDeferredTrigger(trigger: t.DeferredTrigger)
// Note: In Go, trigger is passed as *DeferredTrigger, but we need to check if it's actually
// a *BoundDeferredTrigger or *ViewportDeferredTrigger. Since these embed *DeferredTrigger,
// we use reflection to check the actual type.
func (e *ExpressionSourceHumanizer) VisitDeferredTrigger(trigger *render3.DeferredTrigger) interface{} {
	// Use reflection to get the actual type
	triggerValue := reflect.ValueOf(trigger)
	if triggerValue.Kind() == reflect.Ptr {
		triggerValue = triggerValue.Elem()
	}

	// Check if it's a BoundDeferredTrigger by checking for Value field
	if triggerValue.Kind() == reflect.Struct {
		valueField := triggerValue.FieldByName("Value")
		if valueField.IsValid() && !valueField.IsNil() {
			if value, ok := valueField.Interface().(expression_parser.AST); ok && value != nil {
				e.RecordAst(value)
			}
			return nil
		}

		// Check if it's a ViewportDeferredTrigger by checking for Options field
		optionsField := triggerValue.FieldByName("Options")
		if optionsField.IsValid() && !optionsField.IsNil() {
			if options, ok := optionsField.Interface().(*expression_parser.LiteralMap); ok && options != nil {
				e.RecordAst(options)
			}
			return nil
		}
	}
	return nil
}

// VisitSwitchBlock visits a switch block
func (e *ExpressionSourceHumanizer) VisitSwitchBlock(block *render3.SwitchBlock) interface{} {
	wrapper := &render3VisitorWrapper{humanizer: e}
	if block.Expression != nil {
		block.Expression.Visit(e, nil)
	}
	// Visit cases
	for _, caseBlock := range block.Cases {
		caseBlock.Visit(wrapper)
	}
	return nil
}

// VisitSwitchBlockCase visits a switch block case
func (e *ExpressionSourceHumanizer) VisitSwitchBlockCase(block *render3.SwitchBlockCase) interface{} {
	wrapper := &render3VisitorWrapper{humanizer: e}
	if block.Expression != nil {
		block.Expression.Visit(e, nil)
	}
	render3.VisitAll(wrapper, block.Children)
	return nil
}

// VisitForLoopBlock visits a for loop block
func (e *ExpressionSourceHumanizer) VisitForLoopBlock(block *render3.ForLoopBlock) interface{} {
	wrapper := &render3VisitorWrapper{humanizer: e}
	block.Item.Visit(wrapper)
	// Visit context variables
	for _, variable := range block.ContextVariables {
		variable.Visit(wrapper)
	}
	if block.Expression != nil {
		block.Expression.Visit(e, nil)
	}
	render3.VisitAll(wrapper, block.Children)
	if block.Empty != nil {
		block.Empty.Visit(wrapper)
	}
	return nil
}

// VisitForLoopBlockEmpty visits a for loop block empty
func (e *ExpressionSourceHumanizer) VisitForLoopBlockEmpty(block *render3.ForLoopBlockEmpty) interface{} {
	wrapper := &render3VisitorWrapper{humanizer: e}
	render3.VisitAll(wrapper, block.Children)
	return nil
}

// VisitIfBlock visits an if block
func (e *ExpressionSourceHumanizer) VisitIfBlock(block *render3.IfBlock) interface{} {
	wrapper := &render3VisitorWrapper{humanizer: e}
	// Visit branches
	for _, branch := range block.Branches {
		branch.Visit(wrapper)
	}
	return nil
}

// VisitIfBlockBranch visits an if block branch
func (e *ExpressionSourceHumanizer) VisitIfBlockBranch(block *render3.IfBlockBranch) interface{} {
	wrapper := &render3VisitorWrapper{humanizer: e}
	if block.Expression != nil {
		block.Expression.Visit(e, nil)
	}
	if block.ExpressionAlias != nil {
		block.ExpressionAlias.Visit(wrapper)
	}
	render3.VisitAll(wrapper, block.Children)
	return nil
}

// VisitUnknownBlock visits an unknown block
func (e *ExpressionSourceHumanizer) VisitUnknownBlock(block *render3.UnknownBlock) interface{} {
	return nil
}

// VisitLetDeclaration visits a let declaration
func (e *ExpressionSourceHumanizer) VisitLetDeclaration(decl *render3.LetDeclaration) interface{} {
	if decl.Value != nil {
		decl.Value.Visit(e, nil)
	}
	return nil
}

// VisitComponent visits a component
func (e *ExpressionSourceHumanizer) VisitComponent(component *render3.Component) interface{} {
	wrapper := &render3VisitorWrapper{humanizer: e}
	render3.VisitAll(wrapper, component.Children)
	// Visit directives
	for _, directive := range component.Directives {
		directive.Visit(wrapper)
	}
	// Visit inputs
	for _, input := range component.Inputs {
		input.Visit(wrapper)
	}
	// Visit outputs
	for _, output := range component.Outputs {
		output.Visit(wrapper)
	}
	return nil
}

// VisitDirective visits a directive
func (e *ExpressionSourceHumanizer) VisitDirective(directive *render3.Directive) interface{} {
	wrapper := &render3VisitorWrapper{humanizer: e}
	// Visit inputs
	for _, input := range directive.Inputs {
		input.Visit(wrapper)
	}
	// Visit outputs
	for _, output := range directive.Outputs {
		output.Visit(wrapper)
	}
	return nil
}
