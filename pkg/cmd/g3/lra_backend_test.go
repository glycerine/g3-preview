package main

import (
	"bytes"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"testing"
)

func TestSMTLibRunnerConcreteLRASmoke(t *testing.T) {
	script := `(set-logic QF_UFLRA)
(declare-const x Real)
(declare-const y Real)
(assert (>= x 0))
(assert (<= x 4))
(assert (<= y 2))
(assert (>= (+ x y) 5))
(check-sat-using smt)
(get-info :all-statistics)
`
	var out bytes.Buffer
	if err := run(strings.NewReader(script), &out); err != nil {
		t.Fatal(err, out.String())
	}
	if !strings.HasPrefix(out.String(), "sat\n") || !regexp.MustCompile(`:arith-make-feasible\s+[1-9][0-9]*`).MatchString(out.String()) {
		t.Fatal("runner did not use concrete LAR", out.String())
	}
	if ref := os.Getenv("G3_CPP_REFERENCE"); ref != "" {
		cmd := exec.Command(ref, "-in")
		cmd.Stdin = strings.NewReader(script)
		native, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatal(err, string(native))
		}
		if !strings.HasPrefix(string(native), "sat\n") || !regexp.MustCompile(`:arith-make-feasible\s+[1-9][0-9]*`).Match(native) {
			t.Fatal("reference did not use LAR", string(native))
		}
	}
}

func TestSMTLibRunnerConcreteLRAIsIntNormalization(t *testing.T) {
	script := "(set-logic QF_UFLRA)(declare-const x Real)(assert (is_int x))(check-sat-using smt)\n"
	var out bytes.Buffer
	if err := run(strings.NewReader(script), &out); err != nil {
		t.Fatal(err, out.String())
	}
	if strings.TrimSpace(out.String()) != "unknown" {
		t.Fatal("source unsupported integer cast result", out.String())
	}
	if ref := os.Getenv("G3_CPP_REFERENCE"); ref != "" {
		cmd := exec.Command(ref, "-in")
		cmd.Stdin = strings.NewReader(script)
		native, err := cmd.CombinedOutput()
		if err != nil || strings.TrimSpace(string(native)) != "unknown" {
			t.Fatal(err, string(native))
		}
	}
}

func TestSMTLibRunnerConcreteLRAScriptsMatchCPP(t *testing.T) {
	cases := map[string]string{
		"lra_backend_shared.smt2":        "sat (((distinct (f x) (f z)) true) ((distinct x z) true)) unsat sat (((distinct (f x) (f z)) true) ((distinct x z) true))",
		"lra_backend_models_scopes.smt2": "sat (((and (>= x 0) (<= x 4) (<= y 2) (>= (+ x y) 5)) true)) unsat sat",
		"lra_backend_core.smt2":          "unsat (x_nonneg y_nonneg sum_negative)",
		"lra_backend_division.smt2":      "sat (((= (/ x 2) 3) true)) unsat sat",
	}
	canonical := func(s string) string {
		// An unsat core is a set of assumptions; traversal order may differ
		// with AST/column ordering while every required name must be present.
		fields := strings.Fields(strings.NewReplacer("(", " ", ")", " ").Replace(s))
		if len(fields) == 4 && fields[0] == "unsat" {
			sort.Strings(fields[1:])
			return strings.Join(fields, " ")
		}
		return strings.Join(strings.Fields(strings.NewReplacer("(", " ( ", ")", " ) ").Replace(s)), " ")
	}
	for name, want := range cases {
		t.Run(name, func(t *testing.T) {
			script, err := os.ReadFile("test_vectors/" + name)
			if err != nil {
				t.Fatal(err)
			}
			var out bytes.Buffer
			if err = run(bytes.NewReader(script), &out); err != nil {
				t.Fatal(err, out.String())
			}
			if canonical(out.String()) != canonical(want) {
				t.Fatal("Go runner result", out.String(), "want", want)
			}
			if ref := os.Getenv("G3_CPP_REFERENCE"); ref != "" {
				cmd := exec.Command(ref, "-in")
				cmd.Stdin = bytes.NewReader(script)
				var native, diagnostics bytes.Buffer
				cmd.Stdout, cmd.Stderr = &native, &diagnostics
				if err = cmd.Run(); err != nil || diagnostics.Len() != 0 {
					t.Fatal(err, native.String(), diagnostics.String())
				}
				if canonical(native.String()) != canonical(want) {
					t.Fatal("C++ runner result", native.String(), "want", want)
				}
			}
		})
	}
}

func TestSMTLibRunnerConcreteLRADefaultSelection(t *testing.T) {
	for _, logic := range []string{"QF_LRA", "QF_UFLRA"} {
		t.Run(logic, func(t *testing.T) {
			script := "(set-logic " + logic + ")(declare-const x Real)(declare-const y Real)(assert (>= x 0))(assert (<= x 4))(assert (<= y 2))(assert (>= (+ x y) 5))(check-sat)(get-info :all-statistics)\n"
			var out bytes.Buffer
			if err := run(strings.NewReader(script), &out); err != nil {
				t.Fatal(err, out.String())
			}
			if !strings.HasPrefix(out.String(), "sat\n") || !regexp.MustCompile(`:arith-make-feasible\s+[1-9][0-9]*`).MatchString(out.String()) {
				t.Fatal("default runner did not use LAR", out.String())
			}
		})
	}
}
