// Source: src/sat/sat_types.h; src/sat/sat_probing.cpp.
package sat

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"reflect"
	"testing"
)

func TestSATSolverProbingAssignedNativeGrowth(t *testing.T) {
	s, _ := satProbingFixture(t, 8)
	before := s.propagation().memory.Stats()
	cacheLimit, e := s.propagation().memory.AllocationSize()
	satOK(t, e)
	s.cacheLimit = cacheLimit
	used := s.budget.Used()
	s.budget.FailAfter(0)
	defer s.budget.FailAfter(-1)
	for i := uint32(0); i < 1024; i++ {
		satOK(t, s.assigned.Insert(Literal(i)))
		satOK(t, s.assigned.Insert(Literal(i)))
	}
	if s.assigned.size() != 1024 || s.budget.Used() != used || s.propagation().memory.Stats() != before {
		t.Fatal("native unique literal storage consumed Runtime resources")
	}
	for i, l := range s.assigned.values {
		if l != Literal(i) || !s.assigned.contains(l) {
			t.Fatal("unique insertion order", i, l)
		}
	}
	if s.assigned.contains(2048) {
		t.Fatal("unset literal")
	}
	if e := s.assigned.Insert(Literal(^uint32(0))); !errors.Is(e, arena.ErrBounds) || s.assigned.size() != 1024 {
		t.Fatal("invalid wrapped literal changed set", e)
	}
	satProbingAudit(t, s, nil, nil)
	s.budget.FailAfter(-1)
	satOK(t, s.cacheBins(0, 0))
	if !s.cache.blockLive() {
		t.Fatal("native literal storage changed Runtime cache cutoff")
	}
	satProbingAudit(t, s, nil, nil)
}

func TestSATSolverProbingSourceFinalizePermitsNextRound(t *testing.T) {
	s, _ := satProbingFixture(t, 1)
	done, e := s.run(true)
	satOK(t, e)
	if !done {
		t.Fatal("first normal round")
	}
	done, e = s.run(true)
	satOK(t, e)
	if !done {
		t.Fatal("second normal round")
	}
	satOK(t, s.process(0))
	satProbingAudit(t, s, nil, nil)
}

func TestSATSolverProbingAssignedResetAndReuse(t *testing.T) {
	var set satLiteralSetStorage
	for _, l := range []Literal{14, 2, 7, 14, 0} {
		satOK(t, set.Insert(l))
	}
	want := []Literal{14, 2, 7, 0}
	if !reflect.DeepEqual(set.values, want) {
		t.Fatal(set.values)
	}
	memberCapacity, valueCapacity := cap(set.membership), cap(set.values)
	set.Reset()
	if set.contains(14) || set.size() != 0 || cap(set.membership) != memberCapacity || cap(set.values) != valueCapacity {
		t.Fatal("reset must forget membership and retain backing")
	}
	// Regrowing within retained capacity must not resurrect old membership.
	satOK(t, set.Insert(2))
	satOK(t, set.Insert(14))
	if !reflect.DeepEqual(set.values, []Literal{2, 14}) || set.contains(7) || set.contains(0) {
		t.Fatal(set.values)
	}
	set.Finalize()
	if set.membership != nil || set.values != nil {
		t.Fatal("finalize retained backing")
	}
	satOK(t, set.Insert(7))
	if !set.contains(7) || set.size() != 1 {
		t.Fatal("finalized set is reusable")
	}
}
