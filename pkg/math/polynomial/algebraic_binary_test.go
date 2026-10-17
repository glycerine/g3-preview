package polynomial

import (
	"errors"
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/z3exception"
)

func algebraicExpectBetween(t *testing.T, c *AlgebraicRootCore, a *Algebraic, lower, upper int32) {
	t.Helper()
	q := unumeral.NewRational()
	defer func() { serviceMust(t, c.config.Rationals.Del(&q)) }()
	serviceMust(t, c.config.Rationals.SetInt32(&q, lower))
	l, e := c.CompareRational(a, &q)
	serviceMust(t, e)
	serviceMust(t, c.config.Rationals.SetInt32(&q, upper))
	u, e := c.CompareRational(a, &q)
	serviceMust(t, e)
	if l <= 0 || u >= 0 {
		t.Fatal("algebraic value outside independent rational bounds", l, u, lower, upper)
	}
}

func TestAlgebraicBinaryEveryReachedCancellationAndRetry(t *testing.T) {
	for _, factor := range []bool{false, true} {
		t.Run(map[bool]string{false: "square-free", true: "factor"}[factor], func(t *testing.T) {
			succeeded := false
			for allowance := uint64(0); allowance < 512 && !succeeded; allowance++ {
				t.Run(strconv.FormatUint(allowance, 10), func(t *testing.T) {
					c, p, control := algebraicControlledFixture(t)
					c.factor = factor
					a, b := algebraicResultantInputs(t, c, p)
					q := unumeral.NewRational()
					serviceMust(t, c.config.Rationals.SetInt32(&q, 17))
					var out Algebraic
					serviceMust(t, c.SetRational(&out, &q))
					defer func() {
						serviceMust(t, c.Del(&out))
						serviceMust(t, c.Del(&b))
						serviceMust(t, c.Del(&a))
						serviceMust(t, c.config.Rationals.Del(&q))
					}()
					control.limit = control.visits + allowance
					body := c.Add(&a, &b, &out)
					control.limit = 1000000
					if body == nil {
						succeeded = true
						algebraicExpectBetween(t, c, &out, 3, 4)
						return
					}
					if !errors.Is(body, ErrUPolynomialCanceled) {
						t.Fatal("noncancellation failure", body)
					}
					cmp, e := c.CompareRational(&out, &q)
					serviceMust(t, e)
					if cmp != 0 {
						t.Fatal("canceled selector changed the prior output", cmp)
					}
					if !errors.Is(c.Add(&a, &b, &out), ErrAlgebraicPending) {
						t.Fatal("stopped binary scope was overwritten")
					}
					serviceMust(t, c.CleanupRetained())
					serviceMust(t, c.Add(&a, &b, &out))
					algebraicExpectBetween(t, c, &out, 3, 4)
				})
			}
			if !succeeded {
				t.Fatal("binary operation never passed its cancellation frontier")
			}
		})
	}
}

