package sat

import (
	"bytes"
	"errors"
	"os"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/z3exception"
)

func TestSATIntegrityEliminatedClauseSourceTrace(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	for i := 0; i < 3; i++ {
		_, err := s.MkVar(false, true)
		satOK(t, err)
	}
	cl := satCleanerClause(t, s.Lifecycle, []Literal{0, 2, 4}, true, true)
	s.Variables.eliminated[0] = true
	defer func() { s.Variables.eliminated[0] = false }()
	c, err := newSATIntegrityChecker(s.Lifecycle, s.gc)
	satOK(t, err)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	s.Lifecycle.traceTags = tags
	_, err = c.checkClause(cl)
	if satStorageSourceDebug {
		if !errors.Is(err, ErrSATAssertion) {
			t.Fatalf("eliminated clause assertion: %v", err)
		}
	} else {
		satOK(t, err)
	}
	if output.Len() != 0 {
		t.Fatalf("disabled trace emitted %q", output.String())
	}
	tags.Enable("sat_bug")
	_, err = c.checkClause(cl)
	if satStorageSourceDebug {
		if !errors.Is(err, ErrSATAssertion) {
			t.Fatalf("traced clause assertion: %v", err)
		}
	} else {
		satOK(t, err)
	}
	want := ""
	if tags.ShouldTrace("sat_bug") {
		want = "-------- [sat_bug] check_clause src/sat/sat_integrity_checker.cpp:56 ---------\n" +
			"l: 0\nc: (0 1 2)+*\n(sat\n(0 1 2)+*\n)\n" +
			"------------------------------------------------\n"
	}
	if got := output.String(); got != want {
		t.Fatalf("eliminated-clause source trace: got %q, want %q", got, want)
	}
	for ordinal, body := range []string{
		"l: 0\nc: ",
		"l: 0\nc: (0 1 2)+*\n",
	} {
		output.Reset()
		s.Lifecycle.budget.FailAfter(int64(ordinal))
		_, err = c.checkClause(cl)
		if satStorageSourceDebug {
			if !errors.Is(err, ErrSATAssertion) {
				t.Fatalf("eliminated clause assertion after formatter refusal: %v", err)
			}
		} else {
			satOK(t, err)
		}
		wantPrefix := ""
		if tags.ShouldTrace("sat_bug") {
			wantPrefix = "-------- [sat_bug] check_clause src/sat/sat_integrity_checker.cpp:56 ---------\n" + body
		}
		if got := output.String(); got != wantPrefix {
			t.Fatalf("formatter refusal %d: got %q, want %q", ordinal, got, wantPrefix)
		}
	}
	s.Lifecycle.budget.FailAfter(-1)
	debug, input, assertionFile := satIntegrityActionFixture(t, "T")
	satOK(t, debug.EnableAssertions(false))
	output.Reset()
	yes, err := c.checkClause(cl)
	satOK(t, err)
	if !yes || output.String() != want {
		t.Fatalf("disabled assertions: result=%v, trace=%q, want %q", yes, output.String(), want)
	}
	satOK(t, debug.EnableAssertions(true))
	satOK(t, debug.EnableDebug("cleaner_bug"))
	output.Reset()
	err = s.cleaner.assertInvariant()
	if satStorageSourceDebug {
		var ex z3exception.DefaultException
		if !errors.As(err, &ex) {
			t.Fatalf("real cleaner clause assertion: %v", err)
		}
		if got := output.String(); got != want {
			t.Fatalf("real cleaner trace: got %q, want %q", got, want)
		}
		state, e := input.State()
		satOK(t, e)
		if state.Position != 1 {
			t.Fatalf("real cleaner consumed %d actions", state.Position)
		}
		contents, e := os.ReadFile(assertionFile.Name())
		satOK(t, e)
		wantAssertion := "ASSERTION VIOLATION\nFile: src/sat/sat_integrity_checker.cpp\nLine: 60\n!s.was_eliminated(c[i].var())\n" +
			"(C)ontinue, (A)bort, (S)top, (T)hrow exception, Invoke (G)DB\n"
		if string(contents) != wantAssertion {
			t.Fatalf("real cleaner assertion: got %q, want %q", contents, wantAssertion)
		}
	} else {
		satOK(t, err)
		if output.Len() != 0 {
			t.Fatalf("release cleaner reached checked trace: %q", output.String())
		}
	}
}
