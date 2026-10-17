package sat

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

// Source: src/sat/sat_solver.cpp:2000,2032,2063. The opening record sees the
// solver and trail before the core reset, and the not-literal record prints
// its actual justification before antecedent processing.
func TestSATUnsatCoreSourceTraceStartJustificationAndLowerLevel(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	for i := 0; i < 2; i++ {
		_, err := s.MkVar(true, true)
		satOK(t, err)
	}
	satOK(t, s.propagation.assign(0, satJustification{}))
	_, err := s.Lifecycle.propagate(false)
	satOK(t, err)
	satOK(t, s.Lifecycle.push())
	satOK(t, s.propagation.assign(2, newSATBinaryJustification(1)))
	s.propagation.setConflict(newSATBinaryJustification(1), 2)
	satOK(t, s.conflict.assumptions.Push(0))
	satOK(t, s.conflict.assumptionSet.Insert(0))
	var before bytes.Buffer
	satOK(t, s.traceUserDisplayInto(&before))
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("sat")
	s.conflict.traceTags = tags
	ok, err := s.conflict.resolveConflict()
	satOK(t, err)
	if ok {
		t.Fatal("assumption conflict resolved as satisfiable")
	}
	if !tags.ShouldTrace("sat") {
		if output.Len() != 0 {
			t.Fatalf("erased trace = %q", output.String())
		}
		return
	}
	wantStart := satClauseTraceRecord("sat", "resolve_conflict_for_unsat_core", "2000", before.String()+"0 1: 1 \n")
	if got := output.String(); !strings.Contains(got, wantStart) {
		t.Fatalf("core start trace = %q, missing %q", got, wantStart)
	}
	wantNot := satClauseTraceRecord("sat", "resolve_conflict_for_unsat_core", "2032", "not_l: 1\nbinary -0\n")
	if got := output.String(); !strings.Contains(got, wantNot) {
		t.Fatalf("core not-literal trace = %q, missing %q", got, wantNot)
	}
	wantConsequent := satClauseTraceRecord("sat", "process_consequent_for_unsat_core", "1836",
		"processing consequent: 1\njs kind: binary -0\n")
	if got := output.String(); strings.Count(got, wantConsequent) != 3 || strings.Index(got, wantConsequent) < strings.Index(got, wantNot) {
		t.Fatalf("core consequent trace = %q, want three after not-literal %q", got, wantConsequent)
	}
	wantLower := satClauseTraceRecord("sat", "resolve_conflict_for_unsat_core", "2063", "0 at level 0\n")
	if got := output.String(); !strings.Contains(got, wantLower) {
		t.Fatalf("core lower-level trace = %q, missing %q", got, wantLower)
	}
}
