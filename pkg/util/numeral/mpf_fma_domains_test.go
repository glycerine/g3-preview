// Source: src/util/mpf.cpp:748-949.
package numeral

import (
	"errors"
	"math"
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func TestArbitraryFloatFMAEntryFormatSourceActions(t *testing.T) {
	if !modularSourceDebug {
		t.Skip("source Z3DEBUG assertion only")
	}
	d, log := bindMPNDebugActions(t, "TC")
	m, _, _ := mpfNumeralHelpersFixture(t)
	var x, y, z, out ArbitraryFloat
	serviceMust(t, m.MkZero(11, 53, false, &x))
	serviceMust(t, m.MkOne(8, 24, false, &y))
	serviceMust(t, m.MkInf(11, 53, true, &z))
	serviceMust(t, m.MkOne(11, 53, false, &out))
	before := out
	if e := m.FMA(ArbitraryFloatNearestEven, &x, &y, &z, &out); !isMPNVerificationThrow(e) || out != before {
		t.Fatal("entry format Throw before output publication", e, out)
	}
	contents, e := os.ReadFile(log.Name())
	serviceMust(t, e)
	want := "File: src/util/mpf.cpp\nLine: 749\nx.sbits == y.sbits && x.ebits == y.ebits && x.sbits == y.sbits && z.ebits == z.ebits\n"
	if n := strings.Count(string(contents), want); n != 1 {
		t.Fatal("source FMA format assertion", n, string(contents))
	}
	serviceMust(t, m.FMA(ArbitraryFloatNearestEven, &x, &y, &z, &out))
	if out != z {
		t.Fatal("Continue follows source infinity shortcut", out, z)
	}
	serviceMust(t, d.EnableAssertions(false))
	serviceMust(t, m.FMA(ArbitraryFloatNearestEven, &x, &y, &z, &out))
	if out != z {
		t.Fatal("disabled assertions follow source infinity shortcut", out, z)
	}
	serviceMust(t, d.Close())
	if e := m.FMA(ArbitraryFloatNearestEven, &x, &x, &z, &out); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("valid FMA reads closed assertion owner", e)
	}
	for _, v := range []*ArbitraryFloat{&out, &x, &y, &z} {
		serviceMust(t, m.Del(v))
	}
}

func TestArbitraryFloatFMASpecialsSignedZerosAndModes(t *testing.T) {
	m, _, _ := mpfNumeralHelpersFixture(t)
	values := []float64{0, math.Copysign(0, -1), 1, -1, math.Inf(1), math.Inf(-1), math.NaN()}
	for _, a := range values {
		for _, b := range values {
			for _, c := range values {
				for _, mode := range mpfIntegralModes {
					for alias := 0; alias < 4; alias++ {
						var x, y, z, separate ArbitraryFloat
						for i, v := range []*ArbitraryFloat{&x, &y, &z} {
							serviceMust(t, m.SetDouble(v, 11, 53, []float64{a, b, c}[i]))
						}
						out := []*ArbitraryFloat{&separate, &x, &y, &z}[alias]
						serviceMust(t, m.FMA(mode, &x, &y, &z, out))
						got, e := m.ToDouble(out)
						serviceMust(t, e)
						want := math.FMA(a, b, c)
						if want == 0 {
							negative := mode == ArbitraryFloatTowardNegative
							if (a == 0 || b == 0) && c == 0 {
								productSign := math.Signbit(a) != math.Signbit(b)
								zSign := math.Signbit(c)
								negative = productSign && zSign || mode == ArbitraryFloatTowardNegative && productSign != zSign
							}
							want = 0
							if negative {
								want = math.Copysign(0, -1)
							}
						}
						if math.IsNaN(want) {
							if !math.IsNaN(got) || out.Sign() {
								t.Fatal("canonical fused NaN", a, b, c, mode, alias, out)
							}
						} else if math.Float64bits(got) != math.Float64bits(want) {
							t.Fatal("source special/zero result", a, b, c, mode, alias, got, want)
						}
						for _, v := range []*ArbitraryFloat{&separate, &x, &y, &z} {
							serviceMust(t, m.Del(v))
						}
					}
				}
			}
		}
	}
}

func TestArbitraryFloatFMALiteralFormatAssertionAndSmallPrecision(t *testing.T) {
	if modularSourceDebug {
		bindMPNDebugActions(t, "TT")
	}
	m, _, _ := mpfNumeralHelpersFixture(t)
	var x, y, z, out ArbitraryFloat
	serviceMust(t, m.MkOne(11, 53, false, &x))
	serviceMust(t, m.MkOne(11, 53, false, &y))
	serviceMust(t, m.MkInf(8, 24, true, &z))
	serviceMust(t, m.FMA(ArbitraryFloatNearestEven, &x, &y, &z, &out))
	if out.format != z.format || out.exponent != z.exponent || out.significand != z.significand {
		t.Fatal("source initial assertion does not check z format", out, z)
	}
	for _, v := range []*ArbitraryFloat{&out, &x, &y, &z} {
		serviceMust(t, m.Del(v))
	}
	for _, sb := range []uint32{2, 3} {
		for _, v := range []*ArbitraryFloat{&x, &y, &z} {
			serviceMust(t, m.MkOne(11, sb, false, v))
		}
		body := m.FMA(ArbitraryFloatNearestEven, &x, &y, &z, &out)
		if modularSourceDebug {
			if !isMPNVerificationThrow(body) || m.fma.roundFrame != &m.RoundScope || !isMPNVerificationThrow(m.RoundScope.bodyCause) {
				t.Fatal("literal small-precision shift reaches Round assertion", sb, body)
			}
		} else {
			serviceMust(t, body)
			actual, e := m.integer.String(&out.significand)
			serviceMust(t, e)
			// The literal extra six-bit shift scales this exact sum by 64.
			// Unsigned lz wraps, then Round publishes emin+1 and the expanded
			// significand with only one post-normalization shift.
			want := new(big.Int).Lsh(big.NewInt(1), uint(1027+sb))
			want.Sub(want, new(big.Int).Lsh(big.NewInt(1), uint(sb-1)))
			if actual != want.String() || out.exponent != -1021 {
				t.Fatal("literal small-precision release fields", sb, out, actual, want)
			}
			q := NewRational()
			serviceMust(t, m.ToRational(&out, m.rational, &q))
			n, e := m.integer.Int64(&q.Num)
			serviceMust(t, e)
			d, e := m.integer.Int64(&q.Den)
			serviceMust(t, e)
			if n != 128 || d != 1 {
				t.Fatal("literal source scale", sb, n, d)
			}
			serviceMust(t, m.rational.Del(&q))
		}
		serviceMust(t, m.CleanupFMA())
		for _, v := range []*ArbitraryFloat{&out, &x, &y, &z} {
			serviceMust(t, m.CleanupValue(v))
		}
	}
}
