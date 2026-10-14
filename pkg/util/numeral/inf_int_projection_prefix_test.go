// Source: src/util/inf_int_rational.h:144-152.
package numeral

import (
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestInfIntProjectionPhysicalFailurePrefixes(t *testing.T) {
	num := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 160), big.NewInt(5)).String()
	den := new(big.Int).Add(new(big.Int).Lsh(big.NewInt(1), 150), big.NewInt(3)).String()
	for _, tc := range []struct {
		name string
		den  bool
		want string
	}{
		{"numerator", false, num},
		{"denominator", true, den},
	} {
		t.Run(tc.name, func(t *testing.T) {
			failures, success := 0, false
			for prefix := int64(0); prefix < 128; prefix++ {
				budget := arena.NewBudget(1 << 24)
				config := serviceConfig(budget)
				config.Integers.Profile = umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
				s, e := NewRationalService(config)
				serviceMust(t, e)
				src := infIntValue(t, s, num+"/"+den, 0)
				budget.FailAfter(prefix)
				var out OwnedInfIntRational
				if tc.den {
					out, e = s.InfIntDenominator(&src)
				} else {
					out, e = s.InfIntNumerator(&src)
				}
				serviceAuditRationalRoots(t, s, &src.First, &out.First)
				if e != nil {
					if !numericPhysicalReservationRefusal(e) || out.ownerID != 0 {
						t.Fatal(prefix, e, out.ownerID)
					}
					failures++
				} else {
					success = true
				}
				budget.FailAfter(-1)
				serviceMust(t, s.Manager.Z.CleanupPendingAllocation())
				if success {
					infIntCheck(t, s, &out, tc.want, 0)
					serviceMust(t, s.InfIntDel(&out))
				}
				infIntCheck(t, s, &src, num+"/"+den, 0)
				serviceMust(t, s.InfIntDel(&src))
				serviceMust(t, s.Close())
				if budget.Used() != 0 {
					t.Fatal(prefix, budget.Used())
				}
				if success {
					break
				}
			}
			if !success || failures == 0 {
				t.Fatal("missing refusal/success boundary", failures, success)
			}
		})
	}
}
