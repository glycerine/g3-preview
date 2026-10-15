package sat

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestSATSolverDiffLevelsNativeBounds(t *testing.T) {
	b := arena.NewBudget(^uint64(0))
	var c satVariableCharColumn
	b.FailAfter(0)
	c.extend(4)
	for i := int8(0); i < 4; i++ {
		satOK(t, c.write(uint32(i), i))
	}
	if len(c.data) != 4 || b.Used() != 0 {
		t.Fatal(b.Used())
	}
	v, e := c.read(3)
	satOK(t, e)
	if v != 3 {
		t.Fatal(v)
	}
	want := error(arena.ErrBounds)
	if satStorageSourceDebug {
		want = ErrSATAssertion
	}
	if _, e = c.read(4); e != want || c.write(4, 7) != want {
		t.Fatal("logical bounds", e)
	}
	c.finalize()
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
func TestSATSolverDiffLevelsNativeColdGrowth(t *testing.T) {
	b := arena.NewBudget(^uint64(0))
	var c satVariableCharColumn
	b.FailAfter(0)
	c.extend(129)
	for i := 0; i < 129; i++ {
		satOK(t, c.write(uint32(i), int8(i)))
	}
	if len(c.data) != 129 || c.data[0] != 0 || c.data[128] != -128 || b.Used() != 0 {
		t.Fatal(c, b.Used())
	}
	b.FailAfter(-1)
	c.finalize()
}
