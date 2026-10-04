// Source: src/sat/sat_probing.cpp; src/sat/sat_probing.h. Licensed under the MIT license.
package g3

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"reflect"
)

// Scalar provenance is separate from the source LP64 cache_entry bytes. Actual
// returned Block tokens are retained; neither a slot-as-pointer encoding nor a
// native pointer ABI is claimed. Available is authoritative in the raw entry.
type satProbingCacheRow struct {
	block, pending  memory.Block
	pendingReplaces bool
}
type satProbingCache struct {
	block, pending, retired memory.Block
	rows                    []satProbingCacheRow
}

func (c *satProbingCache) blockLive() bool { return c.block != (memory.Block{}) }
func (c *satProbingCache) size(p *satSolverProbing) (uint32, error) {
	if !c.blockLive() {
		return 0, nil
	}
	return satAssignmentReadWord(p.propagation().memory, c.block, 4)
}
func (c *satProbingCache) capacity(p *satSolverProbing) (uint32, error) {
	if !c.blockLive() {
		return 0, nil
	}
	return satAssignmentReadWord(p.propagation().memory, c.block, 0)
}

// Wrapped source unsigned overflow transport is distinct from full typed-array
// representability. Accepted wrapped raw requests stay literal; endangered
// header/element accesses are checked at their actual read/write sites.
func satProbingCacheNext(old uint32, maxHost uint64) (uint32, uint32, uint64, error) {
	n, request, e := satScalarColumnNext(old, 16)
	if e != nil {
		return 0, 0, 0, e
	}
	side := uint64(n) * uint64(reflect.TypeOf(satProbingCacheRow{}).Size())
	if uint64(n) > maxHost || side > maxHost {
		return 0, 0, 0, arena.ErrBounds
	}
	return n, request, side, nil
}
func (c *satProbingCache) expand(p *satSolverProbing) error {
	if c.pending != (memory.Block{}) || c.retired != (memory.Block{}) {
		return ErrSATOwner
	}
	old, e := c.capacity(p)
	if e != nil {
		return e
	}
	if c.blockLive() && old == 0 {
		if satStorageSourceDebug {
			return ErrSATAssertion
		}
		return NewDefaultException("Overflow encountered when expanding vector")
	}
	n, request, _, e := satProbingCacheNext(old, uint64(^uint(0)>>1))
	if e != nil {
		return e
	}
	if uint64(n) > uint64(cap(c.rows)) {
		stride := uint64(reflect.TypeOf(satProbingCacheRow{}).Size())
		bytes := uint64(n) * stride
		if e = p.budget.Reserve(bytes); e != nil {
			return e
		}
		next := make([]satProbingCacheRow, n)
		copy(next, c.rows)
		p.budget.Release(uint64(cap(c.rows)) * stride)
		c.rows = next
	}
	r := p.propagation().memory
	h, e := r.Allocate(uint64(request))
	if e != nil {
		if h != (memory.Block{}) {
			c.pending = h
		}
		return e
	}
	if !c.blockLive() {
		if e = satAssignmentWriteWord(r, h, 0, n); e == nil {
			e = satAssignmentWriteWord(r, h, 4, 0)
		}
		if e != nil {
			c.pending = h
			return e
		}
		c.block = h
		return nil
	}
	size, e := c.size(p)
	if e != nil {
		c.pending = h
		return e
	}
	if e = satAssignmentWriteWord(r, h, 4, size); e != nil {
		c.pending = h
		return e
	}
	oldBlock := c.block
	c.block = h
	c.retired = oldBlock // source publishes new m_data before move.
	for i := uint32(0); i < size; i++ {
		for j := uint64(0); j < 16; j++ {
			if j > 0 && j < 8 {
				continue
			}
			b, e := r.Read(oldBlock, 8+16*uint64(i)+j)
			if e != nil {
				return e
			}
			if e = r.Write(h, 8+16*uint64(i)+j, b); e != nil {
				return e
			}
		}
		// The actual inner scalar tokens moved once when the sidecar was prepared;
		// old cache_entry destructor sees the source moved-from null inner vector.
	}
	if e = r.Deallocate(&c.retired); e != nil {
		return e
	}
	return satAssignmentWriteWord(r, c.block, 0, n)
}
func (c *satProbingCache) reserve(p *satSolverProbing, needed uint32) error {
	old, e := c.size(p)
	if e != nil {
		return e
	}
	if needed <= old {
		return nil
	}
	for {
		capacity, e := c.capacity(p)
		if e != nil {
			return e
		}
		if needed <= capacity {
			break
		}
		if e = c.expand(p); e != nil {
			return e
		}
	}
	r := p.propagation().memory
	// Source resize publishes the final logical size BEFORE default construction.
	if e = satAssignmentWriteWord(r, c.block, 4, needed); e != nil {
		return e
	}
	for i := old; i < needed; i++ {
		// Only named fields are constructed (padding has no source initialization).
		if e = r.Write(c.block, 8+16*uint64(i), 0); e != nil {
			return e
		}
		for j := uint64(8); j < 16; j++ {
			if e = r.Write(c.block, 8+16*uint64(i)+j, 0); e != nil {
				return e
			}
		}
	}
	return nil
}

