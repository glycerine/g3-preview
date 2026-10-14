package numeral

import (
	"errors"
	"testing"
)

func TestFastFloatSetRationalTemporariesRetainSourceCleanupOrder(t *testing.T) {
	m := fastManager(t, 2)
	before := m.live
	serviceMust(t, m.SetInt32(&m.rationalNum, 7))
	serviceMust(t, m.SetInt32(&m.rationalDen, 3))
	m.rationalPending = true
	num, den := m.rationalNum.bits, m.rationalDen.bits
	m.rationalDen.bits = ffPack(m.capacity, 0, false)
	if e := m.CleanupSetRationalTemporaries(); !errors.Is(e, ErrFastFloatOwner) || !m.rationalPending || m.rationalCursor != 0 || m.rationalNum.bits != num {
		t.Fatal("denominator refusal crossed numerator", e)
	}
	if e := m.open(); !errors.Is(e, ErrFastFloatRationalPending) {
		t.Fatal("retained rational setter allowed reentry", e)
	}
	m.rationalDen.bits = den
	m.rationalNum.bits = ffPack(m.capacity, 0, false)
	if e := m.CleanupSetRationalTemporaries(); !errors.Is(e, ErrFastFloatOwner) || !m.rationalPending || m.rationalCursor != 1 || m.rationalDen.bits != 0 {
		t.Fatal("retry replayed denominator or lost numerator", e)
	}
	m.rationalNum.bits = num
	serviceMust(t, m.CleanupSetRationalTemporaries())
	if m.rationalPending || m.rationalNum.bits != 0 || m.rationalDen.bits != 0 || m.live != before {
		t.Fatal("owning retry retained rational setter temporaries")
	}
	serviceMust(t, m.open())
}
