package smtcore

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func checkNativeLRANumeric(t *testing.T) {
	t.Helper()
	reference := os.Getenv("G3_CPP_REFERENCE")
	if reference == "" {
		return
	}
	script, err := os.ReadFile("test_vectors/lra_numeric_objective.smt2")
	astTestMust(t, err)
	cmd := exec.Command(reference, "-in")
	cmd.Stdin = bytes.NewReader(script)
	var out, diagnostics bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &diagnostics
	if err = cmd.Run(); err != nil || diagnostics.Len() != 0 {
		t.Fatal(err, out.String(), diagnostics.String())
	}
	text := out.String()
	start := strings.Index(text, "(:arith-")
	if start < 0 {
		t.Fatal("missing native LRA statistics", text)
	}
	end := strings.IndexByte(text[start:], ')')
	if end < 0 {
		t.Fatal("unterminated native statistics", text)
	}
	end += start + 1
	statistics := text[start:end]
	// These counters belong to theory_lra.cpp, not the substituted Go
	// tableau. Only volatile statistics are removed from the output oracle.
	if !strings.Contains(statistics, ":arith-make-feasible") || !strings.Contains(statistics, ":arith-bound-propagations-lp") {
		t.Fatal("reference did not select LRA", statistics)
	}
	text = text[:start] + text[end:]
	expected := `sat
(objectives ((+ 3.0 (* 2.0 x) (- y)) (+ 9.0 (* (- 2.0) epsilon))))
(((< x 3) true) ((>= x 0) true) ((>= y 0) true) ((<= y 2) true))
sat (objectives ((+ 3.0 (* 2.0 x) (- y)) 7))
sat (objectives ((+ 3.0 (* 2.0 x) (- y)) (+ 9.0 (* (- 2.0) epsilon))))
unsat sat`
	// Preserve parenthesis structure as well as all values and SAT statuses.
	spaces := strings.NewReplacer("(", " ( ", ")", " ) ")
	canonical := func(s string) string { return strings.Join(strings.Fields(spaces.Replace(s)), " ") }
	if canonical(text) != canonical(expected) {
		t.Fatal("native LRA oracle", text, "want", expected)
	}
}

func checkNativeLRACoupledProgression(t *testing.T) {
	t.Helper()
	reference := os.Getenv("G3_CPP_REFERENCE")
	if reference == "" {
		return
	}
	script, err := os.ReadFile("test_vectors/lra_numeric_coupled.smt2")
	astTestMust(t, err)
	cmd := exec.Command(reference, "-in")
	cmd.Stdin = bytes.NewReader(script)
	var out, diagnostics bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &diagnostics
	astTestMust(t, cmd.Run())
	spaces := strings.NewReplacer("(", " ( ", ")", " ) ")
	canonical := func(s string) string { return strings.Join(strings.Fields(spaces.Replace(s)), " ") }
	if canonical(out.String()) != canonical("sat (objectives ((+ 2.0 (* 5.0 x) (* 3.0 y)) 20))") {
		t.Fatal("native coupled optimum", out.String())
	}
	expected := "(optimize:check-sat) (optimize:sat) (optsmt lower bound: 8) (optsmt lower bound: 18) (optsmt lower bound: 20)"
	if canonical(diagnostics.String()) != canonical(expected) {
		t.Fatal("native local-bound progression", diagnostics.String())
	}
}
