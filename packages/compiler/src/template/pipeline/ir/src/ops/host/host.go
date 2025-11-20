package ops_host

import (
	"ngc-go/packages/compiler/src/output"
	"ngc-go/packages/compiler/src/template/pipeline/ir"
	ir_operations "ngc-go/packages/compiler/src/template/pipeline/ir/src/operations"
	"ngc-go/packages/compiler/src/util"
)

// DomPropertyOp is a logical operation representing a binding to a native DOM property
type DomPropertyOp struct {
	ir_operations.OpBase
	Name            string
	Expression      interface{} // output.OutputExpression | *Interpolation
	BindingKind     ir.BindingKind
	I18nContext     ir_operations.XrefId
	SecurityContext interface{} // core.SecurityContext | []core.SecurityContext
	Sanitizer       output.OutputExpression
	SourceSpan      *util.ParseSourceSpan
}

// NewDomPropertyOp creates a new DomPropertyOp
func NewDomPropertyOp(
	name string,
	expression interface{}, // output.OutputExpression | *Interpolation
	bindingKind ir.BindingKind,
	i18nContext ir_operations.XrefId,
	securityContext interface{}, // core.SecurityContext | []core.SecurityContext
	sourceSpan *util.ParseSourceSpan,
) *DomPropertyOp {
	return &DomPropertyOp{
		OpBase:          ir_operations.NewOpBase(),
		Name:            name,
		Expression:      expression,
		BindingKind:     bindingKind,
		I18nContext:     i18nContext,
		SecurityContext: securityContext,
		Sanitizer:       nil,
		SourceSpan:      sourceSpan,
	}
}

// GetKind returns the operation kind
func (d *DomPropertyOp) GetKind() ir.OpKind {
	return ir.OpKindDomProperty
}

// GetXref returns 0 (DomPropertyOp doesn't have a target xref in host context)
func (d *DomPropertyOp) GetXref() ir_operations.XrefId {
	return 0
}

// SetXref does nothing (DomPropertyOp doesn't have a target xref in host context)
func (d *DomPropertyOp) SetXref(xref ir_operations.XrefId) {
	// No-op
}

// HasConsumesVarsTrait implements ConsumesVarsTraitInterface
func (d *DomPropertyOp) HasConsumesVarsTrait() bool {
	return true
}
