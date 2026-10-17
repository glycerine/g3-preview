package polynomial

import (
	"errors"
	"reflect"
	"strconv"
	"testing"
)

func TestAlgebraicIsolationSignsEveryReachedCheckpointAndRetry(t *testing.T) {
	for _, tc := range []struct {
		name     string
		terms    []polynomialEvalTerm
		assigned bool
		roots    int
		want     []int
	}{
		{"quadratic", []polynomialEvalTerm{{1, [3]uint32{0, 0, 2}}, {-2, [3]uint32{}}}, false, 2, []int{2, -1, 2}},
		{"algebraic-coefficient", []polynomialEvalTerm{{1, [3]uint32{0, 0, 1}}, {-1, [3]uint32{1, 0, 0}}}, true, 1, []int{-1, 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, external := range []bool{false, true} {
				t.Run(strconv.FormatBool(external), func(t *testing.T) {
					succeeded, sawPartialSigns := false, false
					for allowance := uint64(0); allowance < 1024 && !succeeded; allowance++ {
						passed := t.Run(strconv.FormatUint(allowance, 10), func(t *testing.T) {
							c, up, control := algebraicControlledFixture(t)
							pm, p, pmControl := polynomialEvalFixtureWithRationals(t, c.config.Rationals, tc.terms)
							values := &algebraicIsolationAssignments{}
							if tc.assigned {
								values = algebraicIsolationAssignmentsFixture(t, c, up, false)
							}
							roots, e := c.NewVector()
							serviceMust(t, e)
							defer func() { serviceMust(t, roots.Close()) }()
							signs := []int{71}
							if external {
								pmControl.limit = pmControl.checkpoints + uint32(allowance)
							} else {
								control.limit = control.visits + allowance
							}
							body := c.IsolateMultivariateRootsWithSigns(pm, p, values, roots, &signs)
							succeeded = body == nil
							if body != nil && !errors.Is(body, ErrUPolynomialCanceled) {
								t.Fatal("unexpected checkpoint failure", body)
							}
							if body != nil && len(signs) > 1 {
								sawPartialSigns = true
							}
							if signs[0] != 71 {
								t.Fatal("body replaced caller sign prefix", signs)
							}
							prefix := append([]int(nil), signs...)
							count := roots.Len()
							control.limit = 1000000
							pmControl.limit = 100000
							serviceMust(t, c.CleanupRetained())
							if !reflect.DeepEqual(signs, prefix) || roots.Len() != count {
								t.Fatal("owning cleanup rolled back caller outputs", signs, prefix, roots.Len(), count)
							}
							serviceMust(t, roots.Reset())
							signs = signs[:1]
							serviceMust(t, c.IsolateMultivariateRootsWithSigns(pm, p, values, roots, &signs))
							if roots.Len() != uint32(tc.roots) || !reflect.DeepEqual(signs[1:], tc.want) {
								t.Fatal("retry outputs", roots.Len(), signs)
							}
						})
						if !passed {
							return
						}
					}
					if !succeeded {
						t.Fatal("did not reach successful checkpoint endpoint")
					}
					if external && !sawPartialSigns {
						t.Fatal("did not reach failure after sign prefix publication")
					}
				})
			}
		})
	}
}
