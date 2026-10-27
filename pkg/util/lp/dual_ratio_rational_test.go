// Source: src/util/lp/lp_dual_core_solver.hpp:575-599,660-693.
package lp

import (
	"fmt"
	"math/big"
	"math/rand"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestLPRationalDualRatioIndependentGroups(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	rng := rand.New(rand.NewSource(33171))
	type candidate struct {
		j                 uint32
		time, pivot, loss *big.Rat
		boxed, upper      bool
	}
	for world := 0; world < 240; world++ {
		h := newLPPairHarness(t)
		n := 3 + rng.Intn(8)
		f, m := lpRationalDualBreakpointFixture(t, h, uint32(n))
		sign := int64(1)
		if world%2 == 0 {
			sign = -1
		}
		f.types[0] = ColumnBoxed
		h.must(h.m.SetInt32(&f.up.Data[0], 10))
		x := int32(13)
		if sign < 0 {
			x = -3
		}
		h.must(h.m.SetInt32(&f.x.Data[0], x))
		magnitude := big.NewRat(int64(1+rng.Intn(100)), 4)
		if world%7 == 0 {
			magnitude.SetInt(new(big.Int).Lsh(big.NewInt(1), uint(90+world%31)))
		}
		initial := new(big.Rat).Mul(magnitude, big.NewRat(sign, 1))
		h.must(h.m.Copy(&m.Delta, lpScalarBoundValue(h, initial.RatString())))
		candidates := make([]candidate, 0, n)
		for j := uint32(1); j <= uint32(n); j++ {
			boxed := rng.Intn(4) != 0
			upper := boxed && rng.Intn(2) == 0
			f.types[j] = ColumnLowBound
			if boxed {
				f.types[j] = ColumnBoxed
			}
			span := big.NewRat(int64(1+rng.Intn(9)), 4)
			h.must(h.m.Copy(&f.up.Data[j], lpScalarBoundValue(h, span.RatString())))
			if upper {
				h.must(h.m.Copy(&f.x.Data[j], &f.up.Data[j]))
			}
			a := big.NewRat(int64(1+rng.Intn(8)), 4)
			tm := big.NewRat(int64(rng.Intn(9)), 2)
			pivot := new(big.Rat).Mul(a, big.NewRat(sign, 1))
			d := new(big.Rat).Mul(tm, a)
			if upper {
				pivot.Neg(pivot)
				d.Neg(d)
			}
			h.must(h.m.Copy(&m.PivotRow.Data[j], lpScalarBoundValue(h, pivot.RatString())))
			h.must(h.m.Copy(&m.D.Data[j], lpScalarBoundValue(h, d.RatString())))
			candidates = append(candidates, candidate{j, tm, a, new(big.Rat).Mul(span, a), boxed, upper})
		}
		sort.Slice(candidates, func(i, j int) bool {
			c := candidates[i].time.Cmp(candidates[j].time)
			if c != 0 {
				return c < 0
			}
			return candidates[i].j < candidates[j].j
		})
		flipped := map[uint32]bool{}
		delta := new(big.Rat).Set(initial)
		var chosen candidate
		var tight map[uint32]bool
		for start := 0; start < len(candidates); {
			end := start + 1
			for end < len(candidates) && candidates[end].time.Cmp(candidates[start].time) == 0 {
				end++
			}
			group := candidates[start:end]
			chosen = group[0]
			boxed := true
			loss := new(big.Rat)
			for _, c := range group {
				if c.pivot.Cmp(chosen.pivot) > 0 {
					chosen = c
				}
				boxed = boxed && c.boxed
				loss.Add(loss, c.loss)
			}
			tight = map[uint32]bool{}
			for _, c := range group {
				if c.j != chosen.j {
					tight[c.j] = true
				}
			}
			remaining := new(big.Rat).Sub(new(big.Rat).Abs(delta), loss)
			if !boxed || remaining.Sign() <= 0 || end == len(candidates) {
				break
			}
			for _, c := range group {
				flipped[c.j] = true
			}
			delta.Mul(remaining, big.NewRat(sign, 1))
			start = end
		}
		found, e := m.RatioTest()
		h.must(e)
		if !found || m.Q != int32(chosen.j) || m.Status != StatusFeasible || !reflect.DeepEqual(m.FlippedBoxed, flipped) || !reflect.DeepEqual(m.TightSet, tight) {
			t.Fatal(world, found, m.Q, chosen.j, m.Status, m.FlippedBoxed, flipped, m.TightSet, tight)
		}
		lpSparseRationalCheck(t, h, &m.Delta, delta)
		lpSparseRationalCheck(t, h, &m.ThetaD, new(big.Rat).Mul(chosen.time, big.NewRat(sign, 1)))
		for j := uint32(1); j <= uint32(n); j++ {
			if m.BreakpointSet[j] == flipped[j] {
				t.Fatal("breakpoint erasure", world, j, m.BreakpointSet, flipped)
			}
		}
		position := PositionAtLowBound
		if chosen.upper {
			position = PositionAtUpperBound
		}
		if m.EnteringBoundaryPosition != position {
			t.Fatal(position, m.EnteringBoundaryPosition)
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPRationalDualRatioLossIndependentOracleAndAliases(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for world := 0; world < 200; world++ {
		h := newLPPairHarness(t)
		f, m := lpRationalDualBreakpointFixture(t, h, 4)
		m.Q = 3
		m.TightSet = map[uint32]bool{1: true, 2: true, 4: true}
		sum := new(big.Rat)
		for j := uint32(1); j <= 4; j++ {
			lo := big.NewRat(int64(world+int(j)), 17)
			span := new(big.Rat).SetFrac(new(big.Int).Lsh(big.NewInt(int64(j+1)), uint(81+world%23)), big.NewInt(19))
			if world%2 == 0 {
				span.Neg(span)
			}
			up := new(big.Rat).Add(lo, span)
			a := big.NewRat(int64(j)-3, 23)
			h.must(h.m.Copy(&f.low.Data[j], lpScalarBoundValue(h, lo.RatString())))
			h.must(h.m.Copy(&f.up.Data[j], lpScalarBoundValue(h, up.RatString())))
			h.must(h.m.Copy(&m.PivotRow.Data[j], lpScalarBoundValue(h, a.RatString())))
			sum.Add(sum, new(big.Rat).Abs(new(big.Rat).Mul(span, a)))
		}
		out := lpScalarBoundValue(h, "97")
		// These outputs alias a later input; only the complete sum may publish.
		switch world % 4 {
		case 1:
			out = &f.low.Data[4]
		case 2:
			out = &f.up.Data[4]
		case 3:
			out = &m.PivotRow.Data[4]
		}
		h.must(m.DeltaLostOnFlipsOfTightBreakpoints(out))
		lpSparseRationalCheck(t, h, out, sum)
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPRationalDualRatioEmptyAndLastGroup(t *testing.T) {
	for _, kind := range []ColumnType{ColumnBoxed, ColumnLowBound, ColumnFixed} {
		h := newLPPairHarness(t)
		f, m := lpRationalDualBreakpointFixture(t, h, 1)
		f.types[0] = ColumnLowBound
		f.types[1] = kind
		h.must(h.m.SetInt32(&f.x.Data[0], -3))
		h.must(h.m.SetInt32(&m.Delta, -100))
		h.must(h.m.SetInt32(&f.up.Data[1], 2))
		h.must(h.m.SetInt32(&m.PivotRow.Data[1], -1))
		h.must(h.m.SetInt32(&m.D.Data[1], 1))
		m.FlippedBoxed = map[uint32]bool{91: true}
		h.must(h.m.SetInt32(&m.ThetaD, 97))
		m.Q = 42
		for attempt := 0; attempt < 2; attempt++ {
			found, e := m.RatioTest()
			h.must(e)
			if kind == ColumnFixed {
				want := StatusTentativeDualUnbounded
				if attempt == 1 {
					want = StatusDualUnbounded
				}
				if found || m.Status != want || m.Q != 42 || len(m.FlippedBoxed) != 0 {
					t.Fatal(found, m.Status, m.Q, m.FlippedBoxed)
				}
				lpSparseRationalCheck(t, h, &m.ThetaD, big.NewRat(97, 1))
			} else {
				if !found || m.Q != 1 || len(m.FlippedBoxed) != 0 || !m.BreakpointSet[1] {
					t.Fatal("last group must remain", found, m.Q, m.FlippedBoxed, m.BreakpointSet)
				}
				lpSparseRationalCheck(t, h, &m.Delta, big.NewRat(-100, 1))
				lpSparseRationalCheck(t, h, &m.ThetaD, big.NewRat(-1, 1))
			}
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func lpRationalDualRatioLargeFixture(t *testing.T, h *lpPairHarness) (*lpRationalCoreFixture, *RationalDualCoreSolver) {
	t.Helper()
	f, m := lpRationalDualBreakpointFixture(t, h, 3)
	huge := strings.Repeat("1234567890", 12)
	f.types[0] = ColumnLowBound
	h.must(h.m.SetInt32(&f.x.Data[0], -3))
	delta := new(big.Int)
	delta.SetString(huge+"7", 10)
	delta.Exp(delta, big.NewInt(3), nil)
	delta.Neg(delta)
	h.must(h.m.SetString(&m.Delta, delta.String()))
	for j := uint32(1); j <= 3; j++ {
		f.types[j] = ColumnBoxed
		h.must(h.m.SetString(&f.up.Data[j], huge+"1/13"))
		h.must(h.m.SetString(&m.PivotRow.Data[j], "-"+huge+"3/17"))
		positive := lpScalarBoundValue(h, huge+"3/17")
		times := lpScalarBoundValue(h, fmt.Sprint(j))
		h.must(h.m.Mul(positive, times, &m.D.Data[j]))
	}
	m.Q = 1
	m.TightSet = map[uint32]bool{2: true, 3: true}
	return f, m
}
func TestLPRationalDualRatioPhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, op := range []string{"span", "loss", "ratio"} {
		for _, direct := range []bool{false, true} {
			success, failures := false, 0
			for ordinal := int64(0); ordinal < 2000; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("%s/%t/%d", op, direct, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					f, m := lpRationalDualRatioLargeFixture(t, h)
					out := lpScalarBoundValue(h, "97")
					h.budget.FailAfter(ordinal)
					var e error
					switch op {
					case "span":
						e = m.BoundSpan(1, out)
					case "loss":
						e = m.DeltaLostOnFlipsOfTightBreakpoints(out)
					case "ratio":
						var found bool
						found, e = m.RatioTest()
						if e == nil && (!found || m.Q != 3 || !m.FlippedBoxed[1] || !m.FlippedBoxed[2]) {
							t.Fatal("full ratio did not flip", found, m.Q, m.FlippedBoxed)
						}
					}
					h.budget.FailAfter(-1)
					if e == nil {
						success, done = true, true
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(e) {
							t.Fatal(e)
						}
						if _, retry := m.RatioTest(); retry != ErrExactVectorPending {
							t.Fatal("stopped retry", retry)
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
			if !success || direct && failures == 0 {
				t.Fatal(op, direct, success, failures)
			}
		}
	}
}

func TestLPRationalDualRatioBoundSpanAliases(t *testing.T) {
	for _, alias := range []string{"out", "low", "upper"} {
		h := newLPPairHarness(t)
		f, m := lpRationalDualBreakpointFixture(t, h, 1)
		h.must(h.m.SetString(&f.low.Data[1], "123456789012345678901234567890/17"))
		h.must(h.m.SetString(&f.up.Data[1], "987654321098765432109876543210/19"))
		low, _ := new(big.Rat).SetString("123456789012345678901234567890/17")
		up, _ := new(big.Rat).SetString("987654321098765432109876543210/19")
		want := new(big.Rat).Sub(up, low)
		out := lpScalarBoundValue(h, "97")
		if alias == "low" {
			out = &f.low.Data[1]
		} else if alias == "upper" {
			out = &f.up.Data[1]
		}
		h.must(m.BoundSpan(1, out))
		lpSparseRationalCheck(t, h, out, want)
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}
func TestLPRationalDualRatioSetOrderAndShortCircuit(t *testing.T) {
	h := newLPPairHarness(t)
	f, m := lpRationalDualBreakpointFixture(t, h, 3)
	m.Q = 1
	m.TightSet = map[uint32]bool{2: true, 3: true}
	f.types[1] = ColumnLowBound
	types := f.types
	f.types = f.types[:2]
	boxed, e := m.TightBreakpointsAreAllBoxed()
	h.must(e)
	if boxed {
		t.Fatal("q type must stop before tight extent")
	}
	f.types = types
	for j := 1; j <= 3; j++ {
		f.types[j] = ColumnBoxed
	}
	boxed, e = m.TightBreakpointsAreAllBoxed()
	h.must(e)
	if !boxed {
		t.Fatal("boxed group")
	}
	m.FlippedBoxed = map[uint32]bool{9: true}
	h.must(m.AddTightBreakpointsAndQToFlippedSet())
	if !reflect.DeepEqual(m.FlippedBoxed, map[uint32]bool{1: true, 2: true, 3: true, 9: true}) {
		t.Fatal(m.FlippedBoxed)
	}
	m.BreakpointSet = map[uint32]bool{1: true, 2: true, 3: true, 4: true}
	h.must(m.EraseTightBreakpointsAndQFromBreakpointSet())
	if !reflect.DeepEqual(m.BreakpointSet, map[uint32]bool{4: true}) {
		t.Fatal(m.BreakpointSet)
	}
	h.must(m.Close())
	f.close()
	lpExactVectorNamedCensus(t, h, nil)
}
func TestLPRationalDualRatioMetadataAndStoppedPrefixes(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, phase := range []string{"span-low", "span-upper", "loss-pivot", "loss-q", "boxed-q", "boxed-tight", "loss-local", "trial-local", "ratio-local", "alpha", "stopped"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			f, m := lpRationalDualRatioLargeFixture(t, h)
			out := lpScalarBoundValue(h, "97")
			low, up, pivot, types := f.low.Data, f.up.Data, m.PivotRow.Data, f.types
			var e error
			switch phase {
			case "span-low":
				f.low.Data = nil
				e = m.BoundSpan(1, out)
			case "span-upper":
				f.up.Data = nil
				e = m.BoundSpan(1, out)
			case "loss-pivot":
				m.PivotRow.Data = nil
				e = m.DeltaLostOnFlipsOfTightBreakpoints(out)
			case "loss-q":
				m.Q = -1
				e = m.DeltaLostOnFlipsOfTightBreakpoints(out)
			case "boxed-q":
				m.Q = -1
				_, e = m.TightBreakpointsAreAllBoxed()
			case "boxed-tight":
				f.types = f.types[:2]
				_, e = m.TightBreakpointsAreAllBoxed()
			case "loss-local":
				m.flipLossValues = NewIndexedRationalVector(h.m, 2)
				e = m.DeltaLostOnFlipsOfTightBreakpoints(out)
			case "trial-local":
				m.ratioTrial = NewIndexedRationalVector(h.m, 1)
				_, e = m.RatioTest()
			case "ratio-local":
				m.ratioLoss = NewIndexedRationalVector(h.m, 1)
				_, e = m.RatioTest()
			case "alpha":
				m.P = -1
				_, e = m.RatioTest()
			case "stopped":
				m.stopped = true
				_, e = m.RatioTest()
			}
			if e == nil {
				t.Fatal("missing rejection")
			}
			f.low.Data, f.up.Data, m.PivotRow.Data, f.types = low, up, pivot, types
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}
func TestLPRationalDualRatioPreservesOlderComparison(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, op := range []string{"span", "loss", "ratio"} {
		matched := false
		for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
			h := newLPPairHarness(t)
			f, m := lpRationalDualRatioLargeFixture(t, h)
			v := h.pair(strings.Repeat("9876543210", 100)+"/7", strings.Repeat("1234567890", 100)+"/13")
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
				case "span":
					e = m.BoundSpan(1, out)
				case "loss":
					e = m.DeltaLostOnFlipsOfTightBreakpoints(out)
				case "ratio":
					_, e = m.RatioTest()
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
