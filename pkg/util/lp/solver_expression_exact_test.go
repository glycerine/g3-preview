package lp

import (
	"errors"
	"math/big"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
)

// A failed copy of the returned compound-result reference must retain the
// completed expression temporary, with an incomplete return destination.
// Values differ from both operands, so a failed compound operation cannot
// satisfy this test accidentally.
func TestLPRationalExpressionPhysicalReturnCopies(t *testing.T) {
	for _, phase := range []string{"shift", "minimum", "pin", "fixed", "rhs", "obsolete"} {
		t.Run(phase, func(t *testing.T) {
			seenReturnCopy, done := false, false
			for ordinal := int64(0); ordinal < 650 && !done; ordinal++ {
				h := lpNewDirectSparseHarness(t)
				m := lpRationalInputModel(t, h)
				aText, bText := strings.Repeat("1234567890", 12)+"7/13", strings.Repeat("9876543210", 10)+"1/17"
				a, b, one := h.pair(aText, "0"), h.pair(bText, "0"), h.pair("1", "0")
				out := h.pair("91", "0")
				av, _ := new(big.Rat).SetString(aText)
				bv, _ := new(big.Rat).SetString(bText)
				want := new(big.Rat).Mul(av, bv)
				h.must(m.SetRowColumnCoefficient(100, 10, &a.X))
				h.must(m.SetLowBound(10, &b.X))
				h.must(m.SetUpperBound(10, &b.X))
				h.must(m.AddConstraint(RelationLessOrEqual, &one.X, 100))
				if phase == "fixed" {
					h.must(m.SetFixedValue(10, &b.X))
				}
				if phase == "rhs" {
					h.must(m.FillMatrixAAndInitRightSide())
					want.Sub(big.NewRat(1, 1), want)
				} else if phase == "obsolete" {
					want.Sub(want, big.NewRat(1, 1))
				}
				row, _ := m.AValues.Get2(100)
				h.budget.FailAfter(ordinal)
				var err error
				var tmp, result *numeral.Rational
				switch phase {
				case "shift":
					err = m.LowBoundShiftForRow(100, &out.X)
					if m.shiftWork != nil {
						tmp, result = &m.shiftWork.Data[3], &m.shiftWork.Data[2]
					}
				case "minimum":
					_, err = m.MinimalRowValue(row, &out.X)
					if m.shiftWork != nil {
						tmp, result = &m.shiftWork.Data[3], &m.shiftWork.Data[2]
					}
				case "pin":
					err = m.PinVarsOnRowWithSign(row, &b.X)
					if m.pinWork != nil {
						tmp, result = &m.pinWork.Data[5], &m.pinWork.Data[3]
					}
				case "fixed":
					err = m.RemoveFixedOrZeroColumnsFromRow(100, row)
					if m.work != nil {
						tmp, result = &m.work.Data[3], &m.work.Data[2]
					}
				case "rhs":
					err = m.FillB()
					if m.work != nil {
						tmp, result = &m.work.Data[2], &m.work.Data[1]
					}
				case "obsolete":
					_, err = m.RowIsObsolete(row, 100)
					if m.work != nil {
						tmp, result = &m.work.Data[5], &m.work.Data[3]
					}
				}
				h.budget.FailAfter(-1)
				if err == nil {
					done = true
				} else {
					if !lpExactPriorityPhysicalRefusal(err) {
						t.Fatal(ordinal, err)
					}
					// A completed compound expression has no unfinished
					// normalization, GCD or division invocation. Exclude
					// those earlier failures from the return-copy witness.
					gcd, div := h.m.Z.GCDInvocation(), h.m.Z.DivisionInvocation()
					if tmp != nil && h.m.RawInvocationState().Active == [3]bool{} && !gcd.Matches(h.m.Z) && !div.Matches(h.m.Z) {
						text, e := h.m.String(tmp)
						if e == nil {
							got, ok := new(big.Rat).SetString(text)
							if ok && got.Cmp(want) == 0 {
								returned, readErr := h.m.String(result)
								if readErr != nil && !errors.Is(readErr, arena.ErrInvalidSlot) {
									t.Fatal(readErr)
								}
								value, valid := new(big.Rat).SetString(returned)
								if readErr != nil || !valid || value.Cmp(want) != 0 {
									seenReturnCopy = true
								}
							}
						} else if !errors.Is(e, arena.ErrInvalidSlot) {
							t.Fatal(e)
						}
						// Earlier interrupted compound arithmetic may leave an
						// invalid temporary until its actual receipt is cleaned.
					}
				}
				h.must(m.Close())
				lpExactVectorNamedCensus(t, h, nil)
			}
			if !done || !seenReturnCopy {
				t.Fatal("missing successful completion or actual returned-reference copy refusal", done, seenReturnCopy)
			}
		})
	}
}

func TestLPRationalExpressionPinConditionLifetime(t *testing.T) {
	h := newLPPairHarness(t)
	m := lpRationalInputModel(t, h)
	a := h.pair(strings.Repeat("1234567890", 12)+"7/13", "0")
	sign := h.pair(strings.Repeat("9876543210", 10)+"1/17", "0")
	h.must(m.SetRowColumnCoefficient(100, 10, &a.X))
	c, e := m.GetOrCreateColumnInfo(10)
	h.must(e)
	h.must(c.Close())
	row, _ := m.AValues.Get2(100)
	if e := m.PinVarsOnRowWithSign(row, &sign.X); e != ErrExactVectorPending {
		t.Fatal("expected reached closed child getter", e)
	}
	// The multiplication finished before the branch, and both expression
	// owners died before the child getter. The loop's coefficient stays live.
	lpSparseRationalCheck(t, h, &m.pinWork.Data[3], new(big.Rat))
	lpSparseRationalCheck(t, h, &m.pinWork.Data[5], new(big.Rat))
	av, _ := new(big.Rat).SetString(strings.Repeat("1234567890", 12) + "7/13")
	lpSparseRationalCheck(t, h, &m.pinWork.Data[2], av)
	h.must(m.Close())
	lpExactVectorNamedCensus(t, h, nil)
}
