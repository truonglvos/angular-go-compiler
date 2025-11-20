package viewi18n

import (
	"strings"

	i18n "ngc-go/packages/compiler/i18n"
	i18n_parser "ngc-go/packages/compiler/i18n/parser"
	"ngc-go/packages/compiler/ml_parser"
	"ngc-go/packages/compiler/output"
	"ngc-go/packages/compiler/schema"
	"ngc-go/packages/compiler/util"
)

// I18nMeta represents i18n metadata
type I18nMeta struct {
	ID          *string
	CustomID    *string
	LegacyIDs   []string
	Description *string
	Meaning     *string
}

// I18nMetaVisitor walks over HTML parse tree and converts information stored in
// i18n-related attributes ("i18n" and "i18n-*") into i18n meta object that is
// stored with other element's and attribute's information.
type I18nMetaVisitor struct {
	// whether visited nodes contain i18n information
	HasI18nMeta                     bool
	keepI18nAttrs                   bool
	enableI18nLegacyMessageIdFormat bool
	containerBlocks                 map[string]bool
	preserveSignificantWhitespace   bool
	retainEmptyTokens               bool
	errors                          []*util.ParseError
}

// NewI18nMetaVisitor creates a new I18nMetaVisitor
func NewI18nMetaVisitor(
	keepI18nAttrs bool,
	enableI18nLegacyMessageIdFormat bool,
	containerBlocks map[string]bool,
	preserveSignificantWhitespace bool,
) *I18nMetaVisitor {
	return &I18nMetaVisitor{
		keepI18nAttrs:                   keepI18nAttrs,
		enableI18nLegacyMessageIdFormat: enableI18nLegacyMessageIdFormat,
		containerBlocks:                 containerBlocks,
		preserveSignificantWhitespace:   preserveSignificantWhitespace,
		retainEmptyTokens:               !preserveSignificantWhitespace,
		errors:                          make([]*util.ParseError, 0),
	}
}

// Visit implements the Visitor interface
func (v *I18nMetaVisitor) Visit(node ml_parser.Node, context interface{}) interface{} {
	return node.Visit(v, context)
}

// VisitAllWithErrors visits all nodes and returns errors
func (v *I18nMetaVisitor) VisitAllWithErrors(nodes []ml_parser.Node) *ml_parser.ParseTreeResult {
	result := make([]ml_parser.Node, 0, len(nodes))
	for _, node := range nodes {
		visited := node.Visit(v, nil)
		if visitedNode, ok := visited.(ml_parser.Node); ok {
			result = append(result, visitedNode)
		}
	}
	return ml_parser.NewParseTreeResult(result, v.errors)
}

// VisitElement visits an element node
func (v *I18nMetaVisitor) VisitElement(element *ml_parser.Element, context interface{}) interface{} {
	v.visitElementLike(element)
	return element
}

// VisitComponent visits a component node
func (v *I18nMetaVisitor) VisitComponent(component *ml_parser.Component, context interface{}) interface{} {
	v.visitElementLike(component)
	return component
}

// VisitExpansion visits an expansion node
func (v *I18nMetaVisitor) VisitExpansion(expansion *ml_parser.Expansion, context interface{}) interface{} {
	var message *i18n.Message
	meta := expansion.I18n()
	v.HasI18nMeta = true
	if icuPlaceholder, ok := meta.(*i18n.IcuPlaceholder); ok {
		// set ICU placeholder name (e.g. "ICU_1"),
		// generated while processing root element contents,
		// so we can reference it when we output translation
		name := icuPlaceholder.Name
		message = v.generateI18nMessage([]ml_parser.Node{expansion}, meta, nil)
		icu := IcuFromI18nMessage(message)
		if icu != nil {
			icu.Name = name
			if currentMessage, ok := context.(*i18n.Message); ok && currentMessage != nil {
				// Also update the placeholderToMessage map with this new message
				if currentMessage.PlaceholderToMessage == nil {
					currentMessage.PlaceholderToMessage = make(map[string]*i18n.Message)
				}
				currentMessage.PlaceholderToMessage[name] = message
			}
		}
	} else {
		// ICU is a top level message, try to use metadata from container element if provided via
		// `context` argument. Note: context may not be available for standalone ICUs (without
		// wrapping element), so fallback to ICU metadata in this case.
		var currentMessage *i18n.Message
		if msg, ok := context.(*i18n.Message); ok {
			currentMessage = msg
		}
		if currentMessage == nil {
			var metaStr string
			if str, ok := meta.(string); ok {
				metaStr = str
			}
			message = v.generateI18nMessage([]ml_parser.Node{expansion}, metaStr, nil)
		} else {
			message = v.generateI18nMessage([]ml_parser.Node{expansion}, currentMessage, nil)
		}
	}
	if expansion.NodeWithI18n != nil {
		expansion.NodeWithI18n.SetI18n(message)
	}
	return expansion
}

