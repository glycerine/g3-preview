// Source: src/util/id_gen.h; src/util/vector.h. Licensed under the MIT license.
// Translation of src/util/id_gen.h, Leonardo de Moura, 2008-02-02.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/utilhash"
	"github.com/glycerine/g3/golang/utilid"
	"math"
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
	budget      *arena.Budget
}

func newUtilIDGen(start uint32, outOfMemory func() bool) utilIDGen {
	return utilIDGen{next: start, outOfMemory: outOfMemory}
}

// The generator is embedded cold metadata, charged in the enclosing owner's
// actual header. Only its separate free-ID arrays are charged here. A nil
// budget preserves the unlimited standalone source-helper contract.
func newBudgetedUtilIDGen(start uint32, budget *arena.Budget, outOfMemory func() bool) utilIDGen {
	return utilIDGen{next: start, budget: budget, outOfMemory: outOfMemory}
}

func idFreeGrowth(old uint32) (uint32, error) {
	n, _, e := utilid.VectorGrowth(old)
	if e != nil {
		return 0, arena.ErrBounds
	}
	// Legacy Go-typed free-list adapter checks its host slice extent too.
	if uint64(n)*4 > uint64(^uint(0)>>1) {
		return 0, arena.ErrBounds
	}
	return n, nil
}

// Preallocating this cold workspace lets an enclosing AST owner prepare for
// fallible reclamation bookkeeping before publishing a new node. It changes
// no logical IDs, uses the source vector's capacities and keeps any successful
// growth prefix on failure. Reset retains it; cleanup refunds it.
func (g *utilIDGen) reserveFreeIDs(needed uint32) error {
	if uint64(cap(g.free)) > math.MaxUint32 {
		return arena.ErrBounds
	}
	for uint64(cap(g.free)) < uint64(needed) {
		capacity, e := idFreeGrowth(uint32(cap(g.free)))
		if e != nil {
			return e
		}
		if e = g.budget.Reserve(uint64(capacity) * 4); e != nil {
			return e
		}
		next := make([]uint32, len(g.free), capacity)
		copy(next, g.free)
		old := uint64(cap(g.free)) * 4
		g.free = next
		g.budget.Release(old)
	}
	return nil
}
func (g *utilIDGen) mk() uint32 {
	i, ok := utilid.PopIndex(uint32(len(g.free)))
	if !ok {
		return utilid.Fresh(&g.next)
	}
	id := g.free[i]
	g.free = g.free[:i]
	return id
}
func (g *utilIDGen) recycle(id uint32) error {
	if g.outOfMemory != nil && g.outOfMemory() {
		return nil
	}
	if uint64(len(g.free)) >= math.MaxUint32 {
		return arena.ErrBounds
	}
	if e := g.reserveFreeIDs(uint32(len(g.free)) + 1); e != nil {
		return e
	}
	g.free = append(g.free, id)
	return nil
}
func (g *utilIDGen) reset(start uint32) { g.next = start; g.free = g.free[:0] }
func (g *utilIDGen) cleanup(start uint32) {
	g.next = start
	g.budget.Release(uint64(cap(g.free)) * 4)
	g.free = nil
}
func (g *utilIDGen) idRange() uint32 { return g.next }
func (g *utilIDGen) setNextID(id uint32) uint32 {
	g.next = id
	// Legacy accessor cannot fail; the shared scan still consumes source order.
	v, e := utilid.SkipFree(&g.next, uint32(len(g.free)), func(i uint32) (uint32, error) { return g.free[i], nil })
	if e != nil {
		panic(e)
	}
	return v
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
