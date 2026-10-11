package lp

import (
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"testing"
)

func TestLARChangedBoundsIndependentTableauOracle(t *testing.T) {
	h := newLPPairHarness(t)
	for world := 0; world < 160; world++ {
		a := big.NewRat(int64(world%17-8), int64(world%7+1))
		b := big.NewRat(int64(world%13-6), int64(world%11+1))
		s := lpNewRemovalModel(t, h, a.RatString(), b.RatString(), true)
		c := s.Core
		costs := world%2 == 0
		if costs {
			s.Settings.SimplexStrategy = StrategyTableauCosts
			c.RPrimal.UsingInfeasCosts = true
		}
		h.must(s.BasicColumnsWithChangedCost.Resize(4))
		low := h.pair("3/7", "1/5")
		upper := h.pair("-2/11", "-1/7")
		h.must(s.Bounds.Low.Set(0, low))
		h.must(s.Bounds.Types.Set(0, ColumnLowBound))
		h.must(s.Bounds.Upper.Set(1, upper))
		h.must(s.Bounds.Types.Set(1, ColumnUpperBound))
		x := new(big.Rat).Add(new(big.Rat).Mul(a, big.NewRat(3, 7)), new(big.Rat).Mul(b, big.NewRat(-2, 11)))
		y := new(big.Rat).Add(new(big.Rat).Mul(a, big.NewRat(1, 5)), new(big.Rat).Mul(b, big.NewRat(-1, 7)))
		// Opposite sides of the final basic values independently force costs +1/-1.
		h.must(s.Bounds.Low.Set(2, h.pair(new(big.Rat).Add(x, big.NewRat(1, 13)).RatString(), y.RatString())))
		h.must(s.Bounds.Types.Set(2, ColumnLowBound))
		h.must(s.Bounds.Upper.Set(3, h.pair(new(big.Rat).Sub(x, big.NewRat(1, 17)).RatString(), y.RatString())))
		h.must(s.Bounds.Types.Set(3, ColumnUpperBound))
		h.must(s.Bounds.ChangedColumns.Clear())
		for _, j := range []uint32{1, 0, 2, 3} {
			h.must(s.Bounds.ChangedColumns.Insert(j))
		}
		h.must(s.RowsWithChangedBounds.Clear())
		h.must(s.DetectRowsWithChangedBounds())
		if !reflect.DeepEqual(s.RowsWithChangedBounds.Indices(), []uint32{0, 1}) {
			t.Fatal("row order", s.RowsWithChangedBounds.Indices())
		}
		if world%3 == 0 {
			h.must(s.UpdateXAndInfCostsForColumnsWithChangedBounds())
		} else {
			h.must(s.UpdateXAndInfCostsForColumnsWithChangedBoundsTableau())
		}
		h.check(&c.RX.Data[0], big.NewRat(3, 7), big.NewRat(1, 5))
		h.check(&c.RX.Data[1], big.NewRat(-2, 11), big.NewRat(-1, 7))
		for _, j := range []uint32{2, 3} {
			h.check(&c.RX.Data[j], x, y)
			if !c.RPrimal.InfSet.Contains(j) {
				t.Fatal("missing infeasible basic", world, j)
			}
		}
		if costs && world%3 != 0 {
			lpSparseRationalCheck(t, h, &c.Costs.Data[2], big.NewRat(1, 1))
			lpSparseRationalCheck(t, h, &c.Costs.Data[3], big.NewRat(-1, 1))
			// Equal rows, opposite costs cancel in both nonbasic reduced costs.
			for j := range c.RPrimal.D.Data {
				lpSparseRationalCheck(t, h, &c.RPrimal.D.Data[j], big.NewRat(0, 1))
			}
		}
		if !reflect.DeepEqual(s.Bounds.ChangedColumns.Indices(), []uint32{1, 0, 2, 3}) {
			t.Fatal("consumed changed columns")
		}
		h.must(s.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLARChangedBoundsBasicMembershipTransitions(t *testing.T) {
	for _, strategy := range []SimplexStrategy{StrategyTableauRows, StrategyTableauCosts, StrategyLU, StrategyUndecided} {
		for _, was := range []bool{false, true} {
			for _, now := range []bool{false, true} {
				h := newLPPairHarness(t)
				s := lpNewRemovalModel(t, h, "2", "3", false)
				s.Settings.SimplexStrategy = strategy
				h.must(s.BasicColumnsWithChangedCost.Resize(3))
				if was {
					h.must(s.Core.RPrimal.InfSet.Insert(2))
				}
				if now {
					h.must(s.Bounds.Low.Set(2, h.pair("0", "1/7")))
					h.must(s.Bounds.Types.Set(2, ColumnLowBound))
				}
				h.must(s.UpdateXAndInfCostsForColumnWithChangedBounds(2))
				if s.Core.RPrimal.InfSet.Contains(2) != now || s.BasicColumnsWithChangedCost.Contains(2) != ((was != now) && strategy != StrategyTableauRows) {
					t.Fatal(strategy, was, now)
				}
				h.must(s.Close())
				lpExactVectorNamedCensus(t, h, nil)
			}
		}
	}
}

func TestLARChangedBoundsActualLURowsAndValues(t *testing.T) {
	h := newLPPairHarness(t)
	for world := 0; world < 100; world++ {
		a := big.NewRat(int64(world%17-8), 7)
		basis := big.NewRat(int64(world%11+1), 13)
		s := lpPreparedLARPivot(t, h, a.RatString(), "3", basis.RatString())
		h.must(s.RowsWithChangedBounds.Resize(1))
		h.must(s.RowsWithChangedBounds.Clear())
		// Force a resize and then the equal-size clear branch on repeated calls.
		h.must(s.ColumnBuffer.Resize(3))
		h.must(s.DetectRowsOfColumnWithBoundChange(0))
		if s.RowsWithChangedBounds.Contains(0) != (a.Sign() != 0) {
			t.Fatal("LU sparse row", world)
		}
		h.must(s.DetectRowsWithChangedBoundsForColumn(0))
		h.must(s.DetectRowsOfColumnWithBoundChange(2))
		if !s.RowsWithChangedBounds.Contains(0) {
			t.Fatal("basic row")
		}
		h.must(s.Bounds.Low.Set(0, h.pair("2/7", "1/5")))
		h.must(s.Bounds.Types.Set(0, ColumnLowBound))
		h.must(s.UpdateXAndInfCostsForColumnWithChangedBounds(0))
		h.check(&s.Core.RX.Data[0], big.NewRat(2, 7), big.NewRat(1, 5))
		factor := new(big.Rat).Neg(new(big.Rat).Quo(a, basis))
		h.check(&s.Core.RX.Data[2], new(big.Rat).Mul(factor, big.NewRat(2, 7)), new(big.Rat).Mul(factor, big.NewRat(1, 5)))
		h.must(s.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLARChangedBoundsPhysicalOwnership(t *testing.T) {
	for _, direct := range []bool{false, true} {
		for _, op := range []string{"update", "tableau-costs", "LU-detect"} {
			failures := 0
			success := false
			for ordinal := int64(0); ordinal < 300; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("direct=%t/%s/%d", direct, op, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					huge := strings.Repeat("1234567890", 100) + "/7"
					var s *LARSolver
					if op != "LU-detect" {
						s = lpNewRemovalModel(t, h, huge, "3", false)
						h.must(s.Bounds.Low.Set(0, h.pair(huge, huge)))
						h.must(s.Bounds.Types.Set(0, ColumnLowBound))
						if op == "tableau-costs" {
							s.Settings.SimplexStrategy = StrategyTableauCosts
							s.Core.RPrimal.UsingInfeasCosts = true
							h.must(s.BasicColumnsWithChangedCost.Resize(3))
							h.must(s.Bounds.Upper.Set(2, h.pair("1", "0")))
							h.must(s.Bounds.Types.Set(2, ColumnUpperBound))
							h.must(s.Bounds.ChangedColumns.Clear())
							h.must(s.Bounds.ChangedColumns.Insert(0))
						}
					} else {
						s = lpPreparedLARPivot(t, h, huge, "3", "1"+strings.Repeat("0", 999)+"3")
					}
					h.must(s.RowsWithChangedBounds.Resize(s.Core.RA.RowCount()))
					h.must(s.RowsWithChangedBounds.Clear())
					h.budget.FailAfter(ordinal)
					var err error
					if op == "update" {
						err = s.UpdateXAndInfCostsForColumnWithChangedBounds(0)
					} else if op == "tableau-costs" {
						err = s.UpdateXAndInfCostsForColumnsWithChangedBoundsTableau()
					} else {
						err = s.DetectRowsWithChangedBoundsForColumn(0)
					}
					h.budget.FailAfter(-1)
					if err == nil {
						success = true
						done = true
					} else {
						failures++
						if (s.Core.RPrimal.PairCoreSolverBase.calls.pending() || s.ColumnBuffer.calls.pending() || (s.Core.RPrimal.Factorization != nil && s.Core.RPrimal.Factorization.stopped)) && s.calls.pending() {
							t.Fatal("parent adopted child failure")
						}
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						if s.UpdateXAndInfCostsForColumnWithChangedBounds(0) != ErrExactVectorPending {
							t.Fatal("lost stopped owner")
						}
						if op == "update" && s.pendingCleanupDelta == nil {
							t.Fatal("lost local delta owner")
						}
					}
					h.must(s.Close())
					lpExactVectorNamedCensus(t, h, nil)
				})
				if done {
					break
				}
			}
			if failures == 0 || !success {
				t.Fatal("incomplete physical campaign", direct, op, failures, success)
			}
		}
	}
}

func TestLARChangedBoundsSourcePrefixesAndRetainedMarks(t *testing.T) {
	for _, op := range []string{"heading", "tableau-column", "row-domain", "basic-cost-domain", "batch-prefix", "Ax", "cost-child", "free", "cost-refresh", "LU-row-domain", "basic-inf-domain", "stale-reduced-cost"} {
		t.Run(op, func(t *testing.T) {
			h := newLPPairHarness(t)
			s := lpNewRemovalModel(t, h, "2", "3", false)
			c := s.Core
			h.must(s.RowsWithChangedBounds.Clear())
			h.must(s.Bounds.ChangedColumns.Clear())
			var err error
			want := ErrBounds
			switch op {
			case "heading":
				err = s.DetectRowsWithChangedBoundsForColumn(99)
			case "tableau-column":
				err = s.DetectRowsOfBoundChangeColumnForNBasicColumnTableau(99)
			case "row-domain":
				h.must(s.RowsWithChangedBounds.Resize(0))
				err = s.DetectRowsWithChangedBoundsForColumn(0)
			case "basic-cost-domain":
				s.Settings.SimplexStrategy = StrategyTableauCosts
				h.must(s.Bounds.Low.Set(2, h.pair("1", "0")))
				h.must(s.Bounds.Types.Set(2, ColumnLowBound))
				err = s.UpdateXAndInfCostsForColumnWithChangedBounds(2)
				if !c.RPrimal.InfSet.Contains(2) {
					t.Fatal("membership must precede cost mark refusal")
				}
			case "batch-prefix":
				h.must(s.Bounds.ChangedColumns.Resize(100))
				h.must(s.Bounds.ChangedColumns.Insert(0))
				h.must(s.Bounds.ChangedColumns.Insert(99))
				err = s.DetectRowsWithChangedBounds()
				if !s.RowsWithChangedBounds.Contains(0) {
					t.Fatal("lost earlier row mark")
				}
			case "Ax":
				h.must(h.m.SetInt32(&c.RX.Data[2].X, 1))
				err = s.UpdateXAndInfCostsForColumnsWithChangedBoundsTableau()
				want = nil
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "cost-child":
				s.Settings.SimplexStrategy = StrategyTableauCosts
				c.RPrimal.UsingInfeasCosts = true
				h.must(s.BasicColumnsWithChangedCost.Resize(100))
				h.must(s.BasicColumnsWithChangedCost.Insert(99))
				err = s.UpdateXAndInfCostsForColumnsWithChangedBoundsTableau()
			case "free":
				h.must(c.RPrimal.InfSet.Insert(0))
				err = s.UpdateXAndInfCostsForColumnWithChangedBounds(0)
				want = nil
				if !c.RPrimal.InfSet.Contains(0) || s.pendingCleanupDelta != nil {
					t.Fatal("source nonbasic no-change skips membership")
				}
			case "LU-row-domain":
				h.must(s.Close())
				s = lpPreparedLARPivot(t, h, "2", "3", "5")
				c = s.Core
				err = s.DetectRowsOfBoundChangeColumnForNBasicColumn(0)
				if len(s.ColumnBuffer.Index) != 1 {
					t.Fatal("solve must finish before row marking")
				}
			case "basic-inf-domain":
				s.Settings.SimplexStrategy = StrategyTableauCosts
				h.must(s.Bounds.Low.Set(2, h.pair("1", "0")))
				h.must(s.Bounds.Types.Set(2, ColumnLowBound))
				h.must(c.RPrimal.InfSet.Resize(0))
				err = s.UpdateXAndInfCostsForColumnWithChangedBounds(2)
				if !c.RPrimal.PairCoreSolverBase.stopped || s.calls.pending() {
					t.Fatal("basic membership failure owner")
				}
			case "stale-reduced-cost":
				s.Settings.SimplexStrategy = StrategyTableauCosts
				c.RPrimal.UsingInfeasCosts = true
				h.must(h.m.SetInt32(&c.RPrimal.D.Data[0], 1))
				_ = lpCaptureFeasibleStdout(t, func() { err = s.UpdateXAndInfCostsForColumnsWithChangedBoundsTableau() })
				want = nil
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "cost-refresh":
				s.Settings.SimplexStrategy = StrategyTableauCosts
				c.RPrimal.UsingInfeasCosts = true
				h.must(s.BasicColumnsWithChangedCost.Resize(3))
				h.must(s.Bounds.Low.Set(0, h.pair("2", "0")))
				h.must(s.Bounds.Types.Set(0, ColumnLowBound))
				h.must(s.Bounds.Upper.Set(2, h.pair("1", "0")))
				h.must(s.Bounds.Types.Set(2, ColumnUpperBound))
				h.must(s.Bounds.ChangedColumns.Insert(0))
				err = s.UpdateXAndInfCostsForColumnsWithChangedBoundsTableau()
				want = nil
				h.check(&c.RX.Data[2], big.NewRat(4, 1), big.NewRat(0, 1))
				lpSparseRationalCheck(t, h, &c.Costs.Data[2], big.NewRat(-1, 1))
				lpSparseRationalCheck(t, h, &c.RPrimal.D.Data[0], big.NewRat(-2, 1))
				lpSparseRationalCheck(t, h, &c.RPrimal.D.Data[1], big.NewRat(-3, 1))
				if !s.BasicColumnsWithChangedCost.Contains(2) || !s.Bounds.ChangedColumns.Contains(0) {
					t.Fatal("consumed source marks")
				}
			}
			if err != want {
				t.Fatal(op, err, want)
			}
			if err != nil && s.DetectRowsWithChangedBounds() != ErrExactVectorPending {
				t.Fatal("lost stop")
			}
			h.must(s.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLARChangedBoundsStoppedEntries(t *testing.T) {
	h := newLPPairHarness(t)
	s := lpNewRemovalModel(t, h, "2", "3", false)
	if s.UpdateXAndInfCostsForColumnWithChangedBounds(99) != ErrBounds {
		t.Fatal("missing heading guard")
	}
	for _, call := range []func() error{
		func() error { return s.DetectRowsOfBoundChangeColumnForNBasicColumn(0) },
		func() error { return s.DetectRowsOfBoundChangeColumnForNBasicColumnTableau(0) },
		func() error { return s.DetectRowsWithChangedBoundsForColumn(0) },
		func() error { return s.DetectRowsOfColumnWithBoundChange(0) },
		s.DetectRowsWithChangedBounds,
		func() error { return s.UpdateXAndInfCostsForColumnWithChangedBounds(0) },
		s.UpdateXAndInfCostsForColumnsWithChangedBounds,
		s.UpdateXAndInfCostsForColumnsWithChangedBoundsTableau,
	} {
		if err := call(); err != ErrExactVectorPending {
			t.Fatal("stopped entry", err)
		}
	}
	h.must(s.Close())
	lpExactVectorNamedCensus(t, h, nil)
}
