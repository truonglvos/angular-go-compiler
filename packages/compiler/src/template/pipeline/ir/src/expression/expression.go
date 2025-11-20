package expression

import (
	"fmt"

	"ngc-go/packages/compiler/src/output"
	"ngc-go/packages/compiler/src/render3"
	"ngc-go/packages/compiler/src/template/pipeline/ir"
	ir_operations "ngc-go/packages/compiler/src/template/pipeline/ir/src/operations"
	ops_create "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/create"
	"ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/shared"
	ops_update "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/update"
	ir_traits "ngc-go/packages/compiler/src/template/pipeline/ir/src/traits"
	"ngc-go/packages/compiler/src/util"
)

// ExpressionTransform is a transformer type which converts expressions into general `o.Expression`s
type ExpressionTransform func(expr output.OutputExpression, flags VisitorContextFlag) output.OutputExpression

// VisitorContextFlag represents flags for visitor context
type VisitorContextFlag int

const (
	// VisitorContextFlagNone - No flags
	VisitorContextFlagNone VisitorContextFlag = 0
	// VisitorContextFlagInChildOperation - In child operations
	VisitorContextFlagInChildOperation VisitorContextFlag = 0b0001
)

// IrExpression is an interface for IR expressions that can transform their internal expressions
type IrExpression interface {
	output.OutputExpression
	TransformInternalExpressions(transform ExpressionTransform, flags VisitorContextFlag)
}

// ExpressionBase is the base type used for all logical IR expressions
type ExpressionBase struct {
	Type       output.Type
	SourceSpan *util.ParseSourceSpan
	Kind       ir.ExpressionKind
}

// NewExpressionBase creates a new ExpressionBase
func NewExpressionBase(kind ir.ExpressionKind, sourceSpan *util.ParseSourceSpan) *ExpressionBase {
	return &ExpressionBase{
		Type:       nil,
		SourceSpan: sourceSpan,
		Kind:       kind,
	}
}

// GetType returns the type
func (e *ExpressionBase) GetType() output.Type {
	return e.Type
}

// GetSourceSpan returns the source span
func (e *ExpressionBase) GetSourceSpan() *util.ParseSourceSpan {
	return e.SourceSpan
}

// TransformInternalExpressions runs the transformer against any nested expressions
func (e *ExpressionBase) TransformInternalExpressions(
	transform ExpressionTransform,
	flags VisitorContextFlag,
) {
	// To be implemented by subclasses
}

// IsIrExpression checks whether a given `o.Expression` is a logical IR expression type
func IsIrExpression(expr output.OutputExpression) bool {
	_, ok := expr.(IrExpression)
	return ok
}

// LexicalReadExpr represents a lexical read of a variable name
type LexicalReadExpr struct {
	*ExpressionBase
	Name string
}

// NewLexicalReadExpr creates a new LexicalReadExpr
func NewLexicalReadExpr(name string) *LexicalReadExpr {
	return &LexicalReadExpr{
		ExpressionBase: NewExpressionBase(ir.ExpressionKindLexicalRead, nil),
		Name:           name,
	}
}

// VisitExpression implements OutputExpression interface
func (l *LexicalReadExpr) VisitExpression(visitor output.ExpressionVisitor, context interface{}) interface{} {
	return nil
}

// IsEquivalent checks if two expressions are equivalent
func (l *LexicalReadExpr) IsEquivalent(other output.OutputExpression) bool {
	if otherLex, ok := other.(*LexicalReadExpr); ok {
		return l.Name == otherLex.Name
	}
	return false
}

// IsConstant returns false for lexical reads
func (l *LexicalReadExpr) IsConstant() bool {
	return false
}

// Clone clones the expression
func (l *LexicalReadExpr) Clone() output.OutputExpression {
	return NewLexicalReadExpr(l.Name)
}

// TransformInternalExpressions transforms internal expressions
func (l *LexicalReadExpr) TransformInternalExpressions(transform ExpressionTransform, flags VisitorContextFlag) {
	// No nested expressions
}

// ReferenceExpr is a runtime operations to retrieve the value of a local reference
type ReferenceExpr struct {
	*ExpressionBase
	Target     ir_operations.XrefId
	TargetSlot *ir.SlotHandle
	Offset     int
}

// NewReferenceExpr creates a new ReferenceExpr
func NewReferenceExpr(target ir_operations.XrefId, targetSlot *ir.SlotHandle, offset int) *ReferenceExpr {
	return &ReferenceExpr{
		ExpressionBase: NewExpressionBase(ir.ExpressionKindReference, nil),
		Target:         target,
		TargetSlot:     targetSlot,
		Offset:         offset,
	}
}

// VisitExpression implements OutputExpression interface
func (r *ReferenceExpr) VisitExpression(visitor output.ExpressionVisitor, context interface{}) interface{} {
	return nil
}

// IsEquivalent checks if two expressions are equivalent
func (r *ReferenceExpr) IsEquivalent(other output.OutputExpression) bool {
	if otherRef, ok := other.(*ReferenceExpr); ok {
		return r.Target == otherRef.Target
	}
	return false
}

// IsConstant returns false
func (r *ReferenceExpr) IsConstant() bool {
	return false
}

// Clone clones the expression
func (r *ReferenceExpr) Clone() output.OutputExpression {
	return NewReferenceExpr(r.Target, r.TargetSlot, r.Offset)
}

// TransformInternalExpressions transforms internal expressions
func (r *ReferenceExpr) TransformInternalExpressions(transform ExpressionTransform, flags VisitorContextFlag) {
	// No nested expressions
}

// ContextExpr is a reference to the current view context
type ContextExpr struct {
	*ExpressionBase
	View ir_operations.XrefId
}

// NewContextExpr creates a new ContextExpr
func NewContextExpr(view ir_operations.XrefId) *ContextExpr {
	return &ContextExpr{
		ExpressionBase: NewExpressionBase(ir.ExpressionKindContext, nil),
		View:           view,
	}
}

// VisitExpression implements OutputExpression interface
func (c *ContextExpr) VisitExpression(visitor output.ExpressionVisitor, context interface{}) interface{} {
	return nil
}

// IsEquivalent checks if two expressions are equivalent
func (c *ContextExpr) IsEquivalent(other output.OutputExpression) bool {
	if otherCtx, ok := other.(*ContextExpr); ok {
		return c.View == otherCtx.View
	}
	return false
}

// IsConstant returns false
func (c *ContextExpr) IsConstant() bool {
	return false
}

// Clone clones the expression
func (c *ContextExpr) Clone() output.OutputExpression {
	return NewContextExpr(c.View)
}

// TransformInternalExpressions transforms internal expressions
func (c *ContextExpr) TransformInternalExpressions(transform ExpressionTransform, flags VisitorContextFlag) {
	// No nested expressions
}

// TrackContextExpr is a reference to the current view context inside a track function
type TrackContextExpr struct {
	*ExpressionBase
	View ir_operations.XrefId
}

// NewTrackContextExpr creates a new TrackContextExpr
func NewTrackContextExpr(view ir_operations.XrefId) *TrackContextExpr {
	return &TrackContextExpr{
		ExpressionBase: NewExpressionBase(ir.ExpressionKindTrackContext, nil),
		View:           view,
	}
}

// VisitExpression implements OutputExpression interface
func (t *TrackContextExpr) VisitExpression(visitor output.ExpressionVisitor, context interface{}) interface{} {
	return nil
}

// IsEquivalent checks if two expressions are equivalent
func (t *TrackContextExpr) IsEquivalent(other output.OutputExpression) bool {
	if otherTrack, ok := other.(*TrackContextExpr); ok {
		return t.View == otherTrack.View
	}
	return false
}

// IsConstant returns false
func (t *TrackContextExpr) IsConstant() bool {
	return false
}

// Clone clones the expression
func (t *TrackContextExpr) Clone() output.OutputExpression {
	return NewTrackContextExpr(t.View)
}

// TransformInternalExpressions transforms internal expressions
func (t *TrackContextExpr) TransformInternalExpressions(transform ExpressionTransform, flags VisitorContextFlag) {
	// No nested expressions
}

