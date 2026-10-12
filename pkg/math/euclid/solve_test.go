package euclid

import "testing"

func TestEuclideanSolveNonUnitThenUnitToParameterSolution(t *testing.T) {
	m := euclidNumeralManager(t)
	s, e := newSolverSetup(m)
	euclidMust(t, e)
	for i := 0; i < 2; i++ {
		_, e := s.mkVar(false)
		euclidMust(t, e)
	}
	euclideanAssertSmall(t, s, []int32{2, 3}, []uint32{0, 1}, 4, nullEquationIndex)
	consistent, e := s.solve()
	euclidMust(t, e)
	if !consistent || len(s.solution) != 2 || len(s.equations) != 0 ||
		len(s.solved) != 3 || s.solved[0] != 0 || s.solved[1] != 1 ||
		s.solved[2] != nullEquationIndex || !s.parameter[2] {
		t.Fatalf("solver state: consistent=%v solution=%d todo=%d solved=%v parameter=%v",
			consistent, len(s.solution), len(s.equations), s.solved, s.parameter)
	}
	checkEuclideanEquation(t, m.Integers(), s.solution[0], []string{"-1", "-1", "1"}, "-2")
	checkEuclideanEquation(t, m.Integers(), s.solution[1], []string{"-1", "-2"}, "0")
	consistent, e = s.solve()
	euclidMust(t, e)
	if !consistent || len(s.solution) != 2 {
		t.Fatal("repeated solve changed completed solution")
	}
	euclidMust(t, s.Close())
}

func TestEuclideanSolveContradictionAfterUnitSubstitution(t *testing.T) {
	m := euclidNumeralManager(t)
	s, e := newSolverSetup(m)
	euclidMust(t, e)
	_, e = s.mkVar(false)
	euclidMust(t, e)
	euclideanAssertSmall(t, s, []int32{1}, []uint32{0}, 0, nullEquationIndex)
	euclideanAssertSmall(t, s, []int32{1}, []uint32{0}, 1, nullEquationIndex)
	consistent, e := s.solve()
	euclidMust(t, e)
	if consistent || s.inconsistent == nullEquationIndex || s.equations[s.inconsistent] == nil {
		t.Fatalf("substitution contradiction: consistent=%v index=%d", consistent, s.inconsistent)
	}
	consistent, e = s.solve()
	euclidMust(t, e)
	if consistent {
		t.Fatal("inconsistent solve returned true")
	}
	euclidMust(t, s.Close())
}

func TestEuclideanSolveEmptyAndAlreadyInconsistent(t *testing.T) {
	m := euclidNumeralManager(t)
	s, e := newSolverSetup(m)
	euclidMust(t, e)
	consistent, e := s.solve()
	euclidMust(t, e)
	if !consistent {
		t.Fatal("empty solver is inconsistent")
	}
	euclideanAssertSmall(t, s, nil, nil, 2, nullEquationIndex)
	consistent, e = s.solve()
	euclidMust(t, e)
	if consistent || s.inconsistent != 0 {
		t.Fatal("preexisting inconsistency was lost")
	}
	euclidMust(t, s.Close())
}
