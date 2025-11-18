package output

import (
	"strings"

	"ngc-go/packages/compiler/util"
)

// TypeModifier represents type modifiers
type TypeModifier int

const (
	TypeModifierNone  TypeModifier = 0
	TypeModifierConst TypeModifier = 1 << 0
)

// Type is the base interface for all types
type Type interface {
	VisitType(visitor TypeVisitor, context interface{}) interface{}
	HasModifier(modifier TypeModifier) bool
}

// BuiltinTypeName represents builtin type names
type BuiltinTypeName int

const (
	BuiltinTypeNameDynamic BuiltinTypeName = iota
	BuiltinTypeNameBool
	BuiltinTypeNameString
	BuiltinTypeNameInt
	BuiltinTypeNameNumber
	BuiltinTypeNameFunction
	BuiltinTypeNameInferred
	BuiltinTypeNameNone
)

// BuiltinType represents a builtin type
type BuiltinType struct {
	Name      BuiltinTypeName
	Modifiers TypeModifier
}

// NewBuiltinType creates a new BuiltinType
func NewBuiltinType(name BuiltinTypeName, modifiers TypeModifier) *BuiltinType {
	return &BuiltinType{
		Name:      name,
		Modifiers: modifiers,
	}
}

// VisitType implements Type interface
func (b *BuiltinType) VisitType(visitor TypeVisitor, context interface{}) interface{} {
	return visitor.VisitBuiltinType(b, context)
}

// HasModifier checks if the type has a modifier
func (b *BuiltinType) HasModifier(modifier TypeModifier) bool {
	return (b.Modifiers & modifier) != 0
}

// ExpressionType represents an expression type
type ExpressionType struct {
	Value      OutputExpression
	Modifiers  TypeModifier
	TypeParams []Type
}

// NewExpressionType creates a new ExpressionType
func NewExpressionType(value OutputExpression, modifiers TypeModifier, typeParams []Type) *ExpressionType {
	return &ExpressionType{
		Value:      value,
		Modifiers:  modifiers,
		TypeParams: typeParams,
	}
}

// VisitType implements Type interface
func (e *ExpressionType) VisitType(visitor TypeVisitor, context interface{}) interface{} {
	return visitor.VisitExpressionType(e, context)
}

// HasModifier checks if the type has a modifier
func (e *ExpressionType) HasModifier(modifier TypeModifier) bool {
	return (e.Modifiers & modifier) != 0
}

// ArrayType represents an array type
type ArrayType struct {
	Of        Type
	Modifiers TypeModifier
}

// NewArrayType creates a new ArrayType
func NewArrayType(of Type, modifiers TypeModifier) *ArrayType {
	return &ArrayType{
		Of:        of,
		Modifiers: modifiers,
	}
}

// VisitType implements Type interface
func (a *ArrayType) VisitType(visitor TypeVisitor, context interface{}) interface{} {
	return visitor.VisitArrayType(a, context)
}

// HasModifier checks if the type has a modifier
func (a *ArrayType) HasModifier(modifier TypeModifier) bool {
	return (a.Modifiers & modifier) != 0
}

// MapType represents a map type
type MapType struct {
	ValueType *Type
	Modifiers TypeModifier
}

// NewMapType creates a new MapType
func NewMapType(valueType *Type, modifiers TypeModifier) *MapType {
	return &MapType{
		ValueType: valueType,
		Modifiers: modifiers,
	}
}

// VisitType implements Type interface
func (m *MapType) VisitType(visitor TypeVisitor, context interface{}) interface{} {
	return visitor.VisitMapType(m, context)
}

// HasModifier checks if the type has a modifier
func (m *MapType) HasModifier(modifier TypeModifier) bool {
	return (m.Modifiers & modifier) != 0
}

// TransplantedType represents a transplanted type
type TransplantedType struct {
	Type      interface{}
	Modifiers TypeModifier
}

// NewTransplantedType creates a new TransplantedType
func NewTransplantedType(typ interface{}, modifiers TypeModifier) *TransplantedType {
	return &TransplantedType{
		Type:      typ,
		Modifiers: modifiers,
	}
}

// VisitType implements Type interface
func (t *TransplantedType) VisitType(visitor TypeVisitor, context interface{}) interface{} {
	return visitor.VisitTransplantedType(t, context)
}

// HasModifier checks if the type has a modifier
func (t *TransplantedType) HasModifier(modifier TypeModifier) bool {
	return (t.Modifiers & modifier) != 0
}

// TypeVisitor is the interface for visiting types
type TypeVisitor interface {
	VisitBuiltinType(typ *BuiltinType, context interface{}) interface{}
	VisitExpressionType(typ *ExpressionType, context interface{}) interface{}
	VisitArrayType(typ *ArrayType, context interface{}) interface{}
	VisitMapType(typ *MapType, context interface{}) interface{}
	VisitTransplantedType(typ *TransplantedType, context interface{}) interface{}
}

// Predefined type constants
var (
	DynamicType  = NewBuiltinType(BuiltinTypeNameDynamic, TypeModifierNone)
	InferredType = NewBuiltinType(BuiltinTypeNameInferred, TypeModifierNone)
	BoolType     = NewBuiltinType(BuiltinTypeNameBool, TypeModifierNone)
	IntType      = NewBuiltinType(BuiltinTypeNameInt, TypeModifierNone)
	NumberType   = NewBuiltinType(BuiltinTypeNameNumber, TypeModifierNone)
	StringType   = NewBuiltinType(BuiltinTypeNameString, TypeModifierNone)
	FunctionType = NewBuiltinType(BuiltinTypeNameFunction, TypeModifierNone)
	NoneType     = NewBuiltinType(BuiltinTypeNameNone, TypeModifierNone)
)

// UnaryOperator represents unary operators
type UnaryOperator int

const (
	UnaryOperatorMinus UnaryOperator = iota
	UnaryOperatorPlus
)

// BinaryOperator represents binary operators
type BinaryOperator int

const (
	BinaryOperatorEquals BinaryOperator = iota
	BinaryOperatorNotEquals
	BinaryOperatorAssign
	BinaryOperatorIdentical
	BinaryOperatorNotIdentical
	BinaryOperatorMinus
	BinaryOperatorPlus
	BinaryOperatorDivide
	BinaryOperatorMultiply
	BinaryOperatorModulo
	BinaryOperatorAnd
	BinaryOperatorOr
	BinaryOperatorBitwiseOr
	BinaryOperatorBitwiseAnd
	BinaryOperatorLower
	BinaryOperatorLowerEquals
	BinaryOperatorBigger
	BinaryOperatorBiggerEquals
	BinaryOperatorNullishCoalesce
	BinaryOperatorExponentiation
	BinaryOperatorIn
	BinaryOperatorAdditionAssignment
	BinaryOperatorSubtractionAssignment
	BinaryOperatorMultiplicationAssignment
	BinaryOperatorDivisionAssignment
	BinaryOperatorRemainderAssignment
	BinaryOperatorExponentiationAssignment
	BinaryOperatorAndAssignment
	BinaryOperatorOrAssignment
	BinaryOperatorNullishCoalesceAssignment
)

// OutputExpression represents an expression in the output AST
// This interface extends the placeholder from constant_pool.go
type OutputExpression interface {
	GetType() Type
	GetSourceSpan() *util.ParseSourceSpan
	VisitExpression(visitor ExpressionVisitor, context interface{}) interface{}
	IsEquivalent(e OutputExpression) bool
	IsConstant() bool
	Clone() OutputExpression
}

