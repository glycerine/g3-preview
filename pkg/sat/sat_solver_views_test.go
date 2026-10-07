// Source: src/sat/sat_solver.h:245-286,318-323,539-542.
// Licensed under the MIT license. Source author: Leonardo de Moura, 2011-05-21.
package sat

import (
	"errors"
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

func TestSATSolverBorrowedModelCoreCurrentMembers(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	mv, e := s.borrowModel()
	satOK(t, e)
	cv, e := s.BorrowCore()
	satOK(t, e)
	for i := uint32(0); i < 30; i++ {
		satOK(t, s.model.Push(uint32(util.LTrue)))
		satOK(t, s.conflict.core.Push(i))
		n, e := mv.Size()
		satOK(t, e)
		if n != i+1 {
			t.Fatal(n, i)
		}
		v, e := mv.Read(BoolVar(i))
		satOK(t, e)
		if v != util.LTrue {
			t.Fatal(v)
		}
		l, e := cv.Read(i)
		satOK(t, e)
		if l != Literal(i) {
			t.Fatal(l, i)
		}
	}
	satOK(t, s.model.Shrink(0))
	satOK(t, s.conflict.core.Shrink(0))
	n, e := mv.Size()
	satOK(t, e)
	if n != 0 {
		t.Fatal(n)
	}
	n, e = cv.Size()
	satOK(t, e)
	if n != 0 {
		t.Fatal(n)
	}
	satFullSolverRuntimeAudit(t, s)
	satOK(t, s.Close())
	if _, e = mv.Size(); e == nil {
		t.Fatal("closed model borrow")
	}
	if _, e = cv.Size(); e == nil {
		t.Fatal("closed core borrow")
	}
}
func TestSATSolverBorrowedScalarSourceAuthorities(t *testing.T) {
	s := satInspectionFixture(t)
	n, e := s.sourceNumVars()
	satOK(t, e)
	if n != 4 {
		t.Fatal(n)
	}
	s.gc.restarts = 13
	s.gc.restartStat = 99
	n, e = s.sourceNumRestarts()
	satOK(t, e)
	if n != 13 {
		t.Fatal(n)
	}
	satOK(t, s.sourceSetExternal(2))
	b, e := s.SourceIsExternal(2)
	satOK(t, e)
	if !b {
		t.Fatal(b)
	}
	l, e := s.sourceValueLiteral(3)
	satOK(t, e)
	if l != util.LTrue {
		t.Fatal(l)
	}
	l, e = s.sourceValueVar(1)
	satOK(t, e)
	if l != util.LFalse {
		t.Fatal(l)
	}
	n, e = s.sourceInitTrailSize()
	satOK(t, e)
	if n != 1 {
		t.Fatal(n)
	}
	cfg, e := s.borrowConfig()
	satOK(t, e)
	if cfg != s.config {
		t.Fatal("copied config")
	}
	mc, e := s.borrowModelConverter()
	satOK(t, e)
	if mc != s.converter {
		t.Fatal("copied converter")
	}
	satFullSolverRuntimeAudit(t, s)
}
func TestSATSolverClauseIteratorSameBackingMutationsAndRelocation(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	for i := 0; i < 4; i++ {
		_, e := s.MkVar(false, true)
		satOK(t, e)
	}
	_, e := s.Lifecycle.MkClauseCore(3, []Literal{0, 2, 4}, false)
	satOK(t, e)
	it, e := s.beginOriginalClauses()
	satOK(t, e)
	c := s.Lifecycle.Original[0]
	// Source pointers observe the actual retained backing slot.
	s.Lifecycle.Original[0] = c
	got, yes, e := it.Next()
	satOK(t, e)
	if !yes || got != c {
		t.Fatal(got, yes)
	}
	_, yes, e = it.Next()
	satOK(t, e)
	if yes {
		t.Fatal("captured end")
	}
	it, e = s.beginOriginalClauses()
	satOK(t, e)
	for len(s.Lifecycle.Original) < cap(s.Lifecycle.Original) {
		_, e := s.Lifecycle.MkClauseCore(3, []Literal{0, 2, 4}, false)
		satOK(t, e)
	}
	// Real source array mutation transfers the two existing owning scalar slots.
	mut, e := s.beginOriginalClauses()
	satOK(t, e)
	if len(s.Lifecycle.Original) >= 2 {
		first, second := s.Lifecycle.Original[0], s.Lifecycle.Original[1]
		s.Lifecycle.Original[0], s.Lifecycle.Original[1] = second, first
		got, yes, e = mut.Next()
		satOK(t, e)
		if !yes || got != second {
			t.Fatal(got, second)
		}
		s.Lifecycle.Original[0], s.Lifecycle.Original[1] = first, second
	}
	oldCap := cap(s.Lifecycle.Original)
	got, yes, e = it.Next()
	satOK(t, e)
	if !yes || got != c {
		t.Fatal(got, yes)
	}
	_, yes, e = it.Next()
	satOK(t, e)
	if yes {
		t.Fatal("captured end changed after same-cap append")
	}
	it, e = s.beginOriginalClauses()
	satOK(t, e)
	_, e = s.Lifecycle.MkClauseCore(3, []Literal{0, 2, 4}, false)
	satOK(t, e)
	if cap(s.Lifecycle.Original) <= oldCap {
		t.Fatal("no relocation")
	}
	oldIndex := it.index
	if _, _, e = it.Next(); e != arena.ErrBounds {
		t.Fatal("reached old backing guard", e)
	}
	if it.index != oldIndex {
		t.Fatal("failed dereference advanced", it.index, oldIndex)
	}
}

func TestSATSolverBorrowedIndependentAssignmentAndScopeViews(t *testing.T) {
	for variable := 0; variable < 4; variable++ {
		for sign := -1; sign < 2; sign++ {
			t.Run(fmt.Sprintf("%d/%d", variable, sign), func(t *testing.T) {
				s, _, _ := satSolverCheckFixture(t)
				for i := 0; i < 4; i++ {
					_, e := s.MkVar(false, true)
					satOK(t, e)
				}
				if sign >= 0 {
					satOK(t, s.propagation.assign(Literal(uint32(variable)*2+uint32(sign)), satJustification{}))
				}
				for v := 0; v < 4; v++ {
					want := util.LUndef
					if v == variable && sign >= 0 {
						want = util.LTrue
						if sign == 1 {
							want = util.LFalse
						}
					}
					got, e := s.sourceValueVar(BoolVar(v))
					satOK(t, e)
					if got != want {
						t.Fatal(v, got, want)
					}
					for b := 0; b < 2; b++ {
						x, e := s.sourceValueLiteral(Literal(2*v + b))
						satOK(t, e)
						w := want
						if b != 0 {
							w = w.Negated()
						}
						if x != w {
							t.Fatal(x, w)
						}
					}
				}
				_, e := s.Lifecycle.propagate(false, false, false)
				satOK(t, e)
				satOK(t, s.Lifecycle.push())
				n, e := s.sourceScopeLevel()
				satOK(t, e)
				if n != 1 {
					t.Fatal(n)
				}
				n, e = s.sourceInitTrailSize()
				satOK(t, e)
				want := uint32(0)
				if sign >= 0 {
					want = 1
				}
				if n != want {
					t.Fatal(n, want)
				}
				levelWant := ^uint32(0)
				if variable == 0 && sign >= 0 {
					levelWant = 0
				}
				n, e = s.sourceLevelVar(0)
				satOK(t, e)
				if n != levelWant {
					t.Fatal(n, levelWant)
				}
				n, e = s.sourceLevelLiteral(1)
				satOK(t, e)
				if n != levelWant {
					t.Fatal(n, levelWant)
				}
				b, e := s.sourceWasEliminated(0)
				satOK(t, e)
				if b {
					t.Fatal(b)
				}
				b, e = s.sourceInconsistent()
				satOK(t, e)
				if b {
					t.Fatal(b)
				}
				satFullSolverRuntimeAudit(t, s)
			})
		}
	}
}
func TestSATSolverBorrowedCanceledActualResourcePrefix(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	n := s.Limit.Count()
	canceled, e := s.sourceCanceled()
	satOK(t, e)
	if canceled || s.Limit.Count() != n+1 {
		t.Fatal(canceled, s.Limit.Count(), n)
	}
	s.Limit.Cancel()
	n = s.Limit.Count()
	canceled, e = s.sourceCanceled()
	satOK(t, e)
	if !canceled || s.Limit.Count() != n+1 {
		t.Fatal(canceled, s.Limit.Count(), n)
	}
	s.Limit.ResetCancel()
	satFullSolverRuntimeAudit(t, s)
}
func TestSATSolverBorrowedZeroAllocAndCopiedClosedGuards(t *testing.T) {
	s := satInspectionFixture(t)
	mv, e := s.borrowModel()
	satOK(t, e)
	cv, e := s.BorrowCore()
	satOK(t, e)
	before := s.budget.Used()
	s.budget.FailAfter(0)
	for i := 0; i < 20; i++ {
		_, e = s.sourceNumVars()
		satOK(t, e)
		_, e = mv.Size()
		satOK(t, e)
		_, e = cv.Size()
		satOK(t, e)
		_, e = s.sourceModelIsCurrent()
		satOK(t, e)
		_, e = s.borrowConfig()
		satOK(t, e)
		it, e := s.beginOriginalClauses()
		satOK(t, e)
		_, _, e = it.Next()
		satOK(t, e)
	}
	if s.budget.Used() != before {
		t.Fatal(before, s.budget.Used())
	}
	satFullSolverRuntimeAudit(t, s)
	s.budget.FailAfter(-1)
	copy := *s
	if _, e = copy.borrowModel(); e != ErrSATOwner {
		t.Fatal(e)
	}
	if _, e = copy.sourceNumVars(); e != ErrSATOwner {
		t.Fatal(e)
	}
	satOK(t, s.Close())
	if _, e = s.sourceNumVars(); e == nil {
		t.Fatal("closed scalar owner")
	}
}

func TestSATSolverBorrowedReachedCurrentHeaderReadGuards(t *testing.T) {
	s := satInspectionFixture(t)
	old := s.Variables.level.Raw.Block
	fresh, e := s.runtime.memory.Allocate(8)
	satOK(t, e)
	// A real allocated but unreadable source header is a bounded safety-domain fixture.
	// The displaced real backing remains an explicit caller-owned local until restoration.
	s.Variables.level.Raw.Block = fresh
	n, e := s.sourceNumVars()
	if !errors.Is(e, util.ErrRuntimeUninitialized) || n != 0 {
		t.Fatal(n, e)
	}
	roots := satFullSolverRuntimeRoots(s)
	roots = append(roots, old)
	satScalarAuditRoots(t, s.Variables, roots...)
	s.Variables.level.Raw.Block = old
	satOK(t, s.runtime.memory.Deallocate(&fresh))
	satFullSolverRuntimeAudit(t, s)
}

func TestSATSolverClauseIteratorLearnedEmptyAndOwnerLifetime(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	empty, e := s.beginLearnedClauses()
	satOK(t, e)
	for i := 0; i < 4; i++ {
		_, e := s.MkVar(false, true)
		satOK(t, e)
	}
	c, e := s.Lifecycle.MkClauseCore(4, []Literal{0, 2, 4, 6}, true)
	satOK(t, e)
	_, yes, e := empty.Next()
	satOK(t, e)
	if yes {
		t.Fatal("captured null end")
	}
	it, e := s.beginLearnedClauses()
	satOK(t, e)
	got, yes, e := it.Next()
	satOK(t, e)
	if !yes || got != c {
		t.Fatal(got, c)
	}
	satFullSolverRuntimeAudit(t, s)
	satOK(t, s.Close())
	if _, _, e = it.Next(); e == nil {
		t.Fatal("closed iterator owner")
	}
}
