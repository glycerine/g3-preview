// Source: src/sat/sat_solver.cpp:3285–3299,3503–3636.
// Licensed under the MIT license. Source author: Nikolaj Bjorner.
package sat

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestSATSolverConsequenceReasonShortCircuitAndClauseDomainTodoPrefix(t *testing.T) {
	for _, kind := range []uint32{0, 1, 2, 3, 4} {
		t.Run(string(rune('A'+kind)), func(t *testing.T) {
			s, _, _ := satSolverCheckFixture(t)
			for i := 0; i < 4; i++ {
				_, e := s.MkVar(true, true)
				satOK(t, e)
			}
			satOK(t, s.Lifecycle.push())
			for _, lit := range []Literal{0, 2, 4} {
				satOK(t, s.propagation.assign(lit, satJustification{}))
			}
			var j satJustification
			switch kind {
			case 1:
				j = newSATBinaryJustification(1)
			case 2:
				j = newSATTernaryJustification(1, 3)
			case 3:
				// Actual clause is owned by the lifecycle and can already be satisfied;
				// use its real allocator reason, with the consequent at the fourth slot.
				c, e := s.Clauses.MkClause([]Literal{1, 3, 5, 6}, false)
				satOK(t, e)
				satOK(t, s.Lifecycle.pushClause(false, c))
				satOK(t, s.Lifecycle.attachClause(c))
				off, e := s.Clauses.Offset(c)
				satOK(t, e)
				j = newSATClauseJustification(off)
			case 4:
				j = newSATExternalJustification(1)
			}
			// Concrete source assign stores this actual reason in the real two-word
			// Runtime column; it is not an extension-success callback or a shadow state.
			satOK(t, s.propagation.assign(6, j))
			var set satSolverIndexSet
			satOK(t, set.init(s.runtime.memory))
			t.Cleanup(func() { satOK(t, set.close()) })
			if kind == 4 {
				satOK(t, s.analysis.extAntecedents.Push(6))
			}
			got, e := s.extractConsequenceAssumptions(6, &set)
			if kind == 4 {
				wantError := error(ErrSATMissingSourceExtBinding)
				if satStorageSourceDebug {
					wantError = ErrSATAssertion
				}
				if !errors.Is(e, wantError) {
					t.Fatal(e)
				}
				extSize, er := s.analysis.extAntecedents.Size()
				satOK(t, er)
				wantExt := uint32(0)
				if satStorageSourceDebug {
					wantExt = 1
				}
				if extSize != wantExt {
					t.Fatal("actual ext guard/reset prefix", extSize, wantExt)
				}
				n, e := s.todoAntecedents.Size()
				satOK(t, e)
				if n != 0 {
					t.Fatal(n)
				}
				return
			}
			satOK(t, e)
			if got != (kind == 0) {
				t.Fatal(got, kind)
			}
			n, e := s.todoAntecedents.Size()
			satOK(t, e)
			want := uint32(1)
			if kind == 0 {
				want = 0
			}
			if kind == 3 {
				want = 3
			}
			if n != want {
				t.Fatal("source first-miss vs clause all-domain traversal", kind, n, want)
			}
			seen := uint32(0)
			for i := uint32(0); i < n; i++ {
				v, e := s.todoAntecedents.Read(i)
				satOK(t, e)
				if kind == 3 {
					if v > 4 || v%2 != 0 || seen&(1<<v) != 0 {
						t.Fatal("actual clause domains must queue each missing antecedent once", v, seen)
					}
					seen |= 1 << v
				} else if v != i*2 {
					t.Fatal(v, i)
				}
			}
		})
	}
}
func TestSATSolverConsequenceCopiedOutputAndCurrentModelQueryLifetime(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	for i := 0; i < 2; i++ {
		_, e := s.MkVar(true, true)
		satOK(t, e)
	}
	satOK(t, s.MkClause([]Literal{0}))
	r, e := s.Check(nil)
	satOK(t, e)
	if r != util.LTrue {
		t.Fatal(r)
	}
	old, e := s.AppendModel(nil)
	satOK(t, e)
	out, e := newSATSolverLiteralRows(s.runtime.memory, s.budget)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, out.CloseChecked()) })
	copied := *out
	if _, e = copied.rowCount(); !errors.Is(e, ErrSATOwner) {
		t.Fatal(e)
	}
	if e = copied.CloseChecked(); !errors.Is(e, ErrSATOwner) {
		t.Fatal(e)
	}
	r, e = s.getConsequences(nil, []BoolVar{0, 1}, out)
	satOK(t, e)
	if r != util.LTrue {
		t.Fatal(r)
	}
	now, e := s.AppendModel(nil)
	satOK(t, e)
	if len(old) != len(now) {
		t.Fatal(old, now)
	}
	for i := range old {
		if old[i] != now[i] {
			t.Fatal("source saved model restored", old, now)
		}
	}
	satOK(t, out.CloseChecked())
	if _, e = s.getConsequences(nil, nil, out); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}
func TestSATSolverConsequenceRealCancellationRetainsCallerOutputPrefix(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	_, e := s.MkVar(true, true)
	satOK(t, e)
	out, e := newSATSolverLiteralRows(s.runtime.memory, s.budget)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, out.CloseChecked()) })
	satOK(t, out.appendLiterals([]Literal{0}))
	s.Limit.Cancel()
	_, e = s.getConsequences(nil, []BoolVar{0}, out)
	if !errors.Is(e, ErrSATSolverCanceled) {
		t.Fatal(e)
	}
	row, e := out.captureRow(0)
	satOK(t, e)
	v, e := row.read(0)
	satOK(t, e)
	if v != 0 || row.count != 1 {
		t.Fatal(v, row.count)
	}
	satFullSolverRuntimeAudit(t, s, out)
}

func TestSATSolverConsequencesWarmedActualGraphPlateau(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	for i := 0; i < 3; i++ {
		_, e := s.MkVar(true, true)
		satOK(t, e)
	}
	satOK(t, s.MkClause([]Literal{1, 2}))
	satOK(t, s.MkClause([]Literal{3, 4}))
	out, e := newSATSolverLiteralRows(s.runtime.memory, s.budget)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, out.CloseChecked()) })
	var used uint64
	var blocks uint32
	for pass := 0; pass < 6; pass++ {
		r, e := s.getConsequences([]Literal{0}, []BoolVar{1, 2}, out)
		satOK(t, e)
		if r != util.LTrue {
			t.Fatal(r)
		}
		n, e := out.rowCount()
		satOK(t, e)
		if n != 2 {
			t.Fatal(n)
		}
		satFullSolverRuntimeAudit(t, s, out)
		st := s.runtime.memory.Stats()
		if pass == 2 {
			used = s.budget.Used()
			blocks = st.LiveBlocks
		}
		if pass > 2 && (used != s.budget.Used() || blocks != st.LiveBlocks) {
			t.Fatal("warm physical graph did not plateau", used, s.budget.Used(), blocks, st.LiveBlocks)
		}
		satOK(t, out.SourceFinalize())
	}
}
