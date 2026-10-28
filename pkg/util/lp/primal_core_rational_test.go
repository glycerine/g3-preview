package lp

import (
	"fmt"
	"math"
	"math/big"
	"reflect"
	"strings"
	"testing"
)

func TestLPRationalPrimalConstructionAndMembershipOracle(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	for world := 0; world < 160; world++ {
		h := newLPPairHarness(t)
		f := lpNewRationalCoreFixture(t, h, 3, 6)
		f.basis = []uint32{4, 0, 2}
		f.settings.HarrisFeasibilityTolerance = math.NaN()
		if world%2 == 0 {
			f.settings.SimplexStrategy = StrategyLU
		}
		for i, j := range f.basis {
			h.must(f.a.Set(uint32(i), j, &h.pair("1", "0").X))
		}
		m, e := NewRationalPrimalCoreSolver(f.input())
		h.must(e)
		lpSparseRationalCheck(t, h, &m.EpsilonOfReducedCost, big.NewRat(1, 10000000))
		lpSparseRationalCheck(t, h, &m.ConvertedHarrisEpsilon, new(big.Rat))
		if m.Status != StatusUnknown || m.BlandModeThreshold != 1000 || len(m.Beta.Data) != 3 || len(m.CostsBackup.Data) != 0 || m.BreakpointQueue.Size() != 0 || (m.Factorization != nil) != (world%2 == 0) {
			t.Fatal("constructor members", world)
		}
		wantMembers := []uint32{}
		for j := range f.x.Data {
			kind := ColumnType((world + j) % 5)
			f.types[j] = kind
			lo, up := big.NewRat(-2, 3), big.NewRat(5, 7)
			if kind == ColumnFixed {
				up.Set(lo)
			}
			if kind == ColumnBoxed && world%7 == 0 {
				lo.SetInt64(2)
				up.SetInt64(-2)
			}
			x := big.NewRat(int64((world+j)%9-4), 3)
			if world%4 == 0 {
				x.Set(lo)
			}
			if world%4 == 1 {
				x.Set(up)
			}
			for k, v := range []*big.Rat{x, lo, up} {
				out := []*IndexedRationalVector{f.x, f.low, f.up}[k]
				h.must(h.m.SetString(&out.Data[j], v.RatString()))
			}
			below := kind != ColumnFree && kind != ColumnUpperBound && x.Cmp(lo) < 0
			above := kind != ColumnFree && kind != ColumnLowBound && x.Cmp(up) > 0
			want := int32(0)
			// This query checks the lower side first, unlike cost initialization.
			if below {
				want = -1
			} else if above {
				want = 1
			}
			got, e := m.GetColumnOutOfBoundsDeltaSign(uint32(j))
			h.must(e)
			if got != want {
				t.Fatal(world, j, got, want)
			}
			if j%2 == 0 && (below || above) {
				wantMembers = append(wantMembers, uint32(j))
			}
		}
		h.must(m.InfSet.Insert(5))
		h.must(m.InitInfSet())
		if !reflect.DeepEqual(m.InfSet.index, wantMembers) {
			t.Fatal(world, m.InfSet.index, wantMembers)
		}
		for _, strategy := range []SimplexStrategy{StrategyLU, StrategyTableauRows, StrategyTableauCosts} {
			m.Settings.SimplexStrategy = strategy
			for _, using := range []bool{false, true} {
				m.UsingInfeasCosts = using
				want := strategy != StrategyTableauRows && ((len(wantMembers) == 0) == using)
				if m.NeedToSwitchCosts() != want {
					t.Fatal("cost switch predicate", world, strategy, using)
				}
			}
		}
		h.must(m.Close())
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPRationalPrimalConstructionAndBoundsPhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, direct := range []bool{false, true} {
		for _, op := range []string{"constructor", "init-inf", "delta"} {
			done, failures := false, 0
			for ordinal := int64(0); ordinal < 800 && !done; ordinal++ {
				t.Run(fmt.Sprintf("%t/%s/%d", direct, op, ordinal), func(t *testing.T) {
					h := newLPPairHarness(t)
					if direct {
						h = lpNewDirectSparseHarness(t)
					}
					f := lpNewRationalCoreFixture(t, h, 2, 3)
					f.settings.SimplexStrategy = StrategyLU
					v := h.pair(strings.Repeat("9876543210", 100)+"/7", strings.Repeat("1234567890", 100)+"/13")
					for i := uint32(0); i < 2; i++ {
						h.must(f.a.Set(i, i, &v.X))
						f.types[i] = ColumnLowBound
						h.must(h.m.Copy(&f.low.Data[i], &v.X))
						h.must(h.m.Copy(&f.x.Data[i], &v.Y))
					}
					var m *RationalPrimalCoreSolver
					var err error
					if op == "constructor" {
						h.budget.FailAfter(ordinal)
						m, err = NewRationalPrimalCoreSolver(f.input())
					} else {
						m, err = NewRationalPrimalCoreSolver(f.input())
						h.must(err)
						h.budget.FailAfter(ordinal)
						if op == "init-inf" {
							err = m.InitInfSet()
						} else {
							_, err = m.GetColumnOutOfBoundsDeltaSign(0)
						}
					}
					h.budget.FailAfter(-1)
					if m == nil {
						t.Fatal("missing partial owner", err)
					}
					if err == nil {
						done = true
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						if m.calls.pending() && m.RationalCoreSolverBase.calls.pending() {
							t.Fatal("adopted base arithmetic")
						}
						if e := m.InitInfSet(); e != ErrExactVectorPending {
							t.Fatal("stopped mutation", e)
						}
					}
					h.must(m.Close())
					f.close()
					lpExactVectorNamedCensus(t, h, nil)
				})
			}
			if !done || failures == 0 {
				t.Fatal(direct, op, done, failures)
			}
		}
	}
}

func TestLPRationalPrimalConstructionAndReachedGuards(t *testing.T) {
	if m, e := NewRationalPrimalCoreSolver(RationalCoreInputs{}); m != nil || e != ErrBounds {
		t.Fatal("missing borrowed inputs", m, e)
	}
	for _, phase := range []string{"empty", "boxed-constructor", "inf-heading", "inf-membership", "inf-prefix", "delta-type", "delta-x", "delta-low", "delta-up", "delta-unknown", "late-settings", "nonbasic-no-settings"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			rows, cols := uint32(2), uint32(3)
			if phase == "empty" {
				rows, cols = 0, 0
			}
			f := lpNewRationalCoreFixture(t, h, rows, cols)
			if phase == "boxed-constructor" {
				f.types[0] = ColumnBoxed
				h.must(h.m.SetInt32(&f.low.Data[0], 1))
			}
			m, err := NewRationalPrimalCoreSolver(f.input())
			if phase == "boxed-constructor" && sourceDebugEnabled {
				if err != ErrAssertion || m == nil || m.Beta != nil || m.CostsBackup != nil {
					t.Fatal("base assertion prefix", err)
				}
			} else {
				h.must(err)
				if cols > 0 {
					f.types[0] = ColumnLowBound
					h.must(h.m.SetInt32(&f.x.Data[0], -1))
				}
				want := error(nil)
				switch phase {
				case "empty", "boxed-constructor":
					err = m.InitInfSet()
				case "inf-heading":
					f.heading = nil
					h.must(m.InfSet.Insert(2))
					err = m.InitInfSet()
					want = ErrBounds
					if !m.InfSet.Empty() {
						t.Fatal("clear before heading access")
					}
				case "inf-membership":
					h.must(m.InfSet.Resize(0))
					err = m.InitInfSet()
					want = ErrBounds
				case "inf-prefix":
					f.types[1] = ColumnLowBound
					h.must(f.low.resizeData(1))
					err = m.InitInfSet()
					want = ErrBounds
					if !m.InfSet.Contains(0) || m.InfSet.Size() != 1 {
						t.Fatal("ascending member prefix")
					}
				case "delta-type":
					f.types = nil
					_, err = m.GetColumnOutOfBoundsDeltaSign(0)
					want = ErrBounds
				case "delta-x":
					h.must(f.x.resizeData(0))
					_, err = m.GetColumnOutOfBoundsDeltaSign(0)
					want = ErrBounds
				case "delta-low":
					h.must(f.low.resizeData(0))
					_, err = m.GetColumnOutOfBoundsDeltaSign(0)
					want = ErrBounds
				case "delta-up":
					f.types[0] = ColumnUpperBound
					h.must(f.up.resizeData(0))
					_, err = m.GetColumnOutOfBoundsDeltaSign(0)
					want = ErrBounds
				case "delta-unknown":
					f.types[0] = ColumnType(99)
					var got int32
					got, err = m.GetColumnOutOfBoundsDeltaSign(0)
					if got != 0 {
						t.Fatal(got)
					}
					if sourceDebugEnabled {
						want = ErrAssertion
					}
				case "late-settings":
					m.Settings = nil
					err = m.InitInfeasibilityCostForColumn(0)
					want = ErrBounds
					lpSparseRationalCheck(t, h, &f.costs.Data[0], big.NewRat(-1, 1))
					if !m.InfSet.Contains(0) {
						t.Fatal("settings read before membership publication")
					}
				case "nonbasic-no-settings":
					m.Settings = nil
					f.types = nil
					err = m.InitInfeasibilityCostForColumn(2)
				}
				if err != want {
					t.Fatal(err, want)
				}
			}
			h.must(m.Close())
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
			if e := m.InitInfSet(); e != ErrExactVectorPending {
				t.Fatal(e)
			}
			if _, e := m.GetColumnOutOfBoundsDeltaSign(0); e != ErrExactVectorPending {
				t.Fatal(e)
			}
		})
	}
}
