package partial

import (
	"errors"
	"strings"

	"ngc-go/packages/compiler/src/core"
	"ngc-go/packages/compiler/src/output"
	"ngc-go/packages/compiler/src/render3"
	r3_identifiers "ngc-go/packages/compiler/src/render3/r3_identifiers"
	"ngc-go/packages/compiler/src/render3/view"
	"ngc-go/packages/compiler/src/util"
)

// DeclareComponentTemplateInfo contains information about a component template
type DeclareComponentTemplateInfo struct {
	// Content is the string contents of the template.
	//
	// This is the "logical" template string, after expansion of any escaped characters (for inline
	// templates). This may differ from the actual template bytes as they appear in the .ts file.
	Content string

	// SourceUrl is a full path to the file which contains the template.
	//
	// This can be either the original .ts file if the template is inline, or the .html file if an
	// external file was used.
	SourceUrl string

	// IsInline indicates whether the template was inline (using `template`) or external (using `templateUrl`).
	IsInline bool

	// InlineTemplateLiteralExpression is the literal expression if the template was defined inline by a direct string literal.
	// Otherwise `nil`, if the template was not defined inline or was not a literal.
	InlineTemplateLiteralExpression output.OutputExpression
}

// CompileDeclareComponentFromMetadata compiles a component declaration defined by the `R3ComponentMetadata`.
func CompileDeclareComponentFromMetadata(
	meta *view.R3ComponentMetadata,
	template *view.ParsedTemplate,
	additionalTemplateInfo DeclareComponentTemplateInfo,
) render3.R3CompiledExpression {
	definitionMap := CreateComponentDefinitionMap(meta, template, additionalTemplateInfo)

	expression := output.NewInvokeFunctionExpr(
		output.NewExternalExpr(r3_identifiers.DeclareComponent, nil, nil, nil),
		[]output.OutputExpression{definitionMap.ToLiteralMap()},
		nil,
		nil,
		false,
	)
	typ := createComponentType(meta)

	return render3.R3CompiledExpression{
		Expression: expression,
		Type:       typ,
		Statements: []output.OutputStatement{},
	}
}

// CreateComponentDefinitionMap gathers the declaration fields for a component into a `DefinitionMap`.
func CreateComponentDefinitionMap(
	meta *view.R3ComponentMetadata,
	template *view.ParsedTemplate,
	templateInfo DeclareComponentTemplateInfo,
) *view.DefinitionMap {
	definitionMap := CreateDirectiveDefinitionMap(&meta.R3DirectiveMetadata)
	blockVisitor := NewBlockPresenceVisitor()
	render3.VisitAll(blockVisitor, template.Nodes)

	definitionMap.Set("template", getTemplateExpression(template, templateInfo))

	if templateInfo.IsInline {
		definitionMap.Set("isInline", output.NewLiteralExpr(true, output.InferredType, nil))
	}

	// Set the minVersion to 17.0.0 if the component is using at least one block in its template.
	// We don't do this for templates without blocks, in order to preserve backwards compatibility.
	if blockVisitor.HasBlocks {
		definitionMap.Set("minVersion", output.NewLiteralExpr("17.0.0", output.InferredType, nil))
	}

	definitionMap.Set("styles", ToOptionalLiteralArray(meta.Styles, func(value string) output.OutputExpression {
		return output.NewLiteralExpr(value, output.InferredType, nil)
	}))
	definitionMap.Set("dependencies", compileUsedDependenciesMetadata(meta))
	if meta.ViewProviders != nil {
		definitionMap.Set("viewProviders", *meta.ViewProviders)
	}
	if meta.Animations != nil {
		definitionMap.Set("animations", *meta.Animations)
	}

	if meta.ChangeDetection != nil {
		if changeDetectionStrategy, ok := meta.ChangeDetection.(core.ChangeDetectionStrategy); ok {
			strategyName := getChangeDetectionStrategyName(changeDetectionStrategy)
			definitionMap.Set(
				"changeDetection",
				output.NewReadPropExpr(
					output.NewExternalExpr(r3_identifiers.ChangeDetectionStrategy, nil, nil, nil),
					strategyName,
					output.InferredType,
					nil,
				),
			)
		} else {
			panic(errors.New("Impossible state! Change detection flag is not resolved!"))
		}
	}
	if meta.Encapsulation != core.ViewEncapsulationEmulated {
		encapsulationName := getViewEncapsulationName(meta.Encapsulation)
		definitionMap.Set(
			"encapsulation",
			output.NewReadPropExpr(
				output.NewExternalExpr(r3_identifiers.ViewEncapsulation, nil, nil, nil),
				encapsulationName,
				output.InferredType,
				nil,
			),
		)
	}

	if template.PreserveWhitespaces != nil && *template.PreserveWhitespaces {
		definitionMap.Set("preserveWhitespaces", output.NewLiteralExpr(true, output.InferredType, nil))
	}

	if meta.Defer.Mode == view.DeferBlockDepsEmitModePerBlock {
		resolvers := []output.OutputExpression{}
		hasResolvers := false

		for _, deps := range meta.Defer.Blocks {
			// Note: we need to push a `null` even if there are no dependencies, because matching of
			// defer resolver functions to defer blocks happens by index and not adding an array
			// entry for a block can throw off the blocks coming after it.
			if deps == nil {
				resolvers = append(resolvers, output.NewLiteralExpr(nil, output.InferredType, nil))
			} else {
				resolvers = append(resolvers, *deps)
				hasResolvers = true
			}
		}
		// If *all* the resolvers are null, we can skip the field.
		if hasResolvers {
			definitionMap.Set("deferBlockDependencies", output.NewLiteralArrayExpr(resolvers, nil, nil))
		}
	} else {
		panic(errors.New("Unsupported defer function emit mode in partial compilation"))
	}

	return definitionMap
}