// ExpressionVisitor is the interface for visiting expressions
type ExpressionVisitor interface {
	VisitReadVarExpr(ast *ReadVarExpr, context interface{}) interface{}
	VisitInvokeFunctionExpr(ast *InvokeFunctionExpr, context interface{}) interface{}
	VisitTaggedTemplateLiteralExpr(ast *TaggedTemplateLiteralExpr, context interface{}) interface{}
	VisitTemplateLiteralExpr(ast *TemplateLiteralExpr, context interface{}) interface{}
	VisitTemplateLiteralElementExpr(ast *TemplateLiteralElementExpr, context interface{}) interface{}
	VisitInstantiateExpr(ast *InstantiateExpr, context interface{}) interface{}
	VisitLiteralExpr(ast *LiteralExpr, context interface{}) interface{}
	VisitLocalizedString(ast *LocalizedString, context interface{}) interface{}
	VisitExternalExpr(ast *ExternalExpr, context interface{}) interface{}
	VisitConditionalExpr(ast *ConditionalExpr, context interface{}) interface{}
	VisitDynamicImportExpr(ast *DynamicImportExpr, context interface{}) interface{}
	VisitNotExpr(ast *NotExpr, context interface{}) interface{}
	VisitFunctionExpr(ast *FunctionExpr, context interface{}) interface{}
	VisitUnaryOperatorExpr(ast *UnaryOperatorExpr, context interface{}) interface{}
	VisitBinaryOperatorExpr(ast *BinaryOperatorExpr, context interface{}) interface{}
	VisitReadPropExpr(ast *ReadPropExpr, context interface{}) interface{}
	VisitReadKeyExpr(ast *ReadKeyExpr, context interface{}) interface{}
	VisitLiteralArrayExpr(ast *LiteralArrayExpr, context interface{}) interface{}
	VisitLiteralMapExpr(ast *LiteralMapExpr, context interface{}) interface{}
	VisitCommaExpr(ast *CommaExpr, context interface{}) interface{}
	VisitWrappedNodeExpr(ast *WrappedNodeExpr, context interface{}) interface{}
	VisitTypeofExpr(ast *TypeofExpr, context interface{}) interface{}
	VisitVoidExpr(ast *VoidExpr, context interface{}) interface{}
	VisitArrowFunctionExpr(ast *ArrowFunctionExpr, context interface{}) interface{}
	VisitParenthesizedExpr(ast *ParenthesizedExpr, context interface{}) interface{}
	VisitRegularExpressionLiteral(ast *RegularExpressionLiteralExpr, context interface{}) interface{}
}

// ExpressionBase is the base struct for all expressions
type ExpressionBase struct {
	Type       Type
	SourceSpan *util.ParseSourceSpan
}

// GetType returns the type of the expression
func (e *ExpressionBase) GetType() Type {
	return e.Type
}

// GetSourceSpan returns the source span
func (e *ExpressionBase) GetSourceSpan() *util.ParseSourceSpan {
	return e.SourceSpan
}

// ReadVarExpr represents a variable read expression
type ReadVarExpr struct {
	ExpressionBase
	Name string
}

// NewReadVarExpr creates a new ReadVarExpr
func NewReadVarExpr(name string, typ Type, sourceSpan *util.ParseSourceSpan) *ReadVarExpr {
	return &ReadVarExpr{
		ExpressionBase: ExpressionBase{
			Type:       typ,
			SourceSpan: sourceSpan,
		},
		Name: name,
	}
}

// VisitExpression implements OutputExpression interface
func (r *ReadVarExpr) VisitExpression(visitor ExpressionVisitor, context interface{}) interface{} {
	return visitor.VisitReadVarExpr(r, context)
}

// IsEquivalent checks if two expressions are equivalent
func (r *ReadVarExpr) IsEquivalent(e OutputExpression) bool {
	if other, ok := e.(*ReadVarExpr); ok {
		return r.Name == other.Name
	}
	return false
}

// IsConstant returns false for variable reads
func (r *ReadVarExpr) IsConstant() bool {
	return false
}

// Clone clones the expression
func (r *ReadVarExpr) Clone() OutputExpression {
	return NewReadVarExpr(r.Name, r.Type, r.SourceSpan)
}

// Set creates an assignment expression
func (r *ReadVarExpr) Set(value OutputExpression) *BinaryOperatorExpr {
	return NewBinaryOperatorExpr(
		BinaryOperatorAssign,
		r,
		value,
		r.Type,
		r.SourceSpan,
	)
}

// LiteralExpr represents a literal expression
type LiteralExpr struct {
	ExpressionBase
	Value interface{} // number | string | bool | nil
}

// NewLiteralExpr creates a new LiteralExpr
func NewLiteralExpr(value interface{}, typ Type, sourceSpan *util.ParseSourceSpan) *LiteralExpr {
	return &LiteralExpr{
		ExpressionBase: ExpressionBase{
			Type:       typ,
			SourceSpan: sourceSpan,
		},
		Value: value,
	}
}

// VisitExpression implements OutputExpression interface
func (l *LiteralExpr) VisitExpression(visitor ExpressionVisitor, context interface{}) interface{} {
	return visitor.VisitLiteralExpr(l, context)
}

// IsEquivalent checks if two expressions are equivalent
func (l *LiteralExpr) IsEquivalent(e OutputExpression) bool {
	if other, ok := e.(*LiteralExpr); ok {
		return l.Value == other.Value
	}
	return false
}

// IsConstant returns true for literals
func (l *LiteralExpr) IsConstant() bool {
	return true
}

// Clone clones the expression
func (l *LiteralExpr) Clone() OutputExpression {
	return NewLiteralExpr(l.Value, l.Type, l.SourceSpan)
}

// Predefined expressions
var (
	NullExpr      = NewLiteralExpr(nil, nil, nil)
	TypedNullExpr = NewLiteralExpr(nil, InferredType, nil)
)

// BinaryOperatorExpr represents a binary operator expression
type BinaryOperatorExpr struct {
	ExpressionBase
	Operator BinaryOperator
	Lhs      OutputExpression
	Rhs      OutputExpression
}

// NewBinaryOperatorExpr creates a new BinaryOperatorExpr
func NewBinaryOperatorExpr(operator BinaryOperator, lhs, rhs OutputExpression, typ Type, sourceSpan *util.ParseSourceSpan) *BinaryOperatorExpr {
	exprType := typ
	if exprType == nil && lhs != nil {
		exprType = lhs.GetType()
	}
	return &BinaryOperatorExpr{
		ExpressionBase: ExpressionBase{
			Type:       exprType,
			SourceSpan: sourceSpan,
		},
		Operator: operator,
		Lhs:      lhs,
		Rhs:      rhs,
	}
}

// VisitExpression implements OutputExpression interface
func (b *BinaryOperatorExpr) VisitExpression(visitor ExpressionVisitor, context interface{}) interface{} {
	return visitor.VisitBinaryOperatorExpr(b, context)
}

// IsEquivalent checks if two expressions are equivalent
func (b *BinaryOperatorExpr) IsEquivalent(e OutputExpression) bool {
	if other, ok := e.(*BinaryOperatorExpr); ok {
		return b.Operator == other.Operator &&
			b.Lhs.IsEquivalent(other.Lhs) &&
			b.Rhs.IsEquivalent(other.Rhs)
	}
	return false
}

// IsConstant returns false for binary operators
func (b *BinaryOperatorExpr) IsConstant() bool {
	return false
}

