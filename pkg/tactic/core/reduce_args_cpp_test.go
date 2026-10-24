package core_test

import (
	"bytes"
	"io"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/cmd_context"
	"github.com/glycerine/g3/pkg/parsers/smt2"
	"github.com/glycerine/g3/pkg/tactic"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

func reduceArgsContext(t *testing.T, script string) *cmd_context.Context {
	t.Helper()
	c, err := cmd_context.NewStandaloneContext()
	ctxMust(t, err)
	var budget *arena.Budget
	t.Cleanup(func() {
		ctxMust(t, c.Close())
		if budget != nil && budget.Used() != 0 {
			t.Errorf("reduce-args retained %d bytes", budget.Used())
		}
	})
	ctxMust(t, smt2.ParseCommands(c, strings.NewReader(script), io.Discard))
	m, err := c.Manager()
	ctxMust(t, err)
	budget = m.Budget()
	g, err := c.GlobalParameters()
	ctxMust(t, err)
	previous, err := uparams.BindProcessGlobalParamsService(g)
	ctxMust(t, err)
	t.Cleanup(func() { _, err := uparams.BindProcessGlobalParamsService(previous); ctxMust(t, err) })
	return c
}
func reduceArgsCanonical(s string) string {
	names := make(map[string]string)
	s = regexp.MustCompile(`[A-Za-z_][A-Za-z_0-9]*![0-9]+`).ReplaceAllStringFunc(s, func(name string) string {
		if replacement, ok := names[name]; ok {
			return replacement
		}
		replacement := string(rune('A' + len(names)))
		names[name] = replacement
		return replacement
	})
	return strings.Join(strings.Fields(s), " ")
}
func TestReduceArgsExactGoalsCPP(t *testing.T) {
	for _, tc := range []struct{ name, decls, forms string }{
		{"empty", "", ""},
		{"false", "", `(assert false)`},
		{"constants", `(declare-const x Int)`, `(assert (= x 3))`},
		{"all-values", `(declare-fun f (Int Int) Bool)`, `(assert (f 1 2))(assert (not (f 2 3)))`},
		{"first-values", `(declare-fun f (Int Int) Bool)(declare-const x Int)(declare-const y Int)`, `(assert (f 1 x))(assert (not (f 2 y)))`},
		{"last-values", `(declare-fun f (Int Int) Bool)(declare-const x Int)(declare-const y Int)`, `(assert (f x 1))(assert (not (f y 2)))`},
		{"common-ground", `(declare-fun f (Int Int) Bool)(declare-const a Int)(declare-const x Int)(declare-const y Int)`, `(assert (f a x))(assert (not (f a y)))`},
		{"different-bases", `(declare-fun f (Int Int) Bool)(declare-const a Int)(declare-const b Int)(declare-const x Int)(declare-const y Int)`, `(assert (f a x))(assert (not (f b y)))`},
		{"mixed-value-base", `(declare-fun f (Int Int) Bool)(declare-const a Int)(declare-const x Int)(declare-const y Int)`, `(assert (f 1 x))(assert (not (f a y)))`},
		{"bound-variables", `(declare-fun f (Int Int) Bool)`, `(assert (forall ((x Int)) (and (f 1 x) (f 2 x))))`},
		{"non-candidate", `(declare-fun f (Int Int) Bool)`, `(assert (forall ((x Int)(y Int)) (and (f x y) (f 1 x))))`},
		{"patterns", `(declare-fun f (Int Int) Int)`, `(assert (forall ((x Int)) (! (= (f 1 x) x) :pattern ((f 2 x)))))`},
		{"nested", `(declare-fun f (Int Int) Bool)(declare-fun g (Int) Int)(declare-const x Int)(declare-const y Int)`, `(assert (f (g 1) x))(assert (not (f (g 1) y)))`},
		{"offset-left", `(declare-fun f ((_ BitVec 4) (_ BitVec 4)) Bool)(declare-const x (_ BitVec 4))(declare-const y (_ BitVec 4))(declare-const z (_ BitVec 4))`, `(assert (f (bvadd #x1 x) y))(assert (not (f (bvadd #x2 x) z)))`},
		{"offset-right", `(declare-fun f ((_ BitVec 4) (_ BitVec 4)) Bool)(declare-const x (_ BitVec 4))(declare-const y (_ BitVec 4))(declare-const z (_ BitVec 4))`, `(assert (f (bvadd x #x1) y))(assert (not (f (bvadd x #x2) z)))`},
		{"offset-nary", `(declare-fun f ((_ BitVec 4) (_ BitVec 4)) Bool)(declare-const x (_ BitVec 4))(declare-const y (_ BitVec 4))(declare-const z (_ BitVec 4))`, `(assert (f (bvadd #x1 x y) y))(assert (not (f (bvadd #x2 x y) z)))`},
		{"Boolean-values", `(declare-fun f (Bool Bool) Bool)(declare-const p Bool)(declare-const q Bool)`, `(assert (f true p))(assert (not (f false q)))`},
		{"multi-mask", `(declare-fun f (Int Bool Int) Bool)(declare-const x Int)(declare-const y Int)`, `(assert (f 1 true x))(assert (not (f 2 false y)))`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			script := `(set-logic ALL)` + tc.decls + tc.forms
			c := reduceArgsContext(t, script)
			m, err := c.Manager()
			ctxMust(t, err)
			goal, err := tactic.NewGoal(m, false, true, false)
			ctxMust(t, err)
			t.Cleanup(func() { ctxMust(t, goal.Close()) })
			ctxMust(t, c.AssertGoal(goal))
			r, err := c.NewBuiltinTactic("reduce-args")
			ctxMust(t, err)
			ctxMust(t, r.IncRef())
			t.Cleanup(func() { ctxMust(t, r.DecRef()) })
			var result tactic.TacticResult
			t.Cleanup(func() { ctxMust(t, result.Close()) })
			ctxMust(t, r.Apply(goal, &result))
			if len(result.Goals) != 1 || result.Goals[0] != goal {
				t.Fatal("result does not borrow original goal")
			}
			var output bytes.Buffer
			output.WriteString("(goals\n")
			ctxMust(t, goal.DisplayWithAttributes(&output))
			output.WriteString(")\n")
			if os.Getenv("G3_CPP_REFERENCE") != "" {
				cpp := ctxNative(t, script+`(apply reduce-args)`)
				if reduceArgsCanonical(output.String()) != reduceArgsCanonical(cpp) {
					t.Fatalf("Go: %s\nC++: %s", output.String(), cpp)
				}
			}
		})
	}
}
