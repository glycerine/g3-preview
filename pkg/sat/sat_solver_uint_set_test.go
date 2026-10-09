// Source: src/sat/sat_types.h:153-204; src/sat/sat_model_converter.cpp.
package sat

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"reflect"
	"testing"
)

func satSolverSetOwner(t *testing.T) (*satSolverUintSet, *arena.Budget) {
	t.Helper()
	b := arena.NewBudget(^uint64(0))
	s, e := newSATSolverUintSet(b)
	paramsMust(t, e)
	t.Cleanup(func() {
		paramsMust(t, s.Close())
		if b.Used() != 0 {
			t.Fatal("set retained", b.Used())
		}
	})
	return s, b
}

func TestSATSolverUintSetNativeGrowthAndReset(t *testing.T) {
	s, b := satSolverSetOwner(t)
	used := b.Used()
	b.FailAfter(0)
	defer b.FailAfter(-1)
	for _, v := range []uint32{400, 1, 7, 400, 0} {
		paramsMust(t, s.Insert(v))
	}
	if !reflect.DeepEqual(s.values, []uint32{400, 1, 7, 0}) || b.Used() != used {
		t.Fatal("unique insertion order/native growth", s.values)
	}
	membershipCap, valuesCap := cap(s.inSet), cap(s.values)
	paramsMust(t, s.Reset())
	if s.Size() != 0 || s.Contains(400) || cap(s.inSet) != membershipCap || cap(s.values) != valuesCap {
		t.Fatal("reset lost capacity or membership")
	}
	paramsMust(t, s.Insert(7))
	paramsMust(t, s.Insert(400))
	if s.Contains(1) || !reflect.DeepEqual(s.values, []uint32{7, 400}) {
		t.Fatal("reset resurrected old membership", s.values)
	}
	paramsMust(t, s.Finalize())
	if s.values != nil || s.inSet != nil {
		t.Fatal("finalize retained backing")
	}
	paramsMust(t, s.Insert(1))
	if !s.Contains(1) || s.Size() != 1 {
		t.Fatal("set not reusable")
	}
	if e := s.Insert(^uint32(0)); !errors.Is(e, arena.ErrBounds) || s.Size() != 1 {
		t.Fatal("wrapped index mutated set", e)
	}
}

func TestSATSolverUintSetIndependentNativeModel(t *testing.T) {
	s, _ := satSolverSetOwner(t)
	seen := make(map[uint32]bool)
	var order []uint32
	state := uint32(17)
	for i := 0; i < 2000; i++ {
		state = state*1664525 + 1013904223
		v := state % 256
		if i%47 == 0 {
			paramsMust(t, s.Reset())
			clear(seen)
			order = order[:0]
		}
		paramsMust(t, s.Insert(v))
		if !seen[v] {
			order = append(order, v)
			seen[v] = true
		}
		if !reflect.DeepEqual(s.values, order) {
			t.Fatal("independent order", i, s.values, order)
		}
		for candidate := uint32(0); candidate < 256; candidate++ {
			if s.Contains(candidate) != seen[candidate] {
				t.Fatal("independent membership", i, candidate)
			}
		}
	}
}

func TestSATSolverModelConverterActualCollectVars(t *testing.T) {
	c, _ := satConverterTestOwner(t)
	s, b := satSolverSetOwner(t)
	paramsMust(t, s.Insert(2))
	for _, v := range []BoolVar{2, 8, 1} {
		_, e := c.Mk(satModelBlockLit, v)
		paramsMust(t, e)
	}
	used := b.Used()
	b.FailAfter(0)
	defer b.FailAfter(-1)
	paramsMust(t, c.CollectVars(s))
	if !reflect.DeepEqual(s.values, []uint32{2, 8, 1}) || b.Used() != used {
		t.Fatal("source traversal/prepopulated dedup", s.values)
	}
	paramsMust(t, s.Reset())
	paramsMust(t, c.CollectVars(s))
	if !reflect.DeepEqual(s.values, []uint32{2, 8, 1}) {
		t.Fatal("repeated collection", s.values)
	}
}
