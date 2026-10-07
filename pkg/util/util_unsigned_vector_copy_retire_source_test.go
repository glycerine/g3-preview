// Source: src/util/vector.h:102-120,159-167,191-202.
// Source author: Leonardo de Moura.
// Licensed under the MIT license; see LICENSE.txt.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestRuntimeUnsignedVectorSuccessfulAssignRetiresTransferredAcquisitionMetadata(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	r, e := NewRuntime(RuntimeConfig{Budget: b})
	paramsMust(t, e)
	src, e := NewRuntimeUnsignedVector(r, b)
	paramsMust(t, e)
	dst, e := NewRuntimeUnsignedVector(r, b)
	paramsMust(t, e)
	paramsMust(t, src.Resize(9, 17))
	for i := 0; i < 64; i++ {
		paramsMust(t, dst.Assign(src))
		unsignedCopyPhysical(t, r, b, src, dst)
		if dst.copyAcquisitions != nil || unsignedCopyCold(t, dst) != 0 {
			t.Fatal("completed pointer transfer must retire cold invocation owner", i, unsignedCopyCold(t, dst))
		}
		if b.Used() != r.Stats().TotalBytes+src.header+dst.header {
			t.Fatal("bounded exact named cold graph", i, b.Used())
		}
		n, e := dst.Size()
		paramsMust(t, e)
		if n != 9 {
			t.Fatal(n)
		}
		x, e := dst.At(8)
		paramsMust(t, e)
		if x != 17 {
			t.Fatal(x)
		}
	}
	paramsMust(t, dst.Close())
	paramsMust(t, src.Close())
	paramsMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestRuntimeUnsignedVectorSuccessfulRetryRetiresPrephysicalEmptyInvocation(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	r, e := NewRuntime(RuntimeConfig{Budget: b})
	paramsMust(t, e)
	src, e := NewRuntimeUnsignedVector(r, b)
	paramsMust(t, e)
	dst, e := NewRuntimeUnsignedVector(r, b)
	paramsMust(t, e)
	empty, e := NewRuntimeUnsignedVector(r, b)
	paramsMust(t, e)
	paramsMust(t, src.Resize(9, 13))
	var first error
	for i := 0; i < 32; i++ {
		paramsMust(t, dst.Assign(empty))
		live := r.Stats().LiveBlocks
		// The existing Runtime directory is already prepared by src. One actual
		// cold context reservation completes, then physical Allocate is refused.
		b.FailAfter(1)
		e = dst.Assign(src)
		var failure Failure
		if !errors.As(e, &failure) || failure.ErrorCode() != 101 || !r.State().OutOfMemory {
			t.Fatal("actual prephysical allocation refusal", e, r.State())
		}
		a := dst.copyAcquisitions
		if a == nil || !a.called || a.acquired || a.published || a.block != (Block{}) || r.Stats().LiveBlocks != live || dst.Column.Raw.Block != (Block{}) {
			t.Fatal("no acquired token", i, e, r.Stats())
		}
		if first == nil {
			first = e
		}
		if dst.assignmentCause != first {
			t.Fatal("sticky first source cause")
		}
		unsignedCopyPhysical(t, r, b, src, dst, empty)
		b.FailAfter(-1)
		paramsMust(t, dst.Assign(src))
		unsignedCopyPhysical(t, r, b, src, dst, empty)
		if dst.copyAcquisitions != nil || unsignedCopyCold(t, dst) != 0 {
			t.Fatal("completed retry has no empty invocation owner", i, unsignedCopyCold(t, dst))
		}
		if dst.assignmentCause != first {
			t.Fatal("retirement must not clear source cause")
		}
		x, e := dst.At(8)
		paramsMust(t, e)
		if x != 13 {
			t.Fatal(x)
		}
	}
	paramsMust(t, dst.Close())
	paramsMust(t, empty.Close())
	paramsMust(t, src.Close())
	paramsMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
