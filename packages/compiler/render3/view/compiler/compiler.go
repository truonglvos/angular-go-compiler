package compiler

import (
	"regexp"
	"strings"

	"ngc-go/packages/compiler/core"
	"ngc-go/packages/compiler/css"
	"ngc-go/packages/compiler/expression_parser"
	"ngc-go/packages/compiler/output"
	constant "ngc-go/packages/compiler/pool"
	"ngc-go/packages/compiler/render3"
	"ngc-go/packages/compiler/render3/r3_identifiers"
	"ngc-go/packages/compiler/render3/view"
	"ngc-go/packages/compiler/schema"
	pipeline "ngc-go/packages/compiler/template/pipeline/src"
	"ngc-go/packages/compiler/template/pipeline/src/compilation"
	"ngc-go/packages/compiler/templateparser"
	"ngc-go/packages/compiler/util"
)

const COMPONENT_VARIABLE = "%COMP%"
const HOST_ATTR = "_nghost-" + COMPONENT_VARIABLE
const CONTENT_ATTR = "_ngcontent-" + COMPONENT_VARIABLE
const ANIMATE_LEAVE = "animate.leave"

var HOST_REG_EXP = regexp.MustCompile(`^(?:\[([^\]]+)\])|(?:\(([^\)]+)\))$`)

// HostBindingGroup represents the groups in the HOST_REG_EXP regex
type HostBindingGroup int

const (
	// HostBindingGroupBinding - group 1: "prop" from "[prop]", or "attr.role" from "[attr.role]", or @anim from [@anim]
	HostBindingGroupBinding HostBindingGroup = 1

	// HostBindingGroupEvent - group 2: "event" from "(event)"
	HostBindingGroupEvent HostBindingGroup = 2
)

// ParsedHostBindings defines Host Bindings structure that contains attributes, listeners, and properties,
// parsed from the `host` object defined for a Type.
type ParsedHostBindings struct {
	Attributes        map[string]output.OutputExpression
	Listeners         map[string]string
	Properties        map[string]string
	SpecialAttributes view.R3HostSpecialAttributes
}

// baseDirectiveFields creates the base fields for a directive definition map
func baseDirectiveFields(
	meta *view.R3DirectiveMetadata,
	constantPool *constant.ConstantPool,
	bindingParser templateparser.BindingParser,
) *view.DefinitionMap {
	definitionMap := view.NewDefinitionMap()
	selectors := core.ParseSelectorToR3Selector(meta.Selector)

	// e.g. `type: MyDirective`
	definitionMap.Set("type", meta.Type.Value)

	// e.g. `selectors: [['', 'someDir', '']]`
	if len(selectors) > 0 {
		// Convert R3CssSelectorList to []interface{} for AsLiteral
		selectorsInterface := make([]interface{}, len(selectors))
		for i, sel := range selectors {
			selectorsInterface[i] = sel
		}
		definitionMap.Set("selectors", view.AsLiteral(selectorsInterface))
	}

	if len(meta.Queries) > 0 {
		// e.g. `contentQueries: (rf, ctx, dirIndex) => { ... }
		definitionMap.Set(
			"contentQueries",
			createContentQueriesFunction(meta.Queries, constantPool, meta.Name),
		)
	}

	if len(meta.ViewQueries) > 0 {
		definitionMap.Set(
			"viewQuery",
			createViewQueriesFunction(meta.ViewQueries, constantPool, meta.Name),
		)
	}

	// e.g. `hostBindings: (rf, ctx) => { ... }
	selectorStr := ""
	if meta.Selector != nil {
		selectorStr = *meta.Selector
	}
	definitionMap.Set(
		"hostBindings",
		createHostBindingsFunction(
			meta.Host,
			meta.TypeSourceSpan,
			bindingParser,
			constantPool,
			selectorStr,
			meta.Name,
			definitionMap,
		),
	)

	// Convert Inputs map[string]R3InputMetadata to map[string]interface{} for ConditionallyCreateDirectiveBindingLiteral
	inputsMap := make(map[string]interface{})
	for key, value := range meta.Inputs {
		bindingValue := &view.DirectiveBindingValue{
			ClassPropertyName:   value.ClassPropertyName,
			BindingPropertyName: value.BindingPropertyName,
			IsSignal:            value.IsSignal,
		}
		if value.TransformFunction != nil {
			bindingValue.TransformFunction = *value.TransformFunction
		}
		inputsMap[key] = bindingValue
	}
	// e.g 'inputs: {a: 'a'}`
	definitionMap.Set("inputs", view.ConditionallyCreateDirectiveBindingLiteral(inputsMap, true))

	// e.g 'outputs: {a: 'a'}`
	outputsMap := make(map[string]interface{})
	for key, value := range meta.Outputs {
		outputsMap[key] = value
	}
	definitionMap.Set("outputs", view.ConditionallyCreateDirectiveBindingLiteral(outputsMap, false))

	if meta.ExportAs != nil && len(meta.ExportAs) > 0 {
		// Convert []string to []output.OutputExpression
		exportAsExprs := make([]output.OutputExpression, len(meta.ExportAs))
		for i, e := range meta.ExportAs {
			exportAsExprs[i] = output.NewLiteralExpr(e, output.InferredType, nil)
		}
		definitionMap.Set("exportAs", output.NewLiteralArrayExpr(exportAsExprs, nil, nil))
	}

	if !meta.IsStandalone {
		definitionMap.Set("standalone", output.NewLiteralExpr(false, output.InferredType, nil))
	}
	if meta.IsSignal {
		definitionMap.Set("signals", output.NewLiteralExpr(true, output.InferredType, nil))
	}

	return definitionMap
}

// hasAnimationHostBinding checks if the metadata has animation host binding
func hasAnimationHostBinding(meta interface{}) bool {
	var host view.R3HostMetadata
	switch m := meta.(type) {
	case *view.R3DirectiveMetadata:
		host = m.Host
	case *view.R3ComponentMetadata:
		host = m.Host
	default:
		return false
	}
	_, hasAttr := host.Attributes[ANIMATE_LEAVE]
	_, hasProp := host.Properties[ANIMATE_LEAVE]
	_, hasListener := host.Listeners[ANIMATE_LEAVE]
	return hasAttr || hasProp || hasListener
}

// QueryFlags represents flags used with queries
type QueryFlags int

