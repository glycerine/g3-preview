package polynomial

// Source: src/math/polynomial/polynomial.cpp:449-505; src/math/polynomial/polynomial.h.

import (
	"errors"
	"math"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func positionFixture(t *testing.T) (*MonomialManager, *MonomialPositions, []Monomial) {
	t.Helper()
	m := monomialFixture(t)
	for i := 0; i < 8; i++ {
		_, e := m.MkVar()
		serviceMust(t, e)
	}
	xs := make([]Monomial, 8)
	for i := range xs {
		xs[i] = monomialOwn(t, m, []MonomialPower{{uint32(i), 1}})
	}
	p, e := NewMonomialPositions(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, p.CloseChecked()) })
	return m, p, xs
}
func TestMonomialPositionsSourceMapping(t *testing.T) {
	_, p, xs := positionFixture(t)
	for i, a := range xs {
		v, e := p.Get(a)
		serviceMust(t, e)
		if v != math.MaxUint32 {
			t.Fatal(v)
		}
		serviceMust(t, p.Set(a, uint32(i*7)))
	}
	for i, a := range xs {
		v, e := p.Get(a)
		serviceMust(t, e)
		if v != uint32(i*7) {
			t.Fatal(i, v)
		}
	}
	if len(p.values) != 9 {
		t.Fatal(len(p.values))
	}
	for i, a := range xs {
		serviceMust(t, p.Reset(a))
		serviceMust(t, p.Set(a, uint32(i)))
	}
	for i, a := range xs {
		v, e := p.Get(a)
		serviceMust(t, e)
		if v != uint32(i) {
			t.Fatal(i, v)
		}
	}
	serviceMust(t, p.SourceFinalize())
	if p.values != nil {
		t.Fatal("native position storage survives finalization")
	}
}
func TestMonomialPositionsAssertionsAndOwners(t *testing.T) {
	polynomialAssertionActionsFixture(t, "TT")
	_, p, xs := positionFixture(t)
	e := p.Reset(xs[0])
	if polynomialSourceDebug {
		if !errors.Is(e, ErrMonomialAssertion) {
			t.Fatal(e)
		}
	} else if !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	serviceMust(t, p.Set(xs[0], 3))
	e = p.Set(xs[0], 5)
	if polynomialSourceDebug {
		if !errors.Is(e, ErrMonomialAssertion) {
			t.Fatal(e)
		}
	} else {
		serviceMust(t, e)
		v, e := p.Get(xs[0])
		serviceMust(t, e)
		if v != 5 {
			t.Fatal(v)
		}
	}
	copy := *p
	if _, e = copy.Get(xs[0]); !errors.Is(e, ErrMonomialOwner) {
		t.Fatal(e)
	}
	if _, e = p.Get(Monomial{}); !errors.Is(e, ErrMonomialOwner) {
		t.Fatal(e)
	}
	serviceMust(t, p.CloseChecked())
	if _, e = p.Get(xs[0]); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
	var zero MonomialPositions
	if e = zero.CloseChecked(); !errors.Is(e, ErrMonomialOwner) {
		t.Fatal(e)
	}
}
func TestMonomialPositionsNativeGrowthIgnoresArenaFailure(t *testing.T) {
	m, p, xs := positionFixture(t)
	before := m.config.Budget.Used()
	blocks := m.memory.Stats()
	m.config.Budget.FailAfter(0)
	v, e := p.Get(xs[7])
	m.config.Budget.FailAfter(-1)
	serviceMust(t, e)
	if v != math.MaxUint32 || m.config.Budget.Used() != before || m.memory.Stats() != blocks {
		t.Fatal(v, before, m.config.Budget.Used(), blocks, m.memory.Stats())
	}
}
func TestMonomialPositionsColdChargeAndWarm(t *testing.T) {
	m, p, xs := positionFixture(t)
	if p.header != uint64(reflect.TypeOf(*p).Size()) {
		t.Fatal(p.header)
	}
	for i, a := range xs {
		serviceMust(t, p.Set(a, uint32(i)))
	}
	before := m.config.Budget.Used()
	blocks := m.memory.Stats()
	alloc := testing.AllocsPerRun(100, func() {
		for _, a := range xs {
			if _, e := p.Get(a); e != nil {
				panic(e)
			}
		}
	})
	if alloc != 0 || m.config.Budget.Used() != before || m.memory.Stats() != blocks {
		t.Fatal(alloc)
	}
}
func TestMonomialPositionsScalarIDOnlyAndRejectedWrap(t *testing.T) {
	m, p, xs := positionFixture(t)
	l, e := m.location(xs[0])
	serviceMust(t, e)
	original := l.raw
	raw, e := m.allocator.Allocate(24)
	serviceMust(t, e)
	serviceMust(t, m.allocator.WriteUint32(raw, 4, 1))
	l.raw = raw
	v, e := p.Get(xs[0])
	serviceMust(t, e)
	if v != math.MaxUint32 {
		t.Fatal(v)
	}
	// No other monomial header word is initialized. The checked Go boundary
	// rejects the source id+1 wrap without changing the map.
	serviceMust(t, m.allocator.WriteUint32(raw, 4, math.MaxUint32))
	_, e = p.Get(xs[0])
	if polynomialSourceDebug {
		if !errors.Is(e, ErrMonomialAssertion) {
			t.Fatal(e)
		}
	} else if !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	l.raw = original
	serviceMust(t, m.allocator.Deallocate(24, raw))
}
