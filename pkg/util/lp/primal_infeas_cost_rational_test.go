package lp

import (
	"fmt"
	"math/big"
	"reflect"
	"strings"
	"testing"
)

func TestLPRationalPrimalInfeasCostUnknownAndMembershipPrefixes(t *testing.T) {
	for _, op := range []string{"unknown", "unknown-breakpoints", "unknown-zero", "nonbasic", "membership", "after-noop", "after-index", "all-prefix", "changed-prefix"} {
		h := newLPPairHarness(t)
		f := lpNewRationalCoreFixture(t, h, 1, 2)
		h.must(f.a.Set(0, 0, &h.pair("1", "0").X))
		m, e := NewRationalPrimalCoreSolver(f.input())
		h.must(e)
		f.types[0] = ColumnType(99)
		h.must(h.m.SetInt32(&f.costs.Data[0], -17))
		h.must(h.m.SetInt32(&f.costs.Data[1], 31))
		h.must(m.InfSet.Insert(1))
		want := error(nil)
		switch op {
		case "unknown-breakpoints":
			m.Settings.UseBreakpointsInFeasibilitySearch = true
		case "unknown-zero":
			h.must(h.m.SetInt32(&f.costs.Data[0], 0))
		case "nonbasic":
			f.types = nil
			h.must(f.x.resizeData(0))
		case "membership":
			f.types[0] = ColumnLowBound
			h.must(f.low.copySlot(&f.low.Data[0], &h.pair("1", "0").X))
			h.must(m.InfSet.Clear())
			h.must(m.InfSet.Resize(0))
			want = ErrBounds
		case "after-noop":
			m.Ed.Index = []uint32{99}
		case "after-index":
			m.UsingInfeasCosts = true
			want = ErrBounds
		case "all-prefix":
			want = nil
			if sourceDebugEnabled {
				want = ErrAssertion
			}
		case "changed-prefix":
			h.must(m.Ed.SetValue(&h.pair("1", "0").X, 0))
			m.Ed.Index = append(m.Ed.Index, 99)
			f.types[0] = ColumnFree
			want = ErrBounds
		}
		var err error
		switch op {
		case "after-noop", "after-index":
			err = m.InitInfeasibilityAfterUpdateXIfInf(99)
		case "all-prefix":
			err = m.InitInfeasibilityCosts()
			lpSparseRationalCheck(t, h, &f.costs.Data[1], big.NewRat(0, 1))
			if sourceDebugEnabled && m.UsingInfeasCosts {
				t.Fatal("published using flag before failure")
			}
		case "changed-prefix":
			err = m.InitInfeasibilityCostsForChangedBasisOnly()
			lpSparseRationalCheck(t, h, &f.costs.Data[0], big.NewRat(0, 1))
			if m.UsingInfeasCosts {
				t.Fatal("published flag before invalid indexed row")
			}
		case "nonbasic":
			err = m.InitInfeasibilityCostForColumn(1)
		default:
			err = m.InitInfeasibilityCostForColumn(0)
		}
		unknown := op == "unknown" || op == "unknown-breakpoints" || op == "unknown-zero"
		if unknown && sourceDebugEnabled {
			want = ErrAssertion
		}
		if err != want {
			t.Fatal(op, err, want)
		}
		if unknown {
			cost := int64(17)
			member := true
			if sourceDebugEnabled {
				cost = -17
				member = false
			} else if op == "unknown-breakpoints" {
				cost = -17
			} else if op == "unknown-zero" {
				cost = 0
				member = false
			}
			if sourceDebugEnabled && op == "unknown-zero" {
				cost = 0
			}
			lpSparseRationalCheck(t, h, &f.costs.Data[0], big.NewRat(cost, 1))
			if m.InfSet.Contains(0) != member {
				t.Fatal("unknown membership", op)
			}
		}
		if op == "membership" {
			lpSparseRationalCheck(t, h, &f.costs.Data[0], big.NewRat(-1, 1))
		}
		if op == "nonbasic" {
			lpSparseRationalCheck(t, h, &f.costs.Data[1], big.NewRat(0, 1))
			if m.InfSet.Contains(1) {
				t.Fatal("nonbasic not erased")
			}
		}
		if op == "after-noop" {
			lpSparseRationalCheck(t, h, &f.costs.Data[1], big.NewRat(31, 1))
		}
		if err != nil {
			if e = m.InitInfeasibilityCosts(); e != ErrExactVectorPending {
				t.Fatal(e)
			}
			if e = m.InitInfeasibilityCostsForChangedBasisOnly(); e != ErrExactVectorPending {
				t.Fatal(e)
			}
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPRationalPrimalInfeasCostPhysicalOwnership(t *testing.T) {
	ops := []string{"column", "all", "changed", "after"}
	if !sourceDebugEnabled {
		ops = append(ops, "unknown-minimum")
	}
	for _, op := range ops {
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
					for i := uint32(0); i < 2; i++ {
						h.must(f.a.Set(i, i, &h.pair("1", "0").X))
					}
					m, e := NewRationalPrimalCoreSolver(f.input())
					h.must(e)
					huge := strings.Repeat("1234567890", 12)
					for j := 0; j < 4; j++ {
						h.must(h.m.SetInt32(&f.costs.Data[j], 17))
						f.types[j] = ColumnBoxed
						h.must(f.x.copySlot(&f.x.Data[j], &h.pair(huge+"9/7", huge+"7/11").X))
						h.must(f.low.copySlot(&f.low.Data[j], &h.pair(huge+"1/13", huge+"3/17").X))
						h.must(f.up.copySlot(&f.up.Data[j], &h.pair(huge+"3/19", huge+"5/23").X))
					}
					h.must(m.Ed.SetValue(&h.pair("0", "0").X, 1))
					h.must(m.Ed.SetValue(&h.pair("1", "0").X, 0))
					if op == "unknown-minimum" {
						f.types[0] = ColumnType(99)
						h.must(h.m.SetInt32(&f.costs.Data[0], -2147483648))
					}
					if op == "after" {
						m.UsingInfeasCosts = true
						h.must(m.InfSet.Insert(3))
					}
					h.budget.FailAfter(ordinal)
					var err error
					switch op {
					case "column", "unknown-minimum":
						err = m.InitInfeasibilityCostForColumn(0)
					case "all":
						err = m.InitInfeasibilityCosts()
					case "changed":
						err = m.InitInfeasibilityCostsForChangedBasisOnly()
					case "after":
						err = m.InitInfeasibilityAfterUpdateXIfInf(3)
					}
					h.budget.FailAfter(-1)
					if err == nil {
						success, done = true, true
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						if m.calls.pending() && m.RationalCoreSolverBase.calls.pending() {
							t.Fatal("adopted base comparison")
						}
						if op == "unknown-minimum" && !m.InfSet.Contains(0) {
							t.Fatal("lost membership before negation refusal")
						}
						if e = m.InitInfeasibilityCostForColumn(0); e != ErrExactVectorPending {
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
			if !success || failures == 0 {
				t.Fatal(op, direct, success, failures)
			}
		}
	}
}

func TestLPRationalPrimalInfeasCostLeavesOlderComparisonOwned(t *testing.T) {
	for _, op := range []string{"column", "all", "changed", "after"} {
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
				h.must(h.m.SetInt32(&f.costs.Data[0], 17))
				h.must(m.Ed.SetValue(&v.X, 0))
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
					switch op {
					case "column":
						err = m.InitInfeasibilityCostForColumn(0)
					case "all":
						err = m.InitInfeasibilityCosts()
					case "changed":
						err = m.InitInfeasibilityCostsForChangedBasisOnly()
					case "after":
						err = m.InitInfeasibilityAfterUpdateXIfInf(0)
					}
					if err == nil || m.calls.pending() || m.RationalCoreSolverBase.calls.pending() || m.work != nil {
						t.Fatal("adopted older comparison", err)
					}
					lpSparseRationalCheck(t, h, &f.costs.Data[0], big.NewRat(17, 1))
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

func TestLPRationalPrimalInfeasCostMetadataBoundaries(t *testing.T) {
	for _, phase := range []string{"heading", "type", "x", "upper", "lower", "cost", "nonbasic-cost", "all-x", "all-type", "stopped-after"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			f := lpNewRationalCoreFixture(t, h, 1, 1)
			m, err := NewRationalPrimalCoreSolver(f.input())
			h.must(err)
			f.types[0] = ColumnBoxed
			h.must(h.m.SetInt32(&f.costs.Data[0], 17))
			switch phase {
			case "heading":
				f.heading = nil
			case "type", "all-type":
				f.types = nil
			case "x", "all-x":
				h.must(f.x.resizeData(0))
			case "upper":
				h.must(f.up.resizeData(0))
			case "lower":
				h.must(f.low.resizeData(0))
			case "cost", "nonbasic-cost":
				h.must(f.costs.resizeData(0))
				if phase == "nonbasic-cost" {
					f.heading[0] = -1
				}
			case "stopped-after":
				if err = m.InitInfeasibilityCostForColumn(99); err != ErrBounds {
					t.Fatal(err)
				}
			}
			want := ErrBounds
			switch phase {
			case "all-x", "all-type":
				err = m.InitInfeasibilityCosts()
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "stopped-after":
				err = m.InitInfeasibilityAfterUpdateXIfInf(99)
				want = ErrExactVectorPending
			default:
				err = m.InitInfeasibilityCostForColumn(0)
			}
			if err != want {
				t.Fatal(err, want)
			}
			if m.UsingInfeasCosts || !m.InfSet.Empty() {
				t.Fatal("premature cost state publication")
			}
			if len(f.costs.Data) != 0 {
				lpSparseRationalCheck(t, h, &f.costs.Data[0], big.NewRat(17, 1))
			}
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPRationalPrimalInfeasCostIndependentStateOracle(t *testing.T) {
	for _, op := range []string{"column", "all", "changed", "after"} {
		for world := 0; world < 160; world++ {
			h := newLPPairHarness(t)
			f := lpNewRationalCoreFixture(t, h, 3, 6)
			f.basis = []uint32{2, 0, 4}
			for i, j := range f.basis {
				h.must(f.a.Set(uint32(i), j, &h.pair("1", "0").X))
			}
			m, e := NewRationalPrimalCoreSolver(f.input())
			h.must(e)
			m.UsingInfeasCosts = world%2 == 0
			m.Settings.UseBreakpointsInFeasibilitySearch = world%3 == 0
			h.must(m.InfSet.Resize(6))
			var members []uint32
			for _, j := range []uint32{5, 2, 0, 3, 4, 1} {
				if (world+int(j))%2 == 0 {
					h.must(m.InfSet.Insert(j))
					members = append(members, j)
				}
			}
			raw := make([]int64, 6)
			want := make([]int64, 6)
			for j := 0; j < 6; j++ {
				kind := ColumnType((world + j) % 5)
				f.types[j] = kind
				lowx, upx := big.NewRat(-2, 3), big.NewRat(5, 3)
				if kind == ColumnFixed {
					upx.Set(lowx)
				} else if kind == ColumnBoxed && world%7 == 0 {
					lowx.SetInt64(2)
					upx.SetInt64(-2)
				}
				xx := big.NewRat(int64((world+j)%13-6), 3)
				if world%5 == 0 {
					xx.Set(lowx)
				}
				if world%7 == 0 {
					xx.Add(lowx, new(big.Rat).SetFrac(big.NewInt(-1), new(big.Int).Exp(big.NewInt(10), big.NewInt(100), nil)))
				}
				h.must(h.m.SetString(&f.x.Data[j], xx.RatString()))
				h.must(h.m.SetString(&f.low.Data[j], lowx.RatString()))
				h.must(h.m.SetString(&f.up.Data[j], upx.RatString()))
				want[j] = int64(j + 7)
				h.must(h.m.SetInt64(&f.costs.Data[j], want[j]))
				h.must(h.m.SetInt32(&m.D.Data[j], int32(31+j)))
				basic := j == 0 || j == 2 || j == 4
				if basic {
					if (kind == ColumnFixed || kind == ColumnBoxed || kind == ColumnUpperBound) && xx.Cmp(upx) > 0 {
						raw[j] = 1
					} else if (kind == ColumnFixed || kind == ColumnBoxed || kind == ColumnLowBound) && xx.Cmp(lowx) < 0 {
						raw[j] = -1
					}
				}
			}
			// Cost initialization visits every indexed row, including a stored zero Ed.
			h.must(m.Ed.SetValue(&h.pair("0", "0").X, 2))
			h.must(m.Ed.SetValue(&h.pair("1", "0").X, 0))
			erase := func(j uint32) {
				for k, v := range members {
					if v == j {
						members[k] = members[len(members)-1]
						members = members[:len(members)-1]
						return
					}
				}
			}
			insert := func(j uint32) {
				for _, v := range members {
					if v == j {
						return
					}
				}
				members = append(members, j)
			}
			init := func(j uint32) {
				want[j] = raw[j]
				if raw[j] == 0 {
					erase(j)
				} else {
					insert(j)
				}
				if !m.Settings.UseBreakpointsInFeasibilitySearch {
					want[j] = -want[j]
				}
			}
			using := m.UsingInfeasCosts
			switch op {
			case "column":
				init(uint32(world % 6))
				e = m.InitInfeasibilityCostForColumn(uint32(world % 6))
			case "all":
				for j := 5; j >= 0; j-- {
					init(uint32(j))
				}
				using = true
				e = m.InitInfeasibilityCosts()
			case "changed":
				init(4)
				init(2)
				using = true
				e = m.InitInfeasibilityCostsForChangedBasisOnly()
			case "after":
				if using {
					init(4)
					init(2)
					want[1] = 0
					erase(1)
				}
				e = m.InitInfeasibilityAfterUpdateXIfInf(1)
			}
			h.must(e)
			if m.UsingInfeasCosts != using || !reflect.DeepEqual(m.InfSet.Indices(), members) {
				t.Fatal(op, world, m.UsingInfeasCosts, using, m.InfSet.Indices(), members)
			}
			for j := 0; j < 6; j++ {
				lpSparseRationalCheck(t, h, &f.costs.Data[j], big.NewRat(want[j], 1))
				lpSparseRationalCheck(t, h, &m.D.Data[j], big.NewRat(int64(31+j), 1))
			}
			for _, strategy := range []SimplexStrategy{StrategyTableauRows, StrategyLU, StrategyTableauCosts, StrategyUndecided} {
				m.Settings.SimplexStrategy = strategy
				expected := strategy != StrategyTableauRows && ((len(members) == 0) == using)
				if m.NeedToSwitchCosts() != expected {
					t.Fatal("switch predicate", op, world, strategy)
				}
			}
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		}
	}
}
