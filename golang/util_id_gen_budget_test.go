// Source: src/util/id_gen.h; src/util/vector.h.
package g3

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"math"
	"testing"
)

func idBudgetMust(t *testing.T, e error) {
	t.Helper()
	if e != nil {
		t.Fatal(e)
	}
}

func TestIDGeneratorBudgetSourceGrowthAndReset(t *testing.T) {
	b := arena.NewBudget(4096)
	g := newBudgetedUtilIDGen(7, b, nil)
	// The generator is a value embedded in its caller's cold owner, whose header
	// charge includes this metadata. Only its separately allocated array is here.
	if b.Used() != 0 || g.mk() != 7 {
		t.Fatal("constructor allocated a free list")
	}
	for i, capacity := range []int{2, 2, 3, 5, 5, 8, 8, 8, 12} {
		idBudgetMust(t, g.recycle(uint32(i)))
		if cap(g.free) != capacity || b.Used() != uint64(capacity)*4 {
			t.Fatal(i, cap(g.free), b.Used())
		}
	}
	for i := uint32(9); i > 0; i-- {
		if g.mk() != i-1 {
			t.Fatal("source LIFO changed")
		}
	}
	g.reset(40)
	if len(g.free) != 0 || cap(g.free) != 12 || b.Used() != 48 || g.mk() != 40 {
		t.Fatal("reset must retain physical capacity")
	}
	g.cleanup(50)
	if g.free != nil || b.Used() != 0 || g.mk() != 50 {
		t.Fatal("cleanup must refund physical array")
	}
	g.cleanup(60)
}

func TestIDGeneratorBudgetExactOverlapAndFailedPublication(t *testing.T) {
	for _, limit := range []uint64{19, 20} {
		b := arena.NewBudget(limit)
		g := newBudgetedUtilIDGen(17, b, nil)
		idBudgetMust(t, g.recycle(10))
		idBudgetMust(t, g.recycle(11))
		hash := g.showHash()
		e := g.recycle(12) // old2*4 + new3*4 = twenty bytes, not twelve.
		if limit == 19 {
			if !errors.Is(e, arena.ErrBudget) || len(g.free) != 2 || cap(g.free) != 2 || b.Used() != 8 || g.free[0] != 10 || g.free[1] != 11 || g.next != 17 {
				t.Fatal("failed growth published state", e, g.free, b.Used())
			}
			if g.showHash() != hash {
				t.Fatal("failed recycle changed source hash")
			}
		} else {
			idBudgetMust(t, e)
			if len(g.free) != 3 || cap(g.free) != 3 || b.Used() != 12 || g.mk() != 12 {
				t.Fatal("successful growth charge/order")
			}
		}
		g.cleanup(0)
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	}
}

func TestIDGeneratorBudgetReservationPrefixesAndOOMSkip(t *testing.T) {
	for prefix, capacity := range []int{0, 2, 3, 5, 8, 12} {
		b := arena.NewBudget(4096)
		g := newBudgetedUtilIDGen(7, b, nil)
		b.FailAfter(int64(prefix))
		e := g.reserveFreeIDs(10)
		if prefix < 5 && !errors.Is(e, arena.ErrInjectedFailure) || prefix == 5 && e != nil {
			t.Fatal(prefix, e)
		}
		// Check the complete immediately retained graph before diagnostics/cleanup.
		if len(g.free) != 0 || cap(g.free) != capacity || b.Used() != uint64(capacity)*4 || g.next != 7 {
			t.Fatal("reservation prefix", prefix, cap(g.free), b.Used())
		}
		b.FailAfter(-1)
		idBudgetMust(t, g.reserveFreeIDs(10))
		if cap(g.free) != 12 || b.Used() != 48 {
			t.Fatal("retry did not reuse published capacity")
		}
		g.cleanup(0)
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	}
	b := arena.NewBudget(0)
	oom := true
	g := newBudgetedUtilIDGen(math.MaxUint32, b, func() bool { return oom })
	b.FailAfter(0)
	idBudgetMust(t, g.recycle(4))
	if g.free != nil || b.Used() != 0 || g.mk() != math.MaxUint32 || g.mk() != 0 {
		t.Fatal("source OOM skip/wrap")
	}
	oom = false
	if e := g.recycle(4); !errors.Is(e, arena.ErrBudget) || g.free != nil {
		t.Fatal("regular recycle skipped budget", e)
	}
	g.cleanup(0)
}

func TestIDGeneratorBudgetWarmEmbeddedValueAndCapacityBounds(t *testing.T) {
	b := arena.NewBudget(4096)
	g := newBudgetedUtilIDGen(10, b, nil)
	idBudgetMust(t, g.reserveFreeIDs(64))
	baseline := b.Used()
	b.FailAfter(0)
	var failure error
	var ids [64]uint32
	if allocations := testing.AllocsPerRun(100, func() {
		for i := range ids {
			ids[i] = g.mk()
		}
		for _, id := range ids {
			if failure == nil {
				failure = g.recycle(id)
			}
		}
	}); failure != nil || allocations != 0 || b.Used() != baseline {
		t.Fatal("warm free list", allocations, failure, b.Used(), baseline)
	}
	b.FailAfter(-1)
	g.cleanup(0)
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
	for _, old := range []uint32{0, 2, 3, 5, 100, 0x40000000, 0x55555555, math.MaxUint32 / 2, math.MaxUint32} {
		// Source SZ unsigned expressions wrap before the capacity shift and
		// byte comparisons; the old wider oracle accepted source overflow.
		want := ((uint64(old)*3 + 1) & math.MaxUint32) / 2
		oldBytes := (uint64(old)*4 + 8) & math.MaxUint32
		newBytes := (want*4 + 8) & math.MaxUint32
		if old == 0 {
			want = 2
		}
		capacity, e := idFreeGrowth(old)
		if (old != 0 && (want <= uint64(old) || newBytes <= oldBytes)) || want*4 > uint64(^uint(0)>>1) {
			if !errors.Is(e, arena.ErrBounds) {
				t.Fatal("capacity overflow accepted", old, capacity, e)
			}
		} else if e != nil || uint64(capacity) != want {
			t.Fatal(old, capacity, want, e)
		}
	}
}
