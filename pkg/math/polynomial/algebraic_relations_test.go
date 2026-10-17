package polynomial

import (
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

func TestAlgebraicRelationsIndependentValues(t *testing.T) {
	c, _, _ := algebraicCompareFixture(t)
	// Independent integer cross products establish the order of rational inputs.
	for n := int64(-3); n <= 3; n++ {
		for d := int64(1); d <= 3; d++ {
			for k := int64(-3); k <= 3; k++ {
				var a, b Algebraic
				q, r := unumeral.NewRational(), unumeral.NewRational()
				serviceMust(t, c.config.Rationals.SetFraction(&q, n, uint64(d)))
				serviceMust(t, c.config.Rationals.SetFraction(&r, k, 2))
				serviceMust(t, c.SetRational(&a, &q))
				serviceMust(t, c.SetRational(&b, &r))
				eq, e := c.Equal(&a, &b)
				serviceMust(t, e)
				lt, e := c.Less(&a, &b)
				serviceMust(t, e)
				if eq != (2*n == k*d) || lt != (2*n < k*d) {
					t.Fatal(n, d, k, eq, lt)
				}
				serviceMust(t, c.Del(&b))
				serviceMust(t, c.Del(&a))
				serviceMust(t, c.config.Rationals.Del(&r))
				serviceMust(t, c.config.Rationals.Del(&q))
			}
		}
	}
	// Two separately allocated defining polynomials describe the same root.
	a := algebraicCompareRoot(t, c, []int32{-2, 0, 1}, 1, 0, 2, 0, true)
	b := algebraicCompareRoot(t, c, []int32{-2, 0, 1}, 1, 0, 2, 0, true)
	z := Algebraic{}
	eq, e := c.Equal(&a, &b)
	serviceMust(t, e)
	if !eq {
		t.Fatal("separately allocated sqrt(2) values differ")
	}
	lt, e := c.Less(&z, &a)
	serviceMust(t, e)
	if !lt {
		t.Fatal("sqrt(2) must be positive")
	}
	lt, e = c.Less(&a, &z)
	serviceMust(t, e)
	if lt {
		t.Fatal("sqrt(2) below zero")
	}
	serviceMust(t, c.Del(&b))
	serviceMust(t, c.Del(&a))
	// Errors must never become an affirmative relation.
	other, _, _ := algebraicCompareFixture(t)
	var foreign Algebraic
	one := unumeral.NewInt(1)
	serviceMust(t, other.SetInteger(&foreign, &one))
	if got, e := c.Equal(&foreign, &z); e == nil || got {
		t.Fatal("foreign equality", got, e)
	}
	if got, e := c.Less(&foreign, &z); e == nil || got {
		t.Fatal("foreign order", got, e)
	}
	serviceMust(t, other.Del(&foreign))
}
