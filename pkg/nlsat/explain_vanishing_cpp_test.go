package nlsat

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestExplanationVanishingPrebuiltCPPOracle(t *testing.T) {
	reference := os.Getenv("G3_CPP_REFERENCE")
	if reference == "" {
		t.Skip("set G3_CPP_REFERENCE to prebuilt z3")
	}
	var script strings.Builder
	script.WriteString("(set-logic QF_NRA)\n(declare-const x0 Real)\n(declare-const x1 Real)\n")
	queries := 0
	query := func(formula string) {
		fmt.Fprintf(&script, "(push)(assert %s)(check-sat-using qfnra-nlsat)(pop)\n", formula)
		queries++
	}
	for root := int32(-4); root <= 4; root++ {
		e, r, n := explanationAssumptionFixture(t)
		evaluatorSetFraction(t, e, 0, int64(root), 1)
		terms := []evaluatorTerm{{1, 1, 3}, {1, 2, 2}, {2, 0, 1}, {7, 0, 0}}
		if root != 0 {
			terms = append(terms, evaluatorTerm{-root, 0, 3}, evaluatorTerm{-root * root, 0, 2})
		}
		p := evaluatorPolynomial(t, e, terms...)
		ref := vanishingRef(t, e, p)
		assignmentMust(t, n.collectVanishing(ref))
		result, err := ref.Get()
		assignmentMust(t, err)
		if len(n.output.result) == 0 {
			t.Fatal("no assumptions for actual vanishing coefficients")
		}
		var assumptions, conditions []string
		for _, l := range n.output.result {
			a := explanationLiteralSMT(t, e, r, l)
			assumptions = append(assumptions, a)
			conditions = append(conditions, "(not "+a+")")
		}
		original := explanationPolynomialSMT(t, e, p)
		reduct := explanationPolynomialSMT(t, e, result)
		number := intervalSMTInteger(root)
		independent := "(+ (* (- x0 " + number + ") x1 x1 x1) (* (- (* x0 x0) " + intervalSMTInteger(root*root) + ") x1 x1) (* 2.0 x1) 7.0)"
		query("(not (= " + original + " " + independent + "))")
		query("(not (= " + reduct + " (+ (* 2.0 x1) 7.0)))")
		query("(and " + strings.Join(conditions, " ") + " (not (= " + original + " " + reduct + ")))")
		query("(and (= x0 " + number + ") (or " + strings.Join(assumptions, " ") + "))")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, reference, "-in", "-smt2")
	cmd.Stdin = strings.NewReader(script.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("C++ vanishing oracle %v: %s", err, out)
	}
	answers := strings.Fields(string(out))
	if len(answers) != queries {
		t.Fatalf("C++ answers %d want %d: %s", len(answers), queries, out)
	}
	for i, answer := range answers {
		if answer != "unsat" {
			t.Fatal("vanishing counterexample", i, answer)
		}
	}
}
