// Licensed under the MIT license.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"testing"
)

func TestUPolynomialIsolationPackedFrameAndScalarExtents(t *testing.T) {
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
	for _, stride := range []uint32{8, 24, 56} {
		n, oldExtent, newExtent, e := upolyIsolationSourceCapacity(0, stride)
		if e != nil || n != 2 || oldExtent != 0 || newExtent != 2*stride+8 {
			t.Fatal(stride, n, oldExtent, newExtent, e)
		}
		n, _, _, e = upolyIsolationSourceCapacity(2, stride)
		if e != nil || n != 3 {
			t.Fatal(stride, n, e)
		}
		if _, _, e := upolyIsolationNextCapacity(2, stride, uint64(stride), uint64(stride)*3-1); !errors.Is(e, arena.ErrBounds) {
			t.Fatal("actual host extent", stride, e)
		}
	}
	// Independently fixed unsigned source overflow values, no giant allocation.
	n, oldExtent, newExtent, e := upolyIsolationSourceCapacity(715827883, 8)
	if n != 1073741825 || oldExtent != 1431655776 || newExtent != 16 || !errors.Is(e, arena.ErrBounds) {
		t.Fatal(n, oldExtent, newExtent, e)
	}
}

func TestUPolynomialBinaryVectorActualAscendingDestruction(t *testing.T) {
	b, budget := binaryManager(t)
	v, e := b.NewVector()
	serviceMust(t, e)
	var input Int
	serviceMust(t, b.z.SetLimbs(&input, []uint32{37, 11, 1}))
	a := NewBinaryRational()
	serviceMust(t, b.SetIntegerPower(&a, &input, 9))
	serviceMust(t, b.z.Del(&input))
	for i := 0; i < 3; i++ {
		serviceMust(t, v.PushCopy(&a))
	}
	serviceMust(t, b.Del(&a))
	slots := [3]arena.Slot{}
	for i := range slots {
		slots[i] = numericSlot(v.state.values[i].num.cell)
	}
	serviceMust(t, v.Reset())
	var probes [3]Int
	for i := range probes {
		serviceMust(t, b.z.SetLimbs(&probes[i], []uint32{37, 11, 1}))
		if numericSlot(probes[i].cell) != slots[2-i] {
			t.Fatal("actual ascending reset", slots, numericSlot(probes[i].cell))
		}
	}
	for i := range probes {
		serviceMust(t, b.z.Del(&probes[i]))
	}
	serviceMust(t, v.Close())
	budget.FailAfter(0)
	var problem error
	allocs := testing.AllocsPerRun(20, func() {
		owner, e := b.NewVector()
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
	var input Int
	serviceMust(t, b.z.SetLimbs(&input, []uint32{17, 7, 1}))
	a := NewBinaryRational()
	serviceMust(t, b.SetInteger(&a, &input))
	serviceMust(t, b.z.Del(&input))
	var slots [6]arena.Slot
	for i := 0; i < 3; i++ {
		f, e := b.pushSS(&s)
		serviceMust(t, e)
		serviceMust(t, b.Copy(&f.lower, &a))
		serviceMust(t, b.Copy(&f.upper, &a))
		slots[2*i] = numericSlot(f.lower.num.cell)
		slots[2*i+1] = numericSlot(f.upper.num.cell)
	}
	serviceMust(t, b.Del(&a))
	serviceMust(t, b.popSS(&s))
	var popped [2]Int
	for i := range popped {
		serviceMust(t, b.z.SetLimbs(&popped[i], []uint32{17, 7, 1}))
		if numericSlot(popped[i].cell) != slots[5-i] {
			t.Fatal("source pop lower then upper", i, slots, numericSlot(popped[i].cell))
		}
	}
	b.dropSS(&s)
	var remainder [4]Int
	for i := range remainder {
		serviceMust(t, b.z.SetLimbs(&remainder[i], []uint32{17, 7, 1}))
		if numericSlot(remainder[i].cell) != slots[3-i] {
			t.Fatal("ascending frame destructor lower then upper", i, slots, numericSlot(remainder[i].cell))
		}
	}
	for i := range popped {
		serviceMust(t, b.z.Del(&popped[i]))
	}
	for i := range remainder {
		serviceMust(t, b.z.Del(&remainder[i]))
	}
}
