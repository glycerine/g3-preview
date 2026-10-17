package polynomial

import (
	"errors"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"strconv"
	"testing"
)

func TestAlgebraicRationalSetCopyAllocationPrefixes(t *testing.T) {
	for _, existing := range []bool{false, true} {
		t.Run(map[bool]string{false: "new", true: "existing"}[existing], func(t *testing.T) {
			complete, issued := false, false
			for ordinal := int64(0); ordinal < 256 && !complete; ordinal++ {
				t.Run(strconv.FormatInt(ordinal, 10), func(t *testing.T) {
					c, _, _ := algebraicCompareFixture(t)
					q := unumeral.NewRational()
					serviceMust(t, c.config.Rationals.SetString(&q, "10000000000000000000000000000000000000001/30000000000000000000000000000000000000001"))
					before := q
					var out [32]Algebraic
					if existing {
						one := unumeral.NewInt(1)
						for i := range out {
							serviceMust(t, c.SetInteger(&out[i], &one))
						}
					}
					z := c.config.Rationals.Integers()
					z.Budget().FailAfter(ordinal)
					var body error
					for i := range out {
						body = c.SetRational(&out[i], &q)
						if body != nil {
							break
						}
					}
					z.Budget().FailAfter(-1)
					if q != before {
						t.Fatal("const setter consumed input")
					}
					if body != nil && !numericPhysicalReservationRefusal(body) {
						t.Fatal("unexpected copy refusal", body)
					}
					if z.MatchesCopyAllocationInvocation(&c.rationalCopy) {
						issued = true
						var untouched Algebraic
						if e := c.SetRational(&untouched, &q); !errors.Is(e, ErrAlgebraicPending) || untouched != (Algebraic{}) {
							t.Fatal("replacement lost retained receipt", e)
						}
					}
					serviceMust(t, c.CleanupRetained())
					for i := range out {
						serviceMust(t, c.CleanupValue(&out[i]))
					}
					serviceMust(t, c.SetRational(&out[0], &q))
					eq, e := c.EqualRational(&out[0], &q)
					serviceMust(t, e)
					if !eq {
						t.Fatal("retry changed rational")
					}
					serviceMust(t, c.Del(&out[0]))
					serviceMust(t, c.config.Rationals.Del(&q))
					complete = body == nil
				})
				if t.Failed() {
					return
				}
			}
			if !complete || !issued {
				t.Fatal("missing completion/actual retained copy frontier", complete, issued)
			}
		})
	}
}
