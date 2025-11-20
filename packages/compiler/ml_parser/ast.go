package ml_parser

import "ngc-go/packages/compiler/util"

// Node represents a node in the HTML/XML AST
type Node interface {
	SourceSpan() *util.ParseSourceSpan
	Visit(visitor Visitor, context interface{}) interface{}
}

// NodeWithI18n is the base class for nodes that can have i18n metadata
type NodeWithI18n struct {
	sourceSpan *util.ParseSourceSpan
	i18n       interface{} // I18nMeta - will be defined later
}

// SourceSpan returns the source span
func (n *NodeWithI18n) SourceSpan() *util.ParseSourceSpan {
	return n.sourceSpan
}

// I18n returns the i18n metadata
func (n *NodeWithI18n) I18n() interface{} {
	return n.i18n
}

// SetI18n sets the i18n metadata
func (n *NodeWithI18n) SetI18n(i18n interface{}) {
	n.i18n = i18n
}

// Text represents a text node
type Text struct {
	*NodeWithI18n
	Value  string
	Tokens []InterpolatedTextToken
}

// NewText creates a new Text node
func NewText(value string, sourceSpan *util.ParseSourceSpan, tokens []InterpolatedTextToken, i18n interface{}) *Text {
	return &Text{
		NodeWithI18n: &NodeWithI18n{sourceSpan: sourceSpan, i18n: i18n},
		Value:        value,
		Tokens:       tokens,
	}
}

// Visit implements the Node interface
func (t *Text) Visit(visitor Visitor, context interface{}) interface{} {
	return visitor.VisitText(t, context)
}

// Expansion represents an ICU expansion node
type Expansion struct {
	*NodeWithI18n
	SwitchValue           string
	Type                  string
	Cases                 []*ExpansionCase
	SwitchValueSourceSpan *util.ParseSourceSpan
}

// NewExpansion creates a new Expansion node
func NewExpansion(switchValue, typ string, cases []*ExpansionCase, sourceSpan, switchValueSourceSpan *util.ParseSourceSpan, i18n interface{}) *Expansion {
	return &Expansion{
		NodeWithI18n:          &NodeWithI18n{sourceSpan: sourceSpan, i18n: i18n},
		SwitchValue:           switchValue,
		Type:                  typ,
		Cases:                 cases,
		SwitchValueSourceSpan: switchValueSourceSpan,
	}
}

// Visit implements the Node interface
func (e *Expansion) Visit(visitor Visitor, context interface{}) interface{} {
	return visitor.VisitExpansion(e, context)
}

// ExpansionCase represents an expansion case
type ExpansionCase struct {
	Value           string
	Expression      []Node
	sourceSpan      *util.ParseSourceSpan
	ValueSourceSpan *util.ParseSourceSpan
	ExpSourceSpan   *util.ParseSourceSpan
}

// NewExpansionCase creates a new ExpansionCase
func NewExpansionCase(value string, expression []Node, sourceSpan, valueSourceSpan, expSourceSpan *util.ParseSourceSpan) *ExpansionCase {
	return &ExpansionCase{
		Value:           value,
		Expression:      expression,
		sourceSpan:      sourceSpan,
		ValueSourceSpan: valueSourceSpan,
		ExpSourceSpan:   expSourceSpan,
	}
}

// Visit implements the Node interface
func (ec *ExpansionCase) Visit(visitor Visitor, context interface{}) interface{} {
	return visitor.VisitExpansionCase(ec, context)
}

// SourceSpan returns the source span
func (ec *ExpansionCase) SourceSpan() *util.ParseSourceSpan {
	return ec.sourceSpan
}

// Attribute represents an attribute node
type Attribute struct {
	*NodeWithI18n
	Name        string
	Value       string
	KeySpan     *util.ParseSourceSpan
	ValueSpan   *util.ParseSourceSpan
	ValueTokens []InterpolatedAttributeToken
}

