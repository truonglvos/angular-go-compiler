package pipeline

import (
	"fmt"
	"strings"

	"ngc-go/packages/compiler/src/constant"
	"ngc-go/packages/compiler/src/expression_parser"
	"ngc-go/packages/compiler/src/i18n"
	"ngc-go/packages/compiler/src/ml_parser"
	"ngc-go/packages/compiler/src/output"
	"ngc-go/packages/compiler/src/render3"
	"ngc-go/packages/compiler/src/render3/view"
	view_i18n "ngc-go/packages/compiler/src/render3/view/i18n"
	"ngc-go/packages/compiler/src/schema"
	"ngc-go/packages/compiler/src/template/pipeline/ir"
	"ngc-go/packages/compiler/src/template/pipeline/ir/src/expression"
	ir_operations "ngc-go/packages/compiler/src/template/pipeline/ir/src/operations"
	ops_create "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/create"
	"ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/shared"
	ops_update "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/update"
	ir_variable "ngc-go/packages/compiler/src/template/pipeline/ir/src/variable"
	"ngc-go/packages/compiler/src/template/pipeline/src/compilation"
	pipeline_convension "ngc-go/packages/compiler/src/template/pipeline/src/convension"
	"ngc-go/packages/compiler/src/util"
)

const compatibilityMode = ir.CompatibilityModeTemplateDefinitionBuilder

// domSchema contains DOM elements and their properties
var domSchema = schema.NewDomElementSchemaRegistry()

// NG_TEMPLATE_TAG_NAME is the tag name of the `ng-template` element
const NG_TEMPLATE_TAG_NAME = "ng-template"

// ANIMATE_PREFIX is the prefix for any animation binding
const ANIMATE_PREFIX = "animate."

// IsI18nRootNode checks if the given meta is an i18n Message (root node)
func IsI18nRootNode(meta interface{}) bool {
	_, ok := meta.(*i18n.Message)
	return ok
}

// IsSingleI18nIcu checks if the given meta is a single ICU i18n message
func IsSingleI18nIcu(meta interface{}) bool {
	if !IsI18nRootNode(meta) {
		return false
	}
	msg, ok := meta.(*i18n.Message)
	if !ok {
		return false
	}
	if len(msg.Nodes) != 1 {
		return false
	}
	_, ok = msg.Nodes[0].(*i18n.Icu)
	return ok
}

// IngestComponent processes a template AST and converts it into a ComponentCompilationJob in the intermediate representation
func IngestComponent(
	componentName string,
	template []render3.Node,
	constantPool *constant.ConstantPool,
	compilationMode compilation.TemplateCompilationMode,
	relativeContextFilePath string,
	i18nUseExternalIds bool,
	deferMeta view.R3ComponentDeferMetadata,
	allDeferrableDepsFn *output.ReadVarExpr,
	relativeTemplatePath *string,
	enableDebugLocations bool,
) *compilation.ComponentCompilationJob {
	job := compilation.NewComponentCompilationJob(
		componentName,
		constantPool,
		compatibilityMode,
		compilationMode,
		relativeContextFilePath,
		i18nUseExternalIds,
		deferMeta,
		allDeferrableDepsFn,
		relativeTemplatePath,
		enableDebugLocations,
	)
	ingestNodes(job.Root, template)
	return job
}

// HostBindingInput represents input for host binding ingestion
type HostBindingInput struct {
	ComponentName     string
	ComponentSelector string
	Properties        []*expression_parser.ParsedProperty
	Attributes        map[string]output.OutputExpression
	Events            []*expression_parser.ParsedEvent
}

// IngestHostBinding processes a host binding AST and converts it into a HostBindingCompilationJob in the intermediate representation
func IngestHostBinding(
	input *HostBindingInput,
	bindingParser interface{}, // TODO: BindingParser type
	constantPool *constant.ConstantPool,
) *compilation.HostBindingCompilationJob {
	job := compilation.NewHostBindingCompilationJob(
		input.ComponentName,
		constantPool,
		compatibilityMode,
		compilation.TemplateCompilationModeDomOnly,
	)

	// TODO: Implement property, attribute, and event ingestion
	// This requires the BindingParser interface to be defined

	return job
}

// ingestNodes ingests the nodes of a template AST into the given ViewCompilationUnit
func ingestNodes(unit *compilation.ViewCompilationUnit, template []render3.Node) {
	for _, node := range template {
		switch n := node.(type) {
		case *render3.Element:
			ingestElement(unit, n)
		case *render3.Template:
			ingestTemplate(unit, n)
		case *render3.Content:
			ingestContent(unit, n)
		case *render3.Text:
			ingestText(unit, n, nil)
		case *render3.BoundText:
			ingestBoundText(unit, n, nil)
		case *render3.IfBlock:
			ingestIfBlock(unit, n)
		case *render3.SwitchBlock:
			ingestSwitchBlock(unit, n)
		case *render3.DeferredBlock:
			ingestDeferBlock(unit, n)
		case *render3.Icu:
			ingestIcu(unit, n)
		case *render3.ForLoopBlock:
			ingestForBlock(unit, n)
		case *render3.LetDeclaration:
			ingestLetDeclaration(unit, n)
		case *render3.Component:
			// TODO(crisbeto): account for selectorless nodes.
		default:
			panic(fmt.Sprintf("Unsupported template node: %T", node))
		}
	}
}

// ingestElement ingests an element AST from the template into the given ViewCompilationUnit
func ingestElement(unit *compilation.ViewCompilationUnit, element *render3.Element) {
	if element.I18n != nil {
		_, isMessage := element.I18n.(*i18n.Message)
		_, isTagPlaceholder := element.I18n.(*i18n.TagPlaceholder)
		if !isMessage && !isTagPlaceholder {
			panic(fmt.Sprintf("Unhandled i18n metadata type for element: %T", element.I18n))
		}
	}

	id := unit.Job.AllocateXrefId()

	namespaceKey, elementName := ml_parser.SplitNsName(element.Name, false)

	var i18nPlaceholder interface{}
	if tagPlaceholder, ok := element.I18n.(*i18n.TagPlaceholder); ok {
		i18nPlaceholder = tagPlaceholder
	}

	startOp := ops_create.NewElementStartOp(
		elementName,
		id,
		pipeline_convension.NamespaceForKey(&namespaceKey),
		i18nPlaceholder,
		element.StartSourceSpan,
		element.SourceSpan(),
	)
	unit.Create.Push(startOp)

	ingestElementBindings(unit, startOp, element)
	ingestReferences(startOp, element)

	// Start i18n, if needed, goes after the element create and bindings, but before the nodes
	var i18nBlockId ir_operations.XrefId = 0
	if msg, ok := element.I18n.(*i18n.Message); ok {
		i18nBlockId = unit.Job.AllocateXrefId()
		unit.Create.Push(
			ops_create.NewI18nStartOp(i18nBlockId, msg, 0, element.StartSourceSpan),
		)
	}

	ingestNodes(unit, element.Children)

	// The source span for the end op is typically the element closing tag. However, if no closing tag
	// exists, such as in `<input>`, we use the start source span instead.
	endSourceSpan := element.EndSourceSpan
	if endSourceSpan == nil {
		endSourceSpan = element.StartSourceSpan
	}
	endOp := ops_create.NewElementEndOp(id, endSourceSpan)
	unit.Create.Push(endOp)

	// If there is an i18n message associated with this element, insert i18n end op
	if i18nBlockId != 0 {
		unit.Create.InsertBefore(
			endOp,
			ops_create.NewI18nEndOp(i18nBlockId, endSourceSpan),
		)
	}
}

// ingestTemplate ingests an `ng-template` node from the AST into the given ViewCompilationUnit
func ingestTemplate(unit *compilation.ViewCompilationUnit, tmpl *render3.Template) {
	if tmpl.I18n != nil {
		_, isMessage := tmpl.I18n.(*i18n.Message)
		_, isTagPlaceholder := tmpl.I18n.(*i18n.TagPlaceholder)
		if !isMessage && !isTagPlaceholder {
			panic(fmt.Sprintf("Unhandled i18n metadata type for template: %T", tmpl.I18n))
		}
	}

	childView := unit.Job.AllocateView(unit.Xref)

	var tagNameWithoutNamespace *string
	var namespacePrefix string
	if tmpl.TagName != nil {
		var tagNameStr string
		namespacePrefix, tagNameStr = ml_parser.SplitNsName(*tmpl.TagName, false)
		tagNameWithoutNamespace = &tagNameStr
	}

	var i18nPlaceholder interface{}
	if tagPlaceholder, ok := tmpl.I18n.(*i18n.TagPlaceholder); ok {
		i18nPlaceholder = tagPlaceholder
	}
	namespace := pipeline_convension.NamespaceForKey(&namespacePrefix)

	var functionNameSuffix string
	if tagNameWithoutNamespace != nil {
		functionNameSuffix = pipeline_convension.PrefixWithNamespace(*tagNameWithoutNamespace, namespace)
	}

	templateKind := ir.TemplateKindNgTemplate
	if !isPlainTemplate(tmpl) {
		templateKind = ir.TemplateKindStructural
	}

	templateOp := ops_create.NewTemplateOp(
		childView.Xref,
		templateKind,
		tagNameWithoutNamespace,
		functionNameSuffix,
		namespace,
		i18nPlaceholder,
		tmpl.StartSourceSpan,
		tmpl.SourceSpan(),
	)
	unit.Create.Push(templateOp)

	ingestTemplateBindings(unit, templateOp, tmpl, templateKind)
	ingestReferences(templateOp, tmpl)
	ingestNodes(childView, tmpl.Children)

	for _, variable := range tmpl.Variables {
		value := variable.Value
		if value == "" {
			value = "$implicit"
		}
		childView.ContextVariables[variable.Name] = value
	}

	// If this is a plain template and there is an i18n message associated with it, insert i18n start
	// and end ops. For structural directive templates, the i18n ops will be added when ingesting the
	// element/template the directive is placed on.
	if templateKind == ir.TemplateKindNgTemplate {
		if msg, ok := tmpl.I18n.(*i18n.Message); ok {
			id := unit.Job.AllocateXrefId()
			childView.Create.InsertAfter(
				childView.Create.Head(),
				ops_create.NewI18nStartOp(id, msg, 0, tmpl.StartSourceSpan),
			)
			endSpan := tmpl.EndSourceSpan
			if endSpan == nil {
				endSpan = tmpl.StartSourceSpan
			}
			childView.Create.InsertBefore(
				childView.Create.Tail(),
				ops_create.NewI18nEndOp(id, endSpan),
			)
		}
	}
}

