package sat

import (
	"errors"
	"os"
	"testing"

	"github.com/glycerine/g3/pkg/util/z3exception"
)

func TestSATCleanerEliminatedWatchAssertionsFollowSourceOrder(t *testing.T) {
	g, _ := satGCFixture(t, 2)
	c := g.cleaner
	v := g.lifecycle().propagation.variables
	first, err := v.watches.Get(0)
	satOK(t, err)
	second, err := v.watches.Get(1)
	satOK(t, err)
	w, err := newSATExternalWatch(3)
	satOK(t, err)
	satOK(t, first.Push(w))
	satOK(t, second.Push(w))
	v.eliminated[0] = true
	defer func() {
		v.eliminated[0] = false
		satOK(t, first.SetEnd(0))
		satOK(t, second.SetEnd(0))
	}()
	debug, input, outputFile := satIntegrityActionFixture(t, "CCCCT")
	satOK(t, debug.EnableDebug("cleaner_bug"))
	satOK(t, debug.EnableAssertions(false))
	satOK(t, c.assertInvariant())
	satOK(t, debug.EnableAssertions(true))
	if !satStorageSourceDebug {
		satOK(t, c.assertInvariant())
		return
	}
	satOK(t, c.assertInvariant())
	var ex z3exception.DefaultException
	if err = c.assertInvariant(); !errors.As(err, &ex) {
		t.Fatalf("nested watch T action: %v", err)
	}
	state, err := input.State()
	satOK(t, err)
	if state.Position != 5 {
		t.Fatalf("nested watch actions consumed %d bytes", state.Position)
	}
	contents, err := os.ReadFile(outputFile.Name())
	satOK(t, err)
	prompt := "(C)ontinue, (A)bort, (S)top, (T)hrow exception, Invoke (G)DB\n"
	assertion := func(line, condition string) string {
		return "ASSERTION VIOLATION\nFile: src/sat/sat_integrity_checker.cpp\nLine: " + line + "\n" + condition + "\n" + prompt
	}
	want := assertion("167", "!s.was_eliminated(l.var()) || wlist.empty()") +
		assertion("167", "!s.was_eliminated(l.var()) || wlist.empty()") +
		assertion("148", "s.get_wlist(literal(v, false)).empty()") +
		assertion("149", "s.get_wlist(literal(v, true)).empty()") +
		assertion("167", "!s.was_eliminated(l.var()) || wlist.empty()")
	if string(contents) != want {
		t.Fatalf("nested watch source records: got %q, want %q", contents, want)
	}
	v.eliminated[0] = false
	satOK(t, first.SetEnd(0))
	satOK(t, second.SetEnd(0))
	satOK(t, c.assertInvariant())
}
