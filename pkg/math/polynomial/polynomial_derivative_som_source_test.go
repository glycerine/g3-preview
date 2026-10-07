// Source: src/math/polynomial/polynomial.cpp:4507-4528,2217-2255.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package polynomial

import (
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialDerivativeMatchedCleanupRetiresActualPendingINC(t *testing.T) {
	found := false
	for ordinal := int64(0); ordinal < 64 && !found; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m := polynomialPSCFullFixture(t)
			c := m.core
			p := polynomialICCPUnownedTerms(t, c, []int32{7}, [][]MonomialPower{{{0, 1}}})
			serviceMust(t, c.IncRef(p))
			defer func() { serviceMust(t, c.DecRef(p)) }()
			var receipt polynomialDerivativeInvocationReceipt
			c.integer.Budget().FailAfter(ordinal)
			result, cause := m.derivativeWithReceipt(p, 0, &receipt)
			polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, result}, nil, 0)
			pending := c.cheap.hasPendingReference
			if pending {
				if !numericPhysicalReservationRefusal(cause) {
					t.Fatal("actual INC before failed pointer push", cause)
				}
				as, e := c.cheap.as.size(c)
				serviceMust(t, e)
				ms, e := c.cheap.ms.size(c)
				serviceMust(t, e)
				if as != 1 || ms != 0 {
					t.Fatal("literal default NUM/Swap then INC/push prefix", as, ms)
				}
				serviceMust(t, m.cleanupDerivativeInvocation(&receipt, false))
				polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, result}, nil, 0)
				if !c.cheap.hasPendingReference {
					t.Fatal("false phase consumed unjoined source INC")
				}
				serviceMust(t, m.cleanupDerivativeInvocation(&receipt, true))
				polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, result}, nil, 0)
				as, e = c.cheap.as.size(c)
				serviceMust(t, e)
				ms, e = c.cheap.ms.size(c)
				serviceMust(t, e)
				if c.cheap.hasPendingReference || as != 0 || ms != 0 {
					t.Fatal("matched derivative abort omitted actual default row/INC", as, ms, c.cheap.hasPendingReference)
				}
				found = true
			}
			c.integer.Budget().FailAfter(-1)
			if !pending {
				serviceMust(t, m.cleanupDerivativeInvocation(&receipt, false))
				serviceMust(t, m.cleanupDerivativeInvocation(&receipt, true))
			}
			if result != (Polynomial{}) {
				serviceMust(t, c.CleanupValue(&result))
			}
		})
	}
	if !found {
		t.Fatal("no reached derivative INC/push refusal")
	}
}

func TestPolynomialDerivativeActualAppendFactoryEveryPrefix(t *testing.T) {
	success := false
	for ordinal := int64(0); ordinal < 96 && !success; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m := polynomialPSCFullFixture(t)
			c := m.core
			p := polynomialICCPUnownedTerms(t, c, []int32{7, 11}, [][]MonomialPower{{{0, 2}}, {{0, 1}}})
			serviceMust(t, c.IncRef(p))
			defer func() { serviceMust(t, c.DecRef(p)) }()
			var receipt polynomialDerivativeInvocationReceipt
			c.integer.Budget().FailAfter(ordinal)
			result, cause := m.derivativeWithReceipt(p, 0, &receipt)
			polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, result}, nil, 0)
			if cause != nil && !numericPhysicalReservationRefusal(cause) {
				t.Fatal("actual source acquisition refusal", cause)
			}
			as, e := c.cheap.as.size(c)
			serviceMust(t, e)
			ms, e := c.cheap.ms.size(c)
			serviceMust(t, e)
			pending := c.cheap.pendingMonomial
			serviceMust(t, m.cleanupDerivativeInvocation(&receipt, false))
			polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, result}, nil, 0)
			nowAs, e := c.cheap.as.size(c)
			serviceMust(t, e)
			nowMs, e := c.cheap.ms.size(c)
			serviceMust(t, e)
			if as != nowAs || ms != nowMs || pending != c.cheap.pendingMonomial {
				t.Fatal("false phase consumed source member acquisitions")
			}
			if cause == nil {
				polynomialQueryExpect(t, c, result, map[[3]uint32]int64{{1, 0, 0}: 14, {}: 11})
				if c.factoryInvocation.active || c.cheap.appendScope.active {
					t.Fatal("successful raw return kept callee ownership")
				}
				serviceMust(t, c.IncRef(result))
			}
			// Recycling a failed polynomial ID can itself reserve backing. Keep
			// its reached stopped graph before lifting the operation-only refusal.
			own := m.cleanupDerivativeInvocation(&receipt, true)
			if own != nil && !numericPhysicalReservationRefusal(own) {
				t.Fatal("matched owning continuation", own)
			}
			polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, result}, nil, 0)
			c.integer.Budget().FailAfter(-1)
			serviceMust(t, m.cleanupDerivativeInvocation(&receipt, true))
			if cause != nil && (m.derivative.bodyCause != cause || result != (Polynomial{})) {
				t.Fatal("own cleanup changed original exception or exposed failed output", cause, m.derivative.bodyCause, result)
			}
			as, e = c.cheap.as.size(c)
			serviceMust(t, e)
			ms, e = c.cheap.ms.size(c)
			serviceMust(t, e)
			if as != 0 || ms != 0 || c.cheap.hasPendingReference {
				t.Fatal("actual derivative append roots survived abort", as, ms)
			}
			polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, result}, nil, 0)
			if cause == nil {
				serviceMust(t, c.DecRef(result))
				success = true
				t.Log("first actual append/factory success", ordinal)
			}
		})
	}
	if !success {
		t.Fatal("every actual prefix reached no successful source call")
	}
}

