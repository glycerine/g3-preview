package numeral

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestIntegerNumeralBufferSourceReserveIndexesAndOwnsSlots(t *testing.T) {
	m, budget := integerManager(t)
	baseLive := m.Cells.Stats().Live
	v, e := NewScopedIntVector(m)
	if e != nil {
		t.Fatal(e)
	}
	if e = v.Reserve(4); e != nil || v.Size() != 4 {
		t.Fatalf("source reserve size = %d, error %v", v.Size(), e)
	}
	for i := uint32(0); i < 4; i++ {
		slot, e := v.BorrowSlot(i)
		if e != nil {
			t.Fatal(e)
		}
		zero, e := m.IsZero(slot)
		if e != nil || !zero {
			t.Fatalf("new numeral slot %d = %v, zero=%v", i, e, zero)
		}
	}
	large := NewInt(0)
	if e = m.SetString(&large, "123456789012345678901234567890"); e != nil {
		t.Fatal(e)
	}
	if e = v.Set(3, &large); e != nil {
		t.Fatal(e)
	}
	if e = v.Reserve(2); e != nil || v.Size() != 4 {
		t.Fatalf("smaller reserve shrank buffer to %d: %v", v.Size(), e)
	}
	if e = v.Reserve(9); e != nil || v.Size() != 9 {
		t.Fatalf("grown reserve size = %d, error %v", v.Size(), e)
	}
	slot, e := v.BorrowSlot(3)
	if e != nil {
		t.Fatal(e)
	}
	got, e := m.String(slot)
	if e != nil || got != "123456789012345678901234567890" {
		t.Fatalf("reserved slot value = %s, error %v", got, e)
	}
	for i := uint32(4); i < 9; i++ {
		slot, e := v.BorrowSlot(i)
		if e != nil {
			t.Fatal(e)
		}
		zero, e := m.IsZero(slot)
		if e != nil || !zero {
			t.Fatalf("grown numeral slot %d = %v, zero=%v", i, e, zero)
		}
	}
	if e = v.Close(); e != nil {
		t.Fatal(e)
	}
	if e = m.Del(&large); e != nil {
		t.Fatal(e)
	}
	if live := m.Cells.Stats().Live; live != baseLive {
		t.Fatalf("reserved numeral slots retained %d live cells", live-baseLive)
	}
	if e = v.Reserve(1); e != arena.ErrClosed {
		t.Fatalf("closed reserve = %v", e)
	}
	m.Close()
	if budget.Used() != 0 {
		t.Fatalf("integer owner retained %d bytes", budget.Used())
	}
}
