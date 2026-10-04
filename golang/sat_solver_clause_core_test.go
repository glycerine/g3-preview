// Licensed under the MIT license.
// Source: sat_solver.cpp:367–465. Independent finite clause/watch oracles.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"reflect"
	"testing"
)

func TestSATSolverClauseCoreFiniteSimplification(t *testing.T) {
	// Three variable assignments, all root/nonroot level subsets, and every
	// ordered clause of up to three literals over these variables.
	for assignments := 0; assignments < 27; assignments++ {
		var values [6]lbool
		code := assignments
		for v := 0; v < 3; v++ {
			value := []lbool{lFalse, lUndef, lTrue}[code%3]
			code /= 3
			values[2*v] = value
			values[2*v+1] = value.negated()
		}
		for levels := 0; levels < 8; levels++ {
			var lvls [3]uint32
			for v := 0; v < 3; v++ {
				lvls[v] = uint32((levels >> v) & 1)
			}
			for _, scope := range []uint32{0, 1} {
				state := satSolverClauseState{assignment: satClauseAssignmentBytes(values[:]), levels: lvls[:]}
				for n := 0; n <= 3; n++ {
					cases := 1
					for i := 0; i < n; i++ {
						cases *= 6
					}
					for formula := 0; formula < cases; formula++ {
						var clause [3]satLiteral
						bits := formula
						var mask uint32
						satisfied := false
						for i := 0; i < n; i++ {
							literal := bits % 6
							bits /= 6
							clause[i] = satLiteral(literal)
							val := values[literal]
							if scope != 0 && lvls[literal/2] != 0 {
								val = lUndef
							}
							if val == lTrue {
								satisfied = true
							}
							if val == lUndef {
								mask |= 1 << literal
							}
						}
						for v := 0; v < 3; v++ {
							if mask&(3<<(2*v)) == 3<<(2*v) {
								satisfied = true
							}
						}
						count := uint32(n)
						keep, e := state.simplifyClause(scope, &count, clause[:n])
						if e != nil || keep == satisfied {
							t.Fatal(assignments, levels, scope, formula, keep, e)
						}
						if !keep {
							if count != uint32(n) {
								t.Fatal("early tautology publishes no final count", count, n)
							}
							continue
						}
						var want []satLiteral
						for l := 0; l < 6; l++ {
							if mask&(1<<l) != 0 {
								want = append(want, satLiteral(l))
							}
						}
						if int(count) != len(want) {
							t.Fatal("clause size", count, want)
						}
						for i, l := range want {
							if clause[i] != l {
								t.Fatal("simplified literals", clause, want)
							}
						}
					}
				}
			}
		}
	}
}

func TestSATSolverClauseCoreWatchSelection(t *testing.T) {
	a := satStore(t, nil)
	literals := satLs(0, 2, 4, 6)
	clause := satMk(t, a, literals, true)
	for assignments := 0; assignments < 81; assignments++ {
		var values [8]lbool
		code := assignments
		for v := 0; v < 4; v++ {
			value := []lbool{lFalse, lUndef, lTrue}[code%3]
			code /= 3
			values[2*v] = value
			values[2*v+1] = value.negated()
		}
		for levels := 0; levels < 81; levels++ {
			var lvls [4]uint32
			code = levels
			for i := range lvls {
				lvls[i] = uint32(code % 3)
				code /= 3
			}
			state := satSolverClauseState{assignment: satClauseAssignmentBytes(values[:]), levels: lvls[:]}
			for start := uint32(0); start < 4; start++ {
				want := uint32(^uint32(0))
				// Independent category priority: true then unknown then false. True/false
				// retain the first extreme-level tie; unknown retains the final candidate.
				for i := start; i < 4; i++ {
					if values[2*i] == lTrue && (want == ^uint32(0) || lvls[i] < lvls[want]) {
						want = i
					}
				}
				if want == ^uint32(0) {
					for i := start; i < 4; i++ {
						if values[2*i] == lUndef {
							want = i
						}
					}
				}
				if want == ^uint32(0) {
					for i := start; i < 4; i++ {
						if want == ^uint32(0) || lvls[i] > lvls[want] {
							want = i
						}
					}
				}
				got, e := state.selectWatchLiteral(a, clause, start)
				if e != nil || got != want {
					t.Fatal(assignments, levels, start, got, want, e)
				}
			}
			if values[2] == lFalse && values[4] == lFalse && values[6] == lFalse {
				want := uint32(1)
				for i := uint32(2); i < 4; i++ {
					if lvls[i] > lvls[want] {
						want = i
					}
				}
				got, e := state.selectLearnedWatchLiteral(a, clause)
				if e != nil || got != want {
					t.Fatal("learned watch", got, want, e)
				}
			}
		}
	}
}

