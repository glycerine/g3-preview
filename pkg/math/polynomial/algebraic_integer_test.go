package polynomial

import (
	"testing"

	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestAlgebraicIntegerSetAndAdditionSourceOverloads(t *testing.T) {
	c, p, _ := algebraicControlledFixture(t)
	a, b := algebraicResultantInputs(t, c, p)
	q := unumeral.NewRational()
	n := unumeral.NewInt(0)
	var out Algebraic
	defer func() {
		serviceMust(t, c.Del(&out))
		serviceMust(t, c.Del(&b))
		serviceMust(t, c.Del(&a))
		serviceMust(t, c.config.Rationals.Del(&q))
		serviceMust(t, c.config.Rationals.Integers().Del(&n))
	}()
	for _, value := range []int64{0, 3, -3, 4294967297, -4294967297} {
		serviceMust(t, c.config.Rationals.Integers().SetInt64(&n, value))
		serviceMust(t, c.SetInteger(&out, &n))
		serviceMust(t, c.config.Rationals.SetInteger(&q, &n))
		cmp, e := c.CompareRational(&out, &q)
		serviceMust(t, e)
		if cmp != 0 {
			t.Fatal("integer set", value, cmp)
		}
		serviceMust(t, c.AddInteger(&a, &n, &out))
		serviceMust(t, c.SubRational(&out, &q, &out))
		cmp, e = c.Compare(&out, &a)
		serviceMust(t, e)
		if cmp != 0 {
			t.Fatal("root plus integer identity", value, cmp)
		}
		if c.integerAdd.active || c.integerSet.active {
			t.Fatal("integer wrapper retained scoped locals")
		}
	}
	serviceMust(t, c.config.Rationals.Integers().SetInt64(&n, -2))
	serviceMust(t, c.AddInteger(&a, &n, &a))
	algebraicExpectBetween(t, c, &a, -1, 0)
}

func TestAlgebraicIntegerAddBasicZeroAndDestinationAlias(t *testing.T) {
	c, _, _ := algebraicControlledFixture(t)
	var a Algebraic
	n := unumeral.NewInt(7)
	serviceMust(t, c.AddInteger(&a, &n, &a))
	algebraicExpectBetween(t, c, &a, 6, 8)
	n = unumeral.NewInt(-7)
	serviceMust(t, c.AddInteger(&a, &n, &a))
	if a != (Algebraic{}) {
		t.Fatal("basic integer sum did not normalize to null zero")
	}
	serviceMust(t, c.SetInteger(&a, &n))
	n = unumeral.NewInt(0)
	serviceMust(t, c.AddInteger(&a, &n, &a))
	algebraicExpectBetween(t, c, &a, -8, -6)
	serviceMust(t, c.Del(&a))
}
