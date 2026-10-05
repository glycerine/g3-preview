// Source: src/util/obj_ref.h; src/math/polynomial/polynomial.cpp:5473-5500.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import (
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestPolynomialFactoryActualIncomingINCTransfersBeforeOldDECCallback(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	old := polynomialICCPUnownedTerms(t, c, []int32{1, 5}, [][]MonomialPower{{{0, 1}}, nil})
	var out polynomialReference
	serviceMust(t, out.assign(c, old))
	var receipt polynomialFactoryInvocationReceipt
	p, e := c.mkCellWithReceipt(0, nil, nil, nil, nil, &receipt)
	serviceMust(t, e)
	var refused memory.Block
	calls := 0
	h, e := NewPolynomialDeletionHandler(c.integer.budget, func(_ *PolynomialDeletionHandler, c *MultivariatePolynomialCore, deleted Polynomial) error {
		if deleted != old {
			return nil
		}
		calls++
		if c.factoryInvocation.result != (Polynomial{}) || c.matchesFactoryInvocation(&receipt) || out.incoming != p || !out.assigning {
			t.Error("actual incoming INC must transfer raw factory before old DEC callback")
		}
		c.integer.budget.FailAfter(0)
		var x error
		refused, x = c.memory.Allocate(8)
		return x
	})
	serviceMust(t, e)
	serviceMust(t, c.AddDeletionHandler(h))
	e = out.assignWithFactoryReceipt(c, p, &receipt)
	if !numericPhysicalReservationRefusal(e) || calls != 1 || !out.assigning {
		t.Fatal("actual old DEC callback refusal not reached", e, calls)
	}
	polynomialImpPhysicalAuditNamedRaw(t, m, []Polynomial{out.value, out.incoming, c.factoryInvocation.result}, nil, h.header, []memory.Block{refused})
	rejected := receipt
	used, state := c.integer.budget.Used(), c.memory.State()
	if reject := out.assignWithFactoryReceipt(c, c.unit, &rejected); reject != ErrPolynomialPending || rejected != receipt || c.integer.budget.Used() != used || c.memory.State() != state {
		t.Fatal("real retained assignment guard changed receipt or graph", reject)
	}
	c.integer.budget.FailAfter(-1)
	// Explicit transfer here makes teardown safe even for the behavioral RED
	// implementation that omitted the earlier real INC frontier.
	serviceMust(t, c.transferFactoryResult(&receipt, p))
	serviceMust(t, c.cleanupFactoryInvocation(&receipt, true))
	serviceMust(t, out.close(c))
	if calls != 1 {
		t.Fatal("actual callback was replayed")
	}
	serviceMust(t, c.RemoveDeletionHandler(h))
	serviceMust(t, h.CloseChecked())
}

func TestPolynomialFactoryReferenceWrongPointerAndForeignReceiptGuards(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	var receipt polynomialFactoryInvocationReceipt
	p, e := c.mkCellWithReceipt(0, nil, nil, nil, nil, &receipt)
	serviceMust(t, e)
	var out polynomialReference
	before := c.factoryInvocation
	used, state := c.integer.budget.Used(), c.memory.State()
	if e = out.assignWithFactoryReceipt(c, c.unit, &receipt); e != ErrPolynomialOwner || out != (polynomialReference{}) || c.factoryInvocation != before || c.integer.budget.Used() != used || c.memory.State() != state {
		t.Fatal("matching wrong pointer reached actual INC", e)
	}
	foreign := receipt
	foreign.manager = nil
	serviceMust(t, out.assignWithFactoryReceipt(c, c.unit, &foreign))
	serviceMust(t, out.close(c))
	if c.factoryInvocation != before || foreign.manager != nil || foreign.frame != receipt.frame || foreign.invocation != receipt.invocation {
		t.Fatal("foreign receipt consumed guarded raw factory")
	}
	polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p}, nil, 0)
	serviceMust(t, c.cleanupFactoryInvocation(&receipt, true))
}
