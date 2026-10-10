// Source: src/sat/sat_simplifier.cpp; src/sat/sat_clause.h.
package sat

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestSATSimplifierCapturedResolventPayload(t *testing.T) {
	s, b := satNestedTestStore(t)
	r, e := s.NewList()
	paramsMust(t, e)
	for _, l := range []Literal{0, 2, 4} {
		paramsMust(t, s.Push(r, l))
	}
	span, e := satSimplifierCapturedSpan(s, r)
	paramsMust(t, e)
	for i, want := range []Literal{0, 2, 4} {
		v, e := span.literal(uint32(i))
		paramsMust(t, e)
		if v != want {
			t.Fatal(v, want)
		}
	}
	a := satStore(t, b)
	c, e := a.mkClauseSpan(3, span, false)
	paramsMust(t, e)
	for i, want := range []Literal{0, 2, 4} {
		v, e := a.Literal(c, uint32(i))
		paramsMust(t, e)
		if v != want {
			t.Fatal(v, want)
		}
	}
	paramsMust(t, a.Delete(c))
	x, e := s.record(r)
	paramsMust(t, e)
	x.backing.data = x.backing.data[:len(x.backing.data):len(x.backing.data)]
	paramsMust(t, s.Push(r, 6))
	if _, e = span.literal(0); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatalf("captured growth lifetime: %v", e)
	}
	// A checked endangered source c_ptr reaches allocator/header/ID publication
	// before its first read fails. The returned actual owner must be retained.
	c, e = a.mkClauseSpan(3, span, false)
	if !errors.Is(e, arena.ErrInvalidSlot) || c == (StoredClauseRef{}) {
		t.Fatalf("actual allocation prefix: %v %v", c, e)
	}
	n, e := a.Size(c)
	paramsMust(t, e)
	if n != 3 {
		t.Fatal(n)
	}
	paramsMust(t, a.Delete(c))
	fresh, e := satSimplifierCapturedSpan(s, r)
	paramsMust(t, e)
	paramsMust(t, s.Free(r))
	if _, e = fresh.literal(0); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatalf("captured free lifetime: %v", e)
	}
	paramsMust(t, a.Close())
}
