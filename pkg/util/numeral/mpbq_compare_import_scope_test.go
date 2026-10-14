package numeral

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestBinaryCompareImportedOwningCleanupPhysicalRetry(t *testing.T) {
	qm, _, _ := rawArithmeticGuardOwner(t)
	diagnostics, e := NewRationalService(serviceConfig(qm.Z.BudgetField))
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, diagnostics.Close()) })
	m, e := NewBinaryRationalManager(qm.Z, BinaryRationalConfig{Diagnostics: diagnostics})
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.CloseChecked()) })
	serviceMust(t, qm.Z.SetLimbs(&m.compareImported.Num, []uint32{7, 0, 1}))
	serviceMust(t, qm.Z.SetLimbs(&m.compareImported.Den, []uint32{9, 0, 1}))
	m.compareImportedPending = true
	den := m.compareImported.Den.Cell
	original := qm.Z.Cell(den)
	bad := original
	bad.Backing.Requested = uint64(math.MaxUint32) + 1
	qm.Z.Store(den, bad)
	if e := m.CleanupCompareImported(); !errors.Is(e, arena.ErrBounds) || !m.compareImportedPending || m.compareImportedCursor != 1 || m.compareImported.Num.Cell != 0 || m.compareImported.Den.Cell != den {
		t.Fatal("physical refusal lost imported denominator", e, m.compareImported)
	}
	if e := m.open(); !errors.Is(e, ErrBinaryComparePending) {
		t.Fatal("retained comparison copy allowed reentry", e)
	}
	qm.Z.Store(den, original)
	serviceMust(t, m.CleanupCompareImported())
	if m.compareImportedPending || m.compareImported.Num.Cell != 0 || m.compareImported.Den.Cell != 0 {
		t.Fatal("owning retry retained imported rational")
	}
}
