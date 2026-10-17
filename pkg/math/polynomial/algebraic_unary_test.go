package polynomial

import (
	"errors"
	"strconv"
	"testing"

	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"github.com/glycerine/g3/pkg/util/z3exception"
)

func TestAlgebraicUnaryPowerRootAndAliases(t *testing.T) {
	for _, factor := range []bool{false, true} {
		for _, alias := range []bool{false, true} {
			c, p, _ := algebraicControlledFixture(t)
			c.factor = factor
			a, b := algebraicResultantInputs(t, c, p)
			var result Algebraic
			out := &result
			if alias {
				out = &a
			}
			serviceMust(t, c.Power(&a, 3, out))
			algebraicExpectBetween(t, c, out, 2, 3)
			serviceMust(t, c.Root(out, 3, out))
			algebraicExpectBetween(t, c, out, 1, 2)
			serviceMust(t, c.Power(out, 2, out))
			q := unumeral.NewRational()
			serviceMust(t, c.config.Rationals.SetInt32(&q, 2))
			cmp, e := c.CompareRational(out, &q)
			serviceMust(t, e)
			if cmp != 0 {
				t.Fatal("sqrt(2) squared", cmp)
			}
			serviceMust(t, c.config.Rationals.Del(&q))
			serviceMust(t, c.Del(&result))
			serviceMust(t, c.Del(&b))
			serviceMust(t, c.Del(&a))
		}
	}
}

func TestAlgebraicUnaryRationalRootsAndDomains(t *testing.T) {
	c, _, _ := algebraicControlledFixture(t)
	q := unumeral.NewRational()
	var a, out Algebraic
	defer func() {
		serviceMust(t, c.Del(&a))
		serviceMust(t, c.Del(&out))
		serviceMust(t, c.config.Rationals.Del(&q))
	}()
	if e := c.Root(nil, 0, &out); !errors.Is(e, ErrAlgebraicZeroRoot) {
		t.Fatal(e)
	}
	if e := c.Power(&a, 0, &out); !errors.Is(e, ErrAlgebraicZeroPower) {
		t.Fatal(e)
	}
	serviceMust(t, c.Root(&a, 3, &out))
	for _, tc := range []struct {
		n, d   int64
		k      uint32
		lo, hi int32
	}{
		{8, 27, 3, 0, 1}, {2, 1, 2, 1, 2}, {2, 3, 3, 0, 1}, {-2, 3, 3, -1, 0}, {-8, 27, 3, -1, 0},
	} {
		serviceMust(t, c.config.Rationals.SetFraction(&q, tc.n, uint64(tc.d)))
		serviceMust(t, c.SetRational(&a, &q))
		serviceMust(t, c.Root(&a, tc.k, &out))
		algebraicExpectBetween(t, c, &out, tc.lo, tc.hi)
		if e := c.Power(&out, tc.k, &out); e != nil {
			t.Fatalf("power after root %+v: %v", tc, e)
		}
		cmp, e := c.CompareRational(&out, &q)
		serviceMust(t, e)
		if cmp != 0 {
			t.Fatal("root/power identity", tc, cmp)
		}
	}
	serviceMust(t, c.config.Rationals.SetInt32(&q, -2))
	serviceMust(t, c.SetRational(&a, &q))
	if e := c.Root(&a, 2, &out); !errors.Is(e, ErrAlgebraicEvenNegativeRoot) {
		t.Fatal(e)
	}
	serviceMust(t, c.Power(&a, 0, &out))
	algebraicExpectBetween(t, c, &out, 0, 2)
	serviceMust(t, c.Power(&a, 1, &a))
	serviceMust(t, c.Root(&a, 1, &a))
}

