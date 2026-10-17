// Source: src/math/polynomial/polynomial.cpp:6101-6176; polynomial.h:884-913.
package polynomial

import (
	"errors"
	"testing"

	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/z3exception"
)

// No corresponding assertions occur in the original polynomial driver. These
// bounded independent expectations preserve its incomplete sign predicates.
func TestPolynomialSignSourceTable(t *testing.T) {
	m, _, ms := polynomialLexFixture(t)
	c := m.core
	cases := []struct {
		a                        []unumeral.Int
		terms                    []Monomial
		vars                     []PolynomialSign
		sign                     PolynomialSign
		pos, neg, nonpos, nonneg bool
	}{
		{nil, nil, nil, PolynomialSignUnknown, false, false, true, true},
		{[]unumeral.Int{unumeral.NewInt(3)}, []Monomial{ms[4]}, nil, PolynomialSignPositive, true, false, false, true},
		{[]unumeral.Int{unumeral.NewInt(-3)}, []Monomial{ms[4]}, nil, PolynomialSignNegative, false, true, true, false},
		{[]unumeral.Int{unumeral.NewInt(3)}, []Monomial{ms[1]}, []PolynomialSign{PolynomialSignNegative}, PolynomialSignPositive, false, false, false, true},
		{[]unumeral.Int{unumeral.NewInt(-3)}, []Monomial{ms[1]}, []PolynomialSign{PolynomialSignNegative}, PolynomialSignNegative, false, false, true, false},
		{[]unumeral.Int{unumeral.NewInt(3)}, []Monomial{ms[1]}, nil, PolynomialSignUnknown, false, false, false, true},
		{[]unumeral.Int{unumeral.NewInt(3)}, []Monomial{ms[2]}, []PolynomialSign{PolynomialSignPositive, PolynomialSignNegative}, PolynomialSignNegative, false, false, false, false},
		{[]unumeral.Int{unumeral.NewInt(3), unumeral.NewInt(5)}, []Monomial{ms[1], ms[4]}, []PolynomialSign{PolynomialSignPositive}, PolynomialSignPositive, true, false, false, true},
		{[]unumeral.Int{unumeral.NewInt(-3), unumeral.NewInt(5)}, []Monomial{ms[1], ms[4]}, []PolynomialSign{PolynomialSignPositive}, PolynomialSignUnknown, false, false, false, false},
	}
	for i, tc := range cases {
		p, e := c.MkTerms(tc.a, tc.terms)
		serviceMust(t, e)
		serviceMust(t, c.IncRef(p))
		got, e := m.Sign(p, tc.vars)
		serviceMust(t, e)
		if got != tc.sign {
			t.Fatalf("case %d sign %v want %v", i, got, tc.sign)
		}
		actual := []bool{}
		v, e := m.IsPos(p)
		serviceMust(t, e)
		actual = append(actual, v)
		v, e = m.IsNeg(p)
		serviceMust(t, e)
		actual = append(actual, v)
		v, e = m.IsNonPos(p)
		serviceMust(t, e)
		actual = append(actual, v)
		v, e = m.IsNonNeg(p)
		serviceMust(t, e)
		actual = append(actual, v)
		want := []bool{tc.pos, tc.neg, tc.nonpos, tc.nonneg}
		for j := range want {
			if actual[j] != want[j] {
				t.Fatalf("case %d predicate %d got %v want %v", i, j, actual[j], want[j])
			}
		}
		serviceMust(t, c.DecRef(p))
	}
}

