package nlsat

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/math/polynomial"
)

func evaluatorSMTOperator(k AtomKind) string {
	switch k {
	case AtomEQ, AtomRootEQ:
		return "="
	case AtomLT, AtomRootLT:
		return "<"
	case AtomGT, AtomRootGT:
		return ">"
	case AtomRootLE:
		return "<="
	case AtomRootGE:
		return ">="
	}
	panic("invalid test kind")
}
func evaluatorOracleQueries(t *testing.T, e *Evaluator, a ArithmeticAtom, neg bool, literal string, script *strings.Builder) {
	t.Helper()
	if neg {
		literal = "(not " + literal + ")"
	}
	evaluatorSetFraction(t, e, 1, 101, 1)
	s, err := e.InfeasibleIntervals(a, neg)
	assignmentMust(t, err)
	fmt.Fprintf(script, "(push)(assert (not (= %s (not %s))))(check-sat)(pop)\n", intervalSMTSet(t, e.ism, s), literal)
	assignmentMust(t, e.ism.DecRef(s))
	evaluatorSetFraction(t, e, 1, 3, 2)
	value, err := e.Eval(a, neg)
	assignmentMust(t, err)
	fmt.Fprintf(script, "(push)(assert (= x (/ 3.0 2.0)))(assert (not (= %s %s)))(check-sat)(pop)\n", literal, strconv.FormatBool(value))
}

func TestEvaluatorProjectionAndValuesPrebuiltCPPOracle(t *testing.T) {
	reference := os.Getenv("G3_CPP_REFERENCE")
	if reference == "" {
		t.Skip("set G3_CPP_REFERENCE to prebuilt z3")
	}
	e := evaluatorFixture(t)
	p := evaluatorPolynomial(t, e, evaluatorTerm{1, 0, 2}, evaluatorTerm{-1, 1, 0})
	q := evaluatorPolynomial(t, e, evaluatorTerm{1, 0, 1}, evaluatorTerm{-1, 0, 0})
	r := evaluatorPolynomial(t, e, evaluatorTerm{1, 1, 0}, evaluatorTerm{-1, 0, 0})
	var script strings.Builder
	script.WriteString("(declare-const x Real)\n")
	count := 0
	for _, y := range []int{-1, 1, 4} {
		evaluatorSetFraction(t, e, 0, int64(y), 1)
		for parity := 0; parity < 8; parity++ {
			factors := []string{"(- (* x x) " + intervalSMTInteger(int32(y)) + ")", "(- x 1.0)", intervalSMTInteger(int32(y - 1))}
			for i, f := range factors {
				if parity&(1<<i) != 0 {
					factors[i] = "(* " + f + " " + f + ")"
				}
			}
			product := "(* " + strings.Join(factors, " ") + ")"
			for _, k := range []AtomKind{AtomEQ, AtomLT, AtomGT} {
				var a InequalityAtom
				assignmentMust(t, a.initialize(k, []polynomial.Polynomial{p, q, r}, []bool{parity&1 != 0, parity&2 != 0, parity&4 != 0}, 1, make([]InequalityFactor, 3)))
				a.boolVar = 17
				literal := "(" + evaluatorSMTOperator(k) + " " + product + " 0.0)"
				for _, neg := range []bool{false, true} {
					evaluatorOracleQueries(t, e, &a, neg, literal, &script)
					count += 2
				}
			}
		}
	}
	for _, y := range []int{-1, 0, 2, 4} {
		evaluatorSetFraction(t, e, 0, int64(y), 1)
		for index := uint32(1); index <= 3; index++ {
			root := ""
			if y == 0 && index == 1 {
				root = "0.0"
			}
			if y == 2 && index <= 2 {
				root = fmt.Sprintf("(root-obj (+ (^ x 2) (- 2)) %d)", index)
			}
			if y == 4 && index <= 2 {
				if index == 1 {
					root = "(- 2.0)"
				} else {
					root = "2.0"
				}
			}
			for _, k := range []AtomKind{AtomRootEQ, AtomRootLT, AtomRootGT, AtomRootLE, AtomRootGE} {
				var a RootAtom
				assignmentMust(t, a.initialize(k, 1, index, p))
				a.boolVar = 21
				literal := "false"
				if root != "" {
					literal = "(" + evaluatorSMTOperator(k) + " x " + root + ")"
				}
				for _, neg := range []bool{false, true} {
					evaluatorOracleQueries(t, e, &a, neg, literal, &script)
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
		t.Fatal("C++ errors or surplus output", len(statuses), count, string(out))
	}
	for i, status := range statuses {
		if status != "unsat" {
			t.Fatal("C++ evaluator disagreement", i, status)
		}
	}
	t.Logf("prebuilt C++ rejects all %d projected-set and value disagreements", count)
}
