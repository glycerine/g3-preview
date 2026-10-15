package sat

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

// Source: src/sat/sat_solver.cpp:1184. Search setup emits its solver display
// after resetting the search counters and before initial propagation.
func TestSATSearchInitSourceDisplayTrace(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	_, err := s.MkVar(false, true)
	satOK(t, err)
	var display bytes.Buffer
	satOK(t, s.traceUserDisplayInto(&display))
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("sat")
	s.traceTags = tags
	_, err = s.checkCore(nil)
	satOK(t, err)
	if !s.gc.initialized {
		t.Fatal("search initialization was skipped")
	}
	if !tags.ShouldTrace("sat") {
		if output.Len() != 0 {
			t.Fatalf("erased trace = %q", output.String())
		}
		return
	}
	want := satClauseTraceRecord("sat", "init_search", "1184", display.String())
	if got := output.String(); !strings.HasPrefix(got, want) || strings.Count(got, "init_search src/sat/sat_solver.cpp:1184") != 1 {
		t.Fatalf("search startup record = %q, want prefix %q", got, want)
	}
}
