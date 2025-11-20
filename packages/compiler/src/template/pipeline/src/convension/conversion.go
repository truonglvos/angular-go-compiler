package pipeline_convension

import (
	"ngc-go/packages/compiler/src/output"
	"ngc-go/packages/compiler/src/template/pipeline/ir"
)

// BinaryOperators maps binary operator strings to their corresponding output.BinaryOperator values
var BinaryOperators = map[string]output.BinaryOperator{
	"&&":  output.BinaryOperatorAnd,
	">":   output.BinaryOperatorBigger,
	">=":  output.BinaryOperatorBiggerEquals,
	"|":   output.BinaryOperatorBitwiseOr,
	"&":   output.BinaryOperatorBitwiseAnd,
	"/":   output.BinaryOperatorDivide,
	"=":   output.BinaryOperatorAssign,
	"==":  output.BinaryOperatorEquals,
	"===": output.BinaryOperatorIdentical,
	"<":   output.BinaryOperatorLower,
	"<=":  output.BinaryOperatorLowerEquals,
	"-":   output.BinaryOperatorMinus,
	"%":   output.BinaryOperatorModulo,
	"**":  output.BinaryOperatorExponentiation,
	"*":   output.BinaryOperatorMultiply,
	"!=":  output.BinaryOperatorNotEquals,
	"!==": output.BinaryOperatorNotIdentical,
	"??":  output.BinaryOperatorNullishCoalesce,
	"||":  output.BinaryOperatorOr,
	"+":   output.BinaryOperatorPlus,
	"in":  output.BinaryOperatorIn,
	"+=":  output.BinaryOperatorAdditionAssignment,
	"-=":  output.BinaryOperatorSubtractionAssignment,
	"*=":  output.BinaryOperatorMultiplicationAssignment,
	"/=":  output.BinaryOperatorDivisionAssignment,
	"%=":  output.BinaryOperatorRemainderAssignment,
	"**=": output.BinaryOperatorExponentiationAssignment,
	"&&=": output.BinaryOperatorAndAssignment,
	"||=": output.BinaryOperatorOrAssignment,
	"??=": output.BinaryOperatorNullishCoalesceAssignment,
}

// NamespaceForKey converts a namespace prefix key to an ir.Namespace
func NamespaceForKey(namespacePrefixKey *string) ir.Namespace {
	namespaces := map[string]ir.Namespace{
		"svg":  ir.NamespaceSVG,
		"math": ir.NamespaceMath,
	}
	if namespacePrefixKey == nil {
		return ir.NamespaceHTML
	}
	if ns, ok := namespaces[*namespacePrefixKey]; ok {
		return ns
	}
	return ir.NamespaceHTML
}

// KeyForNamespace converts an ir.Namespace to a namespace prefix key
func KeyForNamespace(namespace ir.Namespace) *string {
	namespaces := map[string]ir.Namespace{
		"svg":  ir.NamespaceSVG,
		"math": ir.NamespaceMath,
	}
	for k, n := range namespaces {
		if n == namespace {
			key := k
			return &key
		}
	}
	return nil // No namespace prefix for HTML
}

// PrefixWithNamespace prefixes a tag name with its namespace
func PrefixWithNamespace(strippedTag string, namespace ir.Namespace) string {
	if namespace == ir.NamespaceHTML {
		return strippedTag
	}
	key := KeyForNamespace(namespace)
	if key == nil {
		return strippedTag
	}
	return ":" + *key + ":" + strippedTag
}

// LiteralType represents a literal value type
type LiteralType interface{}

// LiteralOrArrayLiteral converts a literal value or array of literals to an output expression
func LiteralOrArrayLiteral(value LiteralType) output.OutputExpression {
	switch v := value.(type) {
	case []interface{}:
		entries := make([]output.OutputExpression, len(v))
		for i, item := range v {
			entries[i] = LiteralOrArrayLiteral(item)
		}
		return output.NewLiteralArrayExpr(entries, nil, nil)
	case string:
		return output.NewLiteralExpr(v, nil, nil)
	case int:
		return output.NewLiteralExpr(v, nil, nil)
	case int64:
		return output.NewLiteralExpr(v, nil, nil)
	case float64:
		return output.NewLiteralExpr(v, nil, nil)
	case bool:
		return output.NewLiteralExpr(v, nil, nil)
	case nil:
		return output.NewLiteralExpr(nil, nil, nil)
	default:
		// Fallback: try to convert to string
		return output.NewLiteralExpr(v, nil, nil)
	}
}
