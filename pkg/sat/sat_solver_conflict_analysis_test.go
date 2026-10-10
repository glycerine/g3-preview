package sat

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"

	"github.com/glycerine/g3/pkg/util/control"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func satLemmaFixture(t *testing.T, n uint32) (*satSolverConflictAnalysis, *satSolverPropagationState, *satSolverVariableState, *arena.Budget) {
	t.Helper()
	p, v, b := satPropagationFixture(t, n)
	s, e := newSATSolverConflictAnalysis(p, b)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, s.Close()) })
	return s, p, v, b
}
func satLemmaAudit(t *testing.T, s *satSolverConflictAnalysis) {
	t.Helper()
	p := s.propagation
	v := p.variables
	roots := []umemory.Block{}
	satScalarAuditRoots(t, v, roots...)
	want := s.ownerBytes + uint64(reflect.TypeOf(satSolverPropagationState{}).Size()) + uint64(cap(p.diffLevels.data)) + v.Stats().TotalBytes
	todo := v.elimTodo
	want += todo.ownerBytes
	a := p.clauses
	small := a.storage.Stats()
	want += a.ownerBytes + 4*uint64(a.ids.FreeCapacity()) + a.largeBytes + a.largeDirectory + small.OwnerBytes + small.PayloadBytes + small.MetadataBytes + small.DirectoryBytes
	mc := p.converter
	want += mc.ownerBytes
	r := p.limit
	want += uint64(reflect.TypeOf(control.Reslimit{}).Size()) + 8*uint64(cap(r.Limits)) + uint64(reflect.TypeOf((*control.Reslimit)(nil)).Size())*uint64(cap(r.Children))
	want += p.memory.Stats().TotalBytes
	if want != s.budget.Used() {
		t.Fatal("actual named once-only graph", want, s.budget.Used())
	}
}
func TestSATSolverLemmaMinimizationOwnedHeaderAndLifetime(t *testing.T) {
	s, p, _, b := satLemmaFixture(t, 1)
	if s.ownerBytes != uint64(reflect.TypeOf(*s).Size()) {
		t.Fatal("actual cold header")
	}
	satLemmaAudit(t, s)
	copy := *s
	if _, e := copy.impliedByMarked(0); !errors.Is(e, ErrSATOwner) {
		t.Fatal(e)
	}
	if e := copy.Close(); !errors.Is(e, ErrSATOwner) {
		t.Fatal(e)
	}
	satOK(t, s.lemma.Push(0))
	satOK(t, s.unmark.Push(0))
	satOK(t, s.minStack.Push(0))
	satOK(t, s.extAntecedents.Push(0))
	satLemmaAudit(t, s)
	before := b.Used()
	satOK(t, s.Close())
	if b.Used() >= before {
		t.Fatal("actual backing/header release")
	}
	if _, e := s.impliedByMarked(0); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
	if s.propagation != p {
		t.Fatal("borrowed identity")
	}
}
func TestSATSolverLemmaAntecedentNativeAppend(t *testing.T) {

	s, _, v, b := satLemmaFixture(t, 2)
	satOK(t, v.setLevel(1, 1))
	s.levelSet.insert(1)
	before := b.Used()
	b.FailAfter(0)
	got, e := s.processAntecedentForMinimization(MakeLiteral(1, false))
	satOK(t, e)
	if !got || !v.mark[1] || satScalarTestUintSize(t, &s.unmark) != 1 || satScalarTestUintSize(t, &s.minStack) != 1 || b.Used() != before {
		t.Fatal("marked antecedent and native work lists", got, v.mark[1], b.Used(), before)
	}
	satLemmaAudit(t, s)
	b.FailAfter(-1)
	satOK(t, s.Close())
	if !v.mark[1] {
		t.Fatal("closing work lists must preserve caller mark")
	}

}