func TestAlgebraicBinaryRootArithmeticAndAliases(t *testing.T) {
	for _, factor := range []bool{false, true} {
		t.Run(map[bool]string{false: "square-free", true: "factor"}[factor], func(t *testing.T) {
			c, p := algebraicFixture(t)
			c.factor = factor
			a, b := algebraicResultantInputs(t, c, p)
			var sum, product, difference, temp Algebraic
			defer func() {
				for _, value := range []*Algebraic{&temp, &difference, &product, &sum, &b, &a} {
					serviceMust(t, c.Del(value))
				}
			}()
			serviceMust(t, c.Add(&a, &b, &sum))
			algebraicExpectBetween(t, c, &sum, 3, 4)
			serviceMust(t, c.Sub(&a, &b, &difference))
			algebraicExpectBetween(t, c, &difference, -1, 0)
			serviceMust(t, c.Mul(&a, &b, &product))
			algebraicExpectBetween(t, c, &product, 2, 3)
			serviceMust(t, c.Sub(&sum, &b, &sum))
			cmp, e := c.Compare(&sum, &a)
			serviceMust(t, e)
			if cmp != 0 {
				t.Fatal("(sqrt2+sqrt3)-sqrt3 != sqrt2", cmp)
			}
			serviceMust(t, c.Neg(&b))
			serviceMust(t, c.Add(&a, &b, &difference))
			algebraicExpectBetween(t, c, &difference, -1, 0)
			serviceMust(t, c.Mul(&a, &b, &product))
			algebraicExpectBetween(t, c, &product, -3, -2)
			serviceMust(t, c.Sub(&a, &b, &b))
			algebraicExpectBetween(t, c, &b, 3, 4)
			serviceMust(t, c.Copy(&temp, &a))
			serviceMust(t, c.Mul(&temp, &temp, &temp))
			two := unumeral.NewRational()
			serviceMust(t, c.config.Rationals.SetInt32(&two, 2))
			cmp, e = c.CompareRational(&temp, &two)
			serviceMust(t, e)
			serviceMust(t, c.config.Rationals.Del(&two))
			if cmp != 0 {
				t.Fatal("aliased sqrt2 squared != 2", cmp)
			}
			serviceMust(t, c.Sub(&a, &a, &a))
			if a != (Algebraic{}) {
				t.Fatal("aliased root subtraction did not produce source zero")
			}
		})
	}
}

func TestAlgebraicBinaryQuadraticCubicRootSelection(t *testing.T) {
	c, p := algebraicFixture(t)
	a, b := algebraicResultantInputs(t, c, p)
	cubic := algebraicPolynomial(t, p, -3, 0, 0, 1)
	v, e := p.wholeView(cubic)
	serviceMust(t, e)
	var lower, upper unumeral.BinaryRational
	serviceMust(t, c.config.Binary.SetInt32(&lower, 1))
	serviceMust(t, c.config.Binary.SetInt32(&upper, 2))
	serviceMust(t, c.SetRoot(&b, v, &lower, &upper, true))
	serviceMust(t, c.config.Binary.Del(&upper))
	serviceMust(t, c.config.Binary.Del(&lower))
	var out Algebraic
	defer func() { serviceMust(t, c.Del(&out)); serviceMust(t, c.Del(&b)); serviceMust(t, c.Del(&a)) }()
	serviceMust(t, c.Add(&a, &b, &out))
	algebraicExpectBetween(t, c, &out, 2, 3)
	serviceMust(t, c.Sub(&a, &b, &out))
	algebraicExpectBetween(t, c, &out, -1, 0)
	serviceMust(t, c.Mul(&a, &b, &out))
	algebraicExpectBetween(t, c, &out, 2, 3)
}

func TestAlgebraicBinaryMixedDispatchAndRefinementFallback(t *testing.T) {
	c, p := algebraicFixture(t)
	a, b := algebraicResultantInputs(t, c, p)
	q := unumeral.NewRational()
	var basic, out Algebraic
	defer func() {
		for _, value := range []*Algebraic{&out, &basic, &b, &a} {
			serviceMust(t, c.Del(value))
		}
		serviceMust(t, c.config.Rationals.Del(&q))
	}()
	serviceMust(t, c.config.Rationals.SetInt32(&q, 2))
	serviceMust(t, c.SetRational(&basic, &q))
	serviceMust(t, c.Sub(&basic, &a, &out))
	algebraicExpectBetween(t, c, &out, 0, 1)
	serviceMust(t, c.Mul(&basic, &a, &out))
	algebraicExpectBetween(t, c, &out, 2, 3)
	var zero Algebraic
	serviceMust(t, c.Mul(&zero, nil, &out))
	if out != (Algebraic{}) {
		t.Fatal("zero product")
	}
	// Wide intervals force refinement; the first input becomes exactly 1.
	poly := algebraicPolynomial(t, p, -1, 0, 1)
	v, e := p.wholeView(poly)
	serviceMust(t, e)
	var lower, upper unumeral.BinaryRational
	serviceMust(t, c.config.Binary.SetInt32(&upper, 2))
	serviceMust(t, c.SetRoot(&a, v, &lower, &upper, false))
	poly2 := algebraicPolynomial(t, p, -2, 0, 1)
	v, e = p.wholeView(poly2)
	serviceMust(t, e)
	serviceMust(t, c.SetRoot(&b, v, &lower, &upper, true))
	serviceMust(t, c.config.Binary.Del(&upper))
	serviceMust(t, c.config.Binary.Del(&lower))
	serviceMust(t, c.Add(&a, &b, &b))
	serviceMust(t, c.config.Rationals.SetInt32(&q, 1))
	cmp, e := c.CompareRational(&a, &q)
	serviceMust(t, e)
	if cmp != 0 {
		t.Fatal("refinement did not discover the rational input", cmp)
	}
	algebraicExpectBetween(t, c, &b, 2, 3)
}

