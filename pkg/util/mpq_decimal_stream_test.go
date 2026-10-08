// Source: src/util/mpq.cpp:156-185; active display calls in src/test/mpq.cpp:111-114.
package util

import (
	"math/big"
	"math/rand/v2"
	"testing"
)

func TestRationalDecimalStreamSourceDriver(t *testing.T) {
	m, _, b := mpfNumeralHelpersFixture(t)
	var out OwnedStringStream
	serviceMust(t, out.Construct(b))
	for _, tc := range []struct {
		n, d   int32
		prefix string
	}{{1, 3, "1/3: "}, {1, 4, "1/4: "}} {
		a := Rational{Num: NewInt(tc.n), Den: NewInt(tc.d)}
		serviceMust(t, out.AppendCString(tc.prefix))
		serviceMust(t, m.rational.DisplayDecimalInto(&out, &a, 10, false))
		serviceMust(t, out.AppendCString("\n"))
	}
	got, e := (OwnedStringView{&out.Buffer.String, out.Buffer.String.Generation, out.Buffer.Position}).String()
	serviceMust(t, e)
	if got != "1/3: 0.3333333333?\n1/4: 0.25\n" {
		t.Fatal("source driver decimal", got)
	}
	out.Buffer.OwnAbort()
}

func TestRationalDecimalStreamIndependentFinite(t *testing.T) {
	m, _, b := mpfNumeralHelpersFixture(t)
	for n := int64(-19); n <= 19; n++ {
		for d := int64(1); d <= 19; d++ {
			a := Rational{Num: NewInt(int32(n)), Den: NewInt(int32(d))}
			before := a
			for _, prec := range []uint32{0, 1, 2, 3, 10} {
				for _, truncate := range []bool{false, true} {
					var out OwnedStringStream
					serviceMust(t, out.Construct(b))
					serviceMust(t, out.AppendCString("prefix:"))
					serviceMust(t, m.rational.DisplayDecimalInto(&out, &a, prec, truncate))
					got, e := (OwnedStringView{&out.Buffer.String, out.Buffer.String.Generation, out.Buffer.Position}).String()
					serviceMust(t, e)
					want := "prefix:" + decimalOracle(new(big.Rat).SetFrac(big.NewInt(n), big.NewInt(d)), prec, truncate)
					s := &m.rational.decimalStream
					if got != want || a != before || s.count != 4 || s.sourceCursor != 4 || !s.bodySucceeded || s.active {
						t.Fatal("independent decimal stream", n, d, prec, truncate, got, want)
					}
					out.Buffer.OwnAbort()
				}
			}
		}
	}
	rng := rand.New(rand.NewChaCha8([32]byte{15, 6, 18, 5}))
	for i := 0; i < 80; i++ {
		n := new(big.Int).SetUint64(rng.Uint64())
		n.Lsh(n, uint(rng.IntN(160)))
		d := new(big.Int).SetUint64(rng.Uint64())
		d.Lsh(d, uint(rng.IntN(160)))
		d.Add(d, big.NewInt(1))
		if i%2 == 0 {
			n.Neg(n)
		}
		a := NewRational()
		serviceMust(t, m.integer.SetString(&a.Num, n.String()))
		serviceMust(t, m.integer.SetString(&a.Den, d.String()))
		var out OwnedStringStream
		serviceMust(t, out.Construct(b))
		serviceMust(t, m.rational.DisplayDecimalInto(&out, &a, 12, false))
		got, e := (OwnedStringView{&out.Buffer.String, out.Buffer.String.Generation, out.Buffer.Position}).String()
		serviceMust(t, e)
		want := decimalOracle(new(big.Rat).SetFrac(n, d), 12, false)
		if got != want {
			t.Fatal("independent BIG decimal", i, got, want)
		}
		out.Buffer.OwnAbort()
		serviceMust(t, m.rational.Del(&a))
	}
}

func TestRationalDecimalStreamLiteralRawNegativeDenominator(t *testing.T) {
	m, _, b := mpfNumeralHelpersFixture(t)
	for _, tc := range []struct {
		n, d int32
		want string
	}{{7, -3, "-2.-3-3?"}, {-7, -3, "--2.-3-3?"}, {9, -3, "-3"}, {-9, -3, "--3"}} {
		a := Rational{Num: NewInt(tc.n), Den: NewInt(tc.d)}
		var out OwnedStringStream
		serviceMust(t, out.Construct(b))
		serviceMust(t, m.rational.DisplayDecimalInto(&out, &a, 2, false))
		got, e := (OwnedStringView{&out.Buffer.String, out.Buffer.String.Generation, out.Buffer.Position}).String()
		serviceMust(t, e)
		if got != tc.want {
			t.Fatal("literal unnormalized source fields", tc, got)
		}
		out.Buffer.OwnAbort()
	}
}
