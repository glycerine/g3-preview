package polynomial

import (
	"errors"
	"reflect"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestPolynomialProjectionVectorSetShrinkReferenceMultiplicity(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	p := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, nil})
	q := polynomialQueryTerms(t, c, []int32{1, 2}, [][]MonomialPower{{{1, 1}}, nil})
	v, e := NewPolynomialRefVector(c)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, v.Close()) })
	pi, e := c.Info(p)
	serviceMust(t, e)
	qi, e := c.Info(q)
	serviceMust(t, e)
	for _, a := range []Polynomial{p, q, p, q, p} {
		serviceMust(t, v.Push(a))
	}
	serviceMust(t, v.Set(0, p))
	serviceMust(t, v.Set(1, p))
	serviceMust(t, v.Shrink(3))
	for _, pair := range []struct {
		p    Polynomial
		refs uint32
	}{{p, pi.RefCount + 3}, {q, qi.RefCount}} {
		info, e := c.Info(pair.p)
		serviceMust(t, e)
		if info.RefCount != pair.refs {
			t.Fatal("SET/self-alias/SHRINK references", info.RefCount, pair.refs)
		}
	}
	for i := uint32(0); i < 3; i++ {
		a, e := v.Get(i)
		serviceMust(t, e)
		if a != p {
			t.Fatal("compaction association", i)
		}
	}
	serviceMust(t, v.Shrink(3))
	serviceMust(t, v.Shrink(0))
	serviceMust(t, v.Push(q))
	serviceMust(t, v.Reset())
	for _, pair := range []struct {
		p    Polynomial
		refs uint32
	}{{p, pi.RefCount}, {q, qi.RefCount}} {
		info, e := c.Info(pair.p)
		serviceMust(t, e)
		if info.RefCount != pair.refs {
			t.Fatal("post-reset references", info.RefCount, pair.refs)
		}
	}
}
func TestPolynomialProjectionVectorStoppedShrinkOwnsForwardTail(t *testing.T) {
	for _, closeInstead := range []bool{false, true} {
		t.Run(map[bool]string{false: "cleanup", true: "close"}[closeInstead], func(t *testing.T) {
			m := polynomialCellFixture(t)
			v, e := NewPolynomialRefVector(m)
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, v.Close()) })
			var polys []Polynomial
			for k := uint32(1); k <= 4; k++ {
				p, e := m.MkVariable(0, k)
				serviceMust(t, e)
				serviceMust(t, v.Push(p))
				polys = append(polys, p)
			}
			stop := errors.New("stop actual shrink deletion")
			var order []Polynomial
			h, e := NewPolynomialDeletionHandler(m.integer.Budget(), func(_ *PolynomialDeletionHandler, _ *MultivariatePolynomialCore, p Polynomial) error {
				order = append(order, p)
				if p == polys[2] {
					return stop
				}
				return nil
			})
			serviceMust(t, e)
			serviceMust(t, m.AddDeletionHandler(h))
			t.Cleanup(func() { serviceMust(t, m.RemoveDeletionHandler(h)); serviceMust(t, h.CloseChecked()) })
			e = v.Shrink(1)
			if !errors.Is(e, stop) || !v.state.shrinking || v.state.drain != 2 || !v.state.drainDec {
				t.Fatal("lost consumed forward shrink prefix", e, v.state.drain)
			}
			n, e := v.Size()
			serviceMust(t, e)
			if n != 4 {
				t.Fatal("extent published before tail destruction", n)
			}
			if e := v.Push(polys[0]); !errors.Is(e, ErrPolynomialPending) {
				t.Fatal("accepted write during pending shrink", e)
			}
			if e := v.Reset(); !errors.Is(e, ErrPolynomialPending) {
				t.Fatal("ordinary RESET adopted stopped shrink", e)
			}
			if closeInstead {
				serviceMust(t, v.Close())
			} else {
				serviceMust(t, v.CleanupRetained())
				n, e = v.Size()
				serviceMust(t, e)
				if n != 1 || !reflect.DeepEqual(order, polys[1:]) {
					t.Fatal("cleanup replayed a DEC or lost tail order", n, order)
				}
				serviceMust(t, v.Close())
			}
			want := append(append([]Polynomial(nil), polys[1:]...), polys[0])
			if !reflect.DeepEqual(order, want) {
				t.Fatal("tail/prefix owning close order", order, want)
			}
		})
	}
}
func TestPolynomialProjectionVectorSetRefusalAndShrinkGuard(t *testing.T) {
	polynomialThrowAssertionFixture(t)
	m := polynomialCellFixture(t)
	p, e := m.MkVariable(0, 1)
	serviceMust(t, e)
	serviceMust(t, m.IncRef(p))
	t.Cleanup(func() { serviceMust(t, m.DecRef(p)) })
	v, e := NewPolynomialRefVector(m)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, v.Close()) })
	serviceMust(t, v.Push(p))
	e = v.Set(7, p)
	if e == nil || !v.state.setActive || v.state.setOldRead || v.state.incoming != p {
		t.Fatal("SET did not preserve reached incoming INC", e)
	}
	info, e := m.Info(p)
	serviceMust(t, e)
	if info.RefCount != 3 {
		t.Fatal("SET must INC before bounds read", info.RefCount)
	}
	serviceMust(t, v.CleanupRetained())
	info, e = m.Info(p)
	serviceMust(t, e)
	if info.RefCount != 2 {
		t.Fatal("owning SET abort lost caller reference", info.RefCount)
	}
	e = v.Shrink(2)
	if polynomialSourceDebug {
		if !linearThrownAssertion(e) {
			t.Fatal("shrink source guard", e)
		}
	} else if !errors.Is(e, arena.ErrBounds) {
		t.Fatal("native out-of-domain shrink guard", e)
	}
	serviceMust(t, v.Shrink(1))
}

func TestPolynomialProjectionIDReadsOnlyIDField(t *testing.T) {
	m := polynomialCellFixture(t)
	p, e := m.MkVariable(0, 1)
	serviceMust(t, e)
	serviceMust(t, m.IncRef(p))
	t.Cleanup(func() { serviceMust(t, m.DecRef(p)) })
	flags, e := m.readWord(p, 4)
	serviceMust(t, e)
	location, e := m.polynomialLocation(p)
	serviceMust(t, e)
	allocator := m.monomials.allocator
	raw, e := allocator.Allocate(12)
	serviceMust(t, e)
	original := location.raw
	location.raw = raw
	defer func() { location.raw = original; serviceMust(t, allocator.Deallocate(12, raw)) }()
	// Qualified read-only prefix: no completed polynomial is claimed here.
	// Only the actual owning location is restored before any source destructor.
	serviceMust(t, allocator.WriteUint32(raw, 4, flags|0x80000000))
	id, e := m.PolynomialID(p)
	serviceMust(t, e)
	if id != flags&0x7fffffff {
		t.Fatal("ID query included lexical-sort flag", id, flags)
	}
}