const (
	QueryFlagsNone                QueryFlags = 0b0000
	QueryFlagsDescendants         QueryFlags = 0b0001
	QueryFlagsIsStatic            QueryFlags = 0b0010
	QueryFlagsEmitDistinctChanges QueryFlags = 0b0100
)

// toQueryFlags translates query metadata into query flags
func toQueryFlags(query view.R3QueryMetadata) int {
	var flags QueryFlags = QueryFlagsNone
	if query.Descendants {
		flags |= QueryFlagsDescendants
	}
	if query.Static {
		flags |= QueryFlagsIsStatic
	}
	if query.EmitDistinctChangesOnly {
		flags |= QueryFlagsEmitDistinctChanges
	}
	return int(flags)
}

// queryAdvancePlaceholder represents a placeholder for query advance statements
type queryAdvancePlaceholder struct{}

var queryAdvancePlaceholderInstance = &queryAdvancePlaceholder{}

// getQueryPredicate gets the query predicate expression
func getQueryPredicate(
	query view.R3QueryMetadata,
	constantPool *constant.ConstantPool,
) output.OutputExpression {
	// Check if predicate is a string array
	if predicateArray, ok := query.Predicate.([]string); ok {
		predicate := []output.OutputExpression{}
		for _, selector := range predicateArray {
			// Split comma-separated selectors
			parts := strings.Split(selector, ",")
			for _, part := range parts {
				trimmed := strings.TrimSpace(part)
				predicate = append(predicate, output.NewLiteralExpr(trimmed, output.InferredType, nil))
			}
		}
		forceShared := true
		return constantPool.GetConstLiteral(
			output.NewLiteralArrayExpr(predicate, nil, nil),
			&forceShared,
		)
	}

	// Otherwise, predicate is a MaybeForwardRefExpression
	if maybeRef, ok := query.Predicate.(render3.MaybeForwardRefExpression); ok {
		switch maybeRef.ForwardRef {
		case render3.ForwardRefHandlingNone, render3.ForwardRefHandlingUnwrapped:
			return maybeRef.Expression
		case render3.ForwardRefHandlingWrapped:
			resolveForwardRef := output.NewExternalExpr(r3_identifiers.ResolveForwardRef, nil, nil, nil)
			return output.NewInvokeFunctionExpr(
				resolveForwardRef,
				[]output.OutputExpression{maybeRef.Expression},
				nil,
				nil,
				false,
			)
		}
	}

	// Fallback: try to cast as OutputExpression directly
	if expr, ok := query.Predicate.(output.OutputExpression); ok {
		return expr
	}

	return nil
}

// queryTypeFns represents query type functions (signal-based and non-signal)
type queryTypeFns struct {
	signalBased output.ExternalReference
	nonSignal   output.ExternalReference
}

// createQueryCreateCall creates a query create call
func createQueryCreateCall(
	query view.R3QueryMetadata,
	constantPool *constant.ConstantPool,
	queryTypeFns queryTypeFns,
	prependParams []output.OutputExpression,
) *output.InvokeFunctionExpr {
	parameters := []output.OutputExpression{}
	if prependParams != nil {
		parameters = append(parameters, prependParams...)
	}
	if query.IsSignal {
		// For signal queries, add the context property read first
		ctxVar := output.NewReadVarExpr(view.CONTEXT_NAME, output.DynamicType, nil)
		propRead := output.NewReadPropExpr(ctxVar, query.PropertyName, output.DynamicType, nil)
		parameters = append(parameters, propRead)
	}
	parameters = append(parameters, getQueryPredicate(query, constantPool))
	parameters = append(parameters, output.NewLiteralExpr(toQueryFlags(query), output.InferredType, nil))
	if query.Read != nil {
		parameters = append(parameters, *query.Read)
	}

	queryCreateFn := queryTypeFns.nonSignal
	if query.IsSignal {
		queryCreateFn = queryTypeFns.signalBased
	}
	fnExpr := output.NewExternalExpr(&queryCreateFn, nil, nil, nil)
	return output.NewInvokeFunctionExpr(fnExpr, parameters, nil, nil, false)
}

// renderFlagCheckIfStmt creates an if statement that checks render flags
func renderFlagCheckIfStmt(flags core.RenderFlags, statements []output.OutputStatement) *output.IfStmt {
	rfVar := output.NewReadVarExpr(view.RENDER_FLAGS, output.DynamicType, nil)
	flagsLiteral := output.NewLiteralExpr(int(flags), output.InferredType, nil)
	condition := output.NewBinaryOperatorExpr(
		output.BinaryOperatorBitwiseAnd,
		rfVar,
		flagsLiteral,
		nil,
		nil,
	)
	return output.NewIfStmt(condition, statements, nil, nil, nil)
}

// collapseAdvanceStatements collapses query advance placeholders in a list of statements
func collapseAdvanceStatements(
	statements []interface{}, // []output.OutputStatement | *queryAdvancePlaceholder
) []output.OutputStatement {
	result := []output.OutputStatement{}
	advanceCollapseCount := 0

	flushAdvanceCount := func() {
		if advanceCollapseCount > 0 {
			var args []output.OutputExpression
			if advanceCollapseCount != 1 {
				args = []output.OutputExpression{output.NewLiteralExpr(advanceCollapseCount, output.InferredType, nil)}
			}
		queryAdvanceExpr := output.NewExternalExpr(r3_identifiers.QueryAdvance, nil, nil, nil)
		callExpr := output.NewInvokeFunctionExpr(queryAdvanceExpr, args, nil, nil, false)
		stmt := output.NewExpressionStatement(callExpr, nil, nil)
			// Insert at beginning (unshift equivalent in Go)
			result = append([]output.OutputStatement{stmt}, result...)
			advanceCollapseCount = 0
		}
	}

	// Iterate through statements in reverse
	for i := len(statements) - 1; i >= 0; i-- {
		st := statements[i]
		if _, ok := st.(*queryAdvancePlaceholder); ok {
			advanceCollapseCount++
		} else {
			flushAdvanceCount()
			if stmt, ok := st.(output.OutputStatement); ok {
				// Insert at beginning (unshift equivalent in Go)
				result = append([]output.OutputStatement{stmt}, result...)
			}
		}
	}
	flushAdvanceCount()
	return result
}

