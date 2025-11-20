package i18n_message_bundle

import (
	"ngc-go/packages/compiler/src/i18n"
	i18n_extractor_merger "ngc-go/packages/compiler/src/i18n/extractor_merger"
	"ngc-go/packages/compiler/src/i18n/serializers"
	"ngc-go/packages/compiler/src/ml_parser"
	"ngc-go/packages/compiler/src/util"
)

// MessageBundle is a container for messages extracted from templates
type MessageBundle struct {
	messages           []*i18n.Message
	htmlParser         ml_parser.HtmlParser
	implicitTags       []string
	implicitAttrs      map[string][]string
	locale             *string
	preserveWhitespace bool
}

// NewMessageBundle creates a new MessageBundle
func NewMessageBundle(
	htmlParser ml_parser.HtmlParser,
	implicitTags []string,
	implicitAttrs map[string][]string,
	locale *string,
	preserveWhitespace bool,
) *MessageBundle {
	return &MessageBundle{
		messages:           []*i18n.Message{},
		htmlParser:         htmlParser,
		implicitTags:       implicitTags,
		implicitAttrs:      implicitAttrs,
		locale:             locale,
		preserveWhitespace: preserveWhitespace,
	}
}

// UpdateFromTemplate updates the bundle from a template source
func (mb *MessageBundle) UpdateFromTemplate(source string, url string) []*util.ParseError {
	tokenizeExpansionForms := true
	options := &ml_parser.TokenizeOptions{
		TokenizeExpansionForms: &tokenizeExpansionForms,
	}
	htmlParserResult := mb.htmlParser.Parse(source, url, options)

	if len(htmlParserResult.Errors) > 0 {
		return htmlParserResult.Errors
	}

	// Trim unnecessary whitespace from extracted messages if requested
	rootNodes := htmlParserResult.RootNodes
	if !mb.preserveWhitespace {
		// TODO: Implement WhitespaceVisitor
		// rootNodes = visitAllWithSiblings(
		//     new WhitespaceVisitor(/* preserveSignificantWhitespace */ false),
		//     htmlParserResult.RootNodes,
		// )
	}

	i18nParserResult := i18n_extractor_merger.ExtractMessages(
		rootNodes,
		mb.implicitTags,
		mb.implicitAttrs,
		mb.preserveWhitespace,
	)

	if len(i18nParserResult.Errors) > 0 {
		return i18nParserResult.Errors
	}

	mb.messages = append(mb.messages, i18nParserResult.Messages...)
	return []*util.ParseError{}
}

// GetMessages returns the messages in the internal format
func (mb *MessageBundle) GetMessages() []*i18n.Message {
	return mb.messages
}

// Write writes the messages using the given serializer
func (mb *MessageBundle) Write(serializer serializers.Serializer, filterSources func(string) string) string {
	messages := make(map[string]*i18n.Message)
	mapperVisitor := NewMapPlaceholderNames()

	// Deduplicate messages based on their ID
	for _, message := range mb.messages {
		id := serializer.Digest(message)
		if _, exists := messages[id]; !exists {
			messages[id] = message
		} else {
			messages[id].Sources = append(messages[id].Sources, message.Sources...)
		}
	}

	// Transform placeholder names using the serializer mapping
	msgList := make([]*i18n.Message, 0, len(messages))
	for id, src := range messages {
		mapper := serializer.CreateNameMapper(src)
		var nodes []i18n.Node
		if mapper != nil {
			nodes = mapperVisitor.Convert(src.Nodes, mapper)
		} else {
			nodes = src.Nodes
		}

		transformedMessage := i18n.NewMessage(
			nodes,
			map[string]i18n.MessagePlaceholder{},
			map[string]*i18n.Message{},
			src.Meaning,
			src.Description,
			id,
		)
		transformedMessage.Sources = src.Sources

		if filterSources != nil {
			for i := range transformedMessage.Sources {
				transformedMessage.Sources[i].FilePath = filterSources(transformedMessage.Sources[i].FilePath)
			}
		}

		msgList = append(msgList, transformedMessage)
	}

	return serializer.Write(msgList, mb.locale)
}

