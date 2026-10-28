package nlsat

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/glycerine/g3/pkg/math/polynomial"
	"github.com/glycerine/g3/pkg/sat"
)

// Render the actual coefficient/monomial data independently. The retained
// C++ display_smt2 helper itself uses infix monomial display for powers.
func explanationPolynomialSMT(t *testing.T, e *Evaluator, p polynomial.Polynomial) string {
	t.Helper()
	info, err := e.core.Info(p)
	assignmentMust(t, err)
	mm, err := e.core.MonomialManager()
	assignmentMust(t, err)
	z, err := e.core.IntegerManager()
	assignmentMust(t, err)
	numVars, err := e.core.NumVars()
	assignmentMust(t, err)
	var terms []string
	for i := uint32(0); i < info.Size; i++ {
		coefficient, err := e.core.Coefficient(p, i)
		assignmentMust(t, err)
		number, err := z.Int64(&coefficient)
		assignmentMust(t, err)
		if number == 0 {
			continue
		}
		text := strconv.FormatInt(number, 10)
		if number < 0 {
			text = "(- " + text[1:] + ".0)"
		} else {
			text += ".0"
		}
		mon, err := e.core.TermMonomial(p, i)
		assignmentMust(t, err)
		factors := []string{text}
		for x := uint32(0); x < numVars; x++ {
			degree, err := mm.DegreeOf(mon, x)
			assignmentMust(t, err)
			for k := uint32(0); k < degree; k++ {
				factors = append(factors, "x"+strconv.FormatUint(uint64(x), 10))
			}
		}
		term := text
		if len(factors) > 1 {
			term = "(* " + strings.Join(factors, " ") + ")"
		}
		terms = append(terms, term)
	}
	if len(terms) == 0 {
		return "0.0"
	}
	if len(terms) == 1 {
		return terms[0]
	}
	return "(+ " + strings.Join(terms, " ") + ")"
}
func explanationLiteralSMT(t *testing.T, e *Evaluator, r *searchState, l sat.Literal) string {
	t.Helper()
	if l == TrueLiteral {
		return "true"
	}
	if l == FalseLiteral {
		return "false"
	}
	a, ok := r.store.atoms.bools.atoms[l.Variable()].(*InequalityAtom)
	if !ok {
		t.Fatal("not an inequality", l)
	}
	var factors []string
	for _, factor := range a.factors {
		p := explanationPolynomialSMT(t, e, factor.Polynomial)
		if factor.Even {
			p = "(* " + p + " " + p + ")"
		}
		factors = append(factors, p)
	}
	product := "1"
	if len(factors) == 1 {
		product = factors[0]
	} else if len(factors) > 1 {
		product = "(* " + strings.Join(factors, " ") + ")"
	}
	expr := "(" + evaluatorSMTOperator(a.kind) + " " + product + " 0)"
	if l.Sign() {
		expr = "(not " + expr + ")"
	}
	return expr
}
func TestExplanationZeroAssumptionsPrebuiltCPPOracle(t *testing.T) {
	reference := os.Getenv("G3_CPP_REFERENCE")
	if reference == "" {
		t.Skip("set G3_CPP_REFERENCE to prebuilt z3")
	}
	var script strings.Builder
	script.WriteString("(set-logic QF_NRA)\n(declare-const x0 Real)\n(declare-const x1 Real)\n")
	queries := 0
	query := func(formula string) {
		fmt.Fprintf(&script, "(push)(assert %s)(check-sat-using qfnra-nlsat)(pop)\n", formula)
		queries++
	}
	for root := int32(-4); root <= 4; root++ {
		e, r, n := explanationAssumptionFixture(t)
		evaluatorSetFraction(t, e, 0, int64(root), 1)
		p := evaluatorPolynomial(t, e, evaluatorTerm{1, 3, 0}, evaluatorTerm{-root, 2, 0}, evaluatorTerm{1, 1, 0}, evaluatorTerm{-root, 0, 0})
		assignmentMust(t, n.collectZero(p))
		if len(n.output.result) != 1 {
			t.Fatal("zero assumption count")
		}
		actual := explanationPolynomialSMT(t, e, p)
		assumption := explanationLiteralSMT(t, e, r, n.output.result[0])
		number := intervalSMTInteger(root)
		input := "(* (- x0 " + number + ") (+ (* x0 x0) 1.0))"
		query("(not (= " + actual + " " + input + "))")
		query("(and (= x0 " + number + ") " + assumption + ")")
		query("(and (not " + assumption + ") (not (= " + input + " 0.0)))")
		query("(not (= " + assumption + " (not (= x0 " + number + "))))")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, reference, "-in", "-smt2")
	cmd.Stdin = strings.NewReader(script.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("C++ assumption oracle %v: %s", err, out)
	}
	answers := strings.Fields(string(out))
	if len(answers) != queries {
		t.Fatalf("C++ answers %d want %d: %s", len(answers), queries, out)
	}
	for i, answer := range answers {
		if answer != "unsat" {
			t.Fatal("zero assumption counterexample", i, answer)
		}
	}
}