// NewAttribute creates a new Attribute node
func NewAttribute(name, value string, sourceSpan, keySpan, valueSpan *util.ParseSourceSpan, valueTokens []InterpolatedAttributeToken, i18n interface{}) *Attribute {
	return &Attribute{
		NodeWithI18n: &NodeWithI18n{sourceSpan: sourceSpan, i18n: i18n},
		Name:         name,
		Value:        value,
		KeySpan:      keySpan,
		ValueSpan:    valueSpan,
		ValueTokens:  valueTokens,
	}
}

// Visit implements the Node interface
func (a *Attribute) Visit(visitor Visitor, context interface{}) interface{} {
	return visitor.VisitAttribute(a, context)
}

// Element represents an element node
type Element struct {
	*NodeWithI18n
	Name            string
	Attrs           []*Attribute
	Directives      []*Directive
	Children        []Node
	IsSelfClosing   bool
	StartSourceSpan *util.ParseSourceSpan
	EndSourceSpan   *util.ParseSourceSpan
	IsVoid          bool
}

// NewElement creates a new Element node
func NewElement(name string, attrs []*Attribute, directives []*Directive, children []Node, isSelfClosing bool, sourceSpan, startSourceSpan, endSourceSpan *util.ParseSourceSpan, isVoid bool, i18n interface{}) *Element {
	return &Element{
		NodeWithI18n:    &NodeWithI18n{sourceSpan: sourceSpan, i18n: i18n},
		Name:            name,
		Attrs:           attrs,
		Directives:      directives,
		Children:        children,
		IsSelfClosing:   isSelfClosing,
		StartSourceSpan: startSourceSpan,
		EndSourceSpan:   endSourceSpan,
		IsVoid:          isVoid,
	}
}

// Visit implements the Node interface
func (e *Element) Visit(visitor Visitor, context interface{}) interface{} {
	return visitor.VisitElement(e, context)
}

// Comment represents a comment node
type Comment struct {
	Value      *string
	sourceSpan *util.ParseSourceSpan
}

// NewComment creates a new Comment node
func NewComment(value *string, sourceSpan *util.ParseSourceSpan) *Comment {
	return &Comment{
		Value:      value,
		sourceSpan: sourceSpan,
	}
}

// SourceSpan returns the source span
func (c *Comment) SourceSpan() *util.ParseSourceSpan {
	return c.sourceSpan
}

// Visit implements the Node interface
func (c *Comment) Visit(visitor Visitor, context interface{}) interface{} {
	return visitor.VisitComment(c, context)
}

// Block represents a block node
type Block struct {
	*NodeWithI18n
	Name            string
	Parameters      []*BlockParameter
	Children        []Node
	NameSpan        *util.ParseSourceSpan
	StartSourceSpan *util.ParseSourceSpan
	EndSourceSpan   *util.ParseSourceSpan
}

// NewBlock creates a new Block node
func NewBlock(name string, parameters []*BlockParameter, children []Node, sourceSpan, nameSpan, startSourceSpan, endSourceSpan *util.ParseSourceSpan, i18n interface{}) *Block {
	return &Block{
		NodeWithI18n:    &NodeWithI18n{sourceSpan: sourceSpan, i18n: i18n},
		Name:            name,
		Parameters:      parameters,
		Children:        children,
		NameSpan:        nameSpan,
		StartSourceSpan: startSourceSpan,
		EndSourceSpan:   endSourceSpan,
	}
}

// Visit implements the Node interface
func (b *Block) Visit(visitor Visitor, context interface{}) interface{} {
	return visitor.VisitBlock(b, context)
}

// Component represents a component node
type Component struct {
	*NodeWithI18n
	ComponentName   string
	TagName         *string
	FullName        string
	Attrs           []*Attribute
	Directives      []*Directive
	Children        []Node
	IsSelfClosing   bool
	StartSourceSpan *util.ParseSourceSpan
	EndSourceSpan   *util.ParseSourceSpan
}

