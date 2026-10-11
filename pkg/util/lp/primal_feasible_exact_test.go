package lp

import (
	"fmt"
	"math/big"
	"strings"
	"testing"
)

func TestLPExactFindFeasibleIndependentSystems(t *testing.T) {
	for _, strategy := range []SimplexStrategy{StrategyLU, StrategyTableauCosts, StrategyTableauRows} {
		for world := 0; world < 120; world++ {
			h := newLPPairHarness(t)
			f := lpNewPairCoreFixture(t, h, 1, 3)
			f.settings.SimplexStrategy = strategy
			gap := big.NewRat(int64(1+world%17), int64(1+world%7))
			a := big.NewRat(int64(1+world%11), int64(1+world%5))
			epsilon := big.NewRat(int64(world%7-3), 13)
			negative := new(big.Rat).Neg(gap)
			h.must(f.a.Set(0, 0, &h.pair("1", "0").X))
			h.must(f.a.Set(0, 1, &h.pair(new(big.Rat).Neg(a).RatString(), "0").X))
			h.must(f.a.Set(0, 2, &h.pair("-1", "0").X))
			f.types[0], f.types[1], f.types[2] = ColumnLowBound, ColumnLowBound, ColumnFixed
			h.must(f.x.Data[0].Copy(h.m, h.pair(negative.RatString(), epsilon.RatString())))
			for _, p := range []*RationalPair{&f.x.Data[2], &f.low.Data[2], &f.up.Data[2]} {
				h.must(p.Copy(h.m, h.pair(negative.RatString(), epsilon.RatString())))
			}
			m, err := NewPairPrimalCoreSolver(f.input())
			h.must(err)
			m.Names = lpFeasibleNames{}
			if strategy != StrategyLU {
				h.must(m.InitInfSet())
			}
			if strategy == StrategyTableauCosts {
				h.must(m.InitReducedCostsTableau())
			}
			m.Status = StatusUnbounded
			f.settings.SetResourceLimit(&lpTestResourceLimit{})
			h.must(m.FindFeasibleSolution())
			if !m.LookForFeasibleSolutionOnly || !m.CurrentXIsFeasible() || m.TotalIterations != 1 {
				t.Fatal(strategy, world, m.Status, m.TotalIterations)
			}
			h.check(&f.x.Data[0], new(big.Rat), new(big.Rat))
			h.check(&f.x.Data[1], new(big.Rat).Quo(gap, a), new(big.Rat).Quo(new(big.Rat).Neg(epsilon), a))
			if off, err := m.AMultXIsOff(); err != nil || off {
				t.Fatal("independent equation", off, err)
			}
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		}
	}
}

