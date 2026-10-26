package smtcore

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestSMTDiffObjectiveNativeOptimizationCaller(t *testing.T) {
	checkNativeArithmeticObjective(t, "diff_objective_weighted.smt2", `sat
 (objectives ((+ (* 2 x) (* 3 y) 7) 20))
 (((= x 2) true) ((= y 3) true))
 sat (objectives ((+ (* 2 x) (* 3 y) 7) 18))
 sat (objectives ((+ (* 2 x) (* 3 y) 7) 20))`)
	checkNativeArithmeticObjective(t, "diff_objective_unbounded.smt2", `sat (objectives (x oo))`)
	reference := os.Getenv("G3_CPP_REFERENCE")
	if reference == "" {
		return
	}
	script, err := os.ReadFile("test_vectors/diff_objective_strict.smt2")
	astTestMust(t, err)
	cmd := exec.Command(reference, "-in")
	cmd.Stdin = bytes.NewReader(script)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	astTestMust(t, cmd.Run())
	// opt_solver.cpp:396-397 guards its RDL mk_ge dispatch on epsilon==0.
	// The strict supremum reaches the retained unhandled-theory warning and
	// true fallback; preserve that diagnostic rather than hiding stderr.
	if strings.TrimSpace(stderr.String()) != "WARNING: unhandled theory N3smt17theory_diff_logicINS_7rdl_extEEE" {
		t.Fatal("native strict difference diagnostic", stderr.String())
	}
	parentheses := strings.NewReplacer("(", " ( ", ")", " ) ")
	got := strings.Join(strings.Fields(parentheses.Replace(stdout.String())), " ")
	want := strings.Join(strings.Fields(parentheses.Replace(`sat (objectives (x (+ 5.0 (* (- 1.0) epsilon)))) (((< x 5) true))`)), " ")
	if got != want {
		t.Fatal("native strict difference optimum", stdout.String())
	}
}