// NewComponent creates a new Component node
func NewComponent(componentName string, tagName *string, fullName string, attrs []*Attribute, directives []*Directive, children []Node, isSelfClosing bool, sourceSpan, startSourceSpan, endSourceSpan *util.ParseSourceSpan, i18n interface{}) *Component {
	return &Component{
		NodeWithI18n:    &NodeWithI18n{sourceSpan: sourceSpan, i18n: i18n},
		ComponentName:   componentName,
		TagName:         tagName,
		FullName:        fullName,
		Attrs:           attrs,
		Directives:      directives,
		Children:        children,
		IsSelfClosing:   isSelfClosing,
		StartSourceSpan: startSourceSpan,
		EndSourceSpan:   endSourceSpan,
	}
}

// Visit implements the Node interface
func (c *Component) Visit(visitor Visitor, context interface{}) interface{} {
	return visitor.VisitComponent(c, context)
}

// Directive represents a directive node
type Directive struct {
	Name            string
	Attrs           []*Attribute
	sourceSpan      *util.ParseSourceSpan
	StartSourceSpan *util.ParseSourceSpan
	EndSourceSpan   *util.ParseSourceSpan
}

// NewDirective creates a new Directive node
func NewDirective(name string, attrs []*Attribute, sourceSpan, startSourceSpan, endSourceSpan *util.ParseSourceSpan) *Directive {
	return &Directive{
		Name:            name,
		Attrs:           attrs,
		sourceSpan:      sourceSpan,
		StartSourceSpan: startSourceSpan,
		EndSourceSpan:   endSourceSpan,
	}
}

// SourceSpan returns the source span
func (d *Directive) SourceSpan() *util.ParseSourceSpan {
	return d.sourceSpan
}

// Visit implements the Node interface
func (d *Directive) Visit(visitor Visitor, context interface{}) interface{} {
	return visitor.VisitDirective(d, context)
}

// BlockParameter represents a block parameter
type BlockParameter struct {
	Expression string
	sourceSpan *util.ParseSourceSpan
}

// NewBlockParameter creates a new BlockParameter node
func NewBlockParameter(expression string, sourceSpan *util.ParseSourceSpan) *BlockParameter {
	return &BlockParameter{
		Expression: expression,
		sourceSpan: sourceSpan,
	}
}

// SourceSpan returns the source span
func (bp *BlockParameter) SourceSpan() *util.ParseSourceSpan {
	return bp.sourceSpan
}

// Visit implements the Node interface
func (bp *BlockParameter) Visit(visitor Visitor, context interface{}) interface{} {
	return visitor.VisitBlockParameter(bp, context)
}

// LetDeclaration represents a let declaration
type LetDeclaration struct {
	Name       string
	Value      string
	sourceSpan *util.ParseSourceSpan
	NameSpan   *util.ParseSourceSpan
	ValueSpan  *util.ParseSourceSpan
}

// NewLetDeclaration creates a new LetDeclaration node
func NewLetDeclaration(name, value string, sourceSpan, nameSpan, valueSpan *util.ParseSourceSpan) *LetDeclaration {
	return &LetDeclaration{
		Name:       name,
		Value:      value,
		sourceSpan: sourceSpan,
		NameSpan:   nameSpan,
		ValueSpan:  valueSpan,
	}
}

// SourceSpan returns the source span
func (ld *LetDeclaration) SourceSpan() *util.ParseSourceSpan {
	return ld.sourceSpan
}

// Visit implements the Node interface
func (ld *LetDeclaration) Visit(visitor Visitor, context interface{}) interface{} {
	return visitor.VisitLetDeclaration(ld, context)
}

// Visitor interface for visiting AST nodes
type Visitor interface {
	Visit(node Node, context interface{}) interface{}
	VisitElement(element *Element, context interface{}) interface{}
	VisitAttribute(attribute *Attribute, context interface{}) interface{}
	VisitText(text *Text, context interface{}) interface{}
	VisitComment(comment *Comment, context interface{}) interface{}
	VisitExpansion(expansion *Expansion, context interface{}) interface{}
	VisitExpansionCase(expansionCase *ExpansionCase, context interface{}) interface{}
	VisitBlock(block *Block, context interface{}) interface{}
	VisitBlockParameter(parameter *BlockParameter, context interface{}) interface{}
	VisitLetDeclaration(decl *LetDeclaration, context interface{}) interface{}
	VisitComponent(component *Component, context interface{}) interface{}
	VisitDirective(directive *Directive, context interface{}) interface{}
}

