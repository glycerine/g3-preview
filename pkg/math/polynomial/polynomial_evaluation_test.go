// Source: src/math/polynomial/polynomial.cpp:6436-6554,7409-7418;
// src/test/polynomial.cpp:1340-1387,1824-1826.
package polynomial

import (
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

type polynomialEvalTerm struct {
	coefficient int32
	degrees     [3]uint32
}
type polynomialEvalControl struct {
	checkpoints, cooperation uint32
	limit                    uint32
}

func (c *polynomialEvalControl) checkpoint() bool { c.checkpoints++; return c.checkpoints <= c.limit }
func (c *polynomialEvalControl) cooperate() error { c.cooperation++; return nil }

type polynomialEvalRationalMap struct {
	manager                   *util.RationalManager
	values                    [3]util.Rational
	managers, contains, reads uint32
}

func (v *polynomialEvalRationalMap) Manager() (*util.RationalManager, error) {
	v.managers++
	return v.manager, nil
}
func (v *polynomialEvalRationalMap) Contains(x uint32) (bool, error) { v.contains++; return x < 3, nil }
func (v *polynomialEvalRationalMap) Rational(x uint32) (*util.Rational, error) {
	v.reads++
	if x >= 3 {
		return nil, ErrPolynomialAssertion
	}
	return &v.values[x], nil
}
func polynomialEvalFixture(t *testing.T, terms []polynomialEvalTerm) (*MultivariatePolynomialManager, Polynomial, *polynomialEvalControl) {
	t.Helper()
	q, _ := rationalManager(t, false)
	return polynomialEvalFixtureWithRationals(t, q, terms)
}
func polynomialEvalFixtureWithRationals(t *testing.T, q *util.RationalManager, terms []polynomialEvalTerm) (*MultivariatePolynomialManager, Polynomial, *polynomialEvalControl) {
	t.Helper()
	control := &polynomialEvalControl{limit: 100000}
	m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Checkpoint: control.checkpoint, Cooperation: control.cooperate}})
	serviceMust(t, e)
	for i := 0; i < 3; i++ {
		_, e = m.core.monomials.MkVar()
		serviceMust(t, e)
	}
	as := make([]util.Int, len(terms))
	ms := make([]Monomial, len(terms))
	for i, term := range terms {
		as[i] = util.NewInt(term.coefficient)
		powers := []MonomialPower{}
		for x, d := range term.degrees {
			if d != 0 {
				powers = append(powers, MonomialPower{Var: uint32(x), Degree: d})
			}
		}
		ms[i], e = m.core.monomials.MkMonomial(powers)
		serviceMust(t, e)
	}
	p, e := m.core.MkTerms(as, ms)
	serviceMust(t, e)
	serviceMust(t, m.core.IncRef(p))
	t.Cleanup(func() {
		if !m.closed {
			serviceMust(t, m.core.DecRef(p))
			serviceMust(t, m.CloseChecked())
		}
		if control.cooperation > control.checkpoints {
			t.Fatal("cooperation without checkpoint")
		}
	})
	return m, p, control
}

