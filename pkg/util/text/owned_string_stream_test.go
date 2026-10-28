package text

// Source: src/util/mpz.cpp:1432-1435.

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestOwnedStringStreamIndependentLengthsAndReturnedString(t *testing.T) {
	for _, n := range []int{0, 1, 14, 15, 16, 17, 29, 30, 31, 511, 512, 513, 1023, 1024, 1025, 2048, 2049} {
		b := arena.NewBudget(10000)
		var out OwnedStringStream
		textMust(t, out.Construct(b))
		text := strings.Repeat("abc", (n+2)/3)[:n]
		textMust(t, out.AppendCString(text))
		if out.Buffer.Position != uint64(n) || out.Buffer.String.Length != uint64(n) {
			t.Fatal(n, out.Buffer.Position, out.Buffer.String.Length)
		}
		var result OwnedString
		textMust(t, out.Str(&result))
		got, e := result.View().String()
		textMust(t, e)
		if got != text {
			t.Fatal(n, "result", got)
		}
		if b.Used() != result.StorageBytes() {
			t.Fatal("returned output accounting")
		}
		// Returned string owns separate backing and survives destruction of buffer.
		out.OwnAbort()
		got, e = result.View().String()
		textMust(t, e)
		if got != text {
			t.Fatal("borrowed return")
		}
		result.SourceDestroy()
		if b.Used() != 0 {
			t.Fatal("leak", n, b.Used())
		}
	}
}

func TestOwnedStringStreamDecimalAndCountedReturn(t *testing.T) {
	b := arena.NewBudget(1000)
	var out OwnedStringStream
	textMust(t, out.Construct(b))
	var expected strings.Builder
	for _, n := range []int32{math.MinInt32, -1234567, -1, 0, 1, 1234567, math.MaxInt32} {
		textMust(t, out.AppendInt32(n))
		textMust(t, out.AppendCString("|\x00ignored"))
		fmt.Fprintf(&expected, "%d|", n)
	}
	// Counted native text preserves embedded NUL at str().
	textMust(t, out.appendText("\x00z"))
	var result OwnedString
	textMust(t, out.Str(&result))
	got, e := result.View().String()
	textMust(t, e)
	if got != expected.String()+"\x00z" {
		t.Fatal([]byte(got))
	}
	out.OwnAbort()
	result.SourceDestroy()
	if b.Used() != 0 {
		t.Fatal("decimal leak")
	}
}

func TestOwnedStringStreamGoBuilderIgnoresArenaFailures(t *testing.T) {
	for _, n := range []int{0, 15, 16, 512, 513, 4096} {
		b := arena.NewBudget(0)
		b.FailAfter(0)
		var out OwnedStringStream
		textMust(t, out.Construct(b))
		text := strings.Repeat("x", n)
		textMust(t, out.AppendCString(text))
		textMust(t, out.AppendInt64(math.MinInt64))
		got, e := (OwnedStringView{Owner: &out.Buffer.String, Generation: out.Buffer.String.Generation, Length: out.Buffer.Position}).String()
		textMust(t, e)
		if got != text+"-9223372036854775808" || b.Used() != 0 {
			t.Fatal(n, got, b.Used())
		}
		out.OwnAbort()
	}
}

func TestOwnedStringStreamOwnerAndNativeTextGuards(t *testing.T) {
	var out OwnedStringStream
	textMust(t, out.Construct(nil))
	copied := out
	if !errors.Is(copied.AppendInt32(1), ErrOwnedStringOwner) {
		t.Fatal("copied stream")
	}
	if !errors.Is(out.Construct(nil), ErrOwnedStringOwner) {
		t.Fatal("double construct")
	}
	textMust(t, out.AppendNumeralText("17"))
	if out.Buffer.Position != 2 {
		t.Fatal("native numeral text", out.Buffer.Position)
	}
	out.OwnAbort()
	if !errors.Is(out.AppendInt32(1), ErrOwnedStringOwner) {
		t.Fatal("closed stream")
	}
}
