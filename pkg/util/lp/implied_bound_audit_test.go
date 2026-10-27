// Source: src/util/lp/implied_bound.h; retained lp_bound_propagator.cpp copies.
package lp

import (
	"fmt"
	"io"
	"strings"
	"testing"
)

func TestLPImpliedBoundDirectAndAliasOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	value := strings.Repeat("2345678901", 100) + "/" + strings.Repeat("7654321098", 99) + "1"
	for _, direct := range []bool{false, true} {
		for _, action := range []string{"construct", "assign", "self"} {
			done, failures := false, 0
			for ordinal := int64(0); ordinal < 200 && !done; ordinal++ {
				t.Run(fmt.Sprintf("%v/%s/%d", direct, action, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					p := h.pair(value, "19")
					source, err := NewImpliedBound(h.m, &p.X, 73, true, true, 99, true)
					h.must(err)
					var b *ImpliedBound
					if action == "assign" {
						b, err = NewImpliedBound(h.m, &p.Y, 18, false, false, 12, false)
						h.must(err)
					}
					if action == "self" {
						b = source
					}
					h.budget.FailAfter(ordinal)
					switch action {
					case "construct":
						b, err = NewImpliedBound(h.m, &p.X, 73, true, true, 99, true)
					case "assign":
						err = b.Assign(source)
					case "self":
						err = b.Assign(b)
					}
					h.budget.FailAfter(-1)
					if b == nil {
						t.Fatal("lost partial owner")
					}
					if err != nil {
						failures++
						if !lpExactPriorityPhysicalRefusal(err) || !b.stopped {
							t.Fatal("actual physical prefix", err, b.stopped)
						}
						if _, e := b.Equal(source); e != ErrExactVectorPending {
							t.Fatal("partial read", e)
						}
						if action == "construct" && (b.Column != 0 || b.IsLowBound || b.CoefficientIsPositive || b.RowOrTermIndex != 0 || b.Strict) {
							t.Fatal("construction scalar publication")
						}
						if action == "assign" && (b.Column != 18 || b.IsLowBound || b.CoefficientIsPositive || b.RowOrTermIndex != 12 || b.Strict) {
							t.Fatal("assignment scalar publication")
						}
					} else {
						done = true
						equal, e := b.Equal(source)
						h.must(e)
						if !equal {
							t.Fatal("source deep copy/evidence")
						}
						if action != "self" {
							h.must(h.m.SetInt32(&source.Bound, 17))
							got, e := h.m.String(&b.Bound)
							h.must(e)
							if got != mustLPRat(t, value).RatString() {
								t.Fatal("shared numerical owner", got)
							}
						}
					}
					if b != source {
						h.must(b.Close())
					}
					h.must(source.Close())
					lpExactVectorNamedCensus(t, h, nil)
				})
			}
			if !done || action != "self" && failures == 0 {
				t.Fatal("physical campaign", direct, action, done, failures)
			}
			if action == "self" && failures != 0 {
				t.Fatal("self-assignment copied storage", direct, failures)
			}
		}
	}
}

func TestLPImpliedBoundPreservesOlderComparison(t *testing.T) {
	for _, assign := range []bool{false, true} {
		found := false
		for ordinal := int64(0); ordinal < 40 && !found; ordinal++ {
			h := newLPPairHarness(t)
			v := h.pair(strings.Repeat("9876543210", 100)+"/7", strings.Repeat("1234567890", 100)+"/13")
			source, err := NewImpliedBound(h.m, &v.X, 7, true, false, 9, true)
			h.must(err)
			b, err := NewImpliedBound(h.m, &v.Y, 1, false, true, 2, false)
			h.must(err)
			var caller exactEtaCalls
			before, err := caller.begin(h.m)
			h.must(err)
			h.budget.FailAfter(ordinal)
			_, err = h.m.Compare(&v.X, &v.Y)
			h.budget.FailAfter(-1)
			err = caller.after(h.m, before, err)
			older := h.m.ComparisonInvocationState()
			var constructed *ImpliedBound
			if older.Active {
				found = true
				if !lpExactPriorityPhysicalRefusal(err) {
					t.Fatal(err)
				}
				if assign {
					_ = b.Assign(source)
				} else {
					constructed, _ = NewImpliedBound(h.m, &v.X, 7, true, false, 9, true)
				}
				if b.calls.comparison != 0 || source.calls.comparison != 0 || constructed != nil && constructed.calls.comparison != 0 {
					t.Fatal("adopted caller comparison")
				}
				// Exact equality compares canonical integer numerator/denominator
				// without a rational cross-product or comparison invocation.
				equal, e := source.Equal(source)
				h.must(e)
				if !equal {
					t.Fatal("canonical equality with an older caller comparison")
				}
				h.must(WriteLConstraintKind(io.Discard, source.Kind()))
			}
			if constructed != nil {
				h.must(constructed.Close())
			}
			h.must(b.Close())
			h.must(source.Close())
			if older.Active && h.m.ComparisonInvocationState() != older {
				t.Fatal("consumed caller comparison")
			}
			h.must(caller.cleanupRaw(h.m))
			h.must(caller.cleanupRoots(h.m))
			lpExactVectorNamedCensus(t, h, nil)
		}
		if !found {
			t.Fatal("missing actual older comparison", assign)
		}
	}
}
