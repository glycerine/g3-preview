// Source: src/math/polynomial/polynomial.cpp:1392-1423,2481-2495.
package polynomial

import (
	"fmt"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestPolynomialFactoryReceiptActualConstructorPrefixes(t *testing.T) {
	success := false
	for ordinal := int64(0); ordinal < 64 && !success; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m := polynomialPSCFullFixture(t)
			c := m.core
			var receipt polynomialFactoryInvocationReceipt
			c.integer.Budget().FailAfter(ordinal)
			p, e := c.mkCellWithReceipt(0, nil, nil, nil, nil, &receipt)
			if e != nil && !numericPhysicalReservationRefusal(e) {
				t.Fatal(e)
			}
			if !c.matchesFactoryInvocation(&receipt) {
				t.Fatal("actual guarded constructor did not stamp receipt")
			}
			polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, c.factoryInvocation.result}, nil, 0)
			c.integer.Budget().FailAfter(-1)
			serviceMust(t, c.cleanupFactoryInvocation(&receipt, false))
			polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, c.factoryInvocation.result}, nil, 0)
			serviceMust(t, c.cleanupFactoryInvocation(&receipt, true))
			polynomialImpPhysicalAuditNamed(t, m, nil, nil, 0)
			if e == nil {
				success = true
				t.Log("actual empty constructor first success", ordinal)
			}
		})
	}
	if !success {
		t.Fatal("actual constructor horizon not reached")
	}
}

func TestPolynomialFactoryReceiptReplacementAndRejectedEntry(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	var old polynomialFactoryInvocationReceipt
	p, e := c.mkCellWithReceipt(0, nil, nil, nil, nil, &old)
	serviceMust(t, e)
	p = polynomialCaller(t, c, p)
	serviceMust(t, c.transferFactoryResult(&old, p))
	var current polynomialFactoryInvocationReceipt
	c.integer.Budget().FailAfter(0)
	_, e = c.mkCellWithReceipt(15000, nil, nil, nil, nil, &current)
	if e == nil {
		t.Fatal("actual allocation refusal not reached")
	}
	c.integer.Budget().FailAfter(-1)
	if current.invocation == old.invocation || current.frame != old.frame {
		t.Fatal("stable frame must have distinct invocation")
	}
	before := c.factoryInvocation
	pending := c.pending
	serviceMust(t, c.cleanupFactoryInvocation(&old, false))
	serviceMust(t, c.cleanupFactoryInvocation(&old, true))
	foreign := current
	foreign.manager = nil
	serviceMust(t, c.cleanupFactoryInvocation(&foreign, true))
	if c.factoryInvocation != before || c.pending != pending {
		t.Fatal("stale/foreign receipt consumed replacement")
	}
	rejected := old
	_, e = c.mkCellWithReceipt(0, nil, nil, nil, nil, &rejected)
	if e == nil || rejected != old {
		t.Fatal("rejected actual constructor changed receipt")
	}
	polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p}, nil, 0)
	serviceMust(t, c.cleanupFactoryInvocation(&current, true))
	polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p}, nil, 0)
}

func TestPolynomialFactoryReceiptCheapTransferOwnsActualFactory(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	a := unumeral.NewInt(7)
	serviceMust(t, c.cheap.add(&a, c.monomials.unit, false))
	var receipt polynomialFactoryInvocationReceipt
	p, e := c.cheap.mkWithFactoryReceipt(false, &receipt)
	serviceMust(t, e)
	if c.factoryInvocation.result != p || !c.matchesFactoryInvocation(&receipt) {
		t.Fatal("actual published factory was not transferred before pending clear")
	}
	n, e := c.cheap.as.size(c)
	serviceMust(t, e)
	if n != 0 || c.cheap.transferred != 0 {
		t.Fatal("source cheap.mk did not transfer and reset its own coefficients")
	}
	polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p}, nil, 0)
	serviceMust(t, c.cleanupFactoryInvocation(&receipt, false))
	value, e := c.Coefficient(p, 0)
	serviceMust(t, e)
	if value != unumeral.NewInt(7) {
		t.Fatal("completed phase consumed unjoined placement")
	}
	serviceMust(t, c.cleanupFactoryInvocation(&receipt, true))
	polynomialImpPhysicalAuditNamed(t, m, nil, nil, 0)
}

func TestPolynomialFactoryReceiptPreservesUnreceiptedReplacement(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	var old polynomialFactoryInvocationReceipt
	c.integer.Budget().FailAfter(0)
	_, e := c.mkCellWithReceipt(15000, nil, nil, nil, nil, &old)
	if e == nil {
		t.Fatal("actual old factory refusal missing")
	}
	c.integer.Budget().FailAfter(-1)
	// The explicit owner consumes this exact old pending placement. A new
	// actual nil-receipt factory invocation must not inherit its authority.
	serviceMust(t, c.cleanupPendingCreation())
	serviceMust(t, c.monomials.allocator.CleanupPending())
	c.integer.Budget().FailAfter(0)
	_, e = c.mkCell(15000, nil, nil, nil, nil)
	if e == nil {
		t.Fatal("actual replacement prefix refusal missing")
	}
	before := c.pending
	roots, x := c.monomials.allocator.PendingRoots()
	serviceMust(t, x)
	if before.generation == old.invocation {
		t.Fatal("actual replacement reused old invocation")
	}
	polynomialImpPhysicalAuditNamed(t, m, nil, nil, 0)
	c.integer.Budget().FailAfter(-1)
	serviceMust(t, c.cleanupFactoryInvocation(&old, true))
	after, x := c.monomials.allocator.PendingRoots()
	serviceMust(t, x)
	if c.pending != before || after != roots {
		t.Fatal("old receipt consumed new actual placement or allocator continuation")
	}
	polynomialImpPhysicalAuditNamed(t, m, nil, nil, 0)
}
