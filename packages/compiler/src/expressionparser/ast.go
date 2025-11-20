package expressionparser

import (
	"ngc-go/packages/compiler/src/core"
	"ngc-go/packages/compiler/src/util"
)

// ParseSpan represents a span within an expression
type ParseSpan struct {
	Start int
	End   int
}

// NewParseSpan creates a new ParseSpan
func NewParseSpan(start, end int) *ParseSpan {
	return &ParseSpan{Start: start, End: end}
}

// ToAbsolute converts a ParseSpan to an AbsoluteSourceSpan
func (ps *ParseSpan) ToAbsolute(absoluteOffset int) *AbsoluteSourceSpan {
	return NewAbsoluteSourceSpan(absoluteOffset+ps.Start, absoluteOffset+ps.End)
}

// AbsoluteSourceSpan records the absolute position of a text span in a source file
type AbsoluteSourceSpan struct {
	Start int
	End   int
}

// NewAbsoluteSourceSpan creates a new AbsoluteSourceSpan
func NewAbsoluteSourceSpan(start, end int) *AbsoluteSourceSpan {
	return &AbsoluteSourceSpan{Start: start, End: end}
}

// AST is the base interface for all AST nodes
type AST interface {
	Span() *ParseSpan
	SourceSpan() *AbsoluteSourceSpan
	Visit(visitor AstVisitor, context interface{}) interface{}
	String() string
}

// ASTWithName is the base class for AST nodes that have a name
type ASTWithName struct {
	span       *ParseSpan
	sourceSpan *AbsoluteSourceSpan
	nameSpan   *AbsoluteSourceSpan
}

// Span returns the parse span
func (a *ASTWithName) Span() *ParseSpan {
	return a.span
}

// SourceSpan returns the absolute source span
func (a *ASTWithName) SourceSpan() *AbsoluteSourceSpan {
	return a.sourceSpan
}

// EmptyExpr represents an empty expression
type EmptyExpr struct {
	span       *ParseSpan
	sourceSpan *AbsoluteSourceSpan
}

// NewEmptyExpr creates a new EmptyExpr
func NewEmptyExpr(span *ParseSpan, sourceSpan *AbsoluteSourceSpan) *EmptyExpr {
	return &EmptyExpr{span: span, sourceSpan: sourceSpan}
}

// Span returns the parse span
func (e *EmptyExpr) Span() *ParseSpan {
	return e.span
}

// SourceSpan returns the absolute source span
func (e *EmptyExpr) SourceSpan() *AbsoluteSourceSpan {
	return e.sourceSpan
}

// Visit implements the AST interface
func (e *EmptyExpr) Visit(visitor AstVisitor, context interface{}) interface{} {
	// do nothing
	return nil
}

// String returns string representation
func (e *EmptyExpr) String() string {
	return "AST"
}

// ImplicitReceiver represents an implicit receiver
type ImplicitReceiver struct {
	span       *ParseSpan
	sourceSpan *AbsoluteSourceSpan
}

// NewImplicitReceiver creates a new ImplicitReceiver
func NewImplicitReceiver(span *ParseSpan, sourceSpan *AbsoluteSourceSpan) *ImplicitReceiver {
	return &ImplicitReceiver{span: span, sourceSpan: sourceSpan}
}

// Span returns the parse span
func (i *ImplicitReceiver) Span() *ParseSpan {
	return i.span
}

// SourceSpan returns the absolute source span
func (i *ImplicitReceiver) SourceSpan() *AbsoluteSourceSpan {
	return i.sourceSpan
}

// Visit implements the AST interface
func (i *ImplicitReceiver) Visit(visitor AstVisitor, context interface{}) interface{} {
	return visitor.VisitImplicitReceiver(i, context)
}

// String returns string representation
func (i *ImplicitReceiver) String() string {
	return "AST"
}

// ThisReceiver represents a receiver when something is accessed through `this`
type ThisReceiver struct {
	*ImplicitReceiver
}

// NewThisReceiver creates a new ThisReceiver
func NewThisReceiver(span *ParseSpan, sourceSpan *AbsoluteSourceSpan) *ThisReceiver {
	return &ThisReceiver{ImplicitReceiver: NewImplicitReceiver(span, sourceSpan)}
}

// Visit implements the AST interface
func (t *ThisReceiver) Visit(visitor AstVisitor, context interface{}) interface{} {
	return visitor.VisitThisReceiver(t, context)
}

// Chain represents multiple expressions separated by a semicolon
type Chain struct {
	span        *ParseSpan
	sourceSpan  *AbsoluteSourceSpan
	Expressions []AST
}

// NewChain creates a new Chain
func NewChain(span *ParseSpan, sourceSpan *AbsoluteSourceSpan, expressions []AST) *Chain {
	return &Chain{
		span:        span,
		sourceSpan:  sourceSpan,
		Expressions: expressions,
	}
}

// Span returns the parse span
func (c *Chain) Span() *ParseSpan {
	return c.span
}

// SourceSpan returns the absolute source span
func (c *Chain) SourceSpan() *AbsoluteSourceSpan {
	return c.sourceSpan
}

// Visit implements the AST interface
func (c *Chain) Visit(visitor AstVisitor, context interface{}) interface{} {
	return visitor.VisitChain(c, context)
}

// String returns string representation
func (c *Chain) String() string {
	return "AST"
}

// Conditional represents a conditional expression (ternary operator)
type Conditional struct {
	span       *ParseSpan
	sourceSpan *AbsoluteSourceSpan
	Condition  AST
	TrueExp    AST
	FalseExp   AST
}

// NewConditional creates a new Conditional
func NewConditional(span *ParseSpan, sourceSpan *AbsoluteSourceSpan, condition, trueExp, falseExp AST) *Conditional {
	return &Conditional{
		span:       span,
		sourceSpan: sourceSpan,
		Condition:  condition,
		TrueExp:    trueExp,
		FalseExp:   falseExp,
	}
}

// Span returns the parse span
func (c *Conditional) Span() *ParseSpan {
	return c.span
}

