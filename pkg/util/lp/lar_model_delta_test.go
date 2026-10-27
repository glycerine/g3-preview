// Source: src/util/lp/lar_core_solver.h:791-801.
package lp

import (
	"fmt"
	"math/big"
	"strings"
	"testing"
)

func lpLARModelDeltaFixture(t *testing.T, h *lpPairHarness, kind ColumnType) *LARSolver {
	t.Helper()
	s, err := NewLARSolver(h.m)
	h.must(err)
	_, err = s.AddVar(17)
	h.must(err)
	s.Bounds.Types.values[0] = kind
	return s
}

func TestLARModelDeltaIndependentBounds(t *testing.T) {
	for _, kind := range []ColumnType{ColumnFree, ColumnLowBound, ColumnUpperBound, ColumnFixed, ColumnBoxed} {
		for world := 0; world < 80; world++ {
			t.Run(fmt.Sprintf("%d/%d", kind, world), func(t *testing.T) {
				h := newLPPairHarness(t)
				s := lpLARModelDeltaFixture(t, h, kind)
				initial := big.NewRat(int64(1+world%11), 19)
				x, y := big.NewRat(int64(world-39), 7), big.NewRat(int64(world%9-4), 13)
				lowGap, upGap := big.NewRat(int64(1+world%3), 101), big.NewRat(int64(1+world%5), 103)
				lowSlope, upSlope := big.NewRat(int64(world%7-2), 3), big.NewRat(int64(world%11-3), 5)
				lx, ly := new(big.Rat).Sub(x, lowGap), new(big.Rat).Add(y, lowSlope)
				ux, uy := new(big.Rat).Add(x, upGap), new(big.Rat).Sub(y, upSlope)
				if kind == ColumnFixed {
					lx, ly, ux, uy = new(big.Rat).Set(x), new(big.Rat).Set(y), new(big.Rat).Set(x), new(big.Rat).Set(y)
				}
				h.must(s.Core.RX.Data[0].Copy(h.m, h.pair(x.RatString(), y.RatString())))
				h.must(s.Bounds.Low.values[0].Copy(h.m, h.pair(lx.RatString(), ly.RatString())))
				h.must(s.Bounds.Upper.values[0].Copy(h.m, h.pair(ux.RatString(), uy.RatString())))
				want := new(big.Rat).Set(initial)
				low := kind == ColumnLowBound || kind == ColumnBoxed || kind == ColumnFixed
				up := kind == ColumnUpperBound || kind == ColumnBoxed || kind == ColumnFixed
				if low && lx.Cmp(x) < 0 && ly.Cmp(y) > 0 {
					limit := new(big.Rat).Quo(new(big.Rat).Sub(x, lx), new(big.Rat).Sub(ly, y))
					if limit.Cmp(want) < 0 {
						want.Set(limit)
					}
				}
				if up && x.Cmp(ux) < 0 && y.Cmp(uy) > 0 {
					limit := new(big.Rat).Quo(new(big.Rat).Sub(ux, x), new(big.Rat).Sub(y, uy))
					if limit.Cmp(want) < 0 {
						want.Set(limit)
					}
				}
				delta := h.pair(initial.RatString(), "77")
				// The output may alias the initial input; its old value survives
				// until all actual child updates have finished.
				h.must(s.Core.FindDeltaForStrictBounds(&delta.X, &delta.X))
				lpSparseRationalCheck(t, h, &delta.X, want)
				h.check(delta, want, big.NewRat(77, 1))
				concrete := new(big.Rat).Add(x, new(big.Rat).Mul(want, y))
				if low && new(big.Rat).Add(lx, new(big.Rat).Mul(want, ly)).Cmp(concrete) > 0 || up && concrete.Cmp(new(big.Rat).Add(ux, new(big.Rat).Mul(want, uy))) > 0 {
					t.Fatal("independent concrete bound validation")
				}
				h.must(s.Close())
				lpExactVectorNamedCensus(t, h, nil)
			})
		}
	}
}

