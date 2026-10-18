package polynomial

import (
	"errors"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"strconv"
	"testing"
)

func TestAlgebraicCleanupPreservesOlderIntegerCopyReceipt(t *testing.T) {
	found := false
	for ordinal := int64(0); ordinal < 128 && !found; ordinal++ {
		t.Run(strconv.FormatInt(ordinal, 10), func(t *testing.T) {
			c, _, _ := algebraicCompareFixture(t)
			z := c.config.Rationals.Integers()
			var input Algebraic
			source := unumeral.Rational{Num: z.Two64, Den: unumeral.NewInt(1)}
			serviceMust(t, c.SetRational(&input, &source))
			// Keep real copied numerals alive to cross descriptor-page growth without
			// warming and recycling those pages through a large parsing prelude.
			var destinations [32]unumeral.Int
			var older unumeral.IntegerCopyAllocationReceipt
			z.Budget().FailAfter(ordinal)
			var body error
			for i := range destinations {
				body = z.CopyFromOwnerWithReceipt(&destinations[i], z, &z.Two64, &older)
				if body != nil {
					break
				}
			}
			z.Budget().FailAfter(-1)
			if body != nil && !numericPhysicalReservationRefusal(body) {
				t.Fatal("unexpected source copy refusal", body)
			}
			if z.MatchesCopyAllocationInvocation(&older) {
				found = true
				state, used := z.CopyAllocation, z.Budget().Used()
				out := unumeral.NewRational()
				if e := c.ToRational(&input, &out); !errors.Is(e, unumeral.ErrNumericPending) || out.Num.NativeScalar() != 1 || out.Num.Cell != z.FailedNullRef {
					t.Fatal("refused conversion lost source copy prefix", e, out)
				}
				if z.MatchesCopyAllocationInvocation(&c.toRationalCopy) {
					t.Fatal("refused conversion acquired independent copy")
				}
				serviceMust(t, c.CleanupToRational())
				serviceMust(t, c.config.Rationals.Del(&out))
				borrowed := unumeral.Rational{Num: z.Two64, Den: unumeral.NewInt(1)}
				var untouched Algebraic
				if e := c.SetRational(&untouched, &borrowed); !errors.Is(e, unumeral.ErrNumericPending) || untouched != (Algebraic{}) {
					t.Fatal("refused rational copy", e)
				}
				if z.MatchesCopyAllocationInvocation(&c.rationalCopy) {
					t.Fatal("refused setter acquired older copy receipt")
				}

				serviceMust(t, c.CleanupRetained())
				if !z.MatchesCopyAllocationInvocation(&older) || z.CopyAllocation != state || z.Budget().Used() != used {
					t.Fatal("unrelated algebraic cleanup acquired older copy authority")
				}
			}
			serviceMust(t, z.CleanupCopyAllocationInvocation(&older))
			serviceMust(t, c.Del(&input))
			for i := range destinations {
				serviceMust(t, z.Del(&destinations[i]))
			}
		})
		if t.Failed() {
			return
		}
	}
	if !found {
		t.Fatal("no actual retained older integer-copy allocation reached")
	}
}
