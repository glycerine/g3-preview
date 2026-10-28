package numeral

// Source: src/util/mpf.cpp:1828-1848; src/util/mpf.h:202-205.

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestArbitraryFloatPublicExponentBoundsIndependentAndWarm(t *testing.T) {
	m, r, b := mpfNumeralHelpersFixture(t)
	for eb := uint32(2); eb <= 63; eb++ {
		top, e := m.MkTopExp(eb)
		serviceMust(t, e)
		bot, e := m.MkBotExp(eb)
		serviceMust(t, e)
		lo, e := m.MkMinExp(eb)
		serviceMust(t, e)
		hi, e := m.MkMaxExp(eb)
		serviceMust(t, e)
		wantTop := int64(uint64(1) << (eb - 1))
		if top != wantTop || bot != 1-wantTop || lo != 2-wantTop || hi != wantTop-1 {
			t.Fatal("independent bounds", eb, top, bot, lo, hi)
		}
		used, state := b.Used(), r.State()
		b.FailAfter(0)
		for _, get := range []func(uint32) (int64, error){m.MkTopExp, m.MkBotExp, m.MkMinExp, m.MkMaxExp} {
			_, e := get(eb)
			serviceMust(t, e)
		}
		if b.Used() != used || r.State() != state {
			t.Fatal("warm bound allocated", eb)
		}
		b.FailAfter(-1)
	}
	lo, e := m.MkMinExp(1)
	serviceMust(t, e)
	hi, e := m.MkMaxExp(1)
	serviceMust(t, e)
	if lo != 1 || hi != 0 {
		t.Fatal("distinct minimum/maximum precondition", lo, hi)
	}
}

func TestArbitraryFloatPublicExponentBoundsSignedBoundary(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	bot, e := m.MkBotExp(64)
	serviceMust(t, e)
	lo, e := m.MkMinExp(64)
	serviceMust(t, e)
	hi, e := m.MkMaxExp(64)
	serviceMust(t, e)
	if bot != -math.MaxInt64 || lo != 1-math.MaxInt64 || hi != math.MaxInt64 {
		t.Fatal("signed64 bounds", bot, lo, hi)
	}
	if modularSourceDebug {
		bindMPNDebugActions(t, "TTTTT")
	}
	top, e := m.MkTopExp(64)
	if modularSourceDebug {
		if !isMPNVerificationThrow(e) {
			t.Fatal("source signed get_int64 assertion", top, e)
		}
		for _, get := range []func(uint32) (int64, error){m.MkTopExp, m.MkBotExp} {
			if _, e := get(1); !isMPNVerificationThrow(e) {
				t.Fatal("top/bottom source assertion", e)
			}
		}
		for _, get := range []func(uint32) (int64, error){m.MkMinExp, m.MkMaxExp} {
			if _, e := get(0); !isMPNVerificationThrow(e) {
				t.Fatal("minimum/maximum source assertion", e)
			}
		}
	} else {
		serviceMust(t, e)
		if top != math.MinInt64 {
			t.Fatal("release signed narrowing", top)
		}
		x := ArbitraryFloat{format: arbitraryFloatFormat(8, 24, false), exponent: 7, significand: NewInt(9)}
		if e := m.MkMaxValue(64, 24, true, &x, nil); !errors.Is(e, arena.ErrBounds) || x.exponent != 7 || !x.Sign() || x.EBits() != 64 || x.SBits() != 24 || x.significand != NewInt(9) {
			t.Fatal("reached top-minus-one overflow prefix", x, e)
		}
	}
}
