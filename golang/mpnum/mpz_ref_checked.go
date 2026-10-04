// Licensed under the MIT license.
//go:build g3checked

package mpnum

import "github.com/glycerine/g3/golang/arena"

type numericCellRef uint64

func makeNumericRef(slot arena.Slot, generation arena.Generation) numericCellRef {
	return numericCellRef(uint64(generation)<<32 | uint64(slot))
}
func numericSlot(ref numericCellRef) arena.Slot { return arena.Slot(uint32(ref)) }
func validNumericRef(s *numericCellSlab, ref numericCellRef) bool {
	return s.IsLive(numericSlot(ref), arena.Generation(uint64(ref)>>32))
}
