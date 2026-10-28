package polynomial

// Source: src/math/polynomial/polynomial.cpp; src/math/polynomial/polynomial.h.

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func orderFixture(t *testing.T) *MonomialManager {
	t.Helper()
	b := arena.NewBudget(1 << 25)
	r, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: b})
	serviceMust(t, e)
	m, e := NewMonomialManager(MonomialManagerConfig{Budget: b, Memory: r, Profile: umemory.SmallAllocatorProfile{PointerBits: 64}})
	serviceMust(t, e)
	t.Cleanup(func() {
		serviceMust(t, m.CloseChecked())
		serviceMust(t, r.Close())
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	})
	for i := 0; i < 6; i++ {
		_, e = m.MkVar()
		serviceMust(t, e)
	}
	return m
}
func orderValue(t *testing.T, m *MonomialManager, d [6]uint32) Monomial {
	t.Helper()
	var p []MonomialPower
	for v, k := range d {
		if k != 0 {
			p = append(p, MonomialPower{uint32(v), k})
		}
	}
	a, e := m.MkMonomial(p)
	serviceMust(t, e)
	serviceMust(t, m.IncRef(a))
	t.Cleanup(func() { serviceMust(t, m.DecRef(a)) })
	return a
}

// All 21 dormant tst_lex calls, preserving four adapted ENSURE conditions each.
// The registered tst_polynomial returns before tst_lex; no active credit is added.
// Polynomial expression construction and ostream output are outside this adapter.
func TestMonomialOrderSourceLexAssertions(t *testing.T) {
	m := orderFixture(t)
	cases := []struct {
		a, b [6]uint32
		lex  int
		min  uint32
		lex2 int
	}{
		{[6]uint32{1, 1, 0, 0, 1}, [6]uint32{10, 3}, 1, 4, -1},
		{[6]uint32{1, 2, 0, 1, 1}, [6]uint32{1, 2, 0, 2, 1}, -1, 3, -1},
		{[6]uint32{2, 2, 0, 1, 1}, [6]uint32{1, 2, 0, 2, 1}, -1, 3, 1},
		{[6]uint32{1, 2, 0, 1, 1}, [6]uint32{1, 2, 0, 1, 1}, 0, 3, 0},
		{[6]uint32{1, 2, 0, 2, 1}, [6]uint32{1, 2, 0, 1, 1}, 1, 3, 1},
		{[6]uint32{0, 2, 0, 0, 1}, [6]uint32{1, 0, 1, 1, 1, 1}, -1, 1, -1},
		{[6]uint32{0, 2, 0, 1, 1}, [6]uint32{1, 1}, 1, 1, 1},
		{[6]uint32{0, 1, 0, 1, 1}, [6]uint32{0, 0, 1, 1, 1}, -1, 2, 1},
		{[6]uint32{0, 1, 0, 1, 1}, [6]uint32{0, 0, 1, 1, 1}, -1, 1, -1},
		{[6]uint32{0, 1, 0, 1, 1}, [6]uint32{1, 0, 1, 1, 1}, -1, 1, -1},
		{[6]uint32{0, 0, 0, 1}, [6]uint32{0, 0, 0, 0, 1}, -1, 1, -1},
		{[6]uint32{0, 0, 0, 1}, [6]uint32{0, 0, 0, 0, 1}, -1, 4, 1},
		{[6]uint32{0, 0, 1, 1}, [6]uint32{0, 0, 0, 0, 1}, -1, 1, -1},
		{[6]uint32{0, 0, 1, 1}, [6]uint32{0, 0, 0, 0, 1}, -1, 4, 1},
		{[6]uint32{0, 0, 0, 1}, [6]uint32{0, 0, 1, 0, 1}, -1, 1, -1},
		{[6]uint32{0, 0, 0, 1}, [6]uint32{0, 0, 1, 0, 1}, -1, 4, 1},
		{[6]uint32{0, 0, 0, 1}, [6]uint32{0, 0, 1, 0, 1}, -1, 4, 1},
		{[6]uint32{}, [6]uint32{0, 0, 0, 1}, -1, 1, -1},
		{[6]uint32{}, [6]uint32{0, 0, 0, 1}, -1, 3, -1},
		{[6]uint32{0, 0, 0, 1}, [6]uint32{}, 1, 3, 1},
		{[6]uint32{0, 0, 0, 0, 1, 1}, [6]uint32{0, 0, 0, 0, 3, 1}, -1, 4, -1},
	}
	n := 0
	for i, c := range cases {
		a, b := orderValue(t, m, c.a), orderValue(t, m, c.b)
		for j := 0; j < 4; j++ {
			var got, want int
			var e error
			switch j {
			case 0:
				got, e = m.Lex(a, b)
				want = c.lex
			case 1:
				got, e = m.LexWithMinimum(a, b, c.min)
				want = c.lex2
			case 2:
				got, e = m.Lex(b, a)
				want = -c.lex
			case 3:
				got, e = m.LexWithMinimum(b, a, c.min)
				want = -c.lex2
			}
			serviceMust(t, e)
			if got != want {
				t.Fatal(i, j, got, want)
			}
			n++
		}
	}
	if n != 84 {
		t.Fatal(n)
	}
}
func orderOracle(a, b [6]uint32, min int) int {
	for v := 5; v >= 0; v-- {
		if v == min {
			continue
		}
		if a[v] < b[v] {
			return -1
		}
		if a[v] > b[v] {
			return 1
		}
	}
	if min >= 0 {
		if a[min] < b[min] {
			return -1
		}
		if a[min] > b[min] {
			return 1
		}
	}
	return 0
}
func orderDegree(a [6]uint32) uint32 {
	var d uint32
	for _, k := range a {
		d += k
	}
	return d
}
func TestMonomialOrderBoundedModels(t *testing.T) {
	m := orderFixture(t)
	var ds [][6]uint32
	var values []Monomial
	for i := 0; i < 27; i++ {
		d := [6]uint32{uint32(i % 3), uint32(i / 3 % 3), uint32(i / 9)}
		ds = append(ds, d)
		values = append(values, orderValue(t, m, d))
	}
	for i, a := range values {
		for j, b := range values {
			lex := orderOracle(ds[i], ds[j], -1)
			got, e := m.Lex(a, b)
			serviceMust(t, e)
			if got != lex {
				t.Fatal(i, j, got, lex)
			}
			for min := 0; min < 3; min++ {
				got, e = m.LexWithMinimum(a, b, uint32(min))
				serviceMust(t, e)
				if got != orderOracle(ds[i], ds[j], min) {
					t.Fatal("minimum", i, j, min, got)
				}
			}
			d1, d2 := orderDegree(ds[i]), orderDegree(ds[j])
			g := lex
			if d1 < d2 {
				g = -1
			} else if d1 > d2 {
				g = 1
			}
			got, e = m.GradedLex(a, b)
			serviceMust(t, e)
			if got != g {
				t.Fatal("graded", i, j, got, g)
			}
			gr := g
			if d1 == d2 {
				gr = -lex
			}
			got, e = m.GradedReverseLex(a, b)
			serviceMust(t, e)
			if got != gr {
				t.Fatal("grev", i, j, got, gr)
			}
			if d1 == d2 {
				got, e = m.ReverseLex(a, b)
				serviceMust(t, e)
				if got != -lex {
					t.Fatal("reverse", i, j, got, -lex)
				}
			}
		}
	}
}
func TestMonomialOrderSourceRangesAndReadGuards(t *testing.T) {
	m := orderFixture(t)
	a := orderValue(t, m, [6]uint32{1, 0, 2})
	b := orderValue(t, m, [6]uint32{2, 0, 1})
	got, e := m.ReverseLexRange(a, 0, 2, b, 0, 2)
	serviceMust(t, e)
	if got != -1 {
		t.Fatal(got)
	}
	got, e = m.ReverseLexRange(Monomial{}, 0, 0, Monomial{}, 0, 0)
	serviceMust(t, e)
	if got != 0 {
		t.Fatal(got)
	}
	got, e = m.Lex(Monomial{}, Monomial{})
	serviceMust(t, e)
	if got != 0 {
		t.Fatal(got)
	}
	_, e = m.GradedLex(Monomial{}, Monomial{})
	if !errors.Is(e, ErrMonomialOwner) {
		t.Fatal(e)
	}
	got, e = m.ReverseLexRange(a, 2, 1, b, 0, 2)
	if polynomialSourceDebug {
		if !errors.Is(e, ErrMonomialAssertion) {
			t.Fatal(e)
		}
	} else {
		serviceMust(t, e)
		if got != 1 {
			t.Fatal(got)
		}
	}
	// Equal nonempty subranges hit the source assertion; release finishes at 0.
	got, e = m.ReverseLexRange(a, 0, 2, a, 0, 2)
	if polynomialSourceDebug {
		if !errors.Is(e, ErrMonomialAssertion) {
			t.Fatal(e)
		}
	} else {
		serviceMust(t, e)
		if got != 0 {
			t.Fatal(got)
		}
	}
	// A source scalar equality shortcut does not dereference an ended pointee.
	stale, e := m.MkVariable(5, 2)
	serviceMust(t, e)
	serviceMust(t, m.IncRef(stale))
	serviceMust(t, m.DecRef(stale))
	got, e = m.Lex(stale, stale)
	serviceMust(t, e)
	if got != 0 {
		t.Fatal(got)
	}
	_, e = m.Lex(stale, a)
	if !errors.Is(e, ErrMonomialOwner) {
		t.Fatal(e)
	}
	// No reads occur for an empty range even with a stale input.
	got, e = m.ReverseLexRange(stale, 0, 0, a, 0, 0)
	serviceMust(t, e)
	if got != 0 {
		t.Fatal(got)
	}

}
func TestMonomialOrderWarmNoAllocation(t *testing.T) {
	m := orderFixture(t)
	a := orderValue(t, m, [6]uint32{1, 0, 2})
	b := orderValue(t, m, [6]uint32{2, 0, 1})
	before := m.memory.Stats()
	count, e := m.memory.AllocationCount()
	serviceMust(t, e)
	allocations := testing.AllocsPerRun(100, func() {
		_, e = m.Lex(a, b)
		if e != nil {
			panic(e)
		}
		_, e = m.GradedLex(a, b)
		if e != nil {
			panic(e)
		}
		_, e = m.GradedReverseLex(a, b)
		if e != nil {
			panic(e)
		}
	})
	after, e := m.memory.AllocationCount()
	serviceMust(t, e)
	if allocations != 0 || before != m.memory.Stats() || count != after {
		t.Fatal(allocations, before, m.memory.Stats(), count, after)
	}
}

