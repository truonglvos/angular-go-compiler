package phases

import (
	"fmt"
	"strings"

	"ngc-go/packages/compiler/src/output"
	ir "ngc-go/packages/compiler/src/template/pipeline/ir/src"
	ir_operation "ngc-go/packages/compiler/src/template/pipeline/ir/src/operations"
	ops_create "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/create"

	pipeline "ngc-go/packages/compiler/src/template/pipeline/src/compilation"
)

const (
	// ESCAPE is the escape sequence used indicate message param values.
	ESCAPE = "\uFFFD"
	// ELEMENT_MARKER is the marker used to indicate an element tag.
	ELEMENT_MARKER = "#"
	// TEMPLATE_MARKER is the marker used to indicate a template tag.
	TEMPLATE_MARKER = "*"
	// TAG_CLOSE_MARKER is the marker used to indicate closing of an element or template tag.
	TAG_CLOSE_MARKER = "/"
	// CONTEXT_MARKER is the marker used to indicate the sub-template context.
	CONTEXT_MARKER = ":"
	// LIST_START_MARKER is the marker used to indicate the start of a list of values.
	LIST_START_MARKER = "["
	// LIST_END_MARKER is the marker used to indicate the end of a list of values.
	LIST_END_MARKER = "]"
	// LIST_DELIMITER is the delimiter used to separate multiple values in a list.
	LIST_DELIMITER = "|"
)

// ExtractI18nMessages formats the param maps on extracted message ops into a maps of `Expression` objects that can be
// used in the final output.
func ExtractI18nMessages(job *pipeline.CompilationJob) {
	// Create an i18n message for each context.
	// TODO: Merge the context op with the message op since they're 1:1 anyways.
	i18nMessagesByContext := make(map[ir_operation.XrefId]*ops_create.I18nMessageOp)
	i18nBlocks := make(map[ir_operation.XrefId]*ops_create.I18nStartOp)
	i18nContexts := make(map[ir_operation.XrefId]*ops_create.I18nContextOp)
	for _, unit := range job.GetUnits() {
		for op := unit.GetCreate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			switch op.GetKind() {
			case ir.OpKindI18nContext:
				i18nContextOp, ok := op.(*ops_create.I18nContextOp)
				if !ok {
					continue
				}
				i18nMessageOp := createI18nMessage(job, i18nContextOp, nil)
				unit.GetCreate().Push(i18nMessageOp)
				i18nMessagesByContext[i18nContextOp.Xref] = i18nMessageOp
				i18nContexts[i18nContextOp.Xref] = i18nContextOp
			case ir.OpKindI18nStart:
				i18nStartOp, ok := op.(*ops_create.I18nStartOp)
				if !ok {
					continue
				}
				i18nBlocks[i18nStartOp.Xref] = i18nStartOp
			}
		}
	}

	// Associate sub-messages for ICUs with their root message. At this point we can also remove the
	// ICU start/end ops, as they are no longer needed.
	var currentIcu *ops_create.IcuStartOp = nil
	for _, unit := range job.GetUnits() {
		for op := unit.GetCreate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			switch op.GetKind() {
			case ir.OpKindIcuStart:
				icuStartOp, ok := op.(*ops_create.IcuStartOp)
				if !ok {
					continue
				}
				currentIcu = icuStartOp
				unit.GetCreate().Remove(op)
				// Skip any contexts not associated with an ICU.
				icuContext, exists := i18nContexts[icuStartOp.Context]
				if !exists {
					continue
				}
				if icuContext.ContextKind != ir.I18nContextKindIcu {
					continue
				}
				// Skip ICUs that share a context with their i18n message. These represent root-level
				// ICUs, not sub-messages.
				i18nBlock, exists := i18nBlocks[icuContext.I18nBlock]
				if !exists {
					continue
				}
				if i18nBlock.Context == icuContext.Xref {
					continue
				}
				// Find the root message and push this ICUs message as a sub-message.
				rootI18nBlock, exists := i18nBlocks[i18nBlock.Root]
				if !exists {
					panic("AssertionError: ICU sub-message should belong to a root message.")
				}
				rootMessage, exists := i18nMessagesByContext[rootI18nBlock.Context]
				if !exists {
					panic("AssertionError: ICU sub-message should belong to a root message.")
				}
				subMessage, exists := i18nMessagesByContext[icuContext.Xref]
				if !exists {
					continue
				}
				subMessage.MessagePlaceholder = &icuStartOp.MessagePlaceholder
				rootMessage.SubMessages = append(rootMessage.SubMessages, subMessage.Xref)
			case ir.OpKindIcuEnd:
				currentIcu = nil
				unit.GetCreate().Remove(op)
			case ir.OpKindIcuPlaceholder:
				icuPlaceholderOp, ok := op.(*ops_create.IcuPlaceholderOp)
				if !ok {
					continue
				}
				// Add ICU placeholders to the message, then remove the ICU placeholder ops.
				if currentIcu == nil || currentIcu.Context == 0 {
					panic("AssertionError: Unexpected ICU placeholder outside of i18n context")
				}
				msg, exists := i18nMessagesByContext[currentIcu.Context]
				if !exists {
					continue
				}
				formattedPlaceholder := formatIcuPlaceholder(icuPlaceholderOp)
				msg.PostprocessingParams[icuPlaceholderOp.Name] = output.NewLiteralExpr(formattedPlaceholder, nil, nil)
				unit.GetCreate().Remove(op)
			}
		}
	}
}

