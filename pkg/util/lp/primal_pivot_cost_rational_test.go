package lp

import (
	"errors"
	"fmt"
	"github.com/glycerine/g3/pkg/arena"
	"math/big"
	"reflect"
	"strings"
	"testing"
)

// Recompute y from the exchanged basis, independently of the incremental dq
// formula. Exercise real LU pivot rows, both leaving rows and stored feasibility.
func TestLPRationalPivotCostIndependentBasisOracle(t *testing.T) {
	for world := 0; world < 240; world++ {
		h := newLPPairHarness(t)
		f := lpNewRationalCoreFixture(t, h, 2, 6)
		f.settings.SimplexStrategy = StrategyLU
		f.settings.UseBreakpointsInFeasibilitySearch = world/2%2 == 0
		a := [][]*big.Rat{
			{big.NewRat(2, 1), big.NewRat(1, 1), big.NewRat(3, 7), big.NewRat(-1, 3), big.NewRat(2, 17), new(big.Rat)},
			{big.NewRat(1, 1), big.NewRat(3, 1), big.NewRat(-2, 11), big.NewRat(5, 13), big.NewRat(4, 19), new(big.Rat)},
		}
		for i := range a {
			for j, v := range a[i] {
				if v.Sign() != 0 {
					h.must(f.a.Set(uint32(i), uint32(j), &h.pair(v.RatString(), "0").X))
				}
			}
		}
		costs := make([]*big.Rat, 6)
		for j := range costs {
			costs[j] = big.NewRat(int64((world*7+j*11)%47-23), int64(7+j))
			h.must(h.m.SetString(&f.costs.Data[j], costs[j].RatString()))
		}
		m, e := NewRationalPrimalCoreSolver(f.input())
		h.must(e)
		y := []*big.Rat{
			new(big.Rat).Quo(new(big.Rat).Sub(new(big.Rat).Mul(big.NewRat(3, 1), costs[0]), costs[1]), big.NewRat(5, 1)),
			new(big.Rat).Quo(new(big.Rat).Sub(new(big.Rat).Mul(big.NewRat(2, 1), costs[1]), costs[0]), big.NewRat(5, 1)),
		}
		for j := range costs {
			d := new(big.Rat).Set(costs[j])
			for i := range y {
				d.Sub(d, new(big.Rat).Mul(y[i], a[i][j]))
			}
			h.must(h.m.SetString(&m.D.Data[j], d.RatString()))
		}
		m.D.Index = []uint32{5, 0}
		oldIndex := append([]uint32(nil), m.D.Index...)
		m.ColumnsNZ = make([]uint32, 6)
		m.RowsNZ = make([]uint32, 2)
		leave := uint32(world % 2)
		h.must(m.SolveBd(2))
		h.must(m.CalculatePivotRowOfBInverse(leave))
		h.must(m.CalculatePivotRowWhenInverseReady(leave))
		h.must(m.PivotRow.SetValue(&h.pair("0", "0").X, 5))
		for i, j := 0, len(m.PivotRow.Index)-1; i < j; i, j = i+1, j-1 {
			m.PivotRow.Index[i], m.PivotRow.Index[j] = m.PivotRow.Index[j], m.PivotRow.Index[i]
		}
		ok, e := m.UpdateBasisAndX(2, int32(leave), lpScalarBoundValue(h, "0"))
		h.must(e)
		if !ok {
			t.Fatal("actual basis exchange failed", world)
		}
		infeasible := world/4%2 == 0
		if infeasible {
			h.must(m.InfSet.Insert(1 - leave))
		}
		m.IterationsNoCostGrowth = 37
		m.Status = StatusOptimal
		h.must(m.UpdateReducedCostsFromPivotRow(2, leave))
		if infeasible && !m.Settings.UseBreakpointsInFeasibilitySearch {
			costs[leave] = new(big.Rat)
		}
		b0, b1 := f.basis[0], f.basis[1]
		det := new(big.Rat).Sub(new(big.Rat).Mul(a[0][b0], a[1][b1]), new(big.Rat).Mul(a[0][b1], a[1][b0]))
		y[0].Quo(new(big.Rat).Sub(new(big.Rat).Mul(costs[b0], a[1][b1]), new(big.Rat).Mul(costs[b1], a[1][b0])), det)
		y[1].Quo(new(big.Rat).Sub(new(big.Rat).Mul(costs[b1], a[0][b0]), new(big.Rat).Mul(costs[b0], a[0][b1])), det)
		for j := range costs {
			want := new(big.Rat).Set(costs[j])
			for i := range y {
				want.Sub(want, new(big.Rat).Mul(y[i], a[i][j]))
			}
			lpSparseRationalCheck(t, h, &m.D.Data[j], want)
			lpSparseRationalCheck(t, h, &f.costs.Data[j], costs[j])
		}
		if !reflect.DeepEqual(oldIndex, m.D.Index) || m.work != nil || m.IterationsNoCostGrowth != 37 || m.Status != StatusOptimal || m.CurrentXIsInfeasible() != infeasible {
			t.Fatal("changed unrelated state", world)
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPRationalPivotCostLiveAliases(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		for _, costAlias := range []bool{false, true} {
			h := newLPPairHarness(t)
			f := lpNewRationalCoreFixture(t, h, 1, 4)
			f.basis[0] = 3
			m, e := NewRationalPrimalCoreSolver(f.input())
			h.must(e)
			h.must(m.ChangeBasis(2, 3))
			h.must(m.D.Resize(5))
			for j, v := range []int32{5, 7, 11, 13, 17} {
				h.must(h.m.SetInt32(&m.D.Data[j], v))
			}
			savedRow, savedCosts := m.PivotRow.Data, m.Costs
			m.PivotRow.Data = m.D.Data[1:5]
			m.PivotRow.Index = []uint32{1, 0, 2, 3}
			if reverse {
				m.PivotRow.Index = []uint32{0, 1, 2, 3}
			}
			if costAlias {
				m.Costs = &m.D.Data
				h.must(m.InfSet.Insert(0))
				m.Settings.UseBreakpointsInFeasibilitySearch = false
			}
			h.must(m.UpdateReducedCostsFromPivotRow(2, 3))
			want := []*big.Rat{big.NewRat(1175, 169), big.NewRat(-30, 13), new(big.Rat), big.NewRat(-11, 13), big.NewRat(17, 1)}
			if reverse {
				want[0] = big.NewRat(-12, 13)
			}
			if costAlias {
				want[3] = new(big.Rat)
			}
			for j, v := range want {
				lpSparseRationalCheck(t, h, &m.D.Data[j], v)
			}
			m.PivotRow.Data = savedRow
			m.Costs = savedCosts
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		}
	}
}

func TestLPRationalPivotCostPhysicalOwnership(t *testing.T) {
	for _, direct := range []bool{false, true} {
		for _, correction := range []bool{false, true} {
			success, failures := false, 0
			for ordinal := int64(0); ordinal < 1000; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("direct=%t/correction=%t/%d", direct, correction, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					f := lpNewRationalCoreFixture(t, h, 1, 4)
					f.basis[0] = 3
					m, e := NewRationalPrimalCoreSolver(f.input())
					h.must(e)
					h.must(m.ChangeBasis(2, 3))
					for j := uint32(0); j < 4; j++ {
						h.must(h.m.SetString(&m.D.Data[j], strings.Repeat("1234567890", 12)+fmt.Sprintf("%d/7", j+1)))
						h.must(m.PivotRow.SetValue(&h.pair(strings.Repeat("9876543210", 12)+fmt.Sprintf("%d/13", j+1), "0").X, j))
						h.must(h.m.SetString(&f.costs.Data[j], strings.Repeat("3141592653", 12)+"1/19"))
					}
					if correction {
						h.must(m.InfSet.Insert(0))
					}
					m.Settings.UseBreakpointsInFeasibilitySearch = false
					h.budget.FailAfter(ordinal)
					err := m.UpdateReducedCostsFromPivotRow(2, 3)
					h.budget.FailAfter(-1)
					if err == nil {
						success, done = true, true
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						if e := m.UpdateReducedCostsFromPivotRow(2, 3); e != ErrExactVectorPending {
							t.Fatal("stopped retry", e)
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
				t.Fatal("campaign incomplete", direct, correction, failures)
			}
		}
	}
}

func TestLPRationalPivotCostReachedPrefixes(t *testing.T) {
	for _, phase := range []string{"pivot-range", "d-range", "index-heading", "index-d", "leaving-d", "cost-range", "debug-entering", "debug-leaving"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			f := lpNewRationalCoreFixture(t, h, 1, 4)
			f.basis[0] = 3
			m, e := NewRationalPrimalCoreSolver(f.input())
			h.must(e)
			h.must(m.ChangeBasis(2, 3))
			h.must(h.m.SetInt32(&m.PivotRow.Data[2], 2))
			h.must(h.m.SetInt32(&m.D.Data[2], 6))
			h.must(h.m.SetInt32(&m.D.Data[0], 10))
			h.must(h.m.SetInt32(&m.PivotRow.Data[0], 1))
			m.PivotRow.Index = []uint32{0}
			want := ErrBounds
			savedCosts := m.Costs
			switch phase {
			case "pivot-range":
				h.must(m.PivotRow.Resize(2))
			case "d-range":
				h.must(m.D.Resize(2))
			case "index-heading":
				m.PivotRow.Index = []uint32{0, 9}
			case "index-d":
				h.must(m.D.Resize(3))
				m.PivotRow.Index = []uint32{0, 3}
				(*m.Heading)[3] = -1
				// leaving is deliberately changed so index3 reaches the D read.
			case "leaving-d":
				h.must(m.D.Resize(3))
			case "cost-range":
				costs := f.costs.Data[:2]
				m.Costs = &costs
				h.must(m.InfSet.Insert(0))
				m.Settings.UseBreakpointsInFeasibilitySearch = false
			case "debug-entering":
				if !sourceDebugEnabled {
					h.must(m.Close())
					f.close()
					return
				}
				(*m.Heading)[2] = -1
				want = ErrAssertion
			case "debug-leaving":
				if !sourceDebugEnabled {
					h.must(m.Close())
					f.close()
					return
				}
				(*m.Heading)[3] = 0
				want = ErrAssertion
			}
			leaving := uint32(3)
			if phase == "index-d" {
				leaving = 1
			}
			if err := m.UpdateReducedCostsFromPivotRow(2, leaving); err != want {
				t.Fatal(err, want)
			}
			if phase == "index-heading" || phase == "index-d" || phase == "leaving-d" || phase == "cost-range" {
				lpSparseRationalCheck(t, h, &m.D.Data[0], big.NewRat(7, 1))
			}
			if err := m.UpdateReducedCostsFromPivotRow(2, leaving); err != ErrExactVectorPending {
				t.Fatal("retry", err)
			}
			m.Costs = savedCosts
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPRationalPivotPreservesOlderComparison(t *testing.T) {
	for _, op := range []string{"compare", "cost"} {
		matched := false
		for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
			h := newLPPairHarness(t)
			f := lpNewRationalCoreFixture(t, h, 1, 3)
			m, e := NewRationalPrimalCoreSolver(f.input())
			h.must(e)
			v := h.pair(strings.Repeat("9876543210", 100)+"/7", strings.Repeat("1234567890", 100)+"/13")
			h.must(h.m.Copy(&m.Ed.Data[0], &v.X))
			h.must(h.m.Copy(&m.PivotRow.Data[1], &v.Y))
			h.must(h.m.Copy(&m.D.Data[1], &v.X))
			m.PivotRow.Index = []uint32{1}
			if op == "cost" {
				h.must(m.ChangeBasis(1, 0))
			}
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
				if op == "cost" {
					err = m.UpdateReducedCostsFromPivotRow(1, 0)
				} else {
					_, err = m.PivotsInColumnAndRowAreDifferent(1, 0)
				}
				// A real older failed comparison can stop subsequent numeral reads
				// with ErrInvalidSlot. It must retain its own generation and roots.
				if err != nil && !lpExactPriorityPhysicalRefusal(err) && !errors.Is(err, arena.ErrInvalidSlot) {
					t.Fatal(op, ordinal, err)
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
