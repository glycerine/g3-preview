// Source: src/util/permutation.h:56-91; src/math/polynomial/polynomial.cpp:1334-1335.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func runtimePermutationFixture(t *testing.T, n uint64) (*arena.Budget, *Runtime, Block, RuntimeUint32View) {
	t.Helper()
	b := arena.NewBudget(1 << 20)
	r, e := NewRuntime(RuntimeConfig{Budget: b})
	if e != nil {
		t.Fatal(e)
	}
	h, e := r.Allocate(n)
	if e != nil {
		t.Fatal(e)
	}
	return b, r, h, BorrowRuntimeUint32(r, h, 0)
}
func runtimePermutationMust(t *testing.T, e error) {
	t.Helper()
	if e != nil {
		t.Fatal(e)
	}
}
func TestRuntimePermutationPointerCyclesIndependentFiniteOracle(t *testing.T) {
	b, r, h, data := runtimePermutationFixture(t, 48)
	p := data.Offset(6)
	for n := 0; n <= 6; n++ {
		perm := make([]uint32, n)
		for i := range perm {
			perm[i] = uint32(i)
		}
		var visit func(int)
		visit = func(k int) {
			if k < n {
				for i := k; i < n; i++ {
					perm[k], perm[i] = perm[i], perm[k]
					visit(k + 1)
					perm[k], perm[i] = perm[i], perm[k]
				}
				return
			}
			for _, core := range []bool{false, true} {
				for i := 0; i < n; i++ {
					runtimePermutationMust(t, data.Write(uint32(i), uint32(i)*701+9))
					runtimePermutationMust(t, p.Write(uint32(i), perm[i]))
				}
				state, used := r.State(), b.Used()
				b.FailAfter(0)
				var e error
				if core {
					e = ApplyPermutationCoreRuntimeUint32(uint32(n), data, p)
				} else {
					e = ApplyPermutationRuntimeUint32(uint32(n), data, p)
				}
				runtimePermutationMust(t, e)
				if r.State() != state || b.Used() != used {
					t.Fatal("cycle must allocate nothing")
				}
				b.FailAfter(-1)
				for i := 0; i < n; i++ {
					v, e := data.Read(uint32(i))
					runtimePermutationMust(t, e)
					if v != perm[i]*701+9 {
						t.Fatal(n, perm, i, v)
					}
					m, e := p.Read(uint32(i))
					runtimePermutationMust(t, e)
					want := perm[i]
					if core {
						want = ^want
					}
					if m != want {
						t.Fatal("literal marker", m, want)
					}
				}
			}
		}
		visit(0)
	}
	runtimePermutationMust(t, r.Deallocate(&h))
	runtimePermutationMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
func TestRuntimePermutationPointerMarkBeforeReachedReadAndAliasing(t *testing.T) {
	b, r, h, p := runtimePermutationFixture(t, 16)
	runtimePermutationMust(t, p.Write(0, 1))
	runtimePermutationMust(t, p.Write(1, 0))
	data := p.Offset(2)
	runtimePermutationMust(t, data.Write(0, 19))
	e := ApplyPermutationRuntimeUint32(2, data, p)
	if !errors.Is(e, ErrRuntimeUninitialized) {
		t.Fatal(e)
	}
	m, e := p.Read(0)
	runtimePermutationMust(t, e)
	if m != ^uint32(1) {
		t.Fatal("mark before failed swap", m)
	}
	v, e := data.Read(0)
	runtimePermutationMust(t, e)
	if v != 19 {
		t.Fatal("both reads before first swap store", v)
	}
	for i := uint32(0); i < 3; i++ {
		runtimePermutationMust(t, p.Write(i, i))
	}
	runtimePermutationMust(t, ApplyPermutationCoreRuntimeUint32(3, p, p))
	for i := uint32(0); i < 3; i++ {
		v, e := p.Read(i)
		runtimePermutationMust(t, e)
		if v != ^i {
			t.Fatal(v)
		}
	}
	runtimePermutationMust(t, r.Deallocate(&h))
	runtimePermutationMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
func TestRuntimePermutationPointerUnreadZeroSignedLoopAndLifetime(t *testing.T) {
	var nilView RuntimeUint32View
	runtimePermutationMust(t, ApplyPermutationRuntimeUint32(0, nilView, nilView))
	runtimePermutationMust(t, ApplyPermutationCoreRuntimeUint32(0x80000000, nilView, nilView))
	b, r, h, p := runtimePermutationFixture(t, 4)
	runtimePermutationMust(t, p.Write(0, 0))
	runtimePermutationMust(t, r.Deallocate(&h))
	if _, e := p.Read(0); !errors.Is(e, ErrRuntimeOwner) {
		t.Fatal("captured stale block", e)
	}
	runtimePermutationMust(t, r.Close())
	runtimePermutationMust(t, ApplyPermutationRuntimeUint32(0, p, p))
	if _, e := p.Read(0); e == nil {
		t.Fatal("reached provider lifetime")
	}
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestRuntimePermutationPointerReachedAssertionBoundsAndRestorationPrefix(t *testing.T) {
	b, r, h, p := runtimePermutationFixture(t, 8)
	runtimePermutationMust(t, p.Write(0, 1))
	runtimePermutationMust(t, p.Write(1, 0))
	e := ApplyPermutationRuntimeUint32(2, p.Offset(2), p)
	if !errors.Is(e, arena.ErrBounds) {
		t.Fatal("actual first data load extent", e)
	}
	x, e := p.Read(0)
	runtimePermutationMust(t, e)
	if x != ^uint32(1) {
		t.Fatal("mark survives guard", x)
	}
	// Signed core outer loop skips for this count; restoring wrapper still reads
	// unsigned indices and publishes its first inverse marker before next guard.
	runtimePermutationMust(t, p.Write(0, ^uint32(0)))
	runtimePermutationMust(t, p.Write(1, ^uint32(1)))
	e = ApplyPermutationRuntimeUint32(0x80000000, RuntimeUint32View{}, p)
	if !errors.Is(e, arena.ErrBounds) {
		t.Fatal("reached restoring read", e)
	}
	for i := uint32(0); i < 2; i++ {
		x, e := p.Read(i)
		runtimePermutationMust(t, e)
		if x != i {
			t.Fatal("literal restoration prefix", i, x)
		}
	}
	if sourceDebug {
		runtimePermutationMust(t, p.Write(0, 2))
		runtimePermutationMust(t, p.Write(1, 0))
		e = ApplyPermutationCoreRuntimeUint32(2, RuntimeUint32View{}, p)
		if !errors.Is(e, ErrPermutationAssertion) {
			t.Fatal("p_j assertion before data/marker", e)
		}
		x, e = p.Read(0)
		runtimePermutationMust(t, e)
		if x != 2 {
			t.Fatal(x)
		}
	}
	runtimePermutationMust(t, r.Deallocate(&h))
	runtimePermutationMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
