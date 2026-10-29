package hilbert

import (
	"errors"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/util/lbool"
)

func rationals(values ...int64) []big.Rat {
	result := make([]big.Rat, len(values))
	for i, value := range values {
		result[i].SetInt64(value)
	}
	return result
}

func rational(value int64) *big.Rat {
	return new(big.Rat).SetInt64(value)
}

func validateBasis(t *testing.T, basis *Basis) {
	t.Helper()
	for i := uint32(0); i < basis.BasisSize(); i++ {
		solution, initial, err := basis.BasisSolution(i)
		if err != nil {
			t.Fatalf("basis solution %d: %v", i, err)
		}
		for j := range basis.inequalities {
			coefficients, bound, equal, err := basis.Inequality(uint32(j))
			if err != nil {
				t.Fatalf("inequality %d: %v", j, err)
			}
			var sum big.Rat
			for k := range solution {
				var product big.Rat
				product.Mul(&coefficients[k], &solution[k])
				sum.Add(&sum, &product)
			}
			if initial {
				if equal && sum.Cmp(&bound) != 0 || !equal && sum.Cmp(&bound) < 0 {
					t.Fatalf("initial basis solution %d violates inequality %d: sum=%s bound=%s equal=%v", i, j, &sum, &bound, equal)
				}
			} else if equal && sum.Sign() != 0 || !equal && sum.Sign() < 0 {
				t.Fatalf("homogeneous basis solution %d violates inequality %d: sum=%s equal=%v", i, j, &sum, equal)
			}
		}
	}
}

func TestBasisSourceExamples(t *testing.T) {
	tests := []struct {
		name string
		add  func(*testing.T, *Basis)
	}{
		{
			name: "Ajili-Contejean-example-9",
			add: func(t *testing.T, basis *Basis) {
				t.Helper()
				if err := basis.AddEQ(rationals(1, 1, -2), rational(0)); err != nil {
					t.Fatal(err)
				}
				if err := basis.AddEQ(rationals(1, 0, -1), rational(0)); err != nil {
					t.Fatal(err)
				}
				if err := basis.AddLE(rationals(0, 1, -1), rational(0)); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "affine-equalities",
			add: func(t *testing.T, basis *Basis) {
				t.Helper()
				if err := basis.AddEQ(rationals(0, 1), rational(0)); err != nil {
					t.Fatal(err)
				}
				if err := basis.AddEQ(rationals(1, -1), rational(2)); err != nil {
					t.Fatal(err)
				}
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			basis := New(nil)
			test.add(t, basis)
			result, err := basis.Saturate()
			if err != nil || result != lbool.True {
				t.Fatalf("Saturate() = %v, %v", result, err)
			}
			if basis.BasisSize() == 0 {
				t.Fatal("expected non-empty basis")
			}
			validateBasis(t, basis)
		})
	}
}

func TestBasisIntegerCoordinatesAndCancellation(t *testing.T) {
	basis := New(func() bool { return false })
	basis.SetIsInt(0)
	if !basis.IsInt(0) {
		t.Fatal("integer coordinate was not retained")
	}
	if err := basis.AddGE(rationals(1), rational(0)); err != nil {
		t.Fatal(err)
	}
	result, err := basis.Saturate()
	if err != nil || result != lbool.Undef {
		t.Fatalf("cancelled Saturate() = %v, %v", result, err)
	}
}

func TestBasisRefusesInvalidNumeralsAndDimensions(t *testing.T) {
	basis := New(nil)
	if err := basis.AddGE([]big.Rat{*big.NewRat(1, 2)}, rational(0)); !errors.Is(err, ErrOverflow) {
		t.Fatalf("fractional coefficient error = %v", err)
	}
	if err := basis.AddGE(rationals(1, 2), rational(0)); err != nil {
		t.Fatal(err)
	}
	if err := basis.AddGE(rationals(1), rational(0)); !errors.Is(err, ErrDimension) {
		t.Fatalf("dimension error = %v", err)
	}
}
