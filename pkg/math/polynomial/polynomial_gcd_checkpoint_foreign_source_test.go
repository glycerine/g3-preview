package polynomial

// Source: src/math/polynomial/polynomial.cpp:4352-4506;
// src/util/rlimit.h; src/util/cooperate.h.

import (
	"errors"
	"fmt"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	"testing"
)

func TestPolynomialGCDCheckpointPreservesActualForeignPrimitiveCreation(t *testing.T) {
	seen := false
	for ordinal := int64(0); ordinal < 64 && !seen; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			full := polynomialPSCFullFixture(t)
			c := full.core
			p := polynomialQueryTerms(t, c, []int32{1, 1, 1, 2}, [][]MonomialPower{{{0, 1}, {1, 2}}, {{0, 1}, {1, 1}}, {{1, 1}}, nil})
			q := polynomialQueryTerms(t, c, []int32{1, 3}, [][]MonomialPower{{{0, 1}, {1, 1}}, nil})
			r, e := NewPolynomialRef(c)
			serviceMust(t, e)
			var returned Polynomial
			var callbackError error
			called := false
			t.Cleanup(func() {
				c.integer.Budget().FailAfter(-1)
				serviceMust(t, full.CleanupGCD())
				serviceMust(t, c.CleanupConstruction())
				if returned != (Polynomial{}) {
					serviceMust(t, c.CleanupValue(&returned))
				}
				serviceMust(t, r.Close())
			})
			c.config.Checkpoint = func() bool {
				if called {
					return false
				}
				called = true
				c.integer.Budget().FailAfter(ordinal)
				returned, callbackError = c.MkVariable(0, 17)
				c.integer.Budget().FailAfter(-1)
				return false
			}
			cause := full.Resultant(p, q, 1, r)
			if !called || !errors.Is(cause, ErrUPolynomialCanceled) {
				t.Fatal("actual checkpoint body cancellation", called, cause)
			}
			if callbackError != nil && !numericPhysicalReservationRefusal(callbackError) {
				t.Fatal("actual callback source allocation", callbackError)
			}
			before := c.pending
			if before.raw == (umemory.ObjectRef{}) && before.value == (Polynomial{}) && returned == (Polynomial{}) {
				return
			}
			seen = true
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, q, r.root.value, returned}, nil, r.header)
			serviceMust(t, full.CleanupGCD())
			if c.pending != before {
				t.Fatal("GCD cleanup swept actual checkpoint-owned primitive placement", ordinal, before, c.pending)
			}
			if returned != (Polynomial{}) {
				info, e := c.Info(returned)
				serviceMust(t, e)
				if info.Size != 1 {
					t.Fatal("callback published value lost", info)
				}
			}
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, q, r.root.value, returned}, nil, r.header)
		})
	}
	if !seen {
		t.Fatal("no actual callback construction prefix")
	}
}

func TestPolynomialGCDCooperationPreservesActualForeignPrimitiveCreation(t *testing.T) {
	seen := false
	for ordinal := int64(0); ordinal < 64 && !seen; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			full := polynomialPSCFullFixture(t)
			c := full.core
			p := polynomialQueryTerms(t, c, []int32{1, 1, 1, 2}, [][]MonomialPower{{{0, 1}, {1, 2}}, {{0, 1}, {1, 1}}, {{1, 1}}, nil})
			q := polynomialQueryTerms(t, c, []int32{1, 3}, [][]MonomialPower{{{0, 1}, {1, 1}}, nil})
			r, e := NewPolynomialRef(c)
			serviceMust(t, e)
			var returned Polynomial
			var callbackError error
			called := false
			t.Cleanup(func() {
				c.integer.Budget().FailAfter(-1)
				serviceMust(t, full.CleanupGCD())
				serviceMust(t, c.CleanupConstruction())
				if returned != (Polynomial{}) {
					serviceMust(t, c.CleanupValue(&returned))
				}
				serviceMust(t, r.Close())
			})
			c.config.Checkpoint = func() bool { return true }
			c.config.Cooperation = func() error {
				if called {
					return ErrUPolynomialCanceled
				}
				called = true
				c.integer.Budget().FailAfter(ordinal)
				returned, callbackError = c.MkVariable(0, 17)
				c.integer.Budget().FailAfter(-1)
				return ErrUPolynomialCanceled
			}
			cause := full.Resultant(p, q, 1, r)
			if !called || !errors.Is(cause, ErrUPolynomialCanceled) {
				t.Fatal("actual checkpoint body cancellation", called, cause)
			}
			if callbackError != nil && !numericPhysicalReservationRefusal(callbackError) {
				t.Fatal("actual callback source allocation", callbackError)
			}
			before := c.pending
			if before.raw == (umemory.ObjectRef{}) && before.value == (Polynomial{}) && returned == (Polynomial{}) {
				return
			}
			seen = true
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, q, r.root.value, returned}, nil, r.header)
			serviceMust(t, full.CleanupGCD())
			if c.pending != before {
				t.Fatal("GCD cleanup swept actual checkpoint-owned primitive placement", ordinal, before, c.pending)
			}
			if returned != (Polynomial{}) {
				info, e := c.Info(returned)
				serviceMust(t, e)
				if info.Size != 1 {
					t.Fatal("callback published value lost", info)
				}
			}
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, q, r.root.value, returned}, nil, r.header)
		})
	}
	if !seen {
		t.Fatal("no actual callback construction prefix")
	}
}
