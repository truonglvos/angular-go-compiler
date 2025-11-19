package i18n

import (
	"ngc-go/packages/compiler/util"
)

// MessagePlaceholder describes the text contents of a placeholder as it appears in an ICU expression
type MessagePlaceholder struct {
	// Text is the text contents of the placeholder
	Text string

	// SourceSpan is the source span of the placeholder
	SourceSpan *util.ParseSourceSpan
}

// MessageSpan represents a span in the source file
// line and columns indexes are 1 based
type MessageSpan struct {
	FilePath  string
	StartLine int
	StartCol  int
	EndLine   int
	EndCol    int
}

// Message represents an i18n message
type Message struct {
	Sources              []MessageSpan
	ID                   string
	LegacyIDs            []string
	MessageString        string
	Nodes                []Node
	Placeholders         map[string]MessagePlaceholder
	PlaceholderToMessage map[string]*Message
	Meaning              string
	Description          string
	CustomID             string
}

// NewMessage creates a new Message
func NewMessage(
	nodes []Node,
	placeholders map[string]MessagePlaceholder,
	placeholderToMessage map[string]*Message,
	meaning string,
	description string,
	customID string,
) *Message {
	msg := &Message{
		Nodes:                nodes,
		Placeholders:         placeholders,
		PlaceholderToMessage: placeholderToMessage,
		Meaning:              meaning,
		Description:          description,
		CustomID:             customID,
		LegacyIDs:            []string{},
	}

	msg.ID = msg.CustomID
	msg.MessageString = SerializeMessage(msg.Nodes)

	if len(nodes) > 0 {
		firstNode := nodes[0]
		lastNode := nodes[len(nodes)-1]
		msg.Sources = []MessageSpan{
			{
				FilePath:  firstNode.SourceSpan().Start.File.URL,
				StartLine: firstNode.SourceSpan().Start.Line + 1,
				StartCol:  firstNode.SourceSpan().Start.Col + 1,
				EndLine:   lastNode.SourceSpan().End.Line + 1,
				EndCol:    firstNode.SourceSpan().Start.Col + 1,
			},
		}
	} else {
		msg.Sources = []MessageSpan{}
	}

	return msg
}

// Node is the base interface for all i18n AST nodes
type Node interface {
	SourceSpan() *util.ParseSourceSpan
	Visit(visitor Visitor, context interface{}) interface{}
}

// Text represents a text node
type Text struct {
	Value      string
	sourceSpan *util.ParseSourceSpan
}

// NewText creates a new Text node
func NewText(value string, sourceSpan *util.ParseSourceSpan) *Text {
	return &Text{
		Value:      value,
		sourceSpan: sourceSpan,
	}
}

// SourceSpan returns the source span
func (t *Text) SourceSpan() *util.ParseSourceSpan {
	return t.sourceSpan
}

// Visit visits the node with a visitor
func (t *Text) Visit(visitor Visitor, context interface{}) interface{} {
	return visitor.VisitText(t, context)
}

// Container represents a container node
type Container struct {
	Children   []Node
	sourceSpan *util.ParseSourceSpan
}

// NewContainer creates a new Container node
func NewContainer(children []Node, sourceSpan *util.ParseSourceSpan) *Container {
	return &Container{
		Children:   children,
		sourceSpan: sourceSpan,
	}
}

// SourceSpan returns the source span
func (c *Container) SourceSpan() *util.ParseSourceSpan {
	return c.sourceSpan
}

// Visit visits the node with a visitor
func (c *Container) Visit(visitor Visitor, context interface{}) interface{} {
	return visitor.VisitContainer(c, context)
}

// Icu represents an ICU message node
type Icu struct {
	Expression            string
	Type                  string
	Cases                 map[string]Node
	sourceSpan            *util.ParseSourceSpan
	ExpressionPlaceholder string
}

// NewIcu creates a new Icu node
func NewIcu(expression string, icuType string, cases map[string]Node, sourceSpan *util.ParseSourceSpan, expressionPlaceholder string) *Icu {
	return &Icu{
		Expression:            expression,
		Type:                  icuType,
		Cases:                 cases,
		sourceSpan:            sourceSpan,
		ExpressionPlaceholder: expressionPlaceholder,
	}
}

// SourceSpan returns the source span
func (i *Icu) SourceSpan() *util.ParseSourceSpan {
	return i.sourceSpan
}

// Visit visits the node with a visitor
func (i *Icu) Visit(visitor Visitor, context interface{}) interface{} {
	return visitor.VisitIcu(i, context)
}

