package simplex

import (
	"math/big"
	"testing"

	z3rand "github.com/glycerine/g3/pkg/util/random"
)

func TestArithmeticGomoryIntegerCutExcludesCandidateAndPreservesSolutions(t *testing.T) {
	s := NewSolver(nil)
	row, err := s.AddRationalRow(0, []uint32{0, 1, 2, 3}, integerRow(4, -2, -1, -2))
	if err != nil {
		t.Fatal(err)
	}
	for _, column := range []uint32{1, 2} {
		if err = s.SetLower(column, RationalInfinitesimal(new(big.Rat))); err != nil {
			t.Fatal(err)
		}
	}
	one := RationalInfinitesimal(big.NewRat(1, 1))
	if err = s.SetLower(3, one); err != nil {
		t.Fatal(err)
	}
	if err = s.SetUpper(3, one); err != nil {
		t.Fatal(err)
	}
	cut, ready, err := s.ArithmeticGomory(row, map[uint32]bool{0: true, 1: true, 2: true, 3: true})
	if err != nil || !ready || cut.Conflict || len(cut.Terms) != 3 {
		t.Fatal(cut, ready, err)
	}
	if len(cut.Bounds) != 3 {
		t.Fatal("missing cut endpoint explanations", cut.Bounds)
	}
	for i, want := range []*big.Rat{big.NewRat(1, 1), big.NewRat(1, 2), big.NewRat(1, 1)} {
		use := &cut.Bounds[i]
		if use.Column != uint32(i+1) || !use.Lower || use.Coefficient.Cmp(want) != 0 {
			t.Fatal("proof weight changed during denominator clearing", i, use)
		}
	}
	value := new(big.Rat)
	for _, term := range cut.Terms {
		v := s.Value(term.Column)
		value.Add(value, new(big.Rat).Mul(&term.Coefficient, &v.Main))
	}
	if value.Cmp(&cut.Bound) >= 0 {
		t.Fatal("cut did not exclude fractional candidate", value, &cut.Bound)
	}
	checked := 0
	for y := int64(0); y <= 8; y++ {
		for z := int64(0); z <= 8; z++ {
			x := big.NewRat(2*y+z+2, 4)
			if !x.IsInt() {
				continue
			}
			checked++
			values := map[uint32]*big.Rat{0: x, 1: big.NewRat(y, 1), 2: big.NewRat(z, 1), 3: big.NewRat(1, 1)}
			left := new(big.Rat)
			for _, term := range cut.Terms {
				left.Add(left, new(big.Rat).Mul(&term.Coefficient, values[term.Column]))
			}
			if left.Cmp(&cut.Bound) < 0 {
				t.Fatal("Gomory cut removed feasible integer assignment", x, y, z, left, &cut.Bound)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no integer assignments checked")
	}
}

func TestArithmeticGomorySingleTermDirectionsAndMixedReals(t *testing.T) {
	for _, divisor := range []int64{2, 3} {
		for _, sign := range []int64{-1, 1} {
			for _, integer := range []bool{false, true} {
				for _, lower := range []bool{false, true} {
					s := NewSolver(nil)
					row, err := s.AddRationalRow(0, []uint32{0, 1}, integerRow(divisor, sign))
					if err != nil {
						t.Fatal(err)
					}
					one := RationalInfinitesimal(big.NewRat(1, 1))
					if lower {
						err = s.SetLower(1, one)
					} else {
						err = s.SetUpper(1, one)
						s.SetValue(1, one)
					}
					if err != nil {
						t.Fatal(err)
					}
					cut, ready, err := s.ArithmeticGomory(row, map[uint32]bool{0: true, 1: integer})
					want := int64(0)
					if lower {
						want = divisor
					}
					if err != nil || !ready || len(cut.Terms) != 1 || cut.Terms[0].Column != 1 || cut.Lower != lower || cut.Bound.Cmp(big.NewRat(want, 1)) != 0 {
						t.Fatal(integer, lower, cut, ready, err)
					}
					weight := big.NewRat(-1, 1)
					if lower {
						weight.SetFrac64(1, divisor-1)
					}
					if len(cut.Bounds) != 1 || cut.Bounds[0].Column != 1 || cut.Bounds[0].Lower != lower || cut.Bounds[0].Coefficient.Cmp(weight) != 0 {
						t.Fatal("proof weight changed during single-term normalization", divisor, sign, integer, lower, cut.Bounds)
					}
				}
			}
		}
	}
}

func TestArithmeticIntegerBasePrefersSmallestBoundedRange(t *testing.T) {
	s := NewSolver(nil)
	for _, row := range []struct{ base, other uint32 }{{0, 3}, {1, 4}, {2, 5}} {
		if _, err := s.AddRationalRow(row.base, []uint32{row.base, row.other}, integerRow(2, -1)); err != nil {
			t.Fatal(err)
		}
		s.SetValue(row.other, RationalInfinitesimal(big.NewRat(1, 1)))
	}
	for _, item := range []struct {
		column uint32
		upper  int64
	}{{0, 10}, {1, 2}} {
		if err := s.SetLower(item.column, RationalInfinitesimal(new(big.Rat))); err != nil {
			t.Fatal(err)
		}
		if err := s.SetUpper(item.column, RationalInfinitesimal(big.NewRat(item.upper, 1))); err != nil {
			t.Fatal(err)
		}
	}
	random := z3rand.New(7)
	column, found := s.ArithmeticIntegerBase(map[uint32]bool{0: true, 1: true, 2: true}, random)
	if !found || column != 1 {
		t.Fatal("integer base selection ignored bounded range", column, found)
	}
}
