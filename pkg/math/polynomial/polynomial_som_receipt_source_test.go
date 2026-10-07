// Source: src/math/polynomial/polynomial.cpp:2020-2080,2159-2214.
// Licensed under the MIT license. Author: Leonardo de Moura.
package polynomial

import (
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialSOMStoppedCalleeReplacementIdentity(t *testing.T) {
	m := polynomialCellFixture(t)
	v, e := NewPolynomialSOMVector(m)
	serviceMust(t, e)
	serviceMust(t, v.SetOwner(m))
	b, e := v.At(0)
	serviceMust(t, e)
	x, e := m.monomials.MkVariable(0, 1)
	serviceMust(t, e)
	p, e := m.MkTerms([]util.Int{util.NewInt(3)}, []Monomial{x})
	serviceMust(t, e)
	polynomialCaller(t, m, p)
	a := util.NewInt(2)
	m.integer.Budget().FailAfter(0)
	e = b.AddMul(&a, x, p)
	if !numericPhysicalReservationRefusal(e) || !b.calleeCall {
		t.Fatal("actual failed product factory", e, b.calleeCall)
	}
	polynomialSOMPhysicalAudit(t, m, v, []Monomial{x}, &a)
	m.integer.Budget().FailAfter(-1)
	serviceMust(t, m.monomials.CleanupCreation())
	m.integer.Budget().FailAfter(0)
	_, e = m.monomials.MkMonomial([]MonomialPower{{0, 31}, {1, 37}, {2, 41}})
	if !numericPhysicalReservationRefusal(e) {
		t.Fatal("replacement real factory", e)
	}
	before := m.monomials.creation
	polynomialSOMPhysicalAudit(t, m, v, []Monomial{x}, &a)
	m.integer.Budget().FailAfter(-1)
	serviceMust(t, v.CloseChecked())
	if m.monomials.creation != before {
		t.Fatal("stale buffer receipt swept foreign replacement", before, m.monomials.creation)
	}
	serviceMust(t, m.monomials.CleanupCreation())
}

func TestPolynomialSOMStoppedGeneratedResultReplacement(t *testing.T) {
	reached := false
	for ordinal := int64(0); ordinal < 32 && !reached; ordinal++ {
		m := polynomialCellFixture(t)
		v, e := NewPolynomialSOMVector(m)
		serviceMust(t, e)
		serviceMust(t, v.SetOwner(m))
		b, e := v.At(0)
		serviceMust(t, e)
		x, e := m.monomials.MkVariable(0, 1)
		serviceMust(t, e)
		p, e := m.MkTerms([]util.Int{util.NewInt(3)}, []Monomial{x})
		serviceMust(t, e)
		polynomialCaller(t, m, p)
		a := util.NewInt(2)
		m.integer.Budget().FailAfter(ordinal)
		e = b.AddMul(&a, x, p)
		polynomialSOMPhysicalAudit(t, m, v, []Monomial{x}, &a)
		reached = b.generatedCall
		m.integer.Budget().FailAfter(-1)
		if !reached {
			serviceMust(t, v.CloseChecked())
			continue
		}
		if !numericPhysicalReservationRefusal(e) {
			t.Fatal("actual product publication prefix", e)
		}
		old := m.generatedMonomial
		serviceMust(t, m.cleanupGeneratedMonomial())
		replacement, e := m.generatedProduct(x, x)
		serviceMust(t, e)
		if replacement == old || !m.generatedActive {
			t.Fatal("distinct actual replacement", old, replacement)
		}
		polynomialSOMPhysicalAudit(t, m, v, []Monomial{x}, &a)
		serviceMust(t, v.CloseChecked())
		if !m.generatedActive || m.generatedMonomial != replacement {
			t.Fatal("stale buffer receipt swept replacement generated result", replacement, m.generatedMonomial)
		}
		serviceMust(t, m.cleanupGeneratedMonomial())
		t.Log("actual generated prefix", ordinal)
	}
	if !reached {
		t.Fatal("no actual generated prefix")
	}
}
