package lp

import (
	"fmt"
	"math/big"
	"sort"
	"strings"
	"testing"

	z3rand "github.com/glycerine/g3/pkg/util/random"
)

func TestLPRationalPrimalBreakpointAdvanceIndependentOracle(t *testing.T) {
	randomStops, continuedZeros, exhausted := 0, 0, 0
	for world := 0; world < 480; world++ {
		h := newLPPairHarness(t)
		f, m := lpRationalPricingFixture(t, h, 2, 3)
		h.must(m.BreakpointQueue.Resize(6))
		direction := int32(world%3 - 1)
		m.SignOfEnteringDelta = direction
		ed := []*big.Rat{big.NewRat(int64(world%4), 3), big.NewRat(int64(world%5), 3)}
		for i := range ed {
			h.must(h.m.SetString(&m.Ed.Data[i], ed[i].RatString()))
		}
		slope := big.NewRat(int64(world%13-8), 3)
		epsilon := big.NewRat(int64(world%3-1), 7)
		if world%4 == 0 {
			slope.SetInt64(0)
			epsilon.SetInt64(0)
		}
		h.must(h.m.SetString(&m.D.Data[2], slope.RatString()))
		h.must(h.m.SetString(&m.EpsilonOfReducedCost, epsilon.RatString()))
		seed := uint32(world + 5)
		f.settings.RandomSeed(seed)
		rng := z3rand.New(seed)
		type rec struct {
			j   uint32
			typ BreakpointType
			v   *big.Rat
		}
		records := make([]rec, 6)
		for k := range records {
			rank := (k*5 + world) % 6
			v := big.NewRat(int64(rank+1), 7)
			if (k+world)%2 == 0 {
				v.Neg(v)
			}
			j := uint32(k % 2)
			typ := BreakpointType((k + world) % 3)
			if k == world%6 {
				j = 2
				typ = BreakpointUpper
			}
			if k == 0 && world%7 == 0 && j != 2 {
				v.SetInt64(0)
				typ = BreakpointFixed
			}
			records[k] = rec{j, typ, v}
			h.must(m.AddBreakpoint(j, lpScalarBoundValue(h, v.RatString()), typ))
		}
		order := []int{0, 1, 2, 3, 4, 5}
		sort.Slice(order, func(i, j int) bool {
			return new(big.Rat).Abs(records[order[i]].v).Cmp(new(big.Rat).Abs(records[order[j]].v)) < 0
		})
		used, last := 0, -1
		for _, k := range order {
			r := records[k]
			inc := big.NewRat(int64(direction), 1)
			if r.j != 2 {
				inc.Mul(inc, new(big.Rat).Abs(ed[r.j]))
				if r.typ == BreakpointFixed && r.v.Sign() != 0 {
					inc.Mul(inc, big.NewRat(2, 1))
				}
			}
			slope.Add(slope, inc)
			used++
			last = k
			product := new(big.Rat).Mul(slope, big.NewRat(int64(direction), 1))
			if product.Cmp(new(big.Rat).Neg(epsilon)) > 0 {
				break
			}
			if slope.Sign() == 0 {
				if rng.Next()%2 == 0 {
					randomStops++
					break
				}
				continuedZeros++
			}
		}
		if used == 6 {
			exhausted++
		}
		out := lpScalarBoundValue(h, "37/41")
		if world%3 == 0 {
			out = &m.D.Data[2]
		} else if world%3 == 1 {
			out = m.Breakpoints[(world+2)%6].Delta
		}
		leaving, e := m.AdvanceOnSortedBreakpoints(2, out)
		h.must(e)
		if leaving != int(records[last].j) || int(m.BreakpointQueue.Size()) != 6-used {
			t.Fatal(world, leaving, last, m.BreakpointQueue.Size(), used)
		}
		lpSparseRationalCheck(t, h, out, records[last].v)
		if f.settings.RandomNext() != rng.Next() {
			t.Fatal("RNG consumption", world)
		}
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
	if randomStops == 0 || continuedZeros == 0 || exhausted == 0 {
		t.Fatal("missing branch", randomStops, continuedZeros, exhausted)
	}
}

func TestLPRationalPrimalBreakpointAdvancePhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, direct := range []bool{false, true} {
		success, failures, queue := false, 0, 0
		for ordinal := int64(0); ordinal < 1200; ordinal++ {
			done := false
			t.Run(fmt.Sprintf("%t/%d", direct, ordinal), func(t *testing.T) {
				var h *lpPairHarness
				if direct {
					h = lpNewDirectSparseHarness(t)
				} else {
					h = newLPPairHarness(t)
				}
				f, m := lpRationalPricingFixture(t, h, 1, 2)
				h.must(m.BreakpointQueue.Resize(4))
				m.SignOfEnteringDelta = 1
				huge := strings.Repeat("1234567890", 12)
				h.must(h.m.SetString(&m.Ed.Data[0], huge+"1/7"))
				h.must(h.m.SetString(&m.D.Data[1], "-"+huge+"3/11"))
				h.must(h.m.SetString(&m.EpsilonOfReducedCost, "-1/17"))
				for _, den := range []string{"13", "19", "23", "29"} {
					h.must(m.AddBreakpoint(0, lpScalarBoundValue(h, huge+"5/"+den), BreakpointFixed))
				}
				out := lpScalarBoundValue(h, "31/37")
				h.budget.FailAfter(ordinal)
				_, err := m.AdvanceOnSortedBreakpoints(1, out)
				h.budget.FailAfter(-1)
				if err == nil {
					success = true
					done = true
				} else {
					failures++
					if !lpExactPriorityPhysicalRefusal(err) {
						t.Fatal(err)
					}
					if m.BreakpointQueue.calls.pending() {
						queue++
						if m.calls.pending() {
							t.Fatal("adopted queue receipt")
						}
					}
					if _, e := m.AdvanceOnSortedBreakpoints(1, out); e != ErrExactVectorPending {
						t.Fatal("lost stop", e)
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
		if !success || failures == 0 || (direct && queue == 0) {
			t.Fatal(direct, success, failures, queue)
		}
	}
}

func TestLPRationalPrimalBreakpointAdvanceWrapper(t *testing.T) {
	h := newLPPairHarness(t)
	f, m := lpRationalPricingFixture(t, h, 1, 2)
	h.must(m.BreakpointQueue.Resize(4))
	m.SignOfEnteringDelta = 1
	f.types[0] = ColumnLowBound
	f.types[1] = ColumnFree
	h.must(h.m.SetInt32(&f.x.Data[0], -1))
	h.must(h.m.SetInt32(&f.low.Data[0], 0))
	h.must(m.Ed.SetValue(lpScalarBoundValue(h, "-1"), 0))
	h.must(h.m.SetInt32(&m.D.Data[1], -1))
	out := lpScalarBoundValue(h, "17")
	leaving, e := m.FindLeavingAndTWithBreakpoints(1, out)
	if sourceDebugEnabled {
		if e != ErrAssertion || leaving != -1 || len(m.Breakpoints) != 0 {
			t.Fatal(leaving, e)
		}
		lpSparseRationalCheck(t, h, out, big.NewRat(17, 1))
	} else {
		h.must(e)
		if leaving != 0 {
			t.Fatal(leaving)
		}
		lpSparseRationalCheck(t, h, out, big.NewRat(1, 1))
	}
	h.must(m.Close())
	f.close()
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPRationalPrimalBreakpointAdvanceGuards(t *testing.T) {
	for _, phase := range []string{"empty", "entering-index", "record-index", "record-nil", "nil-output", "wrapper-index"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			f, m := lpRationalPricingFixture(t, h, 1, 2)
			h.must(m.BreakpointQueue.Resize(4))
			m.SignOfEnteringDelta = 1
			out := lpScalarBoundValue(h, "97")
			want := ErrBounds
			entering := uint32(1)
			switch phase {
			case "empty":
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "entering-index", "wrapper-index":
				entering = 99
			case "record-index":
				h.must(m.BreakpointQueue.Enqueue(0, lpScalarBoundValue(h, "1")))
			case "record-nil":
				h.must(m.AddBreakpoint(0, lpScalarBoundValue(h, "1"), BreakpointLow))
				h.must(h.m.Del(m.Breakpoints[0].Delta))
				m.Breakpoints[0] = nil
			case "nil-output":
				out = nil
			}
			var leaving int
			var err error
			if phase == "wrapper-index" {
				leaving, err = m.FindLeavingAndTWithBreakpoints(entering, out)
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			} else {
				leaving, err = m.AdvanceOnSortedBreakpoints(entering, out)
			}
			if err != want || leaving != -1 {
				t.Fatal(leaving, err, want)
			}
			if out != nil {
				lpSparseRationalCheck(t, h, out, big.NewRat(97, 1))
			}
			if _, e := m.AdvanceOnSortedBreakpoints(1, lpScalarBoundValue(h, "0")); e != ErrExactVectorPending {
				t.Fatal(e)
			}
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPRationalPrimalBreakpointAdvanceAlreadyStoppedQueue(t *testing.T) {
	matched := false
	for ordinal := int64(0); ordinal < 80 && !matched; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			h := lpNewDirectSparseHarness(t)
			f, m := lpRationalPricingFixture(t, h, 1, 2)
			h.must(m.BreakpointQueue.Resize(2))
			huge := strings.Repeat("1234567890", 12)
			h.must(m.AddBreakpoint(2, lpScalarBoundValue(h, huge+"1/3"), BreakpointLow))
			h.must(m.AddBreakpoint(2, lpScalarBoundValue(h, huge+"5/7"), BreakpointUpper))
			priority := lpScalarBoundValue(h, huge+"9/13")
			h.budget.FailAfter(ordinal)
			err := m.BreakpointQueue.Enqueue(1, priority)
			h.budget.FailAfter(-1)
			if m.BreakpointQueue.calls.pending() {
				matched = true
				if !lpExactPriorityPhysicalRefusal(err) {
					t.Fatal(err)
				}
				old := h.m.ComparisonInvocationState()
				output := lpScalarBoundValue(h, "97")
				if _, e := m.AdvanceOnSortedBreakpoints(0, output); e != ErrExactQueuePending {
					t.Fatal(e)
				}
				lpSparseRationalCheck(t, h, output, big.NewRat(97, 1))
				if m.calls.pending() || m.breakpointSlope == nil || len(m.Breakpoints) != 2 {
					t.Fatal("lost queue/primal ownership separation")
				}
				if h.m.ComparisonInvocationState() != old {
					t.Fatal("changed queue comparison before cleanup")
				}
			}
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
	if !matched {
		t.Fatal("no real stopped queue")
	}
}

func TestLPRationalPrimalBreakpointAdvanceThresholdAndOutputFailures(t *testing.T) {
	for _, op := range []string{"threshold", "output"} {
		for _, direct := range []bool{false, true} {
			success, failures, comparisons := false, 0, 0
			for ordinal := int64(0); ordinal < 600; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("%s/%t/%d", op, direct, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					f, m := lpRationalPricingFixture(t, h, 1, 2)
					h.must(m.BreakpointQueue.Resize(1))
					m.SignOfEnteringDelta = 1
					huge := strings.Repeat("1234567890", 12)
					h.must(h.m.SetString(&m.D.Data[1], "-"+huge+"1/7"))
					if op == "threshold" {
						h.must(h.m.SetString(&m.EpsilonOfReducedCost, huge+"5/13"))
					}
					h.must(m.AddBreakpoint(0, lpScalarBoundValue(h, huge+"7/17"), BreakpointFixed))
					out := lpScalarBoundValue(h, "97")
					h.budget.FailAfter(ordinal)
					_, err := m.AdvanceOnSortedBreakpoints(1, out)
					h.budget.FailAfter(-1)
					if err == nil {
						success = true
						done = true
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						if m.calls.comparison != 0 {
							comparisons++
						}
						if m.calls.pending() && m.BreakpointQueue.calls.pending() {
							t.Fatal("receipt ownership overlap")
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
			if !success || direct && failures == 0 || op == "threshold" && comparisons == 0 {
				t.Fatal(op, direct, success, failures, comparisons)
			}
		}
	}
}
