package view

import (
	"fmt"
	"regexp"
	"strings"

	"ngc-go/packages/compiler/src/core"
	"ngc-go/packages/compiler/src/css"
	"ngc-go/packages/compiler/src/expression_parser"
	"ngc-go/packages/compiler/src/ml_parser"
	"ngc-go/packages/compiler/src/output"
	"ngc-go/packages/compiler/src/render3"
	"ngc-go/packages/compiler/src/util"
)

// UNSAFE_OBJECT_KEY_NAME_REGEXP checks whether an object key contains potentially unsafe chars
var UNSAFE_OBJECT_KEY_NAME_REGEXP = regexp.MustCompile(`[-.]`)

// TEMPORARY_NAME is the name of the temporary to use during data binding
const TEMPORARY_NAME = "_t"

// CONTEXT_NAME is the name of the context parameter passed into a template function
const CONTEXT_NAME = "ctx"

// RENDER_FLAGS is the name of the RenderFlag passed into a template function
const RENDER_FLAGS = "rf"

// TemporaryAllocatorFunc is a function that allocates a temporary variable
type TemporaryAllocatorFunc func() *output.ReadVarExpr

// TemporaryAllocator creates an allocator for a temporary variable.
// A variable declaration is added to the statements the first time the allocator is invoked.
func TemporaryAllocator(
	pushStatement func(output.OutputStatement),
	name string,
) TemporaryAllocatorFunc {
	var temp *output.ReadVarExpr
	return func() *output.ReadVarExpr {
		if temp == nil {
			pushStatement(output.NewDeclareVarStmt(
				TEMPORARY_NAME,
				nil,
				output.DynamicType,
				output.StmtModifierNone,
				nil,
				nil,
			))
			temp = output.NewReadVarExpr(name, output.DynamicType, nil)
		}
		return temp
	}
}

// Invalid throws an error for invalid visitor state
func Invalid(visitor interface{}, arg interface{}) {
	var visitorType, argType string
	if visitor != nil {
		visitorType = "Visitor"
	}
	if arg != nil {
		argType = "unknown"
		// Try to determine the type
		switch arg.(type) {
		case output.OutputExpression:
			argType = "Expression"
		case output.OutputStatement:
			argType = "Statement"
		case interface{ SourceSpan() *util.ParseSourceSpan }:
			argType = "Node"
		}
	}
	panic(fmt.Errorf("Invalid state: Visitor %s doesn't handle %s", visitorType, argType))
}

// AsLiteral converts a value to a literal expression
func AsLiteral(value interface{}) output.OutputExpression {
	if arr, ok := value.([]interface{}); ok {
		literals := make([]output.OutputExpression, len(arr))
		for i, v := range arr {
			literals[i] = AsLiteral(v)
		}
		return output.NewLiteralArrayExpr(literals, nil, nil)
	}
	return output.NewLiteralExpr(value, output.InferredType, nil)
}

// DirectiveBindingValue represents a value in a directive binding map
type DirectiveBindingValue struct {
	ClassPropertyName   string
	BindingPropertyName string
	TransformFunction   output.OutputExpression
	IsSignal            bool
}

// ConditionallyCreateDirectiveBindingLiteral serializes inputs and outputs for defineDirective and defineComponent.
// This will attempt to generate optimized data structures to minimize memory or file size of fully compiled applications.
func ConditionallyCreateDirectiveBindingLiteral(
	bindingMap map[string]interface{}, // string | DirectiveBindingValue
	forInputs bool,
) output.OutputExpression {
	if len(bindingMap) == 0 {
		return nil
	}

	entries := []*output.LiteralMapEntry{}
	for key, value := range bindingMap {
		var declaredName, publicName, minifiedName string
		var expressionValue output.OutputExpression

		if strValue, ok := value.(string); ok {
			// canonical syntax: `dirProp: publicProp`
			declaredName = key
			minifiedName = key
			publicName = strValue
			expressionValue = AsLiteral(publicName)
		} else if bindingValue, ok := value.(*DirectiveBindingValue); ok {
			minifiedName = key
			declaredName = bindingValue.ClassPropertyName
			publicName = bindingValue.BindingPropertyName

			differentDeclaringName := publicName != declaredName
			hasDecoratorInputTransform := bindingValue.TransformFunction != nil
			flags := core.InputFlagsNone

			// Build up input flags
			if bindingValue.IsSignal {
				flags |= core.InputFlagsSignalBased
			}
			if hasDecoratorInputTransform {
				flags |= core.InputFlagsHasDecoratorInputTransform
			}

			// Inputs, compared to outputs, will track their declared name (for `ngOnChanges`), support
			// decorator input transform functions, or store flag information if there is any.
			if forInputs && (differentDeclaringName || hasDecoratorInputTransform || flags != core.InputFlagsNone) {
				result := []output.OutputExpression{
					output.NewLiteralExpr(int(flags), nil, nil),
					AsLiteral(publicName),
				}

				if differentDeclaringName || hasDecoratorInputTransform {
					result = append(result, AsLiteral(declaredName))

					if hasDecoratorInputTransform {
						result = append(result, bindingValue.TransformFunction)
					}
				}

				expressionValue = output.NewLiteralArrayExpr(result, nil, nil)
			} else {
				expressionValue = AsLiteral(publicName)
			}
		} else {
			// Fallback: treat as string
			declaredName = key
			minifiedName = key
			publicName = key
			expressionValue = AsLiteral(publicName)
		}

		// put quotes around keys that contain potentially unsafe characters
		quoted := UNSAFE_OBJECT_KEY_NAME_REGEXP.MatchString(minifiedName)
		entries = append(entries, output.NewLiteralMapEntry(minifiedName, expressionValue, quoted))
	}

	return output.NewLiteralMapExpr(entries, nil, nil)
}

