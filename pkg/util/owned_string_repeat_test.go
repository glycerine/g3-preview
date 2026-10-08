// Source: selected basic_string.tcc:258-272; mpf.cpp:1639-1654,1673,1686.
package util

import (
	"errors"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestOwnedStringRepeatedConstructorAndReverse(t *testing.T) {
	for _, n := range []uint64{0, 1, 2, 14, 15, 16, 29, 30, 31, 64, 120, 1024} {
		b := arena.NewBudget(1 << 20)
		var s OwnedString
		serviceMust(t, s.ConstructRepeated(b, n, '1'))
		wantCapacity := uint64(15)
		wantBytes := uint64(0)
		if n > 15 {
			wantCapacity = n
			wantBytes = n + 1
		}
		if s.Length != n || s.Capacity != wantCapacity || b.Used() != wantBytes || !s.Matches(strings.Repeat("1", int(n))) {
			t.Fatal("repeated constructor profile", n, s.Capacity, b.Used())
		}
		// Exercise byte swaps and keep the existing borrow valid: reversal does not
		// replace storage or change the source string's length/capacity/generation.
		for i := uint64(0); i < n; i++ {
			s.Put(i, byte(i%251))
		}
		view, generation, used := s.View(), s.Generation, b.Used()
		b.FailAfter(0)
		serviceMust(t, s.Reverse())
		if (s.External != nil && s.External[n] != 0) || (s.External == nil && s.Inline[n] != 0) {
			t.Fatal("reverse changed trailing NUL", n)
		}
		if s.Generation != generation || b.Used() != used || s.Length != n || s.Capacity != wantCapacity {
			t.Fatal("reverse replaced backing", n)
		}
		for i := uint64(0); i < n; i++ {
			v, e := view.Byte(i)
			serviceMust(t, e)
			if v != byte((n-1-i)%251) {
				t.Fatal("reverse bytes", n, i, v)
			}
		}
		serviceMust(t, s.Reverse())
		for i := uint64(0); i < n; i++ {
			v, e := view.Byte(i)
			serviceMust(t, e)
			if v != byte(i%251) {
				t.Fatal("second reverse", n, i, v)
			}
		}
		b.FailAfter(-1)
		s.SourceDestroy()
		if b.Used() != 0 {
			t.Fatal("repeat refunds", n, b.Used())
		}
	}
}

func TestOwnedStringRepeatedAllByteValues(t *testing.T) {
	for ch := 0; ch < 256; ch++ {
		for _, n := range []uint64{0, 1, 15, 16, 31} {
			b := arena.NewBudget(128)
			var s OwnedString
			serviceMust(t, s.ConstructRepeated(b, n, byte(ch)))
			view := s.View()
			for i := uint64(0); i < n; i++ {
				actual, e := view.Byte(i)
				serviceMust(t, e)
				if actual != byte(ch) {
					t.Fatal("repeat byte", ch, n, i, actual)
				}
			}
			if (s.External != nil && s.External[n] != 0) || (s.External == nil && s.Inline[n] != 0) {
				t.Fatal("constructor trailing NUL", ch, n)
			}
			s.SourceDestroy()
			if b.Used() != 0 {
				t.Fatal("repeat retained backing", ch, n, b.Used())
			}
		}
	}
}

func TestOwnedStringRepeatedConstructorFailureAndAuthority(t *testing.T) {
	for _, n := range []uint64{0, 15, 16, 120} {
		b := arena.NewBudget(1 << 20)
		b.FailAfter(0)
		var s OwnedString
		e := s.ConstructRepeated(b, n, 0)
		if n <= 15 {
			serviceMust(t, e)
			if !s.Ready || s.Length != n {
				t.Fatal("inline repeat failed")
			}
		} else if !errors.Is(e, arena.ErrInjectedFailure) || s.Ready || s.External != nil || s.Length != 0 || b.Used() != 0 {
			t.Fatal("failed constructor prefix", n, e)
		}
		if e := s.Reverse(); n > 15 && !errors.Is(e, ErrParamsPrecondition) {
			t.Fatal("failed constructor gained reverse", e)
		}
		copy := s
		if e := copy.Reverse(); !errors.Is(e, ErrParamsPrecondition) {
			t.Fatal("copied string gained reverse", e)
		}
		if e := s.ConstructRepeated(b, 1, 'x'); !errors.Is(e, ErrParamsPrecondition) {
			t.Fatal("repeat reinitialized live/failed owner", e)
		}
		s.OwnAbort()
		if e := s.Reverse(); !errors.Is(e, ErrParamsPrecondition) {
			t.Fatal("closed string gained reverse", e)
		}
		if b.Used() != 0 {
			t.Fatal("failed repeat refunds", b.Used())
		}
	}
	b := arena.NewBudget(1 << 20)
	var s OwnedString
	if e := s.ConstructRepeated(b, uint64(^uint(0)>>1), 'x'); !errors.Is(e, arena.ErrBounds) || s.Ready || b.Used() != 0 {
		t.Fatal("size bound", e)
	}
	s.OwnAbort()
	var nilString *OwnedString
	if e := nilString.ConstructRepeated(b, 1, 'x'); !errors.Is(e, ErrParamsPrecondition) {
		t.Fatal("nil constructor", e)
	}
	if e := nilString.Reverse(); !errors.Is(e, ErrParamsPrecondition) {
		t.Fatal("nil reverse", e)
	}
}
