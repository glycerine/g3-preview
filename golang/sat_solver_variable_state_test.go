// Copyright (c) 2011 Microsoft Corporation. Licensed under the MIT license.
package g3

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func satVariableStateFixture(t *testing.T) (*satSolverVariableState, *satSolverUintSet, *arena.Budget) {
	t.Helper()
	b := arena.NewBudget(^uint64(0))
	todo, e := newSATSolverUintSet(b)
	satOK(t, e)
	mem, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b})
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
		if s.numVars() != i+1 || s.mkVarCount != i+1 || s.modelIsCurrent || !s.modelKnown || s.qhead != 0 || s.scopeLevel != 0 || s.activityInc != 128 {
			t.Fatal(s.Stats())
		}
		if !s.queue.Contains(v) || !todo.Contains(i) {
			t.Fatal("actual final queue/todo callbacks", i)
		}
		for _, l := range []satLiteral{satMakeLiteral(v, false), satMakeLiteral(v, true)} {
			value, e := s.valueLiteral(l)
			satOK(t, e)
			if value != lUndef {
				t.Fatal(l, value)
			}
		}
		if satScalarTestUint(t, &s.level, i) != ^uint32(0) || s.justification.data[i] != (satJustification{}) || satAssignmentTestValue(t, &s.phase, i) != 2 || satAssignmentTestValue(t, &s.previousPhase, i) != 2 {
			t.Fatal("source values", i)
		}
		if satAssignmentTestValue(t, &s.decision, i) != int8(map[bool]int{true: 1}[i%3 != 0]) || satAssignmentTestValue(t, &s.external, i) != int8(i%2) {
			t.Fatal(i)
		}
	}
	view := s.clauseState()
	if view.variables != s {
		t.Fatal("source char assignment")
	}
	satOK(t, s.assignment.write(0, int8(lTrue)))
	value, e := view.value(0)
	satOK(t, e)
	if value != lTrue {
		t.Fatal("no copied assignment view", value)
	}
	todoBytes := todo.Stats().OwnerBytes + todo.Stats().MembershipCapacity + 4*todo.Stats().ValuesCapacity
	if s.Stats().TotalBytes+todoBytes+s.memory.Stats().TotalBytes != b.Used() {
		t.Fatal(s.Stats(), todo.Stats(), b.Used())
	}
}

