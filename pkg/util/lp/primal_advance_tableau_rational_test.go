package lp

import (
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestLPRationalTableauAdvanceEqualIndependentModels(t *testing.T) {
	for world := 0; world < 320; world++ {
		h := newLPPairHarness(t)
		f := lpNewRationalCoreFixture(t, h, 2, 4)
		using, only := world%2 == 0, world/2%2 == 0
		f.settings.SimplexStrategy = StrategyTableauCosts
		if !using && world/4%2 == 0 {
			f.settings.SimplexStrategy = StrategyTableauRows
		}
		f.settings.UseBreakpointsInFeasibilitySearch = world/8%2 == 0
		a := [][]*big.Rat{{big.NewRat(1, 1), new(big.Rat), big.NewRat(3, 7), big.NewRat(-1, 3)}, {new(big.Rat), big.NewRat(1, 1), big.NewRat(-2, 11), big.NewRat(5, 13)}}
		x := []*big.Rat{big.NewRat(int64(world%7-3), 2), big.NewRat(int64(world%5-2), 3), new(big.Rat), big.NewRat(2, 7)}

		original, cost := make([]*big.Rat, 4), make([]*big.Rat, 4)
		f.types[0], f.types[1] = ColumnBoxed, ColumnLowBound
		lows, ups := []*big.Rat{big.NewRat(-1, 1), new(big.Rat)}, []*big.Rat{big.NewRat(1, 1), new(big.Rat)}
		for i := 0; i < 2; i++ {
			h.must(h.m.Copy(&f.low.Data[i], lpScalarBoundValue(h, lows[i].RatString())))
			h.must(h.m.Copy(&f.up.Data[i], lpScalarBoundValue(h, ups[i].RatString())))
		}
		for j := range x {
			original[j] = big.NewRat(int64(7+j+world%5), 13)
			cost[j] = new(big.Rat).Set(original[j])
			h.must(h.m.Copy(&f.x.Data[j], lpScalarBoundValue(h, x[j].RatString())))
			h.must(h.m.SetString(&f.costs.Data[j], cost[j].RatString()))
		}
		for i := range a {
			bx := new(big.Rat)
			for j, v := range a[i] {
				if v.Sign() != 0 {
					h.must(f.a.Set(uint32(i), uint32(j), lpScalarBoundValue(h, v.RatString())))
				}
				bx.Add(bx, new(big.Rat).Mul(v, x[j]))

			}
			h.must(h.m.Copy(&f.b.Data[i], lpScalarBoundValue(h, bx.RatString())))
		}
		m, e := NewRationalPrimalCoreSolver(f.input())
		h.must(e)
		h.must(m.BackupAndNormalizeCosts())
		h.must(m.InitInfSet())
		if using {
			h.must(m.InitInfeasibilityCosts())
			for j := range cost {
				c := int64(0)
				if j < 2 {
					c = lpRationalAdvanceOracleCost(x[j], f.types[j], lows[j], ups[j])
					if f.settings.UseBreakpointsInFeasibilitySearch {
						c = -c
					}
				}
				cost[j].SetInt64(c)
			}
		}
		oldCost := make([]*big.Rat, 4)
		for j := range cost {
			oldCost[j] = new(big.Rat).Set(cost[j])
			h.must(h.m.SetInt32(&m.D.Data[j], 97))
		}
		m.LookForFeasibleSolutionOnly = only
		m.IterationsNoCostGrowth = 73
		m.Status = StatusOptimal
		sign := int64(world/16%3 - 1)
		m.SignOfEnteringDelta = int32(sign)
		tx := big.NewRat(int64(world%11-5), 7)
		step := lpScalarBoundValue(h, tx.RatString())
		dx := new(big.Rat).Mul(tx, big.NewRat(sign, 1))
		x[2].Add(x[2], dx)

		members := []bool{false, false}
		for i := range a {
			x[i].Sub(x[i], new(big.Rat).Mul(a[i][2], dx))

			c := lpRationalAdvanceOracleCost(x[i], f.types[i], lows[i], ups[i])
			members[i] = c != 0
			if using {
				if f.settings.UseBreakpointsInFeasibilitySearch {
					c = -c
				}
				cost[i].SetInt64(c)
			}
		}
		feasible := !members[0] && !members[1]
		early := only && feasible
		switched := !early && f.settings.SimplexStrategy != StrategyTableauRows && (feasible == using)
		wantUsing := using
		if switched {
			if feasible {
				wantUsing = false
				for j := range cost {
					cost[j].Set(original[j])
				}
			} else {
				wantUsing = true
				for j := range cost {
					c := int64(0)
					if j < 2 {
						c = lpRationalAdvanceOracleCost(x[j], f.types[j], lows[j], ups[j])
						if f.settings.UseBreakpointsInFeasibilitySearch {
							c = -c
						}
					}
					cost[j].SetInt64(c)
				}
			}
		}
		h.must(m.AdvanceOnEnteringEqualLeavingTableau(2, step))
		for j := range x {
			lpSparseRationalCheck(t, h, &f.x.Data[j], x[j])
			lpSparseRationalCheck(t, h, &f.costs.Data[j], cost[j])
			d := big.NewRat(97, 1)
			if switched {
				d.SetInt64(0)
				if j >= 2 {
					d.Set(cost[j])
					for i := range a {
						d.Sub(d, new(big.Rat).Mul(cost[i], a[i][j]))
					}
				}
			} else if using && j >= 2 {
				for i := range a {
					d.Add(d, new(big.Rat).Mul(new(big.Rat).Sub(oldCost[i], cost[i]), a[i][j]))
				}
			}
			lpSparseRationalCheck(t, h, &m.D.Data[j], d)
		}
		lpSparseRationalCheck(t, h, step, tx)
		counter := uint32(0)
		if early {
			counter = 73
		}
		if m.UsingInfeasCosts != wantUsing || m.InfSet.Contains(0) != members[0] || m.InfSet.Contains(1) != members[1] || m.IterationsNoCostGrowth != counter || m.Status != StatusOptimal || m.tableauDelta != nil || m.Factorization != nil || !reflect.DeepEqual(f.basis, []uint32{0, 1}) {
			t.Fatal("state", world)
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPRationalTableauAdvancePivotIndependentModels(t *testing.T) {
	for world := 0; world < 120; world++ {
		h := newLPPairHarness(t)
		f := lpNewRationalCoreFixture(t, h, 2, 4)
		f.settings.SimplexStrategy = StrategyTableauCosts
		if world%2 == 0 {
			f.settings.SimplexStrategy = StrategyTableauRows
		}
		p, q := big.NewRat(int64(1+world%7), 5), big.NewRat(int64(world%9-4), 3)
		a := [][]*big.Rat{{big.NewRat(1, 1), new(big.Rat), new(big.Rat).Set(p), big.NewRat(2, 7)}, {new(big.Rat), big.NewRat(1, 1), new(big.Rat).Set(q), big.NewRat(-3, 11)}}
		x := []*big.Rat{new(big.Rat), new(big.Rat), big.NewRat(1, 7), big.NewRat(-1, 5)}
		cost := []*big.Rat{big.NewRat(3, 1), big.NewRat(-2, 1), big.NewRat(5, 1), big.NewRat(7, 1)}
		for i := range a {
			x[i].Neg(new(big.Rat).Add(new(big.Rat).Mul(a[i][2], x[2]), new(big.Rat).Mul(a[i][3], x[3])))
			for j, v := range a[i] {
				if v.Sign() != 0 {
					h.must(f.a.Set(uint32(i), uint32(j), lpScalarBoundValue(h, v.RatString())))
				}
			}
		}
		for j := range x {
			h.must(h.m.Copy(&f.x.Data[j], lpScalarBoundValue(h, x[j].RatString())))
			h.must(h.m.SetString(&f.costs.Data[j], cost[j].RatString()))
		}
		m, e := NewRationalPrimalCoreSolver(f.input())
		h.must(e)
		h.must(m.InitReducedCostsTableau())
		m.NonBasisList = []uint32{3, 2}
		m.IterationsNoCostGrowth = 73
		m.Status = StatusOptimal
		only := world/2%2 == 0
		m.LookForFeasibleSolutionOnly = only
		sign := int32(1)
		if world/4%2 == 0 {
			sign = -1
		}
		m.SignOfEnteringDelta = sign
		tx := big.NewRat(int64(world%5), 11)
		step := lpScalarBoundValue(h, tx.RatString())
		signed := new(big.Rat).Mul(tx, big.NewRat(int64(sign), 1))
		oldD := make([]*big.Rat, 4)
		for j := range oldD {
			oldD[j] = new(big.Rat)
			if j >= 2 {
				oldD[j].Set(cost[j])
				for i := range a {
					oldD[j].Sub(oldD[j], new(big.Rat).Mul(cost[i], a[i][j]))
				}
			}
		}
		x[2].Add(x[2], signed)
		x[0].Sub(x[0], new(big.Rat).Mul(p, signed))
		x[1].Sub(x[1], new(big.Rat).Mul(q, signed))
		for j := range a[0] {
			a[0][j].Quo(a[0][j], p)
		}
		for j := range a[1] {
			a[1][j].Sub(a[1][j], new(big.Rat).Mul(q, a[0][j]))
		}
		h.must(m.AdvanceOnEnteringAndLeavingTableau(2, 0, step))
		lpSparseRationalCheck(t, h, step, signed)
		for j := range x {
			lpSparseRationalCheck(t, h, &f.x.Data[j], x[j])
			d := new(big.Rat).Set(oldD[j])
			if f.settings.SimplexStrategy == StrategyTableauCosts {
				d.SetInt64(0)
				if f.heading[j] < 0 {
					d.Set(cost[j])
					d.Sub(d, new(big.Rat).Mul(cost[2], a[0][j]))
					d.Sub(d, new(big.Rat).Mul(cost[1], a[1][j]))
				}
			}
			lpSparseRationalCheck(t, h, &m.D.Data[j], d)
		}
		counter := uint32(0)
		if tx.Sign() == 0 {
			counter = 73
		}
		list := []uint32{3, 2}
		if !only && f.settings.SimplexStrategy != StrategyTableauRows {
			list[1] = 0
		}
		if !reflect.DeepEqual(f.basis, []uint32{2, 1}) || !reflect.DeepEqual(m.NonBasisList, list) || m.IterationsNoCostGrowth != counter || m.Status != StatusOptimal || m.Factorization != nil {
			t.Fatal("state", world)
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPRationalTableauAdvancePhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	for _, op := range []string{"equal-positive", "equal-negative", "equal-zero", "restore", "pivot-positive", "pivot-negative", "pivot-zero", "min-negation"} {
		for _, direct := range []bool{false, true} {
			success, failures, childStops := false, 0, 0
			negationFailure, assignmentFailure := false, false
			for ordinal := int64(0); ordinal < 1500; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("%s/%t/%d", op, direct, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					f := lpNewRationalCoreFixture(t, h, 2, 3)
					f.settings.SimplexStrategy = StrategyTableauCosts
					huge := strings.Repeat("1234567890", 12)
					for i := uint32(0); i < 2; i++ {
						h.must(f.a.Set(i, i, lpScalarBoundValue(h, "1")))
						h.must(f.a.Set(i, 2, lpScalarBoundValue(h, huge+fmt.Sprintf("%d/13", i+1))))
					}
					m, e := NewRationalPrimalCoreSolver(f.input())
					h.must(e)
					m.SignOfEnteringDelta = 1
					m.NonBasisList = []uint32{2}
					m.IterationsNoCostGrowth = 73
					step := lpScalarBoundValue(h, huge+"9/19")
					if op == "equal-negative" || op == "pivot-negative" {
						m.SignOfEnteringDelta = -1
					}
					if op == "equal-zero" {
						m.SignOfEnteringDelta = 0
					}
					if op == "pivot-zero" {
						h.must(h.m.Copy(step, lpScalarBoundValue(h, "0")))
					}
					if op == "restore" {
						for j := range f.costs.Data {
							h.must(h.m.SetString(&f.costs.Data[j], huge+"3/17"))
						}
						h.must(m.BackupAndNormalizeCosts())
						h.must(m.InitInfeasibilityCosts())
					}
					if op == "min-negation" {
						m.SignOfEnteringDelta = -1
						m.UsingInfeasCosts = true
						h.must(m.CostsBackup.Resize(3))
						h.must(h.m.SetInt32(step, -2147483648))
					}
					h.budget.FailAfter(ordinal)
					var err error
					equal := strings.HasPrefix(op, "equal-") || op == "restore"
					if equal {
						err = m.AdvanceOnEnteringEqualLeavingTableau(2, step)
					} else {
						err = m.AdvanceOnEnteringAndLeavingTableau(2, 0, step)
					}
					h.budget.FailAfter(-1)
					if err == nil {
						success, done = true, true
						if op == "min-negation" {
							lpSparseRationalCheck(t, h, step, big.NewRat(2147483648, 1))
						}
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						// Later cost initialization also owns work; only unary negation owns one slot.
						if op == "min-negation" && m.work != nil && len(m.work.Data) == 1 {
							value, inspectErr := h.m.String(step)
							if inspectErr == nil {
								if value != "-2147483648" {
									t.Fatal("pre-assignment prefix", value)
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
						if m.calls.pending() && m.RationalCoreSolverBase.calls.pending() {
							t.Fatal("adopted child")
						}
						if m.tableauDelta != nil && m.RationalCoreSolverBase.calls.pending() {
							childStops++
							if m.work != nil {
								t.Fatal("scalar temporary survived child")
							}
						}
						if e = m.AdvanceOnEnteringEqualLeavingTableau(2, step); e != ErrExactVectorPending {
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
			if !success || (direct && op != "equal-zero" && failures == 0) {
				t.Fatal("campaign", op, direct, failures)
			}
			if direct && op == "min-negation" && (!negationFailure || !assignmentFailure) {
				t.Fatal("missing minimum-negation prefixes", negationFailure, assignmentFailure)
			}
			if direct && (op == "equal-positive" || op == "equal-negative" || op == "restore") && childStops == 0 {
				t.Fatal("no actual stopped child borrowing expression", op)
			}
		}
	}
}

func TestLPRationalTableauAdvanceEqualLiveStepAliases(t *testing.T) {
	for _, target := range []int{0, 1, 2} {
		h := newLPPairHarness(t)
		f := lpNewRationalCoreFixture(t, h, 2, 3)
		f.settings.SimplexStrategy = StrategyTableauCosts
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
		m.SignOfEnteringDelta = -1
		tx := big.NewRat(int64(target+3), 1)
		h.must(m.AdvanceOnEnteringEqualLeavingTableau(2, &f.x.Data[target]))
		lpSparseRationalCheck(t, h, &f.x.Data[2], new(big.Rat).Sub(big.NewRat(5, 1), tx))
		for i := 0; i < 2; i++ {
			lpSparseRationalCheck(t, h, &f.x.Data[i], new(big.Rat).Add(big.NewRat(int64(i+3), 1), new(big.Rat).Mul(big.NewRat(int64(i+2), 1), tx)))
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPRationalTableauAdvanceRestoresCostsAndEarlyReturn(t *testing.T) {
	for _, zeroStep := range []bool{false, true} {
		for _, only := range []bool{false, true} {
			for _, breakpoints := range []bool{false, true} {
				h := newLPPairHarness(t)
				f := lpNewRationalCoreFixture(t, h, 1, 2)
				f.settings.SimplexStrategy = StrategyTableauCosts
				f.settings.UseBreakpointsInFeasibilitySearch = breakpoints
				h.must(f.a.Set(0, 0, lpScalarBoundValue(h, "1")))
				h.must(f.a.Set(0, 1, lpScalarBoundValue(h, "-1")))
				tx := int64(1)
				if zeroStep {
					tx = 0
				} else {
					f.types[0] = ColumnLowBound
					h.must(h.m.Copy(&f.x.Data[0], lpScalarBoundValue(h, "-1")))
					h.must(h.m.Copy(&f.x.Data[1], lpScalarBoundValue(h, "-1")))
				}
				h.must(h.m.SetInt32(&f.costs.Data[0], 3))
				h.must(h.m.SetInt32(&f.costs.Data[1], 5))
				m, e := NewRationalPrimalCoreSolver(f.input())
				h.must(e)
				h.must(m.BackupAndNormalizeCosts())
				h.must(m.InitInfeasibilityCosts())
				h.must(m.InitReducedCostsTableau())
				// InitReducedCosts restores a feasible initial model; explicitly activate
				// the zero-cost source profile again for the zero-step restoration case.
				if zeroStep {
					h.must(m.InitInfeasibilityCosts())
					for j := range m.D.Data {
						h.must(h.m.SetInt32(&m.D.Data[j], 0))
					}
				}
				m.SignOfEnteringDelta = 1
				if breakpoints && !zeroStep {
					m.SignOfEnteringDelta = -1
				}
				m.LookForFeasibleSolutionOnly = only
				m.IterationsNoCostGrowth = 73
				m.NonBasisList = []uint32{1}
				m.Status = StatusOptimal
				step := lpScalarBoundValue(h, fmt.Sprint(tx))
				h.must(m.AdvanceOnEnteringAndLeavingTableau(1, 0, step))
				lpSparseRationalCheck(t, h, step, big.NewRat(tx, 1))
				for j := range f.x.Data {
					lpSparseRationalCheck(t, h, &f.x.Data[j], new(big.Rat))
				}
				wantList := []uint32{0}
				if only {
					wantList[0] = 1
				}
				if !reflect.DeepEqual(m.NonBasisList, wantList) || m.UsingInfeasCosts != only || m.InfSet.Size() != 0 || m.Status != StatusOptimal {
					t.Fatal("state", zeroStep, only, breakpoints)
				}
				counter := uint32(0)
				if zeroStep {
					counter = 73
				}
				if m.IterationsNoCostGrowth != counter {
					t.Fatal("counter")
				}
				for j := range f.costs.Data {
					want := int64(3 + 2*j)
					if only {
						want = 0
					}
					lpSparseRationalCheck(t, h, &f.costs.Data[j], big.NewRat(want, 1))
				}
				wantD := int64(8)
				if only {
					wantD = 0
				}
				lpSparseRationalCheck(t, h, &m.D.Data[0], big.NewRat(wantD, 1))
				lpSparseRationalCheck(t, h, &m.D.Data[1], new(big.Rat))
				h.must(m.Close())
				f.close()
				lpExactVectorNamedCensus(t, h, nil)
			}
		}
	}
}

func TestLPRationalTableauAdvanceReachedPrefixes(t *testing.T) {
	for _, phase := range []string{"same-nil", "same-negative-index", "nil", "negative-index", "negative-step", "equal-zero", "empty-list", "wrong-list", "rows-empty-list", "heading-extent", "empty-backup", "active-local", "source-post-residual", "same-residual", "zero-pivot-row", "zero-basis-error", "pivot-empty-backup", "equal-positive", "nil-using", "zero-heading-extent"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			f := lpNewRationalCoreFixture(t, h, 1, 2)
			f.settings.SimplexStrategy = StrategyTableauCosts
			h.must(f.a.Set(0, 0, lpScalarBoundValue(h, "1")))
			h.must(f.a.Set(0, 1, lpScalarBoundValue(h, "2")))
			m, e := NewRationalPrimalCoreSolver(f.input())
			h.must(e)
			m.SignOfEnteringDelta = 1
			m.NonBasisList = []uint32{1}
			m.IterationsNoCostGrowth = 73
			step := lpScalarBoundValue(h, "1")
			en, lv := int32(1), int32(0)
			same := false
			want := error(nil)
			beforeValues := false
			switch phase {
			case "same-nil":
				same = true
				step = nil
				want = ErrBounds
				beforeValues = true
			case "same-negative-index":
				same = true
				en = -1
				want = ErrBounds
				beforeValues = true
			case "nil":
				step = nil
				want = ErrBounds
				beforeValues = true
			case "negative-index":
				lv = -1
				want = ErrBounds
				if sourceDebugEnabled {
					want = ErrAssertion
					beforeValues = true
				}
			case "negative-step":
				h.must(h.m.Copy(step, lpScalarBoundValue(h, "-1")))
				if sourceDebugEnabled {
					want = ErrAssertion
					beforeValues = true
				}
			case "equal-zero":
				lv = en
				h.must(h.m.Copy(step, lpScalarBoundValue(h, "0")))
				if sourceDebugEnabled {
					want = ErrAssertion
				}
				beforeValues = true
			case "empty-list":
				m.NonBasisList = nil
				want = ErrBounds
				if sourceDebugEnabled {
					beforeValues = true
				}
			case "wrong-list":
				m.NonBasisList = []uint32{99}
				if sourceDebugEnabled {
					want = ErrAssertion
					beforeValues = true
				}
			case "rows-empty-list":
				f.settings.SimplexStrategy = StrategyTableauRows
				m.NonBasisList = nil
			case "heading-extent":
				lv = 99
				want = ErrBounds
			case "empty-backup":
				same = true
				m.UsingInfeasCosts = true
				want = ErrBounds
			case "active-local":
				same = true
				m.tableauDelta = NewIndexedRationalVector(h.m, 1)
				want = ErrExactVectorPending
				beforeValues = true
			case "equal-positive":
				lv = en
			case "nil-using":
				m.UsingInfeasCosts = true
				step = nil
				want = ErrBounds
				beforeValues = true
			case "zero-heading-extent":
				h.must(h.m.Copy(step, lpScalarBoundValue(h, "0")))
				lv = 99
				want = ErrBounds
				beforeValues = true
			case "same-residual":
				same = true
				h.must(h.m.Copy(&f.b.Data[0], lpScalarBoundValue(h, "1")))
				if sourceDebugEnabled {
					want = ErrAssertion
					beforeValues = true
				}
			case "zero-pivot-row":
				h.must(h.m.Copy(step, lpScalarBoundValue(h, "0")))
				f.heading[0] = 99
				want = ErrBounds
				beforeValues = true
			case "zero-basis-error":
				h.must(h.m.Copy(step, lpScalarBoundValue(h, "0")))
				f.nb = nil
				want = ErrBounds
				beforeValues = true
			case "pivot-empty-backup":
				m.UsingInfeasCosts = true
				want = ErrBounds
			case "source-post-residual":
				h.must(h.m.Copy(&f.b.Data[0], lpScalarBoundValue(h, "1")))
				if sourceDebugEnabled {
					want = ErrAssertion
					beforeValues = true
				}
			}
			var err error
			lpCaptureFeasibleStdout(t, func() {
				if same {
					err = m.AdvanceOnEnteringEqualLeavingTableau(en, step)
				} else {
					err = m.AdvanceOnEnteringAndLeavingTableau(en, lv, step)
				}
			})
			if err != want {
				t.Fatal(phase, err, want)
			}
			vx := int64(1)
			if beforeValues {
				vx = 0
			}
			if phase == "negative-step" && !sourceDebugEnabled {
				vx = -1
			}
			lpSparseRationalCheck(t, h, &f.x.Data[1], big.NewRat(vx, 1))
			lpSparseRationalCheck(t, h, &f.x.Data[0], big.NewRat(-2*vx, 1))
			if want != nil && m.IterationsNoCostGrowth != 73 && phase != "empty-list" && phase != "pivot-empty-backup" {
				t.Fatal("premature counter reset", phase, m.IterationsNoCostGrowth)
			}
			if phase == "equal-positive" && f.basis[0] != 0 {
				t.Fatal("bound flip changed basis")
			}
			if phase == "same-negative-index" && m.tableauDelta == nil {
				t.Fatal("full expression before value bounds")
			}
			if phase == "empty-backup" || phase == "pivot-empty-backup" {
				if m.UsingInfeasCosts || len(f.costs.Data) != 0 || m.tableauDelta != nil {
					t.Fatal("restore prefix")
				}
			}
			if want != nil {
				if e = m.AdvanceOnEnteringAndLeavingTableau(1, 0, lpScalarBoundValue(h, "1")); e != ErrExactVectorPending {
					t.Fatal("retry", e)
				}
			}
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPRationalTableauAdvancePreservesOlderComparison(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, op := range []string{"equal", "restore", "pivot"} {
		matched := false
		for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
			h := newLPPairHarness(t)
			f := lpNewRationalCoreFixture(t, h, 1, 3)
			f.settings.SimplexStrategy = StrategyTableauCosts
			h.must(f.a.Set(0, 0, lpScalarBoundValue(h, "1")))
			h.must(f.a.Set(0, 1, lpScalarBoundValue(h, "2/7")))
			m, e := NewRationalPrimalCoreSolver(f.input())
			h.must(e)
			v := h.pair(strings.Repeat("9876543210", 100)+"/7", strings.Repeat("1234567890", 100)+"/13")
			h.must(h.m.Copy(&f.costs.Data[0], &v.X))
			h.must(m.CostsBackup.Resize(3))
			h.must(h.m.Copy(&m.CostsBackup.Data[0], &v.Y))
			if op == "restore" {
				m.UsingInfeasCosts = true
			}
			delta := lpScalarBoundValue(h, "1/7")
			m.SignOfEnteringDelta = 1
			m.NonBasisList = []uint32{2, 1}

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
				// Real numeric allocation may encounter the older comparison
				// allocator receipt; the solver must neither adopt nor release it.
				if op == "pivot" {
					err = m.AdvanceOnEnteringAndLeavingTableau(1, 0, delta)
				} else {
					err = m.AdvanceOnEnteringEqualLeavingTableau(1, delta)
				}
				if err != nil && !lpExactPriorityPhysicalRefusal(err) && !errors.Is(err, arena.ErrInvalidSlot) && !errors.Is(err, numeral.ErrRationalRawArithmeticPending) && !errors.Is(err, umemory.ErrAllocatorPending) {
					t.Fatal(op, err)
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
			t.Fatal("no actual older comparison", op)
		}
	}
}

func TestLPRationalTableauAdvanceNonzeroRHSSourceStop(t *testing.T) {
	h := newLPPairHarness(t)
	f := lpNewRationalCoreFixture(t, h, 2, 3)
	f.settings.SimplexStrategy = StrategyTableauCosts
	for _, c := range []struct {
		i, j uint32
		v    string
	}{{0, 0, "1"}, {1, 1, "1"}, {0, 2, "2"}, {1, 2, "3"}} {
		h.must(f.a.Set(c.i, c.j, lpScalarBoundValue(h, c.v)))
	}
	for j, v := range []string{"4", "7", "1"} {
		h.must(h.m.Copy(&f.x.Data[j], lpScalarBoundValue(h, v)))
	}
	for i, v := range []string{"6", "10"} {
		h.must(h.m.Copy(&f.b.Data[i], lpScalarBoundValue(h, v)))
	}
	m, e := NewRationalPrimalCoreSolver(f.input())
	h.must(e)
	m.SignOfEnteringDelta = 1
	m.NonBasisList = []uint32{2}
	m.IterationsNoCostGrowth = 73
	var err error
	output := lpCaptureFeasibleStdout(t, func() { err = m.AdvanceOnEnteringAndLeavingTableau(2, 0, lpScalarBoundValue(h, "1")) })
	want := error(nil)
	counter := uint32(0)
	list := []uint32{0}
	if sourceDebugEnabled {
		want = ErrAssertion
		counter = 73
		list = []uint32{2}
	}
	if err != want || m.IterationsNoCostGrowth != counter || !reflect.DeepEqual(m.NonBasisList, list) {
		t.Fatal("source post-pivot stop", err, counter, m.NonBasisList)
	}
	if sourceDebugEnabled && !strings.Contains(output, "precise x is off") {
		t.Fatal("actual residual diagnostic", output)
	}

	for j, v := range []int64{2, 4, 2} {
		lpSparseRationalCheck(t, h, &f.x.Data[j], big.NewRat(v, 1))
	}
	lpSparseRationalCheck(t, h, &f.b.Data[0], big.NewRat(3, 1))
	lpSparseRationalCheck(t, h, &f.b.Data[1], big.NewRat(10, 1))
	// Correct row elimination gives (-3/2)*x0+x1 = 1, whereas source
	// retains the old RHS 10. The returned success does not prove AX=B.
	out := lpScalarBoundValue(h, "0")
	h.must(f.a.DotRow(1, f.x.Data, out))
	lpSparseRationalCheck(t, h, out, big.NewRat(1, 1))
	h.must(m.Close())
	f.close()
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPRationalTableauAdvanceZeroPivotCostSwitchRecheck(t *testing.T) {
	h := newLPPairHarness(t)
	f := lpNewRationalCoreFixture(t, h, 1, 2)
	f.settings.SimplexStrategy = StrategyTableauCosts
	h.must(f.a.Set(0, 0, lpScalarBoundValue(h, "1")))
	h.must(f.a.Set(0, 1, lpScalarBoundValue(h, "1")))
	f.types[0] = ColumnLowBound
	h.must(h.m.Copy(&f.x.Data[0], lpScalarBoundValue(h, "-1")))
	h.must(h.m.Copy(&f.x.Data[1], lpScalarBoundValue(h, "1")))
	m, e := NewRationalPrimalCoreSolver(f.input())
	h.must(e)
	h.must(m.InitInfSet())
	m.NonBasisList = []uint32{1}
	m.SignOfEnteringDelta = 1
	m.IterationsNoCostGrowth = 73
	// The zero pivot leaves membership unchanged, then switching recomputes it
	// from the new basic set. The new free basic produces no infeasibility costs,
	// so the source's one-pass switch still needs original-cost restoration.
	err := m.AdvanceOnEnteringAndLeavingTableau(1, 0, lpScalarBoundValue(h, "0"))
	want := error(nil)
	list := []uint32{0}
	if sourceDebugEnabled {
		want = ErrAssertion
		list[0] = 1
	}
	if err != want || !m.UsingInfeasCosts || m.InfSet.Size() != 0 || !m.NeedToSwitchCosts() || m.IterationsNoCostGrowth != 73 || !reflect.DeepEqual(m.NonBasisList, list) || f.basis[0] != 1 {
		t.Fatal("cost recheck", err, m.NonBasisList)
	}
	for j := range f.costs.Data {
		lpSparseRationalCheck(t, h, &f.costs.Data[j], new(big.Rat))
		lpSparseRationalCheck(t, h, &m.D.Data[j], new(big.Rat))
	}
	h.must(m.Close())
	f.close()
	lpExactVectorNamedCensus(t, h, nil)
}
