package sat

import (
	"bytes"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/lbool"
)

type satPropagationTraceObserver struct {
	output                                    bytes.Buffer
	state                                     *satSolverPropagationState
	assignmentSeen, assignmentBeforeWrite     bool
	propagationSeen, propagationBeforeAdvance bool
}

func (w *satPropagationTraceObserver) Write(p []byte) (int, error) {
	text := string(p)
	if strings.HasPrefix(text, "0 binary 1 level: 0") {
		w.assignmentSeen = true
		v, err := w.state.variables.valueLiteral(0)
		w.assignmentBeforeWrite = err == nil && v == lbool.Undef
	}
	if strings.HasPrefix(text, "propagating: 0 none") {
		w.propagationSeen = true
		w.propagationBeforeAdvance = w.state.variables.qhead == 0
	}
	return w.output.Write(p)
}

// Source: src/sat/sat_solver.cpp:506,564. Assignment prints its incoming
// justification before level-zero erasure; propagation prints the stored one.
func TestSATAssignmentAndPropagationSourceTraceOrder(t *testing.T) {
	p, v, _ := satPropagationFixture(t, 3)
	w := &satPropagationTraceObserver{state: p}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("sat_assign_core")
	tags.Enable("sat_propagate")
	p.traceTags = tags
	satOK(t, p.assignCore(0, newSATBinaryJustification(2)))
	_, err := p.propagateCore(false)
	satOK(t, err)
	j, err := v.justification.read(0)
	satOK(t, err)
	if !j.None() || v.qhead != 1 {
		t.Fatal("source level-zero assignment and propagation")
	}
	v.scopeLevel = 2
	scoped := newSATTernaryJustification(0, 5)
	satOK(t, p.assignCore(2, scoped))
	_, err = p.propagateCore(false)
	satOK(t, err)
	j, err = v.justification.read(1)
	satOK(t, err)
	if j != scoped || v.qhead != 2 {
		t.Fatal("scoped justification was not retained")
	}
	if !tags.ShouldTrace("sat_assign_core") {
		if w.output.Len() != 0 || w.assignmentSeen || w.propagationSeen {
			t.Fatal("erased propagation TRACE emitted")
		}
		return
	}
	want := satClauseTraceRecord("sat_assign_core", "assign_core", "506", "0 binary 1 level: 0\n") +
		satClauseTraceRecord("sat_propagate", "propagate_core", "564", "propagating: 0 none\n") +
		satClauseTraceRecord("sat_assign_core", "assign_core", "506", "1 ternary 0 -2 level: 2\n") +
		satClauseTraceRecord("sat_propagate", "propagate_core", "564", "propagating: 1 ternary 0 -2\n")
	if got := w.output.String(); got != want || !w.assignmentSeen || !w.assignmentBeforeWrite || !w.propagationSeen || !w.propagationBeforeAdvance {
		t.Fatalf("propagation trace got %q, want %q; assign-before=%v propagation-before=%v", got, want, w.assignmentBeforeWrite, w.propagationBeforeAdvance)
	}
}
