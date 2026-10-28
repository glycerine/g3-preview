package numeral

// Source: src/util/mpf.cpp:1383-1403,1420-1432.

import (
	"errors"
	"testing"
)

func TestArbitraryFloatPartialRemainderLiteralAlignmentAndRoundConstant(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	// The source helper accepts the supplied exp_diff without comparing it to
	// the stored exponents. This literal profile reaches its large alignment
	// branch, including sticky swap, odd increment and hardcoded mpz(32).
	for _, tc := range []struct {
		sb   uint32
		want int64
	}{{4, 15}, {7, 128}, {24, 1 << 24}} {
		hidden := int32(1 << (tc.sb - 1))
		x := ArbitraryFloat{format: arbitraryFloatFormat(11, tc.sb, false), exponent: 40, significand: NewInt(hidden)}
		y := ArbitraryFloat{format: arbitraryFloatFormat(11, tc.sb, true), exponent: 0, significand: NewInt(hidden + hidden/2)}
		serviceMust(t, m.PartialRemainder(&x, &y, 0, false))
		sig, e := m.integer.Int64(&x.significand)
		serviceMust(t, e)
		if sig != tc.want || x.exponent != 39 || x.Sign() {
			t.Fatal("literal alignment/sticky/32 rounding", tc, sig, x)
		}
		for index, live := range m.partialRemainder.live {
			if live {
				t.Fatal("actual completed NUM declaration still live", index)
			}
		}
		copied := m.partialRemainder
		copied.hooks.Policy = &copied
		before := x
		if e := copied.run(&x, &y, 0, false); !errors.Is(e, ErrNumericOwner) {
			t.Fatal("copied scope authority", e)
		}
		if e := m.partialRemainder.run(&x, &y, 0, false); !errors.Is(e, ErrNumericOwner) {
			t.Fatal("ended scope authority", e)
		}
		if x != before {
			t.Fatal("denied scope changed inline output")
		}
		serviceMust(t, m.Del(&x))
		serviceMust(t, m.Del(&y))
	}
}
