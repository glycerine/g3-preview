// Source: src/util/vector.h:102-120,159-167,191-202,536-541.
// Source author: Leonardo de Moura.
// Licensed under the MIT license; see LICENSE.txt.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"reflect"
	"testing"
)

func TestRuntimeUnsignedVectorAssignmentDestroyBeforeExactCopyFailure(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b})
	paramsMust(t, e)
	src, e := NewRuntimeUnsignedVector(r, b)
	paramsMust(t, e)
	dst, e := NewRuntimeUnsignedVector(r, b)
	paramsMust(t, e)
	for i := uint32(0); i < 4; i++ {
		paramsMust(t, src.Push(i+10))
	}
	paramsMust(t, dst.Push(99))
	old := dst.column.raw.block
	b.FailAfter(1)
	e = dst.Assign(src)
	if e == nil {
		t.Fatal("copy allocation refused after destruction")
	}
	if dst.column.raw.block != old || !dst.assignmentDestroyed || r.Stats().LiveBlocks != 1 {
		t.Fatal("literal dangling m_data, real consumed owner receipt", r.Stats())
	}
	if _, e = dst.Size(); !errors.Is(e, memory.ErrRuntimeOwner) {
		t.Fatal("source stale pointer reached read", e)
	}
	satAssignmentAuditBlocks(t, &src.column.raw)
	if b.Used() != r.Stats().TotalBytes+src.header+dst.header+unsignedCopyCold(t, src)+unsignedCopyCold(t, dst) {
		t.Fatal("named cold and surviving source root")
	}
	b.FailAfter(-1)
	if e = dst.Close(); !errors.Is(e, memory.ErrRuntimeOwner) {
		t.Fatal("source DT must not hide dangling pointer", e)
	}
	paramsMust(t, dst.CleanupStopped())
	paramsMust(t, src.Close())
	paramsMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestRuntimeUnsignedVectorCopyPreservesNonNullEmptyCapacity(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b})
	paramsMust(t, e)
	v, e := NewRuntimeUnsignedVector(r, b)
	paramsMust(t, e)
	for i := uint32(0); i < 5; i++ {
		paramsMust(t, v.Push(i))
	}
	paramsMust(t, v.Reset())
	out, e := v.Clone()
	paramsMust(t, e)
	n, e := out.Size()
	paramsMust(t, e)
	cap, e := out.Capacity()
	paramsMust(t, e)
	if n != 0 || cap != 5 || out.column.raw.block == (memory.Block{}) {
		t.Fatal("source m_data test, exact capacity empty copy", n, cap)
	}
	raw, e := r.Size(out.column.raw.block)
	paramsMust(t, e)
	if raw != 28 {
		t.Fatal("copy exact-cap source request", raw)
	}
	satAssignmentAuditBlocks(t, &v.column.raw, out.column.raw.block)
	if b.Used() != r.Stats().TotalBytes+v.header+out.header+unsignedCopyCold(t, v)+unsignedCopyCold(t, out) {
		t.Fatal("actual owned copy cold graph")
	}
	paramsMust(t, out.Close())
	paramsMust(t, v.Close())
	paramsMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestRuntimeUnsignedVectorCopySizeT32ConversionBeforePhysicalAllocation(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 32})
	paramsMust(t, e)
	v, e := NewRuntimeUnsignedVector(r, b)
	paramsMust(t, e)
	paramsMust(t, v.Push(7))
	paramsMust(t, v.Reset())
	// Malformed-capacity source continuation domain: actual size_t conversion
	// wraps sizeof(unsigned)*capacity+8 to8. No giant host allocation is made.
	paramsMust(t, satAssignmentWriteWord(r, v.column.raw.block, 0, 0x40000000))
	out, e := v.Clone()
	paramsMust(t, e)
	n, e := r.Size(out.column.raw.block)
	paramsMust(t, e)
	if n != 8 {
		t.Fatal("actual source size_t request conversion", n)
	}
	cap, e := out.Capacity()
	paramsMust(t, e)
	if cap != 0x40000000 {
		t.Fatal("source header publication", cap)
	}
	satAssignmentAuditBlocks(t, &v.column.raw, out.column.raw.block)
	if b.Used() != r.Stats().TotalBytes+v.header+out.header+unsignedCopyCold(t, v)+unsignedCopyCold(t, out) {
		t.Fatal("raw extent and concrete cold owners")
	}
	paramsMust(t, out.Close())
	paramsMust(t, v.Close())
	paramsMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestRuntimeUnsignedVectorAssignRetryRetainsActualAcquisitionHistory(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	paramsMust(t, e)
	src, e := NewRuntimeUnsignedVector(r, b)
	paramsMust(t, e)
	dst, e := NewRuntimeUnsignedVector(r, b)
	paramsMust(t, e)
	empty, e := NewRuntimeUnsignedVector(r, b)
	paramsMust(t, e)
	paramsMust(t, src.Resize(30000, 7))
	paramsMust(t, r.SetMaxSize(1))
	first := dst.Assign(src)
	var f memory.Failure
	a := dst.copyAcquisitions
	if !errors.As(first, &f) || f.ErrorCode() != 101 || !r.State().OutOfMemory || a == nil || !a.called || !a.acquired || a.published || a.block == (memory.Block{}) || dst.column.raw.block != (memory.Block{}) {
		t.Fatal("actual TLS postphysical acquisition", first, r.State())
	}
	old := a.block
	unsignedCopyPhysical(t, r, b, src, dst, empty)
	state := r.State()
	used := b.Used()
	paramsMust(t, dst.Assign(dst))
	paramsMust(t, dst.Assign(empty))
	if dst.copyAcquisitions != a || a.block != old || r.State() != state || b.Used() != used {
		t.Fatal("literal self/null branches must leave old acquisition unread/retained")
	}
	paramsMust(t, r.SetMaxSize(0))
	paramsMust(t, dst.Assign(src))
	current := dst.copyAcquisitions
	if current != a || a.next != nil || a.block != old || dst.copySerial != a.invocation+1 || dst.column.raw.block == (memory.Block{}) || !r.State().OutOfMemory {
		t.Fatal("native-valid retry current pointer and older unjoined root", r.State())
	}
	for _, i := range []uint32{0, 14999, 29999} {
		x, e := dst.At(i)
		paramsMust(t, e)
		if x != 7 {
			t.Fatal("real copied payload", i, x)
		}
	}
	if dst.assignmentCause != first {
		t.Fatal("first source cause sticky")
	}
	unsignedCopyPhysical(t, r, b, src, dst, empty)
	paramsMust(t, dst.Close())
	paramsMust(t, empty.Close())
	paramsMust(t, src.Close())
	paramsMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func unsignedCopyCold(t *testing.T, v *RuntimeUnsignedVector) uint64 {
	t.Helper()
	var n uint64
	last := v.copySerial + 1
	for a := v.copyAcquisitions; a != nil; a = a.next {
		if a.owner != v || a.runtime != v.column.raw.runtime || a.invocation == 0 || a.invocation >= last || a.header != uint64(reflect.TypeOf(*a).Size()) || !a.called {
			t.Fatal("actual cold acquisition identity", a.invocation, last, a.header)
		}
		last = a.invocation
		n += a.header
	}
	return n
}
func unsignedCopyPhysical(t *testing.T, r *memory.Runtime, b *arena.Budget, owners ...*RuntimeUnsignedVector) {
	t.Helper()
	var roots []memory.Block
	var cold uint64
	primary := &owners[0].column.raw
	for _, v := range owners {
		cold += v.header + unsignedCopyCold(t, v)
		if &v.column.raw != primary {
			if !v.assignmentDestroyed && !v.column.raw.pendingReplaces && v.column.raw.block != (memory.Block{}) {
				roots = append(roots, v.column.raw.block)
			}
			if v.column.raw.pending != (memory.Block{}) {
				roots = append(roots, v.column.raw.pending)
			}
		}
		for a := v.copyAcquisitions; a != nil; a = a.next {
			if a.block != (memory.Block{}) {
				if !a.acquired || a.published {
					t.Fatal("actual node ownership phase")
				}
				roots = append(roots, a.block)
			}
		}
	}
	satAssignmentAuditBlocks(t, primary, roots...)
	if b.Used() != r.Stats().TotalBytes+cold {
		t.Fatal("exact raw and actual cold contexts", b.Used(), r.Stats().TotalBytes, cold)
	}
}

func TestRuntimeUnsignedVectorAcquisitionHistoryStoppedOwningCleanup(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64, ThreadLocal: true})
	paramsMust(t, e)
	src, e := NewRuntimeUnsignedVector(r, b)
	paramsMust(t, e)
	dst, e := NewRuntimeUnsignedVector(r, b)
	paramsMust(t, e)
	paramsMust(t, src.Resize(30000, 9))
	paramsMust(t, r.SetMaxSize(1))
	first := dst.Assign(src)
	var f memory.Failure
	if !errors.As(first, &f) || f.ErrorCode() != 101 {
		t.Fatal(first)
	}
	older := dst.copyAcquisitions
	paramsMust(t, r.SetMaxSize(0))
	paramsMust(t, dst.Assign(src))
	unsignedCopyPhysical(t, r, b, src, dst)
	satHeapTLSNearMinimum(t, r)
	sourceStop := dst.SourceFinalize()
	if !errors.Is(sourceStop, arena.ErrBounds) || dst.column.raw.block != (memory.Block{}) || older.block == (memory.Block{}) {
		t.Fatal("source DT current only, older unjoined remains", sourceStop)
	}
	unsignedCopyPhysical(t, r, b, src, dst)
	cleanup := dst.CleanupStopped()
	if !errors.Is(cleanup, arena.ErrBounds) || older.block != (memory.Block{}) || !older.released || dst.copyAcquisitions != older || dst.closed {
		t.Fatal("actual owning older-node free/TLS stop", cleanup)
	}
	if older.cleanupCause != cleanup || dst.copyCleanupCause != cleanup || dst.assignmentCause != first || dst.destructorCause != sourceStop {
		t.Fatal("first causes and exact node cleanup receipt")
	}
	unsignedCopyPhysical(t, r, b, src, dst)
	paramsMust(t, dst.CleanupStopped())
	if dst.copyCleanupCause != cleanup || dst.destructorCause != sourceStop {
		t.Fatal("cleanup acknowledgment repaired source history")
	}
	if e = src.Close(); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("remaining actual source caller free", e)
	}
	paramsMust(t, src.CleanupStopped())
	paramsMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal("each actual node/header/raw refund once", b.Used())
	}
}