func TestPolynomialDerivativeAbortPreservesLaterForeignSourceRow(t *testing.T) {
	found := false
	for ordinal := int64(0); ordinal < 96 && !found; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m := polynomialPSCFullFixture(t)
			c := m.core
			p := polynomialICCPUnownedTerms(t, c, []int32{7, 11}, [][]MonomialPower{{{0, 2}}, {{0, 1}}})
			serviceMust(t, c.IncRef(p))
			defer func() { serviceMust(t, c.DecRef(p)) }()
			_, e := c.monomials.MkVar()
			serviceMust(t, e)
			foreign, e := c.monomials.MkVariable(1, 1)
			serviceMust(t, e)
			serviceMust(t, c.monomials.IncRef(foreign))
			defer func() { serviceMust(t, c.monomials.DecRef(foreign)) }()
			var receipt polynomialDerivativeInvocationReceipt
			c.integer.Budget().FailAfter(ordinal)
			result, cause := m.derivativeWithReceipt(p, 0, &receipt)
			polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, result}, nil, 0)
			as, e := c.cheap.as.size(c)
			serviceMust(t, e)
			ms, e := c.cheap.ms.size(c)
			serviceMust(t, e)
			eligible := cause != nil && as == ms && ms != 0 && !c.cheap.hasPendingReference && m.derivative.factory.invocation == 0 && m.derivative.calleeGeneration == 0 && c.monomials.creation.stage == 0
			c.integer.Budget().FailAfter(-1)
			if eligible {
				value := util.NewInt(13)
				serviceMust(t, c.cheap.add(&value, foreign, false))
				before, e := c.monomialRefCount(foreign)
				serviceMust(t, e)
				polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p}, nil, 0)
				serviceMust(t, m.cleanupDerivativeInvocation(&receipt, false))
				serviceMust(t, m.cleanupDerivativeInvocation(&receipt, true))
				a, e := c.cheap.as.coefficient(c, 0)
				serviceMust(t, e)
				term, e := c.cheap.ms.monomial(c, 0)
				serviceMust(t, e)
				after, e := c.monomialRefCount(foreign)
				serviceMust(t, e)
				as, e = c.cheap.as.size(c)
				serviceMust(t, e)
				ms, e = c.cheap.ms.size(c)
				serviceMust(t, e)
				if a != value || term != foreign || before != after || as != 1 || ms != 1 || m.derivative.bodyCause != cause {
					t.Fatal("derivative abort took foreign source acquisition or exception history", a, term, before, after, as, ms)
				}
				polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p}, nil, 0)
				serviceMust(t, c.cheap.reset())
				found = true
			} else {
				serviceMust(t, m.cleanupDerivativeInvocation(&receipt, false))
				serviceMust(t, m.cleanupDerivativeInvocation(&receipt, true))
				if result != (Polynomial{}) {
					serviceMust(t, c.CleanupValue(&result))
				}
			}
		})
	}
	if !found {
		t.Fatal("no actual completed row before later derivative refusal")
	}
}

