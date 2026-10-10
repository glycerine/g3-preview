package util

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"strings"
	"testing"
)

func TestOwnedStringReserveAndMoveAssignment(t *testing.T) {
	for _, a := range []int{0, 5, 15, 16, 31} {
		for _, n := range []int{0, 5, 15, 16, 31} {
			b := arena.NewBudget(0)
			b.FailAfter(0)
			var dst, src OwnedString
			serviceMust(t, dst.Construct(b, strings.Repeat("a", a)))
			serviceMust(t, src.Construct(b, strings.Repeat("b", n)))
			oldDst, oldSrc := dst.View(), src.View()
			serviceMust(t, dst.MoveAssign(&src))
			if !dst.Matches(strings.Repeat("b", n)) || !src.Matches("") || b.Used() != 0 {
				t.Fatal(a, n)
			}
			if _, e := oldDst.String(); !errors.Is(e, arena.ErrInvalidSlot) {
				t.Fatal(e)
			}
			if _, e := oldSrc.String(); !errors.Is(e, arena.ErrInvalidSlot) {
				t.Fatal(e)
			}
		}
	}
	var s OwnedString
	serviceMust(t, s.Construct(nil, "abc"))
	old := s.View()
	serviceMust(t, s.Reserve(1024))
	got, e := old.String()
	serviceMust(t, e)
	if got != "abc" {
		t.Fatal(got)
	}
}
func TestOwnedStringCStringConcat(t *testing.T) {
	for _, n := range []int{0, 12, 15, 16, 30, 120} {
		var rhs, result OwnedString
		serviceMust(t, rhs.ConstructRepeated(nil, uint64(n), 'x'))
		serviceMust(t, result.ConstructCStringConcat(nil, "#b\x00ignored", rhs.View()))
		if !result.Matches("#b"+strings.Repeat("x", n)) || !rhs.Matches(strings.Repeat("x", n)) {
			t.Fatal(n)
		}
		rhs.SourceDestroy()
		if !result.Matches("#b" + strings.Repeat("x", n)) {
			t.Fatal("returned lifetime")
		}
	}
}
func TestOwnedStringTransferAuthorityAndBounds(t *testing.T) {
	var s, src OwnedString
	serviceMust(t, s.Construct(nil, "target"))
	serviceMust(t, src.Construct(arena.NewBudget(0), "source"))
	copied := s
	if !errors.Is(copied.Reserve(20), ErrOwnedStringOwner) || !errors.Is(s.MoveAssign(&copied), ErrOwnedStringOwner) {
		t.Fatal("copied owner")
	}
	// Go-managed storage has no foreign-allocator restriction.
	serviceMust(t, s.MoveAssign(&src))
	if !s.Matches("source") {
		t.Fatal("cross-context move")
	}
	serviceMust(t, s.MoveAssign(&s))
	if !s.Matches("source") {
		t.Fatal("self assignment")
	}
	s.Generation = ^uint64(0)
	if !errors.Is(s.MoveAssign(&src), arena.ErrBounds) {
		t.Fatal("generation overflow")
	}
	if !errors.Is(s.Reserve(uint64(^uint(0)>>1)), arena.ErrBounds) {
		t.Fatal("host bounds")
	}
	var absent *OwnedString
	if !errors.Is(absent.MoveAssign(&src), ErrOwnedStringOwner) || !errors.Is(absent.Reserve(0), ErrOwnedStringOwner) {
		t.Fatal("nil owner")
	}
}

func TestOwnedStringCopyAndReconstructionLifetime(t *testing.T) {
	var original, copy OwnedString
	serviceMust(t, original.ConstructRepeated(nil, 32, 'x'))
	original.Put(1, 0)
	serviceMust(t, copy.CopyConstruct(nil, &original))
	original.Put(2, 'z')
	ch, e := copy.View().Byte(2)
	serviceMust(t, e)
	if ch != 'x' {
		t.Fatal("copy aliases original")
	}
	old := original.View()
	original.SourceDestroy()
	serviceMust(t, original.Construct(nil, "new"))
	if _, e = old.Byte(0); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("reconstruction revived stale view", e)
	}
	serviceMust(t, copy.CopyAssign(&original))
	if !copy.Matches("new") {
		t.Fatal("copy assignment")
	}
	before := copy.View()
	serviceMust(t, copy.CopyAssign(&copy))
	text, e := before.String()
	serviceMust(t, e)
	if text != "new" {
		t.Fatal("self assignment")
	}
}
