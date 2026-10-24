package bv_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/cmd_context"
	_ "github.com/glycerine/g3/pkg/engine"
	"github.com/glycerine/g3/pkg/parsers/smt2"
	"github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/tactic/bv"
	"github.com/glycerine/g3/pkg/util/memory"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

func bv1Must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
func bv1Context(t *testing.T, script string) *cmd_context.Context {
	t.Helper()
	c, err := cmd_context.NewStandaloneContext()
	bv1Must(t, err)
	var budget *arena.Budget
	t.Cleanup(func() {
		bv1Must(t, c.Close())
		if budget != nil && budget.Used() != 0 {
			t.Errorf("BV1 context retained %d bytes", budget.Used())
		}
	})
	bv1Must(t, smt2.ParseCommands(c, strings.NewReader(script), io.Discard))
	m, err := c.Manager()
	bv1Must(t, err)
	budget = m.Budget()
	globals, err := c.GlobalParameters()
	bv1Must(t, err)
	previous, err := uparams.BindProcessGlobalParamsService(globals)
	bv1Must(t, err)
	t.Cleanup(func() { _, err := uparams.BindProcessGlobalParamsService(previous); bv1Must(t, err) })
	return c
}
func bv1Canonical(s string) string {
	names := make(map[string]string)
	s = regexp.MustCompile(`k![0-9]+`).ReplaceAllStringFunc(s, func(name string) string {
		if n, ok := names[name]; ok {
			return n
		}
		n := fmt.Sprint("bit", len(names))
		names[name] = n
		return n
	})
	return strings.Join(strings.Fields(strings.NewReplacer("(", " ( ", ")", " ) ").Replace(s)), " ")
}

func TestBV1BlasterExactGoalsCPP(t *testing.T) {
	for _, tc := range []struct{ name, decls, forms string }{
		{"empty", "", ""},
		{"one-bit", `(declare-const a (_ BitVec 1))(declare-const b (_ BitVec 1))`, `(assert (= a b))`},
		{"equality", `(declare-const a (_ BitVec 4))(declare-const b (_ BitVec 4))`, `(assert (= a b))`},
		{"numeral", `(declare-const a (_ BitVec 5))`, `(assert (= a #b10110))`},
		{"extract", `(declare-const a (_ BitVec 8))`, `(assert (= ((_ extract 6 2) a) #b10101))`},
		{"concat", `(declare-const a (_ BitVec 3))(declare-const b (_ BitVec 2))`, `(assert (= (concat a b) #b10110))`},
		{"nested-concat", `(declare-const a (_ BitVec 2))(declare-const b (_ BitVec 2))(declare-const c (_ BitVec 2))`, `(assert (= (concat a (concat b c)) #b101101))`},
		{"ite", `(declare-const p Bool)(declare-const a (_ BitVec 3))(declare-const b (_ BitVec 3))`, `(assert (= (ite p a b) #b101))`},
		{"shared", `(declare-const a (_ BitVec 4))(declare-const b (_ BitVec 4))`, `(assert (or (= a b) (= ((_ extract 3 2) a) ((_ extract 1 0) b))))(assert (not (= a #b0011)))`},
		{"foreign-arithmetic", `(declare-const x Int)`, `(assert (= (+ x 1) 3))`},
		{"UF-return", `(declare-const a (_ BitVec 3))(declare-fun f ((_ BitVec 3)) (_ BitVec 3))`, `(assert (= (f a) a))`},
		{"UF-Boolean", `(declare-const a (_ BitVec 3))(declare-fun f ((_ BitVec 3)) Bool)`, `(assert (f a))`},
		{"wide-numeral", `(declare-const a (_ BitVec 65))`, `(assert (= a (_ bv18446744073709551617 65)))`},
	} {
		for _, proofs := range []bool{false, true} {
			t.Run(fmt.Sprint(tc.name, "/", proofs), func(t *testing.T) {
				script := fmt.Sprintf("(set-option :produce-proofs %v)(set-logic ALL)", proofs) + tc.decls + tc.forms
				c := bv1Context(t, script)
				m, err := c.Manager()
				bv1Must(t, err)
				globals, err := c.GlobalParameters()
				bv1Must(t, err)
				g, err := tactic.NewGoal(m, proofs, true, false)
				bv1Must(t, err)
				t.Cleanup(func() { bv1Must(t, g.Close()) })
				bv1Must(t, c.AssertGoal(g))
				r, err := bv.NewBV1BlasterTactic(m, globals.Context, globals, nil, nil)
				bv1Must(t, err)
				closed := false
				t.Cleanup(func() {
					if !closed {
						bv1Must(t, r.Close())
					}
				})
				var result tactic.TacticResult
				t.Cleanup(func() { bv1Must(t, result.Close()) })
				bv1Must(t, r.Apply(g, &result))
				if result.ModelConverter == nil || len(result.Goals) != 1 || result.Goals[0] != g {
					t.Fatal("missing unconditional model converter or borrowed goal")
				}
				bv1Must(t, r.Close())
				closed = true
				if proofs {
					n, err := g.Size()
					bv1Must(t, err)
					for i := uint32(0); i < n; i++ {
						h, err := g.Form(i)
						bv1Must(t, err)
						pr, err := g.Proof(i)
						bv1Must(t, err)
						fact, err := m.GetProofFact(pr)
						bv1Must(t, err)
						if fact != h {
							t.Fatal("rewritten proof lost its fact")
						}
					}
				}
				var actual bytes.Buffer
				actual.WriteString("(goals\n")
				bv1Must(t, g.DisplayWithAttributes(&actual))
				actual.WriteString(")\n")
				if reference := os.Getenv("G3_CPP_REFERENCE"); reference != "" {
					cmd := exec.Command(reference, "-in")
					cmd.Stdin = strings.NewReader(script + `(apply bv1-blast)`)
					cpp, err := cmd.CombinedOutput()
					if err != nil {
						t.Fatal(err, string(cpp))
					}
					if bv1Canonical(actual.String()) != bv1Canonical(string(cpp)) {
						t.Fatalf("Go: %s\nC++: %s", actual.String(), cpp)
					}
				}
			})
		}
	}
}

