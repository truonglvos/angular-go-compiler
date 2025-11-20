package expression_parser_test

import (
	"testing"

	"ngc-go/packages/compiler/src/expression_parser"
)

// Visitor extends RecursiveAstVisitor to collect visited nodes
type Visitor struct {
	expression_parser.RecursiveAstVisitor
	path []expression_parser.AST
}

// Visit overrides the default visit to collect nodes
func (v *Visitor) Visit(ast expression_parser.AST, context interface{}) interface{} {
	v.path = append(v.path, ast)
	ast.Visit(v, context)
	return nil
}

// VisitCall overrides to visit receiver and args
func (v *Visitor) VisitCall(ast *expression_parser.Call, context interface{}) interface{} {
	// Node already added in Visit, just visit children
	v.Visit(ast.Receiver, context)
	v.VisitAll(ast.Args, context)
	return nil
}

// VisitPropertyRead overrides to visit receiver
func (v *Visitor) VisitPropertyRead(ast *expression_parser.PropertyRead, context interface{}) interface{} {
	// Node already added in Visit, just visit children
	v.Visit(ast.Receiver, context)
	return nil
}

// VisitImplicitReceiver overrides to just collect the node
func (v *Visitor) VisitImplicitReceiver(ast *expression_parser.ImplicitReceiver, context interface{}) interface{} {
	// Node already added in Visit, no children to visit
	return nil
}

func TestRecursiveAstVisitor(t *testing.T) {
	t.Run("should visit every node", func(t *testing.T) {
		lexer := expression_parser.NewLexer()
		parser := expression_parser.NewParser(lexer, false)
		ast := parser.ParseBinding("x.y()", getFakeSpan(""), 0)

		visitor := &Visitor{
			path: []expression_parser.AST{},
		}
		visitor.Visit(ast.AST, nil)
		path := visitor.path

		// If the visitor method of RecursiveAstVisitor is implemented correctly,
		// then we should have collected the full path from root to leaf.
		if len(path) != 4 {
			t.Fatalf("Expected path length 4, got %d", len(path))
		}

		// Check types - in Go we need to use type assertions
		call, ok := path[0].(*expression_parser.Call)
		if !ok {
			t.Errorf("Expected first node to be Call, got %T", path[0])
		}

		yRead, ok := path[1].(*expression_parser.PropertyRead)
		if !ok {
			t.Errorf("Expected second node to be PropertyRead, got %T", path[1])
		}

		xRead, ok := path[2].(*expression_parser.PropertyRead)
		if !ok {
			t.Errorf("Expected third node to be PropertyRead, got %T", path[2])
		}

		_, ok = path[3].(*expression_parser.ImplicitReceiver)
		if !ok {
			t.Errorf("Expected fourth node to be ImplicitReceiver, got %T", path[3])
		}

		if xRead != nil && xRead.Name != "x" {
			t.Errorf("Expected xRead.name to be 'x', got %q", xRead.Name)
		}

		if yRead != nil && yRead.Name != "y" {
			t.Errorf("Expected yRead.name to be 'y', got %q", yRead.Name)
		}

		if call != nil && len(call.Args) != 0 {
			t.Errorf("Expected call.args to be empty, got %d args", len(call.Args))
		}
	})
}
