package numeral

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestFastFloatToIntegerPowerBorrowedCleanupPhysicalRetry(t *testing.T) {
	qm, _, _ := rawArithmeticGuardOwner(t)
	m := fastManager(t, 2)
	serviceMust(t, qm.Z.SetLimbs(&m.integerPower, []uint32{7, 0, 1}))
	m.integerPowerOwner = qm.Z
	m.integerPowerPending = true
	ref := m.integerPower.Cell
	original := qm.Z.Cell(ref)
	bad := original
	bad.Backing.Requested = uint64(math.MaxUint32) + 1
	qm.Z.Store(ref, bad)
	if e := m.cleanupToIntegerPower(true); !errors.Is(e, arena.ErrBounds) || !m.integerPowerPending || m.integerPowerOwner != qm.Z || m.integerPower.Cell != ref {
		t.Fatal("source physical refusal lost borrowed power", e, m.integerPower)
	}
	if e := m.open(); !errors.Is(e, ErrFastFloatIntegerPending) {
		t.Fatal("retained to-integer power allowed reentry", e)
	}
	if e := m.CleanupToIntegerPower(); !errors.Is(e, arena.ErrBounds) || !m.integerPowerPending || m.integerPower.Cell != ref {
		t.Fatal("owning refusal lost source-stopped power", e, m.integerPower)
	}
	qm.Z.Store(ref, original)
	serviceMust(t, m.CleanupToIntegerPower())
	if m.integerPowerPending || m.integerPowerOwner != nil || m.integerPower.Cell != 0 {
		t.Fatal("owning retry retained borrowed power")
	}
	serviceMust(t, m.open())
}