// createI18nMessage creates an i18n message op from an i18n context op.
func createI18nMessage(
	job *pipeline.CompilationJob,
	context *ops_create.I18nContextOp,
	messagePlaceholder *string,
) *ops_create.I18nMessageOp {
	formattedParams := formatParams(context.Params)
	formattedPostprocessingParams := formatParams(context.PostprocessingParams)
	needsPostprocessing := false
	for _, values := range context.Params {
		if len(values) > 1 {
			needsPostprocessing = true
			break
		}
	}
	if messagePlaceholder == nil {
		messagePlaceholder = nil
	}
	return ops_create.NewI18nMessageOp(
		job.AllocateXrefId(),
		context.Xref,
		context.I18nBlock,
		context.Message,
		messagePlaceholder,
		formattedParams,
		formattedPostprocessingParams,
		needsPostprocessing,
	)
}

// formatIcuPlaceholder formats an ICU placeholder into a single string with expression placeholders.
func formatIcuPlaceholder(op *ops_create.IcuPlaceholderOp) string {
	if len(op.Strings) != len(op.ExpressionPlaceholders)+1 {
		panic(fmt.Sprintf(
			"AssertionError: Invalid ICU placeholder with %d strings and %d expressions",
			len(op.Strings),
			len(op.ExpressionPlaceholders),
		))
	}
	values := make([]string, len(op.ExpressionPlaceholders))
	for i, placeholder := range op.ExpressionPlaceholders {
		values[i] = formatValue(placeholder)
	}
	result := make([]string, 0, len(op.Strings)*2)
	for i, str := range op.Strings {
		result = append(result, str)
		if i < len(values) {
			result = append(result, values[i])
		}
	}
	return strings.Join(result, "")
}

// formatParams formats a map of `I18nParamValue[]` values into a map of `Expression` values.
func formatParams(params map[string][]ops_create.I18nParamValue) map[string]output.OutputExpression {
	formattedParams := make(map[string]output.OutputExpression)
	for placeholder, placeholderValues := range params {
		serializedValues := formatParamValues(placeholderValues)
		if serializedValues != "" {
			formattedParams[placeholder] = output.NewLiteralExpr(serializedValues, nil, nil)
		}
	}
	return formattedParams
}

// formatParamValues formats an `I18nParamValue[]` into a string (or empty string for empty array).
func formatParamValues(values []ops_create.I18nParamValue) string {
	if len(values) == 0 {
		return ""
	}
	serializedValues := make([]string, len(values))
	for i, value := range values {
		serializedValues[i] = formatValue(value)
	}
	if len(serializedValues) == 1 {
		return serializedValues[0]
	}
	return LIST_START_MARKER + strings.Join(serializedValues, LIST_DELIMITER) + LIST_END_MARKER
}