// NextContextExpr is a runtime operations to navigate to the next view context
type NextContextExpr struct {
	*ExpressionBase
	Steps int
}

// NewNextContextExpr creates a new NextContextExpr
func NewNextContextExpr() *NextContextExpr {
	return &NextContextExpr{
		ExpressionBase: NewExpressionBase(ir.ExpressionKindNextContext, nil),
		Steps:          1,
	}
}

// VisitExpression implements OutputExpression interface
func (n *NextContextExpr) VisitExpression(visitor output.ExpressionVisitor, context interface{}) interface{} {
	return nil
}

// IsEquivalent checks if two expressions are equivalent
func (n *NextContextExpr) IsEquivalent(other output.OutputExpression) bool {
	if otherNext, ok := other.(*NextContextExpr); ok {
		return n.Steps == otherNext.Steps
	}
	return false
}

// IsConstant returns false
func (n *NextContextExpr) IsConstant() bool {
	return false
}

// Clone clones the expression
func (n *NextContextExpr) Clone() output.OutputExpression {
	expr := NewNextContextExpr()
	expr.Steps = n.Steps
	return expr
}

// TransformInternalExpressions transforms internal expressions
func (n *NextContextExpr) TransformInternalExpressions(transform ExpressionTransform, flags VisitorContextFlag) {
	// No nested expressions
}

// GetCurrentViewExpr is a runtime operations to snapshot the current view context
type GetCurrentViewExpr struct {
	*ExpressionBase
}

// NewGetCurrentViewExpr creates a new GetCurrentViewExpr
func NewGetCurrentViewExpr() *GetCurrentViewExpr {
	return &GetCurrentViewExpr{
		ExpressionBase: NewExpressionBase(ir.ExpressionKindGetCurrentView, nil),
	}
}

// VisitExpression implements OutputExpression interface
func (g *GetCurrentViewExpr) VisitExpression(visitor output.ExpressionVisitor, context interface{}) interface{} {
	return nil
}

// IsEquivalent checks if two expressions are equivalent
func (g *GetCurrentViewExpr) IsEquivalent(other output.OutputExpression) bool {
	_, ok := other.(*GetCurrentViewExpr)
	return ok
}

// IsConstant returns false
func (g *GetCurrentViewExpr) IsConstant() bool {
	return false
}

// Clone clones the expression
func (g *GetCurrentViewExpr) Clone() output.OutputExpression {
	return NewGetCurrentViewExpr()
}

// TransformInternalExpressions transforms internal expressions
func (g *GetCurrentViewExpr) TransformInternalExpressions(transform ExpressionTransform, flags VisitorContextFlag) {
	// No nested expressions
}

// RestoreViewExpr is a runtime operations to restore a snapshotted view
type RestoreViewExpr struct {
	*ExpressionBase
	View interface{} // XrefId | output.OutputExpression
}

// NewRestoreViewExpr creates a new RestoreViewExpr
func NewRestoreViewExpr(view interface{}) *RestoreViewExpr {
	return &RestoreViewExpr{
		ExpressionBase: NewExpressionBase(ir.ExpressionKindRestoreView, nil),
		View:           view,
	}
}

// VisitExpression implements OutputExpression interface
func (r *RestoreViewExpr) VisitExpression(visitor output.ExpressionVisitor, context interface{}) interface{} {
	if expr, ok := r.View.(output.OutputExpression); ok {
		return expr.VisitExpression(visitor, context)
	}
	return nil
}

// IsEquivalent checks if two expressions are equivalent
func (r *RestoreViewExpr) IsEquivalent(other output.OutputExpression) bool {
	otherRestore, ok := other.(*RestoreViewExpr)
	if !ok {
		return false
	}

	// Check if both views are of the same type
	if (r.View == nil) != (otherRestore.View == nil) {
		return false
	}

	if xrefId, ok := r.View.(ir_operations.XrefId); ok {
		if otherXrefId, ok := otherRestore.View.(ir_operations.XrefId); ok {
			return xrefId == otherXrefId
		}
		return false
	}

	if expr, ok := r.View.(output.OutputExpression); ok {
		if otherExpr, ok := otherRestore.View.(output.OutputExpression); ok {
			return expr.IsEquivalent(otherExpr)
		}
		return false
	}

	return false
}

// IsConstant returns false
func (r *RestoreViewExpr) IsConstant() bool {
	return false
}

// Clone clones the expression
func (r *RestoreViewExpr) Clone() output.OutputExpression {
	var clonedView interface{}
	if expr, ok := r.View.(output.OutputExpression); ok {
		clonedView = expr.Clone()
	} else {
		clonedView = r.View
	}
	return NewRestoreViewExpr(clonedView)
}

// TransformInternalExpressions transforms internal expressions
func (r *RestoreViewExpr) TransformInternalExpressions(transform ExpressionTransform, flags VisitorContextFlag) {
	if expr, ok := r.View.(output.OutputExpression); ok {
		r.View = TransformExpressionsInExpression(expr, transform, flags)
	}
}

// ResetViewExpr is a runtime operations to reset the current view context after `RestoreView`
type ResetViewExpr struct {
	*ExpressionBase
	Expr output.OutputExpression
}

// NewResetViewExpr creates a new ResetViewExpr
func NewResetViewExpr(expr output.OutputExpression) *ResetViewExpr {
	return &ResetViewExpr{
		ExpressionBase: NewExpressionBase(ir.ExpressionKindResetView, nil),
		Expr:           expr,
	}
}

// VisitExpression implements OutputExpression interface
func (r *ResetViewExpr) VisitExpression(visitor output.ExpressionVisitor, context interface{}) interface{} {
	return r.Expr.VisitExpression(visitor, context)
}

// IsEquivalent checks if two expressions are equivalent
func (r *ResetViewExpr) IsEquivalent(other output.OutputExpression) bool {
	if otherReset, ok := other.(*ResetViewExpr); ok {
		return r.Expr.IsEquivalent(otherReset.Expr)
	}
	return false
}

// IsConstant returns false
func (r *ResetViewExpr) IsConstant() bool {
	return false
}

// Clone clones the expression
func (r *ResetViewExpr) Clone() output.OutputExpression {
	return NewResetViewExpr(r.Expr.Clone())
}

// TransformInternalExpressions transforms internal expressions
func (r *ResetViewExpr) TransformInternalExpressions(transform ExpressionTransform, flags VisitorContextFlag) {
	r.Expr = TransformExpressionsInExpression(r.Expr, transform, flags)
}

// ReadVariableExpr is a read of a variable declared as an `ir.VariableOp`
type ReadVariableExpr struct {
	*ExpressionBase
	Xref ir_operations.XrefId
	Name *string
}

// NewReadVariableExpr creates a new ReadVariableExpr
func NewReadVariableExpr(xref ir_operations.XrefId) *ReadVariableExpr {
	return &ReadVariableExpr{
		ExpressionBase: NewExpressionBase(ir.ExpressionKindReadVariable, nil),
		Xref:           xref,
		Name:           nil,
	}
}

// VisitExpression implements OutputExpression interface
func (r *ReadVariableExpr) VisitExpression(visitor output.ExpressionVisitor, context interface{}) interface{} {
	return nil
}

// IsEquivalent checks if two expressions are equivalent
func (r *ReadVariableExpr) IsEquivalent(other output.OutputExpression) bool {
	if otherRead, ok := other.(*ReadVariableExpr); ok {
		return r.Xref == otherRead.Xref
	}
	return false
}

// IsConstant returns false
func (r *ReadVariableExpr) IsConstant() bool {
	return false
}

// Clone clones the expression
func (r *ReadVariableExpr) Clone() output.OutputExpression {
	expr := NewReadVariableExpr(r.Xref)
	expr.Name = r.Name
	return expr
}

// TransformInternalExpressions transforms internal expressions
func (r *ReadVariableExpr) TransformInternalExpressions(transform ExpressionTransform, flags VisitorContextFlag) {
	// No nested expressions
}

