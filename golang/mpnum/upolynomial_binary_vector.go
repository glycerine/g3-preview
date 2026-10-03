// Copyright (c) 2011 Microsoft Corporation. Licensed under the MIT license.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"math"
	"reflect"
)

type upolyBinaryVectorState struct {
	values            []BinaryRational
	size, initialized uint32
}
type BinaryRationalVector struct {
	self    *BinaryRationalVector
	manager *BinaryRationalManager
	state   upolyBinaryVectorState
	header  uint64
	closed  bool
}
type BinaryRationalVectorStats struct {
	OwnerBytes, BackingBytes, TotalBytes           uint64
	Size, Capacity, Initialized, OwnedNumeralCells uint32
}

func upolyIsolationSourceCapacity(old, stride uint32) (n, oldExtent, newExtent uint32, e error) {
	if old == 0 {
		return 2, 0, 2*stride + 8, nil
	}
	oldExtent = stride*old + 8
	n = (3*old + 1) >> 1
	newExtent = stride*n + 8
	if n <= old || newExtent <= oldExtent {
		e = arena.ErrBounds
	}
	return
}
func upolyIsolationNextCapacity(old, sourceStride uint32, actualStride, host uint64) (uint32, uint64, error) {
	n, _, _, e := upolyIsolationSourceCapacity(old, sourceStride)
	if e != nil {
		return 0, 0, e
	}
	if uint64(n)*uint64(sourceStride)+8 > math.MaxUint32 || actualStride == 0 || uint64(n) > host/actualStride {
		return 0, 0, arena.ErrBounds
	}
	return n, uint64(n) * actualStride, nil
}
func (m *BinaryRationalManager) NewVector() (*BinaryRationalVector, error) {
	if e := m.open(); e != nil {
		return nil, e
	}
	h := uint64(reflect.TypeOf(BinaryRationalVector{}).Size())
	if e := m.z.budget.Reserve(h); e != nil {
		return nil, e
	}
	v := &BinaryRationalVector{manager: m, header: h}
	v.self = v
	return v, nil
}
func (v *BinaryRationalVector) check() error {
	if v == nil || v.self != v || v.manager == nil {
		return ErrBinaryOwner
	}
	if v.closed {
		return arena.ErrClosed
	}
	return v.manager.open()
}
func (m *BinaryRationalManager) growBinaryVector(s *upolyBinaryVectorState) error {
	n, bytes, e := upolyIsolationNextCapacity(uint32(cap(s.values)), 24, uint64(reflect.TypeOf(BinaryRational{}).Size()), uint64(int(^uint(0)>>1)))
	if e != nil {
		return e
	}
	if e = m.z.budget.Reserve(bytes); e != nil {
		return e
	}
	old := s.values
	next := make([]BinaryRational, n)
	s.values = next
	for i := uint32(0); i < s.size; i++ {
		next[i] = old[i]
		old[i] = BinaryRational{}
	}
	m.z.budget.Release(uint64(cap(old)) * uint64(reflect.TypeOf(BinaryRational{}).Size()))
	s.initialized = s.size
	return nil
}
func (m *BinaryRationalManager) binaryPushDefault(s *upolyBinaryVectorState) (*BinaryRational, error) {
	if s.size == math.MaxUint32 {
		return nil, arena.ErrBounds
	}
	if s.size == uint32(cap(s.values)) {
		if e := m.growBinaryVector(s); e != nil {
			return nil, e
		}
	}
	i := s.size
	s.values[i] = NewBinaryRational()
	s.size++
	if s.size > s.initialized {
		s.initialized = s.size
	}
	return &s.values[i], nil
}
func (m *BinaryRationalManager) binarySlot(s *upolyBinaryVectorState, i uint32) (*BinaryRational, error) {
	if modularSourceDebug && i >= s.size {
		return nil, ErrUPolynomialAssertion
	}
	if i >= s.initialized {
		return nil, arena.ErrBounds
	}
	return &s.values[i], nil
}
func (m *BinaryRationalManager) binaryReset(s *upolyBinaryVectorState) error {
	for i := uint32(0); i < s.size; i++ {
		if e := m.Del(&s.values[i]); e != nil {
			return e
		}
	}
	s.size = 0
	return nil
}
func (m *BinaryRationalManager) binaryDestroy(s *upolyBinaryVectorState) error {
	if e := m.binaryReset(s); e != nil {
		return e
	}
	m.z.budget.Release(uint64(cap(s.values)) * uint64(reflect.TypeOf(BinaryRational{}).Size()))
	*s = upolyBinaryVectorState{}
	return nil
}
func (v *BinaryRationalVector) PushCopy(a *BinaryRational) error {
	if e := v.check(); e != nil {
		return e
	}
	dst, e := v.manager.binaryPushDefault(&v.state)
	if e != nil {
		return e
	}
	return v.manager.Copy(dst, a)
}
func (v *BinaryRationalVector) Size() (uint32, error) {
	if e := v.check(); e != nil {
		return 0, e
	}
	return v.state.size, nil
}
func (v *BinaryRationalVector) Read(i uint32, out *BinaryRational) error {
	if e := v.check(); e != nil {
		return e
	}
	a, e := v.manager.binarySlot(&v.state, i)
	if e != nil {
		return e
	}
	return v.manager.Copy(out, a)
}
func (v *BinaryRationalVector) Set(i uint32, a *BinaryRational) error {
	if e := v.check(); e != nil {
		return e
	}
	dst, e := v.manager.binarySlot(&v.state, i)
	if e != nil {
		return e
	}
	return v.manager.Copy(dst, a)
}
func (v *BinaryRationalVector) Reset() error {
	if e := v.check(); e != nil {
		return e
	}
	return v.manager.binaryReset(&v.state)
}
func (v *BinaryRationalVector) Shrink(n uint32) error {
	if e := v.check(); e != nil {
		return e
	}
	old := v.state.size
	if old == n {
		return nil
	}
	for i := n; i < old; i++ {
		if e := v.manager.Del(&v.state.values[i]); e != nil {
			return e
		}
	}
	if v.state.values == nil {
		if modularSourceDebug && n != 0 {
			return ErrUPolynomialAssertion
		}
		return nil
	}
	if modularSourceDebug && n > old {
		return ErrUPolynomialAssertion
	}
	if n > v.state.initialized {
		return arena.ErrBounds
	}
	v.state.size = n
	return nil
}
func (v *BinaryRationalVector) Resize(n uint32) error {
	if e := v.check(); e != nil {
		return e
	}
	old := v.state.size
	if n <= old {
		if e := v.Shrink(n); e != nil {
			return e
		}
		return nil
	}
	for n > uint32(cap(v.state.values)) {
		if e := v.manager.growBinaryVector(&v.state); e != nil {
			return e
		}
	}
	v.state.size = n
	for i := old; i < n; i++ {
		v.state.values[i] = NewBinaryInt(0)
	}
	if n > v.state.initialized {
		v.state.initialized = n
	}
	return nil
}