// Clone clones the expression
func (b *BinaryOperatorExpr) Clone() OutputExpression {
	return NewBinaryOperatorExpr(
		b.Operator,
		b.Lhs.Clone(),
		b.Rhs.Clone(),
		b.Type,
		b.SourceSpan,
	)
}

// IsAssignment checks if the operator is an assignment operator
func (b *BinaryOperatorExpr) IsAssignment() bool {
	return b.Operator == BinaryOperatorAssign ||
		b.Operator == BinaryOperatorAdditionAssignment ||
		b.Operator == BinaryOperatorSubtractionAssignment ||
		b.Operator == BinaryOperatorMultiplicationAssignment ||
		b.Operator == BinaryOperatorDivisionAssignment ||
		b.Operator == BinaryOperatorRemainderAssignment ||
		b.Operator == BinaryOperatorExponentiationAssignment ||
		b.Operator == BinaryOperatorAndAssignment ||
		b.Operator == BinaryOperatorOrAssignment ||
		b.Operator == BinaryOperatorNullishCoalesceAssignment
}

// Helper functions for null-safe equivalence checking
func NullSafeIsEquivalent(base, other interface{}) bool {
	if base == nil || other == nil {
		return base == other
	}
	if baseEq, ok := base.(interface{ IsEquivalent(interface{}) bool }); ok {
		return baseEq.IsEquivalent(other)
	}
	return false
}

func AreAllEquivalent(base, other []interface{}) bool {
	if len(base) != len(other) {
		return false
	}
	for i := 0; i < len(base); i++ {
		if !NullSafeIsEquivalent(base[i], other[i]) {
			return false
		}
	}
	return true
}

// Placeholder types - will be fully implemented later
// These are minimal implementations to satisfy the interface

type InvokeFunctionExpr struct {
	ExpressionBase
	Fn   OutputExpression
	Args []OutputExpression
	Pure bool
}

func NewInvokeFunctionExpr(fn OutputExpression, args []OutputExpression, typ Type, sourceSpan *util.ParseSourceSpan, pure bool) *InvokeFunctionExpr {
	return &InvokeFunctionExpr{
		ExpressionBase: ExpressionBase{Type: typ, SourceSpan: sourceSpan},
		Fn:             fn,
		Args:           args,
		Pure:           pure,
	}
}

func (i *InvokeFunctionExpr) VisitExpression(visitor ExpressionVisitor, context interface{}) interface{} {
	return visitor.VisitInvokeFunctionExpr(i, context)
}

func (i *InvokeFunctionExpr) IsEquivalent(e OutputExpression) bool {
	if other, ok := e.(*InvokeFunctionExpr); ok {
		return i.Fn.IsEquivalent(other.Fn) && areAllEquivalentExprs(i.Args, other.Args) && i.Pure == other.Pure
	}
	return false
}

func (i *InvokeFunctionExpr) IsConstant() bool {
	return false
}

func (i *InvokeFunctionExpr) Clone() OutputExpression {
	args := make([]OutputExpression, len(i.Args))
	for j, arg := range i.Args {
		args[j] = arg.Clone()
	}
	return NewInvokeFunctionExpr(i.Fn.Clone(), args, i.Type, i.SourceSpan, i.Pure)
}

func areAllEquivalentExprs(base, other []OutputExpression) bool {
	if len(base) != len(other) {
		return false
	}
	for i := 0; i < len(base); i++ {
		if !base[i].IsEquivalent(other[i]) {
			return false
		}
	}
	return true
}

type TaggedTemplateLiteralExpr struct {
	ExpressionBase
	Tag      OutputExpression
	Template *TemplateLiteralExpr
}

func NewTaggedTemplateLiteralExpr(tag OutputExpression, template *TemplateLiteralExpr, typ Type, sourceSpan *util.ParseSourceSpan) *TaggedTemplateLiteralExpr {
	return &TaggedTemplateLiteralExpr{
		ExpressionBase: ExpressionBase{Type: typ, SourceSpan: sourceSpan},
		Tag:            tag,
		Template:       template,
	}
}

func (t *TaggedTemplateLiteralExpr) VisitExpression(visitor ExpressionVisitor, context interface{}) interface{} {
	return visitor.VisitTaggedTemplateLiteralExpr(t, context)
}

func (t *TaggedTemplateLiteralExpr) IsEquivalent(e OutputExpression) bool {
	if other, ok := e.(*TaggedTemplateLiteralExpr); ok {
		return t.Tag.IsEquivalent(other.Tag) && t.Template.IsEquivalent(other.Template)
	}
	return false
}

func (t *TaggedTemplateLiteralExpr) IsConstant() bool {
	return false
}

func (t *TaggedTemplateLiteralExpr) Clone() OutputExpression {
	return NewTaggedTemplateLiteralExpr(t.Tag.Clone(), t.Template.Clone().(*TemplateLiteralExpr), t.Type, t.SourceSpan)
}

type TemplateLiteralExpr struct {
	ExpressionBase
	Elements    []*TemplateLiteralElementExpr
	Expressions []OutputExpression
}

func NewTemplateLiteralExpr(elements []*TemplateLiteralElementExpr, expressions []OutputExpression, sourceSpan *util.ParseSourceSpan) *TemplateLiteralExpr {
	return &TemplateLiteralExpr{
		ExpressionBase: ExpressionBase{Type: nil, SourceSpan: sourceSpan},
		Elements:       elements,
		Expressions:    expressions,
	}
}

func (t *TemplateLiteralExpr) VisitExpression(visitor ExpressionVisitor, context interface{}) interface{} {
	return visitor.VisitTemplateLiteralExpr(t, context)
}

func (t *TemplateLiteralExpr) IsEquivalent(e OutputExpression) bool {
	if other, ok := e.(*TemplateLiteralExpr); ok {
		if len(t.Elements) != len(other.Elements) || len(t.Expressions) != len(other.Expressions) {
			return false
		}
		for i := range t.Elements {
			if t.Elements[i].Text != other.Elements[i].Text {
				return false
			}
		}
		return areAllEquivalentExprs(t.Expressions, other.Expressions)
	}
	return false
}

func (t *TemplateLiteralExpr) IsConstant() bool {
	return false
}

func (t *TemplateLiteralExpr) Clone() OutputExpression {
	elements := make([]*TemplateLiteralElementExpr, len(t.Elements))
	for i, el := range t.Elements {
		elements[i] = el.Clone().(*TemplateLiteralElementExpr)
	}
	expressions := make([]OutputExpression, len(t.Expressions))
	for i, expr := range t.Expressions {
		expressions[i] = expr.Clone()
	}
	return NewTemplateLiteralExpr(elements, expressions, t.SourceSpan)
}

type TemplateLiteralElementExpr struct {
	ExpressionBase
	Text    string
	RawText string
}

func NewTemplateLiteralElementExpr(text string, sourceSpan *util.ParseSourceSpan, rawText string) *TemplateLiteralElementExpr {
	return &TemplateLiteralElementExpr{
		ExpressionBase: ExpressionBase{Type: StringType, SourceSpan: sourceSpan},
		Text:           text,
		RawText:        rawText,
	}
}

func (t *TemplateLiteralElementExpr) VisitExpression(visitor ExpressionVisitor, context interface{}) interface{} {
	return visitor.VisitTemplateLiteralElementExpr(t, context)
}

func (t *TemplateLiteralElementExpr) IsEquivalent(e OutputExpression) bool {
	if other, ok := e.(*TemplateLiteralElementExpr); ok {
		return t.Text == other.Text && t.RawText == other.RawText
	}
	return false
}

