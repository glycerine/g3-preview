// Source: src/util/bit_vector.h:69-239; src/util/bit_vector.cpp:188-235; src/util/hash.cpp.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestBitVectorRuntimeHashCutoffAndEveryStreamPrefix(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b})
	heapMust(t, e)
	v, e := NewRuntimeBitVectorWords(r, b, []uint32{0x81828384, 0xf0}, 40)
	heapMust(t, e)
	for n := uint32(40); ; n-- {
		heapMust(t, v.Shrink(n))
		got, e := v.Hash()
		heapMust(t, e)
		if got != WordBytes([]uint32{0x81828384, 0xf0}, n/8, 0) {
			t.Fatal(n, got)
		}
		if n == 0 {
			break
		}
	}
	heapMust(t, v.Close())
	v, e = NewRuntimeBitVectorWords(r, b, []uint32{0x81828384}, 10)
	heapMust(t, e)
	want := "1110000100"
	for k := 0; k <= len(want); k++ {
		writer := &uintSetPrefixWriter{remaining: k}
		e = v.Display(writer)
		if string(writer.bytes) != want[:k] {
			t.Fatal(k, string(writer.bytes), want[:k])
		}
		if k < len(want) {
			if !errors.Is(e, uintSetStreamRefusal) {
				t.Fatal(k, e)
			}
		} else {
			heapMust(t, e)
		}
	}
	heapMust(t, v.Close())
	heapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
func TestBitVectorRuntimeDistinctProviderSwapFieldsAndColdGuards(t *testing.T) {
	for mode := 0; mode < 4; mode++ {
		b := arena.NewBudget(1 << 24)
		d := arena.NewBudget(1 << 24)
		r, e := NewRuntime(RuntimeConfig{Budget: b})
		heapMust(t, e)
		s, e := NewRuntime(RuntimeConfig{Budget: d})
		heapMust(t, e)
		a, e := NewRuntimeBitVector(r, b)
		heapMust(t, e)
		c, e := NewRuntimeBitVector(s, d)
		heapMust(t, e)
		if mode&1 != 0 {
			heapMust(t, a.Resize(9, true))
		}
		if mode&2 != 0 {
			heapMust(t, c.Resize(65, false))
			heapMust(t, c.SetBit(64))
		}
		an, ac, ab := a.numBits, a.capacity, a.block
		cn, cc, cb := c.numBits, c.capacity, c.block
		beforeR, beforeS := r.State(), s.State()
		beforeB, beforeD := b.Used(), d.Used()
		b.FailAfter(0)
		d.FailAfter(0)
		heapMust(t, a.Swap(c))
		if a.numBits != cn || a.capacity != cc || a.block != cb || c.numBits != an || c.capacity != ac || c.block != ab || a.runtime != s || c.runtime != r {
			t.Fatal("source all3fields plus rawprovider")
		}
		if r.State() != beforeR || s.State() != beforeS || b.Used() != beforeB || d.Used() != beforeD {
			t.Fatal("swap allocation")
		}
		b.FailAfter(-1)
		d.FailAfter(-1)
		if b.Used() != r.Stats().TotalBytes+a.ownerBytes || d.Used() != s.Stats().TotalBytes+c.ownerBytes {
			t.Fatal("provider physical plus ORIGINAL cold")
		}
		cp := *a
		if e = cp.Close(); !errors.Is(e, ErrRuntimeVectorOwner) {
			t.Fatal(e)
		}
		heapMust(t, a.Close())
		heapMust(t, c.Close())
		heapMust(t, r.Close())
		heapMust(t, s.Close())
		if b.Used() != 0 || d.Used() != 0 {
			t.Fatal(b.Used(), d.Used())
		}
	}
}
func TestBitVectorRuntimeWarmZeroAllocAndPhysicalPlateau(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b})
	heapMust(t, e)
	v, e := NewRuntimeBitVectorReserved(r, b, 256)
	heapMust(t, e)
	heapMust(t, v.Resize(65, false))
	before := r.State()
	used := b.Used()
	b.FailAfter(0)
	var runErr error
	n := testing.AllocsPerRun(100, func() {
		if runErr != nil {
			return
		}
		if runErr = v.SetBool(64, true); runErr != nil {
			return
		}
		if runErr = v.Neg(); runErr != nil {
			return
		}
		_, runErr = v.Hash()
	})
	heapMust(t, runErr)
	if n != 0 || r.State() != before || b.Used() != used {
		t.Fatal(n, b.Used(), used, r.State(), before)
	}
	b.FailAfter(-1)
	heapMust(t, v.Close())
	heapMust(t, r.Close())
}
func TestBitVectorRuntimeCapturedRawSpanConstructor(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b})
	heapMust(t, e)
	src, e := r.Allocate(4)
	heapMust(t, e)
	heapMust(t, r.WriteUint32(src, 0, 7))
	span := RuntimeBitVectorSpan{Runtime: r, Block: src}
	v, e := NewRuntimeBitVectorSpan(r, b, span, 65)
	if !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	if v == nil || v.numBits != 65 || v.capacity != 3 {
		t.Fatal(v)
	}
	w, e := v.read(0)
	heapMust(t, e)
	if w != 7 {
		t.Fatal(w)
	}
	// Named caller span is a borrow; its actual source block is independently held.
	heapMust(t, v.CleanupStopped())
	heapMust(t, r.Deallocate(&src))
	heapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
