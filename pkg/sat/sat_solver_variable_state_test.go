// Source: src/sat/sat_solver.h; src/sat/sat_solver.cpp; src/sat/sat_var_queue.h; src/util/vector.h.
package sat

import (
	"errors"
	"github.com/glycerine/g3/pkg/util/lbool"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func satVariableStateFixture(t *testing.T) (*satSolverVariableState, *satSolverUintSet, *arena.Budget) {
	t.Helper()
	return satVariableStateFixtureWithBudget(t, arena.NewBudget(^uint64(0)))
}

func satVariableStateFixtureWithBudget(t *testing.T, b *arena.Budget) (*satSolverVariableState, *satSolverUintSet, *arena.Budget) {
	t.Helper()
	todo, e := newSATSolverUintSet(b)
	satOK(t, e)
	mem, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b})
	satOK(t, e)
	s, e := newSATSolverVariableState(todo, b, arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{Budget: b}}, mem)
	satOK(t, e)
	t.Cleanup(func() {
		satOK(t, s.Close())
		satOK(t, todo.Close())
		satOK(t, mem.Close())
		if b.Used() != 0 {
			t.Fatal("variable graph", b.Used())
		}
	})
	return s, todo, b
}
func TestSATSolverMkVarFullSourceEighteenActionsAndByteClauseView(t *testing.T) {
	s, todo, b := satVariableStateFixture(t)
	for i := uint32(0); i < 7; i++ {
		v, e := s.mkVar(i%2 != 0, i%3 != 0)
		satOK(t, e)
		if uint32(v) != i {
			t.Fatal(v, i)
		}
		if s.NumVars() != i+1 || s.mkVarCount != i+1 || s.modelIsCurrent || !s.modelKnown || s.qhead != 0 || s.scopeLevel != 0 || s.activityInc != 128 {
			t.Fatal(s.Stats())
		}
		if !s.queue.Contains(v) || !todo.Contains(i) {
			t.Fatal("actual final queue/todo callbacks", i)
		}
		for _, l := range []Literal{MakeLiteral(v, false), MakeLiteral(v, true)} {
			value, e := s.valueLiteral(l)
			satOK(t, e)
			if value != lbool.Undef {
				t.Fatal(l, value)
			}
		}
		if s.level[i] != ^uint32(0) || satJustificationTestValue(t, &s.justification, i) != (satJustification{}) || s.phase[i] != 2 || s.previousPhase[i] != 2 {
			t.Fatal("source values", i)
		}
		if s.decision[i] != (i%3 != 0) || s.external[i] != (i%2 != 0) {
			t.Fatal(i)
		}
	}
	view := s.clauseState()
	if view.variables != s {
		t.Fatal("source char assignment")
	}
	satOK(t, s.setAssignment(Literal(0), lbool.True))
	value, e := view.value(0)
	satOK(t, e)
	if value != lbool.True {
		t.Fatal("no copied assignment view", value)
	}
	todoBytes := todo.Stats().OwnerBytes
	if s.Stats().TotalBytes+todoBytes+s.memory.Stats().TotalBytes != b.Used() {
		t.Fatal(s.Stats(), todo.Stats(), b.Used())
	}
}

func satVariableAssertFirstPrefix(t *testing.T, s *satSolverVariableState) {
	t.Helper()
	sizes := []int{len(s.watches.lists), len(s.assignment), int(satJustificationTestSize(t, &s.justification)), len(s.decision), len(s.eliminated), len(s.external), int(s.activity.Size()), len(s.level), len(s.mark), len(s.literalMark), len(s.phase), len(s.previousPhase), len(s.assignedSinceGC)}
	want := []int{2, 2, 1, 1, 1, 1, 1, 1, 1, 2, 1, 1, 1}
	unfinished := false
	for i, n := range sizes {
		if n < 0 || n > want[i] || unfinished && n != 0 {
			t.Fatal("not an actual sequential source prefix", sizes, i)
		}
		if n != want[i] {
			unfinished = true
		}
	}
	if s.mkVarCount != 1 || s.modelIsCurrent || !s.modelKnown {
		t.Fatal("prefix before first allocation", s.Stats())
	}
	if s.queue.Size() != 0 {
		for i, n := range sizes {
			if n != want[i] {
				t.Fatal("queue before columns", sizes)
			}
		}
	}
	if s.elimTodo.Contains(0) && s.queue.Size() != 1 {
		t.Fatal("todo before real heap insertion")
	}
}
func TestSATSolverNativeMkVarFirstPublicationWithoutBackingReservation(t *testing.T) {
	s, todo, b := satVariableStateFixture(t)
	before, raw := b.Used(), s.memory.State()
	b.FailAfter(0)
	v, e := s.mkVar(true, false)
	b.FailAfter(-1)
	satOK(t, e)
	if v != 0 || !todo.Contains(0) || todo.Size() != 1 {
		t.Fatal("complete native columns and queue/todo publication", v)
	}
	if b.Used() != before || s.memory.State() != raw {
		t.Fatal("first variable allocated native column backing through Runtime")
	}
	satVariableAssertFirstPrefix(t, s)
	satScalarAuditOwner(t, s)
	satOK(t, s.checkPhysicalOwners())
}

