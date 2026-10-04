// Source: src/math/polynomial/linear_eq_solver.h:38-49; src/util/mpz.cpp.
// Licensed under the MIT license. Source author: Leonardo de Moura, 2012.
package mpnum

import "github.com/glycerine/g3/golang/arena"

// Source-only Del must not acknowledge a prior consumed physical release.
// Fresh NULL uses the exact shared setter-recorded source state. Dangling
// and interrupted-release descriptors retain the literal capacity guard.
func linearSourceIntDel(z *IntegerManager, a *Int) error {
	if z.failedNullRef != 0 && a.cell == z.failedNullRef {
		_, e := sourceNumeralDel(z, a)
		return e
	}
	if a.cell != 0 {
		if !validNumericRef(z.cells, a.cell) {
			return arena.ErrInvalidSlot
		}
		c := z.cell(a.cell)
		if c.cleanupPending || c.released {
			return arena.ErrInvalidSlot
		}
		if _, e := c.backing.readWord(z.allocator, 4); e != nil {
			return e
		}
	}
	return z.Del(a)
}
func linearSourceRationalDel(q *RationalManager, a *Rational) error {
	if e := linearSourceIntDel(q.z, &a.num); e != nil {
		return e
	}
	return linearSourceIntDel(q.z, &a.den)
}
