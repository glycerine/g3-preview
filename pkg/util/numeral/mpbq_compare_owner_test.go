package numeral

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	"math/big"
	"testing"
)

func TestBinaryComparisonFromIndependentOwner(t *testing.T) {
	source, _ := binaryManager(t)
	receiver, _ := binaryManager(t)
	values := []string{"-340282366920938463463374607431768211459", "-2147483648", "-1", "0", "1", "2147483647", "340282366920938463463374607431768211459"}
	for _, x := range values {
		for _, y := range values {
			for _, powers := range [][2]uint32{{0, 0}, {0, 1}, {1, 0}, {31, 64}, {64, 31}, {70, 70}} {
				a, b := NewBinaryRational(), NewBinaryRational()
				serviceMust(t, source.Z.SetString(&a.Num, x))
				serviceMust(t, source.Z.SetString(&b.Num, y))
				a.KField, b.KField = powers[0], powers[1]
				beforeA, beforeB := a, b
				var receipt IntegerCopyAllocationReceipt
				less, e := receiver.WithAllocationContext(NumericAllocationContext{}).LessFromOwnerTracked(source, &a, &b, &receipt)
				serviceMust(t, e)
				ax, _ := new(big.Int).SetString(x, 10)
				by, _ := new(big.Int).SetString(y, 10)
				ax.Lsh(ax, uint(powers[1]))
				by.Lsh(by, uint(powers[0]))
				if less != (ax.Cmp(by) < 0) || a != beforeA || b != beforeB {
					t.Fatal("cross-owner binary comparison", x, y, powers, less)
				}
				serviceMust(t, source.Del(&a))
				serviceMust(t, source.Del(&b))
			}
		}
	}
}

func TestBinaryIntervalCopyFromIndependentOwner(t *testing.T) {
	sourceBounds, _ := binaryManager(t)
	targetBounds, _ := binaryManager(t)
	source, e := NewBinaryIntervalManager(sourceBounds)
	serviceMust(t, e)
	target, e := NewBinaryIntervalManager(targetBounds)
	serviceMust(t, e)
	defer source.Close()
	defer target.Close()
	lo, hi := NewBinaryIntPower(1, 64), NewBinaryIntPower(1, 63)
	var input, out BinaryInterval
	serviceMust(t, source.SetBounds(&input, &lo, &hi))
	var receipt IntegerCopyAllocationReceipt
	serviceMust(t, target.WithAllocationContext(NumericAllocationContext{}).CopyFromOwnerTracked(&out, source, &input, &receipt))
	if out.lower.KField != 64 || out.upper.KField != 63 || out.lower.Num.Val != 1 || out.upper.Num.Val != 1 {
		t.Fatal("foreign interval fields", out)
	}
	serviceMust(t, source.Del(&input))
	serviceMust(t, target.Del(&out))
}

func TestBinaryIntervalForeignCopyAssertionActions(t *testing.T) {
	if !SourceExtendedGCDDebug {
		t.Skip("source ordering assertion erased")
	}
	sourceBounds, _ := binaryManager(t)
	targetBounds, budget := binaryManager(t)
	source, e := NewBinaryIntervalManager(sourceBounds)
	serviceMust(t, e)
	target, e := NewBinaryIntervalManager(targetBounds)
	serviceMust(t, e)
	defer source.Close()
	defer target.Close()
	input := BinaryIntervalFromBorrowedBounds(NewBinaryInt(2), NewBinaryIntPower(1, 70))
	out := BinaryIntervalFromBorrowedBounds(NewBinaryInt(9), NewBinaryInt(9))
	d, _ := bindMPNDebugActions(t, "TC")
	ops := target.WithAllocationContext(NumericAllocationContext{})
	if e := ops.CopyFromOwnerTracked(&out, source, &input, nil); !isMPNVerificationThrow(e) || out.lower.Num.Val != 9 || out.upper.Num.Val != 9 {
		t.Fatal("ordering Throw preceded bounds publication", e, out)
	}
	serviceMust(t, ops.CopyFromOwnerTracked(&out, source, &input, nil))
	if out != input {
		t.Fatal("ordering Continue skipped literal SET", out, input)
	}
	serviceMust(t, target.Del(&out))
	serviceMust(t, d.EnableAssertions(false))
	input = BinaryIntervalFromBorrowedBounds(NewBinaryInt(3), NewBinaryIntPower(1, 128))
	before := targetBounds.Tmp
	budget.FailAfter(0)
	e = ops.CopyFromOwnerTracked(&out, source, &input, nil)
	budget.FailAfter(-1)
	serviceMust(t, e)
	if out != input || targetBounds.Tmp != before {
		t.Fatal("disabled ordering assertion evaluated its numeric operand")
	}
	serviceMust(t, target.Del(&out))
	beforeOut := out
	serviceMust(t, d.Close())
	if e := ops.CopyFromOwnerTracked(&out, source, &input, nil); !errors.Is(e, arena.ErrClosed) || out != beforeOut {
		t.Fatal("ordering assertion switch did not precede SET", e, out)
	}
}

