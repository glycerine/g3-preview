// Source: src/util/s_integer.h:25-145; src/util/s_integer.cpp:22-70.
package util

import (
	"errors"
	"math/big"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestSIntegerConstructorsParsingAndTraits(t *testing.T) {
	for _, tc := range []struct {
		text string
		want int32
	}{
		{"", 0}, {" +", 0}, {" \t\n\r\v\f-17tail", -17}, {"42.9", 42}, {"0x20", 0}, {"--2", 0}, {"12\x0099", 12}, {"\xc2\xa012", 0},
		{"2147483647", 2147483647}, {"2147483648", -2147483648}, {"4294967295", -1}, {"4294967296", 0},
		{"9223372036854775807", -1}, {"9223372036854775808", -1}, {"99999999999999999999999999999", -1},
		{"-9223372036854775808", 0}, {"-9223372036854775809", 0}, {"-99999999999999999999999999999", 0}, {"-4294967297", -1},
	} {
		if got := NewSIntegerString(tc.text).GetInt(); got != tc.want {
			t.Fatal(tc.text, got, tc.want)
		}
	}
	for _, v := range []int32{-2147483648, -17, -1, 0, 1, 2, 2147483647} {
		x := NewSInteger(v)
		if x.GetInt64() != int64(v) || x.GetUint64() != uint64(int64(v)) || x.GetUnsigned() != uint32(v) || x.Hash() != uint32(v) {
			t.Fatal("signed promotion/narrowing", v)
		}
		if x.IsBig() || !x.IsInt() || !x.IsSInteger() || !x.IsInt64() || !x.IsUint64() || !x.IsUnsigned() || !x.IsRational() {
			t.Fatal("source constant traits", v)
		}
		if x.GetSInteger() != x || x.GetRational() != x || !x.GetInfinitesimal().IsZero() || x.Numerator() != x || x.Denominator() != SIntegerOne() {
			t.Fatal("projections", v)
		}
		if NewSIntegerInt64(int64(v)+(1<<32)) != x || NewSIntegerUint64(uint64(int64(v))) != x {
			t.Fatal("constructor narrowing", v)
		}
		if x.IsZero() != (v == 0) || x.IsOne() != (v == 1) || x.IsMinusOne() != (v == -1) || x.IsNeg() != (v < 0) || x.IsPos() != (v > 0) || x.IsNonneg() != (v >= 0) || x.IsNonpos() != (v <= 0) || x.IsEven() != (v%2 == 0) {
			t.Fatal("predicates", v)
		}
		if x.Floor() != x || x.Ceil() != x {
			t.Fatal("floor/ceil")
		}
		if (SIntegerHashProc{}).Hash(x) != x.Hash() || !(SIntegerEqProc{}).Equal(x, x) {
			t.Fatal("functors")
		}
		copy := x
		copy.Reset()
		if !copy.IsZero() || x.GetInt() != v {
			t.Fatal("value-copy/reset")
		}
	}
	var zero SInteger
	if zero != SIntegerZero() || !SIntegerOne().IsOne() || !SIntegerMinusOne().IsMinusOne() {
		t.Fatal("static values")
	}
}

func TestSIntegerIndependentFiniteArithmeticAndNumberTheory(t *testing.T) {
	for a := int32(-50); a <= 50; a++ {
		for b := int32(-50); b <= 50; b++ {
			x, y := NewSInteger(a), NewSInteger(b)
			if x.Add(y).GetInt() != a+b || x.Sub(y).GetInt() != a-b || x.Mul(y).GetInt() != a*b {
				t.Fatal(a, b)
			}
			z := x
			if z.AddAssign(y) != &z || z != x.Add(y) {
				t.Fatal("add assignment")
			}
			z = x
			if z.SubAssign(y) != &z || z != x.Sub(y) {
				t.Fatal("sub assignment")
			}
			z = x
			if z.MulAssign(y) != &z || z != x.Mul(y) {
				t.Fatal("mul assignment")
			}
			if x.Equal(y) != (a == b) || x.NotEqual(y) != (a != b) || x.Less(y) != (a < b) || x.Greater(y) != (a > b) || x.LessEqual(y) != (a <= b) || x.GreaterEqual(y) != (a >= b) {
				t.Fatal("ordering", a, b)
			}
			z = x
			z.Addmul(y, x)
			if z.GetInt() != a+b*a {
				t.Fatal("addmul")
			}
			z = x
			z.Submul(y, x)
			if z.GetInt() != a-b*a {
				t.Fatal("submul")
			}
			z = x
			z.Swap(&y)
			if z.GetInt() != b || y.GetInt() != a {
				t.Fatal("swap")
			}
			y = NewSInteger(b)
			if b != 0 {
				if x.Div(y).GetInt() != a/b || x.Rem(y).GetInt() != a%b {
					t.Fatal("truncated division", a, b)
				}
				want := a % b
				if want < 0 {
					want += b
				}
				if x.Mod(y).GetInt() != want {
					t.Fatal("original mod", a, b)
				}
				z = x
				if z.DivAssign(y) != &z || z != x.Div(y) {
					t.Fatal("div assignment")
				}
				z = x
				if z.RemAssign(y) != &z || z != x.Rem(y) {
					t.Fatal("rem assignment")
				}
			}
			if a != 0 && b != 0 {
				g := new(big.Int).GCD(nil, nil, big.NewInt(int64(a)), big.NewInt(int64(b))).Int64()
				if SIntegerGCD(x, y).GetInt64() != g || SIntegerLCM(x, y).GetInt64() != (int64(a)/g)*int64(b) {
					t.Fatal("gcd/lcm", a, b, g)
				}
			}
		}
	}
	for a := int32(-3); a <= 3; a++ {
		for n := uint32(0); n <= 7; n++ {
			x := NewSInteger(a)
			got, e := SIntegerPower(x, n)
			if e != nil {
				t.Fatal(e)
			}
			want := new(big.Int).Exp(big.NewInt(int64(a)), big.NewInt(int64(n)), nil).Int64()
			if got.GetInt64() != want || x.Expt(int32(n)).GetInt64() != want {
				t.Fatal(a, n, got, want)
			}
		}
	}
	x := NewSInteger(-7)
	if x.Abs().GetInt() != 7 || x.Negated().GetInt() != 7 || x.Expt(-1) != SIntegerOne() {
		t.Fatal("unary/expt")
	}
	old := x.PostInc()
	if old.GetInt() != -7 || x.GetInt() != -6 || x.Inc() != &x || x.GetInt() != -5 {
		t.Fatal("increment")
	}
	old = x.PostDec()
	if old.GetInt() != -5 || x.GetInt() != -6 || x.Dec() != &x || x.GetInt() != -7 {
		t.Fatal("decrement")
	}
	if x.Assign(SIntegerOne()) != &x || !x.IsOne() {
		t.Fatal("assignment")
	}
	for _, p := range []uint32{1 << 31, ^uint32(0)} {
		if _, e := SIntegerPower(SIntegerOne(), p); e != ErrSIntegerNonterminating {
			t.Fatal("source wrapping-mask loop", e)
		}
	}
	for _, a := range []int32{0, 7} {
		func() {
			defer func() {
				if recover() == nil {
					t.Fatal("source gcd zero divisor domain")
				}
			}()
			SIntegerGCD(NewSInteger(a), SIntegerZero())
		}()
	}
}

func TestSIntegerRationalConversionsAndStream(t *testing.T) {
	s, b := serviceNew(t)
	for _, n := range []int32{-2147483648, -17, 0, 1, 2147483647} {
		x := NewSInteger(n)
		before := b.Used()
		b.FailAfter(0)
		v, e := x.ToRational(s)
		serviceMust(t, e)
		got, e := SIntegerFromRational(s, &v)
		serviceMust(t, e)
		if got != x {
			t.Fatal(n, got)
		}
		if b.Used() != before || !v.Raw.Num.IsSmall() || !v.Raw.Den.IsSmall() || v.Raw.Den.Val != 1 {
			t.Fatal("signed32 constructor introduced allocation or lost inline shape")
		}
		b.FailAfter(-1)
		serviceMust(t, s.Del(&v))
	}
	large := serviceValue(t, s, "4294967297")
	got, e := SIntegerFromRational(s, &large)
	serviceMust(t, e)
	if got.GetInt() != 1 {
		t.Fatal("rational int64 to int32 narrowing", got)
	}
	serviceMust(t, s.Del(&large))
	frac := serviceValue(t, s, "1/2")
	got, e = SIntegerFromRational(s, &frac)
	if modularSourceDebug {
		if e != ErrIntegerAssertion {
			t.Fatal("source debug nonintegral guard", e)
		}
	} else if e != nil || got.GetInt() != 1 {
		t.Fatal("source release erases rational assertion and reads NUM", got, e)
	}
	serviceMust(t, s.Del(&frac))
	out, e := NewNumericTextStream(NumericTextStreamConfig{Budget: b})
	serviceMust(t, e)
	serviceMust(t, out.SetShowPos(true))
	serviceMust(t, out.SetWidth(5))
	serviceMust(t, out.SetFill('_'))
	serviceMust(t, NewSInteger(-17).Display(out))
	serviceMust(t, out.AppendByte(' '))
	serviceMust(t, NewSInteger(3).Display(out))
	text, e := out.String()
	serviceMust(t, e)
	if text != "__-17 +3" {
		t.Fatal(text)
	}
	serviceMust(t, out.Close())
	if NewSInteger(1).Display(nil) != ErrNumericStreamOwner {
		t.Fatal("invalid stream")
	}
	if NewSIntegerString(" -17xxx").ToString() != "-17" {
		t.Fatal("Go convenience string")
	}
	if allocs := testing.AllocsPerRun(1000, func() {
		if NewSIntegerString("  -9223372036854775809end").GetInt() != 0 {
			panic("parse")
		}
	}); allocs != 0 {
		t.Fatal("parsing allocated", allocs)
	}
}

func TestSIntegerIndependentLP64Parsing(t *testing.T) {
	r := rand.New(rand.NewChaCha8([32]byte{29}))
	max := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 63), big.NewInt(1))
	min := new(big.Int).Neg(new(big.Int).Lsh(big.NewInt(1), 63))
	for i := 0; i < 2000; i++ {
		var digits strings.Builder
		if r.Uint32()&1 != 0 {
			digits.WriteByte('-')
		} else {
			digits.WriteByte('+')
		}
		length := int(r.Uint32()%80) + 1
		for j := 0; j < length; j++ {
			digits.WriteByte(byte('0' + r.Uint32()%10))
		}
		text := digits.String()
		wide, ok := new(big.Int).SetString(text, 10)
		if !ok {
			t.Fatal(text)
		}
		if wide.Cmp(max) > 0 {
			wide.Set(max)
		}
		if wide.Cmp(min) < 0 {
			wide.Set(min)
		}
		want := int32(wide.Int64())
		if got := NewSIntegerString(" \t" + text + "tail").GetInt(); got != want {
			t.Fatal(text, got, want)
		}
	}
}

