// Source: src/sat/sat_simplifier.cpp:875–919; src/util/heap.h.
package sat

import (
	"testing"
)

func TestSATSimplifierBlockedHeapMutableActualUnsignedWeights(t *testing.T) {
	s, l := satSimplifierFixture(t, 3)
	satOK(t, s.useList.Init(3))
	c, e := l.propagation.clauses.MkClause([]Literal{0, 2, 4}, false)
	satOK(t, e)
	satOK(t, l.pushClause(false, c))
	satOK(t, s.useList.Insert(c))
	_, e = l.MkClauseCore(2, []Literal{0, 2}, false)
	satOK(t, e)
	q, e := newSATSimplifierBlockedQueue(s)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, q.Close()) })
	for _, idx := range []uint32{1, 3, 5} {
		satOK(t, q.MkVar(BoolVar(idx)))
	}
	got, e := q.NextVar()
	satOK(t, e)
	if got != 5 {
		t.Fatal("actual use+binary count minimum", got)
	}
	u, e := s.useList.Get(0)
	satOK(t, e)
	satOK(t, u.EraseNotRemoved(c))
	satOK(t, q.ActivityIncreased(1)) // exact existing move_up == source decreased
	got, e = q.NextVar()
	satOK(t, e)
	if got != 1 {
		t.Fatal("actual mutated priority source decreased", got)
	}
	got, e = q.NextVar()
	satOK(t, e)
	if got != 3 || !q.Empty() {
		t.Fatal("remaining heap key", got)
	}
}
