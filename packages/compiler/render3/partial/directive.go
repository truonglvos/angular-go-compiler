package partial

import (
	"strings"

	"ngc-go/packages/compiler/output"
	"ngc-go/packages/compiler/render3"
	r3_identifiers "ngc-go/packages/compiler/render3/r3_identifiers"
	"ngc-go/packages/compiler/render3/view"
	view_compiler "ngc-go/packages/compiler/render3/view/compiler"
)

// CompileDeclareDirectiveFromMetadata compiles a directive declaration defined by the `R3DirectiveMetadata`.
func CompileDeclareDirectiveFromMetadata(
	meta *view.R3DirectiveMetadata,
) render3.R3CompiledExpression {
	definitionMap := CreateDirectiveDefinitionMap(meta)

	expression := output.NewInvokeFunctionExpr(
		output.NewExternalExpr(r3_identifiers.DeclareDirective, nil, nil, nil),
		[]output.OutputExpression{definitionMap.ToLiteralMap()},
		nil,
		nil,
		false,
	)
	typ := createDirectiveType(meta)

	return render3.R3CompiledExpression{
		Expression: expression,
		Type:       typ,
		Statements: []output.OutputStatement{},
	}
}

// CreateDirectiveDefinitionMap gathers the declaration fields for a directive into a `DefinitionMap`.
// This allows for reusing this logic for components, as they extend the directive metadata.
func CreateDirectiveDefinitionMap(
	meta *view.R3DirectiveMetadata,
) *view.DefinitionMap {
	definitionMap := view.NewDefinitionMap()
	minVersion := getMinimumVersionForPartialOutput(meta)

	definitionMap.Set("minVersion", output.NewLiteralExpr(minVersion, output.InferredType, nil))
	definitionMap.Set("version", output.NewLiteralExpr("0.0.0-PLACEHOLDER", output.InferredType, nil))

	// e.g. `type: MyDirective`
	definitionMap.Set("type", meta.Type.Value)

	if meta.IsStandalone {
		definitionMap.Set("isStandalone", output.NewLiteralExpr(meta.IsStandalone, output.InferredType, nil))
	}
	if meta.IsSignal {
		definitionMap.Set("isSignal", output.NewLiteralExpr(meta.IsSignal, output.InferredType, nil))
	}

	// e.g. `selector: 'some-dir'`
	if meta.Selector != nil {
		definitionMap.Set("selector", output.NewLiteralExpr(*meta.Selector, output.InferredType, nil))
	}

	if needsNewInputPartialOutput(meta) {
		definitionMap.Set("inputs", createInputsPartialMetadata(meta.Inputs))
	} else {
		definitionMap.Set("inputs", legacyInputsPartialMetadata(meta.Inputs))
	}
	// Convert map[string]string to map[string]interface{} for ConditionallyCreateDirectiveBindingLiteral
	outputsMap := make(map[string]interface{})
	for k, v := range meta.Outputs {
		outputsMap[k] = v
	}
	definitionMap.Set("outputs", view.ConditionallyCreateDirectiveBindingLiteral(outputsMap, false))

	definitionMap.Set("host", compileHostMetadata(meta.Host))

	if meta.Providers != nil {
		definitionMap.Set("providers", *meta.Providers)
	}

	if len(meta.Queries) > 0 {
		queries := make([]output.OutputExpression, len(meta.Queries))
		for i, query := range meta.Queries {
			queries[i] = compileQuery(query)
		}
		definitionMap.Set("queries", output.NewLiteralArrayExpr(queries, nil, nil))
	}

	if len(meta.ViewQueries) > 0 {
		viewQueries := make([]output.OutputExpression, len(meta.ViewQueries))
		for i, query := range meta.ViewQueries {
			viewQueries[i] = compileQuery(query)
		}
		definitionMap.Set("viewQueries", output.NewLiteralArrayExpr(viewQueries, nil, nil))
	}

	if meta.ExportAs != nil && len(meta.ExportAs) > 0 {
		definitionMap.Set("exportAs", view.AsLiteral(meta.ExportAs))
	}

	if meta.UsesInheritance {
		definitionMap.Set("usesInheritance", output.NewLiteralExpr(true, output.InferredType, nil))
	}

	if meta.Lifecycle.UsesOnChanges {
		definitionMap.Set("usesOnChanges", output.NewLiteralExpr(true, output.InferredType, nil))
	}

	if meta.HostDirectives != nil && len(meta.HostDirectives) > 0 {
		definitionMap.Set("hostDirectives", createHostDirectives(meta.HostDirectives))
	}

	definitionMap.Set("ngImport", output.NewExternalExpr(r3_identifiers.Core, nil, nil, nil))

	return definitionMap
}

