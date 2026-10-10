package lp

import (
	"fmt"
	"math/big"
	"reflect"
	"testing"
)

func TestLPExactDenseBlockApplicationErrorPrefixes(t *testing.T) {
	for _, op := range []string{"left", "right", "left-indexed", "right-indexed", "pair"} {
		for _, kind := range []string{"input", "data", "row-map", "column-map"} {
			t.Run(fmt.Sprint(op, "/", kind), func(t *testing.T) {
				h := newLPPairHarness(t)
				m := lpExactDenseFixture(t, h, 0)
				pm := &PairSquareDenseSubmatrix{RationalSquareDenseSubmatrix: m}
				v := NewIndexedRationalVector(h.m, 2)
				v.Index = []uint32{1}
				h.must(h.m.SetInt64(&v.Data[1], 2))
				pw := NewIndexedPairVector(h.m, 2)
				switch kind {
				case "input":
					v.Data = nil
					pw.Data = nil
				case "data":
					m.Data = nil
				case "row-map":
					m.RowPermutation.values = nil
					m.RowPermutation.reverse[0], m.RowPermutation.reverse[1] = 4, 4
				case "column-map":
					m.ColumnPermutation.values = nil
					m.ColumnPermutation.reverse[0], m.ColumnPermutation.reverse[1] = 4, 4
				}
				var err error
				switch op {
				case "left":
					err = m.ApplyLeft(v.Data)
				case "right":
					err = m.ApplyRight(v.Data)
				case "left-indexed":
					err = m.ApplyLeftToT(v, nil)
				case "right-indexed":
					err = m.ApplyRightIndexed(v)
				case "pair":
					err = pm.ApplyLeft(pw.Data)
				}
				if err != ErrBounds {
					t.Fatal("application bounds", err)
				}
				h.must(pm.Close())
				h.must(v.Close())
				h.must(pw.Close())
				h.must(m.Parent.Close())
				lpExactVectorNamedCensus(t, h, nil)
			})
		}
	}
	for _, right := range []bool{false, true} {
		t.Run(fmt.Sprint("short-output/", right), func(t *testing.T) {
			h := newLPPairHarness(t)
			parent := lpNewExactSparse(t, h, 3)
			m, err := NewRationalSquareDenseSubmatrix(parent, 1)
			h.must(err)
			for i, x := range []int64{2, 3, 5, 7} {
				h.must(h.m.SetInt64(&m.Data[i], x))
			}
			v := NewIndexedRationalVector(h.m, 2)
			v.Index = []uint32{1}
			h.must(h.m.SetInt64(&v.Data[1], 2))
			if right {
				err = m.ApplyRightIndexed(v)
			} else {
				err = m.ApplyLeftToT(v, nil)
			}
			if err != ErrBounds {
				t.Fatal(err)
			}
			if right {
				if !reflect.DeepEqual(v.Index, []uint32{1}) {
					t.Fatal("right caller changed before Assign")
				}
				lpSparseRationalCheck(t, h, &v.Data[1], big.NewRat(2, 1))
			} else {
				if !reflect.DeepEqual(v.Index, []uint32{1, 2}) {
					t.Fatal("index before missing destination", v.Index)
				}
				lpSparseRationalCheck(t, h, &v.Data[1], big.NewRat(4, 1))
			}
			h.must(m.Close())
			h.must(v.Close())
			h.must(parent.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
	for _, kind := range []string{"row", "column"} {
		t.Run("conjugate/"+kind, func(t *testing.T) {
			h := newLPPairHarness(t)
			m := lpExactDenseFixture(t, h, 0)
			q := NewPermutation(2)
			if kind == "row" {
				q.values[1] = 2
			} else {
				h.must(q.TransposeLeft(0, 1))
				m.ColumnPermutation = NewPermutation(3)
			}
			want := ErrBounds
			if kind == "column" && sourceDebugTests {
				want = ErrAssertion
			}
			if err := m.Conjugate(q); err != want {
				t.Fatal(err, want)
			}
			if kind == "column" && !reflect.DeepEqual(m.RowPermutation.Values(), q.Values()) {
				t.Fatal("row permutation publication prefix")
			}
			h.must(m.Close())
			h.must(m.Parent.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
	// Manager mismatch is rejected before either owner acquires a workspace.
	h := newLPPairHarness(t)
	other := newLPPairHarness(t)
	m := lpExactDenseFixture(t, h, 0)
	v := NewIndexedRationalVector(other.m, 2)
	if err := m.ApplyLeftToT(v, nil); err != ErrExactVectorOwner {
		t.Fatal(err)
	}
	if err := m.ApplyRightIndexed(v); err != ErrExactVectorOwner {
		t.Fatal(err)
	}
	if m.work != nil {
		t.Fatal("foreign input acquired work")
	}
	h.must(m.Close())
	h.must(m.Parent.Close())
	other.must(v.Close())
	// A stopped parent returns no pair constructor owner.
	parent := lpNewExactSparse(t, h, 1)
	p := h.pair("1", "0")
	if err := parent.AddNewElement(2, 0, &p.X); err != ErrBounds {
		t.Fatal(err)
	}
	if x, err := NewPairSquareDenseSubmatrix(parent, 0); x != nil || err != ErrExactVectorPending {
		t.Fatal(x, err)
	}
	h.must(parent.Close())
	lpExactVectorNamedCensus(t, h, nil)
}
