package numeral

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestMPZScalarGetterSourceSwitchReadAndDisabledConditions(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG assertions only")
	}
	t.Run("closed-owner", func(t *testing.T) {
		d, _ := bindMPNDebugActions(t, "")
		m, _ := integerManager(t)
		var big Int
		serviceMust(t, m.SetLimbs(&big, []uint32{1, 1}))
		small := NewInt(7)
		serviceMust(t, d.Close())
		if v, e := m.Uint64(&small); e != nil || v != 7 {
			t.Fatal("small unsigned getter bypasses source assertion", v, e)
		}
		if v, e := m.Int64(&small); e != nil || v != 7 {
			t.Fatal("small signed getter bypasses source assertion", v, e)
		}
		if _, e := m.Uint64(&big); !errors.Is(e, arena.ErrClosed) {
			t.Fatal("big unsigned getter reads source switch", e)
		}
		if _, e := m.Int64(&big); !errors.Is(e, arena.ErrClosed) {
			t.Fatal("big signed getter reads source switch", e)
		}
		if _, e := m.IsInt64(&big); !errors.Is(e, arena.ErrClosed) {
			t.Fatal("absolute-conversion helper reads its own source switch", e)
		}
	})
	t.Run("disabled", func(t *testing.T) {
		d, _ := bindMPNDebugActions(t, "")
		m, _ := integerManager(t)
		var big Int
		serviceMust(t, m.SetLimbs(&big, []uint32{5, 0, 1}))
		serviceMust(t, d.EnableAssertions(false))
		if v, e := m.Int64(&big); e != nil || v != 5 {
			t.Fatal("disabled signed fit assertion leaves source low-word cast", v, e)
		}
		cell := m.Cell(big.Cell)
		original := cell
		cell.Size = 0
		m.Store(big.Cell, cell)
		if v, e := m.Uint64(&big); e != nil || v != 5 {
			t.Fatal("disabled unsigned size assertion reads retained physical words", v, e)
		}
		m.Store(big.Cell, original)
	})
}
