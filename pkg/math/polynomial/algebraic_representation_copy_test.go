package polynomial

import (
	"errors"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"strconv"
	"testing"
)

func TestAlgebraicDefiningPolynomialCopyReceiptsAndRetry(t *testing.T) {
	for _, root := range []bool{false, true} {
		t.Run(map[bool]string{false: "rational", true: "root"}[root], func(t *testing.T) {
			success, issued := false, false
			for ordinal := int64(0); ordinal < 256 && !success; ordinal++ {
				t.Run(strconv.FormatInt(ordinal, 10), func(t *testing.T) {
					c, p, _ := algebraicCompareFixture(t)
					z, qm := c.config.Rationals.Integers(), c.config.Rationals
					var a Algebraic
					var want []string
					if root {
						coeff := []unumeral.Int{unumeral.NewInt(0), unumeral.NewInt(0), unumeral.NewInt(0)}
						serviceMust(t, z.SetString(&coeff[0], "-20000000000000000000000000000000000000001"))
						serviceMust(t, z.SetString(&coeff[2], "10000000000000000000000000000000000000000"))
						poly, e := p.NewPolynomial()
						serviceMust(t, e)
						serviceMust(t, p.SetInts(poly, coeff))
						view, e := p.wholeView(poly)
						serviceMust(t, e)
						lower, upper := unumeral.NewBinaryInt(1), unumeral.NewBinaryInt(2)
						serviceMust(t, c.SetRoot(&a, view, &lower, &upper, true))
						serviceMust(t, p.DelPolynomial(&poly))
						for i := range coeff {
							serviceMust(t, z.Del(&coeff[i]))
						}
						want = []string{"-20000000000000000000000000000000000000001", "0", "10000000000000000000000000000000000000000"}
					} else {
						q := unumeral.NewRational()
						serviceMust(t, qm.SetString(&q, "10000000000000000000000000000000000000001/30000000000000000000000000000000000000001"))
						serviceMust(t, c.SetRational(&a, &q))
						serviceMust(t, qm.Del(&q))
						want = []string{"-10000000000000000000000000000000000000001", "30000000000000000000000000000000000000001"}
					}
					original := a
					var out [32]UPolynomial
					for i := range out {
						v, e := p.NewPolynomial()
						serviceMust(t, e)
						out[i] = v
					}
					z.Budget().FailAfter(ordinal)
					var body error
					for i := range out {
						body = c.GetPolynomial(&a, out[i])
						if body != nil {
							break
						}
					}
					z.Budget().FailAfter(-1)
					if a != original {
						t.Fatal("getter replaced source value")
					}
					if body != nil && !numericPhysicalReservationRefusal(body) {
						t.Fatal("unexpected refusal", body)
					}
					if z.MatchesCopyAllocationInvocation(&c.representationCopy) {
						issued = true
						if e := c.GetPolynomial(&a, out[0]); !errors.Is(e, ErrAlgebraicPending) {
							t.Fatal("replacement lost exact receipt", e)
						}
					}
					serviceMust(t, c.CleanupRetained())
					for i := range out {
						serviceMust(t, p.DelPolynomial(&out[i]))
					}
					retry, e := p.NewPolynomial()
					serviceMust(t, e)
					serviceMust(t, c.GetPolynomial(&a, retry))
					view, e := p.wholeView(retry)
					serviceMust(t, e)
					if int(view.count) != len(want) {
						t.Fatal("defining polynomial size", view.count)
					}
					for i, text := range want {
						value, e := p.read(view, uint32(i))
						serviceMust(t, e)
						got, e := z.String(value)
						serviceMust(t, e)
						if got != text {
							t.Fatal("independent defining coefficient", i, got, text)
						}
					}
					serviceMust(t, p.DelPolynomial(&retry))
					serviceMust(t, c.Del(&a))
					success = body == nil
				})
				if t.Failed() {
					return
				}
			}
			if !success || !issued {
				t.Fatal("missing success/actual copy receipt frontier", success, issued)
			}
		})
	}
}
