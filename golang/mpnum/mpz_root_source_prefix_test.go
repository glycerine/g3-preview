// Source: src/util/mpz.h:836-837; src/util/mpz.cpp:1942-2012;
// src/util/mpzzp.h:279-280.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestIntegerRootSourceCopyPublishesCallerBeforeBisection(t *testing.T) {
	m, b := mpzPublicationDirectOwner(t)
	a := Int{}
	serviceMust(t, m.Copy(&a, &m.two64))
	defer func() { serviceMust(t, m.Del(&a)) }()
	out := NewInt(7)
	defer func() { serviceMust(t, m.Del(&out)) }()
	b.FailAfter(0)
	_, e := m.Root(&a, 2, &out)
	integerPhysicalAudit(t, m, &a, &out)
	b.FailAfter(-1)
	if !numericPhysicalReservationRefusal(e) || out.cell != m.failedNullRef || out.val != 1 {
		t.Fatal("source SET(output,input) refusal must publish the real caller", e, out)
	}
}

func TestIntegerRootInPlaceRetainsActualRAWOnBodyFailure(t *testing.T) {
	m, b := mpzPublicationDirectOwner(t)
	a := Int{}
	serviceMust(t, m.Copy(&a, &m.two64))
	defer func() { serviceMust(t, m.Del(&a)) }()
	before := a
	b.FailAfter(0)
	_, e := m.RootInPlace(&a, 2)
	integerPhysicalAudit(t, m, &a)
	b.FailAfter(-1)
	if !numericPhysicalReservationRefusal(e) || !m.rootRaw.active || m.rootRaw.endEntered || m.rootRaw.values[0].cell != m.failedNullRef || a != before {
		t.Fatal("source initial POWER retained RAW target", e, m.rootRaw, a)
	}
	generation := m.rootRaw.generation
	_, again := m.RootInPlace(&a, 2)
	if !errors.Is(again, ErrRootPending) || m.rootRaw.generation != generation {
		t.Fatal("later call erased older RAW invocation", again)
	}
	serviceMust(t, m.CleanupRoot())
}
func TestIntegerRootCopyPrecedesDomainGuardAndAlias(t *testing.T) {
	m, _ := mpzPublicationDirectOwner(t)
	for _, tc := range []struct {
		a    int32
		n    uint32
		want error
	}{{-5, 2, ErrRootNegativeEven}, {2, 0, ErrRootDegreeZero}} {
		a, out := NewInt(tc.a), NewInt(7)
		_, e := m.Root(&a, tc.n, &out)
		if !errors.Is(e, tc.want) || out != a {
			t.Fatal("source SET precedes guarded domain", tc, e, out)
		}
	}
	a := NewInt(16)
	exact, e := m.Root(&a, 2, &a)
	serviceMust(t, e)
	if !exact || a != NewInt(4) {
		t.Fatal("actual input/output alias", exact, a)
	}
}

func TestIntegerRootReceiptPreservesLaterRealInvocation(t *testing.T) {
	m, b := mpzPublicationDirectOwner(t)
	var old, current integerRootInvocationReceipt
	b.FailAfter(0)
	_, first := m.rootInPlaceWithReceipt(&m.two64, 2, &old)
	integerPhysicalAudit(t, m)
	b.FailAfter(-1)
	if !numericPhysicalReservationRefusal(first) || old.invocation == 0 {
		t.Fatal("actual first acquisition", first)
	}
	serviceMust(t, m.cleanupRootInvocation(&old, true))
	b.FailAfter(0)
	_, next := m.rootInPlaceWithReceipt(&m.two64, 2, &current)
	integerPhysicalAudit(t, m)
	b.FailAfter(-1)
	if !numericPhysicalReservationRefusal(next) || old.invocation == current.invocation {
		t.Fatal("actual replacement acquisition", next)
	}
	fields, pending, raw, before := m.rootRaw.values, m.pendingCell, m.rootRaw.raw, b.Used()
	serviceMust(t, m.cleanupRootInvocation(&old, true))
	if !m.rootRaw.active || m.rootRaw.values != fields || m.pendingCell != pending || m.rootRaw.raw != raw || b.Used() != before {
		t.Fatal("old receipt swept real replacement")
	}
	serviceMust(t, m.cleanupRootInvocation(&current, true))
}

