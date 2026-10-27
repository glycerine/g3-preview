package lp

import (
	"fmt"
	"math"
	"math/big"
	"math/rand/v2"
	"reflect"
	"testing"
)

func lpLUHeading(columns uint32, basis []uint32) []int32 {
	h := make([]int32, columns)
	for i := range h {
		h[i] = -1
	}
	for i, j := range basis {
		h[j] = int32(i)
	}
	return h
}

func TestLPLURowRefinementIndependentOracle(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	rng := rand.New(rand.NewPCG(991317, 381))
	for world := 0; world < 64; world++ {
		t.Run(fmt.Sprint(world), func(t *testing.T) {
			h := newLPPairHarness(t)
			n := 1 + world%8
			b := lpExactIdentity(n)
			basis := make([]uint32, n)
			order := rng.Perm(n)
			columns := rng.Perm(2*n + 1)
			x := make([]*big.Rat, n)
			for i := range b {
				basis[i] = uint32(columns[i])
				x[i] = big.NewRat(int64(rng.IntN(13)-6), 11)
				for j := range b[i] {
					v := int64(rng.IntN(5) - 2)
					if j == order[i] {
						v = int64(30*n + i)
					}
					b[i][j].SetFrac64(v, 7)
				}
			}
			fa := lpFloatLUSource(t, b, basis)
			a := lpExactLUStatic(t, h, b, basis)
			density := math.Inf(1)
			if (world/8)%2 == 0 {
				density = 0
			}
			s := lpFloatLUSettings(density)
			f, err := NewFloatLU(fa, basis, s)
			lpStaticMust(t, err)
			defer f.Close()
			r, err := NewRationalLU(a, basis, s)
			h.must(err)
			p, err := NewPairLU(a, basis, s)
			h.must(err)
			defer func() { h.must(r.Close()); h.must(p.Close()) }()
			rhs := lpExactMatrixVector(b, x, true)
			fy := make([]float64, n)
			fi := NewIndexedFloatVector(uint32(n))
			for i, v := range rhs {
				fy[i], _ = v.Float64()
				if fy[i] != 0 {
					lpStaticMust(t, fi.SetValue(fy[i], uint32(i)))
				}
			}
			lpStaticMust(t, f.SolveYBWithErrorCheck(fy, basis))
			lpStaticMust(t, f.SolveYBWithErrorCheckIndexed(fi, lpLUHeading(fa.ColumnCount(), basis), basis, s))
			for i := range x {
				lpFloatBlockNear(t, fy[i], x[i])
				lpFloatBlockNear(t, fi.Data[i], x[i])
			}
			for _, pair := range []bool{false, true} {
				dense, indexed := lpExactLUVector(h, rhs), lpExactLUVector(h, rhs)
				if pair {
					h.must(p.SolveYBWithErrorCheck(dense.Data, nil))
					h.must(p.SolveYBWithErrorCheckIndexed(indexed, nil, nil, nil))
				} else {
					h.must(r.SolveYBWithErrorCheck(dense.Data, nil))
					h.must(r.SolveYBWithErrorCheckIndexed(indexed, nil, nil, nil))
				}
				lpExactLUCheckVector(t, h, dense.Data, x)
				lpExactLUCheckVector(t, h, indexed.Data, x)
				h.must(dense.Close())
				h.must(indexed.Close())
			}
			h.must(r.Close())
			h.must(p.Close())
			lpExactVectorNamedCensus(t, h, lpExactLUStaticCells(a))
		})
	}
}

