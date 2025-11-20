package pipeline

import (
	"fmt"

	"ngc-go/packages/compiler/src/constant"
	"ngc-go/packages/compiler/src/output"
	"ngc-go/packages/compiler/src/template/pipeline/ir"
	"ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/shared"

	"ngc-go/packages/compiler/src/template/pipeline/src/compilation"
	"ngc-go/packages/compiler/src/template/pipeline/src/phases"
)

// Phase represents a compilation phase
type Phase struct {
	Kind compilation.CompilationJobKind
	Fn   interface{} // func(*compilation.CompilationJob) | func(*compilation.ComponentCompilationJob) | func(*compilation.HostBindingCompilationJob)
}

var phasesList = []Phase{
	{compilation.CompilationJobKindTmpl, phases.RemoveContentSelectors},
	{compilation.CompilationJobKindBoth, phases.OptimizeRegularExpressions},
	{compilation.CompilationJobKindHost, phases.ParseHostStyleProperties},
	{compilation.CompilationJobKindTmpl, phases.EmitNamespaceChanges},
	{compilation.CompilationJobKindTmpl, phases.PropagateI18nBlocks},
	{compilation.CompilationJobKindTmpl, phases.WrapI18nIcus},
	{compilation.CompilationJobKindBoth, phases.DeduplicateTextBindings},
	{compilation.CompilationJobKindBoth, phases.SpecializeStyleBindings},
	{compilation.CompilationJobKindBoth, phases.SpecializeBindings},
	{compilation.CompilationJobKindBoth, phases.ConvertAnimations},
	{compilation.CompilationJobKindBoth, phases.ExtractAttributes},
	{compilation.CompilationJobKindTmpl, phases.CreateI18nContexts},
	{compilation.CompilationJobKindBoth, phases.ParseExtractedStyles},
	{compilation.CompilationJobKindTmpl, phases.RemoveEmptyBindings},
	{compilation.CompilationJobKindBoth, phases.CollapseSingletonInterpolations},
	{compilation.CompilationJobKindBoth, phases.OrderOps},
	{compilation.CompilationJobKindTmpl, phases.GenerateConditionalExpressions},
	{compilation.CompilationJobKindTmpl, phases.CreatePipes},
	{compilation.CompilationJobKindTmpl, phases.ConfigureDeferInstructions},
	{compilation.CompilationJobKindTmpl, phases.CreateVariadicPipes},
	{compilation.CompilationJobKindBoth, phases.GeneratePureLiteralStructures},
	{compilation.CompilationJobKindTmpl, phases.GenerateProjectionDefs},
	{compilation.CompilationJobKindTmpl, phases.GenerateLocalLetReferences},
	{compilation.CompilationJobKindTmpl, phases.GenerateVariables},
	{compilation.CompilationJobKindTmpl, phases.SaveAndRestoreView},
	{compilation.CompilationJobKindBoth, phases.DeleteAnyCasts},
	{compilation.CompilationJobKindBoth, phases.ResolveDollarEvent},
	{compilation.CompilationJobKindTmpl, phases.GenerateTrackVariables},
	{compilation.CompilationJobKindTmpl, phases.RemoveIllegalLetReferences},
	{compilation.CompilationJobKindBoth, phases.ResolveNames},
	{compilation.CompilationJobKindTmpl, phases.ResolveDeferTargetNames},
	{compilation.CompilationJobKindTmpl, phases.TransformTwoWayBindingSet},
	{compilation.CompilationJobKindTmpl, phases.OptimizeTrackFns},
	{compilation.CompilationJobKindBoth, phases.ResolveContexts},
	{compilation.CompilationJobKindBoth, phases.ResolveSanitizers},
	{compilation.CompilationJobKindTmpl, phases.LiftLocalRefs},
	{compilation.CompilationJobKindBoth, phases.ExpandSafeReads},
	{compilation.CompilationJobKindBoth, phases.StripNonrequiredParentheses},
	{compilation.CompilationJobKindBoth, phases.GenerateTemporaryVariables},
	{compilation.CompilationJobKindBoth, phases.OptimizeVariables},
	{compilation.CompilationJobKindBoth, phases.OptimizeStoreLet},
	{compilation.CompilationJobKindTmpl, phases.ConvertI18nText},
	{compilation.CompilationJobKindTmpl, phases.ConvertI18nBindings},
	{compilation.CompilationJobKindTmpl, phases.RemoveUnusedI18nAttributesOps},
	{compilation.CompilationJobKindTmpl, phases.AssignI18nSlotDependencies},
	{compilation.CompilationJobKindTmpl, phases.ApplyI18nExpressions},
	{compilation.CompilationJobKindTmpl, phases.AllocateSlots},
	{compilation.CompilationJobKindTmpl, phases.ResolveI18nElementPlaceholders},
	{compilation.CompilationJobKindTmpl, phases.ResolveI18nExpressionPlaceholders},
	{compilation.CompilationJobKindTmpl, phases.ExtractI18nMessages},
	{compilation.CompilationJobKindTmpl, phases.CollectI18nConsts},
	{compilation.CompilationJobKindTmpl, phases.CollectConstExpressions},
	{compilation.CompilationJobKindBoth, phases.CollectElementConsts},
	{compilation.CompilationJobKindTmpl, phases.RemoveI18nContexts},
	{compilation.CompilationJobKindBoth, phases.CountVariables},
	{compilation.CompilationJobKindTmpl, phases.GenerateAdvance},
	{compilation.CompilationJobKindBoth, phases.NameFunctionsAndVariables},
	{compilation.CompilationJobKindTmpl, phases.ResolveDeferDepsFns},
	{compilation.CompilationJobKindTmpl, phases.MergeNextContextExpressions},
	{compilation.CompilationJobKindTmpl, phases.GenerateNgContainerOps},
	{compilation.CompilationJobKindTmpl, phases.CollapseEmptyInstructions},
	{compilation.CompilationJobKindTmpl, phases.AttachSourceLocations},
	{compilation.CompilationJobKindTmpl, phases.DisableBindings},
	{compilation.CompilationJobKindBoth, phases.ExtractPureFunctions},
	{compilation.CompilationJobKindBoth, phases.Reify},
	{compilation.CompilationJobKindBoth, phases.Chain},
}

