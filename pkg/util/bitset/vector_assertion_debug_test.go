//go:build g3debug

package bitset

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

// Source: src/util/bit_vector.cpp:106,141,168. A closed borrowed process
// owner proves that each source SASSERT gate is reached at its exact prefix.
func TestBitVectorSourceAssertionGateOrder(t *testing.T) {
	budget := arena.NewBudget(^uint64(0))
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: budget}})
	if e != nil {
		t.Fatal(e)
	}
	d, e := diagnostic.NewDebugContext(region, budget)
	if e != nil {
		t.Fatal(e)
	}
	old, e := diagnostic.BindProcessDebugContext(d)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if _, e := diagnostic.BindProcessDebugContext(old); e != nil {
			t.Error(e)
		}
		if e := region.Close(); e != nil {
			t.Error(e)
		}
		if budget.Used() != 0 {
			t.Error("retained debug owner", budget.Used())
		}
	})
	if e := d.Close(); e != nil {
		t.Fatal(e)
	}

	shifted := bv(1, 0)
	if e := shifted.ShiftRight(33); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("shift assertion gate", e)
	}
	if shifted.size != 34 || shifted.words[0] != 0 || shifted.words[1] != 1 {
		t.Fatal("shift gate must follow resize and whole-word move", shifted)
	}
	if e := shifted.ShiftRight(0); e != nil {
		t.Fatal("zero shift must return before assertion", e)
	}

	left := bv(1, 0)
	right := bv(35, 34)
	if e := left.Or(right); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("OR assertion gate", e)
	}
	if left.size != 35 || left.words[1] != 0 || left.words[0] != 1 {
		t.Fatal("OR gate must follow resize but precede union", left)
	}

	mask := bv(1, 0)
	if e := left.And(mask); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("AND assertion gate", e)
	}
	if left.size != 35 || left.words[0] != 1 {
		t.Fatal("AND gate must precede intersection", left)
	}
	if e := new(Vector).And(left); e != nil {
		t.Fatal("empty AND returns before assertion", e)
	}
	if e := mask.And(right); e != nil {
		t.Fatal("longer-source AND bypasses assertion", e)
	}
}
