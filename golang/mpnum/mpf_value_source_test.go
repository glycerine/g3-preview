// Source: src/util/mpf.cpp:23-59,316-333; src/util/mpf.h:88-90,107-114,196-199.
// Source author: Christoph Wintersteiger (2011-12-01).
// Licensed under the MIT license; see LICENSE.txt.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/utilhash"
	"testing"
)

func TestArbitraryFloatValueGeometryStructuralEqualityAndHash(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	m, e := NewArbitraryFloatManager(serviceConfig(b).Integers)
	serviceMust(t, e)
	defer func() {
		serviceMust(t, m.Close())
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	}()
	var a, c ArbitraryFloat
	if a.EBits() != 0 || a.SBits() != 0 || a.Sign() || a.Exponent() != 0 {
		t.Fatal(a)
	}
	if same, e := m.EqCore(&a, &c); e != nil || !same {
		t.Fatal(same, e)
	}
	for _, exponent := range []int64{0, 1, -1, 1 << 40, -(1 << 40), -1 << 63, 1<<63 - 1} {
		one := NewInt(1)
		serviceMust(t, m.SetComponents(&a, 8, 24, false, exponent, &one))
		serviceMust(t, m.SetComponents(&c, 8, 24, true, exponent, &one))
		ha, e := m.Hash(&a)
		serviceMust(t, e)
		hc, e := m.Hash(&c)
		serviceMust(t, e)
		want := utilhash.UintPair(1, utilhash.Uint64(uint64(exponent)))
		if ha != want || hc != want {
			t.Fatal(exponent, ha, hc, want)
		}
		if same, e := m.EqCore(&a, &c); e != nil || same {
			t.Fatal("structural sign equality", same, e)
		}
		serviceMust(t, m.Copy(&c, &a))
		if same, e := m.EqCore(&a, &c); e != nil || !same {
			t.Fatal(same, e)
		}
		words, e := m.ValueWords(&a)
		serviceMust(t, e)
		decoded := CaptureArbitraryFloatWords(words)
		if same, e := m.EqCore(&a, &decoded); e != nil || !same {
			t.Fatal("physical codec", same, e, words)
		}
	}
	one := NewInt(1)
	serviceMust(t, m.SetComponents(&a, 0xffffffff, 0xffffffff, true, 42, &one))
	if a.EBits() != 32767 || a.SBits() != 65535 || !a.Sign() || a.Exponent() != 42 {
		t.Fatal("source bitfields", a)
	}
	serviceMust(t, m.Del(&a))
	serviceMust(t, m.Del(&c))
}

func TestArbitraryFloatEqualityShortCircuitsBeforeSignificandRead(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	m, e := NewArbitraryFloatManager(serviceConfig(b).Integers)
	serviceMust(t, e)
	defer func() {
		serviceMust(t, m.Close())
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
	}()
	var a, c ArbitraryFloat
	one := NewInt(1)
	serviceMust(t, m.SetComponents(&a, 8, 24, false, 0, &one))
	serviceMust(t, m.SetComponents(&c, 11, 53, false, 0, &one))
	w, e := m.ValueWords(&c)
	serviceMust(t, e)
	// This deliberately invalid captured raw cell tests reached-read timing;
	// it is a Go pointer guard domain, not native numerical input coverage.
	w[4], w[5] = 0xffffffff, 0xffffffff
	bad := CaptureArbitraryFloatWords(w)
	if same, e := m.EqCore(&a, &bad); e != nil || same {
		t.Fatal("format mismatch read unreached significand", same, e)
	}
	if _, e := m.Hash(&bad); e == nil {
		t.Fatal("hash did not validate reached significand")
	}
	serviceMust(t, m.Del(&a))
	serviceMust(t, m.Del(&c))
}

func TestArbitraryFloatSuccessfulBigCopyAllowsContextEnd(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	m, e := NewArbitraryFloatManager(serviceConfig(b).Integers)
	serviceMust(t, e)
	var source Int
	serviceMust(t, m.integer.SetLimbs(&source, []uint32{0xffffffff, 0xffffffff, 0x7fffffff}))
	var a, c ArbitraryFloat
	serviceMust(t, m.SetComponents(&a, 11, 113, false, -42, &source))
	serviceMust(t, m.Copy(&c, &a))
	if same, e := m.EqCore(&a, &c); e != nil || !same {
		t.Fatal(same, e)
	}
	integerPhysicalAudit(t, m.integer, &source, &a.significand, &c.significand)
	serviceMust(t, m.Del(&a))
	serviceMust(t, m.Del(&c))
	serviceMust(t, m.integer.Del(&source))
	if e := m.Close(); e != nil {
		t.Fatal("completed actual copy receipt blocked enclosing context end", e, m.valueCopy)
	}
	if b.Used() != 0 {
		t.Fatal(b.Used())
	}
}

func TestArbitraryFloatEveryCopyReservationAndLiteralFieldPrefix(t *testing.T) {
	for _, copyValue := range []bool{false, true} {
		success := false
		for ordinal := int64(0); ordinal < 24; ordinal++ {
			b := arena.NewBudget(1 << 26)
			m, e := NewArbitraryFloatManager(serviceConfig(b).Integers)
			serviceMust(t, e)
			var source Int
			digits := make([]uint32, 1024)
			for i := range digits {
				digits[i] = 0xffffffff
			}
			digits[len(digits)-1] = 0x7fffffff
			serviceMust(t, m.integer.SetLimbs(&source, digits))
			var input, out ArbitraryFloat
			serviceMust(t, m.SetComponents(&input, 11, 40000, true, -42, &source))
			seven := NewInt(7)
			serviceMust(t, m.SetComponents(&out, 8, 24, false, 99, &seven))
			b.FailAfter(ordinal)
			if copyValue {
				e = m.Copy(&out, &input)
			} else {
				e = m.SetComponents(&out, 11, 40000, true, -42, &source)
			}
			// Name the actual caller numerals and retained leaf before lifting.
			integerPhysicalAudit(t, m.integer, &source, &input.significand, &out.significand)
			var maps uint64
			for i := range m.powers {
				maps += m.powers[i].StorageBytes()
			}
			if want := m.header + m.ValueCopyHistoryBytes() + m.rational.headerBytes + integerActualOwnedBytes(m.integer) + maps; b.Used() != want {
				t.Fatal(copyValue, ordinal, "actual physical/cold graph", b.Used(), want)
			}
			if out.EBits() != 11 || out.SBits() != 40000 || !out.Sign() {
				t.Fatal("scalar source prefix", copyValue, ordinal, out)
			}
			if e != nil {
				if !numericPhysicalReservationRefusal(e) {
					t.Fatal(e)
				}
				want := int64(99)
				if copyValue {
					want = -42
				}
				if out.Exponent() != want {
					t.Fatal("SET/Copy exponent order", copyValue, ordinal, out.Exponent(), want)
				}
			} else if same, eq := m.EqCore(&out, &input); eq != nil || !same {
				t.Fatal(same, eq)
			}
			b.FailAfter(-1)
			serviceMust(t, m.CleanupValue(&out))
			serviceMust(t, m.Del(&input))
			serviceMust(t, m.integer.Del(&source))
			serviceMust(t, m.Close())
			if b.Used() != 0 {
				t.Fatal(copyValue, ordinal, b.Used())
			}
			if e == nil {
				if ordinal == 0 {
					t.Fatal("copy fixture reached no actual reservation")
				}
				t.Logf("copy=%v successful reservation horizon=%d", copyValue, ordinal)
				success = true
				break
			}
		}
		if !success {
			t.Fatal("copy campaign did not reach successful frontier", copyValue)
		}
	}
}
