package polynomial

import (
	"errors"
	"fmt"
	"math"
	"testing"

	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestAlgebraicRationalTranslationBinaryNonbinaryAndAlias(t *testing.T) {
	for _, negative := range []bool{false, true} {
		for _, alias := range []bool{false, true} {
			for _, tc := range []struct {
				n int64
				d uint64
			}{{1, 1}, {1, 2}, {-1, 2}, {1, 3}, {-3, 2}} {
				t.Run(fmt.Sprint(negative, alias, tc.n, tc.d), func(t *testing.T) {
					c, p := algebraicFixture(t)
					poly := algebraicPolynomial(t, p, -2, 0, 1)
					v, e := p.wholeView(poly)
					serviceMust(t, e)
					bm, qm := c.config.Binary, c.config.Rationals
					var lower, upper unumeral.BinaryRational
					lo, hi := int32(1), int32(2)
					if negative {
						lo, hi = -2, -1
					}
					serviceMust(t, bm.SetInt32(&lower, lo))
					serviceMust(t, bm.SetInt32(&upper, hi))
					var a, original, result Algebraic
					serviceMust(t, c.SetRoot(&a, v, &lower, &upper, true))
					serviceMust(t, c.Copy(&original, &a))
					q := unumeral.NewRational()
					serviceMust(t, qm.SetFraction(&q, tc.n, tc.d))
					out := &result
					if alias {
						out = &a
					}
					defer func() {
						serviceMust(t, c.Del(&result))
						serviceMust(t, c.Del(&original))
						serviceMust(t, c.Del(&a))
						serviceMust(t, qm.Del(&q))
						serviceMust(t, bm.Del(&upper))
						serviceMust(t, bm.Del(&lower))
					}()
					serviceMust(t, c.AddRational(&a, &q, out))
					l, e := c.location(out)
					serviceMust(t, e)
					r, e := c.load(l)
					serviceMust(t, e)
					translated, e := c.sourcePolynomial(&r)
					serviceMust(t, e)
					// d^2*(x-n/d)^2-2*d^2, independent of the interval route.
					n, d := int32(tc.n), int32(tc.d)
					want := []int32{n*n - 2*d*d, -2 * n * d, d * d}
					for i, expected := range want {
						value, e := p.read(translated, uint32(i))
						serviceMust(t, e)
						if value.NativeScalar() != expected {
							t.Fatal("translated coefficient", i, value.NativeScalar(), expected)
						}
					}
					if r.flags&1 == 0 {
						t.Fatal("translation lost source minimality")
					}
					positive, e := c.IsPos(out)
					serviceMust(t, e)
					wantPositive := !negative && tc.n != -3
					if positive != wantPositive {
						t.Fatal("translated interval was not normalized across zero", positive)
					}
					serviceMust(t, c.SubRational(out, &q, out))
					cmp, e := c.Compare(out, &original)
					serviceMust(t, e)
					if cmp != 0 {
						t.Fatal("rational translation round trip", cmp)
					}
				})
			}
		}
	}
}

func TestAlgebraicRationalArithmeticBasicAndNormalizedZero(t *testing.T) {
	c, p := algebraicFixture(t)
	qm, bm := c.config.Rationals, c.config.Binary
	q := unumeral.NewRational()
	var a Algebraic
	defer func() { serviceMust(t, c.Del(&a)); serviceMust(t, qm.Del(&q)) }()
	serviceMust(t, qm.SetFraction(&q, 3, 2))
	serviceMust(t, c.AddRational(&a, &q, &a))
	cmp, e := c.CompareRational(&a, &q)
	serviceMust(t, e)
	if cmp != 0 {
		t.Fatal("zero plus rational")
	}
	serviceMust(t, c.SubRational(&a, &q, &a))
	if a != (Algebraic{}) {
		t.Fatal("basic cancellation did not normalize to null")
	}
	poly := algebraicPolynomial(t, p, -1, 0, 1)
	v, e := p.wholeView(poly)
	serviceMust(t, e)
	var lower, upper unumeral.BinaryRational
	serviceMust(t, bm.SetInt32(&upper, 2))
	defer func() { serviceMust(t, bm.Del(&upper)); serviceMust(t, bm.Del(&lower)) }()
	serviceMust(t, c.SetRoot(&a, v, &lower, &upper, false))
	serviceMust(t, qm.SetInt32(&q, -1))
	serviceMust(t, c.AddRational(&a, &q, &a))
	if a != (Algebraic{}) {
		t.Fatal("translated zero root did not normalize to null")
	}
}

func TestAlgebraicRationalTranslationCancellationKeepsCallerValues(t *testing.T) {
	c, p, control := algebraicControlledFixture(t)
	poly := algebraicPolynomial(t, p, -2, 0, 1)
	v, e := p.wholeView(poly)
	serviceMust(t, e)
	bm, qm := c.config.Binary, c.config.Rationals
	var lower, upper unumeral.BinaryRational
	serviceMust(t, bm.SetInt32(&lower, 1))
	serviceMust(t, bm.SetInt32(&upper, 2))
	var a, out Algebraic
	serviceMust(t, c.SetRoot(&a, v, &lower, &upper, true))
	q := unumeral.NewRational()
	serviceMust(t, qm.SetInt32(&q, 7))
	serviceMust(t, c.SetRational(&out, &q))
	serviceMust(t, qm.SetFraction(&q, 1, 3))
	defer func() {
		serviceMust(t, c.Del(&out))
		serviceMust(t, c.Del(&a))
		serviceMust(t, qm.Del(&q))
		serviceMust(t, bm.Del(&upper))
		serviceMust(t, bm.Del(&lower))
	}()
	control.limit = control.visits
	body := c.AddRational(&a, &q, &out)
	control.limit = 1000000
	if !errors.Is(body, ErrUPolynomialCanceled) {
		t.Fatal("translation cancellation", body)
	}
	serviceMust(t, qm.SetInt32(&q, 7))
	cmp, e := c.CompareRational(&out, &q)
	serviceMust(t, e)
	if cmp != 0 {
		t.Fatal("cancelled translation changed the output")
	}
	if c.arithmetic.active {
		t.Fatal("source locals survived completed cleanup")
	}
	serviceMust(t, qm.SetFraction(&q, 1, 3))
	serviceMust(t, c.AddRational(&a, &q, &out))
}

func TestAlgebraicRationalZeroSubtractionPublishesBeforeNegation(t *testing.T) {
	c, _ := algebraicCellSourceFixture(t)
	qm := c.config.Rationals
	q := unumeral.NewRational()
	serviceMust(t, qm.SetInt32(&q, 7))
	var zero, out Algebraic
	serviceMust(t, c.SetRational(&out, &q))
	serviceMust(t, qm.SetInt32(&q, math.MinInt32))
	qm.Budget().FailAfter(0)
	body := c.SubRational(&zero, &q, &out)
	algebraicCellSourcePhysical(t, c, nil, nil, nil, &out)
	qm.Budget().FailAfter(-1)
	if !numericPhysicalReservationRefusal(body) {
		t.Fatal("minimum-integer negation did not stop", body)
	}
	cmp, e := c.CompareRational(&out, &q)
	serviceMust(t, e)
	if cmp != 0 {
		t.Fatal("zero subtraction did not preserve its published input on negation failure")
	}
	receipt := algebraicNegInvocationReceipt{c, &c.negation, c.negation.invocation}
	serviceMust(t, c.cleanupNegation(&receipt, false))
	serviceMust(t, c.Del(&out))
	serviceMust(t, c.CleanupRetained())
	serviceMust(t, qm.Del(&q))
}
