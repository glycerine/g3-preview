package sat

import (
	"bytes"
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/lbool"
)

// Source: src/sat/sat_solver.cpp:419. CTRACE is silent for false literals;
// a nonfalse literal is printed before the following source assertion.
func TestSATLearnedWatchConditionalSourceTraceBeforeAssertion(t *testing.T) {
	a := satStore(t, nil)
	clause := satMk(t, a, satLs(0, 2, 4), true)
	var out bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&out)
	tags.Enable("sat")
	state := satSolverClauseState{
		assignment: satClauseAssignmentBytes([]lbool.LBool{lbool.Undef, lbool.Undef, lbool.False, lbool.True, lbool.False, lbool.True}),
		levels:     []uint32{0, 2, 1},
		traceTags:  tags,
	}
	index, err := state.selectLearnedWatchLiteral(a, clause)
	if err != nil || index != 1 || out.Len() != 0 {
		t.Fatalf("false literals must not emit: index=%d err=%v output=%q", index, err, out.String())
	}
	state.assignment[2] = int8(lbool.Undef)
	index, err = state.selectLearnedWatchLiteral(a, clause)
	if satStorageSourceDebug {
		if !errors.Is(err, ErrSATAssertion) {
			t.Fatalf("assertion after CTRACE: index=%d err=%v", index, err)
		}
	} else if err != nil || index != 1 {
		t.Fatalf("release watch selection: index=%d err=%v", index, err)
	}
	if !tags.ShouldTrace("sat") {
		if out.Len() != 0 {
			t.Fatal("erased CTRACE emitted", out.String())
		}
		return
	}
	want := satClauseTraceRecord("sat", "select_learned_watch_lit", "419", "1:=l_undef")
	if got := out.String(); got != want {
		t.Fatalf("nonfalse CTRACE got %q, want %q", got, want)
	}
}
