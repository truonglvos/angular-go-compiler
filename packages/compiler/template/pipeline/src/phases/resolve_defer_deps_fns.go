package phases

import (
	ir "ngc-go/packages/compiler/template/pipeline/ir/src"
	ops_create "ngc-go/packages/compiler/template/pipeline/ir/src/ops/create"
	pipeline "ngc-go/packages/compiler/template/pipeline/src/compilation"
)

// ResolveDeferDepsFns resolves the dependency function of a deferred block.
func ResolveDeferDepsFns(job *pipeline.ComponentCompilationJob) {
	for _, unit := range job.GetUnits() {
		viewUnit, ok := unit.(*pipeline.ViewCompilationUnit)
		if !ok {
			continue
		}
		for op := viewUnit.Create.Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			if op.GetKind() != ir.OpKindDefer {
				continue
			}

			deferOp, ok := op.(*ops_create.DeferOp)
			if !ok {
				continue
			}

			if deferOp.ResolverFn != nil {
				continue
			}

			if deferOp.OwnResolverFn != nil {
				if deferOp.Handle == nil || deferOp.Handle.Slot == nil {
					panic("AssertionError: slot must be assigned before extracting defer deps functions")
				}
				fullPathName := ""
				if viewUnit.FnName != nil {
					fullPathName = *viewUnit.FnName
					// Replace "_Template" suffix
					if len(fullPathName) > 9 && fullPathName[len(fullPathName)-9:] == "_Template" {
						fullPathName = fullPathName[:len(fullPathName)-9]
					}
				}
				slotNum := 0
				if deferOp.Handle.Slot != nil {
					slotNum = *deferOp.Handle.Slot
				}
				resolverFnName := fullPathName + "_Defer_" + string(rune(slotNum)) + "_DepsFn"
				// TODO: Implement GetSharedFunctionReference using job.Pool.GetSharedFunctionReference
				// For now, just use the ownResolverFn directly
				deferOp.ResolverFn = deferOp.OwnResolverFn
				_ = resolverFnName // TODO: Use this when GetSharedFunctionReference is implemented
			}
		}
	}
}