// SourceSpan returns the absolute source span
func (c *Conditional) SourceSpan() *AbsoluteSourceSpan {
	return c.sourceSpan
}

// Visit implements the AST interface
func (c *Conditional) Visit(visitor AstVisitor, context interface{}) interface{} {
	return visitor.VisitConditional(c, context)
}

// String returns string representation
func (c *Conditional) String() string {
	return "AST"
}

// PropertyRead represents a property read operation
type PropertyRead struct {
	*ASTWithName
	Receiver AST
	Name     string
}

// NewPropertyRead creates a new PropertyRead
func NewPropertyRead(span *ParseSpan, sourceSpan, nameSpan *AbsoluteSourceSpan, receiver AST, name string) *PropertyRead {
	return &PropertyRead{
		ASTWithName: &ASTWithName{
			span:       span,
			sourceSpan: sourceSpan,
			nameSpan:   nameSpan,
		},
		Receiver: receiver,
		Name:     name,
	}
}

// Visit implements the AST interface
func (p *PropertyRead) Visit(visitor AstVisitor, context interface{}) interface{} {
	return visitor.VisitPropertyRead(p, context)
}

// String returns string representation
func (p *PropertyRead) String() string {
	return "AST"
}

// SafePropertyRead represents a safe property read operation (?.)
type SafePropertyRead struct {
	*ASTWithName
	Receiver AST
	Name     string
}

// NewSafePropertyRead creates a new SafePropertyRead
func NewSafePropertyRead(span *ParseSpan, sourceSpan, nameSpan *AbsoluteSourceSpan, receiver AST, name string) *SafePropertyRead {
	return &SafePropertyRead{
		ASTWithName: &ASTWithName{
			span:       span,
			sourceSpan: sourceSpan,
			nameSpan:   nameSpan,
		},
		Receiver: receiver,
		Name:     name,
	}
}

// Visit implements the AST interface
func (s *SafePropertyRead) Visit(visitor AstVisitor, context interface{}) interface{} {
	return visitor.VisitSafePropertyRead(s, context)
}

// String returns string representation
func (s *SafePropertyRead) String() string {
	return "AST"
}

// KeyedRead represents a keyed read operation (array/object access)
type KeyedRead struct {
	span       *ParseSpan
	sourceSpan *AbsoluteSourceSpan
	Receiver   AST
	Key        AST
}

// NewKeyedRead creates a new KeyedRead
func NewKeyedRead(span *ParseSpan, sourceSpan *AbsoluteSourceSpan, receiver, key AST) *KeyedRead {
	return &KeyedRead{
		span:       span,
		sourceSpan: sourceSpan,
		Receiver:   receiver,
		Key:        key,
	}
}

// Span returns the parse span
func (k *KeyedRead) Span() *ParseSpan {
	return k.span
}

// SourceSpan returns the absolute source span
func (k *KeyedRead) SourceSpan() *AbsoluteSourceSpan {
	return k.sourceSpan
}

// Visit implements the AST interface
func (k *KeyedRead) Visit(visitor AstVisitor, context interface{}) interface{} {
	return visitor.VisitKeyedRead(k, context)
}

// String returns string representation
func (k *KeyedRead) String() string {
	return "AST"
}

// SafeKeyedRead represents a safe keyed read operation (?.[])
type SafeKeyedRead struct {
	span       *ParseSpan
	sourceSpan *AbsoluteSourceSpan
	Receiver   AST
	Key        AST
}

// NewSafeKeyedRead creates a new SafeKeyedRead
func NewSafeKeyedRead(span *ParseSpan, sourceSpan *AbsoluteSourceSpan, receiver, key AST) *SafeKeyedRead {
	return &SafeKeyedRead{
		span:       span,
		sourceSpan: sourceSpan,
		Receiver:   receiver,
		Key:        key,
	}
}

// Span returns the parse span
func (s *SafeKeyedRead) Span() *ParseSpan {
	return s.span
}

// SourceSpan returns the absolute source span
func (s *SafeKeyedRead) SourceSpan() *AbsoluteSourceSpan {
	return s.sourceSpan
}

// Visit implements the AST interface
func (s *SafeKeyedRead) Visit(visitor AstVisitor, context interface{}) interface{} {
	return visitor.VisitSafeKeyedRead(s, context)
}

// String returns string representation
func (s *SafeKeyedRead) String() string {
	return "AST"
}

// BindingPipeType represents the type of a pipe reference
type BindingPipeType int

const (
	// ReferencedByName means the pipe is referenced by its name
	ReferencedByName BindingPipeType = iota
	// ReferencedDirectly means the pipe is referenced by its class name
	ReferencedDirectly
)

// BindingPipe represents a pipe operation
type BindingPipe struct {
	*ASTWithName
	Exp  AST
	Name string
	Args []AST
	Type BindingPipeType
}

// NewBindingPipe creates a new BindingPipe
func NewBindingPipe(span *ParseSpan, sourceSpan *AbsoluteSourceSpan, exp AST, name string, args []AST, typ BindingPipeType, nameSpan *AbsoluteSourceSpan) *BindingPipe {
	return &BindingPipe{
		ASTWithName: &ASTWithName{
			span:       span,
			sourceSpan: sourceSpan,
			nameSpan:   nameSpan,
		},
		Exp:  exp,
		Name: name,
		Args: args,
		Type: typ,
	}
}

// Visit implements the AST interface
func (b *BindingPipe) Visit(visitor AstVisitor, context interface{}) interface{} {
	return visitor.VisitPipe(b, context)
}

// String returns string representation
func (b *BindingPipe) String() string {
	return "AST"
}

// LiteralPrimitive represents a primitive literal value
type LiteralPrimitive struct {
	span       *ParseSpan
	sourceSpan *AbsoluteSourceSpan
	Value      interface{}
}

// NewLiteralPrimitive creates a new LiteralPrimitive
func NewLiteralPrimitive(span *ParseSpan, sourceSpan *AbsoluteSourceSpan, value interface{}) *LiteralPrimitive {
	return &LiteralPrimitive{
		span:       span,
		sourceSpan: sourceSpan,
		Value:      value,
	}
}

