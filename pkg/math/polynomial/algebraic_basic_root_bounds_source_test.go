package polynomial

import (
	"strconv"
	"testing"

	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestAlgebraicBasicRootNonexactBoundsEveryReservation(t *testing.T) {
	for _, negative := range []bool{false, true} {
		name := "positive"
		if negative {
			name = "negative"
		}
		t.Run(name, func(t *testing.T) {
			succeeded, refused := false, false
			for prefix := int64(0); prefix < 8192 && !succeeded; prefix++ {
				if !t.Run(strconv.FormatInt(prefix, 10), func(t *testing.T) {
					c, _, _ := algebraicCompareFixtureProfile(t, true)
					q := unumeral.NewRational()
					input := "18446744073709551619/3"
					degree := uint32(2)
					bounds := []string{"2147483648", "4294967296"}
					if negative {
						input = "-18446744073709551619/3"
						degree = 3
						bounds = []string{"-4194304", "-1048576"}
					}
					serviceMust(t, c.config.Rationals.SetString(&q, input))
					var a, out Algebraic
					serviceMust(t, c.SetRational(&a, &q))
					budget := c.config.Rationals.Budget()
					budget.FailAfter(prefix)
					body := c.Root(&a, degree, &out)
					budget.FailAfter(-1)
					if body == nil {
						succeeded = true
					} else {
						refused = true
						if !numericPhysicalReservationRefusal(body) {
							t.Fatal("nonexact root failed outside actual reservation", body)
						}
					}
					if e := c.CleanupRetained(); e != nil {
						t.Fatalf("nonexact root cleanup: %v; original: %v", e, body)
					}
					algebraicExpectNoPendingAllocation(t, c.config.Rationals.Integers())
					if c.basicRootScope.active || c.coefficientNormalizationPending() {
						t.Fatal("nonexact root retained an accepted child")
					}
					zero := unumeral.NewBinaryInt(0)
					if _, e := c.config.Binary.IsZero(&zero); e != nil {
						t.Fatalf("actual bound owner remained pending after root cleanup: %v; original: %v", e, body)
					}
					if body == nil {
						for i, bound := range bounds {
							serviceMust(t, c.config.Rationals.SetString(&q, bound))
							cmp, e := c.CompareRational(&out, &q)
							serviceMust(t, e)
							if i == 0 && cmp <= 0 || i == 1 && cmp >= 0 {
								t.Fatal("independent root bounds", bound, cmp)
							}
						}
					}
					serviceMust(t, c.Del(&out))
					serviceMust(t, c.Del(&a))
					serviceMust(t, c.config.Rationals.Del(&q))
				}) {
					return
				}
			}
			if !succeeded || !refused {
				t.Fatal("nonexact root reservation sweep missed refusal or success", succeeded, refused)
			}

		})
	}
}