func TestSIntegerRepresentableBoundaryArithmetic(t *testing.T) {
	values := []int32{-2147483648, -2147483647, -1073741824, -65536, -2, -1, 0, 1, 2, 65536, 1073741823, 2147483646, 2147483647}
	fits := func(n int64) bool { return n >= -2147483648 && n <= 2147483647 }
	for _, a := range values {
		for _, b := range values {
			x, y := NewSInteger(a), NewSInteger(b)
			if want := int64(a) + int64(b); fits(want) && x.Add(y).GetInt64() != want {
				t.Fatal("boundary addition", a, b)
			}
			if want := int64(a) - int64(b); fits(want) && x.Sub(y).GetInt64() != want {
				t.Fatal("boundary subtraction", a, b)
			}
			if want := int64(a) * int64(b); fits(want) && x.Mul(y).GetInt64() != want {
				t.Fatal("boundary multiplication", a, b)
			}
			if b != 0 && !(a == -2147483648 && b == -1) {
				if x.Div(y).GetInt64() != int64(a)/int64(b) || x.Rem(y).GetInt64() != int64(a)%int64(b) {
					t.Fatal("boundary quotient/remainder", a, b)
				}
			}
			if a != 0 && b != 0 && a != -2147483648 && b != -2147483648 {
				g := new(big.Int).GCD(nil, nil, big.NewInt(int64(a)), big.NewInt(int64(b))).Int64()
				if SIntegerGCD(x, y).GetInt64() != g {
					t.Fatal("boundary gcd", a, b)
				}
				if want := (int64(a) / g) * int64(b); fits(want) && SIntegerLCM(x, y).GetInt64() != want {
					t.Fatal("boundary signed lcm", a, b)
				}
			}
		}
	}
	if got, e := SIntegerPower(SIntegerOne(), 1<<30); e != nil || got != SIntegerOne() {
		t.Fatal("highest terminating mask domain", got, e)
	}
}

