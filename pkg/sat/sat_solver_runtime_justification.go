package sat

import (
	"reflect"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
)

// Source justification consists of two uint32 words. Clause locator provenance
// is separate, charged Go lifetime metadata. The actual allocator's physical
// pointer/segment -> clause_offset32 encoding remains a distinct open contract:
// val1 is currently the producer's scalar slot, never claimed to be an address.
type satJustificationProvenance struct {
	generation, storageKind uint32
	valid                   bool
}
type satSolverRuntimeJustificationColumn struct {
	raw             RuntimeVectorColumn
	clauses         *ClauseAllocator // fixed real non-owning allocator binding
	budget          *arena.Budget
	provenance      []satJustificationProvenance
	provenanceBytes uint64
}

func (c *satSolverRuntimeJustificationColumn) bind(r *util.Runtime, b *arena.Budget) {
	c.raw.Bind(r)
	c.budget = b
}
func (c *satSolverRuntimeJustificationColumn) bindClauses(a *ClauseAllocator) error {
	if e := c.raw.Open(); e != nil {
		return e
	}
	if e := a.open(); e != nil {
		return e
	}
	if c.clauses != nil && c.clauses != a {
		return ErrSATOwner
	}
	c.clauses = a
	return nil
}
func (c *satSolverRuntimeJustificationColumn) size() (uint32, error) { return c.raw.Size() }
func (c *satSolverRuntimeJustificationColumn) checkIndex(i uint32) error {
	if e := c.raw.Open(); e != nil {
		return e
	}
	if satStorageSourceDebug {
		n, e := c.size()
		if e != nil {
			return e
		}
		if i >= n {
			return ErrSATAssertion
		}
	}
	if c.raw.Block == (util.Block{}) {
		return arena.ErrBounds
	}
	return nil
}
func (c *satSolverRuntimeJustificationColumn) read(i uint32) (satJustification, error) {
	if e := c.checkIndex(i); e != nil {
		return satJustification{}, e
	}
	first, e := ReadRuntimeWord(c.raw.Runtime, c.raw.Block, 8+8*uint64(i))
	if e != nil {
		return satJustification{}, e
	}
	second, e := ReadRuntimeWord(c.raw.Runtime, c.raw.Block, 12+8*uint64(i))
	if e != nil {
		return satJustification{}, e
	}
	j := satJustification{val1: first, val2: second}
	if j.Kind() == 3 {
		if c.clauses == nil || uint64(i) >= uint64(len(c.provenance)) || !c.provenance[i].valid {
			return j, ErrSATOwner
		}
		p := c.provenance[i]
		j.locator = satClauseOffset{owner: c.clauses.origin, slot: first, generation: p.generation, storageKind: p.storageKind}
	}
	return j, nil
}
func (c *satSolverRuntimeJustificationColumn) copyWords(i uint32, j satJustification) error {
	// Invalidate previous checked provenance before an interrupted overwrite;
	// this is guard metadata, never rollback of source raw words.
	if uint64(i) < uint64(len(c.provenance)) {
		c.provenance[i].valid = false
	}
	if e := WriteRuntimeWord(c.raw.Runtime, c.raw.Block, 8+8*uint64(i), j.val1); e != nil {
		return e
	}
	if e := WriteRuntimeWord(c.raw.Runtime, c.raw.Block, 12+8*uint64(i), j.val2); e != nil {
		return e
	}
	return nil
}
func (c *satSolverRuntimeJustificationColumn) publishProvenance(i uint32, j satJustification) error {
	if j.Kind() != 3 {
		return nil
	}
	// Source copies both words before this Go-only provenance publication.
	if c.clauses == nil || j.locator.owner != c.clauses.origin {
		return ErrSATOwner
	}
	if uint64(i) >= uint64(len(c.provenance)) {
		capacity, e := c.raw.Capacity()
		if e != nil {
			return e
		}
		if uint64(capacity) > uint64(^uint(0)>>1)/uint64(reflect.TypeOf(satJustificationProvenance{}).Size()) {
			return arena.ErrBounds
		}
		bytes := uint64(capacity) * uint64(reflect.TypeOf(satJustificationProvenance{}).Size())
		if e = c.budget.Reserve(bytes); e != nil {
			return e
		}
		next := make([]satJustificationProvenance, int(capacity))
		copy(next, c.provenance)
		c.budget.Release(c.provenanceBytes)
		c.provenance = next
		c.provenanceBytes = bytes
	}
	c.provenance[i] = satJustificationProvenance{generation: j.locator.generation, storageKind: j.locator.storageKind, valid: true}
	return nil
}
func (c *satSolverRuntimeJustificationColumn) write(i uint32, j satJustification) error {
	if e := c.checkIndex(i); e != nil {
		return e
	}
	if e := c.copyWords(i, j); e != nil {
		return e
	}
	return c.publishProvenance(i, j)
}
func (c *satSolverRuntimeJustificationColumn) push(j satJustification) error {
	if e := c.raw.Open(); e != nil {
		return e
	}
	if c.raw.Pending != (util.Block{}) {
		return ErrSATOwner
	}
	n, e := c.size()
	if e != nil {
		return e
	}
	capacity, e := c.raw.Capacity()
	if e != nil {
		return e
	}
	if n == capacity {
		if e = c.raw.ExpandStride(8); e != nil {
			return e
		}
	}
	if e = c.copyWords(n, j); e != nil {
		return e
	}
	if e = WriteRuntimeWord(c.raw.Runtime, c.raw.Block, 4, n+1); e != nil {
		return e
	}
	return c.publishProvenance(n, j)
}
func (c *satSolverRuntimeJustificationColumn) finalize() error {
	if e := c.raw.Finalize(); e != nil {
		return e
	}
	c.budget.Release(c.provenanceBytes)
	c.provenance = nil
	c.provenanceBytes = 0
	return nil
}
