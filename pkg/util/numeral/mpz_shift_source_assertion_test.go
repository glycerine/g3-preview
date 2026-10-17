package numeral

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/diagnostic"
)

func TestIntegerShiftSourceAssertionSwitchOrder(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG assertions only")
	}
	t.Run("division-before-trace", func(t *testing.T) {
		d, _ := bindMPNDebugActions(t, "")
		m, _ := integerManager(t)
		var a Int
		serviceMust(t, m.SetLimbs(&a, []uint32{1, 1}))
		serviceMust(t, d.Close())
		err := m.MachineDiv2kInPlace(&a, 1)
		if !errors.Is(err, arena.ErrClosed) || m.Cell(a.Cell).Size != 2 || m.Digit(a.Cell, 0) != 1 || m.Digit(a.Cell, 1) != 1 {
			t.Fatal("first assertion switch must precede trace and digit mutation", err)
		}
	})
	t.Run("division-after-trace", func(t *testing.T) {
		d, _ := bindMPNDebugActions(t, "")
		m, _ := integerManager(t)
		var a Int
		serviceMust(t, m.SetLimbs(&a, []uint32{1, 1}))
		w := &mpnCloseDebugOnFlush{debug: d}
		tags := diagnostic.NewTraceTagsWithWriter(w)
		tags.Enable("mpz_2k")
		if !tags.ShouldTrace("mpz_2k") {
			t.Skip("source shift TRACE erased")
		}
		serviceMust(t, m.BindTrace(tags))
		err := m.MachineDiv2kInPlace(&a, 1)
		if !errors.Is(err, arena.ErrClosed) || w.closeErr != nil || w.flushes == 0 || m.Cell(a.Cell).Size != 2 || m.Digit(a.Cell, 0) != 1 || m.Digit(a.Cell, 1) != 1 {
			t.Fatal("second assertion switch must follow trace and precede digit mutation", err, w.closeErr, w.flushes)
		}
	})
	t.Run("multiply-after-capacity", func(t *testing.T) {
		d, _ := bindMPNDebugActions(t, "")
		m, _ := integerManager(t)
		a := NewInt(1)
		serviceMust(t, d.Close())
		err := m.Mul2kInPlace(&a, 32)
		if !errors.Is(err, arena.ErrClosed) || a.IsSmall() || m.Cell(a.Cell).Size != 1 || m.Digit(a.Cell, 0) != 1 {
			t.Fatal("multiply assertion switch must follow capacity preparation and precede shift", err, a)
		}
	})
}
