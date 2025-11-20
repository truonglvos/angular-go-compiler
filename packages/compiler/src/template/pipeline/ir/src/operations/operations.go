package ir_operation

import (
	"ngc-go/packages/compiler/src/template/pipeline/ir"
)

// XrefId is a branded type for a cross-reference ID
type XrefId int

// ConstIndex is a branded type for a constant index
type ConstIndex int

// Op is the base interface for semantic operations being performed within a template
type Op interface {
	GetKind() ir.OpKind
	GetPrev() Op
	SetPrev(op Op)
	GetNext() Op
	SetNext(op Op)
	GetDebugListId() *int
	SetDebugListId(id *int)
	// Next is a convenience method that calls GetNext()
	Next() Op
}

// OpList is a linked list of Op nodes
type OpList struct {
	debugListId int
	head        Op
	tail        Op
	nextListId  int
}

var nextListId = 0

// NewOpList creates a new OpList
func NewOpList() *OpList {
	listId := nextListId
	nextListId++
	head := &ListEndOp{
		debugListId: listId,
	}
	tail := &ListEndOp{
		debugListId: listId,
	}
	head.SetNext(tail)
	tail.SetPrev(head)
	return &OpList{
		debugListId: listId,
		head:        head,
		tail:        tail,
		nextListId:  listId,
	}
}

// ListEndOp is a special operation type used to represent the beginning and end nodes of a linked list
type ListEndOp struct {
	prev        Op
	next        Op
	debugListId int
}

// GetKind returns the operation kind
func (l *ListEndOp) GetKind() ir.OpKind {
	return ir.OpKindListEnd
}

// GetPrev returns the previous operation
func (l *ListEndOp) GetPrev() Op {
	return l.prev
}

// SetPrev sets the previous operation
func (l *ListEndOp) SetPrev(op Op) {
	l.prev = op
}

// GetNext returns the next operation
func (l *ListEndOp) GetNext() Op {
	return l.next
}

// Next is a convenience method that calls GetNext()
func (l *ListEndOp) Next() Op {
	return l.GetNext()
}

// SetNext sets the next operation
func (l *ListEndOp) SetNext(op Op) {
	l.next = op
}

// GetDebugListId returns the debug list ID
func (l *ListEndOp) GetDebugListId() *int {
	return &l.debugListId
}

// SetDebugListId sets the debug list ID
func (l *ListEndOp) SetDebugListId(id *int) {
	if id == nil {
		// Cannot set nil for ListEndOp as it uses int, not *int
		// This should not happen in practice
		return
	}
	l.debugListId = *id
}

// CreateOp is a base interface for creation operations
type CreateOp interface {
	Op
	GetXref() XrefId
	SetXref(xref XrefId)
}

// UpdateOp is a base interface for update operations
type UpdateOp interface {
	Op
	GetXref() XrefId
	SetXref(xref XrefId)
}

// OpBase is a base struct for operations
type OpBase struct {
	prev        Op
	next        Op
	debugListId *int
}

// NewOpBase creates a new OpBase
func NewOpBase() OpBase {
	return OpBase{
		prev:        nil,
		next:        nil,
		debugListId: nil,
	}
}

// GetPrev returns the previous operation
func (o *OpBase) GetPrev() Op {
	return o.prev
}

// SetPrev sets the previous operation
func (o *OpBase) SetPrev(op Op) {
	o.prev = op
}

// GetNext returns the next operation
func (o *OpBase) GetNext() Op {
	return o.next
}

// Next is a convenience method that calls GetNext()
func (o *OpBase) Next() Op {
	return o.GetNext()
}

// SetNext sets the next operation
func (o *OpBase) SetNext(op Op) {
	o.next = op
}

// GetDebugListId returns the debug list ID
func (o *OpBase) GetDebugListId() *int {
	return o.debugListId
}

// SetDebugListId sets the debug list ID
func (o *OpBase) SetDebugListId(id *int) {
	o.debugListId = id
}

// Head returns the head of the list
func (l *OpList) Head() Op {
	return l.head
}

// Tail returns the tail of the list
func (l *OpList) Tail() Op {
	return l.tail
}