// EmptyExpr is an empty expression that will be stripped before generating the final output
type EmptyExpr struct {
	*ExpressionBase
}

// NewEmptyExpr creates a new EmptyExpr
func NewEmptyExpr(sourceSpan *util.ParseSourceSpan) *EmptyExpr {
	return &EmptyExpr{
		ExpressionBase: NewExpressionBase(ir.ExpressionKindEmptyExpr, sourceSpan),
	}
}

// VisitExpression implements OutputExpression interface
func (e *EmptyExpr) VisitExpression(visitor output.ExpressionVisitor, context interface{}) interface{} {
	return nil
}

// IsEquivalent checks if two expressions are equivalent
func (e *EmptyExpr) IsEquivalent(other output.OutputExpression) bool {
	_, ok := other.(*EmptyExpr)
	return ok
}

// IsConstant returns true
func (e *EmptyExpr) IsConstant() bool {
	return true
}

// Clone clones the expression
func (e *EmptyExpr) Clone() output.OutputExpression {
	return NewEmptyExpr(e.SourceSpan)
}

// TransformInternalExpressions transforms internal expressions
func (e *EmptyExpr) TransformInternalExpressions(transform ExpressionTransform, flags VisitorContextFlag) {
	// No nested expressions
}

// TransformExpressionsInExpression transforms all `Expression`s in the AST of `expr` with the `transform` function
func TransformExpressionsInExpression(
	expr output.OutputExpression,
	transform ExpressionTransform,
	flags VisitorContextFlag,
) output.OutputExpression {
	if irExpr, ok := expr.(IrExpression); ok {
		irExpr.TransformInternalExpressions(transform, flags)
	} else if binary, ok := expr.(*output.BinaryOperatorExpr); ok {
		binary.Lhs = TransformExpressionsInExpression(binary.Lhs, transform, flags)
		binary.Rhs = TransformExpressionsInExpression(binary.Rhs, transform, flags)
	} else if unary, ok := expr.(*output.UnaryOperatorExpr); ok {
		unary.Expr = TransformExpressionsInExpression(unary.Expr, transform, flags)
	} else if readProp, ok := expr.(*output.ReadPropExpr); ok {
		readProp.Receiver = TransformExpressionsInExpression(readProp.Receiver, transform, flags)
	} else if readKey, ok := expr.(*output.ReadKeyExpr); ok {
		readKey.Receiver = TransformExpressionsInExpression(readKey.Receiver, transform, flags)
		readKey.Index = TransformExpressionsInExpression(readKey.Index, transform, flags)
	} else if invoke, ok := expr.(*output.InvokeFunctionExpr); ok {
		invoke.Fn = TransformExpressionsInExpression(invoke.Fn, transform, flags)
		for i := 0; i < len(invoke.Args); i++ {
			invoke.Args[i] = TransformExpressionsInExpression(invoke.Args[i], transform, flags)
		}
	} else if literalArray, ok := expr.(*output.LiteralArrayExpr); ok {
		for i := 0; i < len(literalArray.Entries); i++ {
			literalArray.Entries[i] = TransformExpressionsInExpression(literalArray.Entries[i], transform, flags)
		}
	} else if literalMap, ok := expr.(*output.LiteralMapExpr); ok {
		for i := 0; i < len(literalMap.Entries); i++ {
			literalMap.Entries[i].Value = TransformExpressionsInExpression(literalMap.Entries[i].Value, transform, flags)
		}
	} else if conditional, ok := expr.(*output.ConditionalExpr); ok {
		conditional.Condition = TransformExpressionsInExpression(conditional.Condition, transform, flags)
		conditional.TrueCase = TransformExpressionsInExpression(conditional.TrueCase, transform, flags)
		if conditional.FalseCase != nil {
			conditional.FalseCase = TransformExpressionsInExpression(conditional.FalseCase, transform, flags)
		}
	} else if typeof, ok := expr.(*output.TypeofExpr); ok {
		typeof.Expr = TransformExpressionsInExpression(typeof.Expr, transform, flags)
	} else if void, ok := expr.(*output.VoidExpr); ok {
		void.Expr = TransformExpressionsInExpression(void.Expr, transform, flags)
	} else if not, ok := expr.(*output.NotExpr); ok {
		not.Condition = TransformExpressionsInExpression(not.Condition, transform, flags)
	} else if parenthesized, ok := expr.(*output.ParenthesizedExpr); ok {
		parenthesized.Expr = TransformExpressionsInExpression(parenthesized.Expr, transform, flags)
	}
	// Other expression types (ReadVarExpr, ExternalExpr, LiteralExpr, RegularExpressionLiteralExpr) don't need transformation
	return transform(expr, flags)
}

// IsStringLiteral checks whether the given expression is a string literal
func IsStringLiteral(expr output.OutputExpression) bool {
	if literal, ok := expr.(*output.LiteralExpr); ok {
		_, ok := literal.Value.(string)
		return ok
	}
	return false
}

// TwoWayBindingSetExpr is an operations that sets the value of a two-way binding
type TwoWayBindingSetExpr struct {
	*ExpressionBase
	Target output.OutputExpression
	Value  output.OutputExpression
}

// NewTwoWayBindingSetExpr creates a new TwoWayBindingSetExpr
func NewTwoWayBindingSetExpr(target, value output.OutputExpression) *TwoWayBindingSetExpr {
	return &TwoWayBindingSetExpr{
		ExpressionBase: NewExpressionBase(ir.ExpressionKindTwoWayBindingSet, nil),
		Target:         target,
		Value:          value,
	}
}

// VisitExpression implements OutputExpression interface
func (t *TwoWayBindingSetExpr) VisitExpression(visitor output.ExpressionVisitor, context interface{}) interface{} {
	t.Target.VisitExpression(visitor, context)
	t.Value.VisitExpression(visitor, context)
	return nil
}

// IsEquivalent checks if two expressions are equivalent
func (t *TwoWayBindingSetExpr) IsEquivalent(other output.OutputExpression) bool {
	if otherTwoWay, ok := other.(*TwoWayBindingSetExpr); ok {
		return t.Target.IsEquivalent(otherTwoWay.Target) && t.Value.IsEquivalent(otherTwoWay.Value)
	}
	return false
}

// IsConstant returns false
func (t *TwoWayBindingSetExpr) IsConstant() bool {
	return false
}

// Clone clones the expression
func (t *TwoWayBindingSetExpr) Clone() output.OutputExpression {
	return NewTwoWayBindingSetExpr(t.Target.Clone(), t.Value.Clone())
}

// TransformInternalExpressions transforms internal expressions
func (t *TwoWayBindingSetExpr) TransformInternalExpressions(transform ExpressionTransform, flags VisitorContextFlag) {
	t.Target = TransformExpressionsInExpression(t.Target, transform, flags)
	t.Value = TransformExpressionsInExpression(t.Value, transform, flags)
}

// PureFunctionExpr defines and calls a function with change-detected arguments
type PureFunctionExpr struct {
	*ExpressionBase
	VarOffset *int
	Body      output.OutputExpression
	Args      []output.OutputExpression
	Fn        output.OutputExpression
}

// NewPureFunctionExpr creates a new PureFunctionExpr
func NewPureFunctionExpr(expression output.OutputExpression, args []output.OutputExpression) *PureFunctionExpr {
	return &PureFunctionExpr{
		ExpressionBase: NewExpressionBase(ir.ExpressionKindPureFunctionExpr, nil),
		VarOffset:      nil,
		Body:           expression,
		Args:           args,
		Fn:             nil,
	}
}

// HasConsumesVarsTrait implements ConsumesVarsTraitInterface
func (p *PureFunctionExpr) HasConsumesVarsTrait() bool {
	return true
}

// GetVarOffset implements UsesVarOffsetTraitInterface
func (p *PureFunctionExpr) GetVarOffset() *int {
	return p.VarOffset
}

// SetVarOffset implements UsesVarOffsetTraitInterface
func (p *PureFunctionExpr) SetVarOffset(offset int) {
	p.VarOffset = &offset
}

