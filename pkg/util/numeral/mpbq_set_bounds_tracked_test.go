package numeral

// Source: src/util/basic_interval.h:118-121.

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestBinaryIntervalSetBoundsTrackedAssertionActions(t *testing.T) {
	if !SourceExtendedGCDDebug {
		t.Skip("source ordering assertion erased")
	}
	targetBounds, budget := binaryManager(t)
	target, e := NewBinaryIntervalManager(targetBounds)
	serviceMust(t, e)
	defer target.Close()
	input := BinaryIntervalFromBorrowedBounds(NewBinaryInt(2), NewBinaryIntPower(1, 70))
	out := BinaryIntervalFromBorrowedBounds(NewBinaryInt(9), NewBinaryInt(9))
	d, _ := bindMPNDebugActions(t, "TC")
	ops := target.WithAllocationContext(NumericAllocationContext{})
	if e := ops.SetBoundsTracked(&out, &input.lower, &input.upper, nil); !isMPNVerificationThrow(e) || out.lower.Num.Val != 9 || out.upper.Num.Val != 9 {
		t.Fatal("ordering Throw preceded bounds publication", e, out)
	}
	serviceMust(t, ops.SetBoundsTracked(&out, &input.lower, &input.upper, nil))
	if out != input {
		t.Fatal("ordering Continue skipped literal SET", out, input)
	}
	serviceMust(t, target.Del(&out))
	serviceMust(t, d.EnableAssertions(false))
	input = BinaryIntervalFromBorrowedBounds(NewBinaryInt(3), NewBinaryIntPower(1, 128))
	before := targetBounds.Tmp
	budget.FailAfter(0)
	e = ops.SetBoundsTracked(&out, &input.lower, &input.upper, nil)
	budget.FailAfter(-1)
	serviceMust(t, e)
	if out != input || targetBounds.Tmp != before {
		t.Fatal("disabled ordering assertion evaluated its numeric operand")
	}
	serviceMust(t, target.Del(&out))
	beforeOut := out
	serviceMust(t, d.Close())
	if e := ops.SetBoundsTracked(&out, &input.lower, &input.upper, nil); !errors.Is(e, arena.ErrClosed) || out != beforeOut {
		t.Fatal("ordering assertion switch did not precede SET", e, out)
	}
}
