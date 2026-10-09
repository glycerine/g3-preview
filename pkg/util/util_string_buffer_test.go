// Source: src/util/string_buffer.h:48-188.
package util

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestStringBuffer64TextNumbersAndReset(t *testing.T) {
	b, e := NewStringBuffer64(StringBufferConfig{})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if e := b.Close(); e != nil {
			t.Fatal(e)
		}
	})
	for _, appendValue := range []func() error{
		func() error { return b.AppendCString("prefix:") },
		func() error { return b.AppendInt32(-17) },
		func() error { return b.AppendByte('/') },
		func() error { return b.AppendUInt32(math.MaxUint32) },
		func() error { return b.AppendByte('/') },
		func() error { return b.AppendLong(math.MinInt64) },
		func() error { return b.AppendByte('/') },
		func() error { return b.AppendBool(true) },
	} {
		if e := appendValue(); e != nil {
			t.Fatal(e)
		}
	}
	got, e := b.CStr()
	if e != nil {
		t.Fatal(e)
	}
	if got != "prefix:-17/-1/-9223372036854775808/true" {
		t.Fatal(got)
	}
	if b.Empty() || b.Size() != uint32(len(got)) {
		t.Fatal("size", b.Size())
	}
	if e = b.Reset(); e != nil {
		t.Fatal(e)
	}
	if !b.Empty() {
		t.Fatal("reset retained text")
	}
}

func TestStringBuffer64AppendAndEmbeddedNUL(t *testing.T) {
	a, e := NewStringBuffer64(StringBufferConfig{})
	if e != nil {
		t.Fatal(e)
	}
	b, e := NewStringBuffer64(StringBufferConfig{})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		if e := b.Close(); e != nil {
			t.Fatal(e)
		}
		if e := a.Close(); e != nil {
			t.Fatal(e)
		}
	})
	if e = a.AppendCString("left"); e != nil {
		t.Fatal(e)
	}
	if e = b.AppendCString("right"); e != nil {
		t.Fatal(e)
	}
	if e = a.AppendBuffer64(b); e != nil {
		t.Fatal(e)
	}
	if e = a.AppendByte(0); e != nil {
		t.Fatal(e)
	}
	if e = a.AppendCString("hidden"); e != nil {
		t.Fatal(e)
	}
	if got, e := a.CStr(); e != nil || got != "leftright" {
		t.Fatal(got, e)
	}
}

func TestStringBuffer64LifecycleAndLongProfile(t *testing.T) {
	if b, e := NewStringBuffer64(StringBufferConfig{LongBits: 16}); b != nil || !errors.Is(e, ErrStringBufferConfig) {
		t.Fatal(b, e)
	}
	b, e := NewStringBuffer64(StringBufferConfig{LongBits: 32})
	if e != nil {
		t.Fatal(e)
	}
	if e = b.AppendLong(math.MaxInt32 + 1); !errors.Is(e, ErrStringBufferLongRange) {
		t.Fatal(e)
	}
	if e = b.Close(); e != nil {
		t.Fatal(e)
	}
	if e = b.AppendByte('x'); !errors.Is(e, arena.ErrClosed) {
		t.Fatal(e)
	}
}
