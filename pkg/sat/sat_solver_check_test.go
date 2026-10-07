// Source: src/sat/sat_solver.cpp:34–74,723–802,993–1054,1191–1346,2763–2801.
package sat

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// These are finite branch/oracle fixtures, separate from the registered
// sat_user_scope.cpp campaign. No truth-table algorithm is production code.
func satSolverFiniteCNFOracle(clauses [][]Literal, n uint32) util.LBool {
	for bits := uint32(0); bits < 1<<n; bits++ {
		all := true
		for _, c := range clauses {
			yes := false
			for _, l := range c {
				yes = yes || (bits&(1<<uint32(l.variable())) != 0) != l.sign()
			}
			all = all && yes
		}
		if all {
			return util.LTrue
		}
	}
	return util.LFalse
}
func TestSATSolverDefaultCheckSearchIndependentFiniteCNF(t *testing.T) {
	// All 16 subsets of four binary clauses over two variables; some entail a
	// unit only through propagation/branching and the full set is inconsistent.
	for mask := 0; mask < 16; mask++ {
		t.Run(string(rune('A'+mask)), func(t *testing.T) {
			s, _, _ := satSolverCheckFixture(t)
			for i := 0; i < 2; i++ {
				v, e := s.MkVar(false, true)
				satOK(t, e)
				if v != BoolVar(i) {
					t.Fatal(v)
				}
			}
			clauses := [][]Literal{}
			for i := 0; i < 4; i++ {
				if mask&(1<<i) == 0 {
					continue
				}
				c := []Literal{MakeLiteral(0, i&1 != 0), MakeLiteral(1, i&2 != 0)}
				clauses = append(clauses, c)
				input := append([]Literal(nil), c...)
				satOK(t, s.MkClause(input))
			}
			got, e := s.Check(nil)
			satOK(t, e)
			if want := satSolverFiniteCNFOracle(clauses, 2); got != want {
				t.Fatalf("default source search got %v want %v", got, want)
			}
			if got == util.LTrue {
				model, e := s.AppendModel(nil)
				satOK(t, e)
				for _, c := range clauses {
					yes := false
					for _, l := range c {
						yes = yes || model[l.variable()] == util.LTrue && !l.sign() || model[l.variable()] == util.LFalse && l.sign()
					}
					if !yes {
						t.Fatalf("actual converter/model does not satisfy original %v: %v", c, model)
					}
				}
			}
		})
	}
}
func TestSATSolverCheckMaxConflictsZeroSourceReturnsUndef(t *testing.T) {
	s, c, p := satSolverCheckFixture(t)
	k := "max_conflicts"
	satOK(t, c.SetUIntName(&p, &k, 0))
	satOK(t, s.UpdateParams(&p))
	// Source still runs initialization/propagation/default simplify before the
	// zero-limit return; it does not produce a guessed satisfying model.
	got, e := s.Check(nil)
	satOK(t, e)
	if got != util.LUndef {
		t.Fatal(got)
	}
}
func TestSATSolverFullOwnerCopyClosedAndBorrowedRuntimeGuards(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	copied := *s
	if _, e := copied.Check(nil); !errors.Is(e, ErrSATOwner) {
		t.Fatal(e)
	}
	if e := copied.Close(); !errors.Is(e, ErrSATOwner) {
		t.Fatal(e)
	}
	// A Runtime close cannot bulk-delete roots or silently invalidate this live
	// solver owner, even when the formula currently has no variables/clauses.
	if e := s.runtime.Close(); !errors.Is(e, ErrSATRuntimeLive) {
		t.Fatal(e)
	}
	satOK(t, s.Close())
	if _, e := s.Check(nil); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}
