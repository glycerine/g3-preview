package lp

import (
	"fmt"
	"math"
	"math/big"
	"math/rand/v2"
	"reflect"
	"testing"
)

func lpFloatBlockNear(t *testing.T, got float64, want *big.Rat) {
	t.Helper()
	w, _ := want.Float64()
	if math.IsNaN(got) || math.Abs(got-w) > 2e-12*(1+math.Abs(w)) {
		t.Fatalf("got %.17g want %s", got, want)
	}
}

func TestLPFloatDenseBlockIndependentFactorization(t *testing.T) {
	rng := rand.New(rand.NewPCG(38117, 79191))
	swaps := 0
	for world := 0; world < 180; world++ {
		t.Run(fmt.Sprint(world), func(t *testing.T) {
			start, dim := uint32(world%3), uint32(1+rng.IntN(6))
			n := start + dim
			parent := newFloatSparseMatrix(n)
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
			lpStaticMust(t, err)
			parent.ColumnPermutation, err = PermutationFromValues(n, cp)
			lpStaticMust(t, err)
			a, e := lpExactIdentity(int(n)), lpExactIdentity(int(n))
			order := rng.Perm(int(dim))
			for i := start; i < n; i++ {
				for j := start; j < n; j++ {
					x := int64(rng.IntN(5) - 2)
					if uint32(order[j-start]) == i-start {
						x = 25 + int64(i)
					}
					a[i][j].SetInt64(x)
				}
			}
			for i := uint32(0); i < n; i++ {
				for _, jj := range rng.Perm(int(n)) {
					j := uint32(jj)
					if i < start && i != j {
						continue
					}
					if i >= start && j < start {
						continue
					}
					x, _ := a[i][j].Float64()
					if x != 0 || rng.IntN(3) == 0 {
						lpStaticMust(t, parent.AddNewElement(rp[i], ci[j], x))
					}
				}
			}
			block, err := NewFloatSquareDenseSubmatrix(parent, start)
			lpStaticMust(t, err)
			s := &Settings{DropTolerance: 1e-14}
			for i := start; i < n; i++ {
				best := i
				for j := i + 1; j < n; j++ {
					if new(big.Rat).Abs(a[i][j]).Cmp(new(big.Rat).Abs(a[i][best])) > 0 {
						best = j
					}
				}
				pivot, err := block.FindPivotColumn(i)
				lpStaticMust(t, err)
				if pivot != int32(best) {
					t.Fatalf("pivot %d want %d", pivot, best)
				}
				if best != i {
					swaps++
					lpStaticMust(t, parent.SwapColumns(i, best))
					lpStaticMust(t, block.SwapColumns(i, best))
					for k := range a {
						a[k][i], a[k][best] = a[k][best], a[k][i]
					}
				} else {
					lpStaticMust(t, block.SwapColumns(i, i))
				}
				// Independent elementary row operations track E and E*A*P.
				p := new(big.Rat).Set(a[i][i])
				for j := uint32(0); j < n; j++ {
					a[i][j].Quo(a[i][j], p)
					e[i][j].Quo(e[i][j], p)
				}
				for row := i + 1; row < n; row++ {
					mult := new(big.Rat).Set(a[row][i])
					for j := uint32(0); j < n; j++ {
						a[row][j].Sub(a[row][j], new(big.Rat).Mul(mult, a[i][j]))
						e[row][j].Sub(e[row][j], new(big.Rat).Mul(mult, e[i][j]))
					}
				}
				lpStaticMust(t, block.Pivot(i, s))
			}
			lpStaticMust(t, block.UpdateParent(s))
			ok, err := block.IsLMatrix()
			lpStaticMust(t, err)
			if !ok || !block.IsDense() {
				t.Fatal("dense L tail")
			}
			for i := uint32(0); i < n; i++ {
				for j := uint32(0); j < n; j++ {
					x, err := parent.Get(i, j)
					lpStaticMust(t, err)
					lpFloatBlockNear(t, x, a[i][j])
				}
				for k, v := range parent.Rows[i] {
					other := parent.Columns[v.Index].Values[v.Other]
					if other.Index != i || other.Other != uint32(k) || other.Value != v.Value {
						t.Fatal("parent reciprocal link")
					}
				}
			}
			for i := start; i < n; i++ {
				for j := start; j < n; j++ {
					cell, err := block.cell(i, j)
					lpStaticMust(t, err)
					lpFloatBlockNear(t, *cell, e[i][j])
				}
			}
			// Conjugate both axes of the independent tail matrix.
			qv := make([]uint32, n)
			for i, j := range rng.Perm(int(n)) {
				qv[i] = uint32(j)
			}
			q, err := PermutationFromValues(n, qv)
			lpStaticMust(t, err)
			lpStaticMust(t, block.Conjugate(q))
			conjugated := lpExactIdentity(int(n))
			for i := range conjugated {
				for j := range conjugated[i] {
					conjugated[i][j].Set(e[qv[i]][qv[j]])
				}
			}
			b := make([]*big.Rat, n)
			for i := range b {
				b[i] = big.NewRat(int64(rng.IntN(15)-7), 4)
			}
			for _, right := range []bool{false, true} {
				want := lpExactMatrixVector(conjugated, b, right)
				for _, indexed := range []bool{false, true} {
					v := NewIndexedFloatVector(n)
					for _, i := range rng.Perm(int(n)) {
						v.Data[i], _ = b[i].Float64()
						if v.Data[i] != 0 {
							v.Index = append(v.Index, uint32(i))
						}
					}
					if right {
						if indexed {
							err = block.ApplyRightIndexed(v)
						} else {
							err = block.ApplyRight(v.Data)
						}
					} else {
						if indexed {
							err = block.ApplyLeftToT(v, s)
						} else {
							err = block.ApplyLeft(v.Data)
						}
					}
					lpStaticMust(t, err)
					for i := range want {
						lpFloatBlockNear(t, v.Data[i], want[i])
					}
					if indexed {
						seen := map[uint32]bool{}
						for _, i := range v.Index {
							if seen[i] || v.Data[i] == 0 {
								t.Fatal("sparse membership", v)
							}
							seen[i] = true
						}
						for i, x := range v.Data {
							if (x != 0) != seen[uint32(i)] {
								t.Fatal("missing index", v)
							}
						}
					}
				}
			}
		})
	}
	if swaps == 0 {
		t.Fatal("no dense column swaps exercised")
	}
}