func (t *TemplateLiteralElementExpr) IsConstant() bool {
	return true
}

func (t *TemplateLiteralElementExpr) Clone() OutputExpression {
	return NewTemplateLiteralElementExpr(t.Text, t.SourceSpan, t.RawText)
}

type InstantiateExpr struct {
	ExpressionBase
	ClassExpr OutputExpression
	Args      []OutputExpression
}

func NewInstantiateExpr(classExpr OutputExpression, args []OutputExpression, typ Type, sourceSpan *util.ParseSourceSpan) *InstantiateExpr {
	return &InstantiateExpr{
		ExpressionBase: ExpressionBase{Type: typ, SourceSpan: sourceSpan},
		ClassExpr:      classExpr,
		Args:           args,
	}
}

func (i *InstantiateExpr) VisitExpression(visitor ExpressionVisitor, context interface{}) interface{} {
	return visitor.VisitInstantiateExpr(i, context)
}

func (i *InstantiateExpr) IsEquivalent(e OutputExpression) bool {
	if other, ok := e.(*InstantiateExpr); ok {
		return i.ClassExpr.IsEquivalent(other.ClassExpr) && areAllEquivalentExprs(i.Args, other.Args)
	}
	return false
}

func (i *InstantiateExpr) IsConstant() bool {
	return false
}

func (i *InstantiateExpr) Clone() OutputExpression {
	args := make([]OutputExpression, len(i.Args))
	for j, arg := range i.Args {
		args[j] = arg.Clone()
	}
	return NewInstantiateExpr(i.ClassExpr.Clone(), args, i.Type, i.SourceSpan)
}

type LocalizedString struct {
	ExpressionBase
	MetaBlock        *I18nMeta
	MessageParts     []*LiteralPiece
	PlaceholderNames []*PlaceholderPiece
	Expressions      []OutputExpression
}

func NewLocalizedString(
	metaBlock *I18nMeta,
	messageParts []*LiteralPiece,
	placeholderNames []*PlaceholderPiece,
	expressions []OutputExpression,
	sourceSpan *util.ParseSourceSpan,
) *LocalizedString {
	return &LocalizedString{
		ExpressionBase:   ExpressionBase{Type: StringType, SourceSpan: sourceSpan},
		MetaBlock:        metaBlock,
		MessageParts:     messageParts,
		PlaceholderNames: placeholderNames,
		Expressions:      expressions,
	}
}

func (l *LocalizedString) VisitExpression(visitor ExpressionVisitor, context interface{}) interface{} {
	return visitor.VisitLocalizedString(l, context)
}

func (l *LocalizedString) IsEquivalent(e OutputExpression) bool {
	return false // TODO: Implement
}

func (l *LocalizedString) IsConstant() bool {
	return false
}

func (l *LocalizedString) Clone() OutputExpression {
	clonedExpressions := make([]OutputExpression, len(l.Expressions))
	for i, expr := range l.Expressions {
		clonedExpressions[i] = expr.Clone()
	}
	return NewLocalizedString(
		l.MetaBlock,
		l.MessageParts,
		l.PlaceholderNames,
		clonedExpressions,
		l.SourceSpan,
	)
}

type ExternalExpr struct {
	ExpressionBase
	Value      *ExternalReference
	TypeParams []Type
}

type ExternalReference struct {
	ModuleName *string
	Name       *string
}

func NewExternalExpr(value *ExternalReference, typ Type, typeParams []Type, sourceSpan *util.ParseSourceSpan) *ExternalExpr {
	return &ExternalExpr{
		ExpressionBase: ExpressionBase{Type: typ, SourceSpan: sourceSpan},
		Value:          value,
		TypeParams:     typeParams,
	}
}

func (e *ExternalExpr) VisitExpression(visitor ExpressionVisitor, context interface{}) interface{} {
	return visitor.VisitExternalExpr(e, context)
}

func (e *ExternalExpr) IsEquivalent(other OutputExpression) bool {
	if o, ok := other.(*ExternalExpr); ok {
		return (e.Value.Name == o.Value.Name || (e.Value.Name != nil && o.Value.Name != nil && *e.Value.Name == *o.Value.Name)) &&
			(e.Value.ModuleName == o.Value.ModuleName || (e.Value.ModuleName != nil && o.Value.ModuleName != nil && *e.Value.ModuleName == *o.Value.ModuleName))
	}
	return false
}

func (e *ExternalExpr) IsConstant() bool {
	return false
}

func (e *ExternalExpr) Clone() OutputExpression {
	return NewExternalExpr(e.Value, e.Type, e.TypeParams, e.SourceSpan)
}

type ConditionalExpr struct {
	ExpressionBase
	Condition OutputExpression
	TrueCase  OutputExpression
	FalseCase OutputExpression
}

func NewConditionalExpr(condition, trueCase, falseCase OutputExpression, typ Type, sourceSpan *util.ParseSourceSpan) *ConditionalExpr {
	exprType := typ
	if exprType == nil && trueCase != nil {
		exprType = trueCase.GetType()
	}
	return &ConditionalExpr{
		ExpressionBase: ExpressionBase{Type: exprType, SourceSpan: sourceSpan},
		Condition:      condition,
		TrueCase:       trueCase,
		FalseCase:      falseCase,
	}
}

func (c *ConditionalExpr) VisitExpression(visitor ExpressionVisitor, context interface{}) interface{} {
	return visitor.VisitConditionalExpr(c, context)
}

func (c *ConditionalExpr) IsEquivalent(e OutputExpression) bool {
	if other, ok := e.(*ConditionalExpr); ok {
		return c.Condition.IsEquivalent(other.Condition) &&
			c.TrueCase.IsEquivalent(other.TrueCase) &&
			NullSafeIsEquivalent(c.FalseCase, other.FalseCase)
	}
	return false
}

func (c *ConditionalExpr) IsConstant() bool {
	return false
}

func (c *ConditionalExpr) Clone() OutputExpression {
	var falseCase OutputExpression
	if c.FalseCase != nil {
		falseCase = c.FalseCase.Clone()
	}
	return NewConditionalExpr(c.Condition.Clone(), c.TrueCase.Clone(), falseCase, c.Type, c.SourceSpan)
}

type DynamicImportExpr struct {
	ExpressionBase
	URL        interface{} // string | OutputExpression
	URLComment *string
}

func NewDynamicImportExpr(url interface{}, sourceSpan *util.ParseSourceSpan, urlComment *string) *DynamicImportExpr {
	return &DynamicImportExpr{
		ExpressionBase: ExpressionBase{Type: nil, SourceSpan: sourceSpan},
		URL:            url,
		URLComment:     urlComment,
	}
}

func (d *DynamicImportExpr) VisitExpression(visitor ExpressionVisitor, context interface{}) interface{} {
	return visitor.VisitDynamicImportExpr(d, context)
}

func (d *DynamicImportExpr) IsEquivalent(e OutputExpression) bool {
	if other, ok := e.(*DynamicImportExpr); ok {
		return d.URL == other.URL && (d.URLComment == other.URLComment || (d.URLComment != nil && other.URLComment != nil && *d.URLComment == *other.URLComment))
	}
	return false
}

func (d *DynamicImportExpr) IsConstant() bool {
	return false
}

