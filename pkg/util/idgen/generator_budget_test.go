package idgen

// Source: src/util/id_gen.h; src/util/vector.h.

import (
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestIDGeneratorNativeFreeListAndReset(t *testing.T) {
	b := arena.NewBudget(0)
	g := New(7, nil)
	b.FailAfter(0)
	if g.Mk() != 7 {
		t.Fatal("fresh ID")
	}
	for i := uint32(0); i < 129; i++ {
		if e := g.Recycle(i); e != nil {
			t.Fatal("native free-list growth", i, e)
		}
	}
	if g.FreeLen() != 129 || b.Used() != 0 {
		t.Fatal("native storage charged arena", g.FreeLen(), b.Used())
	}
	for i := uint32(129); i > 0; i-- {
		if got := g.Mk(); got != i-1 {
			t.Fatal("LIFO ID reuse", got, i-1)
		}
	}
	capacity := g.FreeCapacity()
	g.Reset(40)
	if g.FreeLen() != 0 || g.FreeCapacity() != capacity || g.Mk() != 40 {
		t.Fatal("reset retained slice", g.FreeLen(), g.FreeCapacity(), capacity)
	}
	g.Cleanup(50)
	if g.free != nil || g.Mk() != 50 || b.Used() != 0 {
		t.Fatal("cleanup", g.free, b.Used())
	}
}

func TestIDGeneratorNativeReserveAndOOMSkip(t *testing.T) {
	b := arena.NewBudget(0)
	oom := true
	g := New(math.MaxUint32, func() bool { return oom })
	b.FailAfter(0)
	if e := g.Recycle(4); e != nil || g.FreeLen() != 0 {
		t.Fatal("OOM skips recycled ID", e, g.FreeLen())
	}
	if g.Mk() != math.MaxUint32 || g.Mk() != 0 {
		t.Fatal("source ID wrap")
	}
	oom = false
	if e := g.Reserve(129); e != nil || g.FreeCapacity() < 129 || g.FreeLen() != 0 || b.Used() != 0 {
		t.Fatal("native reserve", e, g.FreeCapacity(), g.FreeLen(), b.Used())
	}
	if e := g.Recycle(4); e != nil || g.Mk() != 4 {
		t.Fatal("reserved recycle", e)
	}
	g.Cleanup(0)
}
