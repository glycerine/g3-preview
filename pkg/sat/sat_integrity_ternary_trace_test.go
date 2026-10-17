package sat

import (
	"bytes"
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestSATIntegrityMissingTernaryWatchSourceTrace(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	for i := 0; i < 3; i++ {
		_, err := s.MkVar(false, true)
		satOK(t, err)
	}
	cl := satCleanerClause(t, s.Lifecycle, []Literal{0, 2, 4}, true, false)
	c, err := newSATIntegrityChecker(s.Lifecycle, s.gc)
	satOK(t, err)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	s.Lifecycle.traceTags = tags
	_, err = c.checkClause(cl)
	if !errors.Is(err, ErrSATAssertion) || output.Len() != 0 {
		t.Fatalf("disabled trace: error=%v, output=%q", err, output.String())
	}
	tags.Enable("sat_ter_watch_bug")
	_, err = c.checkClause(cl)
	if !errors.Is(err, ErrSATAssertion) {
		t.Fatalf("missing first ternary watch: %v", err)
	}
	want := ""
	if tags.ShouldTrace("sat_ter_watch_bug") {
		want = "-------- [sat_ter_watch_bug] check_clause src/sat/sat_integrity_checker.cpp:69 ---------\n" +
			"(0 1 2)+*\nwatch_list:\n\n" +
			"------------------------------------------------\n"
	}
	if got := output.String(); got != want {
		t.Fatalf("missing ternary source trace: got %q, want %q", got, want)
	}
	list, err := s.Variables.watches.Get(1)
	satOK(t, err)
	other, err := newSATExternalWatch(3)
	satOK(t, err)
	satOK(t, list.Push(other))
	output.Reset()
	_, err = c.checkClause(cl)
	if !errors.Is(err, ErrSATAssertion) {
		t.Fatalf("missing ternary in nonempty watch list: %v", err)
	}
	if tags.ShouldTrace("sat_ter_watch_bug") {
		want = "-------- [sat_ter_watch_bug] check_clause src/sat/sat_integrity_checker.cpp:69 ---------\n" +
			"(0 1 2)+*\nwatch_list:\n3\n" +
			"------------------------------------------------\n"
	}
	if got := output.String(); got != want {
		t.Fatalf("nonempty watch-list source trace: got %q, want %q", got, want)
	}
	for ordinal, body := range []string{"", "(0 1 2)+*\nwatch_list:\n"} {
		output.Reset()
		s.Lifecycle.budget.FailAfter(int64(ordinal))
		_, err = c.checkClause(cl)
		if !errors.Is(err, ErrSATAssertion) {
			t.Fatalf("missing ternary after formatter refusal %d: %v", ordinal, err)
		}
		wantPrefix := ""
		if tags.ShouldTrace("sat_ter_watch_bug") {
			wantPrefix = "-------- [sat_ter_watch_bug] check_clause src/sat/sat_integrity_checker.cpp:69 ---------\n" + body
		}
		if got := output.String(); got != wantPrefix {
			t.Fatalf("formatter refusal %d: got %q, want %q", ordinal, got, wantPrefix)
		}
	}
	s.Lifecycle.budget.FailAfter(-1)
	satOK(t, list.SetEnd(0))
	satOK(t, s.Lifecycle.attachClause(cl))
	output.Reset()
	yes, err := c.checkClause(cl)
	satOK(t, err)
	if !yes || output.Len() != 0 {
		t.Fatalf("attached ternary clause: result=%v, trace=%q", yes, output.String())
	}
}