// ingestContent ingests a content node from the AST into the given ViewCompilationUnit
func ingestContent(unit *compilation.ViewCompilationUnit, content *render3.Content) {
	if content.I18n != nil {
		_, isTagPlaceholder := content.I18n.(*i18n.TagPlaceholder)
		if !isTagPlaceholder {
			panic(fmt.Sprintf("Unhandled i18n metadata type for element: %T", content.I18n))
		}
	}

	var fallbackView *compilation.ViewCompilationUnit

	// Don't capture default content that's only made up of empty text nodes and comments.
	// Note that we process the default content before the projection in order to match the
	// insertion order at runtime.
	hasNonEmptyContent := false
	for _, child := range content.Children {
		_, isComment := child.(*render3.Comment)
		if text, isText := child.(*render3.Text); isText {
			if strings.TrimSpace(text.Value) != "" {
				hasNonEmptyContent = true
				break
			}
		} else if !isComment {
			hasNonEmptyContent = true
			break
		}
	}

	if hasNonEmptyContent {
		fallbackView = unit.Job.AllocateView(unit.Xref)
		ingestNodes(fallbackView, content.Children)
	}

	id := unit.Job.AllocateXrefId()
	fallbackXref := ir_operations.XrefId(0)
	if fallbackView != nil {
		fallbackXref = fallbackView.Xref
	}
	op := ops_create.NewProjectionOp(
		id,
		content.Selector,
		content.I18n,
		fallbackXref,
		content.SourceSpan(),
	)

	for _, attr := range content.Attributes {
		securityContext := domSchema.SecurityContext("ng-content", attr.Name, true)
		unit.Update.Push(
			ops_update.NewBindingOp(
				op.Xref,
				ir.BindingKindAttribute,
				attr.Name,
				output.NewLiteralExpr(attr.Value, nil, nil),
				nil,
				securityContext,
				true,
				false,
				nil,
				asMessage(attr.I18n),
				attr.SourceSpan(),
			),
		)
	}
	unit.Create.Push(op)
}

// ingestText ingests a literal text node from the AST into the given ViewCompilationUnit
func ingestText(unit *compilation.ViewCompilationUnit, text *render3.Text, icuPlaceholder *string) {
	unit.Create.Push(
		ops_create.NewTextOp(unit.Job.AllocateXrefId(), text.Value, icuPlaceholder, text.SourceSpan()),
	)
}

// ingestBoundText ingests an interpolated text node from the AST into the given ViewCompilationUnit
func ingestBoundText(
	unit *compilation.ViewCompilationUnit,
	text *render3.BoundText,
	icuPlaceholder *string,
) {
	value := text.Value
	if astWithSource, ok := value.(*expression_parser.ASTWithSource); ok {
		value = astWithSource.AST
	}

	interpolation, ok := value.(*expression_parser.Interpolation)
	if !ok {
		panic(fmt.Sprintf(
			"AssertionError: expected Interpolation for BoundText node, got %T",
			value,
		))
	}

	if text.I18n != nil {
		_, isContainer := text.I18n.(*i18n.Container)
		if !isContainer {
			panic(fmt.Sprintf(
				"Unhandled i18n metadata type for text interpolation: %T",
				text.I18n,
			))
		}
	}

	var i18nPlaceholders []string
	if container, ok := text.I18n.(*i18n.Container); ok {
		for _, node := range container.Children {
			if placeholder, ok := node.(*i18n.Placeholder); ok {
				i18nPlaceholders = append(i18nPlaceholders, placeholder.Name)
			}
		}
	}

	if len(i18nPlaceholders) > 0 && len(i18nPlaceholders) != len(interpolation.Expressions) {
		panic(fmt.Sprintf(
			"Unexpected number of i18n placeholders (%d) for BoundText with %d expressions",
			len(i18nPlaceholders),
			len(interpolation.Expressions),
		))
	}

	textXref := unit.Job.AllocateXrefId()
	unit.Create.Push(ops_create.NewTextOp(textXref, "", icuPlaceholder, text.SourceSpan()))

	// TemplateDefinitionBuilder does not generate source maps for sub-expressions inside an
	// interpolation. We copy that behavior in compatibility mode.
	var baseSourceSpan *util.ParseSourceSpan
	if unit.Job.Compatibility != ir.CompatibilityModeTemplateDefinitionBuilder {
		baseSourceSpan = text.SourceSpan()
	}

	interpolationObj, err := ops_update.NewInterpolation(
		interpolation.Strings,
		convertExpressions(interpolation.Expressions, unit.GetJob(), baseSourceSpan),
		i18nPlaceholders,
	)
	if err != nil {
		panic(err)
	}

	unit.Update.Push(
		ops_update.NewInterpolateTextOp(
			textXref,
			interpolationObj,
			text.SourceSpan(),
		),
	)
}

// Helper function to convert expressions
func convertExpressions(
	expressions []expression_parser.AST,
	job *compilation.CompilationJob,
	baseSourceSpan *util.ParseSourceSpan,
) []output.OutputExpression {
	result := make([]output.OutputExpression, len(expressions))
	for i, expr := range expressions {
		result[i] = convertAst(expr, job, baseSourceSpan)
	}
	return result
}

// Helper function to insert an op before another op in a list
func OpListInsertBefore(newOp ir_operations.Op, op ir_operations.Op) {
	// Get the list from the op's debug list ID
	if op.GetDebugListId() == nil {
		panic("operations is not owned by a list")
	}
	// Note: In a real implementation, we'd need to track which list owns which op
	// For now, we'll assume the op knows its list context
	// This is a simplified version - the actual implementation would need list tracking
	panic("OpListInsertBefore requires list context - use list.InsertBefore directly")
}

// Helper function to insert an op after another op in a list
func OpListInsertAfter(newOp ir_operations.Op, op ir_operations.Op) {
	// Get the list from the op's debug list ID
	if op.GetDebugListId() == nil {
		panic("operations is not owned by a list")
	}
	// Note: In a real implementation, we'd need to track which list owns which op
	// For now, we'll assume the op knows its list context
	// This is a simplified version - the actual implementation would need list tracking
	panic("OpListInsertAfter requires list context - use list.InsertAfter directly")
}

// ingestIfBlock ingests an `@if` block into the given ViewCompilationUnit
func ingestIfBlock(unit *compilation.ViewCompilationUnit, ifBlock *render3.IfBlock) {
	var firstXref ir_operations.XrefId = 0
	conditions := make([]interface{}, 0) // []*ir.ConditionalCaseExpr

	for i, ifCase := range ifBlock.Branches {
		cView := unit.Job.AllocateView(unit.Xref)
		tagName := ingestControlFlowInsertionPoint(unit, cView.Xref, ifCase)

		if ifCase.ExpressionAlias != nil {
			cView.ContextVariables[ifCase.ExpressionAlias.Name] = ir_variable.CTX_REF
		}

		var ifCaseI18nMeta interface{} // *i18n.BlockPlaceholder
		if ifCase.I18n != nil {
			if blockPlaceholder, ok := ifCase.I18n.(*i18n.BlockPlaceholder); ok {
				ifCaseI18nMeta = blockPlaceholder
			} else {
				panic(fmt.Sprintf("Unhandled i18n metadata type for if block: %T", ifCase.I18n))
			}
		}

		var conditionalCreateOp ir_operations.CreateOp
		if i == 0 {
			tagNamePtr := &tagName
			if tagName == "" {
				tagNamePtr = nil
			}
			conditionalCreateOp = ops_create.NewConditionalCreateOp(
				cView.Xref,
				ir.TemplateKindBlock,
				tagNamePtr,
				"Conditional",
				ir.NamespaceHTML,
				ifCaseI18nMeta,
				ifCase.StartSourceSpan,
				ifCase.SourceSpan(),
			)
		} else {
			tagNamePtr := &tagName
			if tagName == "" {
				tagNamePtr = nil
			}
			conditionalCreateOp = ops_create.NewConditionalBranchCreateOp(
				cView.Xref,
				ir.TemplateKindBlock,
				tagNamePtr,
				"Conditional",
				ir.NamespaceHTML,
				ifCaseI18nMeta,
				ifCase.StartSourceSpan,
				ifCase.SourceSpan(),
			)
		}
		unit.Create.Push(conditionalCreateOp)

		if firstXref == 0 {
			firstXref = cView.Xref
		}

		var caseExpr output.OutputExpression
		if ifCase.Expression != nil {
			caseExpr = convertAst(ifCase.Expression, unit.GetJob(), nil)
		}

		// Get Handle from concrete type
		var handle *ir.SlotHandle
		if condOp, ok := conditionalCreateOp.(*ops_create.ConditionalCreateOp); ok {
			handle = condOp.Handle
		} else if branchOp, ok := conditionalCreateOp.(*ops_create.ConditionalBranchCreateOp); ok {
			handle = branchOp.Handle
		} else {
			panic(fmt.Sprintf("Unexpected conditional create op type: %T", conditionalCreateOp))
		}

		conditionalCaseExpr := expression.NewConditionalCaseExpr(
			caseExpr,
			conditionalCreateOp.GetXref(),
			handle,
			ifCase.ExpressionAlias,
		)
		conditions = append(conditions, conditionalCaseExpr)
		ingestNodes(cView, ifCase.Children)
	}

	if firstXref == 0 {
		panic("ingestIfBlock: firstXref should not be zero")
	}

	unit.Update.Push(ops_update.NewConditionalOp(firstXref, conditions))
}

