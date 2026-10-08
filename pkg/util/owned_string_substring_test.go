// Source: selected libstdc++ bits/basic_string.h:586-594,3208-3211;
// bits/basic_string.tcc:664-677; src/util/mpf.cpp:207-218.
package util

import (
	"bytes"
	"errors"
	"github.com/glycerine/g3/pkg/arena"
	"math/rand/v2"
	"testing"
)

func TestOwnedStringFindByteIndependentCountedBytes(t *testing.T) {
	rng := rand.New(rand.NewChaCha8([32]byte{194, 207, 218}))
	for _, length := range []int{0, 1, 14, 15, 16, 31, 32, 255, 256, 1024} {
		b := arena.NewBudget(1 << 20)
		var src OwnedString
		serviceMust(t, src.ConstructRepeated(b, uint64(length), 0))
		data := make([]byte, length)
		for i := range data {
			data[i] = byte(rng.Uint32())
			src.Put(uint64(i), data[i])
		}
		generation, used := src.Generation, b.Used()
		b.FailAfter(0)
		for ch := 0; ch < 256; ch++ {
			for _, start := range []uint64{0, 1, uint64(length / 2), uint64(length), uint64(length) + 1, OwnedStringNPos} {
				got, e := src.FindByte(byte(ch), start)
				serviceMust(t, e)
				want := OwnedStringNPos
				if start < uint64(length) {
					if i := bytes.IndexByte(data[int(start):], byte(ch)); i >= 0 {
						want = start + uint64(i)
					}
				}
				if got != want {
					t.Fatal("counted find", length, ch, start, got, want)
				}
			}
		}
		if src.Generation != generation || b.Used() != used {
			t.Fatal("find changed borrowed storage")
		}
		src.SourceDestroy()
		if b.Used() != 0 {
			t.Fatal("find refunds")
		}
	}
}

func TestOwnedStringSubstringIndependentPositionsAndLengths(t *testing.T) {
	for _, length := range []int{0, 1, 15, 16, 31, 32, 129, 256} {
		b := arena.NewBudget(1 << 20)
		var src OwnedString
		serviceMust(t, src.ConstructRepeated(b, uint64(length), 0))
		data := make([]byte, length)
		for i := range data {
			data[i] = byte(i)
			src.Put(uint64(i), data[i])
		}
		generation, used := src.Generation, b.Used()
		for pos := 0; pos <= length; pos++ {
			for _, count := range []uint64{0, 1, 14, 15, 16, uint64(length), OwnedStringNPos} {
				var out OwnedString
				serviceMust(t, out.ConstructSubstring(b, &src, uint64(pos), count))
				n := uint64(length - pos)
				if n > count {
					n = count
				}
				want := data[pos : pos+int(n)]
				actual, e := out.View().String()
				serviceMust(t, e)
				capacity, cost := uint64(15), uint64(0)
				if n > 15 {
					capacity = n
					cost = n + 1
				}
				if !bytes.Equal([]byte(actual), want) || out.Length != n || out.Capacity != capacity || !out.Ready || out.Self != &out || b.Used() != used+cost {
					t.Fatal("substring result/storage", length, pos, count, out.Length, out.Capacity)
				}
				if out.External != nil {
					if out.External[n] != 0 {
						t.Fatal("heap NUL")
					}
				} else if out.Inline[n] != 0 {
					t.Fatal("inline NUL")
				}
				if src.Generation != generation || !src.Matches(string(data)) {
					t.Fatal("substring changed borrowed input")
				}
				if n > 0 {
					old, e := src.View().Byte(uint64(pos))
					serviceMust(t, e)
					out.Put(0, old^255)
					got, e := src.View().Byte(uint64(pos))
					serviceMust(t, e)
					if got != old {
						t.Fatal("substring aliases source")
					}
				}
				out.SourceDestroy()
				if b.Used() != used {
					t.Fatal("substring refund")
				}
			}
		}
		src.SourceDestroy()
		if b.Used() != 0 {
			t.Fatal("source refund")
		}
	}
}

