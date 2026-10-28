package polynomial

// Source: upolynomial.cpp:2797-2807; concrete retained refinement calls.

import (
	"errors"
	"fmt"
	"testing"

	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestAlgebraicRefinementNumericContextRejectsReplacementInvocation(t *testing.T) {
	for _, bounds := range []bool{false, true} {
		t.Run(fmt.Sprint("bounds=", bounds), func(t *testing.T) {
			c, p, _ := algebraicCompareFixture(t)
			root := algebraicCompareRoot(t, c, []int32{-2, 0, 1}, 1, 0, 2, 0, true)
			defer func() { serviceMust(t, c.Del(&root)) }()
			q, z := c.config.Rationals, c.config.Rationals.Integers()
			first, second := unumeral.NewRational(), unumeral.NewRational()
			defer func() { serviceMust(t, q.Del(&first)); serviceMust(t, q.Del(&second)) }()
			var old unumeral.NumericAllocationContext
			var invocation uint64
			captured, probe, checked := false, false, false
			prior := p.config.Checkpoint
			defer func() { p.config.Checkpoint = prior }()
			p.config.Checkpoint = func() bool {
				if !prior() {
					return false
				}
				s := &c.refinement
				if bounds {
					s = &c.bounds.refinement
				}
				// This is the actual precision loop's checkpoint, after width construction.
				if !s.executing || s.live&(1<<boundRefinementWidth) == 0 {
					return true
				}
				if !probe && !captured {
					captured = true
					old = s.context()
					invocation = s.invocation
				}
				if probe && !checked {
					checked = true
					if invocation == s.invocation {
						t.Error("replacement did not enter a new refinement")
					}
					a, b, out := unumeral.NewInt(2), unumeral.NewInt(3), unumeral.NewInt(99)
					values, live, used, stats := s.values, s.live, z.Budget().Used(), z.RuntimeOwner().Stats()
					e := z.WithAllocationContext(old).Add(&a, &b, &out)
					if !errors.Is(e, unumeral.ErrNumericOwner) || out != unumeral.NewInt(99) {
						t.Error("stale context acquired replacement refinement", e, out)
					}
					if values != s.values || live != s.live || used != z.Budget().Used() || stats != z.RuntimeOwner().Stats() {
						t.Error("stale context changed actual refinement owners")
					}
					e = z.WithAllocationContext(s.context()).Add(&a, &b, &out)
					if e != nil || out != unumeral.NewInt(5) {
						t.Error("actual current context refused valid arithmetic", e, out)
					}
				}
				return true
			}
			if bounds {
				serviceMust(t, c.GetLowerRationalAtPrecision(&root, &first, 0))
			} else {
				serviceMust(t, c.RefineToPrecision(&root, 1))
			}
			if !captured {
				t.Fatal("actual first refinement checkpoint not reached")
			}
			probe = true
			if bounds {
				serviceMust(t, c.GetUpperRationalAtPrecision(&root, &second, 0))
				a, e := q.String(&first)
				serviceMust(t, e)
				b, e := q.String(&second)
				serviceMust(t, e)
				if a != "1" || b != "3/2" {
					t.Error("copied precision bounds", a, b)
				}
			} else {
				serviceMust(t, c.RefineToPrecision(&root, 1))
			}
			if !checked {
				t.Fatal("actual replacement refinement checkpoint not reached")
			}
		})
	}
}
