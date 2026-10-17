package polynomial

import (
	"errors"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"strconv"
	"testing"
)

func algebraicBigBinaryBoundsRoot(t *testing.T, c *AlgebraicRootCore) Algebraic {
	t.Helper()
	p, z, b := c.config.Polynomials, c.config.Rationals.Integers(), c.config.Binary
	poly, e := p.NewPolynomial()
	serviceMust(t, e)
	serviceMust(t, p.SetInts(poly, []unumeral.Int{unumeral.NewInt(-2), unumeral.NewInt(0), unumeral.NewInt(1)}))
	view, e := p.wholeView(poly)
	serviceMust(t, e)
	var lower, upper unumeral.BinaryRational
	n := unumeral.NewInt(0)
	serviceMust(t, z.SetString(&n, "26087635650665564424"))
	serviceMust(t, b.SetIntegerPower(&lower, &n, 64))
	serviceMust(t, z.SetString(&n, "26087635650665564425"))
	serviceMust(t, b.SetIntegerPower(&upper, &n, 64))
	var root Algebraic
	serviceMust(t, c.SetRoot(&root, view, &lower, &upper, true))
	serviceMust(t, b.Del(&upper))
	serviceMust(t, b.Del(&lower))
	serviceMust(t, z.Del(&n))
	serviceMust(t, p.DelPolynomial(&poly))
	return root
}

func TestAlgebraicBinaryBoundCopyActualReservationAndExponentPublication(t *testing.T) {
	for _, upper := range []bool{false, true} {
		t.Run(map[bool]string{false: "lower", true: "upper"}[upper], func(t *testing.T) {
			success, issued := false, false
			for ordinal := int64(0); ordinal < 256 && !success; ordinal++ {
				t.Run(strconv.FormatInt(ordinal, 10), func(t *testing.T) {
					c, _, _ := algebraicCompareFixture(t)
					root := algebraicBigBinaryBoundsRoot(t, c)
					z, b := c.config.Rationals.Integers(), c.config.Binary
					var out [32]unumeral.BinaryRational
					z.Budget().FailAfter(ordinal)
					var body error
					last := 0
					for i := range out {
						last = i
						if upper {
							body = c.GetUpper(&root, &out[i])
						} else {
							body = c.GetLower(&root, &out[i])
						}
						if body != nil {
							break
						}
					}
					z.Budget().FailAfter(-1)
					if body != nil {
						if !numericPhysicalReservationRefusal(body) {
							t.Fatal("unexpected copy refusal", body)
						}
						if out[last].K() != 0 {
							t.Fatal("failed numerator copy published exponent", out[last].K())
						}
					}
					if z.MatchesCopyAllocationInvocation(&c.bounds.copy) {
						issued = true
						if e := c.GetLower(&root, &out[0]); !errors.Is(e, ErrAlgebraicPending) {
							t.Fatal("replacement lost copy receipt", e)
						}
					}
					serviceMust(t, c.CleanupRetained())
					for i := range out {
						serviceMust(t, b.Del(&out[i]))
					}
					var retry unumeral.BinaryRational
					if upper {
						serviceMust(t, c.GetUpper(&root, &retry))
					} else {
						serviceMust(t, c.GetLower(&root, &retry))
					}
					text, e := z.String(retry.BorrowNumerator())
					serviceMust(t, e)
					want, k := "3260954456333195553", uint32(61)
					if upper {
						want, k = "26087635650665564425", 64
					}
					if text != want || retry.K() != k {
						t.Fatal("independent copied dyadic bound", text, retry.K(), want, k)
					}
					serviceMust(t, b.Del(&retry))
					serviceMust(t, c.Del(&root))
					success = body == nil
				})
				if t.Failed() {
					return
				}
			}
			if !success || !issued {
				t.Fatal("missing success/actual copy allocation frontier", success, issued)
			}
		})
	}
}

func TestAlgebraicPrecisionBasicBoundCopyActualRetention(t *testing.T) {
	for _, upper := range []bool{false, true} {
		t.Run(map[bool]string{false: "lower", true: "upper"}[upper], func(t *testing.T) {
			success, issued := false, false
			for ordinal := int64(0); ordinal < 256 && !success; ordinal++ {
				t.Run(strconv.FormatInt(ordinal, 10), func(t *testing.T) {
					c, _, _ := algebraicCompareFixture(t)
					q := unumeral.NewRational()
					qm := c.config.Rationals
					serviceMust(t, qm.SetString(&q, "10000000000000000000000000000000000000001/30000000000000000000000000000000000000001"))
					var a Algebraic
					serviceMust(t, c.SetRational(&a, &q))
					var out [32]unumeral.Rational
					for i := range out {
						out[i] = unumeral.NewRational()
					}
					qm.Budget().FailAfter(ordinal)
					var body error
					for i := range out {
						if upper {
							body = c.GetUpperRationalAtPrecision(&a, &out[i], 17)
						} else {
							body = c.GetLowerRationalAtPrecision(&a, &out[i], 17)
						}
						if body != nil {
							break
						}
					}
					qm.Budget().FailAfter(-1)
					if body != nil && !numericPhysicalReservationRefusal(body) {
						t.Fatal("unexpected scalar bound refusal", body)
					}
					if qm.Integers().MatchesCopyAllocationInvocation(&c.bounds.copy) {
						issued = true
						if e := c.GetLowerRationalAtPrecision(&a, &out[0], 17); !errors.Is(e, ErrAlgebraicPending) {
							t.Fatal("replacement lost bound-copy receipt", e)
						}
					}
					serviceMust(t, c.CleanupRetained())
					for i := range out {
						serviceMust(t, qm.Del(&out[i]))
					}
					retry := unumeral.NewRational()
					if upper {
						serviceMust(t, c.GetUpperRationalAtPrecision(&a, &retry, 17))
					} else {
						serviceMust(t, c.GetLowerRationalAtPrecision(&a, &retry, 17))
					}
					equal, e := qm.Equal(&retry, &q)
					serviceMust(t, e)
					if !equal {
						t.Fatal("basic precision bound was rounded or changed")
					}
					serviceMust(t, qm.Del(&retry))
					serviceMust(t, c.Del(&a))
					serviceMust(t, qm.Del(&q))
					success = body == nil
				})
				if t.Failed() {
					return
				}
			}
			if !success || !issued {
				t.Fatal("missing scalar copy completion/issued frontier", success, issued)
			}
		})
	}
}
