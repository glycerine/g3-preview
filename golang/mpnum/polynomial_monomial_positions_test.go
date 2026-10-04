// Source: src/math/polynomial/polynomial.cpp; src/math/polynomial/polynomial.h. Licensed under the MIT license.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"math"
	"reflect"
	"testing"
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
	m, p, xs := positionFixture(t)
	before := m.memory.Stats().LiveBlocks
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
	if m.memory.Stats().LiveBlocks != before+1 {
		t.Fatal("one real unsigned vector")
	}
	n, e := m.memory.ReadUint32(p.raw, 4)
	serviceMust(t, e)
	if n != 9 {
		t.Fatal(n)
	}
	size, e := m.memory.Size(p.raw)
	serviceMust(t, e)
	cap, e := m.memory.ReadUint32(p.raw, 0)
	serviceMust(t, e)
	if size != 8+uint64(cap)*4 {
		t.Fatal(size, cap)
	}
	serviceMust(t, p.ResetMonomials(xs))
	serviceMust(t, p.SetMonomials(xs))
	for i, a := range xs {
		v, e := p.Get(a)
		serviceMust(t, e)
		if v != uint32(i) {
			t.Fatal(i, v)
		}
	}
	serviceMust(t, p.SourceFinalize())
	if p.raw != (memory.Block{}) {
		t.Fatal("published source member survives destructor")
	}
}
func TestMonomialPositionsAssertionsAndOwners(t *testing.T) {
	_, p, xs := positionFixture(t)
	e := p.Reset(xs[0])
	if modularSourceDebug {
		if !errors.Is(e, ErrMonomialAssertion) {
			t.Fatal(e)
		}
	} else if e == nil {
		t.Fatal("null source write needs actual bounds guard")
	}
	serviceMust(t, p.Set(xs[0], 3))
	e = p.Set(xs[0], 5)
	if modularSourceDebug {
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
func TestMonomialPositionsRefusalPrefixes(t *testing.T) {
	failures, success := 0, false
	for ordinal := int64(0); ordinal < 20; ordinal++ {
		t.Run(string(rune('a'+ordinal)), func(t *testing.T) {
			m, p, xs := positionFixture(t)
			beforeStats := m.memory.Stats()
			beforeUsed := m.config.Budget.Used()
			before := beforeStats.LiveBlocks
			m.config.Budget.FailAfter(ordinal)
			_, e := p.Get(xs[7])
			if e != nil {
				failures++
				if !errors.Is(e, arena.ErrInjectedFailure) {
					var f memory.Failure
					if !errors.As(e, &f) || f.ErrorCode() != 101 || !m.memory.State().OutOfMemory {
						t.Fatal(e)
					}
				}
			} else {
				success = true
			}
			want := before
			if p.raw != (memory.Block{}) {
				want++
			}
			if p.pending != (memory.Block{}) {
				want++
			}
			if m.memory.Stats().LiveBlocks != want {
				t.Fatal("unnamed raw root", m.memory.Stats(), want)
			}
			now := m.memory.Stats()
			if m.config.Budget.Used() != beforeUsed+now.TotalBytes-beforeStats.TotalBytes {
				t.Fatal("once-only raw graph", m.config.Budget.Used(), beforeUsed, now, beforeStats)
			}
			m.config.Budget.FailAfter(-1)
		})
		if success {
			break
		}
	}
	if failures == 0 || !success {
		t.Fatal(failures, success)
	}
}
func TestMonomialPositionsColdChargeAndWarm(t *testing.T) {
	m, p, xs := positionFixture(t)
	if p.header != uint64(reflect.TypeOf(*p).Size()) {
		t.Fatal(p.header)
	}
	serviceMust(t, p.SetMonomials(xs))
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

func TestMonomialPositionsScalarIDOnlyAndWrap(t *testing.T) {
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
	// No other monomial header word is initialized. Literal source id+1 wraps.
	serviceMust(t, m.allocator.WriteUint32(raw, 4, math.MaxUint32))
	_, e = p.Get(xs[0])
	if modularSourceDebug {
		if !errors.Is(e, ErrMonomialAssertion) {
			t.Fatal(e)
		}
	} else if !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	l.raw = original
	serviceMust(t, m.allocator.Deallocate(24, raw))
}
func TestMonomialPositionsTLSUnpublishedOwnership(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	m, e := NewMonomialManager(MonomialManagerConfig{Budget: b, Memory: r, Profile: memory.SmallAllocatorProfile{PointerBits: 64}})
	serviceMust(t, e)
	p, e := NewMonomialPositions(m)
	serviceMust(t, e)
	a, e := m.Unit()
	serviceMust(t, e)
	// Real borrowed caller allocation approaches the actual TLS synchronization threshold.
	external, e := r.Allocate(uint64(99977 - r.State().LocalSize - 8))
	serviceMust(t, e)
	before := r.Stats().LiveBlocks
	serviceMust(t, r.SetMaxSize(1))
	_, e = p.Get(a)
	var f memory.Failure
	if !errors.As(e, &f) || f.ErrorCode() != 101 || !r.State().OutOfMemory {
		t.Fatal(e)
	}
	if p.raw != (memory.Block{}) || p.pending == (memory.Block{}) || r.Stats().LiveBlocks != before+1 {
		t.Fatal("unreturned initial malloc root", p.raw, p.pending, r.Stats())
	}
	if _, e = r.ReadUint32(p.pending, 0); e == nil {
		t.Fatal("source cap write must not run after allocation exception")
	}
	serviceMust(t, p.SourceFinalize())
	if p.pending == (memory.Block{}) || r.Stats().LiveBlocks != before+1 {
		t.Fatal("source member destructor consumed unpublished malloc")
	}
	serviceMust(t, r.SetMaxSize(0))
	serviceMust(t, p.CloseChecked())
	if r.Stats().LiveBlocks != before {
		t.Fatal(r.Stats())
	}
	serviceMust(t, r.Deallocate(&external))
	serviceMust(t, m.CloseChecked())
	serviceMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
func TestMonomialPositionsNullCapacitySourceOverflow(t *testing.T) {
	m, p, xs := positionFixture(t)
	serviceMust(t, p.Set(xs[0], 0))
	serviceMust(t, m.memory.WriteUint32(p.raw, 0, 0))
	e := p.expand()
	if modularSourceDebug {
		if !errors.Is(e, ErrMonomialAssertion) {
			t.Fatal(e)
		}
	} else {
		var f memory.Failure
		if !errors.As(e, &f) || f.Kind != memory.FailureVectorOverflow {
			t.Fatal(e)
		}
	}
}

func TestMonomialPositionsWrappedHeaderSourceAndOwningCleanup(t *testing.T) {
	b := arena.NewBudget(1 << 24)
	r, e := memory.NewRuntime(memory.RuntimeConfig{Budget: b, SizeTBits: 32})
	serviceMust(t, e)
	m, e := NewMonomialManager(MonomialManagerConfig{Budget: b, Memory: r, Profile: memory.SmallAllocatorProfile{PointerBits: 32}})
	serviceMust(t, e)
	p, e := NewMonomialPositions(m)
	serviceMust(t, e)
	a, e := m.Unit()
	serviceMust(t, e)
	_, e = p.Get(a)
	serviceMust(t, e)
	before := p.raw
	serviceMust(t, r.WriteUint32(p.raw, 0, 0x55555553))
	e = p.expand()
	if !errors.Is(e, arena.ErrBounds) {
		t.Fatal(e)
	}
	if p.raw == before || p.raw == (memory.Block{}) {
		t.Fatal("actual short realloc must publish changed token")
	}
	after := p.raw
	e = p.SourceFinalize()
	if !errors.Is(e, arena.ErrBounds) || p.raw != after {
		t.Fatal("source header guard", e, p.raw)
	}
	serviceMust(t, p.CloseChecked())
	serviceMust(t, m.CloseChecked())
	serviceMust(t, r.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}
