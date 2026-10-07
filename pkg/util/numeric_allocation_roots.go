// Source: src/util/mpz.cpp and source numeric-call ownership adapters.
package util

import (
	"errors"

	"github.com/glycerine/g3/pkg/arena"
)

// Actual source-invisible allocation prefixes of an invoked numeric leaf.
// Identity comes from the returned physical tokens, never manager activity.
type NumericAllocationRoots struct {
	Cell      SourceCellBacking
	Allocator [3]Block
}

func CaptureNumericAllocationRoots(z *IntegerManager) (NumericAllocationRoots, error) {
	roots, e := z.Allocator.PendingRoots()
	return NumericAllocationRoots{Cell: z.PendingCell, Allocator: roots}, e
}

func (r *NumericAllocationRoots) After(z *IntegerManager, before NumericAllocationRoots) error {
	if z.PendingCell.Object != (ObjectRef{}) && z.PendingCell != before.Cell {
		r.Cell = z.PendingCell
	}
	roots, e := z.Allocator.PendingRoots()
	if e != nil {
		return e
	}
	for i := range roots {
		if roots[i] != (Block{}) && roots[i] != before.Allocator[i] {
			r.Allocator[i] = roots[i]
		}
	}
	return nil
}

func (r *NumericAllocationRoots) Cleanup(z *IntegerManager) error {
	if r.Cell.Object != (ObjectRef{}) {
		if z.PendingCell == r.Cell {
			e := z.cleanupPendingAllocation()
			// Tracked deallocation can physically complete before reporting failure.
			r.Cell = z.PendingCell
			if e != nil {
				return e
			}
		} else {
			r.Cell = SourceCellBacking{}
		}
	}
	roots, e := z.Allocator.PendingRoots()
	if e != nil {
		return e
	}
	owned := false
	foreign := false
	for i := range roots {
		if roots[i] != (Block{}) {
			if roots[i] == r.Allocator[i] {
				owned = true
			} else {
				foreign = true
			}
		}
	}
	if owned && foreign {
		return ErrNumericPending
	}
	if owned && !foreign {
		e = z.Allocator.CleanupPending()
		now, x := z.Allocator.PendingRoots()
		if x != nil {
			return JoinNumericErrors(e, x)
		}
		r.Allocator = now
		if e != nil {
			return e
		}
	}
	// A replacement or externally consumed token does not grant cleanup authority.
	for i := range roots {
		if roots[i] != r.Allocator[i] {
			r.Allocator[i] = Block{}
		}
	}
	return nil
}

func NumericAllocationPending(r *NumericAllocationRoots) bool {
	if r.Cell.Object.OwnerID() != 0 {
		return true
	}
	for _, p := range r.Allocator {
		if p.OwnerID() != 0 {
			return true
		}
	}
	return false
}

func JoinNumericErrors(a, b error) error {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	return errors.Join(a, b)
}

// This literal source leaf never acknowledges a previously consumed physical
// release. The descriptor's actual owning release cursor belongs to Close.
func SourceNumeralDel(z *IntegerManager, a *Int) (bool, error) {
	if a.Cell != 0 && ValidNumericRef(z.Cells, a.Cell) {
		c := z.Cell(a.Cell)
		if c.CleanupPending || c.Released {
			return true, arena.ErrInvalidSlot
		}
	}
	return sourceNumeralDel(z, a)
}

// PendingObject and PendingAllocatorBlocks expose the exact retained tokens
// for caller diagnostics. They acquire no authority over replacement roots.
func (r *NumericAllocationRoots) PendingObject() ObjectRef         { return r.Cell.Object }
func (r *NumericAllocationRoots) PendingAllocatorBlocks() [3]Block { return r.Allocator }
func (m *IntegerManager) PendingAllocationObject() ObjectRef       { return m.PendingCell.Object }

// SourceAllocator borrows the numeric manager's actual utility allocator.
func (m *IntegerManager) SourceAllocator() *SmallAllocator { return m.Allocator }
