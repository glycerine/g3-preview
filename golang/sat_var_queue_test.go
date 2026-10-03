package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"reflect"
	"testing"
)

func TestSATVarQueueRealActivityPriorityEventsAndLifecycle(t *testing.T) {
	act, e := NewSATActivityDomain(nil)
	satOK(t, e)
	defer act.Close()
	for _, v := range []uint32{1, 5, 5, 3} {
		satOK(t, act.Push(v))
	}
	q, e := NewSATVarQueue(act, nil)
	satOK(t, e)
	defer q.Close()
	for v := satBoolVar(0); v < 4; v++ {
		satOK(t, q.MkVar(v))
	}
	satOK(t, act.Set(0, 9))
	satOK(t, q.ActivityIncreased(0))
	v, e := q.NextVar()
	if v != 0 || e != nil {
		t.Fatal(v, e)
	}
	satOK(t, q.DelVar(3))
	satOK(t, q.UnassignVar(0))
	v, e = q.NextVar()
	if v != 0 || e != nil {
		t.Fatal(v, e)
	}
	clone, e := q.Clone(nil)
	satOK(t, e)
	defer clone.Close()
	satOK(t, q.Reset())
	if clone.Empty() {
		t.Fatal("copy must deep copy")
	}
	moved, e := clone.MoveClone(nil)
	satOK(t, e)
	defer moved.Close()
	if clone.Empty() || moved.Empty() || clone.Size() != 0 {
		t.Fatal("move")
	}
	if _, ok := reflect.TypeOf(q).MethodByName("Assign"); ok {
		t.Fatal("source assignment deleted")
	}
	satOK(t, q.ActivityIncreased(100))
	satOK(t, q.DelVar(100))
	if _, e = q.NextVar(); !errors.Is(e, ErrSATAssertion) && !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
}
func TestSATVarQueueSourceGhostAndEmptyReset(t *testing.T) {
	act, e := NewSATActivityDomain(nil)
	satOK(t, e)
	defer act.Close()
	satOK(t, act.Push(2))
	satOK(t, act.Push(1))
	b := arena.NewBudget(1 << 20)
	q, e := NewSATVarQueue(act, b)
	satOK(t, e)
	defer q.Close()
	satOK(t, q.MkVar(0))
	b.FailAfter(0)
	if e = q.MkVar(1); !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	if !q.Contains(1) {
		t.Fatal("inverse publication must survive push failure")
	}
	if q.Size() != 1 {
		t.Fatal(q.Size())
	}
	b.FailAfter(-1)
	v, e := q.NextVar()
	if v != 0 || e != nil {
		t.Fatal(v, e)
	}
	satOK(t, q.Reset())
	if !q.Contains(1) {
		t.Fatal("source empty reset preserves ghost")
	}
	if e = q.DelVar(1); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("guard before actual OOB", e)
	}
}
func TestSATVarQueueUnsignedNegativeIndexBoundary(t *testing.T) {
	a, e := NewSATActivityDomain(nil)
	satOK(t, e)
	defer a.Close()
	q, e := NewSATVarQueue(a, nil)
	satOK(t, e)
	defer q.Close()
	for _, v := range []satBoolVar{1 << 31, ^satBoolVar(0)} {
		for _, op := range []func(satBoolVar) error{q.ActivityIncreased, q.DelVar, q.UnassignVar, q.MkVar} {
			if e = op(v); !errors.Is(e, arena.ErrBounds) {
				t.Fatal(v, e)
			}
		}
	}
	for _, v := range []satBoolVar{128, 1<<31 - 1} {
		satOK(t, q.ActivityIncreased(v))
		satOK(t, q.DelVar(v))
	}
}