// createViewQueriesFunction creates a view queries function
func createViewQueriesFunction(
	viewQueries []view.R3QueryMetadata,
	constantPool *constant.ConstantPool,
	name string,
) output.OutputExpression {
	createStatements := []output.OutputStatement{}
	updateStatements := []interface{}{} // []output.OutputStatement | *queryAdvancePlaceholder

	pushStatement := func(st output.OutputStatement) {
		updateStatements = append(updateStatements, st)
	}
	tempAllocator := view.TemporaryAllocator(pushStatement, view.TEMPORARY_NAME)

	for _, query := range viewQueries {
		// Creation call, e.g. r3.viewQuery(somePredicate, true) or
		//                r3.viewQuerySignal(ctx.prop, somePredicate, true);
		queryDefinitionCall := createQueryCreateCall(query, constantPool, queryTypeFns{
			signalBased: *r3_identifiers.ViewQuerySignal,
			nonSignal:   *r3_identifiers.ViewQuery,
		}, nil)
		createStatements = append(createStatements, output.NewExpressionStatement(queryDefinitionCall, nil, nil))

		// Signal queries update lazily and we just advance the index.
		if query.IsSignal {
			updateStatements = append(updateStatements, queryAdvancePlaceholderInstance)
			continue
		}

		// Update, e.g. (r3.queryRefresh(tmp = r3.loadQuery()) && (ctx.someDir = tmp));
		temporary := tempAllocator()
		loadQueryExpr := output.NewExternalExpr(r3_identifiers.LoadQuery, nil, nil, nil)
		getQueryList := output.NewInvokeFunctionExpr(loadQueryExpr, []output.OutputExpression{}, nil, nil, false)
		queryRefreshExpr := output.NewExternalExpr(r3_identifiers.QueryRefresh, nil, nil, nil)
		setExpr := temporary.Set(getQueryList) // temporary.set(getQueryList)
		refresh := output.NewInvokeFunctionExpr(queryRefreshExpr, []output.OutputExpression{setExpr}, nil, nil, false)
		ctxVar := output.NewReadVarExpr(view.CONTEXT_NAME, output.DynamicType, nil)
		ctxProp := output.NewReadPropExpr(ctxVar, query.PropertyName, output.DynamicType, nil)
		var updateDirective output.OutputExpression
		if query.First {
			firstProp := output.NewReadPropExpr(temporary, "first", output.DynamicType, nil)
			updateDirective = ctxProp.Set(firstProp) // ctx.prop = temporary.first
		} else {
			updateDirective = ctxProp.Set(temporary) // ctx.prop = temporary
		}
		combined := output.NewBinaryOperatorExpr(
			output.BinaryOperatorAnd,
			refresh,
			updateDirective,
			nil,
			nil,
		)
		updateStatements = append(updateStatements, output.NewExpressionStatement(combined, nil, nil))
	}

	var viewQueryFnName *string
	if name != "" {
		fnName := name + "_Query"
		viewQueryFnName = &fnName
	}

	statements := []output.OutputStatement{
		renderFlagCheckIfStmt(core.RenderFlagsCreate, createStatements),
		renderFlagCheckIfStmt(core.RenderFlagsUpdate, collapseAdvanceStatements(updateStatements)),
	}

	return output.NewFunctionExpr(
		[]*output.FnParam{
			output.NewFnParam(view.RENDER_FLAGS, output.NumberType),
			output.NewFnParam(view.CONTEXT_NAME, nil),
		},
		statements,
		output.InferredType,
		nil,
		viewQueryFnName,
	)
}

// createContentQueriesFunction creates a content queries function
func createContentQueriesFunction(
	queries []view.R3QueryMetadata,
	constantPool *constant.ConstantPool,
	name string,
) output.OutputExpression {
	createStatements := []output.OutputStatement{}
	updateStatements := []interface{}{} // []output.OutputStatement | *queryAdvancePlaceholder

	pushStatement := func(st output.OutputStatement) {
		updateStatements = append(updateStatements, st)
	}
	tempAllocator := view.TemporaryAllocator(pushStatement, view.TEMPORARY_NAME)

	for _, query := range queries {
		// Creation, e.g. r3.contentQuery(dirIndex, somePredicate, true, null) or
		//                r3.contentQuerySignal(dirIndex, propName, somePredicate, <flags>, <read>).
		dirIndexVar := output.NewReadVarExpr("dirIndex", output.DynamicType, nil)
		queryDefinitionCall := createQueryCreateCall(query, constantPool, queryTypeFns{
			nonSignal:   *r3_identifiers.ContentQuery,
			signalBased: *r3_identifiers.ContentQuerySignal,
		}, []output.OutputExpression{dirIndexVar})
		createStatements = append(createStatements, output.NewExpressionStatement(queryDefinitionCall, nil, nil))

		// Signal queries update lazily and we just advance the index.
		if query.IsSignal {
			updateStatements = append(updateStatements, queryAdvancePlaceholderInstance)
			continue
		}

		// Update, e.g. (r3.queryRefresh(tmp = r3.loadQuery()) && (ctx.someDir = tmp));
		temporary := tempAllocator()
		loadQueryExpr := output.NewExternalExpr(r3_identifiers.LoadQuery, nil, nil, nil)
		getQueryList := output.NewInvokeFunctionExpr(loadQueryExpr, []output.OutputExpression{}, nil, nil, false)
		queryRefreshExpr := output.NewExternalExpr(r3_identifiers.QueryRefresh, nil, nil, nil)
		setExpr := temporary.Set(getQueryList) // temporary.set(getQueryList)
		refresh := output.NewInvokeFunctionExpr(queryRefreshExpr, []output.OutputExpression{setExpr}, nil, nil, false)
		ctxVar := output.NewReadVarExpr(view.CONTEXT_NAME, output.DynamicType, nil)
		ctxProp := output.NewReadPropExpr(ctxVar, query.PropertyName, output.DynamicType, nil)
		var updateDirective output.OutputExpression
		if query.First {
			firstProp := output.NewReadPropExpr(temporary, "first", output.DynamicType, nil)
			updateDirective = ctxProp.Set(firstProp) // ctx.prop = temporary.first
		} else {
			updateDirective = ctxProp.Set(temporary) // ctx.prop = temporary
		}
		combined := output.NewBinaryOperatorExpr(
			output.BinaryOperatorAnd,
			refresh,
			updateDirective,
			nil,
			nil,
		)
		updateStatements = append(updateStatements, output.NewExpressionStatement(combined, nil, nil))
	}

	var contentQueriesFnName *string
	if name != "" {
		fnName := name + "_ContentQueries"
		contentQueriesFnName = &fnName
	}

	statements := []output.OutputStatement{
		renderFlagCheckIfStmt(core.RenderFlagsCreate, createStatements),
		renderFlagCheckIfStmt(core.RenderFlagsUpdate, collapseAdvanceStatements(updateStatements)),
	}

	return output.NewFunctionExpr(
		[]*output.FnParam{
			output.NewFnParam(view.RENDER_FLAGS, output.NumberType),
			output.NewFnParam(view.CONTEXT_NAME, nil),
			output.NewFnParam("dirIndex", nil),
		},
		statements,
		output.InferredType,
		nil,
		contentQueriesFnName,
	)
}

