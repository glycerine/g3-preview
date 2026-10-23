package simplex

import (
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestArithmeticGCDUsesFixedBoundsAndSkipsNonfixedReals(t *testing.T) {
	for _, fixed := range []bool{false, true} {
		s := NewSolver(nil)
		row, err := s.AddRationalRow(0, []uint32{0, 1, 2}, integerRow(2, 4, 1))
		if err != nil {
			t.Fatal(err)
		}
		if fixed {
			value := RationalInfinitesimal(big.NewRat(1, 1))
			if err = s.SetLower(2, value); err != nil {
				t.Fatal(err)
			}
			if err = s.SetUpper(2, value); err != nil {
				t.Fatal(err)
			}
			// A stale candidate says c=2, which would pass. Its fixed bounds
			// say c=1, so the source test must still reject the row.
			s.SetValue(2, RationalInfinitesimal(big.NewRat(2, 1)))
		}
		result, err := s.ArithmeticGCDRow(row, map[uint32]bool{0: true, 1: true})
		want := ArithmeticGCDPass
		if fixed {
			want = ArithmeticGCDConflict
		}
		if err != nil || result != want {
			t.Fatal(fixed, result, want, err)
		}
	}
}

func TestArithmeticExtendedGCDPositiveAndNegativeIntervals(t *testing.T) {
	for _, sign := range []int64{-1, 1} {
		for _, upper := range []int64{2, 5} {
			s := NewSolver(nil)
			row, err := s.AddRationalRow(1, []uint32{1, 0}, integerRow(5, 2*sign))
			if err != nil {
				t.Fatal(err)
			}
			if err = s.SetLower(0, RationalInfinitesimal(big.NewRat(1, 1))); err != nil {
				t.Fatal(err)
			}
			if err = s.SetUpper(0, RationalInfinitesimal(big.NewRat(upper, 1))); err != nil {
				t.Fatal(err)
			}
			if status, err := s.MakeFeasible(); err != nil || status != lbool.True {
				t.Fatal(status, err)
			}
			result, err := s.ArithmeticGCDRow(row, map[uint32]bool{0: true, 1: true})
			want := ArithmeticGCDPass
			if upper == 2 {
				want = ArithmeticExtendedGCDConflict
			}
			if err != nil || result != want {
				t.Fatal(sign, upper, result, want, err)
			}
		}
	}
}