func satVariableAssertFirstPrefix(t *testing.T, s *satSolverVariableState) {
	t.Helper()
	sizes := []int{len(s.watches.lists), int(satAssignmentTestSize(t, &s.assignment)), len(s.justification.data), int(satAssignmentTestSize(t, &s.decision)), int(satAssignmentTestSize(t, &s.eliminated)), int(satAssignmentTestSize(t, &s.external)), int(s.activity.Size()), int(satScalarTestUintSize(t, &s.level)), int(satAssignmentTestSize(t, &s.mark)), int(satAssignmentTestSize(t, &s.literalMark)), int(satAssignmentTestSize(t, &s.phase)), int(satAssignmentTestSize(t, &s.previousPhase)), int(satAssignmentTestSize(t, &s.assignedSinceGC))}
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
func TestSATSolverMkVarAllActualReservationPrefixes(t *testing.T) {
	failures, successes := 0, 0
	seenHeapBeforeTodo, seenTodoBeforeValue := false, false
	for fail := int64(0); fail < 160; fail++ {
		t.Run(fmt.Sprint(fail), func(t *testing.T) {
			s, _, b := satVariableStateFixture(t)
			b.FailAfter(fail)
			v, e := s.mkVar(true, false)
			if v != 0 {
				t.Fatal(v)
			}
			satVariableAssertFirstPrefix(t, s)
			// Census is before fault disable: every raw scalar block has a named owner.
			satScalarAuditOwner(t, s)
			if e2 := s.checkPhysicalOwners(); e2 != nil {
				t.Fatal(fail, e2)
			}
			if e != nil {
				failures++
				if s.queue.Size() == 1 && !s.elimTodo.Contains(0) {
					seenHeapBeforeTodo = true
				}
				if s.elimTodo.Contains(0) && s.elimTodo.Size() == 0 {
					seenTodoBeforeValue = true
				}
				if !satAssignmentActualRefusal(s.memory, e) {
					t.Fatal(fail, e)
				}
			} else {
				successes++
			}
			b.FailAfter(-1)
		})
		if successes != 0 {
			break
		}
	}
	if failures == 0 || successes != 1 {
		t.Fatal(failures, successes)
	}
	if !seenHeapBeforeTodo || !seenTodoBeforeValue {
		t.Fatal("missing actual final publication boundaries", seenHeapBeforeTodo, seenTodoBeforeValue)
	}
	t.Log("all actual first-variable reservations", failures, successes)
}

func TestSATSolverMkVarLateHeapIndexPublicationBeforeRefusal(t *testing.T) {
	found := false
	for fail := int64(0); fail < 30; fail++ {
		t.Run(fmt.Sprint(fail), func(t *testing.T) {
			s, todo, b := satVariableStateFixture(t)
			for i := 0; i < 4; i++ {
				_, e := s.mkVar(false, true)
				satOK(t, e)
			}
			b.FailAfter(fail)
			v, e := s.mkVar(false, true)
			satScalarAuditOwner(t, s)
			if e2 := s.checkPhysicalOwners(); e2 != nil {
				t.Fatal(e2)
			}
			if e != nil && !satAssignmentActualRefusal(s.memory, e) {
				t.Fatal(e)
			}
			if e != nil && s.queue.Contains(4) && s.queue.Size() == 4 {
				found = true
				if v != 4 || s.numVars() != 5 || s.queue.indices[4] != 5 || len(s.queue.values) != 5 || todo.Contains(4) {
					t.Fatal(v, s.Stats(), s.queue.indices, todo.Stats())
				}
			}
			b.FailAfter(-1)
		})
		if found {
			break
		}
	}
	if !found {
		t.Fatal("source heap index-before-push refusal not reached")
	}
}

func TestSATSolverVariableTrailAndConcretePhysicalOwnership(t *testing.T) {
	s, _, b := satVariableStateFixture(t)
	for i := 0; i < 3; i++ {
		_, e := s.mkVar(false, true)
		satOK(t, e)
	}
	for _, l := range []satLiteral{0, 3, 4} {
		satOK(t, s.trailPush(l))
	}
	for i, want := range []satLiteral{0, 3, 4} {
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
		mem, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b})
		satOK(t, e)
		base := b.Used()
		b.FailAfter(fail)
		s, e := newSATSolverVariableState(todo, b, arena.VectorConfig{PageSize: 64, Descriptors: arena.Config{Budget: b}}, mem)
		if e != nil {
			failures++
			if !errors.Is(e, arena.ErrInjectedFailure) {
				t.Fatal(fail, e)
			}
		} else {
			successes++
		}
		b.FailAfter(-1)
		if s != nil {
			satOK(t, s.Close())
		}
		if b.Used() != base || !todo.Contains(19) {
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
func TestSATSolverVariableRetainedSourceRetryAndColdGuards(t *testing.T) {
	// Refusal immediately after first watch insertion is a real completed prefix;
	// repeating source mk_var starts again at level.size, not a repaired cursor.
	found := false
	for fail := int64(0); fail < 30; fail++ {
		s, _, b := satVariableStateFixture(t)
		_, e := s.mkVar(false, true)
		satOK(t, e)
		b.FailAfter(fail)
		_, e = s.mkVar(false, true)
		b.FailAfter(-1)
		if e != nil && len(s.watches.lists) == 3 {
			found = true
			v, e := s.mkVar(false, true)
			satOK(t, e)
			if v != 1 || len(s.watches.lists) != 5 || s.mkVarCount != 3 {
				t.Fatal(v, s.Stats())
			}
			break
		}
	}
	if !found {
		t.Fatal("missing concrete partial watch prefix")
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
