package numeral

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

// Source: src/util/mpzzp.h:97-105. The explicit DEL body stops at its first
// failed numeral; owning retry resumes at that numeral, not at the prefix.
func TestIntegerModularCloseRetainsStoppedDestructorSuffix(t *testing.T) {
	z, _ := integerManager(t)
	m, err := NewIntegerModularManagerWithUint64(z, math.MaxUint64, true)
	if err != nil {
		t.Fatal(err)
	}
	lowerRef, upperRef := m.Lower.Cell, m.Upper.Cell
	if lowerRef == 0 || upperRef == 0 || m.P.Cell == 0 {
		t.Fatal("fixture did not allocate the three source modulus fields")
	}
	lower, upper := z.Cell(lowerRef), z.Cell(upperRef)
	t.Cleanup(func() {
		if ValidNumericRef(z.Cells, lowerRef) {
			z.Store(lowerRef, lower)
		}
		if ValidNumericRef(z.Cells, upperRef) {
			z.Store(upperRef, upper)
		}
		if !m.closed {
			if e := m.Close(); e != nil {
				t.Error(e)
			}
		}
	})

	broken := lower
	broken.Backing.Requested = uint64(math.MaxUint32) + 1
	z.Store(lowerRef, broken)
	if e := m.Close(); !errors.Is(e, arena.ErrBounds) || !m.closing || m.closeCursor != 1 {
		t.Fatal("source destructor did not stop at lower", e, m.closeCursor)
	}
	if m.P.Cell != 0 || m.Lower.Cell != lowerRef || m.Upper.Cell != upperRef || !ValidNumericRef(z.Cells, upperRef) {
		t.Fatal("completed prefix or untouched suffix changed")
	}
	if _, e := m.Finite(); !errors.Is(e, ErrNumericPending) {
		t.Fatal("partly destroyed manager remained usable", e)
	}
	if e := m.Close(); !errors.Is(e, arena.ErrBounds) || m.closeCursor != 1 {
		t.Fatal("owning retry advanced past unresolved lower", e, m.closeCursor)
	}

	z.Store(lowerRef, lower)
	broken = upper
	broken.Backing.Requested = uint64(math.MaxUint32) + 1
	z.Store(upperRef, broken)
	if e := m.Close(); !errors.Is(e, arena.ErrBounds) || m.closeCursor != 2 {
		t.Fatal("owning retry did not stop at upper", e, m.closeCursor)
	}
	if m.Lower.Cell != 0 || m.Upper.Cell != upperRef {
		t.Fatal("second retry did not preserve completed prefix and suffix")
	}
	z.Store(upperRef, upper)
	if e := m.Close(); e != nil || !m.closed || m.closeCursor != 7 {
		t.Fatal("owning cleanup did not finish seven source deletions", e, m.closeCursor)
	}
}
