package arith_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/ast"
	"github.com/glycerine/g3/pkg/cmd_context"
	"github.com/glycerine/g3/pkg/parsers/smt2"
	"github.com/glycerine/g3/pkg/tactic"
	uparams "github.com/glycerine/g3/pkg/util/params"
)

func cardContext(t *testing.T, script string) *cmd_context.Context {
	t.Helper()
	c, err := cmd_context.NewStandaloneContext()
	addBoundsMust(t, err)
	var budget *arena.Budget
	t.Cleanup(func() {
		addBoundsMust(t, c.Close())
		if budget != nil && budget.Used() != 0 {
			t.Errorf("card2bv context retained %d bytes", budget.Used())
		}
	})
	addBoundsMust(t, smt2.ParseCommands(c, strings.NewReader(script), io.Discard))
	m, err := c.Manager()
	addBoundsMust(t, err)
	budget = m.Budget()
	globals, err := c.GlobalParameters()
	addBoundsMust(t, err)
	previous, err := uparams.BindProcessGlobalParamsService(globals)
	addBoundsMust(t, err)
	t.Cleanup(func() { _, err := uparams.BindProcessGlobalParamsService(previous); addBoundsMust(t, err) })
	return c
}
func cardCanonical(s string) string {
	names := make(map[string]string)
	s = regexp.MustCompile(`(?:sn|k)![0-9]+`).ReplaceAllStringFunc(s, func(name string) string {
		if known, ok := names[name]; ok {
			return known
		}
		fresh := fmt.Sprint("aux", len(names))
		names[name] = fresh
		return fresh
	})
	return strings.Join(strings.Fields(strings.NewReplacer("(", " ( ", ")", " ) ").Replace(s)), " ")
}
func TestCard2BVExactGoalsCPP(t *testing.T) {
	for _, mode := range []string{"plain", "proofs", "cores"} {
		for _, tc := range []struct{ name, form string }{
			{"empty", ""},
			{"inconsistent", `(assert false)`},
			{"Boolean", `(assert (or p (not q)))`},
			{"at-most-small", `(assert ((_ at-most 1) p q))`},
			{"at-most-network", `(assert ((_ at-most 2) p q r s v))`},
			{"at-least-network", `(assert ((_ at-least 3) p q r s v))`},
			{"nested", `(assert (= p ((_ at-most 2) q r s v)))`},
			{"negated", `(assert (not ((_ at-least 3) p q r s v)))`},
			{"weighted-le", `(assert ((_ pble 3 2 4 1 2 4) p q r s v))`},
			{"weighted-ge", `(assert ((_ pbge 3 2 4 1 2 4) p q r s v))`},
			{"weighted-eq", `(assert ((_ pbeq 3 2 4 1 2 4) p q r s v))`},
			{"arithmetic-ites", `(assert (<= (+ (ite p 1 0) (ite q 1 0) (ite r 1 0)) 1))`},
			{"unrelated-arithmetic", `(assert (< i 12))`},
			{"quantified-body", `(assert (forall ((b Bool)) ((_ at-most 1) b p q)))`},
			{"duplicate-inputs", `(assert ((_ at-most 1) p p q q))`},
		} {
			t.Run(tc.name+"/"+mode, func(t *testing.T) {
				options := ""
				if mode == "proofs" {
					options = `(set-option :produce-proofs true)`
				} else if mode == "cores" {
					options = `(set-option :produce-unsat-cores true)`
				}
				script := options + `(set-logic ALL)(declare-const p Bool)(declare-const q Bool)(declare-const r Bool)(declare-const s Bool)(declare-const v Bool)(declare-const i Int)` + tc.form
				c := cardContext(t, script)
				m, err := c.Manager()
				addBoundsMust(t, err)
				g, err := tactic.NewGoal(m, mode == "proofs", true, mode == "cores")
				addBoundsMust(t, err)
				t.Cleanup(func() { addBoundsMust(t, g.Close()) })
				addBoundsMust(t, c.AssertGoal(g))
				r, err := c.NewBuiltinTactic("card2bv")
				addBoundsMust(t, err)
				addBoundsMust(t, r.IncRef())
				t.Cleanup(func() { addBoundsMust(t, r.DecRef()) })
				var result tactic.TacticResult
				t.Cleanup(func() { addBoundsMust(t, result.Close()) })
				err = r.Apply(g, &result)
				// The source non-proof PB traversal leaves new_pr2 empty.
				// Splitting its conjunction with enabled proofs dereferences
				// that null proof in C++ (verified SIGSEGV in all five cases).
				// Keep the Go boundary refusal; do not fabricate a proof or
				// repeatedly crash the native reference in routine tests.
				proofCrash := tc.name == "weighted-le" || tc.name == "weighted-ge" || tc.name == "weighted-eq" || tc.name == "arithmetic-ites" || tc.name == "duplicate-inputs"
				if mode == "proofs" && proofCrash {
					if !errors.Is(err, ast.ErrASTHandle) {
						t.Fatal("source null-proof split refusal", err)
					}
					return
				}
				addBoundsMust(t, err)
				if len(result.Goals) != 1 || result.Goals[0] != g || result.ProofConverter != nil {
					t.Fatal("source result metadata")
				}
				var actual bytes.Buffer
				actual.WriteString("(goals\n")
				addBoundsMust(t, g.DisplayWithAttributes(&actual))
				actual.WriteString(")\n")
				if os.Getenv("G3_CPP_REFERENCE") != "" {
					cpp := addBoundsCPP(t, script+`(apply card2bv)`)
					if cardCanonical(actual.String()) != cardCanonical(cpp) {
						t.Fatalf("Go: %s\nC++: %s", actual.String(), cpp)
					}
				}
			})
		}
	}
}
