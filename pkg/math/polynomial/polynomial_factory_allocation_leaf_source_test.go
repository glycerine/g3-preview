// Source: src/math/polynomial/polynomial.cpp:1392-1423,2481-2495;
// src/util/small_object_allocator.cpp:104-145.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialFactoryAllocationActualLeaf(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	rt, e := util.NewRuntime(util.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	q, e := util.NewRationalManager(util.RationalConfig{Integers: util.IntegerConfig{Memory: rt, Cells: arena.Config{PageSize: 8, Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
	serviceMust(t, e)
	mm, e := NewMonomialManager(MonomialManagerConfig{Budget: b, Memory: rt, Profile: util.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	serviceMust(t, e)
	m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Monomials: mm, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
	serviceMust(t, e)
	c := m.core
	pad := int64(99996) - rt.State().LocalSize - 8
	if pad <= 0 {
		t.Fatal("actual setup exceeds sync frontier", rt.State())
	}
	held, e := rt.Allocate(uint64(pad))
	serviceMust(t, e)
	var receipt polynomialFactoryInvocationReceipt
	defer func() {
		serviceMust(t, rt.SetMaxSize(0))
		serviceMust(t, c.cleanupFactoryInvocation(&receipt, true))
		serviceMust(t, c.CleanupConstruction())
		serviceMust(t, m.CloseChecked())
		serviceMust(t, q.CloseChecked())
		serviceMust(t, rt.Deallocate(&held))
		serviceMust(t, rt.Close())
		if b.Used() != 0 {
			t.Fatal("named graph retained", b.Used())
		}
	}()
	serviceMust(t, rt.SetMaxSize(1))
	_, cause := c.mkCellWithReceipt(0, nil, nil, nil, nil, &receipt)
	if !numericPhysicalReservationRefusal(cause) {
		t.Fatal("actual source malloc stop not reached", cause)
	}
	roots, e := mm.allocator.PendingRoots()
	serviceMust(t, e)
	polynomialImpPhysicalAuditNamedExtra(t, m, nil, nil, 0, []util.Block{held})
	s := receipt.frame
	if roots[0] == (util.Block{}) || !s.allocationStarted || s.allocation.Pending != roots[0] || s.allocator != roots || mm.allocator.AllocationOutcomeInvocation(&s.allocation) == 0 || s.executing {
		t.Fatal("actual returned allocation outcome not retained", roots, s)
	}
	if s.allocationCause != cause {
		t.Fatal("actual lower source cause changed")
	}
	// Rejected lower allocation retains the prior embedded outcome verbatim.
	oldOutcome := s.allocation
	_, started, rejected := mm.allocator.AllocateWithOutcomeStarted(12, &s.allocation)
	if started || rejected != util.ErrAllocatorPending || s.allocation != oldOutcome {
		t.Fatal("rejected lower entry changed outcome", started, rejected)
	}
	before := rt.Stats()
	serviceMust(t, c.cleanupFactoryInvocation(&receipt, false))
	copiedOwner := *c
	copiedFrame := *s
	invalid := []polynomialFactoryInvocationReceipt{receipt, receipt, receipt}
	invalid[0].manager = &copiedOwner
	invalid[1].frame = &copiedFrame
	invalid[2].invocation++
	for i := range invalid {
		serviceMust(t, c.cleanupFactoryInvocation(&invalid[i], true))
	}
	if rt.Stats() != before {
		t.Fatal("false/copied/stale cleanup altered pending leaf")
	}
	serviceMust(t, c.cleanupFactoryInvocation(&receipt, true))
	polynomialImpPhysicalAuditNamedExtra(t, m, nil, nil, 0, []util.Block{held})
	if s.allocationCause != cause || s.allocation.Pending != (util.Block{}) {
		t.Fatal("own acknowledgment lost first cause or consumed-root frontier", s)
	}
	if mm.allocator.Stats().PendingBlocks != 0 {
		t.Fatal("exact own leaf retained")
	}
	// A later real nil-receipt factory acquisition must survive the old receipt.
	pad2 := int64(99996) - rt.State().LocalSize - 8
	serviceMust(t, rt.SetMaxSize(0))
	held2, e := rt.Allocate(uint64(pad2))
	serviceMust(t, e)
	defer func() { serviceMust(t, rt.Deallocate(&held2)) }()
	serviceMust(t, rt.SetMaxSize(1))
	_, foreignCause := c.mkCell(0, nil, nil, nil, nil)
	foreignRoots, e := mm.allocator.PendingRoots()
	serviceMust(t, e)
	if !numericPhysicalReservationRefusal(foreignCause) || foreignRoots[0] == (util.Block{}) {
		t.Fatal("later foreign real leaf not reached", foreignCause, foreignRoots)
	}
	polynomialImpPhysicalAuditNamedExtra(t, m, nil, nil, 0, []util.Block{held, held2})
	serviceMust(t, c.cleanupFactoryInvocation(&receipt, true))
	after, e := mm.allocator.PendingRoots()
	serviceMust(t, e)
	if after != foreignRoots || s.allocationCause != cause {
		t.Fatal("old own receipt consumed later foreign allocation", after)
	}
	serviceMust(t, c.CleanupConstruction())
	serviceMust(t, mm.allocator.CleanupPending())
	polynomialImpPhysicalAuditNamedExtra(t, m, nil, nil, 0, []util.Block{held, held2})
}

func TestPolynomialFactoryAllocationRealCallbackForeignStart(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	var old polynomialFactoryInvocationReceipt
	p, e := c.mkCellWithReceipt(0, nil, nil, nil, nil, &old)
	serviceMust(t, e)
	serviceMust(t, c.IncRef(p))
	serviceMust(t, c.transferFactoryResult(&old, p))
	calls := 0
	h, e := NewPolynomialDeletionHandler(c.integer.Budget(), func(_ *PolynomialDeletionHandler, c *MultivariatePolynomialCore, deleted Polynomial) error {
		if deleted != p {
			return nil
		}
		calls++
		if c.callbackDepth == 0 || c.factoryAllocationOutcome(&old) != nil {
			t.Error("actual external callback acquired older factory outcome")
		}
		c.integer.Budget().FailAfter(0)
		_, cause := c.mkCell(15000, nil, nil, nil, nil)
		if !numericPhysicalReservationRefusal(cause) {
			t.Error("real foreign callback refusal absent", cause)
		}
		pending := c.pending
		scope := c.factoryInvocation
		serviceMust(t, c.cleanupFactoryInvocation(&old, true))
		if c.pending != pending || c.factoryInvocation != scope {
			t.Error("old receipt drained callback-owned constructor")
		}
		return cause
	})
	serviceMust(t, e)
	serviceMust(t, c.AddDeletionHandler(h))
	cause := c.DecRef(p)
	if !numericPhysicalReservationRefusal(cause) || calls != 1 {
		t.Fatal("actual callback route", cause, calls)
	}
	polynomialImpPhysicalAuditNamedRaw(t, m, []Polynomial{p}, nil, h.header, nil)
	c.integer.Budget().FailAfter(-1)
	serviceMust(t, c.CleanupConstruction())
	serviceMust(t, c.CleanupValue(&p))
	if calls != 1 {
		t.Fatal("own retry replayed callback")
	}
	serviceMust(t, c.RemoveDeletionHandler(h))
	serviceMust(t, h.CloseChecked())
}

func TestPolynomialFactoryAllocationConcreteFrameGuards(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	var receipt polynomialFactoryInvocationReceipt
	p, e := c.mkCellWithReceipt(0, nil, nil, nil, nil, &receipt)
	serviceMust(t, e)
	s := receipt.frame
	if !s.allocationStarted || s.allocation.Object == (util.ObjectRef{}) || s.allocation.Pending != (util.Block{}) || s.executing {
		t.Fatal("real completed malloc outcome", s)
	}
	if c.factoryAllocationOutcome(&receipt) != nil {
		t.Fatal("completed body reacquired output authority")
	}
	copied := *c
	copiedFrame := *s
	invalid := receipt
	invalid.manager = &copied
	if copied.matchesFactoryInvocation(&invalid) || copied.factoryAllocationOutcome(&invalid) != nil {
		t.Fatal("copied owner authority")
	}
	invalid = receipt
	invalid.frame = &copiedFrame
	if c.matchesFactoryInvocation(&invalid) || c.factoryAllocationOutcome(&invalid) != nil {
		t.Fatal("copied frame authority")
	}
	invalid = receipt
	invalid.invocation++
	if c.factoryAllocationOutcome(&invalid) != nil {
		t.Fatal("stale nonce authority")
	}
	serviceMust(t, c.cleanupFactoryInvocation(&receipt, true))
	_ = p
}

// The signed-host accounting boundary is a guarded helper profile, not an
// ordinary mathematical OOM path. Runtime Bounds does not invoke Transport.
func TestPolynomialFactoryAllocationPostphysicalOwnRetry(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	transports := 0
	rt, e := util.NewRuntime(util.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64, Transport: func(f util.Failure) error { transports++; return f }})
	serviceMust(t, e)
	q, e := util.NewRationalManager(util.RationalConfig{Integers: util.IntegerConfig{Memory: rt, Cells: arena.Config{PageSize: 8, Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
	serviceMust(t, e)
	mm, e := NewMonomialManager(MonomialManagerConfig{Budget: b, Memory: rt, Profile: util.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	serviceMust(t, e)
	m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Monomials: mm, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
	serviceMust(t, e)
	c := m.core
	pad := int64(99996) - rt.State().LocalSize - 8
	held, e := rt.Allocate(uint64(pad))
	serviceMust(t, e)
	serviceMust(t, rt.SetMaxSize(1))
	var receipt polynomialFactoryInvocationReceipt
	_, cause := c.mkCellWithReceipt(15000, nil, nil, nil, nil, &receipt)
	if !numericPhysicalReservationRefusal(cause) || receipt.frame.allocation.Pending == (util.Block{}) {
		t.Fatal("actual large malloc pending absent", cause)
	}
	polynomialImpPhysicalAuditNamedExtra(t, m, nil, nil, 0, []util.Block{held})
	serviceMust(t, rt.SetMaxSize(0))
	contextTLSNearMinimum(t, rt)
	before := rt.Stats().LiveBlocks
	beforeTransport := transports
	e = c.cleanupFactoryInvocation(&receipt, true)
	if !errors.Is(e, arena.ErrBounds) || rt.Stats().LiveBlocks+1 != before || receipt.frame.allocator != ([3]util.Block{}) || receipt.frame.allocation.Pending != (util.Block{}) || receipt.frame.allocationCause != cause || transports != beforeTransport {
		t.Fatal("actual postphysical ack/history/transport", e, rt.State(), receipt.frame)
	}
	polynomialImpPhysicalAuditNamedExtra(t, m, nil, nil, 0, []util.Block{held})
	state := rt.State()
	serviceMust(t, c.cleanupFactoryInvocation(&receipt, true))
	if rt.State() != state || receipt.frame.allocationCause != cause {
		t.Fatal("consumed own retry replayed or lost first cause")
	}
	contextTLSFinish(t, rt, m.CloseChecked)
	contextTLSFinish(t, rt, q.CloseChecked)
	contextTLSFinish(t, rt, mm.CloseChecked)
	contextTLSFinish(t, rt, func() error {
		if held == (util.Block{}) {
			return nil
		}
		return rt.Deallocate(&held)
	})
	contextTLSFinish(t, rt, rt.Close)
	if b.Used() != 0 {
		t.Fatal("exact named graph retained", b.Used())
	}
}