// VisitText visits a text node
func (v *I18nMetaVisitor) VisitText(text *ml_parser.Text, context interface{}) interface{} {
	return text
}

// VisitAttribute visits an attribute node
func (v *I18nMetaVisitor) VisitAttribute(attribute *ml_parser.Attribute, context interface{}) interface{} {
	return attribute
}

// VisitComment visits a comment node
func (v *I18nMetaVisitor) VisitComment(comment *ml_parser.Comment, context interface{}) interface{} {
	return comment
}

// VisitExpansionCase visits an expansion case node
func (v *I18nMetaVisitor) VisitExpansionCase(expansionCase *ml_parser.ExpansionCase, context interface{}) interface{} {
	return expansionCase
}

// VisitBlock visits a block node
func (v *I18nMetaVisitor) VisitBlock(block *ml_parser.Block, context interface{}) interface{} {
	ml_parser.VisitAll(v, block.Children, context)
	return block
}

// VisitBlockParameter visits a block parameter node
func (v *I18nMetaVisitor) VisitBlockParameter(parameter *ml_parser.BlockParameter, context interface{}) interface{} {
	return parameter
}

// VisitLetDeclaration visits a let declaration node
func (v *I18nMetaVisitor) VisitLetDeclaration(decl *ml_parser.LetDeclaration, context interface{}) interface{} {
	return decl
}

// VisitDirective visits a directive node
func (v *I18nMetaVisitor) VisitDirective(directive *ml_parser.Directive, context interface{}) interface{} {
	return directive
}

