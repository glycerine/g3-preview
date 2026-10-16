package sat

import (
	"bytes"
	"errors"
	"os"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/z3exception"
)

func TestSATIntegrityEliminatedWatchSourceTrace(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	_, err := s.MkVar(false, true)
	satOK(t, err)
	list, err := s.Variables.watches.Get(0)
	satOK(t, err)
	w, err := newSATExternalWatch(3)
	satOK(t, err)
	satOK(t, list.Push(w))
	s.Variables.eliminated[0] = true
	defer func() {
		s.Variables.eliminated[0] = false
		satOK(t, list.SetEnd(0))
	}()
	c, err := newSATIntegrityChecker(s.Lifecycle, s.gc)
	satOK(t, err)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	s.Lifecycle.traceTags = tags
	_, err = c.checkWatches()
	if satStorageSourceDebug {
		if !errors.Is(err, ErrSATAssertion) {
			t.Fatalf("eliminated watch assertion: %v", err)
		}
	} else {
		satOK(t, err)
	}
	if output.Len() != 0 {
		t.Fatalf("disabled sat_bug trace emitted %q", output.String())
	}
	tags.Enable("sat_bug")
	_, err = c.checkWatches()
	if satStorageSourceDebug {
		if !errors.Is(err, ErrSATAssertion) {
			t.Fatalf("enabled eliminated watch assertion: %v", err)
		}
	} else {
		satOK(t, err)
	}
	want := ""
	if tags.ShouldTrace("sat_bug") && satStorageSourceDebug {
		want = "-------- [sat_bug] check_watches src/sat/sat_integrity_checker.cpp:162 ---------\n" +
			"l: -0\n0: 3\n-0: \n(sat\n)\n" +
			"------------------------------------------------\n"
	}
	if got := output.String(); got != want {
		t.Fatalf("eliminated-watch source trace: got %q, want %q", got, want)
	}
	debug, input, assertionFile := satIntegrityActionFixture(t, "T")
	satOK(t, debug.EnableAssertions(false))
	output.Reset()
	yes, err := c.checkWatches()
	satOK(t, err)
	if !yes || output.String() != want {
		t.Fatalf("trace with assertions disabled: result=%v, got %q, want %q", yes, output.String(), want)
	}
	for ordinal, body := range []string{
		"l: -0\n",
		"l: -0\n0: 3\n-0: \n",
		"l: -0\n0: 3\n-0: \n(sat\n)\n------------------------------------------------\n",
	} {
		output.Reset()
		s.Lifecycle.budget.FailAfter(int64(ordinal))
		yes, err = c.checkWatches()
		satOK(t, err)
		want = ""
		if tags.ShouldTrace("sat_bug") && satStorageSourceDebug {
			want = "-------- [sat_bug] check_watches src/sat/sat_integrity_checker.cpp:162 ---------\n" + body
		}
		if !yes || output.String() != want {
			t.Fatalf("eliminated-watch formatter reservation %d: result=%v, got %q, want %q", ordinal, yes, output.String(), want)
		}
	}
	s.Lifecycle.budget.FailAfter(-1)
	satOK(t, debug.EnableAssertions(true))
	satOK(t, debug.EnableDebug("cleaner_bug"))
	output.Reset()
	var ex z3exception.DefaultException
	if err = s.cleaner.assertInvariant(); satStorageSourceDebug && !errors.As(err, &ex) {
		t.Fatalf("real cleaner nested assertion action: %v", err)
	} else if !satStorageSourceDebug {
		satOK(t, err)
	}
	if got := output.String(); got != want {
		t.Fatalf("real cleaner trace: got %q, want %q", got, want)
	}
	if satStorageSourceDebug {
		state, e := input.State()
		satOK(t, e)
		if state.Position != 1 {
			t.Fatalf("real cleaner consumed %d debugger actions", state.Position)
		}
		contents, e := os.ReadFile(assertionFile.Name())
		satOK(t, e)
		wantAssertion := "ASSERTION VIOLATION\nFile: src/sat/sat_integrity_checker.cpp\nLine: 167\n!s.was_eliminated(l.var()) || wlist.empty()\n" +
			"(C)ontinue, (A)bort, (S)top, (T)hrow exception, Invoke (G)DB\n"
		if string(contents) != wantAssertion {
			t.Fatalf("real cleaner assertion: got %q, want %q", contents, wantAssertion)
		}
	}
}

func TestSATIntegrityEliminatedWatchTraceWithLiveSolverClause(t *testing.T) {
	s, _, _ := satSolverCheckFixture(t)
	for i := 0; i < 3; i++ {
		_, err := s.MkVar(false, true)
		satOK(t, err)
	}
	satCleanerClause(t, s.Lifecycle, []Literal{0, 2, 4}, true, true)
	list, err := s.Variables.watches.Get(0)
	satOK(t, err)
	watch, err := newSATExternalWatch(3)
	satOK(t, err)
	satOK(t, list.Push(watch))
	s.Variables.eliminated[0] = true
	defer func() {
		s.Variables.eliminated[0] = false
		satOK(t, list.SetEnd(0))
	}()
	debug, _, _ := satIntegrityActionFixture(t, "")
	satOK(t, debug.EnableAssertions(false))
	checker, err := newSATIntegrityChecker(s.Lifecycle, s.gc)
	satOK(t, err)
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	s.Lifecycle.traceTags = tags
	tags.Enable("sat_bug")
	yes, err := checker.checkWatches()
	satOK(t, err)
	want := ""
	if tags.ShouldTrace("sat_bug") && satStorageSourceDebug {
		want = "-------- [sat_bug] check_watches src/sat/sat_integrity_checker.cpp:162 ---------\n" +
			"l: -0\n0: 3\n-0: \n1: \n-1: \n2: \n-2: \n(sat\n(0 1 2)+*\n)\n" +
			"------------------------------------------------\n"
	}
	if !yes || output.String() != want {
		t.Fatalf("nonempty solver display in eliminated-watch trace: result=%v, got %q, want %q", yes, output.String(), want)
	}
}
