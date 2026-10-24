package core_test

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/cmd_context"
	_ "github.com/glycerine/g3/pkg/engine"
	"github.com/glycerine/g3/pkg/parsers/smt2"
	"github.com/glycerine/g3/pkg/tactic"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

func ctxMust(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

const ctxDeclarations = `(declare-const x Int)(declare-const y Int)(declare-const p Bool)(declare-const q Bool)(declare-const r Bool)`

func ctxApplyDisplay(t *testing.T, script string, proofs, cores bool, settings map[string]uint32) string {
	t.Helper()
	c, err := cmd_context.NewStandaloneContext()
	ctxMust(t, err)
	t.Cleanup(func() { ctxMust(t, c.Close()) })
	ctxMust(t, smt2.ParseCommands(c, strings.NewReader(script), io.Discard))
	m, err := c.Manager()
	ctxMust(t, err)
	g, err := tactic.NewGoal(m, proofs, true, cores)
	ctxMust(t, err)
	t.Cleanup(func() { ctxMust(t, g.Close()) })
	ctxMust(t, c.AssertGoal(g))
	runner, err := c.NewBuiltinTactic("ctx-simplify")
	ctxMust(t, err)
	ctxMust(t, runner.IncRef())
	t.Cleanup(func() { ctxMust(t, runner.DecRef()) })
	globals, err := c.GlobalParameters()
	ctxMust(t, err)
	params, err := globals.Context.NewValue()
	ctxMust(t, err)
	t.Cleanup(func() { ctxMust(t, globals.Context.Del(&params)) })
	for name, value := range settings {
		ctxMust(t, globals.Context.SetUIntName(&params, &name, value))
	}
	ctxMust(t, runner.UpdateParams(&params))
	var result tactic.TacticResult
	t.Cleanup(func() { ctxMust(t, result.Close()) })
	ctxMust(t, runner.Apply(g, &result))
	ctxMust(t, runner.Cleanup())
	previous, err := uparams.BindProcessGlobalParamsService(globals)
	ctxMust(t, err)
	t.Cleanup(func() { _, err := uparams.BindProcessGlobalParamsService(previous); ctxMust(t, err) })
	var output bytes.Buffer
	output.WriteString("(goals\n")
	for _, goal := range result.Goals {
		ctxMust(t, goal.DisplayWithAttributes(&output))
	}
	output.WriteString(")\n")
	return output.String()
}
func ctxCanonical(s string) string { return strings.Join(strings.Fields(s), " ") }
func ctxNative(t *testing.T, script string) string {
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

func TestContextSimplifyExactGoalsCPP(t *testing.T) {
	for _, tc := range []struct {
		name, assertions, want string
		proofs, cores          bool
	}{
		{"cross-form", `(assert (= x 3))(assert (= (+ x 1) 4))(assert (or p (not p)))`, `(goals (goal (= x 3) true true :precision precise :depth 1) )`, false, false},
		{"ite-scopes", `(assert (= (ite p (ite p x y) (ite p y x)) x))`, `(goals (goal true :precision precise :depth 1) )`, false, false},
		{"or-forward-backward", `(assert (or p (and (not p) q)))`, `(goals (goal (or p q) :precision precise :depth 1) )`, false, false},
		{"shared-cache", `(assert (and (or p q) (or (not p) q)))(assert (ite p (or p q) (or (not p) q)))`, `(goals (goal (or p q) (or (not p) q) true :precision precise :depth 1) )`, false, false},
		{"proofs-no-cross-facts", `(assert (= x 3))(assert (= x 4))`, `(goals (goal (= x 3) (= x 4) :precision precise :depth 1) )`, true, false},
		{"dependencies-no-cross-facts", `(assert (! (= x 3) :named a))(assert (! (= x 4) :named b))`, `(goals (goal (= x 3) (= x 4) :precision precise :depth 1) )`, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			prefix := "(set-logic ALL)"
			if tc.proofs {
				prefix += "(set-option :produce-proofs true)"
			}
			if tc.cores {
				prefix += "(set-option :produce-unsat-cores true)"
			}
			script := prefix + ctxDeclarations + tc.assertions
			actual := ctxApplyDisplay(t, script, tc.proofs, tc.cores, nil)
			if ctxCanonical(actual) != ctxCanonical(tc.want) {
				t.Fatal("rewritten goal mismatch", actual, tc.want)
			}
			if os.Getenv("G3_CPP_REFERENCE") != "" {
				source := ctxNative(t, script+"(apply ctx-simplify)")
				if ctxCanonical(actual) != ctxCanonical(source) {
					t.Fatal("Go/C++ goal mismatch", actual, source)
				}
			}
		})
	}
}

func TestContextSimplifyDepthAndStepFrontiersCPP(t *testing.T) {
	if os.Getenv("G3_CPP_REFERENCE") == "" {
		t.Skip("set G3_CPP_REFERENCE to prebuilt z3")
	}
	const assertions = `(assert (and (or p q) (or (not p) q)))(assert (= (ite p (ite p x y) (ite p y x)) x))(assert (or (not q) (and q r)))`
	for _, key := range []string{"max_steps", "max_depth"} {
		for n := uint32(0); n <= 32; n++ {
			t.Run(fmt.Sprintf("%s/%d", key, n), func(t *testing.T) {
				script := "(set-logic ALL)" + ctxDeclarations + assertions
				actual := ctxApplyDisplay(t, script, false, false, map[string]uint32{key: n})
				source := ctxNative(t, script+fmt.Sprintf("(apply (using-params ctx-simplify :%s %d))", key, n))
				if ctxCanonical(actual) != ctxCanonical(source) {
					t.Fatal("soft-limit frontier mismatch", actual, source)
				}
			})
		}
	}
}

func TestContextSimplifySharedBooleanDAGsCPP(t *testing.T) {
	if os.Getenv("G3_CPP_REFERENCE") == "" {
		t.Skip("set G3_CPP_REFERENCE to prebuilt z3")
	}
	random := rand.New(rand.NewSource(641))
	atoms := []string{"p", "q", "r", "(= x 0)", "(= y 1)", "(= x y)"}
	var expression func(int) string
	expression = func(depth int) string {
		if depth == 0 || random.Intn(4) == 0 {
			return atoms[random.Intn(len(atoms))]
		}
		switch random.Intn(5) {
		case 0:
			return "(not " + expression(depth-1) + ")"
		case 1:
			return "(and " + expression(depth-1) + " " + expression(depth-1) + ")"
		case 2:
			return "(or " + expression(depth-1) + " " + expression(depth-1) + ")"
		case 3:
			return "(ite " + expression(depth-1) + " " + expression(depth-1) + " " + expression(depth-1) + ")"
		default:
			a := expression(depth - 1)
			return "(and (or p " + a + ") (or (not p) " + a + "))"
		}
	}
	for i := 0; i < 64; i++ {
		assertions := "(assert " + expression(3) + ")(assert " + expression(2) + ")"
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			script := "(set-logic ALL)" + ctxDeclarations + assertions
			actual := ctxApplyDisplay(t, script, false, false, nil)
			source := ctxNative(t, script+"(apply ctx-simplify)")
			if ctxCanonical(actual) != ctxCanonical(source) {
				t.Fatal("scoped DAG mismatch", assertions, actual, source)
			}
		})
	}
}
