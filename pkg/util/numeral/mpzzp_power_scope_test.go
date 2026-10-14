package numeral

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestIntegerModularPowerRawLocalSourceEndAndOwningRetry(t *testing.T) {
	m, z, _ := modularManager(t, 0)
	serviceMust(t, z.SetLimbs(&m.PowerTemporary, []uint32{7, 0, 1}))
	m.PowerPending = true
	ref := m.PowerTemporary.Cell
	original := z.Cell(ref)
	defer func() {
		if ValidNumericRef(z.Cells, ref) {
			z.Store(ref, original)
		}
	}()
	bad := original
	bad.Backing.Requested = uint64(math.MaxUint32) + 1
	z.Store(ref, bad)
	if e := m.finishPowerSource(); !errors.Is(e, arena.ErrBounds) || !m.PowerPending || m.PowerTemporary.Cell != ref {
		t.Fatal("source end-label refusal lost modular power local", e)
	}
	a, out := NewInt(6), NewInt(0)
	if e := m.Power(&a, 3, &out); !errors.Is(e, ErrModularPowerPending) || out != NewInt(0) {
		t.Fatal("pending power allowed reentry", e, out)
	}
	if e := m.Close(); !errors.Is(e, arena.ErrBounds) || m.closed {
		t.Fatal("close discarded stopped power local", e)
	}
	if e := m.CleanupPowerTemporary(); !errors.Is(e, arena.ErrBounds) || !m.PowerPending || m.PowerTemporary.Cell != ref {
		t.Fatal("owning refusal lost modular power local", e)
	}
	z.Store(ref, original)
	serviceMust(t, m.CleanupPowerTemporary())
	if m.PowerPending || m.PowerTemporary.Cell != 0 {
		t.Fatal("owning retry retained modular power local")
	}
	serviceMust(t, m.Power(&a, 3, &out))
	if out != NewInt(216) || m.PowerPending || m.PowerTemporary.Cell != 0 {
		t.Fatal("modular power did not resume and finish", out)
	}
}
