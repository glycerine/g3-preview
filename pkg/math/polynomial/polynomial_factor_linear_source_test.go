// Source: src/math/polynomial/polynomial.cpp:6591-6598;
// src/test/polynomial.cpp:1017-1080.
package polynomial

import (
	"errors"
	"testing"
)

func TestPolynomialLinearFactorLiteralPushAndMultiplicity(t *testing.T) {
	for k := uint32(1); k <= 4; k++ {
		m, p, _ := polynomialEvalFixture(t, []polynomialEvalTerm{{1, [3]uint32{1, 0, 0}}, {1, [3]uint32{0, 1, 0}}})
		f, e := NewPolynomialFactors(m)
		serviceMust(t, e)
		before, e := m.core.readWord(p, 0)
		serviceMust(t, e)
		serviceMust(t, m.factorLinearSquareFreePrimitive(p, f, 0, k))
		n, e := f.Distinct()
		serviceMust(t, e)
		value, e := f.factor(0)
		serviceMust(t, e)
		d, e := f.Degree(0)
		serviceMust(t, e)
		total, e := f.Total()
		serviceMust(t, e)
		after, e := m.core.readWord(p, 0)
		serviceMust(t, e)
		if n != 1 || value != p || d != k || total != k || after != before+1 {
			t.Fatal("literal linear push/INC", n, value, p, d, k, after, before)
		}
		polynomialFactorsAudit(t, f, []Polynomial{p}, nil)
		serviceMust(t, f.CloseChecked())
		after, e = m.core.readWord(p, 0)
		serviceMust(t, e)
		if after != before {
			t.Fatal("factor destructor did not consume exact INC", after, before)
		}
	}
}

func TestPolynomialLinearFactorSourceAssertionsAreCompiledOnly(t *testing.T) {
	polynomialThrowAssertionFixture(t)
	m, p, _ := polynomialEvalFixture(t, []polynomialEvalTerm{{1, [3]uint32{2, 0, 0}}, {1, [3]uint32{}}})
	f, e := NewPolynomialFactors(m)
	serviceMust(t, e)
	defer func() { serviceMust(t, f.CloseChecked()) }()
	budget, raw := m.core.integer.Budget().Used(), m.core.memory.State()
	e = m.factorLinearSquareFreePrimitive(p, f, 0, 1)
	if polynomialSourceDebug {
		if !errors.Is(e, ErrPolynomialAssertion) || m.core.integer.Budget().Used() != budget || m.core.memory.State() != raw {
			t.Fatal("source degree assertion frontier", e)
		}
		n, x := f.Distinct()
		serviceMust(t, x)
		if n != 0 {
			t.Fatal("source failed assertion published factor", n)
		}
	} else {
		serviceMust(t, e)
		v, x := f.factor(0)
		serviceMust(t, x)
		if v != p {
			t.Fatal("erased assertion route", v, p)
		}
	}
	polynomialFactorsAudit(t, f, []Polynomial{p}, nil)
}

func TestPolynomialLinearFactorEveryActualPushReservation(t *testing.T) {
	for ordinal := int64(0); ordinal < 64; ordinal++ {
		m, p, _ := polynomialEvalFixture(t, []polynomialEvalTerm{{1, [3]uint32{1, 0, 0}}, {-1, [3]uint32{}}})
		f, e := NewPolynomialFactors(m)
		serviceMust(t, e)
		callerRC, x := m.core.readWord(p, 0)
		serviceMust(t, x)
		m.core.integer.Budget().FailAfter(ordinal)
		e = m.factorLinearSquareFreePrimitive(p, f, 0, 3)
		polynomialFactorsAudit(t, f, []Polynomial{p}, nil)
		if e != nil && !numericPhysicalReservationRefusal(e) {
			t.Fatal("actual linear prefix", ordinal, e)
		}
		// No source local owns p: even an unjoined pointer publication is borrowed
		// from this real caller. Explicit factors cleanup can retire only actual INC.
		m.core.integer.Budget().FailAfter(-1)
		serviceMust(t, m.CleanupGCD())
		serviceMust(t, f.Cleanup())
		polynomialFactorsAudit(t, f, []Polynomial{p}, nil)
		serviceMust(t, f.CloseChecked())
		afterRC, x := m.core.readWord(p, 0)
		serviceMust(t, x)
		if afterRC != callerRC {
			t.Fatal("actual Push/cleanup changed caller INC", ordinal, afterRC, callerRC)
		}
		if e == nil {
			t.Log("actual linear success", ordinal)
			return
		}
	}
	t.Fatal("bounded linear Push campaign did not complete")
}
