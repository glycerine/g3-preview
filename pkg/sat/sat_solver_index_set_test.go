package sat

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func satNativeIndexValues(t *testing.T, s *satSolverIndexSet) []uint32 {
	t.Helper()
	it, e := s.capture()
	satOK(t, e)
	var values []uint32
	for {
		v, e := it.next()
		satOK(t, e)
		if !v.present {
			return values
		}
		values = append(values, v.value)
	}
}
func TestSATSolverConsequenceNativeIndexOrderAndDeepMapCopy(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	var set satSolverIndexSet
	defer set.close()
	want := []uint32{17, 1, 9}
	for _, key := range want {
		satOK(t, set.insert(key))
	}
	if got := satNativeIndexValues(t, &set); !reflect.DeepEqual(got, want) {
		t.Fatal(got, want)
	}
	satOK(t, s.antecedents.insert(7, &set))
	satOK(t, set.reset())
	stored, e := s.antecedents.find(7)
	satOK(t, e)
	if got := satNativeIndexValues(t, stored); !reflect.DeepEqual(got, want) {
		t.Fatal("deep copy", got)
	}
	satOK(t, set.insert(100))
	yes, e := stored.contains(100)
	satOK(t, e)
	if yes {
		t.Fatal("map borrowed caller dictionary")
	}
	satFullSolverRuntimeAudit(t, s)
}

func TestSATSolverConsequenceNativeMapGrowthAndReplacement(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	f, e := s.consequenceFrame()
	satOK(t, e)
	f.indexSet()
	defer func() { satOK(t, s.consequenceDrop(f, nil)) }()
	before := s.budget.Used()
	runtimeBefore := s.runtime.memory.Stats().LiveBlocks
	s.budget.FailAfter(0)
	for k := uint32(0); k < 257; k++ {
		satOK(t, f.index.reset())
		satOK(t, f.index.insert(k))
		satOK(t, f.index.insert(k+1000))
		satOK(t, s.antecedents.insert(k, &f.index))
	}
	for k := uint32(0); k < 257; k++ {
		row, e := s.antecedents.find(k)
		satOK(t, e)
		if got := satNativeIndexValues(t, row); !reflect.DeepEqual(got, []uint32{k, k + 1000}) {
			t.Fatal(k, got)
		}
	}
	row, e := s.antecedents.find(0)
	satOK(t, e)
	it, e := row.capture()
	satOK(t, e)
	satOK(t, f.index.reset())
	satOK(t, f.index.insert(17))
	satOK(t, f.index.insert(29))
	satOK(t, s.antecedents.insert(0, &f.index))
	replaced, e := s.antecedents.find(0)
	satOK(t, e)
	if replaced != row || len(s.antecedents.entries) != 257 {
		t.Fatal("replacement changed map identity or size")
	}
	if _, e = it.next(); !errors.Is(e, ErrSATOwner) {
		t.Fatal("assignment retained old iterator", e)
	}
	if got := satNativeIndexValues(t, replaced); !reflect.DeepEqual(got, []uint32{17, 29}) {
		t.Fatal(got)
	}
	satOK(t, s.antecedents.insert(0, replaced)) // Actual self assignment keeps the row.
	if got := satNativeIndexValues(t, replaced); !reflect.DeepEqual(got, []uint32{17, 29}) {
		t.Fatal(got)
	}
	satOK(t, s.antecedents.reset())
	satOK(t, s.antecedents.reset())
	if s.budget.Used() != before || s.runtime.memory.Stats().LiveBlocks != runtimeBefore {
		t.Fatal("native backing charged custom memory")
	}
	satFullSolverRuntimeAudit(t, s)
	s.budget.FailAfter(-1)
}

func TestSATSolverConsequenceNativeSetUnionAssignmentOracle(t *testing.T) {
	var sets [2]satSolverIndexSet
	var orders [2][]uint32
	var membership [2]map[uint32]bool
	membership[0], membership[1] = make(map[uint32]bool), make(map[uint32]bool)
	add := func(i int, k uint32) {
		if !membership[i][k] {
			membership[i][k] = true
			orders[i] = append(orders[i], k)
		}
	}
	seed := uint32(0x392a53)
	for step := 0; step < 4096; step++ {
		seed = seed*1664525 + 1013904223
		dst, src := int(seed>>31), int((seed>>30)&1)
		switch seed % 13 {
		case 0:
			satOK(t, sets[dst].reset())
			clear(membership[dst])
			orders[dst] = nil
		case 1, 2:
			satOK(t, sets[dst].assign(&sets[src]))
			if dst != src {
				clear(membership[dst])
				orders[dst] = nil
				for _, k := range orders[src] {
					add(dst, k)
				}
			}
		case 3, 4:
			satOK(t, sets[dst].union(&sets[src]))
			if dst != src {
				for _, k := range orders[src] {
					add(dst, k)
				}
			}
		default:
			k := (seed >> 8) % 257
			satOK(t, sets[dst].insert(k))
			add(dst, k)
		}
		for i := range sets {
			if got := satNativeIndexValues(t, &sets[i]); !reflect.DeepEqual(got, orders[i]) {
				t.Fatal("ordered oracle", step, i, got, orders[i])
			}
			for _, k := range []uint32{0, 17, 29, 256, 257, ^uint32(0)} {
				yes, e := sets[i].contains(k)
				satOK(t, e)
				if yes != membership[i][k] {
					t.Fatal("membership oracle", step, i, k, yes)
				}
			}
		}
	}
	for i := range sets {
		satOK(t, sets[i].close())
	}
	if e := sets[0].union(&sets[1]); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("closed set accepted", e)
	}
}