func (d *DynamicImportExpr) Clone() OutputExpression {
	var url interface{}
	if str, ok := d.URL.(string); ok {
		url = str
	} else if expr, ok := d.URL.(OutputExpression); ok {
		url = expr.Clone()
	}
	return NewDynamicImportExpr(url, d.SourceSpan, d.URLComment)
}

type NotExpr struct {
	ExpressionBase
	Condition OutputExpression
}

func NewNotExpr(condition OutputExpression, sourceSpan *util.ParseSourceSpan) *NotExpr {
	return &NotExpr{
		ExpressionBase: ExpressionBase{Type: BoolType, SourceSpan: sourceSpan},
		Condition:      condition,
	}
}

func (n *NotExpr) VisitExpression(visitor ExpressionVisitor, context interface{}) interface{} {
	return visitor.VisitNotExpr(n, context)
}

func (n *NotExpr) IsEquivalent(e OutputExpression) bool {
	if other, ok := e.(*NotExpr); ok {
		return n.Condition.IsEquivalent(other.Condition)
	}
	return false
}

func (n *NotExpr) IsConstant() bool {
	return false
}

func (n *NotExpr) Clone() OutputExpression {
	return NewNotExpr(n.Condition.Clone(), n.SourceSpan)
}

type FunctionExpr struct {
	ExpressionBase
	Params     []*FnParam
	Statements []OutputStatement
	Name       *string
}

type FnParam struct {
	Name string
	Type Type
}

func NewFnParam(name string, typ Type) *FnParam {
	return &FnParam{Name: name, Type: typ}
}

func NewFunctionExpr(params []*FnParam, statements []OutputStatement, typ Type, sourceSpan *util.ParseSourceSpan, name *string) *FunctionExpr {
	return &FunctionExpr{
		ExpressionBase: ExpressionBase{Type: typ, SourceSpan: sourceSpan},
		Params:         params,
		Statements:     statements,
		Name:           name,
	}
}

func (f *FunctionExpr) VisitExpression(visitor ExpressionVisitor, context interface{}) interface{} {
	return visitor.VisitFunctionExpr(f, context)
}

func (f *FunctionExpr) IsEquivalent(e OutputExpression) bool {
	if fn, ok := e.(*FunctionExpr); ok {
		if len(f.Params) != len(fn.Params) || len(f.Statements) != len(fn.Statements) {
			return false
		}
		for i := range f.Params {
			if f.Params[i].Name != fn.Params[i].Name {
				return false
			}
		}
		// TODO: Compare statements properly
		return true
	}
	return false
}

// IsEquivalentToStmt checks if this FunctionExpr is equivalent to a DeclareFunctionStmt
func (f *FunctionExpr) IsEquivalentToStmt(stmt *DeclareFunctionStmt) bool {
	if len(f.Params) != len(stmt.Params) || len(f.Statements) != len(stmt.Statements) {
		return false
	}
	for i := range f.Params {
		if f.Params[i].Name != stmt.Params[i].Name {
			return false
		}
	}
	// TODO: Compare statements properly
	return true
}

func (f *FunctionExpr) IsConstant() bool {
	return false
}

func (f *FunctionExpr) Clone() OutputExpression {
	params := make([]*FnParam, len(f.Params))
	for i, p := range f.Params {
		params[i] = &FnParam{Name: p.Name, Type: p.Type}
	}
	return NewFunctionExpr(params, f.Statements, f.Type, f.SourceSpan, f.Name)
}

// ToDeclStmt converts a FunctionExpr to a DeclareFunctionStmt
func (f *FunctionExpr) ToDeclStmt(name string, modifiers StmtModifier) *DeclareFunctionStmt {
	return NewDeclareFunctionStmt(
		name,
		f.Params,
		f.Statements,
		f.Type,
		modifiers,
		f.SourceSpan,
		nil,
	)
}

type UnaryOperatorExpr struct {
	ExpressionBase
	Operator UnaryOperator
	Expr     OutputExpression
	Parens   bool
}

func NewUnaryOperatorExpr(operator UnaryOperator, expr OutputExpression, typ Type, sourceSpan *util.ParseSourceSpan, parens bool) *UnaryOperatorExpr {
	exprType := typ
	if exprType == nil {
		exprType = NumberType
	}
	return &UnaryOperatorExpr{
		ExpressionBase: ExpressionBase{Type: exprType, SourceSpan: sourceSpan},
		Operator:       operator,
		Expr:           expr,
		Parens:         parens,
	}
}

func (u *UnaryOperatorExpr) VisitExpression(visitor ExpressionVisitor, context interface{}) interface{} {
	return visitor.VisitUnaryOperatorExpr(u, context)
}

func (u *UnaryOperatorExpr) IsEquivalent(e OutputExpression) bool {
	if other, ok := e.(*UnaryOperatorExpr); ok {
		return u.Operator == other.Operator && u.Expr.IsEquivalent(other.Expr)
	}
	return false
}

func (u *UnaryOperatorExpr) IsConstant() bool {
	return false
}

func (u *UnaryOperatorExpr) Clone() OutputExpression {
	return NewUnaryOperatorExpr(u.Operator, u.Expr.Clone(), u.Type, u.SourceSpan, u.Parens)
}

type ReadPropExpr struct {
	ExpressionBase
	Receiver OutputExpression
	Name     string
}

func NewReadPropExpr(receiver OutputExpression, name string, typ Type, sourceSpan *util.ParseSourceSpan) *ReadPropExpr {
	return &ReadPropExpr{
		ExpressionBase: ExpressionBase{Type: typ, SourceSpan: sourceSpan},
		Receiver:       receiver,
		Name:           name,
	}
}

func (r *ReadPropExpr) VisitExpression(visitor ExpressionVisitor, context interface{}) interface{} {
	return visitor.VisitReadPropExpr(r, context)
}

func (r *ReadPropExpr) IsEquivalent(e OutputExpression) bool {
	if other, ok := e.(*ReadPropExpr); ok {
		return r.Receiver.IsEquivalent(other.Receiver) && r.Name == other.Name
	}
	return false
}

func (r *ReadPropExpr) IsConstant() bool {
	return false
}

func (r *ReadPropExpr) Clone() OutputExpression {
	return NewReadPropExpr(r.Receiver.Clone(), r.Name, r.Type, r.SourceSpan)
}

func (r *ReadPropExpr) Set(value OutputExpression) *BinaryOperatorExpr {
	return NewBinaryOperatorExpr(BinaryOperatorAssign, r, value, nil, r.SourceSpan)
}

type ReadKeyExpr struct {
	ExpressionBase
	Receiver OutputExpression
	Index    OutputExpression
}

func NewReadKeyExpr(receiver, index OutputExpression, typ Type, sourceSpan *util.ParseSourceSpan) *ReadKeyExpr {
	return &ReadKeyExpr{
		ExpressionBase: ExpressionBase{Type: typ, SourceSpan: sourceSpan},
		Receiver:       receiver,
		Index:          index,
	}
}

func (r *ReadKeyExpr) VisitExpression(visitor ExpressionVisitor, context interface{}) interface{} {
	return visitor.VisitReadKeyExpr(r, context)
}

func (r *ReadKeyExpr) IsEquivalent(e OutputExpression) bool {
	if other, ok := e.(*ReadKeyExpr); ok {
		return r.Receiver.IsEquivalent(other.Receiver) && r.Index.IsEquivalent(other.Index)
	}
	return false
}

func (r *ReadKeyExpr) IsConstant() bool {
	return false
}

