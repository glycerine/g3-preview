// Source: src/math/polynomial/polynomial.cpp; src/math/polynomial/polynomial.h.
package polynomial

import (
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

// These are independent bounded constructor/arithmetic oracles for the actual
// selected source owner. The registered polynomial driver reaches PSC, whose
// construction and subresultant dependencies are not established by these tests.
func polynomialCellFixture(t *testing.T) *MultivariatePolynomialCore {
	t.Helper()
	return polynomialCellFixtureWithProfile(t, util.SmallAllocatorProfile{})
}

// Explicit direct object allocation keeps real factory-failure coverage after
// native locator growth ceases to reserve artificial container storage.
func polynomialCellDirectObjectFixture(t *testing.T) *MultivariatePolynomialCore {
	t.Helper()
	return polynomialCellFixtureWithProfile(t, util.SmallAllocatorProfile{PointerBits: 64, Debug: true})
}

func polynomialCellFixtureWithProfile(t *testing.T, profile util.SmallAllocatorProfile) *MultivariatePolynomialCore {
	t.Helper()
	q, b := rationalManager(t, false)
	z := q.Integers()
	mm, e := NewMonomialManager(MonomialManagerConfig{Budget: b, Memory: z.RuntimeOwner(), Profile: profile})
	serviceMust(t, e)
	serviceMust(t, mm.IncManagerRef())
	var visits uint32
	m, e := NewMultivariatePolynomialCore(MultivariatePolynomialConfig{
		Integer: z, Rationals: q, Monomials: mm,
		Checkpoint:  func() bool { visits++; return visits < 100000 },
		Cooperation: func() error { return nil },
	})
	serviceMust(t, e)
	t.Cleanup(func() {
		if !m.closed {
			serviceMust(t, m.CloseChecked())
		}
		serviceMust(t, mm.DecManagerRef())
	})
	for i := 0; i < 3; i++ {
		_, e = mm.MkVar()
		serviceMust(t, e)
	}
	return m
}

func polynomialCaller(t *testing.T, m *MultivariatePolynomialCore, p Polynomial) Polynomial {
	t.Helper()
	serviceMust(t, m.IncRef(p))
	t.Cleanup(func() { serviceMust(t, m.DecRef(p)) })
	return p
}

func TestPolynomialCellSourceTakeAndFirstMaximum(t *testing.T) {
	m := polynomialCellFixture(t)
	ms := [3]Monomial{}
	for i, powers := range [3][]MonomialPower{{{0, 7}}, {{2, 1}}, {{2, 3}}} {
		var e error
		ms[i], e = m.monomials.MkMonomial(powers)
		serviceMust(t, e)
		serviceMust(t, m.monomials.IncRef(ms[i]))
	}
	as := [3]util.Int{util.NewInt(11), util.NewInt(-5), util.NewInt(9)}
	p, e := m.MkCoreTake(as[:], ms[:])
	serviceMust(t, e)
	polynomialCaller(t, m, p)
	for i := range as {
		if as[i] != (util.Int{}) {
			t.Fatal("source ascending swap did not consume", i, as[i])
		}
	}
	info, e := m.Info(p)
	serviceMust(t, e)
	if info.Size != 3 || info.RefCount != 1 || info.LexSorted {
		t.Fatal(info)
	}
	wantCoefficients := [3]int64{9, -5, 11}
	wantMonomials := [3]Monomial{ms[2], ms[1], ms[0]}
	for i := uint32(0); i < 3; i++ {
		a, e := m.Coefficient(p, i)
		serviceMust(t, e)
		v, e := m.integer.Int64(&a)
		serviceMust(t, e)
		term, e := m.TermMonomial(p, i)
		serviceMust(t, e)
		if v != wantCoefficients[i] || term != wantMonomials[i] {
			t.Fatal(i, v, term)
		}
	}
}

func TestPolynomialCellSourceDuplicateCancellation(t *testing.T) {
	m := polynomialCellFixture(t)
	x, e := m.monomials.MkMonomial([]MonomialPower{{1, 2}})
	serviceMust(t, e)
	u, e := m.monomials.Unit()
	serviceMust(t, e)
	as := [4]util.Int{util.NewInt(7), util.NewInt(-7), util.NewInt(3), util.NewInt(0)}
	p, e := m.MkTerms(as[:], []Monomial{x, x, u, x})
	serviceMust(t, e)
	polynomialCaller(t, m, p)
	info, e := m.Info(p)
	serviceMust(t, e)
	if info.Size != 1 {
		t.Fatal(info)
	}
	a, e := m.Coefficient(p, 0)
	serviceMust(t, e)
	v, e := m.integer.Int64(&a)
	serviceMust(t, e)
	if v != 3 {
		t.Fatal(v)
	}
	for i, want := range [4]int32{7, -7, 3, 0} {
		if as[i] != util.NewInt(want) {
			t.Fatal("const-term copy route", i, as[i])
		}
	}
}

func TestPolynomialCellIndependentArithmeticIdentity(t *testing.T) {
	m := polynomialCellFixture(t)
	x, e := m.MkVariable(0, 1)
	serviceMust(t, e)
	polynomialCaller(t, m, x)
	one := util.NewInt(1)
	u, e := m.MkConstantTake(&one)
	serviceMust(t, e)
	polynomialCaller(t, m, u)
	a, e := m.Add(x, u)
	serviceMust(t, e)
	polynomialCaller(t, m, a)
	b, e := m.Sub(x, u)
	serviceMust(t, e)
	polynomialCaller(t, m, b)
	p, e := m.Mul(a, b)
	serviceMust(t, e)
	polynomialCaller(t, m, p)
	info, e := m.Info(p)
	serviceMust(t, e)
	if info.Size != 2 {
		t.Fatal(info)
	}
	for v := int64(-4); v <= 4; v++ {
		var evaluated int64
		for i := uint32(0); i < info.Size; i++ {
			c, e := m.Coefficient(p, i)
			serviceMust(t, e)
			n, e := m.integer.Int64(&c)
			serviceMust(t, e)
			term, e := m.TermMonomial(p, i)
			serviceMust(t, e)
			d, e := m.monomials.DegreeOf(term, 0)
			serviceMust(t, e)
			power := int64(1)
			for k := uint32(0); k < d; k++ {
				power *= v
			}
			evaluated += n * power
		}
		if evaluated != v*v-1 {
			t.Fatal(v, evaluated)
		}
	}
}
