package arith

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/util/memory"
)

func TestPropagateIneqsAdapterCPPGoalShapes(t *testing.T) {
	reference := os.Getenv("G3_CPP_REFERENCE")
	if reference == "" {
		t.Skip("set G3_CPP_REFERENCE to prebuilt z3")
	}
	for _, mode := range []string{"empty", "opposite", "strict", "linear", "inconsistent"} {
		t.Run(mode, func(t *testing.T) {
			r := inequalityAdapterFixture(t, false)
			m := r.manager
			x := fixDLConst(t, m, false, "x")
			y := fixDLConst(t, m, false, "y")
			number := func(s string) ast.ASTHandle { return fixDLNumber(t, m, false, s) }
			app := func(op int32, args ...ast.ASTHandle) ast.ASTHandle { return fixDLApp(t, m, op, args...) }
			var forms []ast.ASTHandle
			switch mode {
			case "opposite":
				forms = []ast.ASTHandle{app(3, x, number("2")), app(2, x, number("2"))}
			case "strict":
				lower, err := m.MkNot(app(2, x, number("1.25")))
				equationMust(t, err)
				fixDLPin(t, m, lower)
				forms = []ast.ASTHandle{lower, app(2, x, number("2.75"))}
			case "linear":
				forms = []ast.ASTHandle{app(3, x, number("2")), app(2, x, number("2")),
					app(3, app(6, y, app(9, number("2"), x)), number("9")), app(2, y, number("5"))}
			case "inconsistent":
				forms = []ast.ASTHandle{app(3, x, number("3")), app(2, x, number("2"))}
			}
			goal := fixDLGoal(t, m, false, false, false, forms...)
			var script strings.Builder
			script.WriteString("(set-logic QF_LRA)(declare-const x Real)(declare-const y Real)\n")
			for _, form := range forms {
				text, err := r.core.current.builder.traceSMT2Term(form)
				equationMust(t, err)
				fmt.Fprintf(&script, "(assert %s)\n", text)
			}
			if mode == "strict" {
				// The pretty-printer spells nonintegral numeral ASTs as
				// division applications. Parse decimals to give C++ the same
				// numeral nodes, without adding a simplification stage.
				script.Reset()
				script.WriteString("(set-logic QF_LRA)(declare-const x Real)(assert (not (<= x 1.25)))(assert (<= x 2.75))\n")
			}
			script.WriteString("(apply propagate-ineqs)")
			var result tactic.TacticResult
			t.Cleanup(func() { equationMust(t, result.Close()) })
			equationMust(t, r.Apply(goal, &result))
			var actual bytes.Buffer
			actual.WriteString("(goals (goal ")
			out := result.Goals[0]
			size, err := out.Size()
			equationMust(t, err)
			for i := uint32(0); i < size; i++ {
				form, err := out.Form(i)
				equationMust(t, err)
				text, err := r.core.current.builder.traceSMT2Term(form)
				equationMust(t, err)
				actual.WriteString(text + " ")
			}
			depth, err := out.Depth()
			equationMust(t, err)
			fmt.Fprintf(&actual, ":precision precise :depth %d))", depth)
			cmd := exec.Command(reference, "-in")
			cmd.Stdin = strings.NewReader(script.String())
			native, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatal(err, string(native))
			}
			canonical := func(s string) string {
				return strings.Join(strings.Fields(strings.NewReplacer("(", " ( ", ")", " ) ").Replace(s)), " ")
			}
			if canonical(actual.String()) != canonical(string(native)) {
				t.Fatalf("goal mismatch\nGo: %s\nC++: %s", actual.String(), native)
			}
		})
	}
}

func inequalityAdapterFixture(t *testing.T, proofs bool) *PropagateIneqsTactic {
	t.Helper()
	base, m := fixDLFixture(t, proofs)
	r, err := NewPropagateIneqsTactic(m, base.context, nil)
	equationMust(t, err)
	t.Cleanup(func() { equationMust(t, r.Close()) })
	return r
}

