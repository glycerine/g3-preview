// Source: src/util/vector.h:82-100; src/util/mpz.h:97-99. Licensed under the MIT license.
package polynomial

import (
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// retired is an explicitly charged owning Go continuation for actual initialized
// numeral fields discarded by source trivial reset/placement overwrite or a
// nontrivial growth's omitted tail. It creates no source Runtime allocation.
// Captures from old bytes are tentative nonowning scalar projections until the
// old physical block is consumed; owning cleanup before consumption drains the
// real old fields instead. No registry is used to discover numeral ownership.
func (v *polynomialVector) retainNumeral(m *MultivariatePolynomialCore, a util.Int) error {
	if a.NativeCellToken() == 0 {
		return nil
	}
	n := len(v.retired) + 1
	if n > cap(v.retired) {
		old := cap(v.retired)
		c := old + old/2 + 1
		stride := uint64(reflect.TypeOf(util.Int{}).Size())
		if uint64(c) > uint64(^uint(0)>>1)/stride {
			return arena.ErrBounds
		}
		if e := m.integer.Budget().Reserve(uint64(c) * stride); e != nil {
			return e
		}
		b := make([]util.Int, n, c)
		copy(b, v.retired)
		v.retired = b
		m.integer.Budget().Release(uint64(old) * stride)
	} else {
		v.retired = v.retired[:n]
	}
	v.retired[n-1] = a
	return nil
}
func (v *polynomialVector) expandNumerals(m *MultivariatePolynomialCore, n, bytes uint32) error {
	var e error
	v.pending, e = m.memory.Allocate(uint64(bytes))
	if e != nil {
		return e
	}
	oldSize, e := v.size(m)
	if e != nil {
		return e
	}
	if e = m.memory.WriteUint32(v.pending, 4, oldSize); e != nil {
		return e
	}
	v.old = v.raw
	v.oldSize = oldSize
	v.oldInitialized = v.initialized
	v.moved = 0
	v.oldCleanupIndex = 0
	v.raw = v.pending
	v.pending = util.Block{}
	v.initialized = 0
	v.interrupted = true
	v.retirementStart = uint32(len(v.retired))
	oldView := polynomialVector{raw: v.old, width: 12}
	for v.moved < oldSize {
		i := v.moved
		a, e := oldView.coefficient(m, i)
		if e != nil {
			return e
		}
		if e = v.putCoefficient(m, i, util.IntFromNativeWords([3]uint32{uint32(a.NativeScalar()), uint32(0), uint32(uint64(0) >> 32)})); e != nil {
			return e
		}
		if e = v.putCoefficient(m, i, a); e != nil {
			return e
		}
		a.Move()
		if e = oldView.putCoefficient(m, i, a); e != nil {
			return e
		}
		v.moved++
		v.initialized = v.moved
	}
	for i := oldSize; i < v.oldInitialized; i++ {
		a, e := oldView.coefficient(m, i)
		if e != nil {
			return e
		}
		if e = v.retainNumeral(m, a); e != nil {
			return e
		}
	}
	e = m.memory.Deallocate(&v.old)
	if v.old == (util.Block{}) {
		v.retiredActive = uint32(len(v.retired))
		v.oldInitialized = 0
	}
	if e != nil {
		return e
	}
	if e = m.memory.WriteUint32(v.raw, 0, n); e != nil {
		return e
	}
	v.interrupted = false
	return nil
}
func (v *polynomialVector) cleanupNumerals(m *MultivariatePolynomialCore) error {
	if v.old != (util.Block{}) {
		oldView := polynomialVector{raw: v.old, width: 12}
		if v.oldCleanupIndex < v.moved {
			v.oldCleanupIndex = v.moved
		}
		for v.oldCleanupIndex < v.oldInitialized {
			i := v.oldCleanupIndex
			a, e := oldView.coefficient(m, i)
			if e != nil {
				return e
			}
			e = m.deleteNumeral(&a)
			if w := oldView.putCoefficient(m, i, a); w != nil {
				return w
			}
			if e != nil {
				return e
			}
			v.oldCleanupIndex++
		}
		e := m.memory.Deallocate(&v.old)
		if v.old == (util.Block{}) {
			v.retired = v.retired[:v.retirementStart]
			v.oldInitialized = 0
		}
		if e != nil {
			return e
		}
	}
	for i := uint32(0); i < v.retiredActive; i++ {
		if e := m.deleteNumeral(&v.retired[i]); e != nil {
			return e
		}
	}
	m.integer.Budget().Release(uint64(cap(v.retired)) * uint64(reflect.TypeOf(util.Int{}).Size()))
	v.retired = nil
	v.retiredActive = 0
	v.retirementStart = 0
	// Logical buffer users delete their actual fields first. Initialized retained
	// tails remain actual roots even after source reset, and are drained here.
	if v.raw != (util.Block{}) {
		for i := uint32(0); i < v.initialized; i++ {
			a, e := v.coefficient(m, i)
			if e != nil {
				return e
			}
			e = m.deleteNumeral(&a)
			if w := v.putCoefficient(m, i, a); w != nil {
				return w
			}
			if e != nil {
				return e
			}
		}
	}
	return nil
}
