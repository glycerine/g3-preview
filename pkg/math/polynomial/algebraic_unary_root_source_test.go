package polynomial

import (
	"io"
	"strconv"
	"testing"

	"github.com/glycerine/g3/pkg/util/diagnostic"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestAlgebraicUnaryRootTraceMultiwordReservationPrefixes(t *testing.T) {
	succeeded, refused, rootRAW := false, false, false
	for prefix := int64(0); prefix < 8192 && !succeeded; prefix++ {
		if !t.Run(strconv.FormatInt(prefix, 10), func(t *testing.T) {
			c, _, _ := algebraicCompareFixtureProfile(t, true)
			c.factor = false
			a := algebraicLargeDyadicPromotionRoot(t, c)
			var out Algebraic
			tags := diagnostic.NewTraceTagsWithWriter(io.Discard)
			tags.Enable("algebraic")
			serviceMust(t, c.config.Rationals.BindTrace(tags))
			budget := c.config.Rationals.Budget()
			budget.FailAfter(prefix)
			body := c.Root(&a, 3, &out)
			budget.FailAfter(-1)
			if body == nil {
				succeeded = true
				algebraicExpectBetween(t, c, &out, 1, 2)
			} else {
				refused = true
				if !numericPhysicalReservationRefusal(body) {
					t.Fatal("root trace failed outside an actual reservation", body)
				}
			}
			serviceMust(t, c.config.Rationals.BindTrace(nil))
			for i := range c.binary.intervalRoots {
				child := &c.binary.intervalRoots[i]
				if child.Pending() && child.SourceCause() == nil {
					rootRAW = true
					if c.binary.intervalConstructed || c.binary.saved[0].owned {
						t.Fatal("unjoined root RAW protected older completed interval locals")
					}
				}
			}
			serviceMust(t, c.CleanupRetained())
			if c.binary.active || c.rootTracePending() || c.coefficientNormalizationPending() {
				t.Fatal("root cleanup retained an accepted child")
			}
			probe := unumeral.NewInt(1)
			if _, e := c.config.Rationals.Integers().RootInPlace(&probe, 2); e != nil {
				t.Fatalf("actual integer root owner not ready after unary cleanup: %v; original: %v", e, body)
			}
			serviceMust(t, c.Del(&out))
			serviceMust(t, c.Del(&a))
		}) {
			return
		}
	}
	if !succeeded || !refused || !rootRAW {
		t.Fatal("root trace sweep missed refusal, success or RAW", succeeded, refused, rootRAW)
	}
}
