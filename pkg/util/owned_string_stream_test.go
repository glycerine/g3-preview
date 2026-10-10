// Source: src/util/mpz.cpp:1432-1435.
package util

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
		serviceMust(t, out.Construct(b))
		text := strings.Repeat("abc", (n+2)/3)[:n]
		serviceMust(t, out.AppendCString(text))
		if out.Buffer.Position != uint64(n) || out.Buffer.String.Length != uint64(n) {
			t.Fatal(n, out.Buffer.Position, out.Buffer.String.Length)
		}
		var result OwnedString
		serviceMust(t, out.Str(&result))
		got, e := result.View().String()
		serviceMust(t, e)
		if got != text {
			t.Fatal(n, "result", got)
		}
		if b.Used() != result.StorageBytes() {
			t.Fatal("returned output accounting")
		}
		// Returned string owns separate backing and survives destruction of buffer.
		out.OwnAbort()
		got, e = result.View().String()
		serviceMust(t, e)
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
	serviceMust(t, out.Construct(b))
	var expected strings.Builder
	for _, n := range []int32{math.MinInt32, -1234567, -1, 0, 1, 1234567, math.MaxInt32} {
		serviceMust(t, out.AppendInt32(n))
		serviceMust(t, out.AppendCString("|\x00ignored"))
		fmt.Fprintf(&expected, "%d|", n)
	}
	// Counted native text preserves embedded NUL at str().
	serviceMust(t, out.appendText("\x00z"))
	var result OwnedString
	serviceMust(t, out.Str(&result))
	got, e := result.View().String()
	serviceMust(t, e)
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
		serviceMust(t, out.Construct(b))
		text := strings.Repeat("x", n)
		serviceMust(t, out.AppendCString(text))
		serviceMust(t, out.AppendInt64(math.MinInt64))
		got, e := (OwnedStringView{&out.Buffer.String, out.Buffer.String.Generation, out.Buffer.Position}).String()
		serviceMust(t, e)
		if got != text+"-9223372036854775808" || b.Used() != 0 {
			t.Fatal(n, got, b.Used())
		}
		out.OwnAbort()
	}
}

func TestOwnedStringStreamOwnerAndNativeTextGuards(t *testing.T) {
	var out OwnedStringStream
	serviceMust(t, out.Construct(nil))
	copied := out
	if !errors.Is(copied.AppendInt32(1), ErrOwnedStringOwner) {
		t.Fatal("copied stream")
	}
	if !errors.Is(out.Construct(nil), ErrOwnedStringOwner) {
		t.Fatal("double construct")
	}
	serviceMust(t, out.AppendNumeralText("17"))
	if out.Buffer.Position != 2 {
		t.Fatal("native numeral text", out.Buffer.Position)
	}
	out.OwnAbort()
	if !errors.Is(out.AppendInt32(1), ErrOwnedStringOwner) {
		t.Fatal("closed stream")
	}
}

func TestOwnedStringStreamSourceIntegerEveryColdReservation(t *testing.T) {
	for _, sync := range []bool{false, true} {
		for _, text := range []string{"-2147483648", "123456789012345678901234567890123456789", "-" + strings.Repeat("9", 600)} {
			success := false
			for fail := int64(0); fail < 100; fail++ {
				p, b := mpqInfFixture(t, sync)
				x := mpqInfValue(t, p, text, "0")
				var out OwnedStringStream
				serviceMust(t, out.Construct(b))
				b.FailAfter(fail)
				var result OwnedString
				e := p.m.Z.insertSourceInteger(&out, &x.First.Num)
				if e == nil {
					e = out.Str(&result)
				}
				qopsAuditLiveOwnership(t, p.m, []*Int{&x.First.Num, &x.First.Den, &x.Second.Num, &x.Second.Den})
				actual := p.header + p.m.HeaderBytes + integerActualOwnedBytes(p.m.Z) + uint64(out.Buffer.String.StorageBytes()+result.StorageBytes())
				if b.Used() != actual {
					t.Fatal(sync, len(text), fail, "immediate ownership", b.Used(), actual)
				}
				if e != nil {
					if !numericPhysicalReservationRefusal(e) {
						t.Fatal(sync, len(text), fail, e)
					}
				} else {
					got, err := result.View().String()
					serviceMust(t, err)
					if got != text {
						t.Fatal("source integer result", got, text)
					}
					success = true
				}
				b.FailAfter(-1)
				serviceMust(t, p.m.Z.Natural.cleanupOutputFormatting())
				result.OwnAbort()
				out.OwnAbort()
				serviceMust(t, p.Del(&x))
				serviceMust(t, p.Close())
				if b.Used() != 0 {
					t.Fatal("source integer refund", b.Used())
				}
				if success {
					break
				}
			}
			if !success {
				t.Fatal("no successful source integer", sync, len(text))
			}
		}
	}
}
