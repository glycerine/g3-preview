// Source: src/math/polynomial/polynomial.cpp:5499-5572;
// src/test/polynomial.cpp:702-723,762-765.
package polynomial

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

// The registered driver dispatches optimized PSC, not this dormant chain1
// alternative. These literal fixtures retain no new active-driver credit.
func TestPolynomialSubresultantChainSourceRegularFixture(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	p := polynomialQueryTerms(t, c, []int32{1, 1, 1, 2}, [][]MonomialPower{{{0, 1}, {1, 2}}, {{0, 1}, {1, 1}}, {{1, 1}}, nil})
	q := polynomialQueryTerms(t, c, []int32{1, 3}, [][]MonomialPower{{{0, 1}, {1, 1}}, nil})
	want := polynomialQueryTerms(t, c, []int32{6, -1}, [][]MonomialPower{{{0, 1}}, {{0, 2}}})
	out, e := NewPolynomialRefVector(c)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, out.Close()) })
	serviceMust(t, m.SubresultantChain(p, q, 1, out))
	n, e := out.Size()
	serviceMust(t, e)
	if n != 3 {
		t.Fatal("literal degree-indexed source range", n)
	}
	for i, token := range []Polynomial{q, p} {
		got, e := out.Get(uint32(i + 1))
		serviceMust(t, e)
		if got != token {
			t.Fatal("upper source input references", i, got, token)
		}
	}
	got, e := out.Get(0)
	serviceMust(t, e)
	equal, e := c.Equal(got, want)
	serviceMust(t, e)
	if !equal {
		negative, e := c.Neg(want)
		serviceMust(t, e)
		serviceMust(t, c.IncRef(negative))
		equal, e = c.Equal(got, negative)
		serviceMust(t, e)
		serviceMust(t, c.DecRef(negative))
	}
	if !equal {
		t.Fatal("source principal coefficient ENSURE polynomial.cpp:702-723")
	}
}

// Independent bounded evaluations at the linear divisor root verify the final
// constant; expected intermediate rows follow the literal defective branch.
func TestPolynomialSubresultantChainDefectiveGapAndSwappedInputs(t *testing.T) {
	for _, swapped := range []bool{false, true} {
		m := polynomialPSCFullFixture(t)
		c := m.core
		p := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 4}}, nil})
		q := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, nil})
		out, e := NewPolynomialRefVector(c)
		serviceMust(t, e)
		t.Cleanup(func() { serviceMust(t, out.Close()) })
		a, b := p, q
		if swapped {
			a, b = b, a
		}
		serviceMust(t, m.SubresultantChain(a, b, 0, out))
		n, e := out.Size()
		serviceMust(t, e)
		if n != 5 {
			t.Fatal("degree-indexed output", n)
		}
		for i, want := range []Polynomial{q, c.zero, q, p} {
			got, e := out.Get(uint32(i + 1))
			serviceMust(t, e)
			equal, e := c.Equal(got, want)
			serviceMust(t, e)
			if !equal {
				t.Fatal("literal defective gap/degree row", i+1)
			}
		}
		got, e := out.Get(0)
		serviceMust(t, e)
		constant, e := c.IsConstant(got)
		serviceMust(t, e)
		if !constant {
			t.Fatal("linear-divisor result has spurious variables")
		}
		v, e := c.Coefficient(got, 0)
		serviceMust(t, e)
		if v != unumeral.NewInt(2) {
			t.Fatal("independent p(-1) result", v)
		}
	}
}

// Checked SASSERT boundaries are retained, including the source equal-degree
// limitation. These are assertion-domain cases, not successful chain fixtures.
func TestPolynomialSubresultantChainCheckedSourceAssertions(t *testing.T) {
	if !polynomialSourceDebug {
		return
	}
	m := polynomialPSCFullFixture(t)
	c := m.core
	out, e := NewPolynomialRefVector(c)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.CleanupSubresultantChain()); serviceMust(t, out.Close()) })
	serviceMust(t, out.Push(c.unit))
	err := m.SubresultantChain(c.unit, c.unit, 0, out)
	if !errors.Is(err, ErrPolynomialAssertion) || m.subresultant.constructed != 0 {
		t.Fatal("degree assertion must precede reset and constructors", err)
	}
	n, e := out.Size()
	serviceMust(t, e)
	if n != 1 {
		t.Fatal("degree guard mutated caller output", n)
	}
	p := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 2}}, nil})
	q := polynomialQueryTerms(t, c, []int32{1, 2}, [][]MonomialPower{{{0, 2}}, nil})
	err = m.SubresultantChain(p, q, 0, out)
	if !errors.Is(err, ErrPolynomialAssertion) || m.subresultant.constructed != 0 {
		t.Fatal("literal r<j assertion/local unwind", err, m.subresultant.constructed)
	}
	n, e = out.Size()
	serviceMust(t, e)
	if n != 3 {
		t.Fatal("partial output range lost", n)
	}
	for i, want := range []Polynomial{q, p} {
		got, e := out.Get(uint32(i + 1))
		serviceMust(t, e)
		if got != want {
			t.Fatal("partial source input slot", i, got, want)
		}
	}
	serviceMust(t, m.CleanupSubresultantChain())
}

// Source unsigned n+1 wraps before pointer resize; no giant allocation or
// reference constructor is substituted at the reached empty-range SET guard.
func TestPolynomialSubresultantChainActualUnsignedResizeWrap(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	p := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, math.MaxUint32}}, nil})
	q := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, nil})
	out, e := NewPolynomialRefVector(c)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.CleanupSubresultantChain()); serviceMust(t, out.Close()) })
	serviceMust(t, out.Push(c.unit))
	before, e := c.readWord(p, 0)
	serviceMust(t, e)
	err := m.SubresultantChain(p, q, 0, out)
	after, e := c.readWord(p, 0)
	serviceMust(t, e)
	if after != before+1 || out.state.incoming != p || !out.state.setActive {
		t.Fatal("source INC before invalid range assertion lost", before, after, out.state.incoming, out.state.setActive)
	}
	subresultantNamedAudit(t, m, out, []Polynomial{p, q}, nil)
	want := error(arena.ErrBounds)
	if polynomialSourceDebug {
		want = ErrPolynomialAssertion
	}
	if !errors.Is(err, want) || m.subresultant.constructed != 0 {
		t.Fatal("wrapped resize actual SET boundary", err, m.subresultant.constructed)
	}
	n, e := out.Size()
	serviceMust(t, e)
	if n != 0 {
		t.Fatal("unsigned zero source range", n)
	}
	serviceMust(t, m.CleanupSubresultantChain())
	after, e = c.readWord(p, 0)
	serviceMust(t, e)
	subresultantNamedAudit(t, m, out, []Polynomial{p, q}, nil)
	if after != before || out.state.incoming != (Polynomial{}) || out.state.setActive {
		t.Fatal("invalid-index exact incoming ACK", before, after, out.state.incoming)
	}
}
