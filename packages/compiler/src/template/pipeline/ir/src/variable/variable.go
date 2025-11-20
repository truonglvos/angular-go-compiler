package ir_variable

import (
	"ngc-go/packages/compiler/src/output"
	"ngc-go/packages/compiler/src/template/pipeline/ir"
	ir_operations "ngc-go/packages/compiler/src/template/pipeline/ir/src/operations"
)

// CTX_REF is a marker constant for context reference
const CTX_REF = "CTX_REF_MARKER"

// SemanticVariable is a union type for the different kinds of variables
type SemanticVariable interface {
	GetKind() ir.SemanticVariableKind
	GetName() *string
	SetName(name string)
}

// SemanticVariableBase is the base struct for semantic variables
type SemanticVariableBase struct {
	Kind ir.SemanticVariableKind
	Name *string
}

// GetKind returns the variable kind
func (s *SemanticVariableBase) GetKind() ir.SemanticVariableKind {
	return s.Kind
}

// GetName returns the variable name
func (s *SemanticVariableBase) GetName() *string {
	return s.Name
}

// SetName sets the variable name
func (s *SemanticVariableBase) SetName(name string) {
	s.Name = &name
}

// ContextVariable represents the context of a particular view
type ContextVariable struct {
	SemanticVariableBase
	View ir_operations.XrefId
}

// NewContextVariable creates a new ContextVariable
func NewContextVariable(view ir_operations.XrefId) *ContextVariable {
	return &ContextVariable{
		SemanticVariableBase: SemanticVariableBase{
			Kind: ir.SemanticVariableKindContext,
		},
		View: view,
	}
}

// IdentifierVariable represents a specific identifier within a template
type IdentifierVariable struct {
	SemanticVariableBase
	Identifier string
	Local      bool
}

// NewIdentifierVariable creates a new IdentifierVariable
func NewIdentifierVariable(identifier string, local bool) *IdentifierVariable {
	return &IdentifierVariable{
		SemanticVariableBase: SemanticVariableBase{
			Kind: ir.SemanticVariableKindIdentifier,
		},
		Identifier: identifier,
		Local:      local,
	}
}

// SavedViewVariable represents a saved view context
type SavedViewVariable struct {
	SemanticVariableBase
	View ir_operations.XrefId
}

// NewSavedViewVariable creates a new SavedViewVariable
func NewSavedViewVariable(view ir_operations.XrefId) *SavedViewVariable {
	return &SavedViewVariable{
		SemanticVariableBase: SemanticVariableBase{
			Kind: ir.SemanticVariableKindSavedView,
		},
		View: view,
	}
}

// AliasVariable will be inlined at every location it is used
type AliasVariable struct {
	SemanticVariableBase
	Identifier string
	Expression output.OutputExpression
}

// NewAliasVariable creates a new AliasVariable
func NewAliasVariable(identifier string, expression output.OutputExpression) *AliasVariable {
	return &AliasVariable{
		SemanticVariableBase: SemanticVariableBase{
			Kind: ir.SemanticVariableKindAlias,
		},
		Identifier: identifier,
		Expression: expression,
	}
}