func TestLARModelDeltaReachedPrefixes(t *testing.T) {
	for _, phase := range []string{"stopped", "nil-initial", "nil-output", "types", "low", "upper", "x-low", "x-upper", "late-upper", "free-short-x", "unknown", "invalid-low"} {
		t.Run(phase, func(t *testing.T) {
			h := newLPPairHarness(t)
			s := lpLARModelDeltaFixture(t, h, ColumnBoxed)
			c := s.Core
			h.must(c.RX.Data[0].Copy(h.m, h.pair("1", "0")))
			h.must(s.Bounds.Low.values[0].Copy(h.m, h.pair("0", "4")))
			h.must(s.Bounds.Upper.values[0].Copy(h.m, h.pair("2", "0")))
			types, low, upper, x := s.Bounds.Types.values, s.Bounds.Low.values, s.Bounds.Upper.values, c.RX.Data
			arg, result := h.pair("1/2", "19"), h.pair("29", "0")
			initial, out := &arg.X, &result.X
			want := error(ErrBounds)
			switch phase {
			case "stopped":
				c.stopped = true
				want = ErrExactVectorPending
			case "nil-initial":
				initial = nil
			case "nil-output":
				out = nil
			case "types":
				s.Bounds.Types.values = nil
			case "low":
				s.Bounds.Low.values = nil
			case "upper", "late-upper":
				s.Bounds.Upper.values = nil
				if phase == "upper" {
					s.Bounds.Types.values[0] = ColumnUpperBound
				}
			case "x-low", "x-upper", "free-short-x":
				c.RX.Data = nil
				if phase == "x-upper" {
					s.Bounds.Types.values[0] = ColumnUpperBound
				} else if phase == "free-short-x" {
					s.Bounds.Types.values[0] = ColumnFree
					want = nil
				}
			case "unknown":
				s.Bounds.Types.values[0] = ColumnType(255)
				want = nil
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			case "invalid-low":
				h.must(s.Bounds.Low.values[0].Copy(h.m, h.pair("2", "0")))
				want = nil
				if sourceDebugEnabled {
					want = ErrAssertion
				}
			}
			if err := c.FindDeltaForStrictBounds(initial, out); err != want {
				t.Fatal("reached source prefix", phase, err, want)
			}
			if want != nil {
				lpSparseRationalCheck(t, h, &result.X, big.NewRat(29, 1))
				if c.FindDeltaForStrictBounds(&arg.X, &result.X) != ErrExactVectorPending {
					t.Fatal("stopped retry")
				}
			} else {
				lpSparseRationalCheck(t, h, &result.X, big.NewRat(1, 2))
			}
			if phase == "late-upper" {
				lpSparseRationalCheck(t, h, &c.strictDelta.Data[0], big.NewRat(1, 4))
			}
			s.Bounds.Types.values, s.Bounds.Low.values, s.Bounds.Upper.values, c.RX.Data = types, low, upper, x
			h.must(s.Close())
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
}

func TestLARModelDeltaPhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, profile := range []string{"lower", "upper", "initial-copy"} {
		for _, direct := range []bool{false, true} {
			done, failures := false, 0
			for ordinal := int64(0); ordinal < 1200 && !done; ordinal++ {
				t.Run(fmt.Sprintf("%s/%v/%d", profile, direct, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					s := lpLARModelDeltaFixture(t, h, ColumnLowBound)
					huge := strings.Repeat("1234567890", 20)
					h.must(s.Bounds.Low.values[0].Copy(h.m, h.pair("0", huge+"/7")))
					h.must(s.Core.RX.Data[0].Copy(h.m, h.pair("1/13", "0")))
					arg, result := h.pair("1/2", "0"), h.pair("29", "0")
					want := new(big.Rat).Quo(big.NewRat(1, 13), mustLPRat(t, huge+"/7"))
					if profile == "upper" {
						s.Bounds.Types.values[0] = ColumnUpperBound
						h.must(s.Core.RX.Data[0].Copy(h.m, h.pair("0", huge+"/7")))
						h.must(s.Bounds.Upper.values[0].Copy(h.m, h.pair("1/13", "0")))
					} else if profile == "initial-copy" {
						s.Bounds.Types.values[0] = ColumnFree
						want = mustLPRat(t, huge+"/7")
						h.must(h.m.SetString(&arg.X, want.RatString()))
					}
					h.budget.FailAfter(ordinal)
					err := s.Core.FindDeltaForStrictBounds(&arg.X, &result.X)
					h.budget.FailAfter(-1)
					if err == nil {
						done = true
						lpSparseRationalCheck(t, h, &result.X, want)
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(err) || s.Core.FindDeltaForStrictBounds(&arg.X, &result.X) != ErrExactVectorPending {
							t.Fatal("actual physical refusal", err)
						}
					}
					h.must(s.Close())
					lpExactVectorNamedCensus(t, h, nil)
				})
			}
			if !done || direct && failures == 0 {
				t.Fatal("failure campaign did not reach success", direct, failures)
			}
		}
	}
}

func TestLARModelDeltaMixedColumnsAndLateFailure(t *testing.T) {
	for _, invalid := range []bool{false, true} {
		h := newLPPairHarness(t)
		s := lpLARModelDeltaFixture(t, h, ColumnLowBound)
		_, err := s.AddVar(31)
		h.must(err)
		_, err = s.AddVar(47)
		h.must(err)
		s.Bounds.Types.values[2] = ColumnUpperBound
		h.must(s.Core.RX.Data[0].Copy(h.m, h.pair("1/3", "0")))
		h.must(s.Bounds.Low.values[0].Copy(h.m, h.pair("0", "2")))
		h.must(s.Core.RX.Data[2].Copy(h.m, h.pair("2/7", "5")))
		h.must(s.Bounds.Upper.values[2].Copy(h.m, h.pair("3/7", "0")))
		bounds := s.Bounds.Upper.values
		if invalid {
			s.Bounds.Upper.values = bounds[:2]
		}
		v := h.pair("1/2", "29")
		err = s.Core.FindDeltaForStrictBounds(&v.X, &v.Y)
		if invalid {
			if err != ErrBounds {
				t.Fatal("third column extent", err)
			}
			lpSparseRationalCheck(t, h, &s.Core.strictDelta.Data[0], big.NewRat(1, 6))
			lpSparseRationalCheck(t, h, &v.Y, big.NewRat(29, 1))
		} else {
			h.must(err)
			lpSparseRationalCheck(t, h, &v.Y, big.NewRat(1, 35))
		}
		s.Bounds.Upper.values = bounds
		h.must(s.Close())
		lpExactVectorNamedCensus(t, h, nil)
	}
}

func TestLARModelDeltaPreservesOlderComparison(t *testing.T) {
	matched := false
	for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			h := newLPPairHarness(t)
			s := lpLARModelDeltaFixture(t, h, ColumnLowBound)
			h.must(s.Core.RX.Data[0].Copy(h.m, h.pair("1/13", "0")))
			h.must(s.Bounds.Low.values[0].Copy(h.m, h.pair("0", "123456789012345678901/7")))
			arg := h.pair("1/2", "29")
			v := h.pair(strings.Repeat("9876543210", 100)+"/7", strings.Repeat("1234567890", 100)+"/13")
			var caller exactEtaCalls
			before, err := caller.begin(h.m)
			h.must(err)
			h.budget.FailAfter(ordinal)
			_, err = h.m.Compare(&v.X, &v.Y)
			h.budget.FailAfter(-1)
			err = caller.after(h.m, before, err)
			older := h.m.ComparisonInvocationState()
			if older.Active {
				matched = true
				if !lpExactPriorityPhysicalRefusal(err) {
					t.Fatal(err)
				}
				if err = s.Core.FindDeltaForStrictBounds(&arg.X, &arg.Y); err == nil || s.Core.calls.comparison != 0 {
					t.Fatal("callee adopted older comparison", err)
				}
			}
			h.must(s.Close())
			if older.Active && h.m.ComparisonInvocationState() != older {
				t.Fatal("callee consumed older comparison")
			}
			h.must(caller.cleanupRaw(h.m))
			h.must(caller.cleanupRoots(h.m))
			lpExactVectorNamedCensus(t, h, nil)
		})
	}
	if !matched {
		t.Fatal("no actual pending comparison")
	}
}