// Transform runs all transformation phases in the correct order against a compilation job.
// After this processing, the compilation should be in a state where it can be emitted.
func Transform(job interface{}, kind compilation.CompilationJobKind) {
	// Get the base CompilationJob to check Kind
	var baseJob *compilation.CompilationJob
	switch j := job.(type) {
	case *compilation.ComponentCompilationJob:
		baseJob = j.CompilationJob
	case *compilation.HostBindingCompilationJob:
		baseJob = j.CompilationJob
	case *compilation.CompilationJob:
		baseJob = j
	default:
		panic(fmt.Sprintf("Transform: unexpected job type %T", job))
	}

	for _, phase := range phasesList {
		if phase.Kind == kind || phase.Kind == compilation.CompilationJobKindBoth {
			// Type assertion to call the appropriate function
			switch fn := phase.Fn.(type) {
			case func(*compilation.CompilationJob):
				fn(baseJob)
			case func(*compilation.ComponentCompilationJob):
				if componentJob, ok := job.(*compilation.ComponentCompilationJob); ok {
					fn(componentJob)
				}
			case func(*compilation.HostBindingCompilationJob):
				if hostJob, ok := job.(*compilation.HostBindingCompilationJob); ok {
					fn(hostJob)
				}
			}
		}
	}
}

// EmitTemplateFn compiles all views in the given ComponentCompilationJob into the final template function,
// which may reference constants defined in a ConstantPool.
func EmitTemplateFn(tpl *compilation.ComponentCompilationJob, pool *constant.ConstantPool) *output.FunctionExpr {
	rootFn := emitView(tpl.Root)
	emitChildViews(tpl.Root, pool)
	return rootFn
}

func emitChildViews(parent *compilation.ViewCompilationUnit, pool *constant.ConstantPool) {
	for _, unit := range parent.Job.GetUnits() {
		viewUnit, ok := unit.(*compilation.ViewCompilationUnit)
		if !ok {
			continue
		}
		if viewUnit.Parent == nil || *viewUnit.Parent != parent.Xref {
			continue
		}

		// Child views are emitted depth-first.
		emitChildViews(viewUnit, pool)

		viewFn := emitView(viewUnit)
		if viewFn.Name == nil {
			panic(fmt.Sprintf("AssertionError: view function %d is unnamed", viewUnit.Xref))
		}
		pool.AddStatement(viewFn.ToDeclStmt(*viewFn.Name, output.StmtModifierNone))
	}
}