func TestOwnedStringSubstringEveryColdReservation(t *testing.T) {
	for _, n := range []uint64{0, 1, 15, 16, 31, 120, 1024} {
		for ordinal := int64(0); ordinal < 2; ordinal++ {
			b := arena.NewBudget(1 << 20)
			var src, out OwnedString
			serviceMust(t, src.ConstructRepeated(b, n+2, 0))
			for i := uint64(0); i < src.Length; i++ {
				src.Put(i, byte(i))
			}
			used := b.Used()
			b.FailAfter(ordinal)
			e := out.ConstructSubstring(b, &src, 1, n)
			if n > 15 && ordinal == 0 {
				if !errors.Is(e, arena.ErrInjectedFailure) || out.Ready || out.External != nil || out.Pending != nil || out.Length != 0 || out.Capacity != 15 || out.Self != &out || b.Used() != used {
					t.Fatal("failed constructor prefix", n, e)
				}
			} else {
				serviceMust(t, e)
				cost := uint64(0)
				if n > 15 {
					cost = n + 1
				}
				if b.Used() != used+cost {
					t.Fatal("actual substring backing charge", n, b.Used())
				}
			}
			out.OwnAbort()
			if b.Used() != used {
				t.Fatal("constructor owning refund")
			}
			src.SourceDestroy()
			if b.Used() != 0 {
				t.Fatal("cold final refund")
			}
		}
	}
}

func TestOwnedStringSubstringAuthorityAndBounds(t *testing.T) {
	b := arena.NewBudget(1 << 20)
	var src, out OwnedString
	serviceMust(t, src.ConstructRepeated(b, 32, 'q'))
	used := b.Used()
	for _, pos := range []uint64{33, OwnedStringNPos} {
		if e := out.ConstructSubstring(b, &src, pos, 0); !errors.Is(e, arena.ErrBounds) || out.Self != nil || b.Used() != used {
			t.Fatal("unreached constructor", pos, e)
		}
	}
	if e := out.ConstructSubstring(b, nil, 0, 0); !errors.Is(e, ErrParamsOwner) || out.Self != nil {
		t.Fatal("nil source", e)
	}
	var nilOut *OwnedString
	if e := nilOut.ConstructSubstring(b, &src, 0, 0); !errors.Is(e, ErrParamsPrecondition) {
		t.Fatal("nil result", e)
	}
	copied := src
	if _, e := copied.FindByte('q', OwnedStringNPos); !errors.Is(e, ErrParamsOwner) {
		t.Fatal("copied source searched", e)
	}
	if e := out.ConstructSubstring(b, &copied, 0, 0); !errors.Is(e, ErrParamsOwner) || out.Self != nil {
		t.Fatal("copied source constructed", e)
	}
	if e := src.ConstructSubstring(b, &src, 0, 1); !errors.Is(e, ErrParamsPrecondition) || b.Used() != used {
		t.Fatal("self reconstruction", e)
	}
	// Only malformed checked-adapter states are diagnosed by these span guards;
	// no native invalid object layout behavior is credited.
	src.Length = OwnedStringNPos
	if _, e := src.FindByte('q', 0); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("length span", e)
	}
	if e := out.ConstructSubstring(b, &src, 0, OwnedStringNPos); !errors.Is(e, arena.ErrBounds) || out.Self != nil {
		t.Fatal("length constructor span", e)
	}
	src.Length = 32
	serviceMust(t, out.ConstructSubstring(b, &src, 0, 16))
	outUsed := b.Used()
	if e := out.ConstructSubstring(b, &src, 0, 1); !errors.Is(e, ErrParamsPrecondition) || out.Length != 16 || b.Used() != outUsed {
		t.Fatal("live reconstruction", e)
	}
	out.SourceDestroy()
	src.SourceDestroy()
	if _, e := src.FindByte('q', OwnedStringNPos); !errors.Is(e, ErrParamsOwner) {
		t.Fatal("closed source searched", e)
	}
	var fresh OwnedString
	if e := fresh.ConstructSubstring(b, &src, 0, 0); !errors.Is(e, ErrParamsOwner) || fresh.Self != nil {
		t.Fatal("closed source constructed", e)
	}
	if b.Used() != 0 {
		t.Fatal("authority refunds", b.Used())
	}
}

