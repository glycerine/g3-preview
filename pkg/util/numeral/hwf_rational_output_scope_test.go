package numeral

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestHardwareFloatToRationalScopedIntegersRetainReverseCleanup(t *testing.T) {
	qm, _, _ := rawArithmeticGuardOwner(t)
	serviceMust(t, qm.Z.SetLimbs(&qm.hwfN, []uint32{7, 0, 1}))
	serviceMust(t, qm.Z.SetLimbs(&qm.hwfD, []uint32{11, 0, 1}))
	qm.hwfPending = true
	num, den := qm.hwfN.Cell, qm.hwfD.Cell
	originalNum, originalDen := qm.Z.Cell(num), qm.Z.Cell(den)
	defer func() {
		if ValidNumericRef(qm.Z.Cells, num) {
			qm.Z.Store(num, originalNum)
		}
		if ValidNumericRef(qm.Z.Cells, den) {
			qm.Z.Store(den, originalDen)
		}
	}()
	bad := originalDen
	bad.Backing.Requested = uint64(math.MaxUint32) + 1
	qm.Z.Store(den, bad)
	if e := qm.cleanupHWFConversion(true); !errors.Is(e, arena.ErrBounds) || !qm.hwfPending || qm.hwfCursor != 0 || qm.hwfN.Cell != num || qm.hwfD.Cell != den {
		t.Fatal("source denominator refusal crossed numerator", e)
	}
	if e := qm.checkOwnerReady(); !errors.Is(e, ErrRationalHWFConversionPending) {
		t.Fatal("stopped conversion allowed rational reentry", e)
	}
	if e := qm.SourceFinalize(); !errors.Is(e, ErrRationalLexicalPending) {
		t.Fatal("source finalization discarded stopped conversion", e)
	}
	qm.Z.Store(den, originalDen)
	bad = originalNum
	bad.Backing.Requested = uint64(math.MaxUint32) + 1
	qm.Z.Store(num, bad)
	if e := qm.CleanupHWFConversion(); !errors.Is(e, arena.ErrBounds) || !qm.hwfPending || qm.hwfCursor != 1 || qm.hwfD.Cell != 0 || qm.hwfN.Cell != num {
		t.Fatal("owning retry replayed denominator or lost numerator", e)
	}
	qm.Z.Store(num, originalNum)
	serviceMust(t, qm.CleanupHWFConversion())
	if qm.hwfPending || qm.hwfCursor != 0 || qm.hwfN.Cell != 0 || qm.hwfD.Cell != 0 {
		t.Fatal("owning retry retained conversion scope")
	}
	serviceMust(t, qm.checkOwnerReady())
}
