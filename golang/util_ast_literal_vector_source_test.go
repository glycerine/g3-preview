// Source: src/util/vector.h:43-125,215-240,350-365,389-433,449-455,531-546.
// Source: src/test/sorting_network.cpp:150-181; src/util/sorting_network.h:120-1046.
// Source author: Leonardo de Moura; Nikolaj Bjorner.
// Licensed under the MIT license; see LICENSE.txt.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestASTLiteralVectorActualPointerPayloadAndNonowningReset(t *testing.T) {
	m := astTestManager(t)
	sort, e := m.MkBoolSort()
	astTestMust(t, e)
	x, e := m.MkFreshConst("x", sort)
	astTestMust(t, e)
	astTestPin(t, m, x)
	y, e := m.MkFreshConst("y", sort)
	astTestMust(t, e)
	astTestPin(t, m, y)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: m.budget})
	astTestMust(t, e)
	v, e := NewRuntimeASTLiteralVector(r, m.budget, nil)
	astTestMust(t, e)
	before, e := m.GetRefCount(x)
	astTestMust(t, e)
	astTestMust(t, v.Push(x))
	astTestMust(t, v.Push(y))
	capacity, e := v.Capacity()
	astTestMust(t, e)
	if capacity != 2 {
		t.Fatal("source first capacity", capacity)
	}
	size, e := v.Size()
	astTestMust(t, e)
	if size != 2 {
		t.Fatal("source size", size)
	}
	span, e := v.CPtr()
	astTestMust(t, e)
	astTestMust(t, v.Set(0, y))
	got, e := span.At(0)
	astTestMust(t, e)
	if got != y {
		t.Fatal("same-cap mutation", got)
	}
	astTestMust(t, v.Reset())
	got, e = span.At(1)
	astTestMust(t, e)
	if got != y {
		t.Fatal("source raw spare remains initialized", got)
	}
	after, e := m.GetRefCount(x)
	astTestMust(t, e)
	if after != before {
		t.Fatal("ptr_vector changed AST references", before, after)
	}
	astTestMust(t, v.Close())
	if _, e = span.At(0); e == nil {
		t.Fatal("destroyed actual captured backing")
	}
	astTestMust(t, r.Close())
}

func astLiteralNamedGraph(t *testing.T, b *arena.Budget, r *memory.Runtime, external uint64, owners []*RuntimeASTLiteralVector, anchors ...memory.Block) {
	t.Helper()
	roots := map[memory.Block]bool{}
	cold := external
	add := func(h memory.Block) {
		if h == (memory.Block{}) {
			return
		}
		if roots[h] {
			t.Fatal("duplicate physical root", h)
		}
		roots[h] = true
		_, e := r.Size(h)
		astTestMust(t, e)
	}
	for _, h := range anchors {
		add(h)
	}
	for _, v := range owners {
		cold += v.header
		if !v.raw.pendingReplaces && !v.assignmentDestroyed {
			add(v.raw.block)
		} else {
			if _, e := r.Size(v.raw.block); e == nil {
				t.Fatal("consumed realloc pointer still physical")
			}
		}
		add(v.raw.pending)
		for n := v.growthAcquisitions; n != nil; n = n.next {
			if n.owner != v || n.runtime != r || n.invocation == 0 || n.invocation > v.growthSerial {
				t.Fatal("actual archive authority")
			}
			cold += n.header
			add(n.block)
		}
		for n := v.copyAcquisitions; n != nil; n = n.next {
			if n.owner != v || n.runtime != r || n.invocation == 0 || n.invocation > v.copySerial {
				t.Fatal("exact copy invocation")
			}
			cold += n.header
			add(n.block)
		}
	}
	stats := r.Stats()
	if stats.LiveBlocks != uint32(len(roots)) || b.Used() != stats.TotalBytes+cold {
		t.Fatal("named literal raw/cold graph", stats, len(roots), b.Used(), cold)
	}
}
func TestASTLiteralVectorEveryActualResizeReservation(t *testing.T) {
	for _, bits := range []uint8{32, 64} {
		reached := int64(-1)
		for ordinal := int64(0); ordinal < 32; ordinal++ {
			m := astTestManager(t)
			sort, e := m.MkBoolSort()
			astTestMust(t, e)
			h, e := m.MkFreshConst("x", sort)
			astTestMust(t, e)
			astTestPin(t, m, h)
			r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: m.budget, SizeTBits: bits})
			astTestMust(t, e)
			v, e := NewRuntimeASTLiteralVector(r, m.budget, nil)
			astTestMust(t, e)
			// The caller AST graph is unchanged by nonowning literal storage.
			external := m.budget.Used() - r.Stats().TotalBytes - v.header
			refs, e := m.GetRefCount(h)
			astTestMust(t, e)
			m.budget.FailAfter(ordinal)
			cause := v.Resize(33, h)
			astLiteralNamedGraph(t, m.budget, r, external, []*RuntimeASTLiteralVector{v})
			size, e := v.Size()
			astTestMust(t, e)
			now, e := m.GetRefCount(h)
			astTestMust(t, e)
			if now != refs {
				t.Fatal("ptr resize owns AST", ordinal)
			}
			if cause != nil && size != 0 {
				t.Fatal("reserve all capacities before SIZE", ordinal, size, cause)
			}
			if cause == nil {
				reached = ordinal
				if size != 33 {
					t.Fatal(size)
				}
				for i := uint32(0); i < 33; i++ {
					got, e := v.At(i)
					astTestMust(t, e)
					if got != h {
						t.Fatal("ascending placement", i)
					}
				}
			}
			m.budget.FailAfter(-1)
			astTestMust(t, v.Close())
			astTestMust(t, r.Close())
			if reached >= 0 {
				t.Log("actual literal resize first success", bits, reached)
				break
			}
		}
		if reached < 0 {
			t.Fatal("no actual complete allocation horizon", bits)
		}
	}
}
func TestASTLiteralVectorRealNullRetryAndSourceDestructorRoots(t *testing.T) {
	m := astTestManager(t)
	sort, e := m.MkBoolSort()
	astTestMust(t, e)
	h, e := m.MkFreshConst("x", sort)
	astTestMust(t, e)
	astTestPin(t, m, h)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: m.budget, ThreadLocal: true})
	astTestMust(t, e)
	v, e := NewRuntimeASTLiteralVector(r, m.budget, nil)
	astTestMust(t, e)
	external := m.budget.Used() - r.Stats().TotalBytes - v.header
	anchor := unsignedGrowthPrime(t, r)
	cause := v.Push(h)
	if cause == nil || v.raw.block != (memory.Block{}) || v.raw.pending == (memory.Block{}) {
		t.Fatal("real lower returned NULL acquisition", cause, v.raw)
	}
	old := v.raw.pending
	astLiteralNamedGraph(t, m.budget, r, external, []*RuntimeASTLiteralVector{v}, anchor)
	astTestMust(t, r.SetMaxSize(0))
	astTestMust(t, v.Push(h))
	got, e := v.At(0)
	astTestMust(t, e)
	if got != h {
		t.Fatal("retry source pointer publication")
	}
	astLiteralNamedGraph(t, m.budget, r, external, []*RuntimeASTLiteralVector{v}, anchor)
	astTestMust(t, v.SourceFinalize())
	_, e = r.Size(old)
	astTestMust(t, e)
	astLiteralNamedGraph(t, m.budget, r, external, []*RuntimeASTLiteralVector{v}, anchor)
	astTestMust(t, v.Close())
	astTestMust(t, r.Deallocate(&anchor))
	astTestMust(t, r.Close())
}
