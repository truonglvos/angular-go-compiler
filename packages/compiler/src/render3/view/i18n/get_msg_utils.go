package viewi18n

import (
	"strings"

	i18n "ngc-go/packages/compiler/src/i18n"
	"ngc-go/packages/compiler/src/output"
)

// Closure uses `goog.getMsg(message)` to lookup translations
const GOOG_GET_MSG = "goog.getMsg"

// CreateGoogleGetMsgStatements generates a `goog.getMsg()` statement and reassignment.
func CreateGoogleGetMsgStatements(
	variable *output.ReadVarExpr,
	message *i18n.Message,
	closureVar *output.ReadVarExpr,
	placeholderValues map[string]output.OutputExpression,
) []output.OutputStatement {
	messageString := SerializeI18nMessageForGetMsg(message)
	args := []output.OutputExpression{
		output.NewLiteralExpr(messageString, nil, nil),
	}
	if len(placeholderValues) > 0 {
		// Message template parameters containing the magic strings replaced by the Angular runtime with
		// real data, e.g. `{'interpolation': '\uFFFD0\uFFFD'}`.
		formattedParams := FormatI18nPlaceholderNamesInMap(placeholderValues, true /* useCamelCase */)
		entries := make([]*output.LiteralMapEntry, 0, len(formattedParams))
		for key, value := range formattedParams {
			entries = append(entries, output.NewLiteralMapEntry(key, value, true /* quoted */))
		}
		args = append(args, output.NewLiteralMapExpr(entries, nil, nil))

		// Message options object, which contains original source code for placeholders (as they are
		// present in a template, e.g.
		// `{original_code: {'interpolation': '{{ name }}', 'startTagSpan': '<span>'}}`.
		originalCodeEntries := make([]*output.LiteralMapEntry, 0, len(placeholderValues))
		for param := range placeholderValues {
			var value output.OutputExpression
			if placeholder, ok := message.Placeholders[param]; ok {
				// Get source span for typical placeholder if it exists.
				value = output.NewLiteralExpr(placeholder.SourceSpan.String(), nil, nil)
			} else {
				// Otherwise must be an ICU expression, get it's source span.
				if msg, ok := message.PlaceholderToMessage[param]; ok {
					parts := make([]string, 0, len(msg.Nodes))
					for _, node := range msg.Nodes {
						parts = append(parts, node.SourceSpan().String())
					}
					value = output.NewLiteralExpr(strings.Join(parts, ""), nil, nil)
				} else {
					value = output.NewLiteralExpr("", nil, nil)
				}
			}
			formattedName := FormatI18nPlaceholderName(param, false)
			originalCodeEntries = append(originalCodeEntries, output.NewLiteralMapEntry(formattedName, value, true /* quoted */))
		}
		originalCodeMap := output.NewLiteralMapExpr(originalCodeEntries, nil, nil)
		optionsEntries := []*output.LiteralMapEntry{
			output.NewLiteralMapEntry("original_code", originalCodeMap, false /* quoted */),
		}
		args = append(args, output.NewLiteralMapExpr(optionsEntries, nil, nil))
	}

	// /**
	//  * @desc description of message
	//  * @meaning meaning of message
	//  */
	// const MSG_... = goog.getMsg(..);
	// I18N_X = MSG_...;
	// Convert message to I18nMeta - in TypeScript, message is passed directly to i18nMetaToJSDoc
	// because Message has fields compatible with I18nMeta
	meta := I18nMeta{
		ID:          &message.ID,
		CustomID:    &message.CustomID,
		LegacyIDs:   message.LegacyIDs,
		Description: &message.Description,
		Meaning:     &message.Meaning,
	}
	googGetMsgStmt := output.NewDeclareVarStmt(
		closureVar.Name,
		output.NewInvokeFunctionExpr(
			output.NewReadVarExpr(GOOG_GET_MSG, nil, nil),
			args,
			nil,   /* typ */
			nil,   /* sourceSpan */
			false, /* pure */
		),
		output.InferredType,
		output.StmtModifierFinal,
		nil, /* sourceSpan */
		[]*output.LeadingComment{
			{
				Text:            I18nMetaToJSDoc(meta).String(),
				Multiline:       true,
				TrailingNewline: true,
			},
		},
	)
	i18nAssignmentStmt := output.NewExpressionStatement(
		variable.Set(closureVar),
		nil, /* leadingComments */
		nil, /* sourceSpan */
	)
	return []output.OutputStatement{googGetMsgStmt, i18nAssignmentStmt}
}

