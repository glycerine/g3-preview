package numeral

import (
	"errors"
	"testing"
)

func TestFastFloatFractionTemporariesRetainSourceCleanupOrder(t *testing.T) {
	m := fastManager(t, 2)
	before := m.live
	serviceMust(t, m.SetInt32(&m.fractionNum, 7))
	serviceMust(t, m.SetInt32(&m.fractionDen, 3))
	m.fractionPending = true
	num, den := m.fractionNum.bits, m.fractionDen.bits
	m.fractionDen.bits = ffPack(m.capacity, 0, false)
	if e := m.CleanupFractionTemporaries(); !errors.Is(e, ErrFastFloatOwner) || !m.fractionPending || m.fractionCursor != 0 || m.fractionNum.bits != num {
		t.Fatal("denominator refusal crossed numerator", e)
	}
	if e := m.open(); !errors.Is(e, ErrFastFloatFractionPending) {
		t.Fatal("retained fraction setter allowed reentry", e)
	}
	m.fractionDen.bits = den
	m.fractionNum.bits = ffPack(m.capacity, 0, false)
	if e := m.CleanupFractionTemporaries(); !errors.Is(e, ErrFastFloatOwner) || !m.fractionPending || m.fractionCursor != 1 || m.fractionDen.bits != 0 {
		t.Fatal("retry replayed denominator or lost numerator", e)
	}
	m.fractionNum.bits = num
	serviceMust(t, m.CleanupFractionTemporaries())
	if m.fractionPending || m.fractionNum.bits != 0 || m.fractionDen.bits != 0 || m.live != before {
		t.Fatal("owning retry retained fraction temporaries")
	}
	serviceMust(t, m.open())
}
