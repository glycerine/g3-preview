package sat

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/lbool"
)

type satUserScopeTraceObserver struct {
	output                         bytes.Buffer
	solver                         *Solver
	pushSeen, pushAfterPublication bool
	trailSeen, trailBeforeUnassign bool
}

func (w *satUserScopeTraceObserver) Write(p []byte) (int, error) {
	switch string(p) {
	case "user_push: 0\n":
		w.pushSeen = true
		n, err := w.solver.conflict.userScopeLiterals.Size()
		w.pushAfterPublication = err == nil && n == 1
	case "0\n":
		w.trailSeen = true
		value, err := w.solver.Variables.valueLiteral(0)
		w.trailBeforeUnassign = err == nil && value == lbool.True
	}
	return w.output.Write(p)
}

// Source: src/sat/sat_solver.cpp:2625,2741. User push traces after literal
// publication; the matched user-pop trail is printed before unassignment.
func TestSATUserScopeSourceTracePublicationAndTrail(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	w := &satUserScopeTraceObserver{solver: s}
	tags := diagnostic.NewTraceTagsWithWriter(w)
	tags.Enable("sat")
	s.traceTags = tags
	satOK(t, s.UserPush())
	satOK(t, s.propagation.assign(0, satJustification{}))
	satOK(t, s.UserPop(1))
	if !tags.ShouldTrace("sat") {
		if w.output.Len() != 0 || w.pushSeen || w.trailSeen {
			t.Fatal("erased user-scope TRACE emitted")
		}
		return
	}
	want := satClauseTraceRecord("sat", "user_push", "2625", "user_push: 0\n") +
		satClauseTraceRecord("sat", "user_pop", "2727", "(sat\n0 \n)\n") +
		satClauseTraceRecord("sat", "user_pop", "2737", "gc: 0\n(sat\n0 \n)\n") +
		satClauseTraceRecord("sat", "user_pop", "2741", "0\n")
	if got := w.output.String(); got != want || !w.pushSeen || !w.pushAfterPublication || !w.trailSeen || !w.trailBeforeUnassign {
		t.Fatalf("user-scope trace got %q, want %q; push=%v trail=%v", got, want, w.pushAfterPublication, w.trailBeforeUnassign)
	}
}

// Source: src/sat/sat_solver.cpp:2727,2737. The first display includes the
// user-scoped original clause; the second follows both clause GC passes.
func TestSATUserPopSourceTraceBeforeAndAfterClauseCollection(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	satOK(t, s.UserPush())
	for i := 0; i < 3; i++ {
		_, err := s.MkVar(false, true)
		satOK(t, err)
	}
	satOK(t, s.MkClause([]Literal{2, 4, 6}))
	if len(s.Lifecycle.Original) != 1 {
		t.Fatal("missing scoped source clause")
	}
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	tags.Enable("sat")
	s.traceTags = tags
	satOK(t, s.UserPop(1))
	if len(s.Lifecycle.Original) != 0 {
		t.Fatal("scoped clause survived GC")
	}
	if !tags.ShouldTrace("sat") {
		if output.Len() != 0 {
			t.Fatal("erased user-pop displays emitted")
		}
		return
	}
	want := satClauseTraceRecord("sat", "user_pop", "2727", "(sat\n(0 1 2 3)+\n)\n") +
		satClauseTraceRecord("sat", "user_pop", "2737", "gc: 0\n(sat\n)\n")
	if got := output.String(); got != want {
		t.Fatalf("user-pop snapshots got %q, want %q", got, want)
	}
}
