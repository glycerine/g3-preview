// Source: src/util/lp/lp_dual_core_solver.hpp:264-295,599-660.
package lp

import (
	"fmt"
	"math/big"
	"math/rand"
	"reflect"
	"strings"
	"testing"
)

func lpRationalDualBreakpointFixture(t *testing.T, h *lpPairHarness, n uint32) (*lpRationalCoreFixture, *RationalDualCoreSolver) {
	t.Helper()
	f := lpNewRationalCoreFixture(t, h, 1, n+1)
	f.settings.SimplexStrategy = StrategyLU
	h.must(f.a.Set(0, 0, lpScalarBoundValue(h, "1")))
	can := make([]bool, n+1)
	for j := uint32(1); j <= n; j++ {
		can[j] = true
	}
	m, e := NewRationalDualCoreSolver(f.input(), &can)
	h.must(e)
	return f, m
}
func TestLPRationalDualBreakpointEligibilityExactOracle(t *testing.T) {
	for _, kind := range []ColumnType{ColumnLowBound, ColumnUpperBound, ColumnBoxed, ColumnFixed, ColumnFree, ColumnType(99)} {
		for _, sign := range []int{-1, 0, 1} {
			for _, a := range []int64{-3, 0, 2} {
				for _, x := range []int64{0, 1, 2} {
					h := newLPPairHarness(t)
					f, m := lpRationalDualBreakpointFixture(t, h, 1)
					m.SignOfAlphaR = sign
					f.types[1] = kind
					h.must(h.m.SetInt32(&f.up.Data[1], 2))
					h.must(h.m.SetInt32(&f.x.Data[1], int32(x)))
					h.must(h.m.Copy(&m.PivotRow.Data[1], lpScalarBoundValue(h, big.NewRat(a, 1000000007).RatString())))
					got, e := m.CanBeBreakpoint(1)
					assertion := a != 0 && (kind == ColumnLowBound && x != 0 || kind == ColumnUpperBound && x != 2)
					if assertion && sourceDebugEnabled {
						if e != ErrAssertion {
							t.Fatal(kind, sign, a, x, e)
						}
					} else {
						h.must(e)
						want := false
						if a != 0 {
							switch kind {
							case ColumnLowBound:
								want = int64(sign)*a > 0
							case ColumnUpperBound:
								want = int64(sign)*a < 0
							case ColumnBoxed:
								want = (x == 0) == (int64(sign)*a > 0)
							case ColumnFree:
								want = true
							}
						}
						if got != want {
							t.Fatal(kind, sign, a, x, got, want)
						}
					}
					h.must(m.Close())
					f.close()
					lpExactVectorNamedCensus(t, h, nil)
				}
			}
		}
	}
}

