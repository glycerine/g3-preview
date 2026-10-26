package lp

import (
	"errors"
	"fmt"
	"math/big"
	"strings"
	"testing"
)

func lpDispatchModel(t *testing.T, h *lpPairHarness, pivot bool, c0, c1 string) (*LARSolver, uint32) {
	t.Helper()
	s, err := NewLARSolver(h.m)
	h.must(err)
	if pivot {
		s.Settings.SimplexStrategy = StrategyLU
	}
	for _, ext := range []uint32{31, 67} {
		_, err = s.AddVar(ext)
		h.must(err)
	}
	c := h.pair(c0, c1)
	constant := h.pair("5", "0")
	inputs := NewSparseRationalVector(h.m)
	h.must(inputs.Append(0, &c.X))
	h.must(inputs.Append(1, &c.Y))
	term, err := s.AddTerm(inputs.Data, &constant.X)
	h.must(err)
	if pivot {
		h.must(s.AddRowFromTermNoConstraint(s.OriginalTerms[0], term))
		v := s.Core.RPrimal.PivotRow
		h.must(v.resizeData(3))
		for j, value := range []*big.Rat{mustLPRat(t, c0), mustLPRat(t, c1)} {
			negative := new(big.Rat).Neg(value)
			h.must(h.m.Copy(&v.Data[j], &h.pair(negative.RatString(), "0").X))
			v.Index = append(v.Index, uint32(j))
		}
	}
	lo := h.pair("-3", "2")
	hi := h.pair("7", "11")
	for j := uint32(0); j < 2; j++ {
		low, high := &lo.X, &hi.X
		if j == 1 {
			low, high = &lo.Y, &hi.Y
		}
		_, err = s.AddVarBound(j, LConstraintGE, low)
		h.must(err)
		kind := LConstraintLE
		if j == 1 {
			kind = LConstraintLT
		}
		_, err = s.AddVarBound(j, kind, high)
		h.must(err)
	}
	h.must(inputs.Close())
	return s, term
}

func lpDispatchRun(s *LARSolver, pivot bool, p *LARBoundPropagator) error {
	if pivot {
		return s.AnalyzeNewBoundsOnRow(0, p)
	}
	return s.AnalyzeNewBoundsOnRowTableau(0, p)
}