// visitElementLike visits an element-like node (Element or Component)
func (v *I18nMetaVisitor) visitElementLike(node ml_parser.Node) {
	var message *i18n.Message

	var element *ml_parser.Element
	var component *ml_parser.Component
	var tagName string
	var attrs []*ml_parser.Attribute

	switch n := node.(type) {
	case *ml_parser.Element:
		element = n
		tagName = n.Name
		attrs = n.Attrs
	case *ml_parser.Component:
		component = n
		if n.TagName != nil {
			tagName = *n.TagName
		}
		attrs = n.Attrs
	default:
		return
	}

	if HasI18nAttrs(node) {
		v.HasI18nMeta = true
		nonI18nAttrs := make([]*ml_parser.Attribute, 0)
		attrsMeta := make(map[string]string)

		for _, attr := range attrs {
			if attr.Name == I18N_ATTR {
				// root 'i18n' node attribute
				var i18nValue interface{}
				if element != nil {
					if element.I18n() != nil {
						i18nValue = element.I18n()
					} else {
						i18nValue = attr.Value
					}
				} else if component != nil {
					if component.I18n() != nil {
						i18nValue = component.I18n()
					} else {
						i18nValue = attr.Value
					}
				}

				// Generate a new AST with whitespace trimmed, but also generate a map
				// to correlate each new node to its original so we can apply i18n
				// information to the original node based on the trimmed content.
				originalNodeMap := make(map[ml_parser.Node]ml_parser.Node)
				var trimmedNodes []ml_parser.Node
				if v.preserveSignificantWhitespace {
					if element != nil {
						trimmedNodes = element.Children
					} else if component != nil {
						trimmedNodes = component.Children
					}
				} else {
					var children []ml_parser.Node
					if element != nil {
						children = element.Children
					} else if component != nil {
						children = component.Children
					}
					trimmedNodesResults := ml_parser.VisitAllWithSiblings(
						ml_parser.NewWhitespaceVisitor(false, originalNodeMap, true),
						children,
						nil,
					)
					// Convert []interface{} to []ml_parser.Node
					trimmedNodes = make([]ml_parser.Node, 0, len(trimmedNodesResults))
					for _, tn := range trimmedNodesResults {
						if node, ok := tn.(ml_parser.Node); ok {
							trimmedNodes = append(trimmedNodes, node)
						}
					}
				}
				message = v.generateI18nMessage(trimmedNodes, i18nValue, setI18nRefs(originalNodeMap))
				if message != nil && len(message.Nodes) == 0 {
					// Ignore the message if it is empty.
					message = nil
				}
				// Store the message on the element
				if element != nil && element.NodeWithI18n != nil {
					element.NodeWithI18n.SetI18n(message)
				} else if component != nil && component.NodeWithI18n != nil {
					component.NodeWithI18n.SetI18n(message)
				}
			} else if strings.HasPrefix(attr.Name, I18N_ATTR_PREFIX) {
				// 'i18n-*' attributes
				name := attr.Name[len(I18N_ATTR_PREFIX):]
				var isTrustedType bool
				if component != nil {
					if component.TagName == nil {
						isTrustedType = false
					} else {
						isTrustedType = schema.IsTrustedTypesSink(*component.TagName, name)
					}
				} else {
					isTrustedType = schema.IsTrustedTypesSink(tagName, name)
				}

				if isTrustedType {
					v.reportError(node, "Translating attribute '"+name+"' is disallowed for security reasons.")
				} else {
					attrsMeta[name] = attr.Value
				}
			} else {
				// non-i18n attributes
				nonI18nAttrs = append(nonI18nAttrs, attr)
			}
		}

		// set i18n meta for attributes
		if len(attrsMeta) > 0 {
			for _, attr := range nonI18nAttrs {
				meta, hasMeta := attrsMeta[attr.Name]
				// do not create translation for empty attributes
				if hasMeta && attr.Value != "" {
					var i18nValue interface{}
					if attr.I18n() != nil {
						i18nValue = attr.I18n()
					} else {
						i18nValue = meta
					}
					if attr.NodeWithI18n != nil {
						attr.NodeWithI18n.SetI18n(v.generateI18nMessage([]ml_parser.Node{attr}, i18nValue, nil))
					}
				}
			}
		}

		if !v.keepI18nAttrs {
			// update element's attributes,
			// keeping only non-i18n related ones
			if element != nil {
				element.Attrs = nonI18nAttrs
			} else if component != nil {
				component.Attrs = nonI18nAttrs
			}
		}
	}
	var children []ml_parser.Node
	if element != nil {
		children = element.Children
	} else if component != nil {
		children = component.Children
	}
	ml_parser.VisitAll(v, children, message)
}

// generateI18nMessage generates an i18n message from nodes
func (v *I18nMetaVisitor) generateI18nMessage(
	nodes []ml_parser.Node,
	meta interface{},
	visitNodeFn i18n_parser.VisitNodeFn,
) *i18n.Message {
	parsedMeta := v.parseMetadata(meta)
	createI18nMessage := i18n_parser.CreateI18nMessageFactory(
		v.containerBlocks,
		v.retainEmptyTokens,
		v.preserveSignificantWhitespace,
	)
	message := createI18nMessage(nodes, parsedMeta.Meaning, parsedMeta.Description, parsedMeta.CustomID, visitNodeFn)
	v.setMessageId(message, meta)
	v.setLegacyIds(message, meta)
	return message
}