// TagPlaceholder represents a tag placeholder node
type TagPlaceholder struct {
	Tag             string
	Attrs           map[string]string
	StartName       string
	CloseName       string
	Children        []Node
	IsVoid          bool
	sourceSpan      *util.ParseSourceSpan
	StartSourceSpan *util.ParseSourceSpan
	EndSourceSpan   *util.ParseSourceSpan
}

// NewTagPlaceholder creates a new TagPlaceholder node
func NewTagPlaceholder(
	tag string,
	attrs map[string]string,
	startName string,
	closeName string,
	children []Node,
	isVoid bool,
	sourceSpan *util.ParseSourceSpan,
	startSourceSpan *util.ParseSourceSpan,
	endSourceSpan *util.ParseSourceSpan,
) *TagPlaceholder {
	return &TagPlaceholder{
		Tag:             tag,
		Attrs:           attrs,
		StartName:       startName,
		CloseName:       closeName,
		Children:        children,
		IsVoid:          isVoid,
		sourceSpan:      sourceSpan,
		StartSourceSpan: startSourceSpan,
		EndSourceSpan:   endSourceSpan,
	}
}

// SourceSpan returns the source span
func (t *TagPlaceholder) SourceSpan() *util.ParseSourceSpan {
	return t.sourceSpan
}

// Visit visits the node with a visitor
func (t *TagPlaceholder) Visit(visitor Visitor, context interface{}) interface{} {
	return visitor.VisitTagPlaceholder(t, context)
}

// Placeholder represents a placeholder node
type Placeholder struct {
	Value      string
	Name       string
	sourceSpan *util.ParseSourceSpan
}

// NewPlaceholder creates a new Placeholder node
func NewPlaceholder(value string, name string, sourceSpan *util.ParseSourceSpan) *Placeholder {
	return &Placeholder{
		Value:      value,
		Name:       name,
		sourceSpan: sourceSpan,
	}
}

// SourceSpan returns the source span
func (p *Placeholder) SourceSpan() *util.ParseSourceSpan {
	return p.sourceSpan
}

// Visit visits the node with a visitor
func (p *Placeholder) Visit(visitor Visitor, context interface{}) interface{} {
	return visitor.VisitPlaceholder(p, context)
}

// IcuPlaceholder represents an ICU placeholder node
type IcuPlaceholder struct {
	Value           *Icu
	Name            string
	sourceSpan      *util.ParseSourceSpan
	PreviousMessage *Message
}

// NewIcuPlaceholder creates a new IcuPlaceholder node
func NewIcuPlaceholder(value *Icu, name string, sourceSpan *util.ParseSourceSpan) *IcuPlaceholder {
	return &IcuPlaceholder{
		Value:      value,
		Name:       name,
		sourceSpan: sourceSpan,
	}
}

// SourceSpan returns the source span
func (i *IcuPlaceholder) SourceSpan() *util.ParseSourceSpan {
	return i.sourceSpan
}

// Visit visits the node with a visitor
func (i *IcuPlaceholder) Visit(visitor Visitor, context interface{}) interface{} {
	return visitor.VisitIcuPlaceholder(i, context)
}

// BlockPlaceholder represents a block placeholder node
type BlockPlaceholder struct {
	Name            string
	Parameters      []string
	StartName       string
	CloseName       string
	Children        []Node
	sourceSpan      *util.ParseSourceSpan
	StartSourceSpan *util.ParseSourceSpan
	EndSourceSpan   *util.ParseSourceSpan
}

// NewBlockPlaceholder creates a new BlockPlaceholder node
func NewBlockPlaceholder(
	name string,
	parameters []string,
	startName string,
	closeName string,
	children []Node,
	sourceSpan *util.ParseSourceSpan,
	startSourceSpan *util.ParseSourceSpan,
	endSourceSpan *util.ParseSourceSpan,
) *BlockPlaceholder {
	return &BlockPlaceholder{
		Name:            name,
		Parameters:      parameters,
		StartName:       startName,
		CloseName:       closeName,
		Children:        children,
		sourceSpan:      sourceSpan,
		StartSourceSpan: startSourceSpan,
		EndSourceSpan:   endSourceSpan,
	}
}

// SourceSpan returns the source span
func (b *BlockPlaceholder) SourceSpan() *util.ParseSourceSpan {
	return b.sourceSpan
}

// Visit visits the node with a visitor
func (b *BlockPlaceholder) Visit(visitor Visitor, context interface{}) interface{} {
	return visitor.VisitBlockPlaceholder(b, context)
}