func TestSATSolverLemmaMinimizationNativeWorkLists(t *testing.T) {

	s, p, v, b := satLemmaFixture(t, 6)
	for id := uint32(0); id < 6; id++ {
		v.scopeLevel = 1
		j := satJustification{}
		if id == 2 || id == 5 {
			j = newSATBinaryJustification(2)
		}
		if id == 3 {
			j = newSATBinaryJustification(8)
		}
		satOK(t, p.assignCore(MakeLiteral(BoolVar(id), false), j))
	}
	for _, id := range []uint32{1, 2, 3, 5} {
		v.mark[id] = true
	}
	for _, id := range []uint32{0, 1, 2, 3, 5} {
		satOK(t, s.lemma.Push(2*id+1))
	}
	before := b.Used()
	b.FailAfter(0)
	satOK(t, s.minimizeLemma())
	if satScalarTestUintSize(t, &s.lemma) != 3 || s.minimizedLits != 2 || b.Used() != before {
		t.Fatal("lemma minimization", s.lemma.values, s.minimizedLits, b.Used(), before)
	}
	for i, want := range []uint32{1, 3, 7} {
		if got := satScalarTestUint(t, &s.lemma, uint32(i)); got != want {
			t.Fatal(i, got, want)
		}
	}
	for _, id := range []uint32{2, 4, 5} {
		if v.mark[id] {
			t.Fatal("completed mark reset", id)
		}
	}
	satLemmaAudit(t, s)
	b.FailAfter(-1)

}

func TestSATSolverLemmaMinimizationConstructorBudgetIdentityAndRefusal(t *testing.T) {
	p, _, b := satPropagationFixture(t, 1)
	foreign := arena.NewBudget(^uint64(0))
	before := b.Used()
	s, e := newSATSolverConflictAnalysis(p, foreign)
	if s != nil {
		satOK(t, s.Close())
	}
	if !errors.Is(e, ErrSATOwner) || b.Used() != before || foreign.Used() != 0 {
		t.Fatal("actual borrowed cohort budget identity", e, b.Used(), before, foreign.Used())
	}

	q, e := newSATSolverPropagationState(p.variables, p.clauses, p.converter, p.limit, p.memory, p.config, foreign)
	satOK(t, e)
	t.Cleanup(func() {
		satOK(t, q.Close())
		if foreign.Used() != 0 {
			t.Fatal("borrowed mismatch fixture exact cleanup", foreign.Used())
		}
	})
	s, e = newSATSolverConflictAnalysis(q, foreign)
	if s != nil {
		satOK(t, s.Close())
	}
	if !errors.Is(e, ErrSATOwner) {
		t.Fatal("actual Runtime budget must match the selected cohort", e)
	}
	b.FailAfter(0)
	s, e = newSATSolverConflictAnalysis(p, b)
	if !errors.Is(e, arena.ErrInjectedFailure) || s != nil || b.Used() != before {
		t.Fatal("cold header refusal before publication", s, e, b.Used(), before)
	}
	b.FailAfter(-1)
}

func TestSATSolverLemmaMinimizationNativeDeepAntecedents(t *testing.T) {

	s, p, v, b := satLemmaFixture(t, 8)
	c, e := p.clauses.MkClause([]Literal{14, 2, 4, 6, 8, 10, 12}, false)
	satOK(t, e)
	o, e := p.clauses.Offset(c)
	satOK(t, e)
	for id := uint32(0); id < 8; id++ {
		v.scopeLevel = 1
		j := satJustification{}
		if id == 7 {
			j = newSATClauseJustification(o)
		}
		satOK(t, p.assignCore(MakeLiteral(BoolVar(id), false), j))
	}
	v.mark[7] = true
	satOK(t, s.lemma.Push(1))
	satOK(t, s.lemma.Push(15))
	before := b.Used()
	b.FailAfter(0)
	satOK(t, s.minimizeLemma())
	if satScalarTestUintSize(t, &s.lemma) != 2 || s.minimizedLits != 0 || satScalarTestUintSize(t, &s.unmark) != 0 || b.Used() != before {
		t.Fatal("decision leaves preserve lemma and reset temporary marks", s.lemma.values, s.unmark.values)
	}
	for id := uint32(1); id <= 6; id++ {
		if v.mark[id] {
			t.Fatal("temporary antecedent mark", id)
		}
	}
	if !v.mark[7] {
		t.Fatal("original lemma mark")
	}
	satLemmaAudit(t, s)
	b.FailAfter(-1)
	satOK(t, p.clauses.Delete(c))

}
