package sat

import (
	"errors"
	"os"
	"testing"

	"github.com/glycerine/g3/pkg/util/z3exception"
)

func TestSATCleanerTernaryVerifySourceActions(t *testing.T) {
	g, _ := satGCFixture(t, 3)
	l := g.lifecycle()
	cl := satCleanerClause(t, l, []Literal{0, 2, 4}, false, false)
	satOK(t, l.attachClause(cl))
	watches := [3]uint32{1, 3, 5}
	for _, index := range watches {
		list, err := l.propagation.variables.watches.Get(index)
		satOK(t, err)
		prior, err := list.Read(0)
		satOK(t, err)
		satOK(t, list.SetEnd(0))
		defer func() {
			satOK(t, list.SetEnd(0))
			satOK(t, list.Push(prior))
		}()
	}
	checker, err := newSATIntegrityChecker(l, g)
	satOK(t, err)
	if _, err = checker.checkClause(cl); !errors.Is(err, ErrSATAssertion) {
		t.Fatalf("unbound ternary VERIFY: %v", err)
	}
	debug, input, outputFile := satIntegrityActionFixture(t, "CCT")
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
		t.Fatalf("ternary third VERIFY T action: %v", err)
	}
	state, err := input.State()
	satOK(t, err)
	if state.Position != 3 {
		t.Fatalf("ternary VERIFY actions consumed: %d", state.Position)
	}
	contents, err := os.ReadFile(outputFile.Name())
	satOK(t, err)
	want := "Failed to verify: contains_watched(s.get_wlist(~c[0]), c[1], c[2])\n" +
		"ASSERTION VIOLATION\nFile: src/sat/sat_integrity_checker.cpp\nLine: 73\nUNREACHABLE CODE WAS REACHED.\n" +
		"(C)ontinue, (A)bort, (S)top, (T)hrow exception, Invoke (G)DB\n" +
		"Failed to verify: contains_watched(s.get_wlist(~c[1]), c[0], c[2])\n" +
		"ASSERTION VIOLATION\nFile: src/sat/sat_integrity_checker.cpp\nLine: 74\nUNREACHABLE CODE WAS REACHED.\n" +
		"(C)ontinue, (A)bort, (S)top, (T)hrow exception, Invoke (G)DB\n" +
		"Failed to verify: contains_watched(s.get_wlist(~c[2]), c[0], c[1])\n" +
		"ASSERTION VIOLATION\nFile: src/sat/sat_integrity_checker.cpp\nLine: 75\nUNREACHABLE CODE WAS REACHED.\n" +
		"(C)ontinue, (A)bort, (S)top, (T)hrow exception, Invoke (G)DB\n"
	if string(contents) != want {
		t.Fatalf("ternary source VERIFY records: got %q, want %q", contents, want)
	}
	// Deferred restorations run before the solver's invariant-checking close.
}
