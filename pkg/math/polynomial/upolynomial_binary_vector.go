package polynomial

import (
	"math"
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

type upolyBinaryVectorState struct {
	values []unumeral.BinaryRational
	size   uint32
}
type BinaryRationalVector struct {
	self    *BinaryRationalVector
	manager *unumeral.BinaryRationalManager
	state   upolyBinaryVectorState
	header  uint64
	closed  bool
}
type BinaryRationalVectorStats struct {
	OwnerBytes, BackingBytes, TotalBytes           uint64
	Size, Capacity, Initialized, OwnedNumeralCells uint32
}

func numericBinaryRationalManager_NewVector(m *unumeral.BinaryRationalManager) (*BinaryRationalVector, error) {
	if e := m.Open(); e != nil {
		return nil, e
	}
	h := uint64(reflect.TypeOf(BinaryRationalVector{}).Size())
	if e := m.Integers().Budget().Reserve(h); e != nil {
		return nil, e
	}
	v := &BinaryRationalVector{manager: m, header: h}
	v.self = v
	return v, nil
}
func (v *BinaryRationalVector) check() error {
	if v == nil || v.self != v || v.manager == nil {
		return unumeral.ErrBinaryOwner
	}
	if v.closed {
		return arena.ErrClosed
	}
	return v.manager.Open()
}

// Native field extent is retained for borrowed reads and owning cleanup;
// logical size separately records the currently active binary rationals.
func upolyBinaryExtend(s *upolyBinaryVectorState, n uint32) error {
	if uint64(n) > uint64(^uint(0)>>1) {
		return arena.ErrBounds
	}
	old := s.values
	if int(n) > len(old) {
		s.values = append(old, make([]unumeral.BinaryRational, int(n)-len(old))...)
		if len(old) != 0 && &old[0] != &s.values[0] {
			clear(old)
		}
	}
	clear(s.values[s.size:n])
	s.size = n
	return nil
}
func numericBinaryRationalManager_binaryPushDefault(m *unumeral.BinaryRationalManager, s *upolyBinaryVectorState) (*unumeral.BinaryRational, error) {
	if s.size == math.MaxUint32 {
		return nil, arena.ErrBounds
	}
	i := s.size
	if e := upolyBinaryExtend(s, i+1); e != nil {
		return nil, e
	}
	return &s.values[i], nil
}
func numericBinaryRationalManager_binarySlot(m *unumeral.BinaryRationalManager, s *upolyBinaryVectorState, i uint32) (*unumeral.BinaryRational, error) {
	if polynomialSourceDebug && i >= s.size {
		return nil, ErrUPolynomialAssertion
	}
	if uint64(i) >= uint64(len(s.values)) {
		return nil, arena.ErrBounds
	}
	return &s.values[i], nil
}
func numericBinaryRationalManager_binaryReset(m *unumeral.BinaryRationalManager, s *upolyBinaryVectorState) error {
	for i := uint32(0); i < s.size; i++ {
		if e := m.Del(&s.values[i]); e != nil {
			return e
		}
	}
	s.size = 0
	return nil
}
func numericBinaryRationalManager_binaryDestroy(m *unumeral.BinaryRationalManager, s *upolyBinaryVectorState) error {
	if e := numericBinaryRationalManager_binaryReset(m, s); e != nil {
		return e
	}
	*s = upolyBinaryVectorState{}
	return nil
}
func (v *BinaryRationalVector) PushCopy(a *unumeral.BinaryRational) error {
	if e := v.check(); e != nil {
		return e
	}
	dst, e := numericBinaryRationalManager_binaryPushDefault(v.manager, &v.state)
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
func (v *BinaryRationalVector) Read(i uint32, out *unumeral.BinaryRational) error {
	if e := v.check(); e != nil {
		return e
	}
	a, e := numericBinaryRationalManager_binarySlot(v.manager, &v.state, i)
	if e != nil {
		return e
	}
	return v.manager.Copy(out, a)
}
func (v *BinaryRationalVector) Set(i uint32, a *unumeral.BinaryRational) error {
	if e := v.check(); e != nil {
		return e
	}
	dst, e := numericBinaryRationalManager_binarySlot(v.manager, &v.state, i)
	if e != nil {
		return e
	}
	return v.manager.Copy(dst, a)
}
func (v *BinaryRationalVector) Reset() error {
	if e := v.check(); e != nil {
		return e
	}
	return numericBinaryRationalManager_binaryReset(v.manager, &v.state)
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
		if polynomialSourceDebug && n != 0 {
			return ErrUPolynomialAssertion
		}
		return nil
	}
	if polynomialSourceDebug && n > old {
		return ErrUPolynomialAssertion
	}
	if uint64(n) > uint64(len(v.state.values)) {
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
	return upolyBinaryExtend(&v.state, n)
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
	if e := numericBinaryRationalManager_binaryDestroy(v.manager, &v.state); e != nil {
		return e
	}
	v.manager.Integers().Budget().Release(v.header)
	v.header = 0
	v.closed = true
	return nil
}
func (v *BinaryRationalVector) Stats() BinaryRationalVectorStats {
	if v.check() != nil {
		return BinaryRationalVectorStats{}
	}
	s := &v.state
	r := BinaryRationalVectorStats{OwnerBytes: v.header, Size: s.size, Capacity: uint32(cap(s.values)), Initialized: uint32(len(s.values))}
	r.TotalBytes = r.OwnerBytes + r.BackingBytes
	for i := uint32(0); i < s.size; i++ {
		if v.manager.Integers().IsLiveCellToken((*s.values[i].BorrowNumerator()).NativeCellToken()) {
			r.OwnedNumeralCells++
		}
	}
	return r
}
func (m *UPolynomialManager) isolationOwners(b *unumeral.BinaryRationalManager, r, l, u *BinaryRationalVector) error {
	if e := m.check(); e != nil {
		return e
	}
	if b == nil || b.Integers() != m.config.Integer {
		return ErrUPolynomialOwner
	}
	if e := b.Open(); e != nil {
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
