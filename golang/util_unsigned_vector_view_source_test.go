// Source: src/util/vector.h:449-451; src/util/permutation.h:56-91.
// Source author: Leonardo de Moura.
// Licensed under the MIT license; see LICENSE.txt.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"github.com/glycerine/g3/golang/utilbits"
	"testing"
)

func TestRuntimeUnsignedVectorCPtrCapturesPhysicalLifetimeAndSameCapacityMutation(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b})
	paramsMust(t, e)
	v, e := NewRuntimeUnsignedVector(r, b)
	paramsMust(t, e)
	paramsMust(t, v.Push(0))
	paramsMust(t, v.Push(1))
	p, e := v.CPtr()
	paramsMust(t, e)
	paramsMust(t, v.Set(1, 7))
	x, e := p.Read(1)
	paramsMust(t, e)
	if x != 7 {
		t.Fatal(x)
	}
	paramsMust(t, v.Reset())
	x, e = p.Read(1)
	paramsMust(t, e)
	if x != 7 {
		t.Fatal("raw pointer not logical index", x)
	}
	paramsMust(t, v.Push(0))
	paramsMust(t, v.Push(1))
	paramsMust(t, v.Push(2))
	if _, e = p.Read(0); !errors.Is(e, memory.ErrRuntimeOwner) {
		t.Fatal("captured old physical token after real realloc", e)
	}
	p, e = v.CPtr()
	paramsMust(t, e)
	state, used := r.State(), b.Used()
	b.FailAfter(0)
	paramsMust(t, utilbits.ApplyPermutationCoreRuntimeUint32(3, p, p))
	if r.State() != state || b.Used() != used {
		t.Fatal("borrowed kernel allocations")
	}
	unsignedCopyPhysical(t, r, b, v)
	b.FailAfter(-1)
	paramsMust(t, v.Close())
	if _, e = p.Read(0); !errors.Is(e, memory.ErrRuntimeOwner) {
		t.Fatal(e)
	}
	paramsMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
func TestRuntimeUnsignedVectorCPtrDoesNotReadUnreadHeader(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b})
	paramsMust(t, e)
	v, e := NewRuntimeUnsignedVector(r, b)
	paramsMust(t, e)
	h, e := r.Allocate(12)
	paramsMust(t, e)
	v.column.raw.block = h
	paramsMust(t, r.WriteUint32(h, 8, 23))
	p, e := v.CPtr()
	paramsMust(t, e)
	x, e := p.Read(0)
	paramsMust(t, e)
	if x != 23 {
		t.Fatal(x)
	}
	unsignedCopyPhysical(t, r, b, v)
	paramsMust(t, v.Close())
	paramsMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
