// Licensed under the MIT license.
// Source-owner extraction bridge charges and nominal errors, tests first.
package g3

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
)

func TestMemoryBridgeColdHeadersAndConstructorUnwind(t *testing.T) {
	for stop := int64(0); stop < 3; stop++ {
		b := arena.NewBudget(1 << 20)
		r, e := newUtilMemoryRuntime(utilMemoryRuntimeConfig{Budget: b, SizeTBits: 64})
		smallObjectMust(t, e)
		baseline := b.Used()
		b.FailAfter(stop)
		a, e := newUtilSmallObjectAllocator(utilSmallObjectConfig{Runtime: r, Profile: utilSmallObjectProfile{PointerBits: 64}})
		if a != nil {
			b.FailAfter(-1)
			_ = a.Close()
		}
		if e != arena.ErrInjectedFailure || b.Used() != baseline || r.Stats().LiveBlocks != 0 || r.State().GlobalCount != 0 {
			t.Fatal("three actual cold owner headers/unwind", stop, e, b.Used(), baseline, r.Stats(), r.State())
		}
		b.FailAfter(-1)
		smallObjectMust(t, r.Close())
		if b.Used() != 0 {
			t.Fatal("runtime charge", b.Used())
		}
	}
	b := arena.NewBudget(1 << 20)
	r, e := newUtilMemoryRuntime(utilMemoryRuntimeConfig{Budget: b, SizeTBits: 64})
	smallObjectMust(t, e)
	before := b.Used()
	a, e := newUtilSmallObjectAllocator(utilSmallObjectConfig{Runtime: r, Profile: utilSmallObjectProfile{PointerBits: 64}})
	smallObjectMust(t, e)
	if a.Stats().OwnerBytes <= uint64(reflect.TypeOf(*a).Size()) || b.Used()-before != a.Stats().TotalBytes {
		t.Fatal("owned core/IO/wrapper headers missing", a.Stats(), b.Used(), before)
	}
	smallObjectMust(t, a.Close())
	smallObjectMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal("header teardown", b.Used())
	}
}

func TestMemoryBridgeNominalOOMAndCountErrors(t *testing.T) {
	for _, code := range []ErrorCode{ErrorMemout, ErrorAllocExceeded} {
		b := arena.NewBudget(1 << 20)
		r, e := newUtilMemoryRuntime(utilMemoryRuntimeConfig{Budget: b, SizeTBits: 64})
		smallObjectMust(t, e)
		if code == ErrorMemout {
			smallObjectMust(t, r.SetMaxSize(1))
		} else {
			smallObjectMust(t, r.SetMaxCount(1))
			h, e := r.Allocate(1)
			smallObjectMust(t, e)
			smallObjectMust(t, r.Deallocate(&h))
		}
		h, e := r.Allocate(1)
		var z Z3Error
		if !errors.As(e, &z) || z.ErrorCode() != code || h != (utilMemoryBlock{}) {
			t.Fatal("main nominal source error bridge", code, h, e)
		}
		smallObjectMust(t, r.Close())
		if b.Used() != 0 {
			t.Fatal("error owner cleanup", b.Used())
		}
	}
}

func TestMemoryBridgeDefaultExceptionAndCopiedOwner(t *testing.T) {
	e := memoryErrorTransport(memory.Failure{Kind: memory.FailureVectorOverflow})
	var d DefaultException
	if !errors.As(e, &d) || d.HasErrorCode() || d.ErrorCode() != ErrorOK || d.Message() != "Overflow encountered when expanding vector" {
		t.Fatal("nominal main no-code class", e)
	}
	b := arena.NewBudget(1 << 20)
	r, e := newUtilMemoryRuntime(utilMemoryRuntimeConfig{Budget: b, SizeTBits: 64})
	smallObjectMust(t, e)
	before := b.Used()
	b.FailAfter(0)
	if a, e := newUtilSmallObjectAllocator(utilSmallObjectConfig{Runtime: r, Profile: utilSmallObjectProfile{PointerBits: 32}}); e != arena.ErrConfig || a != nil || b.Used() != before {
		t.Fatal("invalid profile before cold allocation", a, e, b.Used(), before)
	}
	b.FailAfter(-1)
	a, e := newUtilSmallObjectAllocator(utilSmallObjectConfig{Runtime: r, Profile: utilSmallObjectProfile{PointerBits: 64}})
	smallObjectMust(t, e)
	copied := *a
	charge := b.Used()
	state := r.State()
	if e := copied.Close(); e != ErrSmallObjectOwner || b.Used() != charge || r.State() != state {
		t.Fatal("copied wrapper changed ownership", e, b.Used(), charge, r.State(), state)
	}
	smallObjectMust(t, a.Close())
	if _, e := a.Allocate(7); e != arena.ErrClosed {
		t.Fatal("closed forwarded owner", e)
	}
	smallObjectMust(t, a.Close())
	smallObjectMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal("bridge copied/closed charges", b.Used())
	}
}

func TestMemoryBridgePendingFinalizationRetainsAllHeaders(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	r, e := newUtilMemoryRuntime(utilMemoryRuntimeConfig{Budget: b, SizeTBits: 64})
	smallObjectMust(t, e)
	a, e := newUtilSmallObjectAllocator(utilSmallObjectConfig{Runtime: r, Profile: utilSmallObjectProfile{PointerBits: 64}})
	smallObjectMust(t, e)
	ref, e := a.Allocate(^uint64(0))
	if e != arena.ErrBounds || ref != (utilSmallObjectRef{}) || a.Stats().PendingBlocks != 1 {
		t.Fatal("actual wrapped pending allocation", ref, e, a.Stats())
	}
	before := a.Stats()
	charge := b.Used()
	if e := a.Close(); e != ErrSmallObjectPending || a.Stats() != before || b.Used() != charge {
		t.Fatal("wrapper/IO/core charge lost before pending cleanup", e, a.Stats(), before, b.Used(), charge)
	}
	smallObjectMust(t, a.CleanupPending())
	if _, e := a.Allocate(1); e != arena.ErrClosed {
		t.Fatal("pending cleanup reopened source destructor", e)
	}
	smallObjectMust(t, a.Close())
	if a.Stats().TotalBytes != 0 || b.Used() != r.Stats().TotalBytes || r.Stats().LiveBlocks != 0 {
		t.Fatal("actual pending owner finalization", a.Stats(), r.Stats(), b.Used())
	}
	smallObjectMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal("pending bridge teardown", b.Used())
	}
}
