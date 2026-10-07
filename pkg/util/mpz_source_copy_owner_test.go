// Source: src/util/mpz.h:662-672; src/util/mpz.cpp:1186-1213.
package util

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestIntegerSourceCopyFromOwnerEveryDestructivePrefix(t *testing.T) {
	success := false
	for ordinal := int64(0); ordinal < 128; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			source, _ := mpzPublicationDirectOwner(t)
			target, budget := mpzPublicationDirectOwner(t)
			var input, out Int
			digits := make([]uint32, 65)
			digits[0] = 17
			digits[64] = 1
			serviceMust(t, source.SetLimbs(&input, digits))
			serviceMust(t, source.Neg(&input))
			serviceMust(t, target.Copy(&out, &target.Two64))
			old := target.Cell(out.Cell).Backing
			inputToken := input
			sourceState := source.Memory.State()
			sourceBytes := source.BudgetField.Used()
			defer func() {
				budget.FailAfter(-1)
				serviceMust(t, target.CleanupPendingAllocation())
				serviceMust(t, target.Allocator.CleanupPending())
				serviceMust(t, target.Del(&out))
				serviceMust(t, source.Del(&input))
			}()
			budget.FailAfter(ordinal)
			e := target.CopyFromOwner(&out, source, &input)
			integerPhysicalAudit(t, target, &out)
			integerPhysicalAudit(t, source, &input)
			if input != inputToken || source.Memory.State() != sourceState || source.BudgetField.Used() != sourceBytes {
				t.Fatal("source owner modified")
			}
			if e != nil {
				if !numericPhysicalReservationRefusal(e) {
					t.Fatal("actual target refusal", ordinal, e)
				}
				if out.Val != -1 || out.Cell != target.FailedRef {
					t.Fatal("literal destructive sign/old-free/dead-pointer publication", out)
				}
				if _, x := target.Memory.Size(old.Raw); x == nil {
					t.Fatal("old target physical block survived source SET growth")
				}
			} else {
				success = true
				if target.Cell(out.Cell).Capacity != source.Cell(input.Cell).Capacity || target.Cell(out.Cell).Size != 65 {
					t.Fatal("source size/capacity preserved")
				}
				budget.FailAfter(-1) // observation string is outside the operation fault epoch
				text, e := target.String(&out)
				serviceMust(t, e)
				original, e := source.String(&input)
				serviceMust(t, e)
				if text != original {
					t.Fatal("actual source limb copy", text, original)
				}
			}
			budget.FailAfter(-1)
		})
		if success {
			t.Log("first real target success", ordinal)
			break
		}
	}
	if !success {
		t.Fatal("complete target reservation horizon")
	}
}

func TestIntegerSourceCopyFromOwnerPendingReplacement(t *testing.T) {
	source, _ := mpzPublicationDirectOwner(t)
	budget := arena.NewBudget(1 << 24)
	target, e := NewIntegerManager(IntegerConfig{Profile: SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{PageSize: 8, SlotLimit: 8, Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}})
	serviceMust(t, e)
	var input, out, foreign Int
	serviceMust(t, source.Copy(&input, &source.Two64))
	defer func() {
		serviceMust(t, target.CleanupPendingAllocation())
		serviceMust(t, target.Allocator.CleanupPending())
		serviceMust(t, target.Del(&out))
		serviceMust(t, target.Del(&foreign))
		serviceMust(t, source.Del(&input))
		serviceMust(t, target.CloseChecked())
		if budget.Used() != 0 {
			t.Fatal("target physical graph refund", budget.Used())
		}
	}()
	e = target.CopyFromOwner(&out, source, &input)
	if !errors.Is(e, arena.ErrSlotLimit) || out.Cell != target.FailedNullRef || target.PendingCell.Object == (ObjectRef{}) {
		t.Fatal("actual fresh NULL/allocated payload before descriptor refusal", e, out)
	}
	original := target.PendingCell
	integerPhysicalAudit(t, target, &out)
	integerPhysicalAudit(t, source, &input)
	serviceMust(t, target.CleanupPendingAllocation())
	e = target.CopyFromOwner(&foreign, source, &input)
	if !errors.Is(e, arena.ErrSlotLimit) || target.PendingCell.Object == original.Object {
		t.Fatal("actual later ObjectRef generation", e)
	}
	replacement := target.PendingCell
	integerPhysicalAudit(t, target, &out, &foreign)
	integerPhysicalAudit(t, source, &input)
	serviceMust(t, target.Del(&out))
	if target.PendingCell != replacement {
		t.Fatal("old caller NULL abort swept new physical backing")
	}
}
