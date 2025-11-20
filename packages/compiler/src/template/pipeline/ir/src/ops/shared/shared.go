package ops

import (
	"ngc-go/packages/compiler/src/output"
	ir "ngc-go/packages/compiler/src/template/pipeline/ir/src"
	ir_operations "ngc-go/packages/compiler/src/template/pipeline/ir/src/operations"
)

// StatementOp is an `Op` which directly wraps an output `Statement`
type StatementOp struct {
	ir_operations.OpBase
	Statement output.OutputStatement
	Xref      ir_operations.XrefId
}

// NewStatementOp creates a new StatementOp
func NewStatementOp(statement output.OutputStatement) *StatementOp {
	return &StatementOp{
		OpBase:    ir_operations.NewOpBase(),
		Statement: statement,
		Xref:      0,
	}
}

// GetKind returns the operation kind
func (s *StatementOp) GetKind() ir.OpKind {
	return ir.OpKindStatement
}

// GetXref returns the xref ID
func (s *StatementOp) GetXref() ir_operations.XrefId {
	return s.Xref
}

// SetXref sets the xref ID
func (s *StatementOp) SetXref(xref ir_operations.XrefId) {
	s.Xref = xref
}

// VariableOp declares and initializes a `SemanticVariable`, that is valid either in create or update IR
type VariableOp struct {
	ir_operations.OpBase
	Xref        ir_operations.XrefId
	Variable    interface{} // variable.SemanticVariable
	Initializer output.OutputExpression
	Flags       ir.VariableFlags
}

// NewVariableOp creates a new VariableOp
func NewVariableOp(
	xref ir_operations.XrefId,
	variable interface{}, // variable.SemanticVariable
	initializer output.OutputExpression,
	flags ir.VariableFlags,
) *VariableOp {
	return &VariableOp{
		OpBase:      ir_operations.NewOpBase(),
		Xref:        xref,
		Variable:    variable,
		Initializer: initializer,
		Flags:       flags,
	}
}

// GetKind returns the operation kind
func (v *VariableOp) GetKind() ir.OpKind {
	return ir.OpKindVariable
}

// GetXref returns the xref ID
func (v *VariableOp) GetXref() ir_operations.XrefId {
	return v.Xref
}

// SetXref sets the xref ID
func (v *VariableOp) SetXref(xref ir_operations.XrefId) {
	v.Xref = xref
}