// ingestSwitchBlock ingests an `@switch` block into the given ViewCompilationUnit
func ingestSwitchBlock(unit *compilation.ViewCompilationUnit, switchBlock *render3.SwitchBlock) {
	// Don't ingest empty switches since they won't render anything
	if len(switchBlock.Cases) == 0 {
		return
	}

	var firstXref ir_operations.XrefId = 0
	conditions := make([]interface{}, 0) // []*ir.ConditionalCaseExpr

	for i, switchCase := range switchBlock.Cases {
		cView := unit.Job.AllocateView(unit.Xref)
		tagName := ingestControlFlowInsertionPoint(unit, cView.Xref, switchCase)

		var switchCaseI18nMeta interface{} // *i18n.BlockPlaceholder
		if switchCase.I18n != nil {
			if blockPlaceholder, ok := switchCase.I18n.(*i18n.BlockPlaceholder); ok {
				switchCaseI18nMeta = blockPlaceholder
			} else {
				panic(fmt.Sprintf("Unhandled i18n metadata type for switch block: %T", switchCase.I18n))
			}
		}

		var conditionalCreateOp ir_operations.CreateOp
		if i == 0 {
			tagNamePtr := &tagName
			if tagName == "" {
				tagNamePtr = nil
			}
			conditionalCreateOp = ops_create.NewConditionalCreateOp(
				cView.Xref,
				ir.TemplateKindBlock,
				tagNamePtr,
				"Case",
				ir.NamespaceHTML,
				switchCaseI18nMeta,
				switchCase.StartSourceSpan,
				switchCase.SourceSpan(),
			)
		} else {
			tagNamePtr := &tagName
			if tagName == "" {
				tagNamePtr = nil
			}
			conditionalCreateOp = ops_create.NewConditionalBranchCreateOp(
				cView.Xref,
				ir.TemplateKindBlock,
				tagNamePtr,
				"Case",
				ir.NamespaceHTML,
				switchCaseI18nMeta,
				switchCase.StartSourceSpan,
				switchCase.SourceSpan(),
			)
		}
		unit.Create.Push(conditionalCreateOp)

		if firstXref == 0 {
			firstXref = cView.Xref
		}

		var caseExpr output.OutputExpression
		if switchCase.Expression != nil {
			caseExpr = convertAst(switchCase.Expression, unit.GetJob(), switchBlock.StartSourceSpan)
		}

		// Get Handle from concrete type
		var handle *ir.SlotHandle
		if condOp, ok := conditionalCreateOp.(*ops_create.ConditionalCreateOp); ok {
			handle = condOp.Handle
		} else if branchOp, ok := conditionalCreateOp.(*ops_create.ConditionalBranchCreateOp); ok {
			handle = branchOp.Handle
		} else {
			panic(fmt.Sprintf("Unexpected conditional create op type: %T", conditionalCreateOp))
		}

		conditionalCaseExpr := expression.NewConditionalCaseExpr(
			caseExpr,
			conditionalCreateOp.GetXref(),
			handle,
			nil, // switch cases don't have expression aliases
		)
		conditions = append(conditions, conditionalCaseExpr)
		ingestNodes(cView, switchCase.Children)
	}

	if firstXref == 0 {
		panic("ingestSwitchBlock: firstXref should not be zero")
	}

	switchExpr := convertAst(switchBlock.Expression, unit.GetJob(), nil)
	conditionalOp := ops_update.NewConditionalOp(firstXref, conditions)
	conditionalOp.ContextValue = switchExpr
	unit.Update.Push(conditionalOp)
}

// ingestControlFlowInsertionPoint extracts tag name and attributes from a control flow block
// for content projection purposes
func ingestControlFlowInsertionPoint(
	unit *compilation.ViewCompilationUnit,
	xref ir_operations.XrefId,
	node interface{}, // render3.IfBlockBranch | render3.SwitchBlockCase | render3.ForLoopBlock | render3.ForLoopBlockEmpty
) string {
	var root interface{} // render3.Element | render3.Template

	// Get children based on node type
	var children []render3.Node
	switch n := node.(type) {
	case *render3.IfBlockBranch:
		children = n.Children
	case *render3.SwitchBlockCase:
		children = n.Children
	case *render3.ForLoopBlock:
		children = n.Children
	case *render3.ForLoopBlockEmpty:
		children = n.Children
	default:
		panic(fmt.Sprintf("Unsupported node type for ingestControlFlowInsertionPoint: %T", node))
	}

	for _, child := range children {
		// Skip over comment nodes and @let declarations since
		// it doesn't matter where they end up in the DOM.
		if _, ok := child.(*render3.Comment); ok {
			continue
		}
		if _, ok := child.(*render3.LetDeclaration); ok {
			continue
		}

		// We can only infer the tag name/attributes if there's a single root node.
		if root != nil {
			return ""
		}

		// Root nodes can only elements or templates with a tag name (e.g. `<div *foo></div>`).
		if element, ok := child.(*render3.Element); ok {
			root = element
		} else if template, ok := child.(*render3.Template); ok && template.TagName != nil {
			root = template
		} else {
			return ""
		}
	}

	// If we've found a single root node, its tag name and attributes can be
	// copied to the surrounding template to be used for content projection.
	if root != nil {
		var rootElement *render3.Element
		var rootTemplate *render3.Template
		var tagName string

		if element, ok := root.(*render3.Element); ok {
			rootElement = element
			tagName = element.Name
		} else if template, ok := root.(*render3.Template); ok {
			rootTemplate = template
			if template.TagName != nil {
				tagName = *template.TagName
			}
		}

		// Collect the static attributes for content projection purposes.
		if rootElement != nil {
			for _, attr := range rootElement.Attributes {
				if !strings.HasPrefix(attr.Name, ANIMATE_PREFIX) {
					securityContext := domSchema.SecurityContext(NG_TEMPLATE_TAG_NAME, attr.Name, true)
					unit.Update.Push(ops_update.NewBindingOp(
						xref,
						ir.BindingKindAttribute,
						attr.Name,
						output.NewLiteralExpr(attr.Value, nil, nil),
						nil,
						securityContext,
						true,
						false,
						nil,
						asMessage(attr.I18n),
						attr.SourceSpan(),
					))
				}
			}

			// Also collect the inputs since they participate in content projection as well.
			for _, input := range rootElement.Inputs {
				if input.Type != expression_parser.BindingTypeLegacyAnimation &&
					input.Type != expression_parser.BindingTypeAnimation &&
					input.Type != expression_parser.BindingTypeAttribute {
					securityContext := domSchema.SecurityContext(NG_TEMPLATE_TAG_NAME, input.Name, true)
					unit.Create.Push(ops_create.NewExtractedAttributeOp(
						xref,
						ir.BindingKindProperty,
						nil,
						input.Name,
						nil,
						0,
						nil,
						securityContext,
					))
				}
			}
		} else if rootTemplate != nil {
			// Templates don't have attributes in the same way, but we can still collect inputs
			for _, input := range rootTemplate.Inputs {
				if input.Type != expression_parser.BindingTypeLegacyAnimation &&
					input.Type != expression_parser.BindingTypeAnimation &&
					input.Type != expression_parser.BindingTypeAttribute {
					securityContext := domSchema.SecurityContext(NG_TEMPLATE_TAG_NAME, input.Name, true)
					unit.Create.Push(ops_create.NewExtractedAttributeOp(
						xref,
						ir.BindingKindProperty,
						nil,
						input.Name,
						nil,
						0,
						nil,
						securityContext,
					))
				}
			}
		}

		// Don't pass along `ng-template` tag name since it enables directive matching.
		if tagName == NG_TEMPLATE_TAG_NAME {
			return ""
		}
		return tagName
	}

	return ""
}

// ingestDeferView creates a view for a defer block section (main, loading, placeholder, error)
func ingestDeferView(
	unit *compilation.ViewCompilationUnit,
	suffix string,
	i18nMeta interface{}, // i18n.I18nMeta
	children []render3.Node,
	sourceSpan *util.ParseSourceSpan,
) *ops_create.TemplateOp {
	if i18nMeta != nil {
		if _, ok := i18nMeta.(*i18n.BlockPlaceholder); !ok {
			panic("Unhandled i18n metadata type for defer block")
		}
	}
	if children == nil {
		return nil
	}
	secondaryView := unit.Job.AllocateView(unit.Xref)
	ingestNodes(secondaryView, children)

	var i18nPlaceholder interface{}
	if blockPlaceholder, ok := i18nMeta.(*i18n.BlockPlaceholder); ok {
		i18nPlaceholder = blockPlaceholder
	}

	functionNameSuffix := fmt.Sprintf("Defer%s", suffix)
	templateOp := ops_create.NewTemplateOp(
		secondaryView.Xref,
		ir.TemplateKindBlock,
		nil,
		functionNameSuffix,
		ir.NamespaceHTML,
		i18nPlaceholder,
		sourceSpan,
		sourceSpan,
	)
	unit.Create.Push(templateOp)
	return templateOp
}

// calcDeferBlockFlags calculates flags for a defer block
func calcDeferBlockFlags(deferBlock *render3.DeferredBlock) ir.TDeferDetailsFlags {
	if deferBlock.HydrateTriggers != nil {
		// Check if there are any hydrate triggers
		hasHydrateTriggers := deferBlock.HydrateTriggers.Idle != nil ||
			deferBlock.HydrateTriggers.Immediate != nil ||
			deferBlock.HydrateTriggers.Timer != nil ||
			deferBlock.HydrateTriggers.Hover != nil ||
			deferBlock.HydrateTriggers.Interaction != nil ||
			deferBlock.HydrateTriggers.Viewport != nil ||
			deferBlock.HydrateTriggers.Never != nil ||
			deferBlock.HydrateTriggers.When != nil
		if hasHydrateTriggers {
			return ir.TDeferDetailsFlagsHasHydrateTriggers
		}
	}
	return 0
}

