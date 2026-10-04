// Licensed under the MIT license.
// Concrete MPBQ instantiation of scoped_numeral[_vector].h and MPBQ macros.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"reflect"
)

type ScopedBinary struct {
	manager *BinaryRationalManager
	num     BinaryRational
	self    *ScopedBinary
	closed  bool
}

func NewScopedBinary(m *BinaryRationalManager) (*ScopedBinary, error) {
	if e := m.open(); e != nil {
		return nil, e
	}
	if e := m.z.budget.Reserve(uint64(reflect.TypeOf(ScopedBinary{}).Size())); e != nil {
		return nil, e
	}
	s := &ScopedBinary{manager: m}
	s.self = s
	return s, nil
}
func (s *ScopedBinary) Get() (*BinaryRational, error) {
	if s == nil || s.self != s {
		return nil, ErrScopedCopy
	}
	if s.closed {
		return nil, arena.ErrClosed
	}
	if e := s.manager.check(&s.num); e != nil {
		return nil, e
	}
	return &s.num, nil
}
func (s *ScopedBinary) Manager() (*BinaryRationalManager, error) {
	if _, e := s.Get(); e != nil {
		return nil, e
	}
	return s.manager, nil
}
func (s *ScopedBinary) Set(v *BinaryRational) error {
	if _, e := s.Get(); e != nil {
		return e
	}
	return s.manager.Copy(&s.num, v)
}
func (s *ScopedBinary) SetInt32(n int32) error {
	if _, e := s.Get(); e != nil {
		return e
	}
	return s.manager.SetInt32(&s.num, n)
}
func (s *ScopedBinary) SetInteger(n *Int) error {
	if _, e := s.Get(); e != nil {
		return e
	}
	return s.manager.SetInteger(&s.num, n)
}
func (s *ScopedBinary) Assign(other *ScopedBinary) error {
	if _, e := s.Get(); e != nil {
		return e
	}
	if _, e := other.Get(); e != nil {
		return e
	}
	if s.manager != other.manager {
		return ErrScopedOwner
	}
	if s == other {
		return nil
	}
	return s.Set(&other.num)
}
func (s *ScopedBinary) Clone() (*ScopedBinary, error) {
	if _, e := s.Get(); e != nil {
		return nil, e
	}
	r, e := NewScopedBinary(s.manager)
	if e != nil {
		return nil, e
	}
	if e = r.Set(&s.num); e != nil {
		r.Close()
		return nil, e
	}
	return r, nil
}
func (s *ScopedBinary) Reset() error {
	if _, e := s.Get(); e != nil {
		return e
	}
	return s.manager.Reset(&s.num)
}
func (s *ScopedBinary) Swap(other *ScopedBinary) error {
	if _, e := s.Get(); e != nil {
		return e
	}
	if _, e := other.Get(); e != nil {
		return e
	}
	if s.manager != other.manager {
		return ErrScopedOwner
	}
	return s.manager.Swap(&s.num, &other.num)
}
func (s *ScopedBinary) SwapValue(other *BinaryRational) error {
	if _, e := s.Get(); e != nil {
		return e
	}
	return s.manager.Swap(&s.num, other)
}
func (s *ScopedBinary) Close() error {
	if s == nil || s.self != s {
		return ErrScopedCopy
	}
	if s.closed {
		return nil
	}
	if e := s.manager.Del(&s.num); e != nil {
		return e
	}
	s.manager.z.budget.Release(uint64(reflect.TypeOf(ScopedBinary{}).Size()))
	s.closed = true
	return nil
}
func (s *ScopedBinary) Neg() error {
	if _, e := s.Get(); e != nil {
		return e
	}
	return s.manager.Neg(&s.num)
}
func (s *ScopedBinary) IsZero() (bool, error) {
	if _, e := s.Get(); e != nil {
		return false, e
	}
	return s.manager.IsZero(&s.num)
}
func (s *ScopedBinary) IsPos() (bool, error) {
	if _, e := s.Get(); e != nil {
		return false, e
	}
	return s.manager.IsPos(&s.num)
}
func (s *ScopedBinary) IsNeg() (bool, error) {
	if _, e := s.Get(); e != nil {
		return false, e
	}
	return s.manager.IsNeg(&s.num)
}
func (s *ScopedBinary) IsNonPos() (bool, error) {
	if _, e := s.Get(); e != nil {
		return false, e
	}
	return s.manager.IsNonPos(&s.num)
}
func (s *ScopedBinary) IsNonNeg() (bool, error) {
	if _, e := s.Get(); e != nil {
		return false, e
	}
	return s.manager.IsNonNeg(&s.num)
}
func (s *ScopedBinary) String() (string, error) {
	if _, e := s.Get(); e != nil {
		return "", e
	}
	return s.manager.String(&s.num)
}
func (s *ScopedBinary) compound(other *ScopedBinary, op uint8) error {
	if _, e := s.Get(); e != nil {
		return e
	}
	if _, e := other.Get(); e != nil {
		return e
	}
	if s.manager != other.manager {
		return ErrScopedOwner
	}
	return s.compoundRaw(&other.num, op)
}
func (s *ScopedBinary) compoundRaw(other *BinaryRational, op uint8) error {
	if _, e := s.Get(); e != nil {
		return e
	}
	switch op {
	case 0:
		return s.manager.Add(&s.num, other, &s.num)
	case 1:
		return s.manager.Sub(&s.num, other, &s.num)
	default:
		return s.manager.Mul(&s.num, other, &s.num)
	}
}
func (s *ScopedBinary) AddAssign(other *ScopedBinary) error        { return s.compound(other, 0) }
func (s *ScopedBinary) SubAssign(other *ScopedBinary) error        { return s.compound(other, 1) }
func (s *ScopedBinary) MulAssign(other *ScopedBinary) error        { return s.compound(other, 2) }
func (s *ScopedBinary) AddAssignValue(other *BinaryRational) error { return s.compoundRaw(other, 0) }
func (s *ScopedBinary) SubAssignValue(other *BinaryRational) error { return s.compoundRaw(other, 1) }
func (s *ScopedBinary) MulAssignValue(other *BinaryRational) error { return s.compoundRaw(other, 2) }
func (s *ScopedBinary) value(other *ScopedBinary, op uint8) (*ScopedBinary, error) {
	if _, e := other.Get(); e != nil {
		return nil, e
	}
	if _, e := s.Get(); e != nil {
		return nil, e
	}
	if s.manager != other.manager {
		return nil, ErrScopedOwner
	}
	return s.valueRaw(&other.num, op)
}
func (s *ScopedBinary) valueRaw(other *BinaryRational, op uint8) (*ScopedBinary, error) {
	r, e := s.Clone()
	if e != nil {
		return nil, e
	}
	defer r.Close()
	if e = r.compoundRaw(other, op); e != nil {
		return nil, e
	}
	return r.Clone()
}
func (s *ScopedBinary) Add(other *ScopedBinary) (*ScopedBinary, error) { return s.value(other, 0) }
func (s *ScopedBinary) Sub(other *ScopedBinary) (*ScopedBinary, error) { return s.value(other, 1) }
func (s *ScopedBinary) Mul(other *ScopedBinary) (*ScopedBinary, error) { return s.value(other, 2) }
func (s *ScopedBinary) AddValue(other *BinaryRational) (*ScopedBinary, error) {
	return s.valueRaw(other, 0)
}
func (s *ScopedBinary) SubValue(other *BinaryRational) (*ScopedBinary, error) {
	return s.valueRaw(other, 1)
}
func (s *ScopedBinary) MulValue(other *BinaryRational) (*ScopedBinary, error) {
	return s.valueRaw(other, 2)
}
func (s *ScopedBinary) macroValue(n *Int, op uint8) (*ScopedBinary, error) {
	if _, e := s.Get(); e != nil {
		return nil, e
	}
	if e := s.manager.z.check(n); e != nil {
		return nil, e
	}
	b, e := NewScopedBinary(s.manager)
	if e != nil {
		return nil, e
	}
	defer b.Close()
	if e = b.SetInteger(n); e != nil {
		return nil, e
	}
	r, e := NewScopedBinary(s.manager)
	if e != nil {
		return nil, e
	}
	switch op {
	case 0:
		e = s.manager.Add(&s.num, &b.num, &r.num)
	case 1:
		e = s.manager.Sub(&s.num, &b.num, &r.num)
	case 2:
		e = s.manager.Mul(&s.num, &b.num, &r.num)
	}
	if e != nil {
		r.Close()
		return nil, e
	}
	return r, nil
}
func (s *ScopedBinary) AddInt(n int32) (*ScopedBinary, error) {
	v := NewInt(n)
	return s.macroValue(&v, 0)
}
func (s *ScopedBinary) SubInt(n int32) (*ScopedBinary, error) {
	v := NewInt(n)
	return s.macroValue(&v, 1)
}
func (s *ScopedBinary) MulInt(n int32) (*ScopedBinary, error) {
	v := NewInt(n)
	return s.macroValue(&v, 2)
}
func (s *ScopedBinary) AddInteger(n *Int) (*ScopedBinary, error) { return s.macroValue(n, 0) }
func (s *ScopedBinary) SubInteger(n *Int) (*ScopedBinary, error) { return s.macroValue(n, 1) }
func (s *ScopedBinary) MulInteger(n *Int) (*ScopedBinary, error) { return s.macroValue(n, 2) }
func (s *ScopedBinary) compareRaw(b *BinaryRational, op uint8) (bool, error) {
	if _, e := s.Get(); e != nil {
		return false, e
	}
	switch op {
	case 0:
		return s.manager.Equal(&s.num, b)
	case 1:
		v, e := s.manager.Equal(&s.num, b)
		return !v, e
	case 2:
		return s.manager.Less(&s.num, b)
	case 3:
		return s.manager.LessEqual(&s.num, b)
	case 4:
		return s.manager.Greater(&s.num, b)
	default:
		return s.manager.GreaterEqual(&s.num, b)
	}
}
func (s *ScopedBinary) macroCompare(n *Int, op uint8) (bool, error) {
	if _, e := s.Get(); e != nil {
		return false, e
	}
	if e := s.manager.z.check(n); e != nil {
		return false, e
	}
	b, e := NewScopedBinary(s.manager)
	if e != nil {
		return false, e
	}
	defer b.Close()
	if e = b.SetInteger(n); e != nil {
		return false, e
	}
	if op == 1 {
		return s.manager.NotEqual(&s.num, &b.num)
	}
	return s.compareRaw(&b.num, op)
}

