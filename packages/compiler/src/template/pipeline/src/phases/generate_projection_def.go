package phases

import (
	"ngc-go/packages/compiler/src/core"
	"ngc-go/packages/compiler/src/output"
	"ngc-go/packages/compiler/src/template/pipeline/ir"
	ops_create "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/create"

	"ngc-go/packages/compiler/src/template/pipeline/src/compilation"
	pipeline_convension "ngc-go/packages/compiler/src/template/pipeline/src/convension"
)

// GenerateProjectionDefs locates projection slots, populates the each component's `ngContentSelectors`
// literal field, populates `project` arguments, and generates the required `projectionDef` instruction
// for the job's root view.
func GenerateProjectionDefs(job *compilation.ComponentCompilationJob) {
	// TODO: Why does TemplateDefinitionBuilder force a shared constant?
	share := job.Compatibility == ir.CompatibilityModeTemplateDefinitionBuilder
	sharePtr := &share

	// Collect all selectors from this component, and its nested views. Also, assign each projection a
	// unique ascending projection slot index.
	selectors := make([]string, 0)
	projectionSlotIndex := 0
	for _, unit := range job.GetUnits() {
		for op := unit.GetCreate().Head(); op != nil; op = op.Next() {
			if projectionOp, ok := op.(*ops_create.ProjectionOp); ok {
				selectors = append(selectors, projectionOp.Selector)
				projectionOp.ProjectionSlotIndex = projectionSlotIndex
				projectionSlotIndex++
			}
		}
	}

	if len(selectors) > 0 {
		// Create the projectionDef array. If we only found a single wildcard selector, then we use the
		// default behavior with no arguments instead.
		var defExpr output.OutputExpression
		if len(selectors) > 1 || selectors[0] != "*" {
			def := make([]interface{}, len(selectors))
			for i, s := range selectors {
				if s == "*" {
					def[i] = s
				} else {
					selectorPtr := &s
					r3Selector := core.ParseSelectorToR3Selector(selectorPtr)
					// Convert R3CssSelectorList to interface{} for LiteralOrArrayLiteral
					def[i] = r3Selector
				}
			}
			defExpr = job.Pool.GetConstLiteral(pipeline_convension.LiteralOrArrayLiteral(def), sharePtr)
		} else {
			defExpr = nil
		}

		// Create the ngContentSelectors constant.
		selectorsInterface := make([]interface{}, len(selectors))
		for i, s := range selectors {
			selectorsInterface[i] = s
		}
		job.ContentSelectors = job.Pool.GetConstLiteral(pipeline_convension.LiteralOrArrayLiteral(selectorsInterface), sharePtr)

		// The projection def instruction goes at the beginning of the root view, before any
		// `projection` instructions.
		if defExpr != nil {
			projectionDefOp := ops_create.NewProjectionDefOp(defExpr)
			// Insert at the beginning of the create list
			rootCreate := job.Root.GetCreate()
			if rootCreate.Head() != nil {
				rootCreate.InsertBefore(rootCreate.Head(), projectionDefOp)
			} else {
				rootCreate.Push(projectionDefOp)
			}
		}
	}
}