// emitView emits a template function for an individual ViewCompilationUnit
// (which may be either the root view or an embedded view).
func emitView(view *compilation.ViewCompilationUnit) *output.FunctionExpr {
	if view.FnName == nil {
		panic(fmt.Sprintf("AssertionError: view %d is unnamed", view.Xref))
	}

	createStatements := []output.OutputStatement{}
	for op := view.Create.Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
		if op.GetKind() != ir.OpKindStatement {
			panic(fmt.Sprintf(
				"AssertionError: expected all create ops to have been compiled, but got %v",
				op.GetKind(),
			))
		}
		stmtOp, ok := op.(*shared.StatementOp)
		if !ok {
			panic(fmt.Sprintf("AssertionError: expected StatementOp, but got %T", op))
		}
		createStatements = append(createStatements, stmtOp.Statement)
	}

	updateStatements := []output.OutputStatement{}
	for op := view.Update.Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
		if op.GetKind() != ir.OpKindStatement {
			panic(fmt.Sprintf(
				"AssertionError: expected all update ops to have been compiled, but got %v",
				op.GetKind(),
			))
		}
		stmtOp, ok := op.(*shared.StatementOp)
		if !ok {
			panic(fmt.Sprintf("AssertionError: expected StatementOp, but got %T", op))
		}
		updateStatements = append(updateStatements, stmtOp.Statement)
	}

	createCond := maybeGenerateRfBlock(1, createStatements)
	updateCond := maybeGenerateRfBlock(2, updateStatements)
	allStatements := append(createCond, updateCond...)

	return output.NewFunctionExpr(
		[]*output.FnParam{
			output.NewFnParam("rf", nil),
			output.NewFnParam("ctx", nil),
		},
		allStatements,
		nil, // type
		nil, // sourceSpan
		view.FnName,
	)
}

func maybeGenerateRfBlock(flag int, statements []output.OutputStatement) []output.OutputStatement {
	if len(statements) == 0 {
		return []output.OutputStatement{}
	}

	// Create: (rf & flag) condition
	rfVar := output.NewReadVarExpr("rf", nil, nil)
	flagLiteral := output.NewLiteralExpr(flag, nil, nil)
	condition := output.NewBinaryOperatorExpr(
		output.BinaryOperatorBitwiseAnd,
		rfVar,
		flagLiteral,
		nil,
		nil,
	)

	return []output.OutputStatement{
		output.NewIfStmt(condition, statements, nil, nil, nil),
	}
}

// EmitHostBindingFunction emits a host binding function
func EmitHostBindingFunction(job *compilation.HostBindingCompilationJob) *output.FunctionExpr {
	if job.Root.GetFnName() == nil {
		panic("AssertionError: host binding function is unnamed")
	}

	createStatements := []output.OutputStatement{}
	for op := job.Root.GetCreate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
		if op.GetKind() != ir.OpKindStatement {
			panic(fmt.Sprintf(
				"AssertionError: expected all create ops to have been compiled, but got %v",
				op.GetKind(),
			))
		}
		stmtOp, ok := op.(*shared.StatementOp)
		if !ok {
			panic(fmt.Sprintf("AssertionError: expected StatementOp, but got %T", op))
		}
		createStatements = append(createStatements, stmtOp.Statement)
	}

	updateStatements := []output.OutputStatement{}
	for op := job.Root.GetUpdate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
		if op.GetKind() != ir.OpKindStatement {
			panic(fmt.Sprintf(
				"AssertionError: expected all update ops to have been compiled, but got %v",
				op.GetKind(),
			))
		}
		stmtOp, ok := op.(*shared.StatementOp)
		if !ok {
			panic(fmt.Sprintf("AssertionError: expected StatementOp, but got %T", op))
		}
		updateStatements = append(updateStatements, stmtOp.Statement)
	}

	if len(createStatements) == 0 && len(updateStatements) == 0 {
		return nil
	}

	createCond := maybeGenerateRfBlock(1, createStatements)
	updateCond := maybeGenerateRfBlock(2, updateStatements)
	allStatements := append(createCond, updateCond...)

	fnName := job.Root.GetFnName()
	return output.NewFunctionExpr(
		[]*output.FnParam{
			output.NewFnParam("rf", nil),
			output.NewFnParam("ctx", nil),
		},
		allStatements,
		nil, // type
		nil, // sourceSpan
		fnName,
	)
}
