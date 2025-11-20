package ir_traits

import (
	"ngc-go/packages/compiler/src/template/pipeline/ir"
	ir_operations "ngc-go/packages/compiler/src/template/pipeline/ir/src/operations"
	"ngc-go/packages/compiler/src/util"
)

// ConsumesSlot is a marker symbol for ConsumesSlotOpTrait
// In Go, we use a constant string instead of Symbol
const ConsumesSlot = "ConsumesSlot"

// DependsOnSlotContext is a marker symbol for DependsOnSlotContextOpTrait
const DependsOnSlotContext = "DependsOnSlotContext"

// ConsumesVarsTrait is a marker symbol for ConsumesVars trait
const ConsumesVarsTrait = "ConsumesVars"

// UsesVarOffset is a marker symbol for UsesVarOffset trait
const UsesVarOffset = "UsesVarOffset"

// ConsumesSlotOpTrait marks an operations as requiring allocation of one or more data slots for storage
type ConsumesSlotOpTrait struct {
	// Assigned data slot (the starting index, if more than one slot is needed) for this operations, or
	// `null` if slots have not yet been assigned.
	Handle *ir.SlotHandle

	// The number of slots which will be used by this operations. By default 1, but can be increased if
	// necessary.
	NumSlotsUsed int

	// `XrefId` of this operations (e.g. the element stored in the assigned slot). This `XrefId` is
	// used to link this `ConsumesSlotOpTrait` operations with `DependsOnSlotContextTrait` or
	// `UsesSlotIndexExprTrait` implementors and ensure that the assigned slot is propagated through
	// the IR to all consumers.
	Xref ir_operations.XrefId
}

// DependsOnSlotContextOpTrait marks an operations as depending on the runtime's implicit slot context being set to a particular slot
type DependsOnSlotContextOpTrait struct {
	// `XrefId` of the `ConsumesSlotOpTrait` which the implicit slot context must reference before
	// this operations can be executed.
	Target ir_operations.XrefId

	SourceSpan *util.ParseSourceSpan
}

// ConsumesVarsTraitInterface is a marker trait indicating that an operations or expression consumes variable storage space
type ConsumesVarsTraitInterface interface {
	HasConsumesVarsTrait() bool
}

// UsesVarOffsetTraitInterface is a marker trait indicating that an expression requires knowledge of the number of variable storage
// slots used prior to it
type UsesVarOffsetTraitInterface interface {
	GetVarOffset() *int
	SetVarOffset(offset int)
}

// HasConsumesSlotTrait tests whether an operations implements `ConsumesSlotOpTrait`
func HasConsumesSlotTrait(op ir_operations.Op) bool {
	_, ok := op.(interface {
		GetConsumesSlotTrait() *ConsumesSlotOpTrait
	})
	return ok
}

// HasDependsOnSlotContextTrait tests whether an operations or expression implements `DependsOnSlotContextOpTrait`
func HasDependsOnSlotContextTrait(value interface{}) bool {
	_, ok := value.(interface {
		GetDependsOnSlotContextTrait() *DependsOnSlotContextOpTrait
	})
	return ok
}

// HasConsumesVarsTrait tests whether an operations or expression implements `ConsumesVarsTrait`
func HasConsumesVarsTrait(value interface{}) bool {
	if trait, ok := value.(ConsumesVarsTraitInterface); ok {
		return trait.HasConsumesVarsTrait()
	}
	return false
}

// HasUsesVarOffsetTrait tests whether an expression implements `UsesVarOffsetTrait`
func HasUsesVarOffsetTrait(value interface{}) bool {
	_, ok := value.(UsesVarOffsetTraitInterface)
	return ok
}
