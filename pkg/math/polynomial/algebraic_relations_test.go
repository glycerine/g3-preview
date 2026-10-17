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

func TestAlgebraicRelationsRationalDispatch(t *testing.T) {
	c, _, control := algebraicCompareFixture(t)
	for n := int64(-3); n <= 3; n++ {
		for d := int64(1); d <= 3; d++ {
			var a Algebraic
			q := unumeral.NewRational()
			serviceMust(t, c.config.Rationals.SetFraction(&q, n, uint64(d)))
			serviceMust(t, c.SetRational(&a, &q))
			for k := int64(-3); k <= 3; k++ {
				b := unumeral.NewRational()
				serviceMust(t, c.config.Rationals.SetFraction(&b, k, 2))
				eq, e := c.EqualRational(&a, &b)
				serviceMust(t, e)
				lt, e := c.LessRational(&a, &b)
				serviceMust(t, e)
				gt, e := c.GreaterRational(&a, &b)
				serviceMust(t, e)
				if eq != (2*n == k*d) || lt != (2*n < k*d) || gt != (2*n > k*d) {
					t.Fatal(n, d, k, eq, lt, gt)
				}
				serviceMust(t, c.config.Rationals.Del(&b))
			}
			serviceMust(t, c.Del(&a))
			serviceMust(t, c.config.Rationals.Del(&q))
		}
	}
	// Raw mpq equality compares numerator and denominator, not cross products.
	var one Algebraic
	unit := unumeral.NewInt(1)
	serviceMust(t, c.SetInteger(&one, &unit))
	raw := unumeral.Rational{Num: unumeral.NewInt(2), Den: unumeral.NewInt(2)}
	eq, e := c.EqualRational(&one, &raw)
	serviceMust(t, e)
	lt, e := c.LessRational(&one, &raw)
	serviceMust(t, e)
	gt, e := c.GreaterRational(&one, &raw)
	serviceMust(t, e)
	if eq || lt || gt {
		t.Fatal("raw equality must preserve source representation", eq, lt, gt)
	}
	serviceMust(t, c.Del(&one))
	root := algebraicCompareRoot(t, c, []int32{-2, 0, 1}, 1, 0, 2, 0, true)
	defer func() { serviceMust(t, c.Del(&root)) }()
	for _, v := range []struct {
		n    int64
		d    uint64
		less bool
	}{{1, 1, false}, {3, 2, true}, {2, 1, true}} {
		b := unumeral.NewRational()
		serviceMust(t, c.config.Rationals.SetFraction(&b, v.n, v.d))
		eq, e := c.EqualRational(&root, &b)
		serviceMust(t, e)
		lt, e := c.LessRational(&root, &b)
		serviceMust(t, e)
		gt, e := c.GreaterRational(&root, &b)
		serviceMust(t, e)
		if eq || lt != v.less || gt == v.less {
			t.Fatal(v, eq, lt, gt)
		}
		serviceMust(t, c.config.Rationals.Del(&b))
	}
	if control.checks != 0 {
		t.Fatal("mixed comparison unexpectedly refined roots", control.checks)
	}
}
