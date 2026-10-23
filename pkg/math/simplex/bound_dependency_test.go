package simplex

import (
	"math/big"
	"testing"
)

func TestBoundConflictDependenciesOwnAndInvalidateSnapshots(t *testing.T) {
	s := NewSolver(nil)
	dependencies := []BoundDependency{{Literal: 12, Coefficient: *big.NewRat(-5, 7)}}
	s.RecordBoundConflictDependencies(dependencies, true)
	dependencies[0].Coefficient.SetInt64(99)
	copy, known := s.BoundConflictDependencies()
	if !known || len(copy) != 1 || copy[0].Coefficient.Cmp(big.NewRat(-5, 7)) != 0 {
		t.Fatal(copy, known)
	}
	copy[0].Coefficient.SetInt64(42)
	copy, _ = s.BoundConflictDependencies()
	if copy[0].Coefficient.Cmp(big.NewRat(-5, 7)) != 0 {
		t.Fatal("borrowed conflict coefficient")
	}
	if err := s.SetLower(1, RationalInfinitesimal(big.NewRat(0, 1))); err != nil {
		t.Fatal(err)
	}
	if _, known = s.BoundConflictDependencies(); known {
		t.Fatal("bound update retained stale conflict")
	}
	s.RecordBoundConflictDependencies(dependencies, true)
	s.UnsetLower(1)
	if _, known = s.BoundConflictDependencies(); known {
		t.Fatal("unset retained stale conflict")
	}
	s.RecordBoundConflictDependencies(dependencies, true)
	s.Reset()
	if _, known = s.BoundConflictDependencies(); known {
		t.Fatal("reset retained conflict")
	}
	s.RecordBoundConflictDependencies(dependencies, false)
	if copy, known = s.BoundConflictDependencies(); known || len(copy) != 0 {
		t.Fatal("unknown conflict acquired premises", copy, known)
	}
}

func TestBoundDependenciesOwnCoefficientsAndInvalidateModelBounds(t *testing.T) {
	s := NewSolver(nil)
	dependencies := []BoundDependency{{Literal: 12, Coefficient: *big.NewRat(-7, 3)}}
	bound := RationalInfinitesimal(big.NewRat(2, 1))
	if err := s.SetLowerWithDependencies(1, bound, dependencies); err != nil {
		t.Fatal(err)
	}
	dependencies[0].Coefficient.SetInt64(99)
	got, known := s.BoundDependencies(1, true)
	if !known || len(got) != 1 || got[0].Coefficient.Cmp(big.NewRat(-7, 3)) != 0 {
		t.Fatal(got, known)
	}
	got[0].Coefficient.SetInt64(42)
	got, _ = s.BoundDependencies(1, true)
	if got[0].Coefficient.Cmp(big.NewRat(-7, 3)) != 0 {
		t.Fatal("borrowed dependency coefficient")
	}
	if err := s.SetUpperWithDependencies(1, bound, nil); err != nil {
		t.Fatal(err)
	}
	if got, known = s.BoundDependencies(1, false); !known || len(got) != 0 {
		t.Fatal("unconditional bound", got, known)
	}
	// A rejected update must preserve both the old bound and its explanation.
	if err := s.SetLower(1, RationalInfinitesimal(big.NewRat(3, 1))); err == nil {
		t.Fatal("accepted inconsistent bound")
	}
	if _, known = s.BoundDependencies(1, true); !known {
		t.Fatal("failed update erased explanation")
	}
	if err := s.SetLower(1, bound); err != nil {
		t.Fatal(err)
	}
	if _, known = s.BoundDependencies(1, true); known {
		t.Fatal("candidate bound retained logical explanation")
	}
	s.UnsetUpper(1)
	if _, known = s.BoundDependencies(1, false); known {
		t.Fatal("unset bound retained explanation")
	}
}

func TestArithmeticGCDExplanationNormalizedEndpoints(t *testing.T) {
	for _, extended := range []bool{false, true} {
		for _, sign := range []int64{-1, 1} {
			s := NewSolver(nil)
			column, coefficient := uint32(2), big.NewRat(1, 2)
			if extended {
				_, err := s.AddRationalRow(1, []uint32{1, 0}, integerRow(5, 2*sign))
				if err != nil {
					t.Fatal(err)
				}
				column, coefficient = 0, big.NewRat(2*sign, 5)
			} else {
				_, err := s.AddRationalRow(0, []uint32{0, 1, 2}, integerRow(2, 4, sign))
				if err != nil {
					t.Fatal(err)
				}
				coefficient.SetFrac64(sign, 2)
			}
			upper := int64(1)
			if extended {
				upper = 2
			}
			if err := s.SetLower(column, RationalInfinitesimal(big.NewRat(1, 1))); err != nil {
				t.Fatal(err)
			}
			if err := s.SetUpper(column, RationalInfinitesimal(big.NewRat(upper, 1))); err != nil {
				t.Fatal(err)
			}
			result, checks, bounds, err := s.ArithmeticGCDWithExplanation(map[uint32]bool{0: true, 1: true})
			want := ArithmeticGCDConflict
			if extended {
				want = ArithmeticExtendedGCDConflict
			}
			if err != nil || result != want || checks != 1 || len(bounds) != 2 {
				t.Fatal(extended, sign, result, checks, bounds, err)
			}
			for i, use := range bounds {
				if use.Column != column || use.Lower != (i == 0) || use.Coefficient.Cmp(coefficient) != 0 {
					t.Fatal("endpoint explanation", use, coefficient)
				}
			}
		}
	}
}
