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

func TestLPExactAdvanceDifferentLiveSteps(t *testing.T) {
	for _, phase := range []string{"basic", "negative-basic", "entering"} {
		h := newLPPairHarness(t)
		f, m := lpAdvanceDifferentFixture(t, h)
		h.must(f.x.Data[0].Copy(h.m, h.pair("1", "0")))
		step := &f.x.Data[0]
		rhs := "2"
		wx, wy := big.NewRat(11, 14), big.NewRat(1, 1)
		if phase == "negative-basic" {
			m.SignOfEnteringDelta = -1
			wx.Neg(wx)
			wy.Neg(wy)
		}
		if phase == "entering" {
			h.must(f.x.Data[1].Copy(h.m, h.pair("1", "0")))
			step = &f.x.Data[1]
			rhs = "17/7"
			wx, wy = big.NewRat(4, 7), big.NewRat(2, 1)
		}
		h.must(f.b.Data[0].Copy(h.m, h.pair(rhs, "0")))
		m.LookForFeasibleSolutionOnly = true
		h.must(m.AdvanceOnEnteringAndLeaving(1, 0, step))
		h.check(&f.x.Data[0], wx, new(big.Rat))
		h.check(&f.x.Data[1], wy, new(big.Rat))
		if m.Status != StatusFeasible || f.basis[0] != 1 || m.NonBasisList[1] != 1 || m.IterationsNoCostGrowth != 0 {
			t.Fatal("alias publication")
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPExactAdvanceDifferentOlderComparison(t *testing.T) {
	matched := false
	for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
		h := newLPPairHarness(t)
		f, m := lpAdvanceDifferentFixture(t, h)
		v := h.pair(strings.Repeat("9876543210", 100)+"/7", strings.Repeat("1234567890", 100)+"/13")
		step := h.pair("1", "0")
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
			if m.calls.comparison != 0 || m.PairCoreSolverBase.calls.comparison != 0 {
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

func TestLPExactAdvanceDifferentIndependentSystems(t *testing.T) {
	for world := 0; world < 160; world++ {
		h := newLPPairHarness(t)
		f := lpNewPairCoreFixture(t, h, 2, 4)
		f.settings.SimplexStrategy = StrategyLU
		a := [][]*big.Rat{{big.NewRat(2, 1), big.NewRat(1, 1), big.NewRat(3, 7), big.NewRat(-1, 3)}, {big.NewRat(1, 1), big.NewRat(3, 1), big.NewRat(-2, 11), big.NewRat(5, 13)}}
		x := []*big.Rat{big.NewRat(int64(world%9), 3), big.NewRat(int64(world%7), 5), new(big.Rat), big.NewRat(2, 7)}
		costs := make([]*big.Rat, 4)
		for j := range x {
			h.must(f.x.Data[j].Copy(h.m, h.pair(x[j].RatString(), "0")))
			costs[j] = big.NewRat(int64(world%23+j-11), int64(7+j))
			h.must(h.m.SetString(&f.costs.Data[j], costs[j].RatString()))
		}
		for i := range a {
			rhs := new(big.Rat)
			for j, v := range a[i] {
				h.must(f.a.Set(uint32(i), uint32(j), &h.pair(v.RatString(), "0").X))
				rhs.Add(rhs, new(big.Rat).Mul(v, x[j]))
			}
			h.must(f.b.Data[i].Copy(h.m, h.pair(rhs.RatString(), "0")))
		}
		m, e := NewPairPrimalCoreSolver(f.input())
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
		step, eps := big.NewRat(int64(world%5), 7), big.NewRat(int64(world%3), 11)
		if world%7 == 0 {
			step.SetInt64(0)
			eps.SetInt64(0)
		}
		tstep := h.pair(step.RatString(), eps.RatString())
		sign := big.NewRat(int64(m.SignOfEnteringDelta), 1)
		step.Mul(step, sign)
		eps.Mul(eps, sign)
		// B inverse independently equals [[3,-1],[-1,2]]/5.
		d := []*big.Rat{
			new(big.Rat).Quo(new(big.Rat).Sub(new(big.Rat).Mul(big.NewRat(3, 1), a[0][2]), a[1][2]), big.NewRat(5, 1)),
			new(big.Rat).Quo(new(big.Rat).Sub(new(big.Rat).Mul(big.NewRat(2, 1), a[1][2]), a[0][2]), big.NewRat(5, 1)),
		}
		leave := uint32(world % 2)
		h.must(m.AdvanceOnEnteringAndLeaving(2, int32(leave), tstep))
		for j := range x {
			wx, wy := new(big.Rat).Set(x[j]), new(big.Rat)
			if j < 2 {
				wx.Sub(wx, new(big.Rat).Mul(d[j], step))
				wy.Neg(new(big.Rat).Mul(d[j], eps))
			}
			if j == 2 {
				wx.Add(wx, step)
				wy.Set(eps)
			}
			h.check(&f.x.Data[j], wx, wy)
		}
		h.check(tstep, step, eps)
		if f.basis[leave] != 2 || m.Status != StatusFeasible {
			t.Fatal("basis/status", world, f.basis, m.Status)
		}
		wantCounter := uint32(0)
		if step.Sign() == 0 && eps.Sign() == 0 {
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

func lpAdvanceDifferentFixture(t *testing.T, h *lpPairHarness) (*lpPairCoreFixture, *PairPrimalCoreSolver) {
	f := lpNewPairCoreFixture(t, h, 1, 3)
	f.settings.SimplexStrategy = StrategyLU
	h.must(f.a.Set(0, 0, &h.pair("2", "0").X))
	h.must(f.a.Set(0, 1, &h.pair("3/7", "0").X))
	h.must(f.a.Set(0, 2, &h.pair("5/11", "0").X))
	m, e := NewPairPrimalCoreSolver(f.input())
	h.must(e)
	m.ColumnsNZ = make([]uint32, 3)
	m.RowsNZ = make([]uint32, 1)
	h.must(m.SolveBd(1))
	m.NonBasisList = []uint32{2, 1}
	m.SignOfEnteringDelta = 1
	m.IterationsNoCostGrowth = 29
	return f, m
}

func TestLPExactAdvanceDifferentPivotOutcomes(t *testing.T) {
	for _, phase := range []string{"opposite", "zero", "drift", "singular-refactor"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			f, m := lpAdvanceDifferentFixture(t, h)
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
			h.must(m.AdvanceOnEnteringAndLeaving(1, 0, h.pair("0", "0")))
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

func TestLPExactAdvanceDifferentPhysicalOwnership(t *testing.T) {
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
					f, m := lpAdvanceDifferentFixture(t, h)
					h.must(h.m.SetString(&m.D.Data[1], strings.Repeat("1234567890", 12)+"1/13"))
					step := h.pair(strings.Repeat("9876543210", 12)+"1/17", strings.Repeat("3141592653", 12)+"1/19")
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
						step = h.pair("0", "0")
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

func TestLPExactAdvanceDifferentRestoresCosts(t *testing.T) {
	for _, breakpoints := range []bool{false, true} {
		for _, only := range []bool{false, true} {
			h := newLPPairHarness(t)
			f, m := lpAdvanceDifferentFixture(t, h)
			f.types[0] = ColumnLowBound
			h.must(f.x.Data[0].Copy(h.m, h.pair("-1", "0")))
			h.must(f.b.Data[0].Copy(h.m, h.pair("-2", "0")))
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
			step := h.pair("14/3", "0")
			if breakpoints {
				h.must(step.Negate(h.m))
			}
			h.must(m.AdvanceOnEnteringAndLeaving(1, 0, step))
			h.check(&f.x.Data[0], new(big.Rat), new(big.Rat))
			h.check(&f.x.Data[1], big.NewRat(-14, 3), new(big.Rat))
			h.check(step, big.NewRat(-14, 3), new(big.Rat))
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

func TestLPExactAdvanceDifferentRollback(t *testing.T) {
	for _, only := range []bool{false, true} {
		for _, lostBackup := range []bool{false, true} {
			h := newLPPairHarness(t)
			f := lpNewPairCoreFixture(t, h, 2, 3)
			f.settings.SimplexStrategy = StrategyLU
			for _, cell := range [][2]uint32{{0, 0}, {1, 1}, {0, 2}, {1, 2}} {
				h.must(f.a.Set(cell[0], cell[1], &h.pair("1", "0").X))
			}
			if lostBackup {
				h.must(f.x.Data[0].Copy(h.m, h.pair("1", "0")))
				h.must(f.b.Data[0].Copy(h.m, h.pair("1", "0")))
			}
			m, e := NewPairPrimalCoreSolver(f.input())
			h.must(e)
			m.ColumnsNZ = make([]uint32, 3)
			m.RowsNZ = make([]uint32, 2)
			h.must(m.SolveBd(2))
			// Mutate the remaining column after the entering solve: the refreshed
			// current basis is nonsingular, while the proposed new basis is singular.
			// Its old inverse row still agrees with Ed, so the outer comparator takes0.
			h.must(f.a.Set(0, 1, &h.pair("1", "0").X))
			m.Factorization.RefactorCounter = 200
			m.NonBasisList = []uint32{2}
			m.SignOfEnteringDelta = 1
			m.IterationsNoCostGrowth = 29
			m.LookForFeasibleSolutionOnly = only
			step := h.pair("0", "0")
			if lostBackup {
				step = h.pair("1", "0")
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

func TestLPExactAdvanceDifferentSourcePrefixes(t *testing.T) {
	for _, phase := range []string{"same", "same-zero", "negative-entering", "empty-list", "wrong-tail", "negative-step", "negative-leaving", "nil-step", "late-heading", "late-list", "tableau", "nil-negative", "late-cost-type", "missing-row-counts", "missing-factor"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			f, m := lpAdvanceDifferentFixture(t, h)
			en, lv := int32(1), int32(0)
			step := h.pair("1", "0")
			want := ErrBounds
			switch phase {
			case "same":
				lv = 1
				m.LookForFeasibleSolutionOnly = true
				want = nil
			case "same-zero":
				lv = 1
				step = h.pair("0", "0")
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
				step = h.pair("-1", "0")
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
				h.check(&f.x.Data[0], big.NewRat(-3, 14), new(big.Rat))
				h.check(&f.x.Data[1], big.NewRat(1, 1), new(big.Rat))
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

func TestLPExactAdvanceDifferentCostSwitchPhysicalOwnership(t *testing.T) {
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
				f, m := lpAdvanceDifferentFixture(t, h)
				f.types[0] = ColumnLowBound
				h.must(f.x.Data[0].Copy(h.m, h.pair("-1", "0")))
				h.must(f.b.Data[0].Copy(h.m, h.pair("-2", "0")))
				for j := range f.costs.Data {
					h.must(h.m.SetString(&f.costs.Data[j], strings.Repeat("1234567890", 12)+fmt.Sprintf("%d/13", j+1)))
				}
				h.must(m.BackupAndNormalizeCosts())
				h.must(m.InitInfSet())
				h.must(m.InitInfeasibilityCosts())
				h.must(m.InitReducedCostsForOneIteration())
				m.Settings.UseBreakpointsInFeasibilitySearch = false
				m.SignOfEnteringDelta = -1
				step := h.pair("14/3", "0")
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

func TestLPExactAdvanceDifferentMinimumNegation(t *testing.T) {
	for _, direct := range []bool{false, true} {
		success, negationFailure := false, false
		for ordinal := int64(0); ordinal < 600; ordinal++ {
			var h *lpPairHarness
			if direct {
				h = lpNewDirectSparseHarness(t)
			} else {
				h = newLPPairHarness(t)
			}
			f, m := lpAdvanceDifferentFixture(t, h)
			m.UsingInfeasCosts = true
			m.LookForFeasibleSolutionOnly = true
			m.SignOfEnteringDelta = -1
			step := h.pair("0", "0")
			h.must(h.m.SetInt32(&step.X, -2147483648))
			h.budget.FailAfter(ordinal)
			err := m.AdvanceOnEnteringAndLeaving(1, 0, step)
			h.budget.FailAfter(-1)
			if err == nil {
				success = true
				h.check(step, big.NewRat(2147483648, 1), new(big.Rat))
			} else {
				if !lpExactPriorityPhysicalRefusal(err) {
					t.Fatal(err)
				}
				if m.work != nil && m.pairWork == nil {
					negationFailure = true
					h.check(step, big.NewRat(-2147483648, 1), new(big.Rat))
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
		if !success || (direct && !negationFailure) {
			t.Fatal("minimum negation campaign", direct, negationFailure)
		}
	}
}
