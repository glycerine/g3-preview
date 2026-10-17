package sat

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

// Source: src/sat/sat_solver.cpp:1695. The level-zero trace precedes the
// early false return, including the source null-literal spelling.
func TestSATConflictLevelZeroSourceTraceBeforeReturn(t *testing.T) {
	s := satConflictFixture(t, 1)
	s.analysis.propagation.setConflict(satJustification{}, DefaultLiteral())
	var out bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&out)
	tags.Enable("sat")
	s.traceTags = tags
	ok, err := s.resolveConflict()
	satOK(t, err)
	if ok {
		t.Fatal("level-zero conflict resolved")
	}
	if !tags.ShouldTrace("sat") {
		if out.Len() != 0 {
			t.Fatalf("erased trace emitted %q", out.String())
		}
		return
	}
	want := satClauseTraceRecord("sat", "resolve_conflict_core", "1695", "conflict detected at level 0 for null literal\n")
	if out.String() != want {
		t.Fatalf("level-zero trace got %q, want %q", out.String(), want)
	}
}

// Source: src/sat/sat_solver.cpp:1718,1726,1788,1795. The live first-UIP
// path reports marks before each consequent and the lemma after UIP choice.
func TestSATConflictFirstUIPSourceTraceOrder(t *testing.T) {
	s := satConflictFixture(t, 4)
	satConflictDAG(t, s)
	var out bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&out)
	for _, name := range []string{"sat_conflict", "sat_conflict_detail", "sat_lemma"} {
		tags.Enable(name)
	}
	s.traceTags = tags
	ok, err := s.resolveConflict()
	satOK(t, err)
	if !ok {
		t.Fatal("first-UIP conflict unresolved")
	}
	if !tags.ShouldTrace("sat_lemma") {
		if out.Len() != 0 {
			t.Fatalf("erased trace emitted %q", out.String())
		}
		return
	}
	want := satClauseTraceRecord("sat_conflict", "resolve_conflict_core", "1718", "not_l: 3\n") +
		satClauseTraceRecord("sat_conflict_detail", "resolve_conflict_core", "1726", "processing consequent: 3\nnum_marks: 1, js: binary -1\n") +
		satClauseTraceRecord("sat_conflict_detail", "resolve_conflict_core", "1726", "processing consequent: 3\nnum_marks: 1, js: binary -2\n") +
		satClauseTraceRecord("sat_conflict_detail", "resolve_conflict_core", "1726", "processing consequent: 2\nnum_marks: 1, js: ternary -0 -1\n") +
		satClauseTraceRecord("sat_lemma", "resolve_conflict_core", "1788", "new lemma size: 2\n-1 -0\n")
	cfg, err := s.analysis.propagation.config.State()
	satOK(t, err)
	if cfg.MinimizeLemmas {
		want += satClauseTraceRecord("sat_lemma", "resolve_conflict_core", "1795", "new lemma (after minimization) size: 2\n-1 -0\n")
	}
	if out.String() != want {
		t.Fatalf("first-UIP trace got %q, want %q", out.String(), want)
	}
}

// Source: src/sat/sat_solver.cpp:1812. The solver display observes the
// backtracked level-one assignment before the learned clause is published.
func TestSATConflictFullSolverPostBackjumpDisplayTrace(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	for i := 0; i < 4; i++ {
		_, err := s.MkVar(true, true)
		satOK(t, err)
	}
	p := s.propagation
	satOK(t, s.Lifecycle.push())
	satOK(t, p.assign(0, satJustification{}))
	_, err := s.Lifecycle.propagate(false)
	satOK(t, err)
	satOK(t, s.Lifecycle.push())
	satOK(t, p.assign(2, satJustification{}))
	satOK(t, p.assign(4, newSATTernaryJustification(1, 3)))
	satOK(t, p.assign(6, newSATBinaryJustification(5)))
	p.setConflict(newSATBinaryJustification(3), 6)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("sat_conflict_detail")
	s.conflict.traceTags = tags
	ok, err := s.conflict.resolveConflict()
	satOK(t, err)
	if !ok {
		t.Fatal("first-UIP conflict unresolved")
	}
	want := ""
	if tags.ShouldTrace("sat_conflict_detail") {
		want = satClauseTraceRecord("sat_conflict_detail", "resolve_conflict_core", "1726", "processing consequent: 3\nnum_marks: 1, js: binary -1\n") +
			satClauseTraceRecord("sat_conflict_detail", "resolve_conflict_core", "1726", "processing consequent: 3\nnum_marks: 1, js: binary -2\n") +
			satClauseTraceRecord("sat_conflict_detail", "resolve_conflict_core", "1726", "processing consequent: 2\nnum_marks: 1, js: ternary -0 -1\n") +
			satClauseTraceRecord("sat_conflict_detail", "resolve_conflict_core", "1812", "(sat\n)\nassignment:\n0\n")
	}
	if got := output.String(); got != want {
		t.Fatalf("post-backjump source trace = %q, want %q", got, want)
	}
}
