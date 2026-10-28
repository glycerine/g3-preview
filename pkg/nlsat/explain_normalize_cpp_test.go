package nlsat

import (
	"context"
	"fmt"
	"github.com/glycerine/g3/pkg/math/polynomial"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestExplanationNormalizePrebuiltCPPOracle(t *testing.T) {
	reference := os.Getenv("G3_CPP_REFERENCE")
	if reference == "" {
		t.Skip("set G3_CPP_REFERENCE to prebuilt z3")
	}
	var script strings.Builder
	script.WriteString("(set-logic QF_NRA)\n(declare-const x0 Real)\n(declare-const x1 Real)\n")
	var expected []string
	query := func(formula, answer string) {
		fmt.Fprintf(&script, "(push)(assert %s)(check-sat-using qfnra-nlsat)(pop)\n", formula)
		expected = append(expected, answer)
	}
	for _, value := range []int32{-3, -2, -1, 0, 1, 2} {
		for _, even := range []bool{false, true} {
			for _, kind := range []AtomKind{AtomEQ, AtomLT, AtomGT} {
				for _, negative := range []bool{false, true} {
					e, r, n := explanationAssumptionFixture(t)
					evaluatorSetFraction(t, e, 0, int64(value), 1)
					terms := []evaluatorTerm{{1, 1, 3}, {1, 1, 1}, {2, 0, 1}, {1, 0, 0}}
					if value != 0 {
						terms = append(terms, evaluatorTerm{-value, 0, 3})
					}
					p := evaluatorPolynomial(t, e, terms...)
					y := evaluatorPolynomial(t, e, evaluatorTerm{1, 1, 0})
					l := normalizationLiteral(t, r, kind, []polynomial.Polynomial{p, y}, []bool{false, even})
					if negative {
						l = l.Negated()
					}
					result, err := n.collectNormalize(l, 1)
					assignmentMust(t, err)
					normalizationOwnLiteral(t, r, result)
					input := explanationLiteralSMT(t, e, r, l)
					output := explanationLiteralSMT(t, e, r, result)
					var assumptions, conditions []string
					for _, a := range n.output.result {
						expr := explanationLiteralSMT(t, e, r, a)
						assumptions = append(assumptions, expr)
						conditions = append(conditions, "(not "+expr+")")
					}
					if len(conditions) == 0 {
						t.Fatal("actual source assumptions absent")
					}
					model := "(= x0 " + intervalSMTInteger(value) + ")"
					query("(and "+model+" (or "+strings.Join(assumptions, " ")+"))", "unsat")
					query("(and "+strings.Join(conditions, " ")+" (not (= "+input+" "+output+")))", "unsat")
					lower := "x0"
					factor := int32(value)
					if even {
						lower = "(* x0 x0)"
						factor = value * value
					}
					polynomial := "(+ (* (- x0 " + intervalSMTInteger(value) + ") x1 x1 x1) (* (+ x0 2.0) x1) 1.0)"
					independent := "(" + evaluatorSMTOperator(kind) + " (* " + polynomial + " " + lower + ") 0.0)"
					simplified := "(" + evaluatorSMTOperator(kind) + " (* (+ (* " + intervalSMTInteger(value+2) + " x1) 1.0) " + intervalSMTInteger(factor) + ") 0.0)"
					if negative {
						independent = "(not " + independent + ")"
						simplified = "(not " + simplified + ")"
					}
					query("(not (= "+input+" "+independent+"))", "unsat")
					query("(and "+model+" (not (= "+output+" "+simplified+")))", "unsat")
				}
			}
		}
	}
	// The source's exact -1 test has a mathematical counterexample for raw -2.
	// Verify the corrected result, and retain the independent source counterexample.
	e, r, n := explanationAssumptionFixture(t)
	evaluatorSetFraction(t, e, 0, -2, 1)
	y := evaluatorPolynomial(t, e, evaluatorTerm{1, 1, 0})
	x := evaluatorPolynomial(t, e, evaluatorTerm{1, 0, 1})
	l := normalizationLiteral(t, r, AtomGT, []polynomial.Polynomial{y, x}, []bool{false, false})
	result, err := n.collectNormalize(l, 1)
	assignmentMust(t, err)
	normalizationOwnLiteral(t, r, result)
	input, output := explanationLiteralSMT(t, e, r, l), explanationLiteralSMT(t, e, r, result)
	query("(and (= x0 (- 2.0)) (= x1 1.0) (not (= "+input+" "+output+")))", "unsat")
	query("(and (= x0 (- 2.0)) (= x1 1.0) (not (= "+input+" (> x1 0.0))))", "sat")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, reference, "-in", "-smt2")
	cmd.Stdin = strings.NewReader(script.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("C++ normalization oracle %v: %s", err, out)
	}
	answers := strings.Fields(string(out))
	if len(answers) != len(expected) {
		t.Fatalf("C++ answers %d want %d: %s", len(answers), len(expected), out)
	}
	for i, a := range answers {
		if a != expected[i] {
			t.Fatal("normalization disagreement", i, a, "want", expected[i])
		}
	}
}
