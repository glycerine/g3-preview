package numeral

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestIntegerEnsureCapacitySourceAssertionPublicationOrder(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG assertions only")
	}
	t.Run("small-pointer-before-assertion", func(t *testing.T) {
		d, _ := bindMPNDebugActions(t, "")
		m, _ := integerManager(t)
		a := NewInt(-17)
		serviceMust(t, d.Close())
		err := m.EnsureOwnedCapacity(&a, 33)
		if !errors.Is(err, arena.ErrClosed) || a.Cell == 0 || a.Val != -17 {
			t.Fatal("small source pointer must publish before assertion, with old sign", err, a)
		}
		if cell := m.Cell(a.Cell); cell.Unpublished || cell.Capacity != 33 || cell.Size != 0 {
			t.Fatal("small source cell stopped before digit and size writes", cell)
		}
	})
	t.Run("big-replacement-before-assertion", func(t *testing.T) {
		d, _ := bindMPNDebugActions(t, "")
		m, _ := integerManager(t)
		var a Int
		serviceMust(t, m.SetLimbs(&a, []uint32{0, 1}))
		old := a.Cell
		live := m.Cells.Stats().Live
		serviceMust(t, d.Close())
		err := m.EnsureOwnedCapacity(&a, 33)
		if !errors.Is(err, arena.ErrClosed) || a.Cell != old || a.Val != 1 || m.Cells.Stats().Live != live+1 {
			t.Fatal("big source replacement remains separate before copy/publication", err, a)
		}
		if cell := m.Cell(old); cell.Size != 2 || cell.Capacity != 6 || m.Digit(old, 0) != 0 || m.Digit(old, 1) != 1 {
			t.Fatal("old big source cell remains intact", cell)
		}
	})
}
