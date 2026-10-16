package sat

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/z3exception"
)

func TestSATCleanerFalseBinaryWatchSourceAssertion(t *testing.T) {
	c, l := satCleanerFixture(t, 2)
	list, err := l.propagation.variables.watches.Get(1)
	satOK(t, err)
	w, err := newSATBinaryWatch(2, false)
	satOK(t, err)
	satOK(t, list.Push(w))
	satOK(t, l.propagation.assign(3, satJustification{}))
	defer func() {
		satOK(t, list.SetEnd(0))
		satOK(t, l.unassignVars(0))
	}()
	debug, input, outputFile := satIntegrityActionFixture(t, "CT")
	c.debugActions = true
	defer func() { c.debugActions = false }()
	satOK(t, debug.EnableAssertions(false))
	satOK(t, c.cleanupWatches())
	state, err := input.State()
	satOK(t, err)
	if state.Position != 0 {
		t.Fatalf("disabled assertion consumed %d actions", state.Position)
	}
	satOK(t, list.Push(w))
	satOK(t, debug.EnableAssertions(true))
	if !satStorageSourceDebug {
		satOK(t, c.cleanupWatches())
		return
	}
	satOK(t, c.cleanupWatches())
	satOK(t, list.Push(w))
	var ex z3exception.DefaultException
	if err = c.cleanupWatches(); !errors.As(err, &ex) {
		t.Fatalf("false-watch T action: %v", err)
	}
	state, err = input.State()
	satOK(t, err)
	if state.Position != 2 {
		t.Fatalf("false-watch actions consumed: %d", state.Position)
	}
	contents, err := os.ReadFile(outputFile.Name())
	satOK(t, err)
	want := "ASSERTION VIOLATION\nFile: src/sat/sat_cleaner.cpp\nLine: 56\n" +
		"s.value(it2->get_literal()) == l_true || s.value(it2->get_literal()) == l_undef\n" +
		"(C)ontinue, (A)bort, (S)top, (T)hrow exception, Invoke (G)DB\n"
	if string(contents) != strings.Repeat(want, 2) {
		t.Fatalf("false-watch source actions: got %q, want %q", contents, strings.Repeat(want, 2))
	}
}