func TestPolynomialDerivativeFactoryReceiptPreservesForeignReplacement(t *testing.T) {
	found := false
	for ordinal := int64(0); ordinal < 96 && !found; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m := polynomialPSCFullFixture(t)
			c := m.core
			p := polynomialICCPUnownedTerms(t, c, []int32{7, 11}, [][]MonomialPower{{{0, 2}}, {{0, 1}}})
			serviceMust(t, c.IncRef(p))
			defer func() { serviceMust(t, c.DecRef(p)) }()
			var receipt polynomialDerivativeInvocationReceipt
			c.integer.Budget().FailAfter(ordinal)
			result, cause := m.derivativeWithReceipt(p, 0, &receipt)
			polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, result}, nil, 0)
			old := m.derivative.factory
			eligible := cause != nil && c.matchesFactoryInvocation(&old)
			c.integer.Budget().FailAfter(-1)
			if eligible {
				serviceMust(t, c.cleanupFactoryInvocation(&old, true))
				var foreign polynomialFactoryInvocationReceipt
				c.integer.Budget().FailAfter(0)
				later, laterCause := c.mkCellWithReceipt(0, nil, nil, nil, nil, &foreign)
				if laterCause != nil && !numericPhysicalReservationRefusal(laterCause) {
					t.Fatal(laterCause)
				}
				if !c.matchesFactoryInvocation(&foreign) || foreign.invocation == old.invocation || foreign.frame != old.frame {
					t.Fatal("actual foreign constructor did not replace guarded nonce", old, foreign)
				}
				polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, later, c.factoryInvocation.result}, nil, 0)
				c.integer.Budget().FailAfter(-1)
				factory, pending := c.factoryInvocation, c.pending
				serviceMust(t, m.cleanupDerivativeInvocation(&receipt, false))
				serviceMust(t, m.cleanupDerivativeInvocation(&receipt, true))
				if c.factoryInvocation != factory || c.pending != pending || m.derivative.bodyCause != cause {
					t.Fatal("old derivative consumed foreign factory or source history")
				}
				polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, later, c.factoryInvocation.result}, nil, 0)
				serviceMust(t, c.cleanupFactoryInvocation(&foreign, false))
				serviceMust(t, c.cleanupFactoryInvocation(&foreign, true))
				polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p}, nil, 0)
				found = true
			} else {
				serviceMust(t, m.cleanupDerivativeInvocation(&receipt, false))
				serviceMust(t, m.cleanupDerivativeInvocation(&receipt, true))
				if result != (Polynomial{}) {
					serviceMust(t, c.CleanupValue(&result))
				}
			}
		})
	}
	if !found {
		t.Fatal("no actual failed derivative factory acquisition")
	}
}

func TestPolynomialDerivativeReturnedCallerAndForeignCallbackSurviveAbort(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	p := polynomialICCPUnownedTerms(t, c, []int32{7, 11}, [][]MonomialPower{{{0, 2}}, {{0, 1}}})
	serviceMust(t, c.IncRef(p))
	defer func() { serviceMust(t, c.DecRef(p)) }()
	_, e := c.monomials.MkVar()
	serviceMust(t, e)
	old := polynomialICCPUnownedTerms(t, c, []int32{3, 5}, [][]MonomialPower{{{0, 1}}, nil})
	var out polynomialReference
	serviceMust(t, out.assign(c, old))
	var receipt polynomialDerivativeInvocationReceipt
	result, e := m.derivativeWithReceipt(p, 0, &receipt)
	serviceMust(t, e)
	if c.factoryInvocation.active || c.cheap.appendScope.active {
		t.Fatal("completed raw return was not handed off")
	}
	var foreign Monomial
	var foreignNUM util.Int
	calls := 0
	h, e := NewPolynomialDeletionHandler(c.integer.Budget(), func(_ *PolynomialDeletionHandler, c *MultivariatePolynomialCore, deleted Polynomial) error {
		if deleted != old {
			return nil
		}
		calls++
		var e error
		foreign, e = c.monomials.MkVariable(1, 1)
		if e != nil {
			return e
		}
		digits := make([]uint32, 80)
		digits[0], digits[79] = 17, 1
		if e = c.integer.SetLimbs(&foreignNUM, digits); e != nil {
			return e
		}
		c.integer.Budget().FailAfter(0)
		_, e = c.memory.Allocate(8)
		return e
	})
	serviceMust(t, e)
	serviceMust(t, c.AddDeletionHandler(h))
	defer func() {
		c.integer.Budget().FailAfter(-1)
		serviceMust(t, c.RemoveDeletionHandler(h))
		serviceMust(t, h.CloseChecked())
		serviceMust(t, out.close(c))
		serviceMust(t, c.integer.Del(&foreignNUM))
		serviceMust(t, c.monomials.CleanupValue(&foreign))
	}()
	cause := out.assign(c, result)
	polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, out.value, out.incoming}, []*util.Int{&foreignNUM}, h.header)
	if !numericPhysicalReservationRefusal(cause) || calls != 1 || out.incoming != result || !out.assigning {
		t.Fatal("actual caller incoming INC then foreign callback refusal", cause, calls, out)
	}
	c.integer.Budget().FailAfter(-1)
	savedNUM, savedOut := foreignNUM, out
	serviceMust(t, m.cleanupDerivativeInvocation(&receipt, false))
	serviceMust(t, m.cleanupDerivativeInvocation(&receipt, true))
	if foreignNUM != savedNUM || out != savedOut || calls != 1 {
		t.Fatal("derivative abort consumed caller publication or callback NUM")
	}
	degree, e := c.monomials.DegreeOf(foreign, 1)
	serviceMust(t, e)
	if degree != 1 {
		t.Fatal("callback-created monomial disappeared")
	}
	polynomialQueryExpect(t, c, out.incoming, map[[3]uint32]int64{{1, 0, 0}: 14, {}: 11})
	polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, out.value, out.incoming}, []*util.Int{&foreignNUM}, h.header)
}
