package sat

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/lbool"
)

// Source: src/sat/sat_solver.cpp:1014-1031. The learned conflict returns to
// level zero; cleanup then discovers the already-false frozen clause.
func TestSATBackjumpCleanupConflictSourceTrace(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	for i := 0; i < 3; i++ {
		_, e := s.MkVar(false, true)
		satOK(t, e)
	}
	p := s.propagation
	satOK(t, p.assign(0, satJustification{}))
	satOK(t, p.assign(2, satJustification{}))
	satCleanerClause(t, s.Lifecycle, []Literal{1, 3}, false, true)
	satOK(t, s.Lifecycle.push())
	satOK(t, p.assign(4, satJustification{}))
	p.setConflict(newSATBinaryJustification(1), 4)
	s.gc.restartThreshold = 100
	var out bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&out)
	tags.Enable("sat")
	s.traceTags = tags
	got, done, e := s.backjumpStep()
	satOK(t, e)
	if got != lbool.False || !done || !p.inconsistent {
		t.Fatalf("backjump result %v done=%v inconsistent=%v", got, done, p.inconsistent)
	}
	if !tags.ShouldTrace("sat") {
		if out.Len() != 0 {
			t.Fatalf("erased trace emitted %q", out.String())
		}
		return
	}
	want := "-------- [sat] propagate_and_backjump_step src/sat/sat_solver.cpp:1027 ---------\n" +
		"conflict at level 0\n------------------------------------------------\n"
	if got := out.String(); got != want {
		t.Fatalf("source trace got %q, want %q", got, want)
	}
}
