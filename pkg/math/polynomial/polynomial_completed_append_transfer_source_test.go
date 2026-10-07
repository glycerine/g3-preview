// Source: src/math/polynomial/polynomial.cpp:3264-3277,6607-6609;
// src/util/obj_ref.h. Licensed under the MIT license.
// Source author: Leonardo de Moura.
package polynomial

import (
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialCoefficientSequentialActualCompletedTransfer(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	p := polynomialICCPUnownedTerms(t, c, []int32{1, 2, 3}, [][]MonomialPower{{{0, 2}}, {{0, 1}}, nil})
	serviceMust(t, c.IncRef(p))
	defer func() { serviceMust(t, c.DecRef(p)) }()
	var coeff [3]polynomialCoefficientInvocationReceipt
	var appendReceipts [3]polynomialCheapAppendInvocationReceipt
	var factory [3]polynomialFactoryInvocationReceipt
	var refs [3]polynomialReference
	defer func() {
		for i := 2; i >= 0; i-- {
			serviceMust(t, c.cleanupFactoryInvocation(&factory[i], true))
			serviceMust(t, c.cleanupCoefficientInvocation(&coeff[i], true))
			serviceMust(t, c.cheap.cleanupAppendInvocation(&appendReceipts[i], true))
			serviceMust(t, refs[i].close(c))
		}
	}()
	for i := 0; i < 3; i++ {
		a, e := c.coeffWithReceipts(p, 0, 2-uint32(i), &coeff[i], &appendReceipts[i], &factory[i])
		polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, a, refs[0].value, refs[1].value, refs[2].value}, nil, 0)
		serviceMust(t, e)
		serviceMust(t, refs[i].assignWithFactoryReceipt(c, a, &factory[i]))
		polynomialQueryExpect(t, c, refs[i].value, map[[3]uint32]int64{{}: int64(i + 1)})
	}
}

func TestPolynomialCompletedAppendActualINCBeforeForeignOldDECCallback(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	p := polynomialICCPUnownedTerms(t, c, []int32{1, 2, 3}, [][]MonomialPower{{{0, 2}}, {{0, 1}}, nil})
	serviceMust(t, c.IncRef(p))
	defer func() { serviceMust(t, c.DecRef(p)) }()
	old := polynomialICCPUnownedTerms(t, c, []int32{7, 1}, [][]MonomialPower{{{0, 1}}, nil})
	var out, foreignOut polynomialReference
	serviceMust(t, out.assign(c, old))
	var coeff, foreignCoeff polynomialCoefficientInvocationReceipt
	var appendReceipt, foreignAppend polynomialCheapAppendInvocationReceipt
	var factory, foreignFactory polynomialFactoryInvocationReceipt
	a, e := c.coeffWithReceipts(p, 0, 2, &coeff, &appendReceipt, &factory)
	serviceMust(t, e)
	calls := 0
	h, e := NewPolynomialDeletionHandler(c.integer.Budget(), func(_ *PolynomialDeletionHandler, c *MultivariatePolynomialCore, deleted Polynomial) error {
		if deleted != old {
			return nil
		}
		calls++
		if c.cheap.matchesAppendInvocation(&appendReceipt) || c.matchesFactoryInvocation(&factory) || out.incoming != a || !out.assigning {
			t.Error("actual incoming INC must acknowledge completed transfer before older callback")
		}
		later, x := c.coeffWithReceipts(p, 0, 1, &foreignCoeff, &foreignAppend, &foreignFactory)
		if x != nil {
			return x
		}
		if x = foreignOut.assignWithFactoryReceipt(c, later, &foreignFactory); x != nil {
			return x
		}
		c.integer.Budget().FailAfter(0)
		_, x = c.memory.Allocate(8)
		return x
	})
	serviceMust(t, e)
	serviceMust(t, c.AddDeletionHandler(h))
	defer func() {
		c.integer.Budget().FailAfter(-1)
		serviceMust(t, c.RemoveDeletionHandler(h))
		serviceMust(t, h.CloseChecked())
		serviceMust(t, c.cleanupFactoryInvocation(&factory, true))
		serviceMust(t, c.cleanupCoefficientInvocation(&coeff, true))
		serviceMust(t, c.cheap.cleanupAppendInvocation(&appendReceipt, true))
		serviceMust(t, c.cleanupFactoryInvocation(&foreignFactory, true))
		serviceMust(t, c.cleanupCoefficientInvocation(&foreignCoeff, true))
		serviceMust(t, c.cheap.cleanupAppendInvocation(&foreignAppend, true))
		serviceMust(t, out.close(c))
		serviceMust(t, foreignOut.close(c))
	}()
	cause := out.assignWithFactoryReceipt(c, a, &factory)
	polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, out.value, out.incoming, foreignOut.value, foreignOut.incoming, c.factoryInvocation.result}, nil, h.header)
	if !numericPhysicalReservationRefusal(cause) || calls != 1 || foreignOut.value == (Polynomial{}) {
		t.Fatal("real later callback coefficient then refusal", cause, calls)
	}
	c.integer.Budget().FailAfter(-1)
	serviceMust(t, c.cleanupFactoryInvocation(&factory, true))
	serviceMust(t, c.cleanupCoefficientInvocation(&coeff, true))
	serviceMust(t, c.cheap.cleanupAppendInvocation(&appendReceipt, true))
	polynomialQueryExpect(t, c, foreignOut.value, map[[3]uint32]int64{{}: 2})
	polynomialQueryExpect(t, c, out.incoming, map[[3]uint32]int64{{}: 1})
	if calls != 1 {
		t.Fatal("older callback replay")
	}
}

