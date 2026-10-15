package sat

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/z3exception"
)

func TestSATSolverDiffLevelsSourceWidthAndRetainedSlots(t *testing.T) {
	b := arena.NewBudget(^uint64(0))
	var c satVariableCharColumn
	for i := int8(0); i < 4; i++ {
		satOK(t, c.push(b, i))
	}
	if cap(c.data) != 5 {
		t.Fatal(cap(c.data))
	}
	if b.Used() != 5 {
		t.Fatal(b.Used())
	}
	satOK(t, c.shrink(1))
	v, e := c.read(3)
	if satStorageSourceDebug {
		if e != ErrSATAssertion {
			t.Fatal(v, e)
		}
	} else {
		satOK(t, e)
		if v != 3 {
			t.Fatal(v)
		}
	}
	if _, e = c.read(4); e == nil {
		t.Fatal("never initialized spare slot")
	}
	c.finalize(b)
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
func TestSATSolverVariableColumnGrowthFailureAndScalarBounds(t *testing.T) {
	b := arena.NewBudget(^uint64(0))
	var c satVariableCharColumn
	satOK(t, c.push(b, 11))
	satOK(t, c.push(b, 12))
	b.FailAfter(0)
	if e := c.push(b, 13); !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	if len(c.data) != 2 || cap(c.data) != 2 || c.initialized != 2 || c.data[0] != 11 || b.Used() != 2 {
		t.Fatal(c, b.Used())
	}
	b.FailAfter(-1)
	c.finalize(b)
	for _, s := range []uint32{1, 4, 8} {
		n, bytes, e := satVariableNextCapacity(2, 3, uint64(s), ^uint64(0))
		satOK(t, e)
		if n != 3 || bytes != 9 {
			t.Fatal(n, bytes)
		}
	}
	if _, _, e := satVariableNextCapacity(8, 1, 1, 11); e != arena.ErrBounds {
		t.Fatal(e)
	}
	if _, _, e := satVariableNextCapacity(^uint32(0), 1, 1, ^uint64(0)); e == nil {
		t.Fatal("source overflow")
	} else {
		var source z3exception.DefaultException
		if !errors.As(e, &source) || source.Message() != "Overflow encountered when expanding vector" {
			t.Fatal("source class/message", e)
		}
	}
	// Literal uint32 extent admits this wrapped growth, but the source's
	// undersized allocation cannot represent the full typed backing domain.
	if _, _, e := satVariableNextCapacity(1<<30, 4, 4, ^uint64(0)); e != arena.ErrBounds {
		t.Fatal("separate full-capacity domain", e)
	}
}