func TestLPFloatDenseBlockLiteralStorageAndApplications(t *testing.T) {
	parent := newFloatSparseMatrix(3)
	lpStaticMust(t, parent.AddNewElement(1, 1, 2))
	lpStaticMust(t, parent.AddNewElement(1, 1, 3))
	lpStaticMust(t, parent.AddNewElement(2, 2, 4))
	m, err := NewFloatSquareDenseSubmatrix(parent, 1)
	lpStaticMust(t, err)
	if !reflect.DeepEqual(m.Data, []float64{3, 0, 0, 4}) {
		t.Fatal("last duplicate wins", m.Data)
	}
	// Missing coefficients are zero-initialized. Strict tie keeps first logical column.
	m.Data = []float64{2, -2, 0, 0}
	j, err := m.FindPivotColumn(1)
	lpStaticMust(t, err)
	if j != 1 {
		t.Fatal("strict max tie", j)
	}
	j, err = m.FindPivotColumn(2)
	lpStaticMust(t, err)
	if j != -1 {
		t.Fatal("zero row pivot", j)
	}
	m.Data = []float64{math.NaN(), 1, 0, 1}
	j, err = m.FindPivotColumn(1)
	lpStaticMust(t, err)
	if j != 2 {
		t.Fatal("NaN max comparison", j)
	}
	m.Data = []float64{1, 0, 0, 1}
	left, right := []float64{2, 3, 4, 9}, []float64{2, 3, 4, 9}
	lpStaticMust(t, m.ApplyLeft(left))
	lpStaticMust(t, m.ApplyRight(right))
	if left[3] != 9 || right[3] != 0 {
		t.Fatal("dense trailing-slot distinction", left, right)
	}
	// Dense IEEE products include stored zero times infinity. Sparse routes
	// use only Index and do not read hidden data as input.
	w := &IndexedFloatVector{Data: []float64{7, 3, math.Inf(1)}, Index: []uint32{1}}
	lpStaticMust(t, m.ApplyLeftToT(w, &Settings{DropTolerance: 1e-14}))
	if !reflect.DeepEqual(w.Data, []float64{0, 3, 0}) || !reflect.DeepEqual(w.Index, []uint32{1}) {
		t.Fatal("hidden sparse input", w)
	}
	w = &IndexedFloatVector{Data: []float64{7, 3, 4}, Index: []uint32{1}}
	lpStaticMust(t, m.ApplyRightIndexed(w))
	if !reflect.DeepEqual(w.Data, []float64{7, 3, 4}) {
		t.Fatal("assignment preserves unindexed destination slots", w)
	}
	// Right work cleanup is fixed1e-14; left uses settings, even when zero.
	for _, right := range []bool{false, true} {
		w = &IndexedFloatVector{Data: []float64{0, 1e-15, 0}, Index: []uint32{1}}
		if right {
			err = m.ApplyRightIndexed(w)
		} else {
			err = m.ApplyLeftToT(w, &Settings{})
		}
		lpStaticMust(t, err)
		if right {
			if len(w.Index) != 0 || w.Data[1] != 0 {
				t.Fatal("fixed right cleanup", w)
			}
		} else if !reflect.DeepEqual(w.Index, []uint32{0, 1, 2}) || w.Data[1] != 1e-15 {
			t.Fatal("zero settings tolerance", w)
		}
	}
	// Once the parent grows, right applications preserve the new identity area
	// through permutation fallback outside the old permutation extents.
	lpStaticMust(t, parent.Resize(5))
	w = &IndexedFloatVector{Data: []float64{2, 3, 4, 5, 6}, Index: []uint32{4, 3, 2, 1, 0}}
	lpStaticMust(t, m.ApplyRightIndexed(w))
	if !reflect.DeepEqual(w.Data, []float64{2, 3, 4, 5, 6}) {
		t.Fatal("parent growth identity area", w)
	}
	data := []float64{2, 3, 4, 5, 6}
	lpStaticMust(t, m.ApplyRight(data))
	if !reflect.DeepEqual(data, []float64{2, 3, 4, 5, 6}) {
		t.Fatal("dense growth identity area", data)
	}
}

