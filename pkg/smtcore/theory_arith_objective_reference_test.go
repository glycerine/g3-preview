package smtcore

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// These inputs exercise the retained C++ opt_solver caller of theory_arith's
// callbacks. Go tests invoke the concrete callbacks and check the same optima,
// blockers and scope recovery; the higher Go optimization shell remains open.
func checkNativeArithmeticObjective(t *testing.T, name, expected string) {
	t.Helper()
	reference := os.Getenv("G3_CPP_REFERENCE")
	if reference == "" {
		return
	}
	script, err := os.ReadFile("test_vectors/" + name)
	astTestMust(t, err)
	cmd := exec.Command(reference, "-in")
	cmd.Stdin = bytes.NewReader(script)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err = cmd.Run(); err != nil || stderr.Len() != 0 {
		t.Fatal("native arithmetic objective", err, stdout.String(), stderr.String())
	}
	// Only whitespace is canonicalized. Rational values, epsilon coefficients,
	// objective expressions and every SAT/UNSAT scope result remain exact.
	parentheses := strings.NewReplacer("(", " ( ", ")", " ) ")
	if strings.Join(strings.Fields(parentheses.Replace(stdout.String())), " ") != strings.Join(strings.Fields(parentheses.Replace(expected)), " ") {
		t.Fatal("native arithmetic objective changed", stdout.String(), "want", expected)
	}
}
