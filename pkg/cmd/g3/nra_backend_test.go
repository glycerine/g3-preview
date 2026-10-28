package main

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// Exercise parser, solver selection, exact algebraic values, model evaluation,
// and incremental scopes together. Compare truth values, not witness choices.
func TestSMTLibRunnerUnivariateNRA(t *testing.T) {
	cases := []struct{ name, formula, want string }{
		{"positive-root", "(and (= (* x x) 2) (> x 0))", "sat"},
		{"negative-root", "(and (= (* x x) 2) (< x 0))", "sat"},
		{"no-real-root", "(= (+ (* x x) 1) 0)", "unsat"},
		{"strict-boundary", "(and (<= (* x x) 0) (> x 0))", "unsat"},
		{"closed-boundary", "(and (<= (* x x) 0) (>= x 0))", "sat"},
		{"rational-root", "(and (= (* x x) 0.25) (> x 0))", "sat"},
		{"rational-coefficient", "(and (= (* 0.5 x x) 1) (> x 0))", "sat"},
		{"disequality", "(and (not (= (* x x) 2)) (> x 0))", "sat"},
		{"isolated-excluded", "(and (<= (* x x) 0) (not (= x 0)))", "unsat"},
		{"cubic", "(and (= (* x x x) 2) (> x 0))", "sat"},
		{"constant-contradiction", "(and (= (* x x) 2) (< (- x x) 0))", "unsat"},
	}
	canonical := func(s string) string { return strings.Join(strings.Fields(s), " ") }
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			script := "(set-option :produce-models true)(set-logic QF_NRA)(declare-const x Real)(assert " + tc.formula + ")(check-sat)"
			want := tc.want
			if tc.want == "sat" {
				script += "(get-value (" + tc.formula + "))(push)(assert (< (* x x) 0))(check-sat)(pop)(check-sat)(get-value (" + tc.formula + "))"
				want += " ((" + tc.formula + " true)) unsat sat ((" + tc.formula + " true))"
			}
			var out bytes.Buffer
			if err := run(strings.NewReader(script), &out); err != nil {
				t.Fatal(err, out.String())
			}
			if canonical(out.String()) != canonical(want) {
				t.Fatalf("Go: %s; want: %s", out.String(), want)
			}
			if reference := os.Getenv("G3_CPP_REFERENCE"); reference != "" {
				cmd := exec.Command(reference, "-in")
				cmd.Stdin = strings.NewReader(script)
				native, err := cmd.CombinedOutput()
				if err != nil || canonical(string(native)) != canonical(want) {
					t.Fatalf("C++: %s, %v; want: %s", native, err, want)
				}
			}
		})
	}
}

func TestSMTLibRunnerUnivariateNRAUnsupported(t *testing.T) {
	// The specialization must not interpret an unsupported expression as a new
	// independent polynomial variable or ignore another real constant.
	for _, formula := range []string{
		"(and (= (* x x) 2) (> x 0) (= x y) (< y 0))",
		"(and (= (* x x) 2) (or (> x 0) (< x 0)))",
	} {
		script := "(set-logic QF_NRA)(declare-const x Real)(declare-const y Real)(assert " + formula + ")(check-sat)"
		var out bytes.Buffer
		if err := run(strings.NewReader(script), &out); err != nil {
			t.Fatal(err, out.String())
		}
		status := strings.TrimSpace(out.String())
		if status != "unknown" && status != "unsat" && status != "sat" {
			t.Fatal(out.String())
		}
		if strings.Contains(formula, "(= x y)") && status == "sat" {
			t.Fatal("inconsistent multivariate input accepted", out.String())
		}
	}
}
