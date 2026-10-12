package euclid

import (
	"testing"

	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestEuclideanNonUnitDecompositionParameterAndAuxiliaryEquation(t *testing.T) {
	for _, pivot := range []int32{2, -2} {
		name := "positive"
		if pivot < 0 {
			name = "negative"
		}
		t.Run(name, func(t *testing.T) {
			m := euclidNumeralManager(t)
			z := m.Integers()
			s, e := newSolverSetup(m)
			euclidMust(t, e)
			for i := 0; i < 2; i++ {
				_, e := s.mkVar(false)
				euclidMust(t, e)
			}
			euclideanAssertSmall(t, s, []int32{pivot, 3}, []uint32{0, 1}, 4, nullEquationIndex)
			found, e := s.selectNextVar()
			euclidMust(t, e)
			if !found || s.nextX != 0 || s.nextEq != 0 {
				t.Fatalf("selected non-unit pivot x%d eq%d", s.nextX, s.nextEq)
			}
			euclidMust(t, s.decomposeAndEliminate())
			if len(s.solved) != 3 || !s.parameter[2] || s.solved[0] != 0 ||
				!equalUint32Slices(s.equations[0].xs, []uint32{1, 2}) ||
				!equalUint32Slices(s.solution[0].xs, []uint32{0, 1, 2}) ||
				len(s.occurrences[0]) != 0 || !equalUint32Slices(s.occurrences[1], []uint32{0}) ||
				len(s.occurrences[2]) != 0 {
				t.Fatalf("decomposition IDs/occurrences: solved=%v todo=%v solution=%v occ=%v",
					s.solved, s.equations[0].xs, s.solution[0].xs, s.occurrences)
			}
			wantAuxX1, wantAuxC, wantParam := "-1", "-2", "2"
			if pivot < 0 {
				wantAuxX1, wantAuxC, wantParam = "1", "2", "-2"
			}
			checkEuclideanEquation(t, z, s.equations[0], []string{"1", wantParam}, "0")
			checkEuclideanEquation(t, z, s.solution[0], []string{"-1", wantAuxX1, "1"}, wantAuxC)
			if s.decomposeBuffer.Size() != 3 {
				t.Fatalf("auxiliary source buffer size = %d", s.decomposeBuffer.Size())
			}
			euclidMust(t, s.Close())
		})
	}
}

func TestEuclideanNonUnitDecompositionZeroRemainderDropsOccurrence(t *testing.T) {
	m := euclidNumeralManager(t)
	z := m.Integers()
	s, e := newSolverSetup(m)
	euclidMust(t, e)
	for i := 0; i < 3; i++ {
		_, e := s.mkVar(false)
		euclidMust(t, e)
	}
	euclideanAssertSmall(t, s, []int32{2, 4, 3}, []uint32{0, 1, 2}, 4, nullEquationIndex)
	found, e := s.selectNextVar()
	euclidMust(t, e)
	if !found || s.nextX != 0 {
		t.Fatalf("selected x%d, want x0", s.nextX)
	}
	euclidMust(t, s.decomposeAndEliminate())
	if !equalUint32Slices(s.equations[0].xs, []uint32{2, 3}) ||
		len(s.occurrences[1]) != 0 || !equalUint32Slices(s.occurrences[2], []uint32{0}) {
		t.Fatalf("zero remainder retained variable: xs=%v occ=%v", s.equations[0].xs, s.occurrences)
	}
	checkEuclideanEquation(t, z, s.equations[0], []string{"1", "2"}, "0")
	checkEuclideanEquation(t, z, s.solution[0], []string{"-1", "-2", "-1", "1"}, "-2")
	euclidMust(t, s.Close())
}

func TestEuclideanNonUnitDecompositionSubstitutesOtherEquations(t *testing.T) {
	m := euclidNumeralManager(t)
	z := m.Integers()
	s, e := newSolverSetup(m)
	euclidMust(t, e)
	for i := 0; i < 3; i++ {
		_, e := s.mkVar(false)
		euclidMust(t, e)
	}
	euclideanAssertSmall(t, s, []int32{2, 3}, []uint32{0, 1}, 4, nullEquationIndex)
	euclideanAssertSmall(t, s, []int32{5, 7}, []uint32{0, 2}, 6, nullEquationIndex)
	found, e := s.selectNextVar()
	euclidMust(t, e)
	if !found || s.nextX != 0 || s.nextEq != 0 {
		t.Fatalf("selected pivot x%d eq%d", s.nextX, s.nextEq)
	}
	euclidMust(t, s.decomposeAndEliminate())
	if !equalUint32Slices(s.equations[1].xs, []uint32{1, 2, 3}) ||
		len(s.occurrences[0]) != 0 || !equalUint32Slices(s.occurrences[1], []uint32{0, 1}) ||
		!equalUint32Slices(s.occurrences[2], []uint32{1}) || !equalUint32Slices(s.occurrences[3], []uint32{1}) {
		t.Fatalf("decomposition substitution IDs/occurrences: xs=%v occ=%v", s.equations[1].xs, s.occurrences)
	}
	checkEuclideanEquation(t, z, s.equations[1], []string{"-5", "7", "5"}, "-4")
	euclidMust(t, s.Close())
}

func TestEuclideanNonUnitDecompositionNegativeOtherCoefficient(t *testing.T) {
	m := euclidNumeralManager(t)
	z := m.Integers()
	s, e := newSolverSetup(m)
	euclidMust(t, e)
	for i := 0; i < 2; i++ {
		_, e := s.mkVar(false)
		euclidMust(t, e)
	}
	euclideanAssertSmall(t, s, []int32{2, -3}, []uint32{0, 1}, 4, nullEquationIndex)
	found, e := s.selectNextVar()
	euclidMust(t, e)
	if !found || s.nextX != 0 {
		t.Fatalf("selected x%d, want x0", s.nextX)
	}
	euclidMust(t, s.decomposeAndEliminate())
	checkEuclideanEquation(t, z, s.equations[0], []string{"-1", "2"}, "0")
	checkEuclideanEquation(t, z, s.solution[0], []string{"-1", "1", "1"}, "-2")
	euclidMust(t, s.Close())
}

func checkEuclideanEquation(t *testing.T, z *numeral.IntegerManager, eq *equation, coefficients []string, constant string) {
	t.Helper()
	if len(eq.as) != len(coefficients) {
		t.Fatalf("equation has %d coefficients, want %d", len(eq.as), len(coefficients))
	}
	for i, want := range coefficients {
		got, e := z.String(&eq.as[i])
		euclidMust(t, e)
		if got != want {
			t.Fatalf("coefficient[%d] = %s, want %s", i, got, want)
		}
	}
	got, e := z.String(&eq.c)
	euclidMust(t, e)
	if got != constant {
		t.Fatalf("equation constant = %s, want %s", got, constant)
	}
}