func TestLPRationalDualBreakpointHarrisIndependentWindows(t *testing.T) {
	rng := rand.New(rand.NewSource(33171))
	for world := 0; world < 240; world++ {
		h := newLPPairHarness(t)
		f, m := lpRationalDualBreakpointFixture(t, h, 7)
		tolerance := big.NewRat(int64(world%5), 37)
		h.must(h.m.Copy(&m.HarrisTolerance, lpScalarBoundValue(h, tolerance.RatString())))
		times, plain, pivots := map[uint32]*big.Rat{}, map[uint32]*big.Rat{}, map[uint32]*big.Rat{}
		var min *big.Rat
		for j := uint32(1); j <= 7; j++ {
			f.types[j] = ColumnBoxed
			h.must(h.m.SetInt32(&f.up.Data[j], 2))
			low := rng.Intn(2) == 0
			if !low {
				h.must(h.m.SetInt32(&f.x.Data[j], 2))
			}
			a := big.NewRat(int64(1+rng.Intn(9)), int64(1+rng.Intn(11)))
			if rng.Intn(2) == 0 {
				a.Neg(a)
			}
			d := new(big.Rat).SetFrac(new(big.Int).Lsh(big.NewInt(int64(1+rng.Intn(7))), uint(81+world%29)), big.NewInt(13))
			if rng.Intn(2) == 0 {
				d.Neg(d)
			}
			h.must(h.m.Copy(&m.PivotRow.Data[j], lpScalarBoundValue(h, a.RatString())))
			h.must(h.m.Copy(&m.D.Data[j], lpScalarBoundValue(h, d.RatString())))
			selected := new(big.Rat).Set(d)
			if low && d.Sign() < 0 || !low && d.Sign() > 0 {
				selected.SetInt64(0)
			}
			plain[j] = new(big.Rat).Abs(new(big.Rat).Quo(selected, a))
			shifted := new(big.Rat).Set(selected)
			if low {
				shifted.Add(shifted, tolerance)
			} else {
				shifted.Sub(shifted, tolerance)
			}
			times[j] = new(big.Rat).Abs(new(big.Rat).Quo(shifted, a))
			pivots[j] = new(big.Rat).Abs(a)
			if min == nil || times[j].Cmp(min) < 0 {
				min = new(big.Rat).Set(times[j])
			}
			if m.BreakpointSet == nil {
				m.BreakpointSet = map[uint32]bool{}
			}
			m.BreakpointSet[j] = true
		}
		out := lpScalarBoundValue(h, "97")
		h.must(m.CalculateHarrisDeltaOnBreakpointSet(out))
		lpSparseRationalCheck(t, h, out, min)
		h.must(m.FillTightSetOnHarrisDelta(out))
		want := map[uint32]bool{}
		q := uint32(0)
		for j := uint32(1); j <= 7; j++ {
			if plain[j].Cmp(min) <= 0 {
				want[j] = true
				if q == 0 || pivots[j].Cmp(pivots[q]) > 0 {
					q = j
				}
			}
		}
		if !reflect.DeepEqual(m.TightSet, want) {
			t.Fatal(world, m.TightSet, want)
		}
		h.must(m.FindQOnTightSet())
		delete(want, q)
		if m.Q != int32(q) || !reflect.DeepEqual(m.TightSet, want) {
			t.Fatal("largest pivot/ascending ties", world, m.Q, q, m.TightSet, want)
		}
		h.must(m.FindQAndTightSet())
		if m.Q != int32(q) {
			t.Fatal(m.Q, q)
		}
		wantPosition := PositionAtLowBound
		at, e := h.m.IsZero(&f.x.Data[q])
		h.must(e)
		if !at {
			wantPosition = PositionAtUpperBound
		}
		if m.EnteringBoundaryPosition != wantPosition {
			t.Fatal(m.EnteringBoundaryPosition, wantPosition)
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}
func TestLPRationalDualBreakpointPositionAndEmptySourceProfiles(t *testing.T) {
	for _, kind := range []ColumnType{ColumnFixed, ColumnLowBound, ColumnUpperBound, ColumnBoxed, ColumnFree, ColumnType(99)} {
		for _, x := range []int32{0, 1, 2} {
			h := newLPPairHarness(t)
			f, m := lpRationalDualBreakpointFixture(t, h, 1)
			f.types[1] = kind
			h.must(h.m.SetInt32(&f.up.Data[1], 2))
			h.must(h.m.SetInt32(&f.x.Data[1], x))
			got, e := m.GetNonBasicColumnValuePosition(1)
			want := PositionNotAtBound
			switch kind {
			case ColumnFree:
				want = PositionFreeOfBounds
			case ColumnUpperBound:
				if x == 2 {
					want = PositionAtUpperBound
				}
			case ColumnFixed:
				if x == 0 {
					want = PositionAtFixed
				}
			case ColumnLowBound:
				if x == 0 {
					want = PositionAtLowBound
				}
			case ColumnBoxed:
				if x == 0 {
					want = PositionAtLowBound
				} else if x == 2 {
					want = PositionAtUpperBound
				}
			default:
				want = PositionAtLowBound
			}
			if kind == ColumnType(99) && sourceDebugEnabled {
				if e != ErrAssertion {
					t.Fatal(e)
				}
			} else {
				h.must(e)
				if got != want {
					t.Fatal(kind, x, got, want)
				}
			}
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		}
	}
	for _, op := range []string{"harris", "q"} {
		h := newLPPairHarness(t)
		f, m := lpRationalDualBreakpointFixture(t, h, 1)
		out := lpScalarBoundValue(h, "97")
		var e error
		if op == "harris" {
			e = m.CalculateHarrisDeltaOnBreakpointSet(out)
		} else {
			m.Q = 42
			e = m.FindQOnTightSet()
		}
		if sourceDebugEnabled {
			if e != ErrAssertion {
				t.Fatal(e)
			}
		} else {
			h.must(e)
			if op == "harris" {
				lpSparseRationalCheck(t, h, out, new(big.Rat))
			} else if m.Q != -1 {
				t.Fatal(m.Q)
			}
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPRationalDualBreakpointPhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, op := range []string{"eligible", "fill", "harris", "tight", "q", "combined"} {
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
					f, m := lpRationalDualBreakpointFixture(t, h, 3)
					m.SignOfAlphaR = 1
					huge := strings.Repeat("1234567890", 16)
					for j := uint32(1); j <= 3; j++ {
						f.types[j] = ColumnBoxed
						h.must(h.m.SetString(&f.low.Data[j], huge+"1/13"))
						h.must(h.m.Copy(&f.x.Data[j], &f.low.Data[j]))
						h.must(h.m.Copy(&f.up.Data[j], &f.low.Data[j]))
						h.must(h.m.SetString(&m.D.Data[j], huge+fmt.Sprintf("%d/17", j)))
						h.must(h.m.SetString(&m.PivotRow.Data[j], huge+fmt.Sprintf("%d/19", j)))
					}
					m.BreakpointSet = map[uint32]bool{1: true, 2: true, 3: true}
					m.TightSet = map[uint32]bool{1: true, 2: true, 3: true}
					out := lpScalarBoundValue(h, "97")
					delta := lpScalarBoundValue(h, huge+"7/23")
					h.budget.FailAfter(ordinal)
					var e error
					switch op {
					case "eligible":
						_, e = m.CanBeBreakpoint(1)
					case "fill":
						e = m.FillBreakpointSet()
					case "harris":
						e = m.CalculateHarrisDeltaOnBreakpointSet(out)
					case "tight":
						e = m.FillTightSetOnHarrisDelta(delta)
					case "q":
						e = m.FindQOnTightSet()
					case "combined":
						e = m.FindQAndTightSet()
					}
					h.budget.FailAfter(-1)
					if e == nil {
						success, done = true, true
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(e) {
							t.Fatal(e)
						}
						if retry := m.FindQAndTightSet(); retry != ErrExactVectorPending {
							t.Fatal("stopped retry", retry)
						}
					}
					if op == "fill" && (m.BreakpointSet[1] && (!m.BreakpointSet[2] || !m.BreakpointSet[3]) || m.BreakpointSet[2] && !m.BreakpointSet[3]) {
						t.Fatal("nonbasis publication prefix", m.BreakpointSet)
					}
					if op == "tight" && (m.TightSet[3] && (!m.TightSet[1] || !m.TightSet[2]) || m.TightSet[2] && !m.TightSet[1]) {
						t.Fatal("tight ascending publication prefix", m.TightSet)
					}
					h.must(m.Close())
					f.close()
					lpExactVectorNamedCensus(t, h, nil)
				})
				if done {
					break
				}
			}
			if !success || direct && failures == 0 {
				t.Fatal(op, direct, success, failures)
			}
		}
	}
}

