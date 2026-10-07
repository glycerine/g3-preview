// Source: src/util/id_gen.h; src/util/vector.h.
// Translation of src/util/id_gen.h, Leonardo de Moura, 2008-02-02.
package util

import (
	"math"
	"strconv"
	"strings"

	"github.com/glycerine/g3/pkg/arena"
)

// utilIDGen owns recycled source IDs. They are not arena slots or generations;
// source AST ID compression/recycling must call this generator explicitly.
// Go assignment aliases the owned free list; do not copy an initialized owner.
// outOfMemory is one cold owner resource-service callback, not a per-ID closure.
// Nil means the source memory service is not reporting out-of-memory.
type IDGen struct {
	Next        uint32
	Free        []uint32
	OutOfMemory func() bool
	Budget      *arena.Budget
}

func NewIDGen(start uint32, outOfMemory func() bool) IDGen {
	return IDGen{Next: start, OutOfMemory: outOfMemory}
}

// The generator is embedded cold metadata, charged in the enclosing owner's
// actual header. Only its separate free-ID arrays are charged here. A nil
// budget preserves the unlimited standalone source-helper contract.
func NewBudgetedIDGen(start uint32, budget *arena.Budget, outOfMemory func() bool) IDGen {
	return IDGen{Next: start, Budget: budget, OutOfMemory: outOfMemory}
}

func IDFreeGrowth(old uint32) (uint32, error) {
	n, _, e := VectorGrowth(old)
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
func (g *IDGen) ReserveFreeIDs(needed uint32) error {
	if uint64(cap(g.Free)) > math.MaxUint32 {
		return arena.ErrBounds
	}
	for uint64(cap(g.Free)) < uint64(needed) {
		capacity, e := IDFreeGrowth(uint32(cap(g.Free)))
		if e != nil {
			return e
		}
		if e = g.Budget.Reserve(uint64(capacity) * 4); e != nil {
			return e
		}
		next := make([]uint32, len(g.Free), capacity)
		copy(next, g.Free)
		old := uint64(cap(g.Free)) * 4
		g.Free = next
		g.Budget.Release(old)
	}
	return nil
}
func (g *IDGen) Mk() uint32 {
	i, ok := PopIndex(uint32(len(g.Free)))
	if !ok {
		return Fresh(&g.Next)
	}
	id := g.Free[i]
	g.Free = g.Free[:i]
	return id
}
func (g *IDGen) Recycle(id uint32) error {
	if g.OutOfMemory != nil && g.OutOfMemory() {
		return nil
	}
	if uint64(len(g.Free)) >= math.MaxUint32 {
		return arena.ErrBounds
	}
	if e := g.ReserveFreeIDs(uint32(len(g.Free)) + 1); e != nil {
		return e
	}
	g.Free = append(g.Free, id)
	return nil
}
func (g *IDGen) Reset(start uint32) { g.Next = start; g.Free = g.Free[:0] }
func (g *IDGen) Cleanup(start uint32) {
	g.Next = start
	g.Budget.Release(uint64(cap(g.Free)) * 4)
	g.Free = nil
}
func (g *IDGen) IdRange() uint32 { return g.Next }
func (g *IDGen) SetNextID(id uint32) uint32 {
	g.Next = id
	// Legacy accessor cannot fail; the shared scan still consumes source order.
	v, e := SkipFree(&g.Next, uint32(len(g.Free)), func(i uint32) (uint32, error) { return g.Free[i], nil })
	if e != nil {
		panic(e)
	}
	return v
}
func (g *IDGen) ShowHash() uint32 {
	// Preserve the baseline byte order/sign policy of utilhash. Empty free IDs
	// need no invalid &vector[0] address. Source hash byte length is uint32.
	if uint64(len(g.Free)) > uint64(^uint32(0))/4 {
		panic("id generator: hash byte length overflow")
	}
	h := WordBytes(g.Free, uint32(len(g.Free))*4, 17)
	return UintPair(h, g.Next)
}
func (g *IDGen) DisplayFreeIDs() string {
	var out strings.Builder
	for i, id := range g.Free {
		if i != 0 {
			out.WriteByte(' ')
		}
		out.WriteString(strconv.FormatUint(uint64(id), 10))
	}
	return out.String()
}
