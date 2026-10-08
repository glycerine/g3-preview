// Source: src/util/approx_nat.h:26-43; src/util/approx_nat.cpp:22-61.
package util

import (
	"errors"
	"io"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestApproxNatBoundariesAndSourceComparisons(t *testing.T) {
	const limit = uint32(1073741823)
	var zero ApproxNat
	if zero.GetValue() != 0 || zero.IsHuge() {
		t.Fatal("zero-value constructor")
	}
	values := []uint32{0, 1, 2, limit - 1, limit, limit + 1, 1 << 31, ^uint32(0)}
	for _, a := range values {
		x := NewApproxNat(a)
		want := a
		if a > limit {
			want = ^uint32(0)
		}
		if x.GetValue() != want || x.IsHuge() != (a > limit) {
			t.Fatal(a, x)
		}
		copy := x
		if copy.Assign(7) != &copy || copy.GetValue() != 7 || x.GetValue() != want {
			t.Fatal("copy/assignment")
		}
		for _, b := range values {
			y := NewApproxNat(b)
			if x.LessUnsigned(b) != (a <= limit && a < b) {
				t.Fatal("unsigned comparison", a, b)
			}
			if x.Less(y) != (a <= limit && b <= limit && a < b) {
				t.Fatal("class comparison", a, b)
			}
		}
	}
	finite, huge := NewApproxNat(0), NewApproxNat(limit+1)
	finite.MulAssign(huge)
	if finite.IsHuge() {
		t.Fatal("finite zero times huge is zero")
	}
	huge.MulUnsignedAssign(0)
	if !huge.IsHuge() {
		t.Fatal("huge times zero remains huge")
	}
	finite.AddAssign(huge)
	if !finite.IsHuge() {
		t.Fatal("zero plus huge is huge")
	}
}

func TestApproxNatIndependentArithmetic(t *testing.T) {
	const limit = uint64(1073741823)
	check := func(a, b uint32) {
		t.Helper()
		for _, multiply := range []bool{false, true} {
			x := NewApproxNat(a)
			v := uint64(a) + uint64(b)
			if multiply {
				v = uint64(a) * uint64(b)
			}
			want := uint32(v)
			if uint64(a) > limit || (!multiply && uint64(b) > limit) || v > limit {
				want = ^uint32(0)
			}
			var ret *ApproxNat
			if multiply {
				ret = x.MulUnsignedAssign(b)
			} else {
				ret = x.AddUnsignedAssign(b)
			}
			if ret != &x || x.GetValue() != want {
				t.Fatal(a, b, multiply, x.GetValue(), want)
			}
			z := NewApproxNat(a)
			y := NewApproxNat(b)
			if multiply {
				z.MulAssign(y)
			} else {
				z.AddAssign(y)
			}
			// Class operands contribute their stored sentinel, not their original input.
			classValue := uint64(a) + uint64(y.GetValue())
			if multiply {
				classValue = uint64(a) * uint64(y.GetValue())
			}
			classWant := uint32(classValue)
			if uint64(a) > limit || (!multiply && y.IsHuge()) || classValue > limit {
				classWant = ^uint32(0)
			}
			if z.GetValue() != classWant {
				t.Fatal("class arithmetic", a, b, multiply, z.GetValue(), classWant)
			}
		}
	}
	for a := uint32(0); a < 128; a++ {
		for b := uint32(0); b < 128; b++ {
			check(a, b)
		}
	}
	values := []uint32{0, 1, 2, 32767, 32768, 65535, 65536, 1073741822, 1073741823, 1073741824, 2147483647, ^uint32(0)}
	for _, a := range values {
		for _, b := range values {
			check(a, b)
		}
	}
	r := rand.New(rand.NewChaCha8([32]byte{17}))
	for i := 0; i < 10000; i++ {
		check(r.Uint32(), r.Uint32())
		check(r.Uint32()>>2, r.Uint32()>>2)
	}
	x := NewApproxNat(32768)
	x.MulAssign(x)
	if !x.IsHuge() {
		t.Fatal("self multiplication crossing limit")
	}
	x.Assign(1)
	for i := 0; i < 30; i++ {
		x.AddAssign(x)
	}
	if !x.IsHuge() {
		t.Fatal("self addition crossing limit")
	}
}

func TestApproxNatDisplayAndEveryColdReservation(t *testing.T) {
	for _, tc := range []struct {
		value uint32
		text  string
	}{{0, "0"}, {1073741823, "1073741823"}, {1073741824, "[huge]"}, {^uint32(0), "[huge]"}} {
		x := NewApproxNat(tc.value)
		completed := false
		for ordinal := int64(0); ordinal < 64; ordinal++ {
			b := arena.NewBudget(^uint64(0))
			out, e := NewNumericTextStream(NumericTextStreamConfig{Budget: b})
			paramsMust(t, e)
			b.FailAfter(ordinal)
			cause := x.Display(out)
			stats := out.Stats()
			if b.Used() != stats.OwnerBytes+stats.ScratchBytes+stats.OutputOwnerBytes+stats.OutputOverflowBytes || x != NewApproxNat(tc.value) {
				t.Fatal("cold graph/value", ordinal)
			}
			got, e := out.String()
			paramsMust(t, e)
			if !strings.HasPrefix(tc.text, got) {
				t.Fatal("output prefix", got, tc.text)
			}
			if cause != nil && !errors.Is(cause, arena.ErrInjectedFailure) {
				t.Fatal(cause)
			}
			b.FailAfter(-1)
			paramsMust(t, out.Close())
			if b.Used() != 0 {
				t.Fatal("cleanup refund")
			}
			if cause == nil {
				if got != tc.text {
					t.Fatal(got)
				}
				completed = true
				break
			}
		}
		if !completed {
			t.Fatal("missing completion horizon")
		}
	}
	for _, tc := range []struct {
		value uint32
		text  string
	}{{63, "___0X3F"}, {^uint32(0), "_[huge]"}} {
		f, out, b := numericFileFixture(t)
		paramsMust(t, out.SetIntegerBase(16))
		paramsMust(t, out.SetUppercase(true))
		paramsMust(t, out.SetShowBase(true))
		paramsMust(t, out.SetShowPos(true))
		paramsMust(t, out.SetFill('_'))
		paramsMust(t, out.SetWidth(7))
		x := NewApproxNat(tc.value)
		before := b.Used()
		b.FailAfter(0)
		paramsMust(t, x.Display(out))
		if b.Used() != before {
			t.Fatal("file insertion allocates")
		}
		b.FailAfter(-1)
		_, e := f.Seek(0, io.SeekStart)
		paramsMust(t, e)
		got, e := io.ReadAll(f)
		paramsMust(t, e)
		if string(got) != tc.text {
			t.Fatal(string(got), tc.text)
		}
		paramsMust(t, out.Close())
		if _, e = f.Write([]byte("caller")); e != nil {
			t.Fatal("borrowed file closed", e)
		}
		if x.Display(out) == nil || x.Display(nil) != ErrNumericStreamOwner {
			t.Fatal("invalid stream")
		}
	}
}
