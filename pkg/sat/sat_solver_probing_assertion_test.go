package sat

import (
	"errors"
	"os"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/z3exception"
)

func TestSATProbingReadinessAssertionsUseSeparateSourceActions(t *testing.T) {
	s, life := satProbingFixture(t, 1)
	satOK(t, life.propagation.assignCore(0, satJustification{}))
	debug, input, output := satIntegrityActionFixture(t, "CTCTT")
	satOK(t, debug.EnableAssertions(false))
	if err := s.assertReady(0, 61, "s.value(l.var()) == l_undef"); err != nil {
		t.Fatal("disabled readiness evaluated an assertion", err)
	}
	satOK(t, debug.EnableAssertions(true))
	if !satStorageSourceDebug {
		satOK(t, s.assertReady(0, 61, "s.value(l.var()) == l_undef"))
		return
	}
	for _, row := range []struct {
		line int32
		expr string
	}{
		{61, "s.value(l.var()) == l_undef"},
		{112, "s.value(v) == l_undef"},
	} {
		var ex z3exception.DefaultException
		if err := s.assertReady(0, row.line, row.expr); !errors.As(err, &ex) {
			t.Fatal("second source assertion did not Throw", row.line, err)
		}
	}
	var ex z3exception.DefaultException
	if err := s.assertQHead(199); !errors.As(err, &ex) {
		t.Fatal("post-propagation source assertion did not Throw", err)
	}
	state, err := input.State()
	satOK(t, err)
	if state.Position != 5 {
		t.Fatal("source debugger action count", state.Position)
	}
	got, err := os.ReadFile(output.Name())
	satOK(t, err)
	prompt := "(C)ontinue, (A)bort, (S)top, (T)hrow exception, Invoke (G)DB\n"
	assertion := func(line, expr string) string {
		return "ASSERTION VIOLATION\nFile: src/sat/sat_probing.cpp\nLine: " + line + "\n" + expr + "\n" + prompt
	}
	want := assertion("61", "s.m_qhead == s.m_trail.size()") + assertion("62", "s.value(l.var()) == l_undef") +
		assertion("112", "s.m_qhead == s.m_trail.size()") + assertion("113", "s.value(v) == l_undef") +
		assertion("199", "s.m_qhead == s.m_trail.size()")
	if string(got) != want {
		t.Fatalf("source assertion sequence: got %q, want %q", got, want)
	}
}

func TestSATProbingRunReadsReachedSourceAssertionSwitch(t *testing.T) {
	s, _ := satProbingFixture(t, 0)
	debug, _, _ := satIntegrityActionFixture(t, "")
	satOK(t, debug.Close())
	yes, err := s.run(false)
	if satStorageSourceDebug {
		if yes || !errors.Is(err, arena.ErrClosed) {
			t.Fatal("run skipped reached line 199", yes, err)
		}
	} else if err != nil || !yes {
		t.Fatal("release build evaluated erased assertion", yes, err)
	}
}
