// Source: src/util/mpf.cpp:1639-1654,1673,1686.
package text

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"strings"
	"testing"
)

func TestOwnedStringRepeatedConstructorAndReverse(t *testing.T) {
	for _, n := range []uint64{0, 1, 2, 14, 15, 16, 31, 64, 120, 1024} {
		b := arena.NewBudget(0)
		b.FailAfter(0)
		var s OwnedString
		textMust(t, s.ConstructRepeated(b, n, '1'))
		if s.Length != n || !s.Matches(strings.Repeat("1", int(n))) || b.Used() != 0 {
			t.Fatal(n)
		}
		for i := uint64(0); i < n; i++ {
			s.Put(i, byte(i%251))
		}
		view := s.View()
		textMust(t, s.Reverse())
		for i := uint64(0); i < n; i++ {
			ch, e := view.Byte(i)
			textMust(t, e)
			if ch != byte((n-1-i)%251) {
				t.Fatal(n, i, ch)
			}
		}
		textMust(t, s.Reverse())
		for i := uint64(0); i < n; i++ {
			ch, e := view.Byte(i)
			textMust(t, e)
			if ch != byte(i%251) {
				t.Fatal(n, i, ch)
			}
		}
		s.SourceDestroy()
	}
}
func TestOwnedStringRepeatedAllByteValues(t *testing.T) {
	for ch := 0; ch < 256; ch++ {
		for _, n := range []uint64{0, 1, 15, 16, 31} {
			var s OwnedString
			textMust(t, s.ConstructRepeated(nil, n, byte(ch)))
			for i := uint64(0); i < n; i++ {
				got, e := s.View().Byte(i)
				textMust(t, e)
				if got != byte(ch) {
					t.Fatal(ch, n, i)
				}
			}
			got, e := s.CStringByte(n)
			textMust(t, e)
			if got != 0 {
				t.Fatal("terminator")
			}
			s.SourceDestroy()
		}
	}
}
func TestOwnedStringRepeatedConstructorFailureAndAuthority(t *testing.T) {
	var s OwnedString
	if e := s.ConstructRepeated(nil, uint64(^uint(0)>>1), 'x'); !errors.Is(e, arena.ErrBounds) || s.Ready {
		t.Fatal(e)
	}
	textMust(t, s.ConstructRepeated(nil, 120, 0))
	copied := s
	if !errors.Is(copied.Reverse(), ErrOwnedStringOwner) {
		t.Fatal("copied owner")
	}
	if !errors.Is(s.ConstructRepeated(nil, 1, 'x'), ErrOwnedStringOwner) {
		t.Fatal("reconstruction")
	}
	s.OwnAbort()
	if !errors.Is(s.Reverse(), ErrOwnedStringOwner) {
		t.Fatal("closed owner")
	}
	var absent *OwnedString
	if !errors.Is(absent.ConstructRepeated(nil, 1, 'x'), ErrOwnedStringOwner) || !errors.Is(absent.Reverse(), ErrOwnedStringOwner) {
		t.Fatal("nil owner")
	}
}
