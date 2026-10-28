package polynomial

// Source: src/util/ref_buffer.h:38-99; src/util/buffer.h:39-46,146-151;
// src/math/polynomial/polynomial.cpp:5704.

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

// Each cycle crosses the old inline and heap growth frontiers. Native storage
// cannot acquire Runtime blocks, and every nonzero row owns one actual INC.
func TestPolynomialPSCBufferNativeGrowthAndReuse(t *testing.T) {
	m := polynomialCellFixture(t)
	p := polynomialQueryTerms(t, m, []int32{7}, [][]MonomialPower{{{0, 1}}})
	b := &m.pscBuffer
	choices := []Polynomial{p, m.unit, Polynomial{}, m.zero}
	for _, n := range []int{4097, 0, 33, 1, 129} {
		serviceMust(t, b.construct(m))
		before, used := m.memory.Stats(), m.integer.Budget().Used()
		refs := make([]uint32, len(choices))
		for i, p := range choices {
			if p != (Polynomial{}) {
				var e error
				refs[i], e = m.readWord(p, 0)
				serviceMust(t, e)
			}
		}
		m.integer.Budget().FailAfter(0)
		for i := 0; i < n; i++ {
			serviceMust(t, b.push(m, choices[i%len(choices)]))
		}
		if len(b.values) != n || m.memory.Stats() != before || m.integer.Budget().Used() != used {
			t.Fatal("native growth changed Runtime storage", n, len(b.values), before, m.memory.Stats())
		}
		for i := 0; i < n; i++ {
			got, e := b.get(m, uint32(i))
			serviceMust(t, e)
			if got != choices[i%len(choices)] {
				t.Fatal("native row", i, got)
			}
		}
		for i, p := range choices {
			if p == (Polynomial{}) {
				continue
			}
			added := uint32((n + len(choices) - 1 - i) / len(choices))
			got, e := m.readWord(p, 0)
			serviceMust(t, e)
			if got != refs[i]+added {
				t.Fatal("one INC per completed row", i, got, refs[i], added)
			}
		}
		if _, e := b.get(m, uint32(n)); !errors.Is(e, arena.ErrBounds) && !errors.Is(e, ErrPolynomialAssertion) {
			t.Fatal("native bounds", e)
		}
		// A rejected actual reference cannot publish a row or consume earlier rows.
		invalid := Polynomial{owner: m.origin, slot: ^uint32(0), generation: 1}
		if e := b.push(m, invalid); e == nil || len(b.values) != n {
			t.Fatal("invalid reference published", e)
		}
		polynomialPSCPhysical(t, m, nil, []Polynomial{p}, nil)
		serviceMust(t, b.finalize(m, false))
		if !b.finalized || b.values != nil || m.memory.Stats() != before || m.integer.Budget().Used() != used {
			t.Fatal("native finalization changed Runtime storage")
		}
		for i, p := range choices {
			if p == (Polynomial{}) {
				continue
			}
			got, e := m.readWord(p, 0)
			serviceMust(t, e)
			if got != refs[i] {
				t.Fatal("native reference cleanup", i, got, refs[i])
			}
		}
		serviceMust(t, b.finalize(m, true))
		m.integer.Budget().FailAfter(-1)
	}
}

func TestPolynomialPSCBufferPendingLocalBlocksOwningCoreClose(t *testing.T) {
	m := polynomialCellFixture(t)
	p, e := m.MkVariable(0, 1)
	serviceMust(t, e)
	b := &m.pscBuffer
	serviceMust(t, b.construct(m))
	serviceMust(t, b.push(m, p))
	t.Cleanup(func() { serviceMust(t, b.finalize(m, true)) })
	// The only extra polynomial reference is the real completed H local.
	if e = m.CloseChecked(); !errors.Is(e, ErrPolynomialPending) {
		t.Fatal("owning close must identify actual pending local before caller guard", e)
	}
	serviceMust(t, b.finalize(m, false))
}

func TestPolynomialPSCBufferNativeStoppedReferenceCleanup(t *testing.T) {
	m := polynomialCellFixture(t)
	b := &m.pscBuffer
	serviceMust(t, b.construct(m))
	indices := map[Polynomial]int{}
	for i := 0; i < 129; i++ {
		p := polynomialICCPUnownedTerms(t, m, []int32{int32(101 + i)}, [][]MonomialPower{nil})
		indices[p] = i
		serviceMust(t, b.push(m, p))
	}
	var calls []int
	h, e := NewPolynomialDeletionHandler(m.integer.Budget(), func(_ *PolynomialDeletionHandler, _ *MultivariatePolynomialCore, p Polynomial) error {
		i, ok := indices[p]
		if !ok {
			t.Fatal("unexpected deleted polynomial", p)
		}
		calls = append(calls, i)
		if i == 17 {
			return arena.ErrBounds
		}
		return nil
	})
	serviceMust(t, e)
	serviceMust(t, m.AddDeletionHandler(h))
	t.Cleanup(func() {
		serviceMust(t, m.RemoveDeletionHandler(h))
		serviceMust(t, h.CloseChecked())
		serviceMust(t, b.finalize(m, true))
	})
	e = b.finalize(m, false)
	if !errors.Is(e, arena.ErrBounds) || b.cursor != 17 || !b.drainDec || b.finalized || len(calls) != 18 {
		t.Fatal("real destructor stop", e, b.cursor, b.drainDec, len(calls))
	}
	for i := 0; i < 17; i++ {
		if b.values[i] != (Polynomial{}) {
			t.Fatal("completed reference retained", i)
		}
	}
	polynomialNewtonPhysicalAudit(t, m, nil, nil, nil, nil, b.values, []*PolynomialDeletionHandler{h})
	if e = b.finalize(m, false); !errors.Is(e, arena.ErrBounds) || b.cursor != 17 || len(calls) != 18 {
		t.Fatal("source retry advanced stopped destructor", e, b.cursor, len(calls))
	}
	serviceMust(t, b.finalize(m, true))
	if len(calls) != 129 || !b.finalized || b.values != nil {
		t.Fatal("owning continuation", len(calls), b.finalized)
	}
	for i, got := range calls {
		if got != i {
			t.Fatal("ascending reference cleanup or duplicate callback", i, got)
		}
	}
	polynomialNewtonPhysicalAudit(t, m, nil, nil, nil, nil, b.values, []*PolynomialDeletionHandler{h})
}
