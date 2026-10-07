// Source: src/math/polynomial/polynomial.cpp:2217-2255,2280-2299.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package polynomial

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialCheapAppendReceiptActualActionsAndZeroGuard(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	var r polynomialCheapAppendInvocationReceipt
	zero := util.NewInt(0)
	serviceMust(t, c.cheap.addWithAppendReceipt(&zero, c.monomials.unit, false, &r))
	if r != (polynomialCheapAppendInvocationReceipt{}) {
		t.Fatal("source zero guard created an append receipt")
	}
	a := util.NewInt(7)
	serviceMust(t, c.cheap.addWithAppendReceipt(&a, c.monomials.unit, false, &r))
	if !c.cheap.matchesAppendInvocation(&r) || c.cheap.appendScope.coefficients != 1 || c.cheap.appendScope.monomials != 1 {
		t.Fatal("actual coefficient construction and INC publication were not recorded")
	}
	before := c.cheap.appendScope
	used, state := c.integer.Budget().Used(), c.memory.State()
	serviceMust(t, c.cheap.cleanupAppendInvocation(&r, false))
	if !testNumericScopeEqual(c.cheap.appendScope, before) {
		t.Fatal("completed phase consumed an unjoined shared-buffer cohort")
	}
	if c.integer.Budget().Used() != used || c.memory.State() != state {
		t.Fatal("completed phase changed physical graph")
	}
	polynomialImpPhysicalAuditNamed(t, m, nil, nil, 0)
	serviceMust(t, c.cheap.cleanupAppendInvocation(&r, true))
	n, e := c.cheap.ms.size(c)
	serviceMust(t, e)
	as, e := c.cheap.as.size(c)
	serviceMust(t, e)
	if n != 0 || as != 0 {
		t.Fatal("matched actual owning append cohort was not retired")
	}
	polynomialImpPhysicalAuditNamed(t, m, nil, nil, 0)
}

func TestPolynomialCheapAppendReceiptEveryActualConstructionPrefix(t *testing.T) {
	for _, large := range []bool{false, true} {
		success := false
		constructedBeforeStop := false
		for ordinal := int64(0); ordinal < 64 && !success; ordinal++ {
			t.Run(fmt.Sprint(large, "/", ordinal), func(t *testing.T) {
				m := polynomialPSCFullFixture(t)
				c := m.core
				var r polynomialCheapAppendInvocationReceipt
				a := util.NewInt(7)
				if large {
					limbs := make([]uint32, 8192)
					limbs[0], limbs[len(limbs)-1] = 1, 1
					serviceMust(t, c.integer.SetLimbs(&a, limbs))
					t.Cleanup(func() { serviceMust(t, c.integer.Del(&a)) })
				}
				c.integer.Budget().FailAfter(ordinal)
				e := c.cheap.addWithAppendReceipt(&a, c.monomials.unit, false, &r)
				if e != nil && !numericPhysicalReservationRefusal(e) {
					t.Fatal(e)
				}
				if e != nil && c.cheap.appendScope.coefficients == 1 && c.cheap.appendScope.monomials == 0 && c.cheap.appendScope.pending == (Monomial{}) {
					constructedBeforeStop = true
				}
				polynomialImpPhysicalAuditNamed(t, m, nil, []*util.Int{&a}, 0)
				c.integer.Budget().FailAfter(-1)
				serviceMust(t, c.cheap.cleanupAppendInvocation(&r, false))
				polynomialImpPhysicalAuditNamed(t, m, nil, []*util.Int{&a}, 0)
				serviceMust(t, c.cheap.cleanupAppendInvocation(&r, true))
				polynomialImpPhysicalAuditNamed(t, m, nil, []*util.Int{&a}, 0)
				if e == nil {
					success = true
					t.Log("actual source append first success", ordinal)
				}
			})
		}
		if !success {
			t.Fatal("actual append first success missing")
		}
		if large && !constructedBeforeStop {
			t.Fatal("BIG SET stop after actual NUM construction and before INC not reached")
		}
	}
}

