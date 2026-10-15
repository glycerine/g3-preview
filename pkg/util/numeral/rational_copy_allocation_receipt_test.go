package numeral

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestRationalCopyAllocationReceiptSuccessfulCopyHasNoPendingOwner(t *testing.T) {
	s, _ := serviceNew(t)
	src := serviceValue(t, s, "12345678901234567890123456789/998877665544332211")
	dst, e := s.NewValue()
	serviceMust(t, e)
	var receipt RationalCopyAllocationReceipt
	serviceMust(t, s.CopyWithAllocationReceipt(&dst, &src, &receipt))
	if receipt.Active() || !errors.Is(s.CleanupCopyAllocation(&receipt), arena.ErrInvalidSlot) {
		t.Fatal("completed numerator and denominator retained allocation authority")
	}
	equal, e := s.Equal(&dst, &src)
	if e != nil || !equal {
		t.Fatal("completed tracked copy changed rational value", equal, e)
	}
	serviceMust(t, s.Del(&dst))
	serviceMust(t, s.Del(&src))
}

func TestRationalCopyAllocationReceiptDenominatorPostphysicalRefusal(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	runtime, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b, SizeTBits: 64})
	serviceMust(t, e)
	s, e := NewRationalService(RationalServiceConfig{Integers: IntegerConfig{Memory: runtime, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{PageSize: 2, SlotLimit: 10, Budget: b}, Words: arena.VectorConfig{PageSize: 16, Descriptors: arena.Config{PageSize: 2, Budget: b}}}})
	serviceMust(t, e)
	t.Cleanup(func() {
		serviceMust(t, s.Close())
		serviceMust(t, runtime.Close())
		if b.Used() != 0 {
			t.Error("retained numeric owner", b.Used())
		}
	})
	src, e := s.NewValue()
	serviceMust(t, e)
	serviceMust(t, s.SetInt64(&src, 1))
	digits := make([]uint32, 32768)
	digits[0], digits[len(digits)-1] = 17, 1
	serviceMust(t, s.Manager.Z.SetLimbs(&src.Raw.Den, digits))
	valid, e := s.IsRational(&src)
	if e != nil || !valid {
		t.Fatal("actual large denominator source", valid, e)
	}
	dst, e := s.NewValue()
	serviceMust(t, e)
	occupied, e := s.NewValue()
	serviceMust(t, e)
	serviceMust(t, s.Manager.Z.SetLimbs(&occupied.Raw.Num, []uint32{1, 0, 0, 0, 0, 1}))
	var receipt RationalCopyAllocationReceipt
	e = s.CopyWithAllocationReceipt(&dst, &src, &receipt)
	if !errors.Is(e, arena.ErrSlotLimit) || !receipt.Active() || s.Manager.Z.PendingCell.Object == (umemory.ObjectRef{}) {
		t.Fatal("denominator copy did not retain exact postphysical allocation", e, receipt.Active(), s.Manager.Z.PendingCell)
	}
	if dst.Raw.Num.Val != 1 || dst.Raw.Num.Cell != 0 || dst.Raw.Den.Cell != s.Manager.Z.FailedNullRef {
		t.Fatal("numerator prefix or failed denominator changed", dst.Raw)
	}
	serviceMust(t, s.CleanupCopyAllocation(&receipt))
	if s.Manager.Z.PendingCell.Object != (umemory.ObjectRef{}) || receipt.Active() {
		t.Fatal("denominator allocation receipt failed to drain")
	}
	serviceMust(t, s.Del(&dst))
	serviceMust(t, s.Del(&src))
	serviceMust(t, s.Del(&occupied))
}