// ingestDeferTriggers processes defer triggers and creates DeferOnOp and DeferWhenOp operations
func ingestDeferTriggers(
	modifier ir.DeferOpModifierKind,
	triggers *render3.DeferredBlockTriggers,
	onOps []ir_operations.CreateOp, // []*ops.DeferOnOp
	whenOps []ir_operations.UpdateOp, // []*ops.DeferWhenOp
	unit *compilation.ViewCompilationUnit,
	deferXref ir_operations.XrefId,
) ([]ir_operations.CreateOp, []ir_operations.UpdateOp) {
	if triggers == nil {
		return onOps, whenOps
	}

	if triggers.Idle != nil {
		trigger := &ops_create.DeferIdleTrigger{Kind: ir.DeferTriggerKindIdle}
		deferOnOp := ops_create.NewDeferOnOp(
			deferXref,
			trigger,
			modifier,
			triggers.Idle.SourceSpan(),
		)
		onOps = append(onOps, deferOnOp)
	}

	if triggers.Immediate != nil {
		trigger := &ops_create.DeferImmediateTrigger{Kind: ir.DeferTriggerKindImmediate}
		deferOnOp := ops_create.NewDeferOnOp(
			deferXref,
			trigger,
			modifier,
			triggers.Immediate.SourceSpan(),
		)
		onOps = append(onOps, deferOnOp)
	}

	if triggers.Timer != nil {
		trigger := &ops_create.DeferTimerTrigger{
			Kind:  ir.DeferTriggerKindTimer,
			Delay: triggers.Timer.Delay,
		}
		deferOnOp := ops_create.NewDeferOnOp(
			deferXref,
			trigger,
			modifier,
			triggers.Timer.SourceSpan(),
		)
		onOps = append(onOps, deferOnOp)
	}

	if triggers.Hover != nil {
		trigger := &ops_create.DeferHoverTrigger{
			DeferTriggerWithTargetBase: ops_create.DeferTriggerWithTargetBase{
				Kind:                ir.DeferTriggerKindHover,
				TargetName:          triggers.Hover.Reference,
				TargetXref:          0,
				TargetSlot:          nil,
				TargetView:          0,
				TargetSlotViewSteps: nil,
			},
		}
		deferOnOp := ops_create.NewDeferOnOp(
			deferXref,
			trigger,
			modifier,
			triggers.Hover.SourceSpan(),
		)
		onOps = append(onOps, deferOnOp)
	}

	if triggers.Interaction != nil {
		trigger := &ops_create.DeferInteractionTrigger{
			DeferTriggerWithTargetBase: ops_create.DeferTriggerWithTargetBase{
				Kind:                ir.DeferTriggerKindInteraction,
				TargetName:          triggers.Interaction.Reference,
				TargetXref:          0,
				TargetSlot:          nil,
				TargetView:          0,
				TargetSlotViewSteps: nil,
			},
		}
		deferOnOp := ops_create.NewDeferOnOp(
			deferXref,
			trigger,
			modifier,
			triggers.Interaction.SourceSpan(),
		)
		onOps = append(onOps, deferOnOp)
	}

	if triggers.Viewport != nil {
		var options output.OutputExpression
		if triggers.Viewport.Options != nil {
			options = convertAst(triggers.Viewport.Options, unit.GetJob(), triggers.Viewport.SourceSpan())
		}
		trigger := &ops_create.DeferViewportTrigger{
			DeferTriggerWithTargetBase: ops_create.DeferTriggerWithTargetBase{
				Kind:                ir.DeferTriggerKindViewport,
				TargetName:          triggers.Viewport.Reference,
				TargetXref:          0,
				TargetSlot:          nil,
				TargetView:          0,
				TargetSlotViewSteps: nil,
			},
			Options: options,
		}
		deferOnOp := ops_create.NewDeferOnOp(
			deferXref,
			trigger,
			modifier,
			triggers.Viewport.SourceSpan(),
		)
		onOps = append(onOps, deferOnOp)
	}

	if triggers.Never != nil {
		trigger := &ops_create.DeferNeverTrigger{Kind: ir.DeferTriggerKindNever}
		deferOnOp := ops_create.NewDeferOnOp(
			deferXref,
			trigger,
			modifier,
			triggers.Never.SourceSpan(),
		)
		onOps = append(onOps, deferOnOp)
	}

	if triggers.When != nil {
		if _, ok := triggers.When.Value.(*expression_parser.Interpolation); ok {
			panic("Unexpected interpolation in defer block when trigger")
		}
		deferWhenOp := ops_update.NewDeferWhenOp(
			deferXref,
			convertAst(triggers.When.Value, unit.GetJob(), triggers.When.SourceSpan()),
			modifier,
			triggers.When.SourceSpan(),
		)
		whenOps = append(whenOps, deferWhenOp)
	}

	return onOps, whenOps
}

// ingestDeferBlock ingests a `@defer` block into the given ViewCompilationUnit
func ingestDeferBlock(unit *compilation.ViewCompilationUnit, deferBlock *render3.DeferredBlock) {
	var ownResolverFn output.OutputExpression

	// Check if we need a per-block resolver function
	if unit.Job.DeferMeta.Mode == view.DeferBlockDepsEmitModePerBlock {
		// TODO: Look up deferBlock in unit.Job.DeferMeta.Blocks map
		// For now, we'll leave it as nil
		ownResolverFn = nil
	}

	// Generate the defer main view and all secondary views.
	main := ingestDeferView(
		unit,
		"",
		deferBlock.I18n,
		deferBlock.Children,
		deferBlock.SourceSpan(),
	)
	if main == nil {
		panic("Defer block must have children")
	}

	var loading *ops_create.TemplateOp
	if deferBlock.Loading != nil {
		loading = ingestDeferView(
			unit,
			"Loading",
			deferBlock.Loading.I18n,
			deferBlock.Loading.Children,
			deferBlock.Loading.SourceSpan(),
		)
	}

	var placeholder *ops_create.TemplateOp
	if deferBlock.Placeholder != nil {
		placeholder = ingestDeferView(
			unit,
			"Placeholder",
			deferBlock.Placeholder.I18n,
			deferBlock.Placeholder.Children,
			deferBlock.Placeholder.SourceSpan(),
		)
	}

	var errorView *ops_create.TemplateOp
	if deferBlock.Error != nil {
		errorView = ingestDeferView(
			unit,
			"Error",
			deferBlock.Error.I18n,
			deferBlock.Error.Children,
			deferBlock.Error.SourceSpan(),
		)
	}

	// Create the main defer op, and ops for all secondary views.
	deferXref := unit.Job.AllocateXrefId()
	deferOp := ops_create.NewDeferOp(
		deferXref,
		main.Xref,
		main.Handle,
		ownResolverFn,
		unit.Job.AllDeferrableDepsFn,
		deferBlock.SourceSpan(),
	)

	if placeholder != nil {
		deferOp.PlaceholderView = placeholder.Xref
		deferOp.PlaceholderSlot = placeholder.Handle
	}
	if loading != nil {
		deferOp.LoadingSlot = loading.Handle
	}
	if errorView != nil {
		deferOp.ErrorSlot = errorView.Handle
	}

	if deferBlock.Placeholder != nil {
		deferOp.PlaceholderMinimumTime = deferBlock.Placeholder.MinimumTime
	}
	if deferBlock.Loading != nil {
		deferOp.LoadingMinimumTime = deferBlock.Loading.MinimumTime
		deferOp.LoadingAfterTime = deferBlock.Loading.AfterTime
	}

	deferOp.Flags = calcDeferBlockFlags(deferBlock)
	unit.Create.Push(deferOp)

	// Configure all defer `on` conditions.
	var deferOnOps []ir_operations.CreateOp
	var deferWhenOps []ir_operations.UpdateOp

	// Ingest the hydrate triggers first since they set up all the other triggers during SSR.
	if deferBlock.HydrateTriggers != nil {
		deferOnOps, deferWhenOps = ingestDeferTriggers(
			ir.DeferOpModifierKindHydrate,
			deferBlock.HydrateTriggers,
			deferOnOps,
			deferWhenOps,
			unit,
			deferXref,
		)
	}

	if deferBlock.Triggers != nil {
		deferOnOps, deferWhenOps = ingestDeferTriggers(
			ir.DeferOpModifierKindNone,
			deferBlock.Triggers,
			deferOnOps,
			deferWhenOps,
			unit,
			deferXref,
		)
	}

	if deferBlock.PrefetchTriggers != nil {
		deferOnOps, deferWhenOps = ingestDeferTriggers(
			ir.DeferOpModifierKindPrefetch,
			deferBlock.PrefetchTriggers,
			deferOnOps,
			deferWhenOps,
			unit,
			deferXref,
		)
	}

	// If no (non-prefetching or hydrating) defer triggers were provided, default to `idle`.
	hasConcreteTrigger := false
	for _, op := range deferOnOps {
		if deferOnOp, ok := op.(*ops_create.DeferOnOp); ok && deferOnOp.Modifier == ir.DeferOpModifierKindNone {
			hasConcreteTrigger = true
			break
		}
	}
	if !hasConcreteTrigger {
		for _, op := range deferWhenOps {
			if deferWhenOp, ok := op.(*ops_update.DeferWhenOp); ok && deferWhenOp.Modifier == ir.DeferOpModifierKindNone {
				hasConcreteTrigger = true
				break
			}
		}
	}

	if !hasConcreteTrigger {
		trigger := &ops_create.DeferIdleTrigger{Kind: ir.DeferTriggerKindIdle}
		deferOnOp := ops_create.NewDeferOnOp(
			deferXref,
			trigger,
			ir.DeferOpModifierKindNone,
			nil,
		)
		deferOnOps = append(deferOnOps, deferOnOp)
	}

	// Push all defer ops
	for _, op := range deferOnOps {
		unit.Create.Push(op)
	}
	for _, op := range deferWhenOps {
		unit.Update.Push(op)
	}
}