// addFeatures adds features to the definition map
func addFeatures(
	definitionMap *view.DefinitionMap,
	meta interface{},
) {
	features := []output.OutputExpression{}

	var providers *output.OutputExpression
	var viewProviders *output.OutputExpression

	switch m := meta.(type) {
	case *view.R3DirectiveMetadata:
		providers = m.Providers
	case *view.R3ComponentMetadata:
		providers = m.R3DirectiveMetadata.Providers
		viewProviders = m.ViewProviders
	}

	if providers != nil || viewProviders != nil {
		args := []output.OutputExpression{}
		if providers != nil {
			args = append(args, *providers)
		} else {
			args = append(args, output.NewLiteralArrayExpr([]output.OutputExpression{}, nil, nil))
		}
		if viewProviders != nil {
			args = append(args, *viewProviders)
		}
		features = append(features, output.NewInvokeFunctionExpr(
			output.NewExternalExpr(r3_identifiers.ProvidersFeature, nil, nil, nil),
			args,
			nil,
			nil,
			false,
		))
	}

	var hostDirectives []view.R3HostDirectiveMetadata
	var usesInheritance bool
	var lifecycle view.R3LifecycleMetadata
	var externalStyles []string

	switch m := meta.(type) {
	case *view.R3DirectiveMetadata:
		if m.HostDirectives != nil {
			hostDirectives = m.HostDirectives
		}
		usesInheritance = m.UsesInheritance
		lifecycle = m.Lifecycle
	case *view.R3ComponentMetadata:
		if m.HostDirectives != nil {
			hostDirectives = m.HostDirectives
		}
		usesInheritance = m.UsesInheritance
		lifecycle = m.Lifecycle
		if m.ExternalStyles != nil {
			externalStyles = m.ExternalStyles
		}
	}

	if hostDirectives != nil && len(hostDirectives) > 0 {
		features = append(features, output.NewInvokeFunctionExpr(
			output.NewExternalExpr(r3_identifiers.HostDirectivesFeature, nil, nil, nil),
			[]output.OutputExpression{createHostDirectivesFeatureArg(hostDirectives)},
			nil,
			nil,
			false,
		))
	}

	if usesInheritance {
		features = append(features, output.NewExternalExpr(r3_identifiers.InheritDefinitionFeature, nil, nil, nil))
	}

	if lifecycle.UsesOnChanges {
		features = append(features, output.NewExternalExpr(r3_identifiers.NgOnChangesFeature, nil, nil, nil))
	}

	if len(externalStyles) > 0 {
		externalStyleNodes := make([]output.OutputExpression, len(externalStyles))
		for i, style := range externalStyles {
			externalStyleNodes[i] = output.NewLiteralExpr(style, output.InferredType, nil)
		}
		features = append(features, output.NewInvokeFunctionExpr(
			output.NewExternalExpr(r3_identifiers.ExternalStylesFeature, nil, nil, nil),
			[]output.OutputExpression{output.NewLiteralArrayExpr(externalStyleNodes, nil, nil)},
			nil,
			nil,
			false,
		))
	}

	if len(features) > 0 {
		definitionMap.Set("features", output.NewLiteralArrayExpr(features, nil, nil))
	}
}

// CompileDirectiveFromMetadata compiles a directive for the render3 runtime as defined by the `R3DirectiveMetadata`.
func CompileDirectiveFromMetadata(
	meta *view.R3DirectiveMetadata,
	constantPool *constant.ConstantPool,
	bindingParser templateparser.BindingParser,
) render3.R3CompiledExpression {
	definitionMap := baseDirectiveFields(meta, constantPool, bindingParser)
	addFeatures(definitionMap, meta)
	expression := output.NewInvokeFunctionExpr(
		output.NewExternalExpr(r3_identifiers.DefineDirective, nil, nil, nil),
		[]output.OutputExpression{definitionMap.ToLiteralMap()},
		nil,
		nil,
		true,
	)
	typ := createDirectiveType(meta)

	return render3.R3CompiledExpression{
		Expression: expression,
		Type:       typ,
		Statements: []output.OutputStatement{},
	}
}