// VisitExpression implements OutputExpression interface
func (p *PureFunctionExpr) VisitExpression(visitor output.ExpressionVisitor, context interface{}) interface{} {
	if p.Body != nil {
		p.Body.VisitExpression(visitor, context)
	}
	for _, arg := range p.Args {
		arg.VisitExpression(visitor, context)
	}
	return nil
}

// IsEquivalent checks if two expressions are equivalent
func (p *PureFunctionExpr) IsEquivalent(other output.OutputExpression) bool {
	otherPure, ok := other.(*PureFunctionExpr)
	if !ok || len(otherPure.Args) != len(p.Args) {
		return false
	}
	if otherPure.Body == nil || p.Body == nil {
		return false
	}
	if !otherPure.Body.IsEquivalent(p.Body) {
		return false
	}
	for i, arg := range p.Args {
		if !arg.IsEquivalent(otherPure.Args[i]) {
			return false
		}
	}
	return true
}

// IsConstant returns false
func (p *PureFunctionExpr) IsConstant() bool {
	return false
}

// Clone clones the expression
func (p *PureFunctionExpr) Clone() output.OutputExpression {
	clonedArgs := make([]output.OutputExpression, len(p.Args))
	for i, arg := range p.Args {
		clonedArgs[i] = arg.Clone()
	}
	var clonedBody output.OutputExpression
	if p.Body != nil {
		clonedBody = p.Body.Clone()
	}
	expr := NewPureFunctionExpr(clonedBody, clonedArgs)
	if p.Fn != nil {
		expr.Fn = p.Fn.Clone()
	}
	expr.VarOffset = p.VarOffset
	return expr
}

// TransformInternalExpressions transforms internal expressions
func (p *PureFunctionExpr) TransformInternalExpressions(transform ExpressionTransform, flags VisitorContextFlag) {
	if p.Body != nil {
		p.Body = TransformExpressionsInExpression(p.Body, transform, flags|VisitorContextFlagInChildOperation)
	} else if p.Fn != nil {
		p.Fn = TransformExpressionsInExpression(p.Fn, transform, flags)
	}
	for i := 0; i < len(p.Args); i++ {
		p.Args[i] = TransformExpressionsInExpression(p.Args[i], transform, flags)
	}
}

// PureFunctionParameterExpr indicates a positional parameter to a pure function definition
type PureFunctionParameterExpr struct {
	*ExpressionBase
	Index int
}

// NewPureFunctionParameterExpr creates a new PureFunctionParameterExpr
func NewPureFunctionParameterExpr(index int) *PureFunctionParameterExpr {
	return &PureFunctionParameterExpr{
		ExpressionBase: NewExpressionBase(ir.ExpressionKindPureFunctionParameterExpr, nil),
		Index:          index,
	}
}

// VisitExpression implements OutputExpression interface
func (p *PureFunctionParameterExpr) VisitExpression(visitor output.ExpressionVisitor, context interface{}) interface{} {
	return nil
}

// IsEquivalent checks if two expressions are equivalent
func (p *PureFunctionParameterExpr) IsEquivalent(other output.OutputExpression) bool {
	if otherParam, ok := other.(*PureFunctionParameterExpr); ok {
		return p.Index == otherParam.Index
	}
	return false
}

// IsConstant returns true
func (p *PureFunctionParameterExpr) IsConstant() bool {
	return true
}

// Clone clones the expression
func (p *PureFunctionParameterExpr) Clone() output.OutputExpression {
	return NewPureFunctionParameterExpr(p.Index)
}

// TransformInternalExpressions transforms internal expressions
func (p *PureFunctionParameterExpr) TransformInternalExpressions(transform ExpressionTransform, flags VisitorContextFlag) {
	// No nested expressions
}

// PipeBindingExpr is a binding to a pipe transformation
type PipeBindingExpr struct {
	*ExpressionBase
	VarOffset  *int
	Target     ir_operations.XrefId
	TargetSlot *ir.SlotHandle
	Name       string
	Args       []output.OutputExpression
}

// NewPipeBindingExpr creates a new PipeBindingExpr
func NewPipeBindingExpr(
	target ir_operations.XrefId,
	targetSlot *ir.SlotHandle,
	name string,
	args []output.OutputExpression,
) *PipeBindingExpr {
	return &PipeBindingExpr{
		ExpressionBase: NewExpressionBase(ir.ExpressionKindPipeBinding, nil),
		VarOffset:      nil,
		Target:         target,
		TargetSlot:     targetSlot,
		Name:           name,
		Args:           args,
	}
}

// HasConsumesVarsTrait implements ConsumesVarsTraitInterface
func (p *PipeBindingExpr) HasConsumesVarsTrait() bool {
	return true
}

// GetVarOffset implements UsesVarOffsetTraitInterface
func (p *PipeBindingExpr) GetVarOffset() *int {
	return p.VarOffset
}

// SetVarOffset implements UsesVarOffsetTraitInterface
func (p *PipeBindingExpr) SetVarOffset(offset int) {
	p.VarOffset = &offset
}

// VisitExpression implements OutputExpression interface
func (p *PipeBindingExpr) VisitExpression(visitor output.ExpressionVisitor, context interface{}) interface{} {
	for _, arg := range p.Args {
		arg.VisitExpression(visitor, context)
	}
	return nil
}

// IsEquivalent returns false (pipe bindings are not equivalent)
func (p *PipeBindingExpr) IsEquivalent(other output.OutputExpression) bool {
	return false
}

// IsConstant returns false
func (p *PipeBindingExpr) IsConstant() bool {
	return false
}

// Clone clones the expression
func (p *PipeBindingExpr) Clone() output.OutputExpression {
	clonedArgs := make([]output.OutputExpression, len(p.Args))
	for i, arg := range p.Args {
		clonedArgs[i] = arg.Clone()
	}
	expr := NewPipeBindingExpr(p.Target, p.TargetSlot, p.Name, clonedArgs)
	expr.VarOffset = p.VarOffset
	return expr
}

// TransformInternalExpressions transforms internal expressions
func (p *PipeBindingExpr) TransformInternalExpressions(transform ExpressionTransform, flags VisitorContextFlag) {
	for i := 0; i < len(p.Args); i++ {
		p.Args[i] = TransformExpressionsInExpression(p.Args[i], transform, flags)
	}
}

// PipeBindingVariadicExpr is a binding to a pipe transformation with a variable number of arguments
type PipeBindingVariadicExpr struct {
	*ExpressionBase
	VarOffset  *int
	Target     ir_operations.XrefId
	TargetSlot *ir.SlotHandle
	Name       string
	Args       output.OutputExpression
	NumArgs    int
}

// NewPipeBindingVariadicExpr creates a new PipeBindingVariadicExpr
func NewPipeBindingVariadicExpr(
	target ir_operations.XrefId,
	targetSlot *ir.SlotHandle,
	name string,
	args output.OutputExpression,
	numArgs int,
) *PipeBindingVariadicExpr {
	return &PipeBindingVariadicExpr{
		ExpressionBase: NewExpressionBase(ir.ExpressionKindPipeBindingVariadic, nil),
		VarOffset:      nil,
		Target:         target,
		TargetSlot:     targetSlot,
		Name:           name,
		Args:           args,
		NumArgs:        numArgs,
	}
}

// HasConsumesVarsTrait implements ConsumesVarsTraitInterface
func (p *PipeBindingVariadicExpr) HasConsumesVarsTrait() bool {
	return true
}

// GetVarOffset implements UsesVarOffsetTraitInterface
func (p *PipeBindingVariadicExpr) GetVarOffset() *int {
	return p.VarOffset
}

// SetVarOffset implements UsesVarOffsetTraitInterface
func (p *PipeBindingVariadicExpr) SetVarOffset(offset int) {
	p.VarOffset = &offset
}

// VisitExpression implements OutputExpression interface
func (p *PipeBindingVariadicExpr) VisitExpression(visitor output.ExpressionVisitor, context interface{}) interface{} {
	p.Args.VisitExpression(visitor, context)
	return nil
}

