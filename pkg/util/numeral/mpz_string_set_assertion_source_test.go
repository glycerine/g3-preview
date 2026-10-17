package numeral

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestIntegerStringSetDigitAssertionSourceStop(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG assertions only")
	}
	t.Run("digit", func(t *testing.T) {
		d, _ := bindMPNDebugActions(t, "")
		m, _ := integerManager(t)
		out := NewInt(71)
		serviceMust(t, d.Close())
		err := m.SetString(&out, "  -2")
		s := &m.stringSet
		if !errors.Is(err, arena.ErrClosed) || out != NewInt(0) || !s.declared || s.index != 3 || s.stage != 0 || !s.negative || s.endReached {
			t.Fatal("assertion must stop after RESET and declarations, before Mul", err, out, s.index, s.stage)
		}
		serviceMust(t, m.CleanupStringSet())
	})
	t.Run("nondigit", func(t *testing.T) {
		d, _ := bindMPNDebugActions(t, "")
		m, _ := integerManager(t)
		out := NewInt(71)
		serviceMust(t, d.Close())
		if err := m.SetString(&out, "  -x"); err != nil || out != NewInt(0) || !m.stringSet.endReached {
			t.Fatal("non-digit skips the source assertion", err, out)
		}
	})
}
