package render3

import (
	"ngc-go/packages/compiler/output"
	"ngc-go/packages/compiler/render3/r3_identifiers"
)

// TypeWithParameters creates an ExpressionType with the given number of parameters
func TypeWithParameters(typ output.OutputExpression, numParams int) output.Type {
	if numParams == 0 {
		return output.NewExpressionType(typ, output.TypeModifierNone, nil)
	}
	params := make([]output.Type, numParams)
	for i := 0; i < numParams; i++ {
		params[i] = output.DynamicType
	}
	return output.NewExpressionType(typ, output.TypeModifierNone, params)
}

// R3Reference represents a reference with value and type
type R3Reference struct {
	Value output.OutputExpression
	Type  output.OutputExpression
}

// R3CompiledExpression represents the result of compilation of a render3 code unit
type R3CompiledExpression struct {
	Expression output.OutputExpression
	Type       output.Type
	Statements []output.OutputStatement
}

const LEGACY_ANIMATE_SYMBOL_PREFIX = "@"

// PrepareSyntheticPropertyName prepares a synthetic property name
func PrepareSyntheticPropertyName(name string) string {
	return LEGACY_ANIMATE_SYMBOL_PREFIX + name
}

// PrepareSyntheticListenerName prepares a synthetic listener name
func PrepareSyntheticListenerName(name, phase string) string {
	return LEGACY_ANIMATE_SYMBOL_PREFIX + name + "." + phase
}

// GetSafePropertyAccessString returns a safe property access string
func GetSafePropertyAccessString(accessor, name string) string {
	escapedName := EscapeIdentifier(name, false, false)
	if escapedName != name {
		return accessor + "[" + escapedName + "]"
	}
	return accessor + "." + name
}

// PrepareSyntheticListenerFunctionName prepares a synthetic listener function name
func PrepareSyntheticListenerFunctionName(name, phase string) string {
	return "animation_" + name + "_" + phase
}

// JitOnlyGuardedExpression creates a JIT-only guarded expression
func JitOnlyGuardedExpression(expr output.OutputExpression) output.OutputExpression {
	return GuardedExpression("ngJitMode", expr)
}

// DevOnlyGuardedExpression creates a dev-only guarded expression
func DevOnlyGuardedExpression(expr output.OutputExpression) output.OutputExpression {
	return GuardedExpression("ngDevMode", expr)
}

// GuardedExpression creates a guarded expression
func GuardedExpression(guard string, expr output.OutputExpression) output.OutputExpression {
	guardName := guard
	guardExpr := output.NewExternalExpr(&output.ExternalReference{Name: &guardName, ModuleName: nil}, nil, nil, nil)
	guardNotDefined := output.NewBinaryOperatorExpr(
		output.BinaryOperatorIdentical,
		output.NewTypeofExpr(guardExpr, nil, nil),
		output.NewLiteralExpr("undefined", nil, nil),
		nil,
		nil,
	)
	guardUndefinedOrTrue := output.NewBinaryOperatorExpr(
		output.BinaryOperatorOr,
		guardNotDefined,
		guardExpr,
		nil,
		nil,
	)
	return output.NewBinaryOperatorExpr(
		output.BinaryOperatorAnd,
		guardUndefinedOrTrue,
		expr,
		nil,
		nil,
	)
}

// WrapReference wraps a value in an R3Reference
func WrapReference(value interface{}) R3Reference {
	wrapped := output.NewWrappedNodeExpr(value, nil, nil)
	return R3Reference{
		Value: wrapped,
		Type:  wrapped,
	}
}

// RefsToArray converts references to an array expression
func RefsToArray(refs []R3Reference, shouldForwardDeclare bool) output.OutputExpression {
	values := make([]output.OutputExpression, len(refs))
	for i, ref := range refs {
		values[i] = ref.Value
	}
	arrExpr := output.NewLiteralArrayExpr(values, nil, nil)
	if shouldForwardDeclare {
		return output.NewArrowFunctionExpr(nil, arrExpr, nil, nil)
	}
	return arrExpr
}

// MaybeForwardRefExpression describes an expression that may have been wrapped in a forwardRef() guard
type MaybeForwardRefExpression struct {
	// The unwrapped expression
	Expression output.OutputExpression
	// Specified whether the expression contains a reference to something that has not yet been defined
	ForwardRef ForwardRefHandling
}

// CreateMaybeForwardRefExpression creates a MaybeForwardRefExpression
func CreateMaybeForwardRefExpression(expr output.OutputExpression, forwardRef ForwardRefHandling) MaybeForwardRefExpression {
	return MaybeForwardRefExpression{
		Expression: expr,
		ForwardRef: forwardRef,
	}
}

// ConvertFromMaybeForwardRefExpression converts a MaybeForwardRefExpression to an Expression
func ConvertFromMaybeForwardRefExpression(maybeRef MaybeForwardRefExpression) output.OutputExpression {
	switch maybeRef.ForwardRef {
	case ForwardRefHandlingNone, ForwardRefHandlingWrapped:
		return maybeRef.Expression
	case ForwardRefHandlingUnwrapped:
		return GenerateForwardRef(maybeRef.Expression)
	default:
		return maybeRef.Expression
	}
}

// GenerateForwardRef generates an expression that has the given expr wrapped in forwardRef(() => expr)
func GenerateForwardRef(expr output.OutputExpression) output.OutputExpression {
	forwardRefExpr := output.NewExternalExpr(r3_identifiers.ForwardRef, nil, nil, nil)
	return output.NewInvokeFunctionExpr(forwardRefExpr, []output.OutputExpression{
		output.NewArrowFunctionExpr(nil, expr, nil, nil),
	}, nil, nil, false)
}

// ForwardRefHandling specifies how a forward ref has been handled
type ForwardRefHandling int

const (
	// ForwardRefHandlingNone means the expression was not wrapped in a forwardRef() call
	ForwardRefHandlingNone ForwardRefHandling = iota
	// ForwardRefHandlingWrapped means the expression is still wrapped in a forwardRef() call
	ForwardRefHandlingWrapped
	// ForwardRefHandlingUnwrapped means the expression was wrapped but has since been unwrapped
	ForwardRefHandlingUnwrapped
)

// EscapeIdentifier escapes an identifier for use in code generation
func EscapeIdentifier(name string, escapeDollar, escapeUnicode bool) string {
	return output.EscapeIdentifier(name, escapeDollar, escapeUnicode)
}
