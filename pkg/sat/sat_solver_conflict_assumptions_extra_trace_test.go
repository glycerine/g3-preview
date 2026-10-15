package sat

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

// Source: src/sat/sat_solver.cpp:1825. The trace observes the mark before
// the function mutates it, including an already-marked return.
func TestSATUnsatCoreAntecedentSourceTraceBeforeMark(t *testing.T) {
	s := satConflictFixture(t, 1)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("sat")
	s.traceTags = tags
	satOK(t, s.processAntecedentForUnsatCore(0))
	satOK(t, s.processAntecedentForUnsatCore(0))
	marked, e := s.analysis.marked(0)
	satOK(t, e)
	if !marked {
		t.Fatal("antecedent was not marked")
	}
	satOK(t, s.analysis.resetUnmark(0))
	want := ""
	if tags.ShouldTrace("sat") {
		want = satClauseTraceRecord("sat", "process_antecedent_for_unsat_core", "1825", "0 -\n") +
			satClauseTraceRecord("sat", "process_antecedent_for_unsat_core", "1825", "0 +\n")
	}
	if got := output.String(); got != want {
		t.Fatalf("antecedent trace = %q, want %q", got, want)
	}
}

// Source: src/sat/sat_solver.cpp:1885-1894. The non-null conflicting literal
// is printed after the placeholder lemma and before antecedent processing.
func TestSATConflictInitNotLiteralSourceTrace(t *testing.T) {
	s := satConflictFixture(t, 2)
	p := s.analysis.propagation
	satConflictPush(t, s)
	satOK(t, p.assign(0, satJustification{}))
	satOK(t, p.assign(2, newSATBinaryJustification(1)))
	p.setConflict(newSATBinaryJustification(1), 2)
	s.conflictLevel = 1
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("sat")
	s.traceTags = tags
	ok, e := s.resolveConflictForInit()
	satOK(t, e)
	if !ok {
		t.Fatal("source init resolution did not complete")
	}
	want := ""
	if tags.ShouldTrace("sat") {
		want = satClauseTraceRecord("sat", "resolve_conflict_for_init", "1892", "not_l: 1\n")
	}
	if got := output.String(); got != want {
		t.Fatalf("init trace = %q, want %q", got, want)
	}
}
