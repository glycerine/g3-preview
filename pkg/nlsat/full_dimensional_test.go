package nlsat

import (
	"testing"

	"github.com/glycerine/g3/pkg/math/polynomial"
	"github.com/glycerine/g3/pkg/sat"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestFullDimensionalLiteralClassification(t *testing.T) {
	e, r := searchStateFixture(t)
	p := evaluatorPolynomial(t, e, evaluatorTerm{1, 1, 0})
	for _, row := range []struct {
		kind AtomKind
		sign bool
		want bool
	}{
		{AtomEQ, false, false}, {AtomEQ, true, true},
		{AtomLT, false, true}, {AtomLT, true, false},
		{AtomGT, false, true}, {AtomGT, true, false},
	} {
		literal := normalizationLiteral(t, r, row.kind, []polynomial.Polynomial{p}, []bool{false})
		if row.sign {
			literal = literal.Negated()
		}
		got, err := r.store.isFullDimensionalLiteral(literal)
		assignmentMust(t, err)
		if got != row.want {
			t.Fatal(row.kind, row.sign, got, row.want)
		}
	}
	for _, row := range []struct {
		kind AtomKind
		sign bool
		want bool
	}{
		{AtomRootEQ, false, false}, {AtomRootEQ, true, true},
		{AtomRootLT, false, true}, {AtomRootLT, true, false},
		{AtomRootGT, false, true}, {AtomRootGT, true, false},
		{AtomRootLE, false, false}, {AtomRootLE, true, true},
		{AtomRootGE, false, false}, {AtomRootGE, true, true},
	} {
		b, err := r.store.atoms.mkRoot(row.kind, 0, 1, p)
		assignmentMust(t, err)
		literal := sat.MakeLiteral(b, row.sign)
		normalizationOwnLiteral(t, r, literal)
		got, err := r.store.isFullDimensionalLiteral(literal)
		assignmentMust(t, err)
		if got != row.want {
			t.Fatal(row.kind, row.sign, got, row.want)
		}
	}
}

func TestSolverCheckPublishesFullDimensionalExplanationMode(t *testing.T) {
	s := solverFixture(t)
	_, err := s.MkVar(false)
	assignmentMust(t, err)
	p := evaluatorPolynomial(t, s.evaluator, evaluatorTerm{1, 1, 0})
	strict, err := s.MkInequality(AtomGT, []polynomial.Polynomial{p}, []bool{false}, false)
	assignmentMust(t, err)
	assignmentMust(t, s.AddClause([]sat.Literal{strict}, Assumption{}))
	result, err := s.Check()
	assignmentMust(t, err)
	if result != lbool.True || !s.explanation.fullDimensional {
		t.Fatal("strict problem did not enable full-dimensional explanations", result, s.explanation.fullDimensional)
	}
	equality, err := s.MkInequality(AtomEQ, []polynomial.Polynomial{p}, []bool{false}, false)
	assignmentMust(t, err)
	assignmentMust(t, s.AddClause([]sat.Literal{equality}, Assumption{}))
	result, err = s.Check()
	assignmentMust(t, err)
	if result != lbool.False || s.explanation.fullDimensional {
		t.Fatal("equality did not disable full-dimensional explanations", result, s.explanation.fullDimensional)
	}
}
