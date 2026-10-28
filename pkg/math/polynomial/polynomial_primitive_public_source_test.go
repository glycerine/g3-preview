package polynomial

// Source: src/math/polynomial/polynomial.cpp:3536-3541; src/math/polynomial/polynomial.h.

import (
	"fmt"
	"testing"
)

func TestPolynomialPublicPrimitivePartLiteralContent(t *testing.T) {
	m := polynomialPrimitiveFixture(t)
	c := m.core
	p := polynomialQueryTerms(t, c, []int32{2, 2}, [][]MonomialPower{{{0, 1}}, nil})
	r, e := NewPolynomialRef(c)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.CleanupGCD()); serviceMust(t, r.Close()) })
	serviceMust(t, m.PrimitivePart(p, 0, r))
	polynomialQueryExpect(t, c, r.root.value, map[[3]uint32]int64{{1, 0, 0}: 1, {}: 1})
	polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, r.root.value}, nil, r.header)
}

func TestPolynomialPrimitivePredicateLiteralCoefficientContent(t *testing.T) {
	m := polynomialPrimitiveFixture(t)
	c := m.core
	for _, tc := range []struct {
		a    int32
		want bool
	}{{2, false}, {1, true}} {
		p := polynomialQueryTerms(t, c, []int32{tc.a, tc.a}, [][]MonomialPower{{{0, 1}}, nil})
		got, e := m.IsPrimitive(p, 0)
		serviceMust(t, e)
		if got != tc.want {
			t.Fatal("source ICCP then polynomial equality", tc.a, got)
		}
		polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p}, nil, 0)
	}
}

func TestPolynomialPrimitivePublicEveryReservation(t *testing.T) {
	for op := 0; op < 2; op++ {
		success := false
		for ordinal := int64(0); ordinal < 256; ordinal++ {
			done := false
			t.Run(fmt.Sprint(op, "/", ordinal), func(t *testing.T) {
				m := polynomialPrimitiveFixture(t)
				c := m.core
				p := polynomialQueryTerms(t, c, []int32{2, 4, 2}, [][]MonomialPower{{{0, 2}}, {{0, 1}, {1, 1}}, {{1, 2}}})
				out, e := NewPolynomialRef(c)
				serviceMust(t, e)
				var got bool
				c.integer.Budget().FailAfter(ordinal)
				if op == 0 {
					e = m.PrimitivePart(p, 0, out)
				} else {
					got, e = m.IsPrimitive(p, 0)
				}
				polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, out.root.value, out.root.incoming}, nil, out.header)
				c.integer.Budget().FailAfter(-1)
				if e == nil {
					done = true
					if op == 0 {
						polynomialQueryExpect(t, c, out.root.value, map[[3]uint32]int64{{2, 0, 0}: 1, {1, 1, 0}: 2, {0, 2, 0}: 1})
					} else if got {
						t.Fatal("independent nonunit integer content")
					}
				} else if !numericPhysicalReservationRefusal(e) {
					t.Fatal("actual reservation prefix", e)
				}
				serviceMust(t, m.CleanupGCD())
				serviceMust(t, out.Close())
				polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p}, nil, 0)
			})
			if done {
				t.Logf("op%d actual failures0..%d first success%d", op, ordinal-1, ordinal)
				success = true
				break
			}
		}
		if !success {
			t.Fatal("full reservation horizon did not reach success", op)
		}
	}
}
