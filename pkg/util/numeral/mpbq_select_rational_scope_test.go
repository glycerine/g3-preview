package numeral

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestBinarySelectionScopedRationalsResumeSourceDestructionOrder(t *testing.T) {
	qm, _, _ := rawArithmeticGuardOwner(t)
	diagnostics, e := NewRationalService(serviceConfig(qm.Z.BudgetField))
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, diagnostics.Close()) })
	m, e := NewBinaryRationalManager(qm.Z, BinaryRationalConfig{Diagnostics: diagnostics})
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.CloseChecked()) })
	serviceMust(t, m.beginSelectRational(qm, selectRationalBoth))
	parts := [4]*Int{&m.selectRationalUpper.Num, &m.selectRationalUpper.Den, &m.selectRationalLower.Num, &m.selectRationalLower.Den}
	var refs [4]NumericCellRef
	var originals [4]NumericCell
	for i, part := range parts {
		serviceMust(t, qm.Z.SetLimbs(part, []uint32{uint32(i + 3), 0, 1}))
		refs[i] = part.Cell
		originals[i] = qm.Z.Cell(refs[i])
		bad := originals[i]
		bad.Backing.Requested = uint64(math.MaxUint32) + 1
		qm.Z.Store(refs[i], bad)
	}
	m.selectRationalExecuting = false
	for stage, part := range parts {
		e := m.finishSelectRationalSource()
		if !errors.Is(e, arena.ErrBounds) || m.selectRationalCursor != uint8(stage) || !m.selectRationalPending || part.Cell != refs[stage] {
			t.Fatalf("source destructor stage %d: %v cursor=%d", stage, e, m.selectRationalCursor)
		}
		if e := m.open(); !errors.Is(e, ErrBinarySelectPending) {
			t.Fatal("new binary operation accepted stopped scope", e)
		}
		for earlier := 0; earlier < stage; earlier++ {
			if parts[earlier].Cell != 0 {
				t.Fatalf("source destructor revisited completed component %d", earlier)
			}
		}
		qm.Z.Store(refs[stage], originals[stage])
	}
	serviceMust(t, m.CleanupSelectRationalTemporaries())
	if m.selectRationalPending || m.selectRationalOwner != nil || m.selectRationalCursor != 4 {
		t.Fatal("owning continuation did not drain both rationals")
	}
	for _, part := range parts {
		if part.Cell != 0 {
			t.Fatal("owning continuation retained a component")
		}
	}
}
