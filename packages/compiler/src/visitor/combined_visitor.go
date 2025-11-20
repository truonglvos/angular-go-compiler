package visitor

// ExpressionAST represents an expression AST node
// This interface allows us to avoid import cycle
type ExpressionAST interface {
	Visit(visitor ExpressionAstVisitor, context interface{}) interface{}
}

// ExpressionAstVisitor is the interface for visiting expression AST nodes
// This interface allows us to avoid import cycle
type ExpressionAstVisitor interface {
	Visit(ast ExpressionAST, context interface{}) interface{}
}

// Render3Node represents a node in the render3 AST
// This is a placeholder interface that will be properly defined when render3 AST is implemented
type Render3Node interface {
	Visit(visitor Render3Visitor) interface{}
}

// Render3Visitor is the interface for visiting render3 AST nodes
// This is a placeholder interface that will be properly defined when render3 AST is implemented
type Render3Visitor interface {
	VisitElement(element Render3Node) interface{}
	VisitTemplate(template Render3Node) interface{}
	VisitContent(content Render3Node) interface{}
	VisitVariable(variable Render3Node) interface{}
	VisitReference(reference Render3Node) interface{}
	VisitTextAttribute(attribute Render3Node) interface{}
	VisitBoundAttribute(attribute Render3Node) interface{}
	VisitBoundEvent(event Render3Node) interface{}
	VisitText(text Render3Node) interface{}
	VisitBoundText(text Render3Node) interface{}
	VisitIcu(icu Render3Node) interface{}
	VisitDeferredBlock(deferred Render3Node) interface{}
	VisitDeferredBlockPlaceholder(placeholder Render3Node) interface{}
	VisitDeferredBlockLoading(loading Render3Node) interface{}
	VisitDeferredBlockError(error Render3Node) interface{}
	VisitIfBlock(ifBlock Render3Node) interface{}
	VisitSwitchBlock(switchBlock Render3Node) interface{}
	VisitForBlock(forBlock Render3Node) interface{}
}

// RecursiveVisitor is a visitor that recursively visits nodes
type RecursiveVisitor interface {
	Visit(node Render3Node) interface{}
}

// RecurseVisitor is a visitor that can recurse through nodes
type RecurseVisitor struct {
	visitor Render3Visitor
}

// NewRecurseVisitor creates a new RecurseVisitor
func NewRecurseVisitor(visitor Render3Visitor) *RecurseVisitor {
	return &RecurseVisitor{
		visitor: visitor,
	}
}

// Visit visits a node recursively
func (rv *RecurseVisitor) Visit(node Render3Node) interface{} {
	return node.Visit(rv.visitor)
}

// CombinedVisitor combines multiple visitors
type CombinedVisitor struct {
	visitors []Render3Visitor
}

// NewCombinedVisitor creates a new CombinedVisitor
func NewCombinedVisitor(visitors ...Render3Visitor) *CombinedVisitor {
	return &CombinedVisitor{
		visitors: visitors,
	}
}

// VisitElement visits an element node
func (cv *CombinedVisitor) VisitElement(element Render3Node) interface{} {
	var result interface{}
	for _, visitor := range cv.visitors {
		result = visitor.VisitElement(element)
	}
	return result
}

// VisitTemplate visits a template node
func (cv *CombinedVisitor) VisitTemplate(template Render3Node) interface{} {
	var result interface{}
	for _, visitor := range cv.visitors {
		result = visitor.VisitTemplate(template)
	}
	return result
}

// VisitContent visits a content node
func (cv *CombinedVisitor) VisitContent(content Render3Node) interface{} {
	var result interface{}
	for _, visitor := range cv.visitors {
		result = visitor.VisitContent(content)
	}
	return result
}

// VisitVariable visits a variable node
func (cv *CombinedVisitor) VisitVariable(variable Render3Node) interface{} {
	var result interface{}
	for _, visitor := range cv.visitors {
		result = visitor.VisitVariable(variable)
	}
	return result
}

