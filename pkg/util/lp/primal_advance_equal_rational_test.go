package lp

import (
	"fmt"
	"math/big"
	"strings"
	"testing"
)

func lpRationalAdvanceOracleCost(x *big.Rat, kind ColumnType, low, upper *big.Rat) int64 {
	above := x.Cmp(upper) > 0
	below := x.Cmp(low) < 0
	switch kind {
	case ColumnFixed, ColumnBoxed:
		if above {
			return -1
		}
		if below {
			return 1
		}
	case ColumnLowBound:
		if below {
			return 1
		}
	case ColumnUpperBound:
		if above {
			return -1
		}
	}
	return 0
}

func TestLPRationalAdvanceEqualIndependentModels(t *testing.T) {
	for world := 0; world < 200; world++ {
		h := newLPPairHarness(t)
		f := lpNewRationalCoreFixture(t, h, 2, 4)
		f.settings.SimplexStrategy = StrategyLU
		a := [][]*big.Rat{{big.NewRat(2, 1), big.NewRat(1, 1), big.NewRat(3, 7), big.NewRat(-1, 3)}, {big.NewRat(1, 1), big.NewRat(3, 1), big.NewRat(-2, 11), big.NewRat(5, 13)}}
		x := []*big.Rat{big.NewRat(int64(world%7-3), 2), big.NewRat(int64(world%5-2), 3), new(big.Rat), big.NewRat(2, 7)}
		original, cost := make([]*big.Rat, 4), make([]*big.Rat, 4)
		for j := range x {
			h.must(h.m.Copy(&f.x.Data[j], lpScalarBoundValue(h, x[j].RatString())))
			original[j] = big.NewRat(int64(7+j+world%5), 13)
			cost[j] = new(big.Rat).Set(original[j])
			h.must(h.m.SetString(&f.costs.Data[j], cost[j].RatString()))
		}
		f.types[0] = ColumnBoxed
		f.types[1] = ColumnLowBound
		h.must(h.m.Copy(&f.low.Data[0], lpScalarBoundValue(h, "-1")))
		h.must(h.m.Copy(&f.up.Data[0], lpScalarBoundValue(h, "1")))
		for i := range a {
			rhs := new(big.Rat)
			for j, v := range a[i] {
				h.must(f.a.Set(uint32(i), uint32(j), lpScalarBoundValue(h, v.RatString())))
				rhs.Add(rhs, new(big.Rat).Mul(v, x[j]))
			}
			h.must(h.m.Copy(&f.b.Data[i], lpScalarBoundValue(h, rhs.RatString())))
		}
		m, e := NewRationalPrimalCoreSolver(f.input())
		h.must(e)
		h.must(m.BackupAndNormalizeCosts())
		h.must(m.InitColumnRowNonZeroes())
		h.must(m.SolveBd(2))
		h.must(m.InitInfSet())
		using, only := world%2 == 0, world/2%2 == 0
		if using {
			h.must(m.InitInfeasibilityCosts())
		}
		m.LookForFeasibleSolutionOnly = only
		m.IterationsNoCostGrowth = 73
		for j := range m.D.Data {
			h.must(h.m.SetInt32(&m.D.Data[j], 97))
		}
		for i := range m.Y.Data {
			h.must(h.m.SetInt32(&m.Y.Data[i], 101))
		}
		lows := []*big.Rat{big.NewRat(-1, 1), new(big.Rat)}
		ups := []*big.Rat{big.NewRat(1, 1), new(big.Rat)}
		members := []bool{lpRationalAdvanceOracleCost(x[0], f.types[0], lows[0], ups[0]) != 0, lpRationalAdvanceOracleCost(x[1], f.types[1], lows[1], ups[1]) != 0}
		if using {
			for j := range cost {
				cost[j] = new(big.Rat)
				if j < 2 {
					cost[j].SetInt64(lpRationalAdvanceOracleCost(x[j], f.types[j], lows[j], ups[j]))
				}
			}
		}
		sign := int64(world/4%3 - 1)
		m.SignOfEnteringDelta = int32(sign)
		tx := big.NewRat(int64(world%11-5), 7)
		step := lpScalarBoundValue(h, tx.RatString())
		dx := new(big.Rat).Mul(tx, big.NewRat(sign, 1))
		d0 := new(big.Rat).Quo(new(big.Rat).Sub(new(big.Rat).Mul(big.NewRat(3, 1), a[0][2]), a[1][2]), big.NewRat(5, 1))
		d1 := new(big.Rat).Quo(new(big.Rat).Sub(new(big.Rat).Mul(big.NewRat(2, 1), a[1][2]), a[0][2]), big.NewRat(5, 1))
		x[2].Add(x[2], dx)
		for i, d := range []*big.Rat{d0, d1} {
			x[i].Sub(x[i], new(big.Rat).Mul(d, dx))
		}
		if using {
			for i := 0; i < 2; i++ {
				c := lpRationalAdvanceOracleCost(x[i], f.types[i], lows[i], ups[i])
				cost[i].SetInt64(c)
				members[i] = c != 0
			}
		}
		feasible := !members[0] && !members[1]
		early := only && feasible
		recompute := !early && (feasible == using || !feasible)
		if recompute {
			if feasible && using {
				using = false
				for j := range cost {
					cost[j].Set(original[j])
				}
			} else if !feasible && !using {
				using = true
				for j := range cost {
					cost[j] = new(big.Rat)
					if j < 2 {
						c := lpRationalAdvanceOracleCost(x[j], f.types[j], lows[j], ups[j])
						cost[j].SetInt64(c)
						members[j] = c != 0
					}
				}
			}
		}
		h.must(m.AdvanceOnEnteringEqualLeaving(2, step))
		for j := range x {
			lpSparseRationalCheck(t, h, &f.x.Data[j], x[j])
			lpSparseRationalCheck(t, h, &f.costs.Data[j], cost[j])
		}
		lpSparseRationalCheck(t, h, step, tx)
		if m.UsingInfeasCosts != using || m.InfSet.Contains(0) != members[0] || m.InfSet.Contains(1) != members[1] {
			t.Fatal(world, "cost state")
		}
		counter := uint32(0)
		if early {
			counter = 73
		}
		if m.IterationsNoCostGrowth != counter {
			t.Fatal(world, "counter early return")
		}
		if recompute {
			y0 := new(big.Rat).Quo(new(big.Rat).Sub(new(big.Rat).Mul(big.NewRat(3, 1), cost[0]), cost[1]), big.NewRat(5, 1))
			y1 := new(big.Rat).Quo(new(big.Rat).Sub(new(big.Rat).Mul(big.NewRat(2, 1), cost[1]), cost[0]), big.NewRat(5, 1))
			for j := range cost {
				want := new(big.Rat)
				if j >= 2 {
					want.Set(cost[j])
					want.Sub(want, new(big.Rat).Mul(y0, a[0][j]))
					want.Sub(want, new(big.Rat).Mul(y1, a[1][j]))
				}
				lpSparseRationalCheck(t, h, &m.D.Data[j], want)
			}
		} else {
			for j := range m.D.Data {
				lpSparseRationalCheck(t, h, &m.D.Data[j], big.NewRat(97, 1))
			}
		}
		if f.basis[0] != 0 || f.basis[1] != 1 || m.advanceDelta != nil {
			t.Fatal("changed basis or retained expression")
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPRationalAdvanceEqualPhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, op := range []string{"positive", "negative", "zero", "cost-switch"} {
		for _, direct := range []bool{false, true} {
			success, failures := false, 0
			childStops := 0
			for ordinal := int64(0); ordinal < 1000; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("%s/%t/%d", op, direct, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					f := lpNewRationalCoreFixture(t, h, 2, 3)
					f.settings.SimplexStrategy = StrategyLU
					huge := strings.Repeat("1234567890", 12)
					for i := uint32(0); i < 2; i++ {
						h.must(f.a.Set(i, i, lpScalarBoundValue(h, fmt.Sprint(3+i))))
						h.must(f.a.Set(i, 2, lpScalarBoundValue(h, huge+"7/13")))
					}
					m, e := NewRationalPrimalCoreSolver(f.input())
					h.must(e)
					h.must(m.InitColumnRowNonZeroes())
					h.must(m.SolveBd(2))
					m.SignOfEnteringDelta = 1
					if op == "negative" {
						m.SignOfEnteringDelta = -1
					}
					if op == "zero" {
						m.SignOfEnteringDelta = 0
					}
					if op == "cost-switch" {
						for j := range f.costs.Data {
							h.must(h.m.SetString(&f.costs.Data[j], huge+"3/17"))
						}
						h.must(m.BackupAndNormalizeCosts())
						h.must(m.InitInfeasibilityCosts())
					}
					m.IterationsNoCostGrowth = 73
					step := lpScalarBoundValue(h, huge+"9/19")
					h.budget.FailAfter(ordinal)
					err := m.AdvanceOnEnteringEqualLeaving(2, step)
					h.budget.FailAfter(-1)
					if err == nil {
						success, done = true, true
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						if m.calls.pending() && m.RationalCoreSolverBase.calls.pending() {
							t.Fatal("adopted child receipt")
						}
						if m.advanceDelta != nil && m.RationalCoreSolverBase.calls.pending() {
							childStops++
							if m.work != nil {
								t.Fatal("scalar constructor temps survived into UpdateX")
							}
						}
						if m.IterationsNoCostGrowth != 73 {
							t.Fatal("counter reset before success")
						}
						if e = m.AdvanceOnEnteringEqualLeaving(2, step); e != ErrExactVectorPending {
							t.Fatal(e)
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
				t.Fatal(op, direct, success, failures)
			}
			if direct && op != "zero" && childStops == 0 {
				t.Fatal("no stopped child borrowing scalar advancement value", op)
			}
		}
	}
}

func TestLPRationalAdvanceEqualLiveStepAliases(t *testing.T) {
	for _, target := range []int{0, 1, 2} {
		h := newLPPairHarness(t)
		f := lpNewRationalCoreFixture(t, h, 2, 3)
		f.settings.SimplexStrategy = StrategyLU
		for i := uint32(0); i < 2; i++ {
			h.must(f.a.Set(i, i, lpScalarBoundValue(h, "1")))
			h.must(f.a.Set(i, 2, lpScalarBoundValue(h, fmt.Sprint(i+2))))
		}
		for j := range f.x.Data {
			h.must(h.m.Copy(&f.x.Data[j], lpScalarBoundValue(h, fmt.Sprint(j+3))))
		}
		for i := 0; i < 2; i++ {
			h.must(h.m.Copy(&f.b.Data[i], lpScalarBoundValue(h, fmt.Sprint(i+3+(i+2)*5))))
		}
		m, e := NewRationalPrimalCoreSolver(f.input())
		h.must(e)
		h.must(m.InitColumnRowNonZeroes())
		h.must(m.SolveBd(2))
		m.SignOfEnteringDelta = -1
		tx := big.NewRat(int64(target+3), 1)
		h.must(m.AdvanceOnEnteringEqualLeaving(2, &f.x.Data[target]))
		lpSparseRationalCheck(t, h, &f.x.Data[2], new(big.Rat).Sub(big.NewRat(5, 1), tx))
		for i := 0; i < 2; i++ {
			lpSparseRationalCheck(t, h, &f.x.Data[i], new(big.Rat).Add(big.NewRat(int64(i+3), 1), new(big.Rat).Mul(big.NewRat(int64(i+2), 1), tx)))
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPRationalAdvanceEqualSourcePrefixes(t *testing.T) {
	for _, phase := range []string{"nil", "negative-index", "residual", "cost-assert", "cost-missing", "changed-index", "tableau-infeasible", "tableau-feasible", "late-bounds", "reduced-failure"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			f := lpNewRationalCoreFixture(t, h, 1, 2)
			h.must(f.a.Set(0, 0, lpScalarBoundValue(h, "1")))
			h.must(f.a.Set(0, 1, lpScalarBoundValue(h, "2")))
			m, e := NewRationalPrimalCoreSolver(f.input())
			h.must(e)
			h.must(m.Ed.SetValue(lpScalarBoundValue(h, "2"), 0))
			m.IterationsNoCostGrowth = 73
			m.SignOfEnteringDelta = 1
			for j := range m.D.Data {
				h.must(h.m.SetInt32(&m.D.Data[j], 97))
			}
			step := lpScalarBoundValue(h, "1")
			entering := int32(1)
			want := error(nil)
			switch phase {
			case "nil":
				step = nil
				want = ErrBounds
			case "negative-index":
				entering = -1
				want = ErrBounds
			case "residual":
				h.must(h.m.Copy(&f.x.Data[1], lpScalarBoundValue(h, "1")))
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "cost-assert":
				m.UsingInfeasCosts = true
				m.LookForFeasibleSolutionOnly = true
				h.must(h.m.SetInt32(&f.costs.Data[1], 5))
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "cost-missing":
				m.UsingInfeasCosts = true
				h.must(f.costs.resizeData(0))
				want = ErrBounds
			case "changed-index":
				m.UsingInfeasCosts = true
				m.Ed.Index = []uint32{99}
				want = ErrBounds
			case "tableau-infeasible":
				m.UsingInfeasCosts = true
				f.types[0] = ColumnLowBound
				if sourceDebugEnabled {
					want = ErrAssertion
				} else {
					want = ErrBounds
				}
			case "tableau-feasible":
				m.UsingInfeasCosts = true
			case "late-bounds":
				m.UsingInfeasCosts = true
				f.types[0] = ColumnLowBound
				h.must(f.low.resizeData(0))
				want = ErrBounds
			case "reduced-failure":
				m.Settings.SimplexStrategy = StrategyLU
				h.must(m.InfSet.Insert(0))
				want = ErrBounds
			}
			var err error
			out := lpCaptureFeasibleStdout(t, func() { err = m.AdvanceOnEnteringEqualLeaving(entering, step) })
			if err != want {
				t.Fatal(err, want)
			}
			preStop := phase == "nil" || phase == "negative-index" || (phase == "residual" && sourceDebugEnabled)
			if preStop {
				x := int64(0)
				if phase == "residual" {
					x = 1
				}
				lpSparseRationalCheck(t, h, &f.x.Data[1], big.NewRat(x, 1))
				lpSparseRationalCheck(t, h, &f.x.Data[0], big.NewRat(0, 1))
			} else {
				x := int64(1)
				if phase == "residual" {
					x = 2
				}
				lpSparseRationalCheck(t, h, &f.x.Data[1], big.NewRat(x, 1))
				lpSparseRationalCheck(t, h, &f.x.Data[0], big.NewRat(-2, 1))
			}
			if phase == "negative-index" && m.advanceDelta == nil {
				t.Fatal("lost full expression on child guard")
			}
			if phase == "residual" && sourceDebugEnabled {
				if out != "precise x is off (m_b[0] = 0 left side = 2 delta = -2 iters = 0)\n" {
					t.Fatal(out)
				}
			} else if out != "" {
				t.Fatal("unexpected residual diagnostic", out)
			}
			expectedCounter := uint32(73)
			if phase == "tableau-feasible" || (phase == "residual" && !sourceDebugEnabled) {
				expectedCounter = 0
			}
			if m.IterationsNoCostGrowth != expectedCounter {
				t.Fatal("counter publication prefix")
			}
			for j := range m.D.Data {
				lpSparseRationalCheck(t, h, &m.D.Data[j], big.NewRat(97, 1))
			}
			if phase == "cost-assert" {
				lpSparseRationalCheck(t, h, &f.costs.Data[1], big.NewRat(5, 1))
			}
			if phase == "tableau-infeasible" {
				lpSparseRationalCheck(t, h, &f.costs.Data[0], big.NewRat(1, 1))
				if !m.InfSet.Contains(0) {
					t.Fatal("lost infeasible membership prefix")
				}
			}
			if err != nil {
				if e = m.AdvanceOnEnteringEqualLeaving(1, step); e != ErrExactVectorPending {
					t.Fatal(e)
				}
			}
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPRationalAdvanceEqualOlderComparison(t *testing.T) {
	for _, large := range []bool{false, true} {
		matched := false
		for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
			t.Run(fmt.Sprintf("%t/%d", large, ordinal), func(t *testing.T) {
				h := newLPPairHarness(t)
				f := lpNewRationalCoreFixture(t, h, 1, 2)
				h.must(f.a.Set(0, 0, lpScalarBoundValue(h, "1")))
				h.must(f.a.Set(0, 1, lpScalarBoundValue(h, "2")))
				m, e := NewRationalPrimalCoreSolver(f.input())
				h.must(e)
				m.SignOfEnteringDelta = 1
				step := lpScalarBoundValue(h, "3")
				v := h.pair(strings.Repeat("9876543210", 100)+"/7", strings.Repeat("1234567890", 100)+"/13")
				if large {
					step = &v.X
				}
				var caller exactEtaCalls
				before, e := caller.begin(h.m)
				h.must(e)
				h.budget.FailAfter(ordinal)
				_, err := h.m.Compare(&v.X, &v.Y)
				h.budget.FailAfter(-1)
				err = caller.after(h.m, before, err)
				old := h.m.ComparisonInvocationState()
				if old.Active {
					matched = true
					if !lpExactPriorityPhysicalRefusal(err) {
						t.Fatal(err)
					}
					err = m.AdvanceOnEnteringEqualLeaving(1, step)
					if m.calls.comparison != 0 || m.RationalCoreSolverBase.calls.comparison != 0 || f.a.calls.comparison != 0 {
						t.Fatal("adopted caller comparison", err)
					}
					if err == nil {
						tx := big.NewRat(3, 1)
						if large {
							tx = mustLPRat(t, strings.Repeat("9876543210", 100)+"/7")
						}
						lpSparseRationalCheck(t, h, &f.x.Data[1], tx)
						lpSparseRationalCheck(t, h, &f.x.Data[0], new(big.Rat).Mul(big.NewRat(-2, 1), tx))
					}
				} else if err != nil && !lpExactPriorityPhysicalRefusal(err) {
					t.Fatal(err)
				}
				h.must(m.Close())
				f.close()
				if old.Active && old != h.m.ComparisonInvocationState() {
					t.Fatal("cleanup consumed caller comparison")
				}
				h.must(caller.cleanupRaw(h.m))
				h.must(caller.cleanupRoots(h.m))
				lpExactVectorNamedCensus(t, h, nil)
			})
		}
		if !matched {
			t.Fatal("no actual caller comparison")
		}
	}
}