// getTemplateExpression gets the template expression
func getTemplateExpression(
	_ *view.ParsedTemplate,
	templateInfo DeclareComponentTemplateInfo,
) output.OutputExpression {
	// If the template has been defined using a direct literal, we use that expression directly
	// without any modifications. This is ensures proper source mapping from the partially
	// compiled code to the source file declaring the template. Note that this does not capture
	// template literals referenced indirectly through an identifier.
	if templateInfo.InlineTemplateLiteralExpression != nil {
		return templateInfo.InlineTemplateLiteralExpression
	}

	// If the template is defined inline but not through a literal, the template has been resolved
	// through static interpretation. We create a literal but cannot provide any source span. Note
	// that we cannot use the expression defining the template because the linker expects the template
	// to be defined as a literal in the declaration.
	if templateInfo.IsInline {
		return output.NewLiteralExpr(templateInfo.Content, output.InferredType, nil)
	}

	// The template is external so we must synthesize an expression node with
	// the appropriate source-span.
	contents := templateInfo.Content
	file := util.NewParseSourceFile(contents, templateInfo.SourceUrl)
	start := util.NewParseLocation(file, 0, 0, 0)
	end := computeEndLocation(file, contents)
	span := util.NewParseSourceSpan(start, end, start, nil)
	return output.NewLiteralExpr(contents, output.InferredType, span)
}

// computeEndLocation computes the end location for a file
func computeEndLocation(file *util.ParseSourceFile, contents string) *util.ParseLocation {
	length := len(contents)
	lineStart := 0
	lastLineStart := 0
	line := 0
	for {
		lineStart = strings.Index(contents[lastLineStart:], "\n")
		if lineStart != -1 {
			lineStart += lastLineStart
			lastLineStart = lineStart + 1
			line++
		} else {
			break
		}
	}

	return util.NewParseLocation(file, length, line, length-lastLineStart)
}

