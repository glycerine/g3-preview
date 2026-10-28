package sat

// Source: sat_solver.cpp:2218-2238; independent finite level-set oracle.

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestSATSolverLiteralSpanDiffLevelsFiniteOracle(t *testing.T) {
	s, v, _ := satPropagationFixture(t, 4)
	v.scopeLevel = 3
	for i := uint32(0); i < 4; i++ {
		satOK(t, s.assign(Literal(i*2), satJustification{}))
	}
	for shape := 0; shape < 256; shape++ {
		code := shape
		for i := 0; i < 4; i++ {
			satOK(t, v.setLevel(BoolVar(uint32(i)), uint32(code%4)))
			code /= 4
		}
		for mask := 0; mask < 16; mask++ {
			var lits [8]Literal
			n := uint32(0)
			var seen [4]bool
			want := uint32(0)
			for i := 0; i < 4; i++ {
				if mask&(1<<i) == 0 {
					continue
				}
				level := uint32(shape >> (2 * i) & 3)
				if !seen[level] {
					seen[level] = true
					want++
				}
				// Both signs and repeated variables count their level once.
				lits[n], lits[n+1] = Literal(i*2), Literal(i*2+1)
				n += 2
			}
			got, e := s.numDiffLevelsLiterals(n, lits[:n])
			satOK(t, e)
			if got != want {
				t.Fatalf("shape=%d mask=%d levels=%d want=%d", shape, mask, got, want)
			}
			for i, mark := range s.diffLevels.data {
				if mark != 0 {
					t.Fatalf("mark %d survives second source traversal", i)
				}
			}
		}
	}
	satPropagationAssertGraph(t, s)
}

func TestSATSolverLiteralSpanDiffLevelsAccessAndFailurePrefixes(t *testing.T) {
	s, v, b := satPropagationFixture(t, 2)
	v.scopeLevel = 3
	satOK(t, s.assign(0, satJustification{}))
	satOK(t, s.assign(2, satJustification{}))
	satOK(t, v.setLevel(BoolVar(0), 1))
	satOK(t, v.setLevel(BoolVar(1), 2))
	// Native mark storage grows even while the shared arena refuses reservations.
	before := b.Used()
	b.FailAfter(0)
	_, e := s.numDiffLevelsLiterals(1, nil)
	if !errors.Is(e, arena.ErrBounds) || len(s.diffLevels.data) != 4 || b.Used() != before {
		t.Fatal("native growth before literal access", e, len(s.diffLevels.data), b.Used(), before)
	}
	satPropagationAssertGraph(t, s)
	b.FailAfter(-1)
	// No whole-span preflight: the first mark precedes the second bad access.
	got, e := s.numDiffLevelsLiterals(2, []Literal{0})
	if !errors.Is(e, arena.ErrBounds) || got != 1 || s.diffLevels.data[1] != 1 {
		t.Fatal("literal access prefix", got, e, s.diffLevels.data)
	}
	satPropagationAssertGraph(t, s)
	// A failed traversal leaves its mark; there is no invented rollback/reset.
	got, e = s.numDiffLevelsLiterals(2, []Literal{0, 2})
	satOK(t, e)
	if got != 1 || s.diffLevels.data[1] != 0 || s.diffLevels.data[2] != 0 {
		t.Fatal("retained mark and actual successful reset", got, s.diffLevels.data)
	}
	// num=0 must not inspect the supplied pointer/literals.
	got, e = s.numDiffLevelsLiterals(0, []Literal{DefaultLiteral()})
	satOK(t, e)
	if got != 0 {
		t.Fatal(got)
	}
	// Go spans respect current logical length; callers pass the initialized array.
	backing := [2]Literal{0, 2}
	got, e = s.numDiffLevelsLiterals(2, backing[:0])
	if !errors.Is(e, arena.ErrBounds) || got != 0 {
		t.Fatal("native empty span bounds", got, e)
	}
	got, e = s.numDiffLevelsLiterals(2, backing[:])
	satOK(t, e)
	if got != 2 || s.diffLevels.data[1] != 0 || s.diffLevels.data[2] != 0 {
		t.Fatal("native initialized span", got, s.diffLevels.data)
	}

}

func TestSATSolverLiteralSpanDiffLevelsSourceDebugAndWarm(t *testing.T) {
	s, v, b := satPropagationFixture(t, 2)
	v.scopeLevel = 2
	satOK(t, s.assign(0, satJustification{}))
	satOK(t, v.setLevel(BoolVar(0), 1))
	satOK(t, v.setLevel(BoolVar(1), 2))
	got, e := s.numDiffLevelsLiterals(2, []Literal{0, 2})
	if satStorageSourceDebug {
		if !errors.Is(e, ErrSATAssertion) || got != 1 || s.diffLevels.data[1] != 1 {
			t.Fatal("DEBUG second unassigned literal", got, e)
		}
		// Explicit fixture reset after observing the exact stopped history.
		satOK(t, s.diffLevels.write(1, 0))
	} else {
		satOK(t, e)
		if got != 2 {
			t.Fatal("release reads source level without assignment assertion", got)
		}
	}
	satOK(t, s.assign(2, satJustification{}))
	satOK(t, v.setLevel(BoolVar(1), 2))
	lits := [4]Literal{0, 2, 1, 3}
	before := b.Used()
	b.FailAfter(0)
	allocs := testing.AllocsPerRun(100, func() {
		got, e := s.numDiffLevelsLiterals(4, lits[:])
		if e != nil || got != 2 {
			panic("warm source level counting")
		}
	})
	if allocs != 0 || b.Used() != before {
		t.Fatal("warm physical plateau", allocs, b.Used(), before)
	}
	b.FailAfter(-1)
	satPropagationAssertGraph(t, s)
}

func TestSATSolverLiteralSpanDiffLevelsWrappedReserveAndOwner(t *testing.T) {
	s, v, b := satPropagationFixture(t, 1)
	v.scopeLevel = ^uint32(0)
	before := b.Used()
	b.FailAfter(0)
	got, e := s.numDiffLevelsLiterals(0, nil)
	satOK(t, e)
	if got != 0 || len(s.diffLevels.data) != 0 || cap(s.diffLevels.data) != 0 || b.Used() != before {
		t.Fatal("source unsigned reserve argument wraps to zero", got, s.diffLevels.data)
	}
	b.FailAfter(-1)
	copy := *s
	if _, e = copy.numDiffLevelsLiterals(0, nil); !errors.Is(e, ErrSATOwner) {
		t.Fatal("copied cold owner", e)
	}
	var missing *satSolverPropagationState
	if _, e = missing.numDiffLevelsLiterals(0, nil); !errors.Is(e, ErrSATOwner) {
		t.Fatal("nil cold owner", e)
	}
	satOK(t, s.Close())
	if _, e = s.numDiffLevelsLiterals(0, nil); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("closed cold owner", e)
	}
}
