//go:build g3debug

// Source: src/util/lp/lp_settings.hpp:94-103; permutation_matrix.hpp:104,137.
package lp

import (
	"fmt"
	"strings"
	"testing"
)

func TestLPPermutationExactVectorEqualityCopiedLeftAndEarlyReturn(t *testing.T) {
	for world := 0; world < 30; world++ {
		h := newLPPairHarness(t)
		s := newExactPermutationScratch(h.m, 0)
		s.debug = NewIndexedRationalVector(h.m, 5)
		w := NewIndexedRationalVector(h.m, 5)
		mismatch := world%6 - 1
		for i := 0; i < 5; i++ {
			p := h.pair(strings.Repeat("1234567890", 20)+fmt.Sprint(world*7+i)+"/13", "0")
			h.must(s.debug.SetValue(&p.X, uint32(i)))
			right := p
			if i == mismatch {
				right = h.pair("-1/17", "0")
			}
			h.must(w.SetValue(&right.X, uint32(i)))
		}
		equal, e := s.equal(w)
		h.must(e)
		if equal != (mismatch < 0) {
			t.Fatal(world, equal)
		}
		for i := range s.difference.Data {
			zero, e := h.m.IsZero(&s.difference.Data[i])
			h.must(e)
			if !zero {
				t.Fatal("expression temporary outlived element", i)
			}
		}
		h.must(s.close())
		h.must(w.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPPermutationExactVectorEqualityLengthAndEmpty(t *testing.T) {
	for _, sizes := range [][2]uint32{{0, 0}, {1, 0}, {0, 1}, {2, 1}} {
		h := newLPPairHarness(t)
		s := newExactPermutationScratch(h.m, 0)
		s.debug = NewIndexedRationalVector(h.m, sizes[0])
		w := NewIndexedRationalVector(h.m, sizes[1])
		equal, e := s.equal(w)
		h.must(e)
		if equal != (sizes[0] == sizes[1]) {
			t.Fatal(sizes, equal)
		}
		h.must(s.close())
		h.must(w.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPPermutationExactVectorEqualityPhysicalRefusals(t *testing.T) {
	for _, direct := range []bool{false, true} {
		failures, succeeded := 0, false
		for ordinal := int64(0); ordinal < 64 && !succeeded; ordinal++ {
			if !t.Run(fmt.Sprintf("direct=%v/%d", direct, ordinal), func(t *testing.T) {
				var h *lpPairHarness
				if direct {
					h = lpNewDirectSparseHarness(t)
				} else {
					h = newLPPairHarness(t)
				}
				p := h.pair(strings.Repeat("1234567890", 100)+"/7", "0")
				s := newExactPermutationScratch(h.m, 0)
				s.debug = NewIndexedRationalVector(h.m, 1)
				w := NewIndexedRationalVector(h.m, 1)
				h.must(s.debug.SetValue(&p.X, 0))
				h.must(w.SetValue(&p.X, 0))
				t.Cleanup(func() { h.budget.FailAfter(-1); h.must(s.close()); h.must(w.Close()) })
				h.budget.FailAfter(ordinal)
				equal, e := s.equal(w)
				h.budget.FailAfter(-1)
				if e == nil {
					if !equal {
						t.Fatal("independent equal operands")
					}
					succeeded = true
				} else {
					if !lpExactPriorityPhysicalRefusal(e) {
						t.Fatal(e)
					}
					failures++
				}
				h.must(s.close())
				h.must(w.Close())
				lpExactVectorNamedCensus(t, h, nil)
			}) {
				t.FailNow()
			}
		}
		if !succeeded || failures == 0 {
			t.Fatal("incomplete copied-left refusal campaign", direct, failures, succeeded)
		}
	}
}
