// Source: src/sat/sat_solver.cpp:2620–2756; src/util/vector.h:389–405.
package sat

import (
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestSATSolverUserScopeSourcePushPopPreservesOriginalFormula(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	for i := 0; i < 2; i++ {
		_, e := s.MkVar(false, true)
		satOK(t, e)
	}
	satOK(t, s.MkClause([]Literal{0, 2}))
	satOK(t, s.UserPush())
	satOK(t, s.MkClause([]Literal{1}))
	satOK(t, s.MkClause([]Literal{3}))
	got, e := s.Check(nil)
	satOK(t, e)
	if got != util.LFalse {
		t.Fatal(got)
	}
	satOK(t, s.UserPop(1))
	got, e = s.Check(nil)
	satOK(t, e)
	if got != util.LTrue {
		t.Fatal(got)
	}
}
func TestSATSolverOwningWatchShrinkRetainsOuterCapacity(t *testing.T) {
	w, _ := satVariableWatchFixture(t)
	for i := 0; i < 4; i++ {
		satOK(t, w.PushEmpty())
	}
	size := uint32(len(w.lists))
	cap := cap(w.lists)
	first := w.lists[0]
	tail := w.lists[size-1]
	satOK(t, w.Shrink(2))
	if len(w.lists) != 2 || cap != int(w.Stats().Capacity) || !w.store.IsLive(first) || w.store.IsLive(tail) {
		t.Fatal("actual shrinking members/backing")
	}
	satOK(t, w.PushEmpty())
	if len(w.lists) != 3 || cap != int(w.Stats().Capacity) {
		t.Fatal("destroyed slot placement")
	}
}
