package polynomial

// Source: src/test/polynomial.cpp:702-723,741-821.

import (
	"strconv"
	"strings"
	"testing"
)

// A bounded independent TEST-only symbolic model for original expressions.
// It does not invoke any production polynomial algorithm while expanding.
type pscModel map[[11]uint32]int64
type pscModelParser struct {
	s string
	i int
}

func pscModelConstant(c int64) pscModel { return pscModel{[11]uint32{}: c} }
func pscModelProduct(a, b pscModel) pscModel {
	r := pscModel{}
	for x, c := range a {
		for y, d := range b {
			var k [11]uint32
			for i := range k {
				k[i] = x[i] + y[i]
			}
			r[k] += c * d
		}
	}
	return r
}
func (p *pscModelParser) skip() {
	for p.i < len(p.s) && p.s[p.i] == ' ' {
		p.i++
	}
}
func (p *pscModelParser) atom() pscModel {
	p.skip()
	var a pscModel
	if p.s[p.i] == '(' {
		p.i++
		a = p.sum()
		p.skip()
		if p.s[p.i] != ')' {
			panic("model closing parenthesis")
		}
		p.i++
	} else {
		start := p.i
		if p.s[p.i] >= '0' && p.s[p.i] <= '9' {
			for p.i < len(p.s) && p.s[p.i] >= '0' && p.s[p.i] <= '9' {
				p.i++
			}
			c, e := strconv.ParseInt(p.s[start:p.i], 10, 64)
			if e != nil {
				panic(e)
			}
			a = pscModelConstant(c)
		} else {
			p.i++
			v := uint32(0)
			ch := p.s[start]
			if ch >= 'a' && ch <= 'f' {
				v = uint32(ch - 'a')
			} else if ch == 'y' {
				v = 10
			} else if ch == 'x' {
				v = 9
				start = p.i
				for p.i < len(p.s) && p.s[p.i] >= '0' && p.s[p.i] <= '9' {
					p.i++
				}
				if start != p.i {
					n, e := strconv.ParseUint(p.s[start:p.i], 10, 32)
					if e != nil {
						panic(e)
					}
					v = uint32(n)
				}
			} else {
				panic("model variable")
			}
			var k [11]uint32
			k[v] = 1
			a = pscModel{k: 1}
		}
	}
	p.skip()
	if p.i < len(p.s) && p.s[p.i] == '^' {
		p.i++
		p.skip()
		start := p.i
		for p.i < len(p.s) && p.s[p.i] >= '0' && p.s[p.i] <= '9' {
			p.i++
		}
		n, e := strconv.ParseUint(p.s[start:p.i], 10, 32)
		if e != nil {
			panic(e)
		}
		r := pscModelConstant(1)
		for i := uint64(0); i < n; i++ {
			r = pscModelProduct(r, a)
		}
		a = r
	}
	return a
}
func (p *pscModelParser) product() pscModel {
	a := p.atom()
	p.skip()
	for p.i < len(p.s) && p.s[p.i] == '*' {
		p.i++
		a = pscModelProduct(a, p.atom())
		p.skip()
	}
	return a
}
func (p *pscModelParser) sum() pscModel {
	a := p.product()
	p.skip()
	for p.i < len(p.s) && (p.s[p.i] == '+' || p.s[p.i] == '-') {
		sign := int64(1)
		if p.s[p.i] == '-' {
			sign = -1
		}
		p.i++
		b := p.product()
		for k, c := range b {
			a[k] += sign * c
		}
		p.skip()
	}
	return a
}
func pscOriginalPolynomial(t *testing.T, m *MultivariatePolynomialCore, s string) Polynomial {
	t.Helper()
	p := pscModelParser{s: strings.TrimSpace(s)}
	a := p.sum()
	p.skip()
	if p.i != len(p.s) {
		t.Fatal("unparsed model", s, p.i)
	}
	var coeff []int32
	var powers [][]MonomialPower
	for k, c := range a {
		if c == 0 {
			continue
		}
		if c < int64(-1<<31) || c > int64(1<<31-1) {
			t.Fatal("bounded coefficient", c)
		}
		coeff = append(coeff, int32(c))
		var term []MonomialPower
		for v, d := range k {
			if d != 0 {
				term = append(term, MonomialPower{uint32(v), d})
			}
		}
		powers = append(powers, term)
	}
	if len(coeff) == 0 {
		return m.zero
	}
	return polynomialQueryTerms(t, m, coeff, powers)
}

func TestPolynomialPSCAllEightOriginalCoefficientFixtures(t *testing.T) {
	for i, tc := range polynomialPSCOriginalCases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			full := polynomialPSCFullFixture(t)
			m := full.core
			p := pscOriginalPolynomial(t, m, tc.p)
			q := pscOriginalPolynomial(t, m, tc.q)
			first := pscOriginalPolynomial(t, m, tc.first)
			second := pscOriginalPolynomial(t, m, tc.second)
			S, e := NewPolynomialRefVector(m)
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, S.Close()) })
			serviceMust(t, full.PSCChain(p, q, tc.x, S))
			n, e := S.Size()
			serviceMust(t, e)
			for j, want := range []Polynomial{first, second} {
				if uint32(j) >= n {
					continue
				}
				got, e := S.Get(uint32(j))
				serviceMust(t, e)
				equal, e := m.Equal(got, want)
				serviceMust(t, e)
				if !equal {
					neg, e := m.Neg(want)
					serviceMust(t, e)
					serviceMust(t, m.IncRef(neg))
					equal, e = m.Equal(got, neg)
					serviceMust(t, e)
					serviceMust(t, m.DecRef(neg))
				}
				if !equal {
					t.Fatal("original reached coefficient ENSURE", i, j, n)
				}
			}
		})
	}
}
