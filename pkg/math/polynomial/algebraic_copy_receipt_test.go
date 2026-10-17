package polynomial

import (
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
				serviceMust(t, c.CleanupRetained())
				if !z.MatchesCopyAllocationInvocation(&older) || z.CopyAllocation != state || z.Budget().Used() != used {
					t.Fatal("unrelated algebraic cleanup acquired older copy authority")
				}
			}
			serviceMust(t, z.CleanupCopyAllocationInvocation(&older))
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
