// Source: src/sat/sat_simplifier.cpp:1296,1302,1315.
package sat

import (
	"bytes"
	"strings"
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
	tags.Enable("resolution_detail")
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
			"0 num_pos: 1 neg_pos: 1\n------------------------------------------------\n" +
			"-------- [resolution] try_eliminate src/sat/sat_simplifier.cpp:1338 ---------\n" +
			"0 num_pos: 1 neg_pos: 1 before_lits: 6\n------------------------------------------------\n"
	}
	if tags.ShouldTrace("resolution_detail") {
		want += "-------- [resolution_detail] try_eliminate src/sat/sat_simplifier.cpp:1354 ---------\n" +
			"collecting number of after_clauses\n------------------------------------------------\n"
	}
	if tags.ShouldTrace("resolution") {
		want += "-------- [resolution] try_eliminate src/sat/sat_simplifier.cpp:1375 ---------\n" +
			"found var to eliminate, before: 2 after: 0\n------------------------------------------------\n"
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

func TestSATSimplifierTooManyResolventsSourceTrace(t *testing.T) {
	s, l := satSimplifierFixture(t, 9)
	satOK(t, s.useList.Init(9))
	satOK(t, s.kernel.InitVisited(9))
	s.kernel.elimCounter = 1000
	for _, base := range []uint32{0, 1} {
		for i := uint32(0); i < 3; i++ {
			a := Literal(2 + 2*i)
			last := Literal(14)
			if base == 1 {
				a += 6
				last = 16
			}
			c, err := l.MkClauseCore(3, []Literal{Literal(base), a, last}, false)
			satOK(t, err)
			satOK(t, s.useList.Insert(c))
		}
	}
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("resolution")
	l.traceTags = tags
	yes, err := s.tryEliminate(0)
	satOK(t, err)
	if yes || l.propagation.converter.Size() != 0 {
		t.Fatal("source refusal before converter publication")
	}
	want := ""
	if tags.ShouldTrace("resolution") {
		want = "-------- [resolution] try_eliminate src/sat/sat_simplifier.cpp:1315 ---------\n" +
			"0 num_pos: 3 neg_pos: 3\n------------------------------------------------\n" +
			"-------- [resolution] try_eliminate src/sat/sat_simplifier.cpp:1338 ---------\n" +
			"0 num_pos: 3 neg_pos: 3 before_lits: 18\n------------------------------------------------\n" +
			"-------- [resolution] try_eliminate src/sat/sat_simplifier.cpp:1369 ---------\n" +
			"too many after clauses: 7\n------------------------------------------------\n"
	}
	if got := output.String(); got != want || strings.Contains(got, "src/sat/sat_simplifier.cpp:1375") {
		t.Fatalf("source too-many refusal trace: got %q, want %q", got, want)
	}
}

func TestSATSimplifierActualResolventBodiesSourceTraces(t *testing.T) {
	s, l := satSimplifierFixture(t, 4)
	satOK(t, s.useList.Init(4))
	satOK(t, s.kernel.InitVisited(4))
	s.kernel.elimCounter = 1000
	for _, literals := range [][]Literal{{0, 2, 4}, {1, 4, 6}} {
		c, err := l.MkClauseCore(3, literals, false)
		satOK(t, err)
		satOK(t, s.useList.Insert(c))
	}
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("resolution_detail")
	tags.Enable("resolution_new_cls")
	l.traceTags = tags
	yes, err := s.tryEliminate(0)
	satOK(t, err)
	if !yes || l.propagation.converter.Size() != 1 {
		t.Fatal("live resolvent publication")
	}
	want := ""
	if tags.ShouldTrace("resolution_detail") {
		want += "-------- [resolution_detail] try_eliminate src/sat/sat_simplifier.cpp:1354 ---------\n" +
			"collecting number of after_clauses\n------------------------------------------------\n" +
			"-------- [resolution_detail] try_eliminate src/sat/sat_simplifier.cpp:1365 ---------\n" +
			"(0 1 2)\n(-0 2 3)\n-->\n1 2 3 \n------------------------------------------------\n"
	}
	if tags.ShouldTrace("resolution_new_cls") {
		want += "-------- [resolution_new_cls] try_eliminate src/sat/sat_simplifier.cpp:1401 ---------\n" +
			"(0 1 2)\n(-0 2 3)\n-->\n1 2 3\n------------------------------------------------\n"
	}
	if got := output.String(); got != want {
		t.Fatalf("source resolvent body traces: got %q, want %q", got, want)
	}
}
