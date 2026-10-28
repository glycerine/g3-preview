package polynomial

// Source: src/math/polynomial/polynomial.cpp:852-869;
// src/util/small_object_allocator.cpp:97-144; src/util/memory_manager.cpp.

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestMonomialFactoryAllocatorActualPhysicalPrefix(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	m, e := NewMonomialManager(MonomialManagerConfig{Budget: b, Memory: rt, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	serviceMust(t, e)
	_, e = m.MkVar()
	serviceMust(t, e)
	// Real direct allocation remains the source profile under test; the held
	// caller block brings the next object malloc to the actual TLS sync frontier.
	pad := int64(99980) - rt.State().LocalSize - 8
	if pad <= 0 {
		t.Fatal("selected genuine source prefix unavailable", rt.State())
	}
	held, e := rt.Allocate(uint64(pad))
	serviceMust(t, e)
	var outcome monomialFactoryOutcome
	defer func() {
		serviceMust(t, rt.SetMaxSize(0))
		serviceMust(t, m.CleanupCreation())

		serviceMust(t, m.CloseChecked())
		serviceMust(t, rt.Deallocate(&held))
		serviceMust(t, rt.Close())
		if b.Used() != 0 {
			t.Fatal("named graph retained", b.Used())
		}
	}()
	serviceMust(t, rt.SetMaxSize(1))
	_, cause := m.mkMonomialCreated([]MonomialPower{{0, 2}}, &outcome)
	if !numericPhysicalReservationRefusal(cause) || !outcome.started || outcome.allocator[0] == (umemory.Block{}) {
		t.Fatal("actual object malloc physical failure not reached", cause, outcome, m.creation)
	}
	monomialPhysicalAudit(t, m, held)
	current, e := m.allocator.PendingRoots()
	serviceMust(t, e)
	if current != outcome.allocator || m.creation.stage != 2 || m.locations[m.creation.slot-1].raw != (umemory.ObjectRef{}) {
		t.Fatal("exact unpublished allocation prefix", current, outcome, m.creation)
	}
	// A source/ordinary retry cannot silently consume the original unjoined root.
	before := rt.Stats()
	var rejected monomialFactoryOutcome
	_, e = m.mkMonomialCreated([]MonomialPower{{0, 3}}, &rejected)
	if e == nil || rt.Stats() != before || rejected.started {
		t.Fatal("ordinary retry altered the actual older acquisition", e, rejected)
	}

	receipt := outcome.allocatorInvocation
	serviceMust(t, m.cleanupMonomialAllocatorInvocation(&receipt, false))
	copiedOwner := *m
	copiedFrame := *receipt.frame
	copiedOutcome := *receipt.outcome
	invalid := []monomialAllocatorInvocationReceipt{receipt, receipt, receipt, receipt}
	invalid[0].manager = &copiedOwner
	invalid[1].frame = &copiedFrame
	invalid[2].outcome = &copiedOutcome
	invalid[3].invocation++
	for i := range invalid {
		serviceMust(t, m.cleanupMonomialAllocatorInvocation(&invalid[i], true))
	}
	now, e := m.allocator.PendingRoots()
	serviceMust(t, e)
	if now != current || m.allocatorInvocation.sourceCause != cause {
		t.Fatal("nonacquired/false receipt changed real source history", now, cause)
	}
	monomialPhysicalAudit(t, m, held)
	serviceMust(t, m.cleanupMonomialAllocatorInvocation(&outcome.allocatorInvocation, true))
	if m.allocator.Stats().PendingBlocks != 0 {
		t.Fatal("exact own leaf helper did not drain its acquired pending token")
	}
	monomialPhysicalAudit(t, m, held)
}

func TestMonomialFactoryAllocatorRejectedNewCreationPreservesEarlierLeaf(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	m, e := NewMonomialManager(MonomialManagerConfig{Budget: b, Memory: rt, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	serviceMust(t, e)
	_, e = m.MkVar()
	serviceMust(t, e)
	// Real direct allocation remains the source profile under test; the held
	// caller block brings the next object malloc to the actual TLS sync frontier.
	pad := int64(99980) - rt.State().LocalSize - 8
	if pad <= 0 {
		t.Fatal("selected genuine source prefix unavailable", rt.State())
	}
	held, e := rt.Allocate(uint64(pad))
	serviceMust(t, e)
	var outcome monomialFactoryOutcome
	defer func() {
		serviceMust(t, rt.SetMaxSize(0))
		serviceMust(t, m.CleanupCreation())

		serviceMust(t, m.CloseChecked())
		serviceMust(t, rt.Deallocate(&held))
		serviceMust(t, rt.Close())
		if b.Used() != 0 {
			t.Fatal("named graph retained", b.Used())
		}
	}()
	serviceMust(t, rt.SetMaxSize(1))
	_, cause := m.mkMonomialCreated([]MonomialPower{{0, 2}}, &outcome)
	if !numericPhysicalReservationRefusal(cause) || !outcome.started || outcome.allocator[0] == (umemory.Block{}) {
		t.Fatal("actual object malloc physical failure not reached", cause, outcome, m.creation)
	}
	monomialPhysicalAudit(t, m, held)
	current, e := m.allocator.PendingRoots()
	serviceMust(t, e)
	if current != outcome.allocator || m.creation.stage != 2 || m.locations[m.creation.slot-1].raw != (umemory.ObjectRef{}) {
		t.Fatal("exact unpublished allocation prefix", current, outcome, m.creation)
	}

	receipt := outcome.allocatorInvocation
	frameBefore := m.allocatorInvocation
	serviceMust(t, m.CleanupCreation()) // remove only the actual temporary source binding
	before := rt.Stats()
	var rejected monomialFactoryOutcome
	_, e = m.mkMonomialCreated([]MonomialPower{{0, 3}}, &rejected)
	if e != umemory.ErrAllocatorPending || !rejected.started || rejected.allocatorInvocation.invocation != 0 || rejected.allocator != ([3]umemory.Block{}) || m.allocatorInvocation != frameBefore || rt.Stats() != before {
		t.Fatal("new source creation acquired older rejected leaf", e, rejected)
	}
	monomialPhysicalAudit(t, m, held)
	serviceMust(t, m.cleanupMonomialAllocatorInvocation(&receipt, true))
	monomialPhysicalAudit(t, m, held)
	if m.creation.stage != 2 || m.allocator.Stats().PendingBlocks != 0 {
		t.Fatal("exact older abort touched newer source binding", m.creation)
	}
	serviceMust(t, m.CleanupCreation())
	serviceMust(t, rt.SetMaxSize(0))
	var newer monomialFactoryOutcome
	value, e := m.mkMonomialCreated([]MonomialPower{{0, 4}}, &newer)
	serviceMust(t, e)
	if newer.allocatorInvocation.invocation == 0 || newer.allocatorInvocation.invocation == receipt.invocation {
		t.Fatal("actual retry lacked guarded new nonce", newer)
	}
	after := rt.Stats()
	serviceMust(t, m.cleanupMonomialAllocatorInvocation(&receipt, true))
	if rt.Stats() != after || !m.matchesMonomialAllocatorInvocation(&newer.allocatorInvocation) {
		t.Fatal("stale abort altered genuine later allocation")
	}
	monomialPhysicalAudit(t, m, held)
	serviceMust(t, m.CleanupValue(&value))
}

func TestPolynomialDerivativeOwnsActualMonomialAllocatorLeaf(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	q, e := unumeral.NewRationalManager(unumeral.RationalConfig{Integers: unumeral.IntegerConfig{Memory: rt, Cells: arena.Config{PageSize: 8, Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
	serviceMust(t, e)
	mm, e := NewMonomialManager(MonomialManagerConfig{Budget: b, Memory: rt, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	serviceMust(t, e)
	m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Monomials: mm, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
	serviceMust(t, e)
	_, e = mm.MkVar()
	serviceMust(t, e)
	term, e := mm.MkVariable(0, 2)
	serviceMust(t, e)
	p, e := m.core.MkTerms([]unumeral.Int{unumeral.NewInt(3)}, []Monomial{term})
	serviceMust(t, e)
	serviceMust(t, m.core.IncRef(p))
	pad := int64(99980) - rt.State().LocalSize - 8
	if pad <= 0 {
		t.Fatal("actual setup exceeds selected sync frontier", rt.State())
	}
	held, e := rt.Allocate(uint64(pad))
	serviceMust(t, e)
	defer func() {
		serviceMust(t, rt.SetMaxSize(0))
		serviceMust(t, mm.CleanupCreation())
		serviceMust(t, m.CleanupDerivative())
		serviceMust(t, m.core.CleanupValue(&p))
		serviceMust(t, m.CloseChecked())
		serviceMust(t, q.CloseChecked())
		serviceMust(t, rt.Deallocate(&held))
		serviceMust(t, rt.Close())
		if b.Used() != 0 {
			t.Fatal("exact fixture graph", b.Used())
		}
	}()
	serviceMust(t, rt.SetMaxSize(1))
	var receipt polynomialDerivativeInvocationReceipt
	_, cause := m.derivativeWithReceipt(p, 0, &receipt)
	roots, e := mm.allocator.PendingRoots()
	serviceMust(t, e)
	if !numericPhysicalReservationRefusal(cause) || roots[0] == (umemory.Block{}) || mm.creation.stage != 2 {
		t.Fatal("actual monomial malloc leaf not reached", cause, roots, mm.creation)
	}
	polynomialImpPhysicalAuditNamedExtra(t, m, []Polynomial{p}, nil, 0, []umemory.Block{held})
	serviceMust(t, m.cleanupDerivativeInvocation(&receipt, false))
	if after, e := mm.allocator.PendingRoots(); e != nil || after != roots {
		t.Fatal("completed source NUM phase consumed unjoined allocation", after, e)
	}
	serviceMust(t, m.cleanupDerivativeInvocation(&receipt, true))
	if mm.allocator.Stats().PendingBlocks != 0 {
		t.Fatal("matched derivative abort abandoned actual monomial allocation leaf")
	}
	polynomialImpPhysicalAuditNamedExtra(t, m, []Polynomial{p}, nil, 0, []umemory.Block{held})
}

func TestPolynomialDerivativeMonomialAllocatorForeignReplacementPreserved(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	q, e := unumeral.NewRationalManager(unumeral.RationalConfig{Integers: unumeral.IntegerConfig{Memory: rt, Cells: arena.Config{PageSize: 8, Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}}})
	serviceMust(t, e)
	mm, e := NewMonomialManager(MonomialManagerConfig{Budget: b, Memory: rt, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	serviceMust(t, e)
	m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Monomials: mm, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
	serviceMust(t, e)
	_, e = mm.MkVar()
	serviceMust(t, e)
	term, e := mm.MkVariable(0, 2)
	serviceMust(t, e)
	p, e := m.core.MkTerms([]unumeral.Int{unumeral.NewInt(3)}, []Monomial{term})
	serviceMust(t, e)
	serviceMust(t, m.core.IncRef(p))
	pad := int64(99980) - rt.State().LocalSize - 8
	if pad <= 0 {
		t.Fatal("actual setup exceeds selected sync frontier", rt.State())
	}
	held, e := rt.Allocate(uint64(pad))
	serviceMust(t, e)
	var foreignHeld umemory.Block
	defer func() {
		serviceMust(t, rt.SetMaxSize(0))
		serviceMust(t, mm.CleanupCreation())
		serviceMust(t, m.CleanupDerivative())
		serviceMust(t, m.core.CleanupValue(&p))
		serviceMust(t, m.CloseChecked())
		serviceMust(t, q.CloseChecked())
		serviceMust(t, rt.Deallocate(&foreignHeld))
		serviceMust(t, rt.Deallocate(&held))
		serviceMust(t, rt.Close())
		if b.Used() != 0 {
			t.Fatal("exact fixture graph", b.Used())
		}
	}()
	serviceMust(t, rt.SetMaxSize(1))
	var receipt polynomialDerivativeInvocationReceipt
	_, cause := m.derivativeWithReceipt(p, 0, &receipt)
	roots, e := mm.allocator.PendingRoots()
	serviceMust(t, e)
	if !numericPhysicalReservationRefusal(cause) || roots[0] == (umemory.Block{}) || mm.creation.stage != 2 {
		t.Fatal("actual monomial malloc leaf not reached", cause, roots, mm.creation)
	}
	polynomialImpPhysicalAuditNamedExtra(t, m, []Polynomial{p}, nil, 0, []umemory.Block{held})
	serviceMust(t, m.cleanupDerivativeInvocation(&receipt, false))
	if after, e := mm.allocator.PendingRoots(); e != nil || after != roots {
		t.Fatal("completed source NUM phase consumed unjoined allocation", after, e)
	}

	oldAllocator := m.derivative.monomialAllocator
	serviceMust(t, mm.CleanupCreation())
	serviceMust(t, mm.cleanupMonomialAllocatorInvocation(&oldAllocator, true))
	pad = int64(99980) - rt.State().LocalSize - 8
	if pad <= 0 {
		t.Fatal("actual next invocation sync boundary", rt.State())
	}
	foreignHeld, e = rt.Allocate(uint64(pad))
	serviceMust(t, e)
	var foreign monomialFactoryOutcome
	_, foreignCause := mm.mkMonomialCreated([]MonomialPower{{0, 3}}, &foreign)
	if !numericPhysicalReservationRefusal(foreignCause) || foreign.allocatorInvocation.invocation == 0 || foreign.allocatorInvocation.invocation == oldAllocator.invocation {
		t.Fatal("actual foreign replacement not reached", foreignCause, foreign)
	}
	foreignRoots, e := mm.allocator.PendingRoots()
	serviceMust(t, e)
	if foreignRoots == roots || foreignRoots[0] == (umemory.Block{}) {
		t.Fatal("new actual pending identity", foreignRoots)
	}
	polynomialImpPhysicalAuditNamedExtra(t, m, []Polynomial{p}, nil, 0, []umemory.Block{held, foreignHeld})
	serviceMust(t, m.cleanupDerivativeInvocation(&receipt, true))
	after, e := mm.allocator.PendingRoots()
	serviceMust(t, e)
	if after != foreignRoots || !mm.matchesMonomialAllocatorInvocation(&foreign.allocatorInvocation) || mm.creation.generation != foreign.generation || m.derivative.bodyCause != cause {
		t.Fatal("old parent abort swept foreign acquisition/history", after, mm.creation)
	}
	polynomialImpPhysicalAuditNamedExtra(t, m, []Polynomial{p}, nil, 0, []umemory.Block{held, foreignHeld})
	serviceMust(t, mm.CleanupCreation())
	serviceMust(t, mm.cleanupMonomialAllocatorInvocation(&foreign.allocatorInvocation, true))
	polynomialImpPhysicalAuditNamedExtra(t, m, []Polynomial{p}, nil, 0, []umemory.Block{held, foreignHeld})
}

// This public Runtime signed-accounting profile proves the owning leaf's
// postphysical acknowledgement. It is not an ordinary mathematical OOM path.
func TestMonomialAllocatorOwnAcknowledgesPostphysicalRelease(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	m, e := NewMonomialManager(MonomialManagerConfig{Budget: b, Memory: rt, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	serviceMust(t, e)
	powers := make([]MonomialPower, 15000)
	for i := range powers {
		powers[i] = MonomialPower{uint32(i), 1}
	}
	serviceMust(t, m.tempInit(&m.temps[0], powers))
	pad := int64(99980) - rt.State().LocalSize - 8
	if pad <= 0 {
		t.Fatal("selected source allocation frontier", rt.State())
	}
	held, e := rt.Allocate(uint64(pad))
	serviceMust(t, e)
	serviceMust(t, rt.SetMaxSize(1))
	var outcome monomialFactoryOutcome
	_, cause := m.internTempCreated(0, &outcome)
	if !numericPhysicalReservationRefusal(cause) || outcome.allocator[0] == (umemory.Block{}) {
		t.Fatal("actual large object pending", cause, outcome)
	}
	monomialPhysicalAudit(t, m, held)
	serviceMust(t, rt.SetMaxSize(0))
	contextTLSNearMinimum(t, rt)
	before := rt.Stats().LiveBlocks
	e = m.cleanupMonomialAllocatorInvocation(&outcome.allocatorInvocation, true)
	if !errors.Is(e, arena.ErrBounds) || rt.Stats().LiveBlocks+1 != before || m.allocatorInvocation.outcome.Pending != (umemory.Block{}) || m.allocatorInvocation.sourceCause != cause {
		t.Fatal("actual physical consume followed by signed guard", e, rt.State())
	}
	monomialPhysicalAudit(t, m, held)
	after := rt.State()
	serviceMust(t, m.cleanupMonomialAllocatorInvocation(&outcome.allocatorInvocation, true))
	if rt.State() != after {
		t.Fatal("own retry replayed consumed physical token")
	}
	serviceMust(t, m.CleanupCreation())
	contextTLSFinish(t, rt, m.CloseChecked)
	contextTLSFinish(t, rt, func() error {
		if held == (umemory.Block{}) {
			return nil
		}
		return rt.Deallocate(&held)
	})
	contextTLSFinish(t, rt, rt.Close)
	if b.Used() != 0 {
		t.Fatal("named final graph", b.Used())
	}
}
