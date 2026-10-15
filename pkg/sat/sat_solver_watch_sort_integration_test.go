package sat

import "testing"

// Source: src/sat/sat_solver.cpp:1207-1224,1251-1255. The simplifier runs
// before sort_watch_lits and can remove a clause watch; the remaining binary
// and ternary watches must then be sorted by the real solver path.
func TestSATSolverSimplifySortsRetainedWatchLists(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	for i := 0; i < 7; i++ {
		_, e := s.MkVar(true, true)
		satOK(t, e)
	}
	for _, clause := range [][]Literal{{0, 2, 4, 6}, {0, 8, 10}, {0, 12}} {
		satOK(t, s.MkClause(clause))
	}
	w, e := s.Variables.watches.Get(1)
	satOK(t, e)
	classes := func() []uint32 {
		n, e := w.Size()
		satOK(t, e)
		got := make([]uint32, n)
		for i := uint32(0); i < n; i++ {
			v, e := w.Read(i)
			satOK(t, e)
			got[i] = satSortClass(v)
		}
		return got
	}
	if got := classes(); len(got) != 3 || got[0] != 2 || got[1] != 1 || got[2] != 0 {
		t.Fatalf("unsorted source watch list: %v", got)
	}
	s.nextSimplify = 0
	satOK(t, s.simplifyProblem())
	if got := classes(); len(got) != 2 || got[0] != 0 || got[1] != 1 {
		t.Fatalf("real simplify watch order: %v, want binary then ternary", got)
	}
}