// I18nMeta represents i18n metadata
// Each HTML node that is affected by an i18n tag will also have an `i18n` property that is of type
// `I18nMeta`. This information is either a `*Message`, which indicates it is the root of an i18n message,
// or a `Node`, which indicates is it part of a containing `Message`.
type I18nMeta interface{}

// Visitor is the interface for visiting i18n AST nodes
type Visitor interface {
	VisitText(text *Text, context interface{}) interface{}
	VisitContainer(container *Container, context interface{}) interface{}
	VisitIcu(icu *Icu, context interface{}) interface{}
	VisitTagPlaceholder(ph *TagPlaceholder, context interface{}) interface{}
	VisitPlaceholder(ph *Placeholder, context interface{}) interface{}
	VisitIcuPlaceholder(ph *IcuPlaceholder, context interface{}) interface{}
	VisitBlockPlaceholder(ph *BlockPlaceholder, context interface{}) interface{}
}

// CloneVisitor clones the AST
type CloneVisitor struct{}

// NewCloneVisitor creates a new CloneVisitor
func NewCloneVisitor() *CloneVisitor {
	return &CloneVisitor{}
}

// VisitText clones a Text node
func (v *CloneVisitor) VisitText(text *Text, context interface{}) interface{} {
	return NewText(text.Value, text.sourceSpan)
}

// VisitContainer clones a Container node
func (v *CloneVisitor) VisitContainer(container *Container, context interface{}) interface{} {
	children := make([]Node, len(container.Children))
	for i, n := range container.Children {
		children[i] = n.Visit(v, context).(Node)
	}
	return NewContainer(children, container.sourceSpan)
}

// VisitIcu clones an Icu node
func (v *CloneVisitor) VisitIcu(icu *Icu, context interface{}) interface{} {
	cases := make(map[string]Node)
	for key, node := range icu.Cases {
		cases[key] = node.Visit(v, context).(Node)
	}
	return NewIcu(icu.Expression, icu.Type, cases, icu.sourceSpan, icu.ExpressionPlaceholder)
}

// VisitTagPlaceholder clones a TagPlaceholder node
func (v *CloneVisitor) VisitTagPlaceholder(ph *TagPlaceholder, context interface{}) interface{} {
	children := make([]Node, len(ph.Children))
	for i, n := range ph.Children {
		children[i] = n.Visit(v, context).(Node)
	}
	return NewTagPlaceholder(
		ph.Tag,
		ph.Attrs,
		ph.StartName,
		ph.CloseName,
		children,
		ph.IsVoid,
		ph.sourceSpan,
		ph.StartSourceSpan,
		ph.EndSourceSpan,
	)
}

// VisitPlaceholder clones a Placeholder node
func (v *CloneVisitor) VisitPlaceholder(ph *Placeholder, context interface{}) interface{} {
	return NewPlaceholder(ph.Value, ph.Name, ph.sourceSpan)
}

// VisitIcuPlaceholder clones an IcuPlaceholder node
func (v *CloneVisitor) VisitIcuPlaceholder(ph *IcuPlaceholder, context interface{}) interface{} {
	return NewIcuPlaceholder(ph.Value, ph.Name, ph.sourceSpan)
}

// VisitBlockPlaceholder clones a BlockPlaceholder node
func (v *CloneVisitor) VisitBlockPlaceholder(ph *BlockPlaceholder, context interface{}) interface{} {
	children := make([]Node, len(ph.Children))
	for i, n := range ph.Children {
		children[i] = n.Visit(v, context).(Node)
	}
	return NewBlockPlaceholder(
		ph.Name,
		ph.Parameters,
		ph.StartName,
		ph.CloseName,
		children,
		ph.sourceSpan,
		ph.StartSourceSpan,
		ph.EndSourceSpan,
	)
}

// RecurseVisitor visits all the nodes recursively
type RecurseVisitor struct{}

// NewRecurseVisitor creates a new RecurseVisitor
func NewRecurseVisitor() *RecurseVisitor {
	return &RecurseVisitor{}
}

// VisitText visits a Text node
func (v *RecurseVisitor) VisitText(text *Text, context interface{}) interface{} {
	return nil
}

// VisitContainer visits a Container node
func (v *RecurseVisitor) VisitContainer(container *Container, context interface{}) interface{} {
	for _, child := range container.Children {
		child.Visit(v, nil)
	}
	return nil
}

// VisitIcu visits an Icu node
func (v *RecurseVisitor) VisitIcu(icu *Icu, context interface{}) interface{} {
	for _, node := range icu.Cases {
		node.Visit(v, nil)
	}
	return nil
}