// Actual vector::reserve grows logical size and default constructs elements.
func (v *BinaryRationalVector) Reserve(n uint32) error {
	if e := v.check(); e != nil {
		return e
	}
	if n > v.state.size {
		return v.Resize(n)
	}
	return nil
}
func (v *BinaryRationalVector) Close() error {
	if e := v.check(); e != nil {
		return e
	}
	if e := v.manager.binaryDestroy(&v.state); e != nil {
		return e
	}
	v.manager.z.budget.Release(v.header)
	v.header = 0
	v.closed = true
	return nil
}
func (v *BinaryRationalVector) Stats() BinaryRationalVectorStats {
	if v.check() != nil {
		return BinaryRationalVectorStats{}
	}
	s := &v.state
	r := BinaryRationalVectorStats{OwnerBytes: v.header, BackingBytes: uint64(cap(s.values)) * uint64(reflect.TypeOf(BinaryRational{}).Size()), Size: s.size, Capacity: uint32(cap(s.values)), Initialized: s.initialized}
	r.TotalBytes = r.OwnerBytes + r.BackingBytes
	for i := uint32(0); i < s.size; i++ {
		if validNumericRef(v.manager.z.cells, s.values[i].num.cell) {
			r.OwnedNumeralCells++
		}
	}
	return r
}
func (m *UPolynomialManager) isolationOwners(b *BinaryRationalManager, r, l, u *BinaryRationalVector) error {
	if e := m.check(); e != nil {
		return e
	}
	if b == nil || b.z != m.config.Integer {
		return ErrUPolynomialOwner
	}
	if e := b.open(); e != nil {
		return e
	}
	for _, v := range [3]*BinaryRationalVector{r, l, u} {
		if e := v.check(); e != nil {
			return e
		}
		if v.manager != b {
			return ErrUPolynomialOwner
		}
	}
	return nil
}