// IsEquivalent returns false
func (p *PipeBindingVariadicExpr) IsEquivalent(other output.OutputExpression) bool {
	return false
}

// IsConstant returns false
func (p *PipeBindingVariadicExpr) IsConstant() bool {
	return false
}

// Clone clones the expression
func (p *PipeBindingVariadicExpr) Clone() output.OutputExpression {
	expr := NewPipeBindingVariadicExpr(p.Target, p.TargetSlot, p.Name, p.Args.Clone(), p.NumArgs)
	expr.VarOffset = p.VarOffset
	return expr
}

// TransformInternalExpressions transforms internal expressions
func (p *PipeBindingVariadicExpr) TransformInternalExpressions(transform ExpressionTransform, flags VisitorContextFlag) {
	p.Args = TransformExpressionsInExpression(p.Args, transform, flags)
}

// SafePropertyReadExpr is a safe property read requiring expansion into a null check
type SafePropertyReadExpr struct {
	*ExpressionBase
	Receiver output.OutputExpression
	Name     string
}

// NewSafePropertyReadExpr creates a new SafePropertyReadExpr
func NewSafePropertyReadExpr(receiver output.OutputExpression, name string) *SafePropertyReadExpr {
	return &SafePropertyReadExpr{
		ExpressionBase: NewExpressionBase(ir.ExpressionKindSafePropertyRead, nil),
		Receiver:       receiver,
		Name:           name,
	}
}

// GetIndex returns the name (alias for name)
func (s *SafePropertyReadExpr) GetIndex() string {
	return s.Name
}

// VisitExpression implements OutputExpression interface
func (s *SafePropertyReadExpr) VisitExpression(visitor output.ExpressionVisitor, context interface{}) interface{} {
	s.Receiver.VisitExpression(visitor, context)
	return nil
}

// IsEquivalent returns false
func (s *SafePropertyReadExpr) IsEquivalent(other output.OutputExpression) bool {
	return false
}

// IsConstant returns false
func (s *SafePropertyReadExpr) IsConstant() bool {
	return false
}

// Clone clones the expression
func (s *SafePropertyReadExpr) Clone() output.OutputExpression {
	return NewSafePropertyReadExpr(s.Receiver.Clone(), s.Name)
}

// TransformInternalExpressions transforms internal expressions
func (s *SafePropertyReadExpr) TransformInternalExpressions(transform ExpressionTransform, flags VisitorContextFlag) {
	s.Receiver = TransformExpressionsInExpression(s.Receiver, transform, flags)
}

// SafeKeyedReadExpr is a safe keyed read requiring expansion into a null check
type SafeKeyedReadExpr struct {
	*ExpressionBase
	Receiver output.OutputExpression
	Index    output.OutputExpression
}

// NewSafeKeyedReadExpr creates a new SafeKeyedReadExpr
func NewSafeKeyedReadExpr(
	receiver output.OutputExpression,
	index output.OutputExpression,
	sourceSpan *util.ParseSourceSpan,
) *SafeKeyedReadExpr {
	return &SafeKeyedReadExpr{
		ExpressionBase: NewExpressionBase(ir.ExpressionKindSafeKeyedRead, sourceSpan),
		Receiver:       receiver,
		Index:          index,
	}
}

// VisitExpression implements OutputExpression interface
func (s *SafeKeyedReadExpr) VisitExpression(visitor output.ExpressionVisitor, context interface{}) interface{} {
	s.Receiver.VisitExpression(visitor, context)
	s.Index.VisitExpression(visitor, context)
	return nil
}

// IsEquivalent returns false
func (s *SafeKeyedReadExpr) IsEquivalent(other output.OutputExpression) bool {
	return false
}

// IsConstant returns false
func (s *SafeKeyedReadExpr) IsConstant() bool {
	return false
}

// Clone clones the expression
func (s *SafeKeyedReadExpr) Clone() output.OutputExpression {
	return NewSafeKeyedReadExpr(s.Receiver.Clone(), s.Index.Clone(), s.SourceSpan)
}

// TransformInternalExpressions transforms internal expressions
func (s *SafeKeyedReadExpr) TransformInternalExpressions(transform ExpressionTransform, flags VisitorContextFlag) {
	s.Receiver = TransformExpressionsInExpression(s.Receiver, transform, flags)
	s.Index = TransformExpressionsInExpression(s.Index, transform, flags)
}

// SafeInvokeFunctionExpr is a safe function call requiring expansion into a null check
type SafeInvokeFunctionExpr struct {
	*ExpressionBase
	Receiver output.OutputExpression
	Args     []output.OutputExpression
}

// NewSafeInvokeFunctionExpr creates a new SafeInvokeFunctionExpr
func NewSafeInvokeFunctionExpr(receiver output.OutputExpression, args []output.OutputExpression) *SafeInvokeFunctionExpr {
	return &SafeInvokeFunctionExpr{
		ExpressionBase: NewExpressionBase(ir.ExpressionKindSafeInvokeFunction, nil),
		Receiver:       receiver,
		Args:           args,
	}
}

// VisitExpression implements OutputExpression interface
func (s *SafeInvokeFunctionExpr) VisitExpression(visitor output.ExpressionVisitor, context interface{}) interface{} {
	s.Receiver.VisitExpression(visitor, context)
	for _, arg := range s.Args {
		arg.VisitExpression(visitor, context)
	}
	return nil
}

// IsEquivalent returns false
func (s *SafeInvokeFunctionExpr) IsEquivalent(other output.OutputExpression) bool {
	return false
}

// IsConstant returns false
func (s *SafeInvokeFunctionExpr) IsConstant() bool {
	return false
}

// Clone clones the expression
func (s *SafeInvokeFunctionExpr) Clone() output.OutputExpression {
	clonedArgs := make([]output.OutputExpression, len(s.Args))
	for i, arg := range s.Args {
		clonedArgs[i] = arg.Clone()
	}
	return NewSafeInvokeFunctionExpr(s.Receiver.Clone(), clonedArgs)
}

// TransformInternalExpressions transforms internal expressions
func (s *SafeInvokeFunctionExpr) TransformInternalExpressions(transform ExpressionTransform, flags VisitorContextFlag) {
	s.Receiver = TransformExpressionsInExpression(s.Receiver, transform, flags)
	for i := 0; i < len(s.Args); i++ {
		s.Args[i] = TransformExpressionsInExpression(s.Args[i], transform, flags)
	}
}

// SafeTernaryExpr is an intermediate expression that will be expanded from a safe read
type SafeTernaryExpr struct {
	*ExpressionBase
	Guard output.OutputExpression
	Expr  output.OutputExpression
}

// NewSafeTernaryExpr creates a new SafeTernaryExpr
func NewSafeTernaryExpr(guard, expr output.OutputExpression) *SafeTernaryExpr {
	return &SafeTernaryExpr{
		ExpressionBase: NewExpressionBase(ir.ExpressionKindSafeTernaryExpr, nil),
		Guard:          guard,
		Expr:           expr,
	}
}

// VisitExpression implements OutputExpression interface
func (s *SafeTernaryExpr) VisitExpression(visitor output.ExpressionVisitor, context interface{}) interface{} {
	s.Guard.VisitExpression(visitor, context)
	s.Expr.VisitExpression(visitor, context)
	return nil
}

// IsEquivalent returns false
func (s *SafeTernaryExpr) IsEquivalent(other output.OutputExpression) bool {
	return false
}

// IsConstant returns false
func (s *SafeTernaryExpr) IsConstant() bool {
	return false
}

// Clone clones the expression
func (s *SafeTernaryExpr) Clone() output.OutputExpression {
	return NewSafeTernaryExpr(s.Guard.Clone(), s.Expr.Clone())
}

// TransformInternalExpressions transforms internal expressions
func (s *SafeTernaryExpr) TransformInternalExpressions(transform ExpressionTransform, flags VisitorContextFlag) {
	s.Guard = TransformExpressionsInExpression(s.Guard, transform, flags)
	s.Expr = TransformExpressionsInExpression(s.Expr, transform, flags)
}

