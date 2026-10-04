// Source: src/util/mpz.cpp:1871-1929; src/util/mpz.h:820.
// Licensed under the MIT license. Author: Leonardo de Moura.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestIntegerPerfectSquareResetsCallerBeforeRawCopyRefusal(t *testing.T) {
	m, b := mpzPublicationDirectOwner(t)
	input, out := Int{}, Int{}
	limbs := make([]uint32, 80)
	for i := range limbs {
		limbs[i] = 1
	}
	serviceMust(t, m.SetLimbs(&input, limbs))
	serviceMust(t, m.Copy(&out, &m.two64))
	defer func() { b.FailAfter(-1); serviceMust(t, m.Del(&input)); serviceMust(t, m.Del(&out)) }()
	b.FailAfter(0)
	_, e := m.IsPerfectSquare(&input, &out)
	if !numericPhysicalReservationRefusal(e) {
		t.Fatal("actual raw hi Copy reservation", e)
	}
	integerPhysicalAudit(t, m, &input, &out)
	if out.cell != 0 || out.val != 0 {
		t.Fatal("source reset caller before six raw locals", out)
	}
}

func TestIntegerPerfectSquareOwnsActualFailedRawFields(t *testing.T) {
	m, b := mpzPublicationDirectOwner(t)
	input, out := Int{}, Int{}
	serviceMust(t, m.Copy(&input, &m.two64))
	defer func() {
		b.FailAfter(-1)
		serviceMust(t, m.CleanupPerfectSquare())
		serviceMust(t, m.Del(&input))
		serviceMust(t, m.Del(&out))
	}()
	b.FailAfter(1)
	_, e := m.IsPerfectSquare(&input, &out)
	if !numericPhysicalReservationRefusal(e) {
		t.Fatal("real second source reservation", e)
	}
	integerPhysicalAudit(t, m, &input, &out)
	b.FailAfter(-1)
	serviceMust(t, m.CleanupPerfectSquare())
	integerPhysicalAudit(t, m, &input, &out)
}

// Public TLS signed-counter overflow is a guarded host-domain owner-primitive
// proof, not valid allocation exhaustion or an end-to-end square-root recipe.
func TestIntegerPerfectSquareRawEndStopsBeforeLaterDeletion(t *testing.T) {
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
		contextTLSFinish(t, rt, m.CleanupPerfectSquare)
		contextTLSFinish(t, rt, q.CloseChecked)
		serviceMust(t, rt.Close())
		if b.Used() != 0 {
			t.Fatal("actual whole graph refund", b.Used())
		}
	}()
	serviceMust(t, m.beginPerfectSquareRawScope())
	s := &m.perfectSquare
	serviceMust(t, m.SetLimbs(&s.values[0], []uint32{1, 1}))
	serviceMust(t, m.ensureOwnedCapacity(&s.values[0], 40000))
	serviceMust(t, m.Copy(&s.values[1], &m.two64))
	hi := s.values[1]
	contextTLSNearMinimum(t, rt)
	cause := m.endPerfectSquareRawScope()
	if !errors.Is(cause, arena.ErrBounds) || !s.stopped || s.sourceIndex != 0 || s.values[1] != hi {
		t.Fatal("actual first RAW source Del frontier", cause)
	}
	integerPhysicalAudit(t, m)
	state := rt.State()
	first := s.sourceCause
	if e = m.finishPerfectSquare(); e != first || rt.State() != state || s.values[1] != hi {
		t.Fatal("source consumed release replay", e)
	}
	contextTLSFinish(t, rt, m.CleanupPerfectSquare)
	if s.sourceCause != first || s.sourceIndex != 0 {
		t.Fatal("owning abort changed source history")
	}
	integerPhysicalAudit(t, m)
}

