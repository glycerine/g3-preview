package sat

// Source: src/sat/sat_clause_use_list.cpp:36-54; sat_simplifier.cpp:388-402.

import (
	"testing"
)

func TestSATSimplifierRemovedUseListEntrySkippedBeforeCollectorGuard(t *testing.T) {
	s, l := satSimplifierFixture(t, 3)
	satOK(t, s.useList.Init(3))
	satOK(t, s.kernel.InitVisited(3))
	a := l.propagation.clauses
	c1, err := a.MkClause([]Literal{0, 2}, false)
	satOK(t, err)
	dead, err := a.MkClause([]Literal{0, 2, 4}, false)
	satOK(t, err)
	live, err := a.MkClause([]Literal{0, 2, 5}, false)
	satOK(t, err)
	satOK(t, s.useList.Insert(dead))
	satOK(t, s.useList.Insert(live))
	satOK(t, a.SetRemoved(dead, true))
	satOK(t, s.useList.Erase(dead))
	s.kernel.subCounter = 100
	satOK(t, s.kernel.collectSubsumed1Core(c1, s.bsCS, s.literalStore, s.bsLS, 0))
	if s.bsCS.Size() != 1 {
		t.Fatal("only live clause collected", s.bsCS.Size())
	}
	got, err := s.bsCS.Get(0)
	satOK(t, err)
	if got != live {
		t.Fatal("collector exposed removed clause", got, live)
	}
	view, err := s.useList.Get(0)
	satOK(t, err)
	record, err := view.record()
	satOK(t, err)
	if len(record.payload) != 1 || record.payload[0] != live {
		t.Fatal("lexical iterator did not compact removed entry", record.payload)
	}
}
