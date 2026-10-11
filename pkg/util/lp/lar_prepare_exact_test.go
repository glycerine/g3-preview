package lp

import (
	"fmt"
	"math/big"
	"reflect"
	"testing"
)

func TestLARPrepareExactVectorKindsAndStrategies(t *testing.T) {
	for _, strategy := range []SimplexStrategy{StrategyLU, StrategyUndecided, StrategyTableauRows, StrategyTableauCosts} {
		for _, breakpoints := range []bool{false, true} {
			t.Run(fmt.Sprintf("%d/%t", strategy, breakpoints), func(t *testing.T) {
				h := newLPPairHarness(t)
				s := lpNewRemovalModel(t, h, "2", "3", false)
				c := s.Core
				r := c.RPrimal
				s.Settings.SimplexStrategy = strategy
				s.Settings.UseBreakpointsInFeasibilitySearch = breakpoints
				v := h.pair("11/7", "1/13")
				h.must(r.CopyXB.resizeData(5))
				h.must(r.CopyXB.Data[0].Copy(h.m, v))
				h.must(r.Y.resizeData(5))
				h.must(h.m.Copy(&r.Y.Data[0], &v.X))
				h.must(c.RightSides.Data[0].Copy(h.m, v))
				h.must(h.m.Copy(&c.Costs.Data[0], &v.X))
				h.must(h.m.Copy(&r.D.Data[0], &v.Y))
				for _, vec := range []*IndexedRationalVector{r.Ed, r.PivotRow, r.PivotRowB1, r.W} {
					h.must(vec.Resize(5))
					h.must(vec.SetValue(&v.X, 0))
				}
				r.RowsNZ = []uint32{73, 99}
				r.ColumnsNZ = []uint32{0, 91, 0, 99}
				h.must(r.BreakpointQueue.Resize(5))
				h.must(c.PrefixR())
				h.check(&r.CopyXB.Data[0], big.NewRat(11, 7), big.NewRat(1, 13))
				lpSparseRationalCheck(t, h, &r.Y.Data[0], big.NewRat(11, 7))
				h.check(&c.RightSides.Data[0], big.NewRat(11, 7), big.NewRat(1, 13))
				if len(c.RightSides.Data) != 1 {
					t.Fatal("RHS must follow rows")
				}
				lpSparseRationalCheck(t, h, &c.Costs.Data[0], big.NewRat(11, 7))
				lpSparseRationalCheck(t, h, &r.D.Data[0], big.NewRat(1, 13))
				tableau := strategy == StrategyTableauRows || strategy == StrategyTableauCosts
				if !tableau {
					if r.CopyXB.DataSize() != 3 || r.Y.DataSize() != 1 {
						t.Fatal("ordinary scratch extents")
					}
					for _, vec := range []*IndexedRationalVector{r.Ed, r.PivotRow, r.PivotRowB1, r.W} {
						if len(vec.Index) != 0 {
							t.Fatal("indexed scratch not cleared")
						}
						lpSparseRationalCheck(t, h, &vec.Data[0], big.NewRat(0, 1))
					}
					if !reflect.DeepEqual(r.RowsNZ, []uint32{73}) || !reflect.DeepEqual(r.ColumnsNZ, []uint32{1, 91, 1}) {
						t.Fatal("nonzero estimates must survive", r.RowsNZ, r.ColumnsNZ)
					}
				} else {
					if r.CopyXB.DataSize() != 5 || len(r.Ed.Index) != 1 || !reflect.DeepEqual(r.RowsNZ, []uint32{73, 99}) {
						t.Fatal("tableau skipped LU scratch")
					}
				}
				wantQueue := 5
				if strategy != StrategyTableauRows && breakpoints {
					wantQueue = 3
				}
				if len(r.BreakpointQueue.inverse) != wantQueue || r.UsingInfeasCosts != (strategy != StrategyTableauRows) {
					t.Fatal("cost strategy preparation")
				}
				h.must(c.PrefixR())
				h.must(s.Close())
				lpExactVectorNamedCensus(t, h, nil)
			})
		}
	}
}