// getMinimumVersionForPartialOutput determines the minimum linker version for the partial output
// generated for this directive.
//
// Every time we make a breaking change to the declaration interface or partial-linker
// behavior, we must update the minimum versions to prevent old partial-linkers from
// incorrectly processing the declaration.
//
// NOTE: Do not include any prerelease in these versions as they are ignored.
func getMinimumVersionForPartialOutput(meta *view.R3DirectiveMetadata) string {
	// We are starting with the oldest minimum version that can work for common
	// directive partial compilation output. As we discover usages of new features
	// that require a newer partial output emit, we bump the `minVersion`. Our goal
	// is to keep libraries as much compatible with older linker versions as possible.
	minVersion := "14.0.0"

	// Note: in order to allow consuming Angular libraries that have been compiled with 16.1+ in
	// Angular 16.0, we only force a minimum version of 16.1 if input transform feature as introduced
	// in 16.1 is actually used.
	hasDecoratorTransformFunctions := false
	for _, input := range meta.Inputs {
		if input.TransformFunction != nil {
			hasDecoratorTransformFunctions = true
			break
		}
	}
	if hasDecoratorTransformFunctions {
		minVersion = "16.1.0"
	}

	// If there are input flags and we need the new emit, use the actual minimum version,
	// where this was introduced. i.e. in 17.1.0
	// TODO(legacy-partial-output-inputs): Remove in v18.
	if needsNewInputPartialOutput(meta) {
		minVersion = "17.1.0"
	}

	// If there are signal-based queries, partial output generates an extra field
	// that should be parsed by linkers. Ensure a proper minimum linker version.
	hasSignalQueries := false
	for _, q := range meta.Queries {
		if q.IsSignal {
			hasSignalQueries = true
			break
		}
	}
	if !hasSignalQueries {
		for _, q := range meta.ViewQueries {
			if q.IsSignal {
				hasSignalQueries = true
				break
			}
		}
	}
	if hasSignalQueries {
		minVersion = "17.2.0"
	}

	return minVersion
}

// needsNewInputPartialOutput gets whether the given directive needs the new input partial output structure
// that can hold additional metadata like `isRequired`, `isSignal` etc.
func needsNewInputPartialOutput(meta *view.R3DirectiveMetadata) bool {
	for _, input := range meta.Inputs {
		if input.IsSignal {
			return true
		}
	}
	return false
}

// compileQuery compiles the metadata of a single query into its partial declaration form as declared
// by `R3DeclareQueryMetadata`.
func compileQuery(query view.R3QueryMetadata) output.OutputExpression {
	meta := view.NewDefinitionMap()
	meta.Set("propertyName", output.NewLiteralExpr(query.PropertyName, output.InferredType, nil))
	if query.First {
		meta.Set("first", output.NewLiteralExpr(true, output.InferredType, nil))
	}

	var predicate output.OutputExpression
	if arr, ok := query.Predicate.([]string); ok {
		predicate = view.AsLiteral(arr)
	} else {
		predicate = render3.ConvertFromMaybeForwardRefExpression(query.Predicate.(render3.MaybeForwardRefExpression))
	}
	meta.Set("predicate", predicate)

	if !query.EmitDistinctChangesOnly {
		// `emitDistinctChangesOnly` is special because we expect it to be `true`.
		// Therefore we explicitly emit the field, and explicitly place it only when it's `false`.
		meta.Set("emitDistinctChangesOnly", output.NewLiteralExpr(false, output.InferredType, nil))
	} else {
		// The linker will assume that an absent `emitDistinctChangesOnly` flag is by default `true`.
	}
	if query.Descendants {
		meta.Set("descendants", output.NewLiteralExpr(true, output.InferredType, nil))
	}
	if query.Read != nil {
		meta.Set("read", *query.Read)
	}
	if query.Static {
		meta.Set("static", output.NewLiteralExpr(true, output.InferredType, nil))
	}
	if query.IsSignal {
		meta.Set("isSignal", output.NewLiteralExpr(true, output.InferredType, nil))
	}
	return meta.ToLiteralMap()
}