// VisitAll visits all nodes with a visitor
func VisitAll(visitor Visitor, nodes []Node, context interface{}) []interface{} {
	var result []interface{}

	for _, ast := range nodes {
		var astResult interface{}
		// Check if visitor has a Visit method
		if visitorWithVisit, ok := visitor.(interface {
			Visit(node Node, context interface{}) interface{}
		}); ok {
			astResult = visitorWithVisit.Visit(ast, context)
			if astResult == nil {
				astResult = ast.Visit(visitor, context)
			}
		} else {
			astResult = ast.Visit(visitor, context)
		}

		if astResult != nil {
			result = append(result, astResult)
		}
	}

	return result
}

// RecursiveVisitor is a base visitor that recursively visits children
type RecursiveVisitor struct{}

// NewRecursiveVisitor creates a new RecursiveVisitor
func NewRecursiveVisitor() *RecursiveVisitor {
	return &RecursiveVisitor{}
}

// VisitElement visits an element and its children
func (r *RecursiveVisitor) VisitElement(ast *Element, context interface{}) interface{} {
	r.visitChildren(context, func(visit func(children []Node)) {
		// Convert attrs and directives to Node slices for visiting
		// This is a simplified version - actual implementation may differ
		if len(ast.Children) > 0 {
			visit(ast.Children)
		}
	})
	return nil
}

// VisitAttribute visits an attribute
func (r *RecursiveVisitor) VisitAttribute(ast *Attribute, context interface{}) interface{} {
	return nil
}

// VisitText visits a text node
func (r *RecursiveVisitor) VisitText(ast *Text, context interface{}) interface{} {
	return nil
}

// VisitComment visits a comment node
func (r *RecursiveVisitor) VisitComment(ast *Comment, context interface{}) interface{} {
	return nil
}

// VisitExpansion visits an expansion node
func (r *RecursiveVisitor) VisitExpansion(ast *Expansion, context interface{}) interface{} {
	r.visitChildren(context, func(visit func(children []Node)) {
		// Visit expansion cases
		for _, c := range ast.Cases {
			if c.Expression != nil {
				visit(c.Expression)
			}
		}
	})
	return nil
}

// VisitExpansionCase visits an expansion case
func (r *RecursiveVisitor) VisitExpansionCase(ast *ExpansionCase, context interface{}) interface{} {
	return nil
}

// VisitBlock visits a block node
func (r *RecursiveVisitor) VisitBlock(block *Block, context interface{}) interface{} {
	r.visitChildren(context, func(visit func(children []Node)) {
		if len(block.Children) > 0 {
			visit(block.Children)
		}
	})
	return nil
}

// VisitBlockParameter visits a block parameter
func (r *RecursiveVisitor) VisitBlockParameter(ast *BlockParameter, context interface{}) interface{} {
	return nil
}

// VisitLetDeclaration visits a let declaration
func (r *RecursiveVisitor) VisitLetDeclaration(decl *LetDeclaration, context interface{}) interface{} {
	return nil
}

// VisitComponent visits a component node
func (r *RecursiveVisitor) VisitComponent(component *Component, context interface{}) interface{} {
	r.visitChildren(context, func(visit func(children []Node)) {
		if len(component.Children) > 0 {
			visit(component.Children)
		}
	})
	return nil
}

// VisitDirective visits a directive node
func (r *RecursiveVisitor) VisitDirective(directive *Directive, context interface{}) interface{} {
	return nil
}

// Visit is the default visit method
func (r *RecursiveVisitor) Visit(node Node, context interface{}) interface{} {
	return node.Visit(r, context)
}

func (r *RecursiveVisitor) visitChildren(context interface{}, cb func(visit func(children []Node))) {
	var results [][]interface{}

	visit := func(children []Node) {
		if children != nil {
			results = append(results, VisitAll(r, children, context))
		}
	}

	cb(visit)

	// Flatten results - this is a simplified version
	// Results are collected but not used in base implementation
	_ = results
}
