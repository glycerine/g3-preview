package numeral

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestIntegerEnsureCapacityNestedAllocationAssertionOrder(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG assertions only")
	}
	t.Run("small-pointer-after-allocation-assertion", func(t *testing.T) {
		d, _ := bindMPNDebugActions(t, "")
		m, _ := integerManager(t)
		a := NewInt(-17)
		live := m.Cells.Stats().Live
		serviceMust(t, d.Close())
		err := m.EnsureOwnedCapacity(&a, 33)
		if !errors.Is(err, arena.ErrClosed) || a.Cell != 0 || a.Val != -17 || m.Cells.Stats().Live != live {
			t.Fatal("nested allocator assertion precedes small pointer publication", err, a)
		}
	})
	t.Run("big-replacement-after-allocation-assertion", func(t *testing.T) {
		d, _ := bindMPNDebugActions(t, "")
		m, _ := integerManager(t)
		var a Int
		serviceMust(t, m.SetLimbs(&a, []uint32{0, 1}))
		old := a.Cell
		live := m.Cells.Stats().Live
		serviceMust(t, d.Close())
		err := m.EnsureOwnedCapacity(&a, 33)
		if !errors.Is(err, arena.ErrClosed) || a.Cell != old || a.Val != 1 || m.Cells.Stats().Live != live {
			t.Fatal("nested allocator assertion precedes big replacement allocation", err, a)
		}
		if cell := m.Cell(old); cell.Size != 2 || cell.Capacity != 6 || m.Digit(old, 0) != 0 || m.Digit(old, 1) != 1 {
			t.Fatal("old big source cell remains intact", cell)
		}
	})
}
