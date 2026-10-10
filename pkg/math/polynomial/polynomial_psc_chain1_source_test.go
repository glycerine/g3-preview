// Source: src/math/polynomial/polynomial.cpp:5574-5583,5897-5902;
// src/test/polynomial.cpp:702-723,762-765.
package polynomial

import (
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

// This dormant alternative retains degree-indexed zeros and gains no active
// driver assertion credit. The public default PSCChain remains optimized.
func TestPolynomialPSCChain1SourceDegreeIndexedCoefficientLayout(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	p := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 4}}, nil})
	q := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, nil})
	out, e := NewPolynomialRefVector(c)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.CleanupPSCChain1()); serviceMust(t, out.Close()) })
	serviceMust(t, m.PSCChain1(p, q, 0, out))
	n, e := out.Size()
	serviceMust(t, e)
	if n != 5 {
		t.Fatal("literal degree-indexed output", n)
	}
	for i, want := range []Polynomial{c.unit, c.zero, c.zero, c.unit} {
		got, e := out.Get(uint32(i + 1))
		serviceMust(t, e)
		eq, e := c.Equal(got, want)
		serviceMust(t, e)
		if !eq {
			t.Fatal("literal ascending coefficient/last unit", i+1)
		}
	}
	got, e := out.Get(0)
	serviceMust(t, e)
	constant, e := c.IsConstant(got)
	serviceMust(t, e)
	v, e := c.Coefficient(got, 0)
	serviceMust(t, e)
	if !constant || v != unumeral.NewInt(2) {
		t.Fatal("independent p(-1)", constant, v)
	}
}

// Copying the real owning Go object grants no authority over its embedded
// source frame or a caller-vector INC. Readonly history is not acquisition.
func TestPolynomialPSCChain1SubchainExactFrameAndCopiedOwnerRefuse(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	p := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, ^uint32(0)}}, nil})
	q := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, nil})
	out, e := NewPolynomialRefVector(c)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.CleanupPSCChain1()); serviceMust(t, out.Close()) })
	returned := m.PSCChain1(p, q, 0, out)
	if returned == nil || out.state.incoming != p {
		t.Fatal("actual source range stop", returned)
	}
	r := m.pscChain1.subchain
	copiedFrame := *r.frame
	forged := r
	forged.frame = &copiedFrame
	before, e := c.readWord(p, 0)
	serviceMust(t, e)
	serviceMust(t, m.cleanupSubresultantChainInvocation(&forged, false))
	serviceMust(t, m.cleanupSubresultantChainInvocation(&forged, true))
	after, e := c.readWord(p, 0)
	serviceMust(t, e)
	if before != after || out.state.incoming != p {
		t.Fatal("copied child frame consumed caller incoming")
	}
	copied := *m
	if e = copied.finishSubresultantChainSource(); e != ErrPolynomialOwner {
		t.Fatal("copied owner source child accepted", e)
	}
	if e = copied.finishPSCChain1Source(); e != ErrPolynomialOwner {
		t.Fatal("copied owner source caller accepted", e)
	}
	serviceMust(t, m.CleanupPSCChain1())
}
