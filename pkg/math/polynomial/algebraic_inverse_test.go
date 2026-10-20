package polynomial

import (
	"errors"
	"math"
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestAlgebraicInverseBasicAndZero(t *testing.T) {
	c, _ := algebraicFixture(t)
	polynomialAssertionActionsFixture(t, "C")
	var a Algebraic
	want := error(diagnostic.ErrDebugNativeUnreachable)
	if polynomialSourceDebug {
		want = ErrAlgebraicDivisionByZero
	}
	if e := c.Inv(&a); !errors.Is(e, want) || a != (Algebraic{}) {
		t.Fatal("zero inverse", e, a)
	}
	qm := c.config.Rationals
	q := unumeral.NewRational()
	defer func() { serviceMust(t, qm.Del(&q)); serviceMust(t, c.Del(&a)) }()
	for _, n := range []int64{3, -3} {
		serviceMust(t, qm.SetFraction(&q, n, 2))
		serviceMust(t, c.SetRational(&a, &q))
		original := a
		qm.Budget().FailAfter(0)
		e := c.Inv(&a)
		qm.Budget().FailAfter(-1)
		serviceMust(t, e)
		if a != original {
			t.Fatal("basic inverse replaced the cell")
		}
		serviceMust(t, qm.InvInPlace(&q))
		cmp, e := c.CompareRational(&a, &q)
		serviceMust(t, e)
		if cmp != 0 {
			t.Fatal("basic inverse value", n, cmp)
		}
	}
}

func TestAlgebraicInverseMinimumIntegerReservationPrefixes(t *testing.T) {
	succeeded, refused := false, false
	for prefix := int64(0); prefix < 64 && !succeeded; prefix++ {
		t.Run(strconv.FormatInt(prefix, 10), func(t *testing.T) {
			c, _ := algebraicCellSourceFixture(t)
			qm := c.config.Rationals
			q := unumeral.NewRational()
			serviceMust(t, qm.SetInt32(&q, math.MinInt32))
			var a Algebraic
			serviceMust(t, c.SetRational(&a, &q))
			qm.Budget().FailAfter(prefix)
			body := c.Inv(&a)
			algebraicCellSourcePhysical(t, c, nil, nil, nil, &a)
			qm.Budget().FailAfter(-1)
			if body != nil {
				refused = true
				if !numericPhysicalReservationRefusal(body) {
					t.Fatal("inverse failed outside allocation", body)
				}
			} else {
				succeeded = true
				l, e := c.location(&a)
				serviceMust(t, e)
				r, e := c.load(l)
				serviceMust(t, e)
				text, e := qm.String(&r.value)
				serviceMust(t, e)
				if text != "-1/2147483648" {
					t.Fatal("minimum integer inverse", text)
				}
			}
			serviceMust(t, c.CleanupRetained())
			algebraicExpectNoPendingAllocation(t, qm.Integers())
			serviceMust(t, c.Del(&a))
			serviceMust(t, qm.Del(&q))
		})
	}
	if !succeeded || !refused {
		t.Fatal("inverse allocation frontier was not covered", succeeded, refused)
	}
}

func TestAlgebraicInverseRootAndZeroBoundRefinement(t *testing.T) {
	for _, tc := range []struct {
		name         string
		coefficients []int32
		lower, upper int32
		rational     bool
	}{
		{"sqrt2", []int32{-2, 0, 1}, 1, 2, false},
		{"sqrt5-nondyadic-reciprocal-bounds", []int32{-5, 0, 1}, 2, 3, false},
		{"sqrt2-zero-lower", []int32{-2, 0, 1}, 0, 2, false},
		{"negative-sqrt2-zero-upper", []int32{-2, 0, 1}, -2, 0, false},
		{"cube2", []int32{-2, 0, 0, 1}, 0, 2, false},
		{"negative-cube2", []int32{2, 0, 0, 1}, -2, 0, false},
		{"small-root-multiple-halvings", []int32{-1, 0, 8}, 0, 2, false},
		{"negative-small-root-multiple-halvings", []int32{-1, 0, 8}, -2, 0, false},
		{"rational-root-discovered", []int32{-1, 0, 1}, 0, 2, true},
		{"negative-rational-root-discovered", []int32{-1, 0, 1}, -2, 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, p := algebraicFixture(t)
			poly := algebraicPolynomial(t, p, tc.coefficients...)
			v, e := p.wholeView(poly)
			serviceMust(t, e)
			b := c.config.Binary
			var lower, upper unumeral.BinaryRational
			serviceMust(t, b.SetInt32(&lower, tc.lower))
			serviceMust(t, b.SetInt32(&upper, tc.upper))
			var a, original Algebraic
			defer func() {
				serviceMust(t, c.Del(&original))
				serviceMust(t, c.Del(&a))
				serviceMust(t, b.Del(&upper))
				serviceMust(t, b.Del(&lower))
			}()
			serviceMust(t, c.SetRoot(&a, v, &lower, &upper, false))
			serviceMust(t, c.Copy(&original, &a))
			before := a
			serviceMust(t, c.Inv(&a))
			l, e := c.location(&a)
			serviceMust(t, e)
			r, e := c.load(l)
			serviceMust(t, e)
			if tc.rational {
				if l.kind != 0 {
					t.Fatal("halving failed to discover an exact rational root")
				}
			} else {
				if a != before || l.kind != 1 {
					t.Fatal("algebraic inverse replaced its root cell")
				}
				view, e := c.sourcePolynomial(&r)
				serviceMust(t, e)
				for i := uint32(0); i < view.count; i++ {
					value, e := p.read(view, i)
					serviceMust(t, e)
					if value.NativeScalar() != tc.coefficients[len(tc.coefficients)-1-int(i)] {
						t.Fatal("reciprocal coefficient", i, value.NativeScalar())
					}
				}
				sl, e := p.EvalSignBinary(view, r.interval.BorrowLower())
				serviceMust(t, e)
				su, e := p.EvalSignBinary(view, r.interval.BorrowUpper())
				serviceMust(t, e)
				if sl == 0 || sl != -su {
					t.Fatal("reciprocal interval does not isolate the root", sl, su)
				}
			}
			// A second reciprocal restores the source polynomial and value.
			serviceMust(t, c.Inv(&a))
			cmp, e := c.Compare(&a, &original)
			serviceMust(t, e)
			if cmp != 0 {
				t.Fatal("double inverse changed the value", cmp)
			}
		})
	}
}