// MapPlaceholderNames transforms an i18n AST by renaming the placeholder nodes with the given mapper
type MapPlaceholderNames struct {
	*i18n.CloneVisitor
}

// NewMapPlaceholderNames creates a new MapPlaceholderNames visitor
func NewMapPlaceholderNames() *MapPlaceholderNames {
	return &MapPlaceholderNames{
		CloneVisitor: i18n.NewCloneVisitor(),
	}
}

// Convert converts nodes using the mapper
func (v *MapPlaceholderNames) Convert(nodes []i18n.Node, mapper serializers.PlaceholderMapper) []i18n.Node {
	if mapper == nil {
		return nodes
	}
	result := make([]i18n.Node, len(nodes))
	for i, n := range nodes {
		visitResult := n.Visit(v, mapper)
		if node, ok := visitResult.(i18n.Node); ok {
			result[i] = node
		} else {
			result[i] = n
		}
	}
	return result
}

// VisitTagPlaceholder visits a TagPlaceholder node
func (v *MapPlaceholderNames) VisitTagPlaceholder(ph *i18n.TagPlaceholder, context interface{}) interface{} {
	mapper := context.(serializers.PlaceholderMapper)
	startName := ph.StartName
	if publicName := mapper.ToPublicName(ph.StartName); publicName != nil {
		startName = *publicName
	}

	closeName := ph.CloseName
	if ph.CloseName != "" {
		if publicName := mapper.ToPublicName(ph.CloseName); publicName != nil {
			closeName = *publicName
		}
	}

	children := make([]i18n.Node, len(ph.Children))
	for i, child := range ph.Children {
		visitResult := child.Visit(v, mapper)
		if node, ok := visitResult.(i18n.Node); ok {
			children[i] = node
		} else {
			children[i] = child
		}
	}

	return i18n.NewTagPlaceholder(
		ph.Tag,
		ph.Attrs,
		startName,
		closeName,
		children,
		ph.IsVoid,
		ph.SourceSpan(),
		ph.StartSourceSpan,
		ph.EndSourceSpan,
	)
}

// VisitBlockPlaceholder visits a BlockPlaceholder node
func (v *MapPlaceholderNames) VisitBlockPlaceholder(ph *i18n.BlockPlaceholder, context interface{}) interface{} {
	mapper := context.(serializers.PlaceholderMapper)
	startName := ph.StartName
	if publicName := mapper.ToPublicName(ph.StartName); publicName != nil {
		startName = *publicName
	}

	closeName := ph.CloseName
	if ph.CloseName != "" {
		if publicName := mapper.ToPublicName(ph.CloseName); publicName != nil {
			closeName = *publicName
		}
	}

	children := make([]i18n.Node, len(ph.Children))
	for i, child := range ph.Children {
		visitResult := child.Visit(v, mapper)
		if node, ok := visitResult.(i18n.Node); ok {
			children[i] = node
		} else {
			children[i] = child
		}
	}

	return i18n.NewBlockPlaceholder(
		ph.Name,
		ph.Parameters,
		startName,
		closeName,
		children,
		ph.SourceSpan(),
		ph.StartSourceSpan,
		ph.EndSourceSpan,
	)
}

// VisitPlaceholder visits a Placeholder node
func (v *MapPlaceholderNames) VisitPlaceholder(ph *i18n.Placeholder, context interface{}) interface{} {
	mapper := context.(serializers.PlaceholderMapper)
	name := ph.Name
	if publicName := mapper.ToPublicName(ph.Name); publicName != nil {
		name = *publicName
	}
	return i18n.NewPlaceholder(ph.Value, name, ph.SourceSpan())
}

// VisitIcuPlaceholder visits an IcuPlaceholder node
func (v *MapPlaceholderNames) VisitIcuPlaceholder(ph *i18n.IcuPlaceholder, context interface{}) interface{} {
	mapper := context.(serializers.PlaceholderMapper)
	name := ph.Name
	if publicName := mapper.ToPublicName(ph.Name); publicName != nil {
		name = *publicName
	}
	return i18n.NewIcuPlaceholder(ph.Value, name, ph.SourceSpan())
}
