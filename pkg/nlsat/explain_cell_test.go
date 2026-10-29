package nlsat

import (
	"testing"

	"github.com/glycerine/g3/pkg/math/polynomial"
)

func explanationCellVector(t *testing.T, e *Evaluator, values ...polynomial.Polynomial) *polynomial.PolynomialRefVector {
	t.Helper()
	v, err := polynomial.NewPolynomialRefVector(e.core)
	assignmentMust(t, err)
	t.Cleanup(func() { assignmentMust(t, v.Close()) })
	for _, p := range values {
		assignmentMust(t, v.Push(p))
	}
	return v
}

func explanationCellAtoms(t *testing.T, e *Evaluator, r *searchState, n *explanationAssumptions, ps *polynomial.PolynomialRefVector, y uint32, full bool) []ArithmeticAtom {
	t.Helper()
	assignmentMust(t, n.collectCellLiterals(ps, y, full))
	result := make([]ArithmeticAtom, len(n.output.result))
	for i, literal := range n.output.result {
		a := r.store.atoms.bools.atoms[literal.Variable()]
		if a == nil {
			t.Fatal("cell did not construct an arithmetic atom", literal)
		}
		value, err := e.Eval(a, literal.Sign())
		assignmentMust(t, err)
		if value {
			t.Fatal("explanation cell literal is true in the current interpretation", literal)
		}
		result[i] = a
	}
	return result
}

func TestExplanationCellLiteralsExactRootAndNearestBounds(t *testing.T) {
	e, r, n := explanationAssumptionFixture(t)
	x := evaluatorPolynomial(t, e, evaluatorTerm{1, 0, 1})
	square := evaluatorPolynomial(t, e, evaluatorTerm{1, 0, 2}, evaluatorTerm{-2, 0, 0})
	linearUpper := evaluatorPolynomial(t, e, evaluatorTerm{1, 0, 1}, evaluatorTerm{-2, 0, 0})

	evaluatorSetFraction(t, e, 1, 0, 1)
	atoms := explanationCellAtoms(t, e, r, n, explanationCellVector(t, e, x), 1, false)
	if len(atoms) != 1 {
		t.Fatal("exact cell root count", atoms)
	}
	linear, ok := atoms[0].(*InequalityAtom)
	if !ok || linear.kind != AtomEQ || len(linear.factors) != 1 || linear.factors[0].Polynomial != x {
		t.Fatal("exact cell root", atoms)
	}
	assignmentMust(t, n.reset())

	atoms = explanationCellAtoms(t, e, r, n, explanationCellVector(t, e, square, linearUpper), 1, false)
	if len(atoms) != 2 {
		t.Fatal("nearest lower/upper root count", atoms)
	}
	if _, lowerOK := atoms[0].(*InequalityAtom); !lowerOK {
		t.Fatal("quadratic lower boundary did not use Thom signs", atoms)
	}
	if _, upperOK := atoms[1].(*InequalityAtom); !upperOK {
		t.Fatal("quadratic upper boundary did not use Thom signs", atoms)
	}
	if atoms[0].(*InequalityAtom).kind != AtomEQ || atoms[1].(*InequalityAtom).kind != AtomLT {
		t.Fatal("quadratic derivative/polynomial Thom signs", atoms)
	}
	assignmentMust(t, n.reset())

	evaluatorSetFraction(t, e, 1, 3, 1)
	atoms = explanationCellAtoms(t, e, r, n, explanationCellVector(t, e, square, linearUpper), 1, false)
	if len(atoms) != 1 {
		t.Fatal("nearest finite lower root count", atoms)
	}
	linear, ok = atoms[0].(*InequalityAtom)
	if !ok || linear.kind != AtomGT || len(linear.factors) != 1 || linear.factors[0].Polynomial != linearUpper {
		t.Fatal("nearest finite lower root", atoms)
	}
	assignmentMust(t, n.reset())

	atoms = explanationCellAtoms(t, e, r, n, explanationCellVector(t, e, square, linearUpper), 1, true)
	if len(atoms) != 1 {
		t.Fatal("full-dimensional lower root count", atoms)
	}
	linear, ok = atoms[0].(*InequalityAtom)
	if !ok || linear.kind != AtomLT || len(linear.factors) != 1 || linear.factors[0].Polynomial != linearUpper {
		t.Fatal("full-dimensional closed lower boundary", atoms)
	}
}

