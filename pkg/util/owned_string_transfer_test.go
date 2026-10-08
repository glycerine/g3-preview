// Source: selected basic_string.h:858-921,3570-3588; basic_string.tcc:305-321.
package util

import (
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"strings"
	"testing"
)

func TestOwnedStringReserveAndMoveAssignment(t *testing.T) {
	for _, a := range []int{0, 5, 15, 16, 31} {
		for _, bLen := range []int{0, 5, 15, 16, 31} {
			b := arena.NewBudget(1 << 20)
			var dst, src OwnedString
			serviceMust(t, dst.Construct(b, LiteralParamsStringView(strings.Repeat("a", a))))
			serviceMust(t, src.Construct(b, LiteralParamsStringView(strings.Repeat("b", bLen))))
			oldDst, oldSrc := dst.View(), src.View()
			dstBacking, srcBacking := dst.External, src.External
			used := b.Used()
			b.FailAfter(0)
			serviceMust(t, dst.MoveAssign(&src))
			if !dst.Matches(strings.Repeat("b", bLen)) || src.Length != 0 || !src.Ready || src.Closed || b.Used() != used {
				t.Fatal("move bytes/ownership", a, bLen)
			}
			if bLen > 15 {
				if &dst.External[0] != &srcBacking[0] {
					t.Fatal("move did not transfer backing")
				}
				if a > 15 && &src.External[0] != &dstBacking[0] {
					t.Fatal("move did not swap old backing")
				}
			} else if a > 15 && &dst.External[0] != &dstBacking[0] {
				t.Fatal("short move lost destination backing")
			}
			if a > 0 {
				if _, e := oldDst.Byte(0); !errors.Is(e, arena.ErrInvalidSlot) {
					t.Fatal("old destination view", e)
				}
			}
			if bLen > 0 {
				if _, e := oldSrc.Byte(0); !errors.Is(e, arena.ErrInvalidSlot) {
					t.Fatal("old source view", e)
				}
			}
			src.SourceDestroy()
			dst.SourceDestroy()
			if b.Used() != 0 {
				t.Fatal("move refunds", b.Used())
			}
		}
	}
	b := arena.NewBudget(1 << 20)
	var s OwnedString
	serviceMust(t, s.Construct(b, LiteralParamsStringView("abc")))
	old := s.View()
	serviceMust(t, s.Reserve(16))
	if s.Capacity != 30 || s.Length != 3 || !s.Matches("abc") || b.Used() != 31 {
		t.Fatal("reserve growth")
	}
	if _, e := old.Byte(0); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("reserve borrow", e)
	}
	before := s.View()
	used := b.Used()
	b.FailAfter(0)
	serviceMust(t, s.Reserve(29))
	if ch, e := before.Byte(0); e != nil || ch != 'a' {
		t.Fatal("no-op reserve borrow", e)
	}
	if e := s.Reserve(31); !errors.Is(e, arena.ErrInjectedFailure) || b.Used() != used || !s.Matches("abc") {
		t.Fatal("failed reserve", e)
	}
	s.OwnAbort()
	if b.Used() != 0 {
		t.Fatal("reserve refunds")
	}
}

func TestOwnedStringCStringConcat(t *testing.T) {
	for _, n := range []int{0, 12, 13, 14, 15, 16, 28, 29, 30, 31, 120} {
		b := arena.NewBudget(1 << 20)
		var rhs, result OwnedString
		serviceMust(t, rhs.ConstructRepeated(b, uint64(n), 'x'))
		serviceMust(t, result.ConstructCStringConcat(b, LiteralParamsStringView("#b"), rhs.View()))
		if !result.Matches("#b"+strings.Repeat("x", n)) || !rhs.Matches(strings.Repeat("x", n)) {
			t.Fatal("concat", n)
		}
		cap := uint64(15)
		if n+2 > 15 {
			cap = uint64(n + 2)
			if cap < 30 {
				cap = 30
			}
		}
		if result.Capacity != cap {
			t.Fatal("concat reserve capacity", n, result.Capacity, cap)
		}
		result.SourceDestroy()
		rhs.SourceDestroy()
		if b.Used() != 0 {
			t.Fatal("concat refunds")
		}
	}
	b := arena.NewBudget(1 << 20)
	var rhs, result OwnedString
	serviceMust(t, rhs.ConstructRepeated(b, 16, 'x'))
	used := b.Used()
	b.FailAfter(0)
	if e := result.ConstructCStringConcat(b, LiteralParamsStringView("#b"), rhs.View()); !errors.Is(e, arena.ErrInjectedFailure) || !result.Ready || result.Length != 0 || result.External != nil || b.Used() != used {
		t.Fatal("concat failed reserve prefix", e)
	}
	result.OwnAbort()
	rhs.OwnAbort()
	if b.Used() != 0 {
		t.Fatal("concat failure refunds")
	}
}

func TestOwnedStringTransferAuthorityAndBounds(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	var s, src OwnedString
	serviceMust(t, s.Construct(b, LiteralParamsStringView("target")))
	serviceMust(t, src.Construct(b, LiteralParamsStringView("source")))
	copyOwner := s
	if e := copyOwner.Reserve(20); !errors.Is(e, ErrParamsPrecondition) {
		t.Fatal("copied reserve", e)
	}
	if e := s.MoveAssign(&copyOwner); !errors.Is(e, ErrParamsPrecondition) {
		t.Fatal("copied move", e)
	}
	other := arena.NewBudget(1 << 20)
	var foreign OwnedString
	serviceMust(t, foreign.Construct(other, LiteralParamsStringView("foreign")))
	if e := s.MoveAssign(&foreign); !errors.Is(e, ErrParamsPrecondition) || !s.Matches("target") || !foreign.Matches("foreign") {
		t.Fatal("foreign allocator move", e)
	}
	foreign.SourceDestroy()
	s.Generation = ^uint64(0)
	if e := s.MoveAssign(&src); !errors.Is(e, arena.ErrBounds) || !s.Matches("target") || !src.Matches("source") {
		t.Fatal("move generation overflow", e)
	}
	if e := s.Reserve(20); !errors.Is(e, arena.ErrBounds) || s.External != nil {
		t.Fatal("reserve generation overflow", e)
	}
	s.Generation = 1
	if e := s.Reserve(uint64(^uint(0) >> 1)); !errors.Is(e, arena.ErrBounds) || !s.Matches("target") {
		t.Fatal("reserve host bounds", e)
	}
	serviceMust(t, s.MoveAssign(&s))
	if s.Length != 0 || s.Capacity != 15 || s.Closed {
		t.Fatal("self move clear")
	}
	var nilString *OwnedString
	if e := nilString.MoveAssign(&src); !errors.Is(e, ErrParamsPrecondition) {
		t.Fatal("nil move", e)
	}
	if e := nilString.Reserve(0); !errors.Is(e, ErrParamsPrecondition) {
		t.Fatal("nil reserve", e)
	}
	s.SourceDestroy()
	src.SourceDestroy()
	if b.Used() != 0 || other.Used() != 0 {
		t.Fatal("authority refunds")
	}
}
