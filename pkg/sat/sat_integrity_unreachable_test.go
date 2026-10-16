package sat

import (
	"errors"
	"os"
	"testing"

	"github.com/glycerine/g3/pkg/util/z3exception"
)

func TestSATCleanerMissingNaryWatchUnreachableThenVerify(t *testing.T) {
	g, _ := satGCFixture(t, 4)
	l := g.lifecycle()
	cl := satCleanerClause(t, l, []Literal{0, 2, 4, 6}, false, false)
	satOK(t, l.attachClause(cl))
	list, err := l.propagation.variables.watches.Get(1)
	satOK(t, err)
	prior, err := list.Read(0)
	satOK(t, err)
	satOK(t, list.SetEnd(0))
	defer func() {
		satOK(t, list.SetEnd(0))
		satOK(t, list.Push(prior))
	}()
	checker, err := newSATIntegrityChecker(l, g)
	satOK(t, err)
	if _, err = checker.checkClause(cl); !errors.Is(err, ErrSATAssertion) {
		t.Fatalf("unbound missing-watch guard: %v", err)
	}
	debug, input, outputFile := satIntegrityActionFixture(t, "CT")
	satOK(t, debug.EnableDebug("cleaner_bug"))
	satOK(t, debug.EnableAssertions(false))
	satOK(t, g.cleaner.assertInvariant())
	satOK(t, debug.EnableAssertions(true))
	if !satStorageSourceDebug {
		satOK(t, g.cleaner.assertInvariant())
		return
	}
	var ex z3exception.DefaultException
	if err = g.cleaner.assertInvariant(); !errors.As(err, &ex) {
		t.Fatalf("VERIFY after UNREACHABLE C action: %v", err)
	}
	state, err := input.State()
	satOK(t, err)
	if state.Position != 2 {
		t.Fatalf("missing-watch actions consumed: %d", state.Position)
	}
	contents, err := os.ReadFile(outputFile.Name())
	satOK(t, err)
	want := "ASSERTION VIOLATION\nFile: src/sat/sat_integrity_checker.cpp\nLine: 48\nUNREACHABLE CODE WAS REACHED.\n" +
		"(C)ontinue, (A)bort, (S)top, (T)hrow exception, Invoke (G)DB\n" +
		"Failed to verify: contains_watched(s.get_wlist(~c[0]), c, s.get_offset(c))\n" +
		"ASSERTION VIOLATION\nFile: src/sat/sat_integrity_checker.cpp\nLine: 99\nUNREACHABLE CODE WAS REACHED.\n" +
		"(C)ontinue, (A)bort, (S)top, (T)hrow exception, Invoke (G)DB\n"
	if string(contents) != want {
		t.Fatalf("missing-watch source actions: got %q, want %q", contents, want)
	}
	satOK(t, list.Push(prior))
	satOK(t, g.cleaner.assertInvariant())
}