// Span returns the parse span
func (l *LiteralPrimitive) Span() *ParseSpan {
	return l.span
}

// SourceSpan returns the absolute source span
func (l *LiteralPrimitive) SourceSpan() *AbsoluteSourceSpan {
	return l.sourceSpan
}

// Visit implements the AST interface
func (l *LiteralPrimitive) Visit(visitor AstVisitor, context interface{}) interface{} {
	return visitor.VisitLiteralPrimitive(l, context)
}

// String returns string representation
func (l *LiteralPrimitive) String() string {
	return "AST"
}

// LiteralArray represents an array literal
type LiteralArray struct {
	span        *ParseSpan
	sourceSpan  *AbsoluteSourceSpan
	Expressions []AST
}

// NewLiteralArray creates a new LiteralArray
func NewLiteralArray(span *ParseSpan, sourceSpan *AbsoluteSourceSpan, expressions []AST) *LiteralArray {
	return &LiteralArray{
		span:        span,
		sourceSpan:  sourceSpan,
		Expressions: expressions,
	}
}

// Span returns the parse span
func (l *LiteralArray) Span() *ParseSpan {
	return l.span
}

// SourceSpan returns the absolute source span
func (l *LiteralArray) SourceSpan() *AbsoluteSourceSpan {
	return l.sourceSpan
}

// Visit implements the AST interface
func (l *LiteralArray) Visit(visitor AstVisitor, context interface{}) interface{} {
	return visitor.VisitLiteralArray(l, context)
}

// String returns string representation
func (l *LiteralArray) String() string {
	return "AST"
}

// LiteralMapKey represents a key in a literal map
type LiteralMapKey struct {
	Key                    string
	Quoted                 bool
	IsShorthandInitialized bool
}

// LiteralMap represents a map/object literal
type LiteralMap struct {
	span       *ParseSpan
	sourceSpan *AbsoluteSourceSpan
	Keys       []LiteralMapKey
	Values     []AST
}

// NewLiteralMap creates a new LiteralMap
func NewLiteralMap(span *ParseSpan, sourceSpan *AbsoluteSourceSpan, keys []LiteralMapKey, values []AST) *LiteralMap {
	return &LiteralMap{
		span:       span,
		sourceSpan: sourceSpan,
		Keys:       keys,
		Values:     values,
	}
}

// Span returns the parse span
func (l *LiteralMap) Span() *ParseSpan {
	return l.span
}

// SourceSpan returns the absolute source span
func (l *LiteralMap) SourceSpan() *AbsoluteSourceSpan {
	return l.sourceSpan
}

// Visit implements the AST interface
func (l *LiteralMap) Visit(visitor AstVisitor, context interface{}) interface{} {
	return visitor.VisitLiteralMap(l, context)
}

// String returns string representation
func (l *LiteralMap) String() string {
	return "AST"
}

// Interpolation represents an interpolation expression
type Interpolation struct {
	span        *ParseSpan
	sourceSpan  *AbsoluteSourceSpan
	Strings     []string
	Expressions []AST
}

// NewInterpolation creates a new Interpolation
func NewInterpolation(span *ParseSpan, sourceSpan *AbsoluteSourceSpan, strings []string, expressions []AST) *Interpolation {
	return &Interpolation{
		span:        span,
		sourceSpan:  sourceSpan,
		Strings:     strings,
		Expressions: expressions,
	}
}

// Span returns the parse span
func (i *Interpolation) Span() *ParseSpan {
	return i.span
}

// SourceSpan returns the absolute source span
func (i *Interpolation) SourceSpan() *AbsoluteSourceSpan {
	return i.sourceSpan
}

// Visit implements the AST interface
func (i *Interpolation) Visit(visitor AstVisitor, context interface{}) interface{} {
	return visitor.VisitInterpolation(i, context)
}

// String returns string representation
func (i *Interpolation) String() string {
	return "AST"
}

// Binary represents a binary operation
type Binary struct {
	span       *ParseSpan
	sourceSpan *AbsoluteSourceSpan
	Operation  string
	Left       AST
	Right      AST
}

// NewBinary creates a new Binary
func NewBinary(span *ParseSpan, sourceSpan *AbsoluteSourceSpan, operation string, left, right AST) *Binary {
	return &Binary{
		span:       span,
		sourceSpan: sourceSpan,
		Operation:  operation,
		Left:       left,
		Right:      right,
	}
}

// Span returns the parse span
func (b *Binary) Span() *ParseSpan {
	return b.span
}

// SourceSpan returns the absolute source span
func (b *Binary) SourceSpan() *AbsoluteSourceSpan {
	return b.sourceSpan
}

// Visit implements the AST interface
func (b *Binary) Visit(visitor AstVisitor, context interface{}) interface{} {
	return visitor.VisitBinary(b, context)
}

// String returns string representation
func (b *Binary) String() string {
	return "AST"
}

// IsAssignmentOperation checks if an operator is an assignment operation
func IsAssignmentOperation(op string) bool {
	return op == "=" ||
		op == "+=" ||
		op == "-=" ||
		op == "*=" ||
		op == "/=" ||
		op == "%=" ||
		op == "**=" ||
		op == "&&=" ||
		op == "||=" ||
		op == "??="
}

// Unary represents a unary operation
type Unary struct {
	*Binary
	Operator string
	Expr     AST
}

// NewUnary creates a new Unary
func NewUnary(span *ParseSpan, sourceSpan *AbsoluteSourceSpan, operator string, expr AST, binaryOp string, binaryLeft, binaryRight AST) *Unary {
	return &Unary{
		Binary:   NewBinary(span, sourceSpan, binaryOp, binaryLeft, binaryRight),
		Operator: operator,
		Expr:     expr,
	}
}

