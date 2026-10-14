package numeral

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestIntegerLCMRawLocalSourceEndAndOwningRetry(t *testing.T) {
	qm, _, _ := rawArithmeticGuardOwner(t)
	z := qm.Z
	serviceMust(t, z.SetLimbs(&z.LCMTemporary, []uint32{7, 0, 1}))
	z.LCMPending = true
	ref := z.LCMTemporary.Cell
	original := z.Cell(ref)
	defer func() {
		if ValidNumericRef(z.Cells, ref) {
			z.Store(ref, original)
		}
	}()
	bad := original
	bad.Backing.Requested = uint64(math.MaxUint32) + 1
	z.Store(ref, bad)
	if e := z.finishLCMSource(); !errors.Is(e, arena.ErrBounds) || !z.LCMPending || z.LCMTemporary.Cell != ref {
		t.Fatal("source end-label refusal lost LCM local", e)
	}
	if e := z.SourceFinalize(); !errors.Is(e, ErrIntegerLCMPending) {
		t.Fatal("source finalize discarded LCM local", e)
	}
	a, b, out := NewInt(6), NewInt(10), NewInt(0)
	if e := z.LCM(&a, &b, &out); !errors.Is(e, ErrIntegerLCMPending) || out != NewInt(0) {
		t.Fatal("pending LCM allowed reentry", e, out)
	}
	if e := z.CleanupLCMTemporary(); !errors.Is(e, arena.ErrBounds) || !z.LCMPending || z.LCMTemporary.Cell != ref {
		t.Fatal("owning refusal lost raw LCM local", e)
	}
	z.Store(ref, original)
	serviceMust(t, z.CleanupLCMTemporary())
	if z.LCMPending || z.LCMTemporary.Cell != 0 {
		t.Fatal("owning retry retained LCM local")
	}
	serviceMust(t, z.LCM(&a, &b, &out))
	if out.Val != 30 || out.Cell != 0 {
		t.Fatal("LCM did not resume after cleanup", out)
	}
}
