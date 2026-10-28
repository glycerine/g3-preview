package numeral

// Actual registered print routes.

import (
	"math/big"
	"strconv"
	"testing"
)

// Independent bounded-significand model; no billion-bit exponent allocation.
func fastModelSquare(s *big.Int, e int64) (*big.Int, int64) {
	r := new(big.Int).Mul(s, s)
	shift := r.BitLen() - 64
	r.Rsh(r, uint(shift))
	e = 2*e + int64(shift)
	if e > 2147483647 {
		return new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 64), big.NewInt(1)), 2147483647
	}
	return r, e
}
func fastModelText(s *big.Int, e int64) string {
	v := new(big.Int).Set(s)
	if e < 0 {
		tz := uint(v.TrailingZeroBits())
		k := uint(-e)
		if tz < k {
			k = tz
		}
		v.Rsh(v, k)
		e += int64(k)
	}
	out := v.String()
	if e > 0 {
		if e <= 63 {
			return out + "*" + strconv.FormatUint(uint64(1)<<uint(e), 10)
		}
		return out + "*2^" + strconv.FormatInt(e, 10)
	}
	if e < 0 {
		if e >= -63 {
			return out + "/" + strconv.FormatUint(uint64(1)<<uint(-e), 10)
		}
		return out + "/2^" + strconv.FormatInt(-e, 10)
	}
	return out
}