func TestLPFloatDenseBlockDuplicateSparseInput(t *testing.T) {
	m := lpFloatBlockFixture(t, 1)
	m.Data = []float64{1, 0, 0, 1}
	for _, right := range []bool{false, true} {
		v := &IndexedFloatVector{Data: []float64{0, 3, 0}, Index: []uint32{1, 1}}
		var err error
		if right {
			err = m.ApplyRightIndexed(v)
		} else {
			err = m.ApplyLeftToT(v, &Settings{DropTolerance: 1e-14})
		}
		lpStaticMust(t, err)
		if v.Data[1] != 6 || !reflect.DeepEqual(v.Index, []uint32{1}) {
			t.Fatal("duplicate contributions", right, v)
		}
	}
	v := &IndexedFloatVector{Data: []float64{3, 0, 0}, Index: []uint32{0, 0}}
	err := m.ApplyRightIndexed(v)
	if sourceDebugTests {
		if err != ErrAssertion || !reflect.DeepEqual(v.Index, []uint32{0, 0}) || m.WorkVector.Data[0] != 3 {
			t.Fatal("duplicate outside set assertion", err, v, m.WorkVector)
		}
	} else if err != nil || v.Data[0] != 3 || !reflect.DeepEqual(v.Index, []uint32{0, 0}) {
		t.Fatal("duplicate outside assignment", err, v)
	}
}

func TestLPFloatDenseBlockPublicationErrors(t *testing.T) {
	parent := newFloatSparseMatrix(2)
	for i, v := range []float64{2, 4, 6, 8} {
		lpStaticMust(t, parent.AddNewElement(uint32(i/2), uint32(i%2), v))
	}
	m, err := NewFloatSquareDenseSubmatrix(parent, 0)
	lpStaticMust(t, err)
	m.Data = m.Data[:3]
	if err := m.Pivot(0, &Settings{DropTolerance: 1e-14}); err != ErrBounds || !reflect.DeepEqual(m.Data, []float64{.5, 2, -3}) {
		t.Fatal("pivot partial L publication", err, m.Data)
	}
	m.Data = []float64{2, 4, 6, 8}
	w := &IndexedFloatVector{Data: []float64{3}, Index: []uint32{0}}
	if err := m.ApplyRightIndexed(w); err != ErrBounds || w.Data[0] != 3 || m.WorkVector.Data[0] != 6 {
		t.Fatal("right caller/work prefix", err, w, m.WorkVector)
	}
	w = &IndexedFloatVector{Data: []float64{3}, Index: []uint32{0}}
	if err := m.ApplyLeftToT(w, &Settings{DropTolerance: 1e-14}); err != ErrBounds || w.Data[0] != 6 || !reflect.DeepEqual(w.Index, []uint32{0, 1}) {
		t.Fatal("left publication prefix", err, w)
	}
	for _, apply := range []func([]float64) error{m.ApplyLeft, m.ApplyRight} {
		data := []float64{3}
		if err := apply(data); err != ErrBounds || data[0] != 3 {
			t.Fatal("dense product failure", err, data)
		}
	}
	// Row assignment precedes reading the reciprocal column during update.
	parent.Columns[0].Values = nil
	if err := m.UpdateExistingParentRow(0, &Settings{DropTolerance: 1e-14}); err != ErrBounds || parent.Rows[0][0].Value != 1 {
		t.Fatal("parent row publication", err, parent.Rows[0])
	}
}
