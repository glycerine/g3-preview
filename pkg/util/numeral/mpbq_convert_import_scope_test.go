package numeral

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestBinaryConvertImportedOwningCleanupPhysicalRetry(t *testing.T) {
	qm, _, _ := rawArithmeticGuardOwner(t)
	diagnostics, e := NewRationalService(serviceConfig(qm.Z.BudgetField))
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, diagnostics.Close()) })
	m, e := NewBinaryRationalManager(qm.Z, BinaryRationalConfig{Diagnostics: diagnostics})
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.CloseChecked()) })
	serviceMust(t, qm.Z.SetLimbs(&m.convertImported.Num, []uint32{7, 0, 1}))
	serviceMust(t, qm.Z.SetLimbs(&m.convertImported.Den, []uint32{9, 0, 1}))
	m.convertImportedPending = true
	den := m.convertImported.Den.Cell
	original := qm.Z.Cell(den)
	bad := original
	bad.Backing.Requested = uint64(math.MaxUint32) + 1
	qm.Z.Store(den, bad)
	if e := m.CleanupConvertImported(); !errors.Is(e, arena.ErrBounds) || !m.convertImportedPending || m.convertImportedCursor != 1 || m.convertImported.Num.Cell != 0 || m.convertImported.Den.Cell != den {
		t.Fatal("physical refusal lost imported denominator", e, m.convertImported)
	}
	if e := m.open(); !errors.Is(e, ErrBinaryConvertPending) {
		t.Fatal("retained conversion copy allowed reentry", e)
	}
	qm.Z.Store(den, original)
	serviceMust(t, m.CleanupConvertImported())
	if m.convertImportedPending || m.convertImported.Num.Cell != 0 || m.convertImported.Den.Cell != 0 {
		t.Fatal("owning retry retained imported rational")
	}
}
