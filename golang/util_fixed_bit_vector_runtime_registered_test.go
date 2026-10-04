// Source: src/util/fixed_bit_vector.h; src/util/fixed_bit_vector.cpp; src/test/fixed_bit_vector.cpp.
// Licensed under the MIT license. Source authors: Nikolaj Bjorner, 2014-9-15;
// Leonardo de Moura, 2006-10-03.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func fixedRuntimeFixture(t *testing.T, n uint32) (*RuntimeFixedBitVectorManager, *memory.Runtime, *arena.Budget) {
	t.Helper()
	b := arena.NewBudget(1 << 24)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 64})
	heapMust(t, e)
	m, e := NewRuntimeFixedBitVectorManager(n, memory.SmallAllocatorConfig{Runtime: r, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
	heapMust(t, e)
	return m, r, b
}
func fixedRuntimeFinish(t *testing.T, m *RuntimeFixedBitVectorManager, r *memory.Runtime, b *arena.Budget) {
	t.Helper()
	heapMust(t, m.Close())
	heapMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal("unreleased actual ownership", b.Used())
	}
}
func TestFixedBitVectorRuntimeRegisteredAssertions(t *testing.T) {
	assertions := 0
	ensure := func(v bool) {
		t.Helper()
		assertions++
		if !v {
			t.Fatal("source ENSURE", assertions)
		}
	}
	{
		m, r, b := fixedRuntimeFixture(t, 30)
		v, e := m.Allocate0()
		heapMust(t, e)
		heapMust(t, m.SetBool(v, 0, true))
		heapMust(t, m.SetBool(v, 1, false))
		heapMust(t, m.SetBool(v, 2, true))
		for _, p := range []struct {
			i    uint32
			want bool
		}{{0, true}, {1, false}, {2, true}, {3, false}, {29, false}} {
			got, e := m.Get(v, p.i)
			heapMust(t, e)
			ensure(got == p.want)
		}
		heapMust(t, m.Deallocate(v))
		fixedRuntimeFinish(t, m, r, b)
	}
	{
		m, r, b := fixedRuntimeFixture(t, 10)
		a, e := m.Allocate0()
		heapMust(t, e)
		c, e := m.Allocate0()
		heapMust(t, e)
		heapMust(t, m.SetBit(a, 4))
		for _, i := range []uint32{8, 3, 2, 1} {
			heapMust(t, m.SetBit(c, i))
		}
		heapMust(t, m.Union(a, c))
		eq, e := m.Equal(a, c)
		heapMust(t, e)
		ensure(!eq)
		heapMust(t, m.Unset(a, 4))
		eq, e = m.Equal(a, c)
		heapMust(t, e)
		ensure(eq)
		heapMust(t, m.Unset(a, 3))
		eq, e = m.Equal(a, c)
		heapMust(t, e)
		ensure(!eq)
		heapMust(t, m.Deallocate(a))
		heapMust(t, m.Deallocate(c))
		fixedRuntimeFinish(t, m, r, b)
	}
	for _, n := range []uint32{15, 16, 17, 31, 32, 33, 63, 64, 65} {
		m, r, b := fixedRuntimeFixture(t, n)
		a, e := m.Allocate0()
		heapMust(t, e)
		c, e := m.Allocate0()
		heapMust(t, e)
		d, e := m.Allocate0()
		heapMust(t, e)
		heapMust(t, m.SetBool(a, 3, true))
		eq, e := m.Equal(a, c)
		heapMust(t, e)
		ensure(!eq)
		eq, e = m.Equal(c, d)
		heapMust(t, e)
		ensure(eq)
		heapMust(t, m.SetBool(d, 3, true))
		eq, e = m.Equal(a, d)
		heapMust(t, e)
		ensure(eq)
		heapMust(t, m.SetBool(c, n-1, true))
		heapMust(t, m.SetBit(d, n-1))
		heapMust(t, m.Unset(d, 3))
		eq, e = m.Equal(c, d)
		heapMust(t, e)
		ensure(eq)
		heapMust(t, m.Fill0(a))
		heapMust(t, m.Neg(a))
		heapMust(t, m.Fill1(c))
		eq, e = m.Equal(a, c)
		heapMust(t, e)
		ensure(eq)
		heapMust(t, m.Fill0(a))
		for i := uint32(0); i < n; i++ {
			heapMust(t, m.SetBool(a, i, true))
		}
		eq, e = m.Equal(a, c)
		heapMust(t, e)
		ensure(eq)
		for _, v := range []RuntimeFixedBitVector{a, c, d} {
			heapMust(t, m.Deallocate(v))
		}
		fixedRuntimeFinish(t, m, r, b)
	}
	if assertions != 62 {
		t.Fatal("registered dynamic assertion count", assertions)
	}
}