// Actual public signed-counter host-overflow profile. No mathematical Root
// call is claimed to create this source-end deallocation boundary.
func TestIntegerRootEndActualPhysicalStopAndOlderRAW(t *testing.T) {
	b := arena.NewBudget(1 << 25)
	rt, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	serviceMust(t, e)
	cfg := serviceConfig(b).Integers
	cfg.Memory = rt
	cfg.Profile = memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	q, e := NewRationalManager(RationalConfig{Integers: cfg})
	serviceMust(t, e)
	m := q.z
	defer func() {
		contextTLSFinish(t, rt, m.CleanupRoot)
		contextTLSFinish(t, rt, q.CloseChecked)
		serviceMust(t, rt.Close())
		if b.Used() != 0 {
			t.Fatal("actual graph refund", b.Used())
		}
	}()
	serviceMust(t, m.beginRootRawScope())
	s := &m.rootRaw
	serviceMust(t, m.SetLimbs(&s.values[0], []uint32{1, 1}))
	serviceMust(t, m.ensureOwnedCapacity(&s.values[0], 40000))
	serviceMust(t, m.Copy(&s.values[1], &m.two64))
	older := s.values[1]
	contextTLSNearMinimum(t, rt)
	first := m.endRootRawScope()
	if !errors.Is(first, arena.ErrBounds) || !s.stopped || s.sourceIndex != 0 || s.values[1] != older {
		t.Fatal("ascending real source-end stop", first)
	}
	integerPhysicalAudit(t, m)
	state := rt.State()
	if again := m.finishRoot(); again != first || rt.State() != state || s.values[1] != older {
		t.Fatal("source retry consumed older RAW", again)
	}
	contextTLSFinish(t, rt, m.CleanupRoot)
	if s.sourceCause != first || s.sourceIndex != 0 {
		t.Fatal("own abort changed source frontier")
	}
	integerPhysicalAudit(t, m)
}

func TestIntegerRootPendingBackingExactAndForeignReplacement(t *testing.T) {
	b := arena.NewBudget(1 << 22)
	m, e := NewIntegerManager(IntegerConfig{Cells: arena.Config{PageSize: 2, SlotLimit: 8, Budget: b}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	serviceMust(t, e)
	defer func() {
		serviceMust(t, m.CleanupRoot())
		m.Close()
		if b.Used() != 0 {
			t.Fatal("whole named owner graph", b.Used())
		}
	}()
	var receipt integerRootInvocationReceipt
	_, e = m.rootInPlaceWithReceipt(&m.two64, 2, &receipt)
	if !errors.Is(e, arena.ErrSlotLimit) || m.pendingCell.object == (memory.ObjectRef{}) || m.rootRaw.pending != m.pendingCell || m.pendingCell.requested == 0 {
		t.Fatal("real raw-before-cell publication full backing receipt", e, m.pendingCell, m.rootRaw.pending)
	}
	integerPhysicalAudit(t, m)
	before, usedBefore := m.rootRaw, b.Used()
	serviceMust(t, m.cleanupRootInvocation(&receipt, false))
	if m.rootRaw != before || m.pendingCell != before.pending || b.Used() != usedBefore {
		t.Fatal("completed phase consumed failed RAW construction")
	}
	old := m.pendingCell
	serviceMust(t, m.CleanupPendingAllocation())
	foreign := Int{}
	e = m.Copy(&foreign, &m.two64)
	if !errors.Is(e, arena.ErrSlotLimit) || m.pendingCell.object == (memory.ObjectRef{}) || m.pendingCell == old {
		t.Fatal("genuine later caller pending backing", e, m.pendingCell)
	}
	integerPhysicalAudit(t, m, &foreign)
	now, used := m.pendingCell, b.Used()
	serviceMust(t, m.CleanupRoot())
	if m.pendingCell != now || b.Used() != used {
		t.Fatal("root abort swept foreign raw-before-cell continuation")
	}
	integerPhysicalAudit(t, m, &foreign)
	serviceMust(t, m.CleanupPendingAllocation())
	serviceMust(t, m.Del(&foreign))
}
