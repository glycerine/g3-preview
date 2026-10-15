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
	for i := 0; i < 4; i++ {
		_, e := s.MkVar(false, true)
		satOK(t, e)
	}
	p := s.propagation
	r, e := p.clauses.MkClause([]Literal{1, 3, 5}, true)
	if e != nil {
		t.Fatalf("frozen clause construction: %v", e)
	}
	if e = s.Lifecycle.pushClause(true, r); e != nil {
		t.Fatalf("frozen clause publication: %v", e)
	}
	if e = p.clauses.Freeze(r); e != nil {
		t.Fatalf("frozen clause freeze: %v", e)
	}
	satOK(t, p.assign(0, satJustification{}))
	satOK(t, p.assign(2, satJustification{}))
	satOK(t, p.assign(4, satJustification{}))
	_, e = s.Lifecycle.propagate(false, false, false)
	satOK(t, e)
	satOK(t, s.Lifecycle.push())
	satOK(t, p.assign(6, satJustification{}))
	p.setConflict(newSATBinaryJustification(1), 6)
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