func TestOwnedStringSubstringRetainedReplacementAndSeparateBudget(t *testing.T) {
	b, other := arena.NewBudget(1000), arena.NewBudget(1000)
	var src, dead, out OwnedString
	serviceMust(t, src.Construct(b, LiteralParamsStringView("abcdefghijklmnopqrst")))
	serviceMust(t, dead.ConstructRepeated(b, 50, '!'))
	stale := dead.View()
	dead.SourceDestroy()
	if e := src.Append(stale); !errors.Is(e, arena.ErrInvalidSlot) || src.Pending == nil {
		t.Fatal("real pending replacement", e)
	}
	used, generation, pending := b.Used(), src.Generation, len(src.Pending)
	at, e := src.FindByte('t', 0)
	serviceMust(t, e)
	if at != 19 {
		t.Fatal("read surviving source", at)
	}
	serviceMust(t, out.ConstructSubstring(other, &src, 2, 17))
	if !out.Matches("cdefghijklmnopqrs") || other.Used() != 18 || b.Used() != used || src.Generation != generation || len(src.Pending) != pending || !src.Matches("abcdefghijklmnopqrst") {
		t.Fatal("substring stole old/pending backing")
	}
	src.OwnAbort()
	if b.Used() != 0 || !out.Matches("cdefghijklmnopqrs") {
		t.Fatal("independent returned lifetime", b.Used())
	}
	out.SourceDestroy()
	if other.Used() != 0 {
		t.Fatal("separate budget refund")
	}
}

// These are literal dependency sequences from mpf.cpp:207-218, not upstream
// numerical SET assertions (the original driver has none for this overload).
func TestOwnedStringMPFInputSubstringAndFindSequence(t *testing.T) {
	for _, tc := range []struct {
		input, f, e string
		negative    bool
	}{
		{"", "", "0", false}, {"-", "", "0", true}, {"+", "", "0", false},
		{"-1.5p-2", "1.5", "-2", true}, {"+1.5P2", "1.5", "2", false},
		{"1P2p3", "1P2", "3", false}, {"1p2P3", "1", "2P3", false},
		{"--2p0", "-2", "0", true}, {" -2p0", " -2", "0", false},
		{"12345678901234567890", "12345678901234567890", "0", false},
		{"12345678901234567890p-1234567890123456", "12345678901234567890", "-1234567890123456", false},
	} {
		b := arena.NewBudget(1 << 20)
		var v, f, e OwnedString
		serviceMust(t, v.Construct(b, LiteralParamsStringView(tc.input)))
		serviceMust(t, f.ConstructRepeated(b, 0, 0))
		serviceMust(t, e.ConstructRepeated(b, 0, 0))
		var first OwnedString
		serviceMust(t, first.ConstructSubstring(b, &v, 0, 1))
		negative := first.Matches("-")
		first.SourceDestroy()
		positive := false
		if !negative {
			var second OwnedString
			serviceMust(t, second.ConstructSubstring(b, &v, 0, 1))
			positive = second.Matches("+")
			second.SourceDestroy()
		}
		if negative || positive {
			var suffix OwnedString
			serviceMust(t, suffix.ConstructSubstring(b, &v, 1, OwnedStringNPos))
			serviceMust(t, v.MoveAssign(&suffix))
			suffix.SourceDestroy()
		}
		pos, err := v.FindByte('p', 0)
		serviceMust(t, err)
		if pos == OwnedStringNPos {
			pos, err = v.FindByte('P', 0)
			serviceMust(t, err)
		}
		if pos != OwnedStringNPos {
			var fraction, exponent OwnedString
			serviceMust(t, fraction.ConstructSubstring(b, &v, 0, pos))
			serviceMust(t, f.MoveAssign(&fraction))
			fraction.SourceDestroy()
			serviceMust(t, exponent.ConstructSubstring(b, &v, pos+1, OwnedStringNPos))
			serviceMust(t, e.MoveAssign(&exponent))
			exponent.SourceDestroy()
		} else {
			var fraction, exponent OwnedString
			serviceMust(t, fraction.CopyConstruct(b, &v))
			serviceMust(t, f.MoveAssign(&fraction))
			fraction.SourceDestroy()
			serviceMust(t, exponent.Construct(b, LiteralParamsStringView("0")))
			serviceMust(t, e.MoveAssign(&exponent))
			exponent.SourceDestroy()
		}
		wantCapacity := uint64(15)
		if len(tc.f) > 15 {
			wantCapacity = uint64(len(tc.f))
		}
		if negative != tc.negative || !f.Matches(tc.f) || !e.Matches(tc.e) || f.Capacity != wantCapacity {
			t.Fatal("literal MPF string sequence", tc, negative)
		}
		e.SourceDestroy()
		f.SourceDestroy()
		v.SourceDestroy()
		if b.Used() != 0 {
			t.Fatal("MPF string lifetime refunds", b.Used())
		}
	}
}
