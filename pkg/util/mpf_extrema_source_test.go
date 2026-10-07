// Source: src/util/mpf.cpp:1487-1517; src/util/mpf.h:119,122,137-138.
// Licensed under the MIT license. Source author: Christoph Wintersteiger.
package util

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
)

func extremaTestNumeral(bits uint32) ArbitraryFloat {
	e := int64((bits>>23)&255) - 127
	if (bits>>23)&255 == 0 {
		e = -127
	}
	return ArbitraryFloat{format: arbitraryFloatFormat(8, 24, bits>>31 != 0), exponent: e, significand: NewInt(int32(bits & 0x7fffff))}
}
func TestArbitraryFloatExtremaSourceCasesAndAliases(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	m, e := NewArbitraryFloatManager(serviceConfig(b).Integers)
	serviceMust(t, e)
	cases := []struct {
		x, y, min, max uint32
		unsupported    bool
	}{
		{0x3f800000, 0x40000000, 0x3f800000, 0x40000000, false},
		{0xbf800000, 0xc0000000, 0xc0000000, 0xbf800000, false},
		{0x7f800000, 0xff800000, 0xff800000, 0x7f800000, false},
		{0x7fc00001, 0x3f800000, 0x3f800000, 0x3f800000, false},
		{0x3f800000, 0xffc00002, 0x3f800000, 0x3f800000, false},
		{0x7fc00001, 0xffc00002, 0xffc00002, 0xffc00002, false},
		{0, 0, 0, 0, false}, {0x80000000, 0x80000000, 0x80000000, 0x80000000, false},
		{0, 0x80000000, 0, 0, true}, {0x80000000, 0, 0, 0, true},
	}
	for _, tc := range cases {
		for _, maximum := range []bool{false, true} {
			for alias := 0; alias < 3; alias++ {
				x, y := extremaTestNumeral(tc.x), extremaTestNumeral(tc.y)
				out := extremaTestNumeral(0x40800000)
				dst := &out
				if alias == 1 {
					dst = &x
				}
				if alias == 2 {
					dst = &y
				}
				before := *dst
				if maximum {
					e = m.Maximum(&x, &y, dst)
				} else {
					e = m.Minimum(&x, &y, dst)
				}
				if tc.unsupported {
					if e != ErrIntegerAssertion || *dst != before {
						t.Fatal("source mixed-zero unreachable changed destination", tc, maximum, alias, e, *dst, before)
					}
					continue
				}
				serviceMust(t, e)
				bits := tc.min
				if maximum {
					bits = tc.max
				}
				want := extremaTestNumeral(bits)
				if *dst != want {
					t.Fatal("exact source selection", tc, maximum, alias, *dst, want)
				}
			}
		}
	}
	serviceMust(t, m.Close())
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestArbitraryFloatExtremaSelectedCopyPrefixes(t *testing.T) {
	for _, maximum := range []bool{false, true} {
		success := false
		for prefix := int64(0); prefix < 40; prefix++ {
			b := arena.NewBudget(1 << 26)
			m, e := NewArbitraryFloatManager(serviceConfig(b).Integers)
			serviceMust(t, e)
			x := ArbitraryFloat{format: arbitraryFloatFormat(8, 100, false), exponent: 42}
			serviceMust(t, m.integer.SetUint64(&x.significand, 1<<40))
			y := ArbitraryFloat{format: x.format, exponent: 43, significand: NewInt(1)}
			if maximum {
				y.exponent = 41
			}
			_, e = m.IsNaN(&x)
			serviceMust(t, e)
			_, e = m.IsZero(&x)
			serviceMust(t, e)
			var out ArbitraryFloat
			var receipt ArbitraryFloatValueCopyReceipt
			b.FailAfter(prefix)
			if maximum {
				e = m.MaximumWithReceipt(&x, &y, &out, &receipt)
			} else {
				e = m.MinimumWithReceipt(&x, &y, &out, &receipt)
			}
			integerPhysicalAuditWithRaw(t, m.integer, nil, &x.significand, &y.significand, &out.significand)
			var maps uint64
			for i := range m.powers {
				maps += m.powers[i].StorageBytes()
			}
			want := m.header + m.ValueCopyHistoryBytes() + m.powerWrapperBytes + m.rational.HeaderBytes + integerActualOwnedBytes(m.integer) + maps
			if b.Used() != want {
				t.Fatal("selected Copy before-lift cold graph", maximum, prefix, b.Used(), want)
			}
			if e == nil {
				v, read := m.integer.Uint64(&out.significand)
				serviceMust(t, read)
				if v != 1<<40 || out.Exponent() != 42 || out.Sign() {
					t.Fatal("wrong selected BIG", out, v)
				}
			}
			b.FailAfter(-1)
			serviceMust(t, m.CleanupValueCopy(&receipt, false))
			serviceMust(t, m.CleanupValue(&out))
			serviceMust(t, m.CleanupValueCopy(&receipt, true))
			serviceMust(t, m.Del(&x))
			serviceMust(t, m.Del(&y))
			serviceMust(t, m.Close())
			if b.Used() != 0 {
				t.Fatal("selected Copy leak", maximum, prefix, b.Used())
			}
			if e == nil {
				success = true
				break
			}
		}
		if !success {
			t.Fatal("no successful selected Copy prefix", maximum)
		}
	}
}
