package lp

import (
	"fmt"
	"math/big"
	"strings"
	"testing"
)

func TestLPExactBreakpointIndependentRecordQueueOracle(t *testing.T) {
	h := newLPPairHarness(t)
	s, e := NewLARSolver(h.m)
	h.must(e)
	m := s.Core.RPrimal
	h.must(m.BreakpointQueue.Resize(1))
	want := make([][2]*big.Rat, 180)
	for world := range want {
		x, y := big.NewRat(int64(world%19-9), 7), big.NewRat(int64(world%23-11), 13)
		if world%3 == 0 {
			x.SetInt64(0)
		}
		delta := h.pair(x.RatString(), y.RatString())
		h.must(m.AddBreakpoint(uint32(world), delta, BreakpointType(world%3)))
		b := m.Breakpoints[world]
		if b.Column != uint32(world) || b.Type != BreakpointType(world%3) {
			t.Fatal("record metadata", world)
		}
		h.check(b.Delta, x, y)
		if x.Sign() < 0 || (x.Sign() == 0 && y.Sign() < 0) {
			x.Neg(x)
			y.Neg(y)
		}
		want[world] = [2]*big.Rat{x, y}
		h.check(&m.BreakpointQueue.priorities.Data[world], x, y)
		// Both record and queue priority survive mutation of the borrowed input.
		h.must(h.m.SetInt32(&delta.X, 97))
		h.must(h.m.SetInt32(&delta.Y, 101))
	}
	previous := [2]*big.Rat{big.NewRat(0, 1), big.NewRat(0, 1)}
	for !m.BreakpointQueue.IsEmpty() {
		i, e := m.BreakpointQueue.Dequeue()
		h.must(e)
		v := want[i]
		order := previous[0].Cmp(v[0])
		if order == 0 {
			order = previous[1].Cmp(v[1])
		}
		if order > 0 {
			t.Fatal("queue order", i)
		}
		previous = v
	}
	h.must(m.ClearBreakpoints())
	if len(m.Breakpoints) != 0 || m.BreakpointQueue.Size() != 0 {
		t.Fatal("clear prefix")
	}
	h.must(m.AddBreakpoint(7, h.pair("-1/7", "3/11"), BreakpointLow))
	h.check(m.Breakpoints[0].Delta, big.NewRat(-1, 7), big.NewRat(3, 11))
	h.must(s.Close())
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLPExactBreakpointIndependentSlopeOracle(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	for world := 0; world < 360; world++ {
		h := newLPPairHarness(t)
		s := lpNewRemovalModel(t, h, "2", "3", false)
		m := s.Core.RPrimal
		h.must(m.Ed.Resize(1))
		ed := big.NewRat(int64(world%17-8), 19)
		h.must(h.m.SetString(&m.Ed.Data[0], ed.RatString()))
		m.SignOfEnteringDelta = int32(world%3 - 1)
		b := &PairBreakpoint{Column: 2, Type: BreakpointType(world % 3), Delta: h.pair(fmt.Sprint(world%7-3), fmt.Sprint(world%11-5))}
		entering := world%4 == 0
		if entering {
			b.Column = 0
			b.Type = BreakpointLow
			h.must(h.m.SetInt32(&b.Delta.X, 1))
		}
		old := big.NewRat(int64(world%13-6), 23)
		slopePair := h.pair(old.RatString(), "0")
		slope := &slopePair.X
		if world%5 == 0 {
			slope = &m.Ed.Data[0]
			old.Set(ed)
		}
		if world%5 == 1 {
			slope = &b.Delta.X
			old.SetInt64(int64(world%7 - 3))
			if entering {
				old.SetInt64(1)
			}
		}
		increment := big.NewRat(int64(m.SignOfEnteringDelta), 1)
		if !entering {
			increment.Mul(increment, new(big.Rat).Abs(ed))
			if b.Type == BreakpointFixed && (world%7-3 != 0 || world%11-5 != 0) {
				increment.Mul(increment, big.NewRat(2, 1))
			}
		}
		want := new(big.Rat).Add(old, increment)
		h.must(m.ChangeSlopeOnBreakpoint(0, b, slope))
		lpSparseRationalCheck(t, h, slope, want)
		h.must(s.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPExactBreakpointPhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, op := range []string{"enqueue", "enqueue-minimum", "enqueue-epsilon-minimum", "slope-minimum", "slope-fixed", "slope-bound", "slope-entering"} {
		for _, direct := range []bool{false, true} {
			success, failures, queue := false, 0, 0
			for ordinal := int64(0); ordinal < 800; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("%s/%t/%d", op, direct, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					s := lpNewRemovalModel(t, h, "2", "3", false)
					m := s.Core.RPrimal
					h.must(m.Ed.Resize(1))
					m.SignOfEnteringDelta = -1
					huge := strings.Repeat("1234567890", 12)
					h.must(h.m.SetString(&m.Ed.Data[0], huge+"1/7"))
					slope := h.pair(huge+"3/11", "0")
					delta := h.pair(huge+"5/13", huge+"7/17")
					b := &PairBreakpoint{Column: 2, Type: BreakpointFixed, Delta: delta}
					if op == "enqueue-minimum" {
						delta = h.pair("0", "0")
						h.must(h.m.SetInt32(&delta.X, -2147483648))
						h.must(h.m.SetInt32(&delta.Y, -2147483648))
						b.Delta = delta
					}
					if op == "enqueue-epsilon-minimum" {
						delta = h.pair("0", "0")
						h.must(h.m.SetInt32(&delta.Y, -2147483648))
						b.Delta = delta
					}
					if op == "slope-minimum" {
						h.must(h.m.SetInt32(&m.Ed.Data[0], -2147483648))
					}
					if op == "slope-bound" {
						b.Type = BreakpointLow
					}
					if op == "slope-entering" {
						b.Column = 0
						b.Type = BreakpointUpper
					}
					if strings.HasPrefix(op, "enqueue") {
						h.must(m.BreakpointQueue.Resize(2))
						h.must(m.AddBreakpoint(2, h.pair(huge+"1/3", huge+"9/19"), BreakpointLow))
					}
					h.budget.FailAfter(ordinal)
					var err error
					if strings.HasPrefix(op, "enqueue") {
						err = m.AddBreakpoint(2, delta, BreakpointUpper)
					} else {
						err = m.ChangeSlopeOnBreakpoint(0, b, &slope.X)
					}
					h.budget.FailAfter(-1)
					if err == nil {
						success, done = true, true
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
							if len(m.Breakpoints) != 2 {
								t.Fatal("lost appended record prefix")
							}
						}
						if e := m.ClearBreakpoints(); e != ErrExactVectorPending {
							t.Fatal("lost stop", e)
						}
					}
					h.must(s.Close())
					lpExactVectorNamedCensus(t, h, nil)
				})
				if done {
					break
				}
			}
			if !success || ((direct || !strings.HasSuffix(op, "minimum")) && failures == 0) || (op == "enqueue" && queue == 0) {
				t.Fatal(op, direct, success, failures, queue)
			}
		}
	}
}

