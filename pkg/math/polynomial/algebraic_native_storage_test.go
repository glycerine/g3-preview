package polynomial

import (
	"errors"
	"github.com/glycerine/g3/pkg/util"
	"testing"
)

func TestAlgebraicNativeLocatorAndVectorOwnership(t *testing.T) {
	c, _ := algebraicCellSourceFixture(t)
	q := c.config.Rationals
	z := q.Integers()
	budget := q.Budget()
	const count = 4097
	handles := make([]Algebraic, count)
	t.Cleanup(func() {
		budget.FailAfter(-1)
		for i := range handles {
			serviceMust(t, c.Del(&handles[i]))
		}
	})
	var bigToken uint64
	for i := range handles {
		value := util.NewRational()
		if i == 0 {
			serviceMust(t, z.SetUint64(value.BorrowNumerator(), 1<<40|1))
			bigToken = value.BorrowNumerator().NativeCellToken()
		} else {
			serviceMust(t, q.SetFraction(&value, int64(i+2), 3))
		}
		serviceMust(t, c.SetTake(&handles[i], &value))
		serviceMust(t, q.Del(&value))
	}
	stale := handles[0]
	if len(c.locations) != count || c.callers != count {
		t.Fatal("native locator extent", len(c.locations), c.callers)
	}
	v, e := c.NewVector()
	serviceMust(t, e)
	defer func() { budget.FailAfter(-1); serviceMust(t, v.Close()) }()
	used, before, cells := budget.Used(), z.Memory.Stats(), z.StorageStats()
	budget.FailAfter(0)
	for i := range handles {
		serviceMust(t, v.pushTake(&handles[i]))
	}
	if v.Len() != count {
		t.Fatal(v.Len())
	}
	for i := range handles {
		if handles[i] != (Algebraic{}) {
			t.Fatal("take left duplicate owner", i)
		}
		a, e := v.Get(uint32(i))
		serviceMust(t, e)
		r := algebraicCellSourceRecord(t, c, &a)
		n, e := z.Int64(r.value.BorrowNumerator())
		serviceMust(t, e)
		d, e := z.Int64(r.value.BorrowDenominator())
		serviceMust(t, e)
		if i == 0 {
			if n != 1<<40|1 || d != 1 || r.value.BorrowNumerator().NativeCellToken() != bigToken {
				t.Fatal("first actual BIG root changed", n, d)
			}
		} else if n*3 != int64(i+2)*d {
			t.Fatal("independent rational", i, n, d)
		}
	}
	if budget.Used() != used || z.Memory.Stats() != before || z.StorageStats() != cells {
		t.Fatal("native vector growth changed numeric allocation")
	}
	serviceMust(t, v.Reset())
	if z.IsLiveCellToken(bigToken) || c.callers != 0 || v.Len() != 0 {
		t.Fatal("native vector reset retained cell ownership")
	}
	budget.FailAfter(-1)
	value := util.NewRational()
	serviceMust(t, q.SetFraction(&value, 7, 3))
	var fresh Algebraic
	serviceMust(t, c.SetTake(&fresh, &value))
	if fresh.slot != stale.slot || fresh.generation == stale.generation {
		t.Fatal("locator recycling", fresh, stale)
	}
	if _, e := c.location(&stale); !errors.Is(e, ErrAlgebraicOwner) {
		t.Fatal("stale locator accepted", e)
	}
	serviceMust(t, c.Del(&fresh))
	serviceMust(t, q.Del(&value))
}
