package pipeline

import (
	"fmt"

	"ngc-go/packages/compiler/src/output"
	constant_pool "ngc-go/packages/compiler/src/pool"
	ir "ngc-go/packages/compiler/src/template/pipeline/ir/src"
	ops_shared "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/shared"

	pipeline_compilation "ngc-go/packages/compiler/src/template/pipeline/src/compilation"
	"ngc-go/packages/compiler/src/template/pipeline/src/phases"
)

// Phase represents a compilation phase
type Phase struct {
	Kind pipeline_compilation.CompilationJobKind
	Fn   interface{} // func(*pipeline_compilation.CompilationJob) | func(*pipeline_compilation.ComponentCompilationJob) | func(*pipeline_compilation.HostBindingCompilationJob)
}

var phasesList = []Phase{
	{pipeline_compilation.CompilationJobKindTmpl, phases.RemoveContentSelectors},
	{pipeline_compilation.CompilationJobKindBoth, phases.OptimizeRegularExpressions},
	{pipeline_compilation.CompilationJobKindHost, phases.ParseHostStyleProperties},
	{pipeline_compilation.CompilationJobKindTmpl, phases.EmitNamespaceChanges},
	{pipeline_compilation.CompilationJobKindTmpl, phases.PropagateI18nBlocks},
	{pipeline_compilation.CompilationJobKindTmpl, phases.WrapI18nIcus},
	{pipeline_compilation.CompilationJobKindBoth, phases.DeduplicateTextBindings},
	{pipeline_compilation.CompilationJobKindBoth, phases.SpecializeStyleBindings},
	{pipeline_compilation.CompilationJobKindBoth, phases.SpecializeBindings},
	{pipeline_compilation.CompilationJobKindBoth, phases.ConvertAnimations},
	{pipeline_compilation.CompilationJobKindBoth, phases.ExtractAttributes},
	{pipeline_compilation.CompilationJobKindTmpl, phases.CreateI18nContexts},
	{pipeline_compilation.CompilationJobKindBoth, phases.ParseExtractedStyles},
	{pipeline_compilation.CompilationJobKindTmpl, phases.RemoveEmptyBindings},
	{pipeline_compilation.CompilationJobKindBoth, phases.CollapseSingletonInterpolations},
	{pipeline_compilation.CompilationJobKindBoth, phases.OrderOps},
	{pipeline_compilation.CompilationJobKindTmpl, phases.GenerateConditionalExpressions},
	{pipeline_compilation.CompilationJobKindTmpl, phases.CreatePipes},
	{pipeline_compilation.CompilationJobKindTmpl, phases.ConfigureDeferInstructions},
	{pipeline_compilation.CompilationJobKindTmpl, phases.CreateVariadicPipes},
	{pipeline_compilation.CompilationJobKindBoth, phases.GeneratePureLiteralStructures},
	{pipeline_compilation.CompilationJobKindTmpl, phases.GenerateProjectionDefs},
	{pipeline_compilation.CompilationJobKindTmpl, phases.GenerateLocalLetReferences},
	{pipeline_compilation.CompilationJobKindTmpl, phases.GenerateVariables},
	{pipeline_compilation.CompilationJobKindTmpl, phases.SaveAndRestoreView},
	{pipeline_compilation.CompilationJobKindBoth, phases.DeleteAnyCasts},
	{pipeline_compilation.CompilationJobKindBoth, phases.ResolveDollarEvent},
	{pipeline_compilation.CompilationJobKindTmpl, phases.GenerateTrackVariables},
	{pipeline_compilation.CompilationJobKindTmpl, phases.RemoveIllegalLetReferences},
	{pipeline_compilation.CompilationJobKindBoth, phases.ResolveNames},
	{pipeline_compilation.CompilationJobKindTmpl, phases.ResolveDeferTargetNames},
	{pipeline_compilation.CompilationJobKindTmpl, phases.TransformTwoWayBindingSet},
	{pipeline_compilation.CompilationJobKindTmpl, phases.OptimizeTrackFns},
	{pipeline_compilation.CompilationJobKindBoth, phases.ResolveContexts},
	{pipeline_compilation.CompilationJobKindBoth, phases.ResolveSanitizers},
	{pipeline_compilation.CompilationJobKindTmpl, phases.LiftLocalRefs},
	{pipeline_compilation.CompilationJobKindBoth, phases.ExpandSafeReads},
	{pipeline_compilation.CompilationJobKindBoth, phases.StripNonrequiredParentheses},
	{pipeline_compilation.CompilationJobKindBoth, phases.GenerateTemporaryVariables},
	{pipeline_compilation.CompilationJobKindBoth, phases.OptimizeVariables},
	{pipeline_compilation.CompilationJobKindBoth, phases.OptimizeStoreLet},
	{pipeline_compilation.CompilationJobKindTmpl, phases.ConvertI18nText},
	{pipeline_compilation.CompilationJobKindTmpl, phases.ConvertI18nBindings},
	{pipeline_compilation.CompilationJobKindTmpl, phases.RemoveUnusedI18nAttributesOps},
	{pipeline_compilation.CompilationJobKindTmpl, phases.AssignI18nSlotDependencies},
	{pipeline_compilation.CompilationJobKindTmpl, phases.ApplyI18nExpressions},
	{pipeline_compilation.CompilationJobKindTmpl, phases.AllocateSlots},
	{pipeline_compilation.CompilationJobKindTmpl, phases.ResolveI18nElementPlaceholders},
	{pipeline_compilation.CompilationJobKindTmpl, phases.ResolveI18nExpressionPlaceholders},
	{pipeline_compilation.CompilationJobKindTmpl, phases.ExtractI18nMessages},
	{pipeline_compilation.CompilationJobKindTmpl, phases.CollectI18nConsts},
	{pipeline_compilation.CompilationJobKindTmpl, phases.CollectConstExpressions},
	{pipeline_compilation.CompilationJobKindBoth, phases.CollectElementConsts},
	{pipeline_compilation.CompilationJobKindTmpl, phases.RemoveI18nContexts},
	{pipeline_compilation.CompilationJobKindBoth, phases.CountVariables},
	{pipeline_compilation.CompilationJobKindTmpl, phases.GenerateAdvance},
	{pipeline_compilation.CompilationJobKindBoth, phases.NameFunctionsAndVariables},
	{pipeline_compilation.CompilationJobKindTmpl, phases.ResolveDeferDepsFns},
	{pipeline_compilation.CompilationJobKindTmpl, phases.MergeNextContextExpressions},
	{pipeline_compilation.CompilationJobKindTmpl, phases.GenerateNgContainerOps},
	{pipeline_compilation.CompilationJobKindTmpl, phases.CollapseEmptyInstructions},
	{pipeline_compilation.CompilationJobKindTmpl, phases.AttachSourceLocations},
	{pipeline_compilation.CompilationJobKindTmpl, phases.DisableBindings},
	{pipeline_compilation.CompilationJobKindBoth, phases.ExtractPureFunctions},
	{pipeline_compilation.CompilationJobKindBoth, phases.Reify},
	{pipeline_compilation.CompilationJobKindBoth, phases.Chain},
}

