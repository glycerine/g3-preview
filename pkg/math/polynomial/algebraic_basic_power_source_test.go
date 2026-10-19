package polynomial

import (
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
	"math/big"
	"strconv"
	"testing"
)

func TestAlgebraicBasicPowerEveryReservation(t *testing.T) {
	n, _ := new(big.Int).SetString("18446744073709551619", 10)
	expected := new(big.Int).Exp(n, big.NewInt(3), nil).String() + "/343"
	succeeded, refused, raw := false, false, false
	for prefix := int64(0); prefix < 8192 && !succeeded; prefix++ {
		if !t.Run(strconv.FormatInt(prefix, 10), func(t *testing.T) {
			c, _, _ := algebraicCompareFixtureProfile(t, true)
			q := unumeral.NewRational()
			serviceMust(t, c.config.Rationals.SetString(&q, "18446744073709551619/7"))
			var a, out Algebraic
			serviceMust(t, c.SetRational(&a, &q))
			budget := c.config.Rationals.Budget()
			budget.FailAfter(prefix)
			body := c.Power(&a, 3, &out)
			budget.FailAfter(-1)
			if body == nil {
				succeeded = true
			} else {
				refused = true
				if !numericPhysicalReservationRefusal(body) {
					t.Fatal("power failed outside actual reservation", body)
				}
			}
			if c.basicRootScope.power.Pending() && c.basicRootScope.power.SourceCause() == nil {
				raw = true
				if c.basicRootScope.qConstructed {
					t.Fatal("unjoined power RAW protected older completed rational")
				}
			}
			if e := c.CleanupRetained(); e != nil {
				t.Fatalf("power cleanup: %v; original: %v", e, body)
			}
			if c.basicRootScope.active || c.config.Rationals.RawArithmetic.PowerActive || c.config.Rationals.RawArithmetic.NormActive {
				t.Fatalf("power retained actual child after cleanup; original: %v", body)
			}
			if body == nil {
				serviceMust(t, c.config.Rationals.SetString(&q, expected))
				cmp, e := c.CompareRational(&out, &q)
				serviceMust(t, e)
				if cmp != 0 {
					t.Fatal("independent rational cube", cmp)
				}
			}
			serviceMust(t, c.Del(&out))
			serviceMust(t, c.Del(&a))
			serviceMust(t, c.config.Rationals.Del(&q))
		}) {
			return
		}
	}
	if !succeeded || !refused || !raw {
		t.Fatal("power sweep missed refusal, success or actual RAW", succeeded, refused, raw)
	}
}
