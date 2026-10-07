//go:build !g3checked

package util

import (
	"github.com/glycerine/g3/pkg/arena"
)

type rawRef uint32

const checkedRefs = false

func makeRef(slot arena.Slot, generation arena.Generation) rawRef { return rawRef(slot) }
func refSlot(ref rawRef) arena.Slot                               { return arena.Slot(ref) }
func refGeneration(ref rawRef) arena.Generation                   { return 0 }
func validRef(s *arena.Slab32, ref rawRef) bool {
	// A reused release slot cannot distinguish an old internal reference.
	// Source logical ownership remains responsible for avoiding stale reuse.
	_, err := s.Read(refSlot(ref))
	return err == nil
}
