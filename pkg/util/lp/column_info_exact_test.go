package lp

import (
	"fmt"
	"math/big"
	"strings"
	"testing"
)

func TestLPRationalColumnIndependentFlagsAndShift(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for world := 0; world < 160; world++ {
		for mask := 0; mask < 8; mask++ {
			h := newLPPairHarness(t)
			c := NewRationalColumnInfo(h.m, uint32(world))
			out := h.pair("91", "0")
			low, upper, fixed, cost := big.NewRat(int64(world-81), 17), big.NewRat(int64(world+19), 13), big.NewRat(int64(world*3-7), 19), big.NewRat(int64(world*7-31), 23)
			wantShift := new(big.Rat)
			if mask&1 != 0 {
				h.must(c.SetLowBound(&h.pair(low.RatString(), "0").X))
				wantShift.Set(low)
			}
			if mask&2 != 0 {
				h.must(c.SetUpperBound(&h.pair(upper.RatString(), "0").X))
				if mask&1 == 0 {
					wantShift.Set(upper)
				}
			}
			if mask&4 != 0 {
				h.must(c.SetFixedValue(&h.pair(fixed.RatString(), "0").X))
				wantShift.Set(fixed)
			}
			h.must(c.SetCost(&h.pair(cost.RatString(), "0").X))
			c.LowStrict, c.UpperStrict = world%2 == 0, world%3 == 0
			h.must(c.GetCost(&out.X))
			lpSparseRationalCheck(t, h, &out.X, cost)
			wantType := ColumnFree
			if mask&1 != 0 {
				wantType = ColumnLowBound
			}
			if mask&2 != 0 {
				wantType = ColumnUpperBound
				if mask&1 != 0 {
					wantType = ColumnBoxed
				}
			}
			if mask&4 != 0 {
				wantType = ColumnFixed
			}
			if c.ColumnType() != wantType || c.IsFree() != (mask&3 == 0) || c.IsFlipped() != (mask&3 == 2) || c.Index != uint32(world) {
				t.Fatal(world, mask, c.ColumnType(), c.IsFree(), c.IsFlipped())
			}
			h.must(c.Shift(&out.X))
			lpSparseRationalCheck(t, h, &out.X, wantShift)
			if c.LowSet {
				h.must(c.GetLowBound(&out.X))
				lpSparseRationalCheck(t, h, &out.X, low)
			}
			if c.UpperSet {
				h.must(c.GetUpperBound(&out.X))
				lpSparseRationalCheck(t, h, &out.X, upper)
			}
			if c.Fixed {
				h.must(c.GetFixedValue(&out.X))
				lpSparseRationalCheck(t, h, &out.X, fixed)
			}
			err := c.AdjustedUpperBound(&out.X)
			if sourceDebugEnabled && mask&3 != 3 {
				if err != ErrAssertion {
					t.Fatal(mask, err)
				}
			} else {
				h.must(err)
				lo, hi := new(big.Rat), new(big.Rat)
				if c.LowSet {
					lo.Set(low)
				}
				if c.UpperSet {
					hi.Set(upper)
				}
				lpSparseRationalCheck(t, h, &out.X, new(big.Rat).Sub(hi, lo))
			}
			h.must(c.Close())
			h.must(c.Close())
			lpExactVectorNamedCensus(t, h, nil)
		}
	}
}
func TestLPRationalColumnNativeToleranceAndDebugConstructor(t *testing.T) {
	for _, flags := range []int{0, 1, 2, 3, 4} {
		for _, op := range []string{"low", "upper", "both"} {
			for _, value := range []string{"-1/100000000", "0", "1/2", "1", "100000001/100000000"} {
				h := newLPPairHarness(t)
				c := NewRationalColumnInfo(h.m, 0)
				if flags&1 != 0 {
					h.must(c.SetLowBound(&h.pair("0", "0").X))
				}
				if flags&2 != 0 {
					h.must(c.SetUpperBound(&h.pair("1", "0").X))
				}
				if flags == 4 {
					h.must(c.SetFixedValue(&h.pair("99", "0").X))
				}
				c.LowStrict, c.UpperStrict = true, true
				v := h.pair(value, "0")
				var holds bool
				var err error
				switch op {
				case "low":
					holds, err = c.LowBoundHolds(&v.X)
				case "upper":
					holds, err = c.UpperBoundHolds(&v.X)
				case "both":
					holds, err = c.BoundsHold(&v.X)
				}
				reaches := flags&1 != 0 && op != "upper" || flags&2 != 0 && op != "low"
				if sourceDebugEnabled && reaches {
					if err != ErrAssertion || holds {
						t.Fatal(flags, op, value, holds, err)
					}
				} else {
					h.must(err)
					x := mustLPRat(t, value)
					want := true
					if op != "upper" && flags&1 != 0 {
						want = x.Sign() >= 0
					}
					if op != "low" && flags&2 != 0 {
						want = want && x.Cmp(big.NewRat(1, 1)) <= 0
					}
					if holds != want {
						t.Fatal(flags, op, value, holds, want)
					}
				}
				h.must(c.Close())
				lpExactVectorNamedCensus(t, h, nil)
			}
		}
	}
}
func TestLPRationalColumnGetterAssertionsAndEntryGuards(t *testing.T) {
	for _, phase := range []string{"low-unset", "upper-unset", "fixed-unset", "nil-low", "nil-upper", "nil-cost", "nil-fixed", "nil-out", "nil-adjusted", "nil-low-check", "nil-upper-check", "nil-both-check", "stopped", "nil-manager"} {
		h := newLPPairHarness(t)
		c := NewRationalColumnInfo(h.m, InvalidColumnIndex)
		out := h.pair("91", "0")
		want := error(ErrBounds)
		var err error
		switch phase {
		case "low-unset":
			err = c.GetLowBound(&out.X)
			want = sourceAssertionFailure()
		case "upper-unset":
			err = c.GetUpperBound(&out.X)
			want = sourceAssertionFailure()
		case "fixed-unset":
			err = c.GetFixedValue(&out.X)
			want = sourceAssertionFailure()
		case "nil-low":
			err = c.SetLowBound(nil)
		case "nil-upper":
			err = c.SetUpperBound(nil)
		case "nil-cost":
			err = c.SetCost(nil)
		case "nil-fixed":
			err = c.SetFixedValue(nil)
		case "nil-out":
			err = c.GetCost(nil)
		case "nil-adjusted":
			err = c.AdjustedUpperBound(nil)
		case "nil-low-check":
			_, err = c.LowBoundHolds(nil)
		case "nil-upper-check":
			_, err = c.UpperBoundHolds(nil)
		case "nil-both-check":
			_, err = c.BoundsHold(nil)
		case "stopped":
			c.stopped = true
			err = c.SetCost(&out.X)
			want = ErrExactVectorPending
		case "nil-manager":
			c.manager = nil
			err = c.SetCost(&out.X)
			want = ErrExactVectorOwner
		}
		if err != want || c.LowSet || c.UpperSet || c.Fixed {
			t.Fatal(phase, err, want)
		}
		if err == nil {
			lpSparseRationalCheck(t, h, &out.X, new(big.Rat))
		}
		h.must(c.Close())
		if c.SetCost(&out.X) != ErrExactVectorPending {
			t.Fatal("closed column")
		}
		lpExactVectorNamedCensus(t, h, nil)
	}
}
func TestLPRationalColumnPhysicalOwnership(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}
	for _, direct := range []bool{false, true} {
		for _, phase := range []string{"low", "upper", "cost", "fixed", "get", "adjusted", "low-check", "upper-check", "both-check", "free-check"} {
			done, failures, sawFixedBefore, sawFixedAfter := false, 0, false, false
			for ordinal := int64(0); ordinal < 350 && !done; ordinal++ {
				if !t.Run(fmt.Sprintf("%v/%s/%d", direct, phase, ordinal), func(t *testing.T) {
					var h *lpPairHarness
					if direct {
						h = lpNewDirectSparseHarness(t)
					} else {
						h = newLPPairHarness(t)
					}
					c := NewRationalColumnInfo(h.m, 17)
					large := h.pair(strings.Repeat("1234567890", 20)+"7/13", "0")
					out := h.pair("91", "0")
					if phase == "get" {
						h.must(c.SetCost(&large.X))
					}
					if phase == "adjusted" {
						h.must(c.SetLowBound(&h.pair("-7/13", "0").X))
						h.must(c.SetUpperBound(&large.X))
					}
					if phase == "low-check" || phase == "both-check" {
						h.must(c.SetLowBound(&large.X))
					}
					if phase == "upper-check" || phase == "both-check" {
						h.must(c.SetUpperBound(&large.X))
					}
					h.budget.FailAfter(ordinal)
					var err error
					switch phase {
					case "low":
						err = c.SetLowBound(&large.X)
					case "upper":
						err = c.SetUpperBound(&large.X)
					case "cost":
						err = c.SetCost(&large.X)
					case "fixed":
						err = c.SetFixedValue(&large.X)
					case "get":
						err = c.GetCost(&out.X)
					case "adjusted":
						err = c.AdjustedUpperBound(&out.X)
					case "low-check":
						_, err = c.LowBoundHolds(&large.X)
					case "upper-check":
						_, err = c.UpperBoundHolds(&large.X)
					case "both-check", "free-check":
						_, err = c.BoundsHold(&large.X)
					}
					h.budget.FailAfter(-1)
					if err == nil || sourceDebugEnabled && err == ErrAssertion && strings.HasSuffix(phase, "check") {
						done = true
					} else {
						failures++
						if !lpExactPriorityPhysicalRefusal(err) {
							t.Fatal(err)
						}
						if phase == "low" && c.LowSet || phase == "upper" && c.UpperSet {
							t.Fatal("const-reference setter flag preceded failed copy")
						}
						if phase == "fixed" {
							if c.Fixed {
								sawFixedAfter = true
							} else {
								sawFixedBefore = true
							}
						}
						if c.SetCost(&out.X) != ErrExactVectorPending {
							t.Fatal("stopped column retry")
						}
					}
					h.must(c.Close())
					lpExactVectorNamedCensus(t, h, nil)
				}) {
					t.FailNow()
				}
			}
			if !done || direct && failures == 0 {
				t.Fatal("incomplete physical column campaign", direct, phase, done, failures)
			}
			if direct && phase == "fixed" && (!sawFixedBefore || !sawFixedAfter) {
				t.Fatal("missing by-value parameter/flag publication phases", sawFixedBefore, sawFixedAfter)
			}
		}
	}
}
func TestLPRationalColumnPreservesOlderComparison(t *testing.T) {
	if !testing.Short() {
		t.Skip("lp: test too slow, run with -short for full suite.")
	}

	matched := false
	for ordinal := int64(0); ordinal < 40 && !matched; ordinal++ {
		h := newLPPairHarness(t)
		c := NewRationalColumnInfo(h.m, 0)
		large := h.pair(strings.Repeat("1234567890", 20)+"7/13", "0")
		h.must(c.SetFixedValue(&large.X))
		out := h.pair("91", "0")
		v := h.pair(strings.Repeat("9876543210", 100)+"/7", strings.Repeat("1234567890", 100)+"/13")
		var caller exactEtaCalls
		b, err := caller.begin(h.m)
		h.must(err)
		h.budget.FailAfter(ordinal)
		_, err = h.m.Compare(&v.X, &v.Y)
		h.budget.FailAfter(-1)
		err = caller.after(h.m, b, err)
		older := h.m.ComparisonInvocationState()
		if older.Active {
			matched = true
			if !lpExactPriorityPhysicalRefusal(err) {
				t.Fatal(err)
			}
			_ = c.Shift(&out.X)
			if c.calls.comparison != 0 {
				t.Fatal("column adopted older comparison")
			}
		}
		h.must(c.Close())
		if older.Active && h.m.ComparisonInvocationState() != older {
			t.Fatal("column consumed older comparison")
		}
		h.must(caller.cleanupRaw(h.m))
		h.must(caller.cleanupRoots(h.m))
		lpExactVectorNamedCensus(t, h, nil)
	}
	if !matched {
		t.Fatal("no actual older comparison")
	}
}
