// Source: src/util/mpf.cpp:1228-1253,1850-1855.
package numeral

import (
	"errors"
	"math"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestArbitraryFloatToIEEEBVWarmAuthorityAndLiteralAlias(t *testing.T) {
	m, r, b := mpfNumeralHelpersFixture(t)
	var x ArbitraryFloat
	serviceMust(t, m.SetDouble(&x, 8, 3, 1.5))
	out := NewInt(7)
	serviceMust(t, m.ToIEEEBVMPZ(&x, &out))
	used, state := b.Used(), r.State()
	b.FailAfter(0)
	serviceMust(t, m.ToIEEEBVMPZ(&x, &out))
	if out != NewInt(510) || b.Used() != used || r.State() != state {
		t.Fatal("warm inline field assembly allocated", out)
	}
	copied := m.ieeeBV
	copied.hooks.Policy = &copied
	if e := copied.run(&x, &out); !errors.Is(e, ErrNumericOwner) {
		t.Fatal("copied body authority", e)
	}
	if e := m.ieeeBV.run(&x, &out); !errors.Is(e, ErrNumericOwner) || out != NewInt(510) {
		t.Fatal("ended body authority", e, out)
	}
	// Source reads sig(x) only after output SET/shift; an actual shared field
	// therefore adds the already-assembled prefix to itself.
	serviceMust(t, m.ToIEEEBVMPZ(&x, &x.significand))
	if x.significand != NewInt(1016) {
		t.Fatal("literal alias final ADD", x)
	}
	b.FailAfter(-1)
	serviceMust(t, m.integer.Del(&out))
	serviceMust(t, m.Del(&x))
}

func TestArbitraryFloatBiasExpIndependentBounds(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	for _, eb := range []uint32{2, 8, 11, 32, 63, 64} {
		bias := int64((uint64(1) << (eb - 1)) - 1)
		for _, exp := range []int64{-100, -1, 0} {
			got, e := m.BiasExp(eb, exp)
			serviceMust(t, e)
			if got != exp+bias {
				t.Fatal("bias", eb, exp, got)
			}
			back, e := m.UnbiasExp(eb, got)
			serviceMust(t, e)
			if back != exp {
				t.Fatal("unbias", eb, exp, back)
			}
		}
	}
	if _, e := m.BiasExp(64, 1); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("signed bias overflow", e)
	}
	if _, e := m.UnbiasExp(64, -2); !errors.Is(e, arena.ErrBounds) {
		t.Fatal("signed unbias overflow", e)
	}
	got, e := m.UnbiasExp(64, -1)
	serviceMust(t, e)
	if got != math.MinInt64 {
		t.Fatal("signed boundary", got)
	}
}

func TestArbitraryFloatToIEEEBVSourceAssertionsAndBiasOverflow(t *testing.T) {
	var logFile *os.File
	if modularSourceDebug {
		_, logFile = bindMPNDebugActions(t, "TT")
	}
	m, _, _ := mpfNumeralHelpersFixture(t)
	out := NewInt(7)
	if modularSourceDebug {
		var x ArbitraryFloat
		serviceMust(t, m.MkNaN(8, 24, &x, nil))
		before := m.ieeeBV.invocation
		if e := m.ToIEEEBVMPZ(&x, &out); !isMPNVerificationThrow(e) || out != NewInt(7) || m.ieeeBV.invocation != before || m.ieeeBV.active || m.ieeeBV.biasedExpLive {
			t.Fatal("NaN assertion must precede scope", e, out)
		}
		serviceMust(t, m.Del(&x))
		x = ArbitraryFloat{format: arbitraryFloatFormat(32, 24, false), exponent: math.MaxInt32, significand: NewInt(0)}
		if e := m.ToIEEEBVMPZ(&x, &out); !isMPNVerificationThrow(e) || out != NewInt(7) || m.ieeeBV.invocation != before || m.ieeeBV.active || m.ieeeBV.biasedExpLive {
			t.Fatal("exponent assertion before output SET", e, out)
		}
		contents, e := os.ReadFile(logFile.Name())
		serviceMust(t, e)
		for _, site := range []string{"Line: 1229\n!is_nan(x)\n", "Line: 1230\nexp(x) < INT_MAX\n"} {
			if n := strings.Count(string(contents), site); n != 1 {
				t.Fatal("source IEEE bit-vector assertion site", site, n, string(contents))
			}
		}
	} else {
		x := ArbitraryFloat{format: arbitraryFloatFormat(63, 24, false), exponent: math.MaxInt64, significand: NewInt(0)}
		if e := m.ToIEEEBVMPZ(&x, &out); !errors.Is(e, arena.ErrBounds) || out != NewInt(7) || !m.ieeeBV.active || m.ieeeBV.biasedExpLive {
			t.Fatal("bias overflow before output SET", e, out)
		}
		serviceMust(t, m.CleanupToIEEEBVMPZ())
	}
}
