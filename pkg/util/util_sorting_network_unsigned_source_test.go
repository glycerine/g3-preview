// Source: src/util/sorting_network.h:28-115; src/test/sorting_network.cpp:41-131,422-449.
// Source author: Nikolaj Bjorner (2013-11-07).
// Licensed under the MIT license; see LICENSE.txt.
package util

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func sortingUnsignedFixture(t *testing.T, values []uint32) (*arena.Budget, *Runtime, *RuntimeUnsignedSortingNetwork, *RuntimeUnsignedVector, *RuntimeUnsignedVector) {
	t.Helper()
	b := arena.NewBudget(1 << 25)
	r, e := NewRuntime(RuntimeConfig{Budget: b})
	paramsMust(t, e)
	sn, e := NewRuntimeUnsignedSortingNetwork(r, b, nil)
	paramsMust(t, e)
	in, e := NewRuntimeUnsignedVector(r, b)
	paramsMust(t, e)
	out, e := NewRuntimeUnsignedVector(r, b)
	paramsMust(t, e)
	for _, v := range values {
		paramsMust(t, in.Push(v))
	}
	return b, r, sn, in, out
}
func sortingUnsignedClose(t *testing.T, b *arena.Budget, r *Runtime, sn *RuntimeUnsignedSortingNetwork, in, out *RuntimeUnsignedVector) {
	t.Helper()
	paramsMust(t, sn.Close())
	paramsMust(t, out.Close())
	paramsMust(t, in.Close())
	paramsMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
func sortingUnsignedOracle(t *testing.T, out *RuntimeUnsignedVector, values []uint32) {
	t.Helper()
	size, e := out.Size()
	paramsMust(t, e)
	want := len(values)
	if want > 1 {
		n := 1
		for n < want {
			n *= 2
		}
		want = n
	}
	if size != uint32(want) {
		t.Fatal("literal zero padding", size, want)
	}
	counts := map[uint32]int{}
	for _, v := range values {
		counts[v]++
	}
	counts[0] += want - len(values)
	var prev uint32
	for i := uint32(0); i < size; i++ {
		v, e := out.At(i)
		paramsMust(t, e)
		if i > 0 && prev > v {
			t.Fatal("source adjacent ENSURE", i, prev, v)
		}
		counts[v]--
		prev = v
	}
	for v, c := range counts {
		if c != 0 {
			t.Fatal("independent multiset", v, c)
		}
	}
}
func TestUnsignedSortingNetworkSourceRegisteredUnsignedCampaigns(t *testing.T) {
	checks := uint32(0)
	for _, values := range [][]uint32{{0, 1, 0, 1, 1, 0}, {0, 1, 2, 1, 1, 3}} {
		b, r, sn, in, out := sortingUnsignedFixture(t, values)
		paramsMust(t, sn.Sort(in, out))
		sortingUnsignedOracle(t, out, values)
		n, e := out.Size()
		paramsMust(t, e)
		checks += n - 1
		sortingUnsignedClose(t, b, r, sn, in, out)
	}
	for _, n := range []uint32{5, 8} {
		for bits := uint32(0); bits < 1<<n; bits++ {
			values := make([]uint32, n)
			for i := range values {
				values[i] = bits >> uint(i) & 1
			}
			b, r, sn, in, out := sortingUnsignedFixture(t, values)
			paramsMust(t, sn.Sort(in, out))
			sortingUnsignedOracle(t, out, values)
			n, e := out.Size()
			paramsMust(t, e)
			checks += n - 1
			sortingUnsignedClose(t, b, r, sn, in, out)
		}
	}
	if checks != 2030 {
		t.Fatal("active unsigned source ENSURE instances", checks)
	}
	t.Log("active unsigned source ENSURE instances", checks)
}
func TestUnsignedSortingNetworkIndependentFiniteUnsignedOracle(t *testing.T) {
	for n := 0; n <= 6; n++ {
		limit := 1
		for i := 0; i < n; i++ {
			limit *= 3
		}
		for code := 0; code < limit; code++ {
			values := make([]uint32, n)
			x := code
			for i := range values {
				values[i] = uint32(x % 3)
				x /= 3
			}
			b, r, sn, in, out := sortingUnsignedFixture(t, values)
			paramsMust(t, sn.Sort(in, out))
			sortingUnsignedOracle(t, out, values)
			sortingUnsignedClose(t, b, r, sn, in, out)
		}
	}
}

func TestUnsignedSortingNetworkSourceRepeatedStorageAndSelfAlias(t *testing.T) {
	values := []uint32{9, 0, 7, 1, 3, 2}
	b, r, sn, in, out := sortingUnsignedFixture(t, values)
	paramsMust(t, sn.Sort(in, out))
	sortingUnsignedOracle(t, out, values)
	firstA, firstB := sn.currentv.Column.Raw.Block, sn.nextv.Column.Raw.Block
	for i, v := range []uint32{4, 3, 2, 1, 0, 5} {
		paramsMust(t, in.Set(uint32(i), v))
	}
	paramsMust(t, sn.Sort(in, out))
	sortingUnsignedOracle(t, out, []uint32{4, 3, 2, 1, 0, 5})
	for _, v := range []*RuntimeUnsignedVector{&sn.currentv, &sn.nextv} {
		n, e := v.Size()
		paramsMust(t, e)
		if n != 16 {
			t.Fatal("source appends indices without clearing", n)
		}
	}
	if sn.currentv.Column.Raw.Block == firstA || sn.nextv.Column.Raw.Block == firstB {
		t.Fatal("actual dynamic second growth expected")
	}
	current, next := sn.current, sn.next
	paramsMust(t, sn.Sort(in, in))
	n, e := in.Size()
	paramsMust(t, e)
	if n != 0 || sn.current != current || sn.next != next {
		t.Fatal("reset alias erases input before append/early return", n)
	}
	for _, v := range []*RuntimeUnsignedVector{&sn.currentv, &sn.nextv} {
		n, e := v.Size()
		paramsMust(t, e)
		if n != 16 {
			t.Fatal("self alias did not read/reset network storage", n)
		}
	}
	sortingUnsignedClose(t, b, r, sn, in, out)
}

func TestUnsignedSortingNetworkFullUnsignedBoundaryOracle(t *testing.T) {
	for _, values := range [][]uint32{{^uint32(0), 0, 1 << 31, ^uint32(0), 7}, {1 << 31, 1<<31 - 1, 0, 9, 1}} {
		b, r, sn, in, out := sortingUnsignedFixture(t, values)
		paramsMust(t, sn.Sort(in, out))
		sortingUnsignedOracle(t, out, values)
		sortingUnsignedClose(t, b, r, sn, in, out)
	}
}
