// Source: src/util/ptr_scoped_buffer.h:23-52.
package util

// PtrScopedBuffer owns pointer values in a growable Go slice. The source T*
// template becomes validated dynamic Go pointers; destroy may be nil when Go
// garbage collection is the complete ownership action.
type PtrScopedBuffer struct {
	self    *PtrScopedBuffer
	destroy func(any)
	values  []any
	closed  bool
}

// NewPtrScopedBuffer constructs an empty buffer with the source default inline
// capacity. Go may grow the backing slice using its own allocation policy.
func NewPtrScopedBuffer(destroy func(any)) *PtrScopedBuffer {
	buffer := &PtrScopedBuffer{destroy: destroy, values: make([]any, 0, 16)}
	buffer.self = buffer
	return buffer
}

func (buffer *PtrScopedBuffer) requireOpen() {
	if buffer == nil || buffer.self != buffer || buffer.closed {
		panic("g3: invalid scoped pointer buffer owner")
	}
}

func (buffer *PtrScopedBuffer) destroyValue(value any) {
	if !nilScopedPointer(value) && buffer.destroy != nil {
		buffer.destroy(value)
	}
}

func (buffer *PtrScopedBuffer) deallocateAll() {
	for i, value := range buffer.values {
		buffer.destroyValue(value)
		buffer.values[i] = nil
	}
}

// Reset destroys all values first-to-last and retains reusable slice capacity.
func (buffer *PtrScopedBuffer) Reset() {
	buffer.requireOpen()
	buffer.deallocateAll()
	buffer.values = buffer.values[:0]
}

// Finalize destroys all values and releases the backing slice.
func (buffer *PtrScopedBuffer) Finalize() {
	buffer.requireOpen()
	buffer.deallocateAll()
	buffer.values = nil
}

// Release relinquishes ownership without invoking destroy. Clearing the Go
// slots prevents the non-owning buffer from keeping released objects alive.
func (buffer *PtrScopedBuffer) Release() {
	buffer.requireOpen()
	clear(buffer.values)
	buffer.values = buffer.values[:0]
}

func (buffer *PtrScopedBuffer) Size() uint32 {
	buffer.requireOpen()
	return uint32(len(buffer.values))
}

func (buffer *PtrScopedBuffer) Empty() bool { return buffer.Size() == 0 }

func (buffer *PtrScopedBuffer) Begin() PtrScopedBufferIterator {
	buffer.requireOpen()
	return PtrScopedBufferIterator{buffer: buffer}
}

func (buffer *PtrScopedBuffer) End() PtrScopedBufferIterator {
	buffer.requireOpen()
	return PtrScopedBufferIterator{buffer: buffer, index: uint32(len(buffer.values))}
}

func (buffer *PtrScopedBuffer) PushBack(pointer any) {
	buffer.requireOpen()
	if !validScopedPointer(pointer) {
		panic("g3: scoped pointer buffer requires pointer values")
	}
	buffer.values = append(buffer.values, pointer)
}

func (buffer *PtrScopedBuffer) Back() any {
	buffer.requireOpen()
	if len(buffer.values) == 0 {
		panic("g3: scoped pointer buffer back from empty buffer")
	}
	return buffer.values[len(buffer.values)-1]
}

func (buffer *PtrScopedBuffer) PopBack() {
	buffer.requireOpen()
	if len(buffer.values) == 0 {
		panic("g3: scoped pointer buffer pop from empty buffer")
	}
	last := len(buffer.values) - 1
	buffer.destroyValue(buffer.values[last])
	buffer.values[last] = nil
	buffer.values = buffer.values[:last]
}

func (buffer *PtrScopedBuffer) Get(index uint32) any {
	buffer.requireOpen()
	if uint64(index) >= uint64(len(buffer.values)) {
		panic("g3: scoped pointer buffer index out of bounds")
	}
	return buffer.values[index]
}

// Set destroys the old value before publishing a distinct replacement.
func (buffer *PtrScopedBuffer) Set(index uint32, pointer any) {
	buffer.requireOpen()
	if uint64(index) >= uint64(len(buffer.values)) {
		panic("g3: scoped pointer buffer index out of bounds")
	}
	if !validScopedPointer(pointer) {
		panic("g3: scoped pointer buffer requires pointer values")
	}
	old := buffer.values[index]
	if sameScopedPointer(old, pointer) {
		return
	}
	buffer.destroyValue(old)
	buffer.values[index] = pointer
}

// Append adopts every supplied pointer after validating the complete input.
func (buffer *PtrScopedBuffer) Append(pointers []any) {
	buffer.requireOpen()
	for _, pointer := range pointers {
		if !validScopedPointer(pointer) {
			panic("g3: scoped pointer buffer requires pointer values")
		}
	}
	buffer.values = append(buffer.values, pointers...)
}

// Close performs the source destructor and invalidates the owner.
func (buffer *PtrScopedBuffer) Close() {
	buffer.requireOpen()
	buffer.Finalize()
	buffer.closed = true
	buffer.destroy = nil
}

type PtrScopedBufferIterator struct {
	buffer *PtrScopedBuffer
	index  uint32
}

func (iterator PtrScopedBufferIterator) Value() any {
	if iterator.buffer == nil || iterator.index >= iterator.buffer.Size() {
		panic("g3: dereference of end scoped pointer buffer iterator")
	}
	return iterator.buffer.Get(iterator.index)
}

func (iterator *PtrScopedBufferIterator) Advance() {
	if iterator == nil || iterator.buffer == nil || iterator.index >= iterator.buffer.Size() {
		panic("g3: advance of end scoped pointer buffer iterator")
	}
	iterator.index++
}

func (iterator PtrScopedBufferIterator) Equal(other PtrScopedBufferIterator) bool {
	return iterator.buffer == other.buffer && iterator.index == other.index
}

func (iterator PtrScopedBufferIterator) NotEqual(other PtrScopedBufferIterator) bool {
	return !iterator.Equal(other)
}
