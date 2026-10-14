package realclosure

import (
	"math/big"
	"reflect"
	"testing"
)

func TestSignMatrixSourceIntegerSolutions(t *testing.T) {
	cases := []struct {
		name                      string
		equal, positive, negative int
		right                     []int
		want                      []int
	}{
		{"zero-positive", 1, 1, 0, []int{5, 3}, []int{2, 3}},
		{"zero-negative", 1, 0, 1, []int{5, -3}, []int{2, 3}},
		{"positive-negative", 0, 1, 1, []int{5, -1}, []int{2, 3}},
		{"all-signs", 1, 1, 1, []int{6, -1, 5}, []int{1, 2, 3}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			matrix, _ := signDetMatrix(tc.equal, tc.positive, tc.negative)
			originalRight := append([]int(nil), tc.right...)
			got, ok := matrix.solveInt(tc.right)
			if !ok || !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("solve: got %v, ok=%v; want %v", got, ok, tc.want)
			}
			if matrix.at(0, 0).Int64() != 1 || !reflect.DeepEqual(tc.right, originalRight) {
				t.Fatal("solve changed an input")
			}
		})
	}
	m, _ := signDetMatrix(0, 1, 1)
	if got, ok := m.solveInt([]int{1, 0}); ok || got != nil {
		t.Fatalf("noninteger solution accepted: %v", got)
	}
	m.make(2, 2)
	m.setSmall(1, 1, 1, 1)
	if _, ok := m.solveInt([]int{2, 2}); ok {
		t.Fatal("singular matrix accepted")
	}
	m.setSmall(2, 0, 0, 2)
	if _, ok := m.solveInt([]int{3, 4}); ok {
		t.Fatal("nondivisible row gcd accepted")
	}
	if got, ok := m.solveInt([]int{4, 6}); !ok || !reflect.DeepEqual(got, []int{2, 3}) {
		t.Fatalf("divisible row gcd: %v, %v", got, ok)
	}
}

func TestSignMatrixFilterColumnsOwnsValues(t *testing.T) {
	source := signMatrix{}
	source.make(2, 3)
	source.setSmall(1, 2, 3, 4, 5, 6)
	source.at(1, 2).Lsh(big.NewInt(1), 70)
	var selected signMatrix
	selected.filterColumns(&source, []int{0, 2})
	if selected.rows != 2 || selected.cols != 2 || selected.at(0, 0).Int64() != 1 || selected.at(0, 1).Int64() != 3 || selected.at(1, 0).Int64() != 4 || selected.at(1, 1).Cmp(source.at(1, 2)) != 0 {
		t.Fatal("filtered matrix has wrong columns")
	}
	selected.at(1, 1).SetInt64(9)
	if source.at(1, 2).Cmp(new(big.Int).Lsh(big.NewInt(1), 70)) != 0 {
		t.Fatal("filtered result aliases source cell")
	}
	source.filterColumns(&source, []int{0, 2})
	if source.cols != 2 || source.at(1, 1).Cmp(new(big.Int).Lsh(big.NewInt(1), 70)) != 0 {
		t.Fatal("aliased column filter lost source values")
	}
	selected.filterColumns(&source, []int{0, 1})
	if selected.at(1, 1).Cmp(source.at(1, 1)) != 0 {
		t.Fatal("full-column copy lost source value")
	}
}

func TestSignMatrixIntegerSolverSignedTwoByTwoGrid(t *testing.T) {
	// The independent determinant check selects every nonsingular matrix in
	// this bounded grid. The right-hand side comes from a known integer root.
	for a := int64(-2); a <= 2; a++ {
		for b := int64(-2); b <= 2; b++ {
			for c := int64(-2); c <= 2; c++ {
				for d := int64(-2); d <= 2; d++ {
					if a*d == b*c {
						continue
					}
					var m signMatrix
					m.make(2, 2)
					m.setSmall(a, b, c, d)
					right := []int{int(3*a - 2*b), int(3*c - 2*d)}
					got, ok := m.solveInt(right)
					if !ok || !reflect.DeepEqual(got, []int{3, -2}) {
						t.Fatalf("matrix [%d %d; %d %d]: got %v, ok=%v", a, b, c, d, got, ok)
					}
				}
			}
		}
	}
}

func TestSignMatrixIndependentRowsPreserveOriginalPriority(t *testing.T) {
	var source signMatrix
	source.make(3, 2)
	source.setSmall(0, 1, 1, 0, 1, 1)
	var selected signMatrix
	indices := selected.independentRows(&source)
	if !reflect.DeepEqual(indices, []int{0, 1}) || selected.rows != 2 || selected.cols != 2 || selected.at(0, 0).Int64() != 0 || selected.at(0, 1).Int64() != 1 || selected.at(1, 0).Int64() != 1 || selected.at(1, 1).Int64() != 0 {
		t.Fatalf("unexpected selected rows %v", indices)
	}
	selected.at(0, 1).SetInt64(99)
	if source.at(0, 1).Int64() != 1 {
		t.Fatal("selected row shares source cell")
	}
	source.setSmall(1, 0, 2, 0, 0, 1)
	indices = source.independentRows(&source)
	if !reflect.DeepEqual(indices, []int{0, 2}) || source.rows != 2 || source.at(0, 0).Int64() != 1 || source.at(1, 1).Int64() != 1 {
		t.Fatalf("aliased independent-row selection got %v", indices)
	}
}
