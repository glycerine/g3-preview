package sat

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

type satClauseTraceObserver struct {
	output                       bytes.Buffer
	lifecycle                    *satSolverClauseLifecycle
	clause                       StoredClauseRef
	reinitSeen, reinitBeforePush bool
}

func (w *satClauseTraceObserver) Write(p []byte) (int, error) {
	if strings.HasPrefix(string(p), "adding to reinit stack: ") {
		w.reinitSeen = true
		onStack, err := w.lifecycle.propagation.clauses.OnReinitStack(w.clause)
		w.reinitBeforePush = err == nil && !onStack && len(w.lifecycle.reinit) == 0
	}
	return w.output.Write(p)
}

func satClauseTraceRecord(tag, function, line, body string) string {
	return "-------- [" + tag + "] " + function + " src/sat/sat_solver.cpp:" + line + " ---------\n" +
		body + "------------------------------------------------\n"
}

// Source: src/sat/sat_solver.cpp:196,199. The first record sees the borrowed
// input order; the second sees the source simplifier's sorted, unique prefix.
func TestSATClauseCreationSourceTraceBeforeAndAfterSimplification(t *testing.T) {
	s, _, _, _ := satLifecycleFixture(t, 4)
	w := &satClauseTraceObserver{}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("sat")
	tags.Enable("sat_mk_clause")
	s.traceTags = tags
	lits := []Literal{4, 2, 2, 0}
	_, err := s.MkClauseCore(uint32(len(lits)), lits, false)
	satOK(t, err)
	if !tags.ShouldTrace("sat") {
		if w.output.Len() != 0 {
			t.Fatal("erased creation TRACE emitted")
		}
		return
	}
	want := satClauseTraceRecord("sat", "mk_clause_core", "196", "mk_clause: 2 1 1 0 aux\n") +
		satClauseTraceRecord("sat_mk_clause", "mk_clause_core", "199", "mk_clause (after simp), keep: 1\n0 1 2\n")
	if got := w.output.String(); got != want {
		t.Fatalf("clause creation trace got %q, want %q", got, want)
	}
	w.output.Reset()
	_, err = s.MkClauseCore(3, []Literal{4, 2, 0}, true)
	satOK(t, err)
	want = satClauseTraceRecord("sat", "mk_clause_core", "196", "mk_clause: 2 1 0 learned\n")
	if got := w.output.String(); got != want {
		t.Fatalf("learned clause trace got %q, want %q", got, want)
	}
}

// Source: src/sat/sat_solver.cpp:250; the record precedes wrapper publication
// and the clause's on-reinit-stack flag.
func TestSATReinitClauseSourceTraceBeforePublication(t *testing.T) {
	s, _, _, _ := satLifecycleFixture(t, 4)
	r, err := s.MkClauseCore(3, []Literal{0, 2, 4}, false)
	satOK(t, err)
	w := &satClauseTraceObserver{lifecycle: s, clause: r}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("sat_reinit")
	s.traceTags = tags
	satOK(t, s.pushReinitStack(r))
	onStack, err := s.propagation.clauses.OnReinitStack(r)
	satOK(t, err)
	if !onStack || len(s.reinit) != 1 {
		t.Fatal("reinit stack publication")
	}
	if !tags.ShouldTrace("sat_reinit") {
		if w.output.Len() != 0 || w.reinitSeen {
			t.Fatal("erased reinit TRACE emitted")
		}
		return
	}
	want := satClauseTraceRecord("sat_reinit", "push_reinit_stack", "250", "adding to reinit stack: (0 1 2)+\n")
	if got := w.output.String(); got != want || !w.reinitSeen || !w.reinitBeforePush {
		t.Fatalf("reinit trace got %q, want %q; before=%v", got, want, w.reinitBeforePush)
	}
}
