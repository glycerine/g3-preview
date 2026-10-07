// Source: src/math/polynomial/linear_eq_solver.h; src/util/vector.h.
package polynomial

import (
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// Numeral fields inhabit real Runtime allocations. The outer matrix stores
// compact row locators; rows is their charged cold release metadata, created
// only for actual default-constructed row members. It is not a numeric mirror.
type linearNumeralVector struct {
	raw, pending, old                         util.Block
	width, initialized, moved, oldInitialized uint32
	movedComponents                           uint8
	interrupted                               bool
}
type linearStorage struct {
	runtime                          *util.Runtime
	integer                          *util.IntegerManager
	A, b                             linearNumeralVector
	rows                             []linearNumeralVector
	n                                uint32
	header                           uint64
	sourceCause                      error
	sourceStage                      uint8
	flushStage                       uint8
	flushRow, flushColumn            uint32
	closing, sourceFinalized, closed bool
}

func (v *linearNumeralVector) size(rt *util.Runtime) (uint32, error) {
	if v.raw == (util.Block{}) {
		return 0, nil
	}
	return rt.ReadUint32(v.raw, 4)
}
func (v *linearNumeralVector) capacity(rt *util.Runtime) (uint32, error) {
	if v.raw == (util.Block{}) {
		return 0, nil
	}
	return rt.ReadUint32(v.raw, 0)
}
func linearRead64(rt *util.Runtime, b util.Block, o uint64) (uint64, error) {
	a, e := rt.ReadUint32(b, o)
	if e != nil {
		return 0, e
	}
	c, e := rt.ReadUint32(b, o+4)
	return uint64(a) | uint64(c)<<32, e
}
func linearWrite64(rt *util.Runtime, b util.Block, o, v uint64) error {
	if e := rt.WriteUint32(b, o, uint32(v)); e != nil {
		return e
	}
	return rt.WriteUint32(b, o+4, uint32(v>>32))
}
func linearReadInt(rt *util.Runtime, b util.Block, o uint64) (util.Int, error) {
	v, e := rt.ReadUint32(b, o)
	if e != nil {
		return util.Int{}, e
	}
	r, e := linearRead64(rt, b, o+4)
	return util.IntFromNativeWords([3]uint32{uint32(v), uint32(r), uint32(uint64(r) >> 32)}), e
}
func linearWriteInt(rt *util.Runtime, b util.Block, o uint64, a util.Int) error {
	if e := rt.WriteUint32(b, o, uint32(a.NativeScalar())); e != nil {
		return e
	}
	return linearWrite64(rt, b, o+4, uint64(a.NativeCellToken()))
}
func (v *linearNumeralVector) intAt(rt *util.Runtime, i uint32) (util.Int, error) {
	return linearReadInt(rt, v.raw, 8+uint64(i)*uint64(v.width))
}
func (v *linearNumeralVector) putInt(rt *util.Runtime, i uint32, a util.Int) error {
	return linearWriteInt(rt, v.raw, 8+uint64(i)*uint64(v.width), a)
}
func (v *linearNumeralVector) rationalAt(rt *util.Runtime, i uint32) (util.Rational, error) {
	o := 8 + uint64(i)*24
	a, e := linearReadInt(rt, v.raw, o)
	if e != nil {
		return util.Rational{}, e
	}
	b, e := linearReadInt(rt, v.raw, o+12)
	return util.RationalFromBorrowedNumerals(a, b), e
}
func (v *linearNumeralVector) putRational(rt *util.Runtime, i uint32, a util.Rational) error {
	o := 8 + uint64(i)*24
	if e := linearWriteInt(rt, v.raw, o, (*a.BorrowNumerator())); e != nil {
		return e
	}
	return linearWriteInt(rt, v.raw, o+12, (*a.BorrowDenominator()))
}
func (v *linearNumeralVector) rowAt(rt *util.Runtime, i uint32) (uint32, error) {
	return rt.ReadUint32(v.raw, 8+uint64(i)*4)
}
func (v *linearNumeralVector) putRow(rt *util.Runtime, i, r uint32) error {
	return rt.WriteUint32(v.raw, 8+uint64(i)*4, r)
}
func (v *linearNumeralVector) moveField(rt *util.Runtime, dst, src util.Block, i uint32) error {
	o := 8 + uint64(i)*uint64(v.width)
	if v.width == 4 {
		r, e := rt.ReadUint32(src, o)
		if e != nil {
			return e
		}
		if e = rt.WriteUint32(dst, o, r); e != nil {
			return e
		}
		return rt.WriteUint32(src, o, 0)
	}
	a, e := linearReadInt(rt, src, o)
	if e != nil {
		return e
	}
	if e = linearWriteInt(rt, dst, o, a); e != nil {
		return e
	}
	a.Move()
	if e = linearWriteInt(rt, src, o, a); e != nil {
		return e
	}
	v.movedComponents = 1
	if v.width == 24 {
		a, e = linearReadInt(rt, src, o+12)
		if e != nil {
			return e
		}
		if e = linearWriteInt(rt, dst, o+12, a); e != nil {
			return e
		}
		a.Move()
		if e = linearWriteInt(rt, src, o+12, a); e != nil {
			return e
		}
		v.movedComponents = 2
		return nil
	}
	return nil
}

// Both numeral and row types are nontrivially copyable in the source: allocate,
// publish new m_data, move each constructed field, free old, then write capacity.
func (v *linearNumeralVector) expand(rt *util.Runtime) error {
	if v.interrupted || v.pending != (util.Block{}) {
		return ErrPolynomialPending
	}
	cap, e := v.capacity(rt)
	if e != nil {
		return e
	}
	if polynomialSourceDebug && v.raw != (util.Block{}) && cap == 0 {
		return ErrPolynomialAssertion
	}
	next := uint32(2)
	if v.raw != (util.Block{}) {
		next = (3*cap + 1) >> 1
	}
	bytes, oldBytes := v.width*next+8, v.width*cap+8
	if v.raw != (util.Block{}) && (next <= cap || bytes <= oldBytes) {
		return util.Failure{Kind: util.FailureVectorOverflow}
	}
	v.pending, e = rt.Allocate(uint64(bytes))
	if e != nil {
		return e
	}
	if v.raw == (util.Block{}) {
		if e = rt.WriteUint32(v.pending, 0, next); e != nil {
			return e
		}
		if e = rt.WriteUint32(v.pending, 4, 0); e != nil {
			return e
		}
		v.raw = v.pending
		v.pending = util.Block{}
		return nil
	}
	sz, e := v.size(rt)
	if e != nil {
		return e
	}
	if e = rt.WriteUint32(v.pending, 4, sz); e != nil {
		return e
	}
	v.old = v.raw
	v.raw = v.pending
	v.pending = util.Block{}
	v.interrupted = true
	v.moved = 0
	v.movedComponents = 0
	v.oldInitialized = v.initialized
	v.initialized = 0
	for v.moved < sz {
		if e = v.moveField(rt, v.raw, v.old, v.moved); e != nil {
			return e
		}
		v.moved++
		v.initialized = v.moved
		v.movedComponents = 0
	}
	if e = rt.Deallocate(&v.old); e != nil {
		return e
	}
	v.oldInitialized = 0
	if e = rt.WriteUint32(v.raw, 0, next); e != nil {
		return e
	}
	v.interrupted = false
	return nil
}
func (v *linearNumeralVector) appendDefault(rt *util.Runtime, row uint32) error {
	sz, e := v.size(rt)
	if e != nil {
		return e
	}
	cap, e := v.capacity(rt)
	if e != nil {
		return e
	}
	if v.raw == (util.Block{}) || sz == cap {
		if e = v.expand(rt); e != nil {
			return e
		}
	}
	switch v.width {
	case 4:
		e = v.putRow(rt, sz, row)
	case 12:
		e = v.putInt(rt, sz, util.Int{})
	case 24:
		e = v.putRational(rt, sz, util.NewRational())
	default:
		return arena.ErrConfig
	}
	if e != nil {
		return e
	}
	v.initialized = sz + 1
	return rt.WriteUint32(v.raw, 4, sz+1)
}
func (v *linearNumeralVector) resetSize(rt *util.Runtime) error {
	if v.raw == (util.Block{}) {
		return nil
	}
	return rt.WriteUint32(v.raw, 4, 0)
}
func (s *linearStorage) rowIndex(i uint32) (uint32, error) {
	if polynomialSourceDebug {
		sz, e := s.A.size(s.runtime)
		if e != nil {
			return 0, e
		}
		if i >= sz {
			return 0, ErrPolynomialAssertion
		}
	}
	return s.A.rowAt(s.runtime, i)
}
func (s *linearStorage) row(i uint32) (*linearNumeralVector, error) {
	idx, e := s.rowIndex(i)
	if e != nil {
		return nil, e
	}
	if idx == 0 || uint64(idx) > uint64(len(s.rows)) {
		return nil, arena.ErrBounds
	}
	return &s.rows[idx-1], nil
}
func (s *linearStorage) rowMetadata(i uint32) error {
	if uint64(i) < uint64(len(s.rows)) {
		return nil
	}
	if uint64(i) > uint64(len(s.rows)) {
		return arena.ErrBounds
	}
	if len(s.rows) == cap(s.rows) {
		old := cap(s.rows)
		next := old + old/2 + 1
		stride := uint64(reflect.TypeOf(linearNumeralVector{}).Size())
		if uint64(next) > uint64(^uint(0)>>1)/stride {
			return arena.ErrBounds
		}
		if e := s.integer.Budget().Reserve(uint64(next) * stride); e != nil {
			return e
		}
		b := make([]linearNumeralVector, len(s.rows), next)
		copy(b, s.rows)
		s.rows = b
		s.integer.Budget().Release(uint64(old) * stride)
	}
	s.rows = append(s.rows, linearNumeralVector{width: s.b.width})
	return nil
}
func (s *linearStorage) appendRow(i uint32) error {
	if e := s.rowMetadata(i); e != nil {
		return e
	}
	r := &s.rows[i]
	if r.raw != (util.Block{}) || r.pending != (util.Block{}) || r.old != (util.Block{}) {
		return ErrPolynomialPending
	}
	*r = linearNumeralVector{width: s.b.width}
	return s.A.appendDefault(s.runtime, i+1)
}
func (s *linearStorage) swapRows(i, j uint32) error {
	a, e := s.rowIndex(i)
	if e != nil {
		return e
	}
	b, e := s.rowIndex(j)
	if e != nil {
		return e
	}
	if e = s.A.putRow(s.runtime, i, b); e != nil {
		return e
	}
	return s.A.putRow(s.runtime, j, a)
}
func (s *linearStorage) sourceBacking(v *linearNumeralVector) error {
	if v.raw != (util.Block{}) {
		if e := s.runtime.Deallocate(&v.raw); e != nil {
			return e
		}
	}
	v.initialized = 0
	return nil
}
func (s *linearStorage) freeTokens(v *linearNumeralVector) error {
	for _, b := range []*util.Block{&v.old, &v.pending, &v.raw} {
		if *b != (util.Block{}) {
			if e := s.runtime.Deallocate(b); e != nil {
				return e
			}
		}
	}
	*v = linearNumeralVector{width: v.width}
	return nil
}

// Owning abort reads only completed fields and the actual old unread suffix.
// It does not ask locator tables to reconstruct a transferred numeric root.
func (s *linearStorage) ownVectorNumerals(v *linearNumeralVector) error {
	if v.width != 12 && v.width != 24 {
		return nil
	}
	drop := func(raw util.Block, i uint32, component uint8) error {
		at := uint64(8) + uint64(i)*uint64(v.width) + uint64(component)*12
		a, e := linearReadInt(s.runtime, raw, at)
		if e != nil {
			return e
		}
		e = s.integer.Del(&a)
		return polynomialNewtonJoin(e, linearWriteInt(s.runtime, raw, at, a))
	}
	components := uint8(v.width / 12)
	for i := uint32(0); i < v.initialized; i++ {
		for c := uint8(0); c < components; c++ {
			if e := drop(v.raw, i, c); e != nil {
				return e
			}
		}
	}
	for c := uint8(0); c < v.movedComponents; c++ {
		if e := drop(v.raw, v.moved, c); e != nil {
			return e
		}
	}
	if v.old != (util.Block{}) {
		for i := v.moved; i < v.oldInitialized; i++ {
			for c := uint8(0); c < components; c++ {
				if e := drop(v.old, i, c); e != nil {
					return e
				}
			}
		}
	}
	return nil
}
