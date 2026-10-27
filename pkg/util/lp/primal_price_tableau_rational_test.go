package lp

import (
	"fmt"
	"math/rand/v2"
	"reflect"
	"strings"
	"testing"

	z3rand "github.com/glycerine/g3/pkg/util/random"
)

func TestLPRationalTableauPricingIndependentSelection(t *testing.T) {
	rng := rand.New(rand.NewPCG(41279, 6713))
	for world := 0; world < 320; world++ {
		h := newLPPairHarness(t)
		f := lpNewRationalCoreFixture(t, h, 8, 20)
		f.settings.SimplexStrategy = StrategyTableauCosts
		if world%2 == 0 {
			f.settings.SimplexStrategy = StrategyTableauRows
		}
		for i := uint32(0); i < 8; i++ {
			h.must(f.a.Set(i, i, lpScalarBoundValue(h, "1")))
		}
		counts, signs := make([]uint32, 20), make([]int32, 20)
		for j := uint32(8); j < 20; j++ {
			counts[j] = uint32(rng.IntN(9))
			signs[j] = int32(rng.IntN(3) - 1)
			for i := uint32(0); i < counts[j]; i++ {
				h.must(f.a.Set(i, j, lpScalarBoundValue(h, "2/7")))
			}
		}
		m, e := NewRationalPrimalCoreSolver(f.input())
		h.must(e)
		h.must(h.m.SetInt32(&m.EpsilonOfReducedCost, 0))
		for j := 8; j < 20; j++ {
			h.must(h.m.SetInt32(&m.D.Data[j], signs[j]))
		}
		// Opposite cached counts prove pricing reads the actual stored columns.
		m.ColumnsNZ = make([]uint32, 20)
		for j := range counts {
			m.ColumnsNZ[j] = 99 - counts[j]
		}
		m.NonBasisList = nil
		for _, j := range rng.Perm(12) {
			m.NonBasisList = append(m.NonBasisList, uint32(j+8))
		}
		m.BasisSortCounter = 7
		m.SignOfEnteringDelta = 91
		m.UsingInfeasCosts = world/3%2 == 0
		f.settings.UseBreakpointsInFeasibilitySearch = world/7%2 == 0
		seed := uint32(world + 129)
		f.settings.RandomSeed(seed)
		oracle := z3rand.New(seed)
		remaining := uint32(12)
		if world%3 == 0 {
			m.Status = StatusTentativeUnbounded
		} else {
			remaining = oracle.Next() % remaining
			if remaining < 1 {
				remaining = 1
			}
		}
		before := append([]uint32(nil), m.NonBasisList...)
		best, position := uint32(9), -1
		for i, j := range before {
			if remaining == 0 {
				break
			}
			if signs[j] == 0 {
				continue
			}
			if counts[j] < best {
				best, position = counts[j], i
				remaining--
			} else if counts[j] == best && oracle.Next()%2 == 0 {
				position = i
			}
		}
		want := int32(-1)
		wantSign := int32(91)
		expected := append([]uint32(nil), before...)
		if position >= 0 {
			j := before[position]
			want = int32(j)
			wantSign = signs[j]
			if m.UsingInfeasCosts && f.settings.UseBreakpointsInFeasibilitySearch {
				wantSign = -wantSign
			}
			expected = append(append(expected[:position:position], expected[position+1:]...), j)
		}
		got, e := m.ChooseEnteringColumnTableau()
		h.must(e)
		if got != want || m.SignOfEnteringDelta != wantSign || !reflect.DeepEqual(m.NonBasisList, expected) || m.BasisSortCounter != 6 {
			t.Fatal(world, got, want, m.NonBasisList, expected)
		}
		if f.settings.RandomNext() != oracle.Next() {
			t.Fatal("RNG", world)
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPRationalTableauPricingSortAndReachedPrefixes(t *testing.T) {
	for _, phase := range []string{"sort", "empty-budget", "rounded-zero-budget", "no-beneficial", "column-extent", "sort-extent", "heading-extent", "eligibility", "budget-error"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			f := lpNewRationalCoreFixture(t, h, 3, 6)
			f.settings.SimplexStrategy = StrategyTableauCosts
			for i := uint32(0); i < 3; i++ {
				h.must(f.a.Set(i, i, lpScalarBoundValue(h, "1")))
			}
			for j := uint32(3); j < 6; j++ {
				for i := uint32(0); i < 6-j; i++ {
					h.must(f.a.Set(i, j, lpScalarBoundValue(h, "1")))
				}
			}
			m, e := NewRationalPrimalCoreSolver(f.input())
			h.must(e)
			for j := 3; j < 6; j++ {
				h.must(h.m.SetInt32(&m.D.Data[j], 1))
			}
			m.Status = StatusTentativeUnbounded
			m.NonBasisList = []uint32{3, 4, 5}
			saved := f.a.Columns
			wantErr := error(nil)
			want := int32(5)
			switch phase {
			case "empty-budget":
				f.nb = nil
				f.a.Columns = nil
				want = -1
			case "rounded-zero-budget":
				f.nb = make([]uint32, 301)
				f.settings.PercentOfEnteringToCheck = 0
				m.Status = StatusUnknown
				f.a.Columns = nil
				want = -1
			case "no-beneficial":
				for j := 3; j < 6; j++ {
					h.must(h.m.SetInt32(&m.D.Data[j], 0))
				}
				want = -1
			case "column-extent":
				m.BasisSortCounter = 7
				f.a.Columns = f.a.Columns[:3]
				wantErr = ErrBounds
			case "sort-extent":
				f.a.Columns = f.a.Columns[:3]
				wantErr = ErrBounds
			case "heading-extent":
				f.heading = f.heading[:3]
				wantErr = ErrBounds
			case "budget-error":
				m.Settings = nil
				m.Status = StatusUnknown
				wantErr = ErrBounds
			case "eligibility":
				m.BasisSortCounter = 7
				f.types = nil
				wantErr = ErrBounds
			}
			seed := uint32(19)
			f.settings.RandomSeed(seed)
			oracle := z3rand.New(seed)
			got, e := m.ChooseEnteringColumnTableau()
			if e != wantErr {
				t.Fatal(phase, e, wantErr)
			}
			if wantErr == nil {
				if got != want {
					t.Fatal(got, want)
				}
				if phase == "empty-budget" || phase == "rounded-zero-budget" {
					if m.BasisSortCounter != 0 || !reflect.DeepEqual(m.NonBasisList, []uint32{3, 4, 5}) || f.settings.RandomNext() != oracle.Next() {
						t.Fatal("early budget side effect")
					}
				} else if m.BasisSortCounter != 20 || !reflect.DeepEqual(f.nb, []uint32{5, 4, 3}) {
					t.Fatal("sort publication")
				}
			} else {
				if _, e = m.ChooseEnteringColumnTableau(); e != ErrExactVectorPending {
					t.Fatal("retry", e)
				}
				if (phase == "column-extent" || phase == "eligibility") && m.BasisSortCounter != 6 {
					t.Fatal("counter prefix")
				}
				if (phase == "sort-extent" || phase == "heading-extent") && m.BasisSortCounter != 0 {
					t.Fatal("sort error prefix")
				}
			}
			f.a.Columns = saved
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPRationalTableauPricingPhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, profile := range []string{"bound", "epsilon", "breakpoint"} {
		for _, direct := range []bool{false, true} {
			success, failures := false, 0
			for ordinal := int64(0); ordinal < 200; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("%s/%t/%d", profile, direct, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					f := lpNewRationalCoreFixture(t, h, 1, 2)
					f.settings.SimplexStrategy = StrategyTableauCosts
					h.must(f.a.Set(0, 0, lpScalarBoundValue(h, "1")))
					m, e := NewRationalPrimalCoreSolver(f.input())
					h.must(e)
					huge := strings.Repeat("1234567890", 100)
					f.types[1] = ColumnUpperBound
					h.must(h.m.SetString(&m.D.Data[1], huge+"1/7"))
					h.must(h.m.Copy(&f.x.Data[1], lpScalarBoundValue(h, huge+"1/7")))
					h.must(h.m.Copy(&f.up.Data[1], lpScalarBoundValue(h, huge+"3/11")))
					if profile != "bound" {
						m.UsingInfeasCosts = true
						f.settings.UseBreakpointsInFeasibilitySearch = true
						h.must(h.m.SetString(&m.EpsilonOfReducedCost, huge+"3/11"))
						if profile == "epsilon" {
							f.types[1] = ColumnFree
						} else {
							f.types[1] = ColumnLowBound
							h.must(h.m.Copy(&f.x.Data[1], lpScalarBoundValue(h, "0")))
							h.must(h.m.Neg(&m.D.Data[1]))
						}
					}
					m.BasisSortCounter = 1
					m.NonBasisList = []uint32{1}
					h.budget.FailAfter(ordinal)
					_, err := m.ChooseEnteringColumnTableau()
					h.budget.FailAfter(-1)
					if err == nil {
						success, done = true, true
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						if m.calls.pending() && m.RationalCoreSolverBase.calls.pending() {
							t.Fatal("adopted child capture")
						}
						if _, e = m.ChooseEnteringColumnTableau(); e != ErrExactVectorPending {
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
				t.Fatal("campaign", direct, failures)
			}
		}
	}

}

func TestLPRationalTableauPricingOlderComparisonOwner(t *testing.T) {
	for _, op := range []string{"bound", "epsilon", "breakpoint"} {
		matched := false
		for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", op, ordinal), func(t *testing.T) {
				h := newLPPairHarness(t)
				f, m := lpRationalTableauSelectionFixture(t, h, 0, 1)
				v := h.pair(strings.Repeat("9876543210", 100)+"/7", strings.Repeat("1234567890", 100)+"/13")
				f.types[0] = ColumnUpperBound
				h.must(h.m.Copy(&f.up.Data[0], &v.X))
				h.must(h.m.Copy(&f.x.Data[0], &v.Y))
				h.must(h.m.SetInt32(&m.D.Data[0], 1))
				m.BasisSortCounter = 1
				m.ColumnsNZ = []uint32{0}
				m.NonBasisList = []uint32{0}
				if op == "epsilon" {
					f.types[0] = ColumnFree
					m.UsingInfeasCosts = true
					f.settings.UseBreakpointsInFeasibilitySearch = true
					h.must(h.m.SetInt32(&m.D.Data[0], -1))
				}
				if op == "breakpoint" {
					f.types[0] = ColumnLowBound
					h.must(h.m.SetInt32(&f.low.Data[0], 0))
					h.must(h.m.SetInt32(&f.x.Data[0], 0))
					m.UsingInfeasCosts = true
					f.settings.UseBreakpointsInFeasibilitySearch = true
					h.must(h.m.SetInt32(&m.D.Data[0], -1))
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
					_, err = m.ChooseEnteringColumnTableau()
					if err == nil || m.calls.pending() || m.RationalCoreSolverBase.calls.pending() {
						t.Fatal("adopted older comparison", err)
					}
				}
				h.must(m.Close())
				f.close()
				if older.Active && h.m.ComparisonInvocationState() != older {
					t.Fatal("consumed caller comparison")
				}
				h.must(caller.cleanupRaw(h.m))
				h.must(caller.cleanupRoots(h.m))
				lpExactVectorNamedCensus(t, h, nil)
			})
		}
		if !matched {
			t.Fatal("no actual caller comparison", op)
		}
	}
}