// AssignTemporaryExpr is an assignment to a temporary variable
type AssignTemporaryExpr struct {
	*ExpressionBase
	Name *string
	Expr output.OutputExpression
	Xref ir_operations.XrefId
}

// NewAssignTemporaryExpr creates a new AssignTemporaryExpr
func NewAssignTemporaryExpr(expr output.OutputExpression, xref ir_operations.XrefId) *AssignTemporaryExpr {
	return &AssignTemporaryExpr{
		ExpressionBase: NewExpressionBase(ir.ExpressionKindAssignTemporaryExpr, nil),
		Name:           nil,
		Expr:           expr,
		Xref:           xref,
	}
}

// VisitExpression implements OutputExpression interface
func (a *AssignTemporaryExpr) VisitExpression(visitor output.ExpressionVisitor, context interface{}) interface{} {
	a.Expr.VisitExpression(visitor, context)
	return nil
}

// IsEquivalent returns false
func (a *AssignTemporaryExpr) IsEquivalent(other output.OutputExpression) bool {
	return false
}

// IsConstant returns false
func (a *AssignTemporaryExpr) IsConstant() bool {
	return false
}

// Clone clones the expression
func (a *AssignTemporaryExpr) Clone() output.OutputExpression {
	expr := NewAssignTemporaryExpr(a.Expr.Clone(), a.Xref)
	expr.Name = a.Name
	return expr
}

// TransformInternalExpressions transforms internal expressions
func (a *AssignTemporaryExpr) TransformInternalExpressions(transform ExpressionTransform, flags VisitorContextFlag) {
	a.Expr = TransformExpressionsInExpression(a.Expr, transform, flags)
}

// ReadTemporaryExpr is a reference to a temporary variable
type ReadTemporaryExpr struct {
	*ExpressionBase
	Name *string
	Xref ir_operations.XrefId
}

// NewReadTemporaryExpr creates a new ReadTemporaryExpr
func NewReadTemporaryExpr(xref ir_operations.XrefId) *ReadTemporaryExpr {
	return &ReadTemporaryExpr{
		ExpressionBase: NewExpressionBase(ir.ExpressionKindReadTemporaryExpr, nil),
		Name:           nil,
		Xref:           xref,
	}
}

// VisitExpression implements OutputExpression interface
func (r *ReadTemporaryExpr) VisitExpression(visitor output.ExpressionVisitor, context interface{}) interface{} {
	return nil
}

// IsEquivalent checks if two expressions are equivalent
func (r *ReadTemporaryExpr) IsEquivalent(other output.OutputExpression) bool {
	if otherRead, ok := other.(*ReadTemporaryExpr); ok {
		return r.Xref == otherRead.Xref
	}
	return false
}

// IsConstant returns false
func (r *ReadTemporaryExpr) IsConstant() bool {
	return false
}

// Clone clones the expression
func (r *ReadTemporaryExpr) Clone() output.OutputExpression {
	expr := NewReadTemporaryExpr(r.Xref)
	expr.Name = r.Name
	return expr
}

// TransformInternalExpressions transforms internal expressions
func (r *ReadTemporaryExpr) TransformInternalExpressions(transform ExpressionTransform, flags VisitorContextFlag) {
	// No nested expressions
}

// SlotLiteralExpr is an expression that will cause a literal slot index to be emitted
type SlotLiteralExpr struct {
	*ExpressionBase
	Slot *ir.SlotHandle
}

// NewSlotLiteralExpr creates a new SlotLiteralExpr
func NewSlotLiteralExpr(slot *ir.SlotHandle) *SlotLiteralExpr {
	return &SlotLiteralExpr{
		ExpressionBase: NewExpressionBase(ir.ExpressionKindSlotLiteralExpr, nil),
		Slot:           slot,
	}
}

// VisitExpression implements OutputExpression interface
func (s *SlotLiteralExpr) VisitExpression(visitor output.ExpressionVisitor, context interface{}) interface{} {
	return nil
}

// IsEquivalent checks if two expressions are equivalent
func (s *SlotLiteralExpr) IsEquivalent(other output.OutputExpression) bool {
	if otherSlot, ok := other.(*SlotLiteralExpr); ok {
		return s.Slot == otherSlot.Slot
	}
	return false
}

// IsConstant returns true
func (s *SlotLiteralExpr) IsConstant() bool {
	return true
}

// Clone clones the expression
func (s *SlotLiteralExpr) Clone() output.OutputExpression {
	return NewSlotLiteralExpr(s.Slot)
}

// TransformInternalExpressions transforms internal expressions
func (s *SlotLiteralExpr) TransformInternalExpressions(transform ExpressionTransform, flags VisitorContextFlag) {
	// No nested expressions
}

// ConditionalCaseExpr is a test expression for a conditional op
type ConditionalCaseExpr struct {
	*ExpressionBase
	Expr       output.OutputExpression
	Target     ir_operations.XrefId
	TargetSlot *ir.SlotHandle
	Alias      *render3.Variable
}

// NewConditionalCaseExpr creates a new ConditionalCaseExpr
func NewConditionalCaseExpr(
	expr output.OutputExpression,
	target ir_operations.XrefId,
	targetSlot *ir.SlotHandle,
	alias *render3.Variable,
) *ConditionalCaseExpr {
	return &ConditionalCaseExpr{
		ExpressionBase: NewExpressionBase(ir.ExpressionKindConditionalCase, nil),
		Expr:           expr,
		Target:         target,
		TargetSlot:     targetSlot,
		Alias:          alias,
	}
}

// VisitExpression implements OutputExpression interface
func (c *ConditionalCaseExpr) VisitExpression(visitor output.ExpressionVisitor, context interface{}) interface{} {
	if c.Expr != nil {
		c.Expr.VisitExpression(visitor, context)
	}
	return nil
}

// IsEquivalent checks if two expressions are equivalent
func (c *ConditionalCaseExpr) IsEquivalent(other output.OutputExpression) bool {
	if otherCase, ok := other.(*ConditionalCaseExpr); ok {
		if c.Expr == nil {
			return otherCase.Expr == nil
		}
		if otherCase.Expr == nil {
			return false
		}
		return c.Expr.IsEquivalent(otherCase.Expr)
	}
	return false
}

// IsConstant returns true
func (c *ConditionalCaseExpr) IsConstant() bool {
	return true
}

// Clone clones the expression
func (c *ConditionalCaseExpr) Clone() output.OutputExpression {
	var clonedExpr output.OutputExpression
	if c.Expr != nil {
		clonedExpr = c.Expr.Clone()
	}
	return NewConditionalCaseExpr(clonedExpr, c.Target, c.TargetSlot, c.Alias)
}

// TransformInternalExpressions transforms internal expressions
func (c *ConditionalCaseExpr) TransformInternalExpressions(transform ExpressionTransform, flags VisitorContextFlag) {
	if c.Expr != nil {
		c.Expr = TransformExpressionsInExpression(c.Expr, transform, flags)
	}
}

// ConstCollectedExpr is an expression that will be automatically extracted to the component const array
type ConstCollectedExpr struct {
	*ExpressionBase
	Expr output.OutputExpression
}

// NewConstCollectedExpr creates a new ConstCollectedExpr
func NewConstCollectedExpr(expr output.OutputExpression) *ConstCollectedExpr {
	return &ConstCollectedExpr{
		ExpressionBase: NewExpressionBase(ir.ExpressionKindConstCollected, nil),
		Expr:           expr,
	}
}

// TransformInternalExpressions transforms internal expressions
func (c *ConstCollectedExpr) TransformInternalExpressions(transform ExpressionTransform, flags VisitorContextFlag) {
	c.Expr = transform(c.Expr, flags)
}

// VisitExpression implements OutputExpression interface
func (c *ConstCollectedExpr) VisitExpression(visitor output.ExpressionVisitor, context interface{}) interface{} {
	return c.Expr.VisitExpression(visitor, context)
}