func (r *ReadKeyExpr) Clone() OutputExpression {
	return NewReadKeyExpr(r.Receiver.Clone(), r.Index.Clone(), r.Type, r.SourceSpan)
}

func (r *ReadKeyExpr) Set(value OutputExpression) *BinaryOperatorExpr {
	return NewBinaryOperatorExpr(BinaryOperatorAssign, r, value, nil, r.SourceSpan)
}

type LiteralArrayExpr struct {
	ExpressionBase
	Entries []OutputExpression
}

func NewLiteralArrayExpr(entries []OutputExpression, typ Type, sourceSpan *util.ParseSourceSpan) *LiteralArrayExpr {
	return &LiteralArrayExpr{
		ExpressionBase: ExpressionBase{Type: typ, SourceSpan: sourceSpan},
		Entries:        entries,
	}
}

func (l *LiteralArrayExpr) VisitExpression(visitor ExpressionVisitor, context interface{}) interface{} {
	return visitor.VisitLiteralArrayExpr(l, context)
}

func (l *LiteralArrayExpr) IsEquivalent(e OutputExpression) bool {
	if other, ok := e.(*LiteralArrayExpr); ok {
		return areAllEquivalentExprs(l.Entries, other.Entries)
	}
	return false
}

func (l *LiteralArrayExpr) IsConstant() bool {
	for _, entry := range l.Entries {
		if !entry.IsConstant() {
			return false
		}
	}
	return true
}

func (l *LiteralArrayExpr) Clone() OutputExpression {
	entries := make([]OutputExpression, len(l.Entries))
	for i, entry := range l.Entries {
		entries[i] = entry.Clone()
	}
	return NewLiteralArrayExpr(entries, l.Type, l.SourceSpan)
}

type LiteralMapEntry struct {
	Key    string
	Value  OutputExpression
	Quoted bool
}

func NewLiteralMapEntry(key string, value OutputExpression, quoted bool) *LiteralMapEntry {
	return &LiteralMapEntry{Key: key, Value: value, Quoted: quoted}
}

func (l *LiteralMapEntry) IsEquivalent(e *LiteralMapEntry) bool {
	return l.Key == e.Key && l.Value.IsEquivalent(e.Value)
}

func (l *LiteralMapEntry) Clone() *LiteralMapEntry {
	return NewLiteralMapEntry(l.Key, l.Value.Clone(), l.Quoted)
}

type LiteralMapExpr struct {
	ExpressionBase
	Entries   []*LiteralMapEntry
	ValueType *Type
}

func NewLiteralMapExpr(entries []*LiteralMapEntry, typ *MapType, sourceSpan *util.ParseSourceSpan) *LiteralMapExpr {
	var valueType *Type
	if typ != nil {
		valueType = typ.ValueType
	}
	return &LiteralMapExpr{
		ExpressionBase: ExpressionBase{Type: typ, SourceSpan: sourceSpan},
		Entries:        entries,
		ValueType:      valueType,
	}
}

func (l *LiteralMapExpr) VisitExpression(visitor ExpressionVisitor, context interface{}) interface{} {
	return visitor.VisitLiteralMapExpr(l, context)
}

func (l *LiteralMapExpr) IsEquivalent(e OutputExpression) bool {
	if other, ok := e.(*LiteralMapExpr); ok {
		if len(l.Entries) != len(other.Entries) {
			return false
		}
		for i := range l.Entries {
			if !l.Entries[i].IsEquivalent(other.Entries[i]) {
				return false
			}
		}
		return true
	}
	return false
}

func (l *LiteralMapExpr) IsConstant() bool {
	for _, entry := range l.Entries {
		if !entry.Value.IsConstant() {
			return false
		}
	}
	return true
}

func (l *LiteralMapExpr) Clone() OutputExpression {
	entries := make([]*LiteralMapEntry, len(l.Entries))
	for i, entry := range l.Entries {
		entries[i] = entry.Clone()
	}
	var mapType *MapType
	if l.Type != nil {
		if mt, ok := l.Type.(*MapType); ok {
			mapType = mt
		}
	}
	return NewLiteralMapExpr(entries, mapType, l.SourceSpan)
}

type CommaExpr struct {
	ExpressionBase
	Parts []OutputExpression
}

func NewCommaExpr(parts []OutputExpression, sourceSpan *util.ParseSourceSpan) *CommaExpr {
	var typ Type
	if len(parts) > 0 {
		typ = parts[len(parts)-1].GetType()
	}
	return &CommaExpr{
		ExpressionBase: ExpressionBase{Type: typ, SourceSpan: sourceSpan},
		Parts:          parts,
	}
}

func (c *CommaExpr) VisitExpression(visitor ExpressionVisitor, context interface{}) interface{} {
	return visitor.VisitCommaExpr(c, context)
}

func (c *CommaExpr) IsEquivalent(e OutputExpression) bool {
	if other, ok := e.(*CommaExpr); ok {
		return areAllEquivalentExprs(c.Parts, other.Parts)
	}
	return false
}

func (c *CommaExpr) IsConstant() bool {
	return false
}

func (c *CommaExpr) Clone() OutputExpression {
	parts := make([]OutputExpression, len(c.Parts))
	for i, part := range c.Parts {
		parts[i] = part.Clone()
	}
	return NewCommaExpr(parts, c.SourceSpan)
}

type WrappedNodeExpr struct {
	ExpressionBase
	Node interface{}
}

func NewWrappedNodeExpr(node interface{}, typ Type, sourceSpan *util.ParseSourceSpan) *WrappedNodeExpr {
	return &WrappedNodeExpr{
		ExpressionBase: ExpressionBase{Type: typ, SourceSpan: sourceSpan},
		Node:           node,
	}
}

func (w *WrappedNodeExpr) VisitExpression(visitor ExpressionVisitor, context interface{}) interface{} {
	return visitor.VisitWrappedNodeExpr(w, context)
}

func (w *WrappedNodeExpr) IsEquivalent(e OutputExpression) bool {
	if other, ok := e.(*WrappedNodeExpr); ok {
		return w.Node == other.Node
	}
	return false
}

func (w *WrappedNodeExpr) IsConstant() bool {
	return false
}

func (w *WrappedNodeExpr) Clone() OutputExpression {
	return NewWrappedNodeExpr(w.Node, w.Type, w.SourceSpan)
}

type TypeofExpr struct {
	ExpressionBase
	Expr OutputExpression
}

func NewTypeofExpr(expr OutputExpression, typ Type, sourceSpan *util.ParseSourceSpan) *TypeofExpr {
	return &TypeofExpr{
		ExpressionBase: ExpressionBase{Type: typ, SourceSpan: sourceSpan},
		Expr:           expr,
	}
}

func (t *TypeofExpr) VisitExpression(visitor ExpressionVisitor, context interface{}) interface{} {
	return visitor.VisitTypeofExpr(t, context)
}

func (t *TypeofExpr) IsEquivalent(e OutputExpression) bool {
	if other, ok := e.(*TypeofExpr); ok {
		return t.Expr.IsEquivalent(other.Expr)
	}
	return false
}

func (t *TypeofExpr) IsConstant() bool {
	return t.Expr.IsConstant()
}

func (t *TypeofExpr) Clone() OutputExpression {
	return NewTypeofExpr(t.Expr.Clone(), t.Type, t.SourceSpan)
}

type VoidExpr struct {
	ExpressionBase
	Expr OutputExpression
}

func NewVoidExpr(expr OutputExpression, typ Type, sourceSpan *util.ParseSourceSpan) *VoidExpr {
	return &VoidExpr{
		ExpressionBase: ExpressionBase{Type: typ, SourceSpan: sourceSpan},
		Expr:           expr,
	}
}

