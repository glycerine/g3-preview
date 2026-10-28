package numeral

// Source: src/util/mpf.cpp:1584-1588; src/util/mpf.h:143.

import (
	"errors"
	utext "github.com/glycerine/g3/pkg/util/text"
	"math/big"
	"math/rand/v2"
	"testing"
)

func TestArbitraryFloatSMT2StreamIndependentDyadic(t *testing.T) {
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
				var out utext.OwnedStringStream
				serviceMust(t, out.Construct(m.rational.BudgetField))
				decimal := sample%2 == 0
				serviceMust(t, m.DisplaySMT2Into(&out, &x, decimal))
				got, e := (utext.OwnedStringView{Owner: &out.Buffer.String, Generation: out.Buffer.String.Generation, Length: out.Buffer.Position}).String()
				serviceMust(t, e)
				numeral := func(v *big.Int) string {
					text := new(big.Int).Abs(v).String()
					if decimal {
						text += ".0"
					}
					if v.Sign() < 0 {
						text = "(- " + text + ")"
					}
					return text
				}
				expected := numeral(want.Num())
				if want.Denom().Cmp(big.NewInt(1)) != 0 {
					expected = "(/ " + expected + " " + numeral(want.Denom()) + ")"
				}
				if got != expected || x != before {
					t.Fatal("independent rational string", sb, sample, negative, got, want)
				}
				out.Buffer.OwnAbort()
				serviceMust(t, m.Del(&x))
			}
		}
	}
}

func TestArbitraryFloatSMT2StreamWarmAndLaterChildren(t *testing.T) {
	m, r, b := mpfNumeralHelpersFixture(t)
	var x ArbitraryFloat
	serviceMust(t, m.SetDouble(&x, 8, 24, 1.5))
	var first utext.OwnedStringStream
	serviceMust(t, first.Construct(b))
	serviceMust(t, m.DisplaySMT2Into(&first, &x, false))
	first.Buffer.OwnAbort()
	used, state := b.Used(), r.State()
	var out utext.OwnedStringStream
	serviceMust(t, out.Construct(b))
	b.FailAfter(0)
	serviceMust(t, m.DisplaySMT2Into(&out, &x, false))
	got, e := (utext.OwnedStringView{Owner: &out.Buffer.String, Generation: out.Buffer.String.Generation, Length: out.Buffer.Position}).String()
	serviceMust(t, e)
	if got != "(/ 3 2)" || b.Used() != used || r.State() != state {
		t.Fatal("warm wrapper allocated", got, b.Used(), used)
	}
	b.FailAfter(-1)
	q := NewRational()
	serviceMust(t, m.ToRational(&x, m.rational, &q))
	var later utext.OwnedStringStream
	serviceMust(t, later.Construct(b))
	serviceMust(t, m.rational.DisplaySMT2Into(&later, &q, false))
	conversion, format := m.rationalConversion.invocation, m.rational.smt2Stream.invocation
	serviceMust(t, m.CleanupSMT2Stream())
	if m.rationalConversion.invocation != conversion || m.rational.smt2Stream.invocation != format {
		t.Fatal("ended wrapper touched later children")
	}
	out.Buffer.OwnAbort()
	later.Buffer.OwnAbort()
	serviceMust(t, m.rational.Del(&q))
	serviceMust(t, m.CleanupRationalConversion())
	serviceMust(t, m.Del(&x))
}

func TestArbitraryFloatSMT2StreamEveryInsertionException(t *testing.T) {
	prefixes := []string{"", "(/ ", "(/ (- ", "(/ (- 7", "(/ (- 7.0", "(/ (- 7.0)", "(/ (- 7.0) ", "(/ (- 7.0) 4", "(/ (- 7.0) 4.0"}
	for fail := 1; fail <= len(prefixes); fail++ {
		m, _, b := mpfNumeralHelpersFixture(t)
		var x ArbitraryFloat
		serviceMust(t, m.SetDouble(&x, 8, 24, -1.75))
		var out utext.OwnedStringStream
		serviceMust(t, out.Construct(b))
		boom := errors.New("actual borrowed stream exception")
		stream := smt2FailingStream{inner: &out, fail: fail, failure: boom}
		body := m.DisplaySMT2Into(&stream, &x, true)
		if !errors.Is(body, boom) {
			t.Fatal("selected wrapper insertion", fail, body)
		}
		s := &m.smt2Stream
		if s.qLive || s.qCursor != 2 || s.sourceCause != nil {
			t.Fatal("successful child source suffix failed to release scoped q", fail)
		}
		got, e := (utext.OwnedStringView{Owner: &out.Buffer.String, Generation: out.Buffer.String.Generation, Length: out.Buffer.Position}).String()
		serviceMust(t, e)
		if got != prefixes[fail-1] {
			t.Fatal("literal wrapper prefix", fail, got)
		}
		serviceMust(t, m.CleanupSMT2Stream())
		after, e := (utext.OwnedStringView{Owner: &out.Buffer.String, Generation: out.Buffer.String.Generation, Length: out.Buffer.Position}).String()
		serviceMust(t, e)
		if after != got || out.Buffer.String.Closed {
			t.Fatal("owning join changed borrowed output", fail)
		}
		out.Buffer.OwnAbort()
		serviceMust(t, m.Del(&x))
	}
}
