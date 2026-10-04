// Source: src/math/polynomial/polynomial.cpp:1821-1850,2346-2370;
// src/util/vector.h. Licensed under the MIT license.
// Source author: Leonardo de Moura (leonardo), 2011-11-15.
package mpnum

import (
	"github.com/glycerine/g3/golang/memory"
	"reflect"
)

type polynomialImpRetirement struct {
	fields          []Int
	captured, drain uint32
	consumed        bool
}

func (m *MultivariatePolynomialManager) sourceVectorIndex(v *polynomialVector) int {
	for i, p := range [6]*polynomialVector{&m.core.cheap2.as, &m.core.cheap.as, &m.core.som2.as, &m.core.som.as, &m.core.linearTerms, &m.core.rationalTerms} {
		if v == p {
			return i
		}
	}
	return -1
}
func (m *MultivariatePolynomialManager) retireSourceNumeralVector(v *polynomialVector) error {
	i := m.sourceVectorIndex(v)
	if i < 0 {
		return ErrPolynomialOwner
	}
	r := &m.retirements[i]
	if r.consumed {
		return nil
	}
	for r.captured < v.initialized {
		a, e := v.coefficient(m.core, r.captured)
		if e != nil {
			return e
		}
		if len(r.fields) == cap(r.fields) {
			old := cap(r.fields)
			c := old + old/2 + 1
			stride := uint64(reflect.TypeOf(Int{}).Size())
			if e = m.core.integer.budget.Reserve(uint64(c) * stride); e != nil {
				return e
			}
			next := make([]Int, len(r.fields), c)
			copy(next, r.fields)
			r.fields = next
			m.core.integer.budget.Release(uint64(old) * stride)
		}
		r.fields = append(r.fields, a)
		r.captured++
	}
	if v.raw != (memory.Block{}) {
		e := m.core.memory.Deallocate(&v.raw)
		if v.raw == (memory.Block{}) {
			r.consumed = true
		}
		return e
	}
	r.consumed = true
	return nil
}
func (m *MultivariatePolynomialManager) cleanupRetirement(i int) error {
	r := &m.retirements[i]
	if r.consumed {
		for r.drain < uint32(len(r.fields)) {
			if e := m.core.integer.Del(&r.fields[r.drain]); e != nil {
				return e
			}
			r.drain++
		}
		v := [6]*polynomialVector{&m.core.cheap2.as, &m.core.cheap.as, &m.core.som2.as, &m.core.som.as, &m.core.linearTerms, &m.core.rationalTerms}[i]
		v.initialized = 0
	}
	m.core.integer.budget.Release(uint64(cap(r.fields)) * uint64(reflect.TypeOf(Int{}).Size()))
	r.fields = nil
	return nil
}

