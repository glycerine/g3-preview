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
	second, err := s.ArithmeticMaxMin(0, true, ArithmeticOptConfig{MaxEfforts: 20})
	if err != nil || second.Status != ArithmeticOptimized || second.HasShared || second.Value.Main.Cmp(big.NewRat(2, 1)) != 0 {
		t.Fatal("exact upper optimum", second, err)
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
	free := NewSolver(nil)
	free.EnsureVariable(0)
	result, err = free.ArithmeticMaxMin(0, true, ArithmeticOptConfig{MaxEfforts: 20})
	if err != nil || result.Status != ArithmeticUnbounded {
		t.Fatal("unbounded objective", result, err)
	}
}