func TestLPRationalDualBreakpointMetadataPrefixes(t *testing.T) {
	for _, phase := range []string{"pivot-zero-short-types", "pivot", "types", "low", "harris-pivot", "tight-pivot", "q-pivot", "empty-combined", "harris-local", "tight-local", "q-local", "combined-local", "stopped"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			f, m := lpRationalDualBreakpointFixture(t, h, 1)
			m.SignOfAlphaR = 1
			f.types[1] = ColumnBoxed
			h.must(h.m.SetInt32(&m.PivotRow.Data[1], 1))
			m.BreakpointSet = map[uint32]bool{1: true}
			m.TightSet = map[uint32]bool{1: true}
			piv, types, low := m.PivotRow.Data, f.types, f.low.Data
			out := lpScalarBoundValue(h, "97")
			var e error
			switch phase {
			case "pivot-zero-short-types":
				h.must(h.m.SetInt32(&m.PivotRow.Data[1], 0))
				f.types = nil
				var yes bool
				yes, e = m.CanBeBreakpoint(1)
				h.must(e)
				if yes {
					t.Fatal("zero pivot accessed type")
				}
			case "pivot":
				m.PivotRow.Data = nil
				_, e = m.CanBeBreakpoint(1)
			case "types":
				f.types = nil
				_, e = m.CanBeBreakpoint(1)
			case "low":
				f.low.Data = nil
				_, e = m.CanBeBreakpoint(1)
			case "harris-pivot":
				m.PivotRow.Data = nil
				e = m.CalculateHarrisDeltaOnBreakpointSet(out)
			case "tight-pivot":
				m.PivotRow.Data = nil
				e = m.FillTightSetOnHarrisDelta(out)
			case "q-pivot":
				m.PivotRow.Data = nil
				m.Q = 42
				e = m.FindQOnTightSet()
				if m.Q != -1 {
					t.Fatal("Q reset prefix", m.Q)
				}
			case "empty-combined":
				m.BreakpointSet = nil
				e = m.FindQAndTightSet()
			case "harris-local":
				m.harrisValues = NewIndexedRationalVector(h.m, 2)
				e = m.CalculateHarrisDeltaOnBreakpointSet(out)
			case "tight-local":
				m.tightValue = NewIndexedRationalVector(h.m, 1)
				e = m.FillTightSetOnHarrisDelta(out)
			case "q-local":
				m.qValues = NewIndexedRationalVector(h.m, 2)
				e = m.FindQOnTightSet()
			case "combined-local":
				m.harrisDelta = NewIndexedRationalVector(h.m, 1)
				e = m.FindQAndTightSet()
			case "stopped":
				m.stopped = true
				e = m.FillBreakpointSet()
			}
			if phase != "pivot-zero-short-types" && e == nil {
				t.Fatal("missing rejection")
			}
			if phase == "empty-combined" && !sourceDebugEnabled && m.Q != -1 {
				t.Fatal("native empty Q prefix", m.Q)
			}
			m.PivotRow.Data, f.types, f.low.Data = piv, types, low
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPRationalDualBreakpointPreservesOlderComparison(t *testing.T) {
	for _, op := range []string{"eligible", "harris", "tight", "q", "combined"} {
		matched := false
		for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
			h := newLPPairHarness(t)
			f, m := lpRationalDualBreakpointFixture(t, h, 2)
			m.SignOfAlphaR = 1
			v := h.pair(strings.Repeat("9876543210", 100)+"/7", strings.Repeat("1234567890", 100)+"/13")
			for j := uint32(1); j <= 2; j++ {
				f.types[j] = ColumnBoxed
				h.must(h.m.Copy(&f.low.Data[j], &v.X))
				h.must(h.m.Copy(&f.x.Data[j], &v.Y))
				h.must(h.m.Copy(&f.up.Data[j], &v.X))
				h.must(h.m.Copy(&m.D.Data[j], &v.X))
				h.must(h.m.Copy(&m.PivotRow.Data[j], &v.Y))
			}
			m.BreakpointSet = map[uint32]bool{1: true, 2: true}
			m.TightSet = map[uint32]bool{1: true, 2: true}
			out := lpScalarBoundValue(h, "97")
			var caller exactEtaCalls
			before, e := caller.begin(h.m)
			h.must(e)
			h.budget.FailAfter(ordinal)
			_, e = h.m.Compare(&v.X, &v.Y)
			h.budget.FailAfter(-1)
			e = caller.after(h.m, before, e)
			older := h.m.ComparisonInvocationState()
			if older.Active {
				matched = true
				if !lpExactPriorityPhysicalRefusal(e) {
					t.Fatal(e)
				}
				switch op {
				case "eligible":
					_, e = m.CanBeBreakpoint(1)
				case "harris":
					e = m.CalculateHarrisDeltaOnBreakpointSet(out)
				case "tight":
					e = m.FillTightSetOnHarrisDelta(out)
				case "q":
					e = m.FindQOnTightSet()
				case "combined":
					e = m.FindQAndTightSet()
				}
				if m.calls.comparison != 0 || m.RationalCoreSolverBase.calls.comparison != 0 {
					t.Fatal("adopted caller comparison")
				}
			} else if e != nil && !lpExactPriorityPhysicalRefusal(e) {
				t.Fatal(e)
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

func TestLPRationalDualBreakpointLiveHarrisOutput(t *testing.T) {
	for _, alias := range []string{"tolerance", "later-cost", "later-pivot", "later-bound"} {
		h := newLPPairHarness(t)
		f, m := lpRationalDualBreakpointFixture(t, h, 2)
		f.types[1], f.types[2] = ColumnBoxed, ColumnBoxed
		h.must(h.m.SetInt32(&m.PivotRow.Data[1], 2))
		h.must(h.m.SetInt32(&m.PivotRow.Data[2], 1))
		h.must(h.m.SetInt32(&m.D.Data[1], 1))
		h.must(h.m.Copy(&m.D.Data[2], lpScalarBoundValue(h, "1/5")))
		h.must(h.m.Copy(&m.HarrisTolerance, lpScalarBoundValue(h, "1/5")))
		m.BreakpointSet = map[uint32]bool{1: true, 2: true}
		out := &m.HarrisTolerance
		switch alias {
		case "later-cost":
			out = &m.D.Data[2]
		case "later-pivot":
			out = &m.PivotRow.Data[2]
		case "later-bound":
			out = &f.low.Data[2]
		}
		// First time is3/5, second time2/5. Publishing the first candidate into
		// any of these live inputs would change the second candidate and the result.
		h.must(m.CalculateHarrisDeltaOnBreakpointSet(out))
		lpSparseRationalCheck(t, h, out, big.NewRat(2, 5))
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}
