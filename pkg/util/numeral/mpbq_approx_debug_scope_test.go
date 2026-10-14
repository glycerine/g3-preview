//go:build g3debug

package numeral

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestBinaryApproxDebugSnapshotOwningCleanupRetry(t *testing.T) {
	qm, _, _ := rawArithmeticGuardOwner(t)
	diagnostics, e := NewRationalService(serviceConfig(qm.Z.BudgetField))
	serviceMust(t, e)
	m, e := NewBinaryRationalManager(qm.Z, BinaryRationalConfig{Diagnostics: diagnostics})
	serviceMust(t, e)
	t.Cleanup(func() {
		serviceMust(t, m.CloseChecked())
		serviceMust(t, diagnostics.Close())
	})
	serviceMust(t, qm.Z.SetLimbs(&m.approxOld.Num, []uint32{7, 0, 1}))
	m.approxPending = true
	ref := m.approxOld.Num.Cell
	original := qm.Z.Cell(ref)
	bad := original
	bad.Backing.Requested = uint64(math.MaxUint32) + 1
	qm.Z.Store(ref, bad)
	if e := m.CleanupApproxDebug(); !errors.Is(e, arena.ErrBounds) || !m.approxPending || m.approxOld.Num.Cell != ref {
		t.Fatal("physical refusal lost approximation snapshot", e, m.approxOld)
	}
	if e := m.open(); !errors.Is(e, ErrBinaryApproxPending) {
		t.Fatal("retained snapshot allowed reentry", e)
	}
	qm.Z.Store(ref, original)
	serviceMust(t, m.CleanupApproxDebug())
	if m.approxPending || m.approxOld.Num.Cell != 0 {
		t.Fatal("owning retry retained approximation snapshot")
	}
}
