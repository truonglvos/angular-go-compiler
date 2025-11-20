package serializers

import (
	"fmt"
	"ngc-go/packages/compiler/src/i18n"
)

// Serializer is the base interface for all serializers
type Serializer interface {
	// Write serializes messages to a string
	Write(messages []*i18n.Message, locale *string) string

	// Load loads messages from a string
	Load(content string, url string) (locale *string, i18nNodesByMsgID map[string][]i18n.Node)

	// Digest computes the message digest
	Digest(message *i18n.Message) string

	// CreateNameMapper creates a name mapper for placeholder names
	// Returning nil means that no name mapping is used
	CreateNameMapper(message *i18n.Message) PlaceholderMapper
}

// PlaceholderMapper converts placeholder names from internal to serialized representation and back
type PlaceholderMapper interface {
	// ToPublicName converts an internal name to a public name
	ToPublicName(internalName string) *string

	// ToInternalName converts a public name to an internal name
	ToInternalName(publicName string) *string
}

// SimplePlaceholderMapper is a simple mapper that takes a function to transform names
type SimplePlaceholderMapper struct {
	internalToPublic map[string]string
	publicToNextID   map[string]int
	publicToInternal map[string]string
	mapName          func(name string) string
}

// NewSimplePlaceholderMapper creates a new SimplePlaceholderMapper
func NewSimplePlaceholderMapper(message *i18n.Message, mapName func(name string) string) *SimplePlaceholderMapper {
	mapper := &SimplePlaceholderMapper{
		internalToPublic: make(map[string]string),
		publicToNextID:   make(map[string]int),
		publicToInternal: make(map[string]string),
		mapName:          mapName,
	}

	// Visit all nodes to build the mapping
	visitor := &PlaceholderNameVisitor{mapper: mapper}
	for _, node := range message.Nodes {
		node.Visit(visitor, nil)
	}

	return mapper
}

// ToPublicName converts an internal name to a public name
func (m *SimplePlaceholderMapper) ToPublicName(internalName string) *string {
	if publicName, ok := m.internalToPublic[internalName]; ok {
		return &publicName
	}
	return nil
}

// ToInternalName converts a public name to an internal name
func (m *SimplePlaceholderMapper) ToInternalName(publicName string) *string {
	if internalName, ok := m.publicToInternal[publicName]; ok {
		return &internalName
	}
	return nil
}

// visitPlaceholderName visits a placeholder name and creates the mapping
func (m *SimplePlaceholderMapper) visitPlaceholderName(internalName string) {
	if internalName == "" {
		return
	}

	if _, exists := m.internalToPublic[internalName]; exists {
		return
	}

	publicName := m.mapName(internalName)

	if _, exists := m.publicToInternal[publicName]; exists {
		// Create a new name when it has already been used
		nextID := m.publicToNextID[publicName]
		m.publicToNextID[publicName] = nextID + 1
		publicName = publicName + "_" + fmt.Sprintf("%d", nextID)
	} else {
		m.publicToNextID[publicName] = 1
	}

	m.internalToPublic[internalName] = publicName
	m.publicToInternal[publicName] = internalName
}

// PlaceholderNameVisitor visits nodes to extract placeholder names
type PlaceholderNameVisitor struct {
	mapper *SimplePlaceholderMapper
}

// VisitText visits a Text node
func (v *PlaceholderNameVisitor) VisitText(text *i18n.Text, context interface{}) interface{} {
	return nil
}

// VisitContainer visits a Container node
func (v *PlaceholderNameVisitor) VisitContainer(container *i18n.Container, context interface{}) interface{} {
	for _, child := range container.Children {
		child.Visit(v, context)
	}
	return nil
}

// VisitIcu visits an Icu node
func (v *PlaceholderNameVisitor) VisitIcu(icu *i18n.Icu, context interface{}) interface{} {
	for _, node := range icu.Cases {
		node.Visit(v, context)
	}
	return nil
}

// VisitTagPlaceholder visits a TagPlaceholder node
func (v *PlaceholderNameVisitor) VisitTagPlaceholder(ph *i18n.TagPlaceholder, context interface{}) interface{} {
	v.mapper.visitPlaceholderName(ph.StartName)
	for _, child := range ph.Children {
		child.Visit(v, context)
	}
	v.mapper.visitPlaceholderName(ph.CloseName)
	return nil
}

// VisitPlaceholder visits a Placeholder node
func (v *PlaceholderNameVisitor) VisitPlaceholder(ph *i18n.Placeholder, context interface{}) interface{} {
	v.mapper.visitPlaceholderName(ph.Name)
	return nil
}

// VisitIcuPlaceholder visits an IcuPlaceholder node
func (v *PlaceholderNameVisitor) VisitIcuPlaceholder(ph *i18n.IcuPlaceholder, context interface{}) interface{} {
	v.mapper.visitPlaceholderName(ph.Name)
	return nil
}

// VisitBlockPlaceholder visits a BlockPlaceholder node
func (v *PlaceholderNameVisitor) VisitBlockPlaceholder(ph *i18n.BlockPlaceholder, context interface{}) interface{} {
	v.mapper.visitPlaceholderName(ph.StartName)
	for _, child := range ph.Children {
		child.Visit(v, context)
	}
	v.mapper.visitPlaceholderName(ph.CloseName)
	return nil
}
