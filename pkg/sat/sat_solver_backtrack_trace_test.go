package sat

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

type satBacktrackTraceObserver struct {
	output                        bytes.Buffer
	lifecycle                     *satSolverClauseLifecycle
	conflict                      *satSolverConflictResolution
	pushSeen, pushBeforeScope     bool
	forgetSeen, forgetBeforeClear bool
}

func (w *satBacktrackTraceObserver) Write(p []byte) (int, error) {
	text := string(p)
	if strings.HasPrefix(text, "q:") {
		w.pushSeen = true
		v := w.lifecycle.propagation.variables
		w.pushBeforeScope = v.scopeLevel == 0 && len(w.lifecycle.scopes) == 0
	}
	if strings.HasPrefix(text, "forgeting phase of l:") {
		w.forgetSeen = true
		v := w.conflict.analysis.propagation.variables
		phase, err := v.phaseAt(1)
		w.forgetBeforeClear = err == nil && phase == 0 && v.scopeLevel == 2
	}
	return w.output.Write(p)
}

// Source: src/sat/sat_solver.cpp:2533; the queue/trail record is emitted
// before the scope is published.
func TestSATPushScopeSourceTraceBeforePublication(t *testing.T) {
	s, _, v, _ := satLifecycleFixture(t, 1)
	w := &satBacktrackTraceObserver{lifecycle: s}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("sat_verbose")
	s.traceTags = tags
	satOK(t, s.push())
	if v.scopeLevel != 1 || len(s.scopes) != 1 {
		t.Fatal("source scope publication")
	}
	if !tags.ShouldTrace("sat_verbose") {
		if w.output.Len() != 0 || w.pushSeen {
			t.Fatal("erased push TRACE emitted")
		}
		return
	}
	want := satClauseTraceRecord("sat_verbose", "push", "2533", "q:0 trail: 0\n")
	if got := w.output.String(); got != want || !w.pushSeen || !w.pushBeforeScope {
		t.Fatalf("push trace got %q, want %q; before=%v", got, want, w.pushBeforeScope)
	}
}

// Source: src/sat/sat_solver.cpp:2193; only literals at and after the
// selected scope's trail boundary emit, before their phase is cleared.
func TestSATForgetPhaseSourceTraceBeforeClear(t *testing.T) {
	s := satConflictFixture(t, 3)
	satConflictDAGSmall(t, s)
	w := &satBacktrackTraceObserver{conflict: s}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("forget_phase")
	s.traceTags = tags
	satOK(t, s.forgetPhaseOfVars(2))
	phase, err := s.analysis.propagation.variables.phaseAt(1)
	satOK(t, err)
	if phase != 2 {
		t.Fatal("source phase clear", phase)
	}
	if !tags.ShouldTrace("forget_phase") {
		if w.output.Len() != 0 || w.forgetSeen {
			t.Fatal("erased forget TRACE emitted")
		}
		return
	}
	want := satClauseTraceRecord("forget_phase", "forget_phase_of_vars", "2193", "forgeting phase of l: 1\n")
	if got := w.output.String(); got != want || !w.forgetSeen || !w.forgetBeforeClear {
		t.Fatalf("forget trace got %q, want %q; before=%v", got, want, w.forgetBeforeClear)
	}
}
