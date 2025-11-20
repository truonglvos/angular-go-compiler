package phases

import (
	"ngc-go/packages/compiler/src/template/pipeline/ir"
	ir_operation "ngc-go/packages/compiler/src/template/pipeline/ir/src/operations"
	ops_update "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/update"

	pipeline "ngc-go/packages/compiler/src/template/pipeline/src/compilation"
)

// DeduplicateTextBindings deduplicates text bindings, e.g. <div class="cls1" class="cls2">
func DeduplicateTextBindings(job *pipeline.CompilationJob) {
	seen := make(map[ir_operation.XrefId]map[string]bool)
	for _, unit := range job.GetUnits() {
		// Iterate in reverse order
		for op := unit.GetUpdate().Tail(); op != nil; op = op.GetPrev() {
			if bindingOp, ok := op.(*ops_update.BindingOp); ok && bindingOp.IsTextAttribute {
				seenForElement, exists := seen[bindingOp.Target]
				if !exists {
					seenForElement = make(map[string]bool)
					seen[bindingOp.Target] = seenForElement
				}

				if seenForElement[bindingOp.Name] {
					if job.Compatibility == ir.CompatibilityModeTemplateDefinitionBuilder {
						// For most duplicated attributes, TemplateDefinitionBuilder lists all of the values in
						// the consts array. However, for style and class attributes it only keeps the last one.
						// We replicate that behavior here since it has actual consequences for apps with
						// duplicate class or style attrs.
						if bindingOp.Name == "style" || bindingOp.Name == "class" {
							unit.GetUpdate().Remove(op)
						}
					} else {
						// TODO: Determine the correct behavior. It would probably make sense to merge multiple
						// style and class attributes. Alternatively we could just throw an error, as HTML
						// doesn't permit duplicate attributes.
					}
				}
				seenForElement[bindingOp.Name] = true
			}
		}
	}
}
