package numeral

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	"strconv"
	"testing"
)

func TestBinaryCopyFromIndependentOwnerReservationPrefixes(t *testing.T) {
	success, failures := false, 0
	for fault := int64(0); fault < 128 && !success; fault++ {
		if !t.Run(strconv.FormatInt(fault, 10), func(t *testing.T) {
			source, _ := binaryManager(t)
			target, budget := binaryManager(t)
			input, out := NewBinaryRational(), NewBinaryRational()
			digits := make([]uint32, 65)
			digits[0], digits[64] = 17, 1
			serviceMust(t, source.Z.SetLimbs(&input.Num, digits))
			serviceMust(t, source.Z.Neg(&input.Num))
			input.KField = 91
			serviceMust(t, target.Z.Copy(&out.Num, &target.Z.Two64))
			out.KField = 7
			before, used := input, source.Z.Budget().Used()
			var receipt IntegerCopyAllocationReceipt
			budget.FailAfter(fault)
			body := target.CopyFromOwnerWithAllocationReceipt(&out, source, &input, &receipt)
			budget.FailAfter(-1)
			if body != nil {
				failures++
				if !numericStorageReservationRefusal(body) {
					t.Fatal("unexpected foreign copy failure", body)
				}
				if out.KField != 7 {
					t.Fatal("failed numerator copy published exponent", fault, out.KField)
				}
			} else {
				success = true
				if out.KField != 91 || out.Num.Val != -1 || target.Z.Cell(out.Num.Cell).Size != 65 {
					t.Fatal("foreign value shape", out)
				}
				for i, want := range digits {
					if got := target.Z.Digit(out.Num.Cell, uint32(i)); got != want {
						t.Fatal("foreign limb", i, got, want)
					}
				}
			}
			if input != before || source.Z.Budget().Used() != used {
				t.Fatal("receiving copy changed source owner")
			}
			serviceMust(t, target.Z.CleanupCopyAllocationInvocation(&receipt))
			serviceMust(t, target.Z.CleanupRawNumeral(&out.Num))
			serviceMust(t, source.Del(&input))
		}) {
			return
		}
	}
	if !success || failures == 0 {
		t.Fatal("copy campaign did not reach refusal and success", failures, success)
	}
}

func TestBinaryCopyIndependentOwnerSurvivesSourceClose(t *testing.T) {
	source, _ := binaryManager(t)
	target, _ := binaryManager(t)
	input, out := NewBinaryRational(), NewBinaryRational()
	serviceMust(t, source.Z.SetLimbs(&input.Num, []uint32{17, 0, 1}))
	input.KField = 70
	var receipt IntegerCopyAllocationReceipt
	serviceMust(t, target.CopyFromOwnerWithAllocationReceipt(&out, source, &input, &receipt))
	serviceMust(t, source.Del(&input))
	serviceMust(t, source.CloseChecked())
	serviceMust(t, source.Z.CloseChecked())
	if out.KField != 70 || target.Z.Cell(out.Num.Cell).Size != 3 || target.Z.Digit(out.Num.Cell, 0) != 17 || target.Z.Digit(out.Num.Cell, 2) != 1 {
		t.Fatal("receiving bound depended on closed source owner", out)
	}
	serviceMust(t, target.Del(&out))
}

func TestBinaryCopyFromOwnerRetainsExactDescriptorRefusal(t *testing.T) {
	source, _ := binaryManager(t)
	budget := arena.NewBudget(1 << 24)
	diagnostics, e := NewRationalService(serviceConfig(budget))
	serviceMust(t, e)
	z, e := NewIntegerManager(IntegerConfig{Cells: arena.Config{PageSize: 8, SlotLimit: 8, Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	serviceMust(t, e)
	target, e := NewBinaryRationalManager(z, BinaryRationalConfig{Diagnostics: diagnostics})
	serviceMust(t, e)
	t.Cleanup(func() {
		serviceMust(t, target.CloseChecked())
		serviceMust(t, z.CloseChecked())
		serviceMust(t, diagnostics.Close())
		if budget.Used() != 0 {
			t.Error("receiving owner leaked", budget.Used())
		}
	})
	input := BinaryRationalFromBorrowedNumeral(source.Z.Two64, 17)
	out := NewBinaryIntPower(19, 7)
	var receipt IntegerCopyAllocationReceipt
	body := target.CopyFromOwnerWithAllocationReceipt(&out, source, &input, &receipt)
	if !errors.Is(body, arena.ErrSlotLimit) || !z.MatchesCopyAllocationInvocation(&receipt) || z.PendingCell.Object == (umemory.ObjectRef{}) || out.KField != 7 {
		t.Fatal("foreign copy lost exact physical allocation prefix", body, out.KField)
	}
	integerPhysicalAudit(t, z, &out.Num)
	serviceMust(t, z.CleanupCopyAllocationInvocation(&receipt))
	serviceMust(t, z.CleanupRawNumeral(&out.Num))
	serviceMust(t, target.Del(&out))
}
