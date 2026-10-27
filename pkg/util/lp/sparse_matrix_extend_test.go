package lp

import (
	"fmt"
	"math/big"
	"math/rand/v2"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestLPSparseExtensionIndependentBlockOracle(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	rng := rand.New(rand.NewPCG(53791, 17117))
	for world := 0; world < 120; world++ {
		t.Run(fmt.Sprint(world), func(t *testing.T) {
			h := newLPPairHarness(t)
			value := h.pair("0", "0")
			n := uint32(world % 6)
			m, f := lpNewExactSparse(t, h, n), newFloatSparseMatrix(n)
			want := make([][]int64, n)
			stored := uint32(0)
			for i := range want {
				want[i] = make([]int64, n)
			}
			for i := uint32(0); i < n; i++ {
				for j := uint32(0); j < n; j++ {
					if rng.IntN(3) == 0 {
						continue
					}
					v := int64(rng.IntN(15) - 7)
					h.must(h.m.SetInt64(&value.X, v))
					h.must(m.AddNewElement(i, j, &value.X))
					lpStaticMust(t, f.AddNewElement(i, j, float64(v)))
					want[i][j] = v
					stored++
				}
			}
			for round := 0; round < 8; round++ {
				// Independently move logical rows and columns by a full permutation.
				order := rng.Perm(int(n))
				v, inv := make([]uint32, n), make([]uint32, n)
				for i, j := range order {
					v[i], inv[j] = uint32(j), uint32(i)
				}
				p, err := PermutationFromValues(n, v)
				h.must(err)
				h.must(m.MultiplyRight(p))
				lpStaticMust(t, f.MultiplyRight(p))
				a := make([][]int64, n)
				for i := range a {
					a[i] = make([]int64, n)
					for j := range a[i] {
						a[i][j] = want[i][inv[j]]
					}
				}
				h.must(m.MultiplyReverseLeft(p))
				lpStaticMust(t, f.MultiplyReverseLeft(p))
				for i := range want {
					want[i] = append([]int64(nil), a[inv[i]]...)
				}
				// Exercise no-op and multi-row growth; old workspace state survives.
				for i := uint32(0); i < n; i++ {
					m.WorkPivot[i], f.WorkPivot[i] = int32(i+7), int32(i+7)
					m.Processed[i], f.Processed[i] = i%2 == 0, i%2 == 0
				}
				if len(m.pivotQueue.pairs) > 0 {
					h.must(m.pivotQueue.Enqueue(0, 0, 17))
					lpStaticMust(t, f.pivotQueue.Enqueue(0, 0, 17))
				}
				if n > 0 {
					m.Columns[0].ShortenedMarkovitz = 3
					f.Columns[0].ShortenedMarkovitz = 3
				}
				next := n + uint32(rng.IntN(3))
				pairExtent := len(m.pivotQueue.pairs)
				h.must(m.Resize(next))
				lpStaticMust(t, f.Resize(next))
				if len(m.pivotQueue.pairs) != pairExtent || len(f.pivotQueue.pairs) != pairExtent || len(m.pivotQueue.queue.priorities) != int(next) || len(f.pivotQueue.queue.priorities) != int(next) {
					t.Fatal("queue resize changed the pair directory or missed its priority extent")
				}
				for i := uint32(0); i < n; i++ {
					want[i] = append(want[i], make([]int64, next-n)...)
				}
				for j := n; j < next; j++ {
					row := make([]int64, next)
					row[j] = 1
					want = append(want, row)
					if m.WorkPivot[j] != 0 || f.WorkPivot[j] != 0 || m.Processed[j] || f.Processed[j] {
						t.Fatal("new workspace entries must be zero, unlike constructor -1")
					}
				}
				for i := uint32(0); i < n; i++ {
					if m.WorkPivot[i] != int32(i+7) || f.WorkPivot[i] != int32(i+7) || m.Processed[i] != (i%2 == 0) || f.Processed[i] != (i%2 == 0) {
						t.Fatal("old workspace changed")
					}
				}
				if len(m.pivotQueue.pairs) > 0 {
					for _, q := range []*UintPairPriorityQueue{m.pivotQueue, f.pivotQueue} {
						priority, err := q.Priority(0, 0)
						h.must(err)
						if priority != 17 {
							t.Fatal("queue state changed")
						}
					}
				}
				if n > 0 {
					if m.Columns[0].ShortenedMarkovitz != 3 || f.Columns[0].ShortenedMarkovitz != 3 {
						t.Fatal("shortening counter changed")
					}
				}
				stored += next - n
				n = next
				mc, err := m.NonzeroCount()
				h.must(err)
				fc, err := f.NonzeroCount()
				h.must(err)
				if mc != stored || fc != stored || m.active != stored || f.active != stored {
					t.Fatalf("counts %d %d want %d", mc, fc, stored)
				}
				for i := uint32(0); i < n; i++ {
					for k, v := range m.Rows[i] {
						other := m.Columns[v.Index].Values[v.Other]
						if other.Index != i || other.Other != uint32(k) {
							t.Fatal("exact reciprocal link broken by extension")
						}
					}
					for k, v := range f.Rows[i] {
						other := f.Columns[v.Index].Values[v.Other]
						if other.Index != i || other.Other != uint32(k) || other.Value != v.Value {
							t.Fatal("floating reciprocal link broken by extension")
						}
					}
					for j := uint32(0); j < n; j++ {
						v, err := m.Get(i, j)
						h.must(err)
						lpSparseRationalCheck(t, h, v, big.NewRat(want[i][j], 1))
						x, err := f.Get(i, j)
						h.must(err)
						if x != float64(want[i][j]) {
							t.Fatalf("cell %d,%d got %g want %d", i, j, x, want[i][j])
						}
					}
				}
				lpExactVectorNamedCensus(t, h, lpSparseEtaOwningCells(m, nil))
			}
		})
	}
}

func TestLPSparseExtensionNativeInlineOnes(t *testing.T) {
	for _, direct := range []bool{false, true} {
		t.Run(fmt.Sprint(direct), func(t *testing.T) {
			var h *lpPairHarness
			if direct {
				h = lpNewDirectSparseHarness(t)
			} else {
				h = newLPPairHarness(t)
			}
			m := lpNewExactSparse(t, h, 1)
			p := h.pair("123456789123456789123456789123456789", "0")
			h.must(m.AddNewElement(0, 0, &p.X))
			h.budget.FailAfter(0)
			// Inline one and native metadata need no source numeral reservation.
			err := m.Resize(12)
			h.budget.FailAfter(-1)
			h.must(err)
			for j := uint32(1); j < 12; j++ {
				v, err := m.Get(j, j)
				h.must(err)
				lpSparseRationalCheck(t, h, v, big.NewRat(1, 1))
			}
			lpExactVectorNamedCensus(t, h, lpSparseEtaOwningCells(m, nil))
			h.must(m.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPSparseExtensionBoundsShrinkAndStoppedOwner(t *testing.T) {
	h := newLPPairHarness(t)
	m := lpNewExactSparse(t, h, 3)
	f := newFloatSparseMatrix(3)
	// Counting visits only dimension rows in descending order, independent of
	// the active count and any extra headers. Duplicates and zeros all count.
	m.Rows[2] = make([]RationalIndexedValue, 2)
	f.Rows[2] = make([]FloatIndexedValue, 2)
	for i := range m.Rows[2] {
		// Zero-valued numeral cells need the explicit numeral empty constructor.
		m.Rows[2][i].Value = numeral.NewRational()
	}
	m.Rows = append(m.Rows, nil)
	f.Rows = append(f.Rows, nil)
	for _, count := range []func() (uint32, error){m.NonzeroCount, f.NonzeroCount} {
		n, err := count()
		h.must(err)
		if n != 2 {
			t.Fatalf("stored count %d", n)
		}
	}
	// Remove fixture headers before the remaining shape checks.
	m.Rows[2] = nil
	f.Rows[2] = nil
	m.Rows = m.Rows[:1]
	f.Rows = f.Rows[:1]
	for _, count := range []func() (uint32, error){m.NonzeroCount, f.NonzeroCount} {
		if n, err := count(); err != ErrBounds || n != 0 {
			t.Fatalf("count bounds %d %v", n, err)
		}
	}
	m.Rows = append(m.Rows, nil, nil)
	f.Rows = append(f.Rows, nil, nil)
	for _, resize := range []func(uint32) error{m.Resize, f.Resize} {
		err := resize(2)
		if err != sourceAssertionFailure() {
			t.Fatalf("shrink %v", err)
		}
	}
	if sourceDebugTests {
		if m.Dimension() != 3 || f.Dimension() != 3 {
			t.Fatal("debug shrink mutated dimension")
		}
	} else if m.Dimension() != 2 || f.Dimension() != 2 || len(m.Rows) != 3 || len(f.Rows) != 3 {
		t.Fatal("release shrink repaired source retained headers")
	}
	// A real stopped matrix workspace rejects every new operation before any
	// permutation/header mutation, and cleanup remains with that matrix.
	one := h.pair("1", "0")
	if err := m.AddNewElement(9, 0, &one.X); err != ErrBounds {
		t.Fatalf("expected real stopped add: %v", err)
	}
	before := append([]uint32(nil), m.RowPermutation.Values()...)
	p := NewPermutation(m.Dimension())
	for _, operation := range []func() error{
		func() error { _, err := m.NonzeroCount(); return err },
		func() error { return m.MultiplyRight(p) },
		func() error { return m.MultiplyReverseLeft(p) },
		func() error { return m.Resize(8) },
	} {
		if err := operation(); err != ErrExactVectorPending || !reflect.DeepEqual(before, m.RowPermutation.Values()) {
			t.Fatalf("pending guard %v", err)
		}
	}
}

func TestLPSparseExtensionPublicationErrors(t *testing.T) {
	h := newLPPairHarness(t)
	m := lpNewExactSparse(t, h, 2)
	f := newFloatSparseMatrix(2)
	// Both composition wrappers preserve errors from the actual permutation.
	p := NewPermutation(2)
	p.values[1] = 2
	for _, op := range []func(*Permutation) error{m.MultiplyRight, f.MultiplyRight} {
		if err := op(p); err != ErrBounds {
			t.Fatalf("right permutation bounds: %v", err)
		}
	}
	p = NewPermutation(2)
	p.reverse[1] = 2
	for _, op := range []func(*Permutation) error{m.MultiplyReverseLeft, f.MultiplyReverseLeft} {
		if err := op(p); err != ErrBounds {
			t.Fatalf("reverse-left permutation bounds: %v", err)
		}
	}
	// Missing old row headers are not repaired by growth. All extents are
	// published before adding the first identity entry reaches that bad row.
	m.Rows, f.Rows = nil, nil
	for _, op := range []func(uint32) error{m.Resize, f.Resize} {
		if err := op(3); err != ErrBounds {
			t.Fatalf("identity addition bounds: %v", err)
		}
	}
	if m.Dimension() != 3 || f.Dimension() != 3 || len(m.Rows) != 1 || len(f.Rows) != 1 || len(m.WorkPivot) != 3 || len(f.WorkPivot) != 3 {
		t.Fatal("growth publication prefix lost")
	}
	if err := m.Resize(4); err != ErrExactVectorPending || m.Dimension() != 3 {
		t.Fatalf("stopped resize %v", err)
	}
}
