package nlsat

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/math/polynomial"
)

func TestInequalityFactoryCanonicalCubicPrebuiltCPPOracle(t *testing.T) {
	reference := os.Getenv("G3_CPP_REFERENCE")
	if reference == "" {
		t.Skip("set G3_CPP_REFERENCE to prebuilt z3")
	}
	e, c := inequalityCacheFixture(t)
	p := evaluatorPolynomial(t, e, evaluatorTerm{-2, 0, 3}, evaluatorTerm{4, 1, 0})
	q := evaluatorPolynomial(t, e, evaluatorTerm{-1, 0, 1}, evaluatorTerm{1, 0, 0})
	var script strings.Builder
	script.WriteString("(declare-const x Real)\n")
	count := 0
	for _, y := range []int{0, 2} {
		evaluatorSetFraction(t, e, 0, int64(y), 1)
		for n := 1; n <= 2; n++ {
			original := "(+ (* (- 2.0) x x x) " + intervalSMTInteger(int32(4*y)) + ")"
			if n == 2 {
				original = "(* " + original + " (- 1.0 x))"
			}
			for _, k := range []AtomKind{AtomEQ, AtomLT, AtomGT} {
				a, _, err := c.mk(k, []polynomial.Polynomial{p, q}[:n], make([]bool, n))
				assignmentMust(t, err)
				a.boolVar = 17
				literal := "(" + evaluatorSMTOperator(k) + " " + original + " 0.0)"
				for _, neg := range []bool{false, true} {
					evaluatorOracleQueries(t, e, a, neg, literal, &script)
					count += 2
				}
			}
		}
	}
	cmd := exec.Command(reference, "-in", "-T:30")
	cmd.Stdin = strings.NewReader(script.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err, string(out))
	}
	statuses := strings.Fields(string(out))
	if len(statuses) != count {
		t.Fatal("C++ error/surplus output", len(statuses), count, string(out))
	}
	for i, status := range statuses {
		if status != "unsat" {
			t.Fatal("canonical factory disagreement", i, status)
		}
	}
	t.Logf("prebuilt C++ rejects all %d canonical cubic projection/value disagreements", count)
}