// IsEquivalent checks if two expressions are equivalent
func (c *ConstCollectedExpr) IsEquivalent(other output.OutputExpression) bool {
	if otherConst, ok := other.(*ConstCollectedExpr); ok {
		return c.Expr.IsEquivalent(otherConst.Expr)
	}
	return false
}

// IsConstant returns the constant status of the inner expression
func (c *ConstCollectedExpr) IsConstant() bool {
	return c.Expr.IsConstant()
}

// Clone clones the expression
func (c *ConstCollectedExpr) Clone() output.OutputExpression {
	return NewConstCollectedExpr(c.Expr.Clone())
}

// StoreLetExpr is a call storing the value of a `@let` declaration
type StoreLetExpr struct {
	*ExpressionBase
	Target     ir_operations.XrefId
	Value      output.OutputExpression
	SourceSpan *util.ParseSourceSpan
}

// NewStoreLetExpr creates a new StoreLetExpr
func NewStoreLetExpr(
	target ir_operations.XrefId,
	value output.OutputExpression,
	sourceSpan *util.ParseSourceSpan,
) *StoreLetExpr {
	return &StoreLetExpr{
		ExpressionBase: NewExpressionBase(ir.ExpressionKindStoreLet, sourceSpan),
		Target:         target,
		Value:          value,
		SourceSpan:     sourceSpan,
	}
}

// HasConsumesVarsTrait implements ConsumesVarsTraitInterface
func (s *StoreLetExpr) HasConsumesVarsTrait() bool {
	return true
}

// GetDependsOnSlotContextTrait returns the DependsOnSlotContextOpTrait
func (s *StoreLetExpr) GetDependsOnSlotContextTrait() *ir_traits.DependsOnSlotContextOpTrait {
	return &ir_traits.DependsOnSlotContextOpTrait{
		Target:     s.Target,
		SourceSpan: s.SourceSpan,
	}
}

// VisitExpression implements OutputExpression interface
func (s *StoreLetExpr) VisitExpression(visitor output.ExpressionVisitor, context interface{}) interface{} {
	return nil
}

// IsEquivalent checks if two expressions are equivalent
func (s *StoreLetExpr) IsEquivalent(other output.OutputExpression) bool {
	if otherStore, ok := other.(*StoreLetExpr); ok {
		return s.Target == otherStore.Target && s.Value.IsEquivalent(otherStore.Value)
	}
	return false
}

// IsConstant returns false
func (s *StoreLetExpr) IsConstant() bool {
	return false
}

// Clone clones the expression
func (s *StoreLetExpr) Clone() output.OutputExpression {
	return NewStoreLetExpr(s.Target, s.Value.Clone(), s.SourceSpan)
}

// TransformInternalExpressions transforms internal expressions
func (s *StoreLetExpr) TransformInternalExpressions(transform ExpressionTransform, flags VisitorContextFlag) {
	s.Value = TransformExpressionsInExpression(s.Value, transform, flags)
}

// ContextLetReferenceExpr is a reference to a `@let` declaration read from the context view
type ContextLetReferenceExpr struct {
	*ExpressionBase
	Target     ir_operations.XrefId
	TargetSlot *ir.SlotHandle
}

// NewContextLetReferenceExpr creates a new ContextLetReferenceExpr
func NewContextLetReferenceExpr(target ir_operations.XrefId, targetSlot *ir.SlotHandle) *ContextLetReferenceExpr {
	return &ContextLetReferenceExpr{
		ExpressionBase: NewExpressionBase(ir.ExpressionKindContextLetReference, nil),
		Target:         target,
		TargetSlot:     targetSlot,
	}
}

// VisitExpression implements OutputExpression interface
func (c *ContextLetReferenceExpr) VisitExpression(visitor output.ExpressionVisitor, context interface{}) interface{} {
	return nil
}

// IsEquivalent checks if two expressions are equivalent
func (c *ContextLetReferenceExpr) IsEquivalent(other output.OutputExpression) bool {
	if otherCtx, ok := other.(*ContextLetReferenceExpr); ok {
		return c.Target == otherCtx.Target
	}
	return false
}

// IsConstant returns false
func (c *ContextLetReferenceExpr) IsConstant() bool {
	return false
}

// Clone clones the expression
func (c *ContextLetReferenceExpr) Clone() output.OutputExpression {
	return NewContextLetReferenceExpr(c.Target, c.TargetSlot)
}

// TransformInternalExpressions transforms internal expressions
func (c *ContextLetReferenceExpr) TransformInternalExpressions(transform ExpressionTransform, flags VisitorContextFlag) {
	// No nested expressions
}

// transformExpressionsInInterpolation transforms expressions in an interpolation
func transformExpressionsInInterpolation(
	interpolation *ops_update.Interpolation,
	transform ExpressionTransform,
	flags VisitorContextFlag,
) {
	for i := 0; i < len(interpolation.Expressions); i++ {
		interpolation.Expressions[i] = TransformExpressionsInExpression(interpolation.Expressions[i], transform, flags)
	}
}

