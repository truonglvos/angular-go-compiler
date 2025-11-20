package viewi18n

import (
	i18n "ngc-go/packages/compiler/i18n"
	"ngc-go/packages/compiler/output"
	"ngc-go/packages/compiler/util"
)

// CreateLocalizeStatements creates statements for localizing a message
func CreateLocalizeStatements(
	variable *output.ReadVarExpr,
	message *i18n.Message,
	params map[string]output.OutputExpression,
) []output.OutputStatement {
	messageParts, placeHolders := SerializeI18nMessageForLocalize(message)
	sourceSpan := GetSourceSpan(message)
	expressions := make([]output.OutputExpression, 0, len(placeHolders))
	for _, ph := range placeHolders {
		if expr, ok := params[ph.GetText()]; ok {
			expressions = append(expressions, expr)
		}
	}

	// Convert i18n.Message to output.I18nMeta
	// In TypeScript, o.localizedString takes message directly, but in Go we need to convert
	metaBlock := &output.I18nMeta{
		ID:          &message.ID,
		CustomID:    &message.CustomID,
		LegacyIDs:   message.LegacyIDs,
		Description: &message.Description,
		Meaning:     &message.Meaning,
	}

	localizedString := output.NewLocalizedString(
		metaBlock,
		messageParts,
		placeHolders,
		expressions,
		sourceSpan,
	)
	variableInitialization := variable.Set(localizedString)
	return []output.OutputStatement{
		output.NewExpressionStatement(variableInitialization, nil, nil),
	}
}

// LocalizeSerializerVisitor is a visitor that walks over an i18n tree, capturing literal strings and placeholders
// The result can be used for generating the `$localize` tagged template literals.
type LocalizeSerializerVisitor struct {
	placeholderToMessage map[string]*i18n.Message
	pieces               []output.MessagePiece
}

// NewLocalizeSerializerVisitor creates a new LocalizeSerializerVisitor
func NewLocalizeSerializerVisitor(
	placeholderToMessage map[string]*i18n.Message,
	pieces []output.MessagePiece,
) *LocalizeSerializerVisitor {
	return &LocalizeSerializerVisitor{
		placeholderToMessage: placeholderToMessage,
		pieces:               pieces,
	}
}

// VisitText visits a Text node
func (v *LocalizeSerializerVisitor) VisitText(text *i18n.Text, context interface{}) interface{} {
	if len(v.pieces) > 0 {
		if lastPiece, ok := v.pieces[len(v.pieces)-1].(*output.LiteralPiece); ok {
			// Two literal pieces in a row means that there was some comment node in-between.
			lastPiece.Text += text.Value
			return nil
		}
	}
	sourceSpan := util.NewParseSourceSpan(
		text.SourceSpan().FullStart,
		text.SourceSpan().End,
		text.SourceSpan().FullStart,
		text.SourceSpan().Details,
	)
	v.pieces = append(v.pieces, output.NewLiteralPiece(text.Value, sourceSpan))
	return nil
}

// VisitContainer visits a Container node
func (v *LocalizeSerializerVisitor) VisitContainer(container *i18n.Container, context interface{}) interface{} {
	for _, child := range container.Children {
		child.Visit(v, context)
	}
	return nil
}

// VisitIcu visits an Icu node
func (v *LocalizeSerializerVisitor) VisitIcu(icu *i18n.Icu, context interface{}) interface{} {
	v.pieces = append(v.pieces, output.NewLiteralPiece(SerializeIcuNode(icu), icu.SourceSpan()))
	return nil
}

// VisitTagPlaceholder visits a TagPlaceholder node
func (v *LocalizeSerializerVisitor) VisitTagPlaceholder(ph *i18n.TagPlaceholder, context interface{}) interface{} {
	startSourceSpan := ph.StartSourceSpan
	if startSourceSpan == nil {
		startSourceSpan = ph.SourceSpan()
	}
	v.pieces = append(v.pieces, v.createPlaceholderPiece(ph.StartName, startSourceSpan, nil))
	if !ph.IsVoid {
		for _, child := range ph.Children {
			child.Visit(v, context)
		}
		closeSourceSpan := ph.EndSourceSpan
		if closeSourceSpan == nil {
			closeSourceSpan = ph.SourceSpan()
		}
		v.pieces = append(v.pieces, v.createPlaceholderPiece(ph.CloseName, closeSourceSpan, nil))
	}
	return nil
}

// VisitPlaceholder visits a Placeholder node
func (v *LocalizeSerializerVisitor) VisitPlaceholder(ph *i18n.Placeholder, context interface{}) interface{} {
	v.pieces = append(v.pieces, v.createPlaceholderPiece(ph.Name, ph.SourceSpan(), nil))
	return nil
}

