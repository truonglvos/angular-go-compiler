package phases

import (
	"regexp"
	"strings"

	"ngc-go/packages/compiler/src/core"
	"ngc-go/packages/compiler/src/output"
	"ngc-go/packages/compiler/src/template/pipeline/ir"
	"ngc-go/packages/compiler/src/template/pipeline/ir/src/expression"
	ir_operation "ngc-go/packages/compiler/src/template/pipeline/ir/src/operations"
	ops_create "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/create"

	pipeline "ngc-go/packages/compiler/src/template/pipeline/src/compilation"
)

// Any changes here should be ported to the Angular Domino fork.
// https://github.com/angular/domino/blob/main/lib/style_parser.js

const (
	charOpenParen   = 40
	charCloseParen  = 41
	charColon       = 58
	charSemicolon   = 59
	charBackSlash   = 92
	charQuoteNone   = 0 // indicating we are not inside a quote
	charQuoteDouble = 34
	charQuoteSingle = 39
)

// Parse parses string representation of a style and converts it into object literal.
//
// @param value string representation of style as used in the `style` attribute in HTML.
//
//	Example: `color: red; height: auto`.
//
// @returns An array of style property name and value pairs, e.g. `['color', 'red', 'height', 'auto']`
func Parse(value string) []string {
	// we use a string array here instead of a string map
	// because a string-map is not guaranteed to retain the
	// order of the entries whereas a string array can be
	// constructed in a [key, value, key, value] format.
	styles := []string{}

	i := 0
	parenDepth := 0
	quote := charQuoteNone
	valueStart := 0
	propStart := 0
	var currentProp *string
	for i < len(value) {
		token := int(value[i])
		i++
		switch token {
		case charOpenParen:
			parenDepth++
		case charCloseParen:
			parenDepth--
		case charQuoteSingle:
			// valueStart needs to be there since prop values don't
			// have quotes in CSS
			if quote == charQuoteNone {
				quote = charQuoteSingle
			} else if quote == charQuoteSingle {
				// In TypeScript: value.charCodeAt(i - 1) !== Char.BackSlash
				// i-1 is the current quote character, but we need to check the character before it
				// Since i was incremented, i-2 is the character before the quote
				if i-2 < 0 || int(value[i-2]) != charBackSlash {
					quote = charQuoteNone
				}
			}
		case charQuoteDouble:
			// same logic as above
			if quote == charQuoteNone {
				quote = charQuoteDouble
			} else if quote == charQuoteDouble {
				// In TypeScript: value.charCodeAt(i - 1) !== Char.BackSlash
				// i-1 is the current quote character, but we need to check the character before it
				// Since i was incremented, i-2 is the character before the quote
				if i-2 < 0 || int(value[i-2]) != charBackSlash {
					quote = charQuoteNone
				}
			}
		case charColon:
			if currentProp == nil && parenDepth == 0 && quote == charQuoteNone {
				// TODO: Do not hyphenate CSS custom property names like: `--intentionallyCamelCase`
				propName := hyphenateStyleProperty(strings.TrimSpace(value[propStart : i-1]))
				currentProp = &propName
				valueStart = i
			}
		case charSemicolon:
			if currentProp != nil && valueStart > 0 && parenDepth == 0 && quote == charQuoteNone {
				styleVal := strings.TrimSpace(value[valueStart : i-1])
				styles = append(styles, *currentProp, styleVal)
				propStart = i
				valueStart = 0
				currentProp = nil
			}
		}
	}

	if currentProp != nil && valueStart > 0 {
		styleVal := strings.TrimSpace(value[valueStart:])
		styles = append(styles, *currentProp, styleVal)
	}

	return styles
}

// hyphenateStyleProperty converts camelCase to kebab-case
// This matches the TypeScript implementation exactly
func hyphenateStyleProperty(value string) string {
	// Match pattern: [a-z][A-Z] and replace with [a-z]-[A-Z]
	re := regexp.MustCompile(`([a-z])([A-Z])`)
	result := re.ReplaceAllStringFunc(value, func(match string) string {
		runes := []rune(match)
		return string(runes[0]) + "-" + string(runes[1])
	})
	return strings.ToLower(result)
}

