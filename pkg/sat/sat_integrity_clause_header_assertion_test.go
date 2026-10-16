package sat

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/z3exception"
)

func TestSATCleanerFrozenClauseEliminatedLiteralUsesSourceAssertion(t *testing.T) {
	g, _ := satGCFixture(t, 3)
	l := g.lifecycle()
	satCleanerClause(t, l, []Literal{0, 2, 4}, true, true)
	v := l.propagation.variables
	v.eliminated[0] = true
	defer func() { v.eliminated[0] = false }()
	debug, input, outputFile := satIntegrityActionFixture(t, "CT")
	satOK(t, debug.EnableDebug("cleaner_bug"))
	satOK(t, debug.EnableAssertions(false))
	satOK(t, g.cleaner.assertInvariant())
	satOK(t, debug.EnableAssertions(true))
	if !satStorageSourceDebug {
		satOK(t, g.cleaner.assertInvariant())
		return
	}
	satOK(t, g.cleaner.assertInvariant())
	var ex z3exception.DefaultException
	if err := g.cleaner.assertInvariant(); !errors.As(err, &ex) {
		t.Fatalf("eliminated-literal T action: %v", err)
	}
	state, err := input.State()
	satOK(t, err)
	if state.Position != 2 {
		t.Fatalf("clause header consumed %d actions", state.Position)
	}
	contents, err := os.ReadFile(outputFile.Name())
	satOK(t, err)
	want := "ASSERTION VIOLATION\nFile: src/sat/sat_integrity_checker.cpp\nLine: 60\n!s.was_eliminated(c[i].var())\n" +
		"(C)ontinue, (A)bort, (S)top, (T)hrow exception, Invoke (G)DB\n"
	if string(contents) != strings.Repeat(want, 2) {
		t.Fatalf("clause header source records: got %q, want %q", contents, strings.Repeat(want, 2))
	}
	v.eliminated[0] = false
	satOK(t, g.cleaner.assertInvariant())
}
