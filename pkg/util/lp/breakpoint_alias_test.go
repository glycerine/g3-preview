package lp

// Source: src/util/lp/breakpoint.h; lp_primal_core_solver.hpp:1245-1248.

import (
	"fmt"
	"strings"
	"testing"
)

// Source add_breakpoint takes Delta by value before constructing/appending the
// record. Its argument may borrow an already-owned record while push_back
// grows the container. An interrupted later copy must leave that owner intact.
func TestLPBreakpointOwnedDeltaAliasPhysicalPrefixes(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	xText := "-" + strings.Repeat("1234567890", 20) + "/7"
	yText := strings.Repeat("9876543210", 20) + "/13"
	x, y := mustLPRat(t, xText), mustLPRat(t, yText)
	for _, scalar := range []bool{false, true} {
		for _, direct := range []bool{false, true} {
			done, failures := false, 0
			for ordinal := int64(0); ordinal < 800 && !done; ordinal++ {
				t.Run(fmt.Sprintf("%v/%v/%d", scalar, direct, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					input := h.pair(xText, yText)
					if scalar {
						f, m := lpRationalPricingFixture(t, h, 0, 0)
						h.must(m.BreakpointQueue.Resize(1))
						h.must(m.AddBreakpoint(7, &input.X, BreakpointFixed))
						old := m.Breakpoints[0]
						h.budget.FailAfter(ordinal)
						err := m.AddBreakpoint(91, old.Delta, BreakpointUpper)
						h.budget.FailAfter(-1)
						if m.Breakpoints[0] != old || old.Column != 7 || old.Type != BreakpointFixed || len(m.Breakpoints) > 2 {
							t.Fatal("original scalar record publication")
						}
						lpSparseRationalCheck(t, h, old.Delta, x)
						if err != nil {
							failures++
							if !lpExactPriorityPhysicalRefusal(err) || m.AddBreakpoint(92, old.Delta, BreakpointLow) != ErrExactVectorPending {
								t.Fatal("scalar alias failure", err)
							}
						} else {
							done = true
							if len(m.Breakpoints) != 2 || m.BreakpointQueue.Size() != 2 || m.Breakpoints[1].Column != 91 || m.Breakpoints[1].Type != BreakpointUpper || m.Breakpoints[1].Delta == old.Delta {
								t.Fatal("independent scalar record")
							}
							h.must(h.m.SetInt32(old.Delta, 17))
							lpSparseRationalCheck(t, h, m.Breakpoints[1].Delta, x)
							priority, e := m.BreakpointQueue.priority(1)
							h.must(e)
							lpSparseRationalCheck(t, h, priority, mustLPRat(t, strings.TrimPrefix(xText, "-")))
							h.must(m.ClearBreakpoints())
							if len(m.Breakpoints) != 0 || m.BreakpointQueue.Size() != 0 {
								t.Fatal("scalar clear")
							}
						}
						h.must(m.Close())
						f.close()
					} else {
						s, e := NewLARSolver(h.m)
						h.must(e)
						m := s.Core.RPrimal
						h.must(m.BreakpointQueue.Resize(1))
						h.must(m.AddBreakpoint(7, input, BreakpointFixed))
						old := m.Breakpoints[0]
						h.budget.FailAfter(ordinal)
						err := m.AddBreakpoint(91, old.Delta, BreakpointUpper)
						h.budget.FailAfter(-1)
						if m.Breakpoints[0] != old || old.Column != 7 || old.Type != BreakpointFixed || len(m.Breakpoints) > 2 {
							t.Fatal("original pair record publication")
						}
						h.check(old.Delta, x, y)
						if err != nil {
							failures++
							if !lpExactPriorityPhysicalRefusal(err) || m.AddBreakpoint(92, old.Delta, BreakpointLow) != ErrExactVectorPending {
								t.Fatal("pair alias failure", err)
							}
						} else {
							done = true
							if len(m.Breakpoints) != 2 || m.BreakpointQueue.Size() != 2 || m.Breakpoints[1].Column != 91 || m.Breakpoints[1].Type != BreakpointUpper || m.Breakpoints[1].Delta == old.Delta {
								t.Fatal("independent pair record")
							}
							h.must(h.m.SetInt32(&old.Delta.X, 17))
							h.must(h.m.SetInt32(&old.Delta.Y, 19))
							h.check(m.Breakpoints[1].Delta, x, y)
							priority, e := m.BreakpointQueue.priority(1)
							h.must(e)
							h.check(priority, mustLPRat(t, strings.TrimPrefix(xText, "-")), mustLPRat(t, "-"+yText))
							h.must(m.ClearBreakpoints())
							if len(m.Breakpoints) != 0 || m.BreakpointQueue.Size() != 0 {
								t.Fatal("pair clear")
							}
						}
						h.must(s.Close())
					}
					lpExactVectorNamedCensus(t, h, nil)
				})
			}
			if !done || failures == 0 {
				t.Fatal("alias physical campaign", scalar, direct, done, failures)
			}
		}
	}
}