// VisitReference visits a reference node
func (cv *CombinedVisitor) VisitReference(reference Render3Node) interface{} {
	var result interface{}
	for _, visitor := range cv.visitors {
		result = visitor.VisitReference(reference)
	}
	return result
}

// VisitTextAttribute visits a text attribute node
func (cv *CombinedVisitor) VisitTextAttribute(attribute Render3Node) interface{} {
	var result interface{}
	for _, visitor := range cv.visitors {
		result = visitor.VisitTextAttribute(attribute)
	}
	return result
}

// VisitBoundAttribute visits a bound attribute node
func (cv *CombinedVisitor) VisitBoundAttribute(attribute Render3Node) interface{} {
	var result interface{}
	for _, visitor := range cv.visitors {
		result = visitor.VisitBoundAttribute(attribute)
	}
	return result
}

// VisitBoundEvent visits a bound event node
func (cv *CombinedVisitor) VisitBoundEvent(event Render3Node) interface{} {
	var result interface{}
	for _, visitor := range cv.visitors {
		result = visitor.VisitBoundEvent(event)
	}
	return result
}

// VisitText visits a text node
func (cv *CombinedVisitor) VisitText(text Render3Node) interface{} {
	var result interface{}
	for _, visitor := range cv.visitors {
		result = visitor.VisitText(text)
	}
	return result
}

// VisitBoundText visits a bound text node
func (cv *CombinedVisitor) VisitBoundText(text Render3Node) interface{} {
	var result interface{}
	for _, visitor := range cv.visitors {
		result = visitor.VisitBoundText(text)
	}
	return result
}

// VisitIcu visits an ICU node
func (cv *CombinedVisitor) VisitIcu(icu Render3Node) interface{} {
	var result interface{}
	for _, visitor := range cv.visitors {
		result = visitor.VisitIcu(icu)
	}
	return result
}

// VisitDeferredBlock visits a deferred block node
func (cv *CombinedVisitor) VisitDeferredBlock(deferred Render3Node) interface{} {
	var result interface{}
	for _, visitor := range cv.visitors {
		result = visitor.VisitDeferredBlock(deferred)
	}
	return result
}

// VisitDeferredBlockPlaceholder visits a deferred block placeholder node
func (cv *CombinedVisitor) VisitDeferredBlockPlaceholder(placeholder Render3Node) interface{} {
	var result interface{}
	for _, visitor := range cv.visitors {
		result = visitor.VisitDeferredBlockPlaceholder(placeholder)
	}
	return result
}

// VisitDeferredBlockLoading visits a deferred block loading node
func (cv *CombinedVisitor) VisitDeferredBlockLoading(loading Render3Node) interface{} {
	var result interface{}
	for _, visitor := range cv.visitors {
		result = visitor.VisitDeferredBlockLoading(loading)
	}
	return result
}

// VisitDeferredBlockError visits a deferred block error node
func (cv *CombinedVisitor) VisitDeferredBlockError(error Render3Node) interface{} {
	var result interface{}
	for _, visitor := range cv.visitors {
		result = visitor.VisitDeferredBlockError(error)
	}
	return result
}

// VisitIfBlock visits an if block node
func (cv *CombinedVisitor) VisitIfBlock(ifBlock Render3Node) interface{} {
	var result interface{}
	for _, visitor := range cv.visitors {
		result = visitor.VisitIfBlock(ifBlock)
	}
	return result
}

// VisitSwitchBlock visits a switch block node
func (cv *CombinedVisitor) VisitSwitchBlock(switchBlock Render3Node) interface{} {
	var result interface{}
	for _, visitor := range cv.visitors {
		result = visitor.VisitSwitchBlock(switchBlock)
	}
	return result
}

// VisitForBlock visits a for block node
func (cv *CombinedVisitor) VisitForBlock(forBlock Render3Node) interface{} {
	var result interface{}
	for _, visitor := range cv.visitors {
		result = visitor.VisitForBlock(forBlock)
	}
	return result
}