// CompileComponentFromMetadata compiles a component for the render3 runtime as defined by the `R3ComponentMetadata`.
func CompileComponentFromMetadata(
	meta *view.R3ComponentMetadata,
	constantPool *constant.ConstantPool,
	bindingParser templateparser.BindingParser,
) render3.R3CompiledExpression {
	definitionMap := baseDirectiveFields(&meta.R3DirectiveMetadata, constantPool, bindingParser)
	addFeatures(definitionMap, meta)

	var selector *css.CssSelector
	var firstSelector *css.CssSelector
	if meta.Selector != nil && *meta.Selector != "" {
		parsed, err := css.ParseCssSelector(*meta.Selector)
		if err == nil && len(parsed) > 0 {
			selector = parsed[0]
			firstSelector = selector
		}
	}

	// e.g. `attr: ["class", ".my.app"]`
	if firstSelector != nil {
		selectorAttributes := firstSelector.GetAttrs()
		if len(selectorAttributes) > 0 {
			attrsExprs := make([]output.OutputExpression, len(selectorAttributes))
			for i, value := range selectorAttributes {
				if value != "" {
					attrsExprs[i] = output.NewLiteralExpr(value, output.InferredType, nil)
				} else {
					attrsExprs[i] = output.NewLiteralExpr(nil, output.InferredType, nil)
				}
			}
			forceShared := true
			definitionMap.Set(
				"attrs",
				constantPool.GetConstLiteral(
					output.NewLiteralArrayExpr(attrsExprs, nil, nil),
					&forceShared,
				),
			)
		}
	}

	// e.g. `template: function MyComponent_Template(_ctx, _cm) {...}`
	templateTypeName := meta.Name

	var allDeferrableDepsFn *output.ReadVarExpr
	if meta.Defer.Mode == view.DeferBlockDepsEmitModePerComponent && meta.Defer.DependenciesFn != nil {
		fnName := templateTypeName + "_DeferFn"
		constantPool.AddStatement(
			output.NewDeclareVarStmt(
				fnName,
				*meta.Defer.DependenciesFn,
				nil,
				output.StmtModifierFinal,
				nil,
				nil,
			),
		)
		allDeferrableDepsFn = output.NewReadVarExpr(fnName, output.DynamicType, nil)
	}

	compilationMode := compilation.TemplateCompilationModeFull
	if meta.IsStandalone && !meta.HasDirectiveDependencies {
		compilationMode = compilation.TemplateCompilationModeDomOnly
	}

	// First the template is ingested into IR:
	tpl := pipeline.IngestComponent(
		meta.Name,
		meta.Template.Nodes,
		constantPool,
		compilationMode,
		meta.RelativeContextFilePath,
		meta.I18nUseExternalIds,
		meta.Defer,
		allDeferrableDepsFn,
		meta.RelativeTemplatePath,
		view.GetTemplateSourceLocationsEnabled(),
	)

	// Then the IR is transformed to prepare it for code generation.
	pipeline.Transform(tpl, compilation.CompilationJobKindTmpl)

	// Finally we emit the template function:
	templateFn := pipeline.EmitTemplateFn(tpl, constantPool)

	if tpl.ContentSelectors != nil {
		definitionMap.Set("ngContentSelectors", tpl.ContentSelectors)
	}

	definitionMap.Set("decls", output.NewLiteralExpr(tpl.Root.Decls, output.InferredType, nil))
	definitionMap.Set("vars", output.NewLiteralExpr(tpl.Root.Vars, output.InferredType, nil))
	if len(tpl.Consts) > 0 {
		if len(tpl.ConstsInitializers) > 0 {
			statements := append([]output.OutputStatement{}, tpl.ConstsInitializers...)
			statements = append(statements, output.NewReturnStatement(
				output.NewLiteralArrayExpr(tpl.Consts, nil, nil),
				nil,
				nil,
			))
			definitionMap.Set(
				"consts",
				output.NewArrowFunctionExpr([]*output.FnParam{}, statements, nil, nil),
			)
		} else {
			definitionMap.Set("consts", output.NewLiteralArrayExpr(tpl.Consts, nil, nil))
		}
	}
	definitionMap.Set("template", templateFn)

	if meta.DeclarationListEmitMode != view.DeclarationListEmitModeRuntimeResolved && len(meta.Declarations) > 0 {
		declTypes := make([]output.OutputExpression, len(meta.Declarations))
		for i, decl := range meta.Declarations {
			declTypes[i] = decl.Type
		}
		definitionMap.Set(
			"dependencies",
			compileDeclarationList(
				output.NewLiteralArrayExpr(declTypes, nil, nil),
				meta.DeclarationListEmitMode,
			),
		)
	} else if meta.DeclarationListEmitMode == view.DeclarationListEmitModeRuntimeResolved {
		args := []output.OutputExpression{meta.Type.Value}
		if meta.RawImports != nil {
			args = append(args, *meta.RawImports)
		}
		definitionMap.Set(
			"dependencies",
			output.NewInvokeFunctionExpr(
				output.NewExternalExpr(r3_identifiers.GetComponentDepsFactory, nil, nil, nil),
				args,
				nil,
				nil,
				false,
			),
		)
	}

	encapsulation := meta.Encapsulation
	if encapsulation == core.ViewEncapsulationNone {
		encapsulation = core.ViewEncapsulationEmulated
	}

	hasStyles := meta.ExternalStyles != nil && len(meta.ExternalStyles) > 0
	if len(meta.Styles) > 0 {
		var styleValues []string
		if encapsulation == core.ViewEncapsulationEmulated {
			styleValues = compileStyles(meta.Styles, CONTENT_ATTR, HOST_ATTR)
		} else {
			styleValues = meta.Styles
		}
		styleNodes := []output.OutputExpression{}
		for _, style := range styleValues {
			if strings.TrimSpace(style) != "" {
				styleNodes = append(styleNodes, constantPool.GetConstLiteral(
					output.NewLiteralExpr(style, output.InferredType, nil),
					nil, // forceShared
				))
			}
		}

		if len(styleNodes) > 0 {
			hasStyles = true
			definitionMap.Set("styles", output.NewLiteralArrayExpr(styleNodes, nil, nil))
		}
	}

	if !hasStyles && encapsulation == core.ViewEncapsulationEmulated {
		encapsulation = core.ViewEncapsulationNone
	}

	if encapsulation != core.ViewEncapsulationEmulated {
		definitionMap.Set("encapsulation", output.NewLiteralExpr(encapsulation, output.InferredType, nil))
	}

	if meta.Animations != nil {
		definitionMap.Set(
			"data",
			output.NewLiteralMapExpr(
				[]*output.LiteralMapEntry{
					output.NewLiteralMapEntry("animation", *meta.Animations, false),
				},
				nil,
				nil,
			),
		)
	}

	if meta.ChangeDetection != nil {
		if cdStrategy, ok := meta.ChangeDetection.(core.ChangeDetectionStrategy); ok {
			if cdStrategy != core.ChangeDetectionStrategyDefault {
				definitionMap.Set("changeDetection", output.NewLiteralExpr(cdStrategy, output.InferredType, nil))
			}
		} else if cdExpr, ok := meta.ChangeDetection.(output.OutputExpression); ok {
			definitionMap.Set("changeDetection", cdExpr)
		}
	}

	expression := output.NewInvokeFunctionExpr(
		output.NewExternalExpr(r3_identifiers.DefineComponent, nil, nil, nil),
		[]output.OutputExpression{definitionMap.ToLiteralMap()},
		nil,
		nil,
		true,
	)
	typ := createComponentType(meta)

	return render3.R3CompiledExpression{
		Expression: expression,
		Type:       typ,
		Statements: []output.OutputStatement{},
	}
}