// VisitBlockPlaceholder visits a BlockPlaceholder node
func (v *LocalizeSerializerVisitor) VisitBlockPlaceholder(ph *i18n.BlockPlaceholder, context interface{}) interface{} {
	startSourceSpan := ph.StartSourceSpan
	if startSourceSpan == nil {
		startSourceSpan = ph.SourceSpan()
	}
	v.pieces = append(v.pieces, v.createPlaceholderPiece(ph.StartName, startSourceSpan, nil))
	for _, child := range ph.Children {
		child.Visit(v, context)
	}
	closeSourceSpan := ph.EndSourceSpan
	if closeSourceSpan == nil {
		closeSourceSpan = ph.SourceSpan()
	}
	v.pieces = append(v.pieces, v.createPlaceholderPiece(ph.CloseName, closeSourceSpan, nil))
	return nil
}

// VisitIcuPlaceholder visits an IcuPlaceholder node
func (v *LocalizeSerializerVisitor) VisitIcuPlaceholder(ph *i18n.IcuPlaceholder, context interface{}) interface{} {
	var associatedMessage *i18n.Message
	if msg, ok := v.placeholderToMessage[ph.Name]; ok {
		associatedMessage = msg
	}
	v.pieces = append(v.pieces, v.createPlaceholderPiece(ph.Name, ph.SourceSpan(), associatedMessage))
	return nil
}

// createPlaceholderPiece creates a placeholder piece
func (v *LocalizeSerializerVisitor) createPlaceholderPiece(
	name string,
	sourceSpan *util.ParseSourceSpan,
	associatedMessage *i18n.Message,
) *output.PlaceholderPiece {
	var msg interface{}
	if associatedMessage != nil {
		msg = associatedMessage
	}
	return output.NewPlaceholderPiece(
		FormatI18nPlaceholderName(name, false),
		sourceSpan,
		msg,
	)
}

// SerializeI18nMessageForLocalize serializes an i18n message into two arrays: messageParts and placeholders.
// These arrays will be used to generate `$localize` tagged template literals.
func SerializeI18nMessageForLocalize(message *i18n.Message) (
	[]*output.LiteralPiece,
	[]*output.PlaceholderPiece,
) {
	pieces := make([]output.MessagePiece, 0)
	serializerVisitor := NewLocalizeSerializerVisitor(message.PlaceholderToMessage, pieces)
	for _, node := range message.Nodes {
		node.Visit(serializerVisitor, nil)
	}
	return ProcessMessagePieces(serializerVisitor.pieces)
}

// GetSourceSpan gets the source span for a message
func GetSourceSpan(message *i18n.Message) *util.ParseSourceSpan {
	if len(message.Nodes) == 0 {
		return nil
	}
	startNode := message.Nodes[0]
	endNode := message.Nodes[len(message.Nodes)-1]
	return util.NewParseSourceSpan(
		startNode.SourceSpan().FullStart,
		endNode.SourceSpan().End,
		startNode.SourceSpan().FullStart,
		startNode.SourceSpan().Details,
	)
}

// ProcessMessagePieces converts the list of serialized MessagePieces into two arrays.
// One contains the literal string pieces and the other the placeholders that will be replaced by
// expressions when rendering `$localize` tagged template literals.
func ProcessMessagePieces(pieces []output.MessagePiece) (
	[]*output.LiteralPiece,
	[]*output.PlaceholderPiece,
) {
	messageParts := make([]*output.LiteralPiece, 0)
	placeHolders := make([]*output.PlaceholderPiece, 0)

	if len(pieces) > 0 {
		if _, ok := pieces[0].(*output.PlaceholderPiece); ok {
			// The first piece was a placeholder so we need to add an initial empty message part.
			firstPh := pieces[0].(*output.PlaceholderPiece)
			messageParts = append(messageParts, createEmptyMessagePart(firstPh.GetSourceSpan().Start))
		}
	}

	for i := 0; i < len(pieces); i++ {
		part := pieces[i]
		if literalPiece, ok := part.(*output.LiteralPiece); ok {
			messageParts = append(messageParts, literalPiece)
		} else if placeholderPiece, ok := part.(*output.PlaceholderPiece); ok {
			placeHolders = append(placeHolders, placeholderPiece)
			if i > 0 {
				if _, ok := pieces[i-1].(*output.PlaceholderPiece); ok {
					// There were two placeholders in a row, so we need to add an empty message part.
					prevPh := pieces[i-1].(*output.PlaceholderPiece)
					messageParts = append(messageParts, createEmptyMessagePart(prevPh.GetSourceSpan().End))
				}
			}
		}
	}

	if len(pieces) > 0 {
		if lastPh, ok := pieces[len(pieces)-1].(*output.PlaceholderPiece); ok {
			// The last piece was a placeholder so we need to add a final empty message part.
			messageParts = append(messageParts, createEmptyMessagePart(lastPh.GetSourceSpan().End))
		}
	}

	return messageParts, placeHolders
}

// createEmptyMessagePart creates an empty message part
func createEmptyMessagePart(location *util.ParseLocation) *output.LiteralPiece {
	return output.NewLiteralPiece("", util.NewParseSourceSpan(location, location, nil, nil))
}
