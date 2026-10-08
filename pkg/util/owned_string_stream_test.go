// Source: src/util/mpz.cpp:1432-1435; selected libstdc++ bits/sstream.tcc:80-153.
package util

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestOwnedStringStreamIndependentHighWaterAndReturnedString(t *testing.T) {
	for _, n := range []int{0, 1, 14, 15, 16, 17, 29, 30, 31, 511, 512, 513, 1023, 1024, 1025, 2048, 2049} {
		b := arena.NewBudget(10000)
		var out OwnedStringStream
		serviceMust(t, out.Construct(b))
		text := strings.Repeat("abc", (n+2)/3)[:n]
		serviceMust(t, out.AppendCString(text))
		capWant := uint64(15)
		lengthWant := uint64(0)
		if n > 15 {
			capWant = 512
			lengthWant = 16
		}
		for capWant < uint64(n) {
			lengthWant = capWant + 1
			capWant *= 2
		}
		if out.Buffer.Position != uint64(n) || out.Buffer.String.Capacity != capWant || out.Buffer.String.Length != lengthWant {
			t.Fatal(n, out.Buffer.Position, out.Buffer.String.Capacity, out.Buffer.String.Length)
		}
		var result OwnedString
		serviceMust(t, out.Str(&result))
		got, e := result.View().String()
		serviceMust(t, e)
		if got != text {
			t.Fatal(n, "result", got)
		}
		retCap := uint64(15)
		if n > 15 {
			retCap = uint64(n)
			if retCap < 30 {
				retCap = 30
			}
		}
		if result.Capacity != retCap {
			t.Fatal(n, "returned capacity", result.Capacity)
		}
		if b.Used() != uint64(len(out.Buffer.String.External)+len(result.External)) {
			t.Fatal("physical accounting")
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
	// A put byte, unlike a C-string insertion, preserves embedded NUL at str().
	serviceMust(t, out.emit(0))
	serviceMust(t, out.emit('z'))
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

func TestOwnedStringStreamEveryGrowthAndReturnReservation(t *testing.T) {
	for _, n := range []int{16, 513, 1025} {
		reachedSuccess := false
		for fail := int64(0); fail < 8; fail++ {
			b := arena.NewBudget(10000)
			var out OwnedStringStream
			serviceMust(t, out.Construct(b))
			b.FailAfter(fail)
			var result OwnedString
			e := out.AppendCString(strings.Repeat("x", n))
			if e == nil {
				e = out.Str(&result)
			}
			if b.Used() != uint64(len(out.Buffer.String.External)+len(out.Buffer.String.Pending)+len(result.External)+len(result.Pending)) {
				t.Fatal(n, fail, "actual graph", b.Used())
			}
			if e != nil {
				if !errors.Is(e, arena.ErrInjectedFailure) {
					t.Fatal(n, fail, e)
				}
				pos := out.Buffer.Position
				if out.AppendInt32(123) != e || out.Buffer.Position != pos {
					t.Fatal("sticky replay")
				}
				if pos < uint64(n) {
					if pos != 15 && pos != 512 && pos != 1024 {
						t.Fatal("wrong failure prefix", pos)
					}
					for i := uint64(0); i < pos; i++ {
						ch, err := (OwnedStringView{&out.Buffer.String, out.Buffer.String.Generation, pos}).Byte(i)
						serviceMust(t, err)
						if ch != 'x' {
							t.Fatal("lost prefix")
						}
					}
				} else if !result.Ready || result.Length != 0 {
					t.Fatal("return allocation prefix")
				}
			} else {
				got, err := result.View().String()
				serviceMust(t, err)
				if got != strings.Repeat("x", n) {
					t.Fatal("successful result")
				}
				reachedSuccess = true
			}
			b.FailAfter(-1)
			result.OwnAbort()
			out.OwnAbort()
			if b.Used() != 0 {
				t.Fatal("owning refund", n, fail, b.Used())
			}
			if reachedSuccess {
				break
			}
		}
		if !reachedSuccess {
			t.Fatal("no successful completion", n)
		}
	}
}

func TestOwnedStringStreamOwnerAndBorrowGuards(t *testing.T) {
	var out OwnedStringStream
	serviceMust(t, out.Construct(nil))
	copied := out
	if !errors.Is(copied.AppendInt32(1), ErrParamsPrecondition) {
		t.Fatal("copied stream")
	}
	if !errors.Is(out.Construct(nil), ErrParamsPrecondition) {
		t.Fatal("double construct")
	}
	if e := out.AppendNumeralCString(NumeralCStringView{}); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("stale numeral", e)
	}
	if out.Buffer.Position != 0 {
		t.Fatal("stale numeral output")
	}
	if !errors.Is(out.AppendCString("x"), arena.ErrInvalidSlot) {
		t.Fatal("replayed invalid numeral")
	}
	out.OwnAbort()
	if !errors.Is(out.AppendInt32(1), ErrParamsPrecondition) {
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
				actual := p.header + p.m.HeaderBytes + integerActualOwnedBytes(p.m.Z) + uint64(len(out.Buffer.String.External)+len(out.Buffer.String.Pending)+len(result.External)+len(result.Pending))
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