func TestAlgebraicBinaryRestoresOverrefinedInputInterval(t *testing.T) {
	c, p := algebraicFixture(t)
	c.minMagnitude = 0 // source min_mag=0
	a, b := algebraicResultantInputs(t, c, p)
	var out Algebraic
	defer func() { serviceMust(t, c.Del(&out)); serviceMust(t, c.Del(&b)); serviceMust(t, c.Del(&a)) }()
	l, e := c.location(&a)
	serviceMust(t, e)
	before, e := c.load(l)
	serviceMust(t, e)
	// Initially (-1,1) contains both small roots of x^4-10*x^2+1.
	// Selection must refine; the source magnitude threshold then restores a.
	serviceMust(t, c.Sub(&a, &b, &out))
	after, e := c.load(l)
	serviceMust(t, e)
	if before != after {
		t.Fatal("source saved_a did not restore the overrefined interval")
	}
	algebraicExpectBetween(t, c, &out, -1, 0)
}

func TestAlgebraicBinaryEntryAssertionPrecedesLocalOwners(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source assertions are erased")
	}
	c, p := algebraicFixture(t)
	a, b := algebraicResultantInputs(t, c, p)
	q := unumeral.NewRational()
	serviceMust(t, c.config.Rationals.SetInt32(&q, 7))
	var out Algebraic
	serviceMust(t, c.SetRational(&out, &q))
	defer func() {
		serviceMust(t, c.Del(&out))
		serviceMust(t, c.Del(&b))
		serviceMust(t, c.Del(&a))
		serviceMust(t, c.config.Rationals.Del(&q))
	}()
	budget := arena.NewBudget(1 << 20)
	region, e := arena.NewBytesRegion(arena.BytesRegionConfig{DirectOnly: true, Descriptors: arena.Config{Budget: budget}})
	serviceMust(t, e)
	d, e := diagnostic.NewDebugContext(region, budget)
	serviceMust(t, e)
	previous, e := diagnostic.BindProcessDebugContext(d)
	serviceMust(t, e)
	serviceMust(t, d.Close())
	body := c.Add(&a, &b, &out)
	_, e = diagnostic.BindProcessDebugContext(previous)
	serviceMust(t, e)
	serviceMust(t, region.Close())
	if !errors.Is(body, arena.ErrClosed) || c.binary.active || c.multivariate != nil {
		t.Fatal("closed source assertion owner was read after local construction", body)
	}
	polynomialThrowAssertionFixture(t)
	serviceMust(t, c.SetRational(&a, &q))
	body = c.makeBinary(&a, &b, &out, 0)
	var exception z3exception.DefaultException
	if !errors.As(body, &exception) || c.binary.active || c.multivariate != nil {
		t.Fatal("root-tag SASSERT did not use the source T action", body)
	}
	cmp, e := c.CompareRational(&out, &q)
	serviceMust(t, e)
	if cmp != 0 {
		t.Fatal("entry assertion changed the prior output")
	}
}