func TestSATSolverClauseCoreSourcePrefixesAndGuards(t *testing.T) {
	state := satSolverClauseState{assignment: satClauseAssignmentBytes([]lbool{lFalse, lTrue, lUndef, lUndef, lTrue, lFalse}), levels: []uint32{0, 0, 0}}
	clause := satLs(4, 2, 0)
	n := uint32(3)
	keep, e := state.simplifyClause(0, &n, clause)
	// Sort [0,2,4], discard0, publish2 at index0, then true4 returns false.
	if e != nil || keep || n != 3 || !reflect.DeepEqual(clause, satLs(2, 2, 4)) {
		t.Fatal("source early return prefix", clause, n, keep, e)
	}
	state.assignment = state.assignment[:4]
	clause = satLs(8, 2, 0)
	n = 3
	keep, e = state.simplifyClause(0, &n, clause)
	if !errors.Is(e, arena.ErrBounds) || keep || n != 3 || !reflect.DeepEqual(clause, satLs(2, 2, 8)) {
		t.Fatal("guard after actual earlier source writes", clause, n, e)
	}
	// Above-root values become unknown; the level access happens after value read.
	state.assignment = satClauseAssignmentBytes([]lbool{lTrue, lFalse})
	state.levels = nil
	clause = satLs(0)
	n = 1
	if _, e = state.simplifyClause(1, &n, clause); !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
}

func TestSATSolverClauseCoreWarmNoAllocation(t *testing.T) {
	values := []lbool{lUndef, lUndef, lUndef, lUndef, lUndef, lUndef}
	state := satSolverClauseState{assignment: satClauseAssignmentBytes(values), levels: []uint32{0, 0, 0}}
	var literals [6]satLiteral
	var err error
	allocations := testing.AllocsPerRun(100, func() {
		literals = [6]satLiteral{4, 0, 2, 4, 0, 2}
		n := uint32(len(literals))
		_, err = state.simplifyClause(0, &n, literals[:])
	})
	if err != nil || allocations != 0 {
		t.Fatal(allocations, err)
	}
}

func TestSATSolverClauseCoreWatchSourceAssertions(t *testing.T) {
	a := satStore(t, nil)
	clause := satMk(t, a, satLs(0, 2, 4, 6), true)
	state := satSolverClauseState{levels: []uint32{0, 4, 2, 3}}
	got, e := state.selectLearnedWatchLiteral(a, clause)
	// The source only reads assignments in this method's debug assertion.
	if satStorageSourceDebug {
		if !errors.Is(e, arena.ErrBounds) {
			t.Fatal("debug assignment access", got, e)
		}
	} else if e != nil || got != 1 {
		t.Fatal("release level selection", got, e)
	}
	state.assignment = satClauseAssignmentBytes([]lbool{lFalse, lTrue, lUndef, lUndef, lFalse, lTrue, lFalse, lTrue})
	got, e = state.selectLearnedWatchLiteral(a, clause)
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) {
			t.Fatal("source false-literal assertion", got, e)
		}
	} else if e != nil || got != 1 {
		t.Fatal("release ignores assertion", got, e)
	}
	got, e = state.selectWatchLiteral(a, clause, 4)
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) {
			t.Fatal("source candidate assertion", got, e)
		}
	} else if e != nil || got != ^uint32(0) {
		t.Fatal("release empty candidate sentinel", got, e)
	}
	if e = a.Shrink(clause, 1); e != nil {
		t.Fatal(e)
	}
	_, e = state.selectWatchLiteral(a, clause, 0)
	if satStorageSourceDebug && !errors.Is(e, ErrSATAssertion) || !satStorageSourceDebug && e != nil {
		t.Fatal("source watch size assertion", e)
	}
	got, e = state.selectLearnedWatchLiteral(a, clause)
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) {
			t.Fatal("source learned watch size assertion", got, e)
		}
	} else if e != nil || got != ^uint32(0) {
		t.Fatal("release learned sentinel", got, e)
	}
}

func TestSATSolverClauseStateSourceAssignmentWidth(t *testing.T) {
	// solver.h stores literal assignments in char_vector, independently of the
	// int32 lbool enum used for returned values. Keep the hot storage byte-wide.
	var state satSolverClauseState
	if n := reflect.TypeOf(state.assignment).Elem().Size(); n != 1 {
		t.Fatal("source assignment byte width", n)
	}
}

// Fixture conversion preserves the independent enum-valued truth oracles while
// supplying the source char-vector representation to the borrowed view.
func satClauseAssignmentBytes(values []lbool) []int8 {
	out := make([]int8, len(values))
	for i, v := range values {
		out[i] = int8(v)
	}
	return out
}
