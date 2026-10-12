package lp

import (
	"fmt"
	"math/big"
	"math/rand/v2"
	"reflect"
	"testing"
)

// Called after independently verified elimination and parent publication.
func lpExactDenseCheckApplications(t *testing.T, h *lpPairHarness, m *RationalSquareDenseSubmatrix, a [][]*big.Rat, rng *rand.Rand) {
	t.Helper()
	n := len(a)
	ok, err := m.IsLMatrix()
	h.must(err)
	if !ok {
		t.Fatal("completed L")
	}
	qv := make([]uint32, n)
	for i, j := range rng.Perm(n) {
		qv[i] = uint32(j)
	}
	q, err := PermutationFromValues(uint32(n), qv)
	h.must(err)
	h.must(m.Conjugate(q))
	expected := lpExactIdentity(n)
	for i := range expected {
		for j := range expected[i] {
			expected[i][j].Set(a[qv[i]][qv[j]])
		}
	}
	x, y := make([]*big.Rat, n), make([]*big.Rat, n)
	for i := range x {
		x[i] = big.NewRat(int64(rng.IntN(9)-4), 7)
		y[i] = big.NewRat(int64(rng.IntN(11)-5), 13)
	}
	for _, right := range []bool{false, true} {
		want := lpExactMatrixVector(expected, x, right)
		for _, indexed := range []bool{false, true} {
			v := NewIndexedRationalVector(h.m, uint32(n+2))
			for _, i := range rng.Perm(n) {
				h.must(h.m.SetString(&v.Data[i], x[i].RatString()))
				if x[i].Sign() != 0 {
					v.Index = append(v.Index, uint32(i))
				}
			}
			h.must(h.m.SetInt64(&v.Data[n], 19))
			h.must(h.m.SetInt64(&v.Data[n+1], 23))
			var tail RationalTailMatrix = m
			if right {
				if indexed {
					err = tail.ApplyRightIndexed(v)
				} else {
					err = tail.ApplyRight(v.Data)
				}
			} else {
				if indexed {
					err = tail.ApplyLeftToT(v, &Settings{DropTolerance: 1e100})
				} else {
					err = tail.ApplyLeft(v.Data)
				}
			}
			h.must(err)
			for i := range want {
				lpSparseRationalCheck(t, h, &v.Data[i], want[i])
			}
			trailing := int64(19)
			if right && !indexed {
				trailing = 0
			}
			lpSparseRationalCheck(t, h, &v.Data[n], big.NewRat(trailing, 1))
			trailing = 23
			if right && !indexed {
				trailing = 0
			}
			lpSparseRationalCheck(t, h, &v.Data[n+1], big.NewRat(trailing, 1))
			if indexed {
				seen := make(map[uint32]bool)
				for _, i := range v.Index {
					if seen[i] || int(i) >= n || want[i].Sign() == 0 {
						t.Fatal("index membership", v.Index)
					}
					seen[i] = true
				}
				for i, z := range want {
					if (z.Sign() != 0) != seen[uint32(i)] {
						t.Fatal("missing index", i)
					}
				}
				if !right {
					var ascending []uint32
					for i, z := range want {
						if z.Sign() != 0 {
							ascending = append(ascending, uint32(i))
						}
					}
					if !reflect.DeepEqual(v.Index, ascending) && !(len(v.Index) == 0 && len(ascending) == 0) {
						t.Fatal("left rebuild order", v.Index, ascending)
					}
				}
			}
			owned := lpExactDenseCells(m)
			for i := range v.Data {
				owned = append(owned, &v.Data[i])
			}
			lpExactVectorNamedCensus(t, h, owned)
			h.must(v.Close())
		}
	}
	pw := NewIndexedPairVector(h.m, uint32(n+1))
	for i := range x {
		h.must(h.m.SetString(&pw.Data[i].X, x[i].RatString()))
		h.must(h.m.SetString(&pw.Data[i].Y, y[i].RatString()))
	}
	h.must(h.m.SetInt64(&pw.Data[n].X, 29))
	h.must(h.m.SetInt64(&pw.Data[n].Y, 31))
	pair := &PairSquareDenseSubmatrix{RationalSquareDenseSubmatrix: m}
	var tail PairTailMatrix = pair
	h.must(tail.ApplyLeft(pw.Data))
	wantX, wantY := lpExactMatrixVector(expected, x, false), lpExactMatrixVector(expected, y, false)
	for i := range x {
		h.check(&pw.Data[i], wantX[i], wantY[i])
	}
	h.check(&pw.Data[n], big.NewRat(29, 1), big.NewRat(31, 1))
	owned := lpExactDenseCells(m)
	for i := range pw.Data {
		owned = append(owned, &pw.Data[i].X, &pw.Data[i].Y)
	}
	lpExactVectorNamedCensus(t, h, owned)
	h.must(pw.Close())
}

