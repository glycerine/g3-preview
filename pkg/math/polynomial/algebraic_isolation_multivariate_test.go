package polynomial

import (
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"testing"
)

type algebraicIsolationAssignments struct {
	values          [3]Algebraic
	assigned        [3]bool
	reads, contains []uint32
}

func (v *algebraicIsolationAssignments) Contains(x uint32) (bool, error) {
	v.contains = append(v.contains, x)
	return x < 3 && v.assigned[x], nil
}
func (v *algebraicIsolationAssignments) Algebraic(x uint32) (*Algebraic, error) {
	v.reads = append(v.reads, x)
	if x >= 3 || !v.assigned[x] {
		return nil, ErrAlgebraicOwner
	}
	return &v.values[x], nil
}
func algebraicIsolationAssignmentsFixture(t *testing.T, c *AlgebraicRootCore, up *UPolynomialManager, same bool) *algebraicIsolationAssignments {
	t.Helper()
	v := &algebraicIsolationAssignments{assigned: [3]bool{true, true, false}}
	v.values[0], v.values[1] = algebraicResultantInputs(t, c, up)
	if same {
		serviceMust(t, c.Del(&v.values[1]))
		serviceMust(t, c.Copy(&v.values[1], &v.values[0]))
	}
	t.Cleanup(func() {
		for i := range v.values {
			serviceMust(t, c.Del(&v.values[i]))
		}
	})
	return v
}
func algebraicIsolationCheckFourthRoots(t *testing.T, c *AlgebraicRootCore, roots *AlgebraicVector) {
	t.Helper()
	if len(roots.values) != 2 {
		t.Fatal("fourth root count", len(roots.values))
	}
	algebraicExpectBetween(t, c, &roots.values[0], -2, -1)
	algebraicExpectBetween(t, c, &roots.values[1], 1, 2)
	var fourth Algebraic
	defer func() { serviceMust(t, c.Del(&fourth)) }()
	q := unumeral.NewRational()
	serviceMust(t, c.config.Rationals.SetInt32(&q, 2))
	defer func() { serviceMust(t, c.config.Rationals.Del(&q)) }()
	for i := range roots.values {
		serviceMust(t, c.Power(&roots.values[i], 4, &fourth))
		cmp, e := c.CompareRational(&fourth, &q)
		serviceMust(t, e)
		if cmp != 0 {
			t.Fatal("root fourth power", i, cmp)
		}
	}
}
func TestAlgebraicMultivariateIsolationConjugateFiltering(t *testing.T) {
	for _, tc := range []struct {
		name  string
		terms []polynomialEvalTerm
		count int
	}{
		{"linear", []polynomialEvalTerm{{1, [3]uint32{0, 0, 1}}, {-1, [3]uint32{1, 0, 0}}}, 1},
		{"fourth-roots", []polynomialEvalTerm{{1, [3]uint32{0, 0, 2}}, {-1, [3]uint32{1, 0, 0}}}, 2},
		{"spurious-only", []polynomialEvalTerm{{1, [3]uint32{0, 0, 2}}, {1, [3]uint32{1, 0, 0}}}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, up, _ := algebraicControlledFixture(t)
			pm, p, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, tc.terms)
			v := algebraicIsolationAssignmentsFixture(t, c, up, false)
			roots, e := c.NewVector()
			serviceMust(t, e)
			defer func() { serviceMust(t, roots.Close()) }()
			serviceMust(t, c.IsolateMultivariateRoots(pm, p, v, roots))
			if len(roots.values) != tc.count {
				t.Fatal("root count", len(roots.values), tc.count)
			}
			if tc.count == 1 {
				cmp, e := c.Compare(&roots.values[0], &v.values[0])
				serviceMust(t, e)
				if cmp != 0 {
					t.Fatal("linear root", cmp)
				}
			}
			if tc.count == 2 {
				algebraicIsolationCheckFourthRoots(t, c, roots)
			}
		})
	}
}
func TestAlgebraicMultivariateIsolationVanishedResultantBranches(t *testing.T) {
	for _, tc := range []struct {
		name  string
		terms []polynomialEvalTerm
		count int
	}{
		// y=z=sqrt(2); p=(y+z)*(x-y).
		{"linear", []polynomialEvalTerm{{1, [3]uint32{1, 0, 1}}, {1, [3]uint32{0, 1, 1}}, {-1, [3]uint32{2, 0, 0}}, {-1, [3]uint32{1, 1, 0}}}, 1},
		// p=(y+z)*(x^2-y).
		{"quadratic-recursion", []polynomialEvalTerm{{1, [3]uint32{1, 0, 2}}, {1, [3]uint32{0, 1, 2}}, {-1, [3]uint32{2, 0, 0}}, {-1, [3]uint32{1, 1, 0}}}, 2},
		// p=(y-z)*x^3+(y+z)*(x^2-y): the cubic coefficient vanishes.
		{"degree-drop", []polynomialEvalTerm{{1, [3]uint32{1, 0, 3}}, {-1, [3]uint32{0, 1, 3}}, {1, [3]uint32{1, 0, 2}}, {1, [3]uint32{0, 1, 2}}, {-1, [3]uint32{2, 0, 0}}, {-1, [3]uint32{1, 1, 0}}}, 2},
		{"linear-coefficient-zero", []polynomialEvalTerm{{1, [3]uint32{1, 0, 1}}, {-1, [3]uint32{0, 1, 1}}, {1, [3]uint32{1, 0, 0}}, {-1, [3]uint32{0, 1, 0}}}, 0},
		{"all-positive-degree-coefficients-zero", []polynomialEvalTerm{{1, [3]uint32{1, 0, 2}}, {-1, [3]uint32{0, 1, 2}}, {1, [3]uint32{1, 0, 0}}, {-1, [3]uint32{0, 1, 0}}}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, up, _ := algebraicControlledFixture(t)
			pm, p, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, tc.terms)
			v := algebraicIsolationAssignmentsFixture(t, c, up, true)
			roots, e := c.NewVector()
			serviceMust(t, e)
			defer func() { serviceMust(t, roots.Close()) }()
			serviceMust(t, c.IsolateMultivariateRoots(pm, p, v, roots))
			if len(roots.values) != tc.count {
				t.Fatal("root count", len(roots.values), tc.count)
			}
			if tc.count == 1 {
				cmp, e := c.Compare(&roots.values[0], &v.values[0])
				serviceMust(t, e)
				if cmp != 0 {
					t.Fatal("linear root", cmp)
				}
			}
			if tc.count == 2 {
				algebraicIsolationCheckFourthRoots(t, c, roots)
			}
			if pm.algebraicIsolationBorrowers != 0 {
				t.Fatal("completed external manager borrow")
			}
			for _, f := range c.multivariateIsolationFrames {
				if f.active {
					t.Fatal("completed lexical frame retained")
				}
			}
		})
	}
}

