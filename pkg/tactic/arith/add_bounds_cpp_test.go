package arith_test

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/cmd_context"
	_ "github.com/glycerine/g3/pkg/engine"
	"github.com/glycerine/g3/pkg/parsers/smt2"
	"github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/tactic/arith"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

func addBoundsMust(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
func addBoundsContext(t *testing.T, script string) *cmd_context.Context {
	t.Helper()
	c, err := cmd_context.NewStandaloneContext()
	addBoundsMust(t, err)
	t.Cleanup(func() { addBoundsMust(t, c.Close()) })
	addBoundsMust(t, smt2.ParseCommands(c, strings.NewReader(script), io.Discard))
	return c
}
func addBoundsCPP(t *testing.T, script string) string {
	t.Helper()
	reference := os.Getenv("G3_CPP_REFERENCE")
	if reference == "" {
		t.Skip("set G3_CPP_REFERENCE to prebuilt z3")
	}
	cmd := exec.Command(reference, "-in")
	cmd.Stdin = strings.NewReader(script)
	var output, diagnostics bytes.Buffer
	cmd.Stdout, cmd.Stderr = &output, &diagnostics
	if err := cmd.Run(); err != nil {
		t.Fatal(err, output.String(), diagnostics.String())
	}
	return output.String()
}

func TestAddBoundsExactGoalsAndUnboundedProbeCPP(t *testing.T) {
	for _, tc := range []struct {
		name, script string
		unbounded    bool
		added        int
	}{
		{"unbounded", `(set-logic ALL)(declare-const x Int)(assert (= (+ x 1) 4))`, true, 2},
		{"already-bounded", `(set-logic ALL)(declare-const x Int)(assert (>= x 10))(assert (<= x 20))`, false, 2},
		{"lower-only", `(set-logic ALL)(declare-const x Int)(assert (>= x 10))`, true, 2},
		{"upper-only", `(set-logic ALL)(declare-const x Real)(assert (< x 20))`, true, 2},
		{"equality-bound", `(set-logic ALL)(declare-const x Real)(assert (= x 4.0))`, false, 2},
		{"coerced-equality-not-bound", `(set-logic ALL)(declare-const x Real)(assert (= x 4))`, true, 2},
		{"two-variables", `(set-logic ALL)(declare-const x Int)(declare-const y Real)(assert (= (+ x y) 0))`, true, 4},
		{"shared-once", `(set-logic ALL)(declare-const x Int)(assert (= (+ x x x) 0))`, true, 2},
		{"Boolean", `(set-logic ALL)(declare-const p Bool)(assert p)`, false, 0},
		{"bound-variable", `(set-logic ALL)(assert (forall ((x Int)) (= x 0)))`, false, 0},
		{"quantified-free-constant", `(set-logic ALL)(declare-const x Int)(assert (forall ((y Int)) (= (+ x y) y)))`, true, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := addBoundsContext(t, tc.script)
			m, err := c.Manager()
			addBoundsMust(t, err)
			g, err := tactic.NewGoal(m, false, true, false)
			addBoundsMust(t, err)
			t.Cleanup(func() { addBoundsMust(t, g.Close()) })
			addBoundsMust(t, c.AssertGoal(g))
			globals, err := c.GlobalParameters()
			addBoundsMust(t, err)
			probe, err := arith.NewIsUnboundedProbe(m.Budget(), globals.Context, globals)
			addBoundsMust(t, err)
			t.Cleanup(func() { addBoundsMust(t, probe.Close()) })
			value, err := probe.Eval(g)
			addBoundsMust(t, err)
			if (value != 0) != tc.unbounded {
				t.Fatal("unbounded classification", value, tc.unbounded)
			}
			before, err := g.Size()
			addBoundsMust(t, err)
			r, err := c.NewBuiltinTactic("add-bounds")
			addBoundsMust(t, err)
			addBoundsMust(t, r.IncRef())
			t.Cleanup(func() { addBoundsMust(t, r.DecRef()) })
			var result tactic.TacticResult
			t.Cleanup(func() { addBoundsMust(t, result.Close()) })
			addBoundsMust(t, r.Apply(g, &result))
			addBoundsMust(t, r.Cleanup())
			after, err := g.Size()
			addBoundsMust(t, err)
			if int(after-before) != tc.added {
				t.Fatal("bounds inserted", before, after, tc.added)
			}
			if g.UnsatPreserved() != (tc.added == 0) {
				t.Fatal("under-approximation precision")
			}
			previous, err := uparams.BindProcessGlobalParamsService(globals)
			addBoundsMust(t, err)
			t.Cleanup(func() { _, err := uparams.BindProcessGlobalParamsService(previous); addBoundsMust(t, err) })
			var actual bytes.Buffer
			actual.WriteString("(goals\n")
			for _, goal := range result.Goals {
				addBoundsMust(t, goal.DisplayWithAttributes(&actual))
			}
			actual.WriteString(")\n")
			if os.Getenv("G3_CPP_REFERENCE") != "" {
				native := addBoundsCPP(t, tc.script+"(apply add-bounds)")
				if strings.Join(strings.Fields(actual.String()), " ") != strings.Join(strings.Fields(native), " ") {
					t.Fatal("Go/C++ bound goals", actual.String(), native)
				}
				branch := addBoundsCPP(t, tc.script+"(apply (if is-unbounded fail skip))")
				if strings.Contains(branch, "tactic failed: fail tactic") != tc.unbounded {
					t.Fatal("native unbounded branch differs", branch)
				}
			}
		})
	}
}
