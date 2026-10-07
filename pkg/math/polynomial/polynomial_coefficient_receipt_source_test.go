// Source: src/math/polynomial/polynomial.cpp:3264-3277,1392-1423.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package polynomial

import (
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialCoefficientReceiptActualSETBeforeINC(t *testing.T) {
	success, stoppedSet := false, false
	for ordinal := int64(0); ordinal < 128 && !success; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m := polynomialPSCFullFixture(t)
			c := m.core
			var a util.Int
			words := make([]uint32, 8192)
			words[0], words[len(words)-1] = 1, 1
			serviceMust(t, c.integer.SetLimbs(&a, words))
			t.Cleanup(func() { serviceMust(t, c.integer.Del(&a)) })
			term, e := c.monomials.MkMonomial([]MonomialPower{{0, 3}, {1, 7}})
			serviceMust(t, e)
			p, e := c.MkTerms([]util.Int{a}, []Monomial{term})
			serviceMust(t, e)
			p = polynomialCaller(t, c, p)
			var coefficient polynomialCoefficientInvocationReceipt
			var appendReceipt polynomialCheapAppendInvocationReceipt
			var factory polynomialFactoryInvocationReceipt
			c.integer.Budget().FailAfter(ordinal)
			result, e := c.coeffWithReceipts(p, 0, 3, &coefficient, &appendReceipt, &factory)
			if e != nil && !numericPhysicalReservationRefusal(e) {
				t.Fatal(e)
			}
			if e != nil && c.coefficientInvocation.generated != (Monomial{}) && c.cheap.appendScope.coefficients == 1 && c.cheap.appendScope.monomials == 0 && c.cheap.appendScope.pending == (Monomial{}) {
				stoppedSet = true
			}
			polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, result, c.factoryInvocation.result}, []*util.Int{&a}, 0)
			c.integer.Budget().FailAfter(-1)
			serviceMust(t, c.cleanupCoefficientInvocation(&coefficient, false))
			polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, result, c.factoryInvocation.result}, []*util.Int{&a}, 0)
			serviceMust(t, c.cleanupCoefficientInvocation(&coefficient, true))
			serviceMust(t, c.cleanupFactoryInvocation(&factory, true))
			serviceMust(t, c.cheap.cleanupAppendInvocation(&appendReceipt, true))
			polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p}, []*util.Int{&a}, 0)
			if e == nil {
				success = true
				t.Log("first actual coefficient success", ordinal)
			}
		})
	}
	if !success || !stoppedSet {
		t.Fatal("real created monomial then SET-stop before INC horizon missing", success, stoppedSet)
	}
}

func TestPolynomialCoefficientReceiptPreservesOlderForeignConstructor(t *testing.T) {
	found := false
	for ordinal := int64(0); ordinal < 64 && !found; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m := polynomialPSCFullFixture(t)
			c := m.core
			p := polynomialQueryTerms(t, c, []int32{7}, [][]MonomialPower{{{0, 3}, {1, 7}}})
			var older monomialFactoryOutcome
			c.integer.Budget().FailAfter(ordinal)
			v, e := c.monomials.mkMonomialCreated([]MonomialPower{{0, 47}, {1, 59}}, &older)
			c.integer.Budget().FailAfter(-1)
			if e == nil {
				serviceMust(t, c.monomials.CleanupValue(&v))
				return
			}
			if !numericPhysicalReservationRefusal(e) {
				t.Fatal(e)
			}
			if !older.started {
				serviceMust(t, c.monomials.CleanupCreation())
				return
			}
			var receipt polynomialCoefficientInvocationReceipt
			var appendReceipt polynomialCheapAppendInvocationReceipt
			var factory polynomialFactoryInvocationReceipt
			_, e = c.coeffWithReceipts(p, 0, 3, &receipt, &appendReceipt, &factory)
			if e != ErrMonomialPending || c.coefficientInvocation.calleeGeneration != 0 || c.coefficientInvocation.generated != (Monomial{}) {
				t.Fatal("foreign guarded constructor acquired", e)
			}
			before := c.monomials.creation
			used, state := c.integer.Budget().Used(), c.memory.State()
			polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p}, nil, 0)
			serviceMust(t, c.cleanupCoefficientInvocation(&receipt, false))
			serviceMust(t, c.cleanupCoefficientInvocation(&receipt, true))
			if c.monomials.creation != before || c.integer.Budget().Used() != used || c.memory.State() != state {
				t.Fatal("later coefficient receipt consumed older constructor")
			}
			polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p}, nil, 0)
			serviceMust(t, c.monomials.CleanupCreation())
			found = true
		})
	}
	if !found {
		t.Fatal("real older stopped constructor missing")
	}
}