func TestLPLURowRefinementRoutingBoundaries(t *testing.T) {
	for _, columns := range []uint32{120, 121, 360, 361} {
		for _, fill := range []bool{false, true} {
			t.Run(fmt.Sprint(columns, "/", fill), func(t *testing.T) {
				b := lpExactIdentity(3)
				basis := []uint32{5, 1, 9}
				want := []*big.Rat{big.NewRat(1, 1), new(big.Rat), new(big.Rat)}
				if fill {
					b[0][1].SetFrac64(1, 4)
					b[0][2].SetFrac64(1, 2)
					b[1][2].SetFrac64(1, 3)
					want[1].SetFrac64(-1, 4)
					want[2].SetFrac64(-5, 12)
				}
				a := lpFloatLUSourceColumns(t, b, basis, columns)
				s := lpFloatLUSettings(math.Inf(1))
				m, err := NewFloatLU(a, basis, s)
				lpStaticMust(t, err)
				defer m.Close()
				y := NewIndexedFloatVector(3)
				lpStaticMust(t, y.SetValue(1, 0))
				lpStaticMust(t, m.SolveYBWithErrorCheckIndexed(y, lpLUHeading(columns, basis), basis, s))
				for i := range want {
					lpFloatBlockNear(t, y.Data[i], want[i])
				}
				initialSparse := uint32(120) < columns
				finalSparse := uint32(120) < columns
				if fill {
					finalSparse = 360 < columns
				}
				if (m.ii.DataSize() != 0) != (initialSparse && finalSparse) {
					t.Fatal("route markers", columns, fill, m.ii.DataSize())
				}
				if !finalSparse || !initialSparse {
					if len(m.yCopy.Index) != 0 {
						t.Fatal("dense copy index cleanup")
					}
					for _, v := range m.yCopy.Data {
						if v != 0 {
							t.Fatal("dense copy data cleanup")
						}
					}
				}
			})
		}
	}
	for _, columns := range []uint32{30, 31} {
		for _, pair := range []bool{false, true} {
			t.Run(fmt.Sprint("exact/", columns, "/", pair), func(t *testing.T) {
				h := newLPPairHarness(t)
				a := NewRationalStaticMatrix(h.m, 3, columns)
				defer func() { h.must(a.Close()) }()
				one := h.pair("1", "0")
				basis := []uint32{5, 1, 9}
				for i, j := range basis {
					h.must(a.Set(uint32(i), j, &one.X))
				}
				var r *RationalLU
				var p *PairLU
				var err error
				if pair {
					p, err = NewPairLU(a, basis, lpFloatLUSettings(math.Inf(1)))
				} else {
					r, err = NewRationalLU(a, basis, lpFloatLUSettings(math.Inf(1)))
				}
				h.must(err)
				y := NewIndexedRationalVector(h.m, 3)
				h.must(y.SetValue(&one.X, 0))
				defer func() {
					if r != nil {
						h.must(r.Close())
					}
					if p != nil {
						h.must(p.Close())
					}
					h.must(y.Close())
				}()
				if pair {
					h.must(p.SolveYBWithErrorCheckIndexed(y, nil, nil, nil))
				} else {
					h.must(r.SolveYBWithErrorCheckIndexed(y, nil, nil, nil))
				}
				lpExactLUCheckVector(t, h, y.Data, []*big.Rat{big.NewRat(1, 1), new(big.Rat), new(big.Rat)})
				if !reflect.DeepEqual(y.Index, []uint32{0}) {
					t.Fatal(y.Index)
				}
				if pair {
					h.must(p.Close())
				} else {
					h.must(r.Close())
				}
				h.must(y.Close())
				lpExactVectorNamedCensus(t, h, lpExactLUStaticCells(a))
			})
		}
	}
	if !luSparseColumnRoute(1<<31, 1, 30) {
		t.Fatal("source unsigned ratio wrap")
	}
}

