package ackermannization_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/ackermannization"
	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/cmd_context"
	_ "github.com/glycerine/g3/pkg/engine"
	"github.com/glycerine/g3/pkg/parsers/smt2"
	"github.com/glycerine/g3/pkg/tactic"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

func ackMust(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
func ackContext(t *testing.T, script string) *cmd_context.Context {
	t.Helper()
	c, err := cmd_context.NewStandaloneContext()
	ackMust(t, err)
	var budget *arena.Budget
	t.Cleanup(func() {
		ackMust(t, c.Close())
		if budget != nil && budget.Used() != 0 {
			t.Errorf("ackermannization retained %d bytes", budget.Used())
		}
	})
	ackMust(t, smt2.ParseCommands(c, strings.NewReader(script), io.Discard))
	m, err := c.Manager()
	ackMust(t, err)
	budget = m.Budget()
	globals, err := c.GlobalParameters()
	ackMust(t, err)
	previous, err := uparams.BindProcessGlobalParamsService(globals)
	ackMust(t, err)
	t.Cleanup(func() { _, err := uparams.BindProcessGlobalParamsService(previous); ackMust(t, err) })
	return c
}
func ackCPP(t *testing.T, script string) string {
	t.Helper()
	cmd := exec.Command(os.Getenv("G3_CPP_REFERENCE"), "-in")
	cmd.Stdin = strings.NewReader(script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err, string(out))
	}
	return string(out)
}

type ackExpr struct {
	atom     string
	children []*ackExpr
}

func ackParse(t *testing.T, s string) *ackExpr {
	t.Helper()
	tokens := strings.Fields(strings.NewReplacer("(", " ( ", ")", " ) ").Replace(s))
	index := 0
	var parse func() *ackExpr
	parse = func() *ackExpr {
		if index >= len(tokens) {
			t.Fatal("truncated goal", s)
		}
		token := tokens[index]
		index++
		if token != "(" {
			return &ackExpr{atom: token}
		}
		n := &ackExpr{}
		for index < len(tokens) && tokens[index] != ")" {
			n.children = append(n.children, parse())
		}
		if index >= len(tokens) {
			t.Fatal("missing goal terminator", s)
		}
		index++
		return n
	}
	result := parse()
	if index != len(tokens) {
		t.Fatal("extra goal output", s)
	}
	return result
}
func (e *ackExpr) canonical() string {
	if len(e.children) == 0 {
		return e.atom
	}
	// Boolean equality rewriting can put a single negation on either
	// operand, depending on the source hash-table pair enumeration.
	if len(e.children) == 3 && e.children[0].atom == "=" {
		negated := false
		args := make([]string, 2)
		for i, child := range e.children[1:] {
			if len(child.children) == 2 && child.children[0].atom == "not" {
				negated = !negated
				child = child.children[1]
			}
			args[i] = child.canonical()
		}
		sort.Strings(args)
		result := "(= " + strings.Join(args, " ") + ")"
		if negated {
			result = "(not " + result + ")"
		}
		return result
	}
	parts := make([]string, len(e.children))
	for i, c := range e.children {
		parts[i] = c.canonical()
	}
	if parts[0] == "=" || parts[0] == "and" || parts[0] == "or" {
		sort.Strings(parts[1:])
	}
	return "(" + strings.Join(parts, " ") + ")"
}
func ackCanonical(t *testing.T, s string) string {
	t.Helper()
	names := make(map[string]string)
	s = regexp.MustCompile(`[A-Za-z_][A-Za-z_0-9]*![0-9]+`).ReplaceAllStringFunc(s, func(name string) string {
		if n, ok := names[name]; ok {
			return n
		}
		n := fmt.Sprint("fresh", len(names))
		names[name] = n
		return n
	})
	root := ackParse(t, s)
	if len(root.children) != 2 || root.children[0].atom != "goals" {
		t.Fatal("not one goal", s)
	}
	goal := root.children[1]
	var formulas, attributes []string
	for i, c := range goal.children {
		if i == 0 {
			if c.atom != "goal" {
				t.Fatal("missing goal head")
			}
			continue
		}
		if strings.HasPrefix(c.atom, ":") || len(attributes) > 0 {
			attributes = append(attributes, c.canonical())
		} else {
			formulas = append(formulas, c.canonical())
		}
	}
	sort.Strings(formulas)
	return strings.Join(formulas, "\n") + "\n" + strings.Join(attributes, " ")
}

