// Source: src/util/mpz.h; src/util/mpz.cpp (_MP_INTERNAL, digit32).
// Licensed under the MIT license. Source author: Leonardo de Moura.
// Actual mpz.h cell allocation and object-bounded initialized-byte access.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
)

// Go safety metadata, to be embedded/charged in the numeric descriptor.
// The actual source payload remains one contiguous 8+4*capacity object.
type sourceCellBacking struct {
	object                    memory.ObjectRef
	raw                       memory.Block
	offset, extent, requested uint64
}

func sourceMPZCellBytes(capacity uint32) uint64 { return uint64(uint32(8) + uint32(4)*capacity) }

// requested retains the result of the source unsigned32 cell-byte expression,
// widened only for the Go storage API. It cannot legitimately exceed uint32.
// Guard corrupt retained metadata before source size_t conversion can erase it.
func (c sourceCellBacking) deallocate(a *memory.SmallAllocator) (memory.ReleaseProgress, error) {
	if c.requested > uint64(^uint32(0)) {
		return memory.ReleaseProgress{}, arena.ErrBounds
	}
	return a.DeallocateTracked(c.requested, c.object)
}

// The returned scalar root is nonzero on a completed allocation even if the
// capacity write fails. Its caller must retain it before descriptor preparation.
func allocateSourceCell(a *memory.SmallAllocator, capacity uint32) (sourceCellBacking, error) {
	return allocateSourceCellWithOutcome(a, capacity, nil)
}
func allocateSourceCellWithOutcome(a *memory.SmallAllocator, capacity uint32, outcome *memory.SmallAllocatorAllocationOutcome) (sourceCellBacking, error) {
	var c sourceCellBacking
	c.requested = sourceMPZCellBytes(capacity)
	object, e := a.AllocateWithOutcome(c.requested, outcome)
	c.object = object
	if e != nil {
		return c, e
	}
	if object == (memory.ObjectRef{}) {
		return c, arena.ErrBounds
	} // actual null header access after size0 allocation
	c.extent, e = a.ObjectExtent(object)
	if e != nil {
		return c, e
	}
	c.raw, c.offset, e = a.RawLocation(object)
	if e != nil {
		return c, e
	}
	return c, c.writeWord(a, 4, capacity) // allocate() initializes capacity ONLY
}
func (c sourceCellBacking) location(a *memory.SmallAllocator, offset uint64) (memory.Block, uint64, error) {
	// Revalidate object generation before borrowing the underlying pooled chunk.
	// A raw chunk alone would otherwise authorize reads after cell recycle.
	raw, start, e := a.RawLocation(c.object)
	if e != nil {
		return memory.Block{}, 0, e
	}
	extent, e := a.ObjectExtent(c.object)
	if e != nil {
		return memory.Block{}, 0, e
	}
	if offset > extent || extent-offset < 4 {
		return memory.Block{}, 0, arena.ErrBounds
	}
	if raw != c.raw || start != c.offset {
		return memory.Block{}, 0, memory.ErrAllocatorOwner
	}
	if start > ^uint64(0)-offset {
		return memory.Block{}, 0, arena.ErrBounds
	}
	return raw, start + offset, nil
}
func (c sourceCellBacking) readWord(a *memory.SmallAllocator, offset uint64) (uint32, error) {
	raw, at, e := c.location(a, offset)
	if e != nil {
		return 0, e
	}
	return a.RuntimeOwner().ReadUint32(raw, at)
}
func (c sourceCellBacking) writeWord(a *memory.SmallAllocator, offset uint64, word uint32) error {
	raw, at, e := c.location(a, offset)
	if e != nil {
		return e
	}
	return a.RuntimeOwner().WriteUint32(raw, at, word)
}