func TestIntegerPerfectSquarePendingReceiptLeavesForeignReplacement(t *testing.T) {
	b := arena.NewBudget(1 << 22)
	m, e := NewIntegerManager(IntegerConfig{Cells: arena.Config{PageSize: 2, SlotLimit: 8, Budget: b}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	serviceMust(t, e)
	defer func() {
		serviceMust(t, m.CleanupPerfectSquare())
		serviceMust(t, m.CleanupPendingAllocation())
		serviceMust(t, m.CloseChecked())
		if b.Used() != 0 {
			t.Fatal("whole actual owner refund", b.Used())
		}
	}()
	out := Int{}
	_, e = m.IsPerfectSquare(&m.two64, &out)
	if !errors.Is(e, arena.ErrSlotLimit) || m.pendingCell.object == (memory.ObjectRef{}) || m.perfectSquare.pending != m.pendingCell.object {
		t.Fatal("actual internal raw hi pending receipt", e)
	}
	integerPhysicalAudit(t, m, &out)
	old := m.pendingCell
	serviceMust(t, m.CleanupPendingAllocation())
	foreign := Int{}
	e = m.Copy(&foreign, &m.intMin)
	if !errors.Is(e, arena.ErrSlotLimit) || m.pendingCell.object == (memory.ObjectRef{}) || m.pendingCell.object == old.object {
		t.Fatal("actual later caller replacement", e)
	}
	later := m.pendingCell
	if later.raw == old.raw {
		t.Fatal("replacement must prove distinct actual raw identity", old, later)
	}
	integerPhysicalAudit(t, m, &out, &foreign)
	serviceMust(t, m.CleanupPerfectSquare())
	if m.pendingCell != later {
		t.Fatal("raw invocation abort swept later foreign caller")
	}
	integerPhysicalAudit(t, m, &out, &foreign)
	serviceMust(t, m.Del(&out))
	serviceMust(t, m.Del(&foreign))
}

func TestIntegerPerfectSquareEarlyRoutesKeepOlderRawInvocation(t *testing.T) {
	m, b := mpzPublicationDirectOwner(t)
	input, out := Int{}, Int{}
	serviceMust(t, m.Copy(&input, &m.two64))
	defer func() {
		b.FailAfter(-1)
		serviceMust(t, m.CleanupPerfectSquare())
		serviceMust(t, m.Del(&input))
		serviceMust(t, m.Del(&out))
	}()
	b.FailAfter(1)
	_, e := m.IsPerfectSquare(&input, &out)
	if !numericPhysicalReservationRefusal(e) || !m.perfectSquare.active {
		t.Fatal("actual retained raw body", e)
	}
	integerPhysicalAudit(t, m, &input, &out)
	b.FailAfter(-1)
	generation, cause := m.perfectSquare.generation, m.perfectSquare.bodyCause
	for _, v := range []int32{-1, 0, 1} {
		a := NewInt(v)
		out = NewInt(7)
		ok, e := m.IsPerfectSquare(&a, &out)
		serviceMust(t, e)
		if v < 0 {
			if ok || out.val != 7 {
				t.Fatal("negative source before Reset")
			}
		} else if !ok || out.val != v {
			t.Fatal("zero/one source before raw construction")
		}
		if !m.perfectSquare.active || m.perfectSquare.generation != generation || m.perfectSquare.bodyCause != cause {
			t.Fatal("source early route overwrote older raw context")
		}
	}
}

// A fully initialized noncanonical BIG one is a source width/read-profile
// input. Source is_one tests m_ptr==NULL, not mathematical equality.
func TestIntegerPerfectSquareBigOneReachesRawCopy(t *testing.T) {
	m, b := mpzPublicationDirectOwner(t)
	input, out := Int{}, Int{}
	serviceMust(t, m.SetLimbs(&input, []uint32{1, 1}))
	m.resizeCell(input.cell, 1)
	defer func() {
		b.FailAfter(-1)
		serviceMust(t, m.CleanupPerfectSquare())
		serviceMust(t, m.Del(&input))
		serviceMust(t, m.Del(&out))
	}()
	b.FailAfter(0)
	_, e := m.IsPerfectSquare(&input, &out)
	integerPhysicalAudit(t, m, &input, &out)
	if !numericPhysicalReservationRefusal(e) || !m.perfectSquare.active || out.cell != 0 || out.val != 0 {
		t.Fatal("source BIG one is not inline is_one", e, out)
	}
}
