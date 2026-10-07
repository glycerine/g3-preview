// Source: src/util/mpz.cpp:1453-1507; src/util/mpz.h.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestIntegerPowerGeneralRetainsRAWBeforeCallerSET(t *testing.T) {
	m, b := mpzPublicationDirectOwner(t)
	out := NewInt(7)
	b.FailAfter(0)
	e := m.Power(&m.Two64, 2, &out)
	integerPhysicalAudit(t, m, &out)
	b.FailAfter(-1)
	if !numericPhysicalReservationRefusal(e) || !m.PowerRaw.active || m.PowerRaw.endEntered || m.PowerRaw.Value.Cell != m.FailedNullRef || out != NewInt(7) {
		t.Fatal("source RAW COPY before caller SET", e, m.PowerRaw, out)
	}
	nonce := m.PowerRaw.generation
	two := NewInt(2)
	small := NewInt(9)
	serviceMust(t, m.Power(&two, 3, &small))
	if small != NewInt(8) || m.PowerRaw.generation != nonce || !m.PowerRaw.active {
		t.Fatal("quick Power route touched older RAW invocation")
	}
	if e = m.Power(&m.Two64, 2, &out); !errors.Is(e, ErrPowerPending) || m.PowerRaw.generation != nonce {
		t.Fatal("ordinary general call erased older RAW", e)
	}
	serviceMust(t, m.CleanupPower())
}
func TestIntegerPowerGeneralMinusOneUsesActualRAW(t *testing.T) {
	m, _ := mpzPublicationDirectOwner(t)
	a, out := NewInt(-1), NewInt(7)
	before := m.powerGeneration
	serviceMust(t, m.Power(&a, 3, &out))
	if out != NewInt(-1) || m.powerGeneration != before+1 || m.PowerRaw.active || m.PowerRaw.sourceIndex != 1 {
		t.Fatal("minus one must use source general RAW/final-square path", out, m.powerGeneration, m.PowerRaw)
	}
}

func TestIntegerPowerPendingBackingExactAndForeignReplacement(t *testing.T) {
	b := arena.NewBudget(1 << 22)
	m, e := NewIntegerManager(IntegerConfig{Cells: arena.Config{PageSize: 2, SlotLimit: 8, Budget: b}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	serviceMust(t, e)
	defer func() {
		serviceMust(t, m.CleanupPower())
		m.Close()
		if b.Used() != 0 {
			t.Fatal("whole named owner graph", b.Used())
		}
	}()
	out := NewInt(7)
	var receipt integerPowerInvocationReceipt
	e = m.powerWithReceipt(&m.Two64, 2, &out, &receipt)
	if !errors.Is(e, arena.ErrSlotLimit) || m.PendingCell.Object == (ObjectRef{}) || m.PowerRaw.pending != m.PendingCell || m.PendingCell.Requested == 0 || out != NewInt(7) {
		t.Fatal("RAW COPY full actual backing before caller SET", e, m.PendingCell, m.PowerRaw.pending, out)
	}
	integerPhysicalAudit(t, m, &out)
	before, usedBefore := m.PowerRaw, b.Used()
	serviceMust(t, m.CleanupPowerInvocation(&receipt, false))
	if m.PowerRaw != before || m.PendingCell != before.pending || b.Used() != usedBefore {
		t.Fatal("completed phase consumed failed RAW construction")
	}
	old := m.PendingCell
	serviceMust(t, m.CleanupPendingAllocation())
	foreign := Int{}
	e = m.Copy(&foreign, &m.Two64)
	if !errors.Is(e, arena.ErrSlotLimit) || m.PendingCell.Object == (ObjectRef{}) || m.PendingCell == old {
		t.Fatal("genuine replacement caller continuation", e)
	}
	integerPhysicalAudit(t, m, &out, &foreign)
	now, used := m.PendingCell, b.Used()
	serviceMust(t, m.CleanupPower())
	if m.PendingCell != now || b.Used() != used {
		t.Fatal("Power abort swept actual foreign continuation")
	}
	integerPhysicalAudit(t, m, &out, &foreign)
	serviceMust(t, m.CleanupPendingAllocation())
	serviceMust(t, m.Del(&foreign))
}

func TestIntegerPowerCallerOutputPendingExcludedFromRAWOwner(t *testing.T) {
	b := arena.NewBudget(1 << 22)
	m, e := NewIntegerManager(IntegerConfig{Cells: arena.Config{PageSize: 2, SlotLimit: 9, Budget: b}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}, Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	serviceMust(t, e)
	out := NewInt(7)
	defer func() {
		serviceMust(t, m.CleanupPower())
		serviceMust(t, m.CleanupPendingAllocation())
		serviceMust(t, m.Del(&out))
		m.Close()
		if b.Used() != 0 {
			t.Fatal("named caller and RAW owner graph", b.Used())
		}
	}()
	var r integerPowerInvocationReceipt
	e = m.powerWithReceipt(&m.Two64, 2, &out, &r)
	if !errors.Is(e, arena.ErrSlotLimit) || !m.matchesPowerInvocation(&r) || m.PowerRaw.endEntered || m.PendingCell.Object == (ObjectRef{}) || m.PowerRaw.pending.Object != (ObjectRef{}) {
		t.Fatal("actual caller MUL pending must remain outside RAW receipt", e, m.PendingCell, m.PowerRaw)
	}
	integerPhysicalAudit(t, m, &out)
	caller := m.PendingCell
	serviceMust(t, m.CleanupPowerInvocation(&r, true))
	if m.PendingCell != caller {
		t.Fatal("RAW abort consumed caller output allocation")
	}
	integerPhysicalAudit(t, m, &out)
}

func TestIntegerPowerReceiptRejectQuickAndReplacement(t *testing.T) {
	m, b := mpzPublicationDirectOwner(t)
	var old integerPowerInvocationReceipt
	out := NewInt(7)
	b.FailAfter(0)
	e := m.powerWithReceipt(&m.Two64, 2, &out, &old)
	integerPhysicalAudit(t, m, &out)
	b.FailAfter(-1)
	if !numericPhysicalReservationRefusal(e) || !m.matchesPowerInvocation(&old) {
		t.Fatal("actual first guarded acquisition", e)
	}
	rejected := old
	if e = m.powerWithReceipt(&m.Two64, 3, &out, &rejected); !errors.Is(e, ErrPowerPending) || rejected != old {
		t.Fatal("rejected entry changed receipt", e)
	}
	quick, two := NewInt(0), NewInt(2)
	serviceMust(t, m.powerWithReceipt(&two, 3, &quick, &rejected))
	if quick != NewInt(8) || rejected != old {
		t.Fatal("quick source route acquired RAW invocation")
	}
	serviceMust(t, m.CleanupPowerInvocation(&old, true))
	var next integerPowerInvocationReceipt
	b.FailAfter(0)
	e = m.powerWithReceipt(&m.Two64, 2, &out, &next)
	integerPhysicalAudit(t, m, &out)
	b.FailAfter(-1)
	if !numericPhysicalReservationRefusal(e) || !m.matchesPowerInvocation(&next) || next.invocation == old.invocation {
		t.Fatal("real replacement invocation", e)
	}
	before, used := m.PowerRaw, b.Used()
	serviceMust(t, m.CleanupPowerInvocation(&old, true))
	if m.PowerRaw != before || b.Used() != used {
		t.Fatal("old receipt consumed replacement RAW")
	}
	serviceMust(t, m.CleanupPowerInvocation(&next, true))
}
