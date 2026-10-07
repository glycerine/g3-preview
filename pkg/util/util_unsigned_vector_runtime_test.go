// Source: src/util/vector.h:59-100,178-189,215-240,350-365,531-546;
// src/util/permutation.cpp:21-32.
// Source author: Leonardo de Moura.
// Licensed under the MIT license; see LICENSE.txt.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func permutationPhysical(t *testing.T, p *RuntimePermutation, r *Runtime, b *arena.Budget) {
	t.Helper()
	roots := []Block{}
	c := &p.inverse.Column.Raw
	if !c.PendingReplaces && c.Block != (Block{}) {
		roots = append(roots, c.Block)
	}
	if c.Pending != (Block{}) {
		roots = append(roots, c.Pending)
	}
	runtimeVectorAuditBlocks(t, &p.forward.Column.Raw, roots...)
	if b.Used() != r.Stats().TotalBytes+p.header {
		t.Fatal("exact named raw/cold ownership", b.Used(), r.Stats(), p.header)
	}
}
func TestRuntimeUnsignedVectorGeometryResetAndWarmReuse(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	r, e := NewRuntime(RuntimeConfig{Budget: b})
	paramsMust(t, e)
	v, e := NewRuntimeUnsignedVector(r, b)
	paramsMust(t, e)
	for i := uint32(0); i < 5; i++ {
		paramsMust(t, v.Push(i))
	}
	cap, e := v.Capacity()
	paramsMust(t, e)
	if cap != 5 {
		t.Fatal("literal geometric capacity", cap)
	}
	size, e := r.Size(v.Column.Raw.Block)
	paramsMust(t, e)
	if size != 28 {
		t.Fatal("source header+payload", size)
	}
	runtimeVectorAuditBlocks(t, &v.Column.Raw)
	if b.Used() != r.Stats().TotalBytes+v.header {
		t.Fatal("wrapper cold charge")
	}
	h := v.Column.Raw.Block
	state := r.State()
	used := b.Used()
	b.FailAfter(0)
	paramsMust(t, v.Reset())
	for i := uint32(0); i < 5; i++ {
		paramsMust(t, v.Push(20+i))
	}
	if v.Column.Raw.Block != h || r.State() != state || b.Used() != used {
		t.Fatal("reset/push warm storage")
	}
	b.FailAfter(-1)
	paramsMust(t, v.Close())
	paramsMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
func TestRuntimePermutationEveryConstructorReservationSourceMemberUnwind(t *testing.T) {
	success := -1
	for ordinal := int64(0); ordinal < 100; ordinal++ {
		b := arena.NewBudget(1 << 20)
		r, e := NewRuntime(RuntimeConfig{Budget: b})
		paramsMust(t, e)
		b.FailAfter(ordinal)
		p, e := NewRuntimePermutation(r, b, 9)
		if p != nil {
			permutationPhysical(t, p, r, b)
		} else if r.Stats().LiveBlocks != 0 {
			t.Fatal("constructor header refusal created raw allocation")
		}
		if e == nil {
			success = int(ordinal)
			permutationExpect(t, p, []uint32{0, 1, 2, 3, 4, 5, 6, 7, 8})
		} else {
			if !errors.Is(e, arena.ErrInjectedFailure) {
				var f Failure
				if !errors.As(e, &f) || f.ErrorCode() != 101 || !r.State().OutOfMemory {
					t.Fatal("actual refusal category", ordinal, e)
				}
			}
			if p != nil {
				if p.destructorStage != 2 || !p.inverse.sourceDone || !p.forward.sourceDone || r.Stats().LiveBlocks != 0 {
					t.Fatal("failed constructor completed members must destruct inverse before forward", ordinal, p.destructorStage, r.Stats())
				}
			}
		}
		b.FailAfter(-1)
		if p != nil {
			paramsMust(t, p.Close())
		}
		paramsMust(t, r.Close())
		if b.Used() != 0 {
			t.Fatal("constructor abort leak", ordinal, b.Used())
		}
		if success >= 0 {
			break
		}
	}
	if success < 0 {
		t.Fatal("did not reach first success")
	}
	t.Log("first success reservation", success)
}

func TestRuntimeUnsignedVectorResetDoesNotReadOldSizeHeader(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	r, e := NewRuntime(RuntimeConfig{Budget: b})
	paramsMust(t, e)
	v, e := NewRuntimeUnsignedVector(r, b)
	paramsMust(t, e)
	// Checked unsafe-continuation domain: reached cap write completed but size
	// was not initialized. Source reset only writes SIZE and reads no old size.
	v.Column.Raw.Block, e = r.Allocate(16)
	paramsMust(t, e)
	paramsMust(t, WriteRuntimeWord(r, v.Column.Raw.Block, 0, 2))
	paramsMust(t, v.Reset())
	n, e := v.Size()
	paramsMust(t, e)
	if n != 0 {
		t.Fatal(n)
	}
	runtimeVectorAuditBlocks(t, &v.Column.Raw)
	paramsMust(t, v.Close())
	paramsMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestRuntimeUnsignedVectorResizeRetainsSizeUntilAllCapacityReady(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	r, e := NewRuntime(RuntimeConfig{Budget: b})
	paramsMust(t, e)
	v, e := NewRuntimeUnsignedVector(r, b)
	paramsMust(t, e)
	paramsMust(t, v.Push(77))
	b.FailAfter(1)
	if e = v.Resize(9, 42); e == nil {
		t.Fatal("later expansion should refuse")
	}
	n, e := v.Size()
	paramsMust(t, e)
	if n != 1 {
		t.Fatal("source while(capacity) must preserve old SIZE", n)
	}
	runtimeVectorAuditBlocks(t, &v.Column.Raw)
	b.FailAfter(-1)
	paramsMust(t, v.Resize(9, 42))
	n, e = v.Size()
	paramsMust(t, e)
	if n != 9 {
		t.Fatal(n)
	}
	for i := uint32(0); i < n; i++ {
		x, e := v.At(i)
		paramsMust(t, e)
		want := uint32(42)
		if i == 0 {
			want = 77
		}
		if x != want {
			t.Fatal(i, x, want)
		}
	}
	paramsMust(t, v.Close())
	paramsMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestRuntimeUnsignedVectorAndPermutationOwnerAndProviderGuards(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	other := arena.NewBudget(1 << 20)
	r, e := NewRuntime(RuntimeConfig{Budget: b})
	paramsMust(t, e)
	before := b.Used()
	other.FailAfter(0)
	if _, e = NewRuntimeUnsignedVector(r, other); !errors.Is(e, arena.ErrConfig) {
		t.Fatal(e)
	}
	if _, e = NewRuntimePermutation(r, other, 2); !errors.Is(e, arena.ErrConfig) {
		t.Fatal(e)
	}
	if b.Used() != before || other.Used() != 0 {
		t.Fatal("foreign config reservation")
	}
	v, e := NewRuntimeUnsignedVector(r, b)
	paramsMust(t, e)
	copyV := *v
	if e = copyV.Close(); !errors.Is(e, ErrRuntimeVectorOwner) {
		t.Fatal("copied vector", e)
	}
	p, e := NewRuntimePermutation(r, b, 0)
	paramsMust(t, e)
	copyP := *p
	if e = copyP.Close(); !errors.Is(e, ErrRuntimeVectorOwner) {
		t.Fatal("copied permutation", e)
	}
	paramsMust(t, v.Close())
	if _, e = v.Size(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("ended vector", e)
	}
	paramsMust(t, p.Close())
	paramsMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