// parseMetadata parses the general form `meta` passed into extract the explicit metadata needed to create a `Message`.
func (v *I18nMetaVisitor) parseMetadata(meta interface{}) I18nMeta {
	switch m := meta.(type) {
	case string:
		return ParseI18nMeta(m)
	case *i18n.Message:
		// Convert Message to I18nMeta - Message fields are strings, I18nMeta fields are *string
		return I18nMeta{
			ID:          stringPtr(m.ID),
			CustomID:    stringPtr(m.CustomID),
			LegacyIDs:   m.LegacyIDs,
			Description: stringPtr(m.Description),
			Meaning:     stringPtr(m.Meaning),
		}
	default:
		return I18nMeta{}
	}
}

// stringPtr returns a pointer to the given string
func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// setMessageId generates (or restores) message id if not specified already.
func (v *I18nMetaVisitor) setMessageId(message *i18n.Message, meta interface{}) {
	if message.ID == "" {
		var id string
		if msg, ok := meta.(*i18n.Message); ok && msg.ID != "" {
			id = msg.ID
		} else {
			id = i18n.DecimalDigest(message)
		}
		message.ID = id
	}
}

// setLegacyIds updates the `message` with a `legacyId` if necessary.
func (v *I18nMetaVisitor) setLegacyIds(message *i18n.Message, meta interface{}) {
	if v.enableI18nLegacyMessageIdFormat {
		digest := i18n.ComputeDigest(message)
		decimalDigest := i18n.ComputeDecimalDigest(message)
		message.LegacyIDs = []string{digest, decimalDigest}
	} else {
		// Check if meta is not a string - only process if it's a Message or IcuPlaceholder
		if _, isString := meta.(string); !isString {
			var previousMessage *i18n.Message
			switch m := meta.(type) {
			case *i18n.Message:
				previousMessage = m
			case *i18n.IcuPlaceholder:
				previousMessage = m.PreviousMessage
			}
			if previousMessage != nil {
				message.LegacyIDs = previousMessage.LegacyIDs
			} else {
				message.LegacyIDs = []string{}
			}
		}
	}
}

// reportError reports an error
func (v *I18nMetaVisitor) reportError(node ml_parser.Node, msg string) {
	v.errors = append(v.errors, util.NewParseError(node.SourceSpan(), msg))
}

// setI18nRefs creates a VisitNodeFn that sets i18n references
func setI18nRefs(originalNodeMap map[ml_parser.Node]ml_parser.Node) i18n_parser.VisitNodeFn {
	return func(trimmedNode ml_parser.Node, i18nNode i18n.Node) i18n.Node {
		// We need to set i18n properties on the original, untrimmed AST nodes. The i18n nodes needs to
		// use the trimmed content for message IDs to make messages more stable to whitespace changes.
		// But we don't want to actually trim the content, so we can't use the trimmed HTML AST for
		// general code gen. Instead we map the trimmed HTML AST back to the original AST and then
		// attach the i18n nodes so we get trimmed i18n nodes on the original (untrimmed) HTML AST.
		originalNode, ok := originalNodeMap[trimmedNode]
		if !ok {
			originalNode = trimmedNode
		}

		// Check if originalNode has NodeWithI18n embedded
		if element, ok := originalNode.(*ml_parser.Element); ok && element.NodeWithI18n != nil {
			if icuPlaceholder, ok := i18nNode.(*i18n.IcuPlaceholder); ok {
				if msg, ok := element.NodeWithI18n.I18n().(*i18n.Message); ok {
					// This html node represents an ICU but this is a second processing pass, and the legacy id
					// was computed in the previous pass and stored in the `i18n` property as a message.
					// We are about to wipe out that property so capture the previous message to be reused when
					// generating the message for this ICU later. See `_generateI18nMessage()`.
					icuPlaceholder.PreviousMessage = msg
				}
			}
			element.NodeWithI18n.SetI18n(i18nNode)
		} else if component, ok := originalNode.(*ml_parser.Component); ok && component.NodeWithI18n != nil {
			if icuPlaceholder, ok := i18nNode.(*i18n.IcuPlaceholder); ok {
				if msg, ok := component.NodeWithI18n.I18n().(*i18n.Message); ok {
					icuPlaceholder.PreviousMessage = msg
				}
			}
			component.NodeWithI18n.SetI18n(i18nNode)
		} else if attr, ok := originalNode.(*ml_parser.Attribute); ok && attr.NodeWithI18n != nil {
			if icuPlaceholder, ok := i18nNode.(*i18n.IcuPlaceholder); ok {
				if msg, ok := attr.NodeWithI18n.I18n().(*i18n.Message); ok {
					icuPlaceholder.PreviousMessage = msg
				}
			}
			attr.NodeWithI18n.SetI18n(i18nNode)
		} else if text, ok := originalNode.(*ml_parser.Text); ok && text.NodeWithI18n != nil {
			if icuPlaceholder, ok := i18nNode.(*i18n.IcuPlaceholder); ok {
				if msg, ok := text.NodeWithI18n.I18n().(*i18n.Message); ok {
					icuPlaceholder.PreviousMessage = msg
				}
			}
			text.NodeWithI18n.SetI18n(i18nNode)
		} else if expansion, ok := originalNode.(*ml_parser.Expansion); ok && expansion.NodeWithI18n != nil {
			if icuPlaceholder, ok := i18nNode.(*i18n.IcuPlaceholder); ok {
				if msg, ok := expansion.NodeWithI18n.I18n().(*i18n.Message); ok {
					icuPlaceholder.PreviousMessage = msg
				}
			}
			expansion.NodeWithI18n.SetI18n(i18nNode)
		}
		return i18nNode
	}
}

