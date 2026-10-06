// Source: src/test/polynomial.cpp:341-451;
// src/math/polynomial/polynomial.cpp:5473-5500.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
)

func polynomialDiscriminantFixture(t *testing.T) *MultivariatePolynomialManager {
	t.Helper()
	q, _ := rationalManager(t, false)
	m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.z, Rationals: q, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
	serviceMust(t, e)
	t.Cleanup(func() {
		serviceMust(t, m.CleanupDiscriminant())
		serviceMust(t, m.CloseChecked())
	})
	return m
}

// This fixture-only parser evaluates literal source expressions through the
// existing Add/Sub/Mul/Power APIs. Expected expressions never call Discriminant.
type polynomialDiscriminantExpression struct {
	t         *testing.T
	c         *MultivariatePolynomialCore
	variables [5]Polynomial
	text      string
	position  int
}

func (p *polynomialDiscriminantExpression) own(v Polynomial, e error) Polynomial {
	p.t.Helper()
	serviceMust(p.t, e)
	return polynomialCaller(p.t, p.c, v)
}
func (p *polynomialDiscriminantExpression) take(ch byte) bool {
	if p.position < len(p.text) && p.text[p.position] == ch {
		p.position++
		return true
	}
	return false
}
func (p *polynomialDiscriminantExpression) number() uint32 {
	start := p.position
	for p.position < len(p.text) && p.text[p.position] >= '0' && p.text[p.position] <= '9' {
		p.position++
	}
	if start == p.position {
		p.t.Fatalf("source fixture missing number at %q", p.text[start:])
	}
	n, e := strconv.ParseUint(p.text[start:p.position], 10, 32)
	serviceMust(p.t, e)
	return uint32(n)
}
func (p *polynomialDiscriminantExpression) primary() Polynomial {
	var v Polynomial
	if p.take('(') {
		v = p.sum()
		if !p.take(')') {
			p.t.Fatal("source fixture closing parenthesis", p.text)
		}
	} else if p.position < len(p.text) && p.text[p.position] >= '0' && p.text[p.position] <= '9' {
		i := NewInt(int32(p.number()))
		v = p.own(p.c.MkConstantTakeCopy(&i))
	} else {
		if p.position == len(p.text) {
			p.t.Fatal("source fixture missing operand")
		}
		ch := p.text[p.position]
		p.position++
		switch ch {
		case 'a', 'b', 'c', 'd':
			v = p.variables[ch-'a']
		case 'x':
			v = p.variables[4]
		case 'o':
			if !strings.HasPrefix(p.text[p.position:], "ne") {
				p.t.Fatal("source fixture unknown name")
			}
			p.position += 2
			v = p.c.unit
		case 'z':
			if !strings.HasPrefix(p.text[p.position:], "ero") {
				p.t.Fatal("source fixture unknown name")
			}
			p.position += 3
			v = p.c.zero
		default:
			p.t.Fatalf("source fixture unknown operand %c", ch)
		}
	}
	if p.take('^') {
		n := p.number()
		r, e := NewPolynomialRef(p.c)
		serviceMust(p.t, e)
		p.t.Cleanup(func() { serviceMust(p.t, r.Close()) })
		serviceMust(p.t, p.c.Power(v, n, r))
		v = r.root.value
	}
	return v
}
func (p *polynomialDiscriminantExpression) unary() Polynomial {
	if p.take('-') {
		return p.own(p.c.Neg(p.unary()))
	}
	if p.take('+') {
		return p.unary()
	}
	return p.primary()
}
func (p *polynomialDiscriminantExpression) product() Polynomial {
	v := p.unary()
	for p.take('*') {
		v = p.own(p.c.Mul(v, p.unary()))
	}
	return v
}
func (p *polynomialDiscriminantExpression) sum() Polynomial {
	v := p.product()
	for {
		if p.take('+') {
			v = p.own(p.c.Add(v, p.product()))
		} else if p.take('-') {
			v = p.own(p.c.Sub(v, p.product()))
		} else {
			return v
		}
	}
}
func (p *polynomialDiscriminantExpression) evaluate(s string) Polynomial {
	p.text = strings.Join(strings.Fields(s), "")
	p.position = 0
	v := p.sum()
	if p.position != len(p.text) {
		p.t.Fatal("source fixture trailing expression", p.text[p.position:])
	}
	return v
}