func TestPolynomialCheapAppendReceiptMixedForeignRefusal(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	_, e := c.monomials.MkVar()
	serviceMust(t, e)
	variable, e := c.monomials.MkMonomial([]MonomialPower{{0, 1}})
	serviceMust(t, e)
	var r polynomialCheapAppendInvocationReceipt
	a, b := util.NewInt(7), util.NewInt(11)
	serviceMust(t, c.cheap.addWithAppendReceipt(&a, c.monomials.unit, false, &r))
	// An actual nil-receipt append is foreign to the first explicit cohort.
	serviceMust(t, c.cheap.add(&b, variable, false))
	// Invalid-Go adapter provenance fixture: actual source rows remain intact,
	// but the claimed old monomial slot no longer has its acquisition tag. This
	// is guard evidence; the separate known-row test proves successful retirement.
	c.cheap.monomialOwners[0] = 0
	before := c.cheap.appendScope
	used, state := c.integer.Budget().Used(), c.memory.State()
	a0, e := c.cheap.as.coefficient(c, 0)
	serviceMust(t, e)
	a1, e := c.cheap.as.coefficient(c, 1)
	serviceMust(t, e)
	m0, e := c.cheap.ms.monomial(c, 0)
	serviceMust(t, e)
	m1, e := c.cheap.ms.monomial(c, 1)
	serviceMust(t, e)
	polynomialImpPhysicalAuditNamed(t, m, nil, nil, 0)
	e = c.cheap.cleanupAppendInvocation(&r, true)
	if !errors.Is(e, ErrPolynomialPending) {
		t.Fatal("mixed ownership must stop before any cohort is consumed", e)
	}
	n, e := c.cheap.ms.size(c)
	serviceMust(t, e)
	as, e := c.cheap.as.size(c)
	serviceMust(t, e)
	now0, e := c.cheap.as.coefficient(c, 0)
	serviceMust(t, e)
	now1, e := c.cheap.as.coefficient(c, 1)
	serviceMust(t, e)
	term0, e := c.cheap.ms.monomial(c, 0)
	serviceMust(t, e)
	term1, e := c.cheap.ms.monomial(c, 1)
	serviceMust(t, e)
	if n != 2 || as != 2 || now0 != a0 || now1 != a1 || term0 != m0 || term1 != m1 || !testNumericScopeEqual(before, c.cheap.appendScope) {
		t.Fatal("foreign graph changed on mixed ownership refusal")
	}
	if c.integer.Budget().Used() != used || c.memory.State() != state {
		t.Fatal("mixed refusal changed physical graph")
	}
	polynomialImpPhysicalAuditNamed(t, m, nil, nil, 0)
}

func TestPolynomialCheapAppendReceiptUniqueReplacement(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	a := util.NewInt(7)
	var old, current polynomialCheapAppendInvocationReceipt
	serviceMust(t, c.cheap.addWithAppendReceipt(&a, c.monomials.unit, false, &old))
	serviceMust(t, c.cheap.cleanupAppendInvocation(&old, true))
	serviceMust(t, c.cheap.addWithAppendReceipt(&a, c.monomials.unit, false, &current))
	if old.frame != current.frame || old.invocation == current.invocation {
		t.Fatal("stable append frame lacks unique invocation")
	}
	before := c.cheap.appendScope
	used, state := c.integer.Budget().Used(), c.memory.State()
	serviceMust(t, c.cheap.cleanupAppendInvocation(&old, true))
	foreign := current
	foreign.manager = nil
	serviceMust(t, c.cheap.cleanupAppendInvocation(&foreign, true))
	if !testNumericScopeEqual(c.cheap.appendScope, before) || c.integer.Budget().Used() != used || c.memory.State() != state {
		t.Fatal("old/foreign append receipt consumed replacement")
	}
	polynomialImpPhysicalAuditNamed(t, m, nil, nil, 0)
	serviceMust(t, c.cheap.cleanupAppendInvocation(&current, true))
	polynomialImpPhysicalAuditNamed(t, m, nil, nil, 0)
}

