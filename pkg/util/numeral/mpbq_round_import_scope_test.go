package numeral

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestBinaryRoundImportedOwningCleanupPhysicalRetry(t *testing.T) {
	qm, budget, _ := rawArithmeticGuardOwner(t)
	diagnostics, e := NewRationalService(serviceConfig(budget))
	serviceMust(t, e)
	z, e := NewIntegerManager(serviceConfig(budget).Integers)
	serviceMust(t, e)
	m, e := NewBinaryRationalManager(z, BinaryRationalConfig{Diagnostics: diagnostics})
	serviceMust(t, e)
	t.Cleanup(func() {
		serviceMust(t, m.CloseChecked())
		serviceMust(t, z.CloseChecked())
		serviceMust(t, diagnostics.Close())
	})
	serviceMust(t, qm.Z.SetLimbs(&m.roundImported, []uint32{7, 0, 1}))
	m.roundImportedOwner = qm.Z
	m.roundImportedPending = true
	ref := m.roundImported.Cell
	original := qm.Z.Cell(ref)
	bad := original
	bad.Backing.Requested = uint64(math.MaxUint32) + 1
	qm.Z.Store(ref, bad)
	if e := m.CleanupRoundImported(); !errors.Is(e, arena.ErrBounds) || !m.roundImportedPending || m.roundImportedOwner != qm.Z || m.roundImported.Cell != ref {
		t.Fatal("physical refusal lost borrowed integer", e, m.roundImported)
	}
	if e := m.open(); !errors.Is(e, ErrBinaryRoundPending) {
		t.Fatal("retained rounding copy allowed reentry", e)
	}
	qm.Z.Store(ref, original)
	serviceMust(t, m.CleanupRoundImported())
	if m.roundImportedPending || m.roundImportedOwner != nil || m.roundImported.Cell != 0 {
		t.Fatal("owning retry retained borrowed integer")
	}
}
