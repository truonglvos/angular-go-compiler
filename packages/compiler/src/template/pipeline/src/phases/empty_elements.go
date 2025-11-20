package phases

import (
	"ngc-go/packages/compiler/src/template/pipeline/ir"
	"ngc-go/packages/compiler/src/template/pipeline/ir/src/operations"
	ops_create "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/create"

	pipeline "ngc-go/packages/compiler/src/template/pipeline/src/compilation"
)

// Replacements maps end op kinds to their corresponding start and merged kinds
var replacements = map[ir.OpKind][2]ir.OpKind{
	ir.OpKindElementEnd:   {ir.OpKindElementStart, ir.OpKindElement},
	ir.OpKindContainerEnd: {ir.OpKindContainerStart, ir.OpKindContainer},
	ir.OpKindI18nEnd:      {ir.OpKindI18nStart, ir.OpKindI18n},
}

// ignoredOpKinds contains op kinds that should not prevent merging of start/end ops
var ignoredOpKinds = map[ir.OpKind]bool{
	ir.OpKindPipe: true,
}

// CollapseEmptyInstructions replaces sequences of mergable instructions (e.g. `ElementStart` and `ElementEnd`)
// with a consolidated instruction (e.g. `Element`).
func CollapseEmptyInstructions(job *pipeline.CompilationJob) {
	for _, unit := range job.GetUnits() {
		for op := unit.GetCreate().Head(); op != nil; op = op.Next() {
			// Find end ops that may be able to be merged.
			opReplacement, ok := replacements[op.GetKind()]
			if !ok {
				continue
			}
			startKind := opReplacement[0]

			// Locate the previous (non-ignored) op.
			prevOp := op.GetPrev()
			for prevOp != nil && ignoredOpKinds[prevOp.GetKind()] {
				prevOp = prevOp.GetPrev()
			}

			// If the previous op is the corresponding start op, we can merge.
			if prevOp != nil && prevOp.GetKind() == startKind {
				// Replace the start instruction with the merged version
				// This requires creating a new merged op based on the start op
				var mergedOp operations.CreateOp
				switch startKind {
				case ir.OpKindElementStart:
					if elementStart, ok := prevOp.(*ops_create.ElementStartOp); ok {
						tag := ""
						if elementStart.Tag != nil {
							tag = *elementStart.Tag
						}
						mergedOp = ops_create.NewElementOp(
							tag,
							elementStart.Xref,
							elementStart.Namespace,
							elementStart.I18nPlaceholder,
							elementStart.StartSourceSpan,
							elementStart.WholeSourceSpan,
						)
						mergedOpBase := mergedOp.(*ops_create.ElementOp)
						mergedOpBase.Handle = elementStart.Handle
						mergedOpBase.NumSlotsUsed = elementStart.NumSlotsUsed
						mergedOpBase.Attributes = elementStart.Attributes
						mergedOpBase.LocalRefs = elementStart.LocalRefs
						mergedOpBase.NonBindable = elementStart.NonBindable
						mergedOpBase.I18nPlaceholder = elementStart.I18nPlaceholder
					}
				case ir.OpKindContainerStart:
					if containerStart, ok := prevOp.(*ops_create.ContainerStartOp); ok {
						mergedOp = ops_create.NewContainerOp(
							containerStart.Xref,
							containerStart.StartSourceSpan,
							containerStart.WholeSourceSpan,
						)
						mergedOpBase := mergedOp.(*ops_create.ContainerOp)
						mergedOpBase.Handle = containerStart.Handle
						mergedOpBase.NumSlotsUsed = containerStart.NumSlotsUsed
						mergedOpBase.Attributes = containerStart.Attributes
						mergedOpBase.LocalRefs = containerStart.LocalRefs
						mergedOpBase.NonBindable = containerStart.NonBindable
					}
				case ir.OpKindI18nStart:
					// For I18n, we need to create I18nOp
					// TODO: Implement I18nOp creation if needed
					// For now, skip this case
					continue
				}

				if mergedOp != nil {
					// Replace the start instruction with the merged version
					unit.GetCreate().Replace(prevOp, mergedOp)
					// Remove the end instruction
					unit.GetCreate().Remove(op)
				}
			}
		}
	}
}
