//go:build g3debug

package numeral

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestBinaryDebugSnapshotsRetainDestructorOrderAndRetry(t *testing.T) {
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
	inner, e := diagnostics.NewValue()
	serviceMust(t, e)
	outer, e := diagnostics.NewValue()
	serviceMust(t, e)
	serviceMust(t, diagnostics.Manager.Z.SetLimbs(&inner.Raw.Num, []uint32{7, 0, 1}))
	serviceMust(t, diagnostics.Manager.Z.SetLimbs(&outer.Raw.Num, []uint32{9, 0, 1}))
	innerRef, outerRef := inner.Raw.Num.Cell, outer.Raw.Num.Cell
	innerOriginal := diagnostics.Manager.Z.Cell(innerRef)
	outerOriginal := diagnostics.Manager.Z.Cell(outerRef)
	innerBad := innerOriginal
	innerBad.Backing.Requested = uint64(math.MaxUint32) + 1
	diagnostics.Manager.Z.Store(innerRef, innerBad)
	if e := m.debugDel(&inner); !errors.Is(e, arena.ErrBounds) || inner.ownerID != 0 {
		t.Fatal("first stopped destructor did not transfer its owner", e)
	}
	if e := m.debugDel(&outer); !errors.Is(e, ErrBinaryDebugSnapshotPending) || outer.ownerID != 0 || len(m.debugSnapshotQueue) != 2 || m.debugSnapshotQueue[0].Raw.Num.Cell != innerRef || m.debugSnapshotQueue[1].Raw.Num.Cell != outerRef {
		t.Fatal("later destructor did not retain source order", e, m.debugSnapshotQueue)
	}
	if e := m.open(); !errors.Is(e, ErrBinaryDebugSnapshotPending) {
		t.Fatal("retained debug snapshots allowed reentry", e)
	}
	if e := m.CleanupDebugSnapshots(); !errors.Is(e, arena.ErrBounds) || len(m.debugSnapshotQueue) != 2 {
		t.Fatal("owning retry crossed first refusal", e)
	}
	diagnostics.Manager.Z.Store(innerRef, innerOriginal)
	outerBad := outerOriginal
	outerBad.Backing.Requested = uint64(math.MaxUint32) + 1
	diagnostics.Manager.Z.Store(outerRef, outerBad)
	if e := m.CleanupDebugSnapshots(); !errors.Is(e, arena.ErrBounds) || len(m.debugSnapshotQueue) != 1 || m.debugSnapshotQueue[0].Raw.Num.Cell != outerRef {
		t.Fatal("owning retry replayed completed prefix", e, m.debugSnapshotQueue)
	}
	diagnostics.Manager.Z.Store(outerRef, outerOriginal)
	serviceMust(t, m.CleanupDebugSnapshots())
	if len(m.debugSnapshotQueue) != 0 {
		t.Fatal("owning retry retained debug snapshots")
	}
	serviceMust(t, m.open())
}