func TestLARRowDispatchIndependentIntervals(t *testing.T) {
	h := newLPPairHarness(t)
	for _, pivot := range []bool{false, true} {
		for world := 0; world < 80; world++ {
			c0, c1 := int64(world%11+1), int64(world%7+1)
			if world&1 != 0 {
				c0 = -c0
			}
			if world&2 != 0 {
				c1 = -c1
			}
			s, term := lpDispatchModel(t, h, pivot, fmt.Sprint(c0), fmt.Sprint(c1))
			p := s.NewBoundPropagator()
			h.must(lpDispatchRun(s, pivot, p))
			low, high := int64(5), int64(5)
			for j, c := range []int64{c0, c1} {
				a, b := []int64{-3, 2}[j]*c, []int64{7, 11}[j]*c
				if a > b {
					a, b = b, a
				}
				low += a
				high += b
			}
			if len(p.FoundBounds) != 2 {
				t.Fatal(pivot, world, "bounds", len(p.FoundBounds))
			}
			for _, b := range p.FoundBounds {
				want, strict := high, c1 > 0
				if b.IsLowBound {
					want, strict = low, c1 < 0
				}
				if b.Column != term || b.RowOrTermIndex != 0 || b.Strict != strict {
					t.Fatal("row evidence", pivot, world, b)
				}
				lpSparseRationalCheck(t, h, &b.Bound, big.NewRat(want, 1))
			}
			if s.pendingRowAnalyzer != nil || s.pendingRowIterator != nil {
				t.Fatal("successful lexical owners not closed")
			}
			h.must(lpDispatchRun(s, pivot, p))
			if len(p.FoundBounds) != 2 {
				t.Fatal("repeat analysis changed strongest bounds")
			}
			h.must(p.Close())
			h.must(s.Close())
		}
	}
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLARRowDispatchGuardsAndCutoff(t *testing.T) {
	h := newLPPairHarness(t)
	for _, pivot := range []bool{false, true} {
		s, _ := lpDispatchModel(t, h, pivot, "2", "3")
		p := s.NewBoundPropagator()
		if pivot {
			// An oversized tableau row returns before both the mode assertion
			// and the propagator access, even when supplied in LU mode.
			s.Settings.MaxRowLengthForBoundPropagation = 2
			h.must(s.AnalyzeNewBoundsOnRowTableau(0, nil))
			if s.pendingRowIterator != nil || s.stopped {
				t.Fatal("cutoff constructed lexical owner")
			}
		} else {
			s.Settings.MaxRowLengthForBoundPropagation = 3
			h.must(s.AnalyzeNewBoundsOnRowTableau(0, p))
			if len(p.FoundBounds) != 2 {
				t.Fatal("cutoff must be strictly greater")
			}
		}
		var err error
		if pivot {
			err = s.AnalyzeNewBoundsOnRow(99, p)
		} else {
			err = s.AnalyzeNewBoundsOnRowTableau(99, p)
		}
		if !errors.Is(err, ErrBounds) {
			t.Fatal(err)
		}
		if e := lpDispatchRun(s, pivot, p); !errors.Is(e, ErrExactVectorPending) {
			t.Fatal("stopped entry accepted", e)
		}
		h.must(s.Close())
		h.must(p.Close())
	}
	for _, pivot := range []bool{false, true} {
		s, _ := lpDispatchModel(t, h, pivot, "2", "3")
		if e := lpDispatchRun(s, pivot, nil); !errors.Is(e, ErrBounds) {
			t.Fatal("nil propagator", e)
		}
		h.must(s.Close())
	}
	lpExactVectorNamedCensus(t, h, nil)
}

func TestLARRowDispatchPhysicalFailureOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	large := "1" + strings.Repeat("0", 999) + "7"
	for _, direct := range []bool{false, true} {
		for _, pivot := range []bool{false, true} {
			failures, published := 0, 0
			success := false
			for ordinal := int64(0); ordinal < 1000; ordinal++ {
				done := false
				t.Run(fmt.Sprintf("%v/%v/%d", direct, pivot, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					s, _ := lpDispatchModel(t, h, pivot, large, "-"+large)
					p := s.NewBoundPropagator()
					h.budget.FailAfter(ordinal)
					err := lpDispatchRun(s, pivot, p)
					h.budget.FailAfter(-1)
					if err != nil {
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						failures++
						if len(p.FoundBounds) != 0 {
							published++
						}
						raw, division, comparison := h.m.RawInvocationState(), h.m.DivisionInvocationState(), h.m.ComparisonInvocationState()
						if e := lpDispatchRun(s, pivot, p); !errors.Is(e, ErrExactVectorPending) {
							t.Fatal("stopped entry", e)
						}
						if raw != h.m.RawInvocationState() || division != h.m.DivisionInvocationState() || comparison != h.m.ComparisonInvocationState() {
							t.Fatal("stopped entry changed numeric ownership")
						}
						if s.calls.pending() {
							t.Fatal("model adopted child numeric calls")
						}
					} else {
						done = true
					}
					before := len(p.FoundBounds)
					h.must(s.Close())
					if len(p.FoundBounds) != before {
						t.Fatal("model destroyed borrowed propagator")
					}
					h.must(p.Close())
					lpExactVectorNamedCensus(t, h, nil)
				})
				if done {
					success = true
					break
				}
			}
			if !success || failures == 0 || published == 0 {
				t.Fatal(direct, pivot, success, failures, published)
			}
		}
	}
}

func TestLARRowDispatchModeAndForeignOwner(t *testing.T) {
	h := newLPPairHarness(t)
	other := newLPPairHarness(t)
	for _, pivot := range []bool{false, true} {
		s, _ := lpDispatchModel(t, h, pivot, "2", "3")
		p := s.NewBoundPropagator()
		if pivot {
			s.Settings.SimplexStrategy = StrategyTableauRows
		} else {
			s.Settings.SimplexStrategy = StrategyLU
		}
		err := lpDispatchRun(s, pivot, p)
		if sourceDebugEnabled {
			if !errors.Is(err, ErrAssertion) || (s.pendingRowIterator != nil) == pivot || s.pendingRowAnalyzer != nil {
				t.Fatal("source assertion/construction order", pivot, err)
			}
		} else {
			h.must(err)
			if len(p.FoundBounds) != 2 {
				t.Fatal("native assertion must be erased")
			}
		}
		h.must(s.Close())
		h.must(p.Close())

		s, _ = lpDispatchModel(t, h, pivot, "2", "3")
		foreign, _ := lpDispatchModel(t, other, pivot, "2", "3")
		fp := foreign.NewBoundPropagator()
		if e := lpDispatchRun(s, pivot, fp); !errors.Is(e, ErrExactVectorOwner) {
			t.Fatal("foreign propagator accepted", e)
		}
		if s.pendingRowAnalyzer == nil || s.pendingRowIterator == nil || s.calls.pending() || len(fp.FoundBounds) != 0 {
			t.Fatal("foreign-owner constructor prefix")
		}
		h.must(s.Close())
		other.must(fp.Close())
		other.must(foreign.Close())
	}
	lpExactVectorNamedCensus(t, h, nil)
	lpExactVectorNamedCensus(t, other, nil)
}
