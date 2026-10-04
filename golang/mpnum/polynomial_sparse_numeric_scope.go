// Source: src/math/polynomial/polynomial.cpp:3109-3182; src/util/mpz.cpp.
// Licensed under the MIT license. Source author: Leonardo de Moura (leonardo).
package mpnum

import "github.com/glycerine/g3/golang/memory"

// Actual source-invisible allocation prefixes of an invoked numeric leaf.
// Identity comes from the returned physical tokens, never manager activity.
type polynomialNumericRoots struct {
	cell      sourceCellBacking
	allocator [3]memory.Block
}

func polynomialNumericBefore(z *IntegerManager) (polynomialNumericRoots, error) {
	roots, e := z.allocator.PendingRoots()
	return polynomialNumericRoots{cell: z.pendingCell, allocator: roots}, e
}
func (r *polynomialNumericRoots) after(z *IntegerManager, before polynomialNumericRoots) error {
	if z.pendingCell.object != (memory.ObjectRef{}) && z.pendingCell != before.cell {
		r.cell = z.pendingCell
	}
	roots, e := z.allocator.PendingRoots()
	if e != nil {
		return e
	}
	for i := range roots {
		if roots[i] != (memory.Block{}) && roots[i] != before.allocator[i] {
			r.allocator[i] = roots[i]
		}
	}
	return nil
}
func (r *polynomialNumericRoots) cleanup(z *IntegerManager) error {
	if r.cell.object != (memory.ObjectRef{}) {
		if z.pendingCell == r.cell {
			e := z.cleanupPendingAllocation()
			// Tracked deallocation can physically complete before reporting failure.
			r.cell = z.pendingCell
			if e != nil {
				return e
			}
		} else {
			r.cell = sourceCellBacking{}
		}
	}
	roots, e := z.allocator.PendingRoots()
	if e != nil {
		return e
	}
	owned := false
	foreign := false
	for i := range roots {
		if roots[i] != (memory.Block{}) {
			if roots[i] == r.allocator[i] {
				owned = true
			} else {
				foreign = true
			}
		}
	}
	if owned && foreign {
		return ErrPolynomialPending
	}
	if owned && !foreign {
		e = z.allocator.CleanupPending()
		now, x := z.allocator.PendingRoots()
		if x != nil {
			return polynomialNewtonJoin(e, x)
		}
		r.allocator = now
		if e != nil {
			return e
		}
	}
	// A replacement or externally consumed token does not grant cleanup authority.
	for i := range roots {
		if roots[i] != r.allocator[i] {
			r.allocator[i] = memory.Block{}
		}
	}
	return nil
}

func (s *PolynomialSparseInterpolator) numericBefore() (polynomialNumericRoots, error) {
	return polynomialNumericBefore(s.owner.core.integer)
}
func (s *PolynomialSparseInterpolator) numericAfter(before polynomialNumericRoots) error {
	return s.numeric.after(s.owner.core.integer, before)
}
func (s *PolynomialSparseInterpolator) cleanupNumeric() error {
	return s.numeric.cleanup(s.owner.core.integer)
}
