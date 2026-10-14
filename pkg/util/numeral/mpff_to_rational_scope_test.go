package numeral

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestFastFloatToRationalPowerBorrowedCleanupOrder(t *testing.T) {
	qm, _, _ := rawArithmeticGuardOwner(t)
	m := fastManager(t, 2)
	m.rationalPower = NewRational()
	serviceMust(t, qm.Z.SetLimbs(&m.rationalPower.Num, []uint32{7, 0, 1}))
	serviceMust(t, qm.Z.SetLimbs(&m.rationalPower.Den, []uint32{11, 0, 1}))
	m.rationalPowerOwner = qm
	m.rationalPowerPending = true
	num, den := m.rationalPower.Num.Cell, m.rationalPower.Den.Cell
	originalNum, originalDen := qm.Z.Cell(num), qm.Z.Cell(den)
	bad := originalNum
	bad.Backing.Requested = uint64(math.MaxUint32) + 1
	qm.Z.Store(num, bad)
	if e := m.CleanupToRationalPower(); !errors.Is(e, arena.ErrBounds) || !m.rationalPowerPending || m.rationalPowerCursor != 0 || m.rationalPower.Num.Cell != num || m.rationalPower.Den.Cell != den {
		t.Fatal("numerator refusal crossed denominator", e, m.rationalPower)
	}
	if e := m.open(); !errors.Is(e, ErrFastFloatToRationalPending) {
		t.Fatal("retained to-rational power allowed reentry", e)
	}
	qm.Z.Store(num, originalNum)
	bad = originalDen
	bad.Backing.Requested = uint64(math.MaxUint32) + 1
	qm.Z.Store(den, bad)
	if e := m.CleanupToRationalPower(); !errors.Is(e, arena.ErrBounds) || !m.rationalPowerPending || m.rationalPowerCursor != 1 || m.rationalPower.Num.Cell != 0 || m.rationalPower.Den.Cell != den || ValidNumericRef(qm.Z.Cells, num) {
		t.Fatal("denominator refusal replayed numerator or lost denominator", e, m.rationalPower)
	}
	qm.Z.Store(den, originalDen)
	serviceMust(t, m.CleanupToRationalPower())
	if m.rationalPowerPending || m.rationalPowerOwner != nil || m.rationalPowerCursor != 0 || m.rationalPower.Num.Cell != 0 || m.rationalPower.Den.Cell != 0 {
		t.Fatal("owning retry retained borrowed rational power")
	}
	serviceMust(t, m.open())
}
