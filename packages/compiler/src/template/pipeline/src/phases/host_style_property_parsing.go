package phases

import (
	"strings"
	"unicode"

	"ngc-go/packages/compiler/src/template/pipeline/ir"
	ops_update "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/update"

	"ngc-go/packages/compiler/src/template/pipeline/src/compilation"
)

const (
	styleDot      = "style."
	classDot      = "class."
	styleBang     = "style!"
	classBang     = "class!"
	bangImportant = "!important"
)

// ParseHostStyleProperties parses host style properties.
// Host bindings are compiled using a different parser entrypoint, and are parsed quite differently
// as a result. Therefore, we need to do some extra parsing for host style properties, as compared
// to non-host style properties.
// TODO: Unify host bindings and non-host bindings in the parser.
func ParseHostStyleProperties(job *compilation.CompilationJob) {
	// Check if this is a host binding compilation job
	if job.Kind != compilation.CompilationJobKindHost {
		return
	}

	// Get the root unit and cast to HostBindingCompilationUnit
	rootUnit := job.GetRoot()
	hostUnit, ok := rootUnit.(*compilation.HostBindingCompilationUnit)
	if !ok {
		return
	}

	for op := hostUnit.GetUpdate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
		if op.GetKind() != ir.OpKindBinding {
			continue
		}

		bindingOp, ok := op.(*ops_update.BindingOp)
		if !ok || bindingOp.BindingKind != ir.BindingKindProperty {
			continue
		}

		if strings.HasSuffix(bindingOp.Name, bangImportant) {
			// Delete any `!important` suffixes from the binding name.
			bindingOp.Name = bindingOp.Name[:len(bindingOp.Name)-len(bangImportant)]
		}

		if strings.HasPrefix(bindingOp.Name, styleDot) {
			bindingOp.BindingKind = ir.BindingKindStyleProperty
			bindingOp.Name = bindingOp.Name[len(styleDot):]

			if !isCssCustomProperty(bindingOp.Name) {
				bindingOp.Name = hyphenate(bindingOp.Name)
			}

			property, suffix := parseProperty(bindingOp.Name)
			bindingOp.Name = property
			if suffix != "" {
				bindingOp.Unit = &suffix
			}
		} else if strings.HasPrefix(bindingOp.Name, styleBang) {
			bindingOp.BindingKind = ir.BindingKindStyleProperty
			bindingOp.Name = "style"
		} else if strings.HasPrefix(bindingOp.Name, classDot) {
			bindingOp.BindingKind = ir.BindingKindClassName
			property, _ := parseProperty(bindingOp.Name[len(classDot):])
			bindingOp.Name = property
		} else if strings.HasPrefix(bindingOp.Name, classBang) {
			bindingOp.BindingKind = ir.BindingKindClassName
			property, _ := parseProperty(bindingOp.Name[len(classBang):])
			bindingOp.Name = property
		}
	}
}

// isCssCustomProperty checks whether property name is a custom CSS property.
// See: https://www.w3.org/TR/css-variables-1
func isCssCustomProperty(name string) bool {
	return strings.HasPrefix(name, "--")
}

// hyphenate converts camelCase to kebab-case
func hyphenate(value string) string {
	var result strings.Builder
	for i, r := range value {
		if i > 0 && unicode.IsLower(rune(value[i-1])) && unicode.IsUpper(r) {
			result.WriteRune('-')
		}
		result.WriteRune(unicode.ToLower(r))
	}
	return result.String()
}

// parseProperty parses a property name and extracts the property name and unit suffix
func parseProperty(name string) (property string, suffix string) {
	overrideIndex := strings.Index(name, "!important")
	if overrideIndex != -1 {
		if overrideIndex > 0 {
			name = name[:overrideIndex]
		} else {
			name = ""
		}
	}

	suffix = ""
	property = name
	unitIndex := strings.LastIndex(name, ".")
	if unitIndex > 0 {
		suffix = name[unitIndex+1:]
		property = name[:unitIndex]
	}

	return property, suffix
}
