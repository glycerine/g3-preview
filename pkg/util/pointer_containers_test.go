// Source: src/util/buffer.h and vector.h.
package util

import (
	"github.com/glycerine/g3/pkg/arena"
	"testing"
)

func TestPointerBufferSourceSpillRefusal(t *testing.T) {
	b := arena.NewBudget(1 << 16)
	var v PointerBuffer[uint32]
	for i := uint32(0); i < 16; i++ {
		serviceMust(t, v.Push(b, i))
	}
	if b.Used() != 0 {
		t.Fatal("inline storage charged", b.Used())
	}
	b.FailAfter(0)
	if e := v.Push(b, 16); !numericStorageReservationRefusal(e) {
		t.Fatal("spill refusal", e)
	}
	if len(v.Slice()) != 16 || b.Used() != 0 {
		t.Fatal("failed spill changed storage")
	}
	b.FailAfter(-1)
	serviceMust(t, v.Push(b, 16))
	if cap(v.Slice()) != 32 || b.Used() != 128 {
		t.Fatal("source doubled capacity", cap(v.Slice()), b.Used())
	}
	for i, n := range v.Slice() {
		if n != uint32(i) {
			t.Fatal("lost value", i, n)
		}
	}
	v.Close(b)
	if b.Used() != 0 || len(v.Slice()) != 0 {
		t.Fatal("source destructor retained spill")
	}
}
func TestPointerVectorSourceGrowthAndErase(t *testing.T) {
	b := arena.NewBudget(1 << 16)
	var v PointerVector[uint32]
	for i, capacity := range []int{2, 2, 3, 5, 5, 8} {
		serviceMust(t, v.Push(b, uint32(i)))
		if cap(v.Slice()) != capacity || b.Used() != uint64(capacity)*4 {
			t.Fatal("source capacity", i, cap(v.Slice()), b.Used())
		}
	}
	v.Erase(2)
	expected := []uint32{0, 1, 3, 4, 5}
	for i, n := range v.Slice() {
		if n != expected[i] {
			t.Fatal("source erase", i, n)
		}
	}
	v.Close(b)
	if b.Used() != 0 || len(v.Slice()) != 0 {
		t.Fatal("source destructor retained vector")
	}
}
