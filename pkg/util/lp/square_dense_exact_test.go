package lp

import (
	"fmt"
	"math/big"
	"math/rand/v2"
	"testing"

	"github.com/glycerine/g3/pkg/util/numeral"
)

func lpExactDenseCells(m *RationalSquareDenseSubmatrix) []*numeral.Rational {
	out := lpSparseEtaOwningCells(m.Parent, nil)
	for i := range m.Data {
		out = append(out, &m.Data[i])
	}
	for i := range m.WorkVector.Data {
		out = append(out, &m.WorkVector.Data[i])
	}
	return out
}

func TestLPExactDenseBlockIndependentEliminationStages(t *testing.T) {
	rng := rand.New(rand.NewPCG(13571, 87233))
	swaps := 0
	for world := 0; world < 120; world++ {
		t.Run(fmt.Sprint(world), func(t *testing.T) {
			h := newLPPairHarness(t)
			value := h.pair("0", "0")
			start, dim := uint32(world%3), uint32(1+rng.IntN(5))
			n := start + dim
			parent := lpNewExactSparse(t, h, n)
			rp, cp, ci := make([]uint32, n), make([]uint32, n), make([]uint32, n)
			for i, j := range rng.Perm(int(n)) {
				rp[i] = uint32(j)
			}
			for i, j := range rng.Perm(int(n)) {
				cp[i] = uint32(j)
				ci[j] = uint32(i)
			}
			var err error
			parent.RowPermutation, err = PermutationFromValues(n, rp)
			h.must(err)
			parent.ColumnPermutation, err = PermutationFromValues(n, cp)
			h.must(err)
			a, e := lpExactIdentity(int(n)), lpExactIdentity(int(n))
			order := rng.Perm(int(dim))
			for i := start; i < n; i++ {
				for j := start; j < n; j++ {
					a[i][j].SetFrac64(int64(rng.IntN(5)-2), 13)
					if uint32(order[j-start]) == i-start {
						a[i][j].SetFrac64(int64(25+i), 7)
					}
				}
			}
			for i := uint32(0); i < n; i++ {
				for _, jj := range rng.Perm(int(n)) {
					j := uint32(jj)
					if (i < start && i != j) || (i >= start && j < start) {
						continue
					}
					if a[i][j].Sign() == 0 && rng.IntN(3) != 0 {
						continue
					}
					h.must(h.m.SetString(&value.X, a[i][j].RatString()))
					h.must(parent.AddNewElement(rp[i], ci[j], &value.X))
				}
			}
			m, err := NewRationalSquareDenseSubmatrix(parent, start)
			h.must(err)
			if !m.IsDense() {
				t.Fatal("dense marker")
			}
			for i := start; i < n; i++ {
				best := i
				for j := i + 1; j < n; j++ {
					if new(big.Rat).Abs(a[i][j]).Cmp(new(big.Rat).Abs(a[i][best])) > 0 {
						best = j
					}
				}
				got, err := m.FindPivotColumn(i)
				h.must(err)
				if got != int32(best) {
					t.Fatalf("pivot %d want %d", got, best)
				}
				if best != i {
					swaps++
					h.must(parent.SwapColumns(i, best))
					h.must(m.SwapColumns(i, best))
					for k := range a {
						a[k][i], a[k][best] = a[k][best], a[k][i]
					}
				} else {
					h.must(m.SwapColumns(i, i))
				}
				pivot := new(big.Rat).Set(a[i][i])
				for j := uint32(0); j < n; j++ {
					a[i][j].Quo(a[i][j], pivot)
					e[i][j].Quo(e[i][j], pivot)
				}
				for row := i + 1; row < n; row++ {
					mult := new(big.Rat).Set(a[row][i])
					for j := uint32(0); j < n; j++ {
						a[row][j].Sub(a[row][j], new(big.Rat).Mul(mult, a[i][j]))
						e[row][j].Sub(e[row][j], new(big.Rat).Mul(mult, e[i][j]))
					}
				}
				// Precise traits must ignore even a huge floating drop threshold.
				h.must(m.Pivot(i, &Settings{DropTolerance: 1e100}))
				for row := start; row < n; row++ {
					for col := start; col < n; col++ {
						want := a[row][col]
						if col <= i && col <= row {
							want = e[row][col]
						}
						v, err := m.cell(row, col)
						h.must(err)
						lpSparseRationalCheck(t, h, v, want)
					}
				}
				lpExactVectorNamedCensus(t, h, lpExactDenseCells(m))
			}
			h.must(m.UpdateParent(&Settings{DropTolerance: 1e100}))
			wantParent := make(map[IndexPair]*big.Rat)
			for row := uint32(0); row < n; row++ {
				pr, err := parent.AdjustRow(row)
				h.must(err)
				for col := uint32(0); col < n; col++ {
					want := new(big.Rat)
					if row < start {
						if row == col {
							want.SetInt64(1)
						}
					} else if col >= row {
						want.Set(a[row][col])
					}
					pc, err := parent.AdjustColumn(col)
					h.must(err)
					v, err := parent.Get(row, col)
					h.must(err)
					lpSparseRationalCheck(t, h, v, want)
					if want.Sign() != 0 {
						wantParent[IndexPair{pr, pc}] = want
					}
					if row >= start && col >= start {
						lower := new(big.Rat)
						if col <= row {
							lower.Set(e[row][col])
						}
						v, err := m.cell(row, col)
						h.must(err)
						lpSparseRationalCheck(t, h, v, lower)
					}
				}
				if row >= start {
					max := new(big.Rat)
					for k := range parent.Rows[pr] {
						v := &parent.Rows[pr][k].Value
						text, err := h.m.String(v)
						h.must(err)
						r, _ := new(big.Rat).SetString(text)
						r.Abs(r)
						if k == 0 {
							max.Set(r)
						} else if r.Cmp(max) > 0 {
							t.Fatal("maximum not first")
						}
					}
				}
			}
			lpExactVectorNamedCensus(t, h, lpExactDenseCells(m))
			lpExactDenseCheckApplications(t, h, m, e, rng)
			h.must(m.Close())
			lpExactSparseCheck(t, h, parent, wantParent)
			h.must(parent.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
	if swaps == 0 {
		t.Fatal("no exact dense column swaps")
	}
}

func TestLPExactDenseBlockLiteralCopiesAndTinyValues(t *testing.T) {
	h := newLPPairHarness(t)
	parent := lpNewExactSparse(t, h, 2)
	p := h.pair("2", "1/1000000000000000000000000000000")
	h.must(parent.AddNewElement(0, 0, &p.X))
	h.must(parent.AddNewElement(0, 1, &p.Y))
	h.must(h.m.SetInt64(&p.X, 3))
	h.must(parent.AddNewElement(1, 0, &p.X))
	h.must(h.m.SetInt64(&p.X, 4))
	h.must(parent.AddNewElement(1, 1, &p.X))
	m, err := NewRationalSquareDenseSubmatrix(parent, 0)
	h.must(err)
	h.must(m.Pivot(0, nil))
	tiny, _ := new(big.Rat).SetString("1/2000000000000000000000000000000")
	lpSparseRationalCheck(t, h, &m.Data[0], big.NewRat(1, 2))
	lpSparseRationalCheck(t, h, &m.Data[1], tiny)
	lpSparseRationalCheck(t, h, &m.Data[2], big.NewRat(-3, 2))
	lpSparseRationalCheck(t, h, &m.Data[3], new(big.Rat).Sub(big.NewRat(4, 1), new(big.Rat).Mul(tiny, big.NewRat(3, 1))))
	// Dense storage is independent of its borrowed parent's coefficient cells.
	lpSparseRationalCheck(t, h, &parent.Rows[0][0].Value, big.NewRat(2, 1))
	h.must(m.Close())
	h.must(h.m.SetInt64(&p.X, -2))
	h.must(parent.AddNewElement(0, 0, &p.X))
	m, err = NewRationalSquareDenseSubmatrix(parent, 0)
	h.must(err)
	lpSparseRationalCheck(t, h, &m.Data[0], big.NewRat(-2, 1))
	// Equal absolute maxima retain the first logical column.
	h.must(h.m.SetInt64(&m.Data[1], 2))
	j, err := m.FindPivotColumn(0)
	h.must(err)
	if j != 0 {
		t.Fatal("exact stable tie", j)
	}
	h.must(h.m.SetInt64(&m.Data[2], 0))
	h.must(h.m.SetInt64(&m.Data[3], 0))
	j, err = m.FindPivotColumn(1)
	h.must(err)
	if j != -1 {
		t.Fatal("all-zero pivot row", j)
	}
	for _, f := range []func(uint32) (uint32, error){m.AdjustColumn, m.AdjustColumnInverse, m.AdjustRow, m.AdjustRowInverse} {
		j, err := f(7)
		h.must(err)
		if j != 7 {
			t.Fatal("identity fallback", j)
		}
	}
	h.must(m.Close())
	h.must(parent.Close())
	lpExactVectorNamedCensus(t, h, nil)
}
