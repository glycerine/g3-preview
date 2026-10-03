// Copyright (c) 2011 Microsoft Corporation. Source: src/util/mpbq.h/.cpp.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"reflect"
)

// BinaryRational owns its numerator in its implicit IntegerManager domain.
// A zero value is valid. Never shallow-copy owning big values. Constructors
// preserve source raw shape; denominator-power setters normalize.
type BinaryRational struct {
	num Int
	k   uint32
}

func NewBinaryRational() BinaryRational                  { return BinaryRational{} }
func NewBinaryInt(n int32) BinaryRational                { return BinaryRational{num: NewInt(n)} }
func NewBinaryIntPower(n int32, k uint32) BinaryRational { return BinaryRational{num: NewInt(n), k: k} }

type BinaryRationalConfig struct{ Diagnostics *RationalService }

// K is the source scalar accessor. Numerator extraction uses an explicit
// owning manager copy rather than exporting an owning raw-handle copy.
func (a *BinaryRational) K() uint32 { return a.k }

var ErrBinaryOwner = errors.New("mpnum: invalid binary-rational owner")
var ErrBinaryInvariant = errors.New("mpnum: binary-rational source invariant")
var ErrBinaryInterval = errors.New("mpnum: binary-rational source interval precondition")
var ErrBinaryRootDegreeZero = errors.New("mpnum: binary-rational root degree zero")

// This cold operation owner borrows z and diagnostics. Both must outlive it.
// No operations or shared Budget accesses are concurrently safe.
type BinaryRationalManager struct {
	z                                            *IntegerManager
	diagnostics                                  *RationalService
	self                                         *BinaryRationalManager
	tmp, tmp2                                    Int
	addMulTmp                                    BinaryRational
	selectIntTmp1, selectIntTmp2, selectSmallTmp Int
	selectSmallTmp1, selectSmallTmp2             BinaryRational
	divTmp1, divTmp2, divTmp3                    Int
	headerBytes                                  uint64
	closed                                       bool
}
type BinaryRationalStats struct {
	HeaderBytes           uint64
	LiveScratchNumerators uint32
}

