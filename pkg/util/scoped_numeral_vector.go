// Source: src/util/scoped_numeral_vector.h:24-68.
package util

import "github.com/glycerine/g3/pkg/arena"

// ScopedNumeralVector is the non-generic Go form of
// _scoped_numeral_vector. It owns each numeral and borrows its manager.
type ScopedNumeralVector struct {
	self   *ScopedNumeralVector
	buffer *ScopedNumeralBuffer
	closed bool
}

func NewScopedNumeralVector(manager *ScopedNumeralManager) (*ScopedNumeralVector, error) {
	buffer, err := NewScopedNumeralBuffer(manager)
	if err != nil {
		return nil, err
	}
	buffer.values = nil
	vector := &ScopedNumeralVector{buffer: buffer}
	vector.self = vector
	return vector, nil
}

func (vector *ScopedNumeralVector) requireOpen() error {
	if vector == nil || vector.self != vector {
		return ErrScopedCopy
	}
	if vector.closed {
		return arena.ErrClosed
	}
	return nil
}

func (vector *ScopedNumeralVector) Manager() (*ScopedNumeralManager, error) {
	if err := vector.requireOpen(); err != nil {
		return nil, err
	}
	return vector.buffer.Manager()
}

func (vector *ScopedNumeralVector) Size() uint32 {
	if err := vector.requireOpen(); err != nil {
		panic(err)
	}
	return vector.buffer.Size()
}

func (vector *ScopedNumeralVector) Capacity() uint32 {
	if err := vector.requireOpen(); err != nil {
		panic(err)
	}
	return uint32(cap(vector.buffer.values))
}

func (vector *ScopedNumeralVector) Empty() bool { return vector.Size() == 0 }

func (vector *ScopedNumeralVector) Get(index uint32) (*any, error) {
	if err := vector.requireOpen(); err != nil {
		return nil, err
	}
	return vector.buffer.Get(index)
}

func (vector *ScopedNumeralVector) CPtr() ([]any, error) {
	if err := vector.requireOpen(); err != nil {
		return nil, err
	}
	return vector.buffer.CPtr()
}

func (vector *ScopedNumeralVector) PushBack(value any) error {
	if err := vector.requireOpen(); err != nil {
		return err
	}
	return vector.buffer.PushBack(value)
}

func (vector *ScopedNumeralVector) PopBack() error {
	if err := vector.requireOpen(); err != nil {
		return err
	}
	if vector.buffer.Empty() {
		return arena.ErrBounds
	}
	return vector.buffer.Shrink(vector.buffer.Size() - 1)
}

func (vector *ScopedNumeralVector) Reset() error {
	if err := vector.requireOpen(); err != nil {
		return err
	}
	return vector.buffer.Reset()
}

func (vector *ScopedNumeralVector) Shrink(size uint32) error {
	if err := vector.requireOpen(); err != nil {
		return err
	}
	return vector.buffer.Shrink(size)
}

func (vector *ScopedNumeralVector) Resize(size uint32) error {
	if err := vector.requireOpen(); err != nil {
		return err
	}
	return vector.buffer.Resize(size)
}

func (vector *ScopedNumeralVector) Close() error {
	if vector == nil || vector.self != vector {
		return ErrScopedCopy
	}
	if vector.closed {
		return nil
	}
	if err := vector.buffer.Close(); err != nil {
		return err
	}
	vector.buffer = nil
	vector.closed = true
	return nil
}