func TestPolynomialCheapAppendReceiptForeignSameCardinality(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	a, foreign := util.NewInt(7), util.NewInt(11)
	var old polynomialCheapAppendInvocationReceipt
	serviceMust(t, c.cheap.addWithAppendReceipt(&a, c.monomials.unit, false, &old))
	p, e := c.cheap.mk(false)
	serviceMust(t, e)
	p = polynomialCaller(t, c, p)
	// Real source transfer/reset followed by a real foreign constructor.
	// Its row counts coincide with the old receipt's former cohort.
	serviceMust(t, c.cheap.add(&foreign, c.monomials.unit, false))
	before := c.cheap.appendScope
	used, state := c.integer.Budget().Used(), c.memory.State()
	row, e := c.cheap.as.coefficient(c, 0)
	serviceMust(t, e)
	term, e := c.cheap.ms.monomial(c, 0)
	serviceMust(t, e)
	refCount, e := c.monomialRefCount(term)
	serviceMust(t, e)
	if !before.foreignMixed || before.coefficients != 1 || before.monomials != 1 {
		t.Fatal("foreign actual slot constructor was not tagged")
	}
	e = c.cheap.cleanupAppendInvocation(&old, true)
	if e != nil {
		t.Fatal("actual source transfer consumed all old tags; acknowledgement must not own the foreign row", e)
	}
	now, e := c.cheap.as.coefficient(c, 0)
	serviceMust(t, e)
	nowTerm, e := c.cheap.ms.monomial(c, 0)
	serviceMust(t, e)
	nowRefCount, e := c.monomialRefCount(nowTerm)
	serviceMust(t, e)
	if now != row || nowTerm != term || nowRefCount != refCount || before.invocation != c.cheap.appendScope.invocation || c.cheap.appendScope.active || c.integer.Budget().Used() != used || c.memory.State() != state {
		t.Fatal("old receipt consumed same-size foreign cohort")
	}
	polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p}, nil, 0)
}

func TestPolynomialCheapAppendReceiptActualFactoryTransferPrefixes(t *testing.T) {
	success := false
	for ordinal := int64(0); ordinal < 64 && !success; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m := polynomialPSCFullFixture(t)
			c := m.core
			a := util.NewInt(7)
			var appendReceipt polynomialCheapAppendInvocationReceipt
			serviceMust(t, c.cheap.addWithAppendReceipt(&a, c.monomials.unit, false, &appendReceipt))
			var factory polynomialFactoryInvocationReceipt
			c.integer.Budget().FailAfter(ordinal)
			p, e := c.cheap.mkWithAppendReceipt(false, &appendReceipt, &factory)
			if e != nil && !numericPhysicalReservationRefusal(e) {
				t.Fatal(e)
			}
			s := &c.cheap.appendScope
			polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, c.factoryInvocation.result}, []*util.Int{&s.movingCoefficient}, 0)
			c.integer.Budget().FailAfter(-1)
			serviceMust(t, c.cleanupFactoryInvocation(&factory, false))
			serviceMust(t, c.cheap.cleanupAppendInvocation(&appendReceipt, false))
			polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, c.factoryInvocation.result}, []*util.Int{&s.movingCoefficient}, 0)
			if e == nil {
				if s.transferredCoefficients != 1 || s.transferredMonomials != 1 || !s.completedTransfer || s.factoryInvocation != factory.invocation {
					t.Fatal("actual constructor transfer actions were not uniquely recorded")
				}
				value, x := c.Coefficient(p, 0)
				serviceMust(t, x)
				if value != a {
					t.Fatal("source placement move changed coefficient")
				}
			}
			serviceMust(t, c.cleanupFactoryInvocation(&factory, true))
			serviceMust(t, c.cheap.cleanupAppendInvocation(&appendReceipt, true))
			polynomialImpPhysicalAuditNamed(t, m, nil, nil, 0)
			if e == nil {
				success = true
				t.Log("actual owned factory transfer first success", ordinal)
			}
		})
	}
	if !success {
		t.Fatal("actual source transfer success not reached")
	}
}
