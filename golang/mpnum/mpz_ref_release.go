// Licensed under the MIT license.
//go:build !g3checked

package mpnum

import "github.com/glycerine/g3/golang/arena"

type numericCellRef uint32

func makeNumericRef(slot arena.Slot, _ arena.Generation) numericCellRef { return numericCellRef(slot) }
func numericSlot(ref numericCellRef) arena.Slot                         { return arena.Slot(ref) }
func validNumericRef(s *numericCellSlab, ref numericCellRef) bool {
	_, e := s.Read(numericSlot(ref))
	return e == nil
}
