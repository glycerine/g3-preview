package text

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"strings"
	"testing"
)

func TestOwnedStringAppendIndependentLengthsAndSelf(t *testing.T) {
	for a := 0; a <= 70; a++ {
		for n := 0; n <= 70; n++ {
			b := arena.NewBudget(0)
			b.FailAfter(0)
			var dst, src OwnedString
			first, second := strings.Repeat("a", a), strings.Repeat("b", n)
			textMust(t, dst.Construct(b, first))
			textMust(t, src.Construct(b, second))
			textMust(t, dst.Append(src.View()))
			if !dst.Matches(first+second) || !src.Matches(second) || b.Used() != 0 {
				t.Fatal(a, n)
			}
			dst.SourceDestroy()
			src.SourceDestroy()
		}
	}
	for n := 0; n <= 80; n++ {
		var s OwnedString
		before := strings.Repeat("x", n)
		textMust(t, s.Construct(nil, before))
		textMust(t, s.Append(s.View()))
		if !s.Matches(before + before) {
			t.Fatal(n)
		}
	}
}
func TestOwnedStringAppendCountedBinaryAndCString(t *testing.T) {
	var s, src OwnedString
	textMust(t, s.Construct(nil, "a"))
	textMust(t, src.ConstructRepeated(nil, 3, 'x'))
	src.Put(1, 0)
	src.Put(2, 'z')
	textMust(t, s.Append(src.View()))
	textMust(t, s.AppendCString("b\x00ignored"))
	if !s.Matches("ax\x00zb") {
		t.Fatal(s.External)
	}
	textMust(t, s.Append(OwnedStringView{}))
	if ch, e := s.CStringByte(s.Length); e != nil || ch != 0 {
		t.Fatal(ch, e)
	}
}
func TestOwnedStringAppendRefusalAndRetainedReplacement(t *testing.T) {
	// Invalid borrowed input must leave existing Go text intact; no STL pending
	// replacement or allocator failure state is involved.
	var s, src OwnedString
	textMust(t, s.Construct(nil, strings.Repeat("a", 40)))
	textMust(t, src.Construct(nil, strings.Repeat("b", 50)))
	stale := src.View()
	src.SourceDestroy()
	old := s.View()
	if e := s.Append(stale); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal(e)
	}
	got, e := old.String()
	textMust(t, e)
	if got != strings.Repeat("a", 40) {
		t.Fatal(got)
	}
	textMust(t, s.AppendCString("x"))
	if _, e = old.Byte(0); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal(e)
	}
}
func TestOwnedStringAppendOwnerAndUnreachedReads(t *testing.T) {
	var s OwnedString
	textMust(t, s.Construct(nil, "a"))
	copied := s
	if !errors.Is(copied.AppendCString("x"), ErrOwnedStringOwner) {
		t.Fatal("copied owner")
	}
	var absent *OwnedString
	if !errors.Is(absent.Append(OwnedStringView{}), ErrOwnedStringOwner) {
		t.Fatal("nil receiver")
	}
	if !errors.Is(s.Append(OwnedStringView{Length: ^uint64(0)}), arena.ErrBounds) {
		t.Fatal("overflow")
	}
	if !s.Matches("a") {
		t.Fatal("changed on failure")
	}
	s.SourceDestroy()
	if !errors.Is(s.Append(OwnedStringView{}), ErrOwnedStringOwner) {
		t.Fatal("closed receiver")
	}
}
