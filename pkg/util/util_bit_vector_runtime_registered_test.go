// Source: src/test/bit_vector.cpp:104-309; src/util/bit_vector.h; src/util/bit_vector.cpp.
package util

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

// tst_bit_vector returns before tst2 and the random tst1 driver. This translates
// the active OR/AND/EQ assertions; the independent shift output is elsewhere.
func TestBitVectorRuntimeRegisteredOrAndEquality(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := NewRuntime(RuntimeConfig{Budget: b})
	heapMust(t, e)
	a, e := NewRuntimeBitVector(r, b)
	heapMust(t, e)
	c, e := NewRuntimeBitVector(r, b)
	heapMust(t, e)
	start := func(n, m uint32) {
		heapMust(t, a.Reset())
		heapMust(t, c.Reset())
		heapMust(t, a.Resize(n, false))
		heapMust(t, c.Resize(m, false))
	}
	set := func(v *RuntimeBitVector, ix ...uint32) {
		for _, i := range ix {
			heapMust(t, v.SetBit(i))
		}
	}
	eq := func(want bool) {
		t.Helper()
		got, e := a.Equal(c)
		heapMust(t, e)
		if got != want {
			t.Fatal("source equality", got, want)
		}
	}
	bit := func(v *RuntimeBitVector, i uint32, want bool) {
		t.Helper()
		got, e := v.Get(i)
		heapMust(t, e)
		if got != want {
			t.Fatal(i, got, want)
		}
	}
	start(5, 10)
	set(a, 4)
	set(c, 8, 3, 2, 1)
	heapMust(t, a.Union(c))
	if a.numBits != 10 {
		t.Fatal(a.numBits)
	}
	eq(false)
	eq(false)
	heapMust(t, a.Unset(4))
	eq(true)
	heapMust(t, a.Unset(3))
	eq(false)
	start(2, 5)
	set(a, 0)
	set(c, 4)
	heapMust(t, a.Union(c))
	eq(false)
	set(c, 0)
	eq(true)
	start(10, 10)
	set(a, 5)
	set(c, 8, 3)
	heapMust(t, c.Resize(5, false))
	heapMust(t, a.Union(c))
	bit(a, 8, false)
	bit(a, 5, true)
	bit(a, 3, true)
	start(123, 126)
	set(c, 124)
	set(a, 122, 100, 80, 4)
	set(c, 100, 80, 4)
	eq(false)
	heapMust(t, c.Resize(123, false))
	eq(false)
	heapMust(t, a.Resize(120, false))
	heapMust(t, c.Resize(120, false))
	eq(true)
	heapMust(t, a.Unset(80))
	heapMust(t, a.Unset(100))
	eq(false)
	heapMust(t, a.Union(c))
	eq(true)
	start(5, 10)
	set(c, 8, 1)
	set(a, 4, 0)
	heapMust(t, a.Union(c))
	if a.numBits != 10 {
		t.Fatal(a.numBits)
	}
	for _, i := range []uint32{8, 4, 1, 0} {
		bit(a, i, true)
	}
	bit(a, 9, false)
	start(32, 32)
	set(a, 1, 5)
	set(c, 8, 0)
	heapMust(t, a.Union(c))
	for _, i := range []uint32{1, 5, 8, 0} {
		bit(a, i, true)
	}
	bit(a, 11, false)
	start(5, 3)
	set(c, 2)
	set(a, 2, 4)
	heapMust(t, a.Intersect(c))
	bit(a, 4, false)
	bit(a, 2, true)
	start(241, 128)
	set(a, 240, 232, 127, 128, 8)
	set(c, 127, 5)
	heapMust(t, a.Intersect(c))
	for _, i := range []uint32{240, 232, 128, 8, 5} {
		bit(a, i, false)
	}
	bit(a, 127, true)
	// tst_eq has a third vector and deliberately exercises the full last word.
	d, e := NewRuntimeBitVector(r, b)
	heapMust(t, e)
	heapMust(t, d.Resize(32, false))
	start(32, 32)
	set(a, 3)
	eq(false)
	eq(false)
	got, e := c.Equal(d)
	heapMust(t, e)
	if !got {
		t.Fatal("third equal")
	}
	set(d, 3)
	got, e = a.Equal(d)
	heapMust(t, e)
	if !got {
		t.Fatal("a==d")
	}
	got, e = a.Equal(d)
	heapMust(t, e)
	if !got {
		t.Fatal("not inequality")
	}
	set(c, 31)
	set(d, 31)
	heapMust(t, d.Unset(3))
	got, e = c.Equal(d)
	heapMust(t, e)
	if !got {
		t.Fatal("equal bit31")
	}
	got, e = c.Equal(d)
	heapMust(t, e)
	if !got {
		t.Fatal("not inequality bit31")
	}
	bitVectorRuntimeAudit(t, r, a, c, d)
	heapMust(t, a.Close())
	heapMust(t, c.Close())
	heapMust(t, d.Close())
	heapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