// CreateMinus creates a unary minus expression "-x"
func CreateMinus(span *ParseSpan, sourceSpan *AbsoluteSourceSpan, expr AST) *Unary {
	return NewUnary(
		span,
		sourceSpan,
		"-",
		expr,
		"-",
		NewLiteralPrimitive(span, sourceSpan, 0),
		expr,
	)
}

// CreatePlus creates a unary plus expression "+x"
func CreatePlus(span *ParseSpan, sourceSpan *AbsoluteSourceSpan, expr AST) *Unary {
	return NewUnary(
		span,
		sourceSpan,
		"+",
		expr,
		"-",
		expr,
		NewLiteralPrimitive(span, sourceSpan, 0),
	)
}

// Visit implements the AST interface
func (u *Unary) Visit(visitor AstVisitor, context interface{}) interface{} {
	return visitor.VisitUnary(u, context)
}

// PrefixNot represents a prefix not operation (!)
type PrefixNot struct {
	span       *ParseSpan
	sourceSpan *AbsoluteSourceSpan
	Expression AST
}

// NewPrefixNot creates a new PrefixNot
func NewPrefixNot(span *ParseSpan, sourceSpan *AbsoluteSourceSpan, expression AST) *PrefixNot {
	return &PrefixNot{
		span:       span,
		sourceSpan: sourceSpan,
		Expression: expression,
	}
}

// Span returns the parse span
func (p *PrefixNot) Span() *ParseSpan {
	return p.span
}

// SourceSpan returns the absolute source span
func (p *PrefixNot) SourceSpan() *AbsoluteSourceSpan {
	return p.sourceSpan
}

// Visit implements the AST interface
func (p *PrefixNot) Visit(visitor AstVisitor, context interface{}) interface{} {
	return visitor.VisitPrefixNot(p, context)
}

// String returns string representation
func (p *PrefixNot) String() string {
	return "AST"
}

// TypeofExpression represents a typeof expression
type TypeofExpression struct {
	span       *ParseSpan
	sourceSpan *AbsoluteSourceSpan
	Expression AST
}

// NewTypeofExpression creates a new TypeofExpression
func NewTypeofExpression(span *ParseSpan, sourceSpan *AbsoluteSourceSpan, expression AST) *TypeofExpression {
	return &TypeofExpression{
		span:       span,
		sourceSpan: sourceSpan,
		Expression: expression,
	}
}

// Span returns the parse span
func (t *TypeofExpression) Span() *ParseSpan {
	return t.span
}

// SourceSpan returns the absolute source span
func (t *TypeofExpression) SourceSpan() *AbsoluteSourceSpan {
	return t.sourceSpan
}

// Visit implements the AST interface
func (t *TypeofExpression) Visit(visitor AstVisitor, context interface{}) interface{} {
	return visitor.VisitTypeofExpression(t, context)
}

// String returns string representation
func (t *TypeofExpression) String() string {
	return "AST"
}

// VoidExpression represents a void expression
type VoidExpression struct {
	span       *ParseSpan
	sourceSpan *AbsoluteSourceSpan
	Expression AST
}

// NewVoidExpression creates a new VoidExpression
func NewVoidExpression(span *ParseSpan, sourceSpan *AbsoluteSourceSpan, expression AST) *VoidExpression {
	return &VoidExpression{
		span:       span,
		sourceSpan: sourceSpan,
		Expression: expression,
	}
}

// Span returns the parse span
func (v *VoidExpression) Span() *ParseSpan {
	return v.span
}

// SourceSpan returns the absolute source span
func (v *VoidExpression) SourceSpan() *AbsoluteSourceSpan {
	return v.sourceSpan
}

// Visit implements the AST interface
func (v *VoidExpression) Visit(visitor AstVisitor, context interface{}) interface{} {
	return visitor.VisitVoidExpression(v, context)
}

// String returns string representation
func (v *VoidExpression) String() string {
	return "AST"
}

// NonNullAssert represents a non-null assertion operation (!)
type NonNullAssert struct {
	span       *ParseSpan
	sourceSpan *AbsoluteSourceSpan
	Expression AST
}

// NewNonNullAssert creates a new NonNullAssert
func NewNonNullAssert(span *ParseSpan, sourceSpan *AbsoluteSourceSpan, expression AST) *NonNullAssert {
	return &NonNullAssert{
		span:       span,
		sourceSpan: sourceSpan,
		Expression: expression,
	}
}

// Span returns the parse span
func (n *NonNullAssert) Span() *ParseSpan {
	return n.span
}

// SourceSpan returns the absolute source span
func (n *NonNullAssert) SourceSpan() *AbsoluteSourceSpan {
	return n.sourceSpan
}

// Visit implements the AST interface
func (n *NonNullAssert) Visit(visitor AstVisitor, context interface{}) interface{} {
	return visitor.VisitNonNullAssert(n, context)
}

// String returns string representation
func (n *NonNullAssert) String() string {
	return "AST"
}

// Call represents a function call
type Call struct {
	span         *ParseSpan
	sourceSpan   *AbsoluteSourceSpan
	Receiver     AST
	Args         []AST
	ArgumentSpan *AbsoluteSourceSpan
}

// NewCall creates a new Call
func NewCall(span *ParseSpan, sourceSpan *AbsoluteSourceSpan, receiver AST, args []AST, argumentSpan *AbsoluteSourceSpan) *Call {
	return &Call{
		span:         span,
		sourceSpan:   sourceSpan,
		Receiver:     receiver,
		Args:         args,
		ArgumentSpan: argumentSpan,
	}
}

// Span returns the parse span
func (c *Call) Span() *ParseSpan {
	return c.span
}

// SourceSpan returns the absolute source span
func (c *Call) SourceSpan() *AbsoluteSourceSpan {
	return c.sourceSpan
}

// Visit implements the AST interface
func (c *Call) Visit(visitor AstVisitor, context interface{}) interface{} {
	return visitor.VisitCall(c, context)
}

// String returns string representation
func (c *Call) String() string {
	return "AST"
}

// SafeCall represents a safe function call (?.)
type SafeCall struct {
	span         *ParseSpan
	sourceSpan   *AbsoluteSourceSpan
	Receiver     AST
	Args         []AST
	ArgumentSpan *AbsoluteSourceSpan
}

