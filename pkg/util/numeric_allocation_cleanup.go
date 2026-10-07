package util

import (
	"errors"
)

func CleanupNumericAllocationRoots(z *IntegerManager, r *NumericAllocationRoots) error {
	if r.Cell.Object != (ObjectRef{}) {
		old := r.Cell
		if z.PendingCell == old {
			e := z.cleanupPendingAllocation()
			if z.PendingCell != old {
				r.Cell = SourceCellBacking{}
			}
			if e != nil {
				return e
			}
		} else {
			r.Cell = SourceCellBacking{}
		}
	}
	now, e := z.Allocator.PendingRoots()
	if e != nil {
		return e
	}
	owned, foreign := false, false
	for i, x := range now {
		if x != (Block{}) {
			if x == r.Allocator[i] {
				owned = true
			} else {
				foreign = true
			}
		}
	}
	if owned && foreign {
		return ErrNumericCleanupPending
	}
	if owned {
		e = z.Allocator.CleanupPending()
		now, x := z.Allocator.PendingRoots()
		if x != nil {
			return errors.Join(e, x)
		}
		for i, old := range r.Allocator {
			if now[i] != old {
				r.Allocator[i] = Block{}
			}
		}
		if e != nil {
			return e
		}
	} else {
		for i, old := range r.Allocator {
			if now[i] != old {
				r.Allocator[i] = Block{}
			}
		}
	}
	return nil
}
