package polynomial

import (
	"errors"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"strconv"
	"testing"
)

func TestAlgebraicEvalSignExactResultantsAndSignedBounds(t *testing.T) {
	for _, tc := range []struct {
		name  string
		terms []polynomialEvalTerm
		want  int
	}{
		{"square-zero", []polynomialEvalTerm{{1, [3]uint32{2, 0, 0}}, {-2, [3]uint32{}}}, 0},
		{"sum-of-squares-zero", []polynomialEvalTerm{{1, [3]uint32{2, 0, 0}}, {1, [3]uint32{0, 2, 0}}, {-5, [3]uint32{}}}, 0},
		{"positive", []polynomialEvalTerm{{1, [3]uint32{1, 0, 0}}, {1, [3]uint32{0, 1, 0}}, {-3, [3]uint32{}}}, 1},
		{"negative", []polynomialEvalTerm{{-1, [3]uint32{1, 0, 0}}, {-1, [3]uint32{0, 1, 0}}, {3, [3]uint32{}}}, -1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, p, _ := algebraicControlledFixture(t)
			c.minMagnitude = 0
			pm, poly, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, tc.terms)
			a, b := algebraicResultantInputs(t, c, p)
			values := algebraicSignTestValues{}
			values.values[0], values.values[1] = a, b
			sign, e := c.EvalSign(pm, poly, &values)
			serviceMust(t, e)
			if sign != tc.want {
				t.Fatal("exact algebraic sign", sign, tc.want)
			}
			if c.evalSignTmp.slot == 0 || c.multivariate == nil {
				t.Fatal("exact resultant path was not reached")
			}
			if c.signResultant.active || c.signInterval.active || c.signSubstitution.active {
				t.Fatal("completed sign retained lexical owners")
			}
			serviceMust(t, c.Del(&values.values[1]))
			serviceMust(t, c.Del(&values.values[0]))
		})
	}
}

func TestAlgebraicEvalSignEveryExactCheckpointCleanupRetry(t *testing.T) {
	success := false
	for allowance := uint64(0); allowance < 512 && !success; allowance++ {
		t.Run(strconv.FormatUint(allowance, 10), func(t *testing.T) {
			c, p, control := algebraicControlledFixture(t)
			c.minMagnitude = 0
			pm, poly, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, []polynomialEvalTerm{{1, [3]uint32{1, 0, 0}}, {1, [3]uint32{0, 1, 0}}, {-3, [3]uint32{}}})
			a, b := algebraicResultantInputs(t, c, p)
			values := algebraicSignTestValues{}
			values.values[0], values.values[1] = a, b
			control.limit = control.visits + allowance
			sign, body := c.EvalSign(pm, poly, &values)
			control.limit = 1000000
			if body == nil {
				success = true
			} else {
				if !errors.Is(body, ErrUPolynomialCanceled) {
					t.Fatal(body)
				}
				serviceMust(t, c.CleanupRetained())
				sign, body = c.EvalSign(pm, poly, &values)
				serviceMust(t, body)
			}
			if sign != 1 {
				t.Fatal("retry changed exact sign", sign)
			}
			serviceMust(t, c.CleanupRetained())
			serviceMust(t, c.Del(&values.values[1]))
			serviceMust(t, c.Del(&values.values[0]))
		})
	}
	if !success {
		t.Fatal("no complete exact evaluation endpoint")
	}
}

func TestAlgebraicEvalSignPublicRationalShortCircuit(t *testing.T) {
	c, _, _ := algebraicControlledFixture(t)
	pm, p, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, []polynomialEvalTerm{{1, [3]uint32{1, 0, 0}}, {1, [3]uint32{}}})
	values := algebraicSignTestValues{}
	q := unumeral.NewRational()
	serviceMust(t, c.config.Rationals.SetFraction(&q, 3, 2))
	serviceMust(t, c.SetRational(&values.values[0], &q))
	sign, e := c.EvalSign(pm, p, &values)
	serviceMust(t, e)
	if sign != 5 || c.multivariate != nil {
		t.Fatal("source rational sign branch", sign)
	}
	serviceMust(t, c.Del(&values.values[0]))
	serviceMust(t, c.config.Rationals.Del(&q))
}

