// Source: src/util/mpz.cpp:1385-1410; src/util/mpn.cpp:360-399.
package util

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestIntegerCharacterNativeScratchGrowthAndLifetime(t *testing.T) {
	m := naturalManager(t)
	before := m.runtime.Stats()
	m.budget.FailAfter(0)
	defer m.budget.FailAfter(-1)
	for _, n := range []uint32{1023, 1034, 4097, 1023} {
		if e := m.beginIntegerCharacters(n); e != nil {
			t.Fatal(e)
		}
		s := &m.CharFormatting
		if len(s.Buffer) != int(n) || !s.Completed || !s.active {
			t.Fatal("native construction", n)
		}
		for _, b := range s.Buffer {
			if b != 0 {
				t.Fatal("scratch not cleared")
			}
		}
		s.Buffer[0] = '7'
		view := numeralCStringView{owner: m, invocation: s.invocation}
		if b, e := view.Byte(0); e != nil || b != '7' {
			t.Fatal(b, e)
		}
		if _, e := view.Byte(uint64(n)); !errors.Is(e, arena.ErrBounds) {
			t.Fatal("bounds", e)
		}
		if e := m.destroyIntegerCharacters(); e != nil {
			t.Fatal(e)
		}
		if _, e := view.Byte(0); !errors.Is(e, arena.ErrInvalidSlot) {
			t.Fatal("stale view", e)
		}
	}
	if m.runtime.Stats() != before {
		t.Fatal("native scratch touched Runtime")
	}
}
