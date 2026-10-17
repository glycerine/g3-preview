package sat

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/z3exception"
)

func TestSATIntegrityReinitAssertionReadsProcessAtReachedElement(t *testing.T) {
	g, _ := satGCFixture(t, 2)
	l := g.lifecycle()
	checker, e := newSATIntegrityChecker(l, g)
	satOK(t, e)
	debug, _, _ := satIntegrityActionFixture(t, "")
	satOK(t, debug.Close())
	previous := l.reinit
	defer func() { l.reinit = previous }()
	l.reinit = nil
	if yes, e := checker.checkReinitStack(); e != nil || !yes {
		t.Fatal("empty stack read erased assertion owner", yes, e)
	}
	l.reinit = []satClauseWrapper{newSATBinaryWrapper(0, 2)}
	if yes, e := checker.checkReinitStack(); e != nil || !yes {
		t.Fatal("binary wrapper read erased assertion owner", yes, e)
	}
	l.reinit = append(l.reinit, newSATClauseWrapper(StoredClauseRef{}))
	yes, e := checker.checkReinitStack()
	if satStorageSourceDebug {
		if yes || !errors.Is(e, arena.ErrClosed) {
			t.Fatal("nonbinary wrapper skipped reached assertion switch", yes, e)
		}
	} else if e != nil || !yes {
		t.Fatal("release SASSERT was not erased", yes, e)
	}
}

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