func TestAlgebraicMultivariateIsolationRationalEliminationAndVanishingVariable(t *testing.T) {
	for _, tc := range []struct {
		name           string
		terms          []polynomialEvalTerm
		basic0, basic1 bool
		n0, n1         int32
		count          int
	}{
		{"zero", []polynomialEvalTerm{{1, [3]uint32{1, 0, 1}}}, true, false, 0, 0, 0},
		{"constant", []polynomialEvalTerm{{1, [3]uint32{1, 0, 1}}, {1, [3]uint32{}}}, true, false, 0, 0, 0},
		{"remaining-univariate-assigned", []polynomialEvalTerm{{1, [3]uint32{1, 0, 1}}, {1, [3]uint32{0, 1, 0}}}, true, false, 0, 0, 0},
		{"remaining-univariate-unassigned", []polynomialEvalTerm{{1, [3]uint32{1, 0, 1}}, {1, [3]uint32{0, 1, 0}}}, true, true, 2, -6, 1},
		{"all-algebraic-variables-assigned", []polynomialEvalTerm{{1, [3]uint32{1, 0, 0}}, {1, [3]uint32{0, 1, 0}}}, false, false, 0, 0, 0},
		{"constant-resultant", []polynomialEvalTerm{{1, [3]uint32{2, 0, 1}}, {-2, [3]uint32{0, 0, 1}}, {1, [3]uint32{}}}, false, false, 0, 0, 0},
		{"original-univariate-ignores-assignment", []polynomialEvalTerm{{1, [3]uint32{2, 0, 0}}, {-2, [3]uint32{}}}, false, false, 0, 0, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, up, _ := algebraicControlledFixture(t)
			pm, p, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, tc.terms)
			v := algebraicIsolationAssignmentsFixture(t, c, up, false)
			for i, basic := range []bool{tc.basic0, tc.basic1} {
				if basic {
					n := unumeral.NewInt([]int32{tc.n0, tc.n1}[i])
					serviceMust(t, c.SetInteger(&v.values[i], &n))
				}
			}
			roots, e := c.NewVector()
			serviceMust(t, e)
			defer func() { serviceMust(t, roots.Close()) }()
			serviceMust(t, c.IsolateMultivariateRoots(pm, p, v, roots))
			if len(roots.values) != tc.count {
				t.Fatal("root count", len(roots.values), tc.count)
			}
			if tc.count == 1 {
				algebraicExpectBetween(t, c, &roots.values[0], 2, 4)
			}
			if tc.name == "original-univariate-ignores-assignment" && (len(v.reads) != 0 || len(v.contains) != 0) {
				t.Fatal("univariate shortcut read assignments")
			}
		})
	}
}