func ingestIcu(unit *compilation.ViewCompilationUnit, icu *render3.Icu) {
	if msg, ok := icu.I18n.(*i18n.Message); ok && IsSingleI18nIcu(icu.I18n) {
		xref := unit.Job.AllocateXrefId()
		icuPlaceholder := view_i18n.IcuFromI18nMessage(msg)
		if icuPlaceholder == nil {
			panic("ICU placeholder not found in i18n message")
		}
		unit.Create.Push(
			ops_create.NewIcuStartOp(xref, msg, icuPlaceholder.Name, icu.SourceSpan()),
		)

		// Combine vars and placeholders
		allPlaceholders := make(map[string]render3.Node)
		for k, v := range icu.Vars {
			allPlaceholders[k] = v
		}
		for k, v := range icu.Placeholders {
			allPlaceholders[k] = v
		}

		// Process each placeholder
		for placeholder, text := range allPlaceholders {
			placeholderPtr := &placeholder
			if boundText, ok := text.(*render3.BoundText); ok {
				ingestBoundText(unit, boundText, placeholderPtr)
			} else {
				if textNode, ok := text.(*render3.Text); ok {
					ingestText(unit, textNode, placeholderPtr)
				} else {
					panic(fmt.Sprintf("Unexpected node type in ICU placeholder: %T", text))
				}
			}
		}

		unit.Create.Push(ops_create.NewIcuEndOp(xref))
	} else {
		panic(fmt.Sprintf("Unhandled i18n metadata type for ICU: %T", icu.I18n))
	}
}

// getComputedForLoopVariableExpression gets an expression that represents a variable in an `@for` loop
func getComputedForLoopVariableExpression(
	variable *render3.Variable,
	indexName string,
	countName string,
) output.OutputExpression {
	switch variable.Value {
	case "$index":
		return expression.NewLexicalReadExpr(indexName)
	case "$count":
		return expression.NewLexicalReadExpr(countName)
	case "$first":
		indexExpr := expression.NewLexicalReadExpr(indexName)
		zeroExpr := output.NewLiteralExpr(0, nil, nil)
		return output.NewBinaryOperatorExpr(
			output.BinaryOperatorIdentical,
			indexExpr,
			zeroExpr,
			nil,
			nil,
		)
	case "$last":
		indexExpr := expression.NewLexicalReadExpr(indexName)
		countExpr := expression.NewLexicalReadExpr(countName)
		oneExpr := output.NewLiteralExpr(1, nil, nil)
		countMinusOne := output.NewBinaryOperatorExpr(
			output.BinaryOperatorMinus,
			countExpr,
			oneExpr,
			nil,
			nil,
		)
		return output.NewBinaryOperatorExpr(
			output.BinaryOperatorIdentical,
			indexExpr,
			countMinusOne,
			nil,
			nil,
		)
	case "$even":
		indexExpr := expression.NewLexicalReadExpr(indexName)
		twoExpr := output.NewLiteralExpr(2, nil, nil)
		zeroExpr := output.NewLiteralExpr(0, nil, nil)
		moduloExpr := output.NewBinaryOperatorExpr(
			output.BinaryOperatorModulo,
			indexExpr,
			twoExpr,
			nil,
			nil,
		)
		return output.NewBinaryOperatorExpr(
			output.BinaryOperatorIdentical,
			moduloExpr,
			zeroExpr,
			nil,
			nil,
		)
	case "$odd":
		indexExpr := expression.NewLexicalReadExpr(indexName)
		twoExpr := output.NewLiteralExpr(2, nil, nil)
		zeroExpr := output.NewLiteralExpr(0, nil, nil)
		moduloExpr := output.NewBinaryOperatorExpr(
			output.BinaryOperatorModulo,
			indexExpr,
			twoExpr,
			nil,
			nil,
		)
		return output.NewBinaryOperatorExpr(
			output.BinaryOperatorNotIdentical,
			moduloExpr,
			zeroExpr,
			nil,
			nil,
		)
	default:
		panic(fmt.Sprintf("AssertionError: unknown @for loop variable %s", variable.Value))
	}
}

// ingestForBlock ingests an `@for` block into the given ViewCompilationUnit
func ingestForBlock(unit *compilation.ViewCompilationUnit, forBlock *render3.ForLoopBlock) {
	repeaterView := unit.Job.AllocateView(unit.Xref)

	// We copy TemplateDefinitionBuilder's scheme of creating names for `$count` and `$index`
	// that are suffixed with special information, to disambiguate which level of nested loop
	// the below aliases refer to.
	indexName := fmt.Sprintf("ɵ$index_%d", repeaterView.Xref)
	countName := fmt.Sprintf("ɵ$count_%d", repeaterView.Xref)
	indexVarNames := make(map[string]bool)

	// Set all the context variables and aliases available in the repeater.
	if forBlock.Item != nil {
		repeaterView.ContextVariables[forBlock.Item.Name] = ir_variable.CTX_REF
	}

	for _, variable := range forBlock.ContextVariables {
		if variable.Value == "$index" {
			indexVarNames[variable.Name] = true
		}
		if variable.Name == "$index" {
			repeaterView.ContextVariables["$index"] = ir_variable.CTX_REF
			repeaterView.ContextVariables[indexName] = ir_variable.CTX_REF
		} else if variable.Name == "$count" {
			repeaterView.ContextVariables["$count"] = ir_variable.CTX_REF
			repeaterView.ContextVariables[countName] = ir_variable.CTX_REF
		} else {
			alias := ir_variable.NewAliasVariable(
				variable.Name,
				getComputedForLoopVariableExpression(variable, indexName, countName),
			)
			repeaterView.Aliases[*alias] = true
		}
	}

	sourceSpan := convertSourceSpan(forBlock.TrackBy.Span(), forBlock.SourceSpan())
	track := convertAst(forBlock.TrackBy.AST, unit.GetJob(), sourceSpan)

	ingestNodes(repeaterView, forBlock.Children)

	var emptyView *compilation.ViewCompilationUnit
	var emptyTagName *string
	if forBlock.Empty != nil {
		emptyView = unit.Job.AllocateView(unit.Xref)
		ingestNodes(emptyView, forBlock.Empty.Children)
		tagName := ingestControlFlowInsertionPoint(unit, emptyView.Xref, forBlock.Empty)
		if tagName != "" {
			emptyTagName = &tagName
		}
	}

	varNames := ops_create.RepeaterVarNames{
		DollarIndex:    indexVarNames,
		DollarImplicit: "",
	}
	if forBlock.Item != nil {
		varNames.DollarImplicit = forBlock.Item.Name
	}

	var i18nPlaceholder interface{} // *i18n.BlockPlaceholder
	if forBlock.I18n != nil {
		if blockPlaceholder, ok := forBlock.I18n.(*i18n.BlockPlaceholder); ok {
			i18nPlaceholder = blockPlaceholder
		} else {
			panic("AssertionError: Unhandled i18n metadata type for @for")
		}
	}

	var emptyI18nPlaceholder interface{} // *i18n.BlockPlaceholder
	if forBlock.Empty != nil && forBlock.Empty.I18n != nil {
		if blockPlaceholder, ok := forBlock.Empty.I18n.(*i18n.BlockPlaceholder); ok {
			emptyI18nPlaceholder = blockPlaceholder
		} else {
			panic("AssertionError: Unhandled i18n metadata type for @empty")
		}
	}

	tagName := ingestControlFlowInsertionPoint(unit, repeaterView.Xref, forBlock)
	var tagNamePtr *string
	if tagName != "" {
		tagNamePtr = &tagName
	}

	emptyViewXref := ir_operations.XrefId(0)
	if emptyView != nil {
		emptyViewXref = emptyView.Xref
	}

	repeaterCreate := ops_create.NewRepeaterCreateOp(
		repeaterView.Xref,
		emptyViewXref,
		tagNamePtr,
		track,
		varNames,
		emptyTagName,
		i18nPlaceholder,
		emptyI18nPlaceholder,
		forBlock.StartSourceSpan,
		forBlock.SourceSpan(),
	)
	unit.Create.Push(repeaterCreate)

	expression := convertAst(
		forBlock.Expression.AST,
		unit.GetJob(),
		convertSourceSpan(forBlock.Expression.Span(), forBlock.SourceSpan()),
	)
	repeater := ops_update.NewRepeaterOp(
		repeaterCreate.GetXref(),
		expression,
	)
	unit.Update.Push(repeater)
}

func ingestLetDeclaration(unit *compilation.ViewCompilationUnit, node *render3.LetDeclaration) {
	target := unit.Job.AllocateXrefId()

	unit.Create.Push(
		ops_create.NewDeclareLetOp(target, node.Name, node.SourceSpan()),
	)
	unit.Update.Push(
		ops_update.NewStoreLetOp(
			target,
			node.Name,
			convertAst(node.Value, unit.GetJob(), node.ValueSpan),
			node.SourceSpan(),
		),
	)
}

// makeListenerHandlerOps creates handler operations for a listener
func makeListenerHandlerOps(
	unit compilation.CompilationUnit,
	handler expression_parser.AST,
	handlerSpan *util.ParseSourceSpan,
) []ir_operations.UpdateOp {
	handler = astOf(handler)
	handlerOps := make([]ir_operations.UpdateOp, 0)

	// Handle Chain expressions
	var handlerExprs []expression_parser.AST
	if chain, ok := handler.(*expression_parser.Chain); ok {
		handlerExprs = chain.Expressions
	} else {
		handlerExprs = []expression_parser.AST{handler}
	}

	if len(handlerExprs) == 0 {
		panic("Expected listener to have non-empty expression list")
	}

	// Convert expressions
	expressions := make([]output.OutputExpression, len(handlerExprs))
	for i, expr := range handlerExprs {
		expressions[i] = convertAst(expr, unit.GetJob(), handlerSpan)
	}

	// Pop the last expression as return value
	returnExpr := expressions[len(expressions)-1]
	expressions = expressions[:len(expressions)-1]

	// Add expression statements for all but the last
	for _, expr := range expressions {
		handlerOps = append(handlerOps, shared.NewStatementOp(
			output.NewExpressionStatement(expr, expr.GetSourceSpan(), nil),
		))
	}

	// Add return statement
	handlerOps = append(handlerOps, shared.NewStatementOp(
		output.NewReturnStatement(returnExpr, returnExpr.GetSourceSpan(), nil),
	))

	return handlerOps
}