func TestLPExactBreakpointSourcePrefixesAndGuards(t *testing.T) {
	for _, phase := range []string{"zero-capacity", "nil", "nil-slope", "heading-extent", "ed-extent", "entering-fixed", "entering-zero", "heading", "unknown", "zero-ed"} {
		h := newLPPairHarness(t)
		s := lpNewRemovalModel(t, h, "2", "3", false)
		m := s.Core.RPrimal
		h.must(m.Ed.Resize(1))
		h.must(h.m.SetInt32(&m.Ed.Data[0], 2))
		b := &PairBreakpoint{Column: 2, Type: BreakpointLow, Delta: h.pair("1", "0")}
		slope := h.pair("3", "0")
		want := ErrBounds
		var err error
		switch phase {
		case "zero-capacity":
			err = m.AddBreakpoint(2, b.Delta, b.Type)
			if len(m.Breakpoints) != 1 {
				t.Fatal("lost published record")
			}
		case "nil":
			err = m.AddBreakpoint(2, nil, BreakpointLow)
		case "nil-slope":
			err = m.ChangeSlopeOnBreakpoint(0, b, nil)
		case "heading-extent":
			b.Column = 99
			err = m.ChangeSlopeOnBreakpoint(0, b, &slope.X)
		case "ed-extent":
			h.must(m.Ed.Resize(0))
			err = m.ChangeSlopeOnBreakpoint(0, b, &slope.X)
		case "entering-fixed", "entering-zero":
			b.Column = 0
			if phase == "entering-fixed" {
				b.Type = BreakpointFixed
			} else {
				h.must(h.m.SetInt32(&b.Delta.X, 0))
			}
			want = nil
			if sourceDebugEnabled {
				want = ErrAssertion
			}
			err = m.ChangeSlopeOnBreakpoint(0, b, &slope.X)
		case "heading":
			b.Column = 0 // A real nonbasic heading is negative.
			if sourceDebugEnabled {
				want = ErrAssertion
			}
			err = m.ChangeSlopeOnBreakpoint(1, b, &slope.X)
		case "unknown", "zero-ed":
			b.Type = BreakpointType(99)
			want = nil
			if sourceDebugEnabled && phase == "unknown" {
				want = ErrAssertion
			}
			if phase == "zero-ed" {
				h.must(h.m.SetInt32(&m.Ed.Data[0], 0))
			}
			err = m.ChangeSlopeOnBreakpoint(0, b, &slope.X)
		}
		if err != want {
			t.Fatal(phase, err, want)
		}
		if err != nil {
			if e := m.ClearBreakpoints(); e != ErrExactVectorPending {
				t.Fatal("lost stopped owner", e)
			}
			if e := m.ChangeSlopeOnBreakpoint(0, b, &slope.X); e != ErrExactVectorPending {
				t.Fatal("lost slope stop", e)
			}
		}
		h.must(s.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPExactBreakpointClearBeforeStoppedQueue(t *testing.T) {
	matched := false
	for ordinal := int64(0); ordinal < 80 && !matched; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			h := lpNewDirectSparseHarness(t)
			s, e := NewLARSolver(h.m)
			h.must(e)
			m := s.Core.RPrimal
			h.must(m.BreakpointQueue.Resize(2))
			huge := strings.Repeat("1234567890", 12)
			h.must(m.AddBreakpoint(0, h.pair(huge+"1/3", huge+"3/7"), BreakpointLow))
			h.must(m.AddBreakpoint(1, h.pair(huge+"5/7", huge+"7/11"), BreakpointUpper))
			priority := h.pair(huge+"9/13", huge+"1/17")
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
			h.must(s.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
	if !matched {
		t.Fatal("no actual queue comparison refusal")
	}
}

func TestLPExactBreakpointPreservesOlderComparisonOwner(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, op := range []string{"enqueue", "slope"} {
		matched := false
		for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", op, ordinal), func(t *testing.T) {
				h := newLPPairHarness(t)
				s := lpNewRemovalModel(t, h, "2", "3", false)
				var e error
				m := s.Core.RPrimal
				v := h.pair(strings.Repeat("9876543210", 100)+"/7", strings.Repeat("1234567890", 100)+"/13")
				h.must(m.Ed.Resize(1))
				h.must(h.m.SetString(&m.Ed.Data[0], "3/7"))
				m.SignOfEnteringDelta = -1
				delta, slope := h.pair("5/13", "7/17"), h.pair("23/19", "0")
				b := &PairBreakpoint{Column: 2, Type: BreakpointFixed, Delta: delta}
				h.must(m.BreakpointQueue.Resize(2))
				h.must(m.AddBreakpoint(2, h.pair("1/3", "2/7"), BreakpointLow))
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
						err = m.ChangeSlopeOnBreakpoint(0, b, &slope.X)
					}
					if m.calls.comparison != 0 || m.BreakpointQueue.calls.comparison != 0 {
						t.Fatal("adopted older comparison", err)
					}
				}
				h.must(s.Close())
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
