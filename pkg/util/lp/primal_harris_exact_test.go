package lp

import (
	"fmt"
	"math/big"
	"strings"
	"testing"
)

func TestLPExactHarrisIndependentPairOracle(t *testing.T) {
	h := newLPPairHarness(t)
	for world := 0; world < 240; world++ {
		s, e := NewLARSolver(h.m)
		h.must(e)
		m := s.Core.RPrimal
		x, y := big.NewRat(int64(world%23-11), 17), big.NewRat(int64(world%29-14), 19)
		if world%3 == 0 {
			x.SetInt64(0)
		}
		eps := big.NewRat(int64(world%11-5), 10000000)
		bound := h.pair(x.RatString(), y.RatString())
		out := h.pair("23", "29")
		if world%2 == 0 {
			out = bound
		}
		h.must(h.m.SetString(&m.ConvertedHarrisEpsilon, eps.RatString()))
		negative := x.Sign() < 0 || (x.Sign() == 0 && y.Sign() < 0)
		if negative {
			x.Neg(x)
			y.Neg(y)
		}
		wantX := new(big.Rat).Quo(new(big.Rat).Mul(new(big.Rat).Add(big.NewRat(1, 1), new(big.Rat).Quo(x, big.NewRat(10, 1))), eps), big.NewRat(3, 1))
		wantY := new(big.Rat).Quo(new(big.Rat).Mul(new(big.Rat).Quo(y, big.NewRat(10, 1)), eps), big.NewRat(3, 1))
		h.must(m.HarrisEpsForBound(bound, out))
		h.check(out, wantX, wantY)
		h.must(m.ZeroHarrisEps())
		h.must(m.HarrisEpsForBound(bound, out))
		h.check(out, big.NewRat(0, 1), big.NewRat(0, 1))
		h.must(s.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPExactHarrisPhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, direct := range []bool{false, true} {
		for _, zero := range []bool{false, true} {
			for _, negative := range []bool{false, true} {
				success, failures, own := false, 0, 0
				for ordinal := int64(0); ordinal < 800; ordinal++ {
					done := false
					t.Run(fmt.Sprintf("%t/%t/%t/%d", direct, zero, negative, ordinal), func(t *testing.T) {
						var h *lpPairHarness
						if direct {
							h = lpNewDirectSparseHarness(t)
						} else {
							h = newLPPairHarness(t)
						}
						s, e := NewLARSolver(h.m)
						h.must(e)
						m := s.Core.RPrimal
						huge := strings.Repeat("1234567890", 12)
						bound, out := h.pair(huge+"1/7", huge+"3/11"), h.pair("23", "29")
						if negative {
							h.must(h.m.Neg(&bound.X))
						}
						if !zero {
							h.must(h.m.SetString(&m.ConvertedHarrisEpsilon, huge+"9/17"))
						}
						h.budget.FailAfter(ordinal)
						err := m.HarrisEpsForBound(bound, out)
						h.budget.FailAfter(-1)
						if err == nil {
							success, done = true, true
						} else {
							failures++
							if !lpExactPriorityPhysicalRefusal(err) {
								t.Fatal(err)
							}
							if m.calls.pending() {
								own++
							}
							if m.PairCoreSolverBase.calls.pending() {
								t.Fatal("base adopted primal expression")
							}
							if e = m.ZeroHarrisEps(); e != ErrExactVectorPending {
								t.Fatal("lost stopped primal", e)
							}
						}
						h.must(s.Close())
						lpExactVectorNamedCensus(t, h, nil)
					})
					if done {
						break
					}
				}
				if !success || failures == 0 || own == 0 {
					t.Fatal("incomplete physical campaign", direct, zero, success, failures, own)
				}
			}
		}
	}
}

func TestLPExactHarrisGuards(t *testing.T) {
	for _, missing := range []string{"bound", "output"} {
		h := newLPPairHarness(t)
		s, e := NewLARSolver(h.m)
		h.must(e)
		bound, out := h.pair("-2", "7"), h.pair("11", "13")
		if missing == "bound" {
			bound = nil
		} else {
			out = nil
		}
		if e = s.Core.RPrimal.HarrisEpsForBound(bound, out); e != ErrBounds {
			t.Fatal(e)
		}
		if e = s.Core.RPrimal.HarrisEpsForBound(bound, out); e != ErrExactVectorPending {
			t.Fatal(e)
		}
		h.must(s.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLPExactHarrisPreservesOlderComparisonOwner(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, op := range []string{"zero", "nonzero"} {
		matched := false
		for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
			t.Run(fmt.Sprintf("%s/%d", op, ordinal), func(t *testing.T) {
				h := newLPPairHarness(t)
				s, e := NewLARSolver(h.m)
				h.must(e)
				m := s.Core.RPrimal
				v := h.pair(strings.Repeat("9876543210", 100)+"/7", strings.Repeat("1234567890", 100)+"/13")
				bound, out := h.pair("-1/7", "1/11"), h.pair("23", "29")
				if op == "nonzero" {
					h.must(h.m.SetString(&m.ConvertedHarrisEpsilon, "1/10000000"))
				}
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
					err = m.HarrisEpsForBound(bound, out)
					if m.calls.comparison != 0 || m.PairCoreSolverBase.calls.comparison != 0 {
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
