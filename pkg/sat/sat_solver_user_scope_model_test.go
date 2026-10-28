package sat

// Source: src/sat/sat_solver.cpp:2620–2756; src/util/vector.h:389–405.

import (
	"github.com/glycerine/g3/pkg/util/lbool"
	"testing"
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
	if got != lbool.False {
		t.Fatal(got)
	}
	satOK(t, s.UserPop(1))
	got, e = s.Check(nil)
	satOK(t, e)
	if got != lbool.True {
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

// Source: src/sat/sat_solver.cpp:135-160,2818-2841.
func TestSATSolverNativeFlagsScopeTruncationAndRegrowth(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	for i := 0; i < 2; i++ {
		_, e := s.MkVar(false, true)
		satOK(t, e)
	}
	satOK(t, s.MkClause([]Literal{0, 2}))
	satOK(t, s.UserPush())
	for i := 0; i < 2; i++ {
		_, e := s.MkVar(false, true)
		satOK(t, e)
	}
	v := s.Variables
	for i := range v.mark {
		v.mark[i] = true
	}
	for i := range v.literalMark {
		v.literalMark[i] = true
	}
	for i := range v.decision {
		satOK(t, v.activity.Set(uint32(i), 100))
		v.decision[i] = false
		v.external[i] = true
		v.assignedSinceGC[i] = true
		v.eliminated[i] = i >= 2
		v.phase[i] = int8(i % 2)
		v.previousPhase[i] = int8((i + 1) % 2)
	}
	satOK(t, s.UserPop(1))
	if len(v.level) != 2 {
		t.Fatal("scope level length", len(v.level))
	}
	if len(v.phase) != 2 || len(v.previousPhase) != 2 {
		t.Fatal("scope phase lengths", len(v.phase), len(v.previousPhase))
	}
	if len(v.mark) != 2 || len(v.literalMark) != 4 {
		t.Fatal("scope mark lengths", len(v.mark), len(v.literalMark))
	}
	for _, mark := range v.mark {
		if !mark {
			t.Fatal("scope changed retained variable mark")
		}
	}
	for _, mark := range v.literalMark {
		if !mark {
			t.Fatal("scope changed retained literal mark")
		}
	}
	for i := 0; i < 2; i++ {
		if v.activity.values[i] != 100 {
			t.Fatal("scope changed retained activity", i)
		}
		if v.phase[i] != int8(i%2) || v.previousPhase[i] != int8((i+1)%2) {
			t.Fatal("scope changed retained polarities", i)
		}
		if v.decision[i] || !v.external[i] || !v.assignedSinceGC[i] || v.eliminated[i] {
			t.Fatal("scope changed retained flags", i)
		}
	}
	variable, e := s.MkVar(false, true)
	satOK(t, e)
	if variable != 2 || len(v.mark) != 3 || len(v.literalMark) != 6 || v.mark[2] || v.literalMark[4] || v.literalMark[5] {
		t.Fatal("new variable inherited removed marks", variable, v.mark, v.literalMark)
	}
	if len(v.activity.values) != 3 || v.activity.values[2] != 0 {
		t.Fatal("regrowth inherited removed activity", v.activity.values)
	}
	if len(v.level) != 3 || v.level[2] != ^uint32(0) {
		t.Fatal("regrowth inherited removed level", v.level)
	}
	if v.phase[2] != 2 || v.previousPhase[2] != 2 {
		t.Fatal("regrowth inherited removed polarities", v.phase, v.previousPhase)
	}
	if !v.decision[2] || v.external[2] || v.eliminated[2] || v.assignedSinceGC[2] {
		t.Fatal("regrowth inherited removed flags")
	}
	satFullSolverRuntimeAudit(t, s)
}
