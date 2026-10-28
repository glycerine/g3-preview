package sat

// Source: src/sat/sat_elim_eqs.cpp:81,104,138,187,216.

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func satElimEqsTraceRecord(tag, function, line, body string) string {
	return "-------- [" + tag + "] " + function + " src/sat/sat_elim_eqs.cpp:" + line + " ---------\n" +
		body + "------------------------------------------------\n"
}

func TestSATElimEqsClauseAndModelSourceTraces(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		l, p, _, _ := satLifecycleFixture(t, 3)
		e, err := newSATElimEqs(l)
		satOK(t, err)
		t.Cleanup(func() { satOK(t, e.Close()) })
		c, err := p.clauses.MkClause([]Literal{0, 2, 4}, false)
		satOK(t, err)
		satOK(t, l.attachClause(c))
		satOK(t, l.pushClause(false, c))
		var output bytes.Buffer
		tags := diagnostic.NewTraceTagsWithWriter(&output)
		if enabled {
			tags.Enable("elim_eqs")
			tags.Enable("save_elim")
		}
		l.traceTags = tags
		satOK(t, e.run([]Literal{0, 0, 4}, []BoolVar{1}))
		want := ""
		if tags.ShouldTrace("elim_eqs") {
			want += satElimEqsTraceRecord("elim_eqs", "cleanup_clauses", "81", "processing: (0 1 2)+\n")
			want += satElimEqsTraceRecord("elim_eqs", "cleanup_clauses", "104", "after normalization/sorting: (0 0 2)+\n")
			want += satElimEqsTraceRecord("elim_eqs", "cleanup_clauses", "138", "after removing duplicates: (0 2 2)+ j: 2\n")
		}
		if tags.ShouldTrace("save_elim") {
			want += satElimEqsTraceRecord("save_elim", "save_elim", "187", "marking as deleted: 1 l: 1 r: 0\n")
		}
		if got := output.String(); got != want {
			t.Fatalf("enabled=%v: got %q, want %q", enabled, got, want)
		}
		if p.converter.Size() != 1 || p.clauses.IsLive(c) {
			t.Fatalf("source elimination result: converter=%d live=%v", p.converter.Size(), p.clauses.IsLive(c))
		}
	}
}

func TestSATElimEqsFullSolverBinCleanupDisplaySourceTrace(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	_, err := s.MkVar(true, true)
	satOK(t, err)
	observer := &satSimplifierRunDisplayObserver{solver: s}
	tags := diagnostic.NewTraceTagsWithWriter(observer)
	tags.Enable("elim_eqs")
	s.Lifecycle.traceTags = tags
	_, err = s.scc.run()
	satOK(t, err)
	if !tags.ShouldTrace("elim_eqs") {
		if observer.output.Len() != 0 {
			t.Fatal("build-erased equivalence display", observer.output.String())
		}
		return
	}
	wantPrefix := satElimEqsTraceRecord("elim_eqs", "operator()", "216", "after bin cleanup\n")
	got := observer.output.String()
	if len(observer.match) != 1 || !observer.match[0] ||
		!strings.HasPrefix(got, strings.TrimSuffix(wantPrefix, "------------------------------------------------\n")) ||
		strings.Count(got, "src/sat/sat_elim_eqs.cpp:216") != 1 {
		t.Fatalf("source display at bin-cleanup boundary: %v %q", observer.match, got)
	}
}
