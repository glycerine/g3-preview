package g3

import (
	"github.com/glycerine/g3/golang/utilhash"
	"math"
	"testing"
)

// Added tests: src/util/id_gen.h has no dedicated assertion suite. These finite
// expectations preserve source recycling, resource checks and debug ID resets.
func TestIDGeneratorSourceOrder(t *testing.T) {
	outOfMemory := false
	g := newUtilIDGen(10, func() bool { return outOfMemory })
	if g.mk() != 10 || g.mk() != 11 || g.idRange() != 12 {
		t.Fatal("fresh IDs")
	}
	g.recycle(10)
	g.recycle(11)
	if g.mk() != 11 || g.mk() != 10 || g.mk() != 12 {
		t.Fatal("source LIFO recycle")
	}
	outOfMemory = true
	g.recycle(12)
	if g.mk() != 13 {
		t.Fatal("OOM recycle must be ignored")
	}
	outOfMemory = false
	g.recycle(13)
	g.reset(40)
	if g.mk() != 40 || len(g.free) != 0 {
		t.Fatal("reset")
	}
	g.recycle(40)
	g.cleanup(50)
	if g.free != nil || g.mk() != 50 {
		t.Fatal("cleanup")
	}
}
func TestIDGeneratorDebugAndWrap(t *testing.T) {
	g := newUtilIDGen(0, nil)
	for _, v := range []uint32{3, 4, 6} {
		g.recycle(v)
	}
	if got := g.setNextID(3); got != 5 || g.idRange() != 5 {
		t.Fatal(got)
	}
	if g.displayFreeIDs() != "3 4 6" {
		t.Fatal(g.displayFreeIDs())
	}
	if g.mk() != 6 || g.mk() != 4 || g.mk() != 3 || g.mk() != 5 {
		t.Fatal("debug range should leave free IDs intact")
	}
	g.reset(math.MaxUint32)
	if g.mk() != math.MaxUint32 || g.mk() != 0 {
		t.Fatal("source unsigned wrap")
	}
	g.recycle(1)
	g.recycle(1)
	if g.mk() != 1 || g.mk() != 1 {
		t.Fatal("source does not deduplicate recycle")
	}
}
func TestIDGeneratorHash(t *testing.T) {
	g := newUtilIDGen(17, nil)
	if got, want := g.showHash(), utilhash.UintPair(utilhash.String("", 17), 17); got != want {
		t.Fatal(got, want)
	}
	g.recycle(0x04030201)
	g.recycle(0xfefdfcfb)
	// Construct bytes independently of the generator's uint32 free-ID storage.
	bytes := string([]byte{1, 2, 3, 4, 0xfb, 0xfc, 0xfd, 0xfe})
	if got, want := g.showHash(), utilhash.UintPair(utilhash.String(bytes, 17), 17); got != want {
		t.Fatal(got, want)
	}
}
func TestIDGeneratorFiniteUniqueness(t *testing.T) {
	g := newUtilIDGen(0, nil)
	live := map[uint32]bool{}
	state := uint32(1)
	for i := 0; i < 3000; i++ {
		state = state*1664525 + 1013904223
		if len(live) < 64 && (state&3) != 0 {
			id := g.mk()
			if live[id] {
				t.Fatal("duplicate live ID")
			}
			live[id] = true
			continue
		}
		// Pick deterministically by value, independent of Go map traversal.
		for id := uint32(0); id < g.idRange(); id++ {
			if live[id] {
				delete(live, id)
				g.recycle(id)
				break
			}
		}
	}
	if got := testing.AllocsPerRun(100, func() { id := g.mk(); g.recycle(id) }); got != 0 {
		t.Fatal("warm ID recycle allocated", got)
	}
}