// makeTwoWayListenerHandlerOps creates handler operations for a two-way listener
func makeTwoWayListenerHandlerOps(
	unit compilation.CompilationUnit,
	handler expression_parser.AST,
	handlerSpan *util.ParseSourceSpan,
) []ir_operations.UpdateOp {
	handler = astOf(handler)
	handlerOps := make([]ir_operations.UpdateOp, 0)

	// Handle Chain expressions
	if chain, ok := handler.(*expression_parser.Chain); ok {
		if len(chain.Expressions) == 1 {
			handler = chain.Expressions[0]
		} else {
			// This is validated during parsing already, but we do it here just in case
			panic("Expected two-way listener to have a single expression")
		}
	}

	handlerExpr := convertAst(handler, unit.GetJob(), handlerSpan)
	eventReference := expression.NewLexicalReadExpr("$event")
	twoWaySetExpr := expression.NewTwoWayBindingSetExpr(handlerExpr, eventReference)

	handlerOps = append(handlerOps, shared.NewStatementOp(
		output.NewExpressionStatement(twoWaySetExpr, nil, nil),
	))
	handlerOps = append(handlerOps, shared.NewStatementOp(
		output.NewReturnStatement(eventReference, nil, nil),
	))

	return handlerOps
}

// createTemplateBinding creates a binding op for a template
func createTemplateBinding(
	view *compilation.ViewCompilationUnit,
	xref ir_operations.XrefId,
	bindingType expression_parser.BindingType,
	name string,
	value interface{}, // expression_parser.AST | string
	unit *string,
	securityContext interface{}, // core.SecurityContext
	isStructuralTemplateAttribute bool,
	templateKind *ir.TemplateKind,
	i18nMessage *i18n.Message,
	sourceSpan *util.ParseSourceSpan,
) ir_operations.UpdateOp { // ir.BindingOp | ir.ExtractedAttributeOp | nil
	isTextBinding := false
	var strValue string
	if s, ok := value.(string); ok {
		isTextBinding = true
		strValue = s
	}

	// If this is a structural template, then several kinds of bindings should not result in an
	// update instruction.
	if templateKind != nil && *templateKind == ir.TemplateKindStructural {
		if !isStructuralTemplateAttribute {
			switch bindingType {
			case expression_parser.BindingTypeProperty,
				expression_parser.BindingTypeClass,
				expression_parser.BindingTypeStyle:
				// Because this binding doesn't really target the ng-template, it must be a binding on an
				// inner node of a structural template. We can't skip it entirely, because we still need
				// it on the ng-template's consts (e.g. for the purposes of directive matching). However,
				// we should not generate an update instruction for it.
				return ops_create.NewExtractedAttributeOp(
					xref,
					ir.BindingKindProperty,
					nil,
					name,
					nil,
					0,
					i18nMessage,
					securityContext,
				)
			case expression_parser.BindingTypeTwoWay:
				return ops_create.NewExtractedAttributeOp(
					xref,
					ir.BindingKindTwoWayProperty,
					nil,
					name,
					nil,
					0,
					i18nMessage,
					securityContext,
				)
			}
		}

		if !isTextBinding &&
			(bindingType == expression_parser.BindingTypeAttribute ||
				bindingType == expression_parser.BindingTypeLegacyAnimation ||
				bindingType == expression_parser.BindingTypeAnimation) {
			// Again, this binding doesn't really target the ng-template; it actually targets the element
			// inside the structural template. In the case of non-text attribute or animation bindings,
			// the binding doesn't even show up on the ng-template const array, so we just skip it
			// entirely.
			return nil
		}
	}

	bindingKind := BINDING_KINDS[bindingType]

	if templateKind != nil && *templateKind == ir.TemplateKindNgTemplate {
		// We know we are dealing with bindings directly on an explicit ng-template.
		// Static attribute bindings should be collected into the const array as k/v pairs. Property
		// bindings should result in a `property` instruction, and `AttributeMarker.Bindings` const
		// entries.
		//
		// The difficulty is with dynamic attribute, style, and class bindings. These don't really make
		// sense on an `ng-template` and should probably be parser errors. However,
		// TemplateDefinitionBuilder generates `property` instructions for them, and so we do that as
		// well.
		if bindingType == expression_parser.BindingTypeClass ||
			bindingType == expression_parser.BindingTypeStyle ||
			(bindingType == expression_parser.BindingTypeAttribute && !isTextBinding) {
			// TODO: These cases should be parse errors.
			bindingKind = ir.BindingKindProperty
		}
	}

	var expression interface{} // output.OutputExpression | *ops.Interpolation
	if isTextBinding {
		expression = convertAstWithInterpolation(view.GetJob(), strValue, i18nMessage, sourceSpan)
	} else {
		ast, ok := value.(expression_parser.AST)
		if !ok {
			panic(fmt.Sprintf("Expected AST for non-text binding, got %T", value))
		}
		expression = convertAstWithInterpolation(view.GetJob(), ast, i18nMessage, sourceSpan)
	}

	return ops_update.NewBindingOp(
		xref,
		bindingKind,
		name,
		expression,
		unit,
		securityContext,
		isTextBinding,
		isStructuralTemplateAttribute,
		templateKind,
		i18nMessage,
		sourceSpan,
	)
}

// ingestElementBindings processes all of the bindings on an element in the template AST
// and converts them to their IR representation
func ingestElementBindings(
	unit *compilation.ViewCompilationUnit,
	op *ops_create.ElementStartOp,
	element *render3.Element,
) {
	var bindings []ir_operations.UpdateOp // []ir.BindingOp | ir.ExtractedAttributeOp
	i18nAttributeBindingNames := make(map[string]bool)

	for _, attr := range element.Attributes {
		// Attribute literal bindings, such as `attr.foo="bar"`.
		securityContext := domSchema.SecurityContext(element.Name, attr.Name, true)
		binding := ops_update.NewBindingOp(
			op.Xref,
			ir.BindingKindAttribute,
			attr.Name,
			convertAstWithInterpolation(unit.GetJob(), attr.Value, attr.I18n, attr.SourceSpan()),
			nil,
			securityContext,
			true,
			false,
			nil,
			asMessage(attr.I18n),
			attr.SourceSpan(),
		)
		bindings = append(bindings, binding)
		if attr.I18n != nil {
			i18nAttributeBindingNames[attr.Name] = true
		}
	}

	for _, input := range element.Inputs {
		if i18nAttributeBindingNames[input.Name] {
			// TODO: Use proper logging instead of panic
			panic(fmt.Sprintf(
				"On component %s, the binding %s is both an i18n attribute and a property. You may want to remove the property binding. This will become a compilation error in future versions of Angular.",
				unit.Job.ComponentName,
				input.Name,
			))
		}
		// All dynamic bindings (both attribute and property bindings).
		binding := ops_update.NewBindingOp(
			op.Xref,
			BINDING_KINDS[input.Type],
			input.Name,
			convertAstWithInterpolation(unit.GetJob(), astOf(input.Value), input.I18n, input.SourceSpan()),
			input.Unit,
			input.SecurityContext,
			false,
			false,
			nil,
			asMessage(input.I18n),
			input.SourceSpan(),
		)
		bindings = append(bindings, binding)

		// If the input name is 'field', this could be a form control binding which requires a
		// `ControlCreateOp` to properly initialize.
		if input.Type == expression_parser.BindingTypeProperty && input.Name == "field" {
			// TODO: Implement ControlCreateOp
			// unit.Create.Push(ops.NewControlCreateOp(input.SourceSpan))
		}
	}

	// Separate bindings into extracted attributes and regular bindings
	for _, binding := range bindings {
		if binding == nil {
			continue
		}
		if extractedAttr, ok := binding.(*ops_create.ExtractedAttributeOp); ok {
			unit.Create.Push(extractedAttr)
		} else if bindingOp, ok := binding.(*ops_update.BindingOp); ok {
			unit.Update.Push(bindingOp)
		}
	}

	for _, output := range element.Outputs {
		if output.Type == expression_parser.ParsedEventTypeLegacyAnimation && output.Phase == nil {
			panic("Animation listener should have a phase")
		}

		if output.Type == expression_parser.ParsedEventTypeTwoWay {
			unit.Create.Push(
				ops_create.NewTwoWayListenerOp(
					op.Xref,
					op.Handle,
					output.Name,
					op.Tag,
					makeTwoWayListenerHandlerOps(unit, output.Handler, output.HandlerSpan),
					output.SourceSpan(),
				),
			)
		} else if output.Type == expression_parser.ParsedEventTypeAnimation {
			animationKind := ir.AnimationKindEnter
			if !strings.HasSuffix(output.Name, "enter") {
				animationKind = ir.AnimationKindLeave
			}
			unit.Create.Push(
				ops_create.NewAnimationListenerOp(
					op.Xref,
					op.Handle,
					output.Name,
					op.Tag,
					makeListenerHandlerOps(unit, output.Handler, output.HandlerSpan),
					animationKind,
					output.Target,
					false,
					output.SourceSpan(),
				),
			)
		} else {
			unit.Create.Push(
				ops_create.NewListenerOp(
					op.Xref,
					op.Handle,
					output.Name,
					op.Tag,
					makeListenerHandlerOps(unit, output.Handler, output.HandlerSpan),
					output.Phase,
					output.Target,
					false,
					output.SourceSpan(),
				),
			)
		}
	}

	// If any of the bindings on this element have an i18n message, then an i18n attrs configuration
	// op is also required.
	hasI18nMessage := false
	for _, binding := range bindings {
		if bindingOp, ok := binding.(*ops_update.BindingOp); ok && bindingOp.I18nMessage != nil {
			hasI18nMessage = true
			break
		}
	}
	if hasI18nMessage {
		unit.Create.Push(
			ops_create.NewI18nAttributesOp(
				unit.Job.AllocateXrefId(),
				ir.NewSlotHandle(),
				op.Xref,
			),
		)
	}
}