// compileUsedDependenciesMetadata compiles the used dependencies metadata
func compileUsedDependenciesMetadata(
	meta *view.R3ComponentMetadata,
) output.OutputExpression {
	var wrapType func(output.OutputExpression) output.OutputExpression
	if meta.DeclarationListEmitMode != view.DeclarationListEmitModeDirect {
		wrapType = render3.GenerateForwardRef
	} else {
		wrapType = func(expr output.OutputExpression) output.OutputExpression {
			return expr
		}
	}

	if meta.DeclarationListEmitMode == view.DeclarationListEmitModeRuntimeResolved {
		panic(errors.New("Unsupported emit mode"))
	}

	// Convert []R3TemplateDependency to []interface{} for ToOptionalLiteralArray
	decls := make([]interface{}, len(meta.Declarations))
	for i, decl := range meta.Declarations {
		decls[i] = decl
	}
	return ToOptionalLiteralArray(decls, func(decl interface{}) output.OutputExpression {
		switch d := decl.(type) {
		case view.R3DirectiveDependencyMetadata:
			dirDecl := d
			dirMeta := view.NewDefinitionMap()
			dirMeta.Set("kind", output.NewLiteralExpr(
				func() string {
					if dirDecl.IsComponent {
						return "component"
					}
					return "directive"
				}(),
				output.InferredType,
				nil,
			))
			dirMeta.Set("type", wrapType(dirDecl.Type))
			dirMeta.Set("selector", output.NewLiteralExpr(dirDecl.Selector, output.InferredType, nil))
			dirMeta.Set("inputs", ToOptionalLiteralArray(dirDecl.Inputs, func(value string) output.OutputExpression {
				return output.NewLiteralExpr(value, output.InferredType, nil)
			}))
			dirMeta.Set("outputs", ToOptionalLiteralArray(dirDecl.Outputs, func(value string) output.OutputExpression {
				return output.NewLiteralExpr(value, output.InferredType, nil)
			}))
			dirMeta.Set("exportAs", ToOptionalLiteralArray(dirDecl.ExportAs, func(value string) output.OutputExpression {
				return output.NewLiteralExpr(value, output.InferredType, nil)
			}))
			return dirMeta.ToLiteralMap()
		case view.R3PipeDependencyMetadata:
			pipeDecl := d
			pipeMeta := view.NewDefinitionMap()
			pipeMeta.Set("kind", output.NewLiteralExpr("pipe", output.InferredType, nil))
			pipeMeta.Set("type", wrapType(pipeDecl.Type))
			pipeMeta.Set("name", output.NewLiteralExpr(pipeDecl.Name, output.InferredType, nil))
			return pipeMeta.ToLiteralMap()
		case view.R3NgModuleDependencyMetadata:
			ngModuleDecl := d
			ngModuleMeta := view.NewDefinitionMap()
			ngModuleMeta.Set("kind", output.NewLiteralExpr("ngmodule", output.InferredType, nil))
			ngModuleMeta.Set("type", wrapType(ngModuleDecl.Type))
			return ngModuleMeta.ToLiteralMap()
		default:
			// Try to handle as base R3TemplateDependency
			if baseDecl, ok := d.(view.R3TemplateDependency); ok {
				switch baseDecl.Kind {
				case view.R3TemplateDependencyKindDirective:
					panic(errors.New("R3DirectiveDependencyMetadata expected but got R3TemplateDependency"))
				case view.R3TemplateDependencyKindPipe:
					panic(errors.New("R3PipeDependencyMetadata expected but got R3TemplateDependency"))
				case view.R3TemplateDependencyKindNgModule:
					ngModuleMeta := view.NewDefinitionMap()
					ngModuleMeta.Set("kind", output.NewLiteralExpr("ngmodule", output.InferredType, nil))
					ngModuleMeta.Set("type", wrapType(baseDecl.Type))
					return ngModuleMeta.ToLiteralMap()
				}
			}
			panic(errors.New("Unknown dependency type"))
		}
	})
}

// BlockPresenceVisitor is a visitor that checks for block presence
type BlockPresenceVisitor struct {
	render3.RecursiveVisitor
	HasBlocks bool
}

// NewBlockPresenceVisitor creates a new BlockPresenceVisitor
func NewBlockPresenceVisitor() *BlockPresenceVisitor {
	return &BlockPresenceVisitor{
		RecursiveVisitor: *render3.NewRecursiveVisitor(),
		HasBlocks:        false,
	}
}

// VisitDeferredBlock visits a deferred block
func (v *BlockPresenceVisitor) VisitDeferredBlock(deferred *render3.DeferredBlock) interface{} {
	v.HasBlocks = true
	return nil
}