// NewSafeCall creates a new SafeCall
func NewSafeCall(span *ParseSpan, sourceSpan *AbsoluteSourceSpan, receiver AST, args []AST, argumentSpan *AbsoluteSourceSpan) *SafeCall {
	return &SafeCall{
		span:         span,
		sourceSpan:   sourceSpan,
		Receiver:     receiver,
		Args:         args,
		ArgumentSpan: argumentSpan,
	}
}

// Span returns the parse span
func (s *SafeCall) Span() *ParseSpan {
	return s.span
}

// SourceSpan returns the absolute source span
func (s *SafeCall) SourceSpan() *AbsoluteSourceSpan {
	return s.sourceSpan
}

// Visit implements the AST interface
func (s *SafeCall) Visit(visitor AstVisitor, context interface{}) interface{} {
	return visitor.VisitSafeCall(s, context)
}

// String returns string representation
func (s *SafeCall) String() string {
	return "AST"
}

// TaggedTemplateLiteral represents a tagged template literal
type TaggedTemplateLiteral struct {
	span       *ParseSpan
	sourceSpan *AbsoluteSourceSpan
	Tag        AST
	Template   *TemplateLiteral
}

// NewTaggedTemplateLiteral creates a new TaggedTemplateLiteral
func NewTaggedTemplateLiteral(span *ParseSpan, sourceSpan *AbsoluteSourceSpan, tag AST, template *TemplateLiteral) *TaggedTemplateLiteral {
	return &TaggedTemplateLiteral{
		span:       span,
		sourceSpan: sourceSpan,
		Tag:        tag,
		Template:   template,
	}
}

// Span returns the parse span
func (t *TaggedTemplateLiteral) Span() *ParseSpan {
	return t.span
}

// SourceSpan returns the absolute source span
func (t *TaggedTemplateLiteral) SourceSpan() *AbsoluteSourceSpan {
	return t.sourceSpan
}

// Visit implements the AST interface
func (t *TaggedTemplateLiteral) Visit(visitor AstVisitor, context interface{}) interface{} {
	return visitor.VisitTaggedTemplateLiteral(t, context)
}

// String returns string representation
func (t *TaggedTemplateLiteral) String() string {
	return "AST"
}

// TemplateLiteral represents a template literal
type TemplateLiteral struct {
	span        *ParseSpan
	sourceSpan  *AbsoluteSourceSpan
	Elements    []*TemplateLiteralElement
	Expressions []AST
}

// NewTemplateLiteral creates a new TemplateLiteral
func NewTemplateLiteral(span *ParseSpan, sourceSpan *AbsoluteSourceSpan, elements []*TemplateLiteralElement, expressions []AST) *TemplateLiteral {
	return &TemplateLiteral{
		span:        span,
		sourceSpan:  sourceSpan,
		Elements:    elements,
		Expressions: expressions,
	}
}

// Span returns the parse span
func (t *TemplateLiteral) Span() *ParseSpan {
	return t.span
}

// SourceSpan returns the absolute source span
func (t *TemplateLiteral) SourceSpan() *AbsoluteSourceSpan {
	return t.sourceSpan
}

// Visit implements the AST interface
func (t *TemplateLiteral) Visit(visitor AstVisitor, context interface{}) interface{} {
	return visitor.VisitTemplateLiteral(t, context)
}

// String returns string representation
func (t *TemplateLiteral) String() string {
	return "AST"
}

// TemplateLiteralElement represents an element in a template literal
type TemplateLiteralElement struct {
	span       *ParseSpan
	sourceSpan *AbsoluteSourceSpan
	Text       string
}

// NewTemplateLiteralElement creates a new TemplateLiteralElement
func NewTemplateLiteralElement(span *ParseSpan, sourceSpan *AbsoluteSourceSpan, text string) *TemplateLiteralElement {
	return &TemplateLiteralElement{
		span:       span,
		sourceSpan: sourceSpan,
		Text:       text,
	}
}

// Span returns the parse span
func (t *TemplateLiteralElement) Span() *ParseSpan {
	return t.span
}

// SourceSpan returns the absolute source span
func (t *TemplateLiteralElement) SourceSpan() *AbsoluteSourceSpan {
	return t.sourceSpan
}

// Visit implements the AST interface
func (t *TemplateLiteralElement) Visit(visitor AstVisitor, context interface{}) interface{} {
	return visitor.VisitTemplateLiteralElement(t, context)
}

// String returns string representation
func (t *TemplateLiteralElement) String() string {
	return "AST"
}

// ParenthesizedExpression represents a parenthesized expression
type ParenthesizedExpression struct {
	span       *ParseSpan
	sourceSpan *AbsoluteSourceSpan
	Expression AST
}

// NewParenthesizedExpression creates a new ParenthesizedExpression
func NewParenthesizedExpression(span *ParseSpan, sourceSpan *AbsoluteSourceSpan, expression AST) *ParenthesizedExpression {
	return &ParenthesizedExpression{
		span:       span,
		sourceSpan: sourceSpan,
		Expression: expression,
	}
}

// Span returns the parse span
func (p *ParenthesizedExpression) Span() *ParseSpan {
	return p.span
}

// SourceSpan returns the absolute source span
func (p *ParenthesizedExpression) SourceSpan() *AbsoluteSourceSpan {
	return p.sourceSpan
}

// Visit implements the AST interface
func (p *ParenthesizedExpression) Visit(visitor AstVisitor, context interface{}) interface{} {
	return visitor.VisitParenthesizedExpression(p, context)
}

// String returns string representation
func (p *ParenthesizedExpression) String() string {
	return "AST"
}

// RegularExpressionLiteral represents a regular expression literal
type RegularExpressionLiteral struct {
	span       *ParseSpan
	sourceSpan *AbsoluteSourceSpan
	Body       string
	Flags      *string
}

