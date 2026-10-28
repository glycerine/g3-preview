package polynomial

// Source: src/math/polynomial/polynomial.cpp:5588-5658,5897-5902.

import (
	"errors"
	"testing"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

// Independent constants use p(-1), the norm of x+1 modulo x²+1,
// and the norm of 2x+2 modulo x²+1; the second compact item follows
// the source leading-coefficient/sign recurrence, not a PSC expected call.
func TestPolynomialPSCChain2LiteralRegularAndDefectiveReverse(t *testing.T) {
	for _, tc := range []struct {
		p, q   []int32
		pd, qd []uint32
		want   []int32
	}{{[]int32{1, 1}, []int32{1, 1}, []uint32{2, 0}, []uint32{1, 0}, []int32{2}}, {[]int32{1, 2, 1}, []int32{1, 1}, []uint32{3, 1, 0}, []uint32{2, 0}, []int32{2, 1}}, {[]int32{1, 2, 1}, []int32{1, 1}, []uint32{4, 1, 0}, []uint32{2, 0}, []int32{8, -2}}, {[]int32{1, 1}, []int32{1, 1}, []uint32{4, 0}, []uint32{1, 0}, []int32{2}}} {
		m := polynomialPSCFullFixture(t)
		c := m.core
		mk := func(as []int32, ds []uint32) Polynomial {
			powers := make([][]MonomialPower, len(ds))
			for i, d := range ds {
				if d != 0 {
					powers[i] = []MonomialPower{{0, d}}
				}
			}
			return polynomialQueryTerms(t, c, as, powers)
		}
		p, q := mk(tc.p, tc.pd), mk(tc.q, tc.qd)
		out, e := NewPolynomialRefVector(c)
		serviceMust(t, e)
		t.Cleanup(func() { serviceMust(t, m.CleanupPSCChain2()); serviceMust(t, out.Close()) })
		serviceMust(t, m.PSCChain2(p, q, 0, out))
		n, e := out.Size()
		serviceMust(t, e)
		if n != uint32(len(tc.want)) {
			t.Fatal("literal compact output", n, tc.want)
		}
		for i, want := range tc.want {
			v, e := out.Get(uint32(i))
			serviceMust(t, e)
			constant, e := c.IsConstant(v)
			serviceMust(t, e)
			a, e := c.Coefficient(v, 0)
			serviceMust(t, e)
			if !constant || a != unumeral.NewInt(want) {
				t.Fatal("source reverse/independent bounded scalar", i, a, want)
			}
		}
	}
}
func TestPolynomialPSCChain2FirstZeroDivisorGuardsUninitializedN3(t *testing.T) {
	m := polynomialPSCFullFixture(t)
	c := m.core
	p := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 2}}, nil})
	out, e := NewPolynomialRefVector(c)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.CleanupPSCChain2()); serviceMust(t, out.Close()) })
	err := m.PSCChain2(p, c.zero, 0, out)
	if !errors.Is(err, umemory.ErrRuntimeUninitialized) {
		t.Fatal("literal uninitialized n3 read repaired or replaced", err)
	}
	n, e := out.Size()
	serviceMust(t, e)
	if n != 0 {
		t.Fatal("source reset output before undefined read", n)
	}
	s := &m.pscChain2
	if s.constructed != 0 || s.n3Initialized || s.bodyCause != err || s.sourceCause != nil {
		t.Fatal("uninitialized read source suffix/history", s.constructed, s.bodyCause, s.sourceCause)
	}
	serviceMust(t, m.CleanupPSCChain2())
	if s.bodyCause != err {
		t.Fatal("own acknowledgment erased undefined read guard")
	}
	original := err.Error()
	q := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 1}}, nil})
	serviceMust(t, m.PSCChain2(p, q, 0, out))
	if err.Error() != original || !errors.Is(err, umemory.ErrRuntimeUninitialized) {
		t.Fatal("later invocation retargeted original guard")
	}
}
