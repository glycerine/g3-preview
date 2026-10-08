// Source: selected libstdc++ basic_string.tcc:139-158,326-341,414-427.
package util

import (
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestOwnedStringAppendIndependentLengthsAndSelf(t *testing.T) {
	for a := 0; a <= 70; a++ {
		for n := 0; n <= 70; n++ {
			b := arena.NewBudget(10000)
			var dst, src OwnedString
			first, second := strings.Repeat("a", a), strings.Repeat("b", n)
			serviceMust(t, dst.Construct(b, LiteralParamsStringView(first)))
			serviceMust(t, src.Construct(b, LiteralParamsStringView(second)))
			old := dst.View()
			oldCap := dst.Capacity
			serviceMust(t, dst.Append(src.View()))
			got, e := dst.View().String()
			serviceMust(t, e)
			if got != first+second {
				t.Fatal(a, n, got)
			}
			capWant := oldCap
			if uint64(a+n) > capWant {
				capWant = uint64(a + n)
				if capWant < 2*oldCap {
					capWant = 2 * oldCap
				}
			}
			if dst.Capacity != capWant {
				t.Fatal("capacity", a, n, dst.Capacity, capWant)
			}
			if a > 0 {
				_, e = old.Byte(0)
				if capWant != oldCap && !errors.Is(e, arena.ErrInvalidSlot) {
					t.Fatal("old view survived growth", e)
				}
				if capWant == oldCap {
					serviceMust(t, e)
				}
			}
			if dst.External != nil && dst.External[dst.Length] != 0 {
				t.Fatal("missing NUL")
			}
			if dst.External == nil && dst.Inline[dst.Length] != 0 {
				t.Fatal("missing inline NUL")
			}
			owned := uint64(len(dst.External) + len(src.External))
			if b.Used() != owned {
				t.Fatal("physical bytes", b.Used(), owned)
			}
			dst.SourceDestroy()
			src.SourceDestroy()
			if b.Used() != 0 {
				t.Fatal("leak", b.Used())
			}
		}
	}
	for n := 0; n <= 80; n++ {
		b := arena.NewBudget(1000)
		var s OwnedString
		before := strings.Repeat("x", n)
		serviceMust(t, s.Construct(b, LiteralParamsStringView(before)))
		serviceMust(t, s.Append(s.View()))
		got, e := s.View().String()
		serviceMust(t, e)
		if got != before+before {
			t.Fatal("self", n, got)
		}
		s.SourceDestroy()
		if b.Used() != 0 {
			t.Fatal("self leak")
		}
	}
}

func TestOwnedStringAppendCountedBinaryAndCString(t *testing.T) {
	b := arena.NewBudget(100)
	var s, src OwnedString
	serviceMust(t, s.Construct(b, LiteralParamsStringView("a")))
	serviceMust(t, src.Construct(b, LiteralParamsStringView("xyz")))
	src.Put(1, 0)
	serviceMust(t, s.Append(src.View()))
	serviceMust(t, s.AppendCString(LiteralParamsStringView("b\x00ignored")))
	got, e := s.View().String()
	serviceMust(t, e)
	if got != "ax\x00zb" {
		t.Fatal([]byte(got))
	}
	// Count zero erases source byte reads, as _M_append does.
	serviceMust(t, s.Append(OwnedStringView{}))
	s.SourceDestroy()
	src.SourceDestroy()
	if b.Used() != 0 {
		t.Fatal("binary leak")
	}
}

func TestOwnedStringAppendRefusalAndRetainedReplacement(t *testing.T) {
	for _, initial := range []string{"123456789012345", strings.Repeat("a", 20)} {
		b := arena.NewBudget(1000)
		var s, src OwnedString
		serviceMust(t, s.Construct(b, LiteralParamsStringView(initial)))
		serviceMust(t, src.Construct(b, LiteralParamsStringView(strings.Repeat("b", 50))))
		old := s.View()
		used := b.Used()
		b.FailAfter(0)
		e := s.Append(src.View())
		if !errors.Is(e, arena.ErrInjectedFailure) {
			t.Fatal("refusal", e)
		}
		if b.Used() != used || s.Pending != nil {
			t.Fatal("allocation on refusal")
		}
		got, e := old.String()
		serviceMust(t, e)
		if got != initial {
			t.Fatal("old storage changed")
		}
		b.FailAfter(-1)
		stale := src.View()
		src.SourceDestroy()
		used = b.Used()
		e = s.Append(stale)
		if !errors.Is(e, arena.ErrInvalidSlot) {
			t.Fatal("stale", e)
		}
		capWant := uint64(len(initial) + 50)
		if capWant < 2*s.Capacity {
			capWant = 2 * s.Capacity
		}
		if uint64(len(s.Pending)) != capWant+1 || b.Used() != used+capWant+1 {
			t.Fatal("retained actual replacement", b.Used(), len(s.Pending))
		}
		got, e = old.String()
		serviceMust(t, e)
		if got != initial || s.Length != uint64(len(initial)) {
			t.Fatal("premature publication")
		}
		for i := range initial {
			if s.Pending[i] != initial[i] {
				t.Fatal("old prefix missing")
			}
		}
		if !errors.Is(s.AppendCString(LiteralParamsStringView("x")), ErrParamsPrecondition) {
			t.Fatal("replayed stopped append")
		}
		s.OwnAbort()
		if b.Used() != 0 {
			t.Fatal("abort leak", b.Used())
		}
	}
}

func TestOwnedStringAppendOwnerAndUnreachedReads(t *testing.T) {
	b := arena.NewBudget(1000)
	var s, src OwnedString
	serviceMust(t, s.Construct(b, LiteralParamsStringView("a")))
	serviceMust(t, src.Construct(b, LiteralParamsStringView("bc")))
	stale := src.View()
	src.SourceDestroy()
	if e := s.Append(stale); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal(e)
	}
	if s.Length != 1 || s.Inline[0] != 'a' || b.Used() != 0 {
		t.Fatal("inline failure publication")
	}
	// No allocation was reached, so an owning caller may explicitly abort or
	// repair the borrowed source. Pending replacement state itself forbids replay.
	copied := s
	if !errors.Is(copied.AppendCString(LiteralParamsStringView("x")), ErrParamsPrecondition) {
		t.Fatal("copied owner")
	}
	var absent *OwnedString
	if !errors.Is(absent.Append(OwnedStringView{}), ErrParamsPrecondition) {
		t.Fatal("nil receiver")
	}
	bad := OwnedStringView{Length: ^uint64(0)}
	if !errors.Is(s.Append(bad), arena.ErrBounds) {
		t.Fatal("overflow guard")
	}
	if b.Used() != 0 || s.Length != 1 {
		t.Fatal("overflow touched storage")
	}
	s.SourceDestroy()
	if !errors.Is(s.Append(OwnedStringView{}), ErrParamsPrecondition) {
		t.Fatal("closed owner")
	}
}