// NewRegularExpressionLiteral creates a new RegularExpressionLiteral
func NewRegularExpressionLiteral(span *ParseSpan, sourceSpan *AbsoluteSourceSpan, body string, flags *string) *RegularExpressionLiteral {
	return &RegularExpressionLiteral{
		span:       span,
		sourceSpan: sourceSpan,
		Body:       body,
		Flags:      flags,
	}
}

// Span returns the parse span
func (r *RegularExpressionLiteral) Span() *ParseSpan {
	return r.span
}

// SourceSpan returns the absolute source span
func (r *RegularExpressionLiteral) SourceSpan() *AbsoluteSourceSpan {
	return r.sourceSpan
}

// Visit implements the AST interface
func (r *RegularExpressionLiteral) Visit(visitor AstVisitor, context interface{}) interface{} {
	return visitor.VisitRegularExpressionLiteral(r, context)
}

// String returns string representation
func (r *RegularExpressionLiteral) String() string {
	return "AST"
}

// ASTWithSource wraps an AST with source information
type ASTWithSource struct {
	AST            AST
	Source         *string
	Location       string
	AbsoluteOffset int
	Errors         []*util.ParseError
	span           *ParseSpan
	sourceSpan     *AbsoluteSourceSpan
}

// NewASTWithSource creates a new ASTWithSource
func NewASTWithSource(ast AST, source *string, location string, absoluteOffset int, errors []*util.ParseError) *ASTWithSource {
	sourceLen := 0
	if source != nil {
		sourceLen = len(*source)
	}
	span := NewParseSpan(0, sourceLen)
	return &ASTWithSource{
		AST:            ast,
		Source:         source,
		Location:       location,
		AbsoluteOffset: absoluteOffset,
		Errors:         errors,
		span:           span,
		sourceSpan:     span.ToAbsolute(absoluteOffset),
	}
}

// Span returns the parse span
func (a *ASTWithSource) Span() *ParseSpan {
	return a.span
}

// SourceSpan returns the absolute source span
func (a *ASTWithSource) SourceSpan() *AbsoluteSourceSpan {
	return a.sourceSpan
}

// Visit implements the AST interface
func (a *ASTWithSource) Visit(visitor AstVisitor, context interface{}) interface{} {
	return visitor.VisitASTWithSource(a, context)
}

// String returns string representation
func (a *ASTWithSource) String() string {
	if a.Source != nil {
		return *a.Source + " in " + a.Location
	}
	return "null in " + a.Location
}

// TemplateBindingIdentifier represents an identifier in a template binding
type TemplateBindingIdentifier struct {
	Source string
	Span   *AbsoluteSourceSpan
}

// TemplateBinding represents a template binding
type TemplateBinding interface {
	SourceSpan() *AbsoluteSourceSpan
}

// VariableBinding represents a variable binding
type VariableBinding struct {
	sourceSpan *AbsoluteSourceSpan
	Key        *TemplateBindingIdentifier
	Value      *TemplateBindingIdentifier
}

// NewVariableBinding creates a new VariableBinding
func NewVariableBinding(sourceSpan *AbsoluteSourceSpan, key, value *TemplateBindingIdentifier) *VariableBinding {
	return &VariableBinding{
		sourceSpan: sourceSpan,
		Key:        key,
		Value:      value,
	}
}

// SourceSpan returns the source span
func (v *VariableBinding) SourceSpan() *AbsoluteSourceSpan {
	return v.sourceSpan
}

// ExpressionBinding represents an expression binding
type ExpressionBinding struct {
	sourceSpan *AbsoluteSourceSpan
	Key        *TemplateBindingIdentifier
	Value      *ASTWithSource
}

// NewExpressionBinding creates a new ExpressionBinding
func NewExpressionBinding(sourceSpan *AbsoluteSourceSpan, key *TemplateBindingIdentifier, value *ASTWithSource) *ExpressionBinding {
	return &ExpressionBinding{
		sourceSpan: sourceSpan,
		Key:        key,
		Value:      value,
	}
}

// SourceSpan returns the source span
func (e *ExpressionBinding) SourceSpan() *AbsoluteSourceSpan {
	return e.sourceSpan
}

// AstVisitor is the interface for visiting AST nodes
type AstVisitor interface {
	VisitUnary(ast *Unary, context interface{}) interface{}
	VisitBinary(ast *Binary, context interface{}) interface{}
	VisitChain(ast *Chain, context interface{}) interface{}
	VisitConditional(ast *Conditional, context interface{}) interface{}
	VisitThisReceiver(ast *ThisReceiver, context interface{}) interface{}
	VisitImplicitReceiver(ast *ImplicitReceiver, context interface{}) interface{}
	VisitInterpolation(ast *Interpolation, context interface{}) interface{}
	VisitKeyedRead(ast *KeyedRead, context interface{}) interface{}
	VisitLiteralArray(ast *LiteralArray, context interface{}) interface{}
	VisitLiteralMap(ast *LiteralMap, context interface{}) interface{}
	VisitLiteralPrimitive(ast *LiteralPrimitive, context interface{}) interface{}
	VisitPipe(ast *BindingPipe, context interface{}) interface{}
	VisitPrefixNot(ast *PrefixNot, context interface{}) interface{}
	VisitTypeofExpression(ast *TypeofExpression, context interface{}) interface{}
	VisitVoidExpression(ast *VoidExpression, context interface{}) interface{}
	VisitNonNullAssert(ast *NonNullAssert, context interface{}) interface{}
	VisitPropertyRead(ast *PropertyRead, context interface{}) interface{}
	VisitSafePropertyRead(ast *SafePropertyRead, context interface{}) interface{}
	VisitSafeKeyedRead(ast *SafeKeyedRead, context interface{}) interface{}
	VisitCall(ast *Call, context interface{}) interface{}
	VisitSafeCall(ast *SafeCall, context interface{}) interface{}
	VisitTemplateLiteral(ast *TemplateLiteral, context interface{}) interface{}
	VisitTemplateLiteralElement(ast *TemplateLiteralElement, context interface{}) interface{}
	VisitTaggedTemplateLiteral(ast *TaggedTemplateLiteral, context interface{}) interface{}
	VisitParenthesizedExpression(ast *ParenthesizedExpression, context interface{}) interface{}
	VisitRegularExpressionLiteral(ast *RegularExpressionLiteral, context interface{}) interface{}
	VisitASTWithSource(ast *ASTWithSource, context interface{}) interface{}
	Visit(ast AST, context interface{}) interface{}
}

