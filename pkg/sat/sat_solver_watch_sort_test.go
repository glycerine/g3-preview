package sat

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func satSortValue(t *testing.T, i, kind uint32) satWatched {
	t.Helper()
	var w satWatched
	var e error
	switch kind {
	case 0:
		w, e = newSATBinaryWatch(Literal(2*i+2), i%2 != 0)
	case 1:
		w, e = newSATTernaryWatch(Literal(2*i+2), Literal(2*i+3))
	case 2:
		w, e = newSATClauseWatch(Literal(2*i+2), satClauseOffset{owner: 0x123456789abcdef0, slot: i, generation: i + 1, storageKind: 1})
	case 3:
		w, e = newSATExternalWatch(3)
	}
	paramsMust(t, e)
	return w
}
func satSortClass(w satWatched) uint32 {
	if w.Binary() {
		return 0
	}
	if w.Ternary() {
		return 1
	}
	return 2
}
func satSortBound(n uint32) uint64 {
	levels := uint64(0)
	for k := uint64(1); k < uint64(n); k *= 2 {
		levels++
	}
	return 4 * uint64(n) * (levels + 1)
}
func TestSATSolverWatchSortLargeStableOrderAndWorkBound(t *testing.T) {
	s, _ := satWatchStoreFixture(t)
	const n = 2048
	r, e := s.NewList()
	paramsMust(t, e)
	input := make([]satWatched, n)
	for i := uint32(0); i < n; i++ {
		kind := uint32(0)
		if i < n/2 {
			kind = 2 + i%2
		}
		input[i] = satSortValue(t, i, kind)
		paramsMust(t, s.Push(r, input[i]))
	}
	var work satWatchSortWork
	paramsMust(t, s.sortByKindWithWork(r, &work))
	out := uint32(0)
	for class := uint32(0); class < 3; class++ {
		for _, w := range input {
			if satSortClass(w) != class {
				continue
			}
			v, e := s.Read(r, out)
			paramsMust(t, e)
			if v != w {
				t.Fatalf("source stable class%d position%d", class, out)
			}
			out++
		}
	}
	if work.Comparisons > 2*n || work.Moves > satSortBound(n) {
		t.Fatalf("source asymptotic bound: actual comparisons%d moves%d; bounds%d/%d", work.Comparisons, work.Moves, 2*n, satSortBound(n))
	}
}
func TestSATSolverWatchSortThreeClassesAndActualLifetime(t *testing.T) {
	s, _ := satWatchStoreFixture(t)
	r, e := s.NewList()
	paramsMust(t, e)
	xs := []satWatched{satSortValue(t, 0, 3), satSortValue(t, 1, 2), satSortValue(t, 2, 0), satSortValue(t, 3, 1), satSortValue(t, 4, 2)}
	for _, w := range xs {
		paramsMust(t, s.Push(r, w))
	}
	found, e := s.Erase(r, xs[0])
	paramsMust(t, e)
	if !found {
		t.Fatal("erase missing")
	}
	it, e := s.Iter(r)
	paramsMust(t, e)
	paramsMust(t, s.SortByKind(r))
	want := []satWatched{xs[2], xs[3], xs[1], xs[4]}
	for _, w := range want {
		v, ok, e := it.Next()
		paramsMust(t, e)
		if !ok || v != w {
			t.Fatal("sorting must retain captured native backing", v)
		}
	}
	paramsMust(t, s.SetEnd(r, 3))
	if _, e = satPropagationCursorRead(&it, 3); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("native current bounds", e)
	}

}
func TestSATSolverWatchSortIndependentMixedOracleAndWarmAllocation(t *testing.T) {
	s, b := satWatchStoreFixture(t)
	r, e := s.NewList()
	paramsMust(t, e)
	seed := uint32(21)
	var xs [513]satWatched
	for i := range xs {
		seed = 1664525*seed + 1013904223
		xs[i] = satSortValue(t, uint32(i), seed>>30)
	}
	for _, w := range xs {
		paramsMust(t, s.Push(r, w))
	}
	var work satWatchSortWork
	paramsMust(t, s.sortByKindWithWork(r, &work))
	out := uint32(0)
	for class := uint32(0); class < 3; class++ {
		for _, w := range xs {
			if satSortClass(w) == class {
				v, e := s.Read(r, out)
				paramsMust(t, e)
				if v != w {
					t.Fatalf("mixed oracle class%d position%d", class, out)
				}
				out++
			}
		}
	}
	if work.Comparisons > 2*uint64(len(xs)) || work.Moves > satSortBound(uint32(len(xs))) {
		t.Fatal(work)
	}
	b.FailAfter(0)
	if n := testing.AllocsPerRun(20, func() {
		for i, w := range xs {
			if e := s.Write(r, uint32(i), w); e != nil {
				panic(e)
			}
		}
		if e := s.SortByKind(r); e != nil {
			panic(e)
		}
	}); n != 0 {
		t.Fatal("warm sort allocation", n)
	}
}
func TestSATSolverWatchSortBoundaryAndGuardPrefixes(t *testing.T) {
	s, _ := satWatchStoreFixture(t)
	r, e := s.NewList()
	paramsMust(t, e)
	var work satWatchSortWork
	paramsMust(t, s.sortByKindWithWork(r, &work))
	if work != (satWatchSortWork{}) {
		t.Fatal("empty work")
	}
	w := satSortValue(t, 1, 3)
	paramsMust(t, s.Push(r, w))
	paramsMust(t, s.sortByKindWithWork(r, &work))
	if work.Moves != 0 {
		t.Fatal("singleton moves")
	}
	for i := 1; i < 200; i++ {
		paramsMust(t, s.Push(r, w))
	}
	paramsMust(t, s.sortByKindWithWork(r, &work))
	if work.Moves != 0 || work.Comparisons > 400 {
		t.Fatal("equivalent classes must not rotate", work)
	}
	paramsMust(t, s.Close())
	if e = s.SortByKind(r); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("closed sorting", e)
	}
}
