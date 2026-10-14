package numeral

import (
	"errors"
	"testing"
)

func TestSynchronizedFusedRetainsBothComponentsAfterBodyStops(t *testing.T) {
	found := false
	for fault := int64(0); fault < 128 && !found; fault++ {
		m, budget := rationalManager(t, true)
		a, c, out := NewRational(), NewRational(), NewRational()
		var factor Int
		serviceMust(t, m.SetString(&a, "3/2"))
		serviceMust(t, m.Z.SetString(&factor, "18446744073709551617"))
		serviceMust(t, m.SetString(&c, "18446744073709551619/18446744073709551629"))
		serviceMust(t, m.SetString(&out, "1/18446744073709551621"))
		budget.FailAfter(fault)
		e := m.SubMulInt(&a, &factor, &c, &out)
		if e != nil && m.fusedPending && m.fusedTemporary.Num.Cell != 0 && m.fusedTemporary.Den.Cell != 0 {
			if !numericPhysicalReservationRefusal(e) || m.fusedCursor != 0 {
				t.Fatalf("unexpected stopped fused state: %v, cursor=%d", e, m.fusedCursor)
			}
			found = true
		}
		budget.FailAfter(-1)
		if found {
			if e := m.SourceFinalize(); !errors.Is(e, ErrRationalLexicalPending) {
				t.Fatalf("source finalization accepted fused local: %v", e)
			}
			if e := m.SubMulInt(&a, &factor, &c, &out); !errors.Is(e, ErrRationalLexicalPending) {
				t.Fatalf("second invocation accepted fused local: %v", e)
			}
		}
		serviceMust(t, m.CleanupFusedTemporary())
		serviceMust(t, m.CleanupRawArithmetic())
		if found && (m.fusedPending || m.fusedTemporary.Num.Cell != 0 || m.fusedTemporary.Den.Cell != 0 || m.fusedCursor != 2) {
			t.Fatal("owning cleanup did not finish fused local")
		}
		for _, q := range []*Rational{&out, &c, &a} {
			serviceMust(t, m.Del(q))
		}
		serviceMust(t, m.Z.Del(&factor))
		m.Close()
	}
	if !found {
		t.Fatal("reservation sweep never stopped after both fused components were allocated")
	}
}
