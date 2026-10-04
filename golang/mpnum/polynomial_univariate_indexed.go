// Source: src/math/polynomial/polynomial.cpp:2582-2594.
// Licensed under the MIT license. Author: Leonardo de Moura.
package mpnum

import "github.com/glycerine/g3/golang/arena"

// Concrete borrowed source numeral array routes. None acquires a second
// owning backing or preflights an unread later coefficient.
type polynomialUnivariateCoefficients struct {
	values  []Int
	vector  *polynomialVector
	array   integerArrayView
	indexed bool
}

func (as polynomialUnivariateCoefficients) reference(m *MultivariatePolynomialCore, k uint32, local *Int) (*Int, error) {
	if as.vector != nil {
		v, e := as.vector.coefficient(m, k)
		*local = v
		return local, e
	}
	if as.indexed {
		return as.array.reference(k, local)
	}
	if uint64(k) >= uint64(len(as.values)) {
		return nil, arena.ErrBounds
	}
	return &as.values[k], nil
}
func (as polynomialUnivariateCoefficients) store(m *MultivariatePolynomialCore, k uint32, a *Int) error {
	if as.vector != nil {
		return as.vector.putCoefficient(m, k, *a)
	}
	if as.indexed {
		return as.array.store(k, *a)
	}
	return nil // actual slice field already received the source mutation
}
func (m *MultivariatePolynomialCore) mkUnivariateSource(v, n uint32, as polynomialUnivariateCoefficients) (Polynomial, error) {
	size, e := m.cheap.ms.size(m)
	if e != nil {
		return Polynomial{}, e
	}
	if modularSourceDebug && size != 0 {
		return Polynomial{}, ErrPolynomialAssertion
	}
	k := n + 1
	for k > 0 {
		k--
		var local Int
		a, e := as.reference(m, k, &local)
		if e != nil {
			return Polynomial{}, e
		}
		zero, e := m.modular.IsZero(a)
		if e != nil {
			return Polynomial{}, e
		}
		if zero {
			e = m.deleteNumeral(a)
			if w := as.store(m, k, a); w != nil {
				return Polynomial{}, polynomialNewtonJoin(e, w)
			}
			if e != nil {
				return Polynomial{}, e
			}
			continue
		}
		term, e := m.generatedVariable(v, k)
		if e != nil {
			return Polynomial{}, e
		}
		e = m.cheap.add(a, term, true)
		if w := as.store(m, k, a); w != nil {
			return Polynomial{}, polynomialNewtonJoin(e, w)
		}
		if e != nil {
			return Polynomial{}, e
		}
	}
	return m.cheap.mk(false)
}
