// Source: src/util/mpf.cpp:1578-1582; src/util/mpf.h:142.
package util

import (
	"errors"
	"math/big"
	"math/rand/v2"
	"testing"
)

func TestArbitraryFloatDecimalStreamIndependentDyadic(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	rng := rand.New(rand.NewChaCha8([32]byte{15, 72, 15, 76}))
	for _, sb := range []uint32{2, 24, 53, 120} {
		for sample := 0; sample < 20; sample++ {
			fraction := new(big.Int).SetUint64(rng.Uint64())
			fraction.Lsh(fraction, uint(rng.IntN(80)))
			fraction.Mod(fraction, new(big.Int).Lsh(big.NewInt(1), uint(sb-1)))
			exp := int64(rng.IntN(41)) - 20
			full := new(big.Int).Add(fraction, new(big.Int).Lsh(big.NewInt(1), uint(sb-1)))
			if sample == 0 {
				exp = -127
				full.Set(fraction)
			}
			for _, negative := range []bool{false, true} {
				x := ArbitraryFloat{format: arbitraryFloatFormat(8, sb, negative), exponent: exp}
				serviceMust(t, m.integer.SetString(&x.significand, fraction.String()))
				before := x
				effective := exp
				if exp == -127 {
					effective++
				}
				want := mpfAddSubDyadic(full, effective-int64(sb)+1)
				if negative {
					want.Neg(want)
				}
				var out OwnedStringStream
				serviceMust(t, out.Construct(m.rational.BudgetField))
				precision := []uint32{0, 1, 5, 12, 120}[sample%5]
				serviceMust(t, m.DisplayDecimalInto(&out, &x, precision))
				got, e := (OwnedStringView{&out.Buffer.String, out.Buffer.String.Generation, out.Buffer.Position}).String()
				serviceMust(t, e)
				expected := decimalOracle(want, precision, false)
				if got != expected || x != before {
					t.Fatal("independent rational string", sb, sample, negative, got, want)
				}
				out.Buffer.OwnAbort()
				serviceMust(t, m.Del(&x))
			}
		}
	}
}

func TestArbitraryFloatDecimalStreamWarmAndLaterChildren(t *testing.T) {
	m, r, b := mpfNumeralHelpersFixture(t)
	var x ArbitraryFloat
	serviceMust(t, m.SetDouble(&x, 8, 24, 1.5))
	var first OwnedStringStream
	serviceMust(t, first.Construct(b))
	serviceMust(t, m.DisplayDecimalInto(&first, &x, 10))
	first.Buffer.OwnAbort()
	used, state := b.Used(), r.State()
	var out OwnedStringStream
	serviceMust(t, out.Construct(b))
	b.FailAfter(0)
	serviceMust(t, m.DisplayDecimalInto(&out, &x, 10))
	got, e := (OwnedStringView{&out.Buffer.String, out.Buffer.String.Generation, out.Buffer.Position}).String()
	serviceMust(t, e)
	if got != "1.5" || b.Used() != used || r.State() != state {
		t.Fatal("warm decimal wrapper allocated", got, b.Used(), used)
	}
	b.FailAfter(-1)
	q := NewRational()
	serviceMust(t, m.ToRational(&x, m.rational, &q))
	var later OwnedStringStream
	serviceMust(t, later.Construct(b))
	serviceMust(t, m.rational.DisplayDecimalInto(&later, &q, 10, false))
	conversion, format := m.rationalConversion.invocation, m.rational.decimalStream.invocation
	serviceMust(t, m.CleanupDecimalStream())
	if m.rationalConversion.invocation != conversion || m.rational.decimalStream.invocation != format {
		t.Fatal("ended wrapper touched later children")
	}
	out.Buffer.OwnAbort()
	later.Buffer.OwnAbort()
	serviceMust(t, m.rational.Del(&q))
	serviceMust(t, m.CleanupRationalConversion())
	serviceMust(t, m.Del(&x))
}

func TestArbitraryFloatDecimalStreamEveryInsertionException(t *testing.T) {
	prefixes := []string{"", "-", "-1", "-1.", "-1.7"}
	for fail := 1; fail <= len(prefixes); fail++ {
		m, _, b := mpfNumeralHelpersFixture(t)
		var x ArbitraryFloat
		serviceMust(t, m.SetDouble(&x, 8, 24, -1.75))
		var out OwnedStringStream
		serviceMust(t, out.Construct(b))
		boom := errors.New("actual decimal wrapper ostream exception")
		stream := smt2FailingStream{inner: &out, fail: fail, failure: boom}
		body := m.DisplayDecimalInto(&stream, &x, 1)
		if !errors.Is(body, boom) {
			t.Fatal("selected wrapper insertion missing", fail, body)
		}
		s, child := &m.decimalStream, &m.rational.decimalStream
		if s.qLive || s.qCursor != 2 || s.sourceCause != nil || child.sourceCursor != 0 || child.bodySucceeded {
			t.Fatal("failed RAW body/scoped q distinction", fail, s.qCursor, child.sourceCursor)
		}
		got, e := (OwnedStringView{&out.Buffer.String, out.Buffer.String.Generation, out.Buffer.Position}).String()
		serviceMust(t, e)
		if got != prefixes[fail-1] {
			t.Fatal("literal decimal wrapper prefix", fail, got)
		}
		serviceMust(t, m.CleanupDecimalStream())
		after, e := (OwnedStringView{&out.Buffer.String, out.Buffer.String.Generation, out.Buffer.Position}).String()
		serviceMust(t, e)
		if after != got || child.sourceCursor != 0 || child.ownCursor != 4 {
			t.Fatal("owning join replayed RAW source suffix or changed output", fail)
		}
		out.Buffer.OwnAbort()
		serviceMust(t, m.Del(&x))
	}
}