// compileHostMetadata compiles the host metadata into its partial declaration form as declared
// in `R3DeclareDirectiveMetadata['host']`
func compileHostMetadata(meta view.R3HostMetadata) output.OutputExpression {
	hostMetadata := view.NewDefinitionMap()
	hostMetadata.Set(
		"attributes",
		ToOptionalLiteralMap(meta.Attributes, func(expression output.OutputExpression) output.OutputExpression {
			return expression
		}),
	)
	hostMetadata.Set("listeners", ToOptionalLiteralMap(meta.Listeners, func(value string) output.OutputExpression {
		return output.NewLiteralExpr(value, output.InferredType, nil)
	}))
	hostMetadata.Set("properties", ToOptionalLiteralMap(meta.Properties, func(value string) output.OutputExpression {
		return output.NewLiteralExpr(value, output.InferredType, nil)
	}))

	if meta.SpecialAttributes.StyleAttr != nil {
		hostMetadata.Set("styleAttribute", output.NewLiteralExpr(*meta.SpecialAttributes.StyleAttr, output.InferredType, nil))
	}
	if meta.SpecialAttributes.ClassAttr != nil {
		hostMetadata.Set("classAttribute", output.NewLiteralExpr(*meta.SpecialAttributes.ClassAttr, output.InferredType, nil))
	}

	if len(hostMetadata.Values) > 0 {
		return hostMetadata.ToLiteralMap()
	} else {
		return nil
	}
}

// createHostDirectives creates host directives metadata
func createHostDirectives(
	hostDirectives []view.R3HostDirectiveMetadata,
) output.OutputExpression {
	expressions := make([]output.OutputExpression, len(hostDirectives))
	for i, current := range hostDirectives {
		keys := []*output.LiteralMapEntry{
			output.NewLiteralMapEntry(
				"directive",
				func() output.OutputExpression {
					if current.IsForwardReference {
						return render3.GenerateForwardRef(current.Directive.Value)
					}
					return current.Directive.Value
				}(),
				false,
			),
		}

		var inputsLiteral output.OutputExpression
		if current.Inputs != nil && len(current.Inputs) > 0 {
			inputsLiteral = view_compiler.CreateHostDirectivesMappingArray(current.Inputs)
		}
		var outputsLiteral output.OutputExpression
		if current.Outputs != nil && len(current.Outputs) > 0 {
			outputsLiteral = view_compiler.CreateHostDirectivesMappingArray(current.Outputs)
		}

		if inputsLiteral != nil {
			keys = append(keys, output.NewLiteralMapEntry("inputs", inputsLiteral, false))
		}

		if outputsLiteral != nil {
			keys = append(keys, output.NewLiteralMapEntry("outputs", outputsLiteral, false))
		}

		expressions[i] = output.NewLiteralMapExpr(keys, nil, nil)
	}

	// If there's a forward reference, we generate a `function() { return [{directive: HostDir}] }`,
	// otherwise we can save some bytes by using a plain array, e.g. `[{directive: HostDir}]`.
	return output.NewLiteralArrayExpr(expressions, nil, nil)
}

// createInputsPartialMetadata generates partial output metadata for inputs of a directive.
//
// The generated structure is expected to match `R3DeclareDirectiveFacade['inputs']`.
func createInputsPartialMetadata(inputs map[string]view.R3InputMetadata) output.OutputExpression {
	if len(inputs) == 0 {
		return nil
	}

	keys := make([]*output.LiteralMapEntry, 0, len(inputs))
	for declaredName, value := range inputs {
		valueMap := []*output.LiteralMapEntry{
			output.NewLiteralMapEntry("classPropertyName", view.AsLiteral(value.ClassPropertyName), false),
			output.NewLiteralMapEntry("publicName", view.AsLiteral(value.BindingPropertyName), false),
			output.NewLiteralMapEntry("isSignal", view.AsLiteral(value.IsSignal), false),
			output.NewLiteralMapEntry("isRequired", view.AsLiteral(value.Required), false),
		}

		var transformFunction output.OutputExpression
		if value.TransformFunction != nil {
			transformFunction = *value.TransformFunction
		} else {
			transformFunction = output.NewLiteralExpr(nil, output.InferredType, nil)
		}
		valueMap = append(valueMap, output.NewLiteralMapEntry("transformFunction", transformFunction, false))

		// put quotes around keys that contain potentially unsafe characters
		quoted := view.UNSAFE_OBJECT_KEY_NAME_REGEXP.MatchString(declaredName)
		keys = append(keys, output.NewLiteralMapEntry(
			declaredName,
			output.NewLiteralMapExpr(valueMap, nil, nil),
			quoted,
		))
	}

	return output.NewLiteralMapExpr(keys, nil, nil)
}

