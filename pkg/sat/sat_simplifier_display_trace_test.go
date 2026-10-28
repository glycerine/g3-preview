package sat

// Source: src/sat/sat_simplifier.cpp:810-833.

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestSATActualSimplifierSubsumptionSolverDisplaySourceTrace(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	_, err := s.MkVar(true, true)
	satOK(t, err)
	if s.simplifier.cleaner.solver != s {
		t.Fatal("simplifier does not borrow its full source solver")
	}
	var before bytes.Buffer
	satOK(t, s.traceUserDisplayInto(&before))
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("subsumption_bug")
	s.Lifecycle.traceTags = tags
	satOK(t, s.simplifier.Run(false))
	if !tags.ShouldTrace("subsumption_bug") {
		if output.Len() != 0 {
			t.Fatal("build-erased source trace", output.String())
		}
		return
	}
	var after bytes.Buffer
	satOK(t, s.traceUserDisplayInto(&after))
	want := "-------- [subsumption_bug] subsume src/sat/sat_simplifier.cpp:813 ---------\n" +
		before.String() + "------------------------------------------------\n" +
		"-------- [subsumption_bug] subsume src/sat/sat_simplifier.cpp:832 ---------\n" +
		after.String() + "------------------------------------------------\n"
	if got := output.String(); got != want {
		t.Fatalf("source solver display traces: got %q, want %q", got, want)
	}
	if strings.Count(output.String(), "src/sat/sat_simplifier.cpp:832") != 1 {
		t.Fatal("empty todo exits after one loop display")
	}
}
