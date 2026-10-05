// Source: src/math/polynomial/polynomial.cpp:3318-3338;
// src/util/mpz.cpp:big_set; src/util/memory_manager.cpp:allocate.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestPolynomialConstantCoefficientReceiptActualSETLateTLSLeaf(t *testing.T) {
	b := arena.NewBudget(1 << 27)
	runtime, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	cfg := serviceConfig(b).Integers
	cfg.Memory = runtime
	q, e := NewRationalManager(RationalConfig{Integers: cfg})
	serviceMust(t, e)
	m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.z, Rationals: q, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
	serviceMust(t, e)
	c := m.core
	_, e = c.monomials.MkVar()
	serviceMust(t, e)
	large := Int{}
	limbs := make([]uint32, 32768)
	limbs[0] = 17
	limbs[len(limbs)-1] = 1
	serviceMust(t, c.integer.SetLimbs(&large, limbs))
	p, e := c.MkConstantTakeCopy(&large)
	serviceMust(t, e)
	serviceMust(t, c.IncRef(p))
	out := NewInt(19)
	var receipt polynomialConstantCoefficientInvocationReceipt
	serviceMust(t, runtime.SetMaxSize(1))
	_, e = c.constantCoefficientWithReceipt(p, 0, 0, &out, &receipt)
	if e == nil || !runtime.State().OutOfMemory || !c.matchesConstantCoefficientInvocation(&receipt) || !c.integer.matchesCopyAllocationInvocation(&receipt.frame.copy) {
		t.Fatal("real SET late TLS acquisition", e)
	}
	cell := receipt.frame.copy.frame
	roots, e := c.integer.allocator.PendingRoots()
	serviceMust(t, e)
	if roots[0] == (memory.Block{}) || cell.roots.allocator[0] != roots[0] || cell.roots.cell.object != (memory.ObjectRef{}) {
		t.Fatal("actual raw leaf before object publication", roots, cell.roots)
	}
	polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p}, []*Int{&large, &out}, 0)
	before, copyBefore, used, state, stats := *receipt.frame, *cell, b.Used(), runtime.State(), runtime.Stats()
	rejected := receipt
	_, e = c.constantCoefficientWithReceipt(p, 0, 0, &out, &rejected)
	if e != ErrPolynomialPending || rejected != receipt || *receipt.frame != before || *cell != copyBefore || b.Used() != used || runtime.State() != state || runtime.Stats() != stats {
		t.Fatal("rejected source entry changed leaf ownership", e)
	}
	if e := c.CloseChecked(); e != ErrPolynomialPending {
		t.Fatal("owning close must preserve concrete SET child", e)
	}
	serviceMust(t, c.cleanupConstantCoefficientInvocation(&receipt, false))
	if b.Used() != used || runtime.State() != state || runtime.Stats() != stats {
		t.Fatal("no source local destructor may drain retained SET allocation")
	}
	serviceMust(t, runtime.SetMaxSize(0))
	serviceMust(t, c.integer.Del(&out))
	serviceMust(t, c.cleanupConstantCoefficientInvocation(&receipt, true))
	polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p}, []*Int{&large, &out}, 0)
	serviceMust(t, c.DecRef(p))
	serviceMust(t, c.integer.Del(&large))
	serviceMust(t, m.CloseChecked())
	serviceMust(t, q.CloseChecked())
	serviceMust(t, runtime.Close())
	if b.Used() != 0 {
		t.Fatal("actual declared owners retained charge", b.Used())
	}
}
