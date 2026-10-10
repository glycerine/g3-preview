// Source: src/util/mpz.h:836-837; src/util/mpz.cpp:1942-2012;
// src/util/mpzzp.h:279-280.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestIntegerRootSourceCopyPublishesCallerBeforeBisection(t *testing.T) {
	m, b := mpzPublicationDirectOwner(t)
	a := Int{}
	serviceMust(t, m.Copy(&a, &m.Two64))
	defer func() { serviceMust(t, m.Del(&a)) }()
	out := NewInt(7)
	defer func() { serviceMust(t, m.Del(&out)) }()
	b.FailAfter(0)
	_, e := m.Root(&a, 2, &out)
	integerPhysicalAudit(t, m, &a, &out)
	b.FailAfter(-1)
	if !numericPhysicalReservationRefusal(e) || out.Cell != m.FailedNullRef || out.Val != 1 {
		t.Fatal("source SET(output,input) refusal must publish the real caller", e, out)
	}
}

func TestIntegerRootInPlaceRetainsActualRAWOnBodyFailure(t *testing.T) {
	m, b := mpzPublicationDirectOwner(t)
	a := Int{}
	serviceMust(t, m.Copy(&a, &m.Two64))
	defer func() { serviceMust(t, m.Del(&a)) }()
	before := a
	b.FailAfter(0)
	_, e := m.RootInPlace(&a, 2)
	integerPhysicalAudit(t, m, &a)
	b.FailAfter(-1)
	if !numericPhysicalReservationRefusal(e) || !m.RootRaw.active || m.RootRaw.EndEntered || m.RootRaw.Values[0].Cell != m.FailedNullRef || a != before {
		t.Fatal("source initial POWER retained RAW target", e, m.RootRaw, a)
	}
	generation := m.RootRaw.generation
	_, again := m.RootInPlace(&a, 2)
	if !errors.Is(again, ErrRootPending) || m.RootRaw.generation != generation {
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
	_, first := m.rootInPlaceWithReceipt(&m.Two64, 2, &old)
	integerPhysicalAudit(t, m)
	b.FailAfter(-1)
	if !numericPhysicalReservationRefusal(first) || old.invocation == 0 {
		t.Fatal("actual first acquisition", first)
	}
	serviceMust(t, m.CleanupRootInvocation(&old, true))
	b.FailAfter(0)
	_, next := m.rootInPlaceWithReceipt(&m.Two64, 2, &current)
	integerPhysicalAudit(t, m)
	b.FailAfter(-1)
	if !numericPhysicalReservationRefusal(next) || old.invocation == current.invocation {
		t.Fatal("actual replacement acquisition", next)
	}
	fields, pending, raw, before := m.RootRaw.Values, m.PendingCell, m.RootRaw.raw, b.Used()
	serviceMust(t, m.CleanupRootInvocation(&old, true))
	if !m.RootRaw.active || m.RootRaw.Values != fields || m.PendingCell != pending || m.RootRaw.raw != raw || b.Used() != before {
		t.Fatal("old receipt swept real replacement")
	}
	serviceMust(t, m.CleanupRootInvocation(&current, true))
}

// Actual public signed-counter host-overflow profile. No mathematical Root
// call is claimed to create this source-end deallocation boundary.
func TestIntegerRootEndActualPhysicalStopAndOlderRAW(t *testing.T) {
	b := arena.NewBudget(1 << 25)
	rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	serviceMust(t, e)
	cfg := serviceConfig(b).Integers
	cfg.Memory = rt
	cfg.Profile = umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	q, e := NewRationalManager(RationalConfig{Integers: cfg})
	serviceMust(t, e)
	m := q.Z
	defer func() {
		contextTLSFinish(t, rt, m.CleanupRoot)
		contextTLSFinish(t, rt, q.CloseChecked)
		serviceMust(t, rt.Close())
		if b.Used() != 0 {
			t.Fatal("actual graph refund", b.Used())
		}
	}()
	serviceMust(t, m.beginRootRawScope())
	s := &m.RootRaw
	serviceMust(t, m.SetLimbs(&s.Values[0], []uint32{1, 1}))
	serviceMust(t, m.EnsureOwnedCapacity(&s.Values[0], 40000))
	serviceMust(t, m.Copy(&s.Values[1], &m.Two64))
	older := s.Values[1]
	contextTLSNearMinimum(t, rt)
	first := m.endRootRawScope()
	if !errors.Is(first, arena.ErrBounds) || !s.stopped || s.SourceIndex != 0 || s.Values[1] != older {
		t.Fatal("ascending real source-end stop", first)
	}
	integerPhysicalAudit(t, m)
	state := rt.State()
	if again := m.finishRoot(); again != first || rt.State() != state || s.Values[1] != older {
		t.Fatal("source retry consumed older RAW", again)
	}
	contextTLSFinish(t, rt, m.CleanupRoot)
	if s.sourceCause != first || s.SourceIndex != 0 {
		t.Fatal("own abort changed source frontier")
	}
	integerPhysicalAudit(t, m)
}

func TestIntegerRootPendingBackingExactAndForeignReplacement(t *testing.T) {
	b := arena.NewBudget(1 << 22)
	m, e := NewIntegerManager(IntegerConfig{Cells: arena.Config{PageSize: 2, SlotLimit: 8, Budget: b}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	serviceMust(t, e)
	defer func() {
		serviceMust(t, m.CleanupRoot())
		m.Close()
		if b.Used() != 0 {
			t.Fatal("whole named owner graph", b.Used())
		}
	}()
	var receipt integerRootInvocationReceipt
	_, e = m.rootInPlaceWithReceipt(&m.Two64, 2, &receipt)
	if !errors.Is(e, arena.ErrSlotLimit) || m.PendingCell.Object == (umemory.ObjectRef{}) || m.RootRaw.pending != m.PendingCell || m.PendingCell.Requested == 0 {
		t.Fatal("real raw-before-cell publication full backing receipt", e, m.PendingCell, m.RootRaw.pending)
	}
	integerPhysicalAudit(t, m)
	before, usedBefore := m.RootRaw, b.Used()
	serviceMust(t, m.CleanupRootInvocation(&receipt, false))
	if m.RootRaw != before || m.PendingCell != before.pending || b.Used() != usedBefore {
		t.Fatal("completed phase consumed failed RAW construction")
	}
	old := m.PendingCell
	serviceMust(t, m.CleanupPendingAllocation())
	foreign := Int{}
	e = m.Copy(&foreign, &m.Two64)
	if !errors.Is(e, arena.ErrSlotLimit) || m.PendingCell.Object == (umemory.ObjectRef{}) || m.PendingCell == old {
		t.Fatal("genuine later caller pending backing", e, m.PendingCell)
	}
	integerPhysicalAudit(t, m, &foreign)
	now, used := m.PendingCell, b.Used()
	serviceMust(t, m.CleanupRoot())
	if m.PendingCell != now || b.Used() != used {
		t.Fatal("root abort swept foreign raw-before-cell continuation")
	}
	integerPhysicalAudit(t, m, &foreign)
	serviceMust(t, m.CleanupPendingAllocation())
	serviceMust(t, m.Del(&foreign))
}