func TestBV1EqualityProbeAndRefusalsCPP(t *testing.T) {
	for _, tc := range []struct {
		name, decls, forms string
		want               bool
	}{
		{"Boolean", `(declare-const p Bool)`, `(assert p)`, true},
		{"equality", `(declare-const a (_ BitVec 4))(declare-const b (_ BitVec 4))`, `(assert (= a b))`, true},
		{"UF", `(declare-const a (_ BitVec 3))(declare-fun f ((_ BitVec 3)) (_ BitVec 3))`, `(assert (= (f a) a))`, true},
		{"arithmetic", `(declare-const x Int)`, `(assert (= (+ x 1) 4))`, true},
		{"xor", `(declare-const a (_ BitVec 3))(declare-const b (_ BitVec 3))`, `(assert (= (bvxor a b) a))`, false},
		{"addition", `(declare-const a (_ BitVec 3))`, `(assert (= (bvadd a a) a))`, false},
		{"quantifier", "", `(assert (forall ((a (_ BitVec 3))) (= a a)))`, false},
		{"bound-variable", "", `(assert (forall ((p Bool)) p))`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			script := `(set-logic ALL)` + tc.decls + tc.forms
			c := bv1Context(t, script)
			m, err := c.Manager()
			bv1Must(t, err)
			globals, err := c.GlobalParameters()
			bv1Must(t, err)
			goal, err := tactic.NewGoal(m, false, false, false)
			bv1Must(t, err)
			t.Cleanup(func() { bv1Must(t, goal.Close()) })
			bv1Must(t, c.AssertGoal(goal))
			probe, err := bv.NewIsQFBVEqualityProbe(m.Budget(), globals.Context, globals)
			bv1Must(t, err)
			t.Cleanup(func() { bv1Must(t, probe.Close()) })
			value, err := probe.Eval(goal)
			bv1Must(t, err)
			if (value != 0) != tc.want {
				t.Fatal("target classifier", value, tc.want)
			}
			r, err := bv.NewBV1BlasterTactic(m, globals.Context, globals, nil, nil)
			bv1Must(t, err)
			t.Cleanup(func() { bv1Must(t, r.Close()) })
			var result tactic.TacticResult
			t.Cleanup(func() { bv1Must(t, result.Close()) })
			err = r.Apply(goal, &result)
			if tc.want {
				bv1Must(t, err)
			} else {
				var refusal tactic.TacticException
				if !errors.As(err, &refusal) || err.Error() != "bv1 blaster cannot be applied to goal" {
					t.Fatal(err)
				}
				depth, e := goal.Depth()
				bv1Must(t, e)
				if depth != 0 || len(result.Goals) != 0 {
					t.Fatal("refusal published a goal")
				}
			}
			if reference := os.Getenv("G3_CPP_REFERENCE"); reference != "" {
				cmd := exec.Command(reference, "-in")
				cmd.Stdin = strings.NewReader(script + `(apply (if is-qfbv-eq fail skip))`)
				cpp, err := cmd.CombinedOutput()
				if err != nil && !strings.Contains(string(cpp), "tactic failed: fail tactic") {
					t.Fatal(err, string(cpp))
				}
				if strings.Contains(string(cpp), "tactic failed: fail tactic") != tc.want {
					t.Fatal("source probe", string(cpp))
				}
			}
		})
	}
}

