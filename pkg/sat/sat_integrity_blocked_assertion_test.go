package sat

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/z3exception"
)

func TestSATCleanerNaryBlockedLiteralUsesSourceAssertion(t *testing.T) {
	g, _ := satGCFixture(t, 5)
	l := g.lifecycle()
	cl := satCleanerClause(t, l, []Literal{0, 2, 4, 6}, false, false)
	satOK(t, l.attachClause(cl))
	list, err := l.propagation.variables.watches.Get(1)
	satOK(t, err)
	prior, err := list.Read(0)
	satOK(t, err)
	changed := prior
	satOK(t, changed.SetBlockedLiteral(8))
	satOK(t, list.Write(0, changed))
	defer func() { satOK(t, list.Write(0, prior)) }()
	checker, err := newSATIntegrityChecker(l, g)
	satOK(t, err)
	debug, input, outputFile := satIntegrityActionFixture(t, "CT")
	satOK(t, debug.EnableDebug("cleaner_bug"))
	satOK(t, debug.EnableAssertions(false))
	yes, err := checker.checkClause(cl)
	satOK(t, err)
	if !yes {
		t.Fatal("disabled helper assertion refused valid watch identity")
	}
	satOK(t, g.cleaner.assertInvariant())
	satOK(t, debug.EnableAssertions(true))
	if !satStorageSourceDebug {
		satOK(t, g.cleaner.assertInvariant())
		return
	}
	satOK(t, g.cleaner.assertInvariant())
	var ex z3exception.DefaultException
	if err = g.cleaner.assertInvariant(); !errors.As(err, &ex) {
		t.Fatalf("blocked-literal T action: %v", err)
	}
	state, err := input.State()
	satOK(t, err)
	if state.Position != 2 {
		t.Fatalf("blocked-literal action count: %d", state.Position)
	}
	contents, err := os.ReadFile(outputFile.Name())
	satOK(t, err)
	want := "ASSERTION VIOLATION\nFile: src/sat/sat_integrity_checker.cpp\nLine: 43\nc.contains(it->get_blocked_literal())\n" +
		"(C)ontinue, (A)bort, (S)top, (T)hrow exception, Invoke (G)DB\n"
	if string(contents) != strings.Repeat(want, 2) {
		t.Fatalf("blocked-literal source records: got %q, want %q", contents, strings.Repeat(want, 2))
	}
	satOK(t, list.Write(0, prior))
	satOK(t, g.cleaner.assertInvariant())
}