func TestLPLURowRefinementOneCorrection(t *testing.T) {
	for _, columns := range []uint32{120, 121, 361} {
		for _, fill := range []bool{false, true} {
			t.Run(fmt.Sprint(columns, "/", fill), func(t *testing.T) {
				basis := []uint32{5, 1, 9}
				a := lpFloatLUSourceColumns(t, lpExactIdentity(3), basis, columns)
				lpStaticMust(t, a.Set(0, columns-1, 999))
				m, err := NewFloatLU(a, basis, lpFloatLUSettings(math.Inf(1)))
				lpStaticMust(t, err)
				defer m.Close()
				// Inject a known approximate inverse H. For B=I, one correction
				// returns rhs*(2H-H^2), rather than silently iterating to convergence.
				if fill {
					eta := NewFloatRowEtaMatrix(0, 0, 3)
					lpStaticMust(t, eta.Append(1, 0.25))
					lpStaticMust(t, eta.Append(2, 0.5))
					m.Tail = append(m.Tail, eta)
				}
				m.Tail = append(m.Tail, NewFloatDiagonalMatrix(0, 2))
				dense := []float64{4, 0, 0}
				indexed := NewIndexedFloatVector(3)
				lpStaticMust(t, indexed.SetValue(4, 0))
				lpStaticMust(t, m.SolveYBWithErrorCheck(dense, basis))
				lpStaticMust(t, m.SolveYBWithErrorCheckIndexed(indexed, lpLUHeading(columns, basis), basis, m.Settings))
				want := []*big.Rat{big.NewRat(3, 1), new(big.Rat), new(big.Rat)}
				if fill {
					want[1].SetFrac64(1, 4)
					want[2].SetFrac64(1, 2)
				}
				for i := range want {
					lpFloatBlockNear(t, dense[i], want[i])
					lpFloatBlockNear(t, indexed.Data[i], want[i])
				}
				if luSparseColumnRoute(1, columns, 120) && luSparseColumnRoute(uint32(len(indexed.Index)), columns, 120) && len(m.yCopy.Index) == 0 {
					t.Fatal("sparse correction unexpectedly cleared")
				}
			})
		}
	}
}

func TestLPLURowRefinementIndependentResidualsAndUnion(t *testing.T) {
	rng := rand.New(rand.NewPCG(7381, 511))
	const n = 7
	const columns = 15
	for world := 0; world < 1200; world++ {
		a := NewFloatStaticMatrix(n, columns)
		grid := make([][]*big.Rat, n)
		for i := range grid {
			grid[i] = make([]*big.Rat, columns)
			for j := range grid[i] {
				grid[i][j] = big.NewRat(int64(rng.IntN(9)-4), 4)
				v, _ := grid[i][j].Float64()
				lpStaticMust(t, a.Set(uint32(i), uint32(j), v))
			}
		}
		order := rng.Perm(columns)
		basis := make([]uint32, n)
		for i := range basis {
			basis[i] = uint32(order[i])
		}
		m := &FloatLU{Dim: n, A: a, yCopy: NewIndexedFloatVector(n), ii: NewIndexedUintVector(0)}
		y := NewIndexedFloatVector(n)
		rhs := make([]*big.Rat, n)
		for _, i := range rng.Perm(n) {
			v := float64(rng.IntN(9)-4) / 4
			y.Data[i] = v
			if v != 0 {
				y.Index = append(y.Index, uint32(i))
			}
		}
		for _, i := range rng.Perm(n) {
			rhs[i] = big.NewRat(int64(rng.IntN(9)-4), 4)
			v, _ := rhs[i].Float64()
			m.yCopy.Data[i] = v
			if v != 0 {
				m.yCopy.Index = append(m.yCopy.Index, uint32(i))
			}
		}
		want := make([]*big.Rat, n)
		for j := range want {
			want[j] = new(big.Rat).Set(rhs[j])
			for i := range grid {
				want[j].Sub(want[j], new(big.Rat).Mul(grid[i][basis[j]], new(big.Rat).SetFloat64(y.Data[i])))
			}
			if new(big.Rat).Abs(want[j]).Cmp(big.NewRat(1, 2)) < 0 {
				want[j].SetInt64(0)
			}
		}
		lpStaticMust(t, m.findIndexedRowError(y, lpLUHeading(columns, basis), &Settings{DropTolerance: 0.5}))
		for i := range want {
			lpFloatBlockNear(t, m.yCopy.Data[i], want[i])
		}
		lpColumnIndexCheck(t, m.yCopy.Index, want, false)
		// Check the independent dense residual before applying a correction.
		dense := make([]float64, n)
		for i := range rhs {
			dense[i], _ = rhs[i].Float64()
		}
		lpStaticMust(t, m.findRowError(dense, y.Data, basis))
		for j := range dense {
			v := new(big.Rat).Set(rhs[j])
			for i := range grid {
				v.Sub(v, new(big.Rat).Mul(grid[i][basis[j]], new(big.Rat).SetFloat64(y.Data[i])))
			}
			lpFloatBlockNear(t, dense[j], v)
		}
		original := append([]float64(nil), y.Data...)
		oldIndex := append([]uint32(nil), y.Index...)
		deltaIndex := append([]uint32(nil), m.yCopy.Index...)
		for _, i := range deltaIndex {
			original[i] += m.yCopy.Data[i]
		}
		union := append([]uint32(nil), oldIndex...)
		seen := map[uint32]bool{}
		for _, i := range union {
			seen[i] = true
		}
		for _, i := range deltaIndex {
			if !seen[i] {
				seen[i] = true
				union = append(union, i)
			}
		}
		var indexWant []uint32
		for _, i := range union {
			if math.Abs(original[i]) >= 1e-14 {
				indexWant = append(indexWant, i)
			} else if original[i] != 0 {
				original[i] = 0
			}
		}
		lpStaticMust(t, m.addIndexedRowCorrection(y))
		if !reflect.DeepEqual(append([]uint32(nil), y.Index...), indexWant) || !reflect.DeepEqual(y.Data, original) {
			t.Fatal("correction union", world, y.Index, indexWant)
		}
	}
}