// RecursiveAstVisitor is a base visitor that recursively visits all nodes
type RecursiveAstVisitor struct{}

// Visit is the default visit method
func (r *RecursiveAstVisitor) Visit(ast AST, context interface{}) interface{} {
	ast.Visit(r, context)
	return nil
}

// VisitUnary visits a unary expression
func (r *RecursiveAstVisitor) VisitUnary(ast *Unary, context interface{}) interface{} {
	r.Visit(ast.Expr, context)
	return nil
}

// VisitBinary visits a binary expression
func (r *RecursiveAstVisitor) VisitBinary(ast *Binary, context interface{}) interface{} {
	r.Visit(ast.Left, context)
	r.Visit(ast.Right, context)
	return nil
}

// VisitChain visits a chain expression
func (r *RecursiveAstVisitor) VisitChain(ast *Chain, context interface{}) interface{} {
	r.VisitAll(ast.Expressions, context)
	return nil
}

// VisitConditional visits a conditional expression
func (r *RecursiveAstVisitor) VisitConditional(ast *Conditional, context interface{}) interface{} {
	r.Visit(ast.Condition, context)
	r.Visit(ast.TrueExp, context)
	r.Visit(ast.FalseExp, context)
	return nil
}

// VisitPipe visits a pipe expression
func (r *RecursiveAstVisitor) VisitPipe(ast *BindingPipe, context interface{}) interface{} {
	r.Visit(ast.Exp, context)
	r.VisitAll(ast.Args, context)
	return nil
}

// VisitImplicitReceiver visits an implicit receiver
func (r *RecursiveAstVisitor) VisitImplicitReceiver(ast *ImplicitReceiver, context interface{}) interface{} {
	return nil
}

// VisitThisReceiver visits a this receiver
func (r *RecursiveAstVisitor) VisitThisReceiver(ast *ThisReceiver, context interface{}) interface{} {
	return nil
}

// VisitInterpolation visits an interpolation
func (r *RecursiveAstVisitor) VisitInterpolation(ast *Interpolation, context interface{}) interface{} {
	r.VisitAll(ast.Expressions, context)
	return nil
}

// VisitKeyedRead visits a keyed read
func (r *RecursiveAstVisitor) VisitKeyedRead(ast *KeyedRead, context interface{}) interface{} {
	r.Visit(ast.Receiver, context)
	r.Visit(ast.Key, context)
	return nil
}

// VisitLiteralArray visits a literal array
func (r *RecursiveAstVisitor) VisitLiteralArray(ast *LiteralArray, context interface{}) interface{} {
	r.VisitAll(ast.Expressions, context)
	return nil
}

// VisitLiteralMap visits a literal map
func (r *RecursiveAstVisitor) VisitLiteralMap(ast *LiteralMap, context interface{}) interface{} {
	r.VisitAll(ast.Values, context)
	return nil
}

// VisitLiteralPrimitive visits a literal primitive
func (r *RecursiveAstVisitor) VisitLiteralPrimitive(ast *LiteralPrimitive, context interface{}) interface{} {
	return nil
}

// VisitPrefixNot visits a prefix not
func (r *RecursiveAstVisitor) VisitPrefixNot(ast *PrefixNot, context interface{}) interface{} {
	r.Visit(ast.Expression, context)
	return nil
}

// VisitTypeofExpression visits a typeof expression
func (r *RecursiveAstVisitor) VisitTypeofExpression(ast *TypeofExpression, context interface{}) interface{} {
	r.Visit(ast.Expression, context)
	return nil
}

// VisitVoidExpression visits a void expression
func (r *RecursiveAstVisitor) VisitVoidExpression(ast *VoidExpression, context interface{}) interface{} {
	r.Visit(ast.Expression, context)
	return nil
}

// VisitNonNullAssert visits a non-null assertion
func (r *RecursiveAstVisitor) VisitNonNullAssert(ast *NonNullAssert, context interface{}) interface{} {
	r.Visit(ast.Expression, context)
	return nil
}

// VisitPropertyRead visits a property read
func (r *RecursiveAstVisitor) VisitPropertyRead(ast *PropertyRead, context interface{}) interface{} {
	r.Visit(ast.Receiver, context)
	return nil
}

// VisitSafePropertyRead visits a safe property read
func (r *RecursiveAstVisitor) VisitSafePropertyRead(ast *SafePropertyRead, context interface{}) interface{} {
	r.Visit(ast.Receiver, context)
	return nil
}

// VisitSafeKeyedRead visits a safe keyed read
func (r *RecursiveAstVisitor) VisitSafeKeyedRead(ast *SafeKeyedRead, context interface{}) interface{} {
	r.Visit(ast.Receiver, context)
	r.Visit(ast.Key, context)
	return nil
}

// VisitCall visits a call
func (r *RecursiveAstVisitor) VisitCall(ast *Call, context interface{}) interface{} {
	r.Visit(ast.Receiver, context)
	r.VisitAll(ast.Args, context)
	return nil
}

// VisitSafeCall visits a safe call
func (r *RecursiveAstVisitor) VisitSafeCall(ast *SafeCall, context interface{}) interface{} {
	r.Visit(ast.Receiver, context)
	r.VisitAll(ast.Args, context)
	return nil
}

// VisitTemplateLiteral visits a template literal
func (r *RecursiveAstVisitor) VisitTemplateLiteral(ast *TemplateLiteral, context interface{}) interface{} {
	for i := 0; i < len(ast.Elements); i++ {
		r.Visit(ast.Elements[i], context)
		if i < len(ast.Expressions) {
			r.Visit(ast.Expressions[i], context)
		}
	}
	return nil
}