func (v *VoidExpr) VisitExpression(visitor ExpressionVisitor, context interface{}) interface{} {
	return visitor.VisitVoidExpr(v, context)
}

func (v *VoidExpr) IsEquivalent(e OutputExpression) bool {
	if other, ok := e.(*VoidExpr); ok {
		return v.Expr.IsEquivalent(other.Expr)
	}
	return false
}

func (v *VoidExpr) IsConstant() bool {
	return v.Expr.IsConstant()
}

func (v *VoidExpr) Clone() OutputExpression {
	return NewVoidExpr(v.Expr.Clone(), v.Type, v.SourceSpan)
}

type ArrowFunctionExpr struct {
	ExpressionBase
	Params []*FnParam
	Body   interface{} // OutputExpression | []OutputStatement
}

func NewArrowFunctionExpr(params []*FnParam, body interface{}, typ Type, sourceSpan *util.ParseSourceSpan) *ArrowFunctionExpr {
	return &ArrowFunctionExpr{
		ExpressionBase: ExpressionBase{Type: typ, SourceSpan: sourceSpan},
		Params:         params,
		Body:           body,
	}
}

func (a *ArrowFunctionExpr) VisitExpression(visitor ExpressionVisitor, context interface{}) interface{} {
	return visitor.VisitArrowFunctionExpr(a, context)
}

func (a *ArrowFunctionExpr) IsEquivalent(e OutputExpression) bool {
	if other, ok := e.(*ArrowFunctionExpr); ok {
		if len(a.Params) != len(other.Params) {
			return false
		}
		for i := range a.Params {
			if a.Params[i].Name != other.Params[i].Name {
				return false
			}
		}
		// TODO: Compare body
		return true
	}
	return false
}

func (a *ArrowFunctionExpr) IsConstant() bool {
	return false
}

func (a *ArrowFunctionExpr) Clone() OutputExpression {
	params := make([]*FnParam, len(a.Params))
	for i, p := range a.Params {
		params[i] = &FnParam{Name: p.Name, Type: p.Type}
	}
	var body interface{}
	if expr, ok := a.Body.(OutputExpression); ok {
		body = expr.Clone()
	} else if stmts, ok := a.Body.([]OutputStatement); ok {
		body = stmts // TODO: Deep clone statements
	}
	return NewArrowFunctionExpr(params, body, a.Type, a.SourceSpan)
}

type ParenthesizedExpr struct {
	ExpressionBase
	Expr OutputExpression
}

func NewParenthesizedExpr(expr OutputExpression, typ Type, sourceSpan *util.ParseSourceSpan) *ParenthesizedExpr {
	return &ParenthesizedExpr{
		ExpressionBase: ExpressionBase{Type: typ, SourceSpan: sourceSpan},
		Expr:           expr,
	}
}

func (p *ParenthesizedExpr) VisitExpression(visitor ExpressionVisitor, context interface{}) interface{} {
	return visitor.VisitParenthesizedExpr(p, context)
}

func (p *ParenthesizedExpr) IsEquivalent(e OutputExpression) bool {
	if other, ok := e.(*ParenthesizedExpr); ok {
		return p.Expr.IsEquivalent(other.Expr)
	}
	return false
}

func (p *ParenthesizedExpr) IsConstant() bool {
	return p.Expr.IsConstant()
}

func (p *ParenthesizedExpr) Clone() OutputExpression {
	return NewParenthesizedExpr(p.Expr.Clone(), p.Type, p.SourceSpan)
}

type RegularExpressionLiteralExpr struct {
	ExpressionBase
	Body  string
	Flags *string
}

func NewRegularExpressionLiteralExpr(body string, flags *string, sourceSpan *util.ParseSourceSpan) *RegularExpressionLiteralExpr {
	return &RegularExpressionLiteralExpr{
		ExpressionBase: ExpressionBase{Type: nil, SourceSpan: sourceSpan},
		Body:           body,
		Flags:          flags,
	}
}

func (r *RegularExpressionLiteralExpr) VisitExpression(visitor ExpressionVisitor, context interface{}) interface{} {
	return visitor.VisitRegularExpressionLiteral(r, context)
}

func (r *RegularExpressionLiteralExpr) IsEquivalent(e OutputExpression) bool {
	if other, ok := e.(*RegularExpressionLiteralExpr); ok {
		return r.Body == other.Body && (r.Flags == other.Flags || (r.Flags != nil && other.Flags != nil && *r.Flags == *other.Flags))
	}
	return false
}

func (r *RegularExpressionLiteralExpr) IsConstant() bool {
	return true
}

func (r *RegularExpressionLiteralExpr) Clone() OutputExpression {
	return NewRegularExpressionLiteralExpr(r.Body, r.Flags, r.SourceSpan)
}

// StmtModifier represents statement modifiers
type StmtModifier int

const (
	StmtModifierNone     StmtModifier = 0
	StmtModifierFinal    StmtModifier = 1 << 0
	StmtModifierPrivate  StmtModifier = 1 << 1
	StmtModifierExported StmtModifier = 1 << 2
	StmtModifierStatic   StmtModifier = 1 << 3
)

// StatementVisitor is the interface for visiting statements
type StatementVisitor interface {
	VisitDeclareVarStmt(stmt *DeclareVarStmt, context interface{}) interface{}
	VisitDeclareFunctionStmt(stmt *DeclareFunctionStmt, context interface{}) interface{}
	VisitExpressionStmt(stmt *ExpressionStatement, context interface{}) interface{}
	VisitReturnStmt(stmt *ReturnStatement, context interface{}) interface{}
	VisitIfStmt(stmt *IfStmt, context interface{}) interface{}
}

// OutputStatement is a placeholder interface for statements
type OutputStatement interface {
	GetModifiers() StmtModifier
	GetSourceSpan() *util.ParseSourceSpan
	VisitStatement(visitor StatementVisitor, context interface{}) interface{}
	IsEquivalent(stmt OutputStatement) bool
}

// StatementBase is the base struct for all statements
type StatementBase struct {
	Modifiers       StmtModifier
	SourceSpan      *util.ParseSourceSpan
	LeadingComments []*LeadingComment
}

// GetModifiers returns the modifiers
func (s *StatementBase) GetModifiers() StmtModifier {
	return s.Modifiers
}

// GetSourceSpan returns the source span
func (s *StatementBase) GetSourceSpan() *util.ParseSourceSpan {
	return s.SourceSpan
}

// I18nMeta represents i18n metadata
type I18nMeta struct {
	ID          *string
	CustomID    *string
	LegacyIDs   []string
	Description *string
	Meaning     *string
}

// MessagePiece is a union type for message pieces
type MessagePiece interface {
	GetText() string
	GetSourceSpan() *util.ParseSourceSpan
}

// LiteralPiece represents a literal piece of text in a message
type LiteralPiece struct {
	Text       string
	SourceSpan *util.ParseSourceSpan
}

// NewLiteralPiece creates a new LiteralPiece
func NewLiteralPiece(text string, sourceSpan *util.ParseSourceSpan) *LiteralPiece {
	return &LiteralPiece{
		Text:       text,
		SourceSpan: sourceSpan,
	}
}

// GetText returns the text
func (l *LiteralPiece) GetText() string {
	return l.Text
}

// GetSourceSpan returns the source span
func (l *LiteralPiece) GetSourceSpan() *util.ParseSourceSpan {
	return l.SourceSpan
}

