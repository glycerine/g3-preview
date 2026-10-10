// Real mpn.h member sbuffers and mpn.cpp division storage action ordering.
package numeral

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestNaturalMandatoryInlineConstructorAndRuntime(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	n, e := NewNaturalManager(arena.VectorConfig{Descriptors: arena.Config{Budget: b}})
	if e != nil {
		t.Fatal(e)
	}
	runtime := n.RuntimeOwner()
	if runtime == nil || runtime.Stats().LiveBlocks != 0 {
		t.Fatal("inline construction allocated division payload")
	}
	for i := range n.Division {
		if n.Division[i] != nil {
			t.Fatal("eager division scratch", i)
		}
	}
	if e = n.CloseChecked(); e != nil {
		t.Fatal(e)
	}
	if runtime.Stats().TotalBytes != 0 || b.Used() != 0 {
		t.Fatal("actual owned runtime/header leak")
	}
}

func TestNaturalMandatoryIntegerBorrowsActualRuntime(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	z, e := NewIntegerManager(IntegerConfig{Cells: arena.Config{Budget: b}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}})
	if e != nil {
		t.Fatal(e)
	}
	if z.Natural.RuntimeOwner() != z.RuntimeOwner() {
		t.Fatal("disconnected numeric Natural runtime")
	}
	if e = z.CloseChecked(); e != nil {
		t.Fatal(e)
	}
	if b.Used() != 0 {
		t.Fatal("shared child ownership leak")
	}
}

func TestNaturalMandatoryActualOverflowAndShrink(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	n, e := NewNaturalManager(arena.VectorConfig{Descriptors: arena.Config{Budget: b}})
	if e != nil {
		t.Fatal(e)
	}
	r := n.RuntimeOwner()
	if e = n.ResizeDiv(0, 17); e != nil {
		t.Fatal(e)
	}
	if len(n.Division[0]) != 17 || r.Stats().LiveBlocks != 0 {
		t.Fatal("native division workspace")
	}
	for _, v := range n.Division[0] {
		if v != 0 {
			t.Fatal("new scratch not zeroed")
		}
	}
	backing := &n.Division[0][0]
	before, charge := r.Stats(), b.Used()
	if e = n.ResizeDiv(0, 4); e != nil {
		t.Fatal(e)
	}
	if &n.Division[0][0] != backing || len(n.Division[0]) != 4 || r.Stats() != before || b.Used() != charge {
		t.Fatal("shrink changed workspace")
	}
	if e = n.CloseChecked(); e != nil {
		t.Fatal(e)
	}
	if b.Used() != 0 {
		t.Fatal("overflow destructor leak")
	}
}

func naturalDivisionCapacities(m *NaturalManager) (capacities [4]uint32) {
	for i := range m.Division {
		capacities[i] = uint32(cap(m.Division[i]))
	}
	return
}
