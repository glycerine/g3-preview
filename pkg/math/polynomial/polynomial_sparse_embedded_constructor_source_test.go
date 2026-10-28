package polynomial

// Source: src/math/polynomial/polynomial.cpp:3077-3098,4120-4123.

import (
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
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
func TestPolynomialSparseEmbeddedNativeOutputConstruction(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	p := polynomialQueryTerms(t, c, []int32{1, 1, 1, 1}, [][]MonomialPower{{{0, 3}}, {{0, 2}}, {{0, 1}}, nil})
	sk, e := NewPolynomialSkeleton(c, p, 0)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, sk.CloseChecked()) })
	var local PolynomialSparseInterpolator
	before, used := c.memory.Stats(), c.integer.Budget().Used()
	c.integer.Budget().FailAfter(0)
	t.Cleanup(func() { c.integer.Budget().FailAfter(-1); serviceMust(t, local.CloseChecked()) })
	serviceMust(t, constructPolynomialSparseInterpolator(full, sk, &local))
	if local.header != 0 || !local.constructed || len(local.outputs) != 4 {
		t.Fatal("actual embedded output construction", local.header, local.constructed, len(local.outputs))
	}
	for _, a := range local.outputs {
		if a != (unumeral.Int{}) {
			t.Fatal("nonzero default output", a)
		}
	}
	if c.memory.Stats() != before || c.integer.Budget().Used() != used {
		t.Fatal("native embedded outputs charged Runtime")
	}
	polynomialSparsePhysical(t, &local, nil, []Polynomial{p})
	serviceMust(t, local.SourceFinalize())
	serviceMust(t, local.CloseChecked())
	if local.outputs != nil || c.memory.Stats() != before || c.integer.Budget().Used() != used {
		t.Fatal("native embedded source/owning cleanup")
	}
}
