// Copyright (c) 2006 Microsoft Corporation. Licensed under the MIT license.
// Translation of src/util/id_gen.h, Leonardo de Moura, 2008-02-02.
package g3

import (
	"github.com/glycerine/g3/golang/utilhash"
	"strconv"
	"strings"
)

// utilIDGen owns recycled source IDs. They are not arena slots or generations;
// source AST ID compression/recycling must call this generator explicitly.
// Go assignment aliases the owned free list; do not copy an initialized owner.
// outOfMemory is one cold owner resource-service callback, not a per-ID closure.
// Nil means the source memory service is not reporting out-of-memory.
type utilIDGen struct {
	next        uint32
	free        []uint32
	outOfMemory func() bool
}

func newUtilIDGen(start uint32, outOfMemory func() bool) utilIDGen {
	return utilIDGen{next: start, outOfMemory: outOfMemory}
}
func (g *utilIDGen) mk() uint32 {
	if len(g.free) == 0 {
		id := g.next
		g.next++
		return id
	}
	id := g.free[len(g.free)-1]
	g.free = g.free[:len(g.free)-1]
	return id
}
func (g *utilIDGen) recycle(id uint32) {
	if g.outOfMemory != nil && g.outOfMemory() {
		return
	}
	g.free = append(g.free, id)
}
func (g *utilIDGen) reset(start uint32)   { g.next = start; g.free = g.free[:0] }
func (g *utilIDGen) cleanup(start uint32) { g.next = start; g.free = nil }
func (g *utilIDGen) idRange() uint32      { return g.next }
func (g *utilIDGen) setNextID(id uint32) uint32 {
	g.next = id
	for {
		found := false
		for _, free := range g.free {
			if free == g.next {
				found = true
				break
			}
		}
		if !found {
			return g.next
		}
		g.next++
	}
}
func (g *utilIDGen) showHash() uint32 {
	// Preserve the baseline byte order/sign policy of utilhash. Empty free IDs
	// need no invalid &vector[0] address. Source hash byte length is uint32.
	if uint64(len(g.free)) > uint64(^uint32(0))/4 {
		panic("id generator: hash byte length overflow")
	}
	h := utilhash.WordBytes(g.free, uint32(len(g.free))*4, 17)
	return utilhash.UintPair(h, g.next)
}
func (g *utilIDGen) displayFreeIDs() string {
	var out strings.Builder
	for i, id := range g.free {
		if i != 0 {
			out.WriteByte(' ')
		}
		out.WriteString(strconv.FormatUint(uint64(id), 10))
	}
	return out.String()
}
