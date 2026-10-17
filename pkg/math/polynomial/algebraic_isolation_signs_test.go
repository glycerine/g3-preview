package polynomial

import (
	"errors"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"reflect"
	"testing"
)

func TestAlgebraicIsolationSignsSourceIntervalsAndAppend(t *testing.T) {
	for _, tc := range []struct {
		name  string
		terms []polynomialEvalTerm
		roots int
		signs []int
		exact bool
	}{
		{"zero", nil, 0, []int{0}, true},
		{"constant", []polynomialEvalTerm{{-7, [3]uint32{}}}, 0, []int{-7}, true},
		{"no-real-roots", []polynomialEvalTerm{{1, [3]uint32{0, 0, 2}}, {2, [3]uint32{}}}, 0, []int{2}, true},
		{"linear", []polynomialEvalTerm{{1, [3]uint32{0, 0, 1}}}, 1, []int{-1, 1}, true},
		{"quadratic", []polynomialEvalTerm{{1, [3]uint32{0, 0, 2}}, {-2, [3]uint32{}}}, 2, []int{2, -1, 2}, true},
		{"double-root", []polynomialEvalTerm{{1, [3]uint32{0, 0, 2}}}, 1, []int{1, 1}, true},
		{"three-roots", []polynomialEvalTerm{{1, [3]uint32{0, 0, 3}}, {-1, [3]uint32{0, 0, 1}}}, 3, []int{-1, 1, -1, 1}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, _, _ := algebraicControlledFixture(t)
			pm, p, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, tc.terms)
			roots, e := c.NewVector()
			serviceMust(t, e)
			defer func() { serviceMust(t, roots.Close()) }()
			values := &algebraicIsolationAssignments{}
			signs := []int{71, 72}
			serviceMust(t, c.IsolateMultivariateRootsWithSigns(pm, p, values, roots, &signs))
			if len(roots.values) != tc.roots || len(signs) != 2+len(tc.signs) || signs[0] != 71 || signs[1] != 72 {
				t.Fatal("roots/sign prefix", len(roots.values), signs)
			}
			for i, want := range tc.signs {
				got := signs[i+2]
				if !tc.exact {
					if got < 0 {
						got = -1
					} else if got > 0 {
						got = 1
					}
				}
				if got != want {
					t.Fatal("interval sign", i, signs[i+2], want)
				}
			}
			if tc.name == "three-roots" && (signs[2] != -6 || signs[len(signs)-1] != 6) {
				t.Fatal("outside signs were normalized", signs)
			}
			if c.isolationSigns.active || pm.algebraicIsolationBorrowers != 0 {
				t.Fatal("successful signs retained owner")
			}
		})
	}
}
func TestAlgebraicIsolationSignsAlgebraicCoefficientAndPrecision(t *testing.T) {
	c, up, _ := algebraicControlledFixture(t)
	pm, p, _ := polynomialEvalFixtureWithRationals(t, c.config.Rationals, []polynomialEvalTerm{{1, [3]uint32{0, 0, 2}}, {-1, [3]uint32{1, 0, 0}}})
	v := algebraicIsolationAssignmentsFixture(t, c, up, false)
	roots, e := c.NewVector()
	serviceMust(t, e)
	defer func() { serviceMust(t, roots.Close()) }()
	var signs []int
	serviceMust(t, c.IsolateMultivariateRootsWithSigns(pm, p, v, roots, &signs))
	if !reflect.DeepEqual(signs, []int{1, -1, 1}) {
		t.Fatal(signs)
	}

	// RefineToPrecision(2) narrows each irrational root's interval to <1/4.
	for i := range roots.values {
		r, _, e := c.selectionValue(&roots.values[i])
		serviceMust(t, e)
		width := unumeral.NewBinaryRational()
		serviceMust(t, c.config.Binary.Sub(r.interval.BorrowUpper(), r.interval.BorrowLower(), &width))
		small, e := c.config.Binary.LessOneDiv2k(&width, 2)
		serviceMust(t, e)
		serviceMust(t, c.config.Binary.Del(&width))
		if !small {
			t.Fatal("source precision not reached", i)
		}
	}
	algebraicIsolationCheckFourthRoots(t, c, roots)
}

func TestAlgebraicAllUnassignedSourceLookupOrder(t *testing.T) {
	marker := errors.New("source assignment read")
	base := &algebraicSignTestValues{failure: marker}
	fallback := Algebraic{}
	v := algebraicAllUnassigned{base: base, value: &fallback}
	has, e := v.Contains(29)
	serviceMust(t, e)
	if !has || len(base.reads) != 0 {
		t.Fatal("Contains consulted base")
	}
	got, e := v.Algebraic(29)
	serviceMust(t, e)
	if got != &fallback || len(base.reads) != 0 {
		t.Fatal("unassigned fallback lookup")
	}
	_, e = v.Algebraic(1)
	if !errors.Is(e, marker) || !reflect.DeepEqual(base.reads, []uint32{1}) {
		t.Fatal("assigned lookup", base.reads, e)
	}
}