// compileDeclarationList compiles the array literal of declarations into an expression according to the provided emit mode.
func compileDeclarationList(
	list *output.LiteralArrayExpr,
	mode view.DeclarationListEmitMode,
) output.OutputExpression {
	switch mode {
	case view.DeclarationListEmitModeDirect:
		// directives: [MyDir],
		return list
	case view.DeclarationListEmitModeClosure:
		// directives: function () { return [MyDir]; }
		return output.NewArrowFunctionExpr([]*output.FnParam{}, []output.OutputStatement{
			output.NewReturnStatement(list, nil, nil),
		}, nil, nil)
	case view.DeclarationListEmitModeClosureResolved:
		// directives: function () { return [MyDir].map(ng.resolveForwardRef); }
		resolvedList := output.NewInvokeFunctionExpr(
			output.NewReadPropExpr(list, "map", output.DynamicType, nil),
			[]output.OutputExpression{
				output.NewExternalExpr(r3_identifiers.ResolveForwardRef, nil, nil, nil),
			},
			nil,
			nil,
			false,
		)
		return output.NewArrowFunctionExpr([]*output.FnParam{}, []output.OutputStatement{
			output.NewReturnStatement(resolvedList, nil, nil),
		}, nil, nil)
	case view.DeclarationListEmitModeRuntimeResolved:
		panic("Unsupported with an array of pre-resolved dependencies")
	default:
		panic("Unknown DeclarationListEmitMode")
	}
}

// stringAsType creates a type from a string
func stringAsType(str string) output.Type {
	return output.NewExpressionType(
		output.NewLiteralExpr(str, output.InferredType, nil),
		output.TypeModifierNone,
		nil,
	)
}

// stringMapAsLiteralExpression creates a literal map expression from a string map
func stringMapAsLiteralExpression(m map[string]interface{}) *output.LiteralMapExpr {
	mapValues := []*output.LiteralMapEntry{}
	for key, value := range m {
		var literalValue output.OutputExpression
		if arr, ok := value.([]string); ok && len(arr) > 0 {
			literalValue = output.NewLiteralExpr(arr[0], output.InferredType, nil)
		} else if str, ok := value.(string); ok {
			literalValue = output.NewLiteralExpr(str, output.InferredType, nil)
		} else {
			literalValue = output.NewLiteralExpr(value, output.InferredType, nil)
		}
		mapValues = append(mapValues, output.NewLiteralMapEntry(key, literalValue, true))
	}
	return output.NewLiteralMapExpr(mapValues, nil, nil)
}

// stringArrayAsType creates a type from an array of strings
func stringArrayAsType(arr []string) output.Type {
	if len(arr) > 0 {
		literals := make([]output.OutputExpression, len(arr))
		for i, value := range arr {
			literals[i] = output.NewLiteralExpr(value, output.InferredType, nil)
		}
		return output.NewExpressionType(
			output.NewLiteralArrayExpr(literals, nil, nil),
			output.TypeModifierNone,
			nil,
		)
	}
	return output.NoneType
}

// createBaseDirectiveTypeParams creates the base type parameters for a directive
func createBaseDirectiveTypeParams(meta *view.R3DirectiveMetadata) []output.Type {
	selectorForType := ""
	if meta.Selector != nil {
		selectorForType = strings.ReplaceAll(*meta.Selector, "\n", "")
	}

	typeParams := []output.Type{
		render3.TypeWithParameters(meta.Type.Type, meta.TypeArgumentCount),
	}

	if selectorForType != "" {
		typeParams = append(typeParams, stringAsType(selectorForType))
	} else {
		typeParams = append(typeParams, output.NoneType)
	}

	if meta.ExportAs != nil && len(meta.ExportAs) > 0 {
		typeParams = append(typeParams, stringArrayAsType(meta.ExportAs))
	} else {
		typeParams = append(typeParams, output.NoneType)
	}

	typeParams = append(typeParams, output.NewExpressionType(
		getInputsTypeExpression(meta),
		output.TypeModifierNone,
		nil,
	))

	typeParams = append(typeParams, output.NewExpressionType(
		stringMapAsLiteralExpression(convertStringMap(meta.Outputs)),
		output.TypeModifierNone,
		nil,
	))

	queryNames := make([]string, len(meta.Queries))
	for i, q := range meta.Queries {
		queryNames[i] = q.PropertyName
	}
	typeParams = append(typeParams, stringArrayAsType(queryNames))

	return typeParams
}

// convertStringMap converts map[string]string to map[string]interface{}
func convertStringMap(m map[string]string) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m {
		result[k] = v
	}
	return result
}

// getInputsTypeExpression creates a type expression for inputs
func getInputsTypeExpression(meta *view.R3DirectiveMetadata) output.OutputExpression {
	entries := []*output.LiteralMapEntry{}
	for key, value := range meta.Inputs {
		values := []*output.LiteralMapEntry{
			output.NewLiteralMapEntry("alias", output.NewLiteralExpr(value.BindingPropertyName, output.InferredType, nil), true),
			output.NewLiteralMapEntry("required", output.NewLiteralExpr(value.Required, output.InferredType, nil), true),
		}

		if value.IsSignal {
			values = append(values, output.NewLiteralMapEntry("isSignal", output.NewLiteralExpr(value.IsSignal, output.InferredType, nil), true))
		}

		entries = append(entries, output.NewLiteralMapEntry(
			key,
			output.NewLiteralMapExpr(values, nil, nil),
			true,
		))
	}
	return output.NewLiteralMapExpr(entries, nil, nil)
}

// createDirectiveType creates the type specification from the directive meta
func createDirectiveType(meta *view.R3DirectiveMetadata) output.Type {
	typeParams := createBaseDirectiveTypeParams(meta)
	typeParams = append(typeParams, output.NoneType) // ngContentSelectors slot
	typeParams = append(typeParams, output.NewExpressionType(
		output.NewLiteralExpr(meta.IsStandalone, output.InferredType, nil),
		output.TypeModifierNone,
		nil,
	))
	typeParams = append(typeParams, createHostDirectivesType(meta))
	if meta.IsSignal {
		typeParams = append(typeParams, output.NewExpressionType(
			output.NewLiteralExpr(meta.IsSignal, output.InferredType, nil),
			output.TypeModifierNone,
			nil,
		))
	}
	return output.NewExpressionType(
		output.NewExternalExpr(r3_identifiers.DirectiveDeclaration, nil, typeParams, nil),
		output.TypeModifierNone,
		nil,
	)
}