func TestLARPrepareExactNZIndependentSparseCounts(t *testing.T) {
	h := newLPPairHarness(t)
	for world := 0; world < 100; world++ {
		s := lpNewRemovalModel(t, h, fmt.Sprint(world%3), fmt.Sprint(world%5), true)
		r := s.Core.RPrimal
		r.ColumnsNZ = []uint32{41}
		r.RowsNZ = []uint32{0, 97, 999}
		h.must(r.InitColumnRowNonZeroes())
		want := []uint32{41, 0, 1, 1}
		if world%5 != 0 {
			want[1] = 2
		}
		row := uint32(1)
		if world%3 != 0 {
			row++
		}
		if world%5 != 0 {
			row++
		}
		if !reflect.DeepEqual(r.ColumnsNZ, want) || !reflect.DeepEqual(r.RowsNZ, []uint32{row, 97}) {
			t.Fatal(world, r.ColumnsNZ, r.RowsNZ, want, row)
		}
		h.must(s.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLARPrepareExactFailurePrefixes(t *testing.T) {
	for _, phase := range []string{"ed", "pivot", "inverse", "w"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			s := lpPreparedLARPivot(t, h, "2", "3", "5")
			c := s.Core
			r := c.RPrimal
			bad := r.Ed
			switch phase {
			case "pivot":
				bad = r.PivotRow
			case "inverse":
				bad = r.PivotRowB1
			case "w":
				bad = r.W
			}
			bad.Index = []uint32{99}
			if err := c.PrefixR(); err != ErrBounds {
				t.Fatal(err)
			}
			if r.CopyXB.DataSize() != 3 {
				t.Fatal("lost successful resize prefix")
			}
			if c.PrefixR() != ErrExactVectorPending {
				t.Fatal("lost stopped core")
			}
			// Restore malformed external index metadata so actual owned values can close.
			bad.Index = nil
			h.must(s.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLARPrepareExactActualPendingQueuePrefix(t *testing.T) {
	h := lpNewDirectSparseHarness(t)
	s := lpNewRemovalModel(t, h, "2", "3", false)
	c := s.Core
	r := c.RPrimal
	s.Settings.SimplexStrategy = StrategyTableauCosts
	s.Settings.UseBreakpointsInFeasibilitySearch = true
	h.must(r.BreakpointQueue.Resize(3))
	value := h.pair("123456789012345678901234567890123456789012345678901234567890123456789/7", "1/13")
	h.must(r.BreakpointQueue.Enqueue(0, value))
	other := h.pair("1/17", "1/19")
	h.budget.FailAfter(0)
	err := r.BreakpointQueue.Enqueue(1, other)
	h.budget.FailAfter(-1)
	if !lpExactPriorityPhysicalRefusal(err) {
		t.Fatal("expected actual priority copy refusal", err)
	}
	if err = c.PrefixR(); err != ErrExactQueuePending {
		t.Fatal(err)
	}
	if c.RightSides.DataSize() != 1 || r.UsingInfeasCosts || c.RPrimal.calls.pending() {
		t.Fatal("RHS prefix or child ownership")
	}
	if c.PrefixR() != ErrExactVectorPending {
		t.Fatal("lost stopped core")
	}
	h.must(s.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLARPrepareExactStoppedPrimal(t *testing.T) {
	h := newLPPairHarness(t)
	s := lpNewRemovalModel(t, h, "2", "3", false)
	if s.Core.RPrimal.UpdateColumnInInfSet(99) != ErrBounds {
		t.Fatal("missing source bounds guard")
	}
	if s.Core.RPrimal.InitColumnRowNonZeroes() != ErrExactVectorPending || s.Core.PrefixR() != ErrExactVectorPending {
		t.Fatal("lost stopped child")
	}
	h.must(s.Close())
	lpExactVectorNamedCensus(t, h, nil)
}