// legacyInputsPartialMetadata generates pre v18 legacy partial output for inputs.
//
// Previously, inputs did not capture metadata like `isSignal` in the partial compilation output.
// To enable capturing such metadata, we restructured how input metadata is communicated in the
// partial output. This would make libraries incompatible with older Angular FW versions where the
// linker would not know how to handle this new "format". For this reason, if we know this metadata
// does not need to be captured- we fall back to the old format. This is what this function
// generates.
//
// See:
// https://github.com/angular/angular/blob/d4b423690210872b5c32a322a6090beda30b05a3/packages/core/src/compiler/compiler_facade_interface.ts#L197-L199
func legacyInputsPartialMetadata(inputs map[string]view.R3InputMetadata) output.OutputExpression {
	// TODO(legacy-partial-output-inputs): Remove function in v18.

	if len(inputs) == 0 {
		return nil
	}

	keys := make([]*output.LiteralMapEntry, 0, len(inputs))
	for declaredName, value := range inputs {
		publicName := value.BindingPropertyName
		differentDeclaringName := publicName != declaredName
		var result output.OutputExpression

		if differentDeclaringName || value.TransformFunction != nil {
			values := []output.OutputExpression{
				view.AsLiteral(publicName),
				view.AsLiteral(declaredName),
			}
			if value.TransformFunction != nil {
				values = append(values, *value.TransformFunction)
			}
			result = output.NewLiteralArrayExpr(values, nil, nil)
		} else {
			result = view.AsLiteral(publicName)
		}

		// put quotes around keys that contain potentially unsafe characters
		quoted := view.UNSAFE_OBJECT_KEY_NAME_REGEXP.MatchString(declaredName)
		keys = append(keys, output.NewLiteralMapEntry(declaredName, result, quoted))
	}

	return output.NewLiteralMapExpr(keys, nil, nil)
}

// createDirectiveType creates the type specification from the directive meta
// This is a copy of the private function from view/compiler/compiler.go
// to maintain 1:1 logic with TypeScript where it's exported
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

// stringAsType creates a type from a string
func stringAsType(str string) output.Type {
	return output.NewExpressionType(
		output.NewLiteralExpr(str, output.InferredType, nil),
		output.TypeModifierNone,
		nil,
	)
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

// convertStringMap converts map[string]string to map[string]interface{}
func convertStringMap(m map[string]string) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m {
		result[k] = v
	}
	return result
}

// stringMapAsLiteralExpression creates a literal expression from a string map
func stringMapAsLiteralExpression(m map[string]interface{}) output.OutputExpression {
	mapValues := []*output.LiteralMapEntry{}
	for key, value := range m {
		var literalValue output.OutputExpression
		if str, ok := value.(string); ok {
			literalValue = output.NewLiteralExpr(str, output.InferredType, nil)
		} else {
			literalValue = output.NewLiteralExpr(value, output.InferredType, nil)
		}
		mapValues = append(mapValues, output.NewLiteralMapEntry(key, literalValue, true))
	}
	return output.NewLiteralMapExpr(mapValues, nil, nil)
}

// createHostDirectivesType creates the type for host directives
func createHostDirectivesType(meta *view.R3DirectiveMetadata) output.Type {
	if meta.HostDirectives == nil || len(meta.HostDirectives) == 0 {
		return output.NoneType
	}
	// For partial compilation, we don't need to create the full type structure
	// as it's handled by the linker
	return output.NoneType
}