// ingestTemplateBindings processes all of the bindings on a template in the template AST
// and converts them to their IR representation
func ingestTemplateBindings(
	unit *compilation.ViewCompilationUnit,
	op *ops_create.TemplateOp,
	template *render3.Template,
	templateKind ir.TemplateKind,
) {
	var bindings []ir_operations.UpdateOp // []ir.BindingOp | ir.ExtractedAttributeOp

	templateKindPtr := &templateKind

	for _, attr := range template.TemplateAttrs {
		if textAttr, ok := attr.(*render3.TextAttribute); ok {
			securityContext := domSchema.SecurityContext(NG_TEMPLATE_TAG_NAME, textAttr.Name, true)
			binding := createTemplateBinding(
				unit,
				op.Xref,
				expression_parser.BindingTypeAttribute,
				textAttr.Name,
				textAttr.Value,
				nil,
				securityContext,
				true,
				templateKindPtr,
				asMessage(textAttr.I18n),
				textAttr.SourceSpan(),
			)
			if binding != nil {
				bindings = append(bindings, binding)
			}
		} else if boundAttr, ok := attr.(*render3.BoundAttribute); ok {
			binding := createTemplateBinding(
				unit,
				op.Xref,
				boundAttr.Type,
				boundAttr.Name,
				astOf(boundAttr.Value),
				boundAttr.Unit,
				boundAttr.SecurityContext,
				true,
				templateKindPtr,
				asMessage(boundAttr.I18n),
				boundAttr.SourceSpan(),
			)
			if binding != nil {
				bindings = append(bindings, binding)
			}
		}
	}

	for _, attr := range template.Attributes {
		// Attribute literal bindings, such as `attr.foo="bar"`.
		securityContext := domSchema.SecurityContext(NG_TEMPLATE_TAG_NAME, attr.Name, true)
		binding := createTemplateBinding(
			unit,
			op.Xref,
			expression_parser.BindingTypeAttribute,
			attr.Name,
			attr.Value,
			nil,
			securityContext,
			false,
			templateKindPtr,
			asMessage(attr.I18n),
			attr.SourceSpan(),
		)
		if binding != nil {
			bindings = append(bindings, binding)
		}
	}

	for _, input := range template.Inputs {
		// Dynamic bindings (both attribute and property bindings).
		binding := createTemplateBinding(
			unit,
			op.Xref,
			input.Type,
			input.Name,
			astOf(input.Value),
			input.Unit,
			input.SecurityContext,
			false,
			templateKindPtr,
			asMessage(input.I18n),
			input.SourceSpan(),
		)
		if binding != nil {
			bindings = append(bindings, binding)
		}
	}

	// Separate bindings into extracted attributes and regular bindings
	for _, binding := range bindings {
		if binding == nil {
			continue
		}
		if extractedAttr, ok := binding.(*ops_create.ExtractedAttributeOp); ok {
			unit.Create.Push(extractedAttr)
		} else if bindingOp, ok := binding.(*ops_update.BindingOp); ok {
			unit.Update.Push(bindingOp)
		}
	}

	for _, output := range template.Outputs {
		if output.Type == expression_parser.ParsedEventTypeLegacyAnimation && output.Phase == nil {
			panic("Animation listener should have a phase")
		}

		if templateKind == ir.TemplateKindNgTemplate {
			if output.Type == expression_parser.ParsedEventTypeTwoWay {
				unit.Create.Push(
					ops_create.NewTwoWayListenerOp(
						op.Xref,
						op.Handle,
						output.Name,
						op.Tag,
						makeTwoWayListenerHandlerOps(unit, output.Handler, output.HandlerSpan),
						output.SourceSpan(),
					),
				)
			} else {
				unit.Create.Push(
					ops_create.NewListenerOp(
						op.Xref,
						op.Handle,
						output.Name,
						op.Tag,
						makeListenerHandlerOps(unit, output.Handler, output.HandlerSpan),
						output.Phase,
						output.Target,
						false,
						output.SourceSpan(),
					),
				)
			}
		} else {
			// Structural templates don't support outputs
			// TODO: This might need to be handled differently
		}
	}
}

func ingestReferences(op interface{}, element interface{}) {
	var references []*render3.Reference

	// Extract references from element or template
	switch e := element.(type) {
	case *render3.Element:
		references = e.References
	case *render3.Template:
		references = e.References
	default:
		panic(fmt.Sprintf("Unexpected element type: %T", element))
	}

	// Get LocalRefs from op and convert to slice if needed
	switch o := op.(type) {
	case *ops_create.ElementStartOp:
		refs, ok := o.LocalRefs.([]ops_create.LocalRef)
		if !ok {
			refs = make([]ops_create.LocalRef, 0)
		}
		for _, ref := range references {
			refs = append(refs, ops_create.LocalRef{
				Name:   ref.Name,
				Target: ref.Value,
			})
		}
		o.LocalRefs = refs
	case *ops_create.TemplateOp:
		refs, ok := o.LocalRefs.([]ops_create.LocalRef)
		if !ok {
			refs = make([]ops_create.LocalRef, 0)
		}
		for _, ref := range references {
			refs = append(refs, ops_create.LocalRef{
				Name:   ref.Name,
				Target: ref.Value,
			})
		}
		o.LocalRefs = refs
	default:
		panic(fmt.Sprintf("Unexpected op type: %T", op))
	}
}

// Helper functions

// astOf extracts the AST from an ASTWithSource, or returns the AST itself
func astOf(ast expression_parser.AST) expression_parser.AST {
	if astWithSource, ok := ast.(*expression_parser.ASTWithSource); ok {
		return astWithSource.AST
	}
	return ast
}

// asMessage ensures that the i18nMeta, if provided, is an i18n.Message
func asMessage(i18nMeta interface{}) *i18n.Message {
	if i18nMeta == nil {
		return nil
	}
	msg, ok := i18nMeta.(*i18n.Message)
	if !ok {
		panic(fmt.Sprintf("Expected i18n meta to be a Message, but got: %T", i18nMeta))
	}
	return msg
}

// convertSourceSpan creates an absolute ParseSourceSpan from the relative ParseSpan
func convertSourceSpan(
	span *expression_parser.ParseSpan,
	baseSourceSpan *util.ParseSourceSpan,
) *util.ParseSourceSpan {
	if baseSourceSpan == nil {
		return nil
	}
	start := baseSourceSpan.Start.MoveBy(span.Start)
	end := baseSourceSpan.Start.MoveBy(span.End)
	fullStart := baseSourceSpan.FullStart.MoveBy(span.Start)
	return util.NewParseSourceSpan(start, end, fullStart, nil)
}