func NewBinaryRationalManager(z *IntegerManager, c BinaryRationalConfig) (*BinaryRationalManager, error) {
	if z == nil || z.closed || z.cells == nil {
		return nil, ErrBinaryOwner
	}
	if e := binaryCheckDiagnostics(z, c.Diagnostics); e != nil {
		return nil, e
	}
	h := uint64(reflect.TypeOf(BinaryRationalManager{}).Size())
	if e := z.budget.Reserve(h); e != nil {
		return nil, e
	}
	m := &BinaryRationalManager{z: z, diagnostics: c.Diagnostics, headerBytes: h}
	m.self = m
	return m, nil
}
func (m *BinaryRationalManager) open() error {
	if m == nil || m.self != m {
		return ErrBinaryOwner
	}
	if m.closed || m.z.closed {
		return arena.ErrClosed
	}
	return nil
}
func (m *BinaryRationalManager) check(a *BinaryRational) error {
	if e := m.open(); e != nil {
		return e
	}
	if a == nil {
		return arena.ErrInvalidSlot
	}
	return m.z.check(&a.num)
}
func (m *BinaryRationalManager) checkCleanup(a *BinaryRational) error {
	if e := m.open(); e != nil {
		return e
	}
	if a == nil {
		return arena.ErrInvalidSlot
	}
	return m.z.checkCleanup(&a.num)
}
func (m *BinaryRationalManager) pair(a, b *BinaryRational) error {
	if e := m.check(a); e != nil {
		return e
	}
	return m.check(b)
}
func (m *BinaryRationalManager) triple(a, b, r *BinaryRational) error {
	if e := m.pair(a, b); e != nil {
		return e
	}
	return m.check(r)
}
func (m *BinaryRationalManager) Precise() bool { return true }
func (m *BinaryRationalManager) Field() bool   { return false }
func (m *BinaryRationalManager) Close() {
	if m == nil || m.self != m || m.closed {
		return
	}
	m.z.Del(&m.addMulTmp.num)
	m.z.Del(&m.tmp)
	m.z.Del(&m.tmp2)
	m.z.Del(&m.selectIntTmp1)
	m.z.Del(&m.selectIntTmp2)
	m.z.Del(&m.selectSmallTmp)
	m.z.Del(&m.selectSmallTmp1.num)
	m.z.Del(&m.selectSmallTmp2.num)
	m.z.Del(&m.divTmp1)
	m.z.Del(&m.divTmp2)
	m.z.Del(&m.divTmp3)
	m.z.budget.Release(m.headerBytes)
	m.headerBytes = 0
	m.closed = true
}
func (m *BinaryRationalManager) Stats() (BinaryRationalStats, error) {
	if e := m.open(); e != nil {
		return BinaryRationalStats{}, e
	}
	n := uint32(0)
	for _, v := range [...]Int{m.tmp, m.tmp2, m.addMulTmp.num, m.selectIntTmp1, m.selectIntTmp2, m.selectSmallTmp, m.selectSmallTmp1.num, m.selectSmallTmp2.num, m.divTmp1, m.divTmp2, m.divTmp3} {
		if validNumericRef(m.z.cells, v.cell) {
			n++
		}
	}
	return BinaryRationalStats{HeaderBytes: m.headerBytes, LiveScratchNumerators: n}, nil
}
func (m *BinaryRationalManager) Del(a *BinaryRational) error {
	if e := m.checkCleanup(a); e != nil {
		return e
	}
	return m.z.Del(&a.num)
}
func (m *BinaryRationalManager) Reset(a *BinaryRational) error {
	if e := m.checkCleanup(a); e != nil {
		return e
	}
	if e := m.z.Reset(&a.num); e != nil {
		return e
	}
	a.k = 0
	return nil
}
func (m *BinaryRationalManager) Swap(a, b *BinaryRational) error {
	if e := m.pair(a, b); e != nil {
		return e
	}
	*a, *b = *b, *a
	return nil
}
func (m *BinaryRationalManager) Copy(dst, src *BinaryRational) error {
	if e := m.pair(dst, src); e != nil {
		return e
	}
	if e := m.z.Copy(&dst.num, &src.num); e != nil {
		return e
	}
	dst.k = src.k
	return nil
}
func (m *BinaryRationalManager) SetInt32(a *BinaryRational, n int32) error {
	if e := m.check(a); e != nil {
		return e
	}
	if e := m.z.SetInt32(&a.num, n); e != nil {
		return e
	}
	a.k = 0
	return nil
}
func (m *BinaryRationalManager) SetUint32(a *BinaryRational, n uint32) error {
	if e := m.check(a); e != nil {
		return e
	}
	if e := m.z.SetUint32(&a.num, n); e != nil {
		return e
	}
	a.k = 0
	return nil
}
func (m *BinaryRationalManager) SetInt32Power(a *BinaryRational, n int32, k uint32) error {
	if e := m.SetInt32(a, n); e != nil {
		return e
	}
	a.k = k
	return m.normalize(a)
}
func (m *BinaryRationalManager) SetInt64Power(a *BinaryRational, n int64, k uint32) error {
	if e := m.check(a); e != nil {
		return e
	}
	if e := m.z.SetInt64(&a.num, n); e != nil {
		return e
	}
	a.k = k
	return m.normalize(a)
}
func (m *BinaryRationalManager) SetInteger(a *BinaryRational, n *Int) error {
	if e := m.check(a); e != nil {
		return e
	}
	if e := m.z.Copy(&a.num, n); e != nil {
		return e
	}
	a.k = 0
	return nil
}
func (m *BinaryRationalManager) SetIntegerPower(a *BinaryRational, n *Int, k uint32) error {
	if e := m.SetInteger(a, n); e != nil {
		return e
	}
	a.k = k
	return m.normalize(a)
}
func (m *BinaryRationalManager) Numerator(a *BinaryRational, n *Int) error {
	if e := m.check(a); e != nil {
		return e
	}
	return m.z.Copy(n, &a.num)
}
func (m *BinaryRationalManager) DenominatorPower(a *BinaryRational) (uint32, error) {
	if e := m.check(a); e != nil {
		return 0, e
	}
	return a.k, nil
}
func (m *BinaryRationalManager) IsInteger(a *BinaryRational) (bool, error) {
	if e := m.check(a); e != nil {
		return false, e
	}
	return a.k == 0, nil
}
func (m *BinaryRationalManager) IsZero(a *BinaryRational) (bool, error) {
	if e := m.check(a); e != nil {
		return false, e
	}
	return m.z.IsZero(&a.num)
}
func (m *BinaryRationalManager) IsNonzero(a *BinaryRational) (bool, error) {
	v, e := m.IsZero(a)
	return !v, e
}
func (m *BinaryRationalManager) IsOne(a *BinaryRational) (bool, error) {
	if e := m.check(a); e != nil {
		return false, e
	}
	if a.k != 0 {
		return false, nil
	}
	return m.z.IsOne(&a.num)
}
func (m *BinaryRationalManager) IsPos(a *BinaryRational) (bool, error) {
	if e := m.check(a); e != nil {
		return false, e
	}
	return m.z.IsPos(&a.num)
}
func (m *BinaryRationalManager) IsNeg(a *BinaryRational) (bool, error) {
	if e := m.check(a); e != nil {
		return false, e
	}
	return m.z.IsNeg(&a.num)
}
func (m *BinaryRationalManager) IsNonPos(a *BinaryRational) (bool, error) {
	if e := m.check(a); e != nil {
		return false, e
	}
	return m.z.IsNonPos(&a.num)
}
func (m *BinaryRationalManager) IsNonNeg(a *BinaryRational) (bool, error) {
	if e := m.check(a); e != nil {
		return false, e
	}
	return m.z.IsNonNeg(&a.num)
}
func (m *BinaryRationalManager) normalize(a *BinaryRational) error {
	if a.k == 0 {
		return nil
	}
	if a.num.val == 0 {
		a.k = 0
		return nil
	}
	old, e := m.debugValue(a)
	if e != nil {
		return e
	}
	defer m.debugDel(&old)
	k, e := m.z.PowerOfTwoMultiple(&a.num)
	if e != nil {
		return e
	}
	if k > a.k {
		k = a.k
	}
	if e = m.z.MachineDiv2k(&a.num, k, &a.num); e != nil {
		return e
	}
	a.k -= k
	return m.debugEqualValue(&old, a)
}

// Copy bridges retain source raw numerator/denominator shape and never read
// a foreign numeric handle through this owner. The caller supplies provenance.
func (m *BinaryRationalManager) importRational(qm *RationalManager, q *Rational, local *Rational) (*Rational, error) {
	if e := m.open(); e != nil {
		return nil, e
	}
	if qm == nil || qm.z == nil {
		return nil, ErrBinaryOwner
	}
	if e := qm.check(q); e != nil {
		return nil, e
	}
	if qm.z == m.z {
		return q, nil
	}
	if e := m.z.CopyFrom(&local.num, qm.z, &q.num); e != nil {
		return nil, e
	}
	if e := m.z.CopyFrom(&local.den, qm.z, &q.den); e != nil {
		return nil, e
	}
	return local, nil
}
func (m *BinaryRationalManager) dropImported(q *Rational) { m.z.Del(&q.num); m.z.Del(&q.den) }