func TestPolynomialSignSourceUnreadSuffixAndNoReservation(t *testing.T) {
	m, p, _ := polynomialLexFixture(t)
	c := m.core
	// A default unknown at the first actual variable must skip the later term.
	old, e := c.monomialCore(p, 1)
	serviceMust(t, e)
	serviceMust(t, c.writeMonomial(p, 1, Monomial{}))
	got, e := m.Sign(p, nil)
	serviceMust(t, e)
	if got != PolynomialSignUnknown {
		t.Fatal(got)
	}
	// The first term has odd degree: each even-power predicate returns before
	// looking at the poisoned second term or any source coefficient there.
	for _, query := range []func(Polynomial) (bool, error){m.IsPos, m.IsNeg, m.IsNonPos, m.IsNonNeg} {
		v, e := query(p)
		serviceMust(t, e)
		if v {
			t.Fatal("odd term accepted")
		}
	}
	serviceMust(t, c.writeMonomial(p, 1, old))
	before := c.integer.Budget().Used()
	c.integer.Budget().FailAfter(0)
	got, e = m.Sign(p, []PolynomialSign{PolynomialSignPositive, PolynomialSignPositive})
	serviceMust(t, e)
	if got != PolynomialSignPositive {
		t.Fatal(got)
	}
	for _, query := range []func(Polynomial) (bool, error){m.IsPos, m.IsNeg, m.IsNonPos, m.IsNonNeg} {
		_, e = query(p)
		serviceMust(t, e)
	}
	if c.integer.Budget().Used() != before {
		t.Fatal("query changed cold accounting")
	}
	if e = c.integer.Budget().Reserve(0); e == nil {
		t.Fatal("query consumed/disabled actual reservation refusal")
	}
	c.integer.Budget().FailAfter(-1)
	polynomialImpPhysicalAuditValues(t, m, []Polynomial{p})
}

func TestPolynomialSignSourceZeroCoefficientAndOwnerGuards(t *testing.T) {
	m, _, ms := polynomialLexFixture(t)
	c := m.core
	p, e := c.MkTerms([]unumeral.Int{unumeral.NewInt(7)}, []Monomial{ms[4]})
	serviceMust(t, e)
	serviceMust(t, c.IncRef(p))
	// Invalid-Go stored coefficient domain isolates the literal is_pos ternary:
	// source zero maps to l_false; both signed predicates still reject it.
	serviceMust(t, c.writeWord(p, 12, 0))
	got, e := m.Sign(p, nil)
	serviceMust(t, e)
	if got != PolynomialSignNegative {
		t.Fatal(got)
	}
	for _, query := range []func(Polynomial) (bool, error){m.IsPos, m.IsNeg, m.IsNonPos, m.IsNonNeg} {
		v, e := query(p)
		serviceMust(t, e)
		if v {
			t.Fatal("zero coefficient accepted")
		}
	}
	serviceMust(t, c.writeWord(p, 12, 7))
	serviceMust(t, c.DecRef(p))
	copied := *m
	if _, e = copied.Sign(c.zero, nil); e != ErrPolynomialOwner {
		t.Fatal("copied owner", e)
	}
	if _, e = m.Sign(Polynomial{}, nil); e == nil {
		t.Fatal("foreign polynomial accepted")
	}
	var absent *MultivariatePolynomialManager
	if _, e = absent.IsPos(c.zero); e != ErrPolynomialOwner {
		t.Fatal("nil owner", e)
	}
}

func TestPolynomialSignSourceModularCoefficientBoundary(t *testing.T) {
	m, _, ms := polynomialLexFixture(t)
	polynomialThrowAssertionFixture(t)
	c := m.core
	p, e := c.MkTerms([]unumeral.Int{unumeral.NewInt(7)}, []Monomial{ms[4]})
	serviceMust(t, e)
	serviceMust(t, c.IncRef(p))
	defer func() { serviceMust(t, c.modular.SetZ()); serviceMust(t, c.DecRef(p)) }()
	serviceMust(t, c.modular.SetZPUint64(5))
	_, e = m.Sign(p, nil)
	if polynomialSourceDebug {
		var thrown z3exception.DefaultException
		if !errors.As(e, &thrown) {
			t.Fatal("literal reached modular is_pos guard", e)
		}
	} else {
		serviceMust(t, e)
	}
	// Normalized signed field coefficients use the same source wrapper.
	a := unumeral.NewInt(-2)
	q, e := c.MkTerms([]unumeral.Int{a}, []Monomial{ms[4]})
	serviceMust(t, e)
	serviceMust(t, c.IncRef(q))
	got, e := m.Sign(q, nil)
	serviceMust(t, e)
	if got != PolynomialSignNegative {
		t.Fatal(got)
	}
	v, e := m.IsNeg(q)
	serviceMust(t, e)
	if !v {
		t.Fatal("negative normalized constant")
	}
	serviceMust(t, c.DecRef(q))
}
