package nlsat

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/math/polynomial"
)

func TestAtomFactoryRegisteredRootsAndInequalitiesPrebuiltCPPOracle(t *testing.T) {
	reference := os.Getenv("G3_CPP_REFERENCE")
	if reference == "" {
		t.Skip("set G3_CPP_REFERENCE to prebuilt z3")
	}
	e, f := atomFactoryFixture(t)
	_, err := f.mkBoolVar() // The real parent reserves zero for its true clause.
	assignmentMust(t, err)
	p := evaluatorPolynomial(t, e, evaluatorTerm{-1, 0, 2}, evaluatorTerm{1, 1, 0})
	var script strings.Builder
	script.WriteString("(declare-const x Real)\n")
	count := 0
	for _, y := range []int{-1, 0, 2, 4} {
		evaluatorSetFraction(t, e, 0, int64(y), 1)
		for index := uint32(1); index <= 3; index++ {
			root := ""
			switch {
			case y == 0 && index == 1:
				root = "0.0"
			case y == 2 && index <= 2:
				root = fmt.Sprintf("(root-obj (+ (^ x 2) (- 2)) %d)", index)
			case y == 4 && index == 1:
				root = "(- 2.0)"
			case y == 4 && index == 2:
				root = "2.0"
			}
			for _, kind := range []AtomKind{AtomRootEQ, AtomRootLT, AtomRootGT, AtomRootLE, AtomRootGE} {
				b, err := f.mkRoot(kind, 1, index, p)
				assignmentMust(t, err)
				literal := "false"
				if root != "" {
					literal = "(" + evaluatorSMTOperator(kind) + " x " + root + ")"
				}
				for _, neg := range []bool{false, true} {
					evaluatorOracleQueries(t, e, f.bools.atoms[b], neg, literal, &script)
					count += 2
				}
			}
		}
		for _, kind := range []AtomKind{AtomEQ, AtomLT, AtomGT} {
			b, err := f.mkInequality(kind, []polynomial.Polynomial{p}, []bool{false})
			assignmentMust(t, err)
			literal := "(" + evaluatorSMTOperator(kind) + " (- " + intervalSMTInteger(int32(y)) + " (* x x)) 0.0)"
			for _, neg := range []bool{false, true} {
				evaluatorOracleQueries(t, e, f.bools.atoms[b], neg, literal, &script)
				count += 2
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
		t.Fatal("C++ errors or surplus output", len(statuses), count, string(out))
	}
	for i, status := range statuses {
		if status != "unsat" {
			t.Fatal("registered atom disagrees with C++", i, status)
		}
	}
	t.Logf("prebuilt C++ rejects all %d registered-root/inequality projection and value disagreements", count)
}
