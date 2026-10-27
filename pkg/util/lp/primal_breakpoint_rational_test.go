package lp

import (
	"fmt"
	"math/big"
	"strings"
	"testing"
)

func TestLPRationalPrimalBreakpointIndependentRecords(t *testing.T) {
	h := newLPPairHarness(t)
	f, m := lpRationalPricingFixture(t, h, 0, 0)
	h.must(m.BreakpointQueue.Resize(1))
	want := make([]*big.Rat, 180)
	for i := range want {
		v := big.NewRat(int64(i%31-15), int64(i%17+1))
		input := lpScalarBoundValue(h, v.RatString())
		h.must(m.AddBreakpoint(uint32(i), input, BreakpointType(i%3)))
		b := m.Breakpoints[i]
		if b.Column != uint32(i) || b.Type != BreakpointType(i%3) {
			t.Fatal("record metadata", i)
		}
		lpSparseRationalCheck(t, h, b.Delta, v)
		want[i] = new(big.Rat).Abs(v)
		lpSparseRationalCheck(t, h, &m.BreakpointQueue.priorities.Data[i], want[i])
		h.must(h.m.SetInt32(input, 97))
		lpSparseRationalCheck(t, h, b.Delta, v)
	}
	previous := new(big.Rat)
	seen := make(map[uint32]bool)
	for !m.BreakpointQueue.IsEmpty() {
		i, e := m.BreakpointQueue.Dequeue()
		h.must(e)
		if seen[i] || previous.Cmp(want[i]) > 0 {
			t.Fatal("priority order or duplicate", i)
		}
		seen[i] = true
		previous.Set(want[i])
	}
	if len(seen) != len(want) {
		t.Fatal("lost records")
	}
	h.must(m.ClearBreakpoints())
	if len(m.Breakpoints) != 0 || m.BreakpointQueue.Size() != 0 {
		t.Fatal("clear")
	}
	h.must(m.AddBreakpoint(7, lpScalarBoundValue(h, "-1/7"), BreakpointLow))
	h.must(m.Close())
	f.close()
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPRationalPrimalBreakpointIndependentSlopes(t *testing.T) {
	for world := 0; world < 480; world++ {
		h := newLPPairHarness(t)
		f, m := lpRationalPricingFixture(t, h, 1, 2)
		ed := big.NewRat(int64(world%19-9), 23)
		h.must(h.m.SetString(&m.Ed.Data[0], ed.RatString()))
		m.SignOfEnteringDelta = int32(world%3 - 1)
		b := &RationalBreakpoint{Column: 0, Type: BreakpointType(world % 3), Delta: lpScalarBoundValue(h, fmt.Sprint(world%7-3))}
		entering := world%4 == 0
		if entering {
			b.Column = 1
			b.Type = BreakpointLow
			h.must(h.m.SetInt32(b.Delta, 1))
		}
		old := big.NewRat(int64(world%13-6), 29)
		slope := lpScalarBoundValue(h, old.RatString())
		if world%5 == 0 {
			slope = &m.Ed.Data[0]
			old.Set(ed)
		}
		if world%5 == 1 {
			slope = b.Delta
			old.SetInt64(int64(world%7 - 3))
			if entering {
				old.SetInt64(1)
			}
		}
		inc := big.NewRat(int64(m.SignOfEnteringDelta), 1)
		if !entering {
			inc.Mul(inc, new(big.Rat).Abs(ed))
			if b.Type == BreakpointFixed && world%7-3 != 0 {
				inc.Mul(inc, big.NewRat(2, 1))
			}
		}
		want := new(big.Rat).Add(old, inc)
		h.must(m.ChangeSlopeOnBreakpoint(1, b, slope))
		lpSparseRationalCheck(t, h, slope, want)
		h.must(m.Close())
		f.close()
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPRationalPrimalBreakpointPhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, op := range []string{"enqueue", "slope-fixed", "slope-bound", "slope-entering"} {
		for _, direct := range []bool{false, true} {
			success, failures, queue := false, 0, 0
			for ordinal := int64(0); ordinal < 800; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("%s/%t/%d", op, direct, ordinal), func(t *testing.T) {
					h := newLPPairHarness(t)
					if direct {
						h = lpNewDirectSparseHarness(t)
					}
					f, m := lpRationalPricingFixture(t, h, 1, 2)
					m.SignOfEnteringDelta = -1
					huge := strings.Repeat("1234567890", 12)
					h.must(h.m.SetString(&m.Ed.Data[0], huge+"1/7"))
					slope := lpScalarBoundValue(h, huge+"3/11")
					delta := lpScalarBoundValue(h, huge+"5/13")
					b := &RationalBreakpoint{Column: 0, Type: BreakpointFixed, Delta: delta}
					if op == "slope-bound" {
						b.Type = BreakpointLow
					}
					if op == "slope-entering" {
						b.Column = 1
						b.Type = BreakpointUpper
					}
					if op == "enqueue" {
						h.must(m.BreakpointQueue.Resize(2))
						h.must(m.AddBreakpoint(0, lpScalarBoundValue(h, huge+"9/19"), BreakpointLow))
					}
					h.budget.FailAfter(ordinal)
					var err error
					if op == "enqueue" {
						err = m.AddBreakpoint(0, delta, BreakpointUpper)
					} else {
						err = m.ChangeSlopeOnBreakpoint(1, b, slope)
					}
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
							if m.calls.pending() || len(m.Breakpoints) != 2 {
								t.Fatal("queue receipt or record publication")
							}
						}
						if e := m.ClearBreakpoints(); e != ErrExactVectorPending {
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
			if !success || failures == 0 || (op == "enqueue" && queue == 0) {
				t.Fatal(op, direct, success, failures, queue)
			}
		}
	}
}

func TestLPRationalPrimalBreakpointReachedGuards(t *testing.T) {
	for _, phase := range []string{"zero-capacity", "nil", "nil-breakpoint", "nil-slope", "heading-extent", "ed-extent", "entering-fixed", "entering-zero", "heading", "unknown", "zero-ed"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			f, m := lpRationalPricingFixture(t, h, 1, 2)
			h.must(h.m.SetInt32(&m.Ed.Data[0], 2))
			m.SignOfEnteringDelta = 1
			b := &RationalBreakpoint{Column: 0, Type: BreakpointLow, Delta: lpScalarBoundValue(h, "1")}
			slope := lpScalarBoundValue(h, "3")
			want := ErrBounds
			var err error
			switch phase {
			case "zero-capacity":
				err = m.AddBreakpoint(0, b.Delta, b.Type)
				if len(m.Breakpoints) != 1 {
					t.Fatal("lost record prefix")
				}

			case "nil":
				err = m.AddBreakpoint(0, nil, BreakpointLow)
			case "nil-breakpoint":
				err = m.ChangeSlopeOnBreakpoint(1, nil, slope)
			case "nil-slope":
				err = m.ChangeSlopeOnBreakpoint(1, b, nil)
			case "heading-extent":
				b.Column = 99
				err = m.ChangeSlopeOnBreakpoint(1, b, slope)
			case "ed-extent":
				h.must(m.Ed.Resize(0))
				err = m.ChangeSlopeOnBreakpoint(1, b, slope)
			case "entering-fixed", "entering-zero":
				b.Column = 1
				if phase == "entering-fixed" {
					b.Type = BreakpointFixed
				} else {
					h.must(h.m.SetInt32(b.Delta, 0))
				}
				err = m.ChangeSlopeOnBreakpoint(1, b, slope)
				want = nil
				if sourceDebugEnabled {
					want = ErrAssertion
				} else {
					lpSparseRationalCheck(t, h, slope, big.NewRat(4, 1))
				}
			case "heading":
				b.Column = 1
				err = m.ChangeSlopeOnBreakpoint(0, b, slope)
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "unknown":
				b.Type = BreakpointType(99)
				err = m.ChangeSlopeOnBreakpoint(1, b, slope)
				want = nil
				if sourceDebugEnabled {
					want = ErrAssertion
				}
				lpSparseRationalCheck(t, h, slope, big.NewRat(3, 1))
			case "zero-ed":
				h.must(h.m.SetInt32(&m.Ed.Data[0], 0))
				b.Type = BreakpointType(99)
				err = m.ChangeSlopeOnBreakpoint(1, b, slope)
				want = nil
				lpSparseRationalCheck(t, h, slope, big.NewRat(3, 1))
			}
			if err != want {
				t.Fatal(err, want)
			}
			if err != nil {
				if e := m.AddBreakpoint(0, b.Delta, b.Type); e != ErrExactVectorPending {
					t.Fatal("lost stopped state", e)
				}
			}
			h.must(m.Close())
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLPRationalPrimalBreakpointClearBeforeStoppedQueue(t *testing.T) {
	matched := false
	for ordinal := int64(0); ordinal < 80 && !matched; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			h := lpNewDirectSparseHarness(t)
			f, m := lpRationalPricingFixture(t, h, 1, 2)
			var e error
			h.must(m.BreakpointQueue.Resize(2))
			huge := strings.Repeat("1234567890", 12)
			h.must(m.AddBreakpoint(0, lpScalarBoundValue(h, huge+"1/3"), BreakpointLow))
			h.must(m.AddBreakpoint(1, lpScalarBoundValue(h, huge+"5/7"), BreakpointUpper))
			priority := lpScalarBoundValue(h, huge+"9/13")
			h.budget.FailAfter(ordinal)
			err := m.BreakpointQueue.Enqueue(1, priority)
			h.budget.FailAfter(-1)
			if m.BreakpointQueue.calls.pending() {
				matched = true
				if !lpExactPriorityPhysicalRefusal(err) {
					t.Fatal(err)
				}
				if e = m.ClearBreakpoints(); e != ErrExactQueuePending {
					t.Fatal(e)
				}
				if len(m.Breakpoints) != 0 {
					t.Fatal("queue failure preceded record clear")
				}
				if m.calls.pending() {
					t.Fatal("adopted stopped queue")
				}
			}
			h.must(m.Close())
			f.close()
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
	if !matched {
		t.Fatal("no actual queue comparison refusal")
	}
}

func TestLPRationalPrimalBreakpointPreservesOlderComparisonOwner(t *testing.T) {
	for _, op := range []string{"enqueue", "slope"} {
		matched := false
		for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", op, ordinal), func(t *testing.T) {
				h := newLPPairHarness(t)
				f, m := lpRationalPricingFixture(t, h, 1, 2)
				var e error
				v := h.pair(strings.Repeat("9876543210", 100)+"/7", strings.Repeat("1234567890", 100)+"/13")
				h.must(m.Ed.Resize(1))
				h.must(h.m.SetString(&m.Ed.Data[0], "3/7"))
				m.SignOfEnteringDelta = -1
				delta, slope := lpScalarBoundValue(h, "5/13"), lpScalarBoundValue(h, "23/19")
				b := &RationalBreakpoint{Column: 0, Type: BreakpointFixed, Delta: delta}
				h.must(m.BreakpointQueue.Resize(2))
				h.must(m.AddBreakpoint(2, lpScalarBoundValue(h, "1/3"), BreakpointLow))
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
					if op == "enqueue" {
						err = m.AddBreakpoint(2, delta, BreakpointUpper)
					} else {
						err = m.ChangeSlopeOnBreakpoint(1, b, slope)
					}
					if m.calls.comparison != 0 || m.BreakpointQueue.calls.comparison != 0 {
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