// Push adds an operation to the tail of the list
func (l *OpList) Push(op Op) {
	if op.GetKind() == ir.OpKindListEnd {
		panic("cannot push list end node")
	}
	if op.GetDebugListId() != nil {
		panic("operation is already owned by a list")
	}

	listId := l.debugListId
	op.SetDebugListId(&listId)

	prev := l.tail.GetPrev()
	prev.SetNext(op)
	op.SetPrev(prev)
	op.SetNext(l.tail)
	l.tail.SetPrev(op)
}

// InsertBefore inserts a new operation before a given Op
func (l *OpList) InsertBefore(op Op, newOp Op) {
	if newOp.GetKind() == ir.OpKindListEnd {
		panic("cannot insert list end node")
	}
	if newOp.GetDebugListId() != nil {
		panic("operation is already owned by a list")
	}
	if op.GetDebugListId() == nil || *op.GetDebugListId() != l.debugListId {
		panic("operation is not owned by this list")
	}

	listId := l.debugListId
	newOp.SetDebugListId(&listId)

	prev := op.GetPrev()
	prev.SetNext(newOp)
	newOp.SetPrev(prev)
	newOp.SetNext(op)
	op.SetPrev(newOp)
}

// InsertAfter inserts a new operation after a given Op
func (l *OpList) InsertAfter(op Op, newOp Op) {
	if newOp.GetKind() == ir.OpKindListEnd {
		panic("cannot insert list end node")
	}
	if newOp.GetDebugListId() != nil {
		panic("operation is already owned by a list")
	}
	if op.GetDebugListId() == nil || *op.GetDebugListId() != l.debugListId {
		panic("operation is not owned by this list")
	}

	listId := l.debugListId
	newOp.SetDebugListId(&listId)

	next := op.GetNext()
	op.SetNext(newOp)
	newOp.SetPrev(op)
	newOp.SetNext(next)
	next.SetPrev(newOp)
}

// Remove removes an operation from the list
func (l *OpList) Remove(op Op) {
	if op.GetKind() == ir.OpKindListEnd {
		panic("cannot remove list end node")
	}
	if op.GetDebugListId() == nil || *op.GetDebugListId() != l.debugListId {
		panic("operation is not owned by this list")
	}

	prev := op.GetPrev()
	next := op.GetNext()
	prev.SetNext(next)
	next.SetPrev(prev)
	op.SetPrev(nil)
	op.SetNext(nil)
	op.SetDebugListId(nil)
}

// Replace replaces an operation with a new one
func (l *OpList) Replace(oldOp Op, newOp Op) {
	if newOp.GetKind() == ir.OpKindListEnd {
		panic("cannot replace with list end node")
	}
	if newOp.GetDebugListId() != nil {
		panic("new operation is already owned by a list")
	}
	if oldOp.GetKind() == ir.OpKindListEnd {
		panic("cannot replace list end node")
	}
	if oldOp.GetDebugListId() == nil || *oldOp.GetDebugListId() != l.debugListId {
		panic("old operation is not owned by this list")
	}

	listId := l.debugListId
	newOp.SetDebugListId(&listId)

	prev := oldOp.GetPrev()
	next := oldOp.GetNext()
	prev.SetNext(newOp)
	newOp.SetPrev(prev)
	newOp.SetNext(next)
	next.SetPrev(newOp)
	oldOp.SetPrev(nil)
	oldOp.SetNext(nil)
	oldOp.SetDebugListId(nil)
}

// Prepend prepends operations to the head of the list
func (l *OpList) Prepend(ops []Op) {
	// Insert in reverse order so they appear in the correct order
	for i := len(ops) - 1; i >= 0; i-- {
		op := ops[i]
		if op.GetKind() == ir.OpKindListEnd {
			panic("cannot prepend list end node")
		}
		if op.GetDebugListId() != nil {
			panic("operation is already owned by a list")
		}

		listId := l.debugListId
		op.SetDebugListId(&listId)

		head := l.head.GetNext()
		l.head.SetNext(op)
		op.SetPrev(l.head)
		op.SetNext(head)
		head.SetPrev(op)
	}
}
