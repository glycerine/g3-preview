package sat

import (
	"errors"
	"os"
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/util/z3exception"
)

func TestSATCleanerSmallActiveClauseSourceAssertions(t *testing.T) {
	for _, tc := range []struct {
		name        string
		assignments []Literal
		line        int
	}{
		{"empty", []Literal{1, 3, 5}, 123},
		{"unit", []Literal{1, 3}, 130},
	} {
		for _, enabled := range []bool{false, true} {
			name := tc.name + "-disabled"
			if enabled {
				name = tc.name + "-enabled"
			}
			t.Run(name, func(t *testing.T) {
				c, l := satCleanerFixture(t, 3)
				satCleanerClause(t, l, []Literal{0, 2, 4}, false, false)
				for _, lit := range tc.assignments {
					satOK(t, l.propagation.assign(lit, satJustification{}))
				}
				defer func() { satOK(t, l.unassignVars(0)) }()
				debug, input, outputFile := satIntegrityActionFixture(t, "CT")
				c.debugActions = true
				defer func() { c.debugActions = false }()
				satOK(t, debug.EnableAssertions(enabled))
				err := c.cleanupClauses(false)
				if enabled && satStorageSourceDebug {
					var ex z3exception.DefaultException
					if !errors.As(err, &ex) {
						t.Fatalf("small active clause source T action: %v", err)
					}
					state, e := input.State()
					satOK(t, e)
					if state.Position != 2 {
						t.Fatalf("small-clause actions consumed: %d", state.Position)
					}
					contents, e := os.ReadFile(outputFile.Name())
					satOK(t, e)
					first := "ASSERTION VIOLATION\nFile: src/sat/sat_cleaner.cpp\nLine: 119\nc.frozen() || new_sz >= 2\n" +
						"(C)ontinue, (A)bort, (S)top, (T)hrow exception, Invoke (G)DB\n"
					second := "ASSERTION VIOLATION\nFile: src/sat/sat_cleaner.cpp\nLine: " + strconv.Itoa(tc.line) + "\nc.frozen()\n" +
						"(C)ontinue, (A)bort, (S)top, (T)hrow exception, Invoke (G)DB\n"
					if string(contents) != first+second {
						t.Fatalf("small-clause source records: got %q, want %q", contents, first+second)
					}
				} else {
					satOK(t, err)
					state, e := input.State()
					satOK(t, e)
					if state.Position != 0 {
						t.Fatalf("erased assertions consumed %d actions", state.Position)
					}
				}
			})
		}
	}
}
