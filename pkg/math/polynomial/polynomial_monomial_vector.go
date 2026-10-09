// Source: src/math/polynomial/polynomial.cpp; src/math/polynomial/polynomial.h.
package polynomial

import "sort"

func (m *MonomialManager) powerVectorCleanup() error       { m.powers = nil; return nil }
func (m *MonomialManager) powerVectorSourceDestroy() error { m.powers = nil; return nil }
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
	m.powers = m.powers[:0]
	sort.Slice(xs, func(i, j int) bool { return xs[i] < xs[j] })
	for _, x := range xs {
		if polynomialSourceDebug && x >= m.nextVar {
			return Monomial{}, ErrMonomialAssertion
		}
		n := len(m.powers)
		if n != 0 && m.powers[n-1].Var == x {
			m.powers[n-1].Degree++
		} else {
			m.powers = append(m.powers, MonomialPower{x, 1})
		}
	}
	if e := m.tempInit(&m.temps[0], m.powers); e != nil {
		return Monomial{}, e
	}
	return m.internTemp(0)
}