// Returns actual named owning fields; a directory is never swept to recover
// caller values. Repeat traversal is cold and allocation-free.
func (m *MultivariatePolynomialManager) privatePolynomial(index uint32) (Polynomial, bool, error) {
	c := m.core
	roots := [2]Polynomial{c.zero, c.unit}
	for i, p := range roots {
		if c.sourceStage > uint8(i) {
			continue
		}
		if index == 0 {
			return p, true, nil
		}
		index--
	}
	for _, n := range m.newton.children {
		if n == nil || n.closed {
			continue
		}
		fixed := [7]Polynomial{n.scope.temp.value, n.scope.temp.incoming, n.scope.auxPoly.value, n.scope.auxPoly.incoming, n.scope.u.value, n.scope.u.incoming, n.vs.incoming}
		for _, p := range fixed {
			if index == 0 {
				return p, true, nil
			}
			index--
		}
		size, e := n.vs.values.size(c)
		if e != nil {
			return Polynomial{}, false, e
		}
		for j := n.vs.drain; j < size; j++ {
			if index == 0 {
				p, e := n.vs.get(c, j)
				return p, true, e
			}
			index--
		}
	}
	return Polynomial{}, false, nil
}
func (m *MultivariatePolynomialManager) callerGuard() error {
	if m.originalCause != nil {
		return nil
	} // an unreturned failed constructor has no callers.
	distinct := uint32(0)
	for i := uint32(0); ; i++ {
		p, ok, e := m.privatePolynomial(i)
		if e != nil {
			return e
		}
		if !ok {
			break
		}
		if p == (Polynomial{}) {
			continue
		}
		earlier := false
		count := uint32(0)
		for j := uint32(0); ; j++ {
			q, ok, e := m.privatePolynomial(j)
			if e != nil {
				return e
			}
			if !ok {
				break
			}
			if q == p {
				count++
				if j < i {
					earlier = true
				}
			}
		}
		if earlier {
			continue
		}
		l, e := m.core.polynomialLocation(p)
		if e != nil {
			return e
		}
		if l.deleteStage == 0 {
			rc, e := m.core.readWord(p, 0)
			if e != nil {
				return e
			}
			if rc > count {
				return ErrPolynomialLive
			}
		}
		distinct++
	}
	if m.core.live > distinct {
		return ErrPolynomialLive
	}
	return nil
}
func (m *MultivariatePolynomialManager) CloseChecked() error {
	if m != nil && m.self == m && (m.conversion.active || m.conversionHasOrphans()) {
		return ErrPolynomialPending
	}
	if m != nil && m.self == m && m.toPolynomial.active {
		return ErrPolynomialPending
	}
	if m != nil && m.self == m && (m.composition.active || m.compositionHasOrphans()) {
		return ErrPolynomialPending
	}

	if m != nil && m.self == m && m.core != nil && m.core.contentLocal.active {
		return ErrPolynomialPending
	}
	if m != nil && m.self == m && m.monic.active {
		return ErrPolynomialPending
	}
	if m != nil && m.self == m && m.evaluation.active {
		return ErrPolynomialPending
	}
	if m != nil && m.self == m && (m.derivative.active || m.derivativeHasOrphans()) {
		return ErrPolynomialPending
	}
	if e := m.identity(); e != nil {
		return e
	}
	if m.squareRoot.active {
		return ErrPolynomialPending
	}
	if m.core != nil && (m.core.translation.active || m.core.pscPending()) {
		return ErrPolynomialPending
	}
	if cap(m.derivative.orphans) > 0 {
		if e := m.CleanupDerivative(); e != nil {
			return e
		}
	}
	if !m.closing {
		if m.core != nil {
			if e := m.callerGuard(); e != nil {
				return e
			}
		}
		m.closing = true
	}
	if cap(m.conversion.orphans) > 0 {
		if e := m.CleanupConversion(); e != nil {
			return e
		}
	}
	if e := m.closeEvaluationFrames(); e != nil {
		return e
	}
	if e := m.closeSquareRootMetadata(); e != nil {
		return e
	}
	if e := m.closeCompositionMetadata(); e != nil {
		return e
	}
	if m.core == nil {
		m.originalBudget.Release(m.header)
		m.header = 0
		m.closed = true
		return nil
	}
	for m.cleanupStage < 12 {
		var e error
		switch m.cleanupStage {
		case 0:
			if m.constructed >= 5 {
				e = m.core.translateBuffers.CloseChecked()
			}
		case 1:
			if m.constructed >= 5 {
				e = m.core.substitution.positions.CloseChecked()
			}
		case 2:
			if m.constructed >= 5 {
				e = m.core.substitution.max.CloseChecked()
			}
		case 3:
			if m.constructed >= 5 {
				e = m.skeletons.CloseChecked()
			}
		case 4:
			if m.constructed >= 5 {
				e = m.newton.Close()
			}
		case 5:
			if m.constructed >= 5 {
				e = m.iccp.CloseChecked()
			}
		case 6:
			e = m.varDegrees.close(m.core)
		case 7:
			if m.constructed >= 5 {
				e = m.core.substitution.found.close()
			}
		case 8:
			e = m.lexPermutation.close(m.core)
		case 9:
			e = m.lexBuckets.close()
		case 10:
			e = m.degree2pos.close(m.core)
		case 11:
			if m.core.tmp1.raw != (memory.Block{}) {
				e = m.core.memory.Deallocate(&m.core.tmp1.raw)
			}
			if e == nil && m.core.tmp1.pending != (memory.Block{}) {
				e = m.core.memory.Deallocate(&m.core.tmp1.pending)
			}
		}
		if e != nil {
			m.cleanupCause = e
			return e
		}
		m.cleanupStage++
	}
	for i := range m.retirements {
		if e := m.cleanupRetirement(i); e != nil {
			m.cleanupCause = e
			return e
		}
	}
	if m.upm != nil && !m.upm.closed {
		if e := m.upm.Close(); e != nil {
			m.cleanupCause = e
			return e
		}
	}
	// A real body DEC may already have entered/stopped the MM destructor. Only
	// explicit owning continuation acknowledges it; source history remains sticky.
	if (m.mmDropDone || m.mmConstructed) && m.core.monomials != nil && m.core.monomials.managerRefs == 0 && !m.core.monomials.closed {
		if e := m.core.monomials.CloseChecked(); e != nil {
			m.cleanupCause = e
			return e
		}
	}
	if !m.core.closed {
		if e := m.core.CloseChecked(); e != nil {
			m.cleanupCause = e
			return e
		}
	}
	m.originalBudget.Release(m.header)
	m.header = 0
	m.closed = true
	m.cleanupCause = nil
	return nil
}
func (m *MultivariatePolynomialManager) Close() {
	if e := m.CloseChecked(); e != nil {
		panic(e)
	}
}
