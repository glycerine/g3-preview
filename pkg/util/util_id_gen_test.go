package util

import (
	"math"
	"testing"
)

// Added tests: src/util/id_gen.h has no dedicated assertion suite. These finite
// expectations preserve source recycling, resource checks and debug ID resets.
func TestIDGeneratorSourceOrder(t *testing.T) {
	outOfMemory := false
	g := NewIDGen(10, func() bool { return outOfMemory })
	if g.Mk() != 10 || g.Mk() != 11 || g.IdRange() != 12 {
		t.Fatal("fresh IDs")
	}
	g.Recycle(10)
	g.Recycle(11)
	if g.Mk() != 11 || g.Mk() != 10 || g.Mk() != 12 {
		t.Fatal("source LIFO recycle")
	}
	outOfMemory = true
	g.Recycle(12)
	if g.Mk() != 13 {
		t.Fatal("OOM recycle must be ignored")
	}
	outOfMemory = false
	g.Recycle(13)
	g.Reset(40)
	if g.Mk() != 40 || len(g.Free) != 0 {
		t.Fatal("reset")
	}
	g.Recycle(40)
	g.Cleanup(50)
	if g.Free != nil || g.Mk() != 50 {
		t.Fatal("cleanup")
	}
}
func TestIDGeneratorDebugAndWrap(t *testing.T) {
	g := NewIDGen(0, nil)
	for _, v := range []uint32{3, 4, 6} {
		g.Recycle(v)
	}
	if got := g.SetNextID(3); got != 5 || g.IdRange() != 5 {
		t.Fatal(got)
	}
	if g.DisplayFreeIDs() != "3 4 6" {
		t.Fatal(g.DisplayFreeIDs())
	}
	if g.Mk() != 6 || g.Mk() != 4 || g.Mk() != 3 || g.Mk() != 5 {
		t.Fatal("debug range should leave free IDs intact")
	}
	g.Reset(math.MaxUint32)
	if g.Mk() != math.MaxUint32 || g.Mk() != 0 {
		t.Fatal("source unsigned wrap")
	}
	g.Recycle(1)
	g.Recycle(1)
	if g.Mk() != 1 || g.Mk() != 1 {
		t.Fatal("source does not deduplicate recycle")
	}
}
func TestIDGeneratorHash(t *testing.T) {
	g := NewIDGen(17, nil)
	if got, want := g.ShowHash(), UintPair(String("", 17), 17); got != want {
		t.Fatal(got, want)
	}
	g.Recycle(0x04030201)
	g.Recycle(0xfefdfcfb)
	// Construct bytes independently of the generator's uint32 free-ID storage.
	bytes := string([]byte{1, 2, 3, 4, 0xfb, 0xfc, 0xfd, 0xfe})
	if got, want := g.ShowHash(), UintPair(String(bytes, 17), 17); got != want {
		t.Fatal(got, want)
	}
}
func TestIDGeneratorFiniteUniqueness(t *testing.T) {
	g := NewIDGen(0, nil)
	live := map[uint32]bool{}
	state := uint32(1)
	for i := 0; i < 3000; i++ {
		state = state*1664525 + 1013904223
		if len(live) < 64 && (state&3) != 0 {
			id := g.Mk()
			if live[id] {
				t.Fatal("duplicate live ID")
			}
			live[id] = true
			continue
		}
		// Pick deterministically by value, independent of Go map traversal.
		for id := uint32(0); id < g.IdRange(); id++ {
			if live[id] {
				delete(live, id)
				g.Recycle(id)
				break
			}
		}
	}
	if got := testing.AllocsPerRun(100, func() { id := g.Mk(); g.Recycle(id) }); got != 0 {
		t.Fatal("warm ID recycle allocated", got)
	}
}