// formatValue formats a single `I18nParamValue` into a string
func formatValue(value ops_create.I18nParamValue) string {
	// Element tags with a structural directive use a special form that concatenates the element and
	// template values.
	if (value.Flags&ir.I18nParamValueFlagsElementTag != 0) &&
		(value.Flags&ir.I18nParamValueFlagsTemplateTag != 0) {
		compoundValue, ok := value.Value.(struct {
			Element  int
			Template int
		})
		if !ok {
			panic("AssertionError: Expected i18n param value to have an element and template slot")
		}
		elementValue := formatValue(ops_create.I18nParamValue{
			Value:            compoundValue.Element,
			SubTemplateIndex: value.SubTemplateIndex,
			Flags:            value.Flags &^ ir.I18nParamValueFlagsTemplateTag,
		})
		templateValue := formatValue(ops_create.I18nParamValue{
			Value:            compoundValue.Template,
			SubTemplateIndex: value.SubTemplateIndex,
			Flags:            value.Flags &^ ir.I18nParamValueFlagsElementTag,
		})
		// TODO(mmalerba): This is likely a bug in TemplateDefinitionBuilder, we should not need to
		// record the template value twice. For now I'm re-implementing the behavior here to keep the
		// output consistent with TemplateDefinitionBuilder.
		if (value.Flags&ir.I18nParamValueFlagsOpenTag != 0) &&
			(value.Flags&ir.I18nParamValueFlagsCloseTag != 0) {
			return templateValue + elementValue + templateValue
		}
		// To match the TemplateDefinitionBuilder output, flip the order depending on whether the
		// values represent a closing or opening tag (or both).
		// TODO(mmalerba): Figure out if this makes a difference in terms of either functionality,
		// or the resulting message ID. If not, we can remove the special-casing in the future.
		if value.Flags&ir.I18nParamValueFlagsCloseTag != 0 {
			return elementValue + templateValue
		}
		return templateValue + elementValue
	}

	// Self-closing tags use a special form that concatenates the start and close tag values.
	if (value.Flags&ir.I18nParamValueFlagsOpenTag != 0) &&
		(value.Flags&ir.I18nParamValueFlagsCloseTag != 0) {
		openValue := formatValue(ops_create.I18nParamValue{
			Value:            value.Value,
			SubTemplateIndex: value.SubTemplateIndex,
			Flags:            value.Flags &^ ir.I18nParamValueFlagsCloseTag,
		})
		closeValue := formatValue(ops_create.I18nParamValue{
			Value:            value.Value,
			SubTemplateIndex: value.SubTemplateIndex,
			Flags:            value.Flags &^ ir.I18nParamValueFlagsOpenTag,
		})
		return openValue + closeValue
	}

	// If there are no special flags, just return the raw value.
	if value.Flags == ir.I18nParamValueFlagsNone {
		return fmt.Sprintf("%v", value.Value)
	}

	// Encode the remaining flags as part of the value.
	var tagMarker string
	var closeMarker string
	if value.Flags&ir.I18nParamValueFlagsElementTag != 0 {
		tagMarker = ELEMENT_MARKER
	} else if value.Flags&ir.I18nParamValueFlagsTemplateTag != 0 {
		tagMarker = TEMPLATE_MARKER
	}
	if tagMarker != "" {
		if value.Flags&ir.I18nParamValueFlagsCloseTag != 0 {
			closeMarker = TAG_CLOSE_MARKER
		}
	}
	context := ""
	if value.SubTemplateIndex != nil {
		context = fmt.Sprintf("%s%d", CONTEXT_MARKER, *value.SubTemplateIndex)
	}
	return fmt.Sprintf("%s%s%s%v%s%s", ESCAPE, closeMarker, tagMarker, value.Value, context, ESCAPE)
}