func TestBV1BlasterStepFrontiersCPP(t *testing.T) {
	reference := os.Getenv("G3_CPP_REFERENCE")
	if reference == "" {
		t.Skip("set G3_CPP_REFERENCE to prebuilt z3")
	}
	for _, proofs := range []bool{false, true} {
		for limit := 0; limit <= 32; limit++ {
			t.Run(fmt.Sprint(proofs, "/", limit), func(t *testing.T) {
				script := fmt.Sprintf(`(set-logic QF_BV)(set-option :produce-proofs %v)
(declare-const a (_ BitVec 8))(declare-const b (_ BitVec 8))(declare-const p Bool)
(assert (or (= ((_ extract 3 0) (ite p a b)) #xa) (= ((_ extract 7 4) a) #xb)))
(check-sat-using (using-params bv1-blast :max_steps %d))(get-info :reason-unknown)`, proofs, limit)
				at := strings.Index(script, "(check-sat-using")
				c := bv1Context(t, script[:at])
				var actual bytes.Buffer
				bv1Must(t, smt2.ParseCommands(c, strings.NewReader(script[at:]), &actual))
				cmd := exec.Command(reference, "-in")
				cmd.Stdin = strings.NewReader(script)
				cpp, err := cmd.CombinedOutput()
				if err != nil {
					t.Fatal(err, string(cpp))
				}
				if bv1Canonical(actual.String()) != bv1Canonical(string(cpp)) {
					t.Fatalf("Go: %s\nC++: %s", actual.String(), cpp)
				}
			})
		}
	}
}

func TestBV1BlasterModelConverterSurvivesTacticClose(t *testing.T) {
	c := bv1Context(t, `(set-logic QF_BV)(declare-const x (_ BitVec 8))(assert (= x #xa5))`)
	m, err := c.Manager()
	bv1Must(t, err)
	globals, err := c.GlobalParameters()
	bv1Must(t, err)
	goal, err := tactic.NewGoal(m, false, true, false)
	bv1Must(t, err)
	t.Cleanup(func() { bv1Must(t, goal.Close()) })
	bv1Must(t, c.AssertGoal(goal))
	r, err := bv.NewBV1BlasterTactic(m, globals.Context, globals, nil, nil)
	bv1Must(t, err)
	closed := false
	t.Cleanup(func() {
		if !closed {
			bv1Must(t, r.Close())
		}
	})
	var first tactic.TacticResult
	t.Cleanup(func() { bv1Must(t, first.Close()) })
	bv1Must(t, r.Apply(goal, &first))
	bv1Must(t, r.Close())
	closed = true
	smt, err := c.NewBuiltinTactic("smt")
	bv1Must(t, err)
	bv1Must(t, smt.IncRef())
	t.Cleanup(func() { bv1Must(t, smt.DecRef()) })
	var second tactic.TacticResult
	t.Cleanup(func() { bv1Must(t, second.Close()) })
	bv1Must(t, smt.Apply(goal, &second))
	if !second.DecidedSat() {
		t.Fatal("rewritten one-bit constraints did not solve")
	}
	bv1Must(t, smt.Cleanup())
	var md tactic.ModelRef
	t.Cleanup(func() { bv1Must(t, md.Close()) })
	bv1Must(t, tactic.ModelConverterToModel(m, second.ModelConverter, &md))
	bv1Must(t, first.ModelConverter.ApplyUnindexed(&md))
	model, err := md.Get()
	bv1Must(t, err)
	x, err := c.MakeApp("x", nil, nil, ast.ASTHandle{})
	bv1Must(t, err)
	d, err := m.GetDecl(x)
	bv1Must(t, err)
	value, err := model.ConstantInterpretation(d)
	bv1Must(t, err)
	text, yes, err := m.BitVectorNumeralText(value)
	bv1Must(t, err)
	if !yes || text != "165" {
		t.Fatal("model reconstruction after original tactic destruction", text)
	}
}

func TestBV1BlasterConstructorAllocationFrontiers(t *testing.T) {
	success := false
	for stop := int64(0); stop < 512; stop++ {
		t.Run(fmt.Sprint(stop), func(t *testing.T) {
			c := bv1Context(t, `(set-logic QF_BV)`)
			m, err := c.Manager()
			bv1Must(t, err)
			globals, err := c.GlobalParameters()
			bv1Must(t, err)
			m.Budget().FailAfter(stop)
			r, err := bv.NewBV1BlasterTactic(m, globals.Context, globals, nil, nil)
			m.Budget().FailAfter(-1)
			if r != nil {
				bv1Must(t, r.Close())
			}
			var failed memory.ConstructionFailure
			if errors.As(err, &failed) {
				bv1Must(t, failed.Cleanup())
			}
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
		t.Fatal("no successful constructor frontier")
	}
}
