package sat

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

// Source: src/sat/sat_solver.cpp:3285-3297. Each antecedent row is printed
// before union, and the final core follows ordered set iteration.
func TestSATConsequenceCoreFixupSourceTraceOrder(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	satOK(t, s.conflict.core.Push(2))
	satOK(t, s.conflict.core.Push(4))
	var first, second satSolverIndexSet
	satOK(t, first.insert(0))
	satOK(t, first.insert(3))
	satOK(t, second.insert(1))
	satOK(t, s.antecedents.insert(1, &first))
	satOK(t, s.antecedents.insert(2, &second))
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("sat")
	s.traceTags = tags
	satOK(t, s.fixupConsequenceCore())
	if n, e := s.conflict.core.Size(); e != nil || n != 3 {
		t.Fatal(n, e)
	}
	for i, want := range []uint32{0, 3, 1} {
		got, e := s.conflict.core.Read(uint32(i))
		if e != nil || got != want {
			t.Fatal(i, got, want, e)
		}
	}
	want := ""
	if tags.ShouldTrace("sat") {
		want = satClauseTraceRecord("sat", "fixup_consequence_core", "3287", "1 2\n") +
			satClauseTraceRecord("sat", "fixup_consequence_core", "3289", "1: 0 -1 \n") +
			satClauseTraceRecord("sat", "fixup_consequence_core", "3289", "2: -0 \n") +
			satClauseTraceRecord("sat", "fixup_consequence_core", "3297", "0 -1 -0\n")
	}
	if got := output.String(); got != want {
		t.Fatalf("fixup trace = %q, want %q", got, want)
	}
}

// Source: src/sat/sat_solver.cpp:3307,3366. The same borrowed assumptions
// reach either the bounded or ordinary consequence body before map reset.
func TestSATConsequenceEntryAssumptionSourceTrace(t *testing.T) {
	for route := 0; route < 2; route++ {
		t.Run([]string{"ordinary", "bounded"}[route], func(t *testing.T) {
			s, out := satConsequenceFaultFixture(t, route)
			var output bytes.Buffer
			tags := diagnostic.NewTraceTagsWithWriter(&output)
			tags.Enable("sat")
			s.traceTags = tags
			_, err := s.getConsequences([]Literal{0}, []BoolVar{1, 2, 3, 4}, out)
			satOK(t, err)
			line, function := "3366", "get_consequences"
			if route == 1 {
				line, function = "3307", "get_bounded_consequences"
			}
			record := satClauseTraceRecord("sat", function, line, "0\n")
			count := strings.Count(output.String(), record)
			if tags.ShouldTrace("sat") {
				if count != 1 {
					t.Fatalf("assumption record count = %d in %q", count, output.String())
				}
			} else if output.Len() != 0 {
				t.Fatalf("erased TRACE emitted %q", output.String())
			}
		})
	}
}