func TestBinaryForeignComparisonGrowthUsesCallerRoots(t *testing.T) {
	source, _ := binaryManager(t)
	budget := arena.NewBudget(1 << 24)
	diagnostics, e := NewRationalService(serviceConfig(budget))
	serviceMust(t, e)
	z, e := NewIntegerManager(IntegerConfig{Cells: arena.Config{PageSize: 8, SlotLimit: 8, Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	serviceMust(t, e)
	receiver, e := NewBinaryRationalManager(z, BinaryRationalConfig{Diagnostics: diagnostics})
	serviceMust(t, e)
	t.Cleanup(func() {
		serviceMust(t, receiver.CloseChecked())
		serviceMust(t, z.CloseChecked())
		serviceMust(t, diagnostics.Close())
		if budget.Used() != 0 {
			t.Error("receiving comparison roots leaked", budget.Used())
		}
	})
	policy := numericPolicyTestOwner{manager: z, active: true}
	hooks := NumericInvocationHooks{Policy: &policy}
	ops := receiver.WithAllocationContext(NumericAllocationContext{}.WithCaller(&hooks))
	a, b := NewBinaryInt(1), NewBinaryIntPower(1, 128)
	var copy IntegerCopyAllocationReceipt
	_, body := ops.LessFromOwnerTracked(source, &a, &b, &copy)
	if !errors.Is(body, arena.ErrSlotLimit) || !NumericAllocationPending(&policy.roots) || policy.roots.Cell.Object == (umemory.ObjectRef{}) || z.MatchesCopyAllocationInvocation(&copy) {
		t.Fatal("shift growth was not separate from the completed scalar copy", body)
	}
	serviceMust(t, policy.roots.Cleanup(z))
	serviceMust(t, z.CleanupRawNumeral(&receiver.Tmp))
	policy.active = false
	if _, e := ops.LessFromOwnerTracked(source, &a, &b, &copy); !errors.Is(e, ErrNumericOwner) {
		t.Fatal("expired numeric caller context was accepted", e)
	}
}

func TestBinaryIntervalForeignCopyRetainsLowerBeforeUpperRefusal(t *testing.T) {
	sourceBounds, _ := binaryManager(t)
	source, e := NewBinaryIntervalManager(sourceBounds)
	serviceMust(t, e)
	defer source.Close()
	input := BinaryIntervalFromBorrowedBounds(NewBinaryIntPower(1, 70), NewBinaryIntPower(0, 70))
	digits := make([]uint32, 65)
	digits[0], digits[64] = 17, 1
	serviceMust(t, sourceBounds.Z.SetLimbs(&input.upper.Num, digits))
	budget := arena.NewBudget(1 << 24)
	diagnostics, e := NewRationalService(serviceConfig(budget))
	serviceMust(t, e)
	z, e := NewIntegerManager(IntegerConfig{Cells: arena.Config{PageSize: 8, SlotLimit: 8, Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}}, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	serviceMust(t, e)
	bounds, e := NewBinaryRationalManager(z, BinaryRationalConfig{Diagnostics: diagnostics})
	serviceMust(t, e)
	target, e := NewBinaryIntervalManager(bounds)
	serviceMust(t, e)
	t.Cleanup(func() {
		target.Close()
		serviceMust(t, bounds.CloseChecked())
		serviceMust(t, z.CloseChecked())
		serviceMust(t, diagnostics.Close())
		if budget.Used() != 0 {
			t.Error("interval receiving owner leaked", budget.Used())
		}
	})
	out := BinaryIntervalFromBorrowedBounds(NewBinaryIntPower(1, 7), NewBinaryIntPower(2, 7))
	var receipt IntegerCopyAllocationReceipt
	body := target.WithAllocationContext(NumericAllocationContext{}).CopyFromOwnerTracked(&out, source, &input, &receipt)
	if !errors.Is(body, arena.ErrSlotLimit) || !z.MatchesCopyAllocationInvocation(&receipt) || out.lower.Num.Val != 1 || out.lower.KField != 70 || out.upper.KField != 7 {
		t.Fatal("upper refusal lost lower SET or published upper exponent", body, out)
	}
	integerPhysicalAudit(t, z, &out.lower.Num, &out.upper.Num)
	serviceMust(t, z.CleanupCopyAllocationInvocation(&receipt))
	serviceMust(t, z.CleanupRawNumeral(&out.upper.Num))
	serviceMust(t, target.Del(&out))
	serviceMust(t, source.Del(&input))
}
