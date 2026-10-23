package simplex

import (
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestArithmeticIntegerPatchRespectsLCMAndOpenBounds(t *testing.T) {
	s := NewSolver(nil)
	if _, err := s.AddRationalRow(0, []uint32{0, 1}, integerRow(2, -1)); err != nil {
		t.Fatal(err)
	}
	lo, hi := NewInfinitesimal(big.NewRat(0, 1), big.NewRat(1, 1)), NewInfinitesimal(big.NewRat(3, 1), big.NewRat(-1, 1))
	if err := s.SetLower(1, lo); err != nil {
		t.Fatal(err)
	}
	if err := s.SetUpper(1, hi); err != nil {
		t.Fatal(err)
	}
	s.SetValue(1, RationalInfinitesimal(big.NewRat(3, 2)))
	integers := map[uint32]bool{0: true, 1: true}
	changed, err := s.PatchArithmeticIntegerValues(integers)
	if err != nil || !changed {
		t.Fatal(changed, err)
	}
	x, b := s.Value(1), s.Value(0)
	if x.Main.Cmp(big.NewRat(2, 1)) != 0 || b.Main.Cmp(big.NewRat(1, 1)) != 0 || x.Epsilon.Sign() != 0 || b.Epsilon.Sign() != 0 {
		t.Fatal("integer LCM patch", x, b)
	}
	if s.FixArithmeticIntegerNonBases(integers) {
		t.Fatal("integer nonbases still needed rounding")
	}
	if status, err := s.MakeFeasible(); err != nil || status != lbool.True {
		t.Fatal(status, err)
	}
}

func TestArithmeticIntegerPatchMixedAndNegativeInfinitesimal(t *testing.T) {
	s := NewSolver(nil)
	if err := s.SetLower(0, RationalInfinitesimal(big.NewRat(1, 4))); err != nil {
		t.Fatal(err)
	}
	if err := s.SetUpper(0, RationalInfinitesimal(big.NewRat(3, 4))); err != nil {
		t.Fatal(err)
	}
	s.SetValue(0, RationalInfinitesimal(big.NewRat(1, 2)))
	negative := NewInfinitesimal(big.NewRat(-2, 1), big.NewRat(-1, 1))
	s.SetValue(1, negative)
	integers := map[uint32]bool{1: true}
	changed, err := s.PatchArithmeticIntegerValues(integers)
	if err != nil || changed {
		t.Fatal("source rational-component skip or unpatchable real interval", changed, err)
	}
	if !s.FixArithmeticIntegerNonBases(integers) {
		t.Fatal("negative integer infinitesimal was not floored")
	}
	integer, real := s.Value(1), s.Value(0)
	if integer.Main.Cmp(big.NewRat(-3, 1)) != 0 || integer.Epsilon.Sign() != 0 || real.Main.Cmp(big.NewRat(1, 2)) != 0 {
		t.Fatal("mixed nonbase rounding", integer, real)
	}
}

func TestArithmeticIntegerPatchChoosesSourceEndpoints(t *testing.T) {
	for _, kind := range []string{"lower", "upper", "free"} {
		s := NewSolver(nil)
		s.SetValue(0, RationalInfinitesimal(big.NewRat(1, 2)))
		want := int64(0)
		if kind == "lower" {
			if err := s.SetLower(0, RationalInfinitesimal(big.NewRat(-2, 1))); err != nil {
				t.Fatal(err)
			}
			want = -2
		}
		if kind == "upper" {
			if err := s.SetUpper(0, RationalInfinitesimal(big.NewRat(2, 1))); err != nil {
				t.Fatal(err)
			}
			want = 2
		}
		changed, err := s.PatchArithmeticIntegerValues(map[uint32]bool{0: true})
		value := s.Value(0)
		if err != nil || !changed || value.Main.Cmp(big.NewRat(want, 1)) != 0 {
			t.Fatal(kind, value, changed, err)
		}
	}
}
