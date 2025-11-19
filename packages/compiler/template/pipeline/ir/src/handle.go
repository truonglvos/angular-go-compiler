package ir

// SlotHandle represents a slot handle
type SlotHandle struct {
	Slot *int
}

// NewSlotHandle creates a new SlotHandle
func NewSlotHandle() *SlotHandle {
	return &SlotHandle{
		Slot: nil,
	}
}