// Forged size headers expose the selected signed int conversion independently
// of real allocation size. Restore metadata before any destructor; this is a
// guarded source-undefined fixture, not an ordinary monomial construction.
func TestMonomialOrderSourceSignedSizeBoundary(t *testing.T) {
	m := orderFixture(t)
	a := orderValue(t, m, [6]uint32{1})
	b := orderValue(t, m, [6]uint32{0, 1})
	l, e := m.location(a)
	serviceMust(t, e)
	serviceMust(t, m.allocator.WriteUint32(l.raw, 12, 0xffffffff))
	got, e := m.Lex(a, b)
	serviceMust(t, m.allocator.WriteUint32(l.raw, 12, 1))
	serviceMust(t, e)
	if got != -1 {
		t.Fatal("uint32 to selected int32 conversion", got)
	}
	serviceMust(t, m.allocator.WriteUint32(l.raw, 12, 0x80000000))
	_, e = m.Lex(a, b)
	serviceMust(t, m.allocator.WriteUint32(l.raw, 12, 1))
	if !errors.Is(e, arena.ErrBounds) {
		t.Fatal("actual signed subtraction UB boundary", e)
	}
}

// These are real allocated partially initialized encodings, not canonical
// constructed monomials. Only the fields a source branch reaches are present.
func orderPartialValue(t *testing.T, m *MonomialManager, v uint32, degree *uint32) Monomial {
	t.Helper()
	slot, e := m.prepareLocation()
	serviceMust(t, e)
	obj, e := m.allocator.Allocate(28)
	serviceMust(t, e)
	l := &m.locations[slot-1]
	l.raw = obj
	l.requested = 28
	l.live = true
	l.sourceRaw, l.sourceOffset, e = m.allocator.RawLocation(obj)
	serviceMust(t, e)
	serviceMust(t, m.allocator.WriteUint32(obj, 12, 1))
	serviceMust(t, m.allocator.WriteUint32(obj, 20, v))
	if degree != nil {
		serviceMust(t, m.allocator.WriteUint32(obj, 24, *degree))
	}
	a := Monomial{m.origin, slot, l.generation}
	t.Cleanup(func() { serviceMust(t, m.CleanupValue(&a)) })
	return a
}
func TestMonomialOrderSourceDegreeShortCircuit(t *testing.T) {
	t.Run("unequal variables", func(t *testing.T) {
		m := orderFixture(t)
		a := orderPartialValue(t, m, 1, nil)
		b := orderPartialValue(t, m, 2, nil)
		got, e := m.Lex(a, b)
		serviceMust(t, e)
		if got != -1 {
			t.Fatal(got)
		}
		got, e = m.ReverseLex(a, b)
		serviceMust(t, e)
		if got != 1 {
			t.Fatal(got)
		}
		got, e = m.ReverseLexRange(a, 0, 1, b, 0, 1)
		serviceMust(t, e)
		if got != 1 {
			t.Fatal(got)
		}
		_, e = m.orderField(a, 24)
		if !errors.Is(e, umemory.ErrRuntimeUninitialized) {
			t.Fatal("degree must remain unread/uninitialized", e)
		}
	})
	t.Run("minimum reads only its degree", func(t *testing.T) {
		m := orderFixture(t)
		d := uint32(3)
		a := orderPartialValue(t, m, 1, &d)
		b := orderPartialValue(t, m, 2, nil)
		got, e := m.LexWithMinimum(a, b, 1)
		serviceMust(t, e)
		if got != -1 {
			t.Fatal(got)
		}
		got, e = m.LexWithMinimum(b, a, 1)
		serviceMust(t, e)
		if got != 1 {
			t.Fatal(got)
		}
		_, e = m.LexWithMinimum(a, b, 2)
		if !errors.Is(e, umemory.ErrRuntimeUninitialized) {
			t.Fatal("reached minimum degree", e)
		}
	})
	t.Run("equal variables reach degree", func(t *testing.T) {
		m := orderFixture(t)
		a := orderPartialValue(t, m, 1, nil)
		b := orderPartialValue(t, m, 1, nil)
		_, e := m.Lex(a, b)
		if !errors.Is(e, umemory.ErrRuntimeUninitialized) {
			t.Fatal(e)
		}
		_, e = m.ReverseLex(a, b)
		if !errors.Is(e, umemory.ErrRuntimeUninitialized) {
			t.Fatal(e)
		}
	})
	t.Run("both reference guards before fields", func(t *testing.T) {
		polynomialThrowAssertionFixture(t)
		m := orderFixture(t)
		a := orderPartialValue(t, m, 1, nil)
		b := orderPartialValue(t, m, 2, nil)
		l, e := m.location(b)
		serviceMust(t, e)
		serviceMust(t, m.allocator.WriteUint32(l.raw, 12, 0))
		_, e = m.ReverseLexRange(a, 0, 1, b, 0, 1)
		if polynomialSourceDebug {
			if !errors.Is(e, ErrMonomialAssertion) {
				t.Fatal(e)
			}
		} else {
			serviceMust(t, e)
		}
	})
}
