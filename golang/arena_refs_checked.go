//go:build g3checked

package g3

import "github.com/glycerine/g3/golang/arena"

type rawRef uint64

const checkedRefs = true

func makeRef(slot arena.Slot, generation arena.Generation) rawRef {
	return rawRef(uint64(generation)<<32 | uint64(slot))
}
func refSlot(ref rawRef) arena.Slot             { return arena.Slot(uint32(ref)) }
func refGeneration(ref rawRef) arena.Generation { return arena.Generation(uint64(ref) >> 32) }
func validRef(s *arena.Slab32, ref rawRef) bool {
	return s.IsLive(refSlot(ref), refGeneration(ref))
}