func TestAckermannizationExactGoalsCPP(t *testing.T) {
	for _, tc := range []struct{ name, decls, forms string }{
		{"empty", "", ""},
		{"already-false", "", `(assert false)`},
		{"no-functions", `(declare-const x (_ BitVec 3))`, `(assert (= x #b001))`},
		{"one-occurrence", `(declare-const x (_ BitVec 3))(declare-fun f ((_ BitVec 3)) (_ BitVec 3))`, `(assert (= (f x) #b010))`},
		{"congruence", `(declare-const x (_ BitVec 3))(declare-const y (_ BitVec 3))(declare-fun f ((_ BitVec 3)) (_ BitVec 3))`, `(assert (= x y))(assert (not (= (f x) (f y))))`},
		{"different-numerals", `(declare-fun f ((_ BitVec 3)) (_ BitVec 3))`, `(assert (not (= (f #b000) (f #b001))))`},
		{"three-terms", `(declare-const x (_ BitVec 3))(declare-const y (_ BitVec 3))(declare-const z (_ BitVec 3))(declare-fun f ((_ BitVec 3)) (_ BitVec 3))`, `(assert (distinct (f x) (f y) (f z)))`},
		{"two-arguments", `(declare-const x (_ BitVec 3))(declare-const y (_ BitVec 3))(declare-fun f ((_ BitVec 3) (_ BitVec 3)) Bool)`, `(assert (f x #b000))(assert (not (f y #b000)))`},
		{"nested", `(declare-const x (_ BitVec 3))(declare-const y (_ BitVec 3))(declare-fun f ((_ BitVec 3)) (_ BitVec 3))(declare-fun g ((_ BitVec 3)) (_ BitVec 3))`, `(assert (= (f (g x)) (f (g y))))`},
		{"Boolean-functions", `(declare-const p Bool)(declare-const q Bool)(declare-fun f (Bool) Bool)`, `(assert (= (f p) (f q)))`},
		{"integer-functions", `(declare-const x Int)(declare-const y Int)(declare-fun f (Int) Int)`, `(assert (= (f x) (f y)))`},
		{"quantifier", `(declare-fun f ((_ BitVec 3)) (_ BitVec 3))`, `(assert (forall ((x (_ BitVec 3))) (= (f x) x)))`},
	} {
		for _, limit := range []uint32{0, 1, 3, 1000} {
			t.Run(fmt.Sprint(tc.name, "/", limit), func(t *testing.T) {
				script := `(set-logic ALL)` + tc.decls + tc.forms
				c := ackContext(t, script)
				m, err := c.Manager()
				ackMust(t, err)
				g, err := tactic.NewGoal(m, false, true, false)
				ackMust(t, err)
				t.Cleanup(func() { ackMust(t, g.Close()) })
				ackMust(t, c.AssertGoal(g))
				r, err := c.NewBuiltinTactic("ackermannize_bv")
				ackMust(t, err)
				ackMust(t, r.IncRef())
				t.Cleanup(func() { ackMust(t, r.DecRef()) })
				globals, err := c.GlobalParameters()
				ackMust(t, err)
				p, err := globals.Context.NewValue()
				ackMust(t, err)
				t.Cleanup(func() { ackMust(t, globals.Context.Del(&p)) })
				key := "div0_ackermann_limit"
				ackMust(t, globals.Context.SetUIntName(&p, &key, limit))
				ackMust(t, r.UpdateParams(&p))
				var result tactic.TacticResult
				t.Cleanup(func() { ackMust(t, result.Close()) })
				ackMust(t, r.Apply(g, &result))
				if len(result.Goals) != 1 {
					t.Fatal("missing single result goal")
				}
				var actual bytes.Buffer
				actual.WriteString("(goals\n")
				ackMust(t, result.Goals[0].DisplayWithAttributes(&actual))
				actual.WriteString(")\n")
				if os.Getenv("G3_CPP_REFERENCE") != "" {
					cpp := ackCPP(t, script+fmt.Sprintf(`(apply (using-params ackermannize_bv :div0_ackermann_limit %d))`, limit))
					if ackCanonical(t, actual.String()) != ackCanonical(t, cpp) {
						t.Fatalf("Go: %s\nC++: %s", actual.String(), cpp)
					}
				}
			})
		}
	}
}

func TestAckermannBoundProbeCPP(t *testing.T) {
	for _, tc := range []struct {
		name, decls, forms string
		want               float64
	}{
		{"empty", "", "", 0},
		{"constants", `(declare-const x Int)`, `(assert (= x 1))`, 0},
		{"duplicates", `(declare-const x Int)(declare-fun f (Int) Int)`, `(assert (= (f x) (f x)))`, 0},
		{"three", `(declare-const x Int)(declare-const y Int)(declare-const z Int)(declare-fun f (Int) Int)`, `(assert (distinct (f x) (f y) (f z)))`, 3},
		{"quantifier-body", `(declare-fun f (Int) Int)`, `(assert (forall ((x Int)) (= (f x) (f 0))))`, 1},
		{"patterns-ignored", `(declare-fun f (Int) Int)(declare-fun g (Int) Int)`, `(assert (forall ((x Int)) (! (= (f x) 0) :pattern ((g (g x))))))`, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			script := `(set-logic ALL)` + tc.decls + tc.forms
			c := ackContext(t, script)
			m, err := c.Manager()
			ackMust(t, err)
			g, err := tactic.NewGoal(m, false, false, false)
			ackMust(t, err)
			t.Cleanup(func() { ackMust(t, g.Close()) })
			ackMust(t, c.AssertGoal(g))
			probe, err := ackermannization.NewBoundProbe(m.Budget())
			ackMust(t, err)
			t.Cleanup(func() { ackMust(t, probe.Close()) })
			value, err := probe.Eval(g)
			ackMust(t, err)
			if value != tc.want {
				t.Fatal(value, tc.want)
			}
			if os.Getenv("G3_CPP_REFERENCE") != "" {
				cpp := ackCPP(t, script+fmt.Sprintf(`(apply (if (= ackr-bound-probe %.0f) skip fail))`, tc.want))
				if strings.Contains(cpp, "error") {
					t.Fatal(cpp)
				}
			}
		})
	}
}
