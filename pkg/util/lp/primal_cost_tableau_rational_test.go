package lp

import (
	"errors"
	"fmt"
	"math/big"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestLPRationalTableauCostIndependentScalarOracle(t *testing.T) {
	h := newLPPairHarness(t)
	f := lpNewRationalCoreFixture(t, h, 1, 1)
	m, err := NewRationalPrimalCoreSolver(f.input())
	h.must(err)
	out := h.pair("0", "0")
	rng := rand.New(rand.NewPCG(719, 9281))
	for world := 0; world < 1600; world++ {
		kind := []ColumnType{ColumnFree, ColumnLowBound, ColumnUpperBound, ColumnBoxed, ColumnFixed}[world%5]
		lo := big.NewRat(int64(rng.IntN(11)-5), 3)
		up := big.NewRat(int64(rng.IntN(11)-5), 3)
		x := big.NewRat(int64(rng.IntN(17)-8), 5)
		if world%3 == 0 {
			x.Set(up)
		}
		if world%7 == 0 {
			x.Set(lo)
		}
		// Include inverted intervals: the source gives upper priority.
		for k, v := range []*numeral.Rational{&f.low.Data[0], &f.up.Data[0], &f.x.Data[0]} {
			value := []*big.Rat{lo, up, x}[k]
			h.must(h.m.SetString(v, value.RatString()))
		}
		f.types[0] = kind
		f.heading[0] = 0
		nonbasic := world%11 == 0
		if nonbasic {
			f.heading[0] = -1
		}
		f.settings.UseBreakpointsInFeasibilitySearch = world%2 == 0
		want := int64(0)
		if !nonbasic {
			if (kind == ColumnFixed || kind == ColumnBoxed || kind == ColumnUpperBound) && x.Cmp(up) > 0 {
				want = 1
			} else if (kind == ColumnFixed || kind == ColumnBoxed || kind == ColumnLowBound) && x.Cmp(lo) < 0 {
				want = -1
			}
			if !f.settings.UseBreakpointsInFeasibilitySearch {
				want = -want
			}
		}
		h.must(m.GetInfeasibilityCostForColumn(0, &out.X))
		lpSparseRationalCheck(t, h, &out.X, big.NewRat(want, 1))
	}
	h.must(m.Close())
	f.close()
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPRationalTableauCostIndependentReducedCostOracle(t *testing.T) {
	h := newLPPairHarness(t)
	rng := rand.New(rand.NewPCG(71928, 183))
	value := h.pair("0", "0")
	for world := 0; world < 160; world++ {
		f := lpNewRationalCoreFixture(t, h, 2, 4)
		f.settings.SimplexStrategy = StrategyTableauCosts
		coeff := [2][2]*big.Rat{}
		for row := 0; row < 2; row++ {
			h.must(h.m.SetInt32(&value.X, 1))
			h.must(f.a.Set(uint32(row), uint32(row), &value.X))
			for k := 0; k < 2; k++ {
				coeff[row][k] = big.NewRat(int64(rng.IntN(13)-6), int64(rng.IntN(5)+1))
				h.must(h.m.SetString(&value.X, coeff[row][k].RatString()))
				h.must(f.a.Set(uint32(row), uint32(k+2), &value.X))
			}
		}
		m, err := NewRationalPrimalCoreSolver(f.input())
		h.must(err)
		m.UsingInfeasCosts = true
		f.settings.UseBreakpointsInFeasibilitySearch = world%2 == 0
		// Form the full reduced objective c - c_B*A independently of
		// the incremental update under test; the identity basis has zero d_B.
		costs := [4]*big.Rat{}
		for k := range costs {
			costs[k] = big.NewRat(int64(rng.IntN(19)-9), 7)
			h.must(h.m.SetString(&f.costs.Data[k], costs[k].RatString()))
		}
		for k := range costs {
			d := new(big.Rat)
			if k >= 2 {
				d.Set(costs[k])
				for row := 0; row < 2; row++ {
					d.Sub(d, new(big.Rat).Mul(costs[row], coeff[row][k-2]))
				}
			}
			h.must(h.m.SetString(&m.D.Data[k], d.RatString()))
		}
		if world%3 == 0 {
			h.must(m.InfSet.Insert(1))
		}
		initialInf := m.InfSet.Size()
		for row := 0; row < 2; row++ {
			f.types[row] = ColumnBoxed
			h.must(h.m.SetInt32(&f.low.Data[row], -2))
			h.must(h.m.SetInt32(&f.up.Data[row], 2))
			x := int32(world%3-1) * 3
			h.must(h.m.SetInt32(&f.x.Data[row], x))
			nc := int64(0)
			if x > 2 {
				nc = 1
			}
			if x < -2 {
				nc = -1
			}
			if !f.settings.UseBreakpointsInFeasibilitySearch {
				nc = -nc
			}
			h.must(m.UpdateInfCostForColumnTableau(uint32(row)))
			costs[row] = big.NewRat(nc, 1)
			lpSparseRationalCheck(t, h, &f.costs.Data[row], costs[row])
		}
		for k := range costs {
			want := new(big.Rat)
			if k >= 2 {
				want.Set(costs[k])
				for row := 0; row < 2; row++ {
					want.Sub(want, new(big.Rat).Mul(costs[row], coeff[row][k-2]))
				}
			}
			lpSparseRationalCheck(t, h, &m.D.Data[k], want)
			lpSparseRationalCheck(t, h, &f.costs.Data[k], costs[k])
		}
		if m.InfSet.Size() != initialInf || m.InfSet.Contains(1) != (world%3 == 0) {
			t.Fatal("cost update changed infeasibility membership")
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPRationalTableauCostSourcePrefixesAndGuards(t *testing.T) {
	for _, which := range []string{"heading", "nil", "types", "x", "upper", "low", "unknown", "nonbasic", "free", "zero", "cost", "row", "row-column", "negative-row", "debug-strategy", "debug-costs", "delta-alias", "nil-delta", "short-heading", "update-get-error", "upper-first", "free-no-x", "out-alias"} {
		t.Run(which, func(t *testing.T) {
			h := newLPPairHarness(t)
			f := lpNewRationalCoreFixture(t, h, 1, 3)
			f.settings.SimplexStrategy = StrategyTableauCosts
			v := h.pair("2", "0")
			h.must(f.a.Set(0, 1, &v.X))
			h.must(h.m.SetInt32(&v.X, 3))
			h.must(f.a.Set(0, 2, &v.X))
			m, err := NewRationalPrimalCoreSolver(f.input())
			h.must(err)
			m.UsingInfeasCosts = true
			f.settings.UseBreakpointsInFeasibilitySearch = true
			f.types[0] = ColumnLowBound
			h.must(h.m.SetInt32(&f.x.Data[0], -1))
			out := h.pair("17", "0")
			savedX, savedLow, savedUp, savedCosts := f.x.Data, f.low.Data, f.up.Data, f.costs.Data
			var want error
			switch which {
			case "heading":
				f.heading = nil
				err = m.GetInfeasibilityCostForColumn(0, &out.X)
				want = ErrBounds
			case "nil":
				err = m.GetInfeasibilityCostForColumn(0, nil)
				want = ErrBounds
			case "types":
				f.types = nil
				err = m.GetInfeasibilityCostForColumn(0, &out.X)
				want = ErrBounds
			case "x":
				f.x.Data = nil
				err = m.GetInfeasibilityCostForColumn(0, &out.X)
				want = ErrBounds
			case "upper":
				f.types[0] = ColumnBoxed
				f.up.Data = nil
				err = m.GetInfeasibilityCostForColumn(0, &out.X)
				want = ErrBounds
			case "low":
				f.low.Data = nil
				err = m.GetInfeasibilityCostForColumn(0, &out.X)
				want = ErrBounds
			case "unknown":
				f.types[0] = ColumnType(99)
				err = m.GetInfeasibilityCostForColumn(0, &out.X)
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "nonbasic":
				f.heading[0] = -1
				f.types = nil
				err = m.GetInfeasibilityCostForColumn(0, &out.X)
				lpSparseRationalCheck(t, h, &out.X, big.NewRat(0, 1))
			case "free":
				f.types[0] = ColumnFree
				err = m.GetInfeasibilityCostForColumn(0, &out.X)
				lpSparseRationalCheck(t, h, &out.X, big.NewRat(0, 1))
			case "zero":
				h.must(h.m.SetInt32(&f.costs.Data[0], -1))
				f.heading[0] = 99
				err = m.UpdateInfCostForColumnTableau(0)
			case "cost":
				f.costs.Data = nil
				err = m.UpdateInfCostForColumnTableau(0)
				want = ErrBounds
			case "row":
				f.heading[0] = 99
				err = m.UpdateInfCostForColumnTableau(0)
				want = ErrBounds
				lpSparseRationalCheck(t, h, &f.costs.Data[0], big.NewRat(-1, 1))
			case "row-column":
				h.must(m.D.Resize(2))
				err = m.UpdateInfCostForColumnTableau(0)
				want = ErrBounds
				lpSparseRationalCheck(t, h, &m.D.Data[1], big.NewRat(2, 1))
			case "negative-row":
				f.heading[0] = -1
				err = m.UpdateReducedCostForBasicColumnCostChange(&out.X, 0)
				want = ErrBounds
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "debug-strategy":
				f.settings.SimplexStrategy = StrategyTableauRows
				err = m.UpdateInfCostForColumnTableau(0)
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "debug-costs":
				m.UsingInfeasCosts = false
				err = m.UpdateInfCostForColumnTableau(0)
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "nil-delta":
				err = m.UpdateReducedCostForBasicColumnCostChange(nil, 0)
				want = ErrBounds
			case "short-heading":
				f.heading = nil
				err = m.UpdateReducedCostForBasicColumnCostChange(&out.X, 0)
				want = ErrBounds
			case "update-get-error":
				f.heading = nil
				err = m.UpdateInfCostForColumnTableau(0)
				want = ErrBounds
			case "upper-first":
				f.types[0] = ColumnBoxed
				f.low.Data = nil
				h.must(h.m.SetInt32(&f.x.Data[0], 1))
				err = m.GetInfeasibilityCostForColumn(0, &out.X)
				lpSparseRationalCheck(t, h, &out.X, big.NewRat(1, 1))
			case "free-no-x":
				f.types[0] = ColumnFree
				f.x.Data = nil
				err = m.GetInfeasibilityCostForColumn(0, &out.X)
				lpSparseRationalCheck(t, h, &out.X, big.NewRat(0, 1))
			case "out-alias":
				h.must(h.m.SetString(&f.x.Data[0], "-5/3"))
				err = m.GetInfeasibilityCostForColumn(0, &f.x.Data[0])
				lpSparseRationalCheck(t, h, &f.x.Data[0], big.NewRat(-1, 1))
			case "delta-alias":
				h.must(h.m.SetInt32(&m.D.Data[1], 2))
				err = m.UpdateReducedCostForBasicColumnCostChange(&m.D.Data[1], 0)
				lpSparseRationalCheck(t, h, &m.D.Data[1], big.NewRat(6, 1))
				lpSparseRationalCheck(t, h, &m.D.Data[2], big.NewRat(18, 1))
			}
			if !errors.Is(err, want) {
				t.Fatal(err, want)
			}
			if err != nil {
				if m.GetInfeasibilityCostForColumn(0, &out.X) != ErrExactVectorPending || m.UpdateInfCostForColumnTableau(0) != ErrExactVectorPending || m.UpdateReducedCostForBasicColumnCostChange(&out.X, 0) != ErrExactVectorPending {
					t.Fatal("stopped entry")
				}
			}
			// Restore borrowed headers before closing their owners.
			f.x.Data, f.low.Data, f.up.Data, f.costs.Data = savedX, savedLow, savedUp, savedCosts
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPRationalTableauCostPhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, direct := range []bool{false, true} {
		for _, op := range []string{"cost", "reduced", "costcmp-low", "costcmp-upper"} {
			success := false
			failures := 0
			comparisons := 0
			for ordinal := int64(0); ordinal < 240; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("direct=%t/%s/%d", direct, op, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					f := lpNewRationalCoreFixture(t, h, 1, 3)
					f.settings.SimplexStrategy = StrategyTableauCosts
					a := strings.Repeat("9876543210", 100) + "/7"
					b := strings.Repeat("1234567890", 100) + "/13"
					v := h.pair(a, b)
					h.must(f.a.Set(0, 1, &v.X))
					h.must(f.a.Set(0, 2, &v.Y))
					m, err := NewRationalPrimalCoreSolver(f.input())
					h.must(err)
					m.UsingInfeasCosts = true
					f.types[0] = ColumnLowBound
					h.must(h.m.Copy(&f.low.Data[0], &v.X))
					h.must(h.m.Copy(&f.costs.Data[0], &v.X))
					if op == "costcmp-low" {
						h.must(h.m.Copy(&f.x.Data[0], &v.Y))
					}
					if op == "costcmp-upper" {
						f.types[0] = ColumnUpperBound
						h.must(h.m.Copy(&f.x.Data[0], &v.X))
						h.must(h.m.Copy(&f.up.Data[0], &v.Y))
					}
					h.budget.FailAfter(ordinal)
					if op != "reduced" {
						err = m.UpdateInfCostForColumnTableau(0)
					} else {
						err = m.UpdateReducedCostForBasicColumnCostChange(&v.X, 0)
					}
					h.budget.FailAfter(-1)
					if err == nil {
						success = true
						done = true
						delta := mustLPRat(t, a)
						if op != "reduced" {
							nc := int64(1)
							if op == "costcmp-upper" {
								nc = -1
							}
							delta.Sub(delta, big.NewRat(nc, 1))
							lpSparseRationalCheck(t, h, &f.costs.Data[0], big.NewRat(nc, 1))
						}
						lpSparseRationalCheck(t, h, &m.D.Data[1], new(big.Rat).Mul(delta, mustLPRat(t, a)))
						lpSparseRationalCheck(t, h, &m.D.Data[2], new(big.Rat).Mul(delta, mustLPRat(t, b)))
					} else {
						failures++
						if m.RationalCoreSolverBase.calls.comparison != 0 {
							comparisons++
							if m.calls.pending() || m.work != nil {
								t.Fatal("primal adopted base comparison")
							}
							lpSparseRationalCheck(t, h, &f.costs.Data[0], mustLPRat(t, a))
						}
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						if m.UpdateInfCostForColumnTableau(0) != ErrExactVectorPending {
							t.Fatal("stopped")
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
			if (op == "costcmp-low" || op == "costcmp-upper") && comparisons == 0 {
				t.Fatal("no actual base comparison failure", direct, op)
			}
			if !success || failures == 0 {
				t.Fatal("incomplete campaign", direct, op, success, failures)
			}
		}
	}
}

func TestLPRationalTableauCostLeavesOlderComparisonOwned(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	for _, op := range []string{"get", "update"} {
		matched := false
		for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", op, ordinal), func(t *testing.T) {
				h := newLPPairHarness(t)
				f := lpNewRationalCoreFixture(t, h, 1, 1)
				f.settings.SimplexStrategy = StrategyTableauCosts
				m, err := NewRationalPrimalCoreSolver(f.input())
				h.must(err)
				m.UsingInfeasCosts = true
				v := h.pair(strings.Repeat("9876543210", 100)+"/7", strings.Repeat("1234567890", 100)+"/13")
				f.types[0] = ColumnLowBound
				h.must(h.m.Copy(&f.low.Data[0], &v.X))
				h.must(h.m.Copy(&f.x.Data[0], &v.Y))
				out := h.pair("17", "0")
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
					if op == "get" {
						err = m.GetInfeasibilityCostForColumn(0, &out.X)
					} else {
						err = m.UpdateInfCostForColumnTableau(0)
					}
					if err == nil || m.calls.pending() || m.RationalCoreSolverBase.calls.pending() || m.work != nil {
						t.Fatal("adopted older comparison", err)
					}
					lpSparseRationalCheck(t, h, &out.X, big.NewRat(17, 1))
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
