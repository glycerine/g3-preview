package sat

// Source: src/sat/sat_integrity_checker.cpp:167-189; src/util/debug.h:64.

import (
	"errors"
	"os"
	"testing"

	"github.com/glycerine/g3/pkg/util/z3exception"
)

func TestSATIntegrityWatchKindAssertionsUseSourceActions(t *testing.T) {
	cases := []struct {
		name, line, condition string
	}{
		{"binary-eliminated", "171", "!s.was_eliminated(w.get_literal().var())"},
		{"binary-reciprocal", "181", "s.get_wlist(~(w.get_literal())).contains(watched(l, w.is_learned()))"},
		{"ternary-first-eliminated", "184", "!s.was_eliminated(w.get_literal1().var())"},
		{"ternary-second-eliminated", "185", "!s.was_eliminated(w.get_literal2().var())"},
		{"ternary-unsorted", "186", "w.get_literal1().index() < w.get_literal2().index()"},
		{"clause-removed", "189", "!s.m_cls_allocator.get_clause(w.get_clause_offset())->was_removed()"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			g, _ := satGCFixture(t, 3)
			l := g.lifecycle()
			v := l.propagation.variables
			list, err := v.watches.Get(1)
			satOK(t, err)
			var watch satWatched
			var clause StoredClauseRef
			hasClause := false
			switch tc.name {
			case "binary-eliminated", "binary-reciprocal":
				watch, err = newSATBinaryWatch(2, false)
				if tc.name == "binary-eliminated" {
					v.eliminated[1] = true
				}
			case "ternary-first-eliminated":
				watch, err = newSATTernaryWatch(2, 4)
				v.eliminated[1] = true
			case "ternary-second-eliminated":
				watch, err = newSATTernaryWatch(0, 2)
				v.eliminated[1] = true
			case "ternary-unsorted":
				watch, err = newSATTernaryWatch(0, 2)
				watch.val1, watch.val2 = 2, 1
			case "clause-removed":
				clause, err = l.propagation.clauses.MkClause([]Literal{0, 2, 4}, true)
				satOK(t, err)
				hasClause = true
				satOK(t, l.propagation.clauses.SetRemoved(clause, true))
				var offset satClauseOffset
				offset, err = l.propagation.clauses.Offset(clause)
				satOK(t, err)
				watch, err = newSATClauseWatch(0, offset)
			}
			satOK(t, err)
			satOK(t, list.Push(watch))
			defer func() {
				v.eliminated[1] = false
				satOK(t, list.SetEnd(0))
				if hasClause {
					satOK(t, l.propagation.clauses.Delete(clause))
				}
			}()
			debug, input, outputFile := satIntegrityActionFixture(t, "T")
			satOK(t, debug.EnableAssertions(true))
			checker, err := newSATIntegrityChecker(l, g)
			satOK(t, err)
			checker.debugActions = true
			yes, err := checker.checkWatches()
			if !satStorageSourceDebug {
				if !yes || err != nil {
					t.Fatalf("release checker reached assertion: result=%v error=%v", yes, err)
				}
				return
			}
			var ex z3exception.DefaultException
			if yes || !errors.As(err, &ex) {
				t.Fatalf("source assertion action: result=%v error=%v", yes, err)
			}
			state, err := input.State()
			satOK(t, err)
			if state.Position != 1 {
				t.Fatalf("debugger action count: %d", state.Position)
			}
			got, err := os.ReadFile(outputFile.Name())
			satOK(t, err)
			want := "ASSERTION VIOLATION\nFile: src/sat/sat_integrity_checker.cpp\nLine: " + tc.line + "\n" + tc.condition + "\n" +
				"(C)ontinue, (A)bort, (S)top, (T)hrow exception, Invoke (G)DB\n"
			if string(got) != want {
				t.Fatalf("source watch assertion: got %q, want %q", got, want)
			}
		})
	}
}

func TestSATIntegrityBinaryWatchContinuesFromEliminatedToReciprocalGuard(t *testing.T) {
	g, _ := satGCFixture(t, 2)
	l := g.lifecycle()
	v := l.propagation.variables
	list, err := v.watches.Get(1)
	satOK(t, err)
	watch, err := newSATBinaryWatch(2, false)
	satOK(t, err)
	satOK(t, list.Push(watch))
	v.eliminated[1] = true
	defer func() {
		v.eliminated[1] = false
		satOK(t, list.SetEnd(0))
	}()
	debug, input, outputFile := satIntegrityActionFixture(t, "CT")
	satOK(t, debug.EnableAssertions(true))
	checker, err := newSATIntegrityChecker(l, g)
	satOK(t, err)
	checker.debugActions = true
	yes, err := checker.checkWatches()
	if !satStorageSourceDebug {
		if !yes || err != nil {
			t.Fatalf("release checker reached assertion: result=%v error=%v", yes, err)
		}
		return
	}
	var ex z3exception.DefaultException
	if yes || !errors.As(err, &ex) {
		t.Fatalf("second source assertion action: result=%v error=%v", yes, err)
	}
	state, err := input.State()
	satOK(t, err)
	if state.Position != 2 {
		t.Fatalf("source guard order consumed %d actions", state.Position)
	}
	got, err := os.ReadFile(outputFile.Name())
	satOK(t, err)
	prompt := "(C)ontinue, (A)bort, (S)top, (T)hrow exception, Invoke (G)DB\n"
	want := "ASSERTION VIOLATION\nFile: src/sat/sat_integrity_checker.cpp\nLine: 171\n!s.was_eliminated(w.get_literal().var())\n" + prompt +
		"ASSERTION VIOLATION\nFile: src/sat/sat_integrity_checker.cpp\nLine: 181\ns.get_wlist(~(w.get_literal())).contains(watched(l, w.is_learned()))\n" + prompt
	if string(got) != want {
		t.Fatalf("source binary guard order: got %q, want %q", got, want)
	}
}
