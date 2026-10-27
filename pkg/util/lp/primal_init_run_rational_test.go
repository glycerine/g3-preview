package lp

import (
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"testing"
)

func TestLPRationalPrimalInitRunIndependentModels(t *testing.T) {
	for world := 0; world < 240; world++ {
		h := newLPPairHarness(t)
		f := lpNewRationalCoreFixture(t, h, 2, 4)
		f.settings.SimplexStrategy = StrategyLU
		f.settings.UseBreakpointsInFeasibilitySearch = world%2 == 0
		if world%2 == 0 {
			f.basis = []uint32{2, 0}
		} else {
			f.basis = []uint32{1, 3}
		}
		coeff := make([][]*big.Rat, 2)
		for i := range coeff {
			coeff[i] = make([]*big.Rat, 4)
			for j := range coeff[i] {
				coeff[i][j] = big.NewRat(int64(3+i-j), 7)
			}
		}
		coeff[0][f.basis[0]] = big.NewRat(2, 1)
		coeff[0][f.basis[1]] = big.NewRat(1, 1)
		coeff[1][f.basis[0]] = big.NewRat(1, 1)
		coeff[1][f.basis[1]] = big.NewRat(3, 1)
		for i := range coeff {
			for j, a := range coeff[i] {
				if a.Sign() != 0 {
					h.must(f.a.Set(uint32(i), uint32(j), &h.pair(a.RatString(), "0").X))
				}
			}
		}
		original, current := make([]*big.Rat, 4), make([]*big.Rat, 4)
		infeas := make([]int64, 4)
		for j := range original {
			original[j] = big.NewRat(int64(world%19+j-9), 13)
			current[j] = new(big.Rat).Set(original[j])
			h.must(h.m.SetString(&f.costs.Data[j], original[j].RatString()))
			f.types[j] = ColumnType((world + j) % 5)
			upper := int64(2)
			if f.types[j] == ColumnFixed {
				upper = 0
			}
			x, eps := int64((world+j)%5-1), int64((world+j)%3-1)
			xr := new(big.Rat).Add(big.NewRat(x, 1), new(big.Rat).Mul(big.NewRat(eps, 1), new(big.Rat).SetFrac(big.NewInt(1), new(big.Int).Exp(big.NewInt(10), big.NewInt(100), nil))))
			h.must(h.m.SetString(&f.x.Data[j], xr.RatString()))
			h.must(f.up.copySlot(&f.up.Data[j], &h.pair(fmt.Sprint(upper), "0").X))
			above := xr.Cmp(big.NewRat(upper, 1)) > 0
			below := xr.Sign() < 0
			switch f.types[j] {
			case ColumnFixed, ColumnBoxed:
				if above {
					infeas[j] = 1
				} else if below {
					infeas[j] = -1
				}
			case ColumnLowBound:
				if below {
					infeas[j] = -1
				}
			case ColumnUpperBound:
				if above {
					infeas[j] = 1
				}
			}
			if !f.settings.UseBreakpointsInFeasibilitySearch {
				infeas[j] = -infeas[j]
			}
		}
		m, e := NewRationalPrimalCoreSolver(f.input())
		h.must(e)
		only, backup := world/4%2 == 0, world/8%2 == 0
		m.LookForFeasibleSolutionOnly = only
		m.Settings.BackupCosts = backup
		m.UsingInfeasCosts = true
		m.BasisSortCounter = 91
		m.TotalIterations = 77
		m.IterationsNoCostGrowth = 83
		m.ColumnNormUpdateCounter = 88
		h.must(h.m.SetInt32(&m.EpsilonOfReducedCost, 9))
		h.must(m.CostsBackup.resizeData(2))
		for j := range m.CostsBackup.Data {
			h.must(h.m.SetInt32(&m.CostsBackup.Data[j], -37))
		}
		h.must(m.BreakpointQueue.Resize(2))
		h.must(m.BreakpointQueue.Enqueue(1, &h.pair("2", "3").X))
		// Stale membership is replaced by actual basic bound feasibility.
		h.must(m.InfSet.Insert(f.nb[0]))
		m.ColumnsNZ = []uint32{0, 91}
		m.RowsNZ = []uint32{0, 99, 103}
		matching := world/16%2 == 0
		if matching {
			m.ColumnsNZ = []uint32{0, 91, 0, 99}
			m.RowsNZ = []uint32{77}
		}
		oldColumns := append([]uint32(nil), m.ColumnsNZ...)
		oldRows := append([]uint32(nil), m.RowsNZ...)
		feasible := true
		for _, j := range f.basis {
			if infeas[j] != 0 {
				feasible = false
			}
		}
		skip := feasible && only
		h.must(m.InitRun())
		if m.BasisSortCounter != 0 || m.TotalIterations != 0 || m.IterationsNoCostGrowth != 0 || m.ColumnNormUpdateCounter != 88 {
			t.Fatal(world, "counter resets")
		}
		if m.UsingInfeasCosts != (skip || !feasible) {
			t.Fatal(world, "using cost mode")
		}
		epsilon := int64(0)
		if skip {
			epsilon = 9
		}
		lpSparseRationalCheck(t, h, &m.EpsilonOfReducedCost, big.NewRat(epsilon, 1))
		qlen := 4
		if skip {
			qlen = 2
		}
		if len(m.BreakpointQueue.inverse) != qlen || m.BreakpointQueue.Size() != 1 || m.BreakpointQueue.heap[1] != 1 {
			t.Fatal(world, "queue resize cleared retained entries")
		}
		lpSparseRationalCheck(t, h, &m.BreakpointQueue.priorities.Data[1], big.NewRat(2, 1))
		if !skip && backup && !only {
			if len(m.CostsBackup.Data) != 4 {
				t.Fatal("backup extent")
			}
			for j := range original {
				lpSparseRationalCheck(t, h, &m.CostsBackup.Data[j], original[j])
			}
		} else {
			if len(m.CostsBackup.Data) != 2 {
				t.Fatal("unexpected backup")
			}
			for j := range m.CostsBackup.Data {
				lpSparseRationalCheck(t, h, &m.CostsBackup.Data[j], big.NewRat(-37, 1))
			}
		}
		for j := range current {
			if !feasible {
				current[j] = new(big.Rat)
				if f.heading[j] >= 0 {
					current[j].SetInt64(infeas[j])
				}
			}
			lpSparseRationalCheck(t, h, &f.costs.Data[j], current[j])
			if m.InfSet.Contains(uint32(j)) != (f.heading[j] >= 0 && infeas[j] != 0) {
				t.Fatal(world, j, "actual membership")
			}
		}
		if skip {
			for j := range m.D.Data {
				lpSparseRationalCheck(t, h, &m.D.Data[j], big.NewRat(0, 1))
			}
		} else {
			cb0, cb1 := current[f.basis[0]], current[f.basis[1]]
			y0 := new(big.Rat).Quo(new(big.Rat).Sub(new(big.Rat).Mul(big.NewRat(3, 1), cb0), cb1), big.NewRat(5, 1))
			y1 := new(big.Rat).Quo(new(big.Rat).Sub(new(big.Rat).Mul(big.NewRat(2, 1), cb1), cb0), big.NewRat(5, 1))
			lpSparseRationalCheck(t, h, &m.Y.Data[0], y0)
			lpSparseRationalCheck(t, h, &m.Y.Data[1], y1)
			for j := range current {
				want := new(big.Rat)
				if f.heading[j] < 0 {
					want.Set(current[j])
					want.Sub(want, new(big.Rat).Mul(y0, coeff[0][j]))
					want.Sub(want, new(big.Rat).Mul(y1, coeff[1][j]))
				}
				lpSparseRationalCheck(t, h, &m.D.Data[j], want)
			}
		}
		if skip || matching {
			if !reflect.DeepEqual(m.ColumnsNZ, oldColumns) || !reflect.DeepEqual(m.RowsNZ, oldRows) {
				t.Fatal("unexpected estimate refresh")
			}
		} else {
			wantCols := []uint32{0, 91, 0, 0}
			for j := range wantCols {
				if j != 1 {
					for i := range coeff {
						if coeff[i][j].Sign() != 0 {
							wantCols[j]++
						}
					}
				}
			}
			wantRows := []uint32{0, 99}
			for _, v := range coeff[0] {
				if v.Sign() != 0 {
					wantRows[0]++
				}
			}
			if !reflect.DeepEqual(m.ColumnsNZ, wantCols) || !reflect.DeepEqual(m.RowsNZ, wantRows) {
				t.Fatal("source zero-only estimate refresh")
			}
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPRationalPrimalInitRunPhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, op := range []string{"feasible-opt", "infeasible-opt", "infeasible-only"} {
		for _, direct := range []bool{false, true} {
			success, failures := false, 0
			for ordinal := int64(0); ordinal < 1000; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("%s/%t/%d", op, direct, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					f := lpNewRationalCoreFixture(t, h, 2, 4)
					f.settings.SimplexStrategy = StrategyLU
					huge := strings.Repeat("1234567890", 20)
					for i := uint32(0); i < 2; i++ {
						h.must(f.a.Set(i, i, &h.pair(fmt.Sprint(i+3), "0").X))
						h.must(f.a.Set(i, 2, &h.pair(huge+"7/13", "0").X))
						h.must(f.a.Set(i, 3, &h.pair(huge+"9/17", "0").X))
					}
					for j := range f.costs.Data {
						h.must(h.m.SetString(&f.costs.Data[j], huge+"3/23"))
					}
					m, e := NewRationalPrimalCoreSolver(f.input())
					h.must(e)
					if op != "feasible-opt" {
						for j := 0; j < 2; j++ {
							f.types[j] = ColumnLowBound
							h.must(f.low.copySlot(&f.low.Data[j], &h.pair(huge+"7/13", "0").X))
							h.must(f.x.copySlot(&f.x.Data[j], &h.pair(huge+"1/17", "0").X))
						}
					}
					m.LookForFeasibleSolutionOnly = op == "infeasible-only"
					m.UsingInfeasCosts = true
					m.TotalIterations = 77
					m.BasisSortCounter = 91
					m.IterationsNoCostGrowth = 83
					m.ColumnNormUpdateCounter = 88
					h.must(h.m.SetInt32(&m.EpsilonOfReducedCost, 9))
					h.budget.FailAfter(ordinal)
					err := m.InitRun()
					h.budget.FailAfter(-1)
					if err == nil {
						success, done = true, true
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						if m.calls.pending() && (m.RationalCoreSolverBase.calls.pending() || m.BreakpointQueue.calls.pending() || m.Factorization.stopped) {
							t.Fatal("adopted delegated numerical receipt")
						}
						if m.TotalIterations != 0 || m.BasisSortCounter != 0 || m.IterationsNoCostGrowth != 0 || m.ColumnNormUpdateCounter != 88 {
							t.Fatal("lost counter publication prefix")
						}
						m.TotalIterations = 101
						if e = m.InitRun(); e != ErrExactVectorPending || m.TotalIterations != 101 {
							t.Fatal("retry changed stopped run", e)
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
		}
	}
}

func TestLPRationalPrimalInitRunSourcePrefixes(t *testing.T) {
	for _, phase := range []string{"heading", "feasible-only", "tableau", "missing-lu", "same-estimates"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			f := lpNewRationalCoreFixture(t, h, 1, 2)
			h.must(f.a.Set(0, 0, &h.pair("3", "0").X))
			h.must(f.a.Set(0, 1, &h.pair("2", "0").X))
			h.must(h.m.SetInt32(&f.costs.Data[0], 7))
			h.must(h.m.SetInt32(&f.costs.Data[1], 11))
			if phase == "same-estimates" {
				f.settings.SimplexStrategy = StrategyLU
			}
			m, e := NewRationalPrimalCoreSolver(f.input())
			h.must(e)
			m.TotalIterations = 77
			m.BasisSortCounter = 91
			m.IterationsNoCostGrowth = 83
			m.ColumnNormUpdateCounter = 88
			m.UsingInfeasCosts = true
			h.must(h.m.SetInt32(&m.EpsilonOfReducedCost, 9))
			m.ColumnsNZ = []uint32{0, 91}
			m.RowsNZ = nil
			h.must(m.InfSet.Insert(1))
			want := error(nil)
			switch phase {
			case "heading":
				f.heading = nil
				want = ErrBounds
			case "feasible-only":
				m.LookForFeasibleSolutionOnly = true
			case "tableau":
				if sourceDebugEnabled {
					want = ErrAssertion
				} else {
					want = ErrBounds
				}
			case "missing-lu":
				m.Settings.SimplexStrategy = StrategyLU
				want = ErrBounds
			}
			err := m.InitRun()
			if err != want {
				t.Fatal(err, want)
			}
			if m.TotalIterations != 0 || m.BasisSortCounter != 0 || m.IterationsNoCostGrowth != 0 || m.ColumnNormUpdateCounter != 88 {
				t.Fatal("counter reset prefix")
			}
			if !m.InfSet.Empty() {
				t.Fatal("old membership not cleared")
			}
			early := phase == "heading" || phase == "feasible-only"
			if m.UsingInfeasCosts != early {
				t.Fatal("cost mode reset prefix")
			}
			epsilon := int64(0)
			if early {
				epsilon = 9
			}
			lpSparseRationalCheck(t, h, &m.EpsilonOfReducedCost, big.NewRat(epsilon, 1))
			if early {
				if len(m.CostsBackup.Data) != 0 || len(m.BreakpointQueue.inverse) != 0 {
					t.Fatal("modified preparation before early return")
				}
			} else {
				if len(m.CostsBackup.Data) != 2 || len(m.BreakpointQueue.inverse) != 2 {
					t.Fatal("lost backup and queue prefix")
				}
				lpSparseRationalCheck(t, h, &m.CostsBackup.Data[0], big.NewRat(7, 1))
				lpSparseRationalCheck(t, h, &m.CostsBackup.Data[1], big.NewRat(11, 1))
			}
			if !reflect.DeepEqual(m.ColumnsNZ, []uint32{0, 91}) || m.RowsNZ != nil {
				t.Fatal("count estimate changed despite same columns size or earlier failure")
			}
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPRationalPrimalInitRunAlreadyStoppedQueue(t *testing.T) {
	matched := false
	for ordinal := int64(0); ordinal < 80 && !matched; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			h := lpNewDirectSparseHarness(t)
			f := lpNewRationalCoreFixture(t, h, 1, 3)
			h.must(f.a.Set(0, 0, &h.pair("1", "0").X))
			m, e := NewRationalPrimalCoreSolver(f.input())
			h.must(e)
			h.must(m.BreakpointQueue.Resize(2))
			huge := strings.Repeat("1234567890", 12)
			h.must(m.BreakpointQueue.Enqueue(0, &h.pair(huge+"1/3", huge+"3/7").X))
			priority := h.pair(huge+"5/11", huge+"7/13")
			h.budget.FailAfter(ordinal)
			err := m.BreakpointQueue.Enqueue(1, &priority.X)
			h.budget.FailAfter(-1)
			if m.BreakpointQueue.calls.pending() {
				matched = true
				if !lpExactPriorityPhysicalRefusal(err) {
					t.Fatal(err)
				}
				old := h.m.ComparisonInvocationState()
				m.TotalIterations = 77
				m.UsingInfeasCosts = true
				h.must(h.m.SetInt32(&m.EpsilonOfReducedCost, 9))
				if e = m.InitRun(); e != ErrExactQueuePending {
					t.Fatal(e)
				}
				if m.TotalIterations != 0 || m.UsingInfeasCosts || len(m.CostsBackup.Data) != 3 || len(m.BreakpointQueue.inverse) != 2 {
					t.Fatal("wrong queue refusal prefix")
				}
				lpSparseRationalCheck(t, h, &m.EpsilonOfReducedCost, big.NewRat(0, 1))
				if m.calls.pending() || m.RationalCoreSolverBase.calls.pending() || !m.BreakpointQueue.calls.pending() {
					t.Fatal("lost queue receipt separation")
				}
				if old != h.m.ComparisonInvocationState() {
					t.Fatal("changed pending queue comparison")
				}
			}
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
	if !matched {
		t.Fatal("no actual stopped queue")
	}
}

func TestLPRationalPrimalInitRunLeavesOlderComparisonOwned(t *testing.T) {
	for _, op := range []string{"bounded", "free"} {
		matched := false
		for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", op, ordinal), func(t *testing.T) {
				h := newLPPairHarness(t)
				f := lpNewRationalCoreFixture(t, h, 1, 1)
				f.settings.SimplexStrategy = StrategyLU
				h.must(f.a.Set(0, 0, &h.pair("3", "0").X))
				m, err := NewRationalPrimalCoreSolver(f.input())
				h.must(err)
				m.UsingInfeasCosts = true
				v := h.pair(strings.Repeat("9876543210", 100)+"/7", strings.Repeat("1234567890", 100)+"/13")
				f.types[0] = ColumnLowBound
				h.must(h.m.Copy(&f.low.Data[0], &v.X))
				h.must(h.m.Copy(&f.x.Data[0], &v.Y))
				h.must(h.m.SetInt32(&f.costs.Data[0], 17))
				if op == "free" {
					f.types[0] = ColumnFree
				}
				m.TotalIterations = 77
				var caller exactEtaCalls
				before, e := caller.begin(h.m)
				h.must(e)
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
					err = m.InitRun()
					if m.calls.pending() || m.RationalCoreSolverBase.calls.pending() || m.work != nil {
						t.Fatal("adopted older comparison", err)
					}
					lpSparseRationalCheck(t, h, &f.costs.Data[0], big.NewRat(17, 1))
					if m.TotalIterations != 0 {
						t.Fatal("missing reset prefix")
					}
					if op == "bounded" && err == nil {
						t.Fatal("ignored pending bound comparison")
					}
					if err == nil {
						lpSparseRationalCheck(t, h, &m.Y.Data[0], big.NewRat(17, 3))
					}
				} else if err != nil && !lpExactPriorityPhysicalRefusal(err) {
					t.Fatal(err)
				}
				h.must(m.Close())
				if older.Active && h.m.ComparisonInvocationState() != older {
					t.Fatal("consumed caller comparison")
				}
				h.must(caller.cleanupRaw(h.m))
				h.must(caller.cleanupRoots(h.m))
				f.close()
				lpExactVectorNamedCensus(t, h, nil)
			})
		}
		if !matched {
			t.Fatal("no actual caller comparison", op)
		}
	}
}
