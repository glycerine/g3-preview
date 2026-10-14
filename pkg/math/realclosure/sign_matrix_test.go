package realclosure

import (
	"math/big"
	"testing"
)

func TestSignDeterminationSourceMatrices(t *testing.T) {
	cases := []struct {
		name                      string
		equal, positive, negative int
		rows                      []string
	}{
		{"zero-positive", 1, 2, 0, []string{"1,1", "0,1"}},
		{"zero-negative", 1, 0, 2, []string{"1,1", "0,-1"}},
		{"positive-negative", 0, 2, 1, []string{"1,1", "1,-1"}},
		{"all-signs", 1, 2, 3, []string{"1,1,1", "0,1,-1", "0,1,1"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m, ok := signDetMatrix(tc.equal, tc.positive, tc.negative)
			if !ok || m.rows != len(tc.rows) || m.cols != len(tc.rows) {
				t.Fatalf("unexpected matrix shape: %dx%d, ok=%v", m.rows, m.cols, ok)
			}
			for i, want := range tc.rows {
				got := ""
				for j := 0; j < m.cols; j++ {
					if j != 0 {
						got += ","
					}
					got += m.at(i, j).String()
				}
				if got != want {
					t.Fatalf("row %d: got %s, want %s", i, got, want)
				}
			}
		})
	}
	for _, counts := range [][3]int{{0, 0, 0}, {4, 0, 0}, {0, 4, 0}, {0, 0, 4}, {-1, 2, 0}} {
		if _, ok := signDetMatrix(counts[0], counts[1], counts[2]); ok {
			t.Fatalf("nondiscriminating counts %v produced a matrix", counts)
		}
	}
}

func TestSignMatrixTensorAndCopyOwnCells(t *testing.T) {
	a, _ := signDetMatrix(1, 2, 0)
	b, _ := signDetMatrix(0, 2, 1)
	a.at(0, 0).Lsh(big.NewInt(1), 70)
	var copied signMatrix
	copied.copyFrom(&a)
	copied.at(0, 0).SetInt64(7)
	if a.at(0, 0).Cmp(new(big.Int).Lsh(big.NewInt(1), 70)) != 0 {
		t.Fatal("matrix copy shared a big integer cell")
	}
	var product signMatrix
	product.tensorProduct(&a, &b)
	if product.rows != 4 || product.cols != 4 || product.at(0, 0).Cmp(a.at(0, 0)) != 0 || product.at(1, 3).Int64() != -1 || product.at(3, 3).Int64() != -1 {
		t.Fatal("tensor product has incorrect row or column layout")
	}
	a.tensorProduct(&a, &b)
	for i := range product.cells {
		if a.cells[i].Cmp(&product.cells[i]) != 0 {
			t.Fatalf("aliased tensor product differs at cell %d", i)
		}
	}
	product.at(0, 0).SetInt64(0)
	if a.at(0, 0).Sign() == 0 {
		t.Fatal("tensor products share a cell")
	}
}
