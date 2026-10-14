//go:build g3debug

package numeral

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestBinaryDebugIntegerCopyOwningCleanupRetry(t *testing.T) {
	qm, budget, runtime := rawArithmeticGuardOwner(t)
	config := serviceConfig(budget)
	config.Integers.Memory = runtime
	config.Integers.Profile = umemory.SmallAllocatorProfile{PointerBits: 32, Debug: true}
	diagnostics, e := NewRationalService(config)
	serviceMust(t, e)
	m, e := NewBinaryRationalManager(qm.Z, BinaryRationalConfig{Diagnostics: diagnostics})
	serviceMust(t, e)
	t.Cleanup(func() {
		serviceMust(t, m.CloseChecked())
		serviceMust(t, diagnostics.Close())
	})
	m.debugIntegerValue, e = diagnostics.NewValue()
	serviceMust(t, e)
	serviceMust(t, diagnostics.Manager.Z.SetLimbs(&m.debugIntegerValue.Raw.Num, []uint32{7, 0, 1}))
	m.debugIntegerPending = true
	ref := m.debugIntegerValue.Raw.Num.Cell
	original := diagnostics.Manager.Z.Cell(ref)
	bad := original
	bad.Backing.Requested = uint64(math.MaxUint32) + 1
	diagnostics.Manager.Z.Store(ref, bad)
	if e := m.CleanupDebugInteger(); !errors.Is(e, arena.ErrBounds) || !m.debugIntegerPending || m.debugIntegerValue.Raw.Num.Cell != ref {
		t.Fatal("physical refusal lost debug integer copy", e, m.debugIntegerValue)
	}
	if e := m.open(); !errors.Is(e, ErrBinaryDebugIntegerPending) {
		t.Fatal("retained debug integer copy allowed reentry", e)
	}
	diagnostics.Manager.Z.Store(ref, original)
	serviceMust(t, m.CleanupDebugInteger())
	if m.debugIntegerPending || m.debugIntegerValue.ownerID != 0 {
		t.Fatal("owning retry retained debug integer copy")
	}
}