// GetMsgSerializerVisitor is a visitor that walks over i18n tree and generates its string representation,
// including ICUs and placeholders in `{$placeholder}` (for plain messages) or `{PLACEHOLDER}` (inside ICUs) format.
type GetMsgSerializerVisitor struct{}

// NewGetMsgSerializerVisitor creates a new GetMsgSerializerVisitor
func NewGetMsgSerializerVisitor() *GetMsgSerializerVisitor {
	return &GetMsgSerializerVisitor{}
}

// formatPh formats a placeholder value
func (v *GetMsgSerializerVisitor) formatPh(value string) string {
	return "{$" + FormatI18nPlaceholderName(value, true) + "}"
}

// VisitText visits a Text node
func (v *GetMsgSerializerVisitor) VisitText(text *i18n.Text, context interface{}) interface{} {
	return text.Value
}

// VisitContainer visits a Container node
func (v *GetMsgSerializerVisitor) VisitContainer(container *i18n.Container, context interface{}) interface{} {
	parts := make([]string, 0, len(container.Children))
	for _, child := range container.Children {
		result := child.Visit(v, context)
		if str, ok := result.(string); ok {
			parts = append(parts, str)
		}
	}
	return strings.Join(parts, "")
}

// VisitIcu visits an Icu node
func (v *GetMsgSerializerVisitor) VisitIcu(icu *i18n.Icu, context interface{}) interface{} {
	return SerializeIcuNode(icu)
}

// VisitTagPlaceholder visits a TagPlaceholder node
func (v *GetMsgSerializerVisitor) VisitTagPlaceholder(ph *i18n.TagPlaceholder, context interface{}) interface{} {
	if ph.IsVoid {
		return v.formatPh(ph.StartName)
	}
	parts := make([]string, 0, len(ph.Children)+2)
	parts = append(parts, v.formatPh(ph.StartName))
	for _, child := range ph.Children {
		result := child.Visit(v, context)
		if str, ok := result.(string); ok {
			parts = append(parts, str)
		}
	}
	parts = append(parts, v.formatPh(ph.CloseName))
	return strings.Join(parts, "")
}

// VisitPlaceholder visits a Placeholder node
func (v *GetMsgSerializerVisitor) VisitPlaceholder(ph *i18n.Placeholder, context interface{}) interface{} {
	return v.formatPh(ph.Name)
}

// VisitBlockPlaceholder visits a BlockPlaceholder node
func (v *GetMsgSerializerVisitor) VisitBlockPlaceholder(ph *i18n.BlockPlaceholder, context interface{}) interface{} {
	parts := make([]string, 0, len(ph.Children)+2)
	parts = append(parts, v.formatPh(ph.StartName))
	for _, child := range ph.Children {
		result := child.Visit(v, context)
		if str, ok := result.(string); ok {
			parts = append(parts, str)
		}
	}
	parts = append(parts, v.formatPh(ph.CloseName))
	return strings.Join(parts, "")
}

// VisitIcuPlaceholder visits an IcuPlaceholder node
func (v *GetMsgSerializerVisitor) VisitIcuPlaceholder(ph *i18n.IcuPlaceholder, context interface{}) interface{} {
	return v.formatPh(ph.Name)
}

var serializerVisitor = NewGetMsgSerializerVisitor()

// SerializeI18nMessageForGetMsg serializes an i18n message for goog.getMsg
func SerializeI18nMessageForGetMsg(message *i18n.Message) string {
	parts := make([]string, 0, len(message.Nodes))
	for _, node := range message.Nodes {
		result := node.Visit(serializerVisitor, nil)
		if str, ok := result.(string); ok {
			parts = append(parts, str)
		}
	}
	return strings.Join(parts, "")
}