// convertAst converts a template AST expression into an output AST expression
func convertAst(
	ast expression_parser.AST,
	job *compilation.CompilationJob,
	baseSourceSpan *util.ParseSourceSpan,
) output.OutputExpression {
	// Handle ASTWithSource wrapper
	if astWithSource, ok := ast.(*expression_parser.ASTWithSource); ok {
		return convertAst(astWithSource.AST, job, baseSourceSpan)
	}

	// Handle PropertyRead
	if propRead, ok := ast.(*expression_parser.PropertyRead); ok {
		// Whether this is an implicit receiver, *excluding* explicit reads of `this`
		_, isImplicitReceiver := propRead.Receiver.(*expression_parser.ImplicitReceiver)
		_, isThisReceiver := propRead.Receiver.(*expression_parser.ThisReceiver)
		isImplicitReceiver = isImplicitReceiver && !isThisReceiver

		if isImplicitReceiver {
			return expression.NewLexicalReadExpr(propRead.Name)
		} else {
			return output.NewReadPropExpr(
				convertAst(propRead.Receiver, job, baseSourceSpan),
				propRead.Name,
				nil,
				convertSourceSpan(propRead.Span(), baseSourceSpan),
			)
		}
	}

	// Handle Call
	if call, ok := ast.(*expression_parser.Call); ok {
		if _, isImplicitReceiver := call.Receiver.(*expression_parser.ImplicitReceiver); isImplicitReceiver {
			panic("Unexpected ImplicitReceiver")
		}
		args := make([]output.OutputExpression, len(call.Args))
		for i, arg := range call.Args {
			args[i] = convertAst(arg, job, baseSourceSpan)
		}
		return output.NewInvokeFunctionExpr(
			convertAst(call.Receiver, job, baseSourceSpan),
			args,
			nil, // typ
			convertSourceSpan(call.Span(), baseSourceSpan),
			false, // pure
		)
	}

	// Handle LiteralPrimitive
	if literal, ok := ast.(*expression_parser.LiteralPrimitive); ok {
		return output.NewLiteralExpr(
			literal.Value,
			nil,
			convertSourceSpan(literal.Span(), baseSourceSpan),
		)
	}

	// Handle Unary
	if unary, ok := ast.(*expression_parser.Unary); ok {
		var op output.UnaryOperator
		switch unary.Operator {
		case "+":
			op = output.UnaryOperatorPlus
		case "-":
			op = output.UnaryOperatorMinus
		default:
			panic(fmt.Sprintf("AssertionError: unknown unary operator %s", unary.Operator))
		}
		return output.NewUnaryOperatorExpr(
			op,
			convertAst(unary.Expr, job, baseSourceSpan),
			nil,
			convertSourceSpan(unary.Span(), baseSourceSpan),
			true, // parens - default value from TypeScript
		)
	}

	// Handle Binary
	if binary, ok := ast.(*expression_parser.Binary); ok {
		operator, ok := pipeline_convension.BinaryOperators[binary.Operation]
		if !ok {
			panic(fmt.Sprintf("AssertionError: unknown binary operator %s", binary.Operation))
		}
		return output.NewBinaryOperatorExpr(
			operator,
			convertAst(binary.Left, job, baseSourceSpan),
			convertAst(binary.Right, job, baseSourceSpan),
			nil,
			convertSourceSpan(binary.Span(), baseSourceSpan),
		)
	}

	// Handle ThisReceiver
	if thisReceiver, ok := ast.(*expression_parser.ThisReceiver); ok {
		_ = thisReceiver
		return expression.NewContextExpr(job.GetRoot().GetXref())
	}

	// Handle KeyedRead
	if keyedRead, ok := ast.(*expression_parser.KeyedRead); ok {
		return output.NewReadKeyExpr(
			convertAst(keyedRead.Receiver, job, baseSourceSpan),
			convertAst(keyedRead.Key, job, baseSourceSpan),
			nil,
			convertSourceSpan(keyedRead.Span(), baseSourceSpan),
		)
	}

	// Handle Chain
	if _, ok := ast.(*expression_parser.Chain); ok {
		panic("AssertionError: Chain in unknown context")
	}

	// Handle LiteralMap
	if literalMap, ok := ast.(*expression_parser.LiteralMap); ok {
		entries := make([]*output.LiteralMapEntry, len(literalMap.Keys))
		for i, key := range literalMap.Keys {
			entries[i] = output.NewLiteralMapEntry(
				key.Key,
				convertAst(literalMap.Values[i], job, baseSourceSpan),
				key.Quoted,
			)
		}
		return output.NewLiteralMapExpr(
			entries,
			nil,
			convertSourceSpan(literalMap.Span(), baseSourceSpan),
		)
	}

	// Handle LiteralArray
	if literalArray, ok := ast.(*expression_parser.LiteralArray); ok {
		entries := make([]output.OutputExpression, len(literalArray.Expressions))
		for i, expr := range literalArray.Expressions {
			entries[i] = convertAst(expr, job, baseSourceSpan)
		}
		return output.NewLiteralArrayExpr(entries, nil, nil)
	}

	// Handle Conditional
	if conditional, ok := ast.(*expression_parser.Conditional); ok {
		return output.NewConditionalExpr(
			convertAst(conditional.Condition, job, baseSourceSpan),
			convertAst(conditional.TrueExp, job, baseSourceSpan),
			convertAst(conditional.FalseExp, job, baseSourceSpan),
			nil,
			convertSourceSpan(conditional.Span(), baseSourceSpan),
		)
	}

	// Handle NonNullAssert
	if nonNullAssert, ok := ast.(*expression_parser.NonNullAssert); ok {
		// A non-null assertion shouldn't impact generated instructions, so we can just drop it
		return convertAst(nonNullAssert.Expression, job, baseSourceSpan)
	}

	// Handle BindingPipe
	if pipe, ok := ast.(*expression_parser.BindingPipe); ok {
		args := make([]output.OutputExpression, 0, 1+len(pipe.Args))
		args = append(args, convertAst(pipe.Exp, job, baseSourceSpan))
		for _, arg := range pipe.Args {
			args = append(args, convertAst(arg, job, baseSourceSpan))
		}
		return expression.NewPipeBindingExpr(
			job.AllocateXrefId(),
			ir.NewSlotHandle(),
			pipe.Name,
			args,
		)
	}

	// Handle SafeKeyedRead
	if safeKeyedRead, ok := ast.(*expression_parser.SafeKeyedRead); ok {
		return expression.NewSafeKeyedReadExpr(
			convertAst(safeKeyedRead.Receiver, job, baseSourceSpan),
			convertAst(safeKeyedRead.Key, job, baseSourceSpan),
			convertSourceSpan(safeKeyedRead.Span(), baseSourceSpan),
		)
	}

	// Handle SafePropertyRead
	if safePropRead, ok := ast.(*expression_parser.SafePropertyRead); ok {
		return expression.NewSafePropertyReadExpr(
			convertAst(safePropRead.Receiver, job, baseSourceSpan),
			safePropRead.Name,
		)
	}

	// Handle SafeCall
	if safeCall, ok := ast.(*expression_parser.SafeCall); ok {
		args := make([]output.OutputExpression, len(safeCall.Args))
		for i, arg := range safeCall.Args {
			args[i] = convertAst(arg, job, baseSourceSpan)
		}
		return expression.NewSafeInvokeFunctionExpr(
			convertAst(safeCall.Receiver, job, baseSourceSpan),
			args,
		)
	}

	// Handle EmptyExpr
	if emptyExpr, ok := ast.(*expression_parser.EmptyExpr); ok {
		return expression.NewEmptyExpr(convertSourceSpan(emptyExpr.Span(), baseSourceSpan))
	}

	// Handle PrefixNot
	if prefixNot, ok := ast.(*expression_parser.PrefixNot); ok {
		return output.NewNotExpr(
			convertAst(prefixNot.Expression, job, baseSourceSpan),
			convertSourceSpan(prefixNot.Span(), baseSourceSpan),
		)
	}

	// Handle TypeofExpression
	if typeofExpr, ok := ast.(*expression_parser.TypeofExpression); ok {
		return output.NewTypeofExpr(
			convertAst(typeofExpr.Expression, job, baseSourceSpan),
			nil,
			convertSourceSpan(typeofExpr.Span(), baseSourceSpan),
		)
	}

	// Handle VoidExpression
	if voidExpr, ok := ast.(*expression_parser.VoidExpression); ok {
		return output.NewVoidExpr(
			convertAst(voidExpr.Expression, job, baseSourceSpan),
			nil,
			convertSourceSpan(voidExpr.Span(), baseSourceSpan),
		)
	}

	// Handle TemplateLiteral
	if templateLiteral, ok := ast.(*expression_parser.TemplateLiteral); ok {
		return convertTemplateLiteral(templateLiteral, job, baseSourceSpan)
	}

	// Handle TaggedTemplateLiteral
	if taggedTemplateLiteral, ok := ast.(*expression_parser.TaggedTemplateLiteral); ok {
		templateExpr := convertTemplateLiteral(taggedTemplateLiteral.Template, job, baseSourceSpan)
		template, ok := templateExpr.(*output.TemplateLiteralExpr)
		if !ok {
			panic(fmt.Sprintf("convertTemplateLiteral returned %T, expected *output.TemplateLiteralExpr", templateExpr))
		}
		return output.NewTaggedTemplateLiteralExpr(
			convertAst(taggedTemplateLiteral.Tag, job, baseSourceSpan),
			template,
			nil,
			convertSourceSpan(taggedTemplateLiteral.Span(), baseSourceSpan),
		)
	}

	// Handle ParenthesizedExpression
	if parenthesizedExpr, ok := ast.(*expression_parser.ParenthesizedExpression); ok {
		return output.NewParenthesizedExpr(
			convertAst(parenthesizedExpr.Expression, job, baseSourceSpan),
			nil,
			convertSourceSpan(parenthesizedExpr.Span(), baseSourceSpan),
		)
	}

	// Handle RegularExpressionLiteral
	if regexLiteral, ok := ast.(*expression_parser.RegularExpressionLiteral); ok {
		return output.NewRegularExpressionLiteralExpr(
			regexLiteral.Body,
			regexLiteral.Flags,
			baseSourceSpan,
		)
	}

	// Unhandled expression type
	fileURL := ""
	if baseSourceSpan != nil && baseSourceSpan.Start != nil && baseSourceSpan.Start.File != nil {
		fileURL = baseSourceSpan.Start.File.URL
	}
	panic(fmt.Sprintf(
		"Unhandled expression type \"%T\" in file \"%s\"",
		ast,
		fileURL,
	))
}

// convertTemplateLiteral converts a template literal AST to an output expression
func convertTemplateLiteral(
	ast *expression_parser.TemplateLiteral,
	job *compilation.CompilationJob,
	baseSourceSpan *util.ParseSourceSpan,
) output.OutputExpression {
	elements := make([]*output.TemplateLiteralElementExpr, len(ast.Elements))
	for i, el := range ast.Elements {
		elements[i] = output.NewTemplateLiteralElementExpr(
			el.Text,
			convertSourceSpan(el.Span(), baseSourceSpan),
			el.Text, // rawText - TODO: implement proper escaping like TypeScript
		)
	}
	expressions := make([]output.OutputExpression, len(ast.Expressions))
	for i, expr := range ast.Expressions {
		expressions[i] = convertAst(expr, job, baseSourceSpan)
	}
	return output.NewTemplateLiteralExpr(
		elements,
		expressions,
		convertSourceSpan(ast.Span(), baseSourceSpan),
	)
}

// convertAstWithInterpolation converts an AST or string to an output expression or interpolation
func convertAstWithInterpolation(
	job *compilation.CompilationJob,
	value interface{},
	i18nMeta interface{},
	sourceSpan *util.ParseSourceSpan,
) interface{} { // output.OutputExpression | *ops.Interpolation
	var expression interface{}

	if interpolation, ok := value.(*expression_parser.Interpolation); ok {
		placeholders := make([]string, 0)
		if msg := asMessage(i18nMeta); msg != nil {
			for key := range msg.Placeholders {
				placeholders = append(placeholders, key)
			}
		}
		expressions := make([]output.OutputExpression, len(interpolation.Expressions))
		for i, expr := range interpolation.Expressions {
			expressions[i] = convertAst(expr, job, sourceSpan)
		}
		interp, err := ops_update.NewInterpolation(interpolation.Strings, expressions, placeholders)
		if err != nil {
			panic(err)
		}
		expression = interp
	} else if ast, ok := value.(expression_parser.AST); ok {
		expression = convertAst(ast, job, sourceSpan)
	} else if str, ok := value.(string); ok {
		expression = output.NewLiteralExpr(str, nil, nil)
	} else {
		panic(fmt.Sprintf("Unexpected value type: %T", value))
	}

	return expression
}

// BINDING_KINDS maps BindingType to BindingKind
var BINDING_KINDS = map[expression_parser.BindingType]ir.BindingKind{
	expression_parser.BindingTypeProperty:        ir.BindingKindProperty,
	expression_parser.BindingTypeTwoWay:          ir.BindingKindTwoWayProperty,
	expression_parser.BindingTypeAttribute:       ir.BindingKindAttribute,
	expression_parser.BindingTypeClass:           ir.BindingKindClassName,
	expression_parser.BindingTypeStyle:           ir.BindingKindStyleProperty,
	expression_parser.BindingTypeLegacyAnimation: ir.BindingKindLegacyAnimation,
	expression_parser.BindingTypeAnimation:       ir.BindingKindAnimation,
}

func isPlainTemplate(tmpl *render3.Template) bool {
	if tmpl.TagName == nil {
		return false
	}
	_, name := ml_parser.SplitNsName(*tmpl.TagName, false)
	return name == NG_TEMPLATE_TAG_NAME
}