func TestPropagateIneqsAdapterGoalsGuardsAndTranslation(t *testing.T) {
	r := inequalityAdapterFixture(t, true)
	m := r.manager
	x := fixDLConst(t, m, true, "adapter-x")
	bounded := fixDLGoal(t, m, false, true, false, diffNeqTestBounds(t, m, x, "5")...)
	equationMust(t, bounded.SetDepth(4))
	var result tactic.TacticResult
	t.Cleanup(func() { equationMust(t, result.Close()) })
	equationMust(t, r.Apply(bounded, &result))
	if len(result.Goals) != 1 || result.Goals[0] == bounded {
		t.Fatal("bounded result must transfer a fresh goal")
	}
	if depth, err := result.Goals[0].Depth(); err != nil || depth != 5 {
		t.Fatal("bounded result depth", depth, err)
	}
	owned := result.Goals[0]
	equationMust(t, result.Reset())
	if err := owned.CheckOwner(); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("result did not close transferred goal", err)
	}
	equationMust(t, r.Cleanup())
	empty := fixDLGoal(t, m, false, true, false)
	equationMust(t, empty.SetDepth(9))
	equationMust(t, r.Apply(empty, &result))
	if len(result.Goals) != 1 || result.Goals[0] != empty {
		t.Fatal("no bounds must return borrowed input")
	}
	for _, entry := range []struct {
		proof, core bool
		message     string
	}{
		{true, false, "propagate-ineqs does not support proof production"},
		{false, true, "propagate-ineqs does not support unsat core production"},
	} {
		goal := fixDLGoal(t, m, entry.proof, false, entry.core)
		if err := r.Apply(goal, &result); err == nil || err.Error() != entry.message {
			t.Fatal("source guard", err)
		}
		if len(result.Goals) != 1 || result.Goals[0] != empty {
			t.Fatal("source guards must precede result reset")
		}
	}
	destination, err := ast.NewASTManager(ast.ASTManagerConfig{Symbols: m.Symbols, Numerals: m.NumeralService(), Budget: m.Budget()})
	equationMust(t, err)
	t.Cleanup(func() { equationMust(t, destination.Close()) })
	other, err := r.Translate(destination)
	equationMust(t, err)
	translated := other.(*PropagateIneqsTactic)
	t.Cleanup(func() { equationMust(t, translated.Close()) })
	if translated.numbers == r.numbers || translated.core.current.store == r.core.current.store {
		t.Fatal("translation shared mutable arithmetic owners")
	}
	equationMust(t, r.Close())
	equationMust(t, result.Reset())
	targetX := fixDLConst(t, destination, true, "adapter-x")
	targetGoal := fixDLGoal(t, destination, false, true, false, diffNeqTestBounds(t, destination, targetX, "5")...)
	// Results retain their manager even when reset; translation publishes into
	// a separate destination result, as the source goal_ref_buffer does.
	var targetResult tactic.TacticResult
	t.Cleanup(func() { equationMust(t, targetResult.Close()) })
	equationMust(t, translated.Apply(targetGoal, &targetResult))
	equationMust(t, translated.IncRef())
	equationMust(t, translated.DecRef())
	if err := translated.Cleanup(); !errors.Is(err, arena.ErrClosed) {
		t.Fatal("reference teardown", err)
	}
}

func TestPropagateIneqsAdapterAllocationFrontiers(t *testing.T) {
	for _, phase := range []string{"constructor", "apply", "cleanup"} {
		success := false
		for stop := int64(0); stop < 1024; stop++ {
			t.Run(fmt.Sprint(phase, "/", stop), func(t *testing.T) {
				r := inequalityAdapterFixture(t, false)
				m := r.manager
				if phase == "constructor" {
					before := m.Budget().Used()
					m.Budget().FailAfter(stop)
					fresh, err := NewPropagateIneqsTactic(m, r.context, nil)
					m.Budget().FailAfter(-1)
					if fresh != nil {
						equationMust(t, fresh.Close())
					}
					if after := m.Budget().Used(); after != before {
						t.Fatal("constructor leaked reservations", before, after)
					}
					if err == nil {
						success = true
					} else if !errors.Is(err, arena.ErrInjectedFailure) {
						// Private numeral-manager construction uses the source
						// runtime's typed memout transport for allocation refusal.
						var failure memory.Failure
						if !errors.As(err, &failure) || failure.Kind != memory.FailureMemout {
							t.Fatal(err)
						}
					}
					return
				}
				x := fixDLConst(t, m, true, "frontier-x")
				goal := fixDLGoal(t, m, false, true, false, diffNeqTestBounds(t, m, x, "5")...)
				var result tactic.TacticResult
				t.Cleanup(func() { equationMust(t, result.Close()) })
				if phase == "cleanup" {
					equationMust(t, r.Apply(goal, &result))
				}
				m.Budget().FailAfter(stop)
				var err error
				if phase == "cleanup" {
					err = r.Cleanup()
				} else {
					err = r.Apply(goal, &result)
				}
				m.Budget().FailAfter(-1)
				if err == nil {
					success = true
				} else if !errors.Is(err, arena.ErrInjectedFailure) {
					t.Fatal(err)
				}
			})
			if success {
				break
			}
		}
		if !success {
			t.Fatal("no successful reservation frontier", phase)
		}
	}
}