// VisitTemplateLiteralElement visits a template literal element
func (r *RecursiveAstVisitor) VisitTemplateLiteralElement(ast *TemplateLiteralElement, context interface{}) interface{} {
	return nil
}

// VisitTaggedTemplateLiteral visits a tagged template literal
func (r *RecursiveAstVisitor) VisitTaggedTemplateLiteral(ast *TaggedTemplateLiteral, context interface{}) interface{} {
	r.Visit(ast.Tag, context)
	r.Visit(ast.Template, context)
	return nil
}

// VisitParenthesizedExpression visits a parenthesized expression
func (r *RecursiveAstVisitor) VisitParenthesizedExpression(ast *ParenthesizedExpression, context interface{}) interface{} {
	r.Visit(ast.Expression, context)
	return nil
}

// VisitRegularExpressionLiteral visits a regular expression literal
func (r *RecursiveAstVisitor) VisitRegularExpressionLiteral(ast *RegularExpressionLiteral, context interface{}) interface{} {
	return nil
}

// VisitASTWithSource visits an AST with source
func (r *RecursiveAstVisitor) VisitASTWithSource(ast *ASTWithSource, context interface{}) interface{} {
	return r.Visit(ast.AST, context)
}

// VisitAll is a helper method to visit all ASTs in a slice
func (r *RecursiveAstVisitor) VisitAll(asts []AST, context interface{}) {
	for _, ast := range asts {
		r.Visit(ast, context)
	}
}

// ParsedPropertyType represents the type of a parsed property
type ParsedPropertyType int

const (
	ParsedPropertyTypeDefault ParsedPropertyType = iota
	ParsedPropertyTypeLiteralAttr
	ParsedPropertyTypeLegacyAnimation
	ParsedPropertyTypeTwoWay
	ParsedPropertyTypeAnimation
)

// ParsedProperty represents a parsed property
type ParsedProperty struct {
	Name              string
	Expression        *ASTWithSource
	Type              ParsedPropertyType
	SourceSpan        *util.ParseSourceSpan
	KeySpan           *util.ParseSourceSpan
	ValueSpan         *util.ParseSourceSpan
	IsLiteral         bool
	IsLegacyAnimation bool
	IsAnimation       bool
}

// NewParsedProperty creates a new ParsedProperty
func NewParsedProperty(name string, expression *ASTWithSource, typ ParsedPropertyType, sourceSpan, keySpan *util.ParseSourceSpan, valueSpan *util.ParseSourceSpan) *ParsedProperty {
	return &ParsedProperty{
		Name:              name,
		Expression:        expression,
		Type:              typ,
		SourceSpan:        sourceSpan,
		KeySpan:           keySpan,
		ValueSpan:         valueSpan,
		IsLiteral:         typ == ParsedPropertyTypeLiteralAttr,
		IsLegacyAnimation: typ == ParsedPropertyTypeLegacyAnimation,
		IsAnimation:       typ == ParsedPropertyTypeAnimation,
	}
}

// ParsedEventType represents the type of a parsed event
type ParsedEventType int

const (
	ParsedEventTypeRegular ParsedEventType = iota
	ParsedEventTypeLegacyAnimation
	ParsedEventTypeTwoWay
	ParsedEventTypeAnimation
)

// ParsedEvent represents a parsed event
type ParsedEvent struct {
	Name          string
	TargetOrPhase *string
	Type          ParsedEventType
	Handler       *ASTWithSource
	SourceSpan    *util.ParseSourceSpan
	HandlerSpan   *util.ParseSourceSpan
	KeySpan       *util.ParseSourceSpan
}

// NewParsedEvent creates a new ParsedEvent
func NewParsedEvent(name string, targetOrPhase *string, typ ParsedEventType, handler *ASTWithSource, sourceSpan, handlerSpan, keySpan *util.ParseSourceSpan) *ParsedEvent {
	return &ParsedEvent{
		Name:          name,
		TargetOrPhase: targetOrPhase,
		Type:          typ,
		Handler:       handler,
		SourceSpan:    sourceSpan,
		HandlerSpan:   handlerSpan,
		KeySpan:       keySpan,
	}
}

// ParsedVariable represents a variable declaration in a microsyntax expression
type ParsedVariable struct {
	Name       string
	Value      string
	SourceSpan *util.ParseSourceSpan
	KeySpan    *util.ParseSourceSpan
	ValueSpan  *util.ParseSourceSpan
}

// NewParsedVariable creates a new ParsedVariable
func NewParsedVariable(name, value string, sourceSpan, keySpan, valueSpan *util.ParseSourceSpan) *ParsedVariable {
	return &ParsedVariable{
		Name:       name,
		Value:      value,
		SourceSpan: sourceSpan,
		KeySpan:    keySpan,
		ValueSpan:  valueSpan,
	}
}

// BindingType represents the type of a binding
type BindingType int

const (
	BindingTypeProperty BindingType = iota
	BindingTypeAttribute
	BindingTypeClass
	BindingTypeStyle
	BindingTypeLegacyAnimation
	BindingTypeTwoWay
	BindingTypeAnimation
)

// BoundElementProperty represents a bound element property
type BoundElementProperty struct {
	Name            string
	Type            BindingType
	SecurityContext core.SecurityContext
	Value           *ASTWithSource
	Unit            *string
	SourceSpan      *util.ParseSourceSpan
	KeySpan         *util.ParseSourceSpan
	ValueSpan       *util.ParseSourceSpan
}

// NewBoundElementProperty creates a new BoundElementProperty
func NewBoundElementProperty(name string, typ BindingType, securityContext core.SecurityContext, value *ASTWithSource, unit *string, sourceSpan, keySpan, valueSpan *util.ParseSourceSpan) *BoundElementProperty {
	return &BoundElementProperty{
		Name:            name,
		Type:            typ,
		SecurityContext: securityContext,
		Value:           value,
		Unit:            unit,
		SourceSpan:      sourceSpan,
		KeySpan:         keySpan,
		ValueSpan:       valueSpan,
	}
}