// I18n separators for metadata
const I18N_MEANING_SEPARATOR = "|"
const I18N_ID_SEPARATOR = "@@"

// ParseI18nMeta parses i18n metas like:
//   - "@@id",
//   - "description[@@id]",
//   - "meaning|description[@@id]"
//
// and returns an object with parsed output.
func ParseI18nMeta(meta string) I18nMeta {
	var customID *string
	var meaning *string
	var description *string

	meta = strings.TrimSpace(meta)
	if meta != "" {
		idIndex := strings.Index(meta, I18N_ID_SEPARATOR)
		descIndex := strings.Index(meta, I18N_MEANING_SEPARATOR)
		var meaningAndDesc string
		if idIndex > -1 {
			meaningAndDesc = meta[:idIndex]
			customIDStr := meta[idIndex+len(I18N_ID_SEPARATOR):]
			customID = &customIDStr
		} else {
			meaningAndDesc = meta
		}
		if descIndex > -1 {
			meaningStr := meaningAndDesc[:descIndex]
			descStr := meaningAndDesc[descIndex+1:]
			meaning = &meaningStr
			description = &descStr
		} else {
			descStr := meaningAndDesc
			description = &descStr
		}
	}

	return I18nMeta{
		CustomID:    customID,
		Meaning:     meaning,
		Description: description,
	}
}

// I18nMetaToJSDoc converts i18n meta information for a message (id, description, meaning)
// to a JsDoc statement formatted as expected by the Closure compiler.
func I18nMetaToJSDoc(meta I18nMeta) *output.JSDocComment {
	tags := make([]output.JSDocTag, 0)
	if meta.Description != nil {
		descTag := "desc"
		descText := *meta.Description
		tags = append(tags, output.JSDocTag{
			TagName: &descTag,
			Text:    &descText,
		})
	} else {
		// Suppress the JSCompiler warning that a `@desc` was not given for this message.
		suppressTag := "suppress"
		suppressText := "{msgDescriptions}"
		tags = append(tags, output.JSDocTag{
			TagName: &suppressTag,
			Text:    &suppressText,
		})
	}
	if meta.Meaning != nil {
		meaningTag := "meaning"
		meaningText := *meta.Meaning
		tags = append(tags, output.JSDocTag{
			TagName: &meaningTag,
			Text:    &meaningText,
		})
	}
	return output.NewJSDocComment(tags)
}