func TestSIntegerDisplayEveryColdReservation(t *testing.T) {
	for _, n := range []int32{-2147483648, -1, 0, 2147483647} {
		value := NewSInteger(n)
		completed := false
		for ordinal := int64(0); ordinal < 64; ordinal++ {
			b := arena.NewBudget(^uint64(0))
			out, e := NewNumericTextStream(NumericTextStreamConfig{Budget: b})
			memoryRuntimeRequire(t, e)
			b.FailAfter(ordinal)
			cause := value.Display(out)
			stats := out.Stats()
			if b.Used() != stats.OwnerBytes+stats.ScratchBytes+stats.OutputOwnerBytes+stats.OutputOverflowBytes || value.GetInt() != n {
				t.Fatal("immediate display graph/value")
			}
			text, e := out.String()
			memoryRuntimeRequire(t, e)
			want := big.NewInt(int64(n)).String()
			if !strings.HasPrefix(want, text) || cause != nil && !errors.Is(cause, arena.ErrInjectedFailure) {
				t.Fatal(text, cause)
			}
			b.FailAfter(-1)
			memoryRuntimeRequire(t, out.Close())
			if b.Used() != 0 {
				t.Fatal("display refund")
			}
			if cause == nil {
				if text != want {
					t.Fatal(text)
				}
				completed = true
				break
			}
		}
		if !completed {
			t.Fatal("display completion horizon")
		}
	}
}
