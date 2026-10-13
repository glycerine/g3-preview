package numeral

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestScopedArbitraryFloatVectorRetainedMPFSlots(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	v, err := NewScopedArbitraryFloatVector(m)
	if err != nil {
		t.Fatal(err)
	}
	var source ArbitraryFloat
	defer func() {
		if err := m.Del(&source); err != nil {
			t.Error(err)
		}
		if err := v.Close(); err != nil {
			t.Error(err)
		}
	}()
	if err := m.SetInt32(&source, 11, 53, 3); err != nil {
		t.Fatal(err)
	}
	if err := v.Reserve(2); err != nil {
		t.Fatal(err)
	}
	if v.Size() != 2 {
		t.Fatal("source reserve must extend logical size")
	}
	if err := v.Set(1, &source); err != nil {
		t.Fatal(err)
	}
	if err := m.SetInt32(&source, 11, 53, 7); err != nil {
		t.Fatal(err)
	}
	slot, err := v.BorrowSlot(1)
	if err != nil {
		t.Fatal(err)
	}
	got, err := m.ToDouble(slot)
	if err != nil || got != 3 {
		t.Fatalf("retained slot must own its copy: %v, %v", got, err)
	}
	values, err := v.BorrowedValues()
	if err != nil || len(values) != 2 {
		t.Fatalf("source c_ptr view: %d, %v", len(values), err)
	}
	if err := v.Shrink(1); err != nil {
		t.Fatal(err)
	}
	if _, err := v.BorrowSlot(1); !errors.Is(err, arena.ErrBounds) {
		t.Fatalf("removed slot: %v", err)
	}
	if err := v.Reserve(3); err != nil || v.Size() != 3 {
		t.Fatalf("regrow default slots: %d, %v", v.Size(), err)
	}
	if err := m.SetInt32(&source, 11, 120, 3); err != nil {
		t.Fatal(err)
	}
	if err := v.Set(2, &source); err != nil {
		t.Fatal(err)
	}
	heapSlot, err := v.BorrowSlot(2)
	if err != nil {
		t.Fatal(err)
	}
	before, err := m.integer.String(&heapSlot.significand)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.SetInt32(&source, 11, 120, 7); err != nil {
		t.Fatal(err)
	}
	after, err := m.integer.String(&heapSlot.significand)
	if err != nil || after != before {
		t.Fatalf("large significand copy changed with source: %q -> %q, %v", before, after, err)
	}
}
