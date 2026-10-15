package numeral

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestRationalCopyAllocationReceiptSuccessfulCopyHasNoPendingOwner(t *testing.T) {
	s, _ := serviceNew(t)
	src := serviceValue(t, s, "12345678901234567890123456789/998877665544332211")
	dst, e := s.NewValue()
	serviceMust(t, e)
	var receipt RationalCopyAllocationReceipt
	serviceMust(t, s.CopyWithAllocationReceipt(&dst, &src, &receipt))
	if receipt.Active() || !errors.Is(s.CleanupCopyAllocation(&receipt), arena.ErrInvalidSlot) {
		t.Fatal("completed numerator and denominator retained allocation authority")
	}
	equal, e := s.Equal(&dst, &src)
	if e != nil || !equal {
		t.Fatal("completed tracked copy changed rational value", equal, e)
	}
	serviceMust(t, s.Del(&dst))
	serviceMust(t, s.Del(&src))
}