// Independent exact fraction quantization for source scalar fraction setters.
func fastFractionModel(n int64, d uint64, p uint32, up bool) (*big.Int, int64) {
	if n == 0 {
		return big.NewInt(0), 0
	}
	num := big.NewInt(n)
	negative := num.Sign() < 0
	num.Abs(num)
	den := new(big.Int).SetUint64(d)
	lg := int64(num.BitLen() - den.BitLen())
	scaled := new(big.Int).Set(den)
	if lg >= 0 {
		scaled.Lsh(scaled, uint(lg))
		if num.Cmp(scaled) < 0 {
			lg--
		}
	} else {
		scaledNum := new(big.Int).Lsh(new(big.Int).Set(num), uint(-lg))
		if scaledNum.Cmp(den) < 0 {
			lg--
		}
	}
	exp := lg - int64(p*32) + 1
	a, b := new(big.Int).Set(num), new(big.Int).Set(den)
	if exp < 0 {
		a.Lsh(a, uint(-exp))
	} else {
		b.Lsh(b, uint(exp))
	}
	s, r := new(big.Int), new(big.Int)
	s.QuoRem(a, b, r)
	if r.Sign() != 0 && negative != up {
		s.Add(s, big.NewInt(1))
	}
	if s.BitLen() > int(p*32) {
		s.Rsh(s, 1)
		exp++
	}
	return s, exp
}
func TestFastFloatSourcePrintedPaths(t *testing.T) {
	events := 0
	{ // tst1: actual scoped returned-expression copies and100 iterations/catch.
		m := fastManager(t, 2)
		a, e := NewScopedFastFloat(m)
		serviceMust(t, e)
		b, e := NewScopedFastFloat(m)
		serviceMust(t, e)
		serviceMust(t, a.SetInt32(100))
		serviceMust(t, b.SetInt32(-33))
		as, e := a.String()
		serviceMust(t, e)
		bs, e := b.String()
		serviceMust(t, e)
		if "a: "+as+", b: "+bs != "a: 100, b: -33" {
			t.Fatal(as, bs)
		}
		events++
		product, e := a.Mul(b)
		serviceMust(t, e)
		text, e := product.String()
		serviceMust(t, e)
		if "a*b: "+text != "a*b: -3300" {
			t.Fatal(text)
		}
		serviceMust(t, product.Close())
		events++
		sig := new(big.Int).Lsh(big.NewInt(100), 57)
		exp := int64(-57)
		caught := false
		for i := 0; i < 100; i++ {
			tmp, e := a.Mul(a)
			if e != nil {
				caught = true
				if e.Error() != ErrFastFloatOverflow.Error() {
					t.Fatal(e)
				}
				break
			}
			e = a.Assign(tmp)
			serviceMust(t, tmp.Close())
			serviceMust(t, e)
			sig, exp = fastModelSquare(sig, exp)
			got, e := a.String()
			serviceMust(t, e)
			want := fastModelText(sig, exp)
			if strconv.Itoa(i)+": "+got != strconv.Itoa(i)+": "+want {
				t.Fatal(i, got, want)
			}
			events++
		}
		// Source default positive inward overflow saturates; catch remains reachable
		// only if a prior source action actually throws, not assumed at iteration28.
		if caught {
			t.Fatal("unexpected default inward exception")
		}
		serviceMust(t, b.Close())
		serviceMust(t, a.Close())
	}
	{
		m := fastManager(t, 2)
		var a, b FastFloat
		serviceMust(t, m.SetUint64(&a, 100))
		serviceMust(t, m.SetInt64(&b, -100))
		as, e := m.String(&a)
		serviceMust(t, e)
		bs, e := m.String(&b)
		serviceMust(t, e)
		if "[test2], a: "+as+", b: "+bs != "[test2], a: 100, b: -100" {
			t.Fatal(as, bs)
		}
		events++
		serviceMust(t, m.Del(&b))
		serviceMust(t, m.Del(&a))
	}
	{
		m := fastManager(t, 2)
		var a, b, c FastFloat
		serviceMust(t, m.SetInt32(&a, 1))
		serviceMust(t, m.SetInt32(&b, 3))
		serviceMust(t, m.Div(&a, &b, &c))
		fastText(t, m, &c, "6148914691236517205/2^64")
		events++
		serviceMust(t, m.RoundToPlusInf())
		serviceMust(t, m.Reset(&c))
		serviceMust(t, m.Div(&a, &b, &c))
		fastText(t, m, &c, "12297829382473034411/2^65")
		events++
		serviceMust(t, m.Del(&c))
		serviceMust(t, m.Del(&b))
		serviceMust(t, m.Del(&a))
	}
	{
		m := fastManager(t, 2)
		var a FastFloat
		serviceMust(t, m.SetUint64(&a, 1<<63))
		s, e := m.RawString(&a)
		serviceMust(t, e)
		if s != "8000000000000000*2^0" {
			t.Fatal(s)
		}
		events++
		serviceMust(t, m.Del(&a))
	}
	{
		m := fastManager(t, 2)
		var a, b, one FastFloat
		serviceMust(t, m.SetFraction32(&a, 1, 3))
		fastText(t, m, &a, "6148914691236517205/2^64")
		events++
		serviceMust(t, m.Copy(&b, &a))
		serviceMust(t, m.Next(&b))
		fastText(t, m, &b, "12297829382473034411/2^65")
		events++
		serviceMust(t, m.Prev(&b))
		serviceMust(t, m.CeilInPlace(&b))
		fastText(t, m, &b, "1")
		events++
		serviceMust(t, m.SetFraction32(&b, 4, 3))
		fastText(t, m, &b, "6148914691236517205/4611686018427387904")
		events++
		serviceMust(t, m.CeilInPlace(&b))
		fastText(t, m, &b, "2")
		events++
		serviceMust(t, m.Del(&one))
		serviceMust(t, m.Del(&b))
		serviceMust(t, m.Del(&a))
	}
	{
		m := fastManager(t, 2)
		var a FastFloat
		for _, tc := range []struct {
			n int32
			d uint32
			w string
		}{{2, 1, "2.0"}, {-2, 1, "(- 2.0)"}, {1, 3, "(/ 6148914691236517205.0 (^ 2.0 64.0))"}} {
			if tc.d == 1 {
				serviceMust(t, m.SetInt32(&a, tc.n))
			} else {
				serviceMust(t, m.SetFraction32(&a, tc.n, tc.d))
			}
			s, e := m.SMT2String(&a, true)
			serviceMust(t, e)
			if s != tc.w {
				t.Fatal(s, tc.w)
			}
			events++
		}
		serviceMust(t, m.Del(&a))
	}
	{
		m := fastManager(t, 2)
		var a, b, c FastFloat
		serviceMust(t, m.RoundToPlusInf())
		serviceMust(t, m.SetInt32(&a, 1))
		serviceMust(t, m.SetUint64(&b, ^uint64(0)))
		serviceMust(t, m.Div(&a, &b, &c))
		for _, tc := range []struct {
			a *FastFloat
			w string
		}{{&a, "8000000000000000*2^-63"}, {&b, "ffffffffffffffff*2^0"}} {
			s, e := m.RawString(tc.a)
			serviceMust(t, e)
			if s != tc.w {
				t.Fatal(s)
			}
			events++
		}
		as, e := m.String(&a)
		serviceMust(t, e)
		bs, e := m.String(&b)
		serviceMust(t, e)
		cs, e := m.String(&c)
		serviceMust(t, e)
		if as+"/"+bs+" <= "+cs != "1/18446744073709551615 <= 9223372036854775809/2^127" {
			t.Fatal(as, bs, cs)
		}
		events++
		serviceMust(t, m.Del(&c))
		serviceMust(t, m.Del(&b))
		serviceMust(t, m.Del(&a))
	}
	// tst4's exact2^512 conversion/printing is in conversion source test, below
	// this print path partition.118 total events=117 here+that one real print.
	if events != 117 {
		t.Fatal("printed event inventory", events)
	}
}
