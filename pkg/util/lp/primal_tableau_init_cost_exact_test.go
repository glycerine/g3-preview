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

func TestLPExactTableauInitCostIndependentModels(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for world := 0; world < 320; world++ {
		h := newLPPairHarness(t)
		f := lpNewPairCoreFixture(t, h, 2, 6)
		f.basis = []uint32{4, 1}
		f.settings.SimplexStrategy = StrategyTableauCosts
		if world%2 == 0 {
			f.settings.SimplexStrategy = StrategyTableauRows
		}
		f.settings.UseBreakpointsInFeasibilitySearch = world/2%2 == 0
		a := make([][]*big.Rat, 2)
		for i := range a {
			a[i] = make([]*big.Rat, 6)
			for j := range a[i] {
				v := big.NewRat(int64((world+i*7+j*11)%19-9), int64(7+i+j))
				if uint32(j) == f.basis[i] {
					v.SetInt64(1)
				} else if uint32(j) == f.basis[1-i] {
					v.SetInt64(0)
				}
				a[i][j] = v
				if v.Sign() != 0 {
					h.must(f.a.Set(uint32(i), uint32(j), &h.pair(v.RatString(), "0").X))
				}
			}
		}
		m, e := NewPairPrimalCoreSolver(f.input())
		h.must(e)
		h.must(m.CostsBackup.Resize(6))
		m.CostsBackup.Index = nil
		cost := make([]*big.Rat, 6)
		original := make([]*big.Rat, 6)
		members := map[uint32]bool{}
		for j := range cost {
			original[j] = big.NewRat(int64((world+j*5)%29-14), 13)
			cost[j] = new(big.Rat).Add(original[j], big.NewRat(37, 19))
			h.must(h.m.SetString(&f.costs.Data[j], cost[j].RatString()))
			h.must(h.m.SetString(&m.CostsBackup.Data[j], original[j].RatString()))
			h.must(h.m.SetInt32(&m.D.Data[j], 97))
			kind := ColumnType((world + j) % 5)
			f.types[j] = kind
			low, up := big.NewRat(-1, 1), big.NewRat(1, 1)
			if kind == ColumnFixed {
				low.SetInt64(0)
				up.SetInt64(0)
			}
			h.must(f.low.Data[j].Copy(h.m, h.pair(low.RatString(), "0")))
			h.must(f.up.Data[j].Copy(h.m, h.pair(up.RatString(), "0")))
			h.must(f.x.Data[j].Copy(h.m, h.pair(big.NewRat(int64((world+j)%7-3), 2).RatString(), big.NewRat(int64((world+j)%3-1), 7).RatString())))
		}
		storedInf, using, only := world/4%2 == 0, world/8%2 == 0, world/16%2 == 0
		if storedInf {
			h.must(m.InfSet.Insert(4))
			members[4] = true
		}
		m.UsingInfeasCosts = using
		m.LookForFeasibleSolutionOnly = only
		early := !storedInf && using && only
		wantUsing := using
		if storedInf && !using {
			members = map[uint32]bool{}
			wantUsing = true
			for j := range cost {
				c := int64(0)
				if f.heading[j] >= 0 {
					x, y := big.NewRat(int64((world+j)%7-3), 2), big.NewRat(int64((world+j)%3-1), 7)
					low, up := big.NewRat(-1, 1), big.NewRat(1, 1)
					if f.types[j] == ColumnFixed {
						low.SetInt64(0)
						up.SetInt64(0)
					}
					c = lpAdvanceEqualOracleCost(x, y, f.types[j], low, up)
					if f.settings.UseBreakpointsInFeasibilitySearch {
						c = -c
					}
				}
				cost[j] = big.NewRat(c, 1)
				if c != 0 {
					members[uint32(j)] = true
				}
			}
		} else if !storedInf && using && !only {
			wantUsing = false
			for j := range cost {
				cost[j] = new(big.Rat).Set(original[j])
			}
		}
		m.D.Index = []uint32{5, 0}
		oldIndex := append([]uint32(nil), m.D.Index...)
		m.Status = StatusOptimal
		m.IterationsNoCostGrowth = 37
		m.BasisSortCounter = 41
		h.must(m.InitReducedCostsTableau())
		for j := range cost {
			want := new(big.Rat)
			if early {
				want.SetInt64(97)
			} else if f.heading[j] < 0 {
				want.Set(cost[j])
				for i := range a {
					want.Sub(want, new(big.Rat).Mul(cost[f.basis[i]], a[i][j]))
				}
			}
			lpSparseRationalCheck(t, h, &m.D.Data[j], want)
			lpSparseRationalCheck(t, h, &f.costs.Data[j], cost[j])
			if m.InfSet.Contains(uint32(j)) != members[uint32(j)] {
				t.Fatal("membership", world, j)
			}
		}
		if m.UsingInfeasCosts != wantUsing || m.Status != StatusOptimal || m.IterationsNoCostGrowth != 37 || m.BasisSortCounter != 41 || !reflect.DeepEqual(oldIndex, m.D.Index) || m.work != nil || m.Factorization != nil {
			t.Fatal("state", world)
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPExactTableauInitCostPhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, op := range []string{"fill", "restore", "infeasible"} {
		for _, direct := range []bool{false, true} {
			success, failures := false, 0
			for ordinal := int64(0); ordinal < 1500; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("%s/%t/%d", op, direct, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					f := lpNewPairCoreFixture(t, h, 1, 3)
					f.settings.SimplexStrategy = StrategyTableauCosts
					h.must(f.a.Set(0, 0, &h.pair("1", "0").X))
					huge := strings.Repeat("1234567890", 12)
					for j := uint32(1); j < 3; j++ {
						h.must(f.a.Set(0, j, &h.pair(huge+fmt.Sprintf("%d/13", j), "0").X))
					}
					m, e := NewPairPrimalCoreSolver(f.input())
					h.must(e)
					h.must(m.CostsBackup.Resize(3))
					for j := range f.costs.Data {
						h.must(h.m.SetString(&f.costs.Data[j], huge+fmt.Sprintf("%d/17", j+1)))
						h.must(h.m.SetString(&m.CostsBackup.Data[j], huge+fmt.Sprintf("%d/19", j+3)))
					}
					if op == "restore" {
						m.UsingInfeasCosts = true
					}
					if op == "infeasible" {
						h.must(m.InfSet.Insert(0))
						f.types[0] = ColumnLowBound
						h.must(f.x.Data[0].Copy(h.m, h.pair("-"+huge+"1/23", "0")))
						h.must(f.low.Data[0].Copy(h.m, h.pair(huge+"1/29", "0")))
					}
					h.budget.FailAfter(ordinal)
					err := m.InitReducedCostsTableau()
					h.budget.FailAfter(-1)
					if err == nil {
						success, done = true, true
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						if e := m.InitReducedCostsTableau(); e != ErrExactVectorPending {
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
				t.Fatal("campaign", op, direct, failures)
			}
		}
	}
}

func TestLPExactTableauInitCostLiveAliases(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		h := newLPPairHarness(t)
		f := lpNewPairCoreFixture(t, h, 2, 5)
		f.settings.SimplexStrategy = StrategyTableauCosts
		f.basis = []uint32{2, 4}
		h.must(f.a.Set(0, 2, &h.pair("1", "0").X))
		h.must(f.a.Set(1, 4, &h.pair("1", "0").X))
		for _, c := range []struct {
			row, col uint32
			value    string
		}{{0, 0, "2"}, {1, 0, "3"}, {0, 1, "5"}, {1, 1, "7"}, {0, 3, "11"}, {1, 3, "13"}} {
			h.must(f.a.Set(c.row, c.col, &h.pair(c.value, "0").X))
		}
		if reverse {
			f.a.Columns[3][0], f.a.Columns[3][1] = f.a.Columns[3][1], f.a.Columns[3][0]
		}
		m, e := NewPairPrimalCoreSolver(f.input())
		h.must(e)
		h.must(m.D.Resize(6))
		for j, v := range []int32{5, 7, 11, 13, 17, 19} {
			h.must(h.m.SetInt32(&m.D.Data[j], v))
		}
		saved := m.Costs
		borrowed := m.D.Data[1:6]
		m.Costs = &borrowed
		m.D.Index = []uint32{5, 0}
		h.must(m.InitReducedCostsTableau())
		want := []int64{-76, -187, 0, -417, 0, 19}
		if reverse {
			want[3] = 2300
		}
		for j, v := range want {
			lpSparseRationalCheck(t, h, &m.D.Data[j], big.NewRat(v, 1))
		}
		if !reflect.DeepEqual(m.D.Index, []uint32{5, 0}) {
			t.Fatal("sparse metadata")
		}
		m.Costs = saved
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPExactTableauInitCostReachedPrefixes(t *testing.T) {
	for _, phase := range []string{"early", "empty-heading", "heading-extent", "d-extent", "cost-extent", "column-extent", "row-extent", "basic-cost", "coefficient-offset", "second-cell", "child-infeas", "restore-before-error"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			f := lpNewPairCoreFixture(t, h, 1, 3)
			f.settings.SimplexStrategy = StrategyTableauCosts
			h.must(f.a.Set(0, 0, &h.pair("1", "0").X))
			h.must(f.a.Set(0, 1, &h.pair("2", "0").X))
			m, e := NewPairPrimalCoreSolver(f.input())
			h.must(e)
			for j, v := range []int32{3, 5, 7} {
				h.must(h.m.SetInt32(&f.costs.Data[j], v))
				h.must(h.m.SetInt32(&m.D.Data[j], 97))
			}
			savedBackup, savedColumns := m.CostsBackup, f.a.Columns
			want := error(ErrBounds)
			switch phase {
			case "early":
				m.UsingInfeasCosts = true
				m.LookForFeasibleSolutionOnly = true
				m.CostsBackup = nil
				f.heading = nil
				want = nil
			case "empty-heading":
				f.heading = nil
				want = nil
			case "heading-extent":
				f.heading = append(f.heading, 0)
			case "d-extent":
				h.must(m.D.Resize(1))
			case "cost-extent":
				h.must(f.costs.Resize(1))
			case "column-extent":
				f.a.Columns = f.a.Columns[:1]
			case "row-extent":
				f.a.Columns[1][0].Row = 99
			case "basic-cost":
				f.basis[0] = 99
			case "coefficient-offset":
				f.a.Columns[1][0].Offset = 99
			case "second-cell":
				bad := f.a.Columns[1][0]
				bad.Row = 99
				f.a.Columns[1] = append(f.a.Columns[1], bad)
			case "child-infeas":
				h.must(m.InfSet.Insert(0))
				f.types = nil
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "restore-before-error":
				m.UsingInfeasCosts = true
			}
			if e := m.InitReducedCostsTableau(); e != want {
				t.Fatal(phase, e, want)
			}
			switch phase {
			case "early", "empty-heading", "child-infeas":
				for j := range m.D.Data {
					lpSparseRationalCheck(t, h, &m.D.Data[j], big.NewRat(97, 1))
				}
			case "second-cell":
				lpSparseRationalCheck(t, h, &m.D.Data[1], big.NewRat(-1, 1))
				lpSparseRationalCheck(t, h, &m.D.Data[2], big.NewRat(97, 1))
			case "column-extent", "row-extent", "basic-cost", "coefficient-offset":
				lpSparseRationalCheck(t, h, &m.D.Data[1], big.NewRat(5, 1))
			case "restore-before-error":
				if m.UsingInfeasCosts || len(f.costs.Data) != 0 {
					t.Fatal("restored cost publication")
				}
				lpSparseRationalCheck(t, h, &m.D.Data[0], new(big.Rat))
				lpSparseRationalCheck(t, h, &m.D.Data[1], big.NewRat(97, 1))
			}
			if want != nil {
				if e := m.InitReducedCostsTableau(); e != ErrExactVectorPending {
					t.Fatal("retry", e)
				}
			}
			m.CostsBackup = savedBackup
			f.a.Columns = savedColumns
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPExactTableauInitCostPreservesOlderComparison(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, op := range []string{"fill", "restore", "infeasible"} {
		matched := false
		for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
			h := newLPPairHarness(t)
			f := lpNewPairCoreFixture(t, h, 1, 3)
			f.settings.SimplexStrategy = StrategyTableauCosts
			h.must(f.a.Set(0, 0, &h.pair("1", "0").X))
			h.must(f.a.Set(0, 1, &h.pair("2/7", "0").X))
			m, e := NewPairPrimalCoreSolver(f.input())
			h.must(e)
			v := h.pair(strings.Repeat("9876543210", 100)+"/7", strings.Repeat("1234567890", 100)+"/13")
			h.must(h.m.Copy(&f.costs.Data[0], &v.X))
			h.must(m.CostsBackup.Resize(3))
			h.must(h.m.Copy(&m.CostsBackup.Data[0], &v.Y))
			if op == "restore" {
				m.UsingInfeasCosts = true
			}
			if op == "infeasible" {
				h.must(m.InfSet.Insert(0))
				f.types[0] = ColumnLowBound
				h.must(f.x.Data[0].Copy(h.m, v))
				h.must(f.low.Data[0].Copy(h.m, h.pair("1", "0")))
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
				// Real numeric allocation may encounter the older comparison
				// allocator receipt; the solver must neither adopt nor release it.
				err = m.InitReducedCostsTableau()
				if err != nil && !lpExactPriorityPhysicalRefusal(err) && !errors.Is(err, arena.ErrInvalidSlot) && !errors.Is(err, numeral.ErrRationalRawArithmeticPending) && !errors.Is(err, umemory.ErrAllocatorPending) {
					t.Fatal(op, err)
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
			t.Fatal("no actual older comparison", op)
		}
	}
}
