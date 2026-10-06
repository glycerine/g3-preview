// Licensed under the MIT license; see LICENSE.txt.
// Source: src/util/mpz.cpp:1393-1399; src/util/buffer.h:175-221.
// Source author: Leonardo de Moura.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"testing"
)

func TestNumeralCStringViewExactCompletedCharacterInvocation(t *testing.T) {
	s, _ := serviceNew(t)
	value, e := s.NewValue()
	serviceMust(t, e)
	serviceMust(t, s.SetInt64(&value, 1<<40))
	defer s.Del(&value)
	z := s.manager.z
	c := z.cell(value.raw.num.cell)
	size, e := c.backing.readWord(z.allocator, 0)
	serviceMust(t, e)
	n := z.natural
	construct := func() NumeralCStringView {
		serviceMust(t, n.beginIntegerCharacters(uint32(11*size)))
		serviceMust(t, n.sourceToCString(limbView{allocator: z.allocator, backing: c.backing, size: size}, sourceDecimalDestination{character: &n.charFormatting.buffer, assembly: n}, uint64(n.charFormatting.buffer.pos)))
		return NumeralCStringView{owner: n, invocation: n.charFormatting.invocation}
	}
	old := construct()
	for i, want := range []byte("1099511627776\x00") {
		got, e := old.Byte(uint64(i))
		serviceMust(t, e)
		if got != want {
			t.Fatal("actual character bytes", i, got, want)
		}
	}
	if _, e = old.Byte(uint64(n.charFormatting.buffer.capacity())); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("reached physical capacity", e)
	}
	serviceMust(t, n.destroyIntegerCharacters())
	if _, e = old.Byte(0); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("ended source char object", e)
	}
	next := construct()
	if old.invocation == next.invocation {
		t.Fatal("actual lexical invocation reused")
	}
	if _, e = old.Byte(0); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("expired view read replacement char object", e)
	}
	if ch, e := next.Byte(0); e != nil || ch != '1' {
		t.Fatal("current exact view", ch, e)
	}
	serviceMust(t, n.destroyIntegerCharacters())
}
