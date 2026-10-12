package lp

import (
	"fmt"
	"math/big"
	"strings"
	"testing"
)

// set_fixed_value takes mpq by value, then publishes the fixed flag before
// assigning the member. Its parameter may borrow another member of this owner.
func TestLPRationalColumnAliasedFixedParameterPreservesSource(t *testing.T) {
	for _, direct := range []bool{false, true} {
		done, failures, beforeFlag, afterFlag := false, 0, false, false
		for ordinal := int64(0); ordinal < 100 && !done; ordinal++ {
			if !t.Run(fmt.Sprintf("direct=%v/failure=%d", direct, ordinal), func(t *testing.T) {
				var h *lpPairHarness
				if direct {
					h = lpNewDirectSparseHarness(t)
				} else {
					h = newLPPairHarness(t)
				}
				text := strings.Repeat("1234567890", 20) + "7/13"
				p := h.pair(text, "-17/19")
				c := NewRationalColumnInfo(h.m, 71)
				c.Name, c.LowStrict, c.UpperStrict = "borrowed", true, true
				h.must(c.SetLowBound(&p.X))
				h.must(c.SetCost(&p.Y))
				t.Cleanup(func() { h.budget.FailAfter(-1); h.must(c.Close()) })
				h.budget.FailAfter(ordinal)
				err := c.SetFixedValue(&c.Low)
				h.budget.FailAfter(-1)
				want, _ := new(big.Rat).SetString(text)
				lpSparseRationalCheck(t, h, &c.Low, want)
				lpSparseRationalCheck(t, h, &c.Cost, big.NewRat(-17, 19))
				if !c.LowSet || c.UpperSet || !c.LowStrict || !c.UpperStrict || c.Name != "borrowed" || c.Index != 71 {
					t.Fatal("unrelated metadata changed")
				}
				if err != nil {
					if !lpExactPriorityPhysicalRefusal(err) {
						t.Fatal(err)
					}
					failures++
					if c.Fixed {
						afterFlag = true
					} else {
						beforeFlag = true
					}
					if c.SetCost(&p.Y) != ErrExactVectorPending {
						t.Fatal("failed owner accepted another operation")
					}
				} else {
					done = true
					if !c.Fixed || c.ColumnType() != ColumnFixed || c.work != nil {
						t.Fatal("fixed publication or lexical parameter cleanup")
					}
					h.must(h.m.SetInt64(&c.Low, 23))
					lpSparseRationalCheck(t, h, &c.FixedValue, want)
					// A returned by-value getter survives destruction of the column.
					h.must(c.GetFixedValue(&p.Y))
				}
				h.must(c.Close())
				if done {
					lpSparseRationalCheck(t, h, &p.Y, want)
				}
				lpExactVectorNamedCensus(t, h, nil)
			}) {
				t.FailNow()
			}
		}
		// Warm pooled slots can satisfy both copies without requesting backing
		// storage. Direct allocation must exercise both physical refusal stages.
		if !done || direct && (failures == 0 || !beforeFlag || !afterFlag) {
			t.Fatal("missing aliased parameter publication prefix", direct, done, failures, beforeFlag, afterFlag)
		}
	}
}

func TestLPRationalColumnAdjustedExpressionPhysicalPrefixes(t *testing.T) {
	for _, direct := range []bool{false, true} {
		done, sawCopiedLeft := false, false
		for ordinal := int64(0); ordinal < 150 && !done; ordinal++ {
			if !t.Run(fmt.Sprintf("direct=%v/failure=%d", direct, ordinal), func(t *testing.T) {
				var h *lpPairHarness
				if direct {
					h = lpNewDirectSparseHarness(t)
				} else {
					h = newLPPairHarness(t)
				}
				low := strings.Repeat("1234567890", 20) + "7/13"
				upper := strings.Repeat("9876543210", 20) + "1/17"
				p := h.pair(low, upper)
				out := h.pair("91", "0")
				c := NewRationalColumnInfo(h.m, 8)
				h.must(c.SetLowBound(&p.X))
				h.must(c.SetUpperBound(&p.Y))
				t.Cleanup(func() { h.budget.FailAfter(-1); h.must(c.Close()) })
				h.budget.FailAfter(ordinal)
				err := c.AdjustedUpperBound(&out.X)
				h.budget.FailAfter(-1)
				lo, _ := new(big.Rat).SetString(low)
				hi, _ := new(big.Rat).SetString(upper)
				lpSparseRationalCheck(t, h, &c.Low, lo)
				lpSparseRationalCheck(t, h, &c.Upper, hi)
				if err != nil {
					if !lpExactPriorityPhysicalRefusal(err) {
						t.Fatal(err)
					}
					// A completed rational(left) must remain owned on a later
					// subtraction refusal, before expression-result publication.
					if c.work != nil && len(c.work.Data) == 4 {
						// An interrupted numeral copy may be unreadable until
						// its captured raw invocation is cleaned. Only a fully
						// materialized temporary proves this publication prefix.
						tmp, tmpErr := h.m.String(&c.work.Data[3])
						result, resultErr := h.m.String(&c.work.Data[2])
						if tmpErr == nil && resultErr == nil && tmp == hi.RatString() && result == "0" {
							sawCopiedLeft = true
							lpSparseRationalCheck(t, h, &out.X, big.NewRat(91, 1))
						}
					}
				} else {
					done = true
					lpSparseRationalCheck(t, h, &out.X, new(big.Rat).Sub(hi, lo))
					if c.work != nil {
						t.Fatal("returned expression retained lexical owners")
					}
				}
				h.must(c.Close())
				lpExactVectorNamedCensus(t, h, nil)
			}) {
				t.FailNow()
			}
		}
		if !done || direct && !sawCopiedLeft {
			t.Fatal("missing copied-left subtraction refusal", direct, done, sawCopiedLeft)
		}
	}
}