// All twenty expected values are literal dormant source ENSURE inputs. The
// registered driver returns immediately after PSC, BEFORE tst_eval. Dense term
// expansion replaces the expression-builder setup, not the evaluated algorithm.
func TestPolynomialEvaluationDormantTwentySourceRationals(t *testing.T) {
	term := func(c int32, x, y, z uint32) polynomialEvalTerm { return polynomialEvalTerm{c, [3]uint32{x, y, z}} }
	cases := []struct {
		terms    []polynomialEvalTerm
		values   [3][2]int64
		expected [2]int64
	}{
		{[]polynomialEvalTerm{term(2000, 0, 1, 0), term(-1, 1, 0, 0)}, [3][2]int64{{1, 1}, {2, 1}, {3, 1}}, [2]int64{3999, 1}},
		{[]polynomialEvalTerm{term(1, 1, 0, 0), term(1, 0, 0, 0)}, [3][2]int64{{1, 1}, {2, 1}, {3, 1}}, [2]int64{2, 1}},
		{[]polynomialEvalTerm{term(1, 3, 0, 0), term(1, 1, 0, 0), term(1, 0, 0, 0)}, [3][2]int64{{2, 1}, {2, 1}, {3, 1}}, [2]int64{11, 1}},
		{[]polynomialEvalTerm{term(1, 3, 0, 0), term(-2, 1, 0, 0), term(1, 0, 0, 0)}, [3][2]int64{{2, 1}, {2, 1}, {3, 1}}, [2]int64{5, 1}},
		{[]polynomialEvalTerm{term(1, 3, 0, 0), term(-2, 1, 0, 0), term(1, 0, 0, 0)}, [3][2]int64{{-2, 1}, {2, 1}, {3, 1}}, [2]int64{-3, 1}},
		{[]polynomialEvalTerm{term(1, 4, 0, 0), term(-2, 1, 0, 0), term(1, 0, 1, 0), term(1, 0, 0, 0)}, [3][2]int64{{-2, 1}, {10, 1}, {3, 1}}, [2]int64{31, 1}},
		{[]polynomialEvalTerm{term(1, 4, 0, 0), term(-2, 1, 0, 0), term(1, 3, 1, 0), term(1, 0, 1, 0), term(1, 0, 0, 0)}, [3][2]int64{{-2, 1}, {10, 1}, {3, 1}}, [2]int64{-49, 1}},
		{[]polynomialEvalTerm{term(1, 4, 2, 0), term(-2, 1, 2, 0), term(1, 3, 1, 0), term(1, 0, 1, 0), term(1, 2, 0, 0), term(1, 0, 0, 0)}, [3][2]int64{{-2, 1}, {10, 1}, {3, 1}}, [2]int64{1935, 1}},
		{[]polynomialEvalTerm{term(1, 4, 2, 3), term(-2, 1, 2, 3), term(1, 3, 1, 0), term(1, 0, 1, 0), term(1, 2, 0, 0), term(1, 0, 0, 0)}, [3][2]int64{{-2, 1}, {10, 1}, {0, 1}}, [2]int64{-65, 1}},
		{[]polynomialEvalTerm{term(1, 4, 2, 3), term(1, 4, 0, 3), term(-2, 1, 2, 3), term(-2, 1, 0, 3), term(1, 3, 1, 0), term(1, 0, 1, 0), term(1, 2, 0, 0), term(1, 0, 0, 0)}, [3][2]int64{{-2, 1}, {10, 1}, {1, 2}}, [2]int64{375, 2}},
		{[]polynomialEvalTerm{term(1, 1, 1, 1)}, [3][2]int64{{2, 1}, {3, 1}, {1, 1}}, [2]int64{6, 1}},
		{[]polynomialEvalTerm{term(1, 1, 1, 1), term(1, 0, 0, 0)}, [3][2]int64{{2, 1}, {3, 1}, {1, 1}}, [2]int64{7, 1}},
		{[]polynomialEvalTerm{term(1, 0, 0, 0)}, [3][2]int64{{2, 1}, {3, 1}, {1, 1}}, [2]int64{1, 1}},
		{[]polynomialEvalTerm{term(1, 1, 2, 1), term(1, 0, 0, 0)}, [3][2]int64{{2, 1}, {3, 1}, {1, 1}}, [2]int64{19, 1}},
		{[]polynomialEvalTerm{term(1, 1, 2, 1), term(1, 0, 1, 0), term(1, 0, 0, 0)}, [3][2]int64{{2, 1}, {3, 1}, {1, 1}}, [2]int64{22, 1}},
		{[]polynomialEvalTerm{term(1, 1, 2, 1), term(1, 0, 1, 0), term(1, 0, 0, 0), term(2, 0, 1, 2), term(-1, 0, 0, 2)}, [3][2]int64{{2, 1}, {3, 1}, {1, 1}}, [2]int64{27, 1}},
		{[]polynomialEvalTerm{term(1, 5, 0, 0), term(1, 0, 0, 0)}, [3][2]int64{{2, 1}, {3, 1}, {1, 1}}, [2]int64{33, 1}},
		{[]polynomialEvalTerm{term(1, 5, 0, 0), term(1, 1, 1, 0), term(1, 0, 0, 0)}, [3][2]int64{{2, 1}, {1, 1}, {5, 1}}, [2]int64{35, 1}},
		{[]polynomialEvalTerm{term(1, 0, 5, 0), term(1, 1, 1, 0), term(1, 0, 0, 0)}, [3][2]int64{{2, 1}, {1, 1}, {5, 1}}, [2]int64{4, 1}},
		{[]polynomialEvalTerm{term(1, 0, 5, 0), term(1, 1, 2, 0), term(1, 0, 0, 0)}, [3][2]int64{{2, 1}, {-2, 1}, {5, 1}}, [2]int64{-23, 1}},
	}
	for i, tc := range cases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			m, p, control := polynomialEvalFixture(t, tc.terms)
			q := m.core.config.Rationals
			values := &polynomialEvalRationalMap{manager: q}
			for j, f := range tc.values {
				values.values[j] = util.NewRational()
				serviceMust(t, q.SetFraction(&values.values[j], f[0], uint64(f[1])))
			}
			result, expected := util.NewRational(), util.NewRational()
			serviceMust(t, m.EvaluateRational(p, values, &result))
			serviceMust(t, q.SetFraction(&expected, tc.expected[0], uint64(tc.expected[1])))
			equal, e := q.Equal(&result, &expected)
			serviceMust(t, e)
			if !equal {
				t.Fatal("dormant literal expected rational", rationalString(t, q, &result), tc.expected)
			}
			if values.managers != 1 || control.checkpoints != control.cooperation {
				t.Fatal("actual manager/checkpoint binding", values.managers, control)
			}
			serviceMust(t, q.Del(&result))
			serviceMust(t, q.Del(&expected))
			for j := range values.values {
				serviceMust(t, q.Del(&values.values[j]))
			}
		})
	}
}