// DefinitionMapEntry represents an entry in a DefinitionMap
type DefinitionMapEntry struct {
	Key    string
	Quoted bool
	Value  output.OutputExpression
}

// DefinitionMap is a representation for an object literal used during codegen of definition objects.
// The generic type T allows to reference a documented type of the generated structure, such that the
// property names that are set can be resolved to their documented declaration.
type DefinitionMap struct {
	Values []DefinitionMapEntry
}

// NewDefinitionMap creates a new DefinitionMap
func NewDefinitionMap() *DefinitionMap {
	return &DefinitionMap{
		Values: []DefinitionMapEntry{},
	}
}

// Set sets a key-value pair in the map. If the key already exists, it updates the value.
// If value is nil, the key is not added.
func (dm *DefinitionMap) Set(key string, value output.OutputExpression) {
	if value == nil {
		return
	}
	// Find existing entry
	for i := range dm.Values {
		if dm.Values[i].Key == key {
			dm.Values[i].Value = value
			return
		}
	}
	// Add new entry
	dm.Values = append(dm.Values, DefinitionMapEntry{
		Key:    key,
		Quoted: false,
		Value:  value,
	})
}

// ToLiteralMap converts the DefinitionMap to a LiteralMapExpr
func (dm *DefinitionMap) ToLiteralMap() *output.LiteralMapExpr {
	entries := make([]*output.LiteralMapEntry, len(dm.Values))
	for i, entry := range dm.Values {
		entries[i] = output.NewLiteralMapEntry(entry.Key, entry.Value, entry.Quoted)
	}
	return output.NewLiteralMapExpr(entries, nil, nil)
}

// IsI18nAttribute checks if an attribute name is an i18n attribute
func IsI18nAttribute(name string) bool {
	const I18N_ATTR = "i18n"
	const I18N_ATTR_PREFIX = "i18n-"
	return name == I18N_ATTR || strings.HasPrefix(name, I18N_ATTR_PREFIX)
}

// Node represents a node that can be used for CSS selector creation
type Node interface {
	SourceSpan() *util.ParseSourceSpan
}

// CreateCssSelectorFromNode creates a CssSelector from an AST node
func CreateCssSelectorFromNode(node Node) *css.CssSelector {
	var elementName string

	// Try to get element name from different node types
	switch n := node.(type) {
	case interface{ GetName() string }:
		elementName = n.GetName()
	case interface{ GetTagName() *string }:
		if tagName := n.GetTagName(); tagName != nil {
			// For inline templates (e.g., *ngFor on a div), the tagName is the wrapped element's name.
			// But for directive matching, we should use "ng-template" or empty to match template selectors.
			// Check if this is an inline template by seeing if it's not actually an ng-template element
			if *tagName != "ng-template" {
				// This is an inline template like <div *ngFor>, use empty element name for matching
				elementName = ""
			} else {
				elementName = "ng-template"
			}
		} else {
			elementName = "ng-template"
		}
	default:
		elementName = ""
	}

	attributes := GetAttrsForDirectiveMatching(node)
	cssSelector := css.NewCssSelector()
	_, elementNameNoNs := ml_parser.SplitNsName(elementName, false)

	cssSelector.SetElement(elementNameNoNs)

	fmt.Printf("[DEBUG] CreateCssSelectorFromNode: elementNameNoNs=%q, attributes=%v\n", elementNameNoNs, attributes)

	// Sort attribute names for consistent order (Go map iteration is random)
	attrNames := make([]string, 0, len(attributes))
	for name := range attributes {
		attrNames = append(attrNames, name)
	}
	// Sort alphabetically to ensure consistent order
	for i := 0; i < len(attrNames); i++ {
		for j := i + 1; j < len(attrNames); j++ {
			if attrNames[i] > attrNames[j] {
				attrNames[i], attrNames[j] = attrNames[j], attrNames[i]
			}
		}
	}

	for _, name := range attrNames {
		value := attributes[name]
		_, nameNoNs := ml_parser.SplitNsName(name, false)
		fmt.Printf("[DEBUG] CreateCssSelectorFromNode: adding attribute name=%q, value=%q\n", nameNoNs, value)
		cssSelector.AddAttribute(nameNoNs, value)
		if strings.ToLower(name) == "class" {
			classes := strings.Fields(value)
			for _, className := range classes {
				cssSelector.AddClassName(className)
			}
		}
	}

	fmt.Printf("[DEBUG] CreateCssSelectorFromNode: final cssSelector=%v\n", cssSelector)
	return cssSelector
}