func TestPolynomialCoefficientReceiptUniqueFrameAndRejectedEntry(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	p := polynomialQueryTerms(t, c, []int32{7}, [][]MonomialPower{{{0, 3}, {1, 7}}})
	var old, current polynomialCoefficientInvocationReceipt
	var appendReceipt polynomialCheapAppendInvocationReceipt
	var factory polynomialFactoryInvocationReceipt
	c.integer.Budget().FailAfter(0)
	_, e := c.coeffWithReceipts(p, 0, 3, &old, &appendReceipt, &factory)
	if e == nil || !c.matchesCoefficientInvocation(&old) {
		t.Fatal("real old incomplete coefficient invocation missing", e)
	}
	polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p}, nil, 0)
	c.integer.Budget().FailAfter(-1)
	serviceMust(t, c.cleanupCoefficientInvocation(&old, true))
	serviceMust(t, c.cleanupFactoryInvocation(&factory, true))
	serviceMust(t, c.cheap.cleanupAppendInvocation(&appendReceipt, true))
	appendReceipt = polynomialCheapAppendInvocationReceipt{}
	factory = polynomialFactoryInvocationReceipt{}
	c.integer.Budget().FailAfter(0)
	_, e = c.coeffWithReceipts(p, 0, 3, &current, &appendReceipt, &factory)
	if e == nil || !c.matchesCoefficientInvocation(&current) || old.frame != current.frame || old.invocation == current.invocation {
		t.Fatal("real reused coefficient invocation lacks unique nonce", e)
	}
	before, creation := c.coefficientInvocation, c.monomials.creation
	used, state := c.integer.Budget().Used(), c.memory.State()
	serviceMust(t, c.cleanupCoefficientInvocation(&old, false))
	serviceMust(t, c.cleanupCoefficientInvocation(&old, true))
	foreign := current
	foreign.manager = nil
	serviceMust(t, c.cleanupCoefficientInvocation(&foreign, true))
	rejected := old
	_, e = c.coeffWithReceipts(p, 0, 3, &rejected, &appendReceipt, &factory)
	if e != ErrPolynomialPending || rejected != old || c.coefficientInvocation != before || c.monomials.creation != creation || c.integer.Budget().Used() != used || c.memory.State() != state {
		t.Fatal("stale/foreign/rejected coefficient receipt consumed replacement", e)
	}
	polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p}, nil, 0)
	c.integer.Budget().FailAfter(-1)
	serviceMust(t, c.cleanupCoefficientInvocation(&current, true))
	serviceMust(t, c.cleanupFactoryInvocation(&factory, true))
	serviceMust(t, c.cheap.cleanupAppendInvocation(&appendReceipt, true))
	polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p}, nil, 0)
}

func TestPolynomialCoefficientReceiptCachedMonomialCreatesNoLeafAuthority(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	p := polynomialQueryTerms(t, c, []int32{7}, [][]MonomialPower{{{1, 7}}})
	term, e := c.monomialCore(p, 0)
	serviceMust(t, e)
	before, e := c.monomialRefCount(term)
	serviceMust(t, e)
	var receipt polynomialCoefficientInvocationReceipt
	var appendReceipt polynomialCheapAppendInvocationReceipt
	var factory polynomialFactoryInvocationReceipt
	result, e := c.coeffWithReceipts(p, 0, 0, &receipt, &appendReceipt, &factory)
	serviceMust(t, e)
	if c.coefficientInvocation.generated != (Monomial{}) || c.coefficientInvocation.calleeGeneration != 0 || c.matchesCoefficientInvocation(&receipt) {
		t.Fatal("borrowed cache branch created monomial authority")
	}
	polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, result}, nil, 0)
	serviceMust(t, c.cleanupCoefficientInvocation(&receipt, true))
	serviceMust(t, c.cleanupFactoryInvocation(&factory, true))
	serviceMust(t, c.cheap.cleanupAppendInvocation(&appendReceipt, true))
	after, e := c.monomialRefCount(term)
	serviceMust(t, e)
	if after != before {
		t.Fatal("cached source monomial INC was not balanced by its actual factory owner", before, after)
	}
	polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p}, nil, 0)
}

func TestPolynomialCoefficientReceiptActualCalleeCompletionClearsTag(t *testing.T) {
	found := false
	for ordinal := int64(0); ordinal < 64 && !found; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m := polynomialPSCFullFixture(t)
			c := m.core
			p := polynomialQueryTerms(t, c, []int32{7}, [][]MonomialPower{{{0, 3}, {1, 7}}})
			var receipt polynomialCoefficientInvocationReceipt
			var appendReceipt polynomialCheapAppendInvocationReceipt
			var factory polynomialFactoryInvocationReceipt
			c.integer.Budget().FailAfter(ordinal)
			result, e := c.coeffWithReceipts(p, 0, 3, &receipt, &appendReceipt, &factory)
			if e != nil && !numericPhysicalReservationRefusal(e) {
				t.Fatal(e)
			}
			polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, result, c.factoryInvocation.result}, nil, 0)
			c.integer.Budget().FailAfter(-1)
			s := &c.coefficientInvocation
			if s.calleeGeneration != 0 {
				if c.monomials.creation.coefficientParent != s || c.monomials.creation.coefficientInvocation != receipt.invocation || c.monomials.creation.generation != s.calleeGeneration {
					t.Fatal("actual failed callee lacks exact parent tag")
				}
				serviceMust(t, c.monomials.CleanupCreation())
				if s.calleeGeneration != 0 || c.monomials.creation.coefficientParent != nil || c.monomials.creation.coefficientInvocation != 0 || c.generatedCalleePending {
					t.Fatal("actual successful callee completion did not clear its exact parent tag")
				}
				polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p}, nil, 0)
				found = true
			}
			serviceMust(t, c.cleanupCoefficientInvocation(&receipt, true))
			serviceMust(t, c.cleanupFactoryInvocation(&factory, true))
			serviceMust(t, c.cheap.cleanupAppendInvocation(&appendReceipt, true))
		})
	}
	if !found {
		t.Fatal("actual stopped callee completion boundary missing")
	}
}
