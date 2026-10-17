package polynomial

import (
	"errors"
	"math"
	"strconv"
	"testing"

	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestAlgebraicDefiningPolynomialZeroBasicAndNonminimalRoot(t *testing.T) {
	c, p := algebraicFixture(t)
	out, e := p.NewPolynomial()
	serviceMust(t, e)
	defer func() { serviceMust(t, p.DelPolynomial(&out)) }()
	check := func(want ...int32) {
		t.Helper()
		view, e := p.wholeView(out)
		serviceMust(t, e)
		if view.count != uint32(len(want)) {
			t.Fatal("defining polynomial size", view.count, want)
		}
		for i, n := range want {
			v, e := p.read(view, uint32(i))
			serviceMust(t, e)
			if v.NativeScalar() != n {
				t.Fatal("defining polynomial coefficient", i, v.NativeScalar(), n)
			}
		}
	}
	var a Algebraic
	serviceMust(t, c.GetPolynomial(&a, out))
	check(0, 1)
	qm := c.config.Rationals
	q := unumeral.NewRational()
	defer func() { serviceMust(t, c.Del(&a)); serviceMust(t, qm.Del(&q)) }()
	for _, n := range []int64{3, -3} {
		serviceMust(t, qm.SetFraction(&q, n, 2))
		serviceMust(t, c.SetRational(&a, &q))
		serviceMust(t, c.GetPolynomial(&a, out))
		check(-int32(n), 2)
	}
	poly := algebraicPolynomial(t, p, 6, 0, -5, 0, 1) // (x^2-2)(x^2-3)
	view, e := p.wholeView(poly)
	serviceMust(t, e)
	b := c.config.Binary
	var lower, upper unumeral.BinaryRational
	defer func() { serviceMust(t, b.Del(&lower)); serviceMust(t, b.Del(&upper)) }()
	serviceMust(t, b.SetInt32(&lower, 1))
	serviceMust(t, b.SetInt32Power(&upper, 3, 1))
	serviceMust(t, c.SetRoot(&a, view, &lower, &upper, false))
	serviceMust(t, c.GetPolynomial(&a, out))
	check(6, 0, -5, 0, 1)
	// The output is an owning copy, not an alias of the stored root polynomial.
	changed := unumeral.NewInt(7)
	serviceMust(t, p.SetCoefficient(out, 0, &changed))
	serviceMust(t, c.GetPolynomial(&a, out))
	check(6, 0, -5, 0, 1)
	serviceMust(t, c.Del(&a))
	check(6, 0, -5, 0, 1)
	serviceMust(t, c.GetPolynomial(&a, out))
	check(0, 1)
}

func TestAlgebraicBoundsExactAndCopiedPrecision(t *testing.T) {
	c, p := algebraicFixture(t)
	poly := algebraicPolynomial(t, p, -2, 0, 1)
	v, e := p.wholeView(poly)
	serviceMust(t, e)
	b, qm := c.config.Binary, c.config.Rationals
	var lower, upper unumeral.BinaryRational
	serviceMust(t, b.SetInt32(&lower, 1))
	serviceMust(t, b.SetInt32(&upper, 2))
	var a Algebraic
	serviceMust(t, c.SetRoot(&a, v, &lower, &upper, true))
	defer func() { serviceMust(t, c.Del(&a)); serviceMust(t, b.Del(&upper)); serviceMust(t, b.Del(&lower)) }()
	l, e := c.location(&a)
	serviceMust(t, e)
	before, e := c.load(l)
	serviceMust(t, e)
	lo, hi := unumeral.NewRational(), unumeral.NewRational()
	width, tolerance := unumeral.NewRational(), unumeral.NewRational()
	defer func() {
		for _, q := range []*unumeral.Rational{&lo, &hi, &width, &tolerance} {
			serviceMust(t, qm.Del(q))
		}
	}()
	serviceMust(t, c.GetLower(&a, &lower))
	serviceMust(t, c.GetUpper(&a, &upper))
	serviceMust(t, c.GetLowerRational(&a, &lo))
	serviceMust(t, c.GetUpperRational(&a, &hi))
	if rationalString(t, qm, &lo) != "1" || rationalString(t, qm, &hi) != "2" {
		t.Fatal("nonprecision getters changed the stored endpoints")
	}
	serviceMust(t, c.GetLowerRationalAtPrecision(&a, &lo, 8))
	serviceMust(t, c.GetUpperRationalAtPrecision(&a, &hi, 8))
	sl, e := p.EvalSignRational(v, &lo)
	serviceMust(t, e)
	su, e := p.EvalSignRational(v, &hi)
	serviceMust(t, e)
	if sl != -1 || su != 1 {
		t.Fatal("refined bounds do not surround sqrt(2)", sl, su)
	}
	serviceMust(t, qm.Sub(&hi, &lo, &width))
	serviceMust(t, qm.SetFraction(&tolerance, 1, uint64(1)<<32))
	cmp, e := qm.Compare(&width, &tolerance)
	serviceMust(t, e)
	if cmp >= 0 {
		t.Fatal("decimal precision was not converted to four binary bits per digit")
	}
	after, e := c.load(l)
	serviceMust(t, e)
	if before != after {
		t.Fatal("precision getter mutated the stored root")
	}
	serviceMust(t, c.Del(&a))
	serviceMust(t, c.GetLowerRationalAtPrecision(&a, &lo, 8))
	serviceMust(t, c.GetUpperRationalAtPrecision(&a, &hi, 8))
	if rationalString(t, qm, &lo) != "0" || rationalString(t, qm, &hi) != "0" {
		t.Fatal("zero bounds")
	}
	serviceMust(t, qm.SetFraction(&lo, -3, 2))
	serviceMust(t, c.SetRational(&a, &lo))
	serviceMust(t, c.GetLowerRationalAtPrecision(&a, &lo, 8))
	serviceMust(t, c.GetUpperRationalAtPrecision(&a, &hi, 8))
	if rationalString(t, qm, &lo) != "-3/2" || rationalString(t, qm, &hi) != "-3/2" {
		t.Fatal("basic bounds were rounded")
	}
}

func TestAlgebraicDefiningPolynomialMinimumIntegerReservationPrefixes(t *testing.T) {
	succeeded, refused, retained := false, false, false
	for prefix := int64(0); prefix < 64 && !succeeded; prefix++ {
		t.Run(strconv.FormatInt(prefix, 10), func(t *testing.T) {
			c, p := algebraicCellSourceFixture(t)
			qm := c.config.Rationals
			q := unumeral.NewRational()
			serviceMust(t, qm.SetInt32(&q, math.MinInt32))
			var a Algebraic
			serviceMust(t, c.SetRational(&a, &q))
			out, e := p.NewPolynomial()
			serviceMust(t, e)
			qm.Budget().FailAfter(prefix)
			body := c.GetPolynomial(&a, out)
			algebraicCellSourcePhysical(t, c, []UPolynomial{out}, nil, nil, &a)
			qm.Budget().FailAfter(-1)
			if body != nil {
				refused = true
				if !numericPhysicalReservationRefusal(body) {
					t.Fatal(body)
				}
			} else {
				succeeded = true
				view, e := p.wholeView(out)
				serviceMust(t, e)
				first, e := p.read(view, 0)
				serviceMust(t, e)
				text, e := qm.Integers().String(first)
				serviceMust(t, e)
				if text != "2147483648" {
					t.Fatal("negated minimum numerator", text)
				}
			}
			if unumeral.NumericAllocationPending(&c.representationNeg.roots) {
				retained = true
				if e := c.GetPolynomial(&a, out); !errors.Is(e, ErrAlgebraicPending) {
					t.Fatal("pending negation root lost replacement guard", e)
				}
			}
			serviceMust(t, c.CleanupRetained())
			serviceMust(t, p.Reset(out))
			serviceMust(t, c.GetPolynomial(&a, out))
			retry, e := p.wholeView(out)
			serviceMust(t, e)
			first, e := p.read(retry, 0)
			serviceMust(t, e)
			text, e := qm.Integers().String(first)
			serviceMust(t, e)
			if text != "2147483648" {
				t.Fatal("negation retry coefficient", text)
			}

			serviceMust(t, c.Del(&a))
			serviceMust(t, p.DelPolynomial(&out))
			serviceMust(t, qm.Del(&q))
		})
	}
	if !succeeded || !refused || !retained {
		t.Fatal("missing defining-polynomial allocation frontier", succeeded, refused, retained)
	}
}

func TestAlgebraicBoundsDiscoveredRootKeepsSourceUpperEndpoint(t *testing.T) {
	c, p := algebraicFixture(t)
	poly := algebraicPolynomial(t, p, -1, 0, 1)
	v, e := p.wholeView(poly)
	serviceMust(t, e)
	b, qm := c.config.Binary, c.config.Rationals
	var lower, upper unumeral.BinaryRational
	serviceMust(t, b.SetInt32(&upper, 2))
	var a Algebraic
	serviceMust(t, c.SetRoot(&a, v, &lower, &upper, false))
	lo, hi := unumeral.NewRational(), unumeral.NewRational()
	defer func() {
		serviceMust(t, c.Del(&a))
		serviceMust(t, qm.Del(&hi))
		serviceMust(t, qm.Del(&lo))
		serviceMust(t, b.Del(&upper))
		serviceMust(t, b.Del(&lower))
	}()
	serviceMust(t, c.GetLowerRationalAtPrecision(&a, &lo, 8))
	serviceMust(t, c.GetUpperRationalAtPrecision(&a, &hi, 8))
	// refine writes the discovered root only to its lower argument. Unlike
	// display_decimal, get_upper ignores refine's false return and uses upper.
	if rationalString(t, qm, &lo) != "1" || rationalString(t, qm, &hi) != "2" {
		t.Fatal("precision getter changed the source exact-root branch")
	}
	l, e := c.location(&a)
	serviceMust(t, e)
	if l.kind != 1 {
		t.Fatal("a copied-interval query converted the stored root to a basic cell")
	}
}
