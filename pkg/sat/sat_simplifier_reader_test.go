// Source: src/sat/sat_simplifier.cpp:451–458; src/sat/sat_clause.cpp:105–125.
package sat

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestSATSimplifierActualSeparateDummyAllocator(t *testing.T) {
	k, a, _, b := satSimplifierKernelFixture(t, 3)
	d, e := NewSATTmpClause(b)
	satOK(t, e)
	t.Cleanup(func() { satOK(t, d.Close()) })
	satOK(t, d.SetBinary(0, 2, true))
	satOK(t, k.bindDummy(d))
	target := satMk(t, a, []Literal{0, 2, 4}, false)
	out := DefaultLiteral()
	yes, e := k.subsumes1(d.Get(), target, &out)
	if e != nil || !yes || out != DefaultLiteral() {
		t.Fatal("actual separate dummy reader required", yes, out, e)
	}
	// Retain source tmp_clause cached approximation on reuse; no cloned normal
	// clause or foreign-allocator registry lookup supplies the binary input.
	cached, e := d.owner.Approx(d.Get())
	satOK(t, e)
	satOK(t, d.SetBinary(0, 4, false))
	now, e := d.owner.Approx(d.Get())
	satOK(t, e)
	if now != cached {
		t.Fatal("source reuse retains cached approx", cached, now)
	}
	satOK(t, d.SetBinary(0, 3, false))
	out = DefaultLiteral()
	yes, e = k.subsumes1(d.Get(), target, &out)
	satOK(t, e)
	if !yes || out != 2 {
		t.Fatal("actual complementary dummy literal", yes, out)
	}
	borrowed := d.Get()
	satOK(t, d.Close())
	_, e = k.subsumes0(borrowed, target)
	if !errors.Is(e, arena.ErrClosed) {
		t.Fatal("borrowed dummy owner lifetime at read", e)
	}
}
