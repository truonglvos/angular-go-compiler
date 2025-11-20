package phases

import (
	"ngc-go/packages/compiler/src/output"
	"ngc-go/packages/compiler/src/template/pipeline/ir"
	ops_create "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/create"

	pipeline "ngc-go/packages/compiler/src/template/pipeline/src/compilation"
)

// LiftLocalRefs lifts local reference declarations on element-like structures within each view
// into an entry in the `consts` array for the whole component.
func LiftLocalRefs(job *pipeline.ComponentCompilationJob) {
	for _, unit := range job.GetUnits() {
		for op := unit.GetCreate().Head(); op != nil; op = op.Next() {
			switch op.GetKind() {
			case ir.OpKindElementStart:
				if elementStart, ok := op.(*ops_create.ElementStartOp); ok {
					handleLocalRefs(elementStart.LocalRefs, elementStart, job, unit)
				}
			case ir.OpKindConditionalCreate:
				if conditionalCreate, ok := op.(*ops_create.ConditionalCreateOp); ok {
					handleLocalRefs(conditionalCreate.LocalRefs, conditionalCreate, job, unit)
				}
			case ir.OpKindConditionalBranchCreate:
				if branchCreate, ok := op.(*ops_create.ConditionalBranchCreateOp); ok {
					handleLocalRefs(branchCreate.LocalRefs, branchCreate, job, unit)
				}
			case ir.OpKindTemplate:
				if templateOp, ok := op.(*ops_create.TemplateOp); ok {
					handleLocalRefs(templateOp.LocalRefs, templateOp, job, unit)
				}
			}
		}
	}
}

// handleLocalRefs processes local refs for an operation
func handleLocalRefs(
	localRefs interface{},
	op interface{},
	job *pipeline.ComponentCompilationJob,
	unit pipeline.CompilationUnit,
) {
	refs, ok := localRefs.([]ops_create.LocalRef)
	if !ok {
		panic("AssertionError: expected localRefs to be an array still")
	}

	// Get the operation that has NumSlotsUsed field
	var numSlotsUsed *int
	switch o := op.(type) {
	case *ops_create.ElementStartOp:
		numSlotsUsed = &o.NumSlotsUsed
	case *ops_create.ConditionalCreateOp:
		numSlotsUsed = &o.NumSlotsUsed
	case *ops_create.ConditionalBranchCreateOp:
		numSlotsUsed = &o.NumSlotsUsed
	case *ops_create.TemplateOp:
		numSlotsUsed = &o.NumSlotsUsed
	}

	if numSlotsUsed != nil {
		*numSlotsUsed += len(refs)
	}

	if len(refs) > 0 {
		localRefsExpr := serializeLocalRefs(refs)
		constIndex := job.AddConst(localRefsExpr, nil)

		// Update the LocalRefs field
		switch o := op.(type) {
		case *ops_create.ElementStartOp:
			o.LocalRefs = constIndex
		case *ops_create.ConditionalCreateOp:
			o.LocalRefs = constIndex
		case *ops_create.ConditionalBranchCreateOp:
			o.LocalRefs = constIndex
		case *ops_create.TemplateOp:
			o.LocalRefs = constIndex
		}
	} else {
		// Set to nil
		switch o := op.(type) {
		case *ops_create.ElementStartOp:
			o.LocalRefs = nil
		case *ops_create.ConditionalCreateOp:
			o.LocalRefs = nil
		case *ops_create.ConditionalBranchCreateOp:
			o.LocalRefs = nil
		case *ops_create.TemplateOp:
			o.LocalRefs = nil
		}
	}
}

// serializeLocalRefs serializes local refs to a literal array expression
func serializeLocalRefs(refs []ops_create.LocalRef) output.OutputExpression {
	constRefs := make([]output.OutputExpression, 0, len(refs)*2)
	for _, ref := range refs {
		constRefs = append(constRefs,
			output.NewLiteralExpr(ref.Name, nil, nil),
			output.NewLiteralExpr(ref.Target, nil, nil),
		)
	}
	return output.NewLiteralArrayExpr(constRefs, nil, nil)
}
