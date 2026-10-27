package bv_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/parsers/smt2"
	"github.com/glycerine/g3/pkg/tactic"
)

func TestDT2BVGoalsCPP(t *testing.T) {
	for _, tc := range []struct{ name, decls, forms string }{
		{"empty", "", ""},
		{"three", `(declare-datatypes () ((E red amber green)))(declare-const x E)`, `(assert (= x green))`},
		{"two", `(declare-datatypes () ((E red green)))(declare-const x E)`, `(assert (= x green))`},
		{"recognizer", `(declare-datatypes () ((E red amber green)))(declare-const x E)`, `(assert ((_ is green) x))`},
		{"distinct", `(declare-datatypes () ((E red amber green)))(declare-const x E)(declare-const y E)`, `(assert (distinct x y red))`},
		{"nested", `(declare-datatypes () ((E red amber green)))(declare-const x E)`, `(assert (not (= x green)))`},
		{"unsupported-argument", `(declare-datatypes () ((E red amber green)))(declare-const x E)(declare-fun p (E) Bool)`, `(assert (= x green))(assert (p x))`},
		{"unsupported-result", `(declare-datatypes () ((E red amber green)))(declare-fun f (Bool) E)(declare-const x E)`, `(assert (= x green))(assert (= (f true) x))`},
		{"mixed-sorts", `(declare-datatypes () ((E red amber green) (F cold hot)))(declare-const x E)(declare-const y F)(declare-fun p (E) Bool)`, `(assert (= x green))(assert (p x))(assert (= y hot))`},
		{"forall", `(declare-datatypes () ((E red amber green)))`, `(assert (forall ((x E)) (or (= x red) (= x amber) (= x green))))`},
		{"exists", `(declare-datatypes () ((E red amber green)))`, `(assert (exists ((x E)) (= x green)))`},
		{"nonenum", `(declare-datatypes () ((D (mk (val Int)))))(declare-const x D)`, `(assert (= (val x) 3))`},
	} {
		for _, proofs := range []bool{false, true} {
			t.Run(fmt.Sprint(tc.name, "/", proofs), func(t *testing.T) {
				script := fmt.Sprintf("(set-option :produce-proofs %v)(set-logic ALL)", proofs) + tc.decls + tc.forms
				c := bv1Context(t, script)
				m, err := c.Manager()
				bv1Must(t, err)
				g, err := tactic.NewGoal(m, proofs, true, false)
				bv1Must(t, err)
				t.Cleanup(func() { bv1Must(t, g.Close()) })
				bv1Must(t, c.AssertGoal(g))
				r, err := c.NewBuiltinTactic("dt2bv")
				bv1Must(t, err)
				bv1Must(t, r.IncRef())
				closed := false
				t.Cleanup(func() {
					if !closed {
						bv1Must(t, r.DecRef())
					}
				})
				var result tactic.TacticResult
				t.Cleanup(func() { bv1Must(t, result.Close()) })
				bv1Must(t, r.Apply(g, &result))
				if len(result.Goals) != 1 || result.Goals[0] != g {
					t.Fatal("missing source goal")
				}
				bv1Must(t, r.DecRef())
				closed = true
				var actual bytes.Buffer
				actual.WriteString("(goals\n")
				bv1Must(t, g.DisplayWithAttributes(&actual))
				actual.WriteString(")\n")
				if reference := os.Getenv("G3_CPP_REFERENCE"); reference != "" {
					cmd := exec.Command(reference, "-in")
					cmd.Stdin = strings.NewReader(script + `(apply dt2bv)`)
					cpp, err := cmd.CombinedOutput()
					if err != nil {
						t.Fatal(err, string(cpp))
					}
					// C++'s command ast_printer uses internal BV sort names in
					// quantified binders. Normalize this known display-only difference.
					expected := regexp.MustCompile(`\(_ bv ([0-9]+)\)`).ReplaceAllString(string(cpp), "(_ BitVec $1)")
					if bv1Canonical(actual.String()) != bv1Canonical(expected) {
						t.Fatalf("Go: %s\nC++: %s", actual.String(), cpp)
					}
				}
			})
		}
	}
}

func TestDT2BVSMTLIBModelsAndCoresCPP(t *testing.T) {
	for _, tc := range []struct{ name, forms, commands string }{
		{"model", `(assert (= x green))`, `(check-sat-using (then dt2bv smt))(get-value (x))`},
		{"recognizer-model", `(assert ((_ is amber) x))`, `(check-sat-using (then dt2bv smt))(get-value (x))`},
		{"unsat-core", `(assert (! (= x red) :named first))(assert (! (= x green) :named second))`, `(check-sat-using (then dt2bv smt))(get-unsat-core)`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			script := `(set-option :produce-models true)(set-option :produce-unsat-cores true)(set-logic ALL)(declare-datatypes () ((E red amber green)))(declare-const x E)` + tc.forms
			c := bv1Context(t, script)
			var actual bytes.Buffer
			bv1Must(t, smt2.ParseCommands(c, strings.NewReader(tc.commands), &actual))
			switch tc.name {
			case "model":
				if bv1Canonical(actual.String()) != bv1Canonical("sat ((x green))") {
					t.Fatal(actual.String())
				}
			case "recognizer-model":
				if bv1Canonical(actual.String()) != bv1Canonical("sat ((x amber))") {
					t.Fatal(actual.String())
				}
			case "unsat-core":
				if !strings.HasPrefix(actual.String(), "unsat") || !strings.Contains(actual.String(), "first") || !strings.Contains(actual.String(), "second") {
					t.Fatal(actual.String())
				}
			}
			if reference := os.Getenv("G3_CPP_REFERENCE"); reference != "" {
				cmd := exec.Command(reference, "-in")
				cmd.Stdin = strings.NewReader(script + tc.commands)
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
