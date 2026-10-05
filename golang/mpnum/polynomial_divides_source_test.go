// Source: src/math/polynomial/polynomial.cpp:5154-5194;
// src/math/polynomial/polynomial.h:817-819.
// Licensed under the MIT license. Source author: Leonardo de Moura (leonardo).
package mpnum

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestPolynomialDividesIndependentFiniteAndZeroReadMask(t *testing.T) {
	c := polynomialCellFixture(t)
	x1 := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, nil})
	xm1 := polynomialQueryTerms(t, c, []int32{1, -1}, [][]MonomialPower{{{0, 1}}, nil})
	squareMinus1 := polynomialQueryTerms(t, c, []int32{1, -1}, [][]MonomialPower{{{0, 2}}, nil})
	cases := []struct {
		q, p Polynomial
		want bool
	}{
		{x1, squareMinus1, true}, {xm1, squareMinus1, true}, {squareMinus1, x1, false},
		{x1, xm1, false}, {c.unit, x1, true}, {Polynomial{owner: 999}, c.zero, true},
	}
	for i, a := range cases {
		got, e := c.Divides(a.q, a.p)
		serviceMust(t, e)
		if got != a.want {
			t.Fatal("independent exact divisibility", i, got, a.want)
		}
	}
	polynomialPhysicalAudit(t, c)
}

func TestPolynomialDividesEveryActualReservationPrefix(t *testing.T) {
	for _, want := range []bool{true, false} {
		success := false
		for ordinal := int64(0); ordinal < 128 && !success; ordinal++ {
			t.Run(fmt.Sprintf("%v/%d", want, ordinal), func(t *testing.T) {
				budget := arena.NewBudget(1 << 24)
				qowner, e := NewRationalManager(RationalConfig{Integers: IntegerConfig{Cells: arena.Config{Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}}})
				serviceMust(t, e)
				t.Cleanup(func() {
					qowner.Close()
					if budget.Used() != 0 {
						t.Errorf("actual remaining bytes %d", budget.Used())
					}
				})
				full, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: qowner.z, Rationals: qowner, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
				serviceMust(t, e)
				t.Cleanup(func() { serviceMust(t, full.CloseChecked()) })
				c := full.core
				for i := 0; i < 2; i++ {
					_, e = c.monomials.MkVar()
					serviceMust(t, e)
				}
				// The actual quotient x^3*y^2 is absent from both operands;
				// the real Direct source allocator must create its payload.
				q := polynomialQueryTerms(t, c, []int32{1}, [][]MonomialPower{{{0, 2}, {1, 2}}})
				coefficients := []int32{1}
				terms := [][]MonomialPower{{{0, 5}, {1, 4}}}
				if !want {
					coefficients = append(coefficients, 1)
					terms = append(terms, []MonomialPower{{0, 1}})
				}
				p := polynomialQueryTerms(t, c, coefficients, terms)

				t.Cleanup(func() { c.integer.budget.FailAfter(-1); serviceMust(t, c.CleanupDivides()) })
				c.integer.budget.FailAfter(ordinal)
				got, e := c.Divides(q, p)
				if e != nil && !numericPhysicalReservationRefusal(e) {
					t.Fatal("actual source reservation category", ordinal, e)
				}
				polynomialImpPhysicalAuditValues(t, full, []Polynomial{p, q})
				c.integer.budget.FailAfter(-1)
				serviceMust(t, c.CleanupDivides())
				polynomialImpPhysicalAuditValues(t, full, []Polynomial{p, q})
				if e == nil {
					if got != want {
						t.Fatal("independent boolean divisibility", got, want)
					}
					success = true
					t.Log("actual reservation endpoint", want, ordinal)
				}
			})
		}
		if !success {
			t.Fatal("no completed actual divide route", want)
		}
	}
}

func TestPolynomialDividesBigNumericPendingBlocksOwningClose(t *testing.T) {
	budget := arena.NewBudget(1 << 24)
	q, e := NewRationalManager(RationalConfig{Integers: IntegerConfig{Cells: arena.Config{PageSize: 8, SlotLimit: 10, Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}}})
	serviceMust(t, e)
	t.Cleanup(func() {
		q.Close()
		if budget.Used() != 0 {
			t.Errorf("remaining actual bytes %d", budget.Used())
		}
	})
	full, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.z, Rationals: q, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
	serviceMust(t, e)
	c := full.core
	t.Cleanup(func() { serviceMust(t, c.CleanupDivides()); serviceMust(t, full.CloseChecked()) })
	_, e = c.monomials.MkVar()
	serviceMust(t, e)
	divisor := polynomialQueryTerms(t, c, []int32{1}, [][]MonomialPower{{{0, 2}}})
	term, e := c.monomials.MkVariable(0, 5)
	serviceMust(t, e)
	p, e := c.MkTerms([]Int{q.z.two64}, []Monomial{term})
	serviceMust(t, e)
	p = polynomialCaller(t, c, p)
	_, e = c.Divides(divisor, p)
	if !errors.Is(e, arena.ErrSlotLimit) || q.z.pendingCell.object == (memory.ObjectRef{}) || !c.dividesLocal.active {
		t.Fatal("actual source quotient numeric Copy descriptor refusal", e, q.z.pendingCell, c.dividesLocal)
	}
	polynomialImpPhysicalAuditValues(t, full, []Polynomial{p, divisor})
	if e = c.CloseChecked(); !errors.Is(e, ErrPolynomialPending) {
		t.Fatal("core owning close must stop before caller/private mutations", e)
	}
	if e = full.CloseChecked(); !errors.Is(e, ErrPolynomialPending) {
		t.Fatal("full owning close must stop before caller/private mutations", e)
	}
	if c.closing || full.closing {
		t.Fatal("pending preguard started owning shutdown")
	}
	receipt := c.dividesLocal.directReceipt
	actual := q.z.pendingCell
	if receipt.frame.numeric.cell != actual {
		t.Fatal("actual quotient allocation did not acquire its full backing at the leaf", receipt.frame.numeric.cell, actual)
	}
	source := c.dividesInvocationSourceCause(&receipt)
	serviceMust(t, c.cleanupDividesInvocation(&receipt, false))
	if q.z.pendingCell != actual || receipt.frame.numeric.cell != actual || !receipt.frame.active || c.dividesInvocationSourceCause(&receipt) != source {
		t.Fatal("completed suffix consumed unjoined quotient allocation")
	}
	polynomialImpPhysicalAuditValues(t, full, []Polynomial{p, divisor})
	serviceMust(t, c.cleanupDividesInvocation(&receipt, true))
	if c.dividesInvocationSourceCause(&receipt) != source {
		t.Fatal("true own erased source history")
	}
	if q.z.pendingCell.object != (memory.ObjectRef{}) {
		t.Fatal("actual matching numerical pending retained after owning cleanup")
	}
	polynomialImpPhysicalAuditValues(t, full, []Polynomial{p, divisor})
}