// PlaceholderPiece represents a placeholder piece in a message
type PlaceholderPiece struct {
	Text              string
	SourceSpan        *util.ParseSourceSpan
	AssociatedMessage interface{} // *i18n.Message
}

// NewPlaceholderPiece creates a new PlaceholderPiece
func NewPlaceholderPiece(text string, sourceSpan *util.ParseSourceSpan, associatedMessage interface{}) *PlaceholderPiece {
	return &PlaceholderPiece{
		Text:              text,
		SourceSpan:        sourceSpan,
		AssociatedMessage: associatedMessage,
	}
}

// GetText returns the text
func (p *PlaceholderPiece) GetText() string {
	return p.Text
}

// GetSourceSpan returns the source span
func (p *PlaceholderPiece) GetSourceSpan() *util.ParseSourceSpan {
	return p.SourceSpan
}

// JSDocTagName represents JSDoc tag names
type JSDocTagName int

const (
	JSDocTagNameDesc JSDocTagName = iota
	JSDocTagNameId
	JSDocTagNameMeaning
	JSDocTagNameSuppress
)

// JSDocTag represents a JSDoc tag
type JSDocTag struct {
	TagName *string // JSDocTagName as string or custom tag name
	Text    *string
}

// JSDocComment represents a JSDoc comment
type JSDocComment struct {
	Tags []JSDocTag
}

// NewJSDocComment creates a new JSDocComment
func NewJSDocComment(tags []JSDocTag) *JSDocComment {
	return &JSDocComment{
		Tags: tags,
	}
}

// String returns the string representation of the JSDoc comment
func (j *JSDocComment) String() string {
	if len(j.Tags) == 0 {
		return ""
	}

	result := "/**\n"
	for _, tag := range j.Tags {
		if tag.TagName != nil {
			result += " * @" + *tag.TagName
		}
		if tag.Text != nil {
			text := *tag.Text
			// Escape @ in text
			text = strings.ReplaceAll(text, "@", "\\@")
			result += " " + text
		}
		result += "\n"
	}
	result += " */"
	return result
}

// LeadingComment represents a leading comment
type LeadingComment struct {
	Text            string
	Multiline       bool
	TrailingNewline bool
}

// DeclareVarStmt represents a variable declaration statement
type DeclareVarStmt struct {
	StatementBase
	Name  string
	Value OutputExpression
	Type  Type
}

func NewDeclareVarStmt(name string, value OutputExpression, typ Type, modifiers StmtModifier, sourceSpan *util.ParseSourceSpan, leadingComments []*LeadingComment) *DeclareVarStmt {
	stmtType := typ
	if stmtType == nil && value != nil {
		stmtType = value.GetType()
	}
	return &DeclareVarStmt{
		StatementBase: StatementBase{
			Modifiers:       modifiers,
			SourceSpan:      sourceSpan,
			LeadingComments: leadingComments,
		},
		Name:  name,
		Value: value,
		Type:  stmtType,
	}
}

func (d *DeclareVarStmt) VisitStatement(visitor StatementVisitor, context interface{}) interface{} {
	return visitor.VisitDeclareVarStmt(d, context)
}

func (d *DeclareVarStmt) IsEquivalent(stmt OutputStatement) bool {
	if other, ok := stmt.(*DeclareVarStmt); ok {
		return d.Name == other.Name && (d.Value != nil && other.Value != nil && d.Value.IsEquivalent(other.Value) || d.Value == nil && other.Value == nil)
	}
	return false
}

// DeclareFunctionStmt represents a function declaration statement
type DeclareFunctionStmt struct {
	StatementBase
	Name       string
	Params     []*FnParam
	Statements []OutputStatement
	Type       Type
}

func NewDeclareFunctionStmt(name string, params []*FnParam, statements []OutputStatement, typ Type, modifiers StmtModifier, sourceSpan *util.ParseSourceSpan, leadingComments []*LeadingComment) *DeclareFunctionStmt {
	return &DeclareFunctionStmt{
		StatementBase: StatementBase{
			Modifiers:       modifiers,
			SourceSpan:      sourceSpan,
			LeadingComments: leadingComments,
		},
		Name:       name,
		Params:     params,
		Statements: statements,
		Type:       typ,
	}
}

func (d *DeclareFunctionStmt) VisitStatement(visitor StatementVisitor, context interface{}) interface{} {
	return visitor.VisitDeclareFunctionStmt(d, context)
}

func (d *DeclareFunctionStmt) IsEquivalent(stmt OutputStatement) bool {
	// TODO: Implement proper equivalence checking
	return false
}

// ExpressionStatement represents an expression statement
type ExpressionStatement struct {
	StatementBase
	Expr OutputExpression
}

func NewExpressionStatement(expr OutputExpression, sourceSpan *util.ParseSourceSpan, leadingComments []*LeadingComment) *ExpressionStatement {
	return &ExpressionStatement{
		StatementBase: StatementBase{
			Modifiers:       StmtModifierNone,
			SourceSpan:      sourceSpan,
			LeadingComments: leadingComments,
		},
		Expr: expr,
	}
}

func (e *ExpressionStatement) VisitStatement(visitor StatementVisitor, context interface{}) interface{} {
	return visitor.VisitExpressionStmt(e, context)
}

func (e *ExpressionStatement) IsEquivalent(stmt OutputStatement) bool {
	if other, ok := stmt.(*ExpressionStatement); ok {
		return e.Expr.IsEquivalent(other.Expr)
	}
	return false
}

// ReturnStatement represents a return statement
type ReturnStatement struct {
	StatementBase
	Value OutputExpression
}

func NewReturnStatement(value OutputExpression, sourceSpan *util.ParseSourceSpan, leadingComments []*LeadingComment) *ReturnStatement {
	return &ReturnStatement{
		StatementBase: StatementBase{
			Modifiers:       StmtModifierNone,
			SourceSpan:      sourceSpan,
			LeadingComments: leadingComments,
		},
		Value: value,
	}
}

func (r *ReturnStatement) VisitStatement(visitor StatementVisitor, context interface{}) interface{} {
	return visitor.VisitReturnStmt(r, context)
}

func (r *ReturnStatement) IsEquivalent(stmt OutputStatement) bool {
	if other, ok := stmt.(*ReturnStatement); ok {
		return r.Value.IsEquivalent(other.Value)
	}
	return false
}

// IfStmt represents an if statement
type IfStmt struct {
	StatementBase
	Condition OutputExpression
	TrueCase  []OutputStatement
	FalseCase []OutputStatement
}

func NewIfStmt(condition OutputExpression, trueCase, falseCase []OutputStatement, sourceSpan *util.ParseSourceSpan, leadingComments []*LeadingComment) *IfStmt {
	return &IfStmt{
		StatementBase: StatementBase{
			Modifiers:       StmtModifierNone,
			SourceSpan:      sourceSpan,
			LeadingComments: leadingComments,
		},
		Condition: condition,
		TrueCase:  trueCase,
		FalseCase: falseCase,
	}
}

func (i *IfStmt) VisitStatement(visitor StatementVisitor, context interface{}) interface{} {
	return visitor.VisitIfStmt(i, context)
}

func (i *IfStmt) IsEquivalent(stmt OutputStatement) bool {
	if other, ok := stmt.(*IfStmt); ok {
		// TODO: Implement proper equivalence checking for statements
		return i.Condition.IsEquivalent(other.Condition)
	}
	return false
}
