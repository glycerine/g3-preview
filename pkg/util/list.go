// Source: src/util/list.h:25-92.
package util

// List is the project list node used for short persistent lists. Value is kept
// as any because the source class is a template and this port does not use Go
// generics. Retained source specializations store either scalar test values or
// pointers, so callers recover their concrete type with a type assertion.
//
// Go reachability replaces the source region allocator. AppendList copies the
// first list's nodes and shares the second list exactly as the source does.
type List struct {
	head any
	tail *List
}

// NewList constructs a node with head followed by tail.
func NewList(head any, tail *List) *List {
	return &List{head: head, tail: tail}
}

// Head returns the node value.
func (l *List) Head() any {
	if l == nil {
		panic("g3: head of empty list")
	}
	return l.head
}

// SetHead replaces the node value.
func (l *List) SetHead(head any) {
	if l == nil {
		panic("g3: set head of empty list")
	}
	l.head = head
}

// Tail returns the next node.
func (l *List) Tail() *List {
	if l == nil {
		panic("g3: tail of empty list")
	}
	return l.tail
}

// SetTail replaces the next node.
func (l *List) SetTail(tail *List) {
	if l == nil {
		panic("g3: set tail of empty list")
	}
	l.tail = tail
}

// ListIterator is the source list::iterator translated to an ordinary Go
// cursor. A zero iterator is the end iterator.
type ListIterator struct {
	current *List
}

// Begin returns an iterator positioned at l. It also makes a nil list's begin
// equal to End, which is the natural Go representation of an empty list.
func (l *List) Begin() ListIterator { return ListIterator{current: l} }

// End returns the zero end iterator.
func (l *List) End() ListIterator { return ListIterator{} }

// Done reports whether the iterator is at the end.
func (it ListIterator) Done() bool { return it.current == nil }

// Value returns the current value.
func (it ListIterator) Value() any {
	if it.current == nil {
		panic("g3: dereference of end list iterator")
	}
	return it.current.head
}

// Advance implements source prefix increment.
func (it *ListIterator) Advance() {
	if it == nil || it.current == nil {
		panic("g3: advance of end list iterator")
	}
	it.current = it.current.tail
}

// PostAdvance implements source postfix increment and returns the previous
// iterator value.
func (it *ListIterator) PostAdvance() ListIterator {
	previous := *it
	it.Advance()
	return previous
}

// Equal implements source iterator equality.
func (it ListIterator) Equal(other ListIterator) bool { return it.current == other.current }

// NotEqual implements source iterator inequality.
func (it ListIterator) NotEqual(other ListIterator) bool { return it.current != other.current }

// ListLength returns the number of nodes reachable from l.
func ListLength(l *List) uint32 {
	var length uint32
	for ; l != nil; l = l.tail {
		length++
	}
	return length
}

// AppendList non-destructively appends second to first. When second is nil the
// source returns first unchanged. Otherwise the first prefix is copied and its
// final node points at the original second list.
func AppendList(first, second *List) *List {
	if second == nil {
		return first
	}
	prefix := make([]*List, 0, ListLength(first))
	for node := first; node != nil; node = node.tail {
		prefix = append(prefix, node)
	}
	result := second
	for i := len(prefix) - 1; i >= 0; i-- {
		result = NewList(prefix[i].head, result)
	}
	return result
}