func TestLPLURowRefinementFixedCleanupAndPrefix(t *testing.T) {
	m := &FloatLU{yCopy: NewIndexedFloatVector(5), ii: NewIndexedUintVector(0)}
	y := NewIndexedFloatVector(5)
	y.Data = []float64{2, math.Copysign(0, -1), 1e-15, 7, 1e-14}
	y.Index = []uint32{0, 1, 2, 4}
	m.yCopy.Data = []float64{-2, 0, 0, 3, 0}
	m.yCopy.Index = []uint32{0, 3}
	lpStaticMust(t, m.addIndexedRowCorrection(y))
	if !reflect.DeepEqual(y.Index, []uint32{4, 3}) || y.Data[0] != 0 || y.Data[2] != 0 || !math.Signbit(y.Data[1]) || y.Data[3] != 10 {
		t.Fatal("fixed cleanup", y.Data, y.Index)
	}
	m.yCopy.Index = []uint32{3, 5}
	m.yCopy.Data[3] = 1
	old := append([]uint32(nil), y.Index...)
	if err := m.addIndexedRowCorrection(y); err != ErrBounds || y.Data[3] != 11 || !reflect.DeepEqual(y.Index, old) {
		t.Fatal("add prefix", err, y.Data, y.Index)
	}
	// Duplicate old indices assert after the first marker is published.
	m.yCopy.Index = nil
	y.Index = []uint32{3, 3}
	err := m.addIndexedRowCorrection(y)
	if err != sourceAssertionFailure() {
		t.Fatal(err)
	}
	if err == nil && !reflect.DeepEqual(y.Index, []uint32{3, 3}) {
		t.Fatal("native duplicate markers", y.Index)
	}
}

func TestLPLURowRefinementLiteralTolerance(t *testing.T) {
	for _, drop := range []float64{0, math.NaN()} {
		a := NewFloatStaticMatrix(1, 1)
		lpStaticMust(t, a.Set(0, 0, 1))
		m := &FloatLU{Dim: 1, A: a, yCopy: NewIndexedFloatVector(1), ii: NewIndexedUintVector(0)}
		y := NewIndexedFloatVector(1)
		lpStaticMust(t, y.SetValue(1, 0))
		lpStaticMust(t, m.yCopy.SetValue(1, 0))
		lpStaticMust(t, m.findIndexedRowError(y, []int32{0}, &Settings{DropTolerance: drop}))
		if len(m.yCopy.Index) != 1 || m.yCopy.Index[0] != 0 || m.yCopy.Data[0] != 0 {
			t.Fatal("zero residual under strict/NaN tolerance")
		}
	}
	m := &FloatLU{yCopy: NewIndexedFloatVector(2), ii: NewIndexedUintVector(0)}
	y := NewIndexedFloatVector(2)
	y.Data = []float64{math.NaN(), math.Inf(1)}
	y.Index = []uint32{0, 1}
	lpStaticMust(t, m.addIndexedRowCorrection(y))
	if len(y.Index) != 2 || !math.IsNaN(y.Data[0]) || !math.IsInf(y.Data[1], 1) {
		t.Fatal("IEEE fixed cleanup")
	}
}

