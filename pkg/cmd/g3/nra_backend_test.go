package main

import (
	"bytes"
	"fmt"
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
		"(and (= (* x x) 2) (= (/ x (+ x 1)) 0))",
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

func TestSMTLibRunnerUnivariateNRABooleanStructure(t *testing.T) {
	cases := []struct {
		name, formula, want string
	}{
		{"disjunction", "(or (= (* x x) 2) (= x 0))", "sat"},
		{"disjoint-branches-conflict", "(and (or (< x (- 1)) (> x 1)) (= x 0))", "unsat"},
		{"implication", "(=> (= (* x x) 2) (> x 0))", "sat"},
		{"forced-implication-conflict", "(and (= (* x x) 2) (=> (= (* x x) 2) (> x 0)) (< x 0))", "unsat"},
		{"exclusive-or", "(xor (= (* x x) 2) (> x 0))", "sat"},
		{"boolean-equality", "(= (= (* x x) 2) (> x 0))", "sat"},
		{"boolean-distinct", "(distinct (= (* x x) 2) (> x 0))", "sat"},
		{"arithmetic-distinct", "(and (distinct (* x x) 2) (= x 0))", "sat"},
		{"boolean-ite", "(ite (= (* x x) 2) (> x 0) (= x 0))", "sat"},
		{"negated-conjunction", "(and (= (* x x) 2) (not (and (> x 0) (< x 0))))", "sat"},
		{"true-branch", "(or true (= (* x x) 2))", "sat"},
		{"false-branches", "(and (= (* x x) 2) (or false (< (* x x) 0)))", "unsat"},
	}
	canonical := func(s string) string { return strings.Join(strings.Fields(s), " ") }
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			script := "(set-option :produce-models true)(set-logic QF_NRA)(declare-const x Real)(assert " + tc.formula + ")(check-sat)"
			want := tc.want
			if tc.want == "sat" {
				script += "(get-value (" + tc.formula + "))"
				want += " ((" + tc.formula + " true))"
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

func TestSMTLibRunnerUnivariateNRABooleanScopes(t *testing.T) {
	script := `(set-logic QF_NRA)
(declare-const x Real)
(assert (or (= (* x x) 2) (= x 0)))
(check-sat)
(push)
(assert (= x 0))
(check-sat)
(push)
(assert (not (= x 0)))
(check-sat)
(pop)
(check-sat)
(pop)
(assert (not (= x 0)))
(check-sat)
`
	want := "sat sat unsat sat sat"
	canonical := func(s string) string { return strings.Join(strings.Fields(s), " ") }
	var out bytes.Buffer
	if err := run(strings.NewReader(script), &out); err != nil {
		t.Fatal(err, out.String())
	}
	if canonical(out.String()) != want {
		t.Fatalf("Go: %s; want: %s", out.String(), want)
	}
	if reference := os.Getenv("G3_CPP_REFERENCE"); reference != "" {
		cmd := exec.Command(reference, "-in")
		cmd.Stdin = strings.NewReader(script)
		native, err := cmd.CombinedOutput()
		if err != nil || canonical(string(native)) != want {
			t.Fatalf("C++: %s, %v; want: %s", native, err, want)
		}
	}
}

func TestSMTLibRunnerUnivariateNRABooleanDifferential(t *testing.T) {
	base := []string{
		"(= (* x x) 2)",
		"(> x 0)",
		"(< (* x x) 0)",
		"(distinct x 0)",
		"(<= (* x x) 4)",
	}
	var formula func(int, int) string
	formula = func(seed, depth int) string {
		if depth == 0 {
			return base[seed%len(base)]
		}
		a := formula(seed*5+1, depth-1)
		b := formula(seed*7+3, depth-1)
		switch seed % 8 {
		case 0:
			return "(and " + a + " " + b + ")"
		case 1:
			return "(or " + a + " " + b + ")"
		case 2:
			return "(=> " + a + " " + b + ")"
		case 3:
			return "(xor " + a + " " + b + ")"
		case 4:
			return "(= " + a + " " + b + ")"
		case 5:
			return "(distinct " + a + " " + b + ")"
		case 6:
			return "(not " + a + ")"
		default:
			c := formula(seed*11+5, depth-1)
			return "(ite " + a + " " + b + " " + c + ")"
		}
	}
	var script strings.Builder
	script.WriteString("(set-logic QF_NRA)(declare-const x Real)\n")
	const queries = 64
	for i := 0; i < queries; i++ {
		fmt.Fprintf(&script, "(push)(assert %s)(check-sat)(pop)\n", formula(i, 3))
	}
	var out bytes.Buffer
	if err := run(strings.NewReader(script.String()), &out); err != nil {
		t.Fatal(err, out.String())
	}
	statuses := strings.Fields(out.String())
	if len(statuses) != queries {
		t.Fatal("Go response count", len(statuses), out.String())
	}
	for i, status := range statuses {
		if status != "sat" && status != "unsat" {
			t.Fatal("specialized Boolean formula was not decided", i, status)
		}
	}
	if reference := os.Getenv("G3_CPP_REFERENCE"); reference != "" {
		cmd := exec.Command(reference, "-in")
		cmd.Stdin = strings.NewReader(script.String())
		native, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatal(err, string(native))
		}
		want := strings.Fields(string(native))
		if len(want) != len(statuses) {
			t.Fatal("C++ response count", len(want), string(native))
		}
		for i := range statuses {
			if statuses[i] != want[i] {
				t.Fatal("Boolean differential", i, statuses[i], want[i], formula(i, 3))
			}
		}
	}
}
