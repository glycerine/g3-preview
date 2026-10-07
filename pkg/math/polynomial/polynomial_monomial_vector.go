// Source: src/math/polynomial/polynomial.cpp; src/math/polynomial/polynomial.h. Licensed under the MIT license.
package polynomial

import (
	"github.com/glycerine/g3/pkg/util"
)

func (m *MonomialManager) powerVectorCleanup() error {
	if m.powers.pending != (util.Block{}) {
		if e := m.memory.Deallocate(&m.powers.pending); e != nil {
			return e
		}
	}
	if m.powers.raw != (util.Block{}) {
		return m.memory.Deallocate(&m.powers.raw)
	}
	return nil
}
func (m *MonomialManager) powerVectorPush(p MonomialPower) error {
	v := &m.powers
	var n, cap uint32
	var e error
	if v.raw != (util.Block{}) {
		cap, e = m.memory.ReadUint32(v.raw, 0)
		if e != nil {
			return e
		}
		n, e = m.memory.ReadUint32(v.raw, 4)
		if e != nil {
			return e
		}
	}
	if n >= cap {
		nc := uint32(2)
		if cap > 0 {
			nc = (3*cap + 1) >> 1
		}
		nb := 8 + nc*8
		ob := 8 + cap*8
		if cap > 0 && (nc <= cap || nb <= ob) {
			return util.Failure{Kind: util.FailureVectorOverflow}
		}
		if v.raw == (util.Block{}) {
			v.pending, e = m.memory.Allocate(uint64(nb))
			if e != nil {
				return e
			}
			if e = m.memory.WriteUint32(v.pending, 0, nc); e != nil {
				return e
			}
			if e = m.memory.WriteUint32(v.pending, 4, 0); e != nil {
				return e
			}
			v.raw = v.pending
			v.pending = util.Block{}
		} else {
			if e = m.memory.Reallocate(&v.raw, uint64(nb)); e != nil {
				return e
			}
			if e = m.memory.WriteUint32(v.raw, 0, nc); e != nil {
				return e
			}
		}
	}
	if e = m.memory.WriteUint32(v.raw, 8+uint64(n)*8, p.Var); e != nil {
		return e
	}
	if e = m.memory.WriteUint32(v.raw, 12+uint64(n)*8, p.Degree); e != nil {
		return e
	}
	return m.memory.WriteUint32(v.raw, 4, n+1)
}

// Heap sorting is the explicit concrete external std::sort backend adaptation.
func monomialSortVariables(xs []uint32) {
	var sift func(int, int)
	sift = func(root, end int) {
		for {
			child := root*2 + 1
			if child >= end {
				return
			}
			if child+1 < end && xs[child] < xs[child+1] {
				child++
			}
			if xs[root] >= xs[child] {
				return
			}
			xs[root], xs[child] = xs[child], xs[root]
			root = child
		}
	}
	for i := len(xs) / 2; i > 0; {
		i--
		sift(i, len(xs))
	}
	for end := len(xs) - 1; end > 0; end-- {
		xs[0], xs[end] = xs[end], xs[0]
		sift(0, end)
	}
}
func (m *MonomialManager) MkFromVariables(xs []uint32) (Monomial, error) {
	if e := m.check(); e != nil {
		return Monomial{}, e
	}
	if len(xs) == 0 {
		return m.unit, nil
	}
	if len(xs) == 1 {
		return m.MkVariable(xs[0], 1)
	}
	if m.powers.raw != (util.Block{}) {
		if e := m.memory.WriteUint32(m.powers.raw, 4, 0); e != nil {
			return Monomial{}, e
		}
	}
	monomialSortVariables(xs)
	if polynomialSourceDebug && xs[0] >= m.nextVar {
		return Monomial{}, ErrMonomialAssertion
	}
	if e := m.powerVectorPush(MonomialPower{xs[0], 1}); e != nil {
		return Monomial{}, e
	}
	for _, x := range xs[1:] {
		if polynomialSourceDebug && x >= m.nextVar {
			return Monomial{}, ErrMonomialAssertion
		}
		n, e := m.memory.ReadUint32(m.powers.raw, 4)
		if e != nil {
			return Monomial{}, e
		}
		last, e := m.memory.ReadUint32(m.powers.raw, 8+uint64(n-1)*8)
		if e != nil {
			return Monomial{}, e
		}
		if x == last {
			d, e := m.memory.ReadUint32(m.powers.raw, 12+uint64(n-1)*8)
			if e != nil {
				return Monomial{}, e
			}
			if e = m.memory.WriteUint32(m.powers.raw, 12+uint64(n-1)*8, d+1); e != nil {
				return Monomial{}, e
			}
		} else {
			if e = m.powerVectorPush(MonomialPower{x, 1}); e != nil {
				return Monomial{}, e
			}
		}
	}
	n, e := m.memory.ReadUint32(m.powers.raw, 4)
	if e != nil {
		return Monomial{}, e
	}
	t := &m.temps[0]
	if n > t.capacity {
		if e = m.tempIncrease(t, n*2); e != nil {
			return Monomial{}, e
		}
	}
	if polynomialSourceDebug && n >= t.capacity {
		return Monomial{}, ErrMonomialAssertion
	}
	if e = m.tempSetSize(t, n); e != nil {
		return Monomial{}, e
	}
	for i := uint32(0); i < n; i++ {
		x, e := m.memory.ReadUint32(m.powers.raw, 8+uint64(i)*8)
		if e != nil {
			return Monomial{}, e
		}
		d, e := m.memory.ReadUint32(m.powers.raw, 12+uint64(i)*8)
		if e != nil {
			return Monomial{}, e
		}
		if e = m.tempSetPower(t, i, MonomialPower{x, d}); e != nil {
			return Monomial{}, e
		}
	}
	return m.internTemp(0)
}
func (m *MonomialManager) powerVectorSourceDestroy() error {
	if m.powers.raw != (util.Block{}) {
		return m.memory.Deallocate(&m.powers.raw)
	}
	return nil
}
