package polynomial

// Source: src/math/polynomial/polynomial.cpp:3185,6253-6298.

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestPolynomialFoundVariablesNativeGrowthRetainsBits(t *testing.T) {
	m := polynomialCellFixture(t)
	var v polynomialFoundVariables
	b := m.integer.Budget()
	b.FailAfter(0)
	t.Cleanup(func() { b.FailAfter(-1) })
	before, used := m.memory.Stats(), b.Used()
	old := uint32(0)
	for _, n := range []uint32{1, 3, 9, 129, 4097} {
		serviceMust(t, v.reserve(n))
		for i := uint32(0); i < n; i++ {
			got, e := v.at(i)
			serviceMust(t, e)
			if got != (i < old && i%7 == 0) {
				t.Fatal("retained old / zero new flag", n, i, got)
			}
			serviceMust(t, v.put(i, i%7 == 0))
		}
		old = n
	}
	serviceMust(t, v.reserve(2))
	n := uint32(len(v.values))
	if n != 4097 {
		t.Fatal("reserve shrank flags", n)
	}
	for i := uint32(0); i < n; i++ {
		got, e := v.at(i)
		serviceMust(t, e)
		if got != (i%7 == 0) {
			t.Fatal("independent flag", i, got)
		}
		serviceMust(t, v.put(i, false))
	}
	bounds := error(arena.ErrBounds)
	if polynomialSourceDebug {
		bounds = ErrPolynomialAssertion
	}
	if _, e := v.at(n); !errors.Is(e, bounds) {
		t.Fatal("native read bounds", e)
	}
	if e := v.put(n, true); !errors.Is(e, bounds) {
		t.Fatal("native write bounds", e)
	}
	serviceMust(t, v.sourceFinalize())
	if v.values != nil || !v.closed {
		t.Fatal("native scalar cleanup retained flags")
	}
	if e := v.reserve(3); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("closed growth", e)
	}
	if _, e := v.at(0); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("closed read", e)
	}
	serviceMust(t, v.close())
	if m.memory.Stats() != before || b.Used() != used {
		t.Fatal("native flags changed Runtime accounting")
	}
	polynomialPhysicalAudit(t, m)
}
