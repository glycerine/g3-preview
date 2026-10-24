package arith_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/tactic"
	"github.com/glycerine/g3/pkg/tactic/arith"
)

func TestArithmeticFragmentProbesCPP(t *testing.T) {
	if os.Getenv("G3_CPP_REFERENCE") == "" {
		t.Skip("set G3_CPP_REFERENCE to prebuilt z3")
	}
	probes := []struct {
		name string
		kind arith.ArithmeticFragment
	}{
		{"is-qflia", arith.FragmentQFLIA}, {"is-qfauflia", arith.FragmentQFAUFLIA}, {"is-qflra", arith.FragmentQFLRA}, {"is-qflira", arith.FragmentQFLIRA},
		{"is-ilp", arith.FragmentILP}, {"is-qfnia", arith.FragmentQFNIA}, {"is-qfnra", arith.FragmentQFNRA},
		{"is-nia", arith.FragmentNIA}, {"is-nra", arith.FragmentNRA}, {"is-nira", arith.FragmentNIRA}, {"is-lia", arith.FragmentLIA},
		{"is-lra", arith.FragmentLRA}, {"is-lira", arith.FragmentLIRA}, {"is-qfufnra", arith.FragmentQFUFNRA},
	}
	for _, tc := range []struct{ name, declarations, assertions string }{
		{"empty", "", ""},
		{"Boolean", `(declare-const p Bool)`, `(assert p)`},
		{"integer-coefficient-left", `(declare-const x Int)`, `(assert (<= (* 2 x) 4))`},
		{"integer-coefficient-right", `(declare-const x Int)`, `(assert (<= (* x 2) 4))`},
		{"integer-nary-product", `(declare-const x Int)`, `(assert (<= (* 2 3 x) 4))`},
		{"integer-subtraction", `(declare-const x Int)(declare-const y Int)`, `(assert (= (- x y) 0))`},
		{"integer-negative-numeral", `(declare-const x Int)`, `(assert (<= x -3))`},
		{"integer-unary-minus", `(declare-const x Int)`, `(assert (<= x (- 3)))`},
		{"integer-nonlinear", `(declare-const x Int)(declare-const y Int)`, `(assert (= (* x y) 0))`},
		{"real-linear", `(declare-const x Real)`, `(assert (<= (+ x 1.0) 4.0))`},
		{"real-coercion", `(declare-const x Real)`, `(assert (<= (+ x 1) 4))`},
		{"real-nonlinear", `(declare-const x Real)(declare-const y Real)`, `(assert (= (* x y) 0.0))`},
		{"real-nary-product", `(declare-const x Real)(declare-const y Real)(declare-const z Real)`, `(assert (= (* x y z) 0.0))`},
		{"real-division-numeral", `(declare-const x Real)`, `(assert (= (/ x 2.0) 0.0))`},
		{"real-division-symbolic", `(declare-const x Real)(declare-const y Real)`, `(assert (= (/ x y) 0.0))`},
		{"real-power", `(declare-const x Real)`, `(assert (= (^ x 2.0) 4.0))`},
		{"real-power-coerced-exponent", `(declare-const x Real)`, `(assert (= (^ x 2) 4.0))`},
		{"real-is-int", `(declare-const x Real)`, `(assert (is_int x))`},
		{"disequality", `(declare-const x Int)`, `(assert (not (= x 0)))`},
		{"negated-comparison", `(declare-const x Int)`, `(assert (not (<= x 0)))`},
		{"term-ite", `(declare-const x Int)(declare-const y Int)(declare-const p Bool)`, `(assert (= (ite p x y) 0))`},
		{"integer-quantifier", "", `(assert (forall ((x Int)) (<= x 0)))`},
		{"real-quantifier", "", `(assert (exists ((x Real)) (= (* x x) 2.0)))`},
		{"UF-application", `(declare-const x Int)(declare-fun f (Int) Int)`, `(assert (= (f x) 0))`},
		{"array-equality", `(declare-const a (Array Int Int))(declare-const b (Array Int Int))`, `(assert (= a b))`},
		{"array-select", `(declare-const a (Array Int Int))`, `(assert (= (select a 0) 0))`},
		{"mixed-theory-nonlinear", `(declare-sort U 0)(declare-const u U)(declare-const x Real)(declare-const y Real)(declare-fun f (U) U)`, `(assert (= (f u) u))(assert (= (* x y) 0.0))`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			script := "(set-logic ALL)" + tc.declarations + tc.assertions
			c := addBoundsContext(t, script)
			m, err := c.Manager()
			addBoundsMust(t, err)
			globals, err := c.GlobalParameters()
			addBoundsMust(t, err)
			g, err := tactic.NewGoal(m, false, true, false)
			addBoundsMust(t, err)
			t.Cleanup(func() { addBoundsMust(t, g.Close()) })
			addBoundsMust(t, c.AssertGoal(g))
			for _, entry := range probes {
				t.Run(entry.name, func(t *testing.T) {
					p, err := arith.NewArithmeticFragmentProbe(m.Budget(), globals.Context, globals, entry.kind)
					addBoundsMust(t, err)
					t.Cleanup(func() { addBoundsMust(t, p.Close()) })
					value, err := p.Eval(g)
					addBoundsMust(t, err)
					native := addBoundsCPP(t, script+fmt.Sprintf("(apply (if %s fail skip))", entry.name))
					want := strings.Contains(native, "tactic failed: fail tactic")
					if (value != 0) != want {
						t.Fatal("Go/C++ arithmetic probe", value, want, native, script)
					}
				})
			}
		})
	}
}

func TestArithmeticQFUFNRAMetadataGuardsCPP(t *testing.T) {
	for _, mode := range []string{"ordinary", "proofs", "cores"} {
		t.Run(mode, func(t *testing.T) {
			script := "(set-logic ALL)"
			if mode == "proofs" {
				script += "(set-option :produce-proofs true)"
			}
			if mode == "cores" {
				script += "(set-option :produce-unsat-cores true)"
			}
			script += `(declare-const x Real)(declare-const y Real)(assert (= (* x y) 0.0))`
			c := addBoundsContext(t, script)
			m, err := c.Manager()
			addBoundsMust(t, err)
			globals, err := c.GlobalParameters()
			addBoundsMust(t, err)
			g, err := tactic.NewGoal(m, mode == "proofs", true, mode == "cores")
			addBoundsMust(t, err)
			t.Cleanup(func() { addBoundsMust(t, g.Close()) })
			addBoundsMust(t, c.AssertGoal(g))
			p, err := arith.NewArithmeticFragmentProbe(m.Budget(), globals.Context, globals, arith.FragmentQFUFNRA)
			addBoundsMust(t, err)
			t.Cleanup(func() { addBoundsMust(t, p.Close()) })
			value, err := p.Eval(g)
			addBoundsMust(t, err)
			if (value != 0) != (mode == "ordinary") {
				t.Fatal("metadata guard", mode, value)
			}
			if os.Getenv("G3_CPP_REFERENCE") != "" {
				native := addBoundsCPP(t, script+"(apply (if is-qfufnra fail skip))")
				if strings.Contains(native, "tactic failed: fail tactic") != (value != 0) {
					t.Fatal("source metadata route", native, value)
				}
			}
		})
	}
}
