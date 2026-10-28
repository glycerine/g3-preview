package nlsat

import (
	"testing"

	"github.com/glycerine/g3/pkg/math/polynomial"
	"github.com/glycerine/g3/pkg/sat"
)

func TestExplanationMainCollectsAtomsAndBuildsProjectedCell(t *testing.T) {
	e, r, n, todo := projectionCoefficientFixture(t, true)
	evaluatorSetFraction(t, e, 0, 0, 1)
	evaluatorSetFraction(t, e, 1, 1, 1)
	p := evaluatorPolynomial(t, e, evaluatorTerm{1, 0, 2}, evaluatorTerm{1, 1, 0}, evaluatorTerm{-2, 0, 0})
	b, err := r.store.atoms.mkInequality(AtomEQ, []polynomial.Polynomial{p}, []bool{false})
	assignmentMust(t, err)
	literal := sat.MakeLiteral(b, false)
	assignmentMust(t, n.collectExplanation([]sat.Literal{literal}, false))

	empty, err := todo.empty()
	assignmentMust(t, err)
	if !empty || len(n.output.result) != 1 {
		t.Fatal("complete explanation projection frontier", empty, n.output.result)
	}
	boundary := n.output.result[0]
	root, ok := r.store.atoms.bools.atoms[boundary.Variable()].(*RootAtom)
	if !ok || !boundary.Sign() || root.kind != AtomRootLT || root.Variable() != 0 || root.Index() != 1 {
		t.Fatal("projected explanation boundary", boundary, root)
	}
	value, err := e.Eval(root, boundary.Sign())
	assignmentMust(t, err)
	if value {
		t.Fatal("explanation boundary is true in the conflicting interpretation")
	}
}

func TestExplanationCollectPolynomialsPreservesLiteralFactorOrder(t *testing.T) {
	e, r, n := explanationAssumptionFixture(t)
	p := evaluatorPolynomial(t, e, evaluatorTerm{1, 1, 0})
	q := evaluatorPolynomial(t, e, evaluatorTerm{1, 0, 1})
	b, err := r.store.atoms.mkInequality(AtomGT, []polynomial.Polynomial{p, q}, []bool{false, true})
	assignmentMust(t, err)
	root, err := r.store.atoms.mkRoot(AtomRootGT, 1, 1, q)
	assignmentMust(t, err)
	ps := projectionOutput(t, e)
	assignmentMust(t, n.collectPolynomials([]sat.Literal{sat.MakeLiteral(b, true), sat.MakeLiteral(root, false)}, ps))
	values := projectionValues(t, ps)
	if len(values) != 3 || values[0] != p || values[1] != q || values[2] != q {
		t.Fatal("literal/factor collection order", values)
	}
}
