package lp

import (
	"fmt"
	"math/big"
	"sort"
	"strings"
	"testing"

	z3rand "github.com/glycerine/g3/pkg/util/random"
)

func TestLPExactBreakpointAdvanceIndependentOracle(t *testing.T) {
	randomStops, continuedZeros, exhausted := 0, 0, 0
	for world := 0; world < 480; world++ {
		h := newLPPairHarness(t)
		s := lpNewRemovalModel(t, h, "-1", "3", true)
		m := s.Core.RPrimal
		h.must(m.Ed.Resize(2))
		h.must(m.BreakpointQueue.Resize(6))
		direction := int64(world%3 - 1)
		m.SignOfEnteringDelta = int32(direction)
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
		h.must(h.m.SetString(&m.D.Data[0], slope.RatString()))
		h.must(h.m.SetString(&m.EpsilonOfReducedCost, epsilon.RatString()))
		seed := uint32(world + 5)
		s.Settings.RandomSeed(seed)
		rng := z3rand.New(seed)
		type rec struct {
			j    uint32
			typ  BreakpointType
			x, y *big.Rat
		}
		records := make([]rec, 6)
		for k := range records {
			rank := (k*5 + world) % 6
			x := big.NewRat(int64(rank/2), 3)
			y := big.NewRat(int64(rank%2+1), 11)
			if (k+world)%2 == 0 {
				x.Neg(x)
				y.Neg(y)
			}
			j := uint32(2 + k%2)
			typ := BreakpointType((k + world) % 3)
			if k == world%6 {
				j = 0
				typ = BreakpointUpper
			}
			if k == 0 && world%7 == 0 && j != 0 {
				x.SetInt64(0)
				y.SetInt64(0)
				typ = BreakpointFixed
			}
			records[k] = rec{j, typ, x, y}
			h.must(m.AddBreakpoint(j, h.pair(x.RatString(), y.RatString()), typ))
		}
		order := []int{0, 1, 2, 3, 4, 5}
		absolute := func(r rec) (*big.Rat, *big.Rat) {
			x, y := new(big.Rat).Set(r.x), new(big.Rat).Set(r.y)
			sign := x.Sign()
			if sign == 0 {
				sign = y.Sign()
			}
			if sign < 0 {
				x.Neg(x)
				y.Neg(y)
			}
			return x, y
		}
		sort.Slice(order, func(i, j int) bool {
			a, b := records[order[i]], records[order[j]]
			ax, ay := absolute(a)
			bx, by := absolute(b)
			c := ax.Cmp(bx)
			if c != 0 {
				return c < 0
			}
			return ay.Cmp(by) < 0
		})
		used, last := 0, -1
		for _, k := range order {
			b := records[k]
			used++
			last = k
			delta := big.NewRat(direction, 1)
			if b.j != 0 {
				delta.Mul(delta, new(big.Rat).Abs(ed[b.j-2]))
				if b.typ == BreakpointFixed && (b.x.Sign() != 0 || b.y.Sign() != 0) {
					delta.Mul(delta, big.NewRat(2, 1))
				}
			}
			slope.Add(slope, delta)
			product := new(big.Rat).Mul(slope, big.NewRat(direction, 1))
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
		output := h.pair("97", "101")
		if world%5 == 0 {
			output = &(*m.X)[0]
		}
		got, e := m.AdvanceOnSortedBreakpoints(0, output)
		h.must(e)
		want := records[last]
		if got != int(want.j) || int(m.BreakpointQueue.Size()) != 6-used || len(m.Breakpoints) != 6 {
			t.Fatal(world, got, want.j, m.BreakpointQueue.Size(), used)
		}
		h.check(output, want.x, want.y)
		if s.Settings.RandomNext() != rng.Next() {
			t.Fatal("RNG consumption", world)
		}
		if m.breakpointSlope != nil || m.work != nil {
			t.Fatal("retained successful locals")
		}
		h.must(s.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
	if randomStops == 0 || continuedZeros == 0 || exhausted == 0 {
		t.Fatal("missing stopping branches", randomStops, continuedZeros, exhausted)
	}
}

func TestLPExactBreakpointAdvanceWrapperAndGuards(t *testing.T) {
	for _, op := range []string{"wrapper", "wrapper-bad-entering", "empty", "entering-index", "record-index", "record-nil", "nil-output", "record-alias"} {
		t.Run(op, func(t *testing.T) {
			h := newLPPairHarness(t)
			s := lpNewRemovalModel(t, h, "-1", "3", false)
			m := s.Core.RPrimal
			h.must(m.Ed.Resize(1))
			h.must(m.BreakpointQueue.Resize(4))
			m.SignOfEnteringDelta = 1
			h.must(h.m.SetInt32(&m.D.Data[0], -2))
			h.must(m.Ed.SetValue(&h.pair("1", "0").X, 0))
			output := h.pair("97", "101")
			want := ErrBounds
			entering := uint32(0)
			switch op {
			case "wrapper":
				h.must(s.Bounds.Types.Set(2, ColumnLowBound))
				h.must((*m.X)[2].Copy(h.m, h.pair("4", "0")))
				h.must(s.Bounds.Types.Set(0, ColumnBoxed))
				h.must((*m.UpperBounds)[0].Copy(h.m, h.pair("3", "0")))
				h.must(m.AddBreakpoint(1, h.pair("99", "0"), BreakpointFixed))
			case "wrapper-bad-entering":
				entering = 99
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "empty":
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "entering-index":
				entering = 99
			case "record-index":
				h.must(m.BreakpointQueue.Enqueue(0, h.pair("1", "0")))
			case "record-nil":
				h.must(m.AddBreakpoint(2, h.pair("1", "0"), BreakpointLow))
				h.must(m.Breakpoints[0].Delta.Del(h.m))
				m.Breakpoints[0] = nil
			case "nil-output":
				output = nil
			case "record-alias":
				h.must(m.AddBreakpoint(2, h.pair("1", "2"), BreakpointLow))
				output = m.Breakpoints[0].Delta
				want = nil
			}
			var got int
			var err error
			if op == "wrapper" {
				got, err = m.FindLeavingAndTWithBreakpoints(0, output)
				want = nil
				if sourceDebugEnabled {
					want = ErrAssertion
					if len(m.Breakpoints) != 1 || m.Breakpoints[0].Column != 1 {
						t.Fatal("wrapper assertion occurred after fill")
					}
				} else {
					if got != 2 {
						t.Fatal(got)
					}
					h.check(output, big.NewRat(4, 1), big.NewRat(0, 1))
				}
			} else if op == "wrapper-bad-entering" {
				got, err = m.FindLeavingAndTWithBreakpoints(entering, output)
			} else {
				got, err = m.AdvanceOnSortedBreakpoints(entering, output)
			}
			if err != want {
				t.Fatal(got, err, want)
			}
			if err != nil {
				if _, e := m.AdvanceOnSortedBreakpoints(0, h.pair("0", "0")); e != ErrExactVectorPending {
					t.Fatal("missing stopped owner", e)
				}
			}
			h.must(s.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPExactBreakpointAdvancePhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, op := range []string{"full", "threshold", "output"} {
		for _, direct := range []bool{false, true} {
			success, failures, queueReceipts, primalReceipts := false, 0, 0, 0
			for ordinal := int64(0); ordinal < 1200; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("%s/%t/%d", op, direct, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					s := lpNewRemovalModel(t, h, "-1", "3", false)
					m := s.Core.RPrimal
					h.must(m.Ed.Resize(1))
					h.must(m.BreakpointQueue.Resize(5))
					m.SignOfEnteringDelta = 1
					huge := strings.Repeat("1234567890", 12)
					h.must(h.m.SetString(&m.D.Data[0], "-"+huge+"1/7"))
					h.must(h.m.SetString(&m.Ed.Data[0], huge+"3/11"))
					count := 5
					if op != "full" {
						count = 1
						h.must(h.m.SetInt32(&m.Ed.Data[0], 0))
					}
					if op == "threshold" {
						h.must(h.m.SetString(&m.EpsilonOfReducedCost, huge+"5/13"))
					}
					for k := 0; k < count; k++ {
						h.must(m.AddBreakpoint(2, h.pair(huge+fmt.Sprint(k+1)+"/17", huge+fmt.Sprint(k+2)+"/19"), BreakpointFixed))
					}
					output := h.pair("97", "101")
					h.budget.FailAfter(ordinal)
					_, err := m.AdvanceOnSortedBreakpoints(0, output)
					h.budget.FailAfter(-1)
					if err == nil {
						success, done = true, true
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						if m.calls.pending() && m.BreakpointQueue.calls.pending() {
							t.Fatal("adopted queue invocation")
						}
						if m.BreakpointQueue.calls.comparison != 0 {
							queueReceipts++
						}
						if m.calls.comparison != 0 {
							primalReceipts++
						}
						if m.breakpointSlope != nil && len(m.Breakpoints) != count {
							t.Fatal("records lost on failure")
						}
						if _, e := m.AdvanceOnSortedBreakpoints(0, output); e != ErrExactVectorPending {
							t.Fatal(e)
						}
					}
					h.must(s.Close())
					lpExactVectorNamedCensus(t, h, nil)
				})
				if done {
					break
				}
			}
			if !success || (direct && failures == 0) {
				t.Fatal(op, direct, success, failures)
			}
			if direct && op == "full" && queueReceipts == 0 {
				t.Fatal("no queue-owned comparison refusal")
			}
			if op == "threshold" && primalReceipts == 0 {
				t.Fatal("no primal-owned threshold comparison refusal")
			}
			t.Log("failure ownership", op, direct, failures, queueReceipts, primalReceipts)
		}
	}
}

func TestLPExactBreakpointAdvancePreservesOlderComparisonOwner(t *testing.T) {
	matched := false
	for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			h := newLPPairHarness(t)
			s := lpNewRemovalModel(t, h, "-1", "3", false)
			m := s.Core.RPrimal
			h.must(m.Ed.Resize(1))
			h.must(m.BreakpointQueue.Resize(2))
			m.SignOfEnteringDelta = 1
			h.must(h.m.SetInt32(&m.D.Data[0], -2))
			h.must(h.m.SetInt32(&m.Ed.Data[0], 1))
			h.must(m.AddBreakpoint(2, h.pair("1/3", "1/7"), BreakpointFixed))
			h.must(m.AddBreakpoint(2, h.pair("2/3", "2/7"), BreakpointUpper))
			v := h.pair(strings.Repeat("9876543210", 100)+"/7", strings.Repeat("1234567890", 100)+"/13")
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
				_, err = m.AdvanceOnSortedBreakpoints(0, h.pair("97", "101"))
				if m.calls.comparison != 0 || m.BreakpointQueue.calls.comparison != 0 {
					t.Fatal("adopted caller comparison", err)
				}
			}
			h.must(s.Close())
			if older.Active && h.m.ComparisonInvocationState() != older {
				t.Fatal("destroyed caller comparison")
			}
			h.must(caller.cleanupRaw(h.m))
			h.must(caller.cleanupRoots(h.m))
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
	if !matched {
		t.Fatal("no real caller comparison refusal")
	}
}

func TestLPExactBreakpointAdvanceAlreadyStoppedQueue(t *testing.T) {
	matched := false
	for ordinal := int64(0); ordinal < 80 && !matched; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			h := lpNewDirectSparseHarness(t)
			s := lpNewRemovalModel(t, h, "-1", "3", false)
			m := s.Core.RPrimal
			h.must(m.BreakpointQueue.Resize(2))
			huge := strings.Repeat("1234567890", 12)
			h.must(m.AddBreakpoint(2, h.pair(huge+"1/3", huge+"3/7"), BreakpointLow))
			h.must(m.AddBreakpoint(2, h.pair(huge+"5/7", huge+"7/11"), BreakpointUpper))
			priority := h.pair(huge+"9/13", huge+"1/17")
			h.budget.FailAfter(ordinal)
			err := m.BreakpointQueue.Enqueue(1, priority)
			h.budget.FailAfter(-1)
			if m.BreakpointQueue.calls.pending() {
				matched = true
				if !lpExactPriorityPhysicalRefusal(err) {
					t.Fatal(err)
				}
				old := h.m.ComparisonInvocationState()
				output := h.pair("97", "101")
				if _, e := m.AdvanceOnSortedBreakpoints(0, output); e != ErrExactQueuePending {
					t.Fatal(e)
				}
				h.check(output, big.NewRat(97, 1), big.NewRat(101, 1))
				if m.calls.pending() || m.breakpointSlope == nil || len(m.Breakpoints) != 2 {
					t.Fatal("lost queue/primal ownership separation")
				}
				if h.m.ComparisonInvocationState() != old {
					t.Fatal("changed queue comparison before cleanup")
				}
			}
			h.must(s.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
	if !matched {
		t.Fatal("no real stopped queue")
	}
}
