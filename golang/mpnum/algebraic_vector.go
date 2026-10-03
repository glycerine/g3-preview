// Copyright (c) 2011 Microsoft Corporation. Licensed under the MIT license.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"math"
	"reflect"
)

// Owning scoped svector<anum>. Get returns a borrowed scalar snapshot; it must
// never be deleted or retained beyond its owning element's lifetime.
type AlgebraicVector struct {
	self            *AlgebraicVector
	core            *AlgebraicRootCore
	values          []Algebraic
	size            uint32
	header          uint64
	closed, closing bool
}

func (m *AlgebraicRootCore) NewVector() (*AlgebraicVector, error) {
	if e := m.check(); e != nil {
		return nil, e
	}
	if m.vectors == math.MaxUint32 {
		return nil, arena.ErrSlotLimit
	}
	h := uint64(reflect.TypeOf(AlgebraicVector{}).Size())
	if e := m.config.Rationals.budget.Reserve(h); e != nil {
		return nil, e
	}
	v := &AlgebraicVector{core: m, header: h}
	v.self = v
	m.vectors++
	return v, nil
}
func (v *AlgebraicVector) check() error {
	if v == nil || v.self != v {
		return ErrAlgebraicOwner
	}
	if v.closed || v.closing {
		return arena.ErrClosed
	}
	return v.core.check()
}
func (v *AlgebraicVector) Len() uint32 {
	if v == nil || v.self != v || v.closed {
		return 0
	}
	return v.size
}
func (v *AlgebraicVector) Get(i uint32) (Algebraic, error) {
	if e := v.check(); e != nil {
		return Algebraic{}, e
	}
	if i >= v.size {
		return Algebraic{}, arena.ErrBounds
	}
	a := v.values[i]
	_, e := v.core.location(&a)
	return a, e
}
func (v *AlgebraicVector) pushTake(a *Algebraic) error {
	if e := v.check(); e != nil {
		return e
	}
	if v.size == math.MaxUint32 {
		return arena.ErrBounds
	}
	if v.size == uint32(cap(v.values)) {
		old := uint32(cap(v.values))
		n := uint32(2)
		if old != 0 {
			n = (3*old + 1) >> 1
		}
		stride := uint64(reflect.TypeOf(Algebraic{}).Size())
		if n <= old || uint64(n) > uint64(int(^uint(0)>>1))/stride {
			return arena.ErrBounds
		}
		b := v.core.config.Rationals.budget
		if e := b.Reserve(uint64(n) * stride); e != nil {
			return e
		}
		next := make([]Algebraic, n)
		copy(next, v.values)
		b.Release(uint64(old) * stride)
		v.values = next
	}
	v.values[v.size] = *a
	v.size++
	*a = Algebraic{}
	return nil
}
func (v *AlgebraicVector) PushCopy(a *Algebraic) error {
	if e := v.check(); e != nil {
		return e
	}
	if v.core.pending.slot != 0 {
		return ErrAlgebraicPending
	}
	var next Algebraic
	if e := v.core.Copy(&next, a); e != nil {
		if next.slot != 0 {
			v.core.pending = next
			v.core.locations[next.slot-1].published = false
			v.core.callers--
		}
		return e
	}
	if e := v.pushTake(&next); e != nil {
		v.core.pending = next
		if next.slot != 0 {
			v.core.locations[next.slot-1].published = false
			v.core.callers--
		}
		return e
	}
	return nil
}
func (v *AlgebraicVector) Reset() error {
	if e := v.check(); e != nil {
		return e
	}
	for i := uint32(0); i < v.size; i++ {
		if e := v.core.Del(&v.values[i]); e != nil {
			return e
		}
	}
	v.size = 0
	return nil
}
func (v *AlgebraicVector) Close() error {
	if v == nil || v.self != v {
		return ErrAlgebraicOwner
	}
	if v.closed {
		return nil
	}
	v.closing = true
	for i := uint32(0); i < v.size; i++ {
		var e error
		if v.core.closing {
			e = v.core.delClosing(&v.values[i])
		} else {
			e = v.core.Del(&v.values[i])
		}
		if e != nil {
			return e
		}
	}
	b := v.core.config.Rationals.budget
	b.Release(uint64(cap(v.values)) * uint64(reflect.TypeOf(Algebraic{}).Size()))
	b.Release(v.header)
	v.values = nil
	v.header = 0
	v.size = 0
	v.core.vectors--
	v.closed = true
	return nil
}

// These operations use the actual owning elements: source predicates and
// comparisons may replace a root cell with a basic rational cell.
func (v *AlgebraicVector) Compare(i, j uint32) (int, error) {
	if e := v.check(); e != nil {
		return 0, e
	}
	if i >= v.size || j >= v.size {
		return 0, arena.ErrBounds
	}
	return v.core.Compare(&v.values[i], &v.values[j])
}
func (v *AlgebraicVector) IsInt(i uint32) (bool, error) {
	if e := v.check(); e != nil {
		return false, e
	}
	if i >= v.size {
		return false, arena.ErrBounds
	}
	return v.core.IsInt(&v.values[i])
}
func (v *AlgebraicVector) IsRational(i uint32) (bool, error) {
	if e := v.check(); e != nil {
		return false, e
	}
	if i >= v.size {
		return false, arena.ErrBounds
	}
	return v.core.IsRational(&v.values[i])
}