// createComponentType creates the type specification from the component meta
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

// createHostBindingsFunction creates a host bindings function
func createHostBindingsFunction(
	hostBindingsMetadata view.R3HostMetadata,
	typeSourceSpan *util.ParseSourceSpan,
	bindingParser templateparser.BindingParser,
	constantPool *constant.ConstantPool,
	selector string,
	name string,
	definitionMap *view.DefinitionMap,
) output.OutputExpression {
	bindings := bindingParser.CreateBoundHostProperties(
		hostBindingsMetadata.Properties,
		typeSourceSpan,
	)

	eventBindings := bindingParser.CreateDirectiveHostEventAsts(
		hostBindingsMetadata.Listeners,
		typeSourceSpan,
	)

	if hostBindingsMetadata.SpecialAttributes.StyleAttr != nil {
		hostBindingsMetadata.Attributes["style"] = output.NewLiteralExpr(
			*hostBindingsMetadata.SpecialAttributes.StyleAttr,
			output.InferredType,
			nil,
		)
	}
	if hostBindingsMetadata.SpecialAttributes.ClassAttr != nil {
		hostBindingsMetadata.Attributes["class"] = output.NewLiteralExpr(
			*hostBindingsMetadata.SpecialAttributes.ClassAttr,
			output.InferredType,
			nil,
		)
	}

	hostJob := pipeline.IngestHostBinding(
		&pipeline.HostBindingInput{
			ComponentName:     name,
			ComponentSelector: selector,
			Properties:        bindings,
			Events:            eventBindings,
			Attributes:        hostBindingsMetadata.Attributes,
		},
		bindingParser,
		constantPool,
	)
	pipeline.Transform(hostJob, compilation.CompilationJobKindHost)

	definitionMap.Set("hostAttrs", hostJob.Root.Attributes)

	if hostJob.Root.Vars != nil && *hostJob.Root.Vars > 0 {
		definitionMap.Set("hostVars", output.NewLiteralExpr(*hostJob.Root.Vars, output.InferredType, nil))
	}

	return pipeline.EmitHostBindingFunction(hostJob)
}

// parseHostBindings parses host bindings from a host object
func ParseHostBindings(host map[string]interface{}) ParsedHostBindings {
	attributes := make(map[string]output.OutputExpression)
	listeners := make(map[string]string)
	properties := make(map[string]string)
	specialAttributes := view.R3HostSpecialAttributes{}

	for key, value := range host {
		matches := HOST_REG_EXP.FindStringSubmatch(key)

		if len(matches) == 0 {
			switch key {
			case "class":
				if strValue, ok := value.(string); ok {
					specialAttributes.ClassAttr = &strValue
				} else {
					panic("Class binding must be string")
				}
			case "style":
				if strValue, ok := value.(string); ok {
					specialAttributes.StyleAttr = &strValue
				} else {
					panic("Style binding must be string")
				}
			default:
				if strValue, ok := value.(string); ok {
					attributes[key] = output.NewLiteralExpr(strValue, output.InferredType, nil)
				} else if expr, ok := value.(output.OutputExpression); ok {
					attributes[key] = expr
				}
			}
		} else if len(matches) > int(HostBindingGroupBinding) && matches[HostBindingGroupBinding] != "" {
			if strValue, ok := value.(string); ok {
				properties[matches[HostBindingGroupBinding]] = strValue
			} else {
				panic("Property binding must be string")
			}
		} else if len(matches) > int(HostBindingGroupEvent) && matches[HostBindingGroupEvent] != "" {
			if strValue, ok := value.(string); ok {
				listeners[matches[HostBindingGroupEvent]] = strValue
			} else {
				panic("Event binding must be string")
			}
		}
	}

	return ParsedHostBindings{
		Attributes:        attributes,
		Listeners:         listeners,
		Properties:        properties,
		SpecialAttributes: specialAttributes,
	}
}

// verifyHostBindings verifies host bindings and returns the list of errors (if any)
func VerifyHostBindings(
	bindings ParsedHostBindings,
	sourceSpan *util.ParseSourceSpan,
) []*util.ParseError {
	lexer := expression_parser.NewLexer()
	parser := expression_parser.NewParser(lexer, false) // selectorlessEnabled = false
	elementRegistry := schema.NewDomElementSchemaRegistry()
	bindingParser := templateparser.NewBindingParser(parser, elementRegistry, []*util.ParseError{})
	bindingParser.CreateDirectiveHostEventAsts(bindings.Listeners, sourceSpan)
	bindingParser.CreateBoundHostProperties(bindings.Properties, sourceSpan)
	return bindingParser.GetErrors()
}

// compileStyles compiles styles using ShadowCss
func compileStyles(styles []string, selector string, hostSelector string) []string {
	shadowCss := css.NewShadowCss()
	result := make([]string, len(styles))
	for i, style := range styles {
		result[i] = shadowCss.ShimCssText(style, selector, hostSelector)
	}
	return result
}

// EncapsulateStyle encapsulates a CSS stylesheet with emulated view encapsulation
func EncapsulateStyle(style string, componentIdentifier *string) string {
	shadowCss := css.NewShadowCss()
	var selector, hostSelector string
	if componentIdentifier != nil {
		selector = strings.ReplaceAll(CONTENT_ATTR, COMPONENT_VARIABLE, *componentIdentifier)
		hostSelector = strings.ReplaceAll(HOST_ATTR, COMPONENT_VARIABLE, *componentIdentifier)
	} else {
		selector = CONTENT_ATTR
		hostSelector = HOST_ATTR
	}
	return shadowCss.ShimCssText(style, selector, hostSelector)
}

