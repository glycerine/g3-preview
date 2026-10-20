// Source: src/math/polynomial/algebraic_numbers.cpp:1629-1640.
package polynomial

import (
	"errors"
	"math"
	"strconv"
	"testing"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func algebraicInverseExpectNoPendingAllocation(t *testing.T, z *unumeral.IntegerManager) {
	t.Helper()
	blocks, e := z.SourceAllocator().PendingRoots()
	serviceMust(t, e)
	if z.PendingAllocationObject() != (umemory.ObjectRef{}) || blocks != ([3]umemory.Block{}) {
		t.Fatal("inverse left unnamed numeric allocation", z.PendingAllocationObject(), blocks)
	}
}

func TestAlgebraicInverseBasicOwnsEveryNumericAllocationPrefix(t *testing.T) {
	succeeded, owned := false, false
	for prefix := int64(0); prefix < 64 && !succeeded; prefix++ {
		t.Run(strconv.FormatInt(prefix, 10), func(t *testing.T) {
			c, _ := algebraicCellSourceFixtureProfile(t, true)
			qm := c.config.Rationals
			q := unumeral.NewRational()
			serviceMust(t, qm.SetInt32(&q, math.MinInt32))
			var a Algebraic
			serviceMust(t, c.SetRational(&a, &q))
			qm.Budget().FailAfter(prefix)
			body := c.Inv(&a)
			qm.Budget().FailAfter(-1)
			succeeded = body == nil
			if body != nil && !numericPhysicalReservationRefusal(body) {
				t.Fatal("basic inverse left actual allocation path", body)
			}
			if c.inversion.numericPending(c) {
				owned = true
				if c.inversion.owner != c || c.inversion.numericHooks.Policy != &c.inversion {
					t.Fatal("pending inverse allocation lost concrete caller identity")
				}
				if e := c.Del(&a); !errors.Is(e, ErrAlgebraicPending) {
					t.Fatal("pending inverse allocation admitted value deletion", e)
				}
			}
			serviceMust(t, c.CleanupRetained())
			if c.inversion.numericPending(c) {
				t.Fatal("inverse owning cleanup retained its numeric root")
			}
			algebraicInverseExpectNoPendingAllocation(t, qm.Integers())
			serviceMust(t, c.Del(&a))
			serviceMust(t, qm.Del(&q))
		})
	}
	if !succeeded || !owned {
		t.Fatal("basic inverse prefix sweep missed success or owned allocation", succeeded, owned)
	}
}
