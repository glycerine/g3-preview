package lp

import (
	"errors"
	"fmt"
	"math/big"
	"math/rand/v2"
	"reflect"
	"strings"
	"testing"
)

func TestLARInfCleanupIndependentTableauModel(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	h := newLPPairHarness(t)
	rng := rand.New(rand.NewPCG(18713, 4789))
	for world := 0; world < 160; world++ {
		a := big.NewRat(int64(rng.IntN(11)-5), int64(rng.IntN(7)+1))
		b := big.NewRat(int64(rng.IntN(13)-6), int64(rng.IntN(11)+1))
		s := lpNewRemovalModel(t, h, a.RatString(), b.RatString(), false)
		c := s.Core
		r := c.RPrimal
		low := h.pair("3/7", "1/5")
		upper := h.pair("-2/11", "-1/7")
		if world%7 == 0 {
			h.must(low.Copy(h.m, h.pair("0", "0")))
		}
		h.must(s.Bounds.Low.Set(0, low))
		h.must(s.Bounds.Types.Set(0, ColumnLowBound))
		h.must(s.Bounds.Upper.Set(1, upper))
		h.must(s.Bounds.Types.Set(1, ColumnUpperBound))
		lx := big.NewRat(3, 7)
		ly := big.NewRat(1, 5)
		if world%7 == 0 {
			lx.SetInt64(0)
			ly.SetInt64(0)
		}
		wantX := new(big.Rat).Add(new(big.Rat).Mul(a, lx), new(big.Rat).Mul(b, big.NewRat(-2, 11)))
		wantY := new(big.Rat).Add(new(big.Rat).Mul(a, ly), new(big.Rat).Mul(b, big.NewRat(-1, 7)))
		basicLow := new(big.Rat).Set(wantX)
		if world%2 == 0 {
			basicLow.Add(basicLow, big.NewRat(1, 13))
		} else {
			basicLow.Sub(basicLow, big.NewRat(1, 13))
		}
		h.must(s.Bounds.Low.Set(2, h.pair(basicLow.RatString(), wantY.RatString())))
		h.must(s.Bounds.Types.Set(2, ColumnLowBound))
		oldD := [2]*big.Rat{big.NewRat(int64(world-17), 19), big.NewRat(int64(world+3), 23)}
		oldCost := [2]*big.Rat{big.NewRat(int64(world+1), 29), big.NewRat(int64(7-world), 31)}
		for j := 0; j < 2; j++ {
			h.must(h.m.SetString(&r.D.Data[j], oldD[j].RatString()))
			h.must(h.m.SetString(&c.Costs.Data[j], oldCost[j].RatString()))
		}
		h.must(h.m.SetInt32(&c.Costs.Data[2], 7))
		h.must(r.InfSet.Resize(6))
		for _, j := range []uint32{2, 0, 1, 5} {
			h.must(r.InfSet.Insert(j))
		}
		h.must(s.CleanInfSetOfRSolverAfterPop())
		h.check(&c.RX.Data[0], lx, ly)
		h.check(&c.RX.Data[1], big.NewRat(-2, 11), big.NewRat(-1, 7))
		h.check(&c.RX.Data[2], wantX, wantY)
		for j := 0; j < 2; j++ {
			lpSparseRationalCheck(t, h, &r.D.Data[j], new(big.Rat).Sub(oldD[j], oldCost[j]))
			lpSparseRationalCheck(t, h, &c.Costs.Data[j], big.NewRat(0, 1))
		}
		lpSparseRationalCheck(t, h, &c.Costs.Data[2], big.NewRat(7, 1))
		wantSet := []uint32(nil)
		if world%2 == 0 {
			wantSet = []uint32{2}
		}
		if !reflect.DeepEqual(r.InfSet.Indices(), wantSet) && !(len(wantSet) == 0 && r.InfSet.Empty()) {
			t.Fatal(world, r.InfSet.Indices(), wantSet)
		}
		if r.InfSet.DataSize() != 6 || s.pendingCleanupDelta != nil || s.work != nil || s.rationalWork != nil {
			t.Fatal("cleanup ownership/domain")
		}
		h.must(s.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLARInfCleanupActualNonunitLU(t *testing.T) {
	h := newLPPairHarness(t)
	rng := rand.New(rand.NewPCG(47182, 317))
	for world := 0; world < 100; world++ {
		a := big.NewRat(int64(rng.IntN(17)-8), int64(rng.IntN(11)+1))
		basis := big.NewRat(int64(rng.IntN(7)+1), int64(rng.IntN(5)+1))
		if world%2 == 0 {
			basis.Neg(basis)
		}
		s := lpPreparedLARPivot(t, h, a.RatString(), "2/3", basis.RatString())
		c := s.Core
		h.must(c.RX.Data[2].Copy(h.m, h.pair("7/11", "-3/13")))
		lo := h.pair("1/3", "1/5")
		h.must(s.Bounds.Low.Set(0, lo))
		h.must(c.RPrimal.InfSet.Insert(0))
		h.must(c.RPrimal.InfSet.Insert(2))
		h.must(s.CleanInfSetOfRSolverAfterPop())
		coef := new(big.Rat).Quo(a, basis)
		h.check(&c.RX.Data[0], big.NewRat(1, 3), big.NewRat(1, 5))
		h.check(&c.RX.Data[2], new(big.Rat).Sub(big.NewRat(7, 11), new(big.Rat).Mul(coef, big.NewRat(1, 3))), new(big.Rat).Sub(big.NewRat(-3, 13), new(big.Rat).Mul(coef, big.NewRat(1, 5))))
		if !c.RPrimal.InfSet.Empty() {
			t.Fatal("free basic member retained", c.RPrimal.InfSet.Indices())
		}
		h.must(s.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLARInfCleanupTableauCostsAndEmptyLocalSet(t *testing.T) {
	for _, stale := range []bool{false, true} {
		t.Run(fmt.Sprint(stale), func(t *testing.T) {
			h := newLPPairHarness(t)
			s := lpNewRemovalModel(t, h, "2", "3", false)
			c := s.Core
			r := c.RPrimal
			s.Settings.SimplexStrategy = StrategyTableauCosts
			r.UsingInfeasCosts = true
			h.must(s.BasicColumnsWithChangedCost.Resize(3))
			if stale {
				h.must(s.Bounds.Low.Set(0, h.pair("2", "0")))
				h.must(s.Bounds.Types.Set(0, ColumnLowBound))
				h.must(s.Bounds.Upper.Set(2, h.pair("1", "0")))
				h.must(s.Bounds.Types.Set(2, ColumnUpperBound))
				h.must(r.InfSet.Insert(0))
			} else {
				h.must(s.Bounds.Low.Set(2, h.pair("-1", "0")))
				h.must(s.Bounds.Types.Set(2, ColumnLowBound))
				h.must(r.InfSet.Insert(2))
				h.must(h.m.SetInt32(&c.Costs.Data[2], 1))
				h.must(h.m.SetInt32(&r.D.Data[0], 2))
				h.must(h.m.SetInt32(&r.D.Data[1], 3))
			}
			var err error
			stdout := lpCaptureFeasibleStdout(t, func() { err = s.CleanInfSetOfRSolverAfterPop() })
			if stale {
				if sourceDebugEnabled {
					if err != ErrAssertion || !strings.Contains(stdout, "infeasibility_cost_is_correct_for_column does not hold") {
						t.Fatal(err, stdout)
					}
				} else {
					h.must(err)
				}
				h.check(&c.RX.Data[2], big.NewRat(4, 1), big.NewRat(0, 1))
				if !r.InfSet.Contains(2) || !s.BasicColumnsWithChangedCost.Contains(2) {
					t.Fatal("source member changed-cost set missing")
				}
				// Independently: x2=4 exceeds upper1, so its default-sign cost is -1.
				// The empty local set leaves stored cost0 and reduced costs unchanged.
				lpSparseRationalCheck(t, h, &c.Costs.Data[2], big.NewRat(0, 1))
				lpSparseRationalCheck(t, h, &r.D.Data[0], big.NewRat(0, 1))
			} else {
				h.must(err)
				if stdout != "" || !r.InfSet.Empty() {
					t.Fatal(stdout, r.InfSet.Indices())
				}
				for j := 0; j < 3; j++ {
					lpSparseRationalCheck(t, h, &c.Costs.Data[j], big.NewRat(0, 1))
					lpSparseRationalCheck(t, h, &r.D.Data[j], big.NewRat(0, 1))
				}
			}
			h.must(s.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLARInfCleanupSetSnapshotAndSourcePrefixes(t *testing.T) {
	set, err := NewIntSet(10)
	if err != nil {
		t.Fatal(err)
	}
	for _, j := range []uint32{8, 1, 7, 2, 6, 3} {
		if err = set.Insert(j); err != nil {
			t.Fatal(err)
		}
	}
	if err = CleanLargeElementsAfterPop(5, set); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(set.Indices(), []uint32{3, 1, 2}) || set.DataSize() != 10 {
		t.Fatal(set.Indices(), set.DataSize())
	}
	if err = ShrinkInfSetAfterPop(3, set); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(set.Indices(), []uint32{2, 1}) || set.DataSize() != 3 {
		t.Fatal(set.Indices(), set.DataSize())
	}
	if CleanLargeElementsAfterPop(0, nil) != ErrBounds || ShrinkInfSetAfterPop(0, nil) != ErrBounds || ShrinkInfSetAfterPop(^uint32(0), set) != ErrBounds {
		t.Fatal("set metadata guard")
	}
	for _, which := range []string{"heading", "delta", "basic-child", "d", "costs", "basic-feasibility", "cost-update", "validator-child", "empty"} {
		t.Run(which, func(t *testing.T) {
			h := newLPPairHarness(t)
			s := lpNewRemovalModel(t, h, "2", "3", false)
			c := s.Core
			r := c.RPrimal
			h.must(s.Bounds.Low.Set(0, h.pair("1", "0")))
			h.must(s.Bounds.Types.Set(0, ColumnLowBound))
			h.must(r.InfSet.Insert(0))
			savedX, savedCost := c.RX.Data, c.Costs.Data
			var want error = ErrBounds
			switch which {
			case "heading":
				c.RHeading = nil
			case "delta":
				c.RX.Data = nil
			case "basic-child":
				c.RA.Columns[0][0].Offset = 99
			case "d":
				h.must(r.D.Resize(0))
			case "costs":
				c.Costs.Data = nil
			case "basic-feasibility":
				r.InfSet.Erase(0)
				h.must(r.InfSet.Insert(2))
				h.must(s.Bounds.Types.Set(2, ColumnLowBound))
				c.RX.Data = c.RX.Data[:2]
			case "cost-update":
				r.InfSet.Erase(0)
				h.must(r.InfSet.Insert(2))
				s.Settings.SimplexStrategy = StrategyTableauCosts
				r.stopped = true
				want = ErrExactVectorPending
			case "validator-child":
				h.must(r.InfSet.Clear())
				s.Settings.SimplexStrategy = StrategyTableauCosts
				r.UsingInfeasCosts = true
				v := h.pair("1", "0")
				initialError := ErrBounds
				if sourceDebugEnabled {
					initialError = ErrAssertion
				}
				if e := c.RA.Set(99, 0, &v.X); e != initialError {
					t.Fatal(e)
				}
				want = nil
				if sourceDebugEnabled {
					want = ErrExactVectorPending
				}
			case "empty":
				h.must(r.InfSet.Clear())
				want = nil
			}
			err := s.CleanInfSetOfRSolverAfterPop()
			if !errors.Is(err, want) {
				t.Fatal(err, want)
			}
			if which == "delta" || which == "basic-child" {
				if s.pendingCleanupDelta == nil {
					t.Fatal("lost borrowed delta owner")
				}
			}
			if which == "d" || which == "costs" {
				h.check(&savedX[0], big.NewRat(1, 1), big.NewRat(0, 1))
				if s.pendingCleanupDelta != nil {
					t.Fatal("completed delta retained")
				}
			}
			if err != nil && s.CleanInfSetOfRSolverAfterPop() != ErrExactVectorPending {
				t.Fatal("stopped entry")
			}
			c.RX.Data, c.Costs.Data = savedX, savedCost
			h.must(s.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLARInfCleanupPhysicalOwnerBoundaries(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, direct := range []bool{false, true} {
		for _, op := range []string{"clamp", "delta", "subtract", "basic-compare", "cost-update"} {
			failures := 0
			success := false
			owned := 0
			for ordinal := int64(0); ordinal < 280; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("direct=%t/%s/%d", direct, op, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					a := strings.Repeat("9876543210", 100) + "/7"
					b := strings.Repeat("1234567890", 100) + "/13"
					ca := "2/3"
					if op == "delta" {
						ca = a
					}
					s := lpNewRemovalModel(t, h, ca, "-5/7", false)
					c := s.Core
					r := c.RPrimal
					low := h.pair(a, "1/5")
					x := h.pair(b, "0")
					switch op {
					case "clamp":
						h.must(s.Bounds.Low.Set(0, low))
						h.must(s.Bounds.Types.Set(0, ColumnLowBound))
						h.must(c.RX.Data[0].Copy(h.m, x))
						h.must(r.InfSet.Insert(0))
					case "delta":
						h.must(s.Bounds.Low.Set(0, h.pair("1/3", "1/5")))
						h.must(s.Bounds.Types.Set(0, ColumnLowBound))
						h.must(r.InfSet.Insert(0))
					case "subtract":
						h.must(h.m.Copy(&c.Costs.Data[0], &low.X))
						h.must(r.InfSet.Insert(0))
					case "basic-compare":
						h.must(s.Bounds.Low.Set(2, low))
						h.must(s.Bounds.Types.Set(2, ColumnLowBound))
						h.must(c.RX.Data[2].Copy(h.m, x))
						h.must(r.InfSet.Insert(2))
					case "cost-update":
						s.Settings.SimplexStrategy = StrategyTableauCosts
						r.UsingInfeasCosts = true
						h.must(h.m.Copy(&c.Costs.Data[2], &low.X))
						h.must(r.InfSet.Insert(2))
						for j, coef := range []*big.Rat{big.NewRat(2, 3), big.NewRat(-5, 7)} {
							h.must(h.m.SetString(&r.D.Data[j], new(big.Rat).Mul(mustLPRat(t, a), coef).RatString()))
						}
					}
					h.budget.FailAfter(ordinal)
					err := s.CleanInfSetOfRSolverAfterPop()
					h.budget.FailAfter(-1)
					if err == nil {
						success = true
						done = true
						switch op {
						case "clamp":
							h.check(&c.RX.Data[0], mustLPRat(t, a), big.NewRat(1, 5))
							h.check(&c.RX.Data[2], new(big.Rat).Mul(big.NewRat(2, 3), new(big.Rat).Sub(mustLPRat(t, a), mustLPRat(t, b))), big.NewRat(2, 15))
						case "delta":
							h.check(&c.RX.Data[2], new(big.Rat).Mul(mustLPRat(t, a), big.NewRat(1, 3)), new(big.Rat).Mul(mustLPRat(t, a), big.NewRat(1, 5)))
						case "subtract":
							lpSparseRationalCheck(t, h, &r.D.Data[0], new(big.Rat).Neg(mustLPRat(t, a)))
							lpSparseRationalCheck(t, h, &c.Costs.Data[0], big.NewRat(0, 1))
						case "basic-compare":
							if !r.InfSet.Contains(2) {
								t.Fatal("infeasible basic removed")
							}
						case "cost-update":
							for j := 0; j < 3; j++ {
								lpSparseRationalCheck(t, h, &r.D.Data[j], big.NewRat(0, 1))
								lpSparseRationalCheck(t, h, &c.Costs.Data[j], big.NewRat(0, 1))
							}
						}
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						child := r.PairCoreSolverBase.calls.pending() || r.calls.pending()
						if child {
							if s.calls.pending() {
								t.Fatal("outer adopted child invocation")
							}
							owned++
						} else if s.calls.pending() || s.work != nil || s.rationalWork != nil {
							owned++
						}
						if (op == "clamp" || op == "delta") && s.pendingCleanupDelta == nil {
							t.Fatal("lost live source delta")
						}
						if op == "subtract" && s.pendingCleanupDelta != nil {
							t.Fatal("completed delta leaked into arithmetic prefix")
						}
						if s.CleanInfSetOfRSolverAfterPop() != ErrExactVectorPending {
							t.Fatal("stopped entry")
						}
					}
					h.must(s.Close())
					if s.pendingCleanupDelta != nil {
						t.Fatal("delta not destroyed")
					}
					lpExactVectorNamedCensus(t, h, nil)
				})
				if done {
					break
				}
			}
			if !success || failures == 0 || owned == 0 {
				t.Fatal("incomplete campaign", direct, op, failures, owned, success)
			}
		}
	}
}

func TestLARInfCleanupAndCostValidatorLeaveOlderComparisonOwned(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, op := range []string{"cleanup", "cost", "reduced"} {
		matched := false
		for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", op, ordinal), func(t *testing.T) {
				h := newLPPairHarness(t)
				a := strings.Repeat("9876543210", 100) + "/7"
				b := strings.Repeat("1234567890", 100) + "/13"
				s := lpNewRemovalModel(t, h, "2/3", "5/7", false)
				c := s.Core
				r := c.RPrimal
				v := h.pair(a, b)
				h.must(h.m.Copy(&c.RX.Data[0].X, &v.Y))
				h.must(s.Bounds.Low.Set(0, v))
				h.must(s.Bounds.Types.Set(0, ColumnLowBound))
				h.must(r.InfSet.Insert(0))
				if op == "cost" {
					h.must(h.m.Copy(&c.RX.Data[2].X, &v.Y))
					h.must(s.Bounds.Low.Set(2, v))
					h.must(s.Bounds.Types.Set(2, ColumnLowBound))
				}
				if op == "reduced" {
					s.Settings.SimplexStrategy = StrategyTableauCosts
					h.must(h.m.Copy(&c.Costs.Data[2], &v.X))
				}
				var caller exactEtaCalls
				before, err := caller.begin(h.m)
				h.must(err)
				h.budget.FailAfter(ordinal)
				_, err = h.m.Compare(&v.X, &v.Y)
				h.budget.FailAfter(-1)
				err = caller.after(h.m, before, err)
				older := h.m.ComparisonInvocationState()
				if older.Active {
					matched = true
					if !lpExactPriorityPhysicalRefusal(err) {
						t.Fatal(err)
					}
					switch op {
					case "cleanup":
						err = s.CleanInfSetOfRSolverAfterPop()
					case "cost":
						_, err = r.InfeasibilityCostIsCorrectForColumn(2)
					case "reduced":
						_, err = r.ReducedCostsAreCorrectTableau()
					}
					if err == nil || s.calls.pending() || r.PairCoreSolverBase.calls.comparison != 0 {
						t.Fatal("adopted caller comparison", err)
					}
				} else if err != nil && !lpExactPriorityPhysicalRefusal(err) {
					t.Fatal(err)
				}
				h.must(s.Close())
				if older.Active && h.m.ComparisonInvocationState() != older {
					t.Fatal("consumed caller comparison")
				}
				h.must(caller.cleanupRaw(h.m))
				h.must(caller.cleanupRoots(h.m))
				lpExactVectorNamedCensus(t, h, nil)
			})
		}
		if !matched {
			t.Fatal("no actual caller comparison", op)
		}
	}
}