func TestLPExactNonbasicIndependentBoundOracle(t *testing.T) {
	for world := 0; world < 360; world++ {
		h := newLPPairHarness(t)
		f := lpNewPairCoreFixture(t, h, 0, 3)
		f.settings.SimplexStrategy = StrategyTableauRows
		kinds := []ColumnType{ColumnFree, ColumnLowBound, ColumnUpperBound, ColumnFixed, ColumnBoxed}
		kindsForColumns := make([]ColumnType, 3)
		want := true
		for j := 0; j < 3; j++ {
			kind := kinds[(world+j)%5]
			f.types[j] = kind
			kindsForColumns[j] = kind
			low, up := big.NewRat(int64(world%13-6), 7), big.NewRat(int64(world%13-6), 7)
			if kind != ColumnFixed {
				up.Add(up, big.NewRat(2, 1))
			}
			x := new(big.Rat).Add(low, big.NewRat(int64((world/5+j)%5-1), 1))
			y := big.NewRat(int64((world/25+j)%3-1), 11)
			h.must(f.low.Data[j].Copy(h.m, h.pair(low.RatString(), "0")))
			h.must(f.up.Data[j].Copy(h.m, h.pair(up.RatString(), "0")))
			h.must(f.x.Data[j].Copy(h.m, h.pair(x.RatString(), y.RatString())))
			below := x.Cmp(low) < 0 || (x.Cmp(low) == 0 && y.Sign() < 0)
			above := x.Cmp(up) > 0 || (x.Cmp(up) == 0 && y.Sign() > 0)
			feasible := kind == ColumnFree || (kind == ColumnLowBound && !below) || (kind == ColumnUpperBound && !above) || ((kind == ColumnFixed || kind == ColumnBoxed) && !below && !above)
			want = want && feasible
		}
		m, err := NewPairPrimalCoreSolver(f.input())
		h.must(err)
		m.Names = lpFeasibleNames{}
		var ok bool
		output := lpCaptureFeasibleStdout(t, func() { ok, err = m.NonBasicColumnsAreSetCorrectly() })
		h.must(err)
		if ok != want || (want && output != "") || (!want && !strings.Contains(output, "column_index")) {
			t.Fatal(world, kindsForColumns, ok, want, output)
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPExactFindFeasibleEntryPrefixes(t *testing.T) {
	for _, strategy := range []SimplexStrategy{StrategyLU, StrategyTableauCosts, StrategyTableauRows} {
		for _, phase := range []string{"early", "stopped", "cancel-error", "infeasible-nonbasic", "missing-name", "bad-nonbasic-index"} {
			t.Run(fmt.Sprintf("%d/%s", strategy, phase), func(t *testing.T) {
				h := newLPPairHarness(t)
				f, m := lpExactSolveBoundedFixture(t, h, strategy)
				m.Status = StatusUnbounded
				m.TotalIterations = 91
				limit := &lpExactSolveCancelSequence{}
				f.settings.SetResourceLimit(limit)
				wantErr := error(nil)
				wantCount := uint32(0)
				wantStatus := StatusFeasible
				wantOnly := true
				switch phase {
				case "stopped":
					m.stopped = true
					wantErr = ErrExactVectorPending
					wantCount = 91
					wantStatus = StatusUnbounded
					wantOnly = false
				case "cancel-error":
					limit.errorAt = 1
					h.must(m.InfSet.Insert(0))
					f.types[0] = ColumnLowBound
					h.must(f.low.Data[0].Copy(h.m, h.pair("2", "0")))
					wantErr = lpErrExactSolveCancel
					wantCount = 1
					wantStatus = StatusUnknown
				case "infeasible-nonbasic", "missing-name":
					f.types[1] = ColumnBoxed
					h.must(f.low.Data[1].Copy(h.m, h.pair("1", "0")))
					h.must(f.up.Data[1].Copy(h.m, h.pair("2", "0")))
					if phase == "missing-name" {
						m.Names = nil
					}
					if sourceDebugEnabled {
						wantErr = ErrAssertion
						if phase == "missing-name" {
							wantErr = ErrBounds
						}
						wantCount = 91
						wantStatus = StatusUnbounded
					}
				case "bad-nonbasic-index":
					*m.NBasis = []uint32{99}
					if sourceDebugEnabled {
						wantErr = ErrBounds
						wantCount = 91
						wantStatus = StatusUnbounded
					}
				}
				var err error
				lpCaptureFeasibleStdout(t, func() { err = m.FindFeasibleSolution() })
				if err != wantErr || m.TotalIterations != wantCount || m.Status != wantStatus || m.LookForFeasibleSolutionOnly != wantOnly {
					t.Fatal("prefix", err, wantErr, m.TotalIterations, wantCount, m.Status, wantStatus, m.LookForFeasibleSolutionOnly)
				}
				if err != nil && m.FindFeasibleSolution() != ErrExactVectorPending {
					t.Fatal("stopped retry")
				}
				h.must(m.Close())
				f.close()
				lpExactVectorNamedCensus(t, h, nil)
			})
		}
	}
}

func TestLPExactNonbasicEmptyAndStoppedValidation(t *testing.T) {
	for _, bad := range []bool{false, true} {
		h := newLPPairHarness(t)
		f := lpNewPairCoreFixture(t, h, 0, 0)
		f.settings.SimplexStrategy = StrategyTableauRows
		m, err := NewPairPrimalCoreSolver(f.input())
		h.must(err)
		if bad {
			*m.NBasis = []uint32{99}
		}
		ok, err := m.NonBasicColumnsAreSetCorrectly()
		if bad {
			if ok || err != ErrBounds {
				t.Fatal("invalid nonbasic", ok, err)
			}
			if ok, err = m.NonBasicColumnsAreSetCorrectly(); ok || err != ErrExactVectorPending {
				t.Fatal("stopped validator", ok, err)
			}
		} else if !ok || err != nil {
			t.Fatal("empty nonbasic domain", ok, err)
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPExactFindFeasiblePhysicalOwnership(t *testing.T) {
	for _, strategy := range []SimplexStrategy{StrategyLU, StrategyTableauCosts, StrategyTableauRows} {
		for _, direct := range []bool{false, true} {
			success, failures := false, 0
			for ordinal := int64(0); ordinal < 1000; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("%d/%t/%d", strategy, direct, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					f := lpNewPairCoreFixture(t, h, 1, 3)
					f.settings.SimplexStrategy = strategy
					huge := strings.Repeat("1234567890", 20) + "7/13"
					initial := h.pair("-"+huge, "0")
					h.must(f.a.Set(0, 0, &h.pair("1", "0").X))
					h.must(f.a.Set(0, 1, &h.pair("-1", "0").X))
					h.must(f.a.Set(0, 2, &h.pair("-1", "0").X))
					f.types[0], f.types[1], f.types[2] = ColumnLowBound, ColumnLowBound, ColumnFixed
					for _, p := range []*RationalPair{&f.x.Data[0], &f.x.Data[2], &f.low.Data[2], &f.up.Data[2]} {
						h.must(p.Copy(h.m, initial))
					}
					m, err := NewPairPrimalCoreSolver(f.input())
					h.must(err)
					m.Names = lpFeasibleNames{}
					if strategy != StrategyLU {
						h.must(m.InitInfSet())
					}
					if strategy == StrategyTableauCosts {
						h.must(m.InitReducedCostsTableau())
					}
					f.settings.SetResourceLimit(&lpTestResourceLimit{})
					m.Status = StatusUnbounded
					h.budget.FailAfter(ordinal)
					err = m.FindFeasibleSolution()
					h.budget.FailAfter(-1)
					if err == nil {
						success, done = true, true
						if !m.CurrentXIsFeasible() || m.TotalIterations != 1 {
							t.Fatal("feasible success", m.TotalIterations, m.Status)
						}
						h.check(&f.x.Data[1], mustLPRat(t, huge), new(big.Rat))
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						if !m.LookForFeasibleSolutionOnly || m.FindFeasibleSolution() != ErrExactVectorPending {
							t.Fatal("stopped feasibility entry")
						}
					}
					h.must(m.Close())
					f.close()
					lpExactVectorNamedCensus(t, h, nil)
				})
				if done {
					break
				}
			}
			if !success || (direct && failures == 0) {
				t.Fatal("campaign", strategy, direct, success, failures)
			}
		}
	}
}