func TestLPExactDenseBlockIndexedLiteralInputsAndGrowth(t *testing.T) {
	for _, right := range []bool{false, true} {
		t.Run(fmt.Sprint(right), func(t *testing.T) {
			h := newLPPairHarness(t)
			parent := lpNewExactSparse(t, h, 3)
			m, err := NewRationalSquareDenseSubmatrix(parent, 1)
			h.must(err)
			for i, x := range []string{"2", "3", "5", "7"} {
				h.must(h.m.SetString(&m.Data[i], x))
			}
			h.must(parent.Resize(5))
			v := NewIndexedRationalVector(h.m, 6)
			for i, x := range []int64{11, 13, 17, 19, 23, 29} {
				h.must(h.m.SetInt64(&v.Data[i], x))
			}
			// Duplicate inside terms accumulate; outside terms overwrite. Hidden
			// coordinate2 is ignored as input; its destination behavior is literal.
			v.Index = []uint32{3, 1, 0, 1, 3, 4}
			if right {
				v.Index = []uint32{3, 1, 0, 1, 4}
			}
			if right {
				err = m.ApplyRightIndexed(v)
			} else {
				err = m.ApplyLeftToT(v, nil)
			}
			h.must(err)
			want := []int64{11, 52, 130, 19, 23, 29}
			if right {
				want = []int64{11, 52, 78, 19, 23, 29}
			}
			for i, x := range want {
				lpSparseRationalCheck(t, h, &v.Data[i], big.NewRat(x, 1))
			}
			if !reflect.DeepEqual(v.Index, []uint32{0, 1, 2, 3, 4}) && !right {
				t.Fatal("left index order", v.Index)
			}
			h.must(m.Close())
			h.must(v.Close())
			h.must(parent.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
	// Cancellation deletes a computed index; hidden slots survive right Assign.
	h := newLPPairHarness(t)
	parent := lpNewExactSparse(t, h, 2)
	m, err := NewRationalSquareDenseSubmatrix(parent, 0)
	h.must(err)
	for i, x := range []int64{1, 0, -1, 0} {
		h.must(h.m.SetInt64(&m.Data[i], x))
	}
	v := NewIndexedRationalVector(h.m, 2)
	h.must(h.m.SetInt64(&v.Data[0], 3))
	h.must(h.m.SetInt64(&v.Data[1], 3))
	v.Index = []uint32{0, 1}
	h.must(m.ApplyRightIndexed(v))
	if len(v.Index) != 0 {
		t.Fatal("cancelled index", v.Index)
	}
	for i := range v.Data {
		lpSparseRationalCheck(t, h, &v.Data[i], new(big.Rat))
	}
	h.must(m.Close())
	h.must(v.Close())
	h.must(parent.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

// The retained dense-right algorithm treats newly appended parent coordinates
// as identity rows and replaces trailing caller slots with initialized zeros.
func TestLPExactDenseBlockDenseRightParentGrowth(t *testing.T) {
	for _, direct := range []bool{false, true} {
		t.Run(fmt.Sprint(direct), func(t *testing.T) {
			var h *lpPairHarness
			if direct {
				h = lpNewDirectSparseHarness(t)
			} else {
				h = newLPPairHarness(t)
			}
			parent := lpNewExactSparse(t, h, 3)
			m, err := NewRationalSquareDenseSubmatrix(parent, 1)
			h.must(err)
			for i, x := range []int64{2, 3, 5, 7} {
				h.must(h.m.SetInt64(&m.Data[i], x))
			}
			h.must(parent.Resize(5))
			v := NewIndexedRationalVector(h.m, 6)
			for i, x := range []int64{11, 13, 17, 19, 23, 29} {
				h.must(h.m.SetInt64(&v.Data[i], x))
			}
			h.must(m.ApplyRight(v.Data))
			h.must(m.Close())
			for i, x := range []int64{11, 111, 158, 19, 23, 0} {
				lpSparseRationalCheck(t, h, &v.Data[i], big.NewRat(x, 1))
			}
			h.must(v.Close())
			h.must(parent.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}