// createHostDirectivesType creates the type for host directives
func createHostDirectivesType(meta *view.R3DirectiveMetadata) output.Type {
	if meta.HostDirectives == nil || len(meta.HostDirectives) == 0 {
		return output.NoneType
	}

	literals := make([]output.OutputExpression, len(meta.HostDirectives))
	for i, hostMeta := range meta.HostDirectives {
		keys := []*output.LiteralMapEntry{
			output.NewLiteralMapEntry(
				"directive",
				output.NewTypeofExpr(hostMeta.Directive.Type, nil, nil),
				false,
			),
		}

		inputsMap := map[string]interface{}{}
		if hostMeta.Inputs != nil {
			for k, v := range hostMeta.Inputs {
				inputsMap[k] = v
			}
		}
		keys = append(keys, output.NewLiteralMapEntry(
			"inputs",
			stringMapAsLiteralExpression(inputsMap),
			false,
		))

		outputsMap := map[string]interface{}{}
		if hostMeta.Outputs != nil {
			for k, v := range hostMeta.Outputs {
				outputsMap[k] = v
			}
		}
		keys = append(keys, output.NewLiteralMapEntry(
			"outputs",
			stringMapAsLiteralExpression(outputsMap),
			false,
		))

		literals[i] = output.NewLiteralMapExpr(keys, nil, nil)
	}

	return output.NewExpressionType(
		output.NewLiteralArrayExpr(literals, nil, nil),
		output.TypeModifierNone,
		nil,
	)
}

// createHostDirectivesFeatureArg creates the feature argument for host directives
func createHostDirectivesFeatureArg(hostDirectives []view.R3HostDirectiveMetadata) output.OutputExpression {
	expressions := []output.OutputExpression{}
	hasForwardRef := false

	for _, current := range hostDirectives {
		if (current.Inputs == nil || len(current.Inputs) == 0) &&
			(current.Outputs == nil || len(current.Outputs) == 0) {
			expressions = append(expressions, current.Directive.Type)
		} else {
			keys := []*output.LiteralMapEntry{
				output.NewLiteralMapEntry("directive", current.Directive.Type, false),
			}

			if current.Inputs != nil && len(current.Inputs) > 0 {
				inputsLiteral := CreateHostDirectivesMappingArray(current.Inputs)
				if inputsLiteral != nil {
					keys = append(keys, output.NewLiteralMapEntry("inputs", inputsLiteral, false))
				}
			}

			if current.Outputs != nil && len(current.Outputs) > 0 {
				outputsLiteral := CreateHostDirectivesMappingArray(current.Outputs)
				if outputsLiteral != nil {
					keys = append(keys, output.NewLiteralMapEntry("outputs", outputsLiteral, false))
				}
			}

			expressions = append(expressions, output.NewLiteralMapExpr(keys, nil, nil))
		}

		if current.IsForwardReference {
			hasForwardRef = true
		}
	}

	if hasForwardRef {
		return output.NewFunctionExpr(
			[]*output.FnParam{},
			[]output.OutputStatement{
				output.NewReturnStatement(
					output.NewLiteralArrayExpr(expressions, nil, nil),
					nil,
					nil,
				),
			},
			nil,
			nil,
			nil,
		)
	}
	return output.NewLiteralArrayExpr(expressions, nil, nil)
}

// CreateHostDirectivesMappingArray converts an input/output mapping object literal into an array
func CreateHostDirectivesMappingArray(mapping map[string]string) *output.LiteralArrayExpr {
	elements := []output.OutputExpression{}

	for publicName, aliasName := range mapping {
		elements = append(elements,
			output.NewLiteralExpr(publicName, output.InferredType, nil),
			output.NewLiteralExpr(aliasName, output.InferredType, nil),
		)
	}

	if len(elements) > 0 {
		return output.NewLiteralArrayExpr(elements, nil, nil)
	}
	return nil
}

// CompileDeferResolverFunction compiles the dependency resolver function for a defer block
func CompileDeferResolverFunction(
	meta *view.R3DeferResolverFunctionMetadata,
) *output.ArrowFunctionExpr {
	depExpressions := []output.OutputExpression{}

	if meta.Mode == view.DeferBlockDepsEmitModePerBlock {
		for _, dep := range meta.PerBlockDependencies {
			if dep.IsDeferrable {
				// Callback function, e.g. `m () => m.MyCmp;`.
				var propName string
				if dep.IsDefaultImport {
					propName = "default"
				} else {
					propName = dep.SymbolName
				}
				innerFn := output.NewArrowFunctionExpr(
					[]*output.FnParam{
						output.NewFnParam("m", output.DynamicType),
					},
					output.NewReadPropExpr(
						output.NewReadVarExpr("m", output.DynamicType, nil),
						propName,
						output.DynamicType,
						nil,
					),
					output.DynamicType,
					nil,
				)

				// Dynamic import, e.g. `import('./a').then(...)`.
				importExpr := output.NewInvokeFunctionExpr(
					output.NewReadPropExpr(
						output.NewDynamicImportExpr(dep.ImportPath, nil, nil),
						"then",
						output.DynamicType,
						nil,
					),
					[]output.OutputExpression{innerFn},
					nil,
					nil,
					false,
				)
				depExpressions = append(depExpressions, importExpr)
			} else {
				// Non-deferrable symbol, just use a reference to the type
				depExpressions = append(depExpressions, dep.TypeReference)
			}
		}
	} else {
		for _, dep := range meta.PerComponentDependencies {
			// Callback function, e.g. `m () => m.MyCmp;`.
			var propName string
			if dep.IsDefaultImport {
				propName = "default"
			} else {
				propName = dep.SymbolName
			}
			innerFn := output.NewArrowFunctionExpr(
				[]*output.FnParam{
					output.NewFnParam("m", output.DynamicType),
				},
				output.NewReadPropExpr(
					output.NewReadVarExpr("m", output.DynamicType, nil),
					propName,
					output.DynamicType,
					nil,
				),
				output.DynamicType,
				nil,
			)

			// Dynamic import, e.g. `import('./a').then(...)`.
			importExpr := output.NewInvokeFunctionExpr(
				output.NewReadPropExpr(
					output.NewDynamicImportExpr(dep.ImportPath, nil, nil),
					"then",
					output.DynamicType,
					nil,
				),
				[]output.OutputExpression{innerFn},
				nil,
				nil,
				false,
			)
			depExpressions = append(depExpressions, importExpr)
		}
	}

	return output.NewArrowFunctionExpr(
		[]*output.FnParam{},
		output.NewLiteralArrayExpr(depExpressions, nil, nil),
		output.DynamicType,
		nil,
	)
}
