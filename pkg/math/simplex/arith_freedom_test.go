package simplex

import (
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/util/lbool"
	z3rand "github.com/glycerine/g3/pkg/util/random"
)

func TestArithmeticFreedomIntegerMultiplesAndFixedBasePivot(t *testing.T) {
	s := NewSolver(nil)
	_, err := s.AddRationalRow(0, []uint32{0, 1}, integerRow(2, -1)) // b=x/2.
	if err != nil {
		t.Fatal(err)
	}
	for _, bound := range []struct {
		column uint32
		lower  bool
		value  int64
	}{{0, true, 0}, {0, false, 3}, {1, true, 1}} {
		value := RationalInfinitesimal(big.NewRat(bound.value, 1))
		if bound.lower {
			err = s.SetLower(bound.column, value)
		} else {
			err = s.SetUpper(bound.column, value)
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	if status, err := s.MakeFeasible(); err != nil || status != lbool.True {
		t.Fatal(status, err)
	}
	integers := map[uint32]bool{0: true, 1: true}
	interval, found, err := s.ArithmeticFreedom(1, integers)
	if err != nil || !found || !interval.LowerValid || !interval.UpperValid || interval.Lower.Main.Cmp(big.NewRat(1, 1)) != 0 || interval.Upper.Main.Cmp(big.NewRat(6, 1)) != 0 || interval.Multiple.Cmp(big.NewRat(2, 1)) != 0 {
		t.Fatal(interval, found, err)
	}
	var random z3rand.Generator
	random.SetSeed(13)
	for i := 0; i < 20; i++ {
		updated, err := s.ArithmeticRandomUpdate(1, integers, &random)
		if err != nil || !updated {
			t.Fatal(updated, err)
		}
		x, b := s.Value(1), s.Value(0)
		if !x.Main.IsInt() || !b.Main.IsInt() || x.Main.Cmp(big.NewRat(2, 1)) < 0 || x.Main.Cmp(big.NewRat(6, 1)) > 0 || x.Main.Cmp(new(big.Rat).Mul(&b.Main, big.NewRat(2, 1))) != 0 {
			t.Fatal("integer freedom update", x, b)
		}
		if status, err := s.MakeFeasible(); err != nil || status != lbool.True {
			t.Fatal(status, err)
		}
	}
	if err = s.SetLower(0, RationalInfinitesimal(big.NewRat(1, 1))); err != nil {
		t.Fatal(err)
	}
	if err = s.SetUpper(0, RationalInfinitesimal(big.NewRat(1, 1))); err != nil {
		t.Fatal(err)
	}
	if status, err := s.MakeFeasible(); err != nil || status != lbool.True {
		t.Fatal(status, err)
	}
	if err = s.RemoveArithmeticFixedBases(); err != nil {
		t.Fatal(err)
	}
	if s.isBase(0) || !s.isBase(1) {
		t.Fatal("fixed base was not pivoted out")
	}
	if updated, err := s.ArithmeticRandomUpdate(0, integers, &random); err != nil || updated {
		t.Fatal("fixed variable moved", updated, err)
	}
}

func TestArithmeticFreedomRealOpenIntervalsAndOneSidedUpdates(t *testing.T) {
	for _, bounds := range []int{0, 1, 2, 3} {
		s := NewSolver(nil)
		_, err := s.AddRationalRow(0, []uint32{0, 1}, integerRow(3, 6)) // b=-2*x.
		if err != nil {
			t.Fatal(err)
		}
		s.SetValue(1, RationalInfinitesimal(big.NewRat(2, 1)))
		if bounds&1 != 0 {
			if err = s.SetLower(0, RationalInfinitesimal(big.NewRat(-6, 1))); err != nil {
				t.Fatal(err)
			}
		}
		if bounds&2 != 0 {
			if err = s.SetUpper(0, RationalInfinitesimal(big.NewRat(-2, 1))); err != nil {
				t.Fatal(err)
			}
		}
		if bounds == 3 {
			lo, hi := RationalInfinitesimal(big.NewRat(1, 1)), RationalInfinitesimal(big.NewRat(3, 1))
			lo.Epsilon.SetInt64(1)
			hi.Epsilon.SetInt64(-1)
			if err = s.SetLower(1, lo); err != nil {
				t.Fatal(err)
			}
			if err = s.SetUpper(1, hi); err != nil {
				t.Fatal(err)
			}
		}
		if status, err := s.MakeFeasible(); err != nil || status != lbool.True {
			t.Fatal(status, err)
		}
		var random z3rand.Generator
		random.SetSeed(7)
		for i := 0; i < 10; i++ {
			updated, err := s.ArithmeticRandomUpdate(1, nil, &random)
			if err != nil || !updated {
				t.Fatal(bounds, updated, err)
			}
			x, b := s.Value(1), s.Value(0)
			expected := x.Clone()
			expected.Mul(big.NewRat(-2, 1))
			if b.Cmp(&expected) != 0 || s.outside(0) || s.outside(1) {
				t.Fatal("freedom update violated row or bounds", bounds, x, b)
			}
		}
	}
}

func TestArithmeticFreedomNegativeOpenIntegerLCM(t *testing.T) {
	for _, epsilon := range []int64{-1, 1} {
		s := NewSolver(nil)
		for _, row := range []struct {
			base    uint32
			divisor int64
		}{{0, 2}, {2, 3}} {
			if _, err := s.AddRationalRow(row.base, []uint32{row.base, 1}, integerRow(row.divisor, -1)); err != nil {
				t.Fatal(err)
			}
		}
		s.SetValue(1, RationalInfinitesimal(big.NewRat(-12, 1)))
		lo := NewInfinitesimal(big.NewRat(-18, 1), big.NewRat(epsilon, 1))
		hi := NewInfinitesimal(big.NewRat(-6, 1), big.NewRat(epsilon, 1))
		if err := s.SetLower(1, lo); err != nil {
			t.Fatal(err)
		}
		if err := s.SetUpper(1, hi); err != nil {
			t.Fatal(err)
		}
		integers := map[uint32]bool{0: true, 1: true, 2: true}
		interval, found, err := s.ArithmeticFreedom(1, integers)
		if err != nil || !found || interval.Multiple.Cmp(big.NewRat(6, 1)) != 0 {
			t.Fatal("LCM across integer base rows", interval, found, err)
		}
		var random z3rand.Generator
		random.SetSeed(19)
		for i := 0; i < 20; i++ {
			updated, err := s.ArithmeticRandomUpdate(1, integers, &random)
			if err != nil || !updated {
				t.Fatal(updated, err)
			}
			value := s.Value(1)
			if value.Cmp(&lo) < 0 || value.Cmp(&hi) > 0 || !value.Main.IsInt() || value.Epsilon.Sign() != 0 {
				t.Fatal("open negative integer interval", epsilon, value)
			}
			for _, column := range []uint32{0, 2} {
				base := s.Value(column)
				if !base.Main.IsInt() || base.Epsilon.Sign() != 0 || s.outside(column) {
					t.Fatal("LCM movement lost integer base", column, base)
				}
			}
		}
	}
}
