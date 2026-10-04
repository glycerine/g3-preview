// Source: src/util/bit_vector.h; src/util/bit_vector.cpp; src/test/bit_vector.cpp.
// Licensed under the MIT license. Source author: Leonardo de Moura, 2006-10-03.
package g3

import (
	"bytes"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestBitVectorRuntimeRegisteredCrashAndReset(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64})
	heapMust(t, e)
	v, e := NewRuntimeBitVector(r, b)
	heapMust(t, e)
	for _, first := range []bool{true, false} {
		heapMust(t, v.Reset())
		heapMust(t, v.Push(first))
		heapMust(t, v.Resize(64, !first))
		for _, p := range []struct {
			i    uint32
			want bool
		}{{0, first}, {1, !first}, {63, !first}} {
			got, e := v.Get(p.i)
			heapMust(t, e)
			if got != p.want {
				t.Fatal(p, got)
			}
		}
	}
	for n := uint32(1); n < 84; n++ {
		bit := n%2 != 0
		heapMust(t, v.Reset())
		heapMust(t, v.Resize(n, bit))
		for i := uint32(0); i < n; i++ {
			got, e := v.Get(i)
			heapMust(t, e)
			if got != bit {
				t.Fatal(n, i, got)
			}
		}
		for sz2 := n; sz2 < n+10; sz2++ {
			heapMust(t, v.Resize(sz2, !bit))
			for i := n; i < sz2; i++ {
				got, e := v.Get(i)
				heapMust(t, e)
				if got == bit {
					t.Fatal(n, sz2, i, got)
				}
			}
		}
	}
	heapMust(t, v.Close())
	heapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
func TestBitVectorRuntimeIndependentWorldsAndShift(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64})
	heapMust(t, e)
	a, e := NewRuntimeBitVector(r, b)
	heapMust(t, e)
	c, e := NewRuntimeBitVector(r, b)
	heapMust(t, e)
	for x := uint32(0); x < 32; x++ {
		for y := uint32(0); y < 32; y++ {
			heapMust(t, a.Reset())
			heapMust(t, c.Reset())
			heapMust(t, a.Resize(5, false))
			heapMust(t, c.Resize(5, false))
			for i := uint32(0); i < 5; i++ {
				heapMust(t, a.SetBool(i, x&(1<<i) != 0))
				heapMust(t, c.SetBool(i, y&(1<<i) != 0))
			}
			eq, e := a.Equal(c)
			heapMust(t, e)
			if eq != (x == y) {
				t.Fatal(x, y, "equal")
			}
			contains, e := a.Contains(c)
			heapMust(t, e)
			if contains != (x&y == y) {
				t.Fatal(x, y, "contains")
			}
			heapMust(t, a.Union(c))
			for i := uint32(0); i < 5; i++ {
				got, e := a.Get(i)
				heapMust(t, e)
				if got != ((x|y)&(1<<i) != 0) {
					t.Fatal(x, y, i, "or")
				}
			}
			heapMust(t, a.Intersect(c))
			for i := uint32(0); i < 5; i++ {
				got, e := a.Get(i)
				heapMust(t, e)
				if got != (y&(1<<i) != 0) {
					t.Fatal(x, y, i, "and")
				}
			}
		}
	}
	heapMust(t, a.Reset())
	heapMust(t, a.Resize(111, false))
	want := make([]bool, 111)
	for _, i := range []uint32{105, 99, 98, 90, 80, 75, 33, 32, 31, 30, 10} {
		heapMust(t, a.SetBit(i))
		want[i] = true
	}
	for _, k := range []uint32{16, 1, 32, 42, 16, 63} {
		heapMust(t, a.ShiftRight(k))
		want = append(make([]bool, k), want...)
		for i, v := range want {
			got, e := a.Get(uint32(i))
			heapMust(t, e)
			if got != v {
				t.Fatal(k, i)
			}
		}
	}
	var out bytes.Buffer
	heapMust(t, a.Display(&out))
	if out.Len() != len(want) {
		t.Fatal(out.Len())
	}
	for i, v := range want {
		if (out.Bytes()[len(want)-1-i] == '1') != v {
			t.Fatal(i, "display")
		}
	}
	heapMust(t, a.Close())
	heapMust(t, c.Close())
	heapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
func TestBitVectorRuntimePushPublishesBitsBeforeAllocation(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64})
	heapMust(t, e)
	v, e := NewRuntimeBitVector(r, b)
	heapMust(t, e)
	b.FailAfter(0)
	e = v.Push(true)
	intHeapActualRefusal(t, r, e)
	if v.numBits != 1 || v.capacity != 0 || v.block != (memory.Block{}) {
		t.Fatal(v.numBits, v.capacity, v.block)
	}
	if b.Used() != r.Stats().TotalBytes+v.ownerBytes {
		t.Fatal("named immediate graph", b.Used(), r.Stats())
	}
	b.FailAfter(-1)
	heapMust(t, v.Close())
	heapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
func TestFastResetBitVectorRuntimeSourceListBeforeBase(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64})
	heapMust(t, e)
	f, e := NewRuntimeFastResetBitVector(r, b)
	heapMust(t, e)
	for i := uint32(0); i < 65; i++ {
		heapMust(t, f.Push(i%3 == 0))
	}
	heapMust(t, f.SetBool(3, true))
	heapMust(t, f.Reset())
	if f.base.numBits != 65 {
		t.Fatal("reset must retain bit count")
	}
	for i := uint32(0); i < 65; i++ {
		v, e := f.base.Get(i)
		heapMust(t, e)
		if v {
			t.Fatal(i)
		}
	}
	heapMust(t, f.Close())
	heapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestBitVectorRuntimeUnequalLengthsIndependentWordBoundaries(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b})
	heapMust(t, e)
	a, e := NewRuntimeBitVectorReserved(r, b, 65)
	heapMust(t, e)
	c, e := NewRuntimeBitVectorReserved(r, b, 65)
	heapMust(t, e)
	sizes := []uint32{0, 1, 31, 32, 33, 64, 65}
	for _, n := range sizes {
		for _, m := range sizes {
			heapMust(t, a.Reset())
			heapMust(t, c.Reset())
			heapMust(t, a.Resize(n, false))
			heapMust(t, c.Resize(m, false))
			for i := uint32(0); i < n; i++ {
				heapMust(t, a.SetBool(i, i%3 == 0))
			}
			for i := uint32(0); i < m; i++ {
				heapMust(t, c.SetBool(i, i%5 == 0))
			}
			heapMust(t, a.Intersect(c))
			for i := uint32(0); i < n; i++ {
				got, e := a.Get(i)
				heapMust(t, e)
				want := i < m && i%3 == 0 && i%5 == 0
				if got != want {
					t.Fatal(n, m, i, "and")
				}
			}
			heapMust(t, a.Reset())
			heapMust(t, a.Resize(n, false))
			for i := uint32(0); i < n; i++ {
				heapMust(t, a.SetBool(i, i%3 == 0))
			}
			heapMust(t, a.Union(c))
			end := n
			if m > end {
				end = m
			}
			if a.numBits != end {
				t.Fatal(n, m, a.numBits)
			}
			for i := uint32(0); i < end; i++ {
				got, e := a.Get(i)
				heapMust(t, e)
				want := i < n && i%3 == 0 || i < m && i%5 == 0
				if got != want {
					t.Fatal(n, m, i, "or")
				}
			}
		}
	}
	bitVectorRuntimeAudit(t, r, a, c)
	heapMust(t, a.Close())
	heapMust(t, c.Close())
	heapMust(t, r.Close())
}