// The permanently embedded workspace avoids a self-referential stack wrapper
// escaping to an uncharged per-operation heap object. No nested callback uses it.
func (c *satProbingCache) workRow(p *satSolverProbing, i uint32) (*satSolverRuntimeUintColumn, error) {
	if uint64(i) >= uint64(len(c.rows)) {
		return nil, arena.ErrBounds
	}
	out := &p.cacheWork
	row := c.rows[i]
	out.raw.block = row.block
	out.raw.pending = row.pending
	out.raw.pendingReplaces = row.pendingReplaces
	out.raw.closed = false
	c.rows[i] = satProbingCacheRow{}
	return out, nil
}
func (c *satProbingCache) saveWork(p *satSolverProbing, i uint32) {
	out := &p.cacheWork
	c.rows[i] = satProbingCacheRow{out.raw.block, out.raw.pending, out.raw.pendingReplaces}
	out.raw.block = memory.Block{}
	out.raw.pending = memory.Block{}
	out.raw.pendingReplaces = false
	out.raw.closed = false
}
func (c *satProbingCache) finalize(p *satSolverProbing) error {
	// Actual named sidecar owners also retain an interrupted constructed prefix.
	for i := range c.rows {
		out, e := c.workRow(p, uint32(i))
		if e != nil {
			return e
		}
		e = out.finalize()
		c.saveWork(p, uint32(i))
		if e != nil {
			return e
		}
	}
	r := p.propagation().memory
	for _, h := range []*memory.Block{&c.block, &c.retired, &c.pending} {
		if *h != (memory.Block{}) {
			if e := r.Deallocate(h); e != nil {
				return e
			}
		}
	}
	p.budget.Release(uint64(cap(c.rows)) * uint64(reflect.TypeOf(satProbingCacheRow{}).Size()))
	c.rows = nil
	return nil
}

// An entry borrower captures outer storage identity, not a copied owning vector.
type satProbingCacheBorrow struct {
	owner *satSolverProbing
	outer memory.Block
	index uint32
}

func (b satProbingCacheBorrow) row() (satProbingCacheRow, error) {
	if e := b.owner.open(); e != nil {
		return satProbingCacheRow{}, e
	}
	if b.outer != b.owner.cache.block {
		return satProbingCacheRow{}, ErrSATOwner
	}
	if uint64(b.index) >= uint64(len(b.owner.cache.rows)) {
		return satProbingCacheRow{}, arena.ErrBounds
	}
	return b.owner.cache.rows[b.index], nil
}
func (b satProbingCacheBorrow) size() (uint32, error) {
	row, e := b.row()
	if e != nil {
		return 0, e
	}
	if row.block == (memory.Block{}) {
		return 0, nil
	}
	return satAssignmentReadWord(b.owner.propagation().memory, row.block, 4)
}
func (b satProbingCacheBorrow) capture() (memory.Block, uint32, error) {
	row, e := b.row()
	if e != nil {
		return memory.Block{}, 0, e
	}
	n, e := b.size()
	return row.block, n, e
}
func (b satProbingCacheBorrow) readCaptured(h memory.Block, i uint32) (satLiteral, error) {
	if e := b.owner.open(); e != nil {
		return 0, e
	}
	n, e := satAssignmentReadWord(b.owner.propagation().memory, h, 8+4*uint64(i))
	return satLiteral(n), e
}
