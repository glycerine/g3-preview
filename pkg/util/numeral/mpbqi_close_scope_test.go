package numeral

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestBinaryIntervalCloseRetainsStoppedScratchOrder(t *testing.T) {
	qm, _, _ := rawArithmeticGuardOwner(t)
	diagnostics, e := NewRationalService(serviceConfig(qm.Z.BudgetField))
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, diagnostics.Close()) })
	bounds, e := NewBinaryRationalManager(qm.Z, BinaryRationalConfig{Diagnostics: diagnostics})
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, bounds.CloseChecked()) })
	m, e := NewBinaryIntervalManager(bounds)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.CloseChecked()) })
	for i, r := range []*BinaryRational{&m.MulCurr, &m.MulMax, &m.MulMin} {
		serviceMust(t, qm.Z.SetLimbs(&r.Num, []uint32{uint32(i + 3), 0, 1}))
	}
	current, maximum, minimum := m.MulCurr.Num.Cell, m.MulMax.Num.Cell, m.MulMin.Num.Cell
	original := qm.Z.Cell(maximum)
	bad := original
	bad.Backing.Requested = uint64(math.MaxUint32) + 1
	qm.Z.Store(maximum, bad)
	if e := m.CloseChecked(); !errors.Is(e, arena.ErrBounds) || m.cleanupStage != 1 || !m.closing || m.closed {
		t.Fatal("source destructor did not stop at maximum", e, m.cleanupStage)
	}
	if m.MulCurr.Num.Cell != 0 || m.MulMax.Num.Cell != maximum || m.MulMin.Num.Cell != minimum || qm.Z.Cells.Stats().Live == 0 {
		t.Fatal("stopped destructor lost its exact suffix", current, maximum, minimum)
	}
	if e := m.open(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("closing interval accepted new operation", e)
	}
	if e := m.CloseChecked(); !errors.Is(e, arena.ErrBounds) || m.cleanupStage != 1 {
		t.Fatal("repeated refusal advanced destructor", e, m.cleanupStage)
	}
	qm.Z.Store(maximum, original)
	serviceMust(t, m.CloseChecked())
	if !m.closed || m.cleanupStage != 3 || m.HeaderBytes != 0 || m.MulMax.Num.Cell != 0 || m.MulMin.Num.Cell != 0 {
		t.Fatal("owning retry did not drain source destructor")
	}
}