// ParseExtractedStyles parses extracted style and class attributes into separate ExtractedAttributeOps per style or
// class property.
func ParseExtractedStyles(job *pipeline.CompilationJob) {
	elements := make(map[ir_operation.XrefId]ir_operation.CreateOp)

	for _, unit := range job.GetUnits() {
		for op := unit.GetCreate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			createOp, ok := op.(ir_operation.CreateOp)
			if !ok {
				continue
			}
			if ops_create.IsElementOrContainerOp(createOp) {
				elements[createOp.GetXref()] = createOp
			}
		}
	}

	for _, unit := range job.GetUnits() {
		for op := unit.GetCreate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			if op.GetKind() == ir.OpKindExtractedAttribute {
				extractedAttrOp, ok := op.(*ops_create.ExtractedAttributeOp)
				if !ok {
					continue
				}
				if extractedAttrOp.BindingKind == ir.BindingKindAttribute &&
					expression.IsStringLiteral(extractedAttrOp.Expression) {
					target := elements[extractedAttrOp.Target]

					if target != nil {
						// Check if target is Template, ConditionalCreate, or ConditionalBranchCreate
						var templateKind *ir.TemplateKind
						switch targetOp := target.(type) {
						case *ops_create.TemplateOp:
							templateKind = &targetOp.TemplateKind
						case *ops_create.ConditionalCreateOp:
							templateKind = &targetOp.TemplateKind
						case *ops_create.ConditionalBranchCreateOp:
							templateKind = &targetOp.TemplateKind
						default:
							templateKind = nil
						}

						if templateKind != nil && *templateKind == ir.TemplateKindStructural {
							// TemplateDefinitionBuilder will not apply class and style bindings to structural
							// directives; instead, it will leave them as attributes.
							// (It's not clear what that would mean, anyway -- classes and styles on a structural
							// element should probably be a parse error.)
							// TODO: We may be able to remove this once Template Pipeline is the default.
							continue
						}
					}

					if extractedAttrOp.Name == "style" {
						// Get string value from LiteralExpr
						literalExpr, ok := extractedAttrOp.Expression.(*output.LiteralExpr)
						if !ok {
							continue
						}
						strValue, ok := literalExpr.Value.(string)
						if !ok {
							continue
						}
						parsedStyles := Parse(strValue)
						for i := 0; i < len(parsedStyles)-1; i += 2 {
							unit.GetCreate().InsertBefore(
								ops_create.NewExtractedAttributeOp(
									extractedAttrOp.Target,
									ir.BindingKindStyleProperty,
									nil,
									parsedStyles[i],
									output.NewLiteralExpr(parsedStyles[i+1], nil, nil),
									0,   // i18nContext
									nil, // i18nMessage
									core.SecurityContextSTYLE,
								),
								op,
							)
						}
						unit.GetCreate().Remove(op)
					} else if extractedAttrOp.Name == "class" {
						// Get string value from LiteralExpr
						literalExpr, ok := extractedAttrOp.Expression.(*output.LiteralExpr)
						if !ok {
							continue
						}
						strValue, ok := literalExpr.Value.(string)
						if !ok {
							continue
						}
						parsedClasses := strings.Fields(strings.TrimSpace(strValue))
						for _, parsedClass := range parsedClasses {
							if parsedClass == "" {
								continue
							}
							unit.GetCreate().InsertBefore(
								ops_create.NewExtractedAttributeOp(
									extractedAttrOp.Target,
									ir.BindingKindClassName,
									nil,
									parsedClass,
									nil, // expression
									0,   // i18nContext
									nil, // i18nMessage
									core.SecurityContextNONE,
								),
								op,
							)
						}
						unit.GetCreate().Remove(op)
					}
				}
			}
		}
	}
}