func TestPolynomialCompletedAppendForeignMixedRefusalKeepsActualCallerINC(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	p := polynomialICCPUnownedTerms(t, c, []int32{1, 2}, [][]MonomialPower{{{0, 1}}, nil})
	serviceMust(t, c.IncRef(p))
	defer func() { serviceMust(t, c.DecRef(p)) }()
	var coeff polynomialCoefficientInvocationReceipt
	var appendReceipt polynomialCheapAppendInvocationReceipt
	var factory polynomialFactoryInvocationReceipt
	var out polynomialReference
	defer func() {
		serviceMust(t, c.cleanupFactoryInvocation(&factory, true))
		serviceMust(t, c.cleanupCoefficientInvocation(&coeff, true))
		serviceMust(t, out.close(c))
		serviceMust(t, c.cheap.reset())
	}()
	a, e := c.coeffWithReceipts(p, 0, 1, &coeff, &appendReceipt, &factory)
	serviceMust(t, e)
	// A genuine unreceipted source append creates a distinct buffer-owned row.
	// Its actual leaf marks the previous completed cohort as foreign-mixed.
	unit, e := c.monomials.Unit()
	serviceMust(t, e)
	two := util.NewInt(2)
	serviceMust(t, c.cheap.add(&two, unit, false))
	if !c.cheap.appendScope.foreignMixed {
		t.Fatal("actual foreign append mark")
	}
	saved := c.cheap.appendScope
	cause := out.assignWithFactoryReceipt(c, a, &factory)
	polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, out.value, out.incoming, c.factoryInvocation.result}, nil, 0)
	if cause != ErrPolynomialPending || out.incoming != a || !out.assigning || c.factoryInvocation.result != (Polynomial{}) || !testNumericScopeEqual(c.cheap.appendScope, saved) {
		t.Fatal("actual INC must remain caller-owned despite ACK refusal", cause)
	}
	state, used := c.memory.State(), c.integer.Budget().Used()
	foreignCoefficient, e := c.cheap.as.coefficient(c, 0)
	serviceMust(t, e)
	foreignMonomial, e := c.cheap.ms.monomial(c, 0)
	serviceMust(t, e)
	foreignRC, e := c.monomialRefCount(foreignMonomial)
	serviceMust(t, e)
	// The incoming INC remains caller-owned after the completed-transfer ACK
	// refuses. Exact per-slot transfer tags let explicit abort acknowledge the
	// old cohort without acquiring this later source row.
	serviceMust(t, c.cheap.cleanupAppendInvocation(&appendReceipt, true))
	nowCoefficient, e := c.cheap.as.coefficient(c, 0)
	serviceMust(t, e)
	nowMonomial, e := c.cheap.ms.monomial(c, 0)
	serviceMust(t, e)
	nowRC, e := c.monomialRefCount(nowMonomial)
	serviceMust(t, e)
	as, e := c.cheap.as.size(c)
	serviceMust(t, e)
	ms, e := c.cheap.ms.size(c)
	serviceMust(t, e)
	if nowCoefficient != foreignCoefficient || nowMonomial != foreignMonomial || nowRC != foreignRC || as != 1 || ms != 1 || c.cheap.appendScope.active || c.cheap.appendScope.invocation != saved.invocation || out.incoming != a || !out.assigning || c.memory.State() != state || c.integer.Budget().Used() != used {
		t.Fatal("matched abort consumed actual caller INC or later foreign row")
	}
	serviceMust(t, c.cleanupFactoryInvocation(&factory, true))
	polynomialQueryExpect(t, c, out.incoming, map[[3]uint32]int64{{}: 1})
	polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, out.value, out.incoming}, nil, 0)
}

func TestPolynomialActualStoppedAppendGuardPreservesFailurePrefix(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	value := util.Int{}
	serviceMust(t, c.integer.SetLimbs(&value, []uint32{0, 0, 1}))
	defer func() { serviceMust(t, c.integer.Del(&value)) }()
	unit, e := c.monomials.Unit()
	serviceMust(t, e)
	var receipt polynomialCheapAppendInvocationReceipt
	c.integer.Budget().FailAfter(0)
	cause := c.cheap.addWithAppendReceipt(&value, unit, false, &receipt)
	polynomialImpPhysicalAuditNamed(t, m, nil, []*util.Int{&value}, 0)
	if !numericPhysicalReservationRefusal(cause) || !c.cheap.appendScope.stopped {
		t.Fatal("actual source append refusal", cause)
	}
	saved := c.cheap.appendScope
	state, used := c.memory.State(), c.integer.Budget().Used()
	// No factory was constructed: the retained zero marker is tested solely as
	// a concrete adapter guard input, not a source factory or incoming INC.
	if saved.factoryInvocation != 0 {
		t.Fatal("failure precedes factory")
	}
	if e = c.cheap.acknowledgeFactoryAppendTransfer(saved.factoryInvocation); e != ErrPolynomialPending || !testNumericScopeEqual(c.cheap.appendScope, saved) || c.memory.State() != state || c.integer.Budget().Used() != used {
		t.Fatal("stopped append guard changed prefix", e)
	}
	c.integer.Budget().FailAfter(-1)
	serviceMust(t, c.cheap.cleanupAppendInvocation(&receipt, true))
	polynomialImpPhysicalAuditNamed(t, m, nil, []*util.Int{&value}, 0)
}
