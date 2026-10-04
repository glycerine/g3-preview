// Source: src/util/mpz.h:662-672; src/util/mpz.cpp:1186-1213.
// Licensed under the MIT license. Author: Leonardo de Moura.
package mpnum

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
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
			serviceMust(t, target.Copy(&out, &target.two64))
			old := target.cell(out.cell).backing
			inputToken := input
			sourceState := source.memory.State()
			sourceBytes := source.budget.Used()
			defer func() {
				budget.FailAfter(-1)
				serviceMust(t, target.CleanupPendingAllocation())
				serviceMust(t, target.allocator.CleanupPending())
				serviceMust(t, target.Del(&out))
				serviceMust(t, source.Del(&input))
			}()
			budget.FailAfter(ordinal)
			e := target.CopyFromOwner(&out, source, &input)
			integerPhysicalAudit(t, target, &out)
			integerPhysicalAudit(t, source, &input)
			if input != inputToken || source.memory.State() != sourceState || source.budget.Used() != sourceBytes {
				t.Fatal("source owner modified")
			}
			if e != nil {
				if !numericPhysicalReservationRefusal(e) {
					t.Fatal("actual target refusal", ordinal, e)
				}
				if out.val != -1 || out.cell != target.failedRef {
					t.Fatal("literal destructive sign/old-free/dead-pointer publication", out)
				}
				if _, x := target.memory.Size(old.raw); x == nil {
					t.Fatal("old target physical block survived source SET growth")
				}
			} else {
				success = true
				if target.cell(out.cell).capacity != source.cell(input.cell).capacity || target.cell(out.cell).size != 65 {
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
	target, e := NewIntegerManager(IntegerConfig{Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}, Cells: arena.Config{PageSize: 8, SlotLimit: 8, Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}})
	serviceMust(t, e)
	var input, out, foreign Int
	serviceMust(t, source.Copy(&input, &source.two64))
	defer func() {
		serviceMust(t, target.CleanupPendingAllocation())
		serviceMust(t, target.allocator.CleanupPending())
		serviceMust(t, target.Del(&out))
		serviceMust(t, target.Del(&foreign))
		serviceMust(t, source.Del(&input))
		serviceMust(t, target.CloseChecked())
		if budget.Used() != 0 {
			t.Fatal("target physical graph refund", budget.Used())
		}
	}()
	e = target.CopyFromOwner(&out, source, &input)
	if !errors.Is(e, arena.ErrSlotLimit) || out.cell != target.failedNullRef || target.pendingCell.object == (memory.ObjectRef{}) {
		t.Fatal("actual fresh NULL/allocated payload before descriptor refusal", e, out)
	}
	original := target.pendingCell
	integerPhysicalAudit(t, target, &out)
	integerPhysicalAudit(t, source, &input)
	serviceMust(t, target.CleanupPendingAllocation())
	e = target.CopyFromOwner(&foreign, source, &input)
	if !errors.Is(e, arena.ErrSlotLimit) || target.pendingCell.object == original.object {
		t.Fatal("actual later ObjectRef generation", e)
	}
	replacement := target.pendingCell
	integerPhysicalAudit(t, target, &out, &foreign)
	integerPhysicalAudit(t, source, &input)
	serviceMust(t, target.Del(&out))
	if target.pendingCell != replacement {
		t.Fatal("old caller NULL abort swept new physical backing")
	}
}
