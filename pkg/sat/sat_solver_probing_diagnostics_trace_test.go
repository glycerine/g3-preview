package sat

import (
	"bytes"
	"errors"
	"os"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/lbool"
	"github.com/glycerine/g3/pkg/util/z3exception"
)

// Source: src/sat/sat_solver.cpp:3053-3065. Both missed-unit and missed-empty
// paths print before the source unreachable assertion; frozen clauses do not.
func TestSATMissedPropagationSourceTraceBeforeAssertion(t *testing.T) {
	for _, mode := range []string{"unit", "empty", "frozen"} {
		t.Run(mode, func(t *testing.T) {
			_, life := satProbingFixture(t, 3)
			satIntegrityActionFixture(t, "T")
			c, e := life.propagation.clauses.MkClause([]Literal{0, 2, 4}, true)
			satOK(t, e)
			satOK(t, life.pushClause(true, c))
			var output bytes.Buffer
			tags := diagnostic.NewTraceTagsWithWriter(&output)
			tags.Enable("sat_missed_prop")
			life.traceTags = tags
			satOK(t, life.propagation.variables.setAssignment(0, lbool.False))
			satOK(t, life.propagation.variables.setAssignment(2, lbool.False))
			if mode != "unit" {
				satOK(t, life.propagation.variables.setAssignment(4, lbool.False))
			}
			if mode == "frozen" {
				satOK(t, life.propagation.clauses.Freeze(c))
			}
			e = life.checkMissedPropagation()
			if mode == "frozen" {
				satOK(t, e)
			} else if satStorageSourceDebug {
				var ex z3exception.DefaultException
				if !errors.As(e, &ex) {
					t.Fatal(e)
				}
			} else if !errors.Is(e, ErrSATAssertion) {
				t.Fatal(e)
			}
			want := ""
			if mode != "frozen" && tags.ShouldTrace("sat_missed_prop") {
				value := "l_undef"
				if mode == "empty" {
					value = "l_false"
				}
				want = satClauseTraceRecord("sat_missed_prop", "check_missed_propagation", "3061",
					"missed_propagation: (0 1 2)+*\n0: l_false\n1: l_false\n2: "+value+"\n")
			}
			if got := output.String(); got != want {
				t.Fatalf("trace = %q, want %q", got, want)
			}
		})
	}
}

func TestSATMissedPropagationUnreachableAndSeparateAssertions(t *testing.T) {
	for _, mode := range []string{"unit", "empty"} {
		for _, assertions := range []bool{false, true} {
			t.Run(mode+map[bool]string{false: "-off", true: "-on"}[assertions], func(t *testing.T) {
				_, life := satProbingFixture(t, 3)
				satCleanerClause(t, life, []Literal{0, 2, 4}, false, false)
				satOK(t, life.propagation.variables.setAssignment(0, lbool.False))
				satOK(t, life.propagation.variables.setAssignment(2, lbool.False))
				if mode == "empty" {
					satOK(t, life.propagation.variables.setAssignment(4, lbool.False))
				}
				choices := "C"
				if assertions {
					choices = "CT"
				}
				debug, input, output := satIntegrityActionFixture(t, choices)
				satOK(t, debug.EnableAssertions(assertions))
				err := life.checkMissedPropagation()
				if !satStorageSourceDebug {
					if !errors.Is(err, ErrSATAssertion) {
						t.Fatal("release undefined-domain guard", err)
					}
					return
				}
				if assertions {
					var ex z3exception.DefaultException
					if !errors.As(err, &ex) {
						t.Fatal("second source assertion did not throw", err)
					}
				} else {
					satOK(t, err)
				}
				state, err := input.State()
				satOK(t, err)
				wantActions := uint64(1)
				if assertions {
					wantActions = 2
				}
				if state.Position != wantActions {
					t.Fatal("source debugger actions", state.Position, wantActions)
				}
				got, err := os.ReadFile(output.Name())
				satOK(t, err)
				prompt := "(C)ontinue, (A)bort, (S)top, (T)hrow exception, Invoke (G)DB\n"
				want := "ASSERTION VIOLATION\nFile: src/sat/sat_solver.cpp\nLine: 3063\nUNREACHABLE CODE WAS REACHED.\n" + prompt
				if assertions {
					line, condition := "3066", "!is_unit(c)"
					if mode == "empty" {
						line, condition = "3065", "!is_empty(c)"
					}
					want += "ASSERTION VIOLATION\nFile: src/sat/sat_solver.cpp\nLine: " + line + "\n" + condition + "\n" + prompt
				}
				if string(got) != want {
					t.Fatalf("source diagnostic = %q, want %q", got, want)
				}
			})
		}
	}
}
