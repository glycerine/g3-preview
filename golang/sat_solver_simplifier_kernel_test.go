// Copyright (c) 2011 Microsoft Corporation. Licensed under the MIT license.
package g3

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/golang/arena"
)

func satSimplifierKernelFixture(t *testing.T, vars uint32) (*satSimplifierKernel, *satClauseAllocator, *satClauseUseLists, *arena.Budget) {
	t.Helper()
	a, u, b := satOuterUseFixture(t, vars)
	k, e := newSATSimplifierKernel(a, u, b)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, k.Close()) })
	satOK(t, k.InitVisited(vars))
	return k, a, u, b
}

func TestSATSimplifierVisitedSourceStorageAndPrefixes(t *testing.T) {
	k, _, _, b := satSimplifierKernelFixture(t, 0)
	if k.Stats().OwnerBytes != uint64(reflect.TypeOf(satSimplifierKernel{}).Size()) {
		t.Fatal(k.Stats())
	}
	before := b.Used()
	b.FailAfter(1)
	if e := k.InitVisited(4); !errors.Is(e, arena.ErrInjectedFailure) {
		t.Fatal(e)
	}
	if k.length != 0 || k.capacity != 2 || k.Stats().BackingBytes != 2 || b.Used() != before+2 {
		t.Fatal(k.Stats(), b.Used(), before)
	}
	b.FailAfter(-1)
	satOK(t, k.InitVisited(4))
	if k.length != 8 || k.capacity != 8 {
		t.Fatal(k.Stats())
	}
	satOK(t, k.markVisited(2))
	satOK(t, k.InitVisited(2))
	if marked, e := k.isMarked(2); e != nil || marked {
		t.Fatal(marked, e)
	}
	// Unsigned source 2*num_vars wraps before resize, leaving retained storage.
	satOK(t, k.InitVisited(1<<31))
	if k.length != 0 || k.capacity != 8 {
		t.Fatal(k.Stats())
	}
	copyOwner := *k
	if e := copyOwner.Close(); e != ErrSATOwner {
		t.Fatal(e)
	}
	var zero satSimplifierKernel
	if e := zero.Close(); e != ErrSATOwner {
		t.Fatal(e)
	}
	satOK(t, k.Finalize())
	if k.Stats().BackingBytes != 0 || b.Used() != before {
		t.Fatal(k.Stats(), b.Used())
	}
}

func TestSATSimplifierVisitedRawRetainedAndMarkFailure(t *testing.T) {
	k, a, _, _ := satSimplifierKernelFixture(t, 3)
	satOK(t, k.InitVisited(1))
	e := k.markVisited(4)
	if satStorageSourceDebug {
		if e != ErrSATAssertion {
			t.Fatal(e)
		}
	} else {
		satOK(t, e)
	}
	c := satMk(t, a, satLs(0, 40, 2), false)
	if e := k.markAllBut(c, satDefaultLiteral()); e == nil {
		t.Fatal("expected actual second mark bounds")
	}
	if yes, e := k.isMarked(0); e != nil || !yes {
		t.Fatal("completed mark prefix", yes, e)
	}
	satOK(t, k.InitVisited(3))
	c = satMk(t, a, satLs(0, 2, 4), false)
	satOK(t, k.markAllBut(c, 2))
	for _, l := range []satLiteral{0, 2, 4} {
		yes, e := k.isMarked(l)
		if e != nil || yes != (l != 2) {
			t.Fatal(l, yes, e)
		}
	}
	satOK(t, k.unmarkAll(c))
}

func TestSATSimplifierVisitedScalarExtent(t *testing.T) {
	for _, old := range []uint32{0, 2, 3, 5, 8} {
		n, _, e := satSimplifierVisitedCapacity(old, ^uint64(0))
		satOK(t, e)
		want := uint32(2)
		if old != 0 {
			want = (3*old + 1) >> 1
		}
		if n != want {
			t.Fatal(old, n, want)
		}
	}
	if _, _, e := satSimplifierVisitedCapacity(8, 11); e != arena.ErrBounds {
		t.Fatal(e)
	}
	if _, _, e := satSimplifierVisitedCapacity(^uint32(0), ^uint64(0)); e != arena.ErrBounds {
		t.Fatal(e)
	}
}
