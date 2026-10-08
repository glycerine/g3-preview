// Source: src/util/scoped_ptr_vector.h:24-53.
package util

import "reflect"

// ScopedPtrVector owns a sequence of dynamically typed Go pointers. The
// source class is templated over T*; this port validates pointer values at the
// boundary instead of introducing Go generics. destroy may be nil when garbage
// collection is the complete ownership action.
type ScopedPtrVector struct {
	self    *ScopedPtrVector
	destroy func(any)
	values  []any
	closed  bool
}

// NewScopedPtrVector constructs an empty owning pointer vector.
func NewScopedPtrVector(destroy func(any)) *ScopedPtrVector {
	vector := &ScopedPtrVector{destroy: destroy}
	vector.self = vector
	return vector
}

func validScopedPointer(value any) bool {
	if value == nil {
		return true
	}
	return reflect.ValueOf(value).Kind() == reflect.Pointer
}

func nilScopedPointer(value any) bool {
	return value == nil || validScopedPointer(value) && reflect.ValueOf(value).IsNil()
}

func sameScopedPointer(left, right any) bool {
	if nilScopedPointer(left) || nilScopedPointer(right) {
		return nilScopedPointer(left) && nilScopedPointer(right)
	}
	leftValue, rightValue := reflect.ValueOf(left), reflect.ValueOf(right)
	return leftValue.Type() == rightValue.Type() && leftValue.Pointer() == rightValue.Pointer()
}

func (vector *ScopedPtrVector) requireOpen() {
	if vector == nil || vector.self != vector || vector.closed {
		panic("g3: invalid scoped pointer vector owner")
	}
}

func (vector *ScopedPtrVector) destroyValue(value any) {
	if !nilScopedPointer(value) && vector.destroy != nil {
		vector.destroy(value)
	}
}

// Reset destroys values from first to last and retains an empty open owner.
func (vector *ScopedPtrVector) Reset() {
	vector.requireOpen()
	for i, value := range vector.values {
		vector.destroyValue(value)
		vector.values[i] = nil
	}
	vector.values = vector.values[:0]
}

// PushBack adopts pointer.
func (vector *ScopedPtrVector) PushBack(pointer any) {
	vector.requireOpen()
	if !validScopedPointer(pointer) {
		panic("g3: scoped pointer vector requires pointer values")
	}
	vector.values = append(vector.values, pointer)
}

// PopBack destroys and removes the last pointer.
func (vector *ScopedPtrVector) PopBack() {
	vector.requireOpen()
	if len(vector.values) == 0 {
		panic("g3: scoped pointer vector pop from empty vector")
	}
	last := len(vector.values) - 1
	vector.Set(uint32(last), nil)
	vector.values = vector.values[:last]
}

// At returns the pointer at index.
func (vector *ScopedPtrVector) At(index uint32) any {
	vector.requireOpen()
	if uint64(index) >= uint64(len(vector.values)) {
		panic("g3: scoped pointer vector index out of bounds")
	}
	return vector.values[index]
}

// Set destroys the old pointer before publishing pointer. Reassigning the
// identical pointer is a no-op.
func (vector *ScopedPtrVector) Set(index uint32, pointer any) {
	vector.requireOpen()
	if uint64(index) >= uint64(len(vector.values)) {
		panic("g3: scoped pointer vector index out of bounds")
	}
	if !validScopedPointer(pointer) {
		panic("g3: scoped pointer vector requires pointer values")
	}
	old := vector.values[index]
	if sameScopedPointer(old, pointer) {
		return
	}
	vector.destroyValue(old)
	vector.values[index] = pointer
}

// Size returns the number of owned slots.
func (vector *ScopedPtrVector) Size() uint32 {
	vector.requireOpen()
	return uint32(len(vector.values))
}

// Empty reports whether no slots are owned.
func (vector *ScopedPtrVector) Empty() bool { return vector.Size() == 0 }

// Resize shrinks with reverse-order destruction or grows with nil pointers.
func (vector *ScopedPtrVector) Resize(size uint32) {
	vector.requireOpen()
	target := int(size)
	if target < len(vector.values) {
		for i := len(vector.values); i > target; {
			i--
			vector.destroyValue(vector.values[i])
			vector.values[i] = nil
		}
		vector.values = vector.values[:target]
		return
	}
	for len(vector.values) < target {
		vector.values = append(vector.values, nil)
	}
}

// Close performs the source destructor and invalidates the owner.
func (vector *ScopedPtrVector) Close() {
	vector.requireOpen()
	vector.Reset()
	vector.closed = true
	vector.destroy = nil
}
