package simplex

import (
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/util/lbool"
)

func TestArithmeticMaxMinDerivedExtremaAndBestEffort(t *testing.T) {
	s := NewSolver(nil)
	_, err := s.AddRationalRow(2, []uint32{2, 0, 1}, integerRow(1, -1, -1))
	if err != nil {
		t.Fatal(err)
	}
	for _, bound := range []struct {
		variable uint32
		lower    bool
		value    int64
	}{{0, true, 0}, {1, true, 1}, {2, false, 3}} {
		var err error
		if bound.lower {
			err = s.SetLower(bound.variable, RationalInfinitesimal(big.NewRat(bound.value, 1)))
		} else {
			err = s.SetUpper(bound.variable, RationalInfinitesimal(big.NewRat(bound.value, 1)))
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	if status, err := s.MakeFeasible(); err != nil || status != lbool.True {
		t.Fatal(status, err)
	}
	first, err := s.ArithmeticMaxMin(0, true, ArithmeticOptConfig{MaxEfforts: 20})
	if err != nil || first.Status != ArithmeticBestEffort || !first.Changed || first.Value.Main.Cmp(big.NewRat(2, 1)) != 0 {
		t.Fatal("source pivot best effort", first, err)
	}
	if len(first.Bounds) != 0 {
		t.Fatal("best effort produced an optimum explanation", first.Bounds)
	}
	second, err := s.ArithmeticMaxMin(0, true, ArithmeticOptConfig{MaxEfforts: 20})
	if err != nil || second.Status != ArithmeticOptimized || second.HasShared || second.Value.Main.Cmp(big.NewRat(2, 1)) != 0 {
		t.Fatal("exact upper optimum", second, err)
	}
	if len(second.Bounds) != 2 {
		t.Fatal("missing optimized row explanation", second.Bounds)
	}
	for _, use := range second.Bounds {
		coefficient, lower := big.NewRat(1, 1), false
		if use.Column == 1 {
			coefficient, lower = big.NewRat(-1, 1), true
		} else if use.Column != 2 {
			t.Fatal("temporary objective in explanation", use)
		}
		if use.Lower != lower || use.Coefficient.Cmp(coefficient) != 0 {
			t.Fatal("optimized row endpoint or sign", use)
		}
	}
	minimum, err := s.ArithmeticMaxMin(0, false, ArithmeticOptConfig{MaxEfforts: 20})
	if err != nil || minimum.Value.Main.Sign() != 0 {
		t.Fatal("minimum", minimum, err)
	}
	if status, err := s.MakeFeasible(); err != nil || status != lbool.True {
		t.Fatal("optimization damaged feasibility", status, err)
	}
	// Every temporary objective must disappear from the incidence graph.
	for variable := uint32(3); variable < s.NumVariables(); variable++ {
		if len(s.ColumnRows(variable)) != 0 {
			t.Fatal("temporary objective survived", variable)
		}
	}
}

func TestArithmeticMaxMinIntegerGainsAndSharedMovement(t *testing.T) {
	for _, integer := range []bool{false, true} {
		s := NewSolver(nil)
		_, err := s.AddRationalRow(0, []uint32{0, 1}, integerRow(2, -1)) // 2*x = y.
		if err != nil {
			t.Fatal(err)
		}
		if err = s.SetLower(1, RationalInfinitesimal(big.NewRat(0, 1))); err != nil {
			t.Fatal(err)
		}
		if err = s.SetUpper(1, RationalInfinitesimal(big.NewRat(3, 1))); err != nil {
			t.Fatal(err)
		}
		config := ArithmeticOptConfig{MaxEfforts: 20}
		if integer {
			config.IntegerColumns = map[uint32]bool{0: true, 1: true}
		}
		result, err := s.ArithmeticMaxMin(0, true, config)
		if err != nil {
			t.Fatal(err)
		}
		if integer {
			if result.Status != ArithmeticBestEffort || !arithmeticInteger(&result.Value) || result.Value.Main.Cmp(big.NewRat(1, 1)) != 0 {
				t.Fatal("integer coefficient denominator gain", result)
			}
		} else if result.Status != ArithmeticOptimized || result.Value.Main.Cmp(big.NewRat(3, 2)) != 0 {
			t.Fatal("rational optimum", result)
		}
		if integer {
			if len(result.Bounds) != 0 {
				t.Fatal("integer best effort produced a bound explanation")
			}
		} else if len(result.Bounds) != 1 || result.Bounds[0].Column != 1 || result.Bounds[0].Lower || result.Bounds[0].Coefficient.Cmp(big.NewRat(1, 2)) != 0 {
			t.Fatal("normalized rational optimum explanation", result.Bounds)
		}
	}
	s := NewSolver(nil)
	_, err := s.AddRationalRow(0, []uint32{0, 1}, integerRow(1, -2))
	if err != nil {
		t.Fatal(err)
	}
	if err = s.SetLower(1, RationalInfinitesimal(big.NewRat(0, 1))); err != nil {
		t.Fatal(err)
	}
	if err = s.SetUpper(1, RationalInfinitesimal(big.NewRat(2, 1))); err != nil {
		t.Fatal(err)
	}
	result, err := s.ArithmeticMaxMin(0, true, ArithmeticOptConfig{MaxEfforts: 20, SharedColumns: map[uint32]bool{1: true}})
	if err != nil || result.Status != ArithmeticOptimized || !result.HasShared || result.Value.Main.Cmp(big.NewRat(4, 1)) != 0 {
		t.Fatal("shared movement guard", result, err)
	}
	if len(result.Bounds) != 0 {
		t.Fatal("shared optimum produced a logical bound explanation")
	}
	free := NewSolver(nil)
	free.EnsureVariable(0)
	result, err = free.ArithmeticMaxMin(0, true, ArithmeticOptConfig{MaxEfforts: 20})
	if err != nil || result.Status != ArithmeticUnbounded {
		t.Fatal("unbounded objective", result, err)
	}
}

func TestArithmeticMaxMinExplanationPreservesStrictEndpoint(t *testing.T) {
	s := NewSolver(nil)
	_, err := s.AddRationalRow(0, []uint32{0, 1}, integerRow(2, -1))
	if err != nil {
		t.Fatal(err)
	}
	bound := RationalInfinitesimal(big.NewRat(3, 1))
	bound.Epsilon.SetInt64(-1)
	if err = s.SetUpper(1, bound); err != nil {
		t.Fatal(err)
	}
	result, err := s.ArithmeticMaxMin(0, true, ArithmeticOptConfig{MaxEfforts: 20})
	if err == nil && result.Status == ArithmeticBestEffort {
		if len(result.Bounds) != 0 {
			t.Fatal("strict best effort produced an optimum explanation")
		}
		result, err = s.ArithmeticMaxMin(0, true, ArithmeticOptConfig{MaxEfforts: 20})
	}
	if err != nil || result.Status != ArithmeticOptimized || len(result.Bounds) != 1 {
		t.Fatal(result, err)
	}
	use := result.Bounds[0]
	if use.Column != 1 || use.Lower || use.Coefficient.Cmp(big.NewRat(1, 2)) != 0 {
		t.Fatal(use)
	}
	bound.Mul(&use.Coefficient)
	if result.Value.Cmp(&bound) != 0 || result.Value.Epsilon.Cmp(big.NewRat(-1, 2)) != 0 {
		t.Fatal("strict optimum lost infinitesimal coefficient", result.Value, bound)
	}
}
