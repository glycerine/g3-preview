// Source: src/math/polynomial/linear_eq_solver.h:38-49; src/util/mpz.cpp.
package numeral

import (
	"github.com/glycerine/g3/pkg/arena"
)

// Source-only Del must not acknowledge a prior consumed physical release.
// Fresh NULL uses the exact shared setter-recorded source state. Dangling
// and interrupted-release descriptors retain the literal capacity guard.

func SourceIntegerDel(z *IntegerManager, a *Int) error {
	if z.FailedNullRef != 0 && a.Cell == z.FailedNullRef {
		_, e := SourceNumeralDel(z, a)
		return e
	}
	if a.Cell != 0 {
		if !ValidNumericRef(z.Cells, a.Cell) {
			return arena.ErrInvalidSlot
		}
		c := z.Cell(a.Cell)
		if c.CleanupPending || c.Released {
			return arena.ErrInvalidSlot
		}
		if _, e := c.Backing.ReadWord(z.SourceAllocator(), 4); e != nil {
			return e
		}
	}
	return z.Del(a)
}
func SourceRationalDel(q *RationalManager, a *Rational) error {
	if e := SourceIntegerDel(q.Integers(), &a.Num); e != nil {
		return e
	}
	return SourceIntegerDel(q.Integers(), &a.Den)
}
