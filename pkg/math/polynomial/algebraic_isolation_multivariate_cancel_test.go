package polynomial

import (
	"fmt"
	"testing"
)

func TestAlgebraicMultivariateIsolationEveryReachedCheckpoint(t *testing.T) {
	for _, tc := range []struct {
		name  string
		terms []polynomialEvalTerm
		same  bool
		count int
	}{
		{"filter", []polynomialEvalTerm{{1, [3]uint32{0, 0, 1}}, {-1, [3]uint32{1, 0, 0}}}, false, 1},
		{"linear-vanished", []polynomialEvalTerm{{1, [3]uint32{1, 0, 1}}, {1, [3]uint32{0, 1, 1}}, {-1, [3]uint32{2, 0, 0}}, {-1, [3]uint32{1, 1, 0}}}, true, 1},
		{"nested-vanished", []polynomialEvalTerm{{1, [3]uint32{1, 0, 2}}, {1, [3]uint32{0, 1, 2}}, {-1, [3]uint32{2, 0, 0}}, {-1, [3]uint32{1, 1, 0}}}, true, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, external := range []bool{false, true} {
				t.Run(fmt.Sprint("external-", external), func(t *testing.T) {
					succeeded := false
					for allowance := uint64(0); allowance < 4096 && !succeeded; allowance++ {
						passed := t.Run(fmt.Sprint(allowance), func(t *testing.T) {
							c, up, control := algebraicControlledFixture(t)
							pm, p, pmControl := polynomialEvalFixtureWithRationals(t, c.config.Rationals, tc.terms)
							v := algebraicIsolationAssignmentsFixture(t, c, up, tc.same)
							roots, e := c.NewVector()
							serviceMust(t, e)
							defer func() { serviceMust(t, roots.Close()) }()
							if external {
								pmControl.limit = pmControl.checkpoints + uint32(allowance)
							} else {
								control.limit = control.visits + allowance
							}
							body := c.IsolateMultivariateRoots(pm, p, v, roots)
							succeeded = body == nil
							if succeeded {
								t.Log("first successful checkpoint allowance", allowance)
							}
							control.limit = 1000000
							pmControl.limit = 100000
							// Source failure may publish roots and refine assignments. Only the
							// invocation's retained locals are cleaned; retry gets an emptied output.
							serviceMust(t, c.CleanupRetained())
							serviceMust(t, roots.Reset())
							serviceMust(t, c.IsolateMultivariateRoots(pm, p, v, roots))
							if len(roots.values) != tc.count {
								t.Fatal("retry root count", len(roots.values))
							}
							if tc.count == 1 {
								cmp, e := c.Compare(&roots.values[0], &v.values[0])
								serviceMust(t, e)
								if cmp != 0 {
									t.Fatal("retry root", cmp)
								}
							}
							if tc.count == 2 {
								algebraicExpectBetween(t, c, &roots.values[0], -2, -1)
								algebraicExpectBetween(t, c, &roots.values[1], 1, 2)
							}
						})
						if !passed {
							return
						}
					}
					if !succeeded {
						t.Fatal("checkpoint horizon did not reach success")
					}
				})
			}
		})
	}
}