func TestExplanationCellLiteralsSkipLowerStagePolynomials(t *testing.T) {
	e, r, n := explanationAssumptionFixture(t)
	lowerStage := evaluatorPolynomial(t, e, evaluatorTerm{1, 1, 0}, evaluatorTerm{-1, 0, 0})
	current := evaluatorPolynomial(t, e, evaluatorTerm{1, 0, 2}, evaluatorTerm{-2, 0, 0})
	evaluatorSetFraction(t, e, 0, 1, 1)
	evaluatorSetFraction(t, e, 1, 0, 1)
	atoms := explanationCellAtoms(t, e, r, n, explanationCellVector(t, e, lowerStage, current), 1, false)
	if len(atoms) != 2 {
		t.Fatal("lower-stage root count", atoms)
	}
	if _, lowerOK := atoms[0].(*InequalityAtom); !lowerOK {
		t.Fatal("quadratic lower-stage boundary did not use Thom signs", atoms)
	}
	if _, upperOK := atoms[1].(*InequalityAtom); !upperOK {
		t.Fatal("quadratic lower-stage upper boundary did not use Thom signs", atoms)
	}
}

func TestExplanationLinearRootNegativeCoefficientRelationMapping(t *testing.T) {
	e, r, n := explanationAssumptionFixture(t)
	p := evaluatorPolynomial(t, e, evaluatorTerm{-1, 0, 1}, evaluatorTerm{2, 0, 0})
	ps := explanationCellVector(t, e, p)
	for _, tc := range []struct {
		name       string
		value      int64
		full       bool
		kind       AtomKind
		literalNeg bool
	}{
		{"equal", 2, false, AtomEQ, true},
		{"below-open", 1, false, AtomLT, true},
		{"above-open", 3, false, AtomGT, true},
		{"below-closed", 1, true, AtomGT, false},
		{"above-closed", 3, true, AtomLT, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			evaluatorSetFraction(t, e, 1, tc.value, 1)
			assignmentMust(t, n.collectCellLiterals(ps, 1, tc.full))
			defer func() { assignmentMust(t, n.reset()) }()
			if len(n.output.result) != 1 {
				t.Fatal("linear boundary count", n.output.result)
			}
			literal := n.output.result[0]
			ineq, ok := r.store.atoms.bools.atoms[literal.Variable()].(*InequalityAtom)
			if !ok || ineq.kind != tc.kind || len(ineq.factors) != 1 || literal.Sign() != tc.literalNeg {
				if ok {
					t.Fatal("negative-coefficient root mapping", ineq.kind, ineq.factors, literal.Sign(), tc)
				}
				t.Fatal("negative-coefficient root mapping", literal)
			}
			value, err := e.Eval(ineq, literal.Sign())
			assignmentMust(t, err)
			if value {
				t.Fatal("linear root explanation is true in its model", tc, literal)
			}
		})
	}
}

func TestExplanationQuadraticRootDegenerateLeadingCoefficient(t *testing.T) {
	e, r, n := explanationAssumptionFixture(t)
	// x0*x1^2 + x1 - 2 reduces to x1 - 2 at the current x0 = 0.
	p := evaluatorPolynomial(t, e, evaluatorTerm{1, 1, 2}, evaluatorTerm{1, 0, 1}, evaluatorTerm{-2, 0, 0})
	evaluatorSetFraction(t, e, 0, 0, 1)
	evaluatorSetFraction(t, e, 1, 3, 1)
	atoms := explanationCellAtoms(t, e, r, n, explanationCellVector(t, e, p), 1, false)
	if len(atoms) != 3 {
		t.Fatal("degenerate quadratic assumptions", atoms)
	}
	seenEQ, seenGT := false, false
	for _, atom := range atoms {
		ineq, ok := atom.(*InequalityAtom)
		if !ok {
			t.Fatal("degenerate quadratic retained root atom", atom)
		}
		seenEQ = seenEQ || ineq.kind == AtomEQ
		seenGT = seenGT || ineq.kind == AtomGT
	}
	if !seenEQ || !seenGT {
		t.Fatal("missing zero-leading or linear-boundary assumption", atoms)
	}
}