func TestSATSolverVariableTrailAndConcretePhysicalOwnership(t *testing.T) {
	s, _, b := satVariableStateFixture(t)
	for i := 0; i < 3; i++ {
		_, e := s.mkVar(false, true)
		satOK(t, e)
	}
	for _, l := range []Literal{0, 3, 4} {
		satOK(t, s.trailPush(l))
	}
	for i, want := range []Literal{0, 3, 4} {
		got, e := s.trailRead(uint32(i))
		satOK(t, e)
		if got != want {
			t.Fatal(got, want)
		}
	}
	satOK(t, s.checkPhysicalOwners())
	extra, e := s.watches.store.NewList()
	satOK(t, e)
	if e = s.checkPhysicalOwners(); e != ErrSATOwner {
		t.Fatal("registry is not a concrete owner", e)
	}
	satOK(t, s.watches.store.Free(extra))
	satOK(t, s.checkPhysicalOwners())
	if s.Stats().TotalBytes >= b.Used() {
		t.Fatal("borrowed todo must remain separately owned", s.Stats(), b.Used())
	}
}
func TestSATSolverVariableConstructorRefusalAndBorrowedTodo(t *testing.T) {
	failures, successes := 0, 0
	for fail := int64(0); fail < 80; fail++ {
		b := arena.NewBudget(^uint64(0))
		todo, e := newSATSolverUintSet(b)
		satOK(t, e)
		satOK(t, todo.Insert(19))
		mem, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b})
		satOK(t, e)
		base := b.Used()

		b.FailAfter(fail)
		s, e := newSATSolverVariableState(todo, b, arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{Budget: b}}, mem)
		if e != nil {
			failures++
			if !satRuntimeActualRefusal(mem, e) {
				t.Fatal(fail, e)
			}
		} else {
			successes++
		}
		if s != nil {
			satScalarAuditOwner(t, s)
		} else {
			satRuntimeAuditBlocks(t, mem)
		}
		b.FailAfter(-1)
		if s != nil {
			satOK(t, s.Close())
		}
		records := testFieldBySourceName(reflect.ValueOf(mem).Elem(), "blocks")
		retainedDirectory := uint64(records.Cap()) * uint64(records.Type().Elem().Size())
		if b.Used() != base+retainedDirectory || mem.Stats().LiveBlocks != 0 || mem.Stats().PhysicalBytes != 0 || mem.Stats().InitializationBytes != 0 || !todo.Contains(19) {
			t.Fatal("constructor cannot own todo", fail, b.Used(), base)
		}
		satOK(t, todo.Close())
		satOK(t, mem.Close())
		if successes != 0 {
			break
		}
	}
	if failures == 0 || successes != 1 {
		t.Fatal(failures, successes)
	}
}
func TestSATSolverVariableNativeWatchPublicationAndColdGuards(t *testing.T) {
	// The first variable's native columns complete without a reservation;
	// the already constructed queue has room for its first insertion.
	v, _, budget := satVariableStateFixture(t)
	budget.FailAfter(0)
	index, err := v.mkVar(false, true)
	budget.FailAfter(-1)
	satOK(t, err)
	if index != 0 || len(v.watches.lists) != 2 || v.NumVars() != 1 || v.mkVarCount != 1 {
		t.Fatal("native watch/column publication", index, v.Stats())
	}
	index, err = v.mkVar(false, true)
	satOK(t, err)
	if index != 1 || len(v.watches.lists) != 4 || v.mkVarCount != 2 {
		t.Fatal("subsequent variable publication", index, v.Stats())
	}
	s, _, _ := satVariableStateFixture(t)
	copyOwner := *s
	if _, e := copyOwner.mkVar(false, true); e != ErrSATOwner {
		t.Fatal(e)
	}
	var zero satSolverVariableState
	if e := zero.Close(); e != ErrSATOwner {
		t.Fatal(e)
	}
}

func TestSATSolverNativeLevelsCurrentBorrowAndGuards(t *testing.T) {
	v, _, b := satVariableStateFixture(t)
	_, e := v.mkVar(false, true)
	satOK(t, e)
	view := v.clauseState()
	satOK(t, v.setLevel(0, 7))
	for i := 0; i < 100; i++ {
		_, e = v.mkVar(false, true)
		satOK(t, e)
	}
	used, raw := b.Used(), v.memory.State()
	b.FailAfter(0)
	satOK(t, v.setLevel(0, 9))
	level, e := view.level(0)
	satOK(t, e)
	n, e := v.numVarsChecked()
	satOK(t, e)
	b.FailAfter(-1)
	if level != 9 || n != 101 || b.Used() != used || v.memory.State() != raw {
		t.Fatal("native level/count borrow lost current member or touched allocation", level, n)
	}
	copied := *v
	if _, e = copied.levelVar(0); !errors.Is(e, ErrSATOwner) {
		t.Fatal("copied level owner accepted", e)
	}
	_, e = v.levelVar(BoolVar(n))
	want := arena.ErrBounds
	if satStorageSourceDebug {
		want = ErrSATAssertion
	}
	if !errors.Is(e, want) {
		t.Fatal("native level bounds", e)
	}
	satScalarAuditOwner(t, v)
	satOK(t, v.Close())
	if _, e = view.level(0); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("closed level borrow accepted", e)
	}
}
