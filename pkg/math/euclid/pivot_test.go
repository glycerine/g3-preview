package euclid

import (
	"testing"

	"github.com/glycerine/g3/pkg/util/numeral"
)

func euclideanAssertSmall(t *testing.T, s *solverSetup, coefficients []int32, xs []uint32, constant int32, j uint32) {
	t.Helper()
	z := s.manager.Integers()
	as := make([]numeral.Int, len(coefficients))
	for i, a := range coefficients {
		as[i] = numeral.NewInt(a)
	}
	c := numeral.NewInt(constant)
	euclidMust(t, s.assertEquation(as, xs, &c, j))
	for i := range as {
		euclidMust(t, z.Del(&as[i]))
	}
	euclidMust(t, z.Del(&c))
}

func TestEuclideanSelectNextPivotSourceTiesAndUnitStop(t *testing.T) {
	m := euclidNumeralManager(t)
	s, e := newSolverSetup(m)
	euclidMust(t, e)
	for i := 0; i < 6; i++ {
		_, e := s.mkVar(false)
		euclidMust(t, e)
	}
	euclideanAssertSmall(t, s, []int32{3, 5}, []uint32{0, 1}, 0, nullEquationIndex)
	euclideanAssertSmall(t, s, []int32{3, 4, 5}, []uint32{2, 3, 4}, 0, nullEquationIndex)
	found, e := s.selectNextVar()
	euclidMust(t, e)
	if !found || s.nextX != 0 || s.nextEq != 0 || !s.nextPosA {
		t.Fatalf("shorter-equation tie: found=%v x=%d eq=%d", found, s.nextX, s.nextEq)
	}
	gotA, e := m.Integers().String(&s.nextA)
	euclidMust(t, e)
	if gotA != "3" {
		t.Fatalf("selected absolute coefficient = %s", gotA)
	}
	// x0 now occurs twice, so x2 wins the coefficient tie even though its
	// equation has more terms.
	euclideanAssertSmall(t, s, []int32{4, 5}, []uint32{0, 5}, 0, nullEquationIndex)
	found, e = s.selectNextVar()
	euclidMust(t, e)
	if !found || s.nextX != 2 || s.nextEq != 1 {
		t.Fatalf("occurrence-count tie: found=%v x=%d eq=%d", found, s.nextX, s.nextEq)
	}
	// A unit in the last equation stops the reverse scan immediately.
	euclideanAssertSmall(t, s, []int32{-1}, []uint32{5}, 0, nullEquationIndex)
	found, e = s.selectNextVar()
	euclidMust(t, e)
	if !found || s.nextX != 5 || s.nextEq != 3 || s.nextPosA {
		t.Fatalf("unit early stop: found=%v x=%d eq=%d positive=%v", found, s.nextX, s.nextEq, s.nextPosA)
	}
	euclidMust(t, s.Close())
}

func TestEuclideanEliminateUnitSubstitutesOccurrences(t *testing.T) {
	for _, pivot := range []int32{1, -1} {
		t.Run(map[int32]string{1: "positive", -1: "negative"}[pivot], func(t *testing.T) {
			m := euclidNumeralManager(t)
			z := m.Integers()
			s, e := newSolverSetup(m)
			euclidMust(t, e)
			for i := 0; i < 3; i++ {
				_, e := s.mkVar(false)
				euclidMust(t, e)
			}
			for i := 0; i < 2; i++ {
				_, e := s.mkJustification()
				euclidMust(t, e)
			}
			euclideanAssertSmall(t, s, []int32{pivot, 2}, []uint32{0, 1}, 1, 0)
			euclideanAssertSmall(t, s, []int32{2, 3}, []uint32{0, 2}, 4, 1)
			found, e := s.selectNextVar()
			euclidMust(t, e)
			if !found || s.nextX != 0 || s.nextEq != 0 {
				t.Fatalf("selected pivot = x%d in equation %d", s.nextX, s.nextEq)
			}
			euclidMust(t, s.eliminateUnit())
			if s.solved[0] != 0 || s.equations[0] != nil || len(s.solution) != 1 ||
				!equalUint32Slices(s.equations[1].xs, []uint32{1, 2}) ||
				len(s.occurrences[0]) != 0 || !equalUint32Slices(s.occurrences[1], []uint32{1}) ||
				!equalUint32Slices(s.occurrences[2], []uint32{1}) {
				t.Fatalf("elimination ownership/occurrences: solved=%v xs=%v occ=%v", s.solved, s.equations[1].xs, s.occurrences)
			}
			wantA, wantC, wantB := "-4", "2", "-2"
			if pivot < 0 {
				wantA, wantC, wantB = "4", "6", "2"
			}
			eq := s.equations[1]
			a, e := z.String(&eq.as[0])
			euclidMust(t, e)
			c, e := z.String(&eq.c)
			euclidMust(t, e)
			b, e := m.String(&eq.bs[0])
			euclidMust(t, e)
			if a != wantA || c != wantC || b != wantB || !equalUint32Slices(eq.js, []uint32{0, 1}) {
				t.Fatalf("substituted equation: a=%s c=%s b=%s js=%v", a, c, b, eq.js)
			}
			euclidMust(t, s.Close())
		})
	}
}
