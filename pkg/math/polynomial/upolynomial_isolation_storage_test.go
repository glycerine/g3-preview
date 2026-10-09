package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestUPolynomialIsolationPackedFrameFields(t *testing.T) {
	for _, f := range []struct {
		size uint32
		left bool
		want uint32
	}{{0, false, 0}, {7, true, 7}, {1 << 30, true, 0}, {^uint32(0), false, (1 << 30) - 1}} {
		frame := newUPolyDRSFrame(^uint32(0), f.size, f.left)
		if frame.parent != ^uint32(0) || frame.polynomialSize() != f.want || !frame.firstVisit() || frame.leftChild() != f.left {
			t.Fatal(f, frame)
		}
	}

}

func TestUPolynomialBinaryVectorActualAscendingDestruction(t *testing.T) {
	b, budget := binaryManager(t)
	v, e := numericBinaryRationalManager_NewVector(b)
	serviceMust(t, e)
	var input util.Int
	serviceMust(t, b.Integers().SetLimbs(&input, []uint32{37, 11, 1}))
	a := util.NewBinaryRational()
	serviceMust(t, b.SetIntegerPower(&a, &input, 9))
	serviceMust(t, b.Integers().Del(&input))
	for i := 0; i < 3; i++ {
		serviceMust(t, v.PushCopy(&a))
	}
	serviceMust(t, b.Del(&a))
	slots := [3]arena.Slot{}
	for i := range slots {
		slots[i] = util.NumericSlot((*v.state.values[i].BorrowNumerator()).Cell)
	}
	serviceMust(t, v.Reset())
	var probes [3]util.Int
	for i := range probes {
		serviceMust(t, b.Integers().SetLimbs(&probes[i], []uint32{37, 11, 1}))
		if util.NumericSlot((probes[i]).Cell) != slots[2-i] {
			t.Fatal("actual ascending reset", slots, util.NumericSlot((probes[i]).Cell))
		}
	}
	for i := range probes {
		serviceMust(t, b.Integers().Del(&probes[i]))
	}
	serviceMust(t, v.Close())
	budget.FailAfter(0)
	var problem error
	allocs := testing.AllocsPerRun(20, func() {
		owner, e := numericBinaryRationalManager_NewVector(b)
		if owner != nil || !errors.Is(e, arena.ErrInjectedFailure) {
			problem = e
		}
	})
	if problem != nil || allocs != 0 {
		t.Fatal("cold header refusal before host allocation", problem, allocs)
	}
	budget.FailAfter(-1)
}

func TestUPolynomialIsolationSSPopAndDestructorActualOrder(t *testing.T) {
	b, _ := binaryManager(t)
	var s upolySSStack
	var input util.Int
	serviceMust(t, b.Integers().SetLimbs(&input, []uint32{17, 7, 1}))
	a := util.NewBinaryRational()
	serviceMust(t, b.SetInteger(&a, &input))
	serviceMust(t, b.Integers().Del(&input))
	var slots [6]arena.Slot
	for i := 0; i < 3; i++ {
		f, e := numericBinaryRationalManager_pushSS(b, &s)
		serviceMust(t, e)
		serviceMust(t, b.Copy(&f.lower, &a))
		serviceMust(t, b.Copy(&f.upper, &a))
		slots[2*i] = util.NumericSlot((*f.lower.BorrowNumerator()).Cell)
		slots[2*i+1] = util.NumericSlot((*f.upper.BorrowNumerator()).Cell)
	}
	serviceMust(t, b.Del(&a))
	serviceMust(t, numericBinaryRationalManager_popSS(b, &s))
	var popped [2]util.Int
	for i := range popped {
		serviceMust(t, b.Integers().SetLimbs(&popped[i], []uint32{17, 7, 1}))
		if util.NumericSlot((popped[i]).Cell) != slots[5-i] {
			t.Fatal("source pop lower then upper", i, slots, util.NumericSlot((popped[i]).Cell))
		}
	}
	numericBinaryRationalManager_dropSS(b, &s)
	var remainder [4]util.Int
	for i := range remainder {
		serviceMust(t, b.Integers().SetLimbs(&remainder[i], []uint32{17, 7, 1}))
		if util.NumericSlot((remainder[i]).Cell) != slots[3-i] {
			t.Fatal("ascending frame destructor lower then upper", i, slots, util.NumericSlot((remainder[i]).Cell))
		}
	}
	for i := range popped {
		serviceMust(t, b.Integers().Del(&popped[i]))
	}
	for i := range remainder {
		serviceMust(t, b.Integers().Del(&remainder[i]))
	}
}
