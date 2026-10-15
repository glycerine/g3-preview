// Source: src/sat/sat_solver.cpp:1675–2219.
package sat

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/util/control"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func satConflictAudit(t *testing.T, c *satSolverConflictResolution) {
	t.Helper()
	s := c.dynamic.probing
	analysis := c.analysis
	d := c.dynamic
	p := analysis.propagation
	v := p.variables
	l := s.lifecycle
	roots := []umemory.Block{}
	satScalarAuditRoots(t, v, roots...)
	want := c.ownerBytes + s.ownerBytes + analysis.ownerBytes + d.ownerBytes
	want += l.ownerBytes + uint64(cap(l.Original)+cap(l.learned))*uint64(reflect.TypeOf(StoredClauseRef{}).Size()) + uint64(cap(l.reinit))*uint64(reflect.TypeOf(satClauseWrapper{}).Size()) + uint64(cap(l.scopes))*uint64(reflect.TypeOf(satSolverScope{}).Size())
	want += p.ownerBytes + v.Stats().TotalBytes
	todo := v.elimTodo
	want += todo.ownerBytes
	a := p.clauses
	small := a.storage.Stats()
	want += a.ownerBytes + 4*uint64(a.ids.FreeCapacity()) + a.largeBytes + a.largeDirectory + small.OwnerBytes + small.PayloadBytes + small.MetadataBytes + small.DirectoryBytes
	mc := p.converter
	want += mc.ownerBytes
	r := p.limit
	want += uint64(reflect.TypeOf(control.Reslimit{}).Size()) + 8*uint64(cap(r.Limits)) + uint64(reflect.TypeOf((*control.Reslimit)(nil)).Size())*uint64(cap(r.Children)) + p.memory.Stats().TotalBytes
	if want != analysis.budget.Used() {
		t.Fatal("exact named once-only graph", want, analysis.budget.Used())
	}
	expected := uint32(len(l.Original) + len(l.learned))
	if l.pendingCandidate != (StoredClauseRef{}) {
		expected++
	}
	if small.Live != expected {
		t.Fatal("unnamed clause", small.Live, expected)
	}
}
func TestSATSolverConflictWholeOperationAllocationPrefixes(t *testing.T) {
	failures, success := 0, false
	for ordinal := int64(0); ordinal < 128; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			s := satConflictFixture(t, 8)
			p := s.analysis.propagation
			l := s.dynamic.probing.lifecycle
			satConflictPush(t, s)
			for i := Literal(0); i < 10; i += 2 {
				satOK(t, p.assign(i, satJustification{}))
			}
			satConflictPush(t, s)
			satOK(t, p.assign(10, satJustification{}))
			// Actual clause implies c from current b and all five lower decisions.
			reason := []Literal{12, 11, 1, 3, 5, 7, 9}
			c, e := p.clauses.MkClause(reason, false)
			satOK(t, e)
			satOK(t, l.pushClause(false, c))
			off, e := p.clauses.Offset(c)
			satOK(t, e)
			satOK(t, p.assign(12, newSATClauseJustification(off)))
			satOK(t, p.assign(14, newSATBinaryJustification(13)))
			p.setConflict(newSATBinaryJustification(11), 14)
			s.analysis.budget.FailAfter(ordinal)
			ok, e := s.resolveConflict()
			if e != nil {
				failures++
				satProbingExpectedRefusal(t, p, e)
			} else {
				success = true
				if !ok {
					t.Fatal("unexpected unsat")
				}
				want := []Literal{11, 1, 3, 5, 7, 9}
				if !reflect.DeepEqual(satConflictLemma(t, s), want) {
					t.Fatal(satConflictLemma(t, s))
				}
				if len(l.learned) != 1 {
					t.Fatal(l.learned)
				}
				glue, e := p.clauses.Glue(l.learned[0])
				satOK(t, e)
				if glue != 2 {
					t.Fatal(glue)
				}
			}
			satConflictAudit(t, s) // BEFORE disabling the actual refusal.
			if s.conflictStat != 1 || s.conflicts != 1 {
				t.Fatal("counter prefix")
			}
			s.analysis.budget.FailAfter(-1)
		})
		if success {
			break
		}
	}
	if failures == 0 || !success {
		t.Fatal(failures, success)
	}
	t.Log("real refusal prefixes", failures, "first success", failures)
}
func TestSATSolverConflictClauseReasonBothWatchedPositions(t *testing.T) {
	for _, swapped := range []bool{false, true} {
		t.Run(fmt.Sprint(swapped), func(t *testing.T) {
			s := satConflictFixture(t, 4)
			p := s.analysis.propagation
			l := s.dynamic.probing.lifecycle
			satConflictPush(t, s)
			satOK(t, p.assign(0, satJustification{}))
			satConflictPush(t, s)
			satOK(t, p.assign(2, satJustification{}))
			ls := []Literal{4, 3, 1}
			if swapped {
				ls[0], ls[1] = ls[1], ls[0]
			}
			c, e := p.clauses.MkClause(ls, false)
			satOK(t, e)
			satOK(t, l.pushClause(false, c))
			off, e := p.clauses.Offset(c)
			satOK(t, e)
			satOK(t, p.assign(4, newSATClauseJustification(off)))
			satOK(t, p.assign(6, newSATBinaryJustification(5)))
			p.setConflict(newSATBinaryJustification(3), 6)
			ok, e := s.resolveConflict()
			satOK(t, e)
			if !ok || !reflect.DeepEqual(satConflictLemma(t, s), []Literal{3, 1}) {
				t.Fatal(ok, satConflictLemma(t, s))
			}
			satConflictAudit(t, s)
		})
	}
}
func TestSATSolverConflictFiniteImplicationOracle(t *testing.T) {
	// All16 assignments independently certify learned clause from the original
	// two implications and conflict clause, with no reference solver invocation.
	for mask := 0; mask < 16; mask++ {
		truth := func(i int) bool { return mask&(1<<i) != 0 }
		a, b, c, d := truth(0), truth(1), truth(2), truth(3)
		original := (!a || !b || c) && (!c || d) && (!b || !d)
		if original && (a && b) {
			t.Fatal("frozen learned (~b OR ~a) not implied", mask)
		}
	}
	s := satConflictFixture(t, 4)
	satConflictDAG(t, s)
	_, e := s.resolveConflict()
	satOK(t, e)
	if !reflect.DeepEqual(satConflictLemma(t, s), []Literal{3, 1}) {
		t.Fatal(satConflictLemma(t, s))
	}
	satConflictAudit(t, s)
}
func TestSATSolverConflictInitAndOptionalDispatchPrefixes(t *testing.T) {
	s := satConflictFixture(t, 2)
	p := s.analysis.propagation
	satConflictPush(t, s)
	satOK(t, p.assign(0, satJustification{}))
	satOK(t, p.assign(2, newSATBinaryJustification(1)))
	p.setConflict(newSATBinaryJustification(1), 2)
	s.conflictLevel = 1
	ok, e := s.resolveConflictForInit()
	satOK(t, e)
	if !ok || p.inconsistent || p.variables.scopeLevel != 0 || !reflect.DeepEqual(satConflictLemma(t, s), []Literal{1}) {
		t.Fatal(ok, satConflictLemma(t, s))
	}
	satConflictAudit(t, s)
	s.parallelPresent = true
	before := p.variables.scopeLevel
	e = s.popReinit(0)
	if !errors.Is(e, ErrSATMissingParallelBinding) || p.variables.scopeLevel != before {
		t.Fatal(e)
	}
	s.parallelPresent = false
	satConflictPush(t, s)
	p.extensionPresent = true
	p.setConflict(newSATExternalJustification(9), DefaultLiteral())
	_, e = s.resolveConflict()
	if !errors.Is(e, ErrSATMissingSourceExtBinding) {
		t.Fatal(e)
	}
	n, x := s.analysis.extAntecedents.Size()
	satOK(t, x)
	if n != 0 {
		t.Fatal("actual reset before missing EXT call")
	}
	p.extensionPresent = false
}

func TestSATSolverConflictStaleSpanRetainsAllocatedCandidate(t *testing.T) {
	s := satConflictFixture(t, 3)
	a := s.analysis
	satOK(t, a.lemma.Push(1))
	satOK(t, a.lemma.Push(3))
	span := satConflictColumnSpan(&a.lemma)
	satOK(t, a.lemma.Push(5)) // source captured pointer is now stale
	l := s.dynamic.probing.lifecycle
	c, e := l.mkClauseCoreSpan(3, span, true)
	if e == nil || c == (StoredClauseRef{}) {
		t.Fatal("constructor must preserve allocation before guarded source read", c, e)
	}
	if l.pendingCandidate != c || len(l.learned) != 0 {
		t.Fatal("allocated candidate lost before publication", c, l.pendingCandidate)
	}
	satConflictAudit(t, s)
	satOK(t, l.cleanupCandidate())
	satConflictAudit(t, s)
}
