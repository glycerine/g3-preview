// Source: src/util/small_object_allocator.cpp:172–242; src/util/debug.h:63.
package util

import (
	"math"
)

func (a *SmallAllocator) Consolidate() (err error) {
	if e := a.open(); e != nil {
		return e
	}
	if a.hasPending() || a.cleanupMode != 0 || a.cleanupChunksNeeded {
		return ErrAllocatorPending
	}
	if e := a.verbose(false); e != nil {
		return e
	}
	chunks := smallObjectPointerVector{owner: a}
	freeObjs := smallObjectPointerVector{owner: a}
	defer func() {
		if e := chunks.close(); err == nil {
			err = e
		}
	}()
	defer func() {
		if e := freeObjs.close(); err == nil {
			err = e
		}
	}()
	for slot := uint32(1); slot < a.slots(); slot++ {
		if a.free[slot] == 0 {
			continue
		}
		if e := chunks.reset(); e != nil {
			return e
		}
		if e := freeObjs.reset(); e != nil {
			return e
		}
		id := a.chunks[slot]
		steps := 0
		for id != 0 {
			if steps >= len(a.locations) {
				return ErrAllocatorNonterminating
			}
			steps++
			if e := chunks.push(id); e != nil {
				return e
			}
			c, e := a.location(id)
			if e != nil {
				return e
			}
			next, e := smallObjectReadWord(a.config.Runtime, c.raw, 0, a.pointerBytes())
			if e != nil {
				return e
			}
			if next > math.MaxUint32 {
				return ErrAllocatorOwner
			}
			id = uint32(next)
		}
		id = a.free[slot]
		steps = 0
		for id != 0 {
			if steps >= len(a.locations) {
				return ErrAllocatorNonterminating
			}
			steps++
			if e := freeObjs.push(id); e != nil {
				return e
			}
			obj, e := a.location(id)
			if e != nil {
				return e
			}
			next, e := smallObjectReadWord(a.config.Runtime, obj.raw, uint64(obj.offset), a.pointerBytes())
			if e != nil {
				return e
			}
			if next > math.MaxUint32 {
				return ErrAllocatorOwner
			}
			id = uint32(next)
		}
		objectSize := uint32(a.pointerBytes()) * slot
		perChunk := (8192 - 2*uint32(a.pointerBytes())) / objectSize
		if freeObjs.size < perChunk {
			continue
		}
		if e := a.sourceAssert(chunks.size != 0); e != nil {
			return e
		}
		if e := chunks.sort(); e != nil {
			return e
		}
		if e := freeObjs.sort(); e != nil {
			return e
		}
		var lastChunk, lastFree, objectIndex uint32
		for chunkIndex := uint32(0); chunkIndex < chunks.size; chunkIndex++ {
			chunkID, e := chunks.get(chunkIndex)
			if e != nil {
				return e
			}
			chunk, e := a.location(chunkID)
			if e != nil {
				return e
			}
			end := chunk.address + 8192
			saved := objectIndex
			var count uint32
			for objectIndex < freeObjs.size {
				objectID, e := freeObjs.get(objectIndex)
				if e != nil {
					return e
				}
				obj, e := a.location(objectID)
				if e != nil {
					return e
				}
				if obj.address > end {
					break
				}
				objectIndex++
				count++
			}
			if count == perChunk {
				h := chunk.raw
				a.cleanupChunksNeeded = true
				e = a.config.Runtime.Deallocate(&h)
				if h == (Block{}) {
					a.invalidateChunk(chunkID)
				}
				if e != nil {
					return e
				}
			} else {
				if e := smallObjectWriteWord(a.config.Runtime, chunk.raw, 0, a.pointerBytes(), uint64(lastChunk)); e != nil {
					return e
				}
				lastChunk = chunkID
				for i := saved; i < objectIndex; i++ {
					objectID, e := freeObjs.get(i)
					if e != nil {
						return e
					}
					obj, e := a.location(objectID)
					if e != nil {
						return e
					}
					if e = smallObjectWriteWord(a.config.Runtime, obj.raw, uint64(obj.offset), a.pointerBytes(), uint64(lastFree)); e != nil {
						return e
					}
					lastFree = objectID
				}
			}
		}
		a.chunks[slot] = lastChunk
		a.free[slot] = lastFree
	}
	// Source end diagnostics occur BEFORE lexical scratch destructors.
	a.cleanupChunksNeeded = false
	return a.verbose(true)
}
