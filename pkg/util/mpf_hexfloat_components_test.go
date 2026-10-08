// Source: src/util/mpf.cpp:1608-1619; src/util/mpf.h:294.
package util

import (
	"math/big"
	"math/rand/v2"
	"strconv"
	"testing"
)

func TestArbitraryFloatHexfloatComponentsLiteralSequence(t *testing.T) {
	m, _, b := mpfNumeralHelpersFixture(t)
	for _, tc := range []struct {
		sig   int32
		exp   int64
		rbits uint32
		sign  bool
		want  string
	}{
		{12, 3, 0, false, "1.5 3"}, {49, 3, 2, false, "1.5 3"}, {12, 3, 0, true, "-1.5 3"},
		{4, -126, 0, false, "0.5 -126"}, {0, -126, 0, true, "-zero"},
		{3, -126, 1, false, "0.125 -126"}, {4, 0, 0, false, "1.5 0"},
		{16, 0, 0, false, "2.0 0"}, {8, 128, 0, false, "+oo"}, {12, 128, 0, true, "NaN"},
		{-1, 0, 0, false, "0.875 0"},
		{-5, 0, 1, true, "-0.625 0"},
	} {
		sig := NewInt(tc.sig)
		before := sig
		text, e := NewArbitraryFloatFormatText(b)
		serviceMust(t, e)
		view, e := m.ToStringHexfloatComponentsInto(tc.sign, tc.exp, &sig, 8, 4, tc.rbits, text)
		serviceMust(t, e)
		if actual := formatTestString(t, view); actual != tc.want || sig != before {
			t.Fatal("literal component hexfloat", tc, actual)
		}
		serviceMust(t, text.Close())
	}
}

func TestArbitraryFloatHexfloatComponentsIndependentMantissas(t *testing.T) {
	m, _, b := mpfNumeralHelpersFixture(t)
	rng := rand.New(rand.NewChaCha8([32]byte{19, 24, 37}))
	for _, sb := range []uint32{2, 4, 24, 53, 120} {
		for sample := 0; sample < 20; sample++ {
			scale := uint32(sample % 5)
			words := make([]byte, (sb+scale+7)/8)
			for i := range words {
				words[i] = byte(rng.Uint32())
			}
			input := new(big.Int).SetBytes(words)
			input.Mod(input, new(big.Int).Lsh(big.NewInt(1), uint(sb+scale+1)))
			if sample == 0 {
				input.SetInt64(0)
			}
			if sample == 1 {
				input.SetInt64(1)
			}
			if sample == 2 {
				input.Lsh(big.NewInt(1), uint(sb+scale-1))
			}
			exp := int64(sample - 10)
			if sample < 4 {
				exp = -126
			}
			negative := sample%2 != 0
			sig := NewInt(0)
			serviceMust(t, m.integer.SetString(&sig, input.String()))
			before := sig
			text, e := NewArbitraryFloatFormatText(b)
			serviceMust(t, e)
			view, e := m.ToStringHexfloatComponentsInto(negative, exp, &sig, 8, sb, scale, text)
			serviceMust(t, e)
			// Build the represented mantissa mathematically, without using the port.
			restored := new(big.Int).Rsh(new(big.Int).Set(input), uint(scale))
			hidden := new(big.Int).Lsh(big.NewInt(1), uint(sb-1))
			denormal := restored.Cmp(hidden) < 0 && exp == -126
			num := new(big.Int).Set(restored)
			if !denormal && num.Cmp(hidden) < 0 {
				num.Add(num, hidden)
			}
			expected := ""
			if denormal && num.Sign() == 0 {
				expected = "+zero"
				if negative {
					expected = "-zero"
				}
			} else {
				rat := new(big.Rat).SetFrac(num, hidden)
				expected = decimalOracle(rat, sb, false)
				if rat.IsInt() {
					expected += ".0"
				}
				expected += " " + strconv.FormatInt(exp, 10)
				if negative {
					expected = "-" + expected
				}
			}
			if actual := formatTestString(t, view); actual != expected || sig != before {
				t.Fatal("independent component mantissa", sb, sample, actual, expected)
			}
			serviceMust(t, text.Close())
			serviceMust(t, m.integer.Del(&sig))
		}
	}
}

type hexfloatNestedText struct {
	*ArbitraryFloatFormatText
	hook func()
}

func (t *hexfloatNestedText) CString() (ArbitraryFloatFormattedCString, error) {
	if t.hook != nil {
		hook := t.hook
		t.hook = nil
		hook()
	}
	return t.ArbitraryFloatFormatText.CString()
}

func TestArbitraryFloatHexfloatComponentsExactChildWithLaterCallback(t *testing.T) {
	m, _, b := mpfNumeralHelpersFixture(t)
	outer, e := NewArbitraryFloatFormatText(b)
	serviceMust(t, e)
	nested, e := NewArbitraryFloatFormatText(b)
	serviceMust(t, e)
	sig := NewInt(12)
	var nestedView ArbitraryFloatFormattedCString
	proxy := &hexfloatNestedText{ArbitraryFloatFormatText: outer, hook: func() {
		s := &m.hexfloatComponents
		if s.format != &outer.Numeric || s.formatInvocation != outer.Numeric.invocation || s.formatOwned {
			t.Fatal("exact child receipt not issued at begin")
		}
		infinity := ArbitraryFloat{format: arbitraryFloatFormat(8, 4, false), exponent: 128}
		var e error
		nestedView, e = m.ToStringInto(&infinity, nested)
		serviceMust(t, e)
	}}
	view, e := m.ToStringHexfloatComponentsInto(false, 3, &sig, 8, 4, 0, proxy)
	serviceMust(t, e)
	s := &m.hexfloatComponents
	if formatTestString(t, view) != "1.5 3" || formatTestString(t, nestedView) != "+oo" || !outer.Numeric.owned || nested.Numeric.owned || s.format != &outer.Numeric || s.formatInvocation >= nested.Numeric.invocation || m.formatInvocation != nested.Numeric.invocation {
		t.Fatal("component captured the later callback frame")
	}
	before, used := nested.Numeric, b.Used()
	serviceMust(t, m.CleanupHexfloatComponents())
	if nested.Numeric != before || b.Used() != used || formatTestString(t, nestedView) != "+oo" {
		t.Fatal("ended parent adopted later format")
	}
	serviceMust(t, outer.Close())
	serviceMust(t, nested.Close())
}