func TestLPLURowRefinementResidualOutsideSolutionSupport(t *testing.T) {
	a := NewFloatStaticMatrix(2, 2)
	lpStaticMust(t, a.Set(0, 0, 1))
	lpStaticMust(t, a.Set(1, 1, 1))
	m := &FloatLU{Dim: 2, A: a, yCopy: NewIndexedFloatVector(2), ii: NewIndexedUintVector(0)}
	y := NewIndexedFloatVector(2)
	lpStaticMust(t, y.SetValue(2, 0))
	lpStaticMust(t, m.yCopy.SetValue(2, 0))
	lpStaticMust(t, m.yCopy.SetValue(7, 1))
	lpStaticMust(t, m.findIndexedRowError(y, []int32{0, 1}, lpFloatLUSettings(math.Inf(1))))
	if !reflect.DeepEqual(m.ii.Index, []uint32{0, 1}) || !reflect.DeepEqual(m.yCopy.Index, []uint32{1}) || m.yCopy.Data[0] != 0 || m.yCopy.Data[1] != 7 {
		t.Fatal("RHS-only residual lost", m.ii.Index, m.yCopy.Index, m.yCopy.Data)
	}
}

func TestLPLURowRefinementAfterColumnUpdates(t *testing.T) {
	rng := rand.New(rand.NewPCG(9937, 7713))
	for world := 0; world < 24; world++ {
		n := 1 + world%6
		b := lpExactIdentity(n)
		basis := make([]uint32, n)
		for i := range b {
			basis[i] = uint32(2 + 3*i)
			for j := range b[i] {
				v := int64(rng.IntN(5) - 2)
				if i == j {
					v = int64(20*n + i)
				}
				b[i][j].SetFrac64(v, 4)
			}
		}
		a := lpFloatLUSourceColumns(t, b, basis, 1001)
		density := math.Inf(1)
		if (world/6)%2 == 0 {
			density = 0
		}
		s := lpFloatLUSettings(density)
		m, err := NewFloatLU(a, basis, s)
		lpStaticMust(t, err)
		w := NewIndexedFloatVector(uint32(n))
		for step := 0; step < 4; step++ {
			leaving := rng.IntN(n)
			comb := make([]*big.Rat, n)
			for i := range comb {
				comb[i] = big.NewRat(int64(rng.IntN(5)-2), 8)
			}
			comb[leaving].SetInt64(1)
			column := lpExactMatrixVector(b, comb, false)
			entering := uint32(32 + step)
			for i := range column {
				v, _ := column[i].Float64()
				lpStaticMust(t, a.Set(uint32(i), entering, v))
			}
			lpStaticMust(t, m.PrepareEntering(entering, w))
			lpStaticMust(t, m.ReplaceColumn(0, w, uint32(leaving)))
			basis[leaving] = entering
			for i := range b {
				b[i][leaving] = column[i]
			}
			x := make([]*big.Rat, n)
			for i := range x {
				x[i] = new(big.Rat)
			}
			x[rng.IntN(n)].SetFrac64(3, 4)
			rhs := lpExactMatrixVector(b, x, true)
			dense := make([]float64, n)
			indexed := NewIndexedFloatVector(uint32(n))
			for i := range rhs {
				dense[i], _ = rhs[i].Float64()
				if dense[i] != 0 {
					lpStaticMust(t, indexed.SetValue(dense[i], uint32(i)))
				}
			}
			lpStaticMust(t, m.SolveYBWithErrorCheck(dense, basis))
			lpStaticMust(t, m.SolveYBWithErrorCheckIndexed(indexed, lpLUHeading(1001, basis), basis, s))
			for i := range x {
				lpFloatBlockNear(t, dense[i], x[i])
				lpFloatBlockNear(t, indexed.Data[i], x[i])
			}
		}
		lpStaticMust(t, m.Close())
	}
}
