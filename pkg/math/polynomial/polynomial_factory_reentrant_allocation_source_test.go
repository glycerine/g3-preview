// Source: src/math/polynomial/polynomial.cpp:1392-1423,2481-2495;
// src/util/memory_manager.cpp:274-285.
package polynomial

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// ErrorTransport is a supported Go callback profile. This test does not claim
// a native callback at the original allocation exception site.
func TestPolynomialFactoryAllocationTransportCannotRetireExecutingReceipt(t *testing.T) {
	for _, mode := range []string{"receipt", "construction", "manager"} {
		t.Run(mode, func(t *testing.T) { polynomialFactoryTransportCleanupMode(t, mode) })
	}
}

func polynomialFactoryTransportCleanupMode(t *testing.T, mode string) {
	b := arena.NewBudget(1 << 26)
	var c *MultivariatePolynomialCore
	var m *MultivariatePolynomialManager
	var receipt polynomialFactoryInvocationReceipt
	active := false
	callbacks := 0
	var callbackCleanup error
	rt, e := util.NewRuntime(util.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64, Transport: func(f util.Failure) error {
		if active {
			callbacks++
			before := c.pending
			stats := c.memory.Stats()
			used := b.Used()
			closing, stage := m.closing, m.cleanupStage
			switch mode {
			case "receipt":
				callbackCleanup = c.cleanupFactoryInvocation(&receipt, true)
			case "construction":
				callbackCleanup = c.CleanupConstruction()
			case "manager":
				callbackCleanup = m.CloseChecked()
			}
			if c.memory.Stats() != stats || b.Used() != used || m.closing != closing || m.cleanupStage != stage {
				t.Error("callback touched older owner storage/stage")
			}
			if c.pending != before {
				t.Error("callback changed executing pending construction", before, c.pending)
			}
		}
		return f
	}})
	serviceMust(t, e)
	q, e := util.NewRationalManager(util.RationalConfig{Integers: util.IntegerConfig{Memory: rt, Cells: arena.Config{PageSize: 8, Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
	serviceMust(t, e)
	mm, e := NewMonomialManager(MonomialManagerConfig{Budget: b, Memory: rt, Profile: util.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	serviceMust(t, e)
	m, e = NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Monomials: mm, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
	serviceMust(t, e)
	c = m.core
	pad := int64(99996) - rt.State().LocalSize - 8
	if pad <= 0 {
		t.Fatal("source sync frontier not reached", rt.State())
	}
	held, e := rt.Allocate(uint64(pad))
	serviceMust(t, e)
	serviceMust(t, rt.SetMaxSize(1))
	active = true
	_, cause := c.mkCellWithReceipt(0, nil, nil, nil, nil, &receipt)
	active = false
	if !numericPhysicalReservationRefusal(cause) || callbacks != 1 {
		t.Fatal("actual postphysical callback absent", cause, callbacks)
	}
	roots, e := mm.allocator.PendingRoots()
	serviceMust(t, e)
	if !c.matchesFactoryInvocation(&receipt) || receipt.frame.allocator != roots || roots[0] == (util.Block{}) || receipt.frame.allocation.Pending != roots[0] {
		t.Fatalf("callback retired executing receipt: active=%v roots=%v scope=%+v callback=%v body=%v", c.matchesFactoryInvocation(&receipt), roots, receipt.frame, callbackCleanup, cause)
	}
	if callbackCleanup != ErrPolynomialPending {
		t.Fatal("executing own cleanup must refuse", callbackCleanup)
	}
	if receipt.frame.allocationCause != cause || receipt.frame.executing {
		t.Fatal("actual stopped outcome history changed", receipt.frame)
	}
	polynomialImpPhysicalAuditNamedExtra(t, m, nil, nil, 0, []util.Block{held})
	serviceMust(t, c.cleanupFactoryInvocation(&receipt, false))
	if receipt.frame.allocation.Pending != roots[0] {
		t.Fatal("source-only cleanup consumed unjoined malloc")
	}
	serviceMust(t, c.cleanupFactoryInvocation(&receipt, true))
	if receipt.frame.allocationCause != cause || receipt.frame.allocation.Pending != (util.Block{}) {
		t.Fatal("owning acknowledgment lost cause", receipt.frame)
	}
	polynomialImpPhysicalAuditNamedExtra(t, m, nil, nil, 0, []util.Block{held})
	serviceMust(t, rt.SetMaxSize(0))
	old := receipt
	p, e := c.mkCellWithReceipt(0, nil, nil, nil, nil, &receipt)
	serviceMust(t, e)
	current := c.factoryInvocation
	serviceMust(t, c.cleanupFactoryInvocation(&old, true))
	if c.factoryInvocation != current {
		t.Fatal("old invocation acquired new factory authority")
	}
	serviceMust(t, c.cleanupFactoryInvocation(&receipt, true))
	_ = p
	serviceMust(t, m.CloseChecked())
	serviceMust(t, q.CloseChecked())
	serviceMust(t, mm.CloseChecked())
	serviceMust(t, rt.Deallocate(&held))
	serviceMust(t, rt.Close())
	if b.Used() != 0 {
		t.Fatal("named graph retained", b.Used())
	}
}