type ScopedBinaryVector struct {
	manager *BinaryRationalManager
	data    []BinaryRational
	self    *ScopedBinaryVector
	closed  bool
}

func NewScopedBinaryVector(m *BinaryRationalManager) (*ScopedBinaryVector, error) {
	if e := m.open(); e != nil {
		return nil, e
	}
	if e := m.z.budget.Reserve(uint64(reflect.TypeOf(ScopedBinaryVector{}).Size())); e != nil {
		return nil, e
	}
	v := &ScopedBinaryVector{manager: m}
	v.self = v
	return v, nil
}
func (v *ScopedBinaryVector) open() error {
	if v == nil || v.self != v {
		return ErrScopedCopy
	}
	if v.closed {
		return arena.ErrClosed
	}
	return v.manager.open()
}
func (v *ScopedBinaryVector) Size() uint32 {
	if v == nil || v.closed || v.self != v {
		return 0
	}
	return uint32(len(v.data))
}
func (v *ScopedBinaryVector) Capacity() uint32 {
	if v == nil || v.closed || v.self != v {
		return 0
	}
	return uint32(cap(v.data))
}
func (v *ScopedBinaryVector) At(i uint32) (*BinaryRational, error) {
	if e := v.open(); e != nil {
		return nil, e
	}
	if uint64(i) >= uint64(len(v.data)) {
		return nil, arena.ErrBounds
	}
	return &v.data[i], nil
}
func (v *ScopedBinaryVector) Manager() (*BinaryRationalManager, error) {
	if e := v.open(); e != nil {
		return nil, e
	}
	return v.manager, nil
}
func (v *ScopedBinaryVector) grow() error {
	width := uint64(reflect.TypeOf(BinaryRational{}).Size())
	n, e := scopedCapacity(cap(v.data), width)
	if e != nil {
		return e
	}
	if e = v.manager.z.budget.Reserve(uint64(n) * width); e != nil {
		return e
	}
	next := make([]BinaryRational, len(v.data), n)
	copy(next, v.data)
	old := v.data
	v.data = next
	for i := range old {
		old[i].num.cell = 0
	}
	v.manager.z.budget.Release(uint64(cap(old)) * width)
	return nil
}
func (v *ScopedBinaryVector) Push(a *BinaryRational) error {
	if e := v.open(); e != nil {
		return e
	}
	if e := v.manager.check(a); e != nil {
		return e
	}
	// A caller may refer to an element that growth transfers. Save only its
	// nonowning scalar view; the old storage remains a read input until Copy.
	input := *a
	if len(v.data) == cap(v.data) {
		if e := v.grow(); e != nil {
			return e
		}
	}
	v.data = append(v.data, BinaryRational{})
	return v.manager.Copy(&v.data[len(v.data)-1], &input)
}
func (v *ScopedBinaryVector) Shrink(n uint32) error {
	if e := v.open(); e != nil {
		return e
	}
	if uint64(n) > uint64(len(v.data)) {
		return arena.ErrBounds
	}
	for i := int(n); i < len(v.data); i++ {
		if e := v.manager.Del(&v.data[i]); e != nil {
			return e
		}
	}
	v.data = v.data[:n]
	return nil
}
func (v *ScopedBinaryVector) Pop() error {
	if e := v.open(); e != nil {
		return e
	}
	if len(v.data) == 0 {
		return arena.ErrBounds
	}
	return v.Shrink(uint32(len(v.data) - 1))
}
func (v *ScopedBinaryVector) Reset() error { return v.Shrink(0) }
func (v *ScopedBinaryVector) Resize(n uint32) error {
	if e := v.open(); e != nil {
		return e
	}
	if uint64(n) <= uint64(len(v.data)) {
		return v.Shrink(n)
	}
	for uint64(n) > uint64(cap(v.data)) {
		if e := v.grow(); e != nil {
			return e
		}
	}
	old := len(v.data)
	v.data = v.data[:n]
	for i := old; i < len(v.data); i++ {
		v.data[i] = BinaryRational{}
	}
	return nil
}
func (v *ScopedBinaryVector) Close() error {
	if v == nil || v.self != v {
		return ErrScopedCopy
	}
	if v.closed {
		return nil
	}
	if e := v.Reset(); e != nil {
		return e
	}
	v.manager.z.budget.Release(uint64(cap(v.data)) * uint64(reflect.TypeOf(BinaryRational{}).Size()))
	v.data = nil
	v.manager.z.budget.Release(uint64(reflect.TypeOf(ScopedBinaryVector{}).Size()))
	v.closed = true
	return nil
}
func (m *BinaryRationalManager) ResetVector(v *ScopedBinaryVector) error {
	if e := m.open(); e != nil {
		return e
	}
	if e := v.open(); e != nil {
		return e
	}
	if v.manager != m {
		return ErrScopedOwner
	}
	for i := range v.data {
		if e := m.Reset(&v.data[i]); e != nil {
			return e
		}
	}
	v.data = v.data[:0]
	return nil
}
func (s *ScopedBinary) Equal(b *ScopedBinary) (bool, error) {
	if _, e := b.Get(); e != nil {
		return false, e
	}
	if _, e := s.Get(); e != nil {
		return false, e
	}
	if b.manager != s.manager {
		return false, ErrScopedOwner
	}
	return s.compareRaw(&b.num, 0)
}
func (s *ScopedBinary) EqualValue(b *BinaryRational) (bool, error) { return s.compareRaw(b, 0) }
func (s *ScopedBinary) EqualInt(n int32) (bool, error)             { v := NewInt(n); return s.macroCompare(&v, 0) }
func (s *ScopedBinary) EqualInteger(n *Int) (bool, error)          { return s.macroCompare(n, 0) }
func (s *ScopedBinary) NotEqual(b *ScopedBinary) (bool, error) {
	if _, e := b.Get(); e != nil {
		return false, e
	}
	if _, e := s.Get(); e != nil {
		return false, e
	}
	if b.manager != s.manager {
		return false, ErrScopedOwner
	}
	return s.compareRaw(&b.num, 1)
}
func (s *ScopedBinary) NotEqualValue(b *BinaryRational) (bool, error) { return s.compareRaw(b, 1) }
func (s *ScopedBinary) NotEqualInt(n int32) (bool, error) {
	v := NewInt(n)
	return s.macroCompare(&v, 1)
}
func (s *ScopedBinary) NotEqualInteger(n *Int) (bool, error) { return s.macroCompare(n, 1) }
func (s *ScopedBinary) Less(b *ScopedBinary) (bool, error) {
	if _, e := b.Get(); e != nil {
		return false, e
	}
	if _, e := s.Get(); e != nil {
		return false, e
	}
	if b.manager != s.manager {
		return false, ErrScopedOwner
	}
	return s.compareRaw(&b.num, 2)
}
func (s *ScopedBinary) LessValue(b *BinaryRational) (bool, error) { return s.compareRaw(b, 2) }
func (s *ScopedBinary) LessInt(n int32) (bool, error)             { v := NewInt(n); return s.macroCompare(&v, 2) }
func (s *ScopedBinary) LessInteger(n *Int) (bool, error)          { return s.macroCompare(n, 2) }
func (s *ScopedBinary) LessEqual(b *ScopedBinary) (bool, error) {
	if _, e := b.Get(); e != nil {
		return false, e
	}
	if _, e := s.Get(); e != nil {
		return false, e
	}
	if b.manager != s.manager {
		return false, ErrScopedOwner
	}
	return s.compareRaw(&b.num, 3)
}
func (s *ScopedBinary) LessEqualValue(b *BinaryRational) (bool, error) { return s.compareRaw(b, 3) }
func (s *ScopedBinary) LessEqualInt(n int32) (bool, error) {
	v := NewInt(n)
	return s.macroCompare(&v, 3)
}
func (s *ScopedBinary) LessEqualInteger(n *Int) (bool, error) { return s.macroCompare(n, 3) }
func (s *ScopedBinary) Greater(b *ScopedBinary) (bool, error) {
	if _, e := b.Get(); e != nil {
		return false, e
	}
	if _, e := s.Get(); e != nil {
		return false, e
	}
	if b.manager != s.manager {
		return false, ErrScopedOwner
	}
	return s.compareRaw(&b.num, 4)
}
func (s *ScopedBinary) GreaterValue(b *BinaryRational) (bool, error) { return s.compareRaw(b, 4) }
func (s *ScopedBinary) GreaterInt(n int32) (bool, error) {
	v := NewInt(n)
	return s.macroCompare(&v, 4)
}
func (s *ScopedBinary) GreaterInteger(n *Int) (bool, error) { return s.macroCompare(n, 4) }
func (s *ScopedBinary) GreaterEqual(b *ScopedBinary) (bool, error) {
	if _, e := b.Get(); e != nil {
		return false, e
	}
	if _, e := s.Get(); e != nil {
		return false, e
	}
	if b.manager != s.manager {
		return false, ErrScopedOwner
	}
	return s.compareRaw(&b.num, 5)
}
func (s *ScopedBinary) GreaterEqualValue(b *BinaryRational) (bool, error) { return s.compareRaw(b, 5) }
func (s *ScopedBinary) GreaterEqualInt(n int32) (bool, error) {
	v := NewInt(n)
	return s.macroCompare(&v, 5)
}
func (s *ScopedBinary) GreaterEqualInteger(n *Int) (bool, error) { return s.macroCompare(n, 5) }
