package sat

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/z3exception"
)

func TestSATCleanerReinitStackAssertionUsesSourceActions(t *testing.T) {
	g, _ := satGCFixture(t, 5)
	l := g.lifecycle()
	cl := satCleanerClause(t, l, []Literal{0, 2, 4, 6}, false, false)
	satOK(t, l.attachClause(cl))
	satOK(t, l.pushReinitStack(cl))
	satOK(t, l.propagation.clauses.SetReinitStack(cl, false))
	defer func() { satOK(t, l.propagation.clauses.SetReinitStack(cl, true)) }()
	debug, input, outputFile := satIntegrityActionFixture(t, "CT")
	satOK(t, debug.EnableDebug("cleaner_bug"))
	satOK(t, debug.EnableAssertions(false))
	before := l.propagation.limit.Steps
	satOK(t, g.cleaner.assertInvariant())
	if got := l.propagation.limit.Steps; got != before {
		t.Fatalf("disabled operator() reached resource limit: %d", got)
	}
	satOK(t, debug.EnableAssertions(true))
	if !satStorageSourceDebug {
		satOK(t, g.cleaner.assertInvariant())
		return
	}
	satOK(t, g.cleaner.assertInvariant())
	var ex z3exception.DefaultException
	if err := g.cleaner.assertInvariant(); !errors.As(err, &ex) {
		t.Fatalf("reinit T action: %v", err)
	}
	state, err := input.State()
	satOK(t, err)
	if state.Position != 2 {
		t.Fatalf("reinit debugger consumed %d actions", state.Position)
	}
	contents, err := os.ReadFile(outputFile.Name())
	satOK(t, err)
	want := "ASSERTION VIOLATION\nFile: src/sat/sat_integrity_checker.cpp\nLine: 205\nit->get_clause()->on_reinit_stack()\n" +
		"(C)ontinue, (A)bort, (S)top, (T)hrow exception, Invoke (G)DB\n"
	if string(contents) != strings.Repeat(want, 2) {
		t.Fatalf("reinit source records: got %q, want %q", contents, strings.Repeat(want, 2))
	}
	satOK(t, l.propagation.clauses.SetReinitStack(cl, true))
	satOK(t, g.cleaner.assertInvariant())
}
