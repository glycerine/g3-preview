package nlsat

import (
	"testing"

	"github.com/glycerine/g3/pkg/math/polynomial"
	"github.com/glycerine/g3/pkg/sat"
	"github.com/glycerine/g3/pkg/util/lbool"
)

func solverFixture(t *testing.T) *Solver {
	t.Helper()
	am, _, limit := assignmentFixtureWithLimit(t)
	q, err := am.RationalManager()
	assignmentMust(t, err)
	pm, err := polynomial.NewMultivariatePolynomialManager(polynomial.MultivariatePolynomialManagerConfig{Core: polynomial.MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Checkpoint: limit.Inc, Cooperation: func() error { return nil }}})
	assignmentMust(t, err)
	t.Cleanup(func() { assignmentMust(t, pm.CloseChecked()) })
	s, err := NewSolver(SolverConfig{Algebraic: am, Polynomial: pm, Limit: limit, SimplifyCore: true})
	assignmentMust(t, err)
	t.Cleanup(func() { assignmentMust(t, s.Close()) })
	return s
}

func solverUnit(t *testing.T, s *Solver, kind AtomKind, p polynomial.Polynomial) sat.Literal {
	t.Helper()
	literal, err := s.MkInequality(kind, []polynomial.Polynomial{p}, []bool{false}, false)
	assignmentMust(t, err)
	assignmentMust(t, s.AddClause([]sat.Literal{literal}, Assumption{}))
	return literal
}

func TestSolverMixedPolynomialSearchAndRepeatedModel(t *testing.T) {
	s := solverFixture(t)
	for i := uint32(0); i < 2; i++ {
		x, err := s.MkVar(false)
		assignmentMust(t, err)
		if x != i {
			t.Fatal("solver variable order", x, i)
		}
	}
	p := evaluatorPolynomial(t, s.evaluator, evaluatorTerm{1, 2, 0}, evaluatorTerm{1, 0, 2}, evaluatorTerm{-2, 0, 0})
	literal := solverUnit(t, s, AtomEQ, p)
	for check := 0; check < 2; check++ {
		result, err := s.Check()
		assignmentMust(t, err)
		if result != lbool.True {
			t.Fatal("mixed quadratic search", check, result)
		}
		value, err := s.search.value(literal)
		assignmentMust(t, err)
		if value != lbool.True {
			t.Fatal("published assignment violates mixed polynomial", check, value)
		}
		for x := uint32(0); x < 2; x++ {
			var witness polynomial.Algebraic
			assignmentMust(t, s.Value(x, &witness))
			assignmentMust(t, s.config.Algebraic.Del(&witness))
		}
	}
}

func TestSolverMixedLinearConflict(t *testing.T) {
	s := solverFixture(t)
	for i := 0; i < 2; i++ {
		_, err := s.MkVar(false)
		assignmentMust(t, err)
	}
	inputs := []struct {
		kind  AtomKind
		terms []evaluatorTerm
	}{
		{AtomGT, []evaluatorTerm{{1, 1, 0}}},
		{AtomGT, []evaluatorTerm{{1, 0, 1}}},
		{AtomLT, []evaluatorTerm{{1, 1, 0}, {1, 0, 1}}},
	}
	attachments := make([]*AssumptionAttachment, len(inputs))
	for i, input := range inputs {
		attachments[i] = &AssumptionAttachment{}
		p := evaluatorPolynomial(t, s.evaluator, input.terms...)
		literal, err := s.MkInequality(input.kind, []polynomial.Polynomial{p}, []bool{false}, false)
		assignmentMust(t, err)
		assignmentMust(t, s.AddClause([]sat.Literal{literal}, AttachedAssumption(attachments[i])))
	}
	result, err := s.Check()
	assignmentMust(t, err)
	if result != lbool.False {
		t.Fatal("mixed linear contradiction", result)
	}
	var core []Assumption
	assignmentMust(t, s.Core(&core))
	if len(core) != len(attachments) {
		t.Fatal("mixed conflict core size", len(core), len(attachments))
	}
	seen := map[*AssumptionAttachment]bool{}
	for _, assumption := range core {
		seen[assumption.Attachment()] = true
	}
	for _, attachment := range attachments {
		if !seen[attachment] {
			t.Fatal("mixed conflict core lost input dependency", core)
		}
	}
}
