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

func explanationCellAtoms(t *testing.T, e *Evaluator, r *searchState, n *explanationAssumptions, ps *polynomial.PolynomialRefVector, y uint32, full bool) []*RootAtom {
	t.Helper()
	assignmentMust(t, n.collectCellLiterals(ps, y, full))
	result := make([]*RootAtom, len(n.output.result))
	for i, literal := range n.output.result {
		if !literal.Sign() {
			t.Fatal("cell literal must negate its selected root boundary", literal)
		}
		root, ok := r.store.atoms.bools.atoms[literal.Variable()].(*RootAtom)
		if !ok {
			t.Fatal("cell fallback did not construct a root atom", literal)
		}
		value, err := e.Eval(root, literal.Sign())
		assignmentMust(t, err)
		if value {
			t.Fatal("explanation cell literal is true in the current interpretation", literal)
		}
		result[i] = root
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
	if len(atoms) != 1 || atoms[0].kind != AtomRootEQ || atoms[0].index != 1 || atoms[0].polynomial != x {
		t.Fatal("exact cell root", atoms)
	}
	assignmentMust(t, n.reset())

	atoms = explanationCellAtoms(t, e, r, n, explanationCellVector(t, e, square, linearUpper), 1, false)
	if len(atoms) != 2 || atoms[0].kind != AtomRootGT || atoms[0].index != 1 || atoms[0].polynomial != square || atoms[1].kind != AtomRootLT || atoms[1].index != 2 || atoms[1].polynomial != square {
		t.Fatal("nearest lower/upper roots", atoms)
	}
	assignmentMust(t, n.reset())

	evaluatorSetFraction(t, e, 1, 3, 1)
	atoms = explanationCellAtoms(t, e, r, n, explanationCellVector(t, e, square, linearUpper), 1, false)
	if len(atoms) != 1 || atoms[0].kind != AtomRootGT || atoms[0].index != 1 || atoms[0].polynomial != linearUpper {
		t.Fatal("nearest finite lower root", atoms)
	}
	assignmentMust(t, n.reset())

	atoms = explanationCellAtoms(t, e, r, n, explanationCellVector(t, e, square, linearUpper), 1, true)
	if len(atoms) != 1 || atoms[0].kind != AtomRootGE || atoms[0].index != 1 || atoms[0].polynomial != linearUpper {
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
	if len(atoms) != 2 || atoms[0].polynomial != current || atoms[1].polynomial != current {
		t.Fatal("lower-stage polynomial contributed a current cell root", atoms)
	}
}
