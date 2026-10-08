// Source: src/util/ref_buffer.h:29-171.
package util

// RefBuffer is the non-generic Go form of ref_buffer_core, ref_buffer and
// sref_buffer. Values must be nil or Go pointers. inc and dec receive every
// source callback argument, including nil; a concrete wrapper may ignore nil.
type RefBuffer struct {
	self     *RefBuffer
	manager  any
	inc, dec func(any)
	values   []any
	closed   bool
}

// NewRefBuffer constructs an empty reference buffer. manager is retained for
// concrete consumers that need the source get_manager/m accessors.
func NewRefBuffer(manager any, inc, dec func(any)) *RefBuffer {
	buffer := &RefBuffer{manager: manager, inc: inc, dec: dec, values: make([]any, 0, 16)}
	buffer.self = buffer
	return buffer
}

func (buffer *RefBuffer) requireOpen() {
	if buffer == nil || buffer.self != buffer || buffer.closed {
		panic("g3: invalid reference buffer owner")
	}
}

func (buffer *RefBuffer) requirePointer(value any) {
	if !validScopedPointer(value) {
		panic("g3: reference buffer requires pointer values")
	}
}

func (buffer *RefBuffer) increment(value any) {
	if buffer.inc != nil {
		buffer.inc(value)
	}
}

func (buffer *RefBuffer) decrement(value any) {
	if buffer.dec != nil {
		buffer.dec(value)
	}
}

func (buffer *RefBuffer) decRange(begin, end int) {
	for i := begin; i < end; i++ {
		buffer.decrement(buffer.values[i])
	}
}

// PushBack increments before publishing the new pointer.
func (buffer *RefBuffer) PushBack(value any) {
	buffer.requireOpen()
	buffer.requirePointer(value)
	buffer.increment(value)
	buffer.values = append(buffer.values, value)
}

// PopBack removes the pointer before decrementing it.
func (buffer *RefBuffer) PopBack() {
	buffer.requireOpen()
	if len(buffer.values) == 0 {
		panic("g3: reference buffer pop from empty buffer")
	}
	last := len(buffer.values) - 1
	value := buffer.values[last]
	buffer.values[last] = nil
	buffer.values = buffer.values[:last]
	buffer.decrement(value)
}

func (buffer *RefBuffer) Back() any {
	buffer.requireOpen()
	if len(buffer.values) == 0 {
		panic("g3: reference buffer back from empty buffer")
	}
	return buffer.values[len(buffer.values)-1]
}

// BackRef exposes the source mutable back reference and bypasses callbacks.
func (buffer *RefBuffer) BackRef() *any {
	buffer.requireOpen()
	if len(buffer.values) == 0 {
		panic("g3: reference buffer back from empty buffer")
	}
	return &buffer.values[len(buffer.values)-1]
}

// CPtr returns the live Go slice corresponding to the source raw pointer. A
// caller must not retain it across structural edits or mutate it without
// arranging reference callbacks.
func (buffer *RefBuffer) CPtr() []any {
	buffer.requireOpen()
	return buffer.values
}

func (buffer *RefBuffer) Get(index uint32) any {
	buffer.requireOpen()
	if uint64(index) >= uint64(len(buffer.values)) {
		panic("g3: reference buffer index out of bounds")
	}
	return buffer.values[index]
}

// Set increments the incoming pointer, decrements the old pointer, then
// publishes the incoming pointer. Identical pointers still receive both calls.
func (buffer *RefBuffer) Set(index uint32, value any) {
	buffer.requireOpen()
	if uint64(index) >= uint64(len(buffer.values)) {
		panic("g3: reference buffer index out of bounds")
	}
	buffer.requirePointer(value)
	buffer.increment(value)
	buffer.decrement(buffer.values[index])
	buffer.values[index] = value
}

func (buffer *RefBuffer) Size() uint32 {
	buffer.requireOpen()
	return uint32(len(buffer.values))
}

func (buffer *RefBuffer) Empty() bool { return buffer.Size() == 0 }

// Reset decrements values first-to-last and retains reusable slice capacity.
func (buffer *RefBuffer) Reset() {
	buffer.requireOpen()
	buffer.decRange(0, len(buffer.values))
	clear(buffer.values)
	buffer.values = buffer.values[:0]
}

// Finalize decrements values first-to-last and releases the backing slice.
func (buffer *RefBuffer) Finalize() {
	buffer.requireOpen()
	buffer.decRange(0, len(buffer.values))
	clear(buffer.values)
	buffer.values = nil
}

// Append increments and appends values in order. Self append uses a captured
// copy so slice growth cannot change the source range.
func (buffer *RefBuffer) Append(values []any) {
	buffer.requireOpen()
	captured := append([]any(nil), values...)
	for _, value := range captured {
		buffer.PushBack(value)
	}
}

func (buffer *RefBuffer) AppendBuffer(other *RefBuffer) {
	buffer.requireOpen()
	other.requireOpen()
	buffer.Append(other.values)
}

// Resize decrements a removed suffix in increasing index order or appends nil
// slots without increment callbacks.
func (buffer *RefBuffer) Resize(size uint32) {
	buffer.requireOpen()
	target := int(size)
	if target < len(buffer.values) {
		buffer.decRange(target, len(buffer.values))
		clear(buffer.values[target:])
		buffer.values = buffer.values[:target]
		return
	}
	for len(buffer.values) < target {
		buffer.values = append(buffer.values, nil)
	}
}

func (buffer *RefBuffer) Shrink(size uint32) {
	buffer.requireOpen()
	if uint64(size) > uint64(len(buffer.values)) {
		panic("g3: reference buffer shrink grows buffer")
	}
	buffer.Resize(size)
}

func (buffer *RefBuffer) SetX(index uint32, value any) {
	buffer.requireOpen()
	if index >= uint32(len(buffer.values)) {
		buffer.Resize(index + 1)
	}
	buffer.Set(index, value)
}

// Assign implements source copy assignment with a self-assignment no-op.
func (buffer *RefBuffer) Assign(other *RefBuffer) {
	buffer.requireOpen()
	other.requireOpen()
	if buffer == other {
		return
	}
	buffer.Reset()
	buffer.Append(other.values)
}

// Clone implements the intended managed copy constructor with shared manager
// callbacks and independent storage.
func (buffer *RefBuffer) Clone() *RefBuffer {
	buffer.requireOpen()
	clone := NewRefBuffer(buffer.manager, buffer.inc, buffer.dec)
	clone.Append(buffer.values)
	return clone
}

func (buffer *RefBuffer) Manager() any {
	buffer.requireOpen()
	return buffer.manager
}

// Close performs the source destructor exactly once.
func (buffer *RefBuffer) Close() {
	buffer.requireOpen()
	buffer.Finalize()
	buffer.closed = true
	buffer.manager, buffer.inc, buffer.dec = nil, nil, nil
}