func TestPolynomialDiscriminantAllOriginalDormantAssertions(t *testing.T) {
	var fixtures []struct {
		Line     int    `json:"line"`
		Input    string `json:"input"`
		Variable string `json:"variable"`
		Expected string `json:"expected"`
	}
	b, e := os.ReadFile("test_vectors/polynomial_discriminant_original.json")
	serviceMust(t, e)
	serviceMust(t, json.Unmarshal(b, &fixtures))
	if len(fixtures) != 24 {
		t.Fatal("literal original source fixture count", len(fixtures))
	}
	for _, f := range fixtures {
		t.Run(fmt.Sprint(f.Line), func(t *testing.T) {
			m := polynomialDiscriminantFixture(t)
			c := m.core
			parser := polynomialDiscriminantExpression{t: t, c: c}
			for j := uint32(0); j < 5; j++ {
				v, e := c.monomials.MkVar()
				serviceMust(t, e)
				if v != j {
					t.Fatal("source variable construction order")
				}
				parser.variables[j] = polynomialQueryTerms(t, c, []int32{1}, [][]MonomialPower{{{j, 1}}})
			}
			p := parser.evaluate(f.Input)
			want := parser.evaluate(f.Expected)
			x, e := c.MaxVariable(p)
			serviceMust(t, e)
			switch f.Variable {
			case "max_var(input)":
			case "max_var(a)":
				x = 0
			case "max_var(b)":
				x = 1
			default:
				t.Fatal("source variable override", f.Variable)
			}
			r, e := NewPolynomialRef(c)
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, r.Close()) })
			serviceMust(t, m.Discriminant(p, x, r))
			equal, e := c.Equal(r.root.value, want)
			serviceMust(t, e)
			if !equal {
				t.Fatal("original source ENSURE at src/test/polynomial.cpp:347", f.Line)
			}
			serviceMust(t, m.LexSort(r.root.value))
			equal, e = c.Equal(r.root.value, want)
			serviceMust(t, e)
			if !equal {
				t.Fatal("source postassert lex_sort changed value")
			}
		})
	}
}

// A bounded integer determinant is independent of the production resultant,
// derivative and discriminant implementations; dimension is at most seven.
func polynomialDiscriminantDeterminant(a [7][7]int64, n int) int64 {
	var visit func(int, uint32, int64, bool) int64
	visit = func(row int, used uint32, product int64, negative bool) int64 {
		if row == n {
			if negative {
				return -product
			}
			return product
		}
		var sum int64
		for col := 0; col < n; col++ {
			if used&(1<<col) != 0 || a[row][col] == 0 {
				continue
			}
			odd := false
			for prior := col + 1; prior < n; prior++ {
				if used&(1<<prior) != 0 {
					odd = !odd
				}
			}
			sum += visit(row+1, used|(1<<col), product*a[row][col], negative != odd)
		}
		return sum
	}
	return visit(0, 0, 1, false)
}
func TestPolynomialDiscriminantIndependentDegreeZeroThroughFour(t *testing.T) {
	for _, as := range [][]int32{{7}, {2, 3}, {5, 3, 2}, {1, 2, 0, 1}, {-1, 0, 0, 0, 1}} {
		t.Run(fmt.Sprint(len(as)-1), func(t *testing.T) {
			m := polynomialDiscriminantFixture(t)
			c := m.core
			_, e := c.monomials.MkVar()
			serviceMust(t, e)
			powers := make([][]MonomialPower, len(as))
			for j := 1; j < len(as); j++ {
				powers[j] = []MonomialPower{{0, uint32(j)}}
			}
			// The independent coefficient vector includes zeros. Do not construct
			// unowned monomial roots for terms that MkTerms will discard.
			sparseAS := []int32{}
			sparsePowers := [][]MonomialPower{}
			for j, a := range as {
				if a != 0 {
					sparseAS = append(sparseAS, a)
					sparsePowers = append(sparsePowers, powers[j])
				}
			}
			p := polynomialQueryTerms(t, c, sparseAS, sparsePowers)
			r, e := NewPolynomialRef(c)
			serviceMust(t, e)
			t.Cleanup(func() { serviceMust(t, r.Close()) })
			serviceMust(t, m.Discriminant(p, 0, r))
			degree := len(as) - 1
			if degree == 0 {
				if r.root.value != c.zero {
					t.Fatal("source degree-zero branch")
				}
				return
			}
			q := make([]int64, degree)
			for j := 1; j <= degree; j++ {
				q[j-1] = int64(j) * int64(as[j])
			}
			n := degree - 1
			var matrix [7][7]int64
			for row := 0; row < n; row++ {
				for j := 0; j <= degree; j++ {
					matrix[row][row+j] = int64(as[degree-j])
				}
			}
			for row := 0; row < degree; row++ {
				for j := 0; j <= n; j++ {
					matrix[n+row][row+j] = q[n-j]
				}
			}
			resultant := polynomialDiscriminantDeterminant(matrix, degree+n)
			if degree*(degree-1)%4 != 0 {
				resultant = -resultant
			}
			constant, e := c.IsConstant(r.root.value)
			serviceMust(t, e)
			if !constant {
				t.Fatal("bounded integer discriminant has spurious variable terms")
			}
			v, e := c.Coefficient(r.root.value, 0)
			serviceMust(t, e)
			disc, e := c.integer.Int64(&v)
			serviceMust(t, e)
			if int64(as[degree])*disc != resultant {
				t.Fatal("independent derivative/Sylvester determinant identity", disc, resultant)
			}
		})
	}
}