func TestAlgebraicUnaryEveryCheckpointCancellationRetry(t *testing.T) {
	for _, root := range []bool{false, true} {
		success := false
		for allowance := uint64(0); allowance < 512 && !success; allowance++ {
			t.Run(strconv.FormatBool(root)+"/"+strconv.FormatUint(allowance, 10), func(t *testing.T) {
				c, p, control := algebraicControlledFixture(t)
				a, b := algebraicResultantInputs(t, c, p)
				var out Algebraic
				call := func() error {
					if root {
						return c.Root(&a, 3, &out)
					}
					return c.Power(&a, 3, &out)
				}
				control.limit = control.visits + allowance
				body := call()
				control.limit = 1000000
				if body == nil {
					success = true
				} else {
					if !errors.Is(body, ErrUPolynomialCanceled) {
						t.Fatal(body)
					}
					if !errors.Is(call(), ErrAlgebraicPending) {
						t.Fatal("pending selector overwritten")
					}
					serviceMust(t, c.CleanupRetained())
					serviceMust(t, call())
				}
				if root {
					algebraicExpectBetween(t, c, &out, 1, 2)
				} else {
					algebraicExpectBetween(t, c, &out, 2, 3)
				}
				serviceMust(t, c.Del(&out))
				serviceMust(t, c.Del(&b))
				serviceMust(t, c.Del(&a))
			})
		}
		if !success {
			t.Fatal("selector never succeeded")
		}
	}
}

func TestAlgebraicUnaryNegativeRootsAndCubicInput(t *testing.T) {
	c, _, _ := algebraicControlledFixture(t)
	a := algebraicCompareRoot(t, c, []int32{3, 0, 0, 1}, -2, 0, -1, 0, true)
	var out Algebraic
	defer func() { serviceMust(t, c.Del(&out)); serviceMust(t, c.Del(&a)) }()
	serviceMust(t, c.Root(&a, 3, &out))
	algebraicExpectBetween(t, c, &out, -2, -1)
	serviceMust(t, c.Power(&out, 3, &out))
	cmp, e := c.Compare(&a, &out)
	serviceMust(t, e)
	if cmp != 0 {
		t.Fatal("negative root/power identity", cmp)
	}
	serviceMust(t, c.Power(&a, 2, &out))
	algebraicExpectBetween(t, c, &out, 2, 3)
	serviceMust(t, c.Power(&a, 3, &a))
	q := unumeral.NewRational()
	serviceMust(t, c.config.Rationals.SetInt32(&q, -3))
	cmp, e = c.CompareRational(&a, &q)
	serviceMust(t, e)
	if cmp != 0 {
		t.Fatal("negative cube", cmp)
	}
	serviceMust(t, c.config.Rationals.Del(&q))
}

func TestAlgebraicUnaryRationalRefinementFallback(t *testing.T) {
	c, _, control := algebraicControlledFixture(t)
	// x^2-4x+3 has roots 1 and 3. Cubic root bounds (0,2)
	// include both transformed roots; refinement finds the exact input 1.
	a := algebraicCompareRoot(t, c, []int32{3, -4, 1}, 0, 0, 2, 0, false)
	before := control.visits
	serviceMust(t, c.Root(&a, 3, &a))
	l, e := c.location(&a)
	serviceMust(t, e)
	if l == nil || l.kind != 0 || control.visits == before {
		t.Fatal("selector failed to reach rational fallback")
	}
	q := unumeral.NewRational()
	serviceMust(t, c.config.Rationals.SetInt32(&q, 1))
	cmp, e := c.CompareRational(&a, &q)
	serviceMust(t, e)
	if cmp != 0 {
		t.Fatal("rational fallback", cmp)
	}
	serviceMust(t, c.config.Rationals.Del(&q))
	serviceMust(t, c.Del(&a))
}

func TestAlgebraicUnarySourceAssertionsBeforeLocals(t *testing.T) {
	if !polynomialSourceDebug {
		t.Skip("source assertions erased")
	}
	c, _, _ := algebraicControlledFixture(t)
	polynomialAssertionActionsFixture(t, "TTT")
	zero := unumeral.NewRational()
	var out Algebraic
	var exception z3exception.DefaultException
	if e := c.basicRoot(&zero, 3, &out); !errors.As(e, &exception) || c.basicRootScope.active {
		t.Fatal("root zero assertion after locals", e)
	}
	serviceMust(t, c.config.Rationals.SetInt32(&zero, 2))
	if e := c.basicRoot(&zero, 1, &out); !errors.As(e, &exception) || c.basicRootScope.active {
		t.Fatal("degree assertion after locals", e)
	}
	if e := c.makeUnary(&out, &out, 3, false); !errors.As(e, &exception) || c.binary.active {
		t.Fatal("unary tag assertion after locals", e)
	}
	serviceMust(t, c.config.Rationals.Del(&zero))
}
