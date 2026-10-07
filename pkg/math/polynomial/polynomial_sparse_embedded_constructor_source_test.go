// Source: src/math/polynomial/polynomial.cpp:3077-3098,4120-4123.
package polynomial

import (
	"fmt"
	"testing"
)

func TestPolynomialSparseEmbeddedNilConstructorHasNoReservation(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	z := full.core.integer
	var local PolynomialSparseInterpolator
	before := z.Budget().Used()
	events, x := full.core.memory.AllocationCount()
	serviceMust(t, x)
	z.Budget().FailAfter(0)
	e := constructPolynomialSparseInterpolator(full, nil, &local)
	if e != nil || local.header != 0 || !local.constructed {
		t.Fatal("actual null-skeleton default construction", e, local.header, local.constructed)
	}
	after, x := full.core.memory.AllocationCount()
	serviceMust(t, x)
	if z.Budget().Used() != before || after != events {
		t.Fatal("source null-skeleton ctor invented allocation")
	}
	serviceMust(t, local.SourceFinalize())
	z.Budget().FailAfter(-1)
	serviceMust(t, local.CloseChecked())
	polynomialImpPhysicalAudit(t, full)
}
func TestPolynomialSparseEmbeddedEveryMemberPrefix(t *testing.T) {
	success := false
	for ordinal := int64(0); ordinal < 64 && !success; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			full := polynomialPSCFullFixture(t)
			c := full.core
			p := polynomialQueryTerms(t, c, []int32{1, 1, 1, 1}, [][]MonomialPower{{{0, 3}}, {{0, 2}}, {{0, 1}}, nil})
			sk, e := NewPolynomialSkeleton(c, p, 0)
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, sk.CloseChecked()) })
			var local PolynomialSparseInterpolator
			c.integer.Budget().FailAfter(ordinal)
			e = constructPolynomialSparseInterpolator(full, sk, &local)
			if e != nil && !numericPhysicalReservationRefusal(e) {
				t.Fatal("actual embedded ctor prefix", ordinal, e)
			}
			if local.header != 0 || e != nil && local.constructed {
				t.Fatal("source failure completed class or added header", local.header, local.constructed)
			}
			polynomialSparsePhysical(t, &local, nil, []Polynomial{p})
			c.integer.Budget().FailAfter(-1)
			serviceMust(t, local.CloseChecked())
			polynomialSparsePhysical(t, &local, nil, []Polynomial{p})
			if e == nil {
				success = true
				t.Log("actual embedded source ctor endpoint", ordinal)
			}
		})
	}
	if !success {
		t.Fatal("no completed embedded sparse construction")
	}
}