// Transform runs all transformation phases in the correct order against a compilation job.
// After this processing, the compilation should be in a state where it can be emitted.
func Transform(job interface{}, kind pipeline_compilation.CompilationJobKind) {
	// Get the base CompilationJob to check Kind
	var baseJob *pipeline_compilation.CompilationJob
	switch j := job.(type) {
	case *pipeline_compilation.ComponentCompilationJob:
		baseJob = j.CompilationJob
	case *pipeline_compilation.HostBindingCompilationJob:
		baseJob = j.CompilationJob
	case *pipeline_compilation.CompilationJob:
		baseJob = j
	default:
		panic(fmt.Sprintf("Transform: unexpected job type %T", job))
	}

	for _, phase := range phasesList {
		if phase.Kind == kind || phase.Kind == pipeline_compilation.CompilationJobKindBoth {
			// Type assertion to call the appropriate function
			switch fn := phase.Fn.(type) {
			case func(*pipeline_compilation.CompilationJob):
				fn(baseJob)
			case func(*pipeline_compilation.ComponentCompilationJob):
				if componentJob, ok := job.(*pipeline_compilation.ComponentCompilationJob); ok {
					fn(componentJob)
				}
			case func(*pipeline_compilation.HostBindingCompilationJob):
				if hostJob, ok := job.(*pipeline_compilation.HostBindingCompilationJob); ok {
					fn(hostJob)
				}
			}
		}
	}
}

// EmitTemplateFn compiles all views in the given ComponentCompilationJob into the final template function,
// which may reference constants defined in a ConstantPool.
func EmitTemplateFn(tpl *pipeline_compilation.ComponentCompilationJob, pool *constant_pool.ConstantPool) *output.FunctionExpr {
	rootFn := emitView(tpl.Root)
	emitChildViews(tpl.Root, pool)
	return rootFn
}

func emitChildViews(parent *pipeline_compilation.ViewCompilationUnit, pool *constant_pool.ConstantPool) {
	for _, unit := range parent.Job.GetUnits() {
		viewUnit, ok := unit.(*pipeline_compilation.ViewCompilationUnit)
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
func emitView(view *pipeline_compilation.ViewCompilationUnit) *output.FunctionExpr {
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
		stmtOp, ok := op.(*ops_shared.StatementOp)
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
		stmtOp, ok := op.(*ops_shared.StatementOp)
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
func EmitHostBindingFunction(job *pipeline_compilation.HostBindingCompilationJob) *output.FunctionExpr {
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
		stmtOp, ok := op.(*ops_shared.StatementOp)
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
		stmtOp, ok := op.(*ops_shared.StatementOp)
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