// VisitTagPlaceholder visits a TagPlaceholder node
func (v *RecurseVisitor) VisitTagPlaceholder(ph *TagPlaceholder, context interface{}) interface{} {
	for _, child := range ph.Children {
		child.Visit(v, nil)
	}
	return nil
}

// VisitPlaceholder visits a Placeholder node
func (v *RecurseVisitor) VisitPlaceholder(ph *Placeholder, context interface{}) interface{} {
	return nil
}

// VisitIcuPlaceholder visits an IcuPlaceholder node
func (v *RecurseVisitor) VisitIcuPlaceholder(ph *IcuPlaceholder, context interface{}) interface{} {
	return nil
}

// VisitBlockPlaceholder visits a BlockPlaceholder node
func (v *RecurseVisitor) VisitBlockPlaceholder(ph *BlockPlaceholder, context interface{}) interface{} {
	for _, child := range ph.Children {
		child.Visit(v, nil)
	}
	return nil
}

// SerializeMessage serializes the message to the Localize backtick string format
func SerializeMessage(messageNodes []Node) string {
	visitor := &LocalizeMessageStringVisitor{}
	parts := make([]string, len(messageNodes))
	for i, n := range messageNodes {
		result := n.Visit(visitor, nil)
		if str, ok := result.(string); ok {
			parts[i] = str
		} else {
			parts[i] = ""
		}
	}

	// Join all parts
	result := ""
	for _, part := range parts {
		result += part
	}
	return result
}

// LocalizeMessageStringVisitor serializes nodes to Localize format
type LocalizeMessageStringVisitor struct{}

// VisitText serializes a Text node
func (v *LocalizeMessageStringVisitor) VisitText(text *Text, context interface{}) interface{} {
	return text.Value
}

// VisitContainer serializes a Container node
func (v *LocalizeMessageStringVisitor) VisitContainer(container *Container, context interface{}) interface{} {
	parts := make([]string, len(container.Children))
	for i, child := range container.Children {
		result := child.Visit(v, nil)
		if str, ok := result.(string); ok {
			parts[i] = str
		} else {
			parts[i] = ""
		}
	}

	result := ""
	for _, part := range parts {
		result += part
	}
	return result
}

// VisitIcu serializes an Icu node
func (v *LocalizeMessageStringVisitor) VisitIcu(icu *Icu, context interface{}) interface{} {
	strCases := make([]string, 0, len(icu.Cases))
	for k, node := range icu.Cases {
		result := node.Visit(v, nil)
		if str, ok := result.(string); ok {
			strCases = append(strCases, k+" {"+str+"}")
		}
	}

	casesStr := ""
	for i, c := range strCases {
		if i > 0 {
			casesStr += " "
		}
		casesStr += c
	}
	return "{${" + icu.ExpressionPlaceholder + "}, " + icu.Type + ", " + casesStr + "}"
}

// VisitTagPlaceholder serializes a TagPlaceholder node
func (v *LocalizeMessageStringVisitor) VisitTagPlaceholder(ph *TagPlaceholder, context interface{}) interface{} {
	parts := make([]string, len(ph.Children))
	for i, child := range ph.Children {
		result := child.Visit(v, nil)
		if str, ok := result.(string); ok {
			parts[i] = str
		} else {
			parts[i] = ""
		}
	}

	children := ""
	for _, part := range parts {
		children += part
	}
	return "{${" + ph.StartName + "}}" + children + "{${" + ph.CloseName + "}}"
}

// VisitPlaceholder serializes a Placeholder node
func (v *LocalizeMessageStringVisitor) VisitPlaceholder(ph *Placeholder, context interface{}) interface{} {
	return "{${" + ph.Name + "}}"
}

// VisitIcuPlaceholder serializes an IcuPlaceholder node
func (v *LocalizeMessageStringVisitor) VisitIcuPlaceholder(ph *IcuPlaceholder, context interface{}) interface{} {
	return "{${" + ph.Name + "}}"
}

// VisitBlockPlaceholder serializes a BlockPlaceholder node
func (v *LocalizeMessageStringVisitor) VisitBlockPlaceholder(ph *BlockPlaceholder, context interface{}) interface{} {
	parts := make([]string, len(ph.Children))
	for i, child := range ph.Children {
		result := child.Visit(v, nil)
		if str, ok := result.(string); ok {
			parts[i] = str
		} else {
			parts[i] = ""
		}
	}

	children := ""
	for _, part := range parts {
		children += part
	}
	return "{${" + ph.StartName + "}}" + children + "{${" + ph.CloseName + "}}"
}
