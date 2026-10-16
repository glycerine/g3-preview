// Source: src/sat/sat_simplifier.cpp:1296,1302,1315.
package sat

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestSATSimplifierActualVariableEliminationSourceTraces(t *testing.T) {
	s, l := satSimplifierFixture(t, 3)
	satOK(t, s.useList.Init(3))
	satOK(t, s.kernel.InitVisited(3))
	s.kernel.subCounter = 1000
	s.kernel.elimCounter = 1000
	for _, literals := range [][]Literal{{0, 2, 4}, {1, 3, 4}} {
		c, err := l.MkClauseCore(3, literals, false)
		satOK(t, err)
		satOK(t, s.useList.Insert(c))
	}
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("resolution_bug")
	tags.Enable("resolution")
	l.traceTags = tags
	yes, err := s.tryEliminate(0)
	satOK(t, err)
	if !yes {
		t.Fatal("live variable elimination")
	}
	want := ""
	if tags.ShouldTrace("resolution_bug") {
		want += "-------- [resolution_bug] try_eliminate src/sat/sat_simplifier.cpp:1302 ---------\n" +
			"processing: 0\n------------------------------------------------\n"
	}
	if tags.ShouldTrace("resolution") {
		want += "-------- [resolution] try_eliminate src/sat/sat_simplifier.cpp:1315 ---------\n" +
			"0 num_pos: 1 neg_pos: 1\n------------------------------------------------\n"
	}
	if tags.ShouldTrace("resolution_bug") {
		want += "-------- [resolution_bug] remove_clauses src/sat/sat_simplifier.cpp:1296 ---------\n" +
			"del_clause (elim_var): (0 1 2)x+\n------------------------------------------------\n" +
			"-------- [resolution_bug] remove_clauses src/sat/sat_simplifier.cpp:1296 ---------\n" +
			"del_clause (elim_var): (-0 -1 2)x+\n------------------------------------------------\n"
	}
	if got := output.String(); got != want {
		t.Fatalf("source elimination trace: got %q, want %q", got, want)
	}
}