// GetAttrsForDirectiveMatching extracts a map of properties to values for a given element or template node,
// which can be used by the directive matching machinery.
func GetAttrsForDirectiveMatching(elOrTpl Node) map[string]string {
	attributesMap := make(map[string]string)

	// Handle Template nodes (inline templates like *ngFor)
	if template, ok := elOrTpl.(interface {
		GetTagName() *string
		GetAttributes() []*render3.TextAttribute
		GetInputs() []*render3.BoundAttribute
	}); ok {
		if tagName := template.GetTagName(); tagName != nil && *tagName != "ng-template" {
			// For inline templates (*ngFor, *ngIf, etc.), use the typed fields
			fmt.Printf("[DEBUG] GetAttrsForDirectiveMatching: found inline template with tagName=%q\n", *tagName)
			// Get text attributes
			attrs := template.GetAttributes()
			fmt.Printf("[DEBUG] GetAttrsForDirectiveMatching: template.GetAttributes() count=%d\n", len(attrs))
			for i, attr := range attrs {
				name := attr.Name
				fmt.Printf("[DEBUG] GetAttrsForDirectiveMatching: attr[%d].Name=%q, Value=%q\n", i, name, attr.Value)
				if !IsI18nAttribute(name) {
					attributesMap[name] = attr.Value
				}
			}
			// Get inputs (bound attributes)
			inputs := template.GetInputs()
			fmt.Printf("[DEBUG] GetAttrsForDirectiveMatching: template.GetInputs() count=%d\n", len(inputs))
			for i, input := range inputs {
				fmt.Printf("[DEBUG] GetAttrsForDirectiveMatching: input[%d].Name=%q, Type=%d\n", i, input.Name, input.Type)
				if input.Type == expression_parser.BindingTypeProperty || input.Type == expression_parser.BindingTypeTwoWay {
					attributesMap[input.Name] = ""
				}
			}
			fmt.Printf("[DEBUG] GetAttrsForDirectiveMatching: final attributesMap=%v\n", attributesMap)
			return attributesMap
		}
	}

	// Handle Element nodes
	if element, ok := elOrTpl.(interface {
		GetName() string
		GetAttributes() []*render3.TextAttribute
		GetInputs() []*render3.BoundAttribute
		GetOutputs() []*render3.BoundEvent
	}); ok {
		fmt.Printf("[DEBUG] GetAttrsForDirectiveMatching: found Element with name=%q\n", element.GetName())

		// Get text attributes
		attrs := element.GetAttributes()
		fmt.Printf("[DEBUG] GetAttrsForDirectiveMatching: element.GetAttributes() count=%d\n", len(attrs))
		for i, attr := range attrs {
			name := attr.Name
			fmt.Printf("[DEBUG] GetAttrsForDirectiveMatching: attr[%d].Name=%q, Value=%q\n", i, name, attr.Value)
			if !IsI18nAttribute(name) {
				attributesMap[name] = attr.Value
			}
		}

		// Get inputs (bound attributes)
		inputs := element.GetInputs()
		fmt.Printf("[DEBUG] GetAttrsForDirectiveMatching: element.GetInputs() count=%d\n", len(inputs))
		for i, input := range inputs {
			fmt.Printf("[DEBUG] GetAttrsForDirectiveMatching: input[%d].Name=%q, Type=%d\n", i, input.Name, input.Type)
			if input.Type == expression_parser.BindingTypeProperty || input.Type == expression_parser.BindingTypeTwoWay {
				attributesMap[input.Name] = ""
			}
		}

		// Get outputs (bound events)
		outputs := element.GetOutputs()
		fmt.Printf("[DEBUG] GetAttrsForDirectiveMatching: element.GetOutputs() count=%d\n", len(outputs))
		for i, output := range outputs {
			fmt.Printf("[DEBUG] GetAttrsForDirectiveMatching: output[%d].Name=%q\n", i, output.Name)
			attributesMap[output.Name] = ""
		}

		fmt.Printf("[DEBUG] GetAttrsForDirectiveMatching: final attributesMap=%v\n", attributesMap)
		return attributesMap
	}

	return attributesMap
}