func TestAlgebraicEvalSignDegreeOrderAndRationalRestart(t *testing.T) {
	c, p, _ := algebraicControlledFixture(t)
	c.minMagnitude = 0
	pm, poly, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, []polynomialEvalTerm{{1, [3]uint32{2, 0, 0}}, {1, [3]uint32{0, 3, 0}}, {-5, [3]uint32{}}})
	a, b := algebraicResultantInputs(t, c, p)
	serviceMust(t, c.Del(&b))
	b = algebraicCompareRoot(t, c, []int32{-3, 0, 0, 1}, 1, 0, 2, 0, true)
	values := algebraicSignTestValues{}
	values.values[0], values.values[1] = a, b
	sign, e := c.EvalSign(pm, poly, &values)
	serviceMust(t, e)
	if sign != 0 {
		t.Fatal("sqrt(2)^2+cuberoot(3)^3-5", sign)
	}
	if len(c.evalSignVariables.values) != 2 || c.evalSignVariables.values[0] != 0 || c.evalSignVariables.values[1] != 1 {
		t.Fatal("variables not sorted by defining polynomial degree", c.evalSignVariables.values)
	}
	serviceMust(t, c.Del(&values.values[1]))
	serviceMust(t, c.Del(&values.values[0]))
	// Force the exact phase to find rational x=1 and restart the outer loop.
	root := algebraicCompareRoot(t, c, []int32{-1, 0, 1}, 0, 0, 2, 0, false)
	values.values[0] = root
	restartPolynomial := polynomialQueryTerms(t, pm.core, []int32{1, -2}, [][]MonomialPower{{{0, 2}}, nil})
	values.values[1] = Algebraic{}
	sign, e = c.EvalSign(pm, restartPolynomial, &values)
	serviceMust(t, e)
	if sign != -1 {
		t.Fatal("rational restart sign", sign)
	}
	l, e := c.location(&values.values[0])
	serviceMust(t, e)
	if l == nil || l.kind != 0 {
		t.Fatal("exact phase did not discover rational input")
	}
	serviceMust(t, c.Del(&values.values[0]))
}

func TestAlgebraicEvalSignExactResultantRestoresIntervals(t *testing.T) {
	c, p, _ := algebraicControlledFixture(t)
	c.minMagnitude = 0
	pm, poly, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, []polynomialEvalTerm{{1, [3]uint32{1, 0, 0}}, {1, [3]uint32{0, 1, 0}}, {-3, [3]uint32{}}})
	a, b := algebraicResultantInputs(t, c, p)
	values := algebraicSignTestValues{}
	values.values[0], values.values[1] = a, b
	prime, _, constant, e := c.prepareSignPolynomial(pm, poly, &values)
	serviceMust(t, e)
	if constant {
		t.Fatal("unexpected constant")
	}
	_, determined, restart, e := c.intervalSignPrefix(pm, prime, &values)
	serviceMust(t, e)
	if determined || restart {
		t.Fatal("resultant phase not needed")
	}
	var before [2]algebraicRecord
	for i := range before {
		l, e := c.location(&values.values[i])
		serviceMust(t, e)
		before[i], e = c.load(l)
		serviceMust(t, e)
	}
	sign, restart, e := c.exactSignResultant(pm, prime, &values)
	serviceMust(t, e)
	if sign != 1 || restart || len(c.signResultant.saved) != 2 {
		t.Fatal("exact refinement path", sign, restart)
	}
	// The conjugates of sqrt(2)+sqrt(3)-3 satisfy
	// (y+3)^4-10*(y+3)^2+1.
	checkAlgebraicResultantPolynomial(t, p, c.evalSignTmp, -8, 48, 44, 12, 1)
	for i := range before {
		l, e := c.location(&values.values[i])
		serviceMust(t, e)
		after, e := c.load(l)
		serviceMust(t, e)
		if before[i] != after {
			t.Fatal("saved input interval not restored", i)
		}
	}
	serviceMust(t, c.CleanupRetained())
	serviceMust(t, c.Del(&values.values[1]))
	serviceMust(t, c.Del(&values.values[0]))
}
