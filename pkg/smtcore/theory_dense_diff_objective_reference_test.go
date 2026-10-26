package smtcore

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestSMTDenseObjectiveNativeOptimizationCaller(t *testing.T) {
	checkNativeArithmeticObjective(t, "dense_objective_weighted.smt2", `sat
 (objectives ((+ (* 2 x) (* 3 y) 7) 20))
 (((= x 2) true) ((= y 3) true))
 sat (objectives ((+ (* 2 x) (* 3 y) 7) 18))
 sat (objectives ((+ (* 2 x) (* 3 y) 7) 20))`)
	checkNativeArithmeticObjective(t, "dense_objective_unbounded.smt2", `sat (objectives (x oo))`)
	reference := os.Getenv("G3_CPP_REFERENCE")
	if reference == "" {
		return
	}
	script, err := os.ReadFile("test_vectors/dense_objective_strict.smt2")
	astTestMust(t, err)
	cmd := exec.Command(reference, "-in")
	cmd.Stdin = bytes.NewReader(script)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	astTestMust(t, cmd.Run())
	// The retained dense source explicitly drops epsilon in maximize's model
	// assignment (theory_dense_diff_logic_def.h:1011-1017). The invalid strict
	// witness and opt_solver's nonzero-epsilon dispatch warning are recorded
	// exactly, rather than treating this as a source-correct strict model.
	if strings.TrimSpace(stderr.String()) != "WARNING: unhandled theory N3smt23theory_dense_diff_logicINS_6mi_extEEE" {
		t.Fatal("native dense strict diagnostic", stderr.String())
	}
	parentheses := strings.NewReplacer("(", " ( ", ")", " ) ")
	got := strings.Join(strings.Fields(parentheses.Replace(stdout.String())), " ")
	want := strings.Join(strings.Fields(parentheses.Replace(`sat (objectives (x (+ 5.0 (* (- 1.0) epsilon)))) (((< x 5) false))`)), " ")
	if got != want {
		t.Fatal("native strict dense source behavior changed", stdout.String())
	}
}
