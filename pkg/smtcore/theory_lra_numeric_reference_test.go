package smtcore

import (
	"bytes"
	"os"
	"os/exec"
	"regexp"
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

func TestSMTLRAEqualityNativeScopeOracle(t *testing.T) {
	reference := os.Getenv("G3_CPP_REFERENCE")
	if reference == "" {
		t.Skip("G3_CPP_REFERENCE is not set")
	}
	script, err := os.ReadFile("test_vectors/lra_equality_scope.smt2")
	astTestMust(t, err)
	cmd := exec.Command(reference, "-in")
	cmd.Stdin = bytes.NewReader(script)
	var out, diagnostics bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &diagnostics
	if err = cmd.Run(); err != nil || diagnostics.Len() != 0 {
		t.Fatal(err, out.String(), diagnostics.String())
	}
	spaces := strings.NewReplacer("(", " ( ", ")", " ) ")
	canonical := func(s string) string { return strings.Join(strings.Fields(spaces.Replace(s)), " ") }
	expected := "sat unsat sat sat (((>= x 2) true) ((= y 1) true) ((= (+ x (* (- 1) x) y) y) true)) sat"
	if canonical(out.String()) != canonical(expected) {
		t.Fatal("native LRA equality/scope oracle", out.String(), "want", expected)
	}
}

func TestSMTLRALPPropagationNativeOracle(t *testing.T) {
	reference := os.Getenv("G3_CPP_REFERENCE")
	if reference == "" {
		t.Skip("G3_CPP_REFERENCE is not set")
	}
	script, err := os.ReadFile("test_vectors/lra_row_propagation.smt2")
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
		t.Fatal("missing LRA statistics", text)
	}
	end := strings.IndexByte(text[start:], ')')
	if end < 0 {
		t.Fatal("unterminated LRA statistics", text)
	}
	end += start + 1
	statistics := text[start:end]
	if !regexp.MustCompile(`:arith-bound-propagations-lp\s+[1-9][0-9]*`).MatchString(statistics) {
		t.Fatal("native oracle did not exercise LP propagation", statistics)
	}
	text = text[:start] + text[end:]
	spaces := strings.NewReplacer("(", " ( ", ")", " ) ")
	canonical := func(s string) string { return strings.Join(strings.Fields(spaces.Replace(s)), " ") }
	expected := "sat (((>= x 3) true) ((>= (+ x y) 5) true) ((<= y 2) true)) sat (((> x 3) true)) unsat sat sat"
	if canonical(text) != canonical(expected) {
		t.Fatal("native row propagation oracle", text, "want", expected)
	}
}

func TestSMTLRAFixedEqualityNativeOracle(t *testing.T) {
	reference := os.Getenv("G3_CPP_REFERENCE")
	if reference == "" {
		t.Skip("G3_CPP_REFERENCE is not set")
	}
	script, err := os.ReadFile("test_vectors/lra_fixed_equality.smt2")
	astTestMust(t, err)
	cmd := exec.Command(reference, "-in")
	cmd.Stdin = bytes.NewReader(script)
	var out, diagnostics bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &diagnostics
	if err = cmd.Run(); err != nil || diagnostics.Len() != 0 {
		t.Fatal(err, out.String(), diagnostics.String())
	}
	text := out.String()
	start := strings.Index(text, "(:")
	if start < 0 {
		t.Fatal("missing fixed-equality statistics", text)
	}
	end := strings.IndexByte(text[start:], ')')
	if end < 0 {
		t.Fatal("unterminated fixed-equality statistics", text)
	}
	end += start + 1
	if !regexp.MustCompile(`:arith-fixed-eqs\s+[1-9][0-9]*`).MatchString(text[start:end]) {
		t.Fatal("C++ did not exercise LRA fixed-equality propagation", text)
	}
	text = text[:start] + text[end:]
	spaces := strings.NewReplacer("(", " ( ", ")", " ) ")
	canonical := func(s string) string { return strings.Join(strings.Fields(spaces.Replace(s)), " ") }
	expected := "unsat sat (((= x y) true) ((= (f x) (f y)) true))"
	if canonical(text) != canonical(expected) {
		t.Fatal("native fixed-equality/congruence oracle", text, "want", expected)
	}
}

func TestSMTLRASharedEqualityNativeOracle(t *testing.T) {
	reference := os.Getenv("G3_CPP_REFERENCE")
	if reference == "" {
		t.Skip("G3_CPP_REFERENCE is not set")
	}
	script, err := os.ReadFile("test_vectors/lra_shared_equality.smt2")
	astTestMust(t, err)
	cmd := exec.Command(reference, "-in")
	cmd.Stdin = bytes.NewReader(script)
	var out, diagnostics bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &diagnostics
	if err = cmd.Run(); err != nil || diagnostics.Len() != 0 {
		t.Fatal(err, out.String(), diagnostics.String())
	}
	text := out.String()
	start := strings.Index(text, "(:")
	if start < 0 {
		t.Fatal("missing shared-equality statistics", text)
	}
	end := strings.IndexByte(text[start:], ')')
	if end < 0 {
		t.Fatal("unterminated shared-equality statistics", text)
	}
	end += start + 1
	if !regexp.MustCompile(`:interface-eqs\s+[1-9][0-9]*`).MatchString(text[start:end]) || !regexp.MustCompile(`:arith-eq-adapter\s+[1-9][0-9]*`).MatchString(text[start:end]) {
		t.Fatal("C++ did not exercise shared interface equality", text)
	}
	text = text[:start] + text[end:]
	spaces := strings.NewReplacer("(", " ( ", ")", " ) ")
	canonical := func(s string) string { return strings.Join(strings.Fields(spaces.Replace(s)), " ") }
	expected := "unsat sat (((= x y) true) ((= (f x) (f y)) true))"
	if canonical(text) != canonical(expected) {
		t.Fatal("native shared-equality/congruence oracle", text, "want", expected)
	}
}
