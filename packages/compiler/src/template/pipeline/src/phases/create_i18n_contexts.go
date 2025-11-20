package phases

import (
	"ngc-go/packages/compiler/src/i18n"
	ir "ngc-go/packages/compiler/src/template/pipeline/ir/src"
	ir_operation "ngc-go/packages/compiler/src/template/pipeline/ir/src/operations"
	ops_create "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/create"
	ops_update "ngc-go/packages/compiler/src/template/pipeline/ir/src/ops/update"

	pipeline "ngc-go/packages/compiler/src/template/pipeline/src/compilation"
)

// CreateI18nContexts creates one helper context op per i18n block (including generate descending blocks).
// Also, if an ICU exists inside an i18n block that also contains other localizable content (such as
// string), create an additional helper context op for the ICU.
// These context ops are later used for generating i18n messages. (Although we generate at least one
// context op per nested view, we will collect them up the tree later, to generate a top-level
// message.)
func CreateI18nContexts(job *pipeline.CompilationJob) {
	// Create i18n context ops for i18n attrs.
	attrContextByMessage := make(map[*i18n.Message]ir_operation.XrefId)
	for _, unit := range job.GetUnits() {
		for op := unit.GetCreate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			switch op.GetKind() {
			case ir.OpKindBinding:
				bindingOp, ok := op.(*ops_update.BindingOp)
				if !ok {
					continue
				}
				if bindingOp.I18nMessage == nil {
					continue
				}
				if _, exists := attrContextByMessage[bindingOp.I18nMessage]; !exists {
					i18nContext, err := ops_create.NewI18nContextOp(
						ir.I18nContextKindAttr,
						job.AllocateXrefId(),
						0, // i18nBlock - not needed for attr context
						bindingOp.I18nMessage,
						nil, // sourceSpan
					)
					if err != nil {
						panic(err)
					}
					unit.GetCreate().Push(i18nContext)
					attrContextByMessage[bindingOp.I18nMessage] = i18nContext.Xref
				}
				bindingOp.I18nContext = attrContextByMessage[bindingOp.I18nMessage]
			case ir.OpKindProperty:
				// PropertyOp doesn't have I18nMessage field directly, skip for now
				// TODO: Check if PropertyOp needs i18n context
				continue
			case ir.OpKindAttribute:
				// AttributeOp doesn't have I18nMessage field directly, skip for now
				// TODO: Check if AttributeOp needs i18n context
				continue
			case ir.OpKindExtractedAttribute:
				extractedAttrOp, ok := op.(*ops_create.ExtractedAttributeOp)
				if !ok {
					continue
				}
				if extractedAttrOp.I18nMessage == nil {
					continue
				}
				if _, exists := attrContextByMessage[extractedAttrOp.I18nMessage]; !exists {
					i18nContext, err := ops_create.NewI18nContextOp(
						ir.I18nContextKindAttr,
						job.AllocateXrefId(),
						0, // i18nBlock - not needed for attr context
						extractedAttrOp.I18nMessage,
						nil, // sourceSpan
					)
					if err != nil {
						panic(err)
					}
					unit.GetCreate().Push(i18nContext)
					attrContextByMessage[extractedAttrOp.I18nMessage] = i18nContext.Xref
				}
				extractedAttrOp.I18nContext = attrContextByMessage[extractedAttrOp.I18nMessage]
			}
		}
		for op := unit.GetUpdate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			switch op.GetKind() {
			case ir.OpKindBinding:
				bindingOp, ok := op.(*ops_update.BindingOp)
				if !ok {
					continue
				}
				if bindingOp.I18nMessage == nil {
					continue
				}
				if _, exists := attrContextByMessage[bindingOp.I18nMessage]; !exists {
					i18nContext, err := ops_create.NewI18nContextOp(
						ir.I18nContextKindAttr,
						job.AllocateXrefId(),
						0, // i18nBlock - not needed for attr context
						bindingOp.I18nMessage,
						nil, // sourceSpan
					)
					if err != nil {
						panic(err)
					}
					unit.GetCreate().Push(i18nContext)
					attrContextByMessage[bindingOp.I18nMessage] = i18nContext.Xref
				}
				bindingOp.I18nContext = attrContextByMessage[bindingOp.I18nMessage]
			case ir.OpKindProperty:
				// PropertyOp doesn't have I18nMessage field directly, skip for now
				continue
			case ir.OpKindAttribute:
				// AttributeOp doesn't have I18nMessage field directly, skip for now
				continue
			case ir.OpKindExtractedAttribute:
				extractedAttrOp, ok := op.(*ops_create.ExtractedAttributeOp)
				if !ok {
					continue
				}
				if extractedAttrOp.I18nMessage == nil {
					continue
				}
				if _, exists := attrContextByMessage[extractedAttrOp.I18nMessage]; !exists {
					i18nContext, err := ops_create.NewI18nContextOp(
						ir.I18nContextKindAttr,
						job.AllocateXrefId(),
						0, // i18nBlock - not needed for attr context
						extractedAttrOp.I18nMessage,
						nil, // sourceSpan
					)
					if err != nil {
						panic(err)
					}
					unit.GetCreate().Push(i18nContext)
					attrContextByMessage[extractedAttrOp.I18nMessage] = i18nContext.Xref
				}
				extractedAttrOp.I18nContext = attrContextByMessage[extractedAttrOp.I18nMessage]
			}
		}
	}

	// Create i18n context ops for root i18n blocks.
	blockContextByI18nBlock := make(map[ir_operation.XrefId]*ops_create.I18nContextOp)
	for _, unit := range job.GetUnits() {
		for op := unit.GetCreate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			if op.GetKind() == ir.OpKindI18nStart {
				i18nStartOp, ok := op.(*ops_create.I18nStartOp)
				if !ok {
					continue
				}
				if i18nStartOp.Xref == i18nStartOp.Root {
					i18nContext, err := ops_create.NewI18nContextOp(
						ir.I18nContextKindRootI18n,
						job.AllocateXrefId(),
						i18nStartOp.Xref,
						i18nStartOp.Message,
						nil, // sourceSpan
					)
					if err != nil {
						panic(err)
					}
					unit.GetCreate().Push(i18nContext)
					i18nStartOp.Context = i18nContext.Xref
					blockContextByI18nBlock[i18nStartOp.Xref] = i18nContext
				}
			}
		}
	}

	// Assign i18n contexts for child i18n blocks. These don't need their own context, instead they
	// should inherit from their root i18n block.
	for _, unit := range job.GetUnits() {
		for op := unit.GetCreate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			if op.GetKind() == ir.OpKindI18nStart {
				i18nStartOp, ok := op.(*ops_create.I18nStartOp)
				if !ok {
					continue
				}
				if i18nStartOp.Xref != i18nStartOp.Root {
					rootContext, exists := blockContextByI18nBlock[i18nStartOp.Root]
					if !exists {
						panic("AssertionError: Root i18n block i18n context should have been created.")
					}
					i18nStartOp.Context = rootContext.Xref
					blockContextByI18nBlock[i18nStartOp.Xref] = rootContext
				}
			}
		}
	}

	// Create or assign i18n contexts for ICUs.
	var currentI18nOp *ops_create.I18nStartOp = nil
	for _, unit := range job.GetUnits() {
		for op := unit.GetCreate().Head(); op != nil && op.GetKind() != ir.OpKindListEnd; op = op.Next() {
			switch op.GetKind() {
			case ir.OpKindI18nStart:
				if i18nStartOp, ok := op.(*ops_create.I18nStartOp); ok {
					currentI18nOp = i18nStartOp
				}
			case ir.OpKindI18nEnd:
				currentI18nOp = nil
			case ir.OpKindIcuStart:
				if currentI18nOp == nil {
					panic("AssertionError: Unexpected ICU outside of an i18n block.")
				}
				icuStartOp, ok := op.(*ops_create.IcuStartOp)
				if !ok {
					continue
				}
				if icuStartOp.Message.ID != currentI18nOp.Message.ID {
					// This ICU is a sub-message inside its parent i18n block message. We need to give it
					// its own context.
					i18nContext, err := ops_create.NewI18nContextOp(
						ir.I18nContextKindIcu,
						job.AllocateXrefId(),
						currentI18nOp.Root,
						icuStartOp.Message,
						nil, // sourceSpan
					)
					if err != nil {
						panic(err)
					}
					unit.GetCreate().Push(i18nContext)
					icuStartOp.Context = i18nContext.Xref
				} else {
					// This ICU is the only translatable content in its parent i18n block. We need to
					// convert the parent's context into an ICU context.
					icuStartOp.Context = currentI18nOp.Context
					rootContext := blockContextByI18nBlock[currentI18nOp.Xref]
					if rootContext != nil {
						rootContext.ContextKind = ir.I18nContextKindIcu
					}
				}
			}
		}
	}
}
