package sat

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

// Source: src/sat/sat_solver.cpp:2818-2837. The learned-inclusive traversal
// visits its learned watch before the original watch at the next index.
func TestSATCollectBinaryClausesSourceTraceOrder(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	for i := 0; i < 2; i++ {
		_, e := s.MkVar(false, true)
		satOK(t, e)
	}
	_, e := s.Lifecycle.MkClauseCore(2, []Literal{0, 2}, false)
	satOK(t, e)
	_, e = s.Lifecycle.MkClauseCore(2, []Literal{1, 3}, true)
	satOK(t, e)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("cleanup_bug")
	s.traceTags = tags
	original, e := s.collectBinClauses(nil, false)
	satOK(t, e)
	all, e := s.collectBinClauses(nil, true)
	satOK(t, e)
	if len(original) != 1 || original[0] != newSATSolverBinPair(0, 2) ||
		len(all) != 2 || all[0] != newSATSolverBinPair(1, 3) || all[1] != newSATSolverBinPair(0, 2) {
		t.Fatal(original, all)
	}
	want := ""
	if tags.ShouldTrace("cleanup_bug") {
		want = satClauseTraceRecord("cleanup_bug", "collect_bin_clauses", "2834", "collected: 0 1\n") +
			satClauseTraceRecord("cleanup_bug", "collect_bin_clauses", "2834", "collected: -0 -1\n") +
			satClauseTraceRecord("cleanup_bug", "collect_bin_clauses", "2834", "collected: 0 1\n")
	}
	if got := output.String(); got != want {
		t.Fatalf("collection trace = %q, want %q", got, want)
	}
}