// VisitDeferredBlockPlaceholder visits a deferred block placeholder
func (v *BlockPresenceVisitor) VisitDeferredBlockPlaceholder(block *render3.DeferredBlockPlaceholder) interface{} {
	v.HasBlocks = true
	return nil
}

// VisitDeferredBlockLoading visits a deferred block loading
func (v *BlockPresenceVisitor) VisitDeferredBlockLoading(block *render3.DeferredBlockLoading) interface{} {
	v.HasBlocks = true
	return nil
}

// VisitDeferredBlockError visits a deferred block error
func (v *BlockPresenceVisitor) VisitDeferredBlockError(block *render3.DeferredBlockError) interface{} {
	v.HasBlocks = true
	return nil
}

// VisitIfBlock visits an if block
func (v *BlockPresenceVisitor) VisitIfBlock(block *render3.IfBlock) interface{} {
	v.HasBlocks = true
	return nil
}

// VisitIfBlockBranch visits an if block branch
func (v *BlockPresenceVisitor) VisitIfBlockBranch(block *render3.IfBlockBranch) interface{} {
	v.HasBlocks = true
	return nil
}

// VisitForLoopBlock visits a for loop block
func (v *BlockPresenceVisitor) VisitForLoopBlock(block *render3.ForLoopBlock) interface{} {
	v.HasBlocks = true
	return nil
}

// VisitForLoopBlockEmpty visits a for loop block empty
func (v *BlockPresenceVisitor) VisitForLoopBlockEmpty(block *render3.ForLoopBlockEmpty) interface{} {
	v.HasBlocks = true
	return nil
}

// VisitSwitchBlock visits a switch block
func (v *BlockPresenceVisitor) VisitSwitchBlock(block *render3.SwitchBlock) interface{} {
	v.HasBlocks = true
	return nil
}

// VisitSwitchBlockCase visits a switch block case
func (v *BlockPresenceVisitor) VisitSwitchBlockCase(block *render3.SwitchBlockCase) interface{} {
	v.HasBlocks = true
	return nil
}

// createComponentType creates the type specification from the component meta
// This is a copy of the private function from view/compiler/compiler.go
// to maintain 1:1 logic with TypeScript where it's exported
func createComponentType(meta *view.R3ComponentMetadata) output.Type {
	typeParams := createBaseDirectiveTypeParams(&meta.R3DirectiveMetadata)
	typeParams = append(typeParams, stringArrayAsType(meta.Template.NgContentSelectors))
	typeParams = append(typeParams, output.NewExpressionType(
		output.NewLiteralExpr(meta.IsStandalone, output.InferredType, nil),
		output.TypeModifierNone,
		nil,
	))
	typeParams = append(typeParams, createHostDirectivesType(&meta.R3DirectiveMetadata))
	if meta.IsSignal {
		typeParams = append(typeParams, output.NewExpressionType(
			output.NewLiteralExpr(meta.IsSignal, output.InferredType, nil),
			output.TypeModifierNone,
			nil,
		))
	}
	return output.NewExpressionType(
		output.NewExternalExpr(r3_identifiers.ComponentDeclaration, nil, typeParams, nil),
		output.TypeModifierNone,
		nil,
	)
}

// getChangeDetectionStrategyName returns the name of the change detection strategy enum value
func getChangeDetectionStrategyName(strategy core.ChangeDetectionStrategy) string {
	switch strategy {
	case core.ChangeDetectionStrategyOnPush:
		return "OnPush"
	case core.ChangeDetectionStrategyDefault:
		return "Default"
	default:
		return "Default"
	}
}

// getViewEncapsulationName returns the name of the view encapsulation enum value
func getViewEncapsulationName(encapsulation core.ViewEncapsulation) string {
	switch encapsulation {
	case core.ViewEncapsulationEmulated:
		return "Emulated"
	case core.ViewEncapsulationNone:
		return "None"
	case core.ViewEncapsulationShadowDom:
		return "ShadowDom"
	case core.ViewEncapsulationExperimentalIsolatedShadowDom:
		return "ExperimentalIsolatedShadowDom"
	default:
		return "Emulated"
	}
}
