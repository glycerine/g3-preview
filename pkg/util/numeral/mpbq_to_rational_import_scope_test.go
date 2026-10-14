package numeral

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestBinaryToRationalImportedOwningCleanupPhysicalRetry(t *testing.T) {
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
	serviceMust(t, qm.Z.SetLimbs(&m.toRationalImported, []uint32{7, 0, 1}))
	m.toRationalImportedOwner = qm.Z
	m.toRationalImportedPending = true
	ref := m.toRationalImported.Cell
	original := qm.Z.Cell(ref)
	bad := original
	bad.Backing.Requested = uint64(math.MaxUint32) + 1
	qm.Z.Store(ref, bad)
	if e := m.CleanupToRationalImported(); !errors.Is(e, arena.ErrBounds) || !m.toRationalImportedPending || m.toRationalImportedOwner != qm.Z || m.toRationalImported.Cell != ref {
		t.Fatal("physical refusal lost borrowed integer", e, m.toRationalImported)
	}
	if e := m.open(); !errors.Is(e, ErrBinaryToRationalPending) {
		t.Fatal("retained to-rational copy allowed reentry", e)
	}
	qm.Z.Store(ref, original)
	serviceMust(t, m.CleanupToRationalImported())
	if m.toRationalImportedPending || m.toRationalImportedOwner != nil || m.toRationalImported.Cell != 0 {
		t.Fatal("owning retry retained borrowed integer")
	}
}
