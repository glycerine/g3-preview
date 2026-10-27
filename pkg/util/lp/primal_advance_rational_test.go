package lp

import (
	"errors"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestLPRationalAdvanceDifferentLiveSteps(t *testing.T) {
	for _, phase := range []string{"basic", "negative-basic", "entering"} {
		h := newLPPairHarness(t)
		f, m := lpRationalAdvanceDifferentFixture(t, h)
		h.must(h.m.Copy(&f.x.Data[0], lpScalarBoundValue(h, "1")))
		step := &f.x.Data[0]
		rhs := "2"
		wx, wy := big.NewRat(11, 14), big.NewRat(1, 1)
		if phase == "negative-basic" {
			m.SignOfEnteringDelta = -1
			wx.Neg(wx)
			wy.Neg(wy)
		}
		if phase == "entering" {
			h.must(h.m.Copy(&f.x.Data[1], lpScalarBoundValue(h, "1")))
			step = &f.x.Data[1]
			rhs = "17/7"
			wx, wy = big.NewRat(4, 7), big.NewRat(2, 1)
		}
		h.must(h.m.Copy(&f.b.Data[0], lpScalarBoundValue(h, rhs)))
		m.LookForFeasibleSolutionOnly = true
		h.must(m.AdvanceOnEnteringAndLeaving(1, 0, step))
		lpSparseRationalCheck(t, h, &f.x.Data[0], wx)
		lpSparseRationalCheck(t, h, &f.x.Data[1], wy)
		if m.Status != StatusFeasible || f.basis[0] != 1 || m.NonBasisList[1] != 1 || m.IterationsNoCostGrowth != 0 {
			t.Fatal("alias publication")
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPRationalAdvanceDifferentOlderComparison(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	matched := false
	for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
		h := newLPPairHarness(t)
		f, m := lpRationalAdvanceDifferentFixture(t, h)
		v := h.pair(strings.Repeat("9876543210", 100)+"/7", strings.Repeat("1234567890", 100)+"/13")
		step := lpScalarBoundValue(h, "1")
		var caller exactEtaCalls
		before, e := caller.begin(h.m)
		h.must(e)
		h.budget.FailAfter(ordinal)
		_, err := h.m.Compare(&v.X, &v.Y)
		h.budget.FailAfter(-1)
		err = caller.after(h.m, before, err)
		older := h.m.ComparisonInvocationState()
		if older.Active {
			matched = true
			if !lpExactPriorityPhysicalRefusal(err) {
				t.Fatal(err)
			}
			err = m.AdvanceOnEnteringAndLeaving(1, 0, step)
			if err != nil && !lpExactPriorityPhysicalRefusal(err) && !errors.Is(err, arena.ErrInvalidSlot) && !errors.Is(err, numeral.ErrRationalRawArithmeticPending) {
				t.Fatal(err)
			}
			if m.calls.comparison != 0 || m.RationalCoreSolverBase.calls.comparison != 0 {
				t.Fatal("adopted caller comparison")
			}
		} else if err != nil && !lpExactPriorityPhysicalRefusal(err) {
			t.Fatal(err)
		}
		h.must(m.Close())
		f.close()
		if older.Active && h.m.ComparisonInvocationState() != older {
			t.Fatal("consumed caller comparison")
		}
		h.must(caller.cleanupRaw(h.m))
		h.must(caller.cleanupRoots(h.m))
		lpExactVectorNamedCensus(t, h, nil)
	}
	if !matched {
		t.Fatal("no actual caller comparison")
	}
}

func TestLPRationalAdvanceDifferentIndependentSystems(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	for world := 0; world < 160; world++ {
		h := newLPPairHarness(t)
		f := lpNewRationalCoreFixture(t, h, 2, 4)
		f.settings.SimplexStrategy = StrategyLU
		a := [][]*big.Rat{{big.NewRat(2, 1), big.NewRat(1, 1), big.NewRat(3, 7), big.NewRat(-1, 3)}, {big.NewRat(1, 1), big.NewRat(3, 1), big.NewRat(-2, 11), big.NewRat(5, 13)}}
		x := []*big.Rat{big.NewRat(int64(world%9), 3), big.NewRat(int64(world%7), 5), new(big.Rat), big.NewRat(2, 7)}
		costs := make([]*big.Rat, 4)
		for j := range x {
			h.must(h.m.Copy(&f.x.Data[j], lpScalarBoundValue(h, x[j].RatString())))
			costs[j] = big.NewRat(int64(world%23+j-11), int64(7+j))
			h.must(h.m.SetString(&f.costs.Data[j], costs[j].RatString()))
		}
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
		m.ColumnsNZ = make([]uint32, 4)
		m.RowsNZ = make([]uint32, 2)
		h.must(m.InitReducedCostsForOneIteration())
		h.must(m.SolveBd(2))
		m.NonBasisList = []uint32{3, 2}
		m.SignOfEnteringDelta = 1
		if world%2 == 0 {
			m.SignOfEnteringDelta = -1
		}
		m.LookForFeasibleSolutionOnly = world/2%2 == 0
		m.IterationsNoCostGrowth = 41
		if world/4%2 == 0 {
			m.Factorization.RefactorCounter = 200
		}
		step := big.NewRat(int64(world%5), 7)
		if world%7 == 0 {
			step.SetInt64(0)
		}
		tstep := lpScalarBoundValue(h, step.RatString())
		sign := big.NewRat(int64(m.SignOfEnteringDelta), 1)
		step.Mul(step, sign)
		// B inverse independently equals [[3,-1],[-1,2]]/5.
		d := []*big.Rat{
			new(big.Rat).Quo(new(big.Rat).Sub(new(big.Rat).Mul(big.NewRat(3, 1), a[0][2]), a[1][2]), big.NewRat(5, 1)),
			new(big.Rat).Quo(new(big.Rat).Sub(new(big.Rat).Mul(big.NewRat(2, 1), a[1][2]), a[0][2]), big.NewRat(5, 1)),
		}
		leave := uint32(world % 2)
		h.must(m.AdvanceOnEnteringAndLeaving(2, int32(leave), tstep))
		for j := range x {
			wx := new(big.Rat).Set(x[j])
			if j < 2 {
				wx.Sub(wx, new(big.Rat).Mul(d[j], step))
			}
			if j == 2 {
				wx.Add(wx, step)
			}
			lpSparseRationalCheck(t, h, &f.x.Data[j], wx)
		}
		lpSparseRationalCheck(t, h, tstep, step)
		if f.basis[leave] != 2 || m.Status != StatusFeasible {
			t.Fatal("basis/status", world, f.basis, m.Status)
		}
		wantCounter := uint32(0)
		if step.Sign() == 0 {
			wantCounter = 41
		}
		if m.IterationsNoCostGrowth != wantCounter {
			t.Fatal("counter", world, m.IterationsNoCostGrowth)
		}
		if m.LookForFeasibleSolutionOnly {
			if m.NonBasisList[1] != 2 {
				t.Fatal("early pricing list publication")
			}
		} else {
			if m.NonBasisList[1] != leave {
				t.Fatal("missing list publication")
			}
			b0, b1 := f.basis[0], f.basis[1]
			det := new(big.Rat).Sub(new(big.Rat).Mul(a[0][b0], a[1][b1]), new(big.Rat).Mul(a[0][b1], a[1][b0]))
			y0 := new(big.Rat).Quo(new(big.Rat).Sub(new(big.Rat).Mul(costs[b0], a[1][b1]), new(big.Rat).Mul(costs[b1], a[1][b0])), det)
			y1 := new(big.Rat).Quo(new(big.Rat).Sub(new(big.Rat).Mul(costs[b1], a[0][b0]), new(big.Rat).Mul(costs[b0], a[0][b1])), det)
			for j := range costs {
				want := new(big.Rat).Sub(costs[j], new(big.Rat).Mul(y0, a[0][j]))
				want.Sub(want, new(big.Rat).Mul(y1, a[1][j]))
				lpSparseRationalCheck(t, h, &m.D.Data[j], want)
			}
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func lpRationalAdvanceDifferentFixture(t *testing.T, h *lpPairHarness) (*lpRationalCoreFixture, *RationalPrimalCoreSolver) {
	f := lpNewRationalCoreFixture(t, h, 1, 3)
	f.settings.SimplexStrategy = StrategyLU
	h.must(f.a.Set(0, 0, lpScalarBoundValue(h, "2")))
	h.must(f.a.Set(0, 1, lpScalarBoundValue(h, "3/7")))
	h.must(f.a.Set(0, 2, lpScalarBoundValue(h, "5/11")))
	m, e := NewRationalPrimalCoreSolver(f.input())
	h.must(e)
	m.ColumnsNZ = make([]uint32, 3)
	m.RowsNZ = make([]uint32, 1)
	h.must(m.SolveBd(1))
	m.NonBasisList = []uint32{2, 1}
	m.SignOfEnteringDelta = 1
	m.IterationsNoCostGrowth = 29
	return f, m
}

func TestLPRationalAdvanceDifferentPivotOutcomes(t *testing.T) {
	for _, phase := range []string{"opposite", "zero", "drift", "singular-refactor"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			f, m := lpRationalAdvanceDifferentFixture(t, h)
			old := m.Factorization
			switch phase {
			case "opposite":
				h.must(h.m.SetInt32(&m.Ed.Data[0], -1))
			case "zero", "singular-refactor":
				h.must(h.m.SetInt32(&m.Ed.Data[0], 0))
			case "drift":
				h.must(h.m.SetInt32(&m.Ed.Data[0], 1))
			}
			if phase == "singular-refactor" {
				v, e := f.a.ColumnValue(f.a.Columns[0][0])
				h.must(e)
				h.must(h.m.SetInt32(v, 0))
			}
			h.must(m.AdvanceOnEnteringAndLeaving(1, 0, lpScalarBoundValue(h, "0")))
			if phase == "opposite" || phase == "singular-refactor" {
				if m.Status != StatusUnstable || m.IterationsNoCostGrowth != 30 || f.basis[0] != 0 || m.NonBasisList[1] != 1 {
					t.Fatal("unstable prefix")
				}
			} else if m.Status != StatusFeasible || f.basis[0] != 1 || m.NonBasisList[1] != 0 || m.Factorization == old {
				t.Fatal("refactor outcome")
			}
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPRationalAdvanceDifferentPhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, direct := range []bool{false, true} {
		for _, direction := range []int32{1, -1, 0} {
			success, failures := false, 0
			for ordinal := int64(0); ordinal < 1200; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("%t/%d/%d", direct, direction, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					f, m := lpRationalAdvanceDifferentFixture(t, h)
					h.must(h.m.SetString(&m.D.Data[1], strings.Repeat("1234567890", 12)+"1/13"))
					step := lpScalarBoundValue(h, strings.Repeat("9876543210", 12)+"1/17")
					if direction == -1 {
						m.SignOfEnteringDelta = -1
					}
					if direction == 0 {
						p, e := f.a.ColumnValue(f.a.Columns[0][0])
						h.must(e)
						h.must(h.m.SetString(p, strings.Repeat("2718281828", 12)+"1/23"))
						h.must(m.InitLU())
						h.must(m.SolveBd(1))
						h.must(h.m.SetInt32(&m.Ed.Data[0], 1))
						step = lpScalarBoundValue(h, "0")
					}
					h.budget.FailAfter(ordinal)
					err := m.AdvanceOnEnteringAndLeaving(1, 0, step)
					h.budget.FailAfter(-1)
					if err == nil {
						success, done = true, true
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						if e := m.AdvanceOnEnteringAndLeaving(1, 0, step); e != ErrExactVectorPending {
							t.Fatal("retry", e)
						}
					}
					h.must(m.Close())
					f.close()
					lpExactVectorNamedCensus(t, h, nil)
				})
				if t.Failed() {
					return
				}
				if done {
					break
				}
			}
			if !success || (direct && failures == 0) {
				t.Fatal("campaign incomplete", direct, direction, failures)
			}
		}
	}
}

func TestLPRationalAdvanceDifferentRestoresCosts(t *testing.T) {
	for _, breakpoints := range []bool{false, true} {
		for _, only := range []bool{false, true} {
			h := newLPPairHarness(t)
			f, m := lpRationalAdvanceDifferentFixture(t, h)
			f.types[0] = ColumnLowBound
			h.must(h.m.Copy(&f.x.Data[0], lpScalarBoundValue(h, "-1")))
			h.must(h.m.Copy(&f.b.Data[0], lpScalarBoundValue(h, "-2")))
			for j, v := range []int32{7, 11, 13} {
				h.must(h.m.SetInt32(&f.costs.Data[j], v))
			}
			h.must(m.BackupAndNormalizeCosts())
			h.must(m.InitInfSet())
			h.must(m.InitInfeasibilityCosts())
			h.must(m.InitReducedCostsForOneIteration())
			m.Settings.UseBreakpointsInFeasibilitySearch = breakpoints
			m.LookForFeasibleSolutionOnly = only
			m.SignOfEnteringDelta = -1
			step := lpScalarBoundValue(h, "14/3")
			if breakpoints {
				h.must(h.m.Neg(step))
			}
			h.must(m.AdvanceOnEnteringAndLeaving(1, 0, step))
			lpSparseRationalCheck(t, h, &f.x.Data[0], new(big.Rat))
			lpSparseRationalCheck(t, h, &f.x.Data[1], big.NewRat(-14, 3))
			lpSparseRationalCheck(t, h, step, big.NewRat(-14, 3))
			if m.CurrentXIsInfeasible() || m.Status != StatusFeasible || m.IterationsNoCostGrowth != 0 {
				t.Fatal("feasibility transition")
			}
			if only {
				if !m.UsingInfeasCosts || m.NonBasisList[1] != 1 {
					t.Fatal("feasibility-only early return")
				}
			} else {
				if m.UsingInfeasCosts || m.NonBasisList[1] != 0 {
					t.Fatal("cost switch/list")
				}
				for j, v := range []int64{7, 11, 13} {
					lpSparseRationalCheck(t, h, &f.costs.Data[j], big.NewRat(v, 1))
				}
				for j, v := range []*big.Rat{big.NewRat(-133, 3), new(big.Rat), big.NewRat(4, 3)} {
					lpSparseRationalCheck(t, h, &m.D.Data[j], v)
				}
			}
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		}
	}
}

func TestLPRationalAdvanceDifferentRollback(t *testing.T) {
	for _, only := range []bool{false, true} {
		for _, lostBackup := range []bool{false, true} {
			h := newLPPairHarness(t)
			f := lpNewRationalCoreFixture(t, h, 2, 3)
			f.settings.SimplexStrategy = StrategyLU
			for _, cell := range [][2]uint32{{0, 0}, {1, 1}, {0, 2}, {1, 2}} {
				h.must(f.a.Set(cell[0], cell[1], lpScalarBoundValue(h, "1")))
			}
			if lostBackup {
				h.must(h.m.Copy(&f.x.Data[0], lpScalarBoundValue(h, "1")))
				h.must(h.m.Copy(&f.b.Data[0], lpScalarBoundValue(h, "1")))
			}
			m, e := NewRationalPrimalCoreSolver(f.input())
			h.must(e)
			m.ColumnsNZ = make([]uint32, 3)
			m.RowsNZ = make([]uint32, 2)
			h.must(m.SolveBd(2))
			// Mutate the remaining column after the entering solve: the refreshed
			// current basis is nonsingular, while the proposed new basis is singular.
			// Its old inverse row still agrees with Ed, so the outer comparator takes0.
			h.must(f.a.Set(0, 1, lpScalarBoundValue(h, "1")))
			m.Factorization.RefactorCounter = 200
			m.NonBasisList = []uint32{2}
			m.SignOfEnteringDelta = 1
			m.IterationsNoCostGrowth = 29
			m.LookForFeasibleSolutionOnly = only
			step := lpScalarBoundValue(h, "0")
			if lostBackup {
				step = lpScalarBoundValue(h, "1")
			}
			h.must(m.AdvanceOnEnteringAndLeaving(2, 0, step))
			want := StatusUnstable
			if only || lostBackup {
				want = StatusFloatingPointError
			}
			wantCounter := uint32(30)
			if lostBackup {
				wantCounter = 29
			}
			if m.Status != want || m.IterationsNoCostGrowth != wantCounter || f.basis[0] != 0 || m.NonBasisList[0] != 2 {
				t.Fatal("rollback", only, m.Status, m.IterationsNoCostGrowth, f.basis, m.NonBasisList)
			}
			if m.Factorization.GetStatus() != LUOK {
				t.Fatal("old basis not restored")
			}
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		}
	}
}

func TestLPRationalAdvanceDifferentSourcePrefixes(t *testing.T) {
	for _, phase := range []string{"same", "same-zero", "negative-entering", "empty-list", "wrong-tail", "negative-step", "negative-leaving", "nil-step", "late-heading", "late-list", "tableau", "nil-negative", "late-cost-type", "missing-row-counts", "missing-factor"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			f, m := lpRationalAdvanceDifferentFixture(t, h)
			en, lv := int32(1), int32(0)
			step := lpScalarBoundValue(h, "1")
			want := ErrBounds
			switch phase {
			case "same":
				lv = 1
				m.LookForFeasibleSolutionOnly = true
				want = nil
			case "same-zero":
				lv = 1
				step = lpScalarBoundValue(h, "0")
				m.LookForFeasibleSolutionOnly = true
				want = nil
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "negative-entering":
				en = -1
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "empty-list":
				m.NonBasisList = nil
			case "wrong-tail":
				if !sourceDebugEnabled {
					h.must(m.Close())
					f.close()
					return
				}
				m.NonBasisList[1] = 2
				want = ErrAssertion
			case "negative-step":
				if !sourceDebugEnabled {
					h.must(m.Close())
					f.close()
					return
				}
				step = lpScalarBoundValue(h, "-1")
				want = ErrAssertion
			case "negative-leaving":
				lv = -1
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "nil-step":
				step = nil
			case "nil-negative":
				step = nil
				m.SignOfEnteringDelta = -1
				m.UsingInfeasCosts = true
			case "late-cost-type":
				m.UsingInfeasCosts = true
				f.types = f.types[:1]
			case "missing-row-counts":
				m.RowsNZ = nil
			case "missing-factor":
				h.must(m.Factorization.Close())
				m.Factorization = nil
			case "late-heading":
				lv = 9
			case "late-list":
				if sourceDebugEnabled {
					h.must(m.Close())
					f.close()
					return
				}
				m.NonBasisList = nil
			case "tableau":
				if !sourceDebugEnabled {
					h.must(m.Close())
					f.close()
					return
				}
				m.Settings.SimplexStrategy = StrategyTableauRows
				want = ErrAssertion
			}
			if e := m.AdvanceOnEnteringAndLeaving(en, lv, step); e != want {
				t.Fatal(phase, e, want)
			}
			if phase == "late-cost-type" {
				if f.basis[0] != 1 || m.IterationsNoCostGrowth != 0 || m.Status != StatusUnknown {
					t.Fatal("cost type prefix")
				}
			}
			if phase == "same" {
				lpSparseRationalCheck(t, h, &f.x.Data[0], big.NewRat(-3, 14))
				lpSparseRationalCheck(t, h, &f.x.Data[1], big.NewRat(1, 1))
				if f.basis[0] != 0 || m.IterationsNoCostGrowth != 29 {
					t.Fatal("same-column dispatch")
				}
			}
			if phase == "late-list" || (!sourceDebugEnabled && phase == "empty-list") {
				if f.basis[0] != 1 || m.Status != StatusFeasible || m.IterationsNoCostGrowth != 0 {
					t.Fatal("late publication prefix")
				}
			}
			if want != nil {
				if e := m.AdvanceOnEnteringAndLeaving(en, lv, step); e != ErrExactVectorPending {
					t.Fatal("stopped retry", e)
				}
			}
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPRationalAdvanceDifferentCostSwitchPhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	for _, direct := range []bool{false, true} {
		success, restoringFailure := false, false
		for ordinal := int64(0); ordinal < 1200; ordinal++ {
			done := false
			t.Run(fmt.Sprintf("%t/%d", direct, ordinal), func(t *testing.T) {
				var h *lpPairHarness
				if direct {
					h = lpNewDirectSparseHarness(t)
				} else {
					h = newLPPairHarness(t)
				}
				f, m := lpRationalAdvanceDifferentFixture(t, h)
				f.types[0] = ColumnLowBound
				h.must(h.m.Copy(&f.x.Data[0], lpScalarBoundValue(h, "-1")))
				h.must(h.m.Copy(&f.b.Data[0], lpScalarBoundValue(h, "-2")))
				for j := range f.costs.Data {
					h.must(h.m.SetString(&f.costs.Data[j], strings.Repeat("1234567890", 12)+fmt.Sprintf("%d/13", j+1)))
				}
				h.must(m.BackupAndNormalizeCosts())
				h.must(m.InitInfSet())
				h.must(m.InitInfeasibilityCosts())
				h.must(m.InitReducedCostsForOneIteration())
				m.Settings.UseBreakpointsInFeasibilitySearch = false
				m.SignOfEnteringDelta = -1
				step := lpScalarBoundValue(h, "14/3")
				h.budget.FailAfter(ordinal)
				err := m.AdvanceOnEnteringAndLeaving(1, 0, step)
				h.budget.FailAfter(-1)
				if err == nil {
					success, done = true, true
				} else {
					if !lpExactPriorityPhysicalRefusal(err) {
						t.Fatal(err)
					}
					if m.Status == StatusFeasible && m.UsingInfeasCosts {
						restoringFailure = true
					}
					if e := m.AdvanceOnEnteringAndLeaving(1, 0, step); e != ErrExactVectorPending {
						t.Fatal("retry", e)
					}
				}
				h.must(m.Close())
				f.close()
				lpExactVectorNamedCensus(t, h, nil)
			})
			if t.Failed() {
				return
			}
			if done {
				break
			}
		}
		if !success || (direct && !restoringFailure) {
			t.Fatal("cost-switch campaign incomplete", direct, restoringFailure)
		}
	}
}

func TestLPRationalAdvanceDifferentMinimumNegation(t *testing.T) {
	for _, direct := range []bool{false, true} {
		success, negationFailure, assignmentFailure := false, false, false
		for ordinal := int64(0); ordinal < 600; ordinal++ {
			var h *lpPairHarness
			if direct {
				h = lpNewDirectSparseHarness(t)
			} else {
				h = newLPPairHarness(t)
			}
			f, m := lpRationalAdvanceDifferentFixture(t, h)
			m.UsingInfeasCosts = true
			m.LookForFeasibleSolutionOnly = true
			m.SignOfEnteringDelta = -1
			step := lpScalarBoundValue(h, "0")
			h.must(h.m.SetInt32(step, -2147483648))
			h.budget.FailAfter(ordinal)
			err := m.AdvanceOnEnteringAndLeaving(1, 0, step)
			h.budget.FailAfter(-1)
			if err == nil {
				success = true
				lpSparseRationalCheck(t, h, step, big.NewRat(2147483648, 1))
			} else {
				if !lpExactPriorityPhysicalRefusal(err) {
					t.Fatal(err)
				}
				if m.work != nil {
					value, inspectErr := h.m.String(step)
					if inspectErr == nil {
						if value != "-2147483648" {
							t.Fatal("unexpected pre-assignment prefix", value)
						}
						negationFailure = true
					} else {
						if !errors.Is(inspectErr, arena.ErrInvalidSlot) {
							t.Fatal(inspectErr)
						}
						lpSparseRationalCheck(t, h, &m.work.Data[0], big.NewRat(2147483648, 1))
						assignmentFailure = true
					}
				}
				if e := m.AdvanceOnEnteringAndLeaving(1, 0, step); e != ErrExactVectorPending {
					t.Fatal("retry", e)
				}
			}
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
			if success {
				break
			}
		}
		if !success || (direct && (!negationFailure || !assignmentFailure)) {
			t.Fatal("minimum negation campaign", direct, negationFailure, assignmentFailure)
		}
	}
}