// TransformExpressionsInOp transforms all expressions in an operations
func TransformExpressionsInOp(
	op ir_operations.Op,
	transform ExpressionTransform,
	flags VisitorContextFlag,
) {
	switch op.GetKind() {
	case ir.OpKindStyleProp, ir.OpKindStyleMap, ir.OpKindClassProp, ir.OpKindClassMap,
		ir.OpKindAnimationString, ir.OpKindAnimationBinding, ir.OpKindBinding:
		// Handle operations with expression or interpolation
		if bindingOp, ok := op.(*ops_update.BindingOp); ok {
			if interpolation, ok := bindingOp.Expression.(*ops_update.Interpolation); ok {
				transformExpressionsInInterpolation(interpolation, transform, flags)
			} else if expr, ok := bindingOp.Expression.(output.OutputExpression); ok {
				bindingOp.Expression = TransformExpressionsInExpression(expr, transform, flags)
			}
		}
	case ir.OpKindProperty, ir.OpKindDomProperty, ir.OpKindAttribute, ir.OpKindControl:
		// Handle property/attribute operations with expression and sanitizer
		if propertyOp, ok := op.(*ops_update.PropertyOp); ok {
			if interpolation, ok := propertyOp.Expression.(*ops_update.Interpolation); ok {
				transformExpressionsInInterpolation(interpolation, transform, flags)
			} else if expr, ok := propertyOp.Expression.(output.OutputExpression); ok {
				propertyOp.Expression = TransformExpressionsInExpression(expr, transform, flags)
			}
			if propertyOp.Sanitizer != nil {
				propertyOp.Sanitizer = TransformExpressionsInExpression(propertyOp.Sanitizer, transform, flags)
			}
		}
	case ir.OpKindTwoWayProperty:
		if twoWayOp, ok := op.(*ops_update.TwoWayPropertyOp); ok {
			twoWayOp.Expression = TransformExpressionsInExpression(twoWayOp.Expression, transform, flags)
			if twoWayOp.Sanitizer != nil {
				twoWayOp.Sanitizer = TransformExpressionsInExpression(twoWayOp.Sanitizer, transform, flags)
			}
		}
	case ir.OpKindI18nExpression:
		if i18nExprOp, ok := op.(*ops_update.I18nExpressionOp); ok {
			i18nExprOp.Expression = TransformExpressionsInExpression(i18nExprOp.Expression, transform, flags)
		}
	case ir.OpKindInterpolateText:
		if interpolateOp, ok := op.(*ops_update.InterpolateTextOp); ok {
			transformExpressionsInInterpolation(interpolateOp.Interpolation, transform, flags)
		}
	case ir.OpKindStatement:
		if stmtOp, ok := op.(*shared.StatementOp); ok {
			TransformExpressionsInStatement(stmtOp.Statement, transform, flags)
		}
	case ir.OpKindVariable:
		if varOp, ok := op.(*shared.VariableOp); ok {
			varOp.Initializer = TransformExpressionsInExpression(varOp.Initializer, transform, flags)
		}
	case ir.OpKindConditional:
		if conditionalOp, ok := op.(*ops_update.ConditionalOp); ok {
			for _, condition := range conditionalOp.Conditions {
				if caseExpr, ok := condition.(*ConditionalCaseExpr); ok && caseExpr.Expr != nil {
					caseExpr.Expr = TransformExpressionsInExpression(caseExpr.Expr, transform, flags)
				}
			}
			if conditionalOp.Processed != nil {
				conditionalOp.Processed = TransformExpressionsInExpression(conditionalOp.Processed, transform, flags)
			}
			if conditionalOp.ContextValue != nil {
				conditionalOp.ContextValue = TransformExpressionsInExpression(conditionalOp.ContextValue, transform, flags)
			}
		}
	case ir.OpKindAnimation, ir.OpKindAnimationListener, ir.OpKindListener, ir.OpKindTwoWayListener:
		// Handle operations with handlerOps
		if listenerOp, ok := op.(*ops_create.ListenerOp); ok && listenerOp.HandlerOps != nil {
			for handlerOp := listenerOp.HandlerOps.Head(); handlerOp != nil && handlerOp.GetKind() != ir.OpKindListEnd; handlerOp = handlerOp.Next() {
				TransformExpressionsInOp(handlerOp, transform, flags|VisitorContextFlagInChildOperation)
			}
		} else if twoWayOp, ok := op.(*ops_create.TwoWayListenerOp); ok && twoWayOp.HandlerOps != nil {
			for handlerOp := twoWayOp.HandlerOps.Head(); handlerOp != nil && handlerOp.GetKind() != ir.OpKindListEnd; handlerOp = handlerOp.Next() {
				TransformExpressionsInOp(handlerOp, transform, flags|VisitorContextFlagInChildOperation)
			}
		} else if animOp, ok := op.(*ops_create.AnimationOp); ok && animOp.HandlerOps != nil {
			for handlerOp := animOp.HandlerOps.Head(); handlerOp != nil && handlerOp.GetKind() != ir.OpKindListEnd; handlerOp = handlerOp.Next() {
				TransformExpressionsInOp(handlerOp, transform, flags|VisitorContextFlagInChildOperation)
			}
		} else if animListenerOp, ok := op.(*ops_create.AnimationListenerOp); ok && animListenerOp.HandlerOps != nil {
			for handlerOp := animListenerOp.HandlerOps.Head(); handlerOp != nil && handlerOp.GetKind() != ir.OpKindListEnd; handlerOp = handlerOp.Next() {
				TransformExpressionsInOp(handlerOp, transform, flags|VisitorContextFlagInChildOperation)
			}
		}
	case ir.OpKindExtractedAttribute:
		if extractedOp, ok := op.(*ops_create.ExtractedAttributeOp); ok {
			if extractedOp.Expression != nil {
				extractedOp.Expression = TransformExpressionsInExpression(extractedOp.Expression, transform, flags)
			}
			if extractedOp.TrustedValueFn != nil {
				extractedOp.TrustedValueFn = TransformExpressionsInExpression(extractedOp.TrustedValueFn, transform, flags)
			}
		}
	case ir.OpKindRepeaterCreate:
		if repeaterOp, ok := op.(*ops_create.RepeaterCreateOp); ok {
			if repeaterOp.TrackByOps == nil {
				repeaterOp.Track = TransformExpressionsInExpression(repeaterOp.Track, transform, flags)
			} else {
				for innerOp := repeaterOp.TrackByOps.Head(); innerOp != nil; innerOp = innerOp.Next() {
					TransformExpressionsInOp(innerOp, transform, flags|VisitorContextFlagInChildOperation)
				}
			}
			if repeaterOp.TrackByFn != nil {
				repeaterOp.TrackByFn = TransformExpressionsInExpression(repeaterOp.TrackByFn, transform, flags)
			}
		}
	case ir.OpKindRepeater:
		if repeaterOp, ok := op.(*ops_update.RepeaterOp); ok {
			repeaterOp.Collection = TransformExpressionsInExpression(repeaterOp.Collection, transform, flags)
		}
	case ir.OpKindDefer:
		if deferOp, ok := op.(*ops_create.DeferOp); ok {
			if deferOp.LoadingConfig != nil {
				deferOp.LoadingConfig = TransformExpressionsInExpression(deferOp.LoadingConfig, transform, flags)
			}
			if deferOp.PlaceholderConfig != nil {
				deferOp.PlaceholderConfig = TransformExpressionsInExpression(deferOp.PlaceholderConfig, transform, flags)
			}
			if deferOp.ResolverFn != nil {
				deferOp.ResolverFn = TransformExpressionsInExpression(deferOp.ResolverFn, transform, flags)
			}
		}
	case ir.OpKindI18nMessage:
		// TODO: Handle I18nMessageOp params and postprocessingParams
		// This requires checking the structure of I18nMessageOp
	case ir.OpKindDeferWhen:
		if deferWhenOp, ok := op.(*ops_update.DeferWhenOp); ok {
			deferWhenOp.Expr = TransformExpressionsInExpression(deferWhenOp.Expr, transform, flags)
		}
	case ir.OpKindStoreLet:
		if storeLetOp, ok := op.(*ops_update.StoreLetOp); ok {
			storeLetOp.Value = TransformExpressionsInExpression(storeLetOp.Value, transform, flags)
		}
	case ir.OpKindAdvance, ir.OpKindContainer, ir.OpKindContainerEnd, ir.OpKindContainerStart,
		ir.OpKindDeferOn, ir.OpKindDisableBindings, ir.OpKindElement, ir.OpKindElementEnd,
		ir.OpKindElementStart, ir.OpKindEnableBindings, ir.OpKindI18n, ir.OpKindI18nApply,
		ir.OpKindI18nContext, ir.OpKindI18nEnd, ir.OpKindI18nStart, ir.OpKindIcuEnd, ir.OpKindIcuStart,
		ir.OpKindNamespace, ir.OpKindPipe, ir.OpKindProjection, ir.OpKindProjectionDef,
		ir.OpKindTemplate, ir.OpKindText, ir.OpKindI18nAttributes, ir.OpKindIcuPlaceholder,
		ir.OpKindDeclareLet, ir.OpKindSourceLocation, ir.OpKindConditionalCreate,
		ir.OpKindConditionalBranchCreate, ir.OpKindControlCreate:
		// These operations contain no expressions.
		break
	default:
		panic(fmt.Sprintf("AssertionError: TransformExpressionsInOp doesn't handle %v", op.GetKind()))
	}
}

// VisitExpressionsInOp visits all expressions in an operations
func VisitExpressionsInOp(
	op ir_operations.Op,
	visitor func(expr output.OutputExpression, flags VisitorContextFlag),
) {
	TransformExpressionsInOp(
		op,
		func(expr output.OutputExpression, flags VisitorContextFlag) output.OutputExpression {
			visitor(expr, flags)
			return expr
		},
		VisitorContextFlagNone,
	)
}

// TransformExpressionsInStatement transforms all expressions in a statement
func TransformExpressionsInStatement(
	stmt output.OutputStatement,
	transform ExpressionTransform,
	flags VisitorContextFlag,
) {
	// Handle different statement types
	switch s := stmt.(type) {
	case *output.ExpressionStatement:
		if s.Expr != nil {
			s.Expr = TransformExpressionsInExpression(s.Expr, transform, flags)
		}
	case *output.ReturnStatement:
		if s.Value != nil {
			s.Value = TransformExpressionsInExpression(s.Value, transform, flags)
		}
	case *output.IfStmt:
		if s.Condition != nil {
			s.Condition = TransformExpressionsInExpression(s.Condition, transform, flags)
		}
		for _, stmt := range s.TrueCase {
			TransformExpressionsInStatement(stmt, transform, flags)
		}
		for _, stmt := range s.FalseCase {
			TransformExpressionsInStatement(stmt, transform, flags)
		}
	case *output.DeclareVarStmt:
		if s.Value != nil {
			s.Value = TransformExpressionsInExpression(s.Value, transform, flags)
		}
		// Other statement types don't contain expressions or are handled elsewhere
	}
}
